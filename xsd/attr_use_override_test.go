package xsd_test

import (
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestAttributeUseOverrideGlobal covers the schema-representation constraint on
// the `use` attribute of a top-level <xs:attribute> that an XSD 1.1 <xs:override>
// registers as a wholesale replacement. Such an attribute's DOM parent is
// <xs:override>, not <xs:schema>, but it is still a GLOBAL (top-level) attribute
// declaration whose schema-for-schemas type (topLevelAttribute) does not permit
// `use`, so a `use` on it must be rejected. xs:override is 1.1-only, so this does
// not affect XSD 1.0 (which ignores xs:override entirely).
func TestAttributeUseOverrideGlobal(t *testing.T) {
	t.Parallel()

	compile11 := func(t *testing.T, fsys fstest.MapFS, mainName string) string {
		t.Helper()
		data, err := fsys.ReadFile(mainName)
		require.NoError(t, err)
		doc, err := helium.NewParser().Parse(t.Context(), data)
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, cerr := xsd.NewCompiler().Version(xsd.Version11).Label(mainName).ErrorHandler(collector).FS(fsys).Compile(t.Context(), doc)
		requireCompileResultErr(t, cerr)
		require.NoError(t, collector.Close())
		_, errStr := partitionCompileErrors(collector.Errors())
		return errStr
	}

	const main = "ovmain.xsd"

	t.Run("use on override global attribute is rejected", func(t *testing.T) {
		for _, use := range []string{"required", "optional", "prohibited"} {
			fsys := fstest.MapFS{
				main: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:m">
  <xs:override schemaLocation="ovtarget.xsd">
    <xs:attribute name="foo" type="xs:string" use="` + use + `"/>
  </xs:override>
</xs:schema>`)},
				"ovtarget.xsd": &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:m">
  <xs:attribute name="foo" type="xs:string"/>
</xs:schema>`)},
			}
			errStr := compile11(t, fsys, main)
			require.Contains(t, errStr, "The attribute 'use' is not allowed",
				"override global attribute with use=%q must be rejected; got: %q", use, errStr)
		}
	})

	t.Run("override global attribute without use compiles", func(t *testing.T) {
		fsys := fstest.MapFS{
			main: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:m">
  <xs:override schemaLocation="ovtarget.xsd">
    <xs:attribute name="foo" type="xs:string"/>
  </xs:override>
</xs:schema>`)},
			"ovtarget.xsd": &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:m">
  <xs:attribute name="foo" type="xs:string"/>
</xs:schema>`)},
		}
		require.Empty(t, compile11(t, fsys, main))
	})
}
