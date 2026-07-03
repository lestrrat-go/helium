package helium

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/sax"
)

// parseCDataContent reads the text inside a CDATA section (up to but not
// including the closing ]]>) and returns it. The caller is responsible for
// consuming ]]> and firing the SAX callback afterward, matching libxml2's
// behavior of reporting the position after the closing delimiter.
func (ctx *parserCtx) parseCDataContent() (string, error) {
	buf := bufferPool.Get()
	defer releaseBuffer(buf)

	cur := ctx.getCursor()
	if cur == nil {
		return "", errNoCursor
	}

	off := 0
	for {
		// Enforce the node-content cap during accumulation so a giant CDATA
		// section fails before its closing ]]> is reached, rather than after
		// buffering the whole run. Checking here also bounds cur.PeekAt(off)
		// growth (and thus the cursor's internal buffer).
		if ctx.nodeContentTooLong(buf.Len()) {
			return "", ErrNodeContentTooLarge
		}
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
			if !isChar(rune(b)) {
				return "", ErrInvalidChar
			}
			buf.WriteByte(b)
			off++
			continue
		}
		r, w, ok := decodeRuneAt(cur, off)
		if !ok {
			break
		}
		if !isCharWidth(r, w) {
			return "", ErrInvalidChar
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
	cur := pctx.getCursor()
	for {
		// Check the context BEFORE cur.Done(), which may refill the cursor
		// from an io.Reader and block; this lets a cancelled context be
		// observed between reads rather than after a blocking refill.
		if err := ctx.Err(); err != nil {
			return err
		}
		if cur.Done() || pctx.instate == psEOF {
			break
		}
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
			// An over-cap whitespace run (e.g. infinite blanks before the
			// root) is a memory-amplification DoS; surface it instead of
			// looping forever over the still-blank cursor.
			if pctx.blankRunErr != nil {
				return pctx.error(ctx, pctx.blankRunErr)
			}
		} else {
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
		// Enforce the node-content cap during accumulation so a giant PI body
		// fails before its closing ?> is reached.
		if pctx.nodeContentTooLong(buf.Len()) {
			return pctx.error(ctx, ErrNodeContentTooLarge)
		}
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
		if !ok || !isCharWidth(r, w) {
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
	name, err := pctx.parseName(ctx)
	if err != nil {
		return "", pctx.error(ctx, err)
	}

	// The name "xml" is reserved for the XML declaration in any case (XML 1.0
	// §2.6), so reject it case-insensitively — matching xmlchar.IsValidPITarget,
	// which the serializer applies, so parse and reparse stay consistent.
	if strings.EqualFold(name, lexicon.PrefixXML) {
		return "", errors.New("XML declaration allowed only at the start of the document")
	}

	if slices.Contains(knownPIs, name) {
		return name, nil
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

// isCharWidth is the width-aware counterpart of isChar. A utf8.RuneError with
// width 1 denotes genuinely invalid UTF-8 and must be rejected, but a real
// U+FFFD decodes as utf8.RuneError with width 3 and is a valid XML Char. The
// fast UTF-8 scanners make the same distinction (see
// internal/strcursor/utf8cursor.go).
func isCharWidth(r rune, w int) bool {
	if r == utf8.RuneError && w == 1 {
		return false
	}
	return isXMLCharValue(uint32(r))
}

func isXMLCharValue(c uint32) bool {
	if c < 0x100 {
		return (0x9 <= c && c <= 0xa) || c == 0xd || 0x20 <= c
	}
	return (0x100 <= c && c <= 0xd7ff) || (0xe000 <= c && c <= 0xfffd) || (0x10000 <= c && c <= 0x10ffff)
}

// isXML11CharValue implements the XML 1.1 Char production:
//
//	Char ::= [#x1-#xD7FF] | [#xE000-#xFFFD] | [#x10000-#x10FFFF]
//
// The C0/C1 control characters (0x1-0x1F, 0x7F-0x9F) the XML 1.0 Char
// production forbids are valid XML 1.1 characters; only U+0000 is disallowed.
// XML 1.1 requires the restricted characters to appear as character references
// rather than literally, so this predicate governs only the char-reference
// value check.
func isXML11CharValue(c uint32) bool {
	if c == 0 {
		return false
	}
	if c < 0x100 {
		return true
	}
	return (0x100 <= c && c <= 0xd7ff) || (0xe000 <= c && c <= 0xfffd) || (0x10000 <= c && c <= 0x10ffff)
}

var (
	ErrCDATANotFinished = errors.New("invalid CDATA section (premature end)")
	ErrCDATAInvalid     = errors.New("invalid CDATA section")
)

func (pctx *parserCtx) parseCDSect(ctx context.Context) error {
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
	if !qok || !isCharWidth(q, qw) {
		return pctx.error(ctx, ErrInvalidChar)
	}
	buf.WriteRune(q)
	off += qw

	r, rw, rok := decodeRuneAt(cur, off)
	if !rok || !isCharWidth(r, rw) {
		return pctx.error(ctx, ErrInvalidChar)
	}
	buf.WriteRune(r)
	off += rw

	for {
		// Enforce the node-content cap during accumulation so a giant comment
		// body fails before its closing --> is reached.
		if pctx.nodeContentTooLong(buf.Len()) {
			return pctx.error(ctx, ErrNodeContentTooLarge)
		}
		c, w, ok := decodeRuneAt(cur, off)
		if !ok {
			return pctx.error(ctx, ErrInvalidComment)
		}
		if !isCharWidth(c, w) {
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
