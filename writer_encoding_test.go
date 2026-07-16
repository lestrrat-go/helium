package helium_test

import (
	"bytes"
	"strings"
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

// TestOutputEncodingUSASCIIEscapesNonASCII asserts that an explicit US-ASCII
// OutputEncoding override escapes every non-ASCII character as a numeric
// character reference rather than emitting raw UTF-8 under a US-ASCII
// declaration. US-ASCII can represent any document via character references, so
// this is not an unsupported-encoding error; the result is valid US-ASCII and
// reparses to the original content.
func TestOutputEncodingUSASCIIEscapesNonASCII(t *testing.T) {
	t.Parallel()

	// U+2603 (BMP, beyond Latin-1) and U+00E9 (Latin-1) together confirm the full
	// non-ASCII range is escaped, not just Latin-1.
	src := []byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?><root a=\"é\">☃</root>")
	doc, err := helium.NewParser().Parse(t.Context(), src)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = helium.NewWriter().OutputEncoding("US-ASCII").WriteTo(&buf, doc)
	require.NoError(t, err)

	out := buf.String()
	require.Contains(t, out, `encoding="US-ASCII"`)
	require.Contains(t, out, "&#x2603;")
	require.Contains(t, out, "&#xE9;")
	// No raw non-ASCII octet may appear under the US-ASCII declaration.
	for i := range len(out) {
		require.Less(t, out[i], byte(0x80), "non-ASCII octet 0x%X at %d in %q", out[i], i, out)
	}

	// The escaped output reparses, and re-serializing (default UTF-8) recovers the
	// original non-ASCII characters — the character references round-trip.
	rt, err := helium.NewParser().Parse(t.Context(), buf.Bytes())
	require.NoError(t, err)
	var rtbuf bytes.Buffer
	require.NoError(t, helium.NewWriter().EscapeNonASCII(false).WriteTo(&rtbuf, rt))
	require.Contains(t, rtbuf.String(), "☃")
	require.Contains(t, rtbuf.String(), "é")
}

// TestOutputEncodingUSASCIIRejectsNonCharRefContexts asserts that under an
// explicit US-ASCII OutputEncoding a non-ASCII character in a context that
// cannot hold a character reference — comment text, CDATA section, PI
// target/data, an element/attribute name — fails with
// ErrUnsupportedOutputEncoding rather than emitting raw UTF-8 under a US-ASCII
// declaration.
func TestOutputEncodingUSASCIIRejectsNonCharRefContexts(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"comment":        `<?xml version="1.0" encoding="UTF-8"?><root><!--é--></root>`,
		"cdata":          `<?xml version="1.0" encoding="UTF-8"?><root><![CDATA[é]]></root>`,
		"pi-data":        `<?xml version="1.0" encoding="UTF-8"?><root><?pi dáta?></root>`,
		"element-name":   `<?xml version="1.0" encoding="UTF-8"?><rôot></rôot>`,
		"attribute-name": `<?xml version="1.0" encoding="UTF-8"?><root ättr="v"></root>`,
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
			require.NoError(t, err)

			var buf bytes.Buffer
			err = helium.NewWriter().OutputEncoding("US-ASCII").WriteTo(&buf, doc)
			require.ErrorIs(t, err, helium.ErrUnsupportedOutputEncoding)
			// No raw non-ASCII octet may have leaked before the error fired.
			for i := range buf.Len() {
				require.Less(t, buf.Bytes()[i], byte(0x80), "leaked non-ASCII octet 0x%X", buf.Bytes()[i])
			}
		})
	}
}

// TestOutputEncodingASCIIAliasesMatchUSASCII asserts that every registered
// US-ASCII alias takes the same char-referencing path as the canonical
// "US-ASCII" name, rather than emitting raw UTF-8 because the loaded encoder
// delegates to UTF-8.
func TestOutputEncodingASCIIAliasesMatchUSASCII(t *testing.T) {
	t.Parallel()

	src := []byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?><root a=\"é\">☃</root>")
	doc, err := helium.NewParser().Parse(t.Context(), src)
	require.NoError(t, err)

	serialize := func(t *testing.T, enc string) string {
		t.Helper()
		var buf bytes.Buffer
		require.NoError(t, helium.NewWriter().OutputEncoding(enc).WriteTo(&buf, doc))
		return buf.String()
	}

	// The canonical form, minus the declaration line (which carries the differing
	// encoding label), is the body every alias must reproduce.
	canonical := serialize(t, "US-ASCII")
	canonicalBody := canonical[strings.IndexByte(canonical, '\n'):]

	for _, alias := range []string{"csASCII", "ANSI_X3.4-1968", "ANSI_X3.4-1986", "iso-ir-6", "ISO646-US", "us", "ascii"} {
		out := serialize(t, alias)
		require.Contains(t, out, `encoding="`+alias+`"`)
		require.Contains(t, out, "&#x2603;")
		require.Contains(t, out, "&#xE9;")
		for i := range len(out) {
			require.Less(t, out[i], byte(0x80), "alias %q leaked non-ASCII octet 0x%X", alias, out[i])
		}
		require.Equal(t, canonicalBody, out[strings.IndexByte(out, '\n'):], "alias %q body differs from US-ASCII", alias)
	}
}

// TestOutputEncodingElementFragmentUnchanged asserts OutputEncoding affects the
// Document path only: serializing a bare element (a fragment, which carries no
// XML declaration) with OutputEncoding("US-ASCII") is byte-identical to
// serializing it with no override. A character above Latin-1 (which the default
// escapeNonASCII does not touch) makes any leaked US-ASCII escaping observable.
func TestOutputEncodingElementFragmentUnchanged(t *testing.T) {
	t.Parallel()

	src := []byte(`<?xml version="1.0" encoding="UTF-8"?><root>世</root>`)
	doc, err := helium.NewParser().Parse(t.Context(), src)
	require.NoError(t, err)
	root := doc.DocumentElement()

	var plain, ascii bytes.Buffer
	require.NoError(t, helium.NewWriter().WriteTo(&plain, root))
	require.NoError(t, helium.NewWriter().OutputEncoding("US-ASCII").WriteTo(&ascii, root))
	require.Equal(t, plain.String(), ascii.String())
	require.Contains(t, plain.String(), "世")
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
