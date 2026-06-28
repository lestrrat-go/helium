package xsd_test

import (
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
