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

		// The INCLUDE body is parsed with the SAME shared declaration step the
		// top-level external subset uses, so parameter-entity references inside
		// the section expand uniformly (a "%pe;" providing a defaulting
		// <!ATTLIST> must be applied, not consumed-without-expansion by
		// skipBlanks). baseLen is the input-stack depth at body entry: a PE that
		// expands inside the section pushes a nested cursor and is popped back to
		// baseLen when exhausted, after which the "]]>" terminator (which lives in
		// this section's own cursor) is examined again.
		baseLen := pctx.inputTab.Len()
		for {
			// Pop spent nested PE/conditional cursors and skip leading blanks on
			// the section's own cursor so the "]]>" terminator and EOF are checked
			// against the enclosing cursor, not an exhausted PE cursor.
			pctx.popSpentExternalSubsetInputs(baseLen)
			if pctx.inputTab.Len() <= baseLen {
				// Inspect the section's OWN cursor (the floor cursor at baseLen-1)
				// directly rather than via getCursor(): if this external DTD's
				// INCLUDE section reaches EOF before its "]]>" terminator,
				// getCursor() would auto-pop the exhausted section cursor and
				// return the enclosing (e.g. main document) cursor, which is not
				// Done — defeating the EOF check and spinning this loop forever.
				sec := pctx.adaptCursor(pctx.inputTab.PeekOne())
				if sec == nil {
					return ErrConditionalSectionNotFinished
				}

				n := 0
				for b := sec.PeekAt(n); isBlankByte(b) && !sec.Done(); b = sec.PeekAt(n) {
					n++
				}
				if n > 0 {
					if err := sec.Advance(n); err != nil {
						return err
					}
				}

				if sec.Done() {
					return ErrConditionalSectionNotFinished
				}

				if sec.Peek() == ']' && sec.PeekAt(1) == ']' && sec.PeekAt(2) == '>' {
					if err := sec.Advance(3); err != nil {
						return err
					}
					return nil
				}
			}

			stop, err := pctx.parseExternalSubsetDeclStep(ctx, baseLen, false)
			if err != nil {
				return err
			}
			// stop=true means the section's own content cursor is exhausted
			// before a "]]>" terminator was seen. Report the unterminated
			// conditional section instead of looping forever.
			if stop {
				return ErrConditionalSectionNotFinished
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

// popSpentExternalSubsetInputs pops any exhausted (Done) parameter-entity or
// conditional-section cursors that sit above baseLen on the input stack, so the
// next declaration resumes in the parent DTD where the expanded content left
// off. It stops at the first non-exhausted cursor or once the stack is back at
// baseLen. Breaking a declaration loop on a Done() nested cursor instead of
// popping it would let the deferred external-subset cleanup pop the PARENT DTD
// cursor too, silently discarding declarations that follow a PE reference.
func (pctx *parserCtx) popSpentExternalSubsetInputs(baseLen int) {
	for pctx.inputTab.Len() > baseLen {
		top := pctx.adaptCursor(pctx.inputTab.PeekOne())
		if top == nil || !top.Done() {
			return
		}
		pctx.popInput()
	}
}

// parseExternalSubsetDeclStep parses one declaration's worth of the external
// subset: a blank-only skip, an explicit parameter-entity expansion, a markup
// declaration or a nested conditional section, plus the spent-cursor cleanup
// and forward-progress guard. It is shared by BOTH the top-level external
// subset loop (TreeBuilder.ExternalSubset) and the body of an external
// <![INCLUDE[ ... ]]> conditional section so PE references expand uniformly in
// both — the previous INCLUDE loop used skipBlanks, whose handlePEReference
// consumes a "%pe;" reference WITHOUT pushing its replacement text, silently
// dropping a defaulting <!ATTLIST> supplied by that PE.
//
// baseLen is the input-stack depth of the ENCLOSING CONTENT CURSOR — the pushed
// DTD cursor for the top-level subset, or the section's own cursor for an
// INCLUDE body. The step pops spent nested PE/conditional cursors (those
// strictly above the floor) back down to it. It returns stop=true once that
// content cursor is exhausted (the stack dropped below the floor, or the floor
// cursor is gone/Done) — or, when tolerateCondError is set, when a nested
// conditional section reports an error — signalling the caller to stop or
// resume its scan.
//
// tolerateCondError mirrors the long-standing top-level external-subset
// behavior: a per-conditional-section error stops the loop WITHOUT failing the
// whole parse, which valid documents whose conditional-section handling is
// otherwise imperfect rely on. The INCLUDE-body caller passes false so a nested
// conditional-section error propagates instead.
//
// Unlike skipBlanks, the blank skip here advances over whitespace ONLY and
// leaves any "%" for the explicit parsePEReference below. In the external
// subset, skipBlanks calls handlePEReference, which CONSUMES a "%pe;" reference
// while only validating it (it does not expand the replacement text). That
// swallows the reference before parsePEReference can push the PE content onto
// the input stack, so the PE's declarations are never parsed.
func (pctx *parserCtx) parseExternalSubsetDeclStep(ctx context.Context, baseLen int, tolerateCondError bool) (bool, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseExternalSubsetDeclStep")
		defer g.IRelease("END parseExternalSubsetDeclStep")
	}

	// Snapshot the cursor position BEFORE consuming blanks so the progress guard
	// below counts everything this step does — whitespace, a markup declaration,
	// AND a parameter-entity reference — as forward progress. The guard must
	// still fire when a step makes no progress at all (e.g. a malformed
	// "<!BOGUS" that parseMarkupDecl ignores).
	startCur := pctx.getCursor()
	var startLine, startCol int
	var startByte byte
	if startCur != nil {
		startLine = startCur.LineNumber()
		startCol = startCur.Column()
		startByte = startCur.Peek()
	}

	// Blank-only skip (see method doc): advance over whitespace, leaving "%" for
	// parsePEReference. Do NOT route through skipBlanks here.
	if c := pctx.getCursor(); c != nil {
		n := 0
		for b := c.PeekAt(n); isBlankByte(b) && !c.Done(); b = c.PeekAt(n) {
			n++
		}
		if n > 0 {
			if err := c.Advance(n); err != nil {
				return false, err
			}
		}
	}

	// The blank skip may have consumed the LAST bytes of a parameter-entity
	// cursor whose replacement text is (or ends with) only whitespace, e.g.
	// `<!ENTITY % ws "   ">` then `%ws;`. Pop the spent NESTED cursors (those
	// strictly above the floor) so the next read resumes in the enclosing
	// content cursor where the expanded content left off.
	pctx.popSpentExternalSubsetInputs(baseLen)
	// Stop once the enclosing content cursor is exhausted: the stack dropped
	// below the floor, or the floor cursor itself is gone/Done. The floor
	// (baseLen) is the depth of the content cursor — the pushed DTD cursor for
	// the top-level subset, or the section's own cursor for an INCLUDE body.
	if pctx.inputTab.Len() < baseLen {
		return true, nil
	}
	cur := pctx.getCursor()
	if cur == nil || (pctx.inputTab.Len() == baseLen && cur.Done()) {
		return true, nil
	}

	cur = pctx.getCursor()
	if cur != nil && cur.Peek() == '<' && cur.PeekAt(1) == '!' && cur.PeekAt(2) == '[' {
		// Nested conditional section. parseConditionalSections is responsible for
		// its own blank/PE handling within the section.
		if err := pctx.parseConditionalSections(ctx); err != nil {
			if tolerateCondError {
				return true, nil
			}
			return false, err
		}
	} else {
		if err := pctx.parseMarkupDecl(ctx); err != nil {
			return false, err
		}

		// Expand a parameter-entity reference at the cursor. parseMarkupDecl does
		// not handle top-level "%pe;" references in the external subset, so this
		// pushes the PE replacement text onto the input stack and lets its
		// declarations be parsed by subsequent steps.
		if err := pctx.parsePEReference(ctx); err != nil {
			return false, err
		}
	}

	// Pop any exhausted parameter-entity (or conditional-section) cursors so the
	// next step resumes in the parent DTD where the expanded content left off.
	pctx.popSpentExternalSubsetInputs(baseLen)

	// Guard against a step that neither advanced the cursor nor reported an
	// error, which would otherwise loop forever (e.g. the malformed "<!BOGUS"
	// declaration that parseMarkupDecl ignores).
	cur = pctx.getCursor()
	if cur == startCur && cur != nil && cur.LineNumber() == startLine && cur.Column() == startCol && cur.Peek() == startByte {
		return false, pctx.error(ctx, ErrDocTypeNotFinished)
	}

	return false, nil
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

			// Capture the PE's replacement text once: Entity.Content()
			// allocates a fresh []byte copy on every call, so we reuse this
			// local for both decoding and the amplification accounting below.
			content := entity.Content()

			decodedContent, err := pctx.decodeEntities(ctx, content, SubstituteBoth)
			if err != nil {
				return fmt.Errorf("failed to decode parameter entity content: %v", err)
			}

			if pdebug.Enabled {
				pdebug.Printf("Decoded parameter entity content: %s", decodedContent)
			}

			// Charge this PE's OWN replacement bytes before pushing it as new
			// input. Without this the PE's direct contribution is free, so a
			// small DTD that references a large PE many times could drive
			// unbounded expansion past the amplification limit.
			//
			// Charge len(content) (the PE's stored replacement text), NOT
			// len(decodedContent): decodeEntities(SubstituteBoth) above already
			// charged every nested entity expansion it performed — general
			// references (&g;) left literal in the stored value, and any
			// parameter references — via its own entityCheck calls.
			// decodedContent is the result AFTER those nested expansions, so
			// charging its length here would double-count those nested bytes and
			// could falsely reject a legitimate DTD whose %p; expands mostly
			// through a nested entity. content is the direct bytes this PE itself
			// contributes.
			if err := pctx.entityCheck(entity, len(content)); err != nil {
				return pctx.error(ctx, err)
			}

			pctx.pushInput(strcursor.NewByteCursor(bytes.NewReader([]byte(decodedContent))))
		}
	}
	pctx.hasPERefs = true
	return nil
}
