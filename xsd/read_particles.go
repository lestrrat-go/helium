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
	for child := range helium.Children(elem) {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		if isXSDElement(ce, elemAttribute) {
			au := c.parseAttributeUse(ctx, ce)
			attrs = append(attrs, au)
		}
	}
	c.schema.attrGroups[qn] = attrs
	// Record the declaring source so the duplicate-attribute-use diagnostic cites
	// the file the group was actually declared in. The declaring file is the
	// include file when inside an include/redefine, else this compiler's filename
	// (mirroring parseNamedGroup / IDConstraint.Source).
	source := c.filename
	if c.includeFile != "" {
		source = c.includeFile
	}
	c.attrGroupSources[qn] = attrGroupSource{line: elem.Line(), source: source}
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
					line:         ce.Line(),
					local:        ce.LocalName(),
					nested:       true,
					maxOccursRaw: getAttr(ce, attrMaxOccurs),
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
