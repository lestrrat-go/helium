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

// parameterEntityReplacement returns the replacement text of a resolved
// parameter entity. For an EXTERNAL parameter entity the replacement text is the
// content of the referenced external resource, which is loaded on demand through
// loadExternalParameterEntityContent (XXE-gated, byte-capped, cached on the
// entity). For an internal parameter entity it is the stored Content().
//
// This is the single chokepoint so a PE reference that appears inside an entity
// value (decodeEntitiesInternal) or in the entity-value reference syntax check
// (expandEntityValueForRefCheck) expands an external PE the same way and
// regardless of reference order, instead of seeing an empty Content() until the
// top-level parsePEReference happens to load and cache it first. When external
// loading is disabled (secure default: parseNoXXE, or the resolver declines) the
// load returns empty, so the result is the same empty replacement text as before
// — no behavior change in secure mode.
func (pctx *parserCtx) parameterEntityReplacement(ctx context.Context, ent sax.Entity) ([]byte, error) {
	if ent.EntityType() == enum.ExternalParameterEntity {
		if he, ok := ent.(*Entity); ok {
			content, _, err := pctx.loadExternalParameterEntityContent(ctx, he)
			if err != nil {
				return nil, err
			}
			return content, nil
		}
	}
	return ent.Content(), nil
}

func (pctx *parserCtx) parseEntityValueInternal(ctx context.Context, qch byte) (string, error) {
	cur := pctx.getCursor()
	if cur == nil {
		return "", pctx.error(ctx, errNoCursor)
	}
	// The entity value is an indivisible content run; scanQuotedLiteral bounds it
	// by the node-content cap and advances in chunks so an unbounded literal (e.g.
	// over an EBCDIC ParseReader stream) fails closed with ErrNodeContentTooLarge
	// instead of growing memory before any per-node cap fires.
	return pctx.scanQuotedLiteral(ctx, cur, qch, false)
}

func (pctx *parserCtx) decodeEntities(ctx context.Context, s []byte, what SubstitutionType) (ret string, err error) {
	ret, err = pctx.decodeEntitiesInternal(ctx, s, what, 0)
	return
}

// entityDecodeSink is the output target for decodeEntitiesToSink. The string
// path (decodeEntitiesInternal) accumulates into a pooled buffer; the
// attribute-value path streams directly into the attribute buffer through the
// node-content cap so an over-cap expansion fails DURING decode instead of being
// fully materialized first. count() reports the running output length so a
// nested expansion's contributed size can be measured for entityCheck without
// materializing the nested replacement string.
type entityDecodeSink interface {
	writeByte(context.Context, byte) error
	write(context.Context, []byte) error
	writeString(context.Context, string) error
	writeRune(context.Context, rune) error
	count() int
}

// entityStringSink accumulates decoded output into a buffer and returns it as a
// string. It imposes no cap (the historical decodeEntities behavior) and ignores
// the context.
type entityStringSink struct {
	buf *bytes.Buffer
}

func (s *entityStringSink) writeByte(_ context.Context, b byte) error {
	return s.buf.WriteByte(b)
}

func (s *entityStringSink) write(_ context.Context, p []byte) error {
	_, err := s.buf.Write(p)
	return err
}

func (s *entityStringSink) writeString(_ context.Context, p string) error {
	_, err := s.buf.WriteString(p)
	return err
}

func (s *entityStringSink) writeRune(_ context.Context, r rune) error {
	_, err := s.buf.WriteRune(r)
	return err
}

func (s *entityStringSink) count() int { return s.buf.Len() }

// attrEntitySink streams decoded entity-replacement bytes into an attribute
// buffer, applying attribute-value whitespace normalization (TAB/CR/LF -> space)
// and enforcing the node-content cap on every byte. Because the cap is checked
// before each append, an over-cap expansion (e.g. <r a="&big;"/> with
// SubstituteEntities, or a forced-replacement namespace attr xmlns:x="&big;")
// fails with ErrNodeContentTooLarge as soon as the running total would exceed
// the remaining budget, never materializing the full replacement first.
type attrEntitySink struct {
	pctx *parserCtx
	b    *bytes.Buffer
}

func (s *attrEntitySink) writeByte(ctx context.Context, by byte) error {
	switch by {
	case 0xD, 0xA, 0x9:
		by = 0x20
	}
	return s.pctx.writeAttrByte(ctx, s.b, by)
}

func (s *attrEntitySink) write(ctx context.Context, p []byte) error {
	for i := range p {
		if err := s.writeByte(ctx, p[i]); err != nil {
			return err
		}
	}
	return nil
}

func (s *attrEntitySink) writeString(ctx context.Context, p string) error {
	for i := range len(p) {
		if err := s.writeByte(ctx, p[i]); err != nil {
			return err
		}
	}
	return nil
}

func (s *attrEntitySink) writeRune(ctx context.Context, r rune) error {
	// A sub-0x80 rune may be a TAB/CR/LF char ref that must normalize to space,
	// so route it through writeByte. A multi-byte rune never contains a
	// 0x09/0x0A/0x0D byte, so writeAttrRune (cap-enforced, no normalization) is
	// correct and avoids a re-encode.
	if r < 0x80 {
		return s.writeByte(ctx, byte(r))
	}
	return s.pctx.writeAttrRune(ctx, s.b, r)
}

func (s *attrEntitySink) count() int { return s.b.Len() }

func (pctx *parserCtx) decodeEntitiesInternal(ctx context.Context, s []byte, what SubstitutionType, depth int) (string, error) {
	out := bufferPool.Get()
	defer releaseBuffer(out)

	sink := &entityStringSink{buf: out}
	if err := pctx.decodeEntitiesToSink(ctx, s, what, depth, sink); err != nil {
		return "", err
	}
	return out.String(), nil
}

// decodeEntitiesToSink performs the entity-substitution decode, writing every
// output byte through sink. It is the shared core of the string-returning
// decodeEntitiesInternal and the bounded attribute-value path: the only
// difference between the two is the sink. A nested expansion's contributed size
// is measured as the sink's count() delta across the recursive call (equal to
// len(rep) in the old string-only code), so amplification accounting is
// unchanged.
func (pctx *parserCtx) decodeEntitiesToSink(ctx context.Context, s []byte, what SubstitutionType, depth int, sink entityDecodeSink) error {
	if depth > 40 {
		return errors.New("entity loop (depth > 40)")
	}

	for len(s) > 0 {
		if bytes.HasPrefix(s, []byte{'&', '#'}) {
			val, width, err := parseStringCharRef(s)
			if err != nil {
				return err
			}
			if err := sink.writeRune(ctx, val); err != nil {
				return err
			}
			s = s[width:]
		} else if s[0] == '&' && what&SubstituteRef == SubstituteRef {
			ent, width, err := pctx.parseStringEntityRef(ctx, s)
			if err != nil {
				return err
			}
			if ent == nil {
				if err := sink.write(ctx, s[:width]); err != nil {
					return err
				}
				s = s[width:]
				continue
			}
			if err := pctx.entityCheck(ent, 0); err != nil {
				return err
			}

			if ent.EntityType() == enum.InternalPredefinedEntity {
				if len(ent.Content()) == 0 {
					return errors.New("predefined entity has no content")
				}
				if err := sink.write(ctx, ent.Content()); err != nil {
					return err
				}
			} else if len(ent.Content()) != 0 {
				before := sink.count()
				if err := pctx.decodeEntitiesToSink(ctx, ent.Content(), what, depth+1, sink); err != nil {
					return err
				}
				if err := pctx.entityCheck(ent, sink.count()-before); err != nil {
					return err
				}
			} else {
				if err := sink.writeString(ctx, ent.Name()); err != nil {
					return err
				}
			}
			s = s[width:]
		} else if s[0] == '%' && what&SubstitutePERef == SubstitutePERef {
			ent, width, err := pctx.parseStringPEReference(ctx, s)
			if err != nil {
				return err
			}
			if ent == nil {
				// An undeclared parameter entity in a context with an external
				// subset (or after a prior PE reference) is a validity error,
				// not a fatal one: parseStringPEReference has already cleared
				// pctx.valid and returns a nil entity with no error. It is not
				// expanded. Skip it without dereferencing the nil entity,
				// mirroring the '&' branch above and expandEntityValueForRefCheck.
				// Still charge the reference against entity-expansion accounting
				// (entityCheck tolerates a nil ent) so an unresolved PE ref can't
				// be used to dodge the amplification/ceiling limits.
				if err := pctx.entityCheck(ent, width); err != nil {
					return err
				}
				s = s[width:]
				continue
			}
			if err := pctx.entityCheck(ent, width); err != nil {
				return err
			}
			peContent, err := pctx.parameterEntityReplacement(ctx, ent)
			if err != nil {
				return err
			}
			before := sink.count()
			if err := pctx.decodeEntitiesToSink(ctx, peContent, what, depth+1, sink); err != nil {
				return err
			}
			if err := pctx.entityCheck(ent, sink.count()-before); err != nil {
				return err
			}
			s = s[width:]
		} else {
			if err := sink.writeByte(ctx, s[0]); err != nil {
				return err
			}
			s = s[1:]
		}
	}
	return nil
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
	return scanEntityValueGeneralRefs(expanded, pctx.maxNameLength)
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
				// WFC: PEs in Internal Subset (XML §2.8). A parameter-entity
				// reference occurring WITHIN a markup declaration — here, inside
				// an EntityValue literal — is a fatal well-formedness error in
				// the internal subset; it is permitted only in the external
				// subset or within an external parameter entity
				// (effectivelyExternal, libxml2's PARSER_EXTERNAL gate in
				// xmlExpandPEsInEntityValue). The check fires only for a RESOLVED
				// PE ref (ent != nil), matching libxml2's early return on an
				// undeclared/malformed reference (W3C not-wf-sa-160/162,
				// ibm-not-wf-P29-ibm29n04, ibm-not-wf-P69-ibm69n06/07).
				if !pctx.effectivelyExternal() {
					return nil, ErrPEReferenceInInternalSubset
				}
				// Expand the PE replacement text. decodeEntitiesInternal
				// recursively substitutes nested parameter entities and resolves
				// character references to their literal characters, so a "&#38;"
				// brought in by the PE becomes a literal '&' that can combine
				// with surrounding text into a general reference. General
				// references (&Name;) in the replacement text are left intact for
				// the subsequent scan. An external PE's replacement text is loaded
				// on demand (parameterEntityReplacement) so the check sees the same
				// bytes regardless of reference order.
				peContent, err := pctx.parameterEntityReplacement(ctx, ent)
				if err != nil {
					return nil, err
				}
				rep, err := pctx.decodeEntitiesInternal(ctx, peContent, SubstitutePERef, depth+1)
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
func scanEntityValueGeneralRefs(s []byte, maxNameLength int) error {
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
		_, width, err := parseStringName(s[1:], maxNameLength)
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
	// The whole <!ENTITY ... > must start and stop in the same input: a closing
	// '>' supplied by a different parameter entity than the one that opened the
	// declaration (e.g. `<!ENTITY greet 'hi'%close;` with %close; -> ">") is a
	// boundary violation, the same fatal condition <!ELEMENT>/<!ATTLIST> enforce.
	startInput := pctx.currentInputID()

	// The mandatory "S" separators — and, in the external subset, a parameter
	// entity supplying part of the declaration (the entity value or an external
	// ID) — are consumed through skipBlanksPE, which leaves a "% " parameter-entity
	// DECLARATION marker for the isParameter check below (it only expands a genuine
	// "%name;" reference). Re-fetch the cursor after each skip since an expand/pop
	// changes the top input.
	adv, err := pctx.skipBlanksPE(ctx)
	if err != nil {
		return pctx.error(ctx, err)
	}
	if !adv {
		return pctx.error(ctx, ErrSpaceRequired)
	}

	cur = pctx.dtdRefetch(cur)
	isParameter := false
	if cur.Peek() == '%' {
		if err := cur.Advance(1); err != nil {
			return err
		}
		adv, err := pctx.skipBlanksPE(ctx)
		if err != nil {
			return pctx.error(ctx, err)
		}
		if !adv {
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

	adv, err = pctx.skipBlanksPE(ctx)
	if err != nil {
		return pctx.error(ctx, err)
	}
	if !adv {
		return pctx.error(ctx, ErrSpaceRequired)
	}
	cur = pctx.dtdRefetch(cur)

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
			// parseExternalID returns (systemURI, publicID, found). Mirror the
			// external general-entity path below: guard on the system URI (a SYSTEM
			// declaration carries no public ID, so guarding on the public ID would
			// drop every SYSTEM parameter entity), and pass publicID/systemID to
			// EntityDecl in that order.
			var found bool
			literal, uri, found, err = pctx.parseExternalID(ctx, true)
			if err != nil {
				// Preserve a resource-limit (ErrNodeContentTooLarge) or
				// parse-abort error verbatim; only an empty/missing literal
				// falls back to the generic ErrValueRequired.
				return pctx.preserveLimitOrAbort(ctx, err, ErrValueRequired)
			}
			// PEDef [74]: EntityValue | ExternalID — with neither an EntityValue
			// (handled above) nor an ExternalID present, the declaration is a fatal
			// well-formedness error.
			if !found {
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
			// Register the external parameter entity whenever an ExternalID was
			// present (found), including a valid empty SystemLiteral (`SYSTEM ""`);
			// gating on literal != "" would drop such a PE and leave
			// GetParameterEntity false.
			if s := pctx.sax; s != nil {
				switch err := s.EntityDecl(ctx, name, enum.ExternalParameterEntity, uri, literal, ""); err {
				case nil, sax.ErrHandlerUnspecified:
				default:
					return pctx.error(ctx, err)
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
			var found bool
			literal, uri, found, err = pctx.parseExternalID(ctx, true)
			if err != nil {
				// Preserve a resource-limit (ErrNodeContentTooLarge) or
				// parse-abort error verbatim; only an empty/missing literal
				// falls back to the generic ErrValueRequired.
				return pctx.preserveLimitOrAbort(ctx, err, ErrValueRequired)
			}
			// EntityDef [73]: EntityValue | (ExternalID NDataDecl?) — with neither
			// an EntityValue (handled above) nor an ExternalID present, the
			// declaration is a fatal well-formedness error (W3C o-p73fail4).
			if !found {
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

			cur = pctx.dtdRefetch(cur)
			// A '%' here can supply the following token (NDATA / '>') ONLY in the
			// external subset; in the internal subset a '%' where an "S" or '>' is
			// required stays an early ErrSpaceRequired (byte-identical to origin).
			if c := cur.Peek(); c != '>' && !isBlankByte(c) && (!pctx.external || c != '%') {
				return pctx.error(ctx, ErrSpaceRequired)
			}

			if _, err := pctx.skipBlanksPE(ctx); err != nil {
				return pctx.error(ctx, err)
			}
			cur = pctx.dtdRefetch(cur)
			if cur.ConsumeString("NDATA") {
				adv, err := pctx.skipBlanksPE(ctx)
				if err != nil {
					return pctx.error(ctx, err)
				}
				if !adv {
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

	if _, err := pctx.skipBlanksPE(ctx); err != nil {
		return pctx.error(ctx, err)
	}
	cur = pctx.dtdRefetch(cur)
	if cur.Peek() != '>' {
		return pctx.error(ctx, errors.New("entity not terminated"))
	}
	if pctx.currentInputID() != startInput {
		return pctx.error(ctx,
			fmt.Errorf("%w: entity declaration doesn't start and stop in the same entity", ErrEntityBoundary))
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
					// DTD-declaration bookkeeping (SetOrig); getEntity does not
					// flag the standalone WFC here (inSubset != 0), so any error
					// is a plain lookup miss and the entity (if any) is used.
					e, _ := pctx.getEntity(ctx, name)
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
// It does NOT touch newctx.doc, newctx.external, or the per-context amplification
// counters (sizeentcopy/inputSize/maxAmpl); those are handled by the caller
// because their lifecycle (document swap, external flag, write-back on return)
// differs between the two paths.
//
// It DOES carry the ebcdicConsumed pointer, because that is a SHARED live
// byte-counter over the underlying EBCDIC stream (not a per-context value): on
// the EBCDIC ParseReader path inputSize was seeded only from the bounded sniff
// prefix, so entityCheckLimits compares the amplification budget against this
// live consumed-byte count instead. A nested entity sub-parse must see the same
// pointer or an internal entity referenced from entity replacement text would be
// falsely rejected as amplification (its newctx.inputSize is the parent's prefix
// size while the real document bytes were already consumed at the top level). It
// is nil on every non-EBCDIC path.
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
	// Carry the node-content cap so a configured Parser.MaxNodeContentSize is
	// enforced on the CDATA/comment/PI/char-data runs of entity-expansion
	// sub-parses too, not just the top-level document.
	newctx.maxNodeContent = pctx.maxNodeContent
	// Carry both the element-depth limit and the current depth so nesting that
	// crosses the entity boundary keeps counting toward MaxDepth.
	newctx.maxElemDepth = pctx.maxElemDepth
	newctx.elemDepth = pctx.elemDepth
	// Carry the shared live EBCDIC consumed-byte counter so the amplification
	// guard inside a nested entity sub-parse compares against the real document
	// size, not the bounded sniff prefix that seeded newctx.inputSize. nil except
	// on the EBCDIC ParseReader path.
	newctx.ebcdicConsumed = pctx.ebcdicConsumed
	// Inherit the parent's security/resolution policy so any external reference
	// reached while expanding this replacement text honors the same FS sandbox,
	// catalog, and base URI as the top-level parse rather than falling back to
	// the permissive os.Open root.
	newctx.fsys = pctx.fsys
	newctx.catalog = pctx.catalog
	newctx.baseURI = pctx.baseURI
	// Carry the fixed top-level document base so a confined-FS retry inside a
	// nested entity sub-parse still relativizes against the document root, not the
	// nested resource's own (moved) baseURI.
	newctx.documentBaseURI = pctx.documentBaseURI
	// Carry the document-scope DTD state that gates the "Entity Declared" VC so a
	// nested entity-replacement sub-parse makes the SAME lenient/strict decision
	// as the top-level document: an undeclared general entity is a validity error
	// only when validating a standalone, fully-INTERNAL DTD (no external subset
	// or external parameter entity). Without this, a nested sub-context would
	// default hasExternalSubset/hasExternalPERef to false and over-reject an
	// undeclared entity in a document that in fact has an external subset/PE.
	newctx.standalone = pctx.standalone
	newctx.hasExternalSubset = pctx.hasExternalSubset
	newctx.hasPERefs = pctx.hasPERefs
	newctx.hasExternalPERef = pctx.hasExternalPERef
}

// parseExternalEntityPrivate loads and parses an external general entity. uri is
// the entity's RESOLVED URI (built from the declared system id against the base in
// EntityDecl); declaredSystemID is the entity's system id AS DECLARED, used only to
// gate the confined-FS base-relative retry (openExternalResource) on original
// relativeness — the resolved uri is already absolute and cannot be distinguished.
func (pctx *parserCtx) parseExternalEntityPrivate(ctx context.Context, uri, declaredSystemID, externalID string) (Node, error) {
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
		// Gate the confined-FS retry for this reference on the declared system id's
		// original relativeness, then clear it so a later ResolveEntity that does
		// not set it cannot inherit a stale eligibility.
		pctx.extRefRelative = systemIDRetryEligible(declaredSystemID)
		resolved, err := s.ResolveEntity(ctx, externalID, uri)
		pctx.extRefRelative = false
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

	// An external parsed general entity's replacement text MAY begin with a
	// TextDecl ('<?xml' VersionInfo? EncodingDecl S? '?>') — VersionInfo OPTIONAL,
	// EncodingDecl REQUIRED, NO StandaloneDecl (XML §4.3.1). Consume it and decode
	// the body per its declared encoding at the same shared chokepoint the external
	// DTD subset and external parameter entities use, so the nested parse is handed
	// post-TextDecl UTF-8 content. Without this the nested parseXMLDecl would reject
	// a version-less TextDecl for a missing version, and a leading '<?xml' left in
	// the decoded stream would be rejected by parseContent as a PI whose target may
	// not be "xml". A malformed TextDecl (e.g. a standalone pseudo-attribute, or a
	// version-only declaration) is rejected here by parseTextDecl.
	content, err = pctx.decodeExternalPEContent(ctx, uri, content)
	if err != nil {
		return nil, err
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

	// A leading TextDecl (and any declared encoding) has already been consumed and
	// the body decoded to UTF-8 by decodeExternalPEContent above, so the byte
	// stream here never begins with a '<?xml' declaration; detectEncoding /
	// switchEncoding still handle a BOM-only external entity carrying no TextDecl.
	if err := newctx.switchEncoding(); err != nil {
		return nil, err
	}

	newRoot := newctx.doc.CreateElement(pseudoRootName)
	newctx.pushNodeEntry(nodeEntry{local: pseudoRootName, qname: pseudoRootName, synthetic: true})
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
								if _, err := elem.SetAttributeNS("base", uri, newNamespace("xml", lexicon.NamespaceXML)); err == nil {
									// Mark this xml:base as parser-synthesized so DTD
									// validation does not treat it as an undeclared
									// attribute (an authored xml:base is never marked).
									if injected := elem.GetAttributeNodeNS("base", lexicon.NamespaceXML); injected != nil {
										injected.syntheticBase = true
									}
								}
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
	newctx.pushNodeEntry(nodeEntry{local: pseudoRootName, qname: pseudoRootName, synthetic: true})
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
	// The replacement text must be well balanced with respect to element nesting
	// (WFC: parsed entities must be well-formed; XML §4.3.2). parseContent stops
	// at a "</" that would close an element opened OUTSIDE this chunk (the
	// pseudo-root), leaving that end-tag — and anything after it — unconsumed. A
	// non-exhausted cursor here therefore means the entity opens with, or crosses,
	// an element boundary (e.g. "</foo><foo>"), which is a fatal error.
	if lc := newctx.getCursor(); lc != nil && !lc.Done() {
		return nil, newctx.error(innerCtx, ErrEntityNotWellBalanced)
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
