package xsd_test

import (
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestIntrinsicBuiltinFacets covers the version-INDEPENDENT intrinsic facets the
// built-in datatypes carry — xs:NMTOKENS/xs:IDREFS/xs:ENTITIES have minLength=1
// (Part 2 §4.3.2) and xs:positiveInteger has minInclusive=1 (§3.3.25) — and the
// inheritance-aware facet-restriction checks that observe them.
//
// In particular the "length and minLength or maxLength" co-occurrence rule
// (§4.3.1.4, W3C bug 6446) permits length beside a min/maxLength ONLY when that
// min/maxLength is genuinely INHERITED and merely RESTATES the inherited value
// (value-consistent with length); a fresh, tighter bound introduced alongside
// length is a schema error.
func TestIntrinsicBuiltinFacets(t *testing.T) {
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
		}{
			{
				// length=5 with a FRESH minLength=2 (own 2 != inherited intrinsic
				// minLength 1): the min/maxLength is not a mere restatement of the
				// inherited value, so length may not co-occur with it.
				name: "length_with_fresh_minLength_on_NMTOKENS",
				schema: `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
					<xsd:element name="e"><xsd:simpleType>
						<xsd:restriction base="xsd:NMTOKENS">
							<xsd:length value="5"/>
							<xsd:minLength value="2"/>
						</xsd:restriction>
					</xsd:simpleType></xsd:element></xsd:schema>`,
			},
			{
				// base maxLength=10, derived step restates it as length=5 with a
				// DIFFERENT maxLength=8 (own 8 != inherited 10): a fresh tighter bound
				// alongside length, an error.
				name: "length_with_restated_maxLength",
				schema: `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
					<xsd:simpleType name="base">
						<xsd:restriction base="xsd:string"><xsd:maxLength value="10"/></xsd:restriction>
					</xsd:simpleType>
					<xsd:element name="e"><xsd:simpleType>
						<xsd:restriction base="base">
							<xsd:length value="5"/>
							<xsd:maxLength value="8"/>
						</xsd:restriction>
					</xsd:simpleType></xsd:element></xsd:schema>`,
			},
			{
				// minLength=0 underruns the intrinsic minLength=1 of xs:NMTOKENS.
				name: "minLength_below_intrinsic_on_NMTOKENS",
				schema: `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
					<xsd:element name="e"><xsd:simpleType>
						<xsd:restriction base="xsd:NMTOKENS"><xsd:minLength value="0"/></xsd:restriction>
					</xsd:simpleType></xsd:element></xsd:schema>`,
			},
			{
				// maxLength=0 falls below the intrinsic minLength=1 of xs:IDREFS.
				name: "maxLength_below_intrinsic_on_IDREFS",
				schema: `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
					<xsd:element name="e"><xsd:simpleType>
						<xsd:restriction base="xsd:IDREFS"><xsd:maxLength value="0"/></xsd:restriction>
					</xsd:simpleType></xsd:element></xsd:schema>`,
			},
			{
				// maxExclusive=1 is not greater than the intrinsic minInclusive=1 of
				// xs:positiveInteger (§4.3.9.4).
				name: "maxExclusive_not_above_intrinsic_on_positiveInteger",
				schema: `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
					<xsd:element name="e"><xsd:simpleType>
						<xsd:restriction base="xsd:positiveInteger"><xsd:maxExclusive value="1"/></xsd:restriction>
					</xsd:simpleType></xsd:element></xsd:schema>`,
			},
			{
				// The intrinsic minLength=1 of xs:NMTOKENS must still bind through an
				// INTERMEDIATE restriction that carries only an unrelated facet (a
				// pattern): a nearest-ancestor lookup would see only the pattern step and
				// miss the inherited minLength.
				name: "minLength_below_intrinsic_through_intermediate",
				schema: `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
					<xsd:simpleType name="mid">
						<xsd:restriction base="xsd:NMTOKENS"><xsd:pattern value=".*"/></xsd:restriction>
					</xsd:simpleType>
					<xsd:element name="e"><xsd:simpleType>
						<xsd:restriction base="mid"><xsd:minLength value="0"/></xsd:restriction>
					</xsd:simpleType></xsd:element></xsd:schema>`,
			},
			{
				// User minLength=5 hidden behind a pattern-only intermediate; derived
				// minLength=3 loosens it.
				name: "minLength_loosened_through_intermediate",
				schema: `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
					<xsd:simpleType name="base"><xsd:restriction base="xsd:string"><xsd:minLength value="5"/></xsd:restriction></xsd:simpleType>
					<xsd:simpleType name="mid"><xsd:restriction base="base"><xsd:pattern value=".*"/></xsd:restriction></xsd:simpleType>
					<xsd:element name="e"><xsd:simpleType>
						<xsd:restriction base="mid"><xsd:minLength value="3"/></xsd:restriction>
					</xsd:simpleType></xsd:element></xsd:schema>`,
			},
			{
				// User maxLength=5 hidden behind a pattern-only intermediate; derived
				// maxLength=7 widens it.
				name: "maxLength_widened_through_intermediate",
				schema: `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
					<xsd:simpleType name="base"><xsd:restriction base="xsd:string"><xsd:maxLength value="5"/></xsd:restriction></xsd:simpleType>
					<xsd:simpleType name="mid"><xsd:restriction base="base"><xsd:pattern value=".*"/></xsd:restriction></xsd:simpleType>
					<xsd:element name="e"><xsd:simpleType>
						<xsd:restriction base="mid"><xsd:maxLength value="7"/></xsd:restriction>
					</xsd:simpleType></xsd:element></xsd:schema>`,
			},
			{
				// Inherited length=5, derived maxLength=4 < length.
				name: "maxLength_below_inherited_length",
				schema: `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
					<xsd:simpleType name="base"><xsd:restriction base="xsd:string"><xsd:length value="5"/></xsd:restriction></xsd:simpleType>
					<xsd:element name="e"><xsd:simpleType>
						<xsd:restriction base="base"><xsd:maxLength value="4"/></xsd:restriction>
					</xsd:simpleType></xsd:element></xsd:schema>`,
			},
			{
				// Inherited length=5, derived minLength=6 > length.
				name: "minLength_above_inherited_length",
				schema: `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
					<xsd:simpleType name="base"><xsd:restriction base="xsd:string"><xsd:length value="5"/></xsd:restriction></xsd:simpleType>
					<xsd:element name="e"><xsd:simpleType>
						<xsd:restriction base="base"><xsd:minLength value="6"/></xsd:restriction>
					</xsd:simpleType></xsd:element></xsd:schema>`,
			},
			{
				// Inherited minLength=10, derived length=5 < minLength.
				name: "length_below_inherited_minLength",
				schema: `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
					<xsd:simpleType name="base"><xsd:restriction base="xsd:string"><xsd:minLength value="10"/></xsd:restriction></xsd:simpleType>
					<xsd:element name="e"><xsd:simpleType>
						<xsd:restriction base="base"><xsd:length value="5"/></xsd:restriction>
					</xsd:simpleType></xsd:element></xsd:schema>`,
			},
			{
				// Inherited maxLength=5, derived length=10 > maxLength.
				name: "length_above_inherited_maxLength",
				schema: `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
					<xsd:simpleType name="base"><xsd:restriction base="xsd:string"><xsd:maxLength value="5"/></xsd:restriction></xsd:simpleType>
					<xsd:element name="e"><xsd:simpleType>
						<xsd:restriction base="base"><xsd:length value="10"/></xsd:restriction>
					</xsd:simpleType></xsd:element></xsd:schema>`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				hasErr, msg := compile(t, tc.schema)
				require.True(t, hasErr, "expected schema error, got none: %s", msg)
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
				// length=5 co-occurring with a minLength=1 that merely RESTATES the
				// intrinsic minLength=1 inherited from xs:IDREFS (own 1 == inherited 1,
				// 1 <= 5): permitted by W3C bug 6446 (W3C IDREFS_length006).
				name: "length_with_restated_intrinsic_minLength_on_IDREFS",
				schema: `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
					<xsd:element name="e"><xsd:simpleType>
						<xsd:restriction base="xsd:IDREFS">
							<xsd:length value="5"/>
							<xsd:minLength value="1"/>
						</xsd:restriction>
					</xsd:simpleType></xsd:element></xsd:schema>`,
			},
			{
				// length=5 alone on xs:NMTOKENS: the intrinsic minLength=1 is purely
				// inherited (not restated), so length may co-occur with it.
				name: "length_alone_on_NMTOKENS",
				schema: `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
					<xsd:element name="e"><xsd:simpleType>
						<xsd:restriction base="xsd:NMTOKENS"><xsd:length value="5"/></xsd:restriction>
					</xsd:simpleType></xsd:element></xsd:schema>`,
			},
			{
				// A minLength=1 inherited through MULTIPLE levels (a pattern-only
				// intermediate) restated alongside length=5 (own 1 == effective inherited
				// 1, 1 <= 5): still a valid restatement, must not regress.
				name: "length_with_multilevel_restated_minLength",
				schema: `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
					<xsd:simpleType name="base"><xsd:restriction base="xsd:string"><xsd:minLength value="1"/></xsd:restriction></xsd:simpleType>
					<xsd:simpleType name="mid"><xsd:restriction base="base"><xsd:pattern value=".*"/></xsd:restriction></xsd:simpleType>
					<xsd:element name="e"><xsd:simpleType>
						<xsd:restriction base="mid">
							<xsd:length value="5"/>
							<xsd:minLength value="1"/>
						</xsd:restriction>
					</xsd:simpleType></xsd:element></xsd:schema>`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				hasErr, msg := compile(t, tc.schema)
				require.False(t, hasErr, "expected no schema error, got: %s", msg)
			})
		}
	})
}
