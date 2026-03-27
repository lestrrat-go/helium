package xsd

import (
	"fmt"

	helium "github.com/lestrrat-go/helium"
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
	if v := getAttr(elem, attrDefault); v != "" {
		defaultValue = &v
	}

	var fixedValue *string
	if v := getAttr(elem, attrFixed); v != "" {
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

func (c *compiler) readWildcard(elem *helium.Element) *Wildcard {
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

func (c *compiler) readElementDecl(elem *helium.Element, opts elementDeclReadOptions) (*ElementDecl, error) {
	decl := &ElementDecl{
		Name:      QName{Local: opts.name, NS: opts.namespace},
		MinOccurs: opts.minOccurs,
		MaxOccurs: opts.maxOccurs,
		Nillable:  getAttr(elem, attrNillable) == attrValTrue,
	}

	if opts.allowAbstract {
		decl.Abstract = getAttr(elem, attrAbstract) == attrValTrue
	}

	if opts.allowSubstitutionGroup {
		if sg := getAttr(elem, attrSubstitutionGroup); sg != "" {
			decl.SubstitutionGroup = c.resolveQName(elem, sg)
		}
	}

	decl.Default, decl.Fixed = readDefaultOrFixed(elem)

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

	if err := c.readElementType(elem, decl, opts.name); err != nil {
		return nil, err
	}
	decl.IDCs = c.parseIDConstraints(elem)
	return decl, nil
}

func (c *compiler) readElementType(elem *helium.Element, decl *ElementDecl, sourceName string) error {
	typeRef := getAttr(elem, attrType)
	if typeRef != "" {
		qn := c.resolveQName(elem, typeRef)
		c.elemRefs[decl] = qn
		c.elemRefSources[decl] = elemRefSource{elemName: sourceName, line: elem.Line()}
		return nil
	}

	for child := range helium.Children(elem) {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce := child.(*helium.Element)
		switch {
		case isXSDElement(ce, elemAnnotation):
			c.checkAnnotation(ce)
		case isXSDElement(ce, elemComplexType):
			td, err := c.parseComplexType(ce)
			if err != nil {
				return err
			}
			decl.Type = td
		case isXSDElement(ce, elemSimpleType):
			td, err := c.parseSimpleType(ce)
			if err != nil {
				return err
			}
			decl.Type = td
		}
	}

	return nil
}

func (c *compiler) readAttributeUseDecl(elem *helium.Element, opts attrUseReadOptions) *AttrUse {
	au := &AttrUse{Name: opts.name}
	if typeRef := getAttr(elem, attrType); typeRef != "" {
		au.TypeName = c.resolveQName(elem, typeRef)
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
	return au
}

func (c *compiler) parseGlobalElement(elem *helium.Element) error {
	c.checkGlobalElement(elem)
	name := getAttr(elem, attrName)
	if name == "" {
		// Still register with a placeholder name to continue parsing.
		return nil
	}

	decl, err := c.readElementDecl(elem, elementDeclReadOptions{
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
		c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaParserError(source, elem.Line(),
			elem.LocalName(), "element",
			"A global element declaration "+qnDisplay+" does already exist."), helium.ErrorLevelFatal))
		c.errorCount++
		return nil
	}

	c.globalElemSources[decl] = elemRefSource{elemName: name, line: elem.Line()}
	c.schema.elements[decl.Name] = decl
	return nil
}

func (c *compiler) parseLocalElement(elem *helium.Element) (*Particle, error) {
	c.checkLocalElement(elem)
	minOcc, maxOcc := parseParticleOccurs(elem)

	// Handle element references (ref="...").
	if ref := getAttr(elem, attrRef); ref != "" {
		qn := c.resolveQName(elem, ref)
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

	edecl, err := c.readElementDecl(elem, elementDeclReadOptions{
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

func (c *compiler) parseWildcard(elem *helium.Element) *Particle {
	minOcc, maxOcc := parseParticleOccurs(elem)
	wc := c.readWildcard(elem)
	return &Particle{
		MinOccurs: minOcc,
		MaxOccurs: maxOcc,
		Term:      wc,
	}
}

func (c *compiler) parseAnyAttribute(elem *helium.Element) *Wildcard {
	return c.readWildcard(elem)
}

func (c *compiler) parseGlobalAttribute(elem *helium.Element) {
	c.checkAttributeUse(elem)
	name := getAttr(elem, attrName)
	if name == "" {
		return
	}
	// Global attributes are always in the target namespace (per spec).
	au := c.readAttributeUseDecl(elem, attrUseReadOptions{
		name: QName{Local: name, NS: c.schema.targetNamespace},
	})
	c.schema.globalAttrs[au.Name] = au
}

func (c *compiler) parseAttributeUse(elem *helium.Element) *AttrUse {
	c.checkAttributeUse(elem)
	// Handle attribute references (ref="...").
	if ref := getAttr(elem, attrRef); ref != "" {
		qn := c.resolveQName(elem, ref)
		au := &AttrUse{Name: qn}
		if getAttr(elem, attrUse) == attrValRequired {
			au.Required = true
		}
		if v := getAttr(elem, attrDefault); v != "" {
			au.Default = &v
		}
		if v := getAttr(elem, attrFixed); v != "" {
			au.Fixed = &v
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
	return c.readAttributeUseDecl(elem, attrUseReadOptions{
		name:       QName{Local: name, NS: attrNS},
		includeUse: true,
	})
}

// parseIDConstraints scans element children for xs:key, xs:keyref, xs:unique declarations.
func (c *compiler) parseIDConstraints(elem *helium.Element) []*IDConstraint {
	var idcs []*IDConstraint
	for child := range helium.Children(elem) {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce := child.(*helium.Element)
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
		idc := c.parseIDConstraint(ce, kind)
		if idc != nil {
			idcs = append(idcs, idc)
		}
	}
	return idcs
}

// parseIDConstraint parses a single xs:key, xs:keyref, or xs:unique declaration.
func (c *compiler) parseIDConstraint(elem *helium.Element, kind IDCKind) *IDConstraint {
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
		ce := child.(*helium.Element)
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
