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
		pick func(a, b, c *helium.Attribute) *helium.Attribute
		want []string
	}{
		{
			name: "first",
			pick: func(a, b, c *helium.Attribute) *helium.Attribute { return a },
			want: []string{"z", "b", "c"},
		},
		{
			name: "middle",
			pick: func(a, b, c *helium.Attribute) *helium.Attribute { return b },
			want: []string{"a", "z", "c"},
		},
		{
			name: "last",
			pick: func(a, b, c *helium.Attribute) *helium.Attribute { return c },
			want: []string{"a", "b", "z"},
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
	require.EqualError(t, err, "cannot replace an attribute with a non-attribute node")

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
