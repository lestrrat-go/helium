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

	c.recordTypeDefSource(td, elem.Line(), false, elem.LocalName())
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
		// XSD 1.1 (spec bug 2074): "extension" is a valid member of a simple
		// type's {final}, so finalDefault="extension" reaches a simple type and
		// blocks a simpleContent extension of it. XSD 1.0 masks extension out.
		mask := FinalRestriction | FinalList | FinalUnion
		if c.version == Version11 {
			mask |= FinalExtension
		}
		td.Final = c.schema.finalDefault & mask
	}

	c.recordTypeDefSource(td, elem.Line(), false, elem.LocalName())
	c.typeKinds[td.Name] = redefineKindSimpleType
	c.schema.types[td.Name] = td
	return nil
}

func (c *compiler) recordAttrGroupRef(td *TypeDef, qn QName, src attrGroupRefUseSource) {
	c.attrGroupRefs[td] = append(c.attrGroupRefs[td], qn)
	c.attrGroupRefUseSources[td] = append(c.attrGroupRefUseSources[td], src)
}

func (c *compiler) parseComplexType(ctx context.Context, elem *helium.Element) (*TypeDef, error) {
	td := &TypeDef{IsComplex: true}
	c.recordTypeDefSource(td, elem.Line(), true, elem.LocalName())

	if c.readBooleanAttr(ctx, elem, "mixed") {
		td.ContentType = ContentTypeMixed
	}
	defaultAttrsApply := c.version == Version11
	if c.version == Version11 {
		if hasAttr(elem, attrDefaultAttrsApply) {
			defaultAttrsApply = c.readBooleanAttr(ctx, elem, attrDefaultAttrsApply)
		}
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
	var anyAttributeSeen bool      // whether an anyAttribute wildcard has been seen; it must be the optional final child
	var assertSeen bool            // whether an xs:assert (trailing region) has been seen; openContent must precede it
	var openContentSeen bool       // whether an xs:openContent has been seen; it is a sibling of the wrapper-free CHOICE branch

	reportExtraContent := func(ce *helium.Element, what string) {
		if c.filename == "" {
			return
		}
		c.schemaError(ctx, schemaComponentError(c.diagSource(), ce.Line(),
			elem.LocalName(), componentLocalComplexType, what))
	}

	for child := range helium.Children(elem) {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}

		// Guard model-group particles: at most one, before any attribute
		// declaration, and not alongside a simple/complexContent wrapper. XSD
		// 3.4.2 fixes the ordering: an optional leading model-group particle,
		// then attribute/attributeGroup uses, then an optional anyAttribute. A
		// model-group particle that appears AFTER an attribute declaration is
		// out of order and rejected.
		isModelGroup := isXSDElement(ce, elemSequence) || isXSDElement(ce, elemChoice) ||
			isXSDElement(ce, elemAll) || isXSDElement(ce, elemGroup)
		if isModelGroup {
			if contentWrapperChild != "" {
				reportExtraContent(ce, fmt.Sprintf("The content model particle '%s' is not allowed together with '%s'.", ce.LocalName(), contentWrapperChild))
				continue
			}
			if directAttrChild != "" {
				reportExtraContent(ce, fmt.Sprintf("The content model particle '%s' must appear before the attribute declaration '%s'.", ce.LocalName(), directAttrChild))
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
					source:       c.diagSource(),
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
			if openContentSeen {
				reportExtraContent(ce, fmt.Sprintf("The wrapper '%s' is not allowed together with 'openContent'.", ce.LocalName()))
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
			if openContentSeen {
				reportExtraContent(ce, fmt.Sprintf("The wrapper '%s' is not allowed together with 'openContent'.", ce.LocalName()))
				continue
			}
			contentWrapperChild = ce.LocalName()
			c.parseSimpleContent(ctx, ce, td)
		case isXSDElement(ce, elemAttribute):
			if contentWrapperChild != "" {
				reportExtraContent(ce, fmt.Sprintf("The attribute declaration '%s' is not allowed together with '%s'; attributes must be declared inside the wrapper's restriction or extension.", ce.LocalName(), contentWrapperChild))
				continue
			}
			if anyAttributeSeen {
				reportExtraContent(ce, fmt.Sprintf("The attribute declaration '%s' must appear before the attribute wildcard 'anyAttribute'.", ce.LocalName()))
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
			if anyAttributeSeen {
				reportExtraContent(ce, fmt.Sprintf("The attribute declaration '%s' must appear before the attribute wildcard 'anyAttribute'.", ce.LocalName()))
				continue
			}
			if directAttrChild == "" {
				directAttrChild = ce.LocalName()
			}
			if ref := getAttr(ce, attrRef); ref != "" {
				qn := c.resolveQName(ctx, ce, ref)
				c.recordAttrGroupRef(td, qn, attrGroupRefUseSource{
					line:      ce.Line(),
					elemLocal: ce.LocalName(),
					attr:      attrRef,
					source:    c.diagSource(),
				})
			}
		case isXSDElement(ce, elemAnyAttribute):
			if contentWrapperChild != "" {
				reportExtraContent(ce, fmt.Sprintf("The attribute wildcard '%s' is not allowed together with '%s'; the wildcard must be declared inside the wrapper's restriction or extension.", ce.LocalName(), contentWrapperChild))
				continue
			}
			if anyAttributeSeen {
				reportExtraContent(ce, fmt.Sprintf("A complex type definition must not have more than one attribute wildcard (found a second '%s').", ce.LocalName()))
				continue
			}
			if directAttrChild == "" {
				directAttrChild = ce.LocalName()
			}
			anyAttributeSeen = true
			td.AnyAttribute = c.parseAnyAttribute(ctx, ce)
		case isXSDElement(ce, elemAssert) && c.version == Version11:
			// XSD 1.1: xs:assert is the optional final content of a complex type,
			// after the attribute uses and anyAttribute wildcard.
			assertSeen = true
			if a := c.parseAssert(ctx, ce); a != nil {
				td.Assertions = append(td.Assertions, a)
			}
		case isXSDElement(ce, elemOpenContent) && c.version == Version11:
			// At most one openContent per complex type.
			if openContentSeen {
				reportExtraContent(ce, "A complex type definition must not have more than one 'openContent'.")
				continue
			}
			// The direct complexType grammar is a CHOICE: either a simpleContent/
			// complexContent WRAPPER, or the (openContent?, particle?, attrs,
			// anyAttribute?, assert*) branch — never both. An openContent alongside a
			// wrapper mixes the two branches and is rejected.
			if contentWrapperChild != "" {
				reportExtraContent(ce, fmt.Sprintf("The 'openContent' is not allowed together with '%s'.", contentWrapperChild))
				continue
			}
			if msg := openContentOrderViolation(contentModelChild, directAttrChild, anyAttributeSeen, assertSeen); msg != "" {
				reportExtraContent(ce, msg)
				continue
			}
			openContentSeen = true
			td.openContentExplicit = true
			td.OpenContent = c.parseOpenContent(ctx, ce)
		}
	}

	// XSD 1.1: capture the schema-level <xs:defaultOpenContent> active in this
	// type's OWN document for resolveOpenContent to apply, but only when the type
	// has no explicit <xs:openContent> (default open content is per-document and
	// suppressed by an explicit openContent, including mode="none").
	if c.version == Version11 && !td.openContentExplicit {
		td.pendingDefaultOpenContent = c.defaultOpenContent
	}

	// If no content model and not mixed, ContentTypeEmpty is the default (no children).
	// Attribute declarations do not change the content type.
	if defaultAttrsApply && c.schema.defaultAttrsSet {
		c.recordAttrGroupRef(td, c.schema.defaultAttributes, c.schema.defaultAttrsSrc)
	}

	return td, nil
}

func (c *compiler) parseComplexContent(ctx context.Context, elem *helium.Element, td *TypeDef) error {
	// XSD 1.1: <xs:complexContent> may carry its own @mixed. The effective
	// mixedness is complexType/@mixed OR complexContent/@mixed; if BOTH are present
	// and disagree it is a schema error (a complexType mixed="true" with
	// complexContent mixed="false"). At entry td.ContentType==Mixed iff the enclosing
	// complexType set mixed="true". Set mixedness BEFORE parsing the derivation so the
	// model-group handlers (which set ElementOnly only when not already Mixed) keep it.
	// Gated to 1.1 so XSD 1.0 stays byte-identical (it never honored this attribute).
	if c.version == Version11 && hasAttr(elem, "mixed") {
		ccMixed := c.readBooleanAttr(ctx, elem, "mixed")
		ctMixed := td.ContentType == ContentTypeMixed
		if ctMixed && !ccMixed && c.filename != "" {
			c.schemaError(ctx, schemaComponentError(c.diagSource(), elem.Line(),
				elem.LocalName(), componentLocalComplexType,
				"The 'mixed' attribute on 'complexType' and 'complexContent' must not conflict."))
		}
		if ccMixed {
			td.ContentType = ContentTypeMixed
		}
	}
	// XSD 3.4.2: the <xs:complexContent> content model is (annotation?, (restriction
	// | extension)). Any other child — including a stray <xs:openContent>, which
	// belongs INSIDE the restriction/extension wrapper, not as a sibling of it — is a
	// grammar error; without this guard such a child was silently ignored and the
	// schema wrongly compiled.
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
			continue
		case isXSDElement(ce, elemRestriction):
			return c.parseRestriction(ctx, ce, td)
		case isXSDElement(ce, elemExtension):
			return c.parseExtension(ctx, ce, td)
		default:
			if c.filename != "" {
				c.schemaError(ctx, schemaComponentError(c.diagSource(), ce.Line(),
					elem.LocalName(), componentLocalComplexType,
					fmt.Sprintf("The element '%s' is not allowed in 'complexContent'; only 'restriction' or 'extension' is permitted.", ce.LocalName())))
			}
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

	// XSD 3.4.2 fixes the child ordering of a derivation: an optional leading
	// model-group particle (sequence|choice|all|group ref) — at most one — then
	// attribute/attributeGroup uses, then an optional anyAttribute. Track the
	// first model-group particle and the first attribute-region child so a
	// second particle (which would silently overwrite td.ContentModel) and a
	// particle that appears AFTER an attribute declaration are both rejected.
	// anyAttributeSeen tracks the optional trailing attribute wildcard so a
	// later attribute/attributeGroup use, or a second wildcard, is rejected.
	var contentModelChild string
	var directAttrChild string
	var anyAttributeSeen bool
	var assertSeen bool
	var openContentSeen bool

	reportOrder := func(ce *helium.Element, what string) {
		if c.filename == "" {
			return
		}
		c.schemaError(ctx, schemaComponentError(c.diagSource(), ce.Line(),
			elem.LocalName(), componentLocalComplexType, what))
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
		if isXSDElement(ce, elemSequence) || isXSDElement(ce, elemChoice) || isXSDElement(ce, elemAll) || isXSDElement(ce, elemGroup) {
			if directAttrChild != "" {
				if c.filename != "" {
					c.schemaError(ctx, schemaComponentError(c.diagSource(), ce.Line(),
						elem.LocalName(), componentLocalComplexType,
						fmt.Sprintf("The content model particle '%s' must appear before the attribute declaration '%s'.", ce.LocalName(), directAttrChild)))
				}
				continue
			}
			if contentModelChild != "" {
				if c.filename != "" {
					c.schemaError(ctx, schemaComponentError(c.diagSource(), ce.Line(),
						elem.LocalName(), componentLocalComplexType,
						fmt.Sprintf("A complex type definition must not have more than one content model particle (found '%s' after '%s').", ce.LocalName(), contentModelChild)))
				}
				continue
			}
			contentModelChild = ce.LocalName()
		}
		if isXSDElement(ce, elemAttribute) || isXSDElement(ce, elemAttributeGroup) || isXSDElement(ce, elemAnyAttribute) {
			if directAttrChild == "" {
				directAttrChild = ce.LocalName()
			}
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
					source:       c.diagSource(),
				}
				td.ContentModel = placeholder
				if td.ContentType != ContentTypeMixed {
					td.ContentType = ContentTypeElementOnly
				}
			}
		case isXSDElement(ce, elemAttribute):
			if anyAttributeSeen {
				reportOrder(ce, fmt.Sprintf("The attribute declaration '%s' must appear before the attribute wildcard 'anyAttribute'.", ce.LocalName()))
				continue
			}
			au := c.parseAttributeUse(ctx, ce)
			td.Attributes = append(td.Attributes, au)
		case isXSDElement(ce, elemAttributeGroup):
			if anyAttributeSeen {
				reportOrder(ce, fmt.Sprintf("The attribute declaration '%s' must appear before the attribute wildcard 'anyAttribute'.", ce.LocalName()))
				continue
			}
			if ref := getAttr(ce, attrRef); ref != "" {
				qn := c.resolveQName(ctx, ce, ref)
				c.recordAttrGroupRef(td, qn, attrGroupRefUseSource{
					line:      ce.Line(),
					elemLocal: ce.LocalName(),
					attr:      attrRef,
					source:    c.diagSource(),
				})
			}
		case isXSDElement(ce, elemAnyAttribute):
			if anyAttributeSeen {
				reportOrder(ce, fmt.Sprintf("A complex type definition must not have more than one attribute wildcard (found a second '%s').", ce.LocalName()))
				continue
			}
			anyAttributeSeen = true
			td.AnyAttribute = c.parseAnyAttribute(ctx, ce)
		case isXSDElement(ce, elemAssert) && c.version == Version11:
			assertSeen = true
			if a := c.parseAssert(ctx, ce); a != nil {
				td.Assertions = append(td.Assertions, a)
			}
		case isXSDElement(ce, elemOpenContent) && c.version == Version11:
			if openContentSeen {
				reportOrder(ce, "A complex type definition must not have more than one 'openContent'.")
				continue
			}
			if msg := openContentOrderViolation(contentModelChild, directAttrChild, anyAttributeSeen, assertSeen); msg != "" {
				reportOrder(ce, msg)
				continue
			}
			openContentSeen = true
			td.openContentExplicit = true
			td.OpenContent = c.parseOpenContent(ctx, ce)
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
	// XSD 3.4.2 fixes the child ordering of a derivation: an optional leading
	// model-group particle (sequence|choice|all|group ref) — at most one — then
	// attribute/attributeGroup uses, then an optional anyAttribute. Track the
	// first model-group particle and the first attribute-region child so a
	// second particle (which would silently overwrite td.ContentModel) and a
	// particle that appears AFTER an attribute declaration are both rejected.
	// anyAttributeSeen tracks the optional trailing attribute wildcard so a
	// later attribute/attributeGroup use, or a second wildcard, is rejected.
	var contentModelChild string
	var directAttrChild string
	var anyAttributeSeen bool
	var assertSeen bool
	var openContentSeen bool

	reportOrder := func(ce *helium.Element, what string) {
		if c.filename == "" {
			return
		}
		c.schemaError(ctx, schemaComponentError(c.diagSource(), ce.Line(),
			elem.LocalName(), componentLocalComplexType, what))
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
		if isXSDElement(ce, elemSequence) || isXSDElement(ce, elemChoice) || isXSDElement(ce, elemAll) || isXSDElement(ce, elemGroup) {
			if directAttrChild != "" {
				if c.filename != "" {
					c.schemaError(ctx, schemaComponentError(c.diagSource(), ce.Line(),
						elem.LocalName(), componentLocalComplexType,
						fmt.Sprintf("The content model particle '%s' must appear before the attribute declaration '%s'.", ce.LocalName(), directAttrChild)))
				}
				continue
			}
			if contentModelChild != "" {
				if c.filename != "" {
					c.schemaError(ctx, schemaComponentError(c.diagSource(), ce.Line(),
						elem.LocalName(), componentLocalComplexType,
						fmt.Sprintf("A complex type definition must not have more than one content model particle (found '%s' after '%s').", ce.LocalName(), contentModelChild)))
				}
				continue
			}
			contentModelChild = ce.LocalName()
		}
		if isXSDElement(ce, elemAttribute) || isXSDElement(ce, elemAttributeGroup) || isXSDElement(ce, elemAnyAttribute) {
			if directAttrChild == "" {
				directAttrChild = ce.LocalName()
			}
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
					source:       c.diagSource(),
				}
				td.ContentModel = placeholder
				if td.ContentType != ContentTypeMixed {
					td.ContentType = ContentTypeElementOnly
				}
			}
		case isXSDElement(ce, elemAttribute):
			if anyAttributeSeen {
				reportOrder(ce, fmt.Sprintf("The attribute declaration '%s' must appear before the attribute wildcard 'anyAttribute'.", ce.LocalName()))
				continue
			}
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
			if anyAttributeSeen {
				reportOrder(ce, fmt.Sprintf("The attribute declaration '%s' must appear before the attribute wildcard 'anyAttribute'.", ce.LocalName()))
				continue
			}
			if ref := getAttr(ce, attrRef); ref != "" {
				qn := c.resolveQName(ctx, ce, ref)
				c.recordAttrGroupRef(td, qn, attrGroupRefUseSource{
					line:      ce.Line(),
					elemLocal: ce.LocalName(),
					attr:      attrRef,
					source:    c.diagSource(),
				})
			}
		case isXSDElement(ce, elemAnyAttribute):
			if anyAttributeSeen {
				reportOrder(ce, fmt.Sprintf("A complex type definition must not have more than one attribute wildcard (found a second '%s').", ce.LocalName()))
				continue
			}
			anyAttributeSeen = true
			td.AnyAttribute = c.parseAnyAttribute(ctx, ce)
		case isXSDElement(ce, elemAssert) && c.version == Version11:
			assertSeen = true
			if a := c.parseAssert(ctx, ce); a != nil {
				td.Assertions = append(td.Assertions, a)
			}
		case isXSDElement(ce, elemOpenContent) && c.version == Version11:
			if openContentSeen {
				reportOrder(ce, "A complex type definition must not have more than one 'openContent'.")
				continue
			}
			if msg := openContentOrderViolation(contentModelChild, directAttrChild, anyAttributeSeen, assertSeen); msg != "" {
				reportOrder(ce, msg)
				continue
			}
			openContentSeen = true
			td.openContentExplicit = true
			td.OpenContent = c.parseOpenContent(ctx, ce)
		}
	}
	return nil
}

func (c *compiler) parseSimpleContent(ctx context.Context, elem *helium.Element, td *TypeDef) {
	td.ContentType = ContentTypeSimple
	td.IsSimpleContent = true
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
			c.parseSimpleContentChildren(ctx, ce, td, DerivationExtension)
		case isXSDElement(ce, elemRestriction):
			baseRef := getAttr(ce, attrBase)
			if baseRef != "" {
				qn := c.resolveQName(ctx, ce, baseRef)
				c.typeRefs[td] = qn
				c.markChameleonEligible(td, ce, baseRef)
			}
			td.Derivation = DerivationRestriction
			c.parseSimpleContentChildren(ctx, ce, td, DerivationRestriction)
		}
	}
}

// parseSimpleContentChildren parses attribute/attributeGroup children within
// a simpleContent extension or restriction element. kind selects whether the
// derivation is an extension or a restriction; on an EXTENSION a prohibited
// attribute use (use="prohibited") is pointless and is warned+skipped, matching
// complexContent xs:extension (parseExtension), so it does not propagate and
// wrongly block an attribute the base wildcard would otherwise admit.
func (c *compiler) parseSimpleContentChildren(ctx context.Context, derivation *helium.Element, td *TypeDef, kind DerivationKind) {
	// XSD 3.4.2: within a simpleContent derivation the attribute/attributeGroup
	// uses come first, followed by an optional trailing anyAttribute wildcard.
	// anyAttributeSeen tracks the wildcard so a later attribute/attributeGroup
	// use, or a second wildcard, is rejected.
	var anyAttributeSeen bool

	reportOrder := func(ae *helium.Element, what string) {
		if c.filename == "" {
			return
		}
		c.schemaError(ctx, schemaComponentError(c.diagSource(), ae.Line(),
			derivation.LocalName(), componentLocalComplexType, what))
	}

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
			if anyAttributeSeen {
				reportOrder(ae, fmt.Sprintf("The attribute declaration '%s' must appear before the attribute wildcard 'anyAttribute'.", ae.LocalName()))
				continue
			}
			if kind == DerivationExtension && getAttr(ae, attrUse) == attrValProhibited {
				if c.filename != "" {
					c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserWarning(c.filename, ae.Line(), ae.LocalName(), "attribute",
						"Skipping attribute use prohibition, since it is pointless when extending a type."), helium.ErrorLevelWarning))
				}
				continue
			}
			au := c.parseAttributeUse(ctx, ae)
			td.Attributes = append(td.Attributes, au)
		case isXSDElement(ae, elemAttributeGroup):
			if anyAttributeSeen {
				reportOrder(ae, fmt.Sprintf("The attribute declaration '%s' must appear before the attribute wildcard 'anyAttribute'.", ae.LocalName()))
				continue
			}
			if ref := getAttr(ae, attrRef); ref != "" {
				qn := c.resolveQName(ctx, ae, ref)
				c.recordAttrGroupRef(td, qn, attrGroupRefUseSource{
					line:      ae.Line(),
					elemLocal: ae.LocalName(),
					attr:      attrRef,
					source:    c.diagSource(),
				})
			}
		case isXSDElement(ae, elemAnyAttribute):
			if anyAttributeSeen {
				reportOrder(ae, fmt.Sprintf("A complex type definition must not have more than one attribute wildcard (found a second '%s').", ae.LocalName()))
				continue
			}
			anyAttributeSeen = true
			td.AnyAttribute = c.parseAnyAttribute(ctx, ae)
		case isXSDElement(ae, elemAssert) && c.version == Version11:
			// XSD 1.1 xs:assert on a complexType with simpleContent. The assert is
			// evaluated against the element after content validation, with $value
			// bound to the element's typed simple value (see checkAssertions).
			if a := c.parseAssert(ctx, ae); a != nil {
				td.Assertions = append(td.Assertions, a)
			}
		case isXSDElement(ae, elemOpenContent) && c.version == Version11:
			// xs:openContent is not permitted in simpleContent: a simple-content type
			// has NO element content for open content to apply to. The direct
			// complexType grammar already rejects an openContent sibling of a
			// simpleContent/complexContent wrapper; this rejects it INSIDE the wrapper.
			reportOrder(ae, "The 'openContent' is not allowed in a 'simpleContent' derivation.")
		}
	}

	// XSD 1.1: a simpleContent RESTRICTION narrows the base content type via a
	// nested <xs:simpleType> OR direct facets. Capture the resulting effective
	// content simple type so validateSimpleContent checks the text against the
	// narrowed type (e.g. an enumeration or a restriction to xs:float) rather than
	// only the base. Gated to 1.1 so XSD 1.0 content validation stays byte-identical.
	if kind == DerivationRestriction && c.version == Version11 {
		td.ContentSimpleType = c.parseSimpleContentRestrictionType(ctx, derivation, td)
	}
}

// parseSimpleContentRestrictionType derives the effective content simple type of
// a simpleContent <xs:restriction>. A nested <xs:simpleType> defines the base;
// the restriction's DIRECT facet children (siblings of that simpleType) further
// constrain it — both must compose. So:
//   - inline simpleType only → that simpleType;
//   - inline simpleType + direct facets → a restriction of the inline type
//     carrying those sibling facets (so both sets apply);
//   - direct facets only → a restriction of the base content type (BaseType =
//     the owning complex type, whose base chain resolves to the builtin base);
//   - neither → nil (the restriction inherits the base content type unchanged).
func (c *compiler) parseSimpleContentRestrictionType(ctx context.Context, derivation *helium.Element, owner *TypeDef) *TypeDef {
	var inline *TypeDef
	for child := range helium.Children(derivation) {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		if isXSDElement(ce, elemSimpleType) {
			st, err := c.parseSimpleType(ctx, ce)
			if err == nil {
				inline = st
			}
			break
		}
	}
	// Direct facet children (xs:enumeration, xs:length, …) of the restriction.
	// parseFacets ignores the nested <xs:simpleType> (not a facet element), so
	// these are exactly the sibling facets.
	fs := c.parseFacets(ctx, derivation)

	switch {
	case inline != nil && fs == nil:
		return inline
	case inline != nil:
		// Compose: restrict the inline type with the sibling facets.
		syn := &TypeDef{
			ContentType: ContentTypeSimple,
			Derivation:  DerivationRestriction,
			BaseType:    inline,
			Facets:      fs,
		}
		// Record the synthetic type so checkFacetConsistency runs its facet
		// applicability / value-against-base checks (an inapplicable sibling facet,
		// e.g. xs:minInclusive on an xs:string base, must be rejected at compile).
		c.recordTypeDefSource(syn, derivation.Line(), true, elemSimpleType)
		return syn
	case fs != nil:
		syn := &TypeDef{
			ContentType: ContentTypeSimple,
			Derivation:  DerivationRestriction,
			BaseType:    owner,
			Facets:      fs,
		}
		c.recordTypeDefSource(syn, derivation.Line(), true, elemSimpleType)
		return syn
	default:
		return nil
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
	c.recordTypeDefSource(td, elem.Line(), true, elem.LocalName())

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
	seenSingletonFacets := make(map[string]struct{})
	duplicateSingletonFacet := func(elem *helium.Element, name string) bool {
		if _, ok := seenSingletonFacets[name]; !ok {
			seenSingletonFacets[name] = struct{}{}
			return false
		}
		c.schemaError(ctx, schemaParserError(c.filename, elem.Line(),
			elem.LocalName(), elem.LocalName(),
			fmt.Sprintf("It is an error for the facet '%s' to be specified more than once on the same type definition.", name)))
		return true
	}

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
		case facetMinInclusive:
			if duplicateSingletonFacet(ce, facetMinInclusive) {
				continue
			}
			if fs == nil {
				fs = &FacetSet{}
			}
			fs.MinInclusive = &val
			fs.MinInclusiveFixed = c.readFacetFixed(ctx, ce)
			fs.MinInclusiveNS = captureFacetNS(ce)
		case facetMaxInclusive:
			if duplicateSingletonFacet(ce, facetMaxInclusive) {
				continue
			}
			if fs == nil {
				fs = &FacetSet{}
			}
			fs.MaxInclusive = &val
			fs.MaxInclusiveFixed = c.readFacetFixed(ctx, ce)
			fs.MaxInclusiveNS = captureFacetNS(ce)
		case facetMinExclusive:
			if duplicateSingletonFacet(ce, facetMinExclusive) {
				continue
			}
			if fs == nil {
				fs = &FacetSet{}
			}
			fs.MinExclusive = &val
			fs.MinExclusiveFixed = c.readFacetFixed(ctx, ce)
			fs.MinExclusiveNS = captureFacetNS(ce)
		case facetMaxExclusive:
			if duplicateSingletonFacet(ce, facetMaxExclusive) {
				continue
			}
			if fs == nil {
				fs = &FacetSet{}
			}
			fs.MaxExclusive = &val
			fs.MaxExclusiveFixed = c.readFacetFixed(ctx, ce)
			fs.MaxExclusiveNS = captureFacetNS(ce)
		case "totalDigits":
			if duplicateSingletonFacet(ce, "totalDigits") {
				continue
			}
			if fs == nil {
				fs = &FacetSet{}
			}
			n := parseOccurs(val, 0)
			fs.TotalDigits = &n
		case "length":
			if duplicateSingletonFacet(ce, "length") {
				continue
			}
			if fs == nil {
				fs = &FacetSet{}
			}
			n := parseOccurs(val, 0)
			fs.Length = &n
		case "minLength":
			if duplicateSingletonFacet(ce, "minLength") {
				continue
			}
			if fs == nil {
				fs = &FacetSet{}
			}
			n := parseOccurs(val, 0)
			fs.MinLength = &n
		case "maxLength":
			if duplicateSingletonFacet(ce, "maxLength") {
				continue
			}
			if fs == nil {
				fs = &FacetSet{}
			}
			n := parseOccurs(val, 0)
			fs.MaxLength = &n
		case "fractionDigits":
			if duplicateSingletonFacet(ce, "fractionDigits") {
				continue
			}
			if fs == nil {
				fs = &FacetSet{}
			}
			n := parseOccurs(val, 0)
			fs.FractionDigits = &n
		case "whiteSpace":
			if duplicateSingletonFacet(ce, "whiteSpace") {
				continue
			}
			if fs == nil {
				fs = &FacetSet{}
			}
			fs.WhiteSpace = &val
		case elemExplicitTimezone:
			if c.version != Version11 {
				continue
			}
			if duplicateSingletonFacet(ce, elemExplicitTimezone) {
				continue
			}
			if fs == nil {
				fs = &FacetSet{}
			}
			val = normalizeWhiteSpace(val, "collapse")
			switch val {
			case attrValOptional, attrValProhibited, attrValRequired:
				fs.ExplicitTimezone = &val
				fs.ExplicitTimezoneFixed = c.readFacetFixed(ctx, ce)
			default:
				c.schemaError(ctx, schemaParserError(c.filename, ce.Line(),
					ce.LocalName(), elemExplicitTimezone,
					fmt.Sprintf("The value '%s' is not a valid value for the 'explicitTimezone' facet.", val)))
			}
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
				c.schemaError(ctx, schemaParserError(c.filename, ce.Line(),
					ce.LocalName(), "pattern",
					fmt.Sprintf("The value '%s' is not a valid regular expression: %s.", val, rerr)))
			}
			fs.compiledPatterns = append(fs.compiledPatterns, re)
		case "assertion":
			// XSD 1.1 <xs:assertion> simple-type facet. Evaluated at simple-value
			// validation time with $value bound to the typed atomic value.
			if c.version != Version11 {
				continue
			}
			if a := c.parseAssertion(ctx, ce, elemAssertion); a != nil {
				if fs == nil {
					fs = &FacetSet{}
				}
				fs.Assertions = append(fs.Assertions, a)
			}
		}
	}

	return fs
}

func (c *compiler) readFacetFixed(ctx context.Context, elem *helium.Element) bool {
	if !hasAttr(elem, attrFixed) {
		return false
	}
	v, ok := parseSchemaBool(getAttr(elem, attrFixed))
	if ok {
		return v
	}
	msg := fmt.Sprintf("'%s' is not a valid value of the atomic type 'xs:boolean'.", normalizeWhiteSpace(getAttr(elem, attrFixed), "collapse"))
	c.schemaError(ctx, schemaParserErrorAttr(c.filename, elem.Line(),
		elem.LocalName(), elem.LocalName(), attrFixed, msg))
	return false
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
