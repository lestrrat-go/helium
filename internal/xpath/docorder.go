package xpath

import (
	"sort"

	helium "github.com/lestrrat-go/helium"
)

type docIndex struct {
	order     int
	positions map[helium.Node]int
}

// DocOrderCache caches document-order positions for nodes grouped by
// document root. Built lazily on first use and shared across an evaluation.
type DocOrderCache struct {
	documents map[helium.Node]docIndex
}

// Position returns the document-order position of a node, or -1 if unknown.
//
// Namespace nodes are virtual (NamespaceNodeWrapper is created fresh on each
// namespace axis traversal) so they cannot be indexed during BuildFrom.
// They receive position parent_pos + 1, which is a dedicated slot between the
// parent element and its first attribute/child (indexWalk uses stride 2).
// SliceStable preserves input order for equal positions, keeping
// same-parent namespace nodes in their traversal order. Namespace nodes
// are deduplicated by {parent, prefix} in DeduplicateNodes/MergeNodeSets,
// so duplicates from different union branches are already eliminated.
func (c *DocOrderCache) Position(n helium.Node) int {
	if n.Type() == helium.NamespaceNode {
		parent := n.Parent()
		if parent == nil {
			return -1
		}
		parentPos := c.Position(parent)
		if parentPos < 0 {
			return -1
		}
		// Namespace nodes sort after their parent element but before
		// attributes and children. The +1 offset lands in the gap
		// left by stride-2 indexing in indexWalk.
		return parentPos + 1
	}
	root := DocumentRoot(n)
	index, ok := c.documents[root]
	if !ok {
		return -1
	}
	pos, ok := index.positions[n]
	if !ok {
		return -1
	}
	return pos
}

// BuildFrom populates the cache by walking the tree rooted at root.
// No-op if that root is already indexed.
func (c *DocOrderCache) BuildFrom(root helium.Node) {
	root = DocumentRoot(root)
	if c.documents == nil {
		c.documents = make(map[helium.Node]docIndex)
	}
	if _, ok := c.documents[root]; ok {
		return
	}
	positions := make(map[helium.Node]int)
	pos := 0
	c.indexWalk(root, positions, &pos)
	c.documents[root] = docIndex{
		order:     len(c.documents),
		positions: positions,
	}
}

func (c *DocOrderCache) indexWalk(cur helium.Node, positions map[helium.Node]int, pos *int) {
	if cur == nil {
		return
	}

	stack := []helium.Node{cur}
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		positions[n] = *pos
		// Stride 2: each node occupies an even slot, leaving odd slots
		// for virtual namespace nodes (position = parent + 1).
		*pos += 2
		if elem, ok := n.(*helium.Element); ok {
			for _, attr := range elem.Attributes() {
				positions[helium.Node(attr)] = *pos
				*pos += 2
			}
		}

		var children []helium.Node
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			children = append(children, child)
		}
		for i := len(children) - 1; i >= 0; i-- {
			stack = append(stack, children[i])
		}
	}
}

// Compare returns the relative document order of a and b.
// A negative result means a comes before b, a positive result means a comes
// after b, and zero means their indexed positions are equal or unknown.
func (c *DocOrderCache) Compare(a, b helium.Node) int {
	ra := DocumentRoot(a)
	rb := DocumentRoot(b)
	if ra == rb {
		pa := c.Position(a)
		pb := c.Position(b)
		switch {
		case pa < pb:
			return -1
		case pa > pb:
			return 1
		default:
			return 0
		}
	}

	ia, oka := c.documents[ra]
	ib, okb := c.documents[rb]
	switch {
	case !oka || !okb:
		return 0
	case ia.order < ib.order:
		return -1
	case ia.order > ib.order:
		return 1
	default:
		return 0
	}
}

func (c *DocOrderCache) Less(a, b helium.Node) bool {
	return c.Compare(a, b) < 0
}

// NSNodeKey identifies a namespace node by its parent element and prefix.
// NamespaceNodeWrapper objects are created fresh each time the namespace axis
// is traversed, so pointer-based identity fails for deduplication.
type NSNodeKey struct {
	Parent helium.Node
	Prefix string
}

// DeduplicateNodes removes duplicate nodes and sorts by document order.
// Returns ErrNodeSetLimit if the result exceeds maxNodes.
func DeduplicateNodes(nodes []helium.Node, cache *DocOrderCache, maxNodes int) ([]helium.Node, error) {
	if len(nodes) <= 1 {
		return nodes, nil
	}
	seen := make(map[helium.Node]struct{}, len(nodes))
	nsKeys := make(map[NSNodeKey]struct{})
	result := make([]helium.Node, 0, len(nodes))
	for _, n := range nodes {
		if _, ok := seen[n]; ok {
			continue
		}
		if n.Type() == helium.NamespaceNode {
			key := NSNodeKey{Parent: n.Parent(), Prefix: n.Name()}
			if _, ok := nsKeys[key]; ok {
				continue
			}
			nsKeys[key] = struct{}{}
		}
		seen[n] = struct{}{}
		result = append(result, n)
	}
	if len(result) > maxNodes {
		return nil, ErrNodeSetLimit
	}
	for _, n := range result {
		cache.BuildFrom(n)
	}
	sort.SliceStable(result, func(i, j int) bool {
		return cache.Less(result[i], result[j])
	})
	return result, nil
}

// MergeNodeSets merges two node slices, deduplicates, and sorts by document order.
func MergeNodeSets(a, b []helium.Node, cache *DocOrderCache, maxNodes int) ([]helium.Node, error) {
	seen := make(map[helium.Node]struct{}, len(a)+len(b))
	nsKeys := make(map[NSNodeKey]struct{})
	var result []helium.Node

	addNode := func(n helium.Node) {
		if _, ok := seen[n]; ok {
			return
		}
		if n.Type() == helium.NamespaceNode {
			key := NSNodeKey{Parent: n.Parent(), Prefix: n.Name()}
			if _, ok := nsKeys[key]; ok {
				return
			}
			nsKeys[key] = struct{}{}
		}
		seen[n] = struct{}{}
		result = append(result, n)
	}

	for _, n := range a {
		addNode(n)
	}
	for _, n := range b {
		addNode(n)
	}
	if len(result) > maxNodes {
		return nil, ErrNodeSetLimit
	}
	for _, n := range result {
		cache.BuildFrom(n)
	}
	sort.SliceStable(result, func(i, j int) bool {
		return cache.Less(result[i], result[j])
	})
	return result, nil
}

// DocumentRoot returns the owning Document or the topmost ancestor.
// Namespace node wrappers may not have OwnerDocument set, so we
// resolve via Parent first for those.
func DocumentRoot(n helium.Node) helium.Node {
	// Namespace node wrappers are created fresh and may lack
	// OwnerDocument; start from the parent element instead.
	if n.Type() == helium.NamespaceNode {
		if p := n.Parent(); p != nil {
			n = p
		}
	}
	if doc := n.OwnerDocument(); doc != nil {
		return doc
	}
	for n.Parent() != nil {
		n = n.Parent()
	}
	return n
}
