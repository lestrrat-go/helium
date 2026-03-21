package helium

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/pdebug"
)

func newElementContent(name string, ctype ElementContentType) (*ElementContent, error) {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START newElementContent '%s' (type = %d)", name, ctype)
		defer g.IRelease("END newElementContent")
	}
	var prefix string
	var local string
	switch ctype {
	case ElementContentElement:
		if name == "" {
			return nil, errors.New("ElementContent (element) must have name")
		}
		if i := strings.IndexByte(name, ':'); i > -1 {
			prefix = name[:i]
			local = name[i+1:]
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
	if r == utf8.RuneError || !isValidNameStartChar(r) {
		return false
	}
	for i := size; i < len(s); {
		r, size = utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError || !isValidNameChar(r) {
			return false
		}
		i += size
	}
	return true
}

// isValidNmtoken checks whether s matches the XML Nmtoken production:
// Nmtoken ::= (NameChar)+
func isValidNmtoken(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError || !isValidNameChar(r) {
			return false
		}
		i += size
	}
	return true
}

// isValidNameStartChar checks the XML 1.0 NameStartChar production (without colon).
// NameStartChar ::= ":" | [A-Z] | "_" | [a-z] | [#xC0-#xD6] | [#xD8-#xF6]
//
//	| [#xF8-#x2FF] | [#x370-#x37D] | [#x37F-#x1FFF] | [#x200C-#x200D]
//	| [#x2070-#x218F] | [#x2C00-#x2FEF] | [#x3001-#xD7FF]
//	| [#xF900-#xFDCF] | [#xFDF0-#xFFFD] | [#x10000-#xEFFFF]
func isValidNameStartChar(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || r == '_' ||
		(r >= 0xC0 && r <= 0xD6) || (r >= 0xD8 && r <= 0xF6) ||
		(r >= 0xF8 && r <= 0x2FF) || (r >= 0x370 && r <= 0x37D) ||
		(r >= 0x37F && r <= 0x1FFF) || (r >= 0x200C && r <= 0x200D) ||
		(r >= 0x2070 && r <= 0x218F) || (r >= 0x2C00 && r <= 0x2FEF) ||
		(r >= 0x3001 && r <= 0xD7FF) || (r >= 0xF900 && r <= 0xFDCF) ||
		(r >= 0xFDF0 && r <= 0xFFFD) || (r >= 0x10000 && r <= 0xEFFFF)
}

// isValidNameChar checks the XML 1.0 NameChar production (without colon).
// NameChar ::= NameStartChar | "-" | "." | [0-9] | #xB7
//
//	| [#x0300-#x036F] | [#x203F-#x2040]
func isValidNameChar(r rune) bool {
	return isValidNameStartChar(r) ||
		(r >= '0' && r <= '9') || r == '-' || r == '.' ||
		r == 0xB7 || (r >= 0x0300 && r <= 0x036F) || (r >= 0x203F && r <= 0x2040)
}

// validateAttributeValueInternal validates that defvalue is legal for the
// declared attribute type. Mirrors xmlValidateAttributeDecl() in libxml2.
func validateAttributeValueInternal(doc *Document, typ enum.AttributeType, defvalue string) error {
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
		for _, tok := range strings.Fields(defvalue) {
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
		for _, tok := range strings.Fields(defvalue) {
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

// ValidationError represents a validity constraint violation.
// It does not stop parsing; the parser continues and collects errors.
type ValidationError struct {
	Errors []string
}

func (e *ValidationError) Error() string {
	return strings.Join(e.Errors, "; ")
}

func (e *ValidationError) addf(format string, args ...any) {
	e.Errors = append(e.Errors, fmt.Sprintf(format, args...))
}

func (e *ValidationError) hasErrors() bool {
	return len(e.Errors) > 0
}

// validCtx carries validation state through the document walk.
type validCtx struct {
	ve     *ValidationError
	ids    map[string]bool // ID values seen (uniqueness check)
	idrefs map[string]bool // IDREF values to resolve (cross-ref check)
}

// docDTDs returns the DTDs to search in order, respecting standalone.
func docDTDs(doc *Document) []*DTD {
	var dtds []*DTD
	if doc.intSubset != nil {
		dtds = append(dtds, doc.intSubset)
	}
	if doc.standalone != StandaloneExplicitYes && doc.extSubset != nil {
		dtds = append(dtds, doc.extSubset)
	}
	return dtds
}

// lookupElementDecl searches both intSubset and extSubset for an element declaration.
func lookupElementDecl(doc *Document, name, prefix string) (*ElementDecl, *DTD) {
	for _, dtd := range docDTDs(doc) {
		if edecl, ok := dtd.LookupElement(name, prefix); ok {
			return edecl, dtd
		}
		if edecl, ok := dtd.LookupElement(name, ""); ok {
			return edecl, dtd
		}
	}
	return nil, nil
}

// validateDocument validates a parsed document against its DTD.
// This is the equivalent of libxml2's xmlValidateDocument.
func validateDocument(doc *Document) *ValidationError {
	vctx := &validCtx{
		ve:     &ValidationError{},
		ids:    make(map[string]bool),
		idrefs: make(map[string]bool),
	}

	if doc.intSubset == nil && doc.extSubset == nil {
		return nil
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
		if dtdName != "" && root.LocalName() != dtdName {
			vctx.ve.addf("root element name %q does not match DTD name %q", root.LocalName(), dtdName)
		}
	}

	// Walk the document tree and validate each element
	_ = Walk(doc, NodeWalkerFunc(func(n Node) error {
		if n.Type() != ElementNode {
			return nil
		}
		elem := n.(*Element)
		validateOneElement(doc, elem, vctx)
		return nil
	}))

	// Cross-reference check: every IDREF must match an existing ID
	validateDocumentFinal(vctx)

	if vctx.ve.hasErrors() {
		return vctx.ve
	}
	return nil
}

// validateOneElement checks a single element against DTD declarations.
func validateOneElement(doc *Document, elem *Element, vctx *validCtx) {
	name := elem.LocalName()

	edecl, dtd := lookupElementDecl(doc, name, elem.Prefix())
	if edecl == nil {
		// VC: Standalone Document Declaration — if standalone="yes" and the
		// element is declared in the external subset with element-only content,
		// whitespace-only text nodes are a validity error (the external subset
		// would have caused them to be treated as ignorable whitespace).
		if doc.standalone == StandaloneExplicitYes && doc.extSubset != nil {
			checkStandaloneWhitespace(doc.extSubset, elem, name, vctx)
		}
		vctx.ve.addf("element %s: no declaration found", name)
		return
	}

	// Validate attributes against their declarations
	validateElementAttributes(doc, elem, edecl, vctx)

	// Validate element content model
	validateElementContent(dtd, elem, edecl, vctx.ve)
}

// validateElementAttributes checks that:
// - All #REQUIRED attributes are present
// - #FIXED attributes have the correct value
// - No undeclared attributes (if element is fully declared)
// - ID values are unique across the document
// - IDREF/IDREFS values are recorded for cross-reference checking
func validateElementAttributes(doc *Document, elem *Element, edecl *ElementDecl, vctx *validCtx) {
	ename := elem.LocalName()
	attrs := elem.Attributes()

	// Build a set of attributes present on the element
	present := make(map[string]string, len(attrs))
	for _, a := range attrs {
		present[a.LocalName()] = a.Value()
	}

	// Check all declared attributes from both subsets, dedup by name
	seen := make(map[string]bool)
	for _, dtd := range docDTDs(doc) {
		for _, adecl := range dtd.AttributesForElement(ename) {
			aname := adecl.name
			if seen[aname] {
				continue
			}
			seen[aname] = true

			val, found := present[aname]

			switch adecl.def {
			case enum.AttrDefaultRequired:
				if !found {
					vctx.ve.addf("element %s: attribute %s is required", ename, aname)
				}
			case enum.AttrDefaultFixed:
				if found && val != adecl.defvalue {
					vctx.ve.addf("element %s: attribute %s has value %q but must be %q", ename, aname, val, adecl.defvalue)
				}
			}

			// Validate attribute value against its type (if present)
			if found {
				if err := validateAttributeValueInternal(doc, adecl.atype, val); err != nil {
					vctx.ve.addf("element %s: attribute %s: %s", ename, aname, err)
				}

				// Check enumeration value against declared tokens
				if adecl.atype == enum.AttrEnumeration && len(adecl.tree) > 0 {
					inEnum := false
					for _, token := range adecl.tree {
						if token == val {
							inEnum = true
							break
						}
					}
					if !inEnum {
						vctx.ve.addf("element %s: attribute %s value %q is not among the enumerated set", ename, aname, val)
					}
				}

				// Track ID uniqueness and collect IDREFs
				switch adecl.atype {
				case enum.AttrID:
					if vctx.ids[val] {
						vctx.ve.addf("element %s: duplicate ID %q", ename, val)
					} else {
						vctx.ids[val] = true
					}
				case enum.AttrIDRef:
					vctx.idrefs[val] = true
				case enum.AttrIDRefs:
					for _, ref := range strings.Fields(val) {
						vctx.idrefs[ref] = true
					}
				case enum.AttrEntity:
					ent, ok := doc.GetEntity(val)
					if !ok {
						vctx.ve.addf("element %s: attribute %s references undeclared entity %q", ename, aname, val)
					} else if ent.EntityType() != enum.ExternalGeneralUnparsedEntity {
						vctx.ve.addf("element %s: attribute %s references entity %q which is not unparsed", ename, aname, val)
					}
				case enum.AttrEntities:
					for _, entName := range strings.Fields(val) {
						ent, ok := doc.GetEntity(entName)
						if !ok {
							vctx.ve.addf("element %s: attribute %s references undeclared entity %q", ename, aname, entName)
						} else if ent.EntityType() != enum.ExternalGeneralUnparsedEntity {
							vctx.ve.addf("element %s: attribute %s references entity %q which is not unparsed", ename, aname, entName)
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
						vctx.ve.addf("element %s: attribute %s references undeclared notation %q", ename, aname, val)
					}
				}
			}
		}
	}
}

// validateDocumentFinal performs post-walk cross-reference checks.
// Every IDREF value must match an ID declared somewhere in the document.
func validateDocumentFinal(vctx *validCtx) {
	for ref := range vctx.idrefs {
		if !vctx.ids[ref] {
			vctx.ve.addf("IDREF %q references unknown ID", ref)
		}
	}
}

// checkStandaloneWhitespace checks whether an element declared in the
// external subset with element-only content contains whitespace-only text
// nodes. Per the VC: Standalone Document Declaration (XML §2.9), a
// standalone="yes" document must not contain such whitespace when the
// element declaration comes from the external subset.
func checkStandaloneWhitespace(extSubset *DTD, elem *Element, name string, vctx *validCtx) {
	extDecl, ok := extSubset.LookupElement(name, elem.Prefix())
	if !ok {
		extDecl, ok = extSubset.LookupElement(name, "")
	}
	if !ok || extDecl.decltype != enum.ElementElementType {
		return
	}
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Type() == TextNode && isBlankContent(child.Content()) {
			vctx.ve.addf("standalone: element %s declared in the external subset contains white spaces nodes", name)
			return
		}
	}
}

// validateElementContent validates that the element's children match the
// declared content model.
func validateElementContent(dtd *DTD, elem *Element, edecl *ElementDecl, ve *ValidationError) {
	ename := elem.LocalName()

	switch edecl.decltype {
	case enum.EmptyElementType:
		// EMPTY elements must have no children
		if elem.FirstChild() != nil {
			ve.addf("element %s: declared EMPTY but has content", ename)
		}
	case enum.AnyElementType:
		// ANY allows anything
	case enum.MixedElementType:
		// Mixed content: (#PCDATA | elem1 | elem2 | ...)*
		// All child elements must be in the declared list
		validateMixedContent(elem, edecl.content, ve)
	case enum.ElementElementType:
		// Element content: must match the content model exactly
		children := collectChildElements(elem)
		if !matchContentModel(edecl.content, children) {
			ve.addf("element %s: content does not match declared content model", ename)
		}
	}
}

// validateMixedContent validates children of a mixed-content element.
// In mixed content (#PCDATA | a | b)*, text nodes are always allowed,
// and element children must appear in the declared list.
func validateMixedContent(elem *Element, content *ElementContent, ve *ValidationError) {
	if content == nil {
		return
	}

	// Collect allowed element names from the content model
	allowed := collectMixedNames(content)

	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		switch child.Type() {
		case TextNode, CDATASectionNode, EntityRefNode, CommentNode, ProcessingInstructionNode:
			// Always allowed in mixed content
		case ElementNode:
			cname := child.(*Element).LocalName()
			if _, ok := allowed[cname]; !ok {
				ve.addf("element %s: child element %s not allowed in mixed content", elem.LocalName(), cname)
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
		names[content.name] = struct{}{}
		return
	}
	collectMixedNamesRecurse(content.c1, names)
	collectMixedNamesRecurse(content.c2, names)
}

// collectChildElements returns a slice of element names from the children
// of an element, ignoring text nodes, comments, PIs, etc.
func collectChildElements(elem *Element) []string {
	var children []string
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		switch child.Type() {
		case ElementNode:
			children = append(children, child.(*Element).LocalName())
		case TextNode:
			// In element-only content, whitespace text is allowed (ignorable
			// whitespace) but non-whitespace text is an error. We skip
			// whitespace-only text and treat non-whitespace as a mismatch
			// that will be caught by the content model check.
			if !isBlankContent(child.Content()) {
				// Use a sentinel to cause content model mismatch
				children = append(children, "#text")
			}
		}
	}
	return children
}

// isBlankContent returns true if the byte slice contains only whitespace.
func isBlankContent(b []byte) bool {
	for _, c := range b {
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			return false
		}
	}
	return true
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
	switch content.coccur {
	case ElementContentOnce:
		if pos < len(children) && children[pos] == content.name {
			return 1, true
		}
		return 0, false
	case ElementContentOpt:
		if pos < len(children) && children[pos] == content.name {
			return 1, true
		}
		return 0, true // optional: 0 matches is ok
	case ElementContentMult:
		// Zero or more
		count := 0
		for pos+count < len(children) && children[pos+count] == content.name {
			count++
		}
		return count, true
	case ElementContentPlus:
		// One or more
		if pos >= len(children) || children[pos] != content.name {
			return 0, false
		}
		count := 1
		for pos+count < len(children) && children[pos+count] == content.name {
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
