# Dependency Graph

Arrows show "imports" direction. Indented items are transitive.

```
cmd/heliumlint → helium, c14n, xsd, xinclude, xpath, catalog, internal/cliutil
shim           → helium, stream, internal/encoding
xinclude       → helium, xpointer
                  → xpath (via xpointer)
                  → internal/encoding
xsd            → helium, xpath
relaxng        → helium
schematron     → helium, xpath
xpointer       → helium, xpath
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
c14n, xpath, html, catalog, relaxng, stream

## Composition layer (depends on processing)
xsd (root + xpath), xpointer (root + xpath), schematron (root + xpath), xinclude (root + xpointer), shim (root + stream)

## Application layer
cmd/heliumlint (uses most packages)
