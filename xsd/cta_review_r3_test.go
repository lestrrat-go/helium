package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestVersion11CTAReviewR3 covers the round-3 gauntlet findings: the default
// element namespace for CTA tests (##local unless xpathDefaultNamespace applies),
// xs:error not bypassed by xsi:nil, retention of xml:* attributes in the CTA
// context, and an empty @type counting as a (malformed) governing-type source.
func TestVersion11CTAReviewR3(t *testing.T) {
	compile := func(t *testing.T, s string) (*xsd.Schema, error) {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(s))
		require.NoError(t, err)
		return xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), doc)
	}
	mustCompile := func(t *testing.T, s string) *xsd.Schema {
		t.Helper()
		schema, err := compile(t, s)
		require.NoError(t, err)
		return schema
	}
	validate := func(t *testing.T, schema *xsd.Schema, instance string) error {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(instance))
		require.NoError(t, err)
		return xsd.NewValidator(schema).Validate(t.Context(), doc)
	}

	// XSDCTA-R3-001: without xpathDefaultNamespace, an unprefixed name test has NO
	// default element namespace (##local) even when the schema document declares a
	// default xmlns, so self::root must NOT match {urn:t}root.
	t.Run("no xpathDefaultNamespace means no default element namespace", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="urn:t" xmlns="urn:t" elementFormDefault="qualified">
  <xs:complexType name="Holder"><xs:sequence/><xs:attribute name="value" type="xs:integer"/></xs:complexType>
  <xs:complexType name="PosHolder">
    <xs:complexContent>
      <xs:restriction base="Holder"><xs:sequence/><xs:attribute name="value" type="xs:positiveInteger"/></xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="Holder">
    <xs:alternative test="self::root" type="PosHolder"/>
  </xs:element>
</xs:schema>`
		schema := mustCompile(t, s)
		// self::root resolves to {}root (no default element namespace), does not match
		// {urn:t}root, so PosHolder is NOT selected and Holder (integer) governs: -5 OK.
		require.NoError(t, validate(t, schema, `<root xmlns="urn:t" value="-5"/>`))
	})

	// XSDCTA-R3-002: xs:error selected by CTA must invalidate the element even when
	// xsi:nil="true" routes it through the nilled-element path.
	t.Run("xs:error not bypassed by xsi:nil", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="MsgT"><xs:sequence/><xs:attribute name="kind" type="xs:string"/></xs:complexType>
  <xs:element name="msg" type="MsgT" nillable="true">
    <xs:alternative test="@kind='bad'" type="xs:error"/>
  </xs:element>
</xs:schema>`
		schema := mustCompile(t, s)
		// kind='bad' selects xs:error: invalid despite xsi:nil.
		require.ErrorIs(t, validate(t, schema,
			`<msg xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" kind="bad" xsi:nil="true"/>`), xsd.ErrValidationFailed)
		// kind='ok' keeps MsgT, which is nillable → valid.
		require.NoError(t, validate(t, schema,
			`<msg xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" kind="ok" xsi:nil="true"/>`))
	})

	// XSDCTA-R3-003: an xml:* attribute (xml:lang) must be retained in the CTA
	// context so a test can discriminate on it.
	t.Run("xml:lang drives CTA", func(t *testing.T) {
		t.Parallel()
		// In XSD 1.1 an undeclared xml:* attribute is not auto-allowed, so the type
		// must permit it (here via an anyAttribute wildcard) — independent of the CTA
		// context retaining it for the @test.
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Holder"><xs:sequence/><xs:attribute name="value" type="xs:integer"/><xs:anyAttribute namespace="##other" processContents="lax"/></xs:complexType>
  <xs:complexType name="PosHolder">
    <xs:complexContent>
      <xs:restriction base="Holder"><xs:sequence/><xs:attribute name="value" type="xs:positiveInteger"/><xs:anyAttribute namespace="##other" processContents="lax"/></xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="e" type="Holder">
    <xs:alternative test="@xml:lang='de'" type="PosHolder"/>
  </xs:element>
</xs:schema>`
		schema := mustCompile(t, s)
		// xml:lang='de' selects PosHolder → -5 (not positive) invalid.
		require.ErrorIs(t, validate(t, schema, `<e xml:lang="de" value="-5"/>`), xsd.ErrValidationFailed)
		require.NoError(t, validate(t, schema, `<e xml:lang="de" value="5"/>`))
		// no xml:lang → Holder (integer) governs → -5 valid.
		require.NoError(t, validate(t, schema, `<e value="-5"/>`))
	})

	// XSDCTA-R3-004: a present-but-empty @type is a governing-type source, so @type=""
	// plus an inline type is a conflict, and a bare @type="" is an invalid QName.
	t.Run("empty type plus inline is a schema error", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="T"><xs:sequence/></xs:complexType>
  <xs:element name="e" type="T">
    <xs:alternative test="@k='x'" type="">
      <xs:complexType><xs:sequence/></xs:complexType>
    </xs:alternative>
  </xs:element>
</xs:schema>`
		_, err := compile(t, s)
		require.ErrorIs(t, err, xsd.ErrCompilationFailed)
	})

	t.Run("bare empty type is an invalid QName", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="T"><xs:sequence/></xs:complexType>
  <xs:element name="e" type="T">
    <xs:alternative test="@k='x'" type=""/>
  </xs:element>
</xs:schema>`
		_, err := compile(t, s)
		require.ErrorIs(t, err, xsd.ErrCompilationFailed)
	})
}
