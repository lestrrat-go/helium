package helium

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/lestrrat/helium/internal/debug"
)

func (p ParsedElement) Local() string {
	return p.local
}

func (p ParsedElement) Attributes() []ParsedAttribute {
	return p.attributes
}

func (ctx *parserCtx) pushNode(e *ParsedElement) {
	e.next = ctx.element
	ctx.element = e
}

func (ctx *parserCtx) peekNode() *ParsedElement {
	return ctx.element
}

func (ctx *parserCtx) popNode() *ParsedElement {
	e := ctx.peekNode()
	if e != nil {
		ctx.element = e.next
	}
	return e
}

func (ctx *parserCtx) release() error {
	ctx.input = nil
	ctx.inputsz = 0
	ctx.sax = nil
	ctx.userData = nil
	return nil
}

func (ctx *parserCtx) init(b []byte) error {
	ctx.encoding = encNone
	// for now assume input is UTF-8
	ctx.input = b
	ctx.inputsz = len(b)
	ctx.instate = psStart
	ctx.lineno = 1
	ctx.remain = ctx.inputsz
	ctx.sax = &TreeBuilder{}
	ctx.userData = ctx // circular dep?!
	return nil
}

func (e ErrParseError) Error() string {
	return fmt.Sprintf(
		"%s at line %d, column %d",
		e.Err,
		e.Line,
		e.Column,
	)
}

func (ctx *parserCtx) error(err error) error {
	// If it's wrapped, just return as is
	if _, ok := err.(ErrParseError); ok {
		return err
	}

	sofar := ctx.input[:ctx.idx]
	return ErrParseError{
		Err:      err,
		Location: ctx.idx,
		Line:     bytes.Count(sofar, []byte{'\n'}) + 1,
		Column:   ctx.idx - bytes.LastIndex(sofar, []byte{'\n'}),
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
	patMaybeXMLDecl = []byte{0x3C, 0x3F, 0x78, 0x6D}
)

func (ctx *parserCtx) detectEncoding() (string, error) {
	if ctx.remaining() > 3 {
		b := ctx.peek(4)
		if bytes.Equal(b, patUCS4BE) {
			ctx.adv(4) // BOM, consume
			return encUCS4BE, nil
		}

		if bytes.Equal(b, patUCS4LE) {
			ctx.adv(4) // BOM, consume
			return encUCS4LE, nil
		}

		if b[0] == 0x00 && b[1] == 0x00 && b[2] == 0x3C && b[3] == 0x00 {
			ctx.adv(4) // BOM, consume
			return encUCS42143, nil
		}

		if b[0] == 0x00 && b[1] == 0x3C && b[2] == 0x00 && b[3] == 0x00 {
			ctx.adv(4) // BOM, consume
			return encUCS43412, nil
		}

		if b[0] == 0x4C && b[1] == 0x6F && b[2] == 0xA7 && b[3] == 0x94 {
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
		if b[0] == 0x3C && b[1] == 0x00 && b[2] == 0x3F && b[3] == 0x00 {
			ctx.adv(4)
			return encUTF16LE, nil
		}

		if b[0] == 0x00 && b[1] == 0x3C && b[2] == 0x00 && b[3] == 0x3F {
			ctx.adv(4)
			return encUTF16BE, nil
		}
	}

	if ctx.remaining() > 2 {
		b := ctx.peek(3)
		if b[0] == 0xEF && b[1] == 0xBB && b[2] == 0xBF {
			ctx.adv(3)
			return encUTF8, nil
		}
	}

	if ctx.remaining() > 1 {
		b := ctx.peek(2)
		if b[0] == 0xFE && b[1] == 0xFF {
			ctx.adv(2)
			return encUTF16BE, nil
		}
		if b[0] == 0xFF && b[1] == 0xFE {
			ctx.adv(2)
			return encUTF16LE, nil
		}
	}
	return encNone, errors.New("failed to detect encoding")
}

func (ctx *parserCtx) adv(n int) {
	ctx.idx += n

	if ctx.instate != psEOF && ctx.idx >= ctx.inputsz {
		ctx.instate = psEOF
		ctx.remain = 0
	}

	if ctx.instate != psEOF {
		diff := ctx.inputsz - ctx.idx
		if diff > 0 {
			ctx.remain = diff
		} else {
			ctx.remain = 0
		}
	}
}

func (ctx *parserCtx) get(n int) []byte {
	b := ctx.peek(n)
	ctx.adv(n)
	return b
}

func (ctx *parserCtx) getRegion(start, end int) []byte {
	return ctx.input[start:end]
}

func (ctx *parserCtx) getRunes(n int) []rune {
	rs, w := ctx.peekRunes(n)
	ctx.adv(w)
	return rs
}

// cur is alias for peekAt(0)
func (ctx *parserCtx) cur() byte {
	return ctx.peekAt(0)
}

func (ctx *parserCtx) curRune() (rune, int) {
	return ctx.peekRuneAt(0)
}

func (ctx *parserCtx) peekAt(n int) byte {
	// make sure we're within bounds
	if ctx.inputsz <= ctx.idx+n {
		return 0
	}

	return ctx.input[ctx.idx+n]
}

func (ctx *parserCtx) peekRuneAt(pos int) (rune, int) {
	for i := ctx.idx; i < ctx.inputsz; {
		r, n := utf8.DecodeRune(ctx.input[i:])
		if r == utf8.RuneError {
			return r, 0
		}

		i += n
		pos--
		if pos <= 0 {
			return r, n
		}
	}
	return utf8.RuneError, 0
}

func (ctx *parserCtx) peek(n int) []byte {
	// make sure we're within bounds
	if ctx.remaining() < n {
		return nil
	}

	b := ctx.input[ctx.idx : ctx.idx+n]
	return b
}

func (ctx *parserCtx) peekRunes(howmany int) ([]rune, int) {
	ret := make([]rune, 0, howmany)
	for i := ctx.idx; i < ctx.inputsz; {
		r, n := utf8.DecodeRune(ctx.input[i:])
		if r == utf8.RuneError {
			return nil, 0
		}

		ret = append(ret, r)
		i += n
		howmany--
		if howmany <= 0 {
			return ret, i - ctx.idx
		}
	}
	return nil, 0
}

func (ctx *parserCtx) remaining() int {
	return ctx.remain
}

func isBlankCh(c byte) bool {
	return c == 0x20 || (0x9 <= c && c <= 0xa) || c == 0xd
}

func (ctx *parserCtx) parse() error {
	// see if we can find the preliminary encoding
	if ctx.encoding == "" && len(ctx.input) > 3 {
		if enc, err := ctx.detectEncoding(); err == nil {
			ctx.encoding = enc
		}
	}

	// nothing left? eek
	if ctx.remaining() == 0 {
		return ctx.error(errors.New("empty document"))
	}

	// XML prolog
	if ctx.hasPrefix("<?xml") && isBlankCh(ctx.peekAt(5)) {
		if err := ctx.parseXMLDecl(); err != nil {
			return ctx.error(err)
		}
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
	if ctx.hasPrefix("<!DOCTYPE") {
		if err := ctx.parseDocTypeDecl(); err != nil {
			return ctx.error(err)
		}
	}

	// Start the actual tree
	if err := ctx.parseContent(); err != nil {
		return ctx.error(err)
	}

	if err := ctx.parseEpilogue(); err != nil {
		return ctx.error(err)
	}

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

	for ctx.remaining() > 0 {
		if ctx.hasPrefix("</") {
			break
		}

		if ctx.hasPrefix("<?") {
			if err := ctx.parsePI(); err != nil {
				return ctx.error(err)
			}
			continue
		}

		if ctx.hasPrefix("<![CDATA[") {
			if err := ctx.parseCDSect(); err != nil {
				return ctx.error(err)
			}
			continue
		}

		if ctx.hasPrefix("<!--") {
			if err := ctx.parseComment(); err != nil {
				return ctx.error(err)
			}
		}

		if ctx.peekAt(0) == '<' {
			if err := ctx.parseElement(); err != nil {
				return ctx.error(err)
			}
			continue
		}

		if ctx.peekAt(0) == '&' {
			panic("unimplemented (reference)")
		}

		if err := ctx.parseCharData(); err != nil {
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
func (ctx *parserCtx) parseCharData() error {
	if debug.Enabled {
		debug.Printf("START parseCharData")
		defer debug.Printf("END   parseCharData")
	}

	for i := 0; i < ctx.remaining(); i++ {
	MORE_SPACES:
		// advance until we exhaust whitespaces
		for ; i < ctx.remaining() && ctx.peekAt(i) == 0x20; i++ {
		}
		// advance until we exhaust new lines
		if ctx.peekAt(i) == 0xA {
			for ; i < ctx.remaining() && ctx.peekAt(i) == 0xA; i++ {
				ctx.lineno++
			}
			goto MORE_SPACES
		}

		if c := ctx.peekAt(i); i > 0 && c == '<' {
			str := ctx.get(i)
			if s := ctx.sax; s != nil {
				if err := s.Characters(ctx, str); err != nil {
					return ctx.error(err)
				}
			}
			return nil
		}

	GET_MORE:
		for i < ctx.remaining() {
			c := ctx.peekAt(i)
			if testCharData[c] == 0x0 {
				break
			}
			i++
		}
		// advance until we exhaust new lines
		if ctx.peekAt(i) == 0xA {
			for ; i < ctx.remaining() && ctx.peekAt(i) == 0xA; i++ {
				ctx.lineno++
			}
			goto GET_MORE
		}

		if ctx.peekAt(i) == ']' {
			if ctx.peekAt(i+1) == ']' && ctx.peekAt(i+2) == '>' {
				return ctx.error(errors.New("misplaced CDATA end"))
			}
			i++
			goto GET_MORE
		}

		str := ctx.get(i)
		if s := ctx.sax; s != nil {
			if err := s.Characters(ctx, str); err != nil {
				return ctx.error(err)
			}
		}
		// We just consumed the buffer, so need to reset the index
		// so we can index into the correct location
		i = 0

		if c := ctx.peekAt(i); c == '<' || c == '&' {
			return nil
		}
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

	if !ctx.hasPrefix("/>") {
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

	if ctx.peekAt(0) != '<' {
		return ctx.error(ErrStartTagRequired)
	}
	ctx.adv(1)

	name, err := ctx.parseName()
	if err != nil {
		return ctx.error(err)
	}

	attrs := []ParsedAttribute{}
	for ctx.instate != psEOF {
		ctx.skipBlanks()
		if ctx.peekAt(0) == '>' {
			ctx.adv(1)
			break
		}

		if ctx.peekAt(0) == '/' && ctx.peekAt(1) == '>' {
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

	if err := ctx.consumePrefix("/>"); err != nil {
		if err := ctx.consumePrefix("</"); err != nil {
			return ctx.error(ErrLtSlashRequired)
		}

		name, err := ctx.parseName()
		if err != nil {
			return ctx.error(err)
		}
		if ctx.peekAt(0) == '>' {
			ctx.adv(1)
		}

		e := ctx.peekNode()
		if e.local != name {
			return ctx.error(errors.New("closing tag does not match"))
		}
	}

	if s := ctx.sax; s != nil {
		e := ctx.popNode()
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
	return ctx.parseQuotedText(func(qch byte) (string, error) {
		return ctx.parseAttributeValueInternal(qch, normalize)
	})
}

func (ctx *parserCtx) parseAttributeValueInternal(qch byte, normalize bool) (string, error) {
	if debug.Enabled {
		debug.Printf("START parseAttributeValueInternal")
		defer debug.Printf("END   parseAttributeValueInternal")
	}
	i := 0
	for ; i < ctx.remaining(); i++ {
		c := ctx.peekAt(i)
		if c == qch || c < 0x20 || c > 0x7f || c == '&' || c == '<' {
			break
		}
	}

	return string(ctx.get(i)), nil
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

	if ctx.cur() != '=' {
		err = ctx.error(ErrEqualSignRequired)
	}
	ctx.adv(1)

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
	i := ctx.idx
	for ; i < ctx.inputsz; i++ {
		if !isBlankCh(ctx.input[i]) {
			break
		}
	}
	ctx.idx = i
}

// should only be here if current buffer is at '<?xml'
func (ctx *parserCtx) parseXMLDecl() error {
	// ctx->input->standalone = -2;

	// we already know <?xml is here
	ctx.adv(5)

	if !isBlankCh(ctx.cur()) {
		return errors.New("blank needed after '<?xml'")
	}

	ctx.skipBlanks()

	v, err := ctx.parseVersionInfo()
	if err != nil {
		return ctx.error(err)
	}
	ctx.version = v

	if !isBlankCh(ctx.cur()) {
		// if the next character isn't blank, we expect the
		// end of XML decl, so return success
		if ctx.peekAt(0) == '?' && ctx.peekAt(1) == '>' {
			ctx.adv(2)
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
		if ctx.peekAt(0) == '?' && ctx.peekAt(1) == '>' {
			ctx.adv(2)
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

	if ctx.peekAt(0) == '?' && ctx.peekAt(1) == '>' {
		ctx.adv(2)
		return nil
	}
	return ctx.error(errors.New("XML declaration not closed"))
}

func (e ErrAttrNotFound) Error() string {
	return "attribute token '" + e.Token + "' not found"
}

func (ctx *parserCtx) parseNamedAttribute(name string, cb qtextHandler) (string, error) {
	ctx.skipBlanks()
	if err := ctx.consumePrefix(name); err != nil {
		return "", ctx.error(ErrAttrNotFound{Token: name})
	}

	ctx.skipBlanks()
	if ctx.peekAt(0) != '=' {
		return "", ErrEqualSignRequired
	}

	ctx.adv(1)
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
func (ctx *parserCtx) parseVersionNum(_ byte) (string, error) {
	if v := ctx.peekAt(0); v > '9' || v < '0' {
		return "", ErrInvalidVersionNum
	}

	if v := ctx.peekAt(1); v != '.' {
		return "", ErrInvalidVersionNum
	}

	if v := ctx.peekAt(2); v > '9' || v < '0' {
		return "", ErrInvalidVersionNum
	}

	for i := 3; i < ctx.inputsz; i++ {
		if v := ctx.peekAt(i); v > '9' || v < '0' {
			return string(ctx.get(i)), nil
		}
	}
	return "", ErrInvalidVersionNum
}

type qtextHandler func(qch byte) (string, error)

func (ctx *parserCtx) parseQuotedText(cb qtextHandler) (string, error) {
	if debug.Enabled {
		debug.Printf("START parseQuotedText")
		defer debug.Printf("END   parseQuotedText")
	}

	q := ctx.peekAt(0)
	switch q {
	case '"', '\'':
		ctx.adv(1)
	default:
		return "", errors.New("string not started")
	}
	v, err := cb(q)
	if err != nil {
		return "", err
	}
	if debug.Enabled {
		debug.Printf("--> v = '%s'", v)
	}

	if ctx.peekAt(0) != q {
		return "", errors.New("string not closed")
	}
	ctx.adv(1)

	return v, nil
}

func (ctx *parserCtx) parseEncodingDecl() (string, error) {
	return ctx.parseNamedAttribute("encoding", ctx.parseEncodingName)
}

func (ctx *parserCtx) parseEncodingName(_ byte) (string, error) {
	c := ctx.cur()

	// first char needs to be alphabets
	if !(c >= 'a' && c <= 'z') && !(c >= 'A' && c <= 'Z') {
		return "", ctx.error(ErrInvalidEncodingName)
	}

	i := 1
	for ; i < ctx.inputsz; i++ {
		c = ctx.peekAt(i)
		if !(c >= 'a' && c <= 'z') && !(c >= 'A' && c <= 'Z') && !(c >= '0' && c <= '9') && c != '.' && c != '_' && c != '-' {
			break
		}
	}

	return string(ctx.get(i)), nil
}

func (ctx *parserCtx) parseStandaloneDecl() (bool, error) {
	v, err := ctx.parseNamedAttribute("standalone", ctx.parseStandaloneDeclValue)
	if err != nil {
		return false, err
	}
	return v == "yes", nil
}

func (ctx *parserCtx) parseStandaloneDeclValue(_ byte) (string, error) {
	if ctx.peekAt(0) == 'y' && ctx.peekAt(1) == 'e' && ctx.peekAt(2) == 's' {
		return string(ctx.get(3)), nil
	}

	if ctx.peekAt(0) == 'n' && ctx.peekAt(1) == 'o' {
		return string(ctx.get(2)), nil
	}

	return "", errors.New("invalid standalone declaration")
}

func (ctx *parserCtx) parseMisc() error {
	for ctx.instate != psEOF {
		if ctx.hasPrefix("<?") {
			if err := ctx.parsePI(); err != nil {
				return ctx.error(err)
			}
		} else if ctx.hasPrefix("<!--") {
			if err := ctx.parseComment(); err != nil {
				return ctx.error(err)
			}
		} else if isBlankCh(ctx.peekAt(0)) {
			ctx.skipBlanks()
		} else {
			break
		}
	}

	return nil
}

func (ctx *parserCtx) hasPrefix(s string) bool {
	return bytes.Equal(ctx.peek(len(s)), []byte(s))
}

func (ctx *parserCtx) consumePrefix(s string) error {
	if ctx.hasPrefix(s) {
		ctx.adv(len(s))
		return nil
	}
	return errors.New("prefix '" + s + "' not found")
}

var knownPIs = []string{
	"xml-stylesheet",
	"xml-model",
}

func (ctx *parserCtx) parsePI() error {
	if err := ctx.consumePrefix("<?"); err != nil {
		return ctx.error(err)
	}
	oldstate := ctx.instate
	ctx.instate = psPI
	defer func() { ctx.instate = oldstate }()

	target, err := ctx.parsePITarget()
	if err != nil {
		return ctx.error(err)
	}

	if err := ctx.consumePrefix("?>"); err == nil {
		if s := ctx.sax; s != nil {
			s.ProcessingInstruction(ctx.userData, target, "")
		}
		return ctx.error(errors.New("processing instruction not closed"))
	}

	if !isBlankCh(ctx.cur()) {
		return ctx.error(ErrSpaceRequired)
	}

	ctx.skipBlanks()
	i := 0
	for ; i < ctx.remaining(); i++ {
		if ctx.peekAt(i) == '?' && ctx.peekAt(i+1) == '>' {
			break
		}

		if !isChar(rune(ctx.peekAt(i))) {
			break
		}
	}

	data := string(ctx.get(i))

	if err := ctx.consumePrefix("?>"); err != nil {
		return ctx.error(err)
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
	if ctx.instate == psEOF {
		return "", ctx.error(ErrPrematureEOF)
	}

	c := ctx.peekAt(0)
	if !(c >= 0x61 && c <= 0x7A) && !(c >= 0x41 && c <= 0x5A) && c != '_' && c != ':' {
		// this is not simple ASCII name! go do the complex thing
		return ctx.parseNameComplex()
	}

	// at this point we have at least 1 character name.
	// see how much more we got here
	i := 1
	for ; i < ctx.inputsz; i++ {
		c = ctx.peekAt(i)
		if !(c >= 0x61 && c <= 0x7A) && !(c >= 0x41 && c <= 0x5A) && !(c >= 0x30 && c <= 0x39) && c != '_' && c != '-' && c != ':' && c != '.' {
			break
		}
	}
	if i > MaxNameLength {
		return "", ctx.error(ErrNameTooLong)
	}

	return string(ctx.get(i)), nil
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
	var v string
	v, err = ctx.parseNCName()
	if err != nil {
		oerr := err
		if ctx.cur() != ':' {
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

	if ctx.cur() != ':' {
		local = v
		err = nil
		return
	}

	ctx.adv(1)
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
	i := 0
	for r, _ := ctx.peekRuneAt(i); ctx.remaining() > 0; i++ {
		if !isNameChar(r) {
			break
		}
	}

	return string(ctx.getRunes(i)), nil
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
	if ctx.instate == psEOF {
		return "", ctx.error(ErrPrematureEOF)
	}

	c := ctx.peekAt(0)
	if !(c >= 0x61 && c <= 0x7A) && !(c >= 0x41 && c <= 0x5A) && c != '_' {
		// this is not simple ASCII name! go do the complex thing
		return ctx.parseNameComplex()
	}

	// at this point we have at least 1 character name.
	// see how much more we got here
	i := 1
	for ; i < ctx.inputsz; i++ {
		c = ctx.peekAt(i)
		if !(c >= 0x61 && c <= 0x7A) && !(c >= 0x41 && c <= 0x5A) && !(c >= 0x30 && c <= 0x39) && c != '_' && c != '-' && c != '.' {
			break
		}
	}
	if i > MaxNameLength {
		return "", ctx.error(ErrNameTooLong)
	}

	return string(ctx.get(i)), nil
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
	if err := ctx.consumePrefix("<![CDATA["); err != nil {
		return ctx.error(err)
	}
	ctx.instate = psCDATA
	defer func() { ctx.instate = psContent }()

	start := ctx.idx

	// first char
	r, rl := ctx.curRune()
	if !isChar(r) {
		return ctx.error(ErrCDATANotFinished)
	}
	ctx.adv(rl)

	// second char
	s, sl := ctx.curRune()
	if !isChar(s) {
		return ctx.error(ErrCDATANotFinished)
	}
	ctx.adv(sl)

	// third char
	cur, curl := ctx.curRune()

	for isChar(cur) && (r != ']' || s != ']' || cur != '>') {
		ctx.adv(curl)
		// trickle down one by one...
		r = s
		rl = sl
		s = cur
		sl = curl

		cur, curl = ctx.curRune()
	}
	ctx.adv(curl)

	str := ctx.getRegion(start, ctx.idx-(rl+sl+curl))
	if sh := ctx.sax; sh != nil {
		sh.CDATABlock(ctx, str)
	}
	return nil
}

func (ctx *parserCtx) parseComment() error {
	if err := ctx.consumePrefix("<!--"); err != nil {
		return ctx.error(err)
	}

	start := ctx.idx

	q, ql := ctx.curRune()
	if !isChar(q) {
		return ctx.error(ErrInvalidChar)
	}
	ctx.adv(ql)

	r, rl := ctx.curRune()
	if !isChar(r) {
		return ctx.error(ErrInvalidChar)
	}
	ctx.adv(rl)

	cur, curl := ctx.curRune()
	for isChar(cur) && (q != '-' || r !='-' || cur != '>' ) {
		ctx.adv(curl)
		if q == '-' && r == '-' {
			return ctx.error(ErrHyphenInComment)
		}

		q = r
		ql = rl
		r = cur
		rl = curl
		cur, curl = ctx.curRune()
	}
	ctx.adv(curl)

	str := ctx.getRegion(start, ctx.idx-(ql+rl+curl))
	if sh := ctx.sax; sh != nil {
		sh.Comment(ctx, str)
	}

	return nil
}

func (ctx *parserCtx) parseNameComplex() (string, error) {
	return "", nil
}

func (ctx *parserCtx) parseDocTypeDecl() error {
	return nil
}

func (ctx *parserCtx) parseEpilogue() error {
	return nil
}
