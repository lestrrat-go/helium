package xsd

import (
	"fmt"
	"strings"

	helium "github.com/lestrrat-go/helium"
)

func (c *compiler) parseNamedComplexType(elem *helium.Element) error {
	name := getAttr(elem, "name")
	if name == "" {
		return fmt.Errorf("xsd: named complexType missing name")
	}

	td, err := c.parseComplexType(elem)
	if err != nil {
		return err
	}
	td.Name = QName{Local: name, NS: c.schema.targetNamespace}
	td.Abstract = getAttr(elem, "abstract") == attrValTrue

	// Parse final attribute with schema default.
	if hasAttr(elem, "final") {
		td.Final = parseElemFinalFlags(getAttr(elem, "final"))
		td.FinalSet = true
	} else {
		td.Final = c.schema.finalDefault & (FinalExtension | FinalRestriction)
	}

	c.typeDefSources[td] = typeDefSource{line: elem.Line(), isLocal: false}
	c.schema.types[td.Name] = td
	return nil
}

func (c *compiler) parseNamedSimpleType(elem *helium.Element) error {
	name := getAttr(elem, "name")
	if name == "" {
		return fmt.Errorf("xsd: named simpleType missing name")
	}

	td, err := c.parseSimpleType(elem)
	if err != nil {
		return err
	}
	td.Name = QName{Local: name, NS: c.schema.targetNamespace}

	// Parse final attribute with schema default.
	if hasAttr(elem, "final") {
		td.Final = parseFinalFlags(getAttr(elem, "final"))
		td.FinalSet = true
	} else {
		td.Final = c.schema.finalDefault & (FinalRestriction | FinalList | FinalUnion)
	}

	c.typeDefSources[td] = typeDefSource{line: elem.Line(), isLocal: false}
	c.schema.types[td.Name] = td
	return nil
}

func (c *compiler) parseComplexType(elem *helium.Element) (*TypeDef, error) {
	td := &TypeDef{}
	c.typeDefSources[td] = typeDefSource{line: elem.Line(), isLocal: true}

	mixed := getAttr(elem, "mixed")
	if mixed == attrValTrue {
		td.ContentType = ContentTypeMixed
	}

	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce := child.(*helium.Element)
		switch {
		case isXSDElement(ce, "sequence"):
			mg, err := c.parseModelGroup(ce, CompositorSequence)
			if err != nil {
				return nil, err
			}
			td.ContentModel = mg
			if td.ContentType != ContentTypeMixed {
				td.ContentType = ContentTypeElementOnly
			}
		case isXSDElement(ce, "choice"):
			mg, err := c.parseModelGroup(ce, CompositorChoice)
			if err != nil {
				return nil, err
			}
			td.ContentModel = mg
			if td.ContentType != ContentTypeMixed {
				td.ContentType = ContentTypeElementOnly
			}
		case isXSDElement(ce, "all"):
			mg, err := c.parseModelGroup(ce, CompositorAll)
			if err != nil {
				return nil, err
			}
			td.ContentModel = mg
			if td.ContentType != ContentTypeMixed {
				td.ContentType = ContentTypeElementOnly
			}
		case isXSDElement(ce, "group"):
			ref := getAttr(ce, "ref")
			if ref != "" {
				placeholder := &ModelGroup{MinOccurs: 1, MaxOccurs: 1}
				if v := getAttr(ce, "minOccurs"); v != "" {
					placeholder.MinOccurs = parseOccurs(v, 1)
				}
				if v := getAttr(ce, "maxOccurs"); v != "" {
					placeholder.MaxOccurs = parseOccurs(v, 1)
				}
				qn := c.resolveQName(ce, ref)
				c.groupRefs[placeholder] = qn
				td.ContentModel = placeholder
				if td.ContentType != ContentTypeMixed {
					td.ContentType = ContentTypeElementOnly
				}
			}
		case isXSDElement(ce, "complexContent"):
			if err := c.parseComplexContent(ce, td); err != nil {
				return nil, err
			}
		case isXSDElement(ce, "simpleContent"):
			c.parseSimpleContent(ce, td)
		case isXSDElement(ce, "attribute"):
			au := c.parseAttributeUse(ce)
			td.Attributes = append(td.Attributes, au)
		case isXSDElement(ce, "attributeGroup"):
			if ref := getAttr(ce, "ref"); ref != "" {
				qn := c.resolveQName(ce, ref)
				c.attrGroupRefs[td] = append(c.attrGroupRefs[td], qn)
			}
		case isXSDElement(ce, "anyAttribute"):
			td.AnyAttribute = c.parseAnyAttribute(ce)
		}
	}

	// If no content model and not mixed, ContentTypeEmpty is the default (no children).
	// Attribute declarations do not change the content type.

	return td, nil
}

func (c *compiler) parseComplexContent(elem *helium.Element, td *TypeDef) error {
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce := child.(*helium.Element)
		switch {
		case isXSDElement(ce, "restriction"):
			return c.parseRestriction(ce, td)
		case isXSDElement(ce, "extension"):
			return c.parseExtension(ce, td)
		}
	}
	return nil
}

func (c *compiler) parseRestriction(elem *helium.Element, td *TypeDef) error {
	td.Derivation = DerivationRestriction
	baseRef := getAttr(elem, "base")
	if baseRef != "" {
		qn := c.resolveQName(elem, baseRef)
		c.typeRefs[td] = qn
	}

	// Parse child model groups and attributes.
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce := child.(*helium.Element)
		switch {
		case isXSDElement(ce, "sequence"):
			mg, err := c.parseModelGroup(ce, CompositorSequence)
			if err != nil {
				return err
			}
			td.ContentModel = mg
			if td.ContentType != ContentTypeMixed {
				td.ContentType = ContentTypeElementOnly
			}
		case isXSDElement(ce, "choice"):
			mg, err := c.parseModelGroup(ce, CompositorChoice)
			if err != nil {
				return err
			}
			td.ContentModel = mg
			if td.ContentType != ContentTypeMixed {
				td.ContentType = ContentTypeElementOnly
			}
		case isXSDElement(ce, "all"):
			mg, err := c.parseModelGroup(ce, CompositorAll)
			if err != nil {
				return err
			}
			td.ContentModel = mg
			if td.ContentType != ContentTypeMixed {
				td.ContentType = ContentTypeElementOnly
			}
		case isXSDElement(ce, "attribute"):
			au := c.parseAttributeUse(ce)
			td.Attributes = append(td.Attributes, au)
		case isXSDElement(ce, "attributeGroup"):
			if ref := getAttr(ce, "ref"); ref != "" {
				qn := c.resolveQName(ce, ref)
				c.attrGroupRefs[td] = append(c.attrGroupRefs[td], qn)
			}
		case isXSDElement(ce, "anyAttribute"):
			td.AnyAttribute = c.parseAnyAttribute(ce)
		}
	}
	return nil
}

func (c *compiler) parseExtension(elem *helium.Element, td *TypeDef) error {
	td.Derivation = DerivationExtension
	baseRef := getAttr(elem, "base")
	if baseRef != "" {
		qn := c.resolveQName(elem, baseRef)
		c.typeRefs[td] = qn
	}
	// Parse child content model (if any).
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce := child.(*helium.Element)
		switch {
		case isXSDElement(ce, "sequence"):
			mg, err := c.parseModelGroup(ce, CompositorSequence)
			if err != nil {
				return err
			}
			td.ContentModel = mg
			if td.ContentType != ContentTypeMixed {
				td.ContentType = ContentTypeElementOnly
			}
		case isXSDElement(ce, "choice"):
			mg, err := c.parseModelGroup(ce, CompositorChoice)
			if err != nil {
				return err
			}
			td.ContentModel = mg
			if td.ContentType != ContentTypeMixed {
				td.ContentType = ContentTypeElementOnly
			}
		case isXSDElement(ce, "all"):
			mg, err := c.parseModelGroup(ce, CompositorAll)
			if err != nil {
				return err
			}
			td.ContentModel = mg
			if td.ContentType != ContentTypeMixed {
				td.ContentType = ContentTypeElementOnly
			}
		case isXSDElement(ce, "attribute"):
			if getAttr(ce, "use") == "prohibited" {
				if c.filename != "" {
					c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaParserWarning(c.filename, ce.Line(), ce.LocalName(), "attribute",
						"Skipping attribute use prohibition, since it is pointless when extending a type."), helium.ErrorLevelWarning))
				}
				continue
			}
			au := c.parseAttributeUse(ce)
			td.Attributes = append(td.Attributes, au)
		case isXSDElement(ce, "attributeGroup"):
			if ref := getAttr(ce, "ref"); ref != "" {
				qn := c.resolveQName(ce, ref)
				c.attrGroupRefs[td] = append(c.attrGroupRefs[td], qn)
			}
		case isXSDElement(ce, "anyAttribute"):
			td.AnyAttribute = c.parseAnyAttribute(ce)
		}
	}
	return nil
}

func (c *compiler) parseSimpleContent(elem *helium.Element, td *TypeDef) {
	td.ContentType = ContentTypeSimple
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce := child.(*helium.Element)
		switch {
		case isXSDElement(ce, "extension"):
			baseRef := getAttr(ce, "base")
			if baseRef != "" {
				qn := c.resolveQName(ce, baseRef)
				c.typeRefs[td] = qn
			}
			td.Derivation = DerivationExtension
			c.parseSimpleContentChildren(ce, td)
		case isXSDElement(ce, "restriction"):
			baseRef := getAttr(ce, "base")
			if baseRef != "" {
				qn := c.resolveQName(ce, baseRef)
				c.typeRefs[td] = qn
			}
			td.Derivation = DerivationRestriction
			c.parseSimpleContentChildren(ce, td)
		}
	}
}

// parseSimpleContentChildren parses attribute/attributeGroup children within
// a simpleContent extension or restriction element.
func (c *compiler) parseSimpleContentChildren(derivation *helium.Element, td *TypeDef) {
	for child := derivation.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Type() != helium.ElementNode {
			continue
		}
		ae := child.(*helium.Element)
		switch {
		case isXSDElement(ae, "attribute"):
			au := c.parseAttributeUse(ae)
			td.Attributes = append(td.Attributes, au)
		case isXSDElement(ae, "attributeGroup"):
			if ref := getAttr(ae, "ref"); ref != "" {
				qn := c.resolveQName(ae, ref)
				c.attrGroupRefs[td] = append(c.attrGroupRefs[td], qn)
			}
		case isXSDElement(ae, "anyAttribute"):
			td.AnyAttribute = c.parseAnyAttribute(ae)
		}
	}
}

func (c *compiler) parseSimpleType(elem *helium.Element) (*TypeDef, error) {
	td := &TypeDef{
		ContentType: ContentTypeSimple,
	}

	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce := child.(*helium.Element)
		switch {
		case isXSDElement(ce, "restriction"):
			baseRef := getAttr(ce, "base")
			if baseRef != "" {
				qn := c.resolveQName(ce, baseRef)
				c.typeRefs[td] = qn
			} else {
				// Look for inline <simpleType> child as the base type.
				for gc := ce.FirstChild(); gc != nil; gc = gc.NextSibling() {
					if gc.Type() != helium.ElementNode {
						continue
					}
					gce := gc.(*helium.Element)
					if isXSDElement(gce, "simpleType") {
						baseTD, err := c.parseSimpleType(gce)
						if err != nil {
							return nil, err
						}
						td.BaseType = baseTD
						break
					}
				}
			}
			td.Derivation = DerivationRestriction
			td.Facets = c.parseFacets(ce)
		case isXSDElement(ce, "list"):
			td.Variety = TypeVarietyList
			itemRef := getAttr(ce, "itemType")
			if itemRef != "" {
				qn := c.resolveQName(ce, itemRef)
				c.itemTypeRefs[td] = qn
			} else {
				// Look for inline <simpleType> child as the item type.
				for gc := ce.FirstChild(); gc != nil; gc = gc.NextSibling() {
					if gc.Type() != helium.ElementNode {
						continue
					}
					gce := gc.(*helium.Element)
					if isXSDElement(gce, "simpleType") {
						itemTD, err := c.parseSimpleType(gce)
						if err != nil {
							return nil, err
						}
						td.ItemType = itemTD
						break
					}
				}
			}
		case isXSDElement(ce, "union"):
			td.Variety = TypeVarietyUnion
			// Parse memberTypes attribute (space-separated QNames).
			if memberTypesAttr := getAttr(ce, "memberTypes"); memberTypesAttr != "" {
				for _, ref := range strings.Fields(memberTypesAttr) {
					qn := c.resolveQName(ce, ref)
					c.unionMemberRefs = append(c.unionMemberRefs, unionMemberRef{owner: td, name: qn})
				}
			}
			// Parse inline <simpleType> children.
			for gc := ce.FirstChild(); gc != nil; gc = gc.NextSibling() {
				if gc.Type() != helium.ElementNode {
					continue
				}
				gce := gc.(*helium.Element)
				if isXSDElement(gce, "simpleType") {
					memberTD, err := c.parseSimpleType(gce)
					if err != nil {
						return nil, err
					}
					td.MemberTypes = append(td.MemberTypes, memberTD)
				}
			}
		}
	}

	return td, nil
}

// parseFacets extracts facet constraints from an xs:restriction element.
func (c *compiler) parseFacets(restriction *helium.Element) *FacetSet {
	var fs *FacetSet

	for child := restriction.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce := child.(*helium.Element)
		if ce.URI() != xsdNS {
			continue
		}
		val := getAttr(ce, "value")

		switch ce.LocalName() {
		case "enumeration":
			if fs == nil {
				fs = &FacetSet{}
			}
			fs.Enumeration = append(fs.Enumeration, val)
			nsCtx := collectNSContext(ce)
			nsCopy := make(map[string]string, len(nsCtx))
			for prefix, uri := range nsCtx {
				nsCopy[prefix] = uri
			}
			fs.EnumerationNS = append(fs.EnumerationNS, nsCopy)
		case "minInclusive":
			if fs == nil {
				fs = &FacetSet{}
			}
			fs.MinInclusive = &val
		case "maxInclusive":
			if fs == nil {
				fs = &FacetSet{}
			}
			fs.MaxInclusive = &val
		case "minExclusive":
			if fs == nil {
				fs = &FacetSet{}
			}
			fs.MinExclusive = &val
		case "maxExclusive":
			if fs == nil {
				fs = &FacetSet{}
			}
			fs.MaxExclusive = &val
		case "totalDigits":
			if fs == nil {
				fs = &FacetSet{}
			}
			n := parseOccurs(val, 0)
			fs.TotalDigits = &n
		case "length":
			if fs == nil {
				fs = &FacetSet{}
			}
			n := parseOccurs(val, 0)
			fs.Length = &n
		case "minLength":
			if fs == nil {
				fs = &FacetSet{}
			}
			n := parseOccurs(val, 0)
			fs.MinLength = &n
		case "maxLength":
			if fs == nil {
				fs = &FacetSet{}
			}
			n := parseOccurs(val, 0)
			fs.MaxLength = &n
		case "fractionDigits":
			if fs == nil {
				fs = &FacetSet{}
			}
			n := parseOccurs(val, 0)
			fs.FractionDigits = &n
		case "whiteSpace":
			if fs == nil {
				fs = &FacetSet{}
			}
			fs.WhiteSpace = &val
		case "pattern":
			if fs == nil {
				fs = &FacetSet{}
			}
			fs.Pattern = &val
		}
	}

	return fs
}
