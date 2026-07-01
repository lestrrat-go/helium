package xsd

import (
	"context"
	"fmt"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
)

func (c *compiler) parseNamedGroup(ctx context.Context, elem *helium.Element) error {
	name := getAttr(elem, attrName)
	if name == "" {
		return fmt.Errorf("xsd: named group missing name")
	}

	qn := QName{Local: name, NS: c.schema.targetNamespace}

	// Check for a duplicate model group BEFORE parsing the compositor body, so a
	// rejected component records no group references that would produce unrelated
	// follow-on errors. An xs:redefine override that was validated and consumed
	// by the redefine loop is pre-authorized and skips the report.
	if _, exists := c.schema.groups[qn]; exists && !c.redefineConsumed(redefineKindGroup, qn) {
		c.reportDuplicateComponent(ctx, elem, "group", "A global model group definition", qn)
		return nil
	}

	// A named model group definition has the content model (annotation?, (all |
	// choice | sequence)) per XSD Structures §3.7.2: exactly one model group child
	// (sequence, choice, or all), at most one annotation which must PRECEDE the
	// model group, and no other element children. This is a version-INDEPENDENT
	// schema-representation rule, enforced in both XSD 1.0 and 1.1.
	reportGrammar := func(ce *helium.Element, msg string) {
		c.schemaError(ctx, schemaParserError(c.diagSource(), ce.Line(), ce.LocalName(), "group", msg))
	}
	var mgSeen, annotSeen, hadGrammarError bool
	for child := range helium.Children(elem) {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		var compositor ModelGroupKind
		switch {
		case isXSDElement(ce, elemSequence):
			compositor = CompositorSequence
		case isXSDElement(ce, elemChoice):
			compositor = CompositorChoice
		case isXSDElement(ce, elemAll):
			compositor = CompositorAll
		case isXSDElement(ce, elemAnnotation):
			switch {
			case annotSeen:
				reportGrammar(ce, "A model group definition must not have more than one annotation.")
				hadGrammarError = true
			case mgSeen:
				reportGrammar(ce, "The annotation must appear before the model group ('all', 'choice' or 'sequence').")
				hadGrammarError = true
			}
			annotSeen = true
			continue
		default:
			reportGrammar(ce, fmt.Sprintf("The content of a model group definition is restricted to (annotation?, (all | choice | sequence)); '%s' is not allowed.", ce.LocalName()))
			hadGrammarError = true
			continue
		}
		if mgSeen {
			reportGrammar(ce, "A model group definition must contain exactly one model group ('all', 'choice' or 'sequence').")
			hadGrammarError = true
			continue
		}
		mgSeen = true
		mg, err := c.parseModelGroup(ctx, ce, compositor)
		if err != nil {
			return err
		}
		c.schema.groups[qn] = mg
		// Record the declaring source so cos-element-consistent can be run over
		// standalone named groups (those no complex type references). The
		// declaring file is the include file when inside an include/redefine,
		// else this compiler's filename (mirroring IDConstraint.Source).
		source := c.filename
		if c.includeFile != "" {
			source = c.includeFile
		}
		c.groupSources[qn] = groupSource{line: elem.Line(), source: source}
	}
	// The model group child is REQUIRED. A group carrying only an annotation (or
	// only stray children) is invalid; suppress the duplicate diagnostic when a
	// stray/misplaced child already produced a grammar error on this group.
	if !mgSeen && !hadGrammarError {
		reportGrammar(elem, "A model group definition must contain exactly one model group ('all', 'choice' or 'sequence').")
	}
	return nil
}

func (c *compiler) parseNamedAttributeGroup(ctx context.Context, elem *helium.Element) error {
	name := getAttr(elem, attrName)
	if name == "" {
		return fmt.Errorf("xsd: named attributeGroup missing name")
	}

	qn := QName{Local: name, NS: c.schema.targetNamespace}
	// An xs:redefine override that was validated and consumed by the redefine
	// loop is pre-authorized and skips the report.
	if _, exists := c.schema.attrGroups[qn]; exists && !c.redefineConsumed(redefineKindAttrGroup, qn) {
		c.reportDuplicateComponent(ctx, elem, "attributeGroup", "A global attribute group definition", qn)
		return nil
	}
	var attrs []*AttrUse
	// XSD 1.1: an xs:anyAttribute, if present, must be the OPTIONAL FINAL child of
	// the group (XSD 3.6.2), and there may be at most one. anyAttributeSeen tracks
	// it so a later attribute/attributeGroup child, or a second wildcard, is
	// rejected. Gated on Version11 (1.0 ignores group wildcards entirely, so its
	// grammar handling stays byte-identical).
	var anyAttributeSeen bool
	reportAfterWildcard := func(ce *helium.Element) {
		c.schemaError(ctx, schemaParserError(c.diagSource(), ce.Line(), ce.LocalName(), "attributeGroup",
			fmt.Sprintf("The attribute declaration '%s' must appear before the attribute wildcard 'anyAttribute'.", ce.LocalName())))
	}
	for child := range helium.Children(elem) {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		if isXSDElement(ce, elemAttribute) {
			if c.version == Version11 && anyAttributeSeen {
				reportAfterWildcard(ce)
				continue
			}
			// A use="prohibited" attribute declared directly inside an
			// <xs:attributeGroup> is pointless: it cannot remove a use the group
			// itself declares, and propagating it as a blocking use would wrongly
			// stop an xs:anyAttribute wildcard in a referencing type from admitting
			// the attribute. libxml2 warns and SKIPS it, so the wildcard still
			// admits the attribute. Mirror that here (parent type is attributeGroup).
			if getAttr(ce, attrUse) == attrValProhibited {
				// XSD 1.1 Schema Representation Constraint (Attribute Declaration
				// Representation OK): a prohibited attribute use must not carry a
				// value constraint. The use itself is skipped as pointless (below),
				// but the fixed-value defect is still a schema error in 1.1. Gated on
				// Version11 so 1.0 stays byte-identical (default is already rejected
				// in both versions by checkAttributeUse's "default requires
				// use=optional" rule, but that path is not reached here — a prohibited
				// use with default inside a group is silently skipped in 1.0, matching
				// origin, so only fixed is flagged and only in 1.1).
				if c.version == Version11 && hasAttr(ce, attrFixed) {
					c.schemaError(ctx, schemaParserError(c.diagSource(), ce.Line(), ce.LocalName(), "attribute",
						"The attribute 'fixed' is not allowed when the value of the attribute 'use' is 'prohibited'."))
				}
				if c.filename != "" {
					c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserWarning(c.diagSource(), ce.Line(), ce.LocalName(), "attribute",
						"Skipping attribute use prohibition, since it is pointless inside an <attributeGroup>."), helium.ErrorLevelWarning))
				}
				continue
			}
			au := c.parseAttributeUse(ctx, ce)
			attrs = append(attrs, au)
			continue
		}
		// XSD 1.1: capture an xs:anyAttribute wildcard declared in the group so a
		// referencing type can intersect it into its effective attribute wildcard.
		// It must be the final child and unique.
		if c.version == Version11 && isXSDElement(ce, elemAnyAttribute) {
			if anyAttributeSeen {
				c.schemaError(ctx, schemaParserError(c.diagSource(), ce.Line(), ce.LocalName(), "attributeGroup",
					fmt.Sprintf("An attribute group definition must not have more than one attribute wildcard (found a second '%s').", ce.LocalName())))
				continue
			}
			anyAttributeSeen = true
			c.attrGroupWildcards[qn] = c.parseAnyAttribute(ctx, ce)
			continue
		}
		// Record nested xs:attributeGroup ref children so checkAttrGroupDuplicates
		// can flatten the transitively-referenced groups and detect a duplicate
		// attribute use introduced through a reference (ag-props-correct.2).
		//
		// A direct SELF-reference (ref resolves to the group being defined) is a
		// CIRCULAR reference, which is disallowed OUTSIDE <redefine> (XSD §3.6.2
		// src-attribute_group.3 / Attribute Group Definition Representation OK 3).
		// The legitimate self-reference inside <redefine> is parsed by the redefine
		// override path (compile_imports.go), not here, so any self-reference that
		// reaches this point is genuinely circular and is reported and dropped (the
		// reference is cut to avoid further confusion, matching libxml2).
		if isXSDElement(ce, elemAttributeGroup) {
			if c.version == Version11 && anyAttributeSeen {
				reportAfterWildcard(ce)
				continue
			}
			if hasAttr(ce, attrRef) {
				ref := getAttr(ce, attrRef)
				refQN := c.resolveQName(ctx, ce, ref)
				if ref != "" && refQN == qn {
					// XSD 1.1 permits circular attribute group definitions (W3C bug
					// 15795 / attgC010-C031): a direct self-reference contributes
					// nothing new (the group's own attribute uses are already
					// collected), so it is silently dropped without a diagnostic.
					// XSD 1.0 rejects it (src-attribute_group.3), byte-identical.
					if c.version != Version11 {
						c.reportCircularAttrGroupRef(ctx, ce, qn)
					}
					continue
				}
				c.attrGroupRefChildren[qn] = append(c.attrGroupRefChildren[qn], refQN)
				// Record the back-edge ref element's own source so an indirect-cycle
				// diagnostic cites this <xs:attributeGroup ref="..."> line/file, not
				// the owning group's declaration line.
				c.attrGroupRefSources[qn] = append(c.attrGroupRefSources[qn], attrGroupSource{line: ce.Line(), source: c.diagSource()})
			}
		}
	}
	c.schema.attrGroups[qn] = attrs
	// Record the declaring source so the duplicate-attribute-use diagnostic cites
	// the file the group was actually declared in: the include file when inside an
	// include/redefine, else this compiler's filename (mirroring parseNamedGroup /
	// IDConstraint.Source).
	c.attrGroupSources[qn] = attrGroupSource{line: elem.Line(), source: c.diagSource()}
	return nil
}

func (c *compiler) parseModelGroup(ctx context.Context, elem *helium.Element, compositor ModelGroupKind) (*ModelGroup, error) {
	// An xs:all compositor has special occurrence constraints (XSD Part 1
	// §3.8.6 / cos-all-limited): its own minOccurs must be 0 or 1, its maxOccurs
	// must be 1, and each element particle directly inside it must have
	// minOccurs/maxOccurs of 0 or 1. libxml2 reports these with all-specific
	// wording instead of the generic xs:nonNegativeInteger/xs:allNNI diagnostics,
	// so the all compositor is validated by validateAllOccurs, not the generic
	// validateOccursAttrs.
	if compositor == CompositorAll {
		c.validateAllOccurs(ctx, elem)
	} else {
		c.validateOccursAttrs(ctx, elem)
	}
	minOccurs, maxOccurs := parseParticleOccurs(elem)
	mg := &ModelGroup{
		Compositor: compositor,
		MinOccurs:  minOccurs,
		MaxOccurs:  maxOccurs,
	}

	// The XML representation of an inline model group has the content model
	// (annotation?, particle*) per XSD Structures §3.8.2: an xs:sequence/xs:choice
	// admits (element | group | choice | sequence | any) particles, while an
	// xs:all admits only (element) in XSD 1.0 — 1.1 additionally admits (any |
	// group). In all three compositors at most one annotation is permitted and it
	// must PRECEDE every particle, and no non-particle element (e.g. xs:simpleType,
	// xs:complexType, xs:attribute) may appear. An inline model group also carries
	// no @name attribute. These are schema-representation rules; the version split
	// is confined to the xs:all particle-kind set so the sequence/choice grammar
	// and the XSD 1.0 output stay byte-identical for valid model groups.
	compElem := elemSequence
	switch compositor {
	case CompositorChoice:
		compElem = elemChoice
	case CompositorAll:
		compElem = elemAll
	}
	expectedContent := "(annotation?, (element | group | choice | sequence | any)*)"
	if compositor == CompositorAll {
		expectedContent = "(annotation?, element*)"
		if c.version == Version11 {
			expectedContent = "(annotation?, (element | any | group)*)"
		}
	}
	reportGrammar := func(ce *helium.Element, msg string) {
		c.schemaError(ctx, schemaParserError(c.diagSource(), ce.Line(), ce.LocalName(), compElem, msg))
	}
	reportStray := func(ce *helium.Element) {
		reportGrammar(ce, fmt.Sprintf("The content is not valid. Expected is %s.", expectedContent))
	}
	// An inline model group must not carry a @name attribute.
	if hasAttr(elem, attrName) {
		c.schemaError(ctx, schemaParserErrorAttr(c.diagSource(), elem.Line(), elem.LocalName(), compElem, attrName,
			"Attribute 'name' is not allowed on an inline model group."))
	}
	var annotSeen, particleSeen bool

	for child := range helium.Children(elem) {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		// annotation? — at most one, and it must precede every particle.
		if isXSDElement(ce, elemAnnotation) {
			switch {
			case annotSeen:
				reportGrammar(ce, "A model group must not have more than one annotation.")
			case particleSeen:
				reportGrammar(ce, "The annotation must appear before the particles of a model group.")
			}
			annotSeen = true
			continue
		}
		particleSeen = true
		switch {
		case isXSDElement(ce, elemElement):
			p, err := c.parseLocalElement(ctx, ce)
			if err != nil {
				return nil, err
			}
			// An element particle directly inside an xs:all may repeat at most
			// once: both minOccurs and maxOccurs must be 0 or 1 (cos-all-limited).
			// checkLocalElement has already emitted the generic occurs ordering
			// (e.g. min>max), so this only adds the all-specific diagnostic.
			if compositor == CompositorAll {
				c.checkAllElementParticleOccurs(ctx, ce)
			}
			mg.Particles = append(mg.Particles, p)
		case isXSDElement(ce, elemSequence):
			// xs:all admits no nested sequence in either version.
			if compositor == CompositorAll {
				reportStray(ce)
				continue
			}
			sub, err := c.parseModelGroup(ctx, ce, CompositorSequence)
			if err != nil {
				return nil, err
			}
			mg.Particles = append(mg.Particles, &Particle{
				MinOccurs: sub.MinOccurs,
				MaxOccurs: sub.MaxOccurs,
				Term:      sub,
			})
		case isXSDElement(ce, elemChoice):
			// xs:all admits no nested choice in either version.
			if compositor == CompositorAll {
				reportStray(ce)
				continue
			}
			sub, err := c.parseModelGroup(ctx, ce, CompositorChoice)
			if err != nil {
				return nil, err
			}
			mg.Particles = append(mg.Particles, &Particle{
				MinOccurs: sub.MinOccurs,
				MaxOccurs: sub.MaxOccurs,
				Term:      sub,
			})
		case isXSDElement(ce, elemAll):
			// cos-all-limited / Schema Component Constraint: All Group Limited
			// (XSD Part 1 §3.8.6): an 'all' model group may only constitute the
			// whole content model of a complex type — it must NOT appear nested
			// inside an xs:sequence or xs:choice. libxml2 rejects this as invalid
			// content of the enclosing compositor.
			if compositor == CompositorSequence || compositor == CompositorChoice {
				if c.filename != "" {
					parent := elemSequence
					if compositor == CompositorChoice {
						parent = elemChoice
					}
					c.schemaError(ctx, schemaParserError(c.diagSource(), ce.Line(), ce.LocalName(), parent,
						"The content is not valid. Expected is (annotation?, (element | group | choice | sequence | any)*)."))
				}
				continue
			}
			// XSD 1.1: an INLINE <xs:all> directly inside another <xs:all> is still
			// forbidden by cos-all-limited — only a <xs:group ref> resolving to an
			// all group (occurrence 1/1) is the relaxed allowed nesting. Reject and
			// SKIP so the invalid inline nested all is never built into the model
			// (where the matcher/subsumption flatteners would otherwise treat it as
			// the allowed group-ref case and silently accept it). Gated on Version11
			// so the XSD 1.0 path stays byte-identical.
			if compositor == CompositorAll && c.version == Version11 {
				if c.filename != "" {
					c.schemaError(ctx, schemaParserError(c.diagSource(), ce.Line(), ce.LocalName(), elemAll,
						"The content is not valid. Expected is (annotation?, (element | any | group)*)."))
				}
				continue
			}
			sub, err := c.parseModelGroup(ctx, ce, CompositorAll)
			if err != nil {
				return nil, err
			}
			mg.Particles = append(mg.Particles, &Particle{
				MinOccurs: sub.MinOccurs,
				MaxOccurs: sub.MaxOccurs,
				Term:      sub,
			})
		case isXSDElement(ce, elemAny):
			p := c.parseWildcard(ctx, ce)
			mg.Particles = append(mg.Particles, p)
		case isXSDElement(ce, elemGroup):
			// XSD 1.0 xs:all content is (annotation?, element*): a group reference
			// is not admitted. The 1.1 relaxation (a <xs:group ref> resolving to an
			// all group, occurrence 1/1) is handled by the ref path below and gated
			// downstream in checkAllGroupRef.
			if compositor == CompositorAll && c.version != Version11 {
				reportStray(ce)
				continue
			}
			ref := getAttr(ce, attrRef)
			if ref != "" {
				c.validateOccursAttrs(ctx, ce)
				placeholderMin, placeholderMax := parseParticleOccurs(ce)
				// Group reference — create a placeholder model group.
				placeholder := &ModelGroup{
					MinOccurs: placeholderMin,
					MaxOccurs: placeholderMax,
				}
				qn := c.resolveQName(ctx, ce, ref)
				c.groupRefs[placeholder] = qn
				// Nested reference: this group ref is contained inside another
				// model group (xs:sequence/xs:choice/xs:all), so a resolved 'all'
				// model group is forbidden here.
				c.groupRefSources[placeholder] = groupRefSource{
					line:             ce.Line(),
					local:            ce.LocalName(),
					nested:           true,
					parentCompositor: compositor,
					maxOccursRaw:     getAttr(ce, attrMaxOccurs),
					source:           c.diagSource(),
				}
				mg.Particles = append(mg.Particles, &Particle{
					MinOccurs: placeholder.MinOccurs,
					MaxOccurs: placeholder.MaxOccurs,
					Term:      placeholder,
				})
			}
		default:
			// Any other XSD-namespace child (e.g. xs:simpleType, xs:complexType,
			// xs:attribute) is not a particle and is not admitted by a model group's
			// (annotation?, particle*) content model. Foreign-namespace children are
			// left untouched to avoid over-rejecting extension content.
			if ce.URI() == lexicon.NamespaceXSD {
				reportStray(ce)
			}
		}
	}

	return mg, nil
}
