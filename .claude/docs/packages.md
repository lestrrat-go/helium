# Package Map

Go implementation of libxml2. Module: `github.com/lestrrat-go/helium`

## Root (`helium`)

XML parsing, DOM tree, serialization. Entry point for all XML processing.

- **NewParser() → Parser** — create fluent builder for XML parsing (clone-on-write value type). **Secure by default** for untrusted input: `BlockXXE` on (no external entity/DTD loading), `AllowNetwork` off, `FS` is a deny-all FS (`internal/iofs.DenyAll` — opens nothing), and element depth is capped at 256 (`MaxDepth`). Entity substitution, external-DTD loading, XInclude, and DTD validation are all off by default. Opt back in explicitly, e.g. `NewParser().BlockXXE(false).LoadExternalDTD(true).FS(helium.PermissiveFS())`.
  - Flag methods: `RecoverOnError(bool)`, `SubstituteEntities(bool)`, `LoadExternalDTD(bool)`, `DefaultDTDAttributes(bool)`, `ValidateDTD(bool)`, `SuppressErrors(bool)`, `SuppressWarnings(bool)`, `PedanticErrors(bool)`, `StripBlanks(bool)`, `AllowNetwork(bool)` (default false), `CleanNamespaces(bool)`, `MergeCDATA(bool)`, `FixBaseURIs(bool)`, `IgnoreEncoding(bool)`, `BlockXXE(bool)` (default true), `SkipIDs(bool)`, `LenientXMLDecl(bool)`
  - Per-limit knobs (each takes an int, `0` = default, negative = no limit): `MaxNameLength(int)` (default `DefaultMaxNameLength` 50000), `MaxEntityAmplification(int)` (default `DefaultMaxEntityAmplification` 5; 1 GiB hard ceiling always applies), `MaxContentModelDepth(int)` (default `DefaultMaxContentModelDepth` 128), `MaxNodeContentSize(int)` (default `DefaultMaxNodeContentSize` 10 MiB) — caps a single indivisible content run (CDATA section / comment body / PI body / char-data run / attribute value); the SAME cap also bounds a single contiguous run of XML whitespace (a blank skip — prolog/epilogue/inter-root, and the blank skips inside the external DTD subset and INCLUDE sections), so an unbounded whitespace run cannot grow the cursor buffer; over-cap → `ErrNodeContentTooLarge`, fired during accumulation. A negative value (`MaxNodeContentSize(-1)`) disables BOTH the node-content and the blank-run cap. A streaming SAX consumer with `CharBufferSize > 0` receives char data in bounded chunks and is exempt from the char-data cap (its CDATA/comment/PI runs are still capped)
  - Config methods: `SAXHandler(sax.SAX2Handler)`, `BaseURI(string)`, `CharBufferSize(int)`, `MaxDepth(int)` (default 256; 0 = unlimited), `MaxExternalDTDBytes(int)`, `Catalog(CatalogResolver)`, `FS(fs.FS)`, `ErrorHandler(ErrorHandler)`, `XInclude(XIncludeProcessor)`
  - `Parser.XInclude(XIncludeProcessor)` — inject an XInclude processor (the `XIncludeProcessor` interface: `Process(context.Context, *Document) (int, error)`, satisfied by `xinclude.Processor`). When set, `Parse`/`ParseReader`/`ParseFile` run it over the built tree so the returned document has `xi:include` elements expanded; if `ValidateDTD` is also set, the expanded tree is validated. Off by default (nil disables). The interface is the dependency-inversion seam: the root package cannot import `xinclude` (which imports `helium`), so the caller builds and injects a configured processor
  - **PermissiveFS() → fs.FS** — returns `internal/iofs.PermissiveRoot` (opens any path via `os.Open`), the public escape hatch for restoring host-filesystem access that `NewParser` does not grant by default
  - `Parser.FS(fs.FS)` — sets the `fs.FS` used to load external resources referenced by the document: external DTD subsets (`LoadExternalDTD`) and external general entities resolved through `TreeBuilder.ResolveEntity`. The default (and what a nil value restores) is `internal/iofs.DenyAll`, which refuses every open. Pass `helium.PermissiveFS()` (any `os.Open` path) or a confined FS to enable loading; prefer `os.Root.FS` for a confined FS — it refuses a symlink escape, whereas `os.DirFS` follows an in-root symlink out of its root (path-escape-safe, not a symlink sandbox). A relative SYSTEM id is resolved against the document's base URI (absolute whenever set — `ParseFile` uses the file's absolute path); the direct entity/DTD paths hand the FS the raw resolved name first (a `file:` URI verbatim, so `PermissiveRoot`/os.Open and a file-URI-keyed FS are unchanged) and, on an `fs.ErrInvalid` rejection from an `fs.ValidPath`-enforcing FS (`os.DirFS`/`os.Root.FS`/`fs.Sub`), retry with the name made relative to the **fixed top-level document base**'s directory (`parserCtx.documentBaseURI`, captured once at parse start; `openExternalResource`/`baseRelativeFSName` in `tree_builder.go`), so a confined FS rooted at the document's directory resolves the reference — including a nested resource in a subdirectory, which relativizes against the document root, not its own moving base. The retry fires **only for an originally-relative reference** (`systemIDRetryEligible` on the SYSTEM id as declared, before URI resolution / catalog mapping): ineligible when the id is absolute, carries a URI scheme, or has a colon anywhere in its first path segment (catching a one-letter RFC 3986 scheme like `x:opaque` and a bare drive letter), so an absolute, `file:`-URI, or otherwise scheme-carrying SYSTEM id is never retried, even one naming an in-root file. The supported confined-FS document base is an **absolute path or `file:` URI**; a relative document base is out of scope (`BuildURI` yields a valid-but-absent relative path that fails with `fs.ErrNotExist`, not `fs.ErrInvalid`, so the retry never fires — the deferred `helium.DirFS(root)` adapter is the general fix). The retry is a validated `fs.ValidPath` (leading `/` or surviving `..` disqualifies it → blocks `../`/absolute path escape, but does NOT confine symlinks). `openExternalResource` is the single network-guard enforcement point: a network-scheme name is refused (`ErrNetworkAccessForbidden`) before **either** open — primary or retry
  - `Parser.ErrorHandler(ErrorHandler)` — sets the handler for validation errors produced during DTD validation (`ValidateDTD`); individual errors are delivered as they occur and `Parse` returns `ErrDTDValidationFailed` on failure. If the handler is an `io.Closer`, it is closed only after the DTD validation pass runs (i.e. when `ValidateDTD` is enabled and the document was parsed); it is not auto-closed for non-validating parses or for parse errors that abort before validation
  - Terminal methods: `Parse(ctx, []byte) → (*Document, error)`, `ParseReader(ctx, io.Reader) → (*Document, error)`, `ParseFile(ctx, string) → (*Document, error)`, `ParseInNodeContext(ctx, Node, []byte) → (Node, error)`, `NewPushParser(ctx) → *PushParser`
- **NewWriter() → Writer** — create fluent XML writer builder
  - Writer methods: `Format(bool)`, `IndentString(string)`, `SelfCloseEmptyElements(bool)`, `XMLDeclaration(bool)`, `IncludeDTD(bool)`, `EscapeNonASCII(bool)`, `AllowPrefixUndeclarations(bool)`, `RejectInvalidChars(bool)` (fail with `ErrInvalidXMLChar` — the XSLT SERE0006 error — instead of replacing an XML-invalid char with U+FFFD; folded into the escape pass, no extra traversal), plus the serialization-parameter knobs `OutputVersion(string)` (override the effective output XML version — empty keeps the document's version, byte-identical — driving BOTH the declaration text AND the XML 1.1 escaping/undeclaration rules; the effective version must be a valid `VersionNum` `'1.' [0-9]+`, else serialization fails with `ErrInvalidOutputVersion` before any output byte), `OutputEncoding(string)` (override the encoding pseudo-attribute of the XML declaration — empty keeps the document's own encoding, byte-identical; the encoding pseudo-attribute is omitted only when the resulting effective encoding is empty. A non-empty effective encoding must be a well-formed `EncName`, else serialization fails with `ErrUnsupportedOutputEncoding` before any output byte (validated ahead of the transcoding encoder, so no BOM leaks). Affects the **Document serialization path only**: a bare element/fragment carries no declaration, so `asciiOutput` stays off and fragment bytes are byte-identical to output without an override. When set on a `WriteTo`-to-io.Writer Document path the emitted octets are RE-ENCODED to the effective encoding so the bytes agree with the declaration, and an effective encoding the writer cannot emit — one the internal encoder table can't load and that is not UTF-8/US-ASCII — fails with `ErrUnsupportedOutputEncoding`; this hard error is scoped to the override, a document's own unloadable parsed encoding stays declaration-only. An explicit US-ASCII override — matched by ANY IANA alias via `internal/encoding.IsASCII` (`us-ascii`, `ascii`, `csASCII`, `ANSI_X3.4-1968`, `ANSI_X3.4-1986`, `iso-ir-6`, `ISO646-US`, `us`, `IBM367`, `cp367`) — installs no encoder but escapes EVERY non-ASCII character — the full Unicode range, not just the Latin-1 that `EscapeNonASCII` covers — as a numeric character reference (`&#xNNNN;`), so the octets are pure US-ASCII and agree with the declaration. Text and attribute values character-reference fine; ANY other non-ASCII byte reaching the output has no faithful US-ASCII serialization and fails with `ErrUnsupportedOutputEncoding` — a name, comment, CDATA, PI target/data, namespace prefix, DTD-internal name (DOCTYPE, `<!ELEMENT>`/`<!ATTLIST>`/`<!ENTITY>` names, enumeration tokens), a character-map replacement, a DTD external/system/public-ID literal, an entity/notation value/name, or any future raw-write site. Per-site guards give early labelled errors on the common paths and an exhaustive output-writer net (`asciiRejectWriter`) rejects any surviving byte ≥ 0x80. Without an override a document's own US-ASCII encoding stays byte-identical. fn:serialize uses an internal declaration-only mode instead, keeping octets UTF-8 with char-reference escaping — a US-ASCII encoding there still character-references non-ASCII text/attr values, but keeps reference-less content and character-map replacements RAW (the result is a UTF-8 string), so the net and guards key on `asciiOutput && !declOnlyEncoding`), `Standalone(bool)` / `OmitStandalone()` (force the declaration's standalone pseudo-attribute to yes/no, or force its omission, overriding the document's own standalone status), `CharacterMap(map[rune]string)` (substitute a mapped rune with its raw replacement in text/attribute content — XSLT/XQuery Serialization 3.1 §7), `Normalization(string)` (the normalization-form serialization parameter — `"NFC"`/`"NFD"`/`"NFKC"`/`"NFKD"` normalize text-node and attribute-value character content; `""`/`"none"` disable it; ANY OTHER value is an error: `WriteTo` fails with `ErrUnsupportedNormalizationForm` before any output byte rather than silently disabling normalization), `CDATASectionElements(map[string]struct{})` (emit the named elements' direct text children as CDATA), and `SuppressIndentElements(map[string]struct{})` (serialize the named subtrees without indentation even when `Format` is on) — the last two match by EXACT expanded `{uri}local` name
  - XML 1.1 output: when the effective output version is `"1.1"` — either the serialized document declares version `"1.1"` (`Document.Version() == "1.1"`, set via `Document.SetVersion`) or `OutputVersion("1.1")` overrides it — the writer emits the XML 1.1 **restricted** control characters (`#x1-#x8`, `#xB-#xC`, `#xE-#x1F`, `#x7F-#x84`, `#x86-#x9F`; §2.11, tab/LF/CR excluded) **plus the two end-of-line characters NEL `#x85` and LINE SEPARATOR `#x2028`** (excluded from `RestrictedChar` but normalized to `#xA` on §2.11 input, so they must be char-ref'd to round-trip — the serialization set is `isXML11SerializeAsCharRef` = `isXML11RestrictedChar` ∪ {`#x85`, `#x2028`}) as **decimal** character references (`&#N;`) in text and attribute content instead of hex (`escapeNonASCII`) or U+FFFD replacement/`SERE0006` rejection. Gated on the document version, so XML 1.0 output is byte-identical (the `xml11` branch sits after the `RejectInvalidChars` check and before the `escapeNonASCII` hex branch, so it adds no extra walk; in XML 1.0 NEL/LS are ordinary characters written literally). The `stream.Writer` has the parallel `XMLVersion("1.1")` fluent method (also set by `StartDocument` when the declaration version is `1.1`): its text/attribute validation admits the restricted chars and `writeEscaped` serializes the same `isXML11SerializeAsCharRef` set as decimal refs (comment/PI/CDATA content, which cannot carry a reference, stay strictly rejected)
  - Terminal method: `WriteTo(io.Writer, Node) → error`
- **Write(io.Writer, Node) → error** — serialize node with default settings
- **WriteString(Node) → (string, error)** — serialize node to string with default settings
- **Element.FindAttribute(AttributePredicate) → (*Attribute, bool)** — attribute-node lookup by matcher; built-in matchers: `QNamePredicate`, `LocalNamePredicate`, `NSPredicate`
- **Element.GetAttribute(qname) → (string, bool)** / **Element.GetAttributeNS(local, nsURI) → (string, bool)** — attribute value lookup by QName or expanded name
- Key types: `Document`, `Element`, `Attribute`, `Namespace`, `DTD`, `Entity`, `Text`, `CDATASection`, `Comment`, `PI`
- `Node` interface — common for all node types; use ElementType enum to distinguish
- Parse flags configured via fluent methods on Parser (internal bitset, not public)
- `ErrorHandler` interface — delivers compilation/validation errors (synchronously unless the handler is `Sink`-backed); retained by reference and shared across operations, a nil handler is treated as `NilErrorHandler` (discard). The root `Parser` consults it only during DTD validation (see `Parser.ErrorHandler` above)
- `ErrorLeveler` interface — optional `ErrorLevel() ErrorLevel` an error implements to report its severity; `ErrorCollector`'s level filter reads it via `errors.As` (default `ErrorLevelWarning`)
- `DTDValidationError{Message, Level}` (`valid.go`) — structured DTD-validation diagnostic delivered to the `ErrorHandler` under `ValidateDTD(true)`; implements `ErrorLeveler` returning `ErrorLevelError`, so a level-filtered `ErrorCollector` keeps it. Recover via `errors.As`; `.Error()` returns `Message` (byte-identical to the prior `fmt.Errorf` text)
- `CatalogResolver` interface — public interface for custom catalog resolvers (`Resolve(ctx, pubID, sysID)`, `ResolveURI(ctx, uri)`)
- `ErrNoInternalSubset` — sentinel returned by `Document.InternalSubset()` when the document has no internal DTD subset (`IntSubset()` returns nil for the same condition); match with `errors.Is`
- `ErrDuplicateDeclaration` — sentinel returned when a DTD declaration collides with an existing one of the same kind and name: a second `AddElementDecl`, `AddNotation`, or `AddAttributeDecl` for an already-declared element, notation, or `(element, attribute)` pair. Wrapped (via `%w`) into a message naming the kind and name; match with `errors.Is`
- `ErrInvalidArgument` — sentinel returned when a public builder is given an argument that cannot produce a well-formed result (e.g. an `AddAttributeDecl` element/attribute name that is not a valid XML Name, a leading- or trailing-colon attribute name, an out-of-range attribute type or default kind, an ID attribute whose default is not `#IMPLIED`/`#REQUIRED`, a malformed/duplicate enumeration or NOTATION token (a NOTATION token additionally rejecting a colon, which no NotationDecl name may carry), a default value illegal for the type or not a declared enumeration member, a default value containing a non-XML-`Char`/invalid-UTF-8 byte or a raw `&` that would not round-trip, or a namespace-declaration attribute (`xmlns` or a name with the reserved `xmlns` prefix) whose value-bearing default binds a reserved prefix/URI the parser would reject as a namespace binding). Wrapped (via `%w`) into a message describing the violation; match with `errors.Is`
- `ErrExternalDTDTooLarge` — sentinel error returned when a loaded external DTD subset exceeds the byte cap; enforced against actual bytes read, never the advisory `fs.FileInfo.Size()`
- `ErrNodeContentTooLarge` — sentinel returned when a single CDATA/comment/PI/char-data run or attribute value — or a single contiguous run of XML whitespace (a blank skip) — exceeds `MaxNodeContentSize` (or `DefaultMaxNodeContentSize`); match with `errors.Is`
- `ErrUnsupportedOutputEncoding` — sentinel returned by the writer for an effective `OutputEncoding` that cannot be emitted faithfully, in three cases: (0) the effective encoding is a non-empty label that is NOT a well-formed XML `EncName` (`[A-Za-z] ([A-Za-z0-9._] | '-')*`, checked via `xmlchar.IsValidEncName` at the `writeDoc` entry before any output byte is written, ahead of the transcoding encoder so no BOM leaks) — a label carrying a quote or other illegal character would inject markup into the encoding pseudo-attribute; (1) the encoding is a well-formed label that is not UTF-8/US-ASCII and not loadable by the internal encoder table; (2) the encoding is US-ASCII (any alias) on the octet-producing WriteTo path and a non-ASCII byte reaches the output where no character reference can represent it — text and attribute values are always char-referenced, so ANY OTHER non-ASCII byte fails (names, comment, CDATA, PI, namespace prefix, DTD-internal names, character-map replacement, DTD external/system/public-ID literals, entity/notation value/name), caught by per-site guards plus an exhaustive `asciiRejectWriter` net; fn:serialize's declaration-only mode is excluded (keeps reference-less content raw in its UTF-8 string). The label-well-formedness check (case 0) also fires on a document's OWN encoding (`Document.SetEncoding`), since a malformed EncName cannot be faithfully emitted from any source; the transcoding cases (1)/(2) stay scoped to the override so a document's own unloadable parsed encoding stays declaration-only (default output byte-identical); match with `errors.Is`
- `ErrInvalidOutputVersion` — sentinel returned by the writer when the effective output XML version (the `OutputVersion` override, or the document's own version via `Document.SetVersion`) is not a valid XML `VersionNum` production `'1.' [0-9]+` (XML §2.8, e.g. `1.0`/`1.1`; a non-`1.x` version like `2.0` is also rejected). The version is emitted raw between the double quotes of the declaration's version pseudo-attribute, so the writer validates it (via `isValidXMLVersion`) at the `writeDoc` entry before writing any output byte and fails closed on a value carrying a quote/illegal character (markup injection) or a malformed value (unparseable declaration), emitting nothing. Valid `1.x` versions are byte-identical; match with `errors.Is`
- Writer structural-serialization sentinels (`errors.go`) — each flags a DOM the writer cannot serialize into well-formed XML, wrapped (via `%w`) into a descriptive value-bearing message so a caller can branch with `errors.Is`: `ErrWriterReservedElementName` / `ErrWriterReservedAttributeName` / `ErrWriterReservedNamespacePrefix` (an `xmlns` name or prefix that must go through `DeclareNamespace`, not a literal name), `ErrWriterInvalidElementName` / `ErrWriterInvalidAttributeName` / `ErrWriterInvalidNamespacePrefix` (a name that is not a valid QName/NCName and would inject raw markup), `ErrWriterInvalidComment` (comment content containing `--` or ending `-`), `ErrWriterInvalidPITarget` / `ErrWriterInvalidPIContent` (a bad PI target, or PI content containing `?>`), and `ErrWriterInvalidDTDNode` (a DTD element-content/entity/declaration node whose type field holds an unrecognized value). The name/comment/PI guards record the sticky error and preserve an earlier I/O error
- `ErrUnsupportedNormalizationForm` — sentinel returned by `Writer.WriteTo` when `Writer.Normalization` was given a value outside `{"", "none", "NFC", "NFD", "NFKC", "NFKD"}`. `Normalization` stores the raw form and the check is deferred to `WriteTo` (covering both the Document and bare-element paths), which fails closed before any output byte instead of silently disabling normalization; match with `errors.Is`
- `DefaultMaxNodeContentSize` — default single-construct content byte cap (10 MiB), also bounding a contiguous blank-skip run; used when `MaxNodeContentSize` is unset (0); a negative `MaxNodeContentSize` disables both the node-content and the blank-run cap
- `MaxExternalDTDSize` — default external-DTD byte cap (10 MiB), used when `MaxExternalDTDBytes` is unset or ≤ 0
- `Parser.MaxExternalDTDBytes(n int)` — override the external-DTD byte cap (n ≤ 0 → `MaxExternalDTDSize`)
- `AsNode[T Node](n Node) (T, bool)` — generic safe type assertion for Node types
- `Document.GetElementByID(id)` — O(1) via hash table, O(n) fallback; nil if no match. In a valid document an ID is unique, so at most one match. Duplicate IDs make a document invalid, not not-well-formed (it still parses with `DocWellFormed` set). If several elements share an ID the table path returns the last-registered element (RegisterID overwrites by call order, NOT necessarily last in document order) and the fallback walk the first — don't rely on either
- `Document.RegisterID(id, *Element)` / `Document.IDTable() → map[string]*Element` — register an ID->element entry / read the interned ID table (own map, read-only — mutating it corrupts the ID index; nil for API-built docs). `IDTable` lets a derived doc (e.g. an xsl:strip-space copy) rebuild a faithful ID table by translating each entry's element through an original->copy map, preserving the interned table's identity and O(1) fidelity rather than re-deriving ids through the lazy O(n) `GetElementByID` fallback walk (which itself resolves prefixed elements' qualified ATTLIST correctly)
- `Document.SkipIDs() → bool` / `Document.SetSkipIDs(bool)` — get/set the document's ID-skip state (mirrors the parser `SkipIDs` option). While true, `GetElementByID`/`fn:id` resolve no ids without an O(n) walk, even if an ID table exists; used when producing a derived document (e.g. an xsl:strip-space copy) that must mirror the source's ID semantics
- `Document.Encoding() → string` vs `Document.RawEncoding() → string` — `Encoding()` synthesizes `"utf8"` when the source omitted an encoding declaration; `RawEncoding()` returns the recorded value verbatim (empty = no declaration). The serializer emits `encoding="..."` only when the raw encoding is non-empty, so a faithful document copy must propagate `RawEncoding()`, not `Encoding()`
- `Walk(doc, fn)`, `Children(node)`, `Descendants(node)` — tree traversal
- `CopyNode(src, targetDoc)` — deep copy across documents
- `CopyDoc(src) → (*Document, error)` / `CopyDTDInfo(src, dst) → error` / `CopyExtSubset(src, dst)` — document-level deep copy: `CopyDoc` is a COMPLETE independent copy — the whole tree, BOTH DTD subsets, and the document-level state a caller relies on (version/encoding/standalone, URL, property flags, SkipIDs, and the interned ID table rebuilt against the copy's own elements; no mutable map or DTD is aliased); `CopyDTDInfo` copies the INTERNAL subset's metadata + entities/elements/attributes/notations into `dst` and RETURNS an error (notably when `dst` already has an internal subset); `CopyExtSubset` gives `dst` its own independent deep copy of the source's EXTERNAL DTD subset (`copy.go` / `copy_dtd.go` / `dtd.go`)
- `UnsafeAppendChild(parent MutableNode, child Node) → error` — links `child` into `parent`'s child slice bypassing the per-node cycle/duplicate-attribute preflight that `AddChild` runs; an explicitly-unsafe entry point (same boundary as the `UnsafeSet*` setters) where the caller guarantees an acyclic, duplicate-free child (freshly constructed trees, deep copies) (`tree_fastpath.go`)
- `UnsafeSetParent(n, target Node)` / `UnsafeSetPrevSibling(n, target Node)` / `UnsafeSetNextSibling(n, target Node)` — set ONE of `n`'s `parent`/`prev`/`next` pointers with no cycle detection, auto-unlinking, or reciprocal back-pointer maintenance; a misuse leaves the tree inconsistent or cyclic. These raw setters are deliberately NOT on the `MutableNode` interface (which exposes only guarded mutation: `AddChild`/`AddSibling`/`Replace`/`AppendText`/`SetLine`/`SetOwnerDocument`/`SetTreeDoc`), so ordinary tree mutation cannot reach them by accident. For low-level construction and cycle-guard tests only; ordinary code uses `AddChild`/`AddSibling`/`Replace`/`UnlinkNode`
- `Document.SetDocumentElement(root MutableNode) → error` — sets/replaces the document element; `root` must be an element (non-element kind → `ErrInvalidOperation`, nil/typed-nil root or nil receiver → `ErrNilNode`), leaving the document untouched on rejection
- DTD content-model builder API — `Document.CreateElementContent(name, etype) → (*ElementContent, error)` builds a leaf (`ElementContentElement`/`ElementContentPCDATA`); `Document.CreateElementContentSeq(c1, c2, occur)` / `Document.CreateElementContentChoice(c1, c2, occur)` build validated sequence (,) / choice (|) interior nodes (both children must be non-nil and structurally complete); `ElementContent.SetOccurrence(occur) → (*ElementContent, error)` sets the once/?/*/+ indicator. `DTD.AddElementDecl(name, typ, content)` rejects a structurally-incomplete model (`validateElementContentModel`) before storing, so serialization can never nil-deref. `DTD.RemoveElement(name, prefix) → *ElementDecl` unlinks the decl from the child list (not just the lookup table) and returns it (`document.go` / `dtd_elem.go` / `dtd.go` / `valid.go`)
- DTD declaration builders — the `DTD.Add*` family builds a declaration from public parameters, registers it in the lookup table, AND links it into the DTD child list so it serializes: `AddEntity(name, typ, publicID, systemID, content) → (*Entity, error)`, `AddNotation(name, publicID, systemID) → (*Notation, error)`, `AddElementDecl(name, typ, content) → (*ElementDecl, error)`, and `AddAttributeDecl(elem, name, atype, def, defvalue, enumValues Enumeration) → (*AttributeDecl, error)` — the latter builds an `<!ATTLIST>` declaration and validates every parameter against the same rules the parser's `<!ATTLIST>` path and the declaration-time validity constraints enforce (`validateAttrDeclIdentity` for the identity parts, then `validateAndNormalizeAttrDecl` for the rest), so a declaration it accepts serializes to markup a validating parser accepts and recovers equivalently: `elem`/`name` must be valid XML Names (colon allowed, no leading or trailing colon on `name`, which is split into prefix+local on the first colon as `AddElementDecl` does); `xml:id` must be `AttrID`; an ID attribute's default must be `#IMPLIED`/`#REQUIRED`; `enumValues` must be a non-empty list of well-formed, distinct tokens for `enum.AttrEnumeration`/`enum.AttrNotation` (enumeration tokens are NMTOKENs, colon allowed; NOTATION tokens are colon-free Names, since a NotationDecl name forbids a colon — and is cloned before storing); `#REQUIRED`/`#IMPLIED` carry no `defvalue` (checked against the value as given, before normalization) while `AttrDefaultNone`/`#FIXED` carry one (possibly empty), space-normalized `#x20`-only (`attrNormalizeSpace`, shared with the parser) for a non-CDATA type, its characters required to be legal XML `Char`s and free of a raw `&` (which the `<!ATTLIST>` default-value serializer cannot round-trip), validated against `atype`, and, for enumeration/NOTATION, required to be a declared member; a value-bearing default on a namespace-declaration attribute (`xmlns` or a name with the reserved `xmlns` prefix) additionally must bind a URI the Namespaces-in-XML reserved-prefix/URI rules permit — the parser applies such a default as a namespace binding, so `checkNamespaceDeclDefault` reuses the parser's pure predicates (`reservedPrefixedNamespaceViolation`/`reservedDefaultNamespaceViolation`) to reject a binding the parser would reject (the `xml` prefix bound to a non-XML URI, the `xmlns` prefix declared at all, or either reserved URI bound to another prefix or the default namespace); an empty default never binds and an ordinary `xml:*` attribute is not a namespace declaration, so neither is checked. The duplicate check runs BEFORE the non-identity parameter checks (a repeat of an already-declared attribute wins with `ErrDuplicateDeclaration`, matching the parser). Bad arguments return an error wrapping `ErrInvalidArgument`. The attribute table is keyed by an `attrDeclKey{local, prefix, elem}` struct (not a `local:prefix:elem` string, which would collide distinct triples — mirrors the parser's `specialAttrKey`). The low-level `registerAttribute` (table-only, no child-list link) is unexported; callers use `AddAttributeDecl` (`dtd.go`)
- `node.AddNamespaceDecl(*Namespace)` (promoted to `*Element` etc.) — appends an existing `*Namespace` to the node's declarations WITHOUT allocating a new one (unlike `DeclareNamespace`), so a built tree can reuse one `Namespace` as both a declaration and an element's active namespace; caller owns the ns and must not share it across independently-mutated nodes
- `NodeGetBase(doc, node)` — effective xml:base URI
- `BuildURI(reference, base)` — resolve a relative reference against a base; a byte-faithful port of libxml2 `xmlBuildURI(URI, base)`, so its argument order is `(reference, base)` (reference FIRST), the reverse of `url.URL.ResolveReference`/RFC 3986. c14n and all in-tree callers depend on this order. `node_base.go`
- `ResolveURI(base, ref) → (string, error)` — conventionally-ordered `(base, reference)` wrapper over `BuildURI`; returns an error when the reference cannot be resolved. Use this in new code instead of `BuildURI`'s reversed order. `node_base.go`
- Files: `parser.go` (API), `parserctx.go` (context/state), `parser_document.go`, `parser_element.go`, `parser_whitespace.go`, `parser_xml_decl.go`, `parser_encoding.go`, `parser_decl.go`, `parser_content.go`, `parser_dtd_subset.go`, `parser_dtd_element.go`, `parser_dtd_attr.go`, `parser_entity_decl.go`, `parser_entity_ref.go`, `parser_state_gen.go`, `document.go`, `element.go`, `attribute.go`, `node.go`, `node_leaf.go`, `node_namespace.go`, `node_base.go`, `tree_builder.go`, `tree_namespaces.go`, `tree_fastpath.go`, `writer.go`, `writer_escape.go`, `writer_dtd.go`, `writer_xhtml.go`, `copy.go`, `copy_deep.go`, `copy_dtd.go`, `dtd.go`, `dtd_attr.go`, `dtd_elem.go`, `iter.go`, `errorhandler.go`, `resolver.go`, `doc.go`

## c14n/

W3C Canonical XML. 3 modes: C14N10, ExclusiveC14N10, C14N11.

- **NewCanonicalizer(Mode) → Canonicalizer** — create fluent builder for the given mode
- Canonicalizer methods: Comments(), NodeSet([]Node), InclusiveNamespaces([]string), StrictXMLAttributes()
- Terminal: **Canonicalize(*Document, io.Writer) → error**, **CanonicalizeTo(*Document) → ([]byte, error)**
- C14N 1.1 xml:base is the lexical join (W3C §2.4 / libxml2 xmlC14NFixupBaseAttr) of in-document xml:base values — no external base URI. See `xmlbase.go` (joinURIReference).
- StrictXMLAttributes() opts into W3C-strict node-set xml:base/lang/space handling; default matches libxml2 (a rendered element's own excluded xml:* attribute is still emitted — XMLDSig digest interop). Strict mode is also fail-closed on xml:base: a degenerate/un-canonicalizable value (malformed URI, empty-authority "//"/"///"/"urn://") errors out of Canonicalize, where default emits best-effort bytes.
- Files: `c14n.go` (API), `canonicalizer.go` (engine), `xmlbase.go` (xml:base join), `nsstack.go`, `sort.go`, `escape.go`
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
- **NewEvaluator(EvaluatorOption) → Evaluator** — create evaluator from a flags bitmask (`DefaultEvaluatorOptions` = clone-on-write; `EvalBorrowing` = setters borrow caller-owned maps/slices without cloning)
  - `Evaluate(ctx, *Expression, Node) → (*Result, error)` — terminal method (`ctx` is cancellation only; config comes from the setters below)
- **Expression.Validate(map[string]string) → error** — static namespace-prefix validation; **Expression.EvaluateReuse(ctx, *EvalState, Node) → (Result, error)** — low-allocation evaluation; **Expression.DumpVM(io.Writer) → error** — compiled VM instruction dump
- **Evaluator fluent setters** (each returns an updated copy; maps/slices cloned unless `EvalBorrowing`): `Namespaces(map[string]string)`, `Variables(map[string]Sequence)`, `Functions(byLocal map[string]Function, byQName map[QualifiedName]Function)`, `VariableResolver(VariableResolver)`, `FunctionResolver(FunctionResolver)`, `OpLimit(int)`, `CurrentTime(time.Time)`, `ImplicitTimezone(*time.Location)`, `DefaultLanguage(string)`, `DefaultCollation(string)`, `DefaultDecimalFormat(DecimalFormat)`, `NamedDecimalFormats(map[QualifiedName]DecimalFormat)`, `BaseURI(string)`, `URIResolver(URIResolver)`, `CollectionResolver(CollectionResolver)`, `HTTPClient(*http.Client)`, `Position(int)`, `Size(int)`, `ContextItem(Item)`, `TypeAnnotations(map[helium.Node]string)`, `PreservedIDAnnotations(map[helium.Node]string)`, `IDNodes(map[helium.Node]struct{})` (PSVI is-id node set from `xsd.Validator.IDNodes`; a node here is treated as is-id by `fn:id`/`fn:element-with-id` in addition to those whose type annotation is a subtype of xs:ID — covers a singleton list of xs:ID and a union selecting an xs:ID-derived member), `SchemaDeclarations(SchemaDeclarations)`, `StrictPrefixes()`, `QNameValueNoDefaultNamespace()`, `AllowXML11Chars()`, `DocOrderCache(*DocOrderCache)`, `TraceWriter(io.Writer)`, `Parser(helium.Parser)` (XML parser used by `fn:parse-xml`/`fn:parse-xml-fragment`/`fn:doc`; supplies parse policy — limits, FS, XXE/network; unset → default `helium.NewParser()`)
- `Result` — wraps `Sequence`; methods: `Nodes()`, `IsBoolean()`, `IsNumber()`, `IsString()`, `IsAtomic()`, `Atomics()`, `Sequence()`, `StringValue()`, `Copy()`
- **Reuse:** `Evaluator.NewEvalState(Node) → *EvalState` builds reusable state; `Expression.EvaluateReuse` runs against it. The returned `Result` is valid only until the next `EvaluateReuse` on the same `EvalState` (backing storage is overwritten) — use `Result.Copy()` to retain it. `EvalState` has `SetContextItem`/`SetPosition`/`SetSize` and is NOT concurrency-safe
- **Evaluator.MaxResourceBytes(int64) → Evaluator** — cap bytes read from a single external resource by fn:unparsed-text(-lines/-available), fn:doc, fn:doc-available, fn:json-doc (0 = default cap, negative = unbounded); over-cap reads in fn:unparsed-text/fn:unparsed-text-lines fail FOUT1170 (fn:unparsed-text-available returns false), while fn:doc/fn:json-doc retrieval failures (incl. over-cap) surface as FODC0002 and fn:doc-available returns false
- **PredeclaredNamespace(prefix string) → (string, bool)** / **PredeclaredNamespaces() → map[string]string** — XPath 3.0 predeclared static-context prefix bindings (`fn`, `math`, `map`, `array`, `err`, `xs`). `PredeclaredNamespaces` returns a fresh copy of all bindings. Callers must let explicit in-scope namespace declarations override these fallbacks (used by xslt3 to keep compile-time and runtime pattern prefix resolution symmetric)
- **CompileRegex(pattern, flags string) → (*Regex, error)** — compile an XPath/XML Schema regex (flags `i`/`m`/`s`/`x`/`q`) for reuse by other packages (e.g. xslt3's `xsl:analyze-string`). `*Regex` methods: `MatchString(s) → (bool, error)`; `FindAllSubmatchIndex(s, n) → ([][]int, error)` (all matches, each a flat `(start,end)` index pair per group; `n<0` = unlimited); `EachSubmatchIndex(s, limit int, fn func([]int) bool) → error` — **streams** matches one at a time, calling `fn` per match (slice valid only during the call — copy to retain; unmatched group = `-1,-1`), stopping early when `fn` returns false. The streaming engines never accumulate, so live memory stays bounded regardless of match count and a caller can enforce a match-count budget (or honor a cancelled context) DURING enumeration. Leading-context patterns (e.g. multi-line `^`) can't stream on RE2, so they are matched in one bounded `FindAllStringSubmatchIndex` pass on Go `regexp` (staying linear — no backtracking-ReDoS regression for RE2-compatible patterns like `^(a+)+b`); `limit` (N+1 for a budget of N; `<=0` = uncapped) bounds that pass's allocation to the budget rather than the input match count
- XPath 3.1 features: FLWOR, quantified, if-then-else, try-catch, maps, arrays, inline functions, HOFs, arrow operator, simple map, string concat, value/general/node comparisons
- Built-in functions: 100+ across fn:, math:, map:, array: namespaces
- Type system: Sequence ([]Item), AtomicValue, NodeItem, MapItem, ArrayItem, FunctionItem
- Structured errors: XPathError with W3C error codes (XPTY0004, FOER0000, etc.)
- Limits: recursion 5000, node-set 10M, configurable op limit
- Runtime: `Compile()` first tries a direct fast path for simple path-like expressions and simple predicate comparisons, otherwise lowers AST to a VM instruction graph while collecting the prefix-validation plan, keeping trivial leaves inline in parent payloads and reusing parsed slices on the owned compile path; `Evaluate()` executes compiled refs by opcode and reuses shared eval helpers for semantics; AST/streamability access reparses from `Expression.source` on demand
- Files: `xpath3.go` (API), `parser.go`, `lexer.go`, `expr.go`, `token.go`, `consts.go`, `eval.go`, `eval_path.go`, `eval_operators.go`, `eval_arithmetic.go`, `eval_control.go`, `eval_types.go`, `eval_funcall.go`, `eval_reuse.go`, `evaluator.go`, `vm.go`, `vm_dump.go`, `compile_direct.go`, `compare.go`, `cast.go`, `cast_numeric.go`, `cast_string.go`, `cast_datetime.go`, `types.go`, `float_value.go`, `sequence.go`, `context.go`, `variables.go`, `collation.go`, `regex.go`, `regex_cache.go`, `static_check.go`, `streamability.go`, `node_identity.go`, `uri_resolution.go`, `doc.go`, `errors.go`, `arithmetic_datetime.go`, `parse_ietf_date.go`, `format_datetime.go`, `format_integer.go`, `format_number.go`, `function_library.go`, `function_signatures.go`, `functions.go` (registry + boolean/not/true/false + fn:error/fn:trace), `functions_node.go`, `functions_string.go`, `functions_numeric.go`, `functions_aggregate.go`, `functions_sequence.go`, `functions_datetime.go`, `functions_uri.go`, `functions_qname.go`, `functions_hof.go`, `functions_map.go`, `functions_array.go`, `functions_math.go`, `functions_misc.go`, `functions_json.go`, `functions_json_xml.go`, `functions_serialize.go`, `functions_constructors.go` (XSD typed constructors, incl. xs:error), `functions_unparsed_text.go`
- Imports: helium, internal/xpath, internal/lexicon, internal/icu, internal/unparsedtext, internal/strcursor, internal/sequence

## xslt3/

XSLT 3.0 stylesheet compilation + transformation on helium DOM with `xpath3` evaluation.

- **CompileStylesheet(ctx, *Document) → (*Stylesheet, error)** — convenience compile wrapper
- **NewCompiler() → Compiler** — builder for stylesheet compilation
- **Compiler.BaseURI(string) → Compiler** / **Compiler.URIResolver(URIResolver) → Compiler** / **Compiler.PackageResolver(PackageResolver) → Compiler** — compile-time resource/package resolution. Secure by default: `Compiler.URIResolver` is the opt-in for ALL compile-time stylesheet loading — `xsl:import`, `xsl:include`, output-format parameter documents (`xsl:output @parameter-document`), compile-time `fn:transform` `stylesheet-location` (e.g. static-variable evaluation), and compile-time `doc()`/`doc-available()` inside `use-when` (resolved against the module's effective static base, i.e. the module URI with its root `xml:base` folded in — including for an external `xsl:import`/`xsl:include` module's root `use-when`). With no `URIResolver` configured there is no implicit `os.ReadFile`; each of those loads errors out (`xsl:import`/`xsl:include` → "no URIResolver configured"; parameter docs → XTSE0090; `fn:transform` → FOXT0003; use-when `doc-available()` → false). Runtime `fn:transform stylesheet-location` likewise requires the compile-time `URIResolver` carried on the stylesheet.
- **Compiler.StaticParameters(*Parameters) → Compiler** / **Compiler.SetStaticParameter(string, Sequence) → Compiler** / **Compiler.ClearStaticParameters() → Compiler** / **Compiler.ImportSchemas(...*xsd.Schema) → Compiler** — compile-time static params + schema imports
- **Compiler.MaxResourceBytes(int64) → Compiler** — set the per-resource read cap inherited by invocations (0 = [MaxResourceBytes] default, negative = unbounded, positive = that cap)
- **Compiler.Parser(helium.Parser) → Compiler** / **Invocation.Parser(helium.Parser) → Invocation** — the parser governing parse policy (limits, FS, XXE/network) for stylesheet, schema, and runtime source/`fn:doc` parsing; **forwarded** into the `xsd.Compiler`s and `xpath3.Evaluator`s the engine builds internally. xslt3 still forces its functional needs (entity substitution; `fn:doc` default-DTD-attributes/base-uri handling). Unset → the hardened default; `Invocation.Parser` overrides the compiler's for that run
- **Compiler.AllowExternalEntities(bool) → Compiler** — XXE policy for compile-time parses of external stylesheet modules (`xsl:import`/`xsl:include`/`xsl:use-package`, and compile-time `fn:transform` stylesheets). **Default false (XXE BLOCKED): external DTDs / external general entities are NOT loaded or substituted** (parser is `BlockXXE(true).LoadExternalDTD(false).AllowNetwork(false)`). Set true to restore the legacy permissive behavior (resolver-mediated external entity loading via `LoadExternalDTD(true).SubstituteEntities(true)`, subject to `MaxResourceBytes`). The compiled value is carried on the `Stylesheet` and inherited by `fn:transform` nested compiles and (unless overridden) by runtime invocations. Serialization parameter documents and imported XSD schemas are always parsed XXE-blocked.
- **Compiler.Compile(ctx, *Document) → (*Stylesheet, error)** / **Compiler.MustCompile(ctx, *Document) → *Stylesheet** — terminal compile methods
- **Transform(ctx, *Document, *Stylesheet) → (*Document, error)** / **TransformToWriter(ctx, *Document, *Stylesheet, io.Writer) → error** / **TransformString(ctx, *Document, *Stylesheet) → (string, error)** — convenience wrappers; nil `*Stylesheet` returns error here
- **Stylesheet.Transform(*Document) → Invocation** / **Stylesheet.ApplyTemplates(*Document) → Invocation** / **Stylesheet.CallTemplate(string) → Invocation** / **Stylesheet.CallFunction(string, ...Sequence) → Invocation** — invocation entrypoints
- **Invocation.SourceDocument(*Document) → Invocation** / **Mode(string)** / **Selection(Sequence)** *(ApplyTemplates only)* / **GlobalParameters(*Parameters)** / **TunnelParameters(*Parameters)** / **SetParameter(string, Sequence)** / **SetTunnelParameter(string, Sequence)** / **SetInitialTemplateParameter(string, Sequence)** / **SetInitialModeParameter(string, Sequence)** / **MessageHandler(MessageHandler)** / **ResultDocumentHandler(ResultDocumentHandler)** / **RawResultHandler(RawResultHandler)** / **PrimaryItemsHandler(PrimaryItemsHandler)** / **AnnotationHandler(AnnotationHandler)** / **CollectionResolver(xpath3.CollectionResolver)** / **URIResolver(xpath3.URIResolver)** / **HTTPClient(\*http.Client)** / **BaseOutputURI(string)** / **SourceSchemas(...*xsd.Schema)** / **OnMultipleMatch(OnMultipleMatchMode)** / **TraceWriter(io.Writer)** / **GlobalContextSelect(string)** / **MaxResourceBytes(int64)** / **AllowExternalEntities(bool)** — fluent runtime configuration. `GlobalContextSelect` sets an XPath expression (evaluated against the source document after whitespace stripping) that determines the global context item; if it evaluates to an empty sequence the global context item is absent and global variables referencing `.` raise XPDY0002. `AllowExternalEntities` sets the XXE policy for runtime parses of external documents (`fn:doc`/`document()`, `xsl:source-document`, `xsl:merge`, and `fn:transform` stylesheet sources). **Default false (XXE BLOCKED): external DTDs / external general entities are NOT loaded or substituted**; when left unset it inherits the value compiled into the stylesheet (`Compiler.AllowExternalEntities`); set true to restore the legacy permissive behavior for trusted documents. `MaxResourceBytes` caps bytes read from a single runtime external resource: 0 inherits the Compiler/stylesheet cap (then the [MaxResourceBytes] default), negative disables the bound, positive sets that cap. The cap applies to all runtime reads, but the over-cap error differs by layer: XSLT's own loader (`fn:doc`/`document()`, `xsl:source-document`, `xsl:merge`, runtime `xsl:result-document` parameter documents, `xsi:schemaLocation` source schemas, `fn:transform` stylesheet/package sources) fails with [ErrResourceTooLarge], whereas the XPath built-ins `fn:unparsed-text`/`fn:unparsed-text-lines` surface FOUT1170 (`fn:unparsed-text-available` returns false) and `fn:json-doc` surfaces FODC0002 — they honor the cap but do NOT carry the `ErrResourceTooLarge` sentinel. `URIResolver` and `HTTPClient` are the opt-in for runtime resource retrieval — `fn:doc`/`fn:unparsed-text`, plus `xsl:source-document`, `xsl:merge`, and `fn:stream-available`; without them those instructions error (`FODC0002`) or report unavailable per the default-deny model (no implicit `os.ReadFile`).
- **Invocation.Do(ctx) → (*Document, error)** / **Invocation.Serialize(ctx) → (string, error)** / **Invocation.WriteTo(ctx, io.Writer) → error** / **Invocation.ResolvedOutputDef() → *OutputDef** — terminal execution + resolved primary output metadata
- **NewParameters() → *Parameters** — mutable XSLT parameter carrier keyed by expanded name
- **TransformFunction(...TransformOption) → xpath3.Function** — standalone `fn:transform()` for registering on a bare `xpath3.Evaluator` (`Evaluator.Functions(nil, {fn:transform: ...})`), for callers driving xpath3 directly with no outer running stylesheet (e.g. the QT3 harness). Shares its implementation with the in-stylesheet `fn:transform` (`ec.fnTransform` is a thin wrapper over the same `transformFnConfig.run`). Deps the in-stylesheet path inherits from its execution context are supplied explicitly via `TransformOption`: `WithTransformURIResolver(xpath3.URIResolver)` (stylesheet-location reads, nested-compile module loading, and inner `fn:doc`), `WithTransformPackageResolver`, `WithTransformHTTPClient`, `WithTransformBaseURI`, `WithTransformMaxResourceBytes`, `WithTransformAllowExternalEntities`, `WithTransformParser`, `WithTransformImportSchemas`. Handles the `stylesheet-location`, `stylesheet-node`, `stylesheet-text` (inline source), and `package-name` stylesheet-source options. A `stylesheet-node` that is an element node which is not its owner document's document element (a simplified/literal-result stylesheet supplied as a fragment child, or an element nested in a larger tree) is compiled from that element as the stylesheet root — `stylesheetNodeCompileDocument` deep-copies the element into a fresh document so the element itself is the document element — rather than compiling the whole owning document (whose document element would be a different node). The `stylesheet-base-uri` map option sets the base URI for resolving relative `xsl:include`/`xsl:import` inside a `stylesheet-text`/`stylesheet-node` (a relative value is itself resolved against the call's static base URI, `WithTransformBaseURI`); it overrides `WithTransformBaseURI` for `stylesheet-text` and the node's own document base URI (`NodeGetBase`) for `stylesheet-node`. Absent the option, `stylesheet-text` falls back to `WithTransformBaseURI` and `stylesheet-node` to the node's document base URI, so a relative include with no usable base fails (XTSE0165). `initial-template` is resolved preserving its namespace (a QName value becomes Clark `{uri}local`), so a namespaced named template resolves. `run` validates the option map before doing work (`validateTransformOptions`, F&O 3.1 §14.8.3): exactly one of `stylesheet-location`/`stylesheet-node`/`stylesheet-text`/`package-name`, at most one of `initial-template`/`initial-mode`/`initial-function`, `source-node` and `initial-match-selection` mutually exclusive, `delivery-format` ∈ {document, serialized, raw}, `xslt-version` numeric, and each param map (`stylesheet-params`/`static-params`/`template-params`/`tunnel-params`) a map keyed by xs:QName — a structural/enum/exclusivity violation is FOXT0002, a mistyped option value is XPTY0004. `requested-properties` is checked against helium's advertised capabilities (`transformCapabilities`): `is-schema-aware`, `supports-serialization`, `supports-backwards-compatibility`, `supports-namespace-axis`, `supports-dynamic-evaluation`, `supports-higher-order-functions` are true and `supports-streaming` is false (DOM-materialization, not streaming); a recognized XSLT-namespace property requested with an unsatisfiable value raises FOXT0006. When `delivery-format` is `serialized`, the `serialization-params` map overrides the principal result's `OutputDef` (`applySerializationParams`: method, indent, omit-xml-declaration, byte-order-mark, undeclare-prefixes, encoding, version, standalone, media-type, doctype-public/system, item-separator). The QName-valued `cdata-section-elements` and `suppress-indentation` params (sequences of xs:QName, whose {uri}local Clark names are extracted via `mergeSerializationQNames`) and the map-valued `use-character-maps` param (a `map(xs:string, xs:string)` merged via `applyUseCharacterMapsParam`) accumulate onto the stylesheet's own `xsl:output` values: the two element-name sets are unioned, and character-map entries are merged with the serialization-params value winning per character. A recognized param present with an empty-sequence value resets it to the serialization default (`resetSerializationParam`). Two bases are kept SEPARATE (F&O 3.1 §14.8). (1) The PRINCIPAL result-map key: a relative `base-output-uri` is resolved ONCE against the fn:transform call's static base URI (`cfg.baseURI` — `ec.effectiveStaticBaseURI()` in-stylesheet, i.e. the call-site base honoring an `xml:base` on the calling template/element, not the bare module URI; or `WithTransformBaseURI` standalone) via `canonicalResultURIKey`; the result map keys the principal by that resolved value when supplied, else by the literal `"output"`. (2) The base for RESOLVING secondary result-document output URIs (`cfg.outputBaseURI`, seeded into `ec.currentOutputURI`): the resolved `base-output-uri` when supplied, else the call's effective static base URI — so secondary result-map keys are absolute whenever ANY base exists, even when `base-output-uri` is omitted. Only when there is genuinely no base at all (no `base-output-uri` AND no static base — e.g. a standalone call with no `WithTransformBaseURI`) does a secondary key remain best-effort relative; the spec cannot require an absolute URI when no base is available. When `base-output-uri` is absent the principal output has no declared URI (no `canonicalPrimaryURI` XTDE1490 reservation). the principal entry is emitted only when the transformation actually wrote principal content, so a stylesheet producing only secondary `xsl:result-document`s has no principal entry (W3C bug 30209, `documentHasChildren`). Each secondary result document is keyed by its fully-resolved absolute output URI. Internally the `ec.resultDocuments`/`resultDocItems`/`resultDocOutputDefs` stores, the XTDE1490 duplicate-detection set (`usedResultURIs`), and `Document.URL` all key on the SAME resolved canonical URI (`dupKey == ec.currentOutputURI`), so a nested `xsl:result-document` resolves against its enclosing document's dynamic output URI (not the top-level base) and two nested documents writing the same relative href under different enclosing URIs never overwrite one another; the `resultDocCollector` reads `Document.URL` to key the result map, so those distinct absolute URIs both survive. The raw href as written is preserved separately (`ec.resultDocHrefs`) solely for the public `ResultDocumentHandler`, which receives it unchanged. Under `serialized` delivery the xml-family serializer's document-terminating newline is trimmed (`serializeDeliveredResult`/`preservesTrailingNewline`), but a `text`/`json`/`adaptive` result keeps its trailing newline (legitimate content). A `source-node` that is not a document node is itself the initial match selection (the template matches that element), while the source tree's document root supplies the default global context item. The `global-context-item` option (required type `item()`) determines the context item seen by global variables/parameters independently of the initial match selection: a present-but-empty or multi-item value is `XPTY0004` (`validateTransformGlobalContextItem`); an explicit item (node or atomic/map/array/function) overrides the source-node default and is what gets type-checked against `xsl:global-context-item/@as` (`XTTE0590`) — a non-node item is exposed via `ec.contextItem` scoped to global evaluation only and never leaks into template execution. When the stylesheet declares `xsl:global-context-item use="absent"` the global context item is absent regardless of any supplied option (a global `.` raises `XPDY0002`) — this affects ONLY the global context item (`ec.globalContextAbsent`), NOT the source-node/initial match selection, so a supplied source-node still drives apply-templates normally (no spurious `XTDE0040`); when neither a source-node/initial-match-selection nor an explicit item is supplied the global context item is likewise absent (`cfg.globalContextAbsent` → `XPDY0002` on `.`, and `XTDE3086` if `use="required"`), even though a synthetic empty document is still substituted as the navigable source tree for an initial-template/-function entry. A `post-process` function item (`function(xs:string, item()*) as item()*`) is invoked once per delivered result — the principal `output` entry and each secondary result — with `(result-URI, result-value)` (the principal gets `base-output-uri`, each secondary its href key); its return value replaces the entry's value in the result map, across all three delivery formats (document node, serialized string, raw items).
- Key types: `Stylesheet`, `Compiler`, `Invocation`, `Parameters`, `OutputDef`, `URIResolver`, `PackageResolver`, `MessageHandler`, `ResultDocumentHandler`, `RawResultHandler`, `PrimaryItemsHandler`, `AnnotationHandler`, `TransformOption`
- Resource limits: `MaxResourceBytes` (const, 10 MiB default per-resource read cap) + `ErrResourceTooLarge` (error returned when an external resource exceeds the cap); enforced against actual bytes read, configurable per Compiler/Invocation. The same cap doubles as the xsl:analyze-string match-count ceiling: matches are streamed one at a time (via `xpath3.Regex.EachSubmatchIndex`) and the running count is checked during enumeration, so an empty-matching regex over a large input is rejected with `ErrResourceTooLarge` without allocating memory proportional to the match count
- Supports: `xsl:template`, `xsl:apply-templates`, `xsl:call-template`, `xsl:param`/`xsl:variable`, `xsl:include`/`xsl:import`, `xsl:sort`, `xsl:number`, `xsl:message`, `xsl:key`, `xsl:output`, `xsl:import-schema`, `xsl:function`, literal result elements, AVTs, `xsl:attribute-set`, `xsl:map`/`xsl:map-entry`, `xsl:source-document`, `xsl:iterate`, `xsl:fork`, `xsl:accumulator`, `xsl:merge`, `xsl:where-populated`, `xsl:try`/`xsl:catch`, `xsl:for-each-group`, `xsl:result-document`, `xsl:next-match`, `xsl:apply-imports`
- Schema awareness: `xsl:import-schema` compiles XSD schemas, `type=` on `xsl:element`/`xsl:attribute` annotates result nodes, `validation=` on `xsl:copy`/`xsl:copy-of`, `default-validation` stylesheet attribute, `type-available()` function, runtime source validation via `Invocation.SourceSchemas(...)`, annotation callbacks via `AnnotationHandler`
- Streaming: `xsl:source-document` (DOM-materialization), `xsl:iterate`/`xsl:break`/`xsl:next-iteration`/`xsl:on-completion`, `xsl:fork`, `xsl:accumulator`/`xsl:accumulator-rule`, `xsl:merge`/`xsl:merge-source`/`xsl:merge-key`/`xsl:merge-action`; streamability analysis (XTSE3430) post-compilation pass
- Runtime helpers: `current()`, `document()`, `key()`, `generate-id()`, `system-property()`, `unparsed-entity-uri()`, `unparsed-entity-public-id()`, `type-available()`, `snapshot()`, `copy-of()`, `accumulator-before()`/`accumulator-after()`, `current-merge-group()`/`current-merge-key()`, `transform()`
- Output methods: `xml`, `html`, `xhtml`, `text`, `json`, `adaptive`
- Files: `xslt3.go` (package doc + convenience wrappers), `doc.go`, `compile.go` (compiler builder + orchestration), `compile_*.go` (imports/packages/schema/templates/functions/modes/formats/patterns/streaming/instruction compilation), `execute*.go` (runtime), `functions*.go` (built-ins + `fn:transform` bridge), `stylesheet.go`, `invocation.go`, `instruction.go`, `parameters.go`, `options.go`, `output*.go` (`output.go`, `output_xml.go`, `output_html.go`, `output_json.go`, `output_adaptive.go`, `output_charmap.go`), `sort.go`, `types.go`, `avt.go`, `keys.go`, `qname_resolve.go`, `number_words.go`, `source_schema.go`, `schema_constructors.go`, `schema_context.go`, `schema_resolver_fs.go`, `package_*.go`, `streamability*.go`, `errors.go`, `resource_limit.go` (per-resource read cap + `MaxResourceBytes`/`ErrResourceTooLarge`); the XSLT element registry lives in `xslt3/internal/elements` (`elements.go`, `data.go`, see below)
- Imports: helium, xpath3, xsd, html, internal/lexicon, internal/sequence, xslt3/internal/elements
- Tests: hand-written unit tests only. The W3C XSLT 3.0 conformance suite lives in the sibling `helium-w3c-tests` module (fetches upstream, depends on this module via a replace directive)

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

XML Schema (XSD) compilation and validation. Defaults to XSD 1.0; XSD 1.1 is opt-in via `Compiler.Version(xsd.Version11)` (or a `vc:minVersion="1.1"` hint on the schema root). See the "XSD — Version Toggle" section in CLAUDE.md for what is implemented in 1.1.

- **NewCompiler() → Compiler** — create fluent builder for schema compilation
  - `Label(name)`, `BaseDir(dir)`, `FS(fs.FS)`, `ErrorHandler(h)`, `Version(Version)` — builder methods (clone-on-write). `Version(Version10|Version11)` selects the XSD spec version (default `Version10`)
  - `Compiler.Parser(helium.Parser)` — sets the parser used to parse the schema document and all nested `xs:include`/`xs:import`/`xs:redefine` targets; supplies parse policy (limits, FS, XXE/network). Distinct from `FS`, which *fetches* schema bytes; the injected parser governs *parse policy* of those bytes. Unset → default schema parser (`helium.NewParser().SubstituteEntities(true)`) so entity references in schema attribute values are expanded; an explicit parser is used exactly as supplied.
  - `Compiler.FS(fs.FS)` — sets the `fs.FS` used to load schemas referenced by `xs:include`, `xs:import`, and `xs:redefine`. **Secure by default** (mirrors `helium.NewParser`): the default (and what a nil value restores) is a deny-all FS (`internal/iofs.DenyAll`, opens nothing), so an untrusted schema cannot disclose local files or exhaust resources via a hostile `schemaLocation`. Opt into host access with `helium.PermissiveFS()` (any `os.Open` path) or a confined FS. Each nested schema is read through a fixed `maxNestedSchemaSize` byte cap (10 MiB) regardless of FS, so an endless source (e.g. `schemaLocation` → `/dev/zero`) cannot exhaust memory; an over-cap read is fatal (`errSchemaTooLarge`, see `IsFatalSchemaLoad`). Schema-location resolution is URI-aware: when `BaseDir` is a URI (e.g. `https://…` or `file:///…`) a relative include is resolved with RFC 3986 semantics and an absolute-URI include is passed through unchanged, so the name handed to the FS is the canonical nested-schema URI; when `BaseDir` is a local path, names use `filepath.Join` and may be absolute / OS-style (rejected by `fs.ValidPath` FSes like `os.DirFS`/`fstest.MapFS`)
  - `Compile(ctx, *Document) → (*Schema, error)` / `CompileFile(ctx, path) → (*Schema, error)` — terminal methods; return `(nil, ErrCompilationFailed)` on fatal schema diagnostics
- **NewValidator(schema) → Validator** — create fluent builder for validation
  - `Label(name)`, `ErrorHandler(h)`, `Annotations(*TypeAnnotations)`, `NilledElements(*NilledElements)`, `IDNodes(*IDNodes)` — builder methods. `IDNodes` collects the PSVI is-id nodes (XDM 3.1): every attribute/element whose actual validated content is a single xs:ID-derived value — an atomic xs:ID, a SINGLETON list of xs:ID, or a union that SELECTS an xs:ID-derived member. A multi-item list or a non-ID union member is not is-id. Version-independent and runs regardless of `SkipDatatypeIntegrityChecks` (it observes a per-node property, it does not enforce document ID uniqueness). Feeds `xpath3.Evaluator.IDNodes` for `fn:id`/`fn:element-with-id`
  - `Validate(ctx, *Document) → error` — terminal method
- **(*TypeDef).Validate(ctx, value, nsMap) → error** — validate a lexical value against a simple type; nsMap (prefix→URI) may be nil
- **(*TypeDef).ValidateElement(ctx, elem, schema) → error** — validate an element's content against a type
- `Schema.LookupElement(local, ns)`, `Schema.LookupType(local, ns)`, `Schema.NamedTypes()`, `Schema.TargetNamespace()`
- **Schema.Declarations() → xpath3.SchemaDeclarations** — an `xpath3.SchemaDeclarations` view over the compiled schema for schema-aware XPath (schema-element/attribute node tests, schema-aware cast/castable, instance-of/subtype tests, substitution-group membership, typed-value atomization of PSVI-annotated nodes). Pair with `xpath3.Evaluator.SchemaDeclarations(...)` and `TypeAnnotations(...)` (fed by `Validator.Annotations`). Borrows the schema read-only; safe to share across concurrent evaluations. Backed by the `schemaDecls` adapter in `schema_decls.go` — the same one used internally for xs:assert evaluation, and the single canonical implementation xslt3's multi-schema `schemaRegistry` (`xslt3/schema_context.go`) delegates its per-schema `SchemaDeclarations` lookups to. The one method xslt3 does NOT delegate is `IsSubtypeOf`: XSLT SequenceType matching treats a simpleContent COMPLEX type as a subtype of its simple content base (so it matches `element(*, T)`), whereas this adapter deliberately excludes a complex type from its simple ancestry (so `instance of element(*, xs:string)` is false for an xs:assert node)
- **ResolveSchemaURI(ref, base) → (string, error)** / **URIScheme(s) → string** — the single canonical schema-location URI-resolution helper and scheme-detector, shared with `xslt3` so the two layers cannot drift (URI-aware: absolute-URI pass-through, RFC 3986 with `OmitHost` preservation for URI bases, `filepath.Join` + `..`-escape guard for local bases)
- **FatalSchemaLoader** interface (`FatalSchemaLoad() bool`) — a `Compiler.FS` may return an `Open` error whose chain carries a value satisfying this interface to force an `xs:import` load failure to be FATAL instead of the usual warn-and-continue ("Skipping the import."). `xslt3`'s `schemaResolverFS` uses it so an over-cap nested-import read (`ErrResourceTooLarge`) is not silently skipped; the wrapped chain is preserved so callers can still `errors.Is`/`errors.As` the cause
- **IsFatalSchemaLoad(err) → bool** — the SINGLE source of truth for "is this a fatal schema-load condition that must abort compilation rather than warn-and-continue or fall back to a pre-compiled schema". Returns true (via `errors.Is`/`errors.As`) for a schema-location `..`-escape, an `xs:import` depth overflow, an `xs:include`/`xs:redefine` depth overflow (`errIncludeDepthExceeded` — otherwise an over-deep include chain inside an IMPORTED schema would be demoted to a warning and silently ignored by `loadImport`), an over-cap nested-schema read (`errSchemaTooLarge`), and any error satisfying `FatalSchemaLoader`. The two xsd import warn-or-continue sites and `xslt3`'s `xsl:import-schema` fallback guard both route through it (xslt3's `isFatalSchemaLoadError` delegates to it, adding the xslt3-package `ErrResourceTooLarge` sentinel), so the classification cannot drift between the layers. The path-escape / depth sentinels stay unexported; this helper is the public surface
- Supports: complex/simple types, sequences, choices, all, groups, attribute groups, substitution groups, import/include/redefine/override, IDC (xs:unique/key/keyref), XSD 1.1 assertions/assertion facets, conditional type assignment, open content, schema default attributes, wildcard algebra, relaxed xs:all, and relaxed wildcard/UPA behavior
- `ElementDecl.SubstitutionGroup` = first substitution-group head QName for compatibility; `ElementDecl.SubstitutionGroups` = all head QNames. XSD 1.1 `@substitutionGroup` may be XSD-whitespace-separated QName list; XSD 1.0 preserves single-QName behavior. `Schema.SubstGroupMembers(head)` returns the eligible transitive member set after substitution block and derivation filtering.
- `ErrValidationFailed` — sentinel error returned by `Validate()` when the document is invalid; individual errors delivered via `ErrorHandler`. `Validate()` also returns `ErrNilSchema` (no compiled schema) and `ErrNilDocument` (nil document); a nil `ctx` is normalized to `context.Background()`
- `ErrCompilationFailed` — sentinel error returned by `Compile()`/`CompileFile()` when the schema has one or more fatal errors; the returned schema is nil and individual diagnostics are delivered via `ErrorHandler`
- Files: `xsd.go` (API), `doc.go`, `schema.go` (data model), `constants.go`, `compile.go` + `compile_imports.go` (compile orchestration/imports), `resolve_uri.go` (shared schema-location URI resolver `ResolveSchemaURI`/`URIScheme`), `read_types.go` + `read_particles.go` + `read_elements.go` (schema readers), `link_refs.go` + `restriction_particle.go` + `all_subsumption.go` + `substitution_group.go` + `wildcard_algebra.go` + `check_*.go` (`check_element_consistent.go`, `check_elements.go`, `check_facets.go`, `check_upa.go`; reference resolution + constraints), `validate.go` + `validate_elem.go` + `validate_idc.go` + `validate_id.go` (validation flow/content/IDC/ID), `simplevalue_core.go` + `simplevalue_facets.go` (simple-value engine), `assert.go` + `assertion_facet.go` (XSD 1.1 assertions), `alternative.go` (conditional type assignment), `conditional_inclusion.go` (XSD 1.1 conditional inclusion), `opencontent.go` (open content), `override.go` (xs:override), `inherited_attrs.go` (XSD 1.1 inherited attributes), `schema_decls.go` (schema-aware XPath adapter), `errors.go`
- Imports: helium, xpath1/, xpath3/ (XSD 1.1 assertions + conditional type assignment), internal/lexicon
- Status: see `.claude/docs/libxml2-parity.md` for libxml2 golden counts and W3C XSD 1.1 conformance run policy; do not cache branch-specific counts here

## relaxng/

RELAX NG schema compilation and validation.

- **NewCompiler() → Compiler** — create fluent builder for grammar compilation
  - `Label(name)`, `BaseDir(dir)`, `FS(fs.FS)`, `MaxResourceBytes(int)`, `ErrorHandler(h)` — builder methods (clone-on-write)
  - `Compiler.BaseDir(dir)` — base directory for resolving relative paths in `include` and `externalRef` during compilation
  - `Compiler.Parser(helium.Parser)` — sets the parser used to parse the grammar and its `include`/`externalRef` targets; supplies parse policy (limits, FS, XXE/network), distinct from the fetch `FS`. Unset → default `helium.NewParser()`.
  - `Compiler.FS(fs.FS)` — sets the `fs.FS` used to load schemas referenced by `include` and `externalRef`. **Secure by default**: the default (and what a nil value restores) is a deny-all FS (`internal/iofs.DenyAll`, opens nothing), mirroring `helium.NewParser`, so an untrusted schema cannot read host files via `include`/`externalRef`. Pass `helium.PermissiveFS()` (any `os.Open` path) or a confined FS to opt into loading. Resolution (`resolveHref` in `parse.go`) honors an absolute href as-is first; otherwise it resolves against ancestor `xml:base` via `BuildURI`; only when neither applies does it fall back to `filepath.Join(BaseDir, href)`, and finally to the bare href. The resolved name may thus be absolute / OS-style; FS implementations enforcing `fs.ValidPath` (`os.DirFS`, `fstest.MapFS`) reject them, so a sandboxing FS must accept OS-style names
  - `Compiler.MaxResourceBytes(int)` — per-resource byte cap on each `include`/`externalRef` target read (`readResource` in `parse.go`, via `internal/iolimit`). Default 10 MiB (`defaultMaxResourceBytes`); `<= 0` restores the default. An over-cap resource fails to load with an "exceeds the maximum resource size" compile error rather than being read in full
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
- Imports: helium, internal/lexicon, internal/iofs, internal/iolimit, internal/xsd/value, internal/xsdregex, internal/xmlchar, internal/uripath
- Status: 159/159 golden tests passing

## html/

HTML 4.01 parser producing helium DOM or SAX events.

- **NewParser() → Parser** — create fluent parser builder
- Parser methods: `SuppressImplied(bool)`, `StripBlanks(bool)`, `SuppressErrors(bool)`, `SuppressWarnings(bool)`, `Strict(bool)`, `MaxContentSize(int)` (approximate soft per-chunk cap for normal data-state text and raw-text/RCDATA/plaintext — chunks target this size but an indivisible token, e.g. a whole UTF-8 rune or resolved char-ref, is never split, so a chunk may slightly exceed it; HARD cap for comment/bogus-comment/PI — over-cap fails the parse with `ErrContentSizeExceeded` since those are indivisible nodes; normal data-state and RCDATA char-ref resolution share the same cap-aware path (`parseCharRefBounded`) — it uses a FIXED `maxEntityNameLen` (~32 byte) lookahead independent of the cap, so a SHORT resolvable named reference (known entity or legacy prefix) whose run fits the cap is never rejected for being a small name (`&amp;` resolves under `MaxContentSize(2)`); ANY UNRESOLVED named-reference literal (whether short, semicolon-terminated, or unbounded) fails with `ErrContentSizeExceeded` once the bytes it would emit (`&` + name + optional `;`) exceed the cap; a SATURATED ambiguous legacy-prefix run (`&amp` + a tail that overflows the 32-byte lookahead) is consumed into a cap-bounded spool and HARD-FAILS with `ErrContentSizeExceeded` if it exceeds the cap before its end is reached, emitting nothing — only a within-cap saturated run legacy-resolves; a normal data-state run's LEADING whitespace prefix is deferred (buffer `pendingWS`) until its first non-whitespace byte fixes both whitespace-significance (`StripBlanks`) and implied-`<body>` insertion — so `<html> a` keeps the space and `a` in one run under `<body>`, and `<p> &amp;</p>`/`<p> < b</p>` keep the leading space; that deferred prefix is bounded by the cap and HARD-FAILS with `ErrContentSizeExceeded` (regardless of `StripBlanks`) if it reaches the cap before any non-whitespace byte appears; indivisible STRUCTURAL token scans — tag name, end-tag name, attribute name, PUBLIC/SYSTEM DOCTYPE literal, intra-tag whitespace run (`scanTokenLimit`) — are also HARD-capped with `ErrContentSizeExceeded`, but against a separate cap FLOORED at the 16 MiB default (so small `MaxContentSize` never rejects ordinary names like `script`) that grows only when `MaxContentSize` exceeds the floor; `parseDoctype` checks `fatalErr` after EACH scanner so an over-cap run on a streaming reader fails promptly without a further blocking read; default 16 MiB)
- Terminal: **Parse(ctx, []byte)**, **ParseReader(ctx, io.Reader)**, **ParseFile(ctx, path)**, **ParseWithSAX(ctx, []byte, SAXHandler)**, **NewPushParser(ctx)**, **NewSAXPushParser(ctx, SAXHandler)**
- **NewWriter() → Writer** — create fluent writer builder
- Writer methods: `DefaultDTD(bool)`, `Format(bool)`, `PreserveCase(bool)`, `EscapeURIAttributes(bool)`, `EscapeControlChars(bool)`, `NullNamespaceHTMLOnly(bool)` (HTML 4.01 rule: a void element only in the null namespace is minimized; a namespaced one gets an explicit end tag), `CharacterMap(map[rune]string)` (substitute a mapped rune with its raw replacement in text/attribute content — Serialization 3.1 §7 character maps for the html output method; nil/empty is byte-identical)
- Terminal: **WriteTo(io.Writer, Node)**
- **Write(io.Writer, Node) → error** — serialize with default settings
- **WriteString(Node) → (string, error)** — serialize to string with default settings
- Auto-closing, void elements, implicit html/head/body insertion
- Encoding: prescan charset=utf-8 → U+FFFD for invalid bytes; otherwise Latin-1/Win-1252→UTF-8. `ParseReader`/push path: an UNDECLARED stream that keeps proving valid UTF-8 is deferred (buffered) until a non-UTF-8 byte flips the whole prefix to Windows-1252; that undecided prefix is BOUNDED at the configured `MaxContentSize` (16 MiB default), capped chunk-independently — valid UTF-8 ending at/below the cap is accepted (one-byte EOF probe), but the cap filling with more bytes still to come fails closed with `ErrContentSizeExceeded` (`encoding_reader.go`)
- Entity resolution: 2125 WHATWG + 106 legacy HTML4; legacy entities work without `;`
- Files: `html.go` (API), `parser.go`, `entities.go`, `elements.go`, `dump.go` (serializer), `tree.go` (DOM builder), `sax.go`
- Imports: helium, sax/

## xinclude/

XInclude 1.0 processing with recursive inclusion and fallback.

- **NewProcessor() → Processor** — create fluent builder
- Processor methods: `NoXIncludeMarkers()`, `NoBaseFixup()`, `Resolver(Resolver)`, `BaseURI(string)`, `MaxIncludeSize(int)`, `MaxIncludeDepth(int)`, `ErrorHandler(helium.ErrorHandler)`, `Parser(helium.Parser)`
- `Processor.Parser(helium.Parser)` — supplies the **resource limits** (depth/name-length/amplification/content-model-depth) used to parse included documents. XInclude still forces its own loading policy: external-DTD loading is on and the filesystem is confined to the `Resolver`'s sandbox (the injected parser's FS is NOT used for included docs — the `Resolver` is the security boundary). Unset → default `helium.NewParser()` base.
- Terminal: **Process(ctx, *Document) → (int, error)**, **ProcessTree(ctx, Node) → (int, error)**
- `Resolver` interface — custom resource loader; receives the href already resolved against the effective base (base arg is informational only — do NOT re-resolve, or the base directory is double-applied)
- **Secure by default**: an unset `Resolver` denies all filesystem access (`NewFSResolver(iofs.DenyAll{})`), mirroring `helium.NewParser()`'s deny-all FS — untrusted input cannot disclose local files via `<xi:include>`. Opt in with `Resolver(NewFSResolver(fsys))` (confined `fs.FS`, e.g. `os.Root.FS`) or `Resolver(NewFSResolver(helium.PermissiveFS()))` for historical os.Open passthrough. NOTE: `NewFSResolver(nil)` is still permissive — only the processor's *unset* default is deny-all
- `Processor.MaxIncludeSize(int)` — per-include byte cap; unset or ≤ 0 uses the default 10 MiB (unexported `defaultMaxIncludeSize`); over-cap reads fail with `ErrIncludeTooLarge`
- **Aggregate cap (internal, no public knob)**: across the whole expansion the cumulative materialized bytes are bounded at `maxIncludeAggregateMultiplier` (100) × the effective per-include cap (1 GiB by default; proportional, so lowering `MaxIncludeSize` lowers it), and the total spliced-resource count at `maxTotalIncludes` (65536). Counted per occurrence — repeated cache hits included — so many distinct sub-cap includes or one cached resource reused many times both trip it. An xpointer include charges the estimated footprint of each deep-copied selected subtree (`subtreeCopyCost`: `copiedNodeOverhead` per node + leaf content length, measured on the source before copying) against the same aggregate, so a small source whose xpointer selects many overlapping/nested nodes (O(n²) copies) is bounded instead of OOMing — the bytes READ from the source alone would not catch it. Over-aggregate fails with the same `ErrIncludeTooLarge` sentinel as the per-include cap. Guards amplification the per-include cap alone misses
- `Processor.MaxIncludeDepth(int)` — xi:include nesting-depth cap; unset or ≤ 0 uses the default 40 (unexported `defaultMaxIncludeDepth`); over-cap fails with "maximum include depth exceeded". Bounds nesting only — cyclic includes are caught separately by circular detection
- Default `NewFSResolver` converts absolute `file:` hrefs to OS paths via `internal/iofs.FileURIToPath` (non-local hosts rejected)
- Max URI 2000 chars, circular detection, doc/text caching
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

- **Compiler** (fluent, clone-on-write): `NewCompiler()` → `.Label(s)` / `.ErrorHandler(h)` / `.Parser(helium.Parser)` → `.Compile(ctx, doc)` or `.CompileFile(ctx, path)`. `Parser` sets the parser used by `CompileFile` (parse policy: limits, FS, XXE/network); unset → default `helium.NewParser()`
- **Validator** (fluent, clone-on-write): `NewValidator(schema)` → `.Label(s)` / `.Quiet()` / `.ErrorHandler(h)` → `.Validate(ctx, doc)`
- `ErrValidationFailed` — sentinel returned by `Validator.Validate` on validation failure; individual `*ValidationError` delivered to ErrorHandler
- `ErrNoSchema` — sentinel returned by `Validator.Validate` when the Validator has no compiled schema
- `ErrCompileFailed` — sentinel returned by `Compiler.Compile`/`CompileFile` when compilation fails
- Supports: schema, pattern, rule, assert, report, let, name, value-of
- Variable bindings via `<let>` and `<param>`
- Files: `schematron.go` (API + config), `schema.go` (data model), `parse.go` (compilation), `validate.go` (validation), `errors.go` (error types + formatting)
- Imports: helium, internal/xpath, xpath1/, internal/xpath1/number

## catalog/

OASIS XML Catalog resolution for public/system IDs and URIs.

- **Load(ctx, path) → (*Catalog, error)** — convenience wrapper around `NewLoader().Load`
- **NewLoader() → Loader** — fluent value-style loader; methods return updated copies
- **Loader.ErrorHandler(h) → Loader** — deliver parse warnings to a handler
- **Loader.MaxBytes(n) → Loader** — cap catalog file size; exceed → `ErrCatalogTooLarge` (default `MaxCatalogSize`, 10 MiB)
- **Catalog.Resolve(ctx, pubID, sysID) → string** — resolve external identifier
- **Catalog.ResolveURI(ctx, uri) → string** — resolve URI reference
- **Catalog.ResolveResult(ctx, pubID, sysID) → (uri string, broke bool)** / **Catalog.ResolveURIResult(ctx, uri) → (resolved string, broke bool)** — like Resolve/ResolveURI but also report a catalog break (the OASIS/libxml2 "cut" signal: a matching delegate was consulted and every delegate target failed). An exhausted nextCatalog chain is NOT a break — it returns `broke==false`. `broke==true` means "no match, STOP searching"; `broke==false` with `""` means "no match, keep searching". Chain callers (CLI `catalogChain`) honor `broke` to stop falling through to later catalogs
- Const `MaxCatalogSize`; sentinel `ErrCatalogTooLarge`
- Catalog chaining via nextCatalog; URN urn:publicid: support
- Files: `catalog.go`, `load.go`
- Imports: helium, internal/catalog/, internal/iofs/, internal/lexicon/, internal/xmlchar/

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
  - `SignatureAlgorithm(uri)`, `CanonicalizationMethod(uri)`, `Reference(ReferenceConfig)`, `KeyInfo(KeyInfoBuilder)`, `SignatureID(id)`, `AllowSHA1(bool)` — builder methods
  - `SignEnveloped(ctx, doc, parent, key)`, `SignEnveloping(ctx, doc, content, key)`, `SignDetached(ctx, doc, key)` — terminal methods
  - `SignEnveloping` wraps the content nodes in a `<ds:Object>`. A same-document reference (`URI="#id"`) pointing INTO the Signature's own Object content (e.g. a `<ds:Manifest>`/`<ds:SignatureProperties>` carrying an `Id`) resolves and is digested WITHOUT the Signature ever being inserted into the caller's document: reference resolution searches the document and the detached Signature subtree, an in-Object target is canonicalized on its own (the live Signature is moved into a throwaway document only for the c14n walk, rooted under a proxy element that reproduces the target's full inherited canonicalization context — the caller document element's in-scope namespace declarations AND its inherited xml:* attributes, copied per the C14N version to match exactly what helium's own canonicalizer inherits to a node-set apex (Canonical XML 1.0 inherits EVERY xml:* attribute including xml:id; Canonical XML 1.1 inherits only xml:lang/xml:space and lexically joins xml:base, xml:id NOT inherited) — so a reference into the Object verifies under inclusive Canonical XML 1.0 and 1.1 once the Signature is placed under the caller root; exclusive Canonical XML inherits neither namespaces nor xml:* and is unaffected; the proxy is never in the canonicalized node set, and the temporary move is always undone — on normal return, error, or a panic unwinding out of canonicalization), and a document target (`URI="#root"`, even the document element) is digested over its unchanged subtree — byte-identical to a signature with no such internal reference. An id that matches in BOTH the document and the Signature's own Object content is rejected as an ambiguous cross-tree collision (`ErrAmbiguousReference`).
  - Signing never mutates the caller's document. For a detached Signature (`SignDetached`/`SignEnveloping`, `sigElem.Parent()==nil`), `computeAndSetSignatureValue` (`sign.go`) canonicalizes `<ds:SignedInfo>` through the SAME throwaway-document proxy (`canonicalizeDetachedSubtree`, `transforms.go`) — the proxy reproduces the caller document element's in-scope namespaces + inherited xml:* per C14N version, so the SignedInfo digest is byte-identical to canonicalizing it while attached under `doc.DocumentElement()`, and the temporary move is undone on normal return, error, or panic. Enveloped signing (`SignEnveloped`) canonicalizes SignedInfo in place under the parent it was inserted into.
- **NewVerifier(KeySource) → Verifier** — create fluent builder for verification (clone-on-write value type)
  - `AllowSHA1(bool)` — builder method
  - `Verify(ctx, doc)`, `VerifyElement(ctx, doc, sigElem)` — terminal methods
- **NewEnvelopedReference() → ReferenceConfig** — SAML-optimized defaults (enveloped + ExcC14N + SHA-256)
- Key sources: `StaticKey(key)`, `X509CertKeySource(cert)`, `KeySourceFunc`
- Key info builders: `X509DataKeyInfo(certs...)`, `RSAKeyValueKeyInfo()`
- Transforms: `Enveloped()`, `C14NTransform(uri)`, `ExcC14NTransform(prefixes...)`
- Same-document reference (`URI="#id"`) resolution recognizes an attribute as an ID when it is DTD/schema-declared ID-typed (`enum.AttrID`), `xml:id`, or the `id` token in the casings `Id`/`ID`/`id`. This name set is FROZEN in `findElementsByID` (`transforms.go`) — distinct tokens (`wsu:Id`, SAML `AssertionID`) are not recognized by name; such documents must carry ID typing via schema. `>1` match → `ErrAmbiguousReference`.
- Reference transforms run as an ordered pipeline (node-set → octets): a c14n transform ends the pipeline, so a transform/2nd c14n ordered after it is rejected (`ErrUnsupportedTransform`); an omitted final transform defaults to **inclusive C14N 1.0** (not ExcC14N). `ec:InclusiveNamespaces` PrefixList on SignedInfo/CanonicalizationMethod is parsed and threaded into SignedInfo c14n; unknown CanonicalizationMethod parameters and any SignatureMethod child parameter (e.g. HMACOutputLength) are rejected fail-closed.
- Algorithms: RSA-SHA1/SHA256, ECDSA-SHA256/SHA384, HMAC-SHA1/SHA256, Ed25519
- Digests: SHA-1, SHA-256, SHA-384, SHA-512
- **SHA-1 rejected by default** (rsa-sha1/hmac-sha1/sha1) on both sign and verify → `ErrWeakAlgorithm`; opt in with `Signer.AllowSHA1(true)` / `Verifier.AllowSHA1(true)` for legacy interop. SHA-256+ unaffected.
- Errors: `ErrNoKeySource` sentinel — returned by verify when no usable KeySource is configured (nil cfg, untyped-nil, or typed-nil KeySource/func); `ErrWeakAlgorithm` — SHA-1 used without opt-in
- Files: `xmldsig1.go` (API), `constants.go`, `algorithms.go`, `weak_algorithm.go`, `sign.go`, `verify.go`, `transforms.go`, `keyinfo.go`, `errors.go`
- Imports: helium, c14n/

## xmlenc1/

XML Encryption 1.1 (W3C xmlenc-core1). Encrypt and decrypt XML elements/content.

- **NewEncryptor() → Encryptor** — create fluent builder for encryption (clone-on-write value type)
  - `BlockAlgorithm(uri)`, `AllowLegacyCBC(bool)`, `KeyTransportAlgorithm(uri)`, `RecipientPublicKey(key)`, `SessionKey(key)`, `KeyWrapAlgorithm(uri)`, `KeyEncryptionKey(kek)`, `OAEPDigest(uri)`, `OAEPMGF(uri)`, `OAEPParams(params)` — builder methods
  - `EncryptElement(ctx, elem)`, `EncryptContent(ctx, elem)` — terminal methods
- **NewDecryptor() → Decryptor** — create fluent builder for decryption
  - `PrivateKey(key)`, `KeyEncryptionKey(kek)`, `SessionKey(key)`, `AllowUnauthenticatedCBC(bool)`, `MaxEncryptedKeys(n)` — builder methods
  - `Decrypt(ctx, elem)` — terminal method
- `Decryptor.MaxEncryptedKeys(n)` caps trial-decrypted `<EncryptedKey>` candidates (DoS guard): zero → `DefaultMaxEncryptedKeys` (100), negative → unlimited; over-cap fails with `ErrTooManyEncryptedKeys` before any RSA op. The candidate loop also polls `ctx.Err()` between candidates
- Block encryption: AES-128/256-CBC, AES-128/256-GCM
- Secure by default: unset `BlockAlgorithm` → `DefaultBlockAlgorithm` (AES-256-GCM). Selecting a CBC block algorithm for **encryption** requires `Encryptor.AllowLegacyCBC(true)`, else `ErrCBCEncryptionRequiresOptIn`. **Decryption** of CBC requires `Decryptor.AllowUnauthenticatedCBC(true)`, else `ErrCBCRequiresOptIn`
- Key transport: RSA-OAEP (1.0 + 1.1 with configurable digest/MGF; the OAEP label digest and the MGF1 hash may differ, via `rsa.EncryptOAEPWithOptions`/`OAEPOptions` — requires Go ≥ 1.26)
- Key wrapping: AES-128/256-KeyWrap (RFC 3394)
- Key sizes are bound to the declared algorithm URI on encrypt and decrypt (incl. after unwrap/key transport); mismatch → `KeySizeError`
- Multi-recipient: `EncryptedData.EncryptedKeys []*EncryptedKey` holds one EncryptedKey per recipient; decrypt tries each candidate through full block decryption + plaintext validation (a bogus prepended key cannot mask the real one). `EncryptedData.EncryptedKey` is the **deprecated** single-key field — `EncryptedKeys` wins when non-empty, else the single field is treated as a one-element list; parse populates both
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
- Streamability helpers: `IsFnNamespacePrefix`, `StreamFnLocalName` (shared by xpath3/xslt3/xpathstream; normalizes EQName `Q{...}local` fn calls)
- Files: `ns.go`, `xml.go`, `catalog.go`, `xslt.go`, `fn.go`
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
- Constants: `Recover`, `NoEnt`, `DTDLoad`, `DTDAttr`, `DTDValid`, `NoError`, `NoWarning`, `Pedantic`, `NoBlanks`, `XInclude`, `NoNet`, `NoDict`, `NsClean`, `NoCDATA`, `NoXIncNode`, `Compact`, `NoBaseFix`, `IgnoreEnc`, `BigLines`, `NoXXE`, `NoUnzip`, `NoSysCatalog`, `CatalogPI`, `SkipIDs`, `LenientXMLDecl` (the `Huge`/`XML_PARSE_HUGE` bit is retired — replaced by the `Parser` per-limit knobs `MaxNameLength`/`MaxEntityAmplification`/`MaxContentModelDepth`)
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

- **Version10 / Version11** — lexical-rule selector for version-sensitive builtins
- **ValidateBuiltin(value, builtinLocal string, version Version) error** — validate value against an XSD builtin type lexical space under XSD 1.0 or 1.1 rules
- **Compare(a, b, builtinLocal string) (int, bool)** — type-aware comparison (-1/0/+1, ok)
- **CompareDecimal(a, b string) int** — decimal comparison via math/big.Rat (-2 on error)
- **CompareFloatFacetBound(a, b, builtinLocal string) (int, bool)** — float/double bound comparison ordering NaN as equal-to-NaN and greater-than-finite (schema-consistency check)
- **CanonicalKey(s, builtinLocal string) (string, bool)** — canonical value-space key (e.g. for enumeration de-dup)
- **WhiteSpace(builtinLocal string) string** — the type's XSD whiteSpace facet ("preserve"/"replace"/"collapse")
- **Normalize(s, builtinLocal string) string** — apply the type's whiteSpace facet to a lexical value
- **IsFloatNaN(s string) bool** — reports whether a float/double lexical is NaN
- **XSDFields(s string) []string** — split on XSD list whitespace
- **Orderable(builtinLocal string) bool** — whether the primitive value space is ordered (range facets may apply)
- **IsDecimalFamily(builtinLocal string) bool** — whether the type is xs:decimal or a derived integer (digit facets may apply)
- **LengthApplicable(builtinLocal string) bool** — whether length/minLength/maxLength facets apply and CONSTRAIN the value (string-derived, binary, anyURI, QName, NOTATION — enforced per XSD 1.0/libxml2 parity); shared by relaxng and xsd
- **CountTotalDigits(value string) int** — significant total-digit count for the totalDigits facet
- **CountFractionDigits(value string) int** — significant fraction-digit count for the fractionDigits facet
- Files: `validate.go`, `compare.go`, `facets.go`
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
