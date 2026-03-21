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
			elem.ForEachAttribute(func(attr *helium.Attribute) bool {
				positions[helium.Node(attr)] = *pos
				*pos += 2
				return true
			})
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

	// If the document is already indexed in the cache, use the fast
	// position-based sort. Otherwise, use ancestor-chain comparison
	// which avoids the expensive full-document indexing.
	if len(result) > 0 && cache.isIndexed(result[0]) {
		sort.SliceStable(result, func(i, j int) bool {
			return cache.Less(result[i], result[j])
		})
	} else {
		sort.SliceStable(result, func(i, j int) bool {
			return CompareNodeOrder(result[i], result[j]) < 0
		})
	}
	return result, nil
}

// isIndexed returns true if the document containing n is already indexed.
func (c *DocOrderCache) isIndexed(n helium.Node) bool {
	if c.documents == nil {
		return false
	}
	root := DocumentRoot(n)
	_, ok := c.documents[root]
	return ok
}

// CompareNodeOrder compares two nodes by document order using ancestor-chain
// walking. Returns -1 if a comes before b, +1 if after, 0 if same node.
// This is O(depth) per call, avoiding the need to index the entire document.
func CompareNodeOrder(a, b helium.Node) int {
	if a == b {
		return 0
	}

	// Handle attribute nodes: attributes come after their parent element
	// but before the element's children.
	aIsAttr := a.Type() == helium.AttributeNode
	bIsAttr := b.Type() == helium.AttributeNode

	// Handle namespace nodes similarly to attributes.
	aIsNS := a.Type() == helium.NamespaceNode
	bIsNS := b.Type() == helium.NamespaceNode

	// Get the "element-level" ancestor for attrs/ns nodes
	aElem, bElem := a, b
	if aIsAttr || aIsNS {
		aElem = a.Parent()
	}
	if bIsAttr || bIsNS {
		bElem = b.Parent()
	}

	// If both are attrs/ns of the same element
	if aElem == bElem && (aIsAttr || aIsNS) && (bIsAttr || bIsNS) {
		// Namespace nodes come before attribute nodes
		if aIsNS && bIsAttr {
			return -1
		}
		if aIsAttr && bIsNS {
			return 1
		}
		// Both are same-type (attr or ns) on the same element.
		// Walk forward from each to find the other; the one found
		// first when walking forward from the other is later in
		// document order.
		fwdA := a.NextSibling()
		fwdB := b.NextSibling()
		for fwdA != nil || fwdB != nil {
			if fwdA == b {
				return -1 // a comes before b
			}
			if fwdB == a {
				return 1 // b comes before a
			}
			if fwdA != nil {
				fwdA = fwdA.NextSibling()
			}
			if fwdB != nil {
				fwdB = fwdB.NextSibling()
			}
		}
		return 0
	}

	// If one is an attr/ns of the other's element
	if aElem == b && (aIsAttr || aIsNS) {
		return 1 // attr/ns comes after its element
	}
	if bElem == a && (bIsAttr || bIsNS) {
		return -1
	}

	// Compare the element-level ancestors
	if aElem != bElem {
		cmp := compareElementOrder(aElem, bElem)
		if cmp != 0 {
			return cmp
		}
	}

	// Same element, one is attr/ns and the other is the element itself
	if aIsAttr || aIsNS {
		return 1
	}
	if bIsAttr || bIsNS {
		return -1
	}
	return 0
}

// compareElementOrder compares two non-attribute nodes by document order.
func compareElementOrder(a, b helium.Node) int {
	if a == b {
		return 0
	}

	// Compute depths
	depthA := nodeDepth(a)
	depthB := nodeDepth(b)

	// Walk both to the same depth
	ancA, ancB := a, b
	dA, dB := depthA, depthB
	for dA > dB {
		ancA = ancA.Parent()
		dA--
	}
	for dB > dA {
		ancB = ancB.Parent()
		dB--
	}

	// If they converge, one is an ancestor of the other
	if ancA == ancB {
		// The deeper original node is a descendant
		if depthA < depthB {
			return -1 // a is ancestor of b
		}
		return 1 // b is ancestor of a
	}

	// Walk up until we find siblings under a common parent
	for ancA.Parent() != ancB.Parent() {
		ancA = ancA.Parent()
		ancB = ancB.Parent()
	}

	// ancA and ancB are siblings; determine order by interleaved
	// forward/backward search. This is O(distance) between them
	// rather than O(total_siblings).
	return compareSiblingOrder(ancA, ancB)
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
	if len(result) > 0 && cache.isIndexed(result[0]) {
		sort.SliceStable(result, func(i, j int) bool {
			return cache.Less(result[i], result[j])
		})
	} else {
		sort.SliceStable(result, func(i, j int) bool {
			return CompareNodeOrder(result[i], result[j]) < 0
		})
	}
	return result, nil
}

// compareSiblingOrder determines the order of two sibling nodes by
// walking forward and backward from both simultaneously. This is
// O(distance) between the two nodes rather than O(total_siblings).
func compareSiblingOrder(a, b helium.Node) int {
	// Interleave: walk NextSibling from a and PrevSibling from a.
	// Also walk NextSibling from b and PrevSibling from b.
	fwdA := a.NextSibling()
	fwdB := b.NextSibling()
	for fwdA != nil || fwdB != nil {
		if fwdA == b {
			return -1 // a comes before b
		}
		if fwdB == a {
			return 1 // b comes before a
		}
		if fwdA != nil {
			fwdA = fwdA.NextSibling()
		}
		if fwdB != nil {
			fwdB = fwdB.NextSibling()
		}
	}
	// Should not reach here if a and b are truly siblings
	return 0
}

// nodeDepth returns the depth of a node in the tree (0 for root).
func nodeDepth(n helium.Node) int {
	depth := 0
	for p := n.Parent(); p != nil; p = p.Parent() {
		depth++
	}
	return depth
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
	// Walk up the parent chain to find the root. If the topmost node is
	// a document node, return it. Otherwise return the topmost element
	// (parentless node — e.g. elements created in sequence mode).
	// We avoid returning OwnerDocument() directly because parentless
	// elements may be owned by a temporary document they are not rooted in.
	top := n
	for top.Parent() != nil {
		top = top.Parent()
	}
	if top.Type() == helium.DocumentNode || top.Type() == helium.HTMLDocumentNode {
		return top
	}
	// If the topmost ancestor is not a document and OwnerDocument exists,
	// check if the topmost node is actually a child of that document.
	if doc := n.OwnerDocument(); doc != nil {
		for c := doc.FirstChild(); c != nil; c = c.NextSibling() {
			if c == top {
				return doc
			}
		}
	}
	return top
}
