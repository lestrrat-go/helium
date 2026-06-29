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

	t.Run("empty model restriction may not convert a BOUNDED base wildcard to open content", func(t *testing.T) {
		t.Parallel()
		// Base has a BOUNDED content-model wildcard (maxOccurs=1). An interleave
		// open-content wildcard is effectively unbounded, so it admits a SECOND open
		// child the base would reject — the restriction is not a language subset.
		schema := head + `
  <xs:complexType name="B"><xs:sequence>
    <xs:any namespace="http://open.com/" processContents="lax" minOccurs="0" maxOccurs="1"/>
  </xs:sequence></xs:complexType>
  <xs:complexType name="R"><xs:complexContent><xs:restriction base="B">
    <xs:openContent mode="interleave"><xs:any namespace="http://open.com/" processContents="lax"/></xs:openContent>
    <xs:sequence/>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.Error(t, cerr, "converting a bounded base wildcard to unbounded open content must be rejected")
	})

	t.Run("empty model restriction may convert an UNBOUNDED base wildcard to open content (open022 shape)", func(t *testing.T) {
		t.Parallel()
		// Anchor for the correct ACCEPT: the base's content-model wildcard is
		// effectively unbounded, so re-expressing it as open content is valid.
		schema := head + `
  <xs:complexType name="B"><xs:sequence>
    <xs:any namespace="http://open.com/" processContents="lax" minOccurs="0" maxOccurs="unbounded"/>
  </xs:sequence></xs:complexType>
  <xs:complexType name="R"><xs:complexContent><xs:restriction base="B">
    <xs:openContent mode="interleave"><xs:any namespace="http://open.com/" processContents="lax"/></xs:openContent>
    <xs:sequence/>
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="R"/>
</xs:schema>`
		_, _, cerr := compileV11(t, schema)
		require.NoError(t, cerr, "converting an unbounded base wildcard to open content is valid (open022)")
	})
}

// TestOpenContent_ChildOrder covers the gauntlet finding that <xs:openContent>
// must participate in the complex-type child-order checks (XSD §3.4.2): it must
// precede the content-model particle, the attribute uses, and the anyAttribute
// wildcard, in the direct complexType branch as well as in complexContent
// restriction and extension.
func TestOpenContent_ChildOrder(t *testing.T) {
	t.Parallel()
	const head = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">`
	const oc = `<xs:openContent mode="suffix"><xs:any namespace="http://open.com/" processContents="lax"/></xs:openContent>`

	cases := map[string]string{
		"after attribute (direct)": head + `
  <xs:element name="doc"><xs:complexType>
    <xs:sequence><xs:element name="a"/></xs:sequence>
    <xs:attribute name="x" type="xs:string"/>
    ` + oc + `
  </xs:complexType></xs:element></xs:schema>`,
		"after anyAttribute (direct)": head + `
  <xs:element name="doc"><xs:complexType>
    <xs:sequence><xs:element name="a"/></xs:sequence>
    <xs:anyAttribute processContents="lax"/>
    ` + oc + `
  </xs:complexType></xs:element></xs:schema>`,
		"after content model (direct)": head + `
  <xs:element name="doc"><xs:complexType>
    <xs:sequence><xs:element name="a"/></xs:sequence>
    ` + oc + `
  </xs:complexType></xs:element></xs:schema>`,
		"after attribute (restriction)": head + `
  <xs:complexType name="B"><xs:sequence><xs:element name="a"/></xs:sequence></xs:complexType>
  <xs:complexType name="R"><xs:complexContent><xs:restriction base="B">
    <xs:sequence><xs:element name="a"/></xs:sequence>
    <xs:attribute name="x" type="xs:string"/>
    ` + oc + `
  </xs:restriction></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="R"/></xs:schema>`,
		"after attribute (extension)": head + `
  <xs:complexType name="B"><xs:sequence><xs:element name="a"/></xs:sequence></xs:complexType>
  <xs:complexType name="R"><xs:complexContent><xs:extension base="B">
    <xs:sequence><xs:element name="d" minOccurs="0"/></xs:sequence>
    <xs:attribute name="x" type="xs:string"/>
    ` + oc + `
  </xs:extension></xs:complexContent></xs:complexType>
  <xs:element name="doc" type="R"/></xs:schema>`,
	}
	for name, schema := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, _, cerr := compileV11(t, schema)
			require.Error(t, cerr, "out-of-order openContent must be rejected")
		})
	}
}

// TestDefaultOpenContent_CompositionOrder covers the gauntlet finding that a
// composition element (include/import/redefine/override) must precede the
// schema-level <xs:defaultOpenContent>; one appearing AFTER it is out of order.
func TestDefaultOpenContent_CompositionOrder(t *testing.T) {
	t.Parallel()
	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:defaultOpenContent><xs:any namespace="http://open.com/" processContents="lax"/></xs:defaultOpenContent>
  <xs:include schemaLocation="nonexistent.xsd"/>
  <xs:element name="doc"><xs:complexType><xs:sequence><xs:element name="a"/></xs:sequence></xs:complexType></xs:element>
</xs:schema>`
	_, _, cerr := compileV11(t, schema)
	require.Error(t, cerr, "xs:include after defaultOpenContent must be rejected")
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
