package xsd

import (
	"context"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/iofs"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/xmlchar"
	"github.com/lestrrat-go/helium/internal/xsd/value"
)

// compiler holds state during schema compilation.
type compiler struct {
	schema  *Schema
	version Version // XSD specification version targeted by this compilation
	// schemaXPathDefaultNS is the root xs:schema @xpathDefaultNamespace (XSD 1.1)
	// ALREADY RESOLVED to a default element-namespace URI against the schema
	// root's context (so ##defaultNamespace uses the ROOT's default namespace,
	// not a selector/field that may redeclare it). It is inherited by every
	// identity-constraint selector/field AND every xs:assert/xs:assertion that does
	// not set its own. Empty means no default (unprefixed element = no-namespace).
	// It is re-set per document for xs:include/xs:redefine/xs:import.
	schemaXPathDefaultNS string
	// schemaTargetNSSet tracks whether the current schema document has a non-empty
	// effective target namespace. This is distinct from @targetNamespace presence:
	// targetNamespace="" is no target namespace, while chameleon includes inherit
	// the including schema's effective namespace.
	schemaTargetNSSet bool
	baseDir           string         // directory of the schema file, for resolving relative paths
	fsys              fs.FS          // filesystem for loading xs:include/xs:import/xs:redefine targets
	parser            *helium.Parser // parser governing parse policy for nested include/import/redefine schemas
	// unresolved type references: maps from element/type QName to the type ref string
	typeRefs map[*TypeDef]QName
	elemRefs map[*ElementDecl]QName
	// unresolved xs:alternative (conditional type assignment) @type references,
	// resolved in resolveRefs. A slice (not a map) so nil-append works without
	// per-compiler initialization.
	altTypeRefs []altTypeRef
	// source info for element refs, used in unresolved-type error messages
	elemRefSources map[*ElementDecl]elemRefSource
	// unresolved group references: maps from model group placeholder to group QName
	groupRefs map[*ModelGroup]QName
	// source info for group references, used to enforce the all-group reference
	// constraints (cos-all-limited / cos-group-ref) once the ref resolves: an
	// 'all' model group may only be referenced as the entire content of a
	// complex type, and that reference's maxOccurs must be 1.
	groupRefSources map[*ModelGroup]groupRefSource
	// unresolved attribute group references: maps from TypeDef to list of QNames
	attrGroupRefs map[*TypeDef][]QName
	// XSD 1.1 xs:schema/@defaultAttributes declarations. These are schema-level
	// QName references and must resolve even if no complex type applies them.
	schemaDefaultAttrRefs []schemaDefaultAttrRef
	// source info for type-level attribute group references, index-aligned with
	// attrGroupRefs. This covers explicit xs:attributeGroup ref children and
	// the implicit ref injected for XSD 1.1 schema defaultAttributes.
	attrGroupRefUseSources map[*TypeDef][]attrGroupRefUseSource
	// defaultAttrUses records attribute uses contributed by an implicit
	// xs:schema/@defaultAttributes application, keyed by complex type and
	// expanded attribute name. The value is the contributed attribute-use
	// component, so duplicate suppression can require the base and derived type
	// to have received the SAME default use, not merely a same-named use from
	// different default groups.
	defaultAttrUses map[*TypeDef]map[QName]*AttrUse
	// nested xs:attributeGroup ref children of a GLOBAL attribute group, keyed by
	// the containing group's QName. These are flattened (recursively, cycle-guarded)
	// before checkAttrGroupDuplicates so a duplicate attribute use introduced
	// through a referenced group — not just direct xs:attribute children — is
	// reported (ag-props-correct.2).
	attrGroupRefChildren map[QName][]QName
	// XSD 1.1: the xs:anyAttribute wildcard declared directly inside a GLOBAL
	// attribute group, keyed by the group's QName. A type referencing the group
	// intersects this wildcard (and those of nested group refs) into its
	// effective attribute wildcard. Populated only in Version11 so 1.0 behavior
	// (group wildcards dropped) is unchanged.
	attrGroupWildcards map[QName]*Wildcard
	// per-edge source info for the nested xs:attributeGroup ref children recorded in
	// attrGroupRefChildren, keyed by the containing group's QName and index-aligned
	// with the corresponding attrGroupRefChildren slice. Each entry records the
	// line/file of the back-edge <xs:attributeGroup ref="..."> element itself (not
	// the owning group's declaration), so an indirect-cycle diagnostic
	// (checkCircularAttrGroupRefs) cites the ref element that closed the cycle —
	// matching the direct-self-reference path — even when the cycle spans
	// included/redefined schemas that live in different files.
	attrGroupRefSources map[QName][]attrGroupSource
	// source info for named model group definitions (xs:group name="..."), keyed
	// by group QName. Used to run cos-element-consistent over standalone named
	// groups that no complex type references, reporting against the declaring file.
	groupSources map[QName]groupSource
	// source info for global attribute group definitions (xs:attributeGroup
	// name="..."), keyed by group QName. Used to run the duplicate-attribute-use
	// check (ag-props-correct.2) over attribute groups that no complex type
	// references — duplicates inside such a group are otherwise never inspected.
	attrGroupSources map[QName]attrGroupSource
	// source info for global elements, used in substitution group error messages
	globalElemSources map[*ElementDecl]elemRefSource
	// source info for type definitions, used in duplicate attribute errors
	typeDefSources map[*TypeDef]typeDefSource
	// nextTypeDefOrdinal is a monotonic counter assigned to each typeDefSource in
	// parse order. It is the final tie-breaker when ordering facet/notation
	// diagnostics so that multiple anonymous (empty-named) types on the SAME
	// source line have a deterministic error order independent of Go map
	// iteration.
	nextTypeDefOrdinal int
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
	// includeVisited records the resolved fs paths of schema documents already
	// pulled in via xs:include/xs:redefine on this compiler, so a transitive or
	// diamond chain loads each included document at most once. It is the cycle
	// guard for circular includes: a re-include of an already-loaded document is
	// skipped rather than re-parsed (which would re-register its declarations and
	// recurse forever).
	includeVisited map[string]struct{}
	// includeDepth and maxIncludeDepth bound xs:include/xs:redefine nesting as a
	// secondary safety net behind includeVisited. includeDepth is incremented
	// while processing a nested included schema's own includes and restored after.
	includeDepth    int
	maxIncludeDepth int
	// redefine is non-nil while processing the override children of an
	// xs:redefine. It scopes the duplicate-name suppression to the specific
	// (kind, name) components actually loaded by the redefined schema, each
	// consumable exactly once, instead of suppressing the check globally.
	redefine *redefineState
	// loadedRedefinable caches, per resolved schema-document path, the set of
	// redefinable component names a document contributed when first loaded via
	// xs:include or xs:redefine. XSD permits multiple xs:redefine elements
	// targeting the same document (e.g. redefining disjoint components, or a
	// no-op repeat); a later xs:redefine of an already-loaded document cannot
	// recompute its Phase-A delta (the components are already merged), so it
	// validates and consumes its overrides against this cached set instead.
	loadedRedefinable map[string]*redefinableSet
	// xpathDefaultNSSet records whether the current schema document carries a
	// schema-level xpathDefaultNamespace (XSD 1.1). When set, xs:alternative XPath
	// expressions that do not carry their own xpathDefaultNamespace inherit the
	// already-root-resolved schemaXPathDefaultNS value (shared with the
	// identity-constraint path). It is per-document (saved/restored across
	// include/redefine/import) and distinguishes "absent" from "present" so an
	// explicit empty value is still inherited as "no default element namespace".
	xpathDefaultNSSet bool
	// schemaBaseURI is the schema document's URI, exposed to xs:alternative /
	// xs:assert XPath expressions as fn:static-base-uri(). It is sourced from the
	// document URL (or the CompileFile path), NEVER from the diagnostic label, so a
	// caller-supplied error-message label cannot leak into XPath static-base-uri().
	schemaBaseURI string
	// ctaElems collects every element declaration carrying a {type table}
	// (xs:alternative children), so the alternative-type substitutability check
	// (cta-cvc / Type Alternative valid) can run after all types resolve.
	ctaElems []*ElementDecl
	// rootKey is the resolved fs key of the TOP-LEVEL schema document of this
	// compiler (the CompileFile root, or an import sub-compiler's own document). It
	// lets the xs:override cascade terminate a back-edge that points at the
	// overriding root WITHOUT re-loading/re-registering the root's components, and
	// distinguishes the seeded-root entry in includeVisited from a genuine plain
	// xs:include of a document (see override.go).
	rootKey string
	// overrideVisited records the (path + active-override-set fingerprint) keys of
	// schema documents pulled in by the xs:override TRANSFORMATION (override.go),
	// tracked SEPARATELY from includeVisited (plain xs:include/xs:redefine). Keying
	// by path AND active set (not path alone) is required: the SAME document reached
	// with a DIFFERENT active override set is a DISTINCT transformed document and
	// must be loaded again (letting duplicate-component checks fire on a real
	// collision); only a true diamond/cycle reached with the SAME active set
	// terminates here.
	overrideVisited map[string]struct{}
	// overridePaths records every resolved fs path ever pulled in by an
	// xs:override transformation (regardless of active set). It is the path-level
	// companion to overrideVisited used for the include+override CONFLICT check: a
	// document pulled in by BOTH a plain xs:include/xs:redefine AND an xs:override
	// is a fatal conflict (distinct constituents with colliding components), not a
	// silent no-op.
	overridePaths map[string]struct{}
	// notations records the QNames of every <xs:notation> declared in the schema
	// (and its included/imported documents). Used to verify that an xs:NOTATION
	// restriction's enumeration values name declared notations.
	notations map[QName]struct{}
}

// redefinableSet caches the redefinable component names a schema document
// contributed (keys, split by kind) plus the names already consumed by
// xs:redefine overrides across every xs:redefine of that document (consumed).
// The consumed set lets a second xs:redefine of the same document reject a
// component that an earlier xs:redefine already redefined as a duplicate, while
// still accepting redefinitions of disjoint components.
type redefinableSet struct {
	keys     map[redefineKind]map[QName]struct{}
	consumed map[redefineKind]map[QName]struct{}
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
	// consumed, when non-nil, is the cross-redefine consumption set shared with
	// the per-document redefinableSet cache. A component already redefined by an
	// EARLIER xs:redefine of the same document is rejected as a duplicate even
	// though this redefine's own seen map is empty; an accepted override is also
	// recorded here so a later xs:redefine of the document sees it.
	consumed map[redefineKind]map[QName]struct{}
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
	// A component already redefined by an earlier xs:redefine of the same
	// document is a duplicate redefinition even though THIS redefine has not
	// consumed it yet.
	if c.redefine.consumed != nil {
		if _, done := c.redefine.consumed[kind][qn]; done {
			return false
		}
	}
	if c.redefine.seen[kind] == nil {
		c.redefine.seen[kind] = make(map[QName]struct{})
	}
	c.redefine.seen[kind][qn] = struct{}{}
	if c.redefine.consumed != nil {
		if c.redefine.consumed[kind] == nil {
			c.redefine.consumed[kind] = make(map[QName]struct{})
		}
		c.redefine.consumed[kind][qn] = struct{}{}
	}
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

// defaultMaxIncludeDepth bounds xs:include/xs:redefine nesting depth. It is a
// secondary safety net behind the includeVisited loaded-set (which already
// guarantees termination by loading each included document at most once); the
// depth bound also caps pathological deep but acyclic chains. The same modest
// value as defaultMaxImportDepth leaves generous headroom for real schemas.
const defaultMaxIncludeDepth = 40

// elemRefSource tracks source location for error reporting.
type elemRefSource struct {
	elemName string
	line     int
	// source is the declaring file captured at collection time (c.diagSource()):
	// the included/imported schema when the element/ref was parsed inside an
	// xs:include/xs:import/xs:redefine, else empty (top-level). Diagnostics report
	// via c.diagSourceOrRecorded(source) so an unresolved ref in an imported
	// schema cites the imported file, not the importing one.
	source string
}

// groupRefSource tracks where an xs:group ref="..." particle appeared so the
// all-group reference constraints can be enforced after the ref resolves.
type groupRefSource struct {
	line  int
	local string // referencing element display name (e.g. "group")
	// nested is true when the group reference is contained inside another model
	// group (xs:sequence/xs:choice/xs:all) rather than being the sole top-level
	// particle of a complex type's content. A reference to an 'all' model group
	// is forbidden when nested.
	nested bool
	// maxOccursRaw is the lexical maxOccurs attribute on the referencing element
	// ("" if absent, which defaults to 1). A reference to an 'all' model group
	// must have maxOccurs == 1.
	maxOccursRaw string
	// source is the declaring file recorded at parse time (c.diagSource()): the
	// included/imported schema when the ref was parsed inside an
	// xs:include/xs:import/xs:redefine, else the compiler filename. checkAllGroupRef
	// runs after parsing (from resolveRefs / the redefine loop) when c.includeFile
	// has been restored, so the source must be captured here rather than recomputed.
	source string
}

// groupSource tracks where a named model group definition (xs:group name="...")
// appeared so cos-element-consistent diagnostics over standalone named groups
// cite the declaring file and line.
type groupSource struct {
	line   int
	source string // declaring file (this compiler's filename, or an imported file)
}

// attrGroupSource tracks where a global attribute group definition
// (xs:attributeGroup name="...") appeared so the duplicate-attribute-use check
// over unreferenced attribute groups cites the declaring file and line.
type attrGroupSource struct {
	line   int
	source string // declaring file (this compiler's filename, or an imported file)
}

// attrGroupRefUseSource tracks where a complex type referenced an attribute
// group, either explicitly through <xs:attributeGroup ref="..."> or implicitly
// through xs:schema/@defaultAttributes.
type attrGroupRefUseSource struct {
	line      int
	elemLocal string
	attr      string
	source    string
}

type schemaDefaultAttrRef struct {
	qn  QName
	src attrGroupRefUseSource
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
	// source is the declaring file when the attribute use was parsed from an
	// xs:include/xs:import/xs:redefine document (c.includeFile), empty for a
	// top-level declaration. Diagnostics cite this so the line matches the file.
	source string
}

// typeDefSource tracks source location and context for type definitions.
type typeDefSource struct {
	line    int
	isLocal bool // true for anonymous (local) complex types
	// source is the declaring file: this compiler's filename, or the file
	// pulled in by xs:include/xs:import/xs:redefine (c.includeFile) when the
	// type was parsed from an included/imported document. It is empty for
	// types declared directly in the top-level schema. Diagnostics cite this
	// so a type's line number matches the file it is reported against.
	source string
	// elemKind is the XSD element local name the type was parsed from
	// ("complexType" or "simpleType"), recorded at parse time so a diagnostic
	// (e.g. an unresolved base/itemType/memberTypes ref) reports the actual
	// element kind instead of a hard-coded "simpleType".
	elemKind string
	// ordinal is a stable parse-order sequence number, used as the final
	// tie-breaker when ordering diagnostics for types that share a source line
	// and have empty (anonymous) names. See recordTypeDefSource.
	ordinal int
}

// recordTypeDefSource registers the source location for td. The parse-order
// ordinal is assigned on FIRST registration and preserved across later
// overwrites (e.g. parseSimpleType records a type as local, then
// parseNamedSimpleType overwrites it as a named global) so the ordinal always
// reflects when the type was first seen.
func (c *compiler) recordTypeDefSource(td *TypeDef, line int, isLocal bool, elemKind string) {
	if existing, ok := c.typeDefSources[td]; ok {
		c.typeDefSources[td] = typeDefSource{line: line, isLocal: isLocal, source: existing.source, elemKind: elemKind, ordinal: existing.ordinal}
		return
	}
	c.typeDefSources[td] = typeDefSource{line: line, isLocal: isLocal, source: c.includeFile, elemKind: elemKind, ordinal: c.nextTypeDefOrdinal}
	c.nextTypeDefOrdinal++
}

// diagSource returns the file label a diagnostic should be attributed to during
// parsing. A component parsed while inside an xs:include/xs:redefine (or the
// Phase-A pass of a redefining schema) carries that file in c.includeFile; a
// top-level component has no includeFile, so it falls back to the compiler's own
// filename. Using c.filename directly would mis-attribute an included-file line
// number to the including schema. This is the parse-time counterpart to using a
// per-component recorded source (typeDefSource.source / attrGroupSource.source).
func (c *compiler) diagSource() string {
	if c.includeFile != "" {
		return c.includeFile
	}
	return c.filename
}

// diagSourceOrRecorded prefers a per-component recorded source (e.g.
// typeDefSource.source / attrGroupSource.source captured at parse time) and
// falls back to the live parse-time source (c.includeFile, then c.filename) when
// the recorded source is empty.
func (c *compiler) diagSourceOrRecorded(recorded string) string {
	if recorded != "" {
		return recorded
	}
	return c.diagSource()
}

// schemaError emits a fatal schema-compilation diagnostic and increments the
// compiler's error count. It collapses the repeated
// errorHandler.Handle(NewLeveledError(msg, ErrorLevelFatal)) + errorCount++
// pair used throughout the compiler.
func (c *compiler) schemaError(ctx context.Context, msg string) {
	c.errorHandler.Handle(ctx, helium.NewLeveledError(msg, helium.ErrorLevelFatal))
	c.errorCount++
}

// parse parses a nested schema document (xs:include/xs:import/xs:redefine)
// using the injected parser policy when one was configured on the Compiler, or
// the default schema parser otherwise.
func (c *compiler) parse(ctx context.Context, data []byte) (*helium.Document, error) {
	p := defaultSchemaParser()
	if c.parser != nil {
		p = *c.parser
	}
	return p.Parse(ctx, data)
}

func defaultSchemaParser() helium.Parser {
	return helium.NewParser().SubstituteEntities(true)
}

func compileSchema(ctx context.Context, doc *helium.Document, baseDir string, cfg *compileConfig) (*Schema, error) {
	root := findDocumentElement(doc)
	if root == nil {
		return nil, fmt.Errorf("xsd: empty document")
	}

	if !isXSDElement(root, elemSchema) {
		return nil, fmt.Errorf("xsd: root element is not xs:schema")
	}

	// Default to a deny-all FS so an untrusted schema cannot reach the host
	// filesystem via xs:include/xs:import/xs:redefine (local-file disclosure /
	// resource exhaustion). Callers opt into host access explicitly via
	// [Compiler.FS] (e.g. helium.PermissiveFS or a confined fs.FS). This mirrors
	// the secure-by-default flip applied to helium.NewParser.
	fsys := fs.FS(iofs.DenyAll{})
	if cfg != nil && cfg.fsys != nil {
		fsys = cfg.fsys
	}
	var parser *helium.Parser
	if cfg != nil {
		parser = cfg.parser
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
		parser:                   parser,
		typeRefs:                 make(map[*TypeDef]QName),
		elemRefs:                 make(map[*ElementDecl]QName),
		elemRefSources:           make(map[*ElementDecl]elemRefSource),
		groupRefs:                make(map[*ModelGroup]QName),
		groupRefSources:          make(map[*ModelGroup]groupRefSource),
		groupSources:             make(map[QName]groupSource),
		attrGroupSources:         make(map[QName]attrGroupSource),
		attrGroupWildcards:       make(map[QName]*Wildcard),
		attrGroupRefs:            make(map[*TypeDef][]QName),
		attrGroupRefUseSources:   make(map[*TypeDef][]attrGroupRefUseSource),
		defaultAttrUses:          make(map[*TypeDef]map[QName]*AttrUse),
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
		importedNS:               make(map[string]string),
		maxImportDepth:           defaultMaxImportDepth,
		includeVisited:           make(map[string]struct{}),
		maxIncludeDepth:          defaultMaxIncludeDepth,
		loadedRedefinable:        make(map[string]*redefinableSet),
		notations:                make(map[QName]struct{}),
	}
	c.errorHandler = helium.NilErrorHandler{}
	if cfg != nil {
		// Seed the circular-include guard with the root schema's resolved key (set
		// by CompileFile) so a cycle pointing back at the top-level schema
		// (main -> inc -> main) treats the root as already-loaded instead of
		// re-parsing it and emitting spurious duplicate-component errors.
		if cfg.rootKey != "" {
			c.includeVisited[cfg.rootKey] = struct{}{}
			c.rootKey = cfg.rootKey
		}
		c.filename = cfg.label
		if c.filename == "" {
			c.filename = doc.URL()
		}
		if c.filename == "" {
			c.filename = "(string)"
		}
		// XPath fn:static-base-uri() source: the document URL (set by the caller),
		// else the CompileFile path. NEVER the diagnostic label.
		c.schemaBaseURI = doc.URL()
		if c.schemaBaseURI == "" {
			c.schemaBaseURI = cfg.schemaURI
		}
		if cfg.errorHandler != nil {
			c.errorHandler = cfg.errorHandler
		}
	}

	// Resolve the effective XSD version: an explicit Compiler.Version() wins;
	// otherwise a vc:minVersion="1.1" hint on the root <xs:schema> upgrades to
	// 1.1. The resolved version is frozen onto the Schema so the Validator
	// applies the same version-specific semantics without a separate knob.
	c.version = resolveVersion(cfg, root)
	c.schema.version = c.version

	c.schema.targetNamespace = getAttr(root, attrTargetNamespace)
	c.schemaTargetNSSet = c.schema.targetNamespace != ""
	if hasAttr(root, attrXPathDefaultNamespace) {
		c.xpathDefaultNSSet = true
	}
	c.schema.elemFormQualified = getAttr(root, attrElementFormDefault) == attrValQualified
	c.schema.attrFormQualified = getAttr(root, attrAttributeFormDefault) == attrValQualified

	// Register built-in types. Done BEFORE conditional inclusion so the pre-pass
	// can test type/facet availability against the active version's registry.
	registerBuiltinTypes(c.schema, c.version)

	// Conditional inclusion UNLINKS vc:-excluded elements from the schema tree.
	// The top-level `doc` is CALLER-OWNED, so pruning it in place would make
	// Compile side-effecting and non-idempotent (e.g. compiling the same parsed
	// document under Version10 then Version11 would no longer see the 1.1 branch).
	// Defend the caller's DOM by compiling against a deep copy — but ONLY when a vc
	// directive is actually present, so the overwhelmingly common vc-free schema
	// keeps the fast no-copy path (no perf regression). The clone preserves
	// doc.URL() so relative include/import/redefine schemaLocation resolution is
	// unchanged. (Nested include/import/redefine documents are parsed fresh on
	// every Compile, so the pre-pass may prune those in place.)
	if documentHasVCDirective(root) {
		clone, cerr := helium.CopyDoc(doc)
		if cerr != nil {
			return nil, fmt.Errorf("xsd: failed to copy schema document for conditional inclusion: %w", cerr)
		}
		clone.SetURL(doc.URL())
		root = findDocumentElement(clone)
		if root == nil || !isXSDElement(root, elemSchema) {
			return nil, fmt.Errorf("xsd: empty document")
		}
	}

	// Conditional inclusion (XSD 1.1 version-control namespace): prune any
	// elements excluded by their vc: attributes for the active version BEFORE the
	// tree is interpreted, so a removed element (e.g. a 1.1-only xs:assert under a
	// 1.0 processor, or a 1.0 fallback under 1.1) is never compiled. If the ROOT
	// <xs:schema> is itself vc-excluded the whole document contributes nothing:
	// return an empty (valid) schema WITHOUT interpreting or validating its other
	// (non-preserved) root attributes — an excluded root must not fail compilation
	// on, say, a bogus blockDefault it would never use. A MALFORMED 1.1 vc value on
	// the excluded root is still a schema error (reported during the pre-pass), so
	// surface it instead of swallowing it behind the empty-schema short-circuit.
	if c.applyConditionalInclusion(ctx, root) {
		if c.errorCount > 0 {
			return nil, ErrCompilationFailed
		}
		return c.schema, nil
	}

	// XSD 1.1 schema-level default element namespace for identity-constraint
	// XPaths, RESOLVED against the (post-conditional-inclusion) root's context now
	// (so an inherited ##defaultNamespace uses the root's default namespace, not a
	// selector/field's).
	if c.version == Version11 {
		c.schemaXPathDefaultNS = resolveXPathDefaultNSToken(root, getAttr(root, attrXPathDefaultNS), c.schema.targetNamespace)
		c.readSchemaDefaultAttributes(ctx, root)
	}

	// Parse blockDefault attribute.
	if v := getAttr(root, attrBlockDefault); v != "" {
		if !isValidBlock(v) && c.filename != "" {
			c.schemaError(ctx, schemaParserErrorAttr(c.filename, root.Line(), root.LocalName(), elemSchema, attrBlockDefault,
				"The value '"+v+"' is not valid. Expected is '(#all | List of (extension | restriction | substitution))'."))
		} else {
			c.schema.blockDefault = parseBlockFlags(v)
		}
	}

	// Parse finalDefault attribute.
	if v := getAttr(root, attrFinalDefault); v != "" {
		if !isValidFinalDefault(v) && c.filename != "" {
			c.schemaError(ctx, schemaParserErrorAttr(c.filename, root.Line(), root.LocalName(), elemSchema, attrFinalDefault,
				"The value '"+v+"' is not valid. Expected is '(#all | List of (extension | restriction | list | union))'."))
		} else {
			c.schema.finalDefault = parseFinalFlags(v)
		}
	}

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

	// Build the substitution-group membership map BEFORE resolving refs, because
	// resolveRefs runs the UPA (cos-nonambig) check, which expands a content
	// model's substitution-group head leaves to their members. A global element's
	// substitutionGroup affiliation is fixed at read/include time (it is a
	// top-level-only attribute resolved by resolveQName), so the map is fully
	// determined here; this only pre-populates the membership map (and sorts it)
	// and emits no diagnostics, leaving every error-reporting check (circularity,
	// final, element-consistency) in its original order below.
	c.buildSubstGroups()

	// Second pass: resolve type references. (XSD 1.1 ##definedSibling resolution
	// runs INSIDE resolveRefs, after group-ref expansion but before the
	// restriction-derivation checks, so those checks see resolved SiblingNames.)
	c.resolveRefs(ctx)

	// Reject circular simple-type definitions (a union/list/restriction that
	// reaches itself) BEFORE any check that walks the variety/base chain. A
	// cyclic type is fatally broken, and the downstream facet, enumeration, and
	// NOTATION checks repeatedly walk BaseType/ItemType/MemberTypes; several of
	// those walks are not cycle-guarded, so a cyclic type must be removed from
	// consideration up front. When a cycle is found the schema cannot be
	// meaningfully compiled further, so report it and stop here.
	if c.checkCircularSimpleTypes(ctx) {
		return nil, ErrCompilationFailed
	}

	// Check facet consistency after refs are resolved (base types are available).
	c.checkFacetConsistency(ctx)

	// Validate QName/NOTATION enumeration literal prefixes (and the NOTATION
	// no-enumeration rule) now that base types are resolved.
	c.checkEnumQNameAndNotation(ctx)

	// Reject element/attribute declarations whose effective type is the built-in
	// xs:NOTATION (or NOTATION-derived) without an effective enumeration facet.
	c.checkNotationOnDeclarations(ctx)

	// XSD 1.1: xs:anyAtomicType is abstract and must not be used as the base type
	// of a user-defined simple type, nor as a list item type or union member type.
	if c.version == Version11 {
		c.checkAnyAtomicTypeUsage(ctx)
	}

	// XSD 1.1: each conditional-type-assignment alternative's type must be validly
	// substitutable for the element's declared type. Runs after type refs resolve.
	c.checkAltSubstitutability(ctx)

	// Detect circular substitution-group references. The membership map itself was
	// already built by buildSubstGroups (before resolveRefs) so UPA could see it;
	// this only reports the circularity diagnostic, keeping its position in the
	// error-ordering sequence unchanged.
	if c.filename != "" {
		for _, edecl := range c.schema.elements {
			if edecl.SubstitutionGroup == (QName{}) {
				continue
			}
			c.checkCircularSubstGroup(ctx, edecl)
		}
	}

	// Enforce final on type derivations.
	if c.filename != "" && c.errorCount == 0 {
		c.checkFinalOnTypes(ctx)
		c.checkFinalOnSubstGroups(ctx)
	}

	// Enforce cos-element-consistent (Element Declarations Consistent): two
	// element declarations with the same expanded name in one effective content
	// model must have the same type definition. Run AFTER substitution groups are
	// built (it consults schema.substGroups to fold in a head's implicitly-
	// containable members) and gated on errorCount==0 like UPA, since the content
	// models must be fully resolved and structurally sound.
	c.checkElementConsistent(ctx)

	// Identity-constraints (xs:key/xs:unique/xs:keyref) share one symbol space and
	// must be unique by {targetNamespace}name across the whole schema. Reject any
	// repeated name before the keyref resolution below, which would otherwise
	// silently collapse the colliding constraints into one registry entry.
	c.checkDuplicateIDCs(ctx)

	// Resolve XSD 1.1 identity-constraint @ref references (copy the referenced
	// constraint's selector/fields/refer into the referencing constraint) before
	// the keyref/@refer check, since a ref'd keyref inherits its target's @refer.
	c.resolveConstraintRefs(ctx)

	// Resolve every xs:keyref/@refer against the schema-wide registry of
	// key/unique constraints. An unresolved refer must be a fatal schema error:
	// otherwise the keyref is silently skipped at validation time and referential
	// integrity is never enforced.
	c.checkKeyRefRefers(ctx)

	// Fatal schema diagnostics were already delivered to the ErrorHandler as
	// they were discovered. A non-zero error count means the schema is
	// invalid, so the compiled Schema must not be handed back: returning it
	// would let callers validate against a malformed schema. Surface the
	// failure as ErrCompilationFailed (mirrors Validate's ErrValidationFailed).
	if c.errorCount > 0 {
		return nil, ErrCompilationFailed
	}

	return c.schema, nil
}

func (c *compiler) readSchemaDefaultAttributes(ctx context.Context, root *helium.Element) {
	c.schema.defaultAttributes = QName{}
	c.schema.defaultAttrsSet = false
	c.schema.defaultAttrsSrc = attrGroupRefUseSource{}
	if c.version != Version11 || !hasAttr(root, attrDefaultAttributes) {
		return
	}
	ref := normalizeWhiteSpace(getAttr(root, attrDefaultAttributes), "collapse")
	src := attrGroupRefUseSource{
		line:      root.Line(),
		elemLocal: root.LocalName(),
		attr:      attrDefaultAttributes,
		source:    c.diagSource(),
	}
	if !xmlchar.IsValidQName(ref) {
		c.reportInvalidQNameValue(ctx, root, ref)
		return
	}
	qn := QName{Local: ref, NS: c.schema.targetNamespace}
	if prefix, local, ok := strings.Cut(ref, ":"); ok {
		ns := lookupNS(root, prefix)
		if ns == "" && prefix != "" {
			c.reportUnboundQNamePrefix(ctx, root, ref, prefix)
			return
		}
		if c.rejectDeprecatedDatatypeNamespace(ctx, root, ref, ns) {
			return
		}
		qn = QName{Local: local, NS: ns}
	} else {
		if defNS := lookupNS(root, ""); defNS != "" {
			qn.NS = defNS
		}
		if c.rejectDeprecatedDatatypeNamespace(ctx, root, ref, qn.NS) {
			return
		}
	}
	c.schema.defaultAttributes = qn
	c.schema.defaultAttrsSet = true
	c.schema.defaultAttrsSrc = src
	c.schemaDefaultAttrRefs = append(c.schemaDefaultAttrRefs, schemaDefaultAttrRef{qn: qn, src: src})
}

// buildSubstGroups populates c.schema.substGroups, mapping each substitution-group
// head QName to the global element declarations affiliated with it, sorted by
// local name for deterministic downstream output. It is intentionally
// side-effect-free with respect to diagnostics: it neither reports errors nor
// reads resolved type information, so it can run early (before resolveRefs) to let
// the UPA check expand substitution-group heads to their members. The companion
// circularity, final, and element-consistency checks run later, unchanged.
func (c *compiler) buildSubstGroups() {
	for _, edecl := range c.schema.elements {
		if edecl.SubstitutionGroup == (QName{}) {
			continue
		}
		head := edecl.SubstitutionGroup
		c.schema.substGroups[head] = append(c.schema.substGroups[head], edecl)
	}
	for _, members := range c.schema.substGroups {
		sort.Slice(members, func(i, j int) bool {
			return members[i].Name.Local < members[j].Name.Local
		})
	}
}

// checkKeyRefRefers verifies that every xs:keyref/@refer names a key or unique
// identity-constraint that exists somewhere in the schema. The XSD scope rules
// place all identity-constraint names in a single symbol space, so a keyref may
// refer to a key/unique declared on a different element; the prior per-host
// resolution missed those and any typo'd or missing refer silently disabled the
// keyref. A failure here is reported as a fatal schema parser error.
//
// The registry is built from ALL identity-constraints in the schema, not just
// those on GLOBAL element declarations: a keyref (or the key/unique it refers
// to) declared on a LOCAL element declaration buried in a content model must be
// checked too. collectAllIDCs walks every global element/type/group content
// model recursively (with a visited set) to reach those local hosts.
func (c *compiler) checkKeyRefRefers(ctx context.Context) {
	if c.filename == "" {
		return
	}

	idcs := c.collectAllIDCs()

	// Build the set of all declared key/unique constraint QNames. Identity
	// constraints live in the schema's target namespace, so a keyref may refer to
	// a key/unique declared on a DIFFERENT element; matching must be by full
	// {namespace}local identity, not local name only (a local-name match could
	// bind the wrong constraint when two namespaces share a local name).
	keyNames := make(map[QName]struct{})
	for _, idc := range idcs {
		if idc.IsConstraintRef {
			continue
		}
		if idc.Kind == IDCKey || idc.Kind == IDCUnique {
			keyNames[idc.QName] = struct{}{}
		}
	}

	for _, idc := range idcs {
		if idc.Kind != IDCKeyRef {
			continue
		}
		// A @ref keyref whose reference failed to resolve was already reported by
		// resolveConstraintRefs; its Refer was never copied, so skip it here to
		// avoid a spurious "no refer" diagnostic.
		if idc.IsConstraintRef && idc.Refer == "" {
			continue
		}
		// An unbound @refer prefix was already reported as fatal at parse time.
		if idc.referUnbound {
			continue
		}
		// Report against the constraint's declaring file (idc.Source), paired with
		// idc.Line: an IMPORTED keyref's deferred @refer error must cite the
		// imported schema (where the line number is meaningful), not this
		// (top-level) compiler's filename.
		source := idc.Source
		if source == "" {
			source = c.filename
		}
		if idc.Refer == "" {
			msg := fmt.Sprintf("The keyref identity-constraint '%s' has no 'refer' attribute naming a key or unique.", idc.Name)
			c.schemaError(ctx,
				schemaParserErrorAttr(source, idc.Line, elemKeyRef, elemKeyRef, attrRefer, msg))
			continue
		}
		if _, ok := keyNames[idc.ReferQName]; ok {
			continue
		}
		msg := fmt.Sprintf("The keyref identity-constraint '%s' references the unknown key or unique '%s'.", idc.Name, idc.Refer)
		c.schemaError(ctx,
			schemaParserErrorAttr(source, idc.Line, elemKeyRef, elemKeyRef, attrRefer, msg))
	}
}

// resolveConstraintRefs resolves XSD 1.1 identity-constraint @ref references. A
// constraint with @ref reuses a referenced constraint's selector/fields (and,
// for a keyref, its refer); they are copied in so validation treats the
// referencing constraint like any other. Two schema errors are enforced: the
// reference must resolve to an existing identity-constraint (cf. id042), and the
// referencing element's kind must match the referenced constraint's kind — a
// key/unique/keyref may only ref a constraint of the same kind (id041).
func (c *compiler) resolveConstraintRefs(ctx context.Context) {
	if c.filename == "" {
		return
	}
	idcs := c.collectAllIDCs()

	byName := make(map[QName]*IDConstraint)
	for _, idc := range idcs {
		if idc.IsConstraintRef || idc.Name == "" {
			continue
		}
		byName[idc.QName] = idc
	}

	for _, idc := range idcs {
		if !idc.IsConstraintRef {
			continue
		}
		// An unbound @ref prefix was already reported as fatal at parse time; the
		// resolved QName is meaningless, so skip the "unknown constraint" check to
		// avoid a duplicate diagnostic.
		if idc.constraintRefUnbound {
			continue
		}
		source := idc.Source
		if source == "" {
			source = c.filename
		}
		xsdElem := idcKindName(idc.Kind)
		target, ok := byName[idc.ConstraintRefQName]
		if !ok {
			msg := fmt.Sprintf("The identity-constraint '%s' references the unknown identity-constraint '%s'.", xsdElem, idc.ConstraintRef)
			c.schemaError(ctx, schemaParserErrorAttr(source, idc.Line, xsdElem, xsdElem, attrRef, msg))
			continue
		}
		if target.Kind != idc.Kind {
			msg := fmt.Sprintf("The identity-constraint reference '%s' points to a %s constraint, but a %s constraint was expected.", idc.ConstraintRef, idcKindName(target.Kind), xsdElem)
			c.schemaError(ctx, schemaParserErrorAttr(source, idc.Line, xsdElem, xsdElem, attrRef, msg))
			continue
		}

		// Copy the referenced constraint's evaluation state. The referencing
		// constraint keeps its own host/line/source; only the selector/field
		// machinery (and the keyref refer) is inherited. It also adopts the
		// referenced constraint's QName identity so a key/unique applied via @ref
		// is registered in the per-occurrence key table under the same name a
		// keyref refers to (id044: a ref'd keyref resolves against a ref'd key on
		// the same host).
		idc.QName = target.QName
		idc.Name = target.Name
		idc.Selector = target.Selector
		idc.SelectorExpr = target.SelectorExpr
		idc.SelectorDefaultNS = target.SelectorDefaultNS
		idc.Fields = target.Fields
		idc.FieldExprs = target.FieldExprs
		idc.FieldDefaultNS = target.FieldDefaultNS
		idc.Namespaces = target.Namespaces
		if idc.Kind == IDCKeyRef {
			idc.Refer = target.Refer
			idc.ReferQName = target.ReferQName
			idc.referUnbound = target.referUnbound
		}
	}
}

// checkDuplicateIDCs enforces the XSD constraint that identity-constraint
// definitions (xs:key, xs:unique, xs:keyref) all share a single symbol space:
// each {targetNamespace}name must be unique across the entire schema, regardless
// of which element declaration hosts the constraint. Two constraints sharing one
// name silently collapsed into a single registry entry before — corrupting
// keyref resolution and IDC validation — so a collision is now a fatal schema
// parser error, matching reportDuplicateComponent's style and wording.
func (c *compiler) checkDuplicateIDCs(ctx context.Context) {
	if c.filename == "" {
		return
	}

	idcs := c.collectAllIDCs()

	// Sort by source location so the reported collision is deterministic (the
	// later declaration is flagged) regardless of the map-iteration order in
	// collectAllIDCs.
	sort.SliceStable(idcs, func(i, j int) bool {
		if idcs[i].Source != idcs[j].Source {
			return idcs[i].Source < idcs[j].Source
		}
		return idcs[i].Line < idcs[j].Line
	})

	seen := make(map[QName]struct{}, len(idcs))
	for _, idc := range idcs {
		// A @ref constraint declares no name of its own, so it does not occupy the
		// identity-constraint symbol space and cannot be a duplicate.
		if idc.IsConstraintRef {
			continue
		}
		if _, dup := seen[idc.QName]; !dup {
			seen[idc.QName] = struct{}{}
			continue
		}
		c.reportDuplicateIDC(ctx, idc)
	}
}

// reportDuplicateIDC emits the fatal schema-parser error for a redeclared
// identity-constraint, mirroring reportDuplicateComponent. The XSD element name
// (key/unique/keyref) is derived from the constraint kind and the constraint's
// own declaring file/line is used so an imported or included collision cites the
// declaring document.
func (c *compiler) reportDuplicateIDC(ctx context.Context, idc *IDConstraint) {
	xsdElem := elemKey
	switch idc.Kind {
	case IDCUnique:
		xsdElem = elemUnique
	case IDCKeyRef:
		xsdElem = elemKeyRef
	}

	qnDisplay := "'" + idc.QName.NS + "'" + idc.QName.Local
	if idc.QName.NS != "" {
		qnDisplay = "'{" + idc.QName.NS + "}" + idc.QName.Local + "'"
	}

	source := idc.Source
	if source == "" {
		source = c.filename
	}

	c.schemaError(ctx, schemaParserError(source, idc.Line, xsdElem, xsdElem,
		"An identity-constraint definition "+qnDisplay+" does already exist."))
}

// collectAllIDCs returns every identity-constraint declared anywhere in the
// schema, including those on LOCAL element declarations nested inside content
// models. It seeds the walk from all global element declarations, named types,
// and named model groups, then descends each content model recursively. Visited
// sets on *ElementDecl, *ModelGroup, and *TypeDef bound the walk so shared
// group particles, recursive types, and circular references are each traversed
// once.
func (c *compiler) collectAllIDCs() []*IDConstraint {
	w := &idcWalker{
		elems:  make(map[*ElementDecl]struct{}),
		groups: make(map[*ModelGroup]struct{}),
		types:  make(map[*TypeDef]struct{}),
	}
	for _, edecl := range c.schema.elements {
		w.walkElement(edecl)
	}
	for _, td := range c.schema.types {
		w.walkType(td)
	}
	for _, mg := range c.schema.groups {
		w.walkGroup(mg)
	}
	return w.idcs
}

// idcWalker accumulates identity-constraints while recursively descending
// element declarations, types, and model groups, tracking visited nodes to
// terminate on shared/recursive/circular structures.
type idcWalker struct {
	idcs   []*IDConstraint
	elems  map[*ElementDecl]struct{}
	groups map[*ModelGroup]struct{}
	types  map[*TypeDef]struct{}
}

func (w *idcWalker) walkElement(edecl *ElementDecl) {
	if edecl == nil {
		return
	}
	if _, seen := w.elems[edecl]; seen {
		return
	}
	w.elems[edecl] = struct{}{}
	w.idcs = append(w.idcs, edecl.IDCs...)
	w.walkType(edecl.Type)
}

func (w *idcWalker) walkType(td *TypeDef) {
	if td == nil {
		return
	}
	if _, seen := w.types[td]; seen {
		return
	}
	w.types[td] = struct{}{}
	w.walkGroup(td.ContentModel)
	w.walkType(td.BaseType)
}

func (w *idcWalker) walkGroup(mg *ModelGroup) {
	if mg == nil {
		return
	}
	if _, seen := w.groups[mg]; seen {
		return
	}
	w.groups[mg] = struct{}{}
	for _, p := range mg.Particles {
		switch term := p.Term.(type) {
		case *ElementDecl:
			w.walkElement(term)
		case *ModelGroup:
			w.walkGroup(term)
		}
	}
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
		case isXSDElement(elem, elemNotation):
			if name := getAttr(elem, attrName); name != "" {
				c.notations[QName{Local: name, NS: c.schema.targetNamespace}] = struct{}{}
			}
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

// getAttrNS reads an attribute by namespace URI and local name, returning "" if
// absent.
func getAttrNS(elem *helium.Element, ns, name string) string {
	attr, ok := elem.FindAttribute(helium.NSPredicate{Local: name, NamespaceURI: ns})
	if !ok {
		return ""
	}
	return attr.Value()
}

// resolveVersion determines the effective XSD specification version for a
// compilation. An explicit Compiler.Version() (cfg.versionSet) always wins.
// Otherwise a vc:minVersion="1.1" (or higher) hint on the root <xs:schema>
// upgrades to 1.1; absent any hint, the default is Version10. vc:maxVersion is
// not consulted for processor selection — it gates per-element conditional
// inclusion, not the overall version.
//
// The hint is parsed with the SAME rules the conditional-inclusion pre-pass uses
// for vc decimals: ASCII XML whitespace trim only (NOT strings.TrimSpace, which
// also strips NBSP) and EXACT xs:decimal comparison (isValidXSDDecimal +
// value.CompareDecimal, not float64). So an NBSP-padded value or a malformed
// decimal does NOT auto-select 1.1 (treated as no hint → default 1.0), and a
// high-precision value just below 1.1 is not float-rounded up into 1.1.
func resolveVersion(cfg *compileConfig, root *helium.Element) Version {
	if cfg != nil && cfg.versionSet {
		return cfg.version
	}
	if v := getAttrNS(root, lexicon.NamespaceXSDVersioning, "minVersion"); v != "" {
		s := strings.Trim(v, " \t\r\n")
		if isValidXSDDecimal(s) && value.CompareDecimal(s, "1.1") >= 0 {
			return Version11
		}
	}
	return Version10
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
	// The "xml" prefix is predeclared (bound to the XML namespace) and never
	// needs an explicit xmlns:xml declaration, so it is always in scope. Without
	// this, a ref="xml:lang" would resolve to no namespace and could spuriously
	// collide with an unrelated unprefixed "lang" attribute use.
	if prefix == "xml" {
		return lexicon.NamespaceXML
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

// builtinTypeNames is the immutable list of XSD 1.0 built-in datatype local names
// (in the XSD namespace). It is the SINGLE SOURCE for both registration
// (registerBuiltinTypes) and processor-capability detection (builtinTypeAvailable
// / vc:typeAvailable). Capability must consult this fixed set, NOT c.schema.types,
// because that map can already hold user declarations from the including schema
// (shared across nested include/redefine) — a 1.0 schema that defines a type
// literally named {XSD}error must NOT make vc:typeAvailable="xs:error" true.
var builtinTypeNames = []string{
	"string", "boolean", lexicon.TypeDecimal, lexicon.TypeFloat, lexicon.TypeDouble,
	lexicon.TypeInteger, lexicon.TypeNonPositiveInteger, lexicon.TypeNegativeInteger,
	lexicon.TypeLong, lexicon.TypeInt, lexicon.TypeShort, lexicon.TypeByte,
	lexicon.TypeNonNegativeInteger, lexicon.TypeUnsignedLong, lexicon.TypeUnsignedInt, lexicon.TypeUnsignedShort, lexicon.TypeUnsignedByte,
	lexicon.TypePositiveInteger,
	lexicon.TypeNormalizedString, "token", "language", "Name", "NCName",
	"ID", "IDREF", "IDREFS", "ENTITY", "ENTITIES", "NMTOKEN", "NMTOKENS",
	"date", "dateTime", "time", "duration",
	"gYearMonth", "gYear", "gMonthDay", "gDay", "gMonth",
	"hexBinary", "base64Binary",
	"anyURI", lexicon.TypeQName, lexicon.TypeNotation,
	typeAnyType, "anySimpleType",
}

// builtinType11Bases maps each XSD 1.1-only built-in datatype local name to its
// primitive base local name. SINGLE SOURCE for both registration
// (registerBuiltinTypes11, which links BaseType) and the 1.1 capability set
// (builtinTypeAvailable). Keeping the names here (not duplicated in a separate
// list) prevents drift between what is registered and what is reported available.
var builtinType11Bases = map[string]string{
	lexicon.TypeDateTimeStamp:     lexicon.TypeDateTime,
	lexicon.TypeDayTimeDuration:   lexicon.TypeDuration,
	lexicon.TypeYearMonthDuration: lexicon.TypeDuration,
	lexicon.TypeAnyAtomicType:     "anySimpleType",
	lexicon.TypeError:             "anySimpleType",
}

// builtinTypeSet10 is the precomputed lookup set of the 1.0 built-in names.
var builtinTypeSet10 = newStringSet(builtinTypeNames)

func newStringSet(names []string) map[string]struct{} {
	m := make(map[string]struct{}, len(names))
	for _, n := range names {
		m[n] = struct{}{}
	}
	return m
}

// builtinTypeAvailable reports whether the XSD-namespace type local name is a
// built-in KNOWN TO THE PROCESSOR for the active version: the 1.0 built-ins in
// every version, plus the 1.1-only types in Version11. This is a fixed-capability
// check, independent of any user/included schema declarations.
func builtinTypeAvailable(local string, version Version) bool {
	if _, ok := builtinTypeSet10[local]; ok {
		return true
	}
	if version != Version11 {
		return false
	}
	_, ok := builtinType11Bases[local]
	return ok
}

func registerBuiltinTypes(s *Schema, version Version) {
	for _, name := range builtinTypeNames {
		qn := QName{Local: name, NS: lexicon.NamespaceXSD}
		td := &TypeDef{
			Name:        qn,
			ContentType: ContentTypeSimple,
		}
		if name == typeAnyType {
			td.IsComplex = true
			td.ContentType = ContentTypeMixed
			td.AnyAttribute = &Wildcard{
				Namespace:       WildcardNSAny,
				ProcessContents: ProcessLax,
			}
		}
		s.types[qn] = td
	}
	if version == Version11 {
		registerBuiltinTypes11(s)
	}
}

// registerBuiltinTypes11 registers the XSD 1.1-only built-in datatypes. They are
// added only in 1.1 mode so a 1.0 schema referencing e.g. xs:dateTimeStamp still
// fails to resolve the type. BaseType links each new type to its primitive base
// so derivation checks (isDerivedFrom, xsi:type) treat it as a subtype.
func registerBuiltinTypes11(s *Schema) {
	builtin := func(local string) *TypeDef {
		return s.types[QName{Local: local, NS: lexicon.NamespaceXSD}]
	}
	for local, baseLocal := range builtinType11Bases {
		qn := QName{Local: local, NS: lexicon.NamespaceXSD}
		// The XSD 1.1 built-in datatypes here are all derived from their base by
		// RESTRICTION. Recording Derivation (these built-ins ARE BaseType-linked, so
		// the pointer walk reaches the base) lets isDerivationBlocked enforce
		// block="restriction"/"#all" on e.g. xsi:type="xs:dateTimeStamp" over a
		// declared xs:dateTime.
		s.types[qn] = &TypeDef{Name: qn, ContentType: ContentTypeSimple, BaseType: builtin(baseLocal), Derivation: DerivationRestriction}
	}
}
