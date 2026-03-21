package xslt3

import (
	"context"
	"strconv"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xsd"
)

func (ec *execContext) execElement(ctx context.Context, inst *ElementInst) error {
	name, err := inst.Name.evaluate(ctx, ec.contextNode)
	if err != nil {
		return err
	}

	// XTDE0820: validate computed name is a valid QName
	name = strings.TrimSpace(name)
	if name == "" || !isValidQName(name) {
		return dynamicError(errCodeXTDE0820,
			"invalid element name %q: not a valid QName", name)
	}

	// Extract local name for element creation so SetActiveNamespace doesn't double the prefix
	localName := name
	prefix := ""
	if idx := strings.IndexByte(name, ':'); idx >= 0 {
		prefix = name[:idx]
		localName = name[idx+1:]
	}

	elem, err := ec.resultDoc.CreateElement(localName)
	if err != nil {
		return err
	}

	hasNS := false
	if inst.Namespace != nil {
		nsURI, err := inst.Namespace.evaluate(ctx, ec.contextNode)
		if err != nil {
			return err
		}
		if nsURI != "" {
			hasNS = true
			if err := elem.DeclareNamespace(prefix, nsURI); err != nil {
				return err
			}
			if err := elem.SetActiveNamespace(prefix, nsURI); err != nil {
				return err
			}
		}
	} else {
		// No namespace attribute: resolve from compile-time namespace context.
		uri := ""
		if inst.NSBindings != nil {
			uri = inst.NSBindings[prefix]
		}
		if uri == "" {
			uri = ec.resolvePrefix(prefix)
		}
		if uri != "" {
			hasNS = true
			if !ec.isNSDeclaredInScope(prefix, uri) {
				if err := elem.DeclareNamespace(prefix, uri); err != nil {
					return err
				}
			}
			if err := elem.SetActiveNamespace(prefix, uri); err != nil {
				return err
			}
		} else if prefix != "" {
			// XTDE0830: prefix in computed element name is undeclared
			return dynamicError(errCodeXTDE0830,
				"undeclared namespace prefix %q in element name %q", prefix, name)
		}
	}

	// Mark element as eligible for namespace fixup. The namespace
	// declaration from xsl:element is auto-generated from the name
	// resolution, so xsl:namespace can override the prefix.
	if hasNS && prefix != "" {
		if ec.nsFixupAllowed == nil {
			ec.nsFixupAllowed = make(map[*helium.Element]struct{})
		}
		ec.nsFixupAllowed[elem] = struct{}{}
	}

	// If this element has no namespace but there's a default namespace in scope,
	// we need to undeclare it with xmlns=""
	if !hasNS && prefix == "" && ec.hasDefaultNSInScope() {
		if err := elem.DeclareNamespace("", ""); err != nil {
			return err
		}
	}

	if err := ec.addNode(elem); err != nil {
		return err
	}

	if inst.TypeName != "" {
		ec.annotateNode(elem, inst.TypeName)
	}

	// Override static base URI when xsl:element carries xml:base.
	savedBaseOverride := ec.staticBaseURIOverride
	if inst.StaticBaseURI != "" {
		ec.staticBaseURIOverride = inst.StaticBaseURI
	}
	defer func() { ec.staticBaseURIOverride = savedBaseOverride }()

	// Push new output context for children.
	// Temporarily disable sequenceMode so that children are added to this
	// element normally (not captured as separate items in the sequence).
	out := ec.currentOutput()
	savedCurrent := out.current
	savedPrevAtomic := out.prevWasAtomic
	savedSeqMode := out.sequenceMode
	savedWherePop := out.wherePopulated
	out.current = elem
	out.prevWasAtomic = false
	out.sequenceMode = false
	// Clear wherePopulated inside the element body so that xsl:document
	// unwraps its children normally (same rationale as LRE — see
	// execLiteralResultElement).
	out.wherePopulated = false
	defer func() {
		out.current = savedCurrent
		out.prevWasAtomic = savedPrevAtomic
		out.sequenceMode = savedSeqMode
		out.wherePopulated = savedWherePop
	}()

	// Apply attribute sets (before body so body can override)
	if len(inst.UseAttributeSets) > 0 {
		if err := ec.applyAttributeSets(ctx, inst.UseAttributeSets); err != nil {
			return err
		}
	}
	if len(inst.UseAttrSets) > 0 {
		if err := ec.applyAttributeSets(ctx, inst.UseAttrSets); err != nil {
			return err
		}
	}

	if err := ec.executeSequenceConstructor(ctx, inst.Body); err != nil {
		return err
	}

	// inherit-namespaces="no": undeclare parent namespaces on direct
	// child elements so they do not inherit them via the DOM tree.
	if !inst.InheritNamespaces {
		undeclareInheritedNamespaces(elem)
	}

	// F.2: Type-based content validation and normalization.
	// When type="xs:integer" (or similar) is set, validate and normalize the
	// element's text content against the declared type, raising XTTE1510 on
	// failure.
	if inst.TypeName != "" {
		if err := ec.validateAndNormalizeElementContent(elem, inst.TypeName); err != nil {
			return err
		}
		ec.annotateAttributesFromType(elem, inst.TypeName)
	}

	if inst.Validation != "" {
		return ec.validateConstructedElement(ctx, elem, inst.Validation)
	}
	return nil
}

// validateConstructedAttribute validates an attribute value against the schema
// when validation="strict" or validation="lax". Returns XTTE1510 when the
// value is invalid, XTTE1555 when no matching global declaration is found
// (strict only).
func (ec *execContext) validateConstructedAttribute(localName, nsURI, value, validation string) error {
	if ec.schemaRegistry == nil {
		return nil
	}
	typeName, valid, valErr := ec.schemaRegistry.ValidateAttribute(localName, nsURI, value)
	if typeName == "" {
		// No matching global attribute declaration found.
		if validation == "strict" {
			return dynamicError(errCodeXTTE1555,
				"no schema declaration found for attribute {%s}%s (validation=strict)", nsURI, localName)
		}
		return nil // lax: silently skip if no declaration
	}
	if !valid || valErr != nil {
		return dynamicError(errCodeXTTE1510,
			"attribute {%s}%s value %q is not valid for type %s: %v", nsURI, localName, value, typeName, valErr)
	}
	return nil
}

// validateAndNormalizeElementContent validates the text content of elem against
// the declared XSD type name (e.g., "xs:integer"). It raises XTTE1510 when the
// content is invalid. The original text content is preserved in the DOM; the
// type annotation (stored separately) controls typed-value extraction via
// data()/atomization at runtime.
func (ec *execContext) validateAndNormalizeElementContent(elem *helium.Element, typeName string) error {
	// xs:anyType and xs:untyped accept any content — skip validation entirely.
	if typeName == "xs:anyType" || typeName == "xs:untyped" {
		return nil
	}

	// For user-defined types, look up the TypeDef from the schema registry.
	// Complex types need structural validation, not just string casting.
	// When multiple schemas define the same type (e.g., imported from
	// different schema files), try each until one validates successfully.
	if ec.schemaRegistry != nil {
		allDefs := ec.schemaRegistry.LookupAllTypeDefs(typeName)
		if len(allDefs) > 0 {
			var lastErr error
			for _, def := range allDefs {
				td, schema := def.TD, def.Schema
				switch td.ContentType {
				case xsd.ContentTypeElementOnly, xsd.ContentTypeMixed, xsd.ContentTypeEmpty:
					if err := xsd.ValidateElementAgainstType(elem, td, schema); err != nil {
						lastErr = err
						continue
					}
					return nil
				case xsd.ContentTypeSimple:
					content := strings.TrimSpace(elementTextContent(elem))
					if err := xsd.ValidateSimpleValue(content, td); err != nil {
						lastErr = err
						continue
					}
					return nil
				}
			}
			if lastErr != nil {
				return dynamicError(errCodeXTTE1510,
					"element content does not match declared type %s: %v", typeName, lastErr)
			}
		}
	}

	// Built-in XSD type: validate by attempting to cast.
	// Use text-only string value (skip comments/PIs) per XPath data model.
	content := strings.TrimSpace(elementTextContent(elem))
	_, castErr := xpath3.CastFromString(content, typeName)
	if castErr != nil {
		// Fall back to schema-defined simple type validation.
		if ec.schemaRegistry != nil {
			_, schemaErr := ec.schemaRegistry.CastToSchemaType(content, typeName)
			if schemaErr != nil {
				return dynamicError(errCodeXTTE1510,
					"content %q is not a valid value for type %s: %v", content, typeName, schemaErr)
			}
			return nil
		}
		return dynamicError(errCodeXTTE1510,
			"content %q is not a valid value for type %s: %v", content, typeName, castErr)
	}

	// Content is valid — original text is preserved in the DOM.
	return nil
}

// elementTextContent returns the concatenation of all descendant text nodes,
// skipping comments and processing instructions (XPath string-value semantics).
func elementTextContent(elem *helium.Element) string {
	var buf strings.Builder
	collectTextContent(elem.FirstChild(), &buf)
	return buf.String()
}

func collectTextContent(node helium.Node, buf *strings.Builder) {
	for ; node != nil; node = node.NextSibling() {
		switch node.Type() {
		case helium.TextNode, helium.CDATASectionNode:
			buf.Write(node.Content())
		case helium.ElementNode:
			if elem, ok := node.(*helium.Element); ok {
				collectTextContent(elem.FirstChild(), buf)
			}
		}
	}
}

// validateConstructedElement validates a constructed element node against the
// imported schemas and applies type annotations to the result tree.
func (ec *execContext) validateConstructedElement(ctx context.Context, elem *helium.Element, validation string) error {
	switch validation {
	case "strip":
		ec.stripAnnotations(elem)
		return nil
	case "preserve":
		// preserve: keep any existing type annotations unchanged; nothing to do here
		return nil
	case "strict", "lax":
		if ec.schemaRegistry == nil {
			return nil
		}
		// Create a temporary document containing a deep copy of the element.
		tmpDoc := helium.NewDefaultDocument()
		copied, err := helium.CopyNode(elem, tmpDoc)
		if err != nil {
			return err
		}
		if err := tmpDoc.AddChild(copied); err != nil {
			return err
		}
		ann, valErr := ec.schemaRegistry.ValidateDoc(ctx, tmpDoc)
		if valErr != nil {
			switch validation {
			case "strict":
				return dynamicError(errCodeXTTE1510, "validation of constructed element failed: %v", valErr)
			case "lax":
				return dynamicError(errCodeXTTE1515, "lax validation of constructed element failed: %v", valErr)
			}
		} else if ann == nil {
			// No matching schema found.
			if validation == "strict" {
				// Strict validation: unknown validity = failure if element is in a
				// schema-governed namespace but has no matching declaration.
				elemNS := elem.URI()
				elemLocal := elem.LocalName()
				if _, found := ec.schemaRegistry.LookupElement(elemLocal, elemNS); !found {
					return dynamicError(errCodeXTTE1510,
						"no schema declaration found for element {%s}%s (validation=strict)", elemNS, elemLocal)
				}
			}
		} else {
			// Schema validation passed (ann != nil). Perform additional value-level
			// validation using CastFromString to catch calendar-invalid dates etc.
			// (the XSD regex validator may accept e.g. "2006-02-31" as a valid date).
			elemNS := elem.URI()
			elemLocal := elem.LocalName()
			if typeName, found := ec.schemaRegistry.LookupElement(elemLocal, elemNS); found && typeName != "" {
				// Only validate for simple types (built-in xs: types); skip for complex types.
				if isBuiltinSimpleType(typeName) {
					content := strings.TrimSpace(string(elem.Content()))
					if _, castErr := xpath3.CastFromString(content, typeName); castErr != nil {
						switch validation {
						case "strict":
							return dynamicError(errCodeXTTE1510,
								"element {%s}%s content %q is not valid for type %s: %v",
								elemNS, elemLocal, content, typeName, castErr)
						case "lax":
							return dynamicError(errCodeXTTE1515,
								"element {%s}%s content %q is not valid for type %s: %v",
								elemNS, elemLocal, content, typeName, castErr)
						}
					}
				}
			}
		}
		// Merge type annotations for the actual (non-copy) element by walking
		// the temp tree and live tree in parallel.
		if len(ann) > 0 {
			ec.mapAnnotationsFromValidation(ann, copied, elem)
		}
		return nil
	}
	return nil
}

// mapAnnotationsFromValidation maps type annotations from a validated copy
// tree back to the corresponding live tree nodes.
func (ec *execContext) mapAnnotationsFromValidation(ann xsd.TypeAnnotations, src, dst helium.Node) {
	if typeName, ok := ann[src]; ok {
		ec.annotateNode(dst, typeName)
	}
	// Map attribute annotations and copy default/fixed attributes from validated copy.
	if srcElem, ok := src.(*helium.Element); ok {
		if dstElem, ok := dst.(*helium.Element); ok {
			for _, srcAttr := range srcElem.Attributes() {
				// Check if this attribute exists on the destination.
				dstFound := false
				for _, dstAttr := range dstElem.Attributes() {
					if srcAttr.LocalName() == dstAttr.LocalName() && srcAttr.URI() == dstAttr.URI() {
						dstFound = true
						if typeName, ok := ann[srcAttr]; ok {
							ec.annotateNode(dstAttr, typeName)
						}
						break
					}
				}
				// Attribute exists on copy but not on original — it was inserted
				// as a default/fixed value by schema validation. Copy it over.
				if !dstFound {
					// Copy the default/fixed attribute from the validated copy.
					dstElem.SetLiteralAttribute(srcAttr.Name(), srcAttr.Value())
					// Annotate the newly added attribute.
					if typeName, ok := ann[srcAttr]; ok {
						for _, dstAttr := range dstElem.Attributes() {
							if srcAttr.LocalName() == dstAttr.LocalName() && srcAttr.URI() == dstAttr.URI() {
								ec.annotateNode(dstAttr, typeName)
								break
							}
						}
					}
				}
			}
		}
	}
	// Recurse into children
	srcChild := src.FirstChild()
	dstChild := dst.FirstChild()
	for srcChild != nil && dstChild != nil {
		ec.mapAnnotationsFromValidation(ann, srcChild, dstChild)
		srcChild = srcChild.NextSibling()
		dstChild = dstChild.NextSibling()
	}
}

// stripAnnotations removes type annotations from a node and all its descendants.
func (ec *execContext) stripAnnotations(node helium.Node) {
	if ec.typeAnnotations == nil {
		return
	}
	delete(ec.typeAnnotations, node)
	// Also strip annotations from attributes on elements.
	if elem, ok := node.(*helium.Element); ok {
		for _, attr := range elem.Attributes() {
			delete(ec.typeAnnotations, attr)
		}
	}
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		ec.stripAnnotations(child)
	}
}

// annotateAttributesFromType annotates each attribute on elem with the type
// declared in the complex type definition identified by typeName.  This is
// needed so that "instance of attribute(*, xs:integer)" works on attributes
// of elements constructed with xsl:type or type="…".
func (ec *execContext) annotateAttributesFromType(elem *helium.Element, typeName string) {
	if ec.schemaRegistry == nil {
		return
	}
	allDefs := ec.schemaRegistry.LookupAllTypeDefs(typeName)
	if len(allDefs) == 0 {
		return
	}
	td := allDefs[0].TD
	if len(td.Attributes) == 0 {
		return
	}
	const nsXSD = "http://www.w3.org/2001/XMLSchema"
	for _, attr := range elem.Attributes() {
		for _, au := range td.Attributes {
			if au.Name.Local == attr.LocalName() && au.Name.NS == attr.URI() {
				if au.TypeName.Local != "" {
					var ann string
					if au.TypeName.NS == nsXSD {
						ann = "xs:" + au.TypeName.Local
					} else {
						ann = xpath3.QAnnotation(au.TypeName.NS, au.TypeName.Local)
					}
					ec.annotateNode(attr, ann)
				}
				break
			}
		}
	}
}

func (ec *execContext) execAttribute(ctx context.Context, inst *AttributeInst) error {
	name, err := inst.Name.evaluate(ctx, ec.contextNode)
	if err != nil {
		return err
	}

	var value string
	if inst.Select != nil {
		sep := " "
		if inst.Separator != nil {
			sep, err = inst.Separator.evaluate(ctx, ec.contextNode)
			if err != nil {
				return err
			}
		}
		xpathCtx := ec.newXPathContext(ec.contextNode)
		result, err := inst.Select.Evaluate(xpathCtx, ec.contextNode)
		if err != nil {
			return err
		}
		value = stringifyResultWithSep(result, sep)
	} else if len(inst.Body) > 0 {
		sep := ""
		if inst.Separator != nil {
			sep, err = inst.Separator.evaluate(ctx, ec.contextNode)
			if err != nil {
				return err
			}
		}
		// XSLT 2.0: attribute body is temporary output state (XTDE1480).
		// XSLT 3.0 relaxes this restriction.
		isV2TempOutput := ec.stylesheet.version != "" && ec.stylesheet.version < "3.0"
		if isV2TempOutput {
			ec.temporaryOutputDepth++
		}
		val, err := ec.evaluateBodyForAttr(ctx, inst.Body)
		if isV2TempOutput {
			ec.temporaryOutputDepth--
		}
		if err != nil {
			return err
		}
		// Per XSLT spec §5.7.2: adjacent text nodes in the result are merged
		// before the separator is applied.
		val = mergeAdjacentTextNodes(val)
		value = stringifySequenceWithSep(val, sep)
	}

	// When a type annotation is present (e.g. type="xs:integer"), cast the
	// string value to the target type and back so that the canonical lexical
	// form is used (e.g. "0023" → 23 → "23").
	if inst.TypeName != "" {
		av, castErr := xpath3.CastFromString(value, inst.TypeName)
		if castErr != nil {
			// Fall back to schema-defined type validation for user-defined types.
			if ec.schemaRegistry != nil {
				normalized, schemaErr := ec.schemaRegistry.CastToSchemaType(value, inst.TypeName)
				if schemaErr != nil {
					return dynamicError(errCodeXTTE1515,
						"attribute value %q does not match type %s: %v", value, inst.TypeName, schemaErr)
				}
				value = normalized
			} else {
				return dynamicError(errCodeXTTE1515, "attribute value %q does not match type %s", value, inst.TypeName)
			}
		} else if s, sErr := xpath3.AtomicToString(av); sErr == nil {
			value = s
		}
	}

	// In sequence mode (variable/param with as), capture the attribute as a
	// standalone item rather than attaching it to an element.
	out := ec.currentOutput()
	if out.sequenceMode {
		// Resolve namespace for prefixed attribute names.
		var attrNS *helium.Namespace
		if idx := strings.IndexByte(name, ':'); idx >= 0 {
			prefix := name[:idx]
			if nsURI := ec.resolvePrefix(prefix); nsURI != "" {
				ns, _ := out.doc.CreateNamespace(prefix, nsURI)
				attrNS = ns
			}
		}
		attr, attrErr := out.doc.CreateAttribute(name, value, attrNS)
		if attrErr != nil {
			return attrErr
		}
		ni := xpath3.NodeItem{Node: attr}
		if inst.TypeName != "" {
			ec.annotateNode(attr, inst.TypeName)
			ni.TypeAnnotation = inst.TypeName
			// Set ListItemType for list types (built-in or schema-defined).
			if ec.schemaRegistry != nil {
				if itemType, ok := ec.schemaRegistry.ListItemType(inst.TypeName); ok {
					ni.ListItemType = itemType
				}
			}
		}
		out.pendingItems = append(out.pendingItems, ni)
		out.noteOutput()
		return nil
	}

	// Inside xsl:where-populated, a zero-length attribute is treated as absent
	// (XSLT 3.0 §11.1.8): skip it so it does not overwrite a non-empty value.
	if out.wherePopulated && value == "" {
		return nil
	}

	// The current output node must be an element
	elem, ok := out.current.(*helium.Element)
	if !ok {
		return dynamicError(errCodeXTDE0820, "xsl:attribute must be added to an element")
	}

	// XTRE0540: cannot add attribute after child content has been added.
	// Inside xsl:where-populated the body is evaluated into a temporary tree
	// and later filtered, so attribute-after-child is permitted during evaluation.
	if elem.FirstChild() != nil && !out.wherePopulated {
		return dynamicError(errCodeXTRE0540, "cannot add attribute to element after children have been added")
	}

	if inst.Namespace != nil {
		nsURI, err := inst.Namespace.evaluate(ctx, ec.contextNode)
		if err != nil {
			return err
		}
		if nsURI != "" {
			prefix := ""
			localName := name
			if idx := strings.IndexByte(name, ':'); idx >= 0 {
				prefix = name[:idx]
				localName = name[idx+1:]
			}
			// Attributes in a namespace require a non-empty prefix (unlike
			// elements, the default namespace does not apply to attributes).
			if prefix == "" {
				prefix = "ns0"
			}
			// Schema validation when validation attribute is set.
			if inst.Validation == "strict" || inst.Validation == "lax" {
				if err := ec.validateConstructedAttribute(localName, nsURI, value, inst.Validation); err != nil {
					return err
				}
			}
			// If the prefix is already bound to a different URI on this element,
			// generate a unique prefix to avoid conflicts.
			prefix = uniqueNSPrefix(elem, prefix, nsURI)
			// Ensure the namespace is declared on the element
			if !hasNSDecl(elem, prefix, nsURI) {
				if err := elem.DeclareNamespace(prefix, nsURI); err != nil {
					return err
				}
			}
			// Remove existing attribute with same expanded name to allow replacement
			elem.RemoveAttributeNS(localName, nsURI)
			ns, err := ec.resultDoc.CreateNamespace(prefix, nsURI)
			if err != nil {
				return err
			}
			// Use literal mode: XSLT evaluation values are plain text
			// that may contain & from resolved entities.
			elem.SetLiteralAttributeNS(localName, value, ns)
			ec.annotateAttr(elem, inst.TypeName, localName, nsURI, value)
			out.noteOutput()
			return nil
		}
		// namespace="" explicitly: strip prefix, use no-namespace attribute
		if idx := strings.IndexByte(name, ':'); idx >= 0 {
			name = name[idx+1:]
		}
		// Schema validation for no-namespace attribute.
		if inst.Validation == "strict" || inst.Validation == "lax" {
			if err := ec.validateConstructedAttribute(name, "", value, inst.Validation); err != nil {
				return err
			}
		}
		elem.RemoveAttribute(name)
		elem.SetLiteralAttribute(name, value)
		ec.annotateAttr(elem, inst.TypeName, name, "", value)
		out.noteOutput()
		return nil
	}

	// Handle prefixed attribute names without explicit namespace
	if idx := strings.IndexByte(name, ':'); idx >= 0 {
		prefix := name[:idx]
		localName := name[idx+1:]
		uri := ec.resolvePrefix(prefix)
		if uri == "" {
			// XTDE0860: prefix in computed attribute name is undeclared
			return dynamicError(errCodeXTDE0860,
				"undeclared namespace prefix %q in attribute name %q", prefix, name)
		}
		// Schema validation for prefixed attribute.
		if inst.Validation == "strict" || inst.Validation == "lax" {
			if err := ec.validateConstructedAttribute(localName, uri, value, inst.Validation); err != nil {
				return err
			}
		}
		// Ensure the namespace is declared on the element
		if !hasNSDecl(elem, prefix, uri) {
			if err := elem.DeclareNamespace(prefix, uri); err != nil {
				return err
			}
		}
		// Remove existing attribute with same expanded name to allow replacement
		elem.RemoveAttributeNS(localName, uri)
		ns, err := ec.resultDoc.CreateNamespace(prefix, uri)
		if err != nil {
			return err
		}
		elem.SetLiteralAttributeNS(localName, value, ns)
		ec.annotateAttr(elem, inst.TypeName, localName, uri, value)
		out.noteOutput()
		return nil
	}

	// Schema validation for simple unprefixed attribute.
	if inst.Validation == "strict" || inst.Validation == "lax" {
		if err := ec.validateConstructedAttribute(name, "", value, inst.Validation); err != nil {
			return err
		}
	}

	// Remove existing attribute with same name to allow replacement
	elem.RemoveAttribute(name)
	elem.SetLiteralAttribute(name, value)
	ec.annotateAttr(elem, inst.TypeName, name, "", value)
	out.noteOutput()
	return nil
}

// copyAttributeToElement copies an attribute to an element, preserving its
// namespace URI and prefix. For non-namespaced attributes, falls back to
// SetLiteralAttribute.
func copyAttributeToElement(elem *helium.Element, attr *helium.Attribute) {
	if uri := attr.URI(); uri != "" {
		prefix := attr.Prefix()
		// Extract local name by stripping prefix from Name()
		name := attr.Name()
		localName := name
		if prefix != "" {
			localName = name[len(prefix)+1:]
		}
		ns := helium.NewNamespace(prefix, uri)
		elem.SetLiteralAttributeNS(localName, attr.Value(), ns)
		return
	}
	elem.SetLiteralAttribute(attr.Name(), attr.Value())
}

// hasNSDecl checks if an element already has a namespace declaration for
// the given prefix and URI.
func hasNSDecl(elem *helium.Element, prefix, uri string) bool {
	for _, ns := range elem.Namespaces() {
		if ns.Prefix() == prefix && ns.URI() == uri {
			return true
		}
	}
	return false
}

// collectInScopeNamespaces collects all in-scope namespace declarations for
// an element, walking up the ancestor chain.  Declarations closer to the
// element take precedence (first-seen prefix wins).
func collectInScopeNamespaces(elem *helium.Element) []*helium.Namespace {
	seen := make(map[string]struct{})
	var result []*helium.Namespace
	for cur := elem; cur != nil; {
		for _, ns := range cur.Namespaces() {
			if _, ok := seen[ns.Prefix()]; !ok {
				seen[ns.Prefix()] = struct{}{}
				result = append(result, ns)
			}
		}
		p := cur.Parent()
		if p == nil {
			break
		}
		pe, ok := p.(*helium.Element)
		if !ok {
			break
		}
		cur = pe
	}
	return result
}

// undeclareInheritedNamespaces adds namespace undeclarations (xmlns:p="") on
// each direct child element for every in-scope namespace visible from parent
// that the child does not itself declare.  This implements the XSLT 3.0
// inherit-namespaces="no" semantics: children must not inherit ANY namespace
// reachable through the parent in the DOM tree.
func undeclareInheritedNamespaces(parent *helium.Element) {
	// Collect all in-scope namespace prefixes visible from the parent,
	// including those inherited from grandparent and beyond.
	inScope := make(map[string]struct{})
	for cur := parent; cur != nil; {
		for _, ns := range cur.Namespaces() {
			if _, ok := inScope[ns.Prefix()]; !ok {
				inScope[ns.Prefix()] = struct{}{}
			}
		}
		p := cur.Parent()
		if p == nil {
			break
		}
		pe, ok := p.(*helium.Element)
		if !ok {
			break
		}
		cur = pe
	}
	if len(inScope) == 0 {
		return
	}
	for child := parent.FirstChild(); child != nil; child = child.NextSibling() {
		childElem, ok := child.(*helium.Element)
		if !ok {
			continue
		}
		for prefix := range inScope {
			// Skip if the child already has an explicit declaration for this prefix.
			alreadyDeclared := false
			for _, cns := range childElem.Namespaces() {
				if cns.Prefix() == prefix {
					alreadyDeclared = true
					break
				}
			}
			if alreadyDeclared {
				continue
			}
			// If the child element itself uses this prefix (i.e., its
			// namespace matches the inherited binding), add an explicit
			// declaration rather than undeclaring. This preserves the
			// element's own namespace while blocking further inheritance.
			childPrefix := childElem.Prefix()
			childURI := childElem.URI()
			if childPrefix == prefix && childURI != "" {
				_ = childElem.DeclareNamespace(prefix, childURI)
				continue
			}
			// Add an undeclaration (empty URI) so the prefix is not visible
			// when walking the ancestor chain.
			_ = childElem.DeclareNamespace(prefix, "")
		}
	}
}

// uniqueNSPrefix returns a prefix for nsURI that doesn't conflict with
// in-scope namespace declarations on elem or its ancestors. If prefix is
// already bound to nsURI, it's returned as-is. If it's bound to a different
// URI, a suffix like _1, _2, ... is appended until a unique prefix is found.
func uniqueNSPrefix(elem *helium.Element, prefix, nsURI string) string {
	if prefixBoundTo(elem, prefix) == nsURI {
		return prefix
	}
	if uri := prefixBoundTo(elem, prefix); uri != "" && uri != nsURI {
		for i := 1; ; i++ {
			candidate := prefix + "_" + strconv.Itoa(i)
			if prefixBoundTo(elem, candidate) == "" {
				return candidate
			}
		}
	}
	return prefix
}

// prefixBoundTo walks the element and its ancestors to find what URI
// a prefix is bound to. Returns "" if not found.
func prefixBoundTo(elem *helium.Element, prefix string) string {
	for node := helium.Node(elem); node != nil; node = node.Parent() {
		e, ok := node.(*helium.Element)
		if !ok {
			continue
		}
		for _, ns := range e.Namespaces() {
			if ns.Prefix() == prefix {
				return ns.URI()
			}
		}
	}
	return ""
}

func (ec *execContext) execComment(ctx context.Context, inst *CommentInst) error {
	var value string
	if inst.Select != nil {
		xpathCtx := ec.newXPathContext(ec.contextNode)
		result, err := inst.Select.Evaluate(xpathCtx, ec.contextNode)
		if err != nil {
			return err
		}
		value = stringifyResult(result)
	} else if len(inst.Body) > 0 {
		// XSLT 2.0: comment body is temporary output state (XTDE1480).
		// XSLT 3.0 relaxes this restriction.
		isV2TempOutput := ec.stylesheet.version != "" && ec.stylesheet.version < "3.0"
		if isV2TempOutput {
			ec.temporaryOutputDepth++
		}
		val, err := ec.evaluateBody(ctx, inst.Body)
		if isV2TempOutput {
			ec.temporaryOutputDepth--
		}
		if err != nil {
			return err
		}
		value = stringifySequence(val)
	}

	// Sanitize comment content per XSLT 3.0 spec §11.1:
	// Replace any occurrence of "--" with "- -" and ensure the value
	// doesn't end with "-" (add a trailing space if so).
	value = sanitizeComment(value)

	comment, err := ec.resultDoc.CreateComment([]byte(value))
	if err != nil {
		return err
	}
	return ec.addNode(comment)
}

// sanitizeComment replaces "--" sequences with "- -" and ensures the
// value does not end with "-", per XSLT comment construction rules.
func sanitizeComment(s string) string {
	var sb strings.Builder
	sb.Grow(len(s))
	prevDash := false
	for i := 0; i < len(s); i++ {
		if s[i] == '-' {
			if prevDash {
				sb.WriteByte(' ')
			}
			sb.WriteByte('-')
			prevDash = true
		} else {
			sb.WriteByte(s[i])
			prevDash = false
		}
	}
	result := sb.String()
	if len(result) > 0 && result[len(result)-1] == '-' {
		result += " "
	}
	return result
}

func (ec *execContext) execPI(ctx context.Context, inst *PIInst) error {
	name, err := inst.Name.evaluate(ctx, ec.contextNode)
	if err != nil {
		return err
	}

	var value string
	if inst.Select != nil {
		xpathCtx := ec.newXPathContext(ec.contextNode)
		result, err := inst.Select.Evaluate(xpathCtx, ec.contextNode)
		if err != nil {
			return err
		}
		value = stringifyResult(result)
	} else if len(inst.Body) > 0 {
		// XSLT 2.0: PI body is temporary output state (XTDE1480).
		// XSLT 3.0 relaxes this restriction.
		isV2TempOutput := ec.stylesheet.version != "" && ec.stylesheet.version < "3.0"
		if isV2TempOutput {
			ec.temporaryOutputDepth++
		}
		val, err := ec.evaluateBody(ctx, inst.Body)
		if isV2TempOutput {
			ec.temporaryOutputDepth--
		}
		if err != nil {
			return err
		}
		value = stringifySequence(val)
	}

	// XSLT 3.0 §11.6: replace "?>" in PI content with "? >" to avoid
	// premature termination of the processing instruction.
	value = strings.ReplaceAll(value, "?>", "? >")

	pi, err := ec.resultDoc.CreatePI(name, value)
	if err != nil {
		return err
	}
	return ec.addNode(pi)
}

func (ec *execContext) execNamespace(ctx context.Context, inst *NamespaceInst) error {
	name, err := inst.Name.evaluate(ctx, ec.contextNode)
	if err != nil {
		return err
	}

	var value string
	if inst.Select != nil {
		xpathCtx := ec.newXPathContext(ec.contextNode)
		result, evalErr := inst.Select.Evaluate(xpathCtx, ec.contextNode)
		if evalErr != nil {
			return evalErr
		}
		value = stringifyResult(result)
	} else if len(inst.Body) > 0 {
		// XSLT 2.0: namespace body is temporary output state (XTDE1480).
		// XSLT 3.0 relaxes this restriction.
		isV2TempOutput := ec.stylesheet.version != "" && ec.stylesheet.version < "3.0"
		if isV2TempOutput {
			ec.temporaryOutputDepth++
		}
		val, bodyErr := ec.evaluateBody(ctx, inst.Body)
		if isV2TempOutput {
			ec.temporaryOutputDepth--
		}
		if bodyErr != nil {
			return bodyErr
		}
		value = stringifySequence(val)
	}

	out := ec.currentOutput()
	// In sequence mode, capture the namespace node as a standalone item.
	if out.sequenceMode {
		ns := helium.NewNamespace(name, value)
		nsNode := helium.NewNamespaceNodeWrapper(ns, nil)
		out.pendingItems = append(out.pendingItems, xpath3.NodeItem{Node: nsNode})
		out.noteOutput()
		return nil
	}
	elem, ok := out.current.(*helium.Element)
	if !ok {
		return dynamicError(errCodeXTDE0420,
			"cannot add namespace node to a non-element node")
	}

	// XTDE0430: it is a non-recoverable dynamic error if two namespace
	// nodes for the same element have the same prefix but different URIs.
	// Exception: when the element's prefix namespace was auto-generated
	// (from xsl:element name resolution), namespace fixup can rename the
	// element's prefix instead of raising an error.
	_, fixupOK := ec.nsFixupAllowed[elem]
	for _, ns := range elem.Namespaces() {
		if ns.Prefix() == name && ns.URI() != value {
			if fixupOK && elem.Prefix() == name && elem.URI() == ns.URI() {
				continue // allow fixup below
			}
			return dynamicError(errCodeXTDE0430,
				"namespace prefix %q is already bound to %q; cannot rebind to %q", name, ns.URI(), value)
		}
	}

	// If the new namespace binding conflicts with the element's own prefix
	// (same prefix, different URI), rename the element's prefix to avoid
	// the collision via namespace fixup.
	if fixupOK && name != "" && elem.Prefix() == name && elem.URI() != value {
		origURI := elem.URI()
		newPrefix := uniqueNSPrefix(elem, name+"_0", origURI)
		elem.RemoveNamespaceByPrefix(name)
		if err := elem.DeclareNamespace(newPrefix, origURI); err != nil {
			return err
		}
		if err := elem.SetActiveNamespace(newPrefix, origURI); err != nil {
			return err
		}
	}

	if err := elem.DeclareNamespace(name, value); err != nil {
		return err
	}
	out.noteOutput()
	return nil
}

