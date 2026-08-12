package helium_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/stretchr/testify/require"
)

func TestWriteNamespaces(t *testing.T) {
	t.Parallel()

	t.Run("inherited namespaces", func(t *testing.T) {
		t.Run("seeded prefix is not re-declared on a using element", func(t *testing.T) {
			t.Parallel()
			// A fragment whose prefix is bound only on an ancestor outside the output:
			// seeding that binding suppresses the otherwise-synthesized re-declaration.
			doc, err := helium.NewParser().Parse(t.Context(),
				[]byte(`<root xmlns:p="urn:p"><child><p:leaf/></child></root>`))
			require.NoError(t, err)
			root := doc.DocumentElement()
			require.NotNil(t, root)
			child := root.FirstChild()
			require.NotNil(t, child)

			var b bytes.Buffer
			w := helium.NewWriter().XMLDeclaration(false).
				InheritedNamespaces(map[string]string{"p": "urn:p"})
			require.NoError(t, w.WriteTo(&b, child))
			require.Equal(t, `<child><p:leaf/></child>`, b.String())
		})

		t.Run("without seeding the inherited prefix is re-declared", func(t *testing.T) {
			t.Parallel()
			doc, err := helium.NewParser().Parse(t.Context(),
				[]byte(`<root xmlns:p="urn:p"><child><p:leaf/></child></root>`))
			require.NoError(t, err)
			child := doc.DocumentElement().FirstChild()
			require.NotNil(t, child)

			var b bytes.Buffer
			w := helium.NewWriter().XMLDeclaration(false)
			require.NoError(t, w.WriteTo(&b, child))
			require.Contains(t, b.String(), `xmlns:p="urn:p"`)
		})
	})

	t.Run("active default namespace", func(t *testing.T) {
		t.Run("active default namespace emits xmlns", func(t *testing.T) {
			t.Parallel()
			doc := helium.NewDefaultDocument()
			root, err := doc.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, doc.SetDocumentElement(root))
			require.NoError(t, root.SetActiveNamespace("", "urn:x"))

			str, err := helium.WriteString(doc)
			require.NoError(t, err)
			require.Contains(t, str, `xmlns="urn:x"`)
		})

		t.Run("active prefixed namespace still emits xmlns:p", func(t *testing.T) {
			t.Parallel()
			doc := helium.NewDefaultDocument()
			root, err := doc.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, doc.SetDocumentElement(root))
			require.NoError(t, root.SetActiveNamespace("p", "urn:x"))

			str, err := helium.WriteString(doc)
			require.NoError(t, err)
			require.Contains(t, str, `xmlns:p="urn:x"`)
		})

		t.Run("parsed default namespace declared exactly once", func(t *testing.T) {
			t.Parallel()
			doc, err := helium.NewParser().Parse(t.Context(), []byte(`<x xmlns="urn:x"/>`))
			require.NoError(t, err)

			str, err := helium.WriteString(doc)
			require.NoError(t, err)
			require.Equal(t, 1, strings.Count(str, `xmlns="urn:x"`))
		})

		t.Run("unprefixed attribute gains no spurious xmlns", func(t *testing.T) {
			t.Parallel()
			doc := helium.NewDefaultDocument()
			root, err := doc.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, doc.SetDocumentElement(root))
			require.NoError(t, root.SetActiveNamespace("p", "urn:x"))
			err = root.SetAttribute("id", "1")
			require.NoError(t, err)

			str, err := helium.WriteString(doc)
			require.NoError(t, err)
			require.NotContains(t, str, `xmlns=""`)
		})

		t.Run("conflicting declared and active default emits a single reparseable xmlns", func(t *testing.T) {
			t.Parallel()
			doc := helium.NewDefaultDocument()
			root, err := doc.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, doc.SetDocumentElement(root))
			// A declared default that conflicts with the element's active default: the
			// active binding wins and only one xmlns is emitted, so the output reparses.
			require.NoError(t, root.DeclareNamespace("", "urn:declared"))
			require.NoError(t, root.SetActiveNamespace("", "urn:active"))

			str, err := helium.WriteString(doc)
			require.NoError(t, err)
			require.Equal(t, 1, strings.Count(str, "xmlns="), "exactly one default declaration: %s", str)
			require.Contains(t, str, `xmlns="urn:active"`)
			require.NotContains(t, str, `xmlns="urn:declared"`)

			_, err = helium.NewParser().Parse(t.Context(), []byte(str))
			require.NoError(t, err, "serialized output must reparse: %s", str)
		})
	})

	// serializing a subtree
	// re-declares any namespace prefix its elements or attributes use but that was
	// bound only on an ancestor outside the subtree, so the output reparses. This
	// is the situation an XSLT result tree creates when it grafts in nodes from a
	// source document (W3C xslt30 si-lre-029/904/905, si-element-029).
	t.Run("subtree namespaces are reconciled", func(t *testing.T) {
		// firstElem returns the first element child of n, or n itself.
		firstElem := func(n helium.Node) helium.Node {
			for c := n.FirstChild(); c != nil; c = c.NextSibling() {
				if c.Type() == helium.ElementNode {
					return c
				}
			}
			return n
		}

		serializeSubtree := func(t *testing.T, src string) string {
			t.Helper()
			doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
			require.NoError(t, err)
			var buf bytes.Buffer
			err = helium.NewWriter().XMLDeclaration(false).WriteTo(&buf, firstElem(doc.DocumentElement()))
			require.NoError(t, err)
			return buf.String()
		}

		t.Run("prefixed attribute bound on outer ancestor", func(t *testing.T) {
			t.Parallel()
			// gml is declared on the root; the serialized <b:elem> subtree uses it
			// only on the gml:id attribute. Both b (element prefix) and gml (attr
			// prefix) must be declared in the fragment.
			out := serializeSubtree(t, `<root xmlns:gml="urn:g" xmlns:b="urn:b"><b:elem gml:id="x1"/></root>`)
			require.Contains(t, out, `xmlns:gml="urn:g"`)
			require.Contains(t, out, `xmlns:b="urn:b"`)
			_, err := helium.NewParser().Parse(t.Context(), []byte(out))
			require.NoError(t, err, "reconciled fragment must reparse: %q", out)
		})

		t.Run("prefix used only on element name", func(t *testing.T) {
			t.Parallel()
			out := serializeSubtree(t, `<root xmlns:p="urn:p"><p:child>text</p:child></root>`)
			require.Contains(t, out, `xmlns:p="urn:p"`)
			_, err := helium.NewParser().Parse(t.Context(), []byte(out))
			require.NoError(t, err, "reconciled fragment must reparse: %q", out)
		})

		t.Run("locally declared prefix not duplicated", func(t *testing.T) {
			t.Parallel()
			// gml is declared on <b:elem> itself; reconciliation must not emit a
			// second xmlns:gml.
			out := serializeSubtree(t, `<root xmlns:b="urn:b"><b:elem xmlns:gml="urn:g" gml:id="x1"/></root>`)
			require.Equal(t, 1, strings.Count(out, `xmlns:gml=`), "no duplicate declaration: %q", out)
			require.Contains(t, out, `xmlns:b="urn:b"`)
			_, err := helium.NewParser().Parse(t.Context(), []byte(out))
			require.NoError(t, err, "reconciled fragment must reparse: %q", out)
		})

		t.Run("xml prefix never reconciled", func(t *testing.T) {
			t.Parallel()
			// xml:lang uses the implicitly-bound xml prefix; the serializer must not
			// synthesize xmlns:xml.
			out := serializeSubtree(t, `<root xmlns:b="urn:b"><b:elem xml:lang="en"/></root>`)
			require.NotContains(t, out, `xmlns:xml=`, "xml prefix is implicitly bound: %q", out)
			_, err := helium.NewParser().Parse(t.Context(), []byte(out))
			require.NoError(t, err, "reconciled fragment must reparse: %q", out)
		})
	})

	t.Run("xml prefix is skipped", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		root, err := doc.CreateElement("root")
		require.NoError(t, err)
		require.NoError(t, doc.SetDocumentElement(root))

		// Add explicit xml: namespace declaration to the element
		require.NoError(t, root.DeclareNamespace("xml", lexicon.NamespaceXML))

		str, err := helium.WriteString(doc)
		require.NoError(t, err)

		// The xml: namespace declaration must NOT appear in the output.
		// libxml2's xmlNsDumpOutput skips prefix "xml" unconditionally.
		require.NotContains(t, str, "xmlns:xml")
	})

	t.Run("write errors propagate", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		root, err := doc.CreateElement("root")
		require.NoError(t, err)
		require.NoError(t, doc.SetDocumentElement(root))
		require.NoError(t, root.DeclareNamespace("p", "urn:test"))

		writer := helium.NewWriter().XMLDeclaration(false)
		err = writer.WriteTo(&namespaceFailWriter{failOn: "xmlns"}, doc)
		require.ErrorIs(t, err, errNamespaceWrite)
	})
}

func TestSerializerNSReconcile(t *testing.T) {
	t.Parallel()

	// the serializer-level guarantee that an
	// element serializes AT MOST ONE xmlns:<prefix> regardless of which mutator
	// created a prefix conflict. The conflict a well-formed DOM cannot express in
	// its own setters is introduced by SetActiveNamespace/SetNamespace AFTER a declaration:
	// nsDefs binds prefix→X while the active namespace binds the same prefix→Y.
	t.Run("reconcile", func(t *testing.T) {
		t.Run("active vs nsDefs conflict: one xmlns:p, active wins", func(t *testing.T) {
			t.Parallel()
			doc := helium.NewDefaultDocument()
			root, err := doc.CreateElement("root")
			require.NoError(t, err)
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

		t.Run("SetNamespace object form after declare: one xmlns:p, active wins", func(t *testing.T) {
			t.Parallel()
			doc := helium.NewDefaultDocument()
			root, err := doc.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, doc.SetDocumentElement(root))

			require.NoError(t, root.DeclareNamespace("p", "urn:declared"))
			root.SetNamespace(helium.NewNamespace("p", "urn:active"))

			str := serializeAndReparse(t, doc)
			require.Equal(t, 1, strings.Count(str, `xmlns:p=`), "exactly one xmlns:p emitted: %s", str)
			require.Contains(t, str, `xmlns:p="urn:active"`)
		})

		t.Run("default namespace conflict still reconciled", func(t *testing.T) {
			t.Parallel()
			doc := helium.NewDefaultDocument()
			root, err := doc.CreateElement("root")
			require.NoError(t, err)
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
			root, err := doc.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, doc.SetDocumentElement(root))

			require.NoError(t, root.SetActiveNamespace("p", "urn:active"))
			// The DOM setter guards this direction, so the conflict never reaches the
			// writer through DeclareNamespace: the only path to a serialized conflict
			// is SetActiveNamespace/SetNamespace AFTER declaration.
			require.Error(t, root.DeclareNamespace("p", "urn:declared"))
		})
	})

	// pins the conservative resolution of
	// the genuinely-unsatisfiable case: the element NAME needs prefix p at URI Y
	// while an ATTRIBUTE on the same element needs the same prefix p at URI X (X≠Y).
	// One prefix cannot bind two URIs on one start tag, so this cannot be made
	// faithful by suppression alone. The writer resolves it deterministically — the
	// element name wins (its binding is emitted, exactly once) and the attribute's
	// conflicting binding is suppressed — so the output stays well-formed and
	// reparses. Faithfully preserving the attribute's distinct URI would require
	// synthesizing a fresh prefix, which is out of scope for this reconciliation.
	t.Run("name attribute conflict", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		root, err := doc.CreateElement("root")
		require.NoError(t, err)
		require.NoError(t, doc.SetDocumentElement(root))
		require.NoError(t, root.SetActiveNamespace("p", "urn:Y"))
		err = root.SetAttributeNS("a", "v", helium.NewNamespace("p", "urn:X"))
		require.NoError(t, err)

		str := serializeAndReparse(t, doc)
		require.Equal(t, 1, strings.Count(str, `xmlns:p=`), "at most one xmlns:p on the start tag: %s", str)
		require.Contains(t, str, `xmlns:p="urn:Y"`, "element name binding wins")
	})

	// pins the byte-exact output of consistent
	// elements (nsDefs and active agree, only nsDefs, or only active) so the
	// reconciliation stays a no-op for documents a real parse produces.
	t.Run("no regression", func(t *testing.T) {
		t.Run("only nsDefs", func(t *testing.T) {
			t.Parallel()
			doc := helium.NewDefaultDocument()
			root, err := doc.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, doc.SetDocumentElement(root))
			require.NoError(t, root.DeclareNamespace("p", "urn:p"))

			str, err := helium.WriteString(doc)
			require.NoError(t, err)
			require.Equal(t, "<?xml version=\"1.0\"?>\n<root xmlns:p=\"urn:p\"/>\n", str)
		})

		t.Run("nsDefs and active agree", func(t *testing.T) {
			t.Parallel()
			doc := helium.NewDefaultDocument()
			root, err := doc.CreateElement("root")
			require.NoError(t, err)
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
			root, err := doc.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, doc.SetDocumentElement(root))
			require.NoError(t, root.SetActiveNamespace("p", "urn:p"))

			str, err := helium.WriteString(doc)
			require.NoError(t, err)
			require.Equal(t, "<?xml version=\"1.0\"?>\n<p:root xmlns:p=\"urn:p\"/>\n", str)
		})
	})
}
