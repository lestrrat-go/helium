package strcursor

import (
	"bytes"
	"io"
	"unicode/utf8"

	"github.com/lestrrat-go/helium/internal/xmlchar"
)

var charDataByteClass = buildCharDataByteClass()
var ncNameByteClass = buildNCNameByteClass()

func buildCharDataByteClass() [256]uint8 {
	var tbl [256]uint8
	for i := range 0x80 {
		tbl[i] = 0
	}
	for i := range 0x20 {
		tbl[i] = 1
	}
	tbl['\t'] = 0
	tbl['\n'] = 0
	tbl['<'] = 1
	tbl['&'] = 1
	tbl['\r'] = 1
	tbl[']'] = 1
	for i := 0x80; i < 0x100; i++ {
		tbl[i] = 1
	}
	return tbl
}

func buildNCNameByteClass() [256]uint8 {
	var tbl [256]uint8
	for i := range 0x100 {
		tbl[i] = 1
	}
	for i := 'A'; i <= 'Z'; i++ {
		tbl[i] = 0
	}
	for i := 'a'; i <= 'z'; i++ {
		tbl[i] = 0
	}
	for i := '0'; i <= '9'; i++ {
		tbl[i] = 0
	}
	tbl['_'] = 0
	tbl['-'] = 0
	tbl['.'] = 0
	return tbl
}

func scanSafeCharDataASCII(data []byte) int {
	off := 0
	for off+16 <= len(data) {
		if charDataByteClass[data[off+0]]|
			charDataByteClass[data[off+1]]|
			charDataByteClass[data[off+2]]|
			charDataByteClass[data[off+3]]|
			charDataByteClass[data[off+4]]|
			charDataByteClass[data[off+5]]|
			charDataByteClass[data[off+6]]|
			charDataByteClass[data[off+7]]|
			charDataByteClass[data[off+8]]|
			charDataByteClass[data[off+9]]|
			charDataByteClass[data[off+10]]|
			charDataByteClass[data[off+11]]|
			charDataByteClass[data[off+12]]|
			charDataByteClass[data[off+13]]|
			charDataByteClass[data[off+14]]|
			charDataByteClass[data[off+15]] != 0 {
			break
		}
		off += 16
	}
	for off < len(data) && charDataByteClass[data[off]] == 0 {
		off++
	}
	return off
}

func scanASCIINameChars(data []byte) int {
	off := 0
	for off+16 <= len(data) {
		if ncNameByteClass[data[off+0]]|
			ncNameByteClass[data[off+1]]|
			ncNameByteClass[data[off+2]]|
			ncNameByteClass[data[off+3]]|
			ncNameByteClass[data[off+4]]|
			ncNameByteClass[data[off+5]]|
			ncNameByteClass[data[off+6]]|
			ncNameByteClass[data[off+7]]|
			ncNameByteClass[data[off+8]]|
			ncNameByteClass[data[off+9]]|
			ncNameByteClass[data[off+10]]|
			ncNameByteClass[data[off+11]]|
			ncNameByteClass[data[off+12]]|
			ncNameByteClass[data[off+13]]|
			ncNameByteClass[data[off+14]]|
			ncNameByteClass[data[off+15]] != 0 {
			break
		}
		off += 16
	}
	for off < len(data) && ncNameByteClass[data[off]] == 0 {
		off++
	}
	return off
}

// UTF8Cursor is a high-performance cursor for UTF-8 encoded input.
// It works directly on a byte buffer, decoding UTF-8 on the fly.
// ASCII bytes (< 0x80) are handled without utf8.DecodeRune overhead.
type UTF8Cursor struct {
	buf    []byte
	buflen int
	bufpos int
	column int
	in     io.Reader
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

func (c *UTF8Cursor) Done() bool {
	if c.bufpos < c.buflen {
		return false
	}
	return c.fillBuffer(1) != nil
}

// Peek returns the byte at the current position, or 0 if at EOF.
func (c *UTF8Cursor) Peek() byte {
	if c.bufpos >= c.buflen {
		if c.fillBuffer(1) != nil {
			return 0
		}
	}
	return c.buf[c.bufpos]
}

// PeekAt returns the byte at offset bytes from the current position (0-indexed).
func (c *UTF8Cursor) PeekAt(offset int) byte {
	pos := c.bufpos + offset
	if pos >= c.buflen {
		if c.fillBuffer(offset+1) != nil {
			return 0
		}
		pos = c.bufpos + offset
		if pos >= c.buflen {
			return 0
		}
	}
	return c.buf[pos]
}

// PeekRune decodes and returns the rune at the current position.
func (c *UTF8Cursor) PeekRune() rune {
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

// PeekString returns n bytes from the current position as a string.
func (c *UTF8Cursor) PeekString(n int) string {
	if c.buflen-c.bufpos < n {
		if c.fillBuffer(n) != nil {
			return ""
		}
	}
	if c.bufpos+n > c.buflen {
		return ""
	}
	return string(c.buf[c.bufpos : c.bufpos+n])
}

// Advance consumes n bytes, updating line number and column tracking.
// The line buffer is not maintained eagerly — Line() reconstructs it on demand.
func (c *UTF8Cursor) Advance(n int) error {
	if c.buflen-c.bufpos < n {
		if err := c.fillBuffer(n); err != nil {
			return err
		}
	}
	if n == 1 {
		if c.buf[c.bufpos] == '\n' {
			c.lineno++
			c.column = 1
		} else {
			c.column++
		}
		c.bufpos++
		return nil
	}
	start := c.bufpos
	end := start + n
	segment := c.buf[start:end]
	lastNewline := -1
	for i, b := range segment {
		if b == '\n' {
			c.lineno++
			lastNewline = i
		}
	}
	if lastNewline >= 0 {
		c.column = len(segment) - lastNewline
	} else {
		c.column += n
	}
	c.bufpos = end
	return nil
}

// AdvanceFast skips per-byte column bookkeeping in the common no-newline case.
func (c *UTF8Cursor) AdvanceFast(n int) error {
	if c.buflen-c.bufpos < n {
		if err := c.fillBuffer(n); err != nil {
			return err
		}
	}
	if n == 1 {
		if c.buf[c.bufpos] == '\n' {
			c.lineno++
			c.column = 1
		} else {
			c.column++
		}
		c.bufpos++
		return nil
	}
	start := c.bufpos
	end := start + n
	segment := c.buf[start:end]
	if idx := bytes.LastIndexByte(segment, '\n'); idx >= 0 {
		c.lineno += bytes.Count(segment, []byte{'\n'})
		c.column = len(segment) - idx
	} else {
		c.column += n
	}
	c.bufpos = end
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
	data := c.buf[c.bufpos:]
	switch n {
	case 0:
		return true
	case 1:
		return data[0] == s[0]
	case 2:
		return data[0] == s[0] && data[1] == s[1]
	case 3:
		return data[0] == s[0] && data[1] == s[1] && data[2] == s[2]
	case 4:
		return data[0] == s[0] && data[1] == s[1] && data[2] == s[2] && data[3] == s[3]
	case 5:
		return data[0] == s[0] && data[1] == s[1] && data[2] == s[2] && data[3] == s[3] && data[4] == s[4]
	case 9:
		return data[0] == s[0] &&
			data[1] == s[1] &&
			data[2] == s[2] &&
			data[3] == s[3] &&
			data[4] == s[4] &&
			data[5] == s[5] &&
			data[6] == s[6] &&
			data[7] == s[7] &&
			data[8] == s[8]
	default:
		for i := range n {
			if data[i] != s[i] {
				return false
			}
		}
		return true
	}
}

func (c *UTF8Cursor) Consume(b []byte) bool {
	if !c.HasPrefix(b) {
		return false
	}
	_ = c.Advance(len(b))
	return true
}

func (c *UTF8Cursor) ConsumeString(s string) bool {
	n := len(s)
	if c.buflen-c.bufpos < n {
		if c.fillBuffer(n) != nil {
			return false
		}
		if c.buflen-c.bufpos < n {
			return false
		}
	}
	data := c.buf[c.bufpos:]
	switch n {
	case 0:
	case 1:
		if data[0] != s[0] {
			return false
		}
	case 2:
		if data[0] != s[0] || data[1] != s[1] {
			return false
		}
	case 3:
		if data[0] != s[0] || data[1] != s[1] || data[2] != s[2] {
			return false
		}
	case 4:
		if data[0] != s[0] || data[1] != s[1] || data[2] != s[2] || data[3] != s[3] {
			return false
		}
	case 5:
		if data[0] != s[0] || data[1] != s[1] || data[2] != s[2] || data[3] != s[3] || data[4] != s[4] {
			return false
		}
	case 9:
		if data[0] != s[0] ||
			data[1] != s[1] ||
			data[2] != s[2] ||
			data[3] != s[3] ||
			data[4] != s[4] ||
			data[5] != s[5] ||
			data[6] != s[6] ||
			data[7] != s[7] ||
			data[8] != s[8] {
			return false
		}
	default:
		for i := range n {
			if data[i] != s[i] {
				return false
			}
		}
	}
	if err := c.AdvanceFast(n); err != nil {
		return false
	}
	return true
}

// Line returns the content of the current line up to the cursor position.
// Reconstructed on demand by scanning backward in the buffer.
func (c *UTF8Cursor) Line() string {
	// Scan backward from bufpos to find the start of the current line.
	start := c.bufpos
	for start > 0 && c.buf[start-1] != '\n' {
		start--
	}
	if start == c.bufpos {
		return ""
	}
	return string(c.buf[start:c.bufpos])
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
// ScanNCNameBytes scans an XML NCName and returns the raw bytes (a slice into
// the cursor's buffer). The caller must copy or intern the bytes before the
// next cursor operation that could trigger buffer compaction.
func (c *UTF8Cursor) ScanNCNameBytes() ([]byte, int) {
	if err := c.fillBuffer(1); err != nil {
		return nil, 0
	}

	// Use offset from bufpos to stay safe across fillBuffer compaction.
	off := 0
	// Check first character: must be NameStartChar (without ':').
	b := c.buf[c.bufpos+off]
	if b < 0x80 {
		if (b < 'A' || b > 'Z') && (b < 'a' || b > 'z') && b != '_' {
			return nil, 0
		}
		off++
	} else {
		_ = c.fillBuffer(utf8.UTFMax)
		r, w := utf8.DecodeRune(c.buf[c.bufpos:c.buflen])
		if r == utf8.RuneError || !xmlchar.IsNCNameStartChar(r) {
			return nil, 0
		}
		off += w
	}

	// Scan remaining NameChars.
	nRunes := 1
	for {
		if c.bufpos+off >= c.buflen {
			if c.fillBuffer(off+1) != nil {
				break
			}
			if c.bufpos+off >= c.buflen {
				break
			}
		}
		runLen := scanASCIINameChars(c.buf[c.bufpos+off : c.buflen])
		if runLen > 0 {
			off += runLen
			nRunes += runLen
			if c.bufpos+off >= c.buflen {
				continue
			}
		}
		b = c.buf[c.bufpos+off]
		if b < 0x80 {
			break
		} else {
			_ = c.fillBuffer(off + utf8.UTFMax)
			r, w := utf8.DecodeRune(c.buf[c.bufpos+off : c.buflen])
			if r == utf8.RuneError || !xmlchar.IsNCNameChar(r) {
				break
			}
			off += w
			nRunes++
		}
	}

	return c.buf[c.bufpos : c.bufpos+off], nRunes
}

// ScanQNameBytes scans a common ASCII QName without consuming it.
// It returns the raw prefix and local-name byte slices, the total byte
// length, and ok=true on success. Non-ASCII input, multiple colons, or
// malformed prefix/local parts return ok=false so callers can fall back to
// the full parser path.
func (c *UTF8Cursor) ScanQNameBytes() (prefix, local []byte, nBytes int, ok bool) {
	if err := c.fillBuffer(1); err != nil {
		return nil, nil, 0, false
	}

	off := 0

	b := c.buf[c.bufpos]
	if (b < 'A' || b > 'Z') && (b < 'a' || b > 'z') && b != '_' {
		return nil, nil, 0, false
	}
	off++

	colon := -1
	for {
		if c.bufpos+off >= c.buflen {
			if c.fillBuffer(off+1) != nil {
				break
			}
			if c.bufpos+off >= c.buflen {
				break
			}
		}

		b = c.buf[c.bufpos+off]
		if b >= utf8.RuneSelf {
			return nil, nil, 0, false
		}
		if b == ':' {
			if colon >= 0 {
				return nil, nil, 0, false
			}
			colon = off
			off++

			if c.bufpos+off >= c.buflen {
				if c.fillBuffer(off+1) != nil {
					return nil, nil, 0, false
				}
				if c.bufpos+off >= c.buflen {
					return nil, nil, 0, false
				}
			}

			b = c.buf[c.bufpos+off]
			if (b < 'A' || b > 'Z') && (b < 'a' || b > 'z') && b != '_' {
				return nil, nil, 0, false
			}
			off++
			continue
		}
		if ncNameByteClass[b] != 0 {
			break
		}
		off++
	}

	if colon < 0 {
		return nil, c.buf[c.bufpos : c.bufpos+off], off, true
	}
	return c.buf[c.bufpos : c.bufpos+colon], c.buf[c.bufpos+colon+1 : c.bufpos+off], off, true
}

// ScanSimpleAttrValue scans a simple attribute value (no entities, no special
// whitespace) between the current position and the given quote character.
// Returns the value string and byte count, or ("", 0) if the value contains
// entities or special characters that require the slow path.
// Does NOT consume — caller must call Advance(nBytes) after.
func (c *UTF8Cursor) ScanSimpleAttrValue(quote byte) (string, int) {
	if c.fillBuffer(1) != nil {
		return "", 0
	}

	off := 0
	for {
		if c.bufpos+off >= c.buflen {
			if c.fillBuffer(off+1) != nil {
				return "", 0
			}
			if c.bufpos+off >= c.buflen {
				return "", 0
			}
		}
		b := c.buf[c.bufpos+off]
		if b == quote {
			// End of value.
			return string(c.buf[c.bufpos : c.bufpos+off]), off
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
		} else {
			_ = c.fillBuffer(off + utf8.UTFMax)
			r, w := utf8.DecodeRune(c.buf[c.bufpos+off : c.buflen])
			if w == 0 || r == utf8.RuneError {
				// Invalid or incomplete UTF-8 — fall back to slow path.
				return "", 0
			}
			off += w
		}
	}
}

// ScanCharDataInto scans XML character data with inline EOL normalization.
// Does NOT consume — caller must call AdvanceFast(nBytes) after processing.
// ScanCharDataSlice scans XML character data with EOL normalization, appending
// to dst. Returns the grown slice and the number of bytes consumed. The caller takes
// ownership of the returned slice. Does NOT consume — call AdvanceFast after.
func (c *UTF8Cursor) ScanCharDataSlice(dst []byte) ([]byte, int) {
	if c.fillBuffer(1) != nil {
		return dst, 0
	}

	off := 0
	data := c.buf[c.bufpos:c.buflen]
	dlen := len(data)

	for off < dlen {
		runLen := scanSafeCharDataASCII(data[off:dlen])
		if runLen > 0 {
			dst = append(dst, data[off:off+runLen]...)
			off += runLen
		}
		if off >= dlen {
			if c.fillBuffer(off+1) != nil {
				break
			}
			data = c.buf[c.bufpos:c.buflen]
			dlen = len(data)
			if off >= dlen {
				break
			}
			continue
		}

		b := data[off]
		if b < 0x80 {
			if b == '<' || b == '&' {
				break
			}
			if b == ']' {
				if off+2 >= dlen {
					_ = c.fillBuffer(off + 3)
					data = c.buf[c.bufpos:c.buflen]
					dlen = len(data)
				}
				if off+2 < dlen && data[off+1] == ']' && data[off+2] == '>' {
					break
				}
				dst = append(dst, ']')
				off++
				continue
			}
			if b == '\r' {
				if off+1 >= dlen {
					_ = c.fillBuffer(off + 2)
					data = c.buf[c.bufpos:c.buflen]
					dlen = len(data)
				}
				dst = append(dst, '\n')
				off++
				if off < dlen && data[off] == '\n' {
					off++
				}
				continue
			}
			// Other control char (not 0x9 or 0xa) — stop.
			break
		}
		// Multi-byte UTF-8.
		if dlen-off < utf8.UTFMax {
			_ = c.fillBuffer(off + utf8.UTFMax)
			// Buffer may have moved — re-derive data slice.
			data = c.buf[c.bufpos:c.buflen]
			dlen = len(data)
		}
		r, w := utf8.DecodeRune(data[off:dlen])
		if r == utf8.RuneError || w == 0 {
			break
		}
		if r < 0x20 {
			break
		}
		dst = append(dst, data[off:off+w]...)
		off += w
	}

	return dst, off
}

func (c *UTF8Cursor) ScanCharDataInto(dst *bytes.Buffer) int {
	if c.fillBuffer(1) != nil {
		return 0
	}

	off := 0
	dst.Grow(c.buflen - c.bufpos)

	for {
		if c.bufpos+off >= c.buflen {
			if c.fillBuffer(off+1) != nil {
				break
			}
			if c.bufpos+off >= c.buflen {
				break
			}
		}
		b := c.buf[c.bufpos+off]
		if b < 0x80 {
			if b == '<' || b == '&' {
				break
			}
			if b < 0x20 && b != 0x9 && b != 0xa && b != 0xd {
				break
			}
			if b == ']' {
				if c.bufpos+off+2 >= c.buflen {
					_ = c.fillBuffer(off + 3)
				}
				if c.bufpos+off+2 < c.buflen && c.buf[c.bufpos+off+1] == ']' && c.buf[c.bufpos+off+2] == '>' {
					break
				}
			}
			if b == '\r' {
				if c.bufpos+off+1 >= c.buflen {
					_ = c.fillBuffer(off + 2)
				}
				dst.WriteByte('\n')
				off++
				if c.bufpos+off < c.buflen && c.buf[c.bufpos+off] == '\n' {
					off++
				}
				continue
			}
			dst.WriteByte(b)
			off++
			continue
		}
		// Multi-byte UTF-8. Try to ensure enough bytes.
		if c.buflen-(c.bufpos+off) < utf8.UTFMax {
			_ = c.fillBuffer(off + utf8.UTFMax)
		}
		r, w := utf8.DecodeRune(c.buf[c.bufpos+off : c.buflen])
		if r == utf8.RuneError || w == 0 {
			break
		}
		if r < 0x20 {
			break
		}
		dst.Write(c.buf[c.bufpos+off : c.bufpos+off+w])
		off += w
	}

	return off
}
