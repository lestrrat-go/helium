package xsd_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestOpenContent_RestrictionEmptyModelStillChecked covers the gauntlet finding
// that an EMPTY derived content model must not waive the open-content restriction
// validity checks (§3.4.6.4). Only the declared-model MODE comparison is
// immaterial for an empty model; the base!=nil, wildcard-subset, and
// processContents-strength checks still apply, so a restriction may NOT add a
// broader or weaker open content than the base allows.
func TestOpenContent_RestrictionEmptyModelStillChecked(t *testing.T) {
	t.Parallel()
	const head = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">`

	t.Run("empty model restriction may not ADD open content to a closed base", func(t *testing.T) {
		t.Parallel()
		// Base B is CLOSED (no open content). R restricts B to an empty content
		// model (valid: drops the optional element) but ADDS open content. The
		// add must be rejected even though the derived model is empty.
		schema := head + `
  <xs:complexType name="B"><xs:sequence>
    <xs:element name="a" minOccurs="0"/>
  </xs:sequence></xs:complexType>
  <xs:complexType name="R"><xs:complexContent><xs:restriction base="B">
    <xs:openContent mode="suffix"><xs:any namespace="http://open.com/" processContents="lax"/></xs:openContent>
    <xs:sequence/>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.Error(t, cerr, "adding open content to a closed base via an empty-model restriction must be rejected")
	})

	t.Run("empty model restriction may not BROADEN the base open content wildcard", func(t *testing.T) {
		t.Parallel()
		// Base open content wildcard is restricted to namespace http://a.com/ with
		// strict processing. The empty-model restriction widens it to ##any with
		// skip — both a non-subset wildcard and a weaker processContents — which
		// must be rejected even though the derived content model is empty.
		schema := head + `
  <xs:complexType name="B" mixed="true">
    <xs:openContent mode="interleave"><xs:any namespace="http://a.com/" processContents="strict"/></xs:openContent>
    <xs:sequence><xs:element name="a" minOccurs="0" maxOccurs="unbounded"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="R"><xs:complexContent mixed="true"><xs:restriction base="B">
    <xs:openContent mode="interleave"><xs:any namespace="##any" processContents="skip"/></xs:openContent>
    <xs:sequence/>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.Error(t, cerr, "broadening the base open content wildcard via an empty-model restriction must be rejected")
	})

	t.Run("empty model restriction keeping a subset open content stays valid", func(t *testing.T) {
		t.Parallel()
		// Regression guard: an empty-model restriction that narrows the mode
		// (suffix base -> interleave) but keeps the SAME wildcard is still valid;
		// the mode comparison is correctly waived for an empty model.
		schema := head + `
  <xs:complexType name="B" mixed="true">
    <xs:openContent mode="suffix"><xs:any namespace="http://open.com/" processContents="lax"/></xs:openContent>
    <xs:sequence><xs:element name="a" minOccurs="0" maxOccurs="unbounded"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="R"><xs:complexContent mixed="true"><xs:restriction base="B">
    <xs:openContent mode="interleave"><xs:any namespace="http://open.com/" processContents="lax"/></xs:openContent>
    <xs:sequence/>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.NoError(t, cerr, "subset open content with an empty model must stay valid")
	})
}

// TestOpenContent_InterleaveRefinementKeepsTrying covers the gauntlet finding
// that interleave partition refinement must not bail on the FIRST trial-match
// error. With declared sequence(a,b) and an interleave wildcard, the second <a>
// is open content and the declared subsequence a,b is valid; refinement must move
// the blocking <a> into the open partition and keep trying.
func TestOpenContent_InterleaveRefinementKeepsTrying(t *testing.T) {
	t.Parallel()
	const head = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">`
	schema := head + `
  <xs:element name="doc"><xs:complexType>
    <xs:openContent mode="interleave"><xs:any processContents="skip"/></xs:openContent>
    <xs:sequence><xs:element name="a"/><xs:element name="b"/></xs:sequence>
  </xs:complexType></xs:element>
</xs:schema>`
	require.NoError(t, validateOC(t, schema, `<doc><a/><a/><b/></doc>`),
		"the second <a> is open content; the declared a,b subsequence is valid")
	// A genuinely missing required element must still be rejected.
	require.Error(t, validateOC(t, schema, `<doc><a/></doc>`),
		"a missing required <b> must still be rejected")
}

// TestOpenContent_MultipleWildcardChildren covers the gauntlet finding that both
// <xs:openContent> and <xs:defaultOpenContent> must REJECT more than one <xs:any>
// wildcard child rather than silently using the first.
func TestOpenContent_MultipleWildcardChildren(t *testing.T) {
	t.Parallel()
	const head = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">`

	t.Run("openContent with two wildcards", func(t *testing.T) {
		t.Parallel()
		schema := head + `
  <xs:element name="doc"><xs:complexType>
    <xs:openContent mode="suffix">
      <xs:any namespace="http://a.com/" processContents="lax"/>
      <xs:any namespace="http://b.com/" processContents="lax"/>
    </xs:openContent>
    <xs:sequence><xs:element name="a"/></xs:sequence>
  </xs:complexType></xs:element>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.Error(t, cerr, "openContent with two any wildcards must be rejected")
	})

	t.Run("defaultOpenContent with two wildcards", func(t *testing.T) {
		t.Parallel()
		schema := head + `
  <xs:defaultOpenContent mode="suffix">
    <xs:any namespace="http://a.com/" processContents="lax"/>
    <xs:any namespace="http://b.com/" processContents="lax"/>
  </xs:defaultOpenContent>
  <xs:element name="doc"><xs:complexType><xs:sequence><xs:element name="a"/></xs:sequence></xs:complexType></xs:element>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.Error(t, cerr, "defaultOpenContent with two any wildcards must be rejected")
	})
}
