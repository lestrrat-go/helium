package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// compileErrorsExact compiles schemaXML under the label "test.xsd", closes the
// ErrorCollector before reading (so the async sink is fully drained and the read
// is not flaky), and returns the concatenated fatal compile-error output. The
// label fallback (top-level duplicate keeps the "test.xsd" prefix instead of
// starting with ":line:") is exercised by the exact-output assertions below.
func compileErrorsExact(t *testing.T, schemaXML string) string {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	_, err = xsd.NewCompiler().Label("test.xsd").ErrorHandler(collector).Compile(t.Context(), doc)
	require.NoError(t, err)
	require.NoError(t, collector.Close())
	_, errors := partitionCompileErrors(collector.Errors())
	return errors
}

// TestDuplicateNamedComponent verifies that two global components sharing the
// same name (simpleType, complexType, model group, or attributeGroup) are
// reported as a single fatal schema parser error — with the correct file-label
// prefix and no extra follow-on diagnostics — instead of silently overwriting
// the first definition.
func TestDuplicateNamedComponent(t *testing.T) {
	t.Parallel()

	t.Run("duplicate simpleType", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="T">
    <xs:restriction base="xs:string"/>
  </xs:simpleType>
  <xs:simpleType name="T">
    <xs:restriction base="xs:int"/>
  </xs:simpleType>
</xs:schema>`
		require.Equal(t,
			"test.xsd:5: element simpleType: Schemas parser error : Element '{http://www.w3.org/2001/XMLSchema}simpleType': A global type definition ''T does already exist.\n",
			compileErrorsExact(t, schemaXML))
	})

	t.Run("duplicate complexType", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="T">
    <xs:sequence/>
  </xs:complexType>
  <xs:complexType name="T">
    <xs:sequence/>
  </xs:complexType>
</xs:schema>`
		require.Equal(t,
			"test.xsd:5: element complexType: Schemas parser error : Element '{http://www.w3.org/2001/XMLSchema}complexType': A global type definition ''T does already exist.\n",
			compileErrorsExact(t, schemaXML))
	})

	t.Run("duplicate group", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:group name="G">
    <xs:sequence>
      <xs:element name="a" type="xs:string"/>
    </xs:sequence>
  </xs:group>
  <xs:group name="G">
    <xs:sequence>
      <xs:element name="b" type="xs:string"/>
    </xs:sequence>
  </xs:group>
</xs:schema>`
		require.Equal(t,
			"test.xsd:7: element group: Schemas parser error : Element '{http://www.w3.org/2001/XMLSchema}group': A global model group definition ''G does already exist.\n",
			compileErrorsExact(t, schemaXML))
	})

	t.Run("duplicate attributeGroup", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attributeGroup name="AG">
    <xs:attribute name="a" type="xs:string"/>
  </xs:attributeGroup>
  <xs:attributeGroup name="AG">
    <xs:attribute name="b" type="xs:string"/>
  </xs:attributeGroup>
</xs:schema>`
		require.Equal(t,
			"test.xsd:5: element attributeGroup: Schemas parser error : Element '{http://www.w3.org/2001/XMLSchema}attributeGroup': A global attribute group definition ''AG does already exist.\n",
			compileErrorsExact(t, schemaXML))
	})

	t.Run("duplicate global attribute", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attribute name="A" type="xs:string"/>
  <xs:attribute name="A" type="xs:int"/>
</xs:schema>`
		require.Equal(t,
			"test.xsd:3: element attribute: Schemas parser error : Element '{http://www.w3.org/2001/XMLSchema}attribute': A global attribute declaration ''A does already exist.\n",
			compileErrorsExact(t, schemaXML))
	})

	t.Run("duplicate global element", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="E" type="xs:string"/>
  <xs:element name="E" type="xs:int"/>
</xs:schema>`
		require.Equal(t,
			"test.xsd:3: element element: Schemas parser error : Element '{http://www.w3.org/2001/XMLSchema}element': A global element declaration ''E does already exist.\n",
			compileErrorsExact(t, schemaXML))
	})
}
