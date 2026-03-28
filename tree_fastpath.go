package helium

import (
	"errors"

	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/internal/lexicon"
)

type attrNamespaceCacheEntry struct {
	prefix string
	ns     *Namespace
}

func appendFastChild(parent MutableNode, child Node) error {
	pdn := parent.baseDocNode()
	cdn := child.baseDocNode()

	last := pdn.lastChild
	if last == nil {
		pdn.firstChild = child
		pdn.lastChild = child
		cdn.parent = parent
		return nil
	}

	ldn := last.baseDocNode()
	if ldn.next == nil {
		ldn.next = child
		cdn.prev = last
		cdn.parent = parent
		pdn.lastChild = child
		return nil
	}

	return last.(MutableNode).AddSibling(child) //nolint:forcetypeassert
}

func (pctx *parserCtx) fastLookupAttributeNamespace(doc *Document, prefix string, cache []attrNamespaceCacheEntry) (*Namespace, []attrNamespaceCacheEntry, error) {
	for i := range cache {
		if cache[i].prefix == prefix {
			return cache[i].ns, cache, nil
		}
	}

	uri := pctx.nsTab.Lookup(prefix)
	if uri == "" {
		if prefix != lexicon.PrefixXML {
			return nil, cache, nil
		}
		uri = lexicon.NamespaceXML
	}

	ns, err := doc.CreateNamespace(prefix, uri)
	if err != nil {
		return nil, cache, err
	}
	cache = append(cache, attrNamespaceCacheEntry{
		prefix: prefix,
		ns:     ns,
	})
	return ns, cache, nil
}

func (pctx *parserCtx) fastStartDocument() {
	pctx.doc = NewDocument(pctx.version, pctx.encoding, pctx.standalone)
	pctx.doc.idsSkip = pctx.loadsubset.IsSet(SkipIDs)
	pctx.doc.url = pctx.baseURI
}

func (pctx *parserCtx) fastEndDocument() {
	if pctx.doc == nil || !pctx.wellFormed {
		return
	}
	pctx.doc.properties |= DocWellFormed
	if pctx.valid {
		pctx.doc.properties |= DocDTDValid
	}
}

func (pctx *parserCtx) fastProcessingInstruction(target, data string) error {
	doc := pctx.doc
	if doc == nil {
		return errors.New("processing instruction placed in wrong location")
	}

	pi := doc.CreatePI(target, data)
	if pctx.currentEntityURI != "" {
		pi.entityBaseURI = pctx.currentEntityURI
	}

	switch pctx.inSubset {
	case 1:
		return doc.IntSubset().AddChild(pi)
	case 2:
		return doc.ExtSubset().AddChild(pi)
	}

	parent := pctx.elem
	if parent == nil {
		return appendFastChild(doc, pi)
	}
	if parent.Type() == ElementNode {
		return appendFastChild(parent, pi)
	}
	return parent.AddSibling(pi)
}

func (pctx *parserCtx) fastStartElement(localname, prefix, uri string, attrs []attrData, nbNs int) error {
	doc := pctx.doc
	if doc == nil {
		return errors.New("element placed in wrong location")
	}

	e := doc.CreateElement(localname)
	e.SetLine(pctx.LineNumber())
	if pctx.currentEntityURI != "" {
		e.entityBaseURI = pctx.currentEntityURI
	}

	if uri != "" {
		if err := e.SetActiveNamespace(prefix, uri); err != nil {
			return err
		}
	}

	if nbNs > 0 {
		for _, ns := range pctx.nsTab.Peek(nbNs) {
			if err := e.DeclareNamespace(ns.Prefix(), ns.URI()); err != nil {
				return err
			}
		}
	}

	registerIDs := !pctx.loadsubset.IsSet(SkipIDs)
	needsAttrDeclLookup := doc.IntSubset() != nil || doc.ExtSubset() != nil
	elemName := localname
	if needsAttrDeclLookup && prefix != "" {
		elemName = prefix + ":" + localname
	}

	var nsCacheBuf [4]attrNamespaceCacheEntry
	nsCache := nsCacheBuf[:0]
	var lastAttr *Attribute
	for i := range attrs {
		attr := attrs[i]
		if attr.isDefault && !pctx.loadsubset.IsSet(CompleteAttrs) {
			continue
		}

		var ns *Namespace
		if attr.prefix != "" {
			var err error
			ns, nsCache, err = pctx.fastLookupAttributeNamespace(doc, attr.prefix, nsCache)
			if err != nil {
				return err
			}
		}

		var created *Attribute
		if pctx.replaceEntities {
			created = doc.createLiteralAttribute(attr.localname, attr.value, ns)
		} else {
			var err error
			created, err = doc.CreateAttribute(attr.localname, attr.value, ns)
			if err != nil {
				return err
			}
		}
		if attr.isDefault {
			created.SetDefault(true)
		}
		if lastAttr == nil {
			e.properties = created
		} else {
			lastAttr.next = created
			created.prev = lastAttr
		}
		created.parent = e
		lastAttr = created

		if needsAttrDeclLookup {
			if decl := lookupAttributeDecl(doc, attr.localname, attr.prefix, elemName); decl != nil {
				created.SetAType(decl.AType())
				if registerIDs && decl.AType() == enum.AttrID {
					doc.RegisterID(attr.value, e)
					continue
				}
			}
		}
		if registerIDs && attr.prefix == lexicon.PrefixXML && attr.localname == "id" {
			doc.RegisterID(attr.value, e)
		}
	}

	var parent MutableNode
	if pctx.elem != nil {
		parent = pctx.elem
	}
	if parent == nil {
		if err := appendFastChild(doc, e); err != nil {
			return err
		}
	} else if parent.Type() == ElementNode {
		if err := appendFastChild(parent, e); err != nil {
			return err
		}
	} else {
		if err := parent.AddSibling(e); err != nil {
			return err
		}
	}

	pctx.elem = e
	return nil
}

func (pctx *parserCtx) fastEndElement() error {
	cur := pctx.elem
	if cur == nil {
		return errors.New("no context node to end")
	}

	parent := cur.Parent()
	if e, ok := parent.(*Element); ok {
		pctx.elem = e
		return nil
	}
	pctx.elem = nil
	return nil
}

func (pctx *parserCtx) fastCharacters(data []byte) error {
	parent := pctx.elem
	if parent == nil {
		return errors.New("text content placed in wrong location")
	}

	pdn := parent.baseDocNode()
	if last := pdn.lastChild; last != nil {
		if t, ok := AsType[*Text](last); ok {
			return t.AppendText(data)
		}
	}

	text := pctx.doc.CreateText(data)
	return appendFastChild(parent, text)
}

func (pctx *parserCtx) fastIgnorableWhitespace(data []byte) error {
	if !pctx.keepBlanks {
		return nil
	}
	return pctx.fastCharacters(data)
}

func (pctx *parserCtx) fastCDataBlock(data []byte) error {
	parent := pctx.elem
	if parent == nil {
		return nil
	}

	cdata := pctx.doc.CreateCDATASection(data)
	return appendFastChild(parent, cdata)
}

func (pctx *parserCtx) fastComment(data []byte) error {
	doc := pctx.doc
	if doc == nil {
		return errors.New("comment placed in wrong location")
	}

	comment := doc.CreateComment(data)
	switch pctx.inSubset {
	case inInternalSubset:
		return doc.IntSubset().AddChild(comment)
	case inExternalSubset:
		return doc.ExtSubset().AddChild(comment)
	}

	if pctx.elem == nil {
		return appendFastChild(doc, comment)
	}
	if pctx.elem.Type() == ElementNode {
		return appendFastChild(pctx.elem, comment)
	}
	return pctx.elem.AddSibling(comment)
}
