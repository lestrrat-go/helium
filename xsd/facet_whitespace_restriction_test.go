package xsd_test

import (
	"fmt"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestWhiteSpaceValidRestriction verifies whiteSpace Valid Restriction (§4.3.6):
// the derived whiteSpace facet must not be LESS restrictive than the base type's
// effective whiteSpace. The ordering is preserve < replace < collapse, so
// restricting xs:normalizedString (replace) or xs:token (collapse) to preserve is
// a schema error in BOTH XSD 1.0 and 1.1. This is the root cause behind the
// msMeta/DataTypes_w3c false-accepts normalizedString_whitespace001 and
// token_whitespace001.
func TestWhiteSpaceValidRestriction(t *testing.T) {
	t.Parallel()

	const wantMsg = "is less restrictive than the 'whiteSpace' value"
	const tokenBase = "xs:token"

	schemaFor := func(base, ws string) string {
		return fmt.Sprintf(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="t">
    <xs:restriction base="%s">
      <xs:whiteSpace value="%s"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="root" type="t"/>
</xs:schema>`, base, ws)
	}

	compile := func(t *testing.T, schemaXML string) (string, error) {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, cerr := xsd.NewCompiler().Label("test.xsd").ErrorHandler(collector).Compile(t.Context(), doc)
		_, errStr := partitionCompileErrors(collector.Errors())
		return errStr, cerr
	}

	reject := []struct{ base, ws string }{
		{"xs:normalizedString", "preserve"}, // replace base, less restrictive
		{tokenBase, "preserve"},             // collapse base, less restrictive
		{tokenBase, "replace"},              // collapse base, less restrictive
		{"xs:language", "preserve"},         // collapse base (token-derived)
	}
	for _, tc := range reject {
		t.Run(fmt.Sprintf("reject %s whiteSpace=%s", tc.base, tc.ws), func(t *testing.T) {
			t.Parallel()
			errStr, cerr := compile(t, schemaFor(tc.base, tc.ws))
			requireCompileResultErr(t, cerr)
			require.Contains(t, errStr, wantMsg)
		})
	}

	accept := []struct{ base, ws string }{
		{"xs:string", "preserve"},           // preserve base, equal
		{"xs:string", "replace"},            // preserve base, more restrictive
		{"xs:string", "collapse"},           // preserve base, more restrictive
		{"xs:normalizedString", "replace"},  // replace base, equal
		{"xs:normalizedString", "collapse"}, // replace base, more restrictive
		{tokenBase, "collapse"},             // collapse base, equal
	}
	for _, tc := range accept {
		t.Run(fmt.Sprintf("accept %s whiteSpace=%s", tc.base, tc.ws), func(t *testing.T) {
			t.Parallel()
			_, cerr := compile(t, schemaFor(tc.base, tc.ws))
			require.NoError(t, cerr)
		})
	}
}
