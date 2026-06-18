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
// item separation there must use XSD whitespace only, consistent with list
// validation. The itemType is xs:string so EVERY value here is content-valid — the
// test therefore exercises only how the IDC canonical key tokenizes the list, not
// content validation (which would otherwise reject an NBSP value earlier and mask
// the IDC behavior).
//
// Two <item> values are compared by their <xs:unique> keys:
//   - "a b"      → two XSD-space items → canonical key "a b"
//   - "a<NBSP>b" → ONE token (NBSP is not an XSD separator) → canonical key
//     "a<NBSP>b", which is DISTINCT from "a b".
//
// So no duplicate-key error must be emitted: the NBSP value is a single, distinct
// key. If the IDC canonicalization were reverted to strings.Fields, the NBSP value
// would split into ["a","b"] → key "a b", collide with the first item, and the
// instance would wrongly fail with a duplicate-key-sequence error. This test thus
// genuinely discriminates the XSD-whitespace fix.
func TestIDCListItemNBSPNotSeparator(t *testing.T) {
	t.Parallel()

	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="strList">
    <xs:list itemType="xs:string"/>
  </xs:simpleType>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" type="strList" maxOccurs="unbounded"/>
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
			// Two genuinely distinct two-item XSD-space lists — no duplicate, valid.
			name:     "distinct xsd-space lists valid",
			instance: `<root><item>a b</item><item>c d</item></root>`,
			valid:    true,
		},
		{
			// Two-item "a b" vs one-token "a<NBSP>b": NBSP is not a separator, so the
			// canonical keys "a b" and "a<NBSP>b" differ — NO duplicate-key error.
			// Under strings.Fields they would both canonicalize to "a b" and collide.
			name:     "nbsp item distinct from xsd-space item",
			instance: `<root><item>a b</item><item>a` + nbsp + `b</item></root>`,
			valid:    true,
		},
		{
			// Sanity: two identical "a b" two-item lists DO collide, proving the
			// duplicate-key machinery is actually engaged on this path.
			name:     "identical xsd-space lists collide",
			instance: `<root><item>a b</item><item>a b</item></root>`,
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
