package xsd_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// compileAssertion / validateAssertion are small local helpers mirroring the
// pattern in version_assert_test.go.
func compileAssertion(t *testing.T, c xsd.Compiler, s string) (*xsd.Schema, error) {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(s))
	require.NoError(t, err)
	return c.Compile(t.Context(), doc)
}

func validateAssertion(t *testing.T, schema *xsd.Schema, instance string) error {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(instance))
	require.NoError(t, err)
	return xsd.NewValidator(schema).Validate(t.Context(), doc)
}

// TestVersion11AssertionFacet covers the XSD 1.1 <xs:assertion> simple-type facet:
// $value bound to the typed atomic value, inheritance along the restriction chain,
// list typing, the absent-context-item rule, XSD 1.0 ignoring the facet, and a
// malformed facet XPath being a compile error.
func TestVersion11AssertionFacet(t *testing.T) {
	t.Run("typed $value comparison", func(t *testing.T) {
		t.Parallel()
		// $value is xs:integer, so the arithmetic/comparison work against a number.
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="even">
    <xs:simpleType>
      <xs:restriction base="xs:integer">
        <xs:assertion test="$value mod 2 = 0"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
		schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
		require.NoError(t, err)
		require.NoError(t, validateAssertion(t, schema, `<even>4</even>`))
		require.ErrorIs(t, validateAssertion(t, schema, `<even>3</even>`), xsd.ErrValidationFailed)
	})

	t.Run("$value typed as date supports lt", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="d">
    <xs:simpleType>
      <xs:restriction base="xs:date">
        <xs:assertion test="$value lt xs:date('2000-01-01')"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
		schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
		require.NoError(t, err)
		require.NoError(t, validateAssertion(t, schema, `<d>1999-12-31</d>`))
		require.ErrorIs(t, validateAssertion(t, schema, `<d>2001-01-01</d>`), xsd.ErrValidationFailed)
	})

	t.Run("inherited along restriction chain (both hold)", func(t *testing.T) {
		t.Parallel()
		// A derived restriction's value must satisfy BOTH the base assertion
		// ($value gt 0) and the derived assertion ($value lt 100).
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="positive">
    <xs:restriction base="xs:integer">
      <xs:assertion test="$value gt 0"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="n">
    <xs:simpleType>
      <xs:restriction base="positive">
        <xs:assertion test="$value lt 100"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
		schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
		require.NoError(t, err)
		require.NoError(t, validateAssertion(t, schema, `<n>50</n>`))
		require.ErrorIs(t, validateAssertion(t, schema, `<n>0</n>`), xsd.ErrValidationFailed)   // base fails
		require.ErrorIs(t, validateAssertion(t, schema, `<n>200</n>`), xsd.ErrValidationFailed) // derived fails
	})

	t.Run("$value is a sequence for a list type", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="ints">
    <xs:list itemType="xs:integer"/>
  </xs:simpleType>
  <xs:element name="uniq">
    <xs:simpleType>
      <xs:restriction base="ints">
        <xs:assertion test="count($value) eq count(distinct-values($value))"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
		schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
		require.NoError(t, err)
		require.NoError(t, validateAssertion(t, schema, `<uniq>1 2 3</uniq>`))
		require.ErrorIs(t, validateAssertion(t, schema, `<uniq>1 2 2</uniq>`), xsd.ErrValidationFailed)
	})

	t.Run("context item is absent (dot raises a dynamic error)", func(t *testing.T) {
		t.Parallel()
		// Per XSD 1.1 the assertion-facet focus is absent, so "." is a dynamic
		// error and the assertion is not satisfied for every value.
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="s">
    <xs:simpleType>
      <xs:restriction base="xs:string">
        <xs:assertion test=". = 'x'"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
		schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
		require.NoError(t, err)
		require.ErrorIs(t, validateAssertion(t, schema, `<s>x</s>`), xsd.ErrValidationFailed)
	})

	t.Run("xpathDefaultNamespace on the facet", func(t *testing.T) {
		t.Parallel()
		// 'double' resolves to xs:double because xpathDefaultNamespace binds the
		// XSD namespace as the default for unprefixed type names.
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="n">
    <xs:simpleType>
      <xs:restriction base="xs:string">
        <xs:assertion test="$value castable as double" xpathDefaultNamespace="http://www.w3.org/2001/XMLSchema"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
		schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
		require.NoError(t, err)
		require.NoError(t, validateAssertion(t, schema, `<n>23.5</n>`))
		require.ErrorIs(t, validateAssertion(t, schema, `<n>10f4</n>`), xsd.ErrValidationFailed)
	})

	t.Run("1.0 ignores the assertion facet", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="even">
    <xs:simpleType>
      <xs:restriction base="xs:integer">
        <xs:assertion test="$value mod 2 = 0"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
		schema, err := compileAssertion(t, xsd.NewCompiler(), schemaXML)
		require.NoError(t, err)
		// The facet is not enforced in 1.0, so an odd value is accepted.
		require.NoError(t, validateAssertion(t, schema, `<even>3</even>`))
	})

	t.Run("malformed facet XPath is a compile error", func(t *testing.T) {
		t.Parallel()
		const bad = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e">
    <xs:simpleType>
      <xs:restriction base="xs:integer">
        <xs:assertion test="$value +"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
		_, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), bad)
		require.ErrorIs(t, err, xsd.ErrCompilationFailed)
	})
}

// TestVersion11AssertEdges covers xs:assert edge cases on complex types beyond the
// basic case in TestVersion11Assert: $value on a simpleContent complex type, an
// assertion that must see a typed attribute, and XDM context isolation (an
// absolute path "//" cannot escape the element subtree).
func TestVersion11AssertEdges(t *testing.T) {
	t.Run("$value typed on simpleContent complex type", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="amount">
    <xs:complexType>
      <xs:simpleContent>
        <xs:extension base="xs:integer">
          <xs:attribute name="cap" type="xs:integer"/>
          <xs:assert test="$value le xs:integer(@cap)"/>
        </xs:extension>
      </xs:simpleContent>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
		require.NoError(t, err)
		require.NoError(t, validateAssertion(t, schema, `<amount cap="10">7</amount>`))
		require.ErrorIs(t, validateAssertion(t, schema, `<amount cap="10">12</amount>`), xsd.ErrValidationFailed)
	})

	t.Run("typed attribute in value comparison", func(t *testing.T) {
		t.Parallel()
		// @length is xs:integer, so a typed comparison "@length eq count(...)"
		// works instead of failing as xs:string vs xs:integer.
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="list">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" type="xs:string" minOccurs="0" maxOccurs="unbounded"/>
      </xs:sequence>
      <xs:attribute name="length" type="xs:nonNegativeInteger"/>
      <xs:assert test="@length eq count(item)"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
		require.NoError(t, err)
		require.NoError(t, validateAssertion(t, schema, `<list length="2"><item>a</item><item>b</item></list>`))
		require.ErrorIs(t, validateAssertion(t, schema, `<list length="3"><item>a</item></list>`), xsd.ErrValidationFailed)
	})

	t.Run("context isolation: absolute path cannot escape subtree", func(t *testing.T) {
		t.Parallel()
		// The assert on <inner> uses //x; the document has two <x> elements but
		// only one is inside the inner subtree. Because the assertion tree is
		// rooted at the element, "//" raises XPDY0050 and the assertion fails —
		// it cannot count document-wide.
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="x" type="xs:string"/>
        <xs:element name="inner" type="innerType"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
  <xs:complexType name="innerType">
    <xs:sequence>
      <xs:element name="x" type="xs:string"/>
    </xs:sequence>
    <xs:assert test="count(//x) eq 2"/>
  </xs:complexType>
</xs:schema>`
		schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
		require.NoError(t, err)
		require.ErrorIs(t, validateAssertion(t, schema,
			`<root><x>a</x><inner><x>b</x></inner></root>`), xsd.ErrValidationFailed)
	})

	t.Run("context isolation: comments excluded from assertion tree", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e">
    <xs:complexType>
      <xs:attribute name="a" use="required"/>
      <xs:assert test="empty(.//comment())"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
		require.NoError(t, err)
		// A comment in the instance must be invisible to the assertion → valid.
		require.NoError(t, validateAssertion(t, schema, `<e a="1"><!-- hi --></e>`))
	})
}

// TestVersion11AssertionGauntletFixes covers the four gauntlet-review fixes:
// (1) required-attribute inheritance for a NON-assert 1.1 restriction; (2) a list
// whose item type is a union typed via per-item active-member resolution; (3) a
// QName-typed $value resolved with namespace context; (4) a named user-defined
// simple type atomizing through its builtin base via SchemaDeclarations.
func TestVersion11AssertionGauntletFixes(t *testing.T) {
	t.Run("non-assert 1.1 restriction still requires inherited attribute", func(t *testing.T) {
		t.Parallel()
		// floatType-style: a simpleContent restriction with NO assert that does not
		// redeclare the base's required attribute must still require it.
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base">
    <xs:simpleContent>
      <xs:extension base="xs:string">
        <xs:attribute name="req" type="xs:int" use="required"/>
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>
  <xs:element name="e">
    <xs:complexType>
      <xs:simpleContent>
        <xs:restriction base="base"/>
      </xs:simpleContent>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
		require.NoError(t, err)
		require.NoError(t, validateAssertion(t, schema, `<e req="5">x</e>`))
		require.ErrorIs(t, validateAssertion(t, schema, `<e>x</e>`), xsd.ErrValidationFailed)
	})

	t.Run("1.0 restriction omitting required attribute stays byte-identical", func(t *testing.T) {
		t.Parallel()
		// In 1.0 helium does not inherit restriction attributes; the historical
		// behavior is preserved (the restriction is rejected at compile time for the
		// missing required base attribute). This guards the 1.1-only gating.
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base">
    <xs:simpleContent>
      <xs:extension base="xs:string">
        <xs:attribute name="req" type="xs:int" use="required"/>
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>
  <xs:element name="e">
    <xs:complexType>
      <xs:simpleContent>
        <xs:restriction base="base"/>
      </xs:simpleContent>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		_, err := compileAssertion(t, xsd.NewCompiler(), schemaXML)
		require.ErrorIs(t, err, xsd.ErrCompilationFailed)
	})

	t.Run("list item type that is a union is typed per active member", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="intOrBool">
    <xs:union memberTypes="xs:int xs:boolean"/>
  </xs:simpleType>
  <xs:simpleType name="ublist">
    <xs:list itemType="intOrBool"/>
  </xs:simpleType>
  <xs:element name="e">
    <xs:simpleType>
      <xs:restriction base="ublist">
        <xs:assertion test="$value[1] instance of xs:int and $value[2] instance of xs:boolean"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
		schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
		require.NoError(t, err)
		require.NoError(t, validateAssertion(t, schema, `<e>5 true</e>`))
		// Reversed: first item is a boolean, second an int — the typed instance-of
		// checks fail, proving the items are genuinely typed (not untypedAtomic).
		require.ErrorIs(t, validateAssertion(t, schema, `<e>true 5</e>`), xsd.ErrValidationFailed)
	})

	t.Run("QName $value resolved with namespace context", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e">
    <xs:simpleType>
      <xs:restriction base="xs:QName">
        <xs:assertion test="namespace-uri-from-QName($value) = 'http://example.com/ns'"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
		schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
		require.NoError(t, err)
		require.NoError(t, validateAssertion(t, schema, `<e xmlns:p="http://example.com/ns">p:foo</e>`))
		require.ErrorIs(t, validateAssertion(t, schema, `<e xmlns:q="http://other/ns">q:foo</e>`), xsd.ErrValidationFailed)
	})

	t.Run("named user simple type atomizes through builtin base", func(t *testing.T) {
		t.Parallel()
		// @len is typed as a NAMED restriction of xs:integer; data(@len) must
		// atomize as xs:integer (via SchemaDeclarations), not xs:untypedAtomic.
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="lengthType">
    <xs:restriction base="xs:integer"/>
  </xs:simpleType>
  <xs:element name="e">
    <xs:complexType>
      <xs:attribute name="len" type="lengthType"/>
      <xs:assert test="data(@len) instance of xs:integer"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
		require.NoError(t, err)
		require.NoError(t, validateAssertion(t, schema, `<e len="5"/>`))
	})

	t.Run("simpleContent restriction enumeration constrains content", func(t *testing.T) {
		t.Parallel()
		// Regression guard for the merge-for-all change: the inherited attribute is
		// now allowed, so the enumeration facet on the restriction must actually
		// constrain the content (it is no longer masked by an attribute rejection).
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base">
    <xs:simpleContent>
      <xs:extension base="xs:string">
        <xs:attribute name="a" type="xs:string"/>
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>
  <xs:element name="e">
    <xs:complexType>
      <xs:simpleContent>
        <xs:restriction base="base">
          <xs:enumeration value="square"/>
        </xs:restriction>
      </xs:simpleContent>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
		require.NoError(t, err)
		require.NoError(t, validateAssertion(t, schema, `<e a="x">square</e>`))
		require.ErrorIs(t, validateAssertion(t, schema, `<e a="x">circle</e>`), xsd.ErrValidationFailed)
	})
}

// TestVersion11AssertionRound3Fixes covers the round-3 gauntlet findings:
// (1) topological (not source-order) restriction attribute merging across a
// forward-referenced chain; (2) version-aware SchemaDeclarations.ValidateCast so a
// 1.1-only lexical form is castable inside an assertion; (3) xs:assert $value
// honoring the element's default/fixed effective value for an empty element.
func TestVersion11AssertionRound3Fixes(t *testing.T) {
	t.Run("forward-referenced restriction chain inherits required attribute", func(t *testing.T) {
		t.Parallel()
		// D restricts B, B restricts A; B is declared AFTER D in source order. A
		// declares a required attribute that B and D inherit. Source-order merging
		// would let D miss it (B not yet merged when D is processed); topological
		// merging inherits it regardless of declaration order.
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="d" type="D"/>
  <xs:complexType name="D">
    <xs:simpleContent>
      <xs:restriction base="B"/>
    </xs:simpleContent>
  </xs:complexType>
  <xs:complexType name="B">
    <xs:simpleContent>
      <xs:restriction base="A"/>
    </xs:simpleContent>
  </xs:complexType>
  <xs:complexType name="A">
    <xs:simpleContent>
      <xs:extension base="xs:string">
        <xs:attribute name="req" type="xs:string" use="required"/>
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>
</xs:schema>`
		schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
		require.NoError(t, err)
		require.NoError(t, validateAssertion(t, schema, `<d req="x">ok</d>`))
		require.ErrorIs(t, validateAssertion(t, schema, `<d>ok</d>`), xsd.ErrValidationFailed)
	})

	t.Run("user-defined castable inside assertion uses the schema version", func(t *testing.T) {
		t.Parallel()
		// year 0000 is a 1.1-only lexical form. `$value castable as t:myDate` must
		// use the schema's 1.1 rules, not TypeDef.Validate's hardcoded 1.0 default.
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="urn:t" xmlns:t="urn:t" elementFormDefault="qualified">
  <xs:simpleType name="myDate">
    <xs:restriction base="xs:date"/>
  </xs:simpleType>
  <xs:element name="e">
    <xs:simpleType>
      <xs:restriction base="xs:string">
        <xs:assertion test="$value castable as t:myDate"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
		schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
		require.NoError(t, err)
		require.NoError(t, validateAssertion(t, schema, `<e xmlns="urn:t">0000-01-01</e>`))
		require.ErrorIs(t, validateAssertion(t, schema, `<e xmlns="urn:t">not-a-date</e>`), xsd.ErrValidationFailed)
	})

	t.Run("xs:assert $value honors element default for empty content", func(t *testing.T) {
		t.Parallel()
		// An empty element with default="5" must expose $value=5 to the assert,
		// not the raw empty text (which would make $value the empty sequence).
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e" default="5">
    <xs:complexType>
      <xs:simpleContent>
        <xs:extension base="xs:integer">
          <xs:assert test="$value eq 5"/>
        </xs:extension>
      </xs:simpleContent>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
		require.NoError(t, err)
		require.NoError(t, validateAssertion(t, schema, `<e/>`))
		// Non-empty content uses the actual value, so 7 fails the assert.
		require.ErrorIs(t, validateAssertion(t, schema, `<e>7</e>`), xsd.ErrValidationFailed)
	})
}

func TestVersion11AssertionFacetSchemaAwareUnionCast(t *testing.T) {
	t.Parallel()
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="urn:t" xmlns:t="urn:t" elementFormDefault="qualified">
  <xs:simpleType name="SmallInt">
    <xs:restriction base="xs:int">
      <xs:maxInclusive value="10"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="SmallIntUnion">
    <xs:union memberTypes="t:SmallInt"/>
  </xs:simpleType>
  <xs:simpleType name="RejectSevenUnion">
    <xs:restriction base="t:SmallIntUnion">
      <xs:assertion test="$value ne 7"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="castable">
    <xs:simpleType>
      <xs:restriction base="xs:string">
        <xs:assertion test="$value castable as t:SmallIntUnion"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
  <xs:element name="cast">
    <xs:simpleType>
      <xs:restriction base="xs:string">
        <xs:assertion test="($value cast as t:SmallIntUnion) instance of t:SmallInt"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
  <xs:element name="targetCastable">
    <xs:simpleType>
      <xs:restriction base="xs:string">
        <xs:assertion test="$value castable as t:RejectSevenUnion"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
  <xs:element name="targetCast">
    <xs:simpleType>
      <xs:restriction base="xs:string">
        <xs:assertion test="($value cast as t:RejectSevenUnion) instance of t:SmallInt"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
	schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
	require.NoError(t, err)
	require.NoError(t, validateAssertion(t, schema, `<castable xmlns="urn:t">5</castable>`))
	require.ErrorIs(t, validateAssertion(t, schema, `<castable xmlns="urn:t">12</castable>`), xsd.ErrValidationFailed)
	require.NoError(t, validateAssertion(t, schema, `<cast xmlns="urn:t">5</cast>`))
	require.ErrorIs(t, validateAssertion(t, schema, `<cast xmlns="urn:t">12</cast>`), xsd.ErrValidationFailed)
	require.NoError(t, validateAssertion(t, schema, `<targetCastable xmlns="urn:t">5</targetCastable>`))
	require.ErrorIs(t, validateAssertion(t, schema, `<targetCastable xmlns="urn:t">7</targetCastable>`), xsd.ErrValidationFailed)
	require.NoError(t, validateAssertion(t, schema, `<targetCast xmlns="urn:t">5</targetCast>`))
	require.ErrorIs(t, validateAssertion(t, schema, `<targetCast xmlns="urn:t">7</targetCast>`), xsd.ErrValidationFailed)
}

func TestVersion11AssertionFacetSchemaAwareCastUsesSourceLexical(t *testing.T) {
	t.Parallel()
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="urn:t" xmlns:t="urn:t" elementFormDefault="qualified">
  <xs:simpleType name="TwoDigit">
    <xs:restriction base="xs:integer">
      <xs:pattern value="[0-9]{2}"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="e">
    <xs:simpleType>
      <xs:restriction base="xs:string">
        <xs:assertion test="$value castable as t:TwoDigit"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
	schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
	require.NoError(t, err)
	require.NoError(t, validateAssertion(t, schema, `<e xmlns="urn:t">05</e>`))
	require.ErrorIs(t, validateAssertion(t, schema, `<e xmlns="urn:t">5</e>`), xsd.ErrValidationFailed)
}

func TestVersion11AssertionFacetSchemaAwareUntypedComparison(t *testing.T) {
	t.Parallel()
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="urn:t" xmlns:t="urn:t" elementFormDefault="qualified">
  <xs:simpleType name="SmallInt">
    <xs:restriction base="xs:int">
      <xs:maxInclusive value="10"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="e">
    <xs:simpleType>
      <xs:restriction base="t:SmallInt">
        <xs:assertion test="$value = xs:untypedAtomic('5')"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
	schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
	require.NoError(t, err)
	require.NoError(t, validateAssertion(t, schema, `<e xmlns="urn:t">5</e>`))
	require.ErrorIs(t, validateAssertion(t, schema, `<e xmlns="urn:t">6</e>`), xsd.ErrValidationFailed)
}

func TestVersion11AssertSkipWildcardXsiTypeIsNotPSVITyped(t *testing.T) {
	t.Parallel()
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:any processContents="skip" minOccurs="0" maxOccurs="unbounded"/>
      </xs:sequence>
      <xs:assert test="data(x) instance of xs:integer"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
	require.NoError(t, err)
	require.ErrorIs(t, validateAssertion(t, schema, `<root><x xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xs="http://www.w3.org/2001/XMLSchema" xsi:type="xs:integer">5</x></root>`), xsd.ErrValidationFailed)
}

// TestVersion11AttrMergeExtensionOfRestriction covers the round-4 finding
// (XSD11-ATTR-MERGE-EXT-001): an EXTENSION whose base is a RESTRICTION that
// itself inherited a required attribute must still require it. The effective
// attribute set must be finalized topologically across both derivation kinds, not
// in the order the extension/restriction passes happen to run.
func TestVersion11AttrMergeExtensionOfRestriction(t *testing.T) {
	// A declares required req; B restricts A (inherits req); E extends B.
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e" type="E"/>
  <xs:complexType name="E">
    <xs:simpleContent>
      <xs:extension base="B"/>
    </xs:simpleContent>
  </xs:complexType>
  <xs:complexType name="B">
    <xs:simpleContent>
      <xs:restriction base="A"/>
    </xs:simpleContent>
  </xs:complexType>
  <xs:complexType name="A">
    <xs:simpleContent>
      <xs:extension base="xs:string">
        <xs:attribute name="req" type="xs:string" use="required"/>
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>
</xs:schema>`

	t.Run("missing inherited required attribute is rejected", func(t *testing.T) {
		t.Parallel()
		schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
		require.NoError(t, err)
		require.ErrorIs(t, validateAssertion(t, schema, `<e>ok</e>`), xsd.ErrValidationFailed)
	})

	t.Run("present inherited required attribute is accepted", func(t *testing.T) {
		t.Parallel()
		schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
		require.NoError(t, err)
		require.NoError(t, validateAssertion(t, schema, `<e req="x">ok</e>`))
	})

	t.Run("complexContent extension of restriction inherits required attribute", func(t *testing.T) {
		t.Parallel()
		// Same chain over complexContent (element-only) content models.
		const cc = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e" type="E"/>
  <xs:complexType name="E">
    <xs:complexContent>
      <xs:extension base="B"/>
    </xs:complexContent>
  </xs:complexType>
  <xs:complexType name="B">
    <xs:complexContent>
      <xs:restriction base="A">
        <xs:sequence>
          <xs:element name="c" type="xs:string"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:complexType name="A">
    <xs:sequence>
      <xs:element name="c" type="xs:string"/>
    </xs:sequence>
    <xs:attribute name="req" type="xs:string" use="required"/>
  </xs:complexType>
</xs:schema>`
		schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), cc)
		require.NoError(t, err)
		require.ErrorIs(t, validateAssertion(t, schema, `<e><c>x</c></e>`), xsd.ErrValidationFailed)
		require.NoError(t, validateAssertion(t, schema, `<e req="y"><c>x</c></e>`))
	})
}

// TestVersion11SimpleContentChainFacets covers the two simpleContent content-type
// findings: (1) a simpleContent EXTENSION of a named simple type must enforce that
// base type's facets/assertions (not skip validateValue); (2) a narrowed content
// type (an ancestor enumeration) is inherited through a further restriction AND a
// further extension. Both compose across the whole simpleContent derivation chain.
func TestVersion11SimpleContentChainFacets(t *testing.T) {
	t.Run("extension of named type with assertion facet enforces it", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="nonempty">
    <xs:restriction base="xs:string">
      <xs:assertion test="string-length($value) gt 0"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="s">
    <xs:complexType>
      <xs:simpleContent>
        <xs:extension base="nonempty"/>
      </xs:simpleContent>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
		require.NoError(t, err)
		require.NoError(t, validateAssertion(t, schema, `<s>x</s>`))
		require.ErrorIs(t, validateAssertion(t, schema, `<s></s>`), xsd.ErrValidationFailed)
	})

	t.Run("extension of named type with length facet enforces it", func(t *testing.T) {
		t.Parallel()
		// A non-assertion base facet must also be enforced through the extension.
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="two">
    <xs:restriction base="xs:string">
      <xs:length value="2"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="s">
    <xs:complexType>
      <xs:simpleContent>
        <xs:extension base="two">
          <xs:attribute name="a" type="xs:string"/>
        </xs:extension>
      </xs:simpleContent>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
		require.NoError(t, err)
		require.NoError(t, validateAssertion(t, schema, `<s a="z">ab</s>`))
		require.ErrorIs(t, validateAssertion(t, schema, `<s a="z">abc</s>`), xsd.ErrValidationFailed)
	})

	t.Run("ancestor enumeration honored through further restriction and extension", func(t *testing.T) {
		t.Parallel()
		// A: simpleContent extension of xs:string. B: restriction of A with
		// enumeration "square". C: restriction of B (no own facet). E: extension of B.
		// Both C and E must still reject a non-enumerated value.
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="A">
    <xs:simpleContent>
      <xs:extension base="xs:string">
        <xs:attribute name="k" type="xs:string"/>
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>
  <xs:complexType name="B">
    <xs:simpleContent>
      <xs:restriction base="A">
        <xs:enumeration value="square"/>
      </xs:restriction>
    </xs:simpleContent>
  </xs:complexType>
  <xs:complexType name="C">
    <xs:simpleContent>
      <xs:restriction base="B"/>
    </xs:simpleContent>
  </xs:complexType>
  <xs:complexType name="E">
    <xs:simpleContent>
      <xs:extension base="B">
        <xs:attribute name="m" type="xs:string"/>
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>
  <xs:element name="c" type="C"/>
  <xs:element name="e" type="E"/>
</xs:schema>`
		schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
		require.NoError(t, err)
		require.NoError(t, validateAssertion(t, schema, `<c>square</c>`))
		require.ErrorIs(t, validateAssertion(t, schema, `<c>circle</c>`), xsd.ErrValidationFailed)
		require.NoError(t, validateAssertion(t, schema, `<e>square</e>`))
		require.ErrorIs(t, validateAssertion(t, schema, `<e>circle</e>`), xsd.ErrValidationFailed)
	})
}

// TestVersion11DeepSimpleContentChain guards against a recursion-depth cutoff in
// effectiveContentSimpleType: a deep (>64 levels) but finite, acyclic simpleContent
// restriction chain whose DEEPEST step carries a narrowing enumeration must still
// have that enumeration enforced. A depth cutoff would return an intermediate type
// before reaching the narrowing facet (which lives on ContentSimpleType, not
// Facets), causing a violating value to be wrongly accepted.
func TestVersion11DeepSimpleContentChain(t *testing.T) {
	const depth = 80 // comfortably beyond the old depth>64 cutoff

	var b strings.Builder
	b.WriteString(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">`)
	// t0: simpleContent extension of xs:string.
	b.WriteString(`<xs:complexType name="t0"><xs:simpleContent><xs:extension base="xs:string"/></xs:simpleContent></xs:complexType>`)
	// t1: the only narrowing — restricts t0 with enumeration "ok".
	b.WriteString(`<xs:complexType name="t1"><xs:simpleContent><xs:restriction base="t0"><xs:enumeration value="ok"/></xs:restriction></xs:simpleContent></xs:complexType>`)
	// t2..tN: pass-through restrictions (no own narrowing) of the level below.
	for i := 2; i <= depth; i++ {
		fmt.Fprintf(&b, `<xs:complexType name="t%d"><xs:simpleContent><xs:restriction base="t%d"/></xs:simpleContent></xs:complexType>`, i, i-1)
	}
	fmt.Fprintf(&b, `<xs:element name="e" type="t%d"/>`, depth)
	b.WriteString(`</xs:schema>`)

	schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), b.String())
	require.NoError(t, err)
	require.NoError(t, validateAssertion(t, schema, `<e>ok</e>`))
	require.ErrorIs(t, validateAssertion(t, schema, `<e>bad</e>`), xsd.ErrValidationFailed)
}

// TestVersion11AssertCastRecursion guards against unbounded recursion when a
// schema-aware cast/castable inside a type's OWN xs:assertion targets that same
// type. The key invariant is TERMINATION (no stack overflow).
func TestVersion11AssertCastRecursion(t *testing.T) {
	t.Run("self-cast terminates via identity", func(t *testing.T) {
		t.Parallel()
		// $value is typed as the user type t:rec (PR859-REVIEW-01 preservation), so
		// `$value castable as t:rec` is identity-true (a value of type T is castable
		// to T) — it short-circuits before the schema-aware cast recursion, so it both
		// TERMINATES and (correctly) holds: t:rec has no constraint beyond this
		// trivially-true self-cast, so the value is VALID.
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="urn:t" xmlns:t="urn:t" elementFormDefault="qualified">
  <xs:simpleType name="rec">
    <xs:restriction base="xs:string">
      <xs:assertion test="$value castable as t:rec"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="e" type="t:rec"/>
</xs:schema>`
		schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
		require.NoError(t, err)
		require.NoError(t, validateAssertion(t, schema, `<e xmlns="urn:t">x</e>`))
	})

	t.Run("mutual recursion terminates fail-closed", func(t *testing.T) {
		t.Parallel()
		// recA's assertion casts to recB and vice versa: the source type never equals
		// the cast target, so the identity short-circuit does NOT apply and the
		// schema-aware path recurses validateCast → validateValue →
		// checkSimpleTypeAssertions → Evaluate → validateCast … The per-validation
		// cast guard must TERMINATE it (fail closed → not castable → invalid) rather
		// than overflow the stack.
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="urn:t" xmlns:t="urn:t" elementFormDefault="qualified">
  <xs:simpleType name="recA">
    <xs:restriction base="xs:string">
      <xs:assertion test="$value castable as t:recB"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="recB">
    <xs:restriction base="xs:string">
      <xs:assertion test="$value castable as t:recA"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="e" type="t:recA"/>
</xs:schema>`
		schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
		require.NoError(t, err)
		require.ErrorIs(t, validateAssertion(t, schema, `<e xmlns="urn:t">x</e>`), xsd.ErrValidationFailed)
	})
}

// TestVersion11FixedDefaultQNameNS verifies that a QName fixed/default value
// substituted into an empty element resolves its prefix against the DECLARATION's
// namespace context (where it was authored), not the instance's bindings.
func TestVersion11FixedDefaultQNameNS(t *testing.T) {
	t.Run("fixed QName resolves against schema ns even when prefix is unbound in instance", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="urn:schema" xmlns:p="urn:schema" elementFormDefault="qualified">
  <xs:element name="e" type="xs:QName" fixed="p:x"/>
</xs:schema>`
		schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
		require.NoError(t, err)
		// Empty element: the fixed "p:x" resolves via the schema's xmlns:p, so the
		// QName is valid even though the instance never binds p.
		require.NoError(t, validateAssertion(t, schema, `<e xmlns="urn:schema"/>`))
	})

	t.Run("default QName $value binds the schema URI regardless of instance bindings", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="urn:schema" xmlns:p="urn:schema" elementFormDefault="qualified">
  <xs:element name="e" default="p:x">
    <xs:complexType>
      <xs:simpleContent>
        <xs:extension base="xs:QName">
          <xs:assert test="namespace-uri-from-QName($value) = 'urn:schema'"/>
        </xs:extension>
      </xs:simpleContent>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
		require.NoError(t, err)
		// No instance binding for p at all → $value still resolves to urn:schema.
		require.NoError(t, validateAssertion(t, schema, `<e xmlns="urn:schema"/>`))
		// Instance binds p to a DIFFERENT URI → the declaration's binding still wins.
		require.NoError(t, validateAssertion(t, schema, `<e xmlns="urn:schema" xmlns:p="urn:instance"/>`))
	})

	t.Run("defaulted QName attribute observed by xs:assert uses the schema ns", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:p="urn:schema">
  <xs:element name="e">
    <xs:complexType>
      <xs:attribute name="a" type="xs:QName" default="p:x"/>
      <xs:assert test="namespace-uri-from-QName(@a) = 'urn:schema'"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
		require.NoError(t, err)
		// The default attribute is materialized as "p:x"; the assert reading @a must
		// see {urn:schema}x even though the instance never declared p.
		require.NoError(t, validateAssertion(t, schema, `<e/>`))
	})
}

// TestVersion10QNameDefaultAttrNoRewrite guards the XSD 1.0 byte-identical
// requirement: the QName default/fixed attribute materialization (a 1.1-only
// fix) must NOT run in 1.0. A default xs:QName attribute whose schema prefix
// collides with a different instance binding must be inserted exactly as
// authored — no fresh-prefix namespace-declaration rewrite.
func TestVersion10QNameDefaultAttrNoRewrite(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:p="urn:schema">
  <xs:element name="e">
    <xs:complexType>
      <xs:attribute name="a" type="xs:QName" default="p:x"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	// Default XSD 1.0 compiler (no Version11).
	schema, err := compileAssertion(t, xsd.NewCompiler(), schemaXML)
	require.NoError(t, err)

	idoc, err := helium.NewParser().Parse(t.Context(), []byte(`<e xmlns:p="urn:instance"/>`))
	require.NoError(t, err)
	require.NoError(t, xsd.NewValidator(schema).Validate(t.Context(), idoc))

	out, err := helium.WriteString(idoc)
	require.NoError(t, err)
	// The default is inserted as authored; no fresh-prefix rewrite.
	require.Contains(t, out, `a="p:x"`)
	require.NotContains(t, out, "p_gen0")
	require.NotContains(t, out, "urn:schema") // no schema-prefix declaration leaked into the instance
}

// TestVersion11SimpleContentInlineTypePlusSiblingFacets verifies that a
// simpleContent restriction with BOTH a nested <xs:simpleType> AND direct sibling
// facets composes the two — the sibling facets are not dropped.
func TestVersion11SimpleContentInlineTypePlusSiblingFacets(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base">
    <xs:simpleContent>
      <xs:extension base="xs:string"/>
    </xs:simpleContent>
  </xs:complexType>
  <xs:element name="e">
    <xs:complexType>
      <xs:simpleContent>
        <xs:restriction base="base">
          <xs:simpleType>
            <xs:restriction base="xs:string">
              <xs:length value="2"/>
            </xs:restriction>
          </xs:simpleType>
          <xs:enumeration value="ab"/>
        </xs:restriction>
      </xs:simpleContent>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
	require.NoError(t, err)
	require.NoError(t, validateAssertion(t, schema, `<e>ab</e>`))
	// "cd" satisfies the inline length=2 but NOT the sibling enumeration "ab".
	require.ErrorIs(t, validateAssertion(t, schema, `<e>cd</e>`), xsd.ErrValidationFailed)
	// "abc" satisfies the enumeration's character set but NOT the inline length=2.
	require.ErrorIs(t, validateAssertion(t, schema, `<e>abc</e>`), xsd.ErrValidationFailed)
}

// TestVersion11UnionValueSchemaAwareMember verifies that union $value active-member
// typing is schema-aware: a member whose own assertion needs `castable as t:T`
// must be selectable when building $value, so the union assertion sees the right
// type (here xs:integer rather than the xs:string fallback member).
func TestVersion11UnionValueSchemaAwareMember(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="urn:t" xmlns:t="urn:t" elementFormDefault="qualified">
  <xs:simpleType name="foo">
    <xs:restriction base="xs:integer"/>
  </xs:simpleType>
  <xs:simpleType name="memberA">
    <xs:restriction base="xs:integer">
      <xs:assertion test="$value castable as t:foo"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="u">
    <xs:union memberTypes="t:memberA xs:string"/>
  </xs:simpleType>
  <xs:element name="e">
    <xs:simpleType>
      <xs:restriction base="t:u">
        <xs:assertion test="$value instance of xs:integer"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
	schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
	require.NoError(t, err)
	// "5" is accepted by memberA (whose schema-aware assertion holds), so $value is
	// typed xs:integer and the union assertion `$value instance of xs:integer`
	// passes. Without schema-aware member probing memberA is skipped, $value falls
	// back to xs:string, and the assertion would fail.
	require.NoError(t, validateAssertion(t, schema, `<e xmlns="urn:t">5</e>`))
}

// TestVersion11AssertIsolationKeepsDefaultNamespace verifies that the isolated
// assertion tree preserves an INHERITED default namespace (xmlns="…" on an
// ancestor), so namespace-uri-for-prefix(”, .) still resolves after isolation.
func TestVersion11AssertIsolationKeepsDefaultNamespace(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="urn:elems" elementFormDefault="qualified">
  <xs:element name="outer">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="inner">
          <xs:complexType>
            <xs:sequence/>
            <xs:assert test="namespace-uri-for-prefix('', .) = 'urn:default'"/>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
	require.NoError(t, err)
	// inner is prefixed (urn:elems) but inherits the default namespace urn:default
	// from outer; isolation must keep that default binding on the copied root.
	require.NoError(t, validateAssertion(t, schema,
		`<t:outer xmlns:t="urn:elems" xmlns="urn:default"><t:inner/></t:outer>`))
}

// TestVersion11UnionFixedEnumSchemaAware verifies that union enumeration/fixed
// comparison resolves active members SCHEMA-AWARELY: a member whose own assertion
// needs `castable as t:T` must be selectable, so the enumeration comparison runs
// in the integer value space (where "05" == "5") rather than falling back to a
// string member.
func TestVersion11UnionFixedEnumSchemaAware(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="urn:t" xmlns:t="urn:t" elementFormDefault="qualified">
  <xs:simpleType name="foo">
    <xs:restriction base="xs:integer"/>
  </xs:simpleType>
  <xs:simpleType name="memberA">
    <xs:restriction base="xs:integer">
      <xs:assertion test="$value castable as t:foo"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="u">
    <xs:union memberTypes="t:memberA xs:string"/>
  </xs:simpleType>
  <xs:element name="e">
    <xs:simpleType>
      <xs:restriction base="t:u">
        <xs:enumeration value="5"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
	schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
	require.NoError(t, err)
	// "05" is value-equal to the enumeration "5" in the integer value space; this
	// only holds if both resolve to memberA (schema-aware), not the string member.
	require.NoError(t, validateAssertion(t, schema, `<e xmlns="urn:t">05</e>`))
	// "7" is not equal to the enumeration "5".
	require.ErrorIs(t, validateAssertion(t, schema, `<e xmlns="urn:t">7</e>`), xsd.ErrValidationFailed)
}

// TestVersion11AssertQNameUnprefixedNoNamespace verifies XSD QName value-space
// semantics in an assertion: an UNPREFIXED xs:QName value has NO namespace (it
// does not pick up the element's default namespace), while a prefixed value still
// resolves.
func TestVersion11AssertQNameUnprefixedNoNamespace(t *testing.T) {
	t.Run("unprefixed value has no namespace", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="urn:e" elementFormDefault="qualified">
  <xs:element name="e">
    <xs:complexType>
      <xs:attribute name="a" type="xs:QName"/>
      <xs:assert test="namespace-uri-from-QName(@a) = ''"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
		require.NoError(t, err)
		require.NoError(t, validateAssertion(t, schema, `<e xmlns="urn:e" a="x"/>`))
	})

	t.Run("prefixed value still resolves its namespace", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="urn:e" elementFormDefault="qualified">
  <xs:element name="e">
    <xs:complexType>
      <xs:attribute name="a" type="xs:QName"/>
      <xs:assert test="namespace-uri-from-QName(@a) = 'urn:p'"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
		require.NoError(t, err)
		require.NoError(t, validateAssertion(t, schema, `<e xmlns="urn:e" xmlns:p="urn:p" a="p:y"/>`))
	})
}

// TestVersion11QNameDefaultAttrWhitespace verifies that a QName default value with
// surrounding whitespace (" p:x ") is whitespace-collapsed before its prefix is
// extracted for namespace materialization, so a later xs:assert resolves the
// prefix.
func TestVersion11QNameDefaultAttrWhitespace(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:p="urn:schema">
  <xs:element name="e">
    <xs:complexType>
      <xs:attribute name="a" type="xs:QName" default=" p:x "/>
      <xs:assert test="namespace-uri-from-QName(@a) = 'urn:schema'"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
	require.NoError(t, err)
	require.NoError(t, validateAssertion(t, schema, `<e/>`))
}

// TestVersion11DefaultValueSchemaAwareAssertion guards the compile-time
// default/fixed value check (checkAttrUseConstraints) and the facet
// value-against-base checks: they evaluate xs:assertion facets, so the throwaway
// validation context must carry the schema. A default whose type's assertion uses
// a schema-aware cast must NOT reject a valid schema at compile time.
func TestVersion11DefaultValueSchemaAwareAssertion(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="urn:t" xmlns:t="urn:t" elementFormDefault="qualified">
  <xs:simpleType name="bar">
    <xs:restriction base="xs:integer"/>
  </xs:simpleType>
  <xs:simpleType name="foo">
    <xs:restriction base="xs:integer">
      <xs:assertion test="$value castable as t:bar"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="e">
    <xs:complexType>
      <xs:attribute name="a" type="t:foo" default="5"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	// The default "5" satisfies foo's schema-aware assertion, so the schema must
	// COMPILE (a nil-schema throwaway context would make `castable as t:bar` fail
	// closed and wrongly reject it).
	schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
	require.NoError(t, err)
	// And the materialized default still validates an instance.
	require.NoError(t, validateAssertion(t, schema, `<e xmlns="urn:t"/>`))

	t.Run("violating default is still rejected", func(t *testing.T) {
		t.Parallel()
		// foo additionally requires the value to be even; default "5" violates it,
		// so the compile-time default check must reject the schema.
		const bad = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="urn:t" xmlns:t="urn:t" elementFormDefault="qualified">
  <xs:simpleType name="bar">
    <xs:restriction base="xs:integer"/>
  </xs:simpleType>
  <xs:simpleType name="foo">
    <xs:restriction base="xs:integer">
      <xs:assertion test="$value castable as t:bar and $value mod 2 = 0"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="e">
    <xs:complexType>
      <xs:attribute name="a" type="t:foo" default="5"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		_, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), bad)
		require.ErrorIs(t, err, xsd.ErrCompilationFailed)
	})
}

// TestVersion11UnionDefaultQNameAttrMaterialize verifies that a default/fixed
// attribute whose type is a UNION with an active QName member has its prefix
// materialized on the instance, so a later xs:assert atomizing it resolves the
// schema-intended namespace.
func TestVersion11UnionDefaultQNameAttrMaterialize(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:p="urn:schema">
  <xs:simpleType name="qnameOrString">
    <xs:union memberTypes="xs:QName xs:string"/>
  </xs:simpleType>
  <xs:element name="e">
    <xs:complexType>
      <xs:attribute name="a" type="qnameOrString" default="p:x"/>
      <xs:assert test="namespace-uri-from-QName(@a) = 'urn:schema'"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
	require.NoError(t, err)
	// The default "p:x" is active in the xs:QName member; its prefix p (declared
	// only in the schema) must be bound on the instance so the assert resolves it.
	require.NoError(t, validateAssertion(t, schema, `<e/>`))
}

func TestVersion11UnionDefaultQNameListAttrMaterialize(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:p="urn:schema">
  <xs:simpleType name="qnames">
    <xs:list itemType="xs:QName"/>
  </xs:simpleType>
  <xs:simpleType name="qnamesOrString">
    <xs:union memberTypes="qnames xs:string"/>
  </xs:simpleType>
  <xs:element name="e">
    <xs:complexType>
      <xs:attribute name="qs" type="qnamesOrString" default="p:a p:b"/>
      <xs:assert test="count(data(@qs)) = 2 and namespace-uri-from-QName(data(@qs)[2]) = 'urn:schema'"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
	require.NoError(t, err)
	require.NoError(t, validateAssertion(t, schema, `<e/>`))
}

func TestVersion11AssertDescendantQNameListDefaultMaterialize(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:p="urn:schema">
  <xs:simpleType name="myQName">
    <xs:restriction base="xs:QName"/>
  </xs:simpleType>
  <xs:simpleType name="qnames">
    <xs:list itemType="myQName"/>
  </xs:simpleType>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="c" type="qnames" default="p:a p:b"/>
      </xs:sequence>
      <xs:assert test="count(data(c)) = 2 and namespace-uri-from-QName(data(c)[1]) = 'urn:schema'"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
	require.NoError(t, err)
	require.NoError(t, validateAssertion(t, schema, `<root><c/></root>`))
}

// TestVersion11XPathDefaultNSWhitespace verifies that xpathDefaultNamespace is
// whitespace-COLLAPSED before its sentinel/URI is resolved, so a value authored
// with surrounding whitespace (" ##targetNamespace ") still resolves like the
// sentinel — both at the schema root and on the assertion element itself.
func TestVersion11XPathDefaultNSWhitespace(t *testing.T) {
	t.Run("root-level xpathDefaultNamespace with surrounding whitespace", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="urn:t" elementFormDefault="qualified"
    xpathDefaultNamespace=" ##targetNamespace ">
  <xs:element name="outer">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="inner" type="xs:string"/>
      </xs:sequence>
      <xs:assert test="exists(inner)"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
		require.NoError(t, err)
		// The unprefixed `inner` in the assert resolves to urn:t (the collapsed
		// ##targetNamespace), matching the namespaced child.
		require.NoError(t, validateAssertion(t, schema, `<outer xmlns="urn:t"><inner>x</inner></outer>`))
	})

	t.Run("assertion-local xpathDefaultNamespace with surrounding whitespace", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="urn:t" elementFormDefault="qualified">
  <xs:element name="outer">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="inner" type="xs:string"/>
      </xs:sequence>
      <xs:assert test="exists(inner)" xpathDefaultNamespace=" ##targetNamespace "/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
		require.NoError(t, err)
		require.NoError(t, validateAssertion(t, schema, `<outer xmlns="urn:t"><inner>x</inner></outer>`))
	})
}

// TestVersion11ImportedXPathDefaultNS verifies that an IMPORTED schema's root
// xpathDefaultNamespace governs its own assertions, so an imported
// `xs:assert test="exists(child)"` with unprefixed names resolves namespaced
// children (PR859-01).
func TestVersion11ImportedXPathDefaultNS(t *testing.T) {
	imported := `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="urn:imp" elementFormDefault="qualified"
    xpathDefaultNamespace="##targetNamespace">
  <xs:element name="outer">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="inner" type="xs:string"/>
      </xs:sequence>
      <xs:assert test="exists(inner)"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	main := `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="urn:main" xmlns:imp="urn:imp" elementFormDefault="qualified">
  <xs:import namespace="urn:imp" schemaLocation="imported.xsd"/>
  <xs:element name="root" type="imp:outer"/>
</xs:schema>`
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "imported.xsd"), []byte(imported), 0o600))

	doc, err := helium.NewParser().Parse(t.Context(), []byte(main))
	require.NoError(t, err)
	schema, err := xsd.NewCompiler().Version(xsd.Version11).FS(os.DirFS(dir)).Compile(t.Context(), doc)
	require.NoError(t, err)
	// The imported assert's unprefixed `inner` resolves to urn:imp.
	require.NoError(t, validateAssertion(t, schema,
		`<root xmlns="urn:main"><inner xmlns="urn:imp">x</inner></root>`))
}

// TestVersion11SimpleContentInapplicableFacetRejected verifies that a synthetic
// simpleContent restriction type with a DIRECT sibling facet is facet-consistency
// checked: an inapplicable facet (xs:minInclusive on an xs:string base) is a
// compile error (PR859-02).
func TestVersion11SimpleContentInapplicableFacetRejected(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base">
    <xs:simpleContent>
      <xs:extension base="xs:string"/>
    </xs:simpleContent>
  </xs:complexType>
  <xs:element name="e">
    <xs:complexType>
      <xs:simpleContent>
        <xs:restriction base="base">
          <xs:minInclusive value="5"/>
        </xs:restriction>
      </xs:simpleContent>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	_, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
	require.ErrorIs(t, err, xsd.ErrCompilationFailed)
}

// TestVersion11UnionActiveListMemberValue verifies that union $value typing
// produces the list-item sequence when the active member is a LIST (PR859-03):
// here the union value "1 2" is active in IntList, so $value is two xs:int items
// (count = 2) rather than one untyped atomic.
func TestVersion11UnionActiveListMemberValue(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="IntList">
    <xs:list itemType="xs:int"/>
  </xs:simpleType>
  <xs:simpleType name="u">
    <xs:union memberTypes="IntList xs:string"/>
  </xs:simpleType>
  <xs:element name="e">
    <xs:simpleType>
      <xs:restriction base="u">
        <xs:assertion test="count($value) eq 2 and ($value[1] instance of xs:int)"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
	schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
	require.NoError(t, err)
	require.NoError(t, validateAssertion(t, schema, `<e>1 2</e>`))
}

// TestVersion11AssertCastUserQNameType verifies that `cast as t:QNameDerived`
// inside an assertion resolves the prefix via namespace context (PR859-04):
// the cast path, like castable, must be namespace-aware for QName-derived user
// types. The prefix is resolved against the assertion's STATIC namespaces (the
// schema's in-scope bindings), so p is declared on the schema, not the instance.
func TestVersion11AssertCastUserQNameType(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="urn:t" xmlns:t="urn:t" xmlns:p="urn:p" elementFormDefault="qualified">
  <xs:simpleType name="myQName">
    <xs:restriction base="xs:QName"/>
  </xs:simpleType>
  <xs:element name="e">
    <xs:complexType>
      <xs:attribute name="a" type="xs:string"/>
      <xs:assert test="namespace-uri-from-QName(@a cast as t:myQName) = 'urn:p'"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
	require.NoError(t, err)
	require.NoError(t, validateAssertion(t, schema, `<e xmlns="urn:t" a="p:y"/>`))
}

// TestVersion11SimpleContentNestedTypeKeepsBaseFacets verifies that a
// simpleContent restriction whose content is given by a nested <xs:simpleType>
// still enforces the BASE complex type's content facets through later derivation
// hops (PR859-CR15-01/REVIEW2-01). The base content is xs:string with
// maxLength=2; the derived restriction supplies a nested xs:string with
// minLength=1; the element extends that derived type. A 3-character value
// satisfies the nested minLength but MUST be rejected by the inherited maxLength=2
// — the nested type restricts, it does not replace, the base content type (XSD
// 3.4.2.2).
func TestVersion11SimpleContentNestedTypeKeepsBaseFacets(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base">
    <xs:simpleContent>
      <xs:restriction base="xs:anySimpleType">
        <xs:simpleType>
          <xs:restriction base="xs:string">
            <xs:maxLength value="2"/>
          </xs:restriction>
        </xs:simpleType>
      </xs:restriction>
    </xs:simpleContent>
  </xs:complexType>
  <xs:complexType name="mid">
    <xs:simpleContent>
      <xs:restriction base="base">
        <xs:simpleType>
          <xs:restriction base="xs:string">
            <xs:minLength value="1"/>
          </xs:restriction>
        </xs:simpleType>
      </xs:restriction>
    </xs:simpleContent>
  </xs:complexType>
  <xs:element name="e">
    <xs:complexType>
      <xs:simpleContent>
        <xs:extension base="mid"/>
      </xs:simpleContent>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
	require.NoError(t, err)
	require.NoError(t, validateAssertion(t, schema, `<e>ab</e>`))
	// "x" satisfies both the nested minLength=1 and the inherited maxLength=2.
	require.NoError(t, validateAssertion(t, schema, `<e>x</e>`))
	// "abc" satisfies the nested minLength=1 but violates the inherited maxLength=2.
	require.ErrorIs(t, validateAssertion(t, schema, `<e>abc</e>`), xsd.ErrValidationFailed)
}

// TestVersion11FixedSimpleContentQNameValueSpace verifies that the NON-EMPTY
// element FIXED-value comparison for a simpleContent complex type compares in the
// EFFECTIVE content simple type (PR859-XSD11-FIXED-SIMPLECONTENT-QNAME). The
// element's content type is a nested simpleContent restriction to xs:QName, so the
// fixed value "p:x" must match instance content "q:x" by QName VALUE-space equality
// (both prefixes bound to the SAME namespace), not by lexical string comparison
// against the outer complex type's base chain.
func TestVersion11FixedSimpleContentQNameValueSpace(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:p="urn:x">
  <xs:complexType name="base">
    <xs:simpleContent>
      <xs:extension base="xs:anySimpleType"/>
    </xs:simpleContent>
  </xs:complexType>
  <xs:element name="e" fixed="p:x">
    <xs:complexType>
      <xs:simpleContent>
        <xs:restriction base="base">
          <xs:simpleType>
            <xs:restriction base="xs:QName"/>
          </xs:simpleType>
        </xs:restriction>
      </xs:simpleContent>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
	require.NoError(t, err)
	// q:x with q bound to urn:x (same URI as p) — QName value-space equal to p:x.
	require.NoError(t, validateAssertion(t, schema, `<e xmlns:q="urn:x">q:x</e>`))
	// p:x verbatim is trivially equal.
	require.NoError(t, validateAssertion(t, schema, `<e xmlns:p="urn:x">p:x</e>`))
	// q:x with q bound to a DIFFERENT URI is not QName-equal — rejected.
	require.ErrorIs(t, validateAssertion(t, schema, `<e xmlns:q="urn:other">q:x</e>`), xsd.ErrValidationFailed)
}

// TestVersion11CompileFixedSimpleContentQNameValueSpace verifies the COMPILE-TIME
// content-model restriction check (restriction_particle.go NameAndTypeOK) compares
// an element's base/derived fixed values in the EFFECTIVE content simple type
// (PR859-CR18-01). The element's type is a simpleContent complex type narrowed (via
// a nested xs:simpleType) to xs:QName, whose OWN base chain does not reach xs:QName;
// without the centralized effectiveContentSimpleType in fixedValueMatches the
// derivation is rejected lexically. base fixed="p:x" and derived fixed="q:x" with
// both prefixes bound to the SAME URI is a valid restriction; a different-URI
// derived fixed is not.
func TestVersion11CompileFixedSimpleContentQNameValueSpace(t *testing.T) {
	schemaXML := func(derivedFixed, extraNS string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:p="urn:x" xmlns:q="urn:x" ` + extraNS + `>
  <xs:complexType name="anyBase">
    <xs:simpleContent>
      <xs:extension base="xs:anySimpleType"/>
    </xs:simpleContent>
  </xs:complexType>
  <xs:complexType name="qnameContent">
    <xs:simpleContent>
      <xs:restriction base="anyBase">
        <xs:simpleType>
          <xs:restriction base="xs:QName"/>
        </xs:simpleType>
      </xs:restriction>
    </xs:simpleContent>
  </xs:complexType>
  <xs:complexType name="B">
    <xs:sequence>
      <xs:element name="c" type="qnameContent" fixed="p:x"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="D">
    <xs:complexContent>
      <xs:restriction base="B">
        <xs:sequence>
          <xs:element name="c" type="qnameContent" fixed="` + derivedFixed + `"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root" type="D"/>
</xs:schema>`
	}
	// q:x (q bound to urn:x, same URI as p) is QName value-space equal to p:x —
	// the restriction is ACCEPTED at compile time.
	_, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML("q:x", ""))
	require.NoError(t, err)
	// z:x (z bound to a DIFFERENT URI) is not value-space equal — REJECTED.
	_, err = compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML("z:x", `xmlns:z="urn:other"`))
	require.Error(t, err)
}

// TestVersion11AssertCastInstanceBoundQName verifies that casting an
// ALREADY-RESOLVED xs:QName value (from data(@q)) to a QName-derived USER type
// succeeds even when the value's prefix is declared ONLY on the instance node and
// is absent from the assertion's static namespace map (PR859-CR17-01). The cast
// must validate using the value's own namespace URI, not by re-resolving the
// serialized `p:x` against the static map.
func TestVersion11AssertCastInstanceBoundQName(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="urn:t" xmlns:t="urn:t" elementFormDefault="qualified">
  <xs:simpleType name="myQName">
    <xs:restriction base="xs:QName"/>
  </xs:simpleType>
  <xs:element name="e">
    <xs:complexType>
      <xs:attribute name="q" type="xs:QName"/>
      <xs:assert test="namespace-uri-from-QName(data(@q) cast as t:myQName) = 'urn:p'"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
	require.NoError(t, err)
	// p is declared on the INSTANCE only; the cast must still resolve to urn:p.
	require.NoError(t, validateAssertion(t, schema, `<e xmlns="urn:t" xmlns:p="urn:p" q="p:x"/>`))
}

// TestVersion11AssertXMLPrefixQName verifies that an xs:QName VALUE using the
// predeclared `xml` prefix atomizes correctly in an assertion (PR859-CR17-02):
// `xml` need not be bound on any node, matching XSD validation which accepts
// xml:lang/xml:space as xs:QName.
func TestVersion11AssertXMLPrefixQName(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e">
    <xs:complexType>
      <xs:attribute name="q" type="xs:QName"/>
      <xs:assert test="namespace-uri-from-QName(data(@q)) = 'http://www.w3.org/XML/1998/namespace'"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
	require.NoError(t, err)
	require.NoError(t, validateAssertion(t, schema, `<e q="xml:space"/>`))
}

// TestVersion11AssertListOfQName verifies that an xs:list whose itemType is
// xs:QName atomizes namespace-aware in an assertion (PR859-CR17-03): each list
// token resolves against the node's in-scope namespaces, so count(data(@qs)) and
// per-item namespace resolution work instead of failing the context-free cast.
func TestVersion11AssertListOfQName(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="qnames">
    <xs:list itemType="xs:QName"/>
  </xs:simpleType>
  <xs:element name="e">
    <xs:complexType>
      <xs:attribute name="qs" type="qnames"/>
      <xs:assert test="count(data(@qs)) = 2 and namespace-uri-from-QName(data(@qs)[1]) = 'urn:p'"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
	require.NoError(t, err)
	require.NoError(t, validateAssertion(t, schema, `<e xmlns:p="urn:p" qs="p:a p:b"/>`))
}

// TestVersion11AssertInlineAnonQNameList verifies that an INLINE ANONYMOUS
// xs:list itemType="xs:QName" preserves its list-item metadata for assert node
// atomization (PR859-CR19-01): the anonymous type has no schema-table name, so it
// is registered under a synthetic annotation name and schemaDecls recovers the
// item type. count(data(@qs)) is 2, and each item resolves namespace-aware.
func TestVersion11AssertInlineAnonQNameList(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="c">
          <xs:complexType>
            <xs:attribute name="qs">
              <xs:simpleType>
                <xs:list itemType="xs:QName"/>
              </xs:simpleType>
            </xs:attribute>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
      <xs:assert test="count(data(c/@qs)) = 2 and namespace-uri-from-QName(data(c/@qs)[2]) = 'urn:p'"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
	require.NoError(t, err)
	require.NoError(t, validateAssertion(t, schema, `<e xmlns:p="urn:p"><c qs="p:a p:b"/></e>`))
}

// TestVersion11AssertInlineAnonQNameUnion verifies that an INLINE ANONYMOUS
// xs:union memberTypes="xs:QName xs:string" preserves its union-member metadata
// for assert node atomization (PR859-CR19-01): without it the anonymous type
// collapses to xs:anyType and a QName-valued union node fails XPTY0004.
func TestVersion11AssertInlineAnonQNameUnion(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e">
    <xs:complexType>
      <xs:attribute name="q">
        <xs:simpleType>
          <xs:union memberTypes="xs:QName xs:string"/>
        </xs:simpleType>
      </xs:attribute>
      <xs:assert test="namespace-uri-from-QName(data(@q)) = 'urn:p'"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
	require.NoError(t, err)
	require.NoError(t, validateAssertion(t, schema, `<e xmlns:p="urn:p" q="p:x"/>`))
}

func TestVersion11AssertInlineAnonUnionMembersThroughRestriction(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e">
    <xs:complexType>
      <xs:attribute name="u">
        <xs:simpleType>
          <xs:restriction>
            <xs:simpleType>
              <xs:union memberTypes="xs:string">
                <xs:simpleType>
                  <xs:restriction base="xs:int">
                    <xs:maxInclusive value="10"/>
                  </xs:restriction>
                </xs:simpleType>
              </xs:union>
            </xs:simpleType>
          </xs:restriction>
        </xs:simpleType>
      </xs:attribute>
      <xs:assert test="data(@u) instance of xs:string"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
	require.NoError(t, err)
	require.NoError(t, validateAssertion(t, schema, `<e u="20"/>`))
	require.ErrorIs(t, validateAssertion(t, schema, `<e u="5"/>`), xsd.ErrValidationFailed)
}

// TestVersion11AssertUnionActiveMemberList verifies that assert NODE atomization
// resolves a union's ACTIVE member value-dependently (PR859-CR19-02): for
// memberTypes="IntList xs:string" with value "1 2" the active member is the LIST,
// so data(@u) is two xs:int items, not one string. (Distinct from the $value path,
// which was already correct.)
func TestVersion11AssertUnionActiveMemberList(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="IntList">
    <xs:list itemType="xs:int"/>
  </xs:simpleType>
  <xs:simpleType name="u">
    <xs:union memberTypes="IntList xs:string"/>
  </xs:simpleType>
  <xs:element name="e">
    <xs:complexType>
      <xs:attribute name="u" type="u"/>
      <xs:assert test="count(data(@u)) = 2 and (data(@u)[1] instance of xs:int)"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
	require.NoError(t, err)
	require.NoError(t, validateAssertion(t, schema, `<e u="1 2"/>`))
	// A string-only value resolves to the xs:string member: one atom, not a list.
	const schemaXML2 = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="IntList">
    <xs:list itemType="xs:int"/>
  </xs:simpleType>
  <xs:simpleType name="u">
    <xs:union memberTypes="IntList xs:string"/>
  </xs:simpleType>
  <xs:element name="e">
    <xs:complexType>
      <xs:attribute name="u" type="u"/>
      <xs:assert test="count(data(@u)) = 1"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	schema2, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML2)
	require.NoError(t, err)
	require.NoError(t, validateAssertion(t, schema2, `<e u="not ints here"/>`))
}

// TestVersion11AssertInlineAnonQNameDescendantElement verifies the PR859-CR19
// metadata fixes apply to a DESCENDANT ELEMENT (not just an attribute): a child
// element typed by an inline anonymous xs:list itemType="xs:QName" atomizes to its
// per-item QName sequence via data(c).
func TestVersion11AssertInlineAnonQNameDescendantElement(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="c">
          <xs:simpleType>
            <xs:list itemType="xs:QName"/>
          </xs:simpleType>
        </xs:element>
      </xs:sequence>
      <xs:assert test="count(data(c)) = 2 and namespace-uri-from-QName(data(c)[1]) = 'urn:p'"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
	require.NoError(t, err)
	require.NoError(t, validateAssertion(t, schema, `<e xmlns:p="urn:p"><c>p:a p:b</c></e>`))
}

// TestVersion11AssertUnionActiveMemberFacet verifies that assert NODE atomization
// selects a union's ACTIVE member with FULL schema-aware validation, not just a
// lexical cast (PR859-CR20-01). The first member ExactlyTwoInts is an xs:list of
// xs:int restricted to length=2; for value "1 2 3" every token casts to xs:int but
// the LIST LENGTH facet fails, so the active member must be the later xs:string —
// data(@u) is one string, not three ints.
func TestVersion11AssertUnionActiveMemberFacet(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="ExactlyTwoInts">
    <xs:restriction>
      <xs:simpleType>
        <xs:list itemType="xs:int"/>
      </xs:simpleType>
      <xs:length value="2"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="u">
    <xs:union memberTypes="ExactlyTwoInts xs:string"/>
  </xs:simpleType>
  <xs:element name="e">
    <xs:complexType>
      <xs:attribute name="u" type="u"/>
      <xs:assert test="count(data(@u)) = 1 and (data(@u) instance of xs:string)"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
	require.NoError(t, err)
	// "1 2 3" fails the length=2 facet of the list member, so xs:string is active.
	require.NoError(t, validateAssertion(t, schema, `<e u="1 2 3"/>`))
	// "1 2" satisfies the length=2 list member, which is active → two xs:int items.
	const schemaXML2 = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="ExactlyTwoInts">
    <xs:restriction>
      <xs:simpleType>
        <xs:list itemType="xs:int"/>
      </xs:simpleType>
      <xs:length value="2"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="u">
    <xs:union memberTypes="ExactlyTwoInts xs:string"/>
  </xs:simpleType>
  <xs:element name="e">
    <xs:complexType>
      <xs:attribute name="u" type="u"/>
      <xs:assert test="count(data(@u)) = 2 and (data(@u)[1] instance of xs:int)"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	schema2, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML2)
	require.NoError(t, err)
	require.NoError(t, validateAssertion(t, schema2, `<e u="1 2"/>`))
}

// TestVersion11AssertNestedUnionActiveLeaf verifies that assert NODE atomization
// descends through NESTED unions to the active LEAF member (PR859-CR21), matching
// $value (fixedUnionActiveMember recurses). Outer = union(Inner, xs:boolean);
// Inner = union(IntList, xs:string); value "1 2" → the active leaf is IntList, so
// data(@u) is two xs:int (not one atomic value typed as the direct member Inner).
func TestVersion11AssertNestedUnionActiveLeaf(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="IntList">
    <xs:list itemType="xs:int"/>
  </xs:simpleType>
  <xs:simpleType name="Inner">
    <xs:union memberTypes="IntList xs:string"/>
  </xs:simpleType>
  <xs:simpleType name="Outer">
    <xs:union memberTypes="Inner xs:boolean"/>
  </xs:simpleType>
  <xs:element name="e">
    <xs:complexType>
      <xs:attribute name="u" type="Outer"/>
      <xs:assert test="count(data(@u)) = 2 and (data(@u)[1] instance of xs:int)"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
	require.NoError(t, err)
	require.NoError(t, validateAssertion(t, schema, `<e u="1 2"/>`))
	// "true" resolves through Inner's xs:string member first (string accepts "true"
	// before xs:boolean is reached at the Outer level): one atomic value, count 1.
	const schemaXML2 = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="IntList">
    <xs:list itemType="xs:int"/>
  </xs:simpleType>
  <xs:simpleType name="Inner">
    <xs:union memberTypes="IntList xs:string"/>
  </xs:simpleType>
  <xs:simpleType name="Outer">
    <xs:union memberTypes="Inner xs:boolean"/>
  </xs:simpleType>
  <xs:element name="e">
    <xs:complexType>
      <xs:attribute name="u" type="Outer"/>
      <xs:assert test="count(data(@u)) = 1 and (data(@u) instance of xs:string)"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	schema2, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML2)
	require.NoError(t, err)
	require.NoError(t, validateAssertion(t, schema2, `<e u="hello world"/>`))
}

// TestVersion11AssertListNBSPToken verifies schema-aware list atomization splits on
// XSD whitespace ONLY (space/tab/CR/LF), not on NBSP or other Unicode whitespace
// (PR859-REV-F01), matching validation/$value. A one-item xs:list value whose sole
// token contains an NBSP stays a SINGLE atom, so count(data(@v)) = 1.
func TestVersion11AssertListNBSPToken(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="strList">
    <xs:list itemType="xs:string"/>
  </xs:simpleType>
  <xs:element name="e">
    <xs:complexType>
      <xs:attribute name="v" type="strList"/>
      <xs:assert test="count(data(@v)) = 1"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
	require.NoError(t, err)
	// "a b" is one xs:string list item (NBSP is not XSD whitespace).
	require.NoError(t, validateAssertion(t, schema, "<e v=\"a b\"/>"))
	// A real space-separated value is two items, confirming the tokenizer still works.
	const schemaXML2 = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="strList">
    <xs:list itemType="xs:string"/>
  </xs:simpleType>
  <xs:element name="e">
    <xs:complexType>
      <xs:attribute name="v" type="strList"/>
      <xs:assert test="count(data(@v)) = 2"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	schema2, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML2)
	require.NoError(t, err)
	require.NoError(t, validateAssertion(t, schema2, `<e v="a b"/>`))
}

// TestVersion11AssertUserNumericListItem verifies a list whose item type is a
// USER-defined numeric type (derived from xs:int) atomizes to numeric atoms usable
// by sum() (PR859-REV-F02): the token is cast through the item type's built-in base
// (xs:int) and typed as the user type, instead of stored as an untyped string.
func TestVersion11AssertUserNumericListItem(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="MyInt">
    <xs:restriction base="xs:int"/>
  </xs:simpleType>
  <xs:simpleType name="MyIntList">
    <xs:list itemType="MyInt"/>
  </xs:simpleType>
  <xs:element name="e">
    <xs:complexType>
      <xs:attribute name="v" type="MyIntList"/>
      <xs:assert test="sum(data(@v)) = 6 and (data(@v)[1] instance of xs:int)"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
	require.NoError(t, err)
	require.NoError(t, validateAssertion(t, schema, `<e v="1 2 3"/>`))
}

// TestVersion11AssertInstanceOfAnonListType verifies that an INLINE ANONYMOUS list
// type (recorded under a synthetic assert annotation name) participates in subtype
// checks (PR859-REV-F03): schemaDecls.IsSubtypeOf resolves the synthetic name via
// the anonymous-type registry, so an instance-of test against the node's type holds.
func TestVersion11AssertInstanceOfAnonListType(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e">
    <xs:complexType>
      <xs:attribute name="v">
        <xs:simpleType>
          <xs:list itemType="xs:int"/>
        </xs:simpleType>
      </xs:attribute>
      <xs:assert test="@v instance of attribute(v, xs:anySimpleType)"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
	require.NoError(t, err)
	require.NoError(t, validateAssertion(t, schema, `<e v="1 2 3"/>`))
}

// TestVersion11AssertUnionAnonAtomicMemberFacet verifies that an inline ANONYMOUS
// ATOMIC union member's facets are honored during assert NODE active-member
// selection (PR859-REV-F04). The first member is an anonymous xs:int restricted to
// maxInclusive=10, followed by xs:string. Value "20" exceeds the facet so the active
// member must be xs:string (data(@u) instance of xs:string, count 1); value "5"
// satisfies the faceted int, so it is the active member.
func TestVersion11AssertUnionAnonAtomicMemberFacet(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="u">
    <xs:union>
      <xs:simpleType>
        <xs:restriction base="xs:int">
          <xs:maxInclusive value="10"/>
        </xs:restriction>
      </xs:simpleType>
      <xs:simpleType>
        <xs:restriction base="xs:string"/>
      </xs:simpleType>
    </xs:union>
  </xs:simpleType>
  <xs:element name="e">
    <xs:complexType>
      <xs:attribute name="u" type="u"/>
      <xs:assert test="count(data(@u)) = 1 and (data(@u) instance of xs:string)"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
	require.NoError(t, err)
	// "20" exceeds maxInclusive=10 on the int member → active member is xs:string.
	require.NoError(t, validateAssertion(t, schema, `<e u="20"/>`))
	// "5" satisfies the faceted int member → active member is the int.
	const schemaXML2 = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="u">
    <xs:union>
      <xs:simpleType>
        <xs:restriction base="xs:int">
          <xs:maxInclusive value="10"/>
        </xs:restriction>
      </xs:simpleType>
      <xs:simpleType>
        <xs:restriction base="xs:string"/>
      </xs:simpleType>
    </xs:union>
  </xs:simpleType>
  <xs:element name="e">
    <xs:complexType>
      <xs:attribute name="u" type="u"/>
      <xs:assert test="count(data(@u)) = 1 and (data(@u) instance of xs:int)"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	schema2, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML2)
	require.NoError(t, err)
	require.NoError(t, validateAssertion(t, schema2, `<e u="5"/>`))
}

// TestVersion11AssertUnionEmptyListMember verifies that an EMPTY list value selects
// a list union member during assert NODE atomization (PR859-REVIEW-01): a plain list
// accepts the empty list, so data(@u) is the empty sequence (count 0, empty() true),
// matching validation/$value — instead of falling through to a later xs:string member.
// A minLength>0 list facet pushes the empty value to the later member instead.
func TestVersion11AssertUnionEmptyListMember(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="IntList">
    <xs:list itemType="xs:int"/>
  </xs:simpleType>
  <xs:simpleType name="u">
    <xs:union memberTypes="IntList xs:string"/>
  </xs:simpleType>
  <xs:element name="e">
    <xs:complexType>
      <xs:attribute name="u" type="u"/>
      <xs:assert test="empty(data(@u)) and count(data(@u)) = 0"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
	require.NoError(t, err)
	// "" is a valid empty IntList → active member is the list → empty sequence.
	require.NoError(t, validateAssertion(t, schema, `<e u=""/>`))

	// With minLength=1 on the list member, the empty value cannot be the list, so the
	// active member is xs:string → one (empty-string) atom, count 1.
	const schemaXML2 = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="IntList">
    <xs:restriction>
      <xs:simpleType><xs:list itemType="xs:int"/></xs:simpleType>
      <xs:minLength value="1"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="u">
    <xs:union memberTypes="IntList xs:string"/>
  </xs:simpleType>
  <xs:element name="e">
    <xs:complexType>
      <xs:attribute name="u" type="u"/>
      <xs:assert test="count(data(@u)) = 1 and (data(@u) instance of xs:string)"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	schema2, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML2)
	require.NoError(t, err)
	require.NoError(t, validateAssertion(t, schema2, `<e u=""/>`))
}

// TestVersion11AssertCastMultiItemTypedValue verifies that cast/castable/arithmetic
// atomize a singleton OPERAND through the typed-value stream (PR859-001): a node whose
// schema typed value is a MULTI-item list/union-list is seen as multiple atoms, so
// `castable as <singleton>` is false and `cast as <singleton>` is a cardinality error,
// matching data() (which already expanded). Single-item typed values still cast.
func TestVersion11AssertCastMultiItemTypedValue(t *testing.T) {
	// Direct list-typed attribute.
	const listSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="IntList">
    <xs:list itemType="xs:int"/>
  </xs:simpleType>
  <xs:element name="e">
    <xs:complexType>
      <xs:attribute name="u" type="IntList"/>
      <xs:assert test="%s"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	// union(IntList, xs:string) attribute (active member is the list for "1 2").
	const unionSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="IntList">
    <xs:list itemType="xs:int"/>
  </xs:simpleType>
  <xs:simpleType name="u">
    <xs:union memberTypes="IntList xs:string"/>
  </xs:simpleType>
  <xs:element name="e">
    <xs:complexType>
      <xs:attribute name="u" type="u"/>
      <xs:assert test="%s"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	run := func(t *testing.T, tmpl, test, instance string) {
		t.Helper()
		schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), fmt.Sprintf(tmpl, test))
		require.NoError(t, err)
		require.NoError(t, validateAssertion(t, schema, instance))
	}

	// Multi-item typed value: NOT castable to a single xs:string; the cardinality
	// makes `cast as xs:string` raise (so the assert uses castable / not()).
	run(t, listSchema, `not(@u castable as xs:string)`, `<e u="1 2"/>`)
	run(t, unionSchema, `not(@u castable as xs:string)`, `<e u="1 2"/>`)
	// Single-item typed value IS castable (and equals the value).
	run(t, listSchema, `@u castable as xs:int and (@u cast as xs:int) eq 7`, `<e u="7"/>`)
	run(t, unionSchema, `@u castable as xs:int and (@u cast as xs:int) eq 7`, `<e u="7"/>`)
	// data() agreement (multi-item) — sanity that the same node expands consistently.
	run(t, listSchema, `count(data(@u)) = 2`, `<e u="1 2"/>`)
}

// TestVersion11AssertSimpleContentChildAtomization verifies that data(c) on a
// DESCENDANT element c whose simpleContent COMPLEX type narrows its content to
// xs:QName / a list / a union atomizes THROUGH the narrowed content type
// (PR859-FINAL-01), agreeing with $value — instead of through the complex type's raw
// base. schemaDecls resolves the simpleContent complex type via
// effectiveContentSimpleType before returning atomization metadata.
func TestVersion11AssertSimpleContentChildAtomization(t *testing.T) {
	// (a) simpleContent narrowed to xs:QName: data(c) must be a QName, so
	// namespace-uri-from-QName resolves the instance prefix.
	const qnameSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="qnameContent">
    <xs:simpleContent>
      <xs:extension base="xs:QName"/>
    </xs:simpleContent>
  </xs:complexType>
  <xs:element name="e">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="c" type="qnameContent"/>
      </xs:sequence>
      <xs:assert test="namespace-uri-from-QName(data(c)) = 'urn:p'"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), qnameSchema)
	require.NoError(t, err)
	require.NoError(t, validateAssertion(t, schema, `<e xmlns:p="urn:p"><c>p:x</c></e>`))

	// (b) simpleContent whose content is a list of xs:int: data(c) is the item
	// sequence, so count and numeric typing hold.
	const listSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="IntList">
    <xs:list itemType="xs:int"/>
  </xs:simpleType>
  <xs:complexType name="listContent">
    <xs:simpleContent>
      <xs:extension base="IntList"/>
    </xs:simpleContent>
  </xs:complexType>
  <xs:element name="e">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="c" type="listContent"/>
      </xs:sequence>
      <xs:assert test="count(data(c)) = 3 and sum(data(c)) = 6"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	schema2, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), listSchema)
	require.NoError(t, err)
	require.NoError(t, validateAssertion(t, schema2, `<e><c>1 2 3</c></e>`))

	// (c) simpleContent whose content is a union(IntList, xs:string): data(c) for
	// "1 2" resolves the active LIST member → two xs:int.
	const unionSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="IntList">
    <xs:list itemType="xs:int"/>
  </xs:simpleType>
  <xs:simpleType name="u">
    <xs:union memberTypes="IntList xs:string"/>
  </xs:simpleType>
  <xs:complexType name="unionContent">
    <xs:simpleContent>
      <xs:extension base="u"/>
    </xs:simpleContent>
  </xs:complexType>
  <xs:element name="e">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="c" type="unionContent"/>
      </xs:sequence>
      <xs:assert test="count(data(c)) = 2 and (data(c)[1] instance of xs:int)"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	schema3, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), unionSchema)
	require.NoError(t, err)
	require.NoError(t, validateAssertion(t, schema3, `<e><c>1 2</c></e>`))
}

// TestVersion11AssertDeepNestedUnionList verifies that assert NODE atomization
// descends an ARBITRARILY-DEEP (here 80 > the old depth-64 cap) acyclic nested-union
// chain to the active LIST leaf (PR859-REVIEW-02), matching validation/$value. The
// chain U1=union(U2,string), …, U80=union(IntList,string); value "1 2" must reach
// IntList (two xs:int), not stop at the cap and fall back to a xs:string member.
func TestVersion11AssertDeepNestedUnionList(t *testing.T) {
	const depth = 80
	var b strings.Builder
	b.WriteString(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">` + "\n")
	b.WriteString(`  <xs:simpleType name="IntList"><xs:list itemType="xs:int"/></xs:simpleType>` + "\n")
	for i := depth; i >= 1; i-- {
		member := "IntList"
		if i < depth {
			member = fmt.Sprintf("U%d", i+1)
		}
		fmt.Fprintf(&b, `  <xs:simpleType name="U%d"><xs:union memberTypes="%s xs:string"/></xs:simpleType>`+"\n", i, member)
	}
	b.WriteString(`  <xs:element name="e"><xs:complexType><xs:attribute name="u" type="U1"/>` + "\n")
	b.WriteString(`    <xs:assert test="count(data(@u)) = 2 and (data(@u)[1] instance of xs:int)"/>` + "\n")
	b.WriteString(`  </xs:complexType></xs:element>` + "\n")
	b.WriteString(`</xs:schema>`)
	schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), b.String())
	require.NoError(t, err)
	require.NoError(t, validateAssertion(t, schema, `<e u="1 2"/>`))
}

// TestVersion11AssertSimpleContentRestrictionUnionFacet verifies that a simpleContent
// RESTRICTION over a UNION base WITH A DIRECT FACET keeps its union metadata for assert
// node atomization (PR859-REVIEW-02 / round-29). effectiveContentSimpleType builds a
// synthetic facet-only simple type whose union variety/members live only up its base
// chain, so UnionMemberTypes must resolve via resolveVariety/resolveUnionMembers (not
// the direct fields). For value "1 2" the active member is the list IntList →
// count(data(c)) = 2 (agreeing with $value), not a single atom.
func TestVersion11AssertSimpleContentRestrictionUnionFacet(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="IntList">
    <xs:list itemType="xs:int"/>
  </xs:simpleType>
  <xs:simpleType name="u">
    <xs:union memberTypes="IntList xs:string"/>
  </xs:simpleType>
  <xs:complexType name="uBase">
    <xs:simpleContent>
      <xs:extension base="u"/>
    </xs:simpleContent>
  </xs:complexType>
  <xs:complexType name="uContent">
    <xs:simpleContent>
      <xs:restriction base="uBase">
        <xs:pattern value=".*"/>
      </xs:restriction>
    </xs:simpleContent>
  </xs:complexType>
  <xs:element name="e">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="c" type="uContent"/>
      </xs:sequence>
      <xs:assert test="count(data(c)) = 2 and (data(c)[1] instance of xs:int)"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
	require.NoError(t, err)
	require.NoError(t, validateAssertion(t, schema, `<e><c>1 2</c></e>`))
}

// TestVersion11AssertDescendantDefaultValue verifies that an xs:assert on a PARENT
// sees the schema DEFAULT/FIXED effective value of an EMPTY descendant element when
// it atomizes data(c) (round-30). isolatedAssertTree materializes the recorded
// effective value onto the isolated copy, so data(c) is the schema-normalized default
// (typed via the child's annotation), matching the child's own $value.
func TestVersion11AssertDescendantDefaultValue(t *testing.T) {
	// (a) xs:integer default="5": data(c) eq 5 for an empty <c/>.
	const intSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="c" type="xs:integer" default="5"/>
      </xs:sequence>
      <xs:assert test="data(c) eq 5"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), intSchema)
	require.NoError(t, err)
	require.NoError(t, validateAssertion(t, schema, `<root><c/></root>`))

	// (b) xs:QName default whose prefix is bound in the DECLARATION's context (on the
	// schema root). The empty <c/> must atomize data(c) as the QName p:x → urn:p.
	const qnameSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:p="urn:p">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="c" type="xs:QName" default="p:x"/>
      </xs:sequence>
      <xs:assert test="namespace-uri-from-QName(data(c)) = 'urn:p'"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	schema2, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), qnameSchema)
	require.NoError(t, err)
	// The instance does NOT bind p; the default resolves via the declaration context.
	require.NoError(t, validateAssertion(t, schema2, `<root><c/></root>`))
}

// TestVersion11AssertDescendantQNameDefaultPrefixCollision verifies that a defaulted
// descendant QName element materialized into the isolated assert tree resolves to its
// DECLARATION-ns URI even when the instance already binds the schema's default-value
// prefix to a DIFFERENT URI (PR859-F01). The schema declares default="p:x" with p →
// urn:decl; the instance binds p → urn:other. The materialized value must resolve to
// urn:decl (a fresh prefix is minted), so namespace-uri-from-QName(data(c)) = urn:decl.
func TestVersion11AssertDescendantQNameDefaultPrefixCollision(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:p="urn:decl">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="c" type="xs:QName" default="p:x"/>
      </xs:sequence>
      <xs:assert test="namespace-uri-from-QName(data(c)) = 'urn:decl'"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
	require.NoError(t, err)
	// The instance binds p to a DIFFERENT URI; the empty <c/> default must still
	// resolve via the declaration's p → urn:decl, not the instance's urn:other.
	require.NoError(t, validateAssertion(t, schema, `<root xmlns:p="urn:other"><c/></root>`))
}

// TestVersion11AssertListOfUnionPerToken verifies that an xs:list whose ITEM TYPE is
// a UNION atomizes each token through its OWN active union member (PR859-F02), not one
// static base. itemType="intOrBool" (union of xs:int, xs:boolean); value "1 true 2"
// must atomize to xs:int, xs:boolean, xs:int — agreeing with $value.
func TestVersion11AssertListOfUnionPerToken(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="intOrBool">
    <xs:union memberTypes="xs:int xs:boolean"/>
  </xs:simpleType>
  <xs:simpleType name="iobList">
    <xs:list itemType="intOrBool"/>
  </xs:simpleType>
  <xs:element name="e">
    <xs:complexType>
      <xs:attribute name="v" type="iobList"/>
      <xs:assert test="count(data(@v)) = 3 and (data(@v)[1] instance of xs:int) and (data(@v)[2] instance of xs:boolean) and (data(@v)[3] instance of xs:int)"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
	require.NoError(t, err)
	require.NoError(t, validateAssertion(t, schema, `<e v="1 true 2"/>`))
}

// TestVersion11AssertSimpleContentComplexNotSimpleTarget verifies that a simpleContent
// COMPLEX type is NOT accepted as a simple/atomic target (PR859-F03): a literal value
// `castable as` the complex type is false (validateCast rejects a complex target), and
// `t:c instance of element(*, xs:anySimpleType)` is false — a complex type is not a
// subtype of xs:anySimpleType — even though data() atomization still resolves its
// content type.
func TestVersion11AssertSimpleContentComplexNotSimpleTarget(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="urn:t" xmlns:t="urn:t" elementFormDefault="qualified">
  <xs:complexType name="ccType">
    <xs:simpleContent>
      <xs:extension base="xs:string"/>
    </xs:simpleContent>
  </xs:complexType>
  <xs:element name="e">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="c" type="t:ccType"/>
      </xs:sequence>
      <xs:assert test="not(t:c instance of element(*, xs:anySimpleType)) and not('hi' castable as t:ccType) and (data(t:c) = 'hi')"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
	require.NoError(t, err)
	require.NoError(t, validateAssertion(t, schema, `<e xmlns="urn:t"><c>hi</c></e>`))
}

// TestVersion11AssertSimpleContentComplexNotSimpleBaseSubtype verifies that a
// simpleContent COMPLEX type is NOT a subtype of its direct SIMPLE base — neither a
// builtin (xs:string) nor a user-defined simple type (t:strRestr) — for instance-of
// (PR859-01): the base-chain walk must not match a non-complex base ancestor. data()
// still atomizes through the narrowed content type.
func TestVersion11AssertSimpleContentComplexNotSimpleBaseSubtype(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="urn:t" xmlns:t="urn:t" elementFormDefault="qualified">
  <xs:simpleType name="strRestr">
    <xs:restriction base="xs:string"/>
  </xs:simpleType>
  <xs:complexType name="ccBuiltinType">
    <xs:simpleContent>
      <xs:extension base="xs:string"/>
    </xs:simpleContent>
  </xs:complexType>
  <xs:complexType name="ccUserType">
    <xs:simpleContent>
      <xs:extension base="t:strRestr"/>
    </xs:simpleContent>
  </xs:complexType>
  <xs:element name="e">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="cBuiltin" type="t:ccBuiltinType"/>
        <xs:element name="cUser" type="t:ccUserType"/>
      </xs:sequence>
      <xs:assert test="not(t:cBuiltin instance of element(*, xs:string)) and not(t:cUser instance of element(*, t:strRestr)) and not(t:cUser instance of element(*, xs:string)) and (data(t:cBuiltin) = 'hi') and (data(t:cUser) = 'hi')"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
	require.NoError(t, err)
	require.NoError(t, validateAssertion(t, schema, `<e xmlns="urn:t"><cBuiltin>hi</cBuiltin><cUser>hi</cUser></e>`))
}

// TestVersion11AssertionFacetValuePreservesUserType verifies that the $value binding
// of an xs:assertion simple-type facet PRESERVES a named user-defined atomic type's
// identity (PR859-REVIEW-01) instead of collapsing it to its builtin base: $value of
// type t:MyInt must satisfy `$value instance of t:MyInt`, agreeing with schema-aware
// data() atomization. Without the fix $value is typed xs:int and the instance-of test
// (a narrower user type) is false.
func TestVersion11AssertionFacetValuePreservesUserType(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:simpleType name="MyInt">
    <xs:restriction base="xs:int">
      <xs:assertion test="$value instance of t:MyInt"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="e" type="t:MyInt"/>
</xs:schema>`
	schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
	require.NoError(t, err)
	require.NoError(t, validateAssertion(t, schema, `<e xmlns="urn:t">5</e>`))
}

// TestVersion11AssertUnionOfListOfUnionPerToken verifies that a UNION whose ACTIVE
// member is a LIST whose item type is itself a UNION atomizes each list token through
// its OWN active union member (PR859-REVIEW-02), not the single static list-item base.
// outer = union(iobList, xs:string); iobList = list(intOrBool); intOrBool =
// union(xs:int, xs:boolean). value "1 true 2" must atomize to xs:int, xs:boolean,
// xs:int — agreeing with $value. Without the fix data(@v)[2] is mis-typed.
func TestVersion11AssertUnionOfListOfUnionPerToken(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="intOrBool">
    <xs:union memberTypes="xs:int xs:boolean"/>
  </xs:simpleType>
  <xs:simpleType name="iobList">
    <xs:list itemType="intOrBool"/>
  </xs:simpleType>
  <xs:simpleType name="outer">
    <xs:union memberTypes="iobList xs:string"/>
  </xs:simpleType>
  <xs:element name="e">
    <xs:complexType>
      <xs:attribute name="v" type="outer"/>
      <xs:assert test="count(data(@v)) = 3 and (data(@v)[1] instance of xs:int) and (data(@v)[2] instance of xs:boolean) and (data(@v)[3] instance of xs:int)"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	schema, err := compileAssertion(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML)
	require.NoError(t, err)
	require.NoError(t, validateAssertion(t, schema, `<e v="1 true 2"/>`))
}
