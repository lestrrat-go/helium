package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestVersion11UPAWeakening verifies that in XSD 1.1 an element particle
// competing with a wildcard is not a UPA (cos-nonambig) violation (the element
// wins), while in XSD 1.0 the same content model is rejected as ambiguous.
func TestVersion11UPAWeakening(t *testing.T) {
	// A choice of an element and a wildcard that admits the element's namespace:
	// ambiguous under 1.0, allowed under 1.1.
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:choice maxOccurs="unbounded">
        <xs:element name="a" type="xs:string"/>
        <xs:any processContents="lax"/>
      </xs:choice>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	compile := func(t *testing.T, c xsd.Compiler) (*xsd.Schema, error) {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		return c.Compile(t.Context(), doc)
	}

	t.Run("1.0 rejects element-vs-wildcard as non-deterministic", func(t *testing.T) {
		t.Parallel()
		_, err := compile(t, xsd.NewCompiler())
		require.ErrorIs(t, err, xsd.ErrCompilationFailed)
	})

	t.Run("1.1 accepts the content model and validates", func(t *testing.T) {
		t.Parallel()
		schema, err := compile(t, xsd.NewCompiler().Version(xsd.Version11))
		require.NoError(t, err)

		// The declared element matches the element particle; an unknown element is
		// admitted by the lax wildcard.
		instance := `<root><a>hi</a><other/></root>`
		doc, err := helium.NewParser().Parse(t.Context(), []byte(instance))
		require.NoError(t, err)
		require.NoError(t, xsd.NewValidator(schema).Validate(t.Context(), doc))
	})
}
