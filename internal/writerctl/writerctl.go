// Package writerctl is an internal bridge that lets sibling packages in this
// module reach unexported helium.Writer configuration without public methods
// on the writer.
//
// xpath3 fn:serialize uses declaration-only encoding because its result is a
// string. xslt3 uses document serialization without the helium.Writer newlines
// normally added after each top-level child. Package helium installs both hooks
// in its init.
//
// Hooks are typed with any (rather than helium.Writer) to avoid an import cycle:
// package helium imports this package to register them, so this package must not
// import helium.
package writerctl

// EnableDeclarationOnlyEncoding returns a copy of the given helium.Writer (both
// argument and result are helium.Writer, passed as any) with declaration-only
// encoding enabled. Package helium installs it in init; it is non-nil whenever
// package helium is linked in, which every caller of this hook transitively is.
var EnableDeclarationOnlyEncoding func(w any) any

// OmitDocumentChildTerminators returns a copy of the given helium.Writer (both
// argument and result are helium.Writer, passed as any) that does not append a
// newline after each serialized Document child. Package helium installs it in
// init; it is non-nil whenever package helium is linked in.
var OmitDocumentChildTerminators func(w any) any
