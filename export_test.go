package helium

// UnsafeSetParentForTesting exposes the package-private raw parent-pointer
// setter to the external helium_test package. It compiles into the test binary
// only, so raw parent linkage stays unreachable from outside the package in an
// ordinary build. Tests use it to build a deliberately corrupt tree and
// exercise the traversal cycle guards.
func UnsafeSetParentForTesting(n Node, parent Node) {
	unsafeSetParent(n, parent)
}
