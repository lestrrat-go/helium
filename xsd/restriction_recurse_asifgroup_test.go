package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// compileFatalErrorsVersion compiles a schema at the requested XSD version and
// returns only the formatted fatal compile errors (warnings stripped).
func compileFatalErrorsVersion(t *testing.T, version xsd.Version, schemaXML string) string {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	_, _ = xsd.NewCompiler().Version(version).Label("test.xsd").ErrorHandler(collector).Compile(t.Context(), doc)
	_, errors := partitionCompileErrors(collector.Errors())
	return errors
}

// TestRecurseAsIfGroupWrapperOccurrence covers §3.9.6 RecurseAsIfGroup: a derived
// ELEMENT particle R restricting a base MODEL GROUP B is checked by treating R as
// if wrapped in a singleton group whose occurrence is {1,1} — NOT R's own
// occurrence. The wrapper {1,1} must validly restrict the base group's occurrence;
// R's full occurrence (and type) is enforced by the inner per-member/branch match.
// All four accept cases mirror W3C xsd10 particles tests that are expected VALID
// but were false-rejected when the outer check compared R's own occurrence against
// the base group's. The two reject guards must stay REJECTED.
func TestRecurseAsIfGroupWrapperOccurrence(t *testing.T) {
	t.Parallel()

	const notValidRestriction = "not a valid restriction"

	// particlesHa022: derived element a? {0,1} maps (via the outer recurse) onto a
	// base nested sequence(a?){1,1}. The wrapper {1,1} restricts the base {1,1}; the
	// inner a?{0,1} restricts base member a?{0,1}. VALID.
	t.Run("Ha022 element {0,1} restricts base sequence(a?){1,1}", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base">
    <xs:sequence>
      <xs:any namespace="##any"/>
      <xs:sequence>
        <xs:element name="a" minOccurs="0"/>
      </xs:sequence>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="derived">
    <xs:complexContent>
      <xs:restriction base="base">
        <xs:sequence>
          <xs:element name="a" type="xs:string"/>
          <xs:element name="a" type="xs:string" minOccurs="0"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`
		require.Empty(t, compileFatalErrorsVersion(t, xsd.Version10, schema))
		require.Empty(t, compileFatalErrorsVersion(t, xsd.Version11, schema))
	})

	// particlesHb010: derived choice(e1?) reduces to element e1{0,1} mapped onto
	// base sequence(e1?, e2?){1,1}. Wrapper {1,1} restricts base {1,1}; inner e1?
	// restricts base e1?; base e2? is skipped by the emptiable-member skip. VALID.
	t.Run("Hb010 choice(e1?) restricts base sequence(e1?,e2?){1,1}", func(t *testing.T) {
		t.Parallel()
		schema := `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema" targetNamespace="http://xsdtesting" xmlns:x="http://xsdtesting" elementFormDefault="qualified">
  <xsd:complexType name="base">
    <xsd:sequence>
      <xsd:element name="e1" minOccurs="0"/>
      <xsd:element name="e2" minOccurs="0"/>
    </xsd:sequence>
  </xsd:complexType>
  <xsd:element name="doc">
    <xsd:complexType>
      <xsd:complexContent>
        <xsd:restriction base="x:base">
          <xsd:choice>
            <xsd:element name="e1" minOccurs="0"/>
          </xsd:choice>
        </xsd:restriction>
      </xsd:complexContent>
    </xsd:complexType>
  </xsd:element>
</xsd:schema>`
		require.Empty(t, compileFatalErrorsVersion(t, xsd.Version10, schema))
		require.Empty(t, compileFatalErrorsVersion(t, xsd.Version11, schema))
	})

	// particlesL003: derived element c1{1,2} maps onto base choice(c1{1,2},c2{1,2}){1,1}.
	// Wrapper {1,1} restricts base {1,1}; inner c1{1,2} restricts base branch c1{1,2}. VALID.
	t.Run("L003 element c1{1,2} restricts base choice(c1{1,2},c2{1,2}){1,1}", func(t *testing.T) {
		t.Parallel()
		schema := `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema" targetNamespace="http://xsdtesting" xmlns:x="http://xsdtesting">
  <xsd:complexType name="B">
    <xsd:sequence>
      <xsd:choice>
        <xsd:element name="c1" maxOccurs="2"/>
        <xsd:element name="c2" maxOccurs="2"/>
      </xsd:choice>
      <xsd:choice>
        <xsd:element name="d1" maxOccurs="2"/>
        <xsd:element name="d2" maxOccurs="2"/>
      </xsd:choice>
    </xsd:sequence>
  </xsd:complexType>
  <xsd:complexType name="R">
    <xsd:complexContent>
      <xsd:restriction base="x:B">
        <xsd:sequence>
          <xsd:element name="c1" maxOccurs="2"/>
          <xsd:element name="d1" maxOccurs="2"/>
        </xsd:sequence>
      </xsd:restriction>
    </xsd:complexContent>
  </xsd:complexType>
  <xsd:element name="doc" type="x:R"/>
</xsd:schema>`
		require.Empty(t, compileFatalErrorsVersion(t, xsd.Version10, schema))
		require.Empty(t, compileFatalErrorsVersion(t, xsd.Version11, schema))
	})

	// particlesM003: derived element c1{3,30} maps onto base branch sequence(c1{2,100},c2?).
	// Wrapper {1,1} restricts the base sequence {1,1}; inner c1{3,30} restricts base
	// member c1{2,100}; base c2? skipped by the emptiable-member skip. VALID.
	t.Run("M003 element c1{3,30} restricts base branch seq(c1{2,100},c2?)", func(t *testing.T) {
		t.Parallel()
		schema := `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema" targetNamespace="http://xsdtesting" xmlns:x="http://xsdtesting">
  <xsd:complexType name="B">
    <xsd:choice>
      <xsd:sequence minOccurs="1" maxOccurs="1">
        <xsd:element name="c1" minOccurs="2" maxOccurs="100"/>
        <xsd:element name="c2" minOccurs="0" maxOccurs="1"/>
      </xsd:sequence>
      <xsd:sequence minOccurs="1" maxOccurs="1">
        <xsd:element name="d1" minOccurs="1" maxOccurs="1"/>
        <xsd:element name="d2" minOccurs="1" maxOccurs="1"/>
      </xsd:sequence>
    </xsd:choice>
  </xsd:complexType>
  <xsd:complexType name="R">
    <xsd:complexContent>
      <xsd:restriction base="x:B">
        <xsd:choice>
          <xsd:element name="c1" minOccurs="3" maxOccurs="30"/>
        </xsd:choice>
      </xsd:restriction>
    </xsd:complexContent>
  </xsd:complexType>
  <xsd:element name="doc" type="x:R"/>
</xsd:schema>`
		require.Empty(t, compileFatalErrorsVersion(t, xsd.Version10, schema))
		require.Empty(t, compileFatalErrorsVersion(t, xsd.Version11, schema))
	})

	// particlesZ001 (reject guard): derived element{0,unbounded} maps onto base
	// choice(element{1,1}, any){0,unbounded}. The wrapper {1,1} restricts the base
	// {0,unbounded}, but the INNER branch match fails: element{0,unbounded} restricts
	// neither element{1,1} (min 0 < 1) nor any{1,1} (min 0 < 1). Must stay REJECTED.
	t.Run("Z001 element{0,unbounded} does not restrict choice(element,any) branch", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="http://xsdtesting" xmlns:a="http://xsdtesting">
  <xs:complexType name="Base">
    <xs:sequence>
      <xs:element name="annotation" minOccurs="0"/>
      <xs:choice minOccurs="0" maxOccurs="unbounded">
        <xs:element name="element" minOccurs="1" maxOccurs="1"/>
        <xs:element name="any"/>
      </xs:choice>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="Derived">
    <xs:complexContent>
      <xs:restriction base="a:Base">
        <xs:sequence>
          <xs:element name="annotation" minOccurs="0"/>
          <xs:element name="element" minOccurs="0" maxOccurs="unbounded"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="doc" type="a:Derived"/>
</xs:schema>`
		require.Contains(t, compileFatalErrorsVersion(t, xsd.Version10, schema), notValidRestriction)
	})

	// particlesHa081 (reject guard): derived seq(e,e,any) restricts base seq(any{2,2}).
	// This routes through the pointless-wildcard path (groupRestrictsWildcard), never
	// elementRestrictsGroup: the derived group emits 3 elements > the wildcard's max 2.
	// Must stay REJECTED (occurrence overflow).
	t.Run("Ha081 seq(e,e,any) overflows base any{2,2}", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base">
    <xs:sequence>
      <xs:any namespace="##any" minOccurs="2" maxOccurs="2"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="derived">
    <xs:complexContent>
      <xs:restriction base="base">
        <xs:sequence>
          <xs:element name="e" type="xs:string"/>
          <xs:element name="e" type="xs:string"/>
          <xs:any namespace="##targetNamespace"/>
        </xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`
		require.Contains(t, compileFatalErrorsVersion(t, xsd.Version10, schema), notValidRestriction)
	})
}
