package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// compileValidateV10 compiles schema with the default (XSD 1.0) compiler and
// validates instance against it, returning the validation error (or a compile
// error). Both documents are supplied as raw XML.
func compileValidateV10(t *testing.T, schema, instance string) error {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(schema))
	require.NoError(t, err)
	sc, err := xsd.NewCompiler().Compile(t.Context(), doc)
	require.NoError(t, err, "schema must compile")
	idoc, err := helium.NewParser().Parse(t.Context(), []byte(instance))
	require.NoError(t, err)
	return xsd.NewValidator(sc).Validate(t.Context(), idoc)
}

// TestValidateResidualsXSD10 covers three XSD 1.0 instance-validation conformance
// bugs (W3C msMeta elemO001, addB065, addB116).
func TestValidateResidualsXSD10(t *testing.T) {
	// cvc-elt.2: an ABSTRACT element declaration used DIRECTLY in an instance (here
	// inside an xs:all, so the child reaches the 1.0 all-matcher) is invalid — only
	// a concrete substitution-group member may appear. (W3C elemO001)
	t.Run("abstract element used directly in xs:all is invalid", func(t *testing.T) {
		t.Parallel()
		schema := `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <xsd:element name="root">
    <xsd:complexType><xsd:sequence>
      <xsd:element ref="fooTest"/>
    </xsd:sequence></xsd:complexType>
  </xsd:element>
  <xsd:element name="fooTest">
    <xsd:complexType><xsd:all>
      <xsd:element ref="parent"/>
    </xsd:all></xsd:complexType>
  </xsd:element>
  <xsd:element name="parent" type="xsd:string" abstract="true"/>
</xsd:schema>`
		instance := `<root><fooTest><parent>content</parent></fooTest></root>`
		require.ErrorIs(t, compileValidateV10(t, schema, instance), xsd.ErrValidationFailed)
	})

	// A concrete substitution-group member for the abstract head is still admitted.
	t.Run("concrete substitution member for abstract element is valid", func(t *testing.T) {
		t.Parallel()
		schema := `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <xsd:element name="fooTest">
    <xsd:complexType><xsd:all>
      <xsd:element ref="parent"/>
    </xsd:all></xsd:complexType>
  </xsd:element>
  <xsd:element name="parent" type="xsd:string" abstract="true"/>
  <xsd:element name="child" type="xsd:string" substitutionGroup="parent"/>
</xsd:schema>`
		instance := `<fooTest><child>content</child></fooTest>`
		require.NoError(t, compileValidateV10(t, schema, instance))
	})

	// cvc-elt.3.2.2: a nilled element (xsi:nil="true") whose declaration carries a
	// fixed {value constraint} is invalid. (W3C addB065 / test73826)
	t.Run("nilled element with fixed value constraint is invalid", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="r">
    <xs:complexType><xs:sequence>
      <xs:element name="i" type="xs:string" minOccurs="0" nillable="true" fixed="abc"/>
    </xs:sequence></xs:complexType>
  </xs:element>
</xs:schema>`
		instance := `<r xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"><i xsi:nil="true"/></r>`
		require.ErrorIs(t, compileValidateV10(t, schema, instance), xsd.ErrValidationFailed)
	})

	// A nillable element WITHOUT a fixed constraint may still be nilled (the fix is
	// scoped to the fixed-value case — no over-rejection).
	t.Run("nilled element without fixed constraint is valid", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="r">
    <xs:complexType><xs:sequence>
      <xs:element name="i" type="xs:string" minOccurs="0" nillable="true"/>
    </xs:sequence></xs:complexType>
  </xs:element>
</xs:schema>`
		instance := `<r xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"><i xsi:nil="true"/></r>`
		require.NoError(t, compileValidateV10(t, schema, instance))
	})

	// cvc-assess-elt: an element matched by a strict wildcard that has no matching
	// global element declaration BUT carries a resolvable xsi:type is assessed
	// against that type rather than rejected. (W3C addB116 / test75092)
	t.Run("strict wildcard child with xsi:type is valid", func(t *testing.T) {
		t.Parallel()
		schema := `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <xsd:complexType name="foo">
    <xsd:sequence>
      <xsd:element name="a"/>
      <xsd:any namespace="##any" processContents="strict" minOccurs="0" maxOccurs="unbounded"/>
    </xsd:sequence>
  </xsd:complexType>
  <xsd:element name="foo" type="foo"/>
</xsd:schema>`
		instance := `<foo xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <a/>
  <b xsi:type="xsd:string">abc</b>
  <c xsi:type="xsd:int">123</c>
</foo>`
		require.NoError(t, compileValidateV10(t, schema, instance))
	})

	// A strict wildcard child that has NEITHER a global declaration NOR an xsi:type
	// still rejects (no over-acceptance).
	t.Run("strict wildcard child without decl or xsi:type is invalid", func(t *testing.T) {
		t.Parallel()
		schema := `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <xsd:complexType name="foo">
    <xsd:sequence>
      <xsd:element name="a"/>
      <xsd:any namespace="##any" processContents="strict" minOccurs="0" maxOccurs="unbounded"/>
    </xsd:sequence>
  </xsd:complexType>
  <xsd:element name="foo" type="foo"/>
</xsd:schema>`
		instance := `<foo><a/><b>abc</b></foo>`
		require.ErrorIs(t, compileValidateV10(t, schema, instance), xsd.ErrValidationFailed)
	})

	// A strict wildcard child with an xsi:type whose CONTENT is invalid against
	// that type still rejects.
	t.Run("strict wildcard child with xsi:type but invalid content is invalid", func(t *testing.T) {
		t.Parallel()
		schema := `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <xsd:complexType name="foo">
    <xsd:sequence>
      <xsd:element name="a"/>
      <xsd:any namespace="##any" processContents="strict" minOccurs="0" maxOccurs="unbounded"/>
    </xsd:sequence>
  </xsd:complexType>
  <xsd:element name="foo" type="foo"/>
</xsd:schema>`
		instance := `<foo xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <a/>
  <c xsi:type="xsd:int">notint</c>
</foo>`
		require.ErrorIs(t, compileValidateV10(t, schema, instance), xsd.ErrValidationFailed)
	})
}
