// Package c14n implements XML canonicalization (C14N) as defined by the W3C
// specifications: Canonical XML 1.0, Exclusive Canonical XML 1.0, and
// Canonical XML 1.1.
//
// Use [NewCanonicalizer] with a [Mode] to create a canonicalizer, then
// configure it with fluent builder methods:
//
//	out, err := c14n.NewCanonicalizer(c14n.ExclusiveC14N10).
//	    Comments().
//	    InclusiveNamespaces([]string{"ns1"}).
//	    CanonicalizeTo(doc)
//
// # Builder Design
//
// Boolean toggles like [Canonicalizer.Comments] are parameterless methods
// because the builder starts from defaults (comments excluded, full document)
// and callers only need to opt into non-default behavior. The method name
// is self-documenting: calling Comments means "include comments."
package c14n
