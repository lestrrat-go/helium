package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestVersion11CTAReviewR6 covers the round-6 gauntlet finding: a present-but-empty
// xsi:type="" must not silently fall back to the declared type (which would let it
// suppress a CTA-selected xs:error type), but report a validity error.
func TestVersion11CTAReviewR6(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="MsgT"><xs:sequence/><xs:attribute name="kind" type="xs:string"/><xs:attribute name="value" type="xs:integer"/></xs:complexType>
  <xs:element name="e" type="MsgT">
    <xs:alternative test="@kind='bad'" type="xs:error"/>
  </xs:element>
</xs:schema>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	schema, err := xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), doc)
	require.NoError(t, err)

	validate := func(t *testing.T, instance string) error {
		t.Helper()
		idoc, perr := helium.NewParser().Parse(t.Context(), []byte(instance))
		require.NoError(t, perr)
		return xsd.NewValidator(schema).Validate(t.Context(), idoc)
	}

	const xsiNS = `xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"`

	t.Run("kind=bad selects xs:error (invalid)", func(t *testing.T) {
		t.Parallel()
		require.ErrorIs(t, validate(t, `<e kind="bad" value="1"/>`), xsd.ErrValidationFailed)
	})

	t.Run("empty xsi:type cannot suppress xs:error", func(t *testing.T) {
		t.Parallel()
		// A malformed empty xsi:type must NOT fall back to the declared MsgT and
		// validate; the element remains invalid (CTA selected xs:error, and the empty
		// xsi:type is itself an unresolved QName).
		require.ErrorIs(t, validate(t, `<e `+xsiNS+` kind="bad" value="1" xsi:type=""/>`), xsd.ErrValidationFailed)
	})

	t.Run("whitespace-only xsi:type cannot suppress xs:error", func(t *testing.T) {
		t.Parallel()
		require.ErrorIs(t, validate(t, `<e `+xsiNS+` kind="bad" value="1" xsi:type="   "/>`), xsd.ErrValidationFailed)
	})

	t.Run("absent xsi:type CTA still works", func(t *testing.T) {
		t.Parallel()
		// kind!=bad: the declared MsgT governs and the instance is valid.
		require.NoError(t, validate(t, `<e kind="ok" value="1"/>`))
	})

	t.Run("valid xsi:type still overrides CTA", func(t *testing.T) {
		t.Parallel()
		// A well-formed xsi:type naming the declared type suppresses CTA as before.
		require.NoError(t, validate(t, `<e `+xsiNS+` kind="bad" value="1" xsi:type="MsgT"/>`))
	})
}
