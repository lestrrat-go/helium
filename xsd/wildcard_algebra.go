package xsd

import (
	"slices"
	"sort"
	"strings"
)

// wildcard_algebra.go implements the XSD 1.1 namespace-constraint algebra
// (union and intersection) for wildcards that may carry @notNamespace and
// @notQName. It normalizes a wildcard's namespace constraint to a single
// internal form (wcConstraint) on which set union/intersection is
// straightforward, then materializes the result back onto a *Wildcard.
//
// The XSD 1.0 union path in link_refs.go (wildcardUnion) is preserved untouched
// for wildcards with NO 1.1 fields, so 1.0 behavior is byte-identical; this file
// is only reached for wildcards carrying notNamespace/notQName or via the
// Version11-gated attribute-wildcard intersection.

// wcConstraint is a normalized wildcard namespace constraint: either a positive
// finite set of admitted namespaces (neg=false) or a negation that admits every
// namespace EXCEPT the listed ones (neg=true). "" denotes the absent namespace.
// neg=true with an empty set is ##any.
type wcConstraint struct {
	neg bool
	set map[string]struct{}
}

// wildcardConstraint normalizes a wildcard's namespace constraint.
func wildcardConstraint(wc *Wildcard) wcConstraint {
	if wc.NotNamespace != nil {
		return wcConstraint{neg: true, set: sliceToSet(wc.NotNamespace)}
	}
	switch wc.Namespace {
	case WildcardNSAny:
		return wcConstraint{neg: true, set: map[string]struct{}{}}
	case WildcardNSOther:
		return wcConstraint{neg: true, set: map[string]struct{}{"": {}, wc.TargetNS: {}}}
	case WildcardNSNotAbsent:
		return wcConstraint{neg: true, set: map[string]struct{}{"": {}}}
	default:
		s := map[string]struct{}{}
		for ns := range wildcardNSSet(wc) {
			s[ns] = struct{}{}
		}
		return wcConstraint{neg: false, set: s}
	}
}

// wildcardMatchesNothing reports whether a wildcard's namespace constraint admits
// NO namespace at all — a positive (non-negated) constraint with an empty set,
// e.g. `namespace=""`. Such a wildcard matches no expanded name, so its language
// is empty. (A negated constraint always admits infinitely many namespaces, and a
// non-empty positive set admits its members, so both are emitting.)
func wildcardMatchesNothing(wc *Wildcard) bool {
	c := wildcardConstraint(wc)
	return !c.neg && len(c.set) == 0
}

func sliceToSet(in []string) map[string]struct{} {
	s := make(map[string]struct{}, len(in))
	for _, v := range in {
		s[v] = struct{}{}
	}
	return s
}

func constraintUnion(a, b wcConstraint) wcConstraint {
	switch {
	case !a.neg && !b.neg:
		return wcConstraint{neg: false, set: setUnion(a.set, b.set)}
	case a.neg && b.neg:
		// not(A) ∪ not(B) admits everything except what BOTH exclude.
		return wcConstraint{neg: true, set: setIntersect(a.set, b.set)}
	default:
		// One positive, one negation: result is a negation excluding the
		// negation's excluded set minus the namespaces the positive set admits.
		neg, pos := a, b
		if !a.neg {
			neg, pos = b, a
		}
		return wcConstraint{neg: true, set: setDifference(neg.set, pos.set)}
	}
}

func constraintIntersect(a, b wcConstraint) wcConstraint {
	switch {
	case !a.neg && !b.neg:
		return wcConstraint{neg: false, set: setIntersect(a.set, b.set)}
	case a.neg && b.neg:
		// not(A) ∩ not(B) excludes A ∪ B.
		return wcConstraint{neg: true, set: setUnion(a.set, b.set)}
	default:
		// One positive, one negation: result is the positive set minus the
		// namespaces the negation excludes.
		neg, pos := a, b
		if !a.neg {
			neg, pos = b, a
		}
		return wcConstraint{neg: false, set: setDifference(pos.set, neg.set)}
	}
}

func setUnion(a, b map[string]struct{}) map[string]struct{} {
	out := make(map[string]struct{}, len(a)+len(b))
	for k := range a {
		out[k] = struct{}{}
	}
	for k := range b {
		out[k] = struct{}{}
	}
	return out
}

func setIntersect(a, b map[string]struct{}) map[string]struct{} {
	out := map[string]struct{}{}
	for k := range a {
		if _, ok := b[k]; ok {
			out[k] = struct{}{}
		}
	}
	return out
}

func setDifference(a, b map[string]struct{}) map[string]struct{} {
	out := map[string]struct{}{}
	for k := range a {
		if _, ok := b[k]; !ok {
			out[k] = struct{}{}
		}
	}
	return out
}

// constraintsIntersect reports whether two normalized namespace constraints
// admit at least one common namespace (used by the UPA wildcard-overlap test).
func constraintsIntersect(a, b wcConstraint) bool {
	switch {
	case !a.neg && !b.neg:
		for ns := range a.set {
			if _, ok := b.set[ns]; ok {
				return true
			}
		}
		return false
	case a.neg && b.neg:
		// not(A) and not(B) both admit the (infinite) universe minus two finite
		// sets, which always share members.
		return true
	default:
		neg, pos := a, b
		if !a.neg {
			neg, pos = b, a
		}
		for ns := range pos.set {
			if _, ok := neg.set[ns]; !ok {
				return true
			}
		}
		// A purely positive empty set admits nothing; a non-empty positive set
		// fully covered by the negation's excluded set also has no overlap.
		// The absent/##any case (pos.set empty, neg admits all) is handled when
		// pos has members; an empty pos means it matches nothing → no overlap.
		return false
	}
}

// constraintToWildcard materializes a normalized constraint onto a fresh
// *Wildcard, carrying the supplied processContents/targetNS, notQName
// disallowed-name set, and resolved ##definedSibling names. siblingNames MUST be
// carried through union/intersection alongside the NotQNameDefinedSibling marker
// — dropping it would let a materialized wildcard re-admit sibling-named children
// the operands excluded.
func constraintToWildcard(con wcConstraint, pc ProcessContentsKind, tns string, notQName []QName, defined, definedSibling bool, siblingNames []QName) *Wildcard {
	wc := &Wildcard{ProcessContents: pc, TargetNS: tns, NotQName: notQName, NotQNameDefined: defined, NotQNameDefinedSibling: definedSibling, SiblingNames: siblingNames}
	if con.neg {
		// Negation: a non-nil NotNamespace signals "match all except set".
		// An empty set means ##any.
		excl := make([]string, 0, len(con.set))
		for ns := range con.set {
			excl = append(excl, ns)
		}
		sort.Strings(excl)
		wc.NotNamespace = excl
		wc.Namespace = WildcardNSAny
		return wc
	}
	// Positive finite set rendered as a namespace list (##local for absent).
	parts := make([]string, 0, len(con.set))
	for ns := range con.set {
		if ns == "" {
			parts = append(parts, WildcardNSLocal)
			continue
		}
		parts = append(parts, ns)
	}
	sort.Strings(parts)
	if len(parts) == 0 {
		// Empty positive set matches nothing; represent as an empty namespace
		// list (degenerate, preserved by readWildcard semantics).
		wc.Namespace = ""
		return wc
	}
	wc.Namespace = strings.Join(parts, " ")
	return wc
}

// notQNameUnion returns the union of two QName sets (used for wildcard
// INTERSECTION: the intersection excludes names excluded by either).
func notQNameUnion(a, b []QName) []QName {
	out := append([]QName(nil), a...)
	for _, qb := range b {
		if !slices.Contains(a, qb) {
			out = append(out, qb)
		}
	}
	return out
}

// intersectWildcards computes the XSD 1.1 attribute-wildcard INTERSECTION of two
// wildcards: a name is admitted iff BOTH admit it. The namespace constraints are
// intersected; the notQName disallowed-name sets are unioned (excluded by
// either); ##defined applies if either has it; and the ##definedSibling
// exclusions are UNIONED (a sibling name either operand excludes is excluded by
// the intersection). The result's processContents is the STRONGER of the two
// (strict > lax > skip) so the intersection is order-independent — a strict
// operand must not be weakened to skip just because it was listed second.
func intersectWildcards(a, b *Wildcard) *Wildcard {
	con := constraintIntersect(wildcardConstraint(a), wildcardConstraint(b))
	return constraintToWildcard(con, strongerProcessContents(a.ProcessContents, b.ProcessContents), a.TargetNS,
		notQNameUnion(a.NotQName, b.NotQName),
		a.NotQNameDefined || b.NotQNameDefined,
		a.NotQNameDefinedSibling || b.NotQNameDefinedSibling,
		notQNameUnion(a.SiblingNames, b.SiblingNames))
}

// strongerProcessContents returns whichever processContents value enforces more
// validation (strict > lax > skip).
func strongerProcessContents(a, b ProcessContentsKind) ProcessContentsKind {
	if processContentsStrength(a) >= processContentsStrength(b) {
		return a
	}
	return b
}

// unionWildcards11 computes the namespace-constraint UNION honoring 1.1 fields
// (XSD 1.1 §3.10.6.3, cos-aw-union). The namespace constraints are unioned
// directly. For the {disallowed names}, a candidate QName (drawn from the
// operands' explicit notQName lists) is in the union iff NEITHER operand admits
// it via its NAMESPACE constraint AND explicit notQName list. Crucially, the
// per-QName test IGNORES ##defined: per the spec, ##defined is folded only as a
// whole (the union keeps ##defined iff BOTH operands carry it), and it does NOT
// make an individual QName "disallowed" for the union. So a name one operand
// excludes only via ##defined, but admits namespace-wise, is still admitted by
// the union even if the other operand excludes it explicitly — see W3C wild083
// (`surprise` is allowed because the ##defined operand admits it by namespace).
// ##definedSibling is folded the same way (kept iff both carry it).
//
// processContents: the union's {process contents} is the SECOND operand's (b's).
// The sole caller where this matters is TYPE EXTENSION, which passes
// wildcardUnion(base, derived) — and per XSD 3.4.2 the extension's effective
// (complete) attribute wildcard takes the DERIVED wildcard's processContents, NOT
// the base's. (This is direction-specific and differs from intersectWildcards,
// which takes the STRONGER of the two; that case is order-independent.)
func unionWildcards11(a, b *Wildcard) *Wildcard {
	con := constraintUnion(wildcardConstraint(a), wildcardConstraint(b))
	var disallowed []QName
	for _, qn := range notQNameUnion(a.NotQName, b.NotQName) {
		if wildcardAdmitsNameIgnoringDefined(a, qn.Local, qn.NS) {
			continue
		}
		if wildcardAdmitsNameIgnoringDefined(b, qn.Local, qn.NS) {
			continue
		}
		disallowed = append(disallowed, qn)
	}
	// ##definedSibling names: the union excludes a sibling name only if NEITHER
	// operand admits it (same rule as the explicit notQName candidates). When the
	// folded marker survives (both operands carry ##definedSibling), these
	// retained names back the subset/exclusion checks; when it does not, the
	// candidates are admitted by one side and so drop out here.
	var siblings []QName
	for _, qn := range notQNameUnion(a.SiblingNames, b.SiblingNames) {
		if wildcardAdmitsNameIgnoringDefined(a, qn.Local, qn.NS) {
			continue
		}
		if wildcardAdmitsNameIgnoringDefined(b, qn.Local, qn.NS) {
			continue
		}
		siblings = append(siblings, qn)
	}
	return constraintToWildcard(con, b.ProcessContents, a.TargetNS, disallowed,
		a.NotQNameDefined && b.NotQNameDefined,
		a.NotQNameDefinedSibling && b.NotQNameDefinedSibling,
		siblings)
}

// wildcardAdmitsNameIgnoringDefined reports whether the wildcard admits a name by
// its NAMESPACE constraint and explicit @notQName/##definedSibling exclusions,
// DELIBERATELY ignoring ##defined. This is the per-QName admission test the
// cos-aw-union {disallowed names} rule uses (##defined is folded separately).
func wildcardAdmitsNameIgnoringDefined(wc *Wildcard, local, ns string) bool {
	return wildcardMatches(wc, ns) && !wildcardExcludesName(wc, local, ns)
}

// wildcardConstraintSubset11 reports whether sub's namespace constraint is a
// subset of super's, honoring XSD 1.1 notNamespace/notQName (cos-ns-subset).
// sub ⊆ super iff every namespace sub admits is also admitted by super, AND
// every name super disallows is also disallowed by sub (sub may not re-admit a
// name super excludes). The per-name subset test is the FULL "allows expanded
// name" test (`wildcardAllowsExpandedName`), so a derived wildcard may discharge
// a base's explicit `notQName="t:g"` via its own `##defined` when t:g has a
// global declaration. schema/isAttr select the declaration table for ##defined.
func wildcardConstraintSubset11(sub, super *Wildcard, schema *Schema, isAttr bool) bool {
	subC := wildcardConstraint(sub)
	supC := wildcardConstraint(super)

	switch {
	case !supC.neg && subC.neg:
		// super admits a finite set, sub admits infinitely many → not a subset.
		return false
	case !supC.neg && !subC.neg:
		// Both finite: sub.set ⊆ super.set.
		for ns := range subC.set {
			if _, ok := supC.set[ns]; !ok {
				return false
			}
		}
	case supC.neg && !subC.neg:
		// super admits all except E_super; sub's admitted set must avoid E_super.
		for ns := range subC.set {
			if _, ok := supC.set[ns]; ok {
				return false
			}
		}
	case supC.neg && subC.neg:
		// Both negations: sub ⊆ super iff super excludes a subset of what sub
		// excludes (super admits more, so each super-excluded ns must also be
		// sub-excluded).
		for ns := range supC.set {
			if _, ok := subC.set[ns]; !ok {
				return false
			}
		}
	}

	// notQName: a name disallowed by super must also be disallowed by sub, else
	// the restriction re-admits it. The test is the FULL expanded-name admission
	// (namespace + explicit notQName + ##defined + ##definedSibling) so sub can
	// discharge a super-excluded global name via its own ##defined.
	for _, qn := range super.NotQName {
		if wildcardAllowsExpandedName(sub, qn.Local, qn.NS, schema, isAttr) {
			return false
		}
	}
	// ##defined as a whole: if super excludes every globally-declared name, sub
	// must too (an individual ##defined-excluded name need not be listed, so this
	// marker-level check complements the per-name loop above).
	if super.NotQNameDefined && !sub.NotQNameDefined {
		return false
	}
	// ##definedSibling: every concrete sibling name super excludes (its resolved
	// SiblingNames) must also be disallowed by sub. This runs REGARDLESS of the
	// NotQNameDefinedSibling marker, because a materialized union can carry a
	// finite SiblingNames set with the marker dropped (the marker survives only
	// when BOTH operands had it) — those names still exclude matches, so sub must
	// honor them or the restriction silently re-admits them.
	for _, qn := range super.SiblingNames {
		if wildcardAllowsExpandedName(sub, qn.Local, qn.NS, schema, isAttr) {
			return false
		}
	}
	// A LIVE ##definedSibling marker excludes the (content-model-derived) sibling
	// set open-endedly; require sub to also carry the marker so a derived wildcard
	// in a different content model cannot re-admit siblings the base would compute
	// there.
	if super.NotQNameDefinedSibling && !sub.NotQNameDefinedSibling {
		return false
	}
	return true
}

// wildcardHas11Fields reports whether a wildcard carries any XSD 1.1 negated
// namespace/name constraint. Used to route to the 1.1-aware algebra without
// disturbing the byte-identical 1.0 union path. Resolved SiblingNames count as
// 1.1 state even when NotQNameDefinedSibling is false — a materialized union can
// retain concrete ##definedSibling exclusions without the live marker, and those
// names still exclude matches, so subset/algebra checks must not skip them.
func wildcardHas11Fields(wc *Wildcard) bool {
	return wc.NotNamespace != nil || len(wc.NotQName) > 0 || wc.NotQNameDefined ||
		wc.NotQNameDefinedSibling || len(wc.SiblingNames) > 0
}
