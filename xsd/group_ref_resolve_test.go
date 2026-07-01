package xsd_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// A <xs:group ref="..."> in a content model must resolve to a globally declared
// model group (XSD Structures §3.7.2 / src-resolve). A ref naming a component in
// a different symbol space (a complexType or an attributeGroup of the same name)
// or a name declared nowhere is a schema-representation error. This is
// version-independent, so it is enforced under both XSD 1.0 (default) and XSD
// 1.1. A valid reference — including one to a chameleon / no-namespace imported
// group — must still compile.
func TestGroup_RefMustResolveToModelGroup(t *testing.T) {
	t.Parallel()

	invalid := []struct {
		name string
		xml  string
	}{
		{
			// ref names a global attributeGroup (wrong symbol space).
			name: "ref-to-attributeGroup",
			xml: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="complexType"><xs:sequence><xs:element name="r1"/></xs:sequence></xs:complexType>
  <xs:element name="elem"><xs:complexType><xs:complexContent><xs:extension base="complexType">
    <xs:group ref="attG"/></xs:extension></xs:complexContent></xs:complexType></xs:element>
  <xs:attributeGroup name="attG"><xs:attribute name="att1"/></xs:attributeGroup>
</xs:schema>`,
		},
		{
			// ref names a global complexType (wrong symbol space).
			name: "ref-to-complexType",
			xml: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="complexType"><xs:sequence><xs:element name="r1"/></xs:sequence></xs:complexType>
  <xs:element name="elem"><xs:complexType><xs:complexContent><xs:extension base="complexType">
    <xs:group ref="complexType"/></xs:extension></xs:complexContent></xs:complexType></xs:element>
</xs:schema>`,
		},
		{
			// ref names a group that does not exist anywhere.
			name: "dangling-ref",
			xml: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="A"><xs:group ref="nope"/></xs:complexType>
  <xs:element name="doc" type="A"/>
</xs:schema>`,
		},
		{
			// dangling ref nested inside a model group.
			name: "dangling-ref-nested",
			xml: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="A"><xs:sequence><xs:group ref="nope"/></xs:sequence></xs:complexType>
  <xs:element name="doc" type="A"/>
</xs:schema>`,
		},
	}
	for _, tc := range invalid {
		t.Run("invalid/"+tc.name, func(t *testing.T) {
			t.Parallel()
			for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
				schema, errs, cerr := compileWith(t, v, tc.xml)
				require.ErrorIs(t, cerr, xsd.ErrCompilationFailed, "version=%v must reject unresolved group ref", v)
				require.Nil(t, schema)
				require.Contains(t, errs, "does not resolve to a(n) model group definition", "version=%v", v)
			}
		})
	}

	valid := []struct {
		name string
		xml  string
	}{
		{
			// ref resolves to a real model group in the same namespace.
			name: "ref-to-model-group",
			xml: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="A"><xs:group ref="xyz"/></xs:complexType>
  <xs:group name="xyz"><xs:sequence><xs:element name="A"/></xs:sequence></xs:group>
  <xs:element name="doc" type="A"/>
</xs:schema>`,
		},
		{
			// nested ref resolves to a real model group.
			name: "nested-ref-to-model-group",
			xml: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="A"><xs:sequence><xs:group ref="xyz"/></xs:sequence></xs:complexType>
  <xs:group name="xyz"><xs:sequence><xs:element name="A"/></xs:sequence></xs:group>
  <xs:element name="doc" type="A"/>
</xs:schema>`,
		},
	}
	for _, tc := range valid {
		t.Run("valid/"+tc.name, func(t *testing.T) {
			t.Parallel()
			for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
				_, errs, cerr := compileWith(t, v, tc.xml)
				require.NoError(t, cerr, "version=%v must accept a resolvable group ref: %s", v, errs)
			}
		})
	}
}
