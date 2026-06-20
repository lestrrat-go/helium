package xsd

import (
	"context"
	"fmt"
	"maps"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/xsdregex"
)

func (c *compiler) parseNamedComplexType(ctx context.Context, elem *helium.Element) error {
	name := getAttr(elem, attrName)
	if name == "" {
		return fmt.Errorf("xsd: named complexType missing name")
	}

	qn := QName{Local: name, NS: c.schema.targetNamespace}

	// Check for a duplicate global type BEFORE parsing the body, so a rejected
	// component leaves no stale refs/state behind that would produce unrelated
	// follow-on errors. An xs:redefine override that was validated and consumed
	// by the redefine loop is pre-authorized and skips the report.
	if _, exists := c.schema.types[qn]; exists && !c.redefineConsumed(redefineKindComplexType, qn) {
		c.reportDuplicateComponent(ctx, elem, "complexType", "A global type definition", qn)
		return nil
	}

	td, err := c.parseComplexType(ctx, elem)
	if err != nil {
		return err
	}
	td.Name = qn
	td.Abstract = c.readBooleanAttr(ctx, elem, attrAbstract)

	// Parse final attribute with schema default.
	if hasAttr(elem, attrFinal) {
		td.Final = parseElemFinalFlags(getAttr(elem, attrFinal))
		td.FinalSet = true
	} else {
		td.Final = c.schema.finalDefault & (FinalExtension | FinalRestriction)
	}

	c.recordTypeDefSource(td, elem.Line(), false)
	c.typeKinds[td.Name] = redefineKindComplexType
	c.schema.types[td.Name] = td
	return nil
}

func (c *compiler) parseNamedSimpleType(ctx context.Context, elem *helium.Element) error {
	name := getAttr(elem, attrName)
	if name == "" {
		return fmt.Errorf("xsd: named simpleType missing name")
	}

	qn := QName{Local: name, NS: c.schema.targetNamespace}

	// Check for a duplicate global type BEFORE parsing the body, so a rejected
	// component leaves no stale refs/state behind that would produce unrelated
	// follow-on errors. An xs:redefine override that was validated and consumed
	// by the redefine loop is pre-authorized and skips the report.
	if _, exists := c.schema.types[qn]; exists && !c.redefineConsumed(redefineKindSimpleType, qn) {
		c.reportDuplicateComponent(ctx, elem, "simpleType", "A global type definition", qn)
		return nil
	}

	td, err := c.parseSimpleType(ctx, elem)
	if err != nil {
		return err
	}
	td.Name = qn

	// Parse final attribute with schema default.
	if hasAttr(elem, attrFinal) {
		td.Final = parseFinalFlags(getAttr(elem, attrFinal))
		td.FinalSet = true
	} else {
		td.Final = c.schema.finalDefault & (FinalRestriction | FinalList | FinalUnion)
	}

	c.recordTypeDefSource(td, elem.Line(), false)
	c.typeKinds[td.Name] = redefineKindSimpleType
	c.schema.types[td.Name] = td
	return nil
}

func (c *compiler) parseComplexType(ctx context.Context, elem *helium.Element) (*TypeDef, error) {
	td := &TypeDef{}
	c.recordTypeDefSource(td, elem.Line(), true)

	if c.readBooleanAttr(ctx, elem, "mixed") {
		td.ContentType = ContentTypeMixed
	}

	// XSD 3.4.2: the content of an xs:complexType is one of:
	//   - a single simpleContent OR complexContent, OR
	//   - an optional model-group particle (sequence|choice|all|group)
	//     followed by attribute uses.
	// These two forms are mutually exclusive, and at most one model-group
	// particle / content-model wrapper may appear. Track what we have seen so
	// a schema that supplies a second model group (silently overwriting the
	// first) or mixes a particle with simple/complexContent is rejected rather
	// than accepting the last-seen child.
	var contentModelChild string   // local name of the first model-group particle seen
	var contentWrapperChild string // "simpleContent" or "complexContent" if seen
	var directAttrChild string     // local name of the first direct attribute/attributeGroup/anyAttribute seen

	reportExtraContent := func(ce *helium.Element, what string) {
		if c.filename == "" {
			return
		}
		c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.diagSource(), ce.Line(),
			elem.LocalName(), componentLocalComplexType, what), helium.ErrorLevelFatal))
		c.errorCount++
	}

	for child := range helium.Children(elem) {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}

		// Guard model-group particles: at most one, and not alongside a
		// simple/complexContent wrapper.
		isModelGroup := isXSDElement(ce, elemSequence) || isXSDElement(ce, elemChoice) ||
			isXSDElement(ce, elemAll) || isXSDElement(ce, elemGroup)
		if isModelGroup {
			if contentWrapperChild != "" {
				reportExtraContent(ce, fmt.Sprintf("The content model particle '%s' is not allowed together with '%s'.", ce.LocalName(), contentWrapperChild))
				continue
			}
			if contentModelChild != "" {
				reportExtraContent(ce, fmt.Sprintf("A complex type definition must not have more than one content model particle (found '%s' after '%s').", ce.LocalName(), contentModelChild))
				continue
			}
			contentModelChild = ce.LocalName()
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
				c.validateOccursAttrs(ctx, ce)
				placeholderMin, placeholderMax := parseParticleOccurs(ce)
				placeholder := &ModelGroup{MinOccurs: placeholderMin, MaxOccurs: placeholderMax}
				qn := c.resolveQName(ctx, ce, ref)
				c.groupRefs[placeholder] = qn
				// Direct reference: this group ref is the sole top-level particle
				// of the complex type's content, so a resolved 'all' model group
				// is permitted here (subject to maxOccurs == 1).
				c.groupRefSources[placeholder] = groupRefSource{
					line:         ce.Line(),
					local:        ce.LocalName(),
					nested:       false,
					maxOccursRaw: getAttr(ce, attrMaxOccurs),
				}
				td.ContentModel = placeholder
				if td.ContentType != ContentTypeMixed {
					td.ContentType = ContentTypeElementOnly
				}
			}
		case isXSDElement(ce, elemComplexContent):
			if contentModelChild != "" {
				reportExtraContent(ce, fmt.Sprintf("The wrapper '%s' is not allowed together with the content model particle '%s'.", ce.LocalName(), contentModelChild))
				continue
			}
			if directAttrChild != "" {
				reportExtraContent(ce, fmt.Sprintf("The wrapper '%s' is not allowed together with the attribute declaration '%s'; attributes must be declared inside the wrapper's restriction or extension.", ce.LocalName(), directAttrChild))
				continue
			}
			if contentWrapperChild != "" {
				reportExtraContent(ce, fmt.Sprintf("A complex type definition must not have more than one of 'simpleContent' or 'complexContent' (found '%s' after '%s').", ce.LocalName(), contentWrapperChild))
				continue
			}
			contentWrapperChild = ce.LocalName()
			if err := c.parseComplexContent(ctx, ce, td); err != nil {
				return nil, err
			}
		case isXSDElement(ce, elemSimpleContent):
			if contentModelChild != "" {
				reportExtraContent(ce, fmt.Sprintf("The wrapper '%s' is not allowed together with the content model particle '%s'.", ce.LocalName(), contentModelChild))
				continue
			}
			if directAttrChild != "" {
				reportExtraContent(ce, fmt.Sprintf("The wrapper '%s' is not allowed together with the attribute declaration '%s'; attributes must be declared inside the wrapper's restriction or extension.", ce.LocalName(), directAttrChild))
				continue
			}
			if contentWrapperChild != "" {
				reportExtraContent(ce, fmt.Sprintf("A complex type definition must not have more than one of 'simpleContent' or 'complexContent' (found '%s' after '%s').", ce.LocalName(), contentWrapperChild))
				continue
			}
			contentWrapperChild = ce.LocalName()
			c.parseSimpleContent(ctx, ce, td)
		case isXSDElement(ce, elemAttribute):
			if contentWrapperChild != "" {
				reportExtraContent(ce, fmt.Sprintf("The attribute declaration '%s' is not allowed together with '%s'; attributes must be declared inside the wrapper's restriction or extension.", ce.LocalName(), contentWrapperChild))
				continue
			}
			if directAttrChild == "" {
				directAttrChild = ce.LocalName()
			}
			au := c.parseAttributeUse(ctx, ce)
			td.Attributes = append(td.Attributes, au)
		case isXSDElement(ce, elemAttributeGroup):
			if contentWrapperChild != "" {
				reportExtraContent(ce, fmt.Sprintf("The attribute declaration '%s' is not allowed together with '%s'; attributes must be declared inside the wrapper's restriction or extension.", ce.LocalName(), contentWrapperChild))
				continue
			}
			if directAttrChild == "" {
				directAttrChild = ce.LocalName()
			}
			if ref := getAttr(ce, attrRef); ref != "" {
				qn := c.resolveQName(ctx, ce, ref)
				c.attrGroupRefs[td] = append(c.attrGroupRefs[td], qn)
			}
		case isXSDElement(ce, elemAnyAttribute):
			if contentWrapperChild != "" {
				reportExtraContent(ce, fmt.Sprintf("The attribute wildcard '%s' is not allowed together with '%s'; the wildcard must be declared inside the wrapper's restriction or extension.", ce.LocalName(), contentWrapperChild))
				continue
			}
			if directAttrChild == "" {
				directAttrChild = ce.LocalName()
			}
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
		c.markChameleonEligible(td, elem, baseRef)
	}

	// At most one model-group particle (sequence|choice|all|group ref) is
	// permitted; a second one would silently overwrite td.ContentModel. Track
	// the first seen.
	var contentModelChild string

	// Parse child model groups and attributes.
	for child := range helium.Children(elem) {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		if isXSDElement(ce, elemSequence) || isXSDElement(ce, elemChoice) || isXSDElement(ce, elemAll) || isXSDElement(ce, elemGroup) {
			if contentModelChild != "" {
				if c.filename != "" {
					c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.diagSource(), ce.Line(),
						elem.LocalName(), componentLocalComplexType,
						fmt.Sprintf("A complex type definition must not have more than one content model particle (found '%s' after '%s').", ce.LocalName(), contentModelChild)), helium.ErrorLevelFatal))
					c.errorCount++
				}
				continue
			}
			contentModelChild = ce.LocalName()
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
		case isXSDElement(ce, elemGroup):
			ref := getAttr(ce, attrRef)
			if ref != "" {
				c.validateOccursAttrs(ctx, ce)
				placeholderMin, placeholderMax := parseParticleOccurs(ce)
				placeholder := &ModelGroup{MinOccurs: placeholderMin, MaxOccurs: placeholderMax}
				qn := c.resolveQName(ctx, ce, ref)
				c.groupRefs[placeholder] = qn
				// Direct reference: this group ref is the sole top-level particle
				// of the derived type's content, so a resolved 'all' model group
				// is permitted here (subject to maxOccurs == 1).
				c.groupRefSources[placeholder] = groupRefSource{
					line:         ce.Line(),
					local:        ce.LocalName(),
					nested:       false,
					maxOccursRaw: getAttr(ce, attrMaxOccurs),
				}
				td.ContentModel = placeholder
				if td.ContentType != ContentTypeMixed {
					td.ContentType = ContentTypeElementOnly
				}
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
		c.markChameleonEligible(td, elem, baseRef)
	}
	// At most one model-group particle (sequence|choice|all|group ref) is
	// permitted; a second one would silently overwrite td.ContentModel. Track
	// the first seen.
	var contentModelChild string

	// Parse child content model (if any).
	for child := range helium.Children(elem) {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		if isXSDElement(ce, elemSequence) || isXSDElement(ce, elemChoice) || isXSDElement(ce, elemAll) || isXSDElement(ce, elemGroup) {
			if contentModelChild != "" {
				if c.filename != "" {
					c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaComponentError(c.diagSource(), ce.Line(),
						elem.LocalName(), componentLocalComplexType,
						fmt.Sprintf("A complex type definition must not have more than one content model particle (found '%s' after '%s').", ce.LocalName(), contentModelChild)), helium.ErrorLevelFatal))
					c.errorCount++
				}
				continue
			}
			contentModelChild = ce.LocalName()
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
		case isXSDElement(ce, elemGroup):
			ref := getAttr(ce, attrRef)
			if ref != "" {
				c.validateOccursAttrs(ctx, ce)
				placeholderMin, placeholderMax := parseParticleOccurs(ce)
				placeholder := &ModelGroup{MinOccurs: placeholderMin, MaxOccurs: placeholderMax}
				qn := c.resolveQName(ctx, ce, ref)
				c.groupRefs[placeholder] = qn
				// Direct reference: this group ref is the sole top-level particle
				// of the derived type's content, so a resolved 'all' model group
				// is permitted here (subject to maxOccurs == 1).
				c.groupRefSources[placeholder] = groupRefSource{
					line:         ce.Line(),
					local:        ce.LocalName(),
					nested:       false,
					maxOccursRaw: getAttr(ce, attrMaxOccurs),
				}
				td.ContentModel = placeholder
				if td.ContentType != ContentTypeMixed {
					td.ContentType = ContentTypeElementOnly
				}
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
				c.markChameleonEligible(td, ce, baseRef)
			}
			td.Derivation = DerivationExtension
			c.parseSimpleContentChildren(ctx, ce, td)
		case isXSDElement(ce, elemRestriction):
			baseRef := getAttr(ce, attrBase)
			if baseRef != "" {
				qn := c.resolveQName(ctx, ce, baseRef)
				c.typeRefs[td] = qn
				c.markChameleonEligible(td, ce, baseRef)
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

	// Record source info for this type as a local (anonymous/inline) simple
	// type. parseNamedSimpleType overwrites this with isLocal:false after the
	// name is assigned. Recording it here ensures reportUnresolvedTypeRef can
	// fire for unresolved base/itemType/memberTypes references inside inline
	// simpleTypes, not just top-level named ones.
	c.recordTypeDefSource(td, elem.Line(), true)

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
				c.markChameleonEligible(td, ce, baseRef)
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
				c.markChameleonEligible(td, ce, itemRef)
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
				for _, ref := range splitSpace(memberTypesAttr) {
					qn := c.resolveQName(ctx, ce, ref)
					c.unionMemberRefs = append(c.unionMemberRefs, unionMemberRef{
						owner:             td,
						name:              qn,
						chameleonEligible: refChameleonEligible(ce, ref),
					})
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
func (c *compiler) parseFacets(ctx context.Context, restriction *helium.Element) *FacetSet {
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
			fs.MinInclusiveNS = captureFacetNS(ce)
		case "maxInclusive":
			if fs == nil {
				fs = &FacetSet{}
			}
			fs.MaxInclusive = &val
			fs.MaxInclusiveNS = captureFacetNS(ce)
		case "minExclusive":
			if fs == nil {
				fs = &FacetSet{}
			}
			fs.MinExclusive = &val
			fs.MinExclusiveNS = captureFacetNS(ce)
		case "maxExclusive":
			if fs == nil {
				fs = &FacetSet{}
			}
			fs.MaxExclusive = &val
			fs.MaxExclusiveNS = captureFacetNS(ce)
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
			// Multiple <xs:pattern> in the same restriction step are ORed.
			// Compile once here, index-aligned with Patterns. XSD regex
			// supports constructs Go's RE2 lacks: \i, \c, \p{Is...} blocks
			// (translated to RE2) and character-class subtraction / large
			// quantifiers (compiled with the regexp2 backtracking engine).
			// A pattern that is not a valid XSD regex is a schema error rather
			// than silently ignored; its compiledPatterns entry stays nil.
			fs.Patterns = append(fs.Patterns, val)
			re, rerr := xsdregex.Compile(val)
			if rerr != nil {
				c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserError(c.filename, ce.Line(),
					ce.LocalName(), "pattern",
					fmt.Sprintf("The value '%s' is not a valid regular expression: %s.", val, rerr)), helium.ErrorLevelFatal))
				c.errorCount++
			}
			fs.compiledPatterns = append(fs.compiledPatterns, re)
		}
	}

	return fs
}

// captureFacetNS records the in-scope namespace bindings at a single range-facet
// element so a namespace-sensitive bound value (e.g. a prefixed xs:QName) can be
// resolved later when that specific bound is validated against its base type.
// Each range facet captures its OWN context because sibling facets in the same
// <xs:restriction> may declare different prefixes; the bound must be resolved
// with the prefixes in scope at its own element, not a shared snapshot.
func captureFacetNS(ce *helium.Element) map[string]string {
	nsCtx := collectNSContext(ce)
	nsCopy := make(map[string]string, len(nsCtx))
	maps.Copy(nsCopy, nsCtx)
	return nsCopy
}
