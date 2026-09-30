package xpath_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
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
// cancelled AFTER the descendant traversal has begun aborts the walk promptly,
// leaving a deep tree only partly walked. Without an in-loop
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
// cancelled partway through enumeration, materializing no full
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

// entityBoundarySrc declares an entity between comments and PIs of the
// internal subset and references it twice, once next to an element and once
// inside one.
const entityBoundarySrc = `<!DOCTYPE a [<!--c0--><?p0 x?><!ENTITY e "<b id='eb'>ent<!--ec--></b>"><!--c1--><?p1 y?>]>` +
	`<a><b id="b1"/>&e;<x>&e;<!--cx--></x></a>`

// axisNodeLabel names a node by kind, and by name, id or content where one
// tells it apart, so an axis result reads as a short list.
func axisNodeLabel(n helium.Node) string {
	switch n.Type() {
	case helium.ElementNode:
		if e, ok := n.(*helium.Element); ok {
			if id, ok := e.GetAttribute("id"); ok {
				return n.Name() + "#" + id
			}
		}
		return n.Name()
	case helium.TextNode:
		return "text(" + string(n.Content()) + ")"
	case helium.CommentNode:
		return "comment(" + string(n.Content()) + ")"
	case helium.ProcessingInstructionNode:
		return "pi(" + n.Name() + ")"
	case helium.EntityRefNode:
		return "entref(" + n.Name() + ")"
	case helium.EntityNode:
		return "entity(" + n.Name() + ")"
	case helium.DTDNode:
		return "dtd"
	case helium.DocumentNode:
		return "doc"
	}
	return "type" + strconv.Itoa(int(n.Type()))
}

func axisLabels(t *testing.T, axis ixpath.AxisType, n helium.Node) []string {
	t.Helper()
	nodes, err := ixpath.TraverseAxis(t.Context(), axis, n, ixpath.DefaultMaxNodeSetLength)
	require.NoError(t, err)
	labels := make([]string, 0, len(nodes))
	for _, m := range nodes {
		labels = append(labels, axisNodeLabel(m))
	}
	return labels
}

// firstOfType returns the first node of type typ in a pre-order walk of n
// that follows every child link, entity references' Entity children included.
func firstOfType(n helium.Node, typ helium.ElementType) helium.Node {
	for c := range helium.Children(n) {
		if c.Type() == typ {
			return c
		}
		if found := firstOfType(c, typ); found != nil {
			return found
		}
	}
	return nil
}

// TestTraverseAxisDescendant_EntityBoundary checks that the descendant axes
// stop at the owned-child boundary, as the child axis does. An entity
// reference's only child is the Entity node, owned by the DTD, whose sibling
// links belong to the DTD's declaration list; the descendant walk must not
// follow them into the DTD's comments and PIs.
func TestTraverseAxisDescendant_EntityBoundary(t *testing.T) {
	t.Run("entities not substituted", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(entityBoundarySrc))
		require.NoError(t, err)
		ref := firstOfType(doc, helium.EntityRefNode)
		require.NotNil(t, ref)
		ent := ref.FirstChild()
		require.Equal(t, helium.EntityNode, ent.Type())
		dtd := doc.IntSubset()
		root := doc.DocumentElement()
		x := childByType(root, helium.ElementNode, "x")
		require.NotNil(t, x)

		cases := []struct {
			name string
			node helium.Node
			desc []string
		}{
			{name: "entity reference", node: ref, desc: []string{}},
			{name: "entity", node: ent, desc: []string{"b#eb", "text(ent)", "comment(ec)"}},
			{name: "dtd", node: dtd, desc: []string{"comment(c0)", "pi(p0)", "comment(c1)", "pi(p1)"}},
			{name: "element holding a reference", node: x, desc: []string{"comment(cx)"}},
			{name: "root element", node: root, desc: []string{"b#b1", "x", "comment(cx)"}},
			{name: "document", node: doc, desc: []string{"a", "b#b1", "x", "comment(cx)"}},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				require.Equal(t, c.desc, axisLabels(t, ixpath.AxisDescendant, c.node))
				want := append([]string{axisNodeLabel(c.node)}, c.desc...)
				require.Equal(t, want, axisLabels(t, ixpath.AxisDescendantOrSelf, c.node))
			})
		}
	})

	t.Run("entities substituted", func(t *testing.T) {
		doc, err := helium.NewParser().SubstituteEntities(true).Parse(t.Context(), []byte(entityBoundarySrc))
		require.NoError(t, err)
		require.Nil(t, firstOfType(doc.DocumentElement(), helium.EntityRefNode))
		x := childByType(doc.DocumentElement(), helium.ElementNode, "x")
		require.NotNil(t, x)
		require.Equal(t, []string{"b#eb", "text(ent)", "comment(ec)", "comment(cx)"},
			axisLabels(t, ixpath.AxisDescendant, x))
		require.Equal(t, []string{"comment(c0)", "pi(p0)", "comment(c1)", "pi(p1)"},
			axisLabels(t, ixpath.AxisDescendant, doc.IntSubset()))
	})

	t.Run("hand-built DTD holding an element", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<a/>`))
		require.NoError(t, err)
		dtd, err := doc.CreateInternalSubset("a", "", "")
		require.NoError(t, err)
		x, err := doc.CreateElement("x")
		require.NoError(t, err)
		require.NoError(t, dtd.AddChild(x))
		_, err = dtd.AddEntity("e", enum.InternalGeneralEntity, "", "", "v")
		require.NoError(t, err)
		ref, err := doc.CreateReference("e")
		require.NoError(t, err)
		require.NoError(t, doc.DocumentElement().AddChild(ref))

		require.Empty(t, axisLabels(t, ixpath.AxisDescendant, ref))
		require.Equal(t, []string{"entref(e)"}, axisLabels(t, ixpath.AxisDescendantOrSelf, ref))
		require.Equal(t, []string{"x"}, axisLabels(t, ixpath.AxisDescendant, dtd))
	})
}
