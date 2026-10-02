package strcursor_test

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/lestrrat-go/helium/internal/strcursor"
	"github.com/stretchr/testify/require"
)

// dataThenErrReader returns its payload together with a non-EOF error on the
// same Read (which io.Reader explicitly permits), then reports EOF. It models a
// reader that detects corruption/truncation only after emitting the final
// bytes, e.g. a checksumming or decompressing stream.
type dataThenErrReader struct {
	data []byte
	err  error
	done bool
}

func (r *dataThenErrReader) Read(p []byte) (int, error) {
	if r.done {
		return 0, io.EOF
	}
	r.done = true
	n := copy(p, r.data)
	return n, r.err
}

func TestByteCursorSurfacesErrorReturnedWithData(t *testing.T) {
	wantErr := errors.New("checksum mismatch")
	cur := strcursor.NewByteCursor(&dataThenErrReader{
		data: []byte("<root/>"),
		err:  wantErr,
	})

	// The buffered bytes must remain readable.
	require.Equal(t, "<root/>", cur.PeekString(7))

	// Consume the buffered bytes.
	require.NoError(t, cur.Advance(7))

	// Once the buffer drains, the cursor must report the underlying error,
	// and must not treat the stream as cleanly terminated.
	require.True(t, cur.Done(), "Done should be true after buffer drains")
	require.ErrorIs(t, cur.Err(), wantErr, "the non-EOF read error must be surfaced after the buffered bytes are consumed")
}

// zeroProgressReader always returns (0, nil) for a non-empty request, never
// advancing and never erroring. A naive fill loop spins on it forever; the
// bounded-retry fill loop must give up after maxZeroProgressReads.
type zeroProgressReader struct{}

func (zeroProgressReader) Read(p []byte) (int, error) {
	return 0, nil
}

func TestByteCursorZeroProgressReaderDoesNotHang(t *testing.T) {
	cur := strcursor.NewByteCursor(zeroProgressReader{})

	type result struct {
		done bool
		err  error
	}
	done := make(chan result, 1)
	go func() {
		d := cur.Done()
		done <- result{done: d, err: cur.Err()}
	}()

	select {
	case res := <-done:
		require.True(t, res.done, "a zero-progress reader must terminate fill, not spin")
		require.ErrorIs(t, res.err, io.ErrNoProgress, "a zero-progress reader must surface io.ErrNoProgress after the bounded retry count")
	case <-time.After(5 * time.Second):
		t.Fatal("ByteCursor fillBuffer hung on a zero-progress reader")
	}
}

// slowSplitReader emits its payload one byte per Read, returning a single
// (0, nil) "waiting" read before each real byte. A fill loop that treats the
// first (0, nil) as fatal would reject this legitimate slow producer.
type slowSplitReader struct {
	data    []byte
	pos     int
	pending bool // true after a (0, nil) stall, so the next Read yields a byte
}

func (r *slowSplitReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	if len(p) == 0 {
		return 0, nil
	}
	if !r.pending {
		r.pending = true
		return 0, nil // stall once before delivering the next byte
	}
	r.pending = false
	p[0] = r.data[r.pos]
	r.pos++
	return 1, nil
}

func TestByteCursorSlowSplitReaderMakesProgress(t *testing.T) {
	cur := strcursor.NewByteCursor(&slowSplitReader{data: []byte("<root/>")})

	type result struct {
		hasPrefix bool
		err       error
	}
	done := make(chan result, 1)
	go func() {
		hp := cur.HasPrefix([]byte("<root/>"))
		done <- result{hasPrefix: hp, err: cur.Err()}
	}()

	select {
	case res := <-done:
		require.True(t, res.hasPrefix, "a slow reader that emits (0, nil) between bytes must still be consumed")
		require.NoError(t, res.err, "a progressing reader must not surface io.ErrNoProgress")
	case <-time.After(5 * time.Second):
		t.Fatal("ByteCursor fillBuffer hung on a slow split reader")
	}
}

func TestByteCursorTreatsEOFWithDataAsCleanEnd(t *testing.T) {
	cur := strcursor.NewByteCursor(&dataThenErrReader{
		data: []byte("<root/>"),
		err:  io.EOF,
	})

	require.Equal(t, "<root/>", cur.PeekString(7))
	require.NoError(t, cur.Advance(7))
	require.True(t, cur.Done())
	require.NoError(t, cur.Err(), "io.EOF must not be reported as an error")
}

// cursorMethod names the advancing call a positionOp makes.
type cursorMethod string

const (
	methodAdvance       cursorMethod = "Advance"
	methodAdvanceFast   cursorMethod = "AdvanceFast"
	methodConsume       cursorMethod = "Consume"
	methodConsumeString cursorMethod = "ConsumeString"
	methodScanCharData  cursorMethod = "ScanCharData"
)

// positionOp is one advancing call made on a cursor, followed by the position
// the cursor must report afterwards.
type positionOp struct {
	method cursorMethod
	n      int    // byte count for Advance and AdvanceFast
	s      string // prefix for Consume and ConsumeString
	ok     bool   // expected result of Consume and ConsumeString
	line   int    // LineNumber() after the call
	col    int    // Column() after the call
	text   string // Line() after the call
}

// applyPositionOp makes one advancing call on cur and checks the position that
// follows it. ScanCharData scans a character-data run and advances over it
// with AdvanceFast, as the parser does.
func applyPositionOp(t *testing.T, cur strcursor.Cursor, op positionOp) {
	t.Helper()
	switch op.method {
	case methodAdvance:
		require.NoError(t, cur.Advance(op.n))
	case methodAdvanceFast:
		require.NoError(t, cur.AdvanceFast(op.n))
	case methodConsume:
		require.Equal(t, op.ok, cur.Consume([]byte(op.s)))
	case methodConsumeString:
		require.Equal(t, op.ok, cur.ConsumeString(op.s))
	case methodScanCharData:
		// A scan covers only the buffered bytes, so scan until the run ends.
		var buf bytes.Buffer
		for n := cur.ScanCharDataInto(&buf, 0); n > 0; n = cur.ScanCharDataInto(&buf, 0) {
			require.NoError(t, cur.AdvanceFast(n))
		}
	default:
		t.Fatalf("unknown cursor method %q", op.method)
	}
	require.Equal(t, op.line, cur.LineNumber(), "line after %s", op.method)
	require.Equal(t, op.col, cur.Column(), "column after %s", op.method)
	require.Equal(t, op.text, cur.Line(), "line text after %s", op.method)
}

// TestCursorPosition checks the line, column, and line text after every
// advancing method. The column is 1 plus the number of bytes since the last
// LF, so a multi-byte character counts once per byte, a tab and a CR count as
// one column each, and only an LF starts a new line. ByteCursor (DTD
// parameter-entity text, the external subset, and the bytes before the
// encoding switch) and UTF8Cursor (the document after the switch) must agree,
// so the same cases run against both.
func TestCursorPosition(t *testing.T) {
	cases := []struct {
		name  string
		input string
		ops   []positionOp
	}{
		{
			name:  "advance across one newline",
			input: "ab\ncd",
			ops: []positionOp{
				{method: methodAdvance, n: 2, line: 1, col: 3, text: "ab"},
				{method: methodAdvance, n: 1, line: 2, col: 1, text: ""},
				{method: methodAdvance, n: 2, line: 2, col: 3, text: "cd"},
			},
		},
		{
			name:  "advance across several newlines at once",
			input: "a\nb\nc\nde",
			ops: []positionOp{
				{method: methodAdvance, n: 7, line: 4, col: 2, text: "d"},
			},
		},
		{
			name:  "advance ending on a newline",
			input: "ab\ncd",
			ops: []positionOp{
				{method: methodAdvance, n: 3, line: 2, col: 1, text: ""},
			},
		},
		{
			name:  "lone CR is one column",
			input: "a\rb",
			ops: []positionOp{
				{method: methodAdvance, n: 3, line: 1, col: 4, text: "a\rb"},
			},
		},
		{
			name:  "CRLF ends the line at the LF",
			input: "a\r\nb",
			ops: []positionOp{
				{method: methodAdvance, n: 2, line: 1, col: 3, text: "a\r"},
				{method: methodAdvance, n: 1, line: 2, col: 1, text: ""},
				{method: methodAdvance, n: 1, line: 2, col: 2, text: "b"},
			},
		},
		{
			name:  "CRLF in one advance",
			input: "a\r\nb",
			ops: []positionOp{
				{method: methodAdvance, n: 4, line: 2, col: 2, text: "b"},
			},
		},
		{
			name:  "tab is one column",
			input: "\t\tx",
			ops: []positionOp{
				{method: methodAdvance, n: 2, line: 1, col: 3, text: "\t\t"},
			},
		},
		{
			name:  "multi-byte characters count their bytes",
			input: "é€x",
			ops: []positionOp{
				{method: methodAdvance, n: 5, line: 1, col: 6, text: "é€"},
				{method: methodAdvance, n: 1, line: 1, col: 7, text: "é€x"},
			},
		},
		{
			name:  "advance fast across newlines",
			input: "ab\ncd\nef",
			ops: []positionOp{
				{method: methodAdvanceFast, n: 2, line: 1, col: 3, text: "ab"},
				{method: methodAdvanceFast, n: 5, line: 3, col: 2, text: "e"},
			},
		},
		{
			name:  "consume advances the column",
			input: "<!ELEMENT r ANY>]",
			ops: []positionOp{
				{method: methodConsume, s: "<!ELEMENT", ok: true, line: 1, col: 10, text: "<!ELEMENT"},
				{method: methodConsumeString, s: " r ANY>", ok: true, line: 1, col: 17, text: "<!ELEMENT r ANY>"},
			},
		},
		{
			name:  "consume across a newline",
			input: "<a\n b>",
			ops: []positionOp{
				{method: methodConsumeString, s: "<a\n ", ok: true, line: 2, col: 2, text: " "},
				{method: methodConsume, s: "b>", ok: true, line: 2, col: 4, text: " b>"},
			},
		},
		{
			name:  "consume across CRLF and tab",
			input: "a\r\n\tb",
			ops: []positionOp{
				{method: methodConsume, s: "a\r\n\t", ok: true, line: 2, col: 2, text: "\t"},
			},
		},
		{
			name:  "consume multi-byte characters",
			input: "été<",
			ops: []positionOp{
				{method: methodConsumeString, s: "été", ok: true, line: 1, col: 6, text: "été"},
			},
		},
		{
			name:  "failed consume keeps the position",
			input: "abcd",
			ops: []positionOp{
				{method: methodAdvance, n: 1, line: 1, col: 2, text: "a"},
				{method: methodConsume, s: "x", ok: false, line: 1, col: 2, text: "a"},
				{method: methodConsumeString, s: "bx", ok: false, line: 1, col: 2, text: "a"},
			},
		},
		{
			name:  "scanned character data with CR, LF, and CRLF",
			input: "x\r\ny\rz\n\té<",
			ops: []positionOp{
				{method: methodScanCharData, line: 3, col: 4, text: "\té"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("ByteCursor", func(t *testing.T) {
				cur := strcursor.NewByteCursor(strings.NewReader(tc.input))
				for _, op := range tc.ops {
					applyPositionOp(t, cur, op)
				}
			})
			// A two-byte buffer makes ByteCursor refill and compact while the
			// line text spans several fills.
			t.Run("ByteCursor small buffer", func(t *testing.T) {
				cur := strcursor.NewByteCursor(strings.NewReader(tc.input), 2)
				for _, op := range tc.ops {
					applyPositionOp(t, cur, op)
				}
			})
			t.Run("UTF8Cursor", func(t *testing.T) {
				cur := strcursor.NewUTF8Cursor(strings.NewReader(tc.input))
				for _, op := range tc.ops {
					applyPositionOp(t, cur, op)
				}
			})
		})
	}
}

// startCursor is a cursor whose first position can be set.
type startCursor interface {
	strcursor.Cursor
	StartAt(strcursor.Position)
}

// TestCursorStartAt checks that a cursor started at a position counts on from
// it: the column and line text continue the given line until an LF ends it.
func TestCursorStartAt(t *testing.T) {
	const decl = `<?xml version="1.0"?>`
	cases := []struct {
		name  string
		start strcursor.Position
		input string
		ops   []positionOp
	}{
		{
			name:  "after a declaration",
			start: strcursor.Position{Line: 1, Column: len(decl) + 1, LineText: decl},
			input: "<r>é\ncd",
			ops: []positionOp{
				{method: methodAdvance, n: 5, line: 1, col: 27, text: decl + "<r>é"},
				{method: methodAdvance, n: 1, line: 2, col: 1, text: ""},
				{method: methodAdvance, n: 2, line: 2, col: 3, text: "cd"},
			},
		},
		{
			name:  "on a later line",
			start: strcursor.Position{Line: 3, Column: 5, LineText: " ab "},
			input: "x>\ny",
			ops: []positionOp{
				{method: methodConsumeString, s: "x>", ok: true, line: 3, col: 7, text: " ab x>"},
				{method: methodAdvanceFast, n: 2, line: 4, col: 2, text: "y"},
			},
		},
		{
			name:  "at the start",
			start: strcursor.Position{Line: 1, Column: 1},
			input: "ab",
			ops: []positionOp{
				{method: methodAdvance, n: 2, line: 1, col: 3, text: "ab"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cursors := map[string]startCursor{
				"ByteCursor": strcursor.NewByteCursor(strings.NewReader(tc.input)),
				"UTF8Cursor": strcursor.NewUTF8Cursor(strings.NewReader(tc.input)),
			}
			for name, cur := range cursors {
				t.Run(name, func(t *testing.T) {
					cur.StartAt(tc.start)
					require.Equal(t, tc.start, strcursor.PositionOf(cur))
					for _, op := range tc.ops {
						applyPositionOp(t, cur, op)
					}
				})
			}
		})
	}
}

// TestUTF8CursorContinuesByteCursor reads an XML declaration on a ByteCursor,
// then the rest of the input on a UTF8Cursor started at the ByteCursor's
// position, as the parser does when it switches encoding.
func TestUTF8CursorContinuesByteCursor(t *testing.T) {
	const decl = "<?xml version=\"1.0\"\n encoding=\"UTF-8\"?>"
	bc := strcursor.NewByteCursor(strings.NewReader(decl + "<r>&x;"))
	require.NoError(t, bc.Advance(len(decl)))
	cur := strcursor.NewUTF8Cursor(bc)
	cur.StartAt(strcursor.PositionOf(bc))
	applyPositionOp(t, cur, positionOp{method: methodAdvance, n: 4, line: 2, col: 24, text: ` encoding="UTF-8"?><r>&`})
}
