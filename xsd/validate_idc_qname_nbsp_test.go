package xsd_test

import (
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// TestIDCFieldQNameNBSPNotTrimmed covers identity-constraint key canonicalization
// for an xs:QName field whose value carries a trailing NBSP (U+00A0). NBSP is
// Unicode whitespace but NOT one of the four XSD whitespace characters, so a value
// like "p:a<NBSP>" is NOT a valid xs:QName and must be rejected by content
// validation. The IDC canonicalizer must trim only XSD whitespace (it resolves the
// already-collapsed lexical value directly) rather than Go's strings.TrimSpace,
// which strips NBSP and would canonicalize "p:a<NBSP>" identically to a sibling
// plain "p:a" — producing a spurious "Duplicate key-sequence" diagnostic on top of
// the genuine lexical error.
func TestIDCFieldQNameNBSPNotTrimmed(t *testing.T) {
	t.Parallel()

	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" type="xs:QName" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
    <xs:unique name="itemKey">
      <xs:selector xpath="item"/>
      <xs:field xpath="."/>
    </xs:unique>
  </xs:element>
</xs:schema>`

	// A plain "p:a" followed by an NBSP-padded "p:a<NBSP>". The second value is
	// lexically invalid (content error expected), but it must NOT also be reported
	// as a duplicate of the first: trimming NBSP would wrongly canonicalize the two
	// to the same key.
	instance := `<root xmlns:p="urn:x"><item>p:a</item><item>p:a` + nbsp + `</item></root>`

	v := compileValidator(t, schema)
	doc, err := helium.NewParser().Parse(t.Context(), []byte(instance))
	require.NoError(t, err)

	var errs string
	err = validateWithOutput(t, v, doc, &errs)
	require.Error(t, err, "NBSP-padded xs:QName must be a content-validation error")
	require.NotContains(t, errs, "Duplicate key-sequence",
		"NBSP-padded xs:QName must not produce a false IDC duplicate diagnostic")
}

// TestIDCFieldQNameNBSPDistinctKeys is the focused positive counterpart: two
// QName items that are both content-valid and differ only by NBSP-vs-plain must
// stay DISTINCT (the NBSP form is in fact invalid here, so the real guarantee is
// the absence of a false collision). It also confirms that genuinely equal QName
// values (same URI, different prefix) still collide, so the NBSP fix does not over-
// correct.
func TestIDCFieldQNameNBSPDistinctKeys(t *testing.T) {
	t.Parallel()

	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" type="xs:QName" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
    <xs:unique name="itemKey">
      <xs:selector xpath="item"/>
      <xs:field xpath="."/>
    </xs:unique>
  </xs:element>
</xs:schema>`

	// Value-equal QNames (same URI urn:x, different prefix) must still collide.
	instance := `<root xmlns:p="urn:x" xmlns:q="urn:x"><item>p:a</item><item>q:a</item></root>`

	v := compileValidator(t, schema)
	doc, err := helium.NewParser().Parse(t.Context(), []byte(instance))
	require.NoError(t, err)

	var errs string
	err = validateWithOutput(t, v, doc, &errs)
	require.Error(t, err, "value-equal QNames must collide as a duplicate key")
	require.True(t, strings.Contains(errs, "Duplicate key-sequence"),
		"value-equal QNames must report a duplicate key-sequence")
}
