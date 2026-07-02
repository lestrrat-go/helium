package xsd_test

import (
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestComplexTypeBlockFinalValue covers the XML representation of the @block and
// @final attributes on an <xs:complexType> (XSD Structures §3.4.2). Both are of
// type (#all | List of (extension | restriction)); neither admits 'substitution'
// (an element-declaration value) nor 'list'/'union' (simpleType values). This is
// a version-independent rule enforced by the default (XSD 1.0) compiler and
// mirrors the W3C xmlschema msMeta ComplexType tests ctA016 (block='substitution')
// and ctA025 (final='substitution'), both of which the schema-for-schemas rejects.
func TestComplexTypeBlockFinalValue(t *testing.T) {
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
				name: "ctA016_block_substitution",
				schema: `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
					<xsd:complexType block="substitution" name="foo"/></xsd:schema>`,
				want: "The value 'substitution' of attribute 'block' is not valid",
			},
			{
				name: "ctA025_final_substitution",
				schema: `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
					<xsd:complexType final="substitution" name="foo"/></xsd:schema>`,
				want: "The value 'substitution' of attribute 'final' is not valid",
			},
			{
				name: "block_list_invalid",
				schema: `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
					<xsd:complexType block="list" name="foo"/></xsd:schema>`,
				want: "The value 'list' of attribute 'block' is not valid",
			},
			{
				name: "final_extension_union_invalid",
				schema: `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
					<xsd:complexType final="extension union" name="foo"/></xsd:schema>`,
				want: "The value 'extension union' of attribute 'final' is not valid",
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				bad, msg := compile(t, tc.schema)
				require.True(t, bad, "expected schema to be rejected")
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
				name: "block_all",
				schema: `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
					<xsd:complexType block="#all" name="foo"/></xsd:schema>`,
			},
			{
				name: "final_extension_restriction",
				schema: `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
					<xsd:complexType final="extension restriction" name="foo"/></xsd:schema>`,
			},
			{
				name: "block_empty",
				schema: `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
					<xsd:complexType block="" name="foo"/></xsd:schema>`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				bad, msg := compile(t, tc.schema)
				require.False(t, bad, "expected schema to compile: %s", msg)
			})
		}
	})
}
