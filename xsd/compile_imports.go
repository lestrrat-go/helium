package xsd

import (
	"fmt"
	"os"
	"path/filepath"

	helium "github.com/lestrrat-go/helium"
)

// processIncludesAndImports handles xs:include and xs:import elements.
func (c *compiler) processIncludes(root *helium.Element) error {
	for child := range helium.Children(root) {
		if child.Type() != helium.ElementNode {
			continue
		}
		elem := child.(*helium.Element)
		switch {
		case isXSDElement(elem, elemInclude):
			loc := getAttr(elem, attrSchemaLocation)
			if loc == "" {
				continue
			}
			if err := c.loadInclude(loc, elem); err != nil {
				return err
			}
		case isXSDElement(elem, elemImport):
			loc := getAttr(elem, attrSchemaLocation)
			if loc == "" {
				continue
			}
			ns := getAttr(elem, attrNamespace)

			// Check if this namespace was already imported.
			if prevLoc, ok := c.importedNS[ns]; ok && c.filename != "" {
				displayLoc := filepath.Join(filepath.Dir(c.filename), loc)
				displayPrevLoc := filepath.Join(filepath.Dir(c.filename), prevLoc)
				c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaParserWarning(c.filename, elem.Line(),
					elem.LocalName(), elemImport,
					"Skipping import of schema located at '"+displayLoc+"' for the namespace '"+ns+"', since this namespace was already imported with the schema located at '"+displayPrevLoc+"'."), helium.ErrorLevelWarning))
				continue
			}

			if err := c.loadImport(loc, ns); err != nil {
				// Import failure — report warning if we have a filename.
				if c.filename != "" {
					displayLoc := filepath.Join(filepath.Dir(c.filename), loc)
					c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(fmt.Sprintf("I/O warning : failed to load \"%s\": %s\n", displayLoc, "No such file or directory"), helium.ErrorLevelWarning))
					c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaParserWarning(c.filename, elem.Line(),
						elem.LocalName(), elemImport,
						"Failed to locate a schema at location '"+displayLoc+"'. Skipping the import."), helium.ErrorLevelWarning))
				}
				continue
			}

			// Track the imported namespace.
			c.importedNS[ns] = loc
		case isXSDElement(elem, elemRedefine):
			loc := getAttr(elem, attrSchemaLocation)
			if loc == "" {
				continue
			}
			if err := c.loadRedefine(loc, elem); err != nil {
				return err
			}
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

	data, err := os.ReadFile(path) //nolint:gosec // path is constructed from schema baseDir + user-supplied location
	if err != nil {
		return fmt.Errorf("xsd: failed to load include %q: %w", location, err)
	}

	doc, err := helium.Parse(c.compileContext(), data)
	if err != nil {
		return fmt.Errorf("xsd: failed to parse include %q: %w", location, err)
	}

	incRoot := findDocumentElement(doc)
	if incRoot == nil || !isXSDElement(incRoot, elemSchema) {
		return fmt.Errorf("xsd: included document %q is not an xs:schema", location)
	}

	// Check target namespace compatibility.
	incTargetNS := getAttr(incRoot, attrTargetNamespace)
	if incTargetNS != "" && incTargetNS != c.schema.targetNamespace {
		displayLoc := location
		if c.filename != "" {
			displayLoc = filepath.Join(filepath.Dir(c.filename), location)
		}
		c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaParserError(c.filename, includeElem.Line(),
			includeElem.LocalName(), elemInclude,
			"The target namespace '"+incTargetNS+"' of the included/redefined schema '"+displayLoc+"' differs from '"+c.schema.targetNamespace+"' of the including/redefining schema."), helium.ErrorLevelFatal))
		c.errorCount++
		return nil
	}

	// Chameleon include: if the included schema has no targetNamespace,
	// it adopts the including schema's targetNamespace.
	// The included schema's elementFormDefault/attributeFormDefault are
	// applied within the included declarations.

	// Save current form-qualified and default settings, then apply included schema's settings.
	savedElemForm := c.schema.elemFormQualified
	savedAttrForm := c.schema.attrFormQualified
	savedBlockDefault := c.schema.blockDefault
	savedFinalDefault := c.schema.finalDefault
	savedIncludeFile := c.includeFile
	if v := getAttr(incRoot, attrElementFormDefault); v != "" {
		c.schema.elemFormQualified = v == attrValQualified
	}
	if v := getAttr(incRoot, attrAttributeFormDefault); v != "" {
		c.schema.attrFormQualified = v == attrValQualified
	}
	if v := getAttr(incRoot, attrBlockDefault); v != "" {
		c.schema.blockDefault = parseBlockFlags(v)
	}
	if v := getAttr(incRoot, attrFinalDefault); v != "" {
		c.schema.finalDefault = parseFinalFlags(v)
	}

	// Set the include file path for duplicate element error reporting.
	if c.filename != "" {
		c.includeFile = filepath.Join(filepath.Dir(c.filename), location)
	}

	// Parse the included schema's declarations into the current compiler.
	err = c.parseSchemaChildren(incRoot)

	// Restore form-qualified settings, defaults, and include file.
	c.schema.elemFormQualified = savedElemForm
	c.schema.attrFormQualified = savedAttrForm
	c.schema.blockDefault = savedBlockDefault
	c.schema.finalDefault = savedFinalDefault
	c.includeFile = savedIncludeFile

	return err
}

// loadRedefine loads a schema via xs:redefine and processes override children.
// It works like xs:include (merging original declarations) but then applies
// redefinitions for complexType, simpleType, group, and attributeGroup children.
func (c *compiler) loadRedefine(location string, redefineElem *helium.Element) error {
	// Phase A: Load the redefined schema (same as include).
	path := location
	if c.baseDir != "" {
		path = filepath.Join(c.baseDir, location)
	}

	data, err := os.ReadFile(path) //nolint:gosec // path is constructed from schema baseDir + user-supplied location
	if err != nil {
		return fmt.Errorf("xsd: failed to load redefine %q: %w", location, err)
	}

	doc, err := helium.Parse(c.compileContext(), data)
	if err != nil {
		return fmt.Errorf("xsd: failed to parse redefine %q: %w", location, err)
	}

	incRoot := findDocumentElement(doc)
	if incRoot == nil || !isXSDElement(incRoot, elemSchema) {
		return fmt.Errorf("xsd: redefined document %q is not an xs:schema", location)
	}

	// Check target namespace compatibility (same rules as include).
	incTargetNS := getAttr(incRoot, attrTargetNamespace)
	if incTargetNS != "" && incTargetNS != c.schema.targetNamespace {
		displayLoc := location
		if c.filename != "" {
			displayLoc = filepath.Join(filepath.Dir(c.filename), location)
		}
		c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaParserError(c.filename, redefineElem.Line(),
			redefineElem.LocalName(), elemRedefine,
			"The target namespace '"+incTargetNS+"' of the included/redefined schema '"+displayLoc+"' differs from '"+c.schema.targetNamespace+"' of the including/redefining schema."), helium.ErrorLevelFatal))
		c.errorCount++
		return nil
	}

	// Save/restore form-qualified settings and defaults (chameleon support).
	savedElemForm := c.schema.elemFormQualified
	savedAttrForm := c.schema.attrFormQualified
	savedBlockDefault := c.schema.blockDefault
	savedFinalDefault := c.schema.finalDefault
	savedIncludeFile := c.includeFile
	if v := getAttr(incRoot, attrElementFormDefault); v != "" {
		c.schema.elemFormQualified = v == attrValQualified
	}
	if v := getAttr(incRoot, attrAttributeFormDefault); v != "" {
		c.schema.attrFormQualified = v == attrValQualified
	}
	if v := getAttr(incRoot, attrBlockDefault); v != "" {
		c.schema.blockDefault = parseBlockFlags(v)
	}
	if v := getAttr(incRoot, attrFinalDefault); v != "" {
		c.schema.finalDefault = parseFinalFlags(v)
	}
	if c.filename != "" {
		c.includeFile = filepath.Join(filepath.Dir(c.filename), location)
	}

	// Parse the included schema's declarations into the current compiler.
	if err := c.parseSchemaChildren(incRoot); err != nil {
		c.schema.elemFormQualified = savedElemForm
		c.schema.attrFormQualified = savedAttrForm
		c.schema.blockDefault = savedBlockDefault
		c.schema.finalDefault = savedFinalDefault
		c.includeFile = savedIncludeFile
		return err
	}

	// Phase B: Process redefine children (overrides).
	for child := range helium.Children(redefineElem) {
		if child.Type() != helium.ElementNode {
			continue
		}
		elem := child.(*helium.Element)
		switch {
		case isXSDElement(elem, elemAnnotation):
			// skip
		case isXSDElement(elem, elemComplexType):
			name := getAttr(elem, attrName)
			if name == "" {
				continue
			}
			qn := QName{Local: name, NS: c.schema.targetNamespace}
			origType := c.schema.types[qn]
			if err := c.parseNamedComplexType(elem); err != nil {
				c.schema.elemFormQualified = savedElemForm
				c.schema.attrFormQualified = savedAttrForm
				c.schema.blockDefault = savedBlockDefault
				c.schema.finalDefault = savedFinalDefault
				c.includeFile = savedIncludeFile
				return err
			}
			// Patch self-reference: redirect the typeRef to a temporary key
			// holding the original type, so resolveRefs handles extension
			// merge (content model + attribute inheritance) naturally.
			newType := c.schema.types[qn]
			if origType != nil {
				if refQN, ok := c.typeRefs[newType]; ok && refQN == qn {
					origKey := QName{Local: "\x00redefine:" + name, NS: qn.NS}
					c.schema.types[origKey] = origType
					c.typeRefs[newType] = origKey
				}
			}
		case isXSDElement(elem, elemSimpleType):
			name := getAttr(elem, attrName)
			if name == "" {
				continue
			}
			qn := QName{Local: name, NS: c.schema.targetNamespace}
			origType := c.schema.types[qn]
			if err := c.parseNamedSimpleType(elem); err != nil {
				c.schema.elemFormQualified = savedElemForm
				c.schema.attrFormQualified = savedAttrForm
				c.schema.blockDefault = savedBlockDefault
				c.schema.finalDefault = savedFinalDefault
				c.includeFile = savedIncludeFile
				return err
			}
			newType := c.schema.types[qn]
			if origType != nil {
				if refQN, ok := c.typeRefs[newType]; ok && refQN == qn {
					origKey := QName{Local: "\x00redefine:" + name, NS: qn.NS}
					c.schema.types[origKey] = origType
					c.typeRefs[newType] = origKey
				}
			}
		case isXSDElement(elem, elemGroup):
			name := getAttr(elem, attrName)
			if name == "" {
				continue
			}
			qn := QName{Local: name, NS: c.schema.targetNamespace}
			origGroup := c.schema.groups[qn]
			// Snapshot existing groupRefs keys.
			existingRefs := make(map[*ModelGroup]bool, len(c.groupRefs))
			for mg := range c.groupRefs {
				existingRefs[mg] = true
			}
			if err := c.parseNamedGroup(elem); err != nil {
				c.schema.elemFormQualified = savedElemForm
				c.schema.attrFormQualified = savedAttrForm
				c.schema.blockDefault = savedBlockDefault
				c.schema.finalDefault = savedFinalDefault
				c.includeFile = savedIncludeFile
				return err
			}
			// Patch self-reference: find newly-added groupRefs entries referencing qn.
			if origGroup != nil {
				for mg, refQN := range c.groupRefs {
					if existingRefs[mg] {
						continue
					}
					if refQN == qn {
						mg.Compositor = origGroup.Compositor
						mg.Particles = origGroup.Particles
						delete(c.groupRefs, mg)
					}
				}
			}
		case isXSDElement(elem, elemAttributeGroup):
			name := getAttr(elem, attrName)
			if name == "" {
				continue
			}
			qn := QName{Local: name, NS: c.schema.targetNamespace}
			origAttrs := c.schema.attrGroups[qn]
			// Build the new attribute list manually, expanding self-references
			// inline. parseNamedAttributeGroup only collects xs:attribute children
			// and doesn't handle xs:attributeGroup ref children within a definition.
			var attrs []*AttrUse
			for gc := range helium.Children(elem) {
				if gc.Type() != helium.ElementNode {
					continue
				}
				gce := gc.(*helium.Element)
				switch {
				case isXSDElement(gce, elemAttribute):
					au := c.parseAttributeUse(gce)
					attrs = append(attrs, au)
				case isXSDElement(gce, elemAttributeGroup):
					if ref := getAttr(gce, attrRef); ref != "" {
						refQN := c.resolveQName(gce, ref)
						if refQN == qn && origAttrs != nil {
							attrs = append(attrs, origAttrs...)
						}
					}
				}
			}
			c.schema.attrGroups[qn] = attrs
		}
	}

	// Restore form-qualified settings, defaults, and include file.
	c.schema.elemFormQualified = savedElemForm
	c.schema.attrFormQualified = savedAttrForm
	c.schema.blockDefault = savedBlockDefault
	c.schema.finalDefault = savedFinalDefault
	c.includeFile = savedIncludeFile

	return nil
}

// loadImport loads an imported schema and merges its declarations.
func (c *compiler) loadImport(location, _ string) error {
	path := location
	if c.baseDir != "" {
		path = filepath.Join(c.baseDir, location)
	}

	data, err := os.ReadFile(path) //nolint:gosec // path is constructed from schema baseDir + import location
	if err != nil {
		return fmt.Errorf("xsd: failed to load import %q: %w", location, err)
	}

	doc, err := helium.Parse(c.compileContext(), data)
	if err != nil {
		return fmt.Errorf("xsd: failed to parse import %q: %w", location, err)
	}

	impRoot := findDocumentElement(doc)
	if impRoot == nil || !isXSDElement(impRoot, elemSchema) {
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

	// Sub-compiler collects errors into its own collector so we can
	// conditionally forward them. This matches libxml2's behavior of
	// stopping error reporting after the first import failure.
	subCollector := helium.NewErrorCollector(c.compileContext(), helium.ErrorLevelNone)
	impC.errorHandler = subCollector

	impC.schema.targetNamespace = getAttr(impRoot, attrTargetNamespace)
	impC.schema.elemFormQualified = getAttr(impRoot, attrElementFormDefault) == attrValQualified
	impC.schema.attrFormQualified = getAttr(impRoot, attrAttributeFormDefault) == attrValQualified
	if v := getAttr(impRoot, attrBlockDefault); v != "" {
		impC.schema.blockDefault = parseBlockFlags(v)
	}
	if v := getAttr(impRoot, attrFinalDefault); v != "" {
		impC.schema.finalDefault = parseFinalFlags(v)
	}

	registerBuiltinTypes(impC.schema)

	if err := impC.parseSchemaChildren(impRoot); err != nil {
		return err
	}

	// Process includes/imports in the imported schema (but skip back-references).
	if err := impC.processIncludes(impRoot); err != nil {
		// Non-fatal for imported schemas.
		_ = err
	}

	_ = subCollector.Close()

	// Only propagate sub-compiler errors to the parent if the parent has no
	// prior errors. This matches libxml2's behavior of stopping error reporting
	// after the first import failure.
	if impC.errorCount > 0 {
		if c.errorCount == 0 {
			for _, e := range subCollector.Errors() {
				c.errorHandler.Handle(c.compileContext(), e)
			}
		}
		c.errorCount += impC.errorCount
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
