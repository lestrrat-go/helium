package helium_test

import (
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// TestSerializerNSReconcile covers the serializer-level guarantee that an
// element serializes AT MOST ONE xmlns:<prefix> regardless of which mutator
// created a prefix conflict. The conflict a well-formed DOM cannot express in
// its own setters is introduced by SetActiveNamespace/SetNs AFTER a declaration:
// nsDefs binds prefix→X while the active namespace binds the same prefix→Y.
func TestSerializerNSReconcile(t *testing.T) {
	t.Parallel()

	t.Run("active vs nsDefs conflict: one xmlns:p, active wins", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		root := doc.CreateElement("root")
		require.NoError(t, doc.SetDocumentElement(root))

		require.NoError(t, root.DeclareNamespace("p", "urn:declared"))
		// SetActiveNamespace rebinds p on the active namespace to a different URI.
		// The DOM setters reject this via DeclareNamespace, but SetActiveNamespace
		// sets n.ns independently and does not, so the conflict reaches the writer.
		require.NoError(t, root.SetActiveNamespace("p", "urn:active"))

		str := serializeAndReparse(t, doc)
		require.Equal(t, 1, strings.Count(str, `xmlns:p=`), "exactly one xmlns:p emitted: %s", str)
		require.Contains(t, str, `xmlns:p="urn:active"`, "active binding wins (element name uses it)")
		require.NotContains(t, str, `urn:declared`, "conflicting nsDefs binding is suppressed")
	})

	t.Run("SetNs object form after declare: one xmlns:p, active wins", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		root := doc.CreateElement("root")
		require.NoError(t, doc.SetDocumentElement(root))

		require.NoError(t, root.DeclareNamespace("p", "urn:declared"))
		root.SetNs(helium.NewNamespace("p", "urn:active"))

		str := serializeAndReparse(t, doc)
		require.Equal(t, 1, strings.Count(str, `xmlns:p=`), "exactly one xmlns:p emitted: %s", str)
		require.Contains(t, str, `xmlns:p="urn:active"`)
	})

	t.Run("default namespace conflict still reconciled", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		root := doc.CreateElement("root")
		require.NoError(t, doc.SetDocumentElement(root))

		require.NoError(t, root.DeclareNamespace("", "urn:d1"))
		require.NoError(t, root.SetActiveNamespace("", "urn:d2"))

		str, err := helium.WriteString(doc)
		require.NoError(t, err)
		require.Equal(t, "<?xml version=\"1.0\"?>\n<root xmlns=\"urn:d2\"/>\n", str)
	})

	t.Run("SetActiveNamespace-first path: DeclareNamespace rejects the conflict", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		root := doc.CreateElement("root")
		require.NoError(t, doc.SetDocumentElement(root))

		require.NoError(t, root.SetActiveNamespace("p", "urn:active"))
		// The DOM setter guards this direction, so the conflict never reaches the
		// writer through DeclareNamespace: the only path to a serialized conflict
		// is SetActiveNamespace/SetNs AFTER declaration.
		require.Error(t, root.DeclareNamespace("p", "urn:declared"))
	})
}

// TestSerializerNSReconcileNoRegression pins the byte-exact output of consistent
// elements (nsDefs and active agree, only nsDefs, or only active) so the
// reconciliation stays a no-op for documents a real parse produces.
func TestSerializerNSReconcileNoRegression(t *testing.T) {
	t.Parallel()

	t.Run("only nsDefs", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		root := doc.CreateElement("root")
		require.NoError(t, doc.SetDocumentElement(root))
		require.NoError(t, root.DeclareNamespace("p", "urn:p"))

		str, err := helium.WriteString(doc)
		require.NoError(t, err)
		require.Equal(t, "<?xml version=\"1.0\"?>\n<root xmlns:p=\"urn:p\"/>\n", str)
	})

	t.Run("nsDefs and active agree", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		root := doc.CreateElement("root")
		require.NoError(t, doc.SetDocumentElement(root))
		require.NoError(t, root.DeclareNamespace("p", "urn:p"))
		require.NoError(t, root.SetActiveNamespace("p", "urn:p"))

		str, err := helium.WriteString(doc)
		require.NoError(t, err)
		require.Equal(t, "<?xml version=\"1.0\"?>\n<p:root xmlns:p=\"urn:p\"/>\n", str)
	})

	t.Run("only active (reconcileOne synthesizes)", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		root := doc.CreateElement("root")
		require.NoError(t, doc.SetDocumentElement(root))
		require.NoError(t, root.SetActiveNamespace("p", "urn:p"))

		str, err := helium.WriteString(doc)
		require.NoError(t, err)
		require.Equal(t, "<?xml version=\"1.0\"?>\n<p:root xmlns:p=\"urn:p\"/>\n", str)
	})
}

// TestSerializerNSReconcileNameAttrConflict pins the conservative resolution of
// the genuinely-unsatisfiable case: the element NAME needs prefix p at URI Y
// while an ATTRIBUTE on the same element needs the same prefix p at URI X (X≠Y).
// One prefix cannot bind two URIs on one start tag, so this cannot be made
// faithful by suppression alone. The writer resolves it deterministically — the
// element name wins (its binding is emitted, exactly once) and the attribute's
// conflicting binding is suppressed — so the output stays well-formed and
// reparses. Faithfully preserving the attribute's distinct URI would require
// synthesizing a fresh prefix, which is out of scope for this reconciliation.
func TestSerializerNSReconcileNameAttrConflict(t *testing.T) {
	t.Parallel()

	doc := helium.NewDefaultDocument()
	root := doc.CreateElement("root")
	require.NoError(t, doc.SetDocumentElement(root))
	require.NoError(t, root.SetActiveNamespace("p", "urn:Y"))
	err := root.SetAttributeNS("a", "v", helium.NewNamespace("p", "urn:X"))
	require.NoError(t, err)

	str := serializeAndReparse(t, doc)
	require.Equal(t, 1, strings.Count(str, `xmlns:p=`), "at most one xmlns:p on the start tag: %s", str)
	require.Contains(t, str, `xmlns:p="urn:Y"`, "element name binding wins")
}
