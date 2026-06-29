package xsd_test

import (
	"os"
	"path/filepath"
	"strings"
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

func TestCompileNestedSchemaInjectedParserDoesNotForceEntityExpansion(t *testing.T) {
	t.Parallel()

	rootSchema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:include schemaLocation="types.xsd"/>
  <xs:element name="root" type="uriType"/>
</xs:schema>`
	rootDoc, err := helium.NewParser().Parse(t.Context(), []byte(rootSchema))
	require.NoError(t, err)

	fsys := fstest.MapFS{
		"types.xsd": &fstest.MapFile{Data: []byte(entityBackedSmallPatternSchema(t))},
	}
	_, err = xsd.NewCompiler().
		FS(fsys).
		Parser(helium.NewParser().MaxNameLength(-1).MaxNodeContentSize(64)).
		Compile(t.Context(), rootDoc)
	require.ErrorIs(t, err, helium.ErrNodeContentTooLarge)
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

func TestCompileFileInjectedParserDoesNotForceEntityExpansion(t *testing.T) {
	t.Parallel()

	schemaPath := writeSchema(t, entityBackedSmallPatternSchema(t))

	_, err := xsd.NewCompiler().
		Parser(helium.NewParser().MaxNameLength(-1).MaxNodeContentSize(64)).
		CompileFile(t.Context(), schemaPath)
	require.ErrorIs(t, err, helium.ErrNodeContentTooLarge)
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
	return writeSchema(t, schemaSrc)
}

func entityBackedSmallPatternSchema(t *testing.T) string {
	t.Helper()

	entityName := "allowed" + strings.Repeat("e", 4096)
	return `<!DOCTYPE xs:schema [
<!ENTITY ` + entityName + ` "ok">
]>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="uriType"/>
  <xs:simpleType name="uriType">
    <xs:restriction base="xs:anyURI">
      <xs:pattern value="&` + entityName + `;"/>
    </xs:restriction>
  </xs:simpleType>
</xs:schema>`
}

func writeSchema(t *testing.T, schemaSrc string) string {
	t.Helper()

	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "schema.xsd")
	require.NoError(t, os.WriteFile(schemaPath, []byte(schemaSrc), 0o644))

	return schemaPath
}

func validateXML(t *testing.T, schema *xsd.Schema, src string) error {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err)
	return xsd.NewValidator(schema).Validate(t.Context(), doc)
}
