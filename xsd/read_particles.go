package xsd

import (
	"fmt"

	helium "github.com/lestrrat-go/helium"
)

func (c *compiler) parseNamedGroup(elem *helium.Element) error {
	name := getAttr(elem, "name")
	if name == "" {
		return fmt.Errorf("xsd: named group missing name")
	}

	// A named group has exactly one child compositor (sequence, choice, or all).
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce := child.(*helium.Element)
		var compositor ModelGroupKind
		switch {
		case isXSDElement(ce, "sequence"):
			compositor = CompositorSequence
		case isXSDElement(ce, "choice"):
			compositor = CompositorChoice
		case isXSDElement(ce, "all"):
			compositor = CompositorAll
		default:
			continue
		}
		mg, err := c.parseModelGroup(ce, compositor)
		if err != nil {
			return err
		}
		qn := QName{Local: name, NS: c.schema.targetNamespace}
		c.schema.groups[qn] = mg
		return nil
	}
	return nil
}

func (c *compiler) parseNamedAttributeGroup(elem *helium.Element) error {
	name := getAttr(elem, "name")
	if name == "" {
		return fmt.Errorf("xsd: named attributeGroup missing name")
	}

	qn := QName{Local: name, NS: c.schema.targetNamespace}
	var attrs []*AttrUse
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce := child.(*helium.Element)
		if isXSDElement(ce, "attribute") {
			au := c.parseAttributeUse(ce)
			attrs = append(attrs, au)
		}
	}
	c.schema.attrGroups[qn] = attrs
	return nil
}

func (c *compiler) parseModelGroup(elem *helium.Element, compositor ModelGroupKind) (*ModelGroup, error) {
	minOccurs, maxOccurs := parseParticleOccurs(elem)
	mg := &ModelGroup{
		Compositor: compositor,
		MinOccurs:  minOccurs,
		MaxOccurs:  maxOccurs,
	}

	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce := child.(*helium.Element)
		switch {
		case isXSDElement(ce, "element"):
			p, err := c.parseLocalElement(ce)
			if err != nil {
				return nil, err
			}
			mg.Particles = append(mg.Particles, p)
		case isXSDElement(ce, "sequence"):
			sub, err := c.parseModelGroup(ce, CompositorSequence)
			if err != nil {
				return nil, err
			}
			mg.Particles = append(mg.Particles, &Particle{
				MinOccurs: sub.MinOccurs,
				MaxOccurs: sub.MaxOccurs,
				Term:      sub,
			})
		case isXSDElement(ce, "choice"):
			sub, err := c.parseModelGroup(ce, CompositorChoice)
			if err != nil {
				return nil, err
			}
			mg.Particles = append(mg.Particles, &Particle{
				MinOccurs: sub.MinOccurs,
				MaxOccurs: sub.MaxOccurs,
				Term:      sub,
			})
		case isXSDElement(ce, "all"):
			sub, err := c.parseModelGroup(ce, CompositorAll)
			if err != nil {
				return nil, err
			}
			mg.Particles = append(mg.Particles, &Particle{
				MinOccurs: sub.MinOccurs,
				MaxOccurs: sub.MaxOccurs,
				Term:      sub,
			})
		case isXSDElement(ce, "any"):
			p := c.parseWildcard(ce)
			mg.Particles = append(mg.Particles, p)
		case isXSDElement(ce, "group"):
			ref := getAttr(ce, "ref")
			if ref != "" {
				placeholderMin, placeholderMax := parseParticleOccurs(ce)
				// Group reference — create a placeholder model group.
				placeholder := &ModelGroup{
					MinOccurs: placeholderMin,
					MaxOccurs: placeholderMax,
				}
				qn := c.resolveQName(ce, ref)
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
