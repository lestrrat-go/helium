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
