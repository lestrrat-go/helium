package xsd_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSimpleTypeGrammar exercises the schema-representation content model of
// xs:simpleType and its restriction/list/union derivation bodies (XSD Structures
// §3.14.2/§3.15.2/§3.16.2), enforced version-independently in the default (XSD
// 1.0) compiler. Each invalid schema was formerly accepted (false-accept); each
// valid schema must still compile cleanly. Mirrors W3C msMeta/SimpleType_w3c
// groups stB*/stC*/stD*/stE* and the stray-facet stF*/stJ*/stK* cases.
func TestSimpleTypeGrammar(t *testing.T) {
	t.Parallel()

	const head = `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">`
	const tail = `</xsd:schema>`

	t.Run("rejects", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name   string
			schema string
		}{
			{"annotation-only", `<xsd:simpleType name="t"><xsd:annotation/></xsd:simpleType>`},
			{"two-annotations", `<xsd:simpleType name="t"><xsd:annotation/><xsd:annotation/></xsd:simpleType>`},
			{"two-restrictions", `<xsd:simpleType name="t"><xsd:restriction base="xsd:string"/><xsd:restriction base="xsd:string"/></xsd:simpleType>`},
			{"restriction-then-annotation", `<xsd:simpleType name="t"><xsd:restriction base="xsd:string"/><xsd:annotation/></xsd:simpleType>`},
			{"restriction-then-stray", `<xsd:simpleType name="t"><xsd:restriction base="xsd:string"/><xsd:attribute name="a"/></xsd:simpleType>`},
			{"two-lists", `<xsd:simpleType name="t"><xsd:list itemType="xsd:string"/><xsd:list itemType="xsd:string"/></xsd:simpleType>`},
			{"list-then-restriction", `<xsd:simpleType name="t"><xsd:list itemType="xsd:string"/><xsd:restriction base="xsd:string"/></xsd:simpleType>`},
			{"annotation-after-derivation-then-derivation", `<xsd:simpleType name="t"><xsd:annotation/><xsd:annotation/><xsd:restriction base="xsd:string"/></xsd:simpleType>`},
			{"restriction-two-simpleTypes", `<xsd:simpleType name="t"><xsd:restriction><xsd:simpleType><xsd:restriction base="xsd:string"/></xsd:simpleType><xsd:simpleType><xsd:restriction base="xsd:integer"/></xsd:simpleType></xsd:restriction></xsd:simpleType>`},
			{"restriction-with-attribute", `<xsd:simpleType name="t"><xsd:restriction base="xsd:integer"><xsd:maxExclusive value="5"/><xsd:attribute name="a"/></xsd:restriction></xsd:simpleType>`},
			{"restriction-base-and-simpleType", `<xsd:simpleType name="t"><xsd:restriction base="xsd:string"><xsd:simpleType><xsd:restriction base="xsd:string"/></xsd:simpleType></xsd:restriction></xsd:simpleType>`},
			{"restriction-stray-facet-lowercase", `<xsd:simpleType name="b"><xsd:list itemType="xsd:string"/></xsd:simpleType><xsd:simpleType name="t"><xsd:restriction base="b"><xsd:whitespace value="preserve"/></xsd:restriction></xsd:simpleType>`},
			{"restriction-stray-duration", `<xsd:simpleType name="b"><xsd:list itemType="xsd:integer"/></xsd:simpleType><xsd:simpleType name="t"><xsd:restriction base="b"><xsd:duration value="P1D"/></xsd:restriction></xsd:simpleType>`},
			{"list-empty-itemType", `<xsd:simpleType name="t"><xsd:list itemType=""/></xsd:simpleType>`},
			{"list-two-simpleTypes", `<xsd:simpleType name="t"><xsd:list><xsd:simpleType><xsd:restriction base="xsd:integer"/></xsd:simpleType><xsd:simpleType><xsd:restriction base="xsd:string"/></xsd:simpleType></xsd:list></xsd:simpleType>`},
			{"list-simpleType-then-annotation", `<xsd:simpleType name="t"><xsd:list><xsd:simpleType><xsd:restriction base="xsd:integer"/></xsd:simpleType><xsd:annotation/></xsd:list></xsd:simpleType>`},
			{"list-itemType-and-simpleType", `<xsd:simpleType name="t"><xsd:list itemType="xsd:integer"><xsd:simpleType><xsd:restriction base="xsd:integer"/></xsd:simpleType></xsd:list></xsd:simpleType>`},
			{"union-only-annotation", `<xsd:simpleType name="t"><xsd:union><xsd:annotation/></xsd:union></xsd:simpleType>`},
			{"union-two-annotations", `<xsd:simpleType name="t"><xsd:union><xsd:annotation/><xsd:annotation/><xsd:simpleType><xsd:restriction base="xsd:integer"/></xsd:simpleType></xsd:union></xsd:simpleType>`},
			{"union-simpleType-then-annotation", `<xsd:simpleType name="t"><xsd:union><xsd:simpleType><xsd:restriction base="xsd:integer"/></xsd:simpleType><xsd:annotation/></xsd:union></xsd:simpleType>`},
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
			{"restriction-base", `<xsd:simpleType name="t"><xsd:restriction base="xsd:string"><xsd:maxLength value="5"/></xsd:restriction></xsd:simpleType>`},
			{"annotation-then-restriction", `<xsd:simpleType name="t"><xsd:annotation/><xsd:restriction base="xsd:string"/></xsd:simpleType>`},
			{"restriction-inline-simpleType-then-facet", `<xsd:simpleType name="t"><xsd:restriction><xsd:simpleType><xsd:restriction base="xsd:string"/></xsd:simpleType><xsd:maxLength value="5"/></xsd:restriction></xsd:simpleType>`},
			{"list-itemType", `<xsd:simpleType name="t"><xsd:list itemType="xsd:string"/></xsd:simpleType>`},
			{"list-inline-simpleType", `<xsd:simpleType name="t"><xsd:list><xsd:simpleType><xsd:restriction base="xsd:integer"/></xsd:simpleType></xsd:list></xsd:simpleType>`},
			{"union-memberTypes", `<xsd:simpleType name="t"><xsd:union memberTypes="xsd:string xsd:integer"/></xsd:simpleType>`},
			{"union-annotation-then-members", `<xsd:simpleType name="t"><xsd:union><xsd:annotation/><xsd:simpleType><xsd:restriction base="xsd:integer"/></xsd:simpleType><xsd:simpleType><xsd:restriction base="xsd:string"/></xsd:simpleType></xsd:union></xsd:simpleType>`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				require.Empty(t, compileSchemaErrors(t, head+tc.schema+tail),
					"expected %s to compile cleanly", tc.name)
			})
		}
	})
}
