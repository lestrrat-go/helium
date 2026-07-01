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
	// Per XSD 1.1 every alternative's type must be validly derived from the
	// element's declared type (or xs:error). Holder is the declared type and
	// PosHolder restricts it (value: positiveInteger ⊂ integer), so the type table
	// is valid while still selecting a different governing type by @kind.
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Holder">
    <xs:sequence/>
    <xs:attribute name="kind" type="xs:string"/>
    <xs:attribute name="value" type="xs:integer"/>
  </xs:complexType>
  <xs:complexType name="PosHolder">
    <xs:complexContent>
      <xs:restriction base="Holder">
        <xs:sequence/>
        <xs:attribute name="kind" type="xs:string"/>
        <xs:attribute name="value" type="xs:positiveInteger"/>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="Holder">
    <xs:alternative test="@kind='pos'" type="PosHolder"/>
    <xs:alternative test="@kind='any'" type="Holder"/>
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

	t.Run("kind=pos selects PosHolder", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, validate(t, v11(t), `<root kind="pos" value="5"/>`))
	})

	t.Run("kind=pos rejects non-positive value", func(t *testing.T) {
		t.Parallel()
		require.ErrorIs(t, validate(t, v11(t), `<root kind="pos" value="-5"/>`), xsd.ErrValidationFailed)
	})

	t.Run("kind=any selects declared Holder", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, validate(t, v11(t), `<root kind="any" value="-5"/>`))
	})

	t.Run("no match falls back to declared Holder", func(t *testing.T) {
		t.Parallel()
		// Holder's value is xs:integer, so -5 is accepted under the fallback; the
		// same value would be rejected by PosHolder (see kind=pos above).
		require.NoError(t, validate(t, v11(t), `<root kind="z" value="-5"/>`))
		require.ErrorIs(t, validate(t, v11(t), `<root kind="z" value="x"/>`), xsd.ErrValidationFailed)
	})

	t.Run("xsi:type takes precedence over CTA", func(t *testing.T) {
		t.Parallel()
		// kind=pos would select PosHolder (rejecting -5), but xsi:type forces Holder.
		instance := `<root kind="pos" value="-5" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="Holder"/>`
		require.NoError(t, validate(t, v11(t), instance))
	})

	t.Run("1.0 ignores xs:alternative (always declared Holder)", func(t *testing.T) {
		t.Parallel()
		schema, err := compile(t, xsd.NewCompiler(), schemaXML)
		require.NoError(t, err)
		// kind=pos would select PosHolder in 1.1 (rejecting -5), but 1.0 uses Holder.
		require.NoError(t, validate(t, schema, `<root kind="pos" value="-5"/>`))
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
