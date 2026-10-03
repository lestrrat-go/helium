// Package nodecontent is an internal bridge that lets sibling packages in this
// module read a helium node's content without the defensive copy
// helium.Node.Content makes.
//
// c14n writes every text node, comment and attribute value of a document, and
// copying each one before escaping it into the output is most of what the
// canonicalizer allocates. Package helium installs the hook in its init.
//
// The hook is typed with any, in place of helium.Node, to avoid an import
// cycle: package helium imports this package to register it, so this package
// must not import helium.
package nodecontent

// Raw returns the content of n (a helium.Node, passed as any). For a Text,
// CDATASection or Comment node the result is the node's own storage, not a
// copy: the CALLER must not modify it, and must not use it after the node is
// changed. For any other node it is a copy, as helium.Node.Content returns.
// Package helium installs it in init; it is non-nil whenever package helium is
// linked in, which every caller of this hook transitively is.
var Raw func(n any) []byte
