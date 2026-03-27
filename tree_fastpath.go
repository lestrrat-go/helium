package helium

import (
	"errors"

	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/internal/lexicon"
)

func (pctx *parserCtx) fastStartDocument() {
	pctx.doc = NewDocument(pctx.version, pctx.encoding, pctx.standalone)
	pctx.doc.ids = make(map[string]*Element)
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
		return doc.AddChild(pi)
	}
	if parent.Type() == ElementNode {
		return parent.AddChild(pi)
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

	for i := range attrs {
		attr := attrs[i]
		if attr.isDefault && !pctx.loadsubset.IsSet(CompleteAttrs) {
			continue
		}

		if attr.prefix != "" {
			ns := lookupNSByPrefix(e, attr.prefix)
			if ns == nil && pctx.elem != nil {
				ns = lookupNSByPrefix(pctx.elem, attr.prefix)
			}
			if pctx.replaceEntities {
				_ = e.SetLiteralAttributeNS(attr.localname, attr.value, ns)
				continue
			}
			if _, err := e.SetAttributeNS(attr.localname, attr.value, ns); err != nil {
				return err
			}
			continue
		}

		if pctx.replaceEntities {
			_ = e.SetLiteralAttribute(attr.localname, attr.value)
			continue
		}
		if _, err := e.SetAttribute(attr.localname, attr.value); err != nil {
			return err
		}
	}

	elemName := localname
	if prefix != "" {
		elemName = prefix + ":" + localname
	}
	registerIDs := !pctx.loadsubset.IsSet(SkipIDs)
	e.ForEachAttribute(func(a *Attribute) bool {
		if decl := lookupAttributeDecl(doc, a.LocalName(), a.Prefix(), elemName); decl != nil {
			a.SetAType(decl.AType())
		}
		if registerIDs && (a.Name() == lexicon.QNameXMLID || a.AType() == enum.AttrID) {
			doc.RegisterID(a.Value(), e)
		}
		return true
	})

	var parent MutableNode
	if pctx.elem != nil {
		parent = pctx.elem
	}
	if parent == nil {
		if err := doc.AddChild(e); err != nil {
			return err
		}
	} else if parent.Type() == ElementNode {
		if err := parent.AddChild(e); err != nil {
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
	n := pctx.elem
	if n == nil {
		return errors.New("text content placed in wrong location")
	}
	return n.AppendText(data)
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
	return parent.AddChild(cdata)
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
		return doc.AddChild(comment)
	}
	if pctx.elem.Type() == ElementNode {
		return pctx.elem.AddChild(comment)
	}
	return pctx.elem.AddSibling(comment)
}
