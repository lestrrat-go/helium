# Package Map

Go implementation of libxml2. Module: `github.com/lestrrat-go/helium`

## Root (`helium`)

XML parsing, DOM tree, serialization. Entry point for all XML processing.

- **NewParser() → Parser** — create fluent builder for XML parsing (clone-on-write value type)
  - Flag methods: `RecoverOnError(bool)`, `SubstituteEntities(bool)`, `LoadExternalDTD(bool)`, `DefaultDTDAttributes(bool)`, `ValidateDTD(bool)`, `SuppressErrors(bool)`, `SuppressWarnings(bool)`, `PedanticErrors(bool)`, `StripBlanks(bool)`, `ProcessXInclude(bool)`, `AllowNetwork(bool)`, `CleanNamespaces(bool)`, `MergeCDATA(bool)`, `XIncludeNodes(bool)`, `CompactTextNodes(bool)`, `FixBaseURIs(bool)`, `RelaxLimits(bool)`, `IgnoreEncoding(bool)`, `BigLineNumbers(bool)`, `BlockXXE(bool)`, `ReuseDict(bool)`, `SkipIDs(bool)`, `LenientXMLDecl(bool)`
  - Config methods: `SAXHandler(sax.SAX2Handler)`, `BaseURI(string)`, `CharBufferSize(int)`, `MaxDepth(int)`, `MaxExternalDTDBytes(int)`, `Catalog(CatalogResolver)`
  - Terminal methods: `Parse(ctx, []byte) → (*Document, error)`, `ParseReader(ctx, io.Reader) → (*Document, error)`, `ParseFile(ctx, string) → (*Document, error)`, `ParseInNodeContext(ctx, Node, []byte) → (Node, error)`, `NewPushParser(ctx) → *PushParser`
- **NewWriter() → Writer** — create fluent XML writer builder
  - Writer methods: `Format(bool)`, `IndentString(string)`, `SelfCloseEmptyElements(bool)`, `XMLDeclaration(bool)`, `IncludeDTD(bool)`, `EscapeNonASCII(bool)`, `AllowPrefixUndeclarations(bool)`
  - Terminal method: `WriteTo(io.Writer, Node) → error`
- **Write(io.Writer, Node) → error** — serialize node with default settings
- **WriteString(Node) → (string, error)** — serialize node to string with default settings
- **Element.FindAttribute(AttributePredicate) → (*Attribute, bool)** — attribute-node lookup by matcher; built-in matchers: `QNamePredicate`, `LocalNamePredicate`, `NSPredicate`
- **Element.GetAttribute(qname) → (string, bool)** / **Element.GetAttributeNS(local, nsURI) → (string, bool)** — attribute value lookup by QName or expanded name
- Key types: `Document`, `Element`, `Attribute`, `Namespace`, `DTD`, `Entity`, `Text`, `CDATASection`, `Comment`, `PI`
- `Node` interface — common for all node types; use ElementType enum to distinguish
- Parse flags configured via fluent methods on Parser (internal bitset, not public)
- `ErrorHandler` interface — async error delivery during parsing
- `CatalogResolver` interface — public interface for custom catalog resolvers (`Resolve(ctx, pubID, sysID)`, `ResolveURI(ctx, uri)`)
- `ErrExternalDTDTooLarge` — sentinel error returned when a loaded external DTD subset exceeds the byte cap; enforced against actual bytes read, never the advisory `fs.FileInfo.Size()`
- `MaxExternalDTDSize` — default external-DTD byte cap (10 MiB), used when `MaxExternalDTDBytes` is unset or ≤ 0
- `Parser.MaxExternalDTDBytes(n int)` — override the external-DTD byte cap (n ≤ 0 → `MaxExternalDTDSize`)
- `AsNode[T Node](n Node) (T, bool)` — generic safe type assertion for Node types
- `Document.GetElementByID(id)` — O(1) via hash table, O(n) fallback
- `Walk(doc, fn)`, `Children(node)`, `Descendants(node)` — tree traversal
- `CopyNode(src, targetDoc)` — deep copy across documents
- `NodeGetBase(doc, node)` — effective xml:base URI
- `BuildURI(base, ref)` — resolve relative URI
- Files: `parser.go` (API), `parserctx.go` (context/state), `parser_document.go`, `parser_element.go`, `parser_whitespace.go`, `parser_xml_decl.go`, `parser_encoding.go`, `parser_decl.go`, `parser_content.go`, `parser_dtd_subset.go`, `parser_dtd_element.go`, `parser_dtd_attr.go`, `parser_entity_decl.go`, `parser_entity_ref.go`, `parser_state_gen.go`, `document.go`, `element.go`, `attribute.go`, `node.go`, `node_leaf.go`, `node_namespace.go`, `node_base.go`, `tree_builder.go`, `tree_namespaces.go`, `writer.go`, `writer_escape.go`, `writer_dtd.go`, `writer_xhtml.go`, `copy.go`, `dtd.go`, `dtd_attr.go`, `dtd_elem.go`, `iter.go`, `errorhandler.go`, `resolver.go`, `doc.go`

## c14n/

W3C Canonical XML. 3 modes: C14N10, ExclusiveC14N10, C14N11.

- **NewCanonicalizer(Mode) → Canonicalizer** — create fluent builder for the given mode
- Canonicalizer methods: Comments(), NodeSet([]Node), InclusiveNamespaces([]string), BaseURI(string)
- Terminal: **Canonicalize(*Document, io.Writer) → error**, **CanonicalizeTo(*Document) → ([]byte, error)**
- Files: `c14n.go` (API), `canonicalizer.go` (engine), `nsstack.go`, `sort.go`, `escape.go`
- Imports: helium

## xpath1/

XPath 1.0 expression parsing and evaluation.

- **Compile(string) → (*Expression, error)** / **MustCompile(string) → *Expression** — parse XPath
- **Expression.Evaluate(ctx, Node) → (*Result, error)**
- **Find(ctx, Node, string) → ([]Node, error)** — convenience: compile+evaluate→node-set
- **Evaluate(ctx, Node, string) → (*Result, error)** — convenience: compile+evaluate
- **WithNamespaces(ctx, ns) → context.Context** / **WithVariables(ctx, vars) → context.Context** / **WithOpLimit(ctx, n) → context.Context** — attach XPath evaluation settings to `context.Context`
- **WithFunction(ctx, name, fn) → context.Context** / **WithFunctionNS(ctx, uri, name, fn) → context.Context** — register custom functions on `context.Context`
- **WithFunctions(ctx, fns) → context.Context** / **WithFunctionsNS(ctx, fns) → context.Context** — bulk function registration
- `Result` types: NodeSetResult, BooleanResult, NumberResult, StringResult
- `FunctionContext` — read-only custom-function evaluation state; retrieve via `GetFunctionContext(ctx)`
- Merge helpers: `WithAdditionalNamespaces(ctx, ns)`, `WithAdditionalVariables(ctx, vars)`
- Limits: recursion 5000, node-set 10M, configurable op limit
- Robustness: `eval` and axis-iteration loops honor `ctx.Err()` so a cancelled context aborts promptly; `Evaluate` on a nil/zero-value `Expression` returns `ErrNilExpression` instead of panicking
- Files: `xpath.go` (API), `parser.go`, `lexer.go`, `eval.go`, `expr.go`, `axes.go`, `functions.go`, `token.go`
- Imports: helium

## xpath3/

XPath 3.1 expression parsing and evaluation.

- **NewCompiler() → Compiler** — create fluent builder for expression compilation
  - `Compile(string) → (*Expression, error)` / `MustCompile(string) → *Expression` / `CompileExpr(Expr) → (*Expression, error)` — terminal methods
- **NewEvaluator(EvaluatorOptions) → Evaluator** — create evaluator
  - `Evaluate(ctx, *Expression, Node) → (*Result, error)` — terminal method
- **Expression.DumpVM(io.Writer) → error** — write compiled VM instruction dump for debugging/tooling
- **WithNamespaces(ctx, ns) → context.Context** / **WithVariables(ctx, vars) → context.Context** / **WithOpLimit(ctx, n) → context.Context** — attach XPath 3.1 evaluation settings to `context.Context`
- **WithFunction(ctx, name, fn) → context.Context** / **WithFunctionNS(ctx, uri, name, fn) → context.Context** — register custom functions on `context.Context`
- **WithFunctions(ctx, fns) → context.Context** / **WithFunctionsNS(ctx, fns) → context.Context** — bulk function registration
- `Result` — wraps `Sequence`; methods: `Nodes()`, `IsBoolean()`, `IsNumber()`, `IsString()`, `IsAtomic()`, `Atomics()`, `Sequence()`
- Merge helpers: `WithAdditionalNamespaces(ctx, ns)`, `WithAdditionalVariables(ctx, vars)`
- Direct mutators also include `WithDefaultLanguage(ctx, lang)`, `WithDefaultCollation(ctx, uri)`, `WithDefaultDecimalFormat(ctx, df)`, `WithNamedDecimalFormats(ctx, dfs)`, `WithBaseURI(ctx, uri)`, `WithURIResolver(ctx, r)`, `WithCollectionResolver(ctx, r)`, `WithHTTPClient(ctx, client)`, `WithImplicitTimezone(ctx, loc)`
- **Evaluator.MaxResourceBytes(int64) → Evaluator** — cap bytes read from a single external resource by fn:unparsed-text(-lines/-available), fn:doc, fn:doc-available, fn:json-doc (0 = default cap, negative = unbounded); over-cap reads in fn:unparsed-text/fn:unparsed-text-lines fail FOUT1170 (fn:unparsed-text-available returns false), while fn:doc/fn:json-doc retrieval failures (incl. over-cap) surface as FODC0002 and fn:doc-available returns false
- XPath 3.1 features: FLWOR, quantified, if-then-else, try-catch, maps, arrays, inline functions, HOFs, arrow operator, simple map, string concat, value/general/node comparisons
- Built-in functions: 100+ across fn:, math:, map:, array: namespaces
- Type system: Sequence ([]Item), AtomicValue, NodeItem, MapItem, ArrayItem, FunctionItem
- Structured errors: XPathError with W3C error codes (XPTY0004, FOER0000, etc.)
- Limits: recursion 5000, node-set 10M, configurable op limit
- Runtime: `Compile()` first tries a direct fast path for simple path-like expressions and simple predicate comparisons, otherwise lowers AST to a VM instruction graph while collecting the prefix-validation plan, keeping trivial leaves inline in parent payloads and reusing parsed slices on the owned compile path; `Evaluate()` executes compiled refs by opcode and reuses shared eval helpers for semantics; AST/streamability access reparses from `Expression.source` on demand
- Files: `xpath3.go` (API), `parser.go`, `lexer.go`, `expr.go`, `token.go`, `eval.go`, `eval_dispatch.go`, `eval_path.go`, `eval_operators.go`, `eval_arithmetic.go`, `eval_control.go`, `eval_types.go`, `eval_funcall.go`, `eval_reuse.go`, `eval_state.go`, `evaluator.go`, `vm.go`, `vm_dump.go`, `compile_direct.go`, `compiler.go`, `compare.go`, `cast.go`, `cast_numeric.go`, `cast_string.go`, `cast_datetime.go`, `types.go`, `float_value.go`, `sequence.go`, `context.go`, `variables.go`, `collation.go`, `regex.go`, `regex_public.go`, `static_check.go`, `streamability.go`, `node_identity.go`, `uri_resolution.go`, `doc.go`, `errors.go`, `arithmetic_datetime.go`, `parse_ietf_date.go`, `format_datetime.go`, `format_integer.go`, `format_number.go`, `function_library.go`, `function_signatures.go`, `functions.go`, `functions_node.go`, `functions_string.go`, `functions_numeric.go`, `functions_boolean.go`, `functions_aggregate.go`, `functions_sequence.go`, `functions_datetime.go`, `functions_uri.go`, `functions_qname.go`, `functions_hof.go`, `functions_map.go`, `functions_array.go`, `functions_math.go`, `functions_error.go`, `functions_misc.go`, `functions_json.go`, `functions_constructors.go`, `functions_unparsed_text.go`
- Imports: helium, internal/xpath, internal/lexicon, internal/icu, internal/unparsedtext, internal/strcursor, internal/sequence

## xslt3/

XSLT 3.0 stylesheet compilation + transformation on helium DOM with `xpath3` evaluation.

- **CompileStylesheet(ctx, *Document) → (*Stylesheet, error)** — convenience compile wrapper
- **NewCompiler() → Compiler** — builder for stylesheet compilation
- **Compiler.BaseURI(string) → Compiler** / **Compiler.URIResolver(URIResolver) → Compiler** / **Compiler.PackageResolver(PackageResolver) → Compiler** — compile-time resource/package resolution. Secure by default: `Compiler.URIResolver` is the opt-in for ALL compile-time stylesheet loading — `xsl:import`, `xsl:include`, output-format parameter documents (`xsl:output @parameter-document`), and compile-time `fn:transform` `stylesheet-location` (e.g. static-variable evaluation). With no `URIResolver` configured there is no implicit `os.ReadFile`; each of those loads errors out (`xsl:import`/`xsl:include` → "no URIResolver configured"; parameter docs → XTSE0090; `fn:transform` → FOXT0003). Runtime `fn:transform stylesheet-location` likewise requires the compile-time `URIResolver` carried on the stylesheet.
- **Compiler.StaticParameters(*Parameters) → Compiler** / **Compiler.SetStaticParameter(string, Sequence) → Compiler** / **Compiler.ClearStaticParameters() → Compiler** / **Compiler.ImportSchemas(...*xsd.Schema) → Compiler** — compile-time static params + schema imports
- **Compiler.MaxResourceBytes(int64) → Compiler** — set the per-resource read cap inherited by invocations (0 = [MaxResourceBytes] default, negative = unbounded, positive = that cap)
- **Compiler.AllowExternalEntities(bool) → Compiler** — XXE policy for compile-time parses of external stylesheet modules (`xsl:import`/`xsl:include`/`xsl:use-package`, and compile-time `fn:transform` stylesheets). **Default false (XXE BLOCKED): external DTDs / external general entities are NOT loaded or substituted** (parser is `BlockXXE(true).LoadExternalDTD(false).AllowNetwork(false)`). Set true to restore the legacy permissive behavior (resolver-mediated external entity loading via `LoadExternalDTD(true).SubstituteEntities(true)`, subject to `MaxResourceBytes`). The compiled value is carried on the `Stylesheet` and inherited by `fn:transform` nested compiles and (unless overridden) by runtime invocations. Serialization parameter documents and imported XSD schemas are always parsed XXE-blocked.
- **Compiler.Compile(ctx, *Document) → (*Stylesheet, error)** / **Compiler.MustCompile(ctx, *Document) → *Stylesheet** — terminal compile methods
- **Transform(ctx, *Document, *Stylesheet) → (*Document, error)** / **TransformToWriter(ctx, *Document, *Stylesheet, io.Writer) → error** / **TransformString(ctx, *Document, *Stylesheet) → (string, error)** — convenience wrappers; nil `*Stylesheet` returns error here
- **Stylesheet.Transform(*Document) → Invocation** / **Stylesheet.ApplyTemplates(*Document) → Invocation** / **Stylesheet.CallTemplate(string) → Invocation** / **Stylesheet.CallFunction(string, ...Sequence) → Invocation** — invocation entrypoints
- **Invocation.SourceDocument(*Document) → Invocation** / **Mode(string)** / **Selection(Sequence)** *(ApplyTemplates only)* / **GlobalParameters(*Parameters)** / **TunnelParameters(*Parameters)** / **SetParameter(string, Sequence)** / **SetTunnelParameter(string, Sequence)** / **SetInitialTemplateParameter(string, Sequence)** / **SetInitialModeParameter(string, Sequence)** / **MessageHandler(MessageHandler)** / **ResultDocumentHandler(ResultDocumentHandler)** / **RawResultHandler(RawResultHandler)** / **PrimaryItemsHandler(PrimaryItemsHandler)** / **AnnotationHandler(AnnotationHandler)** / **CollectionResolver(xpath3.CollectionResolver)** / **URIResolver(xpath3.URIResolver)** / **HTTPClient(\*http.Client)** / **BaseOutputURI(string)** / **SourceSchemas(...*xsd.Schema)** / **OnMultipleMatch(OnMultipleMatchMode)** / **TraceWriter(io.Writer)** / **MaxResourceBytes(int64)** / **AllowExternalEntities(bool)** — fluent runtime configuration. `AllowExternalEntities` sets the XXE policy for runtime parses of external documents (`fn:doc`/`document()`, `xsl:source-document`, `xsl:merge`, and `fn:transform` stylesheet sources). **Default false (XXE BLOCKED): external DTDs / external general entities are NOT loaded or substituted**; when left unset it inherits the value compiled into the stylesheet (`Compiler.AllowExternalEntities`); set true to restore the legacy permissive behavior for trusted documents. `MaxResourceBytes` caps bytes read from a single runtime external resource: 0 inherits the Compiler/stylesheet cap (then the [MaxResourceBytes] default), negative disables the bound, positive sets that cap. The cap applies to all runtime reads, but the over-cap error differs by layer: XSLT's own loader (`fn:doc`/`document()`, `xsl:source-document`, `xsl:merge`, runtime `xsl:result-document` parameter documents, `xsi:schemaLocation` source schemas, `fn:transform` stylesheet/package sources) fails with [ErrResourceTooLarge], whereas the XPath built-ins `fn:unparsed-text`/`fn:unparsed-text-lines` surface FOUT1170 (`fn:unparsed-text-available` returns false) and `fn:json-doc` surfaces FODC0002 — they honor the cap but do NOT carry the `ErrResourceTooLarge` sentinel. `URIResolver` and `HTTPClient` are the opt-in for runtime resource retrieval — `fn:doc`/`fn:unparsed-text`, plus `xsl:source-document`, `xsl:merge`, and `fn:stream-available`; without them those instructions error (`FODC0002`) or report unavailable per the default-deny model (no implicit `os.ReadFile`).
- **Invocation.Do(ctx) → (*Document, error)** / **Invocation.Serialize(ctx) → (string, error)** / **Invocation.WriteTo(ctx, io.Writer) → error** / **Invocation.ResolvedOutputDef() → *OutputDef** — terminal execution + resolved primary output metadata
- **NewParameters() → *Parameters** — mutable XSLT parameter carrier keyed by expanded name
- Key types: `Stylesheet`, `Compiler`, `Invocation`, `Parameters`, `OutputDef`, `URIResolver`, `PackageResolver`, `MessageHandler`, `ResultDocumentHandler`, `RawResultHandler`, `PrimaryItemsHandler`, `AnnotationHandler`
- Resource limits: `MaxResourceBytes` (const, 10 MiB default per-resource read cap) + `ErrResourceTooLarge` (error returned when an external resource exceeds the cap); enforced against actual bytes read, configurable per Compiler/Invocation
- Supports: `xsl:template`, `xsl:apply-templates`, `xsl:call-template`, `xsl:param`/`xsl:variable`, `xsl:include`/`xsl:import`, `xsl:sort`, `xsl:number`, `xsl:message`, `xsl:key`, `xsl:output`, `xsl:import-schema`, `xsl:function`, literal result elements, AVTs, `xsl:attribute-set`, `xsl:map`/`xsl:map-entry`, `xsl:source-document`, `xsl:iterate`, `xsl:fork`, `xsl:accumulator`, `xsl:merge`, `xsl:where-populated`, `xsl:try`/`xsl:catch`, `xsl:for-each-group`, `xsl:result-document`, `xsl:next-match`, `xsl:apply-imports`
- Schema awareness: `xsl:import-schema` compiles XSD schemas, `type=` on `xsl:element`/`xsl:attribute` annotates result nodes, `validation=` on `xsl:copy`/`xsl:copy-of`, `default-validation` stylesheet attribute, `type-available()` function, runtime source validation via `Invocation.SourceSchemas(...)`, annotation callbacks via `AnnotationHandler`
- Streaming: `xsl:source-document` (DOM-materialization), `xsl:iterate`/`xsl:break`/`xsl:next-iteration`/`xsl:on-completion`, `xsl:fork`, `xsl:accumulator`/`xsl:accumulator-rule`, `xsl:merge`/`xsl:merge-source`/`xsl:merge-key`/`xsl:merge-action`; streamability analysis (XTSE3430) post-compilation pass
- Runtime helpers: `current()`, `document()`, `key()`, `generate-id()`, `system-property()`, `unparsed-entity-uri()`, `unparsed-entity-public-id()`, `type-available()`, `snapshot()`, `copy-of()`, `accumulator-before()`/`accumulator-after()`, `current-merge-group()`/`current-merge-key()`, `transform()`
- Output methods: `xml`, `html`, `xhtml`, `text`, `json`, `adaptive`
- Files: `xslt3.go` (package doc + convenience wrappers), `compile.go` (compiler builder + orchestration), `compile_*.go` (imports/packages/schema/templates/functions/modes/formats/patterns/streaming/instruction compilation), `execute*.go` (runtime), `functions*.go` (built-ins + `fn:transform` bridge), `stylesheet.go`, `stylesheet_entry.go`, `invocation.go`, `instruction.go`, `parameters.go`, `receiver.go`, `output.go`, `sort.go`, `types.go`, `avt.go`, `keys.go`, `number_words.go`, `source_schema.go`, `schema_constructors.go`, `schema_context.go`, `package_*.go`, `streamability*.go`, `elements.go`, `errors.go`, `resource_limit.go` (per-resource read cap + `MaxResourceBytes`/`ErrResourceTooLarge`)
- Imports: helium, xpath3, xsd, html, internal/lexicon, internal/sequence, xslt3/internal/elements
- Tests: hand-written unit tests + generated W3C suites `w3c_*_gen_test.go` with shared `w3c_helpers_test.go`; W3C source suite fetched into `testdata/xslt30/source/`

## xslt3/internal/elements/

XSLT element registry: metadata for all ~80 recognized XSLT 3.0 elements.

- **NewRegistry() → *Registry** — create fully initialized element registry
- **Registry.IsKnown(name) → bool** — recognized XSLT element check
- **Registry.IsTopLevel(name) → bool** — allowed as xsl:stylesheet child
- **Registry.IsInstruction(name) → bool** — allowed in sequence constructors
- **Registry.IsImplemented(name) → bool** — recognized and implemented
- **Registry.MinVersion(name) → string** — minimum XSLT version ("1.0", "2.0", "3.0")
- **Registry.AllowedAttrs(name) → (map[string]struct{}, bool)** — element-specific unprefixed attrs
- **Registry.ValidParents(name) → []string** — valid parent elements for child-only elements
- **Registry.IsValidChild(child, parent) → bool** — parent-child validation
- Types: `ElementInfo`, `ElementContext` (bitmask: `CtxTopLevel`, `CtxInstruction`, `CtxChildOnly`, `CtxRoot`)
- Files: `elements.go` (Registry API), `data.go` (element definitions)
- Imports: internal/lexicon

## xsd/

XML Schema (XSD) 1.0 compilation and validation.

- **NewCompiler() → Compiler** — create fluent builder for schema compilation
  - `Label(name)`, `BaseDir(dir)`, `ErrorHandler(h)` — builder methods (clone-on-write)
  - `Compile(ctx, *Document) → (*Schema, error)` / `CompileFile(ctx, path) → (*Schema, error)` — terminal methods; return `(nil, ErrCompilationFailed)` on fatal schema diagnostics
- **NewValidator(schema) → Validator** — create fluent builder for validation
  - `Label(name)`, `ErrorHandler(h)`, `Annotations(*TypeAnnotations)`, `NilledElements(*NilledElements)` — builder methods
  - `Validate(ctx, *Document) → error` — terminal method
- **(*TypeDef).Validate(ctx, value, nsMap) → error** — validate a lexical value against a simple type; nsMap (prefix→URI) may be nil
- **(*TypeDef).ValidateElement(ctx, elem, schema) → error** — validate an element's content against a type
- `Schema.LookupElement(local, ns)`, `Schema.LookupType(local, ns)`, `Schema.NamedTypes()`, `Schema.TargetNamespace()`
- **ResolveSchemaURI(ref, base) → (string, error)** / **URIScheme(s) → string** — the single canonical schema-location URI-resolution helper and scheme-detector, shared with `xslt3` so the two layers cannot drift (URI-aware: absolute-URI pass-through, RFC 3986 with `OmitHost` preservation for URI bases, `filepath.Join` + `..`-escape guard for local bases)
- **FatalSchemaLoader** interface (`FatalSchemaLoad() bool`) — a `Compiler.FS` may return an `Open` error whose chain carries a value satisfying this interface to force an `xs:import` load failure to be FATAL instead of the usual warn-and-continue ("Skipping the import."). `xslt3`'s `schemaResolverFS` uses it so an over-cap nested-import read (`ErrResourceTooLarge`) is not silently skipped; the wrapped chain is preserved so callers can still `errors.Is`/`errors.As` the cause
- **IsFatalSchemaLoad(err) → bool** — the SINGLE source of truth for "is this a fatal schema-load condition that must abort compilation rather than warn-and-continue or fall back to a pre-compiled schema". Returns true (via `errors.Is`/`errors.As`) for a schema-location `..`-escape, an `xs:import` depth overflow, and any error satisfying `FatalSchemaLoader`. The two xsd import warn-or-continue sites and `xslt3`'s `xsl:import-schema` fallback guard both route through it (xslt3's `isFatalSchemaLoadError` delegates to it, adding the xslt3-package `ErrResourceTooLarge` sentinel), so the classification cannot drift between the layers. The path-escape / depth sentinels stay unexported; this helper is the public surface
- Supports: complex/simple types, sequences, choices, all, groups, attribute groups, substitution groups, import/include, IDC (xs:unique/key/keyref)
- `ErrValidationFailed` — sentinel error returned by `Validate()` when the document is invalid; individual errors delivered via `ErrorHandler`. `Validate()` also returns `ErrNilSchema` (no compiled schema) and `ErrNilDocument` (nil document); a nil `ctx` is normalized to `context.Background()`
- `ErrCompilationFailed` — sentinel error returned by `Compile()`/`CompileFile()` when the schema has one or more fatal errors; the returned schema is nil and individual diagnostics are delivered via `ErrorHandler`
- Files: `xsd.go` (API), `schema.go` (data model), `compile.go` + `compile_imports.go` + `compile_helpers.go` (compile orchestration/imports/helpers), `resolve_uri.go` (shared schema-location URI resolver `ResolveSchemaURI`/`URIScheme`), `read_types.go` + `read_particles.go` + `read_elements.go` + `read_decl_helpers.go` (schema readers), `link_refs.go` + `check_*.go` (reference resolution + constraints), `validate_context.go` + `validate.go` + `validate_elem.go` + `validate_idc.go` (validation flow/content/IDC), `simplevalue_*.go` + `typedef_validate.go` (simple-value engine/TypeDef API), `errors.go`
- Imports: helium, xpath1/, internal/lexicon
- Status: 225/226 golden tests passing

## relaxng/

RELAX NG schema compilation and validation.

- **NewCompiler() → Compiler** — create fluent builder for grammar compilation
  - `Label(name)`, `ErrorHandler(h)` — builder methods (clone-on-write)
  - `Compile(ctx, *Document) → (*Grammar, error)` / `CompileFile(ctx, path) → (*Grammar, error)` — terminal methods
- **NewValidator(grammar) → Validator** — create fluent builder for validation
  - `Filename(name)`, `ErrorHandler(h)` — builder methods
  - `Validate(ctx, *Document) → error` — terminal method
- Pattern-based: element, attribute, group, choice, interleave, optional, zeroOrMore, oneOrMore, ref, data, value, list, mixed, notAllowed
- Supports: include with override, externalRef, parentRef, anyName/nsName/ncName, data types
- Group backtracking for greedy pattern over-consumption
- `ValidateError.Output` — libxml2-compatible error string; `ValidateError.Errors` — structured `[]ValidationError`
- `ValidationError{Filename, Line, Element, Message}` — per-error structured type
- Files: `relaxng.go` (API + config), `doc.go`, `grammar.go` (data model), `parse.go` (compiler), `parse_check.go` (compile checks), `validate.go` (engine), `errors.go` (error types + formatting)
- Imports: helium, internal/lexicon, internal/iofs, internal/xsd/value, internal/xmlchar
- Status: 159/159 golden tests passing

## html/

HTML 4.01 parser producing helium DOM or SAX events.

- **NewParser() → Parser** — create fluent parser builder
- Parser methods: `SuppressImplied(bool)`, `StripBlanks(bool)`, `SuppressErrors(bool)`, `SuppressWarnings(bool)`, `Strict(bool)`, `MaxContentSize(int)` (approximate soft per-chunk cap for raw-text/RCDATA/plaintext — chunks target this size but an indivisible token, e.g. a whole UTF-8 rune or resolved char-ref, is never split, so a chunk may slightly exceed it; HARD cap for comment/bogus-comment/PI — over-cap fails the parse with `ErrContentSizeExceeded` since those are indivisible nodes; RCDATA char-ref resolution uses a FIXED `maxEntityNameLen` (~32 byte) lookahead independent of the cap, so a SHORT resolvable named reference (known entity or legacy prefix) whose run fits the cap is never rejected for being a small name (`&amp;` resolves under `MaxContentSize(2)`); ANY UNRESOLVED named-reference literal (whether short, semicolon-terminated, or unbounded) fails with `ErrContentSizeExceeded` once the bytes it would emit (`&` + name + optional `;`) exceed the cap; a SATURATED ambiguous legacy-prefix run (`&amp` + a tail that overflows the 32-byte lookahead) is consumed into a cap-bounded spool and HARD-FAILS with `ErrContentSizeExceeded` if it exceeds the cap before its end is reached, emitting nothing — only a within-cap saturated run legacy-resolves; default 16 MiB)
- Terminal: **Parse(ctx, []byte)**, **ParseReader(ctx, io.Reader)**, **ParseFile(ctx, path)**, **ParseWithSAX(ctx, []byte, SAXHandler)**, **NewPushParser(ctx)**, **NewSAXPushParser(ctx, SAXHandler)**
- **NewWriter() → Writer** — create fluent writer builder
- Writer methods: `DefaultDTD(bool)`, `Format(bool)`, `PreserveCase(bool)`, `EscapeURIAttributes(bool)`, `EscapeControlChars(bool)`
- Terminal: **WriteTo(io.Writer, Node)**
- **Write(io.Writer, Node) → error** — serialize with default settings
- **WriteString(Node) → (string, error)** — serialize to string with default settings
- Auto-closing, void elements, implicit html/head/body insertion
- Encoding: prescan charset=utf-8 → U+FFFD for invalid bytes; otherwise Latin-1/Win-1252→UTF-8
- Entity resolution: 2125 WHATWG + 106 legacy HTML4; legacy entities work without `;`
- Files: `html.go` (API), `parser.go`, `entities.go`, `elements.go`, `dump.go` (serializer), `tree.go` (DOM builder), `sax.go`
- Imports: helium, sax/

## xinclude/

XInclude 1.0 processing with recursive inclusion and fallback.

- **NewProcessor() → Processor** — create fluent builder
- Processor methods: `NoXIncludeMarkers()`, `NoBaseFixup()`, `Resolver(Resolver)`, `BaseURI(string)`, `MaxIncludeSize(int)`, `ErrorHandler(helium.ErrorHandler)`
- Terminal: **Process(ctx, *Document) → (int, error)**, **ProcessTree(ctx, Node) → (int, error)**
- `Resolver` interface — custom resource loader; receives the href already resolved against the effective base (base arg is informational only — do NOT re-resolve, or the base directory is double-applied)
- `MaxIncludeSize` — default per-include byte cap (10 MiB), used when `Processor.MaxIncludeSize` is unset or ≤ 0; over-cap reads fail with `ErrIncludeTooLarge`
- Default `NewFSResolver` converts absolute `file:` hrefs to OS paths via `internal/iofs.FileURIToPath` (non-local hosts rejected)
- Max depth 40, max URI 2000 chars, circular detection, doc/text caching
- Files: `xinclude.go`
- Imports: helium, xpointer/, internal/encoding/, internal/iofs/, internal/lexicon/

## xpointer/

XPointer expression evaluation with scheme cascading.

- **Evaluate(ctx, *Document, string) → ([]Node, error)**
- Schemes: xpointer(), xpath1() → XPath; element(/1/2/3) → child-sequence; xmlns() → ns binding; shorthand → ID lookup
- Multiple scheme parts left-to-right; first non-empty result wins
- `Compile(string) → (*Expression, error)` + `Expression.Evaluate(ctx, *Document) → ([]Node, error)` for reuse across documents
- `ErrNilExpression` — sentinel returned by `Expression.Evaluate` when the receiver is nil or an uncompiled (zero-value) `Expression`
- `ErrNilDocument` — sentinel returned by `Expression.Evaluate`/`Evaluate` when the document is nil
- Files: `xpointer.go`
- Imports: helium, xpath1/, internal/xmlchar/

## schematron/

Schematron schema compilation and validation.

- **Compiler** (fluent, clone-on-write): `NewCompiler()` → `.Label(s)` / `.ErrorHandler(h)` → `.Compile(ctx, doc)` or `.CompileFile(ctx, path)`
- **Validator** (fluent, clone-on-write): `NewValidator(schema)` → `.Label(s)` / `.Quiet()` / `.ErrorHandler(h)` → `.Validate(ctx, doc)`
- `ErrValidationFailed` — sentinel returned by `Validator.Validate` on validation failure; individual `*ValidationError` delivered to ErrorHandler
- `ErrNoSchema` — sentinel returned by `Validator.Validate` when the Validator has no compiled schema
- `ErrCompileFailed` — sentinel returned by `Compiler.Compile`/`CompileFile` when compilation fails
- Supports: schema, pattern, rule, assert, report, let, name, value-of
- Variable bindings via `<let>` and `<param>`
- Files: `schematron.go` (API + config), `schema.go` (data model), `parse.go` (compilation), `validate.go` (validation), `errors.go` (error types + formatting)
- Imports: helium, xpath1/

## catalog/

OASIS XML Catalog resolution for public/system IDs and URIs.

- **Load(ctx, path) → (*Catalog, error)** — convenience wrapper around `NewLoader().Load`
- **NewLoader() → Loader** — fluent value-style loader; methods return updated copies
- **Loader.ErrorHandler(h) → Loader** — deliver parse warnings to a handler
- **Loader.MaxBytes(n) → Loader** — cap catalog file size; exceed → `ErrCatalogTooLarge` (default `MaxCatalogSize`, 10 MiB)
- **Catalog.Resolve(ctx, pubID, sysID) → string** — resolve external identifier
- **Catalog.ResolveURI(ctx, uri) → string** — resolve URI reference
- Const `MaxCatalogSize`; sentinel `ErrCatalogTooLarge`
- Catalog chaining via nextCatalog; URN urn:publicid: support
- Files: `catalog.go`, `load.go`
- Imports: helium, internal/catalog/, internal/lexicon/

## stream/

Streaming XML writer (no DOM needed).

- **NewWriter(io.Writer, ...Option) → *Writer**
- Options: WithIndent(string), WithQuoteChar(byte)
- Methods: StartDocument/EndDocument, StartElement/EndElement, WriteAttribute, WriteString (escaped), WriteRaw (unescaped), WriteComment, WritePI, WriteCDATA, StartDTD/EndDTD, WriteDTDElement/Entity/Attlist/Notation, Flush
- State machine: tracks open elements, namespace scopes, self-close optimization
- Files: `stream.go` (single ~1100 line file)
- Imports: internal/encoding/, internal/xmlchar/

## sax/

SAX2 event-driven parsing interface definitions.

- `SAX2Handler` interface — callbacks: StartDocument, EndDocument, StartElement, EndElement, Characters, Comment, PI, CData, DTD events, entity/notation/element/attribute declarations
- `WithDocumentLocator(ctx, loc)` / `GetDocumentLocator(ctx)` — attach or read the current document locator on callback `context.Context`
- Files: `sax.go`
- Imports: helium (node types)

## push/

Generic push parser infrastructure shared by both XML and HTML push parsers.

- `Source[T]` interface — any parser with `ParseReader(ctx, io.Reader) (T, error)`
- `Parser[T]` struct — manages background goroutine, stream, Push/Write/Close
- `New[T](ctx, Source[T]) → *Parser[T]` — create and start a push parser
- Both `helium.PushParser` and `html.PushParser` are type aliases for `push.Parser[*helium.Document]`
- Files: `push.go`

## xmldsig1/

XML Digital Signatures 1.1 (W3C xmldsig-core1). Sign and verify XML documents.

- **NewSigner() → Signer** — create fluent builder for signing (clone-on-write value type)
  - `SignatureAlgorithm(uri)`, `CanonicalizationMethod(uri)`, `Reference(ReferenceConfig)`, `KeyInfo(KeyInfoBuilder)`, `SignatureID(id)` — builder methods
  - `SignEnveloped(ctx, doc, parent, key)`, `SignEnveloping(ctx, doc, content, key)`, `SignDetached(ctx, doc, key)` — terminal methods
- **NewVerifier(KeySource) → Verifier** — create fluent builder for verification
  - `Verify(ctx, doc)`, `VerifyElement(ctx, doc, sigElem)` — terminal methods
- **NewEnvelopedReference() → ReferenceConfig** — SAML-optimized defaults (enveloped + ExcC14N + SHA-256)
- Key sources: `StaticKey(key)`, `X509CertKeySource(cert)`, `KeySourceFunc`
- Key info builders: `X509DataKeyInfo(certs...)`, `RSAKeyValueKeyInfo()`
- Transforms: `Enveloped()`, `C14NTransform(uri)`, `ExcC14NTransform(prefixes...)`
- Algorithms: RSA-SHA1/SHA256, ECDSA-SHA256/SHA384, HMAC-SHA1/SHA256, Ed25519
- Digests: SHA-1, SHA-256, SHA-384, SHA-512
- Errors: `ErrNoKeySource` sentinel — returned by verify when no usable KeySource is configured (nil cfg, untyped-nil, or typed-nil KeySource/func)
- Files: `xmldsig1.go` (API), `constants.go`, `algorithms.go`, `sign.go`, `verify.go`, `transforms.go`, `keyinfo.go`, `errors.go`
- Imports: helium, c14n/

## xmlenc1/

XML Encryption 1.1 (W3C xmlenc-core1). Encrypt and decrypt XML elements/content.

- **NewEncryptor() → Encryptor** — create fluent builder for encryption (clone-on-write value type)
  - `BlockAlgorithm(uri)`, `AllowLegacyCBC(bool)`, `KeyTransportAlgorithm(uri)`, `RecipientPublicKey(key)`, `SessionKey(key)`, `KeyWrapAlgorithm(uri)`, `KeyEncryptionKey(kek)`, `OAEPDigest(uri)`, `OAEPMGF(uri)`, `OAEPParams(params)` — builder methods
  - `EncryptElement(ctx, elem)`, `EncryptContent(ctx, elem)` — terminal methods
- **NewDecryptor() → Decryptor** — create fluent builder for decryption
  - `PrivateKey(key)`, `KeyEncryptionKey(kek)`, `SessionKey(key)`, `AllowUnauthenticatedCBC(bool)` — builder methods
  - `Decrypt(ctx, elem)` — terminal method
- Block encryption: AES-128/256-CBC, AES-128/256-GCM
- Secure by default: unset `BlockAlgorithm` → `DefaultBlockAlgorithm` (AES-256-GCM). Selecting a CBC block algorithm for **encryption** requires `Encryptor.AllowLegacyCBC(true)`, else `ErrCBCEncryptionRequiresOptIn`. **Decryption** of CBC requires `Decryptor.AllowUnauthenticatedCBC(true)`, else `ErrCBCRequiresOptIn`
- Key transport: RSA-OAEP (1.0 + 1.1 with configurable digest/MGF)
- Key wrapping: AES-128/256-KeyWrap (RFC 3394)
- Key sizes are bound to the declared algorithm URI on encrypt and decrypt (incl. after unwrap/key transport); mismatch → `KeySizeError`
- Files: `xmlenc1.go` (API), `constants.go`, `block.go`, `keytransport.go`, `keywrap.go`, `types.go`, `serialize.go`, `parse.go`, `errors.go`
- Imports: helium

## shim/

Drop-in replacement for encoding/xml backed by helium parser.

- **NewDecoder(ctx, io.Reader) → *Decoder** / **NewTokenDecoder(ctx, TokenReader) → *Decoder** / **NewEncoder(io.Writer) → *Encoder**
- **Marshal(v) → ([]byte, error)** / **Unmarshal([]byte, v) → error**
- API mirrors encoding/xml; strict mode only; undeclared namespace prefixes rejected
- Known differences: empty elements self-closed, xmlns before regular attrs, InputOffset approximate
- Files: `decoder.go`, `encoder.go`, `marshal.go`, `unmarshal.go`, `types.go`, `namespace.go`, `escape.go`, `doc.go`
- Imports: helium, stream/, internal/encoding/, internal/xmlchar/

## sink/

Generic channel-based async event sink.

- **New[T](ctx, Handler[T], ...Option) → *Sink[T]** — nil handler is replaced with a no-op (delivery never panics)
- **Sink.Handle(ctx, T)** — async send (blocks if buffer full); re-entrant call from within a Handler is best-effort non-blocking
- **Sink.Close()** — drain and stop; self-close from within a Handler returns immediately (no deadlock)
- WithBufferSize(n) — default 256; negative values clamped to 0 (unbuffered)
- Nil-safe: Handle() on nil *Sink is no-op
- Re-entrancy-safe: a Handler may call Close or Handle on its own Sink without deadlock (worker-goroutine detection)
- When T=error, satisfies helium.ErrorHandler
- Files: `sink.go`
- Imports: none

## enum/

Shared enumeration package for DTD declaration symbols reused across packages.
Values match libxml2 C enums so helium, sax, and downstream packages can share
the same typed constants without redefining parallel enum sets.

- `AttributeType` — CDATA, ID, IDREF, IDREFS, ENTITY, ENTITIES, NMTOKEN, NMTOKENS, ENUMERATION, NOTATION
- `AttributeDefault` — REQUIRED, IMPLIED, FIXED
- `ElementType` — UNDEFINED, EMPTY, ANY, MIXED, ELEMENT
- `EntityType` — InternalGeneralEntity, ExternalGeneralParsedEntity, ExternalGeneralUnparsedEntity, InternalParameterEntity, ExternalParameterEntity, InternalPredefinedEntity
- Files: `enum.go`
- Imports: none

## test/

Shared test helper utilities and fixtures. Not a production package.

## internal/encoding/

Character encoding support wrapping golang.org/x/text/encoding.

## internal/catalog/

Internal OASIS XML Catalog model + resolution engine used by root parser + public `catalog/`.

- Types: `Catalog`, `Entry`, `EntryType`, `Prefer`, `Loader`, `Resolver`
- Helpers: `NormalizePublicID`, `UnwrapURN`, `ResolveURI`, `HasScheme`, `ParsePrefer`, `HasNextCatalog`
- Files: `catalog.go`, `resolve.go`, `normalize.go`, `uri.go`, `urn.go`
- Imports: none

## internal/lexicon/

Shared spec vocabulary strings reused across packages.

- Namespaces: XML Catalog, XSLT, XSD, XSI, XPath/XQuery function namespaces, XML, XMLNS
- XML vocabulary: common prefixes + attribute/value names such as `xml:base`
- Catalog vocabulary: OASIS catalog element names, attribute names, `prefer` values
- XSLT vocabulary: `XSLTElement*` constants for all XSLT element local names
- Files: `ns.go`, `xml.go`, `catalog.go`, `xslt.go`
- Imports: none

- **Load(name) → encoding.Encoding** — lookup by normalized name
- Supports: UTF-8/16/32, ISO-8859-*, Windows-*, KOI8-*, Mac, CJK, EBCDIC, UCS-4
- Files: `encoding.go`, `ebcdic.go`, `ucs4.go`, `c1fallback.go`

## internal/catalog/

Catalog internals: URN decoding (RFC 3151), normalization.

- **UnwrapURN(string) → string** — decode urn:publicid: to public ID
- Files: `urn.go`, `normalize.go`

## internal/icu/

ICU-style number format pattern parsing for `fn:format-number`.

- Files: `format_number.go`
- Imports: none

## internal/sequence/

Generic typed sequence utilities.

- Files: `sequence.go`
- Imports: none

## internal/strcursor/

String cursor for character-by-character parsing.

- Files: `strcursor.go`
- Imports: none

## internal/unparsedtext/

Shared resource loading for `fn:doc`, `fn:doc-available`, `fn:json-doc`, `fn:unparsed-text`, `fn:unparsed-text-available`, `fn:unparsed-text-lines`. Secure by default: with no `URIResolver` and no `HTTPClient`, every retrieval errors out (no implicit `http.DefaultClient`, no implicit `os.ReadFile`). Constructors: `NewHTTPResolver(*http.Client)`, `NewFileResolver(fs.FS)`; `FileURIResolver{BaseDir}` refuses `..` traversal outside `BaseDir`.

- Files: `unparsedtext.go`
- Imports: `internal/encoding`, `internal/lexicon`

## internal/xpathstream/

Streamability analysis helpers for XPath 3.1 expressions. Moved from xpath3 public API to reduce exported surface. Used by xslt3 streaming analysis.

- **WalkExpr(Expr, func(Expr) bool)** — AST walker
- **ExprHasDownwardStep / ExprUsesUpwardAxis / ExprUsesPrecedingAxis / ExprUsesDescendantOrSelf** — axis queries
- **ExprUsesFunction / ExprUsesContextItem / ExprHasUpThenDownNavigation** — expression property queries
- **PredicateIsNonMotionless / PredicateIsNonMotionlessWithStep / ExprTreeHasNonMotionlessPredicate** — predicate analysis
- **CountDownwardSelections** — downward selection counter
- Files: `xpathstream.go`
- Imports: xpath3

## internal/bitset/

Generic bitset operations for bitmask types.

- **Set[T](*T, T)** / **IsSet[T](T, T) → bool**
- Files: `bitset.go`

## internal/parser/

Parser option bitset type and constants. Bit positions match libxml2's XML_PARSE_* constants.

- **Option** — int-based bitset type for parser flags
- Constants: `Recover`, `NoEnt`, `DTDLoad`, `DTDAttr`, `DTDValid`, `NoError`, `NoWarning`, `Pedantic`, `NoBlanks`, `XInclude`, `NoNet`, `NoDict`, `NsClean`, `NoCDATA`, `NoXIncNode`, `Compact`, `NoBaseFix`, `Huge`, `IgnoreEnc`, `BigLines`, `NoXXE`, `NoUnzip`, `NoSysCatalog`, `CatalogPI`, `SkipIDs`, `LenientXMLDecl`
- Methods: `Set(Option)`, `Clear(Option)`, `IsSet(Option) → bool`
- Files: `options.go`
- Imports: internal/bitset

## internal/xmlchar/

XML 1.0 character classification and name validation. Single source of truth for the NCName/QName/Name productions, plus XML Char range, encoding-name, and PI-target validation shared across packages.

- **IsChar(rune) → bool** — XML 1.0 Char production (legal document character)
- **IsNCNameStartChar(rune) → bool** — XML 1.0 NCName start character production
- **IsNCNameChar(rune) → bool** — XML 1.0 NCName continuation character production
- **IsValidNCName(string) → bool** — validates a complete NCName string
- **IsValidQName(string) → bool** — validates a complete QName (NCName, optionally prefixed)
- **IsValidName(string) → bool** — validates a complete XML Name (NCName allowing colons)
- **IsValidEncName(string) → bool** — validates an XML declaration encoding name
- **IsValidPITarget(string) → bool** — validates a processing-instruction target
- Files: `xmlchar.go`
- Imports: none

## internal/xsd/value/

XSD builtin value validation and comparison, extracted from `xsd/`.

- **ValidateBuiltin(value, builtinLocal string) error** — validate value against XSD builtin type lexical space
- **Compare(a, b, builtinLocal string) (int, bool)** — type-aware comparison (-1/0/+1, ok)
- **CompareDecimal(a, b string) int** — decimal comparison via math/big.Rat (-2 on error)
- Files: `validate.go`, `compare.go`
- Imports: `internal/lexicon` (XSD builtin type-name constants)

## internal/stack/

Generic stack with capacity shrinking.

- **stackPop(StackImpl, n)** — pop n items and shrink if oversized
- Files: `stack.go`

## internal/heliumtest/

Test helpers shared across helium packages.

- `CallerDir(skip)` — directory of caller's source file
- `RepoRoot()` — absolute path to repository root (cached)
- `TestDir(path...)` — join path elements under repo root
- Files: `callerdir.go`

## internal/cliutil/

Platform-specific TTY handling for CLI commands.

- Files: `tty_posix.go`, `tty_windows.go`, `tty_bsd.go`

## internal/cli/heliumcmd/

Importable implementation behind `helium` CLI. Used by `cmd/helium` wrapper and executable examples.

- Entry points: `Execute(ctx, args)`, context mutators `WithIO(ctx, stdin, stdout, stderr)`, `WithStdinTTY(ctx, bool)`
- Subcommands: `lint`, `xpath`, `xsd validate`, `relaxng validate`, `schematron validate`, `xslt`
- Context behavior: when stdio carriers are absent, defaults to `os.Stdin`, `os.Stdout`, `os.Stderr`, and TTY detection from `os.Stdin`
- Lint behavior: parse args, detect stdin/TTY, process XML, run XInclude/XSD/XPath/C14N, emit xmllint-style exit codes
- XPath behavior: mandatory positional expr, default engine `3`, `--engine 1|3`, XML from file args or stdin, type-aware result output for xpath1/xpath3
- RELAX NG behavior: compile grammar from mandatory positional schema path, parse XML input(s), validate via `relaxng.NewValidator().Validate`, return schema/validation exit codes
- Schematron behavior: compile schema from mandatory positional schema path, parse XML input(s), validate via `schematron.NewValidator(schema).Validate`, return schema/validation exit codes
- XSD behavior: compile schema from mandatory positional schema path, parse XML input(s), validate via `xsd.NewValidator(schema).Validate`, return schema/validation exit codes
- XSLT behavior: compile stylesheet from mandatory positional path, parse XML input(s), transform via `ss.Transform(doc).WriteTo`, supports `--param`/`--stringparam`/`--output`/`--noout`
- Files: `cli.go`, `exitcode.go`, `lint.go`, `xpath.go`, `relaxng_validate.go`, `schematron_validate.go`, `xsd_validate.go`, `xslt.go`
- Imports: helium, c14n/, relaxng/, schematron/, xsd/, xslt3/, xinclude/, xpath1/, xpath3/, catalog/, internal/cliutil/

## cmd/helium/

Thin executable wrapper around `internal/cli/heliumcmd`.

- Main behavior: `main()` → `os.Exit(heliumcmd.Execute(context.Background(), os.Args[1:]))`
- User docs: `README.md`
- Files: `main.go`
- Imports: internal/cli/heliumcmd/
