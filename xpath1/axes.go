package xpath1

import (
	"context"

	helium "github.com/lestrrat-go/helium"
	ixpath "github.com/lestrrat-go/helium/internal/xpath"
)

// appendAxisMatches appends the nodes on axis from node that satisfy nodeTest
// to dst and returns the grown slice together with the number of nodes the
// axis traversal visited (the operation count the step must charge).
//
// The child, attribute, self and parent axes are walked straight into dst, so a
// step on those axes materializes neither the full candidate list nor a
// separate matched list. The remaining axes are appended to dst by
// [ixpath.AppendAxis] and then compacted in place. Traversal order, the
// per-node context checks, and the visited count are the same either way.
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

	// The remaining axes are collected into dst first and then compacted in
	// place, so the descendant walks still write into the caller's buffer
	// instead of a throwaway candidate slice.
	mark := len(dst)
	dst, err := ixpath.AppendAxis(ctx, dst, axis, node, maxNodeSetLength)
	if err != nil {
		return nil, 0, err
	}
	traversed := len(dst) - mark
	kept := mark
	for i := mark; i < len(dst); i++ {
		if !matchNodeTest(nodeTest, dst[i], axis, ec) {
			continue
		}
		if kept != i {
			dst[kept] = dst[i]
		}
		kept++
	}
	return dst[:kept], traversed, nil
}
