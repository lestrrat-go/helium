package xsd_test

import (
	"strings"
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// The root @xpathDefaultNamespace of an INCLUDED or IMPORTED schema must govern
// that schema's own identity-constraint selector/field XPaths. The schema-level
// default is a PER-document setting (like elementFormDefault/blockDefault), so it
// must be set from the loaded root — not inherited from the including/importing
// schema. Without that, an included/imported IDC selector like xpath="emp" would
// resolve unprefixed names to no-namespace and silently miss duplicates in a
// namespaced instance (a false-accept).
func TestIDCXPathDefaultNamespaceAcrossDocuments(t *testing.T) {
	t.Parallel()

	compileMain := func(t *testing.T, fsys fstest.MapFS, mainName string) *xsd.Schema {
		t.Helper()
		data, err := fsys.ReadFile(mainName)
		require.NoError(t, err)
		doc, err := helium.NewParser().Parse(t.Context(), data)
		require.NoError(t, err)
		schema, err := xsd.NewCompiler().Version(xsd.Version11).Label(mainName).FS(fsys).Compile(t.Context(), doc)
		require.NoError(t, err)
		require.NotNil(t, schema)
		return schema
	}

	t.Run("included schema root xpathDefaultNamespace", func(t *testing.T) {
		t.Parallel()
		const ns = "urn:inc"
		fsys := fstest.MapFS{
			importMainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="` + ns + `">
  <xs:include schemaLocation="inc.xsd"/>
</xs:schema>`)},
			// The INCLUDED root declares xpathDefaultNamespace="##targetNamespace"; main
			// declares none. The unique on <doc> uses unprefixed selector/field, which
			// must resolve into urn:inc for the namespaced instance.
			"inc.xsd": &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:t="` + ns + `" targetNamespace="` + ns + `" elementFormDefault="qualified" xpathDefaultNamespace="##targetNamespace">
  <xs:element name="doc">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="emp" type="t:empType" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
    <xs:unique name="u">
      <xs:selector xpath="emp"/>
      <xs:field xpath="nr"/>
    </xs:unique>
  </xs:element>
  <xs:complexType name="empType">
    <xs:sequence><xs:element name="nr" type="xs:integer"/></xs:sequence>
  </xs:complexType>
</xs:schema>`)},
		}
		schema := compileMain(t, fsys, importMainXSD)

		dup := `<doc xmlns="` + ns + `"><emp><nr>1</nr></emp><emp><nr>1</nr></emp></doc>`
		ddoc, err := helium.NewParser().Parse(t.Context(), []byte(dup))
		require.NoError(t, err)
		require.Error(t, xsd.NewValidator(schema).Validate(t.Context(), ddoc),
			"duplicate <nr> must be caught via the included root's xpathDefaultNamespace")

		ok := `<doc xmlns="` + ns + `"><emp><nr>1</nr></emp><emp><nr>2</nr></emp></doc>`
		odoc, err := helium.NewParser().Parse(t.Context(), []byte(ok))
		require.NoError(t, err)
		require.NoError(t, xsd.NewValidator(schema).Validate(t.Context(), odoc))
	})

	t.Run("imported schema root xpathDefaultNamespace", func(t *testing.T) {
		t.Parallel()
		const mainNS = "urn:t"
		const impNS = "urn:imp"
		fsys := fstest.MapFS{
			importMainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="` + mainNS + `">
  <xs:import namespace="` + impNS + `" schemaLocation="imp.xsd"/>
</xs:schema>`)},
			// The IMPORTED root declares xpathDefaultNamespace="##targetNamespace"
			// (= urn:imp); main declares none.
			"imp.xsd": &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:i="` + impNS + `" targetNamespace="` + impNS + `" elementFormDefault="qualified" xpathDefaultNamespace="##targetNamespace">
  <xs:element name="doc">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="emp" type="i:empType" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
    <xs:unique name="u">
      <xs:selector xpath="emp"/>
      <xs:field xpath="nr"/>
    </xs:unique>
  </xs:element>
  <xs:complexType name="empType">
    <xs:sequence><xs:element name="nr" type="xs:integer"/></xs:sequence>
  </xs:complexType>
</xs:schema>`)},
		}
		schema := compileMain(t, fsys, importMainXSD)

		dup := `<doc xmlns="` + impNS + `"><emp><nr>1</nr></emp><emp><nr>1</nr></emp></doc>`
		ddoc, err := helium.NewParser().Parse(t.Context(), []byte(dup))
		require.NoError(t, err)
		require.Error(t, xsd.NewValidator(schema).Validate(t.Context(), ddoc),
			"duplicate <nr> must be caught via the imported root's xpathDefaultNamespace")

		ok := `<doc xmlns="` + impNS + `"><emp><nr>1</nr></emp><emp><nr>2</nr></emp></doc>`
		odoc, err := helium.NewParser().Parse(t.Context(), []byte(ok))
		require.NoError(t, err)
		require.NoError(t, xsd.NewValidator(schema).Validate(t.Context(), odoc))
	})
}

// TestIDCReferSourceInIncludedSchema covers PR860-REVIEW-012: a malformed or
// unbound-prefix @refer in an INCLUDED schema must be attributed to the INCLUDED
// file (whose line number the diagnostic carries), not the including schema.
func TestIDCReferSourceInIncludedSchema(t *testing.T) {
	t.Parallel()

	const (
		mainXSD = "kr_main.xsd"
		incXSD  = "kr_inc.xsd"
	)

	assert := func(t *testing.T, refer string) {
		t.Helper()
		fsys := fstest.MapFS{
			mainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:include schemaLocation="kr_inc.xsd"/>
</xs:schema>`)},
			// The keyref (with the bad @refer) lives entirely in the included file,
			// so the diagnostic's line number is meaningful only when paired with it.
			incXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="x" maxOccurs="unbounded">
          <xs:complexType><xs:attribute name="y" type="xs:string"/></xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
    <xs:keyref name="kr" refer="` + refer + `">
      <xs:selector xpath="x"/>
      <xs:field xpath="@y"/>
    </xs:keyref>
  </xs:element>
</xs:schema>`)},
		}
		data, err := fsys.ReadFile(mainXSD)
		require.NoError(t, err)
		doc, err := helium.NewParser().Parse(t.Context(), data)
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, err = xsd.NewCompiler().Version(xsd.Version11).Label(mainXSD).ErrorHandler(collector).FS(fsys).Compile(t.Context(), doc)
		requireCompileResultErr(t, err)
		require.NoError(t, collector.Close())
		_, errStr := partitionCompileErrors(collector.Errors())

		require.Contains(t, errStr, incXSD+":",
			"the bad @refer diagnostic must cite the included file; got: %q", errStr)
		require.False(t, strings.Contains(errStr, mainXSD+":"),
			"the diagnostic must not cite the including schema; got: %q", errStr)
	}

	t.Run("malformed refer", func(t *testing.T) {
		t.Parallel()
		assert(t, ":k") // not a valid xs:QName (empty prefix)
	})
	t.Run("unbound-prefix refer", func(t *testing.T) {
		t.Parallel()
		assert(t, "bad:k") // prefix bad is not bound in scope
	})
}
