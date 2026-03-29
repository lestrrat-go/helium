# Dependency Graph

Arrows show "imports" direction. Indented items are transitive.

```
cmd/helium     → internal/cli/heliumcmd
internal/cli/heliumcmd → helium, c14n, relaxng, schematron, xsd, xinclude, xpath1, xpath3, catalog, internal/cliutil
shim           → helium, stream, enum, internal/encoding
xinclude       → helium, xpointer
                  → xpath1 (via xpointer)
                  → internal/encoding
xpath3         → helium, internal/xpath, internal/lexicon, internal/icu, internal/unparsedtext, internal/strcursor, internal/sequence
xslt3          → helium, xpath3, xsd, html, internal/lexicon, internal/sequence, internal/xpathstream, xslt3/internal/elements
xsd            → helium, xpath1, internal/lexicon, internal/xsd/value
relaxng        → helium
schematron     → helium, xpath1
xpointer       → helium, xpath1
c14n           → helium
html           → helium, sax, push
catalog        → helium, internal/catalog, internal/lexicon
stream         → internal/encoding
sax            → helium, enum
helium (root)  → sax, enum, internal/encoding, internal/bitset, internal/parser, push, internal/stack
sink           → (none)
enum           → (none)
internal/lexicon → (none)
internal/icu   → (none)
push → (none)
internal/sequence → (none)
internal/strcursor → (none)
internal/unparsedtext → (none)
internal/xpathstream → xpath3
test           → helium
```

## Leaf packages (no helium deps)
sink, enum, internal/bitset, internal/parser, push, internal/stack, internal/cliutil, internal/catalog, internal/encoding, internal/lexicon, internal/icu, internal/sequence, internal/strcursor, internal/unparsedtext, internal/xsd/value

## Core layer
helium (root) → sax, enum, internal/*

## Processing layer (depends on root)
c14n, xpath1, xpath3, html, catalog, relaxng, stream

## Composition layer (depends on processing)
xsd (root + xpath1 + internal/lexicon), xpointer (root + xpath1), schematron (root + xpath1), xinclude (root + xpointer), xslt3 (root + xpath3 + xsd + html + internal/elements), shim (root + stream)

## Application layer
internal/cli/heliumcmd (CLI implementation)
cmd/helium (CLI wrapper)
