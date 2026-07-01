package xsd_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// A GLOBAL xs:simpleType's @name is an xs:NCName (XSD Structures §3.14.2), and a
// LOCAL (anonymous/inline) xs:simpleType must NOT carry a @name at all — {name}
// is absent for a type nested in a restriction/list/union/element/attribute.
// Both are version-independent schema-representation rules, enforced under XSD
// 1.0 (default) and 1.1.
func TestSimpleType_NameRules(t *testing.T) {
	t.Parallel()

	invalid := []struct {
		name   string
		schema string
	}{
		{"global-colon", `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="a:b"><xs:restriction base="xs:string"/></xs:simpleType>
</xs:schema>`},
		{"global-leading-digit", `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="1foo"><xs:restriction base="xs:string"/></xs:simpleType>
</xs:schema>`},
		{"local-in-restriction", `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="parent"><xs:restriction>
    <xs:simpleType name="fooType"><xs:restriction base="xs:string"/></xs:simpleType>
  </xs:restriction></xs:simpleType>
</xs:schema>`},
		{"local-in-list", `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="parent"><xs:list>
    <xs:simpleType name="fooType"><xs:restriction base="xs:string"/></xs:simpleType>
  </xs:list></xs:simpleType>
</xs:schema>`},
		{"local-in-union", `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="parent"><xs:union>
    <xs:simpleType name="fooType"><xs:restriction base="xs:string"/></xs:simpleType>
  </xs:union></xs:simpleType>
</xs:schema>`},
		{"local-in-attribute", `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attribute name="parent">
    <xs:simpleType name="fooType"><xs:restriction base="xs:string"/></xs:simpleType>
  </xs:attribute>
</xs:schema>`},
		{"local-in-element", `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="parent">
    <xs:simpleType name="fooType"><xs:restriction base="xs:string"/></xs:simpleType>
  </xs:element>
</xs:schema>`},
	}
	for _, tc := range invalid {
		t.Run("invalid/"+tc.name, func(t *testing.T) {
			t.Parallel()
			for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
				schema, _, cerr := compileWith(t, v, tc.schema)
				require.ErrorIs(t, cerr, xsd.ErrCompilationFailed, "version=%v", v)
				require.Nil(t, schema)
			}
		})
	}

	valid := []struct {
		name   string
		schema string
	}{
		{"global-underscore", `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="_foo"><xs:restriction base="xs:string"/></xs:simpleType>
</xs:schema>`},
		{"global-with-anonymous-local", `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="parent"><xs:restriction>
    <xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType>
  </xs:restriction></xs:simpleType>
</xs:schema>`},
		{"anonymous-local-in-element", `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="parent">
    <xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType>
  </xs:element>
</xs:schema>`},
	}
	for _, tc := range valid {
		t.Run("valid/"+tc.name, func(t *testing.T) {
			t.Parallel()
			for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
				_, errs, cerr := compileWith(t, v, tc.schema)
				require.NoError(t, cerr, "version=%v: %s", v, errs)
			}
		})
	}
}
