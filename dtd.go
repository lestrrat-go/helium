package helium

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/internal/lexicon"
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
// It returns an error if a notation with the same name is already declared.
func (dtd *DTD) AddNotation(name, publicID, systemID string) (*Notation, error) {
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
// Every parameter is validated against the same rules the parser's <!ATTLIST>
// path enforces, so a declaration returned with a nil error always serializes to
// an <!ATTLIST> that a validating parser accepts and recovers equivalently:
//
//   - elem and name must be valid XML Names (the parser's Name production, which
//     permits a colon). name may be a QName; its prefix is split off for keying
//     on the FIRST colon, exactly as AddElementDecl does. A name that STARTS with
//     a colon is rejected: it is a legal Name but not a valid QName, and the
//     parser would split it as (prefix="", local="x") — never matching the whole
//     ":x" key stored here — so it could not round-trip.
//   - name "xml:id" must have atype enum.AttrID.
//   - atype must be a valid enum.Attr* type and def a valid
//     enum.AttrDefaultNone/Required/Implied/Fixed kind.
//   - for enum.AttrEnumeration / enum.AttrNotation, enumValues must be a non-empty
//     list of well-formed, distinct tokens (enumeration tokens are NMTOKENs, which
//     permit a colon; NOTATION tokens are colon-free Names, because a NotationDecl
//     name forbids a colon); it is ignored for other types.
//   - #REQUIRED and #IMPLIED must NOT carry a default value (checked against the
//     value as given, before normalization); enum.AttrDefaultNone and #FIXED carry
//     one (possibly the empty string), validated against atype, and an
//     enumeration/NOTATION default must be one of enumValues. Every character of a
//     value-bearing default must be a legal XML Char, and it must not contain a raw
//     '&' (which the <!ATTLIST> default-value serializer cannot round-trip).
//   - an ID attribute's default must be #IMPLIED or #REQUIRED (never a value or
//     #FIXED), per the ID Attribute Default VC.
//
// A cross-declaration VC that needs the owning element declaration — a NOTATION
// attribute is not allowed on an EMPTY element (§3.3.1) — is out of scope here (it
// is enforced by ValidateDTD), so it is not checked against these parameters.
//
// It returns an error wrapping ErrDuplicateDeclaration when an attribute with the
// same local name, prefix, and owning element is already declared, or an error
// wrapping ErrInvalidArgument for any parameter violation above. The duplicate
// check runs BEFORE the non-identity parameter checks, matching the parser: a
// later declaration of the same attribute is ignored entirely, so its (possibly
// invalid) type/default must not mask the duplicate.
func (dtd *DTD) AddAttributeDecl(elem, name string, atype enum.AttributeType, def enum.AttributeDefault, defvalue string, enumValues Enumeration) (*AttributeDecl, error) {
	// Identity validation: only what is needed to split the QName and build the
	// lookup key, so the duplicate check below can run before the non-identity
	// parameter checks.
	local, prefix, err := validateAttrDeclIdentity(elem, name)
	if err != nil {
		return nil, err
	}

	if _, ok := dtd.LookupAttribute(local, prefix, elem); ok {
		return nil, fmt.Errorf("duplicate attribute %s declared for element %s: %w", name, elem, ErrDuplicateDeclaration)
	}

	// Non-identity validation, and normalize the tokenized default the way the
	// parser normalizes it, so the value stored here matches the value the parser
	// recovers on a round-trip.
	defvalue, err = validateAndNormalizeAttrDecl(name, atype, def, defvalue, enumValues)
	if err != nil {
		return nil, err
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

// validateAttrDeclIdentity validates the parts of an AddAttributeDecl call that
// determine the attribute's identity — the owning element name and the attribute
// QName — and returns the QName split into local + prefix on its FIRST colon,
// mirroring AddElementDecl. It is the minimum needed to build the lookup key, so
// AddAttributeDecl can run its duplicate check before the non-identity parameter
// checks. Every violation wraps ErrInvalidArgument.
func validateAttrDeclIdentity(elem, name string) (local, prefix string, err error) {
	if elem == "" {
		return "", "", fmt.Errorf("element name required: %w", ErrInvalidArgument)
	}
	if name == "" {
		return "", "", fmt.Errorf("attribute name required: %w", ErrInvalidArgument)
	}
	if !isValidNameWithColon(elem) {
		return "", "", fmt.Errorf("invalid element name %q: %w", elem, ErrInvalidArgument)
	}
	if !isValidNameWithColon(name) {
		return "", "", fmt.Errorf("invalid attribute name %q: %w", name, ErrInvalidArgument)
	}
	// A leading-colon name is a legal Name but not a valid QName: the parser splits
	// it as (prefix="", local="x"), which never matches the whole ":x" key kept
	// here, so it could not round-trip.
	if name[0] == ':' {
		return "", "", fmt.Errorf("attribute name %q must not start with a colon: %w", name, ErrInvalidArgument)
	}
	local = name
	if i := strings.IndexByte(name, ':'); i > 0 {
		prefix = name[:i]
		local = name[i+1:]
	}
	// A trailing colon leaves an empty local part (e.g. "p:"): keying it would store
	// a degenerate empty-local declaration, so reject it.
	if local == "" {
		return "", "", fmt.Errorf("attribute name %q must not end with a colon: %w", name, ErrInvalidArgument)
	}
	return local, prefix, nil
}

// validateAndNormalizeAttrDecl checks the non-identity AddAttributeDecl parameters
// against the same rules the parser's <!ATTLIST> path enforces (parser_dtd_attr.go
// addAttributeDecl / TreeBuilder.AttributeDecl) and the declaration-time validity
// constraints (valid_dtd_decl.go), so a declaration that passes here serializes to
// an <!ATTLIST> that parses back and validates equivalently. It returns the
// default value normalized the way the parser normalizes a tokenized default.
// name is the full QName (for the xml:id check). Every violation wraps
// ErrInvalidArgument.
func validateAndNormalizeAttrDecl(name string, atype enum.AttributeType, def enum.AttributeDefault, defvalue string, enumValues Enumeration) (string, error) {
	if name == lexicon.QNameXMLID && atype != enum.AttrID {
		return "", fmt.Errorf("attribute %q must have type ID: %w", name, ErrInvalidArgument)
	}

	switch atype {
	case enum.AttrCDATA, enum.AttrID, enum.AttrIDRef, enum.AttrIDRefs, enum.AttrEntity, enum.AttrEntities, enum.AttrNmtoken, enum.AttrNmtokens, enum.AttrEnumeration, enum.AttrNotation:
	default:
		return "", fmt.Errorf("invalid attribute type: %w", ErrInvalidArgument)
	}

	switch def {
	case enum.AttrDefaultNone, enum.AttrDefaultRequired, enum.AttrDefaultImplied, enum.AttrDefaultFixed:
	default:
		return "", fmt.Errorf("invalid attribute default declaration: %w", ErrInvalidArgument)
	}

	// ID Attribute Default VC (§3.3.1): an ID attribute must be #IMPLIED or
	// #REQUIRED, else ValidateDTD(true) would reject the round-tripped declaration.
	// The predicate is shared with the declaration-time validator so the two paths
	// cannot drift.
	if idAttrDefaultInvalid(atype, def) {
		return "", fmt.Errorf("ID attribute must be declared #IMPLIED or #REQUIRED: %w", ErrInvalidArgument)
	}

	// An enumeration or NOTATION type needs a non-empty list of well-formed,
	// distinct tokens, or the serialized "(...)" would not re-parse. Enumeration
	// tokens are NMTOKENs (parseNmtoken, colon permitted). A NOTATION token names a
	// notation, and a NotationDecl Name forbids a colon (parser_dtd_attr.go
	// parseNotationDecl rejects it as "colons are forbidden from notation names",
	// Namespaces in XML §6): a colon-bearing token could not be declared as a
	// notation and so could not round-trip, so validate it with the colon-free Name
	// production (isValidName), matching the parser's effective rule.
	if atype == enum.AttrEnumeration || atype == enum.AttrNotation {
		if len(enumValues) == 0 {
			return "", fmt.Errorf("enumeration/notation attribute requires at least one token: %w", ErrInvalidArgument)
		}
		seen := make(map[string]struct{}, len(enumValues))
		for _, tok := range enumValues {
			ok := isValidNmtoken(tok)
			if atype == enum.AttrNotation {
				ok = isValidName(tok)
			}
			if !ok {
				return "", fmt.Errorf("invalid enumeration/notation token %q: %w", tok, ErrInvalidArgument)
			}
			if _, dup := seen[tok]; dup {
				return "", fmt.Errorf("duplicate enumeration/notation token %q: %w", tok, ErrInvalidArgument)
			}
			seen[tok] = struct{}{}
		}
	}

	// Default-declaration value-presence rules (DefaultDecl grammar):
	// #REQUIRED / #IMPLIED carry no value; enum.AttrDefaultNone and #FIXED carry
	// an AttValue (possibly the empty string) that must be legal for the type.
	switch def {
	case enum.AttrDefaultRequired, enum.AttrDefaultImplied:
		// Checked against the value AS GIVEN (before normalization): a spaces-only
		// value must not be collapsed into acceptance.
		if defvalue != "" {
			return "", fmt.Errorf("a #REQUIRED/#IMPLIED attribute must not carry a default value: %w", ErrInvalidArgument)
		}
	case enum.AttrDefaultNone, enum.AttrDefaultFixed:
		// A tokenized (non-CDATA) default is space-normalized the way the parser
		// normalizes it (attrNormalizeSpace, #x20 only, NOT strings.Fields, which
		// would also fold TAB/NBSP and diverge from the parser).
		if atype != enum.AttrCDATA {
			defvalue = attrNormalizeSpace(defvalue)
		}
		if err := checkAttrDefaultChars(defvalue); err != nil {
			return "", err
		}
		if err := validateAttributeValueInternal(nil, atype, defvalue); err != nil {
			return "", fmt.Errorf("invalid default value %q: %w: %w", defvalue, err, ErrInvalidArgument)
		}
		if atype == enum.AttrEnumeration || atype == enum.AttrNotation {
			if !slices.Contains(enumValues, defvalue) {
				return "", fmt.Errorf("default value %q is not one of the declared tokens: %w", defvalue, ErrInvalidArgument)
			}
		}
	}
	return defvalue, nil
}

// checkAttrDefaultChars rejects a value-bearing default value that the
// default-value serializer cannot round-trip equivalently:
//
//   - a character outside the XML Char production (a NUL or other C0/C1 control, an
//     out-of-range code point) or invalid UTF-8, either of which would serialize to
//     a "&#x0;"-style reference (or a lone U+FFFD) the parser cannot recover. It
//     reuses the writer's isInCharacterRange predicate so the accept/reject
//     boundary matches serialization exactly.
//   - a literal '&'. The writer escapes '&' as "&amp;", but the parser stores an
//     unsubstituted '&'-producing reference in an <!ATTLIST> default as the literal
//     text "&#38;" (parser_element.go parseAttributeValueInternal), not a bare '&',
//     so the value GROWS on each serialize→parse cycle ("&amp;" → "&amp;#38;" → …)
//     and never stabilizes. The other characters the writer escapes ('<', '>', '"',
//     TAB/LF/CR) each decode back to themselves and round-trip, so only '&' is
//     rejected here. (This is a pre-existing writer/parser escaping asymmetry for
//     '&' in attribute values; guarding the constructor keeps AddAttributeDecl's
//     round-trip contract without touching the shared escaping.)
//
// Every violation wraps ErrInvalidArgument.
func checkAttrDefaultChars(s string) error {
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			return fmt.Errorf("default value %q contains invalid UTF-8: %w", s, ErrInvalidArgument)
		}
		if r == '&' {
			return fmt.Errorf("default value %q must not contain a raw '&' (it does not round-trip through the <!ATTLIST> default-value serializer): %w", s, ErrInvalidArgument)
		}
		if !isInCharacterRange(r) {
			return fmt.Errorf("default value %q contains a character outside the XML Char range (U+%04X): %w", s, r, ErrInvalidArgument)
		}
		i += size
	}
	return nil
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
