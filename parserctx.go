package helium

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/lestrrat-go/helium/encoding"
	"github.com/lestrrat-go/helium/sax"
	"github.com/lestrrat-go/pdebug"
	"github.com/lestrrat-go/strcursor"
)

type attrData struct {
	localname string
	prefix    string
	value     string
	isDefault bool
}

func (a attrData) LocalName() string { return a.localname }
func (a attrData) Prefix() string    { return a.prefix }
func (a attrData) Value() string     { return a.value }
func (a attrData) IsDefault() bool   { return a.isDefault }
func (a attrData) Name() string {
	if a.prefix != "" {
		return a.prefix + ":" + a.localname
	}
	return a.localname
}

const _parserState_name = "psEOFpsStartpsPIpsContentpsProloguepsEpiloguepsCDATApsDTDpsEntityDeclpsAttributeValuepsCommentpsStartTagpsEndTagpsSystemLiteralpsPublicLiteralpsEntityValuepsIgnorepsMisc"

var _parserState_index = [...]uint8{0, 5, 12, 16, 25, 35, 45, 52, 57, 69, 85, 94, 104, 112, 127, 142, 155, 163, 169}

func (i parserState) String() string {
	i -= -1
	if i < 0 || i >= parserState(len(_parserState_index)-1) {
		return fmt.Sprintf("parserState(%d)", i+-1)
	}
	return _parserState_name[_parserState_index[i]:_parserState_index[i+1]]
}

func (ctx *parserCtx) pushNS(prefix, uri string) {
	ctx.nsTab.Push(prefix, uri)
}

const (
	cbEntityDecl = iota
	cbGetParameterEntity
)

func (ctx *parserCtx) fireSAXCallback(typ int, args ...interface{}) error {
	// This is ugly, but I *REALLY* wanted to catch all occurences of
	// SAX callbacks being fired in one shot. optimize it later

	s := ctx.sax
	if s == nil {
		return nil
	}

	switch typ {
	case cbEntityDecl:
		if pdebug.Enabled {
			g := pdebug.Marker("EntityDecl callback")
			defer g.End()
		}
		return s.EntityDecl(ctx.userData, args[0].(string), int(InternalParameterEntity), "", "", args[1].(string))
	case cbGetParameterEntity:
		if pdebug.Enabled {
			g := pdebug.Marker("GetParameterEntity callback")
			defer g.End()
		}

		entity, err := s.GetParameterEntity(ctx, args[1].(string))
		if err == nil {
			ret := args[0].(*sax.Entity)
			*ret = entity
			if pdebug.Enabled {
				pdebug.Printf("got entity %s", entity)
			}
		}
		return err
	}
	return nil
}

func (ctx *parserCtx) pushNode(e *Element) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START pushNode (%s)", e.Name())
		defer g.IRelease("END pushNode")

		if l := ctx.nodeTab.Len(); l <= 0 {
			pdebug.Printf("  (EMPTY node stack)")
		} else {
			for i, elem := range ctx.nodeTab.SimpleStack {
				e := elem.(*Element)
				pdebug.Printf("  %003d: %s (%p)", i, e.Name(), e)
			}
		}
	}
	ctx.nodeTab.Push(e)
}

func (ctx *parserCtx) peekNode() *Element {
	return ctx.nodeTab.PeekOne()
}

func (ctx *parserCtx) popNode() (elem *Element) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START popNode")
		defer func() {
			var name string
			if elem == nil {
				name = "nil"
			} else {
				name = elem.Name()
			}
			g.IRelease("END popNode (%s)", name)
		}()

		defer func() {
			if l := ctx.nodeTab.Len(); l <= 0 {
				pdebug.Printf("  (EMPTY node stack)")
			} else {
				for i, elem := range ctx.nodeTab.SimpleStack {
					e := elem.(*Element)
					pdebug.Printf("  %003d: %s (%p)", i, e.Name(), e)
				}
			}
		}()
	}
	return ctx.nodeTab.Pop()
}

func (ctx *parserCtx) lookupNamespace(prefix string) string {
	return ctx.nsTab.Lookup(prefix)
}

func (ctx *parserCtx) release() error {
	ctx.sax = nil
	ctx.userData = nil
	return nil
}

var bufferPool = sync.Pool{
	New: allocByteBuffer,
}

func allocByteBuffer() interface{} {
	if pdebug.Enabled {
		pdebug.Printf("Allocating new bytes.Buffer...")
	}
	return &bytes.Buffer{}
}

func releaseBuffer(b *bytes.Buffer) {
	b.Reset()
	bufferPool.Put(b)
}

func (ctx *parserCtx) pushInput(in interface{}) {
	if pdebug.Enabled {
		pdebug.Printf("pushInput (n = %d -> %d)", ctx.inputTab.Len(), ctx.inputTab.Len()+1)
	}
	ctx.inputTab.Push(in)
}

func (ctx *parserCtx) getByteCursor() *strcursor.ByteCursor {
	cur, ok := ctx.inputTab.PeekOne().(*strcursor.ByteCursor)
	if !ok {
		return nil
	}
	return cur
}

func (ctx *parserCtx) getCursor() strcursor.Cursor {
	// Pop exhausted input streams and return the next available cursor
	for ctx.inputTab.Len() > 0 {
		cur, ok := ctx.inputTab.PeekOne().(strcursor.Cursor)
		if !ok {
			ctx.popInput()
			continue
		}
		if cur.Done() && ctx.inputTab.Len() > 1 {
			// Current input is exhausted, pop it and try the next
			if pdebug.Enabled {
				pdebug.Printf("Popping exhausted input stream, stack depth: %d -> %d", ctx.inputTab.Len(), ctx.inputTab.Len()-1)
			}
			ctx.popInput()
			continue
		}
		return cur
	}
	return nil
}

func (ctx *parserCtx) popInput() interface{} {
	return ctx.inputTab.Pop()
}

func (ctx *parserCtx) init(p *Parser, in io.Reader) error {
	ctx.pushInput(strcursor.NewByteCursor(in))
	ctx.detectedEncoding = encUTF8
	ctx.encoding = ""
	ctx.in = in
	ctx.nbread = 0
	ctx.keepBlanks = true
	ctx.instate = psStart
	ctx.userData = ctx // circular dep?!
	ctx.standalone = StandaloneImplicitNo
	ctx.attsSpecial = map[string]AttributeType{}
	ctx.attsDefault = map[string]map[string]*Attribute{}
	ctx.wellFormed = true
	if p != nil {
		ctx.sax = p.sax
	}
	return nil
}

func (ctx *parserCtx) error(err error) error {
	// If it's wrapped, just return as is
	if _, ok := err.(ErrParseError); ok {
		return err
	}

	e := ErrParseError{Err: err}
	if cur := ctx.getCursor(); cur != nil {
		e.Column = cur.Column()
		e.LineNumber = cur.LineNumber()
		e.Line = cur.Line()
	}
	return e
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

	// Use HasPrefix here because we don't want to consume it
	if cur.HasPrefix(patEBCDIC) {
		encoding = encEBCDIC
		return
	}

	// Use HasPrefix here because we don't want to consume it
	if cur.HasPrefix(patMaybeXMLDecl) {
		encoding = encUTF8
		return
	}

	/*
	 * Although not part of the recommendation, we also
	 * attempt an "auto-recognition" of UTF-16LE and
	 * UTF-16BE encodings.
	 */
	if cur.Consume(patUTF16LE4B) {
		encoding = encUTF16LE
		return
	}

	if cur.Consume(patUTF16BE4B) {
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
	ctx.pushInput(strcursor.NewRuneCursor(b))

	return nil
}

var xmlDeclHint = []byte{'<', '?', 'x', 'm', 'l'}

func (ctx *parserCtx) parseDocument() error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseDocument")
		defer g.IRelease("END parseDocument")
	}

	if s := ctx.sax; s != nil {
		switch err := s.SetDocumentLocator(ctx.userData, nil); err {
		case nil, sax.ErrHandlerUnspecified:
			// no op
		default:
			return ctx.error(err)
		}
	}

	// see if we can find the preliminary encoding
	if ctx.encoding == "" {
		if enc, err := ctx.detectEncoding(); err == nil {
			ctx.detectedEncoding = enc
		}
	}

	// At this stage we MUST be using a ByteCursor, as we
	// don't know what the encoding is.
	bcur := ctx.getByteCursor()
	if bcur == nil {
		return ctx.error(ErrByteCursorRequired)
	}

	// nothing left? eek
	if bcur.Done() {
		return ctx.error(errors.New("empty document"))
	}

	// XML prolog
	if bcur.HasPrefix(xmlDeclHint) {
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
		switch err := s.StartDocument(ctx.userData); err {
		case nil, sax.ErrHandlerUnspecified:
			// no op
		default:
			return ctx.error(err)
		}
	}

	// Misc part of the prolog
	if err := ctx.parseMisc(); err != nil {
		return ctx.error(err)
	}

	cur := ctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	// Doctype declarations and more misc
	if cur.HasPrefixString("<!DOCTYPE") {
		ctx.inSubset = inInternalSubset
		if err := ctx.parseDocTypeDecl(); err != nil {
			return ctx.error(err)
		}

		if cur.HasPrefixString("[") {
			ctx.instate = psDTD
			if err := ctx.parseInternalSubset(); err != nil {
				return ctx.error(err)
			}
		}

		ctx.inSubset = inExternalSubset
		if s := ctx.sax; s != nil {
			switch err := s.ExternalSubset(ctx.userData, ctx.intSubName, ctx.extSubSystem, ctx.extSubURI); err {
			case nil, sax.ErrHandlerUnspecified:
				// no op
			default:
				return ctx.error(err)
			}
		}
		if ctx.instate == psEOF {
			return ctx.error(errors.New("unexpected EOF"))
		}
		ctx.inSubset = notInSubset

		ctx.cleanSpecialAttributes()

		ctx.instate = psPrologue
		if err := ctx.parseMisc(); err != nil {
			return ctx.error(err)
		}
	}
	ctx.skipBlanks()

	if cur.Peek() != '<' {
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
		if !cur.Done() {
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
		switch err := s.EndDocument(ctx.userData); err {
		case nil, sax.ErrHandlerUnspecified:
			// no op
		default:
			return ctx.error(err)
		}
	}

	return nil
}

func (ctx *parserCtx) parseContent() error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseContent")
		defer g.IRelease("END parseContent")
	}
	ctx.instate = psContent

	cur := ctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}

	for !cur.Done() {
		if cur.HasPrefixString("</") {
			break
		}

		if cur.HasPrefixString("<?") {
			if err := ctx.parsePI(); err != nil {
				return ctx.error(err)
			}
			continue
		}

		if cur.HasPrefixString("<![CDATA[") {
			if err := ctx.parseCDSect(); err != nil {
				return ctx.error(err)
			}
			continue
		}

		if cur.HasPrefixString("<!--") {
			if err := ctx.parseComment(); err != nil {
				return ctx.error(err)
			}
			continue
		}

		if cur.HasPrefixString("<") {
			if err := ctx.parseElement(); err != nil {
				return ctx.error(err)
			}
			continue
		}

		if cur.HasPrefixString("&") {
			if err := ctx.parseReference(); err != nil {
				return ctx.error(err)
			}
			continue
		}

		if err := ctx.parseCharData(false); err != nil {
			return err
		}
	}

	return nil
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
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseCharData")
		defer g.IRelease("END parseCharData")
	}

	buf := bufferPool.Get().(*bytes.Buffer)
	defer releaseBuffer(buf)

	cur := ctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}

	i := 0
	for c := cur.PeekN(i + 1); c != 0x0; c = cur.PeekN(i + 1) {
		if !cdata {
			if c == '<' || c == '&' || !isChar(c) {
				break
			}
		}

		if c == ']' && cur.PeekN(i+2) == ']' && cur.PeekN(i+3) == '>' {
			if cdata {
				break
			}

			return ctx.error(ErrMisplacedCDATAEnd)
		}

		buf.WriteRune(c)
		i++
	}

	if i <= 0 {
		pdebug.Dump(cur)
		return errors.New("Invalid char data")
	}

	cur.Advance(i)
	str := buf.String()

	// XXX This is not right, but it's for now the best place to do this
	str = strings.Replace(str, "\r\n", "\n", -1)
	if ctx.instate == psCDATA {
		if s := ctx.sax; s != nil {
			switch err := s.CDataBlock(ctx.userData, []byte(str)); err {
			case nil, sax.ErrHandlerUnspecified:
				// no op
			default:
				return ctx.error(err)
			}
		}
	} else if ctx.areBlanks(str, false) {
		if s := ctx.sax; s != nil {
			switch err := s.IgnorableWhitespace(ctx.userData, []byte(str)); err {
			case nil, sax.ErrHandlerUnspecified:
				// no op
			default:
				return ctx.error(err)
			}
		}
	} else {
		if s := ctx.sax; s != nil {
			switch err := s.Characters(ctx.userData, []byte(str)); err {
			case nil, sax.ErrHandlerUnspecified:
				// no op
			default:
				return ctx.error(err)
			}
		}
	}

	return nil
}

func (ctx *parserCtx) parseElement() error {
	if pdebug.Enabled {
		ctx.elemidx++
		i := ctx.elemidx
		g := pdebug.IPrintf("START parseElement (%d)", i)
		defer g.IRelease("END parseElement (%d)", i)
	}

	// parseStartTag only parses up to the attributes.
	// For example, given <foo>bar</foo>, the next token would
	// be bar</foo>. Given <foo />, the next token would
	// be />
	if err := ctx.parseStartTag(); err != nil {
		return ctx.error(err)
	}

	cur := ctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if !cur.HasPrefixString("/>") {
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
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseStartTag")
		defer g.IRelease("END parseStartTag")
	}

	cur := ctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if cur.Peek() != '<' {
		return ctx.error(ErrStartTagRequired)
	}
	cur.Advance(1)

	local, prefix, err := ctx.parseQName()
	if local == "" {
		return ctx.error(fmt.Errorf("local name empty! local = %s, prefix = %s, err = %s", local, prefix, err))
	}
	if err != nil {
		return ctx.error(err)
	}

	elem, err := ctx.doc.CreateElement(local)
	if err != nil {
		return ctx.error(err)
	}

	nbNs := 0
	attrs := []sax.Attribute{}
	for ctx.instate != psEOF {
		ctx.skipBlanks()
		if cur.Peek() == '>' {
			cur.Advance(1)
			break
		}

		if cur.Peek() == '/' && cur.PeekN(2) == '>' {
			break
		}
		attname, aprefix, attvalue, err := ctx.parseAttribute(local)
		if err != nil {
			return ctx.error(err)
		}

		if attname == XMLNsPrefix && aprefix == "" {
			// <elem xmlns="...">
			ctx.pushNS("", attvalue)
			nbNs++
			//    SkipDefaultNS:
			if cur.Peek() == '>' || cur.HasPrefixString("/>") {
				continue
			}

			if !isBlankCh(cur.Peek()) {
				return ctx.error(ErrSpaceRequired)
			}
			ctx.skipBlanks()
		} else if aprefix == XMLNsPrefix {
			var u *url.URL // predeclare, so we can use goto SkipNS

			// <elem xmlns:foo="...">
			if attname == XMLPrefix { // xmlns:xml
				if attvalue != XMLNamespace {
					return ctx.error(errors.New("xml namespace prefix mapped to wrong URI"))
				}
				// skip storing namespace definition
				goto SkipNS
			}
			if attname == XMLNsPrefix { // xmlns:xmlns="..."
				return ctx.error(errors.New("redefinition of the xmlns prefix forbidden"))
			}

			if attvalue == "http://www.w3.org/2000/xmlns/" {
				return ctx.error(errors.New("reuse of the xmlns namespace name if forbidden"))
			}

			if attvalue == "" {
				return ctx.error(fmt.Errorf("xmlns:%s: Empty XML namespace is not allowed", attname))
			}

			u, err = url.Parse(attvalue)
			if err != nil {
				return ctx.error(fmt.Errorf("xmlns:%s: '%s' is not a validURI", attname, attvalue))
			}
			if ctx.pedantic && u.Scheme == "" {
				return ctx.error(fmt.Errorf("xmlns:%s: URI %s is not absolute", attname, attvalue))
			}

			if ctx.nsTab.Lookup(attname) != "" {
				return ctx.error(errors.New("duplicate attribute is not allowed"))
			}
			ctx.pushNS(attname, attvalue)
			nbNs++

		SkipNS:
			if cur.Peek() == '>' || cur.HasPrefixString("/>") {
				continue
			}

			if !isBlankCh(cur.Peek()) {
				return ctx.error(ErrSpaceRequired)
			}
			ctx.skipBlanks()
			// ctx.input.base != base || inputNr != ctx.inputNr; goto base_changed
			continue
		}

		// Due to various reasons, we cannot create a real Attribute object
		// here. So we create a simple holder for attribute data
		attr := &attrData{
			localname: attname,
			prefix:    aprefix,
			value:     attvalue,
		}

		attrs = append(attrs, attr)
	}

	// attributes defaulting
	// XXX Punting a lot of stuff here. See xmlParseStartTag2
	if len(ctx.attsDefault) > 0 {
		var elemName string
		if prefix != "" {
			elemName = prefix + ":" + local
		} else {
			elemName = local
		}

		if pdebug.Enabled {
			pdebug.Printf("-------> %s", elemName)
		}
		defaults, ok := ctx.lookupAttributeDefault(elemName)
		if ok {
			for _, attr := range defaults {
				attrs = append(attrs, attr)
			}
		}
	}

	// we push the element first, because this way we get to
	// query for the namespace declared on this node as well
	// via lookupNamespace
	nsuri := ctx.lookupNamespace(prefix)
	if prefix != "" && nsuri == "" {
		return ctx.error(errors.New("namespace '" + prefix + "' not found"))
	}
	if nsuri != "" {
		elem.SetNamespace(prefix, nsuri, true)
	}

	if s := ctx.sax; s != nil {
		var nslist []sax.Namespace
		if nbNs > 0 {
			nslist = make([]sax.Namespace, nbNs)
			// workaround []*Namespace != []sax.Namespace
			for i, ns := range ctx.nsTab.Peek(nbNs) {
				nslist[i] = ns.(nsStackItem)
			}
		}
		switch err := s.StartElementNS(ctx.userData, elem.LocalName(), prefix, nsuri, nslist, attrs); err {
		case nil, sax.ErrHandlerUnspecified:
			// no op
		default:
			return ctx.error(err)
		}
	}
	ctx.pushNode(elem)

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
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseEndTag")
		defer g.IRelease("END parseEndTag")
	}

	cur := ctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if !cur.ConsumeString("/>") {
		if !cur.ConsumeString("</") {
			return ctx.error(ErrLtSlashRequired)
		}

		e := ctx.peekNode()
		if !cur.ConsumeString(e.Name()) {
			return ctx.error(errors.New("expected end tag '" + e.Name() + "'"))
		}

		if cur.Peek() != '>' {
			return ctx.error(ErrGtRequired)
		}
		cur.Advance(1)
	}

	e := ctx.peekNode()
	if s := ctx.sax; s != nil {
		switch err := s.EndElementNS(ctx, e.LocalName(), e.Prefix(), e.URI()); err {
		case nil, sax.ErrHandlerUnspecified:
			// no op
		default:
			return ctx.error(err)
		}
	}
	ctx.popNode()

	return nil
}

func (ctx *parserCtx) parseAttributeValue(normalize bool) (value string, entities int, err error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseAttributeValue (normalize=%t)", normalize)
		defer g.IRelease("END parseAttributeValue")
	}

	ctx.parseQuotedText(func(qch rune) (string, error) {
		value, entities, err = ctx.parseAttributeValueInternal(qch, normalize)
		return "", nil
	})
	return
}

// This is based on xmlParseAttValueComplex
func (ctx *parserCtx) parseAttributeValueInternal(qch rune, normalize bool) (value string, entities int, err error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseAttributeValueInternal (qch='%c',normalize=%t)", qch, normalize)
		defer g.IRelease("END parseAttributeValueInternal")
		defer func() {
			pdebug.Printf("value = '%s'", value)
		}()
	}

	cur := ctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	inSpace := false
	b := bufferPool.Get().(*bytes.Buffer)
	defer releaseBuffer(b)

	for {
		c := cur.Peek()
		// qch == quote character.
		if (qch != 0x0 && c == qch) || !isChar(c) || c == '<' {
			break
		}
		switch c {
		case '&':
			entities++
			inSpace = false
			if cur.PeekN(2) == '#' {
				var r rune
				r, err = ctx.parseCharRef()
				if err != nil {
					err = ctx.error(err)
					return
				}

				if r == '&' && !ctx.replaceEntities {
					b.WriteString("&#38;")
				} else {
					b.WriteRune(r)
				}
			} else {
				var ent *Entity
				ent, err = ctx.parseEntityRef()
				if err != nil {
					err = ctx.error(err)
					return
				}

				if ent.entityType == InternalPredefinedEntity {
					if ent.content == "&" && !ctx.replaceEntities {
						b.WriteString("&#38;")
					} else {
						b.WriteString(ent.content)
					}
				} else if ctx.replaceEntities {
					var rep string
					rep, err = ctx.decodeEntities(ent.Content(), SubstituteRef)
					if err != nil {
						err = ctx.error(err)
						return
					}
					for i := 0; i < len(rep); i++ {
						switch rep[i] {
						case 0xD, 0xA, 0x9:
							b.WriteByte(0x20)
						default:
							b.WriteByte(rep[i])
						}
					}
				} else {
					b.WriteString("&")
					b.WriteString(ent.name)
					b.WriteString(";")
				}
			}
		case 0x20, 0xD, 0xA, 0x9:
			if b.Len() > 0 || !normalize {
				if !normalize || !inSpace {
					b.WriteRune(0x20)
				}
				inSpace = true
			}
			cur.Advance(1)
		default:
			inSpace = false
			b.WriteRune(c)
			cur.Advance(1)
		}
	}

	value = b.String()
	if inSpace && normalize {
		if value[len(value)-1] == 0x20 {
			for len(value) > 0 {
				if value[len(value)-1] != 0x20 {
					break
				}
				value = value[:len(value)-1]
			}
		}
	}

	return
}

func (ctx *parserCtx) parseAttribute(elemName string) (local string, prefix string, value string, err error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseAttribute")
		defer g.IRelease("END parseAttribute")
		defer func() {
			pdebug.Printf("local = '%s', prefix = '%s', value = '%s'", local, prefix, value)
		}()
	}
	l, p, err := ctx.parseQName()
	if err != nil {
		err = ctx.error(err)
		return
	}

	normalize := false
	attType, ok := ctx.lookupSpecialAttribute(elemName, l)
	if pdebug.Enabled {
		pdebug.Printf("looked up attribute %s:%s -> %d (%t)", elemName, l, attType, ok)
	}
	if ok && attType != AttrInvalid {
		normalize = true
	}
	ctx.skipBlanks()

	cur := ctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if cur.Peek() != '=' {
		err = ctx.error(ErrEqualSignRequired)
	}
	cur.Advance(1)
	ctx.skipBlanks()

	v, entities, err := ctx.parseAttributeValue(normalize)
	if err != nil {
		err = ctx.error(err)
		return
	}

	/*
	 * Sometimes a second normalisation pass for spaces is needed
	 * but that only happens if charrefs or entities refernces
	 * have been used in the attribute value, i.e. the attribute
	 * value have been extracted in an allocated string already.
	 */
	if normalize {
		if pdebug.Enabled {
			pdebug.Printf("normalize is true, checking if entities have been expanded...")
		}
		if entities > 0 {
			if pdebug.Enabled {
				pdebug.Printf("entities seems to have been expanded (%d): doint second normalization", entities)
			}
			v = ctx.attrNormalizeSpace(v)
		}
	}

	// If this is one of those the well known tags, check for the validity
	// of the attribute value

	local = l
	prefix = p
	value = v
	err = nil
	return
}

func (ctx *parserCtx) skipBlanks() bool {
	i := 0
	if pdebug.Enabled {
		g := pdebug.IPrintf("START skipBlanks")
		defer func() {
			g.IRelease("END skipBlanks (skipped %d)", i)
		}()
	}
	cur := ctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	for c := cur.PeekN(i + 1); isBlankCh(c) && !cur.Done(); c = cur.PeekN(i + 1) {
		i++
	}
	if i > 0 {
		cur.Advance(i)

		if cur.Peek() == '%' {
			pdebug.Printf("Found possible parameter entity reference")
			ctx.handlePEReference()
		}
		return true
	}
	return false
}

func (ctx *parserCtx) skipBlankBytes(cur *strcursor.ByteCursor) bool {
	i := 0
	if pdebug.Enabled {
		g := pdebug.IPrintf("START skipBlankBytes")
		defer func() {
			g.IRelease("END skipBlankBytes (skipped %d)", i)
		}()
	}
	for c := cur.PeekN(i + 1); c != 0x0 && isBlankCh(rune(c)); c = cur.PeekN(i + 1) {
		i++
	}
	if i > 0 {
		cur.Advance(i)

		if cur.Peek() == '%' {
			pdebug.Printf("Found possible parameter entity reference")
			ctx.handlePEReference()
		}
		return true
	}
	return false
}

// should only be here if current buffer is at '<?xml'
func (ctx *parserCtx) parseXMLDecl() error {
	cur := ctx.getByteCursor()
	if cur == nil {
		return ErrByteCursorRequired
	}

	if !cur.Consume(xmlDeclHint) {
		return ctx.error(ErrInvalidXMLDecl)
	}

	if !ctx.skipBlankBytes(cur) {
		return errors.New("blank needed after '<?xml'")
	}

	v, err := ctx.parseVersionInfo()
	if err != nil {
		return ctx.error(err)
	}
	ctx.version = v

	if !isBlankCh(rune(cur.Peek())) {
		// if the next character isn't blank, we expect the
		// end of XML decl, so return success
		if cur.Peek() == '?' && cur.PeekN(2) == '>' {
			cur.Advance(2)
			return nil
		}

		// otherwise, we just saw something unexpected
		return ctx.error(ErrSpaceRequired)
	}

	// we *may* have encoding decl
	v, err = ctx.parseEncodingDecl()
	if err == nil {
		// ctx.encoding contains the explicit encoding specified
		ctx.encoding = v

		// if the encoding decl is found, then we *could* have
		// the end of the XML declaration
		if cur.Peek() == '?' && cur.PeekN(2) == '>' {
			cur.Advance(2)
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

	if cur.Peek() == '?' && cur.PeekN(2) == '>' {
		cur.Advance(2)
		return nil
	}
	return ctx.error(errors.New("XML declaration not closed"))
}

func (e ErrAttrNotFound) Error() string {
	return "attribute token '" + e.Token + "' not found"
}

func (ctx *parserCtx) parseNamedAttribute(name string, cb qtextHandler) (string, error) {
	ctx.skipBlanks()

	cur := ctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if !cur.ConsumeString(name) {
		return "", ctx.error(ErrAttrNotFound{Token: name})
	}

	ctx.skipBlanks()
	if cur.Peek() != '=' {
		return "", ErrEqualSignRequired
	}

	cur.Advance(1)
	ctx.skipBlanks()
	return ctx.parseQuotedText(cb)
}

// parse the XML version info (version="1.0")
var versionBytes = []byte{'v', 'e', 'r', 's', 'i', 'o', 'n'}

func (ctx *parserCtx) parseVersionInfo() (string, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseVersionInfo")
		defer g.IRelease("END parseVersionInfo")
	}

	return ctx.parseNamedAttributeBytes(versionBytes, ctx.parseVersionNum)
}

func (ctx *parserCtx) parseNamedAttributeBytes(name []byte, valueParser qtextHandler) (string, error) {
	cur := ctx.getByteCursor()
	if cur == nil {
		return "", ErrByteCursorRequired
	}

	ctx.skipBlankBytes(cur)
	if !cur.Consume(name) {
		return "", ctx.error(ErrAttrNotFound{Token: string(name)})
	}

	ctx.skipBlankBytes(cur)
	if cur.Peek() != '=' {
		return "", ErrEqualSignRequired
	}
	cur.Advance(1)

	ctx.skipBlankBytes(cur)

	return ctx.parseQuotedTextBytes(valueParser)
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
	cur := ctx.getByteCursor()
	if cur == nil {
		return "", ErrByteCursorRequired
	}

	if v := cur.Peek(); v > '9' || v < '0' {
		return "", ErrInvalidVersionNum
	}

	if v := cur.PeekN(2); v != '.' {
		return "", ErrInvalidVersionNum
	}

	if v := cur.PeekN(3); v > '9' || v < '0' {
		return "", ErrInvalidVersionNum
	}

	for i := 4; ; i++ {
		if v := cur.PeekN(i); v > '9' || v < '0' {
			b := bufferPool.Get().(*bytes.Buffer)
			defer releaseBuffer(b)

			for x := 1; x < i; x++ {
				b.WriteRune(cur.PeekN(x))
			}
			cur.Advance(i - 1)
			return b.String(), nil
		}
	}
	return "", ErrInvalidVersionNum
}

type qtextHandler func(qch rune) (string, error)

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
		cur.Advance(1)
	default:
		err = errors.New("string not started (got '" + string([]rune{q}) + "')")
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
	cur.Advance(1)

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
		panic("did not get rune cursor")
	}
	q := cur.Peek()
	switch q {
	case '"', '\'':
		cur.Advance(1)
	default:
		err = errors.New("string not started (got '" + string([]rune{q}) + "')")
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
	cur.Advance(1)

	return
}

var encodingBytes = []byte{'e', 'n', 'c', 'o', 'd', 'i', 'n', 'g'}

func (ctx *parserCtx) parseEncodingDecl() (string, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseEncodingDecl")
		defer g.IRelease("END parseEncodingDecl")
	}
	return ctx.parseNamedAttributeBytes(encodingBytes, ctx.parseEncodingName)
}

func (ctx *parserCtx) parseEncodingName(_ rune) (string, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseEncodingName")
		defer g.IRelease("END parseEncodingName")
	}
	cur := ctx.getByteCursor()
	if cur == nil {
		return "", ErrByteCursorRequired
	}
	c := cur.Peek()

	buf := bufferPool.Get().(*bytes.Buffer)
	defer releaseBuffer(buf)

	// first char needs to be alphabets
	if !(c >= 'a' && c <= 'z') && !(c >= 'A' && c <= 'Z') {
		return "", ctx.error(ErrInvalidEncodingName)
	}
	buf.WriteRune(c)

	i := 2
	for c = cur.PeekN(i); c != 0x0; c = cur.PeekN(i) {
		if !(c >= 'a' && c <= 'z') && !(c >= 'A' && c <= 'Z') && !(c >= '0' && c <= '9') && c != '.' && c != '_' && c != '-' {
			i--
			break
		}
		buf.WriteRune(c)
		i++
	}

	cur.Advance(i)

	return buf.String(), nil
}

var standaloneBytes = []byte{'s', 't', 'a', 'n', 'd', 'a', 'l', 'o', 'n', 'e'}

func (ctx *parserCtx) parseStandaloneDecl() (DocumentStandaloneType, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseStandaloneDecl")
		defer g.IRelease("END parseStandaloneDecl")
	}

	v, err := ctx.parseNamedAttributeBytes(standaloneBytes, ctx.parseStandaloneDeclValue)
	if err != nil {
		return StandaloneInvalidValue, err
	}
	if v == "yes" {
		return StandaloneExplicitYes, nil
	} else {
		return StandaloneExplicitNo, nil
	}
}

func (ctx *parserCtx) parseStandaloneDeclValue(_ rune) (string, error) {
	const (
		yes = "yes"
		no  = "no"
	)
	cur := ctx.getByteCursor()
	if cur == nil {
		return "", ErrByteCursorRequired
	}
	if cur.ConsumeString(yes) {
		return string(yes), nil
	}

	if cur.ConsumeString(no) {
		return string(no), nil
	}

	return "", errors.New("invalid standalone declaration")
}

func (ctx *parserCtx) parseMisc() error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseMisc")
		defer g.IRelease("END parseMisc")
	}

	cur := ctx.getCursor()
	for !cur.Done() && ctx.instate != psEOF {
		if cur.HasPrefixString("<?") {
			if err := ctx.parsePI(); err != nil {
				return ctx.error(err)
			}
		} else if cur.HasPrefixString("<!--") {
			if err := ctx.parseComment(); err != nil {
				return ctx.error(err)
			}
		} else if isBlankCh(cur.Peek()) {
			ctx.skipBlanks()
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

func (ctx *parserCtx) parsePI() error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parsePI")
		defer g.IRelease("END parsePI")
	}

	cur := ctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if !cur.ConsumeString("<?") {
		return ctx.error(ErrInvalidProcessingInstruction)
	}
	oldstate := ctx.instate
	ctx.instate = psPI
	defer func() { ctx.instate = oldstate }()

	target, err := ctx.parsePITarget()
	if err != nil {
		return ctx.error(err)
	}

	if cur.ConsumeString("?>") {
		if s := ctx.sax; s != nil {
			switch err := s.ProcessingInstruction(ctx.userData, target, ""); err {
			case nil, sax.ErrHandlerUnspecified:
				// no op
			default:
				return ctx.error(err)
			}
		}
		return nil
	}

	if !isBlankCh(cur.Peek()) {
		return ctx.error(ErrSpaceRequired)
	}

	ctx.skipBlanks()
	buf := bufferPool.Get().(*bytes.Buffer)
	defer releaseBuffer(buf)

	i := 0
	for c := cur.PeekN(i + 1); c != 0x0; c = cur.PeekN(i + 1) {
		if c == '?' && cur.PeekN(i+2) == '>' {
			break
		}

		if !isChar(c) {
			break
		}
		buf.WriteRune(c)
		i++
	}

	cur.Advance(i)
	data := buf.String()

	if !cur.ConsumeString("?>") {
		return ctx.error(ErrInvalidProcessingInstruction)
	}

	if s := ctx.sax; s != nil {
		switch err := s.ProcessingInstruction(ctx.userData, target, data); err {
		case nil, sax.ErrHandlerUnspecified:
			// no op
		default:
			return ctx.error(err)
		}
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
func (ctx *parserCtx) parseName() (name string, err error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseName")
		defer g.IRelease("END parseName")
		defer func() { pdebug.Printf("name = '%s'", name) }()
	}
	if ctx.instate == psEOF {
		err = ctx.error(ErrPrematureEOF)
		return
	}

	cur := ctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}

	buf := bufferPool.Get().(*bytes.Buffer)
	defer releaseBuffer(buf)

	// first letter
	c := cur.Peek()
	if c == ' ' || c == '>' || c == '/' || /* accelerators */ (!unicode.IsLetter(c) && c != '_' && c != ':') {
		err = ctx.error(fmt.Errorf("invalid first letter '%c'", c))
		return
	}
	buf.WriteRune(c)

	i := 2
	for c = cur.PeekN(i); c != 0x0; c = cur.PeekN(i) {
		if c == ' ' || c == '>' || c == '/' { /* accelerator */
			i--
			break
		}
		if !unicode.IsLetter(c) && !unicode.IsDigit(c) && c != '.' && c != '-' && c != '_' && c != ':' /* && !isCombining(c) && !isExtender(c) */ {
			i--
			break
		}
		buf.WriteRune(c)

		i++
	}
	if i > MaxNameLength {
		err = ctx.error(ErrNameTooLong)
		return
	}

	cur.Advance(i)
	name = buf.String()
	if name == "" {
		err = ctx.error(errors.New("internal error: parseName returned with empty name"))
		return
	}
	err = nil
	return
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
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseQName")
		defer g.IRelease("END parseQName")
		defer func() { pdebug.Printf("local='%s' prefix='%s'", local, prefix) }()
	}

	cur := ctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	var v string
	v, err = ctx.parseNCName()
	if err != nil {
		oerr := err
		if cur.Peek() != ':' {
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

	if cur.Peek() != ':' {
		local = v
		err = nil
		return
	}

	cur.Advance(1)
	prefix = v

	v, err = ctx.parseNCName()
	if err == nil {
		local = v
		return
	}

	v, err = ctx.parseNmtoken()
	if err == nil {
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

func isNameStartChar(r rune) bool {
	return unicode.IsLetter(r) || r == '_' || r == ':'
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
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseNmtoken")
		defer g.IRelease("END parseNmtoken")
	}

	i := 1
	cur := ctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	buf := bufferPool.Get().(*bytes.Buffer)
	defer releaseBuffer(buf)

	for c := cur.PeekN(i); c != 0x0; i++ {
		if !isNameChar(c) {
			break
		}
		buf.WriteRune(c)
	}

	return buf.String(), nil
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
func (ctx *parserCtx) parseNCName() (ncname string, err error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseNCName")
		defer g.IRelease("END parseNCName")
		defer func() {
			pdebug.Printf("ncname = '%s'", ncname)
		}()
	}
	if ctx.instate == psEOF {
		err = ctx.error(ErrPrematureEOF)
		return
	}

	cur := ctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	buf := bufferPool.Get().(*bytes.Buffer)
	defer releaseBuffer(buf)

	var c rune
	if c = cur.Peek(); c == ' ' || c == '>' || c == '/' || !isNameStartChar(c) {
		err = ctx.error(errors.New("invalid name start char"))
		return
	}
	buf.WriteRune(c)

	// at this point we have at least 1 character name.
	// see how much more we got here
	i := 2
	for c = cur.PeekN(i); c != 0x0; c = cur.PeekN(i) {
		if !unicode.IsLetter(c) && !unicode.IsDigit(c) && c != '_' && c != '-' && c != '.' {
			i--
			break
		}
		buf.WriteRune(c)
		i++
	}
	if i > MaxNameLength {
		err = ctx.error(ErrNameTooLong)
		return
	}
	cur.Advance(i)
	ncname = buf.String()
	return
}

func (ctx *parserCtx) parsePITarget() (string, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parsePITarget")
		defer g.IRelease("END parsePITarget")
	}

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
func (ctx *parserCtx) areBlanks(s string, blankChars bool) (ret bool) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START areBlanks (%v)", []byte(s))
		defer g.IRelease("END areBlanks")
		defer func() { pdebug.Printf("ret = '%t'", ret) }()
	}

	// Check for xml:space value.
	if ctx.space == 1 || ctx.space == -2 {
		ret = false
		return
	}

	// Check that the string is made of blanks
	if !blankChars {
		for _, r := range s {
			if !isBlankCh(r) {
				ret = false
				return
			}
		}
	}

	// Look if the element is mixed content in the DTD if available
	if ctx.peekNode() == nil {
		ret = false
		return
	}
	if ctx.doc != nil {
		ok, _ := ctx.doc.IsMixedElement(ctx.peekNode().Name())
		ret = !ok
		return
	}

	cur := ctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if c := cur.Peek(); c != '<' && c != 0xD {
		ret = false
		return
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
	ret = true
	return
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
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseCDSect")
		defer g.IRelease("END parseCDSect")
	}

	cur := ctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if !cur.ConsumeString("<![CDATA[") {
		return ctx.error(ErrInvalidCDSect)
	}

	ctx.instate = psCDATA
	defer func() { ctx.instate = psContent }()

	if err := ctx.parseCharData(true); err != nil {
		return ctx.error(err)
	}

	if !cur.ConsumeString("]]>") {
		return ctx.error(ErrCDATANotFinished)
	}
	return nil
}

func (ctx *parserCtx) parseComment() error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseComment")
		defer g.IRelease("END parseComment")
	}

	cur := ctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if !cur.ConsumeString("<!--") {
		return ctx.error(ErrInvalidComment)
	}

	buf := bufferPool.Get().(*bytes.Buffer)
	defer releaseBuffer(buf)

	i := 0
	q := cur.PeekN(i + 1)
	if !isChar(q) {
		return ctx.error(ErrInvalidChar)
	}
	i++
	buf.WriteRune(q)

	r := cur.PeekN(i + 1)
	if !isChar(r) {
		return ctx.error(ErrInvalidChar)
	}
	i++
	buf.WriteRune(r)

	for c := cur.PeekN(i + 1); isChar(c) && (q != '-' || r != '-' || c != '>'); c = cur.PeekN(i + 1) {
		if q == '-' && r == '-' {
			return ctx.error(ErrHyphenInComment)
		}
		buf.WriteRune(c)
		q = r
		r = c
		i++
	}

	// -2 for "-->" (note: '>' has not been consumed, so we use -2 instead of -3
	buf.Truncate(buf.Len() - 2)
	str := buf.Bytes()
	// i+1 because '>' was not consumed in the loop
	cur.Advance(i + 1)

	if sh := ctx.sax; sh != nil {
		str = bytes.Replace(str, []byte{'\r', '\n'}, []byte{'\n'}, -1)
		switch err := sh.Comment(ctx, str); err {
		case nil, sax.ErrHandlerUnspecified:
			// no op
		default:
			return ctx.error(err)
		}
	}

	return nil
}

func (ctx *parserCtx) parseDocTypeDecl() error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseDocTypeDecl")
		defer g.IRelease("END parseDocTypeDecl")
	}

	cur := ctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if !cur.ConsumeString("<!DOCTYPE") {
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
		switch err := s.InternalSubset(ctx.userData, name, eid, u); err {
		case nil, sax.ErrHandlerUnspecified:
			// no op
		default:
			return ctx.error(err)
		}
	}

	/*
	 * Is there any internal subset declarations ?
	 * they are handled separately in parseInternalSubset()
	 */
	c := cur.Peek()
	if c == '[' {
		return nil
	}

	// Otherwise this should be the end of DTD
	if c != '>' {
		return ctx.error(ErrDocTypeNotFinished)
	}
	cur.Advance(1)

	return nil
}

func (ctx *parserCtx) parseInternalSubset() error {
	// equiv: xmlParseInternalSubset (parser.c)
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseInternalSubset")
		defer g.IRelease("END parseInternalSubset")
	}

	cur := ctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if cur.Peek() != '[' {
		goto FinishDTD
	}
	ctx.instate = psDTD
	cur.Advance(1)

	for {
		// Get current cursor in case parameter entity expansion changed the input
		cur = ctx.getCursor()
		if cur == nil || cur.Done() || cur.Peek() == ']' {
			break
		}
		
		ctx.skipBlanks()
		if err := ctx.parseMarkupDecl(); err != nil {
			return ctx.error(err)
		}
		if err := ctx.parsePEReference(); err != nil {
			return ctx.error(err)
		}
	}

	// Get final cursor state  
	cur = ctx.getCursor()
	if cur != nil && cur.Peek() == ']' {
		cur.Advance(1)
		ctx.skipBlanks()
	}

FinishDTD:
	// Ensure we have the current cursor
	cur = ctx.getCursor()
	if cur != nil && cur.Peek() != '>' {
		return ctx.error(ErrDocTypeNotFinished)
	}
	if cur != nil {
		cur.Advance(1)
	}

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
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseMarkupDecl")
		defer g.IRelease("END parseMarkupDecl")
	}

	cur := ctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if cur.Peek() == '<' {
		if cur.PeekN(2) == '!' {
			switch cur.PeekN(3) {
			case 'E':
				c := cur.PeekN(4)
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
		} else if cur.PeekN(2) == '?' {
			return ctx.parsePI()
		}
	}

	if ctx.instate == psEOF {
		return nil
	}

	// This is only for internal subset. On external entities,
	// the replacement is done before parsing stage
	if !ctx.external && ctx.inputTab.Len() == 1 {
		if err := ctx.parsePEReference(); err != nil {
			return ctx.error(err)
		}
	}
	/*
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
	if pdebug.Enabled {
		g := pdebug.Marker("parsePEReference")
		defer g.End()
	}

	cur := ctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if cur.Peek() != '%' {
		// This is not an error. just be done
		if pdebug.Enabled {
			pdebug.Printf("no parameter entities here, returning...")
		}
		return nil
	}
	cur.Advance(1)

	name, err := ctx.parseName()
	if err != nil {
		return ctx.error(err)
	}

	if cur.Peek() != ';' {
		return ctx.error(ErrSemicolonRequired)
	}
	cur.Advance(1)

	/*
		ctx.nbentities++ // number of entities parsed
	*/
	var entity sax.Entity
	if s := ctx.sax; s != nil {
		_ = ctx.fireSAXCallback(cbGetParameterEntity, &entity, name)
	}

	// XXX Why check here?
	if ctx.instate == psEOF {
		return nil
	}

	if entity == nil {
		/*
		 * [ WFC: Entity Declared ]
		 * In a document without any DTD, a document with only an
		 * internal DTD subset which contains no parameter entity
		 * references, or a document with "standalone='yes'", ...
		 * ... The declaration of a parameter entity must precede
		 * any reference to it...
		 */
		if ctx.standalone == StandaloneExplicitYes || (!ctx.hasExternalSubset && !ctx.hasPERefs) {
			return fmt.Errorf("parse error: PEReference: %%%s; not found", name)
		}
		/*
		 * [ VC: Entity Declared ]
		 * In a document with an external subset or external
		 * parameter entities with "standalone='no'", ...
		 * ... The declaration of a parameter entity must
		 * precede any reference to it...
		 */
		/*
		   xmlWarningMsg(ctxt, XML_WAR_UNDECLARED_ENTITY,
		                 "PEReference: %%%s; not found\n",
		                 name, NULL);
		*/
		ctx.valid = false
		if err := ctx.entityCheck(entity, 0, 0); err != nil {
			return ctx.error(err)
		}
	} else {
		/*
		 * Internal checking in case the entity quest barfed
		 */
		if etype := EntityType(entity.EntityType()); etype != InternalParameterEntity && etype != ExternalParameterEntity {
			/*
			   xmlWarningMsg(ctxt, XML_WAR_UNDECLARED_ENTITY,
			         "Internal: %%%s; is not a parameter entity\n",
			                 name, NULL);
			*/
			/*
			   } else if (ctxt->input->free != deallocblankswrapper) {
			           input = xmlNewBlanksWrapperInputStream(ctxt, entity);
			           if (xmlPushInput(ctxt, input) < 0)
			               return;
			*/
		} else {
			// Handle the parameter entity expansion
			// c.f. http://www.w3.org/TR/REC-xml#as-PE
			if pdebug.Enabled {
				pdebug.Printf("Expanding parameter entity '%s' with content: %s", name, string(entity.Content()))
			}
			
			// Decode character references and other entities in the parameter entity content
			decodedContent, err := ctx.decodeEntities(entity.Content(), SubstituteBoth)
			if err != nil {
				return fmt.Errorf("failed to decode parameter entity content: %v", err)
			}
			
			if pdebug.Enabled {
				pdebug.Printf("Decoded parameter entity content: %s", decodedContent)
			}
			
			// Push the decoded content as new input stream
			ctx.pushInput(strcursor.NewByteCursor(bytes.NewReader([]byte(decodedContent))))
			
			// Note: External parameter entities may need text declaration parsing
			// but for now we only handle internal parameter entities
		}
	}
	ctx.hasPERefs = true
	return nil
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
func (ctx *parserCtx) parseElementDecl() (ElementTypeVal, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseElementDecl")
		defer g.IRelease("END parseElementDecl")
	}

	cur := ctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if !cur.ConsumeString("<!ELEMENT") {
		return UndefinedElementType, ctx.error(ErrInvalidElementDecl)
	}

	if !isBlankCh(cur.Peek()) {
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

	if !isBlankCh(cur.Peek()) {
		return UndefinedElementType, ctx.error(ErrSpaceRequired)
	}
	ctx.skipBlanks()

	var etype ElementTypeVal
	var content *ElementContent
	if cur.ConsumeString("EMPTY") {
		etype = EmptyElementType
	} else if cur.ConsumeString("ANY") {
		etype = AnyElementType
	} else if cur.Peek() == '(' {
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

	if cur.Peek() != '>' {
		return UndefinedElementType, ctx.error(ErrGtRequired)
	}
	cur.Advance(1)

	/*
	           if (input != ctxt->input) {
	               xmlFatalErrMsg(ctxt, XML_ERR_ENTITY_BOUNDARY,
	   "Element declaration doesn't start and stop in the same entity\n");
	           }
	*/

	if s := ctx.sax; s != nil {
		switch err := s.ElementDecl(ctx.userData, name, int(etype), content); err {
		case nil, sax.ErrHandlerUnspecified:
			// no op
		default:
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

func (ctx *parserCtx) parseElementContentDecl() (*ElementContent, ElementTypeVal, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseElementContentDecl")
		defer g.IRelease("END parseElementContentDecl")
	}

	cur := ctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if cur.Peek() != '(' {
		return nil, UndefinedElementType, ctx.error(ErrOpenParenRequired)
	}
	cur.Advance(1)

	if ctx.instate == psEOF {
		return nil, UndefinedElementType, ctx.error(ErrEOF)
	}

	ctx.skipBlanks()

	var ec *ElementContent
	var err error
	var etype ElementTypeVal
	if cur.HasPrefixString("#PCDATA") {
		ec, err = ctx.parseElementMixedContentDecl()
		if err != nil {
			return nil, UndefinedElementType, ctx.error(err)
		}
		etype = MixedElementType
	} else {
		ec, err = ctx.parseElementChildrenContentDeclPriv(0)
		if err != nil {
			return nil, UndefinedElementType, ctx.error(err)
		}
		etype = ElementElementType
	}

	ctx.skipBlanks()
	return ec, etype, nil
}

func (ctx *parserCtx) parseElementMixedContentDecl() (*ElementContent, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseElementMixedContentDecl")
		defer g.IRelease("END parseElementMixedContentDecl")
	}

	cur := ctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if !cur.ConsumeString("#PCDATA") {
		return nil, ctx.error(ErrPCDATARequired)
	}

	if cur.Peek() == ')' {
		/*
		               if ((ctxt->validate) && (ctxt->input->id != inputchk)) {
		                   xmlValidityError(ctxt, XML_ERR_ENTITY_BOUNDARY,
		   "Element content declaration doesn't start and stop in the same entity\n",
		                                    NULL, NULL);
		               }
		*/
		cur.Advance(1)
		ret, err := ctx.doc.CreateElementContent("", ElementContentPCDATA)
		if err != nil {
			return nil, ctx.error(err)
		}

		if cur.Peek() == '*' {
			ret.coccur = ElementContentMult
			cur.Advance(1)
		}

		return ret, nil
	}

	var err error
	var retelem *ElementContent
	var curelem *ElementContent
	if c := cur.Peek(); c == '(' || c == '|' {
		retelem, err = ctx.doc.CreateElementContent("", ElementContentPCDATA)
		if err != nil {
			return nil, ctx.error(err)
		}
		curelem = retelem
	}

	var elem string
	for cur.Peek() == '|' {
		cur.Advance(1)
		if elem == "" {
			retelem, err = ctx.doc.CreateElementContent("", ElementContentOr)
			if err != nil {
				return nil, ctx.error(err)
			}

			retelem.c1 = curelem
			if curelem != nil {
				curelem.parent = retelem
			}
			curelem = retelem
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
			curelem.c2 = n
			n.parent = curelem
			curelem = n
		}
		ctx.skipBlanks()
		elem, err = ctx.parseName()
		if err != nil {
			return nil, ctx.error(err)
		}
		ctx.skipBlanks()
	}
	if cur.Peek() == ')' && cur.PeekN(2) == '*' {
		cur.Advance(2)
		if elem != "" {
			curelem.c2, err = ctx.doc.CreateElementContent(elem, ElementContentElement)
			if err != nil {
				return nil, ctx.error(err)
			}
			curelem.c2.parent = curelem
		}

		if retelem != nil {
			retelem.coccur = ElementContentMult
		}
		/*
		               if ((ctxt->validate) && (ctxt->input->id != inputchk)) {
		                   xmlValidityError(ctxt, XML_ERR_ENTITY_BOUNDARY,
		   "Element content declaration doesn't start and stop in the same entity\n",
		                                    NULL, NULL);
		   					}
		*/
	}
	return retelem, nil
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
func (ctx *parserCtx) parseElementChildrenContentDeclPriv(depth int) (*ElementContent, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseElementChildrenContentDeclPriv(%d)", depth)
		defer g.IRelease("END parseElementChildrenContentDeclPriv(%d)", depth)
	}

	if depth > 128 { // XML_PARSE_HUGE -> 2048
		return nil, fmt.Errorf("xmlParseElementChildrenContentDecl : depth %d too deep", depth)
	}

	var curelem *ElementContent
	var retelem *ElementContent
	ctx.skipBlanks()
	cur := ctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if cur.Peek() == '(' {
		cur.Advance(1)
		ctx.skipBlanks()
		retelem, err := ctx.parseElementChildrenContentDeclPriv(depth + 1)
		if err != nil {
			return nil, ctx.error(err)
		}
		curelem = retelem
		ctx.skipBlanks()
	} else {
		elem, err := ctx.parseName()
		if err != nil {
			return nil, ctx.error(err)
		}

		retelem, err = ctx.doc.CreateElementContent(elem, ElementContentElement)
		if err != nil {
			return nil, ctx.error(err)
		}
		curelem = retelem

		switch cur.Peek() {
		case '?':
			curelem.coccur = ElementContentOpt
			cur.Advance(1)
		case '*':
			curelem.coccur = ElementContentMult
			cur.Advance(1)
		case '+':
			curelem.coccur = ElementContentPlus
			cur.Advance(1)
		}
	}

	ctx.skipBlanks()

	// XXX closures aren't the most efficient thing golang has to offer,
	// but I really don't want to write the same code twice...
	var sep rune
	var last *ElementContent
	createElementContent := func(c rune, typ ElementContentType) error {
		// Detect "Name | Name, Name"
		if sep == 0x0 {
			sep = c
		} else if sep != c {
			return ctx.error(fmt.Errorf("'%c' expected", sep))
		}
		cur.Advance(1)

		op, err := ctx.doc.CreateElementContent("", typ)
		if err != nil {
			return ctx.error(err)
		}

		if last == nil {
			op.c1 = retelem
			if retelem != nil {
				retelem.parent = op
			}
			curelem = op
			retelem = op
		} else {
			curelem.c2 = op
			op.parent = curelem
			op.c1 = last
			if last != nil {
				last.parent = op
			}
			curelem = op
			last = nil
		}
		return nil
	}

LOOP:
	for !cur.Done() {
		c := cur.Peek()
		switch c {
		case ')': // end
			break LOOP // need label, or otherwise break only breaks from switch
		case ',':
			if err := createElementContent(c, ElementContentSeq); err != nil {
				return nil, ctx.error(err)
			}
		case '|':
			if err := createElementContent(c, ElementContentOr); err != nil {
				return nil, ctx.error(err)
			}
		default:
			return nil, ctx.error(ErrElementContentNotFinished)
		}

		ctx.skipBlanks()

		if cur.Peek() == '(' {
			cur.Advance(1)
			ctx.skipBlanks()
			// recurse
			var err error
			last, err = ctx.parseElementChildrenContentDeclPriv(depth + 1)
			if err != nil {
				return nil, ctx.error(err)
			}
			ctx.skipBlanks()
		} else {
			elem, err := ctx.parseName()
			if err != nil {
				return nil, ctx.error(err)
			}

			last, err = ctx.doc.CreateElementContent(elem, ElementContentElement)
			if err != nil {
				return nil, ctx.error(err)
			}

			switch cur.Peek() {
			case '?':
				last.coccur = ElementContentOpt
				cur.Advance(1)
			case '*':
				last.coccur = ElementContentMult
				cur.Advance(1)
			case '+':
				last.coccur = ElementContentPlus
				cur.Advance(1)
			}
		}
		ctx.skipBlanks()
	}
	if last != nil {
		curelem.c2 = last
		last.parent = curelem
	}
	cur.Advance(1)
	/*
	   	    if ((ctxt->validate) && (ctxt->input->id != inputchk)) {
	           xmlValidityError(ctxt, XML_ERR_ENTITY_BOUNDARY,
	   "Element content declaration doesn't start and stop in the same entity\n",
	                            NULL, NULL);
	       }
	*/

	c := cur.Peek()
	switch c {
	case '?':
		// XXX why would ret be null?
		if retelem != nil {
			if retelem.coccur == ElementContentPlus {
				retelem.coccur = ElementContentMult
			} else {
				retelem.coccur = ElementContentOpt
			}
		}
		cur.Advance(1)
	case '*':
		if retelem != nil {
			retelem.coccur = ElementContentMult
			curelem = retelem
			/*
			 * Some normalization:
			 * (a | b* | c?)* == (a | b | c)*
			 */
			for curelem != nil && curelem.ctype == ElementContentOr {
				if curelem.c1 != nil && (curelem.c1.coccur == ElementContentOpt || curelem.c1.coccur == ElementContentMult) {
					curelem.c1.coccur = ElementContentOnce
				}

				if curelem.c2 != nil && (curelem.c2.coccur == ElementContentOpt || curelem.c2.coccur == ElementContentMult) {
					curelem.c2.coccur = ElementContentOnce
				}
				curelem = curelem.c2
			}
		}
	case '+':
		if retelem.coccur == ElementContentOpt {
			retelem.coccur = ElementContentMult
		} else {
			retelem.coccur = ElementContentPlus
		}

		/*
		 * Some normalization:
		 * (a | b*)+ == (a | b)*
		 * (a | b?)+ == (a | b)*
		 */
		found := false
		for curelem != nil && curelem.ctype == ElementContentOr {
			if curelem.c1 != nil && (curelem.c1.coccur == ElementContentOpt || curelem.c1.coccur == ElementContentMult) {
				curelem.c1.coccur = ElementContentOnce
				found = true
			}

			if curelem.c2 != nil && (curelem.c2.coccur == ElementContentOpt || curelem.c2.coccur == ElementContentMult) {
				curelem.c2.coccur = ElementContentOnce
				found = true
			}
			curelem = curelem.c2
		}
		if found {
			retelem.coccur = ElementContentMult
		}
	}

	return retelem, nil
}

func (ctx *parserCtx) parseEntityValueInternal(qch rune) (string, error) {
	/*
	 * NOTE: 4.4.5 Included in Literal
	 * When a parameter entity reference appears in a literal entity
	 * value, ... a single or double quote character in the replacement
	 * text is always treated as a normal data character and will not
	 * terminate the literal.
	 * In practice it means we stop the loop only when back at parsing
	 * the initial entity and the quote is found
	 */
	cur := ctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	buf := bufferPool.Get().(*bytes.Buffer)
	defer releaseBuffer(buf)

	i := 0
	for c := cur.PeekN(i + 1); isChar(c) && c != qch; c = cur.PeekN(i + 1) {
		buf.WriteRune(c)
		i++
	}
	if i > 0 {
		cur.Advance(i)
		return buf.String(), nil
	}
	return "", nil
}

/*
 * Takes a entity string content and process to do the adequate substitutions.
 *
 * [67] Reference ::= EntityRef | CharRef
 *
 * [69] PEReference ::= '%' Name ';'
 *
 * Returns A newly allocated string with the substitution done.
 */
func (ctx *parserCtx) decodeEntities(s []byte, what SubstitutionType) (ret string, err error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START decodeEntitites (%s)", s)
		defer func() {
			g.IRelease("END decodeEntities ('%s' -> '%s')", s, ret)
		}()
	}
	ret, err = ctx.decodeEntitiesInternal(s, what, 0)
	return
}

func (ctx *parserCtx) decodeEntitiesInternal(s []byte, what SubstitutionType, depth int) (string, error) {
	if depth > 40 {
		return "", errors.New("entity loop (depth > 40)")
	}

	out := bufferPool.Get().(*bytes.Buffer)
	defer releaseBuffer(out)

	for len(s) > 0 {
		pdebug.Printf("s[0] -> %c", s[0])
		if bytes.HasPrefix(s, []byte{'&', '#'}) {
			val, width, err := parseStringCharRef(s)
			if err != nil {
				return "", err
			}
			out.WriteRune(val)
			s = s[width:] // advance
		} else if s[0] == '&' && what&SubstituteRef == SubstituteRef {
			ent, width, err := ctx.parseStringEntityRef(s)
			if err != nil {
				return "", err
			}
			if err := ctx.entityCheck(ent, 0, 0); err != nil {
				return "", err
			}

			if EntityType(ent.EntityType()) == InternalPredefinedEntity {
				if len(ent.Content()) == 0 {
					return "", errors.New("predefined entity has no content")
				}
				out.Write(ent.Content())
			} else if len(ent.Content()) != 0 {
				rep, err := ctx.decodeEntitiesInternal(ent.Content(), what, depth+1)
				if err != nil {
					return "", err
				}

				out.WriteString(rep)
			} else {
				out.WriteString(ent.Name())
			}
			s = s[width:]
		} else if s[0] == '%' && what&SubstitutePERef == SubstitutePERef {
			ent, width, err := ctx.parseStringPEReference(s)
			if err != nil {
				return "", err
			}
			if err := ctx.entityCheck(ent, width, 0); err != nil {
				return "", err
			}
			rep, err := ctx.decodeEntitiesInternal(ent.Content(), what, depth+1)
			if err != nil {
				return "", err
			}
			out.WriteString(rep)
			s = s[width:]
		} else {
			out.WriteByte(s[0])
			s = s[1:]
		}
	}
	return out.String(), nil
}

/*
 * parse a value for ENTITY declarations
 *
 * [9] EntityValue ::= '"' ([^%&"] | PEReference | Reference)* '"' |
 *                     "'" ([^%&'] | PEReference | Reference)* "'"
 *
 * Returns the EntityValue parsed with reference substituted or NULL
 */
func (ctx *parserCtx) parseEntityValue() (string, string, error) {
	if pdebug.Enabled {
		g := pdebug.Marker("parseEntityValue")
		defer g.End()
	}

	ctx.instate = psEntityValue

	literal, err := ctx.parseQuotedText(func(qch rune) (string, error) {
		return ctx.parseEntityValueInternal(qch)
	})

	val, err := ctx.decodeEntities([]byte(literal), SubstitutePERef)
	if err != nil {
		return "", "", ctx.error(err)
	}

	if pdebug.Enabled {
		pdebug.Printf("parsed entity value '%s'", val)
	}

	return literal, val, nil
}

/*
 * parse <!ENTITY declarations
 *
 * [70] EntityDecl ::= GEDecl | PEDecl
 *
 * [71] GEDecl ::= '<!ENTITY' S Name S EntityDef S? '>'
 *
 * [72] PEDecl ::= '<!ENTITY' S '%' S Name S PEDef S? '>'
 *
 * [73] EntityDef ::= EntityValue | (ExternalID NDataDecl?)
 *
 * [74] PEDef ::= EntityValue | ExternalID
 *
 * [76] NDataDecl ::= S 'NDATA' S Name
 *
 * [ VC: Notation Declared ]
 * The Name must match the declared name of a notation.
 */
func (ctx *parserCtx) parseEntityDecl() error {
	if pdebug.Enabled {
		g := pdebug.Marker("parseEntityDecl")
		defer g.End()
	}

	cur := ctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if !cur.ConsumeString("<!ENTITY") {
		return ctx.error(errors.New("<!ENTITY not started"))
	}

	if !ctx.skipBlanks() {
		return ctx.error(ErrSpaceRequired)
	}

	isParameter := false
	if cur.Peek() == '%' {
		cur.Advance(1)
		if !ctx.skipBlanks() {
			return ctx.error(ErrSpaceRequired)
		}
		isParameter = true
	}

	name, err := ctx.parseName()
	if err != nil {
		return ctx.error(err)
	}
	if strings.IndexByte(name, ':') > -1 {
		return ctx.error(errors.New("colons are forbidden from entity names"))
	}

	if !ctx.skipBlanks() {
		return ctx.error(ErrSpaceRequired)
	}

	ctx.instate = psEntityDecl
	var literal string
	var value string
	var uri string

	if isParameter {
		if pdebug.Enabled {
			pdebug.Printf("Found parameter entity")
		}

		if c := cur.Peek(); c == '"' || c == '\'' {
			if pdebug.Enabled {
				pdebug.Printf("parseEntityDecl, isParameter = true, calling parseEntityValue")
			}
			literal, value, err = ctx.parseEntityValue()
			if pdebug.Enabled {
				pdebug.Printf("entity declaration '%s' -> '%s'", name, value)
			}

			if err == nil {
				switch err := ctx.fireSAXCallback(cbEntityDecl, name, value); err {
				case nil, sax.ErrHandlerUnspecified:
					// no op
				default:
					return ctx.error(err)
				}
			}
		} else {
			if pdebug.Enabled {
				pdebug.Printf("Attempting to parse external ID")
			}
			literal, uri, err = ctx.parseExternalID()
			if err != nil {
				return ctx.error(ErrValueRequired)
			}

			if uri != "" {
				u, err := url.Parse(uri)
				if err != nil {
					return ctx.error(err)
				}

				if u.Fragment != "" {
					return ctx.error(errors.New("err uri fragment"))
				} else {
					if s := ctx.sax; s != nil {
						switch err := s.EntityDecl(ctx.userData, name, int(ExternalParameterEntity), literal, uri, ""); err {
						case nil, sax.ErrHandlerUnspecified:
							// no op
						default:
							return ctx.error(err)
						}
					}
				}
			}
		}
	} else {
		if pdebug.Enabled {
			pdebug.Printf("Found entity")
		}
		if c := cur.Peek(); c == '"' || c == '\'' {
			literal, value, err = ctx.parseEntityValue()
			if err == nil {
				if s := ctx.sax; s != nil {
					switch err := s.EntityDecl(ctx.userData, name, int(InternalGeneralEntity), "", "", value); err {
					case nil, sax.ErrHandlerUnspecified:
						// no op
					default:
						return ctx.error(err)
					}
				}
			}
		} else {
			literal, uri, err = ctx.parseExternalID()
			if err != nil {
				return ctx.error(ErrValueRequired)
			}

			if uri != "" {
				u, err := url.Parse(uri)
				if err != nil {
					return ctx.error(err)
				}

				if u.Fragment != "" {
					return ctx.error(errors.New("err uri fragment"))
				} else {
					if s := ctx.sax; s != nil {
						switch err := s.EntityDecl(ctx.userData, name, int(ExternalGeneralParsedEntity), literal, uri, ""); err {
						case nil, sax.ErrHandlerUnspecified:
							// no op
						default:
							return ctx.error(err)
						}
					}
				}
			}

			if c := cur.Peek(); c != '>' && !isBlankCh(c) {
				return ctx.error(ErrSpaceRequired)
			}

			ctx.skipBlanks()
			if cur.ConsumeString("NDATA") {
				if !ctx.skipBlanks() {
					return ctx.error(ErrSpaceRequired)
				}

				ndata, err := ctx.parseName()
				if err != nil {
					return ctx.error(err)
				}
				if s := ctx.sax; s != nil {
					switch err := s.EntityDecl(ctx.userData, name, int(ExternalParameterEntity), literal, uri, ndata); err {
					case nil, sax.ErrHandlerUnspecified:
						// no op
					default:
						return ctx.error(err)
					}
				}
			} else {
				if s := ctx.sax; s != nil {
					if pdebug.Enabled {
						pdebug.Printf("Calling s.EntityDecl with %s -> %s", name, literal)
					}
					switch err := s.EntityDecl(ctx.userData, name, int(ExternalParameterEntity), literal, uri, ""); err {
					case nil, sax.ErrHandlerUnspecified:
						// no op
					default:
						return ctx.error(err)
					}
				}
				/*
				    // For expat compatibility in SAX mode.
				    // assuming the entity repalcement was asked for
				   if ((ctxt->replaceEntities != 0) &&
				       ((ctxt->myDoc == NULL) ||
				       (xmlStrEqual(ctxt->myDoc->version, SAX_COMPAT_MODE)))) {
				       if (ctxt->myDoc == NULL) {
				           ctxt->myDoc = xmlNewDoc(SAX_COMPAT_MODE);
				           if (ctxt->myDoc == NULL) {
				               xmlErrMemory(ctxt, "New Doc failed");
				               return;
				           }
				           ctxt->myDoc->properties = XML_DOC_INTERNAL;
				       }

				       if (ctxt->myDoc->intSubset == NULL)
				           ctxt->myDoc->intSubset = xmlNewDtd(ctxt->myDoc,
				                               BAD_CAST "fake", NULL, NULL);
				       xmlSAX2EntityDecl(ctxt, name,
				                         XML_EXTERNAL_GENERAL_PARSED_ENTITY,
				                         literal, URI, NULL);
				   }
				*/
			}
		}
	}

	pdebug.Printf("============================")

	ctx.skipBlanks()
	if cur.Peek() != '>' {
		return ctx.error(errors.New("entity not terminated"))
	}
	cur.Advance(1)

	// Ugly mechanism to save the raw entity value.
	// Note: This happens because the SAX interface doesn't have a way to
	// pass this non-standard information to the handler
	var curent sax.Entity
	if isParameter {
		if s := ctx.sax; s != nil {
			curent, _ = s.GetParameterEntity(ctx.userData, name)
		}
	} else {
		if s := ctx.sax; s != nil {
			curent, _ = s.GetEntity(ctx.userData, name)
			/*
			   if ((cur == NULL) && (ctxt->userData==ctxt)) {
			       cur = xmlSAX2GetEntity(ctxt, name);
			   }
			*/
		}
	}
	if curent != nil {
		curent.SetOrig(literal)
	}

	return nil
}

/*
 * parse an Notation attribute type.
 *
 * Note: the leading 'NOTATION' S part has already being parsed...
 *
 * [58] NotationType ::= 'NOTATION' S '(' S? Name (S? '|' S? Name)* S? ')'
 *
 * [ VC: Notation Attributes ]
 * Values of this type must match one of the notation names included
 * in the declaration; all notation names in the declaration must be declared.
 *
 * Returns: the notation attribute tree built while parsing
 */
func (ctx *parserCtx) parseNotationType() (Enumeration, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseNotationType")
		defer g.IRelease("END parseNotationType")
	}

	cur := ctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if cur.Peek() != '(' {
		return nil, ctx.error(ErrNotationNotStarted)
	}
	cur.Advance(1)
	ctx.skipBlanks()

	names := map[string]struct{}{}

	var enum Enumeration
	for ctx.instate != psEOF {
		name, err := ctx.parseName()
		if err != nil {
			return nil, ctx.error(ErrNotationNameRequired)
		}
		if _, ok := names[name]; ok {
			return nil, ctx.error(ErrDTDDupToken{Name: name})
		}

		enum = append(enum, name)
		ctx.skipBlanks()

		if cur.Peek() != '|' {
			break
		}
		cur.Advance(1)
		ctx.skipBlanks()
	}

	if cur.Peek() != ')' {
		return nil, ctx.error(ErrNotationNotFinished)
	}
	cur.Advance(1)
	return enum, nil
}

func (ctx *parserCtx) parseEnumerationType() (Enumeration, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseEnumerationType")
		defer g.IRelease("END parseEnumerationType")
	}

	cur := ctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if cur.Peek() != '(' {
		return nil, ctx.error(ErrAttrListNotStarted)
	}
	cur.Advance(1)
	ctx.skipBlanks()

	names := map[string]struct{}{}

	var enum Enumeration
	for ctx.instate != psEOF {
		name, err := ctx.parseNmtoken()
		if err != nil {
			return nil, ctx.error(ErrNmtokenRequired)
		}
		if _, ok := names[name]; ok {
			return nil, ctx.error(ErrDTDDupToken{Name: name})
		}

		enum = append(enum, name)
		ctx.skipBlanks()

		if cur.Peek() != '|' {
			break
		}
		cur.Advance(1)
		ctx.skipBlanks()
	}

	if cur.Peek() != ')' {
		return nil, ctx.error(ErrAttrListNotFinished)
	}
	cur.Advance(1)
	return enum, nil
}

/*
 * parse an Enumerated attribute type.
 *
 * [57] EnumeratedType ::= NotationType | Enumeration
 *
 * [58] NotationType ::= 'NOTATION' S '(' S? Name (S? '|' S? Name)* S? ')'
 *
 *
 * Returns: XML_ATTRIBUTE_ENUMERATION or XML_ATTRIBUTE_NOTATION
 */
func (ctx *parserCtx) parseEnumeratedType() (AttributeType, Enumeration, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseEnumeratedType")
		defer g.IRelease("END parseEnumeratedType")
	}

	cur := ctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if cur.ConsumeString("NOTATION") {
		if !isBlankCh(cur.Peek()) {
			return AttrInvalid, nil, ctx.error(ErrSpaceRequired)
		}
		ctx.skipBlanks()
		enum, err := ctx.parseNotationType()
		if err != nil {
			return AttrInvalid, nil, ctx.error(err)
		}

		return AttrNotation, enum, nil
	}

	enum, err := ctx.parseEnumerationType()
	if err != nil {
		return AttrInvalid, enum, ctx.error(err)
	}
	return AttrEnumeration, enum, nil
}

/*
 * parse the Attribute list def for an element
 *
 * [54] AttType ::= StringType | TokenizedType | EnumeratedType
 *
 * [55] StringType ::= 'CDATA'
 *
 * [56] TokenizedType ::= 'ID' | 'IDREF' | 'IDREFS' | 'ENTITY' |
 *                        'ENTITIES' | 'NMTOKEN' | 'NMTOKENS'
 *
 * Validity constraints for attribute values syntax are checked in
 * xmlValidateAttributeValue()
 *
 * [ VC: ID ]
 * Values of type ID must match the Name production. A name must not
 * appear more than once in an XML document as a value of this type;
 * i.e., ID values must uniquely identify the elements which bear them.
 *
 * [ VC: One ID per Element Type ]
 * No element type may have more than one ID attribute specified.
 *
 * [ VC: ID Attribute Default ]
 * An ID attribute must have a declared default of #IMPLIED or #REQUIRED.
 *
 * [ VC: IDREF ]
 * Values of type IDREF must match the Name production, and values
 * of type IDREFS must match Names; each IDREF Name must match the value
 * of an ID attribute on some element in the XML document; i.e. IDREF
 * values must match the value of some ID attribute.
 *
 * [ VC: Entity Name ]
 * Values of type ENTITY must match the Name production, values
 * of type ENTITIES must match Names; each Entity Name must match the
 * name of an unparsed entity declared in the DTD.
 *
 * [ VC: Name Token ]
 * Values of type NMTOKEN must match the Nmtoken production; values
 * of type NMTOKENS must match Nmtokens.
 *
 * Returns the attribute type
 */
func (ctx *parserCtx) parseAttributeType() (AttributeType, Enumeration, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseAttributeType")
		defer g.IRelease("END parseAttributeType")
	}

	cur := ctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if cur.ConsumeString("CDATA") {
		return AttrCDATA, nil, nil
	}
	if cur.ConsumeString("IDREFS") {
		return AttrIDRefs, nil, nil
	}
	if cur.ConsumeString("IDREF") {
		return AttrIDRef, nil, nil
	}
	if cur.ConsumeString("ID") {
		return AttrID, nil, nil
	}
	if cur.ConsumeString("ENTITY") {
		return AttrEntity, nil, nil
	}
	if cur.ConsumeString("ENTITIES") {
		return AttrEntities, nil, nil
	}
	if cur.ConsumeString("NMTOKENS") {
		return AttrNmtokens, nil, nil
	}
	if cur.ConsumeString("NMTOKEN") {
		return AttrNmtoken, nil, nil
	}

	return ctx.parseEnumeratedType()
}

/*
 * Parse an attribute default declaration
 *
 * [60] DefaultDecl ::= '#REQUIRED' | '#IMPLIED' | (('#FIXED' S)? AttValue)
 *
 * [ VC: Required Attribute ]
 * if the default declaration is the keyword #REQUIRED, then the
 * attribute must be specified for all elements of the type in the
 * attribute-list declaration.
 *
 * [ VC: Attribute Default Legal ]
 * The declared default value must meet the lexical constraints of
 * the declared attribute type c.f. xmlValidateAttributeDecl()
 *
 * [ VC: Fixed Attribute Default ]
 * if an attribute has a default value declared with the #FIXED
 * keyword, instances of that attribute must match the default value.
 *
 * [ WFC: No < in Attribute Values ]
 * handled in xmlParseAttValue()
 *
 * returns: XML_ATTRIBUTE_NONE, XML_ATTRIBUTE_REQUIRED, XML_ATTRIBUTE_IMPLIED
 *          or XML_ATTRIBUTE_FIXED.
 */
func (ctx *parserCtx) parseDefaultDecl() (deftype AttributeDefault, defvalue string, err error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseDefaultDecl")
		defer func() {
			g.IRelease("END parseDefaultDecl (deftype = %d, defvalue = '%s')", deftype, defvalue)
		}()
	}

	deftype = AttrDefaultNone
	cur := ctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if cur.ConsumeString("#REQUIRED") {
		deftype = AttrDefaultRequired
		return
	}
	if cur.ConsumeString("#IMPLIED") {
		deftype = AttrDefaultImplied
		return
	}

	if cur.ConsumeString("#FIXED") {
		deftype = AttrDefaultFixed
		if !isBlankCh(cur.Peek()) {
			deftype = AttrDefaultInvalid
			err = ctx.error(ErrSpaceRequired)
			return
		}
		ctx.skipBlanks()
	}

	// XXX does AttValue always have a quote around it?
	defvalue, err = ctx.parseQuotedText(func(qch rune) (string, error) {
		s, _, err := ctx.parseAttributeValueInternal(qch, false)
		return s, err
	})
	if err != nil {
		deftype = AttrDefaultInvalid
		err = ctx.error(err)
		return
	}
	ctx.instate = psDTD
	err = nil
	return
}

/*
 * Normalize the space in non CDATA attribute values:
 * If the attribute type is not CDATA, then the XML processor MUST further
 * process the normalized attribute value by discarding any leading and
 * trailing space (#x20) characters, and by replacing sequences of space
 * (#x20) characters by a single space (#x20) character.
 * Note that the size of dst need to be at least src, and if one doesn't need
 * to preserve dst (and it doesn't come from a dictionary or read-only) then
 * passing src as dst is just fine.
 *
 * Returns a pointer to the normalized value (dst) or NULL if no conversion
 *         is needed.
 */
func (ctx *parserCtx) attrNormalizeSpace(s string) (value string) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START attrNormalizeSpace")
		defer g.IRelease("END attrNormalizeSpace")
		defer func() {
			if s == value {
				pdebug.Printf("no change")
			} else {
				pdebug.Printf("normalized '%s' => '%s'", s, value)
			}
		}()
	}

	// don't bother if we have zero length
	if len(s) == 0 {
		value = s
		return
	}

	// skip leading spaces
	i := 0
	for ; i < len(s); i++ {
		if s[i] != 0x20 {
			break
		}
	}

	// make b
	out := make([]byte, 0, len(s))

	for i < len(s) {
		// not a space, no problem. just append
		if s[i] != 0x20 {
			out = append(out, s[i])
			i++
			continue
		}

		// skip dupes.
		for i < len(s) && s[i] == 0x20 {
			i++
		}
		out = append(out, 0x20) // append a single space
	}

	if out[len(out)-1] == 0x20 {
		out = out[:len(out)-1]
	}
	value = string(out)
	return
}

/* Trim the list of attributes defined to remove all those of type
 * CDATA as they are not special. This call should be done when finishing
 * to parse the DTD and before starting to parse the document root.
 */
func (ctx *parserCtx) cleanSpecialAttributes() {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START cleanSpecialAttribute")
		defer g.IRelease("END cleanSpecialAttribute")
	}
	for k, v := range ctx.attsSpecial {
		if v == AttrCDATA {
			if pdebug.Enabled {
				pdebug.Printf("removing %s from special attribute set", k)
			}
			delete(ctx.attsSpecial, k)
		}
	}
}

func (ctx *parserCtx) addSpecialAttribute(elemName, attrName string, typ AttributeType) {
	key := elemName + ":" + attrName
	if pdebug.Enabled {
		g := pdebug.IPrintf("START addSpecialAttribute(%s, %d)", key, typ)
		defer g.IRelease("END addSpecialAttribute")
	}
	ctx.attsSpecial[key] = typ
}

func (ctx *parserCtx) lookupSpecialAttribute(elemName, attrName string) (AttributeType, bool) {
	key := elemName + ":" + attrName
	if pdebug.Enabled {
		g := pdebug.IPrintf("START lookupSpecialAttribute(%s)", key)
		defer g.IRelease("END lookupSpecialAttribute")
	}
	v, ok := ctx.attsSpecial[key]
	return v, ok
}

func validateAttributeValueInternal(doc *Document, typ AttributeType, defvalue string) error {
	return nil
}

func (ctx *parserCtx) addAttributeDecl(dtd *DTD, elem string, name string, prefix string, atype AttributeType, def AttributeDefault, defvalue string, tree Enumeration) (attr *AttributeDecl, err error) {
	if dtd == nil {
		err = errors.New("dtd required")
		return
	}
	if name == "" {
		err = errors.New("name required")
	}
	if elem == "" {
		err = errors.New("element required")
	}

	switch atype {
	case AttrCDATA, AttrID, AttrIDRef, AttrIDRefs, AttrEntity, AttrEntities, AttrNmtoken, AttrNmtokens, AttrEnumeration, AttrNotation:
		// ok. no op
	default:
		err = errors.New("invalid attribute type")
		return
	}

	if defvalue != "" {
		if err = validateAttributeValueInternal(dtd.doc, atype, defvalue); err != nil {
			err = fmt.Errorf("attribute %s of %s: invalid default value: %s", elem, name, err)
			ctx.valid = false
			return
		}
	}

	// Check first that an attribute defined in the external subset wasn't
	// already defined in the internal subset
	if doc := dtd.doc; doc != nil && doc.extSubset == dtd && doc.intSubset != nil && len(doc.intSubset.attributes) == 0 {
		if _, ok := dtd.LookupAttribute(name, prefix, elem); !ok {
			err = fmt.Errorf("attribute %s of %s: already defined in internal subset", elem, name)
			return
		}
	}

	attr = newAttributeDecl()
	attr.atype = atype
	attr.doc = dtd.doc
	attr.name = name
	attr.prefix = prefix
	attr.elem = elem
	attr.def = def
	attr.tree = tree
	attr.defvalue = defvalue

	// Validity Check: Search the DTD for previous declarations of the ATTLIST
	// (RegisterAttribute should return error if this attr already exists)
	if err = dtd.RegisterAttribute(attr); err != nil {
		attr = nil
		return
	}

	/*
	       // Validity Check:
	       // Multiple ID per element
	       //
	       elemDef = xmlGetDtdElementDesc2(dtd, elem, 1);
	       if (elemDef != NULL) {

	   // #ifdef LIBXML_VALID_ENABLED
	           if ((type == XML_ATTRIBUTE_ID) &&
	               (xmlScanIDAttributeDecl(NULL, elemDef, 1) != 0)) {
	               xmlErrValidNode(ctxt, (xmlNodePtr) dtd, XML_DTD_MULTIPLE_ID,
	              "Element %s has too may ID attributes defined : %s\n",
	                      elem, name, NULL);
	               if (ctxt != NULL)
	                   ctxt->valid = 0;
	           }
	   // #endif LIBXML_VALID_ENABLED

	           // Insert namespace default def first they need to be
	           // processed first.
	           //
	           if ((xmlStrEqual(ret->name, BAD_CAST "xmlns")) ||
	               ((ret->prefix != NULL &&
	                (xmlStrEqual(ret->prefix, BAD_CAST "xmlns"))))) {
	               ret->nexth = elemDef->attributes;
	               elemDef->attributes = ret;
	           } else {
	               xmlAttributePtr tmp = elemDef->attributes;

	               while ((tmp != NULL) &&
	                      ((xmlStrEqual(tmp->name, BAD_CAST "xmlns")) ||
	                       ((ret->prefix != NULL &&
	                        (xmlStrEqual(ret->prefix, BAD_CAST "xmlns")))))) {
	                   if (tmp->nexth == NULL)
	                       break;
	                   tmp = tmp->nexth;
	               }
	               if (tmp != NULL) {
	                   ret->nexth = tmp->nexth;
	                   tmp->nexth = ret;
	               } else {
	                   ret->nexth = elemDef->attributes;
	                   elemDef->attributes = ret;
	               }
	           }
	       }
	*/

	dtd.AddChild(attr)
	return attr, nil
}

func (ctx *parserCtx) addAttributeDefault(elemName, attrName, defaultValue string) {
	// detect attribute redefinition
	if _, ok := ctx.lookupSpecialAttribute(elemName, attrName); ok {
		return
	}

	// XXX seems like when your language has a map, you can do just
	// kinda do away with a bunch of stuff..  See xmlAddDefAttrs for
	// details of what the original code is doing
	m, ok := ctx.attsDefault[elemName]
	if !ok {
		m = map[string]*Attribute{}
		ctx.attsDefault[elemName] = m
	}

	var prefix string
	var local string
	if i := strings.IndexByte(attrName, ':'); i > -1 {
		prefix = attrName[:i]
		local = attrName[i+1:]
	} else {
		local = attrName
	}

	uri := ctx.lookupNamespace(prefix)
	attr, err := ctx.doc.CreateAttribute(local, defaultValue, newNamespace(prefix, uri))
	if err != nil {
		// XXX Unhandled?!
		return
	}

	attr.SetDefault(true)
	m[attrName] = attr

	/*
	   	hmm, let's think about this when the time comes
	       if (ctxt->external)
	           defaults->values[5 * defaults->nbAttrs + 4] = BAD_CAST "external";
	       else
	           defaults->values[5 * defaults->nbAttrs + 4] = NULL;
	*/
}

func (ctx *parserCtx) lookupAttributeDefault(elemName string) (map[string]*Attribute, bool) {
	v, ok := ctx.attsDefault[elemName]
	return v, ok
}

/*
 * : parse the Attribute list def for an element
 *
 * [52] AttlistDecl ::= '<!ATTLIST' S Name AttDef* S? '>'
 *
 * [53] AttDef ::= S Name S AttType S DefaultDecl
 */
func (ctx *parserCtx) parseAttributeListDecl() error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseAttributeListDecl")
		defer g.IRelease("END parseAttributeListDecl")
	}

	cur := ctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if !cur.ConsumeString("<!ATTLIST") {
		return nil
	}

	if !isBlankCh(cur.Peek()) {
		return ctx.error(ErrSpaceRequired)
	}
	ctx.skipBlanks()

	elemName, err := ctx.parseName()
	if err != nil {
		return ctx.error(err)
	}
	ctx.skipBlanks()

	for cur.Peek() != '>' && ctx.instate != psEOF {
		attrName, err := ctx.parseName()
		if err != nil {
			return ctx.error(ErrAttributeNameRequired)
		}
		if !isBlankCh(cur.Peek()) {
			return ctx.error(ErrSpaceRequired)
		}
		ctx.skipBlanks()

		typ, enum, err := ctx.parseAttributeType()
		if err != nil {
			return ctx.error(err)
		}

		if !isBlankCh(cur.Peek()) {
			return ctx.error(ErrSpaceRequired)
		}
		ctx.skipBlanks()

		def, defvalue, err := ctx.parseDefaultDecl()
		if err != nil {
			return ctx.error(err)
		}

		if typ != AttrCDATA && def != AttrDefaultInvalid {
			defvalue = ctx.attrNormalizeSpace(defvalue)
		}

		if c := cur.Peek(); c != '>' {
			if !isBlankCh(c) {
				return ctx.error(ErrSpaceRequired)
			}
			ctx.skipBlanks()
		}
		/*
		   if (check == CUR_PTR) {
		       xmlFatalErr(ctxt, XML_ERR_INTERNAL_ERROR,
		                   "in xmlParseAttributeListDecl\n");
		       if (defaultValue != NULL)
		           xmlFree(defaultValue);
		       if (tree != NULL)
		           xmlFreeEnumeration(tree);
		       break;
		   }
		*/
		if s := ctx.sax; s != nil {
			switch err := s.AttributeDecl(ctx.userData, elemName, attrName, int(typ), int(def), defvalue, enum); err {
			case nil, sax.ErrHandlerUnspecified:
				// no op
			default:
				return ctx.error(err)
			}
		}

		if defvalue != "" && def != AttrDefaultImplied && def != AttrDefaultRequired {
			ctx.addAttributeDefault(elemName, attrName, defvalue)
		}

		// note: in libxml2, this is only triggered when SAX2 is enabled.
		// as we only support SAX2, we just register it regardless
		ctx.addSpecialAttribute(elemName, attrName, typ)

		if cur.Peek() == '>' {
			/*
			           if (input != ctxt->input) {
			               xmlValidityError(ctxt, XML_ERR_ENTITY_BOUNDARY,
			   "Attribute list declaration doesn't start and stop in the same entity\n",
			                                NULL, NULL);
			           }
			*/
			cur.Advance(1)
			break
		}
	}
	return nil
}

func (ctx *parserCtx) parseNotationDecl() error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseNotationDecl")
		defer g.IRelease("END parseNotationDecl")
	}
	return nil
}

func (ctx *parserCtx) parseExternalID() (string, string, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseExternalID")
		defer g.IRelease("END parseExternalID")
	}
	return "", "", nil
}

func (ctx *parserCtx) parseEpilogue() error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseEpilogue")
		defer g.IRelease("END parseEpilogue")
	}

	return nil
}

func (ctx *parserCtx) parseExternalEntityPrivate(uri, externalID string) (Node, error) {
	return nil, errors.New("unimplemented")
}

var ErrParseSucceeded = errors.New("parse succeeded")

func (ctx *parserCtx) parseBalancedChunkInternal(chunk []byte, userData interface{}) (Node, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseBalancedChunkInternal")
		defer g.IRelease("END parseBalancedChunkInternal")
	}

	ctx.depth++
	defer func() { ctx.depth-- }()

	if ctx.depth > 40 {
		return nil, errors.New("entity loop")
	}

	newctx := &parserCtx{}
	newctx.init(nil, bytes.NewReader(chunk))
	defer newctx.release()

	if userData != nil {
		newctx.userData = userData
	} else {
		newctx.userData = newctx
	}

	if ctx.doc == nil {
		ctx.doc = NewDocument("1.0", "", StandaloneExplicitNo)
	}

	// save the document's children
	fc := ctx.doc.FirstChild()
	lc := ctx.doc.LastChild()
	ctx.doc.setFirstChild(nil)
	ctx.doc.setLastChild(nil)
	defer func() {
		ctx.doc.setFirstChild(fc)
		ctx.doc.setLastChild(lc)
	}()
	newctx.doc = ctx.doc
	newctx.sax = ctx.sax
	newctx.attsDefault = ctx.attsDefault
	newctx.depth = ctx.depth + 1

	// create a dummy node
	newRoot, err := newctx.doc.CreateElement("pseudoroot")
	if err != nil {
		return nil, ctx.error(err)
	}
	newctx.pushNode(newRoot)
	newctx.doc.AddChild(newRoot)
	newctx.switchEncoding()
	if err := newctx.parseContent(); err != nil {
		return nil, err
	}

	if child := newctx.doc.FirstChild(); child != nil {
		if grandchild := child.FirstChild(); grandchild != nil {
			for e := grandchild; e != nil; e = e.NextSibling() {
				e.SetTreeDoc(ctx.doc)
				e.SetParent(nil)
			}
			return grandchild, nil
		}
	}

	// this means that the parsing was successful, but there weren't
	// any nodes generated as a result of parsing
	return nil, ErrParseSucceeded
}

/*
 * parse and handle entity references in content, depending on the SAX
 * interface, this may end-up in a call to character() if this is a
 * CharRef, a predefined entity, if there is no reference() callback.
 * or if the parser was asked to switch to that mode.
 *
 * [67] Reference ::= EntityRef | CharRef
 */
func (ctx *parserCtx) parseReference() error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseReference")
		defer g.IRelease("END parseReference")
	}

	cur := ctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if cur.Peek() != '&' {
		return ctx.error(ErrAmpersandRequired)
	}

	// "&#..." CharRef
	if cur.PeekN(2) == '#' {
		v, err := ctx.parseCharRef()
		if err != nil {
			return ctx.error(err)
		}
		l := utf8.RuneLen(v)
		b := make([]byte, l)
		utf8.EncodeRune(b, v)
		if s := ctx.sax; s != nil {
			switch err := s.Characters(ctx.userData, b); err {
			case nil, sax.ErrHandlerUnspecified:
				// no op
			default:
				return ctx.error(err)
			}
		}
		return nil
	}

	// &...
	ent, err := ctx.parseEntityRef()
	if err != nil {
		return ctx.error(err)
	}
	// if !ctx.wellFormed { return } ??

	wasChecked := ent.checked

	// special case for predefined entities
	if ent.name == "" || EntityType(ent.EntityType()) == InternalPredefinedEntity {
		if ent.content == "" {
			return nil
		}
		if s := ctx.sax; s != nil {
			switch err := s.Characters(ctx.userData, []byte(ent.content)); err {
			case nil, sax.ErrHandlerUnspecified:
				// no op
			default:
				return ctx.error(err)
			}
		}
		return nil
	}

	// The first reference to the entity trigger a parsing phase
	// where the ent->children is filled with the result from
	// the parsing.
	// Note: external parsed entities will not be loaded, it is not
	// required for a non-validating parser, unless the parsing option
	// of validating, or substituting entities were given. Doing so is
	// far more secure as the parser will only process data coming from
	// the document entity by default.
	var parsedEnt Node
	if (wasChecked == 0 || (ent.firstChild == nil && ctx.options.IsSet(ParseNoEnt))) && (EntityType(ent.EntityType()) != ExternalGeneralParsedEntity || ctx.options.IsSet(ParseNoEnt|ParseDTDValid)) {
		var userData interface{}
		if ctx.userData != ctx {
			userData = ctx.userData
		}

		if EntityType(ent.EntityType()) == InternalGeneralEntity {
			parsedEnt, err = ctx.parseBalancedChunkInternal([]byte(ent.Content()), userData)
			switch err {
			case nil, ErrParseSucceeded:
				// may not have generated nodes, but parse was successful
			default:
				return err
			}
		} else if EntityType(ent.EntityType()) == ExternalGeneralParsedEntity {
			parsedEnt, err = ctx.parseExternalEntityPrivate(ent.uri, ent.externalID)
			switch err {
			case nil, ErrParseSucceeded:
				// may not have generated nodes, but parse was successful
			default:
				return err
			}
		} else {
			return errors.New("invalid entity type")
		}

		/*
		           // Store the number of entities needing parsing for this entity
		           // content and do checkings
		           ent->checked = (ctxt->nbentities - oldnbent + 1) * 2;
		           if ((ent->content != NULL) && (xmlStrchr(ent->content, '<')))
		               ent->checked |= 1;
		           if (ret == XML_ERR_ENTITY_LOOP) {
		               xmlFatalErr(ctxt, XML_ERR_ENTITY_LOOP, NULL);
		               xmlFreeNodeList(list);
		               return;
		           }
		           if (xmlParserEntityCheck(ctxt, 0, ent, 0)) {
		               xmlFreeNodeList(list);
		               return;
		           }

		           if ((ret == XML_ERR_OK) && (list != NULL)) {
		               if (((ent->etype == XML_INTERNAL_GENERAL_ENTITY) ||
		                (ent->etype == XML_EXTERNAL_GENERAL_PARSED_ENTITY))&&
		                   (ent->children == NULL)) {
		                   ent->children = list;
		                   if (ctxt->replaceEntities) {
		                       // Prune it directly in the generated document
		                       // except for single text nodes.
		                       if (((list->type == XML_TEXT_NODE) &&
		                            (list->next == NULL)) ||
		                           (ctxt->parseMode == XML_PARSE_READER)) {
		                           list->parent = (xmlNodePtr) ent;
		                           list = NULL;
		                           ent->owner = 1;
		                       } else {
		                           ent->owner = 0;
		                           while (list != NULL) {
		                               list->parent = (xmlNodePtr) ctxt->node;
		                               list->doc = ctxt->myDoc;
		                               if (list->next == NULL)
		                                   ent->last = list;
		                               list = list->next;
		                           }
		                           list = ent->children;
		   #ifdef LIBXML_LEGACY_ENABLED
		                           if (ent->etype == XML_EXTERNAL_GENERAL_PARSED_ENTITY)
		                             xmlAddEntityReference(ent, list, NULL);
		   #endif
		                       }
		                   } else {
		                       ent->owner = 1;
		                       while (list != NULL) {
		                           list->parent = (xmlNodePtr) ent;
		                           xmlSetTreeDoc(list, ent->doc);
		                           if (list->next == NULL)
		                               ent->last = list;
		                           list = list->next;
		                       }
		                   }
		               } else {
		                   xmlFreeNodeList(list);
		                   list = NULL;
		               }
		           } else if ((ret != XML_ERR_OK) &&
		                      (ret != XML_WAR_UNDECLARED_ENTITY)) {
		               xmlFatalErrMsgStr(ctxt, XML_ERR_UNDECLARED_ENTITY,
		                        "Entity '%s' failed to parse\n", ent->name);
		               xmlParserEntityCheck(ctxt, 0, ent, 0);
		           } else if (list != NULL) {
		               xmlFreeNodeList(list);
		               list = NULL;
		           }
		           if (ent->checked == 0)
		               ent->checked = 2;
		       } else if (ent->checked != 1) {
		           ctxt->nbentities += ent->checked / 2;
		       }
		*/

		// Now that the entity content has been gathered
		// provide it to the application, this can take different forms based
		// on the parsing modes.
		if ent.firstChild == nil {
			// Probably running in SAX mode and the callbacks don't
			// build the entity content. So unless we already went
			// though parsing for first checking go though the entity
			// content to generate callbacks associated to the entity
			if wasChecked != 0 {
				var userData interface{}
				if ctx.userData != ctx {
					userData = ctx.userData
				}
				if EntityType(ent.EntityType()) == InternalGeneralEntity {
					parsedEnt, err = ctx.parseBalancedChunkInternal([]byte(ent.Content()), userData)
					switch err {
					case nil, ErrParseSucceeded:
						// may not have generated nodes, but parse was successful
					default:
						return err
					}
				} else if EntityType(ent.EntityType()) == ExternalGeneralParsedEntity {
					parsedEnt, err = ctx.parseExternalEntityPrivate(ent.URI(), ent.externalID)
					switch err {
					case nil, ErrParseSucceeded:
						// may not have generated nodes, but parse was successful
					default:
						return err
					}
				} else {
					return errors.New("invalid entity type")
				}
			}
			if s := ctx.sax; s != nil && !ctx.replaceEntities {
				// Entity reference callback comes second, it's somewhat
				// superfluous but a compatibility to historical behaviour
				switch err := s.Reference(ctx.userData, ent.name); err {
				case nil, sax.ErrHandlerUnspecified:
					// no op
				default:
					return err
				}
			}
			return nil
		}

		// If we didn't get any children for the entity being built
		if s := ctx.sax; s != nil && !ctx.replaceEntities {
			// Create a node.
			switch err := s.Reference(ctx.userData, ent.name); err {
			case nil, sax.ErrHandlerUnspecified:
				// no op
			default:
				return err
			}
			return nil
		}
		_ = parsedEnt

		/*
			if ctx.replaceEntities || ent.firstChild == nil {
			           // There is a problem on the handling of _private for entities
			           // (bug 155816): Should we copy the content of the field from
			           // the entity (possibly overwriting some value set by the user
			           // when a copy is created), should we leave it alone, or should
			           // we try to take care of different situations?  The problem
			           // is exacerbated by the usage of this field by the xmlReader.
			           // To fix this bug, we look at _private on the created node
			           // and, if it's NULL, we copy in whatever was in the entity.
			           // If it's not NULL we leave it alone.  This is somewhat of a
			           // hack - maybe we should have further tests to determine
			           // what to do.
				if ctx.peekNode() == nil && ent.firstChild == nil {
			               // Seems we are generating the DOM content, do
			               // a simple tree copy for all references except the first
			               // In the first occurrence list contains the replacement.
					if (parsedEnt == nil && ent.owner == nil) || ctx.parseMode == ParseReaderMode {



			               if (((list == NULL) && (ent->owner == 0)) ||
			                   (ctxt->parseMode == XML_PARSE_READER)) {
			                   xmlNodePtr nw = NULL, cur, firstChild = NULL;
			                   // We are copying here, make sure there is no abuse
			                   ctxt->sizeentcopy += ent->length + 5;
			                   if (xmlParserEntityCheck(ctxt, 0, ent, ctxt->sizeentcopy))
			                       return;

			                   // when operating on a reader, the entities definitions
			                   // are always owning the entities subtree.
			                   // if (ctxt->parseMode == XML_PARSE_READER)
			                   //    ent->owner = 1;
			                   cur = ent->children;
			                   while (cur != NULL) {
			                       nw = xmlDocCopyNode(cur, ctxt->myDoc, 1);
			                       if (nw != NULL) {
			                           if (nw->_private == NULL)
			                               nw->_private = cur->_private;
			                           if (firstChild == NULL){
			                               firstChild = nw;
			                           }
			                           nw = xmlAddChild(ctxt->node, nw);
			                       }
			                       if (cur == ent->last) {
			                           // needed to detect some strange empty
			                           // node cases in the reader tests
			                           if ((ctxt->parseMode == XML_PARSE_READER) &&
			                               (nw != NULL) &&
			                               (nw->type == XML_ELEMENT_NODE) &&
			                               (nw->children == NULL))
			                               nw->extra = 1;

			                           break;
			                       }
			                       cur = cur->next;
			                   }
			               } else if ((list == NULL) || (ctxt->inputNr > 0)) {
			                   xmlNodePtr nw = NULL, cur, next, last,
			                              firstChild = NULL;

			                   // We are copying here, make sure there is no abuse
			                   ctxt->sizeentcopy += ent->length + 5;
			                   if (xmlParserEntityCheck(ctxt, 0, ent, ctxt->sizeentcopy))
			                       return;

			                   // Copy the entity child list and make it the new
			                   // entity child list. The goal is to make sure any
			                   // ID or REF referenced will be the one from the
			                   // document content and not the entity copy.
			                   cur = ent->children;
			                   ent->children = NULL;
			                   last = ent->last;
			                   ent->last = NULL;
			                   while (cur != NULL) {
			                       next = cur->next;
			                       cur->next = NULL;
			                       cur->parent = NULL;
			                       nw = xmlDocCopyNode(cur, ctxt->myDoc, 1);
			                       if (nw != NULL) {
			                           if (nw->_private == NULL)
			                               nw->_private = cur->_private;
			                           if (firstChild == NULL){
			                               firstChild = cur;
			                           }
			                           xmlAddChild((xmlNodePtr) ent, nw);
			                           xmlAddChild(ctxt->node, cur);
			                       }
			                       if (cur == last)
			                           break;
			                       cur = next;
			                   }
			                   if (ent->owner == 0)
			                       ent->owner = 1;
			               } else {
			                   const xmlChar *nbktext;
			                   // the name change is to avoid coalescing of the
			                   // node with a possible previous text one which
			                   // would make ent->children a dangling pointer
			                   nbktext = xmlDictLookup(ctxt->dict, BAD_CAST "nbktext",
			                                           -1);
			                   if (ent->children->type == XML_TEXT_NODE)
			                       ent->children->name = nbktext;
			                   if ((ent->last != ent->children) &&
			                       (ent->last->type == XML_TEXT_NODE))
			                       ent->last->name = nbktext;
			                   xmlAddChildList(ctxt->node, ent->children);
			               }

			               // This is to avoid a nasty side effect, see
			               // characters() in SAX.c
			               ctxt->nodemem = 0;
			               ctxt->nodelen = 0;
			               return;
			           }
			       }
		*/
	}

	return ErrUnimplemented{target: "parseReference"}
}

func accumulateDecimalCharRef(val int32, c rune) (int32, error) {
	if c >= '0' && c <= '9' {
		val = val*10 + (rune(c) - '0')
	} else {
		return 0, errors.New("invalid decimal CharRef")
	}
	return val, nil
}

func accumulateHexCharRef(val int32, c rune) (int32, error) {
	if c >= '0' && c <= '9' {
		val = val*16 + (rune(c) - '0')
	} else if c >= 'a' && c <= 'f' {
		val = val*16 + (rune(c) - 'a') + 10
	} else if c >= 'A' && c <= 'F' {
		val = val*16 + (rune(c) - 'A') + 10
	} else {
		return 0, errors.New("invalid hex CharRef")
	}
	return val, nil
}

// returns rune, byteCount, error
func parseStringCharRef(s []byte) (r rune, width int, err error) {
	if pdebug.Enabled {
		g := pdebug.Marker("parseStringCharRef")
		defer func() {
			pdebug.Printf("r = '%c' (%x), consumed %d bytes", &r, &r, &width)
			g.End()
		}()
	}
	var val int32
	r = utf8.RuneError
	width = 0
	if !bytes.HasPrefix(s, []byte{'&', '#'}) {
		err = errors.New("ampersand (&) was required")
		return
	}

	width += 2
	s = s[2:]

	var accumulator func(int32, rune) (int32, error)
	if s[0] == 'x' {
		s = s[1:]
		width++
		accumulator = accumulateHexCharRef
	} else {
		accumulator = accumulateDecimalCharRef
	}

	for c := s[0]; c != ';'; c = s[0] {
		val, err = accumulator(val, rune(c))
		if err != nil {
			width = 0
			return
		}
		if rune(val) > unicode.MaxRune {
			err = errors.New("hex CharRef out of range")
			width = 0
			return
		}

		s = s[1:]
		width++
	}

	if s[0] == ';' {
		s = s[1:]
		width++
	}

	r = rune(val)
	if !isChar(val) {
		return utf8.RuneError, 0, fmt.Errorf("invalid XML char value %d", val)
	}
	return
}

func parseStringName(s []byte) (string, int, error) {
	i := 0
	r, w := utf8.DecodeRune(s)
	if r == utf8.RuneError {
		return "", 0, errors.New("rune decode failed")
	}

	if !isNameStartChar(r) {
		return "", 0, errors.New("invalid name start char")
	}

	out := bufferPool.Get().(*bytes.Buffer)
	defer releaseBuffer(out)

	out.WriteRune(r)
	i += w
	s = s[w:]

	for {
		r, w = utf8.DecodeRune(s)
		if r == utf8.RuneError {
			return "", 0, errors.New("rune decode failed")
		}

		if !isNameChar(r) {
			break
		}
		out.WriteRune(r)
		i += w
		s = s[w:]
	}

	return out.String(), i, nil
}

// This will be called as a fallback. The SAX handler
// may totally decide to ignore entity related processing
// but we still need to resolve the entity in order for
// the rest of the processing to work.
func (ctx *parserCtx) getEntity(name string) (*Entity, error) {
	if ctx.inSubset == 0 {
		if ret, err := resolvePredefinedEntity(name); err != nil {
			return ret, nil
		}
	}

	var ret *Entity
	var ok bool
	if ctx.doc == nil || ctx.doc.standalone != 1 {
		ret, _ = ctx.doc.GetEntity(name)
	} else {
		if ctx.inSubset == 2 {
			ctx.doc.standalone = 0
			ret, _ = ctx.doc.GetEntity(name)
			ctx.doc.standalone = 1
		} else {
			ret, ok = ctx.doc.GetEntity(name)
			if !ok {
				ctx.doc.standalone = 0
				ret, ok = ctx.doc.GetEntity(name)
				if !ok {
					return nil, errors.New("Entity(" + name + ") document marked standalone but requires eternal subset")
				}
				ctx.doc.standalone = 1
			}
		}
	}
	/*
	   if ((ret != NULL) &&
	       ((ctxt->validate) || (ctxt->replaceEntities)) &&
	       (ret->children == NULL) &&
	       (ret->etype == XML_EXTERNAL_GENERAL_PARSED_ENTITY)) {
	       int val;

	       // for validation purposes we really need to fetch and
	       // parse the external entity
	       xmlNodePtr children;
	       unsigned long oldnbent = ctxt->nbentities;

	       val = xmlParseCtxtExternalEntity(ctxt, ret->URI,
	                                        ret->ExternalID, &children);
	       if (val == 0) {
	           xmlAddChildList((xmlNodePtr) ret, children);
	       } else {
	           xmlFatalErrMsg(ctxt, XML_ERR_ENTITY_PROCESSING,
	                          "Failure to process entity %s\n", name, NULL);
	           ctxt->validate = 0;
	           return(NULL);
	       }
	       ret->owner = 1;
	       if (ret->checked == 0) {
	           ret->checked = (ctxt->nbentities - oldnbent + 1) * 2;
	           if ((ret->content != NULL) && (xmlStrchr(ret->content, '<')))
	               ret->checked |= 1;
	       }
	   }
	*/
	return ret, nil
}

func (ctx *parserCtx) parseStringEntityRef(s []byte) (sax.Entity, int, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseStringEntityRef ('%s')", s)
		defer g.IRelease("END parseStringEntityRef")
	}
	if len(s) == 0 || s[0] != '&' {
		return nil, 0, errors.New("invalid entity ref")
	}

	i := 1 // skip the '&'
	name, width, err := parseStringName(s[1:]) // skip the '&' for name parsing
	if err != nil {
		return nil, 0, errors.New("failed to parse name")
	}
	s = s[width+1:] // skip '&' + name
	i += width

	if s[0] != ';' {
		return nil, 0, ErrSemicolonRequired
	}
	s = s[1:]
	i++

	var loadedEnt sax.Entity

	/*
	 * Ask first SAX for entity resolution, otherwise try the
	 * entities which may have stored in the parser context.
	 */
	if h := ctx.sax; h != nil {
		loadedEnt, err = h.GetEntity(ctx.userData, name)
		if err != nil {
			// Note: libxml2 would try to ask for xmlGetPredefinedEntity
			// next, but that's only when XML_PARSE_OLDSAX is enabled.
			// we won't do that.
			if ctx.wellFormed && ctx.userData == ctx {
				loadedEnt, err = ctx.getEntity(name)
				if err != nil {
					return nil, 0, err
				}
			}
		}
	}
	/*
	 * [ WFC: Entity Declared ]
	 * In a document without any DTD, a document with only an
	 * internal DTD subset which contains no parameter entity
	 * references, or a document with "standalone='yes'", the
	 * Name given in the entity reference must match that in an
	 * entity declaration, except that well-formed documents
	 * need not declare any of the following entities: amp, lt,
	 * gt, apos, quot.
	 * The declaration of a parameter entity must precede any
	 * reference to it.
	 * Similarly, the declaration of a general entity must
	 * precede any reference to it which appears in a default
	 * value in an attribute-list declaration. Note that if
	 * entities are declared in the external subset or in
	 * external parameter entities, a non-validating processor
	 * is not obligated to read and process their declarations;
	 * for such documents, the rule that an entity must be
	 * declared is a well-formedness constraint only if
	 * standalone='yes'.
	 */
	if loadedEnt == nil {
		if ctx.standalone == StandaloneExplicitYes || (!ctx.hasExternalSubset && !ctx.hasPERefs) {
			return nil, 0, fmt.Errorf("entity '%s' not defined", name)
		}
		// xmlParserEntityCheck ?!
	}

	/*
	 * [ WFC: Parsed Entity ]
	 * An entity reference must not contain the name of an
	 * unparsed entity
	 */

	if EntityType(loadedEnt.EntityType()) == ExternalGeneralUnparsedEntity {
		return nil, 0, fmt.Errorf("entity reference to unparsed entity '%s'", name)
	}

	/*
	 * [ WFC: No External Entity References ]
	 * Attribute values cannot contain direct or indirect
	 * entity references to external entities.
	 */
	if ctx.instate == psAttributeValue && EntityType(loadedEnt.EntityType()) == ExternalGeneralParsedEntity {
		return nil, 0, fmt.Errorf("attribute references enternal entity '%s'", name)
	}

	/*
	 * [ WFC: No < in Attribute Values ]
	 * The replacement text of any entity referred to directly or
	 * indirectly in an attribute value (other than "&lt;") must
	 * not contain a <.
	 */
	if ctx.instate == psAttributeValue && len(loadedEnt.Content()) > 0 && EntityType(loadedEnt.EntityType()) == InternalPredefinedEntity && bytes.IndexByte(loadedEnt.Content(), '<') > -1 {
		return nil, 0, fmt.Errorf("'<' in entity '%s' is not allowed in attribute values", name)
	}

	/*
	 * Internal check, no parameter entities here ...
	 */

	switch EntityType(loadedEnt.EntityType()) {
	case InternalParameterEntity, ExternalParameterEntity:
		return nil, 0, fmt.Errorf("attempt to reference the parameter entity '%s'", name)
	}

	return loadedEnt, i, nil
}

func (ctx *parserCtx) parseStringPEReference(s []byte) (sax.Entity, int, error) {
	if len(s) == 0 || s[0] != '%' {
		return nil, 0, errors.New("invalid PEreference")
	}

	i := 1 // skip the '%'
	name, width, err := parseStringName(s[1:]) // skip the '%' for name parsing
	if err != nil {
		return nil, 0, err
	}
	s = s[width+1:] // skip '%' + name
	i += width

	if s[0] != ';' {
		return nil, 0, ErrSemicolonRequired
	}
	s = s[1:]
	i++

	var loadedEnt sax.Entity
	if h := ctx.sax; h != nil {
		loadedEnt, err = h.GetParameterEntity(ctx.userData, name)
		if err != nil {
			return nil, 0, err
		}
	}

	/*
	 * [ WFC: Entity Declared ]
	 * In a document without any DTD, a document with only an
	 * internal DTD subset which contains no parameter entity
	 * references, or a document with "standalone='yes'", ...
	 * ... The declaration of a parameter entity must precede
	 * any reference to it...
	 */
	if loadedEnt == nil {
		if ctx.standalone == StandaloneExplicitYes || (!ctx.hasExternalSubset && !ctx.hasPERefs) {
			return nil, 0, fmt.Errorf("not found: PE rerefence '%%%s'", name)
		} else {
			ctx.valid = false
		}
		// xmlParseEntityCheck(ctxt, 0, NULL, 0)
	} else {
		switch EntityType(loadedEnt.EntityType()) {
		case InternalParameterEntity, ExternalParameterEntity:
		default:
			return nil, 0, fmt.Errorf("not a parmeter entity: %%%s", name)
		}
	}
	ctx.hasPERefs = true

	return loadedEnt, i, nil
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
func (ctx *parserCtx) parseCharRef() (r rune, err error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseCharRef")
		defer g.IRelease("END parseCharRef")
		defer func() { pdebug.Printf("r = '%c' (%x)", r, r) }()
	}

	r = utf8.RuneError

	var val int32
	cur := ctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if cur.ConsumeString("&#x") {
		for c := cur.Peek(); !cur.Done() && c != ';'; c = cur.Peek() {
			if c >= '0' && c <= '9' {
				val = val*16 + (c - '0')
			} else if c >= 'a' && c <= 'f' {
				val = val*16 + (c - 'a') + 10
			} else if c >= 'A' && c <= 'F' {
				val = val*16 + (c - 'A') + 10
			} else {
				err = errors.New("invalid hex CharRef")
				return
			}
			cur.Advance(1)
		}
		if cur.Peek() == ';' {
			cur.Advance(1)
		}
	} else if cur.ConsumeString("&#") {
		for !cur.Done() && cur.Peek() != ';' {
			c := cur.Peek()
			if c >= '0' && c <= '9' {
				val = val*10 + (c - '0')
			} else {
				err = errors.New("invalid decimal CharRef")
				return
			}
			cur.Advance(1)
		}
		if cur.Peek() == ';' {
			cur.Advance(1)
		}
	} else {
		err = errors.New("invalid char ref")
		return
	}

	if isChar(val) && val <= unicode.MaxRune {
		r = rune(val)
		return
	}

	err = ErrInvalidChar
	return
}

/*
 * parse ENTITY references declarations
 *
 * [68] EntityRef ::= '&' Name ';'
 *
 * [ WFC: Entity Declared ]
 * In a document without any DTD, a document with only an internal DTD
 * subset which contains no parameter entity references, or a document
 * with "standalone='yes'", the Name given in the entity reference
 * must match that in an entity declaration, except that well-formed
 * documents need not declare any of the following entities: amp, lt,
 * gt, apos, quot.  The declaration of a parameter entity must precede
 * any reference to it.  Similarly, the declaration of a general entity
 * must precede any reference to it which appears in a default value in an
 * attribute-list declaration. Note that if entities are declared in the
 * external subset or in external parameter entities, a non-validating
 * processor is not obligated to read and process their declarations;
 * for such documents, the rule that an entity must be declared is a
 * well-formedness constraint only if standalone='yes'.
 *
 * [ WFC: Parsed Entity ]
 * An entity reference must not contain the name of an unparsed entity
 *
 * Returns the xmlEntityPtr if found, or NULL otherwise.
 */
func (ctx *parserCtx) parseEntityRef() (ent *Entity, err error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseEntityRef")
		defer func() {
			g.IRelease("END parseEntityRef ent = %#v", ent)
		}()
	}

	cur := ctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if cur.Peek() != '&' {
		err = ctx.error(ErrAmpersandRequired)
		return
	}
	cur.Advance(1)

	name, err := ctx.parseName()
	if err != nil {
		err = ctx.error(ErrNameRequired)
		return
	}

	if cur.Peek() != ';' {
		err = ctx.error(ErrSemicolonRequired)
		return
	}
	cur.Advance(1)

	if ent, err = resolvePredefinedEntity(name); err == nil {
		return
	}

	if s := ctx.sax; s != nil {
		// ask the SAX2 handler nicely
		var loadedEnt sax.Entity
		loadedEnt, err = s.GetEntity(ctx.userData, name)
		if err == nil {
			ent = loadedEnt.(*Entity)
			return
		}

		if loadedEnt == nil && ctx == ctx.userData {
			ent, _ = ctx.getEntity(name)
		}
	}

	// [ WFC: Entity Declared ]
	// In a document without any DTD, a document with only an
	// internal DTD subset which contains no parameter entity
	// references, or a document with "standalone='yes'", the
	// Name given in the entity reference must match that in an
	// entity declaration, except that well-formed documents
	// need not declare any of the following entities: amp, lt,
	// gt, apos, quot.
	// The declaration of a parameter entity must precede any
	// reference to it.
	// Similarly, the declaration of a general entity must
	// precede any reference to it which appears in a default
	// value in an attribute-list declaration. Note that if
	// entities are declared in the external subset or in
	// external parameter entities, a non-validating processor
	// is not obligated to read and process their declarations;
	// for such documents, the rule that an entity must be
	// declared is a well-formedness constraint only if
	// standalone='yes'.
	if ent == nil {
		if ctx.standalone == StandaloneExplicitYes || (!ctx.hasExternalSubset && ctx.hasPERefs) {
			return nil, ctx.error(ErrUndeclaredEntity)
		} else {
			if ctx.inSubset == 0 {
				if s := ctx.sax; s != nil {
					switch err := s.Reference(ctx.userData, name); err {
					case nil, sax.ErrHandlerUnspecified:
						// no op
					default:
						return nil, ctx.error(err)
					}
				}
			}
			// ent is nil, no? why check?
			if err := ctx.entityCheck(ent, 0, 0); err != nil {
				return nil, ctx.error(err)
			}
			ctx.valid = false
		}
	} else if ent.entityType == ExternalGeneralUnparsedEntity {
		// [ WFC: Parsed Entity ]
		// An entity reference must not contain the name of an
		// unparsed entity
		return nil, ctx.error(errors.New("entity reference to unparsed entity"))
	} else if ctx.instate == psAttributeValue && ent.entityType == ExternalGeneralParsedEntity {
		// [ WFC: No External Entity References ]
		// Attribute values cannot contain direct or indirect
		// entity references to external entities.
		return nil, ctx.error(errors.New("attribute references external entity"))
	} else if ctx.instate == psAttributeValue && ent.entityType != InternalPredefinedEntity {
		// [ WFC: No < in Attribute Values ]
		// The replacement text of any entity referred to directly or
		// indirectly in an attribute value (other than "&lt;") must
		// not contain a <.
		if (ent.checked&1 == 1 || ent.checked == 0) && ent.content != "" && strings.IndexByte(ent.content, '<') > -1 {
			return nil, ctx.error(errors.New("'<' in entity is not allowed in attribute values"))
		}
	} else {
		// Internal check, no parameter entities here ...
		switch ent.entityType {
		case InternalParameterEntity:
		case ExternalParameterEntity:
			return nil, ctx.error(errors.New("attempt to reference the parameter entity"))
		}
	}

	if ent == nil {
		panic("at the end of parseEntityRef, ent == nil")
	}
	// [ WFC: No Recursion ]
	// A parsed entity must not contain a recursive reference
	// to itself, either directly or indirectly.
	// Done somewhere else
	return ent, nil
}

/* Function to check non-linear entity expansion behaviour
 * This is here to detect and stop exponential linear entity expansion
 * This is not a limitation of the parser but a safety
 * boundary feature. It can be disabled with the XML_PARSE_HUGE
 * parser option.
 */
func (ctx *parserCtx) entityCheck(ent sax.Entity, size, replacement int) error {
	return nil
	/*
	   size_t consumed = 0;

	   if ((ctxt == NULL) || (ctxt->options & XML_PARSE_HUGE))
	       return (0);
	   if (ctxt->lastError.code == XML_ERR_ENTITY_LOOP)
	       return (1);
	*/

	// This may look absurd but is needed to detect
	// entities problems
	/*
		if ent != nil && EntityType(ent.EntityType()) != InternalPredefinedEntity && ent.Content() != nil && !ent.Checked() {
			rep, err := decodeEntities(ent.Content(), SubstituteRef)
			if err != nil {
				return ctx.error(err)
			}
		}

		return nil
	*/
}

func (ctx *parserCtx) handlePEReference() error {
	if pdebug.Enabled {
		g := pdebug.Marker("handlePEReference")
		defer g.End()
	}

	cur := ctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if cur.Peek() != '%' {
		// it's fine, this is not an error.
		return nil
	}

	switch st := ctx.instate; st {
	case psCDATA, psComment, psStartTag, psEndTag, psEntityDecl, psContent, psAttributeValue, psPI, psSystemLiteral, psPublicLiteral, psEntityValue, psIgnore:
		// NOTE: in the case of entity values, we don't do the
		//       substitution here since we need the literal
		//       entity value to be able to save the internal
		//       subset of the document.
		//       This will be handled by xmlStringDecodeEntities
		if pdebug.Enabled {
			pdebug.Printf("instate == %s, ignoring", st)
		}
		return nil
	case psEOF:
		if pdebug.Enabled {
			pdebug.Printf("parameter entity at EOF")
		}
		return errors.New("handlePEReference: parameter entity at EOF")
	case psPrologue, psStart, psMisc:
		if pdebug.Enabled {
			pdebug.Printf("parameter entity in prologue")
		}
		return errors.New("handlePEReference: parameter entity in prologue")
	case psEpilogue:
		if pdebug.Enabled {
			pdebug.Printf("parameter entity in epilogue")
		}
		return errors.New("handlePEReference: parameter entity in epilogue")
	case psDTD:
		if pdebug.Enabled {
			pdebug.Printf("parameter entity in DTD")
		}
		// [WFC: Well-Formedness Constraint: PEs in Internal Subset]
		// In the internal DTD subset, parameter-entity references
		// can occur only where markup declarations can occur, not
		// within markup declarations.
		// In that case this is handled in xmlParseMarkupDecl
		if pdebug.Enabled {
			pdebug.Printf("DTD external = %t, inputNr = %d", ctx.external, ctx.inputTab.Len())
		}
		if !ctx.external || ctx.inputTab.Len() == 1 {
			if pdebug.Enabled {
				pdebug.Printf("we're NOT in external DTD, bail out")
			}
			return nil
		}

		if c := cur.PeekN(2); isBlankCh(c) || c == 0x0 {
			return nil
		}
	}

	cur.Advance(1)

	name, err := ctx.parseName()
	if err != nil {
		return err
	}
	if pdebug.Enabled {
		pdebug.Printf("entity name: '%s'", name)
	}

	if cur.Peek() != ';' {
		return ErrSemicolonRequired
	}

	cur.Advance(1)

	var entity sax.Entity
	if s := ctx.sax; s != nil {
		entity, _ = s.GetParameterEntity(ctx.userData, name)
	}

	if ctx.instate == psEOF {
		return nil
	}

	if entity == nil {
		// [ WFC: Entity Declared ]
		// In a document without any DTD, a document with only an
		// internal DTD subset which contains no parameter entity
		// references, or a document with "standalone='yes'", ...
		// ... The declaration of a parameter entity must precede
		// any reference to it...
		if ctx.standalone == StandaloneExplicitYes || (!ctx.hasExternalSubset && !ctx.hasPERefs) {
			return fmt.Errorf("undeclared entity: PEReference: %%%s; not found", name)
		}
		// [ VC: Entity Declared ]
		// In a document with an external subset or external
		// parameter entities with "standalone='no'", ...
		// ... The declaration of a parameter entity must precede
		// any reference to it...
		/*
		   if ((ctxt->validate) && (ctxt->vctxt.error != NULL)) {
		       xmlValidityError(ctxt, XML_WAR_UNDECLARED_ENTITY,
		                        "PEReference: %%%s; not found\n",
		                        name, NULL);
		   } else
		       xmlWarningMsg(ctxt, XML_WAR_UNDECLARED_ENTITY,
		                     "PEReference: %%%s; not found\n",
		                     name, NULL);
		*/
		ctx.valid = false
		ctx.entityCheck(nil, 0, 0)
		pdebug.Printf("Should be calling pushInput here")
		/* have no clue what this is for
		   } else if (ctxt->input->free != deallocblankswrapper) {
		           input = xmlNewBlanksWrapperInputStream(ctxt, entity);
		           if (xmlPushInput(ctxt, input) < 0)
		               return;
		*/
	} else {
		switch EntityType(entity.EntityType()) {
		case InternalParameterEntity, ExternalParameterEntity:
			// OK
		default:
			return fmt.Errorf("entity is a parameter: PEReference: %%%s; is not a parameter entity", name)
		}

		// Note: external parameter entities will not be loaded, it
		// is not required for a non-validating parser, unless the
		// option of validating, or substituting entities were
		// given. Doing so is far more secure as the parser will
		// only process data coming from the document entity by
		// default.
		/*
			  if EntityType(entity.EntityType()) == ExternalParameterEntity) &&
				       ((ctxt->options & XML_PARSE_NOENT) == 0) &&
				       ((ctxt->options & XML_PARSE_DTDVALID) == 0) &&
				       ((ctxt->options & XML_PARSE_DTDLOAD) == 0) &&
				       ((ctxt->options & XML_PARSE_DTDATTR) == 0) &&
				       (ctxt->replaceEntities == 0) &&
				       (ctxt->validate == 0))
				       return;
		*/
		if pdebug.Enabled {
			pdebug.Printf("handlePEReference: found entity '%s' with content: %s", name, string(entity.Content()))
		}
		// Note: Parameter entity expansion is handled in parsePEReference, not here
		// This function is called from a different context (skip blanks)

		/*
		           // Get the 4 first bytes and decode the charset
		           // if enc != XML_CHAR_ENCODING_NONE
		           // plug some encoding conversion routines.
		           // Note that, since we may have some non-UTF8
		           // encoding (like UTF16, bug 135229), the 'length'
		           // is not known, but we can calculate based upon
		           // the amount of data in the buffer.
		           GROW
		           if (ctxt->instate == XML_PARSER_EOF)
		               return;
		           if ((ctxt->input->end - ctxt->input->cur)>=4) {
		               start[0] = RAW;
		               start[1] = NXT(1);
		               start[2] = NXT(2);
		               start[3] = NXT(3);
		               enc = xmlDetectCharEncoding(start, 4);
		               if (enc != XML_CHAR_ENCODING_NONE) {
		                   xmlSwitchEncoding(ctxt, enc);
		               }
		           }

		           if ((entity->etype == XML_EXTERNAL_PARAMETER_ENTITY) &&
		               (CMP5(CUR_PTR, '<', '?', 'x', 'm', 'l' )) &&
		               (IS_BLANK_CH(NXT(5)))) {
		               xmlParseTextDecl(ctxt);
		           }
		       } else {
		           xmlFatalErrMsgStr(ctxt, XML_ERR_ENTITY_IS_PARAMETER,
		                    "PEReference: %s is not a parameter entity\n",
		                             name);
		       }
		   }
		*/

	}
	return nil
}
