package helium

import (
	"context"

	"github.com/lestrrat-go/helium/internal/strcursor"
)

// blankScanChunk bounds how many bytes a single blank-run scan peeks ahead
// before advancing the cursor. Peeking an ever-growing offset (the old
// behavior) forces the cursor to buffer the entire whitespace run up front, so
// an attacker-controlled infinite whitespace run in the prolog / inter-root
// position grows the cursor buffer without bound. Scanning in fixed-size chunks
// and advancing as we go keeps the cursor buffer bounded to this size.
const blankScanChunk = 4096

// blankScanner is the subset of a cursor that the bounded blank scan needs.
// Both strcursor.Cursor (returned by getCursor) and *strcursor.ByteCursor (used
// during XML-declaration parsing) satisfy it.
type blankScanner interface {
	PeekAt(int) byte
	Advance(int) error
	// HasByteAt reports whether a byte is present at the given offset, letting
	// the scan tell a real non-blank byte (PeekAt may report 0 for a NUL) from an
	// exhausted buffer where the scan ran out of input.
	HasByteAt(int) bool
	// Err exposes the cursor's sticky read/decode error so the scan can tell a
	// clean end-of-input apart from a read failure that also leaves PeekAt at 0 —
	// most importantly a push-stream Read returning context.Canceled when
	// cancellation unblocks a pending wait.
	Err() error
}

// blankRunLimit returns the maximum number of contiguous whitespace bytes a
// single blank-skip is allowed to consume before the run is treated as a
// memory-amplification DoS. It reuses the resolved node-content cap
// (pctx.maxNodeContent): a positive value caps the run, and 0 (the unlimited
// sentinel that resolveLimit produces from MaxNodeContentSize(-1)) disables the
// blank-run cap just as it disables the node-content cap. NewParser applies
// DefaultMaxNodeContentSize, so a blank run is bounded by default; only an
// explicit MaxNodeContentSize(-1) lifts it (trusted input only).
func (pctx *parserCtx) blankRunLimit() int {
	return pctx.maxNodeContent
}

// skipBlankRun advances cur past a run of XML whitespace in bounded chunks,
// checking the context between chunks and capping the total run length. It
// returns whether any whitespace was consumed and, when the run exceeds the
// blank-run limit, ErrNodeContentTooLarge. Memory stays bounded regardless of
// run length because it advances as it scans rather than peeking an
// ever-growing offset.
func (pctx *parserCtx) skipBlankRun(ctx context.Context, cur blankScanner) (bool, error) {
	limit := pctx.blankRunLimit()
	advanced := false
	total := 0
	for {
		if err := ctx.Err(); err != nil {
			return advanced, err
		}
		i := 0
		for i < blankScanChunk && isBlankByte(cur.PeekAt(i)) {
			i++
		}
		if i > 0 {
			total += i
			if limit > 0 && total > limit {
				return advanced, ErrNodeContentTooLarge
			}
			// Advance consumes the bytes just scanned. It cannot normally fail
			// here (the scan already buffered them via PeekAt), but a read error
			// that slips through is surfaced rather than swallowed so the parse
			// aborts instead of re-scanning the same bytes forever.
			if err := cur.Advance(i); err != nil {
				return advanced, err
			}
			advanced = true
		}
		// A full chunk may be followed by more blanks, so loop to scan again.
		if i == blankScanChunk {
			continue
		}
		// The scan stopped short of a full chunk: the run ends at the current
		// position. PeekAt reporting 0 there is ambiguous — a genuine non-blank
		// byte (possibly a real NUL) versus an exhausted buffer. When no byte is
		// present (HasByteAt is false) the scan ran out of input; if the cursor
		// also recorded a sticky read error, a read FAILED rather than the stream
		// ending cleanly — most importantly a push-stream Read returning
		// context.Canceled when cancellation unblocks a pending wait. Surface that
		// error (and any pending ctx error) so callers propagate cancellation
		// instead of synthesizing a syntax error ("blank needed after '<?xml'").
		// The HasByteAt guard is essential: a reader may return its final bytes
		// together with a non-EOF error, and those buffered bytes must still be
		// parsed before the error surfaces — the error is withheld while real
		// input remains at the scan position.
		if !cur.HasByteAt(0) {
			if err := cur.Err(); err != nil {
				return advanced, err
			}
			if err := ctx.Err(); err != nil {
				return advanced, err
			}
		}
		return advanced, nil
	}
}

func (pctx *parserCtx) skipBlanks(ctx context.Context) bool {
	// Once an over-cap whitespace run has tripped the guard the parse is in a
	// fatal state; stop consuming whitespace entirely so no caller (including
	// ones that do not inspect blankRunErr) can advance over an unbounded run.
	if pctx.blankRunErr != nil {
		return false
	}
	cur := pctx.getCursor()
	if cur == nil {
		return false
	}
	advanced, err := pctx.skipBlankRun(ctx, cur)
	if err != nil {
		pctx.blankRunErr = err
		return advanced
	}
	if advanced {
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
	if pctx.blankRunErr != nil {
		return false
	}
	advanced, err := pctx.skipBlankRun(ctx, cur)
	if err != nil {
		pctx.blankRunErr = err
		return advanced
	}
	if advanced {
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
