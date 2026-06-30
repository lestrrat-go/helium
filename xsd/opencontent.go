package xsd

import (
	"context"
	"fmt"

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
	}
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

// baseModelAdmitsOpenContent reports whether the base type's DECLARED content
// model already admits the (effectively unbounded) derived open-content wildcard,
// so a restriction may validly re-express a base `xs:any` particle as open content
// even though the base carries no {open content} of its own (saxonData
// Open/open022). The check is CONSERVATIVE and fail-closed: it returns true ONLY
// when there exists a wildcard W in the base content model such that ALL of:
//
//	(a) W's namespace constraint is a SUPERSET of the derived open-content
//	    wildcard's (derived ⊆ W) and W's processContents is at least as strong;
//	(b) W is REACHABLE and effectively UNBOUNDED — the product of maxOccurs along
//	    the path from the content-model root to W is unbounded with no zero factor
//	    (a maxOccurs="0" ancestor or particle makes W unreachable; an effectively
//	    bounded W, e.g. maxOccurs="1", cannot admit a second open child);
//	(c) W has NO required siblings — entering the model and consuming W needs no
//	    other content, i.e. at every sequence/all level on the path every OTHER
//	    sibling particle is emptiable (a choice imposes no sibling requirement). A
//	    required sibling element (e.g. `sequence{element a, any}`) means the base
//	    admits the open child ONLY alongside `a`, so the open content is NOT a
//	    language subset and is rejected.
//
// The decision is made STRUCTURALLY during a single descent (see
// openContentAdmitWalker): each wildcard is judged at its ACTUAL position with its
// own ancestor occurrence product and sibling set, never by reconstructing a path
// via *Particle pointer identity. This is required because group-ref expansion
// SHARES the group definition's particle slice (link_refs.go
// `placeholder.Particles = grp.Particles`), so the SAME wildcard *Particle pointer
// can appear at two sibling positions (e.g. `(G, G)*`); a pointer-based path check
// would treat both positions as "on the target path" and miss the OTHER G as a
// required sibling. open022 (a standalone unbounded wildcard with no required
// siblings) is accepted; a required-sibling base, a maxOccurs="0" ancestor, and
// the shared `(G, G)*` group-ref case are rejected.
func baseModelAdmitsOpenContent(base *TypeDef, derived *Wildcard, schema *Schema) bool {
	if base == nil || derived == nil || base.ContentModel == nil {
		return false
	}
	w := openContentAdmitWalker{derived: derived, schema: schema}
	// The root is reachable, not-yet-unbounded, and has no required sibling above
	// it; the root group's own occurrence is folded in by descend.
	return w.descend(base.ContentModel, true, false, true)
}

// openContentAdmitWalker carries the immutable inputs of the
// baseModelAdmitsOpenContent descent: the derived open-content wildcard whose
// admissibility is sought, and the schema for wildcard-subset resolution.
type openContentAdmitWalker struct {
	derived *Wildcard
	schema  *Schema
}

// descend visits model group mg, folding mg's OWN occurrence into the path state:
//   - reachable: the maxOccurs product along the root→mg path has no zero factor
//     (a maxOccurs="0" anywhere on the path makes everything below unreachable);
//   - unbounded: some maxOccurs along that path is unbounded;
//   - noRequiredSibling: every sequence/all sibling NOT descended into so far is
//     emptiable (a choice level contributes no requirement).
//
// It returns true when a wildcard satisfying conditions (a)-(c) is found. A nested
// group particle's occurrence equals the inner group's own MinOccurs/MaxOccurs
// (copied at parse time), so descend folds it once via the recursive call's mg and
// does NOT re-fold the wrapping particle for group terms.
func (w openContentAdmitWalker) descend(mg *ModelGroup, reachable, unbounded, noRequiredSibling bool) bool {
	if mg == nil {
		return false
	}
	reachable = reachable && mg.MaxOccurs != 0
	if !reachable {
		return false
	}
	unbounded = unbounded || mg.MaxOccurs == Unbounded
	isChoice := mg.Compositor == CompositorChoice
	for i, p := range mg.Particles {
		// Descending into particle i: for a sequence/all every OTHER sibling must be
		// emptiable; a choice picks one branch, so siblings impose no requirement.
		siblingOK := noRequiredSibling
		if !isChoice {
			siblingOK = siblingOK && othersEmptiableExcept(mg, i)
		}
		switch term := p.Term.(type) {
		case *Wildcard:
			pReachable := reachable && p.MaxOccurs != 0
			pUnbounded := unbounded || p.MaxOccurs == Unbounded
			if pReachable && pUnbounded && siblingOK &&
				wildcardConstraintSubset(w.derived, term, w.schema, false) &&
				processContentsStrength(w.derived.ProcessContents) >= processContentsStrength(term.ProcessContents) {
				return true
			}
		case *ModelGroup:
			if w.descend(term, reachable, unbounded, siblingOK) {
				return true
			}
		}
	}
	return false
}

// othersEmptiableExcept reports whether every particle in mg other than the one at
// index skip is emptiable. It compares by INDEX, not pointer, so a model group
// holding the same *Particle pointer twice (possible after shared group-ref
// expansion) still evaluates each sibling position independently.
func othersEmptiableExcept(mg *ModelGroup, skip int) bool {
	for i, p := range mg.Particles {
		if i == skip {
			continue
		}
		if !particleEmptiable(p) {
			return false
		}
	}
	return true
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
	switch getAttr(elem, attrMode) {
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

	anyElem, annotations, anyCount := scanOpenContentChildren(elem)
	if annotations > 1 && c.filename != "" {
		c.schemaError(ctx, schemaParserError(c.diagSource(), elem.Line(), elem.LocalName(), elemOpenContent,
			"An 'openContent' must not have more than one 'annotation'."))
	}
	if anyCount > 1 && c.filename != "" {
		c.schemaError(ctx, schemaParserError(c.diagSource(), elem.Line(), elem.LocalName(), elemOpenContent,
			"An 'openContent' must not have more than one 'any' wildcard."))
	}

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

// scanOpenContentChildren walks the children of an <xs:openContent> or
// <xs:defaultOpenContent> element, returning the FIRST <xs:any> wildcard element
// (nil if none), the number of <xs:annotation> children, and the TOTAL number of
// <xs:any> wildcard children seen. Callers reject more than one wildcard child as
// a schema error (the content model permits at most one).
func scanOpenContentChildren(elem *helium.Element) (*helium.Element, int, int) {
	var anyElem *helium.Element
	annotations := 0
	anyCount := 0
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
			annotations++
		case isXSDElement(ce, elemAny):
			anyCount++
			if anyElem == nil {
				anyElem = ce
			}
		}
	}
	return anyElem, annotations, anyCount
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
	switch getAttr(dec, attrMode) {
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

	anyElem, annotations, anyCount := scanOpenContentChildren(dec)
	if annotations > 1 && c.filename != "" {
		c.schemaError(ctx, schemaParserError(c.diagSource(), dec.Line(), dec.LocalName(), elemDefaultOpenContent,
			"A 'defaultOpenContent' must not have more than one 'annotation'."))
	}
	if anyCount > 1 && c.filename != "" {
		c.schemaError(ctx, schemaParserError(c.diagSource(), dec.Line(), dec.LocalName(), elemDefaultOpenContent,
			"A 'defaultOpenContent' must not have more than one 'any' wildcard."))
	}
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
	children := collectChildElements(elem)

	if oc.Mode == OpenContentSuffix {
		consumed, err := vc.matchContentModelSuffix(ctx, elem, mg, children)
		if err != nil {
			return err
		}
		leftover := children[consumed:]
		// A trailing child whose name is declared in the model is a misplaced
		// declared element, not open content (weak-wildcard precedence): the model
		// already had its chance to consume it, so it is unexpected.
		declaredNames := collectModelElementNames(mg, vc.schema)
		for _, ch := range leftover {
			if declaredNames[QName{Local: ch.name, NS: ch.ns}] {
				vc.reportValidityError(ctx, vc.filename, ch.elem.Line(), ch.displayName, "This element is not expected.")
				return fmt.Errorf("unexpected element")
			}
		}
		return vc.validateOpenChildren(ctx, elem, oc.Wildcard, leftover)
	}

	// interleave: §3.4.4.3.2 requires the children to be partitionable into a
	// sub-sequence valid against the declared content model and a sub-sequence each
	// of whose members matches the open wildcard. Start from a name-based split
	// (children whose names are NOT declared and which match the wildcard are open),
	// then refine: a DECLARED-named child the model cannot place at its position is
	// moved to the open sub-sequence when it too matches the wildcard. This handles
	// the case where open content and declared content match the same names (e.g. a
	// second occurrence of a declared element appearing after the model is satisfied).
	declaredNames := collectModelElementNames(mg, vc.schema)
	var declared, open []childElem
	for _, ch := range children {
		qn := QName{Local: ch.name, NS: ch.ns}
		if !declaredNames[qn] && wildcardAllowsExpandedName(oc.Wildcard, ch.name, ch.ns, vc.schema, false) {
			open = append(open, ch)
			continue
		}
		declared = append(declared, ch)
	}
	declared, open = vc.refineInterleavePartition(ctx, elem, mg, oc.Wildcard, declared, open)
	if err := vc.validateContentModelTop(ctx, elem, mg, declared); err != nil {
		return err
	}
	return vc.validateOpenChildren(ctx, elem, oc.Wildcard, open)
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
func (vc *validationContext) refineInterleavePartition(ctx context.Context, elem *helium.Element, mg *ModelGroup, wc *Wildcard, declared, open []childElem) ([]childElem, []childElem) {
	for {
		vc.suppressDepth++
		consumed, _ := vc.matchContentModel(ctx, elem, mg, declared)
		vc.suppressDepth--
		if consumed >= len(declared) {
			return declared, open
		}
		blocker := declared[consumed]
		if !wildcardAllowsExpandedName(wc, blocker.name, blocker.ns, vc.schema, false) {
			// The unplaceable child is not admissible as open content either; leave it
			// in declared so the real match reports it as unexpected.
			return declared, open
		}
		open = append(open, blocker)
		declared = append(declared[:consumed], declared[consumed+1:]...)
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
// the wildcard's namespace constraint is reported as unexpected.
func (vc *validationContext) validateOpenChildren(ctx context.Context, parent *helium.Element, wc *Wildcard, open []childElem) error {
	if len(open) == 0 {
		return nil
	}
	p := &Particle{MinOccurs: 0, MaxOccurs: Unbounded, Term: wc}
	consumed, err := vc.matchWildcardParticle(ctx, parent, p, wc, open, 0, nil)
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
