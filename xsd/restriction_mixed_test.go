package xsd_test

import (
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestComplexContentRestrictionMixedness covers cos-ct-restricts clause 5.3.2
// (§3.4.6.4 Derivation Valid (Restriction, Complex)): a complexContent
// <xs:restriction> whose {content type} is mixed requires the base's {content
// type} to be mixed too. The rule is ASYMMETRIC (unlike the symmetric extension
// rule cos-ct-extends): a mixed base MAY be restricted to element-only, but an
// element-only base may NOT be restricted to mixed. Version-independent; enforced
// by the default (XSD 1.0) compiler. Mirrors W3C xmlschema msMeta ComplexType_w3c
// tests ctF006 and ctZ010e (both expected invalid).
func TestComplexContentRestrictionMixedness(t *testing.T) {
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
				// ctF006: element-only base restricted to mixed derived.
				name: "ctF006_elementonly_base_mixed_restriction",
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
					<xs:complexType name="myType">
						<xs:choice>
							<xs:element name="myElement" type="xs:string"/>
							<xs:element name="myElement2" type="xs:string"/>
						</xs:choice>
					</xs:complexType>
					<xs:complexType name="fooType">
						<xs:complexContent mixed="true">
							<xs:restriction base="myType">
								<xs:sequence>
									<xs:element name="myElement2" type="xs:string"/>
								</xs:sequence>
							</xs:restriction>
						</xs:complexContent>
					</xs:complexType>
					<xs:element name="root" type="fooType"/>
				</xs:schema>`,
			},
			{
				// ctZ010e: element-only base restricted to mixed derived (local type).
				name: "ctZ010e_elementonly_base_mixed_restriction_local",
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
					<xs:complexType name="base">
						<xs:sequence>
							<xs:element name="bar" type="xs:string" minOccurs="0" maxOccurs="unbounded"/>
						</xs:sequence>
					</xs:complexType>
					<xs:element name="foo">
						<xs:complexType mixed="true">
							<xs:complexContent mixed="true">
								<xs:restriction base="base">
									<xs:sequence>
										<xs:element name="bar" type="xs:string" minOccurs="0" maxOccurs="unbounded"/>
									</xs:sequence>
								</xs:restriction>
							</xs:complexContent>
						</xs:complexType>
					</xs:element>
				</xs:schema>`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				bad, msg := compile(t, tc.schema)
				require.True(t, bad, "expected schema to be rejected")
				require.Contains(t, msg, "must either 'mixed' or 'element-only'")
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
				// mixed base restricted to mixed derived — allowed.
				name: "mixed_base_mixed_restriction",
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
					<xs:complexType name="base" mixed="true">
						<xs:sequence>
							<xs:element name="bar" type="xs:string" minOccurs="0" maxOccurs="unbounded"/>
						</xs:sequence>
					</xs:complexType>
					<xs:complexType name="derived">
						<xs:complexContent mixed="true">
							<xs:restriction base="base">
								<xs:sequence>
									<xs:element name="bar" type="xs:string" minOccurs="0" maxOccurs="unbounded"/>
								</xs:sequence>
							</xs:restriction>
						</xs:complexContent>
					</xs:complexType>
					<xs:element name="root" type="derived"/>
				</xs:schema>`,
			},
			{
				// element-only base restricted to element-only derived — allowed.
				name: "elementonly_base_elementonly_restriction",
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
					<xs:complexType name="base">
						<xs:sequence>
							<xs:element name="bar" type="xs:string" minOccurs="0" maxOccurs="unbounded"/>
						</xs:sequence>
					</xs:complexType>
					<xs:complexType name="derived">
						<xs:complexContent>
							<xs:restriction base="base">
								<xs:sequence>
									<xs:element name="bar" type="xs:string" minOccurs="0" maxOccurs="unbounded"/>
								</xs:sequence>
							</xs:restriction>
						</xs:complexContent>
					</xs:complexType>
					<xs:element name="root" type="derived"/>
				</xs:schema>`,
			},
			{
				// mixed base restricted to element-only derived — allowed (asymmetry
				// vs extension; cos-ct-restricts clause 5.3.2.2).
				name: "mixed_base_elementonly_restriction",
				schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
					<xs:complexType name="base" mixed="true">
						<xs:sequence>
							<xs:element name="bar" type="xs:string" minOccurs="0" maxOccurs="unbounded"/>
						</xs:sequence>
					</xs:complexType>
					<xs:complexType name="derived">
						<xs:complexContent>
							<xs:restriction base="base">
								<xs:sequence>
									<xs:element name="bar" type="xs:string" minOccurs="0" maxOccurs="unbounded"/>
								</xs:sequence>
							</xs:restriction>
						</xs:complexContent>
					</xs:complexType>
					<xs:element name="root" type="derived"/>
				</xs:schema>`,
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
