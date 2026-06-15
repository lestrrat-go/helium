package xsd_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestPatternFacetSameStepOR checks that multiple <xs:pattern> facets in the
// same restriction step are ORed: a value is valid if it matches any of them.
func TestPatternFacetSameStepOR(t *testing.T) {
	t.Parallel()

	schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="codeType"/>
  <xs:simpleType name="codeType">
    <xs:restriction base="xs:string">
      <xs:pattern value="[a-z]+"/>
      <xs:pattern value="[0-9]+"/>
    </xs:restriction>
  </xs:simpleType>
</xs:schema>`

	t.Run("matches first pattern", func(t *testing.T) {
		require.NoError(t, compileAndValidate(t, schemaXML, "<root>abc</root>", nil))
	})

	t.Run("matches second pattern", func(t *testing.T) {
		require.NoError(t, compileAndValidate(t, schemaXML, "<root>123</root>", nil))
	})

	t.Run("matches neither pattern", func(t *testing.T) {
		var out string
		err := compileAndValidate(t, schemaXML, "<root>ab12</root>", &out)
		require.Error(t, err)
		require.Contains(t, out, "[facet 'pattern']")
	})
}

// TestPatternFacetCrossStepAND checks that patterns from different derivation
// steps are ANDed: the value must satisfy every step's pattern facet.
func TestPatternFacetCrossStepAND(t *testing.T) {
	t.Parallel()

	schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="step2"/>
  <xs:simpleType name="step1">
    <xs:restriction base="xs:string">
      <xs:pattern value="[a-z0-9]+"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="step2">
    <xs:restriction base="step1">
      <xs:pattern value=".{4}"/>
    </xs:restriction>
  </xs:simpleType>
</xs:schema>`

	t.Run("satisfies both steps", func(t *testing.T) {
		require.NoError(t, compileAndValidate(t, schemaXML, "<root>ab12</root>", nil))
	})

	t.Run("fails outer-step length", func(t *testing.T) {
		var out string
		err := compileAndValidate(t, schemaXML, "<root>abc</root>", &out)
		require.Error(t, err)
		require.Contains(t, out, "[facet 'pattern']")
	})

	t.Run("fails inner-step charset", func(t *testing.T) {
		var out string
		err := compileAndValidate(t, schemaXML, "<root>AB12</root>", &out)
		require.Error(t, err)
		require.Contains(t, out, "[facet 'pattern']")
	})
}
