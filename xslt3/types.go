package xslt3

import (
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

// SequenceType represents a parsed XSLT/XPath sequence type declaration
// from the "as" attribute (e.g., "xs:string*", "element()+", "item()?").
type SequenceType struct {
	ItemType   string // "xs:string", "element()", "node()", "item()", etc.
	Occurrence rune   // 0 = exactly one, '?' = zero or one, '*' = zero or more, '+' = one or more
}

// parseSequenceType parses an "as" attribute value into a SequenceType.
// Examples: "item()*", "xs:string", "element()+", "xs:integer?", "text()", "empty-sequence()".
func parseSequenceType(as string) SequenceType {
	s := strings.TrimSpace(as)
	if s == "" {
		return SequenceType{ItemType: "item()", Occurrence: '*'}
	}

	// empty-sequence() is a special type that matches only the empty sequence
	if s == "empty-sequence()" {
		return SequenceType{ItemType: "empty-sequence()", Occurrence: 'E'}
	}

	// Check for occurrence indicator at the end
	var occ rune
	last := s[len(s)-1]
	switch last {
	case '?', '*', '+':
		occ = rune(last)
		s = strings.TrimSpace(s[:len(s)-1])
	}

	return SequenceType{ItemType: s, Occurrence: occ}
}

// checkSequenceType checks that a sequence matches the declared type.
// Returns the (possibly coerced) sequence on success, or an error on type mismatch.
func checkSequenceType(seq xpath3.Sequence, st SequenceType, errCode string, context string, ec ...*execContext) (xpath3.Sequence, error) {
	var execCtx *execContext
	if len(ec) > 0 {
		execCtx = ec[0]
	}
	// Check cardinality
	count := len(seq)
	switch st.Occurrence {
	case 'E': // empty-sequence() — must be empty
		if count != 0 {
			return nil, dynamicError(errCode, "%s: required empty-sequence(), got %d items", context, count)
		}
		return seq, nil
	case 0: // exactly one
		if count != 1 {
			return nil, dynamicError(errCode, "%s: required exactly 1 item, got %d", context, count)
		}
	case '?': // zero or one
		if count > 1 {
			return nil, dynamicError(errCode, "%s: required 0 or 1 items, got %d", context, count)
		}
	case '+': // one or more
		if count == 0 {
			return nil, dynamicError(errCode, "%s: required 1 or more items, got 0", context)
		}
	case '*': // zero or more — always valid
	}

	if count == 0 {
		return seq, nil
	}

	// Check/coerce item types
	result := make(xpath3.Sequence, 0, count)
	for _, item := range seq {
		coerced, err := coerceItem(item, st.ItemType, execCtx)
		if err != nil {
			return nil, dynamicError(errCode, "%s: %v", context, err)
		}
		result = append(result, coerced)
	}
	return result, nil
}

// coerceItem checks that a single item matches the expected type, applying
// atomization and casting as needed per the XSLT function conversion rules.
func coerceItem(item xpath3.Item, itemType string, ec ...*execContext) (xpath3.Item, error) {
	var execCtx *execContext
	if len(ec) > 0 {
		execCtx = ec[0]
	}
	return coerceItemWithContext(item, itemType, execCtx)
}

// coerceItemWithContext is the inner implementation of coerceItem with an explicit exec context.
func coerceItemWithContext(item xpath3.Item, itemType string, ec *execContext) (xpath3.Item, error) {
	// Strip outer parentheses from the type (e.g., "(function(...) as ...)" → "function(...) as ...")
	if len(itemType) > 2 && itemType[0] == '(' && itemType[len(itemType)-1] == ')' {
		itemType = itemType[1 : len(itemType)-1]
	}
	switch itemType {
	case "item()":
		// Anything matches item()
		return item, nil
	case "node()":
		if _, ok := item.(xpath3.NodeItem); ok {
			return item, nil
		}
		return nil, fmt.Errorf("expected node(), got atomic value")
	case "element()":
		if ni, ok := item.(xpath3.NodeItem); ok {
			if ni.Node.Type() == helium.ElementNode {
				return item, nil
			}
		}
		return nil, fmt.Errorf("expected element(), got %s", describeItem(item))
	case "attribute()":
		if ni, ok := item.(xpath3.NodeItem); ok {
			if ni.Node.Type() == helium.AttributeNode {
				return item, nil
			}
		}
		return nil, fmt.Errorf("expected attribute(), got %s", describeItem(item))
	case "text()":
		if ni, ok := item.(xpath3.NodeItem); ok {
			if ni.Node.Type() == helium.TextNode || ni.Node.Type() == helium.CDATASectionNode {
				return item, nil
			}
		}
		return nil, fmt.Errorf("expected text(), got %s", describeItem(item))
	case "comment()":
		if ni, ok := item.(xpath3.NodeItem); ok {
			if ni.Node.Type() == helium.CommentNode {
				return item, nil
			}
		}
		return nil, fmt.Errorf("expected comment(), got %s", describeItem(item))
	case "processing-instruction()":
		if ni, ok := item.(xpath3.NodeItem); ok {
			if ni.Node.Type() == helium.ProcessingInstructionNode {
				return item, nil
			}
		}
		return nil, fmt.Errorf("expected processing-instruction(), got %s", describeItem(item))
	case "document-node()":
		if ni, ok := item.(xpath3.NodeItem); ok {
			if ni.Node.Type() == helium.DocumentNode {
				return item, nil
			}
		}
		return nil, fmt.Errorf("expected document-node(), got %s", describeItem(item))
	}

	// Handle document-node(element(...)) — document with a specific root element.
	if strings.HasPrefix(itemType, "document-node(") {
		ni, ok := item.(xpath3.NodeItem)
		if !ok || ni.Node.Type() != helium.DocumentNode {
			return nil, fmt.Errorf("expected %s, got %s", itemType, describeItem(item))
		}
		// Extract the inner element test, e.g. "element(foo)" from "document-node(element(foo))".
		inner := strings.TrimSpace(itemType[len("document-node(") : len(itemType)-1])
		if inner == "" {
			return item, nil // document-node() without inner test — already matched above via switch
		}
		// Find the document element and check it against the inner element test.
		doc, isDoc := ni.Node.(*helium.Document)
		if !isDoc {
			return nil, fmt.Errorf("expected %s, got %s", itemType, describeItem(item))
		}
		rootElem := findDocumentElement(doc)
		if rootElem == nil {
			return nil, fmt.Errorf("expected %s: document has no root element", itemType)
		}
		rootItem := xpath3.NodeItem{Node: rootElem}
		if _, err := coerceItemWithContext(rootItem, inner, ec); err != nil {
			return nil, fmt.Errorf("expected %s: root element mismatch: %w", itemType, err)
		}
		return item, nil
	}

	// Handle attribute(name) / attribute(name, type) patterns
	if strings.HasPrefix(itemType, "attribute(") {
		if ni, ok := item.(xpath3.NodeItem); ok {
			if ni.Node.Type() == helium.AttributeNode {
				return item, nil
			}
		}
		return nil, fmt.Errorf("expected %s, got %s", itemType, describeItem(item))
	}

	// Handle map(*) — any map
	if strings.HasPrefix(itemType, "map(") {
		return checkMapItemType(item, itemType)
	}

	// Handle array(*) — any array
	if strings.HasPrefix(itemType, "array(") {
		// Arrays are represented as xpath3.Sequence values; skip strict check
		return item, nil
	}

	// Handle function(...) — any function
	if strings.HasPrefix(itemType, "function(") {
		return item, nil // function items are checked by the XPath layer
	}

	// Handle element(name) / element(name, type) patterns.
	if strings.HasPrefix(itemType, "element(") {
		ni, ok := item.(xpath3.NodeItem)
		if !ok || ni.Node.Type() != helium.ElementNode {
			return nil, fmt.Errorf("expected %s, got %s", itemType, describeItem(item))
		}
		inner := strings.TrimSpace(itemType[len("element(") : len(itemType)-1])
		if inner == "" || inner == "*" {
			return item, nil // element() or element(*) matches any element
		}
		// Split off optional type argument: element(name, type)
		parts := splitTopLevelTypeArgs(inner)
		reqName := strings.TrimSpace(parts[0])
		var reqTypeName string
		if len(parts) == 2 {
			reqTypeName = strings.TrimSpace(parts[1])
		}

		if reqName != "*" {
			// Resolve prefix:local to (local, ns) for namespace-aware comparison.
			reqLocal, reqNS := resolveSchemaQName(reqName, ec)
			elem, isElem := ni.Node.(*helium.Element)
			if !isElem {
				return nil, fmt.Errorf("expected %s, got non-element node", itemType)
			}
			if elem.LocalName() != reqLocal || elem.URI() != reqNS {
				return nil, fmt.Errorf("expected %s, got element %q", itemType, elem.LocalName())
			}
		}

		// If a type was specified, check the element's type annotation.
		if reqTypeName != "" && reqTypeName != "*" && ec != nil && ec.schemaRegistry != nil && ec.typeAnnotations != nil {
			ann := ec.typeAnnotations[ni.Node]
			if ann == "" {
				ann = "xs:untyped"
			}
			reqTypeNorm := normalizeTypeName(reqTypeName)
			if !ec.schemaRegistry.IsSubtypeOf(ann, reqTypeNorm) {
				return nil, fmt.Errorf("expected %s (type %s), element has type %s", itemType, reqTypeNorm, ann)
			}
		}
		return item, nil
	}

	// Handle schema-element(name) — verify element name and check schema declaration.
	if strings.HasPrefix(itemType, "schema-element(") {
		ni, ok := item.(xpath3.NodeItem)
		if !ok || ni.Node.Type() != helium.ElementNode {
			return nil, fmt.Errorf("expected %s, got %s", itemType, describeItem(item))
		}
		inner := strings.TrimSpace(itemType[len("schema-element(") : len(itemType)-1])
		reqLocal, reqNS := resolveSchemaQName(inner, ec)
		elem, isElem := ni.Node.(*helium.Element)
		if !isElem {
			return nil, fmt.Errorf("expected %s, got non-element node", itemType)
		}
		if elem.LocalName() != reqLocal || elem.URI() != reqNS {
			return nil, fmt.Errorf("expected %s, got element %q", itemType, elem.LocalName())
		}
		// Check type annotation against the schema-declared type.
		if ec != nil && ec.schemaRegistry != nil {
			declType, found := ec.schemaRegistry.LookupSchemaElement(reqLocal, reqNS)
			if found {
				ann := ""
				if ec.typeAnnotations != nil {
					ann = ec.typeAnnotations[ni.Node]
				}
				if ann == "" {
					ann = "xs:untyped"
				}
				if !ec.schemaRegistry.IsSubtypeOf(ann, declType) {
					return nil, fmt.Errorf("expected %s (type %s), element has type %s", itemType, declType, ann)
				}
			}
		}
		return item, nil
	}

	// Handle schema-attribute(name) — verify attribute name and check schema declaration.
	if strings.HasPrefix(itemType, "schema-attribute(") {
		ni, ok := item.(xpath3.NodeItem)
		if !ok || ni.Node.Type() != helium.AttributeNode {
			return nil, fmt.Errorf("expected %s, got %s", itemType, describeItem(item))
		}
		inner := strings.TrimSpace(itemType[len("schema-attribute(") : len(itemType)-1])
		reqLocal, reqNS := resolveSchemaQName(inner, ec)
		// Retrieve attribute local name and namespace.
		// Cast to *helium.Attribute to get LocalName() (Node.Name() includes prefix for attributes).
		attr, isAttr := ni.Node.(*helium.Attribute)
		if !isAttr {
			return nil, fmt.Errorf("expected %s, got non-attribute node", itemType)
		}
		attrLocal := attr.LocalName()
		attrNS := attr.URI()
		if attrLocal != reqLocal || attrNS != reqNS {
			return nil, fmt.Errorf("expected %s, got attribute %q", itemType, attrLocal)
		}
		// Check type annotation against the schema-declared type.
		if ec != nil && ec.schemaRegistry != nil {
			declType, found := ec.schemaRegistry.LookupSchemaAttribute(reqLocal, reqNS)
			if found {
				ann := ""
				if ec.typeAnnotations != nil {
					ann = ec.typeAnnotations[ni.Node]
				}
				if ann == "" {
					ann = "xs:untypedAtomic"
				}
				if !ec.schemaRegistry.IsSubtypeOf(ann, declType) {
					return nil, fmt.Errorf("expected %s (type %s), attribute has type %s", itemType, declType, ann)
				}
			}
		}
		return item, nil
	}

	// Atomic type — need to atomize and potentially cast
	return coerceToAtomicType(item, itemType)
}

// coerceToAtomicType atomizes a node/value and casts to the target atomic type.
func coerceToAtomicType(item xpath3.Item, targetType string) (xpath3.Item, error) {
	// If already an atomic value, check/cast the type
	if av, ok := item.(xpath3.AtomicValue); ok {
		return castAtomicToType(av, targetType)
	}

	// Node item — atomize first
	ni, ok := item.(xpath3.NodeItem)
	if !ok {
		return nil, fmt.Errorf("expected %s, got %s", targetType, describeItem(item))
	}

	av, err := xpath3.AtomizeItem(ni)
	if err != nil {
		return nil, fmt.Errorf("cannot atomize node for type %s: %w", targetType, err)
	}

	return castAtomicToType(av, targetType)
}

// castAtomicToType casts an atomic value to the specified target type.
func castAtomicToType(av xpath3.AtomicValue, targetType string) (xpath3.Item, error) {
	// Normalize target type
	target := normalizeTypeName(targetType)

	// If already the right type, return as-is
	if av.TypeName == target {
		return av, nil
	}

	// xs:anyAtomicType matches any atomic value
	if target == "xs:anyAtomicType" {
		return av, nil
	}

	// xs:anyURI -> xs:string promotion (per XPath spec)
	if av.TypeName == xpath3.TypeAnyURI && target == xpath3.TypeString {
		return xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: av.StringVal()}, nil
	}

	// xs:untypedAtomic and xs:string can be cast to any type
	if av.TypeName == xpath3.TypeUntypedAtomic || av.TypeName == xpath3.TypeString {
		s := av.StringVal()
		cast, err := xpath3.CastFromString(s, target)
		if err != nil {
			return nil, fmt.Errorf("cannot cast %q to %s: %w", s, targetType, err)
		}
		return cast, nil
	}

	// Numeric subtype acceptance: a value whose type is a subtype of the
	// target is accepted without changing its type (e.g., xs:integer IS xs:decimal).
	// Only demotion (wider→narrower, e.g., double→float) is rejected (XTTE0570).
	if isNumericType(target) && isNumericType(av.TypeName) {
		srcRank := numericRank(av.TypeName)
		tgtRank := numericRank(target)
		if srcRank > tgtRank {
			return nil, fmt.Errorf("cannot convert %s to %s", av.TypeName, targetType)
		}
		if srcRank <= tgtRank {
			// Subtype: accept as-is without casting (preserves original type)
			return av, nil
		}
	}

	// General fallback: try string-based casting
	s, err := xpath3.AtomicToString(av)
	if err != nil {
		return nil, fmt.Errorf("cannot convert %s to %s", av.TypeName, targetType)
	}
	cast, err := xpath3.CastFromString(s, target)
	if err != nil {
		return nil, fmt.Errorf("cannot convert %s to %s", av.TypeName, targetType)
	}
	return cast, nil
}

// normalizeTypeName normalizes a type name to include the xs: prefix.
func normalizeTypeName(name string) string {
	if strings.HasPrefix(name, "xs:") {
		return name
	}
	// Normalize xsd: prefix to xs: (both map to XML Schema namespace)
	if strings.HasPrefix(name, "xsd:") {
		return "xs:" + name[4:]
	}
	// Map unprefixed names to xs: prefixed
	switch name {
	case "string":
		return xpath3.TypeString
	case "integer":
		return xpath3.TypeInteger
	case "decimal":
		return xpath3.TypeDecimal
	case "double":
		return xpath3.TypeDouble
	case "float":
		return xpath3.TypeFloat
	case "boolean":
		return xpath3.TypeBoolean
	case "date":
		return xpath3.TypeDate
	case "dateTime":
		return xpath3.TypeDateTime
	case "time":
		return xpath3.TypeTime
	case "duration":
		return xpath3.TypeDuration
	case "dayTimeDuration":
		return xpath3.TypeDayTimeDuration
	case "yearMonthDuration":
		return xpath3.TypeYearMonthDuration
	case "anyURI":
		return xpath3.TypeAnyURI
	case "untypedAtomic":
		return xpath3.TypeUntypedAtomic
	}
	return name
}

// numericRank returns the promotion rank of a numeric type.
// Higher rank = wider type. Promotion is only valid from lower to higher rank.
func numericRank(t string) int {
	switch t {
	case xpath3.TypeInteger:
		return 1
	case xpath3.TypeDecimal:
		return 2
	case xpath3.TypeFloat:
		return 3
	case xpath3.TypeDouble:
		return 4
	}
	return 0
}

// isNumericType returns true for xs:integer, xs:decimal, xs:float, xs:double.
func isNumericType(t string) bool {
	switch t {
	case xpath3.TypeInteger, xpath3.TypeDecimal, xpath3.TypeFloat, xpath3.TypeDouble:
		return true
	}
	return false
}

// resolveSchemaQName resolves a QName string (e.g. "my:userNode" or "localName")
// to (localName, namespace) using the stylesheet's namespace bindings.
func resolveSchemaQName(qname string, ec *execContext) (local, ns string) {
	idx := strings.IndexByte(qname, ':')
	if idx < 0 {
		return qname, ""
	}
	prefix := qname[:idx]
	local = qname[idx+1:]
	if ec != nil && ec.stylesheet != nil {
		ns = ec.stylesheet.namespaces[prefix]
	}
	return local, ns
}

// describeItem returns a human-readable description of an item for error messages.
func describeItem(item xpath3.Item) string {
	switch v := item.(type) {
	case xpath3.NodeItem:
		switch v.Node.Type() {
		case helium.ElementNode:
			return "element node"
		case helium.AttributeNode:
			return "attribute node"
		case helium.TextNode, helium.CDATASectionNode:
			return "text node"
		case helium.CommentNode:
			return "comment node"
		case helium.ProcessingInstructionNode:
			return "processing-instruction node"
		case helium.DocumentNode:
			return "document node"
		default:
			return "node"
		}
	case xpath3.AtomicValue:
		return fmt.Sprintf("atomic value of type %s", v.TypeName)
	case xpath3.MapItem:
		return "map"
	default:
		return fmt.Sprintf("%T", item)
	}
}

func checkMapItemType(item xpath3.Item, itemType string) (xpath3.Item, error) {
	m, ok := item.(xpath3.MapItem)
	if !ok {
		return nil, fmt.Errorf("expected %s, got %s", itemType, describeItem(item))
	}

	keyType, valueType, hasMembers, err := parseMapItemType(itemType)
	if err != nil {
		return nil, err
	}
	if !hasMembers {
		return item, nil
	}

	for _, key := range m.Keys() {
		if !atomicMatchesType(key, keyType) {
			return nil, fmt.Errorf("map key %q does not match %s", key.StringVal(), keyType)
		}
		value, _ := m.Get(key)
		if !sequenceMatchesTypeStrict(value, valueType) {
			return nil, fmt.Errorf("map entry for key %q does not match %s", key.StringVal(), formatSequenceType(valueType))
		}
	}
	return item, nil
}

func parseMapItemType(itemType string) (string, SequenceType, bool, error) {
	inner := strings.TrimSpace(itemType[len("map(") : len(itemType)-1])
	if inner == "*" {
		return "", SequenceType{}, false, nil
	}

	parts := splitTopLevelTypeArgs(inner)
	if len(parts) != 2 {
		return "", SequenceType{}, false, fmt.Errorf("invalid map type %q", itemType)
	}
	return strings.TrimSpace(parts[0]), parseSequenceType(strings.TrimSpace(parts[1])), true, nil
}

func splitTopLevelTypeArgs(s string) []string {
	var parts []string
	depth := 0
	start := 0
	for i, r := range s {
		switch r {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, s[start:])
	return parts
}

func sequenceMatchesTypeStrict(seq xpath3.Sequence, st SequenceType) bool {
	count := len(seq)
	switch st.Occurrence {
	case 0:
		if count != 1 {
			return false
		}
	case '?':
		if count > 1 {
			return false
		}
	case '+':
		if count == 0 {
			return false
		}
	}

	for _, item := range seq {
		if !itemMatchesTypeStrict(item, st.ItemType) {
			return false
		}
	}
	return true
}

func itemMatchesTypeStrict(item xpath3.Item, itemType string) bool {
	switch itemType {
	case "item()":
		return true
	case "node()":
		_, ok := item.(xpath3.NodeItem)
		return ok
	case "element()":
		if ni, ok := item.(xpath3.NodeItem); ok {
			return ni.Node.Type() == helium.ElementNode
		}
		return false
	case "attribute()":
		if ni, ok := item.(xpath3.NodeItem); ok {
			return ni.Node.Type() == helium.AttributeNode
		}
		return false
	case "text()":
		if ni, ok := item.(xpath3.NodeItem); ok {
			return ni.Node.Type() == helium.TextNode || ni.Node.Type() == helium.CDATASectionNode
		}
		return false
	case "comment()":
		if ni, ok := item.(xpath3.NodeItem); ok {
			return ni.Node.Type() == helium.CommentNode
		}
		return false
	case "processing-instruction()":
		if ni, ok := item.(xpath3.NodeItem); ok {
			return ni.Node.Type() == helium.ProcessingInstructionNode
		}
		return false
	case "document-node()":
		if ni, ok := item.(xpath3.NodeItem); ok {
			return ni.Node.Type() == helium.DocumentNode
		}
		return false
	}

	if strings.HasPrefix(itemType, "map(") {
		_, err := checkMapItemType(item, itemType)
		return err == nil
	}
	if strings.HasPrefix(itemType, "array(") || strings.HasPrefix(itemType, "function(") {
		return true
	}

	av, ok := item.(xpath3.AtomicValue)
	if !ok {
		return false
	}
	return atomicMatchesType(av, itemType)
}

func atomicMatchesType(av xpath3.AtomicValue, targetType string) bool {
	target := normalizeTypeName(strings.TrimSpace(targetType))
	switch target {
	case "xs:anyAtomicType":
		return true
	case "xs:numeric":
		return isNumericType(av.TypeName)
	}
	if av.TypeName == target {
		return true
	}
	if target == xpath3.TypeDecimal && av.TypeName == xpath3.TypeInteger {
		return true
	}
	return false
}

func formatSequenceType(st SequenceType) string {
	if st.Occurrence == 0 {
		return st.ItemType
	}
	return st.ItemType + string(st.Occurrence)
}

// XSLT type error codes.
const (
	errCodeXTTE0505 = "XTTE0505" // template return type mismatch
	errCodeXTTE0570 = "XTTE0570" // variable/param type mismatch
	errCodeXTTE0780 = "XTTE0780" // function return type mismatch
)
