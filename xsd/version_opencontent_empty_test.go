package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestVersion11OpenContentEmpty covers XSD 1.1 §3.4.2.3.3: xs:openContent on an
// otherwise empty complex type (no particle) makes the type element-only with an
// empty particle plus the open content, so extra wildcard-matched children are
// admitted. A truly empty type with no open content still rejects children.
func TestVersion11OpenContentEmpty(t *testing.T) {
	compile := func(t *testing.T, c xsd.Compiler, s string) *xsd.Schema {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(s))
		require.NoError(t, err)
		schema, err := c.Compile(t.Context(), doc)
		require.NoError(t, err)
		return schema
	}
	validate := func(t *testing.T, schema *xsd.Schema, instance string) error {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(instance))
		require.NoError(t, err)
		return xsd.NewValidator(schema).Validate(t.Context(), doc)
	}

	openOnly := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:openContent mode="interleave">
        <xs:any namespace="##any" processContents="skip"/>
      </xs:openContent>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	empty := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType/>
  </xs:element>
</xs:schema>`

	t.Run("1.1 open-content-only empty type accepts extra child", func(t *testing.T) {
		t.Parallel()
		s := compile(t, xsd.NewCompiler().Version(xsd.Version11), openOnly)
		require.NoError(t, validate(t, s, `<root><extra/></root>`))
		require.NoError(t, validate(t, s, `<root/>`))
	})

	t.Run("1.1 open-content-only empty type rejects non-whitespace text", func(t *testing.T) {
		t.Parallel()
		s := compile(t, xsd.NewCompiler().Version(xsd.Version11), openOnly)
		// The synthesized type is element-only, so character content other than
		// whitespace is not allowed even though extra elements are admitted.
		require.ErrorIs(t, validate(t, s, `<root>text</root>`), xsd.ErrValidationFailed)
		require.ErrorIs(t, validate(t, s, `<root>text<extra/></root>`), xsd.ErrValidationFailed)
		require.NoError(t, validate(t, s, `<root><extra/></root>`))
	})

	t.Run("truly empty type with no open content rejects child", func(t *testing.T) {
		t.Parallel()
		s := compile(t, xsd.NewCompiler().Version(xsd.Version11), empty)
		require.ErrorIs(t, validate(t, s, `<root><extra/></root>`), xsd.ErrValidationFailed)
		require.NoError(t, validate(t, s, `<root/>`))
	})
}
