package html_test

import (
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
// (utf8SanitizeReader on the declared charset=utf-8 path, deferredLatin1Reader
// on the undeclared valid-UTF-8 path) legitimately returns (0, nil) while it
// withholds the incomplete rune, so the cursor fill loop must not treat that as
// fatal.
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
// content padded past the 1024-byte charset prescan, followed by a multibyte
// rune ("é") delivered one byte at a time, must parse successfully. The
// incomplete rune at a chunk boundary makes the encoding reader emit a
// transient (0, nil); a fill loop that fails on the first such read wrongly
// rejected valid streaming input.
//
// Two encoding paths produce that transient (0, nil): utf8SanitizeReader (only
// when charset=utf-8 is declared within the prescan window) and
// deferredLatin1Reader (an undeclared input that proves to be valid UTF-8).
// Both are covered so the split-rune assertion holds on each.
func TestParseReaderSplitMultibyteRuneAfterPrescan(t *testing.T) {
	const meta = `<meta charset="utf-8">`

	testCases := []struct {
		name string
		// reader chooses which encoding path handles the stream:
		//  - declaredUTF8=true  → charset=utf-8 in head → utf8SanitizeReader
		//  - declaredUTF8=false → undeclared valid UTF-8 → deferredLatin1Reader
		declaredUTF8 bool
	}{
		{name: "declared-utf8-sanitize-path", declaredUTF8: true},
		{name: "undeclared-deferred-latin1-path", declaredUTF8: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Build a <head> that the 1024-byte prescan consumes whole, then a
			// multibyte rune delivered one byte at a time. For the declared
			// path the <meta charset="utf-8"> must land within the first 1024
			// bytes; pad the remainder so the boundary falls before "é".
			var prefix string
			if tc.declaredUTF8 {
				prefix = "<html><head>" + meta + "</head><body>"
			} else {
				prefix = "<html><body>"
			}
			pad := strings.Repeat("a", 1024)
			html5 := prefix + pad + "é</body></html>"

			rdr := &oneBytePerReadReader{data: []byte(html5)}

			type result struct {
				doc any
				err error
			}
			done := make(chan result, 1)
			go func() {
				doc, err := html.NewParser().ParseReader(t.Context(), rdr)
				done <- result{doc: doc, err: err}
			}()

			select {
			case res := <-done:
				require.NoError(t, res.err, "split multibyte rune after prescan must parse successfully")
				require.NotNil(t, res.doc, "a document must be produced")
			case <-time.After(10 * time.Second):
				t.Fatal("ParseReader hung on a one-byte-at-a-time split multibyte rune")
			}
		})
	}
}
