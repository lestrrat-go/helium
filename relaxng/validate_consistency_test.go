package relaxng_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

const nbsp = " " // U+00A0 NO-BREAK SPACE: not XML whitespace.

// TestListNBSPTokenize covers item 1 for <list>: list tokenization must split
// on XML whitespace only (#x20, #x9, #xA, #xD), not arbitrary Unicode
// whitespace. An NBSP-joined value stays a single token.
func TestListNBSPTokenize(t *testing.T) {
	t.Parallel()

	schema := `<element name="a" xmlns="http://relaxng.org/ns/structure/1.0">
  <list>
    <oneOrMore><value>foo</value></oneOrMore>
  </list>
</element>`

	t.Run("space-separated is two tokens", func(t *testing.T) {
		t.Parallel()
		err := validateWith(t, schema, `<a>foo foo</a>`)
		require.NoError(t, err, `"foo foo" is two foo tokens`)
	})

	t.Run("nbsp-separated is one non-matching token", func(t *testing.T) {
		t.Parallel()
		err := validateWith(t, schema, "<a>foo"+nbsp+"foo</a>")
		require.Error(t, err, "NBSP-joined value is a single token, not two foo")
	})
}

// TestAttrGroupNBSPTokenize covers item 1 for the attribute <group> path: the
// patternGroup token list in matchAttrContent must split on XML whitespace
// only. An NBSP-joined value is a single token.
func TestAttrGroupNBSPTokenize(t *testing.T) {
	t.Parallel()

	schema := `<element name="a" xmlns="http://relaxng.org/ns/structure/1.0">
  <attribute name="v">
    <group>
      <value>foo</value>
      <value>bar</value>
    </group>
  </attribute>
</element>`

	t.Run("space-separated is two tokens", func(t *testing.T) {
		t.Parallel()
		err := validateWith(t, schema, `<a v="foo bar"/>`)
		require.NoError(t, err, `"foo bar" is two tokens matching the group`)
	})

	t.Run("nbsp-separated is one non-matching token", func(t *testing.T) {
		t.Parallel()
		err := validateWith(t, schema, "<a v=\"foo"+nbsp+"bar\"/>")
		require.Error(t, err, "NBSP-joined value is a single token, not foo+bar")
	})
}

// TestValueTokenNBSP covers item 1 for <value type="token">: the token
// whiteSpace=collapse normalization must collapse XML whitespace only, leaving
// NBSP intact so it does not collapse to a single space.
func TestValueTokenNBSP(t *testing.T) {
	t.Parallel()

	schema := `<element name="a" xmlns="http://relaxng.org/ns/structure/1.0">
  <value type="token">foo bar</value>
</element>`

	t.Run("collapsible xml whitespace matches", func(t *testing.T) {
		t.Parallel()
		// Tabs/newlines collapse to single spaces under whiteSpace=collapse.
		err := validateWith(t, schema, "<a>foo\t\tbar</a>")
		require.NoError(t, err, `collapsed "foo\t\tbar" equals "foo bar"`)
	})

	t.Run("nbsp does not collapse", func(t *testing.T) {
		t.Parallel()
		// NBSP is not XML whitespace, so "foo bar" stays distinct from the
		// expected "foo bar".
		err := validateWith(t, schema, "<a>foo"+nbsp+"bar</a>")
		require.Error(t, err, "NBSP value must not collapse to match \"foo bar\"")
	})
}

// tokenCollapseCase is one instance value checked by TestValueTokenCollapse.
// value is spliced into the instance as-is, so it may carry character
// references (&#9; &#10; &#13;) that survive attribute-value normalization.
type tokenCollapseCase struct {
	name  string
	value string
	match bool
}

// tokenCollapseCases pairs instance values with whether they collapse to
// "foo bar". Already-collapsed values and values needing a collapse (leading,
// trailing, and internal runs of spaces, tabs, newlines, and carriage returns)
// must give the same verdict as their collapsed form.
var tokenCollapseCases = []tokenCollapseCase{
	{"already collapsed", "foo bar", true},
	{"leading spaces", "  foo bar", true},
	{"trailing spaces", "foo bar  ", true},
	{"internal space run", "foo   bar", true},
	{"internal tab", "foo&#9;bar", true},
	{"internal newline", "foo&#10;bar", true},
	{"internal carriage return", "foo&#13;bar", true},
	{"mixed whitespace everywhere", "&#10;&#9; foo &#13;&#10;&#9; bar &#9;&#10;", true},
	{"joined words", "foobar", false},
	{"different word", "foo baz", false},
	{"extra word", "foo bar baz", false},
	{"nbsp separator", "foo" + nbsp + "bar", false},
	{"empty", "", false},
	{"whitespace only", " &#9;&#10; ", false},
}

// TestValueTokenCollapse checks <value type="token"> equality under the
// built-in library's whiteSpace=collapse, on both sides: the instance value
// and the schema literal are each collapsed before comparing, whether or not
// they are already in collapsed form. Each case runs against element
// content, an attribute value, and a <data type="token"> whose <except>
// holds the same <value> (so a collapsing match is rejected there).
func TestValueTokenCollapse(t *testing.T) {
	t.Parallel()

	for _, literal := range []string{"foo bar", " foo \t\n bar\n"} {
		elemSchema := `<element name="a" xmlns="http://relaxng.org/ns/structure/1.0">
  <value type="token">` + literal + `</value>
</element>`
		attrSchema := `<element name="a" xmlns="http://relaxng.org/ns/structure/1.0">
  <attribute name="v"><value type="token">` + literal + `</value></attribute>
</element>`
		exceptSchema := `<element name="a" xmlns="http://relaxng.org/ns/structure/1.0">
  <data type="token"><except><value type="token">` + literal + `</value></except></data>
</element>`

		for _, tc := range tokenCollapseCases {
			t.Run(fmt.Sprintf("literal %q/%s", literal, tc.name), func(t *testing.T) {
				t.Parallel()

				err := validateWith(t, elemSchema, `<a>`+tc.value+`</a>`)
				if tc.match {
					require.NoError(t, err, "element content %q should equal token %q", tc.value, literal)
				} else {
					require.Error(t, err, "element content %q should not equal token %q", tc.value, literal)
				}

				err = validateWith(t, attrSchema, `<a v="`+tc.value+`"/>`)
				if tc.match {
					require.NoError(t, err, "attribute value %q should equal token %q", tc.value, literal)
				} else {
					require.Error(t, err, "attribute value %q should not equal token %q", tc.value, literal)
				}

				err = validateWith(t, exceptSchema, `<a>`+tc.value+`</a>`)
				if tc.match {
					require.Error(t, err, "except should reject %q as equal to token %q", tc.value, literal)
				} else {
					require.NoError(t, err, "except should not reject %q", tc.value)
				}
			})
		}
	}

	t.Run("empty literal matches whitespace", func(t *testing.T) {
		t.Parallel()
		schema := `<element name="a" xmlns="http://relaxng.org/ns/structure/1.0">
  <value type="token"> </value>
</element>`
		require.NoError(t, validateWith(t, schema, "<a>&#9;&#10; </a>"),
			"whitespace-only content collapses to the empty token")
		require.NoError(t, validateWith(t, schema, "<a></a>"), "empty content equals the empty token")
		require.Error(t, validateWith(t, schema, "<a>x</a>"), "non-empty content must not equal the empty token")
	})
}

// TestValueUnknownDatatype covers item 2: a <value> with an unknown datatype
// name (empty library) or an unknown datatypeLibrary must FAIL, and must never
// match by raw string equality.
func TestValueUnknownDatatype(t *testing.T) {
	t.Parallel()

	t.Run("unknown bare type name fails", func(t *testing.T) {
		t.Parallel()
		schema := `<element name="a" xmlns="http://relaxng.org/ns/structure/1.0">
  <value type="bogus">x</value>
</element>`
		err := validateWith(t, schema, `<a>x</a>`)
		require.Error(t, err, `<value type="bogus"> must not match even on identical text`)
	})

	t.Run("unknown datatypeLibrary fails", func(t *testing.T) {
		t.Parallel()
		schema := `<element name="a" xmlns="http://relaxng.org/ns/structure/1.0"
    datatypeLibrary="http://example.com/unknown-datatypes">
  <value type="string">x</value>
</element>`
		err := validateWith(t, schema, `<a>x</a>`)
		require.Error(t, err, "unknown datatypeLibrary must not match by raw equality")
	})

	t.Run("recognized bare XSD type still matches via value space", func(t *testing.T) {
		t.Parallel()
		// Mirrors matchData's documented libxml2-compat fallback: a bare
		// recognized XSD type name (no datatypeLibrary) routes through the XSD
		// value path, so "5" and "+5" are value-equal for integer.
		schema := `<element name="a" xmlns="http://relaxng.org/ns/structure/1.0">
  <value type="integer">5</value>
</element>`
		err := validateWith(t, schema, `<a>+5</a>`)
		require.NoError(t, err, `integer value "+5" equals "5"`)
	})
}

// TestExactLengthFacet covers item 3: the exact <param name="length"> facet
// must be enforced (it was previously ignored).
func TestExactLengthFacet(t *testing.T) {
	t.Parallel()

	schema := `<element name="a" xmlns="http://relaxng.org/ns/structure/1.0"
    datatypeLibrary="http://www.w3.org/2001/XMLSchema-datatypes">
  <data type="string">
    <param name="length">3</param>
  </data>
</element>`

	t.Run("exact length matches", func(t *testing.T) {
		t.Parallel()
		err := validateWith(t, schema, `<a>abc</a>`)
		require.NoError(t, err, `"abc" has length 3`)
	})

	t.Run("shorter rejected", func(t *testing.T) {
		t.Parallel()
		err := validateWith(t, schema, `<a>ab</a>`)
		require.Error(t, err, `"ab" violates length 3`)
	})

	t.Run("longer rejected", func(t *testing.T) {
		t.Parallel()
		err := validateWith(t, schema, `<a>abcd</a>`)
		require.Error(t, err, `"abcd" violates length 3`)
	})
}

// TestLengthFacetTokenCount covers item 3: for XSD list builtins (NMTOKENS,
// IDREFS, ENTITIES) the length facet counts XML-whitespace tokens, not
// characters.
func TestLengthFacetTokenCount(t *testing.T) {
	t.Parallel()

	schema := `<element name="a" xmlns="http://relaxng.org/ns/structure/1.0"
    datatypeLibrary="http://www.w3.org/2001/XMLSchema-datatypes">
  <data type="NMTOKENS">
    <param name="length">2</param>
  </data>
</element>`

	t.Run("two tokens match length 2", func(t *testing.T) {
		t.Parallel()
		// "ab cd" is 5 characters but 2 tokens; length is token COUNT.
		err := validateWith(t, schema, `<a>ab cd</a>`)
		require.NoError(t, err, `"ab cd" has token-length 2`)
	})

	t.Run("three tokens rejected", func(t *testing.T) {
		t.Parallel()
		err := validateWith(t, schema, `<a>ab cd ef</a>`)
		require.Error(t, err, `"ab cd ef" has token-length 3, violates length 2`)
	})

	t.Run("one token rejected", func(t *testing.T) {
		t.Parallel()
		err := validateWith(t, schema, `<a>ab</a>`)
		require.Error(t, err, `"ab" has token-length 1, violates length 2`)
	})
}

// TestLengthFacetOctetCount covers item 3: for binary builtins the length facet
// counts DECODED octets, not lexical characters.
func TestLengthFacetOctetCount(t *testing.T) {
	t.Parallel()

	t.Run("hexBinary octet count", func(t *testing.T) {
		t.Parallel()
		schema := `<element name="a" xmlns="http://relaxng.org/ns/structure/1.0"
    datatypeLibrary="http://www.w3.org/2001/XMLSchema-datatypes">
  <data type="hexBinary">
    <param name="length">2</param>
  </data>
</element>`
		// "0A0B" is 4 hex chars decoding to 2 octets.
		err := validateWith(t, schema, `<a>0A0B</a>`)
		require.NoError(t, err, `"0A0B" decodes to 2 octets`)

		// "0A" is 1 octet, violates length 2.
		err = validateWith(t, schema, `<a>0A</a>`)
		require.Error(t, err, `"0A" decodes to 1 octet, violates length 2`)
	})

	t.Run("base64Binary octet count", func(t *testing.T) {
		t.Parallel()
		schema := `<element name="a" xmlns="http://relaxng.org/ns/structure/1.0"
    datatypeLibrary="http://www.w3.org/2001/XMLSchema-datatypes">
  <data type="base64Binary">
    <param name="length">3</param>
  </data>
</element>`
		// "Zm9v" decodes to "foo" = 3 octets.
		err := validateWith(t, schema, `<a>Zm9v</a>`)
		require.NoError(t, err, `"Zm9v" decodes to 3 octets`)

		// "Zm8=" decodes to "fo" = 2 octets, violates length 3.
		err = validateWith(t, schema, `<a>Zm8=</a>`)
		require.Error(t, err, `"Zm8=" decodes to 2 octets, violates length 3`)
	})

	// Regression for the length-facet decoder bug: a base64Binary value whose
	// octet length is exactly 1 must satisfy <param name="length">1</param>, and
	// the facet must never silently fall back to a rune count when decoding
	// fails. With strict padding required by the xsd/value path, the canonical
	// length-1 value is the padded "TQ==" (decodes to one octet, "M"); its
	// unpadded form "TQ" is not a valid base64Binary lexical form and so must be
	// rejected at lexical validation before the facet is even consulted.
	t.Run("base64Binary length 1 padding", func(t *testing.T) {
		t.Parallel()
		schema := `<element name="a" xmlns="http://relaxng.org/ns/structure/1.0"
    datatypeLibrary="http://www.w3.org/2001/XMLSchema-datatypes">
  <data type="base64Binary">
    <param name="length">1</param>
  </data>
</element>`
		// Padded "TQ==" decodes to a single octet, satisfying length 1. If the
		// facet wrongly used the rune count (4) this would fail.
		err := validateWith(t, schema, `<a>TQ==</a>`)
		require.NoError(t, err, `"TQ==" decodes to 1 octet, satisfies length 1`)

		// Unpadded "TQ" is not a valid base64Binary lexical form under the strict
		// xsd/value decoder, so it must fail lexical validation; it must NOT be
		// accepted via a rune-count length of 2 either.
		err = validateWith(t, schema, `<a>TQ</a>`)
		require.Error(t, err, `unpadded "TQ" is not valid base64Binary`)

		// "Zm9v" decodes to 3 octets ("foo"), violating length 1. This exercises
		// the octet-count length-mismatch path for a strictly-valid value.
		err = validateWith(t, schema, `<a>Zm9v</a>`)
		require.Error(t, err, `"Zm9v" decodes to 3 octets, violates length 1`)
	})
}

// TestInheritedXSDLibraryResetByEmpty covers item 4: a child datatypeLibrary=""
// must RESET to the built-in library even under an inherited XSD library, so a
// bare <data type="integer"/> beneath it is NOT treated as XSD. With the reset,
// "integer" is an unknown built-in name and any value is rejected.
func TestInheritedXSDLibraryResetByEmpty(t *testing.T) {
	t.Parallel()

	schema := `<grammar xmlns="http://relaxng.org/ns/structure/1.0"
    datatypeLibrary="http://www.w3.org/2001/XMLSchema-datatypes">
  <start>
    <element name="a">
      <data type="integer" datatypeLibrary=""/>
    </element>
  </start>
</grammar>`

	// datatypeLibrary="" resets to the empty built-in library, which provides
	// only string/token. "integer" is therefore unknown and must fail.
	err := validateWith(t, schema, `<a>5</a>`)
	require.Error(t, err, `datatypeLibrary="" resets to built-in; bare integer must fail`)
}

// TestInheritedXSDLibraryApplies is the contrast to the reset case: without the
// child datatypeLibrary="" reset, the inherited XSD library applies and a bare
// <data type="integer"/> validates against the XSD integer type.
func TestInheritedXSDLibraryApplies(t *testing.T) {
	t.Parallel()

	schema := `<grammar xmlns="http://relaxng.org/ns/structure/1.0"
    datatypeLibrary="http://www.w3.org/2001/XMLSchema-datatypes">
  <start>
    <element name="a">
      <data type="integer"/>
    </element>
  </start>
</grammar>`

	err := validateWith(t, schema, `<a>5</a>`)
	require.NoError(t, err, "inherited XSD library validates integer 5")

	err = validateWith(t, schema, `<a>x</a>`)
	require.Error(t, err, "inherited XSD library rejects non-integer")
}
