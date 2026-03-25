// Package strcursor provides buffered cursor types for reading runes and
// bytes from an io.Reader.  The RuneCursor uses a ring buffer so that
// PeekN(k) is O(1) instead of the O(k) linked-list walk in the original
// external package.
package strcursor

import (
	"bytes"
	"errors"
	"io"
	"unicode/utf8"
)

// Cursor is the byte-oriented interface for reading XML input.
// All offsets and counts are in bytes, not runes.
type Cursor interface {
	io.Reader

	// Advance consumes n bytes from the input, updating line/column tracking.
	Advance(int) error
	// AdvanceFast consumes n bytes, updating only line numbers (not the line buffer).
	AdvanceFast(int) error
	Column() int
	Consume([]byte) bool
	ConsumeString(string) bool
	Done() bool
	HasPrefix([]byte) bool
	HasPrefixString(string) bool
	Line() string
	LineNumber() int
	// Peek returns the byte at the current position, or 0 if at EOF.
	Peek() byte
	// PeekAt returns the byte at offset bytes from the current position (0-indexed).
	PeekAt(int) byte
	// PeekRune decodes and returns the rune at the current position.
	// Use for non-ASCII content (byte >= 0x80). Returns utf8.RuneError on error.
	PeekRune() rune
	// PeekString returns n bytes from the current position as a string.
	PeekString(int) string
	// ScanCharDataInto scans XML character data into dst with EOL normalization.
	// Returns the number of bytes consumed. Does not advance — call AdvanceFast after.
	ScanCharDataInto(dst *bytes.Buffer) int
	Unused() io.Reader
}

// Unused wraps remaining buffered bytes plus the underlying reader.
type Unused struct {
	unused []byte
	rdr    io.Reader
}

func (u *Unused) Read(b []byte) (int, error) {
	if len(u.unused) > 0 {
		n := copy(b, u.unused)
		u.unused = u.unused[n:]
		return n, nil
	}
	return u.rdr.Read(b)
}

// ---------------------------------------------------------------------------
// RuneCursor — ring-buffer backed
// ---------------------------------------------------------------------------

const defaultRuneBufSize = 4096
const defaultByteBufSize = 4096

// runeEntry stores a decoded rune and its byte width.
type runeEntry struct {
	val   rune
	width int
}

// roundUpPow2 rounds n up to the next power of 2 (or n itself if already a power of 2).
func roundUpPow2(n int) int {
	if n <= 0 {
		return 1
	}
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n |= n >> 32
	n++
	return n
}

// RuneCursor reads runes from an io.Reader using a circular buffer so that
// PeekN is O(1).
type RuneCursor struct {
	// ring buffer for decoded runes
	ring    []runeEntry
	head    int // index of first valid rune
	count   int // number of valid runes in ring
	ringCap int // len(ring), always a power of 2
	ringMask int // ringCap - 1, used for fast modulo

	// byte-level read buffer for decoding
	raw    []byte
	rawLen int // valid bytes in raw
	rawPos int // consumed position in raw

	in     io.Reader
	line   bytes.Buffer
	lineno int
	column int
	nread  int
	eof    bool // underlying reader exhausted
}

// NewRuneCursor creates a RuneCursor. An optional bufsize argument controls
// the initial ring capacity (default 4096 runes).
func NewRuneCursor(r io.Reader, bufsize ...int) *RuneCursor {
	n := defaultRuneBufSize
	if len(bufsize) > 0 && bufsize[0] > 0 {
		n = bufsize[0]
	}
	n = roundUpPow2(n)
	return &RuneCursor{
		ring:     make([]runeEntry, n),
		ringCap:  n,
		ringMask: n - 1,
		raw:      make([]byte, 4096),
		rawLen:   0,
		rawPos:   0,
		in:       r,
		lineno:   1,
		column:   1,
	}
}

// ensure ensures at least n runes are available in the ring buffer.
func (c *RuneCursor) ensure(n int) error {
	for c.count < n {
		// Try to decode from raw buffer first.
		decoded := c.decodeRaw()
		if decoded > 0 {
			continue
		}
		// Need more bytes from the reader.
		if c.eof {
			return io.EOF
		}
		if err := c.fillRaw(); err != nil {
			c.eof = true
			// Try one more decode pass in case fillRaw got partial bytes.
			if c.decodeRaw() > 0 {
				continue
			}
			return err
		}
	}
	return nil
}

// fillRaw reads more bytes from the underlying reader into the raw buffer.
func (c *RuneCursor) fillRaw() error {
	// Compact: move unconsumed bytes to front.
	if c.rawPos > 0 {
		remaining := c.rawLen - c.rawPos
		copy(c.raw, c.raw[c.rawPos:c.rawLen])
		c.rawLen = remaining
		c.rawPos = 0
	}
	// Grow raw buffer if full.
	if c.rawLen == len(c.raw) {
		newBuf := make([]byte, len(c.raw)*2)
		copy(newBuf, c.raw[:c.rawLen])
		c.raw = newBuf
	}
	n, err := c.in.Read(c.raw[c.rawLen:])
	c.rawLen += n
	if n == 0 && err != nil {
		return err
	}
	return nil
}

// decodeRaw decodes as many runes as possible from the raw byte buffer and
// appends them to the ring buffer.  Returns the number decoded.
func (c *RuneCursor) decodeRaw() int {
	decoded := 0
	mask := c.ringMask
	rawPos := c.rawPos
	rawLen := c.rawLen
	raw := c.raw
	count := c.count

	for rawPos < rawLen {
		// ASCII fast path: bytes < 0x80 are single-byte runes.
		b := raw[rawPos]
		if b < 0x80 {
			// Grow ring if needed.
			if count == c.ringCap {
				c.rawPos = rawPos
				c.count = count
				c.growRing()
				mask = c.ringMask
			}
			idx := (c.head + count) & mask
			c.ring[idx] = runeEntry{val: rune(b), width: 1}
			rawPos++
			count++
			decoded++
			continue
		}

		// Multi-byte UTF-8.
		r, w := utf8.DecodeRune(raw[rawPos:rawLen])
		if r == utf8.RuneError && w <= 1 {
			// Possibly incomplete rune at end of buffer.
			if rawLen-rawPos < utf8.UTFMax && !c.eof {
				break // wait for more bytes
			}
			// Genuinely invalid — skip 1 byte.
			if w == 0 {
				break
			}
		}
		rawPos += w
		// Grow ring if needed.
		if count == c.ringCap {
			c.rawPos = rawPos
			c.count = count
			c.growRing()
			mask = c.ringMask
		}
		idx := (c.head + count) & mask
		c.ring[idx] = runeEntry{val: r, width: w}
		count++
		decoded++
	}

	c.rawPos = rawPos
	c.count = count
	return decoded
}

func (c *RuneCursor) growRing() {
	newCap := c.ringCap * 2
	newRing := make([]runeEntry, newCap)
	for i := 0; i < c.count; i++ {
		newRing[i] = c.ring[(c.head+i)&c.ringMask]
	}
	c.ring = newRing
	c.head = 0
	c.ringCap = newCap
	c.ringMask = newCap - 1
}

func (c *RuneCursor) Done() bool {
	if c.count > 0 {
		return false
	}
	return c.ensure(1) != nil
}

// Peek returns the first buffered rune without consuming it.
func (c *RuneCursor) Peek() rune {
	// Fast path: data already buffered.
	if c.count >= 1 {
		return c.ring[c.head].val
	}
	if err := c.ensure(1); err != nil {
		return utf8.RuneError
	}
	return c.ring[c.head].val
}

// PeekN returns the n-th rune (1-indexed) without consuming. O(1).
func (c *RuneCursor) PeekN(n int) rune {
	// Fast path: data already buffered (common case).
	if c.count >= n {
		return c.ring[(c.head+n-1)&c.ringMask].val
	}
	if err := c.ensure(n); err != nil {
		return utf8.RuneError
	}
	return c.ring[(c.head+n-1)&c.ringMask].val
}

// PeekString returns the first n runes as a string without consuming them.
// More efficient than building the string via repeated PeekN + buffer writes.
func (c *RuneCursor) PeekString(n int) string {
	if err := c.ensure(n); err != nil {
		return ""
	}
	mask := c.ringMask
	ring := c.ring
	head := c.head

	// Fast path: check if all runes are single-byte ASCII (width == 1).
	// This avoids utf8.EncodeRune overhead for the common case of XML names.
	allASCII := true
	for i := 0; i < n; i++ {
		if ring[(head+i)&mask].width != 1 {
			allASCII = false
			break
		}
	}
	if allASCII {
		var stackBuf [128]byte
		var buf []byte
		if n <= len(stackBuf) {
			buf = stackBuf[:n]
		} else {
			buf = make([]byte, n)
		}
		for i := 0; i < n; i++ {
			buf[i] = byte(ring[(head+i)&mask].val)
		}
		return string(buf)
	}

	// Slow path: multi-byte runes present.
	totalBytes := 0
	for i := 0; i < n; i++ {
		totalBytes += ring[(head+i)&mask].width
	}
	var stackBuf [128]byte
	var buf []byte
	if totalBytes <= len(stackBuf) {
		buf = stackBuf[:totalBytes]
	} else {
		buf = make([]byte, totalBytes)
	}
	pos := 0
	for i := 0; i < n; i++ {
		pos += utf8.EncodeRune(buf[pos:], ring[(head+i)&mask].val)
	}
	return string(buf)
}

func (c *RuneCursor) Cur() rune {
	if c.count < 1 {
		if err := c.ensure(1); err != nil {
			return utf8.RuneError
		}
	}
	r := c.ring[c.head].val
	// Inline advance(1) for speed.
	c.nread += c.ring[c.head].width
	if r == '\n' {
		c.lineno++
		c.line.Reset()
		c.column = 1
	} else {
		c.column++
	}
	if r < utf8.RuneSelf {
		c.line.WriteByte(byte(r))
	} else {
		c.line.WriteRune(r)
	}
	c.head = (c.head + 1) & c.ringMask
	c.count--
	return r
}

func (c *RuneCursor) Advance(n int) error {
	if c.count < n {
		if err := c.ensure(n); err != nil {
			return err
		}
	}
	mask := c.ringMask
	head := c.head
	ring := c.ring
	for i := 0; i < n; i++ {
		e := ring[head]
		c.nread += e.width
		if e.val == '\n' {
			c.lineno++
			c.line.Reset()
			c.column = 1
		} else {
			c.column++
		}
		if e.val < utf8.RuneSelf {
			c.line.WriteByte(byte(e.val))
		} else {
			c.line.WriteRune(e.val)
		}
		head = (head + 1) & mask
	}
	c.head = head
	c.count -= n
	return nil
}

// AdvanceFast advances by n runes, updating only line numbers and byte counts.
// It skips the per-character line buffer tracking that Advance does, making it
// faster for bulk character data consumption where the line context is not
// needed for error reporting.
func (c *RuneCursor) AdvanceFast(n int) error {
	if c.count < n {
		if err := c.ensure(n); err != nil {
			return err
		}
	}
	mask := c.ringMask
	head := c.head
	ring := c.ring
	totalBytes := 0
	lastNewline := -1
	for i := 0; i < n; i++ {
		e := ring[head]
		totalBytes += e.width
		if e.val == '\n' {
			c.lineno++
			lastNewline = i
		}
		head = (head + 1) & mask
	}
	c.nread += totalBytes
	if lastNewline >= 0 {
		c.column = n - lastNewline
		c.line.Reset()
	} else {
		c.column += n
	}
	c.head = head
	c.count -= n
	return nil
}

// ScanCharDataInto scans contiguous XML character data from the ring buffer
// and writes it into dst with XML §2.11 EOL normalization applied inline
// (\r\n → \n, lone \r → \n). It stops at '<', '&', ']]>', or any non-XML-char.
// Returns the rune count consumed. The caller should call AdvanceFast(nRunes).
func (c *RuneCursor) ScanCharDataInto(dst *bytes.Buffer) int {
	if c.count == 0 {
		if err := c.ensure(1); err != nil {
			return 0
		}
	}

	mask := c.ringMask
	ring := c.ring
	head := c.head
	count := c.count
	nRunes := 0

	// Pre-grow to avoid repeated buffer growth.
	dst.Grow(count)

	for nRunes < count {
		e := ring[(head+nRunes)&mask]
		r := e.val
		if r == '<' || r == '&' || r == utf8.RuneError {
			break
		}
		u := uint32(r)
		if u < 0x20 && u != 0x9 && u != 0xa && u != 0xd {
			break
		}
		if r == ']' && nRunes+2 < count {
			if ring[(head+nRunes+1)&mask].val == ']' && ring[(head+nRunes+2)&mask].val == '>' {
				break
			}
		}
		// XML §2.11 EOL normalization: \r\n → \n, lone \r → \n.
		if r == '\r' {
			dst.WriteByte('\n')
			// Skip the following \n if present (\r\n pair).
			if nRunes+1 < count && ring[(head+nRunes+1)&mask].val == '\n' {
				nRunes++ // consume the \n too
			}
			nRunes++
			continue
		}
		if e.width == 1 {
			dst.WriteByte(byte(r))
		} else {
			dst.WriteRune(r)
		}
		nRunes++
	}

	return nRunes
}

func (c *RuneCursor) hasPrefix(s string, n int, consume bool) bool {
	if c.count < n {
		if err := c.ensure(n); err != nil {
			return false
		}
	}
	mask := c.ringMask
	pos := 0
	for i := 0; i < n; i++ {
		r, w := utf8.DecodeRuneInString(s[pos:])
		if r == utf8.RuneError {
			return false
		}
		idx := (c.head + i) & mask
		if c.ring[idx].val != r {
			return false
		}
		pos += w
	}
	if consume {
		_ = c.Advance(n)
	}
	return true
}

func (c *RuneCursor) HasPrefix(b []byte) bool {
	return c.HasPrefixString(string(b))
}

func (c *RuneCursor) HasPrefixString(s string) bool {
	n := utf8.RuneCountInString(s)
	return c.hasPrefix(s, n, false)
}

func (c *RuneCursor) Consume(b []byte) bool {
	return c.ConsumeString(string(b))
}

func (c *RuneCursor) ConsumeString(s string) bool {
	n := utf8.RuneCountInString(s)
	return c.hasPrefix(s, n, true)
}

func (c *RuneCursor) Line() string {
	return c.line.String()
}

func (c *RuneCursor) LineNumber() int {
	return c.lineno
}

func (c *RuneCursor) Column() int {
	return c.column
}

func (c *RuneCursor) Unused() io.Reader {
	ret := &Unused{rdr: c.in}
	// Reconstruct unconsumed bytes from ring buffer + raw buffer.
	var buf bytes.Buffer
	mask := c.ringMask
	for i := 0; i < c.count; i++ {
		idx := (c.head + i) & mask
		buf.WriteRune(c.ring[idx].val)
	}
	if c.rawPos < c.rawLen {
		buf.Write(c.raw[c.rawPos:c.rawLen])
	}
	if buf.Len() > 0 {
		ret.unused = buf.Bytes()
	}
	return ret
}

func (c *RuneCursor) Read(buf []byte) (int, error) {
	// Drain any buffered runes first.
	nread := 0
	mask := c.ringMask
	for c.count > 0 && nread < len(buf) {
		e := c.ring[c.head]
		w := utf8.EncodeRune(buf[nread:], e.val)
		if nread+w > len(buf) {
			break
		}
		nread += w
		c.head = (c.head + 1) & mask
		c.count--
	}
	if nread > 0 {
		if nread >= len(buf) {
			return nread, nil
		}
		n, err := c.in.Read(buf[nread:])
		return nread + n, err
	}
	return c.in.Read(buf)
}

// ---------------------------------------------------------------------------
// ByteCursor — simple byte buffer (no linked list, already O(1))
// ---------------------------------------------------------------------------

// ByteCursor reads bytes from an io.Reader.
type ByteCursor struct {
	buf    []byte
	buflen int
	bufpos int
	column int
	in     io.Reader
	line   bytes.Buffer
	lineno int
}

func NewByteCursor(r io.Reader, nn ...int) *ByteCursor {
	n := defaultByteBufSize
	if len(nn) > 0 && nn[0] > 0 {
		n = nn[0]
	}
	buf := make([]byte, n)
	return &ByteCursor{
		buf:    buf,
		buflen: n,
		bufpos: n, // force fill on first read
		column: 1,
		in:     r,
		lineno: 1,
	}
}

func (c *ByteCursor) fillBuffer(n int) error {
	if c.buflen-c.bufpos >= n {
		return nil
	}

	// Compact remaining bytes to front.
	remaining := c.buflen - c.bufpos
	if c.bufpos > 0 && remaining > 0 {
		copy(c.buf, c.buf[c.bufpos:c.buflen])
	}
	c.bufpos = 0

	// Grow buffer if needed.
	if n > len(c.buf) {
		newBuf := make([]byte, n*2)
		copy(newBuf, c.buf[:remaining])
		c.buf = newBuf
	}

	// Clear the rest.
	for i := remaining; i < len(c.buf); i++ {
		c.buf[i] = 0
	}

	nread, err := c.in.Read(c.buf[remaining:])
	c.buflen = nread + remaining
	if nread == 0 && err != nil {
		if remaining >= n {
			return nil
		}
		return err
	}
	if c.buflen < n {
		return errors.New("fillBuffer request exceeds available data")
	}
	return nil
}

func (c *ByteCursor) Done() bool {
	return c.fillBuffer(1) != nil
}

// Peek returns the byte at the current position, or 0 if at EOF.
func (c *ByteCursor) Peek() byte {
	if c.bufpos >= c.buflen {
		if c.fillBuffer(1) != nil {
			return 0
		}
	}
	return c.buf[c.bufpos]
}

// PeekAt returns the byte at offset bytes from the current position (0-indexed).
func (c *ByteCursor) PeekAt(offset int) byte {
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
// For ByteCursor, each byte is treated as a single character.
func (c *ByteCursor) PeekRune() rune {
	b := c.Peek()
	if b == 0 {
		return utf8.RuneError
	}
	return rune(b)
}

// PeekString returns the first n bytes as a string without consuming them.
func (c *ByteCursor) PeekString(n int) string {
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

func (c *ByteCursor) Advance(n int) error {
	if err := c.fillBuffer(n); err != nil {
		return err
	}
	if i := bytes.IndexByte(c.buf[c.bufpos:c.bufpos+n], '\n'); i > -1 {
		c.lineno++
		c.column = n - i + 1
		c.line.Reset()
		c.line.Write(c.buf[c.bufpos+i : c.bufpos+n])
	} else {
		c.column += n
		c.line.Write(c.buf[c.bufpos : c.bufpos+n])
	}
	c.bufpos += n
	return nil
}

// AdvanceFast advances by n bytes, skipping line buffer tracking.
func (c *ByteCursor) AdvanceFast(n int) error {
	return c.Advance(n)
}

// ScanCharDataInto scans XML character data into dst with EOL normalization.
func (c *ByteCursor) ScanCharDataInto(dst *bytes.Buffer) int {
	if c.fillBuffer(1) != nil {
		return 0
	}
	i := 0
	for c.bufpos+i < c.buflen {
		b := c.buf[c.bufpos+i]
		if b == '<' || b == '&' {
			break
		}
		if b < 0x20 && b != 0x9 && b != 0xa && b != 0xd {
			break
		}
		if b == ']' && c.bufpos+i+2 < c.buflen && c.buf[c.bufpos+i+1] == ']' && c.buf[c.bufpos+i+2] == '>' {
			break
		}
		if b == '\r' {
			dst.WriteByte('\n')
			if c.bufpos+i+1 < c.buflen && c.buf[c.bufpos+i+1] == '\n' {
				i++ // skip the \n in \r\n
			}
			i++
			continue
		}
		dst.WriteByte(b)
		i++
	}
	return i
}

func (c *ByteCursor) hasPrefix(s []byte, consume bool) bool {
	n := len(s)
	if err := c.fillBuffer(n); err != nil {
		return false
	}
	if !bytes.HasPrefix(c.buf[c.bufpos:c.buflen], s) {
		return false
	}
	if consume {
		c.bufpos += n
	}
	return true
}

func (c *ByteCursor) HasPrefix(s []byte) bool {
	return c.hasPrefix(s, false)
}

func (c *ByteCursor) Consume(s []byte) bool {
	return c.hasPrefix(s, true)
}

func (c *ByteCursor) HasPrefixString(s string) bool {
	return c.hasPrefix([]byte(s), false)
}

func (c *ByteCursor) ConsumeString(s string) bool {
	return c.hasPrefix([]byte(s), true)
}

func (c *ByteCursor) Line() string {
	return c.line.String()
}

func (c *ByteCursor) LineNumber() int {
	return c.lineno
}

func (c *ByteCursor) Column() int {
	return c.column
}

func (c *ByteCursor) Unused() io.Reader {
	ret := &Unused{rdr: c.in}
	if buf := c.buf[c.bufpos:c.buflen]; len(buf) > 0 {
		ret.unused = make([]byte, len(buf))
		copy(ret.unused, buf)
	}
	return ret
}

func (c *ByteCursor) Read(buf []byte) (int, error) {
	nread := 0
	if c.bufpos < c.buflen {
		l := len(buf)
		avail := c.buflen - c.bufpos
		if l >= avail {
			copy(buf, c.buf[c.bufpos:c.buflen])
			nread = avail
			buf = buf[nread:]
			c.bufpos = c.buflen
		} else {
			copy(buf, c.buf[c.bufpos:c.bufpos+l])
			c.bufpos += l
			return l, nil
		}
	}
	n, err := c.in.Read(buf)
	return n + nread, err
}
