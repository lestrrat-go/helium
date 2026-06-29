package xsd_test

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

const entityBackedAnyURISchema = `<!DOCTYPE xs:schema [
<!ENTITY allowed "http://example\.com(/[A-Za-z]+)?">
]>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="uriType">
    <xs:restriction base="xs:anyURI">
      <xs:pattern value="&allowed;"/>
    </xs:restriction>
  </xs:simpleType>
</xs:schema>`

func TestCompileNestedSchemaExpandsPatternEntityReferences(t *testing.T) {
	t.Parallel()

	rootSchema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:include schemaLocation="types.xsd"/>
  <xs:element name="root" type="uriType"/>
</xs:schema>`
	rootDoc, err := helium.NewParser().Parse(t.Context(), []byte(rootSchema))
	require.NoError(t, err)

	fsys := fstest.MapFS{
		"types.xsd": &fstest.MapFile{Data: []byte(entityBackedAnyURISchema)},
	}
	schema, err := xsd.NewCompiler().FS(fsys).Compile(t.Context(), rootDoc)
	require.NoError(t, err)

	require.NoError(t, validateXML(t, schema, `<root>http://example.com/path</root>`))
	require.Error(t, validateXML(t, schema, `<root>http://other.example/path</root>`))
}

func TestCompileNestedSchemaUsesInjectedParserPolicy(t *testing.T) {
	t.Parallel()

	rootSchema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:include schemaLocation="types.xsd"/>
  <xs:element name="root" type="uriType"/>
</xs:schema>`
	rootDoc, err := helium.NewParser().Parse(t.Context(), []byte(rootSchema))
	require.NoError(t, err)

	fsys := fstest.MapFS{
		"types.xsd": &fstest.MapFile{Data: []byte(entityBackedAnyURISchema)},
	}
	_, err = xsd.NewCompiler().FS(fsys).Parser(helium.NewParser().MaxDepth(2)).Compile(t.Context(), rootDoc)
	require.Error(t, err, "nested schema parsing must use the injected parser policy")

	schema, err := xsd.NewCompiler().FS(fsys).Compile(t.Context(), rootDoc)
	require.NoError(t, err)
	require.NoError(t, validateXML(t, schema, `<root>http://example.com/path</root>`))
}

func TestCompileNestedSchemaInjectedParserCanExpandEntityReferences(t *testing.T) {
	t.Parallel()

	rootSchema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:include schemaLocation="types.xsd"/>
  <xs:element name="root" type="uriType"/>
</xs:schema>`
	rootDoc, err := helium.NewParser().Parse(t.Context(), []byte(rootSchema))
	require.NoError(t, err)

	fsys := fstest.MapFS{
		"types.xsd": &fstest.MapFile{Data: []byte(entityBackedAnyURISchema)},
	}
	schema, err := xsd.NewCompiler().
		FS(fsys).
		Parser(helium.NewParser().SubstituteEntities(true)).
		Compile(t.Context(), rootDoc)
	require.NoError(t, err)

	require.NoError(t, validateXML(t, schema, `<root>http://example.com/path</root>`))
	require.Error(t, validateXML(t, schema, `<root>http://other.example/path</root>`))
}

func TestCompileFileExpandsPatternEntityReferences(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "schema.xsd")
	schemaSrc := `<!DOCTYPE xs:schema [
<!ENTITY allowed "http://example\.com(/[A-Za-z]+)?">
]>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="uriType"/>
  <xs:simpleType name="uriType">
    <xs:restriction base="xs:anyURI">
      <xs:pattern value="&allowed;"/>
    </xs:restriction>
  </xs:simpleType>
</xs:schema>`
	require.NoError(t, os.WriteFile(schemaPath, []byte(schemaSrc), 0o644))

	schema, err := xsd.NewCompiler().CompileFile(t.Context(), schemaPath)
	require.NoError(t, err)

	require.NoError(t, validateXML(t, schema, `<root>http://example.com/path</root>`))
	require.Error(t, validateXML(t, schema, `<root>http://other.example/path</root>`))
}

func TestCompileFileUsesInjectedParserPolicy(t *testing.T) {
	t.Parallel()

	schemaPath := writeEntityBackedRootSchema(t)

	_, err := xsd.NewCompiler().Parser(helium.NewParser().MaxDepth(2)).CompileFile(t.Context(), schemaPath)
	require.Error(t, err, "top-level schema parsing must use the injected parser policy")

	schema, err := xsd.NewCompiler().CompileFile(t.Context(), schemaPath)
	require.NoError(t, err)
	require.NoError(t, validateXML(t, schema, `<root>http://example.com/path</root>`))
}

func TestCompileFileInjectedParserCanExpandEntityReferences(t *testing.T) {
	t.Parallel()

	schemaPath := writeEntityBackedRootSchema(t)

	schema, err := xsd.NewCompiler().
		Parser(helium.NewParser().SubstituteEntities(true)).
		CompileFile(t.Context(), schemaPath)
	require.NoError(t, err)

	require.NoError(t, validateXML(t, schema, `<root>http://example.com/path</root>`))
	require.Error(t, validateXML(t, schema, `<root>http://other.example/path</root>`))
}

func writeEntityBackedRootSchema(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "schema.xsd")
	schemaSrc := `<!DOCTYPE xs:schema [
<!ENTITY allowed "http://example\.com(/[A-Za-z]+)?">
]>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="uriType"/>
  <xs:simpleType name="uriType">
    <xs:restriction base="xs:anyURI">
      <xs:pattern value="&allowed;"/>
    </xs:restriction>
  </xs:simpleType>
</xs:schema>`
	require.NoError(t, os.WriteFile(schemaPath, []byte(schemaSrc), 0o644))

	return schemaPath
}

func validateXML(t *testing.T, schema *xsd.Schema, src string) error {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err)
	return xsd.NewValidator(schema).Validate(t.Context(), doc)
}
