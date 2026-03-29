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
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/sax"
	"github.com/lestrrat-go/pdebug"
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
		if !ok || !isChar(r) {
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
	if pdebug.Enabled {
		g := pdebug.IPrintf("START decodeEntitites (%s)", s)
		defer func() {
			g.IRelease("END decodeEntities ('%s' -> '%s')", s, ret)
		}()
	}
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
		pdebug.Printf("s[0] -> %c", s[0])
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
	if pdebug.Enabled {
		g := pdebug.Marker("parseEntityValue")
		defer g.End()
	}

	pctx.instate = psEntityValue

	literal, err := pctx.parseQuotedText(func(qch byte) (string, error) {
		return pctx.parseEntityValueInternal(ctx, qch)
	})
	if err != nil {
		return "", "", pctx.error(ctx, err)
	}

	val, err := pctx.decodeEntities(ctx, []byte(literal), SubstitutePERef)
	if err != nil {
		return "", "", pctx.error(ctx, err)
	}

	if pdebug.Enabled {
		pdebug.Printf("parsed entity value '%s'", val)
	}

	return literal, val, nil
}

func (pctx *parserCtx) parseEntityDecl(ctx context.Context) error {
	if pdebug.Enabled {
		g := pdebug.Marker("parseEntityDecl")
		defer g.End()
	}

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
			if err == nil {
				switch err := pctx.fireSAXCallback(ctx, cbEntityDecl, name, value); err {
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
			if err == nil {
				if s := pctx.sax; s != nil {
					switch err := s.EntityDecl(ctx, name, enum.InternalGeneralEntity, "", "", value); err {
					case nil, sax.ErrHandlerUnspecified:
					default:
						return pctx.error(ctx, err)
					}
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

func (pctx *parserCtx) parseExternalEntityPrivate(ctx context.Context, uri, externalID string) (Node, error) {
	if pctx.options.IsSet(parseNoXXE) {
		return nil, nil //nolint:nilnil
	}

	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseExternalEntityPrivate(uri=%s, externalID=%s)", uri, externalID)
		defer g.IRelease("END parseExternalEntityPrivate")
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

	content, err := io.ReadAll(input)
	if err != nil {
		return nil, pctx.error(ctx, fmt.Errorf("reading external entity: %w", err))
	}

	newctx := &parserCtx{}
	if err := newctx.init(nil, bytes.NewReader(content)); err != nil {
		return nil, err
	}
	defer func() {
		if err := newctx.release(); err != nil && pdebug.Enabled {
			pdebug.Printf("newctx.release() failed: %s", err)
		}
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
	newctx.sax = pctx.sax
	newctx.attsDefault = pctx.attsDefault
	newctx.options = pctx.options
	newctx.depth = pctx.depth + 1
	newctx.external = true
	newctx.replaceEntities = pctx.replaceEntities
	newctx.loadsubset = pctx.loadsubset
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

	newRoot := newctx.doc.CreateElement("pseudoroot")
	newctx.pushNodeEntry(nodeEntry{local: "pseudoroot", qname: "pseudoroot"})
	newctx.elem = newRoot
	if err := newctx.doc.AddChild(newRoot); err != nil {
		return nil, err
	}
	if err := newctx.parseContent(innerCtx); err != nil {
		return nil, err
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
	if pdebug.Enabled {
		g := pdebug.IPrintf("START parseBalancedChunkInternal")
		defer g.IRelease("END parseBalancedChunkInternal")
	}

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
		if err := newctx.release(); err != nil && pdebug.Enabled {
			pdebug.Printf("newctx.release() failed: %s", err)
		}
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
	newctx.sax = pctx.sax
	newctx.attsDefault = pctx.attsDefault
	newctx.depth = pctx.depth + 1
	if pctx.elem != nil {
		for _, ns := range collectInScopeNamespaces(pctx.elem) {
			newctx.pushNS(ns.Prefix(), ns.URI())
		}
	}
	newctx.sizeentcopy = pctx.sizeentcopy
	newctx.inputSize = pctx.inputSize
	newctx.maxAmpl = pctx.maxAmpl
	defer func() { pctx.sizeentcopy = newctx.sizeentcopy }()

	newRoot := newctx.doc.CreateElement("pseudoroot")
	newctx.pushNodeEntry(nodeEntry{local: "pseudoroot", qname: "pseudoroot"})
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
