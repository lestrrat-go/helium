package xslt3

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

// keyEntry pairs a typed atomic key value with the node it indexes.
type keyEntry struct {
	key  xpath3.AtomicValue
	node helium.Node
}

// keyTable is a built key index for a specific xsl:key name.
type keyTable struct {
	defs     []*KeyDef
	entries  map[string][]keyEntry // canonicalKey -> entries
	building bool
	built    bool
}

// canonicalKey produces a hash key that groups potentially-equal values
// into the same bucket. Typed comparison refines matches within a bucket.
// Values that could be equal under XPath eq semantics (including
// untypedAtomic promotion) must share the same canonical key.
func canonicalKey(av xpath3.AtomicValue) string {
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

	kt := &keyTable{
		defs:     defs,
		entries:  make(map[string][]keyEntry),
		building: true,
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

	err := helium.Walk(root, func(node helium.Node) error {
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
