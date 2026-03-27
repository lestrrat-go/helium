package helium

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/sax"
	"github.com/lestrrat-go/pdebug"
)

func (pctx *parserCtx) parseElementDecl(ctx context.Context) (enum.ElementType, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseElementDecl")
		defer g.IRelease("END parseElementDecl")
	}

	cur := pctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if !cur.ConsumeString("<!ELEMENT") {
		return enum.UndefinedElementType, pctx.error(ctx, ErrInvalidElementDecl)
	}
	startInput := pctx.currentInputID()

	if !isBlankByte(cur.Peek()) {
		return enum.UndefinedElementType, pctx.error(ctx, ErrSpaceRequired)
	}
	pctx.skipBlanks(ctx)

	name, err := pctx.parseName(ctx)
	if err != nil {
		return enum.UndefinedElementType, pctx.error(ctx, err)
	}

	if !isBlankByte(cur.Peek()) {
		return enum.UndefinedElementType, pctx.error(ctx, ErrSpaceRequired)
	}
	pctx.skipBlanks(ctx)

	var etype enum.ElementType
	var content *ElementContent
	if cur.ConsumeString("EMPTY") {
		etype = enum.EmptyElementType
	} else if cur.ConsumeString("ANY") {
		etype = enum.AnyElementType
	} else if cur.Peek() == '(' {
		content, etype, err = pctx.parseElementContentDecl(ctx)
		if err != nil {
			return enum.UndefinedElementType, pctx.error(ctx, err)
		}
	}

	pctx.skipBlanks(ctx)

	if cur.Peek() != '>' {
		return enum.UndefinedElementType, pctx.error(ctx, ErrGtRequired)
	}
	if err := cur.Advance(1); err != nil {
		return enum.UndefinedElementType, err
	}

	if pctx.currentInputID() != startInput {
		return enum.UndefinedElementType, pctx.error(ctx,
			fmt.Errorf("%w: element declaration doesn't start and stop in the same entity", ErrEntityBoundary))
	}

	if s := pctx.sax; s != nil {
		switch err := s.ElementDecl(ctx, name, etype, content); err {
		case nil, sax.ErrHandlerUnspecified:
		default:
			return enum.UndefinedElementType, pctx.error(ctx, err)
		}
	}

	return etype, nil
}

func (pctx *parserCtx) parseElementContentDecl(ctx context.Context) (*ElementContent, enum.ElementType, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseElementContentDecl")
		defer g.IRelease("END parseElementContentDecl")
	}

	cur := pctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if cur.Peek() != '(' {
		return nil, enum.UndefinedElementType, pctx.error(ctx, ErrOpenParenRequired)
	}
	if err := cur.Advance(1); err != nil {
		return nil, enum.UndefinedElementType, err
	}

	if pctx.instate == psEOF {
		return nil, enum.UndefinedElementType, pctx.error(ctx, ErrEOF)
	}

	pctx.skipBlanks(ctx)

	var ec *ElementContent
	var err error
	var etype enum.ElementType
	if cur.HasPrefixString("#PCDATA") {
		ec, err = pctx.parseElementMixedContentDecl(ctx)
		if err != nil {
			return nil, enum.UndefinedElementType, pctx.error(ctx, err)
		}
		etype = enum.MixedElementType
	} else {
		ec, err = pctx.parseElementChildrenContentDeclPriv(ctx, 0)
		if err != nil {
			return nil, enum.UndefinedElementType, pctx.error(ctx, err)
		}
		etype = enum.ElementElementType
	}

	pctx.skipBlanks(ctx)
	return ec, etype, nil
}

func (pctx *parserCtx) parseElementMixedContentDecl(ctx context.Context) (*ElementContent, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseElementMixedContentDecl")
		defer g.IRelease("END parseElementMixedContentDecl")
	}

	cur := pctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if !cur.ConsumeString("#PCDATA") {
		return nil, pctx.error(ctx, ErrPCDATARequired)
	}
	startInput := pctx.currentInputID()

	pctx.skipBlanks(ctx)

	if cur.Peek() == ')' {
		if pctx.valid && pctx.currentInputID() != startInput {
			_ = pctx.warning(ctx, "element content declaration doesn't start and stop in the same entity\n")
			pctx.valid = false
		}
		if err := cur.Advance(1); err != nil {
			return nil, pctx.error(ctx, err)
		}
		ret, err := pctx.doc.CreateElementContent("", ElementContentPCDATA)
		if err != nil {
			return nil, pctx.error(ctx, err)
		}

		if cur.Peek() == '*' {
			ret.coccur = ElementContentMult
			if err := cur.Advance(1); err != nil {
				return nil, err
			}
		}

		return ret, nil
	}

	var err error
	var retelem *ElementContent
	var curelem *ElementContent
	if c := cur.Peek(); c == '(' || c == '|' {
		retelem, err = pctx.doc.CreateElementContent("", ElementContentPCDATA)
		if err != nil {
			return nil, pctx.error(ctx, err)
		}
		curelem = retelem
	}

	var elem string
	for cur.Peek() == '|' {
		if err := cur.Advance(1); err != nil {
			return nil, err
		}
		if elem == "" {
			retelem, err = pctx.doc.CreateElementContent("", ElementContentOr)
			if err != nil {
				return nil, pctx.error(ctx, err)
			}

			retelem.c1 = curelem
			if curelem != nil {
				curelem.parent = retelem
			}
			curelem = retelem
		} else {
			n, err := pctx.doc.CreateElementContent("", ElementContentOr)
			if err != nil {
				return nil, pctx.error(ctx, err)
			}
			n.c1, err = pctx.doc.CreateElementContent(elem, ElementContentElement)
			if err != nil {
				return nil, pctx.error(ctx, err)
			}
			n.c1.parent = n
			curelem.c2 = n
			n.parent = curelem
			curelem = n
		}
		pctx.skipBlanks(ctx)
		elem, err = pctx.parseName(ctx)
		if err != nil {
			return nil, pctx.error(ctx, err)
		}
		pctx.skipBlanks(ctx)
	}
	if cur.Peek() == ')' && cur.PeekAt(1) == '*' {
		if err := cur.Advance(2); err != nil {
			return nil, err
		}
		if elem != "" {
			curelem.c2, err = pctx.doc.CreateElementContent(elem, ElementContentElement)
			if err != nil {
				return nil, pctx.error(ctx, err)
			}
			curelem.c2.parent = curelem
		}

		if retelem != nil {
			retelem.coccur = ElementContentMult
		}
		if pctx.valid && pctx.currentInputID() != startInput {
			_ = pctx.warning(ctx, "element content declaration doesn't start and stop in the same entity\n")
			pctx.valid = false
		}
	}
	return retelem, nil
}

func (pctx *parserCtx) parseElementChildrenContentDeclPriv(ctx context.Context, depth int) (*ElementContent, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseElementChildrenContentDeclPriv(%d)", depth)
		defer g.IRelease("END parseElementChildrenContentDeclPriv(%d)", depth)
	}

	maxDepth := 128
	if pctx.options.IsSet(parseHuge) {
		maxDepth = 2048
	}
	if depth > maxDepth {
		return nil, fmt.Errorf("xmlParseElementChildrenContentDecl : depth %d too deep", depth)
	}
	startInput := pctx.currentInputID()

	var curelem *ElementContent
	var retelem *ElementContent
	pctx.skipBlanks(ctx)
	cur := pctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if cur.Peek() == '(' {
		if err := cur.Advance(1); err != nil {
			return nil, err
		}
		pctx.skipBlanks(ctx)
		var err error
		retelem, err = pctx.parseElementChildrenContentDeclPriv(ctx, depth+1)
		if err != nil {
			return nil, pctx.error(ctx, err)
		}
		curelem = retelem
		pctx.skipBlanks(ctx)
	} else {
		elem, err := pctx.parseName(ctx)
		if err != nil {
			return nil, pctx.error(ctx, err)
		}

		retelem, err = pctx.doc.CreateElementContent(elem, ElementContentElement)
		if err != nil {
			return nil, pctx.error(ctx, err)
		}
		curelem = retelem

		switch cur.Peek() {
		case '?':
			curelem.coccur = ElementContentOpt
			if err := cur.Advance(1); err != nil {
				return nil, err
			}
		case '*':
			curelem.coccur = ElementContentMult
			if err := cur.Advance(1); err != nil {
				return nil, err
			}
		case '+':
			curelem.coccur = ElementContentPlus
			if err := cur.Advance(1); err != nil {
				return nil, err
			}
		}
	}

	pctx.skipBlanks(ctx)

	var sep rune
	var last *ElementContent
	createElementContent := func(c rune, typ ElementContentType) error {
		if sep == 0x0 {
			sep = c
		} else if sep != c {
			return pctx.error(ctx, fmt.Errorf("'%c' expected", sep))
		}
		if err := cur.Advance(1); err != nil {
			return err
		}

		op, err := pctx.doc.CreateElementContent("", typ)
		if err != nil {
			return pctx.error(ctx, err)
		}

		if last == nil {
			op.c1 = retelem
			if retelem != nil {
				retelem.parent = op
			}
			curelem = op
			retelem = op
		} else {
			curelem.c2 = op
			op.parent = curelem
			op.c1 = last
			if last != nil {
				last.parent = op
			}
			curelem = op
			last = nil
		}
		return nil
	}

LOOP:
	for !cur.Done() {
		c := cur.Peek()
		switch c {
		case ')':
			break LOOP
		case ',':
			if err := createElementContent(rune(c), ElementContentSeq); err != nil {
				return nil, pctx.error(ctx, err)
			}
		case '|':
			if err := createElementContent(rune(c), ElementContentOr); err != nil {
				return nil, pctx.error(ctx, err)
			}
		default:
			return nil, pctx.error(ctx, ErrElementContentNotFinished)
		}

		pctx.skipBlanks(ctx)

		if cur.Peek() == '(' {
			if err := cur.Advance(1); err != nil {
				return nil, err
			}
			pctx.skipBlanks(ctx)
			var err error
			last, err = pctx.parseElementChildrenContentDeclPriv(ctx, depth+1)
			if err != nil {
				return nil, pctx.error(ctx, err)
			}
			pctx.skipBlanks(ctx)
		} else {
			elem, err := pctx.parseName(ctx)
			if err != nil {
				return nil, pctx.error(ctx, err)
			}

			last, err = pctx.doc.CreateElementContent(elem, ElementContentElement)
			if err != nil {
				return nil, pctx.error(ctx, err)
			}

			switch cur.Peek() {
			case '?':
				last.coccur = ElementContentOpt
				if err := cur.Advance(1); err != nil {
					return nil, err
				}
			case '*':
				last.coccur = ElementContentMult
				if err := cur.Advance(1); err != nil {
					return nil, err
				}
			case '+':
				last.coccur = ElementContentPlus
				if err := cur.Advance(1); err != nil {
					return nil, err
				}
			}
		}
		pctx.skipBlanks(ctx)
	}
	if last != nil {
		curelem.c2 = last
		last.parent = curelem
	}
	if err := cur.Advance(1); err != nil {
		return nil, pctx.error(ctx, err)
	}
	if pctx.valid && pctx.currentInputID() != startInput {
		_ = pctx.warning(ctx, "element content declaration doesn't start and stop in the same entity\n")
		pctx.valid = false
	}

	c := cur.Peek()
	switch c {
	case '?':
		if retelem != nil {
			if retelem.coccur == ElementContentPlus {
				retelem.coccur = ElementContentMult
			} else {
				retelem.coccur = ElementContentOpt
			}
		}
		if err := cur.Advance(1); err != nil {
			return nil, err
		}
	case '*':
		if retelem != nil {
			retelem.coccur = ElementContentMult
			curelem = retelem
			for curelem != nil && curelem.ctype == ElementContentOr {
				if curelem.c1 != nil && (curelem.c1.coccur == ElementContentOpt || curelem.c1.coccur == ElementContentMult) {
					curelem.c1.coccur = ElementContentOnce
				}
				if curelem.c2 != nil && (curelem.c2.coccur == ElementContentOpt || curelem.c2.coccur == ElementContentMult) {
					curelem.c2.coccur = ElementContentOnce
				}
				curelem = curelem.c2
			}
		}
		if err := cur.Advance(1); err != nil {
			return nil, err
		}
	case '+':
		if retelem.coccur == ElementContentOpt {
			retelem.coccur = ElementContentMult
		} else {
			retelem.coccur = ElementContentPlus
		}

		found := false
		for curelem != nil && curelem.ctype == ElementContentOr {
			if curelem.c1 != nil && (curelem.c1.coccur == ElementContentOpt || curelem.c1.coccur == ElementContentMult) {
				curelem.c1.coccur = ElementContentOnce
				found = true
			}

			if curelem.c2 != nil && (curelem.c2.coccur == ElementContentOpt || curelem.c2.coccur == ElementContentMult) {
				curelem.c2.coccur = ElementContentOnce
				found = true
			}
			curelem = curelem.c2
		}
		if found {
			retelem.coccur = ElementContentMult
		}
		if err := cur.Advance(1); err != nil {
			return nil, err
		}
	}

	return retelem, nil
}
