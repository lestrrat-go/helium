package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestVersion11CTAReviewFixes covers the round-1 gauntlet findings on conditional
// type assignment: inheritance of wildcard-matched and defaulted attributes, the
// substitutability of a simple alternative against a complex declared type,
// static-base-uri sourcing, the exactly-one-type-source rule, and whitespace
// handling of xpathDefaultNamespace.
func TestVersion11CTAReviewFixes(t *testing.T) {
	compileURL := func(t *testing.T, s, url, label string) (*xsd.Schema, error) {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(s))
		require.NoError(t, err)
		if url != "" {
			doc.SetURL(url)
		}
		c := xsd.NewCompiler().Version(xsd.Version11)
		if label != "" {
			c = c.Label(label)
		}
		return c.Compile(t.Context(), doc)
	}
	compile := func(t *testing.T, s string) (*xsd.Schema, error) {
		t.Helper()
		return compileURL(t, s, "", "")
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

	// XSDCTA-001: an inheritable global attribute admitted through xs:anyAttribute on
	// an ancestor must be visible to a descendant's CTA test.
	t.Run("wildcard-matched inheritable attribute drives CTA", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attribute name="lang" type="xs:string" inheritable="true"/>
  <xs:element name="doc">
    <xs:complexType>
      <xs:sequence><xs:element ref="chap" maxOccurs="unbounded"/></xs:sequence>
      <xs:anyAttribute processContents="lax"/>
    </xs:complexType>
  </xs:element>
  <xs:element name="chap">
    <xs:alternative test="@lang='de'">
      <xs:complexType><xs:sequence><xs:element name="de"/></xs:sequence></xs:complexType>
    </xs:alternative>
    <xs:alternative type="xs:error"/>
  </xs:element>
</xs:schema>`
		schema := mustCompile(t, s)
		// @lang reaches chap only via the wildcard-matched inheritable global attr.
		require.NoError(t, validate(t, schema, `<doc lang="de"><chap><de/></chap></doc>`))
		// lang!=de → testless xs:error default → invalid (proves inheritance path).
		require.ErrorIs(t, validate(t, schema, `<doc lang="fr"><chap><de/></chap></doc>`), xsd.ErrValidationFailed)
	})

	// XSDCTA-002: a defaulted inheritable attribute (absent in the instance) must
	// also be contributed to the inherited-attribute set.
	t.Run("defaulted inheritable attribute drives CTA", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc">
    <xs:complexType>
      <xs:sequence><xs:element ref="chap" maxOccurs="unbounded"/></xs:sequence>
      <xs:attribute name="lang" type="xs:string" default="de" inheritable="true"/>
    </xs:complexType>
  </xs:element>
  <xs:element name="chap">
    <xs:alternative test="@lang='de'">
      <xs:complexType><xs:sequence><xs:element name="de"/></xs:sequence></xs:complexType>
    </xs:alternative>
    <xs:alternative type="xs:error"/>
  </xs:element>
</xs:schema>`
		schema := mustCompile(t, s)
		// doc/@lang is absent, so the default "de" is inserted and inherited by chap.
		require.NoError(t, validate(t, schema, `<doc><chap><de/></chap></doc>`))
	})

	// XSDCTA-003: a simple alternative type against a COMPLEX declared type is never
	// validly substitutable.
	t.Run("simple alternative for complex declared type is a schema error", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Box"><xs:sequence/><xs:attribute name="k" type="xs:string"/></xs:complexType>
  <xs:element name="e" type="Box">
    <xs:alternative test="@k='s'" type="xs:string"/>
  </xs:element>
</xs:schema>`
		_, err := compile(t, s)
		require.ErrorIs(t, err, xsd.ErrCompilationFailed)
	})

	// XSDCTA-004: fn:static-base-uri() reflects the document URL, not the diagnostic
	// label.
	t.Run("static-base-uri reflects document URL not label", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Holder"><xs:sequence/><xs:attribute name="value" type="xs:integer"/></xs:complexType>
  <xs:complexType name="PosHolder">
    <xs:complexContent>
      <xs:restriction base="Holder"><xs:sequence/><xs:attribute name="value" type="xs:positiveInteger"/></xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="e" type="Holder">
    <xs:alternative test="ends-with(static-base-uri(), 'cta-base.xsd')" type="PosHolder"/>
  </xs:element>
</xs:schema>`
		// URL ends with cta-base.xsd; the label deliberately does NOT — the test must
		// resolve PosHolder from the URL, proving the label cannot leak in.
		schema, err := compileURL(t, s, "file:///schemas/cta-base.xsd", "diagnostic-label-only")
		require.NoError(t, err)
		// static-base-uri matches → PosHolder governs → -5 is not positive.
		require.ErrorIs(t, validate(t, schema, `<e value="-5"/>`), xsd.ErrValidationFailed)
		require.NoError(t, validate(t, schema, `<e value="5"/>`))

		// With no URL, static-base-uri does not end with cta-base.xsd → declared
		// Holder governs and -5 (an integer) is valid.
		schemaNoURL, err := compile(t, s)
		require.NoError(t, err)
		require.NoError(t, validate(t, schemaNoURL, `<e value="-5"/>`))
	})

	// XSDCTA-005: exactly one governing-type source is allowed.
	t.Run("both @type and inline type is a schema error", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="T"><xs:sequence/></xs:complexType>
  <xs:element name="e" type="T">
    <xs:alternative test="@k='x'" type="T">
      <xs:complexType><xs:sequence/></xs:complexType>
    </xs:alternative>
  </xs:element>
</xs:schema>`
		_, err := compile(t, s)
		require.ErrorIs(t, err, xsd.ErrCompilationFailed)
	})

	t.Run("multiple inline types is a schema error", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="T"><xs:sequence/></xs:complexType>
  <xs:element name="e" type="T">
    <xs:alternative test="@k='x'">
      <xs:complexType><xs:sequence/></xs:complexType>
      <xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType>
    </xs:alternative>
  </xs:element>
</xs:schema>`
		_, err := compile(t, s)
		require.ErrorIs(t, err, xsd.ErrCompilationFailed)
	})

	// XSDCTA-006: xpathDefaultNamespace is whitespace-collapse, so a ##keyword with
	// surrounding whitespace is still recognised.
	t.Run("xpathDefaultNamespace keyword with surrounding whitespace", func(t *testing.T) {
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
    <xs:alternative test="self::root" type="t:PosHolder" xpathDefaultNamespace="  ##targetNamespace  "/>
  </xs:element>
</xs:schema>`
		schema := mustCompile(t, s)
		// The collapsed ##targetNamespace makes self::root match {urn:t}root → PosHolder.
		require.NoError(t, validate(t, schema, `<root xmlns="urn:t" value="5"/>`))
		require.ErrorIs(t, validate(t, schema, `<root xmlns="urn:t" value="-5"/>`), xsd.ErrValidationFailed)
	})
}
