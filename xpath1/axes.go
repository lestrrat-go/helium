package xpath1

import (
	"context"

	helium "github.com/lestrrat-go/helium"
	ixpath "github.com/lestrrat-go/helium/internal/xpath"
)

// traverseAxis returns the nodes along the given axis from the context node,
// in the order defined by the XPath spec. ctx is checked inside the unbounded
// descendant/following/preceding walks so a cancelled context aborts traversal
// promptly.
func traverseAxis(ctx context.Context, axis AxisType, node helium.Node) ([]helium.Node, error) {
	return ixpath.TraverseAxis(ctx, axis, node, maxNodeSetLength)
}
