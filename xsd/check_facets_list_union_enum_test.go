package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestEnumValueAgainstBaseListUnion verifies that checkEnumValueAgainstBase
// rejects an enumeration facet whose value is not datatype-valid against a
// LIST or UNION base, and that a valid list/union enumeration value still
// compiles. Per XSD §3.16 each enumeration {value} must be datatype-valid
// against the {base type definition}; "+NaN" is not in the xs:float lexical
// space, so a list itemType="xs:float" or a union with an xs:float member that
// enumerates "+NaN" is a schema in error and must be rejected at COMPILE time.
func TestEnumValueAgainstBaseListUnion(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name             string
		schema           string
		wantCompileError bool
		offending        string
	}{
		{
			name: "list float +NaN enum rejected",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="floatList">
    <xs:list itemType="xs:float"/>
  </xs:simpleType>
  <xs:simpleType name="enumFloatList">
    <xs:restriction base="floatList">
      <xs:enumeration value="+NaN"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="root" type="enumFloatList"/>
</xs:schema>`,
			wantCompileError: true,
			offending:        "+NaN",
		},
		{
			name: "list float partial +NaN item rejected",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="floatList">
    <xs:list itemType="xs:float"/>
  </xs:simpleType>
  <xs:simpleType name="enumFloatList">
    <xs:restriction base="floatList">
      <xs:enumeration value="1.5 +NaN 2.5"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="root" type="enumFloatList"/>
</xs:schema>`,
			wantCompileError: true,
			offending:        "1.5 +NaN 2.5",
		},
		{
			name: "union float +NaN admitted by string member compiles",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="floatOrString">
    <xs:union memberTypes="xs:float xs:string"/>
  </xs:simpleType>
  <xs:simpleType name="enumFloatOrString">
    <xs:restriction base="floatOrString">
      <xs:enumeration value="+NaN"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="root" type="enumFloatOrString"/>
</xs:schema>`,
			// "+NaN" is rejected by the xs:float member, but it IS a valid
			// xs:string, so the union admits it. This documents that a union
			// enumeration value valid against ANY member compiles (the fix must
			// NOT over-reject).
			wantCompileError: false,
		},
		{
			name: "union numeric-only +NaN enum rejected",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="floatOrInt">
    <xs:union memberTypes="xs:float xs:int"/>
  </xs:simpleType>
  <xs:simpleType name="enumFloatOrInt">
    <xs:restriction base="floatOrInt">
      <xs:enumeration value="+NaN"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="root" type="enumFloatOrInt"/>
</xs:schema>`,
			wantCompileError: true,
			offending:        "+NaN",
		},
		{
			name: "list float valid multi-item enum compiles",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="floatList">
    <xs:list itemType="xs:float"/>
  </xs:simpleType>
  <xs:simpleType name="enumFloatList">
    <xs:restriction base="floatList">
      <xs:enumeration value="1.5 NaN 2.5E0"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="root" type="enumFloatList"/>
</xs:schema>`,
			wantCompileError: false,
		},
		{
			name: "union valid enum compiles",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="floatOrInt">
    <xs:union memberTypes="xs:float xs:int"/>
  </xs:simpleType>
  <xs:simpleType name="enumFloatOrInt">
    <xs:restriction base="floatOrInt">
      <xs:enumeration value="1.5"/>
      <xs:enumeration value="42"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="root" type="enumFloatOrInt"/>
</xs:schema>`,
			wantCompileError: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if tc.wantCompileError {
				errs := compileSchemaErrors(t, tc.schema)
				require.NotEmpty(t, errs, "expected a compile error for invalid enumeration value")
				require.Contains(t, errs, "facet 'enumeration'", "expected enumeration-facet compile diagnostic")
				require.Contains(t, errs, tc.offending, "expected the offending enumeration value in the diagnostic")
				return
			}

			doc, err := helium.NewParser().Parse(t.Context(), []byte(tc.schema))
			require.NoError(t, err)
			_, err = xsd.NewCompiler().Compile(t.Context(), doc)
			require.NoError(t, err, "valid list/union enumeration must compile")
		})
	}
}
