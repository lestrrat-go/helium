package helium

import (
	"context"
	"sync"

	"github.com/lestrrat-go/pdebug/v3"
	"github.com/pkg/errors"
)

var xmlDeclHint = []byte{'<', '?', 'x', 'm', 'l'}

type parseCtx struct {
	context.Context
	doc *Document
}

var parseCtxPool = sync.Pool{
	New: allocParseCtx,
}

func allocParseCtx() interface{} {
	return &parseCtx{}
}

func getParseCtx() *parseCtx {
	return parseCtxPool.Get().(*parseCtx)
}

func releaseParseCtx(v *parseCtx) {
	v.Context = nil

	parseCtxPool.Put(v)
}

func parse(ctx context.Context, data []byte, options ...ParseOption) (*Document, error) {
	// Parser is just a thin wrapper around parseCtx.
	// we do this so we can safely do `go p.Parse()` and
	// not worry about synchronization

	pctx := getParseCtx()
	pctx.Context = ctx

	if err := parseDocument(pctx); err != nil {
		return nil, errors.Wrap(err, `failed to parse document`)
	}

	return pctx.doc, nil
}

func parseDocument(ctx *parseCtx) error {
	if pdebug.Enabled {
		g := pdebug.FuncMarker()
		defer g.End()
	}

	return nil
}
