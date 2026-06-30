package xsd_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestOpenContent_ChoiceEmptiablePrune covers the gauntlet finding that the
// open-content non-emitting prune (pruneNonEmittingParticles) must be
// semantics-preserving for xs:choice emptiability. An xs:choice with an EMPTY
// branch — intrinsically empty (<xs:sequence/>) or emptied by pruning a
// maxOccurs=0 member — is EMPTIABLE: the empty branch lets the choice match the
// empty string, so a sibling element in the choice stays OPTIONAL. The prune used
// to drop the empty branch outright, turning choice(<empty>, g) into choice(g) and
// falsely REQUIRING g. The fix keeps an emptiable empty-sequence branch in a CHOICE
// while still dropping the no-op empty member from a SEQUENCE (a required sibling in
// a sequence stays required).
func TestOpenContent_ChoiceEmptiablePrune(t *testing.T) {
	t.Parallel()

	t.Run("intrinsically-empty sequence branch keeps the choice emptiable", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc"><xs:complexType>
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="skip"/></xs:openContent>
    <xs:choice>
      <xs:sequence/>
      <xs:element name="g" type="xs:string"/>
    </xs:choice>
  </xs:complexType></xs:element>
</xs:schema>`
		require.NoError(t, validateOC(t, schema, `<doc/>`),
			"the empty <xs:sequence/> branch makes the choice emptiable, so g is optional")
		require.NoError(t, validateOC(t, schema, `<doc><g>x</g></doc>`),
			"the emitting branch g still matches")
		require.NoError(t, validateOC(t, schema, `<doc><z>y</z></doc>`),
			"the choice matches empty and z routes to open content")
	})

	t.Run("branch emptied by a maxOccurs=0 member keeps the choice emptiable", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc"><xs:complexType>
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="skip"/></xs:openContent>
    <xs:choice>
      <xs:sequence>
        <xs:element name="e" type="xs:int" minOccurs="0" maxOccurs="0"/>
      </xs:sequence>
      <xs:element name="g" type="xs:string"/>
    </xs:choice>
  </xs:complexType></xs:element>
</xs:schema>`
		require.NoError(t, validateOC(t, schema, `<doc/>`),
			"the branch whose only member is prohibited prunes to empty and keeps the choice emptiable")
		require.NoError(t, validateOC(t, schema, `<doc><g>x</g></doc>`),
			"the emitting branch g still matches")
	})

	t.Run("empty group in a SEQUENCE is a no-op; a required sibling stays required", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc"><xs:complexType>
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="skip"/></xs:openContent>
    <xs:sequence>
      <xs:sequence/>
      <xs:element name="r" type="xs:string"/>
    </xs:sequence>
  </xs:complexType></xs:element>
</xs:schema>`
		require.Error(t, validateOC(t, schema, `<doc/>`),
			"the empty group is a no-op in a sequence; r is still required")
		require.NoError(t, validateOC(t, schema, `<doc><r>x</r></doc>`),
			"r present satisfies the sequence")
	})

	t.Run("a direct prohibited element leaf does NOT make a choice emptiable (round-16)", func(t *testing.T) {
		t.Parallel()
		// Distinct from the group cases above: a prohibited ELEMENT leaf is dropped
		// outright (it is not a group that matches the empty string), so the sibling
		// emitting branch stays required — the established round-16 behavior.
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="doc"><xs:complexType>
    <xs:openContent mode="interleave"><xs:any namespace="##local" processContents="skip"/></xs:openContent>
    <xs:choice>
      <xs:element name="e" type="xs:int" minOccurs="0" maxOccurs="0"/>
      <xs:element name="g" type="xs:string"/>
    </xs:choice>
  </xs:complexType></xs:element>
</xs:schema>`
		require.Error(t, validateOC(t, schema, `<doc><e>anything</e></doc>`),
			"the prohibited leaf e does not make the choice emptiable; g is still required")
	})
}

// TestComplexContent_StrayChildRejected covers the gauntlet finding that
// <xs:complexContent> has a (annotation?, (restriction | extension)) content model:
// any other child — notably a stray <xs:openContent>, which belongs INSIDE the
// restriction/extension wrapper — must be a schema error rather than silently
// ignored.
func TestComplexContent_StrayChildRejected(t *testing.T) {
	t.Parallel()

	t.Run("openContent BEFORE the derivation under complexContent is a grammar error", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="B"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>
  <xs:complexType name="R">
    <xs:complexContent>
      <xs:openContent mode="suffix"><xs:any namespace="##local" processContents="skip"/></xs:openContent>
      <xs:restriction base="B"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.Error(t, cerr, "a stray xs:openContent before the derivation under xs:complexContent must be rejected")
	})

	t.Run("openContent AFTER the derivation under complexContent is a grammar error", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="B"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>
  <xs:complexType name="R">
    <xs:complexContent>
      <xs:restriction base="B"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:restriction>
      <xs:openContent mode="suffix"><xs:any namespace="##local" processContents="skip"/></xs:openContent>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.Error(t, cerr, "a stray xs:openContent AFTER the derivation must be rejected (no early return)")
	})

	t.Run("two derivation children under complexContent is a grammar error", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="B"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>
  <xs:complexType name="R">
    <xs:complexContent>
      <xs:extension base="B"><xs:sequence><xs:element name="b" type="xs:string"/></xs:sequence></xs:extension>
      <xs:extension base="B"><xs:sequence><xs:element name="c" type="xs:string"/></xs:sequence></xs:extension>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.Error(t, cerr, "exactly one restriction/extension is allowed under complexContent")
	})

	t.Run("stray element after the derivation under complexContent is a grammar error", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="B"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>
  <xs:complexType name="R">
    <xs:complexContent>
      <xs:restriction base="B"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:restriction>
      <xs:sequence><xs:element name="z" type="xs:string"/></xs:sequence>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.Error(t, cerr, "a stray particle after the derivation must be rejected")
	})

	t.Run("annotation, restriction and extension remain valid under complexContent", func(t *testing.T) {
		t.Parallel()
		const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="B"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>
  <xs:complexType name="R">
    <xs:complexContent>
      <xs:annotation><xs:documentation>ok</xs:documentation></xs:annotation>
      <xs:restriction base="B"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:restriction>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.NoError(t, cerr, "annotation before restriction is valid under complexContent")
	})
}
