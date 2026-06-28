package xsd_test

import (
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestVersion11CTAReviewR2 covers the round-2 gauntlet findings: the synthetic CTA
// context node's local name for prefixed elements, substitutability against a
// substitution-group member's effective (inherited) declared type, and the
// per-document CTA static context for included/imported schemas.
func TestVersion11CTAReviewR2(t *testing.T) {
	compile := func(t *testing.T, s string) (*xsd.Schema, error) {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(s))
		require.NoError(t, err)
		return xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), doc)
	}
	validate := func(t *testing.T, schema *xsd.Schema, instance string) error {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(instance))
		require.NoError(t, err)
		return xsd.NewValidator(schema).Validate(t.Context(), doc)
	}
	compileFS := func(t *testing.T, fsys fstest.MapFS, main string) (*xsd.Schema, error) {
		t.Helper()
		data, err := fsys.ReadFile(main)
		require.NoError(t, err)
		doc, err := helium.NewParser().Parse(t.Context(), data)
		require.NoError(t, err)
		return xsd.NewCompiler().Version(xsd.Version11).FS(fsys).Compile(t.Context(), doc)
	}

	// XSDCTA-R2-001: a name test in a CTA @test must match a PREFIXED instance
	// element. The synthetic context node must carry the correct local name.
	t.Run("prefixed instance element name test", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="urn:t" xmlns:t="urn:t" elementFormDefault="qualified">
  <xs:complexType name="Holder"><xs:sequence/><xs:attribute name="value" type="xs:integer"/></xs:complexType>
  <xs:complexType name="PosHolder">
    <xs:complexContent>
      <xs:restriction base="t:Holder"><xs:sequence/><xs:attribute name="value" type="xs:positiveInteger"/></xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="t:Holder">
    <xs:alternative test="self::root" type="t:PosHolder" xpathDefaultNamespace="##targetNamespace"/>
  </xs:element>
</xs:schema>`
		schema, err := compile(t, s)
		require.NoError(t, err)
		// Prefixed instance: self::root must match {urn:t}root → PosHolder governs.
		require.ErrorIs(t, validate(t, schema, `<t:root xmlns:t="urn:t" value="-5"/>`), xsd.ErrValidationFailed)
		require.NoError(t, validate(t, schema, `<t:root xmlns:t="urn:t" value="5"/>`))
	})

	// XSDCTA-R2-002: a substitution-group member with no explicit type inherits the
	// head's (complex) type; a simple alternative against it must be rejected.
	t.Run("substitution-group member alternative substitutability", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="urn:t" xmlns:t="urn:t" elementFormDefault="qualified">
  <xs:complexType name="Head"><xs:sequence><xs:element name="x" type="xs:string"/></xs:sequence></xs:complexType>
  <xs:element name="head" type="t:Head"/>
  <xs:element name="member" substitutionGroup="t:head">
    <xs:alternative test="@k='s'" type="xs:string"/>
  </xs:element>
</xs:schema>`
		// member's effective declared type is the complex Head; the simple xs:string
		// alternative is not validly substitutable for it.
		_, err := compile(t, s)
		require.ErrorIs(t, err, xsd.ErrCompilationFailed)
	})

	// XSDCTA-R2-003a: an included file's CTA must see the INCLUDED document's static
	// base URI (fn:static-base-uri).
	t.Run("included file CTA static-base-uri", func(t *testing.T) {
		t.Parallel()
		fsys := fstest.MapFS{
			importMainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:t="urn:t" targetNamespace="urn:t">
  <xs:include schemaLocation="inc.xsd"/>
</xs:schema>`)},
			"inc.xsd": &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:t="urn:t" targetNamespace="urn:t">
  <xs:complexType name="Holder"><xs:sequence/><xs:attribute name="value" type="xs:integer"/></xs:complexType>
  <xs:complexType name="PosHolder">
    <xs:complexContent>
      <xs:restriction base="t:Holder"><xs:sequence/><xs:attribute name="value" type="xs:positiveInteger"/></xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="e" type="t:Holder">
    <xs:alternative test="ends-with(static-base-uri(), 'inc.xsd')" type="t:PosHolder"/>
  </xs:element>
</xs:schema>`)},
		}
		schema, err := compileFS(t, fsys, importMainXSD)
		require.NoError(t, err)
		// static-base-uri ends with inc.xsd → PosHolder governs → -5 invalid.
		require.ErrorIs(t, validate(t, schema, `<e xmlns="urn:t" value="-5"/>`), xsd.ErrValidationFailed)
		require.NoError(t, validate(t, schema, `<e xmlns="urn:t" value="5"/>`))
	})

	// XSDCTA-R2-003b: an included file's schema-level xpathDefaultNamespace must
	// apply to that file's CTA tests, not the including schema's.
	t.Run("included file CTA xpathDefaultNamespace", func(t *testing.T) {
		t.Parallel()
		fsys := fstest.MapFS{
			importMainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:t="urn:t" targetNamespace="urn:t">
  <xs:include schemaLocation="inc.xsd"/>
</xs:schema>`)},
			"inc.xsd": &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:t="urn:t" targetNamespace="urn:t"
  elementFormDefault="qualified" xpathDefaultNamespace="##targetNamespace">
  <xs:complexType name="Holder"><xs:sequence/><xs:attribute name="value" type="xs:integer"/></xs:complexType>
  <xs:complexType name="PosHolder">
    <xs:complexContent>
      <xs:restriction base="t:Holder"><xs:sequence/><xs:attribute name="value" type="xs:positiveInteger"/></xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="e" type="t:Holder">
    <xs:alternative test="self::e" type="t:PosHolder"/>
  </xs:element>
</xs:schema>`)},
		}
		schema, err := compileFS(t, fsys, importMainXSD)
		require.NoError(t, err)
		// inc's xpathDefaultNamespace=##targetNamespace makes self::e match {urn:t}e.
		require.ErrorIs(t, validate(t, schema, `<e xmlns="urn:t" value="-5"/>`), xsd.ErrValidationFailed)
		require.NoError(t, validate(t, schema, `<e xmlns="urn:t" value="5"/>`))
	})

	// XSDCTA-R2-004: an imported element's CTA must govern, with its named alternative
	// @type ref resolved against the merged (imported) type table.
	t.Run("imported CTA governs and named alt ref resolves", func(t *testing.T) {
		t.Parallel()
		fsys := fstest.MapFS{
			importMainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:i="urn:i" targetNamespace="urn:m">
  <xs:import namespace="urn:i" schemaLocation="imp.xsd"/>
  <xs:element name="wrap" type="xs:string"/>
</xs:schema>`)},
			"imp.xsd": &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:i="urn:i" targetNamespace="urn:i">
  <xs:complexType name="Holder"><xs:sequence/><xs:attribute name="value" type="xs:integer"/></xs:complexType>
  <xs:complexType name="PosHolder">
    <xs:complexContent>
      <xs:restriction base="i:Holder"><xs:sequence/><xs:attribute name="value" type="xs:positiveInteger"/></xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="e" type="i:Holder">
    <xs:alternative test="true()" type="i:PosHolder"/>
  </xs:element>
</xs:schema>`)},
		}
		schema, err := compileFS(t, fsys, importMainXSD)
		require.NoError(t, err)
		// The imported alternative (named ref i:PosHolder) must govern: -5 is invalid.
		require.ErrorIs(t, validate(t, schema, `<i:e xmlns:i="urn:i" value="-5"/>`), xsd.ErrValidationFailed)
		require.NoError(t, validate(t, schema, `<i:e xmlns:i="urn:i" value="5"/>`))
	})
}
