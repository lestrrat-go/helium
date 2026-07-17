package helium

import (
	"bytes"
	"testing"

	"github.com/lestrrat-go/helium/enum"
	"github.com/stretchr/testify/require"
)

// TestAddAttributeDeclSerializes verifies AddAttributeDecl builds a declaration
// from public parameters, links it into the DTD child list, and serializes it as
// an <!ATTLIST> declaration.
func TestAddAttributeDeclSerializes(t *testing.T) {
	doc := NewDocument("1.0", "UTF-8", StandaloneExplicitNo)
	dtd, err := doc.CreateInternalSubset("root", "", "")
	require.NoError(t, err)

	adecl, err := dtd.AddAttributeDecl("item", "count", enum.AttrCDATA, enum.AttrDefaultRequired, "", nil)
	require.NoError(t, err)
	require.Equal(t, "item", adecl.Elem())
	require.Equal(t, enum.AttrCDATA, adecl.AType())

	// A FIXED default value round-trips.
	_, err = dtd.AddAttributeDecl("item", "unit", enum.AttrCDATA, enum.AttrDefaultFixed, "px", nil)
	require.NoError(t, err)

	// An enumeration type emits its token list.
	_, err = dtd.AddAttributeDecl("item", "kind", enum.AttrEnumeration, enum.AttrDefaultImplied, "", Enumeration{"a", "b"})
	require.NoError(t, err)

	// A NOTATION type emits NOTATION (...).
	_, err = dtd.AddAttributeDecl("item", "note", enum.AttrNotation, enum.AttrDefaultImplied, "", Enumeration{"gif"})
	require.NoError(t, err)

	// The decl is retrievable through the public lookup.
	got, ok := dtd.LookupAttribute("count", "", "item")
	require.True(t, ok)
	require.Equal(t, adecl, got)

	var buf bytes.Buffer
	require.NoError(t, Write(&buf, doc))
	out := buf.String()
	require.Contains(t, out, "<!ATTLIST item count CDATA #REQUIRED>")
	require.Contains(t, out, `<!ATTLIST item unit CDATA #FIXED "px">`)
	require.Contains(t, out, "<!ATTLIST item kind (a | b) #IMPLIED>")
	require.Contains(t, out, "<!ATTLIST item note NOTATION (gif) #IMPLIED>")
}

// TestAddAttributeDeclQNameSplit verifies a prefixed attribute name is split into
// prefix + local (mirroring AddElementDecl) and serialized as "prefix:local".
func TestAddAttributeDeclQNameSplit(t *testing.T) {
	doc := NewDocument("1.0", "UTF-8", StandaloneExplicitNo)
	dtd, err := doc.CreateInternalSubset("root", "", "")
	require.NoError(t, err)

	_, err = dtd.AddAttributeDecl("item", "x:id", enum.AttrID, enum.AttrDefaultRequired, "", nil)
	require.NoError(t, err)

	// Keyed under local + prefix.
	_, ok := dtd.LookupAttribute("id", "x", "item")
	require.True(t, ok)

	var buf bytes.Buffer
	require.NoError(t, Write(&buf, doc))
	require.Contains(t, buf.String(), "<!ATTLIST item x:id ID #REQUIRED>")
}

// TestAddAttributeDeclDuplicate verifies a repeat declaration is rejected with
// ErrDuplicateDeclaration.
func TestAddAttributeDeclDuplicate(t *testing.T) {
	doc := NewDocument("1.0", "UTF-8", StandaloneExplicitNo)
	dtd, err := doc.CreateInternalSubset("root", "", "")
	require.NoError(t, err)

	_, err = dtd.AddAttributeDecl("item", "count", enum.AttrCDATA, enum.AttrDefaultImplied, "", nil)
	require.NoError(t, err)

	_, err = dtd.AddAttributeDecl("item", "count", enum.AttrCDATA, enum.AttrDefaultRequired, "", nil)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrDuplicateDeclaration)
}

// TestAddAttributeDeclValidation verifies parameter validation: bad type/default,
// a missing enumeration token list, and an invalid default value.
func TestAddAttributeDeclValidation(t *testing.T) {
	doc := NewDocument("1.0", "UTF-8", StandaloneExplicitNo)
	dtd, err := doc.CreateInternalSubset("root", "", "")
	require.NoError(t, err)

	_, err = dtd.AddAttributeDecl("", "count", enum.AttrCDATA, enum.AttrDefaultNone, "", nil)
	require.Error(t, err, "empty element name must be rejected")

	_, err = dtd.AddAttributeDecl("item", "", enum.AttrCDATA, enum.AttrDefaultNone, "", nil)
	require.Error(t, err, "empty attribute name must be rejected")

	_, err = dtd.AddAttributeDecl("item", "count", enum.AttributeType(999), enum.AttrDefaultNone, "", nil)
	require.Error(t, err, "invalid attribute type must be rejected")

	_, err = dtd.AddAttributeDecl("item", "count", enum.AttrCDATA, enum.AttributeDefault(999), "", nil)
	require.Error(t, err, "invalid default declaration must be rejected")

	_, err = dtd.AddAttributeDecl("item", "kind", enum.AttrEnumeration, enum.AttrDefaultImplied, "", nil)
	require.Error(t, err, "enumeration type without tokens must be rejected")

	_, err = dtd.AddAttributeDecl("item", "id", enum.AttrID, enum.AttrDefaultNone, "not a name", nil)
	require.Error(t, err, "an invalid ID default value must be rejected")
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
