package xsd

import (
	"context"
	"fmt"
	"maps"
	"strings"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
)

func (c *compiler) parseNamedComplexType(ctx context.Context, elem *helium.Element) error {
	name := getAttr(elem, attrName)
	if name == "" {
		return fmt.Errorf("xsd: named complexType missing name")
	}

	td, err := c.parseComplexType(ctx, elem)
	if err != nil {
		return err
	}
	td.Name = QName{Local: name, NS: c.schema.targetNamespace}
	td.Abstract = getAttr(elem, attrAbstract) == attrValTrue

	// Parse final attribute with schema default.
	if hasAttr(elem, attrFinal) {
		td.Final = parseElemFinalFlags(getAttr(elem, attrFinal))
		td.FinalSet = true
	} else {
		td.Final = c.schema.finalDefault & (FinalExtension | FinalRestriction)
	}

	c.typeDefSources[td] = typeDefSource{line: elem.Line(), isLocal: false}
	c.schema.types[td.Name] = td
	return nil
}

func (c *compiler) parseNamedSimpleType(ctx context.Context, elem *helium.Element) error {
	name := getAttr(elem, attrName)
	if name == "" {
		return fmt.Errorf("xsd: named simpleType missing name")
	}

	td, err := c.parseSimpleType(ctx, elem)
	if err != nil {
		return err
	}
	td.Name = QName{Local: name, NS: c.schema.targetNamespace}

	// Parse final attribute with schema default.
	if hasAttr(elem, attrFinal) {
		td.Final = parseFinalFlags(getAttr(elem, attrFinal))
		td.FinalSet = true
	} else {
		td.Final = c.schema.finalDefault & (FinalRestriction | FinalList | FinalUnion)
	}

	c.typeDefSources[td] = typeDefSource{line: elem.Line(), isLocal: false}
	c.schema.types[td.Name] = td
	return nil
}

func (c *compiler) parseComplexType(ctx context.Context, elem *helium.Element) (*TypeDef, error) {
	td := &TypeDef{}
	c.typeDefSources[td] = typeDefSource{line: elem.Line(), isLocal: true}

	mixed := getAttr(elem, "mixed")
	if mixed == attrValTrue {
		td.ContentType = ContentTypeMixed
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
		case isXSDElement(ce, elemSequence):
			mg, err := c.parseModelGroup(ctx, ce, CompositorSequence)
			if err != nil {
				return nil, err
			}
			td.ContentModel = mg
			if td.ContentType != ContentTypeMixed {
				td.ContentType = ContentTypeElementOnly
			}
		case isXSDElement(ce, elemChoice):
			mg, err := c.parseModelGroup(ctx, ce, CompositorChoice)
			if err != nil {
				return nil, err
			}
			td.ContentModel = mg
			if td.ContentType != ContentTypeMixed {
				td.ContentType = ContentTypeElementOnly
			}
		case isXSDElement(ce, elemAll):
			mg, err := c.parseModelGroup(ctx, ce, CompositorAll)
			if err != nil {
				return nil, err
			}
			td.ContentModel = mg
			if td.ContentType != ContentTypeMixed {
				td.ContentType = ContentTypeElementOnly
			}
		case isXSDElement(ce, elemGroup):
			ref := getAttr(ce, attrRef)
			if ref != "" {
				placeholder := &ModelGroup{MinOccurs: 1, MaxOccurs: 1}
				if v := getAttr(ce, "minOccurs"); v != "" {
					placeholder.MinOccurs = parseOccurs(v, 1)
				}
				if v := getAttr(ce, "maxOccurs"); v != "" {
					placeholder.MaxOccurs = parseOccurs(v, 1)
				}
				qn := c.resolveQName(ctx, ce, ref)
				c.groupRefs[placeholder] = qn
				td.ContentModel = placeholder
				if td.ContentType != ContentTypeMixed {
					td.ContentType = ContentTypeElementOnly
				}
			}
		case isXSDElement(ce, elemComplexContent):
			if err := c.parseComplexContent(ctx, ce, td); err != nil {
				return nil, err
			}
		case isXSDElement(ce, elemSimpleContent):
			c.parseSimpleContent(ctx, ce, td)
		case isXSDElement(ce, elemAttribute):
			au := c.parseAttributeUse(ctx, ce)
			td.Attributes = append(td.Attributes, au)
		case isXSDElement(ce, elemAttributeGroup):
			if ref := getAttr(ce, attrRef); ref != "" {
				qn := c.resolveQName(ctx, ce, ref)
				c.attrGroupRefs[td] = append(c.attrGroupRefs[td], qn)
			}
		case isXSDElement(ce, elemAnyAttribute):
			td.AnyAttribute = c.parseAnyAttribute(ctx, ce)
		}
	}

	// If no content model and not mixed, ContentTypeEmpty is the default (no children).
	// Attribute declarations do not change the content type.

	return td, nil
}

func (c *compiler) parseComplexContent(ctx context.Context, elem *helium.Element, td *TypeDef) error {
	for child := range helium.Children(elem) {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		switch {
		case isXSDElement(ce, elemRestriction):
			return c.parseRestriction(ctx, ce, td)
		case isXSDElement(ce, elemExtension):
			return c.parseExtension(ctx, ce, td)
		}
	}
	return nil
}

func (c *compiler) parseRestriction(ctx context.Context, elem *helium.Element, td *TypeDef) error {
	td.Derivation = DerivationRestriction
	baseRef := getAttr(elem, attrBase)
	if baseRef != "" {
		qn := c.resolveQName(ctx, elem, baseRef)
		c.typeRefs[td] = qn
	}

	// Parse child model groups and attributes.
	for child := range helium.Children(elem) {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		switch {
		case isXSDElement(ce, elemSequence):
			mg, err := c.parseModelGroup(ctx, ce, CompositorSequence)
			if err != nil {
				return err
			}
			td.ContentModel = mg
			if td.ContentType != ContentTypeMixed {
				td.ContentType = ContentTypeElementOnly
			}
		case isXSDElement(ce, elemChoice):
			mg, err := c.parseModelGroup(ctx, ce, CompositorChoice)
			if err != nil {
				return err
			}
			td.ContentModel = mg
			if td.ContentType != ContentTypeMixed {
				td.ContentType = ContentTypeElementOnly
			}
		case isXSDElement(ce, elemAll):
			mg, err := c.parseModelGroup(ctx, ce, CompositorAll)
			if err != nil {
				return err
			}
			td.ContentModel = mg
			if td.ContentType != ContentTypeMixed {
				td.ContentType = ContentTypeElementOnly
			}
		case isXSDElement(ce, elemAttribute):
			au := c.parseAttributeUse(ctx, ce)
			td.Attributes = append(td.Attributes, au)
		case isXSDElement(ce, elemAttributeGroup):
			if ref := getAttr(ce, attrRef); ref != "" {
				qn := c.resolveQName(ctx, ce, ref)
				c.attrGroupRefs[td] = append(c.attrGroupRefs[td], qn)
			}
		case isXSDElement(ce, elemAnyAttribute):
			td.AnyAttribute = c.parseAnyAttribute(ctx, ce)
		}
	}
	return nil
}

func (c *compiler) parseExtension(ctx context.Context, elem *helium.Element, td *TypeDef) error {
	td.Derivation = DerivationExtension
	baseRef := getAttr(elem, attrBase)
	if baseRef != "" {
		qn := c.resolveQName(ctx, elem, baseRef)
		c.typeRefs[td] = qn
	}
	// Parse child content model (if any).
	for child := range helium.Children(elem) {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		switch {
		case isXSDElement(ce, elemSequence):
			mg, err := c.parseModelGroup(ctx, ce, CompositorSequence)
			if err != nil {
				return err
			}
			td.ContentModel = mg
			if td.ContentType != ContentTypeMixed {
				td.ContentType = ContentTypeElementOnly
			}
		case isXSDElement(ce, elemChoice):
			mg, err := c.parseModelGroup(ctx, ce, CompositorChoice)
			if err != nil {
				return err
			}
			td.ContentModel = mg
			if td.ContentType != ContentTypeMixed {
				td.ContentType = ContentTypeElementOnly
			}
		case isXSDElement(ce, elemAll):
			mg, err := c.parseModelGroup(ctx, ce, CompositorAll)
			if err != nil {
				return err
			}
			td.ContentModel = mg
			if td.ContentType != ContentTypeMixed {
				td.ContentType = ContentTypeElementOnly
			}
		case isXSDElement(ce, elemAttribute):
			if getAttr(ce, attrUse) == attrValProhibited {
				if c.filename != "" {
					c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserWarning(c.filename, ce.Line(), ce.LocalName(), "attribute",
						"Skipping attribute use prohibition, since it is pointless when extending a type."), helium.ErrorLevelWarning))
				}
				continue
			}
			au := c.parseAttributeUse(ctx, ce)
			td.Attributes = append(td.Attributes, au)
		case isXSDElement(ce, elemAttributeGroup):
			if ref := getAttr(ce, attrRef); ref != "" {
				qn := c.resolveQName(ctx, ce, ref)
				c.attrGroupRefs[td] = append(c.attrGroupRefs[td], qn)
			}
		case isXSDElement(ce, elemAnyAttribute):
			td.AnyAttribute = c.parseAnyAttribute(ctx, ce)
		}
	}
	return nil
}

func (c *compiler) parseSimpleContent(ctx context.Context, elem *helium.Element, td *TypeDef) {
	td.ContentType = ContentTypeSimple
	for child := range helium.Children(elem) {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		switch {
		case isXSDElement(ce, elemExtension):
			baseRef := getAttr(ce, attrBase)
			if baseRef != "" {
				qn := c.resolveQName(ctx, ce, baseRef)
				c.typeRefs[td] = qn
			}
			td.Derivation = DerivationExtension
			c.parseSimpleContentChildren(ctx, ce, td)
		case isXSDElement(ce, elemRestriction):
			baseRef := getAttr(ce, attrBase)
			if baseRef != "" {
				qn := c.resolveQName(ctx, ce, baseRef)
				c.typeRefs[td] = qn
			}
			td.Derivation = DerivationRestriction
			c.parseSimpleContentChildren(ctx, ce, td)
		}
	}
}

// parseSimpleContentChildren parses attribute/attributeGroup children within
// a simpleContent extension or restriction element.
func (c *compiler) parseSimpleContentChildren(ctx context.Context, derivation *helium.Element, td *TypeDef) {
	for child := range helium.Children(derivation) {
		if child.Type() != helium.ElementNode {
			continue
		}
		ae, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		switch {
		case isXSDElement(ae, elemAttribute):
			au := c.parseAttributeUse(ctx, ae)
			td.Attributes = append(td.Attributes, au)
		case isXSDElement(ae, elemAttributeGroup):
			if ref := getAttr(ae, attrRef); ref != "" {
				qn := c.resolveQName(ctx, ae, ref)
				c.attrGroupRefs[td] = append(c.attrGroupRefs[td], qn)
			}
		case isXSDElement(ae, elemAnyAttribute):
			td.AnyAttribute = c.parseAnyAttribute(ctx, ae)
		}
	}
}

func (c *compiler) parseSimpleType(ctx context.Context, elem *helium.Element) (*TypeDef, error) {
	td := &TypeDef{
		ContentType: ContentTypeSimple,
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
		case isXSDElement(ce, elemRestriction):
			baseRef := getAttr(ce, attrBase)
			if baseRef != "" {
				qn := c.resolveQName(ctx, ce, baseRef)
				c.typeRefs[td] = qn
			} else {
				// Look for inline <simpleType> child as the base type.
				for gc := range helium.Children(ce) {
					if gc.Type() != helium.ElementNode {
						continue
					}
					gce, ok := helium.AsNode[*helium.Element](gc)
					if !ok {
						continue
					}
					if isXSDElement(gce, elemSimpleType) {
						baseTD, err := c.parseSimpleType(ctx, gce)
						if err != nil {
							return nil, err
						}
						td.BaseType = baseTD
						break
					}
				}
			}
			td.Derivation = DerivationRestriction
			td.Facets = c.parseFacets(ctx, ce)
		case isXSDElement(ce, elemList):
			td.Variety = TypeVarietyList
			itemRef := getAttr(ce, attrItemType)
			if itemRef != "" {
				qn := c.resolveQName(ctx, ce, itemRef)
				c.itemTypeRefs[td] = qn
			} else {
				// Look for inline <simpleType> child as the item type.
				for gc := range helium.Children(ce) {
					if gc.Type() != helium.ElementNode {
						continue
					}
					gce, ok := helium.AsNode[*helium.Element](gc)
					if !ok {
						continue
					}
					if isXSDElement(gce, elemSimpleType) {
						itemTD, err := c.parseSimpleType(ctx, gce)
						if err != nil {
							return nil, err
						}
						td.ItemType = itemTD
						break
					}
				}
			}
		case isXSDElement(ce, elemUnion):
			td.Variety = TypeVarietyUnion
			// Parse memberTypes attribute (space-separated QNames).
			if memberTypesAttr := getAttr(ce, attrMemberTypes); memberTypesAttr != "" {
				for ref := range strings.FieldsSeq(memberTypesAttr) {
					qn := c.resolveQName(ctx, ce, ref)
					c.unionMemberRefs = append(c.unionMemberRefs, unionMemberRef{owner: td, name: qn})
				}
			}
			// Parse inline <simpleType> children.
			for gc := range helium.Children(ce) {
				if gc.Type() != helium.ElementNode {
					continue
				}
				gce, ok := helium.AsNode[*helium.Element](gc)
				if !ok {
					continue
				}
				if isXSDElement(gce, elemSimpleType) {
					memberTD, err := c.parseSimpleType(ctx, gce)
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
func (c *compiler) parseFacets(_ context.Context, restriction *helium.Element) *FacetSet {
	var fs *FacetSet

	for child := range helium.Children(restriction) {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		if ce.URI() != lexicon.NamespaceXSD {
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
			maps.Copy(nsCopy, nsCtx)
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
