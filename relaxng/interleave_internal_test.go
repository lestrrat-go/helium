package relaxng

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// TestInterleavePartitionRoute builds the routing table of
// interleave(group(a, optional(b)), zeroOrMore(element nsName "urn:x"), text,
// attribute k) directly (bypassing the parser) and checks both the compiled
// table (6.4) and route (6.6) against a handful of representative nodes.
func TestInterleavePartitionRoute(t *testing.T) {
	t.Parallel()

	elemA := &pattern{kind: patternElement, name: "a", nameClass: &nameClass{kind: ncName, name: "a"}}
	elemB := &pattern{kind: patternElement, name: "b", nameClass: &nameClass{kind: ncName, name: "b"}}
	group := &pattern{kind: patternGroup, children: []*pattern{
		elemA,
		{kind: patternOptional, children: []*pattern{elemB}},
	}}

	wildElem := &pattern{kind: patternElement, nameClass: &nameClass{kind: ncNsName, ns: "urn:x"}}
	zeroOrMoreWild := &pattern{kind: patternZeroOrMore, children: []*pattern{wildElem}}

	textPat := &pattern{kind: patternText}

	attrK := &pattern{kind: patternAttribute, name: "k", nameClass: &nameClass{kind: ncName, name: "k"}}

	part := computeInterleavePartition(&pattern{
		kind:     patternInterleave,
		children: []*pattern{group, zeroOrMoreWild, textPat, attrK},
	})

	require.Equal(t, 0, part.byName[ncQName{name: "a"}], "group branch owns a")
	require.Equal(t, 0, part.byName[ncQName{name: "b"}], "group branch owns b")
	require.Equal(t, []int{1}, part.wild, "only the nsName branch is wild")
	require.Equal(t, 2, part.text, "the text branch is index 2")
	require.Empty(t, part.branches[3].elems, "the attribute branch has no element leaves")
	require.Len(t, part.branches[3].attrs, 1, "the attribute branch has one attribute leaf")

	doc, err := helium.NewParser().Parse(t.Context(), []byte(
		`<r xmlns:p="urn:x"><a/><p:q/>text<!--c--><zz/></r>`))
	require.NoError(t, err)
	root := findDocElement(doc)
	require.NotNil(t, root)

	var nodes []helium.Node
	for child := range helium.Children(root) {
		nodes = append(nodes, child)
	}
	require.Len(t, nodes, 5, "a, p:q, text, comment, zz")

	require.Equal(t, 0, part.route(nodes[0]), "<a/> routes to the group branch")
	require.Equal(t, 1, part.route(nodes[1]), "<p:q/> routes to the wild nsName branch")
	require.Equal(t, 2, part.route(nodes[2]), "the text node routes to the text branch")
	require.Equal(t, -2, part.route(nodes[3]), "a comment is dropped, never routed")
	require.Equal(t, -1, part.route(nodes[4]), "<zz/> matches no branch")
}
