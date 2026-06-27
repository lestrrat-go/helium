package relaxng_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestDataPatternFacetXSDRegex covers RNG-104: a <data> pattern facet must be
// evaluated with the XSD/XPath regular-expression engine (internal/xsdregex),
// not Go's regexp package, so XSD-only constructs such as the \i (XML
// name-start) and \c (XML name) character-class shorthands are honoured rather
// than false-rejected.
func TestDataPatternFacetXSDRegex(t *testing.T) {
	t.Parallel()

	const xsd = `datatypeLibrary="http://www.w3.org/2001/XMLSchema-datatypes"`

	schema := `<element name="r" xmlns="http://relaxng.org/ns/structure/1.0" ` + xsd + `>
  <data type="string"><param name="pattern">\i\c*</param></data>
</element>`

	t.Run("accepts a valid XML name", func(t *testing.T) {
		t.Parallel()
		// "_foo-bar" is a valid XML Name (\i name-start, then \c* name chars). Go's
		// regexp engine errors on the \i/\c escapes and so wrongly rejected this.
		require.NoError(t, validateWith(t, schema, `<r>_foo-bar</r>`),
			`\i\c* must accept a valid XML name`)
	})

	t.Run("rejects a non-name value", func(t *testing.T) {
		t.Parallel()
		// "1abc" starts with a digit, which is not an XML name-start character, so
		// it must not match \i\c*.
		require.Error(t, validateWith(t, schema, `<r>1abc</r>`),
			`\i\c* must reject a value whose first char is not a name-start char`)
	})

	t.Run("unicode property class", func(t *testing.T) {
		t.Parallel()
		// \p{Lu} (an uppercase-letter category) is another XSD-regex construct Go's
		// engine does not accept.
		s := `<element name="r" xmlns="http://relaxng.org/ns/structure/1.0" ` + xsd + `>
  <data type="string"><param name="pattern">\p{Lu}+</param></data>
</element>`
		require.NoError(t, validateWith(t, s, `<r>ABC</r>`), `\p{Lu}+ must accept "ABC"`)
		require.Error(t, validateWith(t, s, `<r>abc</r>`), `\p{Lu}+ must reject "abc"`)
	})
}

// TestDataLengthFacetApplicability covers RNG-105: the length facets (length,
// minLength, maxLength) are applicable only to string-derived, binary, anyURI,
// QName and NOTATION datatypes. Applying one to a numeric/boolean/date datatype
// is a schema error and must be reported at compile time, matching the XSD
// facet-applicability rules.
func TestDataLengthFacetApplicability(t *testing.T) {
	t.Parallel()

	const xsd = `datatypeLibrary="http://www.w3.org/2001/XMLSchema-datatypes"`

	mk := func(typ, facet string) string {
		return `<element name="r" xmlns="http://relaxng.org/ns/structure/1.0" ` + xsd + `>
  <data type="` + typ + `"><param name="` + facet + `">3</param></data>
</element>`
	}

	t.Run("length on integer is a compile error", func(t *testing.T) {
		t.Parallel()
		require.NotEmpty(t, compileErrorsFor(t, mk("integer", "length")),
			"length facet on xs:integer must be a compile error")
	})

	t.Run("minLength on boolean is a compile error", func(t *testing.T) {
		t.Parallel()
		require.NotEmpty(t, compileErrorsFor(t, mk("boolean", "minLength")),
			"minLength facet on xs:boolean must be a compile error")
	})

	t.Run("maxLength on date is a compile error", func(t *testing.T) {
		t.Parallel()
		require.NotEmpty(t, compileErrorsFor(t, mk("date", "maxLength")),
			"maxLength facet on xs:date must be a compile error")
	})

	t.Run("length on string is allowed", func(t *testing.T) {
		t.Parallel()
		require.Empty(t, compileErrorsFor(t, mk("string", "length")),
			"length facet on xs:string must compile")
	})

	t.Run("length on hexBinary is allowed", func(t *testing.T) {
		t.Parallel()
		require.Empty(t, compileErrorsFor(t, mk("hexBinary", "length")),
			"length facet on xs:hexBinary must compile")
	})
}
