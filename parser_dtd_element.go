package helium

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/sax"
)

func (pctx *parserCtx) parseElementDecl(ctx context.Context) (enum.ElementType, error) {
	cur := pctx.getCursor()
	if cur == nil {
		return 0, pctx.error(ctx, errNoCursor)
	}
	if !cur.ConsumeString("<!ELEMENT") {
		return enum.UndefinedElementType, pctx.error(ctx, ErrInvalidElementDecl)
	}
	startInput := pctx.currentInputID()

	// Require the mandatory "S" through skipBlanksPE so a parameter entity may
	// supply (or be adjacent to) the element name / content spec in the external
	// subset; its §4.4.8 padding or a crossed PE boundary satisfies the "S". The
	// captured cursor goes stale across a PE expand/pop, so re-fetch it.
	adv, err := pctx.skipBlanksPE(ctx)
	if err != nil {
		return enum.UndefinedElementType, pctx.error(ctx, err)
	}
	if !adv {
		return enum.UndefinedElementType, pctx.error(ctx, ErrSpaceRequired)
	}

	name, err := pctx.parseName(ctx)
	if err != nil {
		return enum.UndefinedElementType, pctx.error(ctx, err)
	}

	adv, err = pctx.skipBlanksPE(ctx)
	if err != nil {
		return enum.UndefinedElementType, pctx.error(ctx, err)
	}
	if !adv {
		return enum.UndefinedElementType, pctx.error(ctx, ErrSpaceRequired)
	}

	cur = pctx.dtdRefetch(cur)
	if cur == nil {
		return enum.UndefinedElementType, pctx.error(ctx, errNoCursor)
	}
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

	if _, err := pctx.skipBlanksPE(ctx); err != nil {
		return enum.UndefinedElementType, pctx.error(ctx, err)
	}

	cur = pctx.dtdRefetch(cur)
	if cur == nil {
		return enum.UndefinedElementType, pctx.error(ctx, errNoCursor)
	}
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
	cur := pctx.getCursor()
	if cur == nil {
		return nil, 0, pctx.error(ctx, errNoCursor)
	}
	if cur.Peek() != '(' {
		return nil, enum.UndefinedElementType, pctx.error(ctx, ErrOpenParenRequired)
	}
	// The input holding this opening '(' is the boundary reference for the
	// group's PE-nesting validity check (XML VC "Proper Group/PE Nesting"): the
	// matching ')' must be read from this same input. Capture it BEFORE advancing
	// so a parameter entity supplying the group's content (e.g. "(%m;)") does not
	// shift the reference to the PE input.
	openInput := pctx.currentInputID()
	if err := cur.Advance(1); err != nil {
		return nil, enum.UndefinedElementType, err
	}

	if pctx.instate == psEOF {
		return nil, enum.UndefinedElementType, pctx.error(ctx, ErrEOF)
	}

	if _, err := pctx.skipBlanksPE(ctx); err != nil {
		return nil, enum.UndefinedElementType, pctx.error(ctx, err)
	}

	cur = pctx.dtdRefetch(cur)
	if cur == nil {
		return nil, enum.UndefinedElementType, pctx.error(ctx, errNoCursor)
	}
	var ec *ElementContent
	var err error
	var etype enum.ElementType
	if cur.HasPrefixString("#PCDATA") {
		ec, err = pctx.parseElementMixedContentDecl(ctx, openInput)
		if err != nil {
			return nil, enum.UndefinedElementType, pctx.error(ctx, err)
		}
		etype = enum.MixedElementType
	} else {
		ec, err = pctx.parseElementChildrenContentDeclPriv(ctx, openInput, 0)
		if err != nil {
			return nil, enum.UndefinedElementType, pctx.error(ctx, err)
		}
		etype = enum.ElementElementType
	}

	if _, err := pctx.skipBlanksPE(ctx); err != nil {
		return nil, enum.UndefinedElementType, pctx.error(ctx, err)
	}
	return ec, etype, nil
}

func (pctx *parserCtx) parseElementMixedContentDecl(ctx context.Context, openInput any) (*ElementContent, error) {
	cur := pctx.getCursor()
	if cur == nil {
		return nil, pctx.error(ctx, errNoCursor)
	}
	if !cur.ConsumeString("#PCDATA") {
		return nil, pctx.error(ctx, ErrPCDATARequired)
	}

	if _, err := pctx.skipBlanksPE(ctx); err != nil {
		return nil, pctx.error(ctx, err)
	}

	cur = pctx.dtdRefetch(cur)
	if cur == nil {
		return nil, pctx.error(ctx, errNoCursor)
	}
	if cur.Peek() == ')' {
		// XML VC "Proper Group/PE Nesting": the group's matching ')' must be read
		// from the SAME input as its '('. A ')' supplied by a different
		// parameter-entity replacement text (the '(' in one PE / entity, the ')' in
		// another, or split between a PE and the containing DTD) is a boundary
		// violation, the same fatal condition the <!ELEMENT> wrapper enforces.
		if pctx.currentInputID() != openInput {
			return nil, pctx.error(ctx,
				fmt.Errorf("%w: element content declaration doesn't start and stop in the same entity", ErrEntityBoundary))
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
		if _, err := pctx.skipBlanksPE(ctx); err != nil {
			return nil, pctx.error(ctx, err)
		}
		elem, err = pctx.parseName(ctx)
		if err != nil {
			return nil, pctx.error(ctx, err)
		}
		if _, err := pctx.skipBlanksPE(ctx); err != nil {
			return nil, pctx.error(ctx, err)
		}
		cur = pctx.dtdRefetch(cur)
		if cur == nil {
			return nil, pctx.error(ctx, errNoCursor)
		}
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
		if pctx.currentInputID() != openInput {
			return nil, pctx.error(ctx,
				fmt.Errorf("%w: element content declaration doesn't start and stop in the same entity", ErrEntityBoundary))
		}
	}
	return retelem, nil
}

func (pctx *parserCtx) parseElementChildrenContentDeclPriv(ctx context.Context, openInput any, depth int) (*ElementContent, error) {
	if pctx.maxCMDepth > 0 && depth > pctx.maxCMDepth {
		return nil, fmt.Errorf("xmlParseElementChildrenContentDecl : depth %d too deep", depth)
	}

	var curelem *ElementContent
	var retelem *ElementContent
	if _, err := pctx.skipBlanksPE(ctx); err != nil {
		return nil, pctx.error(ctx, err)
	}
	cur := pctx.getCursor()
	if cur == nil {
		return nil, pctx.error(ctx, errNoCursor)
	}
	if cur.Peek() == '(' {
		// A nested group: the input holding THIS '(' is the boundary reference for
		// the nested group's own ')'.
		nestedInput := pctx.currentInputID()
		if err := cur.Advance(1); err != nil {
			return nil, err
		}
		if _, err := pctx.skipBlanksPE(ctx); err != nil {
			return nil, pctx.error(ctx, err)
		}
		var err error
		retelem, err = pctx.parseElementChildrenContentDeclPriv(ctx, nestedInput, depth+1)
		if err != nil {
			return nil, pctx.error(ctx, err)
		}
		curelem = retelem
		if _, err := pctx.skipBlanksPE(ctx); err != nil {
			return nil, pctx.error(ctx, err)
		}
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

		cur = pctx.dtdRefetch(cur)
		if cur == nil {
			return nil, pctx.error(ctx, errNoCursor)
		}
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

	if _, err := pctx.skipBlanksPE(ctx); err != nil {
		return nil, pctx.error(ctx, err)
	}
	cur = pctx.dtdRefetch(cur)
	if cur == nil {
		return nil, pctx.error(ctx, errNoCursor)
	}

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
	for {
		cur = pctx.dtdRefetch(cur)
		if cur == nil || cur.Done() {
			break
		}
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

		if _, err := pctx.skipBlanksPE(ctx); err != nil {
			return nil, pctx.error(ctx, err)
		}
		cur = pctx.dtdRefetch(cur)
		if cur == nil {
			return nil, pctx.error(ctx, errNoCursor)
		}

		if cur.Peek() == '(' {
			nestedInput := pctx.currentInputID()
			if err := cur.Advance(1); err != nil {
				return nil, err
			}
			if _, err := pctx.skipBlanksPE(ctx); err != nil {
				return nil, pctx.error(ctx, err)
			}
			var err error
			last, err = pctx.parseElementChildrenContentDeclPriv(ctx, nestedInput, depth+1)
			if err != nil {
				return nil, pctx.error(ctx, err)
			}
			if _, err := pctx.skipBlanksPE(ctx); err != nil {
				return nil, pctx.error(ctx, err)
			}
		} else {
			elem, err := pctx.parseName(ctx)
			if err != nil {
				return nil, pctx.error(ctx, err)
			}

			last, err = pctx.doc.CreateElementContent(elem, ElementContentElement)
			if err != nil {
				return nil, pctx.error(ctx, err)
			}

			cur = pctx.dtdRefetch(cur)
			if cur == nil {
				return nil, pctx.error(ctx, errNoCursor)
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
		if _, err := pctx.skipBlanksPE(ctx); err != nil {
			return nil, pctx.error(ctx, err)
		}
	}
	if last != nil {
		curelem.c2 = last
		last.parent = curelem
	}
	cur = pctx.dtdRefetch(cur)
	if cur == nil {
		return nil, pctx.error(ctx, errNoCursor)
	}
	if pctx.currentInputID() != openInput {
		return nil, pctx.error(ctx,
			fmt.Errorf("%w: element content declaration doesn't start and stop in the same entity", ErrEntityBoundary))
	}
	if err := cur.Advance(1); err != nil {
		return nil, pctx.error(ctx, err)
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
