package xpath

import (
	"cmp"
	"slices"
	"sort"
	"sync"

	helium "github.com/lestrrat-go/helium"
)

// DocOrderCache caches document-order positions for nodes grouped by
// document root. Built lazily on first use and shared across an evaluation.
//
// A single DocOrderCache may be shared across concurrent Evaluate calls, so all
// access to its maps is guarded by mu.
//
// The cached positions describe the tree as it was when first indexed. Callers
// MUST call Reset after mutating any document this cache describes (inserting,
// removing, or moving nodes), otherwise order results may be stale. Subsequent
// lookups after Reset recompute order from the current tree.
type DocOrderCache struct {
	mu sync.Mutex
	// keys holds one entry per indexed node across every indexed document, so
	// a position lookup costs a single hash probe with no parent-chain walk.
	keys map[helium.Node]sortKey
	// documents records the registration order of each indexed document root.
	// It is consulted when indexing a new document and when comparing two
	// nodes of which at least one is not indexed.
	documents map[helium.Node]int
}

// Reset clears all cached document-order state so the same cache value can be
// safely reused after the underlying document(s) are mutated. Subsequent
// lookups recompute order from the current tree.
//
// Callers MUST call Reset after mutating a document this cache describes, or
// order results may be stale.
func (c *DocOrderCache) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.keys = nil
	c.documents = nil
}

// sortKey holds precomputed sort information for a node, avoiding
// repeated map lookups during the O(n log n) sort phase.
type sortKey struct {
	docOrder int // document registration order (for cross-tree comparison)
	position int // position within the document
}

// unknownSortKey is the key of a node no indexed document describes.
var unknownSortKey = sortKey{docOrder: -1, position: -1}

// sortKeyLocked returns the sort key of n from the already-indexed state,
// without indexing anything. The caller must hold c.mu.
func (c *DocOrderCache) sortKeyLocked(n helium.Node) sortKey {
	if k, ok := c.keys[n]; ok {
		return k
	}
	if n.Type() != helium.NamespaceNode {
		return unknownSortKey
	}
	// Namespace node wrappers are created fresh on every namespace-axis
	// traversal, so they are never indexed. They take the odd slot the
	// stride-2 walk leaves right after their parent element.
	parent := n.Parent()
	if parent == nil {
		return unknownSortKey
	}
	pk := c.sortKeyLocked(parent)
	if pk.position < 0 {
		return unknownSortKey
	}
	return sortKey{docOrder: pk.docOrder, position: pk.position + 1}
}

// ensureSortKeyLocked returns the sort key of n, indexing n's document first if
// it is not indexed yet. The caller must hold c.mu.
func (c *DocOrderCache) ensureSortKeyLocked(n helium.Node) sortKey {
	if k, ok := c.keys[n]; ok {
		return k
	}
	if n.Type() == helium.NamespaceNode {
		if parent := n.Parent(); parent != nil {
			pk := c.ensureSortKeyLocked(parent)
			if pk.position < 0 {
				return unknownSortKey
			}
			return sortKey{docOrder: pk.docOrder, position: pk.position + 1}
		}
		// A parentless wrapper resolves to itself as a root; index it anyway so
		// document registration order matches a plain BuildFrom on the node.
		c.buildFromLocked(DocumentRoot(n))
		return unknownSortKey
	}
	c.buildFromLocked(DocumentRoot(n))
	if k, ok := c.keys[n]; ok {
		return k
	}
	return unknownSortKey
}

// indexSortKeys returns the sort key of every node in a followed by every node
// in b, indexing any document not yet described by the cache. b may be nil.
// It takes c.mu once for the whole batch instead of once per node.
func (c *DocOrderCache) indexSortKeys(a, b []helium.Node) []sortKey {
	keys := make([]sortKey, 0, len(a)+len(b))
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, n := range a {
		keys = append(keys, c.ensureSortKeyLocked(n))
	}
	for _, n := range b {
		keys = append(keys, c.ensureSortKeyLocked(n))
	}
	return keys
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
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.sortKeyLocked(n).position
}

// BuildFrom populates the cache by walking the tree rooted at root.
// No-op if that root is already indexed.
func (c *DocOrderCache) BuildFrom(root helium.Node) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.buildFromLocked(DocumentRoot(root))
}

// buildFromLocked indexes the tree rooted at root, which must already be a
// DocumentRoot result. The caller must hold c.mu.
func (c *DocOrderCache) buildFromLocked(root helium.Node) {
	if c.documents == nil {
		c.documents = make(map[helium.Node]int)
	}
	if _, ok := c.documents[root]; ok {
		return
	}
	order := len(c.documents)
	c.documents[root] = order
	if c.keys == nil {
		c.keys = make(map[helium.Node]sortKey)
	}
	pos := 0
	c.indexWalk(root, order, &pos)
}

func (c *DocOrderCache) indexWalk(cur helium.Node, order int, pos *int) {
	if cur == nil {
		return
	}

	stack := make([]helium.Node, 0, 256)
	stack = append(stack, cur)
	// childBuf is reused across stack iterations to collect a node's owned
	// children before pushing them in reverse.
	var childBuf []helium.Node
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		c.keys[n] = sortKey{docOrder: order, position: *pos}
		// Stride 2: each node occupies an even slot, leaving odd slots
		// for virtual namespace nodes (position = parent + 1).
		*pos += 2
		if elem, ok := n.(*helium.Element); ok {
			elem.ForEachAttribute(func(attr *helium.Attribute) bool {
				c.keys[helium.Node(attr)] = sortKey{docOrder: order, position: *pos}
				*pos += 2
				return true
			})
		}

		// Enumerate n's OWNED children via helium.Children, which stops at a
		// foreign-owned child (an entity reference's shared Entity node is owned
		// by the DTD, and its sibling pointers thread into the DTD's declaration
		// list) and is cycle-safe. A raw LastChild/PrevSibling walk would escape
		// into the DTD's declarations and assign them spurious document-order
		// positions. This mirrors axisChild, which also enumerates via
		// helium.Children, so entity-declaration and DTD nodes stay out of the
		// order index. helium.Children iterates forward, so buffer the children
		// and push them right-to-left so the left-most is processed first;
		// entity-free documents get byte-identical positions.
		childBuf = childBuf[:0]
		for child := range helium.Children(n) {
			childBuf = append(childBuf, child)
		}
		for _, child := range slices.Backward(childBuf) {
			stack = append(stack, child)
		}
	}
}

// Compare returns the relative document order of a and b.
// A negative result means a comes before b, a positive result means a comes
// after b, and zero means their indexed positions are equal or unknown.
func (c *DocOrderCache) Compare(a, b helium.Node) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	ka := c.sortKeyLocked(a)
	kb := c.sortKeyLocked(b)
	if ka.docOrder >= 0 && kb.docOrder >= 0 {
		// Both nodes are indexed, so their document registration order
		// already tells whether they share a root.
		if ka.docOrder == kb.docOrder {
			return cmp.Compare(ka.position, kb.position)
		}
		return cmp.Compare(ka.docOrder, kb.docOrder)
	}

	// At least one node is not described by any indexed document. Resolve the
	// roots the long way so an unknown node keeps comparing as it always has.
	ra := DocumentRoot(a)
	rb := DocumentRoot(b)
	if ra == rb {
		return cmp.Compare(ka.position, kb.position)
	}
	oa, oka := c.documents[ra]
	ob, okb := c.documents[rb]
	if !oka || !okb {
		return 0
	}
	return cmp.Compare(oa, ob)
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

// strictlyOrdered reports whether keys is strictly increasing. A strictly
// increasing key sequence proves the corresponding nodes are already in
// document order AND pairwise distinct: every indexed node owns a unique
// position, so two equal keys mean either a repeat or a pair the sort would
// have to settle.
func strictlyOrdered(keys []sortKey) bool {
	for i := 1; i < len(keys); i++ {
		prev, cur := keys[i-1], keys[i]
		if prev.docOrder > cur.docOrder {
			return false
		}
		if prev.docOrder == cur.docOrder && prev.position >= cur.position {
			return false
		}
	}
	return true
}

// sortByPrecomputedKeys sorts result by precomputed sort keys, avoiding
// repeated map lookups during the O(n log n) sort phase.
// Uses SliceStable to preserve input order for equal positions.
func sortByPrecomputedKeys(result []helium.Node, keys []sortKey) {
	// Check if already sorted (common case for single-axis traversals).
	sorted := true
	for i := 1; i < len(keys); i++ {
		ki, kj := keys[i-1], keys[i]
		if ki.docOrder > kj.docOrder || (ki.docOrder == kj.docOrder && ki.position > kj.position) {
			sorted = false
			break
		}
	}
	if sorted {
		return
	}

	// Build an index array and sort that; then permute result in-place.
	n := len(result)
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(i, j int) bool {
		ki, kj := keys[idx[i]], keys[idx[j]]
		if ki.docOrder != kj.docOrder {
			return ki.docOrder < kj.docOrder
		}
		return ki.position < kj.position
	})
	// Apply the permutation to result.
	tmp := make([]helium.Node, n)
	for i, j := range idx {
		tmp[i] = result[j]
	}
	copy(result, tmp)
}

// boundedCap returns a capacity for a dedup buffer (result or seen map) that is
// bounded by the node-set limit so duplicate-heavy input does not over-allocate
// proportional to the full input length. The returned capacity is min(n,
// maxNodes+1): never more than one slot past the limit (the extra slot lets the
// callers detect overflow before returning ErrNodeSetLimit).
//
// This bounds allocation only; it does not enforce the limit. Enforcement is the
// caller's job via the len(result) > maxNodes checks below, which reject any
// result that exceeds the limit. Every caller of DeduplicateNodes,
// DeduplicateNodesPreserveOrder, and MergeNodeSets passes a positive maxNodes
// (DefaultMaxNodeSetLength or a config override), so the limit is always active.
// maxNodes <= 0 is treated as "no allocation cap" here purely as a defensive
// fallback for sizing; it is not a supported "unlimited" mode, because the
// enforcement checks below would reject a non-empty result in that case.
// maxNodes+1 is guarded against integer overflow.
func boundedCap(n, maxNodes int) int {
	if maxNodes <= 0 {
		return n
	}
	limit := maxNodes + 1
	if limit < maxNodes { // overflow: maxNodes was math.MaxInt
		return n
	}
	if n < limit {
		return n
	}
	return limit
}

// DeduplicateNodes removes duplicate nodes and sorts by document order.
// Returns ErrNodeSetLimit if the result exceeds maxNodes.
func DeduplicateNodes(nodes []helium.Node, cache *DocOrderCache, maxNodes int) ([]helium.Node, error) {
	if len(nodes) <= 1 {
		return nodes, nil
	}
	// Resolve every input node's sort key up front, under a single lock, and
	// index any document the cache does not describe yet. Document
	// registration order follows first appearance in the input, which is the
	// order the deduplicated result would have produced.
	keys := cache.indexSortKeys(nodes, nil)
	if len(nodes) <= maxNodes && strictlyOrdered(keys) {
		// The input is already pairwise distinct and in document order, so both
		// the dedup pass and the sort are no-ops. Clamp the capacity so the
		// returned slice never shares spare capacity with the caller's buffer.
		return nodes[:len(nodes):len(nodes)], nil
	}
	// Cap the seen-map and result allocations at the limit (plus one slot to
	// detect overflow) so a large, duplicate-heavy input does not over-allocate
	// buffers sized to the full input when the deduplicated result fits well
	// within maxNodes.
	seen := make(map[helium.Node]struct{}, boundedCap(len(nodes), maxNodes))
	var nsKeys map[NSNodeKey]struct{}
	result := make([]helium.Node, 0, boundedCap(len(nodes), maxNodes))
	resultKeys := make([]sortKey, 0, boundedCap(len(nodes), maxNodes))
	for i, n := range nodes {
		if _, ok := seen[n]; ok {
			continue
		}
		if n.Type() == helium.NamespaceNode {
			if nsKeys == nil {
				nsKeys = make(map[NSNodeKey]struct{})
			}
			key := NSNodeKey{Parent: n.Parent(), Prefix: n.Name()}
			if _, ok := nsKeys[key]; ok {
				continue
			}
			nsKeys[key] = struct{}{}
		}
		seen[n] = struct{}{}
		result = append(result, n)
		resultKeys = append(resultKeys, keys[i])
		if len(result) > maxNodes {
			return nil, ErrNodeSetLimit
		}
	}

	sortByPrecomputedKeys(result, resultKeys)
	return result, nil
}

// DeduplicateNodesPreserveOrder removes duplicate nodes while preserving
// the input order (no document-order sort). Used when the caller has
// explicitly ordered the sequence (e.g. fn:reverse, fn:sort).
// Returns ErrNodeSetLimit if the result exceeds maxNodes.
func DeduplicateNodesPreserveOrder(nodes []helium.Node, maxNodes int) ([]helium.Node, error) {
	if len(nodes) <= 1 {
		return nodes, nil
	}
	// Cap the seen-map and result allocations at the limit (plus one slot to
	// detect overflow) so a large, duplicate-heavy input does not over-allocate
	// buffers sized to the full input when the deduplicated result fits well
	// within maxNodes.
	seen := make(map[helium.Node]struct{}, boundedCap(len(nodes), maxNodes))
	var nsKeys map[NSNodeKey]struct{}
	result := make([]helium.Node, 0, boundedCap(len(nodes), maxNodes))
	for _, n := range nodes {
		if _, ok := seen[n]; ok {
			continue
		}
		if n.Type() == helium.NamespaceNode {
			if nsKeys == nil {
				nsKeys = make(map[NSNodeKey]struct{})
			}
			key := NSNodeKey{Parent: n.Parent(), Prefix: n.Name()}
			if _, ok := nsKeys[key]; ok {
				continue
			}
			nsKeys[key] = struct{}{}
		}
		seen[n] = struct{}{}
		result = append(result, n)
		if len(result) > maxNodes {
			return nil, ErrNodeSetLimit
		}
	}
	return result, nil
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
	// forward/backward search. This is O(distance) between them,
	// well under O(total_siblings).
	return compareSiblingOrder(ancA, ancB)
}

// MergeNodeSets merges two node slices, deduplicates, and sorts by document order.
func MergeNodeSets(a, b []helium.Node, cache *DocOrderCache, maxNodes int) ([]helium.Node, error) {
	// Resolve every input node's sort key up front, under a single lock, and
	// index any document the cache does not describe yet. Document
	// registration order follows first appearance across a then b, which is the
	// order the deduplicated result would have produced.
	keys := cache.indexSortKeys(a, b)
	// Cap the seen-map and result allocations at the limit (plus one slot to
	// detect overflow) so a large, duplicate-heavy input does not over-allocate
	// buffers sized to the full input when the deduplicated result fits well
	// within maxNodes.
	total := len(a) + len(b)
	seen := make(map[helium.Node]struct{}, boundedCap(total, maxNodes))
	var nsKeys map[NSNodeKey]struct{}
	result := make([]helium.Node, 0, boundedCap(total, maxNodes))
	resultKeys := make([]sortKey, 0, boundedCap(total, maxNodes))

	addNode := func(n helium.Node, key sortKey) error {
		if _, ok := seen[n]; ok {
			return nil
		}
		if n.Type() == helium.NamespaceNode {
			if nsKeys == nil {
				nsKeys = make(map[NSNodeKey]struct{})
			}
			nsk := NSNodeKey{Parent: n.Parent(), Prefix: n.Name()}
			if _, ok := nsKeys[nsk]; ok {
				return nil
			}
			nsKeys[nsk] = struct{}{}
		}
		seen[n] = struct{}{}
		result = append(result, n)
		resultKeys = append(resultKeys, key)
		// Early-exit as soon as the result exceeds the limit so we neither
		// process the rest of the input nor grow the result buffer past the
		// bounded capacity.
		if len(result) > maxNodes {
			return ErrNodeSetLimit
		}
		return nil
	}

	for i, n := range a {
		if err := addNode(n, keys[i]); err != nil {
			return nil, err
		}
	}
	for i, n := range b {
		if err := addNode(n, keys[len(a)+i]); err != nil {
			return nil, err
		}
	}

	sortByPrecomputedKeys(result, resultKeys)
	return result, nil
}

// compareSiblingOrder determines the order of two sibling nodes by
// walking forward and backward from both simultaneously. This is
// O(distance) between the two nodes, well under O(total_siblings).
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
