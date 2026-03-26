package html

import (
	"context"
	"os"
	"path/filepath"

	"github.com/lestrrat-go/helium"
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

// Parse parses HTML data and returns a helium Document.
// (libxml2: htmlParseDoc)
func (p Parser) Parse(ctx context.Context, data []byte) (*helium.Document, error) {
	tb := newTreeBuilder()
	hp := newParser(data, tb, p.parseConfig())
	if err := hp.parse(); err != nil {
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
	hp := newParser(data, handler, p.parseConfig())
	return hp.parse()
}

// NewPushParser creates an HTML PushParser that builds a DOM tree.
func (p Parser) NewPushParser(ctx context.Context) *PushParser {
	if ctx == nil {
		ctx = context.Background()
	}
	return &PushParser{ctx: ctx, cfg: p.parseConfig()}
}

// NewSAXPushParser creates an HTML PushParser that fires SAX events
// to the given handler instead of building a DOM tree.
// (libxml2: htmlCreatePushParserCtxt with SAX handler)
func (p Parser) NewSAXPushParser(ctx context.Context, h SAXHandler) *PushParser {
	if ctx == nil {
		ctx = context.Background()
	}
	return &PushParser{ctx: ctx, sax: h, cfg: p.parseConfig()}
}
