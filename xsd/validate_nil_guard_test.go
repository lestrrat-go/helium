package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

func TestValidateNilSchema(t *testing.T) {
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root/>`))
	require.NoError(t, err, "parse should succeed")

	err = xsd.NewValidator(nil).Validate(t.Context(), doc)
	require.Error(t, err, "Validate with nil schema should return an error, not panic")
	require.ErrorIs(t, err, xsd.ErrNilSchema, "should return ErrNilSchema")
}

func TestValidateNilDocument(t *testing.T) {
	schema, err := xsd.NewCompiler().Compile(t.Context(), mustParseXSD(t, `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="xs:string"/>
</xs:schema>`))
	require.NoError(t, err, "compile should succeed")

	err = xsd.NewValidator(schema).Validate(t.Context(), nil)
	require.Error(t, err, "Validate with nil document should return an error, not panic")
	require.ErrorIs(t, err, xsd.ErrNilDocument, "should return ErrNilDocument")
}

func mustParseXSD(t *testing.T, src string) *helium.Document {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err, "parse XSD should succeed")
	return doc
}
