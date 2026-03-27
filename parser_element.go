package helium

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/strcursor"
	"github.com/lestrrat-go/helium/sax"
	"github.com/lestrrat-go/pdebug"
)

/* parse a CharData section.
 * if we are within a CDATA section ']]>' marks an end of section.
 *
 * The right angle bracket (>) may be represented using the string "&gt;",
 * and must, for compatibility, be escaped using "&gt;" or a character
 * reference when it appears in the string "]]>" in content, when that
 * string is not marking the end of a CDATA section.
 *
 * [14] CharData ::= [^<&]* - ([^<&]* ']]>' [^<&]*)
 */
func (pctx *parserCtx) parseCharData(ctx context.Context, cdata bool) error {
	if cdata {
		_, err := pctx.parseCDataContent()
		return err
	}
	return pctx.parseCharDataContent(ctx)
}

func (pctx *parserCtx) parseCharDataContent(ctx context.Context) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseCharDataContent")
		defer g.IRelease("END parseCharDataContent")
	}

	cur := pctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}

	// Fast path: UTF8Cursor can scan directly into a []byte slice,
	// avoiding the bytes.Buffer intermediate.
	if u8, ok := cur.(*strcursor.UTF8Cursor); ok {
		data, i := u8.ScanCharDataSlice(pctx.charBuf[:0])
		if i <= 0 {
			if cur.Peek() == ']' && cur.PeekAt(1) == ']' && cur.PeekAt(2) == '>' {
				return pctx.error(ctx, ErrMisplacedCDATAEnd)
			}
			pdebug.Dump(cur)
			return errors.New("invalid char data")
		}

		if err := cur.AdvanceFast(i); err != nil {
			return err
		}

		// Keep the grown buffer for next call.
		pctx.charBuf = data

		if pctx.areBlanksBytes(data, false) {
			if pctx.treeBuilder != nil && !pctx.disableSAX {
				if err := pctx.fastIgnorableWhitespace(data); err != nil {
					return err
				}
			} else if s := pctx.sax; s != nil && !pctx.disableSAX {
				if err := pctx.deliverCharacters(ctx, s.IgnorableWhitespace, data); err != nil {
					return err
				}
			}
		} else {
			if pctx.treeBuilder != nil && !pctx.disableSAX {
				if err := pctx.fastCharacters(data); err != nil {
					return err
				}
			} else if s := pctx.sax; s != nil && !pctx.disableSAX {
				if err := pctx.deliverCharacters(ctx, s.Characters, data); err != nil {
					return err
				}
			}
		}
		return nil
	}

	// Fallback: use bytes.Buffer for non-UTF8 cursors.
	buf := bufferPool.Get()
	defer releaseBuffer(buf)

	i := cur.ScanCharDataInto(buf)
	if i <= 0 {
		if cur.Peek() == ']' && cur.PeekAt(1) == ']' && cur.PeekAt(2) == '>' {
			return pctx.error(ctx, ErrMisplacedCDATAEnd)
		}
		pdebug.Dump(cur)
		return errors.New("invalid char data")
	}

	if err := cur.AdvanceFast(i); err != nil {
		return err
	}

	data := buf.Bytes()
	if pctx.areBlanksBytes(data, false) {
		if pctx.treeBuilder != nil && !pctx.disableSAX {
			if err := pctx.fastIgnorableWhitespace(data); err != nil {
				return err
			}
		} else if s := pctx.sax; s != nil && !pctx.disableSAX {
			if err := pctx.deliverCharacters(ctx, s.IgnorableWhitespace, data); err != nil {
				return err
			}
		}
	} else {
		if pctx.treeBuilder != nil && !pctx.disableSAX {
			if err := pctx.fastCharacters(data); err != nil {
				return err
			}
		} else if s := pctx.sax; s != nil && !pctx.disableSAX {
			if err := pctx.deliverCharacters(ctx, s.Characters, data); err != nil {
				return err
			}
		}
	}

	return nil
}

func (pctx *parserCtx) parseElement(ctx context.Context) error {
	if pdebug.Enabled {
		pctx.elemidx++
		i := pctx.elemidx
		g := pdebug.IPrintf("START parseElement (%d)", i)
		defer g.IRelease("END parseElement (%d)", i)
	}

	pctx.elemDepth++
	defer func() { pctx.elemDepth-- }()

	if pctx.maxElemDepth > 0 && pctx.elemDepth > pctx.maxElemDepth {
		return pctx.error(ctx, fmt.Errorf("xml: exceeded max depth"))
	}

	// parseStartTag only parses up to the attributes.
	// For example, given <foo>bar</foo>, the next token would
	// be bar</foo>. Given <foo />, the next token would
	// be />
	if err := pctx.parseStartTag(ctx); err != nil {
		return pctx.error(ctx, err)
	}

	cur := pctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if cur.Peek() != '/' || cur.PeekAt(1) != '>' {
		if err := pctx.parseContent(ctx); err != nil {
			return pctx.error(ctx, err)
		}
	}

	if err := pctx.parseEndTag(ctx); err != nil {
		return pctx.error(ctx, err)
	}

	return nil
}

func (pctx *parserCtx) parseStartTag(ctx context.Context) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseStartTag")
		defer g.IRelease("END parseStartTag")
	}

	cur := pctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if cur.Peek() != '<' {
		return pctx.error(ctx, ErrStartTagRequired)
	}
	if err := cur.Advance(1); err != nil {
		return err
	}

	local, prefix, err := pctx.parseQName(ctx)
	if local == "" {
		return pctx.error(ctx, fmt.Errorf("local name empty! local = %s, prefix = %s, err = %s", local, prefix, err))
	}
	if err != nil {
		return pctx.error(ctx, err)
	}

	// Push xml:space stack entry for this element (inherit parent's value by default)
	pctx.spaceTab = append(pctx.spaceTab, -1)

	nbNs := 0
	if pctx.attrBuf == nil {
		pctx.attrBuf = make([]attrData, 0, 8)
	}
	attrs := pctx.attrBuf[:0]
	for pctx.instate != psEOF {
		pctx.skipBlanks(ctx)
		if cur.Peek() == '>' {
			if err := cur.Advance(1); err != nil {
				return err
			}
			break
		}

		if cur.Peek() == '/' && cur.PeekAt(1) == '>' {
			break
		}
		attname, aprefix, attvalue, err := pctx.parseAttribute(ctx, local)
		if err != nil {
			return pctx.error(ctx, err)
		}

		if attname == lexicon.PrefixXMLNS && aprefix == "" {
			// <elem xmlns="...">
			// Namespace URI entity/character references are expanded inline
			// during attribute value parsing (replaceEntities forced true in
			// parseAttribute for namespace attrs), so no post-processing needed.

			// parseNsClean: skip redundant namespace declarations
			if pctx.options.IsSet(parseNsClean) && pctx.nsTab.Lookup("") == attvalue {
				goto SkipDefaultNS
			}
			pctx.pushNS("", attvalue)
			nbNs++
		SkipDefaultNS:
			if cur.Peek() == '>' || (cur.Peek() == '/' && cur.PeekAt(1) == '>') {
				continue
			}

			if !isBlankByte(cur.Peek()) {
				return pctx.error(ctx, ErrSpaceRequired)
			}
			pctx.skipBlanks(ctx)
			continue
		} else if aprefix == lexicon.PrefixXMLNS {
			var u *url.URL
			var existingURI string

			// <elem xmlns:foo="...">
			// Namespace URI entity/character references are expanded inline
			// during attribute value parsing (replaceEntities forced true in
			// parseAttribute for namespace attrs), so no post-processing needed.
			if attname == lexicon.PrefixXML {
				if attvalue != lexicon.NamespaceXML {
					return pctx.namespaceError(ctx, errors.New("xml namespace prefix mapped to wrong URI"))
				}
				goto SkipNS
			}
			if attname == lexicon.PrefixXMLNS {
				return pctx.namespaceError(ctx, errors.New("redefinition of the xmlns prefix forbidden"))
			}

			if attvalue == lexicon.NamespaceXMLNS {
				return pctx.namespaceError(ctx, errors.New("reuse of the xmlns namespace name if forbidden"))
			}

			if attvalue == "" {
				return pctx.namespaceError(ctx, fmt.Errorf("xmlns:%s: Empty XML namespace is not allowed", attname))
			}

			u, err = url.Parse(attvalue)
			if err != nil {
				return pctx.namespaceError(ctx, fmt.Errorf("xmlns:%s: '%s' is not a validURI", attname, attvalue))
			}
			if pctx.pedantic && u.Scheme == "" {
				return pctx.namespaceError(ctx, fmt.Errorf("xmlns:%s: URI %s is not absolute", attname, attvalue))
			}

			// Check only the current element's bindings (top nbNs entries)
			// to detect true duplicates. A prefix bound in an ancestor
			// element is valid shadowing, not a duplicate.
			existingURI = pctx.nsTab.LookupInTopN(attname, nbNs)
			if existingURI != "" {
				if pctx.options.IsSet(parseNsClean) && existingURI == attvalue {
					goto SkipNS
				}
				return pctx.error(ctx, errors.New("duplicate attribute is not allowed"))
			}
			// parseNsClean: skip if an ancestor already binds this prefix
			// to the same URI (redundant redeclaration).
			if pctx.options.IsSet(parseNsClean) && pctx.nsTab.Lookup(attname) == attvalue {
				goto SkipNS
			}
			pctx.pushNS(attname, attvalue)
			nbNs++

		SkipNS:
			if cur.Peek() == '>' || (cur.Peek() == '/' && cur.PeekAt(1) == '>') {
				continue
			}

			if !isBlankByte(cur.Peek()) {
				return pctx.error(ctx, ErrSpaceRequired)
			}
			pctx.skipBlanks(ctx)
			continue
		}

		attr := attrData{
			localname: attname,
			prefix:    aprefix,
			value:     attvalue,
		}

		attrs = append(attrs, attr)
	}

	// Attributes defaulting: apply DTD-declared default attribute values.
	// NOTE: #FIXED/#REQUIRED validation and element content model checking
	// are done post-parse via validateDocument() when parseDTDValid is set.
	// ID/IDREF uniqueness checks are done post-parse via validateDocument().
	if len(pctx.attsDefault) > 0 {
		var elemName string
		if prefix != "" {
			elemName = prefix + ":" + local
		} else {
			elemName = local
		}

		if pdebug.Enabled {
			pdebug.Printf("-------> %s", elemName)
		}
		defaults, ok := pctx.lookupAttributeDefault(elemName)
		if ok {
			// First pass: apply default xmlns="..." (must come before prefixed)
			for _, attr := range defaults {
				if attr.LocalName() == lexicon.PrefixXMLNS && attr.Prefix() == "" {
					pctx.pushNS("", attr.Value())
					nbNs++
				}
			}
			// Second pass: apply xmlns:prefix="..." and regular attributes
			for _, attr := range defaults {
				attname := attr.LocalName()
				aprefix := attr.Prefix()
				if attname == lexicon.PrefixXMLNS && aprefix == "" {
					continue
				} else if aprefix == lexicon.PrefixXMLNS {
					pctx.pushNS(attname, attr.Value())
					nbNs++
				} else {
					dup := false
					for _, ea := range attrs {
						if ea.localname == attname && ea.prefix == aprefix {
							dup = true
							break
						}
					}
					if !dup {
						attrs = append(attrs, attrData{
							localname: attname,
							prefix:    aprefix,
							value:     attr.Value(),
							isDefault: attr.IsDefault(),
						})
					}
				}
			}
		}
	}

	for _, a := range attrs {
		if a.prefix == lexicon.PrefixXML && a.localname == "space" {
			switch a.value {
			case "preserve":
				pctx.spaceTab[len(pctx.spaceTab)-1] = 1
			case "default":
				pctx.spaceTab[len(pctx.spaceTab)-1] = 0
			}
			break
		}
	}

	nsuri := pctx.lookupNamespace(prefix)
	if prefix != "" && nsuri == "" {
		return pctx.namespaceError(ctx, errors.New("namespace '"+prefix+"' not found"))
	}

	if pctx.treeBuilder != nil && !pctx.disableSAX {
		if err := pctx.fastStartElement(local, prefix, nsuri, attrs, nbNs); err != nil {
			return pctx.error(ctx, err)
		}
	} else if s := pctx.sax; s != nil && !pctx.disableSAX {
		var nslist []sax.Namespace
		if nbNs > 0 {
			nslist = make([]sax.Namespace, nbNs)
			for i, ns := range pctx.nsTab.Peek(nbNs) {
				nslist[i] = ns
			}
		}
		var saxAttrs []sax.Attribute
		if len(attrs) > 0 {
			saxAttrs = make([]sax.Attribute, len(attrs))
			for i := range attrs {
				saxAttrs[i] = &attrs[i]
			}
		}
		switch err := s.StartElementNS(ctx, local, prefix, nsuri, nslist, saxAttrs); err {
		case nil, sax.ErrHandlerUnspecified:
			// no op
		default:
			return pctx.error(ctx, err)
		}
	}
	qname := local
	if prefix != "" {
		qname = prefix + ":" + local
	}
	pctx.pushNodeEntry(nodeEntry{local: local, prefix: prefix, uri: nsuri, qname: qname})
	pctx.nsNrTab = append(pctx.nsNrTab, nbNs)
	pctx.attrBuf = attrs[:0]

	return nil
}

/**
 * parse an end of tag
 *
 * [42] ETag ::= '</' Name S? '>'
 *
 * With namespace
 *
 * [NS 9] ETag ::= '</' QName S? '>'
 */
func (pctx *parserCtx) parseEndTag(ctx context.Context) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseEndTag")
		defer g.IRelease("END parseEndTag")
	}

	cur := pctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if cur.Peek() == '/' && cur.PeekAt(1) == '>' {
		if err := cur.Advance(2); err != nil {
			return err
		}
	} else {
		if !cur.ConsumeString("</") {
			return pctx.error(ctx, ErrLtSlashRequired)
		}

		e := pctx.peekNode()
		if !cur.ConsumeString(e.Name()) {
			return pctx.error(ctx, errors.New("expected end tag '"+e.Name()+"'"))
		}

		pctx.skipBlanks(ctx)

		if cur.Peek() != '>' {
			return pctx.error(ctx, ErrGtRequired)
		}
		if err := cur.Advance(1); err != nil {
			return err
		}
	}

	e := pctx.peekNode()
	if pctx.treeBuilder != nil && !pctx.disableSAX {
		if err := pctx.fastEndElement(); err != nil {
			return pctx.error(ctx, err)
		}
	} else if s := pctx.sax; s != nil && !pctx.disableSAX {
		switch err := s.EndElementNS(ctx, e.LocalName(), e.Prefix(), e.URI()); err {
		case nil, sax.ErrHandlerUnspecified:
			// no op
		default:
			return pctx.error(ctx, err)
		}
	}
	pctx.popNode()

	if len(pctx.spaceTab) > 1 {
		pctx.spaceTab = pctx.spaceTab[:len(pctx.spaceTab)-1]
	}

	if n := len(pctx.nsNrTab); n > 0 {
		nbNs := pctx.nsNrTab[n-1]
		pctx.nsNrTab = pctx.nsNrTab[:n-1]
		if nbNs > 0 {
			pctx.nsTab.Pop(nbNs)
		}
	}

	return nil
}

func (pctx *parserCtx) parseAttributeValue(ctx context.Context, normalize bool) (value string, entities int, err error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseAttributeValue (normalize=%t)", normalize)
		defer g.IRelease("END parseAttributeValue")
	}

	cur := pctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	qch := cur.Peek()
	switch qch {
	case '"', '\'':
		if err = cur.Advance(1); err != nil {
			return
		}
	default:
		err = errors.New("string not started (got '" + string([]byte{qch}) + "')")
		return
	}

	value, entities, err = pctx.parseAttributeValueInternal(ctx, qch, normalize)
	if err != nil {
		return
	}

	if cur.Peek() != qch {
		err = errors.New("string not closed")
		return
	}
	err = cur.Advance(1)
	return
}

// This is based on xmlParseAttValueComplex
func (pctx *parserCtx) parseAttributeValueInternal(ctx context.Context, qch byte, normalize bool) (value string, entities int, err error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseAttributeValueInternal (qch='%c',normalize=%t)", qch, normalize)
		defer g.IRelease("END parseAttributeValueInternal")
		defer func() {
			pdebug.Printf("value = '%s'", value)
		}()
	}

	prevState := pctx.instate
	pctx.instate = psAttributeValue
	defer func() { pctx.instate = prevState }()

	cur := pctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}

	if !normalize {
		if u8, ok := cur.(*strcursor.UTF8Cursor); ok {
			if v, nBytes := u8.ScanSimpleAttrValue(qch); nBytes > 0 {
				if err = u8.AdvanceFast(nBytes); err != nil {
					return
				}
				value = v
				return
			}
		}
	}

	inSpace := false
	b := bufferPool.Get()
	defer releaseBuffer(b)

	for {
		c := cur.PeekRune()
		if (qch != 0x0 && c == rune(qch)) || !isChar(c) || c == '<' {
			break
		}
		switch c {
		case '&':
			entities++
			inSpace = false
			if cur.PeekAt(1) == '#' {
				var r rune
				r, err = pctx.parseCharRef()
				if err != nil {
					err = pctx.error(ctx, err)
					return
				}

				if r == '&' && !pctx.replaceEntities {
					_, _ = b.WriteString("&#38;")
				} else {
					_, _ = b.WriteRune(r)
				}
			} else {
				var ent *Entity
				ent, err = pctx.parseEntityRef(ctx)
				if err != nil {
					err = pctx.error(ctx, err)
					return
				}

				if ent == nil {
					continue
				}

				if ent.entityType == enum.InternalPredefinedEntity {
					if ent.content == "&" && !pctx.replaceEntities {
						_, _ = b.WriteString("&#38;")
					} else {
						_, _ = b.WriteString(ent.content)
					}
				} else if pctx.replaceEntities {
					var rep string
					rep, err = pctx.decodeEntities(ctx, ent.Content(), SubstituteRef)
					if err != nil {
						err = pctx.error(ctx, err)
						return
					}
					for i := 0; i < len(rep); i++ {
						switch rep[i] {
						case 0xD, 0xA, 0x9:
							_ = b.WriteByte(0x20)
						default:
							_ = b.WriteByte(rep[i])
						}
					}
				} else {
					if ent.checked == 0 && strings.ContainsRune(ent.content, '&') {
						_, _ = pctx.decodeEntities(ctx, ent.Content(), SubstituteRef)
						ent.checked = 2
					}
					_, _ = b.WriteString("&")
					_, _ = b.WriteString(ent.name)
					_, _ = b.WriteString(";")
				}
			}
		case 0x20, 0xD, 0xA, 0x9:
			if b.Len() > 0 || !normalize {
				if !normalize || !inSpace {
					b.WriteByte(0x20)
				}
				inSpace = true
			}
			if err := cur.Advance(1); err != nil {
				return "", 0, err
			}
		default:
			inSpace = false
			b.WriteRune(c)
			w := utf8.RuneLen(c)
			if w < 1 {
				w = 1
			}
			if err := cur.Advance(w); err != nil {
				return "", 0, err
			}
		}
	}

	value = b.String()
	if inSpace && normalize {
		if value[len(value)-1] == 0x20 {
			for len(value) > 0 {
				if value[len(value)-1] != 0x20 {
					break
				}
				value = value[:len(value)-1]
			}
		}
	}

	return
}

func (pctx *parserCtx) parseAttribute(ctx context.Context, elemName string) (local string, prefix string, value string, err error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseAttribute")
		defer g.IRelease("END parseAttribute")
		defer func() {
			pdebug.Printf("local = '%s', prefix = '%s', value = '%s'", local, prefix, value)
		}()
	}
	l, p, err := pctx.parseQName(ctx)
	if err != nil {
		err = pctx.error(ctx, err)
		return
	}

	normalize := false
	attType, ok := pctx.lookupSpecialAttribute(elemName, l)
	if pdebug.Enabled {
		pdebug.Printf("looked up attribute %s:%s -> %d (%t)", elemName, l, attType, ok)
	}
	if ok && attType != enum.AttrInvalid {
		normalize = true
	}
	pctx.skipBlanks(ctx)

	cur := pctx.getCursor()
	if cur == nil {
		panic("did not get rune cursor")
	}
	if cur.Peek() != '=' {
		err = pctx.error(ctx, ErrEqualSignRequired)
	}
	if err2 := cur.Advance(1); err2 != nil {
		err = err2
		return
	}
	pctx.skipBlanks(ctx)

	isNamespace := (l == lexicon.PrefixXMLNS && p == "") || p == lexicon.PrefixXMLNS
	savedReplaceEntities := pctx.replaceEntities
	if isNamespace {
		pctx.replaceEntities = true
	}

	v, entities, err := pctx.parseAttributeValue(ctx, normalize)

	pctx.replaceEntities = savedReplaceEntities

	if err != nil {
		err = pctx.error(ctx, err)
		return
	}

	if normalize {
		if pdebug.Enabled {
			pdebug.Printf("normalize is true, checking if entities have been expanded...")
		}
		if entities > 0 {
			if pdebug.Enabled {
				pdebug.Printf("entities seems to have been expanded (%d): doint second normalization", entities)
			}
			v = pctx.attrNormalizeSpace(v)
		}
	}

	local = l
	prefix = p
	value = v
	err = nil
	return
}
