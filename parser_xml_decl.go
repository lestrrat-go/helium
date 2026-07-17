package helium

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// The XML versions helium implements. A document declaring anything else is
// rejected or warned about by checkDocumentVersion.
const (
	xmlVersion10 = "1.0"
	xmlVersion11 = "1.1"
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
	if err := pctx.checkDocumentVersion(ctx, v); err != nil {
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
	if err != nil {
		// An "encoding" keyword that is present but malformed (missing '=',
		// missing opening quote, or an invalid EncName) is a fatal error per
		// EncodingDecl [80]/EncName [81]; only a wholly-absent keyword
		// (AttrNotFoundError) is benign and falls through to the optional
		// StandaloneDecl (W3C ibm-not-wf-P80-ibm80n03).
		var nf AttrNotFoundError
		if !errors.As(err, &nf) {
			return pctx.error(ctx, err)
		}
	} else if !pctx.options.IsSet(parseIgnoreEnc) {
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

// documentVersion returns the XML version of the document being parsed — the
// value from the document's XML declaration, or the parser context's recorded
// version when the Document node is not attached (e.g. the throwaway sub-context
// that decodes an external entity's TextDecl, which is seeded with the parent
// document's version). An empty result means no declaration was seen; the caller
// treats that as "1.0".
func (pctx *parserCtx) documentVersion() string {
	if pctx.doc != nil && pctx.doc.Version() != "" {
		return pctx.doc.Version()
	}
	return pctx.version
}

// checkEntityVersion enforces the XML §4.3.4 version-compatibility constraint:
// an external parsed entity (or the external DTD subset) whose TextDecl declares
// an XML version LATER than the referencing document's is a fatal error (libxml2
// XML_ERR_VERSION_MISMATCH). The matrix: a 1.0 document may not reference a 1.1
// entity (fatal); a 1.0 document with a 1.0 or version-less entity is fine; a 1.1
// document may reference a 1.1 OR a 1.0 entity. An absent/empty entity version is
// compatible and never rejected. The comparison is against the actual document
// version (documentVersion) — the sub-context that parses the TextDecl carries no
// Document node, so it is seeded with the parent document's version by
// decodeExternalPEContent / decodeFixedWidthExternalContent.
func (pctx *parserCtx) checkEntityVersion(entityVersion string) error {
	if entityVersion == "" || entityVersion == xmlVersion10 {
		return nil
	}
	docVersion := pctx.documentVersion()
	if docVersion == "" {
		docVersion = xmlVersion10
	}
	if docVersion != xmlVersion10 {
		return nil
	}
	return ErrEntityVersionMismatch
}

// checkDocumentVersion enforces the VersionNum constraint on a DOCUMENT's XML
// declaration (libxml2 xmlParseXMLDecl). XML 1.0 5th edition restricts
// VersionNum to '1.' [0-9]+, but the grammar parseVersionNum accepts is looser —
// [0-9] '.' [0-9]+, mirroring libxml2's xmlParseVersionNum — so the constraint is
// applied here instead:
//
//   - "1.0" and "1.1" are supported and pass silently. libxml2 warns on ANY
//     version other than "1.0", including "1.1"; helium implements XML 1.1
//     (isXML11 drives 1.1 escaping and the §4.3.4 entity matrix), so warning on
//     every 1.1 document would contradict that support. This is the one
//     deliberate divergence here.
//   - any other "1.x" is reported as a warning and parsing CONTINUES
//     (XML_WAR_UNKNOWN_VERSION); the declared version is retained.
//   - anything outside the 1.x family (e.g. "0.0", "2.0") is FATAL
//     (XML_ERR_UNKNOWN_VERSION).
//
// This applies only to a document's XMLDecl, never to an external entity's
// TextDecl — that carries its own §4.3.4 rule (see checkEntityVersion).
func (pctx *parserCtx) checkDocumentVersion(ctx context.Context, version string) error {
	if version == xmlVersion10 || version == xmlVersion11 {
		return nil
	}
	if !strings.HasPrefix(version, "1.") {
		return fmt.Errorf("%w %q", ErrUnsupportedXMLVersion, version)
	}
	return pctx.warning(ctx, "Unsupported version '%s'", version)
}

// parseTextDecl parses an external-entity TextDecl from the byte cursor,
// enforcing the XML grammar:
//
//	TextDecl ::= '<?xml' VersionInfo? EncodingDecl S? '?>'
//
// Unlike an XMLDecl, the VersionInfo is OPTIONAL, the EncodingDecl is REQUIRED,
// and NO StandaloneDecl is permitted. A declaration that begins with "<?xml"
// but violates this grammar — a version-only declaration, one carrying a
// standalone pseudo-attribute, or one missing the encoding — is rejected rather
// than leniently accepted (the lenient XMLDecl parser would wrongly tolerate
// all three). This is the parser used for an external parameter/general entity's
// leading TextDecl, where accepting an out-of-grammar declaration would silently
// misinterpret the entity's replacement text.
func (pctx *parserCtx) parseTextDecl(ctx context.Context) error {
	cur := pctx.getByteCursor()
	if cur == nil {
		return ErrByteCursorRequired
	}

	if !cur.Consume(xmlDeclHint) {
		return pctx.error(ctx, ErrInvalidXMLDecl)
	}

	if !pctx.skipBlankBytes(ctx, cur) {
		return pctx.error(ctx, errors.New("blank needed after '<?xml'"))
	}

	// VersionInfo is OPTIONAL in a TextDecl. Detect it by the literal "version"
	// token (the cursor is positioned at the first pseudo-attribute after the
	// required blank). When present it must be well formed and followed by a
	// blank separating it from the required EncodingDecl.
	if cur.HasPrefix(versionBytes) {
		v, err := pctx.parseVersionInfo(ctx)
		if err != nil {
			return pctx.error(ctx, err)
		}
		if err := pctx.checkEntityVersion(v); err != nil {
			return pctx.error(ctx, err)
		}
		pctx.version = v

		if !isBlankByte(cur.Peek()) {
			return pctx.error(ctx, ErrSpaceRequired)
		}
		pctx.skipBlankBytes(ctx, cur)
	}

	// EncodingDecl is REQUIRED in a TextDecl. A version-only declaration falls
	// through to here and is rejected.
	v, err := pctx.parseEncodingDecl(ctx)
	if err != nil {
		return pctx.error(ctx, errors.New("TextDecl requires an encoding declaration"))
	}
	if !pctx.options.IsSet(parseIgnoreEnc) {
		pctx.encoding = v
	}

	// Optional trailing space, then the required "?>". No StandaloneDecl is
	// permitted: a 'standalone' pseudo-attribute (or any other leftover content)
	// leaves a non-"?>" byte here and is rejected.
	pctx.skipBlankBytes(ctx, cur)
	if cur.Peek() == '?' && cur.PeekAt(1) == '>' {
		if err := cur.Advance(2); err != nil {
			return err
		}
		return nil
	}

	return pctx.error(ctx, errors.New("malformed TextDecl: expected '?>' after encoding declaration"))
}

// parseTextDeclFromCursor parses an external-entity TextDecl from a rune cursor.
// It is the fixed-width-Unicode counterpart of parseTextDecl, used when the
// external content is in UTF-16 / UCS-4 whose bytes switchEncoding already
// decoded (so the TextDecl, being itself encoded, could not be read at byte
// level). It enforces the same grammar:
//
//	TextDecl ::= '<?xml' VersionInfo? EncodingDecl S? '?>'
//
// VersionInfo is OPTIONAL, EncodingDecl is REQUIRED, and NO StandaloneDecl is
// permitted — a version-only, standalone-bearing, or otherwise out-of-grammar
// declaration is rejected. The declared encoding is informational here (the
// BOM/leading-'<' shape already fixed the actual encoding), so it does not drive
// a re-switch; it is recorded only to mirror the byte-level path.
func (pctx *parserCtx) parseTextDeclFromCursor(ctx context.Context) error {
	cur := pctx.getCursor()
	if cur == nil {
		return errors.New("rune cursor required for parseTextDeclFromCursor")
	}

	if !cur.ConsumeString("<?xml") {
		return pctx.error(ctx, ErrInvalidXMLDecl)
	}

	if !pctx.skipBlanks(ctx) {
		return pctx.error(ctx, errors.New("blank needed after '<?xml'"))
	}

	// VersionInfo is OPTIONAL. Detect it by the literal "version" token; when
	// present it must be well formed and followed by a blank separating it from
	// the required EncodingDecl.
	if cur.HasPrefixString("version") {
		v, err := pctx.parseVersionInfoFromCursor(ctx)
		if err != nil {
			return pctx.error(ctx, err)
		}
		if err := pctx.checkEntityVersion(v); err != nil {
			return pctx.error(ctx, err)
		}
		pctx.version = v

		if !isBlankByte(cur.Peek()) {
			return pctx.error(ctx, ErrSpaceRequired)
		}
		pctx.skipBlanks(ctx)
	}

	// EncodingDecl is REQUIRED. A version-only declaration falls through here and
	// is rejected.
	ev, err := pctx.parseEncodingDeclFromCursor(ctx)
	if err != nil {
		return pctx.error(ctx, errors.New("TextDecl requires an encoding declaration"))
	}
	if !pctx.options.IsSet(parseIgnoreEnc) {
		pctx.encoding = ev
	}

	// Optional trailing space, then the required "?>". A 'standalone'
	// pseudo-attribute (or any other leftover content) leaves a non-'?' byte here
	// and is rejected.
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

	return pctx.error(ctx, errors.New("malformed TextDecl: expected '?>' after encoding declaration"))
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
			if err := pctx.checkDocumentVersion(ctx, v); err != nil {
				return pctx.error(ctx, err)
			}
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
	if err := pctx.checkDocumentVersion(ctx, v); err != nil {
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
	if err != nil {
		// A present-but-malformed "encoding" keyword is fatal (EncodingDecl
		// [80]/EncName [81]); only a wholly-absent keyword (AttrNotFoundError)
		// falls through to the optional StandaloneDecl.
		var nf AttrNotFoundError
		if !errors.As(err, &nf) {
			return pctx.error(ctx, err)
		}
	} else if !pctx.options.IsSet(parseIgnoreEnc) {
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
			if err := pctx.checkDocumentVersion(ctx, v); err != nil {
				return pctx.error(ctx, err)
			}
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
