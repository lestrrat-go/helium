package xslt3

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/sequence"
	"github.com/lestrrat-go/helium/xpath3"
)

func (ec *execContext) execIf(ctx context.Context, inst *ifInst) error {
	result, err := ec.evalXPath(ctx, inst.Test, ec.contextNode)
	if err != nil {
		return err
	}
	b, err := xpath3.EBV(result.Sequence())
	if err != nil {
		return err
	}
	if !b {
		return nil
	}
	if err := ec.executeSequenceConstructor(ctx, inst.Body); err != nil {
		return err
	}
	return nil
}

func (ec *execContext) execChoose(ctx context.Context, inst *chooseInst) error {
	// Apply default-collation from xsl:choose
	savedCollation := ec.defaultCollation
	if inst.DefaultCollation != "" {
		ec.defaultCollation = inst.DefaultCollation
	}
	defer func() { ec.defaultCollation = savedCollation }()

	for _, when := range inst.When {
		// Apply per-when xpath-default-namespace and default-collation
		savedNS := ec.xpathDefaultNS
		savedHas := ec.hasXPathDefaultNS
		savedWhenCollation := ec.defaultCollation
		if when.HasXPathDefaultNS {
			ec.xpathDefaultNS = when.XPathDefaultNS
			ec.hasXPathDefaultNS = true
		}
		if when.DefaultCollation != "" {
			ec.defaultCollation = when.DefaultCollation
		}
		var result *xpath3.Result
		var err error
		// Override namespace bindings with per-clause bindings when present
		if when.Namespaces != nil {
			eval := ec.xpathEvaluator(ctx).Namespaces(when.Namespaces).StrictPrefixes()
			result, err = eval.Evaluate(ec.xpathContext(ctx), when.Test, ec.contextNode)
		} else {
			result, err = ec.evalXPath(ctx, when.Test, ec.contextNode)
		}
		if err != nil {
			ec.xpathDefaultNS = savedNS
			ec.hasXPathDefaultNS = savedHas
			ec.defaultCollation = savedWhenCollation
			return err
		}
		b, err := xpath3.EBV(result.Sequence())
		if err != nil {
			ec.xpathDefaultNS = savedNS
			ec.hasXPathDefaultNS = savedHas
			ec.defaultCollation = savedWhenCollation
			return err
		}
		if b {
			if err := ec.executeSequenceConstructor(ctx, when.Body); err != nil {
				ec.xpathDefaultNS = savedNS
				ec.hasXPathDefaultNS = savedHas
				ec.defaultCollation = savedWhenCollation
				return err
			}
			ec.xpathDefaultNS = savedNS
			ec.hasXPathDefaultNS = savedHas
			ec.defaultCollation = savedWhenCollation
			return nil
		}
		ec.xpathDefaultNS = savedNS
		ec.hasXPathDefaultNS = savedHas
		ec.defaultCollation = savedWhenCollation
	}
	// otherwise
	savedNS := ec.xpathDefaultNS
	savedHas := ec.hasXPathDefaultNS
	if inst.HasOtherwiseXPNS {
		ec.xpathDefaultNS = inst.OtherwiseXPNS
		ec.hasXPathDefaultNS = true
	}
	if err := ec.executeSequenceConstructor(ctx, inst.Otherwise); err != nil {
		ec.xpathDefaultNS = savedNS
		ec.hasXPathDefaultNS = savedHas
		return err
	}
	ec.xpathDefaultNS = savedNS
	ec.hasXPathDefaultNS = savedHas
	return nil
}

func (ec *execContext) execForEach(ctx context.Context, inst *forEachInst) error {
	result, err := ec.evalXPath(ctx, inst.Select, ec.contextNode)
	if err != nil {
		return err
	}

	seq := result.Sequence()
	nodes, isNodes := xpath3.NodesFrom(seq)

	if len(inst.Sort) > 0 {
		if isNodes {
			nodes, err = sortNodes(ctx, ec, nodes, inst.Sort)
			if err != nil {
				return err
			}
		} else {
			seq, err = sortItems(ctx, ec, seq, inst.Sort)
			if err != nil {
				return err
			}
		}
	}

	savedCurrent := ec.currentNode
	savedContext := ec.contextNode
	savedItem := ec.contextItem
	savedPos := ec.position
	savedSize := ec.size
	savedTemplate := ec.currentTemplate
	// XSLT spec: inside xsl:for-each, the current template rule is absent.
	ec.setCurrentTemplate(nil)
	defer func() {
		ec.currentNode = savedCurrent
		ec.contextNode = savedContext
		ec.contextItem = savedItem
		ec.position = savedPos
		ec.size = savedSize
		ec.setCurrentTemplate(savedTemplate)
	}()

	if isNodes {
		ec.size = len(nodes)
		for i, node := range nodes {
			if err := ctx.Err(); err != nil {
				return err
			}
			ec.position = i + 1
			ec.currentNode = node
			ec.contextNode = node
			ec.contextItem = nil // clear atomic context when entering node context

			ec.pushVarScope()
			if err := ec.executeSequenceConstructor(ctx, inst.Body); err != nil {
				ec.popVarScope()
				return err
			}
			ec.popVarScope()
		}
	} else {
		ec.size = sequence.Len(seq)
		for i := range sequence.Len(seq) {
			if err := ctx.Err(); err != nil {
				return err
			}
			item := seq.Get(i)
			ec.position = i + 1
			if ni, ok := item.(xpath3.NodeItem); ok {
				ec.currentNode = ni.Node
				ec.contextNode = ni.Node
				ec.contextItem = nil
			} else {
				ec.contextItem = item
				ec.contextNode = nil
				ec.currentNode = nil
			}

			ec.pushVarScope()
			if err := ec.executeSequenceConstructor(ctx, inst.Body); err != nil {
				ec.popVarScope()
				return err
			}
			ec.popVarScope()
		}
	}

	return nil
}

func (ec *execContext) execForEachGroup(ctx context.Context, inst *forEachGroupInst) error {
	result, err := ec.evalXPath(ctx, inst.Select, ec.contextNode)
	if err != nil {
		return err
	}

	seq := result.Sequence()

	// Resolve collation key function: explicit collation attribute takes
	// precedence, then fall back to the default collation in scope.
	var collationKeyFn func(string) string
	if inst.Collation != nil {
		collationURI, collErr := inst.Collation.evaluate(ctx, ec.contextNode)
		if collErr != nil {
			return collErr
		}
		collationKeyFn, err = xpath3.ResolveCollationKeyFunc(collationURI)
		if err != nil {
			return err
		}
	} else if ec.defaultCollation != "" {
		collationKeyFn, err = xpath3.ResolveCollationKeyFunc(ec.defaultCollation)
		if err != nil {
			return err
		}
	}

	// Build groups based on the grouping mode
	var groups []fegGroup

	switch {
	case inst.GroupBy != nil:
		groups, err = ec.groupBy(ctx, seq, inst.GroupBy, inst.Composite, collationKeyFn)
	case inst.GroupAdjacent != nil:
		groups, err = ec.groupAdjacent(ctx, seq, inst.GroupAdjacent, inst.Composite, collationKeyFn)
	case inst.GroupStartingWith != nil:
		groups = ec.groupStartingWith(ctx, seq, inst.GroupStartingWith)
	case inst.GroupEndingWith != nil:
		groups = ec.groupEndingWith(ctx, seq, inst.GroupEndingWith)
	default:
		// No grouping attribute — treat entire sequence as one group
		groups = []fegGroup{{items: xpath3.ItemSlice(sequence.Materialize(seq))}}
	}
	if err != nil {
		return err
	}

	if len(inst.Sort) > 0 {
		groups, err = sortGroups(ctx, ec, groups, inst.Sort, inst.GroupBy != nil || inst.GroupAdjacent != nil)
		if err != nil {
			return err
		}
	}

	savedCurrent := ec.currentNode
	savedContext := ec.contextNode
	savedItem := ec.contextItem
	savedPos := ec.position
	savedSize := ec.size
	savedGroup := ec.currentGroup
	savedGroupKey := ec.currentGroupKey
	savedInGroupCtx := ec.inGroupContext
	savedGroupHasKey := ec.groupHasKey
	ec.size = len(groups)
	ec.inGroupContext = true
	ec.groupHasKey = inst.GroupBy != nil || inst.GroupAdjacent != nil
	defer func() {
		ec.currentNode = savedCurrent
		ec.contextNode = savedContext
		ec.contextItem = savedItem
		ec.position = savedPos
		ec.size = savedSize
		ec.currentGroup = savedGroup
		ec.currentGroupKey = savedGroupKey
		ec.inGroupContext = savedInGroupCtx
		ec.groupHasKey = savedGroupHasKey
	}()

	for i, g := range groups {
		if err := ctx.Err(); err != nil {
			return err
		}
		ec.position = i + 1
		ec.currentGroup = g.items
		ec.currentGroupKey = g.key

		// Context item is the first item of the group
		if len(g.items) > 0 {
			if ni, ok := g.items[0].(xpath3.NodeItem); ok {
				ec.currentNode = ni.Node
				ec.contextNode = ni.Node
				ec.contextItem = nil // clear atomic context when entering node context
			} else {
				ec.contextItem = g.items[0]
				ec.contextNode = nil
				ec.currentNode = nil
			}
		}

		ec.pushVarScope()
		if err := ec.executeSequenceConstructor(ctx, inst.Body); err != nil {
			ec.popVarScope()
			return err
		}
		ec.popVarScope()
	}
	return nil
}

func (ec *execContext) withSortGroupContext(groups []fegGroup, hasKey bool, fn func(i int, node helium.Node) error) error {
	savedCurrent := ec.currentNode
	savedContext := ec.contextNode
	savedItem := ec.contextItem
	savedPos := ec.position
	savedSize := ec.size
	savedGroup := ec.currentGroup
	savedGroupKey := ec.currentGroupKey
	savedInGroupCtx := ec.inGroupContext
	savedGroupHasKey := ec.groupHasKey
	defer func() {
		ec.currentNode = savedCurrent
		ec.contextNode = savedContext
		ec.contextItem = savedItem
		ec.position = savedPos
		ec.size = savedSize
		ec.currentGroup = savedGroup
		ec.currentGroupKey = savedGroupKey
		ec.inGroupContext = savedInGroupCtx
		ec.groupHasKey = savedGroupHasKey
	}()

	ec.size = len(groups)
	ec.inGroupContext = true
	ec.groupHasKey = hasKey

	for i, g := range groups {
		ec.position = i + 1
		ec.currentGroup = g.items
		ec.currentGroupKey = g.key
		ec.currentNode = nil
		ec.contextNode = nil
		ec.contextItem = nil

		var node helium.Node
		if len(g.items) > 0 {
			switch v := g.items[0].(type) {
			case xpath3.NodeItem:
				node = v.Node
				ec.currentNode = v.Node
				ec.contextNode = v.Node
			default:
				ec.contextItem = g.items[0]
			}
		}

		if err := fn(i, node); err != nil {
			return err
		}
	}
	return nil
}

type fegGroup struct {
	key   xpath3.Sequence
	items xpath3.ItemSlice
}

func groupLookupKey(item xpath3.Item, collationKeyFn func(string) string) (string, xpath3.Sequence) {
	av, err := xpath3.AtomizeItem(item)
	if err != nil {
		s := stringifyItem(item)
		if collationKeyFn != nil {
			return collationKeyFn(s), xpath3.ItemSlice{xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: s}}
		}
		return s, xpath3.ItemSlice{xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: s}}
	}

	s, err := xpath3.AtomicToString(av)
	if err != nil {
		s = ""
	}
	if collationKeyFn != nil {
		return collationKeyFn(s), xpath3.ItemSlice{av}
	}
	return canonicalKey(av), xpath3.ItemSlice{av}
}

// groupBy implements group-by: items are grouped by the string value of the
// group-by expression evaluated with each item as context. When composite is
// false and the expression returns a sequence of multiple values, the item is
// added to a group for each value. When composite is true, the entire sequence
// is treated as a single composite key.
func (ec *execContext) groupBy(ctx context.Context, seq xpath3.Sequence, groupByExpr *xpath3.Expression, composite bool, collationKeyFn func(string) string) ([]fegGroup, error) {
	type entry struct {
		key      string
		keyAtom  xpath3.AtomicValue // original atomic value for eq comparison
		keySeq   xpath3.Sequence
		items    xpath3.ItemSlice
		orderIdx int // index into order slice
	}
	var order []string
	groupMap := make(map[string]*entry)
	// numericGroups holds entries whose key is numeric so that
	// non-transitive cross-type comparisons (float/double/decimal)
	// can be resolved via linear scan using AtomicEquals.
	var numericGroups []*entry

	savedPos := ec.position
	savedSize := ec.size
	savedItem := ec.contextItem
	savedContext := ec.contextNode
	savedCurrent := ec.currentNode
	ec.size = sequence.Len(seq)
	defer func() {
		ec.position = savedPos
		ec.size = savedSize
		ec.contextItem = savedItem
		ec.contextNode = savedContext
		ec.currentNode = savedCurrent
	}()

	for i := range sequence.Len(seq) {
		item := seq.Get(i)
		ec.position = i + 1
		var node helium.Node
		if ni, ok := item.(xpath3.NodeItem); ok {
			node = ni.Node
			ec.contextNode = node
			ec.currentNode = node
			ec.contextItem = nil
		} else {
			ec.contextItem = item
			ec.contextNode = nil
			ec.currentNode = nil
		}
		result, err := ec.evalXPath(ctx, groupByExpr, node)
		if err != nil {
			return nil, err
		}

		resultSeq := result.Sequence()
		if resultSeq == nil || sequence.Len(resultSeq) == 0 {
			continue
		}

		if composite {
			// Composite: entire sequence is a single key
			keyVal := compositeKeyString(resultSeq)
			lookupKey := keyVal
			if collationKeyFn != nil {
				lookupKey = collationKeyFn(keyVal)
			}
			if e, ok := groupMap[lookupKey]; ok {
				e.items = append(e.items, item)
			} else {
				groupMap[lookupKey] = &entry{key: keyVal, keySeq: atomizeSequence(resultSeq), items: xpath3.ItemSlice{item}}
				order = append(order, lookupKey)
			}
		} else {
			// Non-composite: each value creates a separate group key.
			// Track which groups this item has already been added to,
			// since an item with duplicate key values (e.g., pop=5
			// and name-length=5) should appear only once in a group.
			addedToGroup := make(map[int]struct{})
			for keyItem := range sequence.Items(resultSeq) {
				av, atomErr := xpath3.AtomizeItem(keyItem)
				isNumeric := atomErr == nil && av.IsNumeric()

				if isNumeric && collationKeyFn == nil {
					// Numeric keys require value-based comparison
					// because of non-transitive equality across
					// float/double/decimal (XSLT erratum E25).
					matched := false
					for _, ng := range numericGroups {
						if xpath3.AtomicEquals(av, ng.keyAtom) {
							if _, already := addedToGroup[ng.orderIdx]; !already {
								ng.items = append(ng.items, item)
								addedToGroup[ng.orderIdx] = struct{}{}
							}
							matched = true
							break
						}
					}
					if !matched {
						idx := len(order)
						lookupKey := "N:" + strconv.Itoa(idx)
						e := &entry{keyAtom: av, keySeq: xpath3.ItemSlice{av}, items: xpath3.ItemSlice{item}, orderIdx: idx}
						groupMap[lookupKey] = e
						numericGroups = append(numericGroups, e)
						order = append(order, lookupKey)
						addedToGroup[idx] = struct{}{}
					}
				} else {
					lookupKey, keySeq := groupLookupKey(keyItem, collationKeyFn)
					// Use a sentinel based on order index for dedup.
					if e, ok := groupMap[lookupKey]; ok {
						if _, already := addedToGroup[e.orderIdx]; !already {
							e.items = append(e.items, item)
							addedToGroup[e.orderIdx] = struct{}{}
						}
					} else {
						idx := len(order)
						e := &entry{keySeq: keySeq, items: xpath3.ItemSlice{item}, orderIdx: idx}
						groupMap[lookupKey] = e
						order = append(order, lookupKey)
						addedToGroup[idx] = struct{}{}
					}
				}
			}
		}
	}

	groups := make([]fegGroup, len(order))
	for i, k := range order {
		e := groupMap[k]
		if e.keySeq != nil {
			groups[i] = fegGroup{key: e.keySeq, items: e.items}
		} else {
			groups[i] = fegGroup{items: e.items}
		}
	}
	return groups, nil
}

// groupAdjacent implements group-adjacent: consecutive items with equal
// grouping key values form a group. When composite is true, the key
// expression returns a sequence treated as a single composite key.
func (ec *execContext) groupAdjacent(ctx context.Context, seq xpath3.Sequence, adjExpr *xpath3.Expression, composite bool, collationKeyFn func(string) string) ([]fegGroup, error) {
	var groups []fegGroup
	var currentKey string
	var currentKeySeq xpath3.Sequence
	var currentItems xpath3.ItemSlice

	savedPos := ec.position
	savedSize := ec.size
	savedItem := ec.contextItem
	savedContext := ec.contextNode
	savedCurrent := ec.currentNode
	ec.size = sequence.Len(seq)
	defer func() {
		ec.position = savedPos
		ec.size = savedSize
		ec.contextItem = savedItem
		ec.contextNode = savedContext
		ec.currentNode = savedCurrent
	}()

	for i := range sequence.Len(seq) {
		item := seq.Get(i)
		ec.position = i + 1
		var node helium.Node
		if ni, ok := item.(xpath3.NodeItem); ok {
			node = ni.Node
			ec.contextNode = node
			ec.currentNode = node
			ec.contextItem = nil
		} else {
			ec.contextItem = item
			ec.contextNode = nil
			ec.currentNode = nil
		}
		result, err := ec.evalXPath(ctx, adjExpr, node)
		if err != nil {
			return nil, err
		}

		var keyVal string
		var keySeq xpath3.Sequence
		if composite {
			rSeq := result.Sequence()
			keyVal = compositeKeyString(rSeq)
			keySeq = atomizeSequence(rSeq)
		} else {
			resultSeq := result.Sequence()
			if resultSeq == nil || sequence.Len(resultSeq) == 0 {
				// XTTE1100: group-adjacent key must not be an empty sequence
				return nil, dynamicError(errCodeXTTE1100,
					"group-adjacent key is an empty sequence")
			}
			if sequence.Len(resultSeq) > 1 {
				// XTTE1100: group-adjacent key must be a single atomic value
				return nil, dynamicError(errCodeXTTE1100,
					"group-adjacent key has %d items (expected 1)", sequence.Len(resultSeq))
			}
			keyVal, keySeq = groupLookupKey(resultSeq.Get(0), nil)
		}

		lookupKey := keyVal
		if collationKeyFn != nil {
			lexical := stringifyResult(result)
			lookupKey = collationKeyFn(lexical)
		}

		if lookupKey == currentKey && len(currentItems) > 0 {
			currentItems = append(currentItems, item)
		} else {
			if len(currentItems) > 0 {
				var gKey xpath3.Sequence
				if currentKeySeq != nil {
					gKey = currentKeySeq
				} else {
					gKey = xpath3.ItemSlice{xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: currentKey}}
				}
				groups = append(groups, fegGroup{key: gKey, items: currentItems})
			}
			currentKey = lookupKey
			currentKeySeq = keySeq
			currentItems = xpath3.ItemSlice{item}
		}
	}
	if len(currentItems) > 0 {
		var gKey xpath3.Sequence
		if currentKeySeq != nil {
			gKey = currentKeySeq
		} else {
			gKey = xpath3.ItemSlice{xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: currentKey}}
		}
		groups = append(groups, fegGroup{key: gKey, items: currentItems})
	}
	return groups, nil
}

// compositeKeyString creates a canonical string representation of a composite
// key for use as a map key. Items are separated by a NUL byte to avoid
// collisions.
func compositeKeyString(seq xpath3.Sequence) string {
	parts := make([]string, sequence.Len(seq))
	for i := range sequence.Len(seq) {
		parts[i] = stringifyItem(seq.Get(i))
	}
	return strings.Join(parts, "\x00")
}

// groupStartingWith implements group-starting-with: a new group starts
// whenever an item matches the pattern.
func (ec *execContext) groupStartingWith(ctx context.Context, seq xpath3.Sequence, pat *pattern) []fegGroup {
	var groups []fegGroup
	var currentItems xpath3.ItemSlice
	for item := range sequence.Items(seq) {
		if pat.matchPatternItem(ctx, ec, item) && len(currentItems) > 0 {
			groups = append(groups, fegGroup{items: currentItems})
			currentItems = nil
		}
		currentItems = append(currentItems, item)
	}
	if len(currentItems) > 0 {
		groups = append(groups, fegGroup{items: currentItems})
	}
	return groups
}

// groupEndingWith implements group-ending-with: a group ends whenever
// an item matches the pattern.
func (ec *execContext) groupEndingWith(ctx context.Context, seq xpath3.Sequence, pat *pattern) []fegGroup {
	var groups []fegGroup
	var currentItems xpath3.ItemSlice
	for item := range sequence.Items(seq) {
		currentItems = append(currentItems, item)
		if pat.matchPatternItem(ctx, ec, item) {
			groups = append(groups, fegGroup{items: currentItems})
			currentItems = nil
		}
	}
	if len(currentItems) > 0 {
		groups = append(groups, fegGroup{items: currentItems})
	}
	return groups
}

func (ec *execContext) execVariable(ctx context.Context, inst *variableInst) error {
	var val xpath3.Sequence
	var evalErr error

	if inst.Select != nil {
		result, err := ec.evalXPath(ctx, inst.Select, ec.contextNode)
		if err != nil {
			// XSLT 3.0 §9.5: circular references are only errors when
			// the variable is actually used. Defer the error so that
			// unused variables with circular dependencies do not cause
			// the transformation to fail.
			if errors.Is(err, ErrCircularRef) {
				ec.setVarDeferred(inst.Name, err)
				return nil
			}
			return err
		}
		val = result.Sequence()
	} else if len(inst.Body) > 0 {
		ec.temporaryOutputDepth++
		if inst.As == "" {
			// Per XSLT spec: variable with content body (no select, no as)
			// produces a document node (temporary tree)
			val, evalErr = ec.evaluateBodyAsDocument(ctx, inst.Body)
		} else if strings.HasPrefix(inst.As, "document-node") {
			// document-node()* or document-node()+: evaluate as sequence
			// so that copy-of of multiple documents produces separate items.
			if strings.HasSuffix(inst.As, "*") || strings.HasSuffix(inst.As, "+") {
				val, evalErr = ec.evaluateBodyAsSequence(ctx, inst.Body)
			} else {
				// document-node() or document-node()?: wrap body in document node
				val, evalErr = ec.evaluateBodyAsDocument(ctx, inst.Body)
				// When the as type allows zero occurrences (e.g. document-node()?)
				// and the body produced an empty document (no children), return
				// an empty sequence instead of the empty document. This handles
				// xsl:where-populated discarding all content.
				if evalErr == nil && val != nil && sequence.Len(val) == 1 {
					if docItem, ok := val.Get(0).(xpath3.NodeItem); ok {
						if doc, ok := docItem.Node.(*helium.Document); ok && doc.FirstChild() == nil {
							if strings.HasSuffix(inst.As, "?") {
								val = nil
							}
						}
					}
				}
			}
		} else {
			// With as attribute: evaluate as sequence constructor,
			// keeping each node as a separate item
			val, evalErr = ec.evaluateBodyAsSequence(ctx, inst.Body)
		}
		ec.temporaryOutputDepth--
		if evalErr != nil {
			if errors.Is(evalErr, ErrCircularRef) {
				ec.setVarDeferred(inst.Name, evalErr)
				return nil
			}
			return evalErr
		}
	} else {
		// No select, no body (or empty body after whitespace stripping).
		// XSLT 3.0 §9.3: if as specifies a sequence type whose occurrence
		// indicator is ? or *, the effective value is an empty sequence.
		if inst.As != "" && (strings.HasSuffix(inst.As, "?") || strings.HasSuffix(inst.As, "*")) {
			val = nil
		} else {
			val = xpath3.SingleString("")
		}
	}

	// Type check against the declared as type
	if inst.As != "" {
		st := parseSequenceType(inst.As)
		checked, err := checkSequenceType(ctx, val, st, errCodeXTTE0570, "variable $"+inst.Name, ec)
		if err != nil {
			return err
		}
		val = checked
	}

	ec.setVar(inst.Name, val)
	return nil
}

func (ec *execContext) execParam(ctx context.Context, inst *paramInst) error {
	// Check if already set (by with-param)
	if _, ok := ec.localVars.lookup(inst.Name); ok {
		return nil
	}
	// Use default
	return ec.execVariable(ctx, &variableInst{
		Name:   inst.Name,
		Select: inst.Select,
		Body:   inst.Body,
		As:     inst.As,
	})
}
