# Dependency Graph

Arrows show "imports" direction. Indented items are transitive.

```
cmd/helium     → internal/cli/heliumcmd
internal/cli/heliumcmd → helium, c14n, relaxng, schematron, xsd, xinclude, xpath1, xpath3, catalog, internal/cliutil
shim           → helium, stream, enum, internal/encoding, internal/xmlchar
xinclude       → helium, xpointer, internal/encoding, internal/iofs, internal/lexicon
                  → xpath1 (via xpointer)
                  → internal/xmlchar (via xpointer)
xpath3         → helium, internal/xpath, internal/lexicon, internal/icu, internal/unparsedtext, internal/strcursor, internal/sequence, internal/xsdregex, internal/xmlchar
xslt3          → helium, xpath3, xsd, html, internal/lexicon, internal/sequence, internal/xpathstream, xslt3/internal/elements
xsd            → helium, xpath1, internal/lexicon, internal/xsd/value, internal/xsdregex
relaxng        → helium, internal/lexicon, internal/iofs, internal/xsd/value, internal/xmlchar
schematron     → helium, xpath1
xpointer       → helium, xpath1, internal/xmlchar
c14n           → helium
xmldsig1       → helium, c14n, internal/lexicon
xmlenc1        → helium
html           → helium, sax, push, internal/xmlchar
catalog        → helium, internal/catalog, internal/lexicon, internal/xmlchar
stream         → internal/encoding, internal/xmlchar
sax            → helium, enum
helium (root)  → sax, enum, internal/encoding, internal/bitset, internal/parser, push, internal/stack
sink           → (none)
enum           → (none)
internal/lexicon → (none)
internal/icu   → (none)
push → (none)
internal/heliumtest → (none)
internal/sequence → (none)
internal/strcursor → (none)
internal/unparsedtext → internal/xmlchar
internal/xsdregex → (none)
internal/xsd/value → internal/lexicon
internal/xpathstream → xpath3
test           → helium
```

## Leaf packages (no helium deps)
sink, enum, internal/bitset, internal/heliumtest, internal/parser, push, internal/stack, internal/cliutil, internal/catalog, internal/encoding, internal/lexicon, internal/icu, internal/sequence, internal/strcursor, internal/unparsedtext, internal/xsdregex

## Core layer
helium (root) → sax, enum, internal/*

## Processing layer (depends on root)
c14n, xpath1, xpath3, html, catalog, relaxng, stream, xmlenc1

## Security layer (depends on processing)
xmldsig1 (root + c14n + internal/lexicon)

## Composition layer (depends on processing)
xsd (root + xpath1 + internal/lexicon), xpointer (root + xpath1 + internal/xmlchar), schematron (root + xpath1), xinclude (root + xpointer + internal/encoding + internal/iofs + internal/lexicon), xslt3 (root + xpath3 + xsd + html + internal/elements), shim (root + stream)

## Application layer
internal/cli/heliumcmd (CLI implementation)
cmd/helium (CLI wrapper)
