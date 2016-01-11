package helium

import (
	"bytes"

	"github.com/lestrrat/helium/internal/debug"
	"github.com/lestrrat/helium/sax"
)

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
	if debug.Enabled {
		g := debug.IPrintf("=== START Parser.Parse ===")
		defer g.IRelease("=== END Parser.Parse ===")
	}

	ctx := &parserCtx{}
	ctx.init(p, bytes.NewReader(b))
	defer ctx.release()

	if err := ctx.parseDocument(); err != nil {
		return nil, err
	}

	return ctx.doc, nil
}

func (p *Parser) SetSAXHandler(s sax.SAX2Handler) {
	p.sax = s
}
