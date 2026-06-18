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
