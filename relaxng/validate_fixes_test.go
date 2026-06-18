package relaxng_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestMatchDataUnknownDatatype covers bug 1: <data> against an unknown
// built-in datatype name (or unknown library) must FAIL, not silently match
// everything.
func TestMatchDataUnknownDatatype(t *testing.T) {
	t.Parallel()

	t.Run("unknown builtin datatype name", func(t *testing.T) {
		t.Parallel()
		schema := `<element name="a" xmlns="http://relaxng.org/ns/structure/1.0">
  <data type="bogus"/>
</element>`
		err := validateWith(t, schema, `<a>anything</a>`)
		require.Error(t, err, `<data type="bogus"/> must not match`)
	})

	t.Run("unknown datatype library", func(t *testing.T) {
		t.Parallel()
		// An explicit, unrecognized datatypeLibrary on <data> must fail closed:
		// matchData returns -1 for an unknown library rather than matching
		// everything. This directly discriminates the <data> library path
		// (the bogus-name case above exercises the unknown built-in NAME path).
		schema := `<element name="a" xmlns="http://relaxng.org/ns/structure/1.0">
  <data type="string" datatypeLibrary="http://example.com/unknown-datatypes"/>
</element>`
		err := validateWith(t, schema, `<a>x</a>`)
		require.Error(t, err, `<data> with an unknown datatypeLibrary must not match`)
	})

	t.Run("builtin token matches", func(t *testing.T) {
		t.Parallel()
		schema := `<element name="a" xmlns="http://relaxng.org/ns/structure/1.0">
  <data type="token"/>
</element>`
		err := validateWith(t, schema, `<a>anything</a>`)
		require.NoError(t, err, `<data type="token"/> should match`)
	})
}

// TestMatchAttrZeroOrMore covers bug 2: a <zeroOrMore><value> inside an
// attribute must reject tokens the value doesn't match.
func TestMatchAttrZeroOrMore(t *testing.T) {
	t.Parallel()

	schema := `<element name="a" xmlns="http://relaxng.org/ns/structure/1.0">
  <attribute name="a">
    <zeroOrMore>
      <value>foo</value>
    </zeroOrMore>
  </attribute>
</element>`

	t.Run("empty matches", func(t *testing.T) {
		t.Parallel()
		err := validateWith(t, schema, `<a a=""/>`)
		require.NoError(t, err, `empty should match zeroOrMore`)
	})

	t.Run("matching tokens match", func(t *testing.T) {
		t.Parallel()
		err := validateWith(t, schema, `<a a="foo foo"/>`)
		require.NoError(t, err, `"foo foo" should match`)
	})

	t.Run("non-matching token rejected", func(t *testing.T) {
		t.Parallel()
		err := validateWith(t, schema, `<a a="bar"/>`)
		require.Error(t, err, `"bar" must be rejected`)
	})
}

// TestMatchAttrOneOrMore covers the patternOneOrMore path in matchAttrContent.
func TestMatchAttrOneOrMore(t *testing.T) {
	t.Parallel()

	schema := `<element name="a" xmlns="http://relaxng.org/ns/structure/1.0">
  <attribute name="a">
    <oneOrMore>
      <value>foo</value>
    </oneOrMore>
  </attribute>
</element>`

	t.Run("matching tokens match", func(t *testing.T) {
		t.Parallel()
		err := validateWith(t, schema, `<a a="foo foo"/>`)
		require.NoError(t, err, `"foo foo" should match`)
	})

	t.Run("non-matching token rejected", func(t *testing.T) {
		t.Parallel()
		err := validateWith(t, schema, `<a a="foo bar"/>`)
		require.Error(t, err, `"foo bar" must be rejected`)
	})
}

// TestValidateWithLengthFacets covers bug 3: minLength/maxLength facets on a
// <data type="string"> must be enforced.
func TestValidateWithLengthFacets(t *testing.T) {
	t.Parallel()

	t.Run("minLength", func(t *testing.T) {
		t.Parallel()
		schema := `<element name="a" xmlns="http://relaxng.org/ns/structure/1.0"
    datatypeLibrary="http://www.w3.org/2001/XMLSchema-datatypes">
  <data type="string">
    <param name="minLength">3</param>
  </data>
</element>`

		err := validateWith(t, schema, `<a>abc</a>`)
		require.NoError(t, err, `"abc" meets minLength 3`)

		err = validateWith(t, schema, `<a>ab</a>`)
		require.Error(t, err, `"ab" violates minLength 3`)
	})

	t.Run("maxLength", func(t *testing.T) {
		t.Parallel()
		schema := `<element name="a" xmlns="http://relaxng.org/ns/structure/1.0"
    datatypeLibrary="http://www.w3.org/2001/XMLSchema-datatypes">
  <data type="string">
    <param name="maxLength">3</param>
  </data>
</element>`

		err := validateWith(t, schema, `<a>abc</a>`)
		require.NoError(t, err, `"abc" meets maxLength 3`)

		err = validateWith(t, schema, `<a>abcd</a>`)
		require.Error(t, err, `"abcd" violates maxLength 3`)
	})
}

// TestLengthFacetsNonStringDatatype covers issue B: minLength/maxLength must be
// enforced for applicable XSD datatypes beyond xs:string (e.g. xs:token), not
// just left to lexical validation.
func TestLengthFacetsNonStringDatatype(t *testing.T) {
	t.Parallel()

	schema := `<element name="a" xmlns="http://relaxng.org/ns/structure/1.0"
    datatypeLibrary="http://www.w3.org/2001/XMLSchema-datatypes">
  <data type="token">
    <param name="maxLength">3</param>
  </data>
</element>`

	t.Run("within maxLength", func(t *testing.T) {
		t.Parallel()
		err := validateWith(t, schema, `<a>abc</a>`)
		require.NoError(t, err, `token "abc" meets maxLength 3`)
	})

	t.Run("exceeds maxLength", func(t *testing.T) {
		t.Parallel()
		err := validateWith(t, schema, `<a>abcd</a>`)
		require.Error(t, err, `token "abcd" violates maxLength 3`)
	})
}

// TestForeignDatatypeLibraryIgnored covers a namespace bug: a foreign-namespaced
// f:datatypeLibrary attribute must NOT be mistaken for the RELAX NG
// datatypeLibrary attribute. Foreign attributes are removed during
// simplification (spec §§4.1, 4.3) before datatypeLibrary inheritance is
// computed, so a foreign f:datatypeLibrary="" must not reset an inherited XSD
// library. A genuine unqualified datatypeLibrary="" still resets to the
// built-in library.
func TestForeignDatatypeLibraryIgnored(t *testing.T) {
	t.Parallel()

	const xsdLib = "http://www.w3.org/2001/XMLSchema-datatypes"

	t.Run("foreign datatypeLibrary is ignored", func(t *testing.T) {
		t.Parallel()
		// The grammar declares the XSD library. The foreign f:datatypeLibrary=""
		// on the <data> element must be ignored, so type="integer" still
		// resolves against the inherited XSD library and validates an integer.
		schema := `<element name="a" xmlns="http://relaxng.org/ns/structure/1.0"
    xmlns:f="urn:example:foreign"
    datatypeLibrary="` + xsdLib + `">
  <data type="integer" f:datatypeLibrary=""/>
</element>`

		err := validateWith(t, schema, `<a>42</a>`)
		require.NoError(t, err, `foreign f:datatypeLibrary="" must not reset the inherited XSD library; integer must validate`)

		err = validateWith(t, schema, `<a>notanint</a>`)
		require.Error(t, err, `non-integer must still be rejected under the inherited XSD library`)
	})

	t.Run("unqualified datatypeLibrary empty resets to builtin", func(t *testing.T) {
		t.Parallel()
		// A genuine unqualified datatypeLibrary="" resets to the built-in
		// library, where "integer" is not a recognized datatype, so the bare
		// integer value must be rejected.
		schema := `<element name="a" xmlns="http://relaxng.org/ns/structure/1.0"
    datatypeLibrary="` + xsdLib + `">
  <data type="integer" datatypeLibrary=""/>
</element>`

		err := validateWith(t, schema, `<a>42</a>`)
		require.Error(t, err, `unqualified datatypeLibrary="" resets to the built-in library where "integer" is unknown`)
	})
}

// TestAttrRepeatXMLWhitespaceTokenize covers issue C: the attribute-repetition
// token list must split on XML whitespace only (#x20, #x9, #xA, #xD), not all
// Unicode whitespace. An NBSP-separated value is a single token and must not be
// treated as two "foo" tokens.
func TestAttrRepeatXMLWhitespaceTokenize(t *testing.T) {
	t.Parallel()

	schema := `<element name="a" xmlns="http://relaxng.org/ns/structure/1.0">
  <attribute name="a">
    <oneOrMore>
      <value>foo</value>
    </oneOrMore>
  </attribute>
</element>`

	t.Run("space-separated is two tokens", func(t *testing.T) {
		t.Parallel()
		err := validateWith(t, schema, `<a a="foo foo"/>`)
		require.NoError(t, err, `"foo foo" is two foo tokens`)
	})

	t.Run("nbsp-separated is one non-matching token", func(t *testing.T) {
		t.Parallel()
		// U+00A0 NO-BREAK SPACE between the two words: XML whitespace
		// tokenization keeps this as a single token "foo foo", which does
		// not equal the <value>foo</value>, so validation must fail.
		err := validateWith(t, schema, "<a a=\"foo foo\"/>")
		require.Error(t, err, "NBSP-joined value is a single token, not two foo")
	})
}
