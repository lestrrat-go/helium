package helium

import (
	"errors"
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/pdebug"
)

// DTD represents an XML Document Type Definition (libxml2: xmlDtd).
type DTD struct {
	docnode
	attributes map[string]*AttributeDecl
	elements   map[string]*ElementDecl
	entities   map[string]*Entity
	pentities  map[string]*Entity
	notations  map[string]*Notation
	externalID string
	systemID   string
}

// Notation is a notation declaration from a DTD.
type Notation struct {
	docnode
	publicID string
	systemID string
}

func (n *Notation) AddChild(cur Node) error  { return addChild(n, cur) }
func (n *Notation) AppendText(b []byte) error { return appendText(n, b) }
func (n *Notation) AddSibling(cur Node) error { return addSibling(n, cur) }
func (n *Notation) Replace(nodes ...Node) error { return replaceNode(n, nodes...) }
func (n *Notation) SetTreeDoc(doc *Document)  { setTreeDoc(n, doc) }
func (n *Notation) Free()                     {}

// AttributeDecl is an xml attribute delcaration from DTD
type AttributeDecl struct {
	docnode
	atype    enum.AttributeType    // attribute type
	def      enum.AttributeDefault // default
	defvalue string                   // ... or the default value
	tree     Enumeration              // ... or the numeration tree, if any
	prefix   string                   // the namespace prefix, if any
	elem     string                   // name of the element holding the attribute
}

// ElementDecl is an xml element declaration from DTD
type ElementDecl struct {
	docnode
	decltype   enum.ElementType
	content    *ElementContent
	attributes *AttributeDecl
	prefix     string
	// xmlRegexpPtr contModel
}

// DeclType returns the element content type declared in the DTD
// (e.g., ElementElementType for element-only content).
func (e *ElementDecl) DeclType() enum.ElementType {
	return e.decltype
}

type ElementContentType int

const (
	ElementContentPCDATA ElementContentType = iota + 1
	ElementContentElement
	ElementContentSeq
	ElementContentOr
)

type ElementContentOccur int

const (
	ElementContentOnce ElementContentOccur = iota + 1
	ElementContentOpt
	ElementContentMult
	ElementContentPlus
)

type ElementContent struct {
	ctype  ElementContentType
	coccur ElementContentOccur
	name   string
	prefix string
	c1     *ElementContent
	c2     *ElementContent
	parent *ElementContent
}

func newDTD() *DTD {
	dtd := &DTD{
		attributes: map[string]*AttributeDecl{},
		elements:   map[string]*ElementDecl{},
		entities:   map[string]*Entity{},
		pentities:  map[string]*Entity{},
		notations:  map[string]*Notation{},
	}
	dtd.etype = DTDNode
	return dtd
}

func (dtd *DTD) AddEntity(name string, typ enum.EntityType, publicID, systemID, content string) (*Entity, error) {
	var table map[string]*Entity

	switch typ {
	case enum.InternalGeneralEntity, enum.ExternalGeneralParsedEntity, enum.ExternalGeneralUnparsedEntity:
		table = dtd.entities
	case enum.InternalParameterEntity, enum.ExternalParameterEntity:
		table = dtd.pentities
	case enum.InternalPredefinedEntity:
		return nil, errors.New("cannot register a predefined entity")
	}

	if table == nil {
		return nil, fmt.Errorf("invalid entity type: %d", typ)
	}

	// XML §4.6: predefined entities (lt, gt, amp, apos, quot) may be
	// redeclared, but only if the content resolves to the same character.
	// The content may contain character references (e.g., "&#60;" for "<")
	// that must be resolved before comparison.
	// libxml2: xmlAddEntity checks this and returns XML_ERR_REDECL_PREDEF_ENTITY.
	if typ == enum.InternalGeneralEntity {
		if expected, ok := predefinedEntityContent[name]; ok && resolveCharRefs(content) != expected {
			return nil, fmt.Errorf("entity '%s' redeclared with wrong content", name)
		}
	}

	// First definition wins (XML spec §4.2): if the entity already
	// exists, return the existing one and silently ignore the
	// redefinition, matching libxml2's behavior.
	if existing, ok := table[name]; ok {
		return existing, nil
	}

	ent := newEntity(name, typ, publicID, systemID, content, "")
	ent.doc = dtd.doc
	table[name] = ent

	if err := dtd.AddChild(ent); err != nil {
		return nil, err
	}
	return ent, nil
}

func (dtd *DTD) AddNotation(name, publicID, systemID string) (*Notation, error) {
	if _, ok := dtd.notations[name]; ok {
		return nil, fmt.Errorf("redefinition of notation %s", name)
	}
	nota := &Notation{}
	nota.etype = NotationNode
	nota.name = name
	nota.publicID = publicID
	nota.systemID = systemID
	nota.doc = dtd.doc
	dtd.notations[name] = nota
	if err := dtd.AddChild(nota); err != nil {
		return nil, err
	}
	return nota, nil
}

func (dtd *DTD) AddElementDecl(name string, typ enum.ElementType, content *ElementContent) (*ElementDecl, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START dtd.AddElementDecl '%s'", name)
		defer g.IRelease("END dtd.AddElementDecl")
	}

	switch typ {
	case enum.EmptyElementType, enum.AnyElementType:
		if content != nil {
			return nil, errors.New("content must be nil for EMPTY/ANY elements")
		}
	case enum.MixedElementType, enum.ElementElementType:
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
		if ok && decl.decltype == enum.UndefinedElementType {
			oldattrs = decl.attributes
			decl.attributes = nil
			doc.intSubset.RemoveElement(name, prefix)
		}
	}

	// The element may already be present if one of its attribute
	// was registered first
	decl, ok := dtd.elements[name+":"+prefix]
	if ok {
		if decl.decltype != enum.UndefinedElementType {
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

func (dtd *DTD) ForEachEntity(fn func(name string, ent *Entity)) {
	for name, ent := range dtd.entities {
		fn(name, ent)
	}
}

func (dtd *DTD) LookupEntity(name string) (*Entity, bool) {
	ret, ok := dtd.entities[name]
	return ret, ok
}

func (dtd *DTD) LookupParameterEntity(name string) (*Entity, bool) {
	ret, ok := dtd.pentities[name]
	return ret, ok
}

func (dtd *DTD) LookupNotation(name string) (*Notation, bool) {
	ret, ok := dtd.notations[name]
	return ret, ok
}

func (dtd *DTD) GetElementDesc(name string) (*ElementDecl, bool) {
	ret, ok := dtd.elements[name]
	return ret, ok
}

// AttributesForElement returns all attribute declarations for the named element.
func (dtd *DTD) AttributesForElement(elem string) []*AttributeDecl {
	var result []*AttributeDecl
	for _, adecl := range dtd.attributes {
		if adecl.elem == elem {
			result = append(result, adecl)
		}
	}
	return result
}

func (dtd *DTD) AddChild(cur Node) error {
	return addChild(dtd, cur)
}

func (dtd *DTD) AppendText(b []byte) error {
	return appendText(dtd, b)
}

func (dtd *DTD) AddSibling(cur Node) error {
	return addSibling(dtd, cur)
}

func (dtd *DTD) Replace(nodes ...Node) error {
	return replaceNode(dtd, nodes...)
}

func (dtd *DTD) SetTreeDoc(doc *Document) {
	setTreeDoc(dtd, doc)
}

func (dtd *DTD) Free() {}

// ExternalID returns the DTD external ID (PUBLIC identifier).
func (dtd *DTD) ExternalID() string {
	return dtd.externalID
}

// SystemID returns the DTD system ID (SYSTEM identifier).
func (dtd *DTD) SystemID() string {
	return dtd.systemID
}
