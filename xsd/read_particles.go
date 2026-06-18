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
	return nil
}

func (c *compiler) parseModelGroup(ctx context.Context, elem *helium.Element, compositor ModelGroupKind) (*ModelGroup, error) {
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
				placeholderMin, placeholderMax := parseParticleOccurs(ce)
				// Group reference — create a placeholder model group.
				placeholder := &ModelGroup{
					MinOccurs: placeholderMin,
					MaxOccurs: placeholderMax,
				}
				qn := c.resolveQName(ctx, ce, ref)
				c.groupRefs[placeholder] = qn
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
