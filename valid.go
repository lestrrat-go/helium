package helium

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/internal/xmlchar"
)

func newElementContent(name string, ctype ElementContentType) (*ElementContent, error) {
	var prefix string
	var local string
	switch ctype {
	case ElementContentElement:
		if name == "" {
			return nil, errors.New("ElementContent (element) must have name")
		}
		if p, l, ok := strings.Cut(name, ":"); ok {
			prefix = p
			local = l
		} else {
			local = name
		}
	case ElementContentPCDATA, ElementContentSeq, ElementContentOr:
		if name != "" {
			return nil, errors.New("ElementContent (element) must NOT have name")
		}
	default:
		return nil, errors.New("invalid element content type")
	}

	ret := ElementContent{
		ctype:  ctype,
		coccur: ElementContentOnce,
		name:   local,
		prefix: prefix,
	}

	return &ret, nil
}

// rawName returns the element leaf's raw qualified name as declared in the
// DTD (prefix:local when a prefix is present, else the local name). DTD
// content-model validation is NOT namespace-aware: the prefix is an opaque
// part of the name and must be matched literally against the element tag as
// written, mirroring libxml2 (which compares node->name, the full qualified
// name).
func (c *ElementContent) rawName() string {
	if c.prefix != "" {
		return c.prefix + ":" + c.name
	}
	return c.name
}

func (elem *ElementContent) copyElementContent() *ElementContent {
	if elem == nil {
		return nil
	}
	ret := &ElementContent{}
	ret.ctype = elem.ctype
	ret.coccur = elem.coccur
	ret.name = elem.name
	ret.prefix = elem.prefix

	if elem.c1 != nil {
		ret.c1 = elem.c1.copyElementContent()
	}
	if ret.c1 != nil {
		ret.c1.parent = ret
	}

	if elem.c2 != nil {
		prev := ret
		for cur := elem.c2; cur != nil; {
			var tmp ElementContent
			tmp.name = cur.name
			tmp.ctype = cur.ctype
			tmp.coccur = cur.coccur
			tmp.prefix = cur.prefix
			prev.c2 = &tmp
			if cur.c1 != nil {
				tmp.c1 = cur.c1.copyElementContent()
			}
			if tmp.c1 != nil {
				tmp.c1.parent = ret
			}

			prev = &tmp
			cur = cur.c2
		}
	}
	return ret
}

// isValidName checks whether s matches the XML Name production:
// Name ::= NameStartChar (NameChar)*
func isValidName(s string) bool {
	if s == "" {
		return false
	}
	r, size := utf8.DecodeRuneInString(s)
	if (r == utf8.RuneError && size == 1) || !isValidNameStartChar(r) {
		return false
	}
	for i := size; i < len(s); {
		r, size = utf8.DecodeRuneInString(s[i:])
		if (r == utf8.RuneError && size == 1) || !isValidNameChar(r) {
			return false
		}
		i += size
	}
	return true
}

// isValidNmtoken checks whether s matches the XML Nmtoken production:
// Nmtoken ::= (NameChar)+
//
// DTD validation is NOT namespace-aware, so an Nmtoken uses the full XML 1.0
// NameChar production, which INCLUDES the colon (e.g. `x:image` is a valid
// NMTOKEN). Unlike the ID/IDREF/ENTITY Name checks, this must not use the
// namespace-aware NCNameChar production.
func isValidNmtoken(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if (r == utf8.RuneError && size == 1) || !isValidNmtokenChar(r) {
			return false
		}
		i += size
	}
	return true
}

// isValidNameStartChar checks the XML 1.0 NameStartChar production (without colon).
func isValidNameStartChar(r rune) bool { return xmlchar.IsNCNameStartChar(r) }

// isValidNameChar checks the XML 1.0 NameChar production (without colon).
func isValidNameChar(r rune) bool { return xmlchar.IsNCNameChar(r) }

// isValidNmtokenChar checks the XML 1.0 NameChar production INCLUDING the colon.
// NameChar ::= NCNameChar | ":"
func isValidNmtokenChar(r rune) bool { return xmlchar.IsNCNameChar(r) || r == ':' }

// validateAttributeValueInternal validates that defvalue is legal for the
// declared attribute type. Mirrors xmlValidateAttributeDecl() in libxml2.
func validateAttributeValueInternal(_ *Document, typ enum.AttributeType, defvalue string) error {
	switch typ {
	case enum.AttrCDATA:
		// Any string is valid for CDATA
		return nil
	case enum.AttrID, enum.AttrIDRef, enum.AttrEntity:
		// Must match the Name production
		if !isValidName(defvalue) {
			return fmt.Errorf("value %q is not a valid Name", defvalue)
		}
	case enum.AttrIDRefs, enum.AttrEntities:
		// Must match Names production: Name (S Name)*
		for tok := range strings.FieldsSeq(defvalue) {
			if !isValidName(tok) {
				return fmt.Errorf("value %q is not a valid Name", tok)
			}
		}
		if len(strings.Fields(defvalue)) == 0 {
			return errors.New("value must not be empty")
		}
	case enum.AttrNmtoken:
		if !isValidNmtoken(defvalue) {
			return fmt.Errorf("value %q is not a valid NMTOKEN", defvalue)
		}
	case enum.AttrNmtokens:
		for tok := range strings.FieldsSeq(defvalue) {
			if !isValidNmtoken(tok) {
				return fmt.Errorf("value %q is not a valid NMTOKEN", tok)
			}
		}
		if len(strings.Fields(defvalue)) == 0 {
			return errors.New("value must not be empty")
		}
	case enum.AttrEnumeration, enum.AttrNotation:
		// These are validated against the enumeration tree at a higher
		// level; the value must match one of the declared tokens, but
		// that information isn't available here. Just check that it's
		// a valid Nmtoken (which is the baseline requirement).
		if !isValidNmtoken(defvalue) {
			return fmt.Errorf("value %q is not a valid NMTOKEN", defvalue)
		}
	}
	return nil
}

// standaloneNormAttr identifies a specified attribute (by owning element name and
// attribute name, as written in the source) whose value was altered by tokenized
// attribute-value normalization driven by an external-subset declaration. It backs
// the VC: Standalone Document Declaration normalization check.
type standaloneNormAttr struct {
	elem string
	attr string
}

// ErrDTDValidationFailed is returned by DTD validation when the document
// does not conform to the DTD. Individual validation errors are delivered
// to the configured [ErrorHandler].
var ErrDTDValidationFailed = errors.New("dtd: validation failed")

// validCtx carries validation state through the document walk.
type validCtx struct {
	handler ErrorHandler
	failed  bool
	ids     map[string]bool // ID values seen (uniqueness check)
	idrefs  map[string]bool // IDREF values to resolve (cross-ref check)
}

func (vc *validCtx) addf(ctx context.Context, format string, args ...any) {
	vc.failed = true
	vc.handler.Handle(ctx, fmt.Errorf(format, args...))
}

// docDTDs returns the DTDs to search for declarations, internal subset first.
// Both subsets are always searched, independent of standalone: a validating
// processor reads the external subset and uses its element/attribute
// declarations to validate document structure regardless of the standalone
// declaration (libxml2 xmlGetDtdElementDesc/xmlGetDtdAttrDesc search both). The
// Standalone Document Declaration VC (§2.9) — that a standalone="yes" document
// must not DEPEND on external declarations for attribute defaults or value
// normalization — is enforced separately by checkStandaloneExternalDefaults and
// checkStandaloneExternalNormalization, not by hiding the external declarations.
func docDTDs(doc *Document) []*DTD {
	var dtds []*DTD
	if doc.intSubset != nil {
		dtds = append(dtds, doc.intSubset)
	}
	if doc.extSubset != nil {
		dtds = append(dtds, doc.extSubset)
	}
	return dtds
}

// lookupDTDEntity searches both intSubset and extSubset for a general entity
// declaration, regardless of the document's standalone status. DTD validity
// (VC: Entity Name) requires the referenced entity to be declared in either
// subset — an unparsed entity in the external subset must be found even for a
// standalone="yes" document — so this deliberately does NOT gate on standalone
// the way Document.GetEntity does.
func lookupDTDEntity(doc *Document, name string) (*Entity, bool) {
	for _, dtd := range docDTDs(doc) {
		if ent, ok := dtd.LookupEntity(name); ok {
			return ent, true
		}
	}
	return nil, false
}

// lookupElementDecl searches both intSubset and extSubset for an element
// declaration. DTD validation compares raw qualified names, not namespaces, so a
// prefixed element requires an <!ELEMENT> declaration for the SAME QName — there is
// no fallback from `p:r` to an unprefixed `r` declaration (for an unprefixed
// element, prefix is "" and this is the only lookup).
func lookupElementDecl(doc *Document, name, prefix string) (*ElementDecl, *DTD) {
	for _, dtd := range docDTDs(doc) {
		if edecl, ok := dtd.LookupElement(name, prefix); ok {
			return edecl, dtd
		}
	}
	return nil, nil
}

// validateDocument validates a parsed document against its DTD.
// This is the equivalent of libxml2's xmlValidateDocument.
func validateDocument(ctx context.Context, doc *Document, handler ErrorHandler) error {
	vctx := &validCtx{
		handler: handler,
		ids:     make(map[string]bool),
		idrefs:  make(map[string]bool),
	}

	// VC: Element Declared (XML §3.2) / libxml2 XML_DTD_NO_DTD "no DTD found!".
	// A validating processor must report a validity error for a document that
	// has neither an internal nor an external subset — nothing declares its
	// elements. This path is reached only under ValidateDTD(true).
	if doc.intSubset == nil && doc.extSubset == nil {
		vctx.addf(ctx, "no DTD found")
		return ErrDTDValidationFailed
	}

	// Check that the root element name matches the DTD name
	if root := doc.DocumentElement(); root != nil {
		var dtdName string
		if doc.intSubset != nil {
			dtdName = doc.intSubset.name
		}
		if dtdName == "" && doc.extSubset != nil {
			dtdName = doc.extSubset.name
		}
		// The DOCTYPE name is the root element's qualified name (e.g. `p:r`), so
		// compare against the element's QName, not just its local part.
		if dtdName != "" && root.Name() != dtdName {
			vctx.addf(ctx, "root element name %q does not match DTD name %q", root.Name(), dtdName)
		}
	}

	// Validate the DTD declarations themselves (declaration-consistency VCs)
	// before walking the instance tree.
	validateDTDDeclarations(ctx, doc, vctx)

	// VC: Standalone Document Declaration (XML §2.9) — attribute values normalized
	// by an external-subset tokenized-type declaration (recorded during parsing).
	checkStandaloneExternalNormalization(ctx, doc, vctx)

	// Walk the document tree and validate each element. A cycle in the tree
	// (ErrWalkCycle) leaves the walk partial, so the document cannot be
	// considered valid.
	if err := Walk(doc, NodeWalkerFunc(func(n Node) error {
		if elem, ok := AsNode[*Element](n); ok {
			validateOneElement(ctx, doc, elem, vctx)
		}
		return nil
	})); err != nil {
		vctx.addf(ctx, "document tree traversal failed: %s", err)
	}

	// Cross-reference check: every IDREF must match an existing ID
	validateDocumentFinal(ctx, vctx)

	if vctx.failed {
		return ErrDTDValidationFailed
	}
	return nil
}

// validateOneElement checks a single element against DTD declarations.
func validateOneElement(ctx context.Context, doc *Document, elem *Element, vctx *validCtx) {
	name := elem.LocalName()

	// VC: Standalone Document Declaration (XML §2.9) — an attribute that takes a
	// default value from an ATTLIST declaration in the external subset changes the
	// document, so it is invalid in a standalone="yes" document. Checked before the
	// element-declaration lookup so it fires independently of whether the element
	// itself carries an <!ELEMENT> declaration (an ATTLIST may exist without one).
	checkStandaloneExternalDefaults(ctx, doc, elem, vctx)

	// VC: Standalone Document Declaration — if standalone="yes" and the element
	// has element-only content declared in the external subset, whitespace-only
	// text nodes directly within it are a validity error (the external subset
	// makes that whitespace ignorable). This consults the external subset
	// directly, so it runs independent of the declaration lookup below — the
	// element is normally FOUND (both subsets are searched regardless of
	// standalone), and the violation must still be reported.
	if doc.standalone == StandaloneExplicitYes && doc.extSubset != nil {
		checkStandaloneWhitespace(ctx, doc.extSubset, elem, name, vctx)
	}

	edecl, dtd := lookupElementDecl(doc, name, elem.Prefix())
	if edecl == nil {
		vctx.addf(ctx, "element %s: no declaration found", name)
		return
	}

	// Validate attributes against their declarations
	validateElementAttributes(ctx, doc, elem, edecl, vctx)

	// Validate element content model
	validateElementContent(ctx, dtd, elem, edecl, vctx)
}

// validateElementAttributes checks that:
// - All #REQUIRED attributes are present
// - #FIXED attributes have the correct value
// - No undeclared attributes (if element is fully declared)
// - ID values are unique across the document
// - IDREF/IDREFS values are recorded for cross-reference checking
func validateElementAttributes(ctx context.Context, doc *Document, elem *Element, _ *ElementDecl, vctx *validCtx) {
	// ATTLIST declarations are keyed by the element's declared QName; match by the
	// instance element's QName so a declaration for `p:r` is not applied to `<r>`.
	ename := elem.Name()
	attrs := elem.Attributes()

	// Build a set of present attributes, keyed by full QName (prefix + local), so a
	// declaration for `p:id` matches only a `p:id` instance attribute, not `q:id`.
	present := make(map[string]string, len(attrs))
	for _, a := range attrs {
		present[a.Prefix()+":"+a.LocalName()] = a.Value()
	}

	// Check all declared attributes from both subsets, dedup by QName
	seen := make(map[string]bool)
	for _, dtd := range docDTDs(doc) {
		for _, adecl := range dtd.AttributesForElement(ename) {
			akey := adecl.prefix + ":" + adecl.name
			aname := adecl.name
			if adecl.prefix != "" {
				aname = adecl.prefix + ":" + adecl.name
			}
			if seen[akey] {
				continue
			}
			seen[akey] = true

			val, found := present[akey]

			switch adecl.def {
			case enum.AttrDefaultRequired:
				if !found {
					vctx.addf(ctx, "element %s: attribute %s is required", ename, aname)
				}
			case enum.AttrDefaultFixed:
				if found && val != adecl.defvalue {
					vctx.addf(ctx, "element %s: attribute %s has value %q but must be %q", ename, aname, val, adecl.defvalue)
				}
			}

			// Validate attribute value against its type (if present)
			if found {
				if err := validateAttributeValueInternal(doc, adecl.atype, val); err != nil {
					vctx.addf(ctx, "element %s: attribute %s: %s", ename, aname, err)
				}

				// Check enumeration value against declared tokens
				if adecl.atype == enum.AttrEnumeration && len(adecl.tree) > 0 {
					if !slices.Contains(adecl.tree, val) {
						vctx.addf(ctx, "element %s: attribute %s value %q is not among the enumerated set", ename, aname, val)
					}
				}

				// Track ID uniqueness and collect IDREFs
				switch adecl.atype {
				case enum.AttrID:
					if vctx.ids[val] {
						vctx.addf(ctx, "element %s: duplicate ID %q", ename, val)
					} else {
						vctx.ids[val] = true
					}
				case enum.AttrIDRef:
					vctx.idrefs[val] = true
				case enum.AttrIDRefs:
					for ref := range strings.FieldsSeq(val) {
						vctx.idrefs[ref] = true
					}
				case enum.AttrEntity:
					ent, ok := lookupDTDEntity(doc, val)
					if !ok {
						vctx.addf(ctx, "element %s: attribute %s references undeclared entity %q", ename, aname, val)
					} else if ent.EntityType() != enum.ExternalGeneralUnparsedEntity {
						vctx.addf(ctx, "element %s: attribute %s references entity %q which is not unparsed", ename, aname, val)
					}
				case enum.AttrEntities:
					for entName := range strings.FieldsSeq(val) {
						ent, ok := lookupDTDEntity(doc, entName)
						if !ok {
							vctx.addf(ctx, "element %s: attribute %s references undeclared entity %q", ename, aname, entName)
						} else if ent.EntityType() != enum.ExternalGeneralUnparsedEntity {
							vctx.addf(ctx, "element %s: attribute %s references entity %q which is not unparsed", ename, aname, entName)
						}
					}
				case enum.AttrNotation:
					notFound := true
					for _, dtd := range docDTDs(doc) {
						if _, ok := dtd.LookupNotation(val); ok {
							notFound = false
							break
						}
					}
					if notFound {
						vctx.addf(ctx, "element %s: attribute %s references undeclared notation %q", ename, aname, val)
					}
					// VC: Notation Attributes — the value must be one of the
					// notation names listed in this attribute's declaration.
					if len(adecl.tree) > 0 && !slices.Contains(adecl.tree, val) {
						vctx.addf(ctx, "element %s: attribute %s value %q is not among the enumerated notations", ename, aname, val)
					}
				}
			}
		}
	}
}

// validateDocumentFinal performs post-walk cross-reference checks.
// Every IDREF value must match an ID declared somewhere in the document.
func validateDocumentFinal(ctx context.Context, vctx *validCtx) {
	for ref := range vctx.idrefs {
		if !vctx.ids[ref] {
			vctx.addf(ctx, "IDREF %q references unknown ID", ref)
		}
	}
}

// checkStandaloneWhitespace checks whether an element declared in the
// external subset with element-only content contains whitespace-only text
// nodes. Per the VC: Standalone Document Declaration (XML §2.9), a
// standalone="yes" document must not contain such whitespace when the
// element declaration comes from the external subset.
func checkStandaloneWhitespace(ctx context.Context, extSubset *DTD, elem *Element, name string, vctx *validCtx) {
	// DTD validation is prefix-literal, so match the element's exact QName with no
	// fallback from a prefixed element to an unprefixed declaration.
	extDecl, ok := extSubset.LookupElement(name, elem.Prefix())
	if !ok || extDecl.decltype != enum.ElementElementType {
		return
	}
	for child := range Children(elem) {
		// Whitespace-only character data — whether written as ordinary text or a
		// CDATA section — is ignorable only because the external element-content
		// declaration says so, so either form violates the standalone constraint.
		switch child.Type() {
		case TextNode, CDATASectionNode:
			if xmlchar.IsAllSpace(child.Content()) {
				vctx.addf(ctx, "standalone: element %s declared in the external subset contains white spaces nodes", name)
				return
			}
		}
	}
}

// checkStandaloneExternalDefaults implements part of the VC: Standalone Document
// Declaration (XML §2.9): in a standalone="yes" document it is a validity error
// for an attribute to take a default value supplied by an ATTLIST declaration in
// the external subset, because omitting the external subset would change whether
// the attribute is present. Mirrors libxml2's XML_DTD_STANDALONE_DEFAULTED report.
func checkStandaloneExternalDefaults(ctx context.Context, doc *Document, elem *Element, vctx *validCtx) {
	if doc.standalone != StandaloneExplicitYes {
		return
	}
	// Drive the check from the ATTLIST declarations, not materialized default
	// Attribute nodes: ValidateDTD(true) does not imply DefaultDTDAttributes(true),
	// so an external default may never be materialized on the instance, yet the
	// declaration still makes the document depend on external markup. Origin is
	// recorded per-declaration (AttributeDecl.external), because an external-PE-
	// supplied ATTLIST is registered in the internal subset's table yet is still
	// external markup. An internal-subset declaration takes precedence (§3.3), and
	// dtdSubsets orders internal first, so the first-seen declaration per attribute
	// wins. ATTLIST declarations are keyed by the element's declared QName, so match
	// by the instance element's QName (a declaration for `p:r` does not apply to `<r>`).
	ename := elem.Name()
	seen := make(map[string]struct{})
	for _, dtd := range dtdSubsets(doc) {
		for _, adecl := range dtd.AttributesForElement(ename) {
			key := adecl.name + ":" + adecl.prefix
			if _, dup := seen[key]; dup {
				continue
			}
			seen[key] = struct{}{}
			if !attrHasDefaultValue(adecl.def) || !adecl.external {
				continue
			}
			// The external default is a validity error unless the instance supplies
			// the attribute explicitly (a value present in the source, not the
			// DTD-supplied default).
			if attrExplicitlySpecified(elem, adecl.name, adecl.prefix) {
				continue
			}
			name := adecl.name
			if adecl.prefix != "" {
				name = adecl.prefix + ":" + adecl.name
			}
			vctx.addf(ctx, "standalone: attribute %s on %s defaulted from external subset", name, ename)
		}
	}
}

// attrExplicitlySpecified reports whether the element carries the named attribute
// as a value present in the source instance — a materialized node that is not a
// DTD-supplied default.
func attrExplicitlySpecified(elem *Element, name, prefix string) bool {
	for _, a := range elem.Attributes() {
		if a.LocalName() == name && a.Prefix() == prefix {
			return !a.IsDefault()
		}
	}
	return false
}

// checkStandaloneExternalNormalization implements part of the VC: Standalone
// Document Declaration (XML §2.9): in a standalone="yes" document it is a validity
// error for the normalized value of a specified attribute to differ from its
// literal value when the attribute's (tokenized) type comes from a declaration in
// the external subset — omitting the external subset would leave the attribute
// CDATA and preserve its whitespace. The parser records each such attribute during
// attribute-value normalization (it alone still holds the pre-normalization
// value); this flushes those records as validity errors. Mirrors libxml2's
// XML_DTD_NOT_STANDALONE normalization report.
func checkStandaloneExternalNormalization(ctx context.Context, doc *Document, vctx *validCtx) {
	if doc.standalone != StandaloneExplicitYes {
		return
	}
	for _, v := range doc.standaloneNormAttrs {
		vctx.addf(ctx, "standalone: normalization of attribute %s on %s by external subset declaration", v.attr, v.elem)
	}
}

// validateElementContent validates that the element's children match the
// declared content model.
func validateElementContent(ctx context.Context, _ *DTD, elem *Element, edecl *ElementDecl, vctx *validCtx) {
	ename := elem.LocalName()

	switch edecl.decltype {
	case enum.EmptyElementType:
		// EMPTY elements must have no children
		if elem.FirstChild() != nil {
			vctx.addf(ctx, "element %s: declared EMPTY but has content", ename)
		}
	case enum.AnyElementType:
		// ANY allows anything
	case enum.MixedElementType:
		// Mixed content: (#PCDATA | elem1 | elem2 | ...)*
		// All child elements must be in the declared list
		validateMixedContent(ctx, elem, edecl.content, vctx)
	case enum.ElementElementType:
		// Element content: must match the content model exactly
		children := collectChildElements(elem)
		if !matchContentModel(edecl.content, children) {
			vctx.addf(ctx, "element %s: content does not match declared content model", ename)
		}
	}
}

// validateMixedContent validates children of a mixed-content element.
// In mixed content (#PCDATA | a | b)*, text nodes are always allowed,
// and element children must appear in the declared list.
func validateMixedContent(ctx context.Context, elem *Element, content *ElementContent, vctx *validCtx) {
	if content == nil {
		return
	}

	// Collect allowed element names from the content model
	allowed := collectMixedNames(content)

	for child := range Children(elem) {
		switch child.Type() {
		case TextNode, CDATASectionNode, EntityRefNode, CommentNode, ProcessingInstructionNode:
			// Always allowed in mixed content
		case ElementNode:
			if ce, ok := AsNode[*Element](child); ok {
				cname := ce.Name()
				if _, ok := allowed[cname]; !ok {
					vctx.addf(ctx, "element %s: child element %s not allowed in mixed content", elem.LocalName(), cname)
				}
			}
		}
	}
}

// collectMixedNames extracts the set of allowed element names from a
// mixed content declaration (#PCDATA | a | b | ...)*
func collectMixedNames(content *ElementContent) map[string]struct{} {
	names := make(map[string]struct{})
	collectMixedNamesRecurse(content, names)
	return names
}

func collectMixedNamesRecurse(content *ElementContent, names map[string]struct{}) {
	if content == nil {
		return
	}
	if content.ctype == ElementContentElement {
		names[content.rawName()] = struct{}{}
		return
	}
	collectMixedNamesRecurse(content.c1, names)
	collectMixedNamesRecurse(content.c2, names)
}

// collectChildElements returns the raw qualified names (prefix:local, as
// written) of an element's child elements, ignoring text nodes, comments, PIs,
// etc. DTD content-model matching is prefix-literal, not namespace-aware, so the
// name is taken verbatim from the child tag.
func collectChildElements(elem *Element) []string {
	var children []string
	for child := range Children(elem) {
		switch child.Type() {
		case ElementNode:
			if ce, ok := AsNode[*Element](child); ok {
				children = append(children, ce.Name())
			}
		case TextNode:
			// In element-only content, whitespace text is allowed (ignorable
			// whitespace) but non-whitespace text is an error. We skip
			// whitespace-only text and treat non-whitespace as a mismatch
			// that will be caught by the content model check.
			if !xmlchar.IsAllSpace(child.Content()) {
				// Use a sentinel to cause content model mismatch
				children = append(children, "#text")
			}
		}
	}
	return children
}

// matchContentModel validates a sequence of child element names against
// an ElementContent tree. Returns true if the children match.
//
// This uses a greedy recursive descent approach, which is correct for
// deterministic content models as required by the XML spec (Section 3.2.1,
// Appendix E). At each position only one particle can match the next
// element, so greedy consumption and first-match-wins in choices always
// produce the correct result. Non-deterministic content models (which
// violate the XML spec) may be matched incorrectly.
func matchContentModel(content *ElementContent, children []string) bool {
	consumed, ok := matchContent(content, children, 0)
	return ok && consumed == len(children)
}

// matchContent tries to match children[pos:] against the content model,
// returning the number of children consumed and whether it matched.
func matchContent(content *ElementContent, children []string, pos int) (int, bool) {
	if content == nil {
		return 0, pos >= len(children)
	}

	switch content.ctype {
	case ElementContentElement:
		return matchElement(content, children, pos)
	case ElementContentSeq:
		return matchSeq(content, children, pos)
	case ElementContentOr:
		return matchOr(content, children, pos)
	case ElementContentPCDATA:
		// #PCDATA in element content — shouldn't appear in element-only
		return 0, true
	}
	return 0, false
}

// matchElement matches a single named element against children[pos].
func matchElement(content *ElementContent, children []string, pos int) (int, bool) {
	name := content.rawName()
	switch content.coccur {
	case ElementContentOnce:
		if pos < len(children) && children[pos] == name {
			return 1, true
		}
		return 0, false
	case ElementContentOpt:
		if pos < len(children) && children[pos] == name {
			return 1, true
		}
		return 0, true // optional: 0 matches is ok
	case ElementContentMult:
		// Zero or more
		count := 0
		for pos+count < len(children) && children[pos+count] == name {
			count++
		}
		return count, true
	case ElementContentPlus:
		// One or more
		if pos >= len(children) || children[pos] != name {
			return 0, false
		}
		count := 1
		for pos+count < len(children) && children[pos+count] == name {
			count++
		}
		return count, true
	}
	return 0, false
}

// matchSeq matches a sequence (a, b, c) against children[pos:].
func matchSeq(content *ElementContent, children []string, pos int) (int, bool) {
	// Flatten the sequence: the tree stores sequences as right-nested c2 chains
	parts := flattenSeq(content)

	matchOnce := func(startPos int) (int, bool) {
		current := startPos
		for _, part := range parts {
			consumed, ok := matchContent(part, children, current)
			if !ok {
				return 0, false
			}
			current += consumed
		}
		return current - startPos, true
	}

	switch content.coccur {
	case ElementContentOnce:
		consumed, ok := matchOnce(pos)
		if !ok {
			return 0, false
		}
		return consumed, true
	case ElementContentOpt:
		consumed, ok := matchOnce(pos)
		if !ok {
			return 0, true // optional
		}
		return consumed, true
	case ElementContentMult:
		total := 0
		for {
			consumed, ok := matchOnce(pos + total)
			if !ok || consumed == 0 {
				break
			}
			total += consumed
		}
		return total, true
	case ElementContentPlus:
		consumed, ok := matchOnce(pos)
		if !ok {
			return 0, false
		}
		total := consumed
		for {
			consumed, ok = matchOnce(pos + total)
			if !ok || consumed == 0 {
				break
			}
			total += consumed
		}
		return total, true
	}
	return 0, false
}

// matchOr matches a choice (a | b | c) against children[pos:].
func matchOr(content *ElementContent, children []string, pos int) (int, bool) {
	alternatives := flattenOr(content)

	matchOnce := func(startPos int) (int, bool) {
		for _, alt := range alternatives {
			consumed, ok := matchContent(alt, children, startPos)
			if ok && consumed > 0 {
				return consumed, true
			}
		}
		// Check for zero-length matches (e.g., optional alternatives)
		for _, alt := range alternatives {
			consumed, ok := matchContent(alt, children, startPos)
			if ok && consumed == 0 {
				return 0, true
			}
		}
		return 0, false
	}

	switch content.coccur {
	case ElementContentOnce:
		return matchOnce(pos)
	case ElementContentOpt:
		consumed, ok := matchOnce(pos)
		if !ok {
			return 0, true
		}
		return consumed, true
	case ElementContentMult:
		total := 0
		for {
			consumed, ok := matchOnce(pos + total)
			if !ok || consumed == 0 {
				break
			}
			total += consumed
		}
		return total, true
	case ElementContentPlus:
		consumed, ok := matchOnce(pos)
		if !ok {
			return 0, false
		}
		total := consumed
		for consumed > 0 {
			consumed, ok = matchOnce(pos + total)
			if !ok {
				break
			}
			total += consumed
		}
		return total, true
	}
	return 0, false
}

// flattenSeq collects all parts of a right-nested sequence into a slice.
func flattenSeq(content *ElementContent) []*ElementContent {
	var parts []*ElementContent
	for cur := content; cur != nil && cur.ctype == ElementContentSeq; cur = cur.c2 {
		if cur.c1 != nil {
			parts = append(parts, cur.c1)
		}
		// If c2 is not a seq, it's the last element
		if cur.c2 != nil && cur.c2.ctype != ElementContentSeq {
			parts = append(parts, cur.c2)
			break
		}
	}
	return parts
}

// flattenOr collects all alternatives of a right-nested choice into a slice.
func flattenOr(content *ElementContent) []*ElementContent {
	var alts []*ElementContent
	for cur := content; cur != nil && cur.ctype == ElementContentOr; cur = cur.c2 {
		if cur.c1 != nil {
			alts = append(alts, cur.c1)
		}
		if cur.c2 != nil && cur.c2.ctype != ElementContentOr {
			alts = append(alts, cur.c2)
			break
		}
	}
	return alts
}
