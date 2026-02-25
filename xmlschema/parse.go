package xmlschema

import (
	"fmt"
	"strconv"

	helium "github.com/lestrrat-go/helium"
)

const xsdNS = "http://www.w3.org/2001/XMLSchema"

// compiler holds state during schema compilation.
type compiler struct {
	schema *Schema
	// unresolved type references: maps from element/type QName to the type ref string
	typeRefs map[*TypeDef]QName
	elemRefs map[*ElementDecl]QName
}

func compileSchema(doc *helium.Document) (*Schema, error) {
	root := findDocumentElement(doc)
	if root == nil {
		return nil, fmt.Errorf("xmlschema: empty document")
	}

	if !isXSDElement(root, "schema") {
		return nil, fmt.Errorf("xmlschema: root element is not xs:schema")
	}

	c := &compiler{
		schema: &Schema{
			elements: make(map[QName]*ElementDecl),
			types:    make(map[QName]*TypeDef),
		},
		typeRefs: make(map[*TypeDef]QName),
		elemRefs: make(map[*ElementDecl]QName),
	}

	c.schema.targetNamespace = getAttr(root, "targetNamespace")

	// Register built-in types.
	registerBuiltinTypes(c.schema)

	// First pass: collect all named types and global elements.
	for child := root.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Type() != helium.ElementNode {
			continue
		}
		elem := child.(*helium.Element)
		switch {
		case isXSDElement(elem, "element"):
			if err := c.parseGlobalElement(elem); err != nil {
				return nil, err
			}
		case isXSDElement(elem, "complexType"):
			if err := c.parseNamedComplexType(elem); err != nil {
				return nil, err
			}
		case isXSDElement(elem, "simpleType"):
			if err := c.parseNamedSimpleType(elem); err != nil {
				return nil, err
			}
		}
	}

	// Second pass: resolve type references.
	if err := c.resolveRefs(); err != nil {
		return nil, err
	}

	return c.schema, nil
}

func (c *compiler) parseGlobalElement(elem *helium.Element) error {
	name := getAttr(elem, "name")
	if name == "" {
		return fmt.Errorf("xmlschema: global element missing name")
	}

	decl := &ElementDecl{
		Name:      QName{Local: name, NS: c.schema.targetNamespace},
		MinOccurs: 1,
		MaxOccurs: 1,
	}

	typeRef := getAttr(elem, "type")
	if typeRef != "" {
		qn := c.resolveQName(elem, typeRef)
		c.elemRefs[decl] = qn
	} else {
		// Look for inline complexType or simpleType.
		for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
			if child.Type() != helium.ElementNode {
				continue
			}
			ce := child.(*helium.Element)
			switch {
			case isXSDElement(ce, "complexType"):
				td, err := c.parseComplexType(ce)
				if err != nil {
					return err
				}
				decl.Type = td
			case isXSDElement(ce, "simpleType"):
				td, err := c.parseSimpleType(ce)
				if err != nil {
					return err
				}
				decl.Type = td
			}
		}
	}

	c.schema.elements[decl.Name] = decl
	return nil
}

func (c *compiler) parseNamedComplexType(elem *helium.Element) error {
	name := getAttr(elem, "name")
	if name == "" {
		return fmt.Errorf("xmlschema: named complexType missing name")
	}

	td, err := c.parseComplexType(elem)
	if err != nil {
		return err
	}
	td.Name = QName{Local: name, NS: c.schema.targetNamespace}
	c.schema.types[td.Name] = td
	return nil
}

func (c *compiler) parseNamedSimpleType(elem *helium.Element) error {
	name := getAttr(elem, "name")
	if name == "" {
		return fmt.Errorf("xmlschema: named simpleType missing name")
	}

	td, err := c.parseSimpleType(elem)
	if err != nil {
		return err
	}
	td.Name = QName{Local: name, NS: c.schema.targetNamespace}
	c.schema.types[td.Name] = td
	return nil
}

func (c *compiler) parseComplexType(elem *helium.Element) (*TypeDef, error) {
	td := &TypeDef{}

	mixed := getAttr(elem, "mixed")
	if mixed == "true" {
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
		case isXSDElement(ce, "complexContent"):
			if err := c.parseComplexContent(ce, td); err != nil {
				return nil, err
			}
		case isXSDElement(ce, "simpleContent"):
			td.ContentType = ContentTypeSimple
		case isXSDElement(ce, "attribute"):
			au := c.parseAttributeUse(ce)
			td.Attributes = append(td.Attributes, au)
		}
	}

	// If no content model and not mixed, check for empty.
	if td.ContentModel == nil && td.ContentType == ContentTypeEmpty {
		// Check if there are attribute declarations — if so, it's still empty content.
		// ContentTypeEmpty is the default (no children).
	}

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
		}
	}
	return nil
}

func (c *compiler) parseExtension(elem *helium.Element, td *TypeDef) error {
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
		case isXSDElement(ce, "attribute"):
			au := c.parseAttributeUse(ce)
			td.Attributes = append(td.Attributes, au)
		}
	}
	return nil
}

func (c *compiler) parseSimpleType(elem *helium.Element) (*TypeDef, error) {
	td := &TypeDef{
		ContentType: ContentTypeSimple,
	}
	return td, nil
}

func (c *compiler) parseModelGroup(elem *helium.Element, compositor ModelGroupKind) (*ModelGroup, error) {
	mg := &ModelGroup{
		Compositor: compositor,
		MinOccurs:  1,
		MaxOccurs:  1,
	}

	if v := getAttr(elem, "minOccurs"); v != "" {
		mg.MinOccurs = parseOccurs(v, 1)
	}
	if v := getAttr(elem, "maxOccurs"); v != "" {
		mg.MaxOccurs = parseOccurs(v, 1)
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
		}
	}

	return mg, nil
}

func (c *compiler) parseLocalElement(elem *helium.Element) (*Particle, error) {
	name := getAttr(elem, "name")
	if name == "" {
		return nil, fmt.Errorf("xmlschema: local element missing name")
	}

	minOcc := 1
	maxOcc := 1
	if v := getAttr(elem, "minOccurs"); v != "" {
		minOcc = parseOccurs(v, 1)
	}
	if v := getAttr(elem, "maxOccurs"); v != "" {
		maxOcc = parseOccurs(v, 1)
	}

	edecl := &ElementDecl{
		Name:      QName{Local: name, NS: c.schema.targetNamespace},
		MinOccurs: minOcc,
		MaxOccurs: maxOcc,
	}

	typeRef := getAttr(elem, "type")
	if typeRef != "" {
		qn := c.resolveQName(elem, typeRef)
		c.elemRefs[edecl] = qn
	} else {
		// Check for inline type definition.
		for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
			if child.Type() != helium.ElementNode {
				continue
			}
			ce := child.(*helium.Element)
			switch {
			case isXSDElement(ce, "complexType"):
				td, err := c.parseComplexType(ce)
				if err != nil {
					return nil, err
				}
				edecl.Type = td
			case isXSDElement(ce, "simpleType"):
				td, err := c.parseSimpleType(ce)
				if err != nil {
					return nil, err
				}
				edecl.Type = td
			}
		}
	}

	return &Particle{
		MinOccurs: minOcc,
		MaxOccurs: maxOcc,
		Term:      edecl,
	}, nil
}

func (c *compiler) parseAttributeUse(elem *helium.Element) *AttrUse {
	au := &AttrUse{}
	au.Name = QName{Local: getAttr(elem, "name")}
	if typeRef := getAttr(elem, "type"); typeRef != "" {
		au.TypeName = c.resolveQName(elem, typeRef)
	}
	if getAttr(elem, "use") == "required" {
		au.Required = true
	}
	return au
}

func (c *compiler) resolveRefs() error {
	for edecl, qn := range c.elemRefs {
		if edecl.Type != nil {
			continue
		}
		td, ok := c.schema.types[qn]
		if !ok {
			// For built-in types and unresolved refs, create a placeholder.
			td = &TypeDef{Name: qn, ContentType: ContentTypeSimple}
			c.schema.types[qn] = td
		}
		edecl.Type = td
	}
	for td, qn := range c.typeRefs {
		base, ok := c.schema.types[qn]
		if !ok {
			base = &TypeDef{Name: qn, ContentType: ContentTypeSimple}
			c.schema.types[qn] = base
		}
		td.BaseType = base
	}
	return nil
}

// resolveQName resolves a prefixed name (like "xsd:string") to a QName
// using the namespace declarations in scope on the given element.
func (c *compiler) resolveQName(elem *helium.Element, ref string) QName {
	local := ref
	ns := c.schema.targetNamespace

	for i := 0; i < len(ref); i++ {
		if ref[i] == ':' {
			prefix := ref[:i]
			local = ref[i+1:]
			ns = lookupNS(elem, prefix)
			break
		}
	}

	return QName{Local: local, NS: ns}
}

// helpers

func findDocumentElement(doc *helium.Document) *helium.Element {
	for child := doc.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Type() == helium.ElementNode {
			return child.(*helium.Element)
		}
	}
	return nil
}

func isXSDElement(elem *helium.Element, localName string) bool {
	return elem.LocalName() == localName && elem.URI() == xsdNS
}

func getAttr(elem *helium.Element, name string) string {
	for _, a := range elem.Attributes() {
		if a.LocalName() == name {
			return a.Value()
		}
	}
	return ""
}

func lookupNS(elem *helium.Element, prefix string) string {
	// Walk up the tree looking for namespace declarations.
	var node helium.Node = elem
	for node != nil {
		if e, ok := node.(*helium.Element); ok {
			for _, ns := range e.Namespaces() {
				if ns.Prefix() == prefix {
					return ns.URI()
				}
			}
			// Also check the element's own namespace.
			if e.Prefix() == prefix {
				return e.URI()
			}
		}
		node = node.Parent()
	}
	return ""
}

func parseOccurs(s string, defaultVal int) int {
	if s == "unbounded" {
		return Unbounded
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return n
}

func registerBuiltinTypes(s *Schema) {
	builtins := []string{
		"string", "boolean", "decimal", "float", "double",
		"integer", "nonPositiveInteger", "negativeInteger",
		"long", "int", "short", "byte",
		"nonNegativeInteger", "unsignedLong", "unsignedInt", "unsignedShort", "unsignedByte",
		"positiveInteger",
		"normalizedString", "token", "language", "Name", "NCName",
		"ID", "IDREF", "IDREFS", "ENTITY", "ENTITIES", "NMTOKEN", "NMTOKENS",
		"date", "dateTime", "time", "duration",
		"gYearMonth", "gYear", "gMonthDay", "gDay", "gMonth",
		"hexBinary", "base64Binary",
		"anyURI", "QName", "NOTATION",
		"anyType", "anySimpleType",
	}
	for _, name := range builtins {
		qn := QName{Local: name, NS: xsdNS}
		ct := ContentTypeSimple
		if name == "anyType" {
			ct = ContentTypeMixed // xs:anyType allows any content
		}
		s.types[qn] = &TypeDef{
			Name:        qn,
			ContentType: ct,
		}
	}
}
