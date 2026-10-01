package xpath_test

import (
	"slices"
	"testing"

	helium "github.com/lestrrat-go/helium"
	ixpath "github.com/lestrrat-go/helium/internal/xpath"
	"github.com/stretchr/testify/require"
)

// unionOtherSrc is the second document whose nodes join the operands, so the
// document registration order of a union shows in later merges.
const unionOtherSrc = `<o k="v" xmlns:p="urn:p"><z id="z1"/>t<z id="z2" p:k="w"/></o>`

// unionElements returns every element of doc, including the elements inside
// entity content, each once.
func unionElements(doc *helium.Document) []*helium.Element {
	seen := map[helium.Node]struct{}{}
	var elems []*helium.Element
	for n := range helium.Descendants(doc) {
		e, ok := n.(*helium.Element)
		if !ok {
			continue
		}
		if _, dup := seen[e]; dup {
			continue
		}
		seen[e] = struct{}{}
		elems = append(elems, e)
	}
	return elems
}

// unionOperands returns operands built from the attributes, owned children
// and namespace nodes of e, of its parent element, and of the document
// element of other: sorted runs, partial runs, unsorted runs, runs with
// duplicates and runs that mix kinds or elements.
func unionOperands(t *testing.T, e *helium.Element, other *helium.Document) [][]helium.Node {
	t.Helper()
	attrs := attrNodes(e)
	kids := slices.Collect(helium.Children(e))
	ns, err := ixpath.TraverseAxis(t.Context(), ixpath.AxisNamespace, e, ixpath.DefaultMaxNodeSetLength)
	require.NoError(t, err)
	// A second traversal yields fresh wrappers for the same namespace nodes.
	nsAgain, err := ixpath.TraverseAxis(t.Context(), ixpath.AxisNamespace, e, ixpath.DefaultMaxNodeSetLength)
	require.NoError(t, err)
	ops := [][]helium.Node{
		nil, attrs, kids, ns, nsAgain, slices.Concat(attrs, kids), slices.Concat(nsAgain, kids), slices.Concat(ns, attrs, kids),
		slices.Concat(kids, kids), slices.Concat(attrs, attrs), slices.Concat(kids, attrs),
		{e}, slices.Concat([]helium.Node{e}, attrs),
	}
	if len(attrs) > 1 {
		ops = append(ops, attrs[:1], attrs[1:], reversedNodes(attrs))
	}
	if len(kids) > 1 {
		ops = append(ops, kids[:1], kids[1:], kids[len(kids)-1:], reversedNodes(kids),
			slices.Concat(attrs[min(1, len(attrs)):], kids[1:]))
	}
	if len(ns) > 1 {
		ops = append(ops, ns[:1], ns[1:])
	}
	if p, ok := e.Parent().(*helium.Element); ok && p != nil {
		ops = append(ops, attrNodes(p), slices.Collect(helium.Children(p)))
	}
	oe := other.DocumentElement()
	ops = append(ops, attrNodes(oe), slices.Collect(helium.Children(oe)),
		slices.Concat(attrNodes(oe), slices.Collect(helium.Children(oe))))
	return ops
}

// reversedNodes returns a reversed copy of nodes.
func reversedNodes(nodes []helium.Node) []helium.Node {
	r := slices.Clone(nodes)
	slices.Reverse(r)
	return r
}

// requireSameUnion checks that UnionNodeSets returns what MergeNodeSets
// returns for a and b, and leaves the cache ordering the two documents the
// same way.
func requireSameUnion(t *testing.T, a, b []helium.Node, doc, other *helium.Document, prebuild bool) {
	t.Helper()
	wantCache := &ixpath.DocOrderCache{}
	gotCache := &ixpath.DocOrderCache{}
	if prebuild {
		wantCache.BuildFrom(doc)
		gotCache.BuildFrom(doc)
	}
	want, wantErr := ixpath.MergeNodeSets(a, b, wantCache, ixpath.DefaultMaxNodeSetLength)
	got, gotErr := ixpath.UnionNodeSets(slices.Concat(a, b), len(a), gotCache, ixpath.DefaultMaxNodeSetLength)
	require.Equal(t, wantErr, gotErr)
	require.Equal(t, want, got)

	// Registration order shows in how the caches order the two documents.
	probe := []helium.Node{other.DocumentElement(), doc.DocumentElement()}
	wantOrder, err := ixpath.MergeNodeSets(probe[:1], probe[1:], wantCache, ixpath.DefaultMaxNodeSetLength)
	require.NoError(t, err)
	gotOrder, err := ixpath.MergeNodeSets(probe[:1], probe[1:], gotCache, ixpath.DefaultMaxNodeSetLength)
	require.NoError(t, err)
	require.Equal(t, wantOrder, gotOrder)

	if len(want) == 0 {
		return
	}
	_, wantErr = ixpath.MergeNodeSets(a, b, &ixpath.DocOrderCache{}, len(want)-1)
	_, gotErr = ixpath.UnionNodeSets(slices.Concat(a, b), len(a), &ixpath.DocOrderCache{}, len(want)-1)
	require.ErrorIs(t, wantErr, ixpath.ErrNodeSetLimit)
	require.ErrorIs(t, gotErr, ixpath.ErrNodeSetLimit)
}

// TestUnionNodeSets compares UnionNodeSets with MergeNodeSets for every pair
// of operands built around every element of documents with namespaces,
// mixed content and entity content, with a fresh and a prebuilt cache.
func TestUnionNodeSets(t *testing.T) {
	srcs := []struct {
		name  string
		src   string
		subst bool
	}{
		{name: "plain", src: stepOrderSrc},
		{name: "entity", src: stepOrderEntitySrc},
		{name: "entity-subst", src: stepOrderEntitySrc, subst: true},
	}
	other := parseStepOrderDoc(t, unionOtherSrc)
	for _, s := range srcs {
		t.Run(s.name, func(t *testing.T) {
			doc, err := helium.NewParser().SubstituteEntities(s.subst).Parse(t.Context(), []byte(s.src))
			require.NoError(t, err)
			for _, e := range unionElements(doc) {
				ops := unionOperands(t, e, other)
				for _, a := range ops {
					for _, b := range ops {
						requireSameUnion(t, a, b, doc, other, false)
						requireSameUnion(t, a, b, doc, other, true)
					}
				}
			}
		})
	}
}
