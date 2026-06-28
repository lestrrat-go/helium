package xsd_test

import (
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
