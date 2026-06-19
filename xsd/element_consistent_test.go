package xsd_test

import (
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestElementConsistent verifies the XSD cos-element-consistent (Element
// Declarations Consistent) constraint: two element declarations with the same
// expanded name appearing in one effective content model must have the same
// type definition. Before the fix an inconsistent pair such as
// <xs:element name="a" type="xs:int"/> followed by
// <xs:element name="a" type="xs:string"/> compiled silently.
func TestElementConsistent(t *testing.T) {
	t.Parallel()

	compileErrors := func(t *testing.T, schemaXML string) string {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, err = xsd.NewCompiler().Label("test.xsd").ErrorHandler(collector).Compile(t.Context(), doc)
		require.NoError(t, err)
		// Close the collector before reading so the async sink is fully drained
		// and the read is not flaky under parallel/-race load (mirrors
		// compileErrorsExact). Without this, the cos-element-consistent diagnostic
		// can still be in flight when Errors() is read.
		require.NoError(t, collector.Close())
		_, errors := partitionCompileErrors(collector.Errors())
		return errors
	}

	const wantMsg = "but different type definitions, appear in the content model."

	t.Run("rejects", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name      string
			schemaXML string
		}{
			{
				name: "different builtin types in a sequence",
				schemaXML: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="a" type="xs:int"/>
        <xs:element name="a" type="xs:string"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
			},
			{
				// A global element ref and a local element of the same name but a
				// different type. The sequence is deterministic (UPA-clean) because
				// the two occurrences are ordered, so the inconsistency is caught by
				// cos-element-consistent rather than UPA.
				name: "global ref vs local of a different type",
				schemaXML: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="a" type="xs:int"/>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element ref="a"/>
        <xs:element name="a" type="xs:string"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
			},
			{
				name: "inconsistent across a nested model group",
				schemaXML: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="a" type="xs:int"/>
        <xs:sequence>
          <xs:element name="a" type="xs:string"/>
        </xs:sequence>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
			},
			{
				name: "inconsistent across an expanded group reference",
				schemaXML: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:group name="g">
    <xs:sequence>
      <xs:element name="a" type="xs:string"/>
    </xs:sequence>
  </xs:group>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="a" type="xs:int"/>
        <xs:group ref="g"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
			},
			{
				name: "named type vs inline anonymous type",
				schemaXML: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="a" type="xs:int"/>
        <xs:element name="a">
          <xs:simpleType>
            <xs:restriction base="xs:string"/>
          </xs:simpleType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				got := compileErrors(t, tc.schemaXML)
				require.Contains(t, got, wantMsg, "expected cos-element-consistent error")
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
				name: "same builtin type repeated in a sequence",
				schemaXML: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="a" type="xs:int"/>
        <xs:element name="a" type="xs:int"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
			},
			{
				name: "same named user type repeated",
				schemaXML: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="myInt">
    <xs:restriction base="xs:int"/>
  </xs:simpleType>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="a" type="myInt"/>
        <xs:element name="a" type="myInt"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
			},
			{
				name: "same global element referenced twice",
				schemaXML: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="a" type="xs:int"/>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element ref="a"/>
        <xs:element ref="a"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
			},
			{
				name: "same-named elements in different complex types",
				schemaXML: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="a" type="xs:int"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
  <xs:element name="other">
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
				got := compileErrors(t, tc.schemaXML)
				require.NotContains(t, got, wantMsg, "did not expect cos-element-consistent error")
				require.Empty(t, strings.TrimSpace(got), "expected a clean compile")
			})
		}
	})
}
