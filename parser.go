package helium

import (
	"github.com/lestrrat/helium/sax"
)
func NewParser() *Parser {
	return &Parser{
		sax: &TreeBuilder{},
	}
}

func (p *Parser) Parse(b []byte) (*Document, error) {
	ctx := &parserCtx{}
	ctx.init(p, b)
	defer ctx.release()
	if err := ctx.parse(); err != nil {
		return nil, err
	}

	return ctx.doc, nil
}

func (p *Parser) SetSAXHandler(s sax.Handler) {
	p.sax = s
}


