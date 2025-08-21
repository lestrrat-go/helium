package helium

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/lestrrat-go/helium/encoding"
	"github.com/lestrrat-go/helium/sax"
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

func (pctx *parserCtx) pushNS(ctx context.Context, prefix, uri string) {
	TraceEvent(ctx, "pushing namespace to stack",
		slog.String("prefix", prefix),
		slog.String("uri", uri),
		slog.Int("ns_stack_depth", pctx.nsTab.Len()+1))
	pctx.nsTab.Push(prefix, uri)
}

const (
	cbEntityDecl = iota
	cbGetParameterEntity
)

func (pctx *parserCtx) fireSAXCallback(ctx context.Context, typ int, args ...interface{}) error {
	// This is ugly, but I *REALLY* wanted to catch all occurences of
	// SAX callbacks being fired in one shot. optimize it later

	s := pctx.sax
	if s == nil {
		return nil
	}

	switch typ {
	case cbEntityDecl:
		return s.EntityDecl(ctx, pctx.userData, args[0].(string), int(InternalParameterEntity), "", "", args[1].(string))
	case cbGetParameterEntity:
		entity, err := s.GetParameterEntity(ctx, pctx, args[1].(string))
		if err == nil {
			ret := args[0].(*sax.Entity)
			*ret = entity
		}
		return err
	}
	return nil
}

func (pctx *parserCtx) pushNode(ctx context.Context, e *Element) {
	TraceEvent(ctx, "pushing node to stack",
		slog.String("element_name", e.Name()),
		slog.Int("stack_depth", pctx.nodeTab.Len()+1))
	pctx.nodeTab.Push(e)
}

func (pctx *parserCtx) peekNode() *Element {
	return pctx.nodeTab.PeekOne()
}

func (pctx *parserCtx) popNode(ctx context.Context) (elem *Element) {
	elem = pctx.nodeTab.Pop()
	if elem != nil {
		TraceEvent(ctx, "popped node from stack",
			slog.String("element_name", elem.Name()),
			slog.Int("stack_depth", pctx.nodeTab.Len()))
	} else {
		TraceEvent(ctx, "attempted to pop from empty node stack")
	}
	return elem
}

func (pctx *parserCtx) lookupNamespace(ctx context.Context, prefix string) string {
	uri := pctx.nsTab.Lookup(prefix)
	TraceEvent(ctx, "namespace lookup",
		slog.String("prefix", prefix),
		slog.String("uri", uri),
		slog.Bool("found", uri != ""))
	return uri
}

func (pctx *parserCtx) release() error {
	pctx.sax = nil
	pctx.userData = nil
	return nil
}

var bufferPool = sync.Pool{
	New: allocByteBuffer,
}

func allocByteBuffer() interface{} {
	return &bytes.Buffer{}
}

func releaseBuffer(b *bytes.Buffer) {
	b.Reset()
	bufferPool.Put(b)
}

func (pctx *parserCtx) pushInput(ctx context.Context, in interface{}) {
	ctx, span := StartSpan(ctx, "pushInput")
	defer span.End()

	TraceEvent(ctx, "pushing input to stack", slog.Int("stack_depth", pctx.inputTab.Len()+1))
	pctx.inputTab.Push(in)
	TraceEvent(ctx, "input pushed", slog.Int("new_stack_depth", pctx.inputTab.Len()))
}

func (pctx *parserCtx) getByteCursor() *strcursor.ByteCursor {
	cur, ok := pctx.inputTab.PeekOne().(*strcursor.ByteCursor)
	if !ok {
		return nil
	}
	return cur
}

func (pctx *parserCtx) getCursor(ctx context.Context) strcursor.Cursor {
	ctx, span := StartSpan(ctx, "getCursor")
	defer span.End()

	// Pop exhausted input streams and return the next available cursor
	for pctx.inputTab.Len() > 0 {
		cur, ok := pctx.inputTab.PeekOne().(strcursor.Cursor)
		if !ok {
			TraceEvent(ctx, "invalid cursor type, popping input")
			pctx.popInput(ctx)
			continue
		}
		if cur.Done() && pctx.inputTab.Len() > 1 {
			// Current input is exhausted, pop it and try the next
			TraceEvent(ctx, "current input exhausted, popping and trying next")
			pctx.popInput(ctx)
			continue
		}
		TraceEvent(ctx, "returning active cursor")
		return cur
	}
	TraceEvent(ctx, "no available cursors")
	return nil
}

func (pctx *parserCtx) popInput(ctx context.Context) interface{} {
	ctx, span := StartSpan(ctx, "popInput")
	defer span.End()

	TraceEvent(ctx, "popping input from stack", slog.Int("stack_depth", pctx.inputTab.Len()))
	result := pctx.inputTab.Pop()
	TraceEvent(ctx, "input popped", slog.Int("new_stack_depth", pctx.inputTab.Len()))
	return result
}

func (pctx *parserCtx) init(ctx context.Context, p *Parser, in io.Reader) error {
	pctx.pushInput(ctx, strcursor.NewByteCursor(in))
	pctx.detectedEncoding = encUTF8
	pctx.encoding = ""
	pctx.in = in
	pctx.nbread = 0
	pctx.keepBlanks = true
	pctx.instate = psStart
	pctx.userData = pctx // circular dep?!
	pctx.standalone = StandaloneImplicitNo
	pctx.attsSpecial = map[string]AttributeType{}
	pctx.attsDefault = map[string]map[string]*Attribute{}
	pctx.wellFormed = true
	if p != nil {
		pctx.sax = p.sax
	}
	return nil
}

func (pctx *parserCtx) error(ctx context.Context, err error) error {
	// If it's wrapped, just return as is
	if _, ok := err.(ErrParseError); ok {
		return err
	}

	e := ErrParseError{Err: err}
	if cur := pctx.getCursor(ctx); cur != nil {
		e.Column = cur.Column()
		e.LineNumber = cur.LineNumber()
		e.Line = cur.Line()
	}
	return e
}

// errorWithoutContext is used for callback functions that don't have access to context
func (pctx *parserCtx) errorWithoutContext(err error) error {
	// If it's wrapped, just return as is
	if _, ok := err.(ErrParseError); ok {
		return err
	}

	e := ErrParseError{Err: err}
	// Skip cursor information since we don't have context
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

func (pctx *parserCtx) detectEncoding() (encoding string, err error) {
	cur := pctx.getByteCursor()
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

func (pctx *parserCtx) switchEncoding(ctx context.Context) error {
	ctx, span := StartSpan(ctx, "switchEncoding")
	defer span.End()

	encName := pctx.encoding
	if encName == "" {
		encName = pctx.detectedEncoding
		if encName == "" {
			encName = "utf8"
		}
	}

	TraceEvent(ctx, "switching encoding", slog.String("encoding_name", encName))

	enc := encoding.Load(encName)
	if enc == nil {
		err := errors.New("encoding '" + encName + "' not supported")
		return err
	}

	cur := pctx.getByteCursor()
	if cur == nil {
		return ErrByteCursorRequired
	}

	TraceEvent(ctx, "creating decoder and switching input stream")
	b := enc.NewDecoder().Reader(cur)
	pctx.popInput(ctx)
	pctx.pushInput(ctx, strcursor.NewRuneCursor(b))

	TraceEvent(ctx, "encoding switch completed", slog.String("encoding", encName))
	return nil
}

var xmlDeclHint = []byte{'<', '?', 'x', 'm', 'l'}

func (pctx *parserCtx) parseDocument(ctx context.Context) error {
	tlog := getTraceLogFromContext(ctx)
	tlog.Debug("START")
	defer tlog.Debug("END")

	if s := pctx.sax; s != nil {
		switch err := s.SetDocumentLocator(ctx, pctx.userData, nil); err {
		case nil, sax.ErrHandlerUnspecified:
			// no op
		default:
			return pctx.error(ctx, err)
		}
	}

	// see if we can find the preliminary encoding
	if pctx.encoding == "" {
		if enc, err := pctx.detectEncoding(); err == nil {
			pctx.detectedEncoding = enc
		}
	}

	// At this stage we MUST be using a ByteCursor, as we
	// don't know what the encoding is.
	bcur := pctx.getByteCursor()
	if bcur == nil {
		return pctx.error(ctx, ErrByteCursorRequired)
	}

	// nothing left? eek
	if bcur.Done() {
		return pctx.error(ctx, errors.New("empty document"))
	}

	// XML prolog
	if bcur.HasPrefix(xmlDeclHint) {
		if err := pctx.parseXMLDecl(ctx); err != nil {
			return pctx.error(ctx, err)
		}
	}

	// At this point we know the encoding, so switch the encoding
	// of the source.
	if err := pctx.switchEncoding(ctx); err != nil {
		return pctx.error(ctx, err)
	}

	if s := pctx.sax; s != nil {
		TraceEvent(ctx, "calling SAX StartDocument")
		switch err := s.StartDocument(ctx, pctx.userData); err {
		case nil, sax.ErrHandlerUnspecified:
			// no op
		default:
			return pctx.error(ctx, err)
		}
	}

	// Misc part of the prolog
	if err := pctx.parseMisc(ctx); err != nil {
		return pctx.error(ctx, err)
	}

	cur := pctx.getCursor(ctx)
	if cur == nil {
		panic("did not get rune cursor")
	}
	// Doctype declarations and more misc
	if cur.HasPrefixString("<!DOCTYPE") {
		pctx.inSubset = inInternalSubset
		if err := pctx.parseDocTypeDecl(ctx); err != nil {
			return pctx.error(ctx, err)
		}

		if cur.HasPrefixString("[") {
			pctx.instate = psDTD
			if err := pctx.parseInternalSubset(ctx); err != nil {
				return pctx.error(ctx, err)
			}
		}

		pctx.inSubset = inExternalSubset
		if s := pctx.sax; s != nil {
			switch err := s.ExternalSubset(ctx, pctx.userData, pctx.intSubName, pctx.extSubSystem, pctx.extSubURI); err {
			case nil, sax.ErrHandlerUnspecified:
				// no op
			default:
				return pctx.error(ctx, err)
			}
		}
		if pctx.instate == psEOF {
			return pctx.error(ctx, errors.New("unexpected EOF"))
		}
		pctx.inSubset = notInSubset

		pctx.cleanSpecialAttributes()

		pctx.instate = psPrologue
		if err := pctx.parseMisc(ctx); err != nil {
			return pctx.error(ctx, err)
		}
	}
	pctx.skipBlanks(ctx)

	if cur.Peek() != '<' {
		return pctx.error(ctx, ErrEmptyDocument)
	} else {
		pctx.instate = psContent
		if err := pctx.parseElement(ctx); err != nil {
			return pctx.error(ctx, err)
		}
		pctx.instate = psEpilogue

		if err := pctx.parseMisc(ctx); err != nil {
			return pctx.error(ctx, err)
		}
		if !cur.Done() {
			return pctx.error(ctx, ErrDocumentEnd)
		}
		pctx.instate = psEOF
	}

	/*
		// Start the actual tree
		if err := ctx.parseContent(context.Background()); err != nil {
			return ctx.error(err)
		}

		if err := ctx.parseEpilogue(); err != nil {
			return ctx.error(err)
		}
	*/

	// All done
	if s := pctx.sax; s != nil {
		TraceEvent(ctx, "calling SAX EndDocument")
		switch err := s.EndDocument(ctx, pctx.userData); err {
		case nil, sax.ErrHandlerUnspecified:
			// no op
		default:
			return pctx.error(ctx, err)
		}
	}

	return nil
}

func (pctx *parserCtx) parseContent(ctx context.Context) error {
	ctx, span := StartSpan(ctx, "parseContent")
	defer span.End()
	pctx.instate = psContent

	cur := pctx.getCursor(ctx)
	if cur == nil {
		panic("did not get rune cursor")
	}

	for !cur.Done() {
		if cur.HasPrefixString("</") {
			break
		}

		if cur.HasPrefixString("<?") {
			if err := pctx.parsePI(ctx); err != nil {
				return pctx.error(ctx, err)
			}
			continue
		}

		if cur.HasPrefixString("<![CDATA[") {
			if err := pctx.parseCDSect(ctx); err != nil {
				return pctx.error(ctx, err)
			}
			continue
		}

		if cur.HasPrefixString("<!--") {
			if err := pctx.parseComment(ctx); err != nil {
				return pctx.error(ctx, err)
			}
			continue
		}

		if cur.HasPrefixString("<") {
			if err := pctx.parseElement(ctx); err != nil {
				return pctx.error(ctx, err)
			}
			continue
		}

		if cur.HasPrefixString("&") {
			if err := pctx.parseReference(ctx); err != nil {
				return pctx.error(ctx, err)
			}
			continue
		}

		if err := pctx.parseCharData(ctx, false); err != nil {
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
func (pctx *parserCtx) parseCharData(ctx context.Context, cdata bool) error {
	ctx, span := StartSpan(ctx, "parseCharData")
	defer span.End()
	buf := bufferPool.Get().(*bytes.Buffer)
	defer releaseBuffer(buf)

	cur := pctx.getCursor(ctx)
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

			return pctx.error(ctx, ErrMisplacedCDATAEnd)
		}

		_, _ = buf.WriteRune(c)
		i++
	}

	if i <= 0 {
		return errors.New("invalid char data")
	}

	if err := cur.Advance(i); err != nil {
		return err
	}
	str := buf.String()

	// XXX This is not right, but it's for now the best place to do this
	str = strings.ReplaceAll(str, "\r\n", "\n")
	if pctx.instate == psCDATA {
		if s := pctx.sax; s != nil {
			TraceEvent(ctx, "calling SAX CDataBlock",
				slog.Int("content_length", len(str)))
			switch err := s.CDataBlock(ctx, pctx.userData, []byte(str)); err {
			case nil, sax.ErrHandlerUnspecified:
				// no op
			default:
				return pctx.error(ctx, err)
			}
		}
	} else if pctx.areBlanks(ctx, str, false) {
		if s := pctx.sax; s != nil {
			TraceEvent(ctx, "calling SAX IgnorableWhitespace",
				slog.Int("whitespace_length", len(str)))
			switch err := s.IgnorableWhitespace(ctx, pctx.userData, []byte(str)); err {
			case nil, sax.ErrHandlerUnspecified:
				// no op
			default:
				return pctx.error(ctx, err)
			}
		}
	} else {
		if s := pctx.sax; s != nil {
			TraceEvent(ctx, "calling SAX Characters",
				slog.String("content_type", "regular"),
				slog.Int("content_length", len(str)))
			switch err := s.Characters(ctx, pctx.userData, []byte(str)); err {
			case nil, sax.ErrHandlerUnspecified:
				// no op
			default:
				return pctx.error(ctx, err)
			}
		}
	}

	return nil
}

func (pctx *parserCtx) parseElement(ctx context.Context) error {
	ctx, span := StartSpan(ctx, "parseElement")
	defer span.End()
	// parseStartTag only parses up to the attributes.
	// For example, given <foo>bar</foo>, the next token would
	// be bar</foo>. Given <foo />, the next token would
	// be />
	if err := pctx.parseStartTag(ctx); err != nil {
		return pctx.error(ctx, err)
	}

	cur := pctx.getCursor(ctx)
	if cur == nil {
		panic("did not get rune cursor")
	}
	if !cur.HasPrefixString("/>") {
		if err := pctx.parseContent(ctx); err != nil {
			return pctx.error(ctx, err)
		}
	}

	if err := pctx.parseEndTag(ctx); err != nil {
		return pctx.error(ctx, err)
	}

	return nil
}

func (pctx *parserCtx) parseStartTag(ctx context.Context) error {
	ctx, span := StartSpan(ctx, "parseStartTag")
	defer span.End()
	cur := pctx.getCursor(ctx)
	if cur == nil {
		panic("did not get rune cursor")
	}
	if cur.Peek() != '<' {
		return pctx.error(ctx, ErrStartTagRequired)
	}
	if err := cur.Advance(1); err != nil {
		return err
	}

	local, prefix, err := pctx.parseQName(ctx)
	if local == "" {
		return pctx.error(ctx, fmt.Errorf("local name empty! local = %s, prefix = %s, err = %s", local, prefix, err))
	}
	if err != nil {
		return pctx.error(ctx, err)
	}

	elem, err := pctx.doc.CreateElement(local)
	if err != nil {
		return pctx.error(ctx, err)
	}

	TraceEvent(ctx, "parsing start tag",
		slog.String("element_name", local),
		slog.String("prefix", prefix))

	nbNs := 0
	attrs := []sax.Attribute{}
	for pctx.instate != psEOF {
		pctx.skipBlanks(ctx)
		if cur.Peek() == '>' {
			if err := cur.Advance(1); err != nil {
				return err
			}
			break
		}

		if cur.Peek() == '/' && cur.PeekN(2) == '>' {
			break
		}
		attname, aprefix, attvalue, err := pctx.parseAttribute(ctx, local)
		if err != nil {
			return pctx.error(ctx, err)
		}

		if attname == XMLNsPrefix && aprefix == "" {
			// <elem xmlns="...">
			pctx.pushNS(ctx, "", attvalue)
			nbNs++
			//    SkipDefaultNS:
			if cur.Peek() == '>' || cur.HasPrefixString("/>") {
				continue
			}

			if !isBlankCh(cur.Peek()) {
				return pctx.error(ctx, ErrSpaceRequired)
			}
			pctx.skipBlanks(ctx)
		} else if aprefix == XMLNsPrefix {
			var u *url.URL // predeclare, so we can use goto SkipNS

			// <elem xmlns:foo="...">
			if attname == XMLPrefix { // xmlns:xml
				if attvalue != XMLNamespace {
					return pctx.error(ctx, errors.New("xml namespace prefix mapped to wrong URI"))
				}
				// skip storing namespace definition
				goto SkipNS
			}
			if attname == XMLNsPrefix { // xmlns:xmlns="..."
				return pctx.error(ctx, errors.New("redefinition of the xmlns prefix forbidden"))
			}

			if attvalue == "http://www.w3.org/2000/xmlns/" {
				return pctx.error(ctx, errors.New("reuse of the xmlns namespace name if forbidden"))
			}

			if attvalue == "" {
				return pctx.error(ctx, fmt.Errorf("xmlns:%s: Empty XML namespace is not allowed", attname))
			}

			u, err = url.Parse(attvalue)
			if err != nil {
				return pctx.error(ctx, fmt.Errorf("xmlns:%s: '%s' is not a validURI", attname, attvalue))
			}
			if pctx.pedantic && u.Scheme == "" {
				return pctx.error(ctx, fmt.Errorf("xmlns:%s: URI %s is not absolute", attname, attvalue))
			}

			if pctx.nsTab.Lookup(attname) != "" {
				return pctx.error(ctx, errors.New("duplicate attribute is not allowed"))
			}
			pctx.pushNS(ctx, attname, attvalue)
			nbNs++

		SkipNS:
			if cur.Peek() == '>' || cur.HasPrefixString("/>") {
				continue
			}

			if !isBlankCh(cur.Peek()) {
				return pctx.error(ctx, ErrSpaceRequired)
			}
			pctx.skipBlanks(ctx)
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
	if len(pctx.attsDefault) > 0 {
		var elemName string
		if prefix != "" {
			elemName = prefix + ":" + local
		} else {
			elemName = local
		}

		defaults, ok := pctx.lookupAttributeDefault(elemName)
		if ok {
			for _, attr := range defaults {
				attrs = append(attrs, attr)
			}
		}
	}

	// we push the element first, because this way we get to
	// query for the namespace declared on this node as well
	// via lookupNamespace
	nsuri := pctx.lookupNamespace(ctx, prefix)
	if prefix != "" && nsuri == "" {
		return pctx.error(ctx, errors.New("namespace '"+prefix+"' not found"))
	}
	if nsuri != "" {
		if err := elem.SetNamespace(prefix, nsuri, true); err != nil {
			return err
		}
	}

	if s := pctx.sax; s != nil {
		var nslist []sax.Namespace
		if nbNs > 0 {
			nslist = make([]sax.Namespace, nbNs)
			// workaround []*Namespace != []sax.Namespace
			for i, ns := range pctx.nsTab.Peek(nbNs) {
				nslist[i] = ns.(nsStackItem)
			}
		}
		TraceEvent(ctx, "calling SAX StartElementNS",
			slog.String("element_name", elem.LocalName()),
			slog.String("prefix", prefix),
			slog.String("uri", nsuri),
			slog.Int("namespace_count", len(nslist)),
			slog.Int("attribute_count", len(attrs)))
		switch err := s.StartElementNS(ctx, pctx.userData, elem.LocalName(), prefix, nsuri, nslist, attrs); err {
		case nil, sax.ErrHandlerUnspecified:
			// no op
		default:
			return pctx.error(ctx, err)
		}
	}
	pctx.pushNode(ctx, elem)

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
func (pctx *parserCtx) parseEndTag(ctx context.Context) error {
	ctx, span := StartSpan(ctx, "parseEndTag")
	defer span.End()
	cur := pctx.getCursor(ctx)
	if cur == nil {
		panic("did not get rune cursor")
	}
	if !cur.ConsumeString("/>") {
		if !cur.ConsumeString("</") {
			return pctx.error(ctx, ErrLtSlashRequired)
		}

		e := pctx.peekNode()
		if !cur.ConsumeString(e.Name()) {
			return pctx.error(ctx, errors.New("expected end tag '"+e.Name()+"'"))
		}

		TraceEvent(ctx, "parsing end tag", slog.String("element_name", e.Name()))

		if cur.Peek() != '>' {
			return pctx.error(ctx, ErrGtRequired)
		}
		if err := cur.Advance(1); err != nil {
			return err
		}
	}

	e := pctx.peekNode()
	if s := pctx.sax; s != nil {
		TraceEvent(ctx, "calling SAX EndElementNS",
			slog.String("element_name", e.LocalName()),
			slog.String("prefix", e.Prefix()),
			slog.String("uri", e.URI()))
		switch err := s.EndElementNS(ctx, pctx, e.LocalName(), e.Prefix(), e.URI()); err {
		case nil, sax.ErrHandlerUnspecified:
			// no op
		default:
			return pctx.error(ctx, err)
		}
	}
	pctx.popNode(ctx)

	return nil
}

func (pctx *parserCtx) parseAttributeValue(ctx context.Context, normalize bool) (value string, entities int, err error) {
	_, err2 := pctx.parseQuotedText(ctx, func(qch rune) (string, error) {
		value, entities, err = pctx.parseAttributeValueInternal(ctx, qch, normalize)
		return "", nil
	})
	if err2 != nil {
		return "", 0, err2
	}
	return
}

// This is based on xmlParseAttValueComplex
func (pctx *parserCtx) parseAttributeValueInternal(ctx context.Context, qch rune, normalize bool) (value string, entities int, err error) {
	cur := pctx.getCursor(ctx)
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
				r, err = pctx.parseCharRef(ctx)
				if err != nil {
					err = pctx.error(ctx, err)
					return
				}

				if r == '&' && !pctx.replaceEntities {
					_, _ = b.WriteString("&#38;")
				} else {
					_, _ = b.WriteRune(r)
				}
			} else {
				var ent *Entity
				ent, err = pctx.parseEntityRef(ctx)
				if err != nil {
					err = pctx.error(ctx, err)
					return
				}

				if ent.entityType == InternalPredefinedEntity {
					if ent.content == "&" && !pctx.replaceEntities {
						_, _ = b.WriteString("&#38;")
					} else {
						_, _ = b.WriteString(ent.content)
					}
				} else if pctx.replaceEntities {
					var rep string
					rep, err = pctx.decodeEntities(ctx, ent.Content(), SubstituteRef)
					if err != nil {
						err = pctx.error(ctx, err)
						return
					}
					for i := 0; i < len(rep); i++ {
						switch rep[i] {
						case 0xD, 0xA, 0x9:
							_ = b.WriteByte(0x20)
						default:
							_ = b.WriteByte(rep[i])
						}
					}
				} else {
					_, _ = b.WriteString("&")
					_, _ = b.WriteString(ent.name)
					_, _ = b.WriteString(";")
				}
			}
		case 0x20, 0xD, 0xA, 0x9:
			if b.Len() > 0 || !normalize {
				if !normalize || !inSpace {
					b.WriteRune(0x20)
				}
				inSpace = true
			}
			if err := cur.Advance(1); err != nil {
				return "", 0, err
			}
		default:
			inSpace = false
			b.WriteRune(c)
			if err := cur.Advance(1); err != nil {
				return "", 0, err
			}
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

func (pctx *parserCtx) parseAttribute(ctx context.Context, elemName string) (local string, prefix string, value string, err error) {
	l, p, err := pctx.parseQName(ctx)
	if err != nil {
		err = pctx.error(ctx, err)
		return
	}

	normalize := false
	attType, ok := pctx.lookupSpecialAttribute(elemName, l)
	if ok && attType != AttrInvalid {
		normalize = true
	}
	pctx.skipBlanks(ctx)

	cur := pctx.getCursor(ctx)
	if cur == nil {
		panic("did not get rune cursor")
	}
	if cur.Peek() != '=' {
		err = pctx.error(ctx, ErrEqualSignRequired)
	}
	if err2 := cur.Advance(1); err2 != nil {
		err = err2
		return
	}
	pctx.skipBlanks(ctx)

	v, entities, err := pctx.parseAttributeValue(ctx, normalize)
	if err != nil {
		err = pctx.error(ctx, err)
		return
	}

	/*
	 * Sometimes a second normalisation pass for spaces is needed
	 * but that only happens if charrefs or entities refernces
	 * have been used in the attribute value, i.e. the attribute
	 * value have been extracted in an allocated string already.
	 */
	if normalize {
		if entities > 0 {
			v = pctx.attrNormalizeSpace(v)
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

func (pctx *parserCtx) skipBlanks(ctx context.Context) bool {
	i := 0
	cur := pctx.getCursor(ctx)
	if cur == nil {
		panic("did not get rune cursor")
	}
	for c := cur.PeekN(i + 1); isBlankCh(c) && !cur.Done(); c = cur.PeekN(i + 1) {
		i++
	}
	if i > 0 {
		if err := cur.Advance(i); err != nil {
			return false
		}

		if cur.Peek() == '%' {
			if err := pctx.handlePEReference(ctx); err != nil {
				return false
			}
		}
		return true
	}
	return false
}

func (pctx *parserCtx) skipBlankBytes(ctx context.Context, cur *strcursor.ByteCursor) bool {
	i := 0
	for c := cur.PeekN(i + 1); c != 0x0 && isBlankCh(rune(c)); c = cur.PeekN(i + 1) {
		i++
	}
	if i > 0 {
		if err := cur.Advance(i); err != nil {
			return false
		}

		if cur.Peek() == '%' {
			if err := pctx.handlePEReference(ctx); err != nil {
				return false
			}
		}
		return true
	}
	return false
}

// should only be here if current buffer is at '<?xml'
func (pctx *parserCtx) parseXMLDecl(ctx context.Context) error {
	cur := pctx.getByteCursor()
	if cur == nil {
		return ErrByteCursorRequired
	}

	if !cur.Consume(xmlDeclHint) {
		return pctx.error(ctx, ErrInvalidXMLDecl)
	}

	if !pctx.skipBlankBytes(ctx, cur) {
		return errors.New("blank needed after '<?xml'")
	}

	v, err := pctx.parseVersionInfo(ctx)
	if err != nil {
		return pctx.error(ctx, err)
	}
	pctx.version = v

	if !isBlankCh(rune(cur.Peek())) {
		// if the next character isn't blank, we expect the
		// end of XML decl, so return success
		if cur.Peek() == '?' && cur.PeekN(2) == '>' {
			if err := cur.Advance(2); err != nil {
				return err
			}
			return nil
		}

		// otherwise, we just saw something unexpected
		return pctx.error(ctx, ErrSpaceRequired)
	}

	// we *may* have encoding decl
	v, err = pctx.parseEncodingDecl(ctx)
	if err == nil {
		// ctx.encoding contains the explicit encoding specified
		pctx.encoding = v

		// if the encoding decl is found, then we *could* have
		// the end of the XML declaration
		if cur.Peek() == '?' && cur.PeekN(2) == '>' {
			if err := cur.Advance(2); err != nil {
				return err
			}
			return nil
		}
	} else if _, ok := err.(ErrAttrNotFound); ok {
		return pctx.error(ctx, err)
	}

	vb, err := pctx.parseStandaloneDecl(ctx)
	if err != nil {
		return err
	}
	pctx.standalone = vb

	if cur.Peek() == '?' && cur.PeekN(2) == '>' {
		if err := cur.Advance(2); err != nil {
			return err
		}
		return nil
	}
	return pctx.error(ctx, errors.New("XML declaration not closed"))
}

func (e ErrAttrNotFound) Error() string {
	return "attribute token '" + e.Token + "' not found"
}

/*
func (pctx *parserCtx) parseNamedAttribute(ctx context.Context, name string, cb qtextHandler) (string, error) {
	ctx.skipBlanks()

	cur := pctx.getCursor(ctx)
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

	if err := cur.Advance(1); err != nil {
		return "", err
	}
	ctx.skipBlanks()
	return ctx.parseQuotedText(cb)
}
*/

// parse the XML version info (version="1.0")
var versionBytes = []byte{'v', 'e', 'r', 's', 'i', 'o', 'n'}

func (pctx *parserCtx) parseVersionInfo(ctx context.Context) (string, error) {
	return pctx.parseNamedAttributeBytes(ctx, versionBytes, pctx.parseVersionNum)
}

func (pctx *parserCtx) parseNamedAttributeBytes(ctx context.Context, name []byte, valueParser qtextHandler) (string, error) {
	cur := pctx.getByteCursor()
	if cur == nil {
		return "", ErrByteCursorRequired
	}

	pctx.skipBlankBytes(ctx, cur)
	if !cur.Consume(name) {
		return "", pctx.error(ctx, ErrAttrNotFound{Token: string(name)})
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

/*
 * parse the XML version value.
 *
 * [26] VersionNum ::= '1.' [0-9]+
 *
 * In practice allow [0-9].[0-9]+ at that level
 *
 * Returns the string giving the XML version number
 */
func (pctx *parserCtx) parseVersionNum(_ rune) (string, error) {
	cur := pctx.getByteCursor()
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
			if err := cur.Advance(i - 1); err != nil {
				return "", err
			}
			return b.String(), nil
		}
	}
}

type qtextHandler func(qch rune) (string, error)

func (pctx *parserCtx) parseQuotedTextBytes(cb qtextHandler) (value string, err error) {
	cur := pctx.getByteCursor()
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
	if err := cur.Advance(1); err != nil {
		return "", err
	}

	return
}

func (pctx *parserCtx) parseQuotedText(ctx context.Context, cb qtextHandler) (value string, err error) {
	cur := pctx.getCursor(ctx)
	if cur == nil {
		panic("did not get rune cursor")
	}
	q := cur.Peek()
	switch q {
	case '"', '\'':
		if err := cur.Advance(1); err != nil {
			return "", err
		}
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
	if err := cur.Advance(1); err != nil {
		return "", err
	}

	return
}

var encodingBytes = []byte{'e', 'n', 'c', 'o', 'd', 'i', 'n', 'g'}

func (pctx *parserCtx) parseEncodingDecl(ctx context.Context) (string, error) {
	return pctx.parseNamedAttributeBytes(ctx, encodingBytes, pctx.parseEncodingName)
}

func (pctx *parserCtx) parseEncodingName(_ rune) (string, error) {
	cur := pctx.getByteCursor()
	if cur == nil {
		return "", ErrByteCursorRequired
	}
	c := cur.Peek()

	buf := bufferPool.Get().(*bytes.Buffer)
	defer releaseBuffer(buf)

	// first char needs to be alphabets
	if !(c >= 'a' && c <= 'z') && !(c >= 'A' && c <= 'Z') { // nolint:staticcheck
		return "", pctx.errorWithoutContext(ErrInvalidEncodingName)
	}
	_, _ = buf.WriteRune(c)

	i := 2
	for c = cur.PeekN(i); c != 0x0; c = cur.PeekN(i) {
		if !(c >= 'a' && c <= 'z') && !(c >= 'A' && c <= 'Z') && !(c >= '0' && c <= '9') && c != '.' && c != '_' && c != '-' { // nolint:staticcheck
			i--
			break
		}
		_, _ = buf.WriteRune(c)
		i++
	}

	if err := cur.Advance(i); err != nil {
		return "", err
	}

	return buf.String(), nil
}

var standaloneBytes = []byte{'s', 't', 'a', 'n', 'd', 'a', 'l', 'o', 'n', 'e'}

func (pctx *parserCtx) parseStandaloneDecl(ctx context.Context) (DocumentStandaloneType, error) {
	v, err := pctx.parseNamedAttributeBytes(ctx, standaloneBytes, pctx.parseStandaloneDeclValue)
	if err != nil {
		return StandaloneInvalidValue, err
	}
	if v == "yes" {
		return StandaloneExplicitYes, nil
	} else {
		return StandaloneExplicitNo, nil
	}
}

func (pctx *parserCtx) parseStandaloneDeclValue(_ rune) (string, error) {
	const (
		yes = "yes"
		no  = "no"
	)
	cur := pctx.getByteCursor()
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

func (pctx *parserCtx) parseMisc(ctx context.Context) error {
	ctx, span := StartSpan(ctx, "parseMisc")
	defer span.End()
	cur := pctx.getCursor(ctx)
	for !cur.Done() && pctx.instate != psEOF {
		if cur.HasPrefixString("<?") {
			if err := pctx.parsePI(ctx); err != nil {
				return pctx.error(ctx, err)
			}
		} else if cur.HasPrefixString("<!--") {
			if err := pctx.parseComment(ctx); err != nil {
				return pctx.error(ctx, err)
			}
		} else if isBlankCh(cur.Peek()) {
			pctx.skipBlanks(ctx)
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
	ctx, span := StartSpan(ctx, "parsePI")
	defer span.End()
	cur := pctx.getCursor(ctx)
	if cur == nil {
		panic("did not get rune cursor")
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
		if s := pctx.sax; s != nil {
			switch err := s.ProcessingInstruction(ctx, pctx.userData, target, ""); err {
			case nil, sax.ErrHandlerUnspecified:
				// no op
			default:
				return pctx.error(ctx, err)
			}
		}
		return nil
	}

	if !isBlankCh(cur.Peek()) {
		return pctx.error(ctx, ErrSpaceRequired)
	}

	pctx.skipBlanks(ctx)
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
		_, _ = buf.WriteRune(c)
		i++
	}

	if err := cur.Advance(i); err != nil {
		return err
	}
	data := buf.String()

	if !cur.ConsumeString("?>") {
		return pctx.error(ctx, ErrInvalidProcessingInstruction)
	}

	if s := pctx.sax; s != nil {
		switch err := s.ProcessingInstruction(ctx, pctx.userData, target, data); err {
		case nil, sax.ErrHandlerUnspecified:
			// no op
		default:
			return pctx.error(ctx, err)
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
func (pctx *parserCtx) parseName(ctx context.Context) (name string, err error) {
	if pctx.instate == psEOF {
		err = pctx.error(ctx, ErrPrematureEOF)
		return
	}

	cur := pctx.getCursor(ctx)
	if cur == nil {
		panic("did not get rune cursor")
	}

	buf := bufferPool.Get().(*bytes.Buffer)
	defer releaseBuffer(buf)

	// first letter
	c := cur.Peek()
	if c == ' ' || c == '>' || c == '/' || /* accelerators */ (!unicode.IsLetter(c) && c != '_' && c != ':') {
		err = pctx.error(ctx, fmt.Errorf("invalid first letter '%c'", c))
		return
	}
	_, _ = buf.WriteRune(c)

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
		_, _ = buf.WriteRune(c)

		i++
	}
	if i > MaxNameLength {
		err = pctx.error(ctx, ErrNameTooLong)
		return
	}

	if err := cur.Advance(i); err != nil {
		return "", err
	}
	name = buf.String()
	if name == "" {
		err = pctx.error(ctx, errors.New("internal error: parseName returned with empty name"))
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
func (pctx *parserCtx) parseQName(ctx context.Context) (local string, prefix string, err error) {
	cur := pctx.getCursor(ctx)
	if cur == nil {
		panic("did not get rune cursor")
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
	if err == nil {
		local = v
		return
	}

	v, err = pctx.parseNmtoken(ctx)
	if err == nil {
		local = v
		return
	}

	v, err = pctx.parseName(ctx)
	if err != nil {
		err = pctx.error(ctx, err)
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
func (pctx *parserCtx) parseNmtoken(ctx context.Context) (string, error) {
	i := 1
	cur := pctx.getCursor(ctx)
	if cur == nil {
		panic("did not get rune cursor")
	}
	buf := bufferPool.Get().(*bytes.Buffer)
	defer releaseBuffer(buf)

	for c := cur.PeekN(i); c != 0x0; i++ {
		if !isNameChar(c) {
			break
		}
		_, _ = buf.WriteRune(c)
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
func (pctx *parserCtx) parseNCName(ctx context.Context) (ncname string, err error) {
	if pctx.instate == psEOF {
		err = pctx.error(ctx, ErrPrematureEOF)
		return
	}

	cur := pctx.getCursor(ctx)
	if cur == nil {
		panic("did not get rune cursor")
	}
	buf := bufferPool.Get().(*bytes.Buffer)
	defer releaseBuffer(buf)

	var c rune
	if c = cur.Peek(); c == ' ' || c == '>' || c == '/' || !isNameStartChar(c) {
		err = pctx.error(ctx, errors.New("invalid name start char"))
		return
	}
	_, _ = buf.WriteRune(c)

	// at this point we have at least 1 character name.
	// see how much more we got here
	i := 2
	for c = cur.PeekN(i); c != 0x0; c = cur.PeekN(i) {
		if !unicode.IsLetter(c) && !unicode.IsDigit(c) && c != '_' && c != '-' && c != '.' {
			i--
			break
		}
		_, _ = buf.WriteRune(c)
		i++
	}
	if i > MaxNameLength {
		err = pctx.error(ctx, ErrNameTooLong)
		return
	}
	if err := cur.Advance(i); err != nil {
		return "", err
	}
	ncname = buf.String()
	return
}

func (pctx *parserCtx) parsePITarget(ctx context.Context) (string, error) {
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

// note: unlike libxml2, we can't differentiate between SAX handlers
// that uses the same IgnorableWhitespace and Character handlers
func (pctx *parserCtx) areBlanks(ctx context.Context, s string, blankChars bool) (ret bool) {
	// Check for xml:space value.
	if pctx.space == 1 || pctx.space == -2 {
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
	if pctx.peekNode() == nil {
		ret = false
		return
	}
	if pctx.doc != nil {
		ok, _ := pctx.doc.IsMixedElement(pctx.peekNode().Name())
		ret = !ok
		return
	}

	cur := pctx.getCursor(ctx)
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

func (pctx *parserCtx) parseCDSect(ctx context.Context) error {
	ctx, span := StartSpan(ctx, "parseCDSect")
	defer span.End()
	cur := pctx.getCursor(ctx)
	if cur == nil {
		panic("did not get rune cursor")
	}
	if !cur.ConsumeString("<![CDATA[") {
		return pctx.error(ctx, ErrInvalidCDSect)
	}

	pctx.instate = psCDATA
	defer func() { pctx.instate = psContent }()

	if err := pctx.parseCharData(ctx, true); err != nil {
		return pctx.error(ctx, err)
	}

	if !cur.ConsumeString("]]>") {
		return pctx.error(ctx, ErrCDATANotFinished)
	}
	return nil
}

func (pctx *parserCtx) parseComment(ctx context.Context) error {
	ctx, span := StartSpan(ctx, "parseComment")
	defer span.End()
	cur := pctx.getCursor(ctx)
	if cur == nil {
		panic("did not get rune cursor")
	}
	if !cur.ConsumeString("<!--") {
		return pctx.error(ctx, ErrInvalidComment)
	}

	buf := bufferPool.Get().(*bytes.Buffer)
	defer releaseBuffer(buf)

	i := 0
	q := cur.PeekN(i + 1)
	if !isChar(q) {
		return pctx.error(ctx, ErrInvalidChar)
	}
	i++
	buf.WriteRune(q)

	r := cur.PeekN(i + 1)
	if !isChar(r) {
		return pctx.error(ctx, ErrInvalidChar)
	}
	i++
	buf.WriteRune(r)

	for c := cur.PeekN(i + 1); isChar(c) && (q != '-' || r != '-' || c != '>'); c = cur.PeekN(i + 1) {
		if q == '-' && r == '-' {
			return pctx.error(ctx, ErrHyphenInComment)
		}
		_, _ = buf.WriteRune(c)
		q = r
		r = c
		i++
	}

	// -2 for "-->" (note: '>' has not been consumed, so we use -2 instead of -3
	buf.Truncate(buf.Len() - 2)
	str := buf.Bytes()
	// i+1 because '>' was not consumed in the loop
	if err := cur.Advance(i + 1); err != nil {
		return err
	}

	if sh := pctx.sax; sh != nil {
		str = bytes.ReplaceAll(str, []byte{'\r', '\n'}, []byte{'\n'})
		switch err := sh.Comment(ctx, pctx, str); err {
		case nil, sax.ErrHandlerUnspecified:
			// no op
		default:
			return pctx.error(ctx, err)
		}
	}

	return nil
}

func (pctx *parserCtx) parseDocTypeDecl(ctx context.Context) error {
	cur := pctx.getCursor(ctx)
	if cur == nil {
		panic("did not get rune cursor")
	}
	if !cur.ConsumeString("<!DOCTYPE") {
		return pctx.error(ctx, ErrInvalidDTD)
	}

	pctx.skipBlanks(ctx)

	name, err := pctx.parseName(ctx)
	if err != nil {
		return pctx.error(ctx, ErrDocTypeNameRequired)
	}
	pctx.intSubName = name

	pctx.skipBlanks(ctx)
	u, eid, err := pctx.parseExternalID()
	if err != nil {
		return pctx.error(ctx, err)
	}

	if u != "" || eid != "" {
		pctx.hasExternalSubset = true
	}
	pctx.extSubURI = u
	pctx.extSubSystem = eid

	pctx.skipBlanks(ctx)

	if s := pctx.sax; s != nil {
		switch err := s.InternalSubset(ctx, pctx.userData, name, eid, u); err {
		case nil, sax.ErrHandlerUnspecified:
			// no op
		default:
			return pctx.error(ctx, err)
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
		return pctx.error(ctx, ErrDocTypeNotFinished)
	}
	if err := cur.Advance(1); err != nil {
		return err
	}

	return nil
}

func (pctx *parserCtx) parseInternalSubset(ctx context.Context) error {
	ctx, span := StartSpan(ctx, "parseInternalSubset")
	defer span.End()
	// equiv: xmlParseInternalSubset (parser.c)
	cur := pctx.getCursor(ctx)
	if cur == nil {
		panic("did not get rune cursor")
	}
	if cur.Peek() != '[' {
		goto FinishDTD
	}
	pctx.instate = psDTD
	if err := cur.Advance(1); err != nil {
		return err
	}

	for {
		// Get current cursor in case parameter entity expansion changed the input
		cur = pctx.getCursor(ctx)
		if cur == nil || cur.Done() || cur.Peek() == ']' {
			break
		}

		pctx.skipBlanks(ctx)
		if err := pctx.parseMarkupDecl(ctx); err != nil {
			return pctx.error(ctx, err)
		}
		if err := pctx.parsePEReference(ctx); err != nil {
			return pctx.error(ctx, err)
		}
	}

	// Get final cursor state
	cur = pctx.getCursor(ctx)
	if cur != nil && cur.Peek() == ']' {
		if err := cur.Advance(1); err != nil {
			return err
		}
		pctx.skipBlanks(ctx)
	}

FinishDTD:
	// Ensure we have the current cursor
	cur = pctx.getCursor(ctx)
	if cur != nil && cur.Peek() != '>' {
		return pctx.error(ctx, ErrDocTypeNotFinished)
	}
	if cur != nil {
		if err := cur.Advance(1); err != nil {
			return err
		}
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
func (pctx *parserCtx) parseMarkupDecl(ctx context.Context) error {
	ctx, span := StartSpan(ctx, "parseMarkupDecl")
	defer span.End()
	cur := pctx.getCursor(ctx)
	if cur == nil {
		panic("did not get rune cursor")
	}
	if cur.Peek() == '<' {
		if cur.PeekN(2) == '!' {
			switch cur.PeekN(3) {
			case 'E':
				c := cur.PeekN(4)
				switch c {
				case 'L': // <!EL...
					if _, err := pctx.parseElementDecl(ctx); err != nil {
						return pctx.error(ctx, err)
					}
				case 'N': // <!EN....
					if err := pctx.parseEntityDecl(ctx); err != nil {
						return pctx.error(ctx, err)
					}
				}
			case 'A': // <!A...
				if err := pctx.parseAttributeListDecl(ctx); err != nil {
					return pctx.error(ctx, err)
				}
			case 'N': // <!N...
				if err := pctx.parseNotationDecl(); err != nil {
					return pctx.error(ctx, err)
				}
			case '-': // <!-...
				if err := pctx.parseComment(ctx); err != nil {
					return pctx.error(ctx, err)
				}
			default:
				// no op: error detected later?
			}
		} else if cur.PeekN(2) == '?' {
			return pctx.parsePI(ctx)
		}
	}

	if pctx.instate == psEOF {
		return nil
	}

	// This is only for internal subset. On external entities,
	// the replacement is done before parsing stage
	if !pctx.external && pctx.inputTab.Len() == 1 {
		if err := pctx.parsePEReference(ctx); err != nil {
			return pctx.error(ctx, err)
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
	pctx.instate = psDTD

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
func (pctx *parserCtx) parsePEReference(ctx context.Context) error {
	ctx, span := StartSpan(ctx, "parsePEReference")
	defer span.End()
	cur := pctx.getCursor(ctx)
	if cur == nil {
		panic("did not get rune cursor")
	}
	if cur.Peek() != '%' {
		// This is not an error. just be done
		return nil
	}
	if err := cur.Advance(1); err != nil {
		return err
	}

	name, err := pctx.parseName(ctx)
	if err != nil {
		return pctx.error(ctx, err)
	}

	if cur.Peek() != ';' {
		return pctx.error(ctx, ErrSemicolonRequired)
	}
	if err := cur.Advance(1); err != nil {
		return err
	}

	/*
		ctx.nbentities++ // number of entities parsed
	*/
	var entity sax.Entity
	if s := pctx.sax; s != nil {
		_ = pctx.fireSAXCallback(ctx, cbGetParameterEntity, &entity, name)
	}

	// XXX Why check here?
	if pctx.instate == psEOF {
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
		if pctx.standalone == StandaloneExplicitYes || (!pctx.hasExternalSubset && !pctx.hasPERefs) {
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
		pctx.valid = false
		if err := pctx.entityCheck(entity, 0, 0); err != nil {
			return pctx.error(ctx, err)
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

			// Decode character references and other entities in the parameter entity content
			decodedContent, err := pctx.decodeEntities(ctx, entity.Content(), SubstituteBoth)
			if err != nil {
				return fmt.Errorf("failed to decode parameter entity content: %v", err)
			}

			// Push the decoded content as new input stream
			pctx.pushInput(ctx, strcursor.NewByteCursor(bytes.NewReader([]byte(decodedContent))))

			// Note: External parameter entities may need text declaration parsing
			// but for now we only handle internal parameter entities
		}
	}
	pctx.hasPERefs = true
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
func (pctx *parserCtx) parseElementDecl(ctx context.Context) (ElementTypeVal, error) {
	cur := pctx.getCursor(ctx)
	if cur == nil {
		panic("did not get rune cursor")
	}
	if !cur.ConsumeString("<!ELEMENT") {
		return UndefinedElementType, pctx.error(ctx, ErrInvalidElementDecl)
	}

	if !isBlankCh(cur.Peek()) {
		return UndefinedElementType, pctx.error(ctx, ErrSpaceRequired)
	}
	pctx.skipBlanks(ctx)

	name, err := pctx.parseName(ctx)
	if err != nil {
		return UndefinedElementType, pctx.error(ctx, err)
	}

	/* XXX WHAT?
	   while ((RAW == 0) && (ctxt->inputNr > 1))
	       xmlPopInput(ctxt);
	*/

	if !isBlankCh(cur.Peek()) {
		return UndefinedElementType, pctx.error(ctx, ErrSpaceRequired)
	}
	pctx.skipBlanks(ctx)

	var etype ElementTypeVal
	var content *ElementContent
	if cur.ConsumeString("EMPTY") {
		etype = EmptyElementType
	} else if cur.ConsumeString("ANY") {
		etype = AnyElementType
	} else if cur.Peek() == '(' {
		content, etype, err = pctx.parseElementContentDecl(ctx)
		if err != nil {
			return UndefinedElementType, pctx.error(ctx, err)
		}
		/*
			} else {
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

	pctx.skipBlanks(ctx)

	/*
	 * Pop-up of finished entities.
	 */
	/*
	   while ((RAW == 0) && (ctxt->inputNr > 1))
	       xmlPopInput(ctxt);
	   SKIP_BLANKS;
	*/

	if cur.Peek() != '>' {
		return UndefinedElementType, pctx.error(ctx, ErrGtRequired)
	}
	if err := cur.Advance(1); err != nil {
		return UndefinedElementType, err
	}

	/*
	           if (input != ctxt->input) {
	               xmlFatalErrMsg(ctxt, XML_ERR_ENTITY_BOUNDARY,
	   "Element declaration doesn't start and stop in the same entity\n");
	           }
	*/

	if s := pctx.sax; s != nil {
		switch err := s.ElementDecl(ctx, pctx.userData, name, int(etype), content); err {
		case nil, sax.ErrHandlerUnspecified:
			// no op
		default:
			return UndefinedElementType, pctx.error(ctx, err)
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

func (pctx *parserCtx) parseElementContentDecl(ctx context.Context) (*ElementContent, ElementTypeVal, error) {
	cur := pctx.getCursor(ctx)
	if cur == nil {
		panic("did not get rune cursor")
	}
	if cur.Peek() != '(' {
		return nil, UndefinedElementType, pctx.error(ctx, ErrOpenParenRequired)
	}
	if err := cur.Advance(1); err != nil {
		return nil, UndefinedElementType, err
	}

	if pctx.instate == psEOF {
		return nil, UndefinedElementType, pctx.error(ctx, ErrEOF)
	}

	pctx.skipBlanks(ctx)

	var ec *ElementContent
	var err error
	var etype ElementTypeVal
	if cur.HasPrefixString("#PCDATA") {
		ec, err = pctx.parseElementMixedContentDecl(ctx)
		if err != nil {
			return nil, UndefinedElementType, pctx.error(ctx, err)
		}
		etype = MixedElementType
	} else {
		ec, err = pctx.parseElementChildrenContentDeclPriv(ctx, 0)
		if err != nil {
			return nil, UndefinedElementType, pctx.error(ctx, err)
		}
		etype = ElementElementType
	}

	pctx.skipBlanks(ctx)
	return ec, etype, nil
}

func (pctx *parserCtx) parseElementMixedContentDecl(ctx context.Context) (*ElementContent, error) {
	cur := pctx.getCursor(ctx)
	if cur == nil {
		panic("did not get rune cursor")
	}
	if !cur.ConsumeString("#PCDATA") {
		return nil, pctx.error(ctx, ErrPCDATARequired)
	}

	if cur.Peek() == ')' {
		/*
		               if ((ctxt->validate) && (ctxt->input->id != inputchk)) {
		                   xmlValidityError(ctxt, XML_ERR_ENTITY_BOUNDARY,
		   "Element content declaration doesn't start and stop in the same entity\n",
		                                    NULL, NULL);
		               }
		*/
		if err := cur.Advance(1); err != nil {
			return nil, pctx.error(ctx, err)
		}
		ret, err := pctx.doc.CreateElementContent("", ElementContentPCDATA)
		if err != nil {
			return nil, pctx.error(ctx, err)
		}

		if cur.Peek() == '*' {
			ret.coccur = ElementContentMult
			if err := cur.Advance(1); err != nil {
				return nil, err
			}
		}

		return ret, nil
	}

	var err error
	var retelem *ElementContent
	var curelem *ElementContent
	if c := cur.Peek(); c == '(' || c == '|' {
		retelem, err = pctx.doc.CreateElementContent("", ElementContentPCDATA)
		if err != nil {
			return nil, pctx.error(ctx, err)
		}
		curelem = retelem
	}

	var elem string
	for cur.Peek() == '|' {
		if err := cur.Advance(1); err != nil {
			return nil, err
		}
		if elem == "" {
			retelem, err = pctx.doc.CreateElementContent("", ElementContentOr)
			if err != nil {
				return nil, pctx.error(ctx, err)
			}

			retelem.c1 = curelem
			if curelem != nil {
				curelem.parent = retelem
			}
			curelem = retelem
		} else {
			n, err := pctx.doc.CreateElementContent("", ElementContentOr)
			if err != nil {
				return nil, pctx.error(ctx, err)
			}
			n.c1, err = pctx.doc.CreateElementContent("", ElementContentElement)
			if err != nil {
				return nil, pctx.error(ctx, err)
			}
			n.c1.parent = n
			curelem.c2 = n
			n.parent = curelem
			curelem = n
		}
		pctx.skipBlanks(ctx)
		elem, err = pctx.parseName(ctx)
		if err != nil {
			return nil, pctx.error(ctx, err)
		}
		pctx.skipBlanks(ctx)
	}
	if cur.Peek() == ')' && cur.PeekN(2) == '*' {
		if err := cur.Advance(2); err != nil {
			return nil, err
		}
		if elem != "" {
			curelem.c2, err = pctx.doc.CreateElementContent(elem, ElementContentElement)
			if err != nil {
				return nil, pctx.error(ctx, err)
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
func (pctx *parserCtx) parseElementChildrenContentDeclPriv(ctx context.Context, depth int) (*ElementContent, error) {
	if depth > 128 { // XML_PARSE_HUGE -> 2048
		return nil, fmt.Errorf("xmlParseElementChildrenContentDecl : depth %d too deep", depth)
	}

	var curelem *ElementContent
	var retelem *ElementContent
	pctx.skipBlanks(ctx)
	cur := pctx.getCursor(ctx)
	if cur == nil {
		panic("did not get rune cursor")
	}
	if cur.Peek() == '(' {
		if err := cur.Advance(1); err != nil {
			return nil, err
		}
		pctx.skipBlanks(ctx)
		retelem, err := pctx.parseElementChildrenContentDeclPriv(ctx, depth+1)
		if err != nil {
			return nil, pctx.error(ctx, err)
		}
		curelem = retelem
		pctx.skipBlanks(ctx)
	} else {
		elem, err := pctx.parseName(ctx)
		if err != nil {
			return nil, pctx.error(ctx, err)
		}

		retelem, err = pctx.doc.CreateElementContent(elem, ElementContentElement)
		if err != nil {
			return nil, pctx.error(ctx, err)
		}
		curelem = retelem

		switch cur.Peek() {
		case '?':
			curelem.coccur = ElementContentOpt
			if err := cur.Advance(1); err != nil {
				return nil, err
			}
		case '*':
			curelem.coccur = ElementContentMult
			if err := cur.Advance(1); err != nil {
				return nil, err
			}
		case '+':
			curelem.coccur = ElementContentPlus
			if err := cur.Advance(1); err != nil {
				return nil, err
			}
		}
	}

	pctx.skipBlanks(ctx)

	// XXX closures aren't the most efficient thing golang has to offer,
	// but I really don't want to write the same code twice...
	var sep rune
	var last *ElementContent
	createElementContent := func(c rune, typ ElementContentType) error {
		// Detect "Name | Name, Name"
		if sep == 0x0 {
			sep = c
		} else if sep != c {
			return pctx.error(ctx, fmt.Errorf("'%c' expected", sep))
		}
		if err := cur.Advance(1); err != nil {
			return err
		}

		op, err := pctx.doc.CreateElementContent("", typ)
		if err != nil {
			return pctx.error(ctx, err)
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
				return nil, pctx.error(ctx, err)
			}
		case '|':
			if err := createElementContent(c, ElementContentOr); err != nil {
				return nil, pctx.error(ctx, err)
			}
		default:
			return nil, pctx.error(ctx, ErrElementContentNotFinished)
		}

		pctx.skipBlanks(ctx)

		if cur.Peek() == '(' {
			if err := cur.Advance(1); err != nil {
				return nil, err
			}
			pctx.skipBlanks(ctx)
			// recurse
			var err error
			last, err = pctx.parseElementChildrenContentDeclPriv(ctx, depth+1)
			if err != nil {
				return nil, pctx.error(ctx, err)
			}
			pctx.skipBlanks(ctx)
		} else {
			elem, err := pctx.parseName(ctx)
			if err != nil {
				return nil, pctx.error(ctx, err)
			}

			last, err = pctx.doc.CreateElementContent(elem, ElementContentElement)
			if err != nil {
				return nil, pctx.error(ctx, err)
			}

			switch cur.Peek() {
			case '?':
				last.coccur = ElementContentOpt
				if err := cur.Advance(1); err != nil {
					return nil, err
				}
			case '*':
				last.coccur = ElementContentMult
				if err := cur.Advance(1); err != nil {
					return nil, err
				}
			case '+':
				last.coccur = ElementContentPlus
				if err := cur.Advance(1); err != nil {
					return nil, err
				}
			}
		}
		pctx.skipBlanks(ctx)
	}
	if last != nil {
		curelem.c2 = last
		last.parent = curelem
	}
	if err := cur.Advance(1); err != nil {
		return nil, pctx.error(ctx, err)
	}
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
		if err := cur.Advance(1); err != nil {
			return nil, err
		}
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

func (pctx *parserCtx) parseEntityValueInternal(ctx context.Context, qch rune) (string, error) {
	/*
	 * NOTE: 4.4.5 Included in Literal
	 * When a parameter entity reference appears in a literal entity
	 * value, ... a single or double quote character in the replacement
	 * text is always treated as a normal data character and will not
	 * terminate the literal.
	 * In practice it means we stop the loop only when back at parsing
	 * the initial entity and the quote is found
	 */
	cur := pctx.getCursor(ctx)
	if cur == nil {
		panic("did not get rune cursor")
	}
	buf := bufferPool.Get().(*bytes.Buffer)
	defer releaseBuffer(buf)

	i := 0
	for c := cur.PeekN(i + 1); isChar(c) && c != qch; c = cur.PeekN(i + 1) {
		_, _ = buf.WriteRune(c)
		i++
	}
	if i > 0 {
		if err := cur.Advance(i); err != nil {
			return "", pctx.error(ctx, err)
		}
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
func (pctx *parserCtx) decodeEntities(ctx context.Context, s []byte, what SubstitutionType) (ret string, err error) {
	ret, err = pctx.decodeEntitiesInternal(ctx, s, what, 0)
	return
}

func (pctx *parserCtx) decodeEntitiesInternal(ctx context.Context, s []byte, what SubstitutionType, depth int) (string, error) {
	if depth > 40 {
		return "", errors.New("entity loop (depth > 40)")
	}

	out := bufferPool.Get().(*bytes.Buffer)
	defer releaseBuffer(out)

	for len(s) > 0 {
		if bytes.HasPrefix(s, []byte{'&', '#'}) {
			val, width, err := parseStringCharRef(s)
			if err != nil {
				return "", err
			}
			out.WriteRune(val)
			s = s[width:] // advance
		} else if s[0] == '&' && what&SubstituteRef == SubstituteRef {
			ent, width, err := pctx.parseStringEntityRef(ctx, s)
			if err != nil {
				return "", err
			}
			if err := pctx.entityCheck(ent, 0, 0); err != nil {
				return "", err
			}

			if EntityType(ent.EntityType()) == InternalPredefinedEntity {
				if len(ent.Content()) == 0 {
					return "", errors.New("predefined entity has no content")
				}
				_, _ = out.Write(ent.Content())
			} else if len(ent.Content()) != 0 {
				rep, err := pctx.decodeEntitiesInternal(ctx, ent.Content(), what, depth+1)
				if err != nil {
					return "", err
				}

				_, _ = out.WriteString(rep)
			} else {
				_, _ = out.WriteString(ent.Name())
			}
			s = s[width:]
		} else if s[0] == '%' && what&SubstitutePERef == SubstitutePERef {
			ent, width, err := pctx.parseStringPEReference(ctx, s)
			if err != nil {
				return "", err
			}
			if err := pctx.entityCheck(ent, width, 0); err != nil {
				return "", err
			}
			rep, err := pctx.decodeEntitiesInternal(ctx, ent.Content(), what, depth+1)
			if err != nil {
				return "", err
			}
			_, _ = out.WriteString(rep)
			s = s[width:]
		} else {
			_ = out.WriteByte(s[0])
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
func (pctx *parserCtx) parseEntityValue(ctx context.Context) (string, string, error) {
	pctx.instate = psEntityValue

	literal, err := pctx.parseQuotedText(ctx, func(qch rune) (string, error) {
		return pctx.parseEntityValueInternal(ctx, qch)
	})
	if err != nil {
		return "", "", pctx.error(ctx, err)
	}

	val, err := pctx.decodeEntities(ctx, []byte(literal), SubstitutePERef)
	if err != nil {
		return "", "", pctx.error(ctx, err)
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
func (pctx *parserCtx) parseEntityDecl(ctx context.Context) error {
	cur := pctx.getCursor(ctx)
	if cur == nil {
		panic("did not get rune cursor")
	}
	if !cur.ConsumeString("<!ENTITY") {
		return pctx.error(ctx, errors.New("<!ENTITY not started"))
	}

	if !pctx.skipBlanks(ctx) {
		return pctx.error(ctx, ErrSpaceRequired)
	}

	isParameter := false
	if cur.Peek() == '%' {
		if err := cur.Advance(1); err != nil {
			return err
		}
		if !pctx.skipBlanks(ctx) {
			return pctx.error(ctx, ErrSpaceRequired)
		}
		isParameter = true
	}

	name, err := pctx.parseName(ctx)
	if err != nil {
		return pctx.error(ctx, err)
	}
	if strings.IndexByte(name, ':') > -1 {
		return pctx.error(ctx, errors.New("colons are forbidden from entity names"))
	}

	if !pctx.skipBlanks(ctx) {
		return pctx.error(ctx, ErrSpaceRequired)
	}

	pctx.instate = psEntityDecl
	var literal string
	var value string
	var uri string

	if isParameter {

		if c := cur.Peek(); c == '"' || c == '\'' {
			literal, value, err = pctx.parseEntityValue(ctx)

			if err == nil {
				switch err := pctx.fireSAXCallback(ctx, cbEntityDecl, name, value); err {
				case nil, sax.ErrHandlerUnspecified:
					// no op
				default:
					return pctx.error(ctx, err)
				}
			}
		} else {
			literal, uri, err = pctx.parseExternalID()
			if err != nil {
				return pctx.error(ctx, ErrValueRequired)
			}

			if uri != "" {
				u, err := url.Parse(uri)
				if err != nil {
					return pctx.error(ctx, err)
				}

				if u.Fragment != "" {
					return pctx.error(ctx, errors.New("err uri fragment"))
				} else {
					if s := pctx.sax; s != nil {
						switch err := s.EntityDecl(ctx, pctx.userData, name, int(ExternalParameterEntity), literal, uri, ""); err {
						case nil, sax.ErrHandlerUnspecified:
							// no op
						default:
							return pctx.error(ctx, err)
						}
					}
				}
			}
		}
	} else {
		if c := cur.Peek(); c == '"' || c == '\'' {
			literal, value, err = pctx.parseEntityValue(ctx)
			if err == nil {
				if s := pctx.sax; s != nil {
					switch err := s.EntityDecl(ctx, pctx.userData, name, int(InternalGeneralEntity), "", "", value); err {
					case nil, sax.ErrHandlerUnspecified:
						// no op
					default:
						return pctx.error(ctx, err)
					}
				}
			}
		} else {
			literal, uri, err = pctx.parseExternalID()
			if err != nil {
				return pctx.error(ctx, ErrValueRequired)
			}

			if uri != "" {
				u, err := url.Parse(uri)
				if err != nil {
					return pctx.error(ctx, err)
				}

				if u.Fragment != "" {
					return pctx.error(ctx, errors.New("err uri fragment"))
				} else {
					if s := pctx.sax; s != nil {
						switch err := s.EntityDecl(ctx, pctx.userData, name, int(ExternalGeneralParsedEntity), literal, uri, ""); err {
						case nil, sax.ErrHandlerUnspecified:
							// no op
						default:
							return pctx.error(ctx, err)
						}
					}
				}
			}

			if c := cur.Peek(); c != '>' && !isBlankCh(c) {
				return pctx.error(ctx, ErrSpaceRequired)
			}

			pctx.skipBlanks(ctx)
			if cur.ConsumeString("NDATA") {
				if !pctx.skipBlanks(ctx) {
					return pctx.error(ctx, ErrSpaceRequired)
				}

				ndata, err := pctx.parseName(ctx)
				if err != nil {
					return pctx.error(ctx, err)
				}
				if s := pctx.sax; s != nil {
					switch err := s.EntityDecl(ctx, pctx.userData, name, int(ExternalParameterEntity), literal, uri, ndata); err {
					case nil, sax.ErrHandlerUnspecified:
						// no op
					default:
						return pctx.error(ctx, err)
					}
				}
			} else {
				if s := pctx.sax; s != nil {
					switch err := s.EntityDecl(ctx, pctx.userData, name, int(ExternalParameterEntity), literal, uri, ""); err {
					case nil, sax.ErrHandlerUnspecified:
						// no op
					default:
						return pctx.error(ctx, err)
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

	pctx.skipBlanks(ctx)
	if cur.Peek() != '>' {
		return pctx.error(ctx, errors.New("entity not terminated"))
	}
	if err := cur.Advance(1); err != nil {
		return err
	}

	// Ugly mechanism to save the raw entity value.
	// Note: This happens because the SAX interface doesn't have a way to
	// pass this non-standard information to the handler
	var curent sax.Entity
	if isParameter {
		if s := pctx.sax; s != nil {
			curent, _ = s.GetParameterEntity(ctx, pctx.userData, name)
		}
	} else {
		if s := pctx.sax; s != nil {
			curent, _ = s.GetEntity(ctx, pctx.userData, name)
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
func (pctx *parserCtx) parseNotationType(ctx context.Context) (Enumeration, error) {
	cur := pctx.getCursor(ctx)
	if cur == nil {
		panic("did not get rune cursor")
	}
	if cur.Peek() != '(' {
		return nil, pctx.error(ctx, ErrNotationNotStarted)
	}
	if err := cur.Advance(1); err != nil {
		return nil, pctx.error(ctx, err)
	}
	pctx.skipBlanks(ctx)

	names := map[string]struct{}{}

	var enum Enumeration
	for pctx.instate != psEOF {
		name, err := pctx.parseName(ctx)
		if err != nil {
			return nil, pctx.error(ctx, ErrNotationNameRequired)
		}
		if _, ok := names[name]; ok {
			return nil, pctx.error(ctx, ErrDTDDupToken{Name: name})
		}

		enum = append(enum, name)
		pctx.skipBlanks(ctx)

		if cur.Peek() != '|' {
			break
		}
		if err := cur.Advance(1); err != nil {
			return nil, pctx.error(ctx, err)
		}
		pctx.skipBlanks(ctx)
	}

	if cur.Peek() != ')' {
		return nil, pctx.error(ctx, ErrNotationNotFinished)
	}
	if err := cur.Advance(1); err != nil {
		return nil, pctx.error(ctx, err)
	}
	return enum, nil
}

func (pctx *parserCtx) parseEnumerationType(ctx context.Context) (Enumeration, error) {
	cur := pctx.getCursor(ctx)
	if cur == nil {
		panic("did not get rune cursor")
	}
	if cur.Peek() != '(' {
		return nil, pctx.error(ctx, ErrAttrListNotStarted)
	}
	if err := cur.Advance(1); err != nil {
		return nil, pctx.error(ctx, err)
	}
	pctx.skipBlanks(ctx)

	names := map[string]struct{}{}

	var enum Enumeration
	for pctx.instate != psEOF {
		name, err := pctx.parseNmtoken(ctx)
		if err != nil {
			return nil, pctx.error(ctx, ErrNmtokenRequired)
		}
		if _, ok := names[name]; ok {
			return nil, pctx.error(ctx, ErrDTDDupToken{Name: name})
		}

		enum = append(enum, name)
		pctx.skipBlanks(ctx)

		if cur.Peek() != '|' {
			break
		}
		if err := cur.Advance(1); err != nil {
			return nil, pctx.error(ctx, err)
		}
		pctx.skipBlanks(ctx)
	}

	if cur.Peek() != ')' {
		return nil, pctx.error(ctx, ErrAttrListNotFinished)
	}
	if err := cur.Advance(1); err != nil {
		return nil, pctx.error(ctx, err)
	}
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
func (pctx *parserCtx) parseEnumeratedType(ctx context.Context) (AttributeType, Enumeration, error) {
	cur := pctx.getCursor(ctx)
	if cur == nil {
		panic("did not get rune cursor")
	}
	if cur.ConsumeString("NOTATION") {
		if !isBlankCh(cur.Peek()) {
			return AttrInvalid, nil, pctx.error(ctx, ErrSpaceRequired)
		}
		pctx.skipBlanks(ctx)
		enum, err := pctx.parseNotationType(ctx)
		if err != nil {
			return AttrInvalid, nil, pctx.error(ctx, err)
		}

		return AttrNotation, enum, nil
	}

	enum, err := pctx.parseEnumerationType(ctx)
	if err != nil {
		return AttrInvalid, enum, pctx.error(ctx, err)
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
func (pctx *parserCtx) parseAttributeType(ctx context.Context) (AttributeType, Enumeration, error) {
	cur := pctx.getCursor(ctx)
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

	return pctx.parseEnumeratedType(ctx)
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
func (pctx *parserCtx) parseDefaultDecl(ctx context.Context) (deftype AttributeDefault, defvalue string, err error) {
	deftype = AttrDefaultNone
	cur := pctx.getCursor(ctx)
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
			err = pctx.error(ctx, ErrSpaceRequired)
			return
		}
		pctx.skipBlanks(ctx)
	}

	// XXX does AttValue always have a quote around it?
	defvalue, err = pctx.parseQuotedText(ctx, func(qch rune) (string, error) {
		s, _, err := pctx.parseAttributeValueInternal(ctx, qch, false)
		return s, err
	})
	if err != nil {
		deftype = AttrDefaultInvalid
		err = pctx.error(ctx, err)
		return
	}
	pctx.instate = psDTD
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
func (pctx *parserCtx) attrNormalizeSpace(s string) (value string) {
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
func (pctx *parserCtx) cleanSpecialAttributes() {
	for k, v := range pctx.attsSpecial {
		if v == AttrCDATA {
			delete(pctx.attsSpecial, k)
		}
	}
}

func (pctx *parserCtx) addSpecialAttribute(elemName, attrName string, typ AttributeType) {
	key := elemName + ":" + attrName
	pctx.attsSpecial[key] = typ
}

func (pctx *parserCtx) lookupSpecialAttribute(elemName, attrName string) (AttributeType, bool) {
	key := elemName + ":" + attrName
	v, ok := pctx.attsSpecial[key]
	return v, ok
}

func validateAttributeValueInternal(doc *Document, typ AttributeType, defvalue string) error {
	return nil
}

func (pctx *parserCtx) addAttributeDecl(dtd *DTD, elem string, name string, prefix string, atype AttributeType, def AttributeDefault, defvalue string, tree Enumeration) (attr *AttributeDecl, err error) {
	if dtd == nil {
		err = errors.New("dtd required")
		return
	}
	if name == "" {
		err = errors.New("name required")
		return
	}
	if elem == "" {
		err = errors.New("element required")
		return
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
			pctx.valid = false
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

	if err := dtd.AddChild(attr); err != nil {
		return nil, err
	}
	return attr, nil
}

func (pctx *parserCtx) addAttributeDefault(elemName, attrName, defaultValue string) {
	// detect attribute redefinition
	if _, ok := pctx.lookupSpecialAttribute(elemName, attrName); ok {
		return
	}

	// XXX seems like when your language has a map, you can do just
	// kinda do away with a bunch of stuff..  See xmlAddDefAttrs for
	// details of what the original code is doing
	m, ok := pctx.attsDefault[elemName]
	if !ok {
		m = map[string]*Attribute{}
		pctx.attsDefault[elemName] = m
	}

	var prefix string
	var local string
	if i := strings.IndexByte(attrName, ':'); i > -1 {
		prefix = attrName[:i]
		local = attrName[i+1:]
	} else {
		local = attrName
	}

	uri := pctx.lookupNamespace(context.Background(), prefix)
	attr, err := pctx.doc.CreateAttribute(local, defaultValue, newNamespace(prefix, uri))
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

func (pctx *parserCtx) lookupAttributeDefault(elemName string) (map[string]*Attribute, bool) {
	v, ok := pctx.attsDefault[elemName]
	return v, ok
}

/*
 * : parse the Attribute list def for an element
 *
 * [52] AttlistDecl ::= '<!ATTLIST' S Name AttDef* S? '>'
 *
 * [53] AttDef ::= S Name S AttType S DefaultDecl
 */
func (pctx *parserCtx) parseAttributeListDecl(ctx context.Context) error {
	cur := pctx.getCursor(ctx)
	if cur == nil {
		panic("did not get rune cursor")
	}
	if !cur.ConsumeString("<!ATTLIST") {
		return nil
	}

	if !isBlankCh(cur.Peek()) {
		return pctx.error(ctx, ErrSpaceRequired)
	}
	pctx.skipBlanks(ctx)

	elemName, err := pctx.parseName(ctx)
	if err != nil {
		return pctx.error(ctx, err)
	}
	pctx.skipBlanks(ctx)

	for cur.Peek() != '>' && pctx.instate != psEOF {
		attrName, err := pctx.parseName(ctx)
		if err != nil {
			return pctx.error(ctx, ErrAttributeNameRequired)
		}
		if !isBlankCh(cur.Peek()) {
			return pctx.error(ctx, ErrSpaceRequired)
		}
		pctx.skipBlanks(ctx)

		typ, enum, err := pctx.parseAttributeType(ctx)
		if err != nil {
			return pctx.error(ctx, err)
		}

		if !isBlankCh(cur.Peek()) {
			return pctx.error(ctx, ErrSpaceRequired)
		}
		pctx.skipBlanks(ctx)

		def, defvalue, err := pctx.parseDefaultDecl(ctx)
		if err != nil {
			return pctx.error(ctx, err)
		}

		if typ != AttrCDATA && def != AttrDefaultInvalid {
			defvalue = pctx.attrNormalizeSpace(defvalue)
		}

		if c := cur.Peek(); c != '>' {
			if !isBlankCh(c) {
				return pctx.error(ctx, ErrSpaceRequired)
			}
			pctx.skipBlanks(ctx)
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
		if s := pctx.sax; s != nil {
			switch err := s.AttributeDecl(ctx, pctx.userData, elemName, attrName, int(typ), int(def), defvalue, enum); err {
			case nil, sax.ErrHandlerUnspecified:
				// no op
			default:
				return pctx.error(ctx, err)
			}
		}

		if defvalue != "" && def != AttrDefaultImplied && def != AttrDefaultRequired {
			pctx.addAttributeDefault(elemName, attrName, defvalue)
		}

		// note: in libxml2, this is only triggered when SAX2 is enabled.
		// as we only support SAX2, we just register it regardless
		pctx.addSpecialAttribute(elemName, attrName, typ)

		if cur.Peek() == '>' {
			/*
			           if (input != ctxt->input) {
			               xmlValidityError(ctxt, XML_ERR_ENTITY_BOUNDARY,
			   "Attribute list declaration doesn't start and stop in the same entity\n",
			                                NULL, NULL);
			           }
			*/
			if err := cur.Advance(1); err != nil {
				return err
			}
			break
		}
	}
	return nil
}

func (pctx *parserCtx) parseNotationDecl() error {
	return nil
}

func (pctx *parserCtx) parseExternalID() (string, string, error) {
	return "", "", nil
}

func (pctx *parserCtx) parseExternalEntityPrivate(uri, externalID string) (Node, error) {
	return nil, errors.New("unimplemented")
}

var ErrParseSucceeded = errors.New("parse succeeded")

func (pctx *parserCtx) parseBalancedChunkInternal(ctx context.Context, chunk []byte, userData interface{}) (Node, error) {
	pctx.depth++
	defer func() { pctx.depth-- }()

	if pctx.depth > 40 {
		return nil, errors.New("entity loop")
	}

	newctx := &parserCtx{}
	if err := newctx.init(ctx, nil, bytes.NewReader(chunk)); err != nil {
		return nil, err
	}
	defer func() {
		if err := newctx.release(); err != nil {
			// Error ignored
		}
	}()

	if userData != nil {
		newctx.userData = userData
	} else {
		newctx.userData = newctx
	}

	if pctx.doc == nil {
		pctx.doc = NewDocument("1.0", "", StandaloneExplicitNo)
	}

	// save the document's children
	fc := pctx.doc.FirstChild()
	lc := pctx.doc.LastChild()
	pctx.doc.setFirstChild(nil)
	pctx.doc.setLastChild(nil)
	defer func() {
		pctx.doc.setFirstChild(fc)
		pctx.doc.setLastChild(lc)
	}()
	newctx.doc = pctx.doc
	newctx.sax = pctx.sax
	newctx.attsDefault = pctx.attsDefault
	newctx.depth = pctx.depth + 1

	// create a dummy node
	newRoot, err := newctx.doc.CreateElement("pseudoroot")
	if err != nil {
		return nil, pctx.error(ctx, err)
	}
	newctx.pushNode(context.Background(), newRoot)
	newctx.elem = newRoot // Set the current element context
	if err := newctx.doc.AddChild(newRoot); err != nil {
		return nil, err
	}
	if err := newctx.switchEncoding(context.Background()); err != nil {
		return nil, err
	}
	if err := newctx.parseContent(context.Background()); err != nil {
		return nil, err
	}

	if child := newctx.doc.FirstChild(); child != nil {
		if grandchild := child.FirstChild(); grandchild != nil {
			for e := grandchild; e != nil; e = e.NextSibling() {
				e.SetTreeDoc(pctx.doc)
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
func (pctx *parserCtx) parseReference(ctx context.Context) error {
	ctx, span := StartSpan(ctx, "parseReference")
	defer span.End()
	cur := pctx.getCursor(ctx)
	if cur == nil {
		panic("did not get rune cursor")
	}
	if cur.Peek() != '&' {
		return pctx.error(ctx, ErrAmpersandRequired)
	}

	// "&#..." CharRef
	if cur.PeekN(2) == '#' {
		v, err := pctx.parseCharRef(ctx)
		if err != nil {
			return pctx.error(ctx, err)
		}
		l := utf8.RuneLen(v)
		b := make([]byte, l)
		utf8.EncodeRune(b, v)
		if s := pctx.sax; s != nil {
			TraceEvent(ctx, "calling SAX Characters",
				slog.String("content_type", "character_reference"),
				slog.Int("content_length", len(b)))
			switch err := s.Characters(ctx, pctx.userData, b); err {
			case nil, sax.ErrHandlerUnspecified:
				// no op
			default:
				return pctx.error(ctx, err)
			}
		}
		return nil
	}

	// &...
	ent, err := pctx.parseEntityRef(ctx)
	if err != nil {
		return pctx.error(ctx, err)
	}
	// if !ctx.wellFormed { return } ??

	wasChecked := ent.checked

	// special case for predefined entities
	if ent.name == "" || EntityType(ent.EntityType()) == InternalPredefinedEntity {
		if ent.content == "" {
			return nil
		}
		if s := pctx.sax; s != nil {
			TraceEvent(ctx, "calling SAX Characters",
				slog.String("content_type", "entity_content"),
				slog.String("entity_name", ent.name),
				slog.Int("content_length", len(ent.content)))
			switch err := s.Characters(ctx, pctx.userData, []byte(ent.content)); err {
			case nil, sax.ErrHandlerUnspecified:
				// no op
			default:
				return pctx.error(ctx, err)
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
	if (wasChecked == 0 || (ent.firstChild == nil && pctx.options.IsSet(ParseNoEnt))) && (EntityType(ent.EntityType()) != ExternalGeneralParsedEntity || pctx.options.IsSet(ParseNoEnt|ParseDTDValid)) {
		var userData interface{}
		if pctx.userData != pctx {
			userData = pctx.userData
		}

		if EntityType(ent.EntityType()) == InternalGeneralEntity {
			parsedEnt, err = pctx.parseBalancedChunkInternal(ctx, []byte(ent.Content()), userData)
			switch err {
			case nil, ErrParseSucceeded:
				// may not have generated nodes, but parse was successful
			default:
				return err
			}
		} else if EntityType(ent.EntityType()) == ExternalGeneralParsedEntity {
			parsedEnt, err = pctx.parseExternalEntityPrivate(ent.uri, ent.externalID)
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
				if pctx.userData != pctx {
					userData = pctx.userData
				}
				if EntityType(ent.EntityType()) == InternalGeneralEntity {
					parsedEnt, err = pctx.parseBalancedChunkInternal(ctx, []byte(ent.Content()), userData)
					_ = parsedEnt
					switch err {
					case nil, ErrParseSucceeded:
						// may not have generated nodes, but parse was successful
					default:
						return err
					}
				} else if EntityType(ent.EntityType()) == ExternalGeneralParsedEntity {
					parsedEnt, err = pctx.parseExternalEntityPrivate(ent.URI(), ent.externalID)
					_ = parsedEnt
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
			if s := pctx.sax; s != nil && !pctx.replaceEntities {
				// Entity reference callback comes second, it's somewhat
				// superfluous but a compatibility to historical behaviour
				switch err := s.Reference(ctx, pctx.userData, ent.name); err {
				case nil, sax.ErrHandlerUnspecified:
					// no op
				default:
					return err
				}
			}
			return nil
		}

		// If we didn't get any children for the entity being built
		if s := pctx.sax; s != nil && !pctx.replaceEntities {
			// Create a node.
			switch err := s.Reference(ctx, pctx.userData, ent.name); err {
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
		_ = s // silence unused warning fornow
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
func (pctx *parserCtx) getEntity(name string) (*Entity, error) {
	if pctx.inSubset == 0 {
		if ret, err := resolvePredefinedEntity(name); err != nil {
			return ret, nil
		}
	}

	var ret *Entity
	var ok bool
	if pctx.doc == nil || pctx.doc.standalone != 1 {
		ret, _ = pctx.doc.GetEntity(name)
	} else {
		if pctx.inSubset == 2 {
			pctx.doc.standalone = 0
			ret, _ = pctx.doc.GetEntity(name)
			pctx.doc.standalone = 1
		} else {
			ret, ok = pctx.doc.GetEntity(name)
			if !ok {
				pctx.doc.standalone = 0
				ret, ok = pctx.doc.GetEntity(name)
				if !ok {
					return nil, errors.New("Entity(" + name + ") document marked standalone but requires eternal subset")
				}
				pctx.doc.standalone = 1
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

func (pctx *parserCtx) parseStringEntityRef(ctx context.Context, s []byte) (sax.Entity, int, error) {
	if len(s) == 0 || s[0] != '&' {
		return nil, 0, errors.New("invalid entity ref")
	}

	i := 1                                     // skip the '&'
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
	_ = s // silence unused warning for now
	i++

	var loadedEnt sax.Entity

	/*
	 * Ask first SAX for entity resolution, otherwise try the
	 * entities which may have stored in the parser context.
	 */
	if h := pctx.sax; h != nil {
		loadedEnt, err = h.GetEntity(ctx, pctx.userData, name)
		if err != nil {
			// Note: libxml2 would try to ask for xmlGetPredefinedEntity
			// next, but that's only when XML_PARSE_OLDSAX is enabled.
			// we won't do that.
			if pctx.wellFormed && pctx.userData == pctx {
				loadedEnt, err = pctx.getEntity(name)
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
		if pctx.standalone == StandaloneExplicitYes || (!pctx.hasExternalSubset && !pctx.hasPERefs) {
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
	if pctx.instate == psAttributeValue && EntityType(loadedEnt.EntityType()) == ExternalGeneralParsedEntity {
		return nil, 0, fmt.Errorf("attribute references enternal entity '%s'", name)
	}

	/*
	 * [ WFC: No < in Attribute Values ]
	 * The replacement text of any entity referred to directly or
	 * indirectly in an attribute value (other than "&lt;") must
	 * not contain a <.
	 */
	if pctx.instate == psAttributeValue && len(loadedEnt.Content()) > 0 && EntityType(loadedEnt.EntityType()) == InternalPredefinedEntity && bytes.IndexByte(loadedEnt.Content(), '<') > -1 {
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

func (pctx *parserCtx) parseStringPEReference(ctx context.Context, s []byte) (sax.Entity, int, error) {
	if len(s) == 0 || s[0] != '%' {
		return nil, 0, errors.New("invalid PEreference")
	}

	i := 1                                     // skip the '%'
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
	_ = s // silence unused warning for now
	i++

	var loadedEnt sax.Entity
	if h := pctx.sax; h != nil {
		loadedEnt, err = h.GetParameterEntity(ctx, pctx.userData, name)
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
		if pctx.standalone == StandaloneExplicitYes || (!pctx.hasExternalSubset && !pctx.hasPERefs) {
			return nil, 0, fmt.Errorf("not found: PE rerefence '%%%s'", name)
		} else {
			pctx.valid = false
		}
		// xmlParseEntityCheck(ctxt, 0, NULL, 0)
	} else {
		switch EntityType(loadedEnt.EntityType()) {
		case InternalParameterEntity, ExternalParameterEntity:
		default:
			return nil, 0, fmt.Errorf("not a parmeter entity: %%%s", name)
		}
	}
	pctx.hasPERefs = true

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
func (pctx *parserCtx) parseCharRef(ctx context.Context) (r rune, err error) {
	r = utf8.RuneError

	var val int32
	cur := pctx.getCursor(ctx)
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
			if err := cur.Advance(1); err != nil {
				return utf8.RuneError, err
			}
		}
		if cur.Peek() == ';' {
			if err := cur.Advance(1); err != nil {
				return utf8.RuneError, err
			}
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
			if err := cur.Advance(1); err != nil {
				return utf8.RuneError, err
			}
		}
		if cur.Peek() == ';' {
			if err := cur.Advance(1); err != nil {
				return utf8.RuneError, err
			}
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
func (pctx *parserCtx) parseEntityRef(ctx context.Context) (ent *Entity, err error) {
	cur := pctx.getCursor(ctx)
	if cur == nil {
		panic("did not get rune cursor")
	}
	if cur.Peek() != '&' {
		err = pctx.error(ctx, ErrAmpersandRequired)
		return
	}
	if err = cur.Advance(1); err != nil {
		return
	}

	name, err := pctx.parseName(ctx)
	if err != nil {
		err = pctx.error(ctx, ErrNameRequired)
		return
	}

	if cur.Peek() != ';' {
		err = pctx.error(ctx, ErrSemicolonRequired)
		return
	}
	if err = cur.Advance(1); err != nil {
		return
	}

	if ent, err = resolvePredefinedEntity(name); err == nil {
		return
	}

	if s := pctx.sax; s != nil {
		// ask the SAX2 handler nicely
		var loadedEnt sax.Entity
		loadedEnt, err = s.GetEntity(ctx, pctx.userData, name)
		if err == nil {
			ent = loadedEnt.(*Entity)
			return
		}

		if loadedEnt == nil && pctx == pctx.userData {
			ent, _ = pctx.getEntity(name)
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
		if pctx.standalone == StandaloneExplicitYes || (!pctx.hasExternalSubset && pctx.hasPERefs) {
			return nil, pctx.error(ctx, ErrUndeclaredEntity)
		} else {
			if pctx.inSubset == 0 {
				if s := pctx.sax; s != nil {
					switch err := s.Reference(ctx, pctx.userData, name); err {
					case nil, sax.ErrHandlerUnspecified:
						// no op
					default:
						return nil, pctx.error(ctx, err)
					}
				}
			}
			// ent is nil, no? why check?
			if err := pctx.entityCheck(ent, 0, 0); err != nil {
				return nil, pctx.error(ctx, err)
			}
			pctx.valid = false
		}
	} else if ent.entityType == ExternalGeneralUnparsedEntity {
		// [ WFC: Parsed Entity ]
		// An entity reference must not contain the name of an
		// unparsed entity
		return nil, pctx.error(ctx, errors.New("entity reference to unparsed entity"))
	} else if pctx.instate == psAttributeValue && ent.entityType == ExternalGeneralParsedEntity {
		// [ WFC: No External Entity References ]
		// Attribute values cannot contain direct or indirect
		// entity references to external entities.
		return nil, pctx.error(ctx, errors.New("attribute references external entity"))
	} else if pctx.instate == psAttributeValue && ent.entityType != InternalPredefinedEntity {
		// [ WFC: No < in Attribute Values ]
		// The replacement text of any entity referred to directly or
		// indirectly in an attribute value (other than "&lt;") must
		// not contain a <.
		if (ent.checked&1 == 1 || ent.checked == 0) && ent.content != "" && strings.IndexByte(ent.content, '<') > -1 {
			return nil, pctx.error(ctx, errors.New("'<' in entity is not allowed in attribute values"))
		}
	} else {
		// Internal check, no parameter entities here ...
		switch ent.entityType {
		case InternalParameterEntity:
		case ExternalParameterEntity:
			return nil, pctx.error(ctx, errors.New("attempt to reference the parameter entity"))
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
func (pctx *parserCtx) entityCheck(ent sax.Entity, size, replacement int) error {
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

func (pctx *parserCtx) handlePEReference(ctx context.Context) error {
	cur := pctx.getCursor(ctx)
	if cur == nil {
		panic("did not get rune cursor")
	}
	if cur.Peek() != '%' {
		// it's fine, this is not an error.
		return nil
	}

	switch st := pctx.instate; st {
	case psCDATA, psComment, psStartTag, psEndTag, psEntityDecl, psContent, psAttributeValue, psPI, psSystemLiteral, psPublicLiteral, psEntityValue, psIgnore:
		// NOTE: in the case of entity values, we don't do the
		//       substitution here since we need the literal
		//       entity value to be able to save the internal
		//       subset of the document.
		//       This will be handled by xmlStringDecodeEntities
		return nil
	case psEOF:
		return errors.New("handlePEReference: parameter entity at EOF")
	case psPrologue, psStart, psMisc:
		return errors.New("handlePEReference: parameter entity in prologue")
	case psEpilogue:
		return errors.New("handlePEReference: parameter entity in epilogue")
	case psDTD:
		// [WFC: Well-Formedness Constraint: PEs in Internal Subset]
		// In the internal DTD subset, parameter-entity references
		// can occur only where markup declarations can occur, not
		// within markup declarations.
		// In that case this is handled in xmlParseMarkupDecl
		if !pctx.external || pctx.inputTab.Len() == 1 {
			return nil
		}

		if c := cur.PeekN(2); isBlankCh(c) || c == 0x0 {
			return nil
		}
	}

	if err := cur.Advance(1); err != nil {
		return err
	}

	name, err := pctx.parseName(ctx)
	if err != nil {
		return err
	}

	if cur.Peek() != ';' {
		return ErrSemicolonRequired
	}

	if err := cur.Advance(1); err != nil {
		return err
	}

	var entity sax.Entity
	if s := pctx.sax; s != nil {
		entity, _ = s.GetParameterEntity(ctx, pctx.userData, name)
	}

	if pctx.instate == psEOF {
		return nil
	}

	if entity == nil {
		// [ WFC: Entity Declared ]
		// In a document without any DTD, a document with only an
		// internal DTD subset which contains no parameter entity
		// references, or a document with "standalone='yes'", ...
		// ... The declaration of a parameter entity must precede
		// any reference to it...
		if pctx.standalone == StandaloneExplicitYes || (!pctx.hasExternalSubset && !pctx.hasPERefs) {
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
		pctx.valid = false
		if err := pctx.entityCheck(nil, 0, 0); err != nil {
			return pctx.error(ctx, err)
		}
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
