package xsd_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// compileParticleVer compiles a schema string under the given version and
// returns the compile error (nil = valid schema).
func compileParticleVer(t *testing.T, version xsd.Version, s string) error {
	t.Helper()
	_, err := compileVer(t, s, version)
	return err
}

// TestVersion11CircularAttributeGroup covers the XSD 1.1 relaxation that permits
// circular attribute group definitions (W3C bug 15795 / msData attgC010-D015):
// a self-referential or mutually-referential attribute group is a schema error
// in 1.0 but valid in 1.1.
func TestVersion11CircularAttributeGroup(t *testing.T) {
	t.Parallel()

	// Direct self-reference (attgC010-style).
	directSelf := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="T" type="test"/>
  <xs:complexType name="test"><xs:attributeGroup ref="test"/></xs:complexType>
  <xs:attributeGroup name="test">
    <xs:attributeGroup ref="test"/>
    <xs:attribute name="foo" type="xs:int"/>
  </xs:attributeGroup>
</xs:schema>`

	// Indirect mutual reference (attgD015-style): foo -> foobar -> foo.
	indirect := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attributeGroup name="foo"><xs:attributeGroup ref="foobar"/></xs:attributeGroup>
  <xs:attributeGroup name="foobar"><xs:attributeGroup ref="foo"/></xs:attributeGroup>
  <xs:complexType name="t"><xs:attributeGroup ref="foo"/></xs:complexType>
  <xs:element name="doc" type="t"/>
</xs:schema>`

	for name, s := range map[string]string{"direct": directSelf, "indirect": indirect} {
		t.Run(name+"/1.1 valid", func(t *testing.T) {
			t.Parallel()
			require.NoError(t, compileParticleVer(t, xsd.Version11, s))
		})
		t.Run(name+"/1.0 rejected", func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, compileParticleVer(t, xsd.Version10, s), xsd.ErrCompilationFailed)
		})
	}
}

// TestVersion11AllMaxOccursZero covers cos-all-limited relaxed in 1.1 to allow
// an xs:all group's own maxOccurs to be 0 (mgO001/mgO018): valid in 1.1, a
// schema error in 1.0. maxOccurs="2" stays invalid in both.
func TestVersion11AllMaxOccursZero(t *testing.T) {
	t.Parallel()

	maxZero := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="foo">
    <xs:all minOccurs="0" maxOccurs="0"><xs:element name="e1"/></xs:all>
  </xs:complexType>
</xs:schema>`
	maxTwo := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="foo">
    <xs:all maxOccurs="2"><xs:element name="e1"/></xs:all>
  </xs:complexType>
</xs:schema>`

	t.Run("maxOccurs=0 valid in 1.1", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, compileParticleVer(t, xsd.Version11, maxZero))
	})
	t.Run("maxOccurs=0 rejected in 1.0", func(t *testing.T) {
		t.Parallel()
		require.ErrorIs(t, compileParticleVer(t, xsd.Version10, maxZero), xsd.ErrCompilationFailed)
	})
	t.Run("maxOccurs=2 rejected in both", func(t *testing.T) {
		t.Parallel()
		require.ErrorIs(t, compileParticleVer(t, xsd.Version11, maxTwo), xsd.ErrCompilationFailed)
		require.ErrorIs(t, compileParticleVer(t, xsd.Version10, maxTwo), xsd.ErrCompilationFailed)
	})
}

// TestVersion11ParticleLanguageSubset covers the XSD 1.1 content-model
// restriction relaxation: a restriction whose derived language is a subset of
// the base's is valid in 1.1 even where the 1.0 syntactic clauses reject it
// (particlesHa161-style — an element restricting a choice). The SOUNDNESS guard
// verifies a genuine over-restriction (deriving an element the base cannot
// accept) is still rejected in 1.1.
func TestVersion11ParticleLanguageSubset(t *testing.T) {
	t.Parallel()

	// derived sequence(a?) restricts base sequence(choice(a,b)?): every string of
	// the derived ("" or "a") is accepted by the base ("", "a", "b"). Valid 1.1,
	// rejected by the 1.0 syntactic rule.
	elemRestrictsChoice := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base">
    <xs:sequence>
      <xs:choice minOccurs="0">
        <xs:element name="a" type="xs:string"/>
        <xs:element name="b" type="xs:string"/>
      </xs:choice>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="derived">
    <xs:complexContent>
      <xs:restriction base="base">
        <xs:sequence><xs:element name="a" type="xs:string" minOccurs="0"/></xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`

	// derived emits <c>, which the base (a|b) never accepts: NOT a subset, so the
	// language-inclusion fallback must NOT rescue it.
	overRestriction := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base">
    <xs:sequence>
      <xs:choice minOccurs="0">
        <xs:element name="a" type="xs:string"/>
        <xs:element name="b" type="xs:string"/>
      </xs:choice>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="derived">
    <xs:complexContent>
      <xs:restriction base="base">
        <xs:sequence><xs:element name="c" type="xs:string" minOccurs="0"/></xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`

	// derived narrows the element type incompatibly (base a is xs:string, derived a
	// is xs:int, not a restriction of xs:string): rejected even though the NAME
	// language is a subset.
	typeIncompatible := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base">
    <xs:sequence>
      <xs:choice minOccurs="0">
        <xs:element name="a" type="xs:string"/>
        <xs:element name="b" type="xs:string"/>
      </xs:choice>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="derived">
    <xs:complexContent>
      <xs:restriction base="base">
        <xs:sequence><xs:element name="a" type="xs:int" minOccurs="0"/></xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`

	// derived makes a REQUIRED base element optional (min-widening): the empty
	// instance is valid against derived but invalid against base, so the derived
	// language is NOT a subset. The automaton must model `{0,1}` as optional (not
	// mandatory) or it false-accepts this. Invalid in BOTH 1.0 and 1.1.
	minWidening := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>
  <xs:complexType name="derived">
    <xs:complexContent>
      <xs:restriction base="base">
        <xs:sequence><xs:element name="a" type="xs:string" minOccurs="0"/></xs:sequence>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`

	t.Run("element restricts choice valid in 1.1", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, compileParticleVer(t, xsd.Version11, elemRestrictsChoice))
	})
	t.Run("required-to-optional min-widening rejected in 1.1", func(t *testing.T) {
		t.Parallel()
		require.ErrorIs(t, compileParticleVer(t, xsd.Version11, minWidening), xsd.ErrCompilationFailed)
	})
	t.Run("required-to-optional min-widening rejected in 1.0", func(t *testing.T) {
		t.Parallel()
		require.ErrorIs(t, compileParticleVer(t, xsd.Version10, minWidening), xsd.ErrCompilationFailed)
	})
	t.Run("element restricts choice rejected in 1.0", func(t *testing.T) {
		t.Parallel()
		require.ErrorIs(t, compileParticleVer(t, xsd.Version10, elemRestrictsChoice), xsd.ErrCompilationFailed)
	})
	t.Run("over-restriction rejected in 1.1", func(t *testing.T) {
		t.Parallel()
		require.ErrorIs(t, compileParticleVer(t, xsd.Version11, overRestriction), xsd.ErrCompilationFailed)
	})
	t.Run("type-incompatible narrowing rejected in 1.1", func(t *testing.T) {
		t.Parallel()
		require.ErrorIs(t, compileParticleVer(t, xsd.Version11, typeIncompatible), xsd.ErrCompilationFailed)
	})
}

// TestVersion11UPANestedOccurrence covers the UPA fix: occurrence-copies of ONE
// element declaration (e.g. a{1,2} nested in a repeating sequence) are the same
// particle and never a cos-nonambig violation in 1.1. The SOUNDNESS guards
// verify two DISTINCT same-named declarations still violate UPA — both as two
// local declarations and as a group referenced twice (whose particles share the
// same *ElementDecl) — so the origin-based exemption cannot mask a real
// ambiguity.
func TestVersion11UPANestedOccurrence(t *testing.T) {
	t.Parallel()

	// (a{1,2}){1,2}: nested bounded occurrence of a single element. The old
	// position automaton unrolled it into distinct same-name positions and flagged
	// it non-deterministic; it is deterministic (one particle) in 1.1.
	nestedOccurrence := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="t">
    <xs:sequence maxOccurs="2">
      <xs:element name="a" maxOccurs="2"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`

	// choice(a, a): two DISTINCT declarations named a — a genuine UPA violation.
	choiceTwoDecls := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="t">
    <xs:choice>
      <xs:element name="a" type="xs:string"/>
      <xs:element name="a" type="xs:string"/>
    </xs:choice>
  </xs:complexType>
</xs:schema>`

	// choice(ref g, ref g) where g = sequence(a): the two branches share g's
	// *ElementDecl through group-ref expansion, but they are still two competing
	// particles — a genuine UPA violation the origin exemption must not mask.
	choiceGroupTwice := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:group name="g"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:group>
  <xs:complexType name="t">
    <xs:choice>
      <xs:group ref="g"/>
      <xs:group ref="g"/>
    </xs:choice>
  </xs:complexType>
</xs:schema>`

	t.Run("nested bounded occurrence valid in 1.1", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, compileParticleVer(t, xsd.Version11, nestedOccurrence))
	})
	t.Run("two same-name declarations rejected in 1.1", func(t *testing.T) {
		t.Parallel()
		require.ErrorIs(t, compileParticleVer(t, xsd.Version11, choiceTwoDecls), xsd.ErrCompilationFailed)
	})
	t.Run("group referenced twice rejected in 1.1", func(t *testing.T) {
		t.Parallel()
		require.ErrorIs(t, compileParticleVer(t, xsd.Version11, choiceGroupTwice), xsd.ErrCompilationFailed)
	})
}
