# Dependency Graph

Arrows show "imports" direction. Indented items are transitive.

```
cmd/helium     → helium, c14n, relaxng, xsd, xinclude, xpath1, xpath3, catalog, internal/cliutil
shim           → helium, stream, internal/encoding
xinclude       → helium, xpointer
                  → xpath1 (via xpointer)
                  → internal/encoding
xpath3         → helium, internal/xpath
xsd            → helium, xpath1
relaxng        → helium
schematron     → helium, xpath1
xpointer       → helium, xpath1
c14n           → helium
html           → helium, sax
catalog        → helium, internal/catalog
stream         → internal/encoding
sax            → helium
helium (root)  → sax, enum, internal/catalog, internal/encoding, internal/bitset, internal/stack
sink           → (none)
enum           → (none)
test           → helium
```

## Leaf packages (no helium deps)
sink, enum, internal/bitset, internal/stack, internal/cliutil

## Core layer
helium (root) → sax, enum, internal/*

## Processing layer (depends on root)
c14n, xpath1, xpath3, html, catalog, relaxng, stream

## Composition layer (depends on processing)
xsd (root + xpath1), xpointer (root + xpath1), schematron (root + xpath1), xinclude (root + xpointer), shim (root + stream)

## Application layer
cmd/helium (CLI + lint implementation)
