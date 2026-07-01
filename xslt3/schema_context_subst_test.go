package xslt3

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

func TestSchemaRegistrySubstitutionGroupMembersUseEligibleTransitiveSet(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="elem0" type="xs:string"/>
  <xs:element name="elem1" type="xs:string" block="substitution"/>
  <xs:element name="elem2" type="xs:string" substitutionGroup="elem0 elem1"/>
  <xs:element name="elem3" type="xs:string" substitutionGroup="elem2"/>
  <xs:element name="blocked" type="xs:string" substitutionGroup="elem1"/>
</xs:schema>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	schema, err := xsd.NewCompiler().Version(xsd.Version11).Label("test.xsd").Compile(t.Context(), doc)
	require.NoError(t, err)

	reg := &schemaRegistry{schemas: []*xsd.Schema{schema}}
	require.True(t, reg.IsSubstitutionGroupMember("elem3", "", "elem0", ""), "schema-element(elem0) should see transitive members")
	require.False(t, reg.IsSubstitutionGroupMember("blocked", "", "elem1", ""), "blocked heads must not expose members to xslt3")
}
