package helium

import (
	"bytes"
	"context"

	"github.com/lestrrat-go/helium/sax"
)

func Parse(ctx context.Context, b []byte) (*Document, error) {
	p := NewParser()
	return p.Parse(ctx, b)
}

func NewParser() *Parser {
	return &Parser{
		sax: NewTreeBuilder(),
	}
}

func (p *Parser) Parse(ctx context.Context, b []byte) (*Document, error) {
	pctx := &parserCtx{}
	if err := pctx.init(p, bytes.NewReader(b)); err != nil {
		return nil, err
	}
	defer func() {
		if err := pctx.release(); err != nil {
			// Log error but don't override the main return error
		}
	}()

	if err := pctx.parseDocument(ctx); err != nil {
		return nil, err
	}

	return pctx.doc, nil
}

func (p *Parser) SetSAXHandler(s sax.SAX2Handler) {
	p.sax = s
}
