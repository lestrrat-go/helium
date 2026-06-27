package html

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/push"
)

// Parser configures HTML parsing. It is a value-style wrapper: fluent
// methods return updated copies and the original is never mutated.
type Parser struct {
	cfg *parserCfg
}

type parserCfg struct {
	parseConfig
}

// NewParser creates a new HTML Parser with default settings.
func NewParser() Parser {
	return Parser{cfg: &parserCfg{}}
}

func (p Parser) clone() Parser {
	if p.cfg == nil {
		return Parser{cfg: &parserCfg{}}
	}
	cp := *p.cfg
	return Parser{cfg: &cp}
}

// SuppressImplied controls whether automatic insertion of implied
// html/head/body elements is suppressed.
// libxml2: HTML_PARSE_NOIMPLIED
// Default: false (implied elements are inserted)
func (p Parser) SuppressImplied(v bool) Parser {
	p = p.clone()
	p.cfg.noImplied = v
	return p
}

// StripBlanks controls whether whitespace-only text nodes are removed
// from the DOM.
// libxml2: HTML_PARSE_NOBLANKS
// Default: false (whitespace nodes are preserved)
func (p Parser) StripBlanks(v bool) Parser {
	p = p.clone()
	p.cfg.noBlanks = v
	return p
}

// SuppressErrors controls whether error messages from the SAX error
// handler are suppressed.
// libxml2: HTML_PARSE_NOERROR
// Default: false (errors are reported)
func (p Parser) SuppressErrors(v bool) Parser {
	p = p.clone()
	p.cfg.noError = v
	return p
}

// SuppressWarnings controls whether warning messages from the SAX
// warning handler are suppressed.
// libxml2: HTML_PARSE_NOWARNING
// Default: false (warnings are reported)
func (p Parser) SuppressWarnings(v bool) Parser {
	p = p.clone()
	p.cfg.noWarning = v
	return p
}

// Strict controls whether non-[ErrHandlerUnspecified] return values from a
// SAX callback abort the parse. With Strict(false) (the default), such
// returns are forwarded to the parser's [WarningHandler] (see
// [SAXCallbacks.SetOnWarning]) and parsing continues — matching libxml2's
// "tolerate and produce a best-effort DOM" semantics. With Strict(true),
// the first such return is captured and surfaced as the [Parser.Parse]
// error after parsing reaches a stable state. [ErrHandlerUnspecified]
// is always filtered before either path.
//
// Default: false.
func (p Parser) Strict(v bool) Parser {
	p = p.clone()
	p.cfg.strict = v
	return p
}

// MaxContentSize bounds, in bytes, the size of a single content section.
//
// For normal data-state text (ordinary element content), raw-text
// (script/style), RCDATA (title/textarea), and plaintext content it bounds the
// streaming scanner's per-chunk working set: the parser flushes accumulated
// content to SAX in temporary chunks that target this size, so the scanner's
// own peak memory stays bounded even for a gigantic or unterminated section,
// which still parses successfully. A chunk may slightly exceed the cap because
// an indivisible token is never split: a whole multi-byte UTF-8 rune (or a
// resolved character reference) is always emitted intact, so a single rune
// larger than the cap is emitted whole. An unresolved named-reference literal —
// in normal data-state text as well as RCDATA — hard-fails with
// [ErrContentSizeExceeded] when the bytes it would emit
// ("&" + name + optional ";") exceed the cap — this applies to ANY unresolved
// literal, whether short, semicolon-terminated, or unbounded. A known-entity
// (';'-terminated) reference is exempt: it is resolved within a fixed lookahead
// window and never charged against the cap. A no-';' LEGACY resolution — a full
// legacy entity (e.g. "&amp") OR a legacy-PREFIX match (e.g. "&ampZ", where the
// "amp" prefix resolves and "Z" is echoed) — is exempt ONLY when its whole
// consumed run ("&" + name) fits within the cap; over the cap it hard-fails with
// [ErrContentSizeExceeded] and emits NOTHING. This is enforced uniformly: a SHORT
// within-lookahead run (e.g. "&ampZ" under a cap of 2) and a SATURATED ambiguous
// run (e.g. "&amp" followed by a long alphanumeric tail) both hard-fail rather
// than emit a partial resolution.
//
// MaxContentSize also bounds the undecided-encoding deferred prefix: an
// undeclared-charset [Parser.ParseReader] or push stream whose bytes keep
// proving valid UTF-8 buffers undecided only up to this cap, and an over-cap
// undecided-encoding stream is rejected with [ErrContentSizeExceeded] rather
// than committing to a single Latin-1/UTF-8 interpretation.
//
// Otherwise this bounds only the streaming scanner / SAX chunk size. DOM
// construction via
// [Parser.Parse] necessarily merges every chunk back into the document tree
// (treeBuilder.AppendText), so the resulting [helium.Document] still retains the
// full content; MaxContentSize does not make DOM parsing memory-bounded for
// large documents. Use a SAX-only consumer to benefit from the streaming bound.
//
// For comments, bogus comments, processing instructions, and attribute values
// it is a HARD cap: these constructs map to a single indivisible SAX event and
// DOM node and cannot be chunked without corrupting the document, so one
// exceeding this size before its terminator fails the parse with
// [ErrContentSizeExceeded] rather than emitting a truncated node. The attribute
// cap is enforced per byte and also covers '&'-led entity and '&#'-led numeric
// runs, so an unterminated value cannot buffer without limit.
//
// Whitespace-deferral exception: the soft cap on normal data-state text has a
// HARD-fail case whenever a text run's leading whitespace must be DEFERRED
// because its insertion target or significance is still undecided. This happens
// in two situations: under [Parser.StripBlanks](true) (a run is suppressed only
// when EVERY byte is whitespace, a decision that cannot be made until the run's
// first non-whitespace byte or its end is seen), and during implied-<body>
// deferral (the body subtree has not been entered, mode < insertInBody, AND
// implied insertion is enabled, so the next non-whitespace byte would open the
// implied <body> and the run's parent is undecided). In either case the scanner
// refuses to flush a run whose leading whitespace prefix alone reaches the cap
// with yet more whitespace beyond it, because doing so would require buffering
// the run unbounded to learn its significance or parent. Such a run fails the
// parse with [ErrContentSizeExceeded] rather than parsing successfully. Once a
// non-whitespace byte is seen the run is known significant and is chunked
// normally (its leading whitespace, including the whole first non-whitespace
// rune even when the cap splits it, rides along in the first chunk). Default-mode
// whitespace with a fixed insertion target and no StripBlanks — including under
// [Parser.SuppressImplied](true) once an element is open — stays a pure soft-cap
// stream with no hard-fail.
//
// Indivisible STRUCTURAL token scans — a tag name, end-tag name, attribute name,
// PUBLIC/SYSTEM DOCTYPE literal, or intra-tag whitespace run — are also HARD-
// capped, but against a separate structural scan cap rather than this value:
// because callers legitimately set MaxContentSize very small, the structural cap
// is FLOORED at the 16 MiB default (so ordinary names like "script" always parse)
// and only grows when MaxContentSize is raised above that floor. An over-cap
// structural run fails the parse with [ErrContentSizeExceeded] and emits no
// partial token.
//
// A value <= 0 selects the default (16 MiB).
//
// Default: 16 MiB.
func (p Parser) MaxContentSize(v int) Parser {
	p = p.clone()
	p.cfg.maxContentSize = v
	return p
}

func (p Parser) parseConfig() parseConfig {
	if p.cfg == nil {
		return parseConfig{}
	}
	return p.cfg.parseConfig
}

// ParseReader parses HTML from an io.Reader, feeding the input through
// encoding detection and normalization wrappers.
//
// Whether the input is processed incrementally depends on its encoding. Once a
// streamable encoding is determined - either declared (BOM, meta charset=utf-8,
// or meta charset=iso-8859-1) or detected as a genuine non-UTF-8 byte sequence -
// bytes are converted and consumed incrementally. An input with no declared
// encoding that keeps proving valid UTF-8 cannot yet be distinguished from a
// Latin-1/Windows-1252 stream, so it is buffered while undecided; this matches
// the materialization behavior of Parse with a []byte. To keep memory bounded,
// that buffering is capped at the configured [Parser.MaxContentSize] (16 MiB by
// default): an undeclared stream that ends with valid UTF-8 at or below the cap
// is accepted, but if the cap fills with no non-UTF-8 byte and more input still
// follows the encoding cannot be settled within the bound, so the reader fails
// closed with a bounded-input error ([ErrContentSizeExceeded]) rather than
// committing to UTF-8 and risking a silently mis-decoded later byte.
func (p Parser) ParseReader(ctx context.Context, r io.Reader) (*helium.Document, error) {
	tb := newTreeBuilder()
	hp := newParserFromReader(ctx, r, tb, p.parseConfig())
	if err := hp.parse(ctx); err != nil {
		return nil, err
	}
	if enc := hp.finalEncoding(); enc != "" {
		tb.doc.SetEncoding(enc)
	}
	return tb.doc, nil
}

// Parse parses HTML data and returns a helium Document.
// (libxml2: htmlParseDoc)
func (p Parser) Parse(ctx context.Context, data []byte) (*helium.Document, error) {
	tb := newTreeBuilder()
	hp := newParser(ctx, data, tb, p.parseConfig())
	if err := hp.parse(ctx); err != nil {
		return nil, err
	}
	if enc := hp.detectedEncoding; enc != "" {
		tb.doc.SetEncoding(enc)
	}
	return tb.doc, nil
}

// ParseFile reads and parses an HTML file.
// (libxml2: htmlParseFile)
func (p Parser) ParseFile(ctx context.Context, filename string) (*helium.Document, error) {
	abs, err := filepath.Abs(filename)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(filename) //nolint:gosec // filename is caller-supplied
	if err != nil {
		return nil, err
	}
	defer f.Close()

	doc, err := p.ParseReader(ctx, f)
	if err != nil {
		return nil, err
	}
	doc.SetURL(abs)
	return doc, nil
}

// ParseWithSAX parses HTML data, firing SAX events to the given handler
// without building a DOM tree.
// (libxml2: htmlSAXParseDoc)
func (p Parser) ParseWithSAX(ctx context.Context, data []byte, handler SAXHandler) error {
	hp := newParser(ctx, data, handler, p.parseConfig())
	return hp.parse(ctx)
}

// saxParser wraps a Parser with a SAX handler so that ParseReader fires
// SAX events instead of building a DOM tree.
type saxParser struct {
	parser  Parser
	handler SAXHandler
}

func (p Parser) withSAXHandler(h SAXHandler) saxParser {
	return saxParser{parser: p, handler: h}
}

func (sp saxParser) ParseReader(ctx context.Context, r io.Reader) (*helium.Document, error) {
	hp := newParserFromReader(ctx, r, sp.handler, sp.parser.parseConfig())
	return nil, hp.parse(ctx)
}

// PushParser provides an incremental HTML parsing interface
// (libxml2: htmlCreatePushParserCtxt).
// Data is pushed via Push or Write. A background goroutine consumes the
// pushed chunks. Parsing becomes progressive only AFTER the initial
// 1024-byte (or EOF) charset prescan AND only once a streamable encoding
// has been settled: the prescan uses a manual read loop reading up to 1024
// bytes until the buffer is full, EOF, or a read error, so an input smaller
// than 1024 bytes is buffered until [PushParser.Close], and larger inputs
// only parse progressively once those first 1024 bytes have arrived.
// Streaming after the prescan applies when an encoding is declared or
// detected (charset=utf-8, charset=iso-8859-1, or a non-UTF-8 head routed to
// Latin-1). An input with no charset declaration whose bytes keep proving valid
// UTF-8 stays undecided and continues to buffer (because a later non-UTF-8 byte
// would force the whole prefix to be reinterpreted as Windows-1252), but only up
// to a bounded prefix (the configured [Parser.MaxContentSize], 16 MiB by
// default): a stream ending with valid UTF-8 at or below the cap is accepted, but
// if the cap fills with no non-UTF-8 byte and more input still follows the
// encoding cannot be settled within the bound, so the reader fails closed with a
// bounded-input error ([ErrContentSizeExceeded]) instead of buffering to
// [PushParser.Close]/EOF or committing to a possibly-wrong UTF-8 interpretation.
// Call [PushParser.Close] to signal end-of-input and retrieve the
// parsed Document.
type PushParser = push.Parser[*helium.Document]

// NewPushParser creates an HTML PushParser that builds a DOM tree.
// A background goroutine is started immediately. It consumes data pushed
// via [PushParser.Push] or [PushParser.Write]; parsing becomes progressive
// only AFTER the initial 1024-byte (or EOF) charset prescan buffers its
// head AND a streamable encoding has been settled. An undeclared input that
// keeps proving valid UTF-8 stays undecided and buffers up to a bounded prefix
// (the configured [Parser.MaxContentSize], 16 MiB by default); if it ends with
// valid UTF-8 at or below the cap it is accepted, otherwise it fails closed with
// a bounded-input error ([ErrContentSizeExceeded]) rather than committing to a
// possibly-wrong UTF-8 interpretation. The completed Document is returned once
// [PushParser.Close] is called.
func (p Parser) NewPushParser(ctx context.Context) *PushParser {
	return push.New[*helium.Document](ctx, p)
}

// NewSAXPushParser creates an HTML PushParser that fires SAX events
// to the given handler instead of building a DOM tree.
// (libxml2: htmlCreatePushParserCtxt with SAX handler)
func (p Parser) NewSAXPushParser(ctx context.Context, h SAXHandler) *PushParser {
	return push.New[*helium.Document](ctx, p.withSAXHandler(h))
}
