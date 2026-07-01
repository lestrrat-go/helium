package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestVersion11CTAEmptyXsiTypeScope covers gauntlet finding XSDCTA-861-REV-001: a
// present-but-empty xsi:type="" hard-errors ONLY where it would otherwise suppress
// a CTA-selected type. Everywhere else (XSD 1.0, and 1.1 with no alternative) it
// must still fall back to the declared type, byte-identical to origin.
func TestVersion11CTAEmptyXsiTypeScope(t *testing.T) {
	validateWith := func(t *testing.T, schema *xsd.Schema, instance string) error {
		t.Helper()
		idoc, err := helium.NewParser().Parse(t.Context(), []byte(instance))
		require.NoError(t, err)
		return xsd.NewValidator(schema).Validate(t.Context(), idoc)
	}
	const xsiNS = `xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"`

	// (a) The round-6 guarantee still holds: under 1.1 with an xs:error alternative,
	// an empty xsi:type cannot suppress the CTA-selected xs:error.
	t.Run("empty xsi:type cannot suppress CTA xs:error (1.1)", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="MsgT"><xs:sequence/><xs:attribute name="kind" type="xs:string"/><xs:attribute name="value" type="xs:integer"/></xs:complexType>
  <xs:element name="e" type="MsgT">
    <xs:alternative test="@kind='bad'" type="xs:error"/>
  </xs:element>
</xs:schema>`
		doc, err := helium.NewParser().Parse(t.Context(), []byte(s))
		require.NoError(t, err)
		schema, err := xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), doc)
		require.NoError(t, err)
		require.ErrorIs(t, validateWith(t, schema, `<e `+xsiNS+` kind="bad" value="1" xsi:type=""/>`), xsd.ErrValidationFailed)
	})

	// (b) Under XSD 1.0, a present empty xsi:type="" still falls back to the declared
	// type (matches origin/feat-xsd11) — no new error.
	t.Run("empty xsi:type falls back to declared type (XSD 1.0)", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="MsgT"><xs:sequence/><xs:attribute name="kind" type="xs:string"/><xs:attribute name="value" type="xs:integer"/></xs:complexType>
  <xs:element name="e" type="MsgT"/>
</xs:schema>`
		doc, err := helium.NewParser().Parse(t.Context(), []byte(s))
		require.NoError(t, err)
		schema, err := xsd.NewCompiler().Compile(t.Context(), doc) // default = XSD 1.0
		require.NoError(t, err)
		require.NoError(t, validateWith(t, schema, `<e `+xsiNS+` kind="ok" value="1" xsi:type=""/>`))
	})

	// (c) Under XSD 1.1 with NO alternative, a present empty xsi:type="" also falls
	// back to the declared type — the new error must not fire when CTA isn't involved.
	t.Run("empty xsi:type falls back to declared type (1.1, no alternative)", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="MsgT"><xs:sequence/><xs:attribute name="kind" type="xs:string"/><xs:attribute name="value" type="xs:integer"/></xs:complexType>
  <xs:element name="e" type="MsgT"/>
</xs:schema>`
		doc, err := helium.NewParser().Parse(t.Context(), []byte(s))
		require.NoError(t, err)
		schema, err := xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), doc)
		require.NoError(t, err)
		require.NoError(t, validateWith(t, schema, `<e `+xsiNS+` kind="ok" value="1" xsi:type=""/>`))
	})
}
