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
		requireCompileResultErr(t, err)
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
				// A substitution-group member resolves its declared type through
				// the head (xs:int). A same-named local element of a genuinely
				// different type (xs:string) is still inconsistent and must be
				// rejected even after substitution-group-aware resolution.
				name: "untyped substitution-group member vs same-named local of a different type",
				schemaXML: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="head" type="xs:int"/>
  <xs:element name="a" substitutionGroup="head"/>
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
				// The content model references the HEAD ('head'). The head's
				// particle implicitly contains its member 'a' (typed via the head
				// as xs:int). A same-named local element 'a' of a genuinely
				// different type (xs:string) therefore collides by name with the
				// implicitly-contained member and is inconsistent. This requires
				// the check to fold substitution-group members into the content
				// model, which in turn requires schema.substGroups to be built
				// before the check runs.
				name: "head ref vs same-named local of a different type",
				schemaXML: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="head" type="xs:int"/>
  <xs:element name="a" type="xs:int" substitutionGroup="head"/>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element ref="head"/>
        <xs:element name="a" type="xs:string"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
			},
			{
				// A standalone named group that no complex type references must
				// still be checked: its two same-named elements have different
				// types and are inconsistent.
				name: "inconsistent standalone named group",
				schemaXML: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:group name="g">
    <xs:sequence>
      <xs:element name="a" type="xs:int"/>
      <xs:element name="a" type="xs:string"/>
    </xs:sequence>
  </xs:group>
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
			{
				// A prohibited element particle (maxOccurs="0") maps to NO
				// particle, so it is not part of the effective content model and
				// cannot conflict with a real same-named particle of a different
				// type.
				name: "prohibited element particle of a different type is ignored",
				schemaXML: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="a" type="xs:int"/>
        <xs:element name="a" type="xs:string" minOccurs="0" maxOccurs="0"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
			},
			{
				// A substitution-group member declared without an explicit type
				// has Type == nil; its effective declared type is the head's
				// (xs:int here). Referencing the member alongside a same-named
				// local element of that same head type must NOT be flagged
				// inconsistent. Before the fix the raw nil Type of the member
				// compared unequal to the local's xs:int and false-rejected.
				name: "substitution-group member with no explicit type matches its head type",
				schemaXML: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="head" type="xs:int"/>
  <xs:element name="a" type="xs:int" substitutionGroup="head"/>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element ref="a"/>
        <xs:element name="a" type="xs:int"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
			},
			{
				// As above but the member carries no type at all, so its
				// effective declared type is resolved through the head. It must
				// compare equal to a same-named local of the head's type.
				name: "untyped substitution-group member resolves through head",
				schemaXML: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="head" type="xs:int"/>
  <xs:element name="a" substitutionGroup="head"/>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element ref="a"/>
        <xs:element name="a" type="xs:int"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
			},
			{
				// A head reference folds in the member 'a' (typed xs:int via the
				// head). A same-named local element 'a' of that SAME type (xs:int)
				// is consistent and must NOT be flagged. Exercises the
				// substitution-group-member folding on the accept side.
				name: "head ref with a consistent same-named local",
				schemaXML: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="head" type="xs:int"/>
  <xs:element name="a" type="xs:int" substitutionGroup="head"/>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element ref="head"/>
        <xs:element name="a" type="xs:int"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
			},
			{
				// A consistent standalone named group (same name, same type) must
				// compile cleanly even though no complex type references it.
				name: "consistent standalone named group",
				schemaXML: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:group name="g">
    <xs:sequence>
      <xs:element name="a" type="xs:int"/>
      <xs:element name="a" type="xs:int"/>
    </xs:sequence>
  </xs:group>
</xs:schema>`,
			},
			{
				// The head carries block="substitution", so NO member may
				// substitute for it. A head reference therefore implicitly
				// contains none of its members, and a same-named local element
				// 'a' of a genuinely different type (xs:string) does NOT collide
				// with the (non-folded) member. This must compile cleanly,
				// mirroring elemMatchesDeclOrSubst's block="substitution" gate.
				// Before the fix the member was folded in regardless of block and
				// false-rejected.
				name: "head with block=substitution does not fold members",
				schemaXML: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="head" type="xs:int" block="substitution"/>
  <xs:element name="a" type="xs:int" substitutionGroup="head"/>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element ref="head"/>
        <xs:element name="a" type="xs:string"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
			},
			{
				// The member 'a' derives its type from the head's type by
				// extension, but the head blocks extension (block="extension"),
				// so 'a' cannot actually substitute for the head. The member is
				// therefore not implicitly contained, and a same-named local 'a'
				// of a different type (xs:string) does not collide with it. This
				// exercises the per-member isDerivationBlocked gate.
				name: "derivation-blocked member is not folded",
				schemaXML: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base"/>
  <xs:complexType name="ext">
    <xs:complexContent>
      <xs:extension base="base"/>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="head" type="base" block="extension"/>
  <xs:element name="a" type="ext" substitutionGroup="head"/>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element ref="head"/>
        <xs:element name="a" type="xs:string"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`,
			},
			{
				// A prohibited group reference (maxOccurs="0") maps to NO
				// particle, so the element declarations it would otherwise
				// expand to do not enter the effective content model.
				name: "prohibited group ref with a different type is ignored",
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
        <xs:group ref="g" minOccurs="0" maxOccurs="0"/>
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
