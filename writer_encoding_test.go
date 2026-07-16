package helium_test

import (
	"bytes"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// TestOutputEncodingMatchesEmittedBytes asserts that when OutputEncoding
// overrides the declaration to a transcodable encoding, the emitted octets are
// re-encoded to match — the declaration and the bytes agree. With
// EscapeNonASCII disabled a non-ASCII character is written literally, so the
// mismatch (raw UTF-8 octets under a Latin-1 declaration) is observable.
func TestOutputEncodingMatchesEmittedBytes(t *testing.T) {
	t.Parallel()

	src := []byte(`<?xml version="1.0" encoding="UTF-8"?><root>é</root>`)
	doc, err := helium.NewParser().Parse(t.Context(), src)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = helium.NewWriter().EscapeNonASCII(false).OutputEncoding("ISO-8859-1").WriteTo(&buf, doc)
	require.NoError(t, err)

	out := buf.String()
	require.Contains(t, out, `encoding="ISO-8859-1"`)
	// U+00E9 must be the single ISO-8859-1 byte 0xE9, NOT the two UTF-8 bytes
	// 0xC3 0xA9 that a Latin-1 declaration would misdescribe.
	require.Contains(t, out, "\xe9")
	require.NotContains(t, out, "\xc3\xa9")
}

// TestOutputEncodingUnsupportedErrors asserts that an explicitly set
// OutputEncoding naming an encoding the writer cannot emit is a hard error,
// rather than silently emitting UTF-8 under a false declaration.
func TestOutputEncodingUnsupportedErrors(t *testing.T) {
	t.Parallel()

	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root/>`))
	require.NoError(t, err)

	var buf bytes.Buffer
	err = helium.NewWriter().OutputEncoding("no-such-enc").WriteTo(&buf, doc)
	require.ErrorIs(t, err, helium.ErrUnsupportedOutputEncoding)
}

// TestSerializeNoEncodingOverrideUnchanged asserts the no-override path is
// byte-identical to prior behavior: default escaping still emits a hex
// character reference under the document's own declaration, and an unloadable
// parsed encoding is NOT turned into a hard error when no override is set.
func TestSerializeNoEncodingOverrideUnchanged(t *testing.T) {
	t.Parallel()

	src := []byte(`<?xml version="1.0" encoding="UTF-8"?><root>é</root>`)
	doc, err := helium.NewParser().Parse(t.Context(), src)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = helium.NewWriter().WriteTo(&buf, doc)
	require.NoError(t, err)

	out := buf.String()
	require.Contains(t, out, `encoding="UTF-8"`)
	require.Contains(t, out, "&#xE9;")
	require.NotContains(t, out, "\xc3\xa9")

	// An unloadable encoding recorded on the document must still serialize
	// (declaration-only) when there is no OutputEncoding override.
	built := helium.NewDefaultDocument()
	root := built.CreateElement("root")
	require.NoError(t, built.SetDocumentElement(root))
	built.SetEncoding("x-unknown-enc")

	var buf2 bytes.Buffer
	err = helium.NewWriter().WriteTo(&buf2, built)
	require.NoError(t, err)
	require.Contains(t, buf2.String(), `encoding="x-unknown-enc"`)
}
