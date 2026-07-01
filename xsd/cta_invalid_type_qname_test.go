package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestVersion11CTAInvalidTypeQName covers CTA-REVIEW-002: an xs:alternative @type
// whose collapsed value is a malformed QName (e.g. a leading colon ":T") must be a
// fatal schema error, not silently resolved (which, when a {}T type happens to
// exist, would compile and wrongly select it). Mirrors idc's @ref/@refer
// validateQName-before-resolution handling.
func TestVersion11CTAInvalidTypeQName(t *testing.T) {
	// A no-targetNamespace schema with a {}T complex type that IS substitutable for
	// the declared type, so the only thing that can reject type=":T" is QName
	// validation (not the substitutability/derivation check).
	schemaWith := func(typeRef string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base">
    <xs:sequence/>
    <xs:attribute name="k" type="xs:string"/>
    <xs:attribute name="value" type="xs:integer"/>
  </xs:complexType>
  <xs:complexType name="T">
    <xs:complexContent>
      <xs:restriction base="Base">
        <xs:sequence/>
        <xs:attribute name="k" type="xs:string"/>
        <xs:attribute name="value" type="xs:integer"/>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="e" type="Base">
    <xs:alternative test="@k='x'" type="` + typeRef + `"/>
  </xs:element>
</xs:schema>`
	}

	t.Run("leading-colon type is rejected", func(t *testing.T) {
		t.Parallel()
		doc, perr := helium.NewParser().Parse(t.Context(), []byte(schemaWith(":T")))
		require.NoError(t, perr)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, err := xsd.NewCompiler().Version(xsd.Version11).ErrorHandler(collector).Compile(t.Context(), doc)
		requireCompileResultErr(t, err)
		require.Error(t, err, "type=\":T\" is a malformed QName and must fail compilation")
		require.NoError(t, collector.Close())
		_, errStr := partitionCompileErrors(collector.Errors())
		require.Contains(t, errStr, "is not a valid QName", "got: %q", errStr)
	})

	t.Run("control: valid type T compiles", func(t *testing.T) {
		t.Parallel()
		doc, perr := helium.NewParser().Parse(t.Context(), []byte(schemaWith("T")))
		require.NoError(t, perr)
		_, err := xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), doc)
		require.NoError(t, err, "the same schema with a well-formed @type must compile")
	})
}
