package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestMissingAttributeTypeReported covers the ctZ002 / xsd003b.e class: an
// <xs:attribute> whose @type names a component that does not resolve to any type
// definition is a schema error (src-resolve). A PREFIXED missing type (in a
// namespace) is never deferrable, so the schema must be rejected — helium
// previously accepted it because a missing attribute @type was never reported.
func TestMissingAttributeTypeReported(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		schema    string
		wantError bool
		offending string
	}{
		{
			// ctZ002: a GLOBAL attribute with a missing XSD-namespace @type
			// (xs:strong is not a built-in), referenced by a complexType.
			name: "global attribute missing xsd-namespace type",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attribute name="attr2" type="xs:strong"/>
  <xs:complexType name="myType">
    <xs:simpleContent>
      <xs:extension base="xs:string">
        <xs:attribute ref="attr2"/>
      </xs:extension>
    </xs:simpleContent>
  </xs:complexType>
</xs:schema>`,
			wantError: true,
			offending: "strong",
		},
		{
			// xsd003b.e: a LOCAL attribute with a missing XSD-namespace @type.
			name: "local attribute missing xsd-namespace type",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="ct">
    <xs:attribute name="add" type="xs:undefined"/>
  </xs:complexType>
</xs:schema>`,
			wantError: true,
			offending: "undefined",
		},
		{
			// A resolvable built-in attribute @type must still compile — the new
			// missing-type report must not over-reject a valid schema.
			name: "resolvable attribute type compiles",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="ct">
    <xs:attribute name="ok" type="xs:string"/>
  </xs:complexType>
</xs:schema>`,
			wantError: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			errs := compileSchemaErrors(t, tc.schema)
			if !tc.wantError {
				require.Empty(t, errs, "valid schema must compile without errors")
				return
			}
			require.NotEmpty(t, errs, "missing attribute @type must be a schema error")
			require.Contains(t, errs, tc.offending, "diagnostic should cite the unresolved type")
		})
	}
}

// TestUnprefixedAttributeRefNoDefaultNamespace covers au_attrdecl00101m1_n: an
// unprefixed <xs:attribute ref="..."> with NO in-scope default namespace resolves
// to NO namespace (attributes never fall back to the schema targetNamespace), so
// it must NOT bind to a same-targetNamespace global attribute. helium previously
// resolved the unprefixed ref to the targetNamespace, wrongly accepting the
// invalid schema.
func TestUnprefixedAttributeRefNoDefaultNamespace(t *testing.T) {
	t.Parallel()

	t.Run("unprefixed ref does not resolve to target namespace", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="urn:t" xmlns:tn="urn:t">
  <xs:attribute name="number" type="xs:integer"/>
  <xs:element name="e">
    <xs:complexType>
      <xs:attribute ref="number" use="required"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		errs := compileSchemaErrors(t, schema)
		require.NotEmpty(t, errs, "unprefixed attribute ref into no namespace must not resolve")
	})

	t.Run("prefixed ref resolves and compiles", func(t *testing.T) {
		t.Parallel()
		schema := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="urn:t" xmlns:tn="urn:t">
  <xs:attribute name="number" type="xs:integer"/>
  <xs:element name="e">
    <xs:complexType>
      <xs:attribute ref="tn:number" use="required"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`
		errs := compileSchemaErrors(t, schema)
		require.Empty(t, errs, "prefixed attribute ref must resolve and compile")
	})
}

// TestExtendEmptyAllByAll covers mgZ003: extending a complex type whose content is
// an EMPTY xs:all by an xs:all (here via an xs:group ref that resolves to an all
// group) yields the derived all as the whole content — the empty base contributes
// nothing, so no wrapping sequence (which would nest an 'all' inside a sequence)
// is formed. A conforming instance must validate.
func TestExtendEmptyAllByAll(t *testing.T) {
	t.Parallel()

	const schema = `<xsd:schema xmlns:xsd="http://www.w3.org/2001/XMLSchema">
	<xsd:group name="group">
		<xsd:all>
			<xsd:element name="e1"/>
		</xsd:all>
	</xsd:group>
	<xsd:element name="doc" type="foo"/>
	<xsd:complexType name="foo">
		<xsd:complexContent>
			<xsd:extension base="bar">
				<xsd:group ref="group"/>
			</xsd:extension>
		</xsd:complexContent>
	</xsd:complexType>
	<xsd:complexType name="bar">
		<xsd:all minOccurs="1" maxOccurs="1"/>
	</xsd:complexType>
</xsd:schema>`

	schemaDOC, err := helium.NewParser().Parse(t.Context(), []byte(schema))
	require.NoError(t, err)
	sch, err := xsd.NewCompiler().Compile(t.Context(), schemaDOC)
	require.NoError(t, err, "schema must compile")

	instDOC, err := helium.NewParser().Parse(t.Context(), []byte("<doc>\n   <e1/>\n</doc>"))
	require.NoError(t, err)
	err = xsd.NewValidator(sch).Validate(t.Context(), instDOC)
	require.NoError(t, err, "conforming instance must validate")
}
