package xsd_test

import (
	"strings"
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestIncludeSchemaLocationRequired covers src-include.1 / src-redefine.1
// (§4.2.3 / §4.2.5 and the schema-for-schemas): @schemaLocation is REQUIRED on
// xs:include and xs:redefine. Its ABSENCE is a schema-representation error
// (reject the schema), distinct from a present-but-unresolvable schemaLocation
// hint (a warning that skips the composition element). Version-independent.
//
// Mirrors the W3C xmlschema msMeta Schema_w3c schB3 (an <xs:include/> with no
// @schemaLocation), which the schema-for-schemas rejects.
func TestIncludeSchemaLocationRequired(t *testing.T) {
	t.Parallel()

	const (
		mainXSD = "main.xsd"
		incXSD  = "inc.xsd"
	)

	compile := func(t *testing.T, src string, fsys fstest.MapFS) (warnings, errStr string) {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		c := xsd.NewCompiler().Label(mainXSD).ErrorHandler(collector)
		if fsys != nil {
			c = c.FS(fsys)
		}
		_, cerr := c.Compile(t.Context(), doc)
		requireCompileResultErr(t, cerr)
		require.NoError(t, collector.Close())
		return partitionCompileErrors(collector.Errors())
	}

	t.Run("schB3_include_without_schemaLocation_rejected", func(t *testing.T) {
		t.Parallel()
		_, errStr := compile(t, `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <xsd:include/>
</xsd:schema>`, nil)
		require.Contains(t, errStr, "The attribute 'schemaLocation' is required but missing.",
			"a missing schemaLocation on xs:include must be a fatal schema-representation error; got: %q", errStr)
	})

	t.Run("redefine_without_schemaLocation_rejected", func(t *testing.T) {
		t.Parallel()
		_, errStr := compile(t, `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <xsd:redefine>
    <xsd:simpleType name="codeType">
      <xsd:restriction base="codeType"><xsd:maxLength value="5"/></xsd:restriction>
    </xsd:simpleType>
  </xsd:redefine>
</xsd:schema>`, nil)
		require.Contains(t, errStr, "The attribute 'schemaLocation' is required but missing.",
			"a missing schemaLocation on xs:redefine must be a fatal schema-representation error; got: %q", errStr)
	})

	t.Run("valid_include_with_schemaLocation_compiles", func(t *testing.T) {
		t.Parallel()
		fsys := fstest.MapFS{
			mainXSD: &fstest.MapFile{Data: []byte(``)},
			incXSD: &fstest.MapFile{Data: []byte(
				`<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <xsd:element name="child" type="xsd:string"/>
</xsd:schema>`)},
		}
		warnings, errStr := compile(t, `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <xsd:include schemaLocation="`+incXSD+`"/>
  <xsd:element name="root" type="xsd:string"/>
</xsd:schema>`, fsys)
		require.Empty(t, errStr, "a valid include with a resolvable schemaLocation must compile cleanly; got: %q", errStr)
		require.NotContains(t, warnings, "Skipping the include.",
			"a resolvable include must not warn; got: %q", warnings)
	})

	t.Run("unresolvable_schemaLocation_warns_not_error", func(t *testing.T) {
		t.Parallel()
		fsys := fstest.MapFS{
			mainXSD: &fstest.MapFile{Data: []byte(``)},
			// missing.xsd is absent so the loader gets fs.ErrNotExist.
		}
		warnings, errStr := compile(t, `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <xsd:include schemaLocation="missing.xsd"/>
  <xsd:element name="root" type="xsd:string"/>
</xsd:schema>`, fsys)
		require.Empty(t, errStr,
			"a present-but-unresolvable schemaLocation must remain a warning, not a fatal error; got: %q", errStr)
		require.Equal(t, 1, strings.Count(warnings, "Skipping the include."),
			"an unresolvable include must warn and skip; got: %q", warnings)
	})
}
