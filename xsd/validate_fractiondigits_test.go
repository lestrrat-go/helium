package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestFractionDigitsTrailingZeros checks that the fractionDigits facet counts
// significant fraction digits of the value, not lexical characters. Trailing
// zeros are not significant: "2.00" has the same value as "2", so it satisfies
// fractionDigits="1".
func TestFractionDigitsTrailingZeros(t *testing.T) {
	t.Parallel()

	schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="oneFracDigit"/>
  <xs:simpleType name="oneFracDigit">
    <xs:restriction base="xs:decimal">
      <xs:fractionDigits value="1"/>
    </xs:restriction>
  </xs:simpleType>
</xs:schema>`

	schemaDOC, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	schema, err := xsd.NewCompiler().Compile(t.Context(), schemaDOC)
	require.NoError(t, err)

	cases := []struct {
		value string
		valid bool
	}{
		{"2.0", true},
		{"2.00", true},  // trailing zero — value has 0 fraction digits
		{"2.000", true}, // more trailing zeros
		{"2", true},
		{"2.5", true},
		{"-3.00", true},
		{"2.50", true},  // one significant fraction digit, then a trailing zero
		{"2.05", false}, // two significant fraction digits
		{"2.55", false}, // two significant fraction digits
	}

	for _, tc := range cases {
		t.Run(tc.value, func(t *testing.T) {
			t.Parallel()
			doc, err := helium.NewParser().Parse(t.Context(), []byte("<root>"+tc.value+"</root>"))
			require.NoError(t, err)

			var errs string
			err = validateWithOutput(t, xsd.NewValidator(schema), doc, &errs)
			if tc.valid {
				require.NoError(t, err, "expected valid, got: %s", errs)
			} else {
				require.Error(t, err)
			}
		})
	}
}
