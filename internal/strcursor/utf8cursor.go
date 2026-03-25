package strcursor

import (
	"bytes"
	"errors"
	"io"
	"unicode/utf8"
)

// UTF8Cursor is a high-performance cursor for UTF-8 encoded input.
// It works directly on a byte buffer, decoding UTF-8 on the fly.
// ASCII bytes (< 0x80) are handled without utf8.DecodeRune overhead.
type UTF8Cursor struct {
	buf    []byte
	buflen int
	bufpos int
	column int
	in     io.Reader
	line   bytes.Buffer
	lineno int
}

// NewUTF8Cursor creates a UTF8Cursor wrapping an existing io.Reader.
func NewUTF8Cursor(r io.Reader) *UTF8Cursor {
	return &UTF8Cursor{
		buf:    make([]byte, 8192),
		buflen: 0,
		bufpos: 0,
		column: 1,
		in:     r,
		lineno: 1,
	}
}

// fillBuffer ensures at least minBytes are available from bufpos.
func (c *UTF8Cursor) fillBuffer(minBytes int) error {
	avail := c.buflen - c.bufpos
	if avail >= minBytes {
		return nil
	}

	// Compact: move unconsumed bytes to front.
	if c.bufpos > 0 {
		if avail > 0 {
			copy(c.buf, c.buf[c.bufpos:c.buflen])
		}
		c.buflen = avail
		c.bufpos = 0
	}

	// Grow buffer if needed.
	if minBytes > len(c.buf) {
		newBuf := make([]byte, minBytes*2)
		copy(newBuf, c.buf[:c.buflen])
		c.buf = newBuf
	}

	// Read until we have enough.
	for c.buflen-c.bufpos < minBytes {
		n, err := c.in.Read(c.buf[c.buflen:])
		c.buflen += n
		if n == 0 && err != nil {
			if c.buflen-c.bufpos >= minBytes {
				return nil
			}
			return err
		}
	}
	return nil
}

// nthRuneOffset returns the byte offset (from bufpos) of the start of the
// n-th rune (0-indexed). It ensures sufficient data is buffered.
func (c *UTF8Cursor) nthRuneOffset(n int) (int, error) {
	off := 0
	for i := 0; i < n; i++ {
		if err := c.fillBuffer(off + 1); err != nil {
			return off, err
		}
		b := c.buf[c.bufpos+off]
		if b < 0x80 {
			off++
		} else {
			if err := c.fillBuffer(off + utf8.UTFMax); err != nil {
				// might still have enough for a shorter sequence
				if c.bufpos+off >= c.buflen {
					return off, err
				}
			}
			_, w := utf8.DecodeRune(c.buf[c.bufpos+off : c.buflen])
			if w == 0 {
				return off, io.EOF
			}
			off += w
		}
	}
	return off, nil
}

func (c *UTF8Cursor) Done() bool {
	if c.bufpos < c.buflen {
		return false
	}
	return c.fillBuffer(1) != nil
}

func (c *UTF8Cursor) Peek() rune {
	if c.bufpos >= c.buflen {
		if c.fillBuffer(1) != nil {
			return utf8.RuneError
		}
	}
	b := c.buf[c.bufpos]
	if b < 0x80 {
		return rune(b)
	}
	if c.buflen-c.bufpos < utf8.UTFMax {
		_ = c.fillBuffer(utf8.UTFMax)
	}
	r, _ := utf8.DecodeRune(c.buf[c.bufpos:c.buflen])
	return r
}

func (c *UTF8Cursor) PeekN(n int) rune {
	// Fast path: if the first n bytes from bufpos are all ASCII, then byte
	// position == rune position. This is the common case for XML names,
	// delimiters, and most content.
	if c.buflen-c.bufpos < n {
		if c.fillBuffer(n) != nil {
			return utf8.RuneError
		}
	}
	allASCII := true
	for i := 0; i < n; i++ {
		if c.buf[c.bufpos+i] >= 0x80 {
			allASCII = false
			break
		}
	}
	if allASCII {
		return rune(c.buf[c.bufpos+n-1])
	}

	// Slow path: multi-byte UTF-8 present. Scan rune boundaries.
	off, err := c.nthRuneOffset(n - 1)
	if err != nil {
		return utf8.RuneError
	}
	if err := c.fillBuffer(off + utf8.UTFMax); err != nil {
		if c.bufpos+off >= c.buflen {
			return utf8.RuneError
		}
	}
	b := c.buf[c.bufpos+off]
	if b < 0x80 {
		return rune(b)
	}
	r, _ := utf8.DecodeRune(c.buf[c.bufpos+off : c.buflen])
	return r
}

func (c *UTF8Cursor) PeekString(n int) string {
	// Fast path: if first n bytes are ASCII, byte count == rune count.
	if err := c.fillBuffer(n); err != nil {
		return ""
	}
	allASCII := true
	for i := 0; i < n; i++ {
		if c.bufpos+i >= c.buflen {
			allASCII = false
			break
		}
		if c.buf[c.bufpos+i] >= 0x80 {
			allASCII = false
			break
		}
	}
	if allASCII {
		return string(c.buf[c.bufpos : c.bufpos+n])
	}

	// Slow path: multi-byte.
	off, err := c.nthRuneOffset(n)
	if err != nil {
		return ""
	}
	return string(c.buf[c.bufpos : c.bufpos+off])
}

func (c *UTF8Cursor) Cur() rune {
	r := c.Peek()
	if r != utf8.RuneError {
		_ = c.Advance(1)
	}
	return r
}

func (c *UTF8Cursor) Advance(n int) error {
	for i := 0; i < n; i++ {
		if err := c.fillBuffer(1); err != nil {
			return err
		}
		b := c.buf[c.bufpos]
		var w int
		var r rune
		if b < 0x80 {
			r = rune(b)
			w = 1
		} else {
			_ = c.fillBuffer(utf8.UTFMax)
			r, w = utf8.DecodeRune(c.buf[c.bufpos:c.buflen])
			if w == 0 {
				return errors.New("invalid UTF-8")
			}
		}
		if r == '\n' {
			c.lineno++
			c.line.Reset()
			c.column = 1
		} else {
			c.column++
		}
		if w == 1 {
			c.line.WriteByte(b)
		} else {
			c.line.Write(c.buf[c.bufpos : c.bufpos+w])
		}
		c.bufpos += w
	}
	return nil
}

func (c *UTF8Cursor) AdvanceFast(n int) error {
	lastNewline := -1
	for i := 0; i < n; i++ {
		if err := c.fillBuffer(1); err != nil {
			return err
		}
		b := c.buf[c.bufpos]
		if b < 0x80 {
			if b == '\n' {
				c.lineno++
				lastNewline = i
			}
			c.bufpos++
		} else {
			_ = c.fillBuffer(utf8.UTFMax)
			_, w := utf8.DecodeRune(c.buf[c.bufpos:c.buflen])
			if w == 0 {
				return errors.New("invalid UTF-8")
			}
			c.bufpos += w
		}
	}
	if lastNewline >= 0 {
		c.column = n - lastNewline
		c.line.Reset()
	} else {
		c.column += n
	}
	return nil
}

func (c *UTF8Cursor) HasPrefix(b []byte) bool {
	n := len(b)
	if err := c.fillBuffer(n); err != nil {
		return false
	}
	return bytes.HasPrefix(c.buf[c.bufpos:c.buflen], b)
}

func (c *UTF8Cursor) HasPrefixString(s string) bool {
	n := len(s)
	if c.buflen-c.bufpos < n {
		if c.fillBuffer(n) != nil {
			return false
		}
		if c.buflen-c.bufpos < n {
			return false
		}
	}
	return string(c.buf[c.bufpos:c.bufpos+n]) == s
}

func (c *UTF8Cursor) Consume(b []byte) bool {
	if !c.HasPrefix(b) {
		return false
	}
	n := utf8.RuneCount(b)
	_ = c.Advance(n)
	return true
}

func (c *UTF8Cursor) ConsumeString(s string) bool {
	if !c.HasPrefixString(s) {
		return false
	}
	n := utf8.RuneCountInString(s)
	_ = c.Advance(n)
	return true
}

func (c *UTF8Cursor) Line() string {
	return c.line.String()
}

func (c *UTF8Cursor) LineNumber() int {
	return c.lineno
}

func (c *UTF8Cursor) Column() int {
	return c.column
}

func (c *UTF8Cursor) Unused() io.Reader {
	ret := &Unused{rdr: c.in}
	if buf := c.buf[c.bufpos:c.buflen]; len(buf) > 0 {
		ret.unused = make([]byte, len(buf))
		copy(ret.unused, buf)
	}
	return ret
}

func (c *UTF8Cursor) Read(buf []byte) (int, error) {
	nread := 0
	if c.bufpos < c.buflen {
		avail := c.buflen - c.bufpos
		if len(buf) >= avail {
			copy(buf, c.buf[c.bufpos:c.buflen])
			nread = avail
			buf = buf[nread:]
			c.bufpos = c.buflen
		} else {
			copy(buf, c.buf[c.bufpos:c.bufpos+len(buf)])
			c.bufpos += len(buf)
			return len(buf), nil
		}
	}
	n, err := c.in.Read(buf)
	return nread + n, err
}

// ScanNCName scans an XML NCName from the current position. Returns the name
// string and the rune count. Returns ("", 0) if the current position is not a
// valid NCName start character. The caller must call Advance(nRunes) after.
func (c *UTF8Cursor) ScanNCName() (string, int) {
	if err := c.fillBuffer(1); err != nil {
		return "", 0
	}

	// Use offset from bufpos to stay safe across fillBuffer compaction.
	off := 0
	// Check first character: must be NameStartChar (without ':').
	b := c.buf[c.bufpos+off]
	if b < 0x80 {
		if !((b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || b == '_') {
			return "", 0
		}
		off++
	} else {
		_ = c.fillBuffer(utf8.UTFMax)
		r, w := utf8.DecodeRune(c.buf[c.bufpos:c.buflen])
		if r == utf8.RuneError || !isNCNameStartChar(r) {
			return "", 0
		}
		off += w
	}

	// Scan remaining NameChars.
	nRunes := 1
	for {
		if c.bufpos+off >= c.buflen {
			if c.fillBuffer(off + 1) != nil {
				break
			}
			if c.bufpos+off >= c.buflen {
				break
			}
		}
		b = c.buf[c.bufpos+off]
		if b < 0x80 {
			if !isASCIINameChar(b) {
				break
			}
			off++
			nRunes++
		} else {
			_ = c.fillBuffer(off + utf8.UTFMax)
			r, w := utf8.DecodeRune(c.buf[c.bufpos+off : c.buflen])
			if r == utf8.RuneError || !isNCNameChar(r) {
				break
			}
			off += w
			nRunes++
		}
	}

	return string(c.buf[c.bufpos : c.bufpos+off]), nRunes
}

// isASCIINameChar checks if b is a valid ASCII XML NameChar (without ':').
func isASCIINameChar(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') ||
		(b >= '0' && b <= '9') || b == '_' || b == '-' || b == '.'
}

// isNCNameStartChar checks NameStartChar production (without ':').
func isNCNameStartChar(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || r == '_' ||
		(r >= 0xC0 && r <= 0xD6) || (r >= 0xD8 && r <= 0xF6) ||
		(r >= 0xF8 && r <= 0x2FF) || (r >= 0x370 && r <= 0x37D) ||
		(r >= 0x37F && r <= 0x1FFF) || (r >= 0x200C && r <= 0x200D) ||
		(r >= 0x2070 && r <= 0x218F) || (r >= 0x2C00 && r <= 0x2FEF) ||
		(r >= 0x3001 && r <= 0xD7FF) || (r >= 0xF900 && r <= 0xFDCF) ||
		(r >= 0xFDF0 && r <= 0xFFFD) || (r >= 0x10000 && r <= 0xEFFFF)
}

// isNCNameChar checks NameChar production (without ':').
func isNCNameChar(r rune) bool {
	return isNCNameStartChar(r) ||
		(r >= '0' && r <= '9') || r == '-' || r == '.' ||
		r == 0xB7 || (r >= 0x0300 && r <= 0x036F) || (r >= 0x203F && r <= 0x2040)
}

// ScanSimpleAttrValue scans a simple attribute value (no entities, no special
// whitespace) between the current position and the given quote character.
// Returns the value string and rune count, or ("", 0) if the value contains
// entities or special characters that require the slow path.
// Does NOT consume — caller must call Advance(nRunes) after.
func (c *UTF8Cursor) ScanSimpleAttrValue(quote byte) (string, int) {
	if c.fillBuffer(1) != nil {
		return "", 0
	}

	off := 0
	nRunes := 0
	for {
		if c.bufpos+off >= c.buflen {
			if c.fillBuffer(off + 1) != nil {
				return "", 0
			}
			if c.bufpos+off >= c.buflen {
				return "", 0
			}
		}
		b := c.buf[c.bufpos+off]
		if b == quote {
			// End of value.
			return string(c.buf[c.bufpos : c.bufpos+off]), nRunes
		}
		if b == '&' || b == '<' {
			// Entity reference or invalid char — need slow path.
			return "", 0
		}
		if b < 0x80 {
			if b < 0x20 && b != 0x9 {
				// \r, \n, or other control chars need normalization.
				return "", 0
			}
			off++
			nRunes++
		} else {
			_ = c.fillBuffer(off + utf8.UTFMax)
			_, w := utf8.DecodeRune(c.buf[c.bufpos+off : c.buflen])
			if w == 0 {
				return "", 0
			}
			off += w
			nRunes++
		}
	}
}

// ScanCharDataInto scans XML character data with inline EOL normalization.
// Does NOT consume — caller must call AdvanceFast(nRunes) after processing.
// The returned rune count counts \r\n as 2 runes to match what AdvanceFast sees.
func (c *UTF8Cursor) ScanCharDataInto(dst *bytes.Buffer) int {
	if c.fillBuffer(1) != nil {
		return 0
	}

	nRunes := 0
	pos := c.bufpos
	dst.Grow(c.buflen - pos)

	for pos < c.buflen {
		b := c.buf[pos]
		if b < 0x80 {
			if b == '<' || b == '&' {
				break
			}
			if b < 0x20 && b != 0x9 && b != 0xa && b != 0xd {
				break
			}
			if b == ']' && pos+2 < c.buflen && c.buf[pos+1] == ']' && c.buf[pos+2] == '>' {
				break
			}
			if b == '\r' {
				dst.WriteByte('\n')
				pos++
				nRunes++ // count \r as a rune
				if pos < c.buflen && c.buf[pos] == '\n' {
					pos++
					nRunes++ // count \n as a rune too — AdvanceFast sees both
				}
				continue
			}
			dst.WriteByte(b)
			pos++
			nRunes++
			continue
		}
		// Multi-byte UTF-8. Try to ensure enough bytes.
		if c.buflen-pos < utf8.UTFMax {
			_ = c.fillBuffer(pos - c.bufpos + utf8.UTFMax)
		}
		r, w := utf8.DecodeRune(c.buf[pos:c.buflen])
		if r == utf8.RuneError || w == 0 {
			break
		}
		if r < 0x20 {
			break
		}
		dst.WriteRune(r)
		pos += w
		nRunes++
	}

	return nRunes
}
