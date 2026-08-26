package helium

// UnsafeSetParentForTesting exposes the package-private raw parent-pointer
// setter to the external helium_test package. It compiles into the test binary
// only, so raw parent linkage stays unreachable from outside the package in an
// ordinary build. Tests use it to build a deliberately corrupt tree and
// exercise the traversal cycle guards.
func UnsafeSetParentForTesting(n Node, parent Node) {
	unsafeSetParent(n, parent)
}

// UnsafeAppendChildForTesting exposes the package-private no-preflight child
// append to the external helium_test package. It compiles into the test binary
// only, so linking a child without the cycle-guard and duplicate-attribute
// preflight stays unreachable from outside the package in an ordinary build.
func UnsafeAppendChildForTesting(parent MutableNode, child Node) error {
	return appendFastChild(parent, child)
}

// UnsafeSetNextSiblingForTesting exposes the package-private raw next-sibling
// pointer setter to the external helium_test package. It compiles into the test
// binary only, so raw sibling linkage stays unreachable from outside the
// package in an ordinary build. Tests use it to build a deliberately corrupt
// tree and exercise the traversal cycle guards.
func UnsafeSetNextSiblingForTesting(n Node, next Node) {
	unsafeSetNextSibling(n, next)
}
