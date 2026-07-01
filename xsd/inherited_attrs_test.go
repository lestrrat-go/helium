package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestVersion11InheritedAttributes covers XSD 1.1 inherited attributes feeding a
// conditional-type-assignment @test: an inheritable ancestor attribute is visible
// to a descendant's CTA context, use-vs-declaration inheritability precedence, and
// the rule that a non-inheritable attribute does not mask an inheritable ancestor.
func TestVersion11InheritedAttributes(t *testing.T) {
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

	// chap (referenced) discriminates on @lang, which is declared inheritable on the
	// ancestor doc. The alternatives select a type requiring <de> or <fr>.
	const inheritSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc">
    <xs:complexType>
      <xs:sequence><xs:element ref="chap" maxOccurs="unbounded"/></xs:sequence>
      <xs:attribute name="lang" inheritable="true"/>
    </xs:complexType>
  </xs:element>
  <xs:element name="chap">
    <xs:alternative test="@lang='de'">
      <xs:complexType><xs:sequence><xs:element name="de"/></xs:sequence></xs:complexType>
    </xs:alternative>
    <xs:alternative test="@lang='fr'">
      <xs:complexType><xs:sequence><xs:element name="fr"/></xs:sequence></xs:complexType>
    </xs:alternative>
    <xs:alternative type="xs:error"/>
  </xs:element>
</xs:schema>`

	t.Run("inheritable ancestor attribute drives CTA", func(t *testing.T) {
		t.Parallel()
		schema := mustCompile(t, inheritSchema)
		require.NoError(t, validate(t, schema, `<doc lang="de"><chap><de/></chap></doc>`))
		require.NoError(t, validate(t, schema, `<doc lang="fr"><chap><fr/></chap></doc>`))
		// lang=fr selects the <fr> type, so <de> content is invalid.
		require.ErrorIs(t, validate(t, schema, `<doc lang="fr"><chap><de/></chap></doc>`), xsd.ErrValidationFailed)
		// no inherited lang → testless default xs:error → invalid.
		require.ErrorIs(t, validate(t, schema, `<doc><chap><de/></chap></doc>`), xsd.ErrValidationFailed)
	})

	t.Run("non-inheritable attribute does not mask an inheritable ancestor", func(t *testing.T) {
		t.Parallel()
		// doc/@lang is inheritable; part/@lang is NOT, so chap still inherits doc's lang.
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc">
    <xs:complexType>
      <xs:sequence><xs:element ref="part" maxOccurs="unbounded"/></xs:sequence>
      <xs:attribute name="lang" inheritable="true"/>
    </xs:complexType>
  </xs:element>
  <xs:element name="part">
    <xs:complexType>
      <xs:sequence><xs:element ref="chap" maxOccurs="unbounded"/></xs:sequence>
      <xs:attribute name="lang" inheritable="false"/>
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
		// chap inherits lang=de from doc even though part/@lang="fr" is non-inheritable.
		require.NoError(t, validate(t, schema, `<doc lang="de"><part lang="fr"><chap><de/></chap></part></doc>`))
	})

	t.Run("inheritable=false on use overrides declaration inheritable=true", func(t *testing.T) {
		t.Parallel()
		// The global lang declaration is inheritable, but the ref use sets false, so
		// the attribute is NOT inherited and the default (xs:error) governs chap.
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc">
    <xs:complexType>
      <xs:sequence><xs:element ref="chap" maxOccurs="unbounded"/></xs:sequence>
      <xs:attribute ref="lang" inheritable="false"/>
    </xs:complexType>
  </xs:element>
  <xs:element name="chap">
    <xs:alternative test="@lang='de'">
      <xs:complexType><xs:sequence><xs:element name="de"/></xs:sequence></xs:complexType>
    </xs:alternative>
    <xs:alternative>
      <xs:complexType><xs:sequence><xs:element name="en"/></xs:sequence></xs:complexType>
    </xs:alternative>
  </xs:element>
  <xs:attribute name="lang" type="xs:string" inheritable="true"/>
</xs:schema>`
		schema := mustCompile(t, s)
		// @lang is not inherited (use wins), so the testless default (<en>) governs.
		require.NoError(t, validate(t, schema, `<doc lang="de"><chap><en/></chap></doc>`))
		require.ErrorIs(t, validate(t, schema, `<doc lang="de"><chap><de/></chap></doc>`), xsd.ErrValidationFailed)
	})
}

// TestVersion11CTASchemaValidity covers the XSD 1.1 conditional-type-assignment
// schema-validity rules: a non-boolean inheritable value, a restriction that
// changes an attribute's inheritability, and a type alternative whose type is not
// validly substitutable for the element's declared type are all schema errors.
func TestVersion11CTASchemaValidity(t *testing.T) {
	compile := func(t *testing.T, s string) error {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(s))
		require.NoError(t, err)
		_, cerr := xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), doc)
		return cerr
	}

	t.Run("non-boolean inheritable is a schema error", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e">
    <xs:complexType>
      <xs:sequence/>
      <xs:attribute name="a" inheritable="2"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		require.ErrorIs(t, compile(t, s), xsd.ErrCompilationFailed)
	})

	t.Run("whitespace-only boolean inheritable is accepted", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e">
    <xs:complexType>
      <xs:sequence/>
      <xs:attribute name="a" inheritable=" 1 "/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		require.NoError(t, compile(t, s))
	})

	t.Run("restriction changing inheritability is a schema error", func(t *testing.T) {
		t.Parallel()
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="Base">
    <xs:sequence/>
    <xs:attribute name="a" inheritable="true"/>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="Base">
        <xs:sequence/>
        <xs:attribute name="a" inheritable="false"/>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`
		require.ErrorIs(t, compile(t, s), xsd.ErrCompilationFailed)
	})

	t.Run("alternative type not substitutable for declared type is a schema error", func(t *testing.T) {
		t.Parallel()
		// declared type B is complex; the alternative's inline complex type is not
		// derived from B, violating the type-table substitutability constraint.
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="B">
    <xs:sequence><xs:element name="x" type="xs:string"/></xs:sequence>
    <xs:attribute name="kind" type="xs:string"/>
  </xs:complexType>
  <xs:element name="e" type="B">
    <xs:alternative test="@kind='y'">
      <xs:complexType><xs:sequence><xs:element name="y" type="xs:string"/></xs:sequence></xs:complexType>
    </xs:alternative>
  </xs:element>
</xs:schema>`
		require.ErrorIs(t, compile(t, s), xsd.ErrCompilationFailed)
	})

	t.Run("alternative type derived from declared type compiles", func(t *testing.T) {
		t.Parallel()
		// A restriction of the declared type is validly substitutable.
		const s = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="B">
    <xs:sequence/>
    <xs:attribute name="kind" type="xs:string"/>
    <xs:attribute name="value" type="xs:integer"/>
  </xs:complexType>
  <xs:complexType name="Pos">
    <xs:complexContent>
      <xs:restriction base="B">
        <xs:sequence/>
        <xs:attribute name="kind" type="xs:string"/>
        <xs:attribute name="value" type="xs:positiveInteger"/>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="e" type="B">
    <xs:alternative test="@kind='pos'" type="Pos"/>
  </xs:element>
</xs:schema>`
		require.NoError(t, compile(t, s))
	})
}
