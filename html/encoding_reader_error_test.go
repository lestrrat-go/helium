package html_test

import (
	"errors"
	"io"
	"testing"

	"github.com/lestrrat-go/helium/html"

	"github.com/stretchr/testify/require"
)

// dataThenErrReader returns its payload together with a non-EOF error on the
// same Read (which io.Reader explicitly permits), then reports the same error
// (or EOF) on subsequent reads. It models a reader that detects
// corruption/truncation only after emitting bytes, e.g. a checksumming or
// decompressing stream.
type dataThenErrReader struct {
	data []byte
	err  error
	done bool
}

func (r *dataThenErrReader) Read(p []byte) (int, error) {
	if r.done {
		return 0, r.err
	}
	r.done = true
	n := copy(p, r.data)
	return n, r.err
}

// exactFillReader returns its entire payload together with a non-EOF error on
// the first Read (modeling a reader that fills the caller's buffer exactly while
// reporting corruption), then reports io.EOF — not the original error — on all
// subsequent reads. The error is therefore observable ONLY from the first Read,
// so dropping it during the charset sniff makes it vanish for good.
type exactFillReader struct {
	data []byte
	err  error
	done bool
}

func (r *exactFillReader) Read(p []byte) (int, error) {
	if r.done {
		return 0, io.EOF
	}
	r.done = true
	n := copy(p, r.data)
	return n, r.err
}

// chunkReader returns one chunk's bytes per logical chunk, handling partial
// reads when the destination buffer is smaller than the chunk. The error
// attached to a chunk is returned together with that chunk's FINAL bytes on the
// same Read call (io.Reader explicitly permits n > 0 with a non-nil error).
type chunkReader struct {
	chunks []chunk
	pos    int // index into chunks
	off    int // offset within the current chunk's data
}

type chunk struct {
	data []byte
	err  error
}

func (r *chunkReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.chunks) {
		return 0, io.EOF
	}
	c := r.chunks[r.pos]
	n := copy(p, c.data[r.off:])
	r.off += n
	if r.off >= len(c.data) {
		// Chunk fully delivered: advance and attach this chunk's error.
		r.pos++
		r.off = 0
		return n, c.err
	}
	return n, nil
}

// TestParseReaderSurfacesReadErrorUndecidedUTF8 covers the deferred-Latin-1
// reader's undecided (still-valid-UTF-8) path: the underlying reader returns
// valid UTF-8 bytes together with a sentinel non-EOF error before any invalid
// byte appears. The buffered bytes must be delivered, but the sentinel error
// must then surface rather than the parse silently succeeding.
func TestParseReaderSurfacesReadErrorUndecidedUTF8(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("checksum mismatch")

	// Pure ASCII / valid UTF-8 document with no invalid bytes: the deferred
	// reader stays undecided and buffers everything until EOF/err.
	r := &dataThenErrReader{
		data: []byte("<html><body><p>hello world</p></body></html>"),
		err:  sentinel,
	}

	_, err := html.NewParser().ParseReader(t.Context(), r)
	require.Error(t, err, "a read error before EOF must not be swallowed")
	require.ErrorIs(t, err, sentinel,
		"the underlying non-EOF read error must surface out of ParseReader")
}

// TestParseReaderSurfacesReadErrorExactFill covers the charset-sniff peek when
// the underlying reader fills the 1024-byte detection buffer EXACTLY and returns
// a non-EOF sentinel error on that same Read (n == 1024, err != nil), then EOF.
// io.ReadFull would have collapsed this to (1024, nil), dropping the error and
// letting a truncated/checksummed stream parse clean. The manual sniff loop must
// preserve the sentinel and surface it out of ParseReader.
func TestParseReaderSurfacesReadErrorExactFill(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("checksum mismatch")

	// Build a valid-UTF-8 document of exactly 1024 bytes so the first peek read
	// fills the detection buffer to the brim and arrives with the sentinel.
	doc := []byte("<html><body>")
	for len(doc) < 1024-len("</body></html>") {
		doc = append(doc, []byte("<p>x</p>")...)
	}
	doc = doc[:1024-len("</body></html>")]
	doc = append(doc, []byte("</body></html>")...)
	require.Len(t, doc, 1024, "regression requires an exact 1024-byte head")

	// exactFillReader returns the whole 1024-byte payload together with the
	// sentinel on the first Read, then reports io.EOF (NOT the sentinel) on every
	// subsequent Read. This isolates the sniff-phase bug: io.ReadFull would
	// collapse (1024, sentinel) to (1024, nil) and, because no later Read repeats
	// the error, the error would be lost entirely.
	r := &exactFillReader{data: doc, err: sentinel}

	_, err := html.NewParser().ParseReader(t.Context(), r)
	require.Error(t, err, "an exact-fill read error must not be swallowed by the sniff")
	require.ErrorIs(t, err, sentinel,
		"the underlying non-EOF read error must surface out of ParseReader")
}

// TestParseReaderSurfacesReadErrorAfterLatin1Switch covers the deferred-Latin-1
// reader's post-switch path: the stream first flips to Latin-1 (a non-UTF-8
// byte appears), and only on a LATER read does the underlying reader deliver
// bytes together with a sentinel non-EOF error. The error must surface instead
// of the parse completing cleanly once the converted bytes drain.
func TestParseReaderSurfacesReadErrorAfterLatin1Switch(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("truncated stream")

	// Build a document larger than the 1024-byte charset-detection window so the
	// Latin-1 switch and the read error both occur during streaming, AFTER the
	// peek phase — exercising the switched (fillLatin1) path rather than the
	// peek-level error handling.
	var head []byte
	head = append(head, []byte("<html><body>")...)
	for len(head) < 2048 {
		head = append(head, []byte("<p>filler text line</p>\n")...)
	}

	// First streamed chunk introduces the lone Latin-1 byte (0xE9) well past the
	// detection window, forcing the switch to Windows-1252. No error yet.
	switchChunk := []byte("<p>caf\xE9 latin one</p>")
	// A later chunk delivers more Latin-1 content together with the sentinel
	// error on the same Read, which io.Reader explicitly permits.
	errChunk := []byte("<p>r\xE9sum\xE9</p></body></html>")

	r := &chunkReader{
		chunks: []chunk{
			{data: head},
			{data: switchChunk},
			{data: errChunk, err: sentinel},
		},
	}

	_, err := html.NewParser().ParseReader(t.Context(), r)
	require.Error(t, err, "a read error after the Latin-1 switch must not be swallowed")
	require.ErrorIs(t, err, sentinel,
		"the underlying non-EOF read error must surface out of ParseReader")
}
