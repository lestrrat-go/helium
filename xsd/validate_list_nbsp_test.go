package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// nbsp is U+00A0 NON-BREAKING SPACE, which is Unicode whitespace but NOT one of
// the four XSD whitespace characters (#x20, #x9, #xD, #xA). xs:list item
// separation is defined over XSD whitespace only, so an item containing NBSP must
// stay a single token (and then fail per-item lexical validation) rather than
// being split into separate valid items.
const nbsp = " "

// TestListItemNBSPNotSeparator covers schema-defined xs:list validation: a value
// like "1<NBSP>2" for xs:list itemType="xs:int" must NOT validate, because NBSP
// is not an XSD list separator. Splitting on NBSP (as strings.Fields does) would
// wrongly turn it into two valid xs:int items "1" and "2". A genuine XSD-space
// separated "1 2" stays valid.
func TestListItemNBSPNotSeparator(t *testing.T) {
	t.Parallel()

	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="intList">
    <xs:list itemType="xs:int"/>
  </xs:simpleType>
  <xs:element name="root" type="intList"/>
</xs:schema>`

	cases := []struct {
		name     string
		instance string
		valid    bool
	}{
		{
			name:     "xsd-space separated ints valid",
			instance: `<root>1 2</root>`,
			valid:    true,
		},
		{
			name:     "nbsp-joined ints invalid",
			instance: `<root>1` + nbsp + `2</root>`,
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
			require.Error(t, err, "expected validation error")
		})
	}
}

// TestTokenNBSPNotWhitespace covers schema-defined xs:token validation: NBSP
// (U+00A0) is NOT XSD whitespace, so a value " abc " whose only padding is NBSP
// has no leading/trailing XSD whitespace and no double ASCII space — it is a
// valid xs:token. Trimming with Go's Unicode strings.TrimSpace (which treats
// NBSP as space) would wrongly reject it.
func TestTokenNBSPNotWhitespace(t *testing.T) {
	t.Parallel()

	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="xs:token"/>
</xs:schema>`

	cases := []struct {
		name     string
		instance string
		valid    bool
	}{
		{
			// NBSP-only padding survives xs:token's collapse facet (NBSP is not XSD
			// whitespace), leaving a token "<NBSP>abc<NBSP>" with no leading/trailing
			// XSD whitespace and no double ASCII space, so it is a valid xs:token.
			name:     "nbsp-padded token valid",
			instance: `<root>` + nbsp + `abc` + nbsp + `</root>`,
			valid:    true,
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
			require.Error(t, err, "expected validation error")
		})
	}
}

// TestIDCListItemNBSPNotSeparator covers the IDC list canonical-key path: list
// item separation there must also use XSD whitespace only, consistent with list
// validation. The XSD-space forms "5 6" and "+5 06" collide item-by-item in
// xs:integer value space. An NBSP-joined "5<NBSP>6" is a single token that is not
// a valid xs:integer, so it is rejected during value validation (which runs
// before IDC evaluation) rather than being split into two valid integers; the
// instance is therefore invalid, proving NBSP is not treated as a separator on
// the IDC list path either.
func TestIDCListItemNBSPNotSeparator(t *testing.T) {
	t.Parallel()

	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="intList">
    <xs:list itemType="xs:integer"/>
  </xs:simpleType>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" type="intList" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
    <xs:unique name="itemKey">
      <xs:selector xpath="item"/>
      <xs:field xpath="."/>
    </xs:unique>
  </xs:element>
</xs:schema>`

	cases := []struct {
		name     string
		instance string
		valid    bool
	}{
		{
			// XSD-space lists collide item-by-item in integer value space.
			name:     "xsd-space integer lists 5 6 and +5 06 collide",
			instance: `<root><item>5 6</item><item>+5 06</item></root>`,
			valid:    false,
		},
		{
			// An NBSP-joined "5<NBSP>6" is a single token, not two integers, so it
			// fails xs:integer item validation and the instance is invalid — NBSP is
			// not split as a list separator on this path.
			name:     "nbsp-joined list item invalid",
			instance: `<root><item>5` + nbsp + `6</item></root>`,
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
			require.Error(t, err, "expected validation error")
		})
	}
}
