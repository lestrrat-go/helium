# libxml2 Parity Status

## Test Results

| Package | Tests | Pass | Skip | Rate | Skip Reasons |
|---------|-------|------|------|------|--------------|
| Core XML (DOM) | 150+ | all | 0 | ~100% | — |
| Core XML (SAX2) | 150+ | all | 0 | ~100% | — |
| C14N | 76 | 66 | 10 | 87% | Parser: duplicate xmlns (6), entity-ref in single-quoted attr (3), missing expected files (3) |
| XSD | 226 | 225 | 1 | 99.6% | libxml2 IDC quirk with ref + attributeFormDefault |
| RELAX NG | 159 | 159 | 0 | 100% | — |
| Schematron | 42 | 42 | 0 | 100% | — |
| HTML SAX | 47 | 47 | 0 | 100% | — |
| HTML Serialization | 47 | 47 | 0 | 100% | — |
| HTML Errors | 5 | 4 | 1 | 80% | encoding-error.html: byte-level context window |

Test data: `testdata/libxml2-compat/` (golden files generated from libxml2's xmllint).

## Parser Limitations

These affect multiple packages (especially C14N test skips):

1. **Duplicate namespace declarations** — helium rejects, libxml2 uses last. Affects 6 C14N tests.
2. **Entity refs in single-quoted attributes** — not expanded. Affects 3 C14N tests.
3. **External entity resolution** — limited; requires explicit config. ParseNoXXE blocks all.

## Feature Status

### Fully Implemented

| Feature | Notes |
|---------|-------|
| XML parsing | Well-formedness, namespace handling, encoding detection/transcoding |
| SAX2 interface | All callbacks: startElementNS, endElementNS, characters, cDataBlock, comment, PI, DTD events, entity/notation/element/attribute decls |
| DOM tree | All node types: Element, Attribute, Text, CDATA, Comment, PI, Document, DTD, Entity, EntityRef, Notation |
| Namespaces | Declarations, prefix resolution, namespace nodes |
| DTD | Internal subsets, external subsets (limited), entity/notation/element/attribute decls |
| Encoding | Auto-detection, BOM, UTF-8/16, ISO-8859-*, Windows-*, CJK, EBCDIC, UCS-4 |
| Tree ops | AppendChild, InsertBefore, RemoveChild, ReplaceChild, CopyNode, Walk |
| XPath 1.0 | Full expression eval, all 13 axes, 27+ functions, custom function registration |
| C14N | All 3 modes (1.0, Exclusive 1.0, 1.1), comments, node-set, inclusive NS, xml:* inheritance |
| XSD | Complex/simple types, all compositors, facets, IDC (key/unique/keyref), substitution groups, import/include, xsi:type/nil |
| RELAX NG | All patterns, name classes, include/override, externalRef, parentRef, data types, interleave, backtracking |
| Schematron | Assert, report, let variables, name, value-of, old (ASCC) + new (ISO) syntax |
| XInclude | Recursive inclusion, fallback, marker nodes, base URI fixup, circular detection |
| Catalog | OASIS XML Catalog, public/system ID resolution, URI resolution, catalog chaining, URN support |
| XML Writer | Streaming output, namespace scopes, indentation, DTD internal subsets, self-close optimization |
| Serialization | WriteDoc, WriteNode, formatted output, encoding handling |

### Partial / Limited

| Feature | What Works | Gap |
|---------|-----------|-----|
| HTML parsing | SAX + DOM, auto-close, void elements, entities, encoding | Structural element nesting not enforced, areBlanks heuristic simpler, attribute deduplication missing |
| encoding/xml shim | Marshal, Unmarshal, Encoder, Decoder, Token, struct tags | Strict-only, xmlns before regular attrs, InputOffset approximate, undeclared prefixes rejected |
| XSD numeric comparison | decimal/integer via big.Rat | No float/double/date/time/duration comparison |
| XSD validation mode | DOM-only | No SAX/streaming validation, no subtree validation |
| Push parser | Buffers all data then parses | Not true incremental |

### Not Implemented

| Feature | libxml2 Equivalent | Notes |
|---------|-------------------|-------|
| XSLT | libxslt | Out of scope for v1 |
| Reader API | xmlTextReader | No pull-parser equivalent |
| Pattern API | xmlPattern | No compiled pattern matching |
| SAX/streaming validation | xmlSchemaSAXPlug | XSD/RELAX NG are DOM-only |
| Custom I/O callbacks | xmlIO | Uses io.Reader/io.Writer directly |
| Automata/Regexp | xmlAutomata, xmlRegexp | Go regexp replaces |
| Global state | xmlInitParser/xmlCleanupParser | Not needed in Go |
| Memory management | xmlMalloc/xmlFree | Go GC replaces |

## ParseOption Parity

| Option | Status | Notes |
|--------|--------|-------|
| ParseRecover | ✅ | Recovery mode on errors |
| ParseNoEnt | ✅ | Substitute entities |
| ParseDTDLoad | ✅ | Load external subsets |
| ParseDTDAttr | ✅ | Default DTD attributes |
| ParseDTDValid | ✅ | Validate with DTD |
| ParseNoError | ✅ | Suppress error reports |
| ParseNoWarning | ✅ | Suppress warnings |
| ParsePedantic | ✅ | Pedantic error reporting |
| ParseNoBlanks | ✅ | Remove blank nodes |
| ParseXInclude | ✅ | XInclude processing |
| ParseNoNet | ✅ | Forbid network access |
| ParseNsClean | ✅ | Remove redundant NS decls |
| ParseNoCDATA | ✅ | Merge CDATA as text |
| ParseNoXIncNode | ✅ | Skip XInclude markers |
| ParseNoBaseFix | ✅ | Skip xml:base fixup |
| ParseHuge | ✅ | Relax limits |
| ParseIgnoreEnc | ✅ | Ignore encoding hint |
| ParseNoXXE | ✅ | Block XXE attacks |
| ParseSkipIDs | ✅ | Skip ID interning |
| ParseLenientXMLDecl | ✅ | Helium extension: relaxed XML decl attribute order |
| ParseCompact | no-op | Go memory model |
| ParseBigLines | no-op | Go ints are 64-bit |
| ParseNoUnzip | no-op | No decompression support |
| ParseNoSysCatalog | no-op | No global catalog |
| ParseCatalogPI | no-op | Not yet implemented |

## Known Issues

- `xinclude/issue733` — DOCTYPE not preserved after XInclude processing
- C14N relative namespace URI check uses heuristic (`!strings.Contains(uri, ":")`) not full URI parse
- HTML attribute deduplication: all kept (libxml2 keeps first)
- HTML areBlanks heuristic simpler than libxml2's

## Intentional Divergences

These are architectural choices, not bugs:

- Go error returns vs C integer return codes + xmlRaiseError
- Go GC vs malloc/free/reference counting
- Package splitting: single .c file → entire Go package
- Go interfaces for node types vs xmlNode.type enum switch
- No global state: explicit context passing
- Functional options (WithX()) vs bitmask flags
- Push parser buffers then parses (vs true incremental)
- Namespace stack: frame-based visibleNSStack vs flat arrays
