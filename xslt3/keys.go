package xslt3

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

// keyEntry pairs a typed atomic key value with the node it indexes.
type keyEntry struct {
	key  xpath3.AtomicValue
	node helium.Node
}

// compositeKeyEntry stores a composite key (sequence of atomic values) alongside
// the node it indexes.
type compositeKeyEntry struct {
	keys []xpath3.AtomicValue
	node helium.Node
}

// keyTable is a built key index for a specific xsl:key name.
type keyTable struct {
	defs           []*KeyDef
	entries        map[string][]keyEntry          // canonicalKey -> entries (non-composite)
	compositeEntry map[string][]compositeKeyEntry // compositeCanonical -> entries (composite)
	building       bool
	built          bool
}

// canonicalKey produces a hash key that groups potentially-equal values
// into the same bucket. Typed comparison refines matches within a bucket.
// Values that could be equal under XPath eq semantics (including
// untypedAtomic promotion) must share the same canonical key.
func canonicalKey(av xpath3.AtomicValue) string {
	// For date/time types, normalize to UTC so that equivalent instants
	// in different timezones share the same canonical key.
	switch av.TypeName {
	case xpath3.TypeDateTime, xpath3.TypeDateTimeStamp:
		if t, ok := av.Value.(time.Time); ok {
			return "DT:" + t.UTC().Format(time.RFC3339Nano)
		}
	case xpath3.TypeDate:
		if t, ok := av.Value.(time.Time); ok {
			return "D:" + t.UTC().Format("2006-01-02Z07:00")
		}
	case xpath3.TypeTime:
		if t, ok := av.Value.(time.Time); ok {
			return "T:" + t.UTC().Format("15:04:05.999999999Z07:00")
		}
	}

	s, _ := xpath3.AtomicToString(av)
	// Group numerics together so integer 4 and double 4.0 can be
	// compared in the same bucket.
	if av.IsNumeric() {
		return "N:" + s
	}
	// Group untypedAtomic, string, and all string-derived types together
	// since untypedAtomic is promoted to string for eq comparison.
	switch av.TypeName {
	case xpath3.TypeUntypedAtomic, xpath3.TypeString,
		xpath3.TypeNormalizedString, xpath3.TypeToken,
		xpath3.TypeLanguage, xpath3.TypeName, xpath3.TypeNCName,
		xpath3.TypeNMTOKEN, xpath3.TypeNMTOKENS,
		xpath3.TypeENTITY, xpath3.TypeID, xpath3.TypeIDREF, xpath3.TypeIDREFS,
		xpath3.TypeAnyURI:
		return "S:" + s
	}
	return av.TypeName + ":" + s
}

// compositeCanonicalKey produces a canonical key for a sequence of atomic values
// used as a composite key. Individual canonical keys are joined with NUL bytes.
func compositeCanonicalKey(avs []xpath3.AtomicValue) string {
	parts := make([]string, len(avs))
	for i, av := range avs {
		parts[i] = canonicalKey(av)
	}
	return strings.Join(parts, "\x00")
}

// compositeAtomicEquals tests whether two composite keys are equal element-by-element.
func compositeAtomicEquals(a, b []xpath3.AtomicValue) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !xpath3.AtomicEquals(a[i], b[i]) {
			return false
		}
	}
	return true
}

// buildKeyTable builds or retrieves a key table for the given key name
// scoped to the given document root.
func (ec *execContext) buildKeyTable(name string, root helium.Node) (*keyTable, error) {
	if ec.keyTables == nil {
		ec.keyTables = make(map[string]*keyTable)
	}

	// Cache key includes both the key name and the document identity
	cacheKey := fmt.Sprintf("%s@%p", name, root)
	if kt, ok := ec.keyTables[cacheKey]; ok {
		if kt.built || kt.building {
			return kt, nil
		}
	}

	defs, ok := ec.stylesheet.keys[name]
	if !ok || len(defs) == 0 {
		return nil, dynamicError(errCodeXTDE1170, "unknown key %q", name)
	}

	composite := len(defs) > 0 && defs[0].Composite

	kt := &keyTable{
		defs:     defs,
		entries:  make(map[string][]keyEntry),
		building: true,
	}
	if composite {
		kt.compositeEntry = make(map[string][]compositeKeyEntry)
	}
	ec.keyTables[cacheKey] = kt

	// Walk the document and build the index.
	// Save/restore contextNode and currentNode so current() works in use expr.
	savedContext := ec.contextNode
	savedCurrent := ec.currentNode
	defer func() {
		ec.contextNode = savedContext
		ec.currentNode = savedCurrent
	}()

	// Track which nodes have been added for each canonical key to avoid
	// duplicates when multiple defs match the same node.
	type seenKey struct {
		canonical string
		node      helium.Node
	}
	seen := make(map[seenKey]struct{})

	// indexNode tries to match a single node against all key defs and index it.
	indexNode := func(node helium.Node) error {
		for _, kd := range defs {
			if !kd.Match.matchPattern(ec, node) {
				continue
			}
			ec.contextNode = node
			ec.currentNode = node

			var items []xpath3.Item
			if kd.Use != nil {
				// use="expr" form
				xpathCtx := ec.newXPathContext(node)
				result, err := kd.Use.Evaluate(xpathCtx, node)
				if err != nil {
					return err
				}
				items = result.Sequence()
			} else if len(kd.Body) > 0 {
				// Content constructor form: evaluate body as sequence
				ctx := ec.transformCtx
				if ctx == nil {
					ctx = context.Background()
				}
				seq, err := ec.evaluateBody(ctx, kd.Body)
				if err != nil {
					return err
				}
				items = seq
			}

			if composite {
				// Composite key: the entire sequence forms a single key tuple
				avs := make([]xpath3.AtomicValue, 0, len(items))
				for _, item := range items {
					av, err := xpath3.AtomizeItem(item)
					if err != nil {
						continue
					}
					avs = append(avs, av)
				}
				if len(avs) == 0 {
					continue
				}
				ck := compositeCanonicalKey(avs)
				sk := seenKey{canonical: ck, node: node}
				if _, dup := seen[sk]; dup {
					continue
				}
				seen[sk] = struct{}{}
				kt.compositeEntry[ck] = append(kt.compositeEntry[ck], compositeKeyEntry{keys: avs, node: node})
			} else {
				// Non-composite: each value is a separate key entry
				for _, item := range items {
					av, err := xpath3.AtomizeItem(item)
					if err != nil {
						continue
					}
					// NaN values never match anything (NaN != NaN),
					// so do not index them.
					if av.IsNaN() {
						continue
					}
					ck := canonicalKey(av)
					sk := seenKey{canonical: ck, node: node}
					if _, dup := seen[sk]; dup {
						continue
					}
					seen[sk] = struct{}{}
					kt.entries[ck] = append(kt.entries[ck], keyEntry{key: av, node: node})
				}
			}
		}
		return nil
	}

	// Walk the document, also visiting attribute and namespace nodes.
	err := helium.Walk(root, func(node helium.Node) error {
		if err := indexNode(node); err != nil {
			return err
		}
		// Also visit attribute and namespace nodes on element nodes.
		// helium.Walk only visits child nodes; key patterns can match
		// attribute::* and namespace-node() which require explicit iteration.
		if elem, ok := node.(*helium.Element); ok {
			for _, attr := range elem.Attributes() {
				if err := indexNode(attr); err != nil {
					return err
				}
			}
			// Collect in-scope namespace nodes (including inherited ones)
			// to match XPath namespace axis semantics.
			nsNodes := collectInScopeNSNodes(elem)
			for _, nsNode := range nsNodes {
				if err := indexNode(nsNode); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		delete(ec.keyTables, cacheKey)
		return nil, err
	}

	kt.building = false
	kt.built = true
	return kt, nil
}

// lookupKey looks up nodes by key name and typed value in the given document root.
func (ec *execContext) lookupKey(name string, value xpath3.AtomicValue, root helium.Node) ([]helium.Node, error) {
	cacheKey := fmt.Sprintf("%s@%p", name, root)
	if kt, ok := ec.keyTables[cacheKey]; ok && kt.building {
		return nil, nil
	}
	kt, err := ec.buildKeyTable(name, root)
	if err != nil {
		return nil, err
	}

	// NaN lookup never matches.
	if value.IsNaN() {
		return nil, nil
	}

	ck := canonicalKey(value)
	candidates := kt.entries[ck]
	if len(candidates) == 0 {
		return nil, nil
	}

	// Filter candidates using typed eq comparison.
	var result []helium.Node
	for _, entry := range candidates {
		if xpath3.AtomicEquals(value, entry.key) {
			result = append(result, entry.node)
		}
	}
	return result, nil
}

// lookupCompositeKey looks up nodes by composite key (sequence of values).
func (ec *execContext) lookupCompositeKey(name string, values []xpath3.AtomicValue, root helium.Node) ([]helium.Node, error) {
	cacheKey := fmt.Sprintf("%s@%p", name, root)
	if kt, ok := ec.keyTables[cacheKey]; ok && kt.building {
		return nil, nil
	}
	kt, err := ec.buildKeyTable(name, root)
	if err != nil {
		return nil, err
	}

	if kt.compositeEntry == nil {
		return nil, nil
	}

	ck := compositeCanonicalKey(values)
	candidates := kt.compositeEntry[ck]
	if len(candidates) == 0 {
		return nil, nil
	}

	var result []helium.Node
	for _, entry := range candidates {
		if compositeAtomicEquals(values, entry.keys) {
			result = append(result, entry.node)
		}
	}
	return result, nil
}

// collectInScopeNSNodes returns NamespaceNodeWrapper nodes for all in-scope
// namespaces of an element, including those inherited from ancestors.
// This matches XPath namespace axis semantics.
func collectInScopeNSNodes(elem *helium.Element) []helium.Node {
	seen := make(map[string]struct{})
	var result []helium.Node

	// Walk from the element up through ancestors collecting namespace declarations.
	var cur helium.Node = elem
	for cur != nil {
		if e, ok := cur.(*helium.Element); ok {
			for _, ns := range e.Namespaces() {
				prefix := ns.Prefix()
				if _, exists := seen[prefix]; exists {
					continue
				}
				// Skip undeclarations (xmlns="" or xmlns:p="")
				if ns.URI() == "" {
					seen[prefix] = struct{}{}
					continue
				}
				seen[prefix] = struct{}{}
				result = append(result, helium.NewNamespaceNodeWrapper(ns, elem))
			}
		}
		cur = cur.Parent()
	}

	// Add the implicit xml namespace if not already present.
	if _, exists := seen["xml"]; !exists {
		xmlNS := helium.NewNamespace("xml", "http://www.w3.org/XML/1998/namespace")
		result = append(result, helium.NewNamespaceNodeWrapper(xmlNS, elem))
	}

	return result
}
