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
// Examples: "item()*", "xs:string", "element()+", "xs:integer?", "text()".
func parseSequenceType(as string) SequenceType {
	s := strings.TrimSpace(as)
	if s == "" {
		return SequenceType{ItemType: "item()", Occurrence: '*'}
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
func checkSequenceType(seq xpath3.Sequence, st SequenceType, errCode string, context string) (xpath3.Sequence, error) {
	// Check cardinality
	count := len(seq)
	switch st.Occurrence {
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
		coerced, err := coerceItem(item, st.ItemType)
		if err != nil {
			return nil, dynamicError(errCode, "%s: %v", context, err)
		}
		result = append(result, coerced)
	}
	return result, nil
}

// coerceItem checks that a single item matches the expected type, applying
// atomization and casting as needed per the XSLT function conversion rules.
func coerceItem(item xpath3.Item, itemType string) (xpath3.Item, error) {
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

	// Handle map(*) — any map
	if strings.HasPrefix(itemType, "map(") {
		if _, ok := item.(xpath3.MapItem); ok {
			return item, nil
		}
		return nil, fmt.Errorf("expected %s, got %s", itemType, describeItem(item))
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

	// Handle element(name) patterns like element(foo), element(ns:foo)
	if strings.HasPrefix(itemType, "element(") {
		if ni, ok := item.(xpath3.NodeItem); ok {
			if ni.Node.Type() == helium.ElementNode {
				// Extract the name from element(name)
				inner := itemType[len("element(") : len(itemType)-1]
				inner = strings.TrimSpace(inner)
				if inner == "" || inner == "*" {
					return item, nil // element() or element(*) matches any element
				}
				// Check element name match
				elemName := ni.Node.Name()
				if elem, ok := ni.Node.(*helium.Element); ok {
					elemName = elem.LocalName()
				}
				// Strip namespace prefix from the required name for matching
				reqName := inner
				if idx := strings.IndexByte(reqName, ','); idx >= 0 {
					reqName = strings.TrimSpace(reqName[:idx]) // element(name, type) — just use name
				}
				if idx := strings.IndexByte(reqName, ':'); idx >= 0 {
					reqName = reqName[idx+1:] // strip prefix
				}
				if elemName == reqName {
					return item, nil
				}
				return nil, fmt.Errorf("expected %s, got element %q", itemType, elemName)
			}
		}
		return nil, fmt.Errorf("expected %s, got %s", itemType, describeItem(item))
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

	// Numeric promotion: only upward — integer→decimal→float→double.
	// Demotion (e.g., double→float) is not allowed (XTTE0570).
	if isNumericType(target) && isNumericType(av.TypeName) {
		if numericRank(av.TypeName) > numericRank(target) {
			return nil, fmt.Errorf("cannot convert %s to %s", av.TypeName, targetType)
		}
		s, err := xpath3.AtomicToString(av)
		if err != nil {
			return nil, fmt.Errorf("cannot convert to %s: %w", targetType, err)
		}
		cast, err := xpath3.CastFromString(s, target)
		if err != nil {
			return nil, fmt.Errorf("cannot cast to %s: %w", targetType, err)
		}
		return cast, nil
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

// XSLT type error codes.
const (
	errCodeXTTE0505 = "XTTE0505" // template return type mismatch
	errCodeXTTE0570 = "XTTE0570" // variable/param type mismatch
	errCodeXTTE0780 = "XTTE0780" // function return type mismatch
)
