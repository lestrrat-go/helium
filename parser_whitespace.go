package helium

import (
	"context"

	"github.com/lestrrat-go/helium/internal/strcursor"
	"github.com/lestrrat-go/pdebug"
)

func (pctx *parserCtx) skipBlanks(ctx context.Context) bool {
	i := 0
	if pdebug.Enabled {
		g := pdebug.IPrintf("START skipBlanks")
		defer func() {
			g.IRelease("END skipBlanks (skipped %d)", i)
		}()
	}
	cur := pctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	for c := cur.PeekAt(i); isBlankByte(c) && !cur.Done(); c = cur.PeekAt(i) {
		i++
	}
	if i > 0 {
		if err := cur.Advance(i); err != nil {
			return false
		}

		if cur.Peek() == '%' {
			pdebug.Printf("Found possible parameter entity reference")
			if err := pctx.handlePEReference(ctx); err != nil {
				return false
			}
		}
		return true
	}
	return false
}

func (pctx *parserCtx) skipBlankBytes(ctx context.Context, cur *strcursor.ByteCursor) bool {
	i := 0
	if pdebug.Enabled {
		g := pdebug.IPrintf("START skipBlankBytes")
		defer func() {
			g.IRelease("END skipBlankBytes (skipped %d)", i)
		}()
	}
	for c := cur.PeekAt(i); c != 0 && isBlankByte(c); c = cur.PeekAt(i) {
		i++
	}
	if i > 0 {
		if err := cur.Advance(i); err != nil {
			return false
		}

		if cur.Peek() == '%' {
			pdebug.Printf("Found possible parameter entity reference")
			if err := pctx.handlePEReference(ctx); err != nil {
				return false
			}
		}
		return true
	}
	return false
}

// note: unlike libxml2, we can't differentiate between SAX handlers
// that uses the same IgnorableWhitespace and Character handlers
// areBlanksBytes is like areBlanks but operates on []byte to avoid string
// allocation on the hot path.
func (ctx *parserCtx) areBlanksBytes(s []byte, blankChars bool) bool {
	if ctx.spaceTab[len(ctx.spaceTab)-1] == 1 {
		return false
	}

	if !blankChars {
		for _, b := range s {
			if !isBlankCh(rune(b)) {
				return false
			}
		}
	}

	if ctx.peekNode() == nil {
		return false
	}
	if ctx.doc != nil {
		ok, _ := ctx.doc.IsMixedElement(ctx.peekNode().Name())
		return !ok
	}

	cur := ctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if c := cur.Peek(); c != '<' && c != 0xD {
		return false
	}

	return true
}
