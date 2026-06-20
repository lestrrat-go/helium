package xsd_test

import (
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestUPADeterminism verifies the XSD cos-nonambig (Unique Particle
// Attribution) constraint via a position automaton. A local first/last
// heuristic accepts genuinely ambiguous models such as `a?, b?, a`: when the
// first optional `a` is skipped, an input `a` can match either the first or the
// final particle, so the model is non-deterministic. The automaton computes
// nullable/firstpos/followpos over the particle tree and flags the overlap.
func TestUPADeterminism(t *testing.T) {
	t.Parallel()

	compileErrors := func(t *testing.T, schemaXML string) string {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, err = xsd.NewCompiler().Label("test.xsd").ErrorHandler(collector).Compile(t.Context(), doc)
		requireCompileResultErr(t, err)
		require.NoError(t, collector.Close())
		_, errors := partitionCompileErrors(collector.Errors())
		return errors
	}

	const wantMsg = "The content model is not determinist."

	t.Run("rejects", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name      string
			schemaXML string
		}{
			{
				// `a?, b?, a`: skipping the first optional `a` makes the final `a`
				// ambiguous with the first one. The local first/last heuristic only
				// inspects adjacent particles and misses this non-adjacent overlap.
				name: "optional prefix re-introduces an earlier element",
				schemaXML: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="a" type="xs:int" minOccurs="0"/>
        <xs:element name="b" type="xs:int" minOccurs="0"/>
        <xs:element name="a" type="xs:int"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
			},
			{
				// `(a?, b?), a`: same ambiguity wrapped in a nested group.
				name: "optional nested group re-introduces an earlier element",
				schemaXML: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:sequence>
          <xs:element name="a" type="xs:int" minOccurs="0"/>
          <xs:element name="b" type="xs:int" minOccurs="0"/>
        </xs:sequence>
        <xs:element name="a" type="xs:int"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
			},
			{
				// A wildcard followed by an element it can also match: after a
				// nullable run, an `a` could start the wildcard or the element.
				name: "wildcard overlaps a following element of the same namespace",
				schemaXML: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:any namespace="##any" processContents="skip" minOccurs="0"/>
        <xs:element name="a" type="xs:int"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				got := compileErrors(t, tc.schemaXML)
				require.Contains(t, got, wantMsg, "expected cos-nonambig (UPA) error")
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
				// `a?, b?, c`: all three element names are distinct, so no input can
				// match two particles. Deterministic.
				name: "optional prefix with a distinct trailing element",
				schemaXML: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="a" type="xs:int" minOccurs="0"/>
        <xs:element name="b" type="xs:int" minOccurs="0"/>
        <xs:element name="c" type="xs:int"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
			},
			{
				// A simple ordered sequence of distinct elements is trivially
				// deterministic.
				name: "ordered distinct elements",
				schemaXML: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="a" type="xs:int"/>
        <xs:element name="b" type="xs:int"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				got := compileErrors(t, tc.schemaXML)
				require.NotContains(t, got, wantMsg, "did not expect a UPA error")
				require.Empty(t, strings.TrimSpace(got), "expected a clean compile")
			})
		}
	})
}
