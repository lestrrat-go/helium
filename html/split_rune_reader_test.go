package html_test

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/lestrrat-go/helium/html"
	"github.com/stretchr/testify/require"
)

// oneBytePerReadReader delivers its payload one byte per Read call. This models
// a slow streaming source that splits a multibyte UTF-8 rune across reads. When
// such a split lands on a rune boundary, the html encoding wrapper
// (utf8SanitizeReader) legitimately returns (0, nil) while it withholds the
// incomplete rune, so the cursor fill loop must not treat that as fatal.
type oneBytePerReadReader struct {
	data []byte
	pos  int
}

func (r *oneBytePerReadReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	if len(p) == 0 {
		return 0, nil
	}
	p[0] = r.data[r.pos]
	r.pos++
	return 1, nil
}

// TestParseReaderSplitMultibyteRuneAfterPrescan guards a streaming regression:
// 1024 ASCII bytes (which the charset prescan consumes whole) followed by a
// multibyte rune ("é") delivered one byte at a time must parse successfully.
// The incomplete rune at a chunk boundary makes utf8SanitizeReader emit a
// transient (0, nil); a fill loop that fails on the first such read wrongly
// rejected valid streaming input.
func TestParseReaderSplitMultibyteRuneAfterPrescan(t *testing.T) {
	// Pad <body> content with 1024 ASCII bytes so the 1024-byte prescan
	// boundary falls before the multibyte rune.
	pad := strings.Repeat("a", 1024)
	html5 := "<html><body>" + pad + "é</body></html>"

	rdr := &oneBytePerReadReader{data: []byte(html5)}

	type result struct {
		doc interface{}
		err error
	}
	done := make(chan result, 1)
	go func() {
		doc, err := html.NewParser().ParseReader(context.Background(), rdr)
		done <- result{doc: doc, err: err}
	}()

	select {
	case res := <-done:
		require.NoError(t, res.err, "split multibyte rune after prescan must parse successfully")
		require.NotNil(t, res.doc, "a document must be produced")
	case <-time.After(10 * time.Second):
		t.Fatal("ParseReader hung on a one-byte-at-a-time split multibyte rune")
	}
}
