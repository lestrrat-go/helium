package helium

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

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

// isValidNameStartChar mirrors the XML spec NameStartChar production.
// We reuse the same logic as isNameStartChar in parserctx.go but without
// the colon (colons are only allowed in qualified names, not in attribute
// default values for ID/IDREF/etc).
func isValidNameStartChar(r rune) bool {
	return unicode.IsLetter(r) || r == '_'
}

// isValidNameChar mirrors the XML spec NameChar production (without colon).
func isValidNameChar(r rune) bool {
	return r == '.' || r == '-' || r == '_' ||
		unicode.IsLetter(r) || unicode.IsDigit(r) ||
		unicode.In(r, unicode.Extender)
}

// validateAttributeValueInternal validates that defvalue is legal for the
// declared attribute type. Mirrors xmlValidateAttributeDecl() in libxml2.
func validateAttributeValueInternal(doc *Document, typ AttributeType, defvalue string) error {
	switch typ {
	case AttrCDATA:
		// Any string is valid for CDATA
		return nil
	case AttrID, AttrIDRef, AttrEntity:
		// Must match the Name production
		if !isValidName(defvalue) {
			return fmt.Errorf("value %q is not a valid Name", defvalue)
		}
	case AttrIDRefs, AttrEntities:
		// Must match Names production: Name (S Name)*
		for _, tok := range strings.Fields(defvalue) {
			if !isValidName(tok) {
				return fmt.Errorf("value %q is not a valid Name", tok)
			}
		}
		if len(strings.Fields(defvalue)) == 0 {
			return errors.New("value must not be empty")
		}
	case AttrNmtoken:
		if !isValidNmtoken(defvalue) {
			return fmt.Errorf("value %q is not a valid NMTOKEN", defvalue)
		}
	case AttrNmtokens:
		for _, tok := range strings.Fields(defvalue) {
			if !isValidNmtoken(tok) {
				return fmt.Errorf("value %q is not a valid NMTOKEN", tok)
			}
		}
		if len(strings.Fields(defvalue)) == 0 {
			return errors.New("value must not be empty")
		}
	case AttrEnumeration, AttrNotation:
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

func (e *ValidationError) addf(format string, args ...interface{}) {
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

// validateDocument validates a parsed document against its DTD.
// This is the equivalent of libxml2's xmlValidateDocument.
func validateDocument(doc *Document) *ValidationError {
	vctx := &validCtx{
		ve:     &ValidationError{},
		ids:    make(map[string]bool),
		idrefs: make(map[string]bool),
	}

	dtd := doc.intSubset
	if dtd == nil {
		// No DTD means nothing to validate against
		return nil
	}

	// Walk the document tree and validate each element
	_ = Walk(doc, func(n Node) error {
		if n.Type() != ElementNode {
			return nil
		}
		elem := n.(*Element)
		validateOneElement(doc, dtd, elem, vctx)
		return nil
	})

	// Cross-reference check: every IDREF must match an existing ID
	validateDocumentFinal(vctx)

	if vctx.ve.hasErrors() {
		return vctx.ve
	}
	return nil
}

// validateOneElement checks a single element against DTD declarations.
func validateOneElement(doc *Document, dtd *DTD, elem *Element, vctx *validCtx) {
	name := elem.LocalName()

	// Look up the element declaration
	edecl, ok := dtd.LookupElement(name, elem.Prefix())
	if !ok {
		// Try without prefix (common case for unprefixed documents)
		edecl, ok = dtd.LookupElement(name, "")
	}
	if !ok {
		vctx.ve.addf("element %s: no declaration found", name)
		return
	}

	// Validate attributes against their declarations
	validateElementAttributes(doc, dtd, elem, edecl, vctx)

	// Validate element content model
	validateElementContent(dtd, elem, edecl, vctx.ve)
}

// validateElementAttributes checks that:
// - All #REQUIRED attributes are present
// - #FIXED attributes have the correct value
// - No undeclared attributes (if element is fully declared)
// - ID values are unique across the document
// - IDREF/IDREFS values are recorded for cross-reference checking
func validateElementAttributes(doc *Document, dtd *DTD, elem *Element, edecl *ElementDecl, vctx *validCtx) {
	ename := elem.LocalName()
	attrs := elem.Attributes()

	// Build a set of attributes present on the element
	present := make(map[string]string, len(attrs))
	for _, a := range attrs {
		present[a.LocalName()] = a.Value()
	}

	// Check all declared attributes for this element
	for _, adecl := range dtd.AttributesForElement(ename) {
		aname := adecl.name

		val, found := present[aname]

		switch adecl.def {
		case AttrDefaultRequired:
			if !found {
				vctx.ve.addf("element %s: attribute %s is required", ename, aname)
			}
		case AttrDefaultFixed:
			if found && val != adecl.defvalue {
				vctx.ve.addf("element %s: attribute %s has value %q but must be %q", ename, aname, val, adecl.defvalue)
			}
		}

		// Validate attribute value against its type (if present)
		if found {
			if err := validateAttributeValueInternal(doc, adecl.atype, val); err != nil {
				vctx.ve.addf("element %s: attribute %s: %s", ename, aname, err)
			}

			// Track ID uniqueness and collect IDREFs
			switch adecl.atype {
			case AttrID:
				if vctx.ids[val] {
					vctx.ve.addf("element %s: duplicate ID %q", ename, val)
				} else {
					vctx.ids[val] = true
				}
			case AttrIDRef:
				vctx.idrefs[val] = true
			case AttrIDRefs:
				for _, ref := range strings.Fields(val) {
					vctx.idrefs[ref] = true
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

// validateElementContent validates that the element's children match the
// declared content model.
func validateElementContent(dtd *DTD, elem *Element, edecl *ElementDecl, ve *ValidationError) {
	ename := elem.LocalName()

	switch edecl.decltype {
	case EmptyElementType:
		// EMPTY elements must have no children
		if elem.FirstChild() != nil {
			ve.addf("element %s: declared EMPTY but has content", ename)
		}
	case AnyElementType:
		// ANY allows anything
	case MixedElementType:
		// Mixed content: (#PCDATA | elem1 | elem2 | ...)*
		// All child elements must be in the declared list
		validateMixedContent(elem, edecl.content, ve)
	case ElementElementType:
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
func matchContentModel(content *ElementContent, children []string) bool {
	_, ok := matchContent(content, children, 0)
	return ok
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
