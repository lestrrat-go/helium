package helium

import (
	"strings"

	"github.com/lestrrat-go/pdebug"
	"github.com/pkg/errors"
)

func newDTD() *DTD {
	dtd := &DTD{
		attributes: map[string]*AttributeDecl{},
		elements:   map[string]*ElementDecl{},
		entities:   map[string]*Entity{},
		pentities:  map[string]*Entity{},
	}
	dtd.etype = DTDNode
	return dtd
}

func (dtd *DTD) AddEntity(name string, typ EntityType, publicID, systemID, content string) (*Entity, error) {
	var table map[string]*Entity

	switch typ {
	case InternalGeneralEntity, ExternalGeneralParsedEntity, ExternalGeneralUnparsedEntity:
		table = dtd.entities
	case InternalParameterEntity, ExternalParameterEntity:
		table = dtd.pentities
	case InternalPredefinedEntity:
		return nil, errors.New("cannot register a predefined entity")
	}

	if table == nil {
		return nil, errors.Errorf("invalid entity type: %d", typ)
	}

	ent := newEntity(name, typ, publicID, systemID, content, "")
	ent.doc = dtd.doc
	table[name] = ent

	dtd.AddChild(ent)
	return ent, nil
}

func (dtd *DTD) AddElementDecl(name string, typ ElementTypeVal, content *ElementContent) (*ElementDecl, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START dtd.AddElementDecl '%s'", name)
		defer g.IRelease("END dtd.AddElementDecl")
	}

	switch typ {
	case EmptyElementType, AnyElementType:
		if content != nil {
			return nil, errors.New("content must be nil for EMPTY/ANY elements")
		}
	case MixedElementType, ElementElementType:
		if content == nil {
			return nil, errors.New("content must be non-nil for MIXED/ELEMENT elements")
		}
	default:
		return nil, errors.New("invalid ElementContent")
	}

	var prefix string
	if i := strings.IndexByte(name, ':'); i > -1 {
		prefix = name[:i]
		name = name[i+1:]
	}

	var oldattrs *AttributeDecl
	// lookup old attributes inserted on an undefined element in the
	// internal subset.
	if doc := dtd.doc; doc != nil && doc.intSubset != nil {
		decl, ok := doc.intSubset.LookupElement(name, prefix)
		if ok && decl.decltype == UndefinedElementType {
			oldattrs = decl.attributes
			decl.attributes = nil
			doc.intSubset.RemoveElement(name, prefix)
		}
	}

	// The element may already be present if one of its attribute
	// was registered first
	decl, ok := dtd.elements[name+":"+prefix]
	if ok {
		if decl.decltype != UndefinedElementType {
			return nil, errors.New("redefinition of element " + name)
		}
	} else {
		decl = newElementDecl()
		decl.name = name
		decl.prefix = prefix
		decl.attributes = oldattrs

		dtd.elements[name+":"+prefix] = decl
	}

	decl.decltype = typ

	/*
	   // Avoid a stupid copy when called by the parser
	   // and flag it by setting a special parent value
	   // so the parser doesn't unallocate it.
	   if ((ctxt != NULL) &&
	       ((ctxt->finishDtd == XML_CTXT_FINISH_DTD_0) ||
	        (ctxt->finishDtd == XML_CTXT_FINISH_DTD_1))) {
	       ret->content = content;
	       if (content != NULL)
	           content->parent = (xmlElementContentPtr) 1;
	   } else {
	       ret->content = xmlCopyDocElementContent(dtd->doc, content);
	   }
	*/
	decl.content = content.copyElementContent()

	decl.doc = dtd.doc
	if err := dtd.AddChild(decl); err != nil {
		return nil, err
	}

	return decl, nil
}

func (dtd *DTD) LookupElement(name, prefix string) (*ElementDecl, bool) {
	key := name + ":" + prefix
	decl, ok := dtd.elements[key]
	if !ok {
		return nil, false
	}
	return decl, true
}

func (dtd *DTD) RemoveElement(name, prefix string) {
	key := name + ":" + prefix
	delete(dtd.elements, key)
}

func (dtd *DTD) LookupAttribute(name, prefix, elem string) (*AttributeDecl, bool) {
	key := name + ":" + prefix + ":" + elem
	decl, ok := dtd.attributes[key]
	if !ok {
		return nil, false
	}
	return decl, ok
}

func (dtd *DTD) RegisterAttribute(attr *AttributeDecl) error {
	// TODO maybe this shouldn't be normalized, check later
	key := attr.name + ":" + attr.prefix + ":" + attr.elem
	_, ok := dtd.attributes[key]
	if ok {
		return errors.New("duplicate attribute declared")
	}
	dtd.attributes[key] = attr
	return nil
}

func (dtd *DTD) LookupEntity(name string) (*Entity, bool) {
	ret, ok := dtd.entities[name]
	return ret, ok
}

func (dtd *DTD) LookupParameterEntity(name string) (*Entity, bool) {
	ret, ok := dtd.pentities[name]
	return ret, ok
}

func (dtd *DTD) GetElementDesc(name string) (*ElementDecl, bool) {
	ret, ok := dtd.elements[name]
	return ret, ok
}

func (dtd *DTD) AddChild(cur Node) error {
	return addChild(dtd, cur)
}

func (dtd *DTD) AddContent(b []byte) error {
	return addContent(dtd, b)
}

func (dtd *DTD) AddSibling(cur Node) error {
	return addSibling(dtd, cur)
}

func (dtd *DTD) Replace(cur Node) {
	replaceNode(dtd, cur)
}

func (dtd *DTD) SetTreeDoc(doc *Document) {
	setTreeDoc(dtd, doc)
}

func (dtd *DTD) Free() {}
