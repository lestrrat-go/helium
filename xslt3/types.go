package xslt3

import (
	"fmt"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/lestrrat-go/helium/internal/sequence"
)

// sequenceType represents a parsed XSLT/XPath sequence type declaration
// from the "as" attribute (e.g., "xs:string*", "element()+", "item()?").
type sequenceType struct {
	ItemType   string // "xs:string", "element()", "node()", "item()", etc.
	Occurrence rune   // 0 = exactly one, '?' = zero or one, '*' = zero or more, '+' = one or more
}

// parseSequenceType parses an "as" attribute value into a sequenceType.
// Examples: "item()*", "xs:string", "element()+", "xs:integer?", "text()", "empty-sequence()".
func parseSequenceType(as string) sequenceType {
	s := stripXPathComments(strings.TrimSpace(as))
	if s == "" {
		return sequenceType{ItemType: "item()", Occurrence: '*'}
	}

	// empty-sequence() is a special type that matches only the empty sequence
	if s == "empty-sequence()" {
		return sequenceType{ItemType: "empty-sequence()", Occurrence: 'E'}
	}

	// Check for occurrence indicator at the end
	var occ rune
	last := s[len(s)-1]
	switch last {
	case '?', '*', '+':
		occ = rune(last)
		s = strings.TrimSpace(s[:len(s)-1])
	}

	return sequenceType{ItemType: s, Occurrence: occ}
}

// allowsEmptySequence returns true if the given 'as' type string allows
// an empty sequence (zero items). Types ending with ? or * allow empty;
// types without an occurrence indicator or with + require at least one item.
func allowsEmptySequence(as string) bool {
	s := strings.TrimSpace(as)
	if s == "" || s == "empty-sequence()" {
		return true
	}
	last := s[len(s)-1]
	return last == '?' || last == '*'
}

// checkSequenceType checks that a sequence matches the declared type.
// Returns the (possibly coerced) sequence on success, or an error on type mismatch.
func checkSequenceType(seq xpath3.Sequence, st sequenceType, errCode string, context string, ec ...*execContext) (xpath3.Sequence, error) {
	var execCtx *execContext
	if len(ec) > 0 {
		execCtx = ec[0]
	}

	// When the target type is atomic, pre-atomize the sequence so that
	// list-type nodes (e.g. xs:list of xs:decimal) expand into multiple
	// atomic items before cardinality and type checks.
	if isAtomicTargetType(st.ItemType) && seq != nil {
		atomized, err := xpath3.AtomizeSequence(seq)
		if err == nil && len(atomized) > 0 {
			items := make(xpath3.ItemSlice, len(atomized))
			for i, av := range atomized {
				items[i] = av
			}
			seq = items
		}
	}

	// Check cardinality
	count := 0
	if seq != nil {
		count = sequence.Len(seq)
	}
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
	result := make(xpath3.ItemSlice, 0, count)
	for item := range sequence.Items(seq) {
		coerced, err := coerceItem(item, st.ItemType, execCtx)
		if err != nil {
			return nil, dynamicError(errCode, "%s: %v", context, err)
		}
		result = append(result, coerced)
	}
	return result, nil
}

// isAtomicTargetType returns true when the item type string names an atomic
// type (e.g. "xs:decimal", "xs:string") rather than a node/function/map test.
func isAtomicTargetType(itemType string) bool {
	if itemType == "" || itemType == "item()" {
		return false
	}
	// Node tests, function tests, map/array tests are not atomic.
	if strings.HasPrefix(itemType, "node(") ||
		strings.HasPrefix(itemType, "element(") ||
		strings.HasPrefix(itemType, "attribute(") ||
		strings.HasPrefix(itemType, "document-node(") ||
		strings.HasPrefix(itemType, "text(") ||
		strings.HasPrefix(itemType, "comment(") ||
		strings.HasPrefix(itemType, "processing-instruction(") ||
		strings.HasPrefix(itemType, "namespace-node(") ||
		strings.HasPrefix(itemType, "schema-element(") ||
		strings.HasPrefix(itemType, "schema-attribute(") ||
		strings.HasPrefix(itemType, "function(") ||
		strings.HasPrefix(itemType, "map(") ||
		strings.HasPrefix(itemType, "array(") {
		return false
	}
	return true
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
	case "namespace-node()":
		if ni, ok := item.(xpath3.NodeItem); ok {
			if ni.Node.Type() == helium.NamespaceNode {
				return item, nil
			}
		}
		return nil, fmt.Errorf("expected namespace-node(), got %s", describeItem(item))
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

	// Handle function(...) — function type matching and coercion
	if strings.HasPrefix(itemType, "function(") {
		// function(*) matches any function item
		if itemType == "function(*)" {
			switch item.(type) {
			case xpath3.FunctionItem, xpath3.MapItem, *xpath3.ArrayItem:
				return item, nil
			}
			return nil, fmt.Errorf("expected %s, got %s", itemType, describeItem(item))
		}
		// Parse the function type
		st, err := xpath3.ParseSequenceType(itemType)
		if err != nil {
			return item, nil //nolint:nilerr // unparseable type → pass through
		}
		seq := xpath3.ItemSlice{item}
		// First try strict sequenceType matching (contravariant params, covariant return).
		if xpath3.MatchesSequenceType(seq, st) {
			return item, nil
		}
		// Check parameter compatibility: the function's declared params must
		// accept the target's param types (contravariance). This catches cases
		// like function($x as xs:float) being used where function(xs:double)
		// is expected — xs:float cannot accept xs:double args.
		if fi, ok := item.(xpath3.FunctionItem); ok {
			if ok := xpath3.CheckFunctionParamCompat(fi, st); !ok {
				return nil, fmt.Errorf("function item does not match required type %s", itemType)
			}
		}
		// Try function coercion for return type flexibility — creates a
		// wrapper that coerces args/return at call time.
		coerced, ok := xpath3.CoerceToSequenceType(seq, st)
		if ok && coerced != nil && sequence.Len(coerced) == 1 {
			return coerced.Get(0), nil
		}
		return nil, fmt.Errorf("function item does not match required type %s", itemType)
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
		if reqTypeName != "" && reqTypeName != "*" && ec != nil && ec.schemaRegistry != nil {
			ann := ""
			if ec.typeAnnotations != nil {
				ann = ec.typeAnnotations[ni.Node]
			}
			reqTypeNorm := normalizeTypeName(reqTypeName, ec)
			// Untyped nodes (source documents not validated against schema): check if the
			// schema declares this element with a compatible type and accept it.
			if ann == "" || ann == "xs:untyped" || ann == "Q{}untyped" {
				elem2, isElem2 := ni.Node.(*helium.Element)
				if isElem2 {
					elemLocal2, elemNS2 := elem2.LocalName(), elem2.URI()
					declType, found := ec.schemaRegistry.LookupSchemaElement(elemLocal2, elemNS2)
					if found && (declType == reqTypeNorm || ec.schemaRegistry.IsSubtypeOf(declType, reqTypeNorm)) {
						return item, nil
					}
				}
				// Fallthrough: treat as type mismatch
				return nil, fmt.Errorf("expected %s (type %s), element has type %s", itemType, reqTypeNorm, ann)
			}
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
		nameMatch := elem.LocalName() == reqLocal && elem.URI() == reqNS
		// schema-element(E) also matches substitution group members of E.
		substMatch := false
		if !nameMatch && ec != nil && ec.schemaRegistry != nil {
			substMatch = ec.schemaRegistry.IsSubstitutionGroupMember(
				elem.LocalName(), elem.URI(), reqLocal, reqNS)
		}
		if !nameMatch && !substMatch {
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
				// Untyped elements (not validated) do not match schema-element().
				if ann == "" || ann == "xs:untyped" || ann == "Q{}untyped" {
					return nil, fmt.Errorf("expected %s, element is untyped (not validated)", itemType)
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
				// Untyped attributes (source documents not validated against schema) match
				// schema-attribute(Q) when name + declaration exist.
				if ann == "" || ann == "xs:untypedAtomic" || ann == "Q{}untypedAtomic" {
					return item, nil
				}
				if !ec.schemaRegistry.IsSubtypeOf(ann, declType) {
					return nil, fmt.Errorf("expected %s (type %s), attribute has type %s", itemType, declType, ann)
				}
			}
		}
		return item, nil
	}

	// Atomic type — need to atomize and potentially cast
	return coerceToAtomicType(item, itemType, ec)
}

// coerceToAtomicType atomizes a node/value and casts to the target atomic type.
func coerceToAtomicType(item xpath3.Item, targetType string, ec *execContext) (xpath3.Item, error) {
	// If already an atomic value, check/cast the type
	if av, ok := item.(xpath3.AtomicValue); ok {
		return castAtomicToType(av, targetType, ec)
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

	return castAtomicToType(av, targetType, ec)
}

// castAtomicToType casts an atomic value to the specified target type.
func castAtomicToType(av xpath3.AtomicValue, targetType string, ec ...*execContext) (xpath3.Item, error) {
	// Normalize target type
	target := normalizeTypeName(targetType, ec...)

	// If already the right type, return as-is
	if av.TypeName == target {
		return av, nil
	}

	// Schema-derived subtypes satisfy a wider target type without recasting.
	if len(ec) > 0 && ec[0] != nil && ec[0].schemaRegistry != nil {
		if ec[0].schemaRegistry.IsSubtypeOf(av.TypeName, target) {
			return av, nil
		}
	}

	// Built-in XSD subtype relationships: xs:integer derives from xs:decimal.
	// An integer value satisfies xs:decimal without promotion/casting.
	if isBuiltinSubtypeOf(av.TypeName, target) {
		return av, nil
	}

	// xs:anyAtomicType matches any atomic value
	if target == "xs:anyAtomicType" {
		return av, nil
	}

	// xs:anyURI -> xs:string promotion (per XPath spec).
	// Also handles schema-defined types derived from xs:anyURI.
	if target == xpath3.TypeString && isAnyURIType(av.TypeName, ec...) {
		return xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: av.StringVal()}, nil
	}

	// xs:string -> xs:anyURI promotion (per XPath spec)
	if av.TypeName == xpath3.TypeString && target == xpath3.TypeAnyURI {
		return xpath3.AtomicValue{TypeName: xpath3.TypeAnyURI, Value: av.StringVal()}, nil
	}

	// xs:untypedAtomic can be cast to any atomic type (function conversion rules).
	// xs:string is NOT implicitly castable to other types — only xs:untypedAtomic
	// has this privilege. xs:string -> xs:integer (etc.) is a type error (XTTE0590).
	if av.TypeName == xpath3.TypeUntypedAtomic {
		s := av.StringVal()
		cast, err := xpath3.CastFromString(s, target)
		if err != nil {
			// Try schema-aware cast for user-defined types.
			if schemaCast, ok := trySchemaCast(s, target, ec...); ok {
				return schemaCast, nil
			}
			return nil, fmt.Errorf("cannot cast %q to %s: %w", s, targetType, err)
		}
		return cast, nil
	}

	// Numeric promotion: a value whose numeric type is narrower than
	// the target is promoted (cast) to the target type. Demotion
	// (wider→narrower, e.g., double→float) is rejected (XTTE0570).
	// This also handles schema-defined types whose builtin base is numeric.
	srcBase, srcIsNum := resolveToNumericBase(av.TypeName, ec...)
	tgtBase, tgtIsNum := resolveToNumericBase(target, ec...)
	if srcIsNum && tgtIsNum {
		srcRank := numericRank(srcBase)
		tgtRank := numericRank(tgtBase)
		if srcRank > tgtRank {
			return nil, fmt.Errorf("cannot convert %s to %s", av.TypeName, targetType)
		}
		if srcRank < tgtRank {
			// Promote: cast to wider type so instance-of checks work.
			s, _ := xpath3.AtomicToString(av)
			cast, err := xpath3.CastFromString(s, target)
			if err != nil {
				return nil, fmt.Errorf("cannot promote %s to %s: %w", av.TypeName, target, err)
			}
			return cast, nil
		}
		// Same rank: accept as-is
		return av, nil
	}

	// User-defined types (union or restriction): check if the value's type
	// is a subtype of the target type using the schema registry.
	if len(ec) > 0 && ec[0] != nil && ec[0].schemaRegistry != nil {
		if ec[0].schemaRegistry.IsSubtypeOf(av.TypeName, target) {
			return av, nil
		}
		// For union types, check if the value's type is a member type.
		td, _, found := ec[0].schemaRegistry.LookupTypeDef(target)
		if found && td != nil && td.Variety == xsd.TypeVarietyUnion {
			for _, member := range td.MemberTypes {
				memberName := xsdTypeNameFromDef(member)
				if av.TypeName == memberName || ec[0].schemaRegistry.IsSubtypeOf(av.TypeName, memberName) {
					return av, nil
				}
			}
		}
	}

	// No implicit cast path found — type mismatch.
	return nil, fmt.Errorf("cannot convert %s to %s", av.TypeName, targetType)
}

// trySchemaCast attempts to cast a string to a user-defined schema type by
// resolving to the built-in base type, casting to that, validating facets,
// and returning the result with the user-defined type name.
func trySchemaCast(s, target string, ec ...*execContext) (xpath3.Item, bool) {
	if len(ec) == 0 || ec[0] == nil || ec[0].schemaRegistry == nil {
		return nil, false
	}
	reg := ec[0].schemaRegistry
	// Resolve to the built-in base type by walking the schema type chain.
	local, ns := splitAnnotationName(target)
	builtinBase := resolveToBuiltin(local, ns, reg)
	if builtinBase != "" {
		cast, err := xpath3.CastFromString(s, builtinBase)
		if err != nil {
			return nil, false
		}
		// Validate facets.
		castStr, _ := xpath3.AtomicToString(cast)
		if facetErr := reg.ValidateCast(castStr, target); facetErr != nil {
			return nil, false
		}
		cast.TypeName = target
		return cast, true
	}
	// For union types, try each member type until one succeeds.
	td, _, found := reg.LookupTypeDef(target)
	if found && td != nil && td.Variety == xsd.TypeVarietyUnion {
		for _, member := range td.MemberTypes {
			memberName := xsdTypeNameFromDef(member)
			cast, err := xpath3.CastFromString(s, memberName)
			if err == nil {
				// Keep the member's type name (not the union name) so that
				// instance-of checks like "instance of xs:time" work.
				return cast, true
			}
		}
	}
	return nil, false
}

// resolveToBuiltin walks the schema type chain from a user-defined type to
// find the ultimate built-in XSD base type.
func resolveToBuiltin(local, ns string, reg *schemaRegistry) string {
	current, currentNS := local, ns
	for i := 0; i < 32; i++ {
		baseType, ok := reg.LookupType(current, currentNS)
		if !ok {
			return ""
		}
		if xpath3.IsKnownXSDType(baseType) {
			return baseType
		}
		newLocal, newNS := splitAnnotationName(baseType)
		current, currentNS = newLocal, newNS
	}
	return ""
}

// normalizeTypeName normalizes a type name. xs: and xsd: prefixes are
// canonicalized to xs:. Other prefixed names are resolved to Q{ns}local
// format using the exec context's namespace bindings (if available).
func normalizeTypeName(name string, ec ...*execContext) string {
	if strings.HasPrefix(name, "xs:") {
		return name
	}
	if strings.HasPrefix(name, "Q{") {
		return name
	}
	// Normalize xsd: prefix to xs: (both map to XML Schema namespace)
	if strings.HasPrefix(name, "xsd:") {
		return "xs:" + name[4:]
	}
	// Resolve other prefix:local names to Q{ns}local using namespace bindings.
	if idx := strings.IndexByte(name, ':'); idx > 0 {
		prefix := name[:idx]
		local := name[idx+1:]
		if len(ec) > 0 && ec[0] != nil {
			if ns, ok := ec[0].stylesheet.namespaces[prefix]; ok {
				if ns == lexicon.NamespaceXSD {
					return "xs:" + local
				}
				return xpath3.QAnnotation(ns, local)
			}
		}
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
	// Unprefixed non-builtin type names: use Q{} annotation form to match
	// xsdTypeNameFromDef which produces Q{}local for no-namespace types.
	if !strings.ContainsAny(name, ":{") {
		return "Q{}" + name
	}
	return name
}

// numericRank returns the promotion rank of a numeric type.
// Higher rank = wider type. Promotion is only valid from lower to higher rank.
// Derived integer types (xs:int, xs:long, etc.) have the same rank as xs:integer.
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
	// Derived integer types all promote at the same rank as xs:integer.
	if xpath3.BuiltinIsSubtypeOf(t, xpath3.TypeInteger) {
		return 1
	}
	// Derived decimal types (not integer) promote at decimal rank.
	if xpath3.BuiltinIsSubtypeOf(t, xpath3.TypeDecimal) {
		return 2
	}
	return 0
}

// isNumericType returns true for xs:integer, xs:decimal, xs:float, xs:double
// and all built-in derived integer/decimal types (xs:int, xs:long, etc.).
func isNumericType(t string) bool {
	switch t {
	case xpath3.TypeInteger, xpath3.TypeDecimal, xpath3.TypeFloat, xpath3.TypeDouble:
		return true
	}
	return xpath3.BuiltinIsSubtypeOf(t, xpath3.TypeDecimal)
}

// resolveToNumericBase resolves a schema-defined type to its builtin numeric
// base type, if any. Returns the builtin type name and true, or ("", false).
func resolveToNumericBase(typeName string, ec ...*execContext) (string, bool) {
	if isNumericType(typeName) {
		return typeName, true
	}
	if len(ec) == 0 || ec[0] == nil || ec[0].schemaRegistry == nil {
		return "", false
	}
	local, ns := splitAnnotationName(typeName)
	base := resolveToBuiltin(local, ns, ec[0].schemaRegistry)
	if base != "" && isNumericType(base) {
		return base, true
	}
	return "", false
}

// isAnyURIType returns true if the type is xs:anyURI or a schema-defined type
// that derives from xs:anyURI.
func isAnyURIType(typeName string, ec ...*execContext) bool {
	if typeName == xpath3.TypeAnyURI {
		return true
	}
	if len(ec) == 0 || ec[0] == nil || ec[0].schemaRegistry == nil {
		return false
	}
	local, ns := splitAnnotationName(typeName)
	base := resolveToBuiltin(local, ns, ec[0].schemaRegistry)
	return base == xpath3.TypeAnyURI
}

// resolveSchemaQName resolves a QName string (e.g. "my:userNode" or "localName")
// to (localName, namespace) using the stylesheet's namespace bindings.
func resolveSchemaQName(qname string, ec *execContext) (local, ns string) {
	idx := strings.IndexByte(qname, ':')
	if idx < 0 {
		// Unprefixed: use xpath-default-namespace if set.
		if ec != nil && ec.hasXPathDefaultNS {
			return qname, ec.xpathDefaultNS
		}
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

func parseMapItemType(itemType string) (string, sequenceType, bool, error) {
	inner := strings.TrimSpace(itemType[len("map(") : len(itemType)-1])
	if inner == "*" {
		return "", sequenceType{}, false, nil
	}

	parts := splitTopLevelTypeArgs(inner)
	if len(parts) != 2 {
		return "", sequenceType{}, false, fmt.Errorf("invalid map type %q", itemType)
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

func sequenceMatchesTypeStrict(seq xpath3.Sequence, st sequenceType) bool {
	count := 0
	if seq != nil {
		count = sequence.Len(seq)
	}
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

	for item := range sequence.Items(seq) {
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

func formatSequenceType(st sequenceType) string {
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
	errCodeXTTE0790 = "XTTE0790" // function parameter type mismatch
)
