package xsd

import (
	"context"
	"fmt"
	"strings"

	helium "github.com/lestrrat-go/helium"
)

func validateDocument(ctx context.Context, doc *helium.Document, schema *Schema, cfg *validateConfig) (string, bool) {
	filename := cfg.filename
	var out strings.Builder
	valid := true

	// Initialize annotations map if requested.
	if cfg.annotations != nil && *cfg.annotations == nil {
		*cfg.annotations = make(TypeAnnotations)
	}

	root := findDocumentElement(doc)
	if root == nil {
		out.WriteString(filename + " fails to validate\n")
		return out.String(), false
	}

	// Walk the document tree for content model validation.
	_ = helium.Walk(doc, func(n helium.Node) error {
		if n.Type() != helium.ElementNode {
			return nil
		}
		elem := n.(*helium.Element)
		if err := validateElement(elem, schema, cfg, filename, &out); err != nil {
			valid = false
		}
		return nil
	})

	// Second walk: evaluate identity constraints (xs:key, xs:keyref, xs:unique).
	_ = helium.Walk(doc, func(n helium.Node) error {
		if n.Type() != helium.ElementNode {
			return nil
		}
		elem := n.(*helium.Element)
		edecl := lookupElemDecl(elem, schema)
		if edecl != nil && len(edecl.IDCs) > 0 {
			if err := validateIDConstraints(ctx, elem, edecl, schema, filename, &out); err != nil {
				valid = false
			}
		}
		return nil
	})

	if valid {
		out.WriteString(filename + " validates\n")
	} else {
		out.WriteString(filename + " fails to validate\n")
	}
	return out.String(), valid
}

func validateElement(elem *helium.Element, schema *Schema, cfg *validateConfig, filename string, out *strings.Builder) error {
	parent := elem.Parent()
	if parent == nil || parent.Type() == helium.DocumentNode {
		// Root element — must match a global element declaration.
		return validateRootElement(elem, schema, cfg, filename, out)
	}
	// Non-root elements are validated by their parent's content model.
	return nil
}

func validateRootElement(elem *helium.Element, schema *Schema, cfg *validateConfig, filename string, out *strings.Builder) error {
	local := elem.LocalName()
	ns := elem.URI()
	edecl, ok := schema.LookupElement(local, ns)
	if !ok {
		// Try with empty namespace.
		edecl, ok = schema.LookupElement(local, "")
	}
	if !ok {
		msg := "No matching global declaration available for the validation root."
		out.WriteString(validityError(filename, elem.Line(), local, msg))
		return fmt.Errorf("no matching global declaration")
	}

	if edecl.Type == nil {
		return nil
	}

	td, err := resolveXsiType(elem, edecl.Type, schema, filename, out)
	if err != nil {
		return err
	}
	// Check block flags against xsi:type derivation.
	if td != edecl.Type && edecl.Type != nil && isDerivationBlocked(td, edecl.Type, edecl.Block) {
		msg := "The xsi:type definition is blocked by the element declaration."
		out.WriteString(validityError(filename, elem.Line(), elemDisplayName(elem), msg))
		td = edecl.Type // fall back to declared type
	}
	if td != nil && td.Abstract {
		msg := "The type definition is abstract."
		out.WriteString(validityError(filename, elem.Line(), elemDisplayName(elem), msg))
		return fmt.Errorf("abstract type")
	}

	// Annotate root element with its type.
	annotateElement(cfg, elem, td)

	if hasXsiNil(elem) {
		return validateNilledElement(elem, edecl, td, schema, cfg, filename, out)
	}

	return validateElementContent(elem, edecl, td, schema, cfg, filename, out)
}

func validateElementContent(elem *helium.Element, edecl *ElementDecl, td *TypeDef, schema *Schema, cfg *validateConfig, filename string, out *strings.Builder) error {
	// Validate attributes and annotate them.
	if err := validateAttributes(elem, td, schema, cfg, filename, out); err != nil {
		return err
	}

	switch td.ContentType {
	case ContentTypeEmpty:
		return validateEmptyContent(elem, filename, out)
	case ContentTypeSimple:
		return validateSimpleContent(elem, edecl, td, filename, out)
	case ContentTypeElementOnly, ContentTypeMixed:
		if td.ContentModel == nil {
			// No content model means anything goes (for mixed) or empty (for element-only).
			if td.ContentType == ContentTypeElementOnly {
				return validateEmptyContent(elem, filename, out)
			}
			return nil
		}
		return validateContentModel(elem, td.ContentModel, schema, cfg, filename, out)
	}
	return nil
}

func validateSimpleContent(elem *helium.Element, edecl *ElementDecl, td *TypeDef, filename string, out *strings.Builder) error {
	// Simple content types must not have child elements.
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Type() == helium.ElementNode {
			out.WriteString(validityError(filename, elem.Line(), elem.LocalName(),
				"Element content is not allowed, because the content type is a simple type definition."))
			return fmt.Errorf("element content not allowed")
		}
	}

	value := elemTextContent(elem)
	isEmpty := value == ""

	// Effective value: substitute default/fixed for empty elements.
	effectiveValue := value
	if isEmpty && edecl != nil {
		if edecl.Fixed != nil {
			effectiveValue = *edecl.Fixed
		} else if edecl.Default != nil {
			effectiveValue = *edecl.Default
		}
	}

	// Fixed value mismatch check (only when element has actual content).
	if !isEmpty && edecl != nil && edecl.Fixed != nil {
		if strings.TrimSpace(value) != strings.TrimSpace(*edecl.Fixed) {
			msg := fmt.Sprintf("The element content '%s' does not match the fixed value constraint '%s'.", strings.TrimSpace(value), *edecl.Fixed)
			out.WriteString(validityError(filename, elem.Line(), elemDisplayName(elem), msg))
			return fmt.Errorf("fixed value constraint")
		}
	}

	// Validate the text value against the type.
	if td != nil && (td.Facets != nil || resolveVariety(td) == TypeVarietyList || resolveVariety(td) == TypeVarietyUnion || builtinBaseLocal(td) != "" && builtinBaseLocal(td) != "string" && builtinBaseLocal(td) != "anySimpleType") {
		return validateValue(effectiveValue, td, elemDisplayName(elem), filename, elem.Line(), out)
	}

	return nil
}

func validateEmptyContent(elem *helium.Element, filename string, out *strings.Builder) error {
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		switch child.Type() {
		case helium.ElementNode:
			ce := child.(*helium.Element)
			out.WriteString(validityError(filename, ce.Line(), ce.LocalName(), "This element is not expected."))
			return fmt.Errorf("not expected")
		case helium.TextNode:
			if !isBlank(child.Content()) {
				out.WriteString(validityError(filename, elem.Line(), elem.LocalName(), "Character content is not allowed, because the type definition is simple."))
				return fmt.Errorf("not expected")
			}
		}
	}
	return nil
}

func validateContentModel(elem *helium.Element, mg *ModelGroup, schema *Schema, cfg *validateConfig, filename string, out *strings.Builder) error {
	children := collectChildElements(elem)
	return validateContentModelTop(elem, mg, children, schema, cfg, filename, out)
}

type childElem struct {
	elem        *helium.Element
	name        string // local name (for matching)
	ns          string // namespace URI (for matching)
	displayName string // namespace-qualified name (for error messages)
}

func collectChildElements(elem *helium.Element) []childElem {
	var children []childElem
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Type() == helium.ElementNode {
			ce := child.(*helium.Element)
			children = append(children, childElem{elem: ce, name: ce.LocalName(), ns: ce.URI(), displayName: elemDisplayName(ce)})
		}
	}
	return children
}

func isSpecialAttr(a *helium.Attribute) bool {
	p := a.Prefix()
	if p == "xmlns" || (p == "" && a.LocalName() == "xmlns") {
		return true
	}
	uri := a.URI()
	if uri == "http://www.w3.org/2001/XMLSchema-instance" {
		return true
	}
	if uri == "http://www.w3.org/XML/1998/namespace" {
		return true
	}
	return false
}

func elemDisplayName(elem *helium.Element) string {
	if elem.URI() != "" {
		return "{" + elem.URI() + "}" + elem.LocalName()
	}
	return elem.LocalName()
}

func attrDisplayName(a *helium.Attribute) string {
	uri := a.URI()
	if uri != "" {
		return "{" + uri + "}" + a.LocalName()
	}
	return a.LocalName()
}

func validateAttributes(elem *helium.Element, td *TypeDef, schema *Schema, cfg *validateConfig, filename string, out *strings.Builder) error {
	var hasErr bool

	if len(td.Attributes) == 0 && td.AnyAttribute == nil {
		// No attribute declarations — check that instance has no attributes
		// (except xsi: namespace attributes and xmlns which are always allowed).
		for _, a := range elem.Attributes() {
			if isSpecialAttr(a) {
				continue
			}
			ad := attrDisplayName(a)
			msg := fmt.Sprintf("The attribute '%s' is not allowed.", ad)
			out.WriteString(validityErrorAttr(filename, elem.Line(), elemDisplayName(elem), ad, msg))
			hasErr = true
		}
		if hasErr {
			return fmt.Errorf("attribute not allowed")
		}
		return nil
	}

	// Build set of allowed attributes.
	allowed := make(map[QName]*AttrUse, len(td.Attributes))
	for _, au := range td.Attributes {
		allowed[au.Name] = au
	}

	// Check for unknown attributes and fixed value constraints.
	for _, a := range elem.Attributes() {
		if isSpecialAttr(a) {
			continue
		}
		aqn := QName{Local: a.LocalName(), NS: a.URI()}
		if au, ok := allowed[aqn]; ok {
			if au.Fixed != nil && a.Value() != *au.Fixed {
				ad := attrDisplayName(a)
				msg := fmt.Sprintf("The value '%s' does not match the fixed value constraint '%s'.", a.Value(), *au.Fixed)
				out.WriteString(validityErrorAttr(filename, elem.Line(), elemDisplayName(elem), ad, msg))
				hasErr = true
			}
			// Annotate the attribute with its declared type.
			annotateAttrUse(cfg, a, au, schema)
			continue
		}
		// Not in explicit declarations — check anyAttribute wildcard.
		if td.AnyAttribute != nil && wildcardMatchesAttr(td.AnyAttribute, a.URI()) {
			if err := validateWildcardAttr(a, elem, td.AnyAttribute, schema, filename, out); err != nil {
				hasErr = true
			}
			continue
		}
		ad := attrDisplayName(a)
		msg := fmt.Sprintf("The attribute '%s' is not allowed.", ad)
		out.WriteString(validityErrorAttr(filename, elem.Line(), elemDisplayName(elem), ad, msg))
		hasErr = true
	}

	// Check for required attributes.
	for _, au := range td.Attributes {
		if !au.Required {
			continue
		}
		found := false
		for _, a := range elem.Attributes() {
			aqn := QName{Local: a.LocalName(), NS: a.URI()}
			if aqn == au.Name {
				found = true
				break
			}
		}
		if !found {
			msg := fmt.Sprintf("The attribute '%s' is required but missing.", au.Name.Local)
			out.WriteString(validityError(filename, elem.Line(), elemDisplayName(elem), msg))
			hasErr = true
		}
	}

	// Insert default/fixed attribute values for absent optional attributes.
	for _, au := range td.Attributes {
		if au.Required {
			continue
		}
		defVal := ""
		if au.Default != nil {
			defVal = *au.Default
		} else if au.Fixed != nil {
			defVal = *au.Fixed
		} else {
			continue
		}
		// Check if the attribute is already present.
		found := false
		for _, a := range elem.Attributes() {
			aqn := QName{Local: a.LocalName(), NS: a.URI()}
			if aqn == au.Name {
				found = true
				break
			}
		}
		if !found {
			// Insert the default/fixed value as an attribute on the element.
			_ = elem.SetAttribute(au.Name.Local, defVal)
			// Annotate the newly inserted attribute.
			for _, a := range elem.Attributes() {
				if a.LocalName() == au.Name.Local && a.URI() == au.Name.NS {
					annotateAttrUse(cfg, a, au, schema)
					break
				}
			}
		}
	}

	if hasErr {
		return fmt.Errorf("attribute validation failed")
	}
	return nil
}

// validateWildcardAttr validates an attribute matched by a wildcard according
// to its processContents setting (strict, lax, or skip).
func validateWildcardAttr(a *helium.Attribute, elem *helium.Element, wc *Wildcard, schema *Schema, filename string, out *strings.Builder) error {
	if wc.ProcessContents == ProcessSkip {
		return nil
	}

	// Look up global attribute declaration.
	aqn := QName{Local: a.LocalName(), NS: a.URI()}
	globalAttr, found := schema.globalAttrs[aqn]

	if !found {
		if wc.ProcessContents == ProcessStrict {
			ad := attrDisplayName(a)
			msg := "No matching global attribute declaration available, but demanded by the strict wildcard."
			out.WriteString(validityErrorAttr(filename, elem.Line(), elemDisplayName(elem), ad, msg))
			return fmt.Errorf("strict wildcard: no global attr")
		}
		// Lax: no global declaration found — skip validation.
		return nil
	}

	// Global attribute found — validate value against its type if known.
	if globalAttr.TypeName.Local != "" {
		attrTD, ok := schema.types[globalAttr.TypeName]
		if ok {
			value := a.Value()
			trimmed := strings.TrimSpace(value)
			builtinLocal := builtinBaseLocal(attrTD)
			if err := validateBuiltinValue(trimmed, builtinLocal); err != nil {
				ad := attrDisplayName(a)
				typeName := typeDisplayName(attrTD)
				msg := fmt.Sprintf("'%s' is not a valid value of the atomic type '%s'.", trimmed, typeName)
				out.WriteString(validityErrorAttr(filename, elem.Line(), elemDisplayName(elem), ad, msg))
				return err
			}
		}
	}

	return nil
}

// wildcardMatchesAttr checks if an attribute namespace matches an anyAttribute wildcard.
func wildcardMatchesAttr(wc *Wildcard, attrNS string) bool {
	return wildcardMatches(wc, attrNS)
}

// elemTextContent returns the concatenated text content of an element,
// including both text nodes and CDATA sections.
// lookupElemDecl finds the global element declaration for an instance element.
func lookupElemDecl(elem *helium.Element, schema *Schema) *ElementDecl {
	edecl, ok := schema.LookupElement(elem.LocalName(), elem.URI())
	if ok {
		return edecl
	}
	edecl, ok = schema.LookupElement(elem.LocalName(), "")
	if ok {
		return edecl
	}
	return nil
}

func elemTextContent(elem *helium.Element) string {
	var buf []byte
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		switch child.Type() {
		case helium.TextNode, helium.CDATASectionNode:
			buf = append(buf, child.Content()...)
		}
	}
	return string(buf)
}

func isBlank(b []byte) bool {
	for _, c := range b {
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			return false
		}
	}
	return true
}

const xsiNS = "http://www.w3.org/2001/XMLSchema-instance"

// hasXsiNil returns true if the element has xsi:nil="true".
func hasXsiNil(elem *helium.Element) bool {
	for _, a := range elem.Attributes() {
		if a.URI() == xsiNS && a.LocalName() == "nil" {
			return a.Value() == "true" || a.Value() == "1"
		}
	}
	return false
}

// validateNilledElement handles an element with xsi:nil="true".
// If the declaration is nillable, validates that the element has no character
// or element content (attributes are still checked).  If not nillable,
// reports a validity error.
func validateNilledElement(elem *helium.Element, edecl *ElementDecl, td *TypeDef, schema *Schema, cfg *validateConfig, filename string, out *strings.Builder) error {
	dn := elemDisplayName(elem)

	if !edecl.Nillable {
		out.WriteString(validityError(filename, elem.Line(), dn,
			"Element is not nillable."))
		return fmt.Errorf("element not nillable")
	}

	// Validate attributes even for nilled elements.
	if td != nil {
		if err := validateAttributes(elem, td, schema, cfg, filename, out); err != nil {
			return err
		}
	}

	// xsi:nil="true" — the element must have no character or element children.
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		switch child.Type() {
		case helium.ElementNode:
			ce := child.(*helium.Element)
			out.WriteString(validityError(filename, ce.Line(), elemDisplayName(ce),
				"This element is not expected, because the element '"+dn+"' is nilled."))
			return fmt.Errorf("content in nilled element")
		case helium.TextNode, helium.CDATASectionNode:
			if !isBlank(child.Content()) {
				out.WriteString(validityError(filename, elem.Line(), dn,
					"Character content is not allowed, because the element is nilled."))
				return fmt.Errorf("content in nilled element")
			}
		}
	}

	return nil
}

// isDerivedFrom returns true if derived is the same type as base, or if any
// ancestor in derived's BaseType chain is base. Also returns true if base is
// xs:anyType (the ur-type from which everything derives).
func isDerivedFrom(derived, base *TypeDef) bool {
	if derived == base {
		return true
	}
	if base.Name.Local == "anyType" && base.Name.NS == xsdNS {
		return true
	}
	for cur := derived.BaseType; cur != nil; cur = cur.BaseType {
		if cur == base {
			return true
		}
	}
	return false
}

// resolveXsiType checks if the element has an xsi:type attribute and, if so,
// resolves it to a type definition in the schema. Returns the resolved type
// or the original declaredType if no xsi:type is present. Returns an error
// if the xsi:type value doesn't resolve or is not derived from the declared type.
func resolveXsiType(elem *helium.Element, declaredType *TypeDef, schema *Schema, filename string, out *strings.Builder) (*TypeDef, error) {
	var xsiTypeVal string
	for _, a := range elem.Attributes() {
		if a.URI() == xsiNS && a.LocalName() == "type" {
			xsiTypeVal = a.Value()
			break
		}
	}
	if xsiTypeVal == "" {
		return declaredType, nil
	}

	// Parse QName value: may be "prefix:local" or just "local".
	local := xsiTypeVal
	var ns string
	if idx := strings.IndexByte(xsiTypeVal, ':'); idx >= 0 {
		prefix := xsiTypeVal[:idx]
		local = xsiTypeVal[idx+1:]
		ns = lookupNS(elem, prefix)
	} else {
		// No prefix — use the default namespace (empty prefix) or schema target namespace.
		ns = lookupNS(elem, "")
	}

	td, ok := schema.LookupType(local, ns)
	if !ok {
		// Try with schema's target namespace.
		td, ok = schema.LookupType(local, schema.TargetNamespace())
	}
	if !ok {
		msg := fmt.Sprintf("The value '%s' of the xsi:type attribute does not resolve to a type definition.", xsiTypeVal)
		out.WriteString(validityError(filename, elem.Line(), elemDisplayName(elem), msg))
		return nil, fmt.Errorf("xsi:type not found")
	}

	// Check derivation: xsi:type must be the same as or derived from the declared type.
	if declaredType != nil && !isDerivedFrom(td, declaredType) {
		msg := fmt.Sprintf("The type definition '%s' is not validly derived from the type definition '%s'.",
			typeDisplayName(td), typeDisplayName(declaredType))
		out.WriteString(validityError(filename, elem.Line(), elemDisplayName(elem), msg))
		return nil, fmt.Errorf("xsi:type not derived")
	}

	return td, nil
}

// xsdTypeName converts a TypeDef to a type name string suitable for annotations.
func xsdTypeName(td *TypeDef) string {
	if td == nil {
		return "xs:untyped"
	}
	if td.Name.NS == xsdNS {
		return "xs:" + td.Name.Local
	}
	if td.Name.NS != "" {
		return "Q{" + td.Name.NS + "}" + td.Name.Local
	}
	if td.Name.Local != "" {
		return td.Name.Local
	}
	return "xs:untyped"
}

// annotateElement records a type annotation for an element node.
func annotateElement(cfg *validateConfig, elem *helium.Element, td *TypeDef) {
	if cfg == nil || cfg.annotations == nil {
		return
	}
	(*cfg.annotations)[elem] = xsdTypeName(td)
}

// annotateAttrUse records a type annotation for an attribute node based on its AttrUse declaration.
func annotateAttrUse(cfg *validateConfig, a *helium.Attribute, au *AttrUse, schema *Schema) {
	if cfg == nil || cfg.annotations == nil {
		return
	}
	if au.TypeName.Local == "" {
		return
	}
	td, ok := schema.types[au.TypeName]
	if !ok {
		return
	}
	(*cfg.annotations)[a] = xsdTypeName(td)
}
