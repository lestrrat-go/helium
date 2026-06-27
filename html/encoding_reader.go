package html

import (
	"bytes"
	"io"
	"unicode/utf8"
)

// Encoding names reported for non-UTF-8 HTML input. ISO-8859-1 is selected only
// by an explicit charset declaration; Windows-1252 is the auto-detected default
// for an undeclared non-UTF-8 stream (matching libxml2's HTML behavior).
const (
	encISO88591    = "ISO-8859-1"
	encWindows1252 = "Windows-1252"
)

// errReader always returns its stored error (and no bytes). It is used to
// re-deliver a non-EOF read error that arrived together with peeked bytes
// during charset detection, so the error is not lost once the buffered bytes
// drain.
type errReader struct{ err error }

func (e *errReader) Read([]byte) (int, error) { return 0, e.err }

// newlineNormReader normalizes line endings in a stream: \r\n → \n, standalone \r → \n.
type newlineNormReader struct {
	r       io.Reader
	pending bool // true if the last byte read was \r (awaiting possible \n)
}

func (nr *newlineNormReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	// Read raw bytes into a scratch area. We read into p directly since the
	// output is never larger than the input (only equal or smaller).
	n, err := nr.r.Read(p)
	if n == 0 {
		return 0, err
	}

	raw := make([]byte, n)
	copy(raw, p[:n])

	j := 0
	for i := range n {
		b := raw[i]
		if nr.pending {
			nr.pending = false
			if b == '\n' {
				// \r\n pair: the \r was already emitted as \n, skip the \n
				continue
			}
		}
		if b == '\r' {
			nr.pending = true
			p[j] = '\n'
			j++
		} else {
			p[j] = b
			j++
		}
	}

	if j == 0 && err == nil {
		// All bytes were consumed (e.g., a single \n after a pending \r).
		// We need to read more to fill p.
		return nr.Read(p)
	}

	return j, err
}

// latin1Reader converts a Latin-1/Windows-1252 byte stream to UTF-8.
type latin1Reader struct {
	r   io.Reader
	enc string // "ISO-8859-1" or "Windows-1252"

	raw    [512]byte // scratch buffer for reading from r
	out    []byte    // buffered UTF-8 output not yet delivered
	outPos int
}

func (lr *latin1Reader) Read(p []byte) (int, error) {
	// Drain any buffered output first.
	if lr.outPos < len(lr.out) {
		n := copy(p, lr.out[lr.outPos:])
		lr.outPos += n
		if lr.outPos >= len(lr.out) {
			lr.out = lr.out[:0]
			lr.outPos = 0
		}
		return n, nil
	}

	// Read raw bytes.
	n, err := lr.r.Read(lr.raw[:])
	if n == 0 {
		return 0, err
	}

	// Convert to UTF-8. Each byte can expand to up to 3 UTF-8 bytes.
	lr.out = lr.out[:0]
	lr.outPos = 0
	var encBuf [4]byte
	for _, b := range lr.raw[:n] {
		if b < 0x80 {
			lr.out = append(lr.out, b)
		} else if b <= 0x9F {
			sz := utf8.EncodeRune(encBuf[:], win1252ToUnicode[b-0x80])
			lr.out = append(lr.out, encBuf[:sz]...)
		} else {
			sz := utf8.EncodeRune(encBuf[:], rune(b))
			lr.out = append(lr.out, encBuf[:sz]...)
		}
	}

	written := copy(p, lr.out[lr.outPos:])
	lr.outPos += written
	if lr.outPos >= len(lr.out) {
		lr.out = lr.out[:0]
		lr.outPos = 0
	}

	return written, err
}

// utf8SanitizeReader replaces invalid UTF-8 byte sequences with U+FFFD.
// It tracks the line/col position of the first invalid byte for error reporting.
type utf8SanitizeReader struct {
	r io.Reader

	raw    []byte // raw bytes read from r (may contain partial UTF-8 at end)
	rawLen int
	rawPos int

	out    []byte // sanitized output ready to consume
	outPos int

	// readErr is a sticky non-EOF error returned by the underlying reader. An
	// io.Reader may return n > 0 together with a non-EOF error in a single Read;
	// surfacing it before the converted output drains would truncate the stream.
	// We remember it and surface it only once all buffered sanitized output has
	// been delivered.
	readErr error

	// Error tracking: position of first invalid byte (after newline normalization).
	hasError bool
	errLine  int
	errCol   int
	curLine  int
	curCol   int
}

func newUTF8SanitizeReader(r io.Reader) *utf8SanitizeReader {
	return &utf8SanitizeReader{
		r:       r,
		raw:     make([]byte, 4096),
		curLine: 1,
		curCol:  1,
	}
}

// EncodingError returns whether an invalid UTF-8 byte was encountered
// and the line/col of the first occurrence.
func (sr *utf8SanitizeReader) EncodingError() (bool, int, int) {
	return sr.hasError, sr.errLine, sr.errCol
}

func (sr *utf8SanitizeReader) Read(p []byte) (int, error) {
	// Drain buffered output first.
	if sr.outPos < len(sr.out) {
		n := copy(p, sr.out[sr.outPos:])
		sr.outPos += n
		if sr.outPos >= len(sr.out) {
			sr.out = sr.out[:0]
			sr.outPos = 0
		}
		return n, nil
	}

	// Buffered output is fully drained. A sticky non-EOF error recorded with an
	// earlier chunk of data surfaces here, now that all of that chunk's
	// converted bytes have been delivered — never before, or the tail of the
	// sanitized stream would be lost.
	if sr.readErr != nil {
		e := sr.readErr
		sr.readErr = nil
		return 0, e
	}

	// Read more raw data. Keep any trailing partial UTF-8 from previous read.
	leftover := sr.rawLen - sr.rawPos
	if leftover > 0 {
		copy(sr.raw, sr.raw[sr.rawPos:sr.rawLen])
	}
	sr.rawPos = 0
	sr.rawLen = leftover

	n, err := sr.r.Read(sr.raw[sr.rawLen:])
	sr.rawLen += n

	// An io.Reader may return n > 0 together with a non-EOF error. Remember the
	// error as sticky and process the bytes we did get; surface it only after
	// the converted output drains. For the partial-rune logic below, any error
	// (EOF or otherwise) means no further bytes are coming, so a truncated
	// trailing rune is genuinely invalid rather than merely incomplete.
	eof := err != nil
	if err != nil && err != io.EOF {
		sr.readErr = err
	}

	if sr.rawLen == 0 {
		if sr.readErr != nil {
			e := sr.readErr
			sr.readErr = nil
			return 0, e
		}
		return 0, err
	}

	// Process raw bytes: pass valid UTF-8 through, replace invalid with U+FFFD.
	sr.out = sr.out[:0]
	sr.outPos = 0
	i := 0
	data := sr.raw[:sr.rawLen]
	for i < len(data) {
		b := data[i]

		// Fast path: ASCII
		if b < 0x80 {
			sr.out = append(sr.out, b)
			sr.trackByte(b)
			i++
			continue
		}

		// Multi-byte UTF-8: check if we have enough bytes.
		r, size := utf8.DecodeRune(data[i:])
		if r == utf8.RuneError && size <= 1 {
			// Could be a genuine error or an incomplete sequence at end of buffer.
			// If we're at the end of the buffer and more data may come, keep it as leftover.
			if i+utf8.UTFMax > len(data) && !eof {
				// Partial sequence at end — keep as leftover for next read.
				sr.rawPos = i
				sr.rawLen = len(data)
				break
			}
			// Genuine invalid byte — replace with U+FFFD.
			if !sr.hasError {
				sr.hasError = true
				sr.errLine = sr.curLine
				sr.errCol = sr.curCol
			}
			sr.out = append(sr.out, "\xef\xbf\xbd"...) // U+FFFD in UTF-8
			sr.curCol++
			i++
		} else {
			sr.out = append(sr.out, data[i:i+size]...)
			// Track position: only \n matters (input is already newline-normalized).
			for _, cb := range data[i : i+size] {
				sr.trackByte(cb)
			}
			i += size
		}
	}

	// If we consumed everything, reset positions.
	if i >= len(data) {
		sr.rawPos = 0
		sr.rawLen = 0
	}

	if len(sr.out) == 0 {
		if sr.readErr != nil {
			e := sr.readErr
			sr.readErr = nil
			return 0, e
		}
		return 0, err
	}

	written := copy(p, sr.out[sr.outPos:])
	sr.outPos += written
	drained := sr.outPos >= len(sr.out)
	if drained {
		sr.out = sr.out[:0]
		sr.outPos = 0
	}

	// Surface an end condition only once all of this chunk's converted output
	// has been delivered. If output remains buffered, defer EOF/error until the
	// next call drains it. A sticky non-EOF error is always deferred to the next
	// call (where the drain check at the top re-surfaces it) so the caller can
	// never stop on the error with bytes still pending.
	if drained && sr.readErr == nil && err == io.EOF {
		return written, io.EOF
	}
	return written, nil
}

func (sr *utf8SanitizeReader) trackByte(b byte) {
	if b == '\n' {
		sr.curLine++
		sr.curCol = 1
	} else {
		sr.curCol++
	}
}

// deferredLatin1Reader handles undeclared-charset HTML streams whose first
// chunk is valid UTF-8 but which may contain non-UTF-8 (Latin-1/Windows-1252)
// bytes further into the document — past the 1024-byte detection window.
//
// It must match the whole-document []byte parse path (html.Parser.Parse),
// which decides the encoding for the ENTIRE document at once: if the whole
// (newline-normalized) document is valid UTF-8 it is left as UTF-8, otherwise
// the WHOLE document is reinterpreted as Latin-1/Windows-1252 — including any
// leading bytes that happened to form valid UTF-8 multibyte sequences.
//
// To replicate that decision over a stream, the reader buffers undecided raw
// bytes (emitting nothing) until either:
//   - EOF is reached with everything still valid UTF-8 → emit the buffer
//     unchanged as UTF-8; or
//   - the first genuine non-UTF-8 byte appears → reinterpret the ENTIRE
//     buffered prefix (and the remainder of the stream) as Latin-1/Windows-1252,
//     exactly as the []byte path would.
//
// deferredLatin1MaxBuffer caps how many undecided UTF-8-valid bytes the
// deferred reader buffers before it must commit to a UTF-8 interpretation. It
// is far larger than the 1024-byte charset sniff window (so a real document's
// first non-UTF-8 byte is virtually always seen first and the libxml2-quirk
// decision is preserved) yet bounded, so an endless all-valid stream cannot be
// buffered without limit.
const deferredLatin1MaxBuffer = 1 << 20 // 1 MiB

// The detected encoding name is reported lazily via detectedEncoding once the
// switch happens.
//
// Buffering is bounded: deferring until EOF would buffer an endless all-valid
// stream whole, defeating streaming and the parser's content caps (an
// unbounded-memory DoS). Once deferredLatin1MaxBuffer bytes have been seen with
// no non-UTF-8 byte the reader commits to UTF-8 and streams the rest through a
// sanitizer that replaces any later invalid byte with U+FFFD. Real (finite)
// documents settle their encoding far below this cap, so the libxml2-quirk
// Latin-1-vs-UTF-8 decision is unchanged for them; only a pathological undeclared
// stream that is valid UTF-8 past the cap and THEN turns non-UTF-8 diverges, and
// it stays well-formed (U+FFFD) rather than reinterpreting the whole document as
// Latin-1 as the in-memory []byte path would.
type deferredLatin1Reader struct {
	r io.Reader

	pending []byte // undecided raw bytes buffered while still valid UTF-8
	out     []byte // converted output ready to consume
	outPos  int

	switched      bool   // true once a non-UTF-8 byte forced Latin-1 interpretation
	committedUTF8 bool   // true once the bounded prefix settled the stream as UTF-8
	eof           bool   // true once the underlying reader has reported EOF
	enc           string // encoding name reported after switching
	encOnHit      string // encoding name to report once a non-UTF-8 byte appears

	// sanitizer wraps the remainder of the stream once the reader has committed
	// to UTF-8 after the bounded prefix. Post-commit bytes are NOT passed through
	// raw: any byte that is no longer valid UTF-8 (e.g. a late Windows-1252 byte
	// in an undeclared >1 MiB-valid stream) is replaced with U+FFFD, so invalid
	// bytes never leak into SAX/DOM. nil until the commit happens.
	sanitizer *utf8SanitizeReader

	// readErr is a sticky non-EOF error returned by the underlying reader. An
	// io.Reader may return n > 0 together with a non-EOF error in a single Read;
	// dropping it would let a truncated/checksummed/decompressing stream look
	// like a clean parse once the buffered/converted bytes drain. We remember it
	// and surface it only after all already-read output has been delivered.
	readErr error
}

// newDeferredLatin1Reader builds the deferred reader for an UNDECLARED stream.
// Its lazy Latin-1 interpretation is always Windows-1252 (a declared
// charset=iso-8859-1 takes the immediate latin1Reader path in wrapReaderForHTML
// and never reaches here).
func newDeferredLatin1Reader(r io.Reader) *deferredLatin1Reader {
	return &deferredLatin1Reader{
		r:        r,
		encOnHit: encWindows1252,
	}
}

// detectedEncoding returns the encoding name once a non-UTF-8 byte has forced
// the reader into Latin-1 mode, or "" while the stream is still pure UTF-8.
func (dr *deferredLatin1Reader) detectedEncoding() string {
	return dr.enc
}

// encodingError reports whether a post-commit invalid byte was sanitized to
// U+FFFD after the reader committed to UTF-8 at the bounded prefix, and the
// line/col (relative to the commit boundary) of the first such byte. It returns
// (false, 0, 0) when no commit/sanitizer is in play — a pure-UTF-8 stream or one
// that switched to Latin-1, neither of which sanitizes. The parser queries this
// so a late invalid byte raises the same "Invalid bytes in character encoding"
// SAX diagnostic the declared-UTF-8 sanitizer path emits.
func (dr *deferredLatin1Reader) encodingError() (bool, int, int) {
	if dr.sanitizer == nil {
		return false, 0, 0
	}
	return dr.sanitizer.EncodingError()
}

func (dr *deferredLatin1Reader) Read(p []byte) (int, error) {
	for {
		// Drain any converted output first.
		if dr.outPos < len(dr.out) {
			n := copy(p, dr.out[dr.outPos:])
			dr.outPos += n
			if dr.outPos >= len(dr.out) {
				dr.out = dr.out[:0]
				dr.outPos = 0
			}
			return n, nil
		}

		// Once switched to Latin-1, every remaining byte is one Latin-1 rune;
		// stream it through directly without further buffering.
		if dr.switched {
			n, err := dr.fillLatin1(p)
			if n > 0 {
				return n, nil
			}
			return 0, err
		}

		// Once committed to UTF-8 after the bounded prefix, pass the remaining
		// bytes through verbatim without further buffering.
		if dr.committedUTF8 {
			n, err := dr.fillUTF8(p)
			if n > 0 {
				return n, nil
			}
			return 0, err
		}

		if dr.eof {
			// EOF reached, decision already made and output drained. Surface a
			// sticky non-EOF read error here so a stream that returned data
			// together with an error is not mistaken for a clean end.
			if dr.readErr != nil {
				err := dr.readErr
				dr.readErr = nil
				return 0, err
			}
			return 0, io.EOF
		}

		// Undecided: read more raw bytes and buffer them while they remain
		// valid UTF-8. Emit nothing until the decision is forced.
		var buf [4096]byte
		n, err := dr.r.Read(buf[:])
		dr.pending = append(dr.pending, buf[:n]...)
		switch {
		case err == io.EOF:
			dr.eof = true
		case err != nil:
			// A non-EOF error. io.Reader allows returning n > 0 alongside an
			// error, so keep any bytes we just buffered and remember the error
			// as sticky; treat the stream as ended and deliver buffered output
			// first, surfacing the error only after it drains.
			dr.readErr = err
			dr.eof = true
		}

		// Re-evaluate the buffered bytes; decide() makes output available once
		// it can (at the first invalid byte, or at EOF). When it stays
		// undecided we loop to read more.
		dr.decide()
	}
}

// decide inspects the buffered pending bytes. If a genuine non-UTF-8 byte is
// present it switches to Latin-1 and converts the WHOLE pending buffer; if EOF
// has been reached with everything valid it flushes pending as UTF-8.
// Returns true if it produced output (or switched), false if still undecided.
func (dr *deferredLatin1Reader) decide() bool {
	data := dr.pending
	for i := 0; i < len(data); {
		b := data[i]
		if b < 0x80 {
			i++
			continue
		}
		r, size := utf8.DecodeRune(data[i:])
		if r == utf8.RuneError && size <= 1 {
			// A decode failure here is either a genuinely invalid byte or a
			// multibyte rune merely truncated by the end of the buffer. Defer
			// ONLY for a truly incomplete trailing rune — one whose continuation
			// bytes have not all arrived yet (!utf8.FullRune). A genuine invalid
			// byte (a lone continuation byte such as 0x80/0x93, or a lead byte
			// with a bad continuation) is a "full" RuneError of size 1 and must
			// switch to Latin-1 right here, BEFORE the cap check below — otherwise
			// an invalid byte landing at the commit boundary would be flushed
			// verbatim as UTF-8 instead of flipping the whole buffer to
			// Windows-1252 the way the in-memory []byte path does.
			if !dr.eof && !utf8.FullRune(data[i:]) {
				break
			}
			// Genuine non-UTF-8 byte: reinterpret the ENTIRE buffer (from the
			// start) as Latin-1/Windows-1252, matching the []byte path.
			dr.switched = true
			dr.enc = dr.encOnHit
			dr.out = latin1ToUTF8(dr.pending)
			dr.outPos = 0
			dr.pending = nil
			return true
		}
		i += size
	}

	// No genuine non-UTF-8 byte found. If the stream is exhausted, the whole
	// document is valid UTF-8: flush it unchanged.
	if dr.eof {
		dr.out = dr.pending
		dr.outPos = 0
		dr.pending = nil
		return true
	}

	// Bounded buffering: a cap's worth of bytes has been seen with no non-UTF-8
	// byte. Commit to UTF-8 rather than buffer the (possibly endless) stream
	// whole. Flush the complete-UTF-8 prefix verbatim (it is already known valid)
	// and route the remainder through a sanitizer.
	//
	// This is a deliberate, documented divergence from the whole-document []byte
	// path, and ONLY for the pathological case of an undeclared stream that stays
	// valid UTF-8 for more than deferredLatin1MaxBuffer bytes and THEN contains a
	// non-UTF-8 byte: the []byte path would reinterpret the whole document as
	// Latin-1, but here we have already committed to UTF-8 to keep memory bounded.
	// Rather than leak the raw invalid byte into SAX/DOM (and leave the document
	// ill-formed), we sanitize it to U+FFFD exactly as the parser handles any
	// decode error. A real document that declares or stays in a single encoding
	// settles far below the cap and is unaffected.
	//
	// A trailing incomplete rune at the commit boundary must NOT be flushed
	// verbatim: its continuation bytes arrive next from the underlying reader, and
	// the sanitizer would otherwise see them orphaned and mangle them. Split it
	// off and feed it back into the sanitizer so the rune reassembles intact.
	if len(dr.pending) >= deferredLatin1MaxBuffer {
		dr.committedUTF8 = true
		head, tail := splitTrailingPartialRune(dr.pending)
		dr.out = head
		dr.outPos = 0
		dr.pending = nil
		dr.sanitizer = newUTF8SanitizeReader(io.MultiReader(bytes.NewReader(tail), dr.r))
		return true
	}

	// Otherwise keep buffering.
	return false
}

// fillUTF8 reads from the post-commit sanitizer into p (used once the reader has
// committed to UTF-8 after the bounded prefix). The sanitizer passes valid UTF-8
// through unchanged but replaces any later invalid byte with U+FFFD, so a late
// non-UTF-8 byte in an undeclared >1 MiB-valid stream never leaks raw into
// SAX/DOM.
func (dr *deferredLatin1Reader) fillUTF8(p []byte) (int, error) {
	// The sanitizer owns sticky-error/drain ordering: it never surfaces a
	// non-EOF read error while it still holds buffered converted output, so a
	// late underlying error (delivered together with data, possibly with more
	// sanitized bytes still buffered inside the sanitizer) cannot truncate the
	// committed UTF-8 stream. Delegate to it directly.
	return dr.sanitizer.Read(p)
}

// splitTrailingPartialRune splits b into a complete-UTF-8 prefix and a trailing
// incomplete multibyte rune (0-3 bytes). b is assumed valid UTF-8 except for a
// possible truncated rune at its very end (the only invalid tail the deferred
// reader can hold at the commit boundary). The complete prefix is safe to emit
// verbatim; the partial tail must ride along into the sanitizer so its
// continuation bytes reassemble.
func splitTrailingPartialRune(b []byte) ([]byte, []byte) {
	for back := 1; back <= utf8.UTFMax-1 && back <= len(b); back++ {
		i := len(b) - back
		c := b[i]
		if c < 0x80 {
			// ASCII byte at the boundary: nothing partial trails it.
			return b, nil
		}
		if utf8.RuneStart(c) {
			// Lead byte: the tail is a complete rune unless DecodeRune reports a
			// truncated one (size <= 1 with RuneError).
			r, size := utf8.DecodeRune(b[i:])
			if r == utf8.RuneError && size <= 1 {
				return b[:i], b[i:]
			}
			return b, nil
		}
		// Continuation byte: keep walking back toward its lead byte.
	}
	return b, nil
}

// fillLatin1 converts raw bytes from the underlying reader as Latin-1/Windows
// -1252 directly into p (used once the reader has switched).
func (dr *deferredLatin1Reader) fillLatin1(p []byte) (int, error) {
	// A sticky non-EOF error was already recorded on a previous read that also
	// returned bytes. Those converted bytes have now drained (we only reach
	// fillLatin1 with empty output), so surface the error WITHOUT reading dr.r
	// again — re-reading a reader that already failed is wrong and could even
	// drop an error that is observable only once.
	if dr.readErr != nil {
		e := dr.readErr
		dr.readErr = nil
		return 0, e
	}

	var buf [2048]byte
	n, err := dr.r.Read(buf[:])
	// io.Reader may deliver data together with a non-EOF error. Remember the
	// error as sticky and convert any bytes we did get; the error is surfaced
	// once the converted output has drained.
	if err != nil && err != io.EOF {
		dr.readErr = err
	}
	if n == 0 {
		if dr.readErr != nil {
			e := dr.readErr
			dr.readErr = nil
			return 0, e
		}
		return 0, err
	}
	dr.out = latin1ToUTF8(buf[:n])
	dr.outPos = 0
	written := copy(p, dr.out)
	dr.outPos += written
	if dr.outPos >= len(dr.out) {
		dr.out = dr.out[:0]
		dr.outPos = 0
	}
	return written, nil
}

// headHasGenuineInvalidUTF8 reports whether the sniff buffer contains a UTF-8
// sequence that is invalid for a reason OTHER than truncation at the end of the
// buffer.
//
// utf8.Valid is too blunt for the 1024-byte sniff window: if byte 1024 happens
// to split a valid multibyte rune, utf8.Valid(head) reports false even though
// the document is perfectly valid UTF-8 — the trailing rune is merely
// incomplete because the window ended mid-rune. Classifying such a document as
// Latin-1/Windows-1252 corrupts every multibyte rune (e.g. é → Ã©).
//
// This walks the buffer and only reports invalidity for a genuine bad sequence.
// A lead byte at the tail with too few continuation bytes to complete its rune
// is treated as an incomplete trailing rune (not invalid), so the caller can
// defer the decision and let more bytes — or EOF — settle it.
func headHasGenuineInvalidUTF8(head []byte) bool {
	for i := 0; i < len(head); {
		b := head[i]
		if b < 0x80 {
			i++
			continue
		}
		r, size := utf8.DecodeRune(head[i:])
		if r == utf8.RuneError && size <= 1 {
			// A decode failure of size <= 1 is either a genuinely invalid byte
			// or an incomplete multibyte sequence cut off by the buffer end.
			// Distinguish: if a valid lead byte sits within UTFMax-1 bytes of
			// the end and the remaining bytes are all valid continuation bytes,
			// it is an incomplete trailing rune — not a genuine error.
			if i+utf8.UTFMax > len(head) && isIncompleteTrailingRune(head[i:]) {
				return false
			}
			return true
		}
		i += size
	}
	return false
}

// isIncompleteTrailingRune reports whether tail is the beginning of a valid
// multibyte UTF-8 rune that was cut off before completion: a valid lead byte
// followed only by valid continuation bytes, but fewer than the lead byte
// requires.
func isIncompleteTrailingRune(tail []byte) bool {
	if len(tail) == 0 {
		return false
	}
	b := tail[0]
	var need int
	switch {
	case b&0xE0 == 0xC0:
		need = 2
	case b&0xF0 == 0xE0:
		need = 3
	case b&0xF8 == 0xF0:
		need = 4
	default:
		// Not a valid lead byte (continuation byte or invalid prefix).
		return false
	}
	if len(tail) >= need {
		// Enough bytes were present to complete the rune, so the failure is a
		// genuine bad sequence, not truncation.
		return false
	}
	// All bytes after the lead must be valid continuation bytes for this to be
	// a clean truncation rather than an invalid sequence.
	for _, c := range tail[1:] {
		if c&0xC0 != 0x80 {
			return false
		}
	}
	return true
}

// wrapReaderForHTML wraps an io.Reader with the appropriate encoding
// transformation chain for HTML parsing:
//  1. Peek first 1024 bytes to detect charset
//  2. Apply newline normalization
//  3. Apply either Latin-1→UTF-8 conversion or UTF-8 sanitization
//
// Returns the wrapped reader, the detected encoding name (empty for UTF-8),
// the sanitizer (non-nil only for the charset=utf-8 path, for error position
// queries), and the deferred Latin-1 reader (non-nil only for the undeclared
// path, queried after parsing for the lazily-detected encoding name).
func wrapReaderForHTML(r io.Reader) (io.Reader, string, *utf8SanitizeReader, *deferredLatin1Reader) {
	// Read up to 1024 bytes for charset detection.
	//
	// We cannot use io.ReadFull here: it only reports an error when it reads
	// fewer than len(buf) bytes. If the underlying reader returns the final
	// sniff byte together with a non-EOF error — i.e. it fills the buffer
	// exactly (n == 1024, err != nil) — io.ReadFull discards that error and
	// returns (1024, nil), so a truncated/checksummed/decompressing stream
	// that happens to fit the detection window would look like a clean parse.
	// Loop manually and preserve any non-EOF error that arrives with the data.
	head := make([]byte, 1024)
	var n int
	var peekErr error
	for n < len(head) {
		m, err := r.Read(head[n:])
		n += m
		if err != nil {
			peekErr = err
			break
		}
		if m == 0 {
			// Reader made no progress and reported no error: treat as the end
			// of the sniff window to avoid spinning on a misbehaving reader.
			break
		}
	}
	head = head[:n]

	// io.EOF just means "stream ended"; it is not a failure. Any other
	// (non-EOF) error is a genuine read failure that may have arrived together
	// with the peeked bytes — it must not be silently dropped. Re-deliver it
	// after the peeked bytes via an errReader appended to the chain below.
	if peekErr == io.EOF {
		peekErr = nil
	}

	// Reconstruct full reader: peeked bytes + remainder (+ any sticky peek
	// error to surface once the buffered bytes drain).
	//
	// When the sniff read returned a non-EOF error, the underlying reader r has
	// already failed; reading it AGAIN would re-invoke a reader that errored
	// (and, for a reader whose error is observable only once, could even drop
	// the error). So when peekErr != nil we replay ONLY the bytes already read
	// (head[:n]) and then the errReader — never r again.
	var full io.Reader
	switch {
	case peekErr != nil && n > 0:
		full = io.MultiReader(bytes.NewReader(head), &errReader{err: peekErr})
	case peekErr != nil:
		full = &errReader{err: peekErr}
	case n > 0:
		full = io.MultiReader(bytes.NewReader(head), r)
	default:
		full = r
	}

	// Apply newline normalization.
	normalized := &newlineNormReader{r: full}

	// Detect encoding from the peeked bytes.
	//
	// Use headHasGenuineInvalidUTF8 rather than !utf8.Valid: the latter reports
	// false when byte 1024 splits an otherwise-valid multibyte rune, which would
	// misclassify a valid UTF-8 document as Latin-1. An incomplete trailing rune
	// is not a genuine error here — it falls through to the deferred reader,
	// whose decide() re-reads more bytes (or EOF) to settle the encoding.
	if n > 0 && headHasGenuineInvalidUTF8(head) {
		// Non-UTF-8 detected in the head. Check if charset is declared.
		if !declaredCharsetIsUTF8(head) {
			enc := encWindows1252
			if declaredCharsetIsLatin1(head) {
				enc = encISO88591
			}
			return &latin1Reader{r: normalized, enc: enc}, enc, nil, nil
		}
		// charset=utf-8 declared but head has invalid bytes — sanitize.
		san := newUTF8SanitizeReader(normalized)
		return san, "", san, nil
	}

	// charset=utf-8 is explicitly declared: any later invalid bytes are
	// genuine encoding errors and must be replaced with U+FFFD.
	if declaredCharsetIsUTF8(head) {
		san := newUTF8SanitizeReader(normalized)
		return san, "", san, nil
	}

	// An explicit charset=iso-8859-1 is a DECLARED encoding: commit to Latin-1
	// immediately rather than routing through the deferred/bounded path. The
	// deferred path is only for UNDECLARED streams; sending a declared Latin-1
	// stream through it lets the 1 MiB commit cap wrongly settle a valid
	// ISO-8859-1 document (ASCII head, first high byte past the cap) as UTF-8,
	// emitting U+FFFD and leaving the encoding unreported. latin1Reader streams
	// the whole document as ISO-8859-1 with no buffering and no cap.
	if declaredCharsetIsLatin1(head) {
		return &latin1Reader{r: normalized, enc: encISO88591}, encISO88591, nil, nil
	}

	// Head is valid UTF-8 (or empty) and not declared as charset=utf-8 or
	// charset=iso-8859-1. The document may still turn out to be Windows-1252 past
	// the detection window, so defer the decision: stay UTF-8 until a non-UTF-8
	// byte appears, then interpret the remainder as Windows-1252 (matching the
	// whole-document []byte parse path).
	dr := newDeferredLatin1Reader(normalized)
	return dr, "", nil, dr
}
