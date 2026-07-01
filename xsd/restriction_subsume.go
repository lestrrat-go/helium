package xsd

import (
	"context"
	"slices"
)

// This file implements the XSD 1.1 content-model restriction relaxation as a
// SOUND, Version11-only FALLBACK to the syntactic Particle Valid (Restriction)
// rules in restriction_particle.go.
//
// The XSD 1.0 syntactic clauses (Recurse/MapAndSum/RecurseAsIfGroup/...) are
// known-incomplete: they reject many restrictions whose derived content-model
// LANGUAGE is genuinely a subset of the base's (e.g. an element restricting a
// choice whose emptiability comes from the enclosing choice's own occurrence,
// not the matched branch). XSD 1.1 accepts such restrictions (W3C MS
// particles*, "Invalid restriction which becomes valid in XSD 1.1"). The
// authoritative criterion is semantic: a derived content model is a valid
// restriction of the base iff every element sequence valid against the derived
// model is also valid against the base model — i.e. L(derived) ⊆ L(base) — with
// each derived element declaration validly restricting (name/type) the base
// declaration that governs it.
//
// particleLanguageSubset PROVES L(derived) ⊆ L(base) by product simulation of
// the two content-model automata over the finite alphabet of names the derived
// model can emit. It is FAIL-CLOSED: it returns true only when it can rigorously
// prove inclusion, and false (keep the original rejection) for anything it
// cannot represent — a DERIVED wildcard (proving a wildcard ⊆ base is not
// attempted), an xs:all group, or an over-large expansion. Because it is used
// only as a fallback AFTER the syntactic check fails and only ever ACCEPTS, it
// can never reject more than before; and because inclusion+type-compat is the
// semantic definition of restriction, an accepted schema is genuinely valid.
// Gated to Version11, so XSD 1.0 is byte-identical.

// These caps bound the number of automaton states, reachable product states,
// and occurrence-unroll copies so a pathological content model cannot blow up
// compilation; exceeding any makes the checker bail (return "not proven", never
// a spurious accept).
const (
	subsumeNFAStateCap  = 4000
	subsumePairStateCap = 100000
	subsumeUnrollCap    = 128
)

// cmSymbol is one automaton transition label: either a concrete element
// (elem!=nil, matching exactly its expanded name) or a base wildcard (wc!=nil,
// matching every name the wildcard admits). A derived-side symbol is always an
// element (a derived wildcard makes the whole check bail).
type cmSymbol struct {
	elem *ElementDecl
	name QName // the single concrete name an element symbol emits
	wc   *Wildcard
}

// reNode is a normalized content-model regular expression over cmSymbol.
type reNode interface{ reNode() }

type reEmpty struct{}             // matches the empty string
type reSym struct{ sym cmSymbol } // matches one symbol
type reConcat struct{ items []reNode }
type reUnion struct{ items []reNode }
type reStar struct{ item reNode }
type reOpt struct{ item reNode }

func (reEmpty) reNode()  {}
func (reSym) reNode()    {}
func (reConcat) reNode() {}
func (reUnion) reNode()  {}
func (reStar) reNode()   {}
func (reOpt) reNode()    {}

// cmBuilder converts particles to normalized regexes. ok goes false the moment
// an unrepresentable construct is seen (derived wildcard, xs:all, over-large
// bounded expansion) so the caller can bail soundly.
type cmBuilder struct {
	schema    *Schema
	isDerived bool
	ok        bool
}

// particleLanguageSubset reports whether L(derived) ⊆ L(base) can be PROVEN,
// with every derived element declaration validly restricting the same-named base
// declaration it may be matched by. A true result means the restriction is
// semantically valid; a false result means "not proven" (the caller keeps its
// existing verdict).
func particleLanguageSubset(ctx context.Context, derived, base *Particle, schema *Schema, version Version) bool {
	db := &cmBuilder{schema: schema, isDerived: true, ok: true}
	dre := db.particleToRe(derived)
	if !db.ok {
		return false
	}
	bb := &cmBuilder{schema: schema, isDerived: false, ok: true}
	bre := bb.particleToRe(base)
	if !bb.ok {
		return false
	}

	dnfa := buildNFA(dre)
	bnfa := buildNFA(bre)
	if dnfa == nil || bnfa == nil {
		return false
	}
	if len(dnfa.states) > subsumeNFAStateCap || len(bnfa.states) > subsumeNFAStateCap {
		return false
	}

	// Type/nillable/fixed pre-check: every derived element declaration must
	// validly restrict every base element declaration that shares its expanded
	// name (a base wildcard imposes no type constraint — NSCompat). If any pair is
	// type-incompatible the restriction is genuinely invalid, so bail.
	derivedElems := collectElemSymbols(dnfa)
	baseElems := collectElemSymbols(bnfa)
	for _, ds := range derivedElems {
		for _, bs := range baseElems {
			if ds.name != bs.name {
				continue
			}
			if !elemSymbolRestricts(ctx, ds.elem, bs.elem, schema, version) {
				return false
			}
		}
	}

	// Alphabet: the names the derived model can emit. L(derived) ⊆ Σ*, so
	// inclusion over Σ is exact.
	alpha := map[QName]struct{}{}
	for _, ds := range derivedElems {
		alpha[ds.name] = struct{}{}
	}

	return nfaLanguageSubset(dnfa, bnfa, alpha, schema)
}

// particleToRe normalizes a particle. A model-group term carries its occurrence
// range on the GROUP (the wrapping particle's copy is redundant, mirroring the
// UPA automaton's walkParticle), so only element/wildcard leaves fold in the
// particle's own occurrence here.
func (b *cmBuilder) particleToRe(p *Particle) reNode {
	if !b.ok {
		return reEmpty{}
	}
	if p.MaxOccurs == 0 {
		return reEmpty{}
	}
	if mg, ok := p.Term.(*ModelGroup); ok {
		return b.modelGroupToRe(mg)
	}
	body := b.leafToRe(p.Term)
	return b.applyOcc(body, p.MinOccurs, p.MaxOccurs)
}

func (b *cmBuilder) modelGroupToRe(mg *ModelGroup) reNode {
	if !b.ok {
		return reEmpty{}
	}
	if mg.MaxOccurs == 0 {
		return reEmpty{}
	}
	// xs:all is an interleaving (shuffle) language; proving inclusion for it is
	// not attempted here. Bail soundly.
	if mg.Compositor == CompositorAll {
		b.ok = false
		return reEmpty{}
	}
	items := make([]reNode, 0, len(mg.Particles))
	for _, cp := range mg.Particles {
		items = append(items, b.particleToRe(cp))
		if !b.ok {
			return reEmpty{}
		}
	}
	var body reNode
	if mg.Compositor == CompositorChoice {
		body = reUnion{items: items}
	} else {
		body = reConcat{items: items}
	}
	return b.applyOcc(body, mg.MinOccurs, mg.MaxOccurs)
}

// leafToRe converts an element or wildcard term to a regex. An element leaf is a
// choice over its expanded concrete emittable names (itself when concrete, plus
// its instance-admissible substitution-group members). A DERIVED wildcard bails.
func (b *cmBuilder) leafToRe(term ParticleTerm) reNode {
	switch t := term.(type) {
	case *ElementDecl:
		syms := b.elemSymbols(t)
		if len(syms) == 0 {
			// An abstract element with no concrete members can emit nothing.
			return reEmpty{}
		}
		if len(syms) == 1 {
			return reSym{sym: syms[0]}
		}
		items := make([]reNode, 0, len(syms))
		for _, s := range syms {
			items = append(items, reSym{sym: s})
		}
		return reUnion{items: items}
	case *Wildcard:
		if b.isDerived {
			// Proving a derived wildcard's language ⊆ base is not attempted.
			b.ok = false
			return reEmpty{}
		}
		return reSym{sym: cmSymbol{wc: t}}
	}
	b.ok = false
	return reEmpty{}
}

// elemSymbols expands an element declaration to the concrete element symbols it
// can emit: itself when it is concrete, plus every instance-admissible
// substitution-group member (abstract members excluded — they never appear).
// Each member symbol carries the member's OWN declaration so the type pre-check
// compares the real governing type.
func (b *cmBuilder) elemSymbols(e *ElementDecl) []cmSymbol {
	var out []cmSymbol
	if !e.Abstract {
		out = append(out, cmSymbol{elem: e, name: e.Name})
	}
	if b.schema != nil {
		for _, m := range instanceSubstMembers(e, b.schema) {
			out = append(out, cmSymbol{elem: m, name: m.Name})
		}
	}
	return out
}

// applyOcc folds an occurrence range over a body regex. Unbounded maxOccurs
// becomes body^min followed by body*; a bounded range becomes body^min followed
// by (max-min) optional copies. An over-large required or optional count bails.
func (b *cmBuilder) applyOcc(body reNode, minOccurs, maxOccurs int) reNode {
	if !b.ok {
		return reEmpty{}
	}
	if maxOccurs == 0 {
		return reEmpty{}
	}
	// Only {1,1} is the identity. {0,1} must NOT take this fast-path — it would
	// model an optional leaf as mandatory, DROPPING the empty-string case and
	// UNDER-modeling the derived language (unsound: false-accepts a required→
	// optional min-widening restriction). {0,1} falls through to the reOpt path
	// below (min=0, max=1 → one reOpt(body)), which models the empty case.
	if minOccurs == 1 && maxOccurs == 1 {
		return body
	}
	if minOccurs > subsumeUnrollCap {
		b.ok = false
		return reEmpty{}
	}
	items := make([]reNode, 0, minOccurs+1)
	for range minOccurs {
		items = append(items, body)
	}
	if maxOccurs == Unbounded {
		items = append(items, reStar{item: body})
	} else {
		optional := maxOccurs - minOccurs
		if optional > subsumeUnrollCap {
			b.ok = false
			return reEmpty{}
		}
		for range optional {
			items = append(items, reOpt{item: body})
		}
	}
	if len(items) == 0 {
		return reEmpty{}
	}
	if len(items) == 1 {
		return items[0]
	}
	return reConcat{items: items}
}

// elemSymbolInfo pairs an element declaration with the concrete name it emits.
type elemSymbolInfo struct {
	elem *ElementDecl
	name QName
}

func collectElemSymbols(n *nfa) []elemSymbolInfo {
	var out []elemSymbolInfo
	seen := map[*ElementDecl]map[QName]bool{}
	for _, edges := range n.states {
		for _, e := range edges {
			if e.eps || e.sym.elem == nil {
				continue
			}
			if seen[e.sym.elem] == nil {
				seen[e.sym.elem] = map[QName]bool{}
			}
			if seen[e.sym.elem][e.sym.name] {
				continue
			}
			seen[e.sym.elem][e.sym.name] = true
			out = append(out, elemSymbolInfo{elem: e.sym.elem, name: e.sym.name})
		}
	}
	return out
}

// elemSymbolRestricts reports whether a derived element declaration validly
// restricts a same-named base element declaration (NameAndTypeOK: identical
// name, derived type, no nillable widening, fixed tightening). Occurrence is
// handled by the automaton, so a {1,1}/{1,1} particle pair isolates the
// name/type check. Reuses the syntactic path's element comparator so the
// fallback stays consistent with it.
func elemSymbolRestricts(ctx context.Context, de, be *ElementDecl, schema *Schema, version Version) bool {
	if de == be {
		return true
	}
	dp := &Particle{MinOccurs: 1, MaxOccurs: 1, Term: de}
	bp := &Particle{MinOccurs: 1, MaxOccurs: 1, Term: be}
	return elementRestrictsElement(ctx, dp, de, bp, be, schema, version)
}

// nfa is an epsilon-NFA built by Thompson construction from a normalized regex.
type nfa struct {
	states []([]nfaEdge)
	start  int
	accept int
}

type nfaEdge struct {
	eps bool
	sym cmSymbol
	to  int
}

func (n *nfa) newState() int {
	n.states = append(n.states, nil)
	return len(n.states) - 1
}

func (n *nfa) addEdge(from int, e nfaEdge) {
	n.states[from] = append(n.states[from], e)
}

func buildNFA(re reNode) *nfa {
	n := &nfa{}
	start, accept, ok := buildFrag(n, re)
	if !ok || len(n.states) > subsumeNFAStateCap {
		return nil
	}
	n.start = start
	n.accept = accept
	return n
}

// buildFrag builds a Thompson fragment for re and returns its (start, accept).
func buildFrag(n *nfa, re reNode) (int, int, bool) {
	if len(n.states) > subsumeNFAStateCap {
		return 0, 0, false
	}
	switch r := re.(type) {
	case reEmpty:
		s := n.newState()
		a := n.newState()
		n.addEdge(s, nfaEdge{eps: true, to: a})
		return s, a, true
	case reSym:
		s := n.newState()
		a := n.newState()
		n.addEdge(s, nfaEdge{sym: r.sym, to: a})
		return s, a, true
	case reConcat:
		if len(r.items) == 0 {
			return buildFrag(n, reEmpty{})
		}
		s, a, ok := buildFrag(n, r.items[0])
		if !ok {
			return 0, 0, false
		}
		for _, item := range r.items[1:] {
			s2, a2, ok := buildFrag(n, item)
			if !ok {
				return 0, 0, false
			}
			n.addEdge(a, nfaEdge{eps: true, to: s2})
			a = a2
		}
		return s, a, true
	case reUnion:
		if len(r.items) == 0 {
			return buildFrag(n, reEmpty{})
		}
		s := n.newState()
		a := n.newState()
		for _, item := range r.items {
			s2, a2, ok := buildFrag(n, item)
			if !ok {
				return 0, 0, false
			}
			n.addEdge(s, nfaEdge{eps: true, to: s2})
			n.addEdge(a2, nfaEdge{eps: true, to: a})
		}
		return s, a, true
	case reStar:
		s := n.newState()
		a := n.newState()
		s2, a2, ok := buildFrag(n, r.item)
		if !ok {
			return 0, 0, false
		}
		n.addEdge(s, nfaEdge{eps: true, to: s2})
		n.addEdge(s, nfaEdge{eps: true, to: a})
		n.addEdge(a2, nfaEdge{eps: true, to: s2})
		n.addEdge(a2, nfaEdge{eps: true, to: a})
		return s, a, true
	case reOpt:
		s := n.newState()
		a := n.newState()
		s2, a2, ok := buildFrag(n, r.item)
		if !ok {
			return 0, 0, false
		}
		n.addEdge(s, nfaEdge{eps: true, to: s2})
		n.addEdge(s, nfaEdge{eps: true, to: a})
		n.addEdge(a2, nfaEdge{eps: true, to: a})
		return s, a, true
	}
	return 0, 0, false
}

// eclose returns the epsilon-closure of a set of states.
func (n *nfa) eclose(set map[int]struct{}) map[int]struct{} {
	stack := make([]int, 0, len(set))
	for s := range set {
		stack = append(stack, s)
	}
	out := map[int]struct{}{}
	for s := range set {
		out[s] = struct{}{}
	}
	for len(stack) > 0 {
		s := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, e := range n.states[s] {
			if !e.eps {
				continue
			}
			if _, ok := out[e.to]; ok {
				continue
			}
			out[e.to] = struct{}{}
			stack = append(stack, e.to)
		}
	}
	return out
}

// move returns the epsilon-closure of states reachable from set on name σ. A
// symbol edge matches σ when it is an element with that exact name, or a
// wildcard admitting it.
func (n *nfa) move(set map[int]struct{}, name QName, schema *Schema) map[int]struct{} {
	dst := map[int]struct{}{}
	for s := range set {
		for _, e := range n.states[s] {
			if e.eps || !symbolMatches(e.sym, name, schema) {
				continue
			}
			dst[e.to] = struct{}{}
		}
	}
	if len(dst) == 0 {
		return dst
	}
	return n.eclose(dst)
}

func symbolMatches(sym cmSymbol, name QName, schema *Schema) bool {
	if sym.elem != nil {
		return sym.name == name
	}
	if sym.wc != nil {
		return wildcardAllowsName(sym.wc, name, schema)
	}
	return false
}

func setKey(set map[int]struct{}) string {
	ids := make([]int, 0, len(set))
	for s := range set {
		ids = append(ids, s)
	}
	slices.Sort(ids)
	// Encode the sorted ids as fixed 4-byte little-endian chunks (one per id).
	// This is injective over the sorted id list, so distinct state sets get
	// distinct keys; determinism only requires a stable key.
	b := make([]byte, 0, len(ids)*3)
	for _, id := range ids {
		b = append(b, byte(id), byte(id>>8), byte(id>>16), byte(id>>24))
	}
	return string(b)
}

// nfaLanguageSubset proves L(d) ⊆ L(b) over the finite alphabet alpha (the names
// d can emit) by simultaneously determinizing both automata (product of subset
// constructions). It returns false the instant it finds a reachable
// configuration where d accepts but b does not, or where d can emit a name b
// cannot follow.
func nfaLanguageSubset(d, b *nfa, alpha map[QName]struct{}, schema *Schema) bool {
	dstart := d.eclose(map[int]struct{}{d.start: {}})
	bstart := b.eclose(map[int]struct{}{b.start: {}})

	type pair struct{ ds, bs map[int]struct{} }
	// A structured [2]string key: setKey returns a raw byte string that may
	// contain any separator byte, so concatenating with a delimiter could make
	// distinct (ds,bs) pairs collide, skip a reachable product state, and
	// false-accept an invalid restriction. A two-element array keeps the two
	// set encodings unambiguously separate.
	visited := map[[2]string]struct{}{}
	queue := []pair{{dstart, bstart}}
	visited[[2]string{setKey(dstart), setKey(bstart)}] = struct{}{}

	for len(queue) > 0 {
		if len(visited) > subsumePairStateCap {
			return false
		}
		p := queue[len(queue)-1]
		queue = queue[:len(queue)-1]

		// d accepts here but b does not: a string in L(d) is not in L(b).
		if _, ok := p.ds[d.accept]; ok {
			if _, ok := p.bs[b.accept]; !ok {
				return false
			}
		}
		for name := range alpha {
			ds2 := d.move(p.ds, name, schema)
			if len(ds2) == 0 {
				continue
			}
			bs2 := b.move(p.bs, name, schema)
			if len(bs2) == 0 {
				// d can still emit name here but b cannot follow it.
				return false
			}
			key := [2]string{setKey(ds2), setKey(bs2)}
			if _, ok := visited[key]; ok {
				continue
			}
			visited[key] = struct{}{}
			queue = append(queue, pair{ds2, bs2})
		}
	}
	return true
}
