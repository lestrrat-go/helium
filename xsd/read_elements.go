package xsd

import (
	"context"
	"fmt"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/xpath1"
)

type elementDeclReadOptions struct {
	name                   string
	namespace              string
	minOccurs              int
	maxOccurs              int
	defaultBlock           BlockFlags
	defaultFinal           FinalFlags
	allowAbstract          bool
	allowFinal             bool
	allowSubstitutionGroup bool
}

type attrUseReadOptions struct {
	name       QName
	includeUse bool
}

func parseParticleOccurs(elem *helium.Element) (int, int) {
	minOccurs := 1
	maxOccurs := 1
	if v := getAttr(elem, attrMinOccurs); v != "" {
		minOccurs = parseOccurs(v, 1)
	}
	if v := getAttr(elem, attrMaxOccurs); v != "" {
		maxOccurs = parseOccurs(v, 1)
	}
	return minOccurs, maxOccurs
}

func readDefaultOrFixed(elem *helium.Element) (*string, *string) {
	var defaultValue *string
	if hasAttr(elem, attrDefault) {
		v := getAttr(elem, attrDefault)
		defaultValue = &v
	}

	var fixedValue *string
	if hasAttr(elem, attrFixed) {
		v := getAttr(elem, attrFixed)
		fixedValue = &v
	}

	return defaultValue, fixedValue
}

func readProcessContents(elem *helium.Element) ProcessContentsKind {
	switch getAttr(elem, attrProcessContents) {
	case attrValLax:
		return ProcessLax
	case attrValSkip:
		return ProcessSkip
	default:
		return ProcessStrict
	}
}

func (c *compiler) readWildcard(_ context.Context, elem *helium.Element) *Wildcard {
	namespace := getAttr(elem, attrNamespace)
	if namespace == "" {
		namespace = WildcardNSAny
	}

	return &Wildcard{
		Namespace:       namespace,
		ProcessContents: readProcessContents(elem),
		TargetNS:        c.schema.targetNamespace,
	}
}

func (c *compiler) readElementDecl(ctx context.Context, elem *helium.Element, opts elementDeclReadOptions) (*ElementDecl, error) {
	decl := &ElementDecl{
		Name:      QName{Local: opts.name, NS: opts.namespace},
		MinOccurs: opts.minOccurs,
		MaxOccurs: opts.maxOccurs,
		Nillable:  c.readBooleanAttr(ctx, elem, attrNillable),
	}

	if opts.allowAbstract {
		decl.Abstract = getAttr(elem, attrAbstract) == attrValTrue
	}

	if opts.allowSubstitutionGroup {
		if sg := getAttr(elem, attrSubstitutionGroup); sg != "" {
			decl.SubstitutionGroup = c.resolveQName(ctx, elem, sg)
		}
	}

	decl.Default, decl.Fixed = readDefaultOrFixed(elem)
	if decl.Fixed != nil {
		decl.FixedNS = collectNSContext(elem)
	}

	if hasAttr(elem, attrBlock) {
		decl.Block = parseBlockFlags(getAttr(elem, attrBlock))
		decl.BlockSet = true
	} else {
		decl.Block = opts.defaultBlock
	}

	if opts.allowFinal {
		if hasAttr(elem, attrFinal) {
			decl.Final = parseElemFinalFlags(getAttr(elem, attrFinal))
			decl.FinalSet = true
		} else {
			decl.Final = opts.defaultFinal
		}
	}

	if err := c.readElementType(ctx, elem, decl, opts.name); err != nil {
		return nil, err
	}
	decl.IDCs = c.parseIDConstraints(ctx, elem)
	return decl, nil
}

// readBooleanAttr reads a schema-side xs:boolean attribute (e.g. nillable),
// applying whitespace-collapse lexical rules (true/false/1/0). An absent
// attribute is false. An invalid lexical is reported as a schema parser error
// and treated as false.
func (c *compiler) readBooleanAttr(ctx context.Context, elem *helium.Element, attr string) bool {
	if !hasAttr(elem, attr) {
		return false
	}
	v := normalizeWhiteSpace(getAttr(elem, attr), "collapse")
	switch v {
	case "true", "1":
		return true
	case "false", "0":
		return false
	}
	msg := fmt.Sprintf("'%s' is not a valid value of the atomic type 'xs:boolean'.", v)
	c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserErrorAttr(c.filename, elem.Line(),
		elem.LocalName(), elemElement, attr, msg), helium.ErrorLevelFatal))
	c.errorCount++
	return false
}

func (c *compiler) readElementType(ctx context.Context, elem *helium.Element, decl *ElementDecl, sourceName string) error {
	typeRef := getAttr(elem, attrType)
	if typeRef != "" {
		qn := c.resolveQName(ctx, elem, typeRef)
		c.elemRefs[decl] = qn
		c.markChameleonEligible(decl, elem, typeRef)
		c.elemRefSources[decl] = elemRefSource{elemName: sourceName, line: elem.Line()}
		return nil
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
		case isXSDElement(ce, elemAnnotation):
			c.checkAnnotation(ctx, ce)
		case isXSDElement(ce, elemComplexType):
			td, err := c.parseComplexType(ctx, ce)
			if err != nil {
				return err
			}
			decl.Type = td
		case isXSDElement(ce, elemSimpleType):
			td, err := c.parseSimpleType(ctx, ce)
			if err != nil {
				return err
			}
			decl.Type = td
		}
	}

	// An element declaration with no explicit type, no inline type, and no
	// substitution-group head to inherit from defaults to the built-in
	// xs:anyType (XSD 3.3.2: {type definition} defaults to xs:anyType). This
	// ensures xsi:nil lexical validation and nilled-empty enforcement run for
	// no-type declarations the same as for typed ones. Substitution-group
	// members are left untyped so they can inherit the head's type at validation.
	if decl.Type == nil && decl.SubstitutionGroup == (QName{}) {
		decl.Type = c.schema.types[QName{Local: "anyType", NS: lexicon.NamespaceXSD}]
	}

	return nil
}

func (c *compiler) readAttributeUseDecl(ctx context.Context, elem *helium.Element, opts attrUseReadOptions) *AttrUse {
	au := &AttrUse{Name: opts.name}
	if typeRef := getAttr(elem, attrType); typeRef != "" {
		au.TypeName = c.resolveQName(ctx, elem, typeRef)
	} else {
		// No type attribute: look for an inline anonymous <xs:simpleType>.
		// (type and inline simpleType are mutually exclusive, enforced by
		// checkAttributeUse.)
		for child := range helium.Children(elem) {
			if child.Type() != helium.ElementNode {
				continue
			}
			ce, ok := helium.AsNode[*helium.Element](child)
			if !ok {
				continue
			}
			if !isXSDElement(ce, elemSimpleType) {
				continue
			}
			td, err := c.parseSimpleType(ctx, ce)
			if err != nil {
				c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserError(c.filename, ce.Line(),
					ce.LocalName(), "attribute", err.Error()), helium.ErrorLevelFatal))
				c.errorCount++
				break
			}
			au.Type = td
			break
		}
	}
	if opts.includeUse {
		switch getAttr(elem, attrUse) {
		case attrValRequired:
			au.Required = true
		case attrValProhibited:
			au.Prohibited = true
		}
	}
	au.Default, au.Fixed = readDefaultOrFixed(elem)
	if au.Fixed != nil {
		au.FixedNS = collectNSContext(elem)
	}
	// Record source info so the default/fixed constraint value can be validated
	// against the attribute's declared simple type once type refs are resolved.
	if au.Default != nil || au.Fixed != nil {
		c.attrUseConstraintSources[au] = attrConstraintSource{
			line:  elem.Line(),
			local: opts.name.Local,
			nsMap: collectNSContext(elem),
		}
	}
	return au
}

func (c *compiler) parseGlobalElement(ctx context.Context, elem *helium.Element) error {
	c.checkGlobalElement(ctx, elem)
	name := getAttr(elem, attrName)
	if name == "" {
		// Still register with a placeholder name to continue parsing.
		return nil
	}

	decl, err := c.readElementDecl(ctx, elem, elementDeclReadOptions{
		name:                   name,
		namespace:              c.schema.targetNamespace,
		minOccurs:              1,
		maxOccurs:              1,
		defaultBlock:           c.schema.blockDefault,
		defaultFinal:           c.schema.finalDefault & (FinalExtension | FinalRestriction),
		allowAbstract:          true,
		allowFinal:             true,
		allowSubstitutionGroup: true,
	})
	if err != nil {
		return err
	}

	// Check for duplicate global element declarations.
	if _, exists := c.schema.elements[decl.Name]; exists {
		qnDisplay := "'" + decl.Name.NS + "'" + decl.Name.Local
		if decl.Name.NS != "" {
			qnDisplay = "'{" + decl.Name.NS + "}" + decl.Name.Local + "'"
		}
		source := c.includeFile
		c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserError(source, elem.Line(),
			elem.LocalName(), "element",
			"A global element declaration "+qnDisplay+" does already exist."), helium.ErrorLevelFatal))
		c.errorCount++
		return nil
	}

	c.globalElemSources[decl] = elemRefSource{elemName: name, line: elem.Line()}
	c.schema.elements[decl.Name] = decl
	return nil
}

func (c *compiler) parseLocalElement(ctx context.Context, elem *helium.Element) (*Particle, error) {
	c.checkLocalElement(ctx, elem)
	minOcc, maxOcc := parseParticleOccurs(elem)

	// Handle element references (ref="...").
	if ref := getAttr(elem, attrRef); ref != "" {
		qn := c.resolveQName(ctx, elem, ref)
		edecl := &ElementDecl{
			Name:      qn,
			MinOccurs: minOcc,
			MaxOccurs: maxOcc,
			IsRef:     true,
		}
		c.elemRefs[edecl] = qn
		c.elemRefSources[edecl] = elemRefSource{elemName: elem.LocalName(), line: elem.Line()}
		return &Particle{
			MinOccurs: minOcc,
			MaxOccurs: maxOcc,
			Term:      edecl,
		}, nil
	}

	name := getAttr(elem, attrName)
	if name == "" {
		return nil, fmt.Errorf("xsd: local element missing name")
	}

	// Determine element namespace based on form and elementFormDefault.
	elemNS := ""
	form := getAttr(elem, attrForm)
	if form == attrValQualified || (form == "" && c.schema.elemFormQualified) {
		elemNS = c.schema.targetNamespace
	}

	edecl, err := c.readElementDecl(ctx, elem, elementDeclReadOptions{
		name:         name,
		namespace:    elemNS,
		minOccurs:    minOcc,
		maxOccurs:    maxOcc,
		defaultBlock: c.schema.blockDefault,
	})
	if err != nil {
		return nil, err
	}

	return &Particle{
		MinOccurs: minOcc,
		MaxOccurs: maxOcc,
		Term:      edecl,
	}, nil
}

func (c *compiler) parseWildcard(ctx context.Context, elem *helium.Element) *Particle {
	minOcc, maxOcc := parseParticleOccurs(elem)
	wc := c.readWildcard(ctx, elem)
	return &Particle{
		MinOccurs: minOcc,
		MaxOccurs: maxOcc,
		Term:      wc,
	}
}

func (c *compiler) parseAnyAttribute(ctx context.Context, elem *helium.Element) *Wildcard {
	return c.readWildcard(ctx, elem)
}

func (c *compiler) parseGlobalAttribute(ctx context.Context, elem *helium.Element) {
	c.checkAttributeUse(ctx, elem)
	name := getAttr(elem, attrName)
	if name == "" {
		return
	}
	// Global attributes are always in the target namespace (per spec).
	au := c.readAttributeUseDecl(ctx, elem, attrUseReadOptions{
		name: QName{Local: name, NS: c.schema.targetNamespace},
	})
	c.schema.globalAttrs[au.Name] = au
}

func (c *compiler) parseAttributeUse(ctx context.Context, elem *helium.Element) *AttrUse {
	c.checkAttributeUse(ctx, elem)
	// Handle attribute references (ref="...").
	if ref := getAttr(elem, attrRef); ref != "" {
		qn := c.resolveQName(ctx, elem, ref)
		au := &AttrUse{Name: qn}
		if getAttr(elem, attrUse) == attrValRequired {
			au.Required = true
		}
		if hasAttr(elem, attrDefault) {
			v := getAttr(elem, attrDefault)
			au.Default = &v
		}
		if hasAttr(elem, attrFixed) {
			v := getAttr(elem, attrFixed)
			au.Fixed = &v
			au.FixedNS = collectNSContext(elem)
		}
		// Record source info so a local default/fixed constraint on a ref'd
		// attribute use is validated against the resolved (global) attribute's
		// simple type once resolveRefs copies the type in.
		if au.Default != nil || au.Fixed != nil {
			c.attrUseConstraintSources[au] = attrConstraintSource{
				line:  elem.Line(),
				local: qn.Local,
				nsMap: collectNSContext(elem),
			}
		}
		c.attrRefs[au] = qn
		return au
	}

	name := getAttr(elem, attrName)
	// Determine attribute namespace based on form and attributeFormDefault.
	attrNS := ""
	form := getAttr(elem, attrForm)
	if form == attrValQualified || (form == "" && c.schema.attrFormQualified) {
		attrNS = c.schema.targetNamespace
	}
	return c.readAttributeUseDecl(ctx, elem, attrUseReadOptions{
		name:       QName{Local: name, NS: attrNS},
		includeUse: true,
	})
}

// parseIDConstraints scans element children for xs:key, xs:keyref, xs:unique declarations.
func (c *compiler) parseIDConstraints(ctx context.Context, elem *helium.Element) []*IDConstraint {
	var idcs []*IDConstraint
	for child := range helium.Children(elem) {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		var kind IDCKind
		switch {
		case isXSDElement(ce, elemUnique):
			kind = IDCUnique
		case isXSDElement(ce, elemKey):
			kind = IDCKey
		case isXSDElement(ce, elemKeyRef):
			kind = IDCKeyRef
		default:
			continue
		}
		idc := c.parseIDConstraint(ctx, ce, kind)
		if idc != nil {
			idcs = append(idcs, idc)
		}
	}
	return idcs
}

// parseIDConstraint parses a single xs:key, xs:keyref, or xs:unique declaration.
func (c *compiler) parseIDConstraint(_ context.Context, elem *helium.Element, kind IDCKind) *IDConstraint {
	name := getAttr(elem, attrName)
	if name == "" {
		return nil
	}
	idc := &IDConstraint{
		Name:       name,
		Kind:       kind,
		Namespaces: collectNSContext(elem),
	}
	if kind == IDCKeyRef {
		idc.Refer = getAttr(elem, attrRefer)
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
		case isXSDElement(ce, elemSelector):
			idc.Selector = getAttr(ce, attrXPath)
		case isXSDElement(ce, elemField):
			idc.Fields = append(idc.Fields, getAttr(ce, attrXPath))
		}
	}

	// Pre-compile selector XPath expression.
	if idc.Selector != "" {
		compiled, err := xpath1.Compile(idc.Selector)
		if err == nil {
			idc.SelectorExpr = compiled
		}
	}

	// Pre-compile field XPath expressions.
	idc.FieldExprs = make([]*xpath1.Expression, len(idc.Fields))
	for i, f := range idc.Fields {
		compiled, err := xpath1.Compile(f)
		if err == nil {
			idc.FieldExprs[i] = compiled
		}
	}

	return idc
}
