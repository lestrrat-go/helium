package html

import (
	"bytes"
	"io"
	"unicode/utf8"
)

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

	// Read more raw data. Keep any trailing partial UTF-8 from previous read.
	leftover := sr.rawLen - sr.rawPos
	if leftover > 0 {
		copy(sr.raw, sr.raw[sr.rawPos:sr.rawLen])
	}
	sr.rawPos = 0
	sr.rawLen = leftover

	n, err := sr.r.Read(sr.raw[sr.rawLen:])
	sr.rawLen += n

	if sr.rawLen == 0 {
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
			if i+utf8.UTFMax > len(data) && err == nil {
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
		return 0, err
	}

	written := copy(p, sr.out[sr.outPos:])
	sr.outPos += written
	if sr.outPos >= len(sr.out) {
		sr.out = sr.out[:0]
		sr.outPos = 0
	}

	return written, err
}

func (sr *utf8SanitizeReader) trackByte(b byte) {
	if b == '\n' {
		sr.curLine++
		sr.curCol = 1
	} else {
		sr.curCol++
	}
}

// wrapReaderForHTML wraps an io.Reader with the appropriate encoding
// transformation chain for HTML parsing:
//  1. Peek first 1024 bytes to detect charset
//  2. Apply newline normalization
//  3. Apply either Latin-1→UTF-8 conversion or UTF-8 sanitization
//
// Returns the wrapped reader, the detected encoding name (empty for UTF-8),
// and the sanitizer (non-nil only for UTF-8 path, for error position queries).
func wrapReaderForHTML(r io.Reader) (io.Reader, string, *utf8SanitizeReader) {
	// Read up to 1024 bytes for charset detection.
	head := make([]byte, 1024)
	n, _ := io.ReadFull(r, head)
	head = head[:n]

	// Reconstruct full reader: peeked bytes + remainder.
	var full io.Reader
	if n > 0 {
		full = io.MultiReader(bytes.NewReader(head), r)
	} else {
		full = r
	}

	// Apply newline normalization.
	normalized := &newlineNormReader{r: full}

	// Detect encoding from the peeked bytes.
	if n > 0 && !utf8.Valid(head) {
		// Non-UTF-8 detected in the head. Check if charset is declared.
		if !declaredCharsetIsUTF8(head) {
			enc := "Windows-1252"
			if declaredCharsetIsLatin1(head) {
				enc = "ISO-8859-1"
			}
			return &latin1Reader{r: normalized, enc: enc}, enc, nil
		}
		// charset=utf-8 declared but head has invalid bytes — sanitize.
		san := newUTF8SanitizeReader(normalized)
		return san, "", san
	}

	// Head is valid UTF-8 (or empty). Still wrap with sanitizer in case
	// invalid bytes appear after the first 1024 bytes.
	if declaredCharsetIsUTF8(head) || n == 0 || utf8.Valid(head) {
		san := newUTF8SanitizeReader(normalized)
		return san, "", san
	}

	san := newUTF8SanitizeReader(normalized)
	return san, "", san
}
