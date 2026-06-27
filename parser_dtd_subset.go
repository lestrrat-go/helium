package helium

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/lestrrat-go/helium/enum"
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
	u, eid, err := pctx.parseExternalID(ctx, true)
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
		if err := ctx.Err(); err != nil {
			return err
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
	cur := pctx.getCursor()
	if cur == nil {
		return ErrPrematureEOF
	}

	if err := cur.Advance(3); err != nil {
		return err
	}

	// skipBlanks records an over-cap whitespace run in pctx.blankRunErr but only
	// returns a bool, so a guard tripped while skipping conditional-section HEADER
	// whitespace (after "<![", after a "%pe;", after INCLUDE/IGNORE) must be
	// surfaced here. Otherwise this function would proceed and return a generic
	// conditional-section sentinel (ErrConditionalSectionKeyword /
	// ErrConditionalSectionNotFinished) which the top-level external-subset loop
	// TOLERATES — downgrading a resource-limit violation to "stop parsing the
	// subset" instead of failing closed at the source.
	pctx.skipBlanks(ctx)
	if pctx.blankRunErr != nil {
		return pctx.blankRunErr
	}

	cur = pctx.getCursor()
	if cur != nil && cur.Peek() == '%' {
		if err := pctx.parsePEReference(ctx); err != nil {
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

				// Bounded blank skip (NOT skipBlanks, which would consume a
				// "%pe;" reference without expanding it). skipBlankRun only
				// advances over whitespace, so it is safe here and caps an
				// oversized blank run inside the section with
				// ErrNodeContentTooLarge.
				if _, err := pctx.skipBlankRun(ctx, sec); err != nil {
					return err
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
		if pctx.blankRunErr != nil {
			return pctx.blankRunErr
		}
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
// behavior: a conditional-section WRAPPER error (an unterminated "]]>" or a
// missing/malformed INCLUDE/IGNORE keyword — ErrConditionalSectionNotFinished
// or ErrConditionalSectionKeyword) stops the loop WITHOUT failing the whole
// parse, which valid documents whose conditional-section handling is otherwise
// imperfect rely on. Actual declaration parse errors from inside an INCLUDE
// body still propagate even when tolerateCondError is set. The INCLUDE-body
// caller passes false so every nested conditional-section error propagates.
//
// Unlike skipBlanks, the blank skip here advances over whitespace ONLY and
// leaves any "%" for the explicit parsePEReference below. In the external
// subset, skipBlanks calls handlePEReference, which CONSUMES a "%pe;" reference
// while only validating it (it does not expand the replacement text). That
// swallows the reference before parsePEReference can push the PE content onto
// the input stack, so the PE's declarations are never parsed.
func (pctx *parserCtx) parseExternalSubsetDeclStep(ctx context.Context, baseLen int, tolerateCondError bool) (bool, error) {
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
		// its own blank/PE handling within the section.
		if err := pctx.parseConditionalSections(ctx); err != nil {
			// Only the conditional-section WRAPPER sentinels (an unterminated
			// "]]>" or a missing/malformed INCLUDE/IGNORE keyword) are tolerated
			// at the top level: those mirror the long-standing best-effort
			// handling of an imperfectly-terminated section. An actual
			// declaration parse error from within an INCLUDE body (e.g. a
			// malformed "<!BOGUS" or a bad entity-value PE) must propagate.
			if tolerateCondError && (errors.Is(err, ErrConditionalSectionNotFinished) || errors.Is(err, ErrConditionalSectionKeyword)) {
				// A resource-limit violation (over-cap whitespace) recorded while
				// the conditional section was being parsed must NEVER be masked by
				// the conditional-section tolerance: propagate it as a real fatal
				// error instead of stopping the loop silently.
				if pctx.blankRunErr != nil {
					return false, pctx.blankRunErr
				}
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
			// same mechanism the external subset body uses. This must NOT go
			// through decodeEntities like an internal PE: the loaded resource is
			// a DTD fragment whose own references are resolved lexically during
			// declaration parsing. When external loading is disabled the load
			// returns empty content and behavior is unchanged (no input pushed).
			ent, ok := entity.(*Entity)
			if !ok {
				return pctx.error(ctx, errors.New("internal: external parameter entity is not *helium.Entity"))
			}
			// Reject a self/mutually recursive external PE BEFORE loading or
			// pushing: while a PE's replacement text is being parsed its input is
			// still on the stack (externalPEActive), so a nested "%pe;" to the same
			// entity would otherwise keep pushing cursors until the amplification
			// ceiling trips. A counter check fails closed and reports a parse error
			// instead. Internal PEs are guarded separately by the decode-depth cap.
			if pctx.externalPEActive(ent) {
				return pctx.error(ctx, fmt.Errorf("parse error: external parameter entity %%%s; references itself", name))
			}
			// loadExternalParameterEntityContent already strips and decodes any
			// leading TextDecl ("<?xml ... encoding=...?>") at the shared
			// load/cache chokepoint, so the bytes returned here are the
			// post-TextDecl replacement text ready for the declaration loop — the
			// "<?xml" is never seen as a stray PI, and the entity-value expansion
			// path sees the identical decoded bytes.
			content, peURI, err := pctx.loadExternalParameterEntityContent(ctx, ent)
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
				pctx.pushExternalPEInput(strcursor.NewByteCursor(bytes.NewReader(content)), peURI, ent)
			}
			pctx.hasPERefs = true
			return nil
		} else {
			// Capture the PE's replacement text once: Entity.Content()
			// allocates a fresh []byte copy on every call, so we reuse this
			// local for both decoding and the amplification accounting below.
			content := entity.Content()

			decodedContent, err := pctx.decodeEntities(ctx, content, SubstituteBoth)
			if err != nil {
				return fmt.Errorf("failed to decode parameter entity content: %v", err)
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

// loadExternalParameterEntityContent returns the replacement text of an external
// parameter entity, loading it from the resolved external resource on first use
// and caching it on the entity for subsequent references. External loading
// honors the parser's secure-default gating: when XXE loading is disabled
// (parseNoXXE) or the resolver declines to open the resource, nothing is loaded
// and empty content is returned, leaving the caller's behavior unchanged. The
// read is byte-capped (externalEntityMaxBytes) and the opened input is closed as
// soon as the bounded read completes, mirroring parseExternalEntityPrivate.
func (pctx *parserCtx) loadExternalParameterEntityContent(ctx context.Context, e *Entity) ([]byte, string, error) {
	if len(e.content) > 0 {
		// Return the URI the bytes were ORIGINALLY loaded from (cached on first
		// load), not e.URI(): the first load may have used a catalog/custom-resolver
		// URI, and relative system IDs inside the cached PE must resolve against
		// that same base regardless of which reference triggered the load first.
		return []byte(e.content), e.resolvedURI, nil
	}
	if pctx.options.IsSet(parseNoXXE) {
		return nil, "", nil
	}

	var input sax.ParseInput
	if s := pctx.sax; s != nil {
		resolved, err := s.ResolveEntity(ctx, e.externalID, e.URI())
		switch err {
		case nil:
			input = resolved
		case sax.ErrHandlerUnspecified:
		default:
			return nil, "", pctx.error(ctx, err)
		}
	}
	if input == nil {
		return nil, "", nil
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
		return nil, "", pctx.error(ctx, fmt.Errorf("reading external parameter entity: %w", err))
	}
	if exceeded {
		return nil, "", pctx.error(ctx, fmt.Errorf("external parameter entity (URI=%s) exceeds maximum size of %d bytes", e.URI(), externalEntityMaxBytes))
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
	content, err = pctx.decodeExternalPEContent(ctx, content)
	if err != nil {
		return nil, "", err
	}

	e.content = string(content)
	e.resolvedURI = uri
	return content, uri, nil
}

// decodeExternalPEContent consumes an OPTIONAL TextDecl at the start of an
// external parameter entity's replacement text and returns the post-TextDecl
// bytes decoded to UTF-8. An external parsed entity may begin with
// "<?xml version=... encoding=...?>"; pushed raw, the DTD declaration loop would
// reject the "<?xml" as a processing instruction (a PI target may not be "xml"),
// so the TextDecl must be stripped here and any declared encoding honored — the
// same treatment parseExternalEntityPrivate gives an external general entity.
// When no TextDecl is present the content is returned unchanged. Only the
// ASCII-compatible TextDecl shape is handled (the DTD-fragment case in scope);
// a non-ASCII-leading encoding is left to the caller's raw push.
func (pctx *parserCtx) decodeExternalPEContent(ctx context.Context, content []byte) ([]byte, error) {
	if len(content) == 0 {
		return content, nil
	}
	if !looksLikeXMLDecl(strcursor.NewByteCursor(bytes.NewReader(content))) {
		return content, nil
	}

	// Parse the TextDecl on a throwaway context over a COPY of the bytes, so the
	// declared encoding switches only this sub-cursor. Inherit the parent's
	// options (encoding-ignore policy). The TextDecl grammar is enforced by
	// parseTextDecl: VersionInfo is optional, EncodingDecl is REQUIRED, and no
	// StandaloneDecl is permitted — a version-only or standalone-bearing
	// declaration is rejected rather than leniently accepted.
	sub := &parserCtx{}
	if err := sub.init(nil, bytes.NewReader(content)); err != nil {
		return nil, err
	}
	defer func() { _ = sub.release() }()
	sub.options = pctx.options

	if bcur := sub.getByteCursor(); bcur != nil && looksLikeXMLDecl(bcur) {
		if err := sub.parseTextDecl(ctx); err != nil {
			return nil, err
		}
	}
	if err := sub.switchEncoding(); err != nil {
		return nil, err
	}

	cur := sub.getCursor()
	if cur == nil {
		return nil, nil
	}
	rest, err := io.ReadAll(cur.Unused())
	if err != nil {
		return nil, err
	}
	return rest, nil
}
