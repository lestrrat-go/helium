package helium_test

import (
	"os"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/stretchr/testify/require"
)

func TestGetElementByID(t *testing.T) {
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
		root := doc.CreateElement("root")
		require.NoError(t, doc.AddChild(root))

		child := doc.CreateElement("child")
		ns := helium.NewNamespace("xml", lexicon.NamespaceXML)
		err := child.SetAttributeNS("id", "myid", ns)
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
		first := doc.CreateElement("a")
		second := doc.CreateElement("b")
		doc.RegisterID("dup", first)
		doc.RegisterID("dup", second)

		got := doc.GetElementByID("dup")
		require.Same(t, second, got,
			"table path must return the last-registered element for a duplicate id")
	})
}

// TestIDTable documents that IDTable returns the document's own live map (not a
// copy): a subsequent RegisterID is visible through a previously returned map,
// and a bare API-built document has no interned table.
func TestIDTable(t *testing.T) {
	t.Parallel()

	doc := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)
	require.Nil(t, doc.IDTable(), "an API-built document has no interned ID table")

	elem := doc.CreateElement("a")
	doc.RegisterID("k1", elem)

	tbl := doc.IDTable()
	require.NotNil(t, tbl)
	require.Same(t, elem, tbl["k1"])

	// The returned map aliases the internal one: a later RegisterID shows through
	// the map already handed out.
	elem2 := doc.CreateElement("b")
	doc.RegisterID("k2", elem2)
	require.Same(t, elem2, tbl["k2"],
		"IDTable returns the live internal map, so later registrations are visible")
}

// TestStandaloneValueSpace round-trips each Standalone* constant through
// NewDocument/Standalone and confirms the constants are distinct.
func TestStandaloneValueSpace(t *testing.T) {
	t.Parallel()

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
}

func TestDocProperties(t *testing.T) {
	t.Parallel()

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
}

func TestCreatePIOwnerDocument(t *testing.T) {
	t.Parallel()
	doc := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)
	pi := doc.CreatePI("p", "data")
	require.Same(t, doc, pi.OwnerDocument(), "PI owner document should be the creating document")
}

func TestCreateCharRefRejectsEmptyName(t *testing.T) {
	t.Parallel()
	doc := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)
	// "&" decodes to an empty name; this must be rejected rather than
	// producing a degenerate entity-ref node with an empty name.
	ref, err := doc.CreateCharRef("&")
	require.Error(t, err, "CreateCharRef with empty decoded name must return an error")
	require.Nil(t, ref)

	// "&;" likewise decodes to an empty name.
	ref, err = doc.CreateCharRef("&;")
	require.Error(t, err)
	require.Nil(t, ref)
}

// TestDocumentAccessors exercises the small Document getter/setter methods that
// are otherwise only touched indirectly.
func TestDocumentAccessors(t *testing.T) {
	t.Parallel()

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
	require.Error(t, doc.AddSibling(doc.CreateElement("x")))
	require.Error(t, doc.Replace())

	// SetTreeDoc on a document is a no-op-ish but must not panic.
	doc.SetTreeDoc(doc)
}
func TestNewHTMLDocument(t *testing.T) {
	t.Parallel()
	doc := helium.NewHTMLDocument()
	require.Equal(t, helium.HTMLDocumentNode, doc.Type())
	require.True(t, doc.HasProperty(helium.DocHTML))
}

// TestDocumentFree builds a parsed document and then frees its slabs. This must
// be safe and idempotent.
func TestDocumentFree(t *testing.T) {
	t.Parallel()
	in, err := os.ReadFile("test/att12.xml")
	require.NoError(t, err)
	doc, err := helium.NewParser().Parse(t.Context(), in)
	require.NoError(t, err)
	doc.Free()
	doc.Free() // idempotent
}
