package xslt3

import (
	"context"
	"errors"
	"math"
	"os"
	"slices"

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
	docURI, fragment := splitURIFragment(uri)

	// Check the document cache first.
	effectiveBase := inst.BaseURI
	if effectiveBase == "" {
		effectiveBase = ec.stylesheet.baseURI
	}
	resolvedURI := resolveAgainstBaseURI(docURI, effectiveBase)

	doc, ok := ec.docCache[resolvedURI]
	if !ok {
		data, err := os.ReadFile(resolvedURI)
		if err != nil {
			return dynamicError(errCodeFODC0002, "xsl:source-document cannot load %q: %v", uri, err)
		}

		p := helium.NewParser()
		p.SetBaseURI(resolvedURI)
		doc, err = p.Parse(ctx, data)
		if err != nil {
			return dynamicError(errCodeFODC0002, "xsl:source-document cannot parse %q: %v", uri, err)
		}

		// Apply xsl:strip-space to the loaded document so that whitespace-only
		// text nodes are removed before XPath evaluation sees them.
		if len(ec.stylesheet.stripSpace) > 0 {
			ec.stripWhitespaceFromDoc(doc)
		}

		if ec.docCache == nil {
			ec.docCache = make(map[string]*helium.Document)
		}
		ec.docCache[resolvedURI] = doc
	}

	startNode := helium.Node(doc)
	if fragment != "" {
		elem := doc.GetElementByID(fragment)
		if elem == nil {
			return dynamicError(errCodeFODC0002, "xsl:source-document fragment %q not found in %q", fragment, uri)
		}
		startNode = elem
	}
	if err := ec.prepareSourceDocumentAccumulators(ctx, inst, doc); err != nil {
		return err
	}

	// Save and restore source document, context nodes, and context item.
	savedSource := ec.sourceDoc
	savedContext := ec.contextNode
	savedCurrent := ec.currentNode
	savedItem := ec.contextItem
	savedPos := ec.position
	savedSize := ec.size
	savedActiveAccums := ec.activeAccumulators
	savedRequireStreamable := ec.requireStreamableAccums
	ec.sourceDoc = doc
	ec.contextNode = startNode
	ec.currentNode = startNode
	ec.contextItem = nil // document node is the context, not an atomic item
	ec.position = 1
	ec.size = 1
	ec.activeAccumulators = make(map[string]struct{}, len(inst.UseAccumulators))
	for _, name := range inst.UseAccumulators {
		ec.activeAccumulators[name] = struct{}{}
	}
	ec.requireStreamableAccums = inst.Streamable
	defer func() {
		ec.sourceDoc = savedSource
		ec.contextNode = savedContext
		ec.currentNode = savedCurrent
		ec.contextItem = savedItem
		ec.position = savedPos
		ec.size = savedSize
		ec.activeAccumulators = savedActiveAccums
		ec.requireStreamableAccums = savedRequireStreamable
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
	paramTypes := make(map[string]string, len(inst.Params))
	for _, p := range inst.Params {
		if p.As != "" {
			paramTypes[p.Name] = p.As
		}
		var val xpath3.Sequence
		if p.Select != nil {
			pCtx := ec.newXPathContext(ec.contextNode)
			pResult, err := p.Select.Evaluate(pCtx, ec.contextNode)
			if err != nil {
				return err
			}
			val = pResult.Sequence()
		} else if len(p.Body) > 0 {
			ec.temporaryOutputDepth++
			v, err := ec.evaluateBody(ctx, p.Body)
			ec.temporaryOutputDepth--
			if err != nil {
				return err
			}
			val = v
		} else {
			val = xpath3.EmptySequence()
		}
		// Apply type coercion if as= is declared.
		if p.As != "" && len(val) > 0 {
			st := parseSequenceType(p.As)
			coerced, err := checkSequenceType(val, st, errCodeXTTE0570, "xsl:iterate parameter $"+p.Name, ec)
			if err != nil {
				return err
			}
			val = coerced
		}
		paramVals[p.Name] = val
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
						// Apply type coercion if as= is declared.
						if asType, ok := paramTypes[name]; ok && asType != "" && len(val) > 0 {
							st := parseSequenceType(asType)
							coerced, coerceErr := checkSequenceType(val, st, errCodeXTTE0570, "xsl:next-iteration parameter $"+name, ec)
							if coerceErr != nil {
								return coerceErr
							}
							val = coerced
						}
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
	atom    xpath3.AtomicValue // the actual typed atomic value
	str     string             // string fallback (used when atom is zero)
	num     float64            // numeric value (used when numeric is true)
	numeric bool               // true when data-type="number" was applied
	isNaN   bool               // true when numeric conversion produced NaN
}

// mergeSourceItems holds the items from one merge source along with
// their pre-extracted sort keys and the source name.
type mergeSourceItems struct {
	name            string
	items           xpath3.Sequence
	keys            [][]mergeKeyValue // keys[i] corresponds to items[i]
	sortBeforeMerge bool              // from parent MergeSource
	sourceIdx       int               // index into inst.Sources
}

// mergeGroup represents one group of items that share the same merge key tuple.
type mergeGroup struct {
	key      xpath3.Sequence            // the merge key tuple for current-merge-key()
	allItems xpath3.Sequence            // all items across all sources
	byName   map[string]xpath3.Sequence // items per named source
}

// mergeKeyOrder tracks the descending flag for each key level.
type mergeKeyOrder struct {
	desc bool
}

// compareMergeKeyValues compares two merge key value arrays using the
// specified orders. Returns -1, 0, or +1. Returns an error (XTTE2230)
// when keys from different sources have incompatible types.
func compareMergeKeyValues(a, b []mergeKeyValue, orders []mergeKeyOrder) (int, error) {
	for i, ord := range orders {
		if i >= len(a) || i >= len(b) {
			break
		}
		c, err := compareSingleMergeKey(a[i], b[i])
		if err != nil {
			return 0, err
		}
		if ord.desc {
			c = -c
		}
		if c != 0 {
			return c, nil
		}
	}
	return 0, nil
}

// compareSingleMergeKey compares two single merge key values.
// Returns XTTE2230 when the key types are incomparable.
func compareSingleMergeKey(a, b mergeKeyValue) (int, error) {
	// Numeric mode: use float64 comparison with NaN handling.
	if a.numeric || b.numeric {
		aNaN := a.isNaN
		bNaN := b.isNaN
		if aNaN && bNaN {
			return 0, nil
		}
		if aNaN {
			return -1, nil // NaN sorts before non-NaN in ascending
		}
		if bNaN {
			return 1, nil
		}
		if a.num < b.num {
			return -1, nil
		}
		if a.num > b.num {
			return 1, nil
		}
		return 0, nil
	}

	// If both have typed atomic values, use XPath value comparison.
	if a.atom.TypeName != "" && b.atom.TypeName != "" {
		lt, err := xpath3.ValueCompare(xpath3.TokenLt, a.atom, b.atom)
		if err == nil {
			if lt {
				return -1, nil
			}
			eq, err2 := xpath3.ValueCompare(xpath3.TokenEq, a.atom, b.atom)
			if err2 == nil && eq {
				return 0, nil
			}
			return 1, nil
		}
		// Types are incomparable — raise XTTE2230.
		return 0, dynamicError(errCodeXTTE2230, "merge keys are not comparable: %s vs %s", a.atom.TypeName, b.atom.TypeName)
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
		return -1, nil
	}
	if aStr > bStr {
		return 1, nil
	}
	return 0, nil
}

// applyNumericMergeKey converts a merge key value to numeric mode.
// When data-type="number", the key's string value is parsed as a number.
// Non-numeric values become NaN, and two NaN values are treated as equal
// during comparison (per XSLT sort specification).
func applyNumericMergeKey(mkv *mergeKeyValue) {
	mkv.numeric = true

	// If we have a typed atomic value, try to extract its numeric value.
	if mkv.atom.TypeName != "" {
		if mkv.atom.IsNumeric() {
			f := mkv.atom.ToFloat64()
			if math.IsNaN(f) {
				mkv.isNaN = true
				return
			}
			mkv.num = f
			return
		}
		// Non-numeric atomic value: get string representation and parse.
		s, err := xpath3.AtomicToString(mkv.atom)
		if err != nil {
			mkv.isNaN = true
			return
		}
		f := parseNumber(s)
		if math.IsNaN(f) {
			mkv.isNaN = true
			return
		}
		mkv.num = f
		return
	}

	// String fallback.
	f := parseNumber(mkv.str)
	if math.IsNaN(f) {
		mkv.isNaN = true
		return
	}
	mkv.num = f
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

		// Tag each mergeSourceItems with its source index for per-source
		// data-type resolution during sort verification.
		for i := range items {
			items[i].sourceIdx = srcIdx
		}

		allSources = append(allSources, items...)
	}

	// Determine sort orders and data-types from first source's key definitions.
	// Order and data-type can be AVTs, so evaluate them at runtime.
	keyDefs := inst.Sources[0].Keys
	orders := make([]mergeKeyOrder, len(keyDefs))
	for i, mk := range keyDefs {
		orderStr := mk.Order
		if mk.OrderAVT != nil {
			evaluated, err := mk.OrderAVT.evaluate(ctx, ec.contextNode)
			if err != nil {
				return err
			}
			orderStr = evaluated
		}
		orders[i] = mergeKeyOrder{desc: orderStr == "descending"}
	}

	// Resolve per-source data-types for each key level.
	// Each source uses its own data-type for sort verification (XTDE2210).
	// The first source's data-type is used for the n-way merge comparison.
	perSourceDataTypes := make([][]string, len(inst.Sources))
	for si, src := range inst.Sources {
		dts := make([]string, len(src.Keys))
		for k, mk := range src.Keys {
			dt := mk.DataType
			if mk.DataTypeAVT != nil {
				evaluated, err := mk.DataTypeAVT.evaluate(ctx, ec.contextNode)
				if err != nil {
					return err
				}
				dt = evaluated
			}
			dts[k] = dt
		}
		perSourceDataTypes[si] = dts
	}

	// XTDE2210: detect inconsistent data-type between sources.
	// Per XSLT spec, if data-type differs between corresponding merge-key
	// elements for different merge sources, the processor may raise XTDE2210.
	if len(perSourceDataTypes) > 1 {
		first := perSourceDataTypes[0]
		for si := 1; si < len(perSourceDataTypes); si++ {
			for k := range first {
				if k < len(perSourceDataTypes[si]) && first[k] != perSourceDataTypes[si][k] {
					return dynamicError(errCodeXTDE2210, "merge sources have inconsistent data-type for merge key %d: %q vs %q", k+1, first[k], perSourceDataTypes[si][k])
				}
			}
		}
	}

	// Sort or verify sort order for each source's items.
	// Each source uses its OWN data-type for sort verification.
	for si := range allSources {
		src := &allSources[si]
		if len(src.items) <= 1 {
			continue
		}

		// Build per-source keys with data-type applied for verification.
		srcDTs := perSourceDataTypes[src.sourceIdx]
		verifyKeys := make([][]mergeKeyValue, len(src.keys))
		for i := range src.keys {
			vk := make([]mergeKeyValue, len(src.keys[i]))
			copy(vk, src.keys[i])
			for k := range vk {
				if k < len(srcDTs) && srcDTs[k] == "number" {
					applyNumericMergeKey(&vk[k])
				}
			}
			verifyKeys[i] = vk
		}

		type indexedEntry struct {
			idx  int
			item xpath3.Item
			keys []mergeKeyValue
		}

		if src.sortBeforeMerge {
			// sort-before-merge="yes": sort the items by merge keys.
			entries := make([]indexedEntry, len(src.items))
			for i := range src.items {
				entries[i] = indexedEntry{idx: i, item: src.items[i], keys: verifyKeys[i]}
			}
			var sortErr error
			slices.SortStableFunc(entries, func(a, b indexedEntry) int {
				if sortErr != nil {
					return 0
				}
				c, err := compareMergeKeyValues(a.keys, b.keys, orders)
				if err != nil {
					sortErr = err
					return 0
				}
				return c
			})
			if sortErr != nil {
				return sortErr
			}
			// Rebuild items and keys in sorted order using a temporary copy
			// of the original keys to avoid in-place overwrite corruption.
			origKeys := make([][]mergeKeyValue, len(src.keys))
			copy(origKeys, src.keys)
			for i, e := range entries {
				src.items[i] = e.item
				src.keys[i] = origKeys[e.idx]
			}
		} else {
			// Default: verify items are already sorted (XTDE2210).
			// Skip verification when merge keys use collation attributes
			// (lang, collation, case-order) that we don't fully support,
			// since data may be validly sorted in a locale-specific order.
			srcKeyDefs := inst.Sources[src.sourceIdx].Keys
			hasCollation := false
			for _, mk := range srcKeyDefs {
				if mk.HasCollation {
					hasCollation = true
					break
				}
			}
			if !hasCollation {
				for i := 1; i < len(verifyKeys); i++ {
					cmp, cmpErr := compareMergeKeyValues(verifyKeys[i-1], verifyKeys[i], orders)
					if cmpErr != nil {
						return cmpErr
					}
					if cmp > 0 {
						return dynamicError(errCodeXTDE2210, "merge input is not sorted according to the declared merge key")
					}
				}
			}
		}
	}

	// Apply the first source's data-type for the n-way merge comparison.
	firstDTs := perSourceDataTypes[0]
	for si := range allSources {
		src := &allSources[si]
		for i := range src.keys {
			for k := range src.keys[i] {
				if k < len(firstDTs) && firstDTs[k] == "number" {
					applyNumericMergeKey(&src.keys[i][k])
				}
			}
		}
	}

	// 3. N-way merge: use cursors to walk through all sources.
	groups, mergeErr := ec.nWayMerge(allSources, orders)
	if mergeErr != nil {
		return mergeErr
	}

	// 4. Execute the action body for each group.
	// Register current-merge-group() and current-merge-key() as XSLT functions.
	// We temporarily add them to the cached function map.
	ec.xsltFunctions() // ensure cachedFns is initialized

	var currentMergeGroupAll xpath3.Sequence
	var currentMergeGroupByName map[string]xpath3.Sequence
	var currentMergeKeySeq xpath3.Sequence

	// Collect valid merge-source names for XTDE3490 validation.
	validSourceNames := make(map[string]struct{})
	for _, src := range inst.Sources {
		if src.Name != "" {
			validSourceNames[src.Name] = struct{}{}
		}
	}

	// Save previous merge functions BEFORE setting new ones (for nested xsl:merge).
	savedMergeGroup := ec.cachedFns["current-merge-group"]
	savedMergeKey := ec.cachedFns["current-merge-key"]
	defer func() {
		if savedMergeGroup != nil {
			ec.cachedFns["current-merge-group"] = savedMergeGroup
		} else {
			delete(ec.cachedFns, "current-merge-group")
		}
		if savedMergeKey != nil {
			ec.cachedFns["current-merge-key"] = savedMergeKey
		} else {
			delete(ec.cachedFns, "current-merge-key")
		}
	}()

	ec.cachedFns["current-merge-group"] = &xsltFunc{
		min: 0, max: 1,
		fn: func(_ context.Context, args []xpath3.Sequence) (xpath3.Sequence, error) {
			// XTDE3480: current-merge-group() is not available outside merge-action.
			if !ec.inMergeAction {
				return nil, dynamicError(errCodeXTDE3480, "current-merge-group() is not available outside the body of xsl:merge-action")
			}
			if len(args) > 0 && len(args[0]) > 0 {
				// current-merge-group('source-name')
				av, err := xpath3.AtomizeItem(args[0][0])
				if err != nil {
					return nil, dynamicError(errCodeXTDE3490, "current-merge-group(): cannot atomize argument: %v", err)
				}
				name, err := xpath3.AtomicToString(av)
				if err != nil {
					return nil, dynamicError(errCodeXTDE3490, "current-merge-group(): cannot convert argument to string: %v", err)
				}
				// XTDE3490: the argument must match a merge-source name.
				if _, ok := validSourceNames[name]; !ok {
					return nil, dynamicError(errCodeXTDE3490, "current-merge-group(%q): no xsl:merge-source with this name", name)
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
			// XTDE3510: current-merge-key() is not available outside merge-action.
			if !ec.inMergeAction {
				return nil, dynamicError(errCodeXTDE3510, "current-merge-key() is not available outside the body of xsl:merge-action")
			}
			return currentMergeKeySeq, nil
		},
	}

	// Save/restore context.
	savedCurrent := ec.currentNode
	savedContext := ec.contextNode
	savedPos := ec.position
	savedSize := ec.size
	savedInMerge := ec.inMergeAction
	ec.size = len(groups)
	defer func() {
		ec.currentNode = savedCurrent
		ec.contextNode = savedContext
		ec.position = savedPos
		ec.size = savedSize
		ec.inMergeAction = savedInMerge
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

		ec.inMergeAction = true
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

			// Load document from URI using the merge-source's effective base URI.
			doc, err := ec.loadMergeDocument(ctx, uri, src.BaseURI)
			if err != nil {
				return nil, err
			}
			if err := ec.prepareMergeSourceAccumulators(ctx, src, doc); err != nil {
				return nil, err
			}

			// Evaluate select against the document.
			items, err := ec.evaluateMergeSelect(ctx, src, doc)
			if err != nil {
				return nil, err
			}

			result = append(result, mergeSourceItems{
				name:            src.Name,
				items:           items,
				sortBeforeMerge: src.SortBeforeMerge,
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
				if err := ec.prepareMergeSourceAccumulators(ctx, src, contextNode); err != nil {
					return nil, err
				}
			}

			// Evaluate select against this item.
			items, err := ec.evaluateMergeSelectOnNode(ctx, src, contextNode)
			if err != nil {
				return nil, err
			}
			result = append(result, mergeSourceItems{
				name:            src.Name,
				items:           items,
				sortBeforeMerge: src.SortBeforeMerge,
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
			name:            src.Name,
			items:           selResult.Sequence(),
			sortBeforeMerge: src.SortBeforeMerge,
		})
	}

	return result, nil
}

func cloneAccumulatorSequence(seq xpath3.Sequence) xpath3.Sequence {
	if len(seq) == 0 {
		return nil
	}
	return append(xpath3.Sequence(nil), seq...)
}

func cloneAccumulatorSnapshot(state map[string]xpath3.Sequence) map[string]xpath3.Sequence {
	if len(state) == 0 {
		return nil
	}
	snapshot := make(map[string]xpath3.Sequence, len(state))
	for name, value := range state {
		snapshot[name] = cloneAccumulatorSequence(value)
	}
	return snapshot
}

func (ec *execContext) storeAccumulatorSnapshot(dst map[helium.Node]map[string]xpath3.Sequence, node helium.Node, state map[string]xpath3.Sequence) {
	if node == nil {
		return
	}
	dst[node] = cloneAccumulatorSnapshot(state)
}

func (ec *execContext) prepareMergeSourceAccumulators(ctx context.Context, src *MergeSource, node helium.Node) error {
	if len(src.UseAccumulators) == 0 || len(ec.stylesheet.accumulators) == 0 || node == nil {
		return nil
	}

	doc := node.OwnerDocument()
	if docNode, ok := node.(*helium.Document); ok {
		doc = docNode
	}
	if doc == nil {
		return nil
	}
	if ec.accumulatorBeforeByNode != nil {
		if _, ok := ec.accumulatorBeforeByNode[doc]; ok {
			return nil
		}
	}

	names := append([]string(nil), ec.stylesheet.accumulatorOrder...)

	return ec.computeAccumulatorStates(ctx, doc, names)
}

func (ec *execContext) prepareSourceDocumentAccumulators(ctx context.Context, inst *SourceDocumentInst, doc helium.Node) error {
	if len(inst.UseAccumulators) == 0 || len(ec.stylesheet.accumulators) == 0 || doc == nil {
		return nil
	}
	if ec.accumulatorBeforeByNode != nil {
		if _, ok := ec.accumulatorBeforeByNode[doc]; ok {
			return nil
		}
	}

	names := append([]string(nil), ec.stylesheet.accumulatorOrder...)
	return ec.computeAccumulatorStates(ctx, doc, names)
}

func (ec *execContext) computeAccumulatorStates(ctx context.Context, doc helium.Node, names []string) error {
	if ec.accumulatorBeforeByNode == nil {
		ec.accumulatorBeforeByNode = make(map[helium.Node]map[string]xpath3.Sequence)
	}
	if ec.accumulatorAfterByNode == nil {
		ec.accumulatorAfterByNode = make(map[helium.Node]map[string]xpath3.Sequence)
	}

	state := make(map[string]xpath3.Sequence, len(names))
	for _, name := range names {
		def, ok := ec.stylesheet.accumulators[name]
		if !ok {
			continue
		}

		switch {
		case def.Initial != nil:
			xpathCtx := ec.newXPathContext(doc)
			result, err := def.Initial.Evaluate(xpathCtx, doc)
			if err != nil {
				return err
			}
			checked, err := ec.checkAccumulatorType(def, result.Sequence())
			if err != nil {
				return err
			}
			state[name] = cloneAccumulatorSequence(checked)
		case len(def.InitialBody) > 0:
			seq, err := ec.evaluateBodyAsSequence(ctx, def.InitialBody)
			if err != nil {
				return err
			}
			checked, err := ec.checkAccumulatorType(def, seq)
			if err != nil {
				return err
			}
			state[name] = cloneAccumulatorSequence(checked)
		default:
			state[name] = xpath3.EmptySequence()
		}
	}

	savedState := ec.accumulatorState
	savedCurrent := ec.currentNode
	savedContext := ec.contextNode
	savedItem := ec.contextItem
	savedEval := ec.evaluatingAccumulator
	ec.accumulatorState = state
	ec.currentNode = doc
	ec.contextNode = doc
	ec.contextItem = nil
	ec.evaluatingAccumulator = true
	defer func() {
		ec.accumulatorState = savedState
		ec.currentNode = savedCurrent
		ec.contextNode = savedContext
		ec.contextItem = savedItem
		ec.evaluatingAccumulator = savedEval
	}()

	return ec.walkAccumulatorTree(ctx, doc, names)
}

func (ec *execContext) walkAccumulatorTree(ctx context.Context, node helium.Node, names []string) error {
	ec.storeAccumulatorSnapshot(ec.accumulatorBeforeByNode, node, ec.accumulatorState)

	if err := ec.applyAccumulatorPhase(ctx, node, names, "start"); err != nil {
		return err
	}

	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if err := ec.walkAccumulatorTree(ctx, child, names); err != nil {
			return err
		}
	}

	if err := ec.applyAccumulatorPhase(ctx, node, names, "end"); err != nil {
		return err
	}

	ec.storeAccumulatorSnapshot(ec.accumulatorAfterByNode, node, ec.accumulatorState)
	return nil
}

func (ec *execContext) applyAccumulatorPhase(ctx context.Context, node helium.Node, names []string, phase string) error {
	savedCurrent := ec.currentNode
	savedContext := ec.contextNode
	savedItem := ec.contextItem
	ec.currentNode = node
	ec.contextNode = node
	ec.contextItem = nil
	defer func() {
		ec.currentNode = savedCurrent
		ec.contextNode = savedContext
		ec.contextItem = savedItem
	}()

	for _, name := range names {
		def, ok := ec.stylesheet.accumulators[name]
		if !ok {
			continue
		}

		for _, rule := range def.Rules {
			if rule.Phase != phase || !rule.Match.matchPattern(ec, node) {
				continue
			}

			currentValue := ec.accumulatorState[name]
			if rule.New {
				currentValue = xpath3.EmptySequence()
			}

			ec.pushVarScope()
			ec.setVar("value", currentValue)

			var (
				newValue xpath3.Sequence
				err      error
			)
			switch {
			case rule.Select != nil:
				xpathCtx := ec.newXPathContext(node)
				result, evalErr := rule.Select.Evaluate(xpathCtx, node)
				if evalErr != nil {
					err = evalErr
				} else {
					newValue = cloneAccumulatorSequence(result.Sequence())
				}
			case len(rule.Body) > 0:
				newValue, err = ec.evaluateBodyAsSequence(ctx, rule.Body)
			default:
				newValue = xpath3.EmptySequence()
			}

			ec.popVarScope()
			if err != nil {
				return err
			}
			checked, err := ec.checkAccumulatorType(def, newValue)
			if err != nil {
				return err
			}
			ec.accumulatorState[name] = checked
		}
	}

	return nil
}

func (ec *execContext) checkAccumulatorType(def *AccumulatorDef, seq xpath3.Sequence) (xpath3.Sequence, error) {
	if def == nil || def.As == "" {
		return seq, nil
	}
	return checkSequenceType(seq, parseSequenceType(def.As), "XPTY0004", "accumulator "+def.Name, ec)
}

// loadMergeDocument loads an XML document from a URI, resolving it relative
// to the given effective base URI (which accounts for xml:base).
func (ec *execContext) loadMergeDocument(ctx context.Context, uri string, effectiveBaseURI string) (*helium.Document, error) {
	// Resolve URI relative to the effective base URI.
	effectiveBase := effectiveBaseURI
	if effectiveBase == "" {
		effectiveBase = ec.stylesheet.baseURI
	}
	resolvedURI := resolveAgainstBaseURI(uri, effectiveBase)

	// Check document cache.
	if doc, ok := ec.docCache[resolvedURI]; ok {
		return doc, nil
	}

	data, readErr := os.ReadFile(resolvedURI)
	if readErr != nil {
		return nil, dynamicError(errCodeFODC0002, "xsl:merge cannot load %q: %v", uri, readErr)
	}

	p := helium.NewParser()
	p.SetBaseURI(resolvedURI)
	doc, parseErr := p.Parse(ctx, data)
	if parseErr != nil {
		return nil, dynamicError(errCodeFODC0002, "xsl:merge cannot parse %q: %v", uri, parseErr)
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
	savedMergeKey := ec.evaluatingMergeKey
	defer func() {
		ec.contextNode = savedContext
		ec.currentNode = savedCurrent
		ec.contextItem = savedItem
		ec.evaluatingMergeKey = savedMergeKey
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

			ec.evaluatingMergeKey = true
			xpathCtx := ec.newXPathContext(node)
			result, err := mk.Select.Evaluate(xpathCtx, node)
			ec.evaluatingMergeKey = false
			if err != nil {
				return nil, err
			}

			seq := result.Sequence()
			// XTTE1020: merge key must evaluate to a single atomic value.
			if len(seq) > 1 {
				return nil, dynamicError(errCodeXTTE1020, "xsl:merge-key select expression must return a single atomic value, got %d items", len(seq))
			}
			// Extract the key value, preserving the atomic type.
			if len(seq) == 1 {
				if av, ok := seq[0].(xpath3.AtomicValue); ok {
					itemKeys[k] = mergeKeyValue{atom: av}
					continue
				}
				// Atomize node items to get typed atomic values.
				av, atomErr := xpath3.AtomizeItem(seq[0])
				if atomErr == nil {
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
// that share the same key values. Returns XTTE2230 if keys from different
// sources are not comparable.
func (ec *execContext) nWayMerge(sources []mergeSourceItems, orders []mergeKeyOrder) ([]mergeGroup, error) {
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
				cmp, err := compareMergeKeyValues(curKeys, minKeys, orders)
				if err != nil {
					return nil, err
				}
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
				cmp, err := compareMergeKeyValues(curKeys, minKeys, orders)
				if err != nil {
					return nil, err
				}
				if cmp != 0 {
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

		// Convert the full key tuple to a sequence for current-merge-key().
		if len(minKeys) > 0 {
			g.key = mergeKeyValuesToSequence(minKeys)
		}

		groups = append(groups, g)
	}

	return groups, nil
}

// mergeKeyValuesToSequence converts the merge key tuple to the XPath sequence
// exposed by current-merge-key().
func mergeKeyValuesToSequence(keys []mergeKeyValue) xpath3.Sequence {
	var seq xpath3.Sequence
	for _, mkv := range keys {
		seq = append(seq, mergeKeyValueToSequence(mkv)...)
	}
	return seq
}

func mergeKeyValueToSequence(mkv mergeKeyValue) xpath3.Sequence {
	if mkv.atom.TypeName != "" {
		return xpath3.SingleAtomic(mkv.atom)
	}
	if mkv.str != "" {
		return xpath3.SingleString(mkv.str)
	}
	return xpath3.EmptySequence()
}
