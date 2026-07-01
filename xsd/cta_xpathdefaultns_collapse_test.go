package xsd_test

import (
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestVersion11CTAXPathDefaultNSCollapse covers XSDCTA-861-F1: a schema-level
// @xpathDefaultNamespace is xs:anyURI (whiteSpace=collapse), so a padded value like
// "  ##targetNamespace  " must be collapsed BEFORE keyword resolution. Otherwise it
// is mistaken for a literal URI and an inherited unprefixed @test name test (e.g.
// self::root) fails to select the conditional type — a false ACCEPT.
//
// The schema-level value is exercised at every site that pre-resolves it: the
// top-level root (compile.go), and the included/imported schema roots
// (compile_imports.go). In each case the alternative selects the Pos type (value =
// positiveInteger) only when the inherited default element namespace resolves to the
// target namespace, so value="-5" is rejected.
func TestVersion11CTAXPathDefaultNSCollapse(t *testing.T) {
	const (
		mainXSD = "main.xsd"
		subXSD  = "sub.xsd"
		impXSD  = "imp.xsd"
	)
	validateWith := func(t *testing.T, schema *xsd.Schema, instance string) error {
		t.Helper()
		idoc, perr := helium.NewParser().Parse(t.Context(), []byte(instance))
		require.NoError(t, perr)
		return xsd.NewValidator(schema).Validate(t.Context(), idoc)
	}

	t.Run("top-level root", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:t="urn:T" targetNamespace="urn:T"
    elementFormDefault="qualified" xpathDefaultNamespace="  ##targetNamespace  ">
  <xs:complexType name="Base"><xs:sequence/><xs:attribute name="value" type="xs:integer"/></xs:complexType>
  <xs:complexType name="Pos">
    <xs:complexContent>
      <xs:restriction base="t:Base"><xs:sequence/><xs:attribute name="value" type="xs:positiveInteger"/></xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="t:Base">
    <xs:alternative test="self::root" type="t:Pos"/>
  </xs:element>
</xs:schema>`
		doc, perr := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, perr)
		schema, cerr := xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), doc)
		require.NoError(t, cerr)
		require.ErrorIs(t, validateWith(t, schema, `<root xmlns="urn:T" value="-5"/>`), xsd.ErrValidationFailed)
		require.NoError(t, validateWith(t, schema, `<root xmlns="urn:T" value="5"/>`))
	})

	t.Run("included schema root", func(t *testing.T) {
		t.Parallel()
		fsys := fstest.MapFS{
			mainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:T">
  <xs:include schemaLocation="sub.xsd"/>
</xs:schema>`)},
			subXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:t="urn:T" targetNamespace="urn:T"
    elementFormDefault="qualified" xpathDefaultNamespace="  ##targetNamespace  ">
  <xs:complexType name="Base"><xs:sequence/><xs:attribute name="value" type="xs:integer"/></xs:complexType>
  <xs:complexType name="Pos">
    <xs:complexContent>
      <xs:restriction base="t:Base"><xs:sequence/><xs:attribute name="value" type="xs:positiveInteger"/></xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="t:Base">
    <xs:alternative test="self::root" type="t:Pos"/>
  </xs:element>
</xs:schema>`)},
		}
		data, rerr := fsys.ReadFile(mainXSD)
		require.NoError(t, rerr)
		doc, perr := helium.NewParser().Parse(t.Context(), data)
		require.NoError(t, perr)
		schema, cerr := xsd.NewCompiler().Version(xsd.Version11).Label(mainXSD).FS(fsys).Compile(t.Context(), doc)
		require.NoError(t, cerr)
		require.ErrorIs(t, validateWith(t, schema, `<root xmlns="urn:T" value="-5"/>`), xsd.ErrValidationFailed)
		require.NoError(t, validateWith(t, schema, `<root xmlns="urn:T" value="5"/>`))
	})

	t.Run("imported schema root", func(t *testing.T) {
		t.Parallel()
		fsys := fstest.MapFS{
			mainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:i="urn:I" targetNamespace="urn:M">
  <xs:import namespace="urn:I" schemaLocation="imp.xsd"/>
  <xs:element name="m" type="xs:string"/>
</xs:schema>`)},
			impXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:i="urn:I" targetNamespace="urn:I"
    elementFormDefault="qualified" xpathDefaultNamespace="  ##targetNamespace  ">
  <xs:complexType name="Base"><xs:sequence/><xs:attribute name="value" type="xs:integer"/></xs:complexType>
  <xs:complexType name="Pos">
    <xs:complexContent>
      <xs:restriction base="i:Base"><xs:sequence/><xs:attribute name="value" type="xs:positiveInteger"/></xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="iroot" type="i:Base">
    <xs:alternative test="self::iroot" type="i:Pos"/>
  </xs:element>
</xs:schema>`)},
		}
		data, rerr := fsys.ReadFile(mainXSD)
		require.NoError(t, rerr)
		doc, perr := helium.NewParser().Parse(t.Context(), data)
		require.NoError(t, perr)
		schema, cerr := xsd.NewCompiler().Version(xsd.Version11).Label(mainXSD).FS(fsys).Compile(t.Context(), doc)
		require.NoError(t, cerr)
		require.ErrorIs(t, validateWith(t, schema, `<iroot xmlns="urn:I" value="-5"/>`), xsd.ErrValidationFailed)
		require.NoError(t, validateWith(t, schema, `<iroot xmlns="urn:I" value="5"/>`))
	})
}
