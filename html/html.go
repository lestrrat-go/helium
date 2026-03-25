// Package html implements an HTML parser compatible with libxml2's HTMLparser.
//
// It parses HTML 4.01 documents, producing a helium DOM tree or firing SAX1
// events. Unlike the XML parser in the parent package, the HTML parser is
// case-insensitive, handles void elements, auto-closes elements, and inserts
// implied html/head/body elements.
package html

import (
	"context"
	"os"

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
	cp := *p.cfg
	return Parser{cfg: &cp}
}

// NoImplied suppresses automatic insertion of implied html/head/body elements.
// (libxml2: HTML_PARSE_NOIMPLIED)
func (p Parser) NoImplied() Parser {
	p = p.clone()
	p.cfg.noImplied = true
	return p
}

// NoBlanks removes whitespace-only text nodes from the DOM.
// (libxml2: HTML_PARSE_NOBLANKS)
func (p Parser) NoBlanks() Parser {
	p = p.clone()
	p.cfg.noBlanks = true
	return p
}

// NoError suppresses error messages from the SAX error handler.
// (libxml2: HTML_PARSE_NOERROR)
func (p Parser) NoError() Parser {
	p = p.clone()
	p.cfg.noError = true
	return p
}

// NoWarning suppresses warning messages from the SAX warning handler.
// (libxml2: HTML_PARSE_NOWARNING)
func (p Parser) NoWarning() Parser {
	p = p.clone()
	p.cfg.noWarning = true
	return p
}

// Parse parses HTML data and returns a helium Document.
// (libxml2: htmlParseDoc)
func (p Parser) Parse(ctx context.Context, data []byte) (*helium.Document, error) {
	tb := newTreeBuilder()
	hp := newParser(data, tb, p.cfg.parseConfig)
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
	return p.Parse(ctx, data)
}

// ParseWithSAX parses HTML data, firing SAX events to the given handler
// without building a DOM tree.
// (libxml2: htmlSAXParseDoc)
func (p Parser) ParseWithSAX(ctx context.Context, data []byte, handler SAXHandler) error {
	hp := newParser(data, handler, p.cfg.parseConfig)
	return hp.parse()
}

// NewPushParser creates an HTML PushParser that builds a DOM tree.
func (p Parser) NewPushParser(ctx context.Context) *PushParser {
	return &PushParser{ctx: ctx, cfg: p.cfg.parseConfig}
}

// NewSAXPushParser creates an HTML PushParser that fires SAX events
// to the given handler instead of building a DOM tree.
// (libxml2: htmlCreatePushParserCtxt with SAX handler)
func (p Parser) NewSAXPushParser(ctx context.Context, h SAXHandler) *PushParser {
	return &PushParser{ctx: ctx, sax: h, cfg: p.cfg.parseConfig}
}
