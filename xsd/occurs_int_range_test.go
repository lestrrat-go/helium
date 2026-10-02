package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestLengthFacetPastInt32 verifies that a length facet past 2^31-1 keeps its
// meaning on every platform. Parsing it into a 32-bit int fails, and the facet
// would read as 0: maxLength would reject every non-empty value and
// minLength <= maxLength would fail to compile.
func TestLengthFacetPastInt32(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="s">
    <xs:restriction base="xs:string">
      <xs:minLength value="5"/>
      <xs:maxLength value="3000000000"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="root" type="s"/>
</xs:schema>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	schema, err := xsd.NewCompiler().Compile(t.Context(), doc)
	require.NoError(t, err)

	require.NoError(t, validateXML(t, schema, `<root>abcdef</root>`))
	require.Error(t, validateXML(t, schema, `<root>abc</root>`))
}

// TestWildcardOccursSumPastInt32 verifies that summing the maxOccurs of the
// derived wildcards cannot wrap. Two wildcards of 2000000000 each sum to
// 4000000000, more than the base wildcard's 2000000000, so the restriction is
// invalid; a sum wrapped in a 32-bit int reads as negative and would accept it.
func TestWildcardOccursSumPastInt32(t *testing.T) {
	t.Parallel()

	mustCompile11Fail(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base">
    <xs:all>
      <xs:any namespace="urn:a urn:b" processContents="lax" maxOccurs="2000000000"/>
    </xs:all>
  </xs:complexType>
  <xs:complexType name="derived">
    <xs:complexContent>
      <xs:restriction base="base">
        <xs:all>
          <xs:any namespace="urn:a" processContents="lax" maxOccurs="2000000000"/>
          <xs:any namespace="urn:b" processContents="lax" maxOccurs="2000000000"/>
        </xs:all>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`)
}
