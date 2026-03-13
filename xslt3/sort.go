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
	Order     *AVT // "ascending" or "descending"
	DataType  *AVT // "text" or "number"
	CaseOrder *AVT // "upper-first" or "lower-first"
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
			xpathCtx := ec.newXPathContext(node)
			result, err := sk.Select.Evaluate(xpathCtx, node)
			if err != nil {
				return nil, dynamicError(errCodeXTDE0700, "sort key evaluation error: %v", err)
			}
			keys[i][ki] = stringifyResult(result)
			// Auto-detect numeric type when data-type not explicitly set
			if types[ki] == "text" && sk.DataType == nil {
				seq := result.Sequence()
				if len(seq) == 1 {
					if av, ok := seq[0].(xpath3.AtomicValue); ok && av.IsNumeric() {
						types[ki] = "number"
					}
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

func compareNumericStrings(a, b string) int {
	// Parse as float64 for comparison
	fa := parseNumber(a)
	fb := parseNumber(b)
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
