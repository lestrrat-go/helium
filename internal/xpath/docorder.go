package xpath

import (
	"sort"

	helium "github.com/lestrrat-go/helium"
)

// DocOrderCache caches document-order positions for all nodes in a document.
// Built lazily on first use and shared across an entire evaluation.
type DocOrderCache struct {
	positions map[helium.Node]int
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
	if c.positions == nil {
		return -1
	}
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
	pos, ok := c.positions[n]
	if !ok {
		return -1
	}
	return pos
}

// BuildFrom populates the cache by walking the tree rooted at root.
// No-op if already populated.
func (c *DocOrderCache) BuildFrom(root helium.Node) {
	if c.positions != nil {
		return
	}
	c.positions = make(map[helium.Node]int)
	pos := 0
	c.indexWalk(root, &pos)
}

func (c *DocOrderCache) indexWalk(cur helium.Node, pos *int) {
	c.positions[cur] = *pos
	// Stride 2: each node occupies an even slot, leaving odd slots
	// for virtual namespace nodes (position = parent + 1).
	*pos += 2
	if elem, ok := cur.(*helium.Element); ok {
		for _, attr := range elem.Attributes() {
			c.positions[helium.Node(attr)] = *pos
			*pos += 2
		}
	}
	for child := cur.FirstChild(); child != nil; child = child.NextSibling() {
		c.indexWalk(child, pos)
	}
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
	seen := make(map[helium.Node]bool, len(nodes))
	nsKeys := make(map[NSNodeKey]bool)
	result := make([]helium.Node, 0, len(nodes))
	for _, n := range nodes {
		if seen[n] {
			continue
		}
		if n.Type() == helium.NamespaceNode {
			key := NSNodeKey{Parent: n.Parent(), Prefix: n.Name()}
			if nsKeys[key] {
				continue
			}
			nsKeys[key] = true
		}
		seen[n] = true
		result = append(result, n)
	}
	if len(result) > maxNodes {
		return nil, ErrNodeSetLimit
	}
	// All nodes belong to the same document within a single evaluation.
	// Multi-document scenarios (fn:doc) are not yet supported; when added,
	// the cache must be partitioned per document.
	cache.BuildFrom(DocumentRoot(result[0]))
	sort.SliceStable(result, func(i, j int) bool {
		return cache.Position(result[i]) < cache.Position(result[j])
	})
	return result, nil
}

// MergeNodeSets merges two node slices, deduplicates, and sorts by document order.
func MergeNodeSets(a, b []helium.Node, cache *DocOrderCache, maxNodes int) ([]helium.Node, error) {
	seen := make(map[helium.Node]bool, len(a)+len(b))
	nsKeys := make(map[NSNodeKey]bool)
	var result []helium.Node

	addNode := func(n helium.Node) {
		if seen[n] {
			return
		}
		if n.Type() == helium.NamespaceNode {
			key := NSNodeKey{Parent: n.Parent(), Prefix: n.Name()}
			if nsKeys[key] {
				return
			}
			nsKeys[key] = true
		}
		seen[n] = true
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
	if len(result) > 0 {
		cache.BuildFrom(DocumentRoot(result[0]))
	}
	sort.SliceStable(result, func(i, j int) bool {
		return cache.Position(result[i]) < cache.Position(result[j])
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
