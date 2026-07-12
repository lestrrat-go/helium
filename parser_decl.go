package helium

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/strcursor"
)

func (pctx *parserCtx) parseVersionInfoFromCursor(ctx context.Context) (string, error) {
	cur := pctx.getCursor()
	pctx.skipBlanks(ctx)
	if !cur.ConsumeString("version") {
		return "", pctx.error(ctx, AttrNotFoundError{Token: "version"})
	}
	pctx.skipBlanks(ctx)
	if cur.Peek() != '=' {
		return "", ErrEqualSignRequired
	}
	if err := cur.Advance(1); err != nil {
		return "", err
	}
	pctx.skipBlanks(ctx)

	q := cur.Peek()
	if q != '"' && q != '\'' {
		return "", pctx.error(ctx, errors.New("string not started"))
	}
	if err := cur.Advance(1); err != nil {
		return "", err
	}

	var buf strings.Builder
	for {
		c := cur.Peek()
		if c == q {
			if err := cur.Advance(1); err != nil {
				return "", err
			}
			break
		}
		if c == 0 {
			return "", pctx.error(ctx, errors.New("unterminated version value"))
		}
		_ = buf.WriteByte(c)
		if err := cur.Advance(1); err != nil {
			return "", err
		}
	}
	return buf.String(), nil
}

func (pctx *parserCtx) parseEncodingDeclFromCursor(ctx context.Context) (string, error) {
	cur := pctx.getCursor()
	pctx.skipBlanks(ctx)
	if !cur.ConsumeString("encoding") {
		return "", AttrNotFoundError{Token: "encoding"}
	}
	pctx.skipBlanks(ctx)
	if cur.Peek() != '=' {
		return "", ErrEqualSignRequired
	}
	if err := cur.Advance(1); err != nil {
		return "", err
	}
	pctx.skipBlanks(ctx)

	q := cur.Peek()
	if q != '"' && q != '\'' {
		return "", pctx.error(ctx, errors.New("string not started"))
	}
	if err := cur.Advance(1); err != nil {
		return "", err
	}

	// EncName [81]: [A-Za-z] ([A-Za-z0-9._] | '-')*. The first character must be
	// a letter (so an empty EncName is rejected), and every subsequent character
	// must be a letter, digit, '.', '_', or '-'. A present-but-malformed EncName
	// is a fatal error, mirroring the byte-path parseEncodingName.
	first := cur.Peek()
	if !isEncNameStart(first) {
		return "", pctx.error(ctx, ErrInvalidEncodingName)
	}

	var buf strings.Builder
	_ = buf.WriteByte(first)
	if err := cur.Advance(1); err != nil {
		return "", err
	}
	for {
		c := cur.Peek()
		if c == q {
			if err := cur.Advance(1); err != nil {
				return "", err
			}
			break
		}
		if c == 0 {
			return "", pctx.error(ctx, errors.New("unterminated encoding value"))
		}
		if !isEncNameChar(c) {
			return "", pctx.error(ctx, ErrInvalidEncodingName)
		}
		_ = buf.WriteByte(c)
		if err := cur.Advance(1); err != nil {
			return "", err
		}
	}
	name := buf.String()
	// Record the declared EncName unconditionally (see parserCtx.declaredEncoding).
	pctx.declaredEncoding = name
	return name, nil
}

// isEncNameStart reports whether b is a valid first character of an EncName [81]
// (an ASCII letter).
func isEncNameStart(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// isEncNameChar reports whether b is a valid non-initial character of an EncName
// [81] (letter, digit, '.', '_', or '-').
func isEncNameChar(b byte) bool {
	return isEncNameStart(b) || (b >= '0' && b <= '9') || b == '.' || b == '_' || b == '-'
}

func (pctx *parserCtx) parseStandaloneDeclFromCursor(ctx context.Context) (DocumentStandaloneType, error) {
	cur := pctx.getCursor()
	pctx.skipBlanks(ctx)
	if !cur.ConsumeString("standalone") {
		return StandaloneImplicitNo, AttrNotFoundError{Token: "standalone"}
	}
	pctx.skipBlanks(ctx)
	if cur.Peek() != '=' {
		return StandaloneImplicitNo, ErrEqualSignRequired
	}
	if err := cur.Advance(1); err != nil {
		return StandaloneImplicitNo, err
	}
	pctx.skipBlanks(ctx)

	q := cur.Peek()
	if q != '"' && q != '\'' {
		return StandaloneImplicitNo, pctx.error(ctx, errors.New("string not started"))
	}
	if err := cur.Advance(1); err != nil {
		return StandaloneImplicitNo, err
	}

	var buf strings.Builder
	for {
		c := cur.Peek()
		if c == q {
			if err := cur.Advance(1); err != nil {
				return StandaloneImplicitNo, err
			}
			break
		}
		if c == 0 {
			return StandaloneImplicitNo, pctx.error(ctx, errors.New("unterminated standalone value"))
		}
		_ = buf.WriteByte(c)
		if err := cur.Advance(1); err != nil {
			return StandaloneImplicitNo, err
		}
	}

	switch buf.String() {
	case lexicon.ValueYes:
		return StandaloneExplicitYes, nil
	case lexicon.ValueNo:
		return StandaloneExplicitNo, nil
	default:
		return StandaloneImplicitNo, pctx.error(ctx, errors.New("standalone accepts only 'yes' or 'no'"))
	}
}

var versionBytes = []byte{'v', 'e', 'r', 's', 'i', 'o', 'n'}

func (pctx *parserCtx) parseVersionInfo(ctx context.Context) (string, error) {
	return pctx.parseNamedAttributeBytes(ctx, versionBytes, pctx.parseVersionNum)
}

type qtextHandler func(qch byte) (string, error)

func (pctx *parserCtx) parseNamedAttributeBytes(ctx context.Context, name []byte, valueParser qtextHandler) (string, error) {
	cur := pctx.getByteCursor()
	if cur == nil {
		return "", ErrByteCursorRequired
	}

	pctx.skipBlankBytes(ctx, cur)
	if !cur.Consume(name) {
		return "", pctx.error(ctx, AttrNotFoundError{Token: string(name)})
	}

	pctx.skipBlankBytes(ctx, cur)
	if cur.Peek() != '=' {
		return "", ErrEqualSignRequired
	}
	if err := cur.Advance(1); err != nil {
		return "", err
	}

	pctx.skipBlankBytes(ctx, cur)

	return pctx.parseQuotedTextBytes(valueParser)
}

func (ctx *parserCtx) parseVersionNum(_ byte) (string, error) {
	cur := ctx.getByteCursor()
	if cur == nil {
		return "", ErrByteCursorRequired
	}

	if v := cur.Peek(); v > '9' || v < '0' {
		return "", ErrInvalidVersionNum
	}

	if v := cur.PeekAt(1); v != '.' {
		return "", ErrInvalidVersionNum
	}

	if v := cur.PeekAt(2); v > '9' || v < '0' {
		return "", ErrInvalidVersionNum
	}

	for i := 3; ; i++ {
		if v := cur.PeekAt(i); v > '9' || v < '0' {
			b := bufferPool.Get()
			defer releaseBuffer(b)

			for x := range i {
				_ = b.WriteByte(cur.PeekAt(x))
			}
			if err := cur.Advance(i); err != nil {
				return "", err
			}
			return b.String(), nil
		}
	}
}

func (ctx *parserCtx) parseQuotedTextBytes(cb qtextHandler) (value string, err error) {
	cur := ctx.getByteCursor()
	if cur == nil {
		return "", ErrByteCursorRequired
	}
	q := cur.Peek()
	switch q {
	case '"', '\'':
		if err := cur.Advance(1); err != nil {
			return "", err
		}
	default:
		err = errors.New("string not started (got '" + string([]byte{q}) + "')")
		return
	}

	value, err = cb(q)
	if err != nil {
		return
	}

	if cur.Peek() != q {
		err = errors.New("string not closed")
		return
	}
	if err := cur.Advance(1); err != nil {
		return "", err
	}

	return
}

func (ctx *parserCtx) parseQuotedText(cb qtextHandler) (value string, err error) {
	cur := ctx.getCursor()
	if cur == nil {
		return "", errNoCursor
	}
	q := cur.Peek()
	switch q {
	case '"', '\'':
		if err := cur.Advance(1); err != nil {
			return "", err
		}
	default:
		err = errors.New("string not started (got '" + string([]byte{q}) + "')")
		return
	}

	value, err = cb(q)
	if err != nil {
		return
	}

	if cur.Peek() != q {
		err = errors.New("string not closed")
		return
	}
	if err := cur.Advance(1); err != nil {
		return "", err
	}

	return
}

var encodingBytes = []byte{'e', 'n', 'c', 'o', 'd', 'i', 'n', 'g'}

func (pctx *parserCtx) parseEncodingDecl(ctx context.Context) (string, error) {
	return pctx.parseNamedAttributeBytes(ctx, encodingBytes, func(qch byte) (string, error) {
		return pctx.parseEncodingName(ctx, qch)
	})
}

func (pctx *parserCtx) parseEncodingName(ctx context.Context, _ byte) (string, error) {
	cur := pctx.getByteCursor()
	if cur == nil {
		return "", ErrByteCursorRequired
	}
	c := cur.Peek()

	buf := bufferPool.Get()
	defer releaseBuffer(buf)

	if !isEncNameStart(c) {
		return "", pctx.error(ctx, ErrInvalidEncodingName)
	}
	_ = buf.WriteByte(c)

	i := 1
	for c = cur.PeekAt(i); c != 0; c = cur.PeekAt(i) {
		if !isEncNameChar(c) {
			break
		}
		_ = buf.WriteByte(c)
		i++
	}

	if err := cur.Advance(i); err != nil {
		return "", err
	}

	name := buf.String()
	// Record the declared EncName unconditionally (see parserCtx.declaredEncoding):
	// the BOM/encoding conflict check must see it even when IgnoreEncoding or
	// LenientXMLDecl suppress decoder switching / relax the decl parse.
	pctx.declaredEncoding = name
	return name, nil
}

var standaloneBytes = []byte{'s', 't', 'a', 'n', 'd', 'a', 'l', 'o', 'n', 'e'}

func (pctx *parserCtx) parseStandaloneDecl(ctx context.Context) (DocumentStandaloneType, error) {
	v, err := pctx.parseNamedAttributeBytes(ctx, standaloneBytes, pctx.parseStandaloneDeclValue)
	if err != nil {
		return StandaloneInvalidValue, err
	}
	if v == lexicon.ValueYes {
		return StandaloneExplicitYes, nil
	}
	return StandaloneExplicitNo, nil
}

func (ctx *parserCtx) parseStandaloneDeclValue(_ byte) (string, error) {
	cur := ctx.getByteCursor()
	if cur == nil {
		return "", ErrByteCursorRequired
	}
	if cur.ConsumeString(lexicon.ValueYes) {
		return lexicon.ValueYes, nil
	}

	if cur.ConsumeString(lexicon.ValueNo) {
		return lexicon.ValueNo, nil
	}

	return "", errors.New("invalid standalone declaration")
}

func (pctx *parserCtx) parseName(ctx context.Context) (name string, err error) {
	if pctx.instate == psEOF {
		err = pctx.error(ctx, ErrPrematureEOF)
		return
	}

	cur := pctx.getCursor()
	if cur == nil {
		err = pctx.error(ctx, errNoCursor)
		return
	}

	b0 := cur.Peek()
	if b0 == 0 {
		err = pctx.error(ctx, ErrPrematureEOF)
		return
	}
	var firstRune rune
	var firstWidth int
	if b0 < 0x80 {
		firstRune = rune(b0)
		firstWidth = 1
	} else {
		firstRune, firstWidth, _ = decodeRuneAt(cur, 0)
	}
	// A real U+FFFD decodes as RuneError with width 3 and is a valid name char;
	// only width 1 is genuinely-invalid UTF-8.
	if firstRune == utf8.RuneError && firstWidth == 1 {
		err = pctx.error(ctx, errInvalidUTF8Name)
		return
	}
	if firstRune == ' ' || firstRune == '>' || firstRune == '/' || (firstRune != ':' && !isValidNameStartChar(firstRune)) {
		err = pctx.error(ctx, fmt.Errorf("invalid first letter '%c'", firstRune))
		return
	}

	off := firstWidth
	for {
		b := cur.PeekAt(off)
		if b == 0 {
			break
		}
		if b < 0x80 {
			if b == ' ' || b == '>' || b == '/' {
				break
			}
			r := rune(b)
			if r != ':' && !isValidNameChar(r) {
				break
			}
			off++
			continue
		}
		r, w, ok := decodeRuneAt(cur, off)
		if !ok || (r == utf8.RuneError && w == 1) {
			err = pctx.error(ctx, errInvalidUTF8Name)
			return
		}
		if !isValidNameChar(r) {
			break
		}
		off += w
	}
	if pctx.nameTooLong(off) {
		err = pctx.error(ctx, ErrNameTooLong)
		return
	}

	name = cur.PeekString(off)
	if err := cur.Advance(off); err != nil {
		return "", err
	}
	if name == "" {
		err = pctx.error(ctx, errors.New("internal error: parseName returned with empty name"))
		return
	}
	err = nil
	return
}

func (pctx *parserCtx) parseQName(ctx context.Context) (local string, prefix string, err error) {
	cur := pctx.getCursor()
	if cur == nil {
		err = pctx.error(ctx, errNoCursor)
		return
	}
	if u8, ok := cur.(*strcursor.UTF8Cursor); ok && cur.Peek() < utf8.RuneSelf {
		prefixBytes, localBytes, nBytes, ok := u8.ScanQNameBytes()
		if ok {
			// Bound the full QName (prefix + ':' + local), not just each part,
			// so a prefixed name can't exceed the cap by splitting across the
			// colon. nBytes is the total scanned QName length.
			if pctx.nameTooLong(nBytes) {
				return "", "", pctx.error(ctx, ErrNameTooLong)
			}
			if len(prefixBytes) > 0 {
				prefix = pctx.internNameBytes(prefixBytes)
			}
			local = pctx.internNameBytes(localBytes)
			if err := u8.AdvanceFast(nBytes); err != nil {
				return "", "", err
			}
			return local, prefix, nil
		}
	}
	var v string
	v, err = pctx.parseNCName(ctx)
	if err != nil {
		oerr := err
		if cur.Peek() != ':' {
			v, err = pctx.parseName(ctx)
			if err != nil {
				err = pctx.error(ctx, errors.New("failed to parse QName '"+v+"'"))
				return
			}
			local = v
			err = nil
			return
		}
		err = pctx.error(ctx, oerr)
		return
	}

	if cur.Peek() != ':' {
		local = v
		err = nil
		return
	}

	if err := cur.Advance(1); err != nil {
		return "", "", err
	}
	prefix = v

	v, err = pctx.parseNCName(ctx)
	if err != nil {
		return "", "", pctx.error(ctx, err)
	}
	local = v
	// Bound the full QName (prefix + ':' + local), not just each NCName part,
	// so a prefixed name can't bypass the cap by splitting across the colon.
	if pctx.nameTooLong(len(prefix) + len(local) + 1) {
		return "", "", pctx.error(ctx, ErrNameTooLong)
	}
	return
}

func isNameStartChar(r rune) bool {
	return r != utf8.RuneError && (r == ':' || isValidNameStartChar(r))
}

func isNameChar(r rune) bool {
	return r != utf8.RuneError && (r == ':' || isValidNameChar(r))
}

// isNameCharW is the width-aware form of isNameChar: a RuneError with width 1 is
// invalid UTF-8, but a width-3 RuneError is a real U+FFFD (a valid NameChar).
func isNameCharW(r rune, w int) bool {
	if r == utf8.RuneError && w == 1 {
		return false
	}
	return r == ':' || isValidNameChar(r)
}

func (ctx *parserCtx) parseNmtoken() (string, error) {
	cur := ctx.getCursor()
	if cur == nil {
		return "", errNoCursor
	}

	off := 0
	for {
		b := cur.PeekAt(off)
		if b == 0 {
			break
		}
		if b < 0x80 {
			if !isNameChar(rune(b)) {
				break
			}
			off++
			continue
		}
		r, w, ok := decodeRuneAt(cur, off)
		if !ok || !isNameCharW(r, w) {
			break
		}
		off += w
	}
	if off == 0 {
		return "", fmt.Errorf("expected Nmtoken, got %q", cur.Peek())
	}
	name := cur.PeekString(off)
	if err := cur.Advance(off); err != nil {
		return "", err
	}

	return name, nil
}

func (pctx *parserCtx) parseNCName(ctx context.Context) (ncname string, err error) {
	if pctx.instate == psEOF {
		err = pctx.error(ctx, ErrPrematureEOF)
		return
	}

	cur := pctx.getCursor()
	if cur == nil {
		err = pctx.error(ctx, errNoCursor)
		return
	}

	if u8, ok := cur.(*strcursor.UTF8Cursor); ok && cur.Peek() < utf8.RuneSelf {
		nameBytes, nRunes := u8.ScanNCNameBytes()
		if nRunes == 0 {
			c := cur.Peek()
			err = pctx.error(ctx, fmt.Errorf("invalid name start char %q (U+%04X)", c, c))
			return
		}
		// The limit is in bytes (see [Parser.MaxNameLength]); use the byte
		// length, not the rune count, so a name with multibyte runes cannot
		// exceed the byte cap and still pass.
		if pctx.nameTooLong(len(nameBytes)) {
			err = pctx.error(ctx, ErrNameTooLong)
			return
		}
		ncname = pctx.internNameBytes(nameBytes)
		if err = u8.AdvanceFast(len(nameBytes)); err != nil {
			return "", err
		}
		return
	}

	firstR, firstW, _ := decodeRuneAt(cur, 0)
	if firstR == utf8.RuneError && firstW == 1 {
		err = pctx.error(ctx, errInvalidUTF8Name)
		return
	}
	if firstR == ' ' || firstR == '>' || firstR == '/' || firstR == ':' || !isValidNameStartChar(firstR) {
		err = pctx.error(ctx, fmt.Errorf("invalid name start char %q (U+%04X)", firstR, firstR))
		return
	}

	off := firstW
	for {
		b := cur.PeekAt(off)
		if b == 0 {
			break
		}
		if b < 0x80 {
			r := rune(b)
			if r == ':' || !isValidNameChar(r) {
				break
			}
			off++
			continue
		}
		r, w, ok := decodeRuneAt(cur, off)
		if !ok || (r == utf8.RuneError && w == 1) {
			err = pctx.error(ctx, errInvalidUTF8Name)
			return
		}
		if r == ':' || !isValidNameChar(r) {
			break
		}
		off += w
	}
	if pctx.nameTooLong(off) {
		err = pctx.error(ctx, ErrNameTooLong)
		return
	}
	ncname = pctx.internName(cur.PeekString(off))
	if err := cur.Advance(off); err != nil {
		return "", err
	}
	return
}
