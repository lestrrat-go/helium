package xsd_test

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

// src-redefine.5 (XSD Part 1 §4.2.3): a <simpleType> or <complexType> child of
// <xs:redefine> must have a <restriction> (or, for a complexType, <extension>)
// whose 'base' names the redefined type itself. A base naming a different type,
// a same-local base in another namespace, or no derivation at all is invalid.
// The rule is version-independent, so it is enforced in the default XSD 1.0
// compiler. Mirrors W3C msMeta/Schema_w3c.xml schJ2/schK2/schK3/schP3/schQ2.
func TestRedefine_SelfDerivation(t *testing.T) {
	t.Parallel()

	const baseXSD = "base.xsd"
	base := &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="st">
    <xs:restriction base="xs:string"/>
  </xs:simpleType>
  <xs:complexType name="ct">
    <xs:sequence><xs:element name="c" type="xs:string"/></xs:sequence>
  </xs:complexType>
</xs:schema>`)}

	for _, tc := range []struct {
		name    string
		redef   string
		wantErr bool
	}{
		{
			name: "simpleType restriction of itself is valid",
			redef: `<xs:simpleType name="st">
    <xs:restriction base="st"><xs:minLength value="1"/></xs:restriction>
  </xs:simpleType>`,
		},
		{
			name: "complexType extension of itself is valid",
			redef: `<xs:complexType name="ct">
    <xs:complexContent><xs:extension base="ct">
      <xs:sequence><xs:element name="d" type="xs:string"/></xs:sequence>
    </xs:extension></xs:complexContent>
  </xs:complexType>`,
		},
		{
			name: "simpleType restriction of a different base is invalid",
			redef: `<xs:simpleType name="st">
    <xs:restriction base="xs:string"><xs:minLength value="1"/></xs:restriction>
  </xs:simpleType>`,
			wantErr: true,
		},
		{
			name: "complexType restriction of a different base is invalid",
			redef: `<xs:complexType name="ct">
    <xs:complexContent><xs:restriction base="ct2">
      <xs:sequence><xs:element name="c" type="xs:string"/></xs:sequence>
    </xs:restriction></xs:complexContent>
  </xs:complexType>`,
			wantErr: true,
		},
		{
			name: "complexType with no derivation is invalid",
			redef: `<xs:complexType name="ct">
    <xs:sequence><xs:element name="c" type="xs:string"/></xs:sequence>
  </xs:complexType>`,
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			const mainXSD = "main.xsd"
			fsys := fstest.MapFS{
				baseXSD: base,
				mainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:redefine schemaLocation="base.xsd">
    ` + tc.redef + `
  </xs:redefine>
  <xs:complexType name="ct2">
    <xs:sequence><xs:element name="x" type="xs:string"/></xs:sequence>
  </xs:complexType>
</xs:schema>`)},
			}
			errStr, err := compileRedefineFS(t, fsys, mainXSD)
			if tc.wantErr {
				require.Error(t, err, "expected src-redefine.5 rejection")
				require.Contains(t, errStr, "src-redefine.5")
				return
			}
			require.NoError(t, err, "valid self-derivation must compile; got: %q", errStr)
		})
	}
}
