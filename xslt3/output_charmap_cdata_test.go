package xslt3

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/text/unicode/norm"
)

// TestNormalizeXMLContentCDATA verifies that Unicode normalization is applied
// to the contents of a CDATA section while the surrounding markup is preserved
// verbatim. The decomposed form of "e" + combining acute accent
// (U+0065 U+0301) inside a CDATA section must be normalized to the composed
// form "é" (U+00E9) under NFC.
func TestNormalizeXMLContentCDATA(t *testing.T) {
	decomposed := "é" // "e" + combining acute accent
	composed := "é"    // precomposed "é"

	input := []byte("<e><![CDATA[" + decomposed + "]]></e>")
	want := "<e><![CDATA[" + composed + "]]></e>"

	got := string(normalizeXMLContent(input, norm.NFC))
	require.Equal(t, want, got, "CDATA section content should be normalized")
}
