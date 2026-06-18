package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestDuplicateNamedComponent verifies that two global components sharing the
// same name (simpleType, complexType, model group, or attributeGroup) are
// reported as a fatal schema parser error instead of silently overwriting the
// first definition.
func TestDuplicateNamedComponent(t *testing.T) {
	t.Parallel()

	const wantMsg = "does already exist"

	compileErrors := func(t *testing.T, schemaXML string) string {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, err = xsd.NewCompiler().Label("test.xsd").ErrorHandler(collector).Compile(t.Context(), doc)
		require.NoError(t, err)
		_, errors := partitionCompileErrors(collector.Errors())
		return errors
	}

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
		require.Contains(t, compileErrors(t, schemaXML), wantMsg)
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
		require.Contains(t, compileErrors(t, schemaXML), wantMsg)
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
		require.Contains(t, compileErrors(t, schemaXML), wantMsg)
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
		require.Contains(t, compileErrors(t, schemaXML), wantMsg)
	})

	t.Run("duplicate global attribute", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attribute name="A" type="xs:string"/>
  <xs:attribute name="A" type="xs:int"/>
</xs:schema>`
		require.Contains(t, compileErrors(t, schemaXML), wantMsg)
	})
}
