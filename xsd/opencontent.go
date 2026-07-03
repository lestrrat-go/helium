package xsd

import (
	"context"
	"fmt"
	"slices"

	helium "github.com/lestrrat-go/helium"
)

// resolveOpenContent computes the EFFECTIVE {open content} of every complex type
// (XSD 1.1 §3.4.2.1/§3.4.2.2), folding in the per-document schema-level
// <xs:defaultOpenContent>, then inheriting/merging across extension derivations
// and checking restriction-derivation validity. It runs after the content models
// and content types are finalized (extension merges done) and is gated to 1.1, so
// XSD 1.0 is byte-identical. Types are processed BASE-FIRST so a derived type sees
// its base's already-resolved open content.
func (c *compiler) resolveOpenContent(ctx context.Context) {
	if c.version != Version11 {
		return
	}
	resolved := make(map[*TypeDef]struct{})
	var resolve func(td *TypeDef)
	resolve = func(td *TypeDef) {
		if td == nil {
			return
		}
		if _, ok := resolved[td]; ok {
			return
		}
		resolved[td] = struct{}{}
		if td.BaseType != nil {
			resolve(td.BaseType)
		}
		c.computeEffectiveOpenContent(ctx, td)
	}
	for _, td := range c.schema.types {
		resolve(td)
	}
	for td := range c.typeDefSources {
		resolve(td)
	}
}

// materializeOpenContentSiblings populates SiblingNames on an open-content wildcard
// carrying @notQName="##definedSibling" — the element names declared in the type's
// OWN content model (the same set resolveDefinedSiblings uses for content-model
// wildcards) — returning a CLONE so a shared *Wildcard (a per-document
// <xs:defaultOpenContent>, or an explicit wildcard reused elsewhere) is not mutated.
// The runtime exclusion (wildcardExcludesName) checks SiblingNames, not the marker;
// crucially this runs on the LOCAL effective open content BEFORE the extension union
// (cos-aw-union, wildcardUnion) so the union carries the materialized SiblingNames as
// finite exclusions even when the live ##definedSibling marker is folded away (a
// one-sided union where the other operand excludes the sibling by notQName/namespace
// drops the marker but must retain the sibling). The base operand is already
// materialized because resolveOpenContent processes BASE-FIRST.
func (c *compiler) materializeOpenContentSiblings(oc *OpenContent, td *TypeDef) *OpenContent {
	if oc == nil || oc.Wildcard == nil || !oc.Wildcard.NotQNameDefinedSibling {
		return oc
	}
	names := collectModelElementNames(td.ContentModel, c.schema)
	siblings := make([]QName, 0, len(names))
	for qn := range names {
		siblings = append(siblings, qn)
	}
	wc := *oc.Wildcard
	wc.SiblingNames = siblings
	return &OpenContent{Mode: oc.Mode, Wildcard: &wc, AppliesToEmpty: oc.AppliesToEmpty}
}

// computeEffectiveOpenContent resolves a single complex type's {open content}:
// the explicit <xs:openContent> (mode="none" → absent) or, absent that, the
// per-document <xs:defaultOpenContent> (applied unless the effective content type
// is empty and appliesToEmpty is false). For an EXTENSION the result is then
// merged with the base's open content (§3.4.2.2: a base interleave mode wins, the
// wildcards union; an extension may not turn a base interleave into suffix); for a
// RESTRICTION its validity against the base open content is checked.
func (c *compiler) computeEffectiveOpenContent(ctx context.Context, td *TypeDef) {
	if td == nil || !td.IsComplex || td.ContentType == ContentTypeSimple {
		return
	}

	// Effective (locally specified, default-folded) open content.
	var eff *OpenContent
	if td.openContentExplicit {
		eff = td.OpenContent // nil for mode="none"
	} else if td.pendingDefaultOpenContent != nil {
		def := td.pendingDefaultOpenContent
		if def.AppliesToEmpty || !contentTypeEmptyForOpenContent(td) {
			eff = &OpenContent{Mode: def.Mode, Wildcard: def.Wildcard}
		}
	}

	// Materialize ##definedSibling SiblingNames on the local effective open content
	// BEFORE any extension union or restriction check, so the union/check carries the
	// finite sibling exclusions (the base operand is already materialized base-first).
	eff = c.materializeOpenContentSiblings(eff, td)

	if td.Derivation == DerivationExtension && td.BaseType != nil {
		baseOC := td.BaseType.OpenContent
		switch {
		case eff == nil:
			td.OpenContent = baseOC // inherit base (§3.4.2.2 4.1)
		case baseOC == nil:
			td.OpenContent = eff // §3.4.2.2 4.2
		default:
			// §3.4.6.2 1.4.3.2.2.2: an extension may not relax a base 'interleave'
			// open content to 'suffix'.
			if baseOC.Mode == OpenContentInterleave && eff.Mode == OpenContentSuffix {
				c.reportOpenContentTypeError(ctx, td,
					"The open content mode 'suffix' is not a valid extension of base open content mode 'interleave'.")
			}
			mode := eff.Mode
			if baseOC.Mode == OpenContentInterleave {
				mode = OpenContentInterleave
			}
			td.OpenContent = &OpenContent{Mode: mode, Wildcard: wildcardUnion(baseOC.Wildcard, eff.Wildcard, c.version)}
		}
		return
	}

	td.OpenContent = eff
	if td.Derivation == DerivationRestriction && td.BaseType != nil {
		c.checkOpenContentRestriction(ctx, td, eff, td.BaseType.OpenContent)
		c.checkOpenContentDropsBaseLocal(ctx, td, eff)
		c.checkOpenContentDropsBaseWildcard(ctx, td, eff)
		c.checkDerivedWildcardReadmitsBaseOpen(ctx, td)
	}
}

// checkDerivedWildcardReadmitsBaseOpen guards QUADRANT B of the restriction
// content-interaction matrix (§3.4.6.4): a restriction may DROP the base's open
// content O but then re-introduce O's admitted language as a DECLARED WILDCARD in the
// derived content model, bypassing every open-content guard. A declared wildcard wins
// attribution, so a child it admits is validated by IT (processContents P_Wd), not by
// O (P_O). For the derived to be a subset of the base, every derived declared wildcard
// W_d that admits children in O's namespace must be a valid RESTRICTION of O. When O is
// INTERLEAVE (no ordering) that means namespace ⊆ O's AND processContents at least as
// strong as P_O. When O is SUFFIX it ALSO imposes a trailing-ORDERING constraint that a
// declared wildcard (non-trailing declared content) cannot preserve, so re-admitting
// O's namespace as a declared wildcard is rejected outright (any namespace/pc), mirroring
// the quadrant-A suffix order-loss handling. Version11 / restriction only; a base WITHOUT
// open content is unaffected.
//
// Scoped to avoid a FALSE-REJECT: a derived declared wildcard that VALIDLY RESTRICTS a
// BASE DECLARED wildcard (W_d ⊆ W_b and pc at least as strong) is a QUADRANT-D
// restriction handled by restriction_particle.go, so it is exempt here; and a derived
// declared wildcard that is itself a valid restriction of O (⊆ namespace, ≥ pc) passes.
func (c *compiler) checkDerivedWildcardReadmitsBaseOpen(ctx context.Context, td *TypeDef) {
	if c.version != Version11 || td.BaseType == nil {
		return
	}
	base := td.BaseType.OpenContent
	if base == nil || base.Wildcard == nil {
		return // quadrant B requires the BASE to carry open content
	}
	baseStrength := processContentsStrength(base.Wildcard.ProcessContents)
	openCon := wildcardConstraint(base.Wildcard)
	baseWCs := collectEmittingDeclaredWildcards(td.BaseType.ContentModel)
	for _, dw := range collectEmittingDeclaredWildcards(td.ContentModel) {
		if !constraintsIntersect(wildcardConstraint(dw.wc), openCon) {
			continue // the derived declared wildcard does not touch the base open content's namespace
		}
		// Exempt a derived declared wildcard that validly RESTRICTS a base DECLARED
		// wildcard — that interaction is quadrant D (restriction_particle.go).
		exempt := false
		for _, bw := range baseWCs {
			if wildcardConstraintSubset(dw.wc, bw.wc, c.schema, false) &&
				processContentsStrength(dw.wc.ProcessContents) >= processContentsStrength(bw.wc.ProcessContents) {
				exempt = true
				break
			}
		}
		if exempt {
			continue
		}
		// The derived declared wildcard re-admits the base OPEN content's language.
		if base.Mode == OpenContentSuffix {
			// SUFFIX imposes an ORDERING constraint — open children must be TRAILING (the
			// runtime rejects a trailing declared-name child / matches the wildcard only in
			// the suffix). Re-admitting the namespace as a DECLARED wildcard makes those
			// children DECLARED content, which can appear NON-trailing, so the derived
			// accepts orderings the base rejects → reject. BUT the order is only lost if
			// the derived declared wildcard ACTUALLY admits a child in the intersecting
			// region: skip/lax always admit, but a STRICT wildcard with NO matching global
			// element declaration there admits NOTHING (it rejects the child), so no order
			// is lost → EXEMPT (mirrors the quadrant-A strict/no-global exemption).
			if dw.wc.ProcessContents == ProcessStrict && !c.strictOpenAdmitsGlobalIn(dw.wc, base.Wildcard) {
				continue
			}
			c.reportOpenContentTypeError(ctx, td,
				"The restriction drops the base type's suffix open content but re-admits its language through a declared wildcard, losing the trailing-ordering constraint (the re-admitted children may appear non-trailing, which the base suffix open content rejects).")
			return
		}
		// INTERLEAVE imposes no ordering: the declared wildcard must be a valid restriction
		// of O — namespace ⊆ O AND processContents at least as strong as P_O.
		if !wildcardConstraintSubset(dw.wc, base.Wildcard, c.schema, false) ||
			processContentsStrength(dw.wc.ProcessContents) < baseStrength {
			c.reportOpenContentTypeError(ctx, td,
				"The restriction drops the base type's open content but re-admits its language through a declared wildcard whose namespace or processContents is not a valid restriction of the base open-content wildcard.")
			return
		}
	}
}

// declaredWildcardInfo pairs a DECLARED content-model wildcard term with its
// effective maximum occurrence (the product of maxOccurs along the path from the
// content-model root, folding ancestor groups via mulOccurs).
type declaredWildcardInfo struct {
	wc     *Wildcard
	effMax int
}

// collectEmittingDeclaredWildcards returns every EMITTING (effective maxOccurs > 0)
// declared wildcard particle in a content model, recursing through nested groups,
// each with its effective maximum occurrence.
func collectEmittingDeclaredWildcards(mg *ModelGroup) []declaredWildcardInfo {
	var out []declaredWildcardInfo
	var walk func(g *ModelGroup, ancestorMax int)
	walk = func(g *ModelGroup, ancestorMax int) {
		if g == nil {
			return
		}
		gMax := mulOccurs(ancestorMax, g.MaxOccurs)
		for _, p := range g.Particles {
			switch term := p.Term.(type) {
			case *Wildcard:
				if pMax := mulOccurs(gMax, p.MaxOccurs); pMax != 0 {
					out = append(out, declaredWildcardInfo{wc: term, effMax: pMax})
				}
			case *ModelGroup:
				walk(term, gMax)
			}
		}
	}
	walk(mg, 1)
	return out
}

// checkOpenContentDropsBaseWildcard is the wildcard-particle analogue of
// checkOpenContentDropsBaseLocal (§3.4.6.4 soundness): a base content model may
// declare an xs:any particle that WINS ATTRIBUTION over the open content (interleave
// seeds a child matching a declared wildcard into the declared sub-sequence; suffix
// matches it in the prefix and a leftover declared-wildcard match — NOT flagged
// misplaced, since that check only catches declared ELEMENT names — still spills to
// open content). Dropping or NARROWING that declared wildcard and re-admitting its
// namespace through the derived open content with weaker enforcement (or losing the
// suffix ordering) lets the derived accept content the base validated more strictly.
//
// For each emitting declared base wildcard bw whose namespace the derived OPEN
// wildcard ACTUALLY re-admits (constraintsIntersect — finding 3: an open content
// that excludes bw's namespace cannot receive its excess, so it is exempt), and
// where the open content does NOT enforce at least as strictly (interleave: open pc
// weaker than bw's; suffix: ANY pc — a suffix base imposes an ordering the derived
// loses), check OCCURRENCE COVERAGE with AGGREGATE capacity (finding 2): the base's
// total declared-wildcard capacity over bw's namespace region (every overlapping
// base wildcard at least as strong as bw, summed) must be covered by the derived's
// total declared-wildcard capacity that fully supersets bw with adequate pc (summed);
// else the uncovered excess spills into the weaker open content → reject. The
// accounting OVER-estimates base capacity (overlap, not subset) and UNDER-estimates
// derived capacity (subset, not overlap), so it is fail-closed toward soundness.
// (UPA forbids two overlapping declared wildcards in one content model, so the
// aggregate reduces to single-wildcard occurrence coverage for constructible
// schemas; the sums are kept for defensive correctness.)
func (c *compiler) checkOpenContentDropsBaseWildcard(ctx context.Context, td *TypeDef, derived *OpenContent) {
	if derived == nil || derived.Wildcard == nil || td.BaseType == nil || td.BaseType.ContentModel == nil {
		return
	}
	pc := derived.Wildcard.ProcessContents
	baseSuffix := td.BaseType.OpenContent != nil && td.BaseType.OpenContent.Mode == OpenContentSuffix
	openCon := wildcardConstraint(derived.Wildcard)
	baseWCs := collectEmittingDeclaredWildcards(td.BaseType.ContentModel)
	derivedWCs := collectEmittingDeclaredWildcards(td.ContentModel)
	for _, bw := range baseWCs {
		bwStrength := processContentsStrength(bw.wc.ProcessContents)
		bwCon := wildcardConstraint(bw.wc)
		if !constraintsIntersect(openCon, bwCon) {
			continue // the derived open content does not re-admit bw's namespace
		}
		// The derived open content actually ACCEPTS something in bw's region only if a
		// STRICT open wildcard can resolve it: strict admits ONLY a child with a matching
		// GLOBAL element declaration, so if no global element's name is admitted by BOTH
		// the open wildcard AND bw, a strict open accepts NOTHING there (no spill) → a
		// valid (more restrictive) drop/narrow. (skip/lax always accept.)
		if pc == ProcessStrict && !c.strictOpenAdmitsGlobalIn(derived.Wildcard, bw.wc) {
			continue
		}
		// Interleave is safe when the open content enforces at least as strictly as bw
		// (no type loss, interleave imposes no ordering). Suffix loses the ordering for
		// ANY pc, so it is always checked.
		if !baseSuffix && processContentsStrength(pc) >= bwStrength {
			continue
		}
		// AGGREGATE occurrence coverage over the SPILL-RELEVANT REGION = bw's namespace
		// ∩ the derived open-content wildcard's namespace. Only names the open content
		// ADMITS can leave the declared model and spill into open content; names bw
		// admits but the open content excludes can never spill, so the derived needn't
		// cover them. baseCap sums base wildcards that INTERSECT the spill region
		// (over-estimate); derivedCap sums derived wildcards that SUPERSET the spill
		// region (under-estimate) — fail-closed toward soundness.
		spillCon := constraintIntersect(bwCon, openCon)
		spillWC := constraintToWildcard(spillCon, bw.wc.ProcessContents, bw.wc.TargetNS, nil, false, false, nil)
		baseCap := 0
		for _, bw2 := range baseWCs {
			if processContentsStrength(bw2.wc.ProcessContents) >= bwStrength &&
				constraintsIntersect(wildcardConstraint(bw2.wc), spillCon) {
				baseCap = maxOccursAdd(baseCap, bw2.effMax)
			}
		}
		derivedCap := 0
		for _, dw := range derivedWCs {
			if processContentsStrength(dw.wc.ProcessContents) >= bwStrength &&
				wildcardConstraintSubset(spillWC, dw.wc, c.schema, false) {
				derivedCap = maxOccursAdd(derivedCap, dw.effMax)
			}
		}
		if occursCovers(derivedCap, baseCap) {
			continue // the derived declared wildcards cover the base's spill-region capacity
		}
		if baseSuffix {
			c.reportOpenContentTypeError(ctx, td,
				"The restriction drops or narrows a base declared wildcard whose children the base's suffix open content requires in the prefix region; the derived re-admits the excess as trailing open content, losing the ordering constraint.")
			return
		}
		c.reportOpenContentTypeError(ctx, td,
			"The restriction drops or narrows a base declared wildcard but re-admits its namespace through an open-content wildcard whose processContents is weaker, so the excess children escape the base wildcard's validation.")
		return
	}
}

// strictOpenAdmitsGlobalIn reports whether a STRICT open-content wildcard could
// actually accept any element in base wildcard bw's namespace region: a strict
// wildcard admits ONLY a child with a matching GLOBAL element declaration, so it can
// spill into open content only if some global element's expanded name is admitted by
// BOTH the open wildcard and bw. With none, the strict open accepts nothing there.
func (c *compiler) strictOpenAdmitsGlobalIn(open, bw *Wildcard) bool {
	for qn := range c.schema.elements {
		if wildcardAllowsExpandedName(open, qn.Local, qn.NS, c.schema, false) &&
			wildcardAllowsExpandedName(bw, qn.Local, qn.NS, c.schema, false) {
			return true
		}
	}
	return false
}

// checkOpenContentDropsBaseLocal rejects a restriction that DROPS a base element
// declaration and re-admits a name the base ADMITTED there — its own name (LOCAL
// declaration or ref-to-global) OR any INSTANCE-ADMISSIBLE substitution-group
// member of it — through a LENIENT open-content wildcard that does not enforce the
// base element's declared type, so the derived would accept content the base
// rejects (not a valid subset). Runtime matching (elemMatchesDeclOrSubst) admits a
// base element particle's concrete transitive substitution members, so a derived
// wildcard that re-admits a member m (e.g. excluding the head h via notQName="h"
// while still admitting m) loses m's type just as it would the head's. The open
// content fails to enforce the type when its wildcard is:
//
//   - processContents="skip" — validateWildcardChild returns BEFORE any global
//     lookup, so the element is NEVER assessed; the base type is lost regardless of
//     whether the base declared the name as a LOCAL, a REF-to-global, or admitted
//     it as a substitution member; OR
//   - processContents="lax" with NO global declaration of that name — nothing to
//     assess against (only arises for a base LOCAL, since a ref/substitution member
//     implies a global exists).
//
// There are two unsoundness sources. TYPE loss: a DROPPED name re-admitted by an
// unenforcing wildcard (processContents="skip", which is never assessed, or "lax"
// with NO global declaration) escapes the base element's type. A "strict" wildcard,
// or "lax" WITH a global declaration, resolves a governing type whose consistency
// the dynamic EDC (validateWildcardElementConsistent) enforces, so it is type-safe.
// ORDER loss: a SUFFIX-mode BASE requires a declared name to appear in the prefix
// region (a leftover declared name is rejected as misplaced); dropping that name
// makes the derived accept it as trailing open content REGARDLESS of processContents
// — the dynamic EDC enforces only the TYPE, not the ORDER — so a dropped name a
// suffix base declares is rejected for ANY processContents. (An interleave base
// imposes no ordering.) A name the derived KEEPS as a declared element is matched by
// the model via weak-wildcard precedence; in INTERLEAVE mode the EXCESS beyond the
// derived particle's maxOccurs spills into open content, so a kept name needs
// OCCURRENCE COVERAGE when the wildcard is unenforcing; SUFFIX keeps a misplaced
// excess child rejected, so a kept name is always safe there. A name the derived
// wildcard EXCLUDES (notQName/##defined) is never re-admitted; a NON-EMITTING base
// name (effective base maxOccurs 0) is admitted by the base only via open content,
// not the element — both exempt. xsi:type is instance-driven (dynamic EDC), not a
// compile-time name source. Substitution: the protected set is every name the base
// ADMITS at runtime (baseAdmissibleElementNames mirrors elemMatchesDeclOrSubst).
func (c *compiler) checkOpenContentDropsBaseLocal(ctx context.Context, td *TypeDef, derived *OpenContent) {
	if derived == nil || derived.Wildcard == nil || td.BaseType == nil || td.BaseType.ContentModel == nil {
		return
	}
	pc := derived.Wildcard.ProcessContents
	baseSuffix := td.BaseType.OpenContent != nil && td.BaseType.OpenContent.Mode == OpenContentSuffix
	derivedInterleave := derived.Mode == OpenContentInterleave
	// EMITTING names only: a derived particle the restriction narrows to maxOccurs=0
	// (prohibited) can no longer match its name, so the name is effectively DROPPED
	// (it spills to open content) and must be treated as dropped, not kept.
	derivedNames := collectEmittingModelElementNames(td.ContentModel, c.schema)
	seen := make(map[QName]struct{})
	for _, bn := range baseAdmissibleElementNames(td.BaseType.ContentModel, c.schema) {
		if _, dup := seen[bn]; dup {
			continue
		}
		seen[bn] = struct{}{}
		// The base must actually EMIT bn through an element particle: a name whose
		// effective base maxOccurs is 0 (its particle, or any ancestor group, is
		// maxOccurs="0" — prohibited/non-emitting) is admitted by the base ONLY via
		// its open content, NOT the element, so dropping it in the derived while
		// keeping the same open content is a valid restriction — do not protect it.
		baseMax := maxOccursForName(td.BaseType.ContentModel, bn, c.schema)
		if baseMax == 0 {
			continue
		}
		if !wildcardAllowsExpandedName(derived.Wildcard, bn.Local, bn.NS, c.schema, false) {
			continue // the open wildcard does not re-admit this name
		}
		// Whether the derived open content ENFORCES bn's declared type: strict always
		// resolves a governing type; lax resolves only when a global declaration exists;
		// skip never assesses.
		enforced := pc == ProcessStrict
		if pc == ProcessLax {
			if _, ok := c.schema.elements[bn]; ok {
				enforced = true
			}
		}
		if !derivedNames[bn] {
			// The derived FULLY DROPS bn. The guard only fires if the derived open
			// content would ACTUALLY ACCEPT bn: skip accepts all (validates nothing);
			// lax accepts (validating only if a global exists); but a STRICT wildcard
			// accepts bn ONLY with a matching GLOBAL declaration — strict-WITHOUT-global
			// REJECTS bn, so the derived accepts FEWER documents than the base (which
			// admitted it via the local declaration): a valid (more restrictive) drop.
			if pc == ProcessStrict {
				if _, ok := c.schema.elements[bn]; !ok {
					continue
				}
			}
			if baseSuffix {
				// ORDER loss: the suffix base requires bn in the prefix region; the
				// derived accepts it as trailing open content even when the type is
				// enforced. Reject for any processContents.
				c.reportOpenContentTypeError(ctx, td,
					"The restriction drops the base type's element declaration '"+bn.Local+
						"' which the base's suffix open content requires in the prefix region; the derived re-admits it as trailing open content, losing the ordering constraint.")
				return
			}
			if !enforced {
				// TYPE loss: an interleave base re-admits bn through an unenforcing
				// wildcard (skip, or lax with no global), so its base type is lost.
				c.reportOpenContentTypeError(ctx, td,
					"The restriction drops the base type's element declaration '"+bn.Local+
						"' but re-admits it through an open-content wildcard that does not enforce its declared type.")
				return
			}
			// CONSTRAINT loss: the dynamic EDC enforces only governing-TYPE
			// substitutability, so the other declaration properties the base LOCAL imposes
			// — {type table}, `fixed`, `nillable`, identity constraints, `block`, and an
			// asymmetric `default` — are LOST when the name is re-admitted via the GLOBAL.
			// Reject (fail-closed) unless the global is at least as restrictive on each.
			// (A base REF or substitution member resolves to the SAME global in base and
			// derived, so localElementDeclsByName — which excludes refs — finds only the
			// true LOCAL decls that can diverge.)
			if global, ok := c.schema.elements[bn]; ok {
				for _, baseLocal := range localElementDeclsByName(td.BaseType.ContentModel, bn) {
					if msg := c.globalDropsLocalConstraint(ctx, baseLocal, global); msg != "" {
						c.reportOpenContentTypeError(ctx, td,
							"The restriction drops the base type's local element declaration '"+bn.Local+
								"' and re-admits it through an open-content wildcard governed by the global declaration, which "+msg+".")
						return
					}
				}
			}
			continue
		}
		// The derived KEEPS bn. In SUFFIX mode a trailing declared-name child is
		// rejected as misplaced (never spilled), so a kept name is safe regardless of
		// occurrence. In INTERLEAVE mode the EXCESS bn children (beyond the derived
		// particle's maxOccurs) spill into open content, so a derived that NARROWS bn's
		// maxOccurs below the base's effective max is only safe if the spill stays within
		// the base's language.
		if derivedInterleave {
			derivedMax := maxOccursForName(td.ContentModel, bn, c.schema)
			if !occursCovers(derivedMax, baseMax) {
				if !enforced {
					// UNENFORCING wildcard: the excess loses the declared TYPE entirely.
					c.reportOpenContentTypeError(ctx, td,
						"The restriction narrows the maxOccurs of the kept element '"+bn.Local+
							"' below the base's while an interleave open-content wildcard re-admits it without enforcing its declared type; the excess children would escape the base type.")
					return
				}
				// ENFORCING wildcard (strict / lax-with-global): the spilled excess IS
				// type-checked, but via the GLOBAL declaration, LOSING the base local's
				// fixed/nillable/identity constraints (same hole as the dropped-local case).
				// Reject when the global is not constraint-compatible with the base local.
				// (A strict wildcard WITHOUT a global rejects the spilled child — no global
				// here means no spill, so it is safe and the loop simply does not run.)
				if global, ok := c.schema.elements[bn]; ok {
					for _, baseLocal := range localElementDeclsByName(td.BaseType.ContentModel, bn) {
						if msg := c.globalDropsLocalConstraint(ctx, baseLocal, global); msg != "" {
							c.reportOpenContentTypeError(ctx, td,
								"The restriction narrows the maxOccurs of the kept element '"+bn.Local+
									"' below the base's; the excess spills into the enforcing interleave open content governed by the global declaration, which "+msg+".")
							return
						}
					}
				}
			}
		}
	}
}

// globalDropsLocalConstraint reports (with a diagnostic phrase) when a GLOBAL element
// declaration is NOT at least as restrictive as a base LOCAL declaration of the same
// name on the SIX properties the dynamic wildcard-EDC does NOT enforce — {type table},
// `fixed`, `nillable`, identity constraints, {disallowed substitutions} (`block`), and
// the asymmetric `default` direction — so re-admitting the name via the global would
// accept content the base local rejected. Returns "" when the global is
// constraint-compatible. Fail-closed: any constraint that cannot be shown at-least-as-
// restrictive is treated as lost. Used by BOTH the dropped-local and kept-narrowed call
// sites (a kept local that narrows maxOccurs spills excess to the global identically).
//
//   - type table: a base local CTA {type table} NOT EQUIVALENT to the global's
//     (typeTablesConsistent, absent-both = equivalent) lets the global attribute the
//     spilled/re-admitted element to a different governing type than the base would —
//     the dynamic EDC checks only type-DEFINITION substitutability, not table equivalence.
//   - fixed: a base local `fixed` value the global does not carry (no fixed) or carries
//     with a DIFFERENT VALUE lets the global admit values the local forbade. The
//     comparison is VALUE-SPACE (fixedValueMatches, accounting for each declaration's
//     FixedNS namespace context) — so a QName/NOTATION fixed with the same lexical form
//     but a different prefix→namespace binding is DIFFERENT (reject), and a numeric/list
//     fixed differing only lexically (e.g. "1.0" vs "1.00") is EQUAL.
//   - nillable: a global that is nillable while the local is NOT would accept
//     xsi:nil="true" the base rejected.
//   - identity constraints: a base local xs:key/xs:unique/xs:keyref the global does not
//     also impose (matched by resolved QName) is lost.
//   - block / {disallowed substitutions}: validateWildcardChild applies the GLOBAL's
//     block, so any restriction/extension/substitution the LOCAL blocks but the global
//     does not is lost — e.g. a base local block="#all" rejecting an xsi:type
//     derivation the global admits. The EFFECTIVE blocked set on each side is the
//     UNION of the element declaration's {disallowed substitutions} and its effective
//     declared TYPE's {prohibited substitutions} (cvc-elt.4.3), so a block carried by
//     the base-local's TYPE (not the element) is preserved too.
//   - default (ASYMMETRIC): a BASE-LOCAL default is NOT a constraint (it only supplies
//     a value for an empty element; it forbids nothing) — so it is not compared. But a
//     GLOBAL default the local LACKS (or a VALUE-SPACE-DIFFERENT one, DefaultNS-aware) IS
//     unsound: an empty <e/> re-admitted via the global gets the global's default
//     substituted, making an otherwise type-invalid empty element valid.
func (c *compiler) globalDropsLocalConstraint(ctx context.Context, local, global *ElementDecl) string {
	if local == nil || global == nil {
		return ""
	}
	if !typeTablesConsistent(local, global) {
		return "has a conditional-type-assignment {type table} that differs from the base declaration's"
	}
	if local.Fixed != nil {
		if global.Fixed == nil || !fixedValueMatches(ctx, *local.Fixed, *global.Fixed, local.Type, local.FixedNS, global.FixedNS, c.schema, c.version) {
			return "does not impose the base declaration's fixed value"
		}
	}
	if global.Nillable && !local.Nillable {
		return "is nillable while the base declaration is not, so it would accept xsi:nil the base rejected"
	}
	// Compare ONLY the derivation bits (extension/restriction). Wildcard assessment
	// of a re-admitted child does not use substitution-group matching, and cvc-elt.4.3
	// derivation blocking ignores the substitution bit — substitution-group blocking
	// is handled by the name-admission / substitution-closure logic, not this same-name
	// global-constraint check.
	const derivBits = BlockExtension | BlockRestriction
	localBlocked := (local.Block | effectiveDeclType(local, c.schema).prohibitedSubstitutions()) & derivBits
	globalBlocked := (global.Block | effectiveDeclType(global, c.schema).prohibitedSubstitutions()) & derivBits
	if localBlocked&^globalBlocked != 0 {
		return "does not block every derivation the base declaration's 'block' forbade"
	}
	if global.Default != nil && (local.Default == nil || !fixedValueMatches(ctx, *local.Default, *global.Default, local.Type, local.DefaultNS, global.DefaultNS, c.schema, c.version)) {
		return "supplies a 'default' the base declaration does not, so it would accept an empty element the base rejected"
	}
	for _, lc := range local.IDCs {
		if lc == nil {
			continue
		}
		found := false
		for _, gc := range global.IDCs {
			if gc != nil && gc.Kind == lc.Kind && gc.QName == lc.QName {
				found = true
				break
			}
		}
		if !found {
			return "does not impose the base declaration's identity constraint '" + lc.Name + "'"
		}
	}
	return ""
}

// occursCovers reports whether a derived maximum-occurrence bound covers a base
// bound: an unbounded base requires an unbounded derived; an unbounded derived
// covers any base; otherwise the derived count must be at least the base's.
func occursCovers(derivedMax, baseMax int) bool {
	if baseMax == Unbounded {
		return derivedMax == Unbounded
	}
	if derivedMax == Unbounded {
		return true
	}
	return derivedMax >= baseMax
}

// maxOccursForName returns the maximum number of element children named n (matched
// DIRECTLY or as a substitution-group member of a particle's declaration, mirroring
// elemMatchesDeclOrSubst) that the content model can match as DECLARED content, with
// Unbounded for an unbounded maximum. A sequence/all sums its members; a choice
// takes the max member; each group multiplies by its own maxOccurs.
func maxOccursForName(mg *ModelGroup, n QName, schema *Schema) int {
	if mg == nil {
		return 0
	}
	inner := 0
	for _, p := range mg.Particles {
		cnt := particleMaxOccursForName(p, n, schema)
		if mg.Compositor == CompositorChoice {
			inner = maxOccursOf(inner, cnt)
		} else {
			inner = maxOccursAdd(inner, cnt)
		}
	}
	return mulOccurs(inner, mg.MaxOccurs)
}

func particleMaxOccursForName(p *Particle, n QName, schema *Schema) int {
	switch term := p.Term.(type) {
	case *ElementDecl:
		if elementDeclAdmitsName(term, n, schema) {
			return p.MaxOccurs
		}
		return 0
	case *ModelGroup:
		// A nested group particle's occurrence equals the inner group's own MaxOccurs
		// (copied at parse), so recurse without re-folding p's occurrence.
		return maxOccursForName(term, n, schema)
	}
	return 0
}

// elementDeclAdmitsName reports whether an element particle's declaration admits a
// child named n — its own name when CONCRETE, or any of its instance-admissible
// (abstract-excluded) substitution-group members — matching elemMatchesDeclOrSubst.
func elementDeclAdmitsName(term *ElementDecl, n QName, schema *Schema) bool {
	if !term.Abstract && term.Name == n {
		return true
	}
	for _, m := range instanceSubstMembers(term, schema) {
		if m.Name == n {
			return true
		}
	}
	return false
}

// maxOccursOf returns the larger of two maximum-occurrence bounds, treating
// Unbounded (-1) as the maximum.
func maxOccursOf(a, b int) int {
	if a == Unbounded || b == Unbounded {
		return Unbounded
	}
	if a > b {
		return a
	}
	return b
}

// baseAdmissibleElementNames returns every element name a content model ADMITS at
// runtime: for each element particle (a LOCAL declaration or a ref-to-global,
// resolved by resolveRefs to carry the global's Type/Abstract/Block) its OWN name
// when CONCRETE, plus its INSTANCE-ADMISSIBLE (abstract-excluded) transitive
// substitution-group members — exactly the set elemMatchesDeclOrSubst matches.
// Recurses through nested model groups. An abstract head's own name is excluded (no
// instance bears it), so a derived wildcard admitting only that name is not flagged.
func baseAdmissibleElementNames(mg *ModelGroup, schema *Schema) []QName {
	var names []QName
	var walk func(g *ModelGroup)
	walk = func(g *ModelGroup) {
		if g == nil {
			return
		}
		for _, p := range g.Particles {
			switch term := p.Term.(type) {
			case *ElementDecl:
				if !term.Abstract {
					names = append(names, term.Name)
				}
				for _, m := range instanceSubstMembers(term, schema) {
					names = append(names, m.Name)
				}
			case *ModelGroup:
				walk(term)
			}
		}
	}
	walk(mg)
	return names
}

// checkOpenContentRestriction enforces §3.4.6.4: a restriction's {open content}
// must be a valid restriction of the base's. A restriction may DROP open content
// (derived absent) but may not ADD it (base absent, derived present); the derived
// wildcard must be a subset of the base's; the derived processContents must be at
// least as strong; and the derived mode may differ from the base only when the
// base is interleave. When the derived content model is EMPTY (it matches only the
// empty sequence) ONLY the declared-model MODE comparison is moot and skipped — the
// base-absent / wildcard-subset / processContents-strength checks still apply, so an
// empty-model restriction may not ADD open content to a base that never admitted
// those children, nor BROADEN or WEAKEN the base's open content.
func (c *compiler) checkOpenContentRestriction(ctx context.Context, td *TypeDef, derived, base *OpenContent) {
	if derived == nil {
		return // dropping open content is always a valid restriction
	}
	emptyModel := !modelGroupHasContent(td.ContentModel)
	if base == nil {
		// The base has no open content. An EMPTY-model restriction may still
		// introduce open content when the base's DECLARED content model already
		// admits those children through a content-model wildcard — that is, the
		// restriction merely re-expresses the base's `xs:any` particle as open
		// content (saxonData Open/open022). It is NOT a valid restriction when the
		// base is genuinely closed (no admitting wildcard), nor for a non-empty
		// derived model, which the original §3.4.6.4 enforcement always rejected.
		if emptyModel && baseModelAdmitsOpenContent(td.BaseType, derived.Wildcard, c.schema) {
			return
		}
		c.reportOpenContentTypeError(ctx, td,
			"The derived type has open content but its base type does not.")
		return
	}
	// The MODE comparison is meaningful only when the derived type has a declared
	// content model: for an EMPTY content model the open content IS the type's whole
	// content and its mode against the base is immaterial. The wildcard-subset and
	// processContents checks below are NOT waived in either case.
	if !emptyModel && base.Mode != OpenContentInterleave && derived.Mode != base.Mode {
		c.reportOpenContentTypeError(ctx, td,
			"The open content mode 'interleave' is not a valid restriction of base open content mode 'suffix'.")
		return
	}
	if !wildcardConstraintSubset(derived.Wildcard, base.Wildcard, c.schema, false) {
		c.reportOpenContentTypeError(ctx, td,
			"The open content wildcard is not a valid restriction of the base type's open content wildcard.")
		return
	}
	if processContentsStrength(derived.Wildcard.ProcessContents) < processContentsStrength(base.Wildcard.ProcessContents) {
		c.reportOpenContentTypeError(ctx, td,
			"The open content wildcard's processContents is weaker than the base type's open content wildcard.")
	}
}

// baseModelAdmitsOpenContent reports whether a restriction may validly re-express
// the base type's DECLARED content model as the (effectively unbounded) derived
// open-content wildcard, even though the base carries no {open content} of its own
// (saxonData Open/open022). The full content-model restriction-subsumption +
// weak-wildcard-attribution problem is out of scope; rather than approximate it,
// this accepts ONLY the single PROVABLY-SOUND shape and rejects everything else
// (fail-closed):
//
//	The base content model contains EXACTLY ONE wildcard particle W and NO
//	element-declaration particles anywhere; W's OWN occurrence is minOccurs=0,
//	maxOccurs=unbounded; and every ANCESTOR group on W's path is at-most-once
//	(maxOccurs=1). Then the base content language is EXACTLY W* — every count in
//	[0, infinity) — so the derived open content (which accepts every count) is a
//	language subset exactly when its wildcard is a namespace SUBSET of W
//	(wildcardConstraintSubset) with processContents at least as strong.
//
// Why no element declarations: a child whose name collides with a base element
// declaration is ATTRIBUTED to that element under the base (weak-wildcard
// attribution) and validated by its — possibly stricter — type, while the derived
// open content would validate it only as open content; the derived would then NOT
// be a subset of the base. A wildcard-only base has no element to win attribution.
// Why W's OWN occurrence 0..unbounded AND every ancestor at-most-once: a min/max
// occurrence PRODUCT of 0..unbounded is NOT sufficient — e.g.
// `sequence(0,unbounded){ any(minOccurs=2) }` has product 0..unbounded but accepts
// counts {0} union [2, infinity), a gap at 1 the derived (every count) would
// exploit. Requiring W itself to be 0..unbounded and all ancestor groups
// at-most-once makes W the sole source of repetition/optionality, so the language
// is exactly W* with no count gaps. A second wildcard, a required/bounded wildcard,
// or any repeating ancestor group is conservatively deferred. open022 (a single
// optional unbounded wildcard, no elements) is accepted; the attribution repro, the
// round-4/round-5 repros, and the shared `(G, G)*` group-ref case (two wildcards)
// are all rejected.
func baseModelAdmitsOpenContent(base *TypeDef, derived *Wildcard, schema *Schema) bool {
	if base == nil || derived == nil || base.ContentModel == nil {
		return false
	}
	var scan baseModelWildcardScan
	scan.walk(base.ContentModel, true, true)
	if scan.elementSeen || scan.wildcardCount != 1 || scan.wildcard == nil {
		return false
	}
	// The wildcard's OWN occurrence must be exactly optional-and-unbounded, and every
	// ancestor group at-most-once, so the base language is EXACTLY W* (every count in
	// [0, infinity)). A min/max occurrence PRODUCT of 0..unbounded is NOT sufficient:
	// e.g. `sequence(0,unbounded){ any(minOccurs=2) }` has product 0..unbounded but
	// accepts counts {0} union [2, infinity) — a gap at 1 that the derived open
	// content (which accepts every count) would exploit, breaking the subset relation.
	if !scan.pathAtMostOnce || scan.wildcardMin != 0 || scan.wildcardMax != Unbounded {
		return false
	}
	return wildcardConstraintSubset(derived, scan.wildcard, schema, false) &&
		processContentsStrength(derived.ProcessContents) >= processContentsStrength(scan.wildcard.ProcessContents)
}

// baseModelWildcardScan accumulates a structural scan of a base content model for
// baseModelAdmitsOpenContent: whether any element-declaration particle was seen,
// the number of wildcard particles, and (for the last/only wildcard) its OWN
// minOccurs/maxOccurs plus whether every ANCESTOR group on its path is at-most-once
// (maxOccurs == 1). When the single wildcard's own occurrence is 0..unbounded and
// every ancestor group is at-most-once, the base content language is exactly W*.
type baseModelWildcardScan struct {
	elementSeen    bool
	wildcardCount  int
	wildcard       *Wildcard
	wildcardMin    int
	wildcardMax    int
	pathAtMostOnce bool
}

// walk descends model group mg. ancestorsAtMostOnce is true iff every group from
// the root down to (but excluding) mg has maxOccurs == 1; ancestorsEmitting is true
// iff no ancestor group has maxOccurs == 0 (prohibited/non-emitting). A NON-EMITTING
// particle (its own maxOccurs == 0, or under a maxOccurs == 0 ancestor) emits
// nothing, so it contributes neither a disqualifying element nor a counted wildcard:
// a base equivalent to W* plus a prohibited element is still effectively
// wildcard-only. A nested group particle's occurrence equals the inner group's own
// MinOccurs/MaxOccurs (copied at parse time), so walk evaluates each group's
// occurrence once via the recursive call's mg and reads the wrapping particle only
// for LEAF occurrence.
func (s *baseModelWildcardScan) walk(mg *ModelGroup, ancestorsAtMostOnce, ancestorsEmitting bool) {
	if mg == nil {
		return
	}
	groupAtMostOnce := ancestorsAtMostOnce && mg.MaxOccurs == 1
	groupEmitting := ancestorsEmitting && mg.MaxOccurs != 0
	for _, p := range mg.Particles {
		emitting := groupEmitting && p.MaxOccurs != 0
		switch term := p.Term.(type) {
		case *ElementDecl:
			if emitting {
				s.elementSeen = true
			}
		case *Wildcard:
			if !emitting {
				continue // a prohibited wildcard emits nothing
			}
			s.wildcardCount++
			s.wildcard = term
			s.wildcardMin = p.MinOccurs
			s.wildcardMax = p.MaxOccurs
			s.pathAtMostOnce = groupAtMostOnce
		case *ModelGroup:
			s.walk(term, groupAtMostOnce, groupEmitting)
		}
	}
}

// reportOpenContentTypeError emits a complex-type-level schema error for an
// open-content derivation violation, using the type's recorded source location.
func (c *compiler) reportOpenContentTypeError(ctx context.Context, td *TypeDef, msg string) {
	src, ok := c.typeDefSources[td]
	if !ok || c.filename == "" {
		return
	}
	component := componentLocalComplexType
	if !src.isLocal {
		component = "complex type '" + td.Name.Local + "'"
	}
	c.schemaError(ctx, schemaComponentError(c.diagSourceOrRecorded(src.source), src.line, "complexType", component, msg))
}

// contentTypeEmptyForOpenContent reports whether a complex type's effective
// content type is empty for the purpose of <xs:defaultOpenContent>/@appliesToEmpty
// (§3.4.2.1): a mixed or simple-content type is never empty; otherwise the type is
// empty iff its content model carries no element/wildcard content.
func contentTypeEmptyForOpenContent(td *TypeDef) bool {
	if td.ContentType == ContentTypeMixed || td.ContentType == ContentTypeSimple {
		return false
	}
	return !modelGroupHasContent(td.ContentModel)
}

// parseOpenContent reads an XSD 1.1 <xs:openContent> element. mode defaults to
// "interleave"; "suffix" restricts open elements to a trailing position; "none"
// disables open content and returns nil. The wildcard is taken from the child
// <xs:any>. Callers must only invoke this in XSD 1.1 mode.
func (c *compiler) parseOpenContent(ctx context.Context, elem *helium.Element) *OpenContent {
	mode := OpenContentInterleave
	isNone := false
	// @mode is an enumeration over xs:token (whiteSpace="collapse"), so the value must
	// be whitespace-collapsed before the enum comparison — e.g. mode=" suffix " is valid.
	switch normalizeWhiteSpace(getAttr(elem, attrMode), "collapse") {
	case "", "interleave":
		mode = OpenContentInterleave
	case "suffix":
		mode = OpenContentSuffix
	case "none":
		// Explicitly no open content (used to override a default open content).
		isNone = true
	default:
		if c.filename != "" {
			c.schemaError(ctx, schemaParserErrorAttr(c.diagSource(), elem.Line(), elem.LocalName(), elemOpenContent, attrMode,
				"The value of 'mode' must be one of 'interleave', 'suffix', or 'none'."))
		}
	}

	scan := scanOpenContentChildren(elem)
	c.reportOpenContentChildGrammar(ctx, elem, elemOpenContent, "An", "openContent", scan)
	anyElem := scan.anyElem

	if isNone {
		// mode="none" must NOT carry an <xs:any> wildcard child (bug 7069).
		if anyElem != nil && c.filename != "" {
			c.schemaError(ctx, schemaParserError(c.diagSource(), elem.Line(), elem.LocalName(), elemOpenContent,
				"An 'openContent' with mode 'none' must not contain an 'any' wildcard."))
		}
		return nil
	}

	if anyElem == nil {
		// An xs:openContent with mode != none requires an xs:any wildcard.
		if c.filename != "" {
			c.schemaError(ctx, schemaParserError(c.diagSource(), elem.Line(), elem.LocalName(), elemOpenContent,
				"An 'openContent' with mode other than 'none' must contain an 'any' wildcard."))
		}
		return nil
	}
	wc := c.parseOpenContentWildcard(ctx, anyElem)
	if wc == nil {
		return nil
	}
	return &OpenContent{Mode: mode, Wildcard: wc}
}

// openContentOrderViolation returns the schema-error message when an
// <xs:openContent> child appears out of order within a complex type's child
// sequence, or "" when it is correctly placed. XSD §3.4.2 fixes the order
// (annotation?, (openContent?, (group|all|choice|sequence)?),
// ((attribute|attributeGroup)*, anyAttribute?), assert*), so an openContent must
// precede the content-model particle, the attribute uses, the anyAttribute
// wildcard, AND the trailing xs:assert region.
func openContentOrderViolation(contentModelChild, directAttrChild string, anyAttributeSeen, assertSeen bool) string {
	switch {
	case contentModelChild != "":
		return fmt.Sprintf("The 'openContent' must appear before the content model particle '%s'.", contentModelChild)
	case directAttrChild != "":
		return fmt.Sprintf("The 'openContent' must appear before the attribute declaration '%s'.", directAttrChild)
	case anyAttributeSeen:
		return "The 'openContent' must appear before the attribute wildcard 'anyAttribute'."
	case assertSeen:
		return "The 'openContent' must appear before the assertion 'assert'."
	}
	return ""
}

// openContentChildScan records the result of scanning the children of an
// <xs:openContent>/<xs:defaultOpenContent>, whose content model is strictly
// (annotation?, any?): the first <xs:any> wildcard, the annotation/any counts, the
// first stray (non-annotation, non-any) child, and whether an annotation appeared
// AFTER the any (out of order). Callers turn the latter three into schema errors.
type openContentChildScan struct {
	anyElem       *helium.Element
	annotations   int
	anyCount      int
	stray         *helium.Element
	annotAfterAny bool
}

// scanOpenContentChildren walks the children of an <xs:openContent> or
// <xs:defaultOpenContent> element and reports its child grammar. Non-element nodes
// (whitespace/comment/PI) are ignored; every element child that is neither
// <xs:annotation> nor <xs:any> is recorded as the first stray.
func scanOpenContentChildren(elem *helium.Element) openContentChildScan {
	var r openContentChildScan
	for child := range helium.Children(elem) {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		switch {
		case isXSDElement(ce, elemAnnotation):
			r.annotations++
			if r.anyCount > 0 {
				r.annotAfterAny = true
			}
		case isXSDElement(ce, elemAny):
			r.anyCount++
			if r.anyElem == nil {
				r.anyElem = ce
			}
		default:
			if r.stray == nil {
				r.stray = ce
			}
		}
	}
	return r
}

// reportOpenContentChildGrammar emits the schema errors for an out-of-grammar
// child of an <xs:openContent>/<xs:defaultOpenContent>: more than one annotation
// or wildcard, an annotation after the wildcard, or a stray child. article+noun
// name the element in diagnostics ("An"/"openContent", "A"/"defaultOpenContent").
func (c *compiler) reportOpenContentChildGrammar(ctx context.Context, elem *helium.Element, comp, article, noun string, scan openContentChildScan) {
	if c.filename == "" {
		return
	}
	if scan.annotations > 1 {
		c.schemaError(ctx, schemaParserError(c.diagSource(), elem.Line(), elem.LocalName(), comp,
			article+" '"+noun+"' must not have more than one 'annotation'."))
	}
	if scan.anyCount > 1 {
		c.schemaError(ctx, schemaParserError(c.diagSource(), elem.Line(), elem.LocalName(), comp,
			article+" '"+noun+"' must not have more than one 'any' wildcard."))
	}
	if scan.annotAfterAny {
		c.schemaError(ctx, schemaParserError(c.diagSource(), elem.Line(), elem.LocalName(), comp,
			"In "+article+" '"+noun+"' the 'annotation' must precede the 'any' wildcard."))
	}
	if scan.stray != nil {
		c.schemaError(ctx, schemaParserError(c.diagSource(), scan.stray.Line(), scan.stray.LocalName(), comp,
			article+" '"+noun+"' may contain only an optional 'annotation' followed by an optional 'any' wildcard; '"+scan.stray.LocalName()+"' is not allowed."))
	}
}

// parseOpenContentWildcard parses the <xs:any> child of an open-content element.
// Unlike a content-model wildcard, an open-content <xs:any> must NOT carry
// minOccurs/maxOccurs (bug 15618): occurrence is governed by the open-content
// mechanism, not the wildcard particle.
func (c *compiler) parseOpenContentWildcard(ctx context.Context, anyElem *helium.Element) *Wildcard {
	if c.filename != "" {
		for _, attr := range []string{attrMinOccurs, attrMaxOccurs} {
			if hasAttr(anyElem, attr) {
				c.schemaError(ctx, schemaParserErrorAttr(c.diagSource(), anyElem.Line(), anyElem.LocalName(), elemAny, attr,
					"The attribute '"+attr+"' is not allowed on the 'any' wildcard of an open content."))
			}
		}
	}
	return c.readWildcard(ctx, anyElem)
}

// readDefaultOpenContent reads the schema-level <xs:defaultOpenContent> child of
// a schema root (XSD 1.1), if present, returning the resulting default open
// content (nil when absent or invalid). It enforces the schema content-model
// position constraint: <xs:defaultOpenContent> may appear only after the leading
// composition (include/import/redefine/override) and annotation children and
// before any schema-level component declaration; at most one is allowed. mode
// defaults to "interleave" and may also be "suffix" ("none" is not a valid
// default-open-content mode); appliesToEmpty defaults to false.
func (c *compiler) readDefaultOpenContent(ctx context.Context, root *helium.Element) *OpenContent {
	if c.version != Version11 {
		return nil
	}
	var dec *helium.Element
	sawDeclaration := false
	sawDefault := false
	for child := range helium.Children(root) {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		if isXSDElement(ce, elemDefaultOpenContent) {
			if sawDefault && c.filename != "" {
				c.schemaError(ctx, schemaParserError(c.diagSource(), ce.Line(), ce.LocalName(), elemDefaultOpenContent,
					"A schema must not have more than one 'defaultOpenContent'."))
				continue
			}
			sawDefault = true
			if sawDeclaration && c.filename != "" {
				c.schemaError(ctx, schemaParserError(c.diagSource(), ce.Line(), ce.LocalName(), elemDefaultOpenContent,
					"The 'defaultOpenContent' must appear before any schema component declaration."))
			}
			dec = ce
			continue
		}
		switch {
		case isXSDElement(ce, elemInclude), isXSDElement(ce, elemImport),
			isXSDElement(ce, elemRedefine), isXSDElement(ce, elemOverride):
			// Composition elements must precede defaultOpenContent: the schema
			// content model is ((include|import|redefine|override|annotation)*,
			// (defaultOpenContent, annotation*)?, ...), so a composition element
			// AFTER the defaultOpenContent is out of order.
			if sawDefault && c.filename != "" {
				c.schemaError(ctx, schemaParserError(c.diagSource(), ce.Line(), ce.LocalName(), ce.LocalName(),
					"A '"+ce.LocalName()+"' must appear before 'defaultOpenContent'."))
			}
		case isXSDElement(ce, elemAnnotation):
			// annotation: allowed both before and after defaultOpenContent
		default:
			sawDeclaration = true
		}
	}
	if dec == nil {
		return nil
	}

	mode := OpenContentInterleave
	// @mode is an enumeration over xs:token (whiteSpace="collapse"): collapse before
	// comparing so e.g. mode="  interleave  " is accepted.
	switch normalizeWhiteSpace(getAttr(dec, attrMode), "collapse") {
	case "", "interleave":
		mode = OpenContentInterleave
	case "suffix":
		mode = OpenContentSuffix
	default:
		if c.filename != "" {
			c.schemaError(ctx, schemaParserErrorAttr(c.diagSource(), dec.Line(), dec.LocalName(), elemDefaultOpenContent, attrMode,
				"The value of 'mode' must be one of 'interleave' or 'suffix'."))
		}
	}
	appliesToEmpty := false
	if hasAttr(dec, attrAppliesToEmpty) {
		appliesToEmpty = c.readBooleanAttr(ctx, dec, attrAppliesToEmpty)
	}

	scan := scanOpenContentChildren(dec)
	c.reportOpenContentChildGrammar(ctx, dec, elemDefaultOpenContent, "A", "defaultOpenContent", scan)
	anyElem := scan.anyElem
	if anyElem == nil {
		if c.filename != "" {
			c.schemaError(ctx, schemaParserError(c.diagSource(), dec.Line(), dec.LocalName(), elemDefaultOpenContent,
				"A 'defaultOpenContent' must contain an 'any' wildcard."))
		}
		return nil
	}
	wc := c.parseOpenContentWildcard(ctx, anyElem)
	if wc == nil {
		return nil
	}
	return &OpenContent{Mode: mode, Wildcard: wc, AppliesToEmpty: appliesToEmpty}
}

// collectModelElementNames returns the set of element expanded names declared
// anywhere in a content model (recursing through nested model groups, including
// substitution-group members). It backs the open-content "interleave" rule:
// weak wildcards never claim an element whose name is declared in the model, so
// such elements must go through the normal content-model match.
func collectModelElementNames(mg *ModelGroup, schema *Schema) map[QName]bool {
	names := make(map[QName]bool)
	var walk func(g *ModelGroup)
	walk = func(g *ModelGroup) {
		if g == nil {
			return
		}
		for _, p := range g.Particles {
			switch term := p.Term.(type) {
			case *ElementDecl:
				names[term.Name] = true
				for _, m := range substitutableMembersFor(term, schema) {
					names[m.Name] = true
				}
			case *ModelGroup:
				walk(term)
			}
		}
	}
	walk(mg)
	return names
}

// modelGroupWildcardAdmitsName reports whether a content model contains a DECLARED
// wildcard particle (recursing through nested groups) that admits the expanded name
// {local, ns}. It backs the interleave open-content partition: a child matching a
// declared xs:any can satisfy that wildcard in the declared sub-sequence rather than
// being forced into open content. Non-emitting (maxOccurs=0) wildcards are skipped —
// they emit nothing, so they cannot consume a child.
func modelGroupWildcardAdmitsName(mg *ModelGroup, local, ns string, schema *Schema) bool {
	if mg == nil {
		return false
	}
	for _, p := range mg.Particles {
		switch term := p.Term.(type) {
		case *Wildcard:
			if p.MaxOccurs != 0 && wildcardAllowsExpandedName(term, local, ns, schema, false) {
				return true
			}
		case *ModelGroup:
			if modelGroupWildcardAdmitsName(term, local, ns, schema) {
				return true
			}
		}
	}
	return false
}

// collectEmittingModelElementNames returns every element name the content model can
// ACTUALLY MATCH as declared content: restricted to EMITTING element particles (a
// particle whose effective maxOccurs is 0 — its own maxOccurs == 0, or any ANCESTOR
// group is maxOccurs == 0/prohibited — emits nothing, so a child of that name is open
// content) AND to names elemMatchesDeclOrSubst can consume — the particle's OWN name
// when CONCRETE plus its INSTANCE-ADMISSIBLE (abstract-EXCLUDED) substitution-group
// members. An ABSTRACT head/member can never appear as declared content, so its name
// is open-content-eligible and must NOT count as a misplaced declared name. Used by
// the open-content drop guard (kept/dropped split) and suffix validation (misplaced
// trailing-name check), consistent with baseAdmissibleElementNames / instanceSubstMembers
// and the drop guard's maxOccursForName==0 non-emitting definition.
func collectEmittingModelElementNames(mg *ModelGroup, schema *Schema) map[QName]bool {
	names := make(map[QName]bool)
	var walk func(g *ModelGroup, ancestorsEmitting bool)
	walk = func(g *ModelGroup, ancestorsEmitting bool) {
		if g == nil {
			return
		}
		groupEmitting := ancestorsEmitting && g.MaxOccurs != 0
		for _, p := range g.Particles {
			switch term := p.Term.(type) {
			case *ElementDecl:
				if groupEmitting && p.MaxOccurs != 0 {
					if !term.Abstract {
						names[term.Name] = true
					}
					for _, m := range instanceSubstMembers(term, schema) {
						names[m.Name] = true
					}
				}
			case *ModelGroup:
				walk(term, groupEmitting)
			}
		}
	}
	walk(mg, true)
	return names
}

// pruneNonEmittingParticles returns a shallow copy of the model group with every
// NON-EMITTING particle removed: a direct particle with maxOccurs == 0, any nested
// group with maxOccurs == 0, and any nested group whose members are all pruned away.
// A group with no emitting members emits nothing, but it is still EMPTIABLE (it
// matches ONLY the empty sequence), and that emptiability is SEMANTICALLY LOAD-BEARING
// inside an xs:choice: an empty branch makes the whole choice emptiable, so silently
// dropping it would turn a previously-emptiable choice into one that REQUIRES another
// branch (a false reject). The prune is therefore semantics-preserving:
//
//   - In a SEQUENCE (or xs:all) parent, an emptied member is a no-op (matching empty
//     consumes nothing), so it is dropped — a required sibling stays required.
//   - In a CHOICE parent, an emptied branch is REPLACED by a normalized emptiable
//     empty-SEQUENCE particle (minOccurs 0). The choice thus stays emptiable (one
//     branch matches empty), and the branch is a SEQUENCE — never a literally-empty
//     choice the matcher would treat as a missing required branch (round-15/16).
//
// Emitting group particles are kept with their children recursively pruned. Element
// and wildcard terms (and emitting non-group particles) are SHARED — the matcher only
// reads them. Used by the XSD 1.1 open-content matcher so a prohibited particle cannot
// consume a child (the child falls through to open content), consistent with the
// non-emitting name-collection / drop-guard definition. The ordinary matcher is never
// pruned. The TOP-LEVEL model may legitimately become empty (no particles); the caller
// (validateContentModelOpen) normalizes that to an empty sequence so all children route
// to open content.
func pruneNonEmittingParticles(mg *ModelGroup) *ModelGroup {
	if mg == nil {
		return nil
	}
	clone := *mg
	clone.Particles = make([]*Particle, 0, len(mg.Particles))
	for _, p := range mg.Particles {
		if p.MaxOccurs == 0 {
			continue // direct prohibited particle: emits nothing
		}
		if grp, ok := p.Term.(*ModelGroup); ok {
			pruned := pruneNonEmittingParticles(grp)
			if pruned == nil || len(pruned.Particles) == 0 {
				// The group emits nothing but is EMPTIABLE. In a SEQUENCE/all parent an
				// empty member is a no-op: drop it. In a CHOICE parent the emptiable
				// branch keeps the choice emptiable: preserve it as a normalized empty
				// SEQUENCE branch (minOccurs 0) so the matcher still matches the choice by
				// consuming nothing, and never via a literally-empty choice.
				if mg.Compositor == CompositorChoice {
					clone.Particles = append(clone.Particles, &Particle{
						MinOccurs: 0,
						MaxOccurs: 1,
						Term:      &ModelGroup{Compositor: CompositorSequence, MinOccurs: 0, MaxOccurs: 1},
					})
				}
				continue
			}
			np := *p
			np.Term = pruned
			clone.Particles = append(clone.Particles, &np)
			continue
		}
		clone.Particles = append(clone.Particles, p)
	}
	return &clone
}

// resolveDefinedSiblings populates SiblingNames on every xs:any wildcard that
// carries @notQName="##definedSibling" (XSD 1.1). The sibling set is the names
// of the element declarations that appear in the SAME content model as the
// wildcard, so the wildcard never claims a child a sibling element declaration
// would match. Runs after group refs are expanded so nested/group-contributed
// siblings are included.
//
// It must visit ALL parsed complex types, not just NAMED ones (c.schema.types):
// an inline ANONYMOUS complexType (e.g. on a local element declaration) also
// carries content models with ##definedSibling wildcards. Anonymous types are
// recorded in c.typeDefSources by parseComplexType, so iterate that map's keys
// in addition to the named types, deduplicating by *TypeDef pointer.
func (c *compiler) resolveDefinedSiblings() {
	visited := make(map[*TypeDef]struct{})
	resolve := func(td *TypeDef) {
		if td == nil || td.ContentModel == nil {
			return
		}
		if _, seen := visited[td]; seen {
			return
		}
		visited[td] = struct{}{}
		if !modelGroupHasDefinedSibling(td.ContentModel) {
			return
		}
		// The content-model tree may be SHARED with other types: group-ref
		// expansion reuses the group definition's particle slice
		// (link_refs.go `placeholder.Particles = grp.Particles`) and type
		// extension embeds the base type's model-group pointer
		// (link_refs.go `Term: baseMG`). assignDefinedSiblings mutates the
		// *Wildcard terms, so a shared wildcard would have ITS SiblingNames
		// overwritten by whichever owning type is resolved last — nondeterministic
		// (map iteration order). Deep-clone this type's content model so it owns
		// its own wildcard objects before assigning. Only types whose content
		// model actually carries a ##definedSibling wildcard pay the clone cost.
		td.ContentModel = cloneModelGroupForSiblings(td.ContentModel)
		names := collectModelElementNames(td.ContentModel, c.schema)
		var siblings []QName
		for qn := range names {
			siblings = append(siblings, qn)
		}
		assignDefinedSiblings(td.ContentModel, siblings)
	}
	for _, td := range c.schema.types {
		resolve(td)
	}
	for td := range c.typeDefSources {
		resolve(td)
	}
}

// modelGroupHasDefinedSibling reports whether a model-group tree contains any
// wildcard term flagged @notQName="##definedSibling".
func modelGroupHasDefinedSibling(mg *ModelGroup) bool {
	if mg == nil {
		return false
	}
	for _, p := range mg.Particles {
		switch term := p.Term.(type) {
		case *Wildcard:
			if term.NotQNameDefinedSibling {
				return true
			}
		case *ModelGroup:
			if modelGroupHasDefinedSibling(term) {
				return true
			}
		}
	}
	return false
}

// cloneModelGroupForSiblings deep-copies a model-group tree, giving it fresh
// ModelGroup, Particle, and Wildcard objects so per-type ##definedSibling
// resolution cannot alias a wildcard shared via group-ref expansion or extension
// embedding. ElementDecl terms are shared (read-only for sibling resolution).
func cloneModelGroupForSiblings(mg *ModelGroup) *ModelGroup {
	if mg == nil {
		return nil
	}
	nmg := *mg
	nmg.Particles = make([]*Particle, len(mg.Particles))
	for i, p := range mg.Particles {
		np := *p
		switch term := p.Term.(type) {
		case *Wildcard:
			wc := *term
			np.Term = &wc
		case *ModelGroup:
			np.Term = cloneModelGroupForSiblings(term)
		}
		nmg.Particles[i] = &np
	}
	return &nmg
}

// assignDefinedSiblings walks a model group tree and, for every wildcard term
// flagged ##definedSibling, sets its SiblingNames to the supplied set.
func assignDefinedSiblings(mg *ModelGroup, siblings []QName) {
	if mg == nil {
		return
	}
	for _, p := range mg.Particles {
		switch term := p.Term.(type) {
		case *Wildcard:
			if term.NotQNameDefinedSibling {
				term.SiblingNames = siblings
			}
		case *ModelGroup:
			assignDefinedSiblings(term, siblings)
		}
	}
}

// validateContentModelOpen validates an element's children against a content
// model carrying XSD 1.1 open content.
//
//   - suffix: the declared content is matched from the start; every remaining
//     trailing child must match the open wildcard.
//   - interleave: children whose expanded name is NOT declared in the model and
//     which match the open wildcard are removed (they are the open content);
//     the rest must satisfy the declared model. An element whose name IS declared
//     always goes through the model (weak-wildcard precedence), so a misplaced or
//     excess declared element is still a violation rather than open content.
func (vc *validationContext) validateContentModelOpen(ctx context.Context, elem *helium.Element, mg *ModelGroup, oc *OpenContent) error {
	// XSD 1.1 open content: a NON-EMITTING declared particle (effective maxOccurs 0 —
	// its own maxOccurs=0, or under a maxOccurs=0 ancestor group) emits nothing, so it
	// must not consume a child in the declared-content matcher — a child of that name
	// falls through to open content. The runtime matcher (matchElementParticle /
	// matchWildcardParticle) still grabs a matching child ONCE before its maxOccurs
	// check, so without pruning a maxOccurs=0 particle would consume the child and
	// validate it against the prohibited element's type. Prune the model fed to the
	// open-content matcher (and the name collection / EDC scope below) so it is
	// consistent with the compile-time non-emitting semantics. Scoped to the Version11
	// open-content path — validateContentModelOpen is only reached when
	// td.OpenContent != nil — so the ordinary (no open content / XSD 1.0) matcher is
	// unchanged.
	mg = pruneNonEmittingParticles(mg)
	if mg == nil || len(mg.Particles) == 0 {
		// The entire declared model is non-emitting (e.g. an xs:choice all of whose
		// branches are prohibited): it matches only the empty sequence, so every child
		// routes to open content. Use an empty SEQUENCE with minOccurs 0 so the matcher
		// reports "match nothing" rather than a missing required branch (an empty choice
		// with minOccurs>=1 would otherwise fail before leftover/open-content handling).
		mg = &ModelGroup{Compositor: CompositorSequence, MinOccurs: 0, MaxOccurs: 1}
	}
	children := collectChildElements(elem)

	if oc.Mode == OpenContentSuffix {
		consumed, err := vc.matchContentModelSuffix(ctx, elem, mg, children)
		if err != nil {
			return err
		}
		leftover := children[consumed:]
		// A trailing child whose name is declared by an EMITTING particle is a
		// misplaced declared element, not open content (weak-wildcard precedence): the
		// model already had its chance to consume it, so it is unexpected. A
		// NON-EMITTING declared name (maxOccurs=0 particle/ancestor) cannot be matched
		// by the model, so such a trailing child is legitimately open content.
		declaredNames := collectEmittingModelElementNames(mg, vc.schema)
		for _, ch := range leftover {
			if declaredNames[QName{Local: ch.name, NS: ch.ns}] {
				vc.reportValidityError(ctx, vc.filename, ch.elem.Line(), ch.displayName, "This element is not expected.")
				return fmt.Errorf("unexpected element")
			}
		}
		return vc.validateOpenChildren(ctx, elem, mg, oc.Wildcard, leftover)
	}

	// interleave: §3.4.4.3.2 requires the children to be partitionable into a
	// sub-sequence valid against the declared content model and a sub-sequence each
	// of whose members matches the open wildcard. Seed the declared sub-sequence with
	// every child that matches a declared ELEMENT NAME or a declared WILDCARD particle
	// (so it can satisfy that particle's occurrence — the declared content model may
	// itself contain xs:any), and the open sub-sequence with the rest that match the
	// open wildcard. Then refine: a declared-seeded child the model cannot place at
	// its position (e.g. an excess occurrence after a bounded element/wildcard
	// particle is exhausted) is moved to the open sub-sequence when it too matches the
	// open wildcard — so declared and open content may match the same names/namespaces.
	declaredNames := collectModelElementNames(mg, vc.schema)
	var declared, open []childElem
	for _, ch := range children {
		qn := QName{Local: ch.name, NS: ch.ns}
		if declaredNames[qn] || modelGroupWildcardAdmitsName(mg, ch.name, ch.ns, vc.schema) {
			declared = append(declared, ch)
			continue
		}
		if wildcardAllowsExpandedName(oc.Wildcard, ch.name, ch.ns, vc.schema, false) {
			open = append(open, ch)
			continue
		}
		declared = append(declared, ch)
	}
	declared, open = vc.refineInterleavePartition(ctx, elem, mg, oc.Wildcard, declared, open)
	if err := vc.validateContentModelTop(ctx, elem, mg, declared); err != nil {
		return err
	}
	return vc.validateOpenChildren(ctx, elem, mg, oc.Wildcard, open)
}

// refineInterleavePartition moves declared-but-unplaceable children that match
// the open wildcard from the declared sub-sequence into the open one, so an
// interleave open content admits a declared name that the content model cannot
// accommodate at its position (e.g. an extra occurrence after a bounded particle
// is exhausted). It TRIALS the model match with diagnostics suppressed; the caller
// re-runs the match for real on the returned declared set. The trial terminates:
// each iteration removes one child from the (finite) declared set.
//
// The trial match may report an ERROR while still having consumed a PREFIX of the
// declared set (the match stopped at the child at index `consumed`): per the
// §3.4.4.3.2 existential partition that child may belong to the OPEN sub-sequence,
// so a trial error must NOT abort refinement. As long as a blocking child remains
// (consumed < len(declared)) and matches the open wildcard, move it to the open
// sub-sequence and re-trial. Refinement stops only when the declared set is fully
// consumed (consumed >= len) — including the "missing required particle at the
// end" case, which no move can fix — or the blocker is not admissible as open
// content (left in declared so the real match reports it as unexpected).
//
// A child a declared WILDCARD structurally CONSUMED but whose CONTENT validation FAILS
// (e.g. a strict declared `xs:any` matching a child with no valid global declaration)
// is counted as consumed by the matcher, so the trial reports an ERROR with `consumed`
// already at the end. Per the same existential partition that child may instead be OPEN
// content, so the trial ERROR (not just a structural shortfall) must also trigger a
// move: when every child was consumed but the trial errored, move the LAST declared
// child that matches the open wildcard AND is NOT a declared ELEMENT NAME (a declared
// name must be validated as the element — weak-wildcard precedence — never spilled) to
// the open sub-sequence and re-trial. A child invalid as BOTH declared and open does not
// match the open wildcard, so it is not moved and the real match / open validation still
// rejects it (no false-accept). Termination: every iteration removes one child.
func (vc *validationContext) refineInterleavePartition(ctx context.Context, elem *helium.Element, mg *ModelGroup, wc *Wildcard, declared, open []childElem) ([]childElem, []childElem) {
	declaredNames := collectModelElementNames(mg, vc.schema)
	for {
		vc.suppressDepth++
		consumed, err := vc.matchContentModel(ctx, elem, mg, declared)
		vc.suppressDepth--
		if err == nil && consumed >= len(declared) {
			return declared, open
		}
		if consumed < len(declared) {
			// STRUCTURAL blocker: the child at `consumed` cannot be placed (an excess
			// occurrence after a bounded element/wildcard particle, or an unexpected name).
			// Move it to open when it matches the open wildcard; otherwise leave it for the
			// real match to report.
			blocker := declared[consumed]
			if !wildcardAllowsExpandedName(wc, blocker.name, blocker.ns, vc.schema, false) {
				return declared, open
			}
			open = append(open, blocker)
			declared = append(declared[:consumed], declared[consumed+1:]...)
			continue
		}
		// CONTENT failure: err != nil but every declared child was structurally consumed —
		// a declared WILDCARD match failed content validation. Move the last open-eligible,
		// non-declared-name child to open and re-trial.
		moved := false
		for i, c := range slices.Backward(declared) {
			if declaredNames[QName{Local: c.name, NS: c.ns}] {
				continue // a declared element name is validated as the element, not spilled
			}
			if !wildcardAllowsExpandedName(wc, c.name, c.ns, vc.schema, false) {
				continue // not admissible as open content
			}
			open = append(open, c)
			declared = append(declared[:i], declared[i+1:]...)
			moved = true
			break
		}
		if !moved {
			return declared, open
		}
	}
}

// matchContentModelSuffix matches the declared content model as a leading PREFIX
// for the open-content suffix mode, returning how many children it consumed
// without reporting trailing children as errors (the caller validates them as open
// content). For an xs:all group it uses the lenient member matcher so a trailing
// open-content child does not abort the all match; for sequence/choice the normal
// matcher already stops at the first non-matching child.
func (vc *validationContext) matchContentModelSuffix(ctx context.Context, parent *helium.Element, mg *ModelGroup, children []childElem) (int, error) {
	if mg.Compositor == CompositorAll && vc.version == Version11 {
		return vc.matchAll11(ctx, parent, mg, children, 0, mg, true)
	}
	return vc.matchContentModel(ctx, parent, mg, children)
}

// validateOpenChildren validates a set of open-content child elements against the
// open wildcard (processContents lax/strict/skip). Any child that does not match
// the wildcard's namespace constraint is reported as unexpected. The declared
// content model mg is threaded as the EDC scope so an open-content-admitted child
// gets the SAME dynamic wildcard Element-Declarations-Consistent check the ordinary
// wildcard-particle path applies (a child whose name collides with a same-named
// local declaration whose type is inconsistent with the wildcard's governing type
// is rejected) — it must NOT be nil.
func (vc *validationContext) validateOpenChildren(ctx context.Context, parent *helium.Element, mg *ModelGroup, wc *Wildcard, open []childElem) error {
	if len(open) == 0 {
		return nil
	}
	p := &Particle{MinOccurs: 0, MaxOccurs: Unbounded, Term: wc}
	consumed, err := vc.matchWildcardParticle(ctx, parent, p, wc, open, 0, mg)
	if err != nil {
		return err
	}
	if consumed < len(open) {
		ce := open[consumed]
		vc.reportValidityError(ctx, vc.filename, ce.elem.Line(), ce.displayName, "This element is not expected.")
		return fmt.Errorf("unexpected element")
	}
	return nil
}
