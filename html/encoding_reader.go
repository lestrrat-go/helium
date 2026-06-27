package html

import (
	"bytes"
	"fmt"
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

	// readErr is a sticky non-EOF error returned by the underlying reader. An
	// io.Reader may return n > 0 together with a non-EOF error in a single Read;
	// surfacing it before the converted output drains would truncate the stream.
	// We remember it and surface it only once all buffered converted output has
	// been delivered.
	readErr error
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

	// Buffered output is fully drained. A sticky non-EOF error recorded with an
	// earlier chunk of data surfaces here, now that all of that chunk's converted
	// bytes have been delivered — never before, or the tail would be lost.
	if lr.readErr != nil {
		e := lr.readErr
		lr.readErr = nil
		return 0, e
	}

	// Read raw bytes. An io.Reader may return n > 0 together with a non-EOF
	// error; remember it as sticky and convert the bytes we did get, surfacing
	// the error only after the converted output drains.
	n, err := lr.r.Read(lr.raw[:])
	if err != nil && err != io.EOF {
		lr.readErr = err
	}
	if n == 0 {
		if lr.readErr != nil {
			e := lr.readErr
			lr.readErr = nil
			return 0, e
		}
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
	drained := lr.outPos >= len(lr.out)
	if drained {
		lr.out = lr.out[:0]
		lr.outPos = 0
	}

	// Surface EOF only once all converted output has drained; defer a sticky
	// non-EOF error to a later call (the drain check above re-surfaces it) so the
	// caller can never stop on the error with converted bytes still pending.
	if drained && lr.readErr == nil && err == io.EOF {
		return written, io.EOF
	}
	return written, nil
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
// The undecided UTF-8-valid prefix is buffered only up to maxBuffer bytes before
// the reader must reach a decision. That bound is the parser's CONFIGURED content
// limit ([Parser.MaxContentSize] / parseConfig.contentLimit(), 16 MiB by default)
// — the same limit the rest of the parser already enforces, so a legitimate
// multi-megabyte ASCII/UTF-8 document under that limit parses normally while an
// endless all-valid stream still cannot be buffered without limit.
//
// The detected encoding name is reported lazily via detectedEncoding once the
// switch happens.
//
// Buffering is bounded and fails closed: deferring until EOF would buffer an
// endless all-valid stream whole, defeating streaming and the parser's content
// caps (an unbounded-memory DoS). The bound is enforced at the cap BOUNDARY,
// independent of the reader's chunk sizes: each undecided read is limited to the
// remaining cap so the pending buffer never grows past maxBuffer, and the
// encoding decision is made on the first maxBuffer bytes alone. Once exactly
// maxBuffer valid-UTF-8 bytes have buffered with no non-UTF-8 byte, a single
// one-byte EOF probe distinguishes the two legitimate outcomes:
//   - the stream ends right at the cap → the whole prefix is valid UTF-8, so it
//     is accepted and flushed as UTF-8; or
//   - at least one more byte follows → the exact encoding decision cannot be
//     made within the memory bound (a later high byte would flip the WHOLE
//     document to Latin-1 per the []byte path, while EOF-while-valid would keep
//     it UTF-8), so the reader returns a bounded-input error
//     (ErrContentSizeExceeded) rather than committing to one interpretation and
//     risking silently mis-decoded SAX/DOM output that diverges from
//     Parse([]byte). The probed byte's value is irrelevant — its mere presence
//     means input ran past the cap.
//
// Real (finite) documents that declare or stay in a single encoding settle their
// encoding far below this cap and are unaffected; only a pathological undeclared
// stream that stays valid UTF-8 past the cap is rejected, fail-closed, instead of
// producing different text.
type deferredLatin1Reader struct {
	r io.Reader

	maxBuffer int // bound on the undecided pending buffer (parseConfig.contentLimit())

	pending []byte // undecided raw bytes buffered while still valid UTF-8
	out     []byte // converted output ready to consume
	outPos  int

	switched bool   // true once a non-UTF-8 byte forced Latin-1 interpretation
	eof      bool   // true once the underlying reader has reported EOF
	enc      string // encoding name reported after switching

	// capErr is the sticky bounded-input error returned once the undecided
	// (all-valid-UTF-8) prefix fills maxBuffer and a one-byte EOF probe proves
	// more input follows. The reader fails closed here rather than committing to
	// UTF-8.
	capErr error

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
// and never reaches here). maxBuffer bounds the undecided pending buffer; a value
// <= 0 falls back to defaultMaxContentSize.
func newDeferredLatin1Reader(r io.Reader, maxBuffer int) *deferredLatin1Reader {
	if maxBuffer <= 0 {
		maxBuffer = defaultMaxContentSize
	}
	return &deferredLatin1Reader{r: r, maxBuffer: maxBuffer}
}

// detectedEncoding returns the encoding name once a non-UTF-8 byte has forced
// the reader into Latin-1 mode, or "" while the stream is still pure UTF-8.
func (dr *deferredLatin1Reader) detectedEncoding() string {
	return dr.enc
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

		// The undecided prefix reached the buffering cap without settling the
		// encoding. Fail closed with the bounded-input error rather than commit to
		// a UTF-8 interpretation that a later high byte would contradict.
		if dr.capErr != nil {
			return 0, dr.capErr
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

		// Undecided. Never buffer past maxBuffer while the encoding is still
		// unsettled: bound each read to the remaining cap so the pending buffer
		// can never grow beyond maxBuffer. This makes the cap boundary
		// chunk-INDEPENDENT — an invalid byte that lands past the cap can no
		// longer be scanned into the buffer and retroactively flip an over-cap
		// prefix to Latin-1; the cap decision is made on the first maxBuffer
		// bytes alone, regardless of how the reader chunked its output.
		remaining := dr.maxBuffer - len(dr.pending)
		if remaining > 0 {
			var buf [4096]byte
			toRead := len(buf)
			if toRead > remaining {
				toRead = remaining
			}
			n, err := dr.r.Read(buf[:toRead])
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

			// Re-evaluate the buffered bytes; decide() makes output available
			// once it can (at the first invalid byte, or at EOF). When it stays
			// undecided we loop to read more.
			dr.decide()
			continue
		}

		// The undecided prefix has filled EXACTLY maxBuffer bytes with no
		// boundary-invalid byte. Settling the encoding would require buffering a
		// byte past the cap, so probe a single byte to tell the two cases apart
		// without ever reading further:
		//   - the stream ends exactly at the cap (valid UTF-8) → accept and
		//     flush the prefix as UTF-8; or
		//   - at least one more byte follows → fail closed
		//     (ErrContentSizeExceeded), independent of the late byte's value.
		var probe [1]byte
		n, err := dr.r.Read(probe[:])
		if n > 0 {
			// More than maxBuffer bytes and still undecided: fail closed rather
			// than commit to a UTF-8 interpretation a later high byte could
			// contradict. The probed byte is discarded unread, not buffered.
			dr.capErr = fmt.Errorf("undeclared HTML stream stayed valid UTF-8 for %d bytes without settling its encoding: %w", dr.maxBuffer, ErrContentSizeExceeded)
			dr.pending = nil
			return 0, dr.capErr
		}
		switch {
		case err == nil:
			// No byte and no error: a misbehaving reader made no progress. Loop
			// to probe again rather than spin, matching the chunk-read path.
			continue
		case err == io.EOF:
			// Stream ended exactly at the cap with everything valid UTF-8.
			dr.eof = true
		default:
			dr.readErr = err
			dr.eof = true
		}
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
			// start) as Latin-1/Windows-1252, matching the []byte path. (The
			// deferred reader is only entered for an UNDECLARED stream, whose lazy
			// Latin-1 interpretation is always Windows-1252.)
			dr.switched = true
			dr.enc = encWindows1252
			dr.out = latin1ToUTF8(dr.pending)
			dr.outPos = 0
			dr.pending = nil
			return true
		}
		i += size
	}

	// No genuine non-UTF-8 byte found. If the stream is exhausted, the whole
	// document is valid UTF-8: flush it unchanged. This includes the exact-cap
	// case (pending == maxBuffer) once Read's one-byte EOF probe has confirmed
	// the stream ended right at the cap.
	if dr.eof {
		dr.out = dr.pending
		dr.outPos = 0
		dr.pending = nil
		return true
	}

	// Still undecided and under the cap: keep buffering. The cap boundary itself
	// is enforced by Read — which bounds each read to the remaining cap and, once
	// pending fills maxBuffer, does a one-byte EOF probe to decide between
	// accept-at-EOF and fail-closed — so decide never has to reject here.
	return false
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
//
// maxBuffer bounds the undeclared deferred reader's undecided buffer; callers
// pass the parser's configured content limit (parseConfig.contentLimit()).
func wrapReaderForHTML(r io.Reader, maxBuffer int) (io.Reader, string, *utf8SanitizeReader, *deferredLatin1Reader) {
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
	dr := newDeferredLatin1Reader(normalized, maxBuffer)
	return dr, "", nil, dr
}
