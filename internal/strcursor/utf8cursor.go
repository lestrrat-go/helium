package strcursor

import (
	"bytes"
	"io"
	"unicode/utf8"

	"github.com/lestrrat-go/helium/internal/xmlchar"
)

// Byte classes for the ASCII fast path of ScanCharDataSlice.
const (
	// charDataPlain is an ASCII literal character other than a delimiter.
	charDataPlain uint8 = iota
	// charDataStop ends the fast path: a delimiter ('<', '&', ']', CR), a
	// byte >= 0x80, or an ASCII byte outside the XML 1.0 Char production.
	charDataStop
	// charDataRestricted is an ASCII XML 1.0 Char that is not a literal
	// character under the document's version (DEL under XML 1.1). It continues
	// the run, as any Char does, but makes the run invalid.
	charDataRestricted
)

// charDataByteClass classifies every byte per XML version ([0] for XML 1.0,
// [1] for XML 1.1). It is built from xmlchar.IsChar and xmlchar.IsLiteralChar,
// so the scanner and the parser's run validators share one definition of
// literal-character validity.
var charDataByteClass = [2][256]uint8{buildCharDataByteClass(false), buildCharDataByteClass(true)}
var ncNameByteClass = buildNCNameByteClass()

// attrValueByteClass marks, per quote character ([0] for the double quote, [1]
// for the apostrophe), the bytes ScanSimpleAttrValue may skip in bulk as
// charDataPlain: printable ASCII (0x20-0x7F) other than the quote, '&' and '<'.
// Every other byte is charDataStop and is left to the scanner's
// byte-at-a-time classification.
var attrValueByteClass = [2][256]uint8{buildAttrValueByteClass('"'), buildAttrValueByteClass('\'')}

func buildAttrValueByteClass(quote byte) [256]uint8 {
	var tbl [256]uint8
	for i := range 0x100 {
		if i < 0x20 || i >= utf8.RuneSelf {
			tbl[i] = charDataStop
		}
	}
	tbl[quote] = charDataStop
	tbl['&'] = charDataStop
	tbl['<'] = charDataStop
	return tbl
}

func buildCharDataByteClass(xml11 bool) [256]uint8 {
	var tbl [256]uint8
	for i := range utf8.RuneSelf {
		switch r := rune(i); {
		case !xmlchar.IsChar(r):
			tbl[i] = charDataStop
		case !xmlchar.IsLiteralChar(r, xml11):
			tbl[i] = charDataRestricted
		}
	}
	tbl['<'] = charDataStop
	tbl['&'] = charDataStop
	tbl['\r'] = charDataStop
	tbl[']'] = charDataStop
	for i := utf8.RuneSelf; i < 0x100; i++ {
		tbl[i] = charDataStop
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

// scanCharDataASCII returns the length of the leading run of data made of
// plain and restricted ASCII bytes (see charDataByteClass), and the offset of
// the first restricted byte in that run, or -1 when there is none.
func scanCharDataASCII(data []byte, class *[256]uint8) (int, int) {
	bad := -1
	n := scanSafeCharDataASCII(data, class)
	for n < len(data) && class[data[n]] == charDataRestricted {
		if bad < 0 {
			bad = n
		}
		n++
		n += scanSafeCharDataASCII(data[n:], class)
	}
	return n, bad
}

// scanSafeCharDataASCII returns the length of the leading run of data that
// class marks charDataPlain.
func scanSafeCharDataASCII(data []byte, class *[256]uint8) int {
	off := 0
	for off+16 <= len(data) {
		if class[data[off+0]]|
			class[data[off+1]]|
			class[data[off+2]]|
			class[data[off+3]]|
			class[data[off+4]]|
			class[data[off+5]]|
			class[data[off+6]]|
			class[data[off+7]]|
			class[data[off+8]]|
			class[data[off+9]]|
			class[data[off+10]]|
			class[data[off+11]]|
			class[data[off+12]]|
			class[data[off+13]]|
			class[data[off+14]]|
			class[data[off+15]] != charDataPlain {
			break
		}
		off += 16
	}
	for off < len(data) && class[data[off]] == charDataPlain {
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
	buf     []byte
	buflen  int
	bufpos  int
	column  int
	in      io.Reader
	lineno  int
	readErr error // sticky non-EOF read error (e.g. a transcoding/decode error)
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

// LineContextMax is the most bytes of the current line that Line returns: the
// text between the last LF and the cursor, cut to its last LineContextMax bytes.
// The cursor keeps those bytes buffered across every refill, so Line returns the
// same text however the input arrives.
const LineContextMax = 1024

// fillBuffer ensures at least minBytes are available from bufpos.
func (c *UTF8Cursor) fillBuffer(minBytes int) error {
	avail := c.buflen - c.bufpos
	if avail >= minBytes {
		return nil
	}

	// Compact only when the request does not fit after bufpos. Reading into
	// the free space after the buffered bytes until then keeps a reader that
	// delivers a few bytes per Read from moving the kept line text on every
	// refill.
	if c.bufpos+minBytes > len(c.buf) {
		c.compact()
	}

	// Grow buffer if needed.
	if c.bufpos+minBytes > len(c.buf) {
		newBuf := make([]byte, (c.bufpos+minBytes)*2)
		copy(newBuf, c.buf[:c.buflen])
		c.buf = newBuf
	}

	// Read until we have enough.
	zeroProgress := 0
	for c.buflen-c.bufpos < minBytes {
		n, err := c.in.Read(c.buf[c.buflen:])
		c.buflen += n
		// A single (0, nil) read is not fatal: io.Reader permits a reader to
		// return no data and no error while it waits for more input. A
		// streaming wrapper that holds an incomplete multibyte rune at a chunk
		// boundary (e.g. html's utf8SanitizeReader) legitimately does this.
		// Only a reader that makes no progress for many CONSECUTIVE reads is
		// genuinely stuck, so retry up to maxZeroProgressReads and reset the
		// counter whenever any bytes arrive; bail with io.ErrNoProgress once the
		// bound is hit so a pathological (0, nil)-forever reader fails fast
		// instead of hanging the parser.
		if n == 0 && err == nil {
			zeroProgress++
			if zeroProgress >= maxZeroProgressReads {
				c.readErr = io.ErrNoProgress
				return io.ErrNoProgress
			}
			continue
		}
		zeroProgress = 0
		if err != nil {
			// Remember a genuine decode/transcoding error (anything other than
			// a clean EOF) so the parser can distinguish malformed input from a
			// normal end-of-stream. Done() treats both as "no more data". The
			// error must be recorded even when this Read also returned data
			// (n > 0), because a small input may deliver its decoded bytes and
			// the decode error together in a single Read.
			if err != io.EOF && c.readErr == nil {
				c.readErr = err
			}
			if c.buflen-c.bufpos >= minBytes {
				return nil
			}
			return err
		}
	}
	return nil
}

// compact moves the text Line needs and the unconsumed bytes to the front of
// the buffer, dropping the consumed bytes before them.
func (c *UTF8Cursor) compact() {
	keep := c.lineStart()
	if keep == 0 {
		return
	}
	c.buflen = copy(c.buf, c.buf[keep:c.buflen])
	c.bufpos -= keep
}

// lineStart returns the buffer offset where Line's text begins: just after the
// last LF before the cursor, or LineContextMax bytes before the cursor when the
// line is longer. compact never drops a byte at or after this offset, so the
// buffer holds the same text here whatever sizes the reads returned.
func (c *UTF8Cursor) lineStart() int {
	lo := max(c.bufpos-LineContextMax, 0)
	if i := bytes.LastIndexByte(c.buf[lo:c.bufpos], '\n'); i >= 0 {
		return lo + i + 1
	}
	return lo
}

// Err returns a sticky non-EOF read error encountered while filling the buffer,
// such as a transcoding/decode error from an underlying encoding decoder. It
// returns nil if the stream ended cleanly.
func (c *UTF8Cursor) Err() error {
	return c.readErr
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
// The buffered case is small enough to inline at a concrete call site; a
// position past the buffered bytes goes through peekAtSlow.
func (c *UTF8Cursor) PeekAt(offset int) byte {
	if pos := c.bufpos + offset; pos < c.buflen {
		return c.buf[pos]
	}
	return c.peekAtSlow(offset)
}

// peekAtSlow is PeekAt for a position past the buffered bytes: it refills the
// buffer and returns 0 when the input ends first.
func (c *UTF8Cursor) peekAtSlow(offset int) byte {
	if c.fillBuffer(offset+1) != nil {
		return 0
	}
	pos := c.bufpos + offset
	if pos >= c.buflen {
		return 0
	}
	return c.buf[pos]
}

// HasByteAt reports whether a byte is available at offset bytes from the
// current position. Unlike PeekAt (which returns 0 both for a genuine NUL byte
// and for a position past EOF), this lets callers distinguish a real U+0000 in
// the input from end-of-stream.
func (c *UTF8Cursor) HasByteAt(offset int) bool {
	if c.bufpos+offset < c.buflen {
		return true
	}
	return c.hasByteAtSlow(offset)
}

// hasByteAtSlow is HasByteAt for a position past the buffered bytes. It is
// kept out of line: inlined, it would push HasByteAt over the inlining budget.
//
//go:noinline
func (c *UTF8Cursor) hasByteAtSlow(offset int) bool {
	if c.fillBuffer(offset+1) != nil {
		return false
	}
	return c.bufpos+offset < c.buflen
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
// Consuming one buffered byte other than a newline, the parser's most frequent
// advance, returns before any refill or bulk scan; everything else goes
// through advanceSlow.
func (c *UTF8Cursor) Advance(n int) error {
	if n == 1 && c.bufpos < c.buflen && c.buf[c.bufpos] != '\n' {
		c.bufpos++
		c.column++
		return nil
	}
	return c.advanceSlow(n)
}

// AdvanceNoNewline consumes n bytes the caller has already scanned and proven
// free of '\n' (a name, or an attribute value the simple scan accepted), so the
// column moves by n and no byte is examined again.
func (c *UTF8Cursor) AdvanceNoNewline(n int) error {
	if c.buflen-c.bufpos >= n {
		c.bufpos += n
		c.column += n
		return nil
	}
	return c.AdvanceFast(n)
}

func (c *UTF8Cursor) advanceSlow(n int) error {
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
	if n <= advanceScanInline {
		// A short run (a name, an attribute value, the whitespace between two
		// tags) costs less to walk once than to hand to two vectorized
		// searches, each with its own call and setup cost.
		lines := 0
		last := -1
		for i, b := range segment {
			if b == '\n' {
				lines++
				last = i
			}
		}
		if last >= 0 {
			c.lineno += lines
			c.column = len(segment) - last
		} else {
			c.column += n
		}
		c.bufpos = end
		return nil
	}
	if idx := bytes.LastIndexByte(segment, '\n'); idx >= 0 {
		c.lineno += bytes.Count(segment[:idx], []byte{'\n'}) + 1
		c.column = len(segment) - idx
	} else {
		c.column += n
	}
	c.bufpos = end
	return nil
}

// advanceScanInline is the run length up to which AdvanceFast counts newlines
// with a plain loop instead of bytes.LastIndexByte and bytes.Count.
const advanceScanInline = 32

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

// Line returns the content of the current line up to the cursor position, cut
// to its last LineContextMax bytes. A cut that lands inside a multi-byte
// character drops that character's remaining bytes. The text is found on demand
// in the buffer, which compact keeps it in, so it does not depend on how the
// input was split across reads.
func (c *UTF8Cursor) Line() string {
	start := c.lineStart()
	if c.bufpos-start == LineContextMax {
		for i := 0; i < utf8.UTFMax-1 && start < c.bufpos && !utf8.RuneStart(c.buf[start]); i++ {
			start++
		}
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
		if (r == utf8.RuneError && w == 1) || !xmlchar.IsNCNameStartChar(r) {
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
			if (r == utf8.RuneError && w == 1) || !xmlchar.IsNCNameChar(r) {
				break
			}
			off += w
			nRunes++
		}
	}

	return c.buf[c.bufpos : c.bufpos+off], nRunes
}

// ScanQNameBytes scans a common ASCII QName without consuming it.
// It returns the raw bytes of the whole QName as written, the offset of its
// colon (-1 when the name has no prefix), and ok=true on success; the prefix
// is name[:colon] and the local name is name[colon+1:]. Non-ASCII input,
// multiple colons, or malformed prefix/local parts return ok=false so callers
// can fall back to the full parser path.
func (c *UTF8Cursor) ScanQNameBytes() (name []byte, colon int, ok bool) {
	if err := c.fillBuffer(1); err != nil {
		return nil, -1, false
	}

	off := 0

	b := c.buf[c.bufpos]
	if (b < 'A' || b > 'Z') && (b < 'a' || b > 'z') && b != '_' {
		return nil, -1, false
	}
	off++

	colon = -1
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
			return nil, -1, false
		}
		if b == ':' {
			if colon >= 0 {
				return nil, -1, false
			}
			colon = off
			off++

			if c.bufpos+off >= c.buflen {
				if c.fillBuffer(off+1) != nil {
					return nil, -1, false
				}
				if c.bufpos+off >= c.buflen {
					return nil, -1, false
				}
			}

			b = c.buf[c.bufpos+off]
			if (b < 'A' || b > 'Z') && (b < 'a' || b > 'z') && b != '_' {
				return nil, -1, false
			}
			off++
			continue
		}
		if ncNameByteClass[b] != 0 {
			break
		}
		off++
	}

	return c.buf[c.bufpos : c.bufpos+off], colon, true
}

// ScanSimpleAttrValue scans a simple attribute value (no entities, no special
// whitespace) between the current position and the given quote character.
// Returns the value string and byte count, or ("", 0) if the value contains
// entities or special characters that require the slow path.
// Does NOT consume — caller must call Advance(nBytes) after.
//
// When maxBytes > 0 the scan bails (returning "", 0) once it has consumed more
// than maxBytes input bytes, so a giant value falls back to the slow path,
// which enforces the node-content cap and reports the error. This bounds the
// cursor's internal buffer growth instead of materializing the whole value.
// maxBytes <= 0 means unbounded.
func (c *UTF8Cursor) ScanSimpleAttrValue(quote byte, maxBytes int) (string, int) {
	if c.fillBuffer(1) != nil {
		return "", 0
	}

	class := &attrValueByteClass[0]
	if quote == '\'' {
		class = &attrValueByteClass[1]
	}
	off := 0
	for {
		if maxBytes > 0 && off > maxBytes {
			// Over the caller's byte budget — defer to the slow path.
			return "", 0
		}
		if c.bufpos+off >= c.buflen {
			if c.fillBuffer(off+1) != nil {
				return "", 0
			}
			if c.bufpos+off >= c.buflen {
				return "", 0
			}
		}
		// Skip the buffered run of printable ASCII other than the quote, '&'
		// and '<' sixteen bytes at a time; the loop below would accept each of
		// those bytes one by one. The budget is checked again before the byte
		// that ends the run is looked at, as the byte-at-a-time walk would
		// have checked it before each byte.
		off += scanSafeCharDataASCII(c.buf[c.bufpos+off:c.buflen], class)
		if maxBytes > 0 && off > maxBytes {
			return "", 0
		}
		if c.bufpos+off >= c.buflen {
			continue
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
			if b < 0x20 {
				// Tab, \r, \n, and other control chars need attribute-value
				// normalization (whitespace -> space) — defer to the slow path.
				return "", 0
			}
			off++
		} else {
			_ = c.fillBuffer(off + utf8.UTFMax)
			r, w := utf8.DecodeRune(c.buf[c.bufpos+off : c.buflen])
			if w == 0 || (r == utf8.RuneError && w == 1) {
				// Invalid or incomplete UTF-8 — fall back to slow path.
				// (A real U+FFFD decodes as RuneError with width 3 and is valid.)
				return "", 0
			}
			if !xmlchar.IsChar(r) {
				// XML-forbidden char — fall back to the slow path, which
				// reports the invalid character.
				return "", 0
			}
			off += w
		}
	}
}

// ScanCharDataSlice scans XML character data with EOL normalization, appending
// to dst. Returns the grown slice, the number of bytes consumed, and whether
// every scanned character may appear literally under the document's XML
// version (xml11 selects XML 1.1, otherwise XML 1.0). The caller takes ownership
// of the returned slice. Does NOT consume — call AdvanceFast after.
//
// The scan validates the run as it goes, so callers need not re-check the
// returned bytes. Where the run stops does not depend on xml11: it stops at '<',
// '&', "]]>", invalid UTF-8, and any character outside the XML 1.0 Char
// production (xmlchar.IsChar), leaving that byte for the caller to diagnose. A
// character that is an XML 1.0 Char but not a literal character under xml11
// (an XML 1.1 RestrictedChar such as DEL or U+0080) does not stop the run: it is
// scanned like any other character and the run is reported not valid
// (valid == false), so the caller rejects the whole run at its start position.
// Under XML 1.0 valid is always true, since every Char is a literal character.
//
// When maxBytes > 0 the scan stops once that many input bytes have been
// consumed (always on a UTF-8 character boundary; a single rune wider than
// maxBytes is still returned whole so progress is guaranteed). This bounds both
// the returned slice and the cursor's internal buffer, letting callers deliver a
// long delimiter-free run in fixed-size chunks instead of materializing it all.
// maxBytes <= 0 means unbounded (scan the whole run up to the next delimiter).
// valid covers only the bytes this call scanned.
func (c *UTF8Cursor) ScanCharDataSlice(dst []byte, maxBytes int, xml11 bool) ([]byte, int, bool) {
	if c.fillBuffer(1) != nil {
		return dst, 0, true
	}

	class := &charDataByteClass[0]
	if xml11 {
		class = &charDataByteClass[1]
	}
	valid := true
	off := 0
	data := c.buf[c.bufpos:c.buflen]
	dlen := len(data)

	for off < dlen {
		if maxBytes > 0 && off >= maxBytes {
			break
		}
		runLen, bad := scanCharDataASCII(data[off:dlen], class)
		if maxBytes > 0 && off+runLen > maxBytes {
			// Cap the ASCII run at the byte budget. ASCII bytes are single-byte,
			// so this never splits a multi-byte character.
			runLen = maxBytes - off
		}
		if bad >= 0 && bad < runLen {
			valid = false
		}
		if runLen > 0 {
			dst = append(dst, data[off:off+runLen]...)
			off += runLen
		}
		if maxBytes > 0 && off >= maxBytes {
			break
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
		if b < utf8.RuneSelf {
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
		if w == 0 || (r == utf8.RuneError && w == 1) {
			// Invalid/incomplete UTF-8. A real U+FFFD decodes as RuneError
			// with width 3 and is a valid XML char, so let IsChar judge it.
			break
		}
		if !xmlchar.IsChar(r) {
			break
		}
		if maxBytes > 0 && off > 0 && off+w > maxBytes {
			// Adding this rune would exceed the byte budget; stop on the
			// character boundary. off > 0 guarantees progress for a lone rune
			// wider than maxBytes (it is returned whole on the next call).
			break
		}
		// Every XML 1.0 Char is a literal character in XML 1.0, so only XML
		// 1.1 needs the second check: a RestrictedChar (U+0080-U+0084,
		// U+0086-U+009F) continues the run but makes it invalid.
		if xml11 && !xmlchar.IsLiteralChar(r, true) {
			valid = false
		}
		dst = append(dst, data[off:off+w]...)
		off += w
	}

	return dst, off, valid
}

// ScanCharDataInto scans XML character data with inline EOL normalization.
// Does NOT consume — caller must call AdvanceFast(nBytes) after processing.
// Unlike ScanCharDataSlice it applies only the XML 1.0 Char rules, so callers
// must validate the returned bytes against the document's XML version.
func (c *UTF8Cursor) ScanCharDataInto(dst *bytes.Buffer, maxBytes int) int {
	if c.fillBuffer(1) != nil {
		return 0
	}

	off := 0
	dst.Grow(c.buflen - c.bufpos)

	// The loop condition bounds dst to maxBytes (maxBytes <= 0 = unbounded).
	for maxBytes <= 0 || dst.Len() < maxBytes {
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
		if w == 0 || (r == utf8.RuneError && w == 1) {
			// Invalid/incomplete UTF-8. A real U+FFFD decodes as RuneError
			// with width 3 and is a valid XML char, so let IsChar judge it.
			break
		}
		if !xmlchar.IsChar(r) {
			break
		}
		dst.Write(c.buf[c.bufpos+off : c.bufpos+off+w])
		off += w
	}

	return off
}
