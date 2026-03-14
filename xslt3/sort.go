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

// keyed pairs an item with its pre-extracted sort key strings.
type keyed[T any] struct {
	item T
	keys []string
}

// resolvedSort holds the fully resolved per-level sort configuration.
// Built once before sorting; the comparison function captures it by value
// so no per-comparison AVT evaluation or mode switching is needed.
type resolvedSort struct {
	comparators []func(a, b string) int // one per sort level
}

// buildResolvedSort evaluates AVTs for order/data-type once and builds a
// comparator function per sort level.
func buildResolvedSort(ctx context.Context, ec *execContext, sortKeys []*SortKey) (resolvedSort, error) {
	comps := make([]func(a, b string) int, len(sortKeys))
	for i, sk := range sortKeys {
		order := "ascending"
		if sk.Order != nil {
			var err error
			order, err = sk.Order.evaluate(ctx, ec.contextNode)
			if err != nil {
				return resolvedSort{}, err
			}
		}
		dataType := "text"
		if sk.DataType != nil {
			var err error
			dataType, err = sk.DataType.evaluate(ctx, ec.contextNode)
			if err != nil {
				return resolvedSort{}, err
			}
		}

		desc := order == "descending"
		numeric := dataType == "number"
		// Capture resolved values; no further AVT evaluation during sort.
		comps[i] = func(a, b string) int {
			var c int
			if numeric {
				c = compareNumeric(a, b)
			} else {
				c = cmp.Compare(a, b)
			}
			if desc {
				c = -c
			}
			return c
		}
	}
	return resolvedSort{comparators: comps}, nil
}

// compare returns the ordering for two keyed items using the pre-built
// comparators. Returns 0 when all levels are equal (stable tie).
func (rs *resolvedSort) compare(a, b []string) int {
	for i, cmpFn := range rs.comparators {
		if c := cmpFn(a[i], b[i]); c != 0 {
			return c
		}
	}
	return 0
}

// extractKeys evaluates sort key expressions/body constructors for a single
// item and returns the string key per sort level.
// node may be nil when the context is an atomic item.
func extractKeys(ctx context.Context, ec *execContext, sortKeys []*SortKey, node helium.Node, autoTypes []bool) ([]string, error) {
	keys := make([]string, len(sortKeys))
	for i, sk := range sortKeys {
		if sk.Select != nil {
			xpathCtx := ec.newXPathContext(node)
			result, err := sk.Select.Evaluate(xpathCtx, node)
			if err != nil {
				return nil, dynamicError(errCodeXTDE0700, "sort key evaluation error: %v", err)
			}
			keys[i] = stringifyResult(result)
			if autoTypes != nil && !autoTypes[i] {
				seq := result.Sequence()
				if len(seq) == 1 {
					if av, ok := seq[0].(xpath3.AtomicValue); ok && av.IsNumeric() {
						autoTypes[i] = true
					}
				}
			}
			continue
		}
		if len(sk.Body) == 0 {
			continue
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
			return nil, dynamicError(errCodeXTDE0700, "sort key evaluation error: %v", err)
		}
		keys[i] = stringifySequence(val)
		if autoTypes != nil && !autoTypes[i] && len(val) == 1 {
			if av, ok := val[0].(xpath3.AtomicValue); ok && av.IsNumeric() {
				autoTypes[i] = true
			}
		}
	}
	return keys, nil
}

// sortNodes sorts nodes according to the given sort keys.
func sortNodes(ctx context.Context, ec *execContext, nodes []helium.Node, sortKeys []*SortKey) ([]helium.Node, error) {
	if len(sortKeys) == 0 || len(nodes) == 0 {
		return nodes, nil
	}

	// Extract keys for each node.
	autoTypes := makeAutoTypes(sortKeys)
	entries := make([]keyed[helium.Node], len(nodes))
	for i, node := range nodes {
		keys, err := extractKeys(ctx, ec, sortKeys, node, autoTypes)
		if err != nil {
			return nil, err
		}
		entries[i] = keyed[helium.Node]{item: node, keys: keys}
	}

	rs, err := buildResolvedSortWithAuto(ctx, ec, sortKeys, autoTypes)
	if err != nil {
		return nil, err
	}

	slices.SortStableFunc(entries, func(a, b keyed[helium.Node]) int {
		return rs.compare(a.keys, b.keys)
	})

	for i, e := range entries {
		nodes[i] = e.item
	}
	return nodes, nil
}

// sortItems sorts a sequence of items (including atomic values) according to sort keys.
func sortItems(ctx context.Context, ec *execContext, items xpath3.Sequence, sortKeys []*SortKey) (xpath3.Sequence, error) {
	if len(sortKeys) == 0 || len(items) == 0 {
		return items, nil
	}

	autoTypes := makeAutoTypes(sortKeys)
	entries := make([]keyed[xpath3.Item], len(items))

	savedItem := ec.contextItem
	defer func() { ec.contextItem = savedItem }()

	for i, item := range items {
		ec.contextItem = item
		var node helium.Node
		if ni, ok := item.(xpath3.NodeItem); ok {
			node = ni.Node
			ec.contextItem = nil
		}
		keys, err := extractKeys(ctx, ec, sortKeys, node, autoTypes)
		if err != nil {
			return nil, err
		}
		entries[i] = keyed[xpath3.Item]{item: item, keys: keys}
	}

	rs, err := buildResolvedSortWithAuto(ctx, ec, sortKeys, autoTypes)
	if err != nil {
		return nil, err
	}

	slices.SortStableFunc(entries, func(a, b keyed[xpath3.Item]) int {
		return rs.compare(a.keys, b.keys)
	})

	for i, e := range entries {
		items[i] = e.item
	}
	return items, nil
}

// makeAutoTypes returns a slice tracking which sort levels have auto-detected
// numeric type. nil entries in sortKeys[i].DataType mean "auto-detect".
func makeAutoTypes(sortKeys []*SortKey) []bool {
	types := make([]bool, len(sortKeys))
	for i, sk := range sortKeys {
		if sk.DataType != nil {
			types[i] = true // already explicit — skip auto-detection
		}
	}
	return types
}

// buildResolvedSortWithAuto is like buildResolvedSort but overrides data-type
// to "number" for levels where auto-detection found numeric keys.
func buildResolvedSortWithAuto(ctx context.Context, ec *execContext, sortKeys []*SortKey, autoTypes []bool) (resolvedSort, error) {
	rs, err := buildResolvedSort(ctx, ec, sortKeys)
	if err != nil {
		return rs, err
	}
	for i, sk := range sortKeys {
		if sk.DataType == nil && autoTypes[i] {
			// Override the comparator to use numeric comparison.
			desc := rs.comparators[i]("b", "a") < 0 // infer direction from existing comparator
			rs.comparators[i] = func(a, b string) int {
				c := compareNumeric(a, b)
				if desc {
					c = -c
				}
				return c
			}
		}
	}
	return rs, nil
}

func compareNumeric(a, b string) int {
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
