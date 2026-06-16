package helium_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// TestXMLCharValidation covers character-validity enforcement across the
// scan paths that previously missed it: CDATA sections never validated the
// XML Char production, and the slow attribute/comment/PI paths rejected a
// valid U+FFFD because they treated every utf8.RuneError as invalid without
// distinguishing genuinely-invalid UTF-8 (width 1) from a real U+FFFD
// (width 3).
func TestXMLCharValidation(t *testing.T) {
	t.Parallel()

	t.Run("cdata with U+0001 is rejected", func(t *testing.T) {
		t.Parallel()
		data := []byte("<root><![CDATA[" + "\x01" + "]]></root>")
		_, err := helium.NewParser().Parse(t.Context(), data)
		require.Error(t, err)
	})

	t.Run("cdata with invalid UTF-8 byte 0xFF is rejected", func(t *testing.T) {
		t.Parallel()
		data := []byte("<root><![CDATA[" + "\xff" + "]]></root>")
		_, err := helium.NewParser().Parse(t.Context(), data)
		require.Error(t, err)
	})

	t.Run("cdata with valid content is accepted", func(t *testing.T) {
		t.Parallel()
		data := []byte("<root><![CDATA[ok]]></root>")
		_, err := helium.NewParser().Parse(t.Context(), data)
		require.NoError(t, err)
	})

	t.Run("comment with U+FFFD is accepted", func(t *testing.T) {
		t.Parallel()
		// U+FFFD (EF BF BD) is a valid XML Char and must survive the slow
		// comment scan path.
		data := []byte("<root><!--" + "\uFFFD" + "--></root>")
		_, err := helium.NewParser().Parse(t.Context(), data)
		require.NoError(t, err)
	})

	t.Run("comment with U+0001 is rejected", func(t *testing.T) {
		t.Parallel()
		data := []byte("<root><!--" + "\x01" + "--></root>")
		_, err := helium.NewParser().Parse(t.Context(), data)
		require.Error(t, err)
	})

	t.Run("slow-path attribute value with U+FFFD is accepted", func(t *testing.T) {
		t.Parallel()
		// The entity reference forces the slow attribute-value scan path
		// (the fast scanner bails on '&'). U+FFFD in that value is valid.
		data := []byte(`<root a="x&amp;` + "\uFFFD" + `"></root>`)
		_, err := helium.NewParser().Parse(t.Context(), data)
		require.NoError(t, err)
	})
}

// TestParseNameUTF8FFFD checks that U+FFFD (a valid XML NameStartChar/NameChar)
// is accepted in element and attribute names, while genuinely-invalid UTF-8 in
// a name is still rejected.
func TestParseNameUTF8FFFD(t *testing.T) {
	t.Parallel()
	for _, in := range []string{
		"<\uFFFD/>",             // element name starting with U+FFFD
		"<a\uFFFD/>",            // U+FFFD inside element name
		"<root \uFFFD=\"v\"/>",  // attr name starting with U+FFFD
		"<root x\uFFFD=\"v\"/>", // U+FFFD inside attr name (ASCII-first fast path)
	} {
		_, err := helium.NewParser().Parse(t.Context(), []byte(in))
		require.NoError(t, err, "valid U+FFFD in name must parse: %q", in)
	}
	_, err := helium.NewParser().Parse(t.Context(), []byte{'<', 0xFF, '/', '>'})
	require.Error(t, err, "invalid UTF-8 lead byte in a name must be rejected")
}

// TestXMLCharValidationOtherSlowPaths exercises the width-aware U+FFFD handling
// in the PI and DTD entity-value scan paths.
func TestXMLCharValidationOtherSlowPaths(t *testing.T) {
	t.Parallel()

	t.Run("PI content with U+FFFD is accepted", func(t *testing.T) {
		_, err := helium.NewParser().Parse(t.Context(), []byte("<root><?pi a\uFFFDb?></root>"))
		require.NoError(t, err)
	})

	t.Run("PI content with U+0001 is rejected", func(t *testing.T) {
		_, err := helium.NewParser().Parse(t.Context(), []byte("<root><?pi a\x01b?></root>"))
		require.Error(t, err)
	})

	t.Run("entity declaration value with U+FFFD is accepted", func(t *testing.T) {
		// The entity-declaration value scanner must accept the valid U+FFFD.
		doc := "<!DOCTYPE root [<!ENTITY e \"a\uFFFDb\">]><root/>"
		_, err := helium.NewParser().Parse(t.Context(), []byte(doc))
		require.NoError(t, err)
	})
}
