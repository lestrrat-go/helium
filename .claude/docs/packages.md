# Package Map

Go implementation of libxml2. Module: `github.com/lestrrat-go/helium`

## Root (`helium`)

XML parsing, DOM tree, serialization. Entry point for all XML processing.

- **Parse(ctx, []byte) → *Document** / **ParseReader(ctx, io.Reader) → *Document** — main parse entry points
- **NewParser()** — configurable parser with SetOption(), SetSAXHandler(), SetCatalog(), SetBaseURI(), SetMaxDepth()
- **NewWriter(opts) → *Writer** — XML serializer; Writer.WriteDoc(io.Writer, *Document), Writer.WriteNode(io.Writer, Node)
- **ParseInNodeContext(ctx, Node, []byte) → Node** — parse fragment in existing node context
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

- **Canonicalize(io.Writer, *Document, Mode, ...Option)** — write canonical form
- **CanonicalizeTo(*Document, Mode, ...Option) → []byte** — return canonical form
- Options: WithComments(), WithNodeSet([]Node), WithInclusiveNamespaces([]string), WithBaseURI(string)
- Files: `c14n.go` (API), `canonicalizer.go` (engine), `nsstack.go`, `sort.go`, `escape.go`
- Imports: helium

## xpath/

XPath 1.0 expression parsing and evaluation.

- **Compile(string) → *Expression** / **MustCompile(string) → *Expression** — parse XPath
- **Expression.Evaluate(Node) → *Result** / **EvaluateWith(Node, *Context) → *Result**
- **Find(Node, string) → []Node** — convenience: compile+evaluate→node-set
- **Evaluate(Node, string) → *Result** — convenience: compile+evaluate
- `Result` types: NodeSetResult, BooleanResult, NumberResult, StringResult
- `Context` — namespace bindings, variables, custom functions, op limits
- `NewContext(opts)` with WithNamespaces(), WithVariables(), WithOpLimit()
- `Context.RegisterFunction(name, fn)` / `RegisterFunctionNS(uri, name, fn)` — custom functions (rejects built-in override)
- Limits: recursion 5000, node-set 10M, configurable op limit
- Files: `xpath.go` (API), `parser.go`, `eval.go`, `axes.go`, `token.go`, `number.go`
- Imports: helium

## xsd/

XML Schema (XSD) 1.0 compilation and validation.

- **Compile(*Document, ...CompileOption) → *Schema** / **CompileFile(path, ...CompileOption) → *Schema**
- **Validate(*Document, *Schema, ...ValidateOption) → error**
- `Schema.LookupElement(local, ns)`, `Schema.LookupType(local, ns)`, `Schema.TargetNamespace()`
- Supports: complex/simple types, sequences, choices, all, groups, attribute groups, substitution groups, import/include, IDC (xs:unique/key/keyref)
- `ValidateError.Output` — libxml2-compatible error string
- Files: `xsd.go` (API), `schema.go` (data model), `parse.go` (compiler), `parse_check.go` (UPA/constraints), `validate.go`, `validate_elem.go`, `validate_value.go`, `validate_idc.go`, `errors.go`
- Imports: helium, xpath/
- Status: 225/226 golden tests passing

## relaxng/

RELAX NG schema compilation and validation.

- **Compile(*Document, ...CompileOption) → *Grammar** / **CompileFile(path, ...CompileOption) → *Grammar**
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

- **Parse(ctx, []byte, ...ParseOption) → *Document**
- **ParseFile(ctx, path, ...ParseOption) → *Document**
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
- Imports: helium, xpath/

## schematron/

Schematron schema compilation and validation.

- **Compile(*Document, ...CompileOption) → *Schema** / **CompileFile(path, ...CompileOption) → *Schema**
- **Validate(*Document, *Schema, ...ValidateOption) → error**
- Supports: schema, pattern, rule, assert, report, let, name, value-of
- Variable bindings via `<let>` and `<param>`
- Files: `schematron.go` (API), `schema.go`, `parse.go`, `validate.go`, `errors.go`
- Imports: helium, xpath/

## catalog/

OASIS XML Catalog resolution for public/system IDs and URIs.

- **Load(path, ...LoadOption) → (*Catalog, error)**
- **Catalog.Resolve(pubID, sysID) → string** — resolve external identifier
- **Catalog.ResolveURI(uri) → string** — resolve URI reference
- Catalog chaining via nextCatalog; URN urn:publicid: support
- Files: `catalog.go`, `load.go`
- Imports: helium, internal/catalog/

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
- `Context` interface — parser state passed to handlers; GetLocator() for line/col
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

- **Load(name) → encoding.Encoding** — lookup by normalized name
- Supports: UTF-8/16/32, ISO-8859-*, Windows-*, KOI8-*, Mac, CJK, EBCDIC, UCS-4
- Files: `encoding.go`, `ebcdic.go`, `ucs4.go`, `c1fallback.go`

## internal/catalog/

Catalog internals: URN decoding (RFC 3151), normalization.

- **UnwrapURN(string) → string** — decode urn:publicid: to public ID
- Files: `urn.go`, `normalize.go`

## internal/bitset/

Generic bitset operations for bitmask types.

- **Set[T](*T, T)** / **IsSet[T](T, T) → bool**
- Files: `bitset.go`

## internal/stack/

Generic stack with capacity shrinking.

- **stackPop(StackImpl, n)** — pop n items and shrink if oversized
- Files: `stack.go`

## internal/cliutil/

Platform-specific TTY handling for heliumlint.

- Files: `tty_posix.go`, `tty_windows.go`, `tty_bsd.go`

## cmd/heliumlint/

CLI tool for parsing, validating, processing XML with libxml2-compatible flags.

- Flags: --recover, --noent, --loaddtd, --valid, --noblanks, --noout, --format, --c14n, --c14n11, --exc-c14n, --xinclude, --schema, --xpath, --catalogs, etc.
- Files: `heliumlint.go`
- Imports: helium, c14n/, xsd/, xinclude/, xpath/, catalog/, internal/cliutil/
