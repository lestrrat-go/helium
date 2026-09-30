package xpath1_test

import (
	"strconv"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/xpath1"
	"github.com/stretchr/testify/require"
)

// entityAxisSrc declares an entity between comments and PIs of the internal
// subset and references it twice, once next to an element and once inside
// one.
const entityAxisSrc = `<!DOCTYPE a [<!--c0--><?p0 x?><!ENTITY e "<b id='eb'>ent<!--ec--></b>"><!--c1--><?p1 y?>]>` +
	`<a><b id="b1"/>&e;<x>&e;<!--cx--></x></a>`

// Expressions and node labels the cases below repeat.
const (
	descendantNodeExpr    = "descendant::node()"
	dotDescendantNodeExpr = ".//node()"
	entityContentLabel    = "b#eb"
	commentCXLabel        = "comment(cx)"
	commentECLabel        = "comment(ec)"
	textEntLabel          = "text(ent)"
)

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
	r, err := xpath1.MustCompile(expr).Evaluate(t.Context(), ctxNode)
	require.NoError(t, err, expr)
	require.Equal(t, xpath1.NodeSetResult, r.Type, expr)
	labels := make([]string, 0, len(r.NodeSet))
	for _, n := range r.NodeSet {
		labels = append(labels, entityAxisLabel(n))
	}
	return labels
}

type entityAxisCase struct {
	expr string
	want []string
}

// TestDescendantAxisEntityBoundary checks that descendant and
// descendant-or-self stop at the owned-child boundary, as the child axis
// does: from an entity reference they never reach the comments, PIs or
// elements of the DTD that holds the referenced entity. libxml2 does not
// descend into an entity declaration reached from a reference and skips DTD
// nodes (xmlXPathNextDescendant); the child axis of an entity reference is
// empty here, so its descendant axis is empty too.
func TestDescendantAxisEntityBoundary(t *testing.T) {
	t.Run("entities not substituted", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(entityAxisSrc))
		require.NoError(t, err)
		ref := firstEntityRef(doc)
		require.NotNil(t, ref)
		ent := ref.FirstChild()
		require.Equal(t, helium.EntityNode, ent.Type())
		x := firstNodeOf(t, doc, "//x")
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
					{expr: descendantNodeExpr, want: []string{}},
					{expr: "descendant-or-self::node()", want: []string{"entref(e)"}},
					{expr: dotDescendantNodeExpr, want: []string{}},
					{expr: "descendant::comment()", want: []string{}},
					{expr: "descendant::processing-instruction()", want: []string{}},
					{expr: ".//comment()", want: []string{}},
					{expr: "descendant::node()/..", want: []string{}},
				},
			},
			{
				name: "entity",
				node: ent,
				cases: []entityAxisCase{
					{expr: descendantNodeExpr, want: []string{entityContentLabel, textEntLabel, commentECLabel}},
					{expr: dotDescendantNodeExpr, want: []string{entityContentLabel, textEntLabel, commentECLabel}},
					{expr: "descendant-or-self::node()", want: []string{"entity(e)", entityContentLabel, textEntLabel, commentECLabel}},
				},
			},
			{
				name: "dtd",
				node: doc.IntSubset(),
				cases: []entityAxisCase{
					{expr: descendantNodeExpr, want: []string{"comment(c0)", "pi(p0)", "comment(c1)", "pi(p1)"}},
					{expr: dotDescendantNodeExpr, want: []string{"comment(c0)", "pi(p0)", "comment(c1)", "pi(p1)"}},
				},
			},
			{
				name: "element holding a reference",
				node: x,
				cases: []entityAxisCase{
					{expr: descendantNodeExpr, want: []string{commentCXLabel}},
					{expr: "descendant-or-self::node()", want: []string{"x", commentCXLabel}},
					{expr: dotDescendantNodeExpr, want: []string{commentCXLabel}},
				},
			},
			{
				name: "document",
				node: doc,
				cases: []entityAxisCase{
					{expr: "//node()", want: []string{"a", "b#b1", "x", commentCXLabel}},
					{expr: descendantNodeExpr, want: []string{"a", "b#b1", "x", commentCXLabel}},
					{expr: "//comment()", want: []string{commentCXLabel}},
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
		x := firstNodeOf(t, doc, "//x")
		require.NotNil(t, x)
		require.Equal(t, []string{entityContentLabel, textEntLabel, commentECLabel, commentCXLabel},
			entityAxisLabels(t, descendantNodeExpr, x))
		require.Equal(t, []string{"a", "b#b1", entityContentLabel, textEntLabel, commentECLabel, "x", entityContentLabel, textEntLabel, commentECLabel, commentCXLabel},
			entityAxisLabels(t, "//node()", doc))
		require.Equal(t, []string{"comment(c0)", "pi(p0)", "comment(c1)", "pi(p1)"},
			entityAxisLabels(t, descendantNodeExpr, doc.IntSubset()))
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

		for _, expr := range []string{"descendant::x", ".//x", descendantNodeExpr, dotDescendantNodeExpr, "descendant::*"} {
			require.Empty(t, entityAxisLabels(t, expr, ref), expr)
		}
		require.Equal(t, []string{"x"}, entityAxisLabels(t, "descendant::x", dtd))
		require.Equal(t, []string{"x"}, entityAxisLabels(t, ".//x", dtd))
	})
}
