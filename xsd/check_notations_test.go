package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestNotationStructuralRules exercises the version-independent schema-for-schemas
// structural rules for <xs:notation> (XSD Structures §3.14.2): placement,
// @name validity, the @public/@system requirement, the (annotation?) content
// model, disallowed attributes, and name uniqueness. All run in DEFAULT (1.0)
// mode. A valid notation — including one that is the enumeration base of an
// xs:NOTATION restriction — must still compile.
func TestNotationStructuralRules(t *testing.T) {
	t.Parallel()

	reject := []struct {
		name   string
		schema string
	}{
		{
			name: "missing name",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:notation public="image/jpeg" system="viewer.exe"/>
</xs:schema>`,
		},
		{
			name: "empty name",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:notation name="" public="image/jpeg"/>
</xs:schema>`,
		},
		{
			name: "name with colon is not an NCName",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:notation name="foo:bar" public="image/jpeg"/>
</xs:schema>`,
		},
		{
			name: "name starting with digit is not an NCName",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:notation name="-2.5foo" public="image/jpeg"/>
</xs:schema>`,
		},
		{
			name: "neither public nor system",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:notation name="foo"/>
</xs:schema>`,
		},
		{
			name: "duplicate notation name",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:notation name="jpeg" public="image/jpeg" system="viewer.exe"/>
  <xs:notation name="jpeg" public="image/jpeg" system="viewer.exe"/>
</xs:schema>`,
		},
		{
			name: "non-whitespace text content",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:notation name="jpeg" public="image/jpeg">Some Text</xs:notation>
</xs:schema>`,
		},
		{
			name: "disallowed non-annotation child",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:notation name="jpeg" public="image/jpeg"><xs:sequence/></xs:notation>
</xs:schema>`,
		},
		{
			name: "misplaced inside complexType",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="foo">
    <xs:sequence>
      <xs:notation name="jpeg" public="image/jpeg" system="viewer.exe"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`,
		},
		{
			name: "disallowed XSD-namespaced attribute",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:a="http://www.w3.org/2001/XMLSchema">
  <xs:notation a:b="c" name="jpeg" public="image/jpeg" system="viewer.exe"/>
</xs:schema>`,
		},
		{
			name: "disallowed unqualified attribute",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:notation foo="bar" name="jpeg" public="image/jpeg" system="viewer.exe"/>
</xs:schema>`,
		},
	}

	for _, tc := range reject {
		t.Run("reject/"+tc.name, func(t *testing.T) {
			t.Parallel()
			errs := compileSchemaErrors(t, tc.schema)
			require.NotEmpty(t, errs, "expected a compile error rejecting the invalid notation")
		})
	}

	accept := []struct {
		name   string
		schema string
	}{
		{
			name: "public only",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:notation name="jpeg" public="image/jpeg"/>
</xs:schema>`,
		},
		{
			name: "system only",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:notation name="png" system="viewer.exe"/>
</xs:schema>`,
		},
		{
			name: "empty system present",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:notation name="jpeg" public="image/jpeg" system=""/>
</xs:schema>`,
		},
		{
			name: "foreign-namespaced attribute allowed",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:a="urn:foo">
  <xs:notation a:b="c" name="jpeg" public="image/jpeg"/>
</xs:schema>`,
		},
		{
			name: "annotation child allowed",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:notation name="jpeg" public="image/jpeg">
    <xs:annotation><xs:documentation>a JPEG image</xs:documentation></xs:annotation>
  </xs:notation>
</xs:schema>`,
		},
		{
			name: "notation as enumeration base of xs:NOTATION restriction",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:p="urn:p" targetNamespace="urn:p">
  <xs:notation name="jpeg" public="image/jpeg"/>
  <xs:notation name="png" public="image/png"/>
  <xs:simpleType name="imageNotation">
    <xs:restriction base="xs:NOTATION">
      <xs:enumeration value="p:jpeg"/>
      <xs:enumeration value="p:png"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="n" type="p:imageNotation"/>
</xs:schema>`,
		},
	}

	for _, tc := range accept {
		t.Run("accept/"+tc.name, func(t *testing.T) {
			t.Parallel()
			doc, err := helium.NewParser().Parse(t.Context(), []byte(tc.schema))
			require.NoError(t, err)
			_, err = xsd.NewCompiler().Compile(t.Context(), doc)
			require.NoError(t, err, "valid notation schema must compile")
		})
	}
}
