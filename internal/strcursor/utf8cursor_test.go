package strcursor_test

import (
	"bytes"
	"io"
	"math/rand/v2"
	"strings"
	"testing"
	"testing/iotest"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/lestrrat-go/helium/internal/strcursor"
	"github.com/lestrrat-go/helium/internal/xmlchar"
	"github.com/stretchr/testify/require"
)

func TestUTF8CursorZeroProgressReaderDoesNotHang(t *testing.T) {
	cur := strcursor.NewUTF8Cursor(zeroProgressReader{})

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
		t.Fatal("UTF8Cursor fillBuffer hung on a zero-progress reader")
	}
}

func TestUTF8CursorSlowSplitReaderMakesProgress(t *testing.T) {
	cur := strcursor.NewUTF8Cursor(&slowSplitReader{data: []byte("héllo")})

	type result struct {
		peeked string
		err    error
	}
	done := make(chan result, 1)
	go func() {
		s := cur.PeekString(len("héllo"))
		done <- result{peeked: s, err: cur.Err()}
	}()

	select {
	case res := <-done:
		require.Equal(t, "héllo", res.peeked, "a slow reader that emits (0, nil) between bytes must still be consumed")
		require.NoError(t, res.err, "a progressing reader must not surface io.ErrNoProgress")
	case <-time.After(5 * time.Second):
		t.Fatal("UTF8Cursor fillBuffer hung on a slow split reader")
	}
}

type chunkedReader struct {
	data  []byte
	chunk int
}

func (r *chunkedReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	n := min(r.chunk, len(r.data), len(p))
	copy(p, r.data[:n])
	r.data = r.data[n:]
	return n, nil
}

func TestUTF8CursorScanCharDataSliceSpansBufferEdge(t *testing.T) {
	cur := strcursor.NewUTF8Cursor(&chunkedReader{
		data:  []byte("    <"),
		chunk: 2,
	})

	data, n, valid := cur.ScanCharDataSlice(nil, 0, false)
	require.True(t, valid)
	require.Equal(t, 4, n)
	require.Equal(t, "    ", string(data))
}

func TestUTF8CursorScanCharDataSliceConsumesCRLFAcrossBufferEdge(t *testing.T) {
	cur := strcursor.NewUTF8Cursor(&chunkedReader{
		data:  []byte("\r\n<"),
		chunk: 1,
	})

	data, n, valid := cur.ScanCharDataSlice(nil, 0, false)
	require.True(t, valid)
	require.Equal(t, 2, n)
	require.Equal(t, "\n", string(data))
}

func TestUTF8CursorScanCharDataSlicePreservesWhitespaceRunAcrossBufferEdge(t *testing.T) {
	cur := strcursor.NewUTF8Cursor(&chunkedReader{
		data:  []byte(strings.Repeat(" ", 7) + "<"),
		chunk: 3,
	})

	data, n, valid := cur.ScanCharDataSlice(nil, 0, false)
	require.True(t, valid)
	require.Equal(t, 7, n)
	require.Equal(t, strings.Repeat(" ", 7), string(data))
}

func TestUTF8CursorScanCharDataSliceReturnsOverBudgetRuneWhole(t *testing.T) {
	// A 3-byte rune with a 1-byte budget: the scan must return the rune whole
	// (never a partial rune) so the caller makes progress without emitting
	// invalid UTF-8.
	cur := strcursor.NewUTF8Cursor(&chunkedReader{
		data:  []byte("世<"),
		chunk: 1,
	})

	data, n, valid := cur.ScanCharDataSlice(nil, 1, false)
	require.True(t, valid)
	require.Equal(t, 3, n, "a lone rune wider than maxBytes is returned whole")
	require.Equal(t, "世", string(data))
}

// charDataPieces are the fragments the char-data differential test builds its
// inputs from: delimiters, CR/LF, every class of literal and non-literal
// character in both XML versions, and malformed UTF-8.
var charDataPieces = []string{
	"a", "Z", " ", "~", "\t", "\n", "\r", "\r\n", "<", "&", "]", "]]", "]]>",
	"\x00", "\x01", "\x08", "\x0b", "\x1f", "\x7f",
	"\u0080", "\u0084", "\u0085", "\u0086", "\u009f", " ", "é", "日", " ",
	"퟿", "", "�", "￾", "￿", "\U00010000", "\U0001F600", "\U0010FFFF",
	"\xed\xa0\x80", "\xc0\x80", "\xe2\x82", "\xf0\x9f\x98", "\x80", "\xbf", "\xc2", "\xff",
	"0123456789abcdef", "plain text run",
}

// TestUTF8CursorScanCharDataSliceValidates checks ScanCharDataSlice against
// referenceScanCharData, a byte-at-a-time model of the scan followed by a
// separate literal-character check of the scanned run, for random inputs in
// both XML versions, under byte budgets and with input split across reads.
func TestUTF8CursorScanCharDataSliceValidates(t *testing.T) {
	t.Parallel()

	t.Run("each piece at every offset of a 16-byte block", func(t *testing.T) {
		t.Parallel()

		for _, piece := range charDataPieces {
			for off := range 33 {
				input := []byte(strings.Repeat("x", off) + piece + "0123456789abcdefghij<")
				for _, xml11 := range []bool{false, true} {
					checkScanCharData(t, input, 0, 0, xml11)
					checkScanCharData(t, input, 0, 1, xml11)
					checkScanCharData(t, input, 0, 3, xml11)
					checkScanCharData(t, input, off, 0, xml11)
					checkScanCharData(t, input, off+1, 0, xml11)
				}
			}
		}
	})

	t.Run("each piece at the cursor buffer edge", func(t *testing.T) {
		t.Parallel()

		// NewUTF8Cursor starts with an 8 KiB buffer; place each piece so it
		// starts just before, on, or just after the first refill.
		for _, piece := range charDataPieces {
			for off := 8192 - 5; off <= 8192+2; off++ {
				input := []byte(strings.Repeat("x", off) + piece + "tail<")
				for _, xml11 := range []bool{false, true} {
					checkScanCharData(t, input, 0, 0, xml11)
					checkScanCharData(t, input, 0, 4096, xml11)
					checkScanCharData(t, input, off, 0, xml11)
				}
			}
		}
	})

	t.Run("random inputs", func(t *testing.T) {
		t.Parallel()

		rng := rand.New(rand.NewPCG(3, 4))
		var buf []byte
		for range 20000 {
			buf = buf[:0]
			for range rng.IntN(24) {
				if rng.IntN(5) == 0 {
					buf = append(buf, byte(rng.IntN(0x100)))
					continue
				}
				buf = append(buf, charDataPieces[rng.IntN(len(charDataPieces))]...)
			}
			budget := 0
			if rng.IntN(2) == 0 {
				budget = 1 + rng.IntN(40)
			}
			chunk := 0
			if rng.IntN(2) == 0 {
				chunk = 1 + rng.IntN(9)
			}
			for _, xml11 := range []bool{false, true} {
				checkScanCharData(t, buf, budget, chunk, xml11)
			}
		}
	})
}

// FuzzScanCharDataSlice runs the TestUTF8CursorScanCharDataSliceValidates
// comparison on fuzzer-chosen input, budget, and read size.
func FuzzScanCharDataSlice(f *testing.F) {
	for _, piece := range charDataPieces {
		f.Add([]byte("0123456789"+piece+"abcdef<"), uint8(0), uint8(0), true)
	}
	f.Fuzz(func(t *testing.T, input []byte, budget, chunk uint8, xml11 bool) {
		checkScanCharData(t, input, int(budget), int(chunk%16), xml11)
	})
}

// checkScanCharData scans input with ScanCharDataSlice and compares the run,
// the bytes consumed, and the validity with referenceScanCharData. chunk > 0
// feeds the cursor chunk bytes per read; budget is the maxBytes argument.
//
// A call may stop where the buffered input ends (after a CR, ']' or multi-byte
// character that consumed the last buffered byte), leaving the rest of the run
// to the next call. Without a budget the test therefore calls
// ScanCharDataSlice until it returns nothing, advancing past each run as the
// parser does, and compares the concatenated runs. A budget applies per call,
// so with one the test compares a single call over input that the first read
// buffers whole (chunk is ignored).
func checkScanCharData(t *testing.T, input []byte, budget, chunk int, xml11 bool) {
	t.Helper()

	var r io.Reader = bytes.NewReader(input)
	if chunk > 0 && budget == 0 {
		r = &chunkedReader{data: input, chunk: chunk}
	}
	cur := strcursor.NewUTF8Cursor(r)

	var got []byte
	consumed := 0
	valid := true
	for {
		run, n, ok := cur.ScanCharDataSlice(nil, budget, xml11)
		if n <= 0 {
			break
		}
		got = append(got, run...)
		consumed += n
		valid = valid && ok
		require.NoError(t, cur.AdvanceFast(n))
		if budget > 0 {
			break
		}
	}

	want, wantConsumed, wantValid := referenceScanCharData(input, budget, xml11)
	if consumed != wantConsumed || !bytes.Equal(got, want) || valid != wantValid {
		require.Failf(t, "ScanCharDataSlice disagrees with the reference",
			"input %q (%d bytes), budget %d, chunk %d, xml11 %t: consumed %d want %d, valid %t want %t, run tail %q want %q",
			tail(input), len(input), budget, chunk, xml11, consumed, wantConsumed, valid, wantValid, tail(got), tail(want))
	}
}

// tail returns at most the last 48 bytes of b, keeping failure messages short
// for inputs that cross the 8 KiB cursor buffer.
func tail(b []byte) []byte {
	return b[max(0, len(b)-48):]
}

// referenceScanCharData models the char-data scan one character at a time:
// the run ends at '<', '&', "]]>", invalid UTF-8, a character outside the XML
// 1.0 Char production, or the byte budget (rounded up to a whole character,
// and a CR LF pair is consumed together). CR and CR LF become LF. Validity is
// checked afterwards over the whole run, as a separate pass: XML 1.0 accepts
// every Char, XML 1.1 rejects RestrictedChar.
func referenceScanCharData(input []byte, budget int, xml11 bool) ([]byte, int, bool) {
	var run []byte
	off := 0
	for off < len(input) {
		if budget > 0 && off >= budget {
			break
		}
		b := input[off]
		if b == '<' || b == '&' {
			break
		}
		if b == ']' && off+2 < len(input) && input[off+1] == ']' && input[off+2] == '>' {
			break
		}
		if b == '\r' {
			run = append(run, '\n')
			off++
			if off < len(input) && input[off] == '\n' {
				off++
			}
			continue
		}
		r, w := utf8.DecodeRune(input[off:])
		if r == utf8.RuneError && w == 1 {
			break
		}
		if !xmlchar.IsChar(r) {
			break
		}
		if budget > 0 && off > 0 && off+w > budget {
			break
		}
		run = append(run, input[off:off+w]...)
		off += w
	}

	valid := true
	for rest := run; len(rest) > 0; {
		r, w := utf8.DecodeRune(rest)
		if xml11 && (!xmlchar.IsXML11Char(r) || xmlchar.IsXML11RestrictedChar(r)) {
			valid = false
		}
		rest = rest[w:]
	}
	return run, off, valid
}

func TestUTF8CursorScanQNameBytesASCIIUnprefixed(t *testing.T) {
	cur := strcursor.NewUTF8Cursor(strings.NewReader("root attr"))

	name, colon, ok := cur.ScanQNameBytes()
	require.True(t, ok)
	require.Equal(t, "root", string(name))
	require.Equal(t, -1, colon)
}

func TestUTF8CursorScanQNameBytesASCIIPrefixed(t *testing.T) {
	cur := strcursor.NewUTF8Cursor(strings.NewReader("x:item attr"))

	name, colon, ok := cur.ScanQNameBytes()
	require.True(t, ok)
	require.Equal(t, "x:item", string(name))
	require.Equal(t, 1, colon)
}

func TestUTF8CursorScanQNameBytesSpansBufferEdge(t *testing.T) {
	cur := strcursor.NewUTF8Cursor(&chunkedReader{
		data:  []byte("x:item attr"),
		chunk: 2,
	})

	name, colon, ok := cur.ScanQNameBytes()
	require.True(t, ok)
	require.Equal(t, "x:item", string(name))
	require.Equal(t, 1, colon)
}

func TestUTF8CursorScanQNameBytesRejectsSecondColon(t *testing.T) {
	cur := strcursor.NewUTF8Cursor(strings.NewReader("a:b:c"))

	name, colon, ok := cur.ScanQNameBytes()
	require.False(t, ok)
	require.Nil(t, name)
	require.Equal(t, -1, colon)
	require.Equal(t, byte('a'), cur.Peek())
}

func TestRuneCursorReadShortBufferBufferedRune(t *testing.T) {
	cur := strcursor.NewRuneCursor(strings.NewReader("é"))
	// Buffer the multibyte rune in the ring.
	require.Equal(t, 'é', cur.Peek())

	// A 1-byte destination cannot hold the 2-byte rune. Read must not panic
	// and must not corrupt or drop the buffered rune.
	first := make([]byte, 1)
	n, err := cur.Read(first)
	require.NoError(t, err)
	require.Zero(t, n, "no full rune fits in a 1-byte buffer")

	// The rune must still be deliverable on a subsequent read.
	rest := make([]byte, 8)
	var got []byte
	for {
		m, rerr := cur.Read(rest)
		got = append(got, rest[:m]...)
		if rerr == io.EOF {
			break
		}
		require.NoError(t, rerr)
		if m == 0 {
			break
		}
	}
	require.Equal(t, "é", string(got), "rune delivered intact across reads")
}

func TestRuneCursorReadShortBufferPartialRuneFit(t *testing.T) {
	cur := strcursor.NewRuneCursor(strings.NewReader("aé"))
	// Buffer both runes in the ring.
	require.Equal(t, 'a', cur.Peek())
	require.Equal(t, 'é', cur.PeekN(2))

	// A 2-byte buffer fits 'a' (1 byte) but not the following 2-byte 'é'.
	// Read must emit only 'a' as a short read with no EOF, leaving 'é'
	// buffered, with no bytes reordered from the underlying reader.
	buf := make([]byte, 2)
	n, err := cur.Read(buf)
	require.NoError(t, err)
	require.Equal(t, 1, n, "only the 1-byte rune fits")
	require.Equal(t, "a", string(buf[:n]))

	// The buffered 'é' is delivered intact on the next read.
	rest := make([]byte, 8)
	m, err := cur.Read(rest)
	require.Equal(t, "é", string(rest[:m]))
	if err != nil {
		require.Equal(t, io.EOF, err)
	}
}

// TestUTF8CursorAdvanceFastLineColumn checks that AdvanceFast leaves the same
// line number and column as Advance, which walks the run byte by byte, for
// runs of 0-40 bytes with no newline, one newline at every position, two
// newlines at every pair of positions, and nothing but newlines, starting at
// the first column, mid-line, and on a later line.
func TestUTF8CursorAdvanceFastLineColumn(t *testing.T) {
	t.Parallel()

	for n := range 41 {
		runs := [][]byte{bytes.Repeat([]byte("x"), n), bytes.Repeat([]byte("\n"), n)}
		for p := range n {
			run := bytes.Repeat([]byte("x"), n)
			run[p] = '\n'
			runs = append(runs, run)
			for q := p + 1; q < n; q++ {
				pair := bytes.Repeat([]byte("x"), n)
				pair[p] = '\n'
				pair[q] = '\n'
				runs = append(runs, pair)
			}
		}
		for _, run := range runs {
			for _, prefix := range []string{"", "ab", "a\nbcd"} {
				checkAdvanceFast(t, prefix, run)
			}
		}
	}
}

// checkAdvanceFast advances one cursor over prefix and run with AdvanceFast and
// another with Advance, and compares where they report the position.
func checkAdvanceFast(t *testing.T, prefix string, run []byte) {
	t.Helper()

	input := append(append([]byte(prefix), run...), "<tail"...)
	fast := strcursor.NewUTF8Cursor(bytes.NewReader(input))
	slow := strcursor.NewUTF8Cursor(bytes.NewReader(input))
	require.NoError(t, fast.Advance(len(prefix)))
	require.NoError(t, slow.Advance(len(prefix)))
	require.NoError(t, fast.AdvanceFast(len(run)))
	require.NoError(t, slow.Advance(len(run)))
	if fast.LineNumber() != slow.LineNumber() || fast.Column() != slow.Column() {
		require.Failf(t, "AdvanceFast disagrees with Advance",
			"prefix %q run %q: line %d column %d, want line %d column %d",
			prefix, run, fast.LineNumber(), fast.Column(), slow.LineNumber(), slow.Column())
	}
}

// TestUTF8CursorAdvanceNoNewline checks that AdvanceNoNewline over a run with
// no newline leaves the same position, line number, and column as Advance, for
// runs of 0-40 bytes with and without multi-byte characters, after a prefix on
// the first and on a later line, from a fully buffered input and from one
// read a byte at a time (where the run is not yet buffered).
func TestUTF8CursorAdvanceNoNewline(t *testing.T) {
	t.Parallel()

	for n := range 41 {
		runs := []string{strings.Repeat("x", n), strings.Repeat("é", n/2)}
		for _, run := range runs {
			for _, prefix := range []string{"", "ab", "a\nbcd"} {
				for _, oneByte := range []bool{false, true} {
					checkAdvanceNoNewline(t, prefix, run, oneByte)
				}
			}
		}
	}
}

func checkAdvanceNoNewline(t *testing.T, prefix, run string, oneByte bool) {
	t.Helper()

	input := prefix + run + "<tail"
	fast := newCursorOver(input, oneByte)
	slow := newCursorOver(input, oneByte)
	require.NoError(t, fast.Advance(len(prefix)))
	require.NoError(t, slow.Advance(len(prefix)))
	require.NoError(t, fast.AdvanceNoNewline(len(run)))
	require.NoError(t, slow.Advance(len(run)))
	require.Equal(t, slow.Peek(), fast.Peek(), "prefix %q run %q", prefix, run)
	require.Equal(t, slow.LineNumber(), fast.LineNumber(), "prefix %q run %q", prefix, run)
	require.Equal(t, slow.Column(), fast.Column(), "prefix %q run %q", prefix, run)
}

// newCursorOver returns a cursor over input, read a byte at a time when
// oneByte is set.
func newCursorOver(input string, oneByte bool) *strcursor.UTF8Cursor {
	if oneByte {
		return strcursor.NewUTF8Cursor(iotest.OneByteReader(strings.NewReader(input)))
	}
	return strcursor.NewUTF8Cursor(strings.NewReader(input))
}

// attrValuePieces extends charDataPieces with both quote characters, so the
// attribute-value differential test sees values that end, and values that
// hold the other quote.
var attrValuePieces = append([]string{`"`, `'`, `a"b`, `a'b`}, charDataPieces...)

// TestUTF8CursorScanSimpleAttrValue checks ScanSimpleAttrValue against
// referenceScanSimpleAttrValue, a byte-at-a-time model, for both quote
// characters, under byte budgets on both sides of the value length, and with
// input split across reads.
func TestUTF8CursorScanSimpleAttrValue(t *testing.T) {
	t.Parallel()

	t.Run("each piece at every offset of a 16-byte block", func(t *testing.T) {
		t.Parallel()

		for _, piece := range attrValuePieces {
			for off := range 34 {
				for _, quote := range []byte{'"', '\''} {
					input := []byte(strings.Repeat("x", off) + piece + "0123456789abcdefghij" + string(quote) + " b")
					for _, budget := range []int{0, off - 1, off, off + 1, off + 2, off + 20, off + 40} {
						checkScanSimpleAttrValue(t, input, quote, max(budget, 0), 0)
					}
					checkScanSimpleAttrValue(t, input, quote, 0, 1)
					checkScanSimpleAttrValue(t, input, quote, 0, 3)
					checkScanSimpleAttrValue(t, input, quote, 0, 17)
				}
			}
		}
	})

	t.Run("each piece at the cursor buffer edge", func(t *testing.T) {
		t.Parallel()

		for _, piece := range attrValuePieces {
			for off := 8192 - 20; off <= 8192+2; off++ {
				input := []byte(strings.Repeat("x", off) + piece + `tail"`)
				checkScanSimpleAttrValue(t, input, '"', 0, 0)
				checkScanSimpleAttrValue(t, input, '"', 0, 4096)
				checkScanSimpleAttrValue(t, input, '"', off, 0)
				checkScanSimpleAttrValue(t, input, '"', off+8, 0)
			}
		}
	})

	t.Run("random inputs", func(t *testing.T) {
		t.Parallel()

		rng := rand.New(rand.NewPCG(5, 6))
		var buf []byte
		for range 20000 {
			buf = buf[:0]
			for range rng.IntN(24) {
				if rng.IntN(5) == 0 {
					buf = append(buf, byte(rng.IntN(0x100)))
					continue
				}
				buf = append(buf, attrValuePieces[rng.IntN(len(attrValuePieces))]...)
			}
			if rng.IntN(4) != 0 {
				buf = append(buf, '"')
			}
			budget := 0
			if rng.IntN(2) == 0 {
				budget = 1 + rng.IntN(60)
			}
			chunk := 0
			if rng.IntN(2) == 0 {
				chunk = 1 + rng.IntN(9)
			}
			for _, quote := range []byte{'"', '\''} {
				checkScanSimpleAttrValue(t, buf, quote, budget, chunk)
			}
		}
	})
}

// FuzzScanSimpleAttrValue runs the TestUTF8CursorScanSimpleAttrValue
// comparison on fuzzer-chosen input, quote, budget, and read size.
func FuzzScanSimpleAttrValue(f *testing.F) {
	for _, piece := range attrValuePieces {
		f.Add([]byte("0123456789"+piece+`abcdef"`), false, uint8(0), uint8(0))
	}
	f.Fuzz(func(t *testing.T, input []byte, apostrophe bool, budget, chunk uint8) {
		quote := byte('"')
		if apostrophe {
			quote = '\''
		}
		checkScanSimpleAttrValue(t, input, quote, int(budget), int(chunk%16))
	})
}

// checkScanSimpleAttrValue scans input with ScanSimpleAttrValue and compares
// the value and byte count with referenceScanSimpleAttrValue. chunk > 0 feeds
// the cursor chunk bytes per read; budget is the maxBytes argument.
func checkScanSimpleAttrValue(t *testing.T, input []byte, quote byte, budget, chunk int) {
	t.Helper()

	var r io.Reader = bytes.NewReader(input)
	if chunk > 0 {
		r = &chunkedReader{data: input, chunk: chunk}
	}
	cur := strcursor.NewUTF8Cursor(r)
	got, n := cur.ScanSimpleAttrValue(quote, budget)
	want, wantN := referenceScanSimpleAttrValue(input, quote, budget)
	if got != want || n != wantN {
		require.Failf(t, "ScanSimpleAttrValue disagrees with the reference",
			"input %q (%d bytes), quote %q, budget %d, chunk %d: value tail %q n %d, want tail %q n %d",
			tail(input), len(input), quote, budget, chunk, tail([]byte(got)), n, tail([]byte(want)), wantN)
	}
}

// referenceScanSimpleAttrValue models the simple attribute-value scan one
// character at a time: the value ends at quote; '&', '<', any byte below 0x20,
// invalid UTF-8, a character outside the XML 1.0 Char production, running out
// of input, and more than budget bytes (when budget > 0) all reject the value.
func referenceScanSimpleAttrValue(input []byte, quote byte, budget int) (string, int) {
	off := 0
	for {
		if budget > 0 && off > budget {
			return "", 0
		}
		if off >= len(input) {
			return "", 0
		}
		b := input[off]
		if b == quote {
			return string(input[:off]), off
		}
		if b == '&' || b == '<' || b < 0x20 {
			return "", 0
		}
		if b < utf8.RuneSelf {
			off++
			continue
		}
		r, w := utf8.DecodeRune(input[off:])
		if r == utf8.RuneError && w == 1 {
			return "", 0
		}
		if !xmlchar.IsChar(r) {
			return "", 0
		}
		off += w
	}
}

// TestUTF8CursorScanSimpleAttrValueCharacters scans a value holding each code
// point in turn. The scan must accept exactly the characters at or above U+0020
// that are XML 1.0 Chars, other than the quote, '&' and '<'; every one of them
// is an XML 1.0 literal character, which is why the parser does not recheck a
// value this scan accepted in an XML 1.0 document.
func TestUTF8CursorScanSimpleAttrValueCharacters(t *testing.T) {
	t.Parallel()

	var input []byte
	for r := rune(0); r <= unicode.MaxRune; r++ {
		if r == '"' || utf8.RuneLen(r) < 0 {
			continue
		}
		input = append(input, 'x')
		input = utf8.AppendRune(input, r)
		input = append(input, '"')
	}

	cur := strcursor.NewUTF8Cursor(bytes.NewReader(input))
	for r := rune(0); r <= unicode.MaxRune; r++ {
		if r == '"' || utf8.RuneLen(r) < 0 {
			continue
		}
		width := 1 + utf8.RuneLen(r)
		v, n := cur.ScanSimpleAttrValue('"', 0)
		accept := r >= 0x20 && r != '&' && r != '<' && xmlchar.IsChar(r)
		if (n > 0) != accept {
			t.Fatalf("U+%04X: scan accepted %t, want %t", r, n > 0, accept)
		}
		if n > 0 {
			if n != width || v != "x"+string(r) {
				t.Fatalf("U+%04X: scanned %q (%d bytes)", r, v, n)
			}
			if !xmlchar.IsLiteralChar(r, false) {
				t.Fatalf("U+%04X: accepted but not an XML 1.0 literal character", r)
			}
		}
		require.NoError(t, cur.Advance(width+1))
	}
}

// TestUTF8CursorLineAcrossReads moves a cursor through inputs in random steps,
// peeking ahead by random amounts so the buffer refills at varying points, and
// checks Line against referenceLine after every step, for readers that return
// a few bytes per Read and for one that returns the whole input. The inputs
// hold short lines, a line longer than the buffer, and a line of two-byte
// characters, so the LineContextMax cut lands on both character boundaries and
// mid-character.
func TestUTF8CursorLineAcrossReads(t *testing.T) {
	t.Parallel()

	inputs := map[string]string{
		"short lines":         strings.Repeat("<a b=\"c\">text é</a>\n", 1200),
		"long ASCII line":     "<r>\n" + strings.Repeat("abcdefgh", 2000) + "\n<x/>",
		"long two-byte line":  "<r>\n" + strings.Repeat("é", 6000) + "\n" + "x" + strings.Repeat("é", 6000),
		"line ends at buffer": strings.Repeat("a", 8191) + "\n" + strings.Repeat("b", 9000),
	}
	// Text a cursor is started after (StartAt) comes first on its first line.
	prefixes := []string{"", `<?xml version="1.0"?>`, strings.Repeat("é", 700)}
	for name, input := range inputs {
		for _, prefix := range prefixes {
			for _, chunk := range []int{1, 2, 3, 7, 4093, 0} {
				rng := rand.New(rand.NewPCG(uint64(len(input)), uint64(chunk)))
				var r io.Reader = strings.NewReader(input)
				if chunk > 0 {
					r = &chunkedReader{data: []byte(input), chunk: chunk}
				}
				cur := strcursor.NewUTF8Cursor(r)
				cur.StartAt(strcursor.Position{Line: 1, Column: len(prefix) + 1, LineText: prefix})
				pos := 0
				for pos < len(input) {
					_ = cur.PeekAt(rng.IntN(64))
					step := min(1+rng.IntN(300), len(input)-pos)
					require.NoError(t, cur.Advance(step))
					pos += step
					require.Equal(t, referenceLine(prefix+input[:pos]), cur.Line(),
						"%s, %d-byte prefix, %d-byte reads, at byte %d", name, len(prefix), chunk, pos)
				}
			}
		}
	}
}

// referenceLine is what Line returns once consumed has been read: the text
// after its last LF, cut to its last LineContextMax bytes, and without the
// trailing bytes of a character the cut splits.
func referenceLine(consumed string) string {
	line := consumed[strings.LastIndexByte(consumed, '\n')+1:]
	if len(line) < strcursor.LineContextMax {
		return line
	}
	line = line[len(line)-strcursor.LineContextMax:]
	for i := 0; i < utf8.UTFMax-1 && line != "" && !utf8.RuneStart(line[0]); i++ {
		line = line[1:]
	}
	return line
}
