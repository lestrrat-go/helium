package xsd

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"path"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/iolimit"
	"github.com/lestrrat-go/helium/internal/uripath"
)

// errImportDepthExceeded signals that xs:import recursion reached the
// configured limit. processIncludes propagates this error rather than
// treating it as a warning the way it treats ordinary I/O failures.
var errImportDepthExceeded = errors.New("xsd: max import depth exceeded")

// errIncludeDepthExceeded signals that xs:include/xs:redefine nesting reached
// the configured limit. It is a secondary guard behind the includeVisited
// loaded-set; processNestedIncludes returns it as a fatal compilation error.
var errIncludeDepthExceeded = errors.New("xsd: max include depth exceeded")

// errSchemaPathEscape signals that a schemaLocation joined onto baseDir
// would escape upward via ".." segments. processIncludes surfaces this
// as a fatal error rather than swallowing it as a generic I/O warning,
// so the containment violation is visible to callers.
var errSchemaPathEscape = errors.New("xsd: schema location escapes base directory")

// errSchemaTooLarge signals that a nested schema (xs:include/xs:import/
// xs:redefine target) exceeded [maxNestedSchemaSize] while being read. It is a
// resource-limit guard, so it is classified fatal by [IsFatalSchemaLoad] and
// must not be silently demoted to an I/O warning on the xs:import path: a
// hostile schemaLocation (e.g. /dev/zero) must abort compilation rather than
// be swallowed.
var errSchemaTooLarge = errors.New("xsd: schema resource exceeds size limit")

// maxNestedSchemaSize bounds the number of bytes read from any single nested
// schema document loaded via xs:include/xs:import/xs:redefine, so an endless or
// oversized source cannot exhaust memory. It mirrors xinclude's per-resource
// cap (10 MiB).
const maxNestedSchemaSize = 10 << 20 // 10 MiB

// readNestedSchema reads path through the configured fs.FS under a strict
// [maxNestedSchemaSize] byte cap, so an endless or oversized source
// (xs:include/xs:import/xs:redefine target) cannot exhaust memory; it replaces
// the unbounded fs.ReadFile every nested-schema loader used to call. It prefers
// the streaming [fs.File] from Open so an endless device (e.g. /dev/zero) is
// bounded while reading (iolimit reads one extra byte so a source that
// under-reports its size is still caught). When the FS does not support Open
// (e.g. a ReadFileFS-only in-memory FS whose Open returns an error), it falls
// back to fs.ReadFile and enforces the same cap on the fully-read result. The
// cap breach is reported as [errSchemaTooLarge], classified fatal by
// [IsFatalSchemaLoad].
func (c *compiler) readNestedSchema(path string) ([]byte, error) {
	if f, err := c.fsys.Open(path); err == nil {
		data, exceeded, readErr := iolimit.ReadAll(f, maxNestedSchemaSize)
		_ = f.Close()
		if exceeded {
			return nil, errSchemaTooLarge
		}
		if readErr != nil {
			return nil, readErr //nolint:wrapcheck // callers wrap with the schemaLocation for context
		}
		return data, nil
	}
	data, err := fs.ReadFile(c.fsys, path)
	if err != nil {
		return nil, err //nolint:wrapcheck // callers wrap with the schemaLocation for context
	}
	if int64(len(data)) > maxNestedSchemaSize {
		return nil, errSchemaTooLarge
	}
	return data, nil
}

// FatalSchemaLoader is implemented by errors raised from a configured [fs.FS]
// (see [Compiler.FS]) that must abort compilation rather than be demoted to a
// warning when they occur while loading an xs:import target. The xsd compiler
// normally treats a failure to load an xs:import target as a non-fatal warning
// ("Failed to locate a schema ... Skipping the import."), matching libxml2.
// A resource-limit guard, however, must not be silently defeated by that
// demotion: an FS whose Open error wraps a value satisfying this interface (and
// returning true) is propagated as a fatal compilation error. The wrapped error
// chain is preserved, so callers can still errors.Is/errors.As the underlying
// cause (e.g. a resource-too-large sentinel).
type FatalSchemaLoader interface {
	FatalSchemaLoad() bool
}

// IsFatalSchemaLoad reports whether err (or anything in its chain) is a fatal
// schema-load condition that must ABORT compilation rather than be demoted to a
// warning or papered over by a fallback to a pre-compiled schema. It is the
// single source of truth for this classification, shared by the xsd import
// warn-and-continue paths and by xslt3's xsl:import-schema fallback guard so the
// two layers cannot drift apart.
//
// A condition is fatal when err (or anything it wraps) satisfies ANY of:
//
//   - the schemaLocation escaped its base directory via ".." ([errSchemaPathEscape]);
//   - xs:import recursion exceeded the configured depth ([errImportDepthExceeded]);
//   - xs:include/xs:redefine nesting exceeded the configured depth
//     ([errIncludeDepthExceeded]) — otherwise an over-deep include/redefine chain
//     inside an IMPORTED schema would be demoted to a warning and silently ignored
//     by loadImport's nested-processing fallback;
//   - a nested schema exceeded the byte cap while being read ([errSchemaTooLarge]);
//   - the configured [fs.FS] returned an error satisfying [FatalSchemaLoader]
//     (e.g. a resource-limit breach such as a too-large external resource).
//
// Everything else — a genuine "schema not found" miss or a not-applicable
// error — is non-fatal and may fall back / warn as before. All sentinels are
// matched via errors.Is / errors.As, so they remain unexported; this helper is
// the public surface.
func IsFatalSchemaLoad(err error) bool {
	if errors.Is(err, errSchemaPathEscape) || errors.Is(err, errImportDepthExceeded) || errors.Is(err, errIncludeDepthExceeded) || errors.Is(err, errSchemaTooLarge) {
		return true
	}
	var f FatalSchemaLoader
	return errors.As(err, &f) && f.FatalSchemaLoad()
}

// validateSchemaPath resolves an xs:include/xs:import/xs:redefine
// schemaLocation against baseDir and returns the name handed to the configured
// fs.FS. It is a thin wrapper over [ResolveSchemaURI], the single canonical
// URI-resolution helper shared with xslt3 (so the two layers cannot drift).
func validateSchemaPath(baseDir, location string) (string, error) {
	return ResolveSchemaURI(location, baseDir)
}

// schemaBaseDir returns the base used to resolve nested includes/imports of
// the schema located at loc. For a URI loc the base is the URI itself (RFC
// 3986 resolution replaces the last path segment), so it is returned verbatim;
// for a local filesystem path it is the containing directory.
func schemaBaseDir(loc string) string {
	if uriScheme(loc) != "" {
		return loc
	}
	// loc is an fs.FS key in forward-slash form (see ResolveSchemaURI); derive
	// its parent directory with path.Dir so the result stays slash-separated on
	// every OS rather than gaining backslashes via filepath.Dir on Windows.
	return path.Dir(uripath.ToSlash(loc))
}

// schemaDisplayLoc builds the human-readable location shown in import/include
// diagnostics: the raw schemaLocation resolved against the parent schema
// reference (filename). URI-aware so absolute URIs and URI bases are not
// collapsed by filepath.
func schemaDisplayLoc(filename, loc string) string {
	if uriScheme(loc) != "" {
		return loc
	}
	if scheme := uriScheme(filename); scheme != "" {
		resolved, err := resolveURIReference(filename, loc)
		if err == nil {
			return resolved
		}
	}
	// Join in forward-slash space so the diagnostic path is stable across OSes
	// (filepath.Join would emit '\' on Windows).
	return path.Join(path.Dir(uripath.ToSlash(filename)), uripath.ToSlash(loc))
}

// processIncludes handles xs:include and xs:import elements.
func (c *compiler) processIncludes(ctx context.Context, root *helium.Element) error {
	// Per-document set of xs:override target paths, so the same document overridden
	// twice within one schema document is rejected (XSD 1.1, W3C over022).
	overrideSeen := make(map[string]struct{})
	for child := range helium.Children(root) {
		if child.Type() != helium.ElementNode {
			continue
		}
		elem, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		switch {
		case isXSDElement(elem, elemInclude):
			loc := getAttr(elem, attrSchemaLocation)
			if loc == "" {
				continue
			}
			if err := c.loadInclude(ctx, loc, elem); err != nil {
				return err
			}
		case isXSDElement(elem, elemImport):
			if err := c.processImport(ctx, elem); err != nil {
				return err
			}
		case isXSDElement(elem, elemRedefine):
			loc := getAttr(elem, attrSchemaLocation)
			if loc == "" {
				continue
			}
			if err := c.loadRedefine(ctx, loc, elem); err != nil {
				return err
			}
		case c.version == Version11 && isXSDElement(elem, elemOverride):
			// xs:override is an XSD 1.1 construct. In 1.0 mode it is ignored
			// (skipped) so existing 1.0 behavior stays byte-identical.
			loc := getAttr(elem, attrSchemaLocation)
			if loc == "" {
				continue
			}
			if !c.recordOverrideTarget(ctx, elem, loc, overrideSeen) {
				continue
			}
			if err := c.loadOverride(ctx, loc, elem); err != nil {
				return err
			}
		}
	}
	return nil
}

// processImport handles a single xs:import element: it enforces the
// already-imported-namespace warning, loads the imported schema, and demotes a
// non-fatal load failure to the established I/O + "Failed to locate" warnings (so
// the import is skipped, matching libxml2). It returns a non-nil error ONLY for a
// fatal load condition ([IsFatalSchemaLoad]) that must abort compilation; every
// non-fatal path returns nil after warning. Shared by processIncludes and the
// xs:override nested processor so an import inside an overridden document gets the
// same diagnostics as a top-level import.
func (c *compiler) processImport(ctx context.Context, elem *helium.Element) error {
	loc := getAttr(elem, attrSchemaLocation)
	if loc == "" {
		return nil
	}
	ns := getAttr(elem, attrNamespace)

	// Check if this namespace was already imported.
	if prevLoc, ok := c.importedNS[ns]; ok && c.filename != "" {
		displayLoc := schemaDisplayLoc(c.filename, loc)
		displayPrevLoc := schemaDisplayLoc(c.filename, prevLoc)
		c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserWarning(c.filename, elem.Line(),
			elem.LocalName(), elemImport,
			"Skipping import of schema located at '"+displayLoc+"' for the namespace '"+ns+"', since this namespace was already imported with the schema located at '"+displayPrevLoc+"'."), helium.ErrorLevelWarning))
		return nil
	}

	if err := c.loadImport(ctx, loc, ns, elem); err != nil {
		// Depth-exceeded and baseDir-escape are security limits, not I/O hiccups;
		// surface them as fatal compilation errors rather than demoting to an I/O
		// warning. A FatalSchemaLoader load failure (e.g. a resource-limit breach
		// from the configured FS) is likewise fatal so the cap cannot be silently
		// defeated for an xs:import target. All conditions route through the one
		// classifier.
		if IsFatalSchemaLoad(err) {
			return err
		}
		// Import failure — report warning if we have a filename.
		if c.filename != "" {
			displayLoc := schemaDisplayLoc(c.filename, loc)
			c.errorHandler.Handle(ctx, helium.NewLeveledError(fmt.Sprintf("I/O warning : failed to load \"%s\": %s\n", displayLoc, "No such file or directory"), helium.ErrorLevelWarning))
			c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserWarning(c.filename, elem.Line(),
				elem.LocalName(), elemImport,
				"Failed to locate a schema at location '"+displayLoc+"'. Skipping the import."), helium.ErrorLevelWarning))
		}
		return nil
	}

	// Track the imported namespace.
	c.importedNS[ns] = loc
	return nil
}

// processNestedIncludes processes the xs:include/xs:import/xs:redefine declared
// by an already-parsed included or redefined schema (incRoot), so a transitive
// chain (main -> inc1 -> inc2) resolves rather than failing on declarations that
// only inc2 defines. The nested references in incRoot are relative to the
// included schema, so baseDir/filename are temporarily switched to it (path is
// its resolved fs key, location its raw schemaLocation). Recursion is bounded by
// the includeVisited loaded-set (registered by the caller) plus a depth cap.
func (c *compiler) processNestedIncludes(ctx context.Context, incRoot *helium.Element, path, location string) error {
	if c.includeDepth >= c.maxIncludeDepth {
		return fmt.Errorf("%w (limit=%d, location=%q)", errIncludeDepthExceeded, c.maxIncludeDepth, location)
	}
	savedBaseDir := c.baseDir
	savedFilename := c.filename
	c.baseDir = schemaBaseDir(path)
	if savedFilename != "" {
		c.filename = schemaDisplayLoc(savedFilename, location)
	}
	c.includeDepth++

	err := c.processIncludes(ctx, incRoot)

	c.includeDepth--
	c.baseDir = savedBaseDir
	c.filename = savedFilename
	return err
}

// loadInclude loads and merges an included schema file.
func (c *compiler) loadInclude(ctx context.Context, location string, includeElem *helium.Element) error {
	path, err := validateSchemaPath(c.baseDir, location)
	if err != nil {
		return fmt.Errorf("xsd: failed to load include %q: %w", location, err)
	}

	// include+override conflict (symmetric to overrideLoadTarget): the document was
	// already transformed by an xs:override, so pulling in its untransformed
	// originals via xs:include would collide. Report the fatal conflict instead of
	// silently skipping.
	if _, overridden := c.overridePaths[path]; overridden {
		c.reportOverrideIncludeConflict(ctx, includeElem, location, elemInclude)
		return nil
	}

	// Load each included document at most once: a transitive/diamond re-include
	// of an already-loaded document is skipped so its declarations are not
	// re-registered and a circular include cannot recurse forever.
	if _, seen := c.includeVisited[path]; seen {
		return nil
	}
	c.includeVisited[path] = struct{}{}

	data, err := c.readNestedSchema(path)
	if err != nil {
		return fmt.Errorf("xsd: failed to load include %q: %w", location, err)
	}

	doc, err := c.parse(ctx, data)
	if err != nil {
		return fmt.Errorf("xsd: failed to parse include %q: %w", location, err)
	}

	incRoot := findDocumentElement(doc)
	if incRoot == nil || !isXSDElement(incRoot, elemSchema) {
		return fmt.Errorf("xsd: included document %q is not an xs:schema", location)
	}

	// Conditional inclusion runs per schema document, BEFORE the targetNamespace
	// compatibility check and the default-attribute interpretation: a vc-excluded
	// root contributes an EMPTY schema and must not be rejected for a TNS mismatch.
	// (For a non-excluded root the same call prunes its vc:-excluded descendants.)
	// c.includeFile is set across the pre-pass so a malformed-vc diagnostic in the
	// included document is attributed to the included file, not the including one,
	// then restored on every path (the later block sets it again for parsing).
	savedIncludeFileVC := c.includeFile
	if c.filename != "" {
		c.includeFile = schemaDisplayLoc(c.filename, location)
	}
	rootExcluded := c.applyConditionalInclusion(ctx, incRoot)
	c.includeFile = savedIncludeFileVC
	if rootExcluded {
		return nil
	}

	// Check target namespace compatibility.
	incTargetNS := getAttr(incRoot, attrTargetNamespace)
	if incTargetNS != "" && incTargetNS != c.schema.targetNamespace {
		displayLoc := location
		if c.filename != "" {
			displayLoc = schemaDisplayLoc(c.filename, location)
		}
		c.schemaError(ctx, schemaParserError(c.filename, includeElem.Line(),
			includeElem.LocalName(), elemInclude,
			"The target namespace '"+incTargetNS+"' of the included/redefined schema '"+displayLoc+"' differs from '"+c.schema.targetNamespace+"' of the including/redefining schema."))
		return nil
	}

	// Chameleon include: if the included schema has no targetNamespace,
	// it adopts the including schema's targetNamespace.
	// The included schema's elementFormDefault/attributeFormDefault are
	// applied within the included declarations.

	// Save current form-qualified and default settings, then apply the included
	// schema's OWN settings. The elementFormDefault/attributeFormDefault/
	// blockDefault/finalDefault attributes are PER schema document, not inherited
	// from the including schema: a chain main -> inc1(elementFormDefault=
	// "qualified") -> inc2(omitted) must parse inc2's declarations as UNQUALIFIED
	// (the spec default), so each document is read against the spec defaults plus
	// only its own declared values. Reset to the spec defaults (unqualified / no
	// flags) before applying this document's attributes; the parent's defaults are
	// restored after processing so siblings are unaffected.
	savedElemForm := c.schema.elemFormQualified
	savedAttrForm := c.schema.attrFormQualified
	savedBlockDefault := c.schema.blockDefault
	savedFinalDefault := c.schema.finalDefault
	savedDefaultAttributes := c.schema.defaultAttributes
	savedDefaultAttrsSet := c.schema.defaultAttrsSet
	savedDefaultAttrsSrc := c.schema.defaultAttrsSrc
	savedIncludeFile := c.includeFile
	savedXPathDefaultNS := c.schemaXPathDefaultNS
	savedSchemaTargetNSSet := c.schemaTargetNSSet
	savedDefaultOpenContent := c.defaultOpenContent
	c.schemaTargetNSSet = c.schema.targetNamespace != ""
	if c.filename != "" {
		c.includeFile = schemaDisplayLoc(c.filename, location)
	}
	c.schema.elemFormQualified = getAttr(incRoot, attrElementFormDefault) == attrValQualified
	c.schema.attrFormQualified = getAttr(incRoot, attrAttributeFormDefault) == attrValQualified
	c.schema.blockDefault = parseBlockFlags(getAttr(incRoot, attrBlockDefault))
	c.schema.finalDefault = parseFinalFlags(getAttr(incRoot, attrFinalDefault))
	// schemaXPathDefaultNS is PER document (used by the SHARED resolveXPathDefaultNS
	// for the included schema's identity-constraint selector/field XPaths AND its
	// xs:assert/xs:assertion XPaths): an included root's @xpathDefaultNamespace must
	// govern its own IDCs/asserts, not inherit the including schema's. Reset to
	// spec-default (none) plus this document's value, RESOLVED against the included
	// root now (so an inherited ##defaultNamespace uses the included root's default
	// namespace).
	c.schemaXPathDefaultNS = ""
	// <xs:defaultOpenContent> is PER document: the included schema's complex types
	// use the included root's own default open content (or none), not the including
	// schema's. Reset and read this document's value before parsing its children.
	c.defaultOpenContent = c.readDefaultOpenContent(ctx, incRoot)
	if c.version == Version11 {
		c.schemaXPathDefaultNS = resolveXPathDefaultNSToken(incRoot, getAttr(incRoot, attrXPathDefaultNS), c.schema.targetNamespace)
		c.readSchemaDefaultAttributes(ctx, incRoot)
	}

	// The CTA static context (static base URI, xpathDefaultNamespace) is PER schema
	// document: an xs:alternative in the included file must see the included
	// document's base URI and its own xpathDefaultNamespace, not the including
	// schema's. Save/restore mirrors the form/default handling above.
	savedSchemaBaseURI := c.schemaBaseURI
	savedCTAXPathDefaultNSSet := c.xpathDefaultNSSet
	savedXPathDefaultNSToken := c.schemaXPathDefaultNSToken
	c.schemaBaseURI = path
	c.xpathDefaultNSSet = hasAttr(incRoot, attrXPathDefaultNamespace)
	c.schemaXPathDefaultNSToken = getAttr(incRoot, attrXPathDefaultNamespace)

	// Snapshot the component-name sets BEFORE parsing so the delta records which
	// components this included document contributes. A later xs:redefine of the
	// same (already-loaded) document needs that set to know which components it
	// may legally override.
	beforeTypes := snapshotKeys(c.schema.types)
	beforeGroups := snapshotKeys(c.schema.groups)
	beforeAttrGroups := snapshotKeys(c.schema.attrGroups)

	// Parse the included schema's declarations into the current compiler.
	// (Conditional inclusion already ran above, before the TNS check.)
	err = c.parseSchemaChildren(ctx, incRoot)

	// Process the included schema's OWN xs:include/xs:import/xs:redefine so a
	// transitive chain resolves (resolved relative to the included schema).
	if err == nil {
		err = c.processNestedIncludes(ctx, incRoot, path, location)
	}

	// Cache the redefinable component set this document contributed so a later
	// xs:redefine of the same already-loaded document can validate its overrides.
	if err == nil {
		c.loadedRedefinable[path] = &redefinableSet{
			keys:     c.computeRedefinableKeys(beforeTypes, beforeGroups, beforeAttrGroups),
			consumed: make(map[redefineKind]map[QName]struct{}),
		}
	}

	// Restore form-qualified settings, defaults, CTA static context, and include file.
	c.schema.elemFormQualified = savedElemForm
	c.schema.attrFormQualified = savedAttrForm
	c.schema.blockDefault = savedBlockDefault
	c.schema.finalDefault = savedFinalDefault
	c.schema.defaultAttributes = savedDefaultAttributes
	c.schema.defaultAttrsSet = savedDefaultAttrsSet
	c.schema.defaultAttrsSrc = savedDefaultAttrsSrc
	c.schemaBaseURI = savedSchemaBaseURI
	c.xpathDefaultNSSet = savedCTAXPathDefaultNSSet
	c.schemaXPathDefaultNSToken = savedXPathDefaultNSToken
	c.includeFile = savedIncludeFile
	c.schemaXPathDefaultNS = savedXPathDefaultNS
	c.schemaTargetNSSet = savedSchemaTargetNSSet
	c.defaultOpenContent = savedDefaultOpenContent

	return err
}

// snapshotKeys captures the current key set of a component map so a later
// delta (newKeysSince) can isolate the keys added between two points.
func snapshotKeys[V any](m map[QName]V) map[QName]struct{} {
	keys := make(map[QName]struct{}, len(m))
	for qn := range m {
		keys[qn] = struct{}{}
	}
	return keys
}

// newKeysSince returns the keys present in m but absent from before, i.e. the
// components added since the snapshot was taken.
func newKeysSince[V any](m map[QName]V, before map[QName]struct{}) map[QName]struct{} {
	added := make(map[QName]struct{})
	for qn := range m {
		if _, existed := before[qn]; existed {
			continue
		}
		added[qn] = struct{}{}
	}
	return added
}

// computeRedefinableKeys builds the per-kind set of component names newly
// registered (afterKeys - beforeKeys) by loading a schema document, splitting
// the type names into simpleType/complexType via c.typeKinds. These are exactly
// the components an xs:redefine of that document may override. It is computed
// once when the document is first loaded (xs:include or xs:redefine) and cached
// in c.loadedRedefinable, since the delta cannot be reconstructed once the
// document's components are merged into the schema.
func (c *compiler) computeRedefinableKeys(beforeTypes, beforeGroups, beforeAttrGroups map[QName]struct{}) map[redefineKind]map[QName]struct{} {
	newTypes := newKeysSince(c.schema.types, beforeTypes)
	phaseASimpleTypes := make(map[QName]struct{})
	phaseAComplexTypes := make(map[QName]struct{})
	for qn := range newTypes {
		switch c.typeKinds[qn] {
		case redefineKindComplexType:
			phaseAComplexTypes[qn] = struct{}{}
		default:
			// simpleType (and builtin/anySimpleType fallbacks) are treated as
			// simple; only an xs:simpleType override may consume them.
			phaseASimpleTypes[qn] = struct{}{}
		}
	}
	return map[redefineKind]map[QName]struct{}{
		redefineKindSimpleType:  phaseASimpleTypes,
		redefineKindComplexType: phaseAComplexTypes,
		redefineKindGroup:       newKeysSince(c.schema.groups, beforeGroups),
		redefineKindAttrGroup:   newKeysSince(c.schema.attrGroups, beforeAttrGroups),
	}
}

// loadRedefine loads a schema via xs:redefine and processes override children.
// It works like xs:include (merging original declarations) but then applies
// redefinitions for complexType, simpleType, group, and attributeGroup children.
func (c *compiler) loadRedefine(ctx context.Context, location string, redefineElem *helium.Element) error {
	// Phase A: Load the redefined schema (same as include).
	path, err := validateSchemaPath(c.baseDir, location)
	if err != nil {
		return fmt.Errorf("xsd: failed to load redefine %q: %w", location, err)
	}

	// include+override conflict (symmetric to overrideLoadTarget): a document
	// already transformed by an xs:override cannot also be redefined.
	if _, overridden := c.overridePaths[path]; overridden {
		c.reportOverrideIncludeConflict(ctx, redefineElem, location, elemRedefine)
		return nil
	}

	// A redefine whose target document is ALREADY loaded — via a prior xs:include
	// or an earlier xs:redefine of the same schema — must not silently drop its
	// override children. Re-running Phase A would re-register the redefined
	// schema's declarations (duplicate components) or recurse forever, so loading
	// is correctly skipped; but XSD permits multiple xs:redefine of the same
	// document (redefining disjoint components, or a no-op repeat), so the
	// document path repeating is NOT itself an error. Process this redefine's
	// override children against the cached Phase-A component set instead. The
	// shared consumed set rejects only a TRUE duplicate — a component an earlier
	// xs:redefine of the same document already redefined.
	if _, seen := c.includeVisited[path]; seen {
		rs := c.loadedRedefinable[path]
		var phaseAKeys, consumed map[redefineKind]map[QName]struct{}
		if rs != nil {
			phaseAKeys = rs.keys
			consumed = rs.consumed
		} else {
			// The document was registered without a recorded redefinable set
			// (e.g. the root schema seeded into includeVisited by CompileFile, or
			// an imported schema's own seed). Nothing from it is overridable, so
			// every override is reported as a duplicate by the override loop.
			phaseAKeys = map[redefineKind]map[QName]struct{}{
				redefineKindSimpleType:  {},
				redefineKindComplexType: {},
				redefineKindGroup:       {},
				redefineKindAttrGroup:   {},
			}
		}
		return c.processRedefineOverrides(ctx, redefineElem, phaseAKeys, consumed)
	}
	c.includeVisited[path] = struct{}{}

	data, err := c.readNestedSchema(path)
	if err != nil {
		return fmt.Errorf("xsd: failed to load redefine %q: %w", location, err)
	}

	doc, err := c.parse(ctx, data)
	if err != nil {
		return fmt.Errorf("xsd: failed to parse redefine %q: %w", location, err)
	}

	incRoot := findDocumentElement(doc)
	if incRoot == nil || !isXSDElement(incRoot, elemSchema) {
		return fmt.Errorf("xsd: redefined document %q is not an xs:schema", location)
	}

	// Conditional inclusion runs per schema document, BEFORE the targetNamespace
	// check and default-attribute interpretation: a vc-excluded root contributes an
	// EMPTY schema (no Phase-A components) and must not be rejected for a TNS
	// mismatch. (For a non-excluded root the same call prunes its descendants.)
	// c.includeFile is set across the pre-pass so a malformed-vc diagnostic in the
	// redefined document is attributed to that file, then restored on every path.
	savedIncludeFileVC := c.includeFile
	if c.filename != "" {
		c.includeFile = schemaDisplayLoc(c.filename, location)
	}
	rootExcluded := c.applyConditionalInclusion(ctx, incRoot)
	c.includeFile = savedIncludeFileVC
	if rootExcluded {
		// The redefined document's root is vc-excluded, so it contributes NO
		// Phase-A components. The <xs:redefine> override children (which live in the
		// REDEFINING schema, not the excluded document) must STILL be validated
		// against that empty target set: XSD rejects an override whose Phase-A
		// target does not exist, so an override of a now-absent component is an
		// error, not a silent no-op. (path is already in includeVisited from above;
		// c.includeFile/form-defaults are the redefining schema's, correct for
		// override-local diagnostics and declarations.)
		emptyKeys := map[redefineKind]map[QName]struct{}{
			redefineKindSimpleType:  {},
			redefineKindComplexType: {},
			redefineKindGroup:       {},
			redefineKindAttrGroup:   {},
		}
		rs := &redefinableSet{
			keys:     emptyKeys,
			consumed: make(map[redefineKind]map[QName]struct{}),
		}
		c.loadedRedefinable[path] = rs
		return c.processRedefineOverrides(ctx, redefineElem, rs.keys, rs.consumed)
	}

	// Check target namespace compatibility (same rules as include).
	incTargetNS := getAttr(incRoot, attrTargetNamespace)
	if incTargetNS != "" && incTargetNS != c.schema.targetNamespace {
		displayLoc := location
		if c.filename != "" {
			displayLoc = schemaDisplayLoc(c.filename, location)
		}
		c.schemaError(ctx, schemaParserError(c.filename, redefineElem.Line(),
			redefineElem.LocalName(), elemRedefine,
			"The target namespace '"+incTargetNS+"' of the included/redefined schema '"+displayLoc+"' differs from '"+c.schema.targetNamespace+"' of the including/redefining schema."))
		return nil
	}

	// Save/restore form-qualified settings and defaults (chameleon support).
	// As with xs:include, the elementFormDefault/attributeFormDefault/
	// blockDefault/finalDefault attributes are PER schema document and are NOT
	// inherited from the redefining schema: reset to the spec defaults
	// (unqualified / no flags) before applying this document's own declared
	// values, so a redefined schema that omits them parses its declarations
	// against the spec defaults rather than the parent's settings.
	savedElemForm := c.schema.elemFormQualified
	savedAttrForm := c.schema.attrFormQualified
	savedBlockDefault := c.schema.blockDefault
	savedFinalDefault := c.schema.finalDefault
	savedDefaultAttributes := c.schema.defaultAttributes
	savedDefaultAttrsSet := c.schema.defaultAttrsSet
	savedDefaultAttrsSrc := c.schema.defaultAttrsSrc
	savedIncludeFile := c.includeFile
	savedXPathDefaultNS := c.schemaXPathDefaultNS
	savedSchemaTargetNSSet := c.schemaTargetNSSet
	savedDefaultOpenContent := c.defaultOpenContent
	c.schemaTargetNSSet = c.schema.targetNamespace != ""
	if c.filename != "" {
		c.includeFile = schemaDisplayLoc(c.filename, location)
	}
	c.schema.elemFormQualified = getAttr(incRoot, attrElementFormDefault) == attrValQualified
	c.schema.attrFormQualified = getAttr(incRoot, attrAttributeFormDefault) == attrValQualified
	c.schema.blockDefault = parseBlockFlags(getAttr(incRoot, attrBlockDefault))
	c.schema.finalDefault = parseFinalFlags(getAttr(incRoot, attrFinalDefault))
	// schemaXPathDefaultNS is PER document, like the form/block/final defaults (see
	// loadInclude): the redefined root's @xpathDefaultNamespace governs its own
	// identity-constraint AND xs:assert/xs:assertion XPaths during Phase A, RESOLVED
	// against the redefined root now (so an inherited ##defaultNamespace uses that
	// root's default ns).
	c.schemaXPathDefaultNS = ""
	// Phase A parses the REDEFINED document's own declarations, so they use the
	// redefined root's default open content (the override children, parsed in Phase
	// B below, get the redefining schema's default restored before then).
	c.defaultOpenContent = c.readDefaultOpenContent(ctx, incRoot)
	if c.version == Version11 {
		c.schemaXPathDefaultNS = resolveXPathDefaultNSToken(incRoot, getAttr(incRoot, attrXPathDefaultNS), c.schema.targetNamespace)
		c.readSchemaDefaultAttributes(ctx, incRoot)
	}
	// The CTA static context (base URI + xpathDefaultNamespace) is per-document too:
	// Phase A parses the REDEFINED document's declarations, so set them here; the
	// override children (from the redefining schema) get the parent's values restored
	// before processRedefineOverrides, like the defaults above.
	savedSchemaBaseURI := c.schemaBaseURI
	savedCTAXPathDefaultNSSet := c.xpathDefaultNSSet
	savedXPathDefaultNSToken := c.schemaXPathDefaultNSToken
	c.schemaBaseURI = path
	c.xpathDefaultNSSet = hasAttr(incRoot, attrXPathDefaultNamespace)
	c.schemaXPathDefaultNSToken = getAttr(incRoot, attrXPathDefaultNamespace)
	// Snapshot the component-name sets per kind BEFORE Phase A. The including
	// (main) schema's root declarations are already registered at this point,
	// so taking the snapshot after Phase A would wrongly treat pre-existing
	// main-schema components as redefinable. Only names ACTUALLY loaded from the
	// redefined schema (afterKeys - beforeKeys) may be overridden.
	beforeTypes := snapshotKeys(c.schema.types)
	beforeGroups := snapshotKeys(c.schema.groups)
	beforeAttrGroups := snapshotKeys(c.schema.attrGroups)

	// Parse the included schema's declarations into the current compiler.
	// (Conditional inclusion already ran above, before the TNS check.)
	if err := c.parseSchemaChildren(ctx, incRoot); err != nil {
		c.schema.elemFormQualified = savedElemForm
		c.schema.attrFormQualified = savedAttrForm
		c.schema.blockDefault = savedBlockDefault
		c.schema.finalDefault = savedFinalDefault
		c.schema.defaultAttributes = savedDefaultAttributes
		c.schema.defaultAttrsSet = savedDefaultAttrsSet
		c.schema.defaultAttrsSrc = savedDefaultAttrsSrc
		c.schemaBaseURI = savedSchemaBaseURI
		c.xpathDefaultNSSet = savedCTAXPathDefaultNSSet
		c.schemaXPathDefaultNSToken = savedXPathDefaultNSToken
		c.includeFile = savedIncludeFile
		c.schemaXPathDefaultNS = savedXPathDefaultNS
		c.schemaTargetNSSet = savedSchemaTargetNSSet
		c.defaultOpenContent = savedDefaultOpenContent
		return err
	}

	// Process the redefined schema's OWN xs:include/xs:import/xs:redefine so a
	// transitive chain resolves (resolved relative to the redefined schema).
	if err := c.processNestedIncludes(ctx, incRoot, path, location); err != nil {
		c.schema.elemFormQualified = savedElemForm
		c.schema.attrFormQualified = savedAttrForm
		c.schema.blockDefault = savedBlockDefault
		c.schema.finalDefault = savedFinalDefault
		c.schema.defaultAttributes = savedDefaultAttributes
		c.schema.defaultAttrsSet = savedDefaultAttrsSet
		c.schema.defaultAttrsSrc = savedDefaultAttrsSrc
		c.schemaBaseURI = savedSchemaBaseURI
		c.xpathDefaultNSSet = savedCTAXPathDefaultNSSet
		c.schemaXPathDefaultNSToken = savedXPathDefaultNSToken
		c.includeFile = savedIncludeFile
		c.schemaXPathDefaultNS = savedXPathDefaultNS
		c.schemaTargetNSSet = savedSchemaTargetNSSet
		c.defaultOpenContent = savedDefaultOpenContent
		return err
	}

	// Phase B: compute the redefinable component set as (afterKeys - beforeKeys)
	// per kind — only the names ACTUALLY loaded by the redefined schema (not
	// pre-existing main-schema components) may be overridden. Cache it keyed by
	// the resolved document path so a later xs:redefine of this same document can
	// validate its overrides against it (the delta cannot be recomputed once the
	// components are merged), then process this redefine's overrides against it.
	phaseAKeys := c.computeRedefinableKeys(beforeTypes, beforeGroups, beforeAttrGroups)
	rs := &redefinableSet{
		keys:     phaseAKeys,
		consumed: make(map[redefineKind]map[QName]struct{}),
	}
	c.loadedRedefinable[path] = rs

	// The override children come from the REDEFINING (main) schema, not the
	// redefined (base) schema loaded in Phase A. Restore c.includeFile to the
	// redefining file's label so duplicate-override diagnostics report the
	// correct source file and line; Phase A above needed the base label.
	// Likewise restore the per-document defaults to the REDEFINING schema's
	// values: override-local declarations must use the redefining schema's
	// elementFormDefault/attributeFormDefault/blockDefault/finalDefault (per
	// XSD), not the redefined document's values applied for Phase A above.
	c.schema.elemFormQualified = savedElemForm
	c.schema.attrFormQualified = savedAttrForm
	c.schema.blockDefault = savedBlockDefault
	c.schema.finalDefault = savedFinalDefault
	c.schema.defaultAttributes = savedDefaultAttributes
	c.schema.defaultAttrsSet = savedDefaultAttrsSet
	c.schema.defaultAttrsSrc = savedDefaultAttrsSrc
	c.schemaBaseURI = savedSchemaBaseURI
	c.xpathDefaultNSSet = savedCTAXPathDefaultNSSet
	c.schemaXPathDefaultNSToken = savedXPathDefaultNSToken
	c.includeFile = savedIncludeFile
	c.schemaXPathDefaultNS = savedXPathDefaultNS
	c.schemaTargetNSSet = savedSchemaTargetNSSet
	// Override children belong to the REDEFINING schema, so they use ITS default
	// open content (restored here), not the redefined document's Phase A value.
	c.defaultOpenContent = savedDefaultOpenContent

	return c.processRedefineOverrides(ctx, redefineElem, phaseAKeys, rs.consumed)
}

// processRedefineOverrides applies the override children of an xs:redefine
// element against phaseAKeys, the component set loaded from the redefined
// document. consumed, when non-nil, is the cross-redefine consumption set shared
// with the document's redefinableSet cache, so a component already redefined by
// an EARLIER xs:redefine of the same document is rejected as a duplicate. Each
// accepted override replaces its same-named Phase-A component exactly once; an
// override targeting a name absent from Phase A, repeated within this element,
// or already consumed by an earlier redefine is reported as a duplicate. The
// override children belong to the REDEFINING schema, so the caller must have
// restored that schema's per-document defaults and include-file label first.
func (c *compiler) processRedefineOverrides(ctx context.Context, redefineElem *helium.Element, phaseAKeys, consumed map[redefineKind]map[QName]struct{}) error {
	savedElemForm := c.schema.elemFormQualified
	savedAttrForm := c.schema.attrFormQualified
	savedBlockDefault := c.schema.blockDefault
	savedFinalDefault := c.schema.finalDefault
	savedIncludeFile := c.includeFile
	c.redefine = &redefineState{
		phaseAKeys: phaseAKeys,
		seen:       make(map[redefineKind]map[QName]struct{}),
		consumed:   consumed,
	}
	defer func() { c.redefine = nil }()
	for child := range helium.Children(redefineElem) {
		if child.Type() != helium.ElementNode {
			continue
		}
		elem, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		switch {
		case isXSDElement(elem, elemAnnotation):
			// skip
		case isXSDElement(elem, elemComplexType):
			name := getAttr(elem, attrName)
			if name == "" {
				continue
			}
			qn := QName{Local: name, NS: c.schema.targetNamespace}
			// Validate and consume the override target before any parse side
			// effects: it must name a type loaded from the redefined schema
			// (Phase A) and may be overridden only once.
			if !c.consumeRedefineTarget(ctx, elem, redefineKindComplexType, qn, "complexType", "A global type definition") {
				continue
			}
			origType := c.schema.types[qn]
			if err := c.parseNamedComplexType(ctx, elem); err != nil {
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
			if !c.consumeRedefineTarget(ctx, elem, redefineKindSimpleType, qn, "simpleType", "A global type definition") {
				continue
			}
			origType := c.schema.types[qn]
			if err := c.parseNamedSimpleType(ctx, elem); err != nil {
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
			if !c.consumeRedefineTarget(ctx, elem, redefineKindGroup, qn, "group", "A global model group definition") {
				continue
			}
			origGroup := c.schema.groups[qn]
			// Snapshot existing groupRefs keys.
			existingRefs := make(map[*ModelGroup]bool, len(c.groupRefs))
			for mg := range c.groupRefs {
				existingRefs[mg] = true
			}
			if err := c.parseNamedGroup(ctx, elem); err != nil {
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
						// The self-reference resolves to the original group's
						// content. resolveRefs deletes this entry from groupRefs
						// before it can run checkAllGroupRef, so the all-group
						// placement rule (cos-all-limited) would be bypassed for a
						// redefine that nests an all-group self-reference inside a
						// sequence/choice. Enforce it here, while the source record
						// is still available, before the entry is removed.
						if origGroup.Compositor == CompositorAll {
							c.checkAllGroupRef(ctx, mg)
						}
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
			// This case writes c.schema.attrGroups directly (bypassing
			// parseNamedAttributeGroup), so enforce the redefine duplicate rule
			// here: the override must target a Phase-A attribute group and may
			// consume it only once. A target absent from Phase A or repeated is
			// reported and skipped.
			if !c.consumeRedefineTarget(ctx, elem, redefineKindAttrGroup, qn, "attributeGroup", "A global attribute group definition") {
				continue
			}
			origAttrs := c.schema.attrGroups[qn]
			// The override REPLACES the Phase-A attribute group, so the nested
			// attribute-group ref set must be rebuilt from the redefining group's
			// children. Snapshot the Phase-A refs first (for self-reference
			// expansion), then clear the slot so stale Phase-A refs cannot leak and
			// the override's own non-self refs are recorded below. Without this,
			// checkAttrGroupDuplicates would flatten the wrong reference set (old
			// refs leak, new refs are ignored).
			origRefChildren := c.attrGroupRefChildren[qn]
			origRefSources := c.attrGroupRefSources[qn]
			delete(c.attrGroupRefChildren, qn)
			delete(c.attrGroupRefSources, qn)
			// XSD 1.1: the override REPLACES the Phase-A group's xs:anyAttribute too.
			// Snapshot the original group wildcard (a self-reference re-contributes
			// it) and clear the slot so a stale base wildcard cannot leak.
			origWildcard := c.attrGroupWildcards[qn]
			delete(c.attrGroupWildcards, qn)
			var ownWildcard, selfRefWildcard *Wildcard
			var anyAttributeSeen bool
			reportAfterWildcard := func(gce *helium.Element) {
				c.schemaError(ctx, schemaParserError(c.diagSource(), gce.Line(), gce.LocalName(), "attributeGroup",
					fmt.Sprintf("The attribute declaration '%s' must appear before the attribute wildcard 'anyAttribute'.", gce.LocalName())))
			}
			// Build the new attribute list manually, expanding self-references
			// inline. parseNamedAttributeGroup only collects xs:attribute children
			// and doesn't handle xs:attributeGroup ref children within a definition.
			var attrs []*AttrUse
			for gc := range helium.Children(elem) {
				if gc.Type() != helium.ElementNode {
					continue
				}
				gce, ok := helium.AsNode[*helium.Element](gc)
				if !ok {
					continue
				}
				switch {
				case isXSDElement(gce, elemAttribute):
					if c.version == Version11 && anyAttributeSeen {
						reportAfterWildcard(gce)
						continue
					}
					// A use="prohibited" attribute is pointless inside an
					// <xs:attributeGroup> (including a redefine override): libxml2
					// warns and skips it so a referencing wildcard still admits the
					// attribute. Mirror parseNamedAttributeGroup here.
					if getAttr(gce, attrUse) == attrValProhibited {
						if c.filename != "" {
							c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserWarning(c.diagSource(), gce.Line(), gce.LocalName(), "attribute",
								"Skipping attribute use prohibition, since it is pointless inside an <attributeGroup>."), helium.ErrorLevelWarning))
						}
						continue
					}
					au := c.parseAttributeUse(ctx, gce)
					attrs = append(attrs, au)
				case c.version == Version11 && isXSDElement(gce, elemAnyAttribute):
					if anyAttributeSeen {
						c.schemaError(ctx, schemaParserError(c.diagSource(), gce.Line(), gce.LocalName(), "attributeGroup",
							fmt.Sprintf("An attribute group definition must not have more than one attribute wildcard (found a second '%s').", gce.LocalName())))
						continue
					}
					anyAttributeSeen = true
					ownWildcard = c.parseAnyAttribute(ctx, gce)
				case isXSDElement(gce, elemAttributeGroup):
					if c.version == Version11 && anyAttributeSeen {
						reportAfterWildcard(gce)
						continue
					}
					if ref := getAttr(gce, attrRef); ref != "" {
						refQN := c.resolveQName(ctx, gce, ref)
						switch refQN {
						case qn:
							// A self-reference resolves to the Phase-A group content,
							// including its xs:anyAttribute wildcard.
							if origAttrs != nil {
								attrs = append(attrs, origAttrs...)
							}
							selfRefWildcard = origWildcard
							if len(origRefChildren) > 0 {
								c.attrGroupRefChildren[qn] = append(c.attrGroupRefChildren[qn], origRefChildren...)
								c.attrGroupRefSources[qn] = append(c.attrGroupRefSources[qn], origRefSources...)
							}
						default:
							// A non-self nested ref in the override is recorded so
							// checkAttrGroupDuplicates flattens the redefining ref set.
							c.attrGroupRefChildren[qn] = append(c.attrGroupRefChildren[qn], refQN)
							c.attrGroupRefSources[qn] = append(c.attrGroupRefSources[qn], attrGroupSource{line: gce.Line(), source: c.diagSource()})
						}
					}
				}
			}
			c.schema.attrGroups[qn] = attrs
			// Store the override's effective group wildcard (Version11): the group's
			// own xs:anyAttribute INTERSECTED with the original wildcard a
			// self-reference re-contributes. The type's "complete wildcard" further
			// intersects the non-self refs recorded above at link time.
			if c.version == Version11 {
				if w := combineGroupWildcards(ownWildcard, selfRefWildcard); w != nil {
					c.attrGroupWildcards[qn] = w
				}
			}
			// The override REPLACES the Phase-A attribute group, so re-record its
			// source to the redefining file/line (c.includeFile is the redefining
			// label here). Without this the duplicate-attribute-use diagnostic over
			// this group would keep the stale Phase-A source from parseNamedAttribute
			// Group and cite the redefined (base) file instead of the redefine.
			c.attrGroupSources[qn] = attrGroupSource{line: elem.Line(), source: c.diagSource()}
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

// loadImport loads an imported schema and merges its declarations. The ns
// argument is the namespace declared on the <xs:import> element; the imported
// schema's targetNamespace must match it (XSD src-import / libxml2 semantics):
// when ns is present the imported schema must declare that targetNamespace, and
// when ns is absent the imported schema must have no targetNamespace. A
// mismatch is a fatal schema error, not an I/O warning, so importElem carries
// the source line for the diagnostic.
func (c *compiler) loadImport(ctx context.Context, location, ns string, importElem *helium.Element) error {
	// Bound the import recursion. Each sub-compiler inherits this limit
	// and tracks its own depth so namespace-cycling chains (A → B → C → A …)
	// cannot exhaust memory / stack even when every link uses a distinct
	// namespace URI.
	if c.importDepth+1 > c.maxImportDepth {
		return fmt.Errorf("%w (limit=%d, location=%q)", errImportDepthExceeded, c.maxImportDepth, location)
	}

	path, err := validateSchemaPath(c.baseDir, location)
	if err != nil {
		return fmt.Errorf("xsd: failed to load import %q: %w", location, err)
	}

	data, err := c.readNestedSchema(path)
	if err != nil {
		return fmt.Errorf("xsd: failed to load import %q: %w", location, err)
	}

	doc, err := c.parse(ctx, data)
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
		impFilename = schemaDisplayLoc(c.filename, location)
	}

	// Create a temporary compiler for the imported schema. The imported schema is
	// compiled under the importing schema's effective XSD version.
	impC := &compiler{
		schema: &Schema{
			version:     c.version,
			elements:    make(map[QName]*ElementDecl),
			types:       make(map[QName]*TypeDef),
			groups:      make(map[QName]*ModelGroup),
			attrGroups:  make(map[QName][]*AttrUse),
			globalAttrs: make(map[QName]*AttrUse),
			substGroups: make(map[QName][]*ElementDecl),
		},
		version:                   c.version,
		baseDir:                   schemaBaseDir(path),
		fsys:                      c.fsys,
		parser:                    c.parser,
		typeRefs:                  make(map[*TypeDef]QName),
		elemRefs:                  make(map[*ElementDecl]QName),
		elemRefSources:            make(map[*ElementDecl]elemRefSource),
		groupRefs:                 make(map[*ModelGroup]QName),
		groupRefSources:           make(map[*ModelGroup]groupRefSource),
		groupSources:              make(map[QName]groupSource),
		attrGroupSources:          make(map[QName]attrGroupSource),
		attrGroupRefs:             make(map[*TypeDef][]QName),
		attrGroupRefUseSources:    make(map[*TypeDef][]attrGroupRefUseSource),
		defaultAttrUses:           make(map[*TypeDef]map[QName]*AttrUse),
		attrGroupRefChildren:      make(map[QName][]QName),
		attrGroupRefSources:       make(map[QName][]attrGroupSource),
		attrGroupWildcards:        make(map[QName]*Wildcard),
		globalElemSources:         make(map[*ElementDecl]elemRefSource),
		typeDefSources:            make(map[*TypeDef]typeDefSource),
		typeKinds:                 make(map[QName]redefineKind),
		itemTypeRefs:              make(map[*TypeDef]QName),
		chameleonEligible:         make(map[any]struct{}),
		attrRefs:                  make(map[*AttrUse]QName),
		attrUseConstraintSources:  make(map[*AttrUse]attrConstraintSource),
		attrUseSources:            make(map[*AttrUse]attrConstraintSource),
		elemDeclConstraintSources: make(map[*ElementDecl]attrConstraintSource),
		filename:                  impFilename,
		importedNS:                make(map[string]string),
		importDepth:               c.importDepth + 1,
		maxImportDepth:            c.maxImportDepth,
		includeVisited:            make(map[string]struct{}),
		maxIncludeDepth:           c.maxIncludeDepth,
		loadedRedefinable:         make(map[string]*redefinableSet),
		notations:                 make(map[QName]struct{}),
	}

	// Seed the imported sub-compiler's circular-include guard with the imported
	// schema's own resolved key, mirroring CompileFile's seeding of the top-level
	// root. Without this, an imported schema that circularly includes back to its
	// own root (import imp.xsd -> include inc.xsd -> include imp.xsd) re-parses
	// imp.xsd and emits spurious duplicate-component errors.
	impC.includeVisited[path] = struct{}{}
	// The imported schema is this sub-compiler's root: record it so an xs:override
	// cascade inside the imported schema that points back at its own root
	// terminates without re-loading it (mirrors the top-level rootKey seeding).
	impC.rootKey = path

	// Sub-compiler collects errors into its own collector so we can
	// conditionally forward them. This matches libxml2's behavior of
	// stopping error reporting after the first import failure.
	subCollector := helium.NewErrorCollector(ctx, helium.ErrorLevelNone)
	// Guarantee the sub-collector's backing sink is drained on every exit path,
	// including the fatal early returns below (parseSchemaChildren failure and the
	// include-depth/path-escape/resource-limit fatal nested-load). Close is
	// idempotent, so the explicit Close before the Errors() read below still runs
	// and the collected diagnostics remain available for forwarding.
	defer func() { _ = subCollector.Close() }()
	impC.errorHandler = subCollector

	// propagateImpErrors drains the import sub-compiler's collected diagnostics and
	// folds its error count into the parent. Preserving libxml2's "stop after the
	// first import failure" rule, the diagnostic TEXT is forwarded only when the
	// parent has no prior errors, but impC.errorCount is ALWAYS added so an
	// imported-schema failure still fails the compile. It is IDEMPOTENT (a
	// `propagated` guard makes the second and later calls no-ops) and is `defer`red
	// immediately below so EVERY exit path after the sub-collector is installed —
	// including the parseSchemaChildren-error and fatal nested-load early returns —
	// flushes the sub-compiler's diagnostics. The explicit calls that remain only
	// fix ORDERING (forward while the parent is still error-free, BEFORE a TNS error
	// is reported; skip the declaration merge when the import failed). Close is
	// idempotent, so the separate Close defer above stays harmless.
	propagated := false
	propagateImpErrors := func() {
		if propagated {
			return
		}
		propagated = true
		_ = subCollector.Close()
		if impC.errorCount == 0 {
			return
		}
		if c.errorCount == 0 {
			for _, e := range subCollector.Errors() {
				c.errorHandler.Handle(ctx, e)
			}
		}
		c.errorCount += impC.errorCount
	}
	// Guaranteed flush on every remaining exit path (idempotent; explicit calls
	// below run first where ordering matters and turn this into a no-op).
	defer propagateImpErrors()

	impC.schema.targetNamespace = getAttr(impRoot, attrTargetNamespace)
	impC.schemaTargetNSSet = impC.schema.targetNamespace != ""
	impC.schema.elemFormQualified = getAttr(impRoot, attrElementFormDefault) == attrValQualified
	impC.schema.attrFormQualified = getAttr(impRoot, attrAttributeFormDefault) == attrValQualified
	if v := getAttr(impRoot, attrBlockDefault); v != "" {
		impC.schema.blockDefault = parseBlockFlags(v)
	}
	if v := getAttr(impRoot, attrFinalDefault); v != "" {
		impC.schema.finalDefault = parseFinalFlags(v)
	}
	// The imported root's @xpathDefaultNamespace governs its own
	// identity-constraint selector/field XPaths (resolveXPathDefaultNS reads the
	// sub-compiler's schemaXPathDefaultNS); without this an imported IDC selector
	// like xpath="emp" would not inherit the imported root's default namespace.
	// Resolved against the imported root now (so an inherited ##defaultNamespace
	// uses the imported root's default namespace, not a selector/field's).
	if impC.version == Version11 {
		impC.schemaXPathDefaultNS = resolveXPathDefaultNSToken(impRoot, getAttr(impRoot, attrXPathDefaultNS), impC.schema.targetNamespace)
	}

	// Seed the imported sub-compiler's CTA static context from the IMPORTED document
	// so an xs:alternative parsed there sees its own schema's static base URI
	// (fn:static-base-uri) and xpathDefaultNamespace, not the importing schema's.
	impC.schemaBaseURI = path
	if hasAttr(impRoot, attrXPathDefaultNamespace) {
		impC.xpathDefaultNSSet = true
	}
	impC.schemaXPathDefaultNSToken = getAttr(impRoot, attrXPathDefaultNamespace)

	registerBuiltinTypes(impC.schema, impC.version)

	// Conditional inclusion runs per schema document, BEFORE the src-import
	// targetNamespace check: a vc-excluded imported root contributes an EMPTY
	// schema and must not be rejected for a namespace mismatch. A malformed 1.1 vc
	// value on the (excluded or non-excluded) root is still a schema error, so its
	// diagnostics are propagated on every exit path below.
	if impC.applyConditionalInclusion(ctx, impRoot) {
		propagateImpErrors()
		return nil
	}

	// The targetNamespace of the located schema must match the namespace
	// declared on <xs:import> (XSD src-import; libxml2 rejects the mismatch
	// rather than merging declarations from the wrong namespace). A present
	// namespace requires that exact targetNamespace; an absent namespace
	// requires the imported schema to have no targetNamespace. This is a
	// fatal schema error, not an I/O warning, so it is emitted directly here
	// and reported via a nil return (mirroring the xs:include check above).
	// Pre-pass diagnostics collected in impC are flushed FIRST (while the parent
	// is still error-free) so they are not dropped by this early return.
	impTargetNS := getAttr(impRoot, attrTargetNamespace)
	if impTargetNS != ns {
		propagateImpErrors()
		displayLoc := location
		if c.filename != "" {
			displayLoc = schemaDisplayLoc(c.filename, location)
		}
		c.schemaError(ctx, schemaParserError(c.filename, importElem.Line(),
			importElem.LocalName(), elemImport,
			"The namespace '"+impTargetNS+"' of the imported schema '"+displayLoc+"' differs from the requested namespace '"+ns+"'."))
		return nil
	}

	if impC.version == Version11 {
		impC.readSchemaDefaultAttributes(ctx, impRoot)
		// The imported document's own <xs:defaultOpenContent> applies to its complex
		// types (it is per-document and does not cross the import boundary). Read it
		// AFTER applyConditionalInclusion (above), matching xs:include/xs:redefine, so
		// a vc-excluded <xs:defaultOpenContent> is not captured and applied.
		impC.defaultOpenContent = impC.readDefaultOpenContent(ctx, impRoot)
	}

	if err := impC.parseSchemaChildren(ctx, impRoot); err != nil {
		return err
	}

	// Process includes/imports in the imported schema (but skip back-references).
	if err := impC.processIncludes(ctx, impRoot); err != nil {
		// Depth-exceeded errors propagate so a hostile import cycle
		// is reported to the caller rather than being silently truncated.
		// A baseDir-escape is a security limit, fatal exactly like in the
		// outer import path, so a nested escaping schemaLocation cannot be
		// swallowed as an I/O warning. A FatalSchemaLoader load failure
		// (e.g. a resource-limit breach from the configured FS) propagates
		// for the same reason: the cap must not be silently defeated for a
		// nested xs:import target. All three conditions route through the
		// one classifier.
		if IsFatalSchemaLoad(err) {
			return err
		}
		// Other errors in nested processing are non-fatal — the import
		// loader treats failure to load referenced schemas as warnings.
		_ = err
	}

	// Propagate sub-compiler diagnostics (same rule as the early-return paths).
	propagateImpErrors()
	if impC.errorCount > 0 {
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
	maps.Copy(c.elemRefs, impC.elemRefs)
	maps.Copy(c.elemRefSources, impC.elemRefSources)
	maps.Copy(c.typeRefs, impC.typeRefs)
	// Offset each imported type's parse-order ordinal past the parent's counter
	// so ordinals remain globally unique across the merged compilers; otherwise
	// a parent type and an imported type sharing a source line and an empty name
	// could collide on the diagnostic tie-breaker.
	base := c.nextTypeDefOrdinal
	for td, src := range impC.typeDefSources {
		src.ordinal += base
		// Preserve the originating file for imported types. A type parsed
		// directly in the imported document (not via a nested include) has an
		// empty source; attribute it to the imported file so diagnostics cite
		// the file whose line number they carry, not the importing schema.
		if src.source == "" {
			src.source = impC.filename
		}
		c.typeDefSources[td] = src
	}
	c.nextTypeDefOrdinal = base + impC.nextTypeDefOrdinal
	maps.Copy(c.groupRefs, impC.groupRefs)
	maps.Copy(c.groupRefSources, impC.groupRefSources)
	// Merge named-group source info, but only for groups the parent does not
	// already define (mirroring the schema.groups merge above): a group present
	// in both keeps the parent's declaration and source.
	for qn, src := range impC.groupSources {
		if _, exists := c.groupSources[qn]; !exists {
			c.groupSources[qn] = src
		}
	}
	// Merge attribute-group source info (mirroring the schema.attrGroups merge
	// above): a group present in both keeps the parent's declaration and source.
	for qn, src := range impC.attrGroupSources {
		if _, exists := c.attrGroupSources[qn]; !exists {
			// An attribute group parsed directly in the imported document (not via
			// a nested include) has an empty source; attribute it to the imported
			// file so its duplicate-attribute-use diagnostic cites the file whose
			// line number it carries, not the importing schema.
			if src.source == "" {
				src.source = impC.filename
			}
			c.attrGroupSources[qn] = src
		}
	}
	maps.Copy(c.attrGroupRefs, impC.attrGroupRefs)
	for _, ref := range impC.schemaDefaultAttrRefs {
		if ref.src.source == "" {
			ref.src.source = impC.filename
		}
		c.schemaDefaultAttrRefs = append(c.schemaDefaultAttrRefs, ref)
	}
	for td, srcs := range impC.attrGroupRefUseSources {
		merged := make([]attrGroupRefUseSource, len(srcs))
		for i, src := range srcs {
			if src.source == "" {
				src.source = impC.filename
			}
			merged[i] = src
		}
		c.attrGroupRefUseSources[td] = merged
	}
	for qn, refs := range impC.attrGroupRefChildren {
		if _, exists := c.attrGroupRefChildren[qn]; !exists {
			c.attrGroupRefChildren[qn] = refs
		}
	}
	// Merge the XSD 1.1 attribute-group wildcards so a type in the importing
	// schema that references an imported group sees the group's xs:anyAttribute.
	for qn, wc := range impC.attrGroupWildcards {
		if _, exists := c.attrGroupWildcards[qn]; !exists {
			c.attrGroupWildcards[qn] = wc
		}
	}
	// Merge the per-edge ref sources alongside attrGroupRefChildren. A ref edge
	// parsed directly in the imported document (not via a nested include) has an
	// empty source; attribute it to the imported file so an indirect-cycle
	// diagnostic cites the file whose ref line number it carries.
	for qn, srcs := range impC.attrGroupRefSources {
		if _, exists := c.attrGroupRefSources[qn]; exists {
			continue
		}
		merged := make([]attrGroupSource, len(srcs))
		for i, src := range srcs {
			if src.source == "" {
				src.source = impC.filename
			}
			merged[i] = src
		}
		c.attrGroupRefSources[qn] = merged
	}
	maps.Copy(c.globalElemSources, impC.globalElemSources)
	maps.Copy(c.itemTypeRefs, impC.itemTypeRefs)
	maps.Copy(c.chameleonEligible, impC.chameleonEligible)
	c.unionMemberRefs = append(c.unionMemberRefs, impC.unionMemberRefs...)
	maps.Copy(c.attrRefs, impC.attrRefs)
	maps.Copy(c.notations, impC.notations)
	// Merge attribute-use default/fixed constraint sources, preserving the
	// originating file. An attribute use parsed directly in the imported document
	// (not via a nested include) has an empty source; attribute it to the
	// imported file so the invalid-default/fixed diagnostic cites the file whose
	// line number it carries, not the importing schema.
	for au, src := range impC.attrUseConstraintSources {
		if src.source == "" {
			src.source = impC.filename
		}
		c.attrUseConstraintSources[au] = src
	}
	// Merge element-declaration default/fixed constraint sources, mirroring the
	// attribute-use merge above so an invalid element default/fixed in an imported
	// document is still checked and the diagnostic cites the imported file.
	for decl, src := range impC.elemDeclConstraintSources {
		if src.source == "" {
			src.source = impC.filename
		}
		c.elemDeclConstraintSources[decl] = src
	}
	// Merge prohibited/ref'd attribute-use sources, preserving the originating
	// file. An attribute use parsed directly in the imported document (not via a
	// nested include) has an empty source; attribute it to the imported file so
	// warnPointlessProhibition cites the file whose line it carries, not the
	// importing schema.
	for au, src := range impC.attrUseSources {
		if src.source == "" {
			src.source = impC.filename
		}
		c.attrUseSources[au] = src
	}

	// Carry the imported CTA bookkeeping into the parent so imported alternatives
	// participate in the parent's deferred resolution and checks: altTypeRefs are
	// resolved by the parent's resolveAltTypeRefs (against the merged type table),
	// and ctaElems are checked by the parent's checkAltSubstitutability.
	c.altTypeRefs = append(c.altTypeRefs, impC.altTypeRefs...)
	c.ctaElems = append(c.ctaElems, impC.ctaElems...)

	return nil
}
