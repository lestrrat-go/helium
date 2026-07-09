package helium

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/internal/lexicon"
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

// splitNormalizedTokens splits an already-attribute-normalized list value
// (NMTOKENS / IDREFS / ENTITIES) into its tokens. The list separator in the
// normalized value is exactly one space (#x20): attribute-value normalization
// (XML §3.3.3) folds every literal whitespace character to #x20, so #x20 is the
// only separator; any other whitespace character present came from a character
// reference (e.g. &#9;) and is a token character, not a separator, so it must
// NOT split the value. Splitting on Go's unicode whitespace (strings.Fields)
// would wrongly treat such a character-reference tab/newline as a separator and
// under-report an invalid NMTOKEN (W3C rmt-e2e-20).
func splitNormalizedTokens(s string) []string {
	var toks []string
	for tok := range strings.SplitSeq(s, " ") {
		if tok != "" {
			toks = append(toks, tok)
		}
	}
	return toks
}

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
		// Must match Names production: Name (S Name)*. The value is already
		// attribute-normalized, so the only token separator is a single space
		// (#x20): literal whitespace was folded to #x20, but a whitespace
		// character introduced by a character reference (e.g. &#9;) survives
		// verbatim and is part of a token, not a separator (XML §3.3.3).
		toks := splitNormalizedTokens(defvalue)
		if len(toks) == 0 {
			return errors.New("value must not be empty")
		}
		for _, tok := range toks {
			if !isValidName(tok) {
				return fmt.Errorf("value %q is not a valid Name", tok)
			}
		}
	case enum.AttrNmtoken:
		if !isValidNmtoken(defvalue) {
			return fmt.Errorf("value %q is not a valid NMTOKEN", defvalue)
		}
	case enum.AttrNmtokens:
		// Split on #x20 only — see AttrIDRefs above for why character-reference
		// whitespace must NOT be treated as a token separator.
		toks := splitNormalizedTokens(defvalue)
		if len(toks) == 0 {
			return errors.New("value must not be empty")
		}
		for _, tok := range toks {
			if !isValidNmtoken(tok) {
				return fmt.Errorf("value %q is not a valid NMTOKEN", tok)
			}
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

	// VC: Attribute Value Type (XML §3.1) — every attribute present on the
	// instance must have been declared. libxml2 xmlValidateOneAttribute reports
	// XML_DTD_UNKNOWN_ATTRIBUTE ("No declaration for attribute %s of element %s")
	// for an ordinary attribute with no matching <!ATTLIST>. After the loop above,
	// `seen` holds every declared attribute key ("prefix:local"), the same key
	// form used for present attributes. Namespace-declaration attributes
	// (xmlns / xmlns:*) are stored in nsDefs, not the attribute chain, so they are
	// checked separately by validateElementNamespaceDecls.
	for _, a := range attrs {
		// The xml:base the parser injects onto external-entity elements is not in
		// the source; libxml2 tracks the entity base without a materialized
		// attribute, so it is exempt from this VC. An AUTHORED xml:base is not
		// marked and is validated normally.
		if a.syntheticBase {
			continue
		}
		if !seen[a.Prefix()+":"+a.LocalName()] {
			vctx.addf(ctx, "element %s: no declaration for attribute %s", ename, a.Name())
		}
	}

	validateElementNamespaceDecls(ctx, doc, elem, ename, vctx)
}

// validateElementNamespaceDecls enforces the Fixed Attribute Default VC on the
// namespace-declaration attributes (xmlns and xmlns:*) declared directly on an
// element. An <!ATTLIST> declaration for a namespace declaration is keyed as
// `xmlns` (default declaration) or, for a prefixed declaration `xmlns:p`, as
// local name `p` with declared prefix `xmlns` (matching how the ATTLIST parser
// splits the qualified name). A #FIXED declaration whose value differs from the
// declared value is VC: Fixed Attribute Default (W3C attr08).
//
// helium deliberately does NOT enforce the "must be declared" VC (Attribute
// Value Type) for a namespace declaration that lacks an <!ATTLIST>: a
// namespace-declaration attribute is exempt, so a DTD-validated namespaced
// document need not declare its xmlns attributes. Flagging an undeclared xmlns
// here would over-reject the ordinary case of validating a namespaced document
// against a namespace-agnostic DTD. (This is why W3C hst-bh-005/hst-bh-006 —
// which assert a namespace-UNAWARE processor rejects an undeclared xmlns:* — are
// out of scope for helium's namespace-aware validator.)
func validateElementNamespaceDecls(ctx context.Context, doc *Document, elem *Element, ename string, vctx *validCtx) {
	for _, ns := range elem.Namespaces() {
		declName, declPrefix, label := lexicon.PrefixXMLNS, "", lexicon.PrefixXMLNS
		if p := ns.Prefix(); p != "" {
			declName, declPrefix, label = p, lexicon.PrefixXMLNS, lexicon.PrefixXMLNS+":"+p
		}

		adecl := lookupAttributeDecl(doc, declName, declPrefix, ename)
		if adecl == nil {
			continue
		}
		// VC: Fixed Attribute Default — a #FIXED namespace declaration must
		// match the declared value.
		if adecl.def == enum.AttrDefaultFixed && ns.URI() != adecl.defvalue {
			vctx.addf(ctx, "element %s: attribute %s has value %q but must be %q", ename, label, ns.URI(), adecl.defvalue)
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
		case CDATASectionNode:
			// A CDATA section is character data, never ignorable white space —
			// even when it is empty or whitespace-only, a CDATA section does not
			// match the S nonterminal (XML §2.4/§3.2.1). So any CDATA section in
			// element-only content is a validity error (VC: Element Valid). The
			// sentinel forces a content-model mismatch. (A whitespace-only Text
			// node above may be ignorable; a CDATA section never is.)
			children = append(children, "#text")
		}
	}
	return children
}

// matchContentModel validates a sequence of child element names against
// an ElementContent tree. Returns true if the children match.
//
// The fast path is a greedy recursive descent (matchContent), which is
// correct for the overwhelmingly common deterministic content models the
// XML spec requires (Section 3.2.1, Appendix E): at each position only one
// particle matches the next element, so greedy consumption and
// first-match-wins in choices produce the right answer with no allocation.
//
// Greedy descent cannot backtrack, though, so a greedy `*`/`+` sub-particle
// that consumes maximally can starve a later iteration of an OUTER
// repetition that needs some of those tokens (e.g. (lhs,(rhs,(com|wfc|vc)*)+)
// over lhs rhs com rhs vc). When the greedy pass FAILS we fall back to
// matchContentModelExact, an NFA-style reachable-position acceptor that is
// exact for the regular language the model denotes. The fallback only ever
// turns a greedy reject into an accept when the string is genuinely in the
// language, so it fixes over-rejection without ever accepting a non-member —
// a genuinely invalid content model is still rejected by both passes. The
// fallback is bounded (position-set memoized, no exponential blowup) and runs
// only on the rare greedy miss, so the common path stays fast.
func matchContentModel(content *ElementContent, children []string) bool {
	consumed, ok := matchContent(content, children, 0)
	if ok && consumed == len(children) {
		return true
	}
	return matchContentModelExact(content, children)
}

// matchContentModelExact reports whether children is in the regular language
// denoted by the content model, using exact NFA-style reachability over child
// positions. It backtracks correctly across nested repetitions where the
// greedy matcher cannot.
func matchContentModelExact(content *ElementContent, children []string) bool {
	if content == nil {
		return len(children) == 0
	}
	m := &contentReacher{children: children, memo: make(map[reachKey]map[int]struct{})}
	ends := m.reach(content, 0)
	_, ok := ends[len(children)]
	return ok
}

type reachKey struct {
	content *ElementContent
	pos     int
}

// contentReacher computes, per (content node, start position), the SET of end
// positions reachable by matching that node (honoring its occurrence) starting
// at that position. Results are memoized so the total work is bounded by
// (nodes × positions), guaranteeing polynomial time with no exponential blowup
// even for deeply nested repetition groups.
type contentReacher struct {
	children []string
	memo     map[reachKey]map[int]struct{}
}

// reach returns the set of end positions reachable by matching content
// (including its occurrence indicator) starting at pos.
func (m *contentReacher) reach(content *ElementContent, pos int) map[int]struct{} {
	if content == nil {
		return map[int]struct{}{pos: {}}
	}

	key := reachKey{content: content, pos: pos}
	if cached, ok := m.memo[key]; ok {
		return cached
	}

	var result map[int]struct{}
	switch content.coccur {
	case ElementContentOnce:
		result = m.reachOnce(content, pos)
	case ElementContentOpt:
		result = map[int]struct{}{pos: {}}
		unionInto(result, m.reachOnce(content, pos))
	case ElementContentMult:
		result = m.closure(content, pos, true)
	case ElementContentPlus:
		result = m.closure(content, pos, false)
	default:
		result = map[int]struct{}{}
	}

	m.memo[key] = result
	return result
}

// reachOnce returns the end positions of exactly ONE application of content,
// ignoring its occurrence indicator.
func (m *contentReacher) reachOnce(content *ElementContent, pos int) map[int]struct{} {
	switch content.ctype {
	case ElementContentElement:
		if pos < len(m.children) && m.children[pos] == content.rawName() {
			return map[int]struct{}{pos + 1: {}}
		}
		return map[int]struct{}{}
	case ElementContentSeq:
		cur := map[int]struct{}{pos: {}}
		for _, part := range flattenSeq(content) {
			next := map[int]struct{}{}
			for p := range cur {
				unionInto(next, m.reach(part, p))
			}
			cur = next
			if len(cur) == 0 {
				break
			}
		}
		return cur
	case ElementContentOr:
		res := map[int]struct{}{}
		for _, alt := range flattenOr(content) {
			unionInto(res, m.reach(alt, pos))
		}
		return res
	case ElementContentPCDATA:
		// #PCDATA in element content consumes nothing.
		return map[int]struct{}{pos: {}}
	}
	return map[int]struct{}{}
}

// closure computes the transitive closure of reachOnce from pos. includeStart
// adds pos itself to the result (ElementContentMult, "zero or more"); when
// false (ElementContentPlus, "one or more") at least one application is
// required. The "if not already seen" frontier guard bounds the iteration by
// the number of positions, so an inner term that can match empty cannot loop
// forever.
func (m *contentReacher) closure(content *ElementContent, pos int, includeStart bool) map[int]struct{} {
	result := map[int]struct{}{}
	var frontier []int
	if includeStart {
		result[pos] = struct{}{}
		frontier = append(frontier, pos)
	} else {
		for q := range m.reachOnce(content, pos) {
			if _, seen := result[q]; !seen {
				result[q] = struct{}{}
				frontier = append(frontier, q)
			}
		}
	}
	for len(frontier) > 0 {
		p := frontier[len(frontier)-1]
		frontier = frontier[:len(frontier)-1]
		for q := range m.reachOnce(content, p) {
			if _, seen := result[q]; !seen {
				result[q] = struct{}{}
				frontier = append(frontier, q)
			}
		}
	}
	return result
}

// unionInto adds every element of src into dst.
func unionInto(dst, src map[int]struct{}) {
	for k := range src {
		dst[k] = struct{}{}
	}
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

// flattenSeq collects the parts of a right-nested sequence into a slice.
//
// The tree stores a sequence's continuation in c2. A continuation node is a
// bare Seq with the default Once occurrence, so it is merged into the same
// part list; but a c2 that is a Seq carrying an EXPLICIT occurrence (+/*/?) is
// a distinct grouped sub-particle — e.g. the (rhs,...)+ group in
// (lhs,(rhs,...)+) — and must be kept whole as ONE part, not flattened away
// (which would silently discard its occurrence and corrupt matching). Any
// non-Seq c2 (element leaf, choice group, or occurrence-bearing seq group) is
// likewise appended as a single part. This mirrors the sub-group test the DTD
// writer uses (writer_dtd.go).
func flattenSeq(content *ElementContent) []*ElementContent {
	var parts []*ElementContent
	for cur := content; ; {
		if cur.c1 != nil {
			parts = append(parts, cur.c1)
		}
		c2 := cur.c2
		if c2 != nil && c2.ctype == ElementContentSeq && c2.coccur == ElementContentOnce {
			cur = c2
			continue
		}
		if c2 != nil {
			parts = append(parts, c2)
		}
		break
	}
	return parts
}

// flattenOr collects the alternatives of a right-nested choice into a slice.
// A c2 that is a bare Once choice is a continuation merged into the same
// alternative list; a c2 that is a choice carrying an explicit occurrence, or
// any non-choice node, is a distinct alternative kept whole (see flattenSeq).
func flattenOr(content *ElementContent) []*ElementContent {
	var alts []*ElementContent
	for cur := content; ; {
		if cur.c1 != nil {
			alts = append(alts, cur.c1)
		}
		c2 := cur.c2
		if c2 != nil && c2.ctype == ElementContentOr && c2.coccur == ElementContentOnce {
			cur = c2
			continue
		}
		if c2 != nil {
			alts = append(alts, c2)
		}
		break
	}
	return alts
}
