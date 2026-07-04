package xsd_test

import (
	"fmt"
	"testing"

	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// xs:NCName and xs:QName both have their whiteSpace facet fixed to "collapse",
// so every NCName-valued (@name) and QName-valued (@type/@ref/@base/@itemType/
// @memberTypes) schema attribute is whitespace-collapsed before it is stored,
// validated, and resolved. A padded-but-valid value must compile; an internal-
// whitespace value (still not a valid NCName/QName after collapsing) must be
// rejected. Version-independent: enforced under both XSD 1.0 and 1.1.
func TestSchemaAttrWhitespaceCollapse(t *testing.T) {
	t.Parallel()

	// Finding 1: a collapsed @name is what is REGISTERED — a ref to the trimmed
	// name resolves against the registered {tns}child declaration. If the trailing
	// space were retained the ref="child" would dangle and compilation would fail.
	t.Run("collapsed-name-is-registered/global-ref-resolves", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element ref="child"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
  <xs:element name="child " type="xs:string"/>
</xs:schema>`
		for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
			_, errs, cerr := compileWith(t, v, schemaXML)
			require.NoError(t, cerr, "version=%v must register the collapsed name so ref resolves: %s", v, errs)
		}
	})

	// The collapsed @name is what an instance is matched against, too: a global
	// element declared with a trailing-space @name validates an instance bearing
	// the trimmed name.
	t.Run("collapsed-name-is-registered/instance-validates", func(t *testing.T) {
		t.Parallel()
		const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root " type="xs:string"/>
</xs:schema>`
		errs, err := validateInstance(t, schemaXML, `<root>hello</root>`)
		require.NoError(t, err, "instance must validate against the collapsed declaration name: %s", errs)
	})

	// Findings 2 & 3: a QName-valued attribute with surrounding whitespace collapses
	// to a valid QName and resolves — at every QName-valued read site.
	validQName := []struct {
		name   string
		schema string
	}{
		{
			"element-type",
			`<xs:element name="e" type="  xs:string "/>`,
		},
		{
			"attribute-type",
			`<xs:element name="e">
    <xs:complexType>
      <xs:attribute name="a" type=" xs:string "/>
    </xs:complexType>
  </xs:element>`,
		},
		{
			"attribute-ref",
			`<xs:attribute name="ga" type="xs:string"/>
  <xs:element name="e">
    <xs:complexType>
      <xs:attribute ref=" ga "/>
    </xs:complexType>
  </xs:element>`,
		},
		{
			"restriction-base",
			`<xs:simpleType name="st">
    <xs:restriction base="   xs:string ">
      <xs:maxLength value="3"/>
    </xs:restriction>
  </xs:simpleType>`,
		},
		{
			"list-itemType",
			`<xs:simpleType name="st">
    <xs:list itemType=" xs:int "/>
  </xs:simpleType>`,
		},
	}
	for _, tc := range validQName {
		t.Run("padded-qname-resolves/"+tc.name, func(t *testing.T) {
			t.Parallel()
			schemaXML := fmt.Sprintf(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  %s
</xs:schema>`, tc.schema)
			for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
				_, errs, cerr := compileWith(t, v, schemaXML)
				require.NoError(t, cerr, "version=%v must accept padded QName value: %s", v, errs)
			}
		})
	}

	// An internal-whitespace value stays invalid after collapsing and must be
	// rejected — never routed into component lookup as a bogus local name.
	rejectInternalWS := []struct {
		name   string
		schema string
		want   string
	}{
		{
			"element-name",
			`<xs:element name="a b" type="xs:string"/>`,
			"is not a valid 'NCName'",
		},
		{
			"attribute-name",
			`<xs:element name="e">
    <xs:complexType>
      <xs:attribute name="a b" type="xs:string"/>
    </xs:complexType>
  </xs:element>`,
			"is not a valid 'NCName'",
		},
		{
			"restriction-base",
			`<xs:simpleType name="st">
    <xs:restriction base="a b">
      <xs:maxLength value="3"/>
    </xs:restriction>
  </xs:simpleType>`,
			"'a b' is not a valid QName",
		},
		{
			"attribute-type",
			`<xs:element name="e">
    <xs:complexType>
      <xs:attribute name="a" type="a b"/>
    </xs:complexType>
  </xs:element>`,
			"'a b' is not a valid QName",
		},
		{
			"element-type",
			`<xs:element name="e" type="a b"/>`,
			"'a b' is not a valid QName",
		},
	}
	for _, tc := range rejectInternalWS {
		t.Run("internal-whitespace-rejected/"+tc.name, func(t *testing.T) {
			t.Parallel()
			schemaXML := fmt.Sprintf(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  %s
</xs:schema>`, tc.schema)
			for _, v := range []xsd.Version{xsd.Version10, xsd.Version11} {
				schema, errs, cerr := compileWith(t, v, schemaXML)
				require.ErrorIs(t, cerr, xsd.ErrCompilationFailed, "version=%v must reject internal-whitespace value", v)
				require.Nil(t, schema)
				require.Contains(t, errs, tc.want, "version=%v", v)
			}
		})
	}
}
