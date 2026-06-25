package helium

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/internal/iolimit"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/sax"
)

func (pctx *parserCtx) parseEntityValueInternal(ctx context.Context, qch byte) (string, error) {
	cur := pctx.getCursor()
	if cur == nil {
		return "", pctx.error(ctx, errNoCursor)
	}
	buf := bufferPool.Get()
	defer releaseBuffer(buf)

	off := 0
	for {
		b := cur.PeekAt(off)
		if b == 0 || b == qch {
			break
		}
		if b < 0x80 {
			if !isChar(rune(b)) {
				break
			}
			buf.WriteByte(b)
			off++
			continue
		}
		r, w, ok := decodeRuneAt(cur, off)
		if !ok || !isCharWidth(r, w) {
			break
		}
		buf.WriteRune(r)
		off += w
	}
	if off > 0 {
		if err := cur.Advance(off); err != nil {
			return "", pctx.error(ctx, err)
		}
		return buf.String(), nil
	}
	return "", nil
}

func (pctx *parserCtx) decodeEntities(ctx context.Context, s []byte, what SubstitutionType) (ret string, err error) {
	ret, err = pctx.decodeEntitiesInternal(ctx, s, what, 0)
	return
}

func (pctx *parserCtx) decodeEntitiesInternal(ctx context.Context, s []byte, what SubstitutionType, depth int) (string, error) {
	if depth > 40 {
		return "", errors.New("entity loop (depth > 40)")
	}

	out := bufferPool.Get()
	defer releaseBuffer(out)

	for len(s) > 0 {
		if bytes.HasPrefix(s, []byte{'&', '#'}) {
			val, width, err := parseStringCharRef(s)
			if err != nil {
				return "", err
			}
			out.WriteRune(val)
			s = s[width:]
		} else if s[0] == '&' && what&SubstituteRef == SubstituteRef {
			ent, width, err := pctx.parseStringEntityRef(ctx, s)
			if err != nil {
				return "", err
			}
			if ent == nil {
				_, _ = out.Write(s[:width])
				s = s[width:]
				continue
			}
			if err := pctx.entityCheck(ent, 0); err != nil {
				return "", err
			}

			if ent.EntityType() == enum.InternalPredefinedEntity {
				if len(ent.Content()) == 0 {
					return "", errors.New("predefined entity has no content")
				}
				_, _ = out.Write(ent.Content())
			} else if len(ent.Content()) != 0 {
				rep, err := pctx.decodeEntitiesInternal(ctx, ent.Content(), what, depth+1)
				if err != nil {
					return "", err
				}
				if err := pctx.entityCheck(ent, len(rep)); err != nil {
					return "", err
				}

				_, _ = out.WriteString(rep)
			} else {
				_, _ = out.WriteString(ent.Name())
			}
			s = s[width:]
		} else if s[0] == '%' && what&SubstitutePERef == SubstitutePERef {
			ent, width, err := pctx.parseStringPEReference(ctx, s)
			if err != nil {
				return "", err
			}
			if err := pctx.entityCheck(ent, width); err != nil {
				return "", err
			}
			rep, err := pctx.decodeEntitiesInternal(ctx, ent.Content(), what, depth+1)
			if err != nil {
				return "", err
			}
			if err := pctx.entityCheck(ent, len(rep)); err != nil {
				return "", err
			}
			_, _ = out.WriteString(rep)
			s = s[width:]
		} else {
			_ = out.WriteByte(s[0])
			s = s[1:]
		}
	}
	return out.String(), nil
}

func (pctx *parserCtx) parseEntityValue(ctx context.Context) (string, string, error) {
	pctx.instate = psEntityValue

	literal, err := pctx.parseQuotedText(func(qch byte) (string, error) {
		return pctx.parseEntityValueInternal(ctx, qch)
	})
	if err != nil {
		return "", "", pctx.error(ctx, err)
	}

	// decodeEntities below only substitutes parameter-entity references; general
	// references are left literal and never syntax-checked. Validate them here so
	// a malformed reference (e.g. a missing semicolon) is rejected rather than
	// silently stored. This does not expand the general references.
	//
	// Validation runs over the PE-EXPANDED lexical stream so that a malformed
	// general reference re-introduced through a parameter entity is still caught.
	// For example, an external DTD with
	//   <!ENTITY % amp "&#38;">  <!ENTITY e "%amp;broken">
	// expands to "&broken" and must be rejected, matching libxml2/xmllint.
	// Direct character references in the literal (e.g. "&#38;") are character
	// data and never form a general reference with following text, so they are
	// consumed as data; only PE replacement text re-participates in ref scanning.
	if err := pctx.validateEntityValueRefs(ctx, []byte(literal)); err != nil {
		return "", "", pctx.error(ctx, err)
	}

	val, err := pctx.decodeEntities(ctx, []byte(literal), SubstitutePERef)
	if err != nil {
		return "", "", pctx.error(ctx, err)
	}

	return literal, val, nil
}

// validateEntityValueRefs checks that every general reference in an EntityValue
// is well formed: a '&' must begin either a character reference (&#...; or
// &#x...;) or a general-entity reference (&Name;). The references are not
// expanded; this only enforces syntax so a malformed reference such as
// "&broken" (missing semicolon) is rejected.
//
// Parameter-entity references (%Name;) ARE expanded first, because their
// replacement text can re-introduce general references (including ones that
// only become malformed after a character reference inside the PE resolves to a
// literal '&'). The PE-expanded buffer is then scanned for general references.
// Character references that appear directly in the literal are character data
// and are consumed without contributing a literal '&' to the scan.
func (pctx *parserCtx) validateEntityValueRefs(ctx context.Context, s []byte) error {
	// The validation expansion below runs decodeEntitiesInternal, which charges
	// the amplification counters via entityCheck. This is only a syntax check —
	// the real PE substitution in parseEntityValue re-expands the same value and
	// charges the counters for real. Snapshot and restore the counters so this
	// pass is side-effect-free and the same parameter entities are not counted
	// twice.
	//
	// The PE-expansion path also resolves parameter-entity references through
	// parseStringPEReference, which MUTATES live parser state — it sets
	// pctx.hasPERefs (and, on an unresolved PE in a non-standalone document,
	// clears pctx.valid). Those mutations belong to the real parse, not to this
	// throwaway syntax check: a validation that fails (or even one that succeeds)
	// must not leave hasPERefs/valid perturbed. Snapshot and restore both so the
	// whole pass is side-effect-free.
	savedSize := pctx.sizeentcopy
	savedHasPERefs := pctx.hasPERefs
	savedValid := pctx.valid
	defer func() {
		pctx.sizeentcopy = savedSize
		pctx.hasPERefs = savedHasPERefs
		pctx.valid = savedValid
	}()

	expanded, err := pctx.expandEntityValueForRefCheck(ctx, s, 0)
	if err != nil {
		return err
	}
	return scanEntityValueGeneralRefs(expanded)
}

// expandEntityValueForRefCheck produces the lexical stream over which general
// references are validated. Parameter-entity references are replaced by their
// replacement text (recursively), and character references found inside that
// replacement text resolve to their literal characters so a "&#38;" coming from
// a PE becomes a literal '&' that can combine with following text to form a
// general reference. Character references that appear directly in the literal
// are ALWAYS emitted as an inert placeholder (a space, never '&', a NameChar, or
// ';') so they remain character data and can never combine with surrounding text
// into a "&Name;". Only parameter-entity replacement text re-enters
// general-reference scanning.
func (pctx *parserCtx) expandEntityValueForRefCheck(ctx context.Context, s []byte, depth int) ([]byte, error) {
	if depth > 40 {
		return nil, errors.New("entity loop (depth > 40)")
	}

	out := bufferPool.Get()
	defer releaseBuffer(out)

	for len(s) > 0 {
		if bytes.HasPrefix(s, []byte{'&', '#'}) {
			// Direct character reference: validate its syntax but treat the
			// result as character data, not as a character that could form a
			// general reference with surrounding text.
			_, width, err := parseStringCharRef(s)
			if err != nil {
				return nil, err
			}
			// Emit an inert placeholder for EVERY direct character reference so
			// it can never combine with surrounding text into a "&Name;". A
			// space is neither '&', a NameChar, nor ';', so it cannot be part of
			// any reference. This is required not only for "&#38;" (which would
			// resolve to a literal '&') but for any char ref: e.g. "&&#97;;"
			// must stay malformed (a bare '&' followed by character data) rather
			// than synthesize "&a;", and "&a&#59;" must not synthesize a
			// trailing ';' to complete "&a;". Only PARAMETER-ENTITY replacement
			// text is allowed to re-enter general-reference scanning.
			out.WriteByte(' ')
			s = s[width:]
			continue
		}
		if s[0] == '%' {
			ent, width, err := pctx.parseStringPEReference(ctx, s)
			if err != nil {
				return nil, err
			}
			if ent != nil {
				// Expand the PE replacement text. decodeEntitiesInternal
				// recursively substitutes nested parameter entities and resolves
				// character references to their literal characters, so a "&#38;"
				// brought in by the PE becomes a literal '&' that can combine
				// with surrounding text into a general reference. General
				// references (&Name;) in the replacement text are left intact for
				// the subsequent scan.
				rep, err := pctx.decodeEntitiesInternal(ctx, ent.Content(), SubstitutePERef, depth+1)
				if err != nil {
					return nil, err
				}
				out.WriteString(rep)
			}
			s = s[width:]
			continue
		}
		out.WriteByte(s[0])
		s = s[1:]
	}

	res := make([]byte, out.Len())
	copy(res, out.Bytes())
	return res, nil
}

// scanEntityValueGeneralRefs validates that every '&' in the (PE-expanded)
// EntityValue stream begins a well-formed character or general reference. A
// missing semicolon or an otherwise malformed reference is rejected.
func scanEntityValueGeneralRefs(s []byte) error {
	for len(s) > 0 {
		i := bytes.IndexByte(s, '&')
		if i < 0 {
			return nil
		}
		s = s[i:]
		if bytes.HasPrefix(s, []byte{'&', '#'}) {
			_, width, err := parseStringCharRef(s)
			if err != nil {
				return err
			}
			s = s[width:]
			continue
		}

		// General-entity reference: &Name; — parse the name then require ';'.
		if len(s) < 2 {
			return errors.New("malformed entity reference in entity value")
		}
		_, width, err := parseStringName(s[1:])
		if err != nil {
			return errors.New("malformed entity reference in entity value")
		}
		rest := s[1+width:]
		if len(rest) == 0 || rest[0] != ';' {
			return ErrSemicolonRequired
		}
		s = rest[1:]
	}
	return nil
}

func (pctx *parserCtx) parseEntityDecl(ctx context.Context) error {
	cur := pctx.getCursor()
	if cur == nil {
		return pctx.error(ctx, errNoCursor)
	}
	if !cur.ConsumeString("<!ENTITY") {
		return pctx.error(ctx, errors.New("<!ENTITY not started"))
	}

	if !pctx.skipBlanks(ctx) {
		return pctx.error(ctx, ErrSpaceRequired)
	}

	isParameter := false
	if cur.Peek() == '%' {
		if err := cur.Advance(1); err != nil {
			return err
		}
		if !pctx.skipBlanks(ctx) {
			return pctx.error(ctx, ErrSpaceRequired)
		}
		isParameter = true
	}

	name, err := pctx.parseName(ctx)
	if err != nil {
		return pctx.error(ctx, err)
	}
	if strings.IndexByte(name, ':') > -1 {
		return pctx.error(ctx, errors.New("colons are forbidden from entity names"))
	}

	if !pctx.skipBlanks(ctx) {
		return pctx.error(ctx, ErrSpaceRequired)
	}

	pctx.instate = psEntityDecl
	var literal string
	var value string
	var uri string
	var hasOrig bool

	if isParameter {
		if c := cur.Peek(); c == '"' || c == '\'' {
			literal, value, err = pctx.parseEntityValue(ctx)
			hasOrig = true
			if err != nil {
				return pctx.error(ctx, err)
			}
			switch err := pctx.fireSAXCallback(ctx, cbEntityDecl, name, value); err {
			case nil, sax.ErrHandlerUnspecified:
			default:
				return pctx.error(ctx, err)
			}
		} else {
			literal, uri, err = pctx.parseExternalID(ctx)
			if err != nil {
				return pctx.error(ctx, ErrValueRequired)
			}

			if uri != "" {
				u, err := url.Parse(uri)
				if err != nil {
					return pctx.error(ctx, err)
				}

				if u.Fragment != "" {
					return pctx.error(ctx, errors.New("err uri fragment"))
				} else if s := pctx.sax; s != nil {
					switch err := s.EntityDecl(ctx, name, enum.ExternalParameterEntity, literal, uri, ""); err {
					case nil, sax.ErrHandlerUnspecified:
					default:
						return pctx.error(ctx, err)
					}
				}
			}
		}
	} else {
		if c := cur.Peek(); c == '"' || c == '\'' {
			literal, value, err = pctx.parseEntityValue(ctx)
			hasOrig = true
			if err != nil {
				return pctx.error(ctx, err)
			}
			if s := pctx.sax; s != nil {
				switch err := s.EntityDecl(ctx, name, enum.InternalGeneralEntity, "", "", value); err {
				case nil, sax.ErrHandlerUnspecified:
				default:
					return pctx.error(ctx, err)
				}
			}
		} else {
			literal, uri, err = pctx.parseExternalID(ctx)
			if err != nil {
				return pctx.error(ctx, ErrValueRequired)
			}

			if literal != "" {
				u, err := url.Parse(literal)
				if err != nil {
					return pctx.error(ctx, err)
				}
				if u.Fragment != "" {
					return pctx.error(ctx, errors.New("err uri fragment"))
				}
			}

			if c := cur.Peek(); c != '>' && !isBlankByte(c) {
				return pctx.error(ctx, ErrSpaceRequired)
			}

			pctx.skipBlanks(ctx)
			if cur.ConsumeString("NDATA") {
				if !pctx.skipBlanks(ctx) {
					return pctx.error(ctx, ErrSpaceRequired)
				}

				ndata, err := pctx.parseName(ctx)
				if err != nil {
					return pctx.error(ctx, err)
				}
				if s := pctx.sax; s != nil {
					switch err := s.UnparsedEntityDecl(ctx, name, uri, literal, ndata); err {
					case nil, sax.ErrHandlerUnspecified:
					default:
						return pctx.error(ctx, err)
					}
				}
			} else {
				if s := pctx.sax; s != nil {
					switch err := s.EntityDecl(ctx, name, enum.ExternalGeneralParsedEntity, uri, literal, ""); err {
					case nil, sax.ErrHandlerUnspecified:
					default:
						return pctx.error(ctx, err)
					}
				}
			}
		}
	}

	pctx.skipBlanks(ctx)
	if cur.Peek() != '>' {
		return pctx.error(ctx, errors.New("entity not terminated"))
	}
	if err := cur.Advance(1); err != nil {
		return err
	}

	if hasOrig {
		var current sax.Entity
		if isParameter {
			if s := pctx.sax; s != nil {
				current, _ = s.GetParameterEntity(ctx, name)
			}
		} else {
			if s := pctx.sax; s != nil {
				current, _ = s.GetEntity(ctx, name)
				if current == nil {
					e, _ := pctx.getEntity(name)
					current = e
				}
			}
		}
		if current != nil {
			if ent, ok := current.(*Entity); ok && ent != nil && ent.orig == "" {
				ent.SetOrig(literal)
			}
		}
	}

	return nil
}

// inheritNestedParserState copies the security/behavior policy and the running
// resource-accounting state from a parent parser context onto a freshly
// initialized nested context used to parse entity replacement text (internal or
// external). Both the internal balanced-chunk path and the external entity path
// must seed identical policy so entity replacement text honors the same depth,
// resource, normalization, and sandbox limits as the top-level parse instead of
// silently falling back to zero-value defaults (which would, for example, reset
// element-depth accounting to 0 and let MaxDepth be bypassed via a substituted
// entity, or default fsys back to the permissive os.Open root).
//
// Notably this copies BOTH maxElemDepth (the configured limit) AND the current
// elemDepth, so element nesting that crosses an entity-expansion boundary keeps
// accumulating against the same limit rather than restarting at 0.
//
// It does NOT touch newctx.doc, newctx.external, or the amplification counters
// (sizeentcopy/inputSize/maxAmpl); those are handled by the caller because their
// lifecycle (document swap, external flag, write-back on return) differs between
// the two paths.
func (pctx *parserCtx) inheritNestedParserState(newctx *parserCtx) {
	newctx.sax = pctx.sax
	newctx.treeBuilder = pctx.treeBuilder
	newctx.attsDefault = pctx.attsDefault
	newctx.options = pctx.options
	newctx.loadsubset = pctx.loadsubset
	newctx.replaceEntities = pctx.replaceEntities
	newctx.keepBlanks = pctx.keepBlanks
	newctx.pedantic = pctx.pedantic
	newctx.charBufferSize = pctx.charBufferSize
	newctx.maxExtDTDSize = pctx.maxExtDTDSize
	// Carry the name-length and content-model-depth caps so a configured limit
	// (Parser.MaxNameLength / Parser.MaxContentModelDepth) is enforced on
	// entity-expansion sub-parses too. (These used to ride in options via
	// XML_PARSE_HUGE; the granular limit knobs store them as separate fields.)
	newctx.maxNameLength = pctx.maxNameLength
	newctx.maxCMDepth = pctx.maxCMDepth
	// Carry both the element-depth limit and the current depth so nesting that
	// crosses the entity boundary keeps counting toward MaxDepth.
	newctx.maxElemDepth = pctx.maxElemDepth
	newctx.elemDepth = pctx.elemDepth
	// Inherit the parent's security/resolution policy so any external reference
	// reached while expanding this replacement text honors the same FS sandbox,
	// catalog, and base URI as the top-level parse rather than falling back to
	// the permissive os.Open root.
	newctx.fsys = pctx.fsys
	newctx.catalog = pctx.catalog
	newctx.baseURI = pctx.baseURI
}

func (pctx *parserCtx) parseExternalEntityPrivate(ctx context.Context, uri, externalID string) (Node, error) {
	if pctx.options.IsSet(parseNoXXE) {
		return nil, nil //nolint:nilnil
	}

	pctx.depth++
	defer func() { pctx.depth-- }()

	if pctx.depth > 40 {
		return nil, errors.New("entity loop")
	}

	var input sax.ParseInput
	if s := pctx.sax; s != nil {
		resolved, err := s.ResolveEntity(ctx, externalID, uri)
		switch err {
		case nil:
			input = resolved
		case sax.ErrHandlerUnspecified:
		default:
			return nil, pctx.error(ctx, err)
		}
	}

	if input == nil {
		return nil, fmt.Errorf("cannot resolve external entity (URI=%s, publicID=%s)", uri, externalID)
	}

	// The resolved input may hold an OS resource (the default resolver returns a
	// fileParseInput embedding an open *os.File). Close it as soon as the bounded
	// read completes — before any size/error handling and before the buffered
	// content is parsed — so the fd is never held open for the lifetime of the
	// nested parse. Not deferred: the close must happen at the read boundary, not
	// at function return.
	closeInput := func() {
		if c, ok := input.(io.Closer); ok {
			_ = c.Close()
		}
	}

	// Read through a bounded reader so an unbounded source (e.g. SYSTEM
	// "/dev/zero") cannot exhaust memory. LimitReader allows one extra byte so a
	// content length exactly at the cap is accepted while anything larger is
	// detected.
	content, exceeded, err := iolimit.ReadAll(input, externalEntityMaxBytes)
	closeInput()
	if err != nil {
		return nil, pctx.error(ctx, fmt.Errorf("reading external entity: %w", err))
	}
	if exceeded {
		return nil, pctx.error(ctx, fmt.Errorf("external entity (URI=%s) exceeds maximum size of %d bytes", uri, externalEntityMaxBytes))
	}

	// Charge the external content to the amplification counters. Without this an
	// external entity that is just under externalEntityMaxBytes could be
	// referenced repeatedly to bypass the entity-expansion limits entirely (the
	// per-reference cost would otherwise be ~0 because external entities carry no
	// inline content). Use the byte-only charge: parseReference already paid the
	// per-reference entityFixedCost via entityCheck, so charging entityCheck here
	// would double-count the fixed cost. entityCheckBytes still enforces both the
	// absolute ceiling and the amplification-ratio check against the accumulated
	// size.
	if err := pctx.entityCheckBytes(len(content)); err != nil {
		return nil, pctx.error(ctx, err)
	}

	newctx := &parserCtx{}
	if err := newctx.init(nil, bytes.NewReader(content)); err != nil {
		return nil, err
	}
	defer func() {
		_ = newctx.release()
	}()

	if pctx.doc == nil {
		pctx.doc = NewDocument("1.0", "", StandaloneExplicitNo)
	}

	fc := pctx.doc.FirstChild()
	lc := pctx.doc.LastChild()
	setFirstChild(pctx.doc, nil)
	setLastChild(pctx.doc, nil)
	defer func() {
		setFirstChild(pctx.doc, fc)
		setLastChild(pctx.doc, lc)
	}()
	newctx.doc = pctx.doc
	newctx.depth = pctx.depth + 1
	newctx.external = true
	// Seed all shared policy/state (security sandbox, normalization, element-depth
	// limit AND current depth, etc.) from the parent so external entity
	// replacement text cannot bypass MaxDepth or escape the configured sandbox.
	pctx.inheritNestedParserState(newctx)
	// Carry the amplification counters through the nested parse so any entity
	// expansion performed while parsing this external entity (including further
	// nested external entities) is charged against the same accumulated budget
	// as the top-level document, and write the running total back on return.
	newctx.sizeentcopy = pctx.sizeentcopy
	newctx.inputSize = pctx.inputSize
	newctx.maxAmpl = pctx.maxAmpl
	defer func() { pctx.sizeentcopy = newctx.sizeentcopy }()
	if pctx.elem != nil {
		for _, ns := range collectInScopeNamespaces(pctx.elem) {
			newctx.pushNS(ns.Prefix(), ns.URI())
		}
	}

	if newctx.encoding == "" {
		if enc, err := newctx.detectEncoding(); err == nil {
			newctx.detectedEncoding = enc
		}
	}

	innerCtx := withParserCtx(ctx, newctx)
	innerCtx = sax.WithDocumentLocator(innerCtx, newctx)
	innerCtx = context.WithValue(innerCtx, stopFuncKey{}, newctx.stop)

	bcur := newctx.getByteCursor()
	if bcur != nil && looksLikeXMLDecl(bcur) {
		if err := newctx.parseXMLDecl(innerCtx); err != nil {
			return nil, err
		}
	}

	if err := newctx.switchEncoding(); err != nil {
		return nil, err
	}

	newRoot := newctx.doc.CreateElement(pseudoRootName)
	newctx.pushNodeEntry(nodeEntry{local: pseudoRootName, qname: pseudoRootName})
	newctx.elem = newRoot
	if err := newctx.doc.AddChild(newRoot); err != nil {
		return nil, err
	}
	if err := newctx.parseContent(innerCtx); err != nil {
		return nil, err
	}
	// A clean parseContent may mask a transcoding/decode error (e.g. an unpaired
	// UTF-16 surrogate the decoder replaced with U+FFFD) in this context's own
	// byte stream. Surface it as fatal rather than inserting U+FFFD, matching the
	// document-level gate in parseDocument.
	if err := newctx.cursorDecodeErr(); err != nil {
		return nil, newctx.error(innerCtx, err)
	}

	if child := newctx.doc.FirstChild(); child != nil {
		if grandchild := child.FirstChild(); grandchild != nil {
			for e := grandchild; e != nil; e = e.NextSibling() {
				e.(MutableNode).SetTreeDoc(pctx.doc) //nolint:forcetypeassert
				e.baseDocNode().parent = nil
				if uri != "" {
					e.baseDocNode().entityBaseURI = uri
					if !pctx.options.IsSet(parseNoBaseFix) {
						if elem, ok := e.(*Element); ok {
							if _, exists := elem.GetAttributeNS("base", lexicon.NamespaceXML); !exists {
								_, _ = elem.SetAttributeNS("base", uri, newNamespace("xml", lexicon.NamespaceXML))
							}
						}
					}
				}
			}
			return grandchild, nil
		}
	}

	return nil, ErrParseSucceeded
}

var ErrParseSucceeded = errors.New("parse succeeded")

func (pctx *parserCtx) parseBalancedChunkInternal(ctx context.Context, chunk []byte) (Node, error) {
	pctx.depth++
	defer func() { pctx.depth-- }()

	if pctx.depth > 40 {
		return nil, errors.New("entity loop")
	}

	newctx := &parserCtx{}
	if err := newctx.init(nil, bytes.NewReader(chunk)); err != nil {
		return nil, err
	}
	defer func() {
		_ = newctx.release()
	}()

	if pctx.doc == nil {
		pctx.doc = NewDocument("1.0", "", StandaloneExplicitNo)
	}

	fc := pctx.doc.FirstChild()
	lc := pctx.doc.LastChild()
	setFirstChild(pctx.doc, nil)
	setLastChild(pctx.doc, nil)
	defer func() {
		setFirstChild(pctx.doc, fc)
		setLastChild(pctx.doc, lc)
	}()
	newctx.doc = pctx.doc
	newctx.depth = pctx.depth + 1
	// Seed all shared policy/state (security sandbox, normalization, element-depth
	// limit AND current depth, etc.) from the parent so internal-entity
	// replacement text honors the same limits as the top-level parse instead of
	// restarting depth accounting at 0 or falling back to zero-value defaults.
	pctx.inheritNestedParserState(newctx)
	if pctx.elem != nil {
		for _, ns := range collectInScopeNamespaces(pctx.elem) {
			newctx.pushNS(ns.Prefix(), ns.URI())
		}
	}
	newctx.sizeentcopy = pctx.sizeentcopy
	newctx.inputSize = pctx.inputSize
	newctx.maxAmpl = pctx.maxAmpl
	defer func() { pctx.sizeentcopy = newctx.sizeentcopy }()

	newRoot := newctx.doc.CreateElement(pseudoRootName)
	newctx.pushNodeEntry(nodeEntry{local: pseudoRootName, qname: pseudoRootName})
	newctx.elem = newRoot
	if err := newctx.doc.AddChild(newRoot); err != nil {
		return nil, err
	}
	if err := newctx.switchEncoding(); err != nil {
		return nil, err
	}
	innerCtx := withParserCtx(ctx, newctx)
	innerCtx = sax.WithDocumentLocator(innerCtx, newctx)
	innerCtx = context.WithValue(innerCtx, stopFuncKey{}, newctx.stop)
	if err := newctx.parseContent(innerCtx); err != nil {
		return nil, err
	}
	// A clean parseContent may mask a transcoding/decode error (e.g. an unpaired
	// UTF-16 surrogate the decoder replaced with U+FFFD) in this context's own
	// byte stream. Surface it as fatal rather than inserting U+FFFD, matching the
	// document-level gate in parseDocument.
	if err := newctx.cursorDecodeErr(); err != nil {
		return nil, newctx.error(innerCtx, err)
	}

	if child := newctx.doc.FirstChild(); child != nil {
		if grandchild := child.FirstChild(); grandchild != nil {
			for e := grandchild; e != nil; e = e.NextSibling() {
				e.(MutableNode).SetTreeDoc(pctx.doc) //nolint:forcetypeassert
				e.baseDocNode().parent = nil
			}
			return grandchild, nil
		}
	}

	return nil, ErrParseSucceeded
}
