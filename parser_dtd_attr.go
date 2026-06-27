package helium

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/sax"
)

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
	pctx.skipBlanks(ctx)

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
		pctx.skipBlanks(ctx)

		if cur.Peek() != '|' {
			break
		}
		if err := cur.Advance(1); err != nil {
			return nil, pctx.error(ctx, err)
		}
		pctx.skipBlanks(ctx)
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
	pctx.skipBlanks(ctx)

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
		pctx.skipBlanks(ctx)

		if cur.Peek() != '|' {
			break
		}
		if err := cur.Advance(1); err != nil {
			return nil, pctx.error(ctx, err)
		}
		pctx.skipBlanks(ctx)
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
		if !isBlankByte(cur.Peek()) {
			return enum.AttrInvalid, nil, pctx.error(ctx, ErrSpaceRequired)
		}
		pctx.skipBlanks(ctx)
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
		if !isBlankByte(cur.Peek()) {
			deftype = enum.AttrDefaultInvalid
			err = pctx.error(ctx, ErrSpaceRequired)
			return
		}
		pctx.skipBlanks(ctx)
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
	ctx.attsSpecial[key] = typ
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

	if defvalue != "" {
		if err = validateAttributeValueInternal(dtd.doc, atype, defvalue); err != nil {
			err = fmt.Errorf("attribute %s of %s: invalid default value: %s", elem, name, err)
			ctx.valid = false
			return
		}
	}

	if doc := dtd.doc; doc != nil && doc.extSubset == dtd && doc.intSubset != nil && len(doc.intSubset.attributes) > 0 {
		if _, ok := doc.intSubset.LookupAttribute(name, prefix, elem); ok {
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

	if !isBlankByte(cur.Peek()) {
		return pctx.error(ctx, ErrSpaceRequired)
	}
	pctx.skipBlanks(ctx)

	elemName, err := pctx.parseName(ctx)
	if err != nil {
		return pctx.error(ctx, err)
	}
	pctx.skipBlanks(ctx)

	for cur.Peek() != '>' && pctx.instate != psEOF {
		attrName, err := pctx.parseName(ctx)
		if err != nil {
			return pctx.error(ctx, ErrAttributeNameRequired)
		}
		if !isBlankByte(cur.Peek()) {
			return pctx.error(ctx, ErrSpaceRequired)
		}
		pctx.skipBlanks(ctx)

		typ, tree, err := pctx.parseAttributeType(ctx)
		if err != nil {
			return pctx.error(ctx, err)
		}

		if !isBlankByte(cur.Peek()) {
			return pctx.error(ctx, ErrSpaceRequired)
		}
		pctx.skipBlanks(ctx)

		def, defvalue, err := pctx.parseDefaultDecl(ctx)
		if err != nil {
			return pctx.error(ctx, err)
		}

		if typ != enum.AttrCDATA && def != enum.AttrDefaultInvalid {
			defvalue = pctx.attrNormalizeSpace(defvalue)
		}

		if c := cur.Peek(); c != '>' {
			if !isBlankByte(c) {
				return pctx.error(ctx, ErrSpaceRequired)
			}
			pctx.skipBlanks(ctx)
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

		if cur.Peek() == '>' {
			if pctx.currentInputID() != startInput {
				_ = pctx.warning(ctx, "attribute list declaration doesn't start and stop in the same entity\n")
				pctx.valid = false
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

	if !pctx.skipBlanks(ctx) {
		return pctx.error(ctx, ErrSpaceRequired)
	}

	name, err := pctx.parseName(ctx)
	if err != nil {
		return pctx.error(ctx, err)
	}

	if !pctx.skipBlanks(ctx) {
		return pctx.error(ctx, ErrSpaceRequired)
	}

	systemID, publicID, err := pctx.parseExternalID(ctx, false)
	if err != nil {
		return pctx.error(ctx, err)
	}

	pctx.skipBlanks(ctx)

	if cur.Peek() != '>' {
		return pctx.error(ctx, errors.New("'>' required to close <!NOTATION"))
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
	}
	return buf.String(), nil
}

func (pctx *parserCtx) parsePubidLiteral(ctx context.Context, qch byte) (string, error) {
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
		// PubidChar is restricted to a subset of ASCII, so any byte >= 0x80
		// (the lead byte of a multi-byte sequence, including a real U+FFFD)
		// is not a valid pubid character.
		if !isPubidChar(rune(b)) {
			break
		}
		buf.WriteByte(b)
		off++
	}
	if off > 0 {
		if err := cur.Advance(off); err != nil {
			return "", pctx.error(ctx, err)
		}
	}
	return buf.String(), nil
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

	if cur.HasPrefixString("SYSTEM") {
		if err := cur.Advance(6); err != nil {
			return "", "", err
		}
		if !isBlankByte(cur.Peek()) {
			return "", "", pctx.error(ctx, ErrSpaceRequired)
		}
		pctx.skipBlanks(ctx)
		uri, err := pctx.parseQuotedText(func(qch byte) (string, error) {
			return pctx.parseSystemLiteral(ctx, qch)
		})
		if err != nil {
			return "", "", pctx.error(ctx, errors.New("system URI required"))
		}
		return uri, "", nil
	} else if cur.HasPrefixString("PUBLIC") {
		if err := cur.Advance(6); err != nil {
			return "", "", err
		}
		if !isBlankByte(cur.Peek()) {
			return "", "", pctx.error(ctx, ErrSpaceRequired)
		}
		pctx.skipBlanks(ctx)
		publicID, err := pctx.parseQuotedText(func(qch byte) (string, error) {
			return pctx.parsePubidLiteral(ctx, qch)
		})
		if err != nil {
			return "", "", pctx.error(ctx, errors.New("public ID required"))
		}
		if strict {
			// ExternalID [75]: "S SystemLiteral" is mandatory after the
			// PubidLiteral, so a space (then the SystemLiteral below) is required.
			if !isBlankByte(cur.Peek()) {
				return "", "", pctx.error(ctx, ErrSpaceRequired)
			}
			pctx.skipBlanks(ctx)
		} else {
			// NotationDecl PublicID [83]: the SystemLiteral is optional, so return
			// the PubidLiteral alone when no quoted SystemLiteral follows.
			if !isBlankByte(cur.Peek()) {
				return "", publicID, nil
			}
			pctx.skipBlanks(ctx)
			if c := cur.Peek(); c != '\'' && c != '"' {
				return "", publicID, nil
			}
		}
		uri, err := pctx.parseQuotedText(func(qch byte) (string, error) {
			return pctx.parseSystemLiteral(ctx, qch)
		})
		if err != nil {
			return "", "", pctx.error(ctx, errors.New("system URI required"))
		}
		return uri, publicID, nil
	}
	return "", "", nil
}
