package xsd_test

import (
	"context"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// closableHandler is a closable ErrorHandler used to assert that Validate
// closes the configured handler on every exit path, including the nil guards.
type closableHandler struct {
	closed bool
}

func (h *closableHandler) Handle(context.Context, error) {}

func (h *closableHandler) Close() error {
	h.closed = true
	return nil
}

func TestValidateNilSchema(t *testing.T) {
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root/>`))
	require.NoError(t, err, "parse should succeed")

	h := &closableHandler{}
	err = xsd.NewValidator(nil).ErrorHandler(h).Validate(t.Context(), doc)
	require.Error(t, err, "Validate with nil schema should return an error, not panic")
	require.ErrorIs(t, err, xsd.ErrNilSchema, "should return ErrNilSchema")
	require.True(t, h.closed, "handler should be closed even on the nil-schema path")
}

func TestValidateNilDocument(t *testing.T) {
	schema, err := xsd.NewCompiler().Compile(t.Context(), mustParseXSD(t, `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="xs:string"/>
</xs:schema>`))
	require.NoError(t, err, "compile should succeed")

	h := &closableHandler{}
	err = xsd.NewValidator(schema).ErrorHandler(h).Validate(t.Context(), nil)
	require.Error(t, err, "Validate with nil document should return an error, not panic")
	require.ErrorIs(t, err, xsd.ErrNilDocument, "should return ErrNilDocument")
	require.True(t, h.closed, "handler should be closed even on the nil-document path")
}

// TestValidateNilContext asserts that Validate normalizes a nil context.Context
// rather than panicking while evaluating an identity-constraint XPath.
func TestValidateNilContext(t *testing.T) {
	const schemaSrc = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" maxOccurs="unbounded">
          <xs:complexType>
            <xs:attribute name="n" type="xs:integer"/>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
    <xs:unique name="itemKey">
      <xs:selector xpath="item"/>
      <xs:field xpath="@n"/>
    </xs:unique>
  </xs:element>
</xs:schema>`

	schema, err := xsd.NewCompiler().Compile(t.Context(), mustParseXSD(t, schemaSrc))
	require.NoError(t, err, "compile should succeed")

	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><item n="1"/><item n="2"/></root>`))
	require.NoError(t, err, "parse instance should succeed")

	var nilCtx context.Context
	require.NotPanics(t, func() {
		err = xsd.NewValidator(schema).Validate(nilCtx, doc)
	}, "Validate with nil context must not panic")
	require.NoError(t, err, "valid document should pass with a nil context")
}

func mustParseXSD(t *testing.T, src string) *helium.Document {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err, "parse XSD should succeed")
	return doc
}
