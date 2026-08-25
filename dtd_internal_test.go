package helium

import (
	"bytes"
	"testing"

	"github.com/lestrrat-go/helium/enum"
	"github.com/stretchr/testify/require"
)

const (
	attrDeclElem  = "item"
	attrDeclCount = "count"
)

func TestAddAttributeDecl(t *testing.T) {
	// AddAttributeDecl builds a declaration from public parameters, links it into
	// the DTD child list, serializes it as an <!ATTLIST> declaration, and — the
	// acceptance bar — a validating parser accepts the serialized document and
	// recovers each declaration equivalently.
	t.Run("serializes and round-trips", func(t *testing.T) {
		doc := NewDocument("1.0", "UTF-8", StandaloneExplicitNo)
		dtd, err := doc.CreateInternalSubset(attrDeclElem, "", "")
		require.NoError(t, err)

		// The NOTATION attribute below refers to this notation (VC: Notation Attributes),
		// so it must be declared for the round-tripped document to validate.
		_, err = dtd.AddNotation("gif", "", "image/gif")
		require.NoError(t, err)

		// ANY content, not EMPTY: a NOTATION attribute is not allowed on an EMPTY
		// element (VC: No Notation on Empty Element).
		_, err = dtd.AddElementDecl(attrDeclElem, enum.AnyElementType, nil)
		require.NoError(t, err)

		adecl, err := dtd.AddAttributeDecl(attrDeclElem, attrDeclCount, enum.AttrCDATA, enum.AttrDefaultRequired, "", nil)
		require.NoError(t, err)
		require.Equal(t, attrDeclElem, adecl.Elem())
		require.Equal(t, enum.AttrCDATA, adecl.AType())

		// A FIXED default value round-trips.
		_, err = dtd.AddAttributeDecl(attrDeclElem, "unit", enum.AttrCDATA, enum.AttrDefaultFixed, "px", nil)
		require.NoError(t, err)

		// An enumeration type emits its token list.
		_, err = dtd.AddAttributeDecl(attrDeclElem, "kind", enum.AttrEnumeration, enum.AttrDefaultImplied, "", Enumeration{"a", "b"})
		require.NoError(t, err)

		// A NOTATION type emits NOTATION (...).
		_, err = dtd.AddAttributeDecl(attrDeclElem, "note", enum.AttrNotation, enum.AttrDefaultImplied, "", Enumeration{"gif"})
		require.NoError(t, err)

		// The decl is retrievable through the public lookup.
		got, ok := dtd.LookupAttribute(attrDeclCount, "", attrDeclElem)
		require.True(t, ok)
		require.Equal(t, adecl, got)

		// A conforming instance so ValidateDTD accepts the round-tripped document.
		root, err := doc.CreateElement(attrDeclElem)
		require.NoError(t, err)
		err = root.SetAttribute(attrDeclCount, "5")
		require.NoError(t, err)
		require.NoError(t, doc.SetDocumentElement(root))

		var buf bytes.Buffer
		require.NoError(t, Write(&buf, doc))
		out := buf.String()
		require.Contains(t, out, "<!ATTLIST item count CDATA #REQUIRED>")
		require.Contains(t, out, `<!ATTLIST item unit CDATA #FIXED "px">`)
		require.Contains(t, out, "<!ATTLIST item kind (a | b) #IMPLIED>")
		require.Contains(t, out, "<!ATTLIST item note NOTATION (gif) #IMPLIED>")

		// Round-trip: a validating parser accepts the serialized document and recovers
		// each declaration equivalently.
		parsed, err := NewParser().ValidateDTD(true).Parse(t.Context(), buf.Bytes())
		require.NoError(t, err)
		rdtd := parsed.IntSubset()
		require.NotNil(t, rdtd)

		assertAttr := func(name string, atype enum.AttributeType, def enum.AttributeDefault, defvalue string, tree Enumeration) {
			t.Helper()
			d, ok := rdtd.LookupAttribute(name, "", attrDeclElem)
			require.True(t, ok, "recovered decl %q", name)
			require.Equal(t, atype, d.atype, "atype of %q", name)
			require.Equal(t, def, d.def, "def of %q", name)
			require.Equal(t, defvalue, d.defvalue, "defvalue of %q", name)
			require.Equal(t, tree, d.tree, "tree of %q", name)
		}
		assertAttr(attrDeclCount, enum.AttrCDATA, enum.AttrDefaultRequired, "", nil)
		assertAttr("unit", enum.AttrCDATA, enum.AttrDefaultFixed, "px", nil)
		assertAttr("kind", enum.AttrEnumeration, enum.AttrDefaultImplied, "", Enumeration{"a", "b"})
		assertAttr("note", enum.AttrNotation, enum.AttrDefaultImplied, "", Enumeration{"gif"})

		// Serialize the reparsed document again; the ATTLIST lines are identical.
		var buf2 bytes.Buffer
		require.NoError(t, Write(&buf2, parsed))
		out2 := buf2.String()
		require.Contains(t, out2, "<!ATTLIST item count CDATA #REQUIRED>")
		require.Contains(t, out2, `<!ATTLIST item unit CDATA #FIXED "px">`)
		require.Contains(t, out2, "<!ATTLIST item kind (a | b) #IMPLIED>")
		require.Contains(t, out2, "<!ATTLIST item note NOTATION (gif) #IMPLIED>")
	})

	// A prefixed attribute name is split into prefix + local (mirroring
	// AddElementDecl) and serialized as "prefix:local".
	t.Run("prefixed name is split", func(t *testing.T) {
		doc := NewDocument("1.0", "UTF-8", StandaloneExplicitNo)
		dtd, err := doc.CreateInternalSubset(attrDeclElem, "", "")
		require.NoError(t, err)

		_, err = dtd.AddAttributeDecl(attrDeclElem, "x:id", enum.AttrID, enum.AttrDefaultRequired, "", nil)
		require.NoError(t, err)

		// Keyed under local + prefix.
		_, ok := dtd.LookupAttribute("id", "x", attrDeclElem)
		require.True(t, ok)

		var buf bytes.Buffer
		require.NoError(t, Write(&buf, doc))
		require.Contains(t, buf.String(), "<!ATTLIST item x:id ID #REQUIRED>")
	})

	// The token list is cloned, so a caller mutating the slice after the call
	// cannot corrupt the serialized decl.
	t.Run("enumeration tokens are cloned", func(t *testing.T) {
		doc := NewDocument("1.0", "UTF-8", StandaloneExplicitNo)
		dtd, err := doc.CreateInternalSubset(attrDeclElem, "", "")
		require.NoError(t, err)

		toks := Enumeration{"a", "b"}
		adecl, err := dtd.AddAttributeDecl(attrDeclElem, "kind", enum.AttrEnumeration, enum.AttrDefaultImplied, "", toks)
		require.NoError(t, err)

		toks[0] = "MUTATED"
		require.Equal(t, Enumeration{"a", "b"}, adecl.tree)

		var buf bytes.Buffer
		require.NoError(t, Write(&buf, doc))
		require.Contains(t, buf.String(), "<!ATTLIST item kind (a | b) #IMPLIED>")
		require.NotContains(t, buf.String(), "MUTATED")
	})

	// A repeat declaration is rejected with ErrDuplicateDeclaration.
	t.Run("duplicate is rejected", func(t *testing.T) {
		doc := NewDocument("1.0", "UTF-8", StandaloneExplicitNo)
		dtd, err := doc.CreateInternalSubset("root", "", "")
		require.NoError(t, err)

		_, err = dtd.AddAttributeDecl(attrDeclElem, attrDeclCount, enum.AttrCDATA, enum.AttrDefaultImplied, "", nil)
		require.NoError(t, err)

		_, err = dtd.AddAttributeDecl(attrDeclElem, attrDeclCount, enum.AttrCDATA, enum.AttrDefaultRequired, "", nil)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrDuplicateDeclaration)
	})

	// An out-of-range enum parameter is rejected (wrapping ErrInvalidArgument)
	// before registration, so nothing is registered or serialized. Like its sibling
	// constructors, AddAttributeDecl validates only the enum parameters — it trusts
	// the caller for well-formed names and values.
	t.Run("out-of-range enum parameter is rejected", func(t *testing.T) {
		tests := []struct {
			name  string
			atype enum.AttributeType
			def   enum.AttributeDefault
		}{
			{"invalid attribute type", enum.AttributeType(999), enum.AttrDefaultImplied},
			{"invalid default declaration", enum.AttrCDATA, enum.AttributeDefault(999)},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				doc := NewDocument("1.0", "UTF-8", StandaloneExplicitNo)
				dtd, err := doc.CreateInternalSubset(attrDeclElem, "", "")
				require.NoError(t, err)

				adecl, err := dtd.AddAttributeDecl(attrDeclElem, attrDeclCount, tc.atype, tc.def, "", nil)
				require.Error(t, err)
				require.ErrorIs(t, err, ErrInvalidArgument)
				require.Nil(t, adecl)

				// Nothing was registered or serialized.
				require.Empty(t, dtd.attributes)
				var buf bytes.Buffer
				require.NoError(t, Write(&buf, doc))
				require.NotContains(t, buf.String(), "<!ATTLIST")
			})
		}
	})

	// Two distinct declarations whose (local, prefix, elem) triples concatenate to
	// the SAME `local + ":" + prefix + ":" + elem` string are both registered and
	// independently retrievable. The old string key aliased them and wrongly
	// rejected the second as a duplicate; the struct key keeps them distinct.
	//
	// Decl A: name "b:a" -> local "a", prefix "b"; elem "c:d". String key "a:b:c:d".
	// Decl B: name "c:a:b" -> local "a:b", prefix "c"; elem "d". String key "a:b:c:d".
	t.Run("aliasing key triples stay distinct", func(t *testing.T) {
		doc := NewDocument("1.0", "UTF-8", StandaloneExplicitNo)
		dtd, err := doc.CreateInternalSubset("root", "", "")
		require.NoError(t, err)

		_, err = dtd.AddAttributeDecl("c:d", "b:a", enum.AttrCDATA, enum.AttrDefaultImplied, "", nil)
		require.NoError(t, err)
		_, err = dtd.AddAttributeDecl("d", "c:a:b", enum.AttrCDATA, enum.AttrDefaultImplied, "", nil)
		require.NoError(t, err, "a string-aliasing triple must not be rejected as a duplicate")

		a, ok := dtd.LookupAttribute("a", "b", "c:d")
		require.True(t, ok)
		require.Equal(t, "a", a.name)
		require.Equal(t, "b", a.prefix)
		require.Equal(t, "c:d", a.elem)

		b, ok := dtd.LookupAttribute("a:b", "c", "d")
		require.True(t, ok)
		require.Equal(t, "a:b", b.name)
		require.Equal(t, "c", b.prefix)
		require.Equal(t, "d", b.elem)

		require.NotSame(t, a, b)
		require.Len(t, dtd.attributes, 2)
	})

	// A well-formed enumeration whose tokens carry a colon is accepted and
	// round-trips through a validating parser.
	t.Run("colon-bearing enumeration token is accepted", func(t *testing.T) {
		doc := NewDocument("1.0", "UTF-8", StandaloneExplicitNo)
		dtd, err := doc.CreateInternalSubset(attrDeclElem, "", "")
		require.NoError(t, err)
		_, err = dtd.AddElementDecl(attrDeclElem, enum.AnyElementType, nil)
		require.NoError(t, err)

		_, err = dtd.AddAttributeDecl(attrDeclElem, "kind", enum.AttrEnumeration, enum.AttrDefaultImplied, "", Enumeration{"x:a", "x:b"})
		require.NoError(t, err)

		root, err := doc.CreateElement(attrDeclElem)
		require.NoError(t, err)
		require.NoError(t, doc.SetDocumentElement(root))

		var buf bytes.Buffer
		require.NoError(t, Write(&buf, doc))
		require.Contains(t, buf.String(), "<!ATTLIST item kind (x:a | x:b) #IMPLIED>")

		parsed, err := NewParser().ValidateDTD(true).Parse(t.Context(), buf.Bytes())
		require.NoError(t, err)
		d, ok := parsed.IntSubset().LookupAttribute("kind", "", attrDeclElem)
		require.True(t, ok)
		require.Equal(t, Enumeration{"x:a", "x:b"}, d.tree)
	})

	// An accepted value-bearing default is stable across serialize→parse→serialize:
	// the default value the parser recovers is identical, and re-serializing the
	// reparsed document yields the same <!ATTLIST> line. It exercises defaults that
	// round-trip through the default-value serializer — a '<' (escaped as "&lt;" and
	// decoded back), a '"' (escaped as "&quot;"), a CDATA-end sequence, and a plain
	// value.
	t.Run("default value round-trips", func(t *testing.T) {
		for _, tc := range []struct {
			name     string
			defvalue string
			want     string
		}{
			{"plain", "px", `<!ATTLIST item label CDATA #FIXED "px">`},
			{"less-than", "a<b", `<!ATTLIST item label CDATA #FIXED "a&lt;b">`},
			{"double-quote", `a"b`, `<!ATTLIST item label CDATA #FIXED "a&quot;b">`},
			{"cdata-end", "]]>", `<!ATTLIST item label CDATA #FIXED "]]&gt;">`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				doc := NewDocument("1.0", "UTF-8", StandaloneExplicitNo)
				dtd, err := doc.CreateInternalSubset(attrDeclElem, "", "")
				require.NoError(t, err)
				_, err = dtd.AddElementDecl(attrDeclElem, enum.AnyElementType, nil)
				require.NoError(t, err)
				_, err = dtd.AddAttributeDecl(attrDeclElem, "label", enum.AttrCDATA, enum.AttrDefaultFixed, tc.defvalue, nil)
				require.NoError(t, err)

				root, err := doc.CreateElement(attrDeclElem)
				require.NoError(t, err)
				require.NoError(t, doc.SetDocumentElement(root))

				var buf bytes.Buffer
				require.NoError(t, Write(&buf, doc))
				require.Contains(t, buf.String(), tc.want)

				parsed, err := NewParser().Parse(t.Context(), buf.Bytes())
				require.NoError(t, err)
				d, ok := parsed.IntSubset().LookupAttribute("label", "", attrDeclElem)
				require.True(t, ok)
				require.Equal(t, tc.defvalue, d.defvalue, "recovered default value")

				var buf2 bytes.Buffer
				require.NoError(t, Write(&buf2, parsed))
				require.Contains(t, buf2.String(), tc.want, "re-serialized <!ATTLIST> is stable")
			})
		}
	})
}

// TestDTDSentinelErrors verifies the DTD/document error sites expose matchable
// sentinels.
func TestDTDSentinelErrors(t *testing.T) {
	doc := NewDocument("1.0", "UTF-8", StandaloneExplicitNo)

	// No internal subset.
	_, err := doc.InternalSubset()
	require.ErrorIs(t, err, ErrNoInternalSubset)

	dtd, err := doc.CreateInternalSubset("root", "", "")
	require.NoError(t, err)

	// Duplicate notation.
	_, err = dtd.AddNotation("n1", "", "sys")
	require.NoError(t, err)
	_, err = dtd.AddNotation("n1", "", "sys")
	require.ErrorIs(t, err, ErrDuplicateDeclaration)

	// Duplicate element.
	_, err = dtd.AddElementDecl("e1", enum.EmptyElementType, nil)
	require.NoError(t, err)
	_, err = dtd.AddElementDecl("e1", enum.EmptyElementType, nil)
	require.ErrorIs(t, err, ErrDuplicateDeclaration)
}

func TestAddElementDecl(t *testing.T) {
	// AddElementDecl must reject a structurally-incomplete content model (a
	// sequence/choice node with nil children, as CreateElementContent alone
	// produces) instead of storing it and letting serialization nil-dereference.
	t.Run("a malformed content model is rejected", func(t *testing.T) {
		doc := NewDocument("1.0", "UTF-8", StandaloneExplicitNo)
		dtd, err := doc.CreateInternalSubset("root", "", "")
		require.NoError(t, err)

		for _, etype := range []ElementContentType{ElementContentSeq, ElementContentOr} {
			content, err := doc.CreateElementContent("", etype)
			require.NoError(t, err)
			_, err = dtd.AddElementDecl("root", enum.ElementElementType, content)
			require.Error(t, err, "a seq/choice node with nil children must be rejected")

			// A rejected model must not have been stored, so serialization must not panic.
			var buf bytes.Buffer
			require.NotPanics(t, func() {
				_ = Write(&buf, doc)
			})
			require.NotContains(t, buf.String(), "<!ELEMENT root")
		}
	})
}

func TestCreateElementContent(t *testing.T) {
	// The safe composite constructors build a valid, serializable content model.
	t.Run("seq and choice compose", func(t *testing.T) {
		doc := NewDocument("1.0", "UTF-8", StandaloneExplicitNo)
		dtd, err := doc.CreateInternalSubset("root", "", "")
		require.NoError(t, err)

		a, err := doc.CreateElementContent("a", ElementContentElement)
		require.NoError(t, err)
		b, err := doc.CreateElementContent("b", ElementContentElement)
		require.NoError(t, err)
		c, err := doc.CreateElementContent("c", ElementContentElement)
		require.NoError(t, err)

		// (b | c)
		choice, err := doc.CreateElementContentChoice(b, c, ElementContentOnce)
		require.NoError(t, err)
		// (a , (b | c)+)
		_, err = choice.SetOccurrence(ElementContentPlus)
		require.NoError(t, err)
		seq, err := doc.CreateElementContentSeq(a, choice, ElementContentOnce)
		require.NoError(t, err)

		_, err = dtd.AddElementDecl("root", enum.ElementElementType, seq)
		require.NoError(t, err)

		var buf bytes.Buffer
		require.NoError(t, Write(&buf, doc))
		require.Contains(t, buf.String(), "<!ELEMENT root (a , (b | c)+)>")
	})

	// The composite constructors reject a nil or incomplete child.
	t.Run("nil or incomplete child rejected", func(t *testing.T) {
		doc := NewDocument("1.0", "UTF-8", StandaloneExplicitNo)
		a, err := doc.CreateElementContent("a", ElementContentElement)
		require.NoError(t, err)

		_, err = doc.CreateElementContentSeq(a, nil, ElementContentOnce)
		require.Error(t, err)

		_, err = doc.CreateElementContentChoice(nil, a, ElementContentOnce)
		require.Error(t, err)

		// An incomplete child (bare seq leaf with nil children) is also rejected.
		bad, err := doc.CreateElementContent("", ElementContentSeq)
		require.NoError(t, err)
		_, err = doc.CreateElementContentSeq(a, bad, ElementContentOnce)
		require.Error(t, err)
	})
}

// RemoveElement drops the declaration from the serialized DTD (not just the
// lookup table) and returns it.
func TestRemoveElement(t *testing.T) {
	doc := NewDocument("1.0", "UTF-8", StandaloneExplicitNo)
	dtd, err := doc.CreateInternalSubset("root", "", "")
	require.NoError(t, err)

	_, err = dtd.AddElementDecl("root", enum.EmptyElementType, nil)
	require.NoError(t, err)
	// A second decl keeps the elements table non-empty so the [...] block is
	// emitted; otherwise dumpDTD short-circuits and hides the leak.
	_, err = dtd.AddElementDecl("keep", enum.EmptyElementType, nil)
	require.NoError(t, err)

	removed := dtd.RemoveElement("root", "")
	require.NotNil(t, removed, "RemoveElement must return the removed declaration")
	require.Equal(t, "root", removed.name)

	_, ok := dtd.LookupElement("root", "")
	require.False(t, ok, "removed decl must be unmapped")

	var buf bytes.Buffer
	require.NoError(t, Write(&buf, doc))
	require.NotContains(t, buf.String(), "<!ELEMENT root", "removed decl must not be serialized")
	require.Contains(t, buf.String(), "<!ELEMENT keep EMPTY>")

	// Removing an absent key returns nil.
	require.Nil(t, dtd.RemoveElement("nope", ""))
}

// An element declaration registered via AddElementDecl can be retrieved through
// GetElementDesc using the same DTD Name. Registration splits the raw spelling
// for lookup; GetElementDesc must compose the same key instead of treating the
// DTD Name as a namespace-aware QName.
func TestGetElementDesc(t *testing.T) {
	t.Run("unprefixed", func(t *testing.T) {
		dtd := newDTD()
		content, err := newElementContent("", ElementContentPCDATA)
		require.NoError(t, err)
		_, err = dtd.AddElementDecl("r", enum.MixedElementType, content)
		require.NoError(t, err)

		decl, ok := dtd.GetElementDesc("r")
		require.True(t, ok, "GetElementDesc must find the registered decl")
		require.Equal(t, enum.MixedElementType, decl.decltype)
	})
	t.Run("prefixed", func(t *testing.T) {
		dtd := newDTD()
		_, err := dtd.AddElementDecl("foo:bar", enum.EmptyElementType, nil)
		require.NoError(t, err)

		decl, ok := dtd.GetElementDesc("foo:bar")
		require.True(t, ok, "GetElementDesc must find the prefixed decl by QName")
		require.Equal(t, enum.EmptyElementType, decl.decltype)
	})
	t.Run("leading colon distinct from unprefixed", func(t *testing.T) {
		// A leading colon is NOT a prefix separator (libxml2 xmlSplitQName3): ":r"
		// is a distinct element name from the unprefixed "r" and must not be
		// reported as a redefinition of it (XML 1.0 5th-edition Name; eduni
		// ibm04v01).
		dtd := newDTD()
		_, err := dtd.AddElementDecl("r", enum.EmptyElementType, nil)
		require.NoError(t, err)
		_, err = dtd.AddElementDecl(":r", enum.AnyElementType, nil)
		require.NoError(t, err, "leading-colon name must not collide with the unprefixed name")

		decl, ok := dtd.GetElementDesc(":r")
		require.True(t, ok, "GetElementDesc must find the leading-colon decl")
		require.Equal(t, enum.AnyElementType, decl.decltype)

		decl, ok = dtd.GetElementDesc("r")
		require.True(t, ok, "GetElementDesc must still find the unprefixed decl")
		require.Equal(t, enum.EmptyElementType, decl.decltype)
	})
}

// Document.IsMixedElement reports an element declared with mixed (#PCDATA)
// content in the internal subset as mixed — the declared-content-model property
// the whitespace-significance classification depends on. The test does not parse
// whitespace itself; the end-to-end whitespace path (which consults
// elementDeclType across both subsets) is covered by TestStripBlanks.
func TestIsMixedElement(t *testing.T) {
	doc := NewDocument("1.0", "UTF-8", StandaloneExplicitNo)
	dtd := newDTD()
	dtd.doc = doc
	doc.intSubset = dtd

	content, err := newElementContent("", ElementContentPCDATA)
	require.NoError(t, err)
	_, err = dtd.AddElementDecl("r", enum.MixedElementType, content)
	require.NoError(t, err)

	mixed, err := doc.IsMixedElement("r")
	require.NoError(t, err)
	require.True(t, mixed, "mixed-content element must be reported as mixed")
}

// AttributesForElement is served from the attrsByElem index, keyed by owning
// element name in registration order, rather than a full scan of dtd.attributes
// (a Go map, whose iteration order is randomized per run). This pins down
// declaration order across both AddAttributeDecl and the parser, across a
// deep-copied DTD, and the "no declarations" case.
func TestAttributesForElementIndex(t *testing.T) {
	t.Run("registration order via AddAttributeDecl", func(t *testing.T) {
		doc := NewDocument("1.0", "UTF-8", StandaloneExplicitNo)
		dtd, err := doc.CreateInternalSubset("root", "", "")
		require.NoError(t, err)

		_, err = dtd.AddAttributeDecl("one", "a", enum.AttrCDATA, enum.AttrDefaultImplied, "", nil)
		require.NoError(t, err)
		_, err = dtd.AddAttributeDecl("two", "x", enum.AttrCDATA, enum.AttrDefaultImplied, "", nil)
		require.NoError(t, err)
		_, err = dtd.AddAttributeDecl("one", "b", enum.AttrCDATA, enum.AttrDefaultImplied, "", nil)
		require.NoError(t, err)
		_, err = dtd.AddAttributeDecl("two", "y", enum.AttrCDATA, enum.AttrDefaultImplied, "", nil)
		require.NoError(t, err)
		_, err = dtd.AddAttributeDecl("one", "c", enum.AttrCDATA, enum.AttrDefaultImplied, "", nil)
		require.NoError(t, err)

		one := dtd.AttributesForElement("one")
		require.Len(t, one, 3)
		require.Equal(t, []string{"a", "b", "c"}, []string{one[0].name, one[1].name, one[2].name})

		two := dtd.AttributesForElement("two")
		require.Len(t, two, 2)
		require.Equal(t, []string{"x", "y"}, []string{two[0].name, two[1].name})
	})

	t.Run("registration order via parsed ATTLIST", func(t *testing.T) {
		const src = `<?xml version="1.0"?>
<!DOCTYPE one [
<!ELEMENT one (#PCDATA)>
<!ATTLIST one a CDATA #IMPLIED>
<!ATTLIST one b CDATA #IMPLIED>
<!ATTLIST one c CDATA #IMPLIED>
]>
<one/>`
		doc, err := NewParser().Parse(t.Context(), []byte(src))
		require.NoError(t, err)
		dtd := doc.IntSubset()
		require.NotNil(t, dtd)

		got := dtd.AttributesForElement("one")
		require.Len(t, got, 3)
		require.Equal(t, []string{"a", "b", "c"}, []string{got[0].name, got[1].name, got[2].name})
	})

	t.Run("unknown element returns nil", func(t *testing.T) {
		doc := NewDocument("1.0", "UTF-8", StandaloneExplicitNo)
		dtd, err := doc.CreateInternalSubset("root", "", "")
		require.NoError(t, err)
		_, err = dtd.AddAttributeDecl("root", "a", enum.AttrCDATA, enum.AttrDefaultImplied, "", nil)
		require.NoError(t, err)

		require.Nil(t, dtd.AttributesForElement("nonexistent"))
	})

	t.Run("deep-copied document preserves order", func(t *testing.T) {
		doc := NewDocument("1.0", "UTF-8", StandaloneExplicitNo)
		dtd, err := doc.CreateInternalSubset("one", "", "")
		require.NoError(t, err)
		_, err = dtd.AddAttributeDecl("one", "a", enum.AttrCDATA, enum.AttrDefaultImplied, "", nil)
		require.NoError(t, err)
		_, err = dtd.AddAttributeDecl("one", "b", enum.AttrCDATA, enum.AttrDefaultImplied, "", nil)
		require.NoError(t, err)
		_, err = dtd.AddAttributeDecl("one", "c", enum.AttrCDATA, enum.AttrDefaultImplied, "", nil)
		require.NoError(t, err)

		cp, err := CopyDoc(doc)
		require.NoError(t, err)
		cpDTD := cp.IntSubset()
		require.NotNil(t, cpDTD)

		got := cpDTD.AttributesForElement("one")
		require.Len(t, got, 3)
		require.Equal(t, []string{"a", "b", "c"}, []string{got[0].name, got[1].name, got[2].name})
	})
}
