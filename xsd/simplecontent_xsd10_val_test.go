package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// compileXSD10 compiles a schema string under the default (XSD 1.0) compiler.
func compileXSD10(t *testing.T, schemaXSD string) (*xsd.Schema, error) {
	t.Helper()
	schemaDOM, err := helium.NewParser().Parse(t.Context(), []byte(schemaXSD))
	require.NoError(t, err, "parse schema")
	return xsd.NewCompiler().Compile(t.Context(), schemaDOM)
}

// validateXSD10 validates an instance against a compiled schema.
func validateXSD10(t *testing.T, schema *xsd.Schema, docXML string) error {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(docXML))
	require.NoError(t, err, "parse instance")
	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	return xsd.NewValidator(schema).ErrorHandler(collector).Validate(t.Context(), doc)
}

// TestXSD10SimpleContentAnySimpleTypeFixedWhitespace mirrors W3C test93160: an
// element typed xs:anySimpleType with a fixed value. xs:anySimpleType has
// whiteSpace="preserve", so an instance whose content carries surrounding
// whitespace does NOT match the fixed value and must be rejected. helium had been
// collapsing the whitespace (treating anySimpleType as collapse) and accepting it.
func TestXSD10SimpleContentAnySimpleTypeFixedWhitespace(t *testing.T) {
	const schemaXSD = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="fooTest" type="xs:anySimpleType" fixed="test information"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	schema, err := compileXSD10(t, schemaXSD)
	require.NoError(t, err, "schema should compile")

	// Surrounding whitespace: preserve semantics keep it, so it must not match.
	const invalidDoc = "<?xml version=\"1.0\"?><root><fooTest>\n\t\ttest information\n\t</fooTest></root>"
	require.Error(t, validateXSD10(t, schema, invalidDoc), "whitespace-padded value must NOT match fixed on xs:anySimpleType")

	// Exact match still validates.
	const validDoc = `<?xml version="1.0"?><root><fooTest>test information</fooTest></root>`
	require.NoError(t, validateXSD10(t, schema, validDoc), "exact fixed value must validate")
}

// TestXSD10SimpleContentExtensionEnforcesBaseFacets mirrors W3C xsd001: a
// complexType with simpleContent that EXTENDS a named faceted simple type must
// enforce the base type's facets (minLength/maxLength) at instance validation in
// XSD 1.0. helium had skipped the base facets for a simpleContent extension.
func TestXSD10SimpleContentExtensionEnforcesBaseFacets(t *testing.T) {
	const schemaXSD = `<?xml version="1.0" encoding="UTF-8"?>
<xsd:schema targetNamespace="http://foo.com" xmlns="http://foo.com" xmlns:xsd="http://www.w3.org/2001/XMLSchema" elementFormDefault="unqualified">
  <xsd:element name="root">
    <xsd:complexType>
      <xsd:sequence>
        <xsd:element name="child" minOccurs="3" maxOccurs="7">
          <xsd:complexType>
            <xsd:simpleContent>
              <xsd:extension base="mytype"/>
            </xsd:simpleContent>
          </xsd:complexType>
        </xsd:element>
      </xsd:sequence>
    </xsd:complexType>
  </xsd:element>
  <xsd:simpleType name="mytype">
    <xsd:restriction base="xsd:string">
      <xsd:minLength value="3"/>
      <xsd:maxLength value="10"/>
    </xsd:restriction>
  </xsd:simpleType>
</xsd:schema>`
	schema, err := compileXSD10(t, schemaXSD)
	require.NoError(t, err, "schema should compile")

	// n04: empty <child/> (length 0 < minLength 3) -> invalid.
	const n04 = `<?xml version="1.0"?><foo:root xmlns:foo="http://foo.com"><child/><child>atleast3</child><child>10atmost  </child></foo:root>`
	require.Error(t, validateXSD10(t, schema, n04), "empty child underruns minLength=3")

	// n05: first child "1234567890-" is 11 chars (> maxLength 10) -> invalid.
	const n05 = `<?xml version="1.0"?><foo:root xmlns:foo="http://foo.com"><child>1234567890-</child><child>atleast3</child><child>10atmost  </child></foo:root>`
	require.Error(t, validateXSD10(t, schema, n05), "over-long child exceeds maxLength=10")

	// n06: first child "--" is 2 chars (< minLength 3) -> invalid.
	const n06 = `<?xml version="1.0"?><foo:root xmlns:foo="http://foo.com"><child>--</child><child>atleast3</child><child>atleast3</child></foo:root>`
	require.Error(t, validateXSD10(t, schema, n06), "short child underruns minLength=3")

	// v00: three whitespace/text children all within [3,10] -> valid.
	const v00 = `<?xml version="1.0"?><foo:root xmlns:foo="http://foo.com"><child>   </child><child>atleast3</child><child>10atmost  </child></foo:root>`
	require.NoError(t, validateXSD10(t, schema, v00), "in-range children must validate")
}

// TestXSD10SimpleContentRestrictAnySimpleTypeFacetRejected mirrors W3C stZ010: a
// simpleContent restriction whose effective base content type is xs:anySimpleType
// may not carry a length facet (minLength is not applicable to xs:anySimpleType).
// The schema must fail to compile.
func TestXSD10SimpleContentRestrictAnySimpleTypeFacetRejected(t *testing.T) {
	const schemaXSD = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="t1">
    <xs:simpleContent>
      <xs:extension base="xs:anySimpleType"/>
    </xs:simpleContent>
  </xs:complexType>
  <xs:complexType name="t2">
    <xs:simpleContent>
      <xs:restriction base="t1">
        <xs:minLength value="1"/>
      </xs:restriction>
    </xs:simpleContent>
  </xs:complexType>
</xs:schema>`
	_, err := compileXSD10(t, schemaXSD)
	require.Error(t, err, "minLength on a restriction of xs:anySimpleType content must be a schema error")
}
