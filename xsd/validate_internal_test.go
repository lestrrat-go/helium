package xsd

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// TestCountPSVINodes checks the element and attribute counts validateDocument
// sizes the per-run PSVI maps with.
func TestCountPSVINodes(t *testing.T) {
	t.Parallel()

	t.Run("counts every element tree", func(t *testing.T) {
		t.Parallel()
		const src = `<?xml version="1.0"?>
<!DOCTYPE r [<!ENTITY e "<x a='1'/>">]>
<r xmlns:p="urn:p" p:a="1" b="2"><c d="3">text<e/><f g="4" h="5"/></c><!-- c -->&e;<i/></r>`
		doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
		require.NoError(t, err)

		h := countPSVINodes(doc)
		// r, c, e, f, i: the element inside the unexpanded entity's declared
		// content is not part of the instance tree and is not counted. xmlns
		// declarations are not attributes.
		require.Equal(t, 5, h.elems)
		require.Equal(t, 5, h.attrs)
	})

	t.Run("counts every document-level element", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		for _, name := range []string{"a", "b"} {
			elem, err := doc.CreateElement(name)
			require.NoError(t, err)
			require.NoError(t, elem.SetAttribute("n", "1"))
			child, err := doc.CreateElement("c")
			require.NoError(t, err)
			require.NoError(t, elem.AddChild(child))
			require.NoError(t, doc.AddChild(elem))
		}

		h := countPSVINodes(doc)
		require.Equal(t, 4, h.elems)
		require.Equal(t, 2, h.attrs)
	})

	t.Run("stops at the budget", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		root, err := doc.CreateElement("r")
		require.NoError(t, err)
		require.NoError(t, doc.AddChild(root))
		for range psviSizeHintBudget + 10 {
			child, err := doc.CreateElement("c")
			require.NoError(t, err)
			require.NoError(t, root.AddChild(child))
		}

		h := countPSVINodes(doc)
		require.Equal(t, psviSizeHintBudget, h.elems)
		require.Zero(t, h.budget)
	})
}

// TestCountPSVINodesAllocs checks that the count allocates nothing. It is not
// parallel because testing.AllocsPerRun refuses to run in a parallel test.
func TestCountPSVINodesAllocs(t *testing.T) {
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<r a="1"><c b="2"><d/></c><c/></r>`))
	require.NoError(t, err)
	allocs := testing.AllocsPerRun(100, func() {
		countPSVINodes(doc)
	})
	require.Zero(t, allocs)
}
