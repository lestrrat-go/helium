package xsd_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSimpleTypeResolution exercises the simple-type type-resolution kind rules
// (XSD Structures §3.14.6 / Part 2 §2.4), enforced version-independently in the
// default (XSD 1.0) compiler: a restriction {base type definition}, a list {item
// type definition}, and each union {member type definition} must resolve to a
// SIMPLE type — a complexType or the ur-type xs:anyType is a schema error — and a
// list item type must not itself be a list. Mirrors W3C msMeta/SimpleType_w3c
// groups stC003/stD019/stE018/stI004/stJ003/stJ019/stK003, each a former
// false-accept.
func TestSimpleTypeResolution(t *testing.T) {
	t.Parallel()

	const head = `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">`
	const tail = `</xsd:schema>`
	const simpleContentCT = `<xsd:complexType name="ct"><xsd:simpleContent><xsd:extension base="xsd:integer"/></xsd:simpleContent></xsd:complexType>`

	t.Run("rejects", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name   string
			schema string
		}{
			{"restriction-base-anyType", `<xsd:simpleType name="t"><xsd:restriction base="xsd:anyType"/></xsd:simpleType>`},
			{"restriction-base-complex", simpleContentCT + `<xsd:simpleType name="t"><xsd:restriction base="ct"/></xsd:simpleType>`},
			{"list-item-complex", simpleContentCT + `<xsd:simpleType name="t"><xsd:list itemType="ct"/></xsd:simpleType>`},
			{"union-member-complex", simpleContentCT + `<xsd:simpleType name="t"><xsd:union memberTypes="ct"/></xsd:simpleType>`},
			{"union-member-complex-second", simpleContentCT + `<xsd:simpleType name="t"><xsd:union memberTypes="xsd:string ct"/></xsd:simpleType>`},
			{"list-item-list-named", `<xsd:simpleType name="l"><xsd:list itemType="xsd:string"/></xsd:simpleType><xsd:simpleType name="t"><xsd:list itemType="l"/></xsd:simpleType>`},
			{"list-item-list-inline-derived", `<xsd:simpleType name="l"><xsd:list><xsd:simpleType><xsd:restriction base="xsd:integer"/></xsd:simpleType></xsd:list></xsd:simpleType><xsd:simpleType name="t"><xsd:list itemType="l"/></xsd:simpleType>`},
			{"local-restriction-base-complex", simpleContentCT + `<xsd:element name="e"><xsd:complexType><xsd:sequence><xsd:element name="c"><xsd:simpleType><xsd:restriction base="ct"/></xsd:simpleType></xsd:element></xsd:sequence></xsd:complexType></xsd:element>`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				require.NotEmpty(t, compileSchemaErrors(t, head+tc.schema+tail),
					"expected %s to be rejected", tc.name)
			})
		}
	})

	t.Run("accepts", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name   string
			schema string
		}{
			{"restriction-base-builtin", `<xsd:simpleType name="t"><xsd:restriction base="xsd:string"><xsd:maxLength value="5"/></xsd:restriction></xsd:simpleType>`},
			{"restriction-base-anySimpleType", `<xsd:simpleType name="t"><xsd:restriction base="xsd:anySimpleType"/></xsd:simpleType>`},
			{"restriction-base-user-list", `<xsd:simpleType name="l"><xsd:list itemType="xsd:string"/></xsd:simpleType><xsd:simpleType name="t"><xsd:restriction base="l"><xsd:length value="2"/></xsd:restriction></xsd:simpleType>`},
			{"list-item-builtin", `<xsd:simpleType name="t"><xsd:list itemType="xsd:integer"/></xsd:simpleType>`},
			{"list-item-user-atomic", `<xsd:simpleType name="a"><xsd:restriction base="xsd:integer"/></xsd:simpleType><xsd:simpleType name="t"><xsd:list itemType="a"/></xsd:simpleType>`},
			{"list-item-union", `<xsd:simpleType name="u"><xsd:union memberTypes="xsd:string xsd:integer"/></xsd:simpleType><xsd:simpleType name="t"><xsd:list itemType="u"/></xsd:simpleType>`},
			{"union-members-builtin", `<xsd:simpleType name="t"><xsd:union memberTypes="xsd:string xsd:integer"/></xsd:simpleType>`},
			{"union-member-user-list", `<xsd:simpleType name="l"><xsd:list itemType="xsd:string"/></xsd:simpleType><xsd:simpleType name="t"><xsd:union memberTypes="l xsd:string"/></xsd:simpleType>`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				require.Empty(t, compileSchemaErrors(t, head+tc.schema+tail),
					"expected %s to compile cleanly", tc.name)
			})
		}
	})
}
