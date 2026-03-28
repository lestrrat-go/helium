package helium

import (
	"bytes"
	"context"
	"fmt"

	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/internal/strcursor"
	"github.com/lestrrat-go/helium/sax"
	"github.com/lestrrat-go/pdebug"
)

func (pctx *parserCtx) parseDocTypeDecl(ctx context.Context) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseDocTypeDecl")
		defer g.IRelease("END parseDocTypeDecl")
	}

	cur := pctx.getCursor()
	if cur == nil {
		return pctx.error(ctx, errNoCursor)
	}
	if !cur.ConsumeString("<!DOCTYPE") {
		return pctx.error(ctx, ErrInvalidDTD)
	}

	pctx.skipBlanks(ctx)

	name, err := pctx.parseName(ctx)
	if err != nil {
		return pctx.error(ctx, ErrDocTypeNameRequired)
	}
	pctx.intSubName = name

	pctx.skipBlanks(ctx)
	u, eid, err := pctx.parseExternalID(ctx)
	if err != nil {
		return pctx.error(ctx, err)
	}

	if u != "" || eid != "" {
		pctx.hasExternalSubset = true
	}
	pctx.extSubURI = u
	pctx.extSubSystem = eid

	pctx.skipBlanks(ctx)

	if s := pctx.sax; s != nil {
		switch err := s.InternalSubset(ctx, name, eid, u); err {
		case nil, sax.ErrHandlerUnspecified:
		default:
			return pctx.error(ctx, err)
		}
	}

	c := cur.Peek()
	if c == '[' {
		return nil
	}

	if c != '>' {
		return pctx.error(ctx, ErrDocTypeNotFinished)
	}
	if err := cur.Advance(1); err != nil {
		return err
	}

	return nil
}

func (pctx *parserCtx) parseInternalSubset(ctx context.Context) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseInternalSubset")
		defer g.IRelease("END parseInternalSubset")
	}

	cur := pctx.getCursor()
	if cur == nil {
		return pctx.error(ctx, errNoCursor)
	}
	if cur.Peek() != '[' {
		goto FinishDTD
	}
	pctx.instate = psDTD
	if err := cur.Advance(1); err != nil {
		return err
	}

	for {
		if pctx.stopped {
			return errParserStopped
		}
		cur = pctx.getCursor()
		if cur == nil || cur.Done() || cur.Peek() == ']' {
			break
		}

		startCur := cur
		startLine := cur.LineNumber()
		startCol := cur.Column()
		startByte := cur.Peek()

		pctx.skipBlanks(ctx)
		if err := pctx.parseMarkupDecl(ctx); err != nil {
			return pctx.error(ctx, err)
		}
		if err := pctx.parsePEReference(ctx); err != nil {
			return pctx.error(ctx, err)
		}

		cur = pctx.getCursor()
		if cur == startCur && cur != nil && cur.LineNumber() == startLine && cur.Column() == startCol && cur.Peek() == startByte {
			return pctx.error(ctx, ErrDocTypeNotFinished)
		}
	}

	cur = pctx.getCursor()
	if cur != nil && cur.Peek() == ']' {
		if err := cur.Advance(1); err != nil {
			return err
		}
		pctx.skipBlanks(ctx)
	}

FinishDTD:
	cur = pctx.getCursor()
	if cur != nil && cur.Peek() != '>' {
		return pctx.error(ctx, ErrDocTypeNotFinished)
	}
	if cur != nil {
		if err := cur.Advance(1); err != nil {
			return err
		}
	}

	return nil
}

func (pctx *parserCtx) parseMarkupDecl(ctx context.Context) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseMarkupDecl")
		defer g.IRelease("END parseMarkupDecl")
	}

	cur := pctx.getCursor()
	if cur == nil {
		return pctx.error(ctx, errNoCursor)
	}
	if cur.Peek() == '<' {
		if cur.PeekAt(1) == '!' {
			switch cur.PeekAt(2) {
			case 'E':
				switch c := cur.PeekAt(3); c {
				case 'L':
					if _, err := pctx.parseElementDecl(ctx); err != nil {
						return pctx.error(ctx, err)
					}
				case 'N':
					if err := pctx.parseEntityDecl(ctx); err != nil {
						return pctx.error(ctx, err)
					}
				}
			case 'A':
				if err := pctx.parseAttributeListDecl(ctx); err != nil {
					return pctx.error(ctx, err)
				}
			case 'N':
				if err := pctx.parseNotationDecl(ctx); err != nil {
					return pctx.error(ctx, err)
				}
			case '-':
				if err := pctx.parseComment(ctx); err != nil {
					return pctx.error(ctx, err)
				}
			}
		} else if cur.PeekAt(1) == '?' {
			return pctx.parsePI(ctx)
		}
	}

	if pctx.instate == psEOF {
		return nil
	}

	if !pctx.external && pctx.inputTab.Len() == 1 {
		if err := pctx.parsePEReference(ctx); err != nil {
			return pctx.error(ctx, err)
		}
	}
	if !pctx.external && pctx.inputTab.Len() > 1 {
		cur = pctx.getCursor()
		if cur != nil && cur.Peek() == '<' && cur.PeekAt(1) == '!' && cur.PeekAt(2) == '[' {
			if err := pctx.parseConditionalSections(ctx); err != nil {
				return pctx.error(ctx, err)
			}
			return nil
		}
	}
	pctx.instate = psDTD

	return nil
}

func (pctx *parserCtx) parseConditionalSections(ctx context.Context) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseConditionalSections")
		defer g.IRelease("END parseConditionalSections")
	}

	cur := pctx.getCursor()
	if cur == nil {
		return ErrPrematureEOF
	}

	if err := cur.Advance(3); err != nil {
		return err
	}

	pctx.skipBlanks(ctx)

	cur = pctx.getCursor()
	if cur != nil && cur.Peek() == '%' {
		if err := pctx.parsePEReference(ctx); err != nil {
			return err
		}
		pctx.skipBlanks(ctx)
	}

	cur = pctx.getCursor()
	if cur == nil {
		return ErrPrematureEOF
	}

	if cur.HasPrefixString("INCLUDE") {
		if err := cur.Advance(7); err != nil {
			return err
		}
		pctx.skipBlanks(ctx)
		cur = pctx.getCursor()
		if cur == nil || cur.Peek() != '[' {
			return ErrConditionalSectionKeyword
		}
		if err := cur.Advance(1); err != nil {
			return err
		}

		for {
			pctx.skipBlanks(ctx)
			cur = pctx.getCursor()
			if cur == nil || cur.Done() {
				return ErrConditionalSectionNotFinished
			}

			if cur.Peek() == ']' && cur.PeekAt(1) == ']' && cur.PeekAt(2) == '>' {
				if err := cur.Advance(3); err != nil {
					return err
				}
				return nil
			}

			if cur.Peek() == '<' && cur.PeekAt(1) == '!' && cur.PeekAt(2) == '[' {
				if err := pctx.parseConditionalSections(ctx); err != nil {
					return err
				}
				continue
			}

			if err := pctx.parseMarkupDecl(ctx); err != nil {
				return err
			}

			cur = pctx.getCursor()
			if cur != nil && cur.Peek() == '%' {
				if err := pctx.parsePEReference(ctx); err != nil {
					return err
				}
			}
		}
	}

	if cur.HasPrefixString("IGNORE") {
		if err := cur.Advance(6); err != nil {
			return err
		}
		pctx.skipBlanks(ctx)
		cur = pctx.getCursor()
		if cur == nil || cur.Peek() != '[' {
			return ErrConditionalSectionKeyword
		}
		if err := cur.Advance(1); err != nil {
			return err
		}

		depth := 1
		for depth > 0 {
			cur = pctx.getCursor()
			if cur == nil || cur.Done() {
				return ErrConditionalSectionNotFinished
			}

			c := cur.Peek()
			if c == '<' && cur.PeekAt(1) == '!' && cur.PeekAt(2) == '[' {
				depth++
				if err := cur.Advance(3); err != nil {
					return err
				}
				continue
			}
			if c == ']' && cur.PeekAt(1) == ']' && cur.PeekAt(2) == '>' {
				depth--
				if err := cur.Advance(3); err != nil {
					return err
				}
				continue
			}
			if err := cur.Advance(1); err != nil {
				return err
			}
		}
		return nil
	}

	return ErrConditionalSectionKeyword
}

func (pctx *parserCtx) parsePEReference(ctx context.Context) error {
	if pdebug.Enabled {
		g := pdebug.Marker("parsePEReference")
		defer g.End()
	}

	cur := pctx.getCursor()
	if cur == nil {
		return pctx.error(ctx, errNoCursor)
	}
	if cur.Peek() != '%' {
		if pdebug.Enabled {
			pdebug.Printf("no parameter entities here, returning...")
		}
		return nil
	}
	if err := cur.Advance(1); err != nil {
		return err
	}

	name, err := pctx.parseName(ctx)
	if err != nil {
		return pctx.error(ctx, err)
	}

	if cur.Peek() != ';' {
		return pctx.error(ctx, ErrSemicolonRequired)
	}
	if err := cur.Advance(1); err != nil {
		return err
	}

	var entity sax.Entity
	if s := pctx.sax; s != nil {
		_ = pctx.fireSAXCallback(ctx, cbGetParameterEntity, &entity, name)
	}

	if pctx.instate == psEOF {
		return nil
	}

	if entity == nil {
		if pctx.standalone == StandaloneExplicitYes || (!pctx.hasExternalSubset && !pctx.hasPERefs) {
			return fmt.Errorf("parse error: PEReference: %%%s; not found", name)
		}
		if err := pctx.warning(ctx, "PEReference: %%%s; not found\n", name); err != nil {
			return err
		}
		pctx.valid = false
		if err := pctx.entityCheck(entity, 0); err != nil {
			return pctx.error(ctx, err)
		}
	} else {
		if etype := entity.EntityType(); etype != enum.InternalParameterEntity && etype != enum.ExternalParameterEntity {
			if err := pctx.warning(ctx, "Internal: %%%s; is not a parameter entity\n", name); err != nil {
				return err
			}
		} else {
			if pdebug.Enabled {
				pdebug.Printf("Expanding parameter entity '%s' with content: %s", name, string(entity.Content()))
			}

			decodedContent, err := pctx.decodeEntities(ctx, entity.Content(), SubstituteBoth)
			if err != nil {
				return fmt.Errorf("failed to decode parameter entity content: %v", err)
			}

			if pdebug.Enabled {
				pdebug.Printf("Decoded parameter entity content: %s", decodedContent)
			}

			pctx.pushInput(strcursor.NewByteCursor(bytes.NewReader([]byte(decodedContent))))
		}
	}
	pctx.hasPERefs = true
	return nil
}
