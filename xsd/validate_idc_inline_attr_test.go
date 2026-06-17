package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestIDCInlineAttrTypeValueSpace covers identity-constraint key comparison when
// the constrained attribute is declared with an INLINE anonymous <xs:simpleType>
// rather than a named type reference. The IDC key canonicalizer must resolve the
// inline type the same way the content validator does (au.Type before
// au.TypeName), so xs:integer fields "5" and "+5" denote the same value and must
// collide for uniqueness and match for keyrefs.
func TestIDCInlineAttrTypeValueSpace(t *testing.T) {
	t.Parallel()

	const uniqueSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" maxOccurs="unbounded">
          <xs:complexType>
            <xs:attribute name="n">
              <xs:simpleType>
                <xs:restriction base="xs:integer"/>
              </xs:simpleType>
            </xs:attribute>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
    <xs:unique name="itemKey">
      <xs:selector xpath="item"/>
      <xs:field xpath="@n"/>
    </xs:unique>
  </xs:element>
</xs:schema>`

	const keyrefSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" maxOccurs="unbounded">
          <xs:complexType>
            <xs:attribute name="n">
              <xs:simpleType>
                <xs:restriction base="xs:integer"/>
              </xs:simpleType>
            </xs:attribute>
          </xs:complexType>
        </xs:element>
        <xs:element name="ref" maxOccurs="unbounded">
          <xs:complexType>
            <xs:attribute name="r">
              <xs:simpleType>
                <xs:restriction base="xs:integer"/>
              </xs:simpleType>
            </xs:attribute>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
    <xs:key name="itemKey">
      <xs:selector xpath="item"/>
      <xs:field xpath="@n"/>
    </xs:key>
    <xs:keyref name="itemRef" refer="itemKey">
      <xs:selector xpath="ref"/>
      <xs:field xpath="@r"/>
    </xs:keyref>
  </xs:element>
</xs:schema>`

	compile := func(t *testing.T, src string) xsd.Validator {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
		require.NoError(t, err)
		schema, err := xsd.NewCompiler().Compile(t.Context(), doc)
		require.NoError(t, err)
		return xsd.NewValidator(schema)
	}

	cases := []struct {
		name     string
		schema   string
		instance string
		valid    bool
	}{
		{
			name:     "inline integer unique 5 and +5 is duplicate",
			schema:   uniqueSchema,
			instance: `<root><item n="5"/><item n="+5"/></root>`,
			valid:    false,
		},
		{
			name:     "inline integer unique 5 and 6 is not duplicate",
			schema:   uniqueSchema,
			instance: `<root><item n="5"/><item n="6"/></root>`,
			valid:    true,
		},
		{
			name:     "inline keyref +5 matches key 5",
			schema:   keyrefSchema,
			instance: `<root><item n="5"/><ref r="+5"/></root>`,
			valid:    true,
		},
		{
			name:     "inline dangling keyref still errors",
			schema:   keyrefSchema,
			instance: `<root><item n="5"/><ref r="7"/></root>`,
			valid:    false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			v := compile(t, tc.schema)

			doc, err := helium.NewParser().Parse(t.Context(), []byte(tc.instance))
			require.NoError(t, err)

			var errs string
			err = validateWithOutput(t, v, doc, &errs)
			if tc.valid {
				require.NoError(t, err, "expected valid, got errors: %s", errs)
				return
			}
			require.Error(t, err, "expected validation error")
		})
	}
}
