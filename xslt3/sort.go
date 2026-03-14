package xslt3

import (
	"context"
	"math"
	"sort"
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

type sortableNodes struct {
	nodes    []helium.Node
	keys     [][]string // keys[nodeIndex][sortKeyIndex]
	sortKeys []*SortKey
	orders   []string
	types    []string
	err      error
}

func (s *sortableNodes) Len() int { return len(s.nodes) }

func (s *sortableNodes) Swap(i, j int) {
	s.nodes[i], s.nodes[j] = s.nodes[j], s.nodes[i]
	s.keys[i], s.keys[j] = s.keys[j], s.keys[i]
}

func (s *sortableNodes) Less(i, j int) bool {
	for k := range s.sortKeys {
		ki := s.keys[i][k]
		kj := s.keys[j][k]

		var cmp int
		if s.types[k] == "number" {
			cmp = compareNumericStrings(ki, kj)
		} else {
			cmp = strings.Compare(ki, kj)
		}

		if cmp == 0 {
			continue
		}
		if s.orders[k] == "descending" {
			return cmp > 0
		}
		return cmp < 0
	}
	return false // stable: preserve document order
}

// sortNodes sorts nodes according to the given sort keys.
func sortNodes(ctx context.Context, ec *execContext, nodes []helium.Node, sortKeys []*SortKey) ([]helium.Node, error) {
	if len(sortKeys) == 0 || len(nodes) == 0 {
		return nodes, nil
	}

	// Compute sort key values for each node
	keys := make([][]string, len(nodes))
	orders := make([]string, len(sortKeys))
	types := make([]string, len(sortKeys))

	for ki, sk := range sortKeys {
		order := "ascending"
		if sk.Order != nil {
			var err error
			order, err = sk.Order.evaluate(ctx, ec.contextNode)
			if err != nil {
				return nil, err
			}
		}
		orders[ki] = order

		dataType := "text"
		if sk.DataType != nil {
			var err error
			dataType, err = sk.DataType.evaluate(ctx, ec.contextNode)
			if err != nil {
				return nil, err
			}
		}
		types[ki] = dataType
	}

	for i, node := range nodes {
		keys[i] = make([]string, len(sortKeys))
		for ki, sk := range sortKeys {
			var keyStr string
			var keySeq xpath3.Sequence
			if sk.Select != nil {
				xpathCtx := ec.newXPathContext(node)
				result, err := sk.Select.Evaluate(xpathCtx, node)
				if err != nil {
					return nil, dynamicError(errCodeXTDE0700, "sort key evaluation error: %v", err)
				}
				keyStr = stringifyResult(result)
				keySeq = result.Sequence()
			} else if len(sk.Body) > 0 {
				// Sequence constructor: evaluate body with context node
				savedCurrent := ec.currentNode
				savedContext := ec.contextNode
				ec.currentNode = node
				ec.contextNode = node
				val, err := ec.evaluateBody(ctx, sk.Body)
				ec.currentNode = savedCurrent
				ec.contextNode = savedContext
				if err != nil {
					return nil, dynamicError(errCodeXTDE0700, "sort key evaluation error: %v", err)
				}
				keyStr = stringifySequence(val)
				keySeq = val
			}
			keys[i][ki] = keyStr
			// Auto-detect numeric type when data-type not explicitly set
			if types[ki] == "text" && sk.DataType == nil && len(keySeq) == 1 {
				if av, ok := keySeq[0].(xpath3.AtomicValue); ok && av.IsNumeric() {
					types[ki] = "number"
				}
			}
		}
	}

	sn := &sortableNodes{
		nodes:    nodes,
		keys:     keys,
		sortKeys: sortKeys,
		orders:   orders,
		types:    types,
	}

	sort.Stable(sn)
	if sn.err != nil {
		return nil, sn.err
	}

	return sn.nodes, nil
}

// sortItems sorts a sequence of items (including atomic values) according to sort keys.
func sortItems(ctx context.Context, ec *execContext, items xpath3.Sequence, sortKeys []*SortKey) (xpath3.Sequence, error) {
	if len(sortKeys) == 0 || len(items) == 0 {
		return items, nil
	}

	keys := make([][]string, len(items))
	orders := make([]string, len(sortKeys))
	types := make([]string, len(sortKeys))

	for ki, sk := range sortKeys {
		order := "ascending"
		if sk.Order != nil {
			var err error
			order, err = sk.Order.evaluate(ctx, ec.contextNode)
			if err != nil {
				return nil, err
			}
		}
		orders[ki] = order

		dataType := "text"
		if sk.DataType != nil {
			var err error
			dataType, err = sk.DataType.evaluate(ctx, ec.contextNode)
			if err != nil {
				return nil, err
			}
		}
		types[ki] = dataType
	}

	savedItem := ec.contextItem
	defer func() { ec.contextItem = savedItem }()

	for i, item := range items {
		keys[i] = make([]string, len(sortKeys))
		// Set context item for sort key evaluation
		ec.contextItem = item
		var node helium.Node
		if ni, ok := item.(xpath3.NodeItem); ok {
			node = ni.Node
			ec.contextItem = nil
		}
		for ki, sk := range sortKeys {
			var keyStr string
			var keySeq xpath3.Sequence
			if sk.Select != nil {
				xpathCtx := ec.newXPathContext(node)
				result, err := sk.Select.Evaluate(xpathCtx, node)
				if err != nil {
					return nil, dynamicError(errCodeXTDE0700, "sort key evaluation error: %v", err)
				}
				keyStr = stringifyResult(result)
				keySeq = result.Sequence()
			} else if len(sk.Body) > 0 {
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
					return nil, dynamicError(errCodeXTDE0700, "sort key evaluation error: %v", err)
				}
				keyStr = stringifySequence(val)
				keySeq = val
			}
			keys[i][ki] = keyStr
			if types[ki] == "text" && sk.DataType == nil && len(keySeq) == 1 {
				if av, ok := keySeq[0].(xpath3.AtomicValue); ok && av.IsNumeric() {
					types[ki] = "number"
				}
			}
		}
	}

	si := &sortableItems{
		items:    items,
		keys:     keys,
		sortKeys: sortKeys,
		orders:   orders,
		types:    types,
	}
	sort.Stable(si)
	return si.items, nil
}

type sortableItems struct {
	items    xpath3.Sequence
	keys     [][]string
	sortKeys []*SortKey
	orders   []string
	types    []string
}

func (s *sortableItems) Len() int { return len(s.items) }

func (s *sortableItems) Swap(i, j int) {
	s.items[i], s.items[j] = s.items[j], s.items[i]
	s.keys[i], s.keys[j] = s.keys[j], s.keys[i]
}

func (s *sortableItems) Less(i, j int) bool {
	for k := range s.sortKeys {
		ki := s.keys[i][k]
		kj := s.keys[j][k]

		var cmp int
		if s.types[k] == "number" {
			cmp = compareNumericStrings(ki, kj)
		} else {
			cmp = strings.Compare(ki, kj)
		}

		if cmp == 0 {
			continue
		}
		if s.orders[k] == "descending" {
			return cmp > 0
		}
		return cmp < 0
	}
	return false
}

func compareNumericStrings(a, b string) int {
	// Parse as float64 for comparison.
	// NaN (non-numeric) values sort before all real numbers.
	fa := parseNumber(a)
	fb := parseNumber(b)
	aNaN := math.IsNaN(fa)
	bNaN := math.IsNaN(fb)
	if aNaN && bNaN {
		return 0
	}
	if aNaN {
		return -1
	}
	if bNaN {
		return 1
	}
	if fa < fb {
		return -1
	}
	if fa > fb {
		return 1
	}
	return 0
}

func parseNumber(s string) float64 {
	s = strings.TrimSpace(s)
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return math.NaN()
	}
	return f
}
