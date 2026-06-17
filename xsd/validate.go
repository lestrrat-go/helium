package xsd

import (
	"context"
	"fmt"
	"strings"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
)

type validationContext struct {
	schema        *Schema
	cfg           *validateConfig
	filename      string
	errorHandler  helium.ErrorHandler
	suppressDepth int
}

func newValidationContext(schema *Schema, cfg *validateConfig, filename string, handler helium.ErrorHandler) *validationContext {
	return &validationContext{
		schema:       schema,
		cfg:          cfg,
		filename:     filename,
		errorHandler: handler,
	}
}

// validationErrors is a synchronous ErrorHandler that accumulates error
// strings in order. Used internally by ValidateElement and tests.
type validationErrors struct {
	errors []string
}

func (ve *validationErrors) Handle(_ context.Context, err error) {
	ve.errors = append(ve.errors, err.Error())
}

// reportValidityError formats a validation error and sends it to the ErrorHandler.
func (vc *validationContext) reportValidityError(ctx context.Context, file string, line int, elemName, msg string) {
	if vc.suppressDepth > 0 {
		return
	}
	ve := &ValidationError{
		Filename: file,
		Line:     line,
		Element:  elemName,
		Message:  msg,
	}
	vc.errorHandler.Handle(ctx, newLeveledValidationError(ve, helium.ErrorLevelError))
}

// reportValidityErrorAttr formats an attribute validation error and sends it to the ErrorHandler.
func (vc *validationContext) reportValidityErrorAttr(ctx context.Context, file string, line int, elemName, attrName, msg string) {
	if vc.suppressDepth > 0 {
		return
	}
	ve := &ValidationError{
		Filename:      file,
		Line:          line,
		Element:       elemName,
		AttributeName: attrName,
		Message:       msg,
	}
	vc.errorHandler.Handle(ctx, newLeveledValidationError(ve, helium.ErrorLevelError))
}

// Validate validates a lexical value against this simple type definition.
// nsMap provides prefix-to-URI mappings for QName/NOTATION resolution and may be nil.
func (td *TypeDef) Validate(ctx context.Context, value string, nsMap map[string]string) error {
	if td == nil {
		return fmt.Errorf("nil type definition")
	}
	if td.ContentType != ContentTypeSimple {
		return fmt.Errorf("type %q is not a simple type", typeQualifiedName(td))
	}
	vc := &validationContext{
		errorHandler: helium.NilErrorHandler{},
	}
	return validateValue(ctx, value, nsMap, td, "", "", 0, vc)
}

// ValidateElement validates an element's content against this type definition.
// This is used by XSLT xsl:type validation where the element is constructed
// in the result tree and must conform to the given type.
func (td *TypeDef) ValidateElement(ctx context.Context, elem *helium.Element, schema *Schema) error {
	if td == nil {
		return fmt.Errorf("nil type definition")
	}
	collector := &validationErrors{}
	vc := newValidationContext(schema, &validateConfig{}, "", collector)
	err := vc.validateElementContent(ctx, elem, nil, td)
	if err == nil {
		return nil
	}
	if len(collector.errors) > 0 {
		var b strings.Builder
		for _, e := range collector.errors {
			b.WriteString(e)
		}
		return fmt.Errorf("%s", strings.TrimSpace(b.String()))
	}
	return err
}

func validateDocument(ctx context.Context, doc *helium.Document, schema *Schema, cfg *validateConfig, handler helium.ErrorHandler) bool {
	filename := cfg.label
	if filename == "" {
		filename = doc.URL()
	}
	if filename == "" {
		filename = "(string)"
	}
	valid := true
	vc := newValidationContext(schema, cfg, filename, handler)

	// Initialize annotations map if requested.
	if cfg.annotations != nil && *cfg.annotations == nil {
		*cfg.annotations = make(TypeAnnotations)
	}
	// Initialize nilled elements map if requested.
	if cfg.nilledElements != nil && *cfg.nilledElements == nil {
		*cfg.nilledElements = make(NilledElements)
	}

	root := findDocumentElement(doc)
	if root == nil {
		return false
	}

	// Walk the document tree for content model validation.
	_ = helium.Walk(doc, helium.NodeWalkerFunc(func(n helium.Node) error {
		if n.Type() != helium.ElementNode {
			return nil
		}
		elem, ok := helium.AsNode[*helium.Element](n)
		if !ok {
			return nil
		}
		if err := vc.validateElement(ctx, elem); err != nil {
			valid = false
		}
		return nil
	}))

	// Second walk: evaluate identity constraints (xs:key, xs:keyref, xs:unique).
	_ = helium.Walk(doc, helium.NodeWalkerFunc(func(n helium.Node) error {
		if n.Type() != helium.ElementNode {
			return nil
		}
		elem, ok := helium.AsNode[*helium.Element](n)
		if !ok {
			return nil
		}
		edecl := lookupElemDecl(elem, vc.schema)
		if edecl != nil && len(edecl.IDCs) > 0 {
			if err := vc.validateIDConstraints(ctx, elem, edecl); err != nil {
				valid = false
			}
		}
		return nil
	}))

	return valid
}

func (vc *validationContext) validateElement(ctx context.Context, elem *helium.Element) error {
	parent := elem.Parent()
	if parent == nil || parent.Type() == helium.DocumentNode {
		// Root element — must match a global element declaration.
		return vc.validateRootElement(ctx, elem)
	}
	// Non-root elements are validated by their parent's content model.
	return nil
}

func (vc *validationContext) validateRootElement(ctx context.Context, elem *helium.Element) error {
	local := elem.LocalName()
	ns := elem.URI()
	// Match on the element's full expanded name. An element with a non-empty
	// namespace must NOT fall back to an unqualified declaration that merely
	// shares the local name: cvc-elt requires the instance and declaration
	// expanded names to be identical (libxml2 rejects {urn:wrong}foo against a
	// no-namespace schema declaring {}foo).
	edecl, ok := vc.schema.LookupElement(local, ns)
	if !ok {
		msg := "No matching global declaration available for the validation root."
		vc.reportValidityError(ctx, vc.filename, elem.Line(), local, msg)
		return fmt.Errorf("no matching global declaration")
	}

	// Keep edecl as the ACTUAL root declaration so its own Nillable flag is
	// honored by the nilled-element check. For a no-type substitution-group
	// member, the effective TYPE is inherited from the head (effectiveDeclType
	// walks the substitutionGroup chain), but the declaration — and thus the
	// nillable flag — stays the member's. This mirrors the particle paths.
	declType := effectiveDeclType(edecl, vc.schema)
	if declType == nil {
		return nil
	}

	td, err := vc.resolveXsiType(ctx, elem, declType)
	if err != nil {
		return err
	}
	// Check block flags against xsi:type derivation.
	if td != declType && isDerivationBlocked(td, declType, edecl.Block) {
		msg := "The xsi:type definition is blocked by the element declaration."
		vc.reportValidityError(ctx, vc.filename, elem.Line(), elemDisplayName(elem), msg)
		td = declType // fall back to declared type
	}
	if td != nil && td.Abstract {
		msg := "The type definition is abstract."
		vc.reportValidityError(ctx, vc.filename, elem.Line(), elemDisplayName(elem), msg)
		return fmt.Errorf("abstract type")
	}

	// Annotate root element with its type.
	vc.annotateElement(ctx, elem, td)

	nilled, err := vc.checkXsiNil(ctx, elem)
	if err != nil {
		return err
	}
	if nilled {
		return vc.validateNilledElement(ctx, elem, edecl, td)
	}

	return vc.validateElementContent(ctx, elem, edecl, td)
}

func (vc *validationContext) validateElementContent(ctx context.Context, elem *helium.Element, edecl *ElementDecl, td *TypeDef) error {
	// Validate attributes and annotate them.
	if err := vc.validateAttributes(ctx, elem, td); err != nil {
		return err
	}

	switch td.ContentType {
	case ContentTypeEmpty:
		return vc.validateEmptyContent(ctx, elem)
	case ContentTypeSimple:
		return vc.validateSimpleContent(ctx, elem, edecl, td)
	case ContentTypeElementOnly, ContentTypeMixed:
		// For element-only content, non-whitespace text children are not allowed.
		if td.ContentType == ContentTypeElementOnly {
			for child := range helium.Children(elem) {
				if child.Type() == helium.TextNode || child.Type() == helium.CDATASectionNode {
					if strings.TrimSpace(string(child.Content())) != "" {
						msg := "Character content other than whitespace is not allowed because the content type is 'element-only'."
						vc.reportValidityError(ctx, vc.filename, elem.Line(), elemDisplayName(elem), msg)
						return fmt.Errorf("text content in element-only type")
					}
				}
			}
		}
		if td.ContentModel == nil {
			// No content model means anything goes (for mixed) or empty (for element-only).
			if td.ContentType == ContentTypeElementOnly {
				return vc.validateEmptyContent(ctx, elem)
			}
			return nil
		}
		return vc.validateContentModel(ctx, elem, td.ContentModel)
	}
	return nil
}

func (vc *validationContext) validateSimpleContent(ctx context.Context, elem *helium.Element, edecl *ElementDecl, td *TypeDef) error {
	// Simple content types must not have child elements.
	for child := range helium.Children(elem) {
		if child.Type() == helium.ElementNode {
			vc.reportValidityError(ctx, vc.filename, elem.Line(), elem.LocalName(),
				"Element content is not allowed, because the content type is a simple type definition.")
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
			vc.reportValidityError(ctx, vc.filename, elem.Line(), elemDisplayName(elem), msg)
			return fmt.Errorf("fixed value constraint")
		}
	}

	// Validate the text value against the type.
	if td != nil && (td.Facets != nil || resolveVariety(td) == TypeVarietyList || resolveVariety(td) == TypeVarietyUnion || builtinBaseLocal(td) != "" && builtinBaseLocal(td) != "string" && builtinBaseLocal(td) != "anySimpleType") {
		return validateValue(ctx, effectiveValue, collectNSContext(elem), td, elemDisplayName(elem), vc.filename, elem.Line(), vc)
	}

	return nil
}

func (vc *validationContext) validateEmptyContent(ctx context.Context, elem *helium.Element) error {
	for child := range helium.Children(elem) {
		switch child.Type() {
		case helium.ElementNode:
			ce, ok := helium.AsNode[*helium.Element](child)
			if !ok {
				continue
			}
			vc.reportValidityError(ctx, vc.filename, ce.Line(), ce.LocalName(), "This element is not expected.")
			return fmt.Errorf("not expected")
		case helium.TextNode, helium.CDATASectionNode:
			if !isBlank(child.Content()) {
				vc.reportValidityError(ctx, vc.filename, elem.Line(), elem.LocalName(), "Character content is not allowed, because the type definition is simple.")
				return fmt.Errorf("not expected")
			}
		}
	}
	return nil
}

func (vc *validationContext) validateContentModel(ctx context.Context, elem *helium.Element, mg *ModelGroup) error {
	children := collectChildElements(elem)
	return vc.validateContentModelTop(ctx, elem, mg, children)
}

type childElem struct {
	elem        *helium.Element
	name        string // local name (for matching)
	ns          string // namespace URI (for matching)
	displayName string // namespace-qualified name (for error messages)
}

func collectChildElements(elem *helium.Element) []childElem {
	var children []childElem
	for child := range helium.Children(elem) {
		if child.Type() == helium.ElementNode {
			ce, ok := helium.AsNode[*helium.Element](child)
			if !ok {
				continue
			}
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
	if uri == lexicon.NamespaceXSI {
		return true
	}
	if uri == lexicon.NamespaceXML {
		return true
	}
	return false
}

func elemDisplayName(elem *helium.Element) string {
	if elem.URI() != "" {
		return helium.ClarkName(elem.URI(), elem.LocalName())
	}
	return elem.LocalName()
}

func attrDisplayName(a *helium.Attribute) string {
	uri := a.URI()
	if uri != "" {
		return helium.ClarkName(uri, a.LocalName())
	}
	return a.LocalName()
}

func (vc *validationContext) validateAttributes(ctx context.Context, elem *helium.Element, td *TypeDef) error {
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
			vc.reportValidityErrorAttr(ctx, vc.filename, elem.Line(), elemDisplayName(elem), ad, msg)
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

	// Build set of present instance attributes (excluding special attrs)
	// for O(1) lookups in the required-check and default-insertion loops.
	present := make(map[QName]struct{}, len(elem.Attributes()))

	// Check for unknown attributes and fixed value constraints.
	for _, a := range elem.Attributes() {
		if isSpecialAttr(a) {
			continue
		}
		aqn := QName{Local: a.LocalName(), NS: a.URI()}
		present[aqn] = struct{}{}
		if au, ok := allowed[aqn]; ok {
			if au.Fixed != nil && a.Value() != *au.Fixed {
				ad := attrDisplayName(a)
				msg := fmt.Sprintf("The value '%s' does not match the fixed value constraint '%s'.", a.Value(), *au.Fixed)
				vc.reportValidityErrorAttr(ctx, vc.filename, elem.Line(), elemDisplayName(elem), ad, msg)
				hasErr = true
			}
			// Validate the attribute value against its declared type
			// (inline anonymous simpleType takes precedence over a named type).
			attrTD, tdOK := vc.attrUseType(au)
			if tdOK && attrTD.ContentType == ContentTypeSimple {
				if err := attrTD.Validate(ctx, a.Value(), collectNSContext(elem)); err != nil {
					ad := attrDisplayName(a)
					msg := fmt.Sprintf("The value '%s' is not valid for the type of attribute '%s'.", a.Value(), ad)
					vc.reportValidityErrorAttr(ctx, vc.filename, elem.Line(), elemDisplayName(elem), ad, msg)
					hasErr = true
				}
			}
			// Annotate the attribute with its declared type.
			vc.annotateAttrUse(ctx, a, au)
			continue
		}
		// Not in explicit declarations — check anyAttribute wildcard.
		if td.AnyAttribute != nil && wildcardMatchesAttr(td.AnyAttribute, a.URI()) {
			if err := vc.validateWildcardAttr(ctx, a, elem, td.AnyAttribute); err != nil {
				hasErr = true
			}
			continue
		}
		ad := attrDisplayName(a)
		msg := fmt.Sprintf("The attribute '%s' is not allowed.", ad)
		vc.reportValidityErrorAttr(ctx, vc.filename, elem.Line(), elemDisplayName(elem), ad, msg)
		hasErr = true
	}

	// Check for required attributes.
	for _, au := range td.Attributes {
		if !au.Required {
			continue
		}
		if _, ok := present[au.Name]; !ok {
			msg := fmt.Sprintf("The attribute '%s' is required but missing.", au.Name.Local)
			vc.reportValidityError(ctx, vc.filename, elem.Line(), elemDisplayName(elem), msg)
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
		if _, ok := present[au.Name]; ok {
			continue
		}
		// Insert the default/fixed value as an attribute on the element.
		_, _ = elem.SetAttribute(au.Name.Local, defVal)
		// Annotate the newly inserted attribute.
		for _, a := range elem.Attributes() {
			if a.LocalName() == au.Name.Local && a.URI() == au.Name.NS {
				vc.annotateAttrUse(ctx, a, au)
				break
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
func (vc *validationContext) validateWildcardAttr(ctx context.Context, a *helium.Attribute, elem *helium.Element, wc *Wildcard) error {
	if wc.ProcessContents == ProcessSkip {
		return nil
	}

	// Look up global attribute declaration.
	aqn := QName{Local: a.LocalName(), NS: a.URI()}
	globalAttr, found := vc.schema.globalAttrs[aqn]

	if !found {
		if wc.ProcessContents == ProcessStrict {
			ad := attrDisplayName(a)
			msg := "No matching global attribute declaration available, but demanded by the strict wildcard."
			vc.reportValidityErrorAttr(ctx, vc.filename, elem.Line(), elemDisplayName(elem), ad, msg)
			return fmt.Errorf("strict wildcard: no global attr")
		}
		// Lax: no global declaration found — skip validation.
		return nil
	}

	// Global attribute found — validate value against its effective type if
	// known (an inline anonymous simpleType takes precedence over a named type).
	// TypeDef.Validate handles facets, lists, and unions, not just the builtin
	// base lexical space.
	attrTD, ok := vc.attrUseType(globalAttr)
	if ok && attrTD.ContentType == ContentTypeSimple {
		value := a.Value()
		if err := attrTD.Validate(ctx, value, collectNSContext(elem)); err != nil {
			ad := attrDisplayName(a)
			typeName := typeDisplayName(attrTD)
			msg := fmt.Sprintf("'%s' is not a valid value of the atomic type '%s'.", strings.TrimSpace(value), typeName)
			vc.reportValidityErrorAttr(ctx, vc.filename, elem.Line(), elemDisplayName(elem), ad, msg)
			return err
		}
	}

	return nil
}

// wildcardMatchesAttr checks if an attribute namespace matches an anyAttribute wildcard.
func wildcardMatchesAttr(wc *Wildcard, attrNS string) bool {
	return wildcardMatches(wc, attrNS)
}

// lookupElemDecl finds the global element declaration for an instance element.
// Matching is on the element's full expanded name: a namespaced element does
// not fall back to an unqualified declaration sharing the local name.
func lookupElemDecl(elem *helium.Element, schema *Schema) *ElementDecl {
	edecl, ok := schema.LookupElement(elem.LocalName(), elem.URI())
	if ok {
		return edecl
	}
	return nil
}

// elemTextContent returns the concatenated text content of an element,
// including both text nodes and CDATA sections.
func elemTextContent(elem *helium.Element) string {
	var buf []byte
	for child := range helium.Children(elem) {
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

// checkXsiNil parses the element's xsi:nil attribute as an xs:boolean (after
// whitespace collapse). It returns whether the element is nilled ("true"/"1").
// "false"/"0" and an absent attribute mean not-nilled. Any other lexical form
// is an invalid xs:boolean value: a validity error is reported and a non-nil
// error is returned so the element is not silently validated as ordinary
// content.
func (vc *validationContext) checkXsiNil(ctx context.Context, elem *helium.Element) (bool, error) {
	for _, a := range elem.Attributes() {
		if a.URI() != lexicon.NamespaceXSI || a.LocalName() != attrNil {
			continue
		}
		v := normalizeWhiteSpace(a.Value(), "collapse")
		switch v {
		case "true", "1":
			return true, nil
		case "false", "0":
			return false, nil
		}
		msg := fmt.Sprintf("'%s' is not a valid value of the atomic type 'xs:boolean'.", v)
		vc.reportValidityErrorAttr(ctx, vc.filename, elem.Line(), elemDisplayName(elem), attrDisplayName(a), msg)
		return false, fmt.Errorf("invalid xsi:nil value %q", a.Value())
	}
	return false, nil
}

// validateNilledElement handles an element with xsi:nil="true".
// If the declaration is nillable, validates that the element has no character
// or element content (attributes are still checked).  If not nillable,
// reports a validity error.
func (vc *validationContext) validateNilledElement(ctx context.Context, elem *helium.Element, edecl *ElementDecl, td *TypeDef) error {
	dn := elemDisplayName(elem)

	if !edecl.Nillable {
		vc.reportValidityError(ctx, vc.filename, elem.Line(), dn,
			"Element is not nillable.")
		return fmt.Errorf("element not nillable")
	}

	// Record the element as nilled for PSVI consumers (e.g. fn:nilled()).
	if vc.cfg != nil && vc.cfg.nilledElements != nil {
		(*vc.cfg.nilledElements)[elem] = struct{}{}
	}

	// Validate attributes even for nilled elements.
	if td != nil {
		if err := vc.validateAttributes(ctx, elem, td); err != nil {
			return err
		}
	}

	// xsi:nil="true" — the element must have no character or element children.
	for child := range helium.Children(elem) {
		switch child.Type() {
		case helium.ElementNode:
			ce, ok := helium.AsNode[*helium.Element](child)
			if !ok {
				continue
			}
			vc.reportValidityError(ctx, vc.filename, ce.Line(), elemDisplayName(ce),
				"This element is not expected, because the element '"+dn+"' is nilled.")
			return fmt.Errorf("content in nilled element")
		case helium.TextNode, helium.CDATASectionNode:
			if !isBlank(child.Content()) {
				vc.reportValidityError(ctx, vc.filename, elem.Line(), dn,
					"Character content is not allowed, because the element is nilled.")
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
	if base.Name.Local == "anyType" && base.Name.NS == lexicon.NamespaceXSD {
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
func (vc *validationContext) resolveXsiType(ctx context.Context, elem *helium.Element, declaredType *TypeDef) (*TypeDef, error) {
	var xsiTypeVal string
	for _, a := range elem.Attributes() {
		if a.URI() == lexicon.NamespaceXSI && a.LocalName() == attrType {
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
	if prefix, rest, ok := strings.Cut(xsiTypeVal, ":"); ok {
		local = rest
		ns = lookupNS(elem, prefix)
	} else {
		// No prefix — use the default namespace (empty prefix) or schema target namespace.
		ns = lookupNS(elem, "")
	}

	td, ok := vc.schema.LookupType(local, ns)
	if !ok {
		// Try with schema's target namespace.
		td, ok = vc.schema.LookupType(local, vc.schema.TargetNamespace())
	}
	if !ok {
		msg := fmt.Sprintf("The value '%s' of the xsi:type attribute does not resolve to a type definition.", xsiTypeVal)
		vc.reportValidityError(ctx, vc.filename, elem.Line(), elemDisplayName(elem), msg)
		return nil, fmt.Errorf("xsi:type not found")
	}

	// Check derivation: xsi:type must be the same as or derived from the declared type.
	if declaredType != nil && !isDerivedFrom(td, declaredType) {
		msg := fmt.Sprintf("The type definition '%s' is not validly derived from the type definition '%s'.",
			typeDisplayName(td), typeDisplayName(declaredType))
		vc.reportValidityError(ctx, vc.filename, elem.Line(), elemDisplayName(elem), msg)
		return nil, fmt.Errorf("xsi:type not derived")
	}

	return td, nil
}

// xsdTypeName converts a TypeDef to a type name string suitable for annotations.
// For anonymous types (no name), it walks up the base type chain to find the
// nearest named ancestor type, since XPath type checks need a concrete type name.
func xsdTypeName(td *TypeDef) string {
	if td == nil {
		return "xs:untyped"
	}
	if td.Name.NS == lexicon.NamespaceXSD {
		return "xs:" + td.Name.Local
	}
	if td.Name.NS != "" {
		return "Q{" + td.Name.NS + "}" + td.Name.Local
	}
	if td.Name.Local != "" {
		return "Q{}" + td.Name.Local
	}
	// Anonymous type: walk up the base type chain to find a named type.
	for cur := td.BaseType; cur != nil; cur = cur.BaseType {
		if cur.Name.NS == lexicon.NamespaceXSD {
			return "xs:" + cur.Name.Local
		}
		if cur.Name.NS != "" {
			return "Q{" + cur.Name.NS + "}" + cur.Name.Local
		}
		if cur.Name.Local != "" {
			return cur.Name.Local
		}
	}
	// Anonymous type with no named ancestor in the base chain: the type
	// was successfully validated, so it implicitly derives from xs:anyType.
	// Returning xs:untyped here would be wrong — xs:untyped means the
	// element was never validated, while xs:anyType means "validated but
	// the type is anonymous."
	return "xs:anyType"
}

// annotateElement records a type annotation for an element node.
func (vc *validationContext) annotateElement(_ context.Context, elem *helium.Element, td *TypeDef) {
	if vc.cfg == nil || vc.cfg.annotations == nil {
		return
	}
	(*vc.cfg.annotations)[elem] = xsdTypeName(td)
}

// attrUseType resolves the effective simple type for an attribute use. An inline
// anonymous <xs:simpleType> (au.Type) takes precedence over a named type
// reference (au.TypeName).
func (vc *validationContext) attrUseType(au *AttrUse) (*TypeDef, bool) {
	if au.Type != nil {
		return au.Type, true
	}
	if au.TypeName.Local == "" {
		return nil, false
	}
	return vc.schema.LookupType(au.TypeName.Local, au.TypeName.NS)
}

// annotateAttrUse records a type annotation for an attribute node based on its AttrUse declaration.
func (vc *validationContext) annotateAttrUse(_ context.Context, a *helium.Attribute, au *AttrUse) {
	if vc.cfg == nil || vc.cfg.annotations == nil {
		return
	}
	td, ok := vc.attrUseType(au)
	if !ok {
		return
	}
	(*vc.cfg.annotations)[a] = xsdTypeName(td)
}
