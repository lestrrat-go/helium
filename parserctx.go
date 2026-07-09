package helium

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"unicode/utf8"

	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/internal/iofs"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/pool"
	"github.com/lestrrat-go/helium/internal/strcursor"
	"github.com/lestrrat-go/helium/sax"
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

const (
	entityAllowedExpansion int64 = 1_000_000 // 1 MB baseline before ratio check
	entityFixedCost        int64 = 20        // fixed byte cost per entity reference
	// externalEntityMaxBytes caps the number of bytes read from a single
	// external parsed entity. Without it, a resolver returning an unbounded
	// source (e.g. SYSTEM "/dev/zero") would be read via io.ReadAll and exhaust
	// memory. 10 MiB is generous for any realistic external entity while
	// blocking the unbounded-read denial-of-service path.
	externalEntityMaxBytes int64 = 10 * 1024 * 1024 // 10 MiB
)

// entityHardCeiling caps total entity expansion even when the ratio check is
// disabled (maxAmpl=0 via [Parser.MaxEntityAmplification](-1)). Without it,
// disabling the ratio check would permit unbounded amplification — a single
// document could expand to many GB of resident memory. 1 GB is permissive
// enough for any realistic XML workload but blocks the unbounded billion-laughs
// path that a hostile document could otherwise exploit. It is a package var
// (not a const) only so tests can lower it to verify the ceiling without
// actually expanding toward 1 GB; production never reassigns it.
var entityHardCeiling int64 = 1_000_000_000 // 1 GB absolute cap, always enforced

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
	// autoEncoding records the Unicode encoding asserted by a real byte-order
	// mark at the document start (UTF-8, UTF-16LE, or UTF-16BE), "" when no BOM
	// was consumed. A declared encoding that contradicts it is a fatal error
	// (XML §4.3.3); see checkBOMEncodingConflict.
	autoEncoding string
	// declaredEncoding is the EncName parsed from an XML/Text declaration,
	// recorded UNCONDITIONALLY at the leaf EncName parsers (parseEncodingName /
	// parseEncodingDeclFromCursor) — independent of IgnoreEncoding (which
	// suppresses the decoder switch and erases ctx.encoding) and LenientXMLDecl
	// (which relaxes the declaration parse). checkBOMEncodingConflict reads it
	// immediately after the document entity's declaration is parsed, so the BOM
	// well-formedness check fires regardless of those knobs.
	declaredEncoding string
	in               io.Reader
	rawInput         []byte // original bytes, used for EBCDIC encoding detection
	// ebcdicStream marks an EBCDIC document read from a streaming io.Reader: in
	// that mode rawInput holds only a bounded sniff prefix (enough for
	// ExtractEBCDICEncoding), and the live cursor over the prefix+remainder
	// stream is decoded in place rather than being reset from rawInput (which
	// would otherwise require buffering the whole — possibly unbounded — input).
	ebcdicStream bool
	// ebcdicConsumed counts the bytes pulled from the underlying reader on the
	// EBCDIC streaming path. In ebcdicStream mode rawInput holds only a bounded
	// sniff prefix, so init cannot seed inputSize with the real document size;
	// the entity-amplification guard (entityCheckLimits) instead compares against
	// this live consumed-byte count so a legitimate large EBCDIC document whose
	// internal entity is referenced once is not falsely rejected. nil on every
	// non-EBCDIC path (where inputSize already reflects the real/known size).
	ebcdicConsumed *countingReader
	nbread         int
	instate        parserState
	keepBlanks     bool
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
	// dtdInputFloor is the input-stack depth of the external subset's own base
	// cursor (the pushed DTD buffer). skipBlanksPE expands parameter-entity
	// references inside/adjacent to markup declarations by pushing their padded
	// replacement text and crosses back over the boundary when a PE input is
	// exhausted, but it must never pop BELOW this floor (which would drop into the
	// main document input and consume post-DOCTYPE content). 0 outside an external
	// subset, so skipBlanksPE performs no PE expansion there.
	dtdInputFloor int
	extSubSystem  string
	extSubURI     string
	version       string
	attsSpecial   map[specialAttrKey]enum.AttributeType
	// attsSpecialExternal records which entries of attsSpecial were declared in the
	// external subset (mirrors libxml2's XML_SPECIAL_EXTERNAL flag). Used for the
	// VC: Standalone Document Declaration attribute-normalization check.
	attsSpecialExternal map[specialAttrKey]struct{}
	// attrNormChanged is a transient flag set while parsing one attribute value: it
	// reports whether tokenized-type normalization (leading/trailing trim or
	// internal-space collapse) altered the value. Read by parseAttribute for the
	// VC: Standalone Document Declaration normalization check.
	attrNormChanged bool
	attsDefault     map[string][]*Attribute
	valid           bool
	hasPERefs       bool
	// hasExternalPERef records that at least one EXTERNAL parameter entity was
	// referenced (its content loaded from an external resource). Unlike a purely
	// internal DTD, an external PE may fail to load or resolve incompletely, so
	// helium cannot be certain an undeclared general entity is truly undeclared
	// rather than declared in unread external markup — the undeclared-entity
	// validity error (VC: Entity Declared) is therefore suppressed in that case.
	hasExternalPERef bool
	pedantic         bool
	wellFormed       bool
	depth            int
	loadsubset       LoadSubsetOption
	charBufferSize   int
	baseURI          string          // document base URI for resolving external references
	catalog          CatalogResolver // XML catalog for entity resolution
	fsys             fs.FS           // filesystem for loading external DTDs and entities
	elem             *Element        // current context element

	nsTab       nsStack
	nsNrTab     []int // number of ns bindings pushed per element (parallel to nodeTab)
	doc         *Document
	nodeTab     nodeStack
	sizeentcopy int64 // cumulative entity expansion bytes (non-entity-specific)
	inputSize   int64 // total input document size
	maxAmpl     int   // max entity amplification factor (default 5; 0 = ratio check disabled)
	// nbentities int
	inputTab         inputStack
	cachedCursor     strcursor.Cursor // cached result of getCursor(); invalidated on push/pop
	stopped          bool
	disableSAX       bool              // suppress SAX callbacks after fatal error in recover mode
	recoverErr       error             // first fatal error saved during recovery
	blankRunErr      error             // sticky error from an over-cap whitespace run (prolog/inter-root DoS guard)
	elemDepth        int               // current element nesting depth
	maxElemDepth     int               // max allowed element nesting depth (0 = unlimited)
	maxNameLength    int               // max element/attribute/NCName length (0 = unlimited)
	maxCMDepth       int               // max DTD content-model declaration depth (0 = unlimited)
	maxExtDTDSize    int               // max bytes read from an external DTD subset (<= 0 = MaxExternalDTDSize)
	maxNodeContent   int               // max bytes of a single CDATA/comment/PI/char-data run or attribute value, AND of a contiguous XML-whitespace blank-skip run (0 = unlimited)
	currentEntityURI string            // URI of the external entity currently being replayed (for base-uri tracking)
	nameCache        map[string]string // per-parse string interning for element/attribute names
	charBuf          []byte            // reusable buffer for parseCharDataContent
	attrBuf          []attrData        // reusable attribute scratch buffer for start-tag parsing
	nsDeclaredBuf    []string          // reusable scratch buffer of ns prefixes declared on the current start tag
	baseURIScopes    []baseURIScope    // per-input baseURI overrides (restored when the input is popped)
	// externalPEScopes records, per pushed external parameter-entity input, the
	// entity whose replacement text the input holds. activeExternalPECount is the
	// set of external PEs currently on the input stack (count per entity, to
	// tolerate the same PE legitimately appearing at sibling positions). Together
	// they reject a self/mutually recursive external PE before it can drive
	// unbounded cursor pushes into the amplification ceiling. The active mark is
	// cleared when the pushed input is popped (popInput), mirroring baseURIScopes.
	externalPEScopes      []externalPEScope
	activeExternalPECount map[*Entity]int
}

// baseURIScope records the baseURI to restore when a particular pushed input
// (its cursor) is popped from the input stack. It is used to scope pctx.baseURI
// to an external parameter entity's own resolved URI while that entity's
// replacement text is being parsed, so relative system IDs in declarations
// INSIDE the entity resolve against the entity's location rather than the
// containing DTD.
type baseURIScope struct {
	input   any
	baseURI string
}

// externalPEScope records which external parameter entity a pushed input belongs
// to, so its active mark can be cleared when that exact input is popped (strictly
// LIFO, like baseURIScope).
type externalPEScope struct {
	input  any
	entity *Entity
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

// isXML11 reports whether the document being parsed declared XML version 1.1
// in its XML declaration. Used to gate XML 1.1-only well-formedness relaxations
// such as namespace prefix undeclaration (xmlns:pfx="").
func (ctx *parserCtx) isXML11() bool {
	return ctx.version == "1.1"
}

const (
	cbEntityDecl = iota
	cbGetParameterEntity
)

func (pctx *parserCtx) fireSAXCallback(ctx context.Context, typ int, args ...any) error {
	// This is ugly, but I *REALLY* wanted to catch all occurrences of
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
		return s.EntityDecl(ctx, args[0].(string), enum.InternalParameterEntity, "", "", args[1].(string)) //nolint:forcetypeassert
	case cbGetParameterEntity:
		entity, err := s.GetParameterEntity(ctx, args[1].(string)) //nolint:forcetypeassert
		if err == nil {
			ret := args[0].(*sax.Entity) //nolint:forcetypeassert
			*ret = entity
		}
		return err
	}
	return nil
}

func (ctx *parserCtx) pushNodeEntry(e nodeEntry) {
	ctx.nodeTab.Push(e)
}

func (ctx *parserCtx) peekNode() *nodeEntry {
	return ctx.nodeTab.PeekOne()
}

func (ctx *parserCtx) popNode() (elem *nodeEntry) {
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
	ctx.inputTab.Push(in)
	ctx.cachedCursor = nil // invalidate cache
}

// pushInputWithBaseURI pushes a new input AND scopes pctx.baseURI to baseURI
// while that input is on the stack. The previous baseURI is restored when this
// exact input is popped (popInput), so the override lasts precisely as long as
// the pushed cursor is being consumed — even though the cursor is drained later
// by the surrounding declaration loop rather than within the call that pushed it.
// An empty baseURI is treated as "no override" (the input is pushed normally),
// because clobbering baseURI to "" would break relative resolution rather than
// help it.
func (ctx *parserCtx) pushInputWithBaseURI(in any, baseURI string) {
	if baseURI == "" {
		ctx.pushInput(in)
		return
	}
	ctx.baseURIScopes = append(ctx.baseURIScopes, baseURIScope{input: in, baseURI: ctx.baseURI})
	ctx.baseURI = baseURI
	ctx.pushInput(in)
}

// pushExternalPEInput pushes an external parameter entity's replacement text
// (scoping baseURI to its resolved URI) AND records the entity as active so a
// nested reference to the same PE — while its earlier input is still being
// drained — is detected as recursion. The active mark is cleared in popInput.
func (ctx *parserCtx) pushExternalPEInput(in any, baseURI string, ent *Entity) {
	ctx.pushInputWithBaseURI(in, baseURI)
	if ctx.activeExternalPECount == nil {
		ctx.activeExternalPECount = make(map[*Entity]int)
	}
	ctx.activeExternalPECount[ent]++
	ctx.externalPEScopes = append(ctx.externalPEScopes, externalPEScope{input: in, entity: ent})
}

// externalPEActive reports whether the given external parameter entity is
// currently on the input stack (its replacement text is being parsed).
func (ctx *parserCtx) externalPEActive(ent *Entity) bool {
	return ctx.activeExternalPECount[ent] > 0
}

// effectivelyExternal reports whether a markup declaration parsed at this point
// counts as EXTERNAL for the VC: Standalone Document Declaration (XML §2.9). A
// declaration is external when it comes from the external subset OR from an
// external parameter entity referenced anywhere (mirrors libxml2's PARSER_EXTERNAL
// = inSubset==2 OR the current input is an XML_EXTERNAL_PARAMETER_ENTITY). An
// external-PE-supplied declaration referenced from the internal subset is external
// markup even though it is registered in the internal subset's declaration table.
func (ctx *parserCtx) effectivelyExternal() bool {
	return ctx.inSubset == inExternalSubset || len(ctx.externalPEScopes) > 0
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

// dtdRefetch returns the top cursor for continued DTD-declaration parsing after
// a skipBlanksPE that may have crossed a parameter-entity boundary. In the
// EXTERNAL subset it re-fetches via getCursor so a boundary crossed by
// skipBlanksPE (a spent PE input popped, or a fresh padded PE input pushed) is
// reflected — a markup declaration may legitimately be assembled across PE
// boundaries there (§4.4.8). In the INTERNAL subset — where a parameter entity
// must NOT supply part of a markup declaration (WFC: PEs in Internal Subset) — it
// returns the caller's existing cursor unchanged, so an exhausted PE input is not
// silently auto-popped across the declaration boundary; the stalled parse then
// surfaces the boundary violation as an error, exactly as before.
func (ctx *parserCtx) dtdRefetch(cur strcursor.Cursor) strcursor.Cursor {
	if ctx.external {
		return ctx.getCursor()
	}
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
			ctx.popInput()
			continue
		}
		ctx.cachedCursor = cur
		return cur
	}
	return nil
}

// cursorDecodeErr returns a sticky transcoding/decode error from the active
// cursor, if any. Such an error (e.g. an unpaired UTF-16 surrogate that the
// decoder replaced with U+FFFD) is otherwise indistinguishable from a clean
// EOF via Done(), so callers must consult this explicitly to reject malformed
// encoded input.
func (ctx *parserCtx) cursorDecodeErr() error {
	switch cur := ctx.getCursor().(type) {
	case *strcursor.UTF8Cursor:
		return cur.Err()
	case *strcursor.ByteCursor:
		return cur.Err()
	default:
		return nil
	}
}

func (ctx *parserCtx) popInput() any { //nolint:unparam // return value used for type generality
	ctx.cachedCursor = nil // invalidate cache
	popped := ctx.inputTab.Pop()
	// Restore any baseURI override scoped to this exact input (see
	// pushInputWithBaseURI). Inputs are popped strictly LIFO, so the override —
	// if any — for the popped input is the top of the scope stack.
	if n := len(ctx.baseURIScopes); n > 0 && ctx.baseURIScopes[n-1].input == popped {
		ctx.baseURI = ctx.baseURIScopes[n-1].baseURI
		ctx.baseURIScopes = ctx.baseURIScopes[:n-1]
	}
	// Clear the active mark for an external parameter entity whose pushed input is
	// being popped (strictly LIFO, like the baseURI scope above), so a later
	// sibling reference to the same PE is not mistaken for recursion.
	if n := len(ctx.externalPEScopes); n > 0 && ctx.externalPEScopes[n-1].input == popped {
		ent := ctx.externalPEScopes[n-1].entity
		ctx.externalPEScopes = ctx.externalPEScopes[:n-1]
		ctx.activeExternalPECount[ent]--
		if ctx.activeExternalPECount[ent] <= 0 {
			delete(ctx.activeExternalPECount, ent)
		}
	}
	return popped
}

func (ctx *parserCtx) currentInputID() any {
	return ctx.inputTab.PeekOne()
}

// resolveLimit maps a builder-supplied limit value to its effective
// parser-context value: zero selects def, a negative value means "no limit"
// (returns 0, the context sentinel for unlimited / disabled), and a positive
// value is used verbatim.
func resolveLimit(v, def int) int {
	if v == 0 {
		return def
	}
	if v < 0 {
		return 0
	}
	return v
}

// nameTooLong reports whether a name of n bytes exceeds the configured maximum
// name length. A maxNameLength of zero means no limit is enforced.
func (ctx *parserCtx) nameTooLong(n int) bool {
	return ctx.maxNameLength > 0 && n > ctx.maxNameLength
}

// nodeContentTooLong reports whether an indivisible content run of n bytes
// exceeds the configured maximum node-content size. A maxNodeContent of zero
// means no limit is enforced. Exactly maxNodeContent bytes is allowed; one more
// fails (strict-greater).
func (ctx *parserCtx) nodeContentTooLong(n int) bool {
	return ctx.maxNodeContent > 0 && n > ctx.maxNodeContent
}

// writeAttrString appends s to the attribute-value buffer, enforcing the
// node-content cap BEFORE the copy so no attribute write path (entity-reference
// name, predefined-entity replacement, char-ref output, or literal text) can
// grow the buffer past the cap. Returns ErrNodeContentTooLarge if the append
// would exceed the cap. Exactly the cap is accepted (strict-greater).
func (ctx *parserCtx) writeAttrString(c context.Context, b *bytes.Buffer, s string) error {
	if ctx.nodeContentTooLong(b.Len() + len(s)) {
		return ctx.error(c, ErrNodeContentTooLarge)
	}
	_, _ = b.WriteString(s)
	return nil
}

// writeAttrByte appends a single byte to the attribute-value buffer with the
// same pre-write cap enforcement as writeAttrString.
func (ctx *parserCtx) writeAttrByte(c context.Context, b *bytes.Buffer, by byte) error {
	if ctx.nodeContentTooLong(b.Len() + 1) {
		return ctx.error(c, ErrNodeContentTooLarge)
	}
	_ = b.WriteByte(by)
	return nil
}

// writeAttrRune appends a rune to the attribute-value buffer with the same
// pre-write cap enforcement as writeAttrString.
func (ctx *parserCtx) writeAttrRune(c context.Context, b *bytes.Buffer, r rune) error {
	if ctx.nodeContentTooLong(b.Len() + utf8.RuneLen(r)) {
		return ctx.error(c, ErrNodeContentTooLarge)
	}
	_, _ = b.WriteRune(r)
	return nil
}

// nodeContentScanBudget returns the byte budget to hand a bounded char-data
// scan so it stops just past the cap (cap + utf8.UTFMax guarantees a run longer
// than the cap is detected even when a multi-byte rune straddles the boundary).
// Zero means unbounded.
func (ctx *parserCtx) nodeContentScanBudget() int {
	if ctx.maxNodeContent <= 0 {
		return 0
	}
	return ctx.maxNodeContent + utf8.UTFMax
}

// countingReader wraps an io.Reader and tracks the total number of bytes read
// through it. The EBCDIC streaming path wraps its reconstructed stream
// (sniff prefix + remainder) in one so the entity-amplification guard can use
// the real number of document bytes consumed so far instead of the bounded
// sniff-prefix length that rawInput holds in that mode. The count can never
// exceed the source's true byte length, so it is a safe lower bound on the
// document size.
type countingReader struct {
	r io.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
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
	ctx.attsSpecial = map[specialAttrKey]enum.AttributeType{}
	ctx.attsSpecialExternal = map[specialAttrKey]struct{}{}
	ctx.attsDefault = map[string][]*Attribute{}
	ctx.wellFormed = true
	ctx.spaceTab = ctx.spaceTab[:0]
	ctx.spaceTab = append(ctx.spaceTab, -1) // initial value before any element
	ctx.inputSize = int64(len(ctx.rawInput))
	ctx.maxAmpl = DefaultMaxEntityAmplification
	ctx.maxNameLength = DefaultMaxNameLength
	ctx.maxCMDepth = DefaultMaxContentModelDepth
	ctx.maxNodeContent = DefaultMaxNodeContentSize
	if p != nil {
		ctx.sax = p.sax
		if tb, ok := p.sax.(*TreeBuilder); ok {
			ctx.treeBuilder = tb
		}
		ctx.charBufferSize = p.charBufferSize
		ctx.options = p.options
		ctx.catalog = p.catalog
		ctx.fsys = p.fsys
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
		if ctx.options.IsSet(parseSkipIDs) {
			ctx.loadsubset.Set(SkipIDs)
		}
		ctx.maxElemDepth = p.maxDepth
		ctx.maxExtDTDSize = p.maxExtDTDSize
		ctx.maxAmpl = resolveLimit(p.maxEntityAmpl, DefaultMaxEntityAmplification)
		ctx.maxNameLength = resolveLimit(p.maxNameLength, DefaultMaxNameLength)
		ctx.maxCMDepth = resolveLimit(p.maxCMDepth, DefaultMaxContentModelDepth)
		ctx.maxNodeContent = resolveLimit(p.maxNodeContent, DefaultMaxNodeContentSize)
	}
	if ctx.fsys == nil {
		ctx.fsys = iofs.DenyAll{}
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
				// A single rune is wider than bufSize (e.g. CharBufferSize(1)
				// with multi-byte text). Splitting it would emit invalid UTF-8
				// fragments, so deliver the whole rune even though it exceeds
				// bufSize: walk forward to the next character boundary.
				end = bufSize
				for end < len(data) && !utf8.RuneStart(data[end]) {
					end++
				}
			}
		}

		switch err := handler(ctx, data[:end]); err {
		case nil, sax.ErrHandlerUnspecified:
			// The handler may have requested a stop on this chunk's callback
			// (including the final chunk, where the loop would otherwise exit
			// with nil). Report the stop so the caller terminates promptly and
			// emits no further chunks.
			if pctx.stopped {
				return errParserStopped
			}
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
	// A blank-run scan that tripped the cancellation/over-cap guard records a
	// sticky pctx.blankRunErr. Because skipBlanks/skipBlankBytes only return a
	// bool, callers that ignore the sticky error (XML declaration, DTD subset,
	// ...) keep going and report a generic follow-on parse error. Prefer the
	// blank-run error verbatim here so it surfaces regardless of which caller hit
	// it: this keeps context.Canceled propagating as cancellation (not a syntax
	// error, preserving the no-partial-document contract) and keeps
	// ErrNodeContentTooLarge from being masked behind a generic XML-decl/DTD error.
	if pctx.blankRunErr != nil {
		err = pctx.blankRunErr
	} else if !isParseAbort(err) {
		// A read failure or cancellation recorded as the active cursor's sticky
		// read error (most importantly a push/streaming-stream Read returning
		// context.Canceled when cancellation unblocks a pending wait) — or a
		// pending ctx cancellation — must never be reported as a synthesized
		// syntax error. Per the cancellation contract a cancelled/failed parse
		// surfaces the underlying cause, not a malformed-document diagnostic.
		// This generalizes the blankRunErr preference above to callers that turn
		// a short read into a follow-on syntax error WITHOUT going through the
		// blank scanner — e.g. a "<?xml" whose required trailing blank was never
		// read, so looksLikeXMLDecl cannot confirm the declaration and it is
		// reparsed as a reserved-target PI ("XML declaration allowed only at the
		// start of the document") instead of propagating the real read error.
		// It mirrors the ctx.Err()/cursorDecodeErr() gate parseDocument already
		// applies at the document-end boundary.
		if rerr := ctx.Err(); rerr != nil {
			err = rerr
		} else if rerr := pctx.cursorDecodeErr(); rerr != nil {
			err = rerr
		}
	}
	// Parse-abort errors (the stop sentinel, context cancellation, deadline
	// expiry) are not genuine parse failures: pass them through unchanged so
	// callers see them directly and SAX error handlers are not fired as if the
	// document were malformed.
	if isParseAbort(err) {
		if errors.Is(err, errParserStopped) {
			return errParserStopped
		}
		return err
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
