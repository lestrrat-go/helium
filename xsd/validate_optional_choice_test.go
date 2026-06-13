package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestOptionalElementBeforeChoice covers a content model where an optional
// element precedes a choice of optional branches:
//
//	sequence{ Lead?, choice{ P?, Q? } }
//
// When the optional Lead element is omitted and the instance selects a choice
// branch other than the first, the lookahead must still consume the chosen
// branch instead of stopping at the first zero-length-matching branch.
func TestOptionalElementBeforeChoice(t *testing.T) {
	t.Parallel()

	schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="Lead" type="xs:string" minOccurs="0"/>
        <xs:choice>
          <xs:element name="P" type="xs:string" minOccurs="0"/>
          <xs:element name="Q" type="xs:string" minOccurs="0"/>
        </xs:choice>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	schemaDOC, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	schema, err := xsd.NewCompiler().Compile(t.Context(), schemaDOC)
	require.NoError(t, err)

	cases := []struct {
		name     string
		instance string
		valid    bool
	}{
		{"omit Lead, pick first branch", `<root><P>x</P></root>`, true},
		{"omit Lead, pick second branch", `<root><Q>x</Q></root>`, true},
		{"include Lead, pick second branch", `<root><Lead>l</Lead><Q>x</Q></root>`, true},
		{"omit Lead, empty choice", `<root/>`, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			doc, err := helium.NewParser().Parse(t.Context(), []byte(tc.instance))
			require.NoError(t, err)

			var errs string
			err = validateWithOutput(t, xsd.NewValidator(schema), doc, &errs)
			if tc.valid {
				require.NoError(t, err, "expected valid, got errors: %s", errs)
			} else {
				require.Error(t, err)
			}
		})
	}
}
