package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestEnumerationValueSpace verifies that the enumeration facet is compared in
// value space, not raw lexical space. A value that is lexically distinct from
// every enumeration member but value-equal to one of them must be accepted; a
// value that is value-distinct from all members must be rejected.
func TestEnumerationValueSpace(t *testing.T) {
	type testCase struct {
		name       string
		baseType   string
		enum       []string
		instance   string
		wantReject bool
	}

	cases := []testCase{
		// decimal — lexical variants that are value-equal to "5".
		{name: "decimal trailing zero", baseType: "xs:decimal", enum: []string{"5"}, instance: "5.0"},
		{name: "decimal more trailing zeros", baseType: "xs:decimal", enum: []string{"5"}, instance: "5.00"},
		{name: "decimal leading zero", baseType: "xs:decimal", enum: []string{"5"}, instance: "05"},
		{name: "decimal plus sign", baseType: "xs:decimal", enum: []string{"5"}, instance: "+5"},
		{name: "decimal non-member", baseType: "xs:decimal", enum: []string{"5"}, instance: "6", wantReject: true},

		// boolean — "true"/"1" and "false"/"0" are value-equal pairs.
		{name: "boolean true vs 1", baseType: "xs:boolean", enum: []string{"true"}, instance: "1"},
		{name: "boolean false vs 0", baseType: "xs:boolean", enum: []string{"false"}, instance: "0"},
		{name: "boolean non-member", baseType: "xs:boolean", enum: []string{"true"}, instance: "0", wantReject: true},

		// float / double — trailing zero and exponent forms.
		{name: "float trailing zero", baseType: "xs:float", enum: []string{"1.5"}, instance: "1.50"},
		{name: "float exponent form", baseType: "xs:float", enum: []string{"1.5"}, instance: "1.5E0"},
		{name: "double exponent form", baseType: "xs:double", enum: []string{"1.5"}, instance: "1.5E0"},
		{name: "double non-member", baseType: "xs:double", enum: []string{"1.5"}, instance: "2.5", wantReject: true},

		// float NaN — per XSD, NaN equals NaN for enumeration purposes.
		{name: "float NaN matches NaN", baseType: "xs:float", enum: []string{"NaN"}, instance: "NaN"},
		{name: "double NaN matches NaN", baseType: "xs:double", enum: []string{"NaN"}, instance: "NaN"},

		// dateTime — same instant written with a different timezone form.
		{name: "dateTime equal across timezone", baseType: "xs:dateTime",
			enum: []string{"2000-01-01T12:00:00Z"}, instance: "2000-01-01T13:00:00+01:00"},
		{name: "dateTime trailing-zero seconds", baseType: "xs:dateTime",
			enum: []string{"2000-01-01T12:00:00Z"}, instance: "2000-01-01T12:00:00.0Z"},
		{name: "dateTime non-member", baseType: "xs:dateTime",
			enum: []string{"2000-01-01T12:00:00Z"}, instance: "2000-01-01T12:00:01Z", wantReject: true},

		// string — lexical members must still be matched lexically.
		{name: "string member", baseType: "xs:string", enum: []string{"alpha", "beta"}, instance: "beta"},
		{name: "string non-member", baseType: "xs:string", enum: []string{"alpha"}, instance: "gamma", wantReject: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var enumXML string
			for _, e := range tc.enum {
				enumXML += `      <xs:enumeration value="` + e + `"/>` + "\n"
			}

			schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:simpleType>
      <xs:restriction base="` + tc.baseType + `">
` + enumXML + `      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`

			instanceXML := `<root>` + tc.instance + `</root>`

			schemaDOC, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
			require.NoError(t, err)

			schema, err := xsd.NewCompiler().Compile(t.Context(), schemaDOC)
			require.NoError(t, err)

			doc, err := helium.NewParser().Parse(t.Context(), []byte(instanceXML))
			require.NoError(t, err)

			var errs string
			err = validateWithOutput(t, xsd.NewValidator(schema), doc, &errs)

			if tc.wantReject {
				require.Error(t, err)
				require.Contains(t, errs, "[facet 'enumeration']")
				return
			}
			require.NoError(t, err, "validation errors: %s", errs)
		})
	}
}
