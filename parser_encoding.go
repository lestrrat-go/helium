package helium

import (
	"errors"
	"unicode/utf8"

	"github.com/lestrrat-go/helium/internal/encoding"
	"github.com/lestrrat-go/helium/internal/strcursor"
	"github.com/lestrrat-go/pdebug"
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
	if pdebug.Enabled {
		g := pdebug.IPrintf("START detectEncoding")
		defer func() {
			g.IRelease("END detecteEncoding '%s'", encoding)
		}()
	}

	cur := ctx.getByteCursor()
	if cur == nil {
		return encNone, ErrByteCursorRequired
	}

	if cur.Consume(patUCS4BE) {
		encoding = encUCS4BE
		return
	}

	if cur.Consume(patUCS4LE) {
		encoding = encUCS4LE
		return
	}

	if cur.Consume(patUCS42143) {
		encoding = encUCS42143
		return
	}

	if cur.Consume(patUCS43412) {
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
		return
	}

	if cur.Consume(patUTF16BE2B) {
		encoding = encUTF16BE
		return
	}

	if cur.Consume(patUTF16LE2B) {
		encoding = encUTF16LE
		return
	}

	encoding = encNone
	err = errors.New("failed to detect encoding")
	return
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
	if pdebug.Enabled {
		g := pdebug.IPrintf("START switchEncoding()")
		defer g.IRelease("END switchEncoding")
	}

	encName := ctx.encoding
	if encName == "" {
		encName = ctx.detectedEncoding
		if encName == "" {
			encName = "utf8"
		}
	}

	if pdebug.Enabled {
		pdebug.Printf("Loading encoding '%s'", encName)
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
