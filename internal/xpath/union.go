package xpath

import (
	helium "github.com/lestrrat-go/helium"
)

// UnionNodeSets returns the union of nodes[:split] and nodes[split:] in
// document order without duplicates: the node-set and order MergeNodeSets
// returns for the two operands, with the same documents registered in cache
// in the same order. It returns ErrNodeSetLimit if the union holds more than
// maxNodes nodes. The result may be nodes itself with its capacity clamped.
//
// MergeNodeSets resolves the sort key of every operand node, which costs a
// map lookup per node and indexes the whole document on first use. Two
// shortcuts avoid that work when the operands already decide the order:
//
//  1. When nodes is a subsequence of the attribute list of one element
//     followed by the owned-child list of that element (inElementOrder), it
//     is returned as is. This covers @*|node() and the other unions of
//     sorted attribute and child steps from one context element.
//  2. Otherwise, once the sort keys are resolved, two operands whose keys are
//     each strictly increasing are merged in one linear pass
//     (mergeIncreasingRuns), with no seen-map and no sort.
//
// Anything else takes the MergeNodeSets path over the resolved keys.
func UnionNodeSets(nodes []helium.Node, split int, cache *DocOrderCache, maxNodes int) ([]helium.Node, error) {
	if len(nodes) > 0 && inElementOrder(nodes) {
		// Index the first node as MergeNodeSets would, so the document
		// registration order stays the same; every node belongs to the
		// subtree of one element and so to one document.
		cache.reserveDocumentOf(nodes[0])
		if len(nodes) > maxNodes {
			return nil, ErrNodeSetLimit
		}
		return nodes[:len(nodes):len(nodes)], nil
	}
	a := nodes[:split:split]
	b := nodes[split:]
	keys := cache.indexSortKeys(a, b)
	if merged, ok, err := mergeIncreasingRuns(a, b, keys, maxNodes); ok {
		return merged, err
	}
	return mergeNodeSetsWithKeys(a, b, keys, maxNodes)
}

// inElementOrder reports whether nodes is a subsequence of the attributes of
// one element, in attribute-list order, followed by the owned children of
// that element, in child-list order. Such a sequence holds no duplicates and
// is in document order: the index (indexWalk) numbers an element, then its
// attributes in list order, then the subtree of each owned child in list
// order, and every walk that reaches an element numbers all of its
// attributes and owned children again together, so their relative order
// never changes. When the element is not indexed at all, no node of its
// subtree is, and MergeNodeSets keeps the input order, which is this order
// too. Namespace nodes are never in either list, so they fail the check.
func inElementOrder(nodes []helium.Node) bool {
	elem, ok := nodes[0].Parent().(*helium.Element)
	if !ok || elem == nil {
		return false
	}
	for _, n := range nodes {
		if n.Parent() != helium.Node(elem) {
			return false
		}
	}
	m := elementOrderMatcher{nodes: nodes}
	if nodes[0].Type() == helium.AttributeNode {
		elem.ForEachAttribute(m.matchAttribute)
	}
	if m.next == len(nodes) {
		return true
	}
	for child := range helium.Children(elem) {
		if child == nodes[m.next] {
			m.next++
			if m.next == len(nodes) {
				return true
			}
		}
	}
	return false
}

// elementOrderMatcher walks the attribute list of an element and counts how
// many leading nodes of nodes it meets in order.
type elementOrderMatcher struct {
	nodes []helium.Node
	next  int
}

// matchAttribute advances past nodes[next] when it is attr. It always
// continues the walk, since a later attribute may match the next node.
func (m *elementOrderMatcher) matchAttribute(attr *helium.Attribute) bool {
	if m.next < len(m.nodes) && m.nodes[m.next] == helium.Node(attr) {
		m.next++
	}
	return m.next < len(m.nodes)
}

// mergeIncreasingRuns merges a and b by their sort keys (keys holds the keys
// of a followed by those of b) when the keys of each operand are strictly
// increasing. Strictly increasing keys mean an operand is sorted and holds no
// duplicates: every indexed node owns its own position, and the nodes that
// share one (the namespace nodes of one element, and nodes no index
// describes) never form a strictly increasing run. The merge emits a node
// common to both operands once; two different nodes with equal keys stop the
// merge. The result is then strictly increasing, which is the one order
// MergeNodeSets can produce for the same node-set.
//
// ok is false when the merge does not apply; the caller then takes the
// general path. Once ok is true the result or ErrNodeSetLimit is final: every
// emitted node is distinct, so a result past maxNodes is past it in
// MergeNodeSets too.
func mergeIncreasingRuns(a, b []helium.Node, keys []sortKey, maxNodes int) ([]helium.Node, bool, error) {
	ka := keys[:len(a)]
	kb := keys[len(a):]
	if !strictlyIncreasing(ka) || !strictlyIncreasing(kb) {
		return nil, false, nil
	}
	result := make([]helium.Node, 0, boundedCap(len(a)+len(b), maxNodes))
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		switch {
		case ka[i].before(kb[j]):
			result = append(result, a[i])
			i++
		case kb[j].before(ka[i]):
			result = append(result, b[j])
			j++
		case a[i] == b[j]:
			result = append(result, a[i])
			i++
			j++
		default:
			return nil, false, nil
		}
		if len(result) > maxNodes {
			return nil, true, ErrNodeSetLimit
		}
	}
	if len(result)+(len(a)-i)+(len(b)-j) > maxNodes {
		return nil, true, ErrNodeSetLimit
	}
	result = append(result, a[i:]...)
	result = append(result, b[j:]...)
	return result, true, nil
}

// strictlyIncreasing reports whether every key sorts strictly after the one
// before it.
func strictlyIncreasing(keys []sortKey) bool {
	for i := 1; i < len(keys); i++ {
		if !keys[i-1].before(keys[i]) {
			return false
		}
	}
	return true
}
