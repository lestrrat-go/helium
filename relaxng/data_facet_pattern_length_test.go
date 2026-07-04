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

// TestDataLengthFacetQNameEnforced covers RNG-104+105 r4: RELAX NG's datatype
// library treats the length facets (length, minLength, maxLength) on xs:QName and
// xs:NOTATION as CONSTRAINING (predating W3C Schema errata 4009), not a no-op — a
// value whose rune count violates the bound is REJECTED. This intentionally DIVERGES
// from the xsd validator, which treats those facets as vacuous on QName/NOTATION per
// errata 4009. An in-bounds value is accepted, the bound is still compile-validated
// as an xs:nonNegativeInteger, and string-type enforcement is unaffected.
func TestDataLengthFacetQNameEnforced(t *testing.T) {
	t.Parallel()

	const xsd = `datatypeLibrary="http://www.w3.org/2001/XMLSchema-datatypes"`

	mk := func(typ, facet, bound string) string {
		return `<element name="r" xmlns="http://relaxng.org/ns/structure/1.0" ` + xsd + `>
  <data type="` + typ + `"><param name="` + facet + `">` + bound + `</param></data>
</element>`
	}

	t.Run("maxLength constrains a QName", func(t *testing.T) {
		t.Parallel()
		// "abc" is a 3-rune lexically valid QName; maxLength="1" rejects it, the same
		// way the xsd validator rejects an out-of-space QName length facet.
		require.Error(t, validateWith(t, mk("QName", "maxLength", "1"), `<r>abc</r>`),
			`maxLength on xs:QName must reject a longer value (XSD 1.0 parity)`)
	})

	t.Run("maxLength accepts an in-bounds QName", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, validateWith(t, mk("QName", "maxLength", "3"), `<r>abc</r>`),
			`maxLength on xs:QName must accept an in-bounds value`)
	})

	t.Run("minLength constrains a QName", func(t *testing.T) {
		t.Parallel()
		require.Error(t, validateWith(t, mk("QName", "minLength", "100"), `<r>abc</r>`),
			`minLength on xs:QName must reject a too-short value`)
	})

	t.Run("length constrains a QName", func(t *testing.T) {
		t.Parallel()
		require.Error(t, validateWith(t, mk("QName", "length", "1"), `<r>abc</r>`),
			`length on xs:QName must reject a value of a different length`)
	})

	t.Run("maxLength constrains a NOTATION", func(t *testing.T) {
		t.Parallel()
		require.Error(t, validateWith(t, mk("NOTATION", "maxLength", "1"), `<r>abc</r>`),
			`maxLength on xs:NOTATION must reject a longer value`)
	})

	t.Run("length facet on QName still compiles", func(t *testing.T) {
		t.Parallel()
		require.Empty(t, compileErrorsFor(t, mk("QName", "length", "1")),
			"length facet on xs:QName is applicable and must compile")
	})

	t.Run("invalid bound on QName is still a compile error", func(t *testing.T) {
		t.Parallel()
		// The bound itself must remain a valid xs:nonNegativeInteger (the r2 bound
		// check is kept).
		require.NotEmpty(t, compileErrorsFor(t, mk("QName", "maxLength", "-1")),
			`maxLength="-1" on xs:QName must still be a compile error`)
	})
}

// TestDataLengthFacetBoundValidity covers RNG-104+105 r2: a length/minLength/
// maxLength bound must be a valid xs:nonNegativeInteger, validated at COMPILE
// time with XSD lexical/whitespace rules (not Go's strconv.Atoi + TrimSpace).
// A negative bound, an NBSP-padded bound, or any non-integer bound is a compile
// error; and a huge-but-valid bound must NOT overflow int into a reject-all.
func TestDataLengthFacetBoundValidity(t *testing.T) {
	t.Parallel()

	const xsd = `datatypeLibrary="http://www.w3.org/2001/XMLSchema-datatypes"`

	mk := func(facet, bound string) string {
		return `<element name="r" xmlns="http://relaxng.org/ns/structure/1.0" ` + xsd + `>
  <data type="string"><param name="` + facet + `">` + bound + `</param></data>
</element>`
	}

	t.Run("negative minLength is a compile error", func(t *testing.T) {
		t.Parallel()
		// Without bound validation, "-1" compiled and then accepted EVERY value
		// (length >= -1 always holds), silently disabling the facet.
		require.NotEmpty(t, compileErrorsFor(t, mk("minLength", "-1")),
			`minLength="-1" must be a compile error, not an accept-all facet`)
	})

	t.Run("negative length is a compile error", func(t *testing.T) {
		t.Parallel()
		require.NotEmpty(t, compileErrorsFor(t, mk("length", "-2")),
			`length="-2" must be a compile error`)
	})

	t.Run("negative maxLength is a compile error", func(t *testing.T) {
		t.Parallel()
		require.NotEmpty(t, compileErrorsFor(t, mk("maxLength", "-1")),
			`maxLength="-1" must be a compile error`)
	})

	t.Run("NBSP-padded bound is a compile error", func(t *testing.T) {
		t.Parallel()
		// U+00A0 (NBSP) is NOT XSD whitespace, so it is not collapsed away: the
		// bound is not a valid xs:nonNegativeInteger. Go's TrimSpace would strip
		// it and Atoi would accept the digits, which is wrong XSD handling.
		require.NotEmpty(t, compileErrorsFor(t, mk("minLength", " 3")),
			"NBSP-padded bound must be a compile error")
	})

	t.Run("fractional bound is a compile error", func(t *testing.T) {
		t.Parallel()
		require.NotEmpty(t, compileErrorsFor(t, mk("maxLength", "3.0")),
			`maxLength="3.0" must be a compile error`)
	})

	t.Run("non-digit bound is a compile error", func(t *testing.T) {
		t.Parallel()
		require.NotEmpty(t, compileErrorsFor(t, mk("length", "abc")),
			`length="abc" must be a compile error`)
	})

	t.Run("XSD-whitespace-padded bound compiles", func(t *testing.T) {
		t.Parallel()
		// Leading/trailing XSD whitespace (here a newline + spaces) IS collapsed,
		// so the bound is the valid xs:nonNegativeInteger "3".
		require.Empty(t, compileErrorsFor(t, mk("length", "\n  3  ")),
			"XSD-whitespace-padded bound must compile")
	})

	t.Run("huge maxLength does not reject valid short values", func(t *testing.T) {
		t.Parallel()
		// A bound far beyond int range must compare width-safely: parsing it with
		// strconv.Atoi overflows and (depending on platform) flips the facet into
		// a reject-all. A 3-char value must still satisfy this enormous maxLength.
		huge := "99999999999999999999999999999999999999"
		require.NoError(t, validateWith(t, mk("maxLength", huge), `<r>abc</r>`),
			"a huge but valid maxLength must accept a short value, not reject-all")
	})

	t.Run("huge minLength still rejects short values", func(t *testing.T) {
		t.Parallel()
		huge := "99999999999999999999999999999999999999"
		require.Error(t, validateWith(t, mk("minLength", huge), `<r>abc</r>`),
			"a huge minLength must reject a short value")
	})

	t.Run("normal bound still enforces length", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, validateWith(t, mk("length", "3"), `<r>abc</r>`),
			`length="3" must accept a 3-char value`)
		require.Error(t, validateWith(t, mk("length", "3"), `<r>ab</r>`),
			`length="3" must reject a 2-char value`)
		require.NoError(t, validateWith(t, mk("minLength", "2"), `<r>abc</r>`),
			`minLength="2" must accept a 3-char value`)
		require.Error(t, validateWith(t, mk("maxLength", "2"), `<r>abc</r>`),
			`maxLength="2" must reject a 3-char value`)
	})
}
