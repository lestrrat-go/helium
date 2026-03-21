package xsd

import (
	"fmt"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath1"
)

func (c *compiler) parseGlobalElement(elem *helium.Element) error {
	c.checkGlobalElement(elem)
	name := getAttr(elem, "name")
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

	// Check for duplicate global element declarations (e.g., from includes).
	if _, exists := c.schema.elements[decl.Name]; exists && c.includeFile != "" {
		qnDisplay := "'" + decl.Name.NS + "'" + decl.Name.Local
		if decl.Name.NS != "" {
			qnDisplay = "'{" + decl.Name.NS + "}" + decl.Name.Local + "'"
		}
		c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaParserError(c.includeFile, elem.Line(),
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
	if ref := getAttr(elem, "ref"); ref != "" {
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

	name := getAttr(elem, "name")
	if name == "" {
		return nil, fmt.Errorf("xsd: local element missing name")
	}

	// Determine element namespace based on form and elementFormDefault.
	elemNS := ""
	form := getAttr(elem, "form")
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
	name := getAttr(elem, "name")
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
	if ref := getAttr(elem, "ref"); ref != "" {
		qn := c.resolveQName(elem, ref)
		au := &AttrUse{Name: qn}
		if getAttr(elem, "use") == "required" {
			au.Required = true
		}
		if v := getAttr(elem, "default"); v != "" {
			au.Default = &v
		}
		if v := getAttr(elem, "fixed"); v != "" {
			au.Fixed = &v
		}
		c.attrRefs[au] = qn
		return au
	}

	name := getAttr(elem, "name")
	// Determine attribute namespace based on form and attributeFormDefault.
	attrNS := ""
	form := getAttr(elem, "form")
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
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce := child.(*helium.Element)
		var kind IDCKind
		switch {
		case isXSDElement(ce, "unique"):
			kind = IDCUnique
		case isXSDElement(ce, "key"):
			kind = IDCKey
		case isXSDElement(ce, "keyref"):
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
	name := getAttr(elem, "name")
	if name == "" {
		return nil
	}
	idc := &IDConstraint{
		Name:       name,
		Kind:       kind,
		Namespaces: collectNSContext(elem),
	}
	if kind == IDCKeyRef {
		idc.Refer = getAttr(elem, "refer")
	}
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce := child.(*helium.Element)
		switch {
		case isXSDElement(ce, "selector"):
			idc.Selector = getAttr(ce, "xpath")
		case isXSDElement(ce, "field"):
			idc.Fields = append(idc.Fields, getAttr(ce, "xpath"))
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
