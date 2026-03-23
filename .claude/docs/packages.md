# Package Map

Go implementation of libxml2. Module: `github.com/lestrrat-go/helium`

## Root (`helium`)

XML parsing, DOM tree, serialization. Entry point for all XML processing.

- **Parse(ctx, []byte) → (*Document, error)** / **ParseReader(ctx, io.Reader) → (*Document, error)** — main parse entry points
- **NewParser()** — configurable parser with SetOption(), SetSAXHandler(), SetCatalog(), SetBaseURI(), SetMaxDepth()
- **NewWriter(opts) → *Writer** — XML serializer; Writer.WriteDoc(io.Writer, *Document), Writer.WriteNode(io.Writer, Node)
- **ParseInNodeContext(ctx, Node, []byte) → (Node, error)** — parse fragment in existing node context
- **Element.FindAttribute(AttributePredicate) → (*Attribute, bool)** — attribute-node lookup by matcher; built-in matchers: `QNamePredicate`, `LocalNamePredicate`, `NSPredicate`
- **Element.GetAttribute(qname) → (string, bool)** / **Element.GetAttributeNS(local, nsURI) → (string, bool)** — attribute value lookup by QName or expanded name
- Key types: `Document`, `Element`, `Attribute`, `Namespace`, `DTD`, `Entity`, `Text`, `CDATASection`, `Comment`, `PI`
- `Node` interface — common for all node types; use ElementType enum to distinguish
- `ParseOption` bitmask — ParseNoEnt, ParseDTDLoad, ParseDTDAttr, ParseDTDValid, ParseNoBlanks, ParseXInclude, ParseNoXXE, ParseRecover, ParseLenientXMLDecl, etc.
- `ErrorHandler` interface — async error delivery during parsing
- `Document.GetElementByID(id)` — O(1) via hash table, O(n) fallback
- `Walk(doc, fn)`, `Children(node)`, `Descendants(node)` — tree traversal
- `CopyNode(src, targetDoc)` — deep copy across documents
- `NodeGetBase(doc, node)` — effective xml:base URI
- `BuildURI(base, ref)` — resolve relative URI
- Files: `parser.go`, `parserctx.go`, `document.go`, `element.go`, `attr.go`, `node.go`, `namespace.go`, `dump.go`, `copy.go`, `dtd.go`, `iter.go`, `base.go`, `errorhandler.go`

## c14n/

W3C Canonical XML. 3 modes: C14N10, ExclusiveC14N10, C14N11.

- **Canonicalize(io.Writer, *Document, Mode, ...Option) → error** — write canonical form
- **CanonicalizeTo(*Document, Mode, ...Option) → ([]byte, error)** — return canonical form
- Options: WithComments(), WithNodeSet([]Node), WithInclusiveNamespaces([]string), WithBaseURI(string)
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
- Files: `xpath.go` (API), `parser.go`, `lexer.go`, `eval.go`, `expr.go`, `axes.go`, `functions.go`, `token.go`
- Imports: helium

## xpath3/

XPath 3.1 expression parsing and evaluation.

- **Compile(string) → (*Expression, error)** / **MustCompile(string) → *Expression** — parse XPath 3.1
- **Expression.Evaluate(ctx, Node) → (*Result, error)**
- **Expression.DumpVM(io.Writer) → error** — write compiled VM instruction dump for debugging/tooling
- **Find(ctx, Node, string) → ([]Node, error)** — convenience: compile+evaluate→node-set
- **Evaluate(ctx, Node, string) → (*Result, error)** — convenience: compile+evaluate
- **WithNamespaces(ctx, ns) → context.Context** / **WithVariables(ctx, vars) → context.Context** / **WithOpLimit(ctx, n) → context.Context** — attach XPath 3.1 evaluation settings to `context.Context`
- **WithFunction(ctx, name, fn) → context.Context** / **WithFunctionNS(ctx, uri, name, fn) → context.Context** — register custom functions on `context.Context`
- **WithFunctions(ctx, fns) → context.Context** / **WithFunctionsNS(ctx, fns) → context.Context** — bulk function registration
- `Result` — wraps `Sequence`; methods: `Nodes()`, `IsBoolean()`, `IsNumber()`, `IsString()`, `IsAtomic()`, `Atomics()`, `Sequence()`
- Merge helpers: `WithAdditionalNamespaces(ctx, ns)`, `WithAdditionalVariables(ctx, vars)`
- Direct mutators also include `WithDefaultLanguage(ctx, lang)`, `WithDefaultCollation(ctx, uri)`, `WithDefaultDecimalFormat(ctx, df)`, `WithNamedDecimalFormats(ctx, dfs)`, `WithBaseURI(ctx, uri)`, `WithURIResolver(ctx, r)`, `WithCollectionResolver(ctx, r)`, `WithHTTPClient(ctx, client)`, `WithImplicitTimezone(ctx, loc)`
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

- **CompileStylesheet(ctx, *Document) → (*Stylesheet, error)** / **CompileFile(ctx, path) → (*Stylesheet, error)** — compile parsed or file-backed stylesheet
- **Transform(ctx, *Document, *Stylesheet) → (*Document, error)** — execute stylesheet, return result DOM; returns error (not panic) if ss is nil
- **TransformToWriter(ctx, *Document, *Stylesheet, io.Writer) → error** / **TransformString(ctx, *Document, *Stylesheet) → (string, error)** — serialize transformation result; returns error (not panic) if ss is nil
- Compile context options: `WithCompileBaseURI(ctx, uri)`, `WithCompileURIResolver(ctx, resolver)`
- Transform context options: `WithParameter(ctx, name, value)`, `WithInitialTemplate(ctx, name)`, `WithMessageHandler(ctx, fn)`
- Key types: `Stylesheet`, `Template`, `Variable`, `Param`, `KeyDef`, `OutputDef`, `URIResolver`
- Supports: `xsl:template`, `xsl:apply-templates`, `xsl:call-template`, `xsl:param`/`xsl:variable`, `xsl:include`/`xsl:import`, `xsl:sort`, `xsl:number`, `xsl:message`, `xsl:key`, `xsl:output`, `xsl:import-schema`, `xsl:function`, literal result elements, AVTs, `xsl:attribute-set`, `xsl:map`/`xsl:map-entry`, `xsl:source-document`, `xsl:iterate`, `xsl:fork`, `xsl:accumulator`, `xsl:merge`, `xsl:where-populated`, `xsl:try`/`xsl:catch`, `xsl:for-each-group`, `xsl:result-document`, `xsl:next-match`, `xsl:apply-imports`
- Schema awareness: `xsl:import-schema` compiles XSD schemas, `type=` on `xsl:element`/`xsl:attribute` annotates result nodes, `validation=` on `xsl:copy`/`xsl:copy-of`, `default-validation` stylesheet attribute, `type-available()` function, type annotations flow to xpath3 via `WithTypeAnnotations`
- Streaming: `xsl:source-document` (DOM-materialization), `xsl:iterate`/`xsl:break`/`xsl:next-iteration`/`xsl:on-completion`, `xsl:fork`, `xsl:accumulator`/`xsl:accumulator-rule`, `xsl:merge`/`xsl:merge-source`/`xsl:merge-key`/`xsl:merge-action`; streamability analysis (XTSE3430) post-compilation pass
- Runtime helpers: `current()`, `document()`, `key()`, `generate-id()`, `system-property()`, `unparsed-entity-uri()`, `unparsed-entity-public-id()`, `type-available()`, `snapshot()`, `copy-of()`, `accumulator-before()`/`accumulator-after()`, `current-merge-group()`/`current-merge-key()`
- Output methods: `xml`, `html`, `text`
- Files: `xslt3.go` (API), `options.go` (context config), `compile.go` (orchestration + shared helpers), `compiler.go`, `compile_imports.go`, `compile_packages.go`, `compile_schema.go`, `compile_templates.go`, `compile_functions_modes.go`, `compile_formats.go`, `compile_instructions.go` (dispatch + static context helpers), `compile_instructions_flow.go`, `compile_instructions_nodes.go`, `compile_instructions_vars.go`, `compile_patterns.go`, `compile_streaming.go`, `execute.go`, `execute_apply.go`, `execute_control.go`, `execute_copy.go`, `execute_element.go`, `execute_instructions.go`, `execute_lre.go`, `execute_misc.go`, `execute_number.go`, `execute_resultdoc.go`, `execute_sequence.go`, `execute_sort.go`, `execute_streaming.go`, `execute_trycatch.go`, `execute_variable.go`, `output.go`, `functions.go`, `keys.go`, `avt.go`, `stylesheet.go`, `stylesheet_entry.go`, `instruction.go`, `invocation.go`, `sort.go`, `types.go`, `parameters.go`, `receiver.go`, `number_words.go`, `source_schema.go`, `schema_constructors.go`, `schema_context.go`, `package_override.go`, `package_version.go`, `package_visibility.go`, `streamability_analysis.go`, `elements.go`, `errors.go`
- Imports: helium, xpath3, xsd, html, internal/lexicon, internal/sequence
- Tests: `xslt3_test.go` runs phase-1 W3C XSLT 3.0 test sets from `testdata/xslt30/source/` when fetched

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

- **Compile(ctx, *Document, ...CompileOption) → (*Schema, error)** / **CompileFile(ctx, path, ...CompileOption) → (*Schema, error)**
- **Validate(*Document, *Schema, ...ValidateOption) → error**
- **ValidateSimpleValue(value, *TypeDef) → error** — validate a lexical value against a compiled simple type definition
- `Schema.LookupElement(local, ns)`, `Schema.LookupType(local, ns)`, `Schema.NamedTypes()`, `Schema.TargetNamespace()`
- Supports: complex/simple types, sequences, choices, all, groups, attribute groups, substitution groups, import/include, IDC (xs:unique/key/keyref)
- `ValidateError.Output` — libxml2-compatible error string
- Files: `xsd.go` (API), `schema.go` (data model), `compile.go` + `compile_imports.go` + `compile_helpers.go` (compile orchestration/imports/helpers), `read_types.go` + `read_particles.go` + `read_elements.go` + `read_decl_helpers.go` (schema readers), `link_refs.go` + `check_*.go` (reference resolution + constraints), `validate_context.go` + `validate.go` + `validate_elem.go` + `validate_idc.go` (validation flow/content/IDC), `simplevalue_*.go` + `validate_value_api.go` (simple-value engine/API), `errors.go`
- Imports: helium, xpath1/, internal/lexicon
- Status: 225/226 golden tests passing

## relaxng/

RELAX NG schema compilation and validation.

- **Compile(ctx, *Document, ...CompileOption) → (*Grammar, error)** / **CompileFile(ctx, path, ...CompileOption) → (*Grammar, error)**
- **Validate(*Document, *Grammar, ...ValidateOption) → error**
- Pattern-based: element, attribute, group, choice, interleave, optional, zeroOrMore, oneOrMore, ref, data, value, list, mixed, notAllowed
- Supports: include with override, externalRef, parentRef, anyName/nsName/ncName, data types
- Group backtracking for greedy pattern over-consumption
- `ValidateError.Output` — libxml2-compatible error string
- Files: `relaxng.go` (API), `grammar.go` (data model), `parse.go` (compiler), `validate.go` (engine), `errors.go`
- Imports: helium
- Status: 159/159 golden tests passing

## html/

HTML 4.01 parser producing helium DOM or SAX events.

- **Parse(ctx, []byte, ...ParseOption) → (*Document, error)**
- **ParseFile(ctx, path, ...ParseOption) → (*Document, error)**
- **ParseWithSAX(ctx, []byte, SAXHandler, ...ParseOption) → error**
- Auto-closing, void elements, implicit html/head/body insertion
- Encoding: prescan charset=utf-8 → U+FFFD for invalid bytes; otherwise Latin-1/Win-1252→UTF-8
- Entity resolution: 2125 WHATWG + 106 legacy HTML4; legacy entities work without `;`
- Files: `html.go` (API), `parser.go`, `entities.go`, `elements.go`, `dump.go` (serializer), `tree.go` (DOM builder), `sax.go`
- Imports: helium, sax/

## xinclude/

XInclude 1.0 processing with recursive inclusion and fallback.

- **Process(*Document, ...Option) → (int, error)** — process entire document
- **ProcessTree(Node, ...Option) → (int, error)** — process subtree
- Options: WithNoXIncludeMarkers(), WithNoBaseFixup(), WithResolver(Resolver), WithBaseURI(string), WithParseFlags(), WithWarningHandler()
- `Resolver` interface — custom resource loader
- Max depth 40, max URI 2000 chars, circular detection, doc/text caching
- Files: `xinclude.go`
- Imports: helium, xpointer/, internal/encoding/

## xpointer/

XPointer expression evaluation with scheme cascading.

- **Evaluate(*Document, string) → ([]Node, error)**
- Schemes: xpointer(), xpath1() → XPath; element(/1/2/3) → child-sequence; xmlns() → ns binding; shorthand → ID lookup
- Multiple scheme parts left-to-right; first non-empty result wins
- Files: `xpointer.go`
- Imports: helium, xpath1/

## schematron/

Schematron schema compilation and validation.

- **Compile(ctx, *Document, ...CompileOption) → (*Schema, error)** / **CompileFile(ctx, path, ...CompileOption) → (*Schema, error)**
- **Validate(*Document, *Schema, ...ValidateOption) → error**
- Supports: schema, pattern, rule, assert, report, let, name, value-of
- Variable bindings via `<let>` and `<param>`
- Files: `schematron.go` (API), `schema.go`, `parse.go`, `validate.go`, `errors.go`
- Imports: helium, xpath1/

## catalog/

OASIS XML Catalog resolution for public/system IDs and URIs.

- **Load(ctx, path, ...LoadOption) → (*Catalog, error)**
- **Catalog.Resolve(pubID, sysID) → string** — resolve external identifier
- **Catalog.ResolveURI(uri) → string** — resolve URI reference
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
- Imports: internal/encoding/

## sax/

SAX2 event-driven parsing interface definitions.

- `SAX2Handler` interface — callbacks: StartDocument, EndDocument, StartElement, EndElement, Characters, Comment, PI, CData, DTD events, entity/notation/element/attribute declarations
- `WithDocumentLocator(ctx, loc)` / `GetDocumentLocator(ctx)` — attach or read the current document locator on callback `context.Context`
- Files: `sax.go`
- Imports: helium (node types)

## shim/

Drop-in replacement for encoding/xml backed by helium parser.

- **NewDecoder(io.Reader) → *Decoder** / **NewEncoder(io.Writer) → *Encoder**
- **Marshal(v) → ([]byte, error)** / **Unmarshal([]byte, v) → error**
- API mirrors encoding/xml; strict mode only; undeclared namespace prefixes rejected
- Known differences: empty elements self-closed, xmlns before regular attrs, InputOffset approximate
- Files: `decoder.go`, `encoder.go`, `marshal.go`, `unmarshal.go`, `types.go`, `namespace.go`, `escape.go`, `doc.go`
- Imports: helium, stream/, internal/encoding/

## sink/

Generic channel-based async event sink.

- **New[T](ctx, Handler[T], ...Option) → *Sink[T]**
- **Sink.Handle(ctx, T)** — async send (blocks if buffer full)
- **Sink.Close()** — drain and stop
- WithBufferSize(n) — default 256
- Nil-safe: Handle() on nil *Sink is no-op
- When T=error, satisfies helium.ErrorHandler
- Files: `sink.go`
- Imports: none

## enum/

DTD declaration enumerations matching libxml2 C enums.

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

Shared `fn:unparsed-text` / `fn:unparsed-text-lines` resource loading.

- Files: `unparsedtext.go`
- Imports: none

## internal/bitset/

Generic bitset operations for bitmask types.

- **Set[T](*T, T)** / **IsSet[T](T, T) → bool**
- Files: `bitset.go`

## internal/stack/

Generic stack with capacity shrinking.

- **stackPop(StackImpl, n)** — pop n items and shrink if oversized
- Files: `stack.go`

## internal/cliutil/

Platform-specific TTY handling for CLI commands.

- Files: `tty_posix.go`, `tty_windows.go`, `tty_bsd.go`

## internal/cli/heliumcmd/

Importable implementation behind `helium` CLI. Used by `cmd/helium` wrapper and executable examples.

- Entry points: `Execute(ctx, args)`, context mutators `WithIO(ctx, stdin, stdout, stderr)`, `WithStdinTTY(ctx, bool)`
- Subcommands: `lint`, `xpath`, `xsd validate`, `relaxng validate`, `schematron validate`
- Context behavior: when stdio carriers are absent, defaults to `os.Stdin`, `os.Stdout`, `os.Stderr`, and TTY detection from `os.Stdin`
- Lint behavior: parse args, detect stdin/TTY, process XML, run XInclude/XSD/XPath/C14N, emit xmllint-style exit codes
- XPath behavior: mandatory positional expr, default engine `3`, `--engine 1|3`, XML from file args or stdin, type-aware result output for xpath1/xpath3
- RELAX NG behavior: compile grammar from mandatory positional schema path, parse XML input(s), validate via `relaxng.Validate`, return schema/validation exit codes
- Schematron behavior: compile schema from mandatory positional schema path, parse XML input(s), validate via `schematron.Validate`, return schema/validation exit codes
- XSD behavior: compile schema from mandatory positional schema path, parse XML input(s), validate via `xsd.Validate`, return schema/validation exit codes
- Files: `cli.go`, `exitcode.go`, `lint.go`, `xpath.go`, `relaxng_validate.go`, `schematron_validate.go`, `xsd_validate.go`
- Imports: helium, c14n/, relaxng/, schematron/, xsd/, xinclude/, xpath1/, xpath3/, catalog/, internal/cliutil/

## cmd/helium/

Thin executable wrapper around `internal/cli/heliumcmd`.

- Main behavior: `main()` → `os.Exit(heliumcmd.Execute(context.Background(), os.Args[1:]))`
- Files: `main.go`
- Imports: internal/cli/heliumcmd/
