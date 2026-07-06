package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestUnresolvedTypeRef verifies that unresolved type references fail in the
// right phase. Base and union-member references are fatal schema errors. Element
// type and list item type references may sit on unused declarations, but fail
// validation when the offending declaration/type is selected.
func TestUnresolvedTypeRef(t *testing.T) {
	t.Parallel()

	const wantMsg = "does not resolve to a(n) type definition"

	compileErrors := func(t *testing.T, schemaXML string) string {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, err = xsd.NewCompiler().Label("test.xsd").ErrorHandler(collector).Compile(t.Context(), doc)
		requireCompileResultErr(t, err)
		_, errors := partitionCompileErrors(collector.Errors())
		return errors
	}
	compileOK := func(t *testing.T, schemaXML string) *xsd.Schema {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		schema, err := xsd.NewCompiler().Label("test.xsd").ErrorHandler(collector).Compile(t.Context(), doc)
		require.NoError(t, err)
		require.NoError(t, collector.Close())
		_, errors := partitionCompileErrors(collector.Errors())
		require.Empty(t, errors)
		return schema
	}
	validateXML := func(t *testing.T, schema *xsd.Schema, xml string) error {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
		require.NoError(t, err)
		return xsd.NewValidator(schema).Validate(t.Context(), doc)
	}
	validateXMLWithOutput := func(t *testing.T, schema *xsd.Schema, xml string) (string, error) {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
		require.NoError(t, err)
		var out string
		err = validateWithOutput(t, xsd.NewValidator(schema), doc, &out)
		return out, err
	}

	t.Run("missing base type", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="derived">
    <xs:restriction base="MissingBase"/>
  </xs:simpleType>
  <xs:element name="root" type="derived"/>
</xs:schema>`
		require.Contains(t, compileErrors(t, schemaXML), wantMsg)
	})

	t.Run("missing list item type is deferred until validation", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="myList">
    <xs:list itemType="MissingItem"/>
  </xs:simpleType>
  <xs:element name="good" type="xs:integer"/>
  <xs:element name="bad" type="myList"/>
</xs:schema>`
		schema := compileOK(t, schemaXML)
		require.NoError(t, validateXML(t, schema, `<good>1</good>`))
		require.ErrorIs(t, validateXML(t, schema, `<bad>1</bad>`), xsd.ErrValidationFailed)
	})

	t.Run("missing union member type", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="myUnion">
    <xs:union memberTypes="xs:string MissingMember"/>
  </xs:simpleType>
  <xs:element name="root" type="myUnion"/>
</xs:schema>`
		require.Contains(t, compileErrors(t, schemaXML), wantMsg)
	})

	t.Run("missing element type is deferred until validation", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="good" type="xs:integer"/>
  <xs:element name="bad" type="MissingType"/>
</xs:schema>`
		schema := compileOK(t, schemaXML)
		require.NoError(t, validateXML(t, schema, `<good>1</good>`))
		require.ErrorIs(t, validateXML(t, schema, `<bad>1</bad>`), xsd.ErrValidationFailed)
	})

	t.Run("missing element type and substitution head are deferred until validation", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="good" type="xs:integer"/>
  <xs:element name="bad" type="MissingType" substitutionGroup="MissingHead"/>
</xs:schema>`
		schema := compileOK(t, schemaXML)
		require.NoError(t, validateXML(t, schema, `<good>1</good>`))
		require.ErrorIs(t, validateXML(t, schema, `<bad>1</bad>`), xsd.ErrValidationFailed)
	})

	t.Run("missing nested element type rejects before xsi type", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="bad" type="MissingType"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		schema := compileOK(t, schemaXML)
		out, err := validateXMLWithOutput(t, schema, `<root xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xs="http://www.w3.org/2001/XMLSchema"><bad xsi:type="xs:string">ok</bad></root>`)
		require.ErrorIs(t, err, xsd.ErrValidationFailed)
		require.Contains(t, out, wantMsg)
		require.NotContains(t, out, "xsi:type definition")
	})

	t.Run("missing anyType child element type rejects before xsi type", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="xs:anyType"/>
  <xs:element name="bad" type="MissingType"/>
</xs:schema>`
		schema := compileOK(t, schemaXML)
		out, err := validateXMLWithOutput(t, schema, `<root xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xs="http://www.w3.org/2001/XMLSchema"><bad xsi:type="xs:string">ok</bad></root>`)
		require.ErrorIs(t, err, xsd.ErrValidationFailed)
		require.Contains(t, out, wantMsg)
		require.NotContains(t, out, "xsi:type definition")
	})

	// Inline (local) simpleTypes follow the same phase split as named types:
	// unresolved bases and union members are compile-fatal, while an unresolved
	// list item type is deferred until the inline list type validates a value.
	t.Run("missing base type in inline simpleType", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:simpleType>
      <xs:restriction base="MissingBase"/>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
		require.Contains(t, compileErrors(t, schemaXML), wantMsg)
	})

	t.Run("missing list item type in inline simpleType is deferred until validation", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:simpleType>
      <xs:list itemType="MissingItem"/>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
		schema := compileOK(t, schemaXML)
		require.ErrorIs(t, validateXML(t, schema, `<root>1</root>`), xsd.ErrValidationFailed)
	})

	t.Run("missing union member type in inline simpleType", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:simpleType>
      <xs:union memberTypes="xs:string MissingMember"/>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
		require.Contains(t, compileErrors(t, schemaXML), wantMsg)
	})
}
