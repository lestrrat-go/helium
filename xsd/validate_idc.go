package xsd

import (
	"context"
	"fmt"
	"slices"
	"strings"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/xsd/value"
	"github.com/lestrrat-go/helium/xpath1"
)

// idcTable holds key-sequences collected during IDC evaluation for a single constraint.
type idcTable struct {
	idc        *IDConstraint
	keys       []idcEntry
	keyMissing bool // an xs:key selected node had an absent field (cvc-identity-constraint.4.2.1)
	fieldError bool // a field XPath selected more than one node (cvc-identity-constraint.3)
}

// idcEntry holds a single key-sequence value and the node that produced it.
type idcEntry struct {
	values []string        // raw field values (for human-readable error display)
	canon  []string        // value-space canonical field values (for key comparison)
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
		table, err := vc.evaluateIDC(ctx, ev, elem, edecl, idc)
		if err != nil {
			continue
		}
		tables[idc.Name] = table

		// A key with an absent field was already reported during
		// evaluation; surface it as a validation failure.
		if table.keyMissing {
			lastErr = fmt.Errorf("missing key field")
		}

		// A field that selected more than one node was already reported during
		// evaluation; surface it as a validation failure.
		if table.fieldError {
			lastErr = fmt.Errorf("field evaluates to more than one node")
		}

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
func (vc *validationContext) evaluateIDC(ctx context.Context, ev xpath1.Evaluator, elem *helium.Element, edecl *ElementDecl, idc *IDConstraint) (*idcTable, error) {
	schema := vc.schema
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
		// fieldErr marks that a field already produced a definitive validity
		// error (e.g. a node-set with more than one member) for this entry, so
		// the absent-field handling below does not also fire for the same node.
		allPresent := true
		fieldErr := false
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
			var fieldNode helium.Node
			switch fieldResult.Type {
			case xpath1.NodeSetResult:
				if len(fieldResult.NodeSet) > 1 {
					// cvc-identity-constraint.3: with the selected node as the
					// context node, each field must evaluate to either an empty
					// node-set or a node-set with exactly one member. More than
					// one selected node is a validity error for every IDC kind.
					if entry.elem != nil {
						idcName := idcDisplayName(idc, vc.schema)
						msg := fmt.Sprintf("The XPath '%s' of a field of %s identity-constraint '%s' evaluates to a node-set with more than one member.",
							fieldXPath, idcKindName(idc.Kind), idcName)
						vc.reportValidityError(ctx, vc.filename, entry.elem.Line(), elemDisplayName(entry.elem), msg)
					}
					table.fieldError = true
					allPresent = false
					fieldErr = true
					break
				}
				if len(fieldResult.NodeSet) == 0 {
					allPresent = false
				} else {
					fieldNode = fieldResult.NodeSet[0]
					value = nodeStringValue(fieldNode)
				}
			case xpath1.StringResult:
				value = fieldResult.String
			default:
				value = fmt.Sprintf("%v", fieldResult.Number)
			}
			entry.values = append(entry.values, value)

			// Canonicalize to the value space using the field's declared
			// simple type, so lexically-distinct but value-equal keys (e.g.
			// "5" and "+5" for xs:integer) compare equal. Falls back to the
			// raw value when the type cannot be resolved.
			builtinLocal := resolveFieldBuiltinLocal(fieldNode, elem, edecl, schema)
			entry.canon = append(entry.canon, canonicalKey(value, builtinLocal))
		}

		if allPresent {
			table.keys = append(table.keys, entry)
			continue
		}

		// A field that resolved to no node leaves the key-sequence
		// incomplete. Per cvc-identity-constraint 4.2.1, every field of an
		// xs:key must evaluate to a node for each selected node, so an
		// absent field is a validity error. xs:unique and xs:keyref
		// tolerate absent fields (the node simply drops out of the
		// qualified node-set), so they only skip the entry.
		if fieldErr {
			continue
		}
		if idc.Kind == IDCKey && entry.elem != nil {
			table.keyMissing = true
			idcName := idcDisplayName(idc, vc.schema)
			msg := fmt.Sprintf("Not all fields of key identity-constraint '%s' evaluate to a node.", idcName)
			vc.reportValidityError(ctx, vc.filename, entry.elem.Line(), elemDisplayName(entry.elem), msg)
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
	seen := make(map[string]struct{})
	var lastErr error

	for _, entry := range table.keys {
		key := formatKeySequence(entry.canon)
		if _, dup := seen[key]; dup {
			elemName := entryDisplayName(entry)
			idcName := idcDisplayName(idc, vc.schema)
			msg := fmt.Sprintf("Duplicate key-sequence %s in unique identity-constraint '%s'.",
				formatKeyDisplay(entry.values), idcName)
			if entry.elem != nil {
				vc.reportValidityError(ctx, vc.filename, entry.elem.Line(), elemName, msg)
			}
			lastErr = fmt.Errorf("duplicate key-sequence")
		}
		seen[key] = struct{}{}
	}

	return lastErr
}

// checkKeyRef checks that every key-sequence in the keyref table has a match in the referenced table.
func (vc *validationContext) checkKeyRef(ctx context.Context, keyrefTable, refTable *idcTable, idc *IDConstraint) error {
	// Build set of referenced key-sequences (value-space canonical).
	refKeys := make(map[string]struct{}, len(refTable.keys))
	for _, entry := range refTable.keys {
		refKeys[formatKeySequence(entry.canon)] = struct{}{}
	}

	var lastErr error
	for _, entry := range keyrefTable.keys {
		key := formatKeySequence(entry.canon)
		if _, ok := refKeys[key]; !ok {
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

// canonicalKey maps a raw field value to a value-space canonical key for the
// given builtin type. An empty builtinLocal (unresolved type) falls back to the
// raw value, preserving the previous lexical-only behavior for that field.
func canonicalKey(raw, builtinLocal string) string {
	if builtinLocal == "" {
		return raw
	}
	key, _ := value.CanonicalKey(raw, builtinLocal)
	return key
}

// resolveFieldBuiltinLocal resolves the builtin XSD base local name of the
// simple type declared for an IDC field node. host/hostDecl are the element the
// constraint is declared on, used to descend the schema content model down to
// the field node. Returns "" when the type cannot be determined, in which case
// the caller falls back to raw-string comparison for that field.
func resolveFieldBuiltinLocal(n helium.Node, host *helium.Element, hostDecl *ElementDecl, schema *Schema) string {
	switch v := n.(type) {
	case *helium.Element:
		td := resolveElemType(v, host, hostDecl, schema)
		if td == nil {
			return ""
		}
		return builtinBaseLocal(td)
	case *helium.Attribute:
		return resolveAttrBuiltinLocal(v, host, hostDecl, schema)
	default:
		return ""
	}
}

// resolveAttrBuiltinLocal resolves the builtin base local of an attribute's
// declared type, preferring the owning element's complex-type attribute uses
// and falling back to a matching global attribute declaration.
func resolveAttrBuiltinLocal(attr *helium.Attribute, host *helium.Element, hostDecl *ElementDecl, schema *Schema) string {
	aqn := QName{Local: attr.LocalName(), NS: attr.URI()}

	if owner, ok := attr.Parent().(*helium.Element); ok {
		if td := resolveElemType(owner, host, hostDecl, schema); td != nil {
			if at := attrUseType(td, aqn, schema); at != nil {
				return builtinBaseLocal(at)
			}
		}
	}

	if ga, ok := schema.globalAttrs[aqn]; ok {
		if td, ok := schema.types[ga.TypeName]; ok {
			return builtinBaseLocal(td)
		}
	}
	return ""
}

// resolveElemType resolves the schema type of an instance element by descending
// the host element's content model along the element's ancestor chain. The
// element must be a descendant of host (true for IDC selector/field results).
// Falls back to a global element declaration lookup.
func resolveElemType(target, host *helium.Element, hostDecl *ElementDecl, schema *Schema) *TypeDef {
	if target == host {
		if hostDecl != nil {
			return hostDecl.Type
		}
		return nil
	}

	// Build the ancestor chain from host's child down to target, tracking
	// whether the walk actually reaches host. If it doesn't (target is not in
	// host's subtree as far as the element ancestry shows), descending host's
	// content model would match unrelated names and yield a wrong type, so fall
	// back to the global element declaration instead.
	var chain []*helium.Element
	reached := false
	for cur := target; cur != nil; {
		if cur == host {
			reached = true
			break
		}
		chain = append(chain, cur)
		parent, ok := cur.Parent().(*helium.Element)
		if !ok {
			break
		}
		cur = parent
	}
	if !reached {
		return resolveElemTypeFallback(target, schema)
	}

	td := hostType(host, hostDecl, schema)
	// Descend from host's type through each level (outermost ancestor last in chain).
	for _, node := range slices.Backward(chain) {
		if td == nil {
			break
		}
		qn := QName{Local: node.LocalName(), NS: node.URI()}
		decl := childElemDecl(td, qn, schema)
		if decl == nil {
			return resolveElemTypeFallback(target, schema)
		}
		td = decl.Type
	}
	if td != nil {
		return td
	}
	return resolveElemTypeFallback(target, schema)
}

func resolveElemTypeFallback(target *helium.Element, schema *Schema) *TypeDef {
	decl := lookupElemDecl(target, schema)
	if decl == nil {
		return nil
	}
	return decl.Type
}

// hostType returns the type of the host element, preferring its declaration.
func hostType(host *helium.Element, hostDecl *ElementDecl, schema *Schema) *TypeDef {
	if hostDecl != nil && hostDecl.Type != nil {
		return hostDecl.Type
	}
	decl := lookupElemDecl(host, schema)
	if decl == nil {
		return nil
	}
	return decl.Type
}

// childElemDecl finds a child element declaration matching qn within a type's
// content model (walking the base-type chain), resolving substitution-group
// members through global declarations as a fallback.
func childElemDecl(td *TypeDef, qn QName, schema *Schema) *ElementDecl {
	for cur := td; cur != nil; cur = cur.BaseType {
		if decl := findElemDeclInGroup(cur.ContentModel, qn); decl != nil {
			return decl
		}
	}
	if decl, ok := schema.LookupElement(qn.Local, qn.NS); ok {
		return decl
	}
	return nil
}

// findElemDeclInGroup searches a model group recursively for an element
// declaration matching qn.
func findElemDeclInGroup(mg *ModelGroup, qn QName) *ElementDecl {
	if mg == nil {
		return nil
	}
	for _, p := range mg.Particles {
		switch term := p.Term.(type) {
		case *ElementDecl:
			if term.Name == qn {
				return term
			}
		case *ModelGroup:
			if decl := findElemDeclInGroup(term, qn); decl != nil {
				return decl
			}
		}
	}
	return nil
}

// attrUseType walks a complex type's base chain to find the declared type of an
// attribute use matching the given QName.
func attrUseType(td *TypeDef, aqn QName, schema *Schema) *TypeDef {
	for cur := td; cur != nil; cur = cur.BaseType {
		for _, au := range cur.Attributes {
			if au.Name != aqn {
				continue
			}
			at, ok := schema.types[au.TypeName]
			if !ok {
				return nil
			}
			return at
		}
	}
	return nil
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

// idcKindName returns the XSD keyword for an IDC kind ("unique", "key",
// "keyref"), used in validity-error designations.
func idcKindName(kind IDCKind) string {
	switch kind {
	case IDCKey:
		return elemKey
	case IDCKeyRef:
		return elemKeyRef
	default:
		return elemUnique
	}
}

// idcDisplayName returns the namespace-qualified display name of an IDC.
func idcDisplayName(idc *IDConstraint, schema *Schema) string {
	if schema.targetNamespace != "" {
		return helium.ClarkName(schema.targetNamespace, idc.Name)
	}
	return idc.Name
}
