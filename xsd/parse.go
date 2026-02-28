package xsd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	helium "github.com/lestrrat-go/helium"
)

const xsdNS = "http://www.w3.org/2001/XMLSchema"

// attrValTrue and attrValQualified are common XML attribute value strings.
const (
	attrValTrue      = "true"
	attrValQualified = "qualified"
)

// compiler holds state during schema compilation.
type compiler struct {
	schema  *Schema
	baseDir string // directory of the schema file, for resolving relative paths
	// unresolved type references: maps from element/type QName to the type ref string
	typeRefs map[*TypeDef]QName
	elemRefs map[*ElementDecl]QName
	// source info for element refs, used in unresolved-type error messages
	elemRefSources map[*ElementDecl]elemRefSource
	// unresolved group references: maps from model group placeholder to group QName
	groupRefs map[*ModelGroup]QName
	// unresolved attribute group references: maps from TypeDef to list of QNames
	attrGroupRefs map[*TypeDef][]QName
	// source info for global elements, used in substitution group error messages
	globalElemSources map[*ElementDecl]elemRefSource
	// source info for type definitions, used in duplicate attribute errors
	typeDefSources map[*TypeDef]typeDefSource
	// unresolved item type references for list types
	itemTypeRefs map[*TypeDef]QName
	// unresolved union member type references
	unionMemberRefs []unionMemberRef
	// unresolved attribute references: maps from AttrUse to global attr QName
	attrRefs map[*AttrUse]QName
	// schema error/warning collection
	schemaErrors   strings.Builder
	schemaWarnings strings.Builder
	filename       string // XSD filename for error messages
	includeFile    string // currently-included file path (for duplicate element errors)
	// importedNS tracks which namespaces have been imported and their schema locations.
	importedNS map[string]string // namespace → schema location
}

// elemRefSource tracks source location for error reporting.
type elemRefSource struct {
	elemName string
	line     int
}

// unionMemberRef tracks an unresolved union member type reference.
type unionMemberRef struct {
	owner *TypeDef
	name  QName
}

// typeDefSource tracks source location and context for type definitions.
type typeDefSource struct {
	line    int
	isLocal bool // true for anonymous (local) complex types
}

func compileSchema(doc *helium.Document, baseDir string, cfg *compileConfig) (*Schema, error) {
	root := findDocumentElement(doc)
	if root == nil {
		return nil, fmt.Errorf("xsd: empty document")
	}

	if !isXSDElement(root, "schema") {
		return nil, fmt.Errorf("xsd: root element is not xs:schema")
	}

	c := &compiler{
		schema: &Schema{
			elements:    make(map[QName]*ElementDecl),
			types:       make(map[QName]*TypeDef),
			groups:      make(map[QName]*ModelGroup),
			attrGroups:  make(map[QName][]*AttrUse),
			globalAttrs: make(map[QName]*AttrUse),
			substGroups: make(map[QName][]*ElementDecl),
		},
		baseDir:           baseDir,
		typeRefs:          make(map[*TypeDef]QName),
		elemRefs:          make(map[*ElementDecl]QName),
		elemRefSources:    make(map[*ElementDecl]elemRefSource),
		groupRefs:         make(map[*ModelGroup]QName),
		attrGroupRefs:     make(map[*TypeDef][]QName),
		globalElemSources: make(map[*ElementDecl]elemRefSource),
		typeDefSources:    make(map[*TypeDef]typeDefSource),
		itemTypeRefs:      make(map[*TypeDef]QName),
		attrRefs:          make(map[*AttrUse]QName),
		importedNS:        make(map[string]string),
	}
	if cfg != nil {
		c.filename = cfg.filename
	}

	c.schema.targetNamespace = getAttr(root, "targetNamespace")
	c.schema.elemFormQualified = getAttr(root, "elementFormDefault") == attrValQualified
	c.schema.attrFormQualified = getAttr(root, "attributeFormDefault") == attrValQualified

	// Register built-in types.
	registerBuiltinTypes(c.schema)

	// First pass: collect all named types and global elements.
	if err := c.parseSchemaChildren(root); err != nil {
		return nil, err
	}

	// Process includes after parsing the main schema's declarations.
	// This matches libxml2's processing order where includes are merged
	// after the including schema's own declarations are registered.
	if err := c.processIncludes(root); err != nil {
		return nil, err
	}

	// Second pass: resolve type references.
	c.resolveRefs()

	// Build substitution group membership map and detect circular references.
	for _, edecl := range c.schema.elements {
		if edecl.SubstitutionGroup == (QName{}) {
			continue
		}
		head := edecl.SubstitutionGroup
		c.schema.substGroups[head] = append(c.schema.substGroups[head], edecl)

		// Check for circular substitution groups.
		if c.filename != "" {
			c.checkCircularSubstGroup(edecl)
		}
	}

	// Sort substitution group members for deterministic error messages.
	for _, members := range c.schema.substGroups {
		sort.Slice(members, func(i, j int) bool {
			return members[i].Name.Local < members[j].Name.Local
		})
	}

	c.schema.compileErrors = c.schemaErrors.String()
	c.schema.compileWarnings = c.schemaWarnings.String()
	return c.schema, nil
}

// parseSchemaChildren parses the children of an xs:schema element.
func (c *compiler) parseSchemaChildren(root *helium.Element) error {
	for child := root.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Type() != helium.ElementNode {
			continue
		}
		elem := child.(*helium.Element)
		switch {
		case isXSDElement(elem, "element"):
			if err := c.parseGlobalElement(elem); err != nil {
				return err
			}
		case isXSDElement(elem, "complexType"):
			if err := c.parseNamedComplexType(elem); err != nil {
				return err
			}
		case isXSDElement(elem, "simpleType"):
			if err := c.parseNamedSimpleType(elem); err != nil {
				return err
			}
		case isXSDElement(elem, "group"):
			if err := c.parseNamedGroup(elem); err != nil {
				return err
			}
		case isXSDElement(elem, "attributeGroup"):
			if err := c.parseNamedAttributeGroup(elem); err != nil {
				return err
			}
		case isXSDElement(elem, "attribute"):
			c.parseGlobalAttribute(elem)
		}
	}
	return nil
}

// processIncludesAndImports handles xs:include and xs:import elements.
func (c *compiler) processIncludes(root *helium.Element) error {
	for child := root.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Type() != helium.ElementNode {
			continue
		}
		elem := child.(*helium.Element)
		switch {
		case isXSDElement(elem, "include"):
			loc := getAttr(elem, "schemaLocation")
			if loc == "" {
				continue
			}
			if err := c.loadInclude(loc, elem); err != nil {
				return err
			}
		case isXSDElement(elem, "import"):
			loc := getAttr(elem, "schemaLocation")
			if loc == "" {
				continue
			}
			ns := getAttr(elem, "namespace")

			// Check if this namespace was already imported.
			if prevLoc, ok := c.importedNS[ns]; ok && c.filename != "" {
				displayLoc := filepath.Join(filepath.Dir(c.filename), loc)
				displayPrevLoc := filepath.Join(filepath.Dir(c.filename), prevLoc)
				c.schemaWarnings.WriteString(schemaParserWarning(c.filename, elem.Line(),
					elem.LocalName(), "import",
					"Skipping import of schema located at '"+displayLoc+"' for the namespace '"+ns+"', since this namespace was already imported with the schema located at '"+displayPrevLoc+"'."))
				continue
			}

			if err := c.loadImport(loc, ns); err != nil {
				// Import failure — report warning if we have a filename.
				if c.filename != "" {
					displayLoc := filepath.Join(filepath.Dir(c.filename), loc)
					fmt.Fprintf(&c.schemaWarnings, "I/O warning : failed to load \"%s\": %s\n", displayLoc, "No such file or directory")
					c.schemaWarnings.WriteString(schemaParserWarning(c.filename, elem.Line(),
						elem.LocalName(), "import",
						"Failed to locate a schema at location '"+displayLoc+"'. Skipping the import."))
				}
				continue
			}

			// Track the imported namespace.
			c.importedNS[ns] = loc
		}
	}
	return nil
}

// loadInclude loads and merges an included schema file.
func (c *compiler) loadInclude(location string, includeElem *helium.Element) error {
	path := location
	if c.baseDir != "" {
		path = filepath.Join(c.baseDir, location)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("xsd: failed to load include %q: %w", location, err)
	}

	doc, err := helium.Parse(data)
	if err != nil {
		return fmt.Errorf("xsd: failed to parse include %q: %w", location, err)
	}

	incRoot := findDocumentElement(doc)
	if incRoot == nil || !isXSDElement(incRoot, "schema") {
		return fmt.Errorf("xsd: included document %q is not an xs:schema", location)
	}

	// Check target namespace compatibility.
	incTargetNS := getAttr(incRoot, "targetNamespace")
	if incTargetNS != "" && incTargetNS != c.schema.targetNamespace {
		displayLoc := location
		if c.filename != "" {
			displayLoc = filepath.Join(filepath.Dir(c.filename), location)
		}
		c.schemaErrors.WriteString(schemaParserError(c.filename, includeElem.Line(),
			includeElem.LocalName(), "include",
			"The target namespace '"+incTargetNS+"' of the included/redefined schema '"+displayLoc+"' differs from '"+c.schema.targetNamespace+"' of the including/redefining schema."))
		return nil
	}

	// Chameleon include: if the included schema has no targetNamespace,
	// it adopts the including schema's targetNamespace.
	// The included schema's elementFormDefault/attributeFormDefault are
	// applied within the included declarations.

	// Save current form-qualified settings and temporarily apply included schema's settings.
	savedElemForm := c.schema.elemFormQualified
	savedAttrForm := c.schema.attrFormQualified
	savedIncludeFile := c.includeFile
	if v := getAttr(incRoot, "elementFormDefault"); v != "" {
		c.schema.elemFormQualified = v == attrValQualified
	}
	if v := getAttr(incRoot, "attributeFormDefault"); v != "" {
		c.schema.attrFormQualified = v == attrValQualified
	}

	// Set the include file path for duplicate element error reporting.
	if c.filename != "" {
		c.includeFile = filepath.Join(filepath.Dir(c.filename), location)
	}

	// Parse the included schema's declarations into the current compiler.
	err = c.parseSchemaChildren(incRoot)

	// Restore form-qualified settings and include file.
	c.schema.elemFormQualified = savedElemForm
	c.schema.attrFormQualified = savedAttrForm
	c.includeFile = savedIncludeFile

	return err
}

// loadImport loads an imported schema and merges its declarations.
func (c *compiler) loadImport(location, _ string) error {
	path := location
	if c.baseDir != "" {
		path = filepath.Join(c.baseDir, location)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	doc, err := helium.Parse(data)
	if err != nil {
		return err
	}

	impRoot := findDocumentElement(doc)
	if impRoot == nil || !isXSDElement(impRoot, "schema") {
		return fmt.Errorf("xsd: imported document %q is not an xs:schema", location)
	}

	// Compute display filename for the imported schema (for error messages).
	var impFilename string
	if c.filename != "" {
		impFilename = filepath.Join(filepath.Dir(c.filename), location)
	}

	// Create a temporary compiler for the imported schema.
	impC := &compiler{
		schema: &Schema{
			elements:    make(map[QName]*ElementDecl),
			types:       make(map[QName]*TypeDef),
			groups:      make(map[QName]*ModelGroup),
			attrGroups:  make(map[QName][]*AttrUse),
			globalAttrs: make(map[QName]*AttrUse),
			substGroups: make(map[QName][]*ElementDecl),
		},
		baseDir:           filepath.Dir(path),
		typeRefs:          make(map[*TypeDef]QName),
		elemRefs:          make(map[*ElementDecl]QName),
		elemRefSources:    make(map[*ElementDecl]elemRefSource),
		groupRefs:         make(map[*ModelGroup]QName),
		attrGroupRefs:     make(map[*TypeDef][]QName),
		globalElemSources: make(map[*ElementDecl]elemRefSource),
		typeDefSources:    make(map[*TypeDef]typeDefSource),
		itemTypeRefs:      make(map[*TypeDef]QName),
		attrRefs:          make(map[*AttrUse]QName),
		filename:          impFilename,
		importedNS:        make(map[string]string),
	}

	impC.schema.targetNamespace = getAttr(impRoot, "targetNamespace")
	impC.schema.elemFormQualified = getAttr(impRoot, "elementFormDefault") == attrValQualified
	impC.schema.attrFormQualified = getAttr(impRoot, "attributeFormDefault") == attrValQualified

	registerBuiltinTypes(impC.schema)

	if err := impC.parseSchemaChildren(impRoot); err != nil {
		return err
	}

	// Process includes/imports in the imported schema (but skip back-references).
	if err := impC.processIncludes(impRoot); err != nil {
		// Non-fatal for imported schemas.
		_ = err
	}

	// Propagate compile errors from the imported schema's parsing phase.
	// Only propagate if there are no prior errors (matches libxml2 behavior
	// of stopping error reporting after first import failure).
	if impC.schemaErrors.Len() > 0 {
		if c.schemaErrors.Len() == 0 {
			c.schemaErrors.WriteString(impC.schemaErrors.String())
		}
		return nil
	}

	// Merge the imported schema's declarations into the main schema.
	for qn, edecl := range impC.schema.elements {
		if _, exists := c.schema.elements[qn]; !exists {
			c.schema.elements[qn] = edecl
		}
	}
	for qn, td := range impC.schema.types {
		if _, exists := c.schema.types[qn]; !exists {
			c.schema.types[qn] = td
		}
	}
	for qn, mg := range impC.schema.groups {
		if _, exists := c.schema.groups[qn]; !exists {
			c.schema.groups[qn] = mg
		}
	}
	for qn, attrs := range impC.schema.attrGroups {
		if _, exists := c.schema.attrGroups[qn]; !exists {
			c.schema.attrGroups[qn] = attrs
		}
	}
	for qn, au := range impC.schema.globalAttrs {
		if _, exists := c.schema.globalAttrs[qn]; !exists {
			c.schema.globalAttrs[qn] = au
		}
	}

	// Merge ref maps from the sub-compiler into the parent compiler.
	// This defers resolution to the parent's resolveRefs(), which has
	// access to all merged declarations (handles circular imports).
	for edecl, qn := range impC.elemRefs {
		c.elemRefs[edecl] = qn
	}
	for edecl, src := range impC.elemRefSources {
		c.elemRefSources[edecl] = src
	}
	for td, qn := range impC.typeRefs {
		c.typeRefs[td] = qn
	}
	for td, src := range impC.typeDefSources {
		c.typeDefSources[td] = src
	}
	for mg, qn := range impC.groupRefs {
		c.groupRefs[mg] = qn
	}
	for td, qns := range impC.attrGroupRefs {
		c.attrGroupRefs[td] = qns
	}
	for edecl, src := range impC.globalElemSources {
		c.globalElemSources[edecl] = src
	}
	for td, qn := range impC.itemTypeRefs {
		c.itemTypeRefs[td] = qn
	}
	c.unionMemberRefs = append(c.unionMemberRefs, impC.unionMemberRefs...)
	for au, qn := range impC.attrRefs {
		c.attrRefs[au] = qn
	}

	return nil
}

func (c *compiler) parseGlobalElement(elem *helium.Element) error {
	c.checkGlobalElement(elem)
	name := getAttr(elem, "name")
	if name == "" {
		// Still register with a placeholder name to continue parsing.
		return nil
	}

	decl := &ElementDecl{
		Name:      QName{Local: name, NS: c.schema.targetNamespace},
		MinOccurs: 1,
		MaxOccurs: 1,
		Abstract:  getAttr(elem, "abstract") == attrValTrue,
		Nillable:  getAttr(elem, "nillable") == attrValTrue,
	}

	if sg := getAttr(elem, "substitutionGroup"); sg != "" {
		decl.SubstitutionGroup = c.resolveQName(elem, sg)
	}
	if v := getAttr(elem, "default"); v != "" {
		decl.Default = &v
	}
	if v := getAttr(elem, "fixed"); v != "" {
		decl.Fixed = &v
	}

	typeRef := getAttr(elem, "type")
	if typeRef != "" {
		qn := c.resolveQName(elem, typeRef)
		c.elemRefs[decl] = qn
		c.elemRefSources[decl] = elemRefSource{elemName: name, line: elem.Line()}
	} else {
		// Look for inline complexType or simpleType.
		for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
			if child.Type() != helium.ElementNode {
				continue
			}
			ce := child.(*helium.Element)
			switch {
			case isXSDElement(ce, "annotation"):
				c.checkAnnotation(ce)
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

	// Parse IDC (identity constraint) declarations.
	decl.IDCs = c.parseIDConstraints(elem)

	// Check for duplicate global element declarations (e.g., from includes).
	if _, exists := c.schema.elements[decl.Name]; exists && c.includeFile != "" {
		qnDisplay := "'" + decl.Name.NS + "'" + decl.Name.Local
		if decl.Name.NS != "" {
			qnDisplay = "'{" + decl.Name.NS + "}" + decl.Name.Local + "'"
		}
		c.schemaErrors.WriteString(schemaParserError(c.includeFile, elem.Line(),
			elem.LocalName(), "element",
			"A global element declaration "+qnDisplay+" does already exist."))
		return nil
	}

	c.globalElemSources[decl] = elemRefSource{elemName: name, line: elem.Line()}
	c.schema.elements[decl.Name] = decl
	return nil
}

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
					c.schemaWarnings.WriteString(schemaParserWarning(c.filename, ce.Line(), ce.LocalName(), "attribute",
						"Skipping attribute use prohibition, since it is pointless when extending a type."))
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
		case isXSDElement(ce, "any"):
			p := c.parseWildcard(ce)
			mg.Particles = append(mg.Particles, p)
		case isXSDElement(ce, "group"):
			ref := getAttr(ce, "ref")
			if ref != "" {
				// Group reference — create a placeholder model group.
				placeholder := &ModelGroup{
					MinOccurs: 1,
					MaxOccurs: 1,
				}
				if v := getAttr(ce, "minOccurs"); v != "" {
					placeholder.MinOccurs = parseOccurs(v, 1)
				}
				if v := getAttr(ce, "maxOccurs"); v != "" {
					placeholder.MaxOccurs = parseOccurs(v, 1)
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

func (c *compiler) parseLocalElement(elem *helium.Element) (*Particle, error) {
	c.checkLocalElement(elem)
	minOcc := 1
	maxOcc := 1
	if v := getAttr(elem, "minOccurs"); v != "" {
		minOcc = parseOccurs(v, 1)
	}
	if v := getAttr(elem, "maxOccurs"); v != "" {
		maxOcc = parseOccurs(v, 1)
	}

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

	edecl := &ElementDecl{
		Name:      QName{Local: name, NS: elemNS},
		MinOccurs: minOcc,
		MaxOccurs: maxOcc,
		Nillable:  getAttr(elem, "nillable") == attrValTrue,
	}
	if v := getAttr(elem, "default"); v != "" {
		edecl.Default = &v
	}
	if v := getAttr(elem, "fixed"); v != "" {
		edecl.Fixed = &v
	}

	typeRef := getAttr(elem, "type")
	if typeRef != "" {
		qn := c.resolveQName(elem, typeRef)
		c.elemRefs[edecl] = qn
		c.elemRefSources[edecl] = elemRefSource{elemName: name, line: elem.Line()}
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

	// Parse IDC (identity constraint) declarations on local elements.
	edecl.IDCs = c.parseIDConstraints(elem)

	return &Particle{
		MinOccurs: minOcc,
		MaxOccurs: maxOcc,
		Term:      edecl,
	}, nil
}

func (c *compiler) parseWildcard(elem *helium.Element) *Particle {
	minOcc := 1
	maxOcc := 1
	if v := getAttr(elem, "minOccurs"); v != "" {
		minOcc = parseOccurs(v, 1)
	}
	if v := getAttr(elem, "maxOccurs"); v != "" {
		maxOcc = parseOccurs(v, 1)
	}

	ns := getAttr(elem, "namespace")
	if ns == "" {
		ns = WildcardNSAny
	}

	pc := ProcessStrict
	switch getAttr(elem, "processContents") {
	case "lax":
		pc = ProcessLax
	case "skip":
		pc = ProcessSkip
	}

	wc := &Wildcard{
		Namespace:       ns,
		ProcessContents: pc,
		TargetNS:        c.schema.targetNamespace,
	}
	return &Particle{
		MinOccurs: minOcc,
		MaxOccurs: maxOcc,
		Term:      wc,
	}
}

func (c *compiler) parseAnyAttribute(elem *helium.Element) *Wildcard {
	ns := getAttr(elem, "namespace")
	if ns == "" {
		ns = WildcardNSAny
	}

	pc := ProcessStrict
	switch getAttr(elem, "processContents") {
	case "lax":
		pc = ProcessLax
	case "skip":
		pc = ProcessSkip
	}

	return &Wildcard{
		Namespace:       ns,
		ProcessContents: pc,
		TargetNS:        c.schema.targetNamespace,
	}
}

func (c *compiler) parseGlobalAttribute(elem *helium.Element) {
	c.checkAttributeUse(elem)
	name := getAttr(elem, "name")
	if name == "" {
		return
	}
	// Global attributes are always in the target namespace (per spec).
	au := &AttrUse{
		Name: QName{Local: name, NS: c.schema.targetNamespace},
	}
	if typeRef := getAttr(elem, "type"); typeRef != "" {
		au.TypeName = c.resolveQName(elem, typeRef)
	}
	if v := getAttr(elem, "default"); v != "" {
		au.Default = &v
	}
	if v := getAttr(elem, "fixed"); v != "" {
		au.Fixed = &v
	}
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

	au := &AttrUse{}
	name := getAttr(elem, "name")
	// Determine attribute namespace based on form and attributeFormDefault.
	attrNS := ""
	form := getAttr(elem, "form")
	if form == attrValQualified || (form == "" && c.schema.attrFormQualified) {
		attrNS = c.schema.targetNamespace
	}
	au.Name = QName{Local: name, NS: attrNS}
	if typeRef := getAttr(elem, "type"); typeRef != "" {
		au.TypeName = c.resolveQName(elem, typeRef)
	}
	switch getAttr(elem, "use") {
	case "required":
		au.Required = true
	case "prohibited":
		au.Prohibited = true
	}
	if v := getAttr(elem, "default"); v != "" {
		au.Default = &v
	}
	if v := getAttr(elem, "fixed"); v != "" {
		au.Fixed = &v
	}
	return au
}

func (c *compiler) resolveRefs() {
	// Resolve element type references.
	// Two passes: the first pass resolves type-name refs and may leave
	// element-to-element refs with nil Type (because the target global element
	// hasn't had its own type resolved yet). The second pass picks those up.
	for range 2 {
		for edecl, qn := range c.elemRefs {
			if edecl.Type != nil {
				continue
			}
			// First check if this is a reference to a global element.
			if ge, ok := c.schema.elements[qn]; ok {
				edecl.Type = ge.Type
				if edecl.Default == nil {
					edecl.Default = ge.Default
				}
				if edecl.Fixed == nil {
					edecl.Fixed = ge.Fixed
				}
				edecl.Nillable = ge.Nillable
				continue
			}
			// For ref elements, report unresolved element declaration error.
			if edecl.IsRef {
				if src, hasSrc := c.elemRefSources[edecl]; hasSrc && c.filename != "" {
					msg := fmt.Sprintf("The QName value '{%s}%s' does not resolve to a(n) element declaration.", qn.NS, qn.Local)
					c.schemaErrors.WriteString(schemaParserErrorAttr(c.filename, src.line, src.elemName, "element", "ref", msg))
				}
				edecl.Type = &TypeDef{Name: qn, ContentType: ContentTypeSimple}
				continue
			}
			td, ok := c.schema.types[qn]
			if !ok {
				// Report unresolved type error for XSD built-in types that should exist.
				if qn.NS == xsdNS {
					if src, hasSrc := c.elemRefSources[edecl]; hasSrc && c.filename != "" {
						msg := fmt.Sprintf("The QName value '{%s}%s' does not resolve to a(n) type definition.", qn.NS, qn.Local)
						c.schemaErrors.WriteString(schemaElemDeclErrorAttr(c.filename, src.line, src.elemName, "type", msg))
					}
				}
				td = &TypeDef{Name: qn, ContentType: ContentTypeSimple}
				c.schema.types[qn] = td
			}
			edecl.Type = td
		}
	}

	// Resolve base type references.
	for td, qn := range c.typeRefs {
		base, ok := c.schema.types[qn]
		if !ok && qn.NS != "" {
			// Try empty namespace as fallback — the type may come from an
			// imported schema with no targetNamespace.
			base, ok = c.schema.types[QName{Local: qn.Local, NS: ""}]
		}
		if !ok {
			base = &TypeDef{Name: qn, ContentType: ContentTypeSimple}
			c.schema.types[qn] = base
		}
		td.BaseType = base
	}

	// Resolve list item type references.
	for td, qn := range c.itemTypeRefs {
		itemTD, ok := c.schema.types[qn]
		if !ok {
			itemTD = &TypeDef{Name: qn, ContentType: ContentTypeSimple}
			c.schema.types[qn] = itemTD
		}
		td.ItemType = itemTD
	}

	// Resolve union member type references.
	for _, ref := range c.unionMemberRefs {
		memberTD, ok := c.schema.types[ref.name]
		if !ok {
			memberTD = &TypeDef{Name: ref.name, ContentType: ContentTypeSimple}
			c.schema.types[ref.name] = memberTD
		}
		ref.owner.MemberTypes = append(ref.owner.MemberTypes, memberTD)
	}

	// Propagate variety and item type through restriction derivation.
	for td := range c.typeRefs {
		if td.Derivation == DerivationRestriction && td.BaseType != nil {
			if td.Variety == TypeVarietyAtomic && td.BaseType.Variety == TypeVarietyList {
				td.Variety = TypeVarietyList
				td.ItemType = td.BaseType.ItemType
			}
		}
	}

	// Propagate variety and member types through restriction derivation of union types.
	for td := range c.typeRefs {
		if td.Derivation == DerivationRestriction && td.BaseType != nil {
			if td.Variety == TypeVarietyAtomic && resolveVariety(td.BaseType) == TypeVarietyUnion {
				td.Variety = TypeVarietyUnion
				if len(td.MemberTypes) == 0 {
					td.MemberTypes = resolveUnionMembers(td.BaseType)
				}
			}
		}
	}

	// Resolve group references — replace placeholder content with actual group content.
	for placeholder, qn := range c.groupRefs {
		grp, ok := c.schema.groups[qn]
		if !ok {
			continue
		}
		// Copy the group's content into the placeholder.
		placeholder.Compositor = grp.Compositor
		placeholder.Particles = grp.Particles
	}

	// Resolve attribute group references.
	for td, qns := range c.attrGroupRefs {
		for _, qn := range qns {
			if attrs, ok := c.schema.attrGroups[qn]; ok {
				td.Attributes = append(td.Attributes, attrs...)
			}
		}
	}

	// Resolve attribute references: copy Default/Fixed/TypeName from global attr.
	for au, qn := range c.attrRefs {
		ga, ok := c.schema.globalAttrs[qn]
		if !ok {
			continue
		}
		if au.Default == nil {
			au.Default = ga.Default
		}
		if au.Fixed == nil {
			au.Fixed = ga.Fixed
		}
		if au.TypeName == (QName{}) {
			au.TypeName = ga.TypeName
		}
	}

	// Merge content models for extension types.
	for td := range c.typeRefs {
		if td.Derivation != DerivationExtension || td.BaseType == nil {
			continue
		}
		if td.ContentType == ContentTypeSimple {
			// simpleContent extension — inherit attributes and wildcard from base.
			if td.BaseType.Attributes != nil {
				td.Attributes = append(td.BaseType.Attributes, td.Attributes...)
			}
			if td.AnyAttribute == nil && td.BaseType.AnyAttribute != nil {
				td.AnyAttribute = td.BaseType.AnyAttribute
			} else if td.AnyAttribute != nil && td.BaseType.AnyAttribute != nil {
				td.AnyAttribute = wildcardUnion(td.BaseType.AnyAttribute, td.AnyAttribute)
			}
			continue
		}
		// cos-ct-extends-1-1: complexContent extension requires the base type
		// to also have complex content (mixed or element-only), not simple content.
		// Only check when the derived type has element content (not empty/attribute-only).
		if td.BaseType.ContentType == ContentTypeSimple && (td.ContentType == ContentTypeElementOnly || td.ContentType == ContentTypeMixed) {
			if src, ok := c.typeDefSources[td]; ok && c.filename != "" {
				component := "local complex type"
				if !src.isLocal {
					component = "complex type '" + td.Name.Local + "'"
				}
				c.schemaErrors.WriteString(schemaComponentError(c.filename, src.line, "complexType", component,
					"The content type of both, the type and its base type, must either 'mixed' or 'element-only'."))
			}
			continue
		}
		baseMG := td.BaseType.ContentModel
		derivedMG := td.ContentModel
		if baseMG != nil && derivedMG != nil {
			// Merge: create a sequence of base content + derived content.
			merged := &ModelGroup{
				Compositor: CompositorSequence,
				MinOccurs:  1,
				MaxOccurs:  1,
				Particles: []*Particle{
					{MinOccurs: baseMG.MinOccurs, MaxOccurs: baseMG.MaxOccurs, Term: baseMG},
					{MinOccurs: derivedMG.MinOccurs, MaxOccurs: derivedMG.MaxOccurs, Term: derivedMG},
				},
			}
			td.ContentModel = merged
		} else if baseMG != nil {
			td.ContentModel = baseMG
		}
		// Inherit content type from base if not already set.
		if td.ContentType == ContentTypeEmpty && td.BaseType.ContentType != ContentTypeEmpty {
			td.ContentType = td.BaseType.ContentType
		}
		// Check for duplicate attributes before merging base type attributes.
		if td.BaseType.Attributes != nil && td.Attributes != nil && c.filename != "" {
			baseAttrNames := make(map[string]bool, len(td.BaseType.Attributes))
			for _, au := range td.BaseType.Attributes {
				baseAttrNames[au.Name.Local] = true
			}
			for _, au := range td.Attributes {
				if baseAttrNames[au.Name.Local] {
					if src, ok := c.typeDefSources[td]; ok {
						component := "local complex type"
						if !src.isLocal {
							component = td.Name.Local
						}
						msg := fmt.Sprintf("Duplicate attribute use '%s'.", au.Name.Local)
						c.schemaErrors.WriteString(schemaComponentError(c.filename, src.line, "complexType", component, msg))
					}
				}
			}
		}
		// Inherit attributes from base.
		if td.BaseType.Attributes != nil {
			td.Attributes = append(td.BaseType.Attributes, td.Attributes...)
		}
		// Inherit/union anyAttribute wildcards.
		if td.AnyAttribute == nil && td.BaseType.AnyAttribute != nil {
			td.AnyAttribute = td.BaseType.AnyAttribute
		} else if td.AnyAttribute != nil && td.BaseType.AnyAttribute != nil {
			td.AnyAttribute = wildcardUnion(td.BaseType.AnyAttribute, td.AnyAttribute)
		}
	}

	// Check restriction attribute compatibility.
	// Collect and sort by source line for deterministic error ordering.
	var restrictionTypes []*TypeDef
	for td := range c.typeRefs {
		if td.Derivation != DerivationRestriction || td.BaseType == nil {
			continue
		}
		restrictionTypes = append(restrictionTypes, td)
	}
	sort.Slice(restrictionTypes, func(i, j int) bool {
		si := c.typeDefSources[restrictionTypes[i]]
		sj := c.typeDefSources[restrictionTypes[j]]
		return si.line < sj.line
	})
	for _, td := range restrictionTypes {
		c.checkRestrictionAttrs(td)
	}

	// Check UPA (Unique Particle Attribution) for all complex types with content models.
	// Only run UPA if there are no prior schema errors (libxml2 skips UPA when
	// the schema has structural parse errors).
	if c.filename != "" && c.schemaErrors.Len() == 0 {
		for td, src := range c.typeDefSources {
			if td.ContentModel != nil {
				c.checkUPA(td, src)
			}
		}
	}

}

// checkRestrictionAttrs validates that a restriction-derived type's attributes
// are compatible with the base type's attribute uses.
func (c *compiler) checkRestrictionAttrs(td *TypeDef) {
	if c.filename == "" {
		return
	}
	src, hasSrc := c.typeDefSources[td]
	if !hasSrc {
		return
	}

	component := "local complex type"
	if !src.isLocal {
		component = "complex type '" + td.Name.Local + "'"
	}

	baseTypeName := td.BaseType.Name.Local
	baseTypeNS := td.BaseType.Name.NS
	baseQualified := fmt.Sprintf("'{%s}%s'", baseTypeNS, baseTypeName)

	// Build map of base type's non-prohibited attributes.
	baseAttrs := make(map[string]*AttrUse, len(td.BaseType.Attributes))
	for _, au := range td.BaseType.Attributes {
		if !au.Prohibited {
			baseAttrs[au.Name.Local] = au
		}
	}

	// Check each derived non-prohibited attribute against the base.
	for _, au := range td.Attributes {
		if au.Prohibited {
			continue
		}
		baseAU, found := baseAttrs[au.Name.Local]
		if found {
			// Check use consistency: optional cannot restrict required.
			if baseAU.Required && !au.Required {
				msg := fmt.Sprintf("The 'optional' attribute use is inconsistent with the corresponding 'required' attribute use of the base complex type definition %s.", baseQualified)
				c.schemaErrors.WriteString(schemaComponentError(c.filename, src.line, "complexType",
					component+", attribute use '"+au.Name.Local+"'", msg))
			}
		} else if td.BaseType.AnyAttribute == nil {
			// No matching attribute and no wildcard in base.
			msg := fmt.Sprintf("Neither a matching attribute use, nor a matching wildcard exists in the base complex type definition %s.", baseQualified)
			c.schemaErrors.WriteString(schemaComponentError(c.filename, src.line, "complexType",
				component+", attribute use '"+au.Name.Local+"'", msg))
		}
	}

	// Check that all required base attributes have a matching non-prohibited derived attribute.
	derivedAttrs := make(map[string]*AttrUse, len(td.Attributes))
	for _, au := range td.Attributes {
		derivedAttrs[au.Name.Local] = au
	}
	for _, baseAU := range td.BaseType.Attributes {
		if !baseAU.Required {
			continue
		}
		derived, found := derivedAttrs[baseAU.Name.Local]
		if !found || derived.Prohibited {
			msg := fmt.Sprintf("A matching attribute use for the 'required' attribute use '%s' of the base complex type definition %s is missing.", baseAU.Name.Local, baseQualified)
			c.schemaErrors.WriteString(schemaComponentError(c.filename, src.line, "complexType", component, msg))
		}
	}

	// derivation-ok-restriction 4: Wildcard checks.
	if td.AnyAttribute != nil {
		// 4.1: Base must also have a wildcard.
		if td.BaseType.AnyAttribute == nil {
			msg := fmt.Sprintf("The complex type definition has an attribute wildcard, but the base complex type definition %s does not have one.", baseQualified)
			c.schemaErrors.WriteString(schemaComponentError(c.filename, src.line, "complexType", component, msg))
		} else {
			// 4.2: Derived namespace must be subset of base namespace.
			if !wildcardNSSubset(td.AnyAttribute, td.BaseType.AnyAttribute) {
				msg := fmt.Sprintf("The attribute wildcard is not a valid subset of the wildcard in the base complex type definition %s.", baseQualified)
				c.schemaErrors.WriteString(schemaComponentError(c.filename, src.line, "complexType", component, msg))
			}
			// 4.3: Derived processContents must be >= base strength (strict > lax > skip).
			// libxml2 attributes this error to the base type's source location.
			if processContentsStrength(td.AnyAttribute.ProcessContents) < processContentsStrength(td.BaseType.AnyAttribute.ProcessContents) {
				errLine := src.line
				errComponent := component
				if baseSrc, ok := c.typeDefSources[td.BaseType]; ok {
					errLine = baseSrc.line
					if !baseSrc.isLocal {
						errComponent = "complex type '" + td.BaseType.Name.Local + "'"
					}
				}
				msg := fmt.Sprintf("The {process contents} of the attribute wildcard is weaker than the one in the base complex type definition %s.", baseQualified)
				c.schemaErrors.WriteString(schemaComponentError(c.filename, errLine, "complexType", errComponent, msg))
			}
		}
	}
}

// wildcardNSSubset checks whether the namespace constraint of sub is a subset
// of the namespace constraint of super, per XSD §3.10.6.
func wildcardNSSubset(sub, super *Wildcard) bool {
	// ##any is a superset of everything.
	if super.Namespace == WildcardNSAny {
		return true
	}
	// If sub is ##any but super is not, sub is not a subset.
	if sub.Namespace == WildcardNSAny {
		return false
	}
	// Both are specific namespace sets — sub must be contained in super.
	subSet := wildcardNSSet(sub)
	superSet := wildcardNSSet(super)
	for ns := range subSet {
		if !superSet[ns] {
			return false
		}
	}
	return true
}

// wildcardNSSet expands a wildcard's namespace constraint into a set of URIs.
func wildcardNSSet(wc *Wildcard) map[string]bool {
	s := make(map[string]bool)
	switch wc.Namespace {
	case WildcardNSAny:
		// Matches everything — not representable as a finite set.
	case WildcardNSOther:
		// Everything except target namespace and absent (empty) — not finite.
		// For subset checking, treat as "not targetNS".
	case WildcardNSLocal:
		s[""] = true
	case WildcardNSTargetNamespace:
		s[wc.TargetNS] = true
	default:
		// Space-separated list of URIs, possibly including ##local and ##targetNamespace.
		for _, token := range strings.Fields(wc.Namespace) {
			switch token {
			case WildcardNSLocal:
				s[""] = true
			case WildcardNSTargetNamespace:
				s[wc.TargetNS] = true
			default:
				s[token] = true
			}
		}
	}
	return s
}

// wildcardUnion computes the union of two attribute wildcards.
// Per XSD 1.0 spec section 3.10.6: Attribute Wildcard Union.
//
// Namespace constraints are classified as:
//   - "any"       → matches everything
//   - "not(ns)"   → ##other: matches everything except ns and absent
//   - "not(absent)" → matches everything except absent (empty namespace)
//   - "set"       → finite set of namespace URIs (empty string = absent)
func wildcardUnion(w1, w2 *Wildcard) *Wildcard {
	pc := w1.ProcessContents
	tns := w1.TargetNS

	// Case 2: If either is ##any, result is ##any.
	if w1.Namespace == WildcardNSAny || w2.Namespace == WildcardNSAny {
		return &Wildcard{Namespace: WildcardNSAny, ProcessContents: pc, TargetNS: tns}
	}

	w1IsNeg := w1.Namespace == WildcardNSOther || w1.Namespace == WildcardNSNotAbsent
	w2IsNeg := w2.Namespace == WildcardNSOther || w2.Namespace == WildcardNSNotAbsent

	// Case 1: Both are the same value.
	if w1.Namespace == w2.Namespace && w1.TargetNS == w2.TargetNS {
		return &Wildcard{Namespace: w1.Namespace, ProcessContents: pc, TargetNS: tns}
	}

	// Case 3: Both are sets (neither is a negation or ##any).
	if !w1IsNeg && !w2IsNeg {
		set := wildcardNSSet(w1)
		for ns := range wildcardNSSet(w2) {
			set[ns] = true
		}
		return wildcardFromSet(set, pc, tns)
	}

	// Case 4: Both are negations.
	if w1IsNeg && w2IsNeg {
		w1NegNS := wildcardNegatedNS(w1)
		w2NegNS := wildcardNegatedNS(w2)
		if w1NegNS == w2NegNS {
			// Same negated value → same result.
			return &Wildcard{Namespace: w1.Namespace, ProcessContents: pc, TargetNS: tns}
		}
		// Different negated values → not(absent).
		return &Wildcard{Namespace: WildcardNSNotAbsent, ProcessContents: pc, TargetNS: tns}
	}

	// Cases 5 & 6: One is a negation, the other is a set.
	var neg, set *Wildcard
	if w1IsNeg {
		neg, set = w1, w2
	} else {
		neg, set = w2, w1
	}

	negNS := wildcardNegatedNS(neg)
	s := wildcardNSSet(set)
	hasAbsent := s[""]
	hasNegated := negNS != "" && s[negNS]

	if negNS == "" {
		// Case 6: neg is not(absent).
		if hasAbsent {
			// 6.1: Set includes absent → any.
			return &Wildcard{Namespace: WildcardNSAny, ProcessContents: pc, TargetNS: tns}
		}
		// 6.2: Set doesn't include absent → not(absent).
		return &Wildcard{Namespace: WildcardNSNotAbsent, ProcessContents: pc, TargetNS: tns}
	}

	// Case 5: neg is not(ns).
	if hasNegated && hasAbsent {
		// 5.1: Set includes both negated ns and absent → any.
		return &Wildcard{Namespace: WildcardNSAny, ProcessContents: pc, TargetNS: tns}
	}
	if hasNegated && !hasAbsent {
		// 5.2: Set includes negated ns but not absent → not(absent).
		return &Wildcard{Namespace: WildcardNSNotAbsent, ProcessContents: pc, TargetNS: tns}
	}
	if !hasNegated && !hasAbsent {
		// 5.4: Set includes neither → the negation.
		return &Wildcard{Namespace: neg.Namespace, ProcessContents: pc, TargetNS: neg.TargetNS}
	}
	// 5.3: Set includes absent but not negated ns → not expressible.
	// Fall back to ##any (permissive).
	return &Wildcard{Namespace: WildcardNSAny, ProcessContents: pc, TargetNS: tns}
}

// wildcardNegatedNS returns the namespace being negated.
// For ##other, it's the target namespace. For ##not-absent, it's "".
func wildcardNegatedNS(wc *Wildcard) string {
	if wc.Namespace == WildcardNSNotAbsent {
		return ""
	}
	// ##other negates the target namespace.
	return wc.TargetNS
}

// wildcardFromSet builds a Wildcard from a namespace set.
func wildcardFromSet(s map[string]bool, pc ProcessContentsKind, tns string) *Wildcard {
	var parts []string
	for ns := range s {
		if ns == "" {
			parts = append(parts, WildcardNSLocal)
		} else {
			parts = append(parts, ns)
		}
	}
	sort.Strings(parts)
	return &Wildcard{
		Namespace:       strings.Join(parts, " "),
		ProcessContents: pc,
		TargetNS:        tns,
	}
}

// processContentsStrength returns the strength of a processContents value.
// strict(2) > lax(1) > skip(0).
func processContentsStrength(pc ProcessContentsKind) int {
	switch pc {
	case ProcessStrict:
		return 2
	case ProcessLax:
		return 1
	default:
		return 0
	}
}

// checkCircularSubstGroup detects if an element's substitution group chain
// leads back to itself. Only reports an error if the element itself is part
// of the cycle (not if it just points to a cyclic element).
func (c *compiler) checkCircularSubstGroup(edecl *ElementDecl) {
	visited := map[QName]bool{}
	current := edecl.SubstitutionGroup
	for current != (QName{}) {
		if current == edecl.Name {
			// Cycle leads back to this element.
			// libxml2 reports this error twice.
			if src, ok := c.globalElemSources[edecl]; ok {
				msg := fmt.Sprintf("The element declaration '%s' defines a circular substitution group to element declaration '%s'.",
					edecl.Name.Local, current.Local)
				errStr := schemaElemDeclError(c.filename, src.line, edecl.Name.Local, msg)
				c.schemaErrors.WriteString(errStr)
				c.schemaErrors.WriteString(errStr)
			}
			return
		}
		if visited[current] {
			// Hit a cycle that doesn't include this element.
			return
		}
		visited[current] = true
		head, ok := c.schema.elements[current]
		if !ok {
			return
		}
		current = head.SubstitutionGroup
	}
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
	return doc.DocumentElement()
}

// collectNSContext collects namespace declarations from a schema element and its ancestors.
func collectNSContext(elem *helium.Element) map[string]string {
	nsMap := make(map[string]string)
	var node helium.Node = elem
	for node != nil {
		if e, ok := node.(*helium.Element); ok {
			for _, ns := range e.Namespaces() {
				prefix := ns.Prefix()
				if _, exists := nsMap[prefix]; !exists {
					nsMap[prefix] = ns.URI()
				}
			}
		}
		node = node.Parent()
	}
	return nsMap
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
	return idc
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
		td := &TypeDef{
			Name:        qn,
			ContentType: ct,
		}
		if name == "anyType" {
			td.ContentType = ContentTypeMixed
			td.AnyAttribute = &Wildcard{
				Namespace:       WildcardNSAny,
				ProcessContents: ProcessLax,
			}
		}
		s.types[qn] = td
	}
}
