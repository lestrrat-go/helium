package xpath_test

import (
	"slices"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	ixpath "github.com/lestrrat-go/helium/internal/xpath"
	"github.com/stretchr/testify/require"
)

const stepOrderSrc = `<?xml version="1.0"?><!--lead-->` +
	`<r xmlns:p="urn:p" id="r">` +
	`<a id="a1" p:k="1"><b id="b1"/>t<b id="b2"><c/></b><!--c--></a>` +
	`<a id="a2" xmlns:q="urn:q"><b id="b3"/><?pi x?><b id="b4"/></a>` +
	`<a id="a3"><d><b id="b5"/></d></a>` +
	`</r><!--trail-->`

func parseStepOrderDoc(t *testing.T, src string) *helium.Document {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err)
	return doc
}

// elementByID returns the element whose id attribute is id.
func elementByID(t *testing.T, doc *helium.Document, id string) *helium.Element {
	t.Helper()
	for n := range helium.Descendants(doc) {
		e, ok := n.(*helium.Element)
		if !ok {
			continue
		}
		if v, ok := e.GetAttribute("id"); ok && v == id {
			return e
		}
	}
	require.FailNow(t, "no element with id "+id)
	return nil
}

// attrNodes returns the attributes of e as nodes.
func attrNodes(e *helium.Element) []helium.Node {
	var nodes []helium.Node
	for _, a := range e.Attributes() {
		nodes = append(nodes, a)
	}
	return nodes
}

// traverseAll concatenates the axis result of every input, as a location
// step does before ordering.
func traverseAll(t *testing.T, axis ixpath.AxisType, inputs []helium.Node) []helium.Node {
	t.Helper()
	var out []helium.Node
	for _, n := range inputs {
		r, err := ixpath.TraverseAxis(t.Context(), axis, n, ixpath.DefaultMaxNodeSetLength)
		require.NoError(t, err)
		out = append(out, r...)
	}
	return out
}

// requireSameAsDedup checks that OrderStepResult returns exactly what
// DeduplicateNodes returns for the same step.
func requireSameAsDedup(t *testing.T, axis ixpath.AxisType, inputs []helium.Node) {
	t.Helper()
	out := traverseAll(t, axis, inputs)
	want, err := ixpath.DeduplicateNodes(slices.Clone(out), &ixpath.DocOrderCache{}, ixpath.DefaultMaxNodeSetLength)
	require.NoError(t, err)
	got, err := ixpath.OrderStepResult(out, inputs, axis, &ixpath.DocOrderCache{}, ixpath.DefaultMaxNodeSetLength)
	require.NoError(t, err)
	require.Equal(t, want, got, "axis %s", axis)
}

var allAxes = []ixpath.AxisType{
	ixpath.AxisChild, ixpath.AxisDescendant, ixpath.AxisParent, ixpath.AxisAncestor,
	ixpath.AxisFollowingSibling, ixpath.AxisPrecedingSibling, ixpath.AxisFollowing,
	ixpath.AxisPreceding, ixpath.AxisAttribute, ixpath.AxisNamespace, ixpath.AxisSelf,
	ixpath.AxisDescendantOrSelf, ixpath.AxisAncestorOrSelf,
}

func TestOrderStepResult(t *testing.T) {
	doc := parseStepOrderDoc(t, stepOrderSrc)
	b2 := elementByID(t, doc, "b2")
	a1 := elementByID(t, doc, "a1")
	a2 := elementByID(t, doc, "a2")
	a3 := elementByID(t, doc, "a3")
	b3 := elementByID(t, doc, "b3")
	b4 := elementByID(t, doc, "b4")
	b5 := elementByID(t, doc, "b5")

	t.Run("one input matches DeduplicateNodes on every axis", func(t *testing.T) {
		attr := a1.Attributes()[0]
		nsNodes := traverseAll(t, ixpath.AxisNamespace, []helium.Node{a2})
		require.NotEmpty(t, nsNodes)
		contexts := []helium.Node{doc, doc.DocumentElement(), a2, b2, b5, attr, nsNodes[0]}
		for _, ctxNode := range contexts {
			for _, axis := range allAxes {
				requireSameAsDedup(t, axis, []helium.Node{ctxNode})
			}
		}
	})

	t.Run("reverse axes from one input come back forward", func(t *testing.T) {
		out := traverseAll(t, ixpath.AxisAncestor, []helium.Node{b2})
		require.Equal(t, []helium.Node{a1, doc.DocumentElement(), doc}, out)
		got, err := ixpath.OrderStepResult(out, []helium.Node{b2}, ixpath.AxisAncestor, &ixpath.DocOrderCache{}, 10)
		require.NoError(t, err)
		require.Equal(t, []helium.Node{doc, doc.DocumentElement(), a1}, got)
		require.Equal(t, len(got), cap(got), "capacity must be clamped")
	})

	t.Run("same-depth inputs match DeduplicateNodes", func(t *testing.T) {
		sameDepthSets := [][]helium.Node{
			{a1, a2, a3},
			{b3, b4},
			{elementByID(t, doc, "b1"), b2, b3, b4},
			attrNodes(a1),
			traverseAll(t, ixpath.AxisNamespace, []helium.Node{a1, a2}),
		}
		for _, inputs := range sameDepthSets {
			for _, axis := range allAxes {
				requireSameAsDedup(t, axis, inputs)
			}
		}
	})

	t.Run("mixed-depth inputs match DeduplicateNodes", func(t *testing.T) {
		mixed := [][]helium.Node{
			{doc.DocumentElement(), a1, b2},
			{a1, b2, a2, b5},
			{b2, b3, b5},
		}
		for _, inputs := range mixed {
			for _, axis := range allAxes {
				requireSameAsDedup(t, axis, inputs)
			}
		}
	})

	t.Run("parent compacts adjacent duplicates", func(t *testing.T) {
		out := []helium.Node{a2, a2, a3}
		got, err := ixpath.OrderStepResult(out, []helium.Node{b3, b4, b5.Parent()}, ixpath.AxisParent, &ixpath.DocOrderCache{}, 10)
		require.NoError(t, err)
		require.Equal(t, []helium.Node{a2, a3}, got)
	})

	t.Run("mixed-depth inputs take the sorting path", func(t *testing.T) {
		// out is deliberately out of order. The skip path would return it
		// unchanged; only DeduplicateNodes sorts it.
		out := []helium.Node{b4, b3}
		got, err := ixpath.OrderStepResult(out, []helium.Node{a1, b2}, ixpath.AxisChild, &ixpath.DocOrderCache{}, 10)
		require.NoError(t, err)
		require.Equal(t, []helium.Node{b3, b4}, got)
	})

	t.Run("same-depth inputs take the skip path", func(t *testing.T) {
		out := []helium.Node{b4, b3}
		out = append(make([]helium.Node, 0, 8), out...)
		got, err := ixpath.OrderStepResult(out, []helium.Node{a1, a2}, ixpath.AxisChild, &ixpath.DocOrderCache{}, 10)
		require.NoError(t, err)
		require.Equal(t, []helium.Node{b4, b3}, got)
		require.Equal(t, len(got), cap(got), "capacity must be clamped")
	})

	t.Run("sorting axes with several inputs take the sorting path", func(t *testing.T) {
		out := []helium.Node{b4, b3}
		got, err := ixpath.OrderStepResult(out, []helium.Node{a1, a2}, ixpath.AxisDescendant, &ixpath.DocOrderCache{}, 10)
		require.NoError(t, err)
		require.Equal(t, []helium.Node{b3, b4}, got)
	})

	t.Run("maxNodes is enforced on the skip paths", func(t *testing.T) {
		out := traverseAll(t, ixpath.AxisChild, []helium.Node{a2})
		require.Len(t, out, 3)
		_, err := ixpath.OrderStepResult(out, []helium.Node{a2}, ixpath.AxisChild, &ixpath.DocOrderCache{}, 2)
		require.ErrorIs(t, err, ixpath.ErrNodeSetLimit)

		out = traverseAll(t, ixpath.AxisChild, []helium.Node{a1, a2})
		_, err = ixpath.OrderStepResult(out, []helium.Node{a1, a2}, ixpath.AxisChild, &ixpath.DocOrderCache{}, 3)
		require.ErrorIs(t, err, ixpath.ErrNodeSetLimit)
	})

	t.Run("zero or one node is returned as is", func(t *testing.T) {
		got, err := ixpath.OrderStepResult(nil, []helium.Node{a1, a2}, ixpath.AxisDescendant, &ixpath.DocOrderCache{}, 10)
		require.NoError(t, err)
		require.Empty(t, got)
		got, err = ixpath.OrderStepResult([]helium.Node{b3}, []helium.Node{a1, a2}, ixpath.AxisDescendant, &ixpath.DocOrderCache{}, 10)
		require.NoError(t, err)
		require.Equal(t, []helium.Node{b3}, got)
	})
}

// TestOrderStepResultDeepInputs checks that inputs deeper than the depth
// bound take the sorting path.
func TestOrderStepResultDeepInputs(t *testing.T) {
	const depth = 70
	var sb strings.Builder
	for range depth {
		sb.WriteString("<e>")
	}
	sb.WriteString(`<a id="x1"><b id="y1"/></a><a id="x2"><b id="y2"/></a>`)
	for range depth {
		sb.WriteString("</e>")
	}
	doc := parseStepOrderDoc(t, sb.String())
	x1 := elementByID(t, doc, "x1")
	x2 := elementByID(t, doc, "x2")
	y1 := elementByID(t, doc, "y1")
	y2 := elementByID(t, doc, "y2")

	out := []helium.Node{y2, y1}
	got, err := ixpath.OrderStepResult(out, []helium.Node{x1, x2}, ixpath.AxisChild, &ixpath.DocOrderCache{}, 10)
	require.NoError(t, err)
	require.Equal(t, []helium.Node{y1, y2}, got)

	requireSameAsDedup(t, ixpath.AxisChild, []helium.Node{x1, x2})
}

// TestOrderStepResultReservesDocument checks that a skipping step registers
// its document in the cache at the point indexing would have, so the order
// between documents and the positions inside the reserved document match a
// cache that indexed the document.
func TestOrderStepResultReservesDocument(t *testing.T) {
	doc1 := parseStepOrderDoc(t, stepOrderSrc)
	doc2 := parseStepOrderDoc(t, stepOrderSrc)
	a1 := elementByID(t, doc1, "a1")
	inputs := []helium.Node{a1}
	doc1Nodes := traverseAll(t, ixpath.AxisChild, inputs)
	doc2Nodes := traverseAll(t, ixpath.AxisChild, []helium.Node{elementByID(t, doc2, "a2")})

	// indexed runs the step through DeduplicateNodes, which indexes doc1.
	indexed := &ixpath.DocOrderCache{}
	_, err := ixpath.DeduplicateNodes(slices.Clone(doc1Nodes), indexed, ixpath.DefaultMaxNodeSetLength)
	require.NoError(t, err)

	// reserved runs the same step through OrderStepResult, which only
	// reserves doc1.
	reserved := &ixpath.DocOrderCache{}
	_, err = ixpath.OrderStepResult(slices.Clone(doc1Nodes), inputs, ixpath.AxisChild, reserved, ixpath.DefaultMaxNodeSetLength)
	require.NoError(t, err)

	t.Run("merge with a second document", func(t *testing.T) {
		want, err := ixpath.MergeNodeSets(doc2Nodes, doc1Nodes, indexed, ixpath.DefaultMaxNodeSetLength)
		require.NoError(t, err)
		got, err := ixpath.MergeNodeSets(doc2Nodes, doc1Nodes, reserved, ixpath.DefaultMaxNodeSetLength)
		require.NoError(t, err)
		require.Equal(t, want, got)
		require.Same(t, doc1Nodes[0], got[0], "doc1 was registered first")
	})

	t.Run("positions and comparisons in the reserved document", func(t *testing.T) {
		fresh := &ixpath.DocOrderCache{}
		_, err := ixpath.OrderStepResult(slices.Clone(doc1Nodes), inputs, ixpath.AxisChild, fresh, ixpath.DefaultMaxNodeSetLength)
		require.NoError(t, err)
		for _, n := range doc1Nodes {
			require.Equal(t, indexed.Position(n), fresh.Position(n))
		}
		require.Negative(t, fresh.Compare(doc1Nodes[0], doc1Nodes[1]))
		require.Positive(t, fresh.Compare(doc1Nodes[1], doc1Nodes[0]))
	})

	t.Run("comparison against a second document", func(t *testing.T) {
		fresh := &ixpath.DocOrderCache{}
		_, err := ixpath.OrderStepResult(slices.Clone(doc1Nodes), inputs, ixpath.AxisChild, fresh, ixpath.DefaultMaxNodeSetLength)
		require.NoError(t, err)
		fresh.BuildFrom(doc2)
		require.Negative(t, fresh.Compare(doc1Nodes[1], doc2Nodes[0]))
		require.Positive(t, fresh.Compare(doc2Nodes[0], doc1Nodes[1]))
	})

	t.Run("Reset drops the reservation", func(t *testing.T) {
		fresh := &ixpath.DocOrderCache{}
		_, err := ixpath.OrderStepResult(slices.Clone(doc1Nodes), inputs, ixpath.AxisChild, fresh, ixpath.DefaultMaxNodeSetLength)
		require.NoError(t, err)
		fresh.Reset()
		got, err := ixpath.MergeNodeSets(doc2Nodes, doc1Nodes, fresh, ixpath.DefaultMaxNodeSetLength)
		require.NoError(t, err)
		require.Same(t, doc2Nodes[0], got[0], "doc2 is registered first after Reset")
	})
}

// stepOrderEntitySrc references entity e, whose content references entity f,
// inside the body, after an element sibling, so the document-order index
// places the content of both entities at their last reference: the content
// of f sits in the middle of the content of e. The internal subset holds a
// comment and a PI on either side of the declarations.
const stepOrderEntitySrc = `<!DOCTYPE r [<!--c0--><?p0 x?><!ENTITY f "<d/>t"><!ENTITY e "<b/>&f;<c/>"><!--c1-->]>` +
	`<r><a id="a1"><z id="z1"/>&e;</a><a id="a2">&f;&e;<z id="z2"/></a></r>`

// TestOrderStepResultEntityInputs checks steps whose inputs are entity
// references, Entity nodes or the DTD against DeduplicateNodes.
func TestOrderStepResultEntityInputs(t *testing.T) {
	doc := parseStepOrderDoc(t, stepOrderEntitySrc)
	a1 := elementByID(t, doc, "a1")
	a2 := elementByID(t, doc, "a2")
	dtd := doc.IntSubset()
	require.NotNil(t, dtd)
	refE1 := childByType(a1, helium.EntityRefNode, "e")
	refF2 := childByType(a2, helium.EntityRefNode, "f")
	refE2 := childByType(a2, helium.EntityRefNode, "e")
	require.NotNil(t, refE1)
	require.NotNil(t, refF2)
	require.NotNil(t, refE2)
	entE := childByType(dtd, helium.EntityNode, "e")
	entF := childByType(dtd, helium.EntityNode, "f")
	require.NotNil(t, entE)
	require.NotNil(t, entF)

	t.Run("one input matches DeduplicateNodes on every axis", func(t *testing.T) {
		for _, ctxNode := range []helium.Node{refE1, refF2, refE2, entE, entF, dtd} {
			for _, axis := range allAxes {
				requireSameAsDedup(t, axis, []helium.Node{ctxNode})
			}
		}
	})

	t.Run("same-depth inputs match DeduplicateNodes", func(t *testing.T) {
		sameDepthSets := [][]helium.Node{
			{refF2, refE2},
			{refE1, refF2, refE2},
			{dtd, doc.DocumentElement()},
			{a1, entE},
			{entE, entF},
		}
		for _, inputs := range sameDepthSets {
			for _, axis := range allAxes {
				requireSameAsDedup(t, axis, inputs)
			}
		}
	})

	t.Run("entity inputs take the sorting path", func(t *testing.T) {
		// The index places the content of f inside the content of e, so the
		// child steps of e and f interleave: the concatenation b c d t is not
		// in document order, and only DeduplicateNodes sorts it.
		out := traverseAll(t, ixpath.AxisChild, []helium.Node{entE, entF})
		require.Len(t, out, 4)
		want, err := ixpath.DeduplicateNodes(slices.Clone(out), &ixpath.DocOrderCache{}, ixpath.DefaultMaxNodeSetLength)
		require.NoError(t, err)
		require.NotEqual(t, out, want)
		got, err := ixpath.OrderStepResult(out, []helium.Node{entE, entF}, ixpath.AxisChild, &ixpath.DocOrderCache{}, ixpath.DefaultMaxNodeSetLength)
		require.NoError(t, err)
		require.Equal(t, want, got)
	})

	t.Run("entity content inputs match DeduplicateNodes", func(t *testing.T) {
		// Every node inside the content of e, including the reference to f
		// and the content of f reached through it.
		var content []helium.Node
		for n := range helium.Descendants(entE) {
			content = append(content, n)
		}
		require.Len(t, content, 6)
		for _, ctxNode := range content {
			for _, axis := range allAxes {
				requireSameAsDedup(t, axis, []helium.Node{ctxNode})
			}
		}
		// Same-depth sets drawn from the content of both entities, and from
		// entity content and the body, sorted as a previous step leaves them.
		sameDepthSets := [][]helium.Node{
			{entE.FirstChild(), entF.FirstChild()},
			slices.Collect(helium.Children(entE)),
			append(slices.Collect(helium.Children(entE)), slices.Collect(helium.Children(entF))...),
			{entE.FirstChild(), elementByID(t, doc, "z1"), elementByID(t, doc, "z2")},
		}
		for _, inputs := range sameDepthSets {
			inputs, err := ixpath.DeduplicateNodes(inputs, &ixpath.DocOrderCache{}, ixpath.DefaultMaxNodeSetLength)
			require.NoError(t, err)
			for _, axis := range allAxes {
				requireSameAsDedup(t, axis, inputs)
			}
		}
	})
}
