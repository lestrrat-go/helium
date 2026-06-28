package xsd_test

import (
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestVersion11CTATypeQNameHandling covers the gauntlet findings on xs:alternative
// @type resolution: the value is whitespace-collapsed (xs:QName) before QName
// resolution, and an unprefixed ref participates in the chameleon/no-targetNamespace
// fallback like an ordinary element @type ref.
func TestVersion11CTATypeQNameHandling(t *testing.T) {
	// CTA-861-001: a valid type=" xs:int " (surrounding whitespace) must compile and
	// govern, because xs:QName collapses whitespace before resolution.
	t.Run("whitespace-padded @type compiles and governs", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e" type="xs:integer">
    <xs:alternative test="true()" type=" xs:int "/>
  </xs:element>
</xs:schema>`
		doc, err := helium.NewParser().Parse(t.Context(), []byte(s))
		require.NoError(t, err)
		schema, err := xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), doc)
		require.NoError(t, err)
		validate := func(instance string) error {
			idoc, perr := helium.NewParser().Parse(t.Context(), []byte(instance))
			require.NoError(t, perr)
			return xsd.NewValidator(schema).Validate(t.Context(), idoc)
		}
		// The collapsed xs:int alternative governs: a value outside int range is
		// rejected, an in-range value accepted.
		require.ErrorIs(t, validate(`<e>99999999999</e>`), xsd.ErrValidationFailed)
		require.NoError(t, validate(`<e>5</e>`))
	})

	// CTA-861-002: a CTA @type ref to a type from a no-targetNamespace (chameleon)
	// imported schema must resolve via the {} fallback, exactly like an ordinary
	// element @type ref.
	t.Run("CTA @type ref resolves a chameleon/no-TNS imported type", func(t *testing.T) {
		t.Parallel()
		fsys := fstest.MapFS{
			// Both the element's declared @type and the CTA @type are unprefixed refs
			// to no-TNS imported types, resolved via the chameleon ({}) fallback. The
			// CTA path must perform that fallback just like the ordinary @type path.
			importMainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t">
  <xs:import schemaLocation="cham.xsd"/>
  <xs:element name="e" type="Holder">
    <xs:alternative test="true()" type="PosHolder"/>
  </xs:element>
</xs:schema>`)},
			// No targetNamespace: a chameleon schema. Its PosHolder/Holder must be
			// reachable from main's unprefixed refs via the {} fallback.
			"cham.xsd": &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Holder"><xs:sequence/><xs:attribute name="value" type="xs:integer"/></xs:complexType>
  <xs:complexType name="PosHolder">
    <xs:complexContent>
      <xs:restriction base="Holder"><xs:sequence/><xs:attribute name="value" type="xs:positiveInteger"/></xs:restriction>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`)},
		}
		data, err := fsys.ReadFile(importMainXSD)
		require.NoError(t, err)
		doc, err := helium.NewParser().Parse(t.Context(), data)
		require.NoError(t, err)
		schema, err := xsd.NewCompiler().Version(xsd.Version11).FS(fsys).Compile(t.Context(), doc)
		require.NoError(t, err)
		validate := func(instance string) error {
			idoc, perr := helium.NewParser().Parse(t.Context(), []byte(instance))
			require.NoError(t, perr)
			return xsd.NewValidator(schema).Validate(t.Context(), idoc)
		}
		// The alternative selects the {}PosHolder (positiveInteger): -5 is rejected.
		require.ErrorIs(t, validate(`<e xmlns="urn:t" value="-5"/>`), xsd.ErrValidationFailed)
		require.NoError(t, validate(`<e xmlns="urn:t" value="5"/>`))
	})
}
