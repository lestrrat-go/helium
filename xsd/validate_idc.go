package xsd

import (
	"context"
	"fmt"
	"strings"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath1"
)

// idcTable holds key-sequences collected during IDC evaluation for a single constraint.
type idcTable struct {
	idc  *IDConstraint
	keys []idcEntry
}

// idcEntry holds a single key-sequence value and the node that produced it.
type idcEntry struct {
	values []string        // one value per field
	node   helium.Node     // the node selected by the selector
	elem   *helium.Element // the element (for line number reporting)
}

// validateIDConstraints evaluates identity constraints declared on the given element.
// It collects key-sequences from selector/field XPath expressions and checks
// uniqueness (unique/key) and referential integrity (keyref).
func (vc *validationContext) validateIDConstraints(ctx context.Context, elem *helium.Element, edecl *ElementDecl) error {
	if len(edecl.IDCs) == 0 {
		return nil
	}

	// Phase 1: Evaluate all constraints and collect key-sequence tables.
	tables := make(map[string]*idcTable, len(edecl.IDCs))
	var lastErr error

	for _, idc := range edecl.IDCs {
		// Use the schema's namespace context for XPath evaluation.
		ev := xpath1.NewEvaluator().AdditionalNamespaces(idc.Namespaces)
		table, err := evaluateIDC(ctx, ev, elem, idc)
		if err != nil {
			continue
		}
		tables[idc.Name] = table

		// Check unique/key constraints immediately.
		if idc.Kind == IDCUnique || idc.Kind == IDCKey {
			if err := vc.checkUniqueness(ctx, table, idc); err != nil {
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

		if err := vc.checkKeyRef(ctx, table, refTable, idc); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// evaluateIDC evaluates the selector and field XPaths for a single IDC.
func evaluateIDC(ctx context.Context, ev xpath1.Evaluator, elem *helium.Element, idc *IDConstraint) (*idcTable, error) {
	// Evaluate selector XPath using pre-compiled expression when available.
	var selectorResult *xpath1.Result
	var err error
	if idc.SelectorExpr != nil {
		selectorResult, err = ev.Evaluate(ctx, idc.SelectorExpr, elem)
	} else {
		compiled, compErr := xpath1.Compile(idc.Selector)
		if compErr != nil {
			return nil, fmt.Errorf("xsd: IDC selector XPath failed: %w", compErr)
		}
		selectorResult, err = ev.Evaluate(ctx, compiled, elem)
	}
	if err != nil {
		return nil, fmt.Errorf("xsd: IDC selector XPath failed: %w", err)
	}
	if selectorResult.Type != xpath1.NodeSetResult {
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
		for i, fieldXPath := range idc.Fields {
			var fieldResult *xpath1.Result
			if i < len(idc.FieldExprs) && idc.FieldExprs[i] != nil {
				fieldResult, err = ev.Evaluate(ctx, idc.FieldExprs[i], node)
			} else {
				compiled, compErr := xpath1.Compile(fieldXPath)
				if compErr != nil {
					allPresent = false
					break
				}
				fieldResult, err = ev.Evaluate(ctx, compiled, node)
			}
			if err != nil {
				allPresent = false
				break
			}

			var value string
			switch fieldResult.Type {
			case xpath1.NodeSetResult:
				if len(fieldResult.NodeSet) == 0 {
					allPresent = false
				} else {
					value = nodeStringValue(fieldResult.NodeSet[0])
				}
			case xpath1.StringResult:
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
func (vc *validationContext) checkUniqueness(ctx context.Context, table *idcTable, idc *IDConstraint) error {
	seen := make(map[string]bool)
	var lastErr error

	for _, entry := range table.keys {
		key := formatKeySequence(entry.values)
		if seen[key] {
			elemName := entryDisplayName(entry)
			idcName := idcDisplayName(idc, vc.schema)
			msg := fmt.Sprintf("Duplicate key-sequence %s in unique identity-constraint '%s'.",
				formatKeyDisplay(entry.values), idcName)
			if entry.elem != nil {
				vc.reportValidityError(ctx, vc.filename, entry.elem.Line(), elemName, msg)
			}
			lastErr = fmt.Errorf("duplicate key-sequence")
		}
		seen[key] = true
	}

	return lastErr
}

// checkKeyRef checks that every key-sequence in the keyref table has a match in the referenced table.
func (vc *validationContext) checkKeyRef(ctx context.Context, keyrefTable, refTable *idcTable, idc *IDConstraint) error {
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
			idcName := idcDisplayName(idc, vc.schema)
			msg := fmt.Sprintf("No match found for key-sequence %s of keyref '%s'.",
				formatKeyDisplay(entry.values), idcName)
			if entry.elem != nil {
				vc.reportValidityError(ctx, vc.filename, entry.elem.Line(), elemName, msg)
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
		return helium.ClarkName(schema.targetNamespace, idc.Name)
	}
	return idc.Name
}
