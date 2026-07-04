package xsd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/iofs"
)

// ErrValidationFailed is returned by Validate when the document does not
// conform to the schema. Individual validation errors are delivered to the
// ErrorHandler configured on the Validator.
var ErrValidationFailed = errors.New("xsd: validation failed")

// ErrCompilationFailed is returned by Compile and CompileFile when the schema
// contains one or more fatal errors. The returned schema is nil; the
// individual diagnostics are delivered to the ErrorHandler configured on the
// Compiler.
var ErrCompilationFailed = errors.New("xsd: schema compilation failed")

// ErrNilSchema is returned by Validate when the Validator has no compiled
// schema (for example, after NewValidator(nil)).
var ErrNilSchema = errors.New("xsd: nil schema")

// ErrNilDocument is returned by Validate when the document to validate is nil.
var ErrNilDocument = errors.New("xsd: nil document")

// Version selects the XML Schema specification version a [Compiler] targets.
// The zero value is [Version10], so a compiler from [NewCompiler] behaves as a
// strict XSD 1.0 processor unless [Compiler.Version] (or a vc: hint on the
// schema) opts into 1.1.
type Version int

const (
	// Version10 targets XML Schema 1.0. This is the default and matches the
	// historical behavior of this package (a libxml2-style XSD 1.0 processor).
	Version10 Version = iota
	// Version11 targets XML Schema 1.1. It enables the 1.1-only lexical rules
	// (e.g. "+INF" for xs:double/xs:float), the 1.1 built-in datatypes
	// (xs:dateTimeStamp, xs:dayTimeDuration, xs:yearMonthDuration,
	// xs:anyAtomicType, xs:error), and — as later phases land — the 1.1
	// structural features (assertions, conditional type assignment, open
	// content, relaxed wildcards/UPA/all, xs:override).
	Version11
)

// String returns "1.0" or "1.1".
func (v Version) String() string {
	if v == Version11 {
		return "1.1"
	}
	return "1.0"
}

type compileConfig struct {
	label   string // label for error messages (e.g. source filename)
	baseDir string // base directory for resolving relative includes
	// version is the XSD specification version the compiler targets. The zero
	// value (Version10) is the default. When versionSet is false, a vc:minVersion
	// hint on the root <xs:schema> may upgrade the effective version to 1.1; an
	// explicit Version() call sets versionSet and always wins.
	version    Version
	versionSet bool
	// defaultVersion is the fallback version used when the caller has NOT forced
	// a version via Version() and the schema declares no vc:minVersion hint. When
	// defaultVersionSet is false the fallback is Version10 (the standalone
	// default). A forced Version() and a vc:minVersion hint both take precedence
	// over it, so this only chooses between 1.0 and 1.1 for a schema silent on
	// version.
	defaultVersion    Version
	defaultVersionSet bool
	// rootKey is the resolved fs.FS key of the TOP-LEVEL schema document, when
	// known (set by CompileFile). compileSchema seeds includeVisited with it so a
	// circular include/redefine that points back at the root (main -> inc -> main)
	// treats the root as already-loaded instead of re-parsing it and emitting
	// spurious duplicate-component errors.
	rootKey string
	// schemaURI is the schema document's URI/path when known (set by CompileFile),
	// used as the XPath fn:static-base-uri() source when the parsed document carries
	// no URL of its own. Distinct from label, which is only a diagnostic source.
	schemaURI    string
	fsys         fs.FS          // filesystem for loading xs:include/xs:import/xs:redefine targets
	parser       *helium.Parser // parser governing schema-document parse policy
	errorHandler helium.ErrorHandler
}

type validateConfig struct {
	label          string
	errorHandler   helium.ErrorHandler
	annotations    *TypeAnnotations
	nilledElements *NilledElements
	// skipDatatypeIntegrity, when true, suppresses the document-wide
	// value-space integrity walks: the xs:ID / xs:IDREF / xs:IDREFS
	// uniqueness+referential-integrity walk (version-independent — runs in both
	// XSD 1.0 and 1.1) and the xs:ENTITY / xs:ENTITIES walk (XSD 1.1-only).
	// Content-model, type, and identity-constraint (xs:key/unique/keyref)
	// validation are unaffected. It exists for callers that validate an
	// element/subtree as a fragment and apply document-scope ID/IDREF integrity
	// themselves at the correct scope (e.g. xslt3).
	skipDatatypeIntegrity bool
}

// TypeAnnotations maps document nodes to their XSD type names.
// Type names use the "xs:localName" format for built-in types and
// "Q{ns}localName" for user-defined types.
type TypeAnnotations map[helium.Node]string

// NilledElements tracks elements whose xsi:nil="true" was confirmed valid
// during schema validation (i.e. the element declaration is nillable).
type NilledElements map[*helium.Element]struct{}

// Compiler compiles XSD documents into Schema values.
// It uses clone-on-write semantics: each builder method returns
// a new Compiler sharing the underlying config until mutation.
type Compiler struct {
	cfg *compileConfig
}

// NewCompiler creates a new Compiler with default settings.
func NewCompiler() Compiler {
	return Compiler{cfg: &compileConfig{}}
}

func (c Compiler) clone() Compiler {
	if c.cfg == nil {
		return Compiler{cfg: &compileConfig{}}
	}
	cp := *c.cfg
	return Compiler{cfg: &cp}
}

// Label sets the label (typically a filename) used in compilation error messages.
// If not set, the label is inferred from the document's URL ([helium.Document.URL]).
// If neither is available, "(string)" is used.
func (c Compiler) Label(name string) Compiler {
	c = c.clone()
	c.cfg.label = name
	return c
}

// BaseDir sets the base directory used to resolve relative paths in
// xs:include and xs:redefine elements during schema compilation.
func (c Compiler) BaseDir(dir string) Compiler {
	c = c.clone()
	c.cfg.baseDir = dir
	return c
}

// FS sets the [fs.FS] used to load schemas referenced by xs:include,
// xs:import, and xs:redefine during compilation.
//
// The default (and what a nil value restores) is a deny-all FS that refuses
// every open: a compiler from [NewCompiler] loads no nested schema from the
// host filesystem, so an untrusted schema cannot disclose local files or
// exhaust resources via a hostile schemaLocation. To opt into host access,
// pass [helium.PermissiveFS] (any os.Open path) or — preferably — a confined
// [fs.FS] rooted at a trusted directory. Each nested schema is read through a
// fixed byte cap regardless of the FS, so an endless source (e.g. a
// schemaLocation pointing at /dev/zero) cannot exhaust memory.
//
// Note: schema-location resolution is URI-aware. When [Compiler.BaseDir]
// is a URI (e.g. "https://example.com/s/main.xsd" or "file:///s/main.xsd"),
// a relative include is resolved against it with RFC 3986 semantics and an
// absolute-URI include is passed through unchanged, so the name handed to
// the FS is the canonical nested-schema URI. When BaseDir is a local
// filesystem path, names are built with [filepath.Join], so they may be
// absolute and may use OS-specific separators on Windows. FS implementations
// that enforce [fs.ValidPath] (notably [os.DirFS] and [testing/fstest.MapFS])
// will reject local OS-style names; for those, supply an FS implementation
// that accepts the names this compiler produces (URI strings, or OS-style
// paths for a local BaseDir).
func (c Compiler) FS(fsys fs.FS) Compiler {
	c = c.clone()
	if fsys == nil {
		fsys = iofs.DenyAll{}
	}
	c.cfg.fsys = fsys
	return c
}

// Parser sets the [helium.Parser] used to parse XSD schema documents — the
// top-level schema in [Compiler.CompileFile] as well as every schema pulled in
// via xs:include, xs:import, and xs:redefine. When unset, the compiler uses a
// default schema parser that expands entity references in schema attribute
// values. The injected parser supplies parse policy — resource limits and
// XXE/network controls — so a caller can apply one uniform policy across every
// helium component. The compiler's [Compiler.FS] still fetches the bytes, and no
// functional options or base URI are forced onto the injected parser.
func (c Compiler) Parser(p helium.Parser) Compiler {
	c = c.clone()
	c.cfg.parser = &p
	return c
}

// Version selects the XML Schema specification version the compiler targets.
// The default is [Version10]. Setting it explicitly overrides any vc:minVersion
// hint on the schema document; when left unset, the effective version may be
// upgraded to 1.1 by a vc:minVersion="1.1" attribute on the root <xs:schema>.
func (c Compiler) Version(v Version) Compiler {
	c = c.clone()
	c.cfg.version = v
	c.cfg.versionSet = true
	return c
}

// DefaultVersion sets the XML Schema specification version used as a fallback
// when the caller has not forced a version via [Compiler.Version] and the
// schema document declares no vc:minVersion hint on its root <xs:schema>.
//
// Version resolution order is: a forced [Compiler.Version] (always wins), then a
// vc:minVersion="1.1"-or-higher hint on the root, then this configured default,
// then [Version10]. So DefaultVersion changes only the "schema is silent on
// version" case; it never overrides an explicit Version() or a vc hint.
//
// The standalone compiler default remains [Version10]; this knob lets an
// embedding layer (e.g. xslt3's xsl:import-schema) opt its imported schemas into
// [Version11] semantics by default while still honoring an explicit version.
func (c Compiler) DefaultVersion(v Version) Compiler {
	c = c.clone()
	c.cfg.defaultVersion = v
	c.cfg.defaultVersionSet = true
	return c
}

// ErrorHandler sets a handler that receives compilation errors.
// When set, errors are delivered to the handler instead of being discarded.
func (c Compiler) ErrorHandler(h helium.ErrorHandler) Compiler {
	c = c.clone()
	c.cfg.errorHandler = h
	return c
}

func (c Compiler) closeHandler() {
	if c.cfg != nil && c.cfg.errorHandler != nil {
		if cl, ok := c.cfg.errorHandler.(io.Closer); ok {
			_ = cl.Close()
		}
	}
}

// Compile compiles an XSD document into a Schema.
// (libxml2: xmlSchemaNewParserCtxt + xmlSchemaParse)
func (c Compiler) Compile(ctx context.Context, doc *helium.Document) (*Schema, error) { //nolint:contextcheck
	if ctx == nil {
		ctx = context.Background()
	}
	cfg := c.cfg
	if cfg == nil {
		cfg = &compileConfig{}
	}
	// Seed the circular-include guard with the root schema's own resolved key,
	// mirroring CompileFile, so a cycle back to the top-level schema
	// (main -> inc -> main) treats the root as already-loaded instead of
	// re-parsing it and emitting spurious duplicate-component errors. Unlike
	// CompileFile, the in-memory/resolver path has no filesystem path to derive
	// the key from, so it is taken from the document's own URL (or, lacking that,
	// a full-URI BaseDir). rootSchemaKey is the single shared helper both compile
	// entry points use, so the seeded key cannot diverge from the key a nested
	// back-reference to the root computes.
	//
	// A clone of cfg keeps the shared config (clone-on-write contract) intact.
	cfgWithRoot := *cfg
	if rootKey, ok := rootSchemaKey(doc.URL(), cfg.baseDir); ok {
		cfgWithRoot.rootKey = rootKey
	}
	schema, err := compileSchema(ctx, doc, cfgWithRoot.baseDir, &cfgWithRoot)
	c.closeHandler()
	return schema, err
}

// CompileFile reads and compiles an XSD file into a Schema.
func (c Compiler) CompileFile(ctx context.Context, path string) (*Schema, error) { //nolint:contextcheck
	if ctx == nil {
		ctx = context.Background()
	}
	data, err := os.ReadFile(path) //nolint:gosec // path is caller-supplied schema file
	if err != nil {
		return nil, fmt.Errorf("xsd: failed to read %q: %w", path, err)
	}
	cfg := c.cfg
	if cfg == nil {
		cfg = &compileConfig{}
	}
	p := defaultSchemaParser()
	if cfg.parser != nil {
		p = *cfg.parser
	}
	doc, err := p.Parse(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("xsd: failed to parse %q: %w", path, err)
	}
	baseDir := filepath.Dir(path)
	// Seed the circular-include guard with the root schema's own resolved key, in
	// the same canonical form a nested xs:include/xs:redefine would compute when it
	// points back at the root (main -> inc -> main). Without this, includeVisited
	// only contains documents loaded via loadInclude/loadRedefine, so a cycle back
	// to the top-level schema re-parses it and emits spurious duplicates.
	// rootSchemaKey is the single shared helper both compile entry points use, so
	// the seeded key cannot diverge from the key a nested back-reference computes.
	// A clone of cfg keeps the shared config (and clone-on-write Compiler
	// contract) intact.
	cfgWithRoot := *cfg
	cfgWithRoot.schemaURI = path
	if rootKey, ok := rootSchemaKey(path, baseDir); ok {
		cfgWithRoot.rootKey = rootKey
	}
	schema, compileErr := compileSchema(ctx, doc, baseDir, &cfgWithRoot)
	c.closeHandler()
	return schema, compileErr
}

// Validator validates documents against a compiled XSD schema.
// It uses clone-on-write semantics: each builder method returns
// a new Validator sharing the underlying config until mutation.
type Validator struct {
	cfg    *validateConfig
	schema *Schema
}

// NewValidator creates a new Validator for the given schema.
func NewValidator(schema *Schema) Validator {
	return Validator{cfg: &validateConfig{}, schema: schema}
}

func (v Validator) clone() Validator {
	if v.cfg == nil {
		return Validator{cfg: &validateConfig{}, schema: v.schema}
	}
	cp := *v.cfg
	return Validator{cfg: &cp, schema: v.schema}
}

// Label sets the label (typically a filename) used in validation error messages.
// If not set, the label is inferred from the document's URL ([helium.Document.URL]).
// If neither is available, "(string)" is used.
func (v Validator) Label(name string) Validator {
	v = v.clone()
	v.cfg.label = name
	return v
}

// ErrorHandler sets a handler that receives validation errors.
func (v Validator) ErrorHandler(h helium.ErrorHandler) Validator {
	v = v.clone()
	v.cfg.errorHandler = h
	return v
}

// Annotations enables collection of per-node type annotations during
// validation. The caller must provide a non-nil pointer to a TypeAnnotations
// value; the map is populated during validation.
func (v Validator) Annotations(ann *TypeAnnotations) Validator {
	v = v.clone()
	v.cfg.annotations = ann
	return v
}

// NilledElements enables collection of nilled element information during
// validation. The caller must provide a non-nil pointer to a NilledElements
// value; the map is populated during validation.
func (v Validator) NilledElements(ne *NilledElements) Validator {
	v = v.clone()
	v.cfg.nilledElements = ne
	return v
}

// SkipDatatypeIntegrityChecks controls whether the document-wide xs:ID /
// xs:IDREF / xs:IDREFS uniqueness+referential-integrity walk and the xs:ENTITY /
// xs:ENTITIES value-space walk run. The ID/IDREF/IDREFS walk is
// version-independent (cvc-id) and runs in both XSD 1.0 and 1.1; the
// xs:ENTITY / xs:ENTITIES walk is XSD 1.1-only. So in XSD 1.0 this option
// suppresses the ID/IDREF/IDREFS walk (the ENTITY walk never runs there).
//
// When enabled, those document-scoped datatype-integrity walks are skipped.
// Content-model validation, simple/complex type validation, and the
// xs:key/xs:unique/xs:keyref identity-constraint walk are unaffected.
//
// This is for callers that validate an element or subtree as a fragment (so a
// self-contained "document" is not the real scope) and enforce document-scope
// ID/IDREF integrity themselves at the correct granularity — notably xslt3,
// which validates a constructed element via a temporary document but must not
// apply whole-document ID uniqueness to an element-level validation.
func (v Validator) SkipDatatypeIntegrityChecks(skip bool) Validator {
	v = v.clone()
	v.cfg.skipDatatypeIntegrity = skip
	return v
}

func (v Validator) closeHandler() {
	if v.cfg != nil && v.cfg.errorHandler != nil {
		if cl, ok := v.cfg.errorHandler.(io.Closer); ok {
			_ = cl.Close()
		}
	}
}

// Validate validates a document against the compiled schema.
//
// It returns ErrNilSchema when the Validator has no compiled schema,
// ErrNilDocument when doc is nil, and ErrValidationFailed when the document
// is invalid; it returns nil when the document is valid. Individual
// validation errors are delivered to the ErrorHandler if one is configured.
// A nil ctx is normalized to context.Background().
// (libxml2: xmlSchemaValidateDoc)
func (v Validator) Validate(ctx context.Context, doc *helium.Document) error { //nolint:contextcheck
	if ctx == nil {
		ctx = context.Background()
	}

	cfg := v.cfg
	if cfg == nil {
		cfg = &validateConfig{}
	}

	handler := cfg.errorHandler
	if handler == nil {
		handler = helium.NilErrorHandler{}
	}

	// Close the handler on every exit path, including the nil guards below,
	// so a closable ErrorHandler (e.g. helium.ErrorCollector) is not leaked.
	defer v.closeHandler()

	if v.schema == nil {
		return ErrNilSchema
	}

	if doc == nil {
		return ErrNilDocument
	}

	valid := validateDocument(ctx, doc, v.schema, cfg, handler)

	if valid {
		return nil
	}
	return ErrValidationFailed
}
