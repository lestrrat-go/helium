package helium

import (
	"errors"
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium/enum"
)

// DTD represents an XML Document Type Definition (libxml2: xmlDtd).
type DTD struct {
	docnode
	attributes map[attrDeclKey]*AttributeDecl
	elements   map[string]*ElementDecl
	entities   map[string]*Entity
	pentities  map[string]*Entity
	notations  map[string]*Notation
	externalID string
	systemID   string
}

// attrDeclKey identifies an attribute declaration by its owning element name and
// the attribute's local name + prefix (the QName split on its first colon). A
// struct key avoids the ambiguity of a `local + ":" + prefix + ":" + elem` string
// concatenation, which collides distinct triples — e.g. ("d","c:a:b") and
// ("c:d","b:a") both concatenate to "a:b:c:d", so the second would be wrongly
// rejected as a duplicate. This mirrors the parser's specialAttrKey
// (parser_dtd_attr.go) and libxml2's two-arg hash lookup (xmlHashQLookup3).
type attrDeclKey struct {
	local  string
	prefix string
	elem   string
}

// Notation is a notation declaration from a DTD.
type Notation struct {
	docnode
	publicID string
	systemID string
}

// AddChild appends cur as the last child of the notation node.
func (n *Notation) AddChild(cur Node) error { return addChild(n, cur) }

// AppendText appends b as a Text child of the notation node.
func (n *Notation) AppendText(b []byte) error { return appendText(n, b) }

// AddSibling appends cur as the last sibling of the notation node.
func (n *Notation) AddSibling(cur Node) error { return addSibling(n, cur) }

// Replace swaps the notation node out of its parent, inserting nodes in its place.
func (n *Notation) Replace(nodes ...Node) error { return replaceNode(n, nodes...) }

// SetTreeDoc sets the owning document of the notation node and its subtree.
func (n *Notation) SetTreeDoc(doc *Document) { setTreeDoc(n, doc) }

// Free is a no-op; it exists to satisfy the Node interface.
func (n *Notation) Free() {}

func newDTD() *DTD {
	dtd := &DTD{
		attributes: map[attrDeclKey]*AttributeDecl{},
		elements:   map[string]*ElementDecl{},
		entities:   map[string]*Entity{},
		pentities:  map[string]*Entity{},
		notations:  map[string]*Notation{},
	}
	dtd.etype = DTDNode
	return dtd
}

// AddEntity declares an entity in the DTD and registers it as a child node,
// routing general entities and parameter entities into their respective tables
// based on typ. Predefined entities cannot be registered, and an unknown typ is
// rejected. Redeclaring an existing general/parameter entity is a no-op that
// returns the existing declaration (first definition wins, per XML §4.2);
// redeclaring a predefined entity (lt, gt, amp, apos, quot) with content that
// does not resolve to the same character is an error.
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

// AddNotation declares a notation in the DTD and registers it as a child node.
// It returns an error (wrapping ErrDuplicateDeclaration) if a notation with the
// same name is already declared, and an error (wrapping ErrInvalidArgument) if
// name contains a colon: a notation name is an XML NCName, so a colon-bearing
// name produces a <!NOTATION> declaration the parser rejects ("colons are
// forbidden from notation names"). Otherwise it trusts the caller for a
// well-formed name, public ID, and system ID.
func (dtd *DTD) AddNotation(name, publicID, systemID string) (*Notation, error) {
	// A notation name is an XML NCName; a colon is forbidden. This mirrors the
	// parser's own NotationDecl Name check exactly (parser_dtd_attr.go), so a
	// name accepted here always reparses.
	if strings.ContainsRune(name, ':') {
		return nil, fmt.Errorf("colon is forbidden in notation name %q: %w", name, ErrInvalidArgument)
	}
	if _, ok := dtd.notations[name]; ok {
		return nil, fmt.Errorf("redefinition of notation %s: %w", name, ErrDuplicateDeclaration)
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

// AddElementDecl declares an element content model in the DTD and registers it
// as a child node. The name may be a QName; its prefix is split off for keying.
// content must be nil for EMPTY/ANY element types and non-nil for MIXED/ELEMENT
// types. A previously undefined declaration (created when one of the element's
// attributes was declared first) is completed in place; a second concrete
// declaration of the same element is an error.
func (dtd *DTD) AddElementDecl(name string, typ enum.ElementType, content *ElementContent) (*ElementDecl, error) {
	switch typ {
	case enum.EmptyElementType, enum.AnyElementType:
		if content != nil {
			return nil, errors.New("content must be nil for EMPTY/ANY elements")
		}
	case enum.MixedElementType, enum.ElementElementType:
		if content == nil {
			return nil, errors.New("content must be non-nil for MIXED/ELEMENT elements")
		}
		// Reject a structurally-incomplete model (e.g. a sequence/choice node with
		// nil children) before it is stored, so serialization can never nil-deref.
		// Validated first, before any mutation of the DTD tables below.
		if err := validateElementContentModel(content); err != nil {
			return nil, fmt.Errorf("invalid content model: %w", err)
		}
	default:
		return nil, errors.New("invalid ElementContent")
	}

	// Split a QName into prefix + local on the FIRST colon, mirroring libxml2's
	// xmlSplitQName3: a leading colon (i == 0) is NOT a prefix separator — the
	// whole string (colon included) is the local name — so ":x" does not collide
	// with the unprefixed "x" (XML 1.0 5th-edition Name; eduni ibm04v01).
	var prefix string
	if i := strings.IndexByte(name, ':'); i > 0 {
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
			return nil, fmt.Errorf("redefinition of element %s: %w", name, ErrDuplicateDeclaration)
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

// LookupElement returns the element declaration registered under the given
// local name and prefix, and reports whether it was found.
func (dtd *DTD) LookupElement(name, prefix string) (*ElementDecl, bool) {
	key := name + ":" + prefix
	decl, ok := dtd.elements[key]
	if !ok {
		return nil, false
	}
	return decl, true
}

// RemoveElement removes the element declaration registered under the given
// local name and prefix. It deletes the lookup-table entry AND unlinks the
// declaration node from the DTD child list, so the declaration is no longer
// serialized. It returns the removed declaration, or nil if none was registered
// under that key.
func (dtd *DTD) RemoveElement(name, prefix string) *ElementDecl {
	key := name + ":" + prefix
	decl, ok := dtd.elements[key]
	if !ok {
		return nil
	}
	delete(dtd.elements, key)
	unlinkNode(decl)
	return decl
}

// LookupAttribute returns the attribute declaration registered for the given
// attribute local name, prefix, and owning element name, and reports whether it
// was found.
func (dtd *DTD) LookupAttribute(name, prefix, elem string) (*AttributeDecl, bool) {
	decl, ok := dtd.attributes[attrDeclKey{local: name, prefix: prefix, elem: elem}]
	if !ok {
		return nil, false
	}
	return decl, ok
}

// AddAttributeDecl declares an attribute for the named element in the DTD and
// registers it as a child node, so it serializes as an <!ATTLIST> declaration
// (mirroring how AddElementDecl/AddEntity/AddNotation link their declarations).
//
// It validates the enum parameters — atype must be a valid enum.Attr* type and
// def a valid enum.AttrDefaultNone/Required/Implied/Fixed kind — and rejects a
// duplicate. name may be a QName; its prefix is split off for keying on the FIRST
// colon, exactly as AddElementDecl does.
//
// Like its sibling constructors (AddNotation checks only the duplicate; AddEntity
// only the type enum; AddElementDecl only the content-model structure), it TRUSTS
// the Go caller to supply well-formed input. It does NOT validate the element or
// attribute name against the Name grammar, the default value's characters, a
// namespace-declaration default's URI, or any cross-declaration validity
// constraint (those are enforced by ValidateDTD when the document is validated).
// A caller that passes a malformed name or value gets a declaration that may not
// round-trip through a validating parse — that is the caller's responsibility,
// exactly as for the siblings.
//
// The caller's enumValues slice is cloned before it is stored, so a later mutation
// of the caller's slice cannot corrupt the serialized declaration.
//
// It returns an error wrapping ErrDuplicateDeclaration when an attribute with the
// same local name, prefix, and owning element is already declared, or an error
// wrapping ErrInvalidArgument when atype or def is out of range.
func (dtd *DTD) AddAttributeDecl(elem, name string, atype enum.AttributeType, def enum.AttributeDefault, defvalue string, enumValues Enumeration) (*AttributeDecl, error) {
	switch atype {
	case enum.AttrCDATA, enum.AttrID, enum.AttrIDRef, enum.AttrIDRefs, enum.AttrEntity, enum.AttrEntities, enum.AttrNmtoken, enum.AttrNmtokens, enum.AttrEnumeration, enum.AttrNotation:
	default:
		return nil, fmt.Errorf("invalid attribute type: %w", ErrInvalidArgument)
	}

	switch def {
	case enum.AttrDefaultNone, enum.AttrDefaultRequired, enum.AttrDefaultImplied, enum.AttrDefaultFixed:
	default:
		return nil, fmt.Errorf("invalid attribute default declaration: %w", ErrInvalidArgument)
	}

	// Split the QName into prefix + local on the FIRST colon, mirroring
	// AddElementDecl.
	local := name
	var prefix string
	if i := strings.IndexByte(name, ':'); i > 0 {
		prefix = name[:i]
		local = name[i+1:]
	}

	if _, ok := dtd.LookupAttribute(local, prefix, elem); ok {
		return nil, fmt.Errorf("duplicate attribute %s declared for element %s: %w", name, elem, ErrDuplicateDeclaration)
	}

	attr := newAttributeDecl()
	attr.atype = atype
	attr.doc = dtd.doc
	attr.name = local
	attr.prefix = prefix
	attr.elem = elem
	attr.def = def
	// Clone the token list: storing the caller's slice by reference would let a
	// later mutation of it corrupt the serialized declaration.
	if len(enumValues) > 0 {
		attr.tree = append(Enumeration(nil), enumValues...)
	}
	attr.defvalue = defvalue

	if err := dtd.registerAttribute(attr); err != nil {
		return nil, err
	}
	if err := dtd.AddChild(attr); err != nil {
		return nil, err
	}
	return attr, nil
}

// registerAttribute records an already-built attribute declaration in the DTD's
// lookup table, keyed by its name, prefix, and owning element. It does NOT link
// the declaration into the DTD child list, so it does not serialize on its own;
// AddAttributeDecl is the public entry point that both registers and links a
// declaration built from public parameters. It returns an error wrapping
// ErrDuplicateDeclaration if an attribute with the same key is already declared.
func (dtd *DTD) registerAttribute(attr *AttributeDecl) error {
	key := attrDeclKey{local: attr.name, prefix: attr.prefix, elem: attr.elem}
	_, ok := dtd.attributes[key]
	if ok {
		return fmt.Errorf("duplicate attribute %s declared for element %s: %w", attr.name, attr.elem, ErrDuplicateDeclaration)
	}
	dtd.attributes[key] = attr
	return nil
}

// ForEachEntity calls fn for every general entity declared in the DTD. The
// iteration order is unspecified.
func (dtd *DTD) ForEachEntity(fn func(name string, ent *Entity)) {
	for name, ent := range dtd.entities {
		fn(name, ent)
	}
}

// LookupEntity returns the general entity declared under name, and reports
// whether it was found.
func (dtd *DTD) LookupEntity(name string) (*Entity, bool) {
	ret, ok := dtd.entities[name]
	return ret, ok
}

// LookupParameterEntity returns the parameter entity declared under name, and
// reports whether it was found.
func (dtd *DTD) LookupParameterEntity(name string) (*Entity, bool) {
	ret, ok := dtd.pentities[name]
	return ret, ok
}

// LookupNotation returns the notation declared under name, and reports whether
// it was found.
func (dtd *DTD) LookupNotation(name string) (*Notation, bool) {
	ret, ok := dtd.notations[name]
	return ret, ok
}

// GetElementDesc returns the element declaration for the given QName, splitting
// off any prefix to compose the lookup key, and reports whether it was found.
func (dtd *DTD) GetElementDesc(name string) (*ElementDecl, bool) {
	// Element decls are registered under a "name:prefix" key with the QName
	// split into local name and prefix (see AddElementDecl). Split the same
	// way here so a QName lookup composes the identical key.
	var prefix string
	if i := strings.IndexByte(name, ':'); i > 0 {
		prefix = name[:i]
		name = name[i+1:]
	}
	return dtd.LookupElement(name, prefix)
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

// AddChild appends cur as the last child of the DTD, detaching it from any
// previous parent first. It returns an error if cur is nil or if the insertion
// would create a cycle.
func (dtd *DTD) AddChild(cur Node) error {
	return addChild(dtd, cur)
}

// AppendText appends b as a Text child of the DTD.
func (dtd *DTD) AppendText(b []byte) error {
	return appendText(dtd, b)
}

// AddSibling appends cur as the last sibling of the DTD.
func (dtd *DTD) AddSibling(cur Node) error {
	return addSibling(dtd, cur)
}

// Replace swaps the DTD out of its parent, inserting nodes in its place.
func (dtd *DTD) Replace(nodes ...Node) error {
	return replaceNode(dtd, nodes...)
}

// SetTreeDoc sets the owning document of the DTD and its subtree.
func (dtd *DTD) SetTreeDoc(doc *Document) {
	setTreeDoc(dtd, doc)
}

// Free is a no-op; it exists to satisfy the Node interface.
func (dtd *DTD) Free() {}

// ExternalID returns the DTD external ID (PUBLIC identifier).
func (dtd *DTD) ExternalID() string {
	return dtd.externalID
}

// SystemID returns the DTD system ID (SYSTEM identifier).
func (dtd *DTD) SystemID() string {
	return dtd.systemID
}
