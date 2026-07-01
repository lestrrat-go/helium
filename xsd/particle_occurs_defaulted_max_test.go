package xsd_test

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

// TestParticleOccursDefaultedMax covers the representation constraint that a
// non-element particle's minOccurs must not exceed its EFFECTIVE maxOccurs,
// where an ABSENT maxOccurs defaults to 1. So minOccurs=2 with no maxOccurs is
// rejected exactly like minOccurs=2 maxOccurs=1. Version-independent, enforced
// in the default XSD 1.0 compiler. Mirrors W3C msMeta/Particles_w3c.xml
// particlesEb015 (group ref) and particlesOa004/Oa008 (xs:any wildcard).
func TestParticleOccursDefaultedMax(t *testing.T) {
	const main = "test.xsd"
	compile := func(t *testing.T, schema string) string {
		t.Helper()
		fsys := fstest.MapFS{main: &fstest.MapFile{Data: []byte(schema)}}
		return compileFSErrors(t, fsys, main)
	}

	const wantMsg = "The value must not be greater than the value of 'maxOccurs'."

	t.Run("group ref minOccurs exceeds defaulted maxOccurs is rejected", func(t *testing.T) {
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:group name="g">
    <xs:sequence><xs:element name="a"/></xs:sequence>
  </xs:group>
  <xs:element name="doc">
    <xs:complexType>
      <xs:group ref="g" minOccurs="2"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		require.Contains(t, compile(t, schema), wantMsg)
	})

	t.Run("xs:any minOccurs exceeds defaulted maxOccurs is rejected", func(t *testing.T) {
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="t">
    <xs:choice>
      <xs:any namespace="##any" minOccurs="2"/>
    </xs:choice>
  </xs:complexType>
</xs:schema>`
		require.Contains(t, compile(t, schema), wantMsg)
	})

	t.Run("xs:choice minOccurs exceeds defaulted maxOccurs is rejected", func(t *testing.T) {
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="t">
    <xs:sequence>
      <xs:choice minOccurs="2">
        <xs:element name="a"/>
        <xs:element name="b"/>
      </xs:choice>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`
		require.Contains(t, compile(t, schema), wantMsg)
	})

	t.Run("valid defaulted-max occurrences still compile", func(t *testing.T) {
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:group name="g">
    <xs:sequence><xs:element name="a"/></xs:sequence>
  </xs:group>
  <xs:complexType name="t">
    <xs:sequence>
      <xs:group ref="g" minOccurs="0"/>
      <xs:choice minOccurs="1">
        <xs:element name="b"/>
      </xs:choice>
      <xs:sequence minOccurs="5" maxOccurs="unbounded">
        <xs:element name="c"/>
      </xs:sequence>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="w">
    <xs:choice>
      <xs:any namespace="##any" minOccurs="2" maxOccurs="4"/>
    </xs:choice>
  </xs:complexType>
</xs:schema>`
		require.Empty(t, compile(t, schema))
	})
}
