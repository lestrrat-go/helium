// Package nodelink is an internal bridge that lets sibling packages in this
// module reach unexported helium node-linkage operations without adding public
// functions to package helium.
//
// xslt3's strip-space copier links freshly built nodes without the per-node
// cycle and duplicate-attribute preflight that helium.MutableNode.AddChild
// runs. The cycle-guard tests in xsd and xmldsig1 build deliberately corrupt
// sibling chains. Package helium installs all three hooks in its init.
//
// Hooks are typed with any, in place of helium.Node, to avoid an import cycle:
// package helium imports this package to register them, so this package must
// not import helium.
package nodelink

// AppendFastChild links child (a helium.Node, passed as any) as the last child
// of parent (a helium.MutableNode, passed as any) without the cycle-guard and
// duplicate-attribute preflight that AddChild performs. The CALLER guarantees a
// child that cannot create a cycle or a duplicate attribute, such as a freshly
// constructed node in a deep copy. Package helium installs it in init; it is
// non-nil whenever package helium is linked in, which every caller of this hook
// transitively is.
var AppendFastChild func(parent, child any) error

// CorruptSelfNextSibling writes n.next = n, making the node its own next
// sibling. It is a raw pointer write with no cycle detection and no reciprocal
// back-pointer maintenance, and it exists ONLY so this module's tests can build
// a corrupt tree that exercises the traversal cycle guards. Nothing else may
// use it. Package helium installs it in init.
var CorruptSelfNextSibling func(n any)

// CorruptTypedNilNextSibling writes n.next = a typed-nil *helium.Element,
// making the node's next sibling a non-nil interface holding a nil pointer. It
// is a raw pointer write with no cycle detection and no reciprocal
// back-pointer maintenance, and it exists ONLY so this module's tests can build
// a corrupt tree that exercises the typed-nil traversal paths. Nothing else may
// use it. Package helium installs it in init.
var CorruptTypedNilNextSibling func(n any)
