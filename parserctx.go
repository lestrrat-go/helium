package helium

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"unicode/utf8"

	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/pool"
	"github.com/lestrrat-go/helium/internal/strcursor"
	"github.com/lestrrat-go/helium/sax"
	"github.com/lestrrat-go/pdebug"
)

//go:generate stringer -type=parserState -output parser_state_gen.go

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
	options parseOption
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
	treeBuilder       *TreeBuilder
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
	baseURI           string          // document base URI for resolving external references
	catalog           CatalogResolver // XML catalog for entity resolution
	elem              *Element        // current context element

	nsTab       nsStack
	nsNrTab     []int // number of ns bindings pushed per element (parallel to nodeTab)
	doc         *Document
	nodeTab     nodeStack
	elemidx     int
	sizeentcopy int64 // cumulative entity expansion bytes (non-entity-specific)
	inputSize   int64 // total input document size
	maxAmpl     int   // max amplification factor (default 5, 0 = disabled via parseHuge)
	// nbentities int
	inputTab         inputStack
	cachedCursor     strcursor.Cursor // cached result of getCursor(); invalidated on push/pop
	stopped          bool
	disableSAX       bool              // suppress SAX callbacks after fatal error in recover mode
	recoverErr       error             // first fatal error saved during recovery
	elemDepth        int               // current element nesting depth
	maxElemDepth     int               // max allowed element nesting depth (0 = unlimited)
	currentEntityURI string            // URI of the external entity currently being replayed (for base-uri tracking)
	nameCache        map[string]string // per-parse string interning for element/attribute names
	charBuf          []byte            // reusable buffer for parseCharDataContent
	attrBuf          []attrData        // reusable attribute scratch buffer for start-tag parsing
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

func (ctx *parserCtx) pushNodeEntry(e nodeEntry) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START pushNode (%s)", e.Name())
		defer g.IRelease("END pushNode")

		if l := ctx.nodeTab.Len(); l <= 0 {
			pdebug.Printf("  (EMPTY node stack)")
		} else {
			for i := range ctx.nodeTab.Stack {
				pdebug.Printf("  %003d: %s", i, ctx.nodeTab.Stack[i].Name())
			}
		}
	}
	ctx.nodeTab.Push(e)
}

func (ctx *parserCtx) peekNode() *nodeEntry {
	return ctx.nodeTab.PeekOne()
}

func (ctx *parserCtx) popNode() (elem *nodeEntry) {
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
				for i := range ctx.nodeTab.Stack {
					pdebug.Printf("  %003d: %s", i, ctx.nodeTab.Stack[i].Name())
				}
			}
		}()
	}
	return ctx.nodeTab.Pop()
}

func (ctx *parserCtx) lookupNamespace(prefix string) string {
	if prefix == lexicon.PrefixXML {
		return lexicon.NamespaceXML
	}
	return ctx.nsTab.Lookup(prefix)
}

func (ctx *parserCtx) release() error { //nolint:unparam // always nil but callers check for future-proofing
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

var bufferPool = pool.New(
	func() *bytes.Buffer { return &bytes.Buffer{} },
	func(b *bytes.Buffer) *bytes.Buffer { b.Reset(); return b },
)

func releaseBuffer(b *bytes.Buffer) {
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

func (ctx *parserCtx) adaptCursor(v any) strcursor.Cursor {
	cur, _ := v.(strcursor.Cursor)
	return cur
}

func (ctx *parserCtx) getCursor() strcursor.Cursor {
	// Fast path: for the common single-input case, the cached cursor remains
	// valid even at EOF and callers can observe that directly via Peek/Done.
	if cc := ctx.cachedCursor; cc != nil {
		if ctx.inputTab.Len() <= 1 {
			return cc
		}
		if !cc.Done() {
			return cc
		}
		// Cached cursor is exhausted and there are more inputs — fall through.
		ctx.cachedCursor = nil
	}
	// Pop exhausted input streams and return the next available cursor
	for ctx.inputTab.Len() > 0 {
		cur := ctx.adaptCursor(ctx.inputTab.PeekOne())
		if cur == nil {
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

func (ctx *parserCtx) popInput() any { //nolint:unparam // return value used for type generality
	ctx.cachedCursor = nil // invalidate cache
	return ctx.inputTab.Pop()
}

func (ctx *parserCtx) currentInputID() any {
	return ctx.inputTab.PeekOne()
}

func (ctx *parserCtx) init(p *parserConfig, in io.Reader) error {
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
		if tb, ok := p.sax.(*TreeBuilder); ok {
			ctx.treeBuilder = tb
		}
		ctx.charBufferSize = p.charBufferSize
		ctx.options = p.options
		ctx.catalog = p.catalog
		if ctx.options.IsSet(parseNoBlanks) {
			ctx.keepBlanks = false
		}
		if ctx.options.IsSet(parsePedantic) {
			ctx.pedantic = true
		}
		if ctx.options.IsSet(parseDTDLoad) {
			ctx.loadsubset.Set(DetectIDs)
		}
		if ctx.options.IsSet(parseDTDAttr) {
			ctx.loadsubset.Set(CompleteAttrs)
		}
		if ctx.options.IsSet(parseNoEnt) {
			ctx.replaceEntities = true
		}
		if ctx.options.IsSet(parseHuge) {
			ctx.maxAmpl = 0
		}
		if ctx.options.IsSet(parseSkipIDs) {
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
			if !pctx.options.IsSet(parseNoWarning) {
				_ = s.Warning(ctx, e)
			}
		default:
			// Fire the SAX Error callback unless parseNoError is set.
			if !pctx.options.IsSet(parseNoError) {
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
	if s := pctx.sax; s != nil && !pctx.options.IsSet(parseNoWarning) {
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
