package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestVersion11ConditionalTypeAssignment covers XSD 1.1 xs:alternative: the
// governing type is chosen by the first matching @test, falling back to the
// declared type when none match; xsi:type takes precedence; 1.0 ignores it.
func TestVersion11ConditionalTypeAssignment(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="TypeA">
    <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
    <xs:attribute name="kind" type="xs:string"/>
  </xs:complexType>
  <xs:complexType name="TypeB">
    <xs:sequence><xs:element name="b" type="xs:string"/></xs:sequence>
    <xs:attribute name="kind" type="xs:string"/>
  </xs:complexType>
  <xs:element name="root" type="TypeA">
    <xs:alternative test="@kind='b'" type="TypeB"/>
    <xs:alternative test="@kind='a'" type="TypeA"/>
  </xs:element>
</xs:schema>`

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

	v11 := func(t *testing.T) *xsd.Schema {
		t.Helper()
		schema, err := compile(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
		require.NoError(t, err)
		return schema
	}

	t.Run("kind=b selects TypeB", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, validate(t, v11(t), `<root kind="b"><b>x</b></root>`))
	})

	t.Run("kind=b rejects TypeA content", func(t *testing.T) {
		t.Parallel()
		require.ErrorIs(t, validate(t, v11(t), `<root kind="b"><a>x</a></root>`), xsd.ErrValidationFailed)
	})

	t.Run("kind=a selects TypeA", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, validate(t, v11(t), `<root kind="a"><a>x</a></root>`))
	})

	t.Run("no match falls back to declared TypeA", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, validate(t, v11(t), `<root kind="z"><a>x</a></root>`))
		require.ErrorIs(t, validate(t, v11(t), `<root kind="z"><b>x</b></root>`), xsd.ErrValidationFailed)
	})

	t.Run("xsi:type takes precedence over CTA", func(t *testing.T) {
		t.Parallel()
		// kind=b would select TypeB, but xsi:type forces TypeA.
		instance := `<root kind="b" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="TypeA"><a>x</a></root>`
		require.NoError(t, validate(t, v11(t), instance))
	})

	t.Run("1.0 ignores xs:alternative (always declared TypeA)", func(t *testing.T) {
		t.Parallel()
		schema, err := compile(t, xsd.NewCompiler(), schemaXML)
		require.NoError(t, err)
		// kind=b would select TypeB in 1.1, but 1.0 uses the declared TypeA.
		require.NoError(t, validate(t, schema, `<root kind="b"><a>x</a></root>`))
		require.ErrorIs(t, validate(t, schema, `<root kind="b"><b>x</b></root>`), xsd.ErrValidationFailed)
	})

	t.Run("1.1 malformed alternative test is a compile error", func(t *testing.T) {
		t.Parallel()
		const bad = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="T"><xs:sequence/></xs:complexType>
  <xs:element name="e" type="T">
    <xs:alternative test="@x +" type="T"/>
  </xs:element>
</xs:schema>`
		_, err := compile(t, xsd.NewCompiler().Version(xsd.Version11), bad)
		require.ErrorIs(t, err, xsd.ErrCompilationFailed)
	})
}
