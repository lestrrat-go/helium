package helium

import (
	"context"

	"github.com/lestrrat-go/helium/internal/strcursor"
)

func (pctx *parserCtx) skipBlanks(ctx context.Context) bool {
	i := 0
	cur := pctx.getCursor()
	if cur == nil {
		return false
	}
	for c := cur.PeekAt(i); isBlankByte(c) && !cur.Done(); c = cur.PeekAt(i) {
		i++
	}
	if i > 0 {
		if err := cur.Advance(i); err != nil {
			return false
		}

		if cur.Peek() == '%' {
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
	for c := cur.PeekAt(i); c != 0 && isBlankByte(c); c = cur.PeekAt(i) {
		i++
	}
	if i > 0 {
		if err := cur.Advance(i); err != nil {
			return false
		}

		if cur.Peek() == '%' {
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
		return false
	}
	if c := cur.Peek(); c != '<' && c != 0xD {
		return false
	}

	return true
}

// whitespaceContextIgnorable reports whether, given only the current parse
// context (xml:space, the node stack, and the mixed-content model), an
// all-whitespace character-data run at this position would be reported as
// ignorable whitespace rather than character data.
//
// Unlike areBlanksBytes it omits the byte-level blankness test and the
// cursor lookahead at the end of the run, so it can be evaluated once for a
// run that is delivered in streaming chunks (where the end-of-run delimiter is
// not yet in view). For the cursorless (pure-SAX, doc == nil) case it returns
// true rather than peeking for the trailing delimiter; the chunked caller
// compensates by tracking blankness incrementally AND by re-applying the
// trailing-delimiter check once the run ends (a blank run ending at '&' rather
// than '<'/CR is character data, matching the single-shot path).
func (ctx *parserCtx) whitespaceContextIgnorable() bool {
	if ctx.spaceTab[len(ctx.spaceTab)-1] == 1 {
		return false
	}
	if ctx.peekNode() == nil {
		return false
	}
	if ctx.doc != nil {
		ok, _ := ctx.doc.IsMixedElement(ctx.peekNode().Name())
		return !ok
	}
	return true
}

// allBlankBytes reports whether every byte of s is XML whitespace.
func allBlankBytes(s []byte) bool {
	for _, b := range s {
		if !isBlankCh(rune(b)) {
			return false
		}
	}
	return true
}
