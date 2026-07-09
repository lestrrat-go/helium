package helium_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// parseValidating parses src as a validating processor with entity substitution,
// returning the DTD-validation errors collected (nil error means the document is
// valid).
func parseECValidate(t *testing.T, src string) error {
	t.Helper()
	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	_, err := helium.NewParser().
		SubstituteEntities(true).
		ValidateDTD(true).
		DefaultDTDAttributes(true).
		ErrorHandler(collector).
		Parse(t.Context(), []byte(src))
	return err
}

// TestElementContentCDATAIsInvalid asserts that a CDATA section in element-only
// content is a validity error (VC: Element Valid) even when the CDATA section is
// empty or whitespace-only: a CDATA section is character data and never matches
// the S nonterminal (XML §2.4/§3.2.1). W3C xmlconf "empty" (sun/invalid).
func TestElementContentCDATAIsInvalid(t *testing.T) {
	t.Parallel()

	t.Run("whitespace-only CDATA in element content is invalid", func(t *testing.T) {
		t.Parallel()
		err := parseECValidate(t, `<!DOCTYPE foo [
<!ELEMENT foo (a+)>
<!ELEMENT a EMPTY>
]>
<foo><a/><![CDATA[ ]]><a/></foo>`)
		require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
	})

	t.Run("empty CDATA in element content is invalid", func(t *testing.T) {
		t.Parallel()
		err := parseECValidate(t, `<!DOCTYPE foo [
<!ELEMENT foo (a+)>
<!ELEMENT a EMPTY>
]>
<foo><a/><![CDATA[]]><a/></foo>`)
		require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
	})
}

// TestElementContentWhitespaceStillValid guards against over-rejection: literal
// ignorable whitespace between child elements in element-only content is valid,
// and CDATA is still permitted in mixed and ANY content.
func TestElementContentWhitespaceStillValid(t *testing.T) {
	t.Parallel()

	t.Run("literal ignorable whitespace in element content is valid", func(t *testing.T) {
		t.Parallel()
		err := parseECValidate(t, `<!DOCTYPE foo [
<!ELEMENT foo (a+)>
<!ELEMENT a EMPTY>
]>
<foo>
  <a/>
  <a/>
</foo>`)
		require.NoError(t, err)
	})

	t.Run("CDATA in mixed content is valid", func(t *testing.T) {
		t.Parallel()
		err := parseECValidate(t, `<!DOCTYPE foo [
<!ELEMENT foo (#PCDATA|a)*>
<!ELEMENT a EMPTY>
]>
<foo><a/><![CDATA[ text ]]><a/></foo>`)
		require.NoError(t, err)
	})

	t.Run("CDATA in ANY content is valid", func(t *testing.T) {
		t.Parallel()
		err := parseECValidate(t, `<!DOCTYPE foo [
<!ELEMENT foo ANY>
<!ELEMENT a EMPTY>
]>
<foo><a/><![CDATA[ text ]]><a/></foo>`)
		require.NoError(t, err)
	})
}

// TestNmtokensCharRefWhitespace asserts that a whitespace character introduced by
// a character reference in an NMTOKENS attribute value is NOT a token separator:
// attribute-value normalization (XML §3.3.3) folds literal whitespace to a single
// space but leaves character-reference whitespace verbatim, so `abc&#9;xyz`
// normalizes to the single token "abc\txyz", which is not a valid NMTOKEN
// (the tab is not a NameChar). W3C xmlconf rmt-e2e-20.
func TestNmtokensCharRefWhitespace(t *testing.T) {
	t.Parallel()

	t.Run("char-ref tab makes an invalid NMTOKEN", func(t *testing.T) {
		t.Parallel()
		err := parseECValidate(t, `<!DOCTYPE foo [
<!ELEMENT foo ANY>
<!ATTLIST foo bar NMTOKENS #IMPLIED>
]>
<foo bar="abc&#9;xyz"/>`)
		require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
	})

	t.Run("literal-space-separated NMTOKENS is valid", func(t *testing.T) {
		t.Parallel()
		err := parseECValidate(t, `<!DOCTYPE foo [
<!ELEMENT foo ANY>
<!ATTLIST foo bar NMTOKENS #IMPLIED>
]>
<foo bar="abc   xyz"/>`)
		require.NoError(t, err)
	})

	t.Run("literal-tab-separated NMTOKENS is valid (normalized to space)", func(t *testing.T) {
		t.Parallel()
		err := parseECValidate(t, "<!DOCTYPE foo [\n<!ELEMENT foo ANY>\n<!ATTLIST foo bar NMTOKENS #IMPLIED>\n]>\n<foo bar=\"abc\txyz\"/>")
		require.NoError(t, err)
	})
}

// TestElementContentCharRefWhitespace asserts that whitespace introduced by a
// character reference (directly or via a general entity whose replacement text
// is a character reference) does NOT match the S nonterminal and is therefore a
// validity error in element-only content, while literal whitespace — including
// an internal entity whose replacement text is itself literal whitespace — stays
// ignorable (XML §3.2.1 as clarified by errata 2e E15). W3C xmlconf rmt-e2e-15*.
func TestElementContentCharRefWhitespace(t *testing.T) {
	t.Parallel()

	// E15g: a direct character reference producing whitespace between element
	// children in element-only content is not ignorable.
	t.Run("direct char-ref whitespace is invalid (E15g)", func(t *testing.T) {
		t.Parallel()
		err := parseECValidate(t, `<!DOCTYPE foo [
<!ELEMENT foo (foo*)>
]>
<foo><foo/>&#32;<foo/></foo>`)
		require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
	})

	// E15h: a general entity whose replacement text is a character reference
	// (&#38;#32; declares the entity value "&#32;") re-parses to a space at
	// inclusion time — that whitespace came from a character reference.
	t.Run("entity-of-char-ref whitespace is invalid (E15h)", func(t *testing.T) {
		t.Parallel()
		err := parseECValidate(t, `<!DOCTYPE foo [
<!ELEMENT foo (foo*)>
<!ENTITY space "&#38;#32;">
]>
<foo><foo/>&space;<foo/></foo>`)
		require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
	})

	// E15e: an internal entity whose replacement text is a literal space is
	// ignorable whitespace — valid.
	t.Run("entity-of-literal-space whitespace is valid (E15e)", func(t *testing.T) {
		t.Parallel()
		err := parseECValidate(t, `<!DOCTYPE foo [
<!ELEMENT foo (foo*)>
<!ENTITY space " ">
]>
<foo><foo/>&space;<foo/></foo>`)
		require.NoError(t, err)
	})

	// E15f: a character reference in an entity's LITERAL value expands at
	// declaration time, so the replacement text is a literal space — the
	// whitespace does not originate from a character reference at inclusion time
	// and stays ignorable — valid.
	t.Run("entity literal &#32; expands at decl time and is valid (E15f)", func(t *testing.T) {
		t.Parallel()
		err := parseECValidate(t, `<!DOCTYPE foo [
<!ELEMENT foo (foo*)>
<!ENTITY space "&#32;">
]>
<foo><foo/>&space;<foo/></foo>`)
		require.NoError(t, err)
	})

	// E15a: a reference is content per XML production [43]; an element declared
	// EMPTY that contains one is invalid even when the reference expands to
	// nothing and leaves the element with no child node.
	t.Run("empty entity reference in an EMPTY element is invalid (E15a)", func(t *testing.T) {
		t.Parallel()
		err := parseECValidate(t, `<!DOCTYPE foo [
<!ELEMENT foo EMPTY>
<!ENTITY empty "">
]>
<foo>&empty;</foo>`)
		require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
	})

	// Literal whitespace mixed with a character reference in element-only content
	// is still invalid — the merged text node carries the char-reference origin.
	t.Run("literal-plus-char-ref whitespace is invalid", func(t *testing.T) {
		t.Parallel()
		err := parseECValidate(t, `<!DOCTYPE foo [
<!ELEMENT foo (foo*)>
]>
<foo><foo/> &#32; <foo/></foo>`)
		require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
	})

	// A character reference producing whitespace in MIXED content is character
	// data, which mixed content permits — still valid.
	t.Run("char-ref whitespace in mixed content is valid", func(t *testing.T) {
		t.Parallel()
		err := parseECValidate(t, `<!DOCTYPE foo [
<!ELEMENT foo (#PCDATA|a)*>
<!ELEMENT a EMPTY>
]>
<foo><a/>&#32;<a/></foo>`)
		require.NoError(t, err)
	})
}

// TestCharRefProvenanceSerializationUnchanged guards the blast radius: the
// char-reference-origin marker on a Text node is invisible to serialization —
// a document whose text came from a character reference serializes byte-for-byte
// identically to the same document written with literal text.
func TestCharRefProvenanceSerializationUnchanged(t *testing.T) {
	t.Parallel()

	parse := func(src string) *helium.Document {
		t.Helper()
		doc, err := helium.NewParser().
			SubstituteEntities(true).
			ValidateDTD(true).
			DefaultDTDAttributes(true).
			Parse(t.Context(), []byte(src))
		require.NoError(t, err)
		return doc
	}

	literal := parse(`<!DOCTYPE foo [<!ELEMENT foo (#PCDATA)>]><foo>a b</foo>`)
	charRef := parse(`<!DOCTYPE foo [<!ELEMENT foo (#PCDATA)>]><foo>a&#32;b</foo>`)

	litStr, err := helium.WriteString(literal)
	require.NoError(t, err)
	refStr, err := helium.WriteString(charRef)
	require.NoError(t, err)
	require.Equal(t, litStr, refStr)
}
