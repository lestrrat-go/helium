package xslt3

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

// Sentinel errors for xsl:break and xsl:next-iteration control flow.
var errBreak = errors.New("xsl:break")
var errNextIter = errors.New("xsl:next-iteration")

// execSourceDocument executes xsl:source-document by loading the referenced
// document into a DOM tree and executing the body with that document as context.
func (ec *execContext) execSourceDocument(ctx context.Context, inst *SourceDocumentInst) error {
	// Evaluate the href AVT to get the URI string.
	uri, err := inst.Href.evaluate(ctx, ec.contextNode)
	if err != nil {
		return err
	}

	// Check the document cache first.
	doc, ok := ec.docCache[uri]
	if !ok {
		// Resolve URI relative to stylesheet base URI.
		resolvedURI := uri
		if ec.stylesheet.baseURI != "" && !strings.Contains(uri, "://") && !filepath.IsAbs(uri) {
			baseDir := filepath.Dir(ec.stylesheet.baseURI)
			resolvedURI = filepath.Join(baseDir, uri)
		}

		data, err := os.ReadFile(resolvedURI)
		if err != nil {
			return dynamicError("FODC0002", "xsl:source-document cannot load %q: %v", uri, err)
		}

		p := helium.NewParser()
		p.SetBaseURI(resolvedURI)
		doc, err = p.Parse(ctx, data)
		if err != nil {
			return dynamicError("FODC0002", "xsl:source-document cannot parse %q: %v", uri, err)
		}

		// Apply xsl:strip-space to the loaded document so that whitespace-only
		// text nodes are removed before XPath evaluation sees them.
		if len(ec.stylesheet.stripSpace) > 0 {
			ec.stripWhitespaceFromDoc(doc)
		}

		if ec.docCache == nil {
			ec.docCache = make(map[string]*helium.Document)
		}
		ec.docCache[uri] = doc
	}

	// Save and restore source document and context nodes.
	savedSource := ec.sourceDoc
	savedContext := ec.contextNode
	savedCurrent := ec.currentNode
	ec.sourceDoc = doc
	ec.contextNode = doc
	ec.currentNode = doc
	defer func() {
		ec.sourceDoc = savedSource
		ec.contextNode = savedContext
		ec.currentNode = savedCurrent
	}()

	// Execute the body with the loaded document as context.
	for _, child := range inst.Body {
		if err := ec.executeInstruction(ctx, child); err != nil {
			return err
		}
	}
	return nil
}

// execIterate executes xsl:iterate, processing each item in the selected
// sequence with mutable iteration parameters.
func (ec *execContext) execIterate(ctx context.Context, inst *IterateInst) error {
	// Evaluate the select expression.
	xpathCtx := ec.newXPathContext(ec.contextNode)
	result, err := inst.Select.Evaluate(xpathCtx, ec.contextNode)
	if err != nil {
		return err
	}
	seq := result.Sequence()

	// Initialize iterate params from their defaults.
	paramVals := make(map[string]xpath3.Sequence, len(inst.Params))
	for _, p := range inst.Params {
		if p.Select != nil {
			pCtx := ec.newXPathContext(ec.contextNode)
			pResult, err := p.Select.Evaluate(pCtx, ec.contextNode)
			if err != nil {
				return err
			}
			paramVals[p.Name] = pResult.Sequence()
		} else if len(p.Body) > 0 {
			val, err := ec.evaluateBody(ctx, p.Body)
			if err != nil {
				return err
			}
			paramVals[p.Name] = val
		} else {
			paramVals[p.Name] = xpath3.EmptySequence()
		}
	}

	// Save and restore context.
	savedCurrent := ec.currentNode
	savedContext := ec.contextNode
	savedPos := ec.position
	savedSize := ec.size
	savedItem := ec.contextItem
	defer func() {
		ec.currentNode = savedCurrent
		ec.contextNode = savedContext
		ec.position = savedPos
		ec.size = savedSize
		ec.contextItem = savedItem
	}()

	ec.size = len(seq)

	completed := true
	for i, item := range seq {
		ec.position = i + 1

		// Set context item/node.
		if ni, ok := item.(xpath3.NodeItem); ok {
			ec.currentNode = ni.Node
			ec.contextNode = ni.Node
			ec.contextItem = nil
		} else {
			ec.contextItem = item
		}

		// Push var scope and set iterate param values.
		ec.pushVarScope()
		for name, val := range paramVals {
			ec.setVar(name, val)
		}

		// Execute body.
		var bodyErr error
		for _, child := range inst.Body {
			bodyErr = ec.executeInstruction(ctx, child)
			if bodyErr != nil {
				break
			}
		}

		ec.popVarScope()

		if bodyErr != nil {
			if errors.Is(bodyErr, errBreak) {
				completed = false
				break
			}
			if errors.Is(bodyErr, errNextIter) {
				// Update params from next-iteration with-params.
				if ec.nextIterParams != nil {
					for name, val := range ec.nextIterParams {
						paramVals[name] = val
					}
					ec.nextIterParams = nil
				}
				continue
			}
			return bodyErr
		}
	}

	if !completed {
		// xsl:break was executed — output the break value if any.
		if ec.breakValue != nil {
			out := ec.currentOutput()
			if out.captureItems {
				// In capture mode (inside variable/function body),
				// append items directly rather than writing to DOM,
				// so non-node items (maps, arrays) are preserved.
				out.pendingItems = append(out.pendingItems, ec.breakValue...)
			} else {
				if err := ec.outputSequence(ec.breakValue); err != nil {
					return err
				}
			}
			ec.breakValue = nil
		}
	} else if len(inst.OnCompletion) > 0 {
		// Execute on-completion if present and loop completed normally.
		// Per spec: within xsl:on-completion, there is no context item,
		// context position, or context size. Set them to "absent" so that
		// any reference raises XPDY0002.
		ec.contextNode = nil
		ec.currentNode = nil
		ec.contextItem = nil
		ec.position = 0
		ec.size = 0

		ec.pushVarScope()
		for name, val := range paramVals {
			ec.setVar(name, val)
		}
		for _, child := range inst.OnCompletion {
			if err := ec.executeInstruction(ctx, child); err != nil {
				ec.popVarScope()
				return err
			}
		}
		ec.popVarScope()
	}

	return nil
}

// execFork executes xsl:fork by running each branch sequentially.
// In a true streaming implementation branches would run concurrently,
// but for the DOM-materialization strategy sequential execution is correct.
func (ec *execContext) execFork(ctx context.Context, inst *ForkInst) error {
	for _, branch := range inst.Branches {
		for _, child := range branch {
			if err := ec.executeInstruction(ctx, child); err != nil {
				return err
			}
		}
	}
	return nil
}

// execBreak executes xsl:break, which terminates the enclosing xsl:iterate.
func (ec *execContext) execBreak(ctx context.Context, inst *BreakInst) error {
	if inst.Select != nil {
		xpathCtx := ec.newXPathContext(ec.contextNode)
		result, err := inst.Select.Evaluate(xpathCtx, ec.contextNode)
		if err != nil {
			return err
		}
		ec.breakValue = result.Sequence()
	} else if len(inst.Body) > 0 {
		val, err := ec.evaluateBody(ctx, inst.Body)
		if err != nil {
			return err
		}
		ec.breakValue = val
	}
	return errBreak
}

// execNextIteration executes xsl:next-iteration, which signals the enclosing
// xsl:iterate to advance to the next item with updated parameter values.
func (ec *execContext) execNextIteration(ctx context.Context, inst *NextIterationInst) error {
	params := make(map[string]xpath3.Sequence, len(inst.Params))
	for _, wp := range inst.Params {
		val, err := ec.evaluateWithParam(ctx, wp)
		if err != nil {
			return err
		}
		params[wp.Name] = val
	}
	ec.nextIterParams = params
	return errNextIter
}

// mergeKeyValue holds a single merge key as an XPath atomic value for
// type-aware comparison (dates, numbers, strings, etc.).
type mergeKeyValue struct {
	atom xpath3.AtomicValue // the actual typed atomic value
	str  string             // string fallback (used when atom is zero)
}

// mergeSourceItems holds the items from one merge source along with
// their pre-extracted sort keys and the source name.
type mergeSourceItems struct {
	name  string
	items xpath3.Sequence
	keys  [][]mergeKeyValue // keys[i] corresponds to items[i]
}

// mergeGroup represents one group of items that share the same merge key.
type mergeGroup struct {
	key      xpath3.Sequence            // the merge key value (first key of group)
	allItems xpath3.Sequence            // all items across all sources
	byName   map[string]xpath3.Sequence // items per named source
}

// mergeKeyOrder tracks the descending flag for each key level.
type mergeKeyOrder struct {
	desc bool
}

// compareMergeKeyValues compares two merge key value arrays using the
// specified orders. Returns -1, 0, or +1.
func compareMergeKeyValues(a, b []mergeKeyValue, orders []mergeKeyOrder) int {
	for i, ord := range orders {
		if i >= len(a) || i >= len(b) {
			break
		}
		c := compareSingleMergeKey(a[i], b[i])
		if ord.desc {
			c = -c
		}
		if c != 0 {
			return c
		}
	}
	return 0
}

// compareSingleMergeKey compares two single merge key values.
func compareSingleMergeKey(a, b mergeKeyValue) int {
	// If both have typed atomic values, use XPath value comparison.
	if a.atom.TypeName != "" && b.atom.TypeName != "" {
		lt, err := xpath3.ValueCompare(xpath3.TokenLt, a.atom, b.atom)
		if err == nil {
			if lt {
				return -1
			}
			eq, err2 := xpath3.ValueCompare(xpath3.TokenEq, a.atom, b.atom)
			if err2 == nil && eq {
				return 0
			}
			return 1
		}
		// Fall back to string comparison if type comparison fails.
	}

	// Fall back to string comparison.
	aStr := a.str
	bStr := b.str
	if a.atom.TypeName != "" {
		s, err := xpath3.AtomicToString(a.atom)
		if err == nil {
			aStr = s
		}
	}
	if b.atom.TypeName != "" {
		s, err := xpath3.AtomicToString(b.atom)
		if err == nil {
			bStr = s
		}
	}
	if aStr < bStr {
		return -1
	}
	if aStr > bStr {
		return 1
	}
	return 0
}

// execMerge executes xsl:merge by loading, sorting, and merging items from
// multiple sources, then executing the merge-action for each group of items
// sharing the same key.
func (ec *execContext) execMerge(ctx context.Context, inst *MergeInst) error {
	// 1. Gather items from all sources.
	var allSources []mergeSourceItems
	for srcIdx, src := range inst.Sources {
		items, err := ec.gatherMergeSourceItems(ctx, src)
		if err != nil {
			return err
		}

		// 2. Evaluate merge keys for items from this source using its own key defs.
		for i := range items {
			keys, err := ec.evaluateMergeKeys(ctx, &items[i], src.Keys)
			if err != nil {
				return err
			}
			items[i].keys = keys
		}

		// For sources after the first, we still need keys evaluated using
		// the source's own key definitions. The comparison uses the key
		// values which are type-compatible across sources.
		_ = srcIdx

		allSources = append(allSources, items...)
	}

	// Determine sort orders from first source's key definitions.
	keyDefs := inst.Sources[0].Keys
	orders := make([]mergeKeyOrder, len(keyDefs))
	for i, mk := range keyDefs {
		orders[i] = mergeKeyOrder{desc: mk.Order == "descending"}
	}

	// Sort each source's items by merge keys.
	for si := range allSources {
		src := &allSources[si]
		if len(src.items) <= 1 {
			continue
		}
		type indexedEntry struct {
			idx  int
			item xpath3.Item
			keys []mergeKeyValue
		}
		entries := make([]indexedEntry, len(src.items))
		for i := range src.items {
			entries[i] = indexedEntry{idx: i, item: src.items[i], keys: src.keys[i]}
		}
		slices.SortStableFunc(entries, func(a, b indexedEntry) int {
			return compareMergeKeyValues(a.keys, b.keys, orders)
		})
		for i, e := range entries {
			src.items[i] = e.item
			src.keys[i] = e.keys
		}
	}

	// 3. N-way merge: use cursors to walk through all sources.
	groups := ec.nWayMerge(allSources, orders)

	// 4. Execute the action body for each group.
	// Register current-merge-group() and current-merge-key() as XSLT functions.
	// We temporarily add them to the cached function map.
	ec.xsltFunctions() // ensure cachedFns is initialized

	var currentMergeGroupAll xpath3.Sequence
	var currentMergeGroupByName map[string]xpath3.Sequence
	var currentMergeKeySeq xpath3.Sequence

	ec.cachedFns["current-merge-group"] = &xsltFunc{
		min: 0, max: 1,
		fn: func(_ context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
			if len(args) > 0 && len(args[0]) > 0 {
				// current-merge-group('source-name')
				av, err := xpath3.AtomizeItem(args[0][0])
				if err != nil {
					return xpath3.EmptySequence(), nil
				}
				name, err := xpath3.AtomicToString(av)
				if err != nil {
					return xpath3.EmptySequence(), nil
				}
				if items, ok := currentMergeGroupByName[name]; ok {
					return items, nil
				}
				return xpath3.EmptySequence(), nil
			}
			return currentMergeGroupAll, nil
		},
	}
	ec.cachedFns["current-merge-key"] = &xsltFunc{
		min: 0, max: 0,
		fn: func(_ context.Context, _ []xpath3.Sequence) (xpath3.Sequence, error) {
			return currentMergeKeySeq, nil
		},
	}

	defer func() {
		delete(ec.cachedFns, "current-merge-group")
		delete(ec.cachedFns, "current-merge-key")
	}()

	// Save/restore context.
	savedCurrent := ec.currentNode
	savedContext := ec.contextNode
	savedPos := ec.position
	savedSize := ec.size
	ec.size = len(groups)
	defer func() {
		ec.currentNode = savedCurrent
		ec.contextNode = savedContext
		ec.position = savedPos
		ec.size = savedSize
	}()

	for i, g := range groups {
		ec.position = i + 1
		currentMergeGroupAll = g.allItems
		currentMergeGroupByName = g.byName
		currentMergeKeySeq = g.key

		// Context item is the first item in the group.
		if len(g.allItems) > 0 {
			if ni, ok := g.allItems[0].(xpath3.NodeItem); ok {
				ec.currentNode = ni.Node
				ec.contextNode = ni.Node
			}
		}

		ec.pushVarScope()
		for _, child := range inst.Action {
			if err := ec.executeInstruction(ctx, child); err != nil {
				ec.popVarScope()
				return err
			}
		}
		ec.popVarScope()
	}

	return nil
}

// gatherMergeSourceItems evaluates for-each-source or for-each-item and select
// for a single merge-source definition, returning one mergeSourceItems per
// source document/item. If for-each-source returns multiple URIs, each becomes
// a separate mergeSourceItems entry sharing the same source name.
func (ec *execContext) gatherMergeSourceItems(ctx context.Context, src *MergeSource) ([]mergeSourceItems, error) {
	var result []mergeSourceItems

	if src.ForEachSource != nil {
		// Evaluate for-each-source to get URI(s).
		xpathCtx := ec.newXPathContext(ec.contextNode)
		uriResult, err := src.ForEachSource.Evaluate(xpathCtx, ec.contextNode)
		if err != nil {
			return nil, err
		}
		uriSeq := uriResult.Sequence()

		for _, uriItem := range uriSeq {
			av, err := xpath3.AtomizeItem(uriItem)
			if err != nil {
				return nil, err
			}
			uri, err := xpath3.AtomicToString(av)
			if err != nil {
				return nil, err
			}

			// Load document from URI.
			doc, err := ec.loadMergeDocument(ctx, uri)
			if err != nil {
				return nil, err
			}

			// Evaluate select against the document.
			items, err := ec.evaluateMergeSelect(ctx, src, doc)
			if err != nil {
				return nil, err
			}

			result = append(result, mergeSourceItems{
				name:  src.Name,
				items: items,
			})
		}
	} else if src.ForEachItem != nil {
		// Evaluate for-each-item to get item(s).
		xpathCtx := ec.newXPathContext(ec.contextNode)
		itemResult, err := src.ForEachItem.Evaluate(xpathCtx, ec.contextNode)
		if err != nil {
			return nil, err
		}
		itemSeq := itemResult.Sequence()

		for _, sourceItem := range itemSeq {
			var contextNode helium.Node
			if ni, ok := sourceItem.(xpath3.NodeItem); ok {
				contextNode = ni.Node
			}

			// Evaluate select against this item.
			items, err := ec.evaluateMergeSelectOnNode(ctx, src, contextNode)
			if err != nil {
				return nil, err
			}

			result = append(result, mergeSourceItems{
				name:  src.Name,
				items: items,
			})
		}
	} else if src.Select != nil {
		// No for-each-source or for-each-item — just evaluate select against current context.
		xpathCtx := ec.newXPathContext(ec.contextNode)
		selResult, err := src.Select.Evaluate(xpathCtx, ec.contextNode)
		if err != nil {
			return nil, err
		}

		result = append(result, mergeSourceItems{
			name:  src.Name,
			items: selResult.Sequence(),
		})
	}

	return result, nil
}

// loadMergeDocument loads an XML document from a URI, resolving it relative
// to the stylesheet base URI.
func (ec *execContext) loadMergeDocument(ctx context.Context, uri string) (*helium.Document, error) {
	// Resolve URI relative to stylesheet base URI.
	resolvedURI := uri
	if ec.stylesheet.baseURI != "" && !strings.Contains(uri, "://") && !filepath.IsAbs(uri) {
		baseDir := filepath.Dir(ec.stylesheet.baseURI)
		resolvedURI = filepath.Join(baseDir, uri)
	}

	// Check document cache.
	if doc, ok := ec.docCache[resolvedURI]; ok {
		return doc, nil
	}

	data, readErr := os.ReadFile(resolvedURI)
	if readErr != nil {
		return nil, dynamicError("FODC0002", "xsl:merge cannot load %q: %v", uri, readErr)
	}

	p := helium.NewParser()
	p.SetBaseURI(resolvedURI)
	doc, parseErr := p.Parse(ctx, data)
	if parseErr != nil {
		return nil, dynamicError("FODC0002", "xsl:merge cannot parse %q: %v", uri, parseErr)
	}

	// Apply xsl:strip-space.
	if len(ec.stylesheet.stripSpace) > 0 {
		ec.stripWhitespaceFromDoc(doc)
	}

	if ec.docCache == nil {
		ec.docCache = make(map[string]*helium.Document)
	}
	ec.docCache[resolvedURI] = doc
	return doc, nil
}

// evaluateMergeSelect evaluates the select expression of a merge source
// against a loaded document.
func (ec *execContext) evaluateMergeSelect(ctx context.Context, src *MergeSource, doc *helium.Document) (xpath3.Sequence, error) {
	if src.Select == nil {
		return xpath3.Sequence{xpath3.NodeItem{Node: doc}}, nil
	}

	savedSource := ec.sourceDoc
	savedContext := ec.contextNode
	savedCurrent := ec.currentNode
	ec.sourceDoc = doc
	ec.contextNode = doc
	ec.currentNode = doc
	defer func() {
		ec.sourceDoc = savedSource
		ec.contextNode = savedContext
		ec.currentNode = savedCurrent
	}()

	xpathCtx := ec.newXPathContext(doc)
	result, err := src.Select.Evaluate(xpathCtx, doc)
	if err != nil {
		return nil, err
	}
	return result.Sequence(), nil
}

// evaluateMergeSelectOnNode evaluates the select expression of a merge source
// against a specific node (used with for-each-item).
func (ec *execContext) evaluateMergeSelectOnNode(ctx context.Context, src *MergeSource, node helium.Node) (xpath3.Sequence, error) {
	if src.Select == nil {
		if node != nil {
			return xpath3.Sequence{xpath3.NodeItem{Node: node}}, nil
		}
		return xpath3.EmptySequence(), nil
	}

	savedSource := ec.sourceDoc
	savedContext := ec.contextNode
	savedCurrent := ec.currentNode
	if doc, ok := node.(*helium.Document); ok {
		ec.sourceDoc = doc
	}
	ec.contextNode = node
	ec.currentNode = node
	defer func() {
		ec.sourceDoc = savedSource
		ec.contextNode = savedContext
		ec.currentNode = savedCurrent
	}()

	xpathCtx := ec.newXPathContext(node)
	result, err := src.Select.Evaluate(xpathCtx, node)
	if err != nil {
		return nil, err
	}
	return result.Sequence(), nil
}

// evaluateMergeKeys evaluates the merge key expressions for all items in a source.
func (ec *execContext) evaluateMergeKeys(ctx context.Context, src *mergeSourceItems, keyDefs []*MergeKey) ([][]mergeKeyValue, error) {
	keys := make([][]mergeKeyValue, len(src.items))

	savedContext := ec.contextNode
	savedCurrent := ec.currentNode
	savedItem := ec.contextItem
	defer func() {
		ec.contextNode = savedContext
		ec.currentNode = savedCurrent
		ec.contextItem = savedItem
	}()

	for i, item := range src.items {
		itemKeys := make([]mergeKeyValue, len(keyDefs))
		var node helium.Node

		if ni, ok := item.(xpath3.NodeItem); ok {
			node = ni.Node
			ec.contextNode = node
			ec.currentNode = node
			ec.contextItem = nil
		} else {
			// Atomic item (e.g., string from unparsed-text-lines).
			ec.contextItem = item
		}

		for k, mk := range keyDefs {
			if mk.Select == nil {
				itemKeys[k] = mergeKeyValue{}
				continue
			}

			xpathCtx := ec.newXPathContext(node)
			result, err := mk.Select.Evaluate(xpathCtx, node)
			if err != nil {
				return nil, err
			}

			seq := result.Sequence()
			// Extract the key value, preserving the atomic type.
			if len(seq) == 1 {
				if av, ok := seq[0].(xpath3.AtomicValue); ok {
					itemKeys[k] = mergeKeyValue{atom: av}
					continue
				}
			}
			// Fall back to string value.
			itemKeys[k] = mergeKeyValue{str: result.StringValue()}
		}
		keys[i] = itemKeys
	}

	return keys, nil
}

// nWayMerge performs an n-way merge of pre-sorted sources, grouping items
// that share the same key values.
func (ec *execContext) nWayMerge(sources []mergeSourceItems, orders []mergeKeyOrder) []mergeGroup {
	// Cursors: one per source, tracking current position.
	cursors := make([]int, len(sources))
	var groups []mergeGroup

	for {
		// Find the minimum key across all sources at their current cursor.
		var minKeys []mergeKeyValue
		minFound := false

		for si, src := range sources {
			if cursors[si] >= len(src.items) {
				continue // exhausted
			}
			curKeys := src.keys[cursors[si]]
			if !minFound {
				minKeys = curKeys
				minFound = true
			} else {
				cmp := compareMergeKeyValues(curKeys, minKeys, orders)
				if cmp < 0 {
					minKeys = curKeys
				}
			}
		}

		if !minFound {
			break // all sources exhausted
		}

		// Collect all items matching the minimum key from all sources.
		g := mergeGroup{
			byName: make(map[string]xpath3.Sequence),
		}

		for si, src := range sources {
			for cursors[si] < len(src.items) {
				curKeys := src.keys[cursors[si]]
				if compareMergeKeyValues(curKeys, minKeys, orders) != 0 {
					break
				}
				item := src.items[cursors[si]]
				g.allItems = append(g.allItems, item)
				if src.name != "" {
					g.byName[src.name] = append(g.byName[src.name], item)
				}
				cursors[si]++
			}
		}

		// Convert the first key to a sequence for current-merge-key().
		if len(minKeys) > 0 {
			g.key = mergeKeyValueToSequence(minKeys[0])
		}

		groups = append(groups, g)
	}

	return groups
}

// mergeKeyValueToSequence converts a mergeKeyValue to an XPath sequence for
// current-merge-key().
func mergeKeyValueToSequence(mkv mergeKeyValue) xpath3.Sequence {
	if mkv.atom.TypeName != "" {
		return xpath3.SingleAtomic(mkv.atom)
	}
	if mkv.str != "" {
		return xpath3.SingleString(mkv.str)
	}
	return xpath3.EmptySequence()
}
