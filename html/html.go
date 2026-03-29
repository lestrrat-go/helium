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

func (p Parser) parseConfig() parseConfig {
	if p.cfg == nil {
		return parseConfig{}
	}
	return p.cfg.parseConfig
}

// ParseReader parses HTML from an io.Reader. The input is streamed through
// encoding detection and normalization wrappers without reading it all into
// memory first.
func (p Parser) ParseReader(ctx context.Context, r io.Reader) (*helium.Document, error) {
	tb := newTreeBuilder()
	hp := newParserFromReader(ctx, r, tb, p.parseConfig())
	if err := hp.parse(ctx); err != nil {
		return nil, err
	}
	if enc := hp.detectedEncoding; enc != "" {
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
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	doc, err := p.Parse(ctx, data)
	if err != nil {
		return nil, err
	}
	abs, err := filepath.Abs(filename)
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
// Data is pushed via Push or Write. A background goroutine waits for all
// data to arrive, then parses it in one shot. Call [PushParser.Close] to
// signal end-of-input and retrieve the parsed Document.
type PushParser = push.Parser[*helium.Document]

// NewPushParser creates an HTML PushParser that builds a DOM tree.
// A background goroutine is started immediately; it waits for data
// pushed via [PushParser.Push] or [PushParser.Write], then parses
// everything in one shot once [PushParser.Close] is called.
func (p Parser) NewPushParser(ctx context.Context) *PushParser {
	return push.New[*helium.Document](ctx, p)
}

// NewSAXPushParser creates an HTML PushParser that fires SAX events
// to the given handler instead of building a DOM tree.
// (libxml2: htmlCreatePushParserCtxt with SAX handler)
func (p Parser) NewSAXPushParser(ctx context.Context, h SAXHandler) *PushParser {
	return push.New[*helium.Document](ctx, p.withSAXHandler(h))
}
