package helium_test

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
	xenc "golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/encoding/unicode/utf32"
)

// latin1Name is the encoding declaration value of the ISO-8859-1 case.
const latin1Name = "ISO-8859-1"

// errorContextEncodings are the input encodings of TestErrorContextChunking:
// UTF-8 read directly, and the encodings switchEncoding decodes through a
// transcoder. Every character the documents use is in ISO-8859-1 and EBCDIC 037.
var errorContextEncodings = []struct {
	label string
	name  string
	enc   xenc.Encoding
}{
	{label: utf8Name, name: utf8Name},
	{label: "UTF-16LE", name: "UTF-16", enc: unicode.UTF16(unicode.LittleEndian, unicode.UseBOM)},
	{label: "UTF-16BE", name: "UTF-16", enc: unicode.UTF16(unicode.BigEndian, unicode.UseBOM)},
	{label: "UCS-4LE", name: "UCS-4", enc: utf32.UTF32(utf32.LittleEndian, utf32.IgnoreBOM)},
	{label: "UCS-4BE", name: "UCS-4", enc: utf32.UTF32(utf32.BigEndian, utf32.IgnoreBOM)},
	{label: latin1Name, name: latin1Name, enc: charmap.ISO8859_1},
	{label: "EBCDIC 037", name: "IBM037", enc: charmap.CodePage037},
}

// errorContextDocs are the malformed document bodies of
// TestErrorContextChunking. context is the text the error's context line must
// show; a long line's context is cut, so for those only its end is given.
var errorContextDocs = []struct {
	name    string
	body    string
	context string
	cut     bool
}{
	{
		name:    "duplicate attribute on a short line",
		body:    "<r>\n  <c>café</c>\n  <d a=\"1\" a=\"2\"/>\n</r>",
		context: `  <d a="1" a="2"`,
	},
	{
		name:    "mismatched end tag past the first buffer",
		body:    "<r>\n" + strings.Repeat("  <c a=\"é\">text é</c>\n", 600) + "  <m>é</n>\n</r>",
		context: "  <m>é</",
	},
	{
		name:    "undeclared entity after a text run longer than the buffer",
		body:    "<r>" + strings.Repeat("é", 5000) + "&nope;</r>",
		context: "&nope;",
		cut:     true,
	},
	{
		name:    "duplicate attribute on a line longer than the kept context",
		body:    "<r>\n<d a=\"" + strings.Repeat("é", 2500) + "xy\" a=\"2\"/>\n</r>",
		context: `xy" a="2"`,
		cut:     true,
	},
	{
		name:    "duplicate attribute after a long line",
		body:    "<r>\n<c a=\"" + strings.Repeat("é", 9000) + "\"/>\n  <d a=\"1\" a=\"2\"/>\n</r>",
		context: `  <d a="1" a="2"`,
	},
}

// chunkSizes are the fixed read and push sizes of TestErrorContextChunking:
// single bytes, sizes that split multi-byte characters and UTF-16/UCS-4 code
// units, and sizes around the cursor's 8192-byte buffer.
var chunkSizes = []int{1, 2, 3, 5, 7, 64, 4093, 8191}

// shortReader returns at most n bytes per Read.
type shortReader struct {
	r io.Reader
	n int
}

func (r *shortReader) Read(p []byte) (int, error) {
	if len(p) > r.n {
		p = p[:r.n]
	}
	return r.r.Read(p)
}

func errorText(t *testing.T, err error) string {
	t.Helper()
	require.Error(t, err)
	return err.Error()
}

func parseReaderChunked(t *testing.T, input []byte, n int) string {
	t.Helper()
	_, err := helium.NewParser().ParseReader(t.Context(), &shortReader{r: bytes.NewReader(input), n: n})
	return errorText(t, err)
}

func pushChunked(t *testing.T, input []byte, n int) string {
	t.Helper()
	pp := helium.NewParser().NewPushParser(t.Context())
	var pushErr error
	for len(input) > 0 && pushErr == nil {
		k := min(n, len(input))
		pushErr = pp.Push(input[:k])
		input = input[k:]
	}
	_, err := pp.Close()
	if err == nil {
		err = pushErr
	}
	return errorText(t, err)
}

// errorContextLine returns the context line of a parse error's text.
func errorContextLine(t *testing.T, msg string) string {
	t.Helper()
	_, rest, ok := strings.Cut(msg, "\n -> '")
	require.True(t, ok, "error text %q has no context line", msg)
	line, ok := strings.CutSuffix(rest, "' <-- around here")
	require.True(t, ok, "error text %q has no context line", msg)
	return line
}

// TestErrorContextChunking parses malformed documents in several encodings
// from the whole input, through ParseReader with fixed-size short reads, and
// through the push parser with fixed-size pushes, and requires every parse to
// report the same error text: message, line, column, and context line. The
// whole-input parse of each encoding must also match the UTF-8 one.
func TestErrorContextChunking(t *testing.T) {
	t.Parallel()

	for _, doc := range errorContextDocs {
		t.Run(doc.name, func(t *testing.T) {
			t.Parallel()

			_, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?>`+"\n"+doc.body))
			want := errorText(t, err)
			line := errorContextLine(t, want)
			require.True(t, strings.HasSuffix(line, doc.context), "context line %q must end with %q", line, doc.context)
			require.True(t, utf8.ValidString(line), "context line %q must be valid UTF-8", line)
			if doc.cut {
				require.Less(t, len(line), len(doc.body), "the context of a long line must be cut")
			} else {
				require.Equal(t, doc.context, line)
			}

			for _, e := range errorContextEncodings {
				t.Run(e.label, func(t *testing.T) {
					t.Parallel()

					input := []byte(`<?xml version="1.0" encoding="` + e.name + `"?>` + "\n" + doc.body)
					if e.enc != nil {
						encoded, encErr := e.enc.NewEncoder().Bytes(input)
						require.NoError(t, encErr)
						input = encoded
					}
					_, parseErr := helium.NewParser().Parse(t.Context(), input)
					require.Equal(t, want, errorText(t, parseErr), "whole input")
					for _, n := range chunkSizes {
						require.Equal(t, want, parseReaderChunked(t, input, n), "ParseReader, %d-byte reads", n)
						require.Equal(t, want, pushChunked(t, input, n), "push parser, %d-byte pushes", n)
					}
				})
			}
		})
	}
}
