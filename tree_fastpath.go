package helium

import (
	"errors"
	"strings"

	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/nodelink"
)

type attrNamespaceCacheEntry struct {
	prefix string
	ns     *Namespace
}

func init() {
	nodelink.AppendFastChild = nodelinkAppendFastChild
	nodelink.BindEntityReference = nodelinkBindEntityReference
	nodelink.CloneEntityReferenceBinding = nodelinkCloneEntityReferenceBinding
}

// nodelinkAppendFastChild adapts appendFastChild to the untyped
// internal/nodelink hook (any in) so a sibling package can link a freshly built
// child without a public function here or an import cycle.
func nodelinkAppendFastChild(parent, child any) error {
	p, ok := parent.(MutableNode)
	if !ok {
		return errors.New("parent is not a mutable node")
	}
	c, ok := child.(Node)
	if !ok {
		return errors.New("child is not a node")
	}
	return appendFastChild(p, c)
}

// nodelinkBindEntityReference adapts bindEntityReference to the untyped
// internal/nodelink hook so a sibling-package copier can retain declaration
// identity without adding a public mutation API.
func nodelinkBindEntityReference(ref, entity any) {
	r, refOK := ref.(*EntityRef)
	e, entityOK := entity.(*Entity)
	if !refOK || !entityOK {
		return
	}
	bindEntityReference(r, e)
}

// nodelinkCloneEntityReferenceBinding gives the strip-space copier a private
// entity replacement view for one reference context. The document's DTD keeps
// its shared declaration unchanged; the private entity carries the declaration
// metadata and receives context-filtered replacement children from the caller.
func nodelinkCloneEntityReferenceBinding(ref, entity any) any {
	r, refOK := ref.(*EntityRef)
	e, entityOK := entity.(*Entity)
	if !refOK || !entityOK {
		return nil
	}
	cp := copyEntity(e, r.OwnerDocument())
	bindEntityReference(r, cp)
	return cp
}

// appendFastChild links child as the last child of parent without running the
// cycle-guard and duplicate-attribute preflight that AddChild performs. The
// CALLER is responsible for passing a child that cannot create a cycle or a
// duplicate attribute (e.g. a freshly-constructed node in a deep copy), because
// those safety checks are skipped. Misuse on an arbitrary live tree can corrupt
// linkage; ordinary code MUST use AddChild instead.
func appendFastChild(parent MutableNode, child Node) error {
	pdn := parent.baseDocNode()
	cdn := child.baseDocNode()

	// A nil tail means parent has no child it owns. Install child as the owned
	// list rather than linking through a foreign head's sibling chain.
	last := resolveOwnedTail(parent, pdn)
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
	pctx.doc.pooledSlabs = true
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

// appendFastChildElem is appendFastChild for a parent known to be an *Element,
// the parser's case: the recorded tail is trusted when it has no successor and
// names this very element as its parent (an element never holds an off-chain
// child claim), and any other shape goes through the generic resolution.
func appendFastChildElem(parent *Element, child Node) error {
	pdn := &parent.docnode
	if l := pdn.lastChild; l != nil && pdn.firstChild != nil {
		ldn := l.baseDocNode()
		if ldn.next == nil && ldn.parent == Node(parent) {
			cdn := child.baseDocNode()
			ldn.next = child
			cdn.prev = l
			cdn.parent = parent
			pdn.lastChild = child
			return nil
		}
	}
	return appendFastChild(parent, child)
}

func (pctx *parserCtx) fastStartElement(localname, prefix, uri string, attrs []attrData, nbNs int) error {
	doc := pctx.doc
	if doc == nil {
		return errors.New("element placed in wrong location")
	}

	// Build the element as CreateElement does, without its colon check:
	// parseQName returns an NCName as the local name (its parseName fallback
	// runs only when parseNCName has already failed on the same input, so it
	// fails too), so localname never holds a colon.
	e := doc.allocElement()
	e.name = localname
	e.etype = ElementNode
	e.doc = doc
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
		if err := declareNamespaces(e, pctx.nsTab.Peek(nbNs)); err != nil {
			return err
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

		// A value without '&' holds no reference, so CreateAttribute would
		// build the same single Text child createLiteralAttribute builds
		// (stringToNodeList's no-reference path), and its colon check cannot
		// fire: attr.localname is an NCName from parseQName, or the local name
		// of a DTD default that CreateAttribute already accepted.
		var created *Attribute
		if pctx.replaceEntities || strings.IndexByte(attr.value, '&') < 0 {
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

	parent := pctx.elem
	if parent == nil {
		if err := appendFastChild(doc, e); err != nil {
			return err
		}
	} else if parent.Type() == ElementNode {
		if err := appendFastChildElem(parent, e); err != nil {
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

	if e, ok := cur.parent.(*Element); ok {
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

	// A plain type assertion plus a nil check answers what AsNode[*Text]
	// answers, without its reflect-based typed-nil probe, on every text run.
	pdn := &parent.docnode
	if t, ok := pdn.lastChild.(*Text); ok && t != nil {
		return t.AppendText(data)
	}

	text := pctx.doc.CreateText(data)
	return appendFastChildElem(parent, text)
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
