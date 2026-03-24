package helium

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/url"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/lestrrat-go/helium/enum"
	icatalog "github.com/lestrrat-go/helium/internal/catalog"
	"github.com/lestrrat-go/helium/internal/encoding"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/sax"
	"github.com/lestrrat-go/pdebug"
	"github.com/lestrrat-go/helium/internal/strcursor"
)

//go:generate stringer -type=parserState

type parserState int

const (
	psEOF parserState = iota - 1
	psStart
	psPI
	psContent
	psPrologue
	psEpilogue
	psCDATA
	psDTD
	psEntityDecl
	psAttributeValue
	psComment
	psStartTag
	psEndTag
	psSystemLiteral
	psPublicLiteral
	psEntityValue
	psIgnore
	psMisc
)

var errInvalidUTF8Name = errors.New("invalid UTF-8 sequence in name")

const MaxNameLength = 50000

const (
	entityAllowedExpansion int64 = 1_000_000 // 1 MB baseline before ratio check
	entityFixedCost        int64 = 20        // fixed byte cost per entity reference
	entityMaxAmplDefault         = 5         // default max amplification factor
)

const (
	notInSubset = iota
	inInternalSubset
	inExternalSubset
)

type parserCtx struct {
	options ParseOption
	// ctx.encoding contains the explicit encoding. ctx.detectedEncoding
	// contains the encoding as detected by inspecting BOM, etc.
	// It is important to differentiate between the two, otherwise
	// we will not be able to reconstruct
	// <?xml version="1.0"?> vs <?xml version="1.0" encoding="utf-8"?>
	encoding         string
	detectedEncoding string
	in               io.Reader
	rawInput         []byte // original bytes, used for EBCDIC encoding detection
	nbread           int
	instate          parserState
	keepBlanks       bool
	// remain            int
	replaceEntities   bool
	sax               sax.SAX2Handler
	spaceTab          []int // xml:space stack: -1=inherit, 0=default, 1=preserve
	standalone        DocumentStandaloneType
	hasExternalSubset bool
	inSubset          int
	intSubName        string
	external          bool // true if parsing external DTDs
	extSubSystem      string
	extSubURI         string
	version           string
	attsSpecial       map[string]enum.AttributeType
	attsDefault       map[string][]*Attribute
	valid             bool
	hasPERefs         bool
	pedantic          bool
	wellFormed        bool
	depth             int
	loadsubset        LoadSubsetOption
	charBufferSize    int
	baseURI           string            // document base URI for resolving external references
	catalog           icatalog.Resolver // XML catalog for entity resolution
	elem              *Element          // current context element

	nsTab       nsStack
	nsNrTab     []int // number of ns bindings pushed per element (parallel to nodeTab)
	doc         *Document
	nodeTab     nodeStack
	elemidx     int
	sizeentcopy int64 // cumulative entity expansion bytes (non-entity-specific)
	inputSize   int64 // total input document size
	maxAmpl     int   // max amplification factor (default 5, 0 = disabled via ParseHuge)
	// nbentities int
	inputTab     inputStack
	cachedCursor strcursor.Cursor // cached result of getCursor(); invalidated on push/pop
	stopped      bool
	disableSAX       bool   // suppress SAX callbacks after fatal error in recover mode
	recoverErr       error  // first fatal error saved during recovery
	elemDepth        int    // current element nesting depth
	maxElemDepth     int    // max allowed element nesting depth (0 = unlimited)
	currentEntityURI string // URI of the external entity currently being replayed (for base-uri tracking)
}

type parserCtxKey struct{}

func withParserCtx(ctx context.Context, pctx *parserCtx) context.Context {
	return context.WithValue(ctx, parserCtxKey{}, pctx)
}

func getParserCtx(ctx context.Context) *parserCtx {
	if ctx == nil {
		return nil
	}
	pctx, _ := ctx.Value(parserCtxKey{}).(*parserCtx)
	return pctx
}

type SubstitutionType int

const (
	SubstituteNone SubstitutionType = iota
	SubstituteRef
	SubstitutePERef
	SubstituteBoth
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

func (ctx *parserCtx) pushNS(prefix, uri string) {
	ctx.nsTab.Push(prefix, uri)
}

const (
	cbEntityDecl = iota
	cbGetParameterEntity
)

func (pctx *parserCtx) fireSAXCallback(ctx context.Context, typ int, args ...any) error {
	// This is ugly, but I *REALLY* wanted to catch all occurences of
	// SAX callbacks being fired in one shot. optimize it later

	s := pctx.sax
	if s == nil {
		return nil
	}
	if pctx.disableSAX {
		return nil
	}

	switch typ {
	case cbEntityDecl:
		if pdebug.Enabled {
			g := pdebug.Marker("EntityDecl callback")
			defer g.End()
		}
		return s.EntityDecl(ctx, args[0].(string), enum.InternalParameterEntity, "", "", args[1].(string))
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
			for i, e := range ctx.nodeTab.Stack {
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
				for i, e := range ctx.nodeTab.Stack {
					pdebug.Printf("  %003d: %s (%p)", i, e.Name(), e)
				}
			}
		}()
	}
	return ctx.nodeTab.Pop()
}

func (ctx *parserCtx) lookupNamespace(prefix string) string {
	if prefix == XMLPrefix {
		return XMLNamespace
	}
	return ctx.nsTab.Lookup(prefix)
}

func (ctx *parserCtx) release() error {
	ctx.sax = nil
	return nil
}

// stop signals the parser to stop at the next opportunity.
func (ctx *parserCtx) stop() {
	ctx.stopped = true
	ctx.instate = psEOF
}

// LineNumber implements sax.DocumentLocator.
func (ctx *parserCtx) LineNumber() int {
	if cur := ctx.getCursor(); cur != nil {
		return cur.LineNumber()
	}
	return 0
}

// ColumnNumber implements sax.DocumentLocator.
func (ctx *parserCtx) ColumnNumber() int {
	if cur := ctx.getCursor(); cur != nil {
		return cur.Column()
	}
	return 0
}

// GetPublicID returns the public identifier of the document being parsed (libxml2: xmlSAXLocator.getPublicId).
// Always returns an empty string (libxml2 always returns NULL).
func (ctx *parserCtx) GetPublicID() string {
	return ""
}

// GetSystemID returns the system identifier (URI/filename) of the document being parsed (libxml2: xmlSAXLocator.getSystemId).
// Returns the base URI of the document being parsed.
func (ctx *parserCtx) GetSystemID() string {
	return ctx.baseURI
}

var bufferPool = sync.Pool{
	New: allocByteBuffer,
}

func allocByteBuffer() any {
	if pdebug.Enabled {
		pdebug.Printf("Allocating new bytes.Buffer...")
	}
	return &bytes.Buffer{}
}

func releaseBuffer(b *bytes.Buffer) {
	b.Reset()
	bufferPool.Put(b)
}

func (ctx *parserCtx) pushInput(in any) {
	if pdebug.Enabled {
		pdebug.Printf("pushInput (n = %d -> %d)", ctx.inputTab.Len(), ctx.inputTab.Len()+1)
	}
	ctx.inputTab.Push(in)
	ctx.cachedCursor = nil // invalidate cache
}

func (ctx *parserCtx) getByteCursor() *strcursor.ByteCursor {
	cur, ok := ctx.inputTab.PeekOne().(*strcursor.ByteCursor)
	if !ok {
		return nil
	}
	return cur
}

func (ctx *parserCtx) getCursor() strcursor.Cursor {
	// Fast path: return cached cursor if still valid (not done or only input).
	if cc := ctx.cachedCursor; cc != nil {
		if !cc.Done() || ctx.inputTab.Len() <= 1 {
			return cc
		}
		// Cached cursor is exhausted and there are more inputs — fall through.
		ctx.cachedCursor = nil
	}
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
		ctx.cachedCursor = cur
		return cur
	}
	return nil
}

func (ctx *parserCtx) popInput() any {
	ctx.cachedCursor = nil // invalidate cache
	return ctx.inputTab.Pop()
}

func (ctx *parserCtx) currentInputID() any {
	return ctx.inputTab.PeekOne()
}

func (ctx *parserCtx) init(p *Parser, in io.Reader) error {
	ctx.pushInput(strcursor.NewByteCursor(in))
	ctx.detectedEncoding = encUTF8
	ctx.encoding = ""
	ctx.in = in
	ctx.nbread = 0
	ctx.keepBlanks = true
	ctx.instate = psStart
	ctx.standalone = StandaloneImplicitNo
	ctx.attsSpecial = map[string]enum.AttributeType{}
	ctx.attsDefault = map[string][]*Attribute{}
	ctx.wellFormed = true
	ctx.spaceTab = ctx.spaceTab[:0]
	ctx.spaceTab = append(ctx.spaceTab, -1) // initial value before any element
	ctx.inputSize = int64(len(ctx.rawInput))
	ctx.maxAmpl = entityMaxAmplDefault
	if p != nil {
		ctx.sax = p.sax
		ctx.charBufferSize = p.charBufferSize
		ctx.options = p.options
		ctx.catalog = p.catalog
		if ctx.options.IsSet(ParseNoBlanks) {
			ctx.keepBlanks = false
		}
		if ctx.options.IsSet(ParsePedantic) {
			ctx.pedantic = true
		}
		if ctx.options.IsSet(ParseDTDLoad) {
			ctx.loadsubset.Set(DetectIDs)
		}
		if ctx.options.IsSet(ParseDTDAttr) {
			ctx.loadsubset.Set(CompleteAttrs)
		}
		if ctx.options.IsSet(ParseNoEnt) {
			ctx.replaceEntities = true
		}
		if ctx.options.IsSet(ParseHuge) {
			ctx.maxAmpl = 0
		}
		if ctx.options.IsSet(ParseSkipIDs) {
			ctx.loadsubset.Set(SkipIDs)
		}
		ctx.maxElemDepth = p.maxDepth
	}
	return nil
}

// deliverCharacters calls the given SAX character handler, splitting data
// into chunks of at most ctx.charBufferSize bytes when the buffer size is
// configured (> 0). Splits never occur in the middle of a multi-byte UTF-8
// character.
func (pctx *parserCtx) deliverCharacters(ctx context.Context, handler func(context.Context, []byte) error, data []byte) error {
	if pctx.disableSAX {
		return nil
	}
	bufSize := pctx.charBufferSize
	if bufSize <= 0 || len(data) <= bufSize {
		switch err := handler(ctx, data); err {
		case nil, sax.ErrHandlerUnspecified:
			if pctx.stopped {
				return errParserStopped
			}
			return nil
		default:
			return pctx.error(ctx, err)
		}
	}

	for len(data) > 0 {
		if pctx.stopped {
			return errParserStopped
		}

		end := bufSize
		if end >= len(data) {
			// Remaining data fits in one chunk.
			end = len(data)
		} else {
			// Walk backward from the proposed split point to find a valid
			// UTF-8 character boundary.
			for end > 0 && !utf8.RuneStart(data[end]) {
				end--
			}
			if end == 0 {
				// Should not happen with valid UTF-8, but avoid infinite loop.
				end = bufSize
				if end > len(data) {
					end = len(data)
				}
			}
		}

		switch err := handler(ctx, data[:end]); err {
		case nil, sax.ErrHandlerUnspecified:
			// no op
		default:
			return pctx.error(ctx, err)
		}
		data = data[end:]
	}
	return nil
}

func (pctx *parserCtx) error(ctx context.Context, err error) error {
	return pctx.errorAtLevel(ctx, err, ErrorLevelFatal)
}

func (pctx *parserCtx) namespaceError(ctx context.Context, err error) error {
	e := pctx.errorAtLevel(ctx, err, ErrorLevelFatal)
	if pe, ok := e.(ErrParseError); ok {
		pe.Domain = ErrorDomainNamespace
		return pe
	}
	return e
}

func (pctx *parserCtx) errorAtLevel(ctx context.Context, err error, level ErrorLevel) error {
	// errParserStopped is not a real error; pass through unwrapped.
	if errors.Is(err, errParserStopped) {
		return errParserStopped
	}
	// If it's wrapped, just return as is
	if _, ok := err.(ErrParseError); ok {
		return err
	}

	e := ErrParseError{Err: err, Level: level, File: pctx.baseURI}
	if cur := pctx.getCursor(); cur != nil {
		e.Column = cur.Column()
		e.LineNumber = cur.LineNumber()
		e.Line = cur.Line()
	}

	if s := pctx.sax; s != nil {
		switch level {
		case ErrorLevelWarning:
			if !pctx.options.IsSet(ParseNoWarning) {
				_ = s.Warning(ctx, e)
			}
		default:
			// Fire the SAX Error callback unless ParseNoError is set.
			if !pctx.options.IsSet(ParseNoError) {
				_ = s.Error(ctx, e)
			}
		}
	}

	return e
}

// warning wraps a warning condition in ErrParseError with ErrorLevelWarning
// and location info, then fires the SAX Warning callback with the structured
// error. Returns nil for non-fatal warnings. If the SAX Warning handler
// returns an error, returns the ErrParseError.
func (pctx *parserCtx) warning(ctx context.Context, format string, args ...any) error {
	msg := fmt.Sprintf(format, args...)
	if s := pctx.sax; s != nil && !pctx.options.IsSet(ParseNoWarning) {
		e := ErrParseError{Err: errors.New(msg), Level: ErrorLevelWarning, File: pctx.baseURI}
		if cur := pctx.getCursor(); cur != nil {
			e.Column = cur.Column()
			e.LineNumber = cur.LineNumber()
			e.Line = cur.Line()
		}
		switch err := s.Warning(ctx, e); err {
		case nil, sax.ErrHandlerUnspecified:
		default:
			return e
		}
	}
	return nil
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

// looksLikeXMLDecl returns true when the byte cursor starts with "<?xml"
// followed by a blank character (space/tab/CR/LF).  Processing instructions
// whose target starts with "xml" (e.g. <?xml-stylesheet …?>) are NOT XML
// declarations and must be parsed by parseMisc, not parseXMLDecl.
func looksLikeXMLDecl(bcur *strcursor.ByteCursor) bool {
	if !bcur.HasPrefix(xmlDeclHint) {
		return false
	}
	sixth := bcur.PeekN(6)
	return sixth == ' ' || sixth == '\t' || sixth == '\r' || sixth == '\n'
}

// looksLikeXMLDeclString is the rune-cursor variant of looksLikeXMLDecl.
func looksLikeXMLDeclString(cur strcursor.Cursor) bool {
	if !cur.HasPrefixString("<?xml") {
		return false
	}
	sixth := cur.PeekN(6)
	return sixth == ' ' || sixth == '\t' || sixth == '\r' || sixth == '\n'
}

func (pctx *parserCtx) parseDocument(ctx context.Context) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseDocument")
		defer g.IRelease("END parseDocument")
	}

	// Store pctx in the context so SAX callbacks (e.g. TreeBuilder) can
	// retrieve it via getParserCtx. Also store the document locator and
	// stop function so helium.StopParser works.
	ctx = withParserCtx(ctx, pctx)
	ctx = sax.WithDocumentLocator(ctx, pctx)
	ctx = context.WithValue(ctx, stopFuncKey{}, pctx.stop)

	if s := pctx.sax; s != nil {
		switch err := s.SetDocumentLocator(ctx, pctx); err {
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

	// For UTF-16 detected encodings, we must switch encoding FIRST
	// because the XML declaration is encoded in UTF-16, not ASCII.
	switch pctx.detectedEncoding {
	case encUTF16LE, encUTF16BE:
		// For UTF-16 detected encodings, we must switch encoding FIRST
		// because the XML declaration is encoded in UTF-16, not ASCII.
		if err := pctx.switchEncoding(); err != nil {
			return pctx.error(ctx, err)
		}
		cur := pctx.getCursor()
		if cur != nil && cur.HasPrefixString("<?xml") {
			if err := pctx.parseXMLDeclFromCursor(ctx); err != nil {
				return pctx.error(ctx, err)
			}
		}
	case encEBCDIC:
		// EBCDIC bytes are not ASCII-compatible, so we cannot parse the
		// XML declaration at byte level. Instead, scan the raw bytes
		// using the EBCDIC invariant character set (shared across all
		// EBCDIC Latin code pages) to extract the encoding name.
		if pctx.rawInput != nil {
			if encName := encoding.ExtractEBCDICEncoding(pctx.rawInput); encName != "" {
				pctx.encoding = encName
			}
		}
		// Fall back to IBM-037 (US EBCDIC) if no encoding was declared.
		if pctx.encoding == "" {
			pctx.encoding = "ibm037"
		}
		// Reset the byte cursor from the raw input so the decoder
		// reads from the beginning of the document.
		pctx.popInput()
		pctx.pushInput(strcursor.NewByteCursor(bytes.NewReader(pctx.rawInput)))
		if err := pctx.switchEncoding(); err != nil {
			return pctx.error(ctx, err)
		}
		// Parse the XML declaration from the decoded rune cursor.
		cur := pctx.getCursor()
		if cur != nil && looksLikeXMLDeclString(cur) {
			if err := pctx.parseXMLDeclFromCursor(ctx); err != nil {
				return pctx.error(ctx, err)
			}
		}
	default:
		// XML prolog (byte-level for ASCII-compatible encodings)
		if looksLikeXMLDecl(bcur) {
			if err := pctx.parseXMLDecl(ctx); err != nil {
				return pctx.error(ctx, err)
			}
		}

		// At this point we know the encoding, so switch the encoding
		// of the source.
		if err := pctx.switchEncoding(); err != nil {
			return pctx.error(ctx, err)
		}
	}

	if s := pctx.sax; s != nil {
		switch err := s.StartDocument(ctx); err {
		case nil, sax.ErrHandlerUnspecified:
			// no op
		default:
			return pctx.error(ctx, err)
		}
	}

	if pctx.stopped {
		return errParserStopped
	}

	// Misc part of the prolog
	if err := pctx.parseMisc(ctx); err != nil {
		return pctx.error(ctx, err)
	}

	cur := pctx.getCursor()
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

		// Query SAX callbacks for subset/standalone status.
		// These mirror libxml2's calls after internal subset parsing.
		if s := pctx.sax; s != nil {
			if has, err := s.HasInternalSubset(ctx); err == nil {
				_ = has // informational; handler may use for validation decisions
			}
			if has, err := s.HasExternalSubset(ctx); err == nil {
				_ = has
			}
		}

		pctx.inSubset = inExternalSubset
		if s := pctx.sax; s != nil {
			switch err := s.ExternalSubset(ctx, pctx.intSubName, pctx.extSubSystem, pctx.extSubURI); err {
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

		if pctx.stopped {
			return errParserStopped
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
		if err := pctx.parseContent(ctx); err != nil {
			return pctx.error(ctx, err)
		}

		if err := pctx.parseEpilogue(); err != nil {
			return pctx.error(ctx, err)
		}
	*/

	// All done
	if s := pctx.sax; s != nil {
		switch err := s.EndDocument(ctx); err {
		case nil, sax.ErrHandlerUnspecified:
			// no op
		default:
			return pctx.error(ctx, err)
		}
	}

	return nil
}

func (pctx *parserCtx) parseContent(ctx context.Context) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseContent")
		defer g.IRelease("END parseContent")
	}
	pctx.instate = psContent

	cur := pctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}

	recover := pctx.options.IsSet(ParseRecover)

	for !cur.Done() && !pctx.stopped {
		if cur.HasPrefixString("</") {
			break
		}

		var err error
		if cur.HasPrefixString("<?") {
			err = pctx.parsePI(ctx)
		} else if cur.HasPrefixString("<![CDATA[") {
			err = pctx.parseCDSect(ctx)
		} else if cur.HasPrefixString("<!--") {
			err = pctx.parseComment(ctx)
		} else if cur.HasPrefixString("<") {
			err = pctx.parseElement(ctx)
		} else if cur.HasPrefixString("&") {
			err = pctx.parseReference(ctx)
		} else {
			if err := pctx.parseCharData(ctx, false); err != nil {
				if !recover || errors.Is(err, errParserStopped) {
					return err
				}
				if pctx.recoverErr == nil {
					pctx.recoverErr = err
				}
				pctx.disableSAX = true
				pctx.wellFormed = false
				pctx.skipToRecoverPoint()
			}
			continue
		}

		if err != nil {
			if !recover || errors.Is(err, errParserStopped) {
				return pctx.error(ctx, err)
			}
			if pctx.recoverErr == nil {
				pctx.recoverErr = err
			}
			pctx.disableSAX = true
			pctx.wellFormed = false

			prevLine, prevCol := cur.LineNumber(), cur.Column()
			pctx.skipToRecoverPoint()
			if !cur.Done() && cur.LineNumber() == prevLine && cur.Column() == prevCol {
				_ = cur.Advance(1)
			}
			continue
		}
	}

	if pctx.stopped {
		return errParserStopped
	}

	if pctx.recoverErr != nil {
		return pctx.recoverErr
	}

	return nil
}

// skipToRecoverPoint advances the cursor past unrecoverable content to the
// next '<' character or EOF, for re-synchronization in ParseRecover mode.
func (ctx *parserCtx) skipToRecoverPoint() {
	cur := ctx.getCursor()
	if cur == nil {
		return
	}
	for !cur.Done() {
		if cur.Peek() == '<' {
			return
		}
		_ = cur.Advance(1)
	}
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
	if cdata {
		_, err := pctx.parseCDataContent()
		return err
	}
	return pctx.parseCharDataContent(ctx)
}

// parseCDataContent reads the text inside a CDATA section (up to but not
// including the closing ]]>) and returns it. The caller is responsible for
// consuming ]]> and firing the SAX callback afterward, matching libxml2's
// behavior of reporting the position after the closing delimiter.
func (ctx *parserCtx) parseCDataContent() (string, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseCDataContent")
		defer g.IRelease("END parseCDataContent")
	}

	buf := bufferPool.Get().(*bytes.Buffer)
	defer releaseBuffer(buf)

	cur := ctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}

	i := 0
	for c := cur.PeekN(i + 1); c != 0x0; c = cur.PeekN(i + 1) {
		if c == ']' && cur.PeekN(i+2) == ']' && cur.PeekN(i+3) == '>' {
			break
		}
		_, _ = buf.WriteRune(c)
		i++
	}

	if err := cur.Advance(i); err != nil {
		return "", err
	}
	str := buf.String()

	// XML §2.11 End-of-Line Handling: normalize \r\n to \n, then lone \r to \n.
	str = strings.ReplaceAll(str, "\r\n", "\n")
	str = strings.ReplaceAll(str, "\r", "\n")
	return str, nil
}

func (pctx *parserCtx) parseCharDataContent(ctx context.Context) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseCharDataContent")
		defer g.IRelease("END parseCharDataContent")
	}

	buf := bufferPool.Get().(*bytes.Buffer)
	defer releaseBuffer(buf)

	cur := pctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}

	needsNormalize := false
	i := 0
	for c := cur.PeekN(i + 1); c != 0x0; c = cur.PeekN(i + 1) {
		if c == '<' || c == '&' || !isChar(c) {
			break
		}

		if c == ']' && cur.PeekN(i+2) == ']' && cur.PeekN(i+3) == '>' {
			return pctx.error(ctx, ErrMisplacedCDATAEnd)
		}

		if c == '\r' {
			needsNormalize = true
		}
		_, _ = buf.WriteRune(c)
		i++
	}

	if i <= 0 {
		pdebug.Dump(cur)
		return errors.New("invalid char data")
	}

	if err := cur.Advance(i); err != nil {
		return err
	}

	// Work with bytes directly to avoid buf.String() + []byte(str) allocations.
	data := buf.Bytes()
	if needsNormalize {
		// XML §2.11 End-of-Line Handling: normalize \r\n to \n, then lone \r to \n.
		data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
		data = bytes.ReplaceAll(data, []byte("\r"), []byte("\n"))
	}
	if pctx.areBlanksBytes(data, false) {
		if s := pctx.sax; s != nil && !pctx.disableSAX {
			if err := pctx.deliverCharacters(ctx, s.IgnorableWhitespace, data); err != nil {
				return err
			}
		}
	} else {
		if s := pctx.sax; s != nil && !pctx.disableSAX {
			if err := pctx.deliverCharacters(ctx, s.Characters, data); err != nil {
				return err
			}
		}
	}

	return nil
}

func (pctx *parserCtx) parseElement(ctx context.Context) error {
	if pdebug.Enabled {
		pctx.elemidx++
		i := pctx.elemidx
		g := pdebug.IPrintf("START parseElement (%d)", i)
		defer g.IRelease("END parseElement (%d)", i)
	}

	pctx.elemDepth++
	defer func() { pctx.elemDepth-- }()

	if pctx.maxElemDepth > 0 && pctx.elemDepth > pctx.maxElemDepth {
		return pctx.error(ctx, fmt.Errorf("xml: exceeded max depth"))
	}

	// parseStartTag only parses up to the attributes.
	// For example, given <foo>bar</foo>, the next token would
	// be bar</foo>. Given <foo />, the next token would
	// be />
	if err := pctx.parseStartTag(ctx); err != nil {
		return pctx.error(ctx, err)
	}

	cur := pctx.getCursor()
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
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseStartTag")
		defer g.IRelease("END parseStartTag")
	}

	cur := pctx.getCursor()
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

	// Push xml:space stack entry for this element (inherit parent's value by default)
	pctx.spaceTab = append(pctx.spaceTab, -1)

	nbNs := 0
	// Use stack-allocated backing array for small attribute counts (common case).
	var attrsBuf [8]sax.Attribute
	attrs := attrsBuf[:0]
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
			// Namespace URI entity/character references are expanded inline
			// during attribute value parsing (replaceEntities forced true in
			// parseAttribute for namespace attrs), so no post-processing needed.

			// ParseNsClean: skip redundant namespace declarations
			if pctx.options.IsSet(ParseNsClean) && pctx.nsTab.Lookup("") == attvalue {
				goto SkipDefaultNS
			}
			pctx.pushNS("", attvalue)
			nbNs++
		SkipDefaultNS:
			if cur.Peek() == '>' || cur.HasPrefixString("/>") {
				continue
			}

			if !isBlankCh(cur.Peek()) {
				return pctx.error(ctx, ErrSpaceRequired)
			}
			pctx.skipBlanks(ctx)
			continue
		} else if aprefix == XMLNsPrefix {
			var u *url.URL         // predeclare, so we can use goto SkipNS
			var existingURI string // predeclare, so we can use goto SkipNS

			// <elem xmlns:foo="...">
			// Namespace URI entity/character references are expanded inline
			// during attribute value parsing (replaceEntities forced true in
			// parseAttribute for namespace attrs), so no post-processing needed.
			if attname == XMLPrefix { // xmlns:xml
				if attvalue != XMLNamespace {
					return pctx.namespaceError(ctx, errors.New("xml namespace prefix mapped to wrong URI"))
				}
				// skip storing namespace definition
				goto SkipNS
			}
			if attname == XMLNsPrefix { // xmlns:xmlns="..."
				return pctx.namespaceError(ctx, errors.New("redefinition of the xmlns prefix forbidden"))
			}

			if attvalue == lexicon.NamespaceXMLNS {
				return pctx.namespaceError(ctx, errors.New("reuse of the xmlns namespace name if forbidden"))
			}

			if attvalue == "" {
				return pctx.namespaceError(ctx, fmt.Errorf("xmlns:%s: Empty XML namespace is not allowed", attname))
			}

			u, err = url.Parse(attvalue)
			if err != nil {
				return pctx.namespaceError(ctx, fmt.Errorf("xmlns:%s: '%s' is not a validURI", attname, attvalue))
			}
			if pctx.pedantic && u.Scheme == "" {
				return pctx.namespaceError(ctx, fmt.Errorf("xmlns:%s: URI %s is not absolute", attname, attvalue))
			}

			// Check only the current element's bindings (top nbNs entries)
			// to detect true duplicates. A prefix bound in an ancestor
			// element is valid shadowing, not a duplicate.
			existingURI = pctx.nsTab.LookupInTopN(attname, nbNs)
			if existingURI != "" {
				if pctx.options.IsSet(ParseNsClean) && existingURI == attvalue {
					goto SkipNS
				}
				return pctx.error(ctx, errors.New("duplicate attribute is not allowed"))
			}
			// ParseNsClean: skip if an ancestor already binds this prefix
			// to the same URI (redundant redeclaration).
			if pctx.options.IsSet(ParseNsClean) && pctx.nsTab.Lookup(attname) == attvalue {
				goto SkipNS
			}
			pctx.pushNS(attname, attvalue)
			nbNs++

		SkipNS:
			if cur.Peek() == '>' || cur.HasPrefixString("/>") {
				continue
			}

			if !isBlankCh(cur.Peek()) {
				return pctx.error(ctx, ErrSpaceRequired)
			}
			pctx.skipBlanks(ctx)
			// pctx.input.base != base || inputNr != pctx.inputNr; goto base_changed
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

	// Attributes defaulting: apply DTD-declared default attribute values.
	// NOTE: #FIXED/#REQUIRED validation and element content model checking
	// are done post-parse via validateDocument() when ParseDTDValid is set.
	// ID/IDREF uniqueness checks are done post-parse via validateDocument().
	if len(pctx.attsDefault) > 0 {
		var elemName string
		if prefix != "" {
			elemName = prefix + ":" + local
		} else {
			elemName = local
		}

		if pdebug.Enabled {
			pdebug.Printf("-------> %s", elemName)
		}
		defaults, ok := pctx.lookupAttributeDefault(elemName)
		if ok {
			// First pass: apply default xmlns="..." (must come before prefixed)
			for _, attr := range defaults {
				if attr.LocalName() == XMLNsPrefix && attr.Prefix() == "" {
					pctx.pushNS("", attr.Value())
					nbNs++
				}
			}
			// Second pass: apply xmlns:prefix="..." and regular attributes
			for _, attr := range defaults {
				attname := attr.LocalName()
				aprefix := attr.Prefix()
				if attname == XMLNsPrefix && aprefix == "" {
					continue // already handled
				} else if aprefix == XMLNsPrefix {
					pctx.pushNS(attname, attr.Value())
					nbNs++
				} else {
					// Skip if an explicit attribute with the same name exists
					dup := false
					for _, ea := range attrs {
						if ea.LocalName() == attname && ea.Prefix() == aprefix {
							dup = true
							break
						}
					}
					if !dup {
						attrs = append(attrs, attr)
					}
				}
			}
		}
	}

	// Scan attributes for xml:space to update the space stack
	for _, a := range attrs {
		if a.Prefix() == XMLPrefix && a.LocalName() == "space" {
			switch a.Value() {
			case "preserve":
				pctx.spaceTab[len(pctx.spaceTab)-1] = 1
			case "default":
				pctx.spaceTab[len(pctx.spaceTab)-1] = 0
			}
			break
		}
	}

	// we push the element first, because this way we get to
	// query for the namespace declared on this node as well
	// via lookupNamespace
	nsuri := pctx.lookupNamespace(prefix)
	if prefix != "" && nsuri == "" {
		return pctx.namespaceError(ctx, errors.New("namespace '"+prefix+"' not found"))
	}
	if nsuri != "" {
		if err := elem.SetActiveNamespace(prefix, nsuri); err != nil {
			return err
		}
	}

	if s := pctx.sax; s != nil && !pctx.disableSAX {
		var nslist []sax.Namespace
		if nbNs > 0 {
			nslist = make([]sax.Namespace, nbNs)
			// workaround []*Namespace != []sax.Namespace
			for i, ns := range pctx.nsTab.Peek(nbNs) {
				nslist[i] = ns
			}
		}
		switch err := s.StartElementNS(ctx, elem.LocalName(), prefix, nsuri, nslist, attrs); err {
		case nil, sax.ErrHandlerUnspecified:
			// no op
		default:
			return pctx.error(ctx, err)
		}
	}
	pctx.pushNode(elem)
	pctx.nsNrTab = append(pctx.nsNrTab, nbNs)

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
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseEndTag")
		defer g.IRelease("END parseEndTag")
	}

	cur := pctx.getCursor()
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

		// [NS 9] ETag ::= '</' QName S? '>'
		pctx.skipBlanks(ctx)

		if cur.Peek() != '>' {
			return pctx.error(ctx, ErrGtRequired)
		}
		if err := cur.Advance(1); err != nil {
			return err
		}
	}

	e := pctx.peekNode()
	if s := pctx.sax; s != nil && !pctx.disableSAX {
		switch err := s.EndElementNS(ctx, e.LocalName(), e.Prefix(), e.URI()); err {
		case nil, sax.ErrHandlerUnspecified:
			// no op
		default:
			return pctx.error(ctx, err)
		}
	}
	pctx.popNode()

	// Pop xml:space stack entry for this element.
	if len(pctx.spaceTab) > 1 {
		pctx.spaceTab = pctx.spaceTab[:len(pctx.spaceTab)-1]
	}

	// Pop namespace bindings that were pushed by this element's start tag.
	// This mirrors libxml2's nsPop(ctxt, nbNs) in xmlParseEndTag2.
	if n := len(pctx.nsNrTab); n > 0 {
		nbNs := pctx.nsNrTab[n-1]
		pctx.nsNrTab = pctx.nsNrTab[:n-1]
		if nbNs > 0 {
			pctx.nsTab.Pop(nbNs)
		}
	}

	return nil
}

func (pctx *parserCtx) parseAttributeValue(ctx context.Context, normalize bool) (value string, entities int, err error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseAttributeValue (normalize=%t)", normalize)
		defer g.IRelease("END parseAttributeValue")
	}

	// Inline quote handling from parseQuotedText to avoid closure allocation.
	cur := pctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	qch := cur.Peek()
	switch qch {
	case '"', '\'':
		if err = cur.Advance(1); err != nil {
			return
		}
	default:
		err = errors.New("string not started (got '" + string([]rune{qch}) + "')")
		return
	}

	value, entities, err = pctx.parseAttributeValueInternal(ctx, qch, normalize)
	if err != nil {
		return
	}

	if cur.Peek() != qch {
		err = errors.New("string not closed")
		return
	}
	err = cur.Advance(1)
	return
}

// This is based on xmlParseAttValueComplex
func (pctx *parserCtx) parseAttributeValueInternal(ctx context.Context, qch rune, normalize bool) (value string, entities int, err error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseAttributeValueInternal (qch='%c',normalize=%t)", qch, normalize)
		defer g.IRelease("END parseAttributeValueInternal")
		defer func() {
			pdebug.Printf("value = '%s'", value)
		}()
	}

	prevState := pctx.instate
	pctx.instate = psAttributeValue
	defer func() { pctx.instate = prevState }()

	cur := pctx.getCursor()
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
				r, err = pctx.parseCharRef()
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

				if ent == nil {
					// Undeclared entity in non-standalone document with external subset.
					// Treat as empty value (matches libxml2 behavior).
					continue
				}

				if ent.entityType == enum.InternalPredefinedEntity {
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
					// Even when not replacing entities, libxml2 validates
					// entity content by resolving nested references. This
					// triggers getEntity callbacks for transitive refs.
					if ent.checked == 0 && strings.ContainsRune(ent.content, '&') {
						_, _ = pctx.decodeEntities(ctx, ent.Content(), SubstituteRef)
						ent.checked = 2
					}
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
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseAttribute")
		defer g.IRelease("END parseAttribute")
		defer func() {
			pdebug.Printf("local = '%s', prefix = '%s', value = '%s'", local, prefix, value)
		}()
	}
	l, p, err := pctx.parseQName(ctx)
	if err != nil {
		err = pctx.error(ctx, err)
		return
	}

	normalize := false
	attType, ok := pctx.lookupSpecialAttribute(elemName, l)
	if pdebug.Enabled {
		pdebug.Printf("looked up attribute %s:%s -> %d (%t)", elemName, l, attType, ok)
	}
	if ok && attType != enum.AttrInvalid {
		normalize = true
	}
	pctx.skipBlanks(ctx)

	cur := pctx.getCursor()
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

	// Namespace URIs must always have entities expanded inline during
	// parsing, matching libxml2's isNamespace flag in xmlParseAttValueInternal.
	// This avoids a second decodeEntities pass that would trigger extra
	// SAX getEntity callbacks.
	isNamespace := (l == XMLNsPrefix && p == "") || p == XMLNsPrefix
	savedReplaceEntities := pctx.replaceEntities
	if isNamespace {
		pctx.replaceEntities = true
	}

	v, entities, err := pctx.parseAttributeValue(ctx, normalize)

	pctx.replaceEntities = savedReplaceEntities

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
		if pdebug.Enabled {
			pdebug.Printf("normalize is true, checking if entities have been expanded...")
		}
		if entities > 0 {
			if pdebug.Enabled {
				pdebug.Printf("entities seems to have been expanded (%d): doint second normalization", entities)
			}
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
	if pdebug.Enabled {
		g := pdebug.IPrintf("START skipBlanks")
		defer func() {
			g.IRelease("END skipBlanks (skipped %d)", i)
		}()
	}
	cur := pctx.getCursor()
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
			pdebug.Printf("Found possible parameter entity reference")
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
		if err := cur.Advance(i); err != nil {
			return false
		}

		if cur.Peek() == '%' {
			pdebug.Printf("Found possible parameter entity reference")
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

	if pctx.options.IsSet(ParseLenientXMLDecl) {
		return pctx.parseXMLDeclLenient(ctx)
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
	if err == nil && !pctx.options.IsSet(ParseIgnoreEnc) {
		pctx.encoding = v
	}

	// we *may* have standalone decl
	pctx.skipBlankBytes(ctx, cur)
	if cur.Peek() == '?' && cur.PeekN(2) == '>' {
		if err := cur.Advance(2); err != nil {
			return err
		}
		return nil
	}

	vb, err := pctx.parseStandaloneDecl(ctx)
	if err == nil {
		pctx.standalone = vb
	}

	pctx.skipBlankBytes(ctx, cur)
	if cur.Peek() == '?' && cur.PeekN(2) == '>' {
		if err := cur.Advance(2); err != nil {
			return err
		}
		return nil
	}
	return pctx.error(ctx, errors.New("XML declaration not closed"))
}

// parseXMLDeclLenient parses the XML declaration pseudo-attributes in any order.
// Called when parseopts.LenientXMLDecl is set.
func (pctx *parserCtx) parseXMLDeclLenient(ctx context.Context) error {
	cur := pctx.getByteCursor()

	for {
		pctx.skipBlankBytes(ctx, cur)
		if cur.Peek() == '?' && cur.PeekN(2) == '>' {
			if err := cur.Advance(2); err != nil {
				return err
			}
			return nil
		}

		if v, err := pctx.parseVersionInfo(ctx); err == nil {
			pctx.version = v
			continue
		}

		if v, err := pctx.parseEncodingDecl(ctx); err == nil {
			if !pctx.options.IsSet(ParseIgnoreEnc) {
				pctx.encoding = v
			}
			continue
		}

		if vb, err := pctx.parseStandaloneDecl(ctx); err == nil {
			pctx.standalone = vb
			continue
		}

		return pctx.error(ctx, errors.New("XML declaration not closed"))
	}
}

// parseXMLDeclFromCursor parses the XML declaration from a rune cursor.
// This is used for UTF-16 documents where the encoding has already been
// switched before parsing the XML declaration.
func (pctx *parserCtx) parseXMLDeclFromCursor(ctx context.Context) error {
	cur := pctx.getCursor()
	if cur == nil {
		return errors.New("rune cursor required for parseXMLDeclFromCursor")
	}

	if !cur.ConsumeString("<?xml") {
		return pctx.error(ctx, ErrInvalidXMLDecl)
	}

	if !pctx.skipBlanks(ctx) {
		return errors.New("blank needed after '<?xml'")
	}

	if pctx.options.IsSet(ParseLenientXMLDecl) {
		return pctx.parseXMLDeclFromCursorLenient(ctx)
	}

	// version
	v, err := pctx.parseVersionInfoFromCursor(ctx)
	if err != nil {
		return pctx.error(ctx, err)
	}
	pctx.version = v

	if !isBlankCh(cur.Peek()) {
		if cur.Peek() == '?' {
			if err := cur.Advance(1); err != nil {
				return err
			}
			if cur.Peek() == '>' {
				if err := cur.Advance(1); err != nil {
					return err
				}
				return nil
			}
			return pctx.error(ctx, errors.New("XML declaration not closed"))
		}
		return pctx.error(ctx, ErrSpaceRequired)
	}

	// encoding (optional)
	ev, err := pctx.parseEncodingDeclFromCursor(ctx)
	if err == nil && !pctx.options.IsSet(ParseIgnoreEnc) {
		pctx.encoding = ev
	}

	// standalone (optional)
	pctx.skipBlanks(ctx)
	if cur.Peek() == '?' {
		if err := cur.Advance(1); err != nil {
			return err
		}
		if cur.Peek() == '>' {
			if err := cur.Advance(1); err != nil {
				return err
			}
			return nil
		}
		return pctx.error(ctx, errors.New("XML declaration not closed"))
	}

	sv, err := pctx.parseStandaloneDeclFromCursor(ctx)
	if err == nil {
		pctx.standalone = sv
	}

	pctx.skipBlanks(ctx)
	if cur.Peek() == '?' {
		if err := cur.Advance(1); err != nil {
			return err
		}
		if cur.Peek() == '>' {
			if err := cur.Advance(1); err != nil {
				return err
			}
			return nil
		}
	}
	return pctx.error(ctx, errors.New("XML declaration not closed"))
}

// parseXMLDeclFromCursorLenient parses the XML declaration pseudo-attributes
// in any order using the rune cursor. Called when parseopts.LenientXMLDecl is set.
func (pctx *parserCtx) parseXMLDeclFromCursorLenient(ctx context.Context) error {
	cur := pctx.getCursor()

	for {
		pctx.skipBlanks(ctx)
		if cur.Peek() == '?' {
			if err := cur.Advance(1); err != nil {
				return err
			}
			if cur.Peek() == '>' {
				if err := cur.Advance(1); err != nil {
					return err
				}
				return nil
			}
			return pctx.error(ctx, errors.New("XML declaration not closed"))
		}

		if v, err := pctx.parseVersionInfoFromCursor(ctx); err == nil {
			pctx.version = v
			continue
		}

		if ev, err := pctx.parseEncodingDeclFromCursor(ctx); err == nil {
			if !pctx.options.IsSet(ParseIgnoreEnc) {
				pctx.encoding = ev
			}
			continue
		}

		if sv, err := pctx.parseStandaloneDeclFromCursor(ctx); err == nil {
			pctx.standalone = sv
			continue
		}

		return pctx.error(ctx, errors.New("XML declaration not closed"))
	}
}

func (pctx *parserCtx) parseVersionInfoFromCursor(ctx context.Context) (string, error) {
	cur := pctx.getCursor()
	pctx.skipBlanks(ctx)
	if !cur.ConsumeString("version") {
		return "", pctx.error(ctx, ErrAttrNotFound{Token: "version"})
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
		buf.WriteRune(c)
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
		return "", ErrAttrNotFound{Token: "encoding"}
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
		buf.WriteRune(c)
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
		return StandaloneImplicitNo, ErrAttrNotFound{Token: "standalone"}
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
		buf.WriteRune(c)
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

func (e ErrAttrNotFound) Error() string {
	return "attribute token '" + e.Token + "' not found"
}

/*
func (pctx *parserCtx) parseNamedAttribute(ctx context.Context, name string, cb qtextHandler) (string, error) {
	pctx.skipBlanks(ctx)

	cur := pctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if !cur.ConsumeString(name) {
		return "", pctx.error(ctx, ErrAttrNotFound{Token: name})
	}

	pctx.skipBlanks(ctx)
	if cur.Peek() != '=' {
		return "", ErrEqualSignRequired
	}

	if err := cur.Advance(1); err != nil {
		return "", err
	}
	pctx.skipBlanks(ctx)
	return pctx.parseQuotedText(cb)
}
*/

// parse the XML version info (version="1.0")
var versionBytes = []byte{'v', 'e', 'r', 's', 'i', 'o', 'n'}

func (pctx *parserCtx) parseVersionInfo(ctx context.Context) (string, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseVersionInfo")
		defer g.IRelease("END parseVersionInfo")
	}

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
			if err := cur.Advance(i - 1); err != nil {
				return "", err
			}
			return b.String(), nil
		}
	}
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
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseEncodingDecl")
		defer g.IRelease("END parseEncodingDecl")
	}
	return pctx.parseNamedAttributeBytes(ctx, encodingBytes, func(qch rune) (string, error) {
		return pctx.parseEncodingName(ctx, qch)
	})
}

func (pctx *parserCtx) parseEncodingName(ctx context.Context, _ rune) (string, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseEncodingName")
		defer g.IRelease("END parseEncodingName")
	}
	cur := pctx.getByteCursor()
	if cur == nil {
		return "", ErrByteCursorRequired
	}
	c := cur.Peek()

	buf := bufferPool.Get().(*bytes.Buffer)
	defer releaseBuffer(buf)

	// first char needs to be alphabets
	if !(c >= 'a' && c <= 'z') && !(c >= 'A' && c <= 'Z') { // nolint:staticcheck
		return "", pctx.error(ctx, ErrInvalidEncodingName)
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
	} else {
		return StandaloneExplicitNo, nil
	}
}

func (ctx *parserCtx) parseStandaloneDeclValue(_ rune) (string, error) {
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
		} else if isBlankCh(cur.Peek()) {
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
		if s := pctx.sax; s != nil && !pctx.disableSAX {
			switch err := s.ProcessingInstruction(ctx, target, ""); err {
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

	if s := pctx.sax; s != nil && !pctx.disableSAX {
		switch err := s.ProcessingInstruction(ctx, target, data); err {
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
		panic("did not get rune cursor")
	}

	// first letter
	c := cur.Peek()
	if c == utf8.RuneError {
		err = pctx.error(ctx, errInvalidUTF8Name)
		return
	}
	if c == ' ' || c == '>' || c == '/' || (c != ':' && !isValidNameStartChar(c)) {
		err = pctx.error(ctx, fmt.Errorf("invalid first letter '%c'", c))
		return
	}

	i := 2
	for c = cur.PeekN(i); c != 0x0; c = cur.PeekN(i) {
		if c == ' ' || c == '>' || c == '/' {
			i--
			break
		}
		if c == utf8.RuneError {
			err = pctx.error(ctx, errInvalidUTF8Name)
			return
		}
		if c != ':' && !isValidNameChar(c) {
			i--
			break
		}

		i++
	}
	if i > MaxNameLength && !pctx.options.IsSet(ParseHuge) {
		err = pctx.error(ctx, ErrNameTooLong)
		return
	}

	name = cur.PeekString(i)
	if err := cur.Advance(i); err != nil {
		return "", err
	}
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
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseQName")
		defer g.IRelease("END parseQName")
		defer func() { pdebug.Printf("local='%s' prefix='%s'", local, prefix) }()
	}

	cur := pctx.getCursor()
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

	v, err = pctx.parseNmtoken()
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
	return r != utf8.RuneError && (r == ':' || isValidNameStartChar(r))
}

func isNameChar(r rune) bool {
	return r != utf8.RuneError && (r == ':' || isValidNameChar(r))
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

	for c := cur.PeekN(i); c != 0x0; c = cur.PeekN(i) {
		if !isNameChar(c) {
			i--
			break
		}
		i++
	}
	name := cur.PeekString(i)
	if err := cur.Advance(i); err != nil {
		return "", err
	}

	return name, nil
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
		panic("did not get rune cursor")
	}

	var c rune
	if c = cur.Peek(); c == utf8.RuneError {
		err = pctx.error(ctx, errInvalidUTF8Name)
		return
	}
	if c == ' ' || c == '>' || c == '/' || c == ':' || !isValidNameStartChar(c) {
		err = pctx.error(ctx, fmt.Errorf("invalid name start char %q (U+%04X)", c, c))
		return
	}

	// Count the length of the name without writing to a buffer.
	i := 2
	for c = cur.PeekN(i); c != 0x0; c = cur.PeekN(i) {
		if c == utf8.RuneError {
			err = pctx.error(ctx, errInvalidUTF8Name)
			return
		}
		if c == ':' || !isValidNameChar(c) {
			i--
			break
		}
		i++
	}
	if i > MaxNameLength && !pctx.options.IsSet(ParseHuge) {
		err = pctx.error(ctx, ErrNameTooLong)
		return
	}
	// Extract the name string directly from the cursor ring buffer.
	ncname = cur.PeekString(i)
	if err := cur.Advance(i); err != nil {
		return "", err
	}
	return
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

// note: unlike libxml2, we can't differentiate between SAX handlers
// that uses the same IgnorableWhitespace and Character handlers
// areBlanksBytes is like areBlanks but operates on []byte to avoid string
// allocation on the hot path.
func (ctx *parserCtx) areBlanksBytes(s []byte, blankChars bool) bool {
	// Check for xml:space value.
	if ctx.spaceTab[len(ctx.spaceTab)-1] == 1 {
		return false
	}

	// Check that the data is made of blanks
	if !blankChars {
		for _, b := range s {
			if !isBlankCh(rune(b)) {
				return false
			}
		}
	}

	// Look if the element is mixed content in the DTD if available
	if ctx.peekNode() == nil {
		return false
	}
	if ctx.doc != nil {
		ok, _ := ctx.doc.IsMixedElement(ctx.peekNode().Name())
		return !ok
	}

	cur := ctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if c := cur.Peek(); c != '<' && c != 0xD {
		return false
	}

	return true
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
		panic("did not get rune cursor")
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

	// Consume ]]> BEFORE firing the SAX callback, matching libxml2's
	// behavior so that the document locator reports the position after
	// the closing delimiter.
	if !cur.ConsumeString("]]>") {
		return pctx.error(ctx, ErrCDATANotFinished)
	}

	if s := pctx.sax; s != nil && !pctx.disableSAX {
		if pctx.options.IsSet(ParseNoCDATA) {
			if err := pctx.deliverCharacters(ctx, s.Characters, []byte(str)); err != nil {
				return err
			}
		} else {
			switch err := s.CDataBlock(ctx, []byte(str)); err {
			case nil, sax.ErrHandlerUnspecified:
				// no op
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

	if sh := pctx.sax; sh != nil && !pctx.disableSAX {
		// XML §2.11 End-of-Line Handling: normalize \r\n to \n, then lone \r to \n.
		str = bytes.ReplaceAll(str, []byte{'\r', '\n'}, []byte{'\n'})
		str = bytes.ReplaceAll(str, []byte{'\r'}, []byte{'\n'})
		switch err := sh.Comment(ctx, str); err {
		case nil, sax.ErrHandlerUnspecified:
			// no op
		default:
			return pctx.error(ctx, err)
		}
	}

	return nil
}

func (pctx *parserCtx) parseDocTypeDecl(ctx context.Context) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseDocTypeDecl")
		defer g.IRelease("END parseDocTypeDecl")
	}

	cur := pctx.getCursor()
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
	u, eid, err := pctx.parseExternalID(ctx)
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
		switch err := s.InternalSubset(ctx, name, eid, u); err {
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
	// equiv: xmlParseInternalSubset (parser.c)
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseInternalSubset")
		defer g.IRelease("END parseInternalSubset")
	}

	cur := pctx.getCursor()
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
		if pctx.stopped {
			return errParserStopped
		}
		// Get current cursor in case parameter entity expansion changed the input
		cur = pctx.getCursor()
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
	cur = pctx.getCursor()
	if cur != nil && cur.Peek() == ']' {
		if err := cur.Advance(1); err != nil {
			return err
		}
		pctx.skipBlanks(ctx)
	}

FinishDTD:
	// Ensure we have the current cursor
	cur = pctx.getCursor()
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
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseMarkupDecl")
		defer g.IRelease("END parseMarkupDecl")
	}

	cur := pctx.getCursor()
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
				if err := pctx.parseNotationDecl(ctx); err != nil {
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
	// Conditional sections are allowed from entities included
	// by PE References in the internal subset.
	if !pctx.external && pctx.inputTab.Len() > 1 {
		cur = pctx.getCursor()
		if cur != nil && cur.Peek() == '<' && cur.PeekN(2) == '!' && cur.PeekN(3) == '[' {
			if err := pctx.parseConditionalSections(ctx); err != nil {
				return pctx.error(ctx, err)
			}
			return nil
		}
	}
	pctx.instate = psDTD

	return nil
}

// parseConditionalSections parses conditional sections in a DTD.
//
//	[61] conditionalSect ::= includeSect | ignoreSect
//	[62] includeSect ::= '<![' S? 'INCLUDE' S? '[' extSubsetDecl ']]>'
//	[63] ignoreSect  ::= '<![' S? 'IGNORE'  S? '[' ignoreSectContents* ']]>'
func (pctx *parserCtx) parseConditionalSections(ctx context.Context) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseConditionalSections")
		defer g.IRelease("END parseConditionalSections")
	}

	cur := pctx.getCursor()
	if cur == nil {
		return ErrPrematureEOF
	}

	// Consume '<![' (3 chars)
	if err := cur.Advance(3); err != nil {
		return err
	}

	pctx.skipBlanks(ctx)

	// Check for PE reference that expands to keyword
	cur = pctx.getCursor()
	if cur != nil && cur.Peek() == '%' {
		if err := pctx.parsePEReference(ctx); err != nil {
			return err
		}
		pctx.skipBlanks(ctx)
	}

	cur = pctx.getCursor()
	if cur == nil {
		return ErrPrematureEOF
	}

	if cur.HasPrefixString("INCLUDE") {
		// Consume 'INCLUDE'
		if err := cur.Advance(7); err != nil {
			return err
		}
		pctx.skipBlanks(ctx)
		cur = pctx.getCursor()
		if cur == nil || cur.Peek() != '[' {
			return ErrConditionalSectionKeyword
		}
		if err := cur.Advance(1); err != nil {
			return err
		}

		// Parse included content until ']]>'
		for {
			pctx.skipBlanks(ctx)
			cur = pctx.getCursor()
			if cur == nil || cur.Done() {
				return ErrConditionalSectionNotFinished
			}

			if cur.Peek() == ']' && cur.PeekN(2) == ']' && cur.PeekN(3) == '>' {
				if err := cur.Advance(3); err != nil {
					return err
				}
				return nil
			}

			if cur.Peek() == '<' && cur.PeekN(2) == '!' && cur.PeekN(3) == '[' {
				if err := pctx.parseConditionalSections(ctx); err != nil {
					return err
				}
				continue
			}

			if err := pctx.parseMarkupDecl(ctx); err != nil {
				return err
			}

			cur = pctx.getCursor()
			if cur != nil && cur.Peek() == '%' {
				if err := pctx.parsePEReference(ctx); err != nil {
					return err
				}
			}
		}
	}

	if cur.HasPrefixString("IGNORE") {
		// Consume 'IGNORE'
		if err := cur.Advance(6); err != nil {
			return err
		}
		pctx.skipBlanks(ctx)
		cur = pctx.getCursor()
		if cur == nil || cur.Peek() != '[' {
			return ErrConditionalSectionKeyword
		}
		if err := cur.Advance(1); err != nil {
			return err
		}

		// Scan char-by-char tracking nesting depth
		depth := 1
		for depth > 0 {
			cur = pctx.getCursor()
			if cur == nil || cur.Done() {
				return ErrConditionalSectionNotFinished
			}

			c := cur.Peek()
			if c == '<' && cur.PeekN(2) == '!' && cur.PeekN(3) == '[' {
				depth++
				if err := cur.Advance(3); err != nil {
					return err
				}
				continue
			}
			if c == ']' && cur.PeekN(2) == ']' && cur.PeekN(3) == '>' {
				depth--
				if err := cur.Advance(3); err != nil {
					return err
				}
				continue
			}
			if err := cur.Advance(1); err != nil {
				return err
			}
		}
		return nil
	}

	return ErrConditionalSectionKeyword
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
	if pdebug.Enabled {
		g := pdebug.Marker("parsePEReference")
		defer g.End()
	}

	cur := pctx.getCursor()
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
		pctx.nbentities++ // number of entities parsed
	*/
	var entity sax.Entity
	if s := pctx.sax; s != nil {
		_ = pctx.fireSAXCallback(ctx, cbGetParameterEntity, &entity, name)
	}

	// GetParameterEntity callback may trigger input switching; bail if EOF.
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
		if err := pctx.warning(ctx, "PEReference: %%%s; not found\n", name); err != nil {
			return err
		}
		pctx.valid = false
		if err := pctx.entityCheck(entity, 0, 0); err != nil {
			return pctx.error(ctx, err)
		}
	} else {
		/*
		 * Internal checking in case the entity quest barfed
		 */
		if etype := entity.EntityType(); etype != enum.InternalParameterEntity && etype != enum.ExternalParameterEntity {
			if err := pctx.warning(ctx, "Internal: %%%s; is not a parameter entity\n", name); err != nil {
				return err
			}
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
			decodedContent, err := pctx.decodeEntities(ctx, entity.Content(), SubstituteBoth)
			if err != nil {
				return fmt.Errorf("failed to decode parameter entity content: %v", err)
			}

			if pdebug.Enabled {
				pdebug.Printf("Decoded parameter entity content: %s", decodedContent)
			}

			// Push the decoded content as new input stream
			pctx.pushInput(strcursor.NewByteCursor(bytes.NewReader([]byte(decodedContent))))

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
func (pctx *parserCtx) parseElementDecl(ctx context.Context) (enum.ElementType, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseElementDecl")
		defer g.IRelease("END parseElementDecl")
	}

	cur := pctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if !cur.ConsumeString("<!ELEMENT") {
		return enum.UndefinedElementType, pctx.error(ctx, ErrInvalidElementDecl)
	}
	startInput := pctx.currentInputID()

	if !isBlankCh(cur.Peek()) {
		return enum.UndefinedElementType, pctx.error(ctx, ErrSpaceRequired)
	}
	pctx.skipBlanks(ctx)

	name, err := pctx.parseName(ctx)
	if err != nil {
		return enum.UndefinedElementType, pctx.error(ctx, err)
	}

	/* XXX WHAT?
	   while ((RAW == 0) && (ctxt->inputNr > 1))
	       xmlPopInput(ctxt);
	*/

	if !isBlankCh(cur.Peek()) {
		return enum.UndefinedElementType, pctx.error(ctx, ErrSpaceRequired)
	}
	pctx.skipBlanks(ctx)

	var etype enum.ElementType
	var content *ElementContent
	if cur.ConsumeString("EMPTY") {
		etype = enum.EmptyElementType
	} else if cur.ConsumeString("ANY") {
		etype = enum.AnyElementType
	} else if cur.Peek() == '(' {
		content, etype, err = pctx.parseElementContentDecl(ctx)
		if err != nil {
			return enum.UndefinedElementType, pctx.error(ctx, err)
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
		return enum.UndefinedElementType, pctx.error(ctx, ErrGtRequired)
	}
	if err := cur.Advance(1); err != nil {
		return enum.UndefinedElementType, err
	}

	if pctx.currentInputID() != startInput {
		return enum.UndefinedElementType, pctx.error(ctx,
			fmt.Errorf("%w: element declaration doesn't start and stop in the same entity", ErrEntityBoundary))
	}

	if s := pctx.sax; s != nil {
		switch err := s.ElementDecl(ctx, name, etype, content); err {
		case nil, sax.ErrHandlerUnspecified:
			// no op
		default:
			return enum.UndefinedElementType, pctx.error(ctx, err)
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

func (pctx *parserCtx) parseElementContentDecl(ctx context.Context) (*ElementContent, enum.ElementType, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseElementContentDecl")
		defer g.IRelease("END parseElementContentDecl")
	}

	cur := pctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if cur.Peek() != '(' {
		return nil, enum.UndefinedElementType, pctx.error(ctx, ErrOpenParenRequired)
	}
	if err := cur.Advance(1); err != nil {
		return nil, enum.UndefinedElementType, err
	}

	if pctx.instate == psEOF {
		return nil, enum.UndefinedElementType, pctx.error(ctx, ErrEOF)
	}

	pctx.skipBlanks(ctx)

	var ec *ElementContent
	var err error
	var etype enum.ElementType
	if cur.HasPrefixString("#PCDATA") {
		ec, err = pctx.parseElementMixedContentDecl(ctx)
		if err != nil {
			return nil, enum.UndefinedElementType, pctx.error(ctx, err)
		}
		etype = enum.MixedElementType
	} else {
		ec, err = pctx.parseElementChildrenContentDeclPriv(ctx, 0)
		if err != nil {
			return nil, enum.UndefinedElementType, pctx.error(ctx, err)
		}
		etype = enum.ElementElementType
	}

	pctx.skipBlanks(ctx)
	return ec, etype, nil
}

func (pctx *parserCtx) parseElementMixedContentDecl(ctx context.Context) (*ElementContent, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseElementMixedContentDecl")
		defer g.IRelease("END parseElementMixedContentDecl")
	}

	cur := pctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if !cur.ConsumeString("#PCDATA") {
		return nil, pctx.error(ctx, ErrPCDATARequired)
	}
	startInput := pctx.currentInputID()

	pctx.skipBlanks(ctx)

	if cur.Peek() == ')' {
		if pctx.valid && pctx.currentInputID() != startInput {
			_ = pctx.warning(ctx, "element content declaration doesn't start and stop in the same entity\n")
			pctx.valid = false
		}
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
			n.c1, err = pctx.doc.CreateElementContent(elem, ElementContentElement)
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
		if pctx.valid && pctx.currentInputID() != startInput {
			_ = pctx.warning(ctx, "element content declaration doesn't start and stop in the same entity\n")
			pctx.valid = false
		}
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
 * NOTE(validation): Parameter-entity replacement text must be properly nested
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
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseElementChildrenContentDeclPriv(%d)", depth)
		defer g.IRelease("END parseElementChildrenContentDeclPriv(%d)", depth)
	}

	maxDepth := 128
	if pctx.options.IsSet(ParseHuge) {
		maxDepth = 2048
	}
	if depth > maxDepth {
		return nil, fmt.Errorf("xmlParseElementChildrenContentDecl : depth %d too deep", depth)
	}
	startInput := pctx.currentInputID()

	var curelem *ElementContent
	var retelem *ElementContent
	pctx.skipBlanks(ctx)
	cur := pctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if cur.Peek() == '(' {
		if err := cur.Advance(1); err != nil {
			return nil, err
		}
		pctx.skipBlanks(ctx)
		var err error
		retelem, err = pctx.parseElementChildrenContentDeclPriv(ctx, depth+1)
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

	// Closure used to avoid duplicating choice/seq content creation logic.
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
	if pctx.valid && pctx.currentInputID() != startInput {
		_ = pctx.warning(ctx, "element content declaration doesn't start and stop in the same entity\n")
		pctx.valid = false
	}

	c := cur.Peek()
	switch c {
	case '?':
		// Guard against nil (empty content model edge case).
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
		if err := cur.Advance(1); err != nil {
			return nil, err
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
		if err := cur.Advance(1); err != nil {
			return nil, err
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
	cur := pctx.getCursor()
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
	if pdebug.Enabled {
		g := pdebug.IPrintf("START decodeEntitites (%s)", s)
		defer func() {
			g.IRelease("END decodeEntities ('%s' -> '%s')", s, ret)
		}()
	}
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
		pdebug.Printf("s[0] -> %c", s[0])
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
			if ent == nil {
				// Entity not found (undeclared in non-standalone doc with
				// external subset). Write entity name as-is.
				_, _ = out.Write(s[:width])
				s = s[width:]
				continue
			}
			if err := pctx.entityCheck(ent, 0, 0); err != nil {
				return "", err
			}

			if ent.EntityType() == enum.InternalPredefinedEntity {
				if len(ent.Content()) == 0 {
					return "", errors.New("predefined entity has no content")
				}
				_, _ = out.Write(ent.Content())
			} else if len(ent.Content()) != 0 {
				rep, err := pctx.decodeEntitiesInternal(ctx, ent.Content(), what, depth+1)
				if err != nil {
					return "", err
				}
				if err := pctx.entityCheck(ent, len(rep), 0); err != nil {
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
			if err := pctx.entityCheck(ent, len(rep), 0); err != nil {
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
	if pdebug.Enabled {
		g := pdebug.Marker("parseEntityValue")
		defer g.End()
	}

	pctx.instate = psEntityValue

	literal, err := pctx.parseQuotedText(func(qch rune) (string, error) {
		return pctx.parseEntityValueInternal(ctx, qch)
	})
	if err != nil {
		return "", "", pctx.error(ctx, err)
	}

	val, err := pctx.decodeEntities(ctx, []byte(literal), SubstitutePERef)
	if err != nil {
		return "", "", pctx.error(ctx, err)
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
func (pctx *parserCtx) parseEntityDecl(ctx context.Context) error {
	if pdebug.Enabled {
		g := pdebug.Marker("parseEntityDecl")
		defer g.End()
	}

	cur := pctx.getCursor()
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
	var hasOrig bool // true when parseEntityValue was called (libxml2: orig != NULL)

	if isParameter {
		if pdebug.Enabled {
			pdebug.Printf("Found parameter entity")
		}

		if c := cur.Peek(); c == '"' || c == '\'' {
			if pdebug.Enabled {
				pdebug.Printf("parseEntityDecl, isParameter = true, calling parseEntityValue")
			}
			literal, value, err = pctx.parseEntityValue(ctx)
			hasOrig = true
			if pdebug.Enabled {
				pdebug.Printf("entity declaration '%s' -> '%s'", name, value)
			}

			if err == nil {
				switch err := pctx.fireSAXCallback(ctx, cbEntityDecl, name, value); err {
				case nil, sax.ErrHandlerUnspecified:
					// no op
				default:
					return pctx.error(ctx, err)
				}
			}
		} else {
			if pdebug.Enabled {
				pdebug.Printf("Attempting to parse external ID")
			}
			literal, uri, err = pctx.parseExternalID(ctx)
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
						switch err := s.EntityDecl(ctx, name, enum.ExternalParameterEntity, literal, uri, ""); err {
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
		if pdebug.Enabled {
			pdebug.Printf("Found entity")
		}
		if c := cur.Peek(); c == '"' || c == '\'' {
			literal, value, err = pctx.parseEntityValue(ctx)
			hasOrig = true
			if err == nil {
				if s := pctx.sax; s != nil {
					switch err := s.EntityDecl(ctx, name, enum.InternalGeneralEntity, "", "", value); err {
					case nil, sax.ErrHandlerUnspecified:
						// no op
					default:
						return pctx.error(ctx, err)
					}
				}
			}
		} else {
			// parseExternalID returns (systemURI, publicID, error)
			literal, uri, err = pctx.parseExternalID(ctx)
			if err != nil {
				return pctx.error(ctx, ErrValueRequired)
			}

			// literal = system URI, uri = public ID (from parseExternalID return convention)
			if literal != "" {
				u, err := url.Parse(literal)
				if err != nil {
					return pctx.error(ctx, err)
				}

				if u.Fragment != "" {
					return pctx.error(ctx, errors.New("err uri fragment"))
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
					// NDATA entity: call UnparsedEntityDecl, not EntityDecl.
					// libxml2: ctxt->sax->unparsedEntityDecl(ctxt->userData, name, literal, URI, ndata)
					switch err := s.UnparsedEntityDecl(ctx, name, uri, literal, ndata); err {
					case nil, sax.ErrHandlerUnspecified:
						// no op
					default:
						return pctx.error(ctx, err)
					}
				}
			} else {
				if s := pctx.sax; s != nil {
					if pdebug.Enabled {
						pdebug.Printf("Calling s.EntityDecl with %s -> %s", name, literal)
					}
					// External parsed entity: publicID=uri, systemID=literal
					switch err := s.EntityDecl(ctx, name, enum.ExternalGeneralParsedEntity, uri, literal, ""); err {
					case nil, sax.ErrHandlerUnspecified:
						// no op
					default:
						return pctx.error(ctx, err)
					}
				}
			}
		}
	}

	pdebug.Printf("============================")

	pctx.skipBlanks(ctx)
	if cur.Peek() != '>' {
		return pctx.error(ctx, errors.New("entity not terminated"))
	}
	if err := cur.Advance(1); err != nil {
		return err
	}

	// Ugly mechanism to save the raw entity value.
	// In libxml2, this block only runs when orig != NULL, i.e. when
	// parseEntityValue was called (internal entities only).
	if hasOrig {
		var curent sax.Entity
		if isParameter {
			if s := pctx.sax; s != nil {
				curent, _ = s.GetParameterEntity(ctx, name)
			}
		} else {
			if s := pctx.sax; s != nil {
				curent, _ = s.GetEntity(ctx, name)
				if curent == nil {
					e, _ := pctx.getEntity(name)
					curent = e
				}
			}
		}
		if curent != nil {
			if ent, ok := curent.(*Entity); ok && ent != nil && ent.orig == "" {
				ent.SetOrig(literal)
			}
		}
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
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseNotationType")
		defer g.IRelease("END parseNotationType")
	}

	cur := pctx.getCursor()
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
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseEnumerationType")
		defer g.IRelease("END parseEnumerationType")
	}

	cur := pctx.getCursor()
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
		name, err := pctx.parseNmtoken()
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
func (pctx *parserCtx) parseEnumeratedType(ctx context.Context) (enum.AttributeType, Enumeration, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseEnumeratedType")
		defer g.IRelease("END parseEnumeratedType")
	}

	cur := pctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if cur.ConsumeString("NOTATION") {
		if !isBlankCh(cur.Peek()) {
			return enum.AttrInvalid, nil, pctx.error(ctx, ErrSpaceRequired)
		}
		pctx.skipBlanks(ctx)
		tree, err := pctx.parseNotationType(ctx)
		if err != nil {
			return enum.AttrInvalid, nil, pctx.error(ctx, err)
		}

		return enum.AttrNotation, tree, nil
	}

	tree, err := pctx.parseEnumerationType(ctx)
	if err != nil {
		return enum.AttrInvalid, tree, pctx.error(ctx, err)
	}
	return enum.AttrEnumeration, tree, nil
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
func (pctx *parserCtx) parseAttributeType(ctx context.Context) (enum.AttributeType, Enumeration, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseAttributeType")
		defer g.IRelease("END parseAttributeType")
	}

	cur := pctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if cur.ConsumeString("CDATA") {
		return enum.AttrCDATA, nil, nil
	}
	if cur.ConsumeString("IDREFS") {
		return enum.AttrIDRefs, nil, nil
	}
	if cur.ConsumeString("IDREF") {
		return enum.AttrIDRef, nil, nil
	}
	if cur.ConsumeString("ID") {
		return enum.AttrID, nil, nil
	}
	if cur.ConsumeString("ENTITY") {
		return enum.AttrEntity, nil, nil
	}
	if cur.ConsumeString("ENTITIES") {
		return enum.AttrEntities, nil, nil
	}
	if cur.ConsumeString("NMTOKENS") {
		return enum.AttrNmtokens, nil, nil
	}
	if cur.ConsumeString("NMTOKEN") {
		return enum.AttrNmtoken, nil, nil
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
func (pctx *parserCtx) parseDefaultDecl(ctx context.Context) (deftype enum.AttributeDefault, defvalue string, err error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseDefaultDecl")
		defer func() {
			g.IRelease("END parseDefaultDecl (deftype = %d, defvalue = '%s')", deftype, defvalue)
		}()
	}

	deftype = enum.AttrDefaultNone
	cur := pctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if cur.ConsumeString("#REQUIRED") {
		deftype = enum.AttrDefaultRequired
		return
	}
	if cur.ConsumeString("#IMPLIED") {
		deftype = enum.AttrDefaultImplied
		return
	}

	if cur.ConsumeString("#FIXED") {
		deftype = enum.AttrDefaultFixed
		if !isBlankCh(cur.Peek()) {
			deftype = enum.AttrDefaultInvalid
			err = pctx.error(ctx, ErrSpaceRequired)
			return
		}
		pctx.skipBlanks(ctx)
	}

	// XML spec [10] AttValue ::= '"' ... '"' | "'" ... "'" — always quoted.
	defvalue, err = pctx.parseQuotedText(func(qch rune) (string, error) {
		s, _, err := pctx.parseAttributeValueInternal(ctx, qch, false)
		return s, err
	})
	if err != nil {
		deftype = enum.AttrDefaultInvalid
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
		if v == enum.AttrCDATA {
			if pdebug.Enabled {
				pdebug.Printf("removing %s from special attribute set", k)
			}
			delete(ctx.attsSpecial, k)
		}
	}
}

func (ctx *parserCtx) addSpecialAttribute(elemName, attrName string, typ enum.AttributeType) {
	if typ == enum.AttrID && ctx.loadsubset.IsSet(SkipIDs) {
		return
	}
	key := elemName + ":" + attrName
	if pdebug.Enabled {
		g := pdebug.IPrintf("START addSpecialAttribute(%s, %d)", key, typ)
		defer g.IRelease("END addSpecialAttribute")
	}
	ctx.attsSpecial[key] = typ
}

func (ctx *parserCtx) lookupSpecialAttribute(elemName, attrName string) (enum.AttributeType, bool) {
	// Fast path: most documents have no DTD attribute declarations.
	if len(ctx.attsSpecial) == 0 {
		return 0, false
	}
	key := elemName + ":" + attrName
	if pdebug.Enabled {
		g := pdebug.IPrintf("START lookupSpecialAttribute(%s)", key)
		defer g.IRelease("END lookupSpecialAttribute")
	}
	v, ok := ctx.attsSpecial[key]
	return v, ok
}

func (ctx *parserCtx) addAttributeDecl(dtd *DTD, elem string, name string, prefix string, atype enum.AttributeType, def enum.AttributeDefault, defvalue string, tree Enumeration) (attr *AttributeDecl, err error) {
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
	case enum.AttrCDATA, enum.AttrID, enum.AttrIDRef, enum.AttrIDRefs, enum.AttrEntity, enum.AttrEntities, enum.AttrNmtoken, enum.AttrNmtokens, enum.AttrEnumeration, enum.AttrNotation:
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
	// already defined in the internal subset. If so, silently skip it
	// (the internal subset declaration takes precedence per XML spec).
	if doc := dtd.doc; doc != nil && doc.extSubset == dtd && doc.intSubset != nil && len(doc.intSubset.attributes) > 0 {
		if _, ok := doc.intSubset.LookupAttribute(name, prefix, elem); ok {
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

	// NOTE: Multiple-ID-per-element check and namespace-default attribute
	// ordering are handled post-parse via validateDocument() when
	// ParseDTDValid is set.

	if err := dtd.AddChild(attr); err != nil {
		return nil, err
	}
	return attr, nil
}

func (ctx *parserCtx) addAttributeDefault(elemName, attrName, defaultValue string) {
	// detect attribute redefinition
	if _, ok := ctx.lookupSpecialAttribute(elemName, attrName); ok {
		return
	}

	// See xmlAddDefAttrs for details of what the original code is doing.
	// Use a slice to preserve declaration order (Go maps randomize iteration).
	existing := ctx.attsDefault[elemName]
	for _, a := range existing {
		if a.Name() == attrName {
			return // already registered
		}
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
		return
	}

	attr.SetDefault(true)
	if decl := lookupAttributeDecl(ctx.doc, local, prefix, elemName); decl != nil {
		attr.SetAType(decl.AType())
	}
	ctx.attsDefault[elemName] = append(existing, attr)

	/*
	   	hmm, let's think about this when the time comes
	       if (ctxt->external)
	           defaults->values[5 * defaults->nbAttrs + 4] = BAD_CAST "external";
	       else
	           defaults->values[5 * defaults->nbAttrs + 4] = NULL;
	*/
}

func (ctx *parserCtx) lookupAttributeDefault(elemName string) ([]*Attribute, bool) {
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
func (pctx *parserCtx) parseAttributeListDecl(ctx context.Context) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseAttributeListDecl")
		defer g.IRelease("END parseAttributeListDecl")
	}

	cur := pctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if !cur.ConsumeString("<!ATTLIST") {
		return nil
	}
	startInput := pctx.currentInputID()

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

		typ, tree, err := pctx.parseAttributeType(ctx)
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

		if typ != enum.AttrCDATA && def != enum.AttrDefaultInvalid {
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
			switch err := s.AttributeDecl(ctx, elemName, attrName, typ, def, defvalue, tree); err {
			case nil, sax.ErrHandlerUnspecified:
				// no op
			default:
				return pctx.error(ctx, err)
			}
		}

		if defvalue != "" && def != enum.AttrDefaultImplied && def != enum.AttrDefaultRequired {
			pctx.addAttributeDefault(elemName, attrName, defvalue)
		}

		// note: in libxml2, this is only triggered when SAX2 is enabled.
		// as we only support SAX2, we just register it regardless
		pctx.addSpecialAttribute(elemName, attrName, typ)

		if cur.Peek() == '>' {
			if pctx.currentInputID() != startInput {
				_ = pctx.warning(ctx, "attribute list declaration doesn't start and stop in the same entity\n")
				pctx.valid = false
			}
			if err := cur.Advance(1); err != nil {
				return err
			}
			break
		}
	}
	return nil
}

// parseNotationDecl parses a notation declaration per XML spec production [82]:
//
//	NotationDecl ::= '<!NOTATION' S Name S (ExternalID | PublicID) S? '>'
//	PublicID      ::= 'PUBLIC' S PubidLiteral
func (pctx *parserCtx) parseNotationDecl(ctx context.Context) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseNotationDecl")
		defer g.IRelease("END parseNotationDecl")
	}

	cur := pctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if !cur.ConsumeString("<!NOTATION") {
		return pctx.error(ctx, errors.New("<!NOTATION not started"))
	}

	if !pctx.skipBlanks(ctx) {
		return pctx.error(ctx, ErrSpaceRequired)
	}

	name, err := pctx.parseName(ctx)
	if err != nil {
		return pctx.error(ctx, err)
	}

	if !pctx.skipBlanks(ctx) {
		return pctx.error(ctx, ErrSpaceRequired)
	}

	// Parse ExternalID or PublicID.
	// ExternalID = 'SYSTEM' S SystemLiteral | 'PUBLIC' S PubidLiteral S SystemLiteral
	// PublicID   = 'PUBLIC' S PubidLiteral
	// parseExternalID handles both SYSTEM and PUBLIC. For PUBLIC without
	// a system literal, it returns ("", publicID, nil).
	systemID, publicID, err := pctx.parseExternalID(ctx)
	if err != nil {
		return pctx.error(ctx, err)
	}

	pctx.skipBlanks(ctx)

	if cur.Peek() != '>' {
		return pctx.error(ctx, errors.New("'>' required to close <!NOTATION"))
	}
	if err := cur.Advance(1); err != nil {
		return err
	}

	// Wire SAX2 NotationDecl callback.
	if s := pctx.sax; s != nil {
		switch err := s.NotationDecl(ctx, name, publicID, systemID); err {
		case nil, sax.ErrHandlerUnspecified:
		default:
			return pctx.error(ctx, err)
		}
	}

	return nil
}

func (pctx *parserCtx) parseSystemLiteral(ctx context.Context, qch rune) (string, error) {
	cur := pctx.getCursor()
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
		if err := cur.Advance(i); err != nil {
			return "", pctx.error(ctx, err)
		}
	}
	return buf.String(), nil
}

func (pctx *parserCtx) parsePubidLiteral(ctx context.Context, qch rune) (string, error) {
	cur := pctx.getCursor()
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
		if err := cur.Advance(i); err != nil {
			return "", pctx.error(ctx, err)
		}
	}
	return buf.String(), nil
}

// parseExternalID parses an external ID (SYSTEM or PUBLIC identifier).
// Returns (systemURI, publicID, error).
func (pctx *parserCtx) parseExternalID(ctx context.Context) (string, string, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseExternalID")
		defer g.IRelease("END parseExternalID")
	}

	cur := pctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}

	if cur.HasPrefixString("SYSTEM") {
		if err := cur.Advance(6); err != nil {
			return "", "", err
		}
		if !isBlankCh(cur.Peek()) {
			return "", "", pctx.error(ctx, ErrSpaceRequired)
		}
		pctx.skipBlanks(ctx)
		uri, err := pctx.parseQuotedText(func(qch rune) (string, error) {
			return pctx.parseSystemLiteral(ctx, qch)
		})
		if err != nil {
			return "", "", pctx.error(ctx, errors.New("system URI required"))
		}
		return uri, "", nil
	} else if cur.HasPrefixString("PUBLIC") {
		if err := cur.Advance(6); err != nil {
			return "", "", err
		}
		if !isBlankCh(cur.Peek()) {
			return "", "", pctx.error(ctx, ErrSpaceRequired)
		}
		pctx.skipBlanks(ctx)
		publicID, err := pctx.parseQuotedText(func(qch rune) (string, error) {
			return pctx.parsePubidLiteral(ctx, qch)
		})
		if err != nil {
			return "", "", pctx.error(ctx, errors.New("public ID required"))
		}
		if !isBlankCh(cur.Peek()) {
			// No system literal follows
			return "", publicID, nil
		}
		pctx.skipBlanks(ctx)
		if c := cur.Peek(); c != '\'' && c != '"' {
			// No system literal follows
			return "", publicID, nil
		}
		uri, err := pctx.parseQuotedText(func(qch rune) (string, error) {
			return pctx.parseSystemLiteral(ctx, qch)
		})
		if err != nil {
			return "", "", pctx.error(ctx, errors.New("system URI required"))
		}
		return uri, publicID, nil
	}
	return "", "", nil
}

func (pctx *parserCtx) parseExternalEntityPrivate(ctx context.Context, uri, externalID string) (Node, error) {
	if pctx.options.IsSet(ParseNoXXE) {
		return nil, nil
	}

	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseExternalEntityPrivate(uri=%s, externalID=%s)", uri, externalID)
		defer g.IRelease("END parseExternalEntityPrivate")
	}

	pctx.depth++
	defer func() { pctx.depth-- }()

	if pctx.depth > 40 {
		return nil, errors.New("entity loop")
	}

	// Resolve the external entity via the SAX ResolveEntity callback.
	// The application must provide an implementation that returns the
	// entity content as a ParseInput (io.Reader + URI).
	var input sax.ParseInput
	if s := pctx.sax; s != nil {
		resolved, err := s.ResolveEntity(ctx, externalID, uri)
		switch err {
		case nil:
			input = resolved
		case sax.ErrHandlerUnspecified:
			// no handler registered — cannot resolve
		default:
			return nil, pctx.error(ctx, err)
		}
	}

	if input == nil {
		return nil, fmt.Errorf("cannot resolve external entity (URI=%s, publicID=%s)", uri, externalID)
	}

	// Read all content from the resolved input
	content, err := io.ReadAll(input)
	if err != nil {
		return nil, pctx.error(ctx, fmt.Errorf("reading external entity: %w", err))
	}

	newctx := &parserCtx{}
	if err := newctx.init(nil, bytes.NewReader(content)); err != nil {
		return nil, err
	}
	defer func() {
		if err := newctx.release(); err != nil {
			if pdebug.Enabled {
				pdebug.Printf("newctx.release() failed: %s", err)
			}
		}
	}()

	if pctx.doc == nil {
		pctx.doc = NewDocument("1.0", "", StandaloneExplicitNo)
	}

	// Save and restore the document's children
	fc := pctx.doc.FirstChild()
	lc := pctx.doc.LastChild()
	setFirstChild(pctx.doc, nil)
	setLastChild(pctx.doc, nil)
	defer func() {
		setFirstChild(pctx.doc, fc)
		setLastChild(pctx.doc, lc)
	}()
	newctx.doc = pctx.doc
	newctx.sax = pctx.sax
	newctx.attsDefault = pctx.attsDefault
	newctx.options = pctx.options
	newctx.depth = pctx.depth + 1
	newctx.external = true
	// Derive option-dependent flags that init() only sets when p != nil.
	newctx.replaceEntities = pctx.replaceEntities
	newctx.loadsubset = pctx.loadsubset
	if pctx.elem != nil {
		for _, ns := range collectInScopeNamespaces(pctx.elem) {
			newctx.pushNS(ns.Prefix(), ns.URI())
		}
	}

	// External parsed entities may have a text declaration (XML decl without
	// standalone). Try to detect and parse encoding.
	if newctx.encoding == "" {
		if enc, err := newctx.detectEncoding(); err == nil {
			newctx.detectedEncoding = enc
		}
	}

	// Enrich context with the new parserCtx for SAX callbacks.
	innerCtx := withParserCtx(ctx, newctx)
	innerCtx = sax.WithDocumentLocator(innerCtx, newctx)
	innerCtx = context.WithValue(innerCtx, stopFuncKey{}, newctx.stop)

	bcur := newctx.getByteCursor()
	if bcur != nil && looksLikeXMLDecl(bcur) {
		if err := newctx.parseXMLDecl(innerCtx); err != nil {
			return nil, err
		}
	}

	if err := newctx.switchEncoding(); err != nil {
		return nil, err
	}

	// Create a dummy root node and parse the content
	newRoot, err := newctx.doc.CreateElement("pseudoroot")
	if err != nil {
		return nil, pctx.error(ctx, err)
	}
	newctx.pushNode(newRoot)
	newctx.elem = newRoot
	if err := newctx.doc.AddChild(newRoot); err != nil {
		return nil, err
	}
	if err := newctx.parseContent(innerCtx); err != nil {
		return nil, err
	}

	if child := newctx.doc.FirstChild(); child != nil {
		if grandchild := child.FirstChild(); grandchild != nil {
			for e := grandchild; e != nil; e = e.NextSibling() {
				e.SetTreeDoc(pctx.doc)
				e.SetParent(nil)
				if uri != "" {
					// Track the entity base URI on each top-level node
					// from the external entity so that base-uri() can
					// resolve it even when ParseNoBaseFix suppresses
					// synthetic xml:base attributes.
					e.baseDocNode().entityBaseURI = uri
					if !pctx.options.IsSet(ParseNoBaseFix) {
						if elem, ok := e.(*Element); ok {
							if _, exists := elem.GetAttributeNS("base", XMLNamespace); !exists {
								_ = elem.SetAttributeNS("base", uri, newNamespace("xml", XMLNamespace))
							}
						}
					}
				}
			}
			return grandchild, nil
		}
	}

	return nil, ErrParseSucceeded
}

var ErrParseSucceeded = errors.New("parse succeeded")

func (pctx *parserCtx) parseBalancedChunkInternal(ctx context.Context, chunk []byte) (Node, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseBalancedChunkInternal")
		defer g.IRelease("END parseBalancedChunkInternal")
	}

	pctx.depth++
	defer func() { pctx.depth-- }()

	if pctx.depth > 40 {
		return nil, errors.New("entity loop")
	}

	newctx := &parserCtx{}
	if err := newctx.init(nil, bytes.NewReader(chunk)); err != nil {
		return nil, err
	}
	defer func() {
		if err := newctx.release(); err != nil {
			if pdebug.Enabled {
				pdebug.Printf("newctx.release() failed: %s", err)
			}
		}
	}()

	if pctx.doc == nil {
		pctx.doc = NewDocument("1.0", "", StandaloneExplicitNo)
	}

	// save the document's children
	fc := pctx.doc.FirstChild()
	lc := pctx.doc.LastChild()
	setFirstChild(pctx.doc, nil)
	setLastChild(pctx.doc, nil)
	defer func() {
		setFirstChild(pctx.doc, fc)
		setLastChild(pctx.doc, lc)
	}()
	newctx.doc = pctx.doc
	newctx.sax = pctx.sax
	newctx.attsDefault = pctx.attsDefault
	newctx.depth = pctx.depth + 1
	if pctx.elem != nil {
		for _, ns := range collectInScopeNamespaces(pctx.elem) {
			newctx.pushNS(ns.Prefix(), ns.URI())
		}
	}
	// Propagate entity amplification tracking from parent context,
	// and collect the accumulated counter back when done.
	newctx.sizeentcopy = pctx.sizeentcopy
	newctx.inputSize = pctx.inputSize
	newctx.maxAmpl = pctx.maxAmpl
	defer func() { pctx.sizeentcopy = newctx.sizeentcopy }()

	// create a dummy node
	newRoot, err := newctx.doc.CreateElement("pseudoroot")
	if err != nil {
		return nil, pctx.error(ctx, err)
	}
	newctx.pushNode(newRoot)
	newctx.elem = newRoot // Set the current element context
	if err := newctx.doc.AddChild(newRoot); err != nil {
		return nil, err
	}
	if err := newctx.switchEncoding(); err != nil {
		return nil, err
	}
	// Enrich context with the new parserCtx for SAX callbacks.
	innerCtx := withParserCtx(ctx, newctx)
	innerCtx = sax.WithDocumentLocator(innerCtx, newctx)
	innerCtx = context.WithValue(innerCtx, stopFuncKey{}, newctx.stop)
	if err := newctx.parseContent(innerCtx); err != nil {
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
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseReference")
		defer g.IRelease("END parseReference")
	}

	cur := pctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if cur.Peek() != '&' {
		return pctx.error(ctx, ErrAmpersandRequired)
	}

	// "&#..." CharRef
	if cur.PeekN(2) == '#' {
		v, err := pctx.parseCharRef()
		if err != nil {
			return pctx.error(ctx, err)
		}
		var buf [utf8.UTFMax]byte
		l := utf8.EncodeRune(buf[:], v)
		b := buf[:l]
		if s := pctx.sax; s != nil {
			if err := pctx.deliverCharacters(ctx, s.Characters, b); err != nil {
				return err
			}
		}
		return nil
	}

	// &...
	ent, err := pctx.parseEntityRef(ctx)
	if err != nil {
		return pctx.error(ctx, err)
	}
	if ent == nil {
		return nil
	}
	// if !pctx.wellFormed { return } ??

	wasChecked := ent.checked

	// special case for predefined entities
	if ent.name == "" || ent.EntityType() == enum.InternalPredefinedEntity {
		if ent.content == "" {
			return nil
		}
		if s := pctx.sax; s != nil {
			if err := pctx.deliverCharacters(ctx, s.Characters, []byte(ent.content)); err != nil {
				return err
			}
		}
		return nil
	}

	// Entity amplification guard: account for entity content being expanded.
	if err := pctx.entityCheck(ent, len(ent.content), 0); err != nil {
		return pctx.error(ctx, err)
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
	if (wasChecked == 0 || (ent.firstChild == nil && pctx.options.IsSet(ParseNoEnt))) && (ent.EntityType() != enum.ExternalGeneralParsedEntity || pctx.options.IsSet(ParseNoEnt|ParseDTDValid)) {
		sizeBefore := pctx.sizeentcopy

		if ent.EntityType() == enum.InternalGeneralEntity {
			parsedEnt, err = pctx.parseBalancedChunkInternal(ctx, []byte(ent.Content()))
			switch err {
			case nil, ErrParseSucceeded:
				// may not have generated nodes, but parse was successful
			default:
				return err
			}
		} else if ent.EntityType() == enum.ExternalGeneralParsedEntity {
			parsedEnt, err = pctx.parseExternalEntityPrivate(ctx, ent.uri, ent.externalID)
			switch err {
			case nil, ErrParseSucceeded:
				// may not have generated nodes, but parse was successful
			default:
				return err
			}
		} else {
			return errors.New("invalid entity type")
		}

		// Mark entity as checked after first parse (libxml2: ent->checked = 2).
		// Also record the cumulative expansion cost so subsequent references
		// to this entity account for the full recursive expansion.
		if ent.checked == 0 {
			ent.checked = 2
		}
		ent.expandedSize = pctx.sizeentcopy - sizeBefore + int64(len(ent.content))
		ent.MarkChecked()

		// Store parsed nodes as entity children (mirrors libxml2).
		// This populates ent.firstChild so subsequent references can
		// reuse the parsed tree without re-parsing.
		if parsedEnt != nil && ent.firstChild == nil {
			for n := parsedEnt; n != nil; {
				next := n.NextSibling()
				// Detach from the old sibling chain before adding
				// to the entity, otherwise addChild/addSibling will
				// follow stale NextSibling links and loop.
				n.SetNextSibling(nil)
				n.SetPrevSibling(nil)
				n.SetParent(nil)
				n.SetTreeDoc(pctx.doc)
				_ = ent.AddChild(n)
				n = next
			}
		}
	}

	// Now that the entity content has been gathered
	// provide it to the application, this can take different forms based
	// on the parsing modes.
	//
	// This block is OUTSIDE the wasChecked==0 condition, matching libxml2's
	// structure: even when wasChecked!=0 (entity already parsed once), SAX
	// mode still needs to re-parse content to generate callbacks.
	if ent.firstChild == nil {
		// Probably running in SAX mode and the callbacks don't
		// build the entity content. So unless we already went
		// though parsing for first checking go though the entity
		// content to generate callbacks associated to the entity
		if wasChecked != 0 {
			if ent.EntityType() == enum.InternalGeneralEntity {
				parsedEnt, err = pctx.parseBalancedChunkInternal(ctx, []byte(ent.Content()))
				_ = parsedEnt
				switch err {
				case nil, ErrParseSucceeded:
					// may not have generated nodes, but parse was successful
				default:
					return err
				}
			} else if ent.EntityType() == enum.ExternalGeneralParsedEntity {
				parsedEnt, err = pctx.parseExternalEntityPrivate(ctx, ent.uri, ent.externalID)
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
			switch err := s.Reference(ctx, ent.name); err {
			case nil, sax.ErrHandlerUnspecified:
				// no op
			default:
				return err
			}
		}
		return nil
	}

	// Entity has children (from prior parse). When replaceEntities is true,
	// copy the entity's children into the current element (mirrors libxml2's
	// entity substitution). When replaceEntities is false, emit a Reference
	// SAX event instead.
	if s := pctx.sax; s != nil && !pctx.replaceEntities {
		switch err := s.Reference(ctx, ent.name); err {
		case nil, sax.ErrHandlerUnspecified:
			// no op
		default:
			return err
		}
		return nil
	}

	if pctx.replaceEntities {
		// Track the entity URI so the TreeBuilder can set entityBaseURI
		// on nodes created from this external entity.
		savedEntityURI := pctx.currentEntityURI
		if ent.EntityType() == enum.ExternalGeneralParsedEntity && ent.uri != "" {
			pctx.currentEntityURI = ent.uri
		}
		for n := ent.firstChild; n != nil; n = n.NextSibling() {
			if err := pctx.replayEntityNode(ctx, n); err != nil {
				pctx.currentEntityURI = savedEntityURI
				return err
			}
		}
		pctx.currentEntityURI = savedEntityURI
	}

	return nil
}

func (pctx *parserCtx) replayEntityNode(ctx context.Context, n Node) error {
	if n == nil || pctx.sax == nil {
		return nil
	}

	switch v := n.(type) {
	case *Element:
		namespaces := make([]sax.Namespace, 0, len(v.Namespaces()))
		for _, ns := range v.Namespaces() {
			namespaces = append(namespaces, ns)
		}

		var attrs []sax.Attribute
		for attr := v.properties; attr != nil; attr = attr.NextAttribute() {
			attrs = append(attrs, attr)
		}

		switch err := pctx.sax.StartElementNS(ctx, v.LocalName(), v.Prefix(), v.URI(), namespaces, attrs); err {
		case nil, sax.ErrHandlerUnspecified:
		default:
			return err
		}

		for child := range Children(v) {
			if err := pctx.replayEntityNode(ctx, child); err != nil {
				return err
			}
		}

		switch err := pctx.sax.EndElementNS(ctx, v.LocalName(), v.Prefix(), v.URI()); err {
		case nil, sax.ErrHandlerUnspecified:
			return nil
		default:
			return err
		}
	case *Text:
		return pctx.deliverCharacters(ctx, pctx.sax.Characters, v.Content())
	case *CDATASection:
		switch err := pctx.sax.CDataBlock(ctx, v.Content()); err {
		case nil:
			return nil
		case sax.ErrHandlerUnspecified:
			return pctx.deliverCharacters(ctx, pctx.sax.Characters, v.Content())
		default:
			return err
		}
	case *Comment:
		switch err := pctx.sax.Comment(ctx, v.Content()); err {
		case nil, sax.ErrHandlerUnspecified:
			return nil
		default:
			return err
		}
	case *ProcessingInstruction:
		switch err := pctx.sax.ProcessingInstruction(ctx, v.Name(), string(v.Content())); err {
		case nil, sax.ErrHandlerUnspecified:
			return nil
		default:
			return err
		}
	default:
		return nil
	}
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
		_ = s // silence unused warning fornow
		width++
	}

	r = rune(val)
	if !isXMLCharValue(uint32(val)) {
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
		if ret, err := resolvePredefinedEntity(name); err == nil {
			return ret, nil
		}
	}

	var ret *Entity
	var ok bool
	if ctx.doc == nil {
		return nil, ErrEntityNotFound
	} else if ctx.doc.standalone != 1 {
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

func (pctx *parserCtx) parseStringEntityRef(ctx context.Context, s []byte) (sax.Entity, int, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseStringEntityRef ('%s')", s)
		defer g.IRelease("END parseStringEntityRef")
	}
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

	// Check predefined entities first (amp, lt, gt, apos, quot).
	// libxml2 does this via xmlGetPredefinedEntity before calling the SAX handler,
	// which avoids triggering user-visible getEntity callbacks for predefined entities.
	if predef, perr := resolvePredefinedEntity(name); perr == nil {
		return predef, i, nil
	}

	/*
	 * Ask first SAX for entity resolution, otherwise try the
	 * entities which may have stored in the parser context.
	 */
	if h := pctx.sax; h != nil {
		loadedEnt, err = h.GetEntity(ctx, name)
		if err != nil {
			if pctx.wellFormed {
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
		// Entity not found but cannot flag as error (external subset/PE refs).
		// Emit a warning matching libxml2's xmlWarningMsg behavior.
		if err := pctx.warning(ctx, "Entity '%s' not defined", name); err != nil {
			return nil, 0, err
		}
		return nil, i, nil
	}

	/*
	 * [ WFC: Parsed Entity ]
	 * An entity reference must not contain the name of an
	 * unparsed entity
	 */

	if loadedEnt.EntityType() == enum.ExternalGeneralUnparsedEntity {
		return nil, 0, fmt.Errorf("entity reference to unparsed entity '%s'", name)
	}

	/*
	 * [ WFC: No External Entity References ]
	 * Attribute values cannot contain direct or indirect
	 * entity references to external entities.
	 */
	if pctx.instate == psAttributeValue && loadedEnt.EntityType() == enum.ExternalGeneralParsedEntity {
		return nil, 0, fmt.Errorf("attribute references enternal entity '%s'", name)
	}

	/*
	 * [ WFC: No < in Attribute Values ]
	 * The replacement text of any entity referred to directly or
	 * indirectly in an attribute value (other than "&lt;") must
	 * not contain a <.
	 */
	if pctx.instate == psAttributeValue && len(loadedEnt.Content()) > 0 && loadedEnt.EntityType() == enum.InternalPredefinedEntity && bytes.IndexByte(loadedEnt.Content(), '<') > -1 {
		return nil, 0, fmt.Errorf("'<' in entity '%s' is not allowed in attribute values", name)
	}

	/*
	 * Internal check, no parameter entities here ...
	 */

	switch loadedEnt.EntityType() {
	case enum.InternalParameterEntity, enum.ExternalParameterEntity:
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
		loadedEnt, err = h.GetParameterEntity(ctx, name)
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
		switch loadedEnt.EntityType() {
		case enum.InternalParameterEntity, enum.ExternalParameterEntity:
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

	if isXMLCharValue(uint32(val)) && val <= unicode.MaxRune {
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
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseEntityRef")
		defer func() {
			g.IRelease("END parseEntityRef ent = %#v", ent)
		}()
	}

	cur := pctx.getCursor()
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

	// Clear error from resolvePredefinedEntity before proceeding
	err = nil

	if s := pctx.sax; s != nil {
		// ask the SAX2 handler nicely
		var loadedEnt sax.Entity
		loadedEnt, _ = s.GetEntity(ctx, name)
		if loadedEnt != nil {
			ent = loadedEnt.(*Entity)
			return
		}

		ent, _ = pctx.getEntity(name)
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
			if err := pctx.warning(ctx, "Entity '%s' not defined", name); err != nil {
				return nil, err
			}
			if pctx.inSubset == 0 && pctx.instate != psAttributeValue {
				if s := pctx.sax; s != nil {
					switch err := s.Reference(ctx, name); err {
					case nil, sax.ErrHandlerUnspecified:
						// no op
					default:
						return nil, pctx.error(ctx, err)
					}
				}
			}
			if err := pctx.entityCheck(ent, 0, 0); err != nil {
				return nil, pctx.error(ctx, err)
			}
			pctx.valid = false
			return nil, nil
		}
	} else if ent.entityType == enum.ExternalGeneralUnparsedEntity {
		// [ WFC: Parsed Entity ]
		// An entity reference must not contain the name of an
		// unparsed entity
		return nil, pctx.error(ctx, errors.New("entity reference to unparsed entity"))
	} else if pctx.instate == psAttributeValue && ent.entityType == enum.ExternalGeneralParsedEntity {
		// [ WFC: No External Entity References ]
		// Attribute values cannot contain direct or indirect
		// entity references to external entities.
		return nil, pctx.error(ctx, errors.New("attribute references external entity"))
	} else if pctx.instate == psAttributeValue && ent.entityType != enum.InternalPredefinedEntity {
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
		case enum.InternalParameterEntity:
		case enum.ExternalParameterEntity:
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
func saturatedAdd(a, b int64) int64 {
	if a > math.MaxInt64-b {
		return math.MaxInt64
	}
	return a + b
}

func (ctx *parserCtx) entityCheck(ent sax.Entity, size, replacement int) error {
	if ctx.maxAmpl == 0 {
		return nil
	}

	// For already-checked entities, use the cached expandedSize which
	// includes the full recursive expansion cost.
	if e, ok := ent.(*Entity); ok && e != nil && e.Checked() {
		ctx.sizeentcopy = saturatedAdd(ctx.sizeentcopy, e.expandedSize)
		ctx.sizeentcopy = saturatedAdd(ctx.sizeentcopy, entityFixedCost)
	} else {
		ctx.sizeentcopy = saturatedAdd(ctx.sizeentcopy, int64(size))
		ctx.sizeentcopy = saturatedAdd(ctx.sizeentcopy, entityFixedCost)
	}

	if ctx.sizeentcopy > entityAllowedExpansion {
		consumed := ctx.inputSize
		if consumed == 0 {
			consumed = 1
		}
		if ctx.sizeentcopy/int64(ctx.maxAmpl) > consumed {
			return errors.New("maximum entity amplification factor exceeded")
		}
	}

	return nil
}

func (pctx *parserCtx) handlePEReference(ctx context.Context) error {
	if pdebug.Enabled {
		g := pdebug.Marker("handlePEReference")
		defer g.End()
	}

	cur := pctx.getCursor()
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
			pdebug.Printf("DTD external = %t, inputNr = %d", pctx.external, pctx.inputTab.Len())
		}
		if !pctx.external || pctx.inputTab.Len() == 1 {
			if pdebug.Enabled {
				pdebug.Printf("we're NOT in external DTD, bail out")
			}
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
	if pdebug.Enabled {
		pdebug.Printf("entity name: '%s'", name)
	}

	if cur.Peek() != ';' {
		return ErrSemicolonRequired
	}

	if err := cur.Advance(1); err != nil {
		return err
	}

	var entity sax.Entity
	if s := pctx.sax; s != nil {
		entity, _ = s.GetParameterEntity(ctx, name)
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
		if err := pctx.warning(ctx, "PEReference: %%%s; not found\n", name); err != nil {
			return err
		}
		pctx.valid = false
		if err := pctx.entityCheck(nil, 0, 0); err != nil {
			return pctx.error(ctx, err)
		}
		pdebug.Printf("Should be calling pushInput here")
		/* have no clue what this is for
		   } else if (ctxt->input->free != deallocblankswrapper) {
		           input = xmlNewBlanksWrapperInputStream(ctxt, entity);
		           if (xmlPushInput(ctxt, input) < 0)
		               return;
		*/
	} else {
		switch entity.EntityType() {
		case enum.InternalParameterEntity, enum.ExternalParameterEntity:
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
