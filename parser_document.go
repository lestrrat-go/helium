package helium

import (
	"bytes"
	"context"
	"errors"

	"github.com/lestrrat-go/helium/internal/encoding"
	"github.com/lestrrat-go/helium/internal/strcursor"
	"github.com/lestrrat-go/helium/sax"
	"github.com/lestrrat-go/pdebug"
)

func (pctx *parserCtx) parseDocument(ctx context.Context) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseDocument")
		defer g.IRelease("END parseDocument")
	}

	// Store pctx in the context so SAX callbacks (e.g. TreeBuilder) can
	// retrieve it via getParserCtx. Also store the document locator and
	// stop function so helium.StopParser works.
	ctx = withParserCtx(ctx, pctx)
	ctx = sax.WithDocumentLocator(ctx, pctx)
	ctx = context.WithValue(ctx, stopFuncKey{}, pctx.stop)

	if s := pctx.sax; s != nil {
		switch err := s.SetDocumentLocator(ctx, pctx); err {
		case nil, sax.ErrHandlerUnspecified:
			// no op
		default:
			return pctx.error(ctx, err)
		}
	}

	// see if we can find the preliminary encoding
	if pctx.encoding == "" {
		if enc, err := pctx.detectEncoding(); err == nil {
			pctx.detectedEncoding = enc
		}
	}

	// At this stage we MUST be using a ByteCursor, as we
	// don't know what the encoding is.
	bcur := pctx.getByteCursor()
	if bcur == nil {
		return pctx.error(ctx, ErrByteCursorRequired)
	}

	// nothing left? eek
	if bcur.Done() {
		return pctx.error(ctx, errors.New("empty document"))
	}

	// For UTF-16 detected encodings, we must switch encoding FIRST
	// because the XML declaration is encoded in UTF-16, not ASCII.
	switch pctx.detectedEncoding {
	case encUTF16LE, encUTF16BE:
		// For UTF-16 detected encodings, we must switch encoding FIRST
		// because the XML declaration is encoded in UTF-16, not ASCII.
		if err := pctx.switchEncoding(); err != nil {
			return pctx.error(ctx, err)
		}
		cur := pctx.getCursor()
		if cur != nil && cur.HasPrefixString("<?xml") {
			if err := pctx.parseXMLDeclFromCursor(ctx); err != nil {
				return pctx.error(ctx, err)
			}
		}
	case encEBCDIC:
		// EBCDIC bytes are not ASCII-compatible, so we cannot parse the
		// XML declaration at byte level. Instead, scan the raw bytes
		// using the EBCDIC invariant character set (shared across all
		// EBCDIC Latin code pages) to extract the encoding name.
		if pctx.rawInput != nil {
			if encName := encoding.ExtractEBCDICEncoding(pctx.rawInput); encName != "" {
				pctx.encoding = encName
			}
		}
		// Fall back to IBM-037 (US EBCDIC) if no encoding was declared.
		if pctx.encoding == "" {
			pctx.encoding = "ibm037"
		}
		// Reset the byte cursor from the raw input so the decoder
		// reads from the beginning of the document.
		pctx.popInput()
		pctx.pushInput(strcursor.NewByteCursor(bytes.NewReader(pctx.rawInput)))
		if err := pctx.switchEncoding(); err != nil {
			return pctx.error(ctx, err)
		}
		// Parse the XML declaration from the decoded rune cursor.
		cur := pctx.getCursor()
		if cur != nil && looksLikeXMLDeclString(cur) {
			if err := pctx.parseXMLDeclFromCursor(ctx); err != nil {
				return pctx.error(ctx, err)
			}
		}
	default:
		// XML prolog (byte-level for ASCII-compatible encodings)
		if looksLikeXMLDecl(bcur) {
			if err := pctx.parseXMLDecl(ctx); err != nil {
				return pctx.error(ctx, err)
			}
		}

		// At this point we know the encoding, so switch the encoding
		// of the source.
		if err := pctx.switchEncoding(); err != nil {
			return pctx.error(ctx, err)
		}
	}

	if pctx.treeBuilder != nil {
		pctx.fastStartDocument()
	} else if s := pctx.sax; s != nil {
		switch err := s.StartDocument(ctx); err {
		case nil, sax.ErrHandlerUnspecified:
			// no op
		default:
			return pctx.error(ctx, err)
		}
	}

	if pctx.stopped {
		return errParserStopped
	}

	// Misc part of the prolog
	if err := pctx.parseMisc(ctx); err != nil {
		return pctx.error(ctx, err)
	}

	cur := pctx.getCursor()
	if cur == nil {
		return pctx.error(ctx, errNoCursor)
	}
	// Doctype declarations and more misc
	if cur.HasPrefixString("<!DOCTYPE") {
		pctx.inSubset = inInternalSubset
		if err := pctx.parseDocTypeDecl(ctx); err != nil {
			return pctx.error(ctx, err)
		}

		if cur.HasPrefixString("[") {
			pctx.instate = psDTD
			if err := pctx.parseInternalSubset(ctx); err != nil {
				return pctx.error(ctx, err)
			}
		}

		// Query SAX callbacks for subset/standalone status.
		// These mirror libxml2's calls after internal subset parsing.
		if s := pctx.sax; s != nil {
			if has, err := s.HasInternalSubset(ctx); err == nil {
				_ = has // informational; handler may use for validation decisions
			}
			if has, err := s.HasExternalSubset(ctx); err == nil {
				_ = has
			}
		}

		pctx.inSubset = inExternalSubset
		if s := pctx.sax; s != nil {
			switch err := s.ExternalSubset(ctx, pctx.intSubName, pctx.extSubSystem, pctx.extSubURI); err {
			case nil, sax.ErrHandlerUnspecified:
				// no op
			default:
				return pctx.error(ctx, err)
			}
		}
		if pctx.instate == psEOF {
			return pctx.error(ctx, errors.New("unexpected EOF"))
		}
		pctx.inSubset = notInSubset

		pctx.cleanSpecialAttributes()

		pctx.instate = psPrologue
		if err := pctx.parseMisc(ctx); err != nil {
			return pctx.error(ctx, err)
		}
	}
	pctx.skipBlanks(ctx)

	if cur.Peek() != '<' {
		return pctx.error(ctx, ErrEmptyDocument)
	}

	pctx.instate = psContent
	if err := pctx.parseElement(ctx); err != nil {
		return pctx.error(ctx, err)
	}

	if pctx.stopped {
		return errParserStopped
	}

	pctx.instate = psEpilogue

	if err := pctx.parseMisc(ctx); err != nil {
		return pctx.error(ctx, err)
	}
	if !cur.Done() {
		return pctx.error(ctx, ErrDocumentEnd)
	}
	pctx.instate = psEOF

	// All done
	if pctx.treeBuilder != nil {
		pctx.fastEndDocument()
	} else if s := pctx.sax; s != nil {
		switch err := s.EndDocument(ctx); err {
		case nil, sax.ErrHandlerUnspecified:
			// no op
		default:
			return pctx.error(ctx, err)
		}
	}

	return nil
}

func (pctx *parserCtx) parseContent(ctx context.Context) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseContent")
		defer g.IRelease("END parseContent")
	}
	pctx.instate = psContent

	cur := pctx.getCursor()
	if cur == nil {
		return pctx.error(ctx, errNoCursor)
	}

	doRecover := pctx.options.IsSet(parseRecover)

	for !cur.Done() && !pctx.stopped {
		if cur.Peek() == '<' && cur.PeekAt(1) == '/' {
			break
		}

		var err error
		switch cur.Peek() {
		case '<':
			switch cur.PeekAt(1) {
			case '?':
				err = pctx.parsePI(ctx)
			case '!':
				switch {
				case cur.PeekAt(2) == '[' &&
					cur.PeekAt(3) == 'C' &&
					cur.PeekAt(4) == 'D' &&
					cur.PeekAt(5) == 'A' &&
					cur.PeekAt(6) == 'T' &&
					cur.PeekAt(7) == 'A' &&
					cur.PeekAt(8) == '[':
					err = pctx.parseCDSect(ctx)
				case cur.PeekAt(2) == '-' && cur.PeekAt(3) == '-':
					err = pctx.parseComment(ctx)
				default:
					err = pctx.parseElement(ctx)
				}
			default:
				err = pctx.parseElement(ctx)
			}
		case '&':
			err = pctx.parseReference(ctx)
		default:
			if err := pctx.parseCharData(ctx, false); err != nil {
				if !doRecover || errors.Is(err, errParserStopped) {
					return err
				}
				if pctx.recoverErr == nil {
					pctx.recoverErr = err
				}
				pctx.disableSAX = true
				pctx.wellFormed = false
				pctx.skipToRecoverPoint()
			}
			continue
		}

		if err != nil {
			if !doRecover || errors.Is(err, errParserStopped) {
				return pctx.error(ctx, err)
			}
			if pctx.recoverErr == nil {
				pctx.recoverErr = err
			}
			pctx.disableSAX = true
			pctx.wellFormed = false

			prevLine, prevCol := cur.LineNumber(), cur.Column()
			pctx.skipToRecoverPoint()
			if !cur.Done() && cur.LineNumber() == prevLine && cur.Column() == prevCol {
				_ = cur.Advance(1)
			}
			continue
		}
	}

	if pctx.stopped {
		return errParserStopped
	}

	if pctx.recoverErr != nil {
		return pctx.recoverErr
	}

	return nil
}

// skipToRecoverPoint advances the cursor past unrecoverable content to the
// next '<' character or EOF, for re-synchronization in parseRecover mode.
func (ctx *parserCtx) skipToRecoverPoint() {
	cur := ctx.getCursor()
	if cur == nil {
		return
	}
	for !cur.Done() {
		if cur.Peek() == '<' {
			return
		}
		_ = cur.Advance(1)
	}
}
