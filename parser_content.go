package helium

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/lestrrat-go/helium/sax"
	"github.com/lestrrat-go/pdebug"
)

// parseCDataContent reads the text inside a CDATA section (up to but not
// including the closing ]]>) and returns it. The caller is responsible for
// consuming ]]> and firing the SAX callback afterward, matching libxml2's
// behavior of reporting the position after the closing delimiter.
func (ctx *parserCtx) parseCDataContent() (string, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseCDataContent")
		defer g.IRelease("END parseCDataContent")
	}

	buf := bufferPool.Get()
	defer releaseBuffer(buf)

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
		if b == ']' && cur.PeekAt(off+1) == ']' && cur.PeekAt(off+2) == '>' {
			break
		}
		if b == '\r' {
			buf.WriteByte('\n')
			off++
			if cur.PeekAt(off) == '\n' {
				off++
			}
			continue
		}
		if b < 0x80 {
			buf.WriteByte(b)
			off++
			continue
		}
		r, w, ok := decodeRuneAt(cur, off)
		if !ok {
			break
		}
		buf.WriteRune(r)
		off += w
	}

	if err := cur.Advance(off); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (pctx *parserCtx) parseMisc(ctx context.Context) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseMisc")
		defer g.IRelease("END parseMisc")
	}

	cur := pctx.getCursor()
	for !cur.Done() && pctx.instate != psEOF {
		if cur.HasPrefixString("<?") {
			if err := pctx.parsePI(ctx); err != nil {
				return pctx.error(ctx, err)
			}
		} else if cur.HasPrefixString("<!--") {
			if err := pctx.parseComment(ctx); err != nil {
				return pctx.error(ctx, err)
			}
		} else if isBlankByte(cur.Peek()) {
			pctx.skipBlanks(ctx)
		} else {
			if pdebug.Enabled {
				pdebug.Printf("Nothing more in misc section...")
			}
			break
		}
	}

	return nil
}

var knownPIs = []string{
	"xml-stylesheet",
	"xml-model",
}

func (pctx *parserCtx) parsePI(ctx context.Context) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parsePI")
		defer g.IRelease("END parsePI")
	}

	cur := pctx.getCursor()
	if cur == nil {
		return pctx.error(ctx, errNoCursor)
	}
	if !cur.ConsumeString("<?") {
		return pctx.error(ctx, ErrInvalidProcessingInstruction)
	}
	oldstate := pctx.instate
	pctx.instate = psPI
	defer func() { pctx.instate = oldstate }()

	target, err := pctx.parsePITarget(ctx)
	if err != nil {
		return pctx.error(ctx, err)
	}

	if cur.ConsumeString("?>") {
		if pctx.treeBuilder != nil && !pctx.disableSAX {
			if err := pctx.fastProcessingInstruction(target, ""); err != nil {
				return pctx.error(ctx, err)
			}
		} else if s := pctx.sax; s != nil && !pctx.disableSAX {
			switch err := s.ProcessingInstruction(ctx, target, ""); err {
			case nil, sax.ErrHandlerUnspecified:
			default:
				return pctx.error(ctx, err)
			}
		}
		return nil
	}

	if !isBlankByte(cur.Peek()) {
		return pctx.error(ctx, ErrSpaceRequired)
	}

	pctx.skipBlanks(ctx)
	buf := bufferPool.Get()
	defer releaseBuffer(buf)

	off := 0
	for {
		b := cur.PeekAt(off)
		if b == 0 {
			break
		}
		if b == '?' && cur.PeekAt(off+1) == '>' {
			break
		}
		if b < 0x80 {
			if !isChar(rune(b)) {
				break
			}
			buf.WriteByte(b)
			off++
			continue
		}
		r, w, ok := decodeRuneAt(cur, off)
		if !ok || !isChar(r) {
			break
		}
		buf.WriteRune(r)
		off += w
	}

	if err := cur.Advance(off); err != nil {
		return err
	}
	data := buf.String()

	if !cur.ConsumeString("?>") {
		return pctx.error(ctx, ErrInvalidProcessingInstruction)
	}

	if pctx.treeBuilder != nil && !pctx.disableSAX {
		if err := pctx.fastProcessingInstruction(target, data); err != nil {
			return pctx.error(ctx, err)
		}
	} else if s := pctx.sax; s != nil && !pctx.disableSAX {
		switch err := s.ProcessingInstruction(ctx, target, data); err {
		case nil, sax.ErrHandlerUnspecified:
		default:
			return pctx.error(ctx, err)
		}
	}

	return nil
}

func (pctx *parserCtx) parsePITarget(ctx context.Context) (string, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parsePITarget")
		defer g.IRelease("END parsePITarget")
	}

	name, err := pctx.parseName(ctx)
	if err != nil {
		return "", pctx.error(ctx, err)
	}

	if name == "xml" {
		return "", errors.New("XML declaration allowed only at the start of the document")
	}

	for _, knownpi := range knownPIs {
		if knownpi == name {
			return name, nil
		}
	}

	if strings.IndexByte(name, ':') > -1 {
		return "", errors.New("colons are forbidden from PI names '" + name + "'")
	}

	return name, nil
}

func isChar(r rune) bool {
	if r == utf8.RuneError {
		return false
	}

	c := uint32(r)
	return isXMLCharValue(c)
}

func isXMLCharValue(c uint32) bool {
	if c < 0x100 {
		return (0x9 <= c && c <= 0xa) || c == 0xd || 0x20 <= c
	}
	return (0x100 <= c && c <= 0xd7ff) || (0xe000 <= c && c <= 0xfffd) || (0x10000 <= c && c <= 0x10ffff)
}

var (
	ErrCDATANotFinished = errors.New("invalid CDATA section (premature end)")
	ErrCDATAInvalid     = errors.New("invalid CDATA section")
)

func (pctx *parserCtx) parseCDSect(ctx context.Context) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseCDSect")
		defer g.IRelease("END parseCDSect")
	}

	cur := pctx.getCursor()
	if cur == nil {
		return pctx.error(ctx, errNoCursor)
	}
	if !cur.ConsumeString("<![CDATA[") {
		return pctx.error(ctx, ErrInvalidCDSect)
	}

	pctx.instate = psCDATA
	defer func() { pctx.instate = psContent }()

	str, err := pctx.parseCDataContent()
	if err != nil {
		return pctx.error(ctx, err)
	}

	if !cur.ConsumeString("]]>") {
		return pctx.error(ctx, ErrCDATANotFinished)
	}

	if pctx.treeBuilder != nil && !pctx.disableSAX {
		if pctx.options.IsSet(parseNoCDATA) {
			if err := pctx.fastCharacters([]byte(str)); err != nil {
				return err
			}
		} else {
			if err := pctx.fastCDataBlock([]byte(str)); err != nil {
				return pctx.error(ctx, err)
			}
		}
	} else if s := pctx.sax; s != nil && !pctx.disableSAX {
		if pctx.options.IsSet(parseNoCDATA) {
			if err := pctx.deliverCharacters(ctx, s.Characters, []byte(str)); err != nil {
				return err
			}
		} else {
			switch err := s.CDataBlock(ctx, []byte(str)); err {
			case nil, sax.ErrHandlerUnspecified:
			default:
				return pctx.error(ctx, err)
			}
		}
	}
	return nil
}

func (pctx *parserCtx) parseComment(ctx context.Context) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseComment")
		defer g.IRelease("END parseComment")
	}

	cur := pctx.getCursor()
	if cur == nil {
		return pctx.error(ctx, errNoCursor)
	}
	if !cur.ConsumeString("<!--") {
		return pctx.error(ctx, ErrInvalidComment)
	}

	buf := bufferPool.Get()
	defer releaseBuffer(buf)

	off := 0
	q, qw, qok := decodeRuneAt(cur, off)
	if !qok || !isChar(q) {
		return pctx.error(ctx, ErrInvalidChar)
	}
	buf.WriteRune(q)
	off += qw

	r, rw, rok := decodeRuneAt(cur, off)
	if !rok || !isChar(r) {
		return pctx.error(ctx, ErrInvalidChar)
	}
	buf.WriteRune(r)
	off += rw

	for {
		c, w, ok := decodeRuneAt(cur, off)
		if !ok {
			return pctx.error(ctx, ErrInvalidComment)
		}
		if !isChar(c) {
			return pctx.error(ctx, ErrInvalidChar)
		}
		if q == '-' && r == '-' && c == '>' {
			break
		}
		if q == '-' && r == '-' {
			return pctx.error(ctx, ErrHyphenInComment)
		}
		buf.WriteRune(c)
		q = r
		r = c
		off += w
	}

	buf.Truncate(buf.Len() - 2)
	str := buf.Bytes()
	if err := cur.Advance(off + 1); err != nil {
		return err
	}

	if pctx.treeBuilder != nil && !pctx.disableSAX {
		str = bytes.ReplaceAll(str, []byte{'\r', '\n'}, []byte{'\n'})
		str = bytes.ReplaceAll(str, []byte{'\r'}, []byte{'\n'})
		if err := pctx.fastComment(str); err != nil {
			return pctx.error(ctx, err)
		}
	} else if sh := pctx.sax; sh != nil && !pctx.disableSAX {
		str = bytes.ReplaceAll(str, []byte{'\r', '\n'}, []byte{'\n'})
		str = bytes.ReplaceAll(str, []byte{'\r'}, []byte{'\n'})
		switch err := sh.Comment(ctx, str); err {
		case nil, sax.ErrHandlerUnspecified:
		default:
			return pctx.error(ctx, err)
		}
	}

	return nil
}
