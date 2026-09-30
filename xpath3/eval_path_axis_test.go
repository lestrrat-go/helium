package xpath3_test

import (
	"strconv"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// entityAxisSrc declares an entity between comments and PIs of the internal
// subset and references it twice, once next to an element and once inside
// one.
const entityAxisSrc = `<!DOCTYPE a [<!--c0--><?p0 x?><!ENTITY e "<b id='eb'>ent<!--ec--></b>"><!--c1--><?p1 y?>]>` +
	`<a><b id="b1"/>&e;<x>&e;<!--cx--></x></a>`

// entityAxisLabel names a node by kind, and by name, id or content where one
// tells it apart.
func entityAxisLabel(n helium.Node) string {
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

func entityAxisLabels(t *testing.T, expr string, ctxNode helium.Node) []string {
	t.Helper()
	nodes, ok := xpath3.NodesFrom(evalExpr(t, ctxNode, expr))
	require.True(t, ok, expr)
	labels := make([]string, 0, len(nodes))
	for _, n := range nodes {
		labels = append(labels, entityAxisLabel(n))
	}
	return labels
}

// firstEntityRef returns the first entity-reference node under n in document
// order, or nil.
func firstEntityRef(n helium.Node) helium.Node {
	for c := range helium.Children(n) {
		if c.Type() == helium.EntityRefNode {
			return c
		}
		if found := firstEntityRef(c); found != nil {
			return found
		}
	}
	return nil
}

type entityAxisCase struct {
	expr string
	want []string
}

// TestDescendantAxisEntityBoundary checks that descendant and
// descendant-or-self stop at the owned-child boundary. The XDM has no entity
// references (the infoset mapping expands them), the child axis of an entity
// reference is empty, and the descendant axis is the transitive closure of
// the child axis, so the descendant axis of an entity reference is empty: it
// never reaches the comments, PIs or elements of the DTD that holds the
// referenced entity.
func TestDescendantAxisEntityBoundary(t *testing.T) {
	t.Run("entities not substituted", func(t *testing.T) {
		doc := mustParseXML(t, entityAxisSrc)
		ref := firstEntityRef(doc)
		require.NotNil(t, ref)
		ent := ref.FirstChild()
		require.Equal(t, helium.EntityNode, ent.Type())
		root := doc.DocumentElement()
		var x helium.Node
		for c := range helium.ChildElements(root) {
			if c.Name() == "x" {
				x = c
			}
		}
		require.NotNil(t, x)

		contexts := []struct {
			name  string
			node  helium.Node
			cases []entityAxisCase
		}{
			{
				name: "entity reference",
				node: ref,
				cases: []entityAxisCase{
					{expr: "child::node()", want: []string{}},
					{expr: "descendant::node()", want: []string{}},
					{expr: "descendant-or-self::node()", want: []string{"entref(e)"}},
					{expr: ".//node()", want: []string{}},
					{expr: "descendant::comment()", want: []string{}},
				},
			},
			{
				name: "entity",
				node: ent,
				cases: []entityAxisCase{
					{expr: "descendant::node()", want: []string{"b#eb", "text(ent)", "comment(ec)"}},
					{expr: "descendant-or-self::node()", want: []string{"entity(e)", "b#eb", "text(ent)", "comment(ec)"}},
				},
			},
			{
				name: "dtd",
				node: doc.IntSubset(),
				cases: []entityAxisCase{
					{expr: "descendant::node()", want: []string{"comment(c0)", "pi(p0)", "comment(c1)", "pi(p1)"}},
				},
			},
			{
				name: "element holding a reference",
				node: x,
				cases: []entityAxisCase{
					{expr: "descendant::node()", want: []string{"comment(cx)"}},
					{expr: "descendant-or-self::node()", want: []string{"x", "comment(cx)"}},
				},
			},
			{
				name: "document",
				node: doc,
				cases: []entityAxisCase{
					{expr: "//node()", want: []string{"a", "b#b1", "x", "comment(cx)"}},
				},
			},
		}
		for _, c := range contexts {
			t.Run(c.name, func(t *testing.T) {
				for _, tc := range c.cases {
					require.Equal(t, tc.want, entityAxisLabels(t, tc.expr, c.node), tc.expr)
				}
			})
		}
	})

	t.Run("entities substituted", func(t *testing.T) {
		doc, err := helium.NewParser().SubstituteEntities(true).Parse(t.Context(), []byte(entityAxisSrc))
		require.NoError(t, err)
		require.Nil(t, firstEntityRef(doc))
		require.Equal(t, []string{"a", "b#b1", "b#eb", "text(ent)", "comment(ec)", "x", "b#eb", "text(ent)", "comment(ec)", "comment(cx)"},
			entityAxisLabels(t, "//node()", doc))
		require.Equal(t, []string{"comment(c0)", "pi(p0)", "comment(c1)", "pi(p1)"},
			entityAxisLabels(t, "descendant::node()", doc.IntSubset()))
	})

	t.Run("hand-built DTD holding an element", func(t *testing.T) {
		doc := mustParseXML(t, `<a/>`)
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

		for _, expr := range []string{"descendant::x", ".//x", "descendant::node()", "descendant::*"} {
			require.Empty(t, entityAxisLabels(t, expr, ref), expr)
		}
		require.Equal(t, []string{"x"}, entityAxisLabels(t, "descendant::x", dtd))
	})
}
