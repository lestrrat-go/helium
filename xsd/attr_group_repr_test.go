package xsd_test

import (
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestAttrGroupRepresentation covers the XML representation of an
// <xs:attributeGroup> (XSD Structures §3.6.2), a version-INDEPENDENT rule
// enforced by the default (XSD 1.0) compiler:
//
//   - a global attributeGroup @name must be a valid xs:NCName;
//   - a nested attributeGroup (inside a definition, a complexType, or a
//     derivation body) is a REFERENCE: it must not carry @name (the definition
//     form) and its content model is (annotation?), so any non-annotation element
//     child is a schema-representation error;
//   - a definition's content model is
//     (annotation?, ((attribute | attributeGroup)*, anyAttribute?)), so a stray
//     element child (e.g. xs:element) is a schema-representation error.
//
// These mirror the W3C xmlschema msMeta AttributeGroup tests attgB002/B003/B004/
// B006 and attgD012/D037/D038/D039, all of which the schema-for-schemas rejects.
func TestAttrGroupRepresentation(t *testing.T) {
	t.Parallel()

	compile := func(t *testing.T, schemaXML string) (bool, string) {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, cerr := xsd.NewCompiler().Label("test.xsd").ErrorHandler(collector).Compile(t.Context(), doc)
		var msg strings.Builder
		for _, e := range collector.Errors() {
			msg.WriteString(e.Error())
		}
		return cerr != nil || len(collector.Errors()) > 0, msg.String()
	}

	t.Run("rejects", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name   string
			schema string
			want   string
		}{
			{
				name: "attgB006_name_not_ncname",
				schema: `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
					<xsd:attributeGroup name="0"><xsd:attribute name="att" type="xsd:int"/></xsd:attributeGroup></xsd:schema>`,
				want: "is not a valid 'xs:NCName'",
			},
			{
				name: "attgB002_nested_definition_in_group",
				schema: `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
					<xsd:attributeGroup name="G"><xsd:attributeGroup name="abc"><xsd:attribute name="att" type="xsd:int"/></xsd:attributeGroup></xsd:attributeGroup></xsd:schema>`,
				want: "not allowed on an attributeGroup reference",
			},
			{
				name: "attgB003_nested_definition_in_complextype",
				schema: `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
					<xsd:complexType name="G"><xsd:attributeGroup name="abc"><xsd:attribute name="att" type="xsd:int"/></xsd:attributeGroup></xsd:complexType></xsd:schema>`,
				want: "not allowed on an attributeGroup reference",
			},
			{
				name: "attgB004_nested_definition_in_extension",
				schema: `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
					<xsd:complexType name="base"><xsd:sequence><xsd:element name="e1"/></xsd:sequence></xsd:complexType>
					<xsd:complexType name="ext"><xsd:complexContent><xsd:extension base="base">
						<xsd:attributeGroup name="abc"/></xsd:extension></xsd:complexContent></xsd:complexType>
					<xsd:attributeGroup name="abc"><xsd:attribute name="att" type="xsd:int"/></xsd:attributeGroup></xsd:schema>`,
				want: "not allowed on an attributeGroup reference",
			},
			{
				name: "attgD037_ref_carries_attribute",
				schema: `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
					<xsd:attributeGroup name="attG"><xsd:attribute name="att1" type="xsd:int"/></xsd:attributeGroup>
					<xsd:complexType name="attgRef"><xsd:attributeGroup ref="attG"><xsd:attribute name="gg"/></xsd:attributeGroup></xsd:complexType></xsd:schema>`,
				want: "restricted to (annotation?)",
			},
			{
				name: "attgD038_ref_carries_nested_ref",
				schema: `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
					<xsd:attributeGroup name="attG"><xsd:attribute name="att1" type="xsd:int"/></xsd:attributeGroup>
					<xsd:complexType name="attgRef"><xsd:attributeGroup ref="attG"><xsd:attributeGroup ref="attG"/></xsd:attributeGroup></xsd:complexType></xsd:schema>`,
				want: "restricted to (annotation?)",
			},
			{
				name: "attgD039_ref_carries_anyattribute",
				schema: `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
					<xsd:attributeGroup name="attG"><xsd:attribute name="att1" type="xsd:int"/></xsd:attributeGroup>
					<xsd:complexType name="attgRef"><xsd:attributeGroup ref="attG"><xsd:anyAttribute namespace="##any"/></xsd:attributeGroup></xsd:complexType></xsd:schema>`,
				want: "restricted to (annotation?)",
			},
			{
				name: "attgD012_stray_element_child",
				schema: `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
					<xsd:attributeGroup name="attG"><xsd:attribute name="att1" type="xsd:int"/><xsd:element name="e1"/></xsd:attributeGroup></xsd:schema>`,
				want: "is not allowed in an attributeGroup",
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				rejected, msg := compile(t, tc.schema)
				require.True(t, rejected, "expected schema to be rejected")
				require.Contains(t, msg, tc.want)
			})
		}
	})

	t.Run("accepts", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name   string
			schema string
		}{
			{
				name: "valid_definition_and_ref",
				schema: `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
					<xsd:attributeGroup name="attG"><xsd:attribute name="att1" type="xsd:int"/></xsd:attributeGroup>
					<xsd:complexType name="attgRef"><xsd:attributeGroup ref="attG"/></xsd:complexType></xsd:schema>`,
			},
			{
				name: "ref_with_annotation_child",
				schema: `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
					<xsd:attributeGroup name="attG"><xsd:attribute name="att1" type="xsd:int"/></xsd:attributeGroup>
					<xsd:complexType name="attgRef"><xsd:attributeGroup ref="attG"><xsd:annotation/></xsd:attributeGroup></xsd:complexType></xsd:schema>`,
			},
			{
				name: "definition_with_leading_annotation",
				schema: `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
					<xsd:attributeGroup name="attG"><xsd:annotation/><xsd:attribute name="att1" type="xsd:int"/></xsd:attributeGroup></xsd:schema>`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				rejected, msg := compile(t, tc.schema)
				require.False(t, rejected, "expected schema to compile: %s", msg)
			})
		}
	})
}
