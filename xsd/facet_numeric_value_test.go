package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestNumericFacetValueLexicalSpace verifies that the {value} of a length-family
// or digit facet must belong to its required built-in integer value space
// (xs:nonNegativeInteger for length/minLength/maxLength/fractionDigits,
// xs:positiveInteger for totalDigits). An out-of-space value such as "1e2",
// "-1", "a", "" or totalDigits="0" makes the schema in error — previously
// parseOccurs silently collapsed the bad value to 0 and the constraint became a
// no-op. This is an XSD 1.0 rule, so it is enforced in BOTH the default (1.0)
// and the Version11 mode; a valid integer value (including one with a leading
// '+', surrounding whitespace, or leading zeros) is still accepted.
func TestNumericFacetValueLexicalSpace(t *testing.T) {
	t.Parallel()

	const wantMsg = "is not a valid value of the type"
	const facetMaxLength = "maxLength"

	// compileFatal compiles schemaXML and returns the concatenated fatal
	// diagnostic text delivered to the ErrorHandler (empty when the schema is
	// valid), asserting the top-level error is only ErrCompilationFailed.
	compileFatal := func(t *testing.T, c xsd.Compiler, schemaXML string) string {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, cerr := c.Label("test.xsd").ErrorHandler(collector).Compile(t.Context(), doc)
		requireCompileResultErr(t, cerr)
		_, errText := partitionCompileErrors(collector.Errors())
		return errText
	}

	simpleTypeSchema := func(facet, val, base string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="bad">
    <xs:restriction base="` + base + `">
      <xs:` + facet + ` value="` + val + `"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="root" type="bad"/>
</xs:schema>`
	}

	// Each invalid case must be rejected under BOTH versions.
	invalid := []struct {
		name  string
		facet string
		val   string
		base  string
	}{
		{"maxLength float notation", facetMaxLength, "1e2", "xs:anyURI"},
		{"maxLength negative", facetMaxLength, "-1", xsStringType},
		{"maxLength non-numeric", facetMaxLength, "a", xsStringType},
		{"maxLength empty", facetMaxLength, "", xsStringType},
		{"minLength float notation", "minLength", "1e2", xsStringType},
		{"length negative", "length", "-1", xsStringType},
		{"totalDigits zero", "totalDigits", "0", xsDecimalType},
		{"totalDigits negative", "totalDigits", "-1", xsDecimalType},
		{"fractionDigits non-numeric", "fractionDigits", "a", xsDecimalType},
		{"fractionDigits float notation", "fractionDigits", "1e2", xsDecimalType},
	}
	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			schema := simpleTypeSchema(tc.facet, tc.val, tc.base)
			require.Contains(t, compileFatal(t, xsd.NewCompiler(), schema), wantMsg,
				"XSD 1.0 must reject %s value=%q", tc.facet, tc.val)
			require.Contains(t, compileFatal(t, xsd.NewCompiler().Version(xsd.Version11), schema), wantMsg,
				"XSD 1.1 must reject %s value=%q", tc.facet, tc.val)
		})
	}

	// Valid integer values must still compile (no over-rejection).
	valid := []struct {
		name  string
		facet string
		val   string
		base  string
	}{
		{"maxLength plain integer", facetMaxLength, "100", xsStringType},
		{"length zero", "length", "0", xsStringType},
		{"minLength leading plus", "minLength", "+5", xsStringType},
		{"maxLength surrounding whitespace", facetMaxLength, "  10  ", xsStringType},
		{"totalDigits one", "totalDigits", "1", xsDecimalType},
		{"fractionDigits zero", "fractionDigits", "0", xsDecimalType},
		{"totalDigits leading zeros", "totalDigits", "007", xsDecimalType},
	}
	for _, tc := range valid {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			schema := simpleTypeSchema(tc.facet, tc.val, tc.base)
			require.NoError(t, compileV(t, xsd.NewCompiler(), schema),
				"XSD 1.0 must accept %s value=%q", tc.facet, tc.val)
			require.NoError(t, compileV(t, xsd.NewCompiler().Version(xsd.Version11), schema),
				"XSD 1.1 must accept %s value=%q", tc.facet, tc.val)
		})
	}
}
