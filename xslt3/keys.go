package xslt3

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/helium"
)

// keyTable is a built key index for a specific xsl:key definition.
type keyTable struct {
	def      *KeyDef
	entries  map[string][]helium.Node // key-value -> matching nodes
	building bool
	built    bool
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

	kd, ok := ec.stylesheet.keys[name]
	if !ok {
		return nil, dynamicError(errCodeXTDE1170, "unknown key %q", name)
	}

	kt := &keyTable{
		def:      kd,
		entries:  make(map[string][]helium.Node),
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

	err := helium.Walk(root, func(node helium.Node) error {
		if !kd.Match.matchPattern(ec, node) {
			return nil
		}
		ec.contextNode = node
		ec.currentNode = node

		if kd.Use != nil {
			// use="expr" form
			xpathCtx := ec.newXPathContext(node)
			result, err := kd.Use.Evaluate(xpathCtx, node)
			if err != nil {
				return err
			}
			for _, item := range result.Sequence() {
				keyVal := stringifyItem(item)
				kt.entries[keyVal] = append(kt.entries[keyVal], node)
			}
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
			for _, item := range seq {
				keyVal := stringifyItem(item)
				kt.entries[keyVal] = append(kt.entries[keyVal], node)
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

// lookupKeyInDoc looks up nodes by key name and value in the given document root.
func (ec *execContext) lookupKeyInDoc(name, value string, root helium.Node) ([]helium.Node, error) {
	cacheKey := fmt.Sprintf("%s@%p", name, root)
	if kt, ok := ec.keyTables[cacheKey]; ok && kt.building {
		return nil, nil
	}
	kt, err := ec.buildKeyTable(name, root)
	if err != nil {
		return nil, err
	}
	return kt.entries[value], nil
}
