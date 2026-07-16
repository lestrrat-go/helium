// Package writerctl is an internal bridge that lets sibling packages in this
// module reach unexported helium.Writer configuration without a public method
// on the writer.
//
// It exists for a single consumer: xpath3 fn:serialize, whose string result
// treats the serialization encoding parameter as declaration-only (W3C
// Serialization) — the encoding labels the XML declaration but the octets stay
// UTF-8, with non-representable characters emitted as character references.
// Package helium installs EnableDeclarationOnlyEncoding in its init.
//
// The hook is typed with any (rather than helium.Writer) to avoid an import
// cycle: package helium imports this package to register the hook, so this
// package must not import helium.
package writerctl

// EnableDeclarationOnlyEncoding returns a copy of the given helium.Writer (both
// argument and result are helium.Writer, passed as any) with declaration-only
// encoding enabled. Package helium installs it in init; it is non-nil whenever
// package helium is linked in, which every caller of this hook transitively is.
var EnableDeclarationOnlyEncoding func(w any) any
