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
// They receive the same position as their parent element. This is correct
// because: (1) the parent has position P and its first attribute has P+1, so
// namespace nodes at P sort between the element and its attributes/children;
// (2) SliceStable preserves input order for equal positions, keeping
// same-parent namespace nodes in their traversal order; (3) namespace nodes
// are deduplicated by {parent, prefix} in DeduplicateNodes/MergeNodeSets,
// so duplicates from different union branches are already eliminated.
func (c *DocOrderCache) Position(n helium.Node) int {
	if c.positions == nil {
		return -1
	}
	if n.Type() == helium.NamespaceNode {
		parent := n.Parent()
		if parent == nil {
			return 0
		}
		return c.Position(parent)
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
	*pos++
	if elem, ok := cur.(*helium.Element); ok {
		for _, attr := range elem.Attributes() {
			c.positions[helium.Node(attr)] = *pos
			*pos++
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
func DocumentRoot(n helium.Node) helium.Node {
	if doc := n.OwnerDocument(); doc != nil {
		return doc
	}
	for n.Parent() != nil {
		n = n.Parent()
	}
	return n
}
