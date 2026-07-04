package xsd

import (
	"context"
	"fmt"
	"slices"
	"strings"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/xsd/value"
	"github.com/lestrrat-go/helium/xpath1"
)

// idcTable holds key-sequences collected during IDC evaluation for a single constraint.
type idcTable struct {
	idc        *IDConstraint
	keys       []idcEntry
	keyMissing bool // an xs:key selected node had an absent field (cvc-identity-constraint.4.2.1)
	fieldError bool // a field XPath selected more than one node (cvc-identity-constraint.3)
	fieldType  bool // a field selected a node whose type is not simple (cvc-identity-constraint.3)
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

	// Evaluate all constraints declared on this element and collect their
	// key-sequence tables. The tables are scoped to THIS element OCCURRENCE: an
	// xs:key/xs:unique table built here is visible only to keyrefs declared on the
	// SAME occurrence. This matches the XSD identity-constraint scope (and
	// xmllint): a keyref resolves against the key table built for its own host
	// occurrence, so two sibling occurrences of a repeating host never leak key
	// spaces into each other.
	var lastErr error

	// keyTables holds this occurrence's evaluated key/unique tables, indexed by
	// the constraint's full QName identity, for resolving same-occurrence keyrefs.
	keyTables := make(map[QName]*idcTable)
	// keyRefs collects this occurrence's keyrefs to resolve after every key/unique
	// on the same occurrence has been evaluated (a keyref may be declared before
	// the key it refers to in document order).
	var keyRefs []pendingKeyRef

	for _, idc := range edecl.IDCs {
		// Use the schema's namespace context for XPath evaluation.
		ev := xpath1.NewEvaluator().AdditionalNamespaces(idc.Namespaces)
		table, err := vc.evaluateIDC(ctx, ev, elem, edecl, idc)
		if err != nil {
			// A selector/field XPath that fails to compile or evaluate must NOT be
			// swallowed: doing so silently disables the constraint and lets an
			// otherwise-invalid document validate. Report it as a validity error.
			// (Malformed XPaths are also rejected at schema compile time, so this
			// path normally fires only on a genuine evaluation failure.)
			msg := fmt.Sprintf("Failed to evaluate identity-constraint '%s': %s", idc.Name, err)
			vc.reportValidityError(ctx, vc.filename, elem.Line(), elemDisplayName(elem), msg)
			lastErr = err
			continue
		}

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

		// A field that selected a node with a non-simple type was already
		// reported during evaluation; surface it as a validation failure.
		if table.fieldType {
			lastErr = fmt.Errorf("field evaluates to a node whose type is not simple")
		}

		if idc.Kind == IDCKeyRef {
			// Defer keyref resolution until every key/unique on this occurrence is
			// evaluated, so a keyref declared before its key (document order) still
			// resolves.
			keyRefs = append(keyRefs, pendingKeyRef{idc: idc, table: table})
			continue
		}

		// Register the key/unique key-sequences under the constraint's full QName
		// identity, scoped to THIS occurrence, for same-occurrence keyref
		// resolution.
		keyTables[idc.QName] = table

		// Check unique/key uniqueness immediately.
		if err := vc.checkUniqueness(ctx, table, idc); err != nil {
			lastErr = err
		}
	}

	// Resolve this occurrence's keyrefs against the key/unique tables in scope for
	// this host occurrence. Per XSD identity-constraint scope, a key/unique table
	// declared on a DESCENDANT element propagates UP to the keyref's host node, so
	// the referenced table is gathered from the host occurrence's OWN SUBTREE (the
	// host element and all its descendants), not only the constraints declared
	// directly on the host. This keeps the scope tight: a key on a SIBLING (or on
	// a different occurrence of a repeating host) is OUTSIDE this subtree and so
	// does NOT satisfy the keyref — matching xmllint, which rejects those — while a
	// key on a child element (bug322411) correctly does.
	for _, pending := range keyRefs {
		refTable := keyTables[pending.idc.ReferQName]
		if refTable == nil {
			// Not declared on the host itself; gather it from the host occurrence's
			// descendant subtree (propagated-up tables).
			refTable = vc.collectSubtreeKeyTable(ctx, elem, pending.idc.ReferQName)
		}
		if refTable == nil {
			refTable = &idcTable{idc: pending.idc}
		}
		if err := vc.checkKeyRef(ctx, pending.table, refTable, pending.idc); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// collectSubtreeKeyTable gathers the merged key-sequence table for the
// key/unique constraint identified by referQN, evaluated over every DESCENDANT
// element occurrence of host that declares a constraint with that QName. This
// models XSD identity-constraint table propagation up the tree: a key/unique
// declared on a child element is in scope for a keyref on an ancestor host.
//
// The host element ITSELF is excluded — constraints declared directly on the
// host are already in the caller's keyTables map (and handled before this path
// is reached). Only the value collection matters here; uniqueness for each
// descendant constraint is checked by that descendant's own pass-2 evaluation,
// so this path does NOT re-report uniqueness violations.
//
// Returns nil when no descendant declares a matching constraint (so the keyref
// resolves against an empty space and every key-sequence is a "no match"
// failure), preserving the out-of-scope rejection for sibling/other-occurrence
// keys.
func (vc *validationContext) collectSubtreeKeyTable(ctx context.Context, host *helium.Element, referQN QName) *idcTable {
	var merged *idcTable
	for child := range helium.Children(host) {
		if child.Type() != helium.ElementNode {
			continue
		}
		ce, ok := helium.AsNode[*helium.Element](child)
		if !ok {
			continue
		}
		// Evaluate any matching key/unique declared on this descendant occurrence.
		if decl := vc.idcHostDecl(ce); decl != nil {
			for _, idc := range decl.IDCs {
				if idc.Kind == IDCKeyRef || idc.QName != referQN {
					continue
				}
				ev := xpath1.NewEvaluator().AdditionalNamespaces(idc.Namespaces)
				// Suppress diagnostics during gathering: this descendant constraint is
				// ALSO evaluated by its own pass-2 walk, which is the canonical place
				// any cvc field/key-missing error is reported. Without suppression those
				// would be double-reported. We only need the key-sequence VALUES here.
				vc.suppressDepth++
				table, err := vc.evaluateIDC(ctx, ev, ce, decl, idc)
				vc.suppressDepth--
				if err != nil {
					continue
				}
				if merged == nil {
					merged = &idcTable{idc: idc}
				}
				merged.keys = append(merged.keys, table.keys...)
			}
		}
		// Recurse so a constraint declared deeper in the subtree also propagates up.
		if sub := vc.collectSubtreeKeyTable(ctx, ce, referQN); sub != nil {
			if merged == nil {
				merged = &idcTable{idc: sub.idc}
			}
			merged.keys = append(merged.keys, sub.keys...)
		}
	}
	return merged
}

// idcHostDecl returns the declaration whose identity constraints apply to an
// element instance. It uses the declaration recorded during pass-1 for ANY
// non-ref local match — including one that carries zero IDCs — because a local
// element that merely shadows a same-named global declaration must NOT inherit
// the global's identity constraints. It falls back to the GLOBAL lookup only
// when no declaration was recorded or the recorded one is a ref (a
// `<xs:element ref="g"/>` correctly resolves to global g and its IDCs). This
// mirrors the selection logic in the pass-2 walk so subtree key-table gathering
// sees the same constraints pass-2 evaluates per element.
func (vc *validationContext) idcHostDecl(elem *helium.Element) *ElementDecl {
	decl := vc.actualElemDecl[elem]
	if decl != nil && !decl.IsRef {
		return decl
	}
	return lookupElemDecl(elem, vc.schema)
}

// evaluateIDC evaluates the selector and field XPaths for a single IDC.
func (vc *validationContext) evaluateIDC(ctx context.Context, ev xpath1.Evaluator, elem *helium.Element, edecl *ElementDecl, idc *IDConstraint) (*idcTable, error) {
	schema := vc.schema
	// Evaluate selector XPath using pre-compiled expression when available. The
	// selector's resolved @xpathDefaultNamespace (XSD 1.1) governs unprefixed
	// element name tests; attributes are never affected.
	selEv := ev.DefaultElementNamespace(idc.SelectorDefaultNS)
	var selectorResult *xpath1.Result
	var err error
	if idc.SelectorExpr != nil {
		selectorResult, err = selEv.Evaluate(ctx, idc.SelectorExpr, elem)
	} else {
		compiled, compErr := xpath1.Compile(idc.Selector)
		if compErr != nil {
			return nil, fmt.Errorf("xsd: IDC selector XPath failed: %w", compErr)
		}
		selectorResult, err = selEv.Evaluate(ctx, compiled, elem)
	}
	if err != nil {
		return nil, fmt.Errorf("xsd: IDC selector XPath failed: %w", err)
	}
	if selectorResult.Type != xpath1.NodeSetResult {
		return nil, fmt.Errorf("selector did not return a node-set")
	}

	table := &idcTable{idc: idc}

	for _, node := range selectorResult.NodeSet {
		// XSD 1.1: a selector must not pick an element inside a
		// processContents="skip" wildcard-matched subtree — such content is not
		// schema-assessed, so it carries no PSVI contribution and cannot
		// participate in a key/unique/keyref. (1.0 keeps selecting it: the set is
		// empty outside 1.1, so this is byte-identical there.)
		if _, skip := vc.skipContentNodes[node]; skip {
			continue
		}
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
			fieldEv := ev
			if i < len(idc.FieldDefaultNS) {
				fieldEv = ev.DefaultElementNamespace(idc.FieldDefaultNS[i])
			}
			var fieldResult *xpath1.Result
			if i < len(idc.FieldExprs) && idc.FieldExprs[i] != nil {
				fieldResult, err = fieldEv.Evaluate(ctx, idc.FieldExprs[i], node)
			} else {
				compiled, compErr := xpath1.Compile(fieldXPath)
				if compErr != nil {
					// A field XPath that fails to compile must NOT be swallowed:
					// marking the field absent silently drops the selected node from
					// the constraint, so a unique/keyref could miss a duplicate or a
					// dangling reference. Surface it like the selector path.
					return nil, fmt.Errorf("xsd: IDC field XPath %q failed to compile: %w", fieldXPath, compErr)
				}
				fieldResult, err = fieldEv.Evaluate(ctx, compiled, node)
			}
			if err != nil {
				// A field XPath that compiles but fails at evaluation (e.g. an
				// unknown function) must likewise surface as an error rather than
				// being silently treated as an absent field.
				return nil, fmt.Errorf("xsd: IDC field XPath %q failed to evaluate: %w", fieldXPath, err)
			}

			var value string
			var fieldNode helium.Node
			switch fieldResult.Type {
			case xpath1.NodeSetResult:
				// §3.11.4 / cvc-identity-constraint.3: classify EVERY node in the
				// field result. A SKIPPED node (an unassessed node in XSD 1.1)
				// contributes NO value and drops out of the qualified node-set. A
				// VIOLATION node (an assessed COMPLEX element, or any unassessed node
				// in XSD 1.0, which has no skipped-node relaxation) invalidates the
				// constraint. After dropping the skipped nodes, at most ONE GOVERNED
				// node may remain — its typed value is the field value; MORE THAN ONE
				// governed node is the genuine cardinality error (a node-set with more
				// than one member).
				var governed helium.Node
				var violating helium.Node
				governedCount := 0
				for _, cand := range fieldResult.NodeSet {
					switch vc.classifyFieldNode(cand) {
					case fieldSkipped:
						// contributes no value; drops out of the qualified node-set
					case fieldViolation:
						if violating == nil {
							violating = cand
						}
					default: // fieldGoverned
						governedCount++
						if governed == nil {
							governed = cand
						}
					}
				}
				if violating != nil {
					if entry.elem != nil {
						idcName := idcDisplayName(idc, vc.schema)
						msg := fmt.Sprintf("The XPath '%s' of a field of %s identity-constraint '%s' evaluates to a node whose type is not simple.",
							fieldXPath, idcKindName(idc.Kind), idcName)
						vc.reportValidityError(ctx, vc.filename, entry.elem.Line(), elemDisplayName(entry.elem), msg)
					}
					table.fieldType = true
					allPresent = false
					fieldErr = true
					break
				}
				if governedCount > 1 {
					// cvc-identity-constraint.3: with the selected node as the
					// context node, each field must evaluate to either an empty
					// node-set or a node-set with exactly one member. More than one
					// GOVERNED member is a validity error for every IDC kind.
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
				if governedCount == 0 {
					// The empty node-set / every-node-skipped case: the field is
					// absent (fieldNode stays nil, value stays ""), handled like the
					// old empty-node-set path below.
					allPresent = false
				} else {
					fieldNode = governed
					value = nodeStringValue(fieldNode)
				}
			case xpath1.StringResult:
				value = fieldResult.String
			default:
				value = fmt.Sprintf("%v", fieldResult.Number)
			}
			entry.values = append(entry.values, value)

			// Canonicalize to the value space using the field's resolved simple
			// type (and, for QName/NOTATION/list, the field node's namespace
			// context), so lexically-distinct but value-equal keys (e.g. "5" and
			// "+5" for xs:integer, "p:a"/"q:a" bound to the same URI, or list
			// "5 6"/"+5 06") compare equal. Falls back to the raw value when the
			// type cannot be resolved.
			fieldTD := vc.resolveFieldType(fieldNode, elem, edecl, schema)
			entry.canon = append(entry.canon, canonicalFieldKey(ctx, value, fieldNode, fieldTD))
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

// canonicalFieldKey maps a raw IDC field value to a value-space canonical key
// using the field's resolved *TypeDef and the field node's namespace context.
// A nil typeDef (unresolved type) falls back to the raw value, preserving the
// previous lexical-only behavior for that field.
//
// Unlike a flat builtin-base-local reduction, this honours the full type:
//   - QName/NOTATION fields resolve the lexical prefix against the field node's
//     in-scope namespaces to a {uri, local} key, so "p:a" and "q:a" bound to the
//     same URI compare equal (and to different URIs compare distinct).
//   - list fields canonicalize each whitespace-separated item in the item type's
//     value space, so "5 6" and "+5 06" compare equal for itemType="xs:integer".
//   - union fields resolve the active member (first member the value validates
//     against, per validateUnionValue) and canonicalize in that member's value
//     space, so a value whose active member is xs:string stays lexically distinct.
func canonicalFieldKey(ctx context.Context, raw string, fieldNode helium.Node, typeDef *TypeDef) string {
	if typeDef == nil {
		return raw
	}
	return canonicalValueKey(ctx, raw, fieldNode, typeDef)
}

// canonicalValueKey canonicalizes raw in the value space of td, dispatching on
// the type's variety (atomic / list / union). fieldNode supplies the namespace
// context needed to resolve QName/NOTATION prefixes; it may be nil, in which case
// only the field node's own bindings are unavailable and lexical-only fallback
// applies for QName-family types.
func canonicalValueKey(ctx context.Context, raw string, fieldNode helium.Node, td *TypeDef) string {
	// Dispatch on the RESOLVED variety, walking restriction derivations: a
	// restriction whose base is an inline list/union keeps Variety==Atomic on the
	// derived TypeDef, so switching on td.Variety alone would mis-route it to the
	// atomic path and lose value-space canonicalization. resolveVariety (the same
	// helper validateValue uses) finds the effective variety through the base
	// chain, and resolveItemType / resolveUnionMembers resolve item/member types
	// through that same chain.
	switch resolveVariety(td) {
	case TypeVarietyList:
		item := resolveItemType(td)
		if item == nil {
			return raw
		}
		// Split list items on XSD whitespace only (space, tab, CR, LF), not the
		// wider set strings.Fields uses: an item containing NBSP must stay one
		// token so it canonicalizes (and validates) as the single invalid value it
		// is, consistent with validateListValue.
		fields := value.XSDFields(raw)
		parts := make([]string, len(fields))
		for i, f := range fields {
			parts[i] = canonicalValueKey(ctx, f, fieldNode, item)
		}
		return strings.Join(parts, " ")
	case TypeVarietyUnion:
		// The active member of a union value is the first DIRECT member
		// (declaration order, descending nested unions only when the value
		// validates against the wrapper) the value actually VALIDATES against —
		// full lexical+facet+nested-union validation, mirroring validateUnionValue's
		// ordered active-member resolution — NOT the first member whose lexical
		// space alone accepts it. So a member whose facets reject the value is
		// skipped and the value falls through to the next member; a nested-union
		// member is validated AS-IS (its wrapper facets included), never pre-
		// flattened to a leaf that would drop those facets. Once the active member
		// is chosen, canonicalize the value in THAT member's space by recursing
		// through canonicalValueKey, so a list member canonicalizes item-by-item
		// and a nested-union member resolves its own active member. So
		// memberTypes="xs:string xs:integer" keeps "5" and "+5" distinct (both
		// active member xs:string), "xs:integer xs:string" collapses them, and
		// memberTypes="intList xs:string" (intList = xs:list itemType="xs:integer")
		// collapses "5 6" and "+5 06".
		if m := unionActiveMember(ctx, raw, fieldNode, td); m != nil {
			return canonicalValueKey(ctx, raw, fieldNode, m)
		}
		return raw
	default:
		return canonicalAtomicKey(raw, fieldNode, td)
	}
}

// canonicalAtomicKey canonicalizes raw for an atomic type td. The value is first
// whitespace-processed per td's effective whiteSpace facet (resolveWhiteSpace —
// the same helper the validator uses), so a restriction of xs:string with
// whiteSpace="collapse" makes "a b" and "a  b" collide. QName/NOTATION then
// resolve the prefix against fieldNode's in-scope namespaces to a Clark-name key;
// everything else delegates to value.CanonicalKey on the builtin base local,
// which returns a value-space canonical form for value-comparable types and the
// whitespace-processed lexical value for lexical-only ones (xs:string family,
// anyURI, …). An unresolvable type or QName falls back to the raw value.
func canonicalAtomicKey(raw string, fieldNode helium.Node, td *TypeDef) string {
	builtinLocal := builtinBaseLocal(td)
	if builtinLocal == "" {
		return raw
	}
	normalized := normalizeWhiteSpace(raw, resolveWhiteSpace(td))
	if builtinLocal == lexicon.TypeQName || builtinLocal == lexicon.TypeNotation {
		ns := fieldNodeNSContext(fieldNode)
		// normalized is already whitespace-processed by normalizeWhiteSpace above
		// (QName/NOTATION have whiteSpace="collapse", so leading/trailing XSD
		// whitespace is gone). Resolve it directly rather than re-trimming with
		// strings.TrimSpace, which strips Unicode whitespace (e.g. NBSP) that is
		// NOT XSD whitespace and would corrupt the canonical key — producing false
		// duplicate/keyref diagnostics for QName values containing such characters.
		//
		// resolveNotationOrQNameValue applies the unprefixed-NOTATION
		// default-namespace resolution (ns[""]) so two NOTATION values whose
		// canonical names agree — a prefixed p:jpeg and an unprefixed jpeg under a
		// default namespace urn:p — collide on the same key and an xs:unique/xs:key
		// duplicate is caught, consistent with the facet comparison paths. xs:QName
		// keeps the no-default value-space rule (the helper's TypeNotation gate
		// leaves the QName branch on resolveLexicalQName).
		qn, err := resolveNotationOrQNameValue(normalized, builtinLocal, ns)
		if err != nil {
			return raw
		}
		return helium.ClarkName(qn.NS, qn.Local)
	}
	key, _ := value.CanonicalKey(normalized, builtinLocal)
	return key
}

// unionActiveMember resolves the active member type of a union value: the first
// DIRECT member (declaration order) the value fully VALIDATES against — matching
// validateUnionValue's active-member semantics. Members are NOT pre-flattened:
// each direct member (resolved through the base chain via resolveUnionMembers) is
// validated AS-IS via the validator's full path (typeAcceptsValue → validateValue:
// lexical space AND the member's own facets AND, for a nested-union member, that
// union's wrapper facets and its own member resolution). So a nested-union member
// whose wrapper restriction rejects the value by FACET is correctly skipped — a
// pre-flattened leaf would lose that wrapper facet and falsely accept the value.
// A member that is itself a list or union is returned as-is; the caller then
// canonicalizes it by recursing through canonicalValueKey, so list/union members
// are handled in their own value space rather than as opaque atoms.
//
// fieldNode supplies the namespace context threaded into member validation so a
// QName/NOTATION member with a QName-valued facet (e.g. an enumeration of
// prefixed names) resolves its prefixes against the field node's in-scope
// namespaces. Returns nil when no member accepts raw (the caller then falls back
// to the raw value).
func unionActiveMember(ctx context.Context, raw string, fieldNode helium.Node, td *TypeDef) *TypeDef {
	for _, m := range resolveUnionMembers(td) {
		if m == nil {
			continue
		}
		if typeAcceptsValue(ctx, m, raw, fieldNode) {
			return m
		}
	}
	return nil
}

// typeAcceptsValue reports whether a value is valid against a simple type using
// the validator's full validation path (lexical space, facets, list/union
// varieties), suppressing all diagnostics. It mirrors what validateValue does in
// the content validator, so IDC active-member selection cannot diverge from
// validation. fieldNode's in-scope namespaces are passed as the value's namespace
// context so QName/NOTATION facets (e.g. enumerations of prefixed names) resolve
// against the same bindings the instance value uses.
func typeAcceptsValue(ctx context.Context, td *TypeDef, raw string, fieldNode helium.Node) bool {
	// IDC field-type resolution has no version source here, so the throwaway
	// context defaults to Version10 (strict). The field value is already validated
	// under the schema's real version on the main path; the only Phase-1 gap is a
	// 1.1-only lexical form (e.g. "+INF") in an IDC field whose type is a
	// float/date union, which a later phase can tighten.
	vc := &validationContext{
		errorHandler:  helium.NilErrorHandler{},
		suppressDepth: 1,
	}
	return validateValue(ctx, raw, fieldNodeNSContext(fieldNode), td, "", "", 0, vc) == nil
}

// fieldNodeNSContext returns the in-scope namespace bindings visible at an IDC
// field node, used to resolve lexical QName/NOTATION prefixes. For an attribute
// node it uses the owning element's context.
func fieldNodeNSContext(n helium.Node) map[string]string {
	switch v := n.(type) {
	case *helium.Element:
		return collectNSContext(v)
	case *helium.Attribute:
		if owner, ok := v.Parent().(*helium.Element); ok {
			return collectNSContext(owner)
		}
	}
	return map[string]string{}
}

// fieldNodeClass classifies an identity-constraint <xs:field> result node per
// cvc-identity-constraint.3 / XSD 1.1 §3.11.4.
type fieldNodeClass int

const (
	// fieldGoverned: the node was genuinely SCHEMA-ASSESSED with a simple (or
	// simple-content) type — it contributes its typed value to the key sequence.
	fieldGoverned fieldNodeClass = iota
	// fieldSkipped: the node contributes NO value; it drops out of the qualified
	// node-set. In XSD 1.1 this covers EVERY unassessed field node — a
	// processContents="skip" wildcard-matched element, an attribute admitted only
	// by a skip/lax-without-global wildcard or whose ancestor element is skipped,
	// and any node whose type could not be assessed.
	fieldSkipped
	// fieldViolation: cvc-identity-constraint.3 is violated — a genuinely-assessed
	// COMPLEX element (element-only/mixed/empty content), or, in XSD 1.0 (which has
	// no skipped-node relaxation), any unassessed/unresolved node.
	fieldViolation
)

// classifyFieldNode classifies a single IDC field-result node. The determination
// is based ONLY on GENUINE ASSESSMENT records — assessedElemType for elements and
// assessedAttrs for attributes, both written exclusively at real pass-1 assessment
// sites — never on the canonicalization-only actualElemType map or a schema-
// declaration fallback. So a skip-content element, an attribute whose ancestor is
// skipped or that was admitted only by a skip/lax-without-global wildcard, and any
// node with no PSVI type are all UNASSESSED and classified version-aware
// (unassessedFieldClass); a wildcard-admitted UNTYPED global attribute
// (xs:anySimpleType) and an untyped declared attribute use ARE assessed and stay
// GOVERNED.
func (vc *validationContext) classifyFieldNode(n helium.Node) fieldNodeClass {
	switch v := n.(type) {
	case *helium.Attribute:
		if _, ok := vc.assessedAttrs[v]; ok {
			// Every assessed attribute has a SIMPLE type (an untyped use is
			// xs:anySimpleType); an attribute can never be complex-typed.
			return fieldGoverned
		}
		return vc.unassessedFieldClass()
	case *helium.Element:
		td, ok := vc.assessedElemType[v]
		if !ok {
			return vc.unassessedFieldClass()
		}
		// A simple type definition, or a complex type with SIMPLE content.
		if !td.IsComplex || td.ContentType == ContentTypeSimple {
			return fieldGoverned
		}
		return fieldViolation
	default:
		return fieldGoverned
	}
}

// unassessedFieldClass classifies a field node carrying no genuine assessment
// record. XSD 1.1 §3.11.4 treats it as skipped (no value); XSD 1.0 has no such
// relaxation, so an unassessed/unresolved node is a cvc-identity-constraint.3
// violation.
func (vc *validationContext) unassessedFieldClass() fieldNodeClass {
	if vc.version == Version11 {
		return fieldSkipped
	}
	return fieldViolation
}

// resolveFieldType resolves the *TypeDef of an IDC field node. host/hostDecl are
// the element the constraint is declared on, used to descend the schema content
// model down to the field node. Returns nil when the type cannot be determined,
// in which case the caller falls back to raw-string comparison for that field.
func (vc *validationContext) resolveFieldType(n helium.Node, host *helium.Element, hostDecl *ElementDecl, schema *Schema) *TypeDef {
	switch v := n.(type) {
	case *helium.Element:
		return vc.resolveElemType(v, host, hostDecl, schema)
	case *helium.Attribute:
		return vc.resolveAttrType(v, host, hostDecl, schema)
	default:
		return nil
	}
}

// resolveAttrType resolves the *TypeDef of an attribute's declared type,
// preferring the owning element's complex-type attribute uses and falling back
// to a matching global attribute declaration.
func (vc *validationContext) resolveAttrType(attr *helium.Attribute, host *helium.Element, hostDecl *ElementDecl, schema *Schema) *TypeDef {
	aqn := QName{Local: attr.LocalName(), NS: attr.URI()}

	if owner, ok := attr.Parent().(*helium.Element); ok {
		if td := vc.resolveElemType(owner, host, hostDecl, schema); td != nil {
			if at := attrUseType(td, aqn, schema); at != nil {
				return at
			}
		}
	}

	if ga, ok := schema.globalAttrs[aqn]; ok {
		if td := attrUseTypeDef(ga, schema); td != nil {
			return td
		}
	}
	return nil
}

// resolveElemType resolves the schema type of an instance element. It first
// consults the actual-type map recorded during pass-1 content validation (which
// already accounts for any xsi:type override). Failing that, it descends the
// host element's content model along the element's ancestor chain. The element
// must be a descendant of host (true for IDC selector/field results). Falls back
// to a global element declaration lookup.
func (vc *validationContext) resolveElemType(target, host *helium.Element, hostDecl *ElementDecl, schema *Schema) *TypeDef {
	if td := vc.actualElemType[target]; td != nil {
		return td
	}
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
	for cur := range baseChain(td) {
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
	for cur := range baseChain(td) {
		for _, au := range cur.Attributes {
			if au.Name != aqn {
				continue
			}
			return attrUseTypeDef(au, schema)
		}
	}
	return nil
}

// attrUseTypeDef resolves the effective simple type for an attribute use,
// mirroring validationContext.attrUseType: an inline anonymous <xs:simpleType>
// (au.Type) takes precedence over a named type reference (au.TypeName).
func attrUseTypeDef(au *AttrUse, schema *Schema) *TypeDef {
	if au.Type != nil {
		return au.Type
	}
	if td, ok := schema.types[au.TypeName]; ok {
		return td
	}
	return nil
}

// attrUseEffectiveTypeDef resolves an attribute use's effective simple type,
// defaulting an ABSENT type to the registered xs:anySimpleType. Per XSD §3.2.2.1
// an attribute declaration with no <xs:simpleType> and no @type has {type
// definition} = xs:anySimpleType (the simple ur-type). Unlike attrUseTypeDef
// (which returns nil so the IDC canonical-key path can fall back to raw-string
// comparison), this never returns nil, so the restriction-derivation check treats
// an untyped derived attribute as the ur-type — a SUPERTYPE — and correctly
// rejects it as a WIDENING of a narrower base attribute type.
func attrUseEffectiveTypeDef(au *AttrUse, schema *Schema) *TypeDef {
	if td := attrUseTypeDef(au, schema); td != nil {
		return td
	}
	return schema.types[QName{Local: typeAnySimpleType, NS: lexicon.NamespaceXSD}]
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
