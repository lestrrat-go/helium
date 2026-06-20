package xsd_test

import (
	"strings"
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestContentModelDuplicateSource verifies that the content-model grammar
// diagnostics added to parseComplexType (a second model-group particle, or a
// model-group particle alongside a simple/complexContent wrapper) are attributed
// to the DECLARING file when the offending xs:complexType is pulled in via
// xs:include. Before the fix these diagnostics used c.filename directly, so an
// included-file line number was paired with the including (top-level) schema
// label, producing a mismatched file:line.
func TestContentModelDuplicateSource(t *testing.T) {
	t.Parallel()

	const (
		mainXSD = "cm_main.xsd"
		incXSD  = "cm_inc.xsd"
	)

	main := []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:include schemaLocation="cm_inc.xsd"/>
  <xs:element name="root" type="xs:string"/>
</xs:schema>`)

	cases := []struct {
		name string
		inc  string
		want string
	}{
		{
			name: "two model group particles",
			inc: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="t">
    <xs:sequence/>
    <xs:choice/>
  </xs:complexType>
</xs:schema>`,
			want: "more than one content model particle",
		},
		{
			name: "particle with complexContent wrapper",
			inc: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="t">
    <xs:sequence/>
    <xs:complexContent>
      <xs:restriction base="xs:anyType"/>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`,
			want: "not allowed together with the content model particle",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fsys := fstest.MapFS{
				mainXSD: &fstest.MapFile{Data: main},
				incXSD:  &fstest.MapFile{Data: []byte(tc.inc)},
			}
			doc, err := helium.NewParser().Parse(t.Context(), main)
			require.NoError(t, err)
			collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
			_, err = xsd.NewCompiler().Label(mainXSD).ErrorHandler(collector).FS(fsys).Compile(t.Context(), doc)
			requireCompileResultErr(t, err)
			require.NoError(t, collector.Close())
			_, errStr := partitionCompileErrors(collector.Errors())

			require.Contains(t, errStr, tc.want, "expected the content-model diagnostic")
			require.Contains(t, errStr, incXSD+":",
				"diagnostic must be attributed to the declaring file; got: %q", errStr)
			require.False(t, strings.Contains(errStr, mainXSD+":"),
				"diagnostic must not cite the top-level schema label; got: %q", errStr)
		})
	}
}
