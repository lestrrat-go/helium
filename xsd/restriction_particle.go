package xsd

import (
	"context"
	"maps"
	"slices"

	"github.com/lestrrat-go/helium/internal/lexicon"
)

// checkRestrictionParticles implements (a conservative subset of) the XSD 1.1
// derivation-ok-restriction constraint for the CONTENT MODEL of a complexContent
// restriction (§3.9.6, "Particle Valid (Restriction)"). It complements
// checkRestrictionAttrs (which only covers attribute uses) by verifying that the
// derived type's effective content model is a valid restriction of the base
// type's: each derived particle must map, order-preservingly, onto a base
// particle it restricts (name/type/occurrence), no derived particle may be added
// that the base does not allow, and any base particle not matched by the
// derivation must be emptiable.
//
// The check is intentionally CONSERVATIVE: whenever a sub-case cannot be decided
// with confidence (e.g. exotic wildcard interactions, substitution-group
// containment, or structural shapes the recursion does not model), it treats the
// derivation as VALID rather than risk a false rejection of a legitimate schema.
// Its job is to catch the clear violations — reordered particles, added/renamed
// particles, and widened occurrence ranges — without breaking the golden suite.
func (c *compiler) checkRestrictionParticles(ctx context.Context, td *TypeDef) {
	if c.filename == "" {
		return
	}
	src, hasSrc := c.typeDefSources[td]
	if !hasSrc {
		return
	}
	base := td.BaseType
	if base == nil {
		return
	}

	// Restriction directly off the ur-type (xs:anyType) is unconstrained: any
	// content model is a valid restriction of xs:anyType, so accept regardless of
	// the derived/base model groups. This must be checked BEFORE the nil-model
	// handling below, otherwise an xs:anyType base (which carries no usable
	// content model) would fall into the baseMG==nil reject path.
	if base.Name.Local == typeAnyType && base.Name.NS == lexicon.NamespaceXSD {
		return
	}

	derivedMG := td.ContentModel
	baseMG := base.ContentModel

	// A derived restriction with NO content model restricts the base to empty
	// content. That is only a valid restriction when the base content model is
	// itself emptiable (can be satisfied by zero elements); restricting a base
	// that requires content down to empty content is a violation. When the base
	// also has no content model (empty/simple base), there is nothing to recurse
	// against and the content-type rules (cos-ct-extends) already cover it.
	if derivedMG == nil {
		if baseMG == nil {
			return
		}
		baseP := &Particle{MinOccurs: baseMG.MinOccurs, MaxOccurs: baseMG.MaxOccurs, Term: baseMG}
		if particleEmptiable(baseP) {
			return
		}
		c.reportInvalidRestriction(ctx, td, base, src)
		return
	}
	// The derived type has a content model but the base does not. A base without a
	// model group is either an EMPTY content type (attribute-only) or a SIMPLE
	// content type (<xs:simpleContent>, which carries character content). A derived
	// model group that emits NO content — e.g. an empty <xs:sequence/> — restricts
	// the base to EMPTY element content. That is a valid restriction of an EMPTY
	// (or mixed-emptiable) base — §3.9.6: the empty sequence is a subset of the
	// base's empty content type — but NOT of a SIMPLE-content base: a simple
	// (character) content type cannot be restricted down to empty element content
	// (cos-ct-restricts §3.4.6.4 forbids simple→empty). So accept the non-emitting
	// derived model only for a non-simple base; a derived model group with emitting
	// particles is rejected either way. (The xs:anyType base case is handled above.)
	if baseMG == nil {
		if base.ContentType != ContentTypeSimple && !modelGroupHasContent(derivedMG) {
			return
		}
		c.reportInvalidRestriction(ctx, td, base, src)
		return
	}

	// Wrap the top-level model groups as particles so the occurrence range of the
	// whole group participates in the check.
	derivedP := &Particle{MinOccurs: derivedMG.MinOccurs, MaxOccurs: derivedMG.MaxOccurs, Term: derivedMG}
	baseP := &Particle{MinOccurs: baseMG.MinOccurs, MaxOccurs: baseMG.MaxOccurs, Term: baseMG}

	// XSD 1.1: thread the BASE type's EFFECTIVE {open content} into the check so the
	// deep wildcard-restricts-model-group decision can tell whether a derived declared
	// wildcard's children are governed as the base's OPEN content (delegating that
	// interaction to the §3.4.6.4 quadrant-B guard, checkDerivedWildcardReadmitsBaseOpen)
	// rather than the base declared model group. Recomputed read-only because
	// resolveOpenContent has not yet populated base.OpenContent.
	if c.version == Version11 {
		if baseOC := c.effectiveOpenContentReadonly(base, map[*TypeDef]bool{}); baseOC != nil && baseOC.Wildcard != nil {
			ctx = withBaseOpenContent(ctx, baseOC)
		}
	}

	if particleValidRestriction(ctx, derivedP, baseP, c.schema, c.version) {
		return
	}

	// XSD 1.1 relaxes Particle Valid (Restriction): a derived content model is a
	// valid restriction whenever its language is a subset of the base's (with
	// type-compatible element declarations), even where the 1.0 syntactic clauses
	// reject it. As a SOUND, fail-closed fallback, prove L(derived) ⊆ L(base) by
	// automaton product simulation. XSD 1.0 keeps the syntactic verdict.
	if c.version == Version11 && particleLanguageSubset(ctx, derivedP, baseP, c.schema, c.version) {
		return
	}

	c.reportInvalidRestriction(ctx, td, base, src)
}

// reportInvalidRestriction emits the fatal derivation-ok-restriction diagnostic
// for a complexContent restriction whose content model does not validly restrict
// its base.
func (c *compiler) reportInvalidRestriction(ctx context.Context, td, base *TypeDef, src typeDefSource) {
	component := componentLocalComplexType
	if !src.isLocal {
		component = "complex type '" + td.Name.Local + "'"
	}
	baseQualified := "'{" + base.Name.NS + "}" + base.Name.Local + "'"
	msg := "The content model is not a valid restriction of the content model of the base complex type definition " + baseQualified + "."
	c.schemaError(ctx, schemaComponentError(c.diagSourceOrRecorded(src.source), src.line, "complexType", component, msg))
}

// baseOpenContentKey carries (in ctx) the BASE type's effective {open content} for
// the current restriction check, so the deep wildcard-restricts-model-group decision
// can tell whether a derived declared wildcard's children are governed as the base's
// OPEN content (and thus by the §3.4.6.4 quadrant-B guard) rather than the base
// declared model group. Nil/absent means the base carries no open content.
type baseOpenContentKey struct{}

func withBaseOpenContent(ctx context.Context, oc *OpenContent) context.Context {
	return context.WithValue(ctx, baseOpenContentKey{}, oc)
}

func baseOpenContentFromContext(ctx context.Context) *OpenContent {
	oc, _ := ctx.Value(baseOpenContentKey{}).(*OpenContent)
	return oc
}

// particleValidRestriction reports whether the restriction particle r is a valid
// restriction of the base particle b. Returning true means "accepted" — and, per
// the conservative contract above, it also returns true for any case the
// recursion is not confident enough to reject.
func particleValidRestriction(ctx context.Context, r, b *Particle, schema *Schema, version Version) bool {
	switch rt := r.Term.(type) {
	case *ElementDecl:
		switch bt := b.Term.(type) {
		case *ElementDecl:
			// NameAndTypeOK
			return elementRestrictsElement(ctx, r, rt, b, bt, schema, version)
		case *Wildcard:
			// NSCompat: element restricts wildcard — the element's name must be
			// allowed by the wildcard and occurrence must be a valid restriction.
			if !occurrenceValidRestriction(r.MinOccurs, r.MaxOccurs, b.MinOccurs, b.MaxOccurs) {
				return false
			}
			return wildcardAllowsName(bt, rt.Name, schema)
		case *ModelGroup:
			// Recurse-As-If-Group (XSD §3.9.6): a derived element against a base
			// model group. Treat the element as a singleton group and map it through
			// the base group's compositor-specific children.
			return elementRestrictsGroup(ctx, r, b, bt, schema, version)
		}
	case *Wildcard:
		switch bt := b.Term.(type) {
		case *Wildcard:
			// NSSubset: the derived wildcard restricts the base wildcard iff its
			// occurrence range is a subset, its namespace constraint is a subset, and
			// its processContents is at least as strong as the base's (strict > lax >
			// skip — a restriction may tighten but never weaken validation).
			if !occurrenceValidRestriction(r.MinOccurs, r.MaxOccurs, b.MinOccurs, b.MaxOccurs) {
				return false
			}
			if processContentsStrength(rt.ProcessContents) < processContentsStrength(bt.ProcessContents) {
				return false
			}
			return wildcardConstraintSubset(rt, bt, schema, false)
		case *ElementDecl:
			// A wildcard can never be a restriction of a single element. This is a
			// clear violation.
			return false
		case *ModelGroup:
			// §3.9.6 (Particle Valid (Restriction)) has NO derivation rule for a
			// WILDCARD restricting a base MODEL GROUP. A restriction is valid only when
			// L(derived) ⊆ L(base), so this case is SOUND / fail-closed: it accepts
			// ONLY the sub-cases it can rigorously prove. This is version-INDEPENDENT —
			// the particleLanguageSubset fallback (Version11-only) cannot model a
			// derived wildcard, so it never proves anything here; deciding directly is
			// the only sound option.
			//
			// EXCEPTION (XSD 1.1 open content): a child matching the derived declared
			// wildcard may be governed as the BASE's OPEN content rather than by the base
			// declared model group — but ONLY when all hold:
			//   1. the base model group at this position is EMPTIABLE (skippable), so a
			//      document that drops it entirely and supplies only wildcard-matched
			//      (open-content) children is still valid against the base;
			//   2. the derived wildcard's namespace is a SUBSET of the base open-content
			//      wildcard, so every name it admits actually lands in the open-content
			//      region (a name outside it is admitted by NEITHER the emptiable base
			//      model nor the open content → not a subset); and
			//   3. the ORDERING the base open content imposes is preserved:
			//      - INTERLEAVE imposes NO ordering, so an open-content-governed child may
			//        appear anywhere; a derived declared wildcard at ANY position keeps the
			//        language a subset — always OK.
			//      - SUFFIX requires open-content children to TRAIL every declared element,
			//        but a derived declared wildcard occupies a NON-trailing position in
			//        this recursion (it is mapped against a base model-group particle, not
			//        the trailing region), so a child it admits could appear BEFORE required
			//        declared content the base forces to come first — the suffix ordering is
			//        not preserved. So a SUFFIX base is delegated only when the derived
			//        wildcard is PROVEN to admit NOTHING, i.e. a STRICT wildcard resolving no
			//        globally-declared element (its language is empty, so no non-trailing
			//        child can arise). Otherwise fail closed. (The ε/∅-NAMESPACE cases are
			//        decided by wildcardRestrictsModelGroup below.)
			// When all hold, the §3.4.6.4 quadrant-B guard
			// (checkDerivedWildcardReadmitsBaseOpen) enforces the remaining soundness —
			// processContents at least as strong — so delegate to it rather than
			// double-reject here. A NON-emptiable base group (its required content is
			// dropped), a wildcard reaching outside the base open content, or a suffix base
			// whose derived wildcard can admit a child is NOT covered: fall through to the
			// sound wildcardRestrictsModelGroup decision.
			if version == Version11 {
				if baseOC := baseOpenContentFromContext(ctx); baseOC != nil && baseOC.Wildcard != nil &&
					particleEmptiable(b) &&
					wildcardConstraintSubset(rt, baseOC.Wildcard, schema, false) &&
					(baseOC.Mode == OpenContentInterleave || strictWildcardAdmitsNoGlobal(rt, schema)) {
					return true
				}
			}
			return wildcardRestrictsModelGroup(r, rt, b, bt, schema)
		}
	case *ModelGroup:
		switch bt := b.Term.(type) {
		case *ModelGroup:
			return groupRestrictsGroup(ctx, r, rt, b, bt, schema, version)
		case *Wildcard:
			// NSRecurseCheckCardinality (XSD §3.9.6): the derived group's effective
			// occurrence range must be within the base wildcard's range, and every
			// element/wildcard LEAF inside the derived group must be admitted by the
			// base wildcard (namespace) and within its cardinality. The base
			// wildcard is reached through the base particle b (b.Term).
			return groupRestrictsWildcard(r, rt, b, schema)
		case *ElementDecl:
			// A derived model GROUP restricting a base single ELEMENT. XSD 1.0 §3.9.6
			// has NO Sequence/Choice/All:Element derivation rule, so this is valid ONLY
			// when the group is §3.9.6-POINTLESS — it folds (safe occurrence hoisting:
			// at each level the group's or the single member's {max occurs} is 1) down
			// to a single element particle that validly restricts the base element. A
			// genuinely REPEATING group (e.g. sequence maxOccurs="2" of element{1,2},
			// which emits the element 1..4 times) is NOT pointless and admits content the
			// base single element does not, so it is rejected. XSD 1.1 keeps the broader
			// language-inclusion leniency (groupRestrictsElement), backed by the
			// particleLanguageSubset fallback, so its behavior is unchanged.
			if version == Version10 {
				red, reduced := pointlessReduce(r)
				if !reduced {
					return false
				}
				return particleValidRestriction(ctx, red, b, schema, version)
			}
			return groupRestrictsElement(ctx, r, rt, b, bt, schema, version)
		}
	}
	return true
}

// elementRestrictsElement checks the element-to-element (NameAndTypeOK) case:
// same expanded name, occurrence range subset, and the derived element's type is
// derived from (or equal to) the base element's type. nillable/fixed tightening
// is checked conservatively.
func elementRestrictsElement(ctx context.Context, r *Particle, rt *ElementDecl, b *Particle, bt *ElementDecl, schema *Schema, version Version) bool {
	if rt.Name.Local != bt.Name.Local || rt.Name.NS != bt.Name.NS {
		return false
	}
	if !occurrenceValidRestriction(r.MinOccurs, r.MaxOccurs, b.MinOccurs, b.MaxOccurs) {
		return false
	}
	// Type derivation: the derived element's type must be the same as, or derived
	// from, the base element's type. When either type is unresolved, accept
	// conservatively. In XSD 1.1, when the base element's type is a union, a
	// derived type validly derived from one of the union's (transitive) members
	// is also a valid restriction (member substitutability), so use the
	// union-aware predicate rather than a plain base-chain walk.
	if rt.Type != nil && bt.Type != nil {
		ok := isDerivedFrom(rt.Type, bt.Type)
		if !ok && version == Version11 {
			ok = isXsiTypeDerivedFromDeclared(rt.Type, bt.Type)
		}
		if !ok {
			return false
		}
	}
	// XSD 1.1 Particle Valid (Restriction) clause 4.6: the derived and base element
	// declarations' {type table}s must be both absent or both present and
	// EQUIVALENT (conditional type assignment). A restriction whose type table
	// selects a different type for the same @test than the base's is invalid
	// (cta0043). Equivalence is the conservative structural comparison.
	if version == Version11 && !typeTablesEquivalent(elementAlternatives(rt, schema), elementAlternatives(bt, schema)) {
		return false
	}
	// Particle Valid (Restriction), NameAndTypeOK clause 3.2.4 (§3.9.6,
	// version-INDEPENDENT): R's declaration's {disallowed substitutions} must be a
	// SUPERSET of B's. In flag terms every block bit set on the base element must
	// also be set on the derived element — a restriction may TIGHTEN the disallowed
	// set but never LOOSEN it (e.g. a base block="#all" cannot be restricted by a
	// derived block="extension"). Both Block values already fold in blockDefault
	// when the attribute is absent, so the comparison is on the effective sets.
	if bt.Block&^rt.Block != 0 {
		return false
	}
	// A base element that is fixed forces the derived element to carry the same
	// fixed value; a base that is not nillable forbids the derived from becoming
	// nillable. These are tightening rules — only flag the clear widening cases.
	if !bt.Nillable && rt.Nillable {
		return false
	}
	if bt.Fixed != nil {
		if rt.Fixed == nil {
			return false
		}
		// Compare the two fixed values in the element's VALUE SPACE, not lexically:
		// a derived fixed that is value-space-equal but lexically different from the
		// base (e.g. base "1" vs derived "01" for xs:integer) is a valid
		// restriction. Reuse the same value-space comparator instance validation
		// uses. Both fixed values are schema-declared, so they resolve any QName
		// prefixes against their respective FixedNS bindings. The element
		// declaration's simple type drives the comparison; when it is unresolved,
		// fixedValueMatches falls back to lexical equality.
		if !fixedValueMatches(ctx, *rt.Fixed, *bt.Fixed, rt.Type, rt.FixedNS, bt.FixedNS, schema, version) {
			return false
		}
	}
	return true
}

// groupRestrictsGroup handles the model-group cases (recurse / map-and-sum). It
// requires the derived group's occurrence range to be a valid restriction of the
// base's, then dispatches on compositor.
// groupHasWildcardFlat reports whether a model group has a wildcard particle
// reachable after flattening nested 1/1 all-groups (an xs:group ref to an all
// that carries an xs:any). A direct-only check would miss a nested all-group ref
// containing a wildcard and route it to the counting fast-path (which rejects
// wildcards) instead of the wildcard-aware subsumption.
func groupHasWildcardFlat(g *ModelGroup) bool {
	for _, p := range flattenAllParticles(g.Particles) {
		if _, ok := p.Term.(*Wildcard); ok {
			return true
		}
	}
	return false
}

func groupRestrictsGroup(ctx context.Context, r *Particle, rg *ModelGroup, b *Particle, bg *ModelGroup, schema *Schema, version Version) bool {
	// XSD 1.1 occurrence-counting subsumption of a base xs:all: a derived xs:all,
	// xs:sequence, or xs:choice restricting a base all is checked by summing the
	// derived side's per-member occurrence contributions (several derived
	// particles, e.g. substitution-group members, may collectively restrict one
	// base member with maxOccurs>1). recurseAll's 1:1 distinct mapping cannot
	// express this. The wildcard cases still route to allRestrictsWithWildcards.
	if version == Version11 && bg.Compositor == CompositorAll &&
		!groupHasWildcardFlat(rg) && !groupHasWildcardFlat(bg) {
		// An empty (non-emitting) derived particle restricts the base all to empty
		// content; valid iff the base all particle itself is emptiable.
		if particleEmitsNothing(r) {
			return particleEmptiable(b)
		}
		// An EMPTIABLE base all particle already accepts the empty sequence, so a
		// derived particle whose minOccurs falls below the base all's declared
		// minOccurs (e.g. base all{1,1} of optional members narrowed to all{0,1})
		// never admits content the base rejects — occurrenceEmptiableRestriction
		// lowers the base's effective minOccurs to 0. Gated on Version11.
		if !occurrenceEmptiableRestriction(r, b, version) {
			return false
		}
		return allRestrictsByCounting(ctx, r, bg, schema, version)
	}
	switch {
	case rg.Compositor == CompositorSequence && bg.Compositor == CompositorSequence:
		if !occurrenceValidRestriction(r.MinOccurs, r.MaxOccurs, b.MinOccurs, b.MaxOccurs) {
			return false
		}
		return recurseOrdered(ctx, rg.Particles, bg.Particles, schema, version)
	case rg.Compositor == CompositorAll && bg.Compositor == CompositorAll:
		if !occurrenceEmptiableRestriction(r, b, version) {
			return false
		}
		// XSD 1.1: an xs:all may contain element wildcards. recurseAll maps each
		// derived particle to ONE base particle, but a derived wildcard may need
		// the UNION of the base all's wildcards, so use the wildcard-aware check.
		if version == Version11 && (groupHasWildcardFlat(rg) || groupHasWildcardFlat(bg)) {
			return allRestrictsWithWildcards(ctx, rg.Particles, bg.Particles, schema, version)
		}
		return recurseAll(ctx, rg.Particles, bg.Particles, schema, version)
	case rg.Compositor == CompositorChoice && bg.Compositor == CompositorChoice:
		if !occurrenceEmptiableRestriction(r, b, version) {
			return false
		}
		return recurseChoiceUnordered(ctx, rg.Particles, bg.Particles, schema, version)
	case rg.Compositor == CompositorSequence && bg.Compositor == CompositorChoice:
		// MapAndSum (XSD §3.9.6): a derived SEQUENCE restricting a base CHOICE. Every
		// member of the derived sequence must be a valid restriction of SOME branch
		// of the base choice, AND the total number of elements the derived sequence
		// can emit must be within the base choice particle's occurrence range — a
		// base choice{bMin,bMax} accepts between bMin and bMax elements (each one
		// matching a branch), so a derived sequence that emits [dMin,dMax] elements
		// is valid iff bMin <= dMin and dMax <= bMax. The comparison is on the
		// derived's TOTAL element-emission range, NOT the derived group's raw
		// occurrence range (always 1,1 for a top-level sequence): base choice{2,2}
		// must accept sequence(a,b) (emits 2), while base choice(a){1,1} must still
		// reject sequence(a,b) (emits 2 > 1).
		dMin, dMax := particleElementRange(r)
		bMin, bMax := particleElementRange(b)
		if !occurrenceValidRestriction(dMin, dMax, bMin, bMax) {
			return false
		}
		for _, rp := range rg.Particles {
			// A prohibited derived member emits nothing — it admits no content the
			// base choice must accept, so it needs no matching base branch.
			if particleEmitsNothing(rp) {
				continue
			}
			if !slices.ContainsFunc(bg.Particles, func(bp *Particle) bool {
				return particleValidRestriction(ctx, rp, bp, schema, version)
			}) {
				return false
			}
		}
		return true
	default:
		// Remaining mixed-compositor pairs: choice:sequence, choice:all,
		// all:sequence, all:choice, and sequence:all. XSD §3.9.6 (Particle Valid
		// (Restriction)) defines a group:group derivation rule ONLY for matching
		// compositors (Recurse/RecurseLax), for a derived sequence against a base
		// choice (MapAndSum, handled above), and for a derived sequence against a
		// base all (RecurseUnordered). The other four pairs have NO derivation rule
		// and are invalid restrictions.
		//
		// sequence:all is RecurseUnordered (XSD §3.9.6): a derived SEQUENCE
		// restricting a base ALL. Order is irrelevant in the base all, so each
		// derived sequence particle must map to a DISTINCT base all particle it
		// validly restricts, and every base particle left unmapped must be
		// emptiable. This is exactly the all→all distinct-mapping (recurseAll) — the
		// derived side being ordered does not further constrain the unordered base —
		// so reuse it after checking the group occurrence range. A derived sequence
		// that adds/renames a particle (no distinct base counterpart) or drops a
		// required base member is rejected.
		//
		// Handle this BEFORE reduceSingletonGroup: a SINGLETON derived sequence
		// (e.g. sequence(a?) over base all(a?, b?)) must be mapped member-by-member
		// through recurseAll against the base all's children. Folding it to a bare
		// element first would route it to elementRestrictsGroup, which compares the
		// lone element against every base all child and over-rejects a valid
		// RecurseUnordered restriction that simply leaves an emptiable base member
		// unmatched.
		if rg.Compositor == CompositorSequence && bg.Compositor == CompositorAll {
			// An explicit empty derived sequence (<xs:sequence/>, no emitting
			// particles) restricts the base all to empty content. That is valid iff
			// the base all PARTICLE itself is emptiable (e.g. minOccurs="0") —
			// semantically the same empty-content restriction the derivedMG==nil path
			// handles. This shortcut is checked BEFORE the occurrence-range subset:
			// a non-emitting derived particle admits no content, so its raw group
			// occurrence is irrelevant (e.g. <xs:sequence maxOccurs="2"/> still emits
			// nothing). Gating it behind occurrenceValidRestriction would false-reject
			// a valid empty-language restriction whose raw occurrence is not a subset
			// of the base all's. recurseAll would instead wrongly demand every base
			// all CHILD be individually emptiable, ignoring that an optional base all
			// particle accepts zero elements even when its members are required.
			if particleEmitsNothing(r) {
				return particleEmptiable(b)
			}
			if !occurrenceValidRestriction(r.MinOccurs, r.MaxOccurs, b.MinOccurs, b.MaxOccurs) {
				return false
			}
			// XSD 1.1 all-with-wildcards: a derived wildcard may be covered by the
			// UNION of the base all's wildcards, which recurseAll's 1:1 mapping
			// cannot express.
			if version == Version11 && (groupHasWildcardFlat(rg) || groupHasWildcardFlat(bg)) {
				return allRestrictsWithWildcards(ctx, rg.Particles, bg.Particles, schema, version)
			}
			return recurseAll(ctx, rg.Particles, bg.Particles, schema, version)
		}
		// Remaining mixed pairs: choice:sequence, choice:all, all:sequence,
		// all:choice — no §3.9.6 derivation rule. Before rejecting, fold away
		// "pointless" single-emitting-child wrappers on either side and re-dispatch:
		// a group with exactly one emitting member is equivalent to that member, so
		// e.g. choice(a) restricting sequence(a) is a valid (element-to-element)
		// restriction once both pointless wrappers are removed. Only re-dispatch when
		// a reduction actually made progress, so the recursion terminates.
		rr := reduceSingletonGroup(r)
		bb := reduceSingletonGroup(b)
		if rr != r || bb != b {
			return particleValidRestriction(ctx, rr, bb, schema, version)
		}
		return false
	}
}

// reduceSingletonGroup folds a "pointless" model-group particle — one whose term
// is a model group with exactly one element-emitting member — down to that single
// member, repeatedly, combining the nesting occurrence ranges. A group that emits
// via exactly one member admits precisely what that member admits (repeated per
// group occurrence), so the wrapper compositor is irrelevant to the content
// model. Returning the original particle unchanged when no reduction applies lets
// callers detect "no progress" via pointer identity. Non-emitting members are
// dropped before counting, so a prohibited sibling never blocks the reduction.
func reduceSingletonGroup(p *Particle) *Particle {
	for {
		mg, ok := p.Term.(*ModelGroup)
		if !ok {
			return p
		}
		var only *Particle
		count := 0
		for _, child := range mg.Particles {
			if particleEmitsNothing(child) {
				continue
			}
			count++
			if count > 1 {
				break
			}
			only = child
		}
		if count != 1 {
			return p
		}
		p = &Particle{
			MinOccurs: mulOccurs(p.MinOccurs, only.MinOccurs),
			MaxOccurs: mulOccurs(p.MaxOccurs, only.MaxOccurs),
			Term:      only.Term,
		}
	}
}

// strictWildcardAdmitsNoGlobal reports whether a STRICT wildcard resolves NO child at
// validation time. Strict processContents accepts only an element with a matching
// GLOBAL element declaration, so a strict wildcard that admits (by namespace/notQName)
// no globally-declared element's expanded name matches nothing — its language is empty.
// Non-strict wildcards (skip/lax) accept unvalidated / laxly-assessed children, so this
// returns false for them (they can admit a child even with no matching global). Used to
// let a SUFFIX open-content delegation accept a derived declared wildcard proven to
// contribute no (non-trailing) child.
func strictWildcardAdmitsNoGlobal(wc *Wildcard, schema *Schema) bool {
	if wc.ProcessContents != ProcessStrict {
		return false
	}
	for qn := range schema.elements {
		if wildcardAllowsExpandedName(wc, qn.Local, qn.NS, schema, false) {
			return false
		}
	}
	return true
}

// wildcardRestrictsModelGroup decides the §3.9.6 WILDCARD-restricting-MODEL-GROUP
// case SOUNDLY (fail-closed): it returns true only when it can PROVE
// L(derived wildcard particle r) ⊆ L(base model group b). §3.9.6 provides no
// derivation rule for this shape, so an accept is justified only by a language
// argument.
//
// The derived side is a SINGLE uniform wildcard rt with occurrence
// [r.MinOccurs, r.MaxOccurs]; its language is the set of sequences of any length in
// that interval whose every child is a name rt admits, each validated at rt's
// processContents. For the base to be a superset it must, for every such child
// count k, accept a length-k sequence of ARBITRARY rt-admitted names ("all-wildcard"
// documents). reduceWildcardOnlyParticle computes the base's ALL-WILDCARD emission
// profile: a CONTIGUOUS count interval [lo, hi] plus a SET of pc-tagged namespace
// "cover" branches such that at every emitted position the base can independently
// admit any name in the union of the covering branches (a choice contributes a set
// of branches; a required element makes the position "dead" — no all-wildcard
// document — and an optional element contributes ε). The derived wildcard is a valid
// restriction iff:
//
//   - EMPTY language {}: a derived wildcard whose NAMESPACE constraint admits no name
//     (namespace="") yet must occur (min>0) is unsatisfiable, so L(derived)=∅ ⊆ every
//     base.
//   - {ε} language: a prohibited (maxOccurs=0) or matchesNothing-with-min0 derived
//     particle matches only ε; {ε} ⊆ L(base) iff the base is emptiable.
//   - otherwise the base reduces (no dead position, no occurrence HOLE) to
//     [lo, hi] + cover, the derived occurrence is within [lo, hi], AND rt's admitted
//     names are covered by the base branches whose processContents is no STRICTER
//     than rt's (a base branch stricter than rt would reject content rt admits, so
//     the derived would not be a subset).
//
// A STRICT wildcard resolving no globally-declared name (strictWildcardAdmitsNoGlobal)
// is ALSO empty-language and is DELIBERATELY not treated as such here: it is the common
// case in a globals-free schema (diverting every strict wildcard away from the sound
// reduction path), and accepting it over an element group would diverge from the
// §3.9.6 syntactic reject (W3C particlesHa163). The reduction below decides those cases
// soundly and conformantly without it. Anything the reduction cannot represent
// language-exactly — a dead base position, a non-uniform base position, an
// occurrence-count HOLE — is REJECTED.
func wildcardRestrictsModelGroup(r *Particle, rt *Wildcard, b *Particle, bt *ModelGroup, schema *Schema) bool {
	_ = bt // base model group reached via b.Term; kept for call-site symmetry
	matchesNothing := wildcardMatchesNothing(rt)
	switch {
	case matchesNothing && r.MinOccurs > 0:
		return true
	case r.MaxOccurs == 0 || matchesNothing:
		return particleEmptiable(b)
	}
	// Reduce the base to its ALL-WILDCARD emission profile. A dead base (a required
	// element on every path) has no all-wildcard document, so an emitting derived
	// wildcard cannot be a subset; a non-representable shape fails closed.
	red, ok := reduceWildcardOnlyParticle(b)
	if !ok || red.dead || len(red.cover) == 0 {
		return false
	}
	if !occurrenceValidRestriction(r.MinOccurs, r.MaxOccurs, red.lo, red.hi) {
		return false
	}
	return coverAdmitsWildcard(red.cover, rt, schema)
}

// coverBranch is one pc-tagged namespace contribution of a base group's
// all-wildcard emission: at a position governed by this branch the base admits any
// name in con, validated at pc. wc is the source wildcard, retained so the
// single-branch coverage check can use the full notQName/##defined-aware subset.
type coverBranch struct {
	wc  *Wildcard
	con wcConstraint
	pc  ProcessContentsKind
}

// wildcardOnlyReduction is the ALL-WILDCARD emission profile of a base particle: a
// CONTIGUOUS occurrence-count interval [lo, hi] (hi == -1 means unbounded) plus the
// SET of cover branches admissible at each emitted position. An empty cover with
// [0,0] is the ε-only reduction (the particle emits no wildcard children). dead=true
// marks a particle with NO all-wildcard document (a required element declaration on
// every path); the caller rejects an emitting derived wildcard against it.
type wildcardOnlyReduction struct {
	dead   bool
	cover  []coverBranch
	lo, hi int
}

// reduceWildcardOnlyParticle computes the all-wildcard emission profile of a base
// particle. It is SOUND and fail-closed: it returns ok=false whenever it cannot
// PROVE the profile is language-exact (a non-uniform position, or an occurrence
// combination that would introduce a count HOLE). An element declaration is handled
// rather than excluded: an emptiable (minOccurs=0) element contributes ε (an
// all-wildcard document emits zero of it), while a required (minOccurs>=1) element
// makes the position dead.
func reduceWildcardOnlyParticle(p *Particle) (wildcardOnlyReduction, bool) {
	if particleEmitsNothing(p) {
		// A prohibited/non-emitting particle contributes only the empty string.
		return wildcardOnlyReduction{lo: 0, hi: 0}, true
	}
	switch t := p.Term.(type) {
	case *Wildcard:
		if wildcardMatchesNothing(t) {
			// A matchesNothing wildcard is an ∅/ε edge case; the caller handles those
			// before reduction, so bail conservatively here.
			return wildcardOnlyReduction{}, false
		}
		return wildcardOnlyReduction{
			cover: []coverBranch{{wc: t, con: wildcardConstraint(t), pc: t.ProcessContents}},
			lo:    p.MinOccurs,
			hi:    p.MaxOccurs,
		}, true
	case *ElementDecl:
		if p.MinOccurs == 0 {
			// An optional element: an all-wildcard document emits zero of it (ε).
			return wildcardOnlyReduction{lo: 0, hi: 0}, true
		}
		// A required element: every document through this position carries it, so
		// there is no all-wildcard document here.
		return wildcardOnlyReduction{dead: true}, true
	case *ModelGroup:
		body, ok := reduceWildcardOnlyGroupBody(t)
		if !ok {
			return wildcardOnlyReduction{}, false
		}
		if body.dead {
			// An optional group over a dead body can be skipped entirely (ε); a
			// required group forces the dead body, so the position stays dead.
			if p.MinOccurs == 0 {
				return wildcardOnlyReduction{lo: 0, hi: 0}, true
			}
			return wildcardOnlyReduction{dead: true}, true
		}
		return applyOccReduction(body, p.MinOccurs, p.MaxOccurs)
	}
	return wildcardOnlyReduction{}, false
}

// reduceWildcardOnlyGroupBody combines a model group's members into one all-wildcard
// profile (BEFORE the group's own occurrence is applied): a sequence/all SUMS member
// profiles (all members contribute at distinct positions, so they must share one
// cover); a choice UNIONS the profiles of its non-dead branches. A sequence with a
// dead member is dead; a choice all of whose branches are dead is dead.
func reduceWildcardOnlyGroupBody(g *ModelGroup) (wildcardOnlyReduction, bool) {
	if g.Compositor == CompositorChoice {
		acc := wildcardOnlyReduction{}
		hasLive := false
		for _, child := range g.Particles {
			m, ok := reduceWildcardOnlyParticle(child)
			if !ok {
				return wildcardOnlyReduction{}, false
			}
			if m.dead {
				// A choice can avoid a dead branch, so it is excluded from the
				// all-wildcard union rather than killing the whole choice.
				continue
			}
			if !hasLive {
				acc = m
				hasLive = true
				continue
			}
			merged, ok := unionReduction(acc, m)
			if !ok {
				return wildcardOnlyReduction{}, false
			}
			acc = merged
		}
		if !hasLive {
			return wildcardOnlyReduction{dead: true}, true
		}
		return acc, true
	}
	// sequence / all: SUM member intervals (a Minkowski sum of contiguous integer
	// intervals stays contiguous, so no gap can arise here).
	acc := wildcardOnlyReduction{lo: 0, hi: 0}
	for _, child := range g.Particles {
		m, ok := reduceWildcardOnlyParticle(child)
		if !ok {
			return wildcardOnlyReduction{}, false
		}
		if m.dead {
			// A required member of a sequence/all forces an element at every
			// document, so no all-wildcard document exists.
			return wildcardOnlyReduction{dead: true}, true
		}
		cover, ok := mergeSequenceCovers(acc.cover, m.cover)
		if !ok {
			return wildcardOnlyReduction{}, false
		}
		acc = wildcardOnlyReduction{cover: cover, lo: acc.lo + m.lo, hi: addOccursMax(acc.hi, m.hi)}
	}
	return acc, true
}

// unionReduction unions two CHOICE-branch profiles (each already non-dead). The
// count intervals union (contiguously, else bail); the covers combine via
// unionCovers, which enforces the per-position uniformity soundness guard.
func unionReduction(a, b wildcardOnlyReduction) (wildcardOnlyReduction, bool) {
	cover, ok := unionCovers(a, b)
	if !ok {
		return wildcardOnlyReduction{}, false
	}
	lo, hi, ok := unionInterval(a.lo, a.hi, b.lo, b.hi)
	if !ok {
		return wildcardOnlyReduction{}, false
	}
	return wildcardOnlyReduction{cover: cover, lo: lo, hi: hi}, true
}

// applyOccReduction folds a group's own occurrence [gmin, gmax] over an already
// contiguous body reduction. It preserves language-exactness only when the result
// stays a CONTIGUOUS interval; a folding that would introduce a HOLE (e.g. a body
// that emits exactly 2 repeated 1..∞ times → {2,4,6,…}) bails. The cover is
// position-uniform, so repeating the body does not change it.
func applyOccReduction(body wildcardOnlyReduction, gmin, gmax int) (wildcardOnlyReduction, bool) {
	if gmax == 0 || len(body.cover) == 0 {
		// Prohibited group, or an ε-only body: emits only the empty string.
		return wildcardOnlyReduction{lo: 0, hi: 0}, true
	}
	switch {
	case gmin == gmax:
		// A fixed number of copies of a contiguous body stays contiguous.
		return wildcardOnlyReduction{cover: body.cover, lo: gmin * body.lo, hi: mulOccurs(gmin, body.hi)}, true
	case body.hi == -1:
		// Each copy already spans to ∞. With gmin >= 1 the smallest copy count
		// reaches [gmin*lo, ∞) and every larger count is a subset, so the union is
		// [gmin*lo, ∞) — contiguous. With gmin == 0 the count set is
		// {0} ∪ [body.lo, ∞): contiguous only when body.lo <= 1 (0 abuts 1). A
		// body.lo > 1 leaves a HOLE at 1..body.lo-1 (e.g. group{0,1} over any{2,∞}
		// emits {0} ∪ [2,∞), never exactly 1), so the reduction is not language-exact
		// and must fail closed — otherwise a derived any{1,1} is wrongly accepted.
		if gmin == 0 && body.lo > 1 {
			return wildcardOnlyReduction{}, false
		}
		return wildcardOnlyReduction{cover: body.cover, lo: gmin * body.lo, hi: -1}, true
	case body.lo == 0:
		// Every copy count includes 0, so the union is [0, gmax*hi] — contiguous.
		return wildcardOnlyReduction{cover: body.cover, lo: 0, hi: mulOccurs(gmax, body.hi)}, true
	default:
		// body.lo >= 1, finite body.hi, gmin < gmax. The union of consecutive copy
		// intervals [k*lo, k*hi] is gap-free iff each abuts the next:
		// k*hi + 1 >= (k+1)*lo. With hi >= lo this is monotone non-decreasing in k,
		// so checking the smallest copy count gmin suffices.
		if gmin*body.hi+1 < (gmin+1)*body.lo {
			return wildcardOnlyReduction{}, false
		}
		return wildcardOnlyReduction{cover: body.cover, lo: gmin * body.lo, hi: mulOccurs(gmax, body.hi)}, true
	}
}

// addOccursMax adds two maxOccurs values, treating -1 as unbounded.
func addOccursMax(a, b int) int {
	if a == -1 || b == -1 {
		return -1
	}
	return a + b
}

// unionInterval merges two contiguous occurrence intervals into one, reporting
// whether the union stays contiguous (the intervals overlap or are adjacent). -1
// is unbounded.
func unionInterval(alo, ahi, blo, bhi int) (int, int, bool) {
	if alo > blo {
		alo, ahi, blo, bhi = blo, bhi, alo, ahi
	}
	// [alo, ahi] is the lower interval. It must reach (adjacently) into [blo, bhi].
	if ahi != -1 && ahi+1 < blo {
		return 0, 0, false
	}
	if ahi == -1 || bhi == -1 {
		return alo, -1, true
	}
	return alo, max(ahi, bhi), true
}

// mergeSequenceCovers combines the covers of two SEQUENCE/ALL members. Because the
// members occupy DIFFERENT positions of the same document, every position must admit
// the SAME name-set for a single uniform derived wildcard to cover them: an empty
// (ε) cover adopts the other, otherwise both must be a SINGLE branch over the SAME
// namespace constraint (the merged branch keeps the STRICTER processContents, since
// the derived wildcard must satisfy the strictest position it covers). A
// multi-branch (choice-derived) member inside a sequence with another non-ε member,
// or two members over different namespaces, is not position-uniform and fails closed.
func mergeSequenceCovers(a, b []coverBranch) ([]coverBranch, bool) {
	if len(a) == 0 {
		return b, true
	}
	if len(b) == 0 {
		return a, true
	}
	if len(a) != 1 || len(b) != 1 || !sameConstraint(a[0].con, b[0].con) {
		return nil, false
	}
	// A sequence is CONJUNCTIVE — every position of the merged run must hold. Two
	// members whose namespace matches but whose name-level 1.1 exclusions
	// (notQName/##defined) differ have DIFFERENT admitted name-sets per position, so
	// collapsing them to one branch would silently drop an exclusion. Fail closed
	// unless neither member carries a name-level exclusion (then the shared namespace
	// constraint fully describes both positions).
	if coverBranchHasNameFields(a[0]) || coverBranchHasNameFields(b[0]) {
		return nil, false
	}
	stronger := a[0]
	if processContentsStrength(b[0].pc) > processContentsStrength(a[0].pc) {
		stronger = b[0]
	}
	return []coverBranch{stronger}, true
}

// coverBranchHasNameFields reports whether a cover branch's source wildcard carries a
// NAME-level 1.1 exclusion (notQName / ##defined / ##definedSibling) that the
// namespace-only wcConstraint does not capture.
func coverBranchHasNameFields(br coverBranch) bool {
	return len(br.wc.NotQName) > 0 || br.wc.NotQNameDefined || br.wc.NotQNameDefinedSibling || len(br.wc.SiblingNames) > 0
}

// unionCovers combines the covers of two CHOICE branches. A choice picks one branch
// per document, so its all-wildcard positions may independently be any of the
// branches' names ONLY when the combination stays position-uniform: if the two
// covers are IDENTICAL (same (constraint, processContents) branches) the union is
// themselves at any occurrence; otherwise the profiles differ per branch, so each
// side must emit AT MOST ONE child (hi in {0,1}) — a multi-child run would force a
// same-branch namespace/pc block that the uniform union wildcard cannot express.
// The result is the deduplicated union of the branch sets, preserving each branch's
// own processContents (the choice offers ALL of them at each position).
func unionCovers(a, b wildcardOnlyReduction) ([]coverBranch, bool) {
	if len(a.cover) == 0 {
		return b.cover, true
	}
	if len(b.cover) == 0 {
		return a.cover, true
	}
	if !coversIdentical(a.cover, b.cover) {
		if a.hi == -1 || a.hi > 1 || b.hi == -1 || b.hi > 1 {
			return nil, false
		}
	}
	out := append([]coverBranch(nil), a.cover...)
	for _, y := range b.cover {
		dup := false
		for _, x := range out {
			if x.pc == y.pc && sameConstraint(x.con, y.con) {
				dup = true
				break
			}
		}
		if !dup {
			out = append(out, y)
		}
	}
	return out, true
}

// coversIdentical reports whether two covers describe the SAME set of
// (namespace-constraint, processContents) branches, order-independently.
func coversIdentical(a, b []coverBranch) bool {
	if len(a) != len(b) {
		return false
	}
	used := make([]bool, len(b))
	for _, x := range a {
		found := false
		for j, y := range b {
			if !used[j] && x.pc == y.pc && sameConstraint(x.con, y.con) {
				used[j] = true
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// coverAdmitsWildcard reports whether the derived wildcard rt's language is covered
// by a base all-wildcard cover: rt's admitted names must all fall in the union of
// the cover branches whose processContents is NO STRICTER than rt's (a stricter base
// branch would reject content rt admits at that name, so the derived would not be a
// subset). A single covering branch uses the full notQName/##defined-aware wildcard
// subset; multiple covering branches fall back to the namespace-constraint union and
// fail closed if any carries a name-level 1.1 exclusion (which the union cannot
// represent soundly). rt's own name-level exclusions only NARROW it, so ignoring
// them there stays conservative.
func coverAdmitsWildcard(cover []coverBranch, rt *Wildcard, schema *Schema) bool {
	wStrength := processContentsStrength(rt.ProcessContents)
	var covering []coverBranch
	for _, br := range cover {
		if processContentsStrength(br.pc) <= wStrength {
			covering = append(covering, br)
		}
	}
	if len(covering) == 0 {
		return false
	}
	if len(covering) == 1 {
		return wildcardConstraintSubset(rt, covering[0].wc, schema, false)
	}
	union := wcConstraint{set: map[string]struct{}{}}
	for _, br := range covering {
		if coverBranchHasNameFields(br) {
			return false
		}
		union = constraintUnion(union, br.con)
	}
	return constraintSubset(wildcardConstraint(rt), union)
}

// sameConstraint reports whether two normalized namespace constraints admit exactly
// the same set of namespaces.
func sameConstraint(a, b wcConstraint) bool {
	return a.neg == b.neg && maps.Equal(a.set, b.set)
}

// constraintSubset reports whether every namespace sub admits is also admitted by
// super (a pure namespace-set subset, ignoring name-level exclusions).
func constraintSubset(sub, super wcConstraint) bool {
	switch {
	case !super.neg && sub.neg:
		return false
	case !super.neg && !sub.neg:
		for ns := range sub.set {
			if _, ok := super.set[ns]; !ok {
				return false
			}
		}
		return true
	case super.neg && !sub.neg:
		for ns := range sub.set {
			if _, ok := super.set[ns]; ok {
				return false
			}
		}
		return true
	default: // both negated
		for ns := range super.set {
			if _, ok := sub.set[ns]; !ok {
				return false
			}
		}
		return true
	}
}

// pointlessReduce folds a §3.9.6-POINTLESS model-group particle down to its
// single element-emitting member, applying SAFE occurrence hoisting only: a
// level folds when the group has exactly one emitting member AND the
// bounds-multiplication is EXACT — the folded range preserves the group's true
// emission count-set with no HOLE. It returns the reduced particle and reports
// whether it fully reduced to a NON-group term (an element or wildcard). A group
// whose fold would introduce a hole (e.g. a genuinely repeating group, or a
// group{0,1} over a member with minOccurs >= 2, which emits {0} ∪ [mmin, mmax])
// is NOT pointless and does not reduce, so the caller rejects it (there is no
// Sequence/Choice:Element rule in XSD 1.0). Unlike reduceSingletonGroup this
// refuses any non-language-preserving fold.
func pointlessReduce(p *Particle) (*Particle, bool) {
	for {
		mg, ok := p.Term.(*ModelGroup)
		if !ok {
			return p, true
		}
		var only *Particle
		count := 0
		for _, child := range mg.Particles {
			if particleEmitsNothing(child) {
				continue
			}
			count++
			if count > 1 {
				break
			}
			only = child
		}
		if count != 1 {
			return p, false
		}
		// Safe hoist only when the bounds-multiplication is EXACT — the folded
		// range [gmin*mmin, gmax*mmax] must equal the group's TRUE emission set
		// {g*m}, with no occurrence-count HOLE. §3.9.6 pointlessness permits
		// erasing a group wrapper only when it preserves the language. This is a
		// CONSERVATIVE, fail-closed subset: two cheaply-recognized hole-free shapes
		// are accepted, and any fold this predicate does NOT recognize is left for
		// the caller to reject. The accepted shapes:
		//   - the member occurs at most once (only.MaxOccurs == 1): each group
		//     iteration contributes 0 or 1, so the total sweeps its range
		//     contiguously; or
		//   - the group occurs at most once (p.MaxOccurs == 1) AND it is either a
		//     pure {1,1} wrapper (min == max) or its member's minOccurs <= 1, so
		//     the "zero iterations" count 0 abuts the member's [mmin, mmax] with no
		//     gap.
		// Some genuinely hole-free folds are NOT recognized and are conservatively
		// rejected — e.g. a fixed-count wrapper group{2,2}(member{3,3}) emits
		// exactly {6} (hole-free) yet neither side has maxOccurs == 1. Rejecting a
		// pointless group is always SOUND (it only forgoes an accept, never admits
		// an invalid restriction); the W3C XSD 1.0 suite shows no conformance
		// regression from this conservatism.
		// A group{0,1} over a member with minOccurs >= 2 emits {0} ∪ [mmin, mmax]
		// (a hole at 1..mmin-1) yet folds to [0, mmax], so it is NOT pointless and
		// does not reduce (the caller then rejects, matching strict §3.9.6).
		//
		// A PROHIBITED (maxOccurs==0) group or member never reaches this guard, so
		// widening it to "reject only when BOTH sides can repeat" would only
		// re-admit the group{0,1}×member{2,2} hole above — it must stay STRICT.
		// A prohibited SINGLE member cannot be `only`: it emits nothing, so the
		// loop above skips it via particleEmitsNothing (like reduceSingletonGroup),
		// hence only.MaxOccurs >= 1 always. A prohibited derived GROUP emits nothing
		// too, and every caller (recurseOrdered/recurseChoiceUnordered/recurseAll,
		// the sequence:choice map-and-sum, and the top-level ModelGroup base) drops
		// a non-emitting derived particle BEFORE particleValidRestriction, so a
		// prohibited group is filtered out before it could fold here.
		safe := only.MaxOccurs == 1 ||
			(p.MaxOccurs == 1 && (p.MinOccurs == p.MaxOccurs || only.MinOccurs <= 1))
		if !safe {
			return p, false
		}
		p = &Particle{
			MinOccurs: mulOccurs(p.MinOccurs, only.MinOccurs),
			MaxOccurs: mulOccurs(p.MaxOccurs, only.MaxOccurs),
			Term:      only.Term,
		}
	}
}

// recurseOrdered implements the order-preserving "Recurse" mapping for
// sequence→sequence (and, used conservatively, choice→choice). Each base
// particle is consumed left-to-right: it either restricts the next derived
// particle (advancing both) or is skipped only if it is emptiable. Every derived
// particle must be consumed, and any trailing base particles must be emptiable.
func recurseOrdered(ctx context.Context, rParticles, bParticles []*Particle, schema *Schema, version Version) bool {
	// A derived particle that emits no elements (maxOccurs=0, a prohibited
	// particle) matches nothing and demands nothing of the base: it neither needs
	// a base counterpart nor blocks subsumption. Drop such particles before the
	// order-preserving mapping so they are not falsely treated as added/unmatched.
	rParticles = nonEmittingFiltered(rParticles)
	ri := 0
	for bi := range bParticles {
		bp := bParticles[bi]
		// A base particle that emits nothing (prohibited) contributes no required
		// content, so it never needs a derived counterpart — skip it.
		if particleEmitsNothing(bp) {
			continue
		}
		if ri < len(rParticles) && particleValidRestriction(ctx, rParticles[ri], bp, schema, version) {
			ri++
			continue
		}
		// This base particle is not matched by the current derived particle. It is
		// only allowed to be unmatched if it is emptiable (can occur zero times).
		if !particleEmptiable(bp) {
			return false
		}
	}
	// Every derived particle must have been consumed by some base particle.
	return ri == len(rParticles)
}

// particleEmitsNothing reports whether a particle can never emit any element
// (its maximum element-emission is zero) — i.e. a prohibited particle
// (maxOccurs=0) or a group all of whose members are themselves non-emitting. A
// non-emitting particle matches nothing and demands nothing, so it is skipped in
// restriction subsumption.
func particleEmitsNothing(p *Particle) bool {
	_, maxEmit := particleElementRange(p)
	return maxEmit == 0
}

// nonEmittingFiltered returns the subset of particles that can emit at least one
// element, dropping every non-emitting (prohibited) particle.
func nonEmittingFiltered(particles []*Particle) []*Particle {
	out := particles[:0:0]
	for _, p := range particles {
		if particleEmitsNothing(p) {
			continue
		}
		out = append(out, p)
	}
	return out
}

// recurseChoiceUnordered implements the choice→choice mapping (RecurseLax,
// XSD §3.9.6): order is NOT significant in a choice, so each derived alternative
// need only be a valid restriction of SOME base alternative, and base
// alternatives may be dropped freely (narrowing a choice to fewer branches is a
// valid restriction). A derived alternative with no matching base alternative is
// a clear violation (it admits content the base choice does not).
func recurseChoiceUnordered(ctx context.Context, rParticles, bParticles []*Particle, schema *Schema, version Version) bool {
	for _, rp := range rParticles {
		// A prohibited derived alternative emits nothing — it admits no content the
		// base must accept, so it needs no matching base alternative.
		if particleEmitsNothing(rp) {
			continue
		}
		matched := false
		for _, bp := range bParticles {
			if particleValidRestriction(ctx, rp, bp, schema, version) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

// recurseAll handles all→all: every derived particle must restrict a DISTINCT
// base particle (order is irrelevant in an all group), and every base particle
// not matched must be emptiable.
func recurseAll(ctx context.Context, rParticles, bParticles []*Particle, schema *Schema, version Version) bool {
	used := make([]bool, len(bParticles))
	for _, rp := range rParticles {
		// A prohibited derived particle emits nothing — it needs no base counterpart.
		if particleEmitsNothing(rp) {
			continue
		}
		matched := false
		for bi, bp := range bParticles {
			if used[bi] {
				continue
			}
			if particleValidRestriction(ctx, rp, bp, schema, version) {
				used[bi] = true
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	for bi, bp := range bParticles {
		if used[bi] {
			continue
		}
		// An unmatched base particle is fine if it is emptiable OR emits nothing
		// (a prohibited base particle requires no derived counterpart).
		if particleEmptiable(bp) || particleEmitsNothing(bp) {
			continue
		}
		return false
	}
	return true
}

// allRestrictsWithWildcards checks a derived all/sequence restricting a base
// xs:all when wildcards are involved (XSD 1.1). Unlike recurseAll's 1:1 mapping,
// a derived wildcard may be covered by the UNION of the base all's wildcards
// (the base all admits wildcard content collectively). Each derived ELEMENT must
// restrict a distinct base element OR be admitted by the base wildcard union;
// each derived WILDCARD must be a namespace/notQName subset of the base wildcard
// union with at-least-as-strong processContents and total cardinality within the
// base wildcards' combined range; every unmatched base element must be emptiable.
func allRestrictsWithWildcards(ctx context.Context, rParticles, bParticles []*Particle, schema *Schema, version Version) bool {
	// Flatten nested 1/1 all-group members on BOTH sides so a base all reached via
	// an xs:group ref (carrying required elements alongside a wildcard) and a
	// derived nested all are both fully accounted for, rather than appearing as an
	// opaque ModelGroup particle that would be silently skipped.
	bParticles = flattenAllParticles(bParticles)
	rParticles = flattenAllParticles(rParticles)

	var baseElems, baseWilds []*Particle
	for _, bp := range bParticles {
		switch bp.Term.(type) {
		case *ElementDecl:
			baseElems = append(baseElems, bp)
		case *Wildcard:
			baseWilds = append(baseWilds, bp)
		}
	}

	// Build the union of the base all's wildcards and its combined cardinality.
	var baseUnion *Wildcard
	baseWildMax := 0
	baseUnbounded := false
	for _, bw := range baseWilds {
		wc, _ := bw.Term.(*Wildcard)
		if baseUnion == nil {
			baseUnion = wc
		} else {
			baseUnion = wildcardUnion(baseUnion, wc, version)
		}
		if bw.MaxOccurs == Unbounded {
			baseUnbounded = true
		} else {
			baseWildMax += bw.MaxOccurs
		}
	}

	// Concrete derived elements mapped to a base element contribute, SUMMED, to
	// that base element's occurrence range (several substitution-group members, or
	// one element repeated across an ordered derived sequence, may collectively
	// restrict one base member with maxOccurs>1) — recurseAll's 1:1 mapping cannot
	// express this.
	sumMin := make([]int, len(baseElems))
	sumMax := make([]int, len(baseElems))
	var derivedWilds []*Particle
	// derivedElems holds CONCRETE derived elements not mapped to a base element but
	// admitted by a base WILDCARD. They consume from the base wildcards' capacity
	// exactly like a derived wildcard confined to their single name, so they take
	// part in the cardinality accounting below (combined totals and per-base
	// min/max) — otherwise extra concrete elements could overload a base wildcard's
	// maxOccurs (false-accept) and concrete elements would be ignored when
	// satisfying a base wildcard's minOccurs (false-reject).
	var derivedElems []*Particle
	derivedWildMax := 0
	derivedUnbounded := false
	for _, rp := range rParticles {
		if particleEmitsNothing(rp) {
			continue
		}
		switch rt := rp.Term.(type) {
		case *ElementDecl:
			// Map the derived element to a base element by name or substitution
			// group (block/abstract-aware), checking NameAndTypeOK against the actual
			// constraining member and summing its occurrence contribution.
			if bi, member := findBaseAllMember(rt, baseElems, schema); bi >= 0 {
				if !derivedElemNameAndTypeOK(ctx, rt, member, schema, version) {
					return false
				}
				sumMin[bi] = occursAdd(sumMin[bi], rp.MinOccurs)
				sumMax[bi] = occursAdd(sumMax[bi], rp.MaxOccurs)
				continue
			}
			// Not a base element — must be admitted by the base wildcard union.
			if baseUnion == nil || !wildcardAllowsName(baseUnion, rt.Name, schema) {
				return false
			}
			derivedElems = append(derivedElems, rp)
			if rp.MaxOccurs == Unbounded {
				derivedUnbounded = true
			} else {
				derivedWildMax += rp.MaxOccurs
			}
		case *Wildcard:
			if baseUnion == nil {
				return false
			}
			// The derived wildcard's processContents must be at least as strong as
			// EVERY base wildcard whose namespace it INTERSECTS — not merely the
			// weakest base wildcard in the whole union. Otherwise a skip derived
			// wildcard could restrict a strict base wildcard in the same namespace
			// just because some OTHER, disjoint base wildcard happens to be skip.
			rtCon := wildcardConstraint(rt)
			for _, bw := range baseWilds {
				bwc, _ := bw.Term.(*Wildcard)
				if !constraintsIntersect(rtCon, wildcardConstraint(bwc)) {
					continue
				}
				if processContentsStrength(rt.ProcessContents) < processContentsStrength(bwc.ProcessContents) {
					return false
				}
			}
			if !wildcardConstraintSubset(rt, baseUnion, schema, false) {
				return false
			}
			derivedWilds = append(derivedWilds, rp)
			if rp.MaxOccurs == Unbounded {
				derivedUnbounded = true
			} else {
				derivedWildMax += rp.MaxOccurs
			}
		case *ModelGroup:
			// A nested sequence/choice on the derived side (nested 1/1 all-groups
			// were already flattened away by flattenAllParticles). Its elements and
			// wildcards are NOT decomposed and accounted against the base wildcards
			// here, so silently accepting it would FALSE-ACCEPT a restriction whose
			// nested group emits content OUTSIDE the base wildcard's namespace (e.g.
			// base all{any ns="X"}, derived sequence(choice(element bad))). Fail
			// closed: reject rather than admit content the base may forbid. No W3C
			// conformance case needs the nested-non-all-group-under-wildcard
			// restriction path (the wildcard restriction cases are all:all with
			// direct wildcards; the sequence:all cases carry no wildcard), so this
			// does not regress any test.
			return false
		}
	}

	// Each base element's summed derived contribution must lie within its range; an
	// unmapped base element (sum 0) must therefore be emptiable.
	for i, bp := range baseElems {
		if !occurrenceValidRestriction(sumMin[i], sumMax[i], bp.MinOccurs, bp.MaxOccurs) {
			return false
		}
	}

	// A base element the derived side DROPS (no derived particle maps to it,
	// sumMax==0) but whose NAME a derived wildcard re-admits is an invalid
	// restriction: in the base the name is governed by the dropped element's own
	// (specific) type, while in the derived it is governed by the wildcard. A lax
	// or skip wildcard, or a strict wildcard resolving to a GLOBAL element whose
	// type does not validly restrict the dropped base element's, lets the derived
	// type ACCEPT content for that name the base type REJECTS — so the derived is
	// not a valid restriction. (wild069: base all{e:union(date,time), …} dropped to
	// all{…, any ##local lax}, with a global <e type="xs:duration"> the lax wildcard
	// admits — zang accepts <e>duration</e> that zing rejects.)
	//
	// SCOPING: only DROPPED base elements are checked. A base element the derived
	// KEEPS (sumMax>0, also admitted by the same ##local/##targetNamespace wildcard,
	// e.g. f here) is governed by its own derived declaration, not the wildcard, so
	// it must be skipped — otherwise every all-with-wildcard restriction whose
	// wildcard namespace covers a kept element would be false-rejected.
	//
	// EXEMPTION: a STRICT derived wildcard resolving to a GLOBAL element whose type
	// validly restricts the dropped base element (derivedElemNameAndTypeOK) keeps the
	// content within the base type, so it is a valid restriction.
	for i, bp := range baseElems {
		if sumMax[i] != 0 {
			continue // kept in the derived — governed by its derived declaration
		}
		if particleEmitsNothing(bp) {
			continue // prohibited base element — admits no content to lose
		}
		be, ok := bp.Term.(*ElementDecl)
		if !ok {
			continue
		}
		for _, dw := range derivedWilds {
			dwc, _ := dw.Term.(*Wildcard)
			if !wildcardAllowsName(dwc, be.Name, schema) {
				continue
			}
			if dwc.ProcessContents == ProcessStrict {
				if g, found := schema.elements[be.Name]; found && derivedElemNameAndTypeOK(ctx, g, be, schema, version) {
					continue
				}
			}
			return false
		}
	}

	// Per-base-wildcard MINIMUM cardinality: each base wildcard requires at least
	// minOccurs elements in ITS namespace. The derived content GUARANTEED to land
	// in that namespace is the sum of minOccurs of the derived wildcards whose
	// namespace is wholly contained in the base wildcard's (a derived wildcard
	// spanning a wider namespace might place its content elsewhere, so it cannot
	// be counted as guaranteed). If that guaranteed minimum is below the base
	// wildcard's minOccurs, a too-small document valid against the derived type is
	// rejected by the base type — an invalid restriction.
	for _, bw := range baseWilds {
		bwc, _ := bw.Term.(*Wildcard)
		guaranteed := 0
		for _, dw := range derivedWilds {
			dwc, _ := dw.Term.(*Wildcard)
			if wildcardConstraintSubset(dwc, bwc, schema, false) {
				guaranteed += dw.MinOccurs
			}
		}
		// A concrete derived element whose name this base wildcard admits places
		// its required occurrences in the base wildcard's namespace region.
		for _, de := range derivedElems {
			det, _ := de.Term.(*ElementDecl)
			if wildcardAllowsName(bwc, det.Name, schema) {
				guaranteed += de.MinOccurs
			}
		}
		if guaranteed < bw.MinOccurs {
			return false
		}
	}

	// Per-base-wildcard MAXIMUM cardinality: a base wildcard that EXCLUSIVELY owns
	// its namespace region (no other base wildcard's namespace intersects it) caps
	// the elements admitted in that region at its own maxOccurs. The most a derived
	// wildcard could place there is bounded by its maxOccurs whenever its namespace
	// intersects the base wildcard's (a spanning derived wildcard could direct ALL
	// its content into this region). If the derived wildcards intersecting an
	// exclusive base wildcard can collectively emit more than the base wildcard
	// allows, a too-large document valid against the derived type is rejected by
	// the base type — an invalid restriction. The exclusivity guard keeps the check
	// from false-rejecting valid restrictions of OVERLAPPING base wildcards (where
	// a region's capacity is shared across several base wildcards).
	for _, bw := range baseWilds {
		if bw.MaxOccurs == Unbounded {
			continue
		}
		bwc, _ := bw.Term.(*Wildcard)
		if !baseWildcardExclusive(bw, bwc, baseWilds) {
			continue
		}
		capacity := 0
		over := false
		for _, dw := range derivedWilds {
			dwc, _ := dw.Term.(*Wildcard)
			if !constraintsIntersect(wildcardConstraint(dwc), wildcardConstraint(bwc)) {
				continue
			}
			if dw.MaxOccurs == Unbounded {
				over = true
				break
			}
			capacity += dw.MaxOccurs
		}
		// Concrete derived elements admitted by this base wildcard also draw on
		// its capacity.
		if !over {
			for _, de := range derivedElems {
				det, _ := de.Term.(*ElementDecl)
				if !wildcardAllowsName(bwc, det.Name, schema) {
					continue
				}
				if de.MaxOccurs == Unbounded {
					over = true
					break
				}
				capacity += de.MaxOccurs
			}
		}
		if over || capacity > bw.MaxOccurs {
			return false
		}
	}

	// Combined wildcard cardinality: the derived wildcards must not admit more
	// content than the base wildcards collectively allow.
	if derivedUnbounded && !baseUnbounded {
		return false
	}
	if !baseUnbounded && derivedWildMax > baseWildMax {
		return false
	}
	return true
}

// baseWildcardExclusive reports whether base wildcard bw is the only base
// wildcard whose namespace covers its region — i.e. no OTHER base wildcard's
// namespace intersects bw's. Only then does bw alone cap its region's capacity.
func baseWildcardExclusive(bw *Particle, bwc *Wildcard, baseWilds []*Particle) bool {
	for _, other := range baseWilds {
		if other == bw {
			continue
		}
		owc, _ := other.Term.(*Wildcard)
		if constraintsIntersect(wildcardConstraint(owc), wildcardConstraint(bwc)) {
			return false
		}
	}
	return true
}

// elementRestrictsGroup implements Recurse-As-If-Group (XSD §3.9.6): a derived
// ELEMENT particle restricting a base MODEL GROUP particle. The derived element
// is treated as a singleton group and mapped through the base group's children
// according to the base compositor:
//   - sequence/all: the element must restrict EXACTLY ONE base child, and every
//     OTHER base child must be emptiable;
//   - choice: the element must restrict SOME base alternative.
//
// The element particle's occurrence range must also be a valid restriction of
// the base group particle's range. When the recursion cannot decide a sub-case
// with confidence it stays conservative (accepts) rather than risk a false
// rejection.
func elementRestrictsGroup(ctx context.Context, r *Particle, b *Particle, bg *ModelGroup, schema *Schema, version Version) bool {
	if !occurrenceValidRestriction(r.MinOccurs, r.MaxOccurs, b.MinOccurs, b.MaxOccurs) {
		return false
	}
	switch bg.Compositor {
	case CompositorChoice:
		// The element must restrict SOME alternative of the base choice.
		return slices.ContainsFunc(bg.Particles, func(bp *Particle) bool {
			return particleValidRestriction(ctx, r, bp, schema, version)
		})
	case CompositorSequence, CompositorAll:
		// The element must restrict exactly one base child; every unmatched base
		// child must be emptiable (or a prohibited base particle that emits nothing).
		matched := false
		for _, bp := range bg.Particles {
			if !matched && particleValidRestriction(ctx, r, bp, schema, version) {
				matched = true
				continue
			}
			if particleEmptiable(bp) || particleEmitsNothing(bp) {
				continue
			}
			return false
		}
		return matched
	default:
		// Unknown compositor — stay conservative.
		return true
	}
}

// groupRestrictsElement handles a derived model GROUP restricting a base single
// ELEMENT particle (the symmetric counterpart of elementRestrictsGroup). A base
// element particle accepts exactly one element (within its occurrence range);
// for the derived group to be a valid restriction, every element sequence valid
// against the group must remain valid against the base element. That holds only
// when the group is a pointless wrapper: it emits exactly one element, and that
// element validly restricts the base element. Concretely:
//   - the group's total element-emission range must be within the base element
//     particle's occurrence range (so it can never emit more — or fewer than the
//     base requires); and
//   - every element/wildcard leaf the group can emit must validly restrict the
//     base element (same name, type-derivation, etc.).
//
// A group that can emit a different element, two-or-more elements, or zero
// elements where the base requires one is rejected.
func groupRestrictsElement(ctx context.Context, r *Particle, rg *ModelGroup, b *Particle, be *ElementDecl, schema *Schema, version Version) bool {
	dMin, dMax := particleElementRange(r)
	if !occurrenceValidRestriction(dMin, dMax, b.MinOccurs, b.MaxOccurs) {
		return false
	}
	// The base element, as a singleton particle, is what each emitted leaf must
	// restrict. Build a one-occurrence base particle so the leaf check compares
	// element-to-element without re-folding the outer occurrence range.
	beP := &Particle{MinOccurs: 1, MaxOccurs: 1, Term: be}
	return groupLeavesRestrictElement(ctx, rg, beP, schema, version)
}

// groupLeavesRestrictElement reports whether every element/wildcard leaf the
// derived group rg can emit is a valid restriction of the base element particle
// beP. A wildcard leaf can match elements other than the base element's name, so
// it can never be a valid restriction of a single named element.
func groupLeavesRestrictElement(ctx context.Context, rg *ModelGroup, beP *Particle, schema *Schema, version Version) bool {
	for _, p := range rg.Particles {
		// A prohibited leaf/group emits nothing — it neither contributes an emitted
		// element nor blocks the wrapper from restricting the base element.
		if particleEmitsNothing(p) {
			continue
		}
		switch t := p.Term.(type) {
		case *ElementDecl:
			leafP := &Particle{MinOccurs: 1, MaxOccurs: 1, Term: t}
			if !particleValidRestriction(ctx, leafP, beP, schema, version) {
				return false
			}
		case *Wildcard:
			// A wildcard can match names other than the base element's — never a
			// valid restriction of a single named element.
			return false
		case *ModelGroup:
			if !groupLeavesRestrictElement(ctx, t, beP, schema, version) {
				return false
			}
		}
	}
	return true
}

// particleElementRange computes the total (min, max) number of ELEMENTS a
// particle can emit across one occurrence of its enclosing context, folding in
// the particle's own occurrence range. A maximum of -1 means unbounded. For a
// model group it combines the children per compositor (sequence/all sum the
// children; choice takes the min-of-mins and max-of-maxes) and multiplies by the
// group's own occurrence range. Used to bound derived emission against a base
// wildcard's or choice's occurrence range (cardinality subsumption).
func particleElementRange(p *Particle) (int, int) {
	mg, ok := p.Term.(*ModelGroup)
	if !ok {
		// ElementDecl or Wildcard leaf: emits exactly its own occurrence range.
		return p.MinOccurs, p.MaxOccurs
	}
	cMin, cMax := modelGroupElementRange(mg)
	return mulOccurs(p.MinOccurs, cMin), mulOccurs(p.MaxOccurs, cMax)
}

// modelGroupElementRange computes the per-occurrence (min, max) element-emission
// range of a model group's content (ignoring the group particle's own occurrence
// range, which the caller folds in). max -1 means unbounded.
func modelGroupElementRange(mg *ModelGroup) (int, int) {
	if len(mg.Particles) == 0 {
		return 0, 0
	}
	switch mg.Compositor {
	case CompositorChoice:
		// A choice emits whatever one chosen branch emits: min is the smallest
		// branch-min, max is the largest branch-max (unbounded wins).
		cMin, cMax := -1, 0
		for _, child := range mg.Particles {
			gMin, gMax := particleElementRange(child)
			if cMin == -1 || gMin < cMin {
				cMin = gMin
			}
			switch {
			case cMax == -1 || gMax == -1:
				cMax = -1
			case gMax > cMax:
				cMax = gMax
			}
		}
		if cMin == -1 {
			cMin = 0
		}
		return cMin, cMax
	default:
		// sequence/all: sum the children's ranges.
		sumMin, sumMax := 0, 0
		for _, child := range mg.Particles {
			gMin, gMax := particleElementRange(child)
			sumMin += gMin
			sumMax = maxOccursAdd(sumMax, gMax)
		}
		return sumMin, sumMax
	}
}

// maxOccursAdd adds two maximum occurrence bounds, treating -1 as unbounded
// (unbounded + anything = unbounded).
func maxOccursAdd(a, b int) int {
	if a == -1 || b == -1 {
		return -1
	}
	return a + b
}

// groupRestrictsWildcard implements NSRecurseCheckCardinality (XSD §3.9.6): a
// derived MODEL GROUP particle restricting a base WILDCARD particle. The derived
// group's EFFECTIVE occurrence range (the product of the nesting occurrence
// ranges down to each leaf) must be within the base wildcard particle's range,
// and every element/wildcard LEAF reachable inside the derived group must be
// admitted by the base wildcard's namespace constraint. When a sub-case cannot
// be decided with confidence (e.g. a nested group whose effective range is
// genuinely undecidable) it stays conservative (accepts). The base wildcard
// itself is reached through the base particle b (b.Term).
func groupRestrictsWildcard(r *Particle, rg *ModelGroup, b *Particle, schema *Schema) bool {
	// The TOTAL number of elements the derived group can emit must be within the
	// base wildcard particle's occurrence range. This is the cardinality bound that
	// matters — NOT the derived group's raw occurrence range (always 1,1 for a
	// top-level sequence), which would wrongly reject a two-element sequence
	// restricting <xs:any minOccurs="2" maxOccurs="2"> (raw 1,1 not within 2,2) even
	// though it emits exactly the two elements the base wildcard requires.
	//
	// A single <xs:any maxOccurs="1"> matches at most one element, so a derived
	// group that can emit two-or-more elements (e.g. sequence(a,b)) is still not a
	// valid restriction even though each leaf individually fits the wildcard.
	// Checking per-leaf cardinality alone (below) misses this because every leaf can
	// have maxOccurs 1 while the group's total emission exceeds the wildcard's
	// maximum.
	dMin, dMax := particleElementRange(r)
	if !occurrenceValidRestriction(dMin, dMax, b.MinOccurs, b.MaxOccurs) {
		return false
	}
	// Each leaf inside the derived group must be admitted by the wildcard, and its
	// effective occurrence maximum (folding in the enclosing group ranges) must not
	// exceed the base wildcard particle's maximum.
	return groupLeavesWithinWildcard(rg, r.MaxOccurs, b, schema)
}

// groupLeavesWithinWildcard walks the derived group's particles, threading the
// accumulated effective occurrence range (encMin/encMax = the product of the
// enclosing groups' occurrence ranges), and checks every element/wildcard leaf
// against the base wildcard particle bw: the leaf's namespace must be admitted
// and its effective occurrence MAXIMUM (enclosing × the leaf's own) must not
// exceed the base wildcard particle's maximum. Only the UPPER bound is checked
// per leaf — the lower bound is a property of the group's TOTAL emission (already
// verified by the caller against bw.MinOccurs), so a single leaf emitting fewer
// than bw.MinOccurs is not a violation when sibling leaves make up the total
// (e.g. sequence(g1,g2) each emitting 1 validly restricts <xs:any minOccurs="2">).
// Returns false on the first clear violation.
func groupLeavesWithinWildcard(rg *ModelGroup, encMax int, bw *Particle, schema *Schema) bool {
	for _, p := range rg.Particles {
		// A prohibited leaf/group emits nothing — it places no element against the
		// base wildcard, so skip it (its namespace/cardinality is irrelevant).
		if particleEmitsNothing(p) {
			continue
		}
		leafMax := mulOccurs(encMax, p.MaxOccurs)
		switch t := p.Term.(type) {
		case *ElementDecl:
			wc, ok := bw.Term.(*Wildcard)
			if !ok {
				return true
			}
			if !wildcardAllowsName(wc, t.Name, schema) {
				return false
			}
			if !occurrenceValidRestriction(0, leafMax, 0, bw.MaxOccurs) {
				return false
			}
		case *Wildcard:
			bwc, ok := bw.Term.(*Wildcard)
			if !ok {
				return true
			}
			// A derived wildcard leaf must be a namespace subset of the base
			// wildcard, with at-least-as-strong processContents and within the
			// wildcard's maximum.
			if processContentsStrength(t.ProcessContents) < processContentsStrength(bwc.ProcessContents) {
				return false
			}
			if !wildcardConstraintSubset(t, bwc, schema, false) {
				return false
			}
			if !occurrenceValidRestriction(0, leafMax, 0, bw.MaxOccurs) {
				return false
			}
		case *ModelGroup:
			if !groupLeavesWithinWildcard(t, leafMax, bw, schema) {
				return false
			}
		}
	}
	return true
}

// mulOccurs multiplies two occurrence bounds, treating -1 as unbounded
// (unbounded × anything non-zero = unbounded; anything × 0 = 0).
func mulOccurs(a, b int) int {
	if a == 0 || b == 0 {
		return 0
	}
	if a == -1 || b == -1 {
		return -1
	}
	return a * b
}

// occurrenceValidRestriction reports whether the occurrence range [rMin,rMax] is
// a valid restriction of [bMin,bMax]: rMin >= bMin and rMax <= bMax (with -1
// meaning unbounded for the maxima).
func occurrenceValidRestriction(rMin, rMax, bMin, bMax int) bool {
	if rMin < bMin {
		return false
	}
	if bMax == -1 {
		return true // base unbounded — any derived max is within range
	}
	if rMax == -1 {
		return false // derived unbounded but base bounded — widening
	}
	return rMax <= bMax
}

// occurrenceEmptiableRestriction is occurrenceValidRestriction with the XSD 1.1
// relaxation that an EMPTIABLE base group particle has an effective minOccurs of
// 0. An emptiable base already accepts the empty sequence, so a derived particle
// whose minOccurs falls below the base's declared minOccurs never admits content
// the base rejects — the derived's "zero occurrences" case is already in the base
// language. This lets a restriction narrow e.g. a base choice{1,1} whose branch
// is emptiable to a derived choice{0,1} (XSD 1.1 §3.9.6, addB118/test74966),
// which XSD 1.0's strict rMin >= bMin rejects. Gated on Version11 so 1.0 stays
// byte-identical.
func occurrenceEmptiableRestriction(r, b *Particle, version Version) bool {
	bMin := b.MinOccurs
	if version == Version11 && particleEmptiable(b) {
		bMin = 0
	}
	return occurrenceValidRestriction(r.MinOccurs, r.MaxOccurs, bMin, b.MaxOccurs)
}

// particleEmptiable reports whether a particle can be satisfied by zero
// occurrences of its content (minOccurs 0, or a group whose minimum effective
// content is empty).
func particleEmptiable(p *Particle) bool {
	if p.MinOccurs == 0 {
		return true
	}
	mg, ok := p.Term.(*ModelGroup)
	if !ok {
		return false
	}
	switch mg.Compositor {
	case CompositorChoice:
		// A choice is emptiable if ANY branch is emptiable.
		return slices.ContainsFunc(mg.Particles, particleEmptiable)
	default:
		// sequence/all: emptiable only if EVERY member is emptiable.
		for _, child := range mg.Particles {
			if !particleEmptiable(child) {
				return false
			}
		}
		return true
	}
}

// wildcardConstraintSubset reports whether the namespace constraint of sub is a
// subset of that of super, per XSD §3.10.6 ("Wildcard Subset"). It handles the
// negation constraints (##any/##other/##not-absent) that the union-oriented
// wildcardNSSubset helper folds into empty sets, so a derived ##other is not
// wrongly treated as a subset of a finite set. The model used:
//   - ##any           = the universe (matches every namespace incl. absent)
//   - ##other         = universe minus {targetNS, absent}
//   - ##not-absent    = universe minus {absent}
//   - finite set      = the listed URIs (##local→absent, ##targetNamespace→TNS)
//
// sub ⊆ super when every namespace sub admits is also admitted by super.
// schema/isAttr are used only by the XSD 1.1 path for ##defined-aware per-name
// subset checks (isAttr selects the elements vs attributes declaration table).
func wildcardConstraintSubset(sub, super *Wildcard, schema *Schema, isAttr bool) bool {
	// XSD 1.1: when either wildcard carries a notNamespace/notQName constraint
	// the 1.0 case analysis below cannot represent it; decide via the general
	// constraint algebra. 1.0-only wildcards keep the byte-identical path below.
	if wildcardHas11Fields(sub) || wildcardHas11Fields(super) {
		return wildcardConstraintSubset11(sub, super, schema, isAttr)
	}
	// super = ##any admits everything.
	if super.Namespace == WildcardNSAny {
		return true
	}
	// super is not ##any, so sub = ##any cannot be a subset.
	if sub.Namespace == WildcardNSAny {
		return false
	}

	subNeg := sub.Namespace == WildcardNSOther || sub.Namespace == WildcardNSNotAbsent
	superNeg := super.Namespace == WildcardNSOther || super.Namespace == WildcardNSNotAbsent

	// Both negations: sub ⊆ super iff super's excluded set ⊆ sub's excluded set
	// (excluding more shrinks the admitted set). ##not-absent excludes only
	// {absent}; ##other excludes {absent, targetNS}.
	if subNeg && superNeg {
		superExcl := wildcardExcludedSet(super)
		subExcl := wildcardExcludedSet(sub)
		for ns := range superExcl {
			if !subExcl[ns] {
				return false
			}
		}
		return true
	}

	// sub is a negation but super is a finite set: a negation admits infinitely
	// many namespaces, a finite set admits finitely many, so it cannot be a
	// subset.
	if subNeg {
		return false
	}

	// sub is a finite set, super is a negation: every URI sub admits must be
	// admitted by super, i.e. not in super's excluded set.
	if superNeg {
		superExcl := wildcardExcludedSet(super)
		for ns := range wildcardNSSet(sub) {
			if superExcl[ns] {
				return false
			}
		}
		return true
	}

	// Both finite sets: ordinary subset.
	superSet := wildcardNSSet(super)
	for ns := range wildcardNSSet(sub) {
		if !superSet[ns] {
			return false
		}
	}
	return true
}

// wildcardExcludedSet returns the namespaces a negation wildcard
// (##other/##not-absent) does NOT admit. The empty namespace is represented by
// "".
func wildcardExcludedSet(wc *Wildcard) map[string]bool {
	switch wc.Namespace {
	case WildcardNSNotAbsent:
		return map[string]bool{"": true}
	case WildcardNSOther:
		return map[string]bool{"": true, wc.TargetNS: true}
	default:
		return map[string]bool{}
	}
}

// wildcardAllowsName reports whether the wildcard admits an ELEMENT with the
// given expanded name. Used for the element-restricts-wildcard (NSCompat) case.
// It delegates to the validator's wildcardAllowsExpandedName so the restriction
// check and instance validation share ONE definition of which names a wildcard
// admits: the namespace constraint (incl. notNamespace, and ##other excluding
// the ABSENT namespace) AND the XSD 1.1 @notQName disallowed-name set
// (explicit QNames, ##definedSibling, and ##defined — the latter consulting
// schema.elements). Without the notQName half the restriction check would
// false-ACCEPT a derived element a base wildcard explicitly excludes.
func wildcardAllowsName(wc *Wildcard, name QName, schema *Schema) bool {
	return wildcardAllowsExpandedName(wc, name.Local, name.NS, schema, false)
}
