package helium_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/stretchr/testify/require"
)

func TestCreateAttribute(t *testing.T) {
	t.Parallel()

	t.Run("rejects colon in name", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()

		// A colon in the name parameter is invalid — the name must be an NCName.
		// Callers should use CreateAttribute(localName, value, ns) with a
		// proper Namespace object instead of passing a QName.
		_, err := doc.CreateAttribute("xml:base", "http://example.com", nil)
		require.Error(t, err)

		// Passing a proper local name should succeed.
		ns := helium.NewNamespace("xml", lexicon.NamespaceXML)
		attr, err := doc.CreateAttribute("base", "http://example.com", ns)
		require.NoError(t, err)
		require.Equal(t, "base", attr.LocalName())
		require.Equal(t, "xml:base", attr.Name())
	})
}

func TestSetAttribute(t *testing.T) {
	t.Parallel()

	t.Run("rejects colon in name", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		elem := doc.CreateElement("root")

		// A colon in the name parameter is invalid — callers should use
		// SetAttributeNS with a proper Namespace object.
		_, err := elem.SetAttribute("xml:space", "preserve")
		require.Error(t, err)

		// Passing a proper local name should succeed.
		_, err = elem.SetAttribute("id", "123")
		require.NoError(t, err)
	})

	t.Run("SetLiteralAttribute rejects colon", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		elem := doc.CreateElement("root")

		// A colon in the name parameter is invalid.
		err := elem.SetLiteralAttribute("xml:lang", "en")
		require.Error(t, err)

		// Passing a proper local name should succeed.
		err = elem.SetLiteralAttribute("lang", "en")
		require.NoError(t, err)
	})

	t.Run("parse setters resolve entity references; literal setters store verbatim", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		elem := doc.CreateElement("root")
		ns := helium.NewNamespace("p", "urn:x")

		// SetAttribute / SetAttributeNS parse the value as attribute-value
		// content, so "&amp;" is resolved to a single '&'.
		_, err := elem.SetAttribute("a", "x&amp;y")
		require.NoError(t, err)
		v, ok := elem.GetAttribute("a")
		require.True(t, ok)
		require.Equal(t, "x&y", v)

		_, err = elem.SetAttributeNS("c", "x&amp;y", ns)
		require.NoError(t, err)
		v, ok = elem.GetAttributeNS("c", "urn:x")
		require.True(t, ok)
		require.Equal(t, "x&y", v)

		// The literal setters store the value verbatim (no entity parsing).
		require.NoError(t, elem.SetLiteralAttribute("b", "x&amp;y"))
		v, ok = elem.GetAttribute("b")
		require.True(t, ok)
		require.Equal(t, "x&amp;y", v)

		require.NoError(t, elem.SetLiteralAttributeNS("d", "x&amp;y", ns))
		v, ok = elem.GetAttributeNS("d", "urn:x")
		require.True(t, ok)
		require.Equal(t, "x&amp;y", v)
	})

	// Each of the four setters must REPLACE a same-name attribute in place:
	// no duplicate appended, the second value wins (not the stale first), the
	// target keeps its original position between its siblings, and the original
	// attribute node is detached. Siblings are added before and after the target
	// so a re-ordering or append regression is caught, and the second value
	// differs from the first so a "success without replacing" regression fails.
	t.Run("SetAttribute replaces in place", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		e := doc.CreateElement("r")

		require.NoError(t, e.SetLiteralAttribute("before", "b0"))
		_, err := e.SetAttribute("a", "1")
		require.NoError(t, err)
		require.NoError(t, e.SetLiteralAttribute("after", "a0"))

		orig, ok := e.FindAttribute(helium.LocalNamePredicate("a"))
		require.True(t, ok)
		require.Equal(t, "1", orig.Value())

		_, err = e.SetAttribute("a", "2")
		require.NoError(t, err)

		attrs := e.Attributes()
		require.Len(t, attrs, 3, "replacement must not append a duplicate")
		require.Equal(t,
			[]string{"before", "a", "after"},
			[]string{attrs[0].LocalName(), attrs[1].LocalName(), attrs[2].LocalName()},
			"order preserved, target stays between its siblings")
		require.Equal(t, "2", attrs[1].Value(), "target holds the second value, not the stale first")
		require.Equal(t, "b0", attrs[0].Value(), "sibling before the target untouched")
		require.Equal(t, "a0", attrs[2].Value(), "sibling after the target untouched")
		require.Nil(t, orig.Parent(), "original target node detached after replacement")
	})

	t.Run("SetLiteralAttribute replaces in place", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		e := doc.CreateElement("r")

		require.NoError(t, e.SetLiteralAttribute("before", "b0"))
		require.NoError(t, e.SetLiteralAttribute("a", "1"))
		require.NoError(t, e.SetLiteralAttribute("after", "a0"))

		orig, ok := e.FindAttribute(helium.LocalNamePredicate("a"))
		require.True(t, ok)
		require.Equal(t, "1", orig.Value())

		require.NoError(t, e.SetLiteralAttribute("a", "2"))

		attrs := e.Attributes()
		require.Len(t, attrs, 3, "replacement must not append a duplicate")
		require.Equal(t,
			[]string{"before", "a", "after"},
			[]string{attrs[0].LocalName(), attrs[1].LocalName(), attrs[2].LocalName()},
			"order preserved, target stays between its siblings")
		require.Equal(t, "2", attrs[1].Value(), "target holds the second value, not the stale first")
		require.Equal(t, "b0", attrs[0].Value(), "sibling before the target untouched")
		require.Equal(t, "a0", attrs[2].Value(), "sibling after the target untouched")
		require.Nil(t, orig.Parent(), "original target node detached after replacement")
	})

	t.Run("SetAttributeNS replaces in place", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		ns := helium.NewNamespace("p", "urn:x")
		e := doc.CreateElement("r")

		require.NoError(t, e.SetLiteralAttribute("before", "b0"))
		_, err := e.SetAttributeNS("a", "1", ns)
		require.NoError(t, err)
		require.NoError(t, e.SetLiteralAttribute("after", "a0"))

		orig := e.GetAttributeNodeNS("a", "urn:x")
		require.NotNil(t, orig)
		require.Equal(t, "1", orig.Value())

		_, err = e.SetAttributeNS("a", "2", ns)
		require.NoError(t, err)

		attrs := e.Attributes()
		require.Len(t, attrs, 3, "replacement must not append a duplicate")
		require.Equal(t,
			[]string{"before", "a", "after"},
			[]string{attrs[0].LocalName(), attrs[1].LocalName(), attrs[2].LocalName()},
			"order preserved, target stays between its siblings")
		require.Equal(t, "2", attrs[1].Value(), "target holds the second value, not the stale first")
		require.Equal(t, "b0", attrs[0].Value(), "sibling before the target untouched")
		require.Equal(t, "a0", attrs[2].Value(), "sibling after the target untouched")
		require.Nil(t, orig.Parent(), "original target node detached after replacement")
	})

	t.Run("SetLiteralAttributeNS replaces in place", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		ns := helium.NewNamespace("p", "urn:x")
		e := doc.CreateElement("r")

		require.NoError(t, e.SetLiteralAttribute("before", "b0"))
		require.NoError(t, e.SetLiteralAttributeNS("a", "1", ns))
		require.NoError(t, e.SetLiteralAttribute("after", "a0"))

		orig := e.GetAttributeNodeNS("a", "urn:x")
		require.NotNil(t, orig)
		require.Equal(t, "1", orig.Value())

		require.NoError(t, e.SetLiteralAttributeNS("a", "2", ns))

		attrs := e.Attributes()
		require.Len(t, attrs, 3, "replacement must not append a duplicate")
		require.Equal(t,
			[]string{"before", "a", "after"},
			[]string{attrs[0].LocalName(), attrs[1].LocalName(), attrs[2].LocalName()},
			"order preserved, target stays between its siblings")
		require.Equal(t, "2", attrs[1].Value(), "target holds the second value, not the stale first")
		require.Equal(t, "b0", attrs[0].Value(), "sibling before the target untouched")
		require.Equal(t, "a0", attrs[2].Value(), "sibling after the target untouched")
		require.Nil(t, orig.Parent(), "original target node detached after replacement")
	})
}

func TestAttributeAType(t *testing.T) {
	t.Run("explicit attributes carry atype", func(t *testing.T) {
		t.Parallel()
		xml := `<?xml version="1.0"?>
<!DOCTYPE person [
  <!ELEMENT person EMPTY>
  <!ATTLIST person
    name CDATA #REQUIRED
    id ID #REQUIRED
    ref IDREF #IMPLIED
    refs IDREFS #IMPLIED
    tok NMTOKEN #IMPLIED
    toks NMTOKENS #IMPLIED
  >
]>
<person name="Alice" id="p1" ref="p1" refs="p1" tok="abc" toks="abc def"/>`

		p := helium.NewParser().DefaultDTDAttributes(true)
		doc, err := p.Parse(t.Context(), []byte(xml))
		require.NoError(t, err)

		root := doc.DocumentElement()
		require.NotNil(t, root)

		expected := map[string]enum.AttributeType{
			"name": enum.AttrCDATA,
			"id":   enum.AttrID,
			"ref":  enum.AttrIDRef,
			"refs": enum.AttrIDRefs,
			"tok":  enum.AttrNmtoken,
			"toks": enum.AttrNmtokens,
		}

		for _, attr := range root.Attributes() {
			want, ok := expected[attr.LocalName()]
			require.True(t, ok, "unexpected attribute %s", attr.LocalName())
			require.Equal(t, want, attr.AType(), "attribute %s should have type %d", attr.LocalName(), want)
		}
	})

	t.Run("default attributes carry atype", func(t *testing.T) {
		t.Parallel()
		xml := `<?xml version="1.0"?>
<!DOCTYPE person [
  <!ELEMENT person EMPTY>
  <!ATTLIST person
    name CDATA "unknown"
    role NMTOKEN "admin"
  >
]>
<person/>`

		p := helium.NewParser().DefaultDTDAttributes(true)
		doc, err := p.Parse(t.Context(), []byte(xml))
		require.NoError(t, err)

		root := doc.DocumentElement()
		require.NotNil(t, root)

		expected := map[string]enum.AttributeType{
			"name": enum.AttrCDATA,
			"role": enum.AttrNmtoken,
		}

		attrs := root.Attributes()
		require.Len(t, attrs, len(expected))
		for _, attr := range attrs {
			want, ok := expected[attr.LocalName()]
			require.True(t, ok, "unexpected attribute %s", attr.LocalName())
			require.Equal(t, want, attr.AType(), "default attribute %s should have type %d", attr.LocalName(), want)
		}
	})

	t.Run("attributes without DTD have AttrInvalid", func(t *testing.T) {
		t.Parallel()
		xml := `<?xml version="1.0"?>
<root attr="value"/>`

		p := helium.NewParser()
		doc, err := p.Parse(t.Context(), []byte(xml))
		require.NoError(t, err)

		root := doc.DocumentElement()
		require.NotNil(t, root)

		attrs := root.Attributes()
		require.Len(t, attrs, 1)
		require.Equal(t, enum.AttrInvalid, attrs[0].AType())
	})

	t.Run("enumeration attributes carry atype", func(t *testing.T) {
		t.Parallel()
		xml := `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root EMPTY>
  <!ATTLIST root color (red|green|blue) "red">
]>
<root color="green"/>`

		p := helium.NewParser().DefaultDTDAttributes(true)
		doc, err := p.Parse(t.Context(), []byte(xml))
		require.NoError(t, err)

		root := doc.DocumentElement()
		require.NotNil(t, root)

		attrs := root.Attributes()
		require.Len(t, attrs, 1)
		require.Equal(t, enum.AttrEnumeration, attrs[0].AType())
	})
}

func TestDTDDefaultNamespaceDoesNotOverrideExplicit(t *testing.T) {
	t.Parallel()

	t.Run("explicit default xmlns wins over DTD default", func(t *testing.T) {
		t.Parallel()
		xml := `<!DOCTYPE r [<!ATTLIST r xmlns CDATA "urn:dtd">]><r xmlns="urn:explicit"/>`

		p := helium.NewParser().DefaultDTDAttributes(true)
		doc, err := p.Parse(t.Context(), []byte(xml))
		require.NoError(t, err)

		root := doc.DocumentElement()
		require.NotNil(t, root)
		require.Equal(t, "urn:explicit", root.URI())
	})

	t.Run("explicit prefixed xmlns wins over DTD default", func(t *testing.T) {
		t.Parallel()
		// The ATTLIST must be declared for the qualified element name "p:r"
		// (the DTD-default lookup key for prefixed elements is the literal
		// "prefix:local"); otherwise the default never applies and the test
		// would pass regardless of the fix.
		xml := `<!DOCTYPE p:r [<!ATTLIST p:r xmlns:p CDATA "urn:dtd">]><p:r xmlns:p="urn:explicit"/>`

		p := helium.NewParser().DefaultDTDAttributes(true)
		doc, err := p.Parse(t.Context(), []byte(xml))
		require.NoError(t, err)

		root := doc.DocumentElement()
		require.NotNil(t, root)
		require.Equal(t, "urn:explicit", root.URI())
	})

	t.Run("DTD default applies when no explicit declaration", func(t *testing.T) {
		t.Parallel()
		xml := `<!DOCTYPE r [<!ATTLIST r xmlns CDATA "urn:dtd">]><r/>`

		p := helium.NewParser().DefaultDTDAttributes(true)
		doc, err := p.Parse(t.Context(), []byte(xml))
		require.NoError(t, err)

		root := doc.DocumentElement()
		require.NotNil(t, root)
		require.Equal(t, "urn:dtd", root.URI())
	})

	t.Run("prefixed DTD default applies when no explicit declaration", func(t *testing.T) {
		t.Parallel()
		xml := `<!DOCTYPE p:r [<!ATTLIST p:r xmlns:p CDATA "urn:dtd">]><p:r/>`

		p := helium.NewParser().DefaultDTDAttributes(true)
		doc, err := p.Parse(t.Context(), []byte(xml))
		require.NoError(t, err)

		root := doc.DocumentElement()
		require.NotNil(t, root)
		require.Equal(t, "urn:dtd", root.URI())
	})

	t.Run("explicit reserved xml prefix wins over DTD default", func(t *testing.T) {
		t.Parallel()
		// An explicit reserved-prefix declaration (xmlns:xml=...) takes an
		// early SkipNS shortcut during parsing. The fix records it in
		// nsDeclared so a conflicting DTD default for the same prefix is
		// suppressed; otherwise xml:lang would bind to "urn:dtd" instead of
		// the reserved XML namespace.
		xml := `<!DOCTYPE r [<!ATTLIST r xmlns:xml CDATA "urn:dtd">]>` +
			`<r xmlns:xml="http://www.w3.org/XML/1998/namespace" xml:lang="en"/>`

		p := helium.NewParser().DefaultDTDAttributes(true)
		doc, err := p.Parse(t.Context(), []byte(xml))
		require.NoError(t, err)

		root := doc.DocumentElement()
		require.NotNil(t, root)

		var lang *helium.Attribute
		for _, a := range root.Attributes() {
			if a.Name() == "xml:lang" {
				lang = a
				break
			}
		}
		require.NotNil(t, lang)
		require.Equal(t, "http://www.w3.org/XML/1998/namespace", lang.URI())
	})

	t.Run("conflicting DTD default xmlns:xml is rejected without explicit decl", func(t *testing.T) {
		t.Parallel()
		// No explicit xmlns:xml on the tag. A DTD-defaulted xmlns:xml that maps
		// the reserved xml prefix to the wrong URI must be rejected before it is
		// pushed; otherwise xml:lang would resolve through "urn:dtd" instead of
		// the reserved XML namespace.
		xml := `<!DOCTYPE r [<!ATTLIST r xmlns:xml CDATA "urn:dtd">]>` +
			`<r xml:lang="en"/>`

		p := helium.NewParser().DefaultDTDAttributes(true)
		_, err := p.Parse(t.Context(), []byte(xml))
		require.Error(t, err)
	})

	t.Run("well-formed DTD default xmlns:xml keeps the implicit XML namespace", func(t *testing.T) {
		t.Parallel()
		// A DTD-defaulted xmlns:xml that maps to the correct URI must not shadow
		// the implicit reserved binding; xml:lang stays in the XML namespace.
		xml := `<!DOCTYPE r [<!ATTLIST r xmlns:xml CDATA "http://www.w3.org/XML/1998/namespace">]>` +
			`<r xml:lang="en"/>`

		p := helium.NewParser().DefaultDTDAttributes(true)
		doc, err := p.Parse(t.Context(), []byte(xml))
		require.NoError(t, err)

		root := doc.DocumentElement()
		require.NotNil(t, root)

		var lang *helium.Attribute
		for _, a := range root.Attributes() {
			if a.Name() == "xml:lang" {
				lang = a
				break
			}
		}
		require.NotNil(t, lang)
		require.Equal(t, "http://www.w3.org/XML/1998/namespace", lang.URI())
	})

	t.Run("DTD default xmlns:xmlns is rejected", func(t *testing.T) {
		t.Parallel()
		xml := `<!DOCTYPE r [<!ATTLIST r xmlns:xmlns CDATA "urn:dtd">]><r/>`

		p := helium.NewParser().DefaultDTDAttributes(true)
		_, err := p.Parse(t.Context(), []byte(xml))
		require.Error(t, err)
	})

	t.Run("DTD default prefixed xmlns with empty URI creates no binding", func(t *testing.T) {
		t.Parallel()
		// An empty default value is never registered as an attribute default
		// (mirrors libxml2: empty defaults are not applied), so the empty-URI
		// case cannot reach the defaulting path and no namespace is pushed.
		xml := `<!DOCTYPE r [<!ATTLIST r xmlns:p CDATA "">]><r/>`

		p := helium.NewParser().DefaultDTDAttributes(true)
		doc, err := p.Parse(t.Context(), []byte(xml))
		require.NoError(t, err)

		root := doc.DocumentElement()
		require.NotNil(t, root)
		require.Empty(t, root.URI())
	})

	t.Run("literal non-xml prefix bound to reserved XML namespace is rejected", func(t *testing.T) {
		t.Parallel()
		// Only the xml prefix may bind the reserved XML namespace URI; a
		// literal xmlns:p declaration of it must be rejected.
		xml := `<r xmlns:p="http://www.w3.org/XML/1998/namespace"/>`

		p := helium.NewParser()
		_, err := p.Parse(t.Context(), []byte(xml))
		require.Error(t, err)
	})

	t.Run("DTD default non-xml prefix bound to reserved XML namespace is rejected", func(t *testing.T) {
		t.Parallel()
		// Same constraint via the DTD-defaulting path.
		xml := `<!DOCTYPE r [<!ATTLIST r xmlns:p CDATA "http://www.w3.org/XML/1998/namespace">]><r/>`

		p := helium.NewParser().DefaultDTDAttributes(true)
		_, err := p.Parse(t.Context(), []byte(xml))
		require.Error(t, err)
	})

	t.Run("literal xmlns:xml bound to reserved XML namespace is allowed", func(t *testing.T) {
		t.Parallel()
		// The xml prefix may explicitly bind its reserved namespace URI.
		xml := `<r xmlns:xml="http://www.w3.org/XML/1998/namespace" xml:lang="en"/>`

		p := helium.NewParser()
		doc, err := p.Parse(t.Context(), []byte(xml))
		require.NoError(t, err)

		root := doc.DocumentElement()
		require.NotNil(t, root)

		var lang *helium.Attribute
		for _, a := range root.Attributes() {
			if a.Name() == "xml:lang" {
				lang = a
				break
			}
		}
		require.NotNil(t, lang)
		require.Equal(t, "http://www.w3.org/XML/1998/namespace", lang.URI())
	})
}

// TestNamespaceWellFormednessConstraints covers the Namespaces in XML
// well-formedness constraints on prefix binding and the reserved namespace
// names (W3C xml suite rmt-ns10-026, rmt-ns-e1.0-13a, rmt-ns-e1.0-13b), plus
// the well-formed namespaced documents that must keep parsing.
func TestNamespaceWellFormednessConstraints(t *testing.T) {
	t.Parallel()

	t.Run("unbound attribute prefix is rejected", func(t *testing.T) {
		t.Parallel()
		// rmt-ns10-026: prefix `a` on an attribute is never declared.
		xml := `<?xml version="1.0"?>` + "\n" + `<foo a:attr="1"/>`

		p := helium.NewParser()
		_, err := p.Parse(t.Context(), []byte(xml))
		require.Error(t, err)
		require.Contains(t, err.Error(), "namespace 'a' not found")
	})

	t.Run("reserved xml URI as default namespace is rejected", func(t *testing.T) {
		t.Parallel()
		// rmt-ns-e1.0-13a: the reserved XML namespace name may not be the
		// default namespace.
		xml := `<foo xmlns="http://www.w3.org/XML/1998/namespace"/>`

		p := helium.NewParser()
		_, err := p.Parse(t.Context(), []byte(xml))
		require.Error(t, err)
	})

	t.Run("reserved xmlns URI as default namespace is rejected", func(t *testing.T) {
		t.Parallel()
		// rmt-ns-e1.0-13b: the reserved XMLNS namespace name may not be the
		// default namespace.
		xml := `<foo xmlns="http://www.w3.org/2000/xmlns/"/>`

		p := helium.NewParser()
		_, err := p.Parse(t.Context(), []byte(xml))
		require.Error(t, err)
	})

	t.Run("DTD-defaulted reserved xml URI as default namespace is rejected", func(t *testing.T) {
		t.Parallel()
		xml := `<!DOCTYPE foo [<!ATTLIST foo xmlns CDATA "http://www.w3.org/XML/1998/namespace">]><foo/>`

		p := helium.NewParser().DefaultDTDAttributes(true)
		_, err := p.Parse(t.Context(), []byte(xml))
		require.Error(t, err)
	})

	t.Run("declared attribute prefix is accepted", func(t *testing.T) {
		t.Parallel()
		xml := `<r xmlns:p="urn:x"><p:a p:b="1"/></r>`

		p := helium.NewParser()
		_, err := p.Parse(t.Context(), []byte(xml))
		require.NoError(t, err)
	})

	t.Run("reserved xml prefix on an attribute needs no declaration", func(t *testing.T) {
		t.Parallel()
		// The xml prefix is always bound; xml:lang/xml:space parse with no
		// namespace declaration.
		xml := `<r xml:lang="en" xml:space="preserve"/>`

		p := helium.NewParser()
		_, err := p.Parse(t.Context(), []byte(xml))
		require.NoError(t, err)
	})

	t.Run("ordinary default namespace is accepted", func(t *testing.T) {
		t.Parallel()
		xml := `<r xmlns="urn:d"><c/></r>`

		p := helium.NewParser()
		_, err := p.Parse(t.Context(), []byte(xml))
		require.NoError(t, err)
	})

	t.Run("prefixed attribute with declared prefix is accepted", func(t *testing.T) {
		t.Parallel()
		xml := `<r xmlns:p="urn:x" p:id="5"/>`

		p := helium.NewParser()
		_, err := p.Parse(t.Context(), []byte(xml))
		require.NoError(t, err)
	})
}

func TestGetAttribute(t *testing.T) {
	t.Parallel()

	t.Run("by local name", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		e := doc.CreateElement("root")
		_, err := e.SetAttribute("id", "123")
		require.NoError(t, err)
		_, err = e.SetAttribute("class", "main")
		require.NoError(t, err)

		val, ok := e.GetAttribute("id")
		require.True(t, ok)
		require.Equal(t, "123", val)

		val, ok = e.GetAttribute("class")
		require.True(t, ok)
		require.Equal(t, "main", val)

		_, ok = e.GetAttribute("missing")
		require.False(t, ok)
	})

	t.Run("by namespace", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		e := doc.CreateElement("root")

		ns := helium.NewNamespace("x", "http://example.com")
		_, err := e.SetAttributeNS("attr", "val", ns)
		require.NoError(t, err)

		val, ok := e.GetAttributeNS("attr", "http://example.com")
		require.True(t, ok)
		require.Equal(t, "val", val)

		_, ok = e.GetAttributeNS("attr", "http://other.com")
		require.False(t, ok)

		_, ok = e.GetAttributeNS("missing", "http://example.com")
		require.False(t, ok)
	})
}

func TestHasAttribute(t *testing.T) {
	t.Parallel()
	doc := helium.NewDefaultDocument()
	e := doc.CreateElement("root")
	_, err := e.SetAttribute("id", "123")
	require.NoError(t, err)

	require.True(t, e.HasAttribute("id"))
	require.False(t, e.HasAttribute("missing"))
}

func TestFindAttribute(t *testing.T) {
	t.Parallel()

	t.Run("by predicates", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		e := doc.CreateElement("root")
		_, err := e.SetAttribute("id", "123")
		require.NoError(t, err)

		ns := helium.NewNamespace("x", "http://example.com")
		_, err = e.SetAttributeNS("attr", "val", ns)
		require.NoError(t, err)

		attr, ok := e.FindAttribute(helium.QNamePredicate("id"))
		require.True(t, ok)
		require.NotNil(t, attr)
		require.Equal(t, "123", attr.Value())

		attr, ok = e.FindAttribute(helium.QNamePredicate("x:attr"))
		require.True(t, ok)
		require.NotNil(t, attr)
		require.Equal(t, "val", attr.Value())

		attr, ok = e.FindAttribute(helium.LocalNamePredicate("attr"))
		require.True(t, ok)
		require.NotNil(t, attr)
		require.Equal(t, "x:attr", attr.Name())

		attr, ok = e.FindAttribute(helium.NSPredicate{Local: "attr", NamespaceURI: "http://example.com"})
		require.True(t, ok)
		require.NotNil(t, attr)
		require.Equal(t, "x:attr", attr.Name())
	})

	t.Run("nil predicate", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		e := doc.CreateElement("root")
		_, err := e.SetAttribute("id", "123")
		require.NoError(t, err)

		var pred helium.AttributePredicate
		attr, ok := e.FindAttribute(pred)
		require.False(t, ok)
		require.Nil(t, attr)
	})
}

func TestGetAttributeNodeNS(t *testing.T) {
	t.Parallel()
	doc := helium.NewDefaultDocument()
	e := doc.CreateElement("root")

	ns := helium.NewNamespace("x", "http://example.com")
	_, err := e.SetAttributeNS("attr", "val", ns)
	require.NoError(t, err)

	attr := e.GetAttributeNodeNS("attr", "http://example.com")
	require.NotNil(t, attr)
	require.Equal(t, "attr", attr.LocalName())
	require.Equal(t, "val", attr.Value())
	require.Equal(t, "http://example.com", attr.URI())

	attr = e.GetAttributeNodeNS("attr", "http://other.com")
	require.Nil(t, attr)

	attr = e.GetAttributeNodeNS("missing", "http://example.com")
	require.Nil(t, attr)
}

func TestSetAttributeNSDuplicate(t *testing.T) {
	t.Parallel()

	t.Run("same namespace URI via different Namespace pointers replaces in place", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		e := doc.CreateElement("root")

		// Two distinct *Namespace values that share the same URI. Per XML
		// rules an element may not carry two attributes with the same
		// (namespace URI, local name), regardless of which namespace
		// declaration (pointer) they reference. Like every other Set*Attribute
		// entry point, SetAttributeNS treats these as the SAME attribute and
		// replaces in place rather than appending a second property or erroring.
		ns1 := helium.NewNamespace("a", "http://example.com/ns")
		ns2 := helium.NewNamespace("b", "http://example.com/ns")

		_, err := e.SetAttributeNS("attr", "first", ns1)
		require.NoError(t, err)

		_, err = e.SetAttributeNS("attr", "second", ns2)
		require.NoError(t, err)

		attrs := e.Attributes()
		require.Len(t, attrs, 1)
		require.Equal(t, "second", attrs[0].Value())
		v, ok := e.GetAttributeNS("attr", "http://example.com/ns")
		require.True(t, ok)
		require.Equal(t, "second", v)
	})

	t.Run("genuinely different namespaces are not duplicates", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		e := doc.CreateElement("root")

		ns1 := helium.NewNamespace("a", "http://example.com/ns1")
		ns2 := helium.NewNamespace("b", "http://example.com/ns2")

		_, err := e.SetAttributeNS("attr", "first", ns1)
		require.NoError(t, err)

		_, err = e.SetAttributeNS("attr", "second", ns2)
		require.NoError(t, err)
		require.Len(t, e.Attributes(), 2)
	})
}

func TestSetLiteralAttributeNSDuplicate(t *testing.T) {
	t.Parallel()

	t.Run("same namespace URI via different prefixes replaces in place", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		e := doc.CreateElement("root")

		// Two distinct *Namespace values that share the same URI but differ
		// in prefix. SetLiteralAttributeNS routes through addProperty, which
		// must treat these as the SAME attribute (expanded name {urn:x}a)
		// and replace in place rather than creating a second property that
		// serializes to a different QName (p:a vs q:a) yet has an identical
		// expanded name.
		ns1 := helium.NewNamespace("p", "urn:x")
		ns2 := helium.NewNamespace("q", "urn:x")

		require.NoError(t, e.SetLiteralAttributeNS("a", "first", ns1))
		require.NoError(t, e.SetLiteralAttributeNS("a", "second", ns2))

		attrs := e.Attributes()
		require.Len(t, attrs, 1)
		require.Equal(t, "second", attrs[0].Value())
		v, ok := e.GetAttributeNS("a", "urn:x")
		require.True(t, ok)
		require.Equal(t, "second", v)
	})

	t.Run("genuinely different namespaces coexist", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		e := doc.CreateElement("root")

		ns1 := helium.NewNamespace("p", "urn:x")
		ns2 := helium.NewNamespace("q", "urn:y")

		require.NoError(t, e.SetLiteralAttributeNS("a", "first", ns1))
		require.NoError(t, e.SetLiteralAttributeNS("a", "second", ns2))
		require.Len(t, e.Attributes(), 2)
	})
}

func TestRemoveAttribute(t *testing.T) {
	t.Parallel()

	t.Run("by local name", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		e := doc.CreateElement("root")
		_, err := e.SetAttribute("a", "1")
		require.NoError(t, err)
		_, err = e.SetAttribute("b", "2")
		require.NoError(t, err)
		_, err = e.SetAttribute("c", "3")
		require.NoError(t, err)

		// Remove middle
		ok := e.RemoveAttribute("b")
		require.True(t, ok)
		require.False(t, e.HasAttribute("b"))
		require.True(t, e.HasAttribute("a"))
		require.True(t, e.HasAttribute("c"))

		// Remove first
		ok = e.RemoveAttribute("a")
		require.True(t, ok)
		require.False(t, e.HasAttribute("a"))
		require.True(t, e.HasAttribute("c"))

		// Remove last (only remaining)
		ok = e.RemoveAttribute("c")
		require.True(t, ok)
		require.Equal(t, 0, len(e.Attributes()))

		// Remove non-existent
		ok = e.RemoveAttribute("missing")
		require.False(t, ok)
	})

	t.Run("by namespace", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		e := doc.CreateElement("root")

		ns := helium.NewNamespace("x", "http://example.com")
		_, err := e.SetAttributeNS("attr", "val", ns)
		require.NoError(t, err)

		ok := e.RemoveAttributeNS("attr", "http://example.com")
		require.True(t, ok)
		require.Equal(t, 0, len(e.Attributes()))

		ok = e.RemoveAttributeNS("attr", "http://example.com")
		require.False(t, ok)
	})
}

func TestForEachAttribute(t *testing.T) {
	t.Parallel()
	doc := helium.NewDefaultDocument()
	e := doc.CreateElement("root")

	_, err := e.SetAttribute("a", "1")
	require.NoError(t, err)
	_, err = e.SetAttribute("b", "2")
	require.NoError(t, err)
	_, err = e.SetAttribute("c", "3")
	require.NoError(t, err)

	expected := e.Attributes()
	var iterated []*helium.Attribute
	e.ForEachAttribute(func(attr *helium.Attribute) bool {
		iterated = append(iterated, attr)
		return true
	})
	require.Equal(t, expected, iterated)

	var stopped []*helium.Attribute
	e.ForEachAttribute(func(attr *helium.Attribute) bool {
		stopped = append(stopped, attr)
		return len(stopped) < 2
	})
	require.Equal(t, expected[:2], stopped)
}

// buildAttrElement creates an element carrying attributes a, b, c (in that
// order) and returns the element together with its three attribute nodes.
func buildAttrElement(t *testing.T, doc *helium.Document) (*helium.Element, *helium.Attribute, *helium.Attribute, *helium.Attribute) {
	t.Helper()
	e := doc.CreateElement("root")
	_, err := e.SetAttribute("a", "1")
	require.NoError(t, err, "set attribute a")
	_, err = e.SetAttribute("b", "2")
	require.NoError(t, err, "set attribute b")
	_, err = e.SetAttribute("c", "3")
	require.NoError(t, err, "set attribute c")
	attrs := e.Attributes()
	require.Len(t, attrs, 3, "element starts with three attributes")
	return e, attrs[0], attrs[1], attrs[2]
}

func attrNames(e *helium.Element) []string {
	var names []string
	for _, a := range e.Attributes() {
		names = append(names, a.Name())
	}
	return names
}

// TestAttributeReplaceInPropertyList verifies that replacing an attribute via
// Attribute.Replace updates the owning element's property list (head and sibling
// chain), not the child list. The replacement must be reachable via
// FindAttribute and report the element as its parent, while the replaced node is
// detached.
func TestAttributeReplaceInPropertyList(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		// pick selects which of the three attributes (a, b, c) to replace.
		pick    func(a, b, c *helium.Attribute) *helium.Attribute
		want    []string
		wantXML string
	}{
		{
			name:    "first",
			pick:    func(a, b, c *helium.Attribute) *helium.Attribute { return a },
			want:    []string{"z", "b", "c"},
			wantXML: `<root z="9" b="2" c="3"/>`,
		},
		{
			name:    "middle",
			pick:    func(a, b, c *helium.Attribute) *helium.Attribute { return b },
			want:    []string{"a", "z", "c"},
			wantXML: `<root a="1" z="9" c="3"/>`,
		},
		{
			name:    "last",
			pick:    func(a, b, c *helium.Attribute) *helium.Attribute { return c },
			want:    []string{"a", "b", "z"},
			wantXML: `<root a="1" b="2" z="9"/>`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			doc := helium.NewDefaultDocument()
			e, a, b, c := buildAttrElement(t, doc)
			target := tc.pick(a, b, c)

			repl, err := doc.CreateAttribute("z", "9", nil)
			require.NoError(t, err, "create replacement attribute")

			require.NoError(t, target.Replace(repl), "replace attribute succeeds")

			require.Equal(t, tc.want, attrNames(e), "property list order is updated")

			got, ok := e.FindAttribute(helium.QNamePredicate("z"))
			require.True(t, ok, "replacement is findable via FindAttribute")
			require.Same(t, repl, got, "FindAttribute returns the replacement node")
			require.Equal(t, helium.Node(e), got.Parent(), "replacement parent is the element")

			_, ok = e.FindAttribute(helium.QNamePredicate(target.Name()))
			require.False(t, ok, "replaced attribute no longer in property list")
			require.Nil(t, target.Parent(), "replaced attribute is detached")
			require.Nil(t, target.PrevSibling(), "replaced attribute has no stale prev")
			require.Nil(t, target.NextSibling(), "replaced attribute has no stale next")

			// Serialization reads the element's property list head/chain, so it
			// must reflect the replacement and never the detached old attribute.
			str, err := helium.WriteString(e)
			require.NoError(t, err, "serialize element")
			require.Equal(t, tc.wantXML, str, "serialization reflects replacement, not stale attr")
		})
	}
}

// TestAttributeReplaceRejectsNonAttribute verifies that an attribute may only be
// replaced by attribute nodes; a non-attribute replacement is rejected and the
// property list is left untouched.
func TestAttributeReplaceRejectsNonAttribute(t *testing.T) {
	t.Parallel()

	doc := helium.NewDefaultDocument()
	e, a, _, _ := buildAttrElement(t, doc)

	txt := doc.CreateText([]byte("nope"))
	err := a.Replace(txt)
	require.Error(t, err, "replacing an attribute with a text node must be rejected")
	require.ErrorContains(t, err, "cannot replace an attribute with a non-attribute node")

	require.Equal(t, []string{"a", "b", "c"}, attrNames(e), "property list is untouched")
	require.Equal(t, helium.Node(e), a.Parent(), "attribute still belongs to the element")
	require.Nil(t, txt.Parent(), "text node was not linked in")
}

// TestAttributeAddSiblingMoveRepairsPropertyList verifies that moving an
// attribute via AddSibling (which auto-unlinks it from its old element first)
// repairs the source element's property list. Previously the unlink only fixed
// firstChild/lastChild, leaving Element.properties pointing at the reparented
// attribute so FindAttribute could return a node whose Parent() no longer
// matched.
func TestAttributeAddSiblingMoveRepairsPropertyList(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		// pick selects which source attribute (a, b, c) to move.
		pick func(a, b, c *helium.Attribute) *helium.Attribute
		want []string
	}{
		{
			name: "first",
			pick: func(a, b, c *helium.Attribute) *helium.Attribute { return a },
			want: []string{"b", "c"},
		},
		{
			name: "middle",
			pick: func(a, b, c *helium.Attribute) *helium.Attribute { return b },
			want: []string{"a", "c"},
		},
		{
			name: "last",
			pick: func(a, b, c *helium.Attribute) *helium.Attribute { return c },
			want: []string{"a", "b"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			doc := helium.NewDefaultDocument()
			src, a, b, c := buildAttrElement(t, doc)
			moving := tc.pick(a, b, c)
			movingName := moving.Name()

			dst := doc.CreateElement("dst")
			_, err := dst.SetAttribute("anchor", "0")
			require.NoError(t, err, "create anchor attribute on dst")
			anchor, ok := dst.FindAttribute(helium.QNamePredicate("anchor"))
			require.True(t, ok, "anchor attribute is present on dst")

			// Moving the attribute as anchor's sibling must auto-unlink it from src
			// and repair src's property list head/chain.
			require.NoError(t, anchor.AddSibling(moving), "moving attribute as sibling succeeds")

			require.Equal(t, tc.want, attrNames(src), "source property list no longer holds the moved attribute")
			_, found := src.FindAttribute(helium.QNamePredicate(movingName))
			require.False(t, found, "moved attribute is no longer findable on src")

			require.Equal(t, helium.Node(dst), moving.Parent(), "moved attribute parent is dst")
			require.Equal(t, moving, anchor.NextSibling(), "moved attribute follows the anchor")
			require.Equal(t, helium.Node(anchor), moving.PrevSibling(), "anchor precedes the moved attribute")
		})
	}
}

// childNames returns the names of an element's child nodes (the firstChild ->
// next chain), which is distinct from the attribute property list.
func childNames(e *helium.Element) []string {
	var names []string
	for c := e.FirstChild(); c != nil; c = c.NextSibling() {
		names = append(names, c.Name())
	}
	return names
}

// TestChildListAttributeAddChildMoveRepairsChildList verifies that an attribute
// forced into the normal child list (via UnsafeAppendChild, NOT the property
// list) is auto-unlinked generically when it is later moved. Moving it onto
// another element via AddChild must repair the SOURCE child list's
// firstChild/lastChild (the generic child-list unlink runs, not the property-list
// splice, because the attribute is not reachable from src.properties), while the
// destination AddChild routes the attribute into dst's PROPERTY list.
func TestChildListAttributeAddChildMoveRepairsChildList(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		// pick selects which child-list attribute (a, b, c) to move.
		pick func(a, b, c *helium.Attribute) *helium.Attribute
		want []string
	}{
		{
			name: "first",
			pick: func(a, b, c *helium.Attribute) *helium.Attribute { return a },
			want: []string{"b", "c"},
		},
		{
			name: "middle",
			pick: func(a, b, c *helium.Attribute) *helium.Attribute { return b },
			want: []string{"a", "c"},
		},
		{
			name: "last",
			pick: func(a, b, c *helium.Attribute) *helium.Attribute { return c },
			want: []string{"a", "b"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			doc := helium.NewDefaultDocument()
			src := doc.CreateElement("src")

			a, err := doc.CreateAttribute("a", "1", nil)
			require.NoError(t, err, "create attribute a")
			b, err := doc.CreateAttribute("b", "2", nil)
			require.NoError(t, err, "create attribute b")
			c, err := doc.CreateAttribute("c", "3", nil)
			require.NoError(t, err, "create attribute c")

			// Force the attributes into the CHILD list, bypassing the property-list
			// routing AddChild would otherwise apply to an attribute operand.
			require.NoError(t, helium.UnsafeAppendChild(src, a), "add attribute a as child")
			require.NoError(t, helium.UnsafeAppendChild(src, b), "add attribute b as child")
			require.NoError(t, helium.UnsafeAppendChild(src, c), "add attribute c as child")
			require.Equal(t, []string{"a", "b", "c"}, childNames(src), "attributes start in the child list")
			require.Empty(t, src.Attributes(), "attributes are not in the property list")

			moving := tc.pick(a, b, c)

			dst := doc.CreateElement("dst")
			require.NoError(t, dst.AddChild(moving), "move child-list attribute onto dst")

			require.Equal(t, tc.want, childNames(src), "source child list is repaired")
			require.Equal(t, helium.Node(dst), moving.Parent(), "moved attribute parent is dst")
			// AddChild routes the attribute into dst's property list, not its children.
			require.Empty(t, childNames(dst), "moved attribute is not a child of dst")
			require.Equal(t, []string{moving.Name()}, attrNames(dst), "moved attribute is now a property of dst")

			require.NotEqual(t, helium.Node(moving), src.FirstChild(), "source firstChild is not the moved node")
			require.NotEqual(t, helium.Node(moving), src.LastChild(), "source lastChild is not the moved node")
		})
	}
}

// TestChildListAttributeAddSiblingMoveRepairsChildList verifies that an
// attribute in the normal child list is treated generically when moved via
// AddSibling: the source element's firstChild/lastChild are repaired and the
// node is relinked next to the new sibling.
func TestChildListAttributeAddSiblingMoveRepairsChildList(t *testing.T) {
	t.Parallel()

	doc := helium.NewDefaultDocument()
	src := doc.CreateElement("src")

	a, err := doc.CreateAttribute("a", "1", nil)
	require.NoError(t, err, "create attribute a")
	b, err := doc.CreateAttribute("b", "2", nil)
	require.NoError(t, err, "create attribute b")

	require.NoError(t, helium.UnsafeAppendChild(src, a), "add attribute a as child")
	require.NoError(t, helium.UnsafeAppendChild(src, b), "add attribute b as child")
	require.Equal(t, []string{"a", "b"}, childNames(src), "attributes start in the child list")

	dst := doc.CreateElement("dst")
	anchor := doc.CreateElement("anchor")
	require.NoError(t, dst.AddChild(anchor), "anchor is dst's child")

	// Move the last child-list attribute (b) to sit beside anchor under dst.
	require.NoError(t, anchor.AddSibling(b), "move child-list attribute as sibling of anchor")

	require.Equal(t, []string{"a"}, childNames(src), "source child list is repaired")
	require.Equal(t, helium.Node(src), a.Parent(), "remaining attribute still belongs to src")
	require.Equal(t, helium.Node(a), src.FirstChild(), "source firstChild is the remaining attribute")
	require.Equal(t, helium.Node(a), src.LastChild(), "source lastChild is the remaining attribute")

	require.Equal(t, helium.Node(dst), b.Parent(), "moved attribute parent is dst")
	require.Equal(t, []string{"anchor", "b"}, childNames(dst), "moved attribute follows the anchor in dst")
	require.Equal(t, b, anchor.NextSibling(), "moved attribute follows the anchor")
	require.Equal(t, helium.Node(anchor), b.PrevSibling(), "anchor precedes the moved attribute")
}

// TestChildListAttributeReplaceRepairsChildList verifies that an attribute in
// the normal child list is replaced through generic child-list semantics:
// firstChild/lastChild are repaired, the replacement (which may be a
// non-attribute node) is spliced into the child list, and the replaced node is
// detached. The property-list splice path must not run for a child-list
// attribute.
func TestChildListAttributeReplaceRepairsChildList(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		// pick selects which child-list attribute (a, b, c) to replace.
		pick func(a, b, c *helium.Attribute) *helium.Attribute
		want []string
	}{
		{
			name: "first",
			pick: func(a, b, c *helium.Attribute) *helium.Attribute { return a },
			want: []string{"repl", "b", "c"},
		},
		{
			name: "middle",
			pick: func(a, b, c *helium.Attribute) *helium.Attribute { return b },
			want: []string{"a", "repl", "c"},
		},
		{
			name: "last",
			pick: func(a, b, c *helium.Attribute) *helium.Attribute { return c },
			want: []string{"a", "b", "repl"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			doc := helium.NewDefaultDocument()
			src := doc.CreateElement("src")

			a, err := doc.CreateAttribute("a", "1", nil)
			require.NoError(t, err, "create attribute a")
			b, err := doc.CreateAttribute("b", "2", nil)
			require.NoError(t, err, "create attribute b")
			c, err := doc.CreateAttribute("c", "3", nil)
			require.NoError(t, err, "create attribute c")

			require.NoError(t, helium.UnsafeAppendChild(src, a), "add attribute a as child")
			require.NoError(t, helium.UnsafeAppendChild(src, b), "add attribute b as child")
			require.NoError(t, helium.UnsafeAppendChild(src, c), "add attribute c as child")
			require.Equal(t, []string{"a", "b", "c"}, childNames(src), "attributes start in the child list")

			target := tc.pick(a, b, c)

			// A non-attribute replacement is allowed here: the target lives in the
			// child list, not the property list, so generic child-list semantics
			// apply and the attribute-only restriction does not.
			repl := doc.CreateElement("repl")
			require.NoError(t, target.Replace(repl), "replace child-list attribute succeeds")

			require.Equal(t, tc.want, childNames(src), "source child list is repaired")
			require.Equal(t, helium.Node(src), repl.Parent(), "replacement parent is the element")
			require.Empty(t, src.Attributes(), "no attributes leaked into the property list")

			require.Nil(t, target.Parent(), "replaced attribute is detached")
			require.Nil(t, target.PrevSibling(), "replaced attribute has no stale prev")
			require.Nil(t, target.NextSibling(), "replaced attribute has no stale next")

			require.NotEqual(t, helium.Node(target), src.FirstChild(), "source firstChild is not the replaced node")
			require.NotEqual(t, helium.Node(target), src.LastChild(), "source lastChild is not the replaced node")
		})
	}
}

func TestAttributeAddSibling(t *testing.T) {
	t.Parallel()

	t.Run("property-list AddSibling keeps attributes out of child list", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		elem := doc.CreateElement("root")

		// Seed a property-list attribute as the anchor.
		_, err := elem.SetAttribute("anchor", "1")
		require.NoError(t, err)
		anchor, ok := elem.FindAttribute(helium.QNamePredicate("anchor"))
		require.True(t, ok, "anchor attribute is reachable from properties")

		// A free-floating attribute to splice in via the anchor.
		moving, err := doc.CreateAttribute("moving", "2", nil)
		require.NoError(t, err)

		require.NoError(t, anchor.AddSibling(moving), "property-list AddSibling succeeds")

		// Attributes are NOT children: the owner element's child pointers stay nil.
		require.Nil(t, elem.FirstChild(), "owner firstChild remains nil")
		require.Nil(t, elem.LastChild(), "owner lastChild remains nil")

		// The moving attribute is now reachable in the property list / chain.
		require.Equal(t, helium.Node(elem), moving.Parent(), "moving attribute parent is the owner element")
		require.Equal(t, helium.Node(anchor), moving.PrevSibling(), "moving attribute follows the anchor")
		require.Equal(t, helium.Node(moving), anchor.NextSibling(), "anchor next is the moving attribute")

		found, ok := elem.FindAttribute(helium.QNamePredicate("moving"))
		require.True(t, ok, "moving attribute is found via FindAttribute")
		require.Equal(t, helium.Node(moving), helium.Node(found), "FindAttribute returns the spliced attribute")
	})

	t.Run("property-list AddSibling rejects non-attribute and leaves old tree intact", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		elem := doc.CreateElement("root")

		_, err := elem.SetAttribute("anchor", "1")
		require.NoError(t, err)
		anchor, ok := elem.FindAttribute(helium.QNamePredicate("anchor"))
		require.True(t, ok, "anchor attribute is reachable from properties")

		// A non-MutableNode operand (NamespaceNodeWrapper) parented elsewhere.
		owner := doc.CreateElement("owner")
		ns := helium.NewNamespace("p", "http://example.com/p")
		wrapper := helium.NewNamespaceNodeWrapper(ns, owner)

		err = anchor.AddSibling(wrapper)
		require.Error(t, err, "non-attribute sibling of a property attribute is rejected")

		// The rejected operand's old parent link is untouched.
		require.Equal(t, helium.Node(owner), wrapper.Parent(), "wrapper old parent is untouched")

		// The anchor chain is unchanged and no child leaked in.
		require.Nil(t, anchor.NextSibling(), "anchor has no spurious next sibling")
		require.Nil(t, elem.FirstChild(), "owner firstChild remains nil")
		require.Nil(t, elem.LastChild(), "owner lastChild remains nil")
		require.Len(t, elem.Attributes(), 1, "only the anchor attribute remains")
	})
}

// TestSetBooleanAttribute covers Element.SetBooleanAttribute for both the
// success path (a value-less attribute) and the colon-rejection error path.
func TestSetBooleanAttribute(t *testing.T) {
	t.Parallel()

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root := doc.CreateElement("input")
	require.NoError(t, doc.AddChild(root))

	require.NoError(t, root.SetBooleanAttribute("checked"), "boolean attribute added")

	require.True(t, root.HasAttribute("checked"), "boolean attribute is present")
	val, ok := root.GetAttribute("checked")
	require.True(t, ok, "boolean attribute is readable")
	require.Empty(t, val, "boolean attribute has no value")

	// A colon in the name is rejected.
	require.Error(t, root.SetBooleanAttribute("ns:bad"))
}

// TestAttributeNodeMethods exercises the Attribute node-interface methods.
func TestAttributeNodeMethods(t *testing.T) {
	t.Parallel()

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	attr, err := doc.CreateAttribute("a", "v", nil)
	require.NoError(t, err)

	require.Equal(t, "v", attr.Value())
	require.Equal(t, "a", attr.Name())

	attr.SetAType(enum.AttrCDATA)
	require.Equal(t, enum.AttrCDATA, attr.AType())

	attr.SetDefault(true)
	require.True(t, attr.IsDefault())

	// AppendText extends the attribute value (text child).
	require.NoError(t, attr.AppendText([]byte("-more")))

	attr.SetTreeDoc(doc)
}

// TestCreateAttributeWithEntityValue drives the stringToNodeList path inside
// CreateAttribute by passing a value containing character and entity references.
func TestCreateAttributeWithEntityValue(t *testing.T) {
	t.Parallel()
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)

	// Value with a decimal char ref, a hex char ref, and a named entity ref.
	attr, err := doc.CreateAttribute("a", "x&#65;y&#x42;z&amp;w", nil)
	require.NoError(t, err)
	require.Equal(t, "a", attr.Name())
	// The attribute has a child node list (text + entity refs).
	require.NotNil(t, attr.FirstChild())

	// Plain value (no '&') takes the fast single-text-node path.
	attr2, err := doc.CreateAttribute("b", "plain", nil)
	require.NoError(t, err)
	require.Equal(t, "plain", attr2.Value())

	// Colon in name is rejected.
	_, err = doc.CreateAttribute("ns:x", "v", nil)
	require.Error(t, err)
}
