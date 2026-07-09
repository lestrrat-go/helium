package helium

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/internal/strcursor"
	"github.com/lestrrat-go/helium/sax"
)

// literalScanChunk bounds how many bytes a single quoted-literal scan peeks
// ahead before advancing the cursor, mirroring blankScanChunk. Peeking an
// ever-growing offset (the old behavior) forces the cursor to buffer the whole
// literal up front, so an attacker-controlled unbounded entity-value or
// SYSTEM/PUBLIC literal — all reachable from an internal DTD, which parses by
// default — would grow the cursor buffer without bound before any per-node cap
// fires, including on an EBCDIC ParseReader stream that streams through the
// normal cursor pipeline. Scanning in fixed-size chunks and advancing as we go
// keeps the cursor buffer bounded to this size.
const literalScanChunk = 4096

// scanQuotedLiteral scans a quoted literal value (entity value, SYSTEM literal,
// or PUBLIC/pubid literal) up to but not including the closing quote qch and
// returns the decoded value. It advances the cursor in bounded chunks, checks
// the context between chunks, and enforces the node-content cap
// (pctx.maxNodeContent, the same cap CDATA/comment/PI/char-data/attribute runs
// use) so neither the output buffer nor the cursor's internal PeekAt buffer
// grows past the cap. An over-cap literal fails closed with
// ErrNodeContentTooLarge. A byte that is not a permitted literal character ends
// the scan WITHOUT error (the caller validates the closing quote), matching the
// prior unbounded scanners. When pubid is true only the ASCII PubidChar subset
// is accepted; otherwise the full XML Char production is accepted (multi-byte
// runes decoded via decodeRuneAt). HasByteAt distinguishes a real end-of-input
// (a clean unterminated literal, returned to the caller for the quote check)
// from a cursor read error such as a push-stream Read returning context.Canceled
// (PeekAt also reports 0 there), which is surfaced rather than swallowed.
func (pctx *parserCtx) scanQuotedLiteral(ctx context.Context, cur strcursor.Cursor, qch byte, pubid bool) (string, error) {
	buf := bufferPool.Get()
	defer releaseBuffer(buf)

	for {
		if err := ctx.Err(); err != nil {
			return "", pctx.error(ctx, err)
		}

		off := 0
		stop := false
		for off < literalScanChunk {
			b := cur.PeekAt(off)
			if b == 0 || b == qch {
				// End of literal: the closing quote, a genuine terminator/NUL, or an
				// exhausted buffer. When the buffer is exhausted (no byte present)
				// distinguish a clean end-of-input from a read failure (e.g. a
				// push-stream Read returning context.Canceled) so cancellation is
				// surfaced rather than treated as an unterminated literal.
				if b == 0 && !cur.HasByteAt(off) {
					if err := cur.Err(); err != nil {
						return "", pctx.error(ctx, err)
					}
					if err := ctx.Err(); err != nil {
						return "", pctx.error(ctx, err)
					}
				}
				stop = true
				break
			}
			if pubid {
				if !isPubidChar(rune(b)) {
					stop = true
					break
				}
				buf.WriteByte(b)
				off++
				continue
			}
			if b < 0x80 {
				if !isChar(rune(b)) {
					stop = true
					break
				}
				buf.WriteByte(b)
				off++
				continue
			}
			r, w, ok := decodeRuneAt(cur, off)
			if !ok || !isCharWidth(r, w) {
				stop = true
				break
			}
			buf.WriteRune(r)
			off += w
		}

		if off > 0 {
			if err := cur.Advance(off); err != nil {
				return "", pctx.error(ctx, err)
			}
		}
		// Enforce the node-content cap after each chunk: a run that fits a single
		// chunk is still bounded, and a multi-chunk run is bounded chunk-by-chunk
		// (peak buffered output stays within the cap plus at most one chunk).
		if pctx.nodeContentTooLong(buf.Len()) {
			return "", pctx.error(ctx, ErrNodeContentTooLarge)
		}
		if stop {
			break
		}
		// A full chunk was consumed without reaching the literal's end; loop to
		// scan and advance the next chunk so the cursor buffer stays bounded.
	}

	return buf.String(), nil
}

func (pctx *parserCtx) parseNotationType(ctx context.Context) (Enumeration, error) {
	cur := pctx.getCursor()
	if cur == nil {
		return nil, pctx.error(ctx, errNoCursor)
	}
	if cur.Peek() != '(' {
		return nil, pctx.error(ctx, ErrNotationNotStarted)
	}
	if err := cur.Advance(1); err != nil {
		return nil, pctx.error(ctx, err)
	}
	// Blanks (and, in the external subset, parameter-entity references supplying
	// the name list) between the '(' and each name/'|' are consumed through
	// skipBlanksPE; re-fetch the cursor after each skip since an expand/pop may
	// have changed the top input.
	if _, err := pctx.skipBlanksPE(ctx); err != nil {
		return nil, pctx.error(ctx, err)
	}
	cur = pctx.dtdRefetch(cur)

	names := map[string]struct{}{}

	var enumv Enumeration
	for pctx.instate != psEOF {
		name, err := pctx.parseName(ctx)
		if err != nil {
			return nil, pctx.error(ctx, ErrNotationNameRequired)
		}
		if _, ok := names[name]; ok {
			return nil, pctx.error(ctx, DTDDupTokenError{Name: name})
		}
		names[name] = struct{}{}

		enumv = append(enumv, name)
		if _, err := pctx.skipBlanksPE(ctx); err != nil {
			return nil, pctx.error(ctx, err)
		}
		cur = pctx.dtdRefetch(cur)

		if cur.Peek() != '|' {
			break
		}
		if err := cur.Advance(1); err != nil {
			return nil, pctx.error(ctx, err)
		}
		if _, err := pctx.skipBlanksPE(ctx); err != nil {
			return nil, pctx.error(ctx, err)
		}
		cur = pctx.dtdRefetch(cur)
	}

	if cur.Peek() != ')' {
		return nil, pctx.error(ctx, ErrNotationNotFinished)
	}
	if err := cur.Advance(1); err != nil {
		return nil, pctx.error(ctx, err)
	}
	return enumv, nil
}

func (pctx *parserCtx) parseEnumerationType(ctx context.Context) (Enumeration, error) {
	cur := pctx.getCursor()
	if cur == nil {
		return nil, pctx.error(ctx, errNoCursor)
	}
	if cur.Peek() != '(' {
		return nil, pctx.error(ctx, ErrAttrListNotStarted)
	}
	if err := cur.Advance(1); err != nil {
		return nil, pctx.error(ctx, err)
	}
	// In the external subset a parameter entity may supply the enumeration name
	// list (e.g. `(%vals;)`); skipBlanksPE expands it and crosses the boundary,
	// so re-fetch the cursor after each skip.
	if _, err := pctx.skipBlanksPE(ctx); err != nil {
		return nil, pctx.error(ctx, err)
	}
	cur = pctx.dtdRefetch(cur)

	names := map[string]struct{}{}

	var enumv Enumeration
	for pctx.instate != psEOF {
		name, err := pctx.parseNmtoken()
		if err != nil {
			return nil, pctx.error(ctx, ErrNmtokenRequired)
		}
		if _, ok := names[name]; ok {
			return nil, pctx.error(ctx, DTDDupTokenError{Name: name})
		}
		names[name] = struct{}{}

		enumv = append(enumv, name)
		if _, err := pctx.skipBlanksPE(ctx); err != nil {
			return nil, pctx.error(ctx, err)
		}
		cur = pctx.dtdRefetch(cur)

		if cur.Peek() != '|' {
			break
		}
		if err := cur.Advance(1); err != nil {
			return nil, pctx.error(ctx, err)
		}
		if _, err := pctx.skipBlanksPE(ctx); err != nil {
			return nil, pctx.error(ctx, err)
		}
		cur = pctx.dtdRefetch(cur)
	}

	if cur.Peek() != ')' {
		return nil, pctx.error(ctx, ErrAttrListNotFinished)
	}
	if err := cur.Advance(1); err != nil {
		return nil, pctx.error(ctx, err)
	}
	return enumv, nil
}

func (pctx *parserCtx) parseEnumeratedType(ctx context.Context) (enum.AttributeType, Enumeration, error) {
	cur := pctx.getCursor()
	if cur == nil {
		return enum.AttrInvalid, nil, pctx.error(ctx, errNoCursor)
	}
	if cur.ConsumeString("NOTATION") {
		adv, err := pctx.skipBlanksPE(ctx)
		if err != nil {
			return enum.AttrInvalid, nil, pctx.error(ctx, err)
		}
		if !adv {
			return enum.AttrInvalid, nil, pctx.error(ctx, ErrSpaceRequired)
		}
		tree, err := pctx.parseNotationType(ctx)
		if err != nil {
			return enum.AttrInvalid, nil, pctx.error(ctx, err)
		}

		return enum.AttrNotation, tree, nil
	}

	tree, err := pctx.parseEnumerationType(ctx)
	if err != nil {
		return enum.AttrInvalid, tree, pctx.error(ctx, err)
	}
	return enum.AttrEnumeration, tree, nil
}

func (pctx *parserCtx) parseAttributeType(ctx context.Context) (enum.AttributeType, Enumeration, error) {
	cur := pctx.getCursor()
	if cur == nil {
		return enum.AttrInvalid, nil, pctx.error(ctx, errNoCursor)
	}
	if cur.ConsumeString("CDATA") {
		return enum.AttrCDATA, nil, nil
	}
	if cur.ConsumeString("IDREFS") {
		return enum.AttrIDRefs, nil, nil
	}
	if cur.ConsumeString("IDREF") {
		return enum.AttrIDRef, nil, nil
	}
	if cur.ConsumeString("ID") {
		return enum.AttrID, nil, nil
	}
	if cur.ConsumeString("ENTITY") {
		return enum.AttrEntity, nil, nil
	}
	if cur.ConsumeString("ENTITIES") {
		return enum.AttrEntities, nil, nil
	}
	if cur.ConsumeString("NMTOKENS") {
		return enum.AttrNmtokens, nil, nil
	}
	if cur.ConsumeString("NMTOKEN") {
		return enum.AttrNmtoken, nil, nil
	}

	return pctx.parseEnumeratedType(ctx)
}

func (pctx *parserCtx) parseDefaultDecl(ctx context.Context) (deftype enum.AttributeDefault, defvalue string, err error) {
	deftype = enum.AttrDefaultNone
	cur := pctx.getCursor()
	if cur == nil {
		err = pctx.error(ctx, errNoCursor)
		return
	}
	if cur.ConsumeString("#REQUIRED") {
		deftype = enum.AttrDefaultRequired
		return
	}
	if cur.ConsumeString("#IMPLIED") {
		deftype = enum.AttrDefaultImplied
		return
	}

	if cur.ConsumeString("#FIXED") {
		deftype = enum.AttrDefaultFixed
		// The mandatory "S" after #FIXED — and, in the external subset, a
		// parameter entity supplying the #FIXED value (e.g. `#FIXED %v;`) — is
		// consumed through skipBlanksPE so a PE-supplied value's opening quote is
		// reached rather than left unexpanded.
		adv, serr := pctx.skipBlanksPE(ctx)
		if serr != nil {
			deftype = enum.AttrDefaultInvalid
			err = pctx.error(ctx, serr)
			return
		}
		if !adv {
			deftype = enum.AttrDefaultInvalid
			err = pctx.error(ctx, ErrSpaceRequired)
			return
		}
	}

	defvalue, err = pctx.parseQuotedText(func(qch byte) (string, error) {
		s, _, err := pctx.parseAttributeValueInternal(ctx, qch, false)
		return s, err
	})
	if err != nil {
		deftype = enum.AttrDefaultInvalid
		err = pctx.error(ctx, err)
		return
	}
	pctx.instate = psDTD
	err = nil
	return
}

func (ctx *parserCtx) attrNormalizeSpace(s string) (value string) {
	if len(s) == 0 {
		value = s
		return
	}

	i := 0
	for ; i < len(s); i++ {
		if s[i] != 0x20 {
			break
		}
	}

	out := make([]byte, 0, len(s))
	for i < len(s) {
		if s[i] != 0x20 {
			out = append(out, s[i])
			i++
			continue
		}
		for i < len(s) && s[i] == 0x20 {
			i++
		}
		out = append(out, 0x20)
	}

	if len(out) == 0 {
		return ""
	}
	if out[len(out)-1] == 0x20 {
		out = out[:len(out)-1]
	}
	value = string(out)
	return
}

func (ctx *parserCtx) cleanSpecialAttributes() {
	for k, v := range ctx.attsSpecial {
		if v == enum.AttrCDATA {
			delete(ctx.attsSpecial, k)
		}
	}
}

func (ctx *parserCtx) addSpecialAttribute(elemName, attrName string, typ enum.AttributeType) {
	if typ == enum.AttrID && ctx.loadsubset.IsSet(SkipIDs) {
		return
	}
	key := elemName + ":" + attrName
	// XML 1.0 §3.3: the first declaration of an attribute is binding. The parse
	// loop invokes this for every <!ATTLIST> declaration, including an ignored
	// duplicate; keeping the first-seen type ensures a later duplicate's type
	// cannot change how the attribute's explicit values are normalized.
	if _, ok := ctx.attsSpecial[key]; ok {
		return
	}
	ctx.attsSpecial[key] = typ
	// Record whether this binding originates in the external subset, for the VC:
	// Standalone Document Declaration normalization check (libxml2 XML_SPECIAL_EXTERNAL).
	if ctx.inSubset == inExternalSubset {
		ctx.attsSpecialExternal[key] = struct{}{}
	}
}

// specialAttributeExternal reports whether the effective tokenized-type binding
// for (elemName, attrName) was declared in the external subset.
func (ctx *parserCtx) specialAttributeExternal(elemName, attrName string) bool {
	if len(ctx.attsSpecialExternal) == 0 {
		return false
	}
	_, ok := ctx.attsSpecialExternal[elemName+":"+attrName]
	return ok
}

func (ctx *parserCtx) lookupSpecialAttribute(elemName, attrName string) (enum.AttributeType, bool) {
	if len(ctx.attsSpecial) == 0 {
		return 0, false
	}
	key := elemName + ":" + attrName
	v, ok := ctx.attsSpecial[key]
	return v, ok
}

func (ctx *parserCtx) addAttributeDecl(dtd *DTD, elem string, name string, prefix string, atype enum.AttributeType, def enum.AttributeDefault, defvalue string, tree Enumeration) (attr *AttributeDecl, err error) { //nolint:unparam // attr unused by callers but kept for API symmetry
	if dtd == nil {
		err = errors.New("dtd required")
		return
	}
	if name == "" {
		err = errors.New("name required")
		return
	}
	if elem == "" {
		err = errors.New("element required")
		return
	}

	switch atype {
	case enum.AttrCDATA, enum.AttrID, enum.AttrIDRef, enum.AttrIDRefs, enum.AttrEntity, enum.AttrEntities, enum.AttrNmtoken, enum.AttrNmtokens, enum.AttrEnumeration, enum.AttrNotation:
	default:
		err = errors.New("invalid attribute type")
		return
	}

	// Duplicate detection runs BEFORE validating this declaration's default
	// value: XML 1.0 §3.3 says a later declaration for the same attribute is
	// ignored ENTIRELY, so its (possibly invalid) default must not be validated
	// or abort the parse. The internal subset takes precedence over the external
	// one; a repeat within the same subset keeps the first declaration. This is a
	// validity warning, not a fatal error (libxml2 warns "already defined" and
	// continues).
	if doc := dtd.doc; doc != nil && doc.extSubset == dtd && doc.intSubset != nil && len(doc.intSubset.attributes) > 0 {
		if _, ok := doc.intSubset.LookupAttribute(name, prefix, elem); ok {
			return
		}
	}
	if existing, ok := dtd.LookupAttribute(name, prefix, elem); ok {
		return existing, nil
	}

	if defvalue != "" {
		if err = validateAttributeValueInternal(dtd.doc, atype, defvalue); err != nil {
			err = fmt.Errorf("attribute %s of %s: invalid default value: %s", elem, name, err)
			ctx.valid = false
			return
		}
	}

	attr = newAttributeDecl()
	attr.atype = atype
	attr.doc = dtd.doc
	attr.name = name
	attr.prefix = prefix
	attr.elem = elem
	attr.def = def
	attr.tree = tree
	attr.defvalue = defvalue

	if err = dtd.RegisterAttribute(attr); err != nil {
		attr = nil
		return
	}

	if err := dtd.AddChild(attr); err != nil {
		return nil, err
	}
	return attr, nil
}

func (ctx *parserCtx) addAttributeDefault(elemName, attrName, defaultValue string) {
	if _, ok := ctx.lookupSpecialAttribute(elemName, attrName); ok {
		return
	}

	existing := ctx.attsDefault[elemName]
	for _, a := range existing {
		if a.Name() == attrName {
			return
		}
	}

	var prefix string
	var local string
	if p, l, ok := strings.Cut(attrName, ":"); ok {
		prefix = p
		local = l
	} else {
		local = attrName
	}

	uri := ctx.lookupNamespace(prefix)
	attr, err := ctx.doc.CreateAttribute(local, defaultValue, newNamespace(prefix, uri))
	if err != nil {
		return
	}

	attr.SetDefault(true)
	if decl := lookupAttributeDecl(ctx.doc, local, prefix, elemName); decl != nil {
		attr.SetAType(decl.AType())
	}
	ctx.attsDefault[elemName] = append(existing, attr)
}

func (ctx *parserCtx) lookupAttributeDefault(elemName string) ([]*Attribute, bool) {
	v, ok := ctx.attsDefault[elemName]
	return v, ok
}

func (pctx *parserCtx) parseAttributeListDecl(ctx context.Context) error {
	cur := pctx.getCursor()
	if cur == nil {
		return pctx.error(ctx, errNoCursor)
	}
	if !cur.ConsumeString("<!ATTLIST") {
		return nil
	}
	startInput := pctx.currentInputID()

	// The "S" after <!ATTLIST, between tokens, and around a supplied token may be
	// provided by a parameter entity's §4.4.8 padding or a crossed PE-input
	// boundary in the external subset, so require it through skipBlanksPE (which
	// also expands a "%pe;" that supplies the element name / attribute
	// definitions) rather than a raw isBlankByte check. Re-fetch the cursor after
	// each skip and after each sub-parse: an expanded PE pushes a new input and an
	// exhausted PE input is popped, so a cursor captured earlier is stale.
	adv, err := pctx.skipBlanksPE(ctx)
	if err != nil {
		return pctx.error(ctx, err)
	}
	if !adv {
		return pctx.error(ctx, ErrSpaceRequired)
	}

	elemName, err := pctx.parseName(ctx)
	if err != nil {
		return pctx.error(ctx, err)
	}
	if _, err := pctx.skipBlanksPE(ctx); err != nil {
		return pctx.error(ctx, err)
	}

	cur = pctx.dtdRefetch(cur)
	if cur == nil {
		return pctx.error(ctx, errNoCursor)
	}
	for cur.Peek() != '>' && pctx.instate != psEOF {
		attrName, err := pctx.parseName(ctx)
		if err != nil {
			return pctx.error(ctx, ErrAttributeNameRequired)
		}
		adv, err := pctx.skipBlanksPE(ctx)
		if err != nil {
			return pctx.error(ctx, err)
		}
		if !adv {
			return pctx.error(ctx, ErrSpaceRequired)
		}

		typ, tree, err := pctx.parseAttributeType(ctx)
		if err != nil {
			return pctx.error(ctx, err)
		}

		adv, err = pctx.skipBlanksPE(ctx)
		if err != nil {
			return pctx.error(ctx, err)
		}
		if !adv {
			return pctx.error(ctx, ErrSpaceRequired)
		}

		def, defvalue, err := pctx.parseDefaultDecl(ctx)
		if err != nil {
			return pctx.error(ctx, err)
		}

		if typ != enum.AttrCDATA && def != enum.AttrDefaultInvalid {
			defvalue = pctx.attrNormalizeSpace(defvalue)
		}

		cur = pctx.dtdRefetch(cur)
		if cur == nil {
			return pctx.error(ctx, errNoCursor)
		}
		if c := cur.Peek(); c != '>' {
			adv, err := pctx.skipBlanksPE(ctx)
			if err != nil {
				return pctx.error(ctx, err)
			}
			if !adv {
				return pctx.error(ctx, ErrSpaceRequired)
			}
		}
		if s := pctx.sax; s != nil {
			switch err := s.AttributeDecl(ctx, elemName, attrName, typ, def, defvalue, tree); err {
			case nil, sax.ErrHandlerUnspecified:
			default:
				return pctx.error(ctx, err)
			}
		}

		if defvalue != "" && def != enum.AttrDefaultImplied && def != enum.AttrDefaultRequired {
			pctx.addAttributeDefault(elemName, attrName, defvalue)
		}

		pctx.addSpecialAttribute(elemName, attrName, typ)

		cur = pctx.dtdRefetch(cur)
		if cur == nil {
			return pctx.error(ctx, errNoCursor)
		}
		if cur.Peek() == '>' {
			// The whole <!ATTLIST ... > must start and stop in the same input: a
			// closing '>' supplied by a parameter entity that began mid-declaration
			// (e.g. `<!ATTLIST foo %e;` with %e; -> "... #IMPLIED>") is a boundary
			// violation, the same fatal condition <!ELEMENT> enforces.
			if pctx.currentInputID() != startInput {
				return pctx.error(ctx,
					fmt.Errorf("%w: attribute list declaration doesn't start and stop in the same entity", ErrEntityBoundary))
			}
			if err := cur.Advance(1); err != nil {
				return err
			}
			break
		}
	}
	return nil
}

func (pctx *parserCtx) parseNotationDecl(ctx context.Context) error {
	cur := pctx.getCursor()
	if cur == nil {
		return pctx.error(ctx, errNoCursor)
	}
	if !cur.ConsumeString("<!NOTATION") {
		return pctx.error(ctx, errors.New("<!NOTATION not started"))
	}
	// The whole <!NOTATION ... > must start and stop in the same input: a closing
	// '>' supplied by a different parameter entity than the one that opened the
	// declaration is a boundary violation, matching <!ELEMENT>/<!ATTLIST>/<!ENTITY>.
	startInput := pctx.currentInputID()

	adv, err := pctx.skipBlanksPE(ctx)
	if err != nil {
		return pctx.error(ctx, err)
	}
	if !adv {
		return pctx.error(ctx, ErrSpaceRequired)
	}

	name, err := pctx.parseName(ctx)
	if err != nil {
		return pctx.error(ctx, err)
	}

	adv, err = pctx.skipBlanksPE(ctx)
	if err != nil {
		return pctx.error(ctx, err)
	}
	if !adv {
		return pctx.error(ctx, ErrSpaceRequired)
	}

	systemID, publicID, err := pctx.parseExternalID(ctx, false)
	if err != nil {
		return pctx.error(ctx, err)
	}

	if _, err := pctx.skipBlanksPE(ctx); err != nil {
		return pctx.error(ctx, err)
	}

	cur = pctx.dtdRefetch(cur)
	if cur.Peek() != '>' {
		return pctx.error(ctx, errors.New("'>' required to close <!NOTATION"))
	}
	if pctx.currentInputID() != startInput {
		return pctx.error(ctx,
			fmt.Errorf("%w: notation declaration doesn't start and stop in the same entity", ErrEntityBoundary))
	}
	if err := cur.Advance(1); err != nil {
		return err
	}

	if s := pctx.sax; s != nil {
		switch err := s.NotationDecl(ctx, name, publicID, systemID); err {
		case nil, sax.ErrHandlerUnspecified:
		default:
			return pctx.error(ctx, err)
		}
	}

	return nil
}

func (pctx *parserCtx) parseSystemLiteral(ctx context.Context, qch byte) (string, error) {
	cur := pctx.getCursor()
	if cur == nil {
		return "", pctx.error(ctx, errNoCursor)
	}
	return pctx.scanQuotedLiteral(ctx, cur, qch, false)
}

func (pctx *parserCtx) parsePubidLiteral(ctx context.Context, qch byte) (string, error) {
	cur := pctx.getCursor()
	if cur == nil {
		return "", pctx.error(ctx, errNoCursor)
	}
	// PubidChar is restricted to a subset of ASCII, so any byte >= 0x80 (the lead
	// byte of a multi-byte sequence, including a real U+FFFD) is not a valid pubid
	// character; the pubid scan rejects it (ends the literal).
	return pctx.scanQuotedLiteral(ctx, cur, qch, true)
}

// isPubidChar reports whether r is a member of the XML PubidChar production:
//
//	PubidChar ::= #x20 | #xD | #xA | [a-zA-Z0-9] | [-'()+,./:=?;!*#@$_%]
func isPubidChar(r rune) bool {
	if r >= 'a' && r <= 'z' {
		return true
	}
	if r >= 'A' && r <= 'Z' {
		return true
	}
	if r >= '0' && r <= '9' {
		return true
	}
	switch r {
	case ' ', '\r', '\n', '-', '\'', '(', ')', '+', ',', '.', '/', ':', '=', '?', ';', '!', '*', '#', '@', '$', '_', '%':
		return true
	}
	return false
}

// externalIDLiteralError maps a SYSTEM/PUBLIC literal scan error to the
// caller-facing parse error. A genuine syntax failure (e.g. an unterminated
// literal surfacing as "string not closed") is reported with the domain-specific
// fallback message. But a resource-limit failure (ErrNodeContentTooLarge) or a
// parse-abort (context cancellation / deadline / StopParser) must propagate
// verbatim so errors.Is keeps matching and a cancelled parse is not disguised as
// a malformed external ID.
func (pctx *parserCtx) externalIDLiteralError(ctx context.Context, err error, fallback string) error {
	return pctx.preserveLimitOrAbort(ctx, err, errors.New(fallback))
}

// preserveLimitOrAbort returns err verbatim when it is a resource-limit failure
// (ErrNodeContentTooLarge) or a parse-abort (context cancellation / deadline /
// StopParser) so errors.Is keeps matching and a cancelled parse is not disguised
// as a generic decl error; otherwise it substitutes fallback (a domain-specific
// "value/ID required" sentinel for a genuine missing/empty literal). Used both by
// parseExternalID's own literal-scan call sites and by parseEntityDecl, which
// must not REPLACE an over-cap external SYSTEM/PUBLIC literal error from
// parseExternalID with a generic ErrValueRequired.
func (pctx *parserCtx) preserveLimitOrAbort(ctx context.Context, err, fallback error) error {
	if errors.Is(err, ErrNodeContentTooLarge) || isParseAbort(err) {
		return pctx.error(ctx, err)
	}
	return pctx.error(ctx, fallback)
}

// parseExternalID parses an ExternalID [75] or, when strict is false, a
// NotationDecl PublicID [83]. ExternalID's PUBLIC form requires "S
// SystemLiteral" after the PubidLiteral; NotationDecl's PublicID form permits
// PUBLIC with only the PubidLiteral. Pass strict=true for every ExternalID
// production (DOCTYPE external subset, entity declarations) and strict=false
// only for NotationDecl.
func (pctx *parserCtx) parseExternalID(ctx context.Context, strict bool) (string, string, error) {
	cur := pctx.getCursor()
	if cur == nil {
		return "", "", pctx.error(ctx, errNoCursor)
	}

	// The mandatory "S" separators here — and, in the external subset, a
	// parameter entity supplying a SYSTEM/PUBLIC literal (e.g.
	// `<!NOTATION n SYSTEM %sid;>`) — are consumed through skipBlanksPE, which
	// expands the PE and positions the cursor at the literal's opening quote. The
	// quoted literal scanners themselves do NOT recognize PE references (a '%'
	// inside a SystemLiteral/PubidLiteral is a literal character).
	if cur.HasPrefixString("SYSTEM") {
		if err := cur.Advance(6); err != nil {
			return "", "", err
		}
		adv, serr := pctx.skipBlanksPE(ctx)
		if serr != nil {
			return "", "", pctx.error(ctx, serr)
		}
		if !adv {
			return "", "", pctx.error(ctx, ErrSpaceRequired)
		}
		uri, err := pctx.parseQuotedText(func(qch byte) (string, error) {
			return pctx.parseSystemLiteral(ctx, qch)
		})
		if err != nil {
			return "", "", pctx.externalIDLiteralError(ctx, err, "system URI required")
		}
		return uri, "", nil
	} else if cur.HasPrefixString("PUBLIC") {
		if err := cur.Advance(6); err != nil {
			return "", "", err
		}
		adv, serr := pctx.skipBlanksPE(ctx)
		if serr != nil {
			return "", "", pctx.error(ctx, serr)
		}
		if !adv {
			return "", "", pctx.error(ctx, ErrSpaceRequired)
		}
		publicID, err := pctx.parseQuotedText(func(qch byte) (string, error) {
			return pctx.parsePubidLiteral(ctx, qch)
		})
		if err != nil {
			return "", "", pctx.externalIDLiteralError(ctx, err, "public ID required")
		}
		if strict {
			// ExternalID [75]: "S SystemLiteral" is mandatory after the
			// PubidLiteral, so a space (then the SystemLiteral below) is required.
			adv, serr := pctx.skipBlanksPE(ctx)
			if serr != nil {
				return "", "", pctx.error(ctx, serr)
			}
			if !adv {
				return "", "", pctx.error(ctx, ErrSpaceRequired)
			}
		} else {
			// NotationDecl PublicID [83]: the SystemLiteral is optional, so return
			// the PubidLiteral alone when no quoted SystemLiteral follows.
			adv, serr := pctx.skipBlanksPE(ctx)
			if serr != nil {
				return "", "", pctx.error(ctx, serr)
			}
			if !adv {
				return "", publicID, nil
			}
			cur = pctx.dtdRefetch(cur)
			if c := cur.Peek(); c != '\'' && c != '"' {
				return "", publicID, nil
			}
		}
		uri, err := pctx.parseQuotedText(func(qch byte) (string, error) {
			return pctx.parseSystemLiteral(ctx, qch)
		})
		if err != nil {
			return "", "", pctx.externalIDLiteralError(ctx, err, "system URI required")
		}
		return uri, publicID, nil
	}
	return "", "", nil
}
