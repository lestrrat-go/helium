package xsd

import (
	"context"
	"fmt"
	"sort"

	helium "github.com/lestrrat-go/helium"
)

// attrValTrue and attrValQualified are common XML attribute value strings.
const (
	attrValTrue      = "true"
	attrValQualified = "qualified"
)

// compiler holds state during schema compilation.
type compiler struct {
	ctx     context.Context
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
	// error handler for reporting schema errors/warnings
	errorHandler helium.ErrorHandler
	errorCount   int    // count of fatal errors reported
	filename     string // XSD filename for error messages
	includeFile  string // currently-included file path (for duplicate element errors)
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

func (c *compiler) compileContext() context.Context {
	if c == nil || c.ctx == nil {
		return context.Background()
	}
	return c.ctx
}

func compileSchema(ctx context.Context, doc *helium.Document, baseDir string, cfg *compileConfig) (*Schema, error) {
	root := findDocumentElement(doc)
	if root == nil {
		return nil, fmt.Errorf("xsd: empty document")
	}

	if !isXSDElement(root, elemSchema) {
		return nil, fmt.Errorf("xsd: root element is not xs:schema")
	}

	c := &compiler{
		ctx: ctx,
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
	c.errorHandler = helium.NilErrorHandler{}
	if cfg != nil {
		c.filename = cfg.filename
		if cfg.errorHandler != nil {
			c.errorHandler = cfg.errorHandler
		}
	}

	c.schema.targetNamespace = getAttr(root, attrTargetNamespace)
	c.schema.elemFormQualified = getAttr(root, attrElementFormDefault) == attrValQualified
	c.schema.attrFormQualified = getAttr(root, attrAttributeFormDefault) == attrValQualified

	// Parse blockDefault attribute.
	if v := getAttr(root, attrBlockDefault); v != "" {
		if !isValidBlock(v) && c.filename != "" {
			c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaParserErrorAttr(c.filename, root.Line(),
				root.LocalName(), elemSchema, attrBlockDefault,
				"The value '"+v+"' is not valid. Expected is '(#all | List of (extension | restriction | substitution))'."), helium.ErrorLevelFatal))
			c.errorCount++
		} else {
			c.schema.blockDefault = parseBlockFlags(v)
		}
	}

	// Parse finalDefault attribute.
	if v := getAttr(root, attrFinalDefault); v != "" {
		if !isValidFinalDefault(v) && c.filename != "" {
			c.errorHandler.Handle(c.compileContext(), helium.NewLeveledError(schemaParserErrorAttr(c.filename, root.Line(),
				root.LocalName(), elemSchema, attrFinalDefault,
				"The value '"+v+"' is not valid. Expected is '(#all | List of (extension | restriction | list | union))'."), helium.ErrorLevelFatal))
			c.errorCount++
		} else {
			c.schema.finalDefault = parseFinalFlags(v)
		}
	}

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

	// Check facet consistency after refs are resolved (base types are available).
	c.checkFacetConsistency()

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

	// Enforce final on type derivations.
	if c.filename != "" && c.errorCount == 0 {
		c.checkFinalOnTypes()
		c.checkFinalOnSubstGroups()
	}

	return c.schema, nil
}

// parseSchemaChildren parses the children of an xs:schema element.
func (c *compiler) parseSchemaChildren(root *helium.Element) error {
	for child := range helium.Children(root) {
		if child.Type() != helium.ElementNode {
			continue
		}
		elem := child.(*helium.Element)
		switch {
		case isXSDElement(elem, elemElement):
			if err := c.parseGlobalElement(elem); err != nil {
				return err
			}
		case isXSDElement(elem, elemComplexType):
			if err := c.parseNamedComplexType(elem); err != nil {
				return err
			}
		case isXSDElement(elem, elemSimpleType):
			if err := c.parseNamedSimpleType(elem); err != nil {
				return err
			}
		case isXSDElement(elem, elemGroup):
			if err := c.parseNamedGroup(elem); err != nil {
				return err
			}
		case isXSDElement(elem, elemAttributeGroup):
			if err := c.parseNamedAttributeGroup(elem); err != nil {
				return err
			}
		case isXSDElement(elem, elemAttribute):
			c.parseGlobalAttribute(elem)
		}
	}
	return nil
}
