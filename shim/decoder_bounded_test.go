package shim_test

import (
	"io"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium/shim"
	"github.com/stretchr/testify/require"
)

// countingReader records how many bytes have been read from the wrapped reader.
type countingReader struct {
	r io.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
}

// largeUTF16Doc builds a multi-megabyte little-endian UTF-16 document with the
// given XML declaration, so a bounded-prefix reader is provably not reading the
// whole stream.
func largeUTF16Doc(t *testing.T, decl string) string {
	t.Helper()
	var b strings.Builder
	b.WriteString(decl)
	b.WriteString("<root>")
	for range 300_000 {
		b.WriteString("<a></a>")
	}
	b.WriteString("</root>")
	src := utf16leBOM(b.String())
	require.Greater(t, len(src), 4_000_000, "want a multi-MB fixed-width document")
	return src
}

// TestDecoderFixedWidthBoundedRead pins that the reader-backed Decoder applies
// its fixed-width encoding gate from a BOUNDED prefix of the stream, never by
// buffering the whole document into memory. A ~4 MB UTF-16 document declaring a
// non-UTF-8 encoding with no CharsetReader is rejected; the rejection verdict
// must be reached after reading only a small prefix, not the entire stream.
func TestDecoderFixedWidthBoundedRead(t *testing.T) {
	src := largeUTF16Doc(t, `<?xml version="1.0" encoding="UTF-16"?>`)

	cr := &countingReader{r: strings.NewReader(src)}
	dec := shim.NewDecoder(t.Context(), cr)
	_, err := dec.Token()
	require.Error(t, err, "a UTF-16 document declaring encoding=UTF-16 with no CharsetReader is rejected")
	require.Less(t, cr.n, int64(64*1024),
		"the encoding gate must read only a bounded prefix; read %d of %d bytes", cr.n, len(src))
}

// TestDecoderUTF8StreamingPrefix confirms the ordinary UTF-8 path is unaffected:
// the first Token() is produced after reading only a tiny prefix, not the whole
// document.
func TestDecoderUTF8StreamingPrefix(t *testing.T) {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?><root>`)
	for range 300_000 {
		b.WriteString("<a></a>")
	}
	b.WriteString("</root>")
	src := b.String()
	require.Greater(t, len(src), 2_000_000, "want a multi-MB UTF-8 document")

	cr := &countingReader{r: strings.NewReader(src)}
	dec := shim.NewDecoder(t.Context(), cr)
	tok, err := dec.Token()
	require.NoError(t, err)
	require.NotNil(t, tok)
	require.Less(t, cr.n, int64(64*1024),
		"the UTF-8 path must stream; read %d of %d bytes before the first token", cr.n, len(src))
}
