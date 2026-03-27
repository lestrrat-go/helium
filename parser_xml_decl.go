package helium

import (
	"context"
	"errors"
)

// should only be here if current buffer is at '<?xml'
func (pctx *parserCtx) parseXMLDecl(ctx context.Context) error {
	cur := pctx.getByteCursor()
	if cur == nil {
		return ErrByteCursorRequired
	}

	if !cur.Consume(xmlDeclHint) {
		return pctx.error(ctx, ErrInvalidXMLDecl)
	}

	if !pctx.skipBlankBytes(ctx, cur) {
		return errors.New("blank needed after '<?xml'")
	}

	if pctx.options.IsSet(parseLenientXMLDecl) {
		return pctx.parseXMLDeclLenient(ctx)
	}

	v, err := pctx.parseVersionInfo(ctx)
	if err != nil {
		return pctx.error(ctx, err)
	}
	pctx.version = v

	if !isBlankByte(cur.Peek()) {
		if cur.Peek() == '?' && cur.PeekAt(1) == '>' {
			if err := cur.Advance(2); err != nil {
				return err
			}
			return nil
		}

		return pctx.error(ctx, ErrSpaceRequired)
	}

	v, err = pctx.parseEncodingDecl(ctx)
	if err == nil && !pctx.options.IsSet(parseIgnoreEnc) {
		pctx.encoding = v
	}

	pctx.skipBlankBytes(ctx, cur)
	if cur.Peek() == '?' && cur.PeekAt(1) == '>' {
		if err := cur.Advance(2); err != nil {
			return err
		}
		return nil
	}

	vb, err := pctx.parseStandaloneDecl(ctx)
	if err == nil {
		pctx.standalone = vb
	}

	pctx.skipBlankBytes(ctx, cur)
	if cur.Peek() == '?' && cur.PeekAt(1) == '>' {
		if err := cur.Advance(2); err != nil {
			return err
		}
		return nil
	}
	return pctx.error(ctx, errors.New("XML declaration not closed"))
}

// parseXMLDeclLenient parses the XML declaration pseudo-attributes in any order.
// Called when parseopts.LenientXMLDecl is set.
func (pctx *parserCtx) parseXMLDeclLenient(ctx context.Context) error {
	cur := pctx.getByteCursor()

	for {
		pctx.skipBlankBytes(ctx, cur)
		if cur.Peek() == '?' && cur.PeekAt(1) == '>' {
			if err := cur.Advance(2); err != nil {
				return err
			}
			return nil
		}

		if v, err := pctx.parseVersionInfo(ctx); err == nil {
			pctx.version = v
			continue
		}

		if v, err := pctx.parseEncodingDecl(ctx); err == nil {
			if !pctx.options.IsSet(parseIgnoreEnc) {
				pctx.encoding = v
			}
			continue
		}

		if vb, err := pctx.parseStandaloneDecl(ctx); err == nil {
			pctx.standalone = vb
			continue
		}

		return pctx.error(ctx, errors.New("XML declaration not closed"))
	}
}

// parseXMLDeclFromCursor parses the XML declaration from a rune cursor.
// This is used for UTF-16 documents where the encoding has already been
// switched before parsing the XML declaration.
func (pctx *parserCtx) parseXMLDeclFromCursor(ctx context.Context) error {
	cur := pctx.getCursor()
	if cur == nil {
		return errors.New("rune cursor required for parseXMLDeclFromCursor")
	}

	if !cur.ConsumeString("<?xml") {
		return pctx.error(ctx, ErrInvalidXMLDecl)
	}

	if !pctx.skipBlanks(ctx) {
		return errors.New("blank needed after '<?xml'")
	}

	if pctx.options.IsSet(parseLenientXMLDecl) {
		return pctx.parseXMLDeclFromCursorLenient(ctx)
	}

	v, err := pctx.parseVersionInfoFromCursor(ctx)
	if err != nil {
		return pctx.error(ctx, err)
	}
	pctx.version = v

	if !isBlankByte(cur.Peek()) {
		if cur.Peek() == '?' {
			if err := cur.Advance(1); err != nil {
				return err
			}
			if cur.Peek() == '>' {
				if err := cur.Advance(1); err != nil {
					return err
				}
				return nil
			}
			return pctx.error(ctx, errors.New("XML declaration not closed"))
		}
		return pctx.error(ctx, ErrSpaceRequired)
	}

	ev, err := pctx.parseEncodingDeclFromCursor(ctx)
	if err == nil && !pctx.options.IsSet(parseIgnoreEnc) {
		pctx.encoding = ev
	}

	pctx.skipBlanks(ctx)
	if cur.Peek() == '?' {
		if err := cur.Advance(1); err != nil {
			return err
		}
		if cur.Peek() == '>' {
			if err := cur.Advance(1); err != nil {
				return err
			}
			return nil
		}
		return pctx.error(ctx, errors.New("XML declaration not closed"))
	}

	sv, err := pctx.parseStandaloneDeclFromCursor(ctx)
	if err == nil {
		pctx.standalone = sv
	}

	pctx.skipBlanks(ctx)
	if cur.Peek() == '?' {
		if err := cur.Advance(1); err != nil {
			return err
		}
		if cur.Peek() == '>' {
			if err := cur.Advance(1); err != nil {
				return err
			}
			return nil
		}
	}
	return pctx.error(ctx, errors.New("XML declaration not closed"))
}

// parseXMLDeclFromCursorLenient parses the XML declaration pseudo-attributes
// in any order using the rune cursor. Called when parseopts.LenientXMLDecl is set.
func (pctx *parserCtx) parseXMLDeclFromCursorLenient(ctx context.Context) error {
	cur := pctx.getCursor()

	for {
		pctx.skipBlanks(ctx)
		if cur.Peek() == '?' {
			if err := cur.Advance(1); err != nil {
				return err
			}
			if cur.Peek() == '>' {
				if err := cur.Advance(1); err != nil {
					return err
				}
				return nil
			}
			return pctx.error(ctx, errors.New("XML declaration not closed"))
		}

		if v, err := pctx.parseVersionInfoFromCursor(ctx); err == nil {
			pctx.version = v
			continue
		}

		if ev, err := pctx.parseEncodingDeclFromCursor(ctx); err == nil {
			if !pctx.options.IsSet(parseIgnoreEnc) {
				pctx.encoding = ev
			}
			continue
		}

		if sv, err := pctx.parseStandaloneDeclFromCursor(ctx); err == nil {
			pctx.standalone = sv
			continue
		}

		return pctx.error(ctx, errors.New("XML declaration not closed"))
	}
}
