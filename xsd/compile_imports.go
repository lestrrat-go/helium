package xsd

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"path"

	helium "github.com/lestrrat-go/helium"
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
//   - the configured [fs.FS] returned an error satisfying [FatalSchemaLoader]
//     (e.g. a resource-limit breach such as a too-large external resource).
//
// Everything else — a genuine "schema not found" miss or a not-applicable
// error — is non-fatal and may fall back / warn as before. All sentinels are
// matched via errors.Is / errors.As, so they remain unexported; this helper is
// the public surface.
func IsFatalSchemaLoad(err error) bool {
	if errors.Is(err, errSchemaPathEscape) || errors.Is(err, errImportDepthExceeded) {
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
			loc := getAttr(elem, attrSchemaLocation)
			if loc == "" {
				continue
			}
			ns := getAttr(elem, attrNamespace)

			// Check if this namespace was already imported.
			if prevLoc, ok := c.importedNS[ns]; ok && c.filename != "" {
				displayLoc := schemaDisplayLoc(c.filename, loc)
				displayPrevLoc := schemaDisplayLoc(c.filename, prevLoc)
				c.errorHandler.Handle(ctx, helium.NewLeveledError(schemaParserWarning(c.filename, elem.Line(),
					elem.LocalName(), elemImport,
					"Skipping import of schema located at '"+displayLoc+"' for the namespace '"+ns+"', since this namespace was already imported with the schema located at '"+displayPrevLoc+"'."), helium.ErrorLevelWarning))
				continue
			}

			if err := c.loadImport(ctx, loc, ns, elem); err != nil {
				// Depth-exceeded and baseDir-escape are security limits,
				// not I/O hiccups; surface them as fatal compilation
				// errors rather than demoting to an I/O warning. A
				// FatalSchemaLoader load failure (e.g. a resource-limit
				// breach from the configured FS) is likewise fatal so the
				// cap cannot be silently defeated for an xs:import target.
				// All three conditions route through the one classifier.
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
				continue
			}

			// Track the imported namespace.
			c.importedNS[ns] = loc
		case isXSDElement(elem, elemRedefine):
			loc := getAttr(elem, attrSchemaLocation)
			if loc == "" {
				continue
			}
			if err := c.loadRedefine(ctx, loc, elem); err != nil {
				return err
			}
		}
	}
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

	// Load each included document at most once: a transitive/diamond re-include
	// of an already-loaded document is skipped so its declarations are not
	// re-registered and a circular include cannot recurse forever.
	if _, seen := c.includeVisited[path]; seen {
		return nil
	}
	c.includeVisited[path] = struct{}{}

	data, err := fs.ReadFile(c.fsys, path)
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
		c.includeFile = schemaDisplayLoc(c.filename, location)
	}

	// Parse the included schema's declarations into the current compiler.
	err = c.parseSchemaChildren(ctx, incRoot)

	// Process the included schema's OWN xs:include/xs:import/xs:redefine so a
	// transitive chain resolves (resolved relative to the included schema).
	if err == nil {
		err = c.processNestedIncludes(ctx, incRoot, path, location)
	}

	// Restore form-qualified settings, defaults, and include file.
	c.schema.elemFormQualified = savedElemForm
	c.schema.attrFormQualified = savedAttrForm
	c.schema.blockDefault = savedBlockDefault
	c.schema.finalDefault = savedFinalDefault
	c.includeFile = savedIncludeFile

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

// loadRedefine loads a schema via xs:redefine and processes override children.
// It works like xs:include (merging original declarations) but then applies
// redefinitions for complexType, simpleType, group, and attributeGroup children.
func (c *compiler) loadRedefine(ctx context.Context, location string, redefineElem *helium.Element) error {
	// Phase A: Load the redefined schema (same as include).
	path, err := validateSchemaPath(c.baseDir, location)
	if err != nil {
		return fmt.Errorf("xsd: failed to load redefine %q: %w", location, err)
	}

	// Load each redefined document at most once (shared with xs:include), so a
	// re-redefine cannot re-register declarations or recurse forever.
	if _, seen := c.includeVisited[path]; seen {
		return nil
	}
	c.includeVisited[path] = struct{}{}

	data, err := fs.ReadFile(c.fsys, path)
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
		c.includeFile = schemaDisplayLoc(c.filename, location)
	}

	// Snapshot the component-name sets per kind BEFORE Phase A. The including
	// (main) schema's root declarations are already registered at this point,
	// so taking the snapshot after Phase A would wrongly treat pre-existing
	// main-schema components as redefinable. Only names ACTUALLY loaded from the
	// redefined schema (afterKeys - beforeKeys) may be overridden.
	beforeTypes := snapshotKeys(c.schema.types)
	beforeGroups := snapshotKeys(c.schema.groups)
	beforeAttrGroups := snapshotKeys(c.schema.attrGroups)

	// Parse the included schema's declarations into the current compiler.
	if err := c.parseSchemaChildren(ctx, incRoot); err != nil {
		c.schema.elemFormQualified = savedElemForm
		c.schema.attrFormQualified = savedAttrForm
		c.schema.blockDefault = savedBlockDefault
		c.schema.finalDefault = savedFinalDefault
		c.includeFile = savedIncludeFile
		return err
	}

	// Process the redefined schema's OWN xs:include/xs:import/xs:redefine so a
	// transitive chain resolves (resolved relative to the redefined schema).
	if err := c.processNestedIncludes(ctx, incRoot, path, location); err != nil {
		c.schema.elemFormQualified = savedElemForm
		c.schema.attrFormQualified = savedAttrForm
		c.schema.blockDefault = savedBlockDefault
		c.schema.finalDefault = savedFinalDefault
		c.includeFile = savedIncludeFile
		return err
	}

	// Phase B: Process redefine children (overrides). Each override may replace
	// the same-named component loaded in Phase A exactly once. Compute the
	// Phase-A keys per kind as (afterKeys - beforeKeys) so the duplicate-name
	// checks are suppressed only for the components actually loaded by the
	// redefined schema, consumed once each — not globally and not for
	// pre-existing main-schema components. An override that targets a name not
	// loaded in Phase A, or repeats a name, is reported as a duplicate.
	// Split the newly-loaded type keys by declared kind so a Phase-A simpleType
	// is only redefinable by a simpleType override (and likewise for complex).
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
	phaseAKeys := map[redefineKind]map[QName]struct{}{
		redefineKindSimpleType:  phaseASimpleTypes,
		redefineKindComplexType: phaseAComplexTypes,
		redefineKindGroup:       newKeysSince(c.schema.groups, beforeGroups),
		redefineKindAttrGroup:   newKeysSince(c.schema.attrGroups, beforeAttrGroups),
	}
	c.redefine = &redefineState{
		phaseAKeys: phaseAKeys,
		seen:       make(map[redefineKind]map[QName]struct{}),
	}
	defer func() { c.redefine = nil }()

	// The override children below come from the REDEFINING (main) schema, not
	// the redefined (base) schema loaded in Phase A. Restore c.includeFile to
	// the redefining file's label so duplicate-override diagnostics report the
	// correct source file and line; Phase A above needed the base label.
	c.includeFile = savedIncludeFile
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
				case isXSDElement(gce, elemAttributeGroup):
					if ref := getAttr(gce, attrRef); ref != "" {
						refQN := c.resolveQName(ctx, gce, ref)
						switch refQN {
						case qn:
							// A self-reference resolves to the Phase-A group content.
							if origAttrs != nil {
								attrs = append(attrs, origAttrs...)
							}
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

	data, err := fs.ReadFile(c.fsys, path)
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

	// The targetNamespace of the located schema must match the namespace
	// declared on <xs:import> (XSD src-import; libxml2 rejects the mismatch
	// rather than merging declarations from the wrong namespace). A present
	// namespace requires that exact targetNamespace; an absent namespace
	// requires the imported schema to have no targetNamespace. This is a
	// fatal schema error, not an I/O warning, so it is emitted directly here
	// and reported via a nil return (mirroring the xs:include check above).
	impTargetNS := getAttr(impRoot, attrTargetNamespace)
	if impTargetNS != ns {
		displayLoc := location
		if c.filename != "" {
			displayLoc = schemaDisplayLoc(c.filename, location)
		}
		c.schemaError(ctx, schemaParserError(c.filename, importElem.Line(),
			importElem.LocalName(), elemImport,
			"The namespace '"+impTargetNS+"' of the imported schema '"+displayLoc+"' differs from the requested namespace '"+ns+"'."))
		return nil
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
		baseDir:                  schemaBaseDir(path),
		fsys:                     c.fsys,
		parser:                   c.parser,
		typeRefs:                 make(map[*TypeDef]QName),
		elemRefs:                 make(map[*ElementDecl]QName),
		elemRefSources:           make(map[*ElementDecl]elemRefSource),
		groupRefs:                make(map[*ModelGroup]QName),
		groupRefSources:          make(map[*ModelGroup]groupRefSource),
		groupSources:             make(map[QName]groupSource),
		attrGroupSources:         make(map[QName]attrGroupSource),
		attrGroupRefs:            make(map[*TypeDef][]QName),
		attrGroupRefChildren:     make(map[QName][]QName),
		attrGroupRefSources:      make(map[QName][]attrGroupSource),
		globalElemSources:        make(map[*ElementDecl]elemRefSource),
		typeDefSources:           make(map[*TypeDef]typeDefSource),
		typeKinds:                make(map[QName]redefineKind),
		itemTypeRefs:             make(map[*TypeDef]QName),
		chameleonEligible:        make(map[any]struct{}),
		attrRefs:                 make(map[*AttrUse]QName),
		attrUseConstraintSources: make(map[*AttrUse]attrConstraintSource),
		attrUseSources:           make(map[*AttrUse]attrConstraintSource),
		filename:                 impFilename,
		importedNS:               make(map[string]string),
		importDepth:              c.importDepth + 1,
		maxImportDepth:           c.maxImportDepth,
		includeVisited:           make(map[string]struct{}),
		maxIncludeDepth:          c.maxIncludeDepth,
	}

	// Sub-compiler collects errors into its own collector so we can
	// conditionally forward them. This matches libxml2's behavior of
	// stopping error reporting after the first import failure.
	subCollector := helium.NewErrorCollector(ctx, helium.ErrorLevelNone)
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

	_ = subCollector.Close()

	// Only propagate sub-compiler errors to the parent if the parent has no
	// prior errors. This matches libxml2's behavior of stopping error reporting
	// after the first import failure.
	if impC.errorCount > 0 {
		if c.errorCount == 0 {
			for _, e := range subCollector.Errors() {
				c.errorHandler.Handle(ctx, e)
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
	for qn, refs := range impC.attrGroupRefChildren {
		if _, exists := c.attrGroupRefChildren[qn]; !exists {
			c.attrGroupRefChildren[qn] = refs
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

	return nil
}
