package xsd_test

import (
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
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
		// wantRejectMsg is the substring expected in the validation error when
		// wantReject is set. It defaults to the enumeration-facet message; cases
		// that are rejected at the lexical-type level (e.g. an out-of-space
		// lexical form) override it with the atomic-type message.
		wantRejectMsg string
	}

	cases := []testCase{
		// decimal — lexical variants that are value-equal to "5".
		{name: "decimal trailing zero", baseType: xsDecimalType, enum: []string{"5"}, instance: "5.0"},
		{name: "decimal more trailing zeros", baseType: xsDecimalType, enum: []string{"5"}, instance: "5.00"},
		{name: "decimal leading zero", baseType: xsDecimalType, enum: []string{"5"}, instance: "05"},
		{name: "decimal plus sign", baseType: xsDecimalType, enum: []string{"5"}, instance: "+5"},
		{name: "decimal non-member", baseType: xsDecimalType, enum: []string{"5"}, instance: "6", wantReject: true},

		// boolean — "true"/"1" and "false"/"0" are value-equal pairs.
		{name: "boolean true vs 1", baseType: xsBooleanType, enum: []string{lexicon.ValueTrue}, instance: "1"},
		{name: "boolean false vs 0", baseType: xsBooleanType, enum: []string{"false"}, instance: "0"},
		{name: "boolean non-member", baseType: xsBooleanType, enum: []string{lexicon.ValueTrue}, instance: "0", wantReject: true},

		// float / double — trailing zero and exponent forms.
		{name: "float trailing zero", baseType: xsFloatType, enum: []string{"1.5"}, instance: "1.50"},
		{name: "float exponent form", baseType: xsFloatType, enum: []string{"1.5"}, instance: "1.5E0"},
		{name: "double exponent form", baseType: xsDoubleType, enum: []string{"1.5"}, instance: "1.5E0"},
		{name: "double non-member", baseType: xsDoubleType, enum: []string{"1.5"}, instance: "2.5", wantReject: true},

		// float NaN — per XSD, NaN equals NaN for enumeration purposes. The only
		// valid lexical form is bare "NaN": signed forms "+NaN"/"-NaN" are not in
		// the xs:float/xs:double lexical space, so they must be rejected outright.
		{name: "float NaN matches NaN", baseType: xsFloatType, enum: []string{nanLexical}, instance: nanLexical},
		{name: "double NaN matches NaN", baseType: xsDoubleType, enum: []string{nanLexical}, instance: nanLexical},
		{name: "float signed NaN rejected", baseType: xsFloatType, enum: []string{nanLexical}, instance: "+NaN", wantReject: true, wantRejectMsg: "is not a valid value of the atomic type 'xs:float'"},
		{name: "double signed NaN rejected", baseType: xsDoubleType, enum: []string{nanLexical}, instance: "-NaN", wantReject: true, wantRejectMsg: "is not a valid value of the atomic type 'xs:double'"},
		// A signed-NaN enumeration member is itself an invalid lexical form; it must
		// not value-match a bare "NaN" instance via the value-equality path.
		{name: "float signed NaN member does not match NaN", baseType: xsFloatType, enum: []string{"+NaN"}, instance: nanLexical, wantReject: true},
		{name: "double signed NaN member does not match NaN", baseType: xsDoubleType, enum: []string{"-NaN"}, instance: nanLexical, wantReject: true},

		// hexBinary — value space is the decoded octets, so case differences are
		// not significant ("0A" == "0a"); a different byte must be rejected.
		{name: "hexBinary case-insensitive", baseType: xsHexBinaryType, enum: []string{"0A"}, instance: "0a"},
		{name: "hexBinary mixed case member", baseType: xsHexBinaryType, enum: []string{"deadBEEF"}, instance: "DEADbeef"},
		{name: "hexBinary non-member", baseType: xsHexBinaryType, enum: []string{"0A"}, instance: "0b", wantReject: true},

		// base64Binary — value space is the decoded octets; whitespace in the
		// lexical form is not significant.
		{name: "base64Binary whitespace insignificant", baseType: "xs:base64Binary", enum: []string{"YWJj"}, instance: "YW Jj"},
		{name: "base64Binary non-member", baseType: "xs:base64Binary", enum: []string{"YWJj"}, instance: "YWJk", wantReject: true},
		// A padded instance that is byte-equal to its member but written with
		// extra whitespace must still value-match (whitespace is insignificant).
		{name: "base64Binary padded whitespace matches", baseType: "xs:base64Binary", enum: []string{"TQ=="}, instance: "TQ =="},

		// dateTime — same instant written with a different timezone form.
		{name: "dateTime equal across timezone", baseType: "xs:dateTime",
			enum: []string{"2000-01-01T12:00:00Z"}, instance: "2000-01-01T13:00:00+01:00"},
		{name: "dateTime trailing-zero seconds", baseType: "xs:dateTime",
			enum: []string{"2000-01-01T12:00:00Z"}, instance: "2000-01-01T12:00:00.0Z"},
		{name: "dateTime non-member", baseType: "xs:dateTime",
			enum: []string{"2000-01-01T12:00:00Z"}, instance: "2000-01-01T12:00:01Z", wantReject: true},

		// string — lexical members must still be matched lexically.
		{name: "string member", baseType: xsStringType, enum: []string{"alpha", "beta"}, instance: "beta"},
		{name: "string non-member", baseType: xsStringType, enum: []string{"alpha"}, instance: "gamma", wantReject: true},

		// string-family types must stay lexical-only: a numeric-looking instance
		// must NOT be accepted via numeric value-space comparison against a
		// numeric-looking member ("5" must not accept "5.0").
		{name: "string numeric lexical not value-equal", baseType: xsStringType,
			enum: []string{"5"}, instance: "5.0", wantReject: true},
		{name: "token numeric lexical not value-equal", baseType: "xs:token",
			enum: []string{"10"}, instance: "1e1", wantReject: true},
		{name: "anyURI numeric lexical not value-equal", baseType: "xs:anyURI",
			enum: []string{"5"}, instance: "5.00", wantReject: true},
		{name: "string numeric member exact", baseType: xsStringType, enum: []string{"5"}, instance: "5"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var enumBuilder strings.Builder
			for _, e := range tc.enum {
				enumBuilder.WriteString(`      <xs:enumeration value="` + e + `"/>` + "\n")
			}
			enumXML := enumBuilder.String()

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
				wantMsg := tc.wantRejectMsg
				if wantMsg == "" {
					wantMsg = "[facet 'enumeration']"
				}
				require.Contains(t, errs, wantMsg)
				return
			}
			require.NoError(t, err, "validation errors: %s", errs)
		})
	}
}
