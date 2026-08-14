package helium_test

import (
	"os"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/stretchr/testify/require"
)

func TestDocument(t *testing.T) {
	t.Parallel()

	// the small Document getter/setter methods that
	// are otherwise only touched indirectly.
	t.Run("accessors", func(t *testing.T) {
		doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneExplicitYes)
		require.Equal(t, "UTF-8", doc.Encoding())
		require.Equal(t, "UTF-8", doc.RawEncoding())
		require.Equal(t, "1.0", doc.Version())

		doc.SetEncoding("ISO-8859-1")
		require.Equal(t, "ISO-8859-1", doc.Encoding())
		require.Equal(t, "ISO-8859-1", doc.RawEncoding())

		// Document with no encoding synthesizes "utf8" for Encoding but empty for raw.
		d2 := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)
		require.Equal(t, "utf8", d2.Encoding())
		require.Equal(t, "", d2.RawEncoding())

		doc.SetURL("http://example.com/doc.xml")
		require.Equal(t, "http://example.com/doc.xml", doc.URL())

		doc.SetProperties(helium.DocHTML)
		require.True(t, doc.HasProperty(helium.DocHTML))
		require.Equal(t, helium.DocHTML, doc.Properties())

		doc.SetSkipIDs(true)
		require.True(t, doc.SkipIDs())
		doc.SetSkipIDs(false)
		require.False(t, doc.SkipIDs())

		require.Equal(t, helium.StandaloneExplicitYes, doc.Standalone())

		// AddSibling/Replace on a document are rejected.
		x, err := doc.CreateElement("x")
		require.NoError(t, err)
		require.Error(t, doc.AddSibling(x))
		require.Error(t, doc.Replace())

		// SetTreeDoc on a document is a no-op-ish but must not panic.
		doc.SetTreeDoc(doc)
	})

	t.Run("properties", func(t *testing.T) {
		t.Run("new default document is user-built", func(t *testing.T) {
			t.Parallel()
			doc := helium.NewDefaultDocument()
			require.True(t, doc.HasProperty(helium.DocUserBuilt))
		})

		t.Run("HasProperty requires all requested bits", func(t *testing.T) {
			t.Parallel()
			doc := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)
			doc.SetProperties(helium.DocWellFormed | helium.DocXInclude)

			require.True(t, doc.HasProperty(helium.DocWellFormed))
			require.True(t, doc.HasProperty(helium.DocXInclude))
			require.True(t, doc.HasProperty(helium.DocWellFormed|helium.DocXInclude))
			require.False(t, doc.HasProperty(helium.DocWellFormed|helium.DocDTDValid))
		})
	})

	// builds a parsed document and then frees its slabs. This must
	// be safe and idempotent.
	t.Run("free", func(t *testing.T) {
		in, err := os.ReadFile("test/att12.xml")
		require.NoError(t, err)
		doc, err := helium.NewParser().Parse(t.Context(), in)
		require.NoError(t, err)
		doc.Free()
		doc.Free() // idempotent
	})

	t.Run("HTML document", func(t *testing.T) {
		doc := helium.NewHTMLDocument()
		require.Equal(t, helium.HTMLDocumentNode, doc.Type())
		require.True(t, doc.HasProperty(helium.DocHTML))
	})

	// round-trips each Standalone* constant through
	// NewDocument/Standalone and confirms the constants are distinct.
	t.Run("standalone value space", func(t *testing.T) {
		cases := []helium.DocumentStandaloneType{
			helium.StandaloneExplicitYes,
			helium.StandaloneExplicitNo,
			helium.StandaloneNoXMLDecl,
			helium.StandaloneImplicitNo,
			helium.StandaloneInvalidValue,
		}
		seen := make(map[helium.DocumentStandaloneType]bool)
		for _, s := range cases {
			doc := helium.NewDocument("1.0", "", s)
			require.Equal(t, s, doc.Standalone(), "standalone must round-trip through NewDocument")
			require.False(t, seen[s], "each Standalone* constant must be a distinct value")
			seen[s] = true
		}
	})

	t.Run("document element", func(t *testing.T) {
		t.Run("with element", func(t *testing.T) {
			doc := helium.NewDefaultDocument()
			e, err := doc.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, doc.AddChild(e))

			got := doc.DocumentElement()
			require.Equal(t, e, got)
		})

		t.Run("without element", func(t *testing.T) {
			doc := helium.NewDefaultDocument()
			got := doc.DocumentElement()
			require.Nil(t, got)
		})

		t.Run("PI before element", func(t *testing.T) {
			doc := helium.NewDefaultDocument()
			pi := doc.CreatePI("target", "data")
			require.NoError(t, doc.AddChild(pi))

			e, err := doc.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, doc.AddChild(e))

			got := doc.DocumentElement()
			require.Equal(t, e, got)
		})
	})

	t.Run("set document element", func(t *testing.T) {
		t.Run("literal nil returns ErrNilNode", func(t *testing.T) {
			doc := helium.NewDefaultDocument()
			err := doc.SetDocumentElement(nil)
			require.ErrorIs(t, err, helium.ErrNilNode)
		})

		t.Run("typed nil returns ErrNilNode without panic", func(t *testing.T) {
			doc := helium.NewDefaultDocument()
			var root *helium.Element
			err := doc.SetDocumentElement(root)
			require.ErrorIs(t, err, helium.ErrNilNode)
		})

		t.Run("document self-insertion is rejected and leaves doc untouched", func(t *testing.T) {
			doc := helium.NewDefaultDocument()
			err := doc.SetDocumentElement(doc)
			require.Error(t, err)
			require.Nil(t, doc.Parent(), "rejected insertion must not link the candidate")
			requireNoCycle(t, doc)
		})

		t.Run("non-element node is rejected and leaves doc untouched", func(t *testing.T) {
			for _, tc := range []struct {
				name string
				node helium.MutableNode
			}{
				{"text node", helium.NewDefaultDocument().CreateText([]byte("x"))},
				{"comment node", helium.NewDefaultDocument().CreateComment([]byte("c"))},
			} {
				t.Run(tc.name, func(t *testing.T) {
					doc := helium.NewDefaultDocument()
					err := doc.SetDocumentElement(tc.node)
					require.ErrorIs(t, err, helium.ErrInvalidOperation)
					require.Nil(t, doc.FirstChild(), "rejected non-element must not be linked as a child")
					require.Nil(t, doc.DocumentElement(), "doc has no document element")
				})
			}
		})

		t.Run("element-kind marker that is not a concrete *Element is rejected", func(t *testing.T) {
			doc := helium.NewDefaultDocument()
			// An XIncludeMarker reports ElementNode but is not a real *Element, so it
			// must not become the document element — DocumentElement() would never
			// return it, leaving the document element effectively nil.
			marker := helium.NewXIncludeMarker(doc, helium.ElementNode, "fake-element")
			err := doc.SetDocumentElement(marker)
			require.ErrorIs(t, err, helium.ErrInvalidOperation)
			require.Nil(t, doc.FirstChild(), "spoofed-kind marker must not be linked as a child")
			require.Nil(t, doc.DocumentElement(), "doc has no document element")
		})

		t.Run("nil receiver returns ErrNilNode", func(t *testing.T) {
			var doc *helium.Document
			root, err := helium.NewDefaultDocument().CreateElement("root")
			require.NoError(t, err)
			err = doc.SetDocumentElement(root)
			require.ErrorIs(t, err, helium.ErrNilNode)
		})

		t.Run("replace document element with existing descendant", func(t *testing.T) {
			doc := helium.NewDefaultDocument()
			root, err := doc.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, doc.AddChild(root))

			child, err := doc.CreateElement("child")
			require.NoError(t, err)
			require.NoError(t, root.AddChild(child))

			require.NoError(t, doc.SetDocumentElement(child))
			require.Equal(t, helium.Node(child), doc.DocumentElement())
			require.Nil(t, root.Parent())
			requireNoCycle(t, doc)
			requireNoCycle(t, child)
		})
	})

	t.Run("URL", func(t *testing.T) {
		t.Run("set and get URL", func(t *testing.T) {
			doc := helium.NewDefaultDocument()
			require.Equal(t, "", doc.URL())

			doc.SetURL("http://example.com/doc.xml")
			require.Equal(t, "http://example.com/doc.xml", doc.URL())
		})

		t.Run("URL used as base in NodeGetBase", func(t *testing.T) {
			doc := helium.NewDefaultDocument()
			doc.SetURL("http://example.com/dir/doc.xml")

			root, err := doc.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, doc.AddChild(root))

			base := helium.NodeGetBase(doc, root)
			require.Equal(t, "http://example.com/dir/doc.xml", base)
		})

		t.Run("URL with relative xml:base", func(t *testing.T) {
			doc := helium.NewDefaultDocument()
			doc.SetURL("http://example.com/dir/doc.xml")

			root, err := doc.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, doc.AddChild(root))

			xmlNS := helium.NewNamespace("xml", lexicon.NamespaceXML)
			err = root.SetAttributeNS("base", "sub/", xmlNS)
			require.NoError(t, err)

			base := helium.NodeGetBase(doc, root)
			require.Equal(t, "http://example.com/dir/sub/", base)
		})

		t.Run("URL set during parsing", func(t *testing.T) {
			const input = `<?xml version="1.0"?><root/>`
			p := helium.NewParser().BaseURI("/some/path/doc.xml")
			doc, err := p.Parse(t.Context(), []byte(input))
			require.NoError(t, err)
			require.Equal(t, "/some/path/doc.xml", doc.URL())
		})
	})

	// Document.AppendText, which appends a Text child
	// to the document, merging into a trailing Text node when possible.
	t.Run("append text", func(t *testing.T) {
		doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
		require.NoError(t, doc.AppendText([]byte("hello")))
		require.NoError(t, doc.AppendText([]byte(" world")))

		var found bool
		for c := doc.FirstChild(); c != nil; c = c.NextSibling() {
			if c.Type() == helium.TextNode {
				found = true
				require.Contains(t, string(c.Content()), "hello")
			}
		}
		require.True(t, found, "document gained a text child")
	})
}

func TestDocumentCreateNode(t *testing.T) {
	t.Parallel()

	t.Run("PI owner document", func(t *testing.T) {
		doc := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)
		pi := doc.CreatePI("p", "data")
		require.Same(t, doc, pi.OwnerDocument(), "PI owner document should be the creating document")
	})

	t.Run("char ref rejects an empty name", func(t *testing.T) {
		doc := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)
		// "&" decodes to an empty name; so this must be rejected. Accepting it
		// would produce a degenerate entity-ref node with an empty name.
		ref, err := doc.CreateCharRef("&")
		require.Error(t, err, "CreateCharRef with empty decoded name must return an error")
		require.Nil(t, ref)

		// "&;" likewise decodes to an empty name.
		ref, err = doc.CreateCharRef("&;")
		require.Error(t, err)
		require.Nil(t, ref)
	})
}

func TestGetElementByID(t *testing.T) {
	t.Parallel()

	t.Run("lookup", func(t *testing.T) {
		t.Run("xml:id via parser", func(t *testing.T) {
			t.Parallel()
			const input = `<?xml version="1.0"?>
<root>
  <a xml:id="first">one</a>
  <b xml:id="second">two</b>
</root>`
			p := helium.NewParser()
			doc, err := p.Parse(t.Context(), []byte(input))
			require.NoError(t, err)

			// O(1) lookup via ID table
			elem := doc.GetElementByID("first")
			require.NotNil(t, elem)
			require.Equal(t, "a", elem.LocalName())

			elem = doc.GetElementByID("second")
			require.NotNil(t, elem)
			require.Equal(t, "b", elem.LocalName())

			// Non-existent ID
			elem = doc.GetElementByID("missing")
			require.Nil(t, elem)
		})

		t.Run("DTD-declared ID via parser", func(t *testing.T) {
			t.Parallel()
			const input = `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root (item*)>
  <!ELEMENT item (#PCDATA)>
  <!ATTLIST item eid ID #IMPLIED>
]>
<root>
  <item eid="x1">alpha</item>
  <item eid="x2">beta</item>
</root>`
			p := helium.NewParser().LoadExternalDTD(true).DefaultDTDAttributes(true)
			doc, err := p.Parse(t.Context(), []byte(input))
			require.NoError(t, err)

			elem := doc.GetElementByID("x1")
			require.NotNil(t, elem)
			require.Equal(t, "item", elem.LocalName())

			elem = doc.GetElementByID("x2")
			require.NotNil(t, elem)
			require.Equal(t, "item", elem.LocalName())
		})

		t.Run("fallback tree walk for programmatic documents", func(t *testing.T) {
			t.Parallel()
			// Documents built without parsing have no ID table,
			// so GetElementByID falls back to O(n) tree walk.
			doc := helium.NewDefaultDocument()
			root, err := doc.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, doc.AddChild(root))

			child, err := doc.CreateElement("child")
			require.NoError(t, err)
			ns := helium.NewNamespace("xml", lexicon.NamespaceXML)
			err = child.SetAttributeNS("id", "myid", ns)
			require.NoError(t, err)
			require.NoError(t, root.AddChild(child))

			elem := doc.GetElementByID("myid")
			require.NotNil(t, elem)
			require.Equal(t, "child", elem.LocalName())

			elem = doc.GetElementByID("missing")
			require.Nil(t, elem)
		})

		t.Run("ID table populated on parse", func(t *testing.T) {
			t.Parallel()
			const input = `<?xml version="1.0"?>
<root xml:id="r">
  <child xml:id="c"/>
</root>`
			p := helium.NewParser()
			doc, err := p.Parse(t.Context(), []byte(input))
			require.NoError(t, err)

			require.NotNil(t, doc.GetElementByID("r"))
			require.NotNil(t, doc.GetElementByID("c"))
		})

		t.Run("after parse", func(t *testing.T) {
			t.Parallel()
			const input = `<root xml:id="root-id"><child xml:id="child-id"/></root>`

			doc, err := helium.NewParser().Parse(t.Context(), []byte(input))
			require.NoError(t, err)
			require.NotNil(t, doc.GetElementByID("root-id"))
			require.NotNil(t, doc.GetElementByID("child-id"))
		})

		t.Run("after parse with SkipIDs", func(t *testing.T) {
			t.Parallel()
			const input = `<root xml:id="root-id"><child xml:id="child-id"/></root>`

			doc, err := helium.NewParser().SkipIDs(true).Parse(t.Context(), []byte(input))
			require.NoError(t, err)
			require.Nil(t, doc.GetElementByID("root-id"))
			require.Nil(t, doc.GetElementByID("child-id"))
		})

		t.Run("SetSkipIDs is authoritative over a populated ID table", func(t *testing.T) {
			t.Parallel()
			// A document parsed normally has a populated ID table, so it resolves
			// xml:id values up front.
			const input = `<root xml:id="root-id"><child xml:id="child-id"/></root>`
			doc, err := helium.NewParser().Parse(t.Context(), []byte(input))
			require.NoError(t, err)
			require.NotNil(t, doc.GetElementByID("root-id"))
			require.NotNil(t, doc.GetElementByID("child-id"))

			// Once SkipIDs is set, the document must resolve NO ids — even though the
			// ID table is still populated. idsSkip is authoritative and checked first.
			doc.SetSkipIDs(true)
			require.True(t, doc.SkipIDs())
			require.Nil(t, doc.GetElementByID("root-id"),
				"SetSkipIDs(true) must make GetElementByID return nothing even with a populated ID table")
			require.Nil(t, doc.GetElementByID("child-id"))

			// Clearing it restores resolution against the existing table.
			doc.SetSkipIDs(false)
			require.NotNil(t, doc.GetElementByID("root-id"),
				"SetSkipIDs(false) must restore resolution against the existing ID table")
			require.NotNil(t, doc.GetElementByID("child-id"))
		})

		t.Run("xml:id value is whitespace-normalized", func(t *testing.T) {
			t.Parallel()
			// xml:id is implicitly xs:ID, so its value undergoes tokenized-type
			// normalization: leading/trailing whitespace stripped and internal
			// whitespace runs (incl. TAB/CR/LF) collapsed to a single space. The
			// stored DOM value must be the normalized form so a serialized element
			// carries the collapsed id (xml:id Recommendation §4).
			const input = "<root>\n  <a xml:id=\"  \t\n  first  \"/>\n" +
				"  <b xml:id=\"mid\tdle\"/>\n</root>"
			doc, err := helium.NewParser().Parse(t.Context(), []byte(input))
			require.NoError(t, err)

			a := doc.GetElementByID("first")
			require.NotNil(t, a, "collapsed xml:id must be resolvable")
			require.Equal(t, "a", a.LocalName())
			for _, attr := range a.Attributes() {
				if attr.Name() == lexicon.QNameXMLID {
					require.Equal(t, "first", attr.Value(),
						"stored xml:id value must be collapsed and trimmed")
				}
			}

			b := doc.GetElementByID("mid dle")
			require.NotNil(t, b, "internal-whitespace xml:id collapses to a single space")
			require.Equal(t, "b", b.LocalName())
		})

		t.Run("duplicate id resolves via table to last registered", func(t *testing.T) {
			t.Parallel()
			// Duplicate ids are invalid XML, but the documented behavior of the O(1)
			// table path is that RegisterID overwrites, so a lookup returns the LAST
			// element registered for that value.
			doc := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)
			first, err := doc.CreateElement("a")
			require.NoError(t, err)
			second, err := doc.CreateElement("b")
			require.NoError(t, err)
			doc.RegisterID("dup", first)
			doc.RegisterID("dup", second)

			got := doc.GetElementByID("dup")
			require.Same(t, second, got,
				"table path must return the last-registered element for a duplicate id")
		})
	})

	// documents that IDTable returns the document's own live map (not a
	// copy): a subsequent RegisterID is visible through a previously returned map,
	// and a bare API-built document has no interned table.
	t.Run("ID table", func(t *testing.T) {
		doc := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)
		require.Nil(t, doc.IDTable(), "an API-built document has no interned ID table")

		elem, err := doc.CreateElement("a")
		require.NoError(t, err)
		doc.RegisterID("k1", elem)

		tbl := doc.IDTable()
		require.NotNil(t, tbl)
		require.Same(t, elem, tbl["k1"])

		// The returned map aliases the internal one: a later RegisterID shows through
		// the map already handed out.
		elem2, err := doc.CreateElement("b")
		require.NoError(t, err)
		doc.RegisterID("k2", elem2)
		require.Same(t, elem2, tbl["k2"],
			"IDTable returns the live internal map, so later registrations are visible")
	})

	// the O(n) tree-walk fallback path of
	// GetElementByID for an API-built document (no parser ID table).
	t.Run("tree-walk fallback", func(t *testing.T) {
		doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
		root, err := doc.CreateElement("root")
		require.NoError(t, err)
		require.NoError(t, doc.AddChild(root))

		child, err := doc.CreateElement("child")
		require.NoError(t, err)
		xmlNS := helium.NewNamespace("xml", "http://www.w3.org/XML/1998/namespace")
		err = child.SetAttributeNS("id", "target", xmlNS)
		require.NoError(t, err)
		require.NoError(t, root.AddChild(child))

		require.Nil(t, doc.IDTable()) // not populated for API-built docs

		found := doc.GetElementByID("target")
		require.Same(t, child, found)

		require.Nil(t, doc.GetElementByID("missing"))

		// SkipIDs short-circuits resolution.
		doc.SetSkipIDs(true)
		require.Nil(t, doc.GetElementByID("target"))
	})
}
