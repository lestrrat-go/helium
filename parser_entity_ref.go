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
	"github.com/lestrrat-go/pdebug"
)

func (pctx *parserCtx) parseReference(ctx context.Context) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseReference")
		defer g.IRelease("END parseReference")
	}

	cur := pctx.getCursor()
	if cur == nil {
		return pctx.error(ctx, errNoCursor)
	}
	if cur.Peek() != '&' {
		return pctx.error(ctx, ErrAmpersandRequired)
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
			if err := pctx.deliverCharacters(ctx, s.Characters, b); err != nil {
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
			case nil, ErrParseSucceeded:
			default:
				return err
			}
		} else if ent.EntityType() == enum.ExternalGeneralParsedEntity {
			parsedEnt, err = pctx.parseExternalEntityPrivate(ctx, ent.uri, ent.externalID)
			switch err {
			case nil, ErrParseSucceeded:
			default:
				return err
			}
		} else {
			return errors.New("invalid entity type")
		}

		if ent.checked == 0 {
			ent.checked = 2
		}
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
				case nil, ErrParseSucceeded:
				default:
					return err
				}
			} else if ent.EntityType() == enum.ExternalGeneralParsedEntity {
				parsedEnt, err = pctx.parseExternalEntityPrivate(ctx, ent.uri, ent.externalID)
				_ = parsedEnt
				switch err {
				case nil, ErrParseSucceeded:
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

func parseStringCharRef(s []byte) (r rune, width int, err error) {
	if pdebug.Enabled {
		g := pdebug.Marker("parseStringCharRef")
		defer func() {
			pdebug.Printf("r = '%c' (%x), consumed %d bytes", &r, &r, &width)
			g.End()
		}()
	}
	var val int32
	r = utf8.RuneError
	width = 0
	if !bytes.HasPrefix(s, []byte{'&', '#'}) {
		err = errors.New("ampersand (&) was required")
		return
	}

	width += 2
	s = s[2:]

	var accumulator func(int32, rune) (int32, error)
	if s[0] == 'x' {
		s = s[1:]
		width++
		accumulator = accumulateHexCharRef
	} else {
		accumulator = accumulateDecimalCharRef
	}

	for c := s[0]; c != ';'; c = s[0] {
		val, err = accumulator(val, rune(c))
		if err != nil {
			width = 0
			return
		}
		if val > unicode.MaxRune {
			err = errors.New("hex CharRef out of range")
			width = 0
			return
		}

		s = s[1:]
		width++
	}

	if s[0] == ';' {
		s = s[1:]
		_ = s
		width++
	}

	r = val
	if !isXMLCharValue(uint32(val)) {
		return utf8.RuneError, 0, fmt.Errorf("invalid XML char value %d", val)
	}
	return
}

func parseStringName(s []byte) (string, int, error) {
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

	return out.String(), i, nil
}

func (ctx *parserCtx) getEntity(name string) (*Entity, error) {
	if ctx.inSubset == 0 {
		if ret, err := resolvePredefinedEntity(name); err == nil {
			return ret, nil
		}
	}

	var ret *Entity
	var ok bool
	if ctx.doc == nil {
		return nil, ErrEntityNotFound
	} else if ctx.doc.standalone != 1 {
		ret, _ = ctx.doc.GetEntity(name)
	} else {
		if ctx.inSubset == 2 {
			ctx.doc.standalone = 0
			ret, _ = ctx.doc.GetEntity(name)
			ctx.doc.standalone = 1
		} else {
			ret, ok = ctx.doc.GetEntity(name)
			if !ok {
				ctx.doc.standalone = 0
				ret, ok = ctx.doc.GetEntity(name)
				if !ok {
					return nil, errors.New("Entity(" + name + ") document marked standalone but requires eternal subset")
				}
				ctx.doc.standalone = 1
			}
		}
	}
	return ret, nil
}

func (pctx *parserCtx) parseStringEntityRef(ctx context.Context, s []byte) (sax.Entity, int, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseStringEntityRef ('%s')", s)
		defer g.IRelease("END parseStringEntityRef")
	}
	if len(s) == 0 || s[0] != '&' {
		return nil, 0, errors.New("invalid entity ref")
	}

	i := 1
	name, width, err := parseStringName(s[1:])
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
				loadedEnt, err = pctx.getEntity(name)
				if err != nil {
					return nil, 0, err
				}
			}
		}
	}
	if loadedEnt == nil {
		if pctx.standalone == StandaloneExplicitYes || (!pctx.hasExternalSubset && !pctx.hasPERefs) {
			return nil, 0, fmt.Errorf("entity '%s' not defined", name)
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
		return nil, 0, fmt.Errorf("attribute references enternal entity '%s'", name)
	}

	if pctx.instate == psAttributeValue && len(loadedEnt.Content()) > 0 && loadedEnt.EntityType() == enum.InternalPredefinedEntity && bytes.IndexByte(loadedEnt.Content(), '<') > -1 {
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
	name, width, err := parseStringName(s[1:])
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
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseCharRef")
		defer g.IRelease("END parseCharRef")
		defer func() { pdebug.Printf("r = '%c' (%x)", r, r) }()
	}

	r = utf8.RuneError

	var val int32
	cur := ctx.getCursor()
	if cur == nil {
		err = errNoCursor
		return
	}
	if cur.ConsumeString("&#x") {
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
			if err := cur.Advance(1); err != nil {
				return utf8.RuneError, err
			}
		}
		if cur.Peek() == ';' {
			if err := cur.Advance(1); err != nil {
				return utf8.RuneError, err
			}
		}
	} else if cur.ConsumeString("&#") {
		for !cur.Done() && cur.Peek() != ';' {
			c := cur.Peek()
			if c >= '0' && c <= '9' {
				val = val*10 + int32(c-'0')
			} else {
				err = errors.New("invalid decimal CharRef")
				return
			}
			if err := cur.Advance(1); err != nil {
				return utf8.RuneError, err
			}
		}
		if cur.Peek() == ';' {
			if err := cur.Advance(1); err != nil {
				return utf8.RuneError, err
			}
		}
	} else {
		err = errors.New("invalid char ref")
		return
	}

	if isXMLCharValue(uint32(val)) && val <= unicode.MaxRune {
		r = val
		return
	}

	err = ErrInvalidChar
	return
}

func (pctx *parserCtx) parseEntityRef(ctx context.Context) (ent *Entity, err error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseEntityRef")
		defer func() {
			g.IRelease("END parseEntityRef ent = %#v", ent)
		}()
	}

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
		return
	}

	err = nil

	if s := pctx.sax; s != nil {
		var loadedEnt sax.Entity
		loadedEnt, _ = s.GetEntity(ctx, name)
		if loadedEnt != nil {
			ent = loadedEnt.(*Entity) //nolint:forcetypeassert
			return
		}

		ent, _ = pctx.getEntity(name)
	}

	if ent == nil {
		if pctx.standalone == StandaloneExplicitYes || (!pctx.hasExternalSubset && pctx.hasPERefs) {
			return nil, pctx.error(ctx, ErrUndeclaredEntity)
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
		return nil, nil
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

	if ent == nil {
		return nil, pctx.error(ctx, errors.New("entity resolution produced nil entity"))
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
	if ctx.maxAmpl == 0 {
		return nil
	}

	if e, ok := ent.(*Entity); ok && e != nil && e.Checked() {
		ctx.sizeentcopy = saturatedAdd(ctx.sizeentcopy, e.expandedSize)
		ctx.sizeentcopy = saturatedAdd(ctx.sizeentcopy, entityFixedCost)
	} else {
		ctx.sizeentcopy = saturatedAdd(ctx.sizeentcopy, int64(size))
		ctx.sizeentcopy = saturatedAdd(ctx.sizeentcopy, entityFixedCost)
	}

	if ctx.sizeentcopy > entityAllowedExpansion {
		consumed := ctx.inputSize
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
	if pdebug.Enabled {
		g := pdebug.Marker("handlePEReference")
		defer g.End()
	}

	cur := pctx.getCursor()
	if cur == nil {
		return pctx.error(ctx, errNoCursor)
	}
	if cur.Peek() != '%' {
		return nil
	}

	switch st := pctx.instate; st {
	case psCDATA, psComment, psStartTag, psEndTag, psEntityDecl, psContent, psAttributeValue, psPI, psSystemLiteral, psPublicLiteral, psEntityValue, psIgnore:
		if pdebug.Enabled {
			pdebug.Printf("instate == %s, ignoring", st)
		}
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

		if c := cur.PeekAt(1); isBlankByte(c) || c == 0 {
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
