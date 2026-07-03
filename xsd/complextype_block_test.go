package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestComplexTypeBlockXsiType covers gauntlet finding complextype-block: a
// complexType's @block ({prohibited substitutions}) must participate in the
// cvc-elt.4.3 xsi:type derivation-block check. Per the spec the blocked set is the
// UNION of the element declaration's {disallowed substitutions} and the DECLARED
// type's {prohibited substitutions}, so an xsi:type derivation blocked by the
// TYPE's @block is rejected even when the element declaration itself carries no
// block. Version-independent (runs in both XSD 1.0 and 1.1).
func TestComplexTypeBlockXsiType(t *testing.T) {
	const xsiNS = `xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"`

	// base has block on the TYPE (not the element); derived extends base. An element
	// declared with type "base" and an instance xsi:type="derived" must be rejected
	// when base blocks extension, accepted when it does not.
	schemaFor := func(block string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
                  xmlns:t="urn:t" targetNamespace="urn:t"
                  elementFormDefault="qualified">
  <xs:complexType name="base" ` + block + `>
    <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="derived">
    <xs:complexContent>
      <xs:extension base="t:base">
        <xs:sequence><xs:element name="b" type="xs:string"/></xs:sequence>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="e" type="t:base"/>
</xs:schema>`
	}

	instance := `<t:e xmlns:t="urn:t" ` + xsiNS + ` xsi:type="t:derived">` +
		`<t:a>x</t:a><t:b>y</t:b></t:e>`

	compileValidate := func(t *testing.T, v xsd.Version, block string) error {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaFor(block)))
		require.NoError(t, err)
		sc, err := xsd.NewCompiler().Version(v).Compile(t.Context(), doc)
		require.NoError(t, err)
		idoc, err := helium.NewParser().Parse(t.Context(), []byte(instance))
		require.NoError(t, err)
		return xsd.NewValidator(sc).Validate(t.Context(), idoc)
	}

	for _, v := range []struct {
		name string
		ver  xsd.Version
	}{{"xsd10", xsd.Version10}, {"xsd11", xsd.Version11}} {
		t.Run(v.name, func(t *testing.T) {
			t.Run("type block=extension rejects xsi:type extension", func(t *testing.T) {
				require.ErrorIs(t, compileValidate(t, v.ver, `block="extension"`), xsd.ErrValidationFailed)
			})
			t.Run("type block=#all rejects xsi:type extension", func(t *testing.T) {
				require.ErrorIs(t, compileValidate(t, v.ver, `block="#all"`), xsd.ErrValidationFailed)
			})
			t.Run("no type block accepts xsi:type extension", func(t *testing.T) {
				require.NoError(t, compileValidate(t, v.ver, ``))
			})
			t.Run("type block=restriction (unrelated) accepts xsi:type extension", func(t *testing.T) {
				// The derivation is by extension, so a restriction-only block does not apply.
				require.NoError(t, compileValidate(t, v.ver, `block="restriction"`))
			})
		})
	}
}
