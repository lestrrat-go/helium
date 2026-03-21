package xpath

import (
	"reflect"

	helium "github.com/lestrrat-go/helium"
)

// isNilNode checks for both interface nil and typed nil (Go interface nil trap).
// IsNilNode returns true if n is nil or a nil pointer wrapped in a
// non-nil interface.
func IsNilNode(n helium.Node) bool {
	if n == nil {
		return true
	}
	v := reflect.ValueOf(n)
	return v.Kind() == reflect.Pointer && v.IsNil()
}

// AxisType identifies one of the 13 XPath axes.
type AxisType int

// AxisChild and the other AxisType constants identify the thirteen XPath axes.
const (
	AxisChild AxisType = iota
	AxisDescendant
	AxisParent
	AxisAncestor
	AxisFollowingSibling
	AxisPrecedingSibling
	AxisFollowing
	AxisPreceding
	AxisAttribute
	AxisNamespace
	AxisSelf
	AxisDescendantOrSelf
	AxisAncestorOrSelf
)

var axisNames = map[AxisType]string{
	AxisChild:            "child",
	AxisDescendant:       "descendant",
	AxisParent:           "parent",
	AxisAncestor:         "ancestor",
	AxisFollowingSibling: "following-sibling",
	AxisPrecedingSibling: "preceding-sibling",
	AxisFollowing:        "following",
	AxisPreceding:        "preceding",
	AxisAttribute:        "attribute",
	AxisNamespace:        "namespace",
	AxisSelf:             "self",
	AxisDescendantOrSelf: "descendant-or-self",
	AxisAncestorOrSelf:   "ancestor-or-self",
}

func (a AxisType) String() string {
	if s, ok := axisNames[a]; ok {
		return s
	}
	return "unknown-axis"
}

// AxisFromName maps an axis name string to its AxisType.
// Returns the axis and true if recognized, or AxisChild and false otherwise.
func AxisFromName(name string) (AxisType, bool) {
	for k, v := range axisNames {
		if v == name {
			return k, true
		}
	}
	return AxisChild, false
}

// TraverseAxis returns the nodes along the given axis from the context node,
// in the order defined by the XPath spec. maxNodes limits the result size
// for unbounded axes; use DefaultMaxNodeSetLength if unsure.
func TraverseAxis(axis AxisType, node helium.Node, maxNodes int) ([]helium.Node, error) {
	if IsNilNode(node) {
		return nil, nil
	}
	switch axis {
	case AxisDescendant:
		return axisDescendant(node, maxNodes)
	case AxisDescendantOrSelf:
		return axisDescendantOrSelf(node, maxNodes)
	case AxisFollowing:
		return axisFollowing(node, maxNodes)
	case AxisPreceding:
		return axisPreceding(node, maxNodes)
	}
	return TraverseAxisSimple(axis, node), nil
}

// TraverseAxisSimple handles axes that cannot fail (bounded result size).
func TraverseAxisSimple(axis AxisType, node helium.Node) []helium.Node {
	if IsNilNode(node) {
		return nil
	}
	switch axis {
	case AxisChild:
		return axisChild(node)
	case AxisParent:
		return axisParent(node)
	case AxisAncestor:
		return axisAncestor(node)
	case AxisAncestorOrSelf:
		return axisAncestorOrSelf(node)
	case AxisFollowingSibling:
		return axisFollowingSibling(node)
	case AxisPrecedingSibling:
		return axisPrecedingSibling(node)
	case AxisSelf:
		return []helium.Node{node}
	case AxisAttribute:
		return axisAttribute(node)
	case AxisNamespace:
		return axisNamespace(node)
	}
	return nil
}

func axisChild(node helium.Node) []helium.Node {
	// In XPath, attributes have no children
	if _, ok := node.(*helium.Attribute); ok {
		return nil
	}
	var result []helium.Node
	for c := node.FirstChild(); c != nil; c = c.NextSibling() {
		result = append(result, c)
	}
	return result
}

func axisDescendant(node helium.Node, maxNodes int) ([]helium.Node, error) {
	var result []helium.Node
	if err := collectDescendants(node, &result, maxNodes); err != nil {
		return nil, err
	}
	return result, nil
}

func appendAxisNode(result *[]helium.Node, node helium.Node, maxNodes int) error {
	*result = append(*result, node)
	if len(*result) > maxNodes {
		return ErrNodeSetLimit
	}
	return nil
}

func collectDescendants(node helium.Node, result *[]helium.Node, maxNodes int) error {
	// In XPath, attributes have no children
	if _, ok := node.(*helium.Attribute); ok {
		return nil
	}
	var stack []helium.Node
	for c := node.LastChild(); c != nil; c = c.PrevSibling() {
		stack = append(stack, c)
	}
	for len(stack) > 0 {
		last := len(stack) - 1
		cur := stack[last]
		stack = stack[:last]

		if err := appendAxisNode(result, cur, maxNodes); err != nil {
			return err
		}
		for child := cur.LastChild(); child != nil; child = child.PrevSibling() {
			stack = append(stack, child)
		}
	}
	return nil
}

func axisDescendantOrSelf(node helium.Node, maxNodes int) ([]helium.Node, error) {
	result := make([]helium.Node, 0, 1)
	if err := appendAxisNode(&result, node, maxNodes); err != nil {
		return nil, err
	}
	if err := collectDescendants(node, &result, maxNodes); err != nil {
		return nil, err
	}
	return result, nil
}

func axisParent(node helium.Node) []helium.Node {
	if p := node.Parent(); p != nil {
		return []helium.Node{p}
	}
	return nil
}

func axisAncestor(node helium.Node) []helium.Node {
	var result []helium.Node
	for p := node.Parent(); p != nil; p = p.Parent() {
		result = append(result, p)
	}
	return result
}

func axisAncestorOrSelf(node helium.Node) []helium.Node {
	if IsNilNode(node) {
		return nil
	}
	result := []helium.Node{node}
	for p := node.Parent(); p != nil; p = p.Parent() {
		result = append(result, p)
	}
	return result
}

func axisFollowingSibling(node helium.Node) []helium.Node {
	var result []helium.Node
	for s := node.NextSibling(); s != nil; s = s.NextSibling() {
		result = append(result, s)
	}
	return result
}

func axisPrecedingSibling(node helium.Node) []helium.Node {
	var result []helium.Node
	for s := node.PrevSibling(); s != nil; s = s.PrevSibling() {
		result = append(result, s)
	}
	return result
}

func axisFollowing(node helium.Node, maxNodes int) ([]helium.Node, error) {
	var result []helium.Node
	for s := node.NextSibling(); s != nil; s = s.NextSibling() {
		if err := appendAxisNode(&result, s, maxNodes); err != nil {
			return nil, err
		}
		if err := collectDescendants(s, &result, maxNodes); err != nil {
			return nil, err
		}
	}
	for p := node.Parent(); p != nil; p = p.Parent() {
		for s := p.NextSibling(); s != nil; s = s.NextSibling() {
			if err := appendAxisNode(&result, s, maxNodes); err != nil {
				return nil, err
			}
			if err := collectDescendants(s, &result, maxNodes); err != nil {
				return nil, err
			}
		}
	}
	return result, nil
}

func axisPreceding(node helium.Node, maxNodes int) ([]helium.Node, error) {
	var result []helium.Node
	for s := node.PrevSibling(); s != nil; s = s.PrevSibling() {
		if err := collectDescendantsReverse(s, &result, maxNodes); err != nil {
			return nil, err
		}
		if err := appendAxisNode(&result, s, maxNodes); err != nil {
			return nil, err
		}
	}
	for p := node.Parent(); p != nil; p = p.Parent() {
		for s := p.PrevSibling(); s != nil; s = s.PrevSibling() {
			if err := collectDescendantsReverse(s, &result, maxNodes); err != nil {
				return nil, err
			}
			if err := appendAxisNode(&result, s, maxNodes); err != nil {
				return nil, err
			}
		}
	}
	return result, nil
}

func collectDescendantsReverse(node helium.Node, result *[]helium.Node, maxNodes int) error {
	type frame struct {
		node     helium.Node
		expanded bool
	}

	var stack []frame
	for c := node.FirstChild(); c != nil; c = c.NextSibling() {
		stack = append(stack, frame{node: c})
	}
	for len(stack) > 0 {
		last := len(stack) - 1
		cur := stack[last]
		stack = stack[:last]

		if cur.expanded {
			if err := appendAxisNode(result, cur.node, maxNodes); err != nil {
				return err
			}
			continue
		}

		stack = append(stack, frame{node: cur.node, expanded: true})
		for child := cur.node.FirstChild(); child != nil; child = child.NextSibling() {
			stack = append(stack, frame{node: child})
		}
	}
	return nil
}

func axisAttribute(node helium.Node) []helium.Node {
	elem, ok := node.(*helium.Element)
	if !ok {
		return nil
	}
	// Keep the zero-attribute case allocation-free; the small append growth
	// cost for the common 1-3 attribute case is an acceptable tradeoff here.
	var result []helium.Node
	elem.ForEachAttribute(func(attr *helium.Attribute) bool {
		result = append(result, attr)
		return true
	})
	return result
}

func axisNamespace(node helium.Node) []helium.Node {
	elem, ok := node.(*helium.Element)
	if !ok {
		return nil
	}

	var ancestors []helium.Node
	for cur := helium.Node(elem); cur != nil; cur = cur.Parent() {
		ancestors = append(ancestors, cur)
	}

	inScope := NamespacePrefixesInScope(ancestors)
	return CollectNamespaceNodes(ancestors, inScope, elem)
}

// NamespacePrefixesInScope returns a map of prefix → active (non-empty URI)
// by walking ancestors from outermost to innermost so inner declarations win.
func NamespacePrefixesInScope(ancestors []helium.Node) map[string]bool {
	type nser interface{ Namespaces() []*helium.Namespace }
	inScope := map[string]bool{}
	for i := len(ancestors) - 1; i >= 0; i-- {
		ns, ok := ancestors[i].(nser)
		if !ok {
			continue
		}
		for _, n := range ns.Namespaces() {
			inScope[n.Prefix()] = n.URI() != ""
		}
	}
	return inScope
}

// CollectNamespaceNodes builds the namespace node list for an element, adding
// the xml prefix first and then ancestor-declared prefixes in document order.
// For each prefix, the innermost (closest ancestor) declaration provides the
// URI, but the output order follows document order (outermost first).
func CollectNamespaceNodes(ancestors []helium.Node, inScope map[string]bool, elem *helium.Element) []helium.Node {
	type nser interface{ Namespaces() []*helium.Namespace }

	// First pass: find the correct namespace for each prefix (innermost wins).
	// ancestors[0] is the element itself (innermost), ancestors[len-1] outermost.
	winner := map[string]*helium.Namespace{}
	for _, anc := range ancestors {
		ns, ok := anc.(nser)
		if !ok {
			continue
		}
		for _, n := range ns.Namespaces() {
			prefix := n.Prefix()
			if _, found := winner[prefix]; found {
				continue // innermost already recorded
			}
			if !inScope[prefix] || n.URI() == "" {
				continue
			}
			winner[prefix] = n
		}
	}

	var result []helium.Node

	xmlNS := helium.NewNamespace("xml", "http://www.w3.org/XML/1998/namespace")
	result = append(result, helium.NewNamespaceNodeWrapper(xmlNS, elem))

	// Second pass: output in document order (outermost to innermost), but
	// use the innermost URI for each prefix.
	emitted := map[string]struct{}{"xml": {}}
	for i := len(ancestors) - 1; i >= 0; i-- {
		ns, ok := ancestors[i].(nser)
		if !ok {
			continue
		}
		for _, n := range ns.Namespaces() {
			prefix := n.Prefix()
			if _, done := emitted[prefix]; done {
				continue
			}
			w := winner[prefix]
			if w == nil {
				continue
			}
			emitted[prefix] = struct{}{}
			result = append(result, helium.NewNamespaceNodeWrapper(w, elem))
		}
	}

	return result
}
