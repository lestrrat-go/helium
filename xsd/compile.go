package xsd

import (
	"context"
	"fmt"
	"io/fs"
	"sort"
	"strconv"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/iofs"
	"github.com/lestrrat-go/helium/internal/lexicon"
)

// compiler holds state during schema compilation.
type compiler struct {
	schema  *Schema
	baseDir string // directory of the schema file, for resolving relative paths
	fsys    fs.FS  // filesystem for loading xs:include/xs:import/xs:redefine targets
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
	// typeKinds records, per registered named type QName, whether it was
	// declared via xs:simpleType or xs:complexType. Both share schema.types,
	// but redefine must distinguish them so a Phase-A simpleType cannot be
	// consumed by a complexType override of the same name (and vice versa).
	typeKinds map[QName]redefineKind
	// unresolved item type references for list types
	itemTypeRefs map[*TypeDef]QName
	// unresolved union member type references
	unionMemberRefs []unionMemberRef
	// chameleonEligible records, per ref owner (*ElementDecl or *TypeDef),
	// whether the lexical ref at the collection site was BOTH unprefixed and
	// had no in-scope default namespace. Only such refs may fall back to the
	// no-targetNamespace ({}) chameleon type; a prefixed ref or an unprefixed
	// ref bound by an xmlns="..." default namespace is qualified and must not.
	chameleonEligible map[any]struct{}
	// unresolved attribute references: maps from AttrUse to global attr QName
	attrRefs map[*AttrUse]QName
	// source info for attribute uses carrying a default/fixed value, used to
	// validate the constraint value against the attribute's simple type once
	// all type references are resolved (deferred to resolveRefs).
	attrUseConstraintSources map[*AttrUse]attrConstraintSource
	// source info for every attribute use, used by post-resolve declaration
	// checks (e.g. an un-enumerated xs:NOTATION typed attribute).
	attrUseSources map[*AttrUse]attrConstraintSource
	// error handler for reporting schema errors/warnings
	errorHandler helium.ErrorHandler
	errorCount   int    // count of fatal errors reported
	filename     string // XSD filename for error messages
	includeFile  string // currently-included file path (for duplicate element errors)
	// importedNS tracks which namespaces have been imported and their schema locations.
	importedNS map[string]string // namespace → schema location
	// importDepth and maxImportDepth bound xs:import recursion. Each sub-compiler
	// created by loadImport inherits maxImportDepth and stores its own
	// importDepth = parent.importDepth + 1. A sub-compiler whose depth would
	// exceed the limit is rejected. This blocks namespace-cycling import
	// chains (A → B → C → A …) where each schema declares a distinct namespace
	// so the per-compiler importedNS map does not detect the cycle.
	importDepth    int
	maxImportDepth int
	// redefine is non-nil while processing the override children of an
	// xs:redefine. It scopes the duplicate-name suppression to the specific
	// (kind, name) components actually loaded by the redefined schema, each
	// consumable exactly once, instead of suppressing the check globally.
	redefine *redefineState
}

// redefineKind identifies the component category a redefine override targets.
type redefineKind int

const (
	// redefineKindSimpleType and redefineKindComplexType are tracked
	// separately so a Phase-A simpleType cannot be consumed by a complexType
	// override of the same name (or vice versa). They share the same
	// schema.types map but are distinct redefine targets.
	redefineKindSimpleType redefineKind = iota
	redefineKindComplexType
	redefineKindGroup
	redefineKindAttrGroup
)

// redefineState scopes duplicate-name suppression during an xs:redefine
// override loop. phaseAKeys records, per kind, the QNames loaded from the
// redefined schema (Phase A); each may be replaced by exactly one override.
// seen records, per kind, the override QNames already consumed so a repeated
// override of the same name is reported as a duplicate.
type redefineState struct {
	phaseAKeys map[redefineKind]map[QName]struct{}
	seen       map[redefineKind]map[QName]struct{}
}

// allowsRedefine reports whether an override of the given (kind, name) may
// replace an existing same-named component. It returns true only the first
// time a name loaded in Phase A is overridden; a name not loaded in Phase A
// or a repeated override returns false so the caller reports a duplicate.
func (c *compiler) allowsRedefine(kind redefineKind, qn QName) bool {
	if c.redefine == nil {
		return false
	}
	if _, ok := c.redefine.phaseAKeys[kind][qn]; !ok {
		return false
	}
	if _, seen := c.redefine.seen[kind][qn]; seen {
		return false
	}
	if c.redefine.seen[kind] == nil {
		c.redefine.seen[kind] = make(map[QName]struct{})
	}
	c.redefine.seen[kind][qn] = struct{}{}
	return true
}

// consumeRedefineTarget validates a redefine override's (kind, name) against
// the Phase-A key set and consumes it, BEFORE the override child is parsed and
// regardless of whether the name currently exists in the schema map. It reports
// a duplicate (and returns false) when the target was not loaded in Phase A or
// has already been consumed by an earlier override; otherwise it marks the name
// consumed and returns true so the caller may parse the override. This closes
// the gap where allowsRedefine only ran under the existing-name branch, letting
// an override of a name ABSENT from the redefined schema be accepted silently.
func (c *compiler) consumeRedefineTarget(ctx context.Context, elem *helium.Element, kind redefineKind, qn QName, component, kindDesc string) bool {
	if c.allowsRedefine(kind, qn) {
		return true
	}
	c.reportDuplicateComponent(ctx, elem, component, kindDesc, qn)
	return false
}

// redefineConsumed reports whether the override of (kind, name) was already
// validated and consumed by the redefine override loop (consumeRedefineTarget).
// The named-component parsers consult it so a pre-authorized override does not
// re-trigger their own duplicate-name report, while a non-redefine duplicate
// (c.redefine == nil, or a name the loop did not consume) still reports.
func (c *compiler) redefineConsumed(kind redefineKind, qn QName) bool {
	if c.redefine == nil {
		return false
	}
	_, ok := c.redefine.seen[kind][qn]
	return ok
}

// defaultMaxImportDepth bounds xs:import recursion depth (not a flat
// import count), so a modest value is enough to terminate adversarial
// namespace-cycling chains while leaving generous headroom for real
// schema hierarchies, which rarely nest more than a few levels deep.
// The limit is not currently exposed as a Compiler option; matches the
// hardcoded defensive caps used by relaxng (include limit), catalog
// (resolution depth), and xpath/xslt (recursion depth).
const defaultMaxImportDepth = 40

// elemRefSource tracks source location for error reporting.
type elemRefSource struct {
	elemName string
	line     int
}

// unionMemberRef tracks an unresolved union member type reference.
type unionMemberRef struct {
	owner *TypeDef
	name  QName
	// chameleonEligible is true when the lexical memberTypes QName was
	// unprefixed with no in-scope default namespace, so it may fall back to
	// the no-targetNamespace ({}) chameleon type if unresolved.
	chameleonEligible bool
}

// attrConstraintSource tracks where an attribute use's default/fixed value
// came from, so its value can be validated against the attribute's declared
// simple type after type references are resolved.
type attrConstraintSource struct {
	line  int
	local string            // attribute display name (local name)
	nsMap map[string]string // in-scope namespaces for value validation (QName/NOTATION)
}

// typeDefSource tracks source location and context for type definitions.
type typeDefSource struct {
	line    int
	isLocal bool // true for anonymous (local) complex types
}

func compileSchema(ctx context.Context, doc *helium.Document, baseDir string, cfg *compileConfig) (*Schema, error) {
	root := findDocumentElement(doc)
	if root == nil {
		return nil, fmt.Errorf("xsd: empty document")
	}

	if !isXSDElement(root, elemSchema) {
		return nil, fmt.Errorf("xsd: root element is not xs:schema")
	}

	fsys := fs.FS(iofs.PermissiveRoot{})
	if cfg != nil && cfg.fsys != nil {
		fsys = cfg.fsys
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
		baseDir:                  baseDir,
		fsys:                     fsys,
		typeRefs:                 make(map[*TypeDef]QName),
		elemRefs:                 make(map[*ElementDecl]QName),
		elemRefSources:           make(map[*ElementDecl]elemRefSource),
		groupRefs:                make(map[*ModelGroup]QName),
		attrGroupRefs:            make(map[*TypeDef][]QName),
		globalElemSources:        make(map[*ElementDecl]elemRefSource),
		typeDefSources:           make(map[*TypeDef]typeDefSource),
		typeKinds:                make(map[QName]redefineKind),
		itemTypeRefs:             make(map[*TypeDef]QName),
		chameleonEligible:        make(map[any]struct{}),
		attrRefs:                 make(map[*AttrUse]QName),
		attrUseConstraintSources: make(map[*AttrUse]attrConstraintSource),
		attrUseSources:           make(map[*AttrUse]attrConstraintSource),
		importedNS:               make(map[string]string),
		maxImportDepth:           defaultMaxImportDepth,
	}
	c.errorHandler = helium.NilErrorHandler{}
	if cfg != nil {
		c.filename = cfg.label
		if c.filename == "" {
			c.filename = doc.URL()
		}
		if c.filename == "" {
			c.filename = "(string)"
		}
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
			c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserErrorAttr(c.filename, root.Line(), root.LocalName(), elemSchema, attrBlockDefault,
				"The value '"+v+"' is not valid. Expected is '(#all | List of (extension | restriction | substitution))'."), helium.ErrorLevelFatal))
			c.errorCount++
		} else {
			c.schema.blockDefault = parseBlockFlags(v)
		}
	}

	// Parse finalDefault attribute.
	if v := getAttr(root, attrFinalDefault); v != "" {
		if !isValidFinalDefault(v) && c.filename != "" {
			c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserErrorAttr(c.filename, root.Line(), root.LocalName(), elemSchema, attrFinalDefault,
				"The value '"+v+"' is not valid. Expected is '(#all | List of (extension | restriction | list | union))'."), helium.ErrorLevelFatal))
			c.errorCount++
		} else {
			c.schema.finalDefault = parseFinalFlags(v)
		}
	}

	// Register built-in types.
	registerBuiltinTypes(c.schema)

	// First pass: collect all named types and global elements.
	if err := c.parseSchemaChildren(ctx, root); err != nil {
		return nil, err
	}

	// Process includes after parsing the main schema's declarations.
	// This matches libxml2's processing order where includes are merged
	// after the including schema's own declarations are registered.
	if err := c.processIncludes(ctx, root); err != nil {
		return nil, err
	}

	// Second pass: resolve type references.
	c.resolveRefs(ctx)

	// Check facet consistency after refs are resolved (base types are available).
	c.checkFacetConsistency(ctx)

	// Validate QName/NOTATION enumeration literal prefixes (and the NOTATION
	// no-enumeration rule) now that base types are resolved.
	c.checkEnumQNameAndNotation(ctx)

	// Reject element/attribute declarations whose effective type is the built-in
	// xs:NOTATION (or NOTATION-derived) without an effective enumeration facet.
	c.checkNotationOnDeclarations(ctx)

	// Build substitution group membership map and detect circular references.
	for _, edecl := range c.schema.elements {
		if edecl.SubstitutionGroup == (QName{}) {
			continue
		}
		head := edecl.SubstitutionGroup
		c.schema.substGroups[head] = append(c.schema.substGroups[head], edecl)

		// Check for circular substitution groups.
		if c.filename != "" {
			c.checkCircularSubstGroup(ctx, edecl)
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
		c.checkFinalOnTypes(ctx)
		c.checkFinalOnSubstGroups(ctx)
	}

	return c.schema, nil
}

// parseSchemaChildren parses the children of an xs:schema element.
func (c *compiler) parseSchemaChildren(ctx context.Context, root *helium.Element) error {
	for child := range helium.Children(root) {
		if child.Type() != helium.ElementNode {
			continue
		}
		elem, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		switch {
		case isXSDElement(elem, elemElement):
			if err := c.parseGlobalElement(ctx, elem); err != nil {
				return err
			}
		case isXSDElement(elem, elemComplexType):
			if err := c.parseNamedComplexType(ctx, elem); err != nil {
				return err
			}
		case isXSDElement(elem, elemSimpleType):
			if err := c.parseNamedSimpleType(ctx, elem); err != nil {
				return err
			}
		case isXSDElement(elem, elemGroup):
			if err := c.parseNamedGroup(ctx, elem); err != nil {
				return err
			}
		case isXSDElement(elem, elemAttributeGroup):
			if err := c.parseNamedAttributeGroup(ctx, elem); err != nil {
				return err
			}
		case isXSDElement(elem, elemAttribute):
			c.parseGlobalAttribute(ctx, elem)
		}
	}
	return nil
}

func findDocumentElement(doc *helium.Document) *helium.Element {
	return doc.DocumentElement()
}

// collectNSContext collects namespace declarations from a schema element and its ancestors.
// inScopeNamespace returns the nearest in-scope namespace declaration on elem
// or its ancestors whose URI matches href, or nil if none is declared. The
// returned declaration's prefix is reused when inserting a qualified default
// attribute so the inserted node mirrors the document's own prefix binding.
func inScopeNamespace(elem *helium.Element, href string) *helium.Namespace {
	var node helium.Node = elem
	for node != nil {
		if e, ok := node.(*helium.Element); ok {
			for _, ns := range e.Namespaces() {
				if ns.URI() == href {
					return ns
				}
			}
		}
		node = node.Parent()
	}
	return nil
}

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

func isXSDElement(elem *helium.Element, localName string) bool {
	return elem.LocalName() == localName && elem.URI() == lexicon.NamespaceXSD
}

// getAttr returns the value of an unqualified (no-namespace) schema attribute.
// XSD schema attributes (name/type/fixed/default/minOccurs/...) are always
// unqualified, so a foreign-namespaced attribute sharing the local name (e.g.
// other:fixed) must not be mistaken for the XSD attribute.
func getAttr(elem *helium.Element, name string) string {
	attr, ok := elem.FindAttribute(helium.NSPredicate{Local: name, NamespaceURI: ""})
	if !ok {
		return ""
	}
	return attr.Value()
}

// parseBlockFlags parses a block attribute value into BlockFlags.
func parseBlockFlags(v string) BlockFlags {
	if v == lexicon.ModeAll {
		return BlockExtension | BlockRestriction | BlockSubstitution
	}
	var f BlockFlags
	for _, part := range splitSpace(v) {
		switch part {
		case attrValExtension:
			f |= BlockExtension
		case attrValRestriction:
			f |= BlockRestriction
		case attrValSubstitution:
			f |= BlockSubstitution
		}
	}
	return f
}

// parseFinalFlags parses a finalDefault or simpleType final attribute value into FinalFlags.
func parseFinalFlags(v string) FinalFlags {
	if v == lexicon.ModeAll {
		return FinalExtension | FinalRestriction | FinalList | FinalUnion
	}
	var f FinalFlags
	for _, part := range splitSpace(v) {
		switch part {
		case attrValExtension:
			f |= FinalExtension
		case attrValRestriction:
			f |= FinalRestriction
		case attrValList:
			f |= FinalList
		case attrValUnion:
			f |= FinalUnion
		}
	}
	return f
}

// parseElemFinalFlags parses a final attribute value for elements/complexTypes
// (only extension/restriction are valid).
func parseElemFinalFlags(v string) FinalFlags {
	if v == lexicon.ModeAll {
		return FinalExtension | FinalRestriction
	}
	var f FinalFlags
	for _, part := range splitSpace(v) {
		switch part {
		case attrValExtension:
			f |= FinalExtension
		case attrValRestriction:
			f |= FinalRestriction
		}
	}
	return f
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
		"anyURI", lexicon.TypeQName, lexicon.TypeNotation,
		"anyType", "anySimpleType",
	}
	for _, name := range builtins {
		qn := QName{Local: name, NS: lexicon.NamespaceXSD}
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
