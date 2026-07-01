package xsd_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestComplexContentBodyGrammar exercises the XSD 1.1 derivation-body content model
// (annotation?, openContent?, (group|all|choice|sequence)?,
// ((attribute|attributeGroup)*, anyAttribute?), assert*) of an <xs:extension> under
// <xs:complexContent>. An extension is used so adding content/attributes is itself
// valid and the ONLY possible error is the body-grammar one under test.
func TestComplexContentBodyGrammar(t *testing.T) {
	t.Parallel()

	ext := func(inner string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="B"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>
  <xs:complexType name="R"><xs:complexContent><xs:extension base="B">
` + inner + `
  </xs:extension></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
	}

	t.Run("valid annotation, model group, attribute, anyAttribute, assert compiles", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, ext(`    <xs:annotation><xs:documentation>ok</xs:documentation></xs:annotation>
    <xs:sequence><xs:element name="b" type="xs:string"/></xs:sequence>
    <xs:attribute name="x" type="xs:string"/>
    <xs:anyAttribute processContents="skip"/>
    <xs:assert test="true()"/>`))
		require.NoError(t, cerr)
	})

	t.Run("two annotations is an error", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, ext(`    <xs:annotation><xs:documentation>one</xs:documentation></xs:annotation>
    <xs:annotation><xs:documentation>two</xs:documentation></xs:annotation>
    <xs:sequence><xs:element name="b" type="xs:string"/></xs:sequence>`))
		require.Error(t, cerr)
	})

	t.Run("annotation after a child is an error", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, ext(`    <xs:sequence><xs:element name="b" type="xs:string"/></xs:sequence>
    <xs:annotation><xs:documentation>too late</xs:documentation></xs:annotation>`))
		require.Error(t, cerr)
	})

	t.Run("model group after assert is an error", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, ext(`    <xs:assert test="true()"/>
    <xs:sequence><xs:element name="b" type="xs:string"/></xs:sequence>`))
		require.Error(t, cerr)
	})

	t.Run("attribute after assert is an error", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, ext(`    <xs:sequence><xs:element name="b" type="xs:string"/></xs:sequence>
    <xs:assert test="true()"/>
    <xs:attribute name="x" type="xs:string"/>`))
		require.Error(t, cerr)
	})

	t.Run("anyAttribute after assert is an error", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, ext(`    <xs:sequence><xs:element name="b" type="xs:string"/></xs:sequence>
    <xs:assert test="true()"/>
    <xs:anyAttribute processContents="skip"/>`))
		require.Error(t, cerr)
	})

	t.Run("stray simpleType child is an error", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, ext(`    <xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType>`))
		require.Error(t, cerr, "simpleType is not a valid child of a complexContent extension")
	})

	t.Run("attribute before model group is an error", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, ext(`    <xs:attribute name="x" type="xs:string"/>
    <xs:sequence><xs:element name="b" type="xs:string"/></xs:sequence>`))
		require.Error(t, cerr)
	})

	t.Run("XSD 1.0 rejects xs:assert as a stray body child", func(t *testing.T) {
		t.Parallel()
		// The complexContent derivation-body grammar is version-independent. xs:assert
		// is not a 1.0 construct, so it is a stray child of the restriction/extension
		// body and is rejected in 1.0 (as it is in 1.1 when out of order).
		schema := ext(`    <xs:sequence><xs:element name="b" type="xs:string"/></xs:sequence>
    <xs:assert test="true()"/>
    <xs:attribute name="x" type="xs:string"/>`)
		_, v10err := compileV10(t, schema)
		require.Error(t, v10err, "1.0 rejects the stray xs:assert in the body")
	})
}

// TestSimpleContentBodyGrammar exercises the XSD 1.1 simpleContent derivation-body
// content models (restriction: annotation?, simpleType?, facet*, attributes,
// anyAttribute?, assert*; extension: annotation?, attributes, anyAttribute?,
// assert*).
func TestSimpleContentBodyGrammar(t *testing.T) {
	t.Parallel()

	// restr restricts SCB, a complex type with simpleContent over xs:string, so the
	// restriction itself is well-formed and only the body grammar is under test.
	restr := func(inner string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="SCB"><xs:simpleContent><xs:extension base="xs:string"/></xs:simpleContent></xs:complexType>
  <xs:complexType name="R"><xs:simpleContent><xs:restriction base="SCB">
` + inner + `
  </xs:restriction></xs:simpleContent></xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
	}
	ext := func(inner string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="R"><xs:simpleContent><xs:extension base="xs:string">
` + inner + `
  </xs:extension></xs:simpleContent></xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
	}

	t.Run("valid restriction: simpleType, facet, assert compiles", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, restr(`    <xs:annotation><xs:documentation>ok</xs:documentation></xs:annotation>
    <xs:simpleType><xs:restriction base="xs:string"><xs:maxLength value="20"/></xs:restriction></xs:simpleType>
    <xs:assert test="true()"/>`))
		require.NoError(t, cerr)
	})

	t.Run("valid restriction: direct facet then assert compiles", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, restr(`    <xs:maxLength value="10"/>
    <xs:assert test="true()"/>`))
		require.NoError(t, cerr)
	})

	t.Run("valid extension: attribute, anyAttribute, assert compiles", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, ext(`    <xs:attribute name="x" type="xs:string"/>
    <xs:anyAttribute processContents="skip"/>
    <xs:assert test="true()"/>`))
		require.NoError(t, cerr)
	})

	t.Run("facet in extension is an error", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, ext(`    <xs:maxLength value="10"/>`))
		require.Error(t, cerr, "facets are not allowed in a simpleContent extension")
	})

	t.Run("simpleType in extension is an error", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, ext(`    <xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType>`))
		require.Error(t, cerr)
	})

	t.Run("simpleType after facet is an error", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, restr(`    <xs:maxLength value="10"/>
    <xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType>`))
		require.Error(t, cerr)
	})

	t.Run("two simpleType children is an error", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, restr(`    <xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType>
    <xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType>`))
		require.Error(t, cerr)
	})

	t.Run("facet after attribute is an error", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, restr(`    <xs:attribute name="x" type="xs:string"/>
    <xs:maxLength value="10"/>`))
		require.Error(t, cerr)
	})

	t.Run("attribute after assert is an error", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, restr(`    <xs:assert test="true()"/>
    <xs:attribute name="x" type="xs:string"/>`))
		require.Error(t, cerr)
	})

	t.Run("anyAttribute before attribute is an error", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, restr(`    <xs:anyAttribute processContents="skip"/>
    <xs:attribute name="x" type="xs:string"/>`))
		require.Error(t, cerr)
	})

	t.Run("two annotations is an error", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, ext(`    <xs:annotation><xs:documentation>one</xs:documentation></xs:annotation>
    <xs:annotation><xs:documentation>two</xs:documentation></xs:annotation>`))
		require.Error(t, cerr)
	})

	t.Run("stray child is an error", func(t *testing.T) {
		t.Parallel()
		_, _, cerr := compileV11(t, ext(`    <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>`))
		require.Error(t, cerr)
	})

	t.Run("XSD 1.0 also rejects a mis-ordered simpleContent extension body", func(t *testing.T) {
		t.Parallel()
		// The simpleContent extension body grammar is version-independent: a facet is
		// not allowed in an extension, so 1.0 rejects it just like 1.1.
		schema := ext(`    <xs:attribute name="x" type="xs:string"/>
    <xs:maxLength value="10"/>`)
		_, v10err := compileV10(t, schema)
		require.Error(t, v10err, "1.0 rejects the facet in a simpleContent extension")
	})
}
