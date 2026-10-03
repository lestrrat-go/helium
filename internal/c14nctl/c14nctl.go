// Package c14nctl is an internal bridge that lets sibling packages in this
// module reach unexported c14n.Canonicalizer configuration without public
// methods on the canonicalizer.
//
// xmldsig1 canonicalizes the node set of one element subtree (a SignedInfo or
// a same-document #id reference) and uses SubtreeRoot so the walk starts at
// that subtree instead of visiting every element of the document. Package
// c14n installs the hook in its init.
//
// The hook is typed with any, in place of c14n.Canonicalizer, to avoid an
// import cycle: package c14n imports this package to register it, so this
// package must not import c14n.
package c14nctl

// SubtreeRoot returns a copy of the given c14n.Canonicalizer (argument and
// result are c14n.Canonicalizer, passed as any) whose node-set walk starts at
// root (a *helium.Element, passed as any) instead of at the document. The
// CALLER guarantees that every member of the configured node set lies in
// root's subtree, root included, or in entity replacement content referenced
// from it; canonicalization then produces the same bytes and the same errors
// as the whole-document walk. The canonicalizer still checks every element of
// the document for a relative namespace URI, so a document the whole-document
// walk rejects is still rejected. It has no effect without a node set, and
// falls back to the whole-document walk when root lies inside entity
// replacement content, inside an excluded subtree, outside the canonicalized
// document, or under an ancestor that has members in the node set, and when
// the document declares an entity with element content. Package c14n installs
// it in init; it is non-nil whenever package c14n is linked in, which every
// caller of this hook transitively is.
var SubtreeRoot func(c, root any) any
