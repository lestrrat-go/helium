package xslt3

import (
	"context"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

func (ec *execContext) execIf(ctx context.Context, inst *IfInst) error {
	xpathCtx := ec.newXPathContext(ec.contextNode)
	result, err := inst.Test.Evaluate(xpathCtx, ec.contextNode)
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

func (ec *execContext) execChoose(ctx context.Context, inst *ChooseInst) error {
	// Apply default-collation from xsl:choose
	savedCollation := ec.defaultCollation
	if inst.DefaultCollation != "" {
		ec.defaultCollation = inst.DefaultCollation
	}
	defer func() { ec.defaultCollation = savedCollation }()

	for _, when := range inst.When {
		// Apply per-when xpath-default-namespace
		savedNS := ec.xpathDefaultNS
		savedHas := ec.hasXPathDefaultNS
		if when.HasXPathDefaultNS {
			ec.xpathDefaultNS = when.XPathDefaultNS
			ec.hasXPathDefaultNS = true
		}
		xpathCtx := ec.newXPathContext(ec.contextNode)
		result, err := when.Test.Evaluate(xpathCtx, ec.contextNode)
		if err != nil {
			ec.xpathDefaultNS = savedNS
			ec.hasXPathDefaultNS = savedHas
			return err
		}
		b, err := xpath3.EBV(result.Sequence())
		if err != nil {
			ec.xpathDefaultNS = savedNS
			ec.hasXPathDefaultNS = savedHas
			return err
		}
		if b {
			if err := ec.executeSequenceConstructor(ctx, when.Body); err != nil {
				ec.xpathDefaultNS = savedNS
				ec.hasXPathDefaultNS = savedHas
				return err
			}
			ec.xpathDefaultNS = savedNS
			ec.hasXPathDefaultNS = savedHas
			return nil
		}
		ec.xpathDefaultNS = savedNS
		ec.hasXPathDefaultNS = savedHas
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

func (ec *execContext) execForEach(ctx context.Context, inst *ForEachInst) error {
	xpathCtx := ec.newXPathContext(ec.contextNode)
	result, err := inst.Select.Evaluate(xpathCtx, ec.contextNode)
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
	defer func() {
		ec.currentNode = savedCurrent
		ec.contextNode = savedContext
		ec.contextItem = savedItem
		ec.position = savedPos
		ec.size = savedSize
	}()

	if isNodes {
		ec.size = len(nodes)
		for i, node := range nodes {
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
		ec.size = len(seq)
		for i, item := range seq {
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

func (ec *execContext) execForEachGroup(ctx context.Context, inst *ForEachGroupInst) error {
	xpathCtx := ec.newXPathContext(ec.contextNode)
	result, err := inst.Select.Evaluate(xpathCtx, ec.contextNode)
	if err != nil {
		return err
	}

	seq := result.Sequence()

	// Resolve collation key function if collation attribute is present
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
	}

	// Build groups based on the grouping mode
	var groups []fegGroup

	switch {
	case inst.GroupBy != nil:
		groups, err = ec.groupBy(ctx, seq, inst.GroupBy, inst.Composite, collationKeyFn)
	case inst.GroupAdjacent != nil:
		groups, err = ec.groupAdjacent(ctx, seq, inst.GroupAdjacent, inst.Composite, collationKeyFn)
	case inst.GroupStartingWith != nil:
		groups = ec.groupStartingWith(seq, inst.GroupStartingWith)
	case inst.GroupEndingWith != nil:
		groups = ec.groupEndingWith(seq, inst.GroupEndingWith)
	default:
		// No grouping attribute — treat entire sequence as one group
		groups = []fegGroup{{items: seq}}
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
	items xpath3.Sequence
}

func groupLookupKey(item xpath3.Item, collationKeyFn func(string) string) (string, xpath3.Sequence) {
	av, err := xpath3.AtomizeItem(item)
	if err != nil {
		s := stringifyItem(item)
		if collationKeyFn != nil {
			return collationKeyFn(s), xpath3.Sequence{xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: s}}
		}
		return s, xpath3.Sequence{xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: s}}
	}

	s, err := xpath3.AtomicToString(av)
	if err != nil {
		s = ""
	}
	if collationKeyFn != nil {
		return collationKeyFn(s), xpath3.Sequence{av}
	}
	return canonicalKey(av), xpath3.Sequence{av}
}

// groupBy implements group-by: items are grouped by the string value of the
// group-by expression evaluated with each item as context. When composite is
// false and the expression returns a sequence of multiple values, the item is
// added to a group for each value. When composite is true, the entire sequence
// is treated as a single composite key.
func (ec *execContext) groupBy(_ context.Context, seq xpath3.Sequence, groupByExpr *xpath3.Expression, composite bool, collationKeyFn func(string) string) ([]fegGroup, error) {
	type entry struct {
		key    string
		keySeq xpath3.Sequence
		items  xpath3.Sequence
	}
	var order []string
	groupMap := make(map[string]*entry)

	savedPos := ec.position
	savedSize := ec.size
	savedItem := ec.contextItem
	savedContext := ec.contextNode
	savedCurrent := ec.currentNode
	ec.size = len(seq)
	defer func() {
		ec.position = savedPos
		ec.size = savedSize
		ec.contextItem = savedItem
		ec.contextNode = savedContext
		ec.currentNode = savedCurrent
	}()

	for i, item := range seq {
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
		xpathCtx := ec.newXPathContext(node)
		result, err := groupByExpr.Evaluate(xpathCtx, node)
		if err != nil {
			return nil, err
		}

		resultSeq := result.Sequence()
		if len(resultSeq) == 0 {
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
				groupMap[lookupKey] = &entry{key: keyVal, keySeq: atomizeSequence(resultSeq), items: xpath3.Sequence{item}}
				order = append(order, lookupKey)
			}
		} else {
			// Non-composite: each value creates a separate group key
			for _, keyItem := range resultSeq {
				lookupKey, keySeq := groupLookupKey(keyItem, collationKeyFn)
				if e, ok := groupMap[lookupKey]; ok {
					e.items = append(e.items, item)
				} else {
					groupMap[lookupKey] = &entry{keySeq: keySeq, items: xpath3.Sequence{item}}
					order = append(order, lookupKey)
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
	var currentItems xpath3.Sequence

	savedPos := ec.position
	savedSize := ec.size
	savedItem := ec.contextItem
	savedContext := ec.contextNode
	savedCurrent := ec.currentNode
	ec.size = len(seq)
	defer func() {
		ec.position = savedPos
		ec.size = savedSize
		ec.contextItem = savedItem
		ec.contextNode = savedContext
		ec.currentNode = savedCurrent
	}()

	for i, item := range seq {
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
		xpathCtx := ec.newXPathContext(node)
		result, err := adjExpr.Evaluate(xpathCtx, node)
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
			if len(resultSeq) == 0 {
				keyVal = ""
				keySeq = xpath3.Sequence{xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: ""}}
			} else {
				keyVal, keySeq = groupLookupKey(resultSeq[0], nil)
			}
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
					gKey = xpath3.Sequence{xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: currentKey}}
				}
				groups = append(groups, fegGroup{key: gKey, items: currentItems})
			}
			currentKey = lookupKey
			currentKeySeq = keySeq
			currentItems = xpath3.Sequence{item}
		}
	}
	if len(currentItems) > 0 {
		var gKey xpath3.Sequence
		if currentKeySeq != nil {
			gKey = currentKeySeq
		} else {
			gKey = xpath3.Sequence{xpath3.AtomicValue{TypeName: xpath3.TypeString, Value: currentKey}}
		}
		groups = append(groups, fegGroup{key: gKey, items: currentItems})
	}
	return groups, nil
}

// compositeKeyString creates a canonical string representation of a composite
// key for use as a map key. Items are separated by a NUL byte to avoid
// collisions.
func compositeKeyString(seq xpath3.Sequence) string {
	parts := make([]string, len(seq))
	for i, item := range seq {
		parts[i] = stringifyItem(item)
	}
	return strings.Join(parts, "\x00")
}

// groupStartingWith implements group-starting-with: a new group starts
// whenever an item matches the pattern.
func (ec *execContext) groupStartingWith(seq xpath3.Sequence, pat *Pattern) []fegGroup {
	var groups []fegGroup
	var currentItems xpath3.Sequence
	for _, item := range seq {
		if pat.matchPatternItem(ec, item) && len(currentItems) > 0 {
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
func (ec *execContext) groupEndingWith(seq xpath3.Sequence, pat *Pattern) []fegGroup {
	var groups []fegGroup
	var currentItems xpath3.Sequence
	for _, item := range seq {
		currentItems = append(currentItems, item)
		if pat.matchPatternItem(ec, item) {
			groups = append(groups, fegGroup{items: currentItems})
			currentItems = nil
		}
	}
	if len(currentItems) > 0 {
		groups = append(groups, fegGroup{items: currentItems})
	}
	return groups
}
