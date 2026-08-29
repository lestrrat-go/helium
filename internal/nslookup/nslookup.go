// Package nslookup is an internal bridge that lets sibling packages in this
// module search helium namespace declarations without exposing the DOM's
// mutable namespace slice through the public API.
//
// Package helium installs both hooks in init. The hooks use any to avoid an
// import cycle: helium imports this package to register them, so this package
// must not import helium.
package nslookup

// PrefixURI walks a helium node and its ancestors for the nearest namespace
// declaration matching prefix. It returns the declaration's URI and whether a
// declaration was found. It does not synthesize the implicit xml binding.
var PrefixURI func(start any, prefix string) (string, bool)

// ByURI walks a helium node and its ancestors for the nearest namespace
// declaration matching uri. It returns a *helium.Namespace as any, plus whether
// a declaration was found. It does not synthesize the implicit xml binding.
var ByURI func(start any, uri string) (any, bool)

// DeclarationAt returns one namespace declaration from a helium element by
// index. It lets sibling packages stop before traversing or copying declarations
// beyond their own resource limits.
var DeclarationAt func(elem any, index int) (any, bool)
