package xsd

// substClosureByNameThreshold is the closure size at or above which
// substClosure builds a byName index. Below it, substMemberFor scans all
// directly — a linear scan over a handful of pointers is faster than a map
// lookup, and skipping the map avoids an allocation per small closure.
const substClosureByNameThreshold = 8

// substClosure is the precomputed substitution-group closure of ONE global
// head element declaration. Immutable once buildSubstClosures returns: no
// exported *Schema method mutates a compiled schema, so nothing writes to a
// substClosure's fields again after it is built.
type substClosure struct {
	head *ElementDecl // the registered global declaration for this QName

	// all is the full transitive closure in the uncached walk's breadth-first
	// affiliation order, abstract members INCLUDED. Handed out with three-index
	// slicing (never a bare reslice) so a caller's append cannot write into the
	// shared backing array.
	all []*ElementDecl

	// concrete is all with abstract members removed. A SEPARATELY allocated
	// slice, never a reslice of all — reusing all's backing array here was the
	// aliasing bug instanceSubstMembers used to have (see
	// TestInstanceSubstMembersDoesNotAliasSubstitutableMembersFor).
	concrete []*ElementDecl

	// byName maps a member's expanded QName to its declaration, for
	// substMemberFor's O(1) lookup. nil below substClosureByNameThreshold, in
	// which case substMemberFor scans all instead. Unambiguous: the uncached
	// walk's seen set dedups all by QName, so byName never needs to choose
	// between two same-named entries.
	byName map[QName]*ElementDecl
}

// buildSubstClosures precomputes the substitution-group closure of every
// global element declaration in s with a non-empty closure, keyed by the
// head's QName. It MUST run only after every compile-time check has read
// s.substGroups/s.elements — compileSchema calls it last, immediately before
// handing the compiled Schema back to the caller, so s.substClosures is nil
// for the whole compile and every compile-time caller keeps taking the
// uncached path against a possibly half-built schema. A QName whose closure
// comes back empty gets no entry; substitutableMembersFor's lookup contract
// treats a global element with no entry as "closure is empty", not "not yet
// built" (that distinction is made by s.substClosures itself being nil).
func (s *Schema) buildSubstClosures() {
	closures := make(map[QName]*substClosure, len(s.elements))
	for name, head := range s.elements {
		all := substitutableMembersForUncached(head, s)
		if len(all) == 0 {
			continue
		}
		concrete := make([]*ElementDecl, 0, len(all))
		for _, m := range all {
			if !m.Abstract {
				concrete = append(concrete, m)
			}
		}
		var byName map[QName]*ElementDecl
		if len(all) >= substClosureByNameThreshold {
			byName = make(map[QName]*ElementDecl, len(all))
			for _, m := range all {
				byName[m.Name] = m
			}
		}
		closures[name] = &substClosure{head: head, all: all, concrete: concrete, byName: byName}
	}
	s.substClosures = closures
}

// closureLookupResult is lookupSubstClosure's verdict on how a caller should
// proceed for a given (edecl, schema) pair.
type closureLookupResult int

const (
	// closureUncached means the cache cannot answer authoritatively — schema
	// is still compiling (schema.substClosures is nil) or edecl is a ref whose
	// resolved head could not be identified from the cache. The caller must
	// fall back to substitutableMembersForUncached.
	closureUncached closureLookupResult = iota
	// closureNone means the cache authoritatively says "no members": edecl
	// blocks its own substitution, is a local particle sharing a head's
	// QName, or is a registered global head with zero substitution-group
	// members. The caller returns nil (or, for instanceSubstMembers, nil)
	// without touching the uncached walk.
	closureNone
	// closureHit means c is the authoritative cached closure for edecl.
	closureHit
)

// lookupSubstClosure resolves the cached substClosure for edecl, if any, with
// exactly one map lookup (schema.substClosures[edecl.Name]) on the
// closureHit/closureNone paths — the same lookup count as the pre-cache
// "is this the registered global?" guard it replaces. edecl and schema may be
// nil.
//
// A ref particle is served from its resolved head's entry only when its
// Block/Type still match the head's: parseLocalElement builds a ref="..."
// declaration with no Block of its own, and resolveRefs then copies
// Name/Type/Abstract/Block from the global head, so a resolved ref's closure
// inputs are identical to its head's by construction today. The equality
// check is a cheap assertion of that invariant — if a future reader change
// ever violates it, this falls back to the uncached walk instead of serving a
// wrong closure.
func lookupSubstClosure(edecl *ElementDecl, schema *Schema) (*substClosure, closureLookupResult) {
	if edecl == nil || schema == nil || edecl.Block&BlockSubstitution != 0 {
		return nil, closureNone
	}
	if schema.substClosures == nil {
		return nil, closureUncached
	}
	c, ok := schema.substClosures[edecl.Name]
	if !ok {
		if !edecl.IsRef {
			return nil, closureNone // local particle sharing a head's QName
		}
		return nil, closureUncached // unresolved ref
	}
	if c.head != edecl && (!edecl.IsRef || edecl.Block != c.head.Block || edecl.Type != c.head.Type) {
		if !edecl.IsRef {
			return nil, closureNone
		}
		return nil, closureUncached
	}
	return c, closureHit
}

// substitutableMembersForUncached is the transitive substitution-group closure
// walk: every element declaration validly substitutable for edecl, in
// breadth-first affiliation order, filtered by block and type-derivation
// rules (Substitution Group OK / cvc-elt.4.3). It is a pure function of
// edecl.Name, edecl.Block, effectiveDeclType(edecl, schema), and
// schema.substGroups/schema.elements — it reads no version state and no
// exported *Schema method mutates the schema after compilation, so the result
// is the same on every call for a given (edecl, schema) pair once schema has
// finished compiling.
func substitutableMembersForUncached(edecl *ElementDecl, schema *Schema) []*ElementDecl {
	if edecl == nil || schema == nil || edecl.Block&BlockSubstitution != 0 {
		return nil
	}
	// A substitution group is a property of the GLOBAL head element declaration
	// (or a ref to it), not of a LOCAL element particle that merely shares the
	// head's QName. Expand only when this declaration IS the registered global
	// head, or is a ref (which resolves to the global head); a distinct local
	// particle admits no substitution members even if it is named like a head.
	if !edecl.IsRef && schema.elements[edecl.Name] != edecl {
		return nil
	}
	type queuedMember struct {
		member *ElementDecl
		head   *ElementDecl
	}
	headType := effectiveDeclType(edecl, schema)
	queue := make([]queuedMember, 0, len(schema.substGroups[edecl.Name]))
	for _, member := range schema.substGroups[edecl.Name] {
		queue = append(queue, queuedMember{member: member, head: edecl})
	}
	seen := map[QName]struct{}{edecl.Name: {}}
	members := make([]*ElementDecl, 0, len(queue))
	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]
		member := item.member
		if member == nil {
			continue
		}
		if _, ok := seen[member.Name]; ok {
			continue
		}
		memberType := effectiveDeclType(member, schema)
		curHeadType := effectiveDeclType(item.head, schema)
		// The head's EFFECTIVE {disallowed substitutions} unions the head element's
		// block with its declared TYPE's {prohibited substitutions} (Substitution
		// Group OK / cvc-elt.4.3), so a member reached by a derivation method the
		// intermediate OR original head's TYPE blocks is not admitted; the
		// substitution-specific walk also honors any INTERMEDIATE type's block on the
		// member's derivation chain.
		if substTypeDerivationBlocked(memberType, curHeadType, item.head.Block) {
			continue
		}
		if substTypeDerivationBlocked(memberType, headType, edecl.Block) {
			continue
		}
		seen[member.Name] = struct{}{}
		members = append(members, member)
		if member.Block&BlockSubstitution != 0 {
			continue
		}
		for _, child := range schema.substGroups[member.Name] {
			queue = append(queue, queuedMember{member: child, head: member})
		}
	}
	return members
}
