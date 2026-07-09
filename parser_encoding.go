package helium

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/lestrrat-go/helium/internal/encoding"
	"github.com/lestrrat-go/helium/internal/strcursor"
)

const (
	encNone     = ""
	encUCS4BE   = "ucs4be"
	encUCS4LE   = "ucs4le"
	encUCS42143 = "ucs4_2143"
	encUCS43412 = "ucs4_3412"
	encEBCDIC   = "ebcdic"
	encUTF8     = "utf8"
	encUTF16LE  = "utf16le"
	encUTF16BE  = "utf16be"

	// ebcdicEncodingSniffMax bounds the EBCDIC prefix buffered on the streaming
	// reader path before parsing. ExtractEBCDICEncoding only scans the first 200
	// bytes for the XML declaration's encoding name; 256 gives a small margin
	// while keeping the pre-parse buffer bounded so the remainder can stream.
	ebcdicEncodingSniffMax = 256

	// maxSniffZeroProgressReads bounds how many CONSECUTIVE (0, nil) reads the
	// EBCDIC sniff loops tolerate before giving up with io.ErrNoProgress. A single
	// zero-progress read is legitimate (a slow producer may return no data and no
	// error while it waits for more input), so a transient empty read must NOT
	// truncate the sniff prefix — otherwise ExtractEBCDICEncoding sees too few
	// bytes, the encoding name is lost, and the parser wrongly defaults to
	// IBM-037, never re-switching to the declared (e.g. CP1141) EBCDIC variant.
	// The bound mirrors the cursor fill loops' maxZeroProgressReads guard so a
	// pathological (0, nil)-forever reader fails fast instead of hanging.
	maxSniffZeroProgressReads = 100
)

var (
	patUCS4BE       = []byte{0x00, 0x00, 0x00, 0x3C}
	patUCS4LE       = []byte{0x3C, 0x00, 0x00, 0x00}
	patUCS42143     = []byte{0x00, 0x00, 0x3C, 0x00}
	patUCS43412     = []byte{0x00, 0x3C, 0x00, 0x00}
	patEBCDIC       = []byte{0x4C, 0x6F, 0xA7, 0x94}
	patUTF16LE4B    = []byte{0x3C, 0x00, 0x3F, 0x00}
	patUTF16BE4B    = []byte{0x00, 0x3C, 0x00, 0x3F}
	patUTF8         = []byte{0xEF, 0xBB, 0xBF}
	patUTF16LE2B    = []byte{0xFF, 0xFE}
	patUTF16BE2B    = []byte{0xFE, 0xFF}
	patMaybeXMLDecl = []byte{0x3C, 0x3F, 0x78, 0x6D}
)

func (ctx *parserCtx) detectEncoding() (encoding string, err error) {
	cur := ctx.getByteCursor()
	if cur == nil {
		return encNone, ErrByteCursorRequired
	}

	// The UCS-4 patterns are the encoded form of the first '<' character
	// (e.g. 0x00 0x00 0x00 0x3C is '<' in UCS-4 BE), NOT a byte-order mark.
	// Detection must PEEK them, never consume, or the leading '<' is lost
	// and the decoded document fails with "start tag expected".
	if cur.HasPrefix(patUCS4BE) {
		encoding = encUCS4BE
		return
	}

	if cur.HasPrefix(patUCS4LE) {
		encoding = encUCS4LE
		return
	}

	if cur.HasPrefix(patUCS42143) {
		encoding = encUCS42143
		return
	}

	if cur.HasPrefix(patUCS43412) {
		encoding = encUCS43412
		return
	}

	if cur.HasPrefix(patEBCDIC) {
		encoding = encEBCDIC
		return
	}

	if cur.HasPrefix(patMaybeXMLDecl) {
		encoding = encUTF8
		return
	}

	if cur.HasPrefix(patUTF16LE4B) {
		encoding = encUTF16LE
		return
	}

	if cur.HasPrefix(patUTF16BE4B) {
		encoding = encUTF16BE
		return
	}

	if cur.Consume(patUTF8) {
		encoding = encUTF8
		ctx.autoEncoding = encUTF8
		return
	}

	if cur.Consume(patUTF16BE2B) {
		encoding = encUTF16BE
		ctx.autoEncoding = encUTF16BE
		return
	}

	if cur.Consume(patUTF16LE2B) {
		encoding = encUTF16LE
		ctx.autoEncoding = encUTF16LE
		return
	}

	encoding = encNone
	err = errors.New("failed to detect encoding")
	return
}

// fixedWidthUnicodeEncoding reports the fixed-width Unicode encoding (UTF-16 /
// UCS-4) that an external resource's replacement text begins with — detected
// from a byte-order mark or the encoded shape of a leading '<'/'<?' — or "" for
// ASCII-compatible content. These encodings are not ASCII-compatible, so their
// bytes (body AND any leading TextDecl) must be decoded to UTF-8 before either
// can be read; a byte-level "<?xml" scan cannot see a TextDecl that is itself
// UTF-16-encoded. The pattern order mirrors detectEncoding. EBCDIC and the
// ASCII-compatible UTF-8 forms are deliberately excluded — those stay on the
// byte-level TextDecl path.
func fixedWidthUnicodeEncoding(content []byte) string {
	switch {
	case bytes.HasPrefix(content, patUCS4BE):
		return encUCS4BE
	case bytes.HasPrefix(content, patUCS4LE):
		return encUCS4LE
	case bytes.HasPrefix(content, patUCS42143):
		return encUCS42143
	case bytes.HasPrefix(content, patUCS43412):
		return encUCS43412
	case bytes.HasPrefix(content, patUTF16LE4B):
		return encUTF16LE
	case bytes.HasPrefix(content, patUTF16BE4B):
		return encUTF16BE
	case bytes.HasPrefix(content, patUTF16BE2B):
		return encUTF16BE
	case bytes.HasPrefix(content, patUTF16LE2B):
		return encUTF16LE
	}
	return ""
}

func isBlankCh(c rune) bool {
	return c == 0x20 || (0x9 <= c && c <= 0xa) || c == 0xd
}

func isBlankByte(c byte) bool {
	return c == 0x20 || (0x9 <= c && c <= 0xa) || c == 0xd
}

func utf8LeadWidth(b byte) int {
	switch {
	case b < 0x80:
		return 1
	case b&0xE0 == 0xC0:
		return 2
	case b&0xF0 == 0xE0:
		return 3
	case b&0xF8 == 0xF0:
		return 4
	default:
		return 1
	}
}

func decodeRuneAt(cur strcursor.Cursor, offset int) (rune, int, bool) {
	b := cur.PeekAt(offset)
	if b == 0 {
		return 0, 0, false
	}
	if b < utf8.RuneSelf {
		return rune(b), 1, true
	}

	width := utf8LeadWidth(b)
	var tmp [utf8.UTFMax]byte
	tmp[0] = b
	for i := 1; i < width; i++ {
		next := cur.PeekAt(offset + i)
		if next == 0 {
			return utf8.RuneError, 1, true
		}
		tmp[i] = next
	}

	r, w := utf8.DecodeRune(tmp[:width])
	if r == utf8.RuneError && w == 0 {
		return utf8.RuneError, 1, true
	}
	return r, w, true
}

func (ctx *parserCtx) switchEncoding() error {
	encName := ctx.encoding
	if encName == "" {
		encName = ctx.detectedEncoding
		if encName == "" {
			encName = "utf8"
		}
	}

	if encoding.IsUTF8(encName) {
		cur := ctx.getByteCursor()
		if cur == nil {
			return ErrByteCursorRequired
		}
		ctx.popInput()
		ctx.pushInput(strcursor.NewUTF8Cursor(cur))
		return nil
	}

	enc := encoding.Load(encName)
	if enc == nil {
		return errors.New("encoding '" + encName + "' not supported")
	}

	cur := ctx.getByteCursor()
	if cur == nil {
		return ErrByteCursorRequired
	}

	b := enc.NewDecoder().Reader(cur)
	ctx.popInput()
	ctx.pushInput(strcursor.NewUTF8Cursor(b))

	return nil
}

// bomAllowedEncodings maps a BOM-asserted Unicode encoding to the set of
// declared-encoding names (case-insensitive) that do NOT contradict it. The
// alias lists mirror libxml2's xmlSetDeclaredEncoding.
var bomAllowedEncodings = map[string]map[string]struct{}{
	encUTF8:    {"utf-8": {}, "utf8": {}},
	encUTF16LE: {"utf-16": {}, "utf-16le": {}, "utf16": {}},
	encUTF16BE: {"utf-16": {}, "utf-16be": {}, "utf16": {}},
}

// checkBOMEncodingConflict reports a fatal error when the document declared an
// encoding that contradicts the Unicode encoding asserted by a leading
// byte-order mark (XML §4.3.3: presenting an entity in an encoding other than
// the one named in its declaration is a fatal error). Only a real consumed BOM
// sets ctx.autoEncoding, so a plain ASCII/UTF-8 `<?xml` start declaring a
// single-byte encoding (e.g. iso-8859-1) is unaffected. libxml2 downgrades this
// to a warning; helium follows the spec and the W3C xml suite (hst-lhs-007/008)
// in treating it as fatal. Declared aliases that match the BOM are accepted.
//
// The check consults ctx.declaredEncoding (the parsed EncName) rather than
// ctx.encoding, so it still fires under IgnoreEncoding(true): that option
// suppresses the decoder switch (erasing ctx.encoding) but must not suppress
// this fatal well-formedness check.
func (ctx *parserCtx) checkBOMEncodingConflict() error {
	if ctx.autoEncoding == "" || ctx.declaredEncoding == "" {
		return nil
	}
	allowed, ok := bomAllowedEncodings[ctx.autoEncoding]
	if !ok {
		return nil
	}
	if _, ok := allowed[strings.ToLower(ctx.declaredEncoding)]; ok {
		return nil
	}
	return fmt.Errorf("%w: declared %q, byte-order mark implies %q",
		ErrEncodingBOMMismatch, ctx.declaredEncoding, ctx.autoEncoding)
}

var xmlDeclHint = []byte{'<', '?', 'x', 'm', 'l'}

func looksLikeXMLDecl(bcur *strcursor.ByteCursor) bool {
	if !bcur.HasPrefix(xmlDeclHint) {
		return false
	}
	sixth := bcur.PeekAt(5)
	return sixth == ' ' || sixth == '\t' || sixth == '\r' || sixth == '\n'
}

func looksLikeXMLDeclString(cur strcursor.Cursor) bool {
	if !cur.HasPrefixString("<?xml") {
		return false
	}
	sixth := cur.PeekAt(5)
	return sixth == ' ' || sixth == '\t' || sixth == '\r' || sixth == '\n'
}
