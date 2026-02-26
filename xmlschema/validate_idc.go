package xmlschema

import (
	"fmt"
	"strings"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath"
)

// idcTable holds key-sequences collected during IDC evaluation for a single constraint.
type idcTable struct {
	idc  *IDConstraint
	keys []idcEntry
}

// idcEntry holds a single key-sequence value and the node that produced it.
type idcEntry struct {
	values []string       // one value per field
	node   helium.Node    // the node selected by the selector
	elem   *helium.Element // the element (for line number reporting)
}

// validateIDConstraints evaluates identity constraints declared on the given element.
// It collects key-sequences from selector/field XPath expressions and checks
// uniqueness (unique/key) and referential integrity (keyref).
func validateIDConstraints(elem *helium.Element, edecl *ElementDecl, schema *Schema, filename string, out *strings.Builder) error {
	if len(edecl.IDCs) == 0 {
		return nil
	}

	// Phase 1: Evaluate all constraints and collect key-sequence tables.
	tables := make(map[string]*idcTable, len(edecl.IDCs))
	var lastErr error

	for _, idc := range edecl.IDCs {
		// Use the schema's namespace context for XPath evaluation.
		nsCtx := &xpath.Context{Namespaces: idc.Namespaces}
		table, err := evaluateIDC(elem, idc, nsCtx)
		if err != nil {
			continue
		}
		tables[idc.Name] = table

		// Check unique/key constraints immediately.
		if idc.Kind == IDCUnique || idc.Kind == IDCKey {
			if err := checkUniqueness(table, idc, schema, filename, out); err != nil {
				lastErr = err
			}
		}
	}

	// Phase 2: Check keyref constraints against referenced key/unique tables.
	for _, idc := range edecl.IDCs {
		if idc.Kind != IDCKeyRef {
			continue
		}
		table := tables[idc.Name]
		if table == nil {
			continue
		}

		// Find the referenced key/unique constraint.
		referName := idc.Refer
		// Handle qualified refer names (prefix:local).
		if idx := strings.IndexByte(referName, ':'); idx >= 0 {
			referName = referName[idx+1:]
		}
		refTable := tables[referName]
		if refTable == nil {
			continue
		}

		if err := checkKeyRef(table, refTable, idc, schema, filename, out); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// evaluateIDC evaluates the selector and field XPaths for a single IDC.
func evaluateIDC(elem *helium.Element, idc *IDConstraint, nsCtx *xpath.Context) (*idcTable, error) {
	// Evaluate selector XPath.
	selectorResult, err := xpath.EvaluateWithContext(elem, idc.Selector, nsCtx)
	if err != nil {
		return nil, err
	}
	if selectorResult.Type != xpath.NodeSetResult {
		return nil, fmt.Errorf("selector did not return a node-set")
	}

	table := &idcTable{idc: idc}

	for _, node := range selectorResult.NodeSet {
		entry := idcEntry{node: node}

		// Resolve the element for this node.
		if e, ok := node.(*helium.Element); ok {
			entry.elem = e
		}

		// Evaluate each field XPath relative to the selected node.
		allPresent := true
		for _, fieldXPath := range idc.Fields {
			fieldResult, err := xpath.EvaluateWithContext(node, fieldXPath, nsCtx)
			if err != nil {
				allPresent = false
				break
			}

			var value string
			switch fieldResult.Type {
			case xpath.NodeSetResult:
				if len(fieldResult.NodeSet) == 0 {
					allPresent = false
				} else {
					value = nodeStringValue(fieldResult.NodeSet[0])
				}
			case xpath.StringResult:
				value = fieldResult.String
			default:
				value = fmt.Sprintf("%v", fieldResult.Number)
			}
			entry.values = append(entry.values, value)
		}

		if allPresent {
			table.keys = append(table.keys, entry)
		}
	}

	return table, nil
}

// nodeStringValue returns the string value of a node (text content for elements, value for attributes).
func nodeStringValue(n helium.Node) string {
	switch v := n.(type) {
	case *helium.Element:
		return elemTextContent(v)
	case *helium.Attribute:
		return v.Value()
	default:
		return string(n.Content())
	}
}

// checkUniqueness checks that all key-sequences in the table are unique.
func checkUniqueness(table *idcTable, idc *IDConstraint, schema *Schema, filename string, out *strings.Builder) error {
	seen := make(map[string]bool)
	var lastErr error

	for _, entry := range table.keys {
		key := formatKeySequence(entry.values)
		if seen[key] {
			elemName := entryDisplayName(entry)
			idcName := idcDisplayName(idc, schema)
			msg := fmt.Sprintf("Duplicate key-sequence %s in unique identity-constraint '%s'.",
				formatKeyDisplay(entry.values), idcName)
			if entry.elem != nil {
				out.WriteString(validityError(filename, entry.elem.Line(), elemName, msg))
			}
			lastErr = fmt.Errorf("duplicate key-sequence")
		}
		seen[key] = true
	}

	return lastErr
}

// checkKeyRef checks that every key-sequence in the keyref table has a match in the referenced table.
func checkKeyRef(keyrefTable, refTable *idcTable, idc *IDConstraint, schema *Schema, filename string, out *strings.Builder) error {
	// Build set of referenced key-sequences.
	refKeys := make(map[string]bool, len(refTable.keys))
	for _, entry := range refTable.keys {
		refKeys[formatKeySequence(entry.values)] = true
	}

	var lastErr error
	for _, entry := range keyrefTable.keys {
		key := formatKeySequence(entry.values)
		if !refKeys[key] {
			elemName := entryDisplayName(entry)
			idcName := idcDisplayName(idc, schema)
			msg := fmt.Sprintf("No match found for key-sequence %s of keyref '%s'.",
				formatKeyDisplay(entry.values), idcName)
			if entry.elem != nil {
				out.WriteString(validityError(filename, entry.elem.Line(), elemName, msg))
			}
			lastErr = fmt.Errorf("keyref not found")
		}
	}

	return lastErr
}

// formatKeySequence creates a unique string key from a sequence of values (for map lookups).
func formatKeySequence(values []string) string {
	return strings.Join(values, "\x00")
}

// formatKeyDisplay formats key values for display in error messages: ['v1', 'v2'].
func formatKeyDisplay(values []string) string {
	parts := make([]string, len(values))
	for i, v := range values {
		parts[i] = "'" + v + "'"
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// entryDisplayName returns the display name of the element for an IDC entry.
func entryDisplayName(entry idcEntry) string {
	if entry.elem != nil {
		return elemDisplayName(entry.elem)
	}
	return ""
}

// idcDisplayName returns the namespace-qualified display name of an IDC.
func idcDisplayName(idc *IDConstraint, schema *Schema) string {
	if schema.targetNamespace != "" {
		return "{" + schema.targetNamespace + "}" + idc.Name
	}
	return idc.Name
}
