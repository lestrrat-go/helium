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

// appendAxisMatches appends the nodes on axis from node that satisfy nodeTest
// to dst and returns the grown slice together with the number of nodes the
// axis traversal visited (the operation count the step must charge).
//
// The child, attribute, self and parent axes are walked straight into dst, so a
// step on those axes materializes neither the full candidate list nor a
// separate matched list. The remaining axes fall back to traverseAxis and
// filter its result into dst. Traversal order, the per-node context checks, and
// the visited count are the same either way.
func appendAxisMatches(ctx context.Context, dst []helium.Node, ec *evalContext, node helium.Node, axis AxisType, nodeTest NodeTest) ([]helium.Node, int, error) {
	if ixpath.IsNilNode(node) {
		return dst, 0, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}
	switch axis {
	case AxisChild:
		// In XPath, attributes have no children.
		if _, ok := node.(*helium.Attribute); ok {
			return dst, 0, nil
		}
		traversed := 0
		for child := range helium.Children(node) {
			if err := ctx.Err(); err != nil {
				return nil, 0, err
			}
			if !ixpath.IsXDMChild(child) {
				continue
			}
			traversed++
			if matchNodeTest(nodeTest, child, axis, ec) {
				dst = append(dst, child)
			}
		}
		return dst, traversed, nil
	case AxisAttribute:
		elem, ok := node.(*helium.Element)
		if !ok {
			return dst, 0, nil
		}
		traversed := 0
		var ctxErr error
		elem.ForEachAttribute(func(attr *helium.Attribute) bool {
			if err := ctx.Err(); err != nil {
				ctxErr = err
				return false
			}
			traversed++
			if matchNodeTest(nodeTest, attr, axis, ec) {
				dst = append(dst, attr)
			}
			return true
		})
		if ctxErr != nil {
			return nil, 0, ctxErr
		}
		return dst, traversed, nil
	case AxisSelf:
		if matchNodeTest(nodeTest, node, axis, ec) {
			dst = append(dst, node)
		}
		return dst, 1, nil
	case AxisParent:
		parent := node.Parent()
		if parent == nil {
			return dst, 0, nil
		}
		if matchNodeTest(nodeTest, parent, axis, ec) {
			dst = append(dst, parent)
		}
		return dst, 1, nil
	}

	candidates, err := traverseAxis(ctx, axis, node)
	if err != nil {
		return nil, 0, err
	}
	for _, candidate := range candidates {
		if matchNodeTest(nodeTest, candidate, axis, ec) {
			dst = append(dst, candidate)
		}
	}
	return dst, len(candidates), nil
}
