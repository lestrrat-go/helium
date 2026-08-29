package xsd_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFacetComponentLabels(t *testing.T) {
	t.Parallel()

	const notationComplex = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="T">
    <xs:simpleContent>
      <xs:restriction base="xs:NOTATION"/>
    </xs:simpleContent>
  </xs:complexType>
</xs:schema>`
	const notationSimple = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="S">
    <xs:restriction base="xs:NOTATION"/>
  </xs:simpleType>
</xs:schema>`

	for _, version := range []struct {
		name string
		v11  bool
	}{
		{testLabelXSD10, false},
		{testLabelXSD11, true},
	} {
		t.Run("notation complex/"+version.name, func(t *testing.T) {
			t.Parallel()
			errs := compileSchemaErrorsVersion(t, notationComplex, version.v11)
			require.Contains(t, errs,
				"complex type 'T': It is an error if the base type is the built-in 'NOTATION' and there is no 'enumeration' facet.")
		})
		t.Run("notation simple/"+version.name, func(t *testing.T) {
			t.Parallel()
			errs := compileSchemaErrorsVersion(t, notationSimple, version.v11)
			require.Contains(t, errs,
				"S: It is an error if the base type is the built-in 'NOTATION' and there is no 'enumeration' facet.")
		})
	}

	t.Run("anyAtomicType complex in xsd11", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="T">
    <xs:simpleContent>
      <xs:restriction base="xs:anyAtomicType"/>
    </xs:simpleContent>
  </xs:complexType>
</xs:schema>`
		errs := compileSchemaErrorsVersion(t, schema, true)
		require.Contains(t, errs, "complex type 'T': The base type must not be the built-in 'anyAtomicType'.")
	})

	t.Run("anyAtomicType simple in xsd11", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="S">
    <xs:restriction base="xs:anyAtomicType"/>
  </xs:simpleType>
</xs:schema>`
		errs := compileSchemaErrorsVersion(t, schema, true)
		require.Contains(t, errs, "S: The base type must not be the built-in 'anyAtomicType'.")
	})

	const anySimpleComplex = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="T">
    <xs:simpleContent>
      <xs:restriction base="xs:anySimpleType"/>
    </xs:simpleContent>
  </xs:complexType>
</xs:schema>`
	const anySimpleLocal = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:simpleContent>
        <xs:restriction base="xs:anySimpleType"/>
      </xs:simpleContent>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	const anySimpleSimple = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="S">
    <xs:restriction base="xs:anySimpleType"/>
  </xs:simpleType>
</xs:schema>`

	for _, version := range []struct {
		name string
		v11  bool
	}{
		{testLabelXSD10, false},
		{testLabelXSD11, true},
	} {
		t.Run("anySimpleType complex/"+version.name, func(t *testing.T) {
			t.Parallel()
			errs := compileSchemaErrorsVersion(t, anySimpleComplex, version.v11)
			require.Contains(t, errs, "complex type 'T': The base type must not be the built-in 'anySimpleType'.")
		})
		t.Run("anySimpleType local/"+version.name, func(t *testing.T) {
			t.Parallel()
			errs := compileSchemaErrorsVersion(t, anySimpleLocal, version.v11)
			require.Contains(t, errs, "local complex type: The base type must not be the built-in 'anySimpleType'.")
		})
		t.Run("anySimpleType simple/"+version.name, func(t *testing.T) {
			t.Parallel()
			errs := compileSchemaErrorsVersion(t, anySimpleSimple, version.v11)
			require.Contains(t, errs, "S: The base type must not be the built-in 'anySimpleType'.")
		})
	}

	t.Run("synthetic local facet type", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="S"><xs:restriction base="xs:decimal"/></xs:simpleType>
  <xs:complexType name="Base">
    <xs:simpleContent><xs:extension base="S"/></xs:simpleContent>
  </xs:complexType>
  <xs:complexType name="T">
    <xs:simpleContent>
      <xs:restriction base="Base">
        <xs:minInclusive value="2"/>
        <xs:maxInclusive value="1"/>
      </xs:restriction>
    </xs:simpleContent>
  </xs:complexType>
</xs:schema>`
		errs := compileSchemaErrorsVersion(t, schema, true)
		require.Contains(t, errs,
			"local simple type: It is an error for the value of 'minInclusive' to be greater than the value of 'maxInclusive'.")
	})
}

func TestCircularComplexTypeComponentLabels(t *testing.T) {
	t.Parallel()

	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="T">
    <xs:complexContent><xs:extension base="U"/></xs:complexContent>
  </xs:complexType>
  <xs:complexType name="U">
    <xs:complexContent><xs:extension base="T"/></xs:complexContent>
  </xs:complexType>
</xs:schema>`
	for _, version := range []struct {
		name string
		v11  bool
	}{
		{testLabelXSD10, false},
		{testLabelXSD11, true},
	} {
		t.Run(version.name, func(t *testing.T) {
			t.Parallel()
			errs := compileSchemaErrorsVersion(t, schema, version.v11)
			for _, name := range []string{"T", "U"} {
				require.Contains(t, errs,
					"element complexType: Schemas parser error : complex type '"+name+"': "+
						"Circular definition of the simple type")
			}
		})
	}
}
