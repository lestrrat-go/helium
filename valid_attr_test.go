package helium_test

import (
	"testing"
	"testing/fstest"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// TestAttributeValidityCompleteness exercises the per-instance attribute VCs
// enforced in validateElementAttributes: Attribute Value Type (every present
// attribute must be declared, for ordinary attributes and for xmlns/xmlns:*
// namespace declarations) and Fixed Attribute Default (a #FIXED namespace
// declaration must match). Each VC has a rejecting case and a valid near-miss so
// the check does not over-reject.
func TestAttributeValidityCompleteness(t *testing.T) {
	t.Parallel()

	// VC: Attribute Value Type — an ordinary undeclared attribute is invalid
	// (W3C ibm-invalid-P41-ibm41i01.xml).
	t.Run("undeclared ordinary attribute", func(t *testing.T) {
		t.Parallel()

		t.Run("rejected", func(t *testing.T) {
			t.Parallel()
			errs, err := parseValidatingDTD(t, `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root (#PCDATA|b)*>
  <!ELEMENT b (#PCDATA)>
  <!ATTLIST b attr2 (abc|def) "abc">
]>
<root><b attr1="value1" attr2="def">x</b></root>`)
			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
			require.True(t, containsError(errs, "no declaration for attribute attr1"))
		})

		t.Run("declared attribute accepted", func(t *testing.T) {
			t.Parallel()
			_, err := parseValidatingDTD(t, `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root (#PCDATA|b)*>
  <!ELEMENT b (#PCDATA)>
  <!ATTLIST b attr1 CDATA #IMPLIED attr2 (abc|def) "abc">
]>
<root><b attr1="value1" attr2="def">x</b></root>`)
			require.NoError(t, err)
		})
	})

	// VC: Attribute Value Type applied to attributes in the reserved xml
	// namespace (W3C inv-required01/inv-required02). xml:space / xml:lang are
	// ordinary attributes that still require declaration.
	t.Run("undeclared reserved xml attribute", func(t *testing.T) {
		t.Parallel()

		t.Run("xml:space rejected", func(t *testing.T) {
			t.Parallel()
			errs, err := parseValidatingDTD(t, `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root EMPTY>
]>
<root xml:space='preserve'/>`)
			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
			require.True(t, containsError(errs, "no declaration for attribute xml:space"))
		})

		t.Run("declared xml:space accepted", func(t *testing.T) {
			t.Parallel()
			_, err := parseValidatingDTD(t, `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root EMPTY>
  <!ATTLIST root xml:space (default|preserve) #IMPLIED>
]>
<root xml:space='preserve'/>`)
			require.NoError(t, err)
		})
	})

	// A namespace declaration (xmlns:*) is EXEMPT from the "must be declared" VC:
	// a namespaced document may DTD-validate against a namespace-agnostic DTD that
	// never declares its xmlns attributes. This is the over-rejection guard for
	// the namespace-declaration path (helium diverges from a namespace-UNAWARE
	// validator here, so W3C hst-bh-005/hst-bh-006 stay out of scope).
	t.Run("undeclared namespace declaration accepted", func(t *testing.T) {
		t.Parallel()
		_, err := parseValidatingDTD(t, `<?xml version="1.0"?>
<!DOCTYPE r [
  <!ELEMENT r (p:a)>
  <!ELEMENT p:a EMPTY>
]>
<r xmlns:p='http://example.org'><p:a/></r>`)
		require.NoError(t, err)
	})

	// VC: Fixed Attribute Default applied to a default (xmlns) namespace
	// declaration (W3C attr08). A #FIXED xmlns must match the declared value.
	t.Run("fixed namespace declaration", func(t *testing.T) {
		t.Parallel()

		t.Run("differing value rejected", func(t *testing.T) {
			t.Parallel()
			errs, err := parseValidatingDTD(t, `<?xml version="1.0"?>
<!DOCTYPE palimpest [
  <!ELEMENT palimpest EMPTY>
  <!ATTLIST palimpest xmlns CDATA #FIXED "http://java.sun.com/historical">
]>
<palimpest xmlns="http://over.the.rainbow.com/somewhere"/>`)
			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
			require.True(t, containsError(errs, "attribute xmlns has value"))
		})

		t.Run("matching value accepted", func(t *testing.T) {
			t.Parallel()
			_, err := parseValidatingDTD(t, `<?xml version="1.0"?>
<!DOCTYPE palimpest [
  <!ELEMENT palimpest EMPTY>
  <!ATTLIST palimpest xmlns CDATA #FIXED "http://java.sun.com/historical">
]>
<palimpest xmlns="http://java.sun.com/historical"/>`)
			require.NoError(t, err)
		})
	})
}

// TestExternalEntitySyntheticBaseNotFlagged is the over-rejection guard for the
// "must be declared" VC: helium injects a synthetic xml:base attribute onto the
// top-level elements of an external parsed entity to record the entity's base
// URI. That attribute is not in the source and must not trip the Attribute Value
// Type VC (W3C valid ext-sa-005/013, sun/valid/ext01). libxml2 tracks the entity
// base without materializing an attribute, so it never flags one.
func TestExternalEntitySyntheticBaseNotFlagged(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"e.ent": {Data: []byte(`<e/><e/>`)},
	}
	src := `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (e*)>
  <!ELEMENT e EMPTY>
  <!ENTITY e SYSTEM "e.ent">
]>
<doc>&e;</doc>`

	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	_, err := helium.NewParser().
		BaseURI("doc.xml").
		BlockXXE(false).
		LoadExternalDTD(true).
		DefaultDTDAttributes(true).
		SubstituteEntities(true).
		ValidateDTD(true).
		FS(fsys).
		ErrorHandler(collector).
		Parse(t.Context(), []byte(src))
	require.NoError(t, err, "external-entity elements carry a synthetic xml:base that must not be flagged as undeclared; errors=%v", collector.Errors())
}

// TestExternalEntityAuthoredBaseFlagged is the adversarial counterpart: only the
// parser-injected xml:base is exempt. An AUTHORED xml:base on an external-entity
// element — even one whose value coincidentally equals the entity's base URI, so
// the parser suppresses its own injection — is a real attribute and must be
// rejected when undeclared (VC: Attribute Value Type), matching
// `xmllint --valid --noent`. The exemption is marker-based, not value-based.
func TestExternalEntityAuthoredBaseFlagged(t *testing.T) {
	t.Parallel()

	// The entity's base URI resolves to "e.ent" (relative to BaseURI "doc.xml"),
	// and the authored xml:base value is exactly that, so a value-equality
	// exemption would wrongly accept it.
	fsys := fstest.MapFS{
		"e.ent": {Data: []byte(`<e xml:base="e.ent"/>`)},
	}
	src := `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (e*)>
  <!ELEMENT e EMPTY>
  <!ENTITY e SYSTEM "e.ent">
]>
<doc>&e;</doc>`

	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	_, err := helium.NewParser().
		BaseURI("doc.xml").
		BlockXXE(false).
		LoadExternalDTD(true).
		DefaultDTDAttributes(true).
		SubstituteEntities(true).
		ValidateDTD(true).
		FS(fsys).
		ErrorHandler(collector).
		Parse(t.Context(), []byte(src))
	require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
	require.True(t, containsError(collector.Errors(), "no declaration for attribute xml:base"),
		"authored undeclared xml:base must be flagged; errors=%v", collector.Errors())
}
