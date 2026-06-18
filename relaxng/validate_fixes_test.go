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
