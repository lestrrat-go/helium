# Dependency Graph

Arrows show "imports" direction. Indented items are transitive.

```
cmd/helium     → internal/cli/heliumcmd
internal/cli/heliumcmd → helium, c14n, relaxng, schematron, xsd, xinclude, xpath1, xpath3, catalog, internal/cliutil, internal/uripath, internal/iofs
shim           → helium, stream, enum, internal/encoding, internal/xmlchar
xinclude       → helium, xpointer, internal/encoding, internal/iofs, internal/lexicon, internal/uripath
                  → xpath1 (via xpointer)
                  → internal/xmlchar (via xpointer)
                  (helium.Parser.XInclude injects an xinclude.Processor through the helium.XIncludeProcessor interface — dependency inversion keeps this edge one-way; helium does NOT import xinclude)
xpath1         → helium, internal/lexicon, internal/domutil
xpath3         → helium, internal/xpath, internal/lexicon, internal/icu, internal/unparsedtext, internal/strcursor, internal/sequence, internal/xsdregex, internal/xmlchar, internal/domutil, internal/writerctl
xslt3          → helium, xpath3, xsd, html, internal/iofs, internal/lexicon, internal/nodelink, internal/sequence, internal/uripath, internal/xpathstream, internal/domutil, internal/writerctl, xslt3/internal/elements
xsd            → helium, xpath1, xpath3, internal/lexicon, internal/xsd/value, internal/xsdregex, internal/uripath, internal/iofs
relaxng        → helium, internal/lexicon, internal/iofs, internal/iolimit, internal/xsd/value, internal/xsdregex, internal/xmlchar, internal/uripath
schematron     → helium, xpath1, xpath3, internal/xpath, internal/xpath1/number
xpointer       → helium, xpath1, internal/xmlchar
c14n           → helium, internal/lexicon, internal/domutil
xmldsig1       → helium, c14n, xpath1, internal/lexicon, internal/domutil, internal/xmlbase64
xmldsig1/transform → helium, xmldsig1, xslt3  (opt-in xslt3-backed XSLTTransformer; kept out of xmldsig1 so the core never imports xslt3)
xmlenc1        → helium, c14n, internal/domutil, internal/xmlbase64  (c14n converts the node-set a same-document xenc:CipherReference names into octets; c14n imports neither xmlenc1 nor xmldsig1, so the edge is one-way)
html           → helium, sax, push, internal/xmlchar
catalog        → helium, internal/catalog, internal/iofs, internal/lexicon, internal/xmlchar
stream         → internal/encoding, internal/xmlchar
sax            → helium, enum
helium (root)  → sax, enum, internal/encoding, internal/bitset, internal/parser, push, internal/stack, internal/uripath, internal/iofs, internal/writerctl, internal/nodelink
                  (helium installs hooks in internal/writerctl in init so xpath3 fn:serialize can enable declaration-only encoding and xslt3 can omit document-child terminators without public methods; the writerctl package imports nothing, so the edge is one-way)
                  (helium installs hooks in internal/nodelink in init so xslt3's strip-space copier can link a freshly built child without AddChild's preflight, and the xsd/xmldsig1 corrupt-tree cycle-guard tests can write a raw next-sibling pointer, all without public functions in helium; the nodelink package imports nothing, so the edge is one-way)
sink           → (none)
enum           → (none)
internal/lexicon → (none)
internal/icu   → (none)
push → (none)
internal/heliumtest → (none)
internal/sequence → (none)
internal/strcursor → (none)
internal/unparsedtext → internal/xmlchar, internal/uripath, internal/iofs
internal/catalog → internal/uripath
internal/uripath → (none)
internal/xsdregex → (none)
internal/xmlbase64 → helium (DecodeElement counts/charges/decodes a base64 value straight off its DOM children; the root never imports it back)
internal/writerctl → (none)
internal/nodelink → (none)
internal/xsd/value → internal/lexicon
internal/domutil → helium, enum, internal/lexicon, internal/xmlchar
internal/xpathstream → xpath3, internal/lexicon
test           → helium
```

## Leaf packages (no helium deps)
sink, enum, internal/bitset, internal/heliumtest, internal/parser, push, internal/stack, internal/cliutil, internal/encoding, internal/lexicon, internal/icu, internal/nodelink, internal/sequence, internal/strcursor, internal/writerctl, internal/xsdregex, internal/uripath

## Core layer
helium (root) → sax, enum, internal/*

## Processing layer (depends on root)
c14n, xpath1, xpath3, html, catalog, relaxng, stream

## Security layer (depends on processing)
xmldsig1 (root + c14n + xpath1 + internal/lexicon; xpath1 backs the XPath filter transform), xmlenc1 (root + c14n; c14n converts the node-set a same-document xenc:CipherReference names into octets)

## Composition layer (depends on processing)
xsd (root + xpath1 + xpath3 + internal/lexicon), xpointer (root + xpath1 + internal/xmlchar), schematron (root + xpath1 + xpath3 + internal/xpath + internal/xpath1/number; xpath1 backs the XPath 1.0 queryBinding, xpath3 the XPath 3.1 one), xinclude (root + xpointer + internal/encoding + internal/iofs + internal/lexicon), xslt3 (root + xpath3 + xsd + html + internal/elements), shim (root + stream)

## Application layer
internal/cli/heliumcmd (CLI implementation)
cmd/helium (CLI wrapper)
