package xsd

import helium "github.com/lestrrat-go/helium"

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

	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
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
