package xpath1

import (
	helium "github.com/lestrrat-go/helium"
	ixpath "github.com/lestrrat-go/helium/internal/xpath"
)

// traverseAxis returns the nodes along the given axis from the context node,
// in the order defined by the XPath spec.
func traverseAxis(axis AxisType, node helium.Node) ([]helium.Node, error) {
	return ixpath.TraverseAxis(axis, node, maxNodeSetLength)
}
