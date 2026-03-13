package xslt3

import (
	"github.com/lestrrat-go/helium"
)

// keyTable is a built key index for a specific xsl:key definition.
type keyTable struct {
	def     *KeyDef
	entries map[string][]helium.Node // key-value -> matching nodes
	built   bool
}

// buildKeyTable builds or retrieves a key table for the given key name.
func (ec *execContext) buildKeyTable(name string) (*keyTable, error) {
	if kt, ok := ec.keyTables[name]; ok && kt.built {
		return kt, nil
	}

	kd, ok := ec.stylesheet.keys[name]
	if !ok {
		return nil, dynamicError(errCodeXTDE1170, "unknown key %q", name)
	}

	kt := &keyTable{
		def:     kd,
		entries: make(map[string][]helium.Node),
	}

	// Walk the entire source document and build the index
	err := helium.Walk(ec.sourceDoc, func(node helium.Node) error {
		if !kd.Match.matchPattern(ec, node) {
			return nil
		}
		xpathCtx := ec.newXPathContext(node)
		result, err := kd.Use.Evaluate(xpathCtx, node)
		if err != nil {
			return err
		}
		keyVal := stringifyResult(result)
		kt.entries[keyVal] = append(kt.entries[keyVal], node)
		return nil
	})
	if err != nil {
		return nil, err
	}

	kt.built = true
	if ec.keyTables == nil {
		ec.keyTables = make(map[string]*keyTable)
	}
	ec.keyTables[name] = kt
	return kt, nil
}

// lookupKey looks up nodes by key name and value.
func (ec *execContext) lookupKey(name, value string) ([]helium.Node, error) {
	kt, err := ec.buildKeyTable(name)
	if err != nil {
		return nil, err
	}
	return kt.entries[value], nil
}
