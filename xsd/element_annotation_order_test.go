package xsd_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestElementAnnotationOrder verifies the §3.3.2 element content-model ordering
// rule: an <xs:annotation> must be the FIRST child of an <xs:element> and may not
// follow a <simpleType>/<complexType>/identity-constraint child. This is
// version-INDEPENDENT (annotation is the leading term in every XSD version's
// element content model). Fixes W3C MS-Element elemQ004/elemQ006.
func TestElementAnnotationOrder(t *testing.T) {
	t.Parallel()

	const wantMsg = "The content is not valid. Expected is (annotation?"

	t.Run("rejects", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name      string
			schemaXML string
		}{
			{
				// elemQ004: annotation after an inline simpleType.
				name: "annotation after simpleType (global)",
				schemaXML: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="myElem">
    <xs:simpleType>
      <xs:restriction base="xs:string"/>
    </xs:simpleType>
    <xs:annotation>
      <xs:documentation>after</xs:documentation>
    </xs:annotation>
  </xs:element>
</xs:schema>`,
			},
			{
				// elemQ006: annotation after an inline complexType.
				name: "annotation after complexType (global)",
				schemaXML: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="myElem">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="a" type="xs:string"/>
      </xs:sequence>
    </xs:complexType>
    <xs:annotation>
      <xs:documentation>after</xs:documentation>
    </xs:annotation>
  </xs:element>
</xs:schema>`,
			},
			{
				name: "annotation after complexType (local)",
				schemaXML: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="child">
          <xs:complexType/>
          <xs:annotation>
            <xs:documentation>after</xs:documentation>
          </xs:annotation>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
			},
			{
				name: "annotation after identity constraint",
				schemaXML: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="a" type="xs:string"/>
      </xs:sequence>
    </xs:complexType>
    <xs:unique name="u">
      <xs:selector xpath="a"/>
      <xs:field xpath="."/>
    </xs:unique>
    <xs:annotation>
      <xs:documentation>after</xs:documentation>
    </xs:annotation>
  </xs:element>
</xs:schema>`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				errs := compileErrorsExact(t, tc.schemaXML)
				require.Contains(t, errs, wantMsg)
			})
		}
	})

	t.Run("accepts", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name      string
			schemaXML string
		}{
			{
				name: "annotation first, then simpleType",
				schemaXML: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="myElem">
    <xs:annotation>
      <xs:documentation>first</xs:documentation>
    </xs:annotation>
    <xs:simpleType>
      <xs:restriction base="xs:string"/>
    </xs:simpleType>
  </xs:element>
</xs:schema>`,
			},
			{
				name: "annotation first, complexType, then IDC",
				schemaXML: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:annotation>
      <xs:documentation>first</xs:documentation>
    </xs:annotation>
    <xs:complexType>
      <xs:sequence>
        <xs:element name="a" type="xs:string"/>
      </xs:sequence>
    </xs:complexType>
    <xs:unique name="u">
      <xs:selector xpath="a"/>
      <xs:field xpath="."/>
    </xs:unique>
  </xs:element>
</xs:schema>`,
			},
			{
				name: "no annotation, complexType only",
				schemaXML: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="myElem">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="a" type="xs:string"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				errs := compileErrorsExact(t, tc.schemaXML)
				require.False(t, strings.Contains(errs, wantMsg),
					"unexpected annotation-order error: %s", errs)
			})
		}
	})
}
