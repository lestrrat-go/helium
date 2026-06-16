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
		data := []byte("<root><!--" + "�" + "--></root>")
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
		data := []byte(`<root a="x&amp;` + "�" + `"></root>`)
		_, err := helium.NewParser().Parse(t.Context(), data)
		require.NoError(t, err)
	})
}
