package xslt3

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/internal/sequence"
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
	defs           []*keyDef
	entries        map[string][]keyEntry          // canonicalKey -> entries (non-composite)
	compositeEntry map[string][]compositeKeyEntry // compositeCanonical -> entries (composite)
	collationKey   func(string) string            // collation key function (nil = codepoint)
	collationCmp   func(string, string) int       // collation compare function (nil = codepoint)
	building       bool
	built          bool
}

// collationCanonicalKey produces a canonical key using the collation key
// function for string/untypedAtomic types. Falls back to canonicalKey
// for non-string types.
func collationCanonicalKey(av xpath3.AtomicValue, keyFn func(string) string) string {
	if keyFn == nil {
		return canonicalKey(av)
	}
	switch av.TypeName {
	case xpath3.TypeUntypedAtomic, xpath3.TypeString,
		xpath3.TypeNormalizedString, xpath3.TypeToken,
		xpath3.TypeLanguage, xpath3.TypeName, xpath3.TypeNCName,
		xpath3.TypeNMTOKEN, xpath3.TypeNMTOKENS,
		xpath3.TypeENTITY, xpath3.TypeID, xpath3.TypeIDREF, xpath3.TypeIDREFS,
		xpath3.TypeAnyURI:
		s, _ := xpath3.AtomicToString(av)
		return "S:" + keyFn(s)
	default:
		return canonicalKey(av)
	}
}

// collationAtomicEquals tests whether two atomic values are equal using
// the collation compare function for string/untypedAtomic types.
func collationAtomicEquals(a, b xpath3.AtomicValue, cmpFn func(string, string) int) bool {
	if cmpFn == nil {
		return xpath3.AtomicEquals(a, b)
	}
	// Only use collation for string-like types
	aIsStr := isStringLikeType(a.TypeName)
	bIsStr := isStringLikeType(b.TypeName)
	if aIsStr && bIsStr {
		sa, _ := xpath3.AtomicToString(a)
		sb, _ := xpath3.AtomicToString(b)
		return cmpFn(sa, sb) == 0
	}
	return xpath3.AtomicEquals(a, b)
}

func isStringLikeType(tn string) bool {
	switch tn {
	case xpath3.TypeUntypedAtomic, xpath3.TypeString,
		xpath3.TypeNormalizedString, xpath3.TypeToken,
		xpath3.TypeLanguage, xpath3.TypeName, xpath3.TypeNCName,
		xpath3.TypeNMTOKEN, xpath3.TypeNMTOKENS,
		xpath3.TypeENTITY, xpath3.TypeID, xpath3.TypeIDREF, xpath3.TypeIDREFS,
		xpath3.TypeAnyURI:
		return true
	default:
		return false
	}
}

// canonicalKey produces a hash key that groups potentially-equal values
// into the same bucket. Typed comparison refines matches within a bucket.
// Values that could be equal under XPath eq semantics (including
// untypedAtomic promotion) must share the same canonical key.
func canonicalKey(av xpath3.AtomicValue) string {
	if qv, ok := canonicalQNameValue(av); ok {
		// QName equality is based on URI and local name, not lexical prefix.
		return xpath3.TypeQName + ":" + qv.URI + "\x00" + qv.Local
	}

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
	// compared in the same bucket. Use the float64 representation as
	// the canonical key so that cross-type comparisons (e.g.
	// xs:float vs xs:double vs xs:decimal) use value equality.
	if av.IsNumeric() {
		f := av.ToFloat64()
		if math.IsNaN(f) {
			return "N:NaN"
		}
		return "N:" + strconv.FormatFloat(f, 'g', -1, 64)
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

func canonicalQNameValue(av xpath3.AtomicValue) (xpath3.QNameValue, bool) {
	if qv, ok := av.Value.(xpath3.QNameValue); ok {
		return qv, true
	}

	promoted := xpath3.PromoteSchemaType(av)
	qv, ok := promoted.Value.(xpath3.QNameValue)
	if !ok || promoted.TypeName != xpath3.TypeQName {
		return xpath3.QNameValue{}, false
	}
	return qv, true
}

// compositeCanonicalKey produces a canonical key for a sequence of atomic values
// used as a composite key. Individual canonical keys are joined with NUL bytes.
// When keyFn is non-nil, string-like values use the collation key function.
func compositeCanonicalKey(avs []xpath3.AtomicValue, keyFn func(string) string) string {
	parts := make([]string, len(avs))
	for i, av := range avs {
		parts[i] = collationCanonicalKey(av, keyFn)
	}
	return strings.Join(parts, "\x00")
}

// compositeAtomicEquals tests whether two composite keys are equal element-by-element.
// When cmpFn is non-nil, string-like values use the collation compare function.
func compositeAtomicEquals(a, b []xpath3.AtomicValue, cmpFn func(string, string) int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !collationAtomicEquals(a[i], b[i], cmpFn) {
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

	// Cache key includes the key name, the document identity, and the
	// owning package (for package-scoped key isolation).
	cacheKey := fmt.Sprintf("%s@%p@%p", name, root, ec.currentPackage)
	if kt, ok := ec.keyTables[cacheKey]; ok {
		if kt.built {
			return kt, nil
		}
		if kt.building {
			isV3 := ec.stylesheet.version >= "3.0" || ec.stylesheet.version == ""
			if isV3 && ec.keyUseExprDepth > 0 {
				// XSLT 3.0 spec (section 20.1.3): when a key's
				// use-expression references a key that is currently
				// being built, the key function returns empty (the
				// index is not yet populated).
				empty := &keyTable{built: true, entries: make(map[string][]keyEntry)}
				return empty, nil
			}
			// XTDE0640: circular key reference detected. In XSLT 2.0,
			// any key circularity is an error. In XSLT 3.0, circularity
			// outside use-expression context (e.g., in a match pattern
			// or variable binding) is also an error.
			return nil, dynamicError(errCodeXTDE0640,
				"circular reference in key %q", name)
		}
	}

	defs, ok := ec.effectiveKeys()[name]
	if !ok || len(defs) == 0 {
		return nil, dynamicError(errCodeXTDE1170, "unknown key %q", name)
	}

	composite := len(defs) > 0 && defs[0].Composite

	kt := &keyTable{
		defs:     defs,
		entries:  make(map[string][]keyEntry),
		building: true,
	}

	// Resolve collation from the key definition (all defs share the same
	// collation per XTSE1220).
	if defs[0].Collation != "" {
		keyFn, keyErr := xpath3.ResolveCollationKeyFunc(defs[0].Collation)
		if keyErr != nil {
			return nil, keyErr
		}
		cmpFn, cmpErr := xpath3.ResolveCollationCompareFunc(defs[0].Collation)
		if cmpErr != nil {
			return nil, cmpErr
		}
		kt.collationKey = keyFn
		kt.collationCmp = cmpFn
	}
	if composite {
		kt.compositeEntry = make(map[string][]compositeKeyEntry)
	}
	ec.keyTables[cacheKey] = kt
	ec.keyBuildingDepth++

	// Walk the document and build the index.
	// Save/restore contextNode, currentNode, and contextItem so current() works
	// in use expr and so that an outer atomic context item (e.g., from
	// xsl:for-each over integers) does not leak into the key use evaluation.
	savedContext := ec.contextNode
	savedCurrent := ec.currentNode
	savedItem := ec.contextItem
	defer func() {
		ec.contextNode = savedContext
		ec.currentNode = savedCurrent
		ec.contextItem = savedItem
	}()

	// Track which nodes have been added for each canonical key to avoid
	// duplicates when multiple defs match the same node.
	type seenKey struct {
		canonical string
		node      helium.Node
	}
	seen := make(map[seenKey]struct{})

	// Check whether any key definition matches attribute nodes so we know
	// whether to visit attributes during the walk.
	needsAttrs := false
	for _, kd := range defs {
		if kd.Match.matchesAttributes() {
			needsAttrs = true
			break
		}
	}

	// indexNode tries to match a single node against all key defs and index it.
	indexNode := func(node helium.Node) error {
		for _, kd := range defs {
			if !kd.Match.matchPattern(ec, node) {
				// Check if a fatal error occurred during pattern matching
				if ec.patternMatchErr != nil {
					err := ec.patternMatchErr
					ec.patternMatchErr = nil
					return err
				}
				continue
			}
			ec.contextNode = node
			ec.currentNode = node
			ec.contextItem = nil // clear atomic context; the key use context is a node

			var items []xpath3.Item
			ec.keyUseExprDepth++
			if kd.Use != nil {
				// use="expr" form
				result, err := ec.evalXPath(kd.Use, node)
				if err != nil {
					ec.keyUseExprDepth--
					return err
				}
				items = sequence.Materialize(result.Sequence())
			} else if len(kd.Body) > 0 {
				// Content constructor form: evaluate body as sequence
				ctx := ec.transformCtx
				if ctx == nil {
					ctx = context.Background()
				}
				ec.temporaryOutputDepth++
				seq, err := ec.evaluateBodyAsSequence(ctx, kd.Body)
				ec.temporaryOutputDepth--
				if err != nil {
					ec.keyUseExprDepth--
					return err
				}
				items = sequence.Materialize(seq)
			}
			ec.keyUseExprDepth--

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
				ck := compositeCanonicalKey(avs, kt.collationKey)
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
					ck := collationCanonicalKey(av, kt.collationKey)
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
	err := helium.Walk(root, helium.NodeWalkerFunc(func(node helium.Node) error {
		if err := indexNode(node); err != nil {
			return err
		}
		// Also visit attribute and namespace nodes on element nodes.
		// helium.Walk only visits child nodes; key patterns can match
		// attribute::* and namespace-node() which require explicit iteration.
		if elem, ok := node.(*helium.Element); ok {
			if needsAttrs {
				for _, attr := range elem.Attributes() {
					if err := indexNode(attr); err != nil {
						return err
					}
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
	}))
	if err != nil {
		ec.keyBuildingDepth--
		delete(ec.keyTables, cacheKey)
		return nil, err
	}

	ec.keyBuildingDepth--
	kt.building = false
	kt.built = true
	return kt, nil
}

// lookupKey looks up nodes by key name and typed value in the given document root.
func (ec *execContext) lookupKey(name string, value xpath3.AtomicValue, root helium.Node) ([]helium.Node, error) {
	kt, err := ec.buildKeyTable(name, root)
	if err != nil {
		return nil, err
	}

	// NaN lookup never matches.
	if value.IsNaN() {
		return nil, nil
	}

	ck := collationCanonicalKey(value, kt.collationKey)
	candidates := kt.entries[ck]
	if len(candidates) == 0 {
		return nil, nil
	}

	// Filter candidates using typed eq comparison (collation-aware).
	var result []helium.Node
	for _, entry := range candidates {
		if collationAtomicEquals(value, entry.key, kt.collationCmp) {
			result = append(result, entry.node)
		}
	}
	return result, nil
}

// lookupCompositeKey looks up nodes by composite key (sequence of values).
func (ec *execContext) lookupCompositeKey(name string, values []xpath3.AtomicValue, root helium.Node) ([]helium.Node, error) {
	kt, err := ec.buildKeyTable(name, root)
	if err != nil {
		return nil, err
	}

	if kt.compositeEntry == nil {
		return nil, nil
	}

	ck := compositeCanonicalKey(values, kt.collationKey)
	candidates := kt.compositeEntry[ck]
	if len(candidates) == 0 {
		return nil, nil
	}

	var result []helium.Node
	for _, entry := range candidates {
		if compositeAtomicEquals(values, entry.keys, kt.collationCmp) {
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
	if _, exists := seen[lexicon.PrefixXML]; !exists {
		xmlNS := helium.NewNamespace(lexicon.PrefixXML, lexicon.NamespaceXML)
		result = append(result, helium.NewNamespaceNodeWrapper(xmlNS, elem))
	}

	return result
}
