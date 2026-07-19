package xpath_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	helium "github.com/lestrrat-go/helium"
	ixpath "github.com/lestrrat-go/helium/internal/xpath"
	"github.com/stretchr/testify/require"
)

func deepChainDepth(t *testing.T) int {
	t.Helper()

	if testing.Short() {
		return 512
	}
	return 5000
}

func buildDeepChain(t *testing.T, depth int) (*helium.Document, *helium.Element, *helium.Element) {
	t.Helper()

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root, err := doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, doc.AddChild(root))

	parent := root
	// go.mod requires Go 1.25, so integer range is part of the supported toolchain.
	for range depth {
		child, err := doc.CreateElement("level")
		require.NoError(t, err)
		require.NoError(t, parent.AddChild(child))
		parent = child
	}

	return doc, root, parent
}

func TestTraverseAxisDescendant_DeepChain(t *testing.T) {
	depth := deepChainDepth(t)

	_, root, leaf := buildDeepChain(t, depth)

	nodes, err := ixpath.TraverseAxis(t.Context(), ixpath.AxisDescendant, root, ixpath.DefaultMaxNodeSetLength)
	require.NoError(t, err)
	require.Len(t, nodes, depth)
	require.Equal(t, helium.Node(leaf), nodes[len(nodes)-1])
}

func TestTraverseAxisPreceding_DeepChain(t *testing.T) {
	depth := deepChainDepth(t)

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root, err := doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, doc.AddChild(root))

	left, err := doc.CreateElement("left")
	require.NoError(t, err)
	require.NoError(t, root.AddChild(left))

	parent := left
	var leaf helium.Node = left
	// go.mod requires Go 1.25, so integer range is part of the supported toolchain.
	for range depth {
		child, err := doc.CreateElement("level")
		require.NoError(t, err)
		require.NoError(t, parent.AddChild(child))
		parent = child
		leaf = child
	}

	right, err := doc.CreateElement("right")
	require.NoError(t, err)
	require.NoError(t, root.AddChild(right))

	nodes, err := ixpath.TraverseAxis(t.Context(), ixpath.AxisPreceding, right, ixpath.DefaultMaxNodeSetLength)
	require.NoError(t, err)
	require.Len(t, nodes, depth+1)
	require.Equal(t, leaf, nodes[0])
	require.Equal(t, helium.Node(left), nodes[len(nodes)-1])
}

// TestTraverseAxisDescendant_ContextCancelled verifies that a context cancelled
// before traversal aborts the descendant walk promptly with context.Canceled
// instead of walking the whole subtree.
func TestTraverseAxisDescendant_ContextCancelled(t *testing.T) {
	// A wide tree so that, absent a context check, traversal would visit a
	// large number of nodes before returning.
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root, err := doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, doc.AddChild(root))

	parent := root
	for range 5000 {
		child, err := doc.CreateElement("level")
		require.NoError(t, err)
		require.NoError(t, parent.AddChild(child))
		parent = child
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	nodes, err := ixpath.TraverseAxis(ctx, ixpath.AxisDescendant, root, ixpath.DefaultMaxNodeSetLength)
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, nodes)
}

// cancelAfterNContext reports a cancelled error from Err() only after Err has
// been consulted cancelAfter times, simulating a context that is cancelled
// AFTER traversal has begun (i.e. partway through the walk). It implements
// context.Context directly (no embedding) to satisfy the containedctx linter.
type cancelAfterNContext struct {
	cancelAfter int
	calls       int
}

func (c *cancelAfterNContext) Deadline() (time.Time, bool) { return time.Time{}, false }
func (c *cancelAfterNContext) Done() <-chan struct{}       { return nil }
func (c *cancelAfterNContext) Value(any) any               { return nil }

func (c *cancelAfterNContext) Err() error {
	c.calls++
	if c.calls > c.cancelAfter {
		return context.Canceled
	}
	return nil
}

// TestTraverseAxisDescendant_ContextCancelledMidWalk verifies that a context
// cancelled AFTER the descendant traversal has begun aborts the walk promptly
// rather than completing the full walk over a deep tree. Without an in-loop
// ctx.Err() check the walk would visit all depth nodes regardless.
func TestTraverseAxisDescendant_ContextCancelledMidWalk(t *testing.T) {
	const depth = 20000
	const cancelAfter = 10

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root, err := doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, doc.AddChild(root))

	parent := root
	for range depth {
		child, err := doc.CreateElement("level")
		require.NoError(t, err)
		require.NoError(t, parent.AddChild(child))
		parent = child
	}

	ctx := &cancelAfterNContext{cancelAfter: cancelAfter}

	nodes, err := ixpath.TraverseAxis(ctx, ixpath.AxisDescendant, root, ixpath.DefaultMaxNodeSetLength)
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, nodes)
	// The walk must have stopped near the cancellation point, not after walking
	// the entire subtree.
	require.LessOrEqual(t, ctx.calls, cancelAfter+1,
		"traversal should stop on the first cancelled Err() observation")
}

// countingContext records how many times Err() is consulted but never reports
// cancellation. It lets a test observe that the child-enumeration loops consult
// ctx.Err() once per child enqueued (in addition to the per-pop check), so a
// cancelled context observed mid-enumeration aborts within O(1) children rather
// than after pushing all O(width) children unchecked.
type countingContext struct {
	calls int
}

func (c *countingContext) Deadline() (time.Time, bool) { return time.Time{}, false }
func (c *countingContext) Done() <-chan struct{}       { return nil }
func (c *countingContext) Value(any) any               { return nil }
func (c *countingContext) Err() error                  { c.calls++; return nil }

// TestTraverseAxisDescendant_WideChildEnumerationChecksContext verifies that
// the forward descendant traversal consults ctx.Err() both while enqueuing a
// very WIDE node's children and while popping them. Each of the width children
// is a leaf, so the per-pop loop alone yields ~width Err() consultations; the
// in-loop check inside the child-enumeration (push) loop adds ~width more.
// Requiring >= 2*width therefore fails if the push loop skips the ctx check,
// which is the condition that would let a cancelled context do O(width) work
// before aborting.
func TestTraverseAxisDescendant_WideChildEnumerationChecksContext(t *testing.T) {
	const width = 20000

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root, err := doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, doc.AddChild(root))

	for range width {
		child, err := doc.CreateElement("child")
		require.NoError(t, err)
		require.NoError(t, root.AddChild(child))
	}

	ctx := &countingContext{}

	nodes, err := ixpath.TraverseAxis(ctx, ixpath.AxisDescendant, root, ixpath.DefaultMaxNodeSetLength)
	require.NoError(t, err)
	require.Len(t, nodes, width)
	require.GreaterOrEqual(t, ctx.calls, 2*width,
		"wide child enumeration must check ctx.Err() per child while enqueuing and popping")
}

// TestTraverseAxisChild_WideContextCancelledMidWalk verifies that the child
// axis (routed through TraverseAxisSimple) aborts promptly when the context is
// cancelled partway through enumeration, rather than materializing the full
// node-set after cancellation. cancelAfterNContext lets the TraverseAxis-entry
// ctx.Err() succeed and only reports cancellation once enumeration is underway,
// so this genuinely exercises the in-loop ctx check inside axisChild.
func TestTraverseAxisChild_WideContextCancelledMidWalk(t *testing.T) {
	const width = 20000
	const cancelAfter = 10

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root, err := doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, doc.AddChild(root))
	for range width {
		child, err := doc.CreateElement("child")
		require.NoError(t, err)
		require.NoError(t, root.AddChild(child))
	}

	ctx := &cancelAfterNContext{cancelAfter: cancelAfter}

	nodes, err := ixpath.TraverseAxis(ctx, ixpath.AxisChild, root, ixpath.DefaultMaxNodeSetLength)
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, nodes)
	require.LessOrEqual(t, ctx.calls, cancelAfter+1,
		"child enumeration should stop on the first cancelled Err() observation")
}

// TestTraverseAxisAttribute_WideContextCancelledMidWalk verifies the same
// in-loop ctx check for the attribute axis (also routed through
// TraverseAxisSimple). Without it, a wide attribute list would be returned in
// full with a nil error after cancellation occurred mid-enumeration.
func TestTraverseAxisAttribute_WideContextCancelledMidWalk(t *testing.T) {
	const width = 5000
	const cancelAfter = 10

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root, err := doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, doc.AddChild(root))
	elem, err := doc.CreateElement("e")
	require.NoError(t, err)
	require.NoError(t, root.AddChild(elem))
	for i := range width {
		err := elem.SetAttribute("a"+strconv.Itoa(i), "v")
		require.NoError(t, err)
	}

	ctx := &cancelAfterNContext{cancelAfter: cancelAfter}

	nodes, err := ixpath.TraverseAxis(ctx, ixpath.AxisAttribute, elem, ixpath.DefaultMaxNodeSetLength)
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, nodes)
	require.LessOrEqual(t, ctx.calls, cancelAfter+1,
		"attribute enumeration should stop on the first cancelled Err() observation")
}

// TestTraverseAxisPreceding_WideChildEnumerationChecksContext verifies the same
// per-child ctx.Err() guarantee for the reverse traversal used by the preceding
// axis, whose collectDescendantsReverse helper has its own child-enumeration
// (push/enqueue) loops.
//
// The leaf-pop path alone already yields ~2*width Err() consultations: each of
// the width leaf children is visited via two stack frames (unexpanded pop +
// expanded pop), and each pop consults ctx. A >= 2*width assertion would
// therefore STILL pass even if the enqueue loop stopped checking ctx, so it
// does not genuinely guard the enqueue-loop check. The initial enqueue loop
// over left's children adds another ~width consultations, so a correct
// implementation reaches ~3*width. Requiring >= 3*width fails if the enqueue
// loop drops its ctx check, which is the condition that would let a cancelled
// context push all O(width) children before aborting.
func TestTraverseAxisPreceding_WideChildEnumerationChecksContext(t *testing.T) {
	const width = 20000

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root, err := doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, doc.AddChild(root))

	// A wide left subtree whose descendants precede the context node; the
	// preceding axis walks them via collectDescendantsReverse.
	left, err := doc.CreateElement("left")
	require.NoError(t, err)
	require.NoError(t, root.AddChild(left))
	for range width {
		child, err := doc.CreateElement("child")
		require.NoError(t, err)
		require.NoError(t, left.AddChild(child))
	}

	ctx0, err := doc.CreateElement("ctx")
	require.NoError(t, err)
	require.NoError(t, root.AddChild(ctx0))

	ctx := &countingContext{}

	nodes, err := ixpath.TraverseAxis(ctx, ixpath.AxisPreceding, ctx0, ixpath.DefaultMaxNodeSetLength)
	require.NoError(t, err)
	// left + its width children all precede ctx0.
	require.Len(t, nodes, width+1)
	require.GreaterOrEqual(t, ctx.calls, 3*width,
		"wide reverse child enumeration must check ctx.Err() in the enqueue loop too, not only on pop")
}

// TestTraverseAxisNamespace_WideContextCancelledMidWalk verifies that the
// namespace axis aborts promptly when the context is cancelled partway through
// scanning an element's many namespace declarations. axisNamespace delegates the
// namespace work to NamespacePrefixesInScope and CollectNamespaceNodes; without
// in-loop ctx.Err() checks inside those helpers the full namespace::* node-set
// would be computed (and could even be returned with a nil error) after
// cancellation had already occurred. cancelAfterNContext lets the ancestor walk
// proceed and only reports cancellation once the helper loops are underway.
func TestTraverseAxisNamespace_WideContextCancelledMidWalk(t *testing.T) {
	const width = 5000
	const cancelAfter = 10

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root, err := doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, doc.AddChild(root))
	elem, err := doc.CreateElement("e")
	require.NoError(t, err)
	require.NoError(t, root.AddChild(elem))
	for i := range width {
		require.NoError(t, elem.DeclareNamespace("p"+strconv.Itoa(i), "urn:ns:"+strconv.Itoa(i)))
	}

	ctx := &cancelAfterNContext{cancelAfter: cancelAfter}

	nodes, err := ixpath.TraverseAxis(ctx, ixpath.AxisNamespace, elem, ixpath.DefaultMaxNodeSetLength)
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, nodes)
	require.LessOrEqual(t, ctx.calls, cancelAfter+1,
		"namespace enumeration should stop on the first cancelled Err() observation")
}

// TestTraverseAxisNamespace_WideEnumerationChecksContext verifies that the
// namespace-axis helpers (NamespacePrefixesInScope and CollectNamespaceNodes)
// consult ctx.Err() per declared namespace in their inner loops, not merely
// once per ancestor. The element declares width namespaces, so a correct
// implementation reaches at least ~2*width consultations (NamespacePrefixesInScope
// scans them once; CollectNamespaceNodes scans them again across its two passes).
// Requiring >= 2*width fails if the inner-loop ctx checks are dropped, which is
// the condition that would let a cancelled context compute the whole node-set.
func TestTraverseAxisNamespace_WideEnumerationChecksContext(t *testing.T) {
	const width = 20000

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root, err := doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, doc.AddChild(root))
	elem, err := doc.CreateElement("e")
	require.NoError(t, err)
	require.NoError(t, root.AddChild(elem))
	for i := range width {
		require.NoError(t, elem.DeclareNamespace("p"+strconv.Itoa(i), "urn:ns:"+strconv.Itoa(i)))
	}

	ctx := &countingContext{}

	nodes, err := ixpath.TraverseAxis(ctx, ixpath.AxisNamespace, elem, ixpath.DefaultMaxNodeSetLength)
	require.NoError(t, err)
	// width declared prefixes plus the implicit xml namespace node.
	require.Len(t, nodes, width+1)
	require.GreaterOrEqual(t, ctx.calls, 2*width,
		"namespace helpers must check ctx.Err() per declared namespace in their inner loops")
}
