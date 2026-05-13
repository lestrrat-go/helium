package xsd_test

import (
	"errors"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// xsd validation errors must surface as *xsd.ValidationError via errors.As so
// callers can read Filename/Line/Element/AttributeName/Message instead of
// parsing the formatted string. Mirrors the API offered by
// relaxng.ValidationError and schematron.ValidationError.
func TestValidationError_AsTyped(t *testing.T) {
	schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="age" type="xs:int"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	instanceXML := `<root><age>not-an-int</age></root>`

	schemaDOC, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	schema, err := xsd.NewCompiler().Compile(t.Context(), schemaDOC)
	require.NoError(t, err)

	doc, err := helium.NewParser().Parse(t.Context(), []byte(instanceXML))
	require.NoError(t, err)

	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	_ = xsd.NewValidator(schema).ErrorHandler(collector).Validate(t.Context(), doc)

	require.NotEmpty(t, collector.Errors(), "expected at least one validation error")

	var ve *xsd.ValidationError
	var found bool
	for _, e := range collector.Errors() {
		if errors.As(e, &ve) {
			found = true
			break
		}
	}
	require.True(t, found, "expected a *xsd.ValidationError in the collected errors")
	require.Equal(t, "age", ve.Element)
	require.NotEmpty(t, ve.Message)
}
