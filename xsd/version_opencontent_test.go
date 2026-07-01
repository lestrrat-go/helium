package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestVersion11OpenContent covers XSD 1.1 xs:openContent in interleave and
// suffix modes, and that 1.0 ignores it.
func TestVersion11OpenContent(t *testing.T) {
	schemaFor := func(mode string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:openContent mode="` + mode + `">
        <xs:any namespace="##any" processContents="skip"/>
      </xs:openContent>
      <xs:sequence>
        <xs:element name="a" type="xs:string"/>
        <xs:element name="b" type="xs:string"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	}

	compile := func(t *testing.T, c xsd.Compiler, s string) (*xsd.Schema, error) {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(s))
		require.NoError(t, err)
		return c.Compile(t.Context(), doc)
	}
	validate := func(t *testing.T, schema *xsd.Schema, instance string) error {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(instance))
		require.NoError(t, err)
		return xsd.NewValidator(schema).Validate(t.Context(), doc)
	}
	v11 := func(t *testing.T, mode string) *xsd.Schema {
		t.Helper()
		schema, err := compile(t, xsd.NewCompiler().Version(xsd.Version11), schemaFor(mode))
		require.NoError(t, err)
		return schema
	}

	t.Run("interleave: open element between declared", func(t *testing.T) {
		t.Parallel()
		s := v11(t, "interleave")
		require.NoError(t, validate(t, s, `<root><a>1</a><x/><b>2</b></root>`))
		require.NoError(t, validate(t, s, `<root><a>1</a><b>2</b></root>`))
	})

	t.Run("interleave: still requires declared content", func(t *testing.T) {
		t.Parallel()
		s := v11(t, "interleave")
		require.ErrorIs(t, validate(t, s, `<root><x/></root>`), xsd.ErrValidationFailed)
	})

	t.Run("suffix: open element only after declared", func(t *testing.T) {
		t.Parallel()
		s := v11(t, "suffix")
		require.NoError(t, validate(t, s, `<root><a>1</a><b>2</b><x/></root>`))
	})

	t.Run("suffix: open element interspersed is rejected", func(t *testing.T) {
		t.Parallel()
		s := v11(t, "suffix")
		require.ErrorIs(t, validate(t, s, `<root><a>1</a><x/><b>2</b></root>`), xsd.ErrValidationFailed)
	})

	t.Run("1.0 ignores openContent (extra element rejected)", func(t *testing.T) {
		t.Parallel()
		schema, err := compile(t, xsd.NewCompiler(), schemaFor("interleave"))
		require.NoError(t, err)
		require.ErrorIs(t, validate(t, schema, `<root><a>1</a><x/><b>2</b></root>`), xsd.ErrValidationFailed)
	})
}
