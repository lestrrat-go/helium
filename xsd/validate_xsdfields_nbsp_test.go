package xsd_test

import (
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// TestUnionFixedListItemNBSPNotSeparator covers the list-vs-list union/fixed
// comparison path (crossMemberValueEqualDepth). The fixed value is the
// XSD-space-separated two-item list "a b"; the instance is the single token
// "a<NBSP>b". NBSP is Unicode whitespace but NOT one of the four XSD whitespace
// characters, so the instance is a one-item list and must NOT compare equal to the
// two-item fixed list. Splitting on NBSP (as strings.Fields would) would wrongly
// produce two items "a"/"b" that match the fixed list. The genuine XSD-space form
// "a b" still matches.
func TestUnionFixedListItemNBSPNotSeparator(t *testing.T) {
	t.Parallel()

	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="tokenList">
    <xs:list itemType="xs:token"/>
  </xs:simpleType>
  <xs:simpleType name="valueUnion">
    <xs:union memberTypes="tokenList xs:token"/>
  </xs:simpleType>
  <xs:element name="root" type="valueUnion" fixed="a b"/>
</xs:schema>`

	cases := []struct {
		name     string
		instance string
		valid    bool
	}{
		{
			// XSD-space separated form matches the fixed two-item list value.
			name:     "xsd-space form matches fixed",
			instance: `<root>a b</root>`,
			valid:    true,
		},
		{
			// "a<NBSP>b" is a single list item; it cannot equal the two-item fixed
			// list "a b", so the fixed-value constraint is violated.
			name:     "nbsp-joined item does not match fixed",
			instance: `<root>a` + nbsp + `b</root>`,
			valid:    false,
		},
	}

	v := compileValidator(t, schema)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			doc, err := helium.NewParser().Parse(t.Context(), []byte(tc.instance))
			require.NoError(t, err)

			var errs string
			err = validateWithOutput(t, v, doc, &errs)
			if tc.valid {
				require.NoError(t, err, "expected valid, got errors: %s", errs)
				return
			}
			require.Error(t, err, "NBSP-joined single item must not satisfy the fixed two-item list")
		})
	}
}

// TestListQNameEnumNBSPSingleItem covers the compile-time enumeration-literal
// prefix-binding check for an xs:list itemType="xs:QName" whose enumeration value
// is "p:a<NBSP>q:b" with both p and q bound. NBSP is not an XSD list separator, so
// the literal is a SINGLE list item that is not a valid xs:QName (it contains an
// NBSP and a second colon) and the schema must be rejected at compile time.
// Splitting on NBSP (as strings.FieldsSeq would) would wrongly yield two valid,
// bound QNames "p:a"/"q:b" and accept the schema.
func TestListQNameEnumNBSPSingleItem(t *testing.T) {
	t.Parallel()

	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
            xmlns:p="urn:p" xmlns:q="urn:q">
  <xs:simpleType name="qnameList">
    <xs:restriction>
      <xs:simpleType>
        <xs:list itemType="xs:QName"/>
      </xs:simpleType>
      <xs:enumeration value="p:a` + nbsp + `q:b"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="root" type="qnameList"/>
</xs:schema>`

	errs := compileSchemaErrors(t, schema)
	require.NotEmpty(t, errs,
		"NBSP-joined enumeration literal is a single invalid xs:QName item and must be rejected")
	require.True(t, strings.Contains(errs, "not a valid value"),
		"compile error must report the enumeration literal is not a valid value: %s", errs)
}
