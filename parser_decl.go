package helium

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/strcursor"
	"github.com/lestrrat-go/pdebug"
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
			return "", pctx.error(ctx, errors.New("unterminated encoding value"))
		}
		_ = buf.WriteByte(c)
		if err := cur.Advance(1); err != nil {
			return "", err
		}
	}
	return buf.String(), nil
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

func (e AttrNotFoundError) Error() string {
	return "attribute token '" + e.Token + "' not found"
}

var versionBytes = []byte{'v', 'e', 'r', 's', 'i', 'o', 'n'}

func (pctx *parserCtx) parseVersionInfo(ctx context.Context) (string, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseVersionInfo")
		defer g.IRelease("END parseVersionInfo")
	}

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
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseQuotedTextBytes")
		defer g.IRelease("END parseQuotedTextBytes")
		defer func() { pdebug.Printf("value = '%s'", value) }()
	}

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
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseQuotedText")
		defer g.IRelease("END parseQuotedText")
		defer func() { pdebug.Printf("value = '%s'", value) }()
	}

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
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseEncodingDecl")
		defer g.IRelease("END parseEncodingDecl")
	}
	return pctx.parseNamedAttributeBytes(ctx, encodingBytes, func(qch byte) (string, error) {
		return pctx.parseEncodingName(ctx, qch)
	})
}

func (pctx *parserCtx) parseEncodingName(ctx context.Context, _ byte) (string, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseEncodingName")
		defer g.IRelease("END parseEncodingName")
	}
	cur := pctx.getByteCursor()
	if cur == nil {
		return "", ErrByteCursorRequired
	}
	c := cur.Peek()

	buf := bufferPool.Get()
	defer releaseBuffer(buf)

	if !(c >= 'a' && c <= 'z') && !(c >= 'A' && c <= 'Z') { //nolint:staticcheck
		return "", pctx.error(ctx, ErrInvalidEncodingName)
	}
	_ = buf.WriteByte(c)

	i := 1
	for c = cur.PeekAt(i); c != 0; c = cur.PeekAt(i) {
		if !(c >= 'a' && c <= 'z') && !(c >= 'A' && c <= 'Z') && !(c >= '0' && c <= '9') && c != '.' && c != '_' && c != '-' { //nolint:staticcheck
			break
		}
		_ = buf.WriteByte(c)
		i++
	}

	if err := cur.Advance(i); err != nil {
		return "", err
	}

	return buf.String(), nil
}

var standaloneBytes = []byte{'s', 't', 'a', 'n', 'd', 'a', 'l', 'o', 'n', 'e'}

func (pctx *parserCtx) parseStandaloneDecl(ctx context.Context) (DocumentStandaloneType, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseStandaloneDecl")
		defer g.IRelease("END parseStandaloneDecl")
	}

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
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseName")
		defer g.IRelease("END parseName")
		defer func() { pdebug.Printf("name = '%s'", name) }()
	}
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
	if firstRune == utf8.RuneError {
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
		if !ok || r == utf8.RuneError {
			err = pctx.error(ctx, errInvalidUTF8Name)
			return
		}
		if !isValidNameChar(r) {
			break
		}
		off += w
	}
	if off > MaxNameLength && !pctx.options.IsSet(parseHuge) {
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
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseQName")
		defer g.IRelease("END parseQName")
		defer func() { pdebug.Printf("local='%s' prefix='%s'", local, prefix) }()
	}

	cur := pctx.getCursor()
	if cur == nil {
		err = pctx.error(ctx, errNoCursor)
		return
	}
	if u8, ok := cur.(*strcursor.UTF8Cursor); ok && cur.Peek() < utf8.RuneSelf {
		prefixBytes, localBytes, nBytes, ok := u8.ScanQNameBytes()
		if ok {
			if !pctx.options.IsSet(parseHuge) {
				if len(prefixBytes) > MaxNameLength || len(localBytes) > MaxNameLength {
					return "", "", pctx.error(ctx, ErrNameTooLong)
				}
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
	return
}

func isNameStartChar(r rune) bool {
	return r != utf8.RuneError && (r == ':' || isValidNameStartChar(r))
}

func isNameChar(r rune) bool {
	return r != utf8.RuneError && (r == ':' || isValidNameChar(r))
}

func (ctx *parserCtx) parseNmtoken() (string, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseNmtoken")
		defer g.IRelease("END parseNmtoken")
	}

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
		if !ok || !isNameChar(r) {
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
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseNCName")
		defer g.IRelease("END parseNCName")
		defer func() {
			pdebug.Printf("ncname = '%s'", ncname)
		}()
	}
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
		if nRunes > MaxNameLength && !pctx.options.IsSet(parseHuge) {
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
	if firstR == utf8.RuneError {
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
		if !ok || r == utf8.RuneError {
			err = pctx.error(ctx, errInvalidUTF8Name)
			return
		}
		if r == ':' || !isValidNameChar(r) {
			break
		}
		off += w
	}
	if off > MaxNameLength && !pctx.options.IsSet(parseHuge) {
		err = pctx.error(ctx, ErrNameTooLong)
		return
	}
	ncname = pctx.internName(cur.PeekString(off))
	if err := cur.Advance(off); err != nil {
		return "", err
	}
	return
}
