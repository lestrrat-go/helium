package helium

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/sax"
)

func (pctx *parserCtx) parseReference(ctx context.Context) error {
	cur := pctx.getCursor()
	if cur == nil {
		return pctx.error(ctx, errNoCursor)
	}
	if cur.Peek() != '&' {
		return pctx.error(ctx, ErrAmpersandRequired)
	}

	// A reference (character or general-entity) is content per XML production
	// [43]. Record it on the containing element so element-content validity can
	// reject a reference inside an element declared EMPTY (errata 2e E15a) even
	// when the reference expands to nothing. parseReference runs only from the
	// element-content loop (parseDocument), so pctx.elem is the element the
	// reference is content of; references in attribute values take a separate
	// path (decodeEntities) and never reach here, so an attribute char/entity
	// reference never marks its element.
	if pctx.elem != nil {
		pctx.elem.contentHasReference = true
	}

	if cur.PeekAt(1) == '#' {
		v, err := pctx.parseCharRef()
		if err != nil {
			return pctx.error(ctx, err)
		}
		var buf [utf8.UTFMax]byte
		l := utf8.EncodeRune(buf[:], v)
		b := buf[:l]
		if s := pctx.sax; s != nil {
			// Character-reference whitespace does not match the S nonterminal, so
			// it is not ignorable in element-only content (XML §3.2.1 / errata 2e
			// E15). Mark the delivery so the resulting Text node records its origin.
			pctx.charDataFromCharRef = true
			err := pctx.deliverCharacters(ctx, s.Characters, b)
			pctx.charDataFromCharRef = false
			if err != nil {
				return err
			}
		}
		return nil
	}

	ent, err := pctx.parseEntityRef(ctx)
	if err != nil {
		return pctx.error(ctx, err)
	}
	if ent == nil {
		return nil
	}

	wasChecked := ent.checked

	if ent.name == "" || ent.EntityType() == enum.InternalPredefinedEntity {
		if ent.content == "" {
			return nil
		}
		if s := pctx.sax; s != nil {
			if err := pctx.deliverCharacters(ctx, s.Characters, []byte(ent.content)); err != nil {
				return err
			}
		}
		return nil
	}

	if err := pctx.entityCheck(ent, len(ent.content)); err != nil {
		return pctx.error(ctx, err)
	}

	var parsedEnt Node
	if (wasChecked == 0 || (ent.firstChild == nil && pctx.options.IsSet(parseNoEnt))) && (ent.EntityType() != enum.ExternalGeneralParsedEntity || pctx.options.IsSet(parseNoEnt|parseDTDValid)) {
		sizeBefore := pctx.sizeentcopy

		if ent.EntityType() == enum.InternalGeneralEntity {
			parsedEnt, err = pctx.parseBalancedChunkInternal(ctx, ent.Content())
			switch err {
			case nil, errParseSucceeded:
			default:
				return err
			}
		} else if ent.EntityType() == enum.ExternalGeneralParsedEntity {
			parsedEnt, err = pctx.parseExternalEntityPrivate(ctx, ent.uri, ent.systemID, ent.externalID)
			switch err {
			case nil, errParseSucceeded:
			default:
				return err
			}
		} else {
			return errors.New("invalid entity type")
		}

		if ent.checked == 0 {
			ent.checked = 2
		}
		// Cache only the expansion bytes, never the per-reference fixed cost:
		// entityCheck adds entityFixedCost on top of expandedSize for every
		// reference. sizeBefore is captured after this reference's own
		// entityCheck (which already paid the fixed cost), and external content
		// is charged via entityCheckBytes (no fixed cost), so the delta here is
		// pure bytes.
		ent.expandedSize = pctx.sizeentcopy - sizeBefore + int64(len(ent.content))
		ent.MarkChecked()

		if parsedEnt != nil && ent.firstChild == nil {
			for n := parsedEnt; n != nil; {
				next := n.NextSibling()
				ndn := n.baseDocNode()
				ndn.next = nil
				ndn.prev = nil
				ndn.parent = nil
				n.(MutableNode).SetTreeDoc(pctx.doc) //nolint:forcetypeassert
				_ = ent.AddChild(n)
				n = next
			}
		}
	}

	if ent.firstChild == nil {
		if wasChecked != 0 {
			if ent.EntityType() == enum.InternalGeneralEntity {
				parsedEnt, err = pctx.parseBalancedChunkInternal(ctx, ent.Content())
				_ = parsedEnt
				switch err {
				case nil, errParseSucceeded:
				default:
					return err
				}
			} else if ent.EntityType() == enum.ExternalGeneralParsedEntity {
				parsedEnt, err = pctx.parseExternalEntityPrivate(ctx, ent.uri, ent.systemID, ent.externalID)
				_ = parsedEnt
				switch err {
				case nil, errParseSucceeded:
				default:
					return err
				}
			} else {
				return errors.New("invalid entity type")
			}
		}
		if s := pctx.sax; s != nil && !pctx.replaceEntities {
			switch err := s.Reference(ctx, ent.name); err {
			case nil, sax.ErrHandlerUnspecified:
			default:
				return err
			}
		}
		return nil
	}

	if s := pctx.sax; s != nil && !pctx.replaceEntities {
		switch err := s.Reference(ctx, ent.name); err {
		case nil, sax.ErrHandlerUnspecified:
		default:
			return err
		}
		return nil
	}

	if pctx.replaceEntities {
		savedEntityURI := pctx.currentEntityURI
		if ent.EntityType() == enum.ExternalGeneralParsedEntity && ent.uri != "" {
			pctx.currentEntityURI = ent.uri
		}
		for n := ent.firstChild; n != nil; n = n.NextSibling() {
			if err := pctx.replayEntityNode(ctx, n); err != nil {
				pctx.currentEntityURI = savedEntityURI
				return err
			}
		}
		pctx.currentEntityURI = savedEntityURI
	}

	return nil
}

func (pctx *parserCtx) replayEntityNode(ctx context.Context, n Node) error {
	if n == nil || pctx.sax == nil {
		return nil
	}

	switch v := n.(type) {
	case *Element:
		// Replaying a cached entity subtree must honor MaxDepth exactly like
		// the live parse path (parseElement). Without this, an entity first
		// expanded at a shallow depth caches its subtree, and a later
		// reference under a deeper element replays that subtree without any
		// depth accounting — bypassing the configured limit.
		pctx.elemDepth++
		defer func() { pctx.elemDepth-- }()

		if pctx.maxElemDepth > 0 && pctx.elemDepth > pctx.maxElemDepth {
			return pctx.error(ctx, fmt.Errorf("xml: exceeded max depth"))
		}

		namespaces := make([]sax.Namespace, 0, len(v.Namespaces()))
		for _, ns := range v.Namespaces() {
			namespaces = append(namespaces, ns)
		}

		var attrs []sax.Attribute
		for attr := v.properties; attr != nil; attr = attr.NextAttribute() {
			attrs = append(attrs, attr)
		}

		switch err := pctx.sax.StartElementNS(ctx, v.LocalName(), v.Prefix(), v.URI(), namespaces, attrs); err {
		case nil, sax.ErrHandlerUnspecified:
		default:
			return err
		}

		for child := range Children(v) {
			if err := pctx.replayEntityNode(ctx, child); err != nil {
				return err
			}
		}

		switch err := pctx.sax.EndElementNS(ctx, v.LocalName(), v.Prefix(), v.URI()); err {
		case nil, sax.ErrHandlerUnspecified:
			return nil
		default:
			return err
		}
	case *Text:
		// Propagate the cached entity Text's char-reference origin so a general
		// entity whose replacement text is a character reference (E15h) yields a
		// non-ignorable whitespace Text node in the content, matching a direct
		// character reference (E15g).
		if v.fromCharRef {
			pctx.charDataFromCharRef = true
			err := pctx.deliverCharacters(ctx, pctx.sax.Characters, v.Content())
			pctx.charDataFromCharRef = false
			return err
		}
		return pctx.deliverCharacters(ctx, pctx.sax.Characters, v.Content())
	case *CDATASection:
		switch err := pctx.sax.CDataBlock(ctx, v.Content()); err {
		case nil:
			return nil
		case sax.ErrHandlerUnspecified:
			return pctx.deliverCharacters(ctx, pctx.sax.Characters, v.Content())
		default:
			return err
		}
	case *Comment:
		switch err := pctx.sax.Comment(ctx, v.Content()); err {
		case nil, sax.ErrHandlerUnspecified:
			return nil
		default:
			return err
		}
	case *ProcessingInstruction:
		switch err := pctx.sax.ProcessingInstruction(ctx, v.Name(), string(v.Content())); err {
		case nil, sax.ErrHandlerUnspecified:
			return nil
		default:
			return err
		}
	default:
		return nil
	}
}

func accumulateDecimalCharRef(val int32, c rune) (int32, error) {
	if c >= '0' && c <= '9' {
		val = val*10 + (c - '0')
	} else {
		return 0, errors.New("invalid decimal CharRef")
	}
	return val, nil
}

func accumulateHexCharRef(val int32, c rune) (int32, error) {
	if c >= '0' && c <= '9' {
		val = val*16 + (c - '0')
	} else if c >= 'a' && c <= 'f' {
		val = val*16 + (c - 'a') + 10
	} else if c >= 'A' && c <= 'F' {
		val = val*16 + (c - 'A') + 10
	} else {
		return 0, errors.New("invalid hex CharRef")
	}
	return val, nil
}

// parseStringCharRef parses a character reference held in stored entity text.
// XML 1.1 permits the C0/C1 control values through character references, while
// XML 1.0 rejects them.
func parseStringCharRef(s []byte, xml11 bool) (r rune, width int, err error) {
	var val int32
	r = utf8.RuneError
	width = 0
	if !bytes.HasPrefix(s, []byte{'&', '#'}) {
		err = errors.New("ampersand (&) was required")
		return
	}

	width += 2
	s = s[2:]

	if len(s) == 0 {
		err = errors.New("malformed CharRef")
		width = 0
		return
	}

	var accumulator func(int32, rune) (int32, error)
	if s[0] == 'x' {
		s = s[1:]
		width++
		accumulator = accumulateHexCharRef
		if len(s) == 0 {
			err = errors.New("malformed CharRef")
			width = 0
			return
		}
	} else {
		accumulator = accumulateDecimalCharRef
	}

	digits := 0
	for len(s) > 0 && s[0] != ';' {
		c := s[0]
		val, err = accumulator(val, rune(c))
		if err != nil {
			width = 0
			return
		}
		if val > unicode.MaxRune {
			err = errors.New("CharRef out of range")
			width = 0
			return
		}

		digits++
		s = s[1:]
		width++
	}

	if digits == 0 {
		err = errors.New("malformed CharRef")
		width = 0
		return
	}

	if len(s) == 0 || s[0] != ';' {
		err = errors.New("malformed CharRef")
		width = 0
		return
	}
	s = s[1:]
	_ = s
	width++

	r = val
	charOK := isXMLCharValue(uint32(val))
	if !charOK && xml11 {
		charOK = isXML11CharValue(uint32(val))
	}
	if !charOK {
		return utf8.RuneError, 0, fmt.Errorf("invalid XML char value %d", val)
	}
	return
}

// parseStringName scans an XML Name from the front of s. maxNameLength bounds
// the name's byte length (0 = no limit) so entity/parameter-entity reference
// names in stored entity values are held to the same MaxNameLength cap as names
// parsed from the document stream.
func parseStringName(s []byte, maxNameLength int) (string, int, error) {
	i := 0
	r, w := utf8.DecodeRune(s)
	if r == utf8.RuneError {
		return "", 0, errors.New("rune decode failed")
	}

	if !isNameStartChar(r) {
		return "", 0, errors.New("invalid name start char")
	}

	out := bufferPool.Get()
	defer releaseBuffer(out)

	out.WriteRune(r)
	i += w
	s = s[w:]

	for {
		r, w = utf8.DecodeRune(s)
		if r == utf8.RuneError {
			return "", 0, errors.New("rune decode failed")
		}

		if !isNameChar(r) {
			break
		}
		out.WriteRune(r)
		i += w
		s = s[w:]
	}

	if maxNameLength > 0 && out.Len() > maxNameLength {
		return "", 0, ErrNameTooLong
	}
	return out.String(), i, nil
}

// getEntity resolves a general entity by name against the document entity
// table, mirroring libxml2 xmlSAX2GetEntity. In a standalone="yes" document a
// reference (in the document body) to a general entity that is declared ONLY in
// the external subset is a fatal well-formedness error — WFC: Entity Declared
// (XML §4.1) as constrained by the Standalone Document Declaration (§2.9): under
// standalone="yes" all consumed declarations must be visible in the internal
// subset. Document.GetEntity hides external declarations while standalone is
// set, so such an entity is found only on the standalone-disabled retry; that
// retry succeeding is exactly the violation (libxml2's XML_ERR_NOT_STANDALONE).
// A returned non-nil entity with a non-nil error is that violation (the entity
// is still returned, matching libxml2, so the caller can surface a precise
// error); a nil entity with a nil error is a genuinely undeclared reference the
// caller must route through the Entity Declared WFC (handleUndeclaredEntity).
func (pctx *parserCtx) getEntity(ctx context.Context, name string) (*Entity, error) {
	if pctx.inSubset == 0 {
		if ret, err := resolvePredefinedEntity(name); err == nil {
			return ret, nil
		}
	}

	var ret *Entity
	var ok bool
	if pctx.doc == nil {
		return nil, ErrEntityNotFound
	} else if pctx.doc.standalone != StandaloneExplicitYes {
		ret, _ = pctx.doc.GetEntity(name)
	} else {
		if pctx.inSubset == 2 {
			pctx.doc.standalone = 0
			ret, _ = pctx.doc.GetEntity(name)
			pctx.doc.standalone = 1
		} else {
			ret, ok = pctx.doc.GetEntity(name)
			if !ok {
				pctx.doc.standalone = 0
				ret, ok = pctx.doc.GetEntity(name)
				pctx.doc.standalone = 1
				if !ok {
					// Genuinely undeclared: let the caller apply the
					// Entity Declared WFC via its nil-entity handling.
					return nil, nil //nolint:nilnil
				}
				// Declared only in the external subset while standalone="yes".
				// A reference in the document body (inSubset == 0) is a fatal
				// WFC violation; references inside the DTD (attribute-list
				// bookkeeping) are not flagged here.
				if pctx.inSubset == 0 {
					return ret, pctx.error(ctx, ErrNotStandalone)
				}
			}
		}
	}
	return ret, nil
}

// lookupGeneralEntity resolves a general entity by name for the attribute-value
// WFC walk, mirroring libxml2 xmlLookupGeneralEntity: a predefined entity first,
// then the SAX getEntity callback, then the document entity table. Resolving
// SAX-first is what lets a pure SAX-event parse (a custom handler replacing the
// tree builder, with no document being built) surface an indirect
// external/unparsed reference — the document table is empty there, but the
// handler's GetEntity still answers. When inAttr, a direct resolution to an
// unparsed entity yields attrWFCUnparsed and to an external parsed entity yields
// attrWFCExternal (the "No External Entity References"/"Parsed Entity" WFCs). An
// undefined entity is reported as (nil, attrWFCNone, nil); the caller applies
// the "Entity Declared" WFC via handleUndeclaredEntity.
func (pctx *parserCtx) lookupGeneralEntity(ctx context.Context, name string, inAttr bool) (*Entity, attrEntityWFC, error) {
	if ent, err := resolvePredefinedEntity(name); err == nil {
		return ent, attrWFCNone, nil
	}

	var ent *Entity
	if s := pctx.sax; s != nil {
		loaded, _ := s.GetEntity(ctx, name)
		if !isNilEntity(loaded) {
			typed, ok := loaded.(*Entity)
			if !ok {
				return nil, attrWFCNone, pctx.error(ctx, fmt.Errorf("SAX GetEntity returned unsupported entity type %T for entity '%s'", loaded, name))
			}
			ent = typed
		}
	}
	if ent == nil {
		var gerr error
		ent, gerr = pctx.getEntity(ctx, name)
		// A non-nil entity with a non-nil error is the standalone WFC violation
		// (declared only in the external subset under standalone="yes"); surface
		// it. A nil entity is undeclared — handled below via the caller's
		// Entity Declared WFC, so its (non-fatal) error is discarded here.
		if gerr != nil && ent != nil {
			return nil, attrWFCNone, gerr
		}
	}
	if ent == nil {
		return nil, attrWFCNone, nil
	}

	switch ent.entityType {
	case enum.ExternalGeneralUnparsedEntity:
		return ent, attrWFCUnparsed, nil
	case enum.ExternalGeneralParsedEntity:
		if inAttr {
			return ent, attrWFCExternal, nil
		}
	}
	return ent, attrWFCNone, nil
}

// undeclaredEntityValidityError promotes an undeclared general-entity reference —
// one that resolved to no declaration and is NOT the fatal WFC case — to the
// "Entity Declared" VALIDITY error when validating a FULLY-INTERNAL DTD: no
// external subset and no external parameter entity (`parseDTDValid` set,
// `!hasExternalSubset && !hasExternalPERef`). An external subset or external PE
// may resolve incompletely (e.g. an empty/unreachable external PE), so helium
// stays lenient there rather than risk rejecting a valid document. It returns nil
// when the reference should stay a non-fatal warning. The VC "Entity Declared"
// applies to ALL general entity references alike — in element content AND in
// attribute values (W3C rmt-e3e-13; matches libxml2's xmlValidityError).
func (pctx *parserCtx) undeclaredEntityValidityError(ctx context.Context, name string) error {
	if !pctx.options.IsSet(parseDTDValid) || pctx.hasExternalSubset || pctx.hasExternalPERef {
		return nil
	}
	return pctx.error(ctx, fmt.Errorf("%w '%s'", ErrUndeclaredEntity, name))
}

// handleUndeclaredEntity applies the "Entity Declared" constraint for a
// general-entity reference that resolved to no declaration, mirroring libxml2
// xmlHandleUndeclaredEntity. It returns a fatal error when the missing
// declaration is a well-formedness violation (standalone='yes', or no external
// subset and no parameter-entity references, so all declarations must be
// visible), or — when validating a fully-internal DTD — the "Entity Declared"
// validity error (undeclaredEntityValidityError). Otherwise it emits a warning,
// marks the parse invalid, and returns nil so the caller can continue (a nested
// entity may still be declared later in an external subset).
func (pctx *parserCtx) handleUndeclaredEntity(ctx context.Context, name string) error {
	if pctx.standalone == StandaloneExplicitYes || (!pctx.hasExternalSubset && !pctx.hasPERefs) {
		return pctx.error(ctx, fmt.Errorf("Entity '%s' not defined", name))
	}
	if err := pctx.undeclaredEntityValidityError(ctx, name); err != nil {
		return err
	}
	if err := pctx.warning(ctx, "Entity '%s' not defined", name); err != nil {
		return err
	}
	pctx.valid = false
	return nil
}

// isNilEntity reports whether e is nil, including a non-nil sax.Entity interface
// value that wraps a nil *Entity. getEntity returns a concrete (*Entity)(nil)
// for an undefined entity; assigned to a sax.Entity interface that becomes a
// typed nil that a plain `== nil` check misses, so any method call on it panics.
func isNilEntity(e sax.Entity) bool {
	if e == nil {
		return true
	}
	if ce, ok := e.(*Entity); ok {
		return ce == nil
	}
	return false
}

func (pctx *parserCtx) parseStringEntityRef(ctx context.Context, s []byte) (sax.Entity, int, error) {
	if len(s) == 0 || s[0] != '&' {
		return nil, 0, errors.New("invalid entity ref")
	}

	i := 1
	name, width, err := parseStringName(s[1:], pctx.maxNameLength)
	if err != nil {
		return nil, 0, errors.New("failed to parse name")
	}
	s = s[width+1:]
	i += width

	if s[0] != ';' {
		return nil, 0, ErrSemicolonRequired
	}
	s = s[1:]
	_ = s
	i++

	if predef, perr := resolvePredefinedEntity(name); perr == nil {
		return predef, i, nil
	}

	var loadedEnt sax.Entity
	if h := pctx.sax; h != nil {
		loadedEnt, err = h.GetEntity(ctx, name)
		if err != nil {
			if pctx.wellFormed {
				var ent *Entity
				ent, err = pctx.getEntity(ctx, name)
				// getEntity reports the standalone WFC violation as a non-nil
				// entity plus a non-nil error; an undeclared entity is (nil,nil)
				// and is handled by the isNilEntity branch below.
				if err != nil {
					return nil, 0, err
				}
				loadedEnt = ent
			}
		}
	}
	// getEntity returns a concrete (*Entity)(nil) for an undefined entity, which
	// becomes a non-nil sax.Entity interface holding a nil pointer when assigned
	// above. A plain `== nil` check misses that typed nil and the EntityType()
	// call below would panic (e.g. an internal entity whose replacement text
	// references an undefined entity, referenced from an attribute value), so
	// detect the typed nil too and treat it as "not defined".
	if isNilEntity(loadedEnt) {
		if pctx.standalone == StandaloneExplicitYes || (!pctx.hasExternalSubset && !pctx.hasPERefs) {
			return nil, 0, fmt.Errorf("entity '%s' not defined", name)
		}
		// Same "Entity Declared" VC promotion as the content path: a fully-internal
		// DTD referencing an undeclared entity from an attribute value is a validity
		// error when validating (W3C rmt-e3e-13 applies to attribute values too).
		if err := pctx.undeclaredEntityValidityError(ctx, name); err != nil {
			return nil, 0, err
		}
		if err := pctx.warning(ctx, "Entity '%s' not defined", name); err != nil {
			return nil, 0, err
		}
		return nil, i, nil
	}

	if loadedEnt.EntityType() == enum.ExternalGeneralUnparsedEntity {
		return nil, 0, fmt.Errorf("entity reference to unparsed entity '%s'", name)
	}

	if pctx.instate == psAttributeValue && loadedEnt.EntityType() == enum.ExternalGeneralParsedEntity {
		return nil, 0, fmt.Errorf("attribute references external entity '%s'", name)
	}

	// An internal general entity whose replacement text contains a literal '<'
	// is a WFC violation ("No < in Attribute Values") when referenced from an
	// attribute value. Predefined entities (&lt; etc.) resolve earlier and are
	// legal, so gate on non-predefined, mirroring the cursor path in
	// parseEntityRef.
	if pctx.instate == psAttributeValue && len(loadedEnt.Content()) > 0 && loadedEnt.EntityType() != enum.InternalPredefinedEntity && bytes.IndexByte(loadedEnt.Content(), '<') > -1 {
		return nil, 0, fmt.Errorf("'<' in entity '%s' is not allowed in attribute values", name)
	}

	switch loadedEnt.EntityType() {
	case enum.InternalParameterEntity, enum.ExternalParameterEntity:
		return nil, 0, fmt.Errorf("attempt to reference the parameter entity '%s'", name)
	}

	return loadedEnt, i, nil
}

func (pctx *parserCtx) parseStringPEReference(ctx context.Context, s []byte) (sax.Entity, int, error) {
	if len(s) == 0 || s[0] != '%' {
		return nil, 0, errors.New("invalid PEreference")
	}

	i := 1
	name, width, err := parseStringName(s[1:], pctx.maxNameLength)
	if err != nil {
		return nil, 0, err
	}
	s = s[width+1:]
	i += width

	if s[0] != ';' {
		return nil, 0, ErrSemicolonRequired
	}
	s = s[1:]
	_ = s
	i++

	var loadedEnt sax.Entity
	if h := pctx.sax; h != nil {
		loadedEnt, err = h.GetParameterEntity(ctx, name)
		if err != nil {
			return nil, 0, err
		}
	}

	if loadedEnt == nil {
		if pctx.standalone == StandaloneExplicitYes || (!pctx.hasExternalSubset && !pctx.hasPERefs) {
			return nil, 0, fmt.Errorf("not found: PE rerefence '%%%s'", name)
		}
		pctx.valid = false
	} else {
		switch loadedEnt.EntityType() {
		case enum.InternalParameterEntity, enum.ExternalParameterEntity:
		default:
			return nil, 0, fmt.Errorf("not a parmeter entity: %%%s", name)
		}
	}
	pctx.hasPERefs = true

	return loadedEnt, i, nil
}

func (ctx *parserCtx) parseCharRef() (r rune, err error) {
	r = utf8.RuneError

	var val int32
	cur := ctx.getCursor()
	if cur == nil {
		err = errNoCursor
		return
	}
	if cur.ConsumeString("&#x") {
		var digits int
		for c := cur.Peek(); !cur.Done() && c != ';'; c = cur.Peek() {
			if c >= '0' && c <= '9' {
				val = val*16 + int32(c-'0')
			} else if c >= 'a' && c <= 'f' {
				val = val*16 + int32(c-'a') + 10
			} else if c >= 'A' && c <= 'F' {
				val = val*16 + int32(c-'A') + 10
			} else {
				err = errors.New("invalid hex CharRef")
				return
			}
			digits++
			// Stop accumulating once the value is out of range so an
			// oversized reference cannot wrap int32 into a valid-looking rune.
			if val > unicode.MaxRune {
				err = ErrInvalidChar
				return
			}
			if err := cur.Advance(1); err != nil {
				return utf8.RuneError, err
			}
		}
		if digits == 0 {
			err = errors.New("invalid hex CharRef: missing digits")
			return
		}
		if cur.Peek() != ';' {
			err = ErrSemicolonRequired
			return
		}
		if err := cur.Advance(1); err != nil {
			return utf8.RuneError, err
		}
	} else if cur.ConsumeString("&#") {
		var digits int
		for !cur.Done() && cur.Peek() != ';' {
			c := cur.Peek()
			if c >= '0' && c <= '9' {
				val = val*10 + int32(c-'0')
			} else {
				err = errors.New("invalid decimal CharRef")
				return
			}
			digits++
			// Stop accumulating once the value is out of range so an
			// oversized reference cannot wrap int32 into a valid-looking rune.
			if val > unicode.MaxRune {
				err = ErrInvalidChar
				return
			}
			if err := cur.Advance(1); err != nil {
				return utf8.RuneError, err
			}
		}
		if digits == 0 {
			err = errors.New("invalid decimal CharRef: missing digits")
			return
		}
		if cur.Peek() != ';' {
			err = ErrSemicolonRequired
			return
		}
		if err := cur.Advance(1); err != nil {
			return utf8.RuneError, err
		}
	} else {
		err = errors.New("invalid char ref")
		return
	}

	charOK := isXMLCharValue(uint32(val))
	if !charOK && ctx.isXML11() {
		// XML 1.1 permits character references to the C0/C1 control
		// characters the 1.0 Char production forbids (all but U+0000).
		charOK = isXML11CharValue(uint32(val))
	}
	if charOK && val <= unicode.MaxRune {
		r = val
		return
	}

	err = ErrInvalidChar
	return
}

func (pctx *parserCtx) parseEntityRef(ctx context.Context) (ent *Entity, err error) {
	cur := pctx.getCursor()
	if cur == nil {
		err = pctx.error(ctx, errNoCursor)
		return
	}
	if cur.Peek() != '&' {
		err = pctx.error(ctx, ErrAmpersandRequired)
		return
	}
	if err = cur.Advance(1); err != nil {
		return
	}

	name, err := pctx.parseName(ctx)
	if err != nil {
		err = pctx.error(ctx, ErrNameRequired)
		return
	}

	if cur.Peek() != ';' {
		err = pctx.error(ctx, ErrSemicolonRequired)
		return
	}
	if err = cur.Advance(1); err != nil {
		return
	}

	if ent, err = resolvePredefinedEntity(name); err == nil {
		return ent, nil
	}

	if s := pctx.sax; s != nil {
		// A non-nil error here is advisory (e.g. "entity not found"): we fall
		// through to the undeclared-entity handling below, matching libxml2's
		// getEntity-returns-NULL behavior. Only a non-nil returned entity is
		// consumed, and it must be a *Entity. A foreign sax.Entity
		// implementation yields a clear error rather than a forced-cast panic.
		loadedEnt, _ := s.GetEntity(ctx, name)
		if loadedEnt != nil {
			typed, ok := loadedEnt.(*Entity)
			if !ok {
				return nil, pctx.error(ctx, fmt.Errorf("SAX GetEntity returned unsupported entity type %T for entity '%s'", loadedEnt, name))
			}
			// Fall through to the well-formedness checks below rather than
			// returning the SAX-resolved entity directly: a direct reference to
			// an external/unparsed/parameter entity (or an internal entity whose
			// replacement text contains '<') from an attribute value is a WFC
			// violation regardless of which resolver produced the entity.
			ent = typed
		} else {
			var gerr error
			ent, gerr = pctx.getEntity(ctx, name)
			// A resolved entity plus a non-nil error is the standalone WFC
			// violation (external-only declaration under standalone="yes");
			// propagate it. A nil entity falls through to the undeclared-entity
			// handling below.
			if gerr != nil && ent != nil {
				return nil, gerr
			}
		}
	}

	if ent == nil {
		if pctx.standalone == StandaloneExplicitYes || (!pctx.hasExternalSubset && !pctx.hasPERefs) {
			return nil, pctx.error(ctx, ErrUndeclaredEntity)
		}
		// A parameter-entity reference or external subset downgrades an undeclared
		// general entity from a fatal WFC to the "Entity Declared" VALIDITY
		// constraint; when validating a fully-internal DTD it is still reported
		// (undeclaredEntityValidityError).
		if err := pctx.undeclaredEntityValidityError(ctx, name); err != nil {
			return nil, err
		}
		if err := pctx.warning(ctx, "Entity '%s' not defined", name); err != nil {
			return nil, err
		}
		if pctx.inSubset == 0 && pctx.instate != psAttributeValue {
			if s := pctx.sax; s != nil {
				switch err := s.Reference(ctx, name); err {
				case nil, sax.ErrHandlerUnspecified:
				default:
					return nil, pctx.error(ctx, err)
				}
			}
		}
		if err := pctx.entityCheck(ent, 0); err != nil {
			return nil, pctx.error(ctx, err)
		}
		pctx.valid = false
		return nil, nil //nolint:nilnil
	} else if ent.entityType == enum.ExternalGeneralUnparsedEntity {
		return nil, pctx.error(ctx, errors.New("entity reference to unparsed entity"))
	} else if pctx.instate == psAttributeValue && ent.entityType == enum.ExternalGeneralParsedEntity {
		return nil, pctx.error(ctx, errors.New("attribute references external entity"))
	} else if pctx.instate == psAttributeValue && ent.entityType != enum.InternalPredefinedEntity {
		if (ent.checked&1 == 1 || ent.checked == 0) && ent.content != "" && strings.IndexByte(ent.content, '<') > -1 {
			return nil, pctx.error(ctx, errors.New("'<' in entity is not allowed in attribute values"))
		}
	} else {
		switch ent.entityType {
		case enum.InternalParameterEntity, enum.ExternalParameterEntity:
			return nil, pctx.error(ctx, errors.New("attempt to reference the parameter entity"))
		}
	}

	return ent, nil
}

func saturatedAdd(a, b int64) int64 {
	if a > math.MaxInt64-b {
		return math.MaxInt64
	}
	return a + b
}

func (ctx *parserCtx) entityCheck(ent sax.Entity, size int) error {
	// Account expanded bytes even when the ratio check is disabled
	// (maxAmpl=0 via MaxEntityAmplification(-1)) so the absolute ceiling
	// below can catch unbounded growth.
	if e, ok := ent.(*Entity); ok && e != nil && e.Checked() {
		ctx.sizeentcopy = saturatedAdd(ctx.sizeentcopy, e.expandedSize)
		ctx.sizeentcopy = saturatedAdd(ctx.sizeentcopy, entityFixedCost)
	} else {
		ctx.sizeentcopy = saturatedAdd(ctx.sizeentcopy, int64(size))
		ctx.sizeentcopy = saturatedAdd(ctx.sizeentcopy, entityFixedCost)
	}

	return ctx.entityCheckLimits()
}

// entityCheckBytes charges raw expansion bytes to the amplification counters
// WITHOUT adding entityFixedCost. The per-reference fixed cost is charged once
// in parseReference (via entityCheck); external content read by
// parseExternalEntityPrivate must not pay it a second time.
func (ctx *parserCtx) entityCheckBytes(size int) error {
	ctx.sizeentcopy = saturatedAdd(ctx.sizeentcopy, int64(size))
	return ctx.entityCheckLimits()
}

func (ctx *parserCtx) entityCheckLimits() error {
	// Absolute ceiling: enforced even when MaxEntityAmplification(-1)
	// disables the amplification-ratio check. Disabling the ratio check
	// asserts that legitimate large entities are OK, not that unbounded
	// billion-laughs expansion is OK.
	if ctx.sizeentcopy > entityHardCeiling {
		// The "maximum entity expansion size" prefix matches the historical
		// error text; substring callers (tests) keep matching. The
		// (current, limit) suffix surfaces the actual numbers so a
		// production diagnosis doesn't require code spelunking.
		return fmt.Errorf("maximum entity expansion size exceeded (%d > %d)",
			ctx.sizeentcopy, entityHardCeiling)
	}

	if ctx.maxAmpl == 0 {
		return nil
	}

	if ctx.sizeentcopy > entityAllowedExpansion {
		consumed := ctx.inputSize
		// On the EBCDIC streaming path inputSize was seeded from the bounded
		// sniff prefix (all rawInput holds there), not the real document size.
		// Use the larger live consumed-byte count so a legitimate large EBCDIC
		// document is not falsely rejected; the count never exceeds the source's
		// true length.
		if ctx.ebcdicConsumed != nil && ctx.ebcdicConsumed.n > consumed {
			consumed = ctx.ebcdicConsumed.n
		}
		if consumed == 0 {
			consumed = 1
		}
		if ctx.sizeentcopy/int64(ctx.maxAmpl) > consumed {
			return errors.New("maximum entity amplification factor exceeded")
		}
	}

	return nil
}

func (pctx *parserCtx) handlePEReference(ctx context.Context) error {
	cur := pctx.getCursor()
	if cur == nil {
		return pctx.error(ctx, errNoCursor)
	}
	if cur.Peek() != '%' {
		return nil
	}

	// A '%' immediately followed by whitespace (or end of input) is the
	// parameter-entity DECLARATION marker of `<!ENTITY % name ...>`, never a PE
	// reference — a reference is `%name;` with a NameStartChar right after '%'.
	// Leave the marker for the declaration parser in EVERY state. Without this,
	// the FIRST declaration of an external subset being a PE declaration reaches
	// skipBlanks (called from parseEntityDecl after `<!ENTITY`) with instate not
	// yet psDTD — parseMarkupDecl sets psDTD only AFTER a declaration — so the
	// psDTD-branch guard below does not apply and the marker is mis-parsed as a
	// reference, yielding a spurious "space required" on that first declaration.
	if c := cur.PeekAt(1); isBlankByte(c) || c == 0 {
		return nil
	}

	switch pctx.instate {
	case psCDATA, psComment, psStartTag, psEndTag, psEntityDecl, psContent, psAttributeValue, psPI, psSystemLiteral, psPublicLiteral, psEntityValue, psIgnore:
		return nil
	case psEOF:
		return errors.New("handlePEReference: parameter entity at EOF")
	case psPrologue, psStart, psMisc:
		return errors.New("handlePEReference: parameter entity in prologue")
	case psEpilogue:
		return errors.New("handlePEReference: parameter entity in epilogue")
	case psDTD:
		if !pctx.external || pctx.inputTab.Len() == 1 {
			return nil
		}
	}

	if err := cur.Advance(1); err != nil {
		return err
	}

	name, err := pctx.parseName(ctx)
	if err != nil {
		return err
	}

	if cur.Peek() != ';' {
		return ErrSemicolonRequired
	}

	if err := cur.Advance(1); err != nil {
		return err
	}

	var entity sax.Entity
	if s := pctx.sax; s != nil {
		entity, _ = s.GetParameterEntity(ctx, name)
	}

	if pctx.instate == psEOF {
		return nil
	}

	if entity == nil {
		if pctx.standalone == StandaloneExplicitYes || (!pctx.hasExternalSubset && !pctx.hasPERefs) {
			return fmt.Errorf("undeclared entity: PEReference: %%%s; not found", name)
		}
		if err := pctx.warning(ctx, "PEReference: %%%s; not found\n", name); err != nil {
			return err
		}
		pctx.valid = false
		if err := pctx.entityCheck(nil, 0); err != nil {
			return pctx.error(ctx, err)
		}
	} else {
		switch entity.EntityType() {
		case enum.InternalParameterEntity, enum.ExternalParameterEntity:
		default:
			return fmt.Errorf("entity is a parameter: PEReference: %%%s; is not a parameter entity", name)
		}
	}
	return nil
}
