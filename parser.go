package helium

import (
	"bytes"

	"github.com/lestrrat-go/helium/sax"
	"github.com/lestrrat-go/pdebug"
)

type Parser struct {
	sax            sax.SAX2Handler
	charBufferSize int
	options        ParseOption
	baseURI        string
}

func Parse(b []byte) (*Document, error) {
	p := NewParser()
	return p.Parse(b)
}

func NewParser() *Parser {
	return &Parser{
		sax: NewTreeBuilder(),
	}
}

func (p *Parser) Parse(b []byte) (*Document, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("=== START Parser.Parse ===")
		defer g.IRelease("=== END Parser.Parse ===")
	}

	ctx := &parserCtx{rawInput: b, baseURI: p.baseURI}
	if err := ctx.init(p, bytes.NewReader(b)); err != nil {
		return nil, err
	}
	defer func() {
		if err := ctx.release(); err != nil {
			// Log error but don't override the main return error
			if pdebug.Enabled {
				pdebug.Printf("ctx.release() failed: %s", err)
			}
		}
	}()

	if err := ctx.parseDocument(); err != nil {
		if p != nil && p.options.IsSet(ParseRecover) {
			// ParseRecover: return the partial document along with the error
			return ctx.doc, err
		}
		return nil, err
	}

	// DTD validation: run post-parse document validation when requested.
	if p != nil && p.options.IsSet(ParseDTDValid) && ctx.doc != nil {
		if ve := validateDocument(ctx.doc); ve != nil {
			return ctx.doc, ve
		}
	}

	return ctx.doc, nil
}

func (p *Parser) SetSAXHandler(s sax.SAX2Handler) {
	p.sax = s
}

func (p *Parser) SetOption(opt ParseOption) {
	p.options.Set(opt)
}

// SetCharBufferSize sets the maximum number of bytes delivered in a single
// Characters or IgnorableWhitespace SAX callback. When size <= 0 (the
// default), all character data is delivered in one call. When size > 0,
// data longer than size bytes is split into chunks of at most size bytes,
// always respecting UTF-8 character boundaries.
func (p *Parser) SetCharBufferSize(size int) {
	p.charBufferSize = size
}

// SetBaseURI sets the document's base URI, used for resolving relative
// references such as external DTD system identifiers.
func (p *Parser) SetBaseURI(uri string) {
	p.baseURI = uri
}
