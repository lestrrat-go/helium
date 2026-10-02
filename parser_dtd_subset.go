package helium

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/internal/encoding"
	"github.com/lestrrat-go/helium/internal/iolimit"
	"github.com/lestrrat-go/helium/internal/strcursor"
	"github.com/lestrrat-go/helium/sax"
)

func (pctx *parserCtx) parseDocTypeDecl(ctx context.Context) error {
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
	// A DOCTYPE's ExternalID is optional (an internal-subset-only doctype has
	// none). Its PRESENCE — not a non-empty literal — marks an external subset:
	// a present-but-empty `SYSTEM ""` is still an external ID (found), so the DTD
	// is not fully internal.
	u, eid, found, err := pctx.parseExternalID(ctx, true)
	if err != nil {
		return pctx.error(ctx, err)
	}

	if found {
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
	cur := pctx.getCursor()
	if cur == nil {
		return pctx.error(ctx, errNoCursor)
	}
	// Depth of the input holding the '[' that opened the internal subset. Only
	// a ']' on that same input closes the subset: a parameter-entity reference
	// between declarations must have replacement text matching extSubsetDecl
	// (XML §2.8 WFC: PE Between Declarations), which cannot contain the "]>"
	// that ends the DOCTYPE. libxml2 keeps parsing declarations while the input
	// stack is deeper than this, and reports the stray ']' as "error detected
	// in Markup declaration".
	baseDepth := pctx.inputTab.Len()
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
		if err := ctx.Err(); err != nil {
			return err
		}
		cur = pctx.getCursor()
		if cur == nil || cur.Done() {
			break
		}
		if cur.Peek() == ']' {
			if pctx.inputTab.Len() > baseDepth {
				return pctx.error(ctx, ErrDocTypeNotFinished)
			}
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
		if err := pctx.parsePEReference(ctx, false); err != nil {
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
		if err := pctx.parsePEReference(ctx, false); err != nil {
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
	cur := pctx.getCursor()
	if cur == nil {
		return ErrPrematureEOF
	}

	if err := cur.Advance(3); err != nil {
		return err
	}

	// Depth of the section's OWN cursor (the one holding "<![ ... ]]>"). The
	// INCLUDE/IGNORE keyword may be supplied by a parameter entity pushed above
	// this depth; that spent PE cursor is popped back to here before the body
	// floor is captured, so the body is not mistaken for already-consumed.
	sectionDepth := pctx.inputTab.Len()

	// Blank-ONLY skip after "<![" — NOT skipBlanks, whose handlePEReference would
	// CONSUME a "%pe;" that supplies the INCLUDE/IGNORE keyword (`<![ %e;` with
	// %e; -> "INCLUDE[") without expanding it, so the keyword would be lost and
	// the whole section dropped. Leaving the "%" for the explicit parsePEReference
	// below pushes the replacement text so the keyword (and body) are parsed.
	// skipBlankRun still records an over-cap whitespace run in pctx.blankRunErr,
	// which must be surfaced here so the specific resource-limit error is reported
	// at its source, in place of a generic conditional-section sentinel
	// (ErrConditionalSectionKeyword / ErrConditionalSectionNotFinished).
	if c := pctx.getCursor(); c != nil {
		if _, err := pctx.skipBlankRun(ctx, c); err != nil {
			return err
		}
	}
	if pctx.blankRunErr != nil {
		return pctx.blankRunErr
	}

	cur = pctx.getCursor()
	if cur != nil && cur.Peek() == '%' {
		if err := pctx.parsePEReference(ctx, false); err != nil {
			return err
		}
		pctx.skipBlanks(ctx)
		if pctx.blankRunErr != nil {
			return pctx.blankRunErr
		}
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
		if pctx.blankRunErr != nil {
			return pctx.blankRunErr
		}
		cur = pctx.getCursor()
		if cur == nil || cur.Peek() != '[' {
			return ErrConditionalSectionKeyword
		}
		if err := pctx.checkCondSectionEntityBoundary(ctx, sectionDepth); err != nil {
			return err
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
		// The INCLUDE keyword and its '[' may have come from a parameter entity
		// (`<![ %e;` with %e; -> "INCLUDE[") whose replacement-text cursor is now
		// fully consumed. Pop that spent PE cursor back to the section's own cursor
		// BEFORE capturing the body floor: otherwise baseLen counts the spent PE
		// cursor, popping it later drops the stack below baseLen, and the body
		// declarations (e.g. a defaulting <!ATTLIST>) are silently skipped.
		pctx.popSpentExternalSubsetInputs(sectionDepth)
		baseLen := pctx.inputTab.Len()
		// crossed records that the section's content input ended before its
		// "]]>", so the terminator comes from a different entity and
		// checkCondSectionClose rejects it.
		crossed := false
		for {
			// Pop spent nested PE/conditional cursors so the "]]>" terminator
			// and EOF are checked against a live cursor.
			pctx.popSpentExternalSubsetInputs(baseLen)
			// A declaration step reads through getCursor(), which auto-pops an
			// exhausted input. When the section's own content input ran out
			// that way, the stack is now BELOW baseLen and the top cursor is
			// the input that referenced the parameter entity holding the
			// section. That input must not silently supply the "]]>".
			if pctx.inputTab.Len() < baseLen {
				n, err := pctx.resumeCondSectionBelow(ctx)
				if err != nil {
					return err
				}
				baseLen = n
				crossed = true
				continue
			}

			// Inspect the top cursor directly, bypassing getCursor(), which
			// would auto-pop an exhausted section cursor and return the
			// enclosing (e.g. main document) cursor.
			sec := pctx.adaptCursor(pctx.inputTab.PeekOne())
			if sec == nil {
				return ErrConditionalSectionNotFinished
			}

			// Bounded blank skip (NOT skipBlanks, which would consume a
			// "%pe;" reference without expanding it). skipBlankRun only
			// advances over whitespace, so it is safe here and caps an
			// oversized blank run inside the section with
			// ErrNodeContentTooLarge.
			if _, err := pctx.skipBlankRun(ctx, sec); err != nil {
				return err
			}

			if sec.Done() {
				// A spent parameter entity inside the section body: pop it
				// at the top of the loop and continue in the section.
				if pctx.inputTab.Len() > baseLen {
					continue
				}
				n, err := pctx.resumeCondSectionBelow(ctx)
				if err != nil {
					return err
				}
				baseLen = n
				crossed = true
				continue
			}

			if sec.Peek() == ']' && sec.PeekAt(1) == ']' && sec.PeekAt(2) == '>' {
				// A "]]>" above baseLen was supplied by a parameter entity
				// referenced inside the section body.
				if err := pctx.checkCondSectionClose(ctx, crossed || pctx.inputTab.Len() > baseLen); err != nil {
					return err
				}
				if err := sec.Advance(3); err != nil {
					return err
				}
				return nil
			}

			// stop=true means the section's content cursor is exhausted
			// before a "]]>" terminator was seen; the top of the loop resumes
			// below it or reports the unterminated section.
			if _, err := pctx.parseExternalSubsetDeclStep(ctx, baseLen); err != nil {
				return err
			}
		}
	}

	if cur.HasPrefixString("IGNORE") {
		if err := cur.Advance(6); err != nil {
			return err
		}
		pctx.skipBlanks(ctx)
		if pctx.blankRunErr != nil {
			return pctx.blankRunErr
		}
		cur = pctx.getCursor()
		if cur == nil || cur.Peek() != '[' {
			return ErrConditionalSectionKeyword
		}
		if err := pctx.checkCondSectionEntityBoundary(ctx, sectionDepth); err != nil {
			return err
		}
		if err := cur.Advance(1); err != nil {
			return err
		}

		// An ignored section recognizes no parameter-entity references, so its
		// body never pushes an input: it is read from the section's content
		// input until that runs out. The cursor is read from the stack
		// directly, not through getCursor(), which would auto-pop an exhausted
		// content input and let the input below it close the section.
		pctx.popSpentExternalSubsetInputs(sectionDepth)
		crossed := false
		depth := 1
		for depth > 0 {
			cur = pctx.adaptCursor(pctx.inputTab.PeekOne())
			if cur == nil {
				return ErrConditionalSectionNotFinished
			}
			if cur.Done() {
				if _, err := pctx.resumeCondSectionBelow(ctx); err != nil {
					return err
				}
				crossed = true
				continue
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
				if depth == 0 {
					if err := pctx.checkCondSectionClose(ctx, crossed); err != nil {
						return err
					}
				}
				if err := cur.Advance(3); err != nil {
					return err
				}
				continue
			}
			r, width, ok := decodeRuneAt(cur, 0)
			if !ok || !pctx.isLiteralCharWidth(r, width) {
				return ErrInvalidChar
			}
			if err := cur.Advance(width); err != nil {
				return err
			}
		}
		return nil
	}

	return ErrConditionalSectionKeyword
}

// checkCondSectionEntityBoundary enforces the opening side of the "Proper
// Conditional Section/PE Nesting" validity constraint (XML §3.4): the "<![",
// the INCLUDE/IGNORE keyword, and the "[" that open a conditional section must
// all come from the same parameter-entity replacement text. sectionDepth is the
// input-stack depth of the cursor holding "<!["; when a parameter entity pushed
// above it supplied the keyword and "[", the input stack is deeper than
// sectionDepth. The spec classifies this as a validity constraint, so helium
// reports it only when validating. This intentionally differs from libxml2
// 2.9.14, which reports the same markup as a fatal parser error ("All markup of
// the conditional section is not in the same entity") even without validation.
func (pctx *parserCtx) checkCondSectionEntityBoundary(ctx context.Context, sectionDepth int) error {
	if !pctx.options.IsSet(parseDTDValid) {
		return nil
	}
	if pctx.inputTab.Len() <= sectionDepth {
		return nil
	}
	return pctx.error(ctx, errCondSectionBoundary)
}

// errCondSectionBoundary reports a conditional section whose "<![", "[" and
// "]]>" are not all in the same entity (libxml2 XML_ERR_ENTITY_BOUNDARY, "All
// markup of the conditional section is not in the same entity").
var errCondSectionBoundary = fmt.Errorf("%w: all markup of the conditional section is not in the same entity", ErrEntityBoundary)

// checkCondSectionClose enforces the closing side of "Proper Conditional
// Section/PE Nesting" (XML §3.4) at a section's "]]>". crossed reports that the
// "]]>" comes from a different input than the section's content: the content
// input ran out first (resumeCondSectionBelow), or a parameter entity
// referenced inside the body supplied the "]]>". Either way a parameter
// entity's replacement text holds only part of the section, so it cannot
// match extSubsetDecl (§2.8 WFC: PE Between Declarations). That is a fatal
// well-formedness error in both subsets, whether or not validating, matching
// libxml2 ("All markup of the conditional section is not in the same entity").
// The opening side (checkCondSectionEntityBoundary) is reported only when
// validating.
func (pctx *parserCtx) checkCondSectionClose(ctx context.Context, crossed bool) error {
	if !crossed {
		return nil
	}
	return pctx.error(ctx, errCondSectionBoundary)
}

// resumeCondSectionBelow handles a conditional section whose content input
// ran out before its "]]>": a parameter entity opened the section and the
// rest of it, if any, follows the reference in the input below. Every path
// ends in a fatal error; they differ only in which error and where.
//
// In the internal subset it fails at once: ErrEntityBoundary when the input
// below continues with the "]]>", and ErrConditionalSectionNotFinished
// otherwise.
//
// In the external subset it returns the depth of the input below, never past
// the external subset's own cursor (dtdInputFloor): reading on into the
// document would let markup after the DOCTYPE close the section. The section
// keeps reading there so that, like libxml2, a "]]>" is reported as
// ErrEntityBoundary by checkCondSectionClose at that "]]>", and a section
// that no "]]>" closes before the external subset ends is reported as
// ErrConditionalSectionNotFinished.
func (pctx *parserCtx) resumeCondSectionBelow(ctx context.Context) (int, error) {
	if !pctx.external {
		cur := pctx.getCursor()
		if cur == nil {
			return 0, ErrConditionalSectionNotFinished
		}
		if _, err := pctx.skipBlankRun(ctx, cur); err != nil {
			return 0, err
		}
		if cur.Peek() == ']' && cur.PeekAt(1) == ']' && cur.PeekAt(2) == '>' {
			return 0, pctx.error(ctx, errCondSectionBoundary)
		}
		return 0, ErrConditionalSectionNotFinished
	}

	for pctx.inputTab.Len() >= pctx.dtdInputFloor {
		top := pctx.adaptCursor(pctx.inputTab.PeekOne())
		if top != nil && !top.Done() {
			return pctx.inputTab.Len(), nil
		}
		if pctx.inputTab.Len() == pctx.dtdInputFloor {
			break
		}
		pctx.popInput()
	}
	return 0, ErrConditionalSectionNotFinished
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
// cursor is gone/Done), signalling the caller to stop or resume its scan.
//
// Every conditional-section error is fatal. The external subset is read fully
// into a buffer before parsing, so an unterminated "]]>"
// (ErrConditionalSectionNotFinished) at buffer EOF is a genuine truncation, not
// a streaming boundary, and a malformed/miscased INCLUDE/IGNORE keyword
// (ErrConditionalSectionKeyword) is a well-formedness error (XML §3.4 P62/P63:
// the keyword is case-sensitive and mandatory). Both propagate — libxml2 reports
// the same "Content error in the external subset" (W3C not-wf-not-sa-004,
// ibm-not-wf-P62-ibm62n07).
//
// Unlike skipBlanks, the blank skip here advances over whitespace ONLY and
// leaves any "%" for the explicit parsePEReference below. In the external
// subset, skipBlanks calls handlePEReference, which CONSUMES a "%pe;" reference
// while only validating it (it does not expand the replacement text). That
// swallows the reference before parsePEReference can push the PE content onto
// the input stack, so the PE's declarations are never parsed.
func (pctx *parserCtx) parseExternalSubsetDeclStep(ctx context.Context, baseLen int) (bool, error) {
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
	// parsePEReference. Do NOT route through skipBlanks here (it would consume a
	// "%pe;" reference without expanding it). skipBlankRun only advances over
	// whitespace, so it is safe here and caps an oversized blank run with
	// ErrNodeContentTooLarge instead of buffering it unbounded.
	if c := pctx.getCursor(); c != nil {
		if _, err := pctx.skipBlankRun(ctx, c); err != nil {
			return false, err
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
	// When we are back at the floor, inspect the floor cursor DIRECTLY rather
	// than via getCursor(): if the floor content cursor is exhausted,
	// getCursor() would auto-pop it and return the cursor BELOW the floor (e.g.
	// the main document cursor for the top-level subset), which is not Done.
	// The step would then parse post-DOCTYPE DOCUMENT markup as if it were
	// external-subset markup — dropping a post-DOCTYPE comment/PI from the
	// document. Stop here instead, the same "don't auto-pop past the floor"
	// principle the INCLUDE loop already applies.
	if pctx.inputTab.Len() == baseLen {
		floor := pctx.adaptCursor(pctx.inputTab.PeekOne())
		if floor == nil || floor.Done() {
			return true, nil
		}
	}
	cur := pctx.getCursor()
	if cur == nil {
		return true, nil
	}

	cur = pctx.getCursor()
	if cur != nil && cur.Peek() == '<' && cur.PeekAt(1) == '!' && cur.PeekAt(2) == '[' {
		// Nested conditional section. parseConditionalSections is responsible for
		// its own blank/PE handling within the section. Every conditional-section
		// error propagates as fatal — an unterminated "]]>"
		// (ErrConditionalSectionNotFinished) at the end of the fully-buffered
		// external subset is a genuine truncation, not a streaming boundary
		// (XML §3.4; libxml2 "Content error in the external subset"), and a
		// malformed/miscased keyword (ErrConditionalSectionKeyword) is a
		// well-formedness error (P62/P63).
		if err := pctx.parseConditionalSections(ctx); err != nil {
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
		if err := pctx.parsePEReference(ctx, false); err != nil {
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

// parsePEReference expands the parameter-entity reference at the cursor by
// pushing its replacement text onto the input stack. When pad is true the pushed
// replacement is enlarged by one leading and one trailing space per XML §4.4.8
// ("Included as PE") — required when a PE is included INSIDE or ADJACENT to a
// markup declaration in the external subset so the boundary separates tokens.
// The between-declaration callers pass pad=false: a PE that stands on its own
// between declarations is already whitespace-separated, so padding is redundant.
func (pctx *parserCtx) parsePEReference(ctx context.Context, pad bool) error {
	cur := pctx.getCursor()
	if cur == nil {
		return pctx.error(ctx, errNoCursor)
	}
	if cur.Peek() != '%' {
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
		} else if etype == enum.ExternalParameterEntity {
			// External parameter entity: the replacement text is the content of
			// the referenced external resource, not inline text. Load it from
			// the resolver (gated by the XXE secure default) and push the RAW
			// bytes so the surrounding DTD declaration loop parses them — the
			// same mechanism the external subset body and an internal PE use: the
			// loaded resource is a DTD fragment whose own references are resolved
			// lexically during declaration parsing. When external loading is
			// disabled the load returns empty content and behavior is unchanged
			// (no input pushed).
			ent, ok := entity.(*Entity)
			if !ok {
				return pctx.error(ctx, errors.New("internal: external parameter entity is not *helium.Entity"))
			}
			// Reject a self/mutually recursive external PE BEFORE loading or
			// pushing: while a PE's replacement text is being parsed its input is
			// still on the stack (parameterEntityActive), so a nested "%pe;" to the
			// same entity would otherwise keep pushing cursors until the
			// amplification ceiling trips. A counter check fails closed and reports
			// a parse error instead.
			if pctx.parameterEntityActive(ent) {
				return pctx.error(ctx, fmt.Errorf("parse error: external parameter entity %%%s; references itself", name))
			}
			// loadExternalParameterEntityContent already strips and decodes any
			// leading TextDecl ("<?xml ... encoding=...?>") at the shared
			// load/cache chokepoint, so the bytes returned here are the
			// post-TextDecl replacement text ready for the declaration loop — the
			// "<?xml" is never seen as a stray PI, and the entity-value expansion
			// path sees the identical decoded bytes.
			content, peURI, peVersion, err := pctx.loadExternalParameterEntityContent(ctx, ent)
			if err != nil {
				return err
			}
			// Charge the loaded bytes (plus the per-reference fixed cost) against
			// the amplification guard so a small DTD cannot reference a large
			// external PE repeatedly to drive unbounded expansion.
			if err := pctx.entityCheck(entity, len(content)); err != nil {
				return pctx.error(ctx, err)
			}
			if len(content) > 0 {
				// Scope baseURI to the PE's OWN resolved URI while its replacement
				// text is parsed, so a relative system ID in a declaration INSIDE
				// the PE (e.g. <!ENTITY e SYSTEM "leaf.ent"> in sub/pe.ent)
				// resolves against the PE's location, not the containing DTD. The
				// override (and the active-recursion mark) is cleared when this
				// pushed cursor is popped.
				pctx.pushExternalPEInput(strcursor.NewByteCursor(bytes.NewReader(padPEContent(content, pad))), peURI, peVersion, ent)
			}
			pctx.hasPERefs = true
			pctx.hasExternalPERef = true
			return nil
		} else {
			// Internal parameter entity: push its replacement text verbatim as a
			// new input, as libxml2 xmlParsePEReference does, so the surrounding
			// declaration loop parses it as markup in place. The stored value is
			// already the replacement text (character references and the PE
			// references of the literal were resolved when it was declared;
			// general references are bypassed, XML §4.5), so it must not be
			// decoded again: a "% " declaration marker, a '%' or '&' that came
			// from a character reference, or a "%name;" naming a PE the text
			// itself declares are all ordinary markup to the parser here.
			ent, _ := entity.(*Entity)
			// A PE whose replacement text references itself (directly or through
			// another PE) would push inputs without end; reject it while its
			// earlier input is still on the stack (WFC: No Recursion).
			if ent != nil && pctx.parameterEntityActive(ent) {
				return pctx.error(ctx, fmt.Errorf("%w: parameter entity %%%s; references itself", errEntityLoop, name))
			}
			content := entity.Content()
			// Charge this PE's OWN replacement bytes before pushing it as new
			// input. Without this the PE's direct contribution is free, so a
			// small DTD that references a large PE many times could drive
			// unbounded expansion past the amplification limit. A reference
			// inside the text is charged by its own parsePEReference call when
			// the parser reaches it.
			if err := pctx.entityCheck(entity, len(content)); err != nil {
				return pctx.error(ctx, err)
			}

			pctx.pushInternalPEInput(strcursor.NewByteCursor(bytes.NewReader(padPEContent(content, pad))), ent)
		}
	}
	pctx.hasPERefs = true
	return nil
}

// padPEContent returns the parameter-entity replacement text enlarged by one
// leading and one trailing space (#x20) per XML §4.4.8 when pad is true,
// otherwise the content unchanged. A fresh slice is always returned when padding
// so the caller's buffer is never aliased.
func padPEContent(content []byte, pad bool) []byte {
	if !pad {
		return content
	}
	out := make([]byte, 0, len(content)+2)
	out = append(out, ' ')
	out = append(out, content...)
	out = append(out, ' ')
	return out
}

// loadExternalParameterEntityContent returns the replacement text, resolved URI,
// and effective TextDecl version of an external parameter entity. It loads and
// caches all three on the entity at first use after validating literal characters
// against that effective version. External loading honors the parser's
// secure-default gating: when XXE loading is disabled (parseNoXXE) or the
// resolver declines to open the resource, nothing is loaded and empty content is
// returned, leaving the caller's behavior unchanged. The read is byte-capped
// (externalEntityMaxBytes) and the opened input is closed as soon as the bounded
// read completes, mirroring parseExternalEntityPrivate.
func (pctx *parserCtx) loadExternalParameterEntityContent(ctx context.Context, e *Entity) ([]byte, string, string, error) {
	if len(e.content) > 0 {
		// Return the URI the bytes were ORIGINALLY loaded from (cached on first
		// load), not e.URI(): the first load may have used a catalog/custom-resolver
		// URI, and relative system IDs inside the cached PE must resolve against
		// that same base regardless of which reference triggered the load first.
		return []byte(e.content), e.resolvedURI, e.textDeclVersion, nil
	}
	if pctx.options.IsSet(parseNoXXE) {
		return nil, "", "", nil
	}

	var input sax.ParseInput
	if s := pctx.sax; s != nil {
		// Gate the confined-FS retry (openExternalResource) for this PE on its
		// DECLARED system id's original relativeness — e.URI() is already resolved
		// to an absolute name and cannot be distinguished. Clear it after so a
		// later ResolveEntity cannot inherit a stale eligibility.
		pctx.extRefRelative = systemIDRetryEligible(e.systemID)
		resolved, err := s.ResolveEntity(ctx, e.externalID, e.URI())
		pctx.extRefRelative = false
		switch err {
		case nil:
			input = resolved
		case sax.ErrHandlerUnspecified:
		default:
			return nil, "", "", pctx.error(ctx, err)
		}
	}
	if input == nil {
		return nil, "", "", nil
	}

	// The resolved input carries the URI actually opened (a catalog-resolved URI
	// or the entity's resolved system URI), which is the correct base for
	// relative system IDs in declarations inside this PE. Fall back to the
	// entity's declared URI when the resolver did not supply one.
	uri := input.URI()
	if uri == "" {
		uri = e.URI()
	}

	// Read through a bounded reader so an unbounded source cannot exhaust memory,
	// and close the input immediately at the read boundary (not deferred) so an
	// underlying OS resource is never held open for the lifetime of the parse.
	content, exceeded, err := iolimit.ReadAll(input, externalEntityMaxBytes)
	if c, ok := input.(io.Closer); ok {
		_ = c.Close()
	}
	if err != nil {
		return nil, "", "", pctx.error(ctx, fmt.Errorf("reading external parameter entity: %w", err))
	}
	if exceeded {
		return nil, "", "", pctx.error(ctx, fmt.Errorf("external parameter entity (URI=%s) exceeds maximum size of %d bytes", e.URI(), externalEntityMaxBytes))
	}

	// Strip and decode any leading TextDecl ("<?xml ... encoding=...?>") HERE, at
	// the single shared load/cache chokepoint, so EVERY consumer of an external
	// PE's replacement text — the top-level "%pe;" declaration loop AND the
	// entity-value expansion path (parameterEntityReplacement →
	// decodeEntitiesInternal / expandEntityValueForRefCheck) — sees the same
	// post-TextDecl bytes regardless of reference order. Caching the decoded
	// bytes means a later reference (from either path) reuses them consistently,
	// instead of one path getting raw bytes that embed the TextDecl into a
	// general entity's stored value.
	content, textDeclVersion, err := pctx.decodeExternalPEContentVersion(ctx, uri, content)
	if err != nil {
		return nil, "", "", err
	}
	// This PE can be expanded inside another EntityValue, where its decoded body
	// becomes stored replacement text instead of being reparsed by the DTD loop.
	// Validate raw literal characters before caching so that path enforces the
	// PE's effective TextDecl version too. Character-reference syntax stays ASCII
	// here and is validated later by the normal reference parser.
	validator := parserCtx{version: textDeclVersion}
	if !validator.literalBytesValid(content) {
		return nil, "", "", pctx.error(ctx, ErrInvalidChar)
	}

	e.content = string(content)
	e.resolvedURI = uri
	e.textDeclVersion = textDeclVersion
	return content, uri, textDeclVersion, nil
}

// decodeExternalPEContentVersion returns decoded external content and its
// effective XML version. A TextDecl version overrides the referencing
// document's version after compatibility is checked. A leading UTF-8
// byte-order mark is not part of the replacement text, so it is dropped, and a
// TextDecl may follow it (XML §4.3.3).
func (pctx *parserCtx) decodeExternalPEContentVersion(ctx context.Context, srcURI string, content []byte) ([]byte, string, error) {
	entityVersion := pctx.documentVersion()
	if len(content) == 0 {
		return content, entityVersion, nil
	}

	// UTF-16 / UCS-4 / EBCDIC external content is not ASCII-compatible: the body,
	// a leading byte-order mark, and any leading TextDecl are all encoded, so the
	// byte-level "<?xml" scan below cannot see the TextDecl. Detect the encoding
	// from the BOM or the encoded leading '<' and decode on a rune cursor instead.
	if nonASCIIExternalEncoding(content) != "" {
		return pctx.decodeNonASCIIExternalContent(ctx, srcURI, content)
	}

	body := bytes.TrimPrefix(content, patUTF8)
	if !looksLikeXMLDecl(strcursor.NewByteCursor(bytes.NewReader(body))) {
		return body, entityVersion, nil
	}

	// Parse the TextDecl on a throwaway context over a COPY of the bytes, so the
	// declared encoding switches only this sub-cursor. Inherit the parent's
	// options (encoding-ignore policy). srcURI (the resolved DTD/PE location) is
	// set as the sub-parser's base URI so a malformed TextDecl reports the source
	// file, matching every other declaration error in that resource. The TextDecl
	// grammar is enforced by parseTextDecl: VersionInfo is optional, EncodingDecl
	// is REQUIRED, and no StandaloneDecl is permitted — a version-only or
	// standalone-bearing declaration is rejected, and never leniently accepted.
	sub := &parserCtx{}
	if err := sub.init(nil, bytes.NewReader(content)); err != nil {
		return nil, "", err
	}
	defer func() { _ = sub.release() }()
	sub.options = pctx.options
	sub.baseURI = srcURI
	// Seed the parent document's XML version so the TextDecl's version-compatibility
	// check (checkEntityVersion) compares against the real document version rather
	// than this doc-less sub-context's default.
	sub.version = pctx.documentVersion()

	// detectEncoding consumes a leading UTF-8 byte-order mark and records it in
	// autoEncoding, which checkBOMEncodingConflict compares with the TextDecl's
	// encoding below.
	if _, err := sub.detectEncoding(); err != nil {
		return nil, "", sub.error(ctx, err)
	}
	if err := sub.parseTextDecl(ctx); err != nil {
		return nil, "", err
	}
	// A TextDecl naming an encoding other than UTF-8 after a UTF-8 byte-order
	// mark is fatal (XML §4.3.3), as it is for the document entity. libxml2
	// rejects a UTF-16 name there but accepts any other name and decodes the
	// body in it.
	if err := sub.checkBOMEncodingConflict(); err != nil {
		return nil, "", sub.error(ctx, err)
	}
	if err := sub.switchEncoding(); err != nil {
		// Wrap through sub.error so the failure (e.g. an unsupported declared
		// encoding) carries srcURI, matching the parseTextDecl branch; otherwise
		// it would be rewrapped at the caller's location and lose the source file.
		return nil, "", sub.error(ctx, err)
	}

	cur := sub.getCursor()
	if cur == nil {
		return nil, sub.version, nil
	}
	rest, err := io.ReadAll(cur.Unused())
	if err != nil {
		return nil, "", sub.error(ctx, err)
	}
	return rest, sub.version, nil
}

// decodeNonASCIIExternalContent decodes an external resource's replacement text
// that is in an encoding which is not ASCII-compatible, returning the body as
// UTF-8. For UTF-16 / UCS-4 the encoding is externally known — fixed by the
// byte-order mark or the encoded shape of the leading '<' — so an OPTIONAL
// leading TextDecl need not (re)declare it and a resource with no TextDecl at
// all (e.g. a UTF-16 external DTD subset that opens on a comment) is still
// decoded. EBCDIC is recognized only by an encoded leading '<?xm'; its code
// page is the name in that TextDecl, read through the EBCDIC invariant
// characters as for the document entity (parseDocument), defaulting to IBM-037.
// The TextDecl is consumed on the decoded rune cursor after switchEncoding,
// enforcing the same grammar as the ASCII path (VersionInfo OPTIONAL,
// EncodingDecl REQUIRED, NO StandaloneDecl). srcURI scopes any error to the
// source resource.
func (pctx *parserCtx) decodeNonASCIIExternalContent(ctx context.Context, srcURI string, content []byte) ([]byte, string, error) {
	sub := &parserCtx{}
	if err := sub.init(nil, bytes.NewReader(content)); err != nil {
		return nil, "", err
	}
	defer func() { _ = sub.release() }()
	sub.options = pctx.options
	sub.baseURI = srcURI
	// Seed the parent document's XML version so the TextDecl's version-compatibility
	// check (checkEntityVersion) compares against the real document version rather
	// than this doc-less sub-context's default.
	sub.version = pctx.documentVersion()

	// Detect the encoding (consuming a 2-byte BOM; peeking a BOM-less 4-byte
	// pattern) and switch the sub-cursor to a UTF-8-decoding rune cursor before
	// reading the TextDecl or the body.
	enc, err := sub.detectEncoding()
	if err != nil {
		return nil, "", sub.error(ctx, err)
	}
	sub.detectedEncoding = enc
	if enc == encEBCDIC {
		sub.encoding = encoding.ExtractEBCDICEncoding(content)
		if sub.encoding == "" {
			sub.encoding = "ibm037"
		}
	}
	if err := sub.switchEncoding(); err != nil {
		return nil, "", sub.error(ctx, err)
	}

	cur := sub.getCursor()
	if cur == nil {
		return nil, sub.version, nil
	}

	// Consume an optional leading TextDecl on the decoded rune cursor. The
	// encoding was already fixed by the BOM/leading-'<' shape or the EBCDIC
	// name, so the declared encoding drives no further switch; a standalone
	// pseudo-attribute or a missing encoding is still rejected by
	// parseTextDeclFromCursor.
	//
	// XML §4.3.3 applies per external parsed entity, so a UTF-16 BOM here that
	// contradicts this TextDecl's declared encoding is also a fatal error.
	// checkBOMEncodingConflict does not run on this path: a declared name that
	// contradicts a UTF-16 BOM is ignored.
	if looksLikeXMLDeclString(cur) {
		if err := sub.parseTextDeclFromCursor(ctx); err != nil {
			return nil, "", err
		}
		cur = sub.getCursor()
		if cur == nil {
			return nil, sub.version, nil
		}
	}

	rest, err := io.ReadAll(cur.Unused())
	if err != nil {
		return nil, "", sub.error(ctx, err)
	}
	return rest, sub.version, nil
}
