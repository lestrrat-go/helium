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
				name: "all compositor with duplicate same-name member",
				schemaXML: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:all>
        <xs:element name="a" type="xs:int"/>
        <xs:element name="a" type="xs:int"/>
      </xs:all>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
			},
			{
				// `(a, a?){2}`: the repeated unit `(a, a?)` is non-nullable, but
				// each iteration's optional trailing `a?` overlaps the next
				// iteration's required leading `a` — after an `a` you cannot tell
				// if it is the current iteration's optional `a?` or the next
				// iteration's required `a`. Non-deterministic.
				name: "repeated unit with optional tail overlapping next iteration",
				schemaXML: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence minOccurs="2" maxOccurs="2">
        <xs:element name="a" type="xs:int"/>
        <xs:element name="a" type="xs:int" minOccurs="0"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
			},
			{
				// `(a, a?){257}`: same boundary nondeterminism as the count=2 case.
				// A large count must NOT collapse the repeated unit to a single
				// copy (which would drop the inter-iteration boundary followpos and
				// wrongly accept the model). For determinism analysis, U{n} with a
				// non-nullable unit and n>=2 is invariant in n: U{2} already
				// exposes every boundary overlap, so the >cap collapse keeps 2
				// copies.
				name: "large repeated unit with optional tail overlapping next iteration",
				schemaXML: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence minOccurs="257" maxOccurs="257">
        <xs:element name="a" type="xs:int"/>
        <xs:element name="a" type="xs:int" minOccurs="0"/>
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
			{
				// An optional `##any` followed by `##targetNamespace`: ##any
				// matches every namespace, so after the optional wildcard is
				// skipped a target-namespace element could match either
				// wildcard. The namespace sets INTERSECT → non-deterministic.
				name: "overlapping wildcards (##any then ##targetNamespace)",
				schemaXML: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:any namespace="##any" processContents="skip" minOccurs="0"/>
        <xs:any namespace="##targetNamespace" processContents="skip"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
			},
			{
				// Two optional `##any` wildcards in sequence: an input in any
				// namespace can match either, and skipping the first
				// re-introduces the ambiguity. Non-deterministic.
				name: "overlapping wildcards (two ##any)",
				schemaXML: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:any namespace="##any" processContents="skip" minOccurs="0"/>
        <xs:any namespace="##any" processContents="skip"/>
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
			{
				// `(a?, b), b`: the trailing `b` is REQUIRED, so the optional `a`'s
				// position must NOT remain in the inner group's lastpos. A
				// previous-segment-nullability bug kept `a` in lastpos and falsely
				// flagged this deterministic model.
				name: "optional prefix before a required element keeps determinism",
				schemaXML: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:sequence>
          <xs:element name="a" type="xs:int" minOccurs="0"/>
          <xs:element name="b" type="xs:int"/>
        </xs:sequence>
        <xs:element name="b" type="xs:int"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
			},
			{
				// `a{2}, a`: a finite counted repetition followed by another `a`.
				// Each of the three `a` occurrences is a distinct position, so the
				// model is deterministic. Treating maxOccurs="2" as an unbounded loop
				// falsely flags it.
				name: "finite counted repetition followed by the same element",
				schemaXML: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="a" type="xs:int" minOccurs="2" maxOccurs="2"/>
        <xs:element name="a" type="xs:int"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
			},
			{
				// `a{257}, a`: a large finite EXACT count followed by another `a`.
				// A required exact run is a deterministic chain regardless of length;
				// collapsing it into a looping optional tail past an expansion cap
				// manufactures a back-edge that falsely flags this model. The >cap
				// collapse keeps 2 required copies (no back-edge), preserving the
				// deterministic `a{3}` shape. xmllint accepts it.
				name: "large finite exact repetition followed by the same element",
				schemaXML: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="a" type="xs:int" minOccurs="257" maxOccurs="257"/>
        <xs:element name="a" type="xs:int"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
			},
			{
				// `(a, a?)` once: a single occurrence of the repeated unit is
				// deterministic — there is no next iteration to create a boundary
				// overlap. Regression guard distinguishing count==1 from count>=2.
				name: "single occurrence of unit with optional tail",
				schemaXML: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="a" type="xs:int"/>
        <xs:element name="a" type="xs:int" minOccurs="0"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
			},
			{
				// `##other?, ##targetNamespace`: the two wildcards match DISJOINT
				// namespace sets — ##other matches any namespace except the target
				// namespace (and absent), ##targetNamespace matches only the
				// target namespace. After an element you can always tell which
				// wildcard matched, so the model is deterministic. xmllint accepts
				// it; a conservative "two wildcards always overlap" check falsely
				// rejected it.
				name: "disjoint wildcards (##other then ##targetNamespace)",
				schemaXML: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:any namespace="##other" processContents="skip" minOccurs="0"/>
        <xs:any namespace="##targetNamespace" processContents="skip"/>
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

// TestUPAFiniteCountedRepetitionInstance verifies that a deterministic counted
// model `a{2}, a` not only compiles cleanly but also validates an instance with
// three `a` children (xmllint accepts this schema + instance).
func TestUPAFiniteCountedRepetitionInstance(t *testing.T) {
	t.Parallel()

	const schemaSrc = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="a" type="xs:int" minOccurs="2" maxOccurs="2"/>
        <xs:element name="a" type="xs:int"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaSrc))
	require.NoError(t, err)
	schema, err := xsd.NewCompiler().Compile(t.Context(), doc)
	require.NoError(t, err)

	inst, err := helium.NewParser().Parse(t.Context(), []byte(
		`<?xml version="1.0"?><root><a>1</a><a>2</a><a>3</a></root>`))
	require.NoError(t, err)
	require.NoError(t, xsd.NewValidator(schema).Validate(t.Context(), inst))
}
