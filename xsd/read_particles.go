package xsd

import (
	"context"
	"fmt"

	helium "github.com/lestrrat-go/helium"
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

	// A named group has exactly one child compositor (sequence, choice, or all).
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
		default:
			continue
		}
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
		return nil
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
			if ref := getAttr(ce, attrRef); ref != "" {
				refQN := c.resolveQName(ctx, ce, ref)
				if refQN == qn {
					c.reportCircularAttrGroupRef(ctx, ce, qn)
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

	for child := range helium.Children(elem) {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
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
		}
	}

	return mg, nil
}
