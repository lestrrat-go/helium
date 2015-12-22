package helium

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/lestrrat/go-strcursor"
	"github.com/lestrrat/helium/encoding"
	"github.com/lestrrat/helium/internal/debug"
	"github.com/lestrrat/helium/sax"
)

func (ctx *parserCtx) pushNode(e *ParsedElement) {
	if debug.Enabled {
		debug.Printf(" --> push node " + e.local)
	}
	e.next = ctx.element
	ctx.element = e
}

func (ctx *parserCtx) peekNode() *ParsedElement {
	return ctx.element
}

func (ctx *parserCtx) popNode() *ParsedElement {
	e := ctx.peekNode()
	if e == nil {
		if debug.Enabled {
			debug.Printf(" <-- pop node (EMPTY)")
		}
	}

	if debug.Enabled {
		debug.Printf(" <-- pop node " + e.local)
	}
	ctx.element = e.next
	return e
}

func (ctx *parserCtx) release() error {
	ctx.sax = nil
	ctx.userData = nil
	return nil
}

func (ctx *parserCtx) init(p *Parser, b []byte) error {
	ctx.encoding = encNone
	ctx.nbread = 0
	ctx.cursor = strcursor.New(b)
	ctx.instate = psStart
	ctx.sax = p.sax
	ctx.userData = ctx // circular dep?!
	ctx.standalone = StandaloneImplicitNo
	return nil
}

func (e ErrParseError) Error() string {
	return fmt.Sprintf(
		"%s at line %d, column %d\n -> '%s' <-- around here",
		e.Err,
		e.LineNumber,
		e.Column,
		e.Line,
	)
}

func (ctx *parserCtx) error(err error) error {
	// If it's wrapped, just return as is
	if _, ok := err.(ErrParseError); ok {
		return err
	}

	return ErrParseError{
		Column:     ctx.cursor.Column(),
		Err:        err,
		Line:       ctx.cursor.CurrentLine(),
		LineNumber: ctx.cursor.LineNumber(),
		Location:   ctx.cursor.OffsetBytes(),
	}
}

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

func (ctx *parserCtx) detectEncoding() (string, error) {
	if debug.Enabled {
		debug.Printf("START detectEncoding")
		defer debug.Printf("END   detecteEncoding")
	}

	if ctx.curLen() >= 4 {
		if debug.Enabled {
			debug.Printf("got 4 bytes, try 4 byte patterns")
		}
		b := ctx.curPeekBytes(4)
		if bytes.Equal(b, patUCS4BE) {
			ctx.curAdvance(4) // BOM, consume
			return encUCS4BE, nil
		}

		if bytes.Equal(b, patUCS4LE) {
			ctx.curAdvance(4) // BOM, consume
			return encUCS4LE, nil
		}

		if bytes.Equal(b, patUCS42143) {
			ctx.curAdvance(4) // BOM, consume
			return encUCS42143, nil
		}

		if bytes.Equal(b, patUCS43412) {
			ctx.curAdvance(4) // BOM, consume
			return encUCS43412, nil
		}

		if bytes.Equal(b, patEBCDIC) {
			// no BOM
			return encEBCDIC, nil
		}

		if bytes.Equal(b, patMaybeXMLDecl) {
			// no BOM, "<?xm"
			return encUTF8, nil
		}

		/*
		 * Although not part of the recommendation, we also
		 * attempt an "auto-recognition" of UTF-16LE and
		 * UTF-16BE encodings.
		 */
		if bytes.Equal(b, patUTF16LE4B) {
			ctx.curAdvance(4)
			return encUTF16LE, nil
		}

		if bytes.Equal(b, patUTF16BE4B) {
			ctx.curAdvance(4)
			return encUTF16BE, nil
		}
	}

	if ctx.curLen() >= 3 {
		if debug.Enabled {
			debug.Printf("got 3 bytes, try 3 byte patterns")
		}
		b := ctx.curPeekBytes(3)
		if bytes.Equal(b, patUTF8) {
			ctx.curAdvance(3)
			return encUTF8, nil
		}
	}

	if ctx.curLen() >= 2 {
		if debug.Enabled {
			debug.Printf("got 2 bytes, try 2 byte patterns")
		}
		b := ctx.curPeekBytes(2)
		if bytes.Equal(b, patUTF16BE2B) {
			ctx.curAdvance(2)
			return encUTF16BE, nil
		}
		if bytes.Equal(b, patUTF16LE2B) {
			ctx.curAdvance(2)
			return encUTF16LE, nil
		}
	}
	return encNone, errors.New("failed to detect encoding")
}

func (ctx *parserCtx) curHasChars(n int) bool {
	return ctx.cursor.HasChars(n)
}

func (ctx *parserCtx) curDone() bool {
	return ctx.cursor.Done()
}

func (ctx *parserCtx) curAdvance(n int) {
	defer ctx.markEOF()
	ctx.cursor.Advance(n)
}

func (ctx *parserCtx) curPeekBytes(n int) []byte {
	return ctx.cursor.PeekBytes(n)
}

func (ctx *parserCtx) curPeek(n int) rune {
	return ctx.cursor.Peek(n)
}

func (ctx *parserCtx) markEOF() {
	if ctx.cursor.Done() {
		debug.Printf("Marking EOF")
		ctx.instate = psEOF
	}
}

func (ctx *parserCtx) curConsume(n int) string {
	defer ctx.markEOF()
	return ctx.cursor.Consume(n)
}

func (ctx *parserCtx) curConsumePrefix(s string) bool {
	defer ctx.markEOF()
	return ctx.cursor.ConsumePrefix(s)
}

func (ctx *parserCtx) curConsumeBytes(n int) []byte {
	defer ctx.markEOF()
	return ctx.cursor.ConsumeBytes(n)
}

func (ctx *parserCtx) curHasPrefix(s string) bool {
	return ctx.cursor.HasPrefix(s)
}

func (ctx *parserCtx) curCharLen(n int) int {
	return ctx.cursor.CharLen(n)
}

func (ctx *parserCtx) curLen() int {
	return ctx.cursor.Len()
}

func isBlankCh(c rune) bool {
	return c == 0x20 || (0x9 <= c && c <= 0xa) || c == 0xd
}

func (ctx *parserCtx) switchEncoding() error {
	if ctx.encoding == "" {
		ctx.encoding = "utf-8"
	}

	enc := encoding.Load(ctx.encoding)
	if enc == nil {
		return errors.New("encoding '" + ctx.encoding + "' not supported")
	}

	// We're going to have to replace the cursor
	b, err := enc.NewDecoder().Bytes(ctx.cursor.Bytes())
	if err != nil {
		return ctx.error(err)
	}

	ctx.cursor = strcursor.New(b)

	return nil
}

func (ctx *parserCtx) parseDocument() error {
	if s := ctx.sax; s != nil {
		if err := s.SetDocumentLocator(ctx.userData, nil); err != nil {
			return ctx.error(err)
		}
	}

	// see if we can find the preliminary encoding
	if ctx.encoding == "" && ctx.curHasChars(4) {
		if enc, err := ctx.detectEncoding(); err == nil {
			ctx.encoding = enc
		}
	}

	// nothing left? eek
	if !ctx.curHasChars(1) {
		return ctx.error(errors.New("empty document"))
	}

	// XML prolog
	if ctx.curHasPrefix("<?xml") {
		if err := ctx.parseXMLDecl(); err != nil {
			return ctx.error(err)
		}
	}

	// At this point we know the encoding, so switch the encoding
	// of the source.
	if err := ctx.switchEncoding(); err != nil {
		return ctx.error(err)
	}

	if s := ctx.sax; s != nil {
		if err := s.StartDocument(ctx.userData); err != nil {
			return ctx.error(err)
		}
	}

	// Misc part of the prolog
	if err := ctx.parseMisc(); err != nil {
		return ctx.error(err)
	}

	// Doctype declarations and more misc
	if ctx.curHasPrefix("<!DOCTYPE") {
		ctx.inSubset = 1
		if err := ctx.parseDocTypeDecl(); err != nil {
			return ctx.error(err)
		}

		if ctx.curHasPrefix("[") {
			ctx.instate = psDTD
			if err := ctx.parseInternalSubset(); err != nil {
				return ctx.error(err)
			}
		}

		ctx.inSubset = 2
		if s := ctx.sax; s != nil {
			if err := s.ExternalSubset(ctx.userData,  ctx.intSubName, ctx.extSubSystem, ctx.extSubURI); err != nil {
				return ctx.error(err)
			}
		}
		if ctx.instate == psEOF {
			return ctx.error(errors.New("unexpected EOF"))
		}
		ctx.inSubset = 0

		// xmlCleanSpecialAttr(ctxt)

		ctx.instate = psPrologue
		if err := ctx.parseMisc(); err != nil {
			return ctx.error(err)
		}
	}
	ctx.skipBlanks()

	if ctx.curPeek(1) != '<' {
		return ctx.error(ErrEmptyDocument)
	} else {
		ctx.instate = psContent
		if err := ctx.parseElement(); err != nil {
			return ctx.error(err)
		}
		ctx.instate = psEpilogue

		if err := ctx.parseMisc(); err != nil {
			return ctx.error(err)
		}
		if !ctx.curDone() {
			return ctx.error(ErrDocumentEnd)
		}
		ctx.instate = psEOF
	}

	/*
		// Start the actual tree
		if err := ctx.parseContent(); err != nil {
			return ctx.error(err)
		}

		if err := ctx.parseEpilogue(); err != nil {
			return ctx.error(err)
		}
	*/

	// All done
	if s := ctx.sax; s != nil {
		if err := s.EndDocument(ctx.userData); err != nil {
			return ctx.error(err)
		}
	}

	return nil
}

func (ctx *parserCtx) parseContent() error {
	if debug.Enabled {
		debug.Printf("START parseContent")
		defer debug.Printf("END   parseContent")
	}
	ctx.instate = psContent

	for ctx.curLen() > 0 {
		if ctx.curHasPrefix("</") {
			break
		}

		if ctx.curHasPrefix("<?") {
			if err := ctx.parsePI(); err != nil {
				return ctx.error(err)
			}
			continue
		}

		if ctx.curHasPrefix("<![CDATA[") {
			if err := ctx.parseCDSect(); err != nil {
				return ctx.error(err)
			}
			continue
		}

		if ctx.curHasPrefix("<!--") {
			if err := ctx.parseComment(); err != nil {
				return ctx.error(err)
			}
		}

		if ctx.curHasPrefix("<") {
			if err := ctx.parseElement(); err != nil {
				return ctx.error(err)
			}
			continue
		}

		if ctx.curHasPrefix("&") {
			panic("unimplemented (reference)")
		}

		if err := ctx.parseCharData(false); err != nil {
			return err
		}
	}

	return nil
}

// used for the test in the inner loop of the char data testing
var testCharData = [256]byte{
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x09, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, /* 0x9, CR/LF separated */
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x00, 0x27, /* & */
	0x28, 0x29, 0x2A, 0x2B, 0x2C, 0x2D, 0x2E, 0x2F,
	0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37,
	0x38, 0x39, 0x3A, 0x3B, 0x00, 0x3D, 0x3E, 0x3F, /* < */
	0x40, 0x41, 0x42, 0x43, 0x44, 0x45, 0x46, 0x47,
	0x48, 0x49, 0x4A, 0x4B, 0x4C, 0x4D, 0x4E, 0x4F,
	0x50, 0x51, 0x52, 0x53, 0x54, 0x55, 0x56, 0x57,
	0x58, 0x59, 0x5A, 0x5B, 0x5C, 0x00, 0x5E, 0x5F, /* ] */
	0x60, 0x61, 0x62, 0x63, 0x64, 0x65, 0x66, 0x67,
	0x68, 0x69, 0x6A, 0x6B, 0x6C, 0x6D, 0x6E, 0x6F,
	0x70, 0x71, 0x72, 0x73, 0x74, 0x75, 0x76, 0x77,
	0x78, 0x79, 0x7A, 0x7B, 0x7C, 0x7D, 0x7E, 0x7F,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, /* non-ascii */
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
}

/* parse a CharData section.
 * if we are within a CDATA section ']]>' marks an end of section.
 *
 * The right angle bracket (>) may be represented using the string "&gt;",
 * and must, for compatibility, be escaped using "&gt;" or a character
 * reference when it appears in the string "]]>" in content, when that
 * string is not marking the end of a CDATA section.
 *
 * [14] CharData ::= [^<&]* - ([^<&]* ']]>' [^<&]*)
 */
func (ctx *parserCtx) parseCharData(cdata bool) error {
	if debug.Enabled {
		debug.Printf("START parseCharData (byte offset = %d, remainig = '%s')", ctx.cursor.OffsetBytes(), ctx.cursor.Bytes())
		defer debug.Printf("END   parseCharData")
	}

	i := 1
	for ; ctx.curHasChars(i); i++ {
		c := ctx.curPeek(i)
		if !cdata {
			if c == '<' || c == '&' || !isChar(c) {
				break
			}
		}

		if c == ']' && ctx.curPeek(i+1) == ']' && ctx.curPeek(i+2) == '>' {
			if cdata {
				break
			}

			return ctx.error(ErrMisplacedCDATAEnd)
		}
	}

	if i > 1 {
		str := ctx.curConsume(i - 1)
		// XXX This is not right, but it's for now the best place to do this
		str = strings.Replace(str, "\r\n", "\n", -1)
		if ctx.areBlanks(str) {
			if s := ctx.sax; s != nil {
				if err := s.IgnorableWhitespace(ctx.userData, []byte(str)); err != nil {
					return ctx.error(err)
				}
			}
		} else {
			if s := ctx.sax; s != nil {
				if err := s.Characters(ctx.userData, []byte(str)); err != nil {
					return ctx.error(err)
				}
			}
		}
		i--
	} else {
		return errors.New("Invalid char data")
	}
	return nil
}

func (ctx *parserCtx) parseElement() error {
	if debug.Enabled {
		ctx.elemidx++
		i := ctx.elemidx
		debug.Printf("START parseElement (%d)", i)
		defer debug.Printf("END   parseElement (%d)", i)
	}

	// parseStartTag only parses up to the attributes.
	// For example, given <foo>bar</foo>, the next token would
	// be bar</foo>. Given <foo />, the next token would
	// be />
	if err := ctx.parseStartTag(); err != nil {
		return ctx.error(err)
	}

	if !ctx.curHasPrefix("/>") {
		if err := ctx.parseContent(); err != nil {
			return ctx.error(err)
		}
	}

	if err := ctx.parseEndTag(); err != nil {
		return ctx.error(err)
	}

	return nil
}

func (ctx *parserCtx) parseStartTag() error {
	if debug.Enabled {
		debug.Printf("START parseStartTag")
		defer debug.Printf("END   parseStartTag")
	}

	if ctx.curPeek(1) != '<' {
		return ctx.error(ErrStartTagRequired)
	}
	ctx.curAdvance(1)

	name, err := ctx.parseName()
	if err != nil {
		return ctx.error(err)
	}

	attrs := []sax.ParsedAttribute{}
	for ctx.instate != psEOF {
		ctx.skipBlanks()
		if ctx.curPeek(1) == '>' {
			ctx.curAdvance(1)
			break
		}

		if ctx.curPeek(1) == '/' && ctx.curPeek(2) == '>' {
			break
		}
		local, value, prefix, err := ctx.parseAttribute()
		if err != nil {
			return ctx.error(err)
		}

		attr := ParsedAttribute{
			local:  local,
			value:  value,
			prefix: prefix,
		}
		attrs = append(attrs, attr)
	}

	elem := &ParsedElement{
		local:      name,
		attributes: attrs,
	}
	ctx.pushNode(elem)
	if s := ctx.sax; s != nil {
		if err := s.StartElement(ctx.userData, elem); err != nil {
			return ctx.error(err)
		}
	}

	return nil
}

/**
 * parse an end of tag
 *
 * [42] ETag ::= '</' Name S? '>'
 *
 * With namespace
 *
 * [NS 9] ETag ::= '</' QName S? '>'
 */
func (ctx *parserCtx) parseEndTag() error {
	if debug.Enabled {
		debug.Printf("START parseEndTag")
		defer debug.Printf("END   parseEndTag")
	}

	if !ctx.curConsumePrefix("/>") {
		if !ctx.curConsumePrefix("</") {
			return ctx.error(ErrLtSlashRequired)
		}

		name, err := ctx.parseName()
		if err != nil {
			return ctx.error(err)
		}
		debug.Printf("  --> end tag %s", name)
		if ctx.curPeek(1) == '>' {
			ctx.curAdvance(1)
		}

		e := ctx.peekNode()
		if e.local != name {
			return ctx.error(
				errors.New("closing tag does not match ('" + e.local + "' != '" + name + "')"))
		}
	}
	e := ctx.popNode()

	if s := ctx.sax; s != nil {
		if err := s.EndElement(ctx, e); err != nil {
			return ctx.error(err)
		}
	}

	return nil
}

func (ctx *parserCtx) parseAttributeValue(normalize bool) (string, error) {
	if debug.Enabled {
		debug.Printf("START parseAttributeValue")
		defer debug.Printf("END   parseAttributeValue")
	}

	return ctx.parseQuotedText(func(qch rune) (string, error) {
		return ctx.parseAttributeValueInternal(qch, normalize)
	})
}

func (ctx *parserCtx) parseAttributeValueInternal(qch rune, normalize bool) (string, error) {
	if debug.Enabled {
		debug.Printf("START parseAttributeValueInternal")
		defer debug.Printf("END   parseAttributeValueInternal")
	}

	v := []byte(nil)
	for {
		i := 1
		for ; ctx.curHasChars(i); i++ {
			c := ctx.curPeek(i)
			// TODO check for valid "c" value
			if c == qch || c == '&' || c == '<' {
				i--
				break
			}
		}

		v = append(v, ctx.curConsume(i)...)
		i = 1
		if ctx.curPeek(i) == '&' {
			if ctx.curPeek(i+1) == '#' {
				r, err := ctx.parseCharRef()
				if err != nil {
					return "", ctx.error(err)
				}
				l := utf8.RuneLen(r)
				b := make([]byte, l)
				utf8.EncodeRune(b, r)
				v = append(v, b...)
			} else {
				ent, err := ctx.parseEntityRef()
				if err != nil {
					return "", ctx.error(err)
				}

				if ent.entityType == InternalPredefinedEntity {
					if !ctx.replaceEntities {
						v = append(v, "&#38;"...)
					} else {
						v = append(v, ent.content...)
					}
				} else {
					// TODO: decodeentities
					v = append(v, ent.content...)
				}
			}

			i = 1
			continue
		}
		break
	}

	debug.Printf(" ----> (%s)", string(v))

	return string(v), nil
}

func (ctx *parserCtx) parseAttribute() (local string, value string, prefix string, err error) {
	if debug.Enabled {
		debug.Printf("START parseAttribute")
		defer debug.Printf("END   parseAttribute")
	}
	l, p, err := ctx.parseQName()
	if err != nil {
		err = ctx.error(err)
		return
	}

	if debug.Enabled {
		debug.Printf("Attribute name = '%s'", l)
	}
	normalize := false
	/*
	    * get the type if needed
	   if (ctxt->attsSpecial != NULL) {
	       int type;

	       type = (int) (long) xmlHashQLookup2(ctxt->attsSpecial,
	                                           pref, elem, *prefix, name);
	       if (type != 0)
	           normalize = 1;
	   }
	*/
	ctx.skipBlanks()

	if ctx.curPeek(1) != '=' {
		err = ctx.error(ErrEqualSignRequired)
	}
	ctx.curAdvance(1)

	v, err := ctx.parseAttributeValue(normalize)
	if err != nil {
		err = ctx.error(err)
		return
	}

	// If this is one of those the well known tags, check for the validity
	// of the attribute value

	local = l
	prefix = p
	value = v
	err = nil
	return
}

func (ctx *parserCtx) skipBlanks() {
	i := 1
	for ; ctx.curHasChars(i); i++ {
		if !isBlankCh(ctx.curPeek(i)) {
			debug.Printf("%d-th character is NOT blank", i)
			break
		}
	}
	i--
	if i > 0 {
		ctx.curAdvance(i)
	}
}

// should only be here if current buffer is at '<?xml'
func (ctx *parserCtx) parseXMLDecl() error {
	if !ctx.curConsumePrefix("<?xml") {
		return ctx.error(ErrInvalidXMLDecl)
	}

	if !isBlankCh(ctx.curPeek(1)) {
		return errors.New("blank needed after '<?xml'")
	}

	ctx.skipBlanks()

	v, err := ctx.parseVersionInfo()
	if err != nil {
		return ctx.error(err)
	}
	ctx.version = v

	if !isBlankCh(ctx.curPeek(1)) {
		// if the next character isn't blank, we expect the
		// end of XML decl, so return success
		if ctx.curPeek(1) == '?' && ctx.curPeek(2) == '>' {
			ctx.curAdvance(2)
			return nil
		}

		// otherwise, we just saw something unexpected
		return ctx.error(ErrSpaceRequired)
	}

	// we *may* have encoding decl
	v, err = ctx.parseEncodingDecl()
	if err == nil {
		ctx.encoding = v

		// if the encoding decl is found, then we *could* have
		// the end of the XML declaration
		if ctx.curPeek(1) == '?' && ctx.curPeek(2) == '>' {
			ctx.curAdvance(2)
			return nil
		}
	} else if _, ok := err.(ErrAttrNotFound); ok {
		return ctx.error(err)
	}

	vb, err := ctx.parseStandaloneDecl()
	if err != nil {
		return err
	}
	ctx.standalone = vb

	if ctx.curPeek(1) == '?' && ctx.curPeek(2) == '>' {
		ctx.curAdvance(2)
		return nil
	}
	return ctx.error(errors.New("XML declaration not closed"))
}

func (e ErrAttrNotFound) Error() string {
	return "attribute token '" + e.Token + "' not found"
}

func (ctx *parserCtx) parseNamedAttribute(name string, cb qtextHandler) (string, error) {
	ctx.skipBlanks()
	if !ctx.curConsumePrefix(name) {
		return "", ctx.error(ErrAttrNotFound{Token: name})
	}

	ctx.skipBlanks()
	if ctx.curPeek(1) != '=' {
		return "", ErrEqualSignRequired
	}

	ctx.curAdvance(1)
	ctx.skipBlanks()
	return ctx.parseQuotedText(cb)
}

// parse the XML version info (version="1.0")
func (ctx *parserCtx) parseVersionInfo() (string, error) {
	return ctx.parseNamedAttribute("version", ctx.parseVersionNum)
}

/*
 * parse the XML version value.
 *
 * [26] VersionNum ::= '1.' [0-9]+
 *
 * In practice allow [0-9].[0-9]+ at that level
 *
 * Returns the string giving the XML version number
 */
func (ctx *parserCtx) parseVersionNum(_ rune) (string, error) {
	if v := ctx.curPeek(1); v > '9' || v < '0' {
		return "", ErrInvalidVersionNum
	}

	if v := ctx.curPeek(2); v != '.' {
		return "", ErrInvalidVersionNum
	}

	if v := ctx.curPeek(3); v > '9' || v < '0' {
		return "", ErrInvalidVersionNum
	}

	for i := 4; ctx.curHasChars(i); i++ {
		if v := ctx.curPeek(i); v > '9' || v < '0' {
			i--
			return ctx.curConsume(i), nil
		}
	}
	return "", ErrInvalidVersionNum
}

type qtextHandler func(qch rune) (string, error)

func (ctx *parserCtx) parseQuotedText(cb qtextHandler) (string, error) {
	if debug.Enabled {
		debug.Printf("START parseQuotedText")
		defer debug.Printf("END   parseQuotedText")
	}

	q := ctx.curPeek(1)
	switch q {
	case '"', '\'':
		ctx.curAdvance(1)
	default:
		return "", errors.New("string not started (got '" + string([]rune{q}) + "')")
	}
	v, err := cb(q)
	if err != nil {
		return "", err
	}
	if debug.Enabled {
		debug.Printf("--> v = '%s'", v)
	}

	if ctx.curPeek(1) != q {
		return "", errors.New("string not closed")
	}
	ctx.curAdvance(1)

	return v, nil
}

func (ctx *parserCtx) parseEncodingDecl() (string, error) {
	return ctx.parseNamedAttribute("encoding", ctx.parseEncodingName)
}

func (ctx *parserCtx) parseEncodingName(_ rune) (string, error) {
	c := ctx.curPeek(1)

	// first char needs to be alphabets
	if !(c >= 'a' && c <= 'z') && !(c >= 'A' && c <= 'Z') {
		return "", ctx.error(ErrInvalidEncodingName)
	}

	i := 2
	for ; ctx.curHasChars(i); i++ {
		c = ctx.curPeek(i)
		if !(c >= 'a' && c <= 'z') && !(c >= 'A' && c <= 'Z') && !(c >= '0' && c <= '9') && c != '.' && c != '_' && c != '-' {
			i--
			break
		}
	}

	return ctx.curConsume(i), nil
}

func (ctx *parserCtx) parseStandaloneDecl() (DocumentStandaloneType, error) {
	v, err := ctx.parseNamedAttribute("standalone", ctx.parseStandaloneDeclValue)
	if err != nil {
		return StandaloneInvalidValue, err
	}
	if v == "yes" {
		return StandaloneExplicitYes, nil
	} else {
		return StandaloneExplicitNo, nil
	}
}

const (
	yes = "yes"
	no  = "no"
)

func (ctx *parserCtx) parseStandaloneDeclValue(_ rune) (string, error) {
	if ctx.curConsumePrefix(yes) {
		return yes, nil
	}

	if ctx.curConsumePrefix(no) {
		return no, nil
	}

	return "", errors.New("invalid standalone declaration")
}

func (ctx *parserCtx) parseMisc() error {
	for !ctx.curDone() && ctx.instate != psEOF {
		if ctx.curHasPrefix("<?") {
			if err := ctx.parsePI(); err != nil {
				return ctx.error(err)
			}
		} else if ctx.curHasPrefix("<!--") {
			if err := ctx.parseComment(); err != nil {
				return ctx.error(err)
			}
		} else if isBlankCh(ctx.curPeek(1)) {
			debug.Printf("ctx.curPeek(1) == '%c'", ctx.curPeek(1))
			ctx.skipBlanks()
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

func (ctx *parserCtx) parsePI() error {
	if !ctx.curConsumePrefix("<?") {
		return ctx.error(ErrInvalidProcessingInstruction)
	}
	oldstate := ctx.instate
	ctx.instate = psPI
	defer func() { ctx.instate = oldstate }()

	target, err := ctx.parsePITarget()
	if err != nil {
		return ctx.error(err)
	}

	if ctx.curConsumePrefix("?>") {
		if s := ctx.sax; s != nil {
			s.ProcessingInstruction(ctx.userData, target, "")
		}
		return ctx.error(errors.New("processing instruction not closed"))
	}

	if !isBlankCh(ctx.curPeek(1)) {
		return ctx.error(ErrSpaceRequired)
	}

	ctx.skipBlanks()
	i := 1
	for ; ctx.curHasChars(i); i++ {
		if ctx.curPeek(i) == '?' && ctx.curPeek(i+1) == '>' {
			i--
			break
		}

		if !isChar(ctx.curPeek(i)) {
			i--
			break
		}
	}

	data := ctx.curConsume(i)

	if !ctx.curConsumePrefix("?>") {
		return ctx.error(ErrInvalidProcessingInstruction)
	}

	if s := ctx.sax; s != nil {
		s.ProcessingInstruction(ctx.userData, target, data)
	}

	return nil
}

/**
 * parse an XML name.
 *
 * [4] NameChar ::= Letter | Digit | '.' | '-' | '_' | ':' |
 *                  CombiningChar | Extender
 *
 * [5] Name ::= (Letter | '_' | ':') (NameChar)*
 *
 * [6] Names ::= Name (#x20 Name)*
 *
 * Returns the Name parsed.
 */
func (ctx *parserCtx) parseName() (string, error) {
	if debug.Enabled {
		debug.Printf("START parseName")
		defer debug.Printf("END   parseName")
	}
	if ctx.instate == psEOF {
		return "", ctx.error(ErrPrematureEOF)
	}

	i := 1
	for ; ctx.curHasChars(i); i++ {
		c := ctx.curPeek(i)
		debug.Printf("----> %c", c)
		if !(c >= 0x61 && c <= 0x7A) && !(c >= 0x41 && c <= 0x5A) && !(c >= 0x30 && c <= 0x39) && c != '_' && c != '-' && c != ':' && c != '.' {
			i--
			break
		}
	}
	if i > MaxNameLength {
		return "", ctx.error(ErrNameTooLong)
	}

	return ctx.curConsume(i), nil
}

/**
 * parse an XML Namespace QName
 *
 * [6]  QName  ::= (Prefix ':')? LocalPart
 * [7]  Prefix  ::= NCName
 * [8]  LocalPart  ::= NCName
 *
 * Returns the Name parsed
 */
func (ctx *parserCtx) parseQName() (local string, prefix string, err error) {
	if debug.Enabled {
		debug.Printf("START parseQName")
		defer debug.Printf("END   parseQName")
	}
	var v string
	v, err = ctx.parseNCName()
	if err != nil {
		oerr := err
		if ctx.curPeek(1) != ':' {
			v, err = ctx.parseName()
			if err != nil {
				err = ctx.error(errors.New("failed to parse QName '" + v + "'"))
				return
			}
			local = v
			err = nil
			return
		}
		err = ctx.error(oerr)
		return
	}

	if ctx.curPeek(1) != ':' {
		local = v
		err = nil
		return
	}

	ctx.curAdvance(1)
	prefix = v

	v, err = ctx.parseNCName()
	if err != nil {
		v, err = ctx.parseNmtoken()
		if err != nil {
			err = ctx.error(err)
			return
		}

		local = v
		return
	}

	v, err = ctx.parseName()
	if err != nil {
		err = ctx.error(err)
		return
	}
	local = v
	return
}

func isNameChar(r rune) bool {
	return r == '.' || r == '-' || r == '_' || r == ':' ||
		unicode.IsLetter(r) || unicode.IsDigit(r) ||
		unicode.In(r, unicode.Extender)
}

/**
 * parse an XML Nmtoken.
 *
 * [7] Nmtoken ::= (NameChar)+
 *
 * [8] Nmtokens ::= Nmtoken (#x20 Nmtoken)*
 *
 * Returns the Nmtoken parsed
 */
func (ctx *parserCtx) parseNmtoken() (string, error) {
	i := 1
	for ; ctx.curHasChars(i); i++ {
		if !isNameChar(ctx.curPeek(i)) {
			break
		}
	}

	return ctx.curConsume(i), nil
}

/**
 * parse an XML name.
 *
 * [4NS] NCNameChar ::= Letter | Digit | '.' | '-' | '_' |
 *                      CombiningChar | Extender
 *
 * [5NS] NCName ::= (Letter | '_') (NCNameChar)*
 *
 * Returns the Name parsed
 */
func (ctx *parserCtx) parseNCName() (string, error) {
	if debug.Enabled {
		debug.Printf("START parseNCName")
		defer debug.Printf("END   parseNCName")
	}
	if ctx.instate == psEOF {
		return "", ctx.error(ErrPrematureEOF)
	}

	// at this point we have at least 1 character name.
	// see how much more we got here
	i := 1
	for ; ctx.curHasChars(i); i++ {
		c := ctx.curPeek(i)
		if !unicode.IsLetter(c) && !unicode.IsDigit(c) && c != '_' && c != '-' && c != '.' {
			i--
			break
		}
	}
	if i > MaxNameLength {
		return "", ctx.error(ErrNameTooLong)
	}

	ncname := ctx.curConsume(i)
	if debug.Enabled {
		debug.Printf("  -> ncname = '%s'", ncname)
	}
	return ncname, nil
}

func (ctx *parserCtx) parsePITarget() (string, error) {
	name, err := ctx.parseName()
	if err != nil {
		return "", ctx.error(err)
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

// note: unlike libxml2, we can't differentiate between SAX handlers
// that uses the same IgnorableWhitespace and Character handlers
func (ctx *parserCtx) areBlanks(s string) bool {
	if debug.Enabled {
		debug.Printf("START areBlanks (%v)", []byte(s))
		defer debug.Printf("END   areBlanks")
	}

	// Check for xml:space value.
	if ctx.space == 1 || ctx.space == -2 {
		return false
	}

	// Check that the string is made of blanks
	/*
	   if (blank_chars == 0) {
	         for (i = 0;i < len;i++)
	             if (!(IS_BLANK_CH(str[i]))) return(0);
	     }
	*/

	// Look if the element is mixed content in the DTD if available
	if ctx.element == nil {
		debug.Printf("ctx.element == nil")
		return false
	}
	if ctx.doc != nil {
		ok, _ := ctx.doc.IsMixedElement(ctx.element.Name())
		debug.Printf("IsMixedElement -> %b", ok)
		return !ok
	}

	if c := ctx.curPeek(1); c != '<' && c != 0xD {
		return false
	}

	/*
	   if ((ctxt->node->children == NULL) &&
	       (RAW == '<') && (NXT(1) == '/')) return(0);

	   lastChild = xmlGetLastChild(ctxt->node);
	   if (lastChild == NULL) {
	       if ((ctxt->node->type != XML_ELEMENT_NODE) &&
	           (ctxt->node->content != NULL)) return(0);
	   } else if (xmlNodeIsText(lastChild))
	       return(0);
	   else if ((ctxt->node->children != NULL) &&
	            (xmlNodeIsText(ctxt->node->children)))
	       return(0);
	*/
	debug.Printf("all else failed, it's blank!")
	return true
}

func isChar(r rune) bool {
	if r == utf8.RuneError {
		return false
	}

	c := uint32(r)
	if c < 0x100 {
		return (0x9 <= c && c <= 0xa) || c == 0xd || 0x20 <= c
	}
	return (0x100 <= c && c <= 0xd7ff) || (0xe000 <= c && c <= 0xfffd) || (0x10000 <= c && c <= 0x10ffff)
}

var (
	ErrCDATANotFinished = errors.New("invalid CDATA section (premature end)")
	ErrCDATAInvalid     = errors.New("invalid CDATA section")
)

func (ctx *parserCtx) parseCDSect() error {
	if !ctx.curConsumePrefix("<![CDATA[") {
		return ctx.error(ErrInvalidCDSect)
	}
	sh := ctx.sax
	if sh != nil {
		sh.StartCDATA(ctx)
	}

	ctx.instate = psCDATA
	defer func() { ctx.instate = psContent }()

	if err := ctx.parseCharData(true); err != nil {
		return ctx.error(err)
	}

	if !ctx.curConsumePrefix("]]>") {
		return ctx.error(ErrCDATANotFinished)
	}
	if sh != nil {
		sh.EndCDATA(ctx)
	}
	return nil
}

func (ctx *parserCtx) parseComment() error {
	if !ctx.curConsumePrefix("<!--") {
		return ctx.error(ErrInvalidComment)
	}

	i := 1
	q := ctx.curPeek(i)
	if !isChar(q) {
		return ctx.error(ErrInvalidChar)
	}
	i++

	r := ctx.curPeek(i)
	if !isChar(r) {
		return ctx.error(ErrInvalidChar)
	}
	i++

	cur := ctx.curPeek(i)
	for isChar(cur) && (q != '-' || r != '-' || cur != '>') {
		i++
		if q == '-' && r == '-' {
			return ctx.error(ErrHyphenInComment)
		}

		q = r
		r = cur
		cur = ctx.curPeek(i)
	}

	// -3 for -->
	str := ctx.curConsumeBytes(ctx.curCharLen(i - 3))
	// and consume the last 3
	ctx.curConsume(3)
	if sh := ctx.sax; sh != nil {
		str = bytes.Replace(str, []byte{'\r', '\n'}, []byte{'\n'}, -1)
		sh.Comment(ctx, str)
	}

	return nil
}

func (ctx *parserCtx) parseDocTypeDecl() error {
	if !ctx.curConsumePrefix("<!DOCTYPE") {
		return ctx.error(ErrInvalidDTD)
	}

	ctx.skipBlanks()

	name, err := ctx.parseName()
	if err != nil {
		return ctx.error(ErrDocTypeNameRequired)
	}
	ctx.intSubName = name

	ctx.skipBlanks()
	u, eid, err := ctx.parseExternalID()
	if err != nil {
		return ctx.error(err)
	}

	if u != "" || eid != "" {
		ctx.hasExternalSubset = true
	}
	ctx.extSubURI = u
	ctx.extSubSystem = eid

	ctx.skipBlanks()

	if s := ctx.sax; s != nil {
		if err := s.InternalSubset(ctx.userData, name, eid, u); err != nil {
			return ctx.error(err)
		}
	}

	/*
	 * Is there any internal subset declarations ?
	 * they are handled separately in parseInternalSubset()
	 */
	c := ctx.curPeek(1)
	if c == '[' {
		return nil
	}

	// Otherwise this should be the end of DTD
	if c != '>' {
		return ctx.error(ErrDocTypeNotFinished)
	}
	ctx.curAdvance(1)

	return nil
}

func (ctx *parserCtx) parseInternalSubset() error {
	if debug.Enabled {
		debug.Printf("START parseInternalSubset")
		defer debug.Printf("END   parseInternalSubset")
	}

	if ctx.curPeek(1) != '[' {
		goto FinishDTD
	}
	ctx.curAdvance(1)

	ctx.instate = psDTD

	for ctx.curHasChars(1) && ctx.curPeek(1) != ']' {
		ctx.skipBlanks()
		if err := ctx.parseMarkupDecl(); err != nil {
			return ctx.error(err)
		}
/*
		if err := ctx.parsePEReference(); err != nil {
			return ctx.error(err)
		}
*/
	}
	if ctx.curPeek(1) == ']' {
		ctx.curAdvance(1)
		ctx.skipBlanks()
	}

FinishDTD:
	if ctx.curPeek(1) != '>' {
		return ctx.error(ErrDocTypeNotFinished)
	}
	ctx.curAdvance(1)

	return nil
}

/**
 * parse Markup declarations
 *
 * [29] markupdecl ::= elementdecl | AttlistDecl | EntityDecl |
 *                     NotationDecl | PI | Comment
 *
 * [ VC: Proper Declaration/PE Nesting ]
 * Parameter-entity replacement text must be properly nested with
 * markup declarations. That is to say, if either the first character
 * or the last character of a markup declaration (markupdecl above) is
 * contained in the replacement text for a parameter-entity reference,
 * both must be contained in the same replacement text.
 *
 * [ WFC: PEs in Internal Subset ]
 * In the internal DTD subset, parameter-entity references can occur
 * only where markup declarations can occur, not within markup declarations.
 * (This does not apply to references that occur in external parameter
 * entities or to the external subset.)
 */
func (ctx *parserCtx) parseMarkupDecl() error {
	if debug.Enabled {
		debug.Printf("START parseMarkupDecl")
		defer debug.Printf("END   parseMarkupDecl")
	}
	if ctx.curPeek(1) == '<' {
		if ctx.curPeek(2) == '!' {
			switch ctx.curPeek(3) {
			case 'E':
				c := ctx.curPeek(4)
				if c == 'L' { // <!EL...
					if _, err := ctx.parseElementDecl(); err != nil {
						return ctx.error(err)
					}
				} else if c == 'N' { // <!EN....
					if err := ctx.parseEntityDecl(); err != nil {
						return ctx.error(err)
					}
				}
			case 'A': // <!A...
				if err := ctx.parseAttributeListDecl(); err != nil {
					return ctx.error(err)
				}
			case 'N': // <!N...
				if err := ctx.parseNotationDecl(); err != nil {
					return ctx.error(err)
				}
			case '-': // <!-...
				if err := ctx.parseComment(); err != nil {
					return ctx.error(err)
				}
			default:
				// no op: error detected later?
			}
		}
	}

	if ctx.instate == psEOF {
		return nil
	}

	/*
	   // This is only for internal subset. On external entities,
	     // the replacement is done before parsing stage
	     if ((ctxt->external == 0) && (ctxt->inputNr == 1))
	         xmlParsePEReference(ctxt);

	      // Conditional sections are allowed from entities included
	      // by PE References in the internal subset.
	     if ((ctxt->external == 0) && (ctxt->inputNr > 1)) {
	         if ((RAW == '<') && (NXT(1) == '!') && (NXT(2) == '[')) {
	             xmlParseConditionalSections(ctxt);
	         }
	     }
	*/
	ctx.instate = psDTD

	return nil
}

/*
 * parse PEReference declarations
 * The entity content is handled directly by pushing it's content as
 * a new input stream.
 *
 * [69] PEReference ::= '%' Name ';'
 *
 * [ WFC: No Recursion ]
 * A parsed entity must not contain a recursive
 * reference to itself, either directly or indirectly.
 *
 * [ WFC: Entity Declared ]
 * In a document without any DTD, a document with only an internal DTD
 * subset which contains no parameter entity references, or a document
 * with "standalone='yes'", ...  ... The declaration of a parameter
 * entity must precede any reference to it...
 *
 * [ VC: Entity Declared ]
 * In a document with an external subset or external parameter entities
 * with "standalone='no'", ...  ... The declaration of a parameter entity
 * must precede any reference to it...
 *
 * [ WFC: In DTD ]
 * Parameter-entity references may only appear in the DTD.
 * NOTE: misleading but this is handled.
 */
func (ctx *parserCtx) parsePEReference() error {
	if ctx.curPeek(1) != '%' {
		return ctx.error(ErrPercentRequired)
	}
	ctx.curAdvance(1)

	name, err := ctx.parseName()
	if err != nil {
		return ctx.error(err)
	}

	if ctx.curPeek(1) != ';' {
		return ctx.error(ErrSemicolonRequired)
	}
	ctx.curAdvance(1)

	/*
		ctx.nbentities++ // number of entities parsed
		if s := ctx.sax; s != nil {
			entity, err := s.GetParameterEntity(ctx, name)
			if err != nil {
			}
		}

		// XXX Why check here?
		if ctx.instate == psEOF {
			return nil
		}

		return nil
	*/
	_ = name
	return ErrUnimplemented{target: "parsePEReference"}
}

/*
 * parse an Element declaration.
 *
 * [45] elementdecl ::= '<!ELEMENT' S Name S contentspec S? '>'
 *
 * [ VC: Unique Element Type Declaration ]
 * No element type may be declared more than once
 *
 * Returns the type of the element, or -1 in case of error
 */
func (ctx *parserCtx) parseElementDecl() (ElementType, error) {
	if debug.Enabled {
		debug.Printf("START parseElementDecl")
		defer debug.Printf("END   parseElementDecl")
	}

	if !ctx.curConsumePrefix("<!ELEMENT") {
		return UndefinedElementType, ctx.error(ErrInvalidElementDecl)
	}

	if !isBlankCh(ctx.curPeek(1)) {
		return UndefinedElementType, ctx.error(ErrSpaceRequired)
	}
	ctx.skipBlanks()

	name, err := ctx.parseName()
	if err != nil {
		return UndefinedElementType, ctx.error(err)
	}

	/* XXX WHAT?
	   while ((RAW == 0) && (ctxt->inputNr > 1))
	       xmlPopInput(ctxt);
	*/

	if !isBlankCh(ctx.curPeek(1)) {
		return UndefinedElementType, ctx.error(ErrSpaceRequired)
	}
	ctx.skipBlanks()

	var etype ElementType
	var content *ElementContent
	if ctx.curConsumePrefix("EMPTY") {
		etype = EmptyElementType
	} else if ctx.curConsumePrefix("ANY") {
		etype = AnyElementType
	} else if ctx.curPeek(1) == '(' {
		content, etype, err = ctx.parseElementContentDecl()
		if err != nil {
			return UndefinedElementType, ctx.error(err)
		}
	} else {
		/*
		   // [ WFC: PEs in Internal Subset ] error handling.
		      if ((RAW == '%') && (ctxt->external == 0) &&
		          (ctxt->inputNr == 1)) {
		          xmlFatalErrMsg(ctxt, XML_ERR_PEREF_IN_INT_SUBSET,
		    "PEReference: forbidden within markup decl in internal subset\n");
		      } else {
		          xmlFatalErrMsg(ctxt, XML_ERR_ELEMCONTENT_NOT_STARTED,
		                "xmlParseElementDecl: 'EMPTY', 'ANY' or '(' expected\n");
		      }
		      return(-1);
		*/
	}

	ctx.skipBlanks()

	/*
	 * Pop-up of finished entities.
	 */
	/*
	   while ((RAW == 0) && (ctxt->inputNr > 1))
	       xmlPopInput(ctxt);
	   SKIP_BLANKS;
	*/

	if ctx.curPeek(1) != '>' {
		return UndefinedElementType, ctx.error(ErrGtRequired)
	}
	ctx.curAdvance(1)

	/*
	           if (input != ctxt->input) {
	               xmlFatalErrMsg(ctxt, XML_ERR_ENTITY_BOUNDARY,
	   "Element declaration doesn't start and stop in the same entity\n");
	           }
	*/

	if s := ctx.sax; s != nil {
		if err := s.ElementDecl(ctx.userData, name, int(etype), content); err != nil {
			return UndefinedElementType, ctx.error(err)
		}
	}
/*
	           if ((ctxt->sax != NULL) && (!ctxt->disableSAX) &&
	               (ctxt->sax->elementDecl != NULL)) {
	               if (content != NULL)
	                   content->parent = NULL;
	               ctxt->sax->elementDecl(ctxt->userData, name, ret,
	                                      content);
	               if ((content != NULL) && (content->parent == NULL)) {
	                   // this is a trick: if xmlAddElementDecl is called,
	                   // instead of copying the full tree it is plugged directly
	                  // if called from the parser. Avoid duplicating the
	                   // interfaces or change the API/ABI
	                  //
	                   xmlFreeDocElementContent(ctxt->myDoc, content);
	               }
	           } else if (content != NULL) {
	               xmlFreeDocElementContent(ctxt->myDoc, content);
	           }
	*/

	_ = name
	_ = etype
	_ = content
	return etype, nil
}

func (ctx *parserCtx) parseElementContentDecl() (*ElementContent, ElementType, error) {
	if ctx.curPeek(1) != '(' {
		return nil, UndefinedElementType, ctx.error(ErrOpenParenRequired)
	}
	ctx.curAdvance(1)

	if ctx.instate == psEOF {
		return nil, UndefinedElementType, ctx.error(ErrEOF)
	}

	ctx.skipBlanks()

	var ec *ElementContent
	var err error
	var etype ElementType
	if ctx.curHasPrefix("#PCDATA") {
		ec, err = ctx.parseElementMixedContentDecl()
		if err != nil {
			return nil, UndefinedElementType, ctx.error(err)
		}
		etype = MixedElementType
	} else {
		ec, err = ctx.parseElementChildrenContentDeclPriv()
		if err != nil {
			return nil, UndefinedElementType, ctx.error(err)
		}
		etype = ElementElementType
	}

	ctx.skipBlanks()
	return ec, etype, nil
}

func (ctx *parserCtx) parseElementMixedContentDecl() (*ElementContent, error) {
	if !ctx.curConsumePrefix("#PCDATA") {
		return nil, ctx.error(ErrPCDATARequired)
	}

	if ctx.curPeek(1) == ')' {
		/*
		               if ((ctxt->validate) && (ctxt->input->id != inputchk)) {
		                   xmlValidityError(ctxt, XML_ERR_ENTITY_BOUNDARY,
		   "Element content declaration doesn't start and stop in the same entity\n",
		                                    NULL, NULL);
		               }
		*/
		ctx.curAdvance(1)
		ret, err := ctx.doc.CreateElementContent("", ElementContentPCDATA)
		if err != nil {
			return nil, ctx.error(err)
		}

		if ctx.curPeek(1) == '*' {
			ret.coccur = ElementContentMult
			ctx.curAdvance(1)
		}

		return ret, nil
	}

	var err error
	var ret *ElementContent
	var cur *ElementContent
	if c := ctx.curPeek(1); c == '(' || c == '|' {
		ret, err = ctx.doc.CreateElementContent("", ElementContentPCDATA)
		if err != nil {
			return nil, ctx.error(err)
		}
		cur = ret
	}

	var elem string
	for ctx.curPeek(1) == '|' {
		ctx.curAdvance(1)
		if elem == "" {
			ret, err = ctx.doc.CreateElementContent("", ElementContentOr)
			if err != nil {
				return nil, ctx.error(err)
			}

			ret.c1 = cur
			if cur != nil {
				cur.parent = ret
			}
			cur = ret
		} else {
			n, err := ctx.doc.CreateElementContent("", ElementContentOr)
			if err != nil {
				return nil, ctx.error(err)
			}
			n.c1, err = ctx.doc.CreateElementContent("", ElementContentElement)
			if err != nil {
				return nil, ctx.error(err)
			}
			n.c1.parent = n
			cur.c2 = n
			n.parent = cur
			cur = n
		}
		ctx.skipBlanks()
		elem, err = ctx.parseName()
		if err != nil {
			return nil, ctx.error(err)
		}
		ctx.skipBlanks()
	}
	if ctx.curPeek(1) == ')' && ctx.curPeek(2) == '*' {
		ctx.curAdvance(2)
		if elem != "" {
			cur.c2, err = ctx.doc.CreateElementContent(elem, ElementContentElement)
			if err != nil {
				return nil, ctx.error(err)
			}
			cur.c2.parent = cur
		}

		if ret != nil {
			ret.coccur = ElementContentMult
		}
		/*
		               if ((ctxt->validate) && (ctxt->input->id != inputchk)) {
		                   xmlValidityError(ctxt, XML_ERR_ENTITY_BOUNDARY,
		   "Element content declaration doesn't start and stop in the same entity\n",
		                                    NULL, NULL);
		   					}
		*/
	}
	return ret, nil
}

/* *
 * parse the declaration for a Mixed Element content
 * The leading '(' and spaces have been skipped in xmlParseElementContentDecl
 *
 *
 * [47] children ::= (choice | seq) ('?' | '*' | '+')?
 *
 * [48] cp ::= (Name | choice | seq) ('?' | '*' | '+')?
 *
 * [49] choice ::= '(' S? cp ( S? '|' S? cp )* S? ')'
 *
 * [50] seq ::= '(' S? cp ( S? ',' S? cp )* S? ')'
 *
 * [ VC: Proper Group/PE Nesting ] applies to [49] and [50]
 * TODO Parameter-entity replacement text must be properly nested
 *      with parenthesized groups. That is to say, if either of the
 *      opening or closing parentheses in a choice, seq, or Mixed
 *      construct is contained in the replacement text for a parameter
 *      entity, both must be contained in the same replacement text. For
 *      interoperability, if a parameter-entity reference appears in a
 *      choice, seq, or Mixed construct, its replacement text should not
 *      be empty, and neither the first nor last non-blank character of
 *      the replacement text should be a connector (| or ,).
 *
 * Returns the tree of xmlElementContentPtr describing the element
 *          hierarchy.
 */
func (ctx *parserCtx) parseElementChildrenContentDeclPriv() (*ElementContent, error) {
	return nil, nil
}

func (ctx *parserCtx) parseEntityDecl() error {
	if debug.Enabled {
		debug.Printf("START parseEntityDecl")
		defer debug.Printf("END   parseEntityDecl")
	}
	return nil
}

func (ctx *parserCtx) parseAttributeListDecl() error {
	return nil
}

func (ctx *parserCtx) parseNotationDecl() error {
	return nil
}

func (ctx *parserCtx) parseExternalID() (string, string, error) {
	return "", "", nil
}

func (ctx *parserCtx) parseEpilogue() error {
	return nil
}

func (ctx *parserCtx) parseReference() (string, error) {
	if debug.Enabled {
		debug.Printf("START parseReference")
		defer debug.Printf("END   parseReference")
	}

	if ctx.curPeek(1) != '&' {
		return "", ctx.error(ErrAmpersandRequired)
	}

	// "&#..." CharRef
	if ctx.curPeek(2) == '#' {
		v, err := ctx.parseCharRef()
		if err != nil {
			return "", ctx.error(err)
		}
		l := utf8.RuneLen(v)
		b := make([]byte, l)
		utf8.EncodeRune(b, v)
		if s := ctx.sax; s != nil {
			if err := s.Characters(ctx.userData, b); err != nil {
				return "", ctx.error(err)
			}
		}
		return string(b), nil
	}

	// &...
	ent, err := ctx.parseEntityRef()
	if err != nil {
		return "", ctx.error(err)
	}

	// if !ctx.wellFormed { return } ??

	if ent.EntityType() == InternalPredefinedEntity {
		/*
			if s := ctx.sax; s != nil {
				if err := s.Characters(ctx.userData, []byte(ent.Content())); err != nil {
					return "", ctx.error(err)
				}
			}
		*/
		return ent.Content(), nil
	}

	return "", ErrUnimplemented{target: "parseReference"}
}

/*
 * parse Reference declarations
 *
 * [66] CharRef ::= '&#' [0-9]+ ';' |
 *                  '&#x' [0-9a-fA-F]+ ';'
 *
 * [ WFC: Legal Character ]
 * Characters referred to using character references must match the
 * production for Char.
 *
 * Returns the value parsed as a rune
 */
func (ctx *parserCtx) parseCharRef() (rune, error) {
	var val int32
	if ctx.curConsumePrefix("&#x") {
		for ctx.curHasChars(1) && ctx.curPeek(1) != ';' {
			c := ctx.curPeek(1)
			if c >= '0' && c <= '9' {
				val = val*16 + (c - '0')
			} else if c >= 'a' && c <= 'f' {
				val = val*16 + (c - 'a') + 10
			} else if c >= 'A' && c <= 'F' {
				val = val*16 + (c - 'A') + 10
			} else {
				return utf8.RuneError, errors.New("invalid hex CharRef")
			}
			ctx.curAdvance(1)
		}
		if ctx.curPeek(1) == ';' {
			ctx.curAdvance(1)
		}
	} else if ctx.curConsumePrefix("&#") {
		for ctx.curHasChars(1) && ctx.curPeek(1) != ';' {
			c := ctx.curPeek(1)
			if c >= '0' && c <= '9' {
				val = val*10 + (c - '0')
			} else {
				return utf8.RuneError, errors.New("invalid decimal CharRef")
			}
			ctx.curAdvance(1)
		}
		if ctx.curPeek(1) == ';' {
			ctx.curAdvance(1)
		}
	} else {
		return utf8.RuneError, errors.New("invalid char ref")
	}
	if isChar(val) && val <= unicode.MaxRune {
		return rune(val), nil
	}

	return utf8.RuneError, ErrInvalidChar
}

func (ctx *parserCtx) parseEntityRef() (*Entity, error) {
	if debug.Enabled {
		debug.Printf("START parseEntityRef")
		defer debug.Printf("END   parseEntityRef")
	}

	if ctx.curPeek(1) != '&' {
		return nil, ctx.error(ErrAmpersandRequired)
	}
	ctx.curAdvance(1)

	name, err := ctx.parseName()
	if err != nil {
		return nil, ctx.error(ErrNameRequired)
	}

	debug.Printf(" ----> name = %s", name)
	debug.Printf(" ----> %c", ctx.curPeek(1))
	if ctx.curPeek(1) != ';' {
		return nil, ctx.error(ErrSemicolonRequired)
	}
	ctx.curAdvance(1)

	if ent := resolvePredefinedEntity(name); ent != nil {
		return ent, nil
	}

	return nil, ErrUnimplemented{target: "parseEntityRef"}
}
