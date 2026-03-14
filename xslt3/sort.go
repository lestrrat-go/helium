package xslt3

import (
	"cmp"
	"context"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

// SortKey is a compiled xsl:sort specification.
type SortKey struct {
	Select    *xpath3.Expression
	Body      []Instruction // sequence constructor (when select is absent)
	Order     *AVT          // "ascending" or "descending"
	DataType  *AVT          // "text" or "number"
	CaseOrder *AVT          // "upper-first" or "lower-first"
	Lang      *AVT
}

// sortMode determines how a sort level compares keys.
type sortMode uint8

const (
	sortModeText   sortMode = iota
	sortModeNumber
)

// dataTypeMode tracks whether a sort level's data type is explicit or auto-detected.
type dataTypeMode uint8

const (
	dataTypeAuto   dataTypeMode = iota // no explicit data-type attr; detect from first numeric result
	dataTypeText                       // explicit data-type="text"
	dataTypeNumber                     // explicit data-type="number"
)

// sortValueKind identifies the type of a pre-extracted sort key.
type sortValueKind uint8

const (
	sortValueText   sortValueKind = iota
	sortValueNumber
	sortValueNaN
)

// sortValue is a pre-extracted, typed sort key. Numeric values are parsed
// once at extraction time; no string→float conversion happens during comparison.
type sortValue struct {
	kind sortValueKind
	str  string
	num  float64
}

// resolvedLevel holds the fully resolved configuration for one sort level.
type resolvedLevel struct {
	mode sortMode
	desc bool
}

// resolvedSort holds the fully resolved per-level sort configuration.
// Built once before sorting; no per-comparison AVT evaluation needed.
type resolvedSort struct {
	levels []resolvedLevel
}

// --- Single-key entry types ---

// keyedNode1 pairs a node with a single inline sort key and original index.
type keyedNode1 struct {
	item  helium.Node
	key   sortValue
	index int
}

// keyedItem1 pairs an item with a single inline sort key and original index.
type keyedItem1 struct {
	item  xpath3.Item
	key   sortValue
	index int
}

// --- Multi-key entry types ---

// keyedNode pairs a node with its pre-extracted typed sort keys and original index.
type keyedNode struct {
	item  helium.Node
	keys  []sortValue
	index int
}

// keyedItem pairs an item with its pre-extracted typed sort keys and original index.
type keyedItem struct {
	item  xpath3.Item
	keys  []sortValue
	index int
}

// --- Sort level resolution ---

// buildResolvedSort evaluates AVTs for order once and builds a resolvedLevel per sort level.
func buildResolvedSort(ctx context.Context, ec *execContext, sortKeys []*SortKey) (resolvedSort, error) {
	levels := make([]resolvedLevel, len(sortKeys))
	for i, sk := range sortKeys {
		order := "ascending"
		if sk.Order != nil {
			var err error
			order, err = sk.Order.evaluate(ctx, ec.contextNode)
			if err != nil {
				return resolvedSort{}, err
			}
		}
		levels[i] = resolvedLevel{
			desc: order == "descending",
		}
	}
	return resolvedSort{levels: levels}, nil
}

func resolveLevel1(ctx context.Context, ec *execContext, sk *SortKey) (resolvedLevel, error) {
	order := "ascending"
	if sk.Order != nil {
		var err error
		order, err = sk.Order.evaluate(ctx, ec.contextNode)
		if err != nil {
			return resolvedLevel{}, err
		}
	}
	return resolvedLevel{desc: order == "descending"}, nil
}

// --- Comparators ---

func compareSortValues(a, b sortValue, level resolvedLevel) int {
	var c int
	switch level.mode {
	case sortModeNumber:
		c = compareFloat64(a, b)
	default:
		c = cmp.Compare(a.str, b.str)
	}
	if level.desc {
		c = -c
	}
	return c
}

func compareFloat64(a, b sortValue) int {
	aNaN := a.kind == sortValueNaN
	bNaN := b.kind == sortValueNaN
	if aNaN && bNaN {
		return 0
	}
	if aNaN {
		return -1
	}
	if bNaN {
		return 1
	}
	if a.num < b.num {
		return -1
	}
	if a.num > b.num {
		return 1
	}
	return 0
}

func (rs *resolvedSort) compareKeys(aKeys, bKeys []sortValue, aIdx, bIdx int) int {
	for i, level := range rs.levels {
		if c := compareSortValues(aKeys[i], bKeys[i], level); c != 0 {
			return c
		}
	}
	return cmp.Compare(aIdx, bIdx)
}

// --- Key evaluation ---

// evaluateSortKey evaluates a single sort key for one item and returns its typed value.
// Uses EvaluateReuse when evalState is non-nil to avoid per-item evalContext allocation.
func evaluateSortKey(ctx context.Context, ec *execContext, sk *SortKey, node helium.Node, dtMode *dataTypeMode, evalState *xpath3.EvalState) (sortValue, error) {
	if sk.Select != nil {
		var result xpath3.Result
		var err error
		if evalState != nil {
			if ec.contextItem != nil {
				evalState.SetContextItem(ec.contextItem)
			}
			result, err = sk.Select.EvaluateReuse(evalState, node)
		} else {
			xpathCtx := ec.newXPathContext(node)
			r, e := sk.Select.Evaluate(xpathCtx, node)
			if e != nil {
				return sortValue{}, dynamicError(errCodeXTDE0700, "sort key evaluation error: %v", e)
			}
			result = *r
			err = nil
		}
		if err != nil {
			return sortValue{}, dynamicError(errCodeXTDE0700, "sort key evaluation error: %v", err)
		}

		seq := result.Sequence()

		if *dtMode == dataTypeNumber {
			return extractNumericValueFromResult(seq, result), nil
		}

		sv := sortValue{kind: sortValueText, str: result.StringValue()}
		if *dtMode == dataTypeAuto && len(seq) == 1 {
			if av, ok := seq[0].(xpath3.AtomicValue); ok && av.IsNumeric() {
				*dtMode = dataTypeNumber
			}
		}
		return sv, nil
	}

	if len(sk.Body) == 0 {
		if *dtMode == dataTypeNumber {
			return sortValue{kind: sortValueNaN}, nil
		}
		return sortValue{}, nil
	}

	savedCurrent := ec.currentNode
	savedContext := ec.contextNode
	if node != nil {
		ec.currentNode = node
		ec.contextNode = node
	}
	val, err := ec.evaluateBody(ctx, sk.Body)
	ec.currentNode = savedCurrent
	ec.contextNode = savedContext
	if err != nil {
		return sortValue{}, dynamicError(errCodeXTDE0700, "sort key evaluation error: %v", err)
	}

	if *dtMode == dataTypeNumber {
		return extractNumericValueFromSeq(val), nil
	}

	sv := sortValue{kind: sortValueText, str: stringifySequence(val)}
	if *dtMode == dataTypeAuto && len(val) == 1 {
		if av, ok := val[0].(xpath3.AtomicValue); ok && av.IsNumeric() {
			*dtMode = dataTypeNumber
		}
	}
	return sv, nil
}

// extractSortValues evaluates all sort keys for one item, returning a slice.
func extractSortValues(ctx context.Context, ec *execContext, sortKeys []*SortKey, node helium.Node, dtModes []dataTypeMode, evalState *xpath3.EvalState) ([]sortValue, error) {
	keys := make([]sortValue, len(sortKeys))
	for i, sk := range sortKeys {
		sv, err := evaluateSortKey(ctx, ec, sk, node, &dtModes[i], evalState)
		if err != nil {
			return nil, err
		}
		keys[i] = sv
	}
	return keys, nil
}

// --- Numeric extraction helpers ---

func extractNumericValueFromResult(seq xpath3.Sequence, result xpath3.Result) sortValue {
	if len(seq) == 1 {
		if av, ok := seq[0].(xpath3.AtomicValue); ok && av.IsNumeric() {
			f := av.ToFloat64()
			if math.IsNaN(f) {
				return sortValue{kind: sortValueNaN}
			}
			return sortValue{kind: sortValueNumber, num: f}
		}
	}
	return parseToNumericSortValue(result.StringValue())
}

func extractNumericValueFromSeq(seq xpath3.Sequence) sortValue {
	if len(seq) == 1 {
		if av, ok := seq[0].(xpath3.AtomicValue); ok && av.IsNumeric() {
			f := av.ToFloat64()
			if math.IsNaN(f) {
				return sortValue{kind: sortValueNaN}
			}
			return sortValue{kind: sortValueNumber, num: f}
		}
	}
	return parseToNumericSortValue(stringifySequence(seq))
}

func parseToNumericSortValue(s string) sortValue {
	f := parseNumber(s)
	if math.IsNaN(f) {
		return sortValue{kind: sortValueNaN}
	}
	return sortValue{kind: sortValueNumber, num: f}
}

// --- Data type mode resolution ---

func initDataTypeModes(ctx context.Context, ec *execContext, sortKeys []*SortKey) ([]dataTypeMode, error) {
	modes := make([]dataTypeMode, len(sortKeys))
	for i, sk := range sortKeys {
		if sk.DataType == nil {
			continue
		}
		dt, err := sk.DataType.evaluate(ctx, ec.contextNode)
		if err != nil {
			return nil, err
		}
		if dt == "number" {
			modes[i] = dataTypeNumber
		} else {
			modes[i] = dataTypeText
		}
	}
	return modes, nil
}

func initDataTypeMode1(ctx context.Context, ec *execContext, sk *SortKey) (dataTypeMode, error) {
	if sk.DataType == nil {
		return dataTypeAuto, nil
	}
	dt, err := sk.DataType.evaluate(ctx, ec.contextNode)
	if err != nil {
		return dataTypeAuto, err
	}
	if dt == "number" {
		return dataTypeNumber, nil
	}
	return dataTypeText, nil
}

// --- Finalization (auto-detect conversion) ---

func finalizeLevels(rs *resolvedSort, dtModes []dataTypeMode, entries interface{ convertAutoNumeric(level int) }) {
	for i, m := range dtModes {
		switch m {
		case dataTypeNumber:
			rs.levels[i].mode = sortModeNumber
			entries.convertAutoNumeric(i)
		default:
			rs.levels[i].mode = sortModeText
		}
	}
}

type keyedNodes []keyedNode

func (kn keyedNodes) convertAutoNumeric(level int) {
	for j := range kn {
		sv := &kn[j].keys[level]
		if sv.kind == sortValueText {
			*sv = parseToNumericSortValue(sv.str)
		}
	}
}

type keyedItems []keyedItem

func (ki keyedItems) convertAutoNumeric(level int) {
	for j := range ki {
		sv := &ki[j].keys[level]
		if sv.kind == sortValueText {
			*sv = parseToNumericSortValue(sv.str)
		}
	}
}

func finalizeLevel1Nodes(level *resolvedLevel, dtMode dataTypeMode, entries []keyedNode1) {
	switch dtMode {
	case dataTypeNumber:
		level.mode = sortModeNumber
		for i := range entries {
			if entries[i].key.kind == sortValueText {
				entries[i].key = parseToNumericSortValue(entries[i].key.str)
			}
		}
	default:
		level.mode = sortModeText
	}
}

func finalizeLevel1Items(level *resolvedLevel, dtMode dataTypeMode, entries []keyedItem1) {
	switch dtMode {
	case dataTypeNumber:
		level.mode = sortModeNumber
		for i := range entries {
			if entries[i].key.kind == sortValueText {
				entries[i].key = parseToNumericSortValue(entries[i].key.str)
			}
		}
	default:
		level.mode = sortModeText
	}
}

// --- Public dispatch ---

func sortNodes(ctx context.Context, ec *execContext, nodes []helium.Node, sortKeys []*SortKey) ([]helium.Node, error) {
	if len(sortKeys) == 0 || len(nodes) == 0 {
		return nodes, nil
	}
	if len(sortKeys) == 1 {
		return sortNodes1(ctx, ec, nodes, sortKeys[0])
	}
	return sortNodesN(ctx, ec, nodes, sortKeys)
}

func sortItems(ctx context.Context, ec *execContext, items xpath3.Sequence, sortKeys []*SortKey) (xpath3.Sequence, error) {
	if len(sortKeys) == 0 || len(items) == 0 {
		return items, nil
	}
	if len(sortKeys) == 1 {
		return sortItems1(ctx, ec, items, sortKeys[0])
	}
	return sortItemsN(ctx, ec, items, sortKeys)
}

// --- Single-key sort paths ---

func sortNodes1(ctx context.Context, ec *execContext, nodes []helium.Node, sk *SortKey) ([]helium.Node, error) {
	dtMode, err := initDataTypeMode1(ctx, ec, sk)
	if err != nil {
		return nil, err
	}

	evalState := ec.sortXPathEvalState()
	entries := make([]keyedNode1, len(nodes))
	for i, node := range nodes {
		sv, err := evaluateSortKey(ctx, ec, sk, node, &dtMode, evalState)
		if err != nil {
			return nil, err
		}
		entries[i] = keyedNode1{item: node, key: sv, index: i}
	}

	level, err := resolveLevel1(ctx, ec, sk)
	if err != nil {
		return nil, err
	}
	finalizeLevel1Nodes(&level, dtMode, entries)

	slices.SortFunc(entries, func(a, b keyedNode1) int {
		if c := compareSortValues(a.key, b.key, level); c != 0 {
			return c
		}
		return cmp.Compare(a.index, b.index)
	})

	for i, e := range entries {
		nodes[i] = e.item
	}
	return nodes, nil
}

func sortItems1(ctx context.Context, ec *execContext, items xpath3.Sequence, sk *SortKey) (xpath3.Sequence, error) {
	dtMode, err := initDataTypeMode1(ctx, ec, sk)
	if err != nil {
		return nil, err
	}

	evalState := ec.sortXPathEvalState()
	entries := make([]keyedItem1, len(items))

	savedItem := ec.contextItem
	defer func() { ec.contextItem = savedItem }()

	for i, item := range items {
		ec.contextItem = item
		var node helium.Node
		if ni, ok := item.(xpath3.NodeItem); ok {
			node = ni.Node
			ec.contextItem = nil
		}
		sv, err := evaluateSortKey(ctx, ec, sk, node, &dtMode, evalState)
		if err != nil {
			return nil, err
		}
		entries[i] = keyedItem1{item: item, key: sv, index: i}
	}

	level, err := resolveLevel1(ctx, ec, sk)
	if err != nil {
		return nil, err
	}
	finalizeLevel1Items(&level, dtMode, entries)

	slices.SortFunc(entries, func(a, b keyedItem1) int {
		if c := compareSortValues(a.key, b.key, level); c != 0 {
			return c
		}
		return cmp.Compare(a.index, b.index)
	})

	for i, e := range entries {
		items[i] = e.item
	}
	return items, nil
}

// --- Multi-key sort paths ---

func sortNodesN(ctx context.Context, ec *execContext, nodes []helium.Node, sortKeys []*SortKey) ([]helium.Node, error) {
	dtModes, err := initDataTypeModes(ctx, ec, sortKeys)
	if err != nil {
		return nil, err
	}

	evalState := ec.sortXPathEvalState()
	entries := make(keyedNodes, len(nodes))
	for i, node := range nodes {
		keys, err := extractSortValues(ctx, ec, sortKeys, node, dtModes, evalState)
		if err != nil {
			return nil, err
		}
		entries[i] = keyedNode{item: node, keys: keys, index: i}
	}

	rs, err := buildResolvedSort(ctx, ec, sortKeys)
	if err != nil {
		return nil, err
	}
	finalizeLevels(&rs, dtModes, entries)

	slices.SortFunc(entries, func(a, b keyedNode) int {
		return rs.compareKeys(a.keys, b.keys, a.index, b.index)
	})

	for i, e := range entries {
		nodes[i] = e.item
	}
	return nodes, nil
}

func sortItemsN(ctx context.Context, ec *execContext, items xpath3.Sequence, sortKeys []*SortKey) (xpath3.Sequence, error) {
	dtModes, err := initDataTypeModes(ctx, ec, sortKeys)
	if err != nil {
		return nil, err
	}

	evalState := ec.sortXPathEvalState()
	entries := make(keyedItems, len(items))

	savedItem := ec.contextItem
	defer func() { ec.contextItem = savedItem }()

	for i, item := range items {
		ec.contextItem = item
		var node helium.Node
		if ni, ok := item.(xpath3.NodeItem); ok {
			node = ni.Node
			ec.contextItem = nil
		}
		keys, err := extractSortValues(ctx, ec, sortKeys, node, dtModes, evalState)
		if err != nil {
			return nil, err
		}
		entries[i] = keyedItem{item: item, keys: keys, index: i}
	}

	rs, err := buildResolvedSort(ctx, ec, sortKeys)
	if err != nil {
		return nil, err
	}
	finalizeLevels(&rs, dtModes, entries)

	slices.SortFunc(entries, func(a, b keyedItem) int {
		return rs.compareKeys(a.keys, b.keys, a.index, b.index)
	})

	for i, e := range entries {
		items[i] = e.item
	}
	return items, nil
}

func parseNumber(s string) float64 {
	s = strings.TrimSpace(s)
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return math.NaN()
	}
	return f
}
