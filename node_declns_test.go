package helium_test

import (
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// serializeRoot serializes doc and reparses the result to prove the output is
// well-formed (at most one declaration per prefix on any element).
func serializeAndReparse(t *testing.T, doc *helium.Document) string {
	t.Helper()
	str, err := helium.WriteString(doc)
	require.NoError(t, err, "serialize succeeds")
	_, err = helium.NewParser().Parse(t.Context(), []byte(str))
	require.NoError(t, err, "serialized output reparses cleanly: %s", str)
	return str
}

// TestDeclareNamespaceCollapse covers the at-most-one-per-prefix contract for
// DeclareNamespace (cases 1-4) and AddNamespaceDecl.
func TestDeclareNamespaceCollapse(t *testing.T) {
	t.Parallel()

	t.Run("case1 fresh prefix appends", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		root := doc.CreateElement("root")
		require.NoError(t, doc.SetDocumentElement(root))

		require.NoError(t, root.DeclareNamespace("p", "urn:p"))
		require.NoError(t, root.DeclareNamespace("q", "urn:q"))
		require.Len(t, root.Namespaces(), 2, "two distinct prefixes → two declarations")

		str := serializeAndReparse(t, doc)
		require.Equal(t, 1, strings.Count(str, `xmlns:p="urn:p"`))
		require.Equal(t, 1, strings.Count(str, `xmlns:q="urn:q"`))
	})

	t.Run("case2 same prefix and uri is a no-op", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		root := doc.CreateElement("root")
		require.NoError(t, doc.SetDocumentElement(root))

		require.NoError(t, root.DeclareNamespace("p", "urn:p"))
		first := root.Namespaces()[0]
		require.NoError(t, root.DeclareNamespace("p", "urn:p"), "same prefix+uri is idempotent")

		ns := root.Namespaces()
		require.Len(t, ns, 1, "no-op: still exactly one declaration")
		require.Same(t, first, ns[0], "no-op: same slot object, no reallocation")

		str := serializeAndReparse(t, doc)
		require.Equal(t, 1, strings.Count(str, `xmlns:p=`))
	})

	t.Run("case3 rebind unused prefix collapses to one", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		root := doc.CreateElement("root")
		require.NoError(t, doc.SetDocumentElement(root))

		require.NoError(t, root.DeclareNamespace("p", "urn:one"))
		old := root.Namespaces()[0]
		require.NoError(t, root.DeclareNamespace("p", "urn:two"), "prefix not in use → collapse")

		ns := root.Namespaces()
		require.Len(t, ns, 1, "collapse: exactly one declaration for p")
		require.Equal(t, "urn:two", ns[0].URI(), "slot rebound to new uri")
		require.Equal(t, "urn:one", old.URI(), "old Namespace object left unmutated")

		str := serializeAndReparse(t, doc)
		require.Equal(t, 1, strings.Count(str, `xmlns:p=`))
		require.Contains(t, str, `xmlns:p="urn:two"`)
	})

	t.Run("case3 default prefix collapses to one", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		root := doc.CreateElement("root")
		require.NoError(t, doc.SetDocumentElement(root))

		require.NoError(t, root.DeclareNamespace("", "urn:one"))
		require.NoError(t, root.DeclareNamespace("", "urn:two"), "default prefix not in use → collapse")

		ns := root.Namespaces()
		require.Len(t, ns, 1, "collapse: exactly one default declaration")
		require.Equal(t, "urn:two", ns[0].URI())

		str := serializeAndReparse(t, doc)
		require.Equal(t, 1, strings.Count(str, `xmlns=`))
	})

	t.Run("case4a active prefix conflict is rejected", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		root := doc.CreateElement("root")
		require.NoError(t, doc.SetDocumentElement(root))

		require.NoError(t, root.DeclareNamespace("p", "urn:old"))
		require.NoError(t, root.SetActiveNamespace("p", "urn:old"))

		err := root.DeclareNamespace("p", "urn:new")
		require.ErrorIs(t, err, helium.ErrInvalidOperation, "in-use element prefix rebind rejected")

		ns := root.Namespaces()
		require.Len(t, ns, 1, "tree unchanged: still one declaration")
		require.Equal(t, "urn:old", ns[0].URI(), "tree unchanged: declaration keeps old uri")
		require.Equal(t, "urn:old", root.URI(), "expanded name unchanged")
	})

	t.Run("case4b attribute prefix conflict is rejected", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		root := doc.CreateElement("root")
		require.NoError(t, doc.SetDocumentElement(root))

		require.NoError(t, root.DeclareNamespace("p", "urn:old"))
		attrNS, err := doc.CreateNamespace("p", "urn:old")
		require.NoError(t, err)
		_, err = root.SetAttributeNS("a", "v", attrNS)
		require.NoError(t, err)

		err = root.DeclareNamespace("p", "urn:new")
		require.ErrorIs(t, err, helium.ErrInvalidOperation, "in-use attribute prefix rebind rejected")

		ns := root.Namespaces()
		require.Len(t, ns, 1, "tree unchanged: still one declaration")
		require.Equal(t, "urn:old", ns[0].URI(), "tree unchanged: declaration keeps old uri")
	})

	t.Run("AddNamespaceDecl dedups a retained handle", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		root := doc.CreateElement("root")
		require.NoError(t, doc.SetDocumentElement(root))

		require.NoError(t, root.DeclareNamespace("p", "urn:one"))
		retained := root.Namespaces()[0]
		require.NoError(t, root.DeclareNamespace("p", "urn:two"))
		require.Equal(t, "urn:one", retained.URI(), "collapse did not mutate the retained handle")

		// Re-attaching the retained handle must not reintroduce a duplicate.
		root.AddNamespaceDecl(retained)
		require.Equal(t, "urn:one", retained.URI(), "AddNamespaceDecl did not mutate the caller object")

		ns := root.Namespaces()
		require.Len(t, ns, 1, "AddNamespaceDecl collapses: exactly one declaration for p")

		str := serializeAndReparse(t, doc)
		require.Equal(t, 1, strings.Count(str, `xmlns:p=`))
	})

	t.Run("AddNamespaceDecl same uri is a no-op", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		root := doc.CreateElement("root")
		require.NoError(t, doc.SetDocumentElement(root))

		require.NoError(t, root.DeclareNamespace("p", "urn:p"))
		first := root.Namespaces()[0]
		dup, err := doc.CreateNamespace("p", "urn:p")
		require.NoError(t, err)
		root.AddNamespaceDecl(dup)

		ns := root.Namespaces()
		require.Len(t, ns, 1, "same prefix+uri is a no-op")
		require.Same(t, first, ns[0], "no-op: original slot retained")
	})

	t.Run("remove then declare yields one declaration", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		root := doc.CreateElement("root")
		require.NoError(t, doc.SetDocumentElement(root))

		// The xslt3 rebind pattern: remove-first, then declare the new binding.
		require.NoError(t, root.DeclareNamespace("p", "urn:old"))
		require.True(t, root.RemoveNamespaceByPrefix("p"))
		require.NoError(t, root.DeclareNamespace("p", "urn:new"))

		ns := root.Namespaces()
		require.Len(t, ns, 1)
		require.Equal(t, "urn:new", ns[0].URI())

		str := serializeAndReparse(t, doc)
		require.Equal(t, 1, strings.Count(str, `xmlns:p=`))
		require.Contains(t, str, `xmlns:p="urn:new"`)
	})
}
