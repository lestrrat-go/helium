package helium

import (
	"bytes"
	"context"
	"encoding/binary"
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
		// section fails before its closing ]]> is reached, and before
		// the whole run is buffered. Checking here also bounds cur.PeekAt(off)
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
			if !ctx.isLiteralChar(rune(b)) {
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
		if !ctx.isLiteralCharWidth(r, w) {
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
		// observed between reads, ahead of any blocking refill.
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
			if !pctx.isLiteralChar(rune(b)) {
				break
			}
			buf.WriteByte(b)
			off++
			continue
		}
		r, w, ok := decodeRuneAt(cur, off)
		if !ok || !pctx.isLiteralCharWidth(r, w) {
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

// isLiteralChar reports whether r is valid in literal document content for the
// parsed XML version. XML 1.1 RestrictedChar values remain valid character
// reference targets but must not appear literally.
func (pctx *parserCtx) isLiteralChar(r rune) bool {
	if r == utf8.RuneError {
		return false
	}
	return pctx.isLiteralCharValue(uint32(r))
}

// isLiteralCharWidth is the width-aware counterpart of isLiteralChar. A real
// U+FFFD is valid, while a width-one RuneError reports invalid UTF-8.
func (pctx *parserCtx) isLiteralCharWidth(r rune, w int) bool {
	if r == utf8.RuneError && w == 1 {
		return false
	}
	return pctx.isLiteralCharValue(uint32(r))
}

func (pctx *parserCtx) isLiteralCharValue(c uint32) bool {
	return literalCharValueValid(c, pctx.isXML11())
}

// literalCharValueValid reports whether c may appear literally in an XML 1.1
// document (xml11) or an XML 1.0 document (!xml11).
func literalCharValueValid(c uint32, xml11 bool) bool {
	if xml11 {
		return isXML11CharValue(c) && !isXML11RestrictedChar(rune(c))
	}
	return isXMLCharValue(c)
}

// literalASCIIValid holds literalCharValueValid for every ASCII byte, indexed
// [0] for XML 1.0 and [1] for XML 1.1.
var literalASCIIValid = buildLiteralASCIIValid()

func buildLiteralASCIIValid() [2][utf8.RuneSelf]bool {
	var tbl [2][utf8.RuneSelf]bool
	for c := range uint32(utf8.RuneSelf) {
		tbl[0][c] = literalCharValueValid(c, false)
		tbl[1][c] = literalCharValueValid(c, true)
	}
	return tbl
}

const (
	wordOnes  = 0x0101010101010101
	wordHighs = 0x8080808080808080
)

// literalWordValid reports whether all eight bytes packed in w are ASCII bytes
// in 0x20-0x7F (0x20-0x7E for XML 1.1), all of which are valid literal
// characters. A false result only means the bytes need a byte-by-byte check:
// tab, LF, and CR are valid yet fail here.
func literalWordValid(w uint64, xml11 bool) bool {
	if w&wordHighs != 0 {
		return false
	}
	// With every high bit clear, (w - 0x20 per byte) &^ w sets a high bit
	// only when some byte is below 0x20.
	if (w-0x20*wordOnes)&^w&wordHighs != 0 {
		return false
	}
	if !xml11 {
		return true
	}
	// XML 1.1 also rejects a literal DEL (0x7F): find a zero byte in w^0x7F.
	d := w ^ 0x7F*wordOnes
	return (d-wordOnes)&^d&wordHighs == 0
}

// literalBytesValid reports whether b holds only characters that may appear
// literally under the parsed XML version, rejecting invalid UTF-8. It checks
// every byte itself and does not rely on how the caller scanned b. Runs of
// printable ASCII are checked eight bytes at a time, other ASCII bytes by
// table, and only non-ASCII bytes are decoded as runes.
func (pctx *parserCtx) literalBytesValid(b []byte) bool {
	xml11 := pctx.isXML11()
	ascii := &literalASCIIValid[0]
	if xml11 {
		ascii = &literalASCIIValid[1]
	}
	i := 0
	for i < len(b) {
		if len(b)-i >= 8 && literalWordValid(binary.LittleEndian.Uint64(b[i:]), xml11) {
			i += 8
			continue
		}
		if c := b[i]; c < utf8.RuneSelf {
			if !ascii[c] {
				return false
			}
			i++
			continue
		}
		r, w := utf8.DecodeRune(b[i:])
		if r == utf8.RuneError && w == 1 {
			return false
		}
		if !literalCharValueValid(uint32(r), xml11) {
			return false
		}
		i += w
	}
	return true
}

// literalStringValid is the string counterpart of literalBytesValid.
func (pctx *parserCtx) literalStringValid(s string) bool {
	xml11 := pctx.isXML11()
	ascii := &literalASCIIValid[0]
	if xml11 {
		ascii = &literalASCIIValid[1]
	}
	i := 0
	for i < len(s) {
		if len(s)-i >= 8 && literalWordValid(stringWord(s[i:]), xml11) {
			i += 8
			continue
		}
		if c := s[i]; c < utf8.RuneSelf {
			if !ascii[c] {
				return false
			}
			i++
			continue
		}
		r, w := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && w == 1 {
			return false
		}
		if !literalCharValueValid(uint32(r), xml11) {
			return false
		}
		i += w
	}
	return true
}

// stringWord packs the first eight bytes of s little-endian, as
// binary.LittleEndian.Uint64 does for a byte slice. s must hold at least eight
// bytes.
func stringWord(s string) uint64 {
	_ = s[7]
	return uint64(s[0]) | uint64(s[1])<<8 | uint64(s[2])<<16 | uint64(s[3])<<24 |
		uint64(s[4])<<32 | uint64(s[5])<<40 | uint64(s[6])<<48 | uint64(s[7])<<56
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
// and never literally, so this predicate governs only the char-reference
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
	if !qok || !pctx.isLiteralCharWidth(q, qw) {
		return pctx.error(ctx, ErrInvalidChar)
	}
	buf.WriteRune(q)
	off += qw

	r, rw, rok := decodeRuneAt(cur, off)
	if !rok || !pctx.isLiteralCharWidth(r, rw) {
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
		if !pctx.isLiteralCharWidth(c, w) {
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
