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

// TestAddAttributeDeclSerializes verifies AddAttributeDecl builds a declaration
// from public parameters, links it into the DTD child list, serializes it as an
// <!ATTLIST> declaration, and — the acceptance bar — that a validating parser
// accepts the serialized document and recovers each declaration equivalently.
func TestAddAttributeDeclSerializes(t *testing.T) {
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
}

// TestAddAttributeDeclQNameSplit verifies a prefixed attribute name is split into
// prefix + local (mirroring AddElementDecl) and serialized as "prefix:local".
func TestAddAttributeDeclQNameSplit(t *testing.T) {
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
}

// TestAddAttributeDeclClonesEnumValues verifies the token list is cloned, so a
// caller mutating the slice after the call cannot corrupt the serialized decl.
func TestAddAttributeDeclClonesEnumValues(t *testing.T) {
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
}

// TestAddAttributeDeclDuplicate verifies a repeat declaration is rejected with
// ErrDuplicateDeclaration.
func TestAddAttributeDeclDuplicate(t *testing.T) {
	doc := NewDocument("1.0", "UTF-8", StandaloneExplicitNo)
	dtd, err := doc.CreateInternalSubset("root", "", "")
	require.NoError(t, err)

	_, err = dtd.AddAttributeDecl(attrDeclElem, attrDeclCount, enum.AttrCDATA, enum.AttrDefaultImplied, "", nil)
	require.NoError(t, err)

	_, err = dtd.AddAttributeDecl(attrDeclElem, attrDeclCount, enum.AttrCDATA, enum.AttrDefaultRequired, "", nil)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrDuplicateDeclaration)
}

// TestAddAttributeDeclRejects verifies an out-of-range enum parameter is rejected
// (wrapping ErrInvalidArgument) before registration, so nothing is registered or
// serialized. Like its sibling constructors, AddAttributeDecl validates only the
// enum parameters — it trusts the caller for well-formed names and values.
func TestAddAttributeDeclRejects(t *testing.T) {
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
}

// TestAddAttributeDeclNoKeyCollision verifies two distinct declarations whose
// (local, prefix, elem) triples concatenate to the SAME `local + ":" + prefix +
// ":" + elem` string are both registered and independently retrievable. The old
// string key aliased them and wrongly rejected the second as a duplicate; the
// struct key keeps them distinct.
//
// Decl A: name "b:a" -> local "a", prefix "b"; elem "c:d". String key "a:b:c:d".
// Decl B: name "c:a:b" -> local "a:b", prefix "c"; elem "d". String key "a:b:c:d".
func TestAddAttributeDeclNoKeyCollision(t *testing.T) {
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
}

// TestAddAttributeDeclEnumerationTokenColonAccepted verifies a well-formed
// enumeration whose tokens carry a colon is accepted and round-trips through a
// validating parser.
func TestAddAttributeDeclEnumerationTokenColonAccepted(t *testing.T) {
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
}

// TestAddAttributeDeclDefaultRoundTripEquivalence verifies an accepted
// value-bearing default is stable across serialize→parse→serialize: the default
// value the parser recovers is identical, and re-serializing the reparsed document
// yields the same <!ATTLIST> line. It exercises defaults that round-trip through
// the default-value serializer — a '<' (escaped as "&lt;" and decoded back), a '"'
// (escaped as "&quot;"), a CDATA-end sequence, and a plain value.
func TestAddAttributeDeclDefaultRoundTripEquivalence(t *testing.T) {
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
