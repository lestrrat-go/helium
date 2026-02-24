package xpath

import (
	helium "github.com/lestrrat-go/helium"
)

// traverseAxis returns the nodes along the given axis from the context node,
// in the order defined by the XPath spec.
func traverseAxis(axis AxisType, node helium.Node) []helium.Node {
	switch axis {
	case AxisChild:
		return axisChild(node)
	case AxisDescendant:
		return axisDescendant(node)
	case AxisDescendantOrSelf:
		return axisDescendantOrSelf(node)
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
	case AxisFollowing:
		return axisFollowing(node)
	case AxisPreceding:
		return axisPreceding(node)
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

func axisDescendant(node helium.Node) []helium.Node {
	var result []helium.Node
	collectDescendants(node, &result)
	return result
}

func collectDescendants(node helium.Node, result *[]helium.Node) {
	// In XPath, attributes have no children
	if _, ok := node.(*helium.Attribute); ok {
		return
	}
	for c := node.FirstChild(); c != nil; c = c.NextSibling() {
		*result = append(*result, c)
		collectDescendants(c, result)
	}
}

func axisDescendantOrSelf(node helium.Node) []helium.Node {
	var result []helium.Node
	result = append(result, node)
	collectDescendants(node, &result)
	return result
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

func axisFollowing(node helium.Node) []helium.Node {
	var result []helium.Node
	// Start with following siblings and their descendants
	for s := node.NextSibling(); s != nil; s = s.NextSibling() {
		result = append(result, s)
		collectDescendants(s, &result)
	}
	// Then move to ancestors' following siblings
	for p := node.Parent(); p != nil; p = p.Parent() {
		for s := p.NextSibling(); s != nil; s = s.NextSibling() {
			result = append(result, s)
			collectDescendants(s, &result)
		}
	}
	return result
}

func axisPreceding(node helium.Node) []helium.Node {
	var result []helium.Node
	// Preceding siblings and their descendants (in reverse document order)
	for s := node.PrevSibling(); s != nil; s = s.PrevSibling() {
		collectDescendantsReverse(s, &result)
		result = append(result, s)
	}
	// Then ancestors' preceding siblings
	for p := node.Parent(); p != nil; p = p.Parent() {
		for s := p.PrevSibling(); s != nil; s = s.PrevSibling() {
			collectDescendantsReverse(s, &result)
			result = append(result, s)
		}
	}
	return result
}

func collectDescendantsReverse(node helium.Node, result *[]helium.Node) {
	for c := node.LastChild(); c != nil; c = c.PrevSibling() {
		collectDescendantsReverse(c, result)
		*result = append(*result, c)
	}
}

func axisAttribute(node helium.Node) []helium.Node {
	elem, ok := node.(*helium.Element)
	if !ok {
		return nil
	}
	attrs := elem.Attributes()
	result := make([]helium.Node, len(attrs))
	for i, a := range attrs {
		result[i] = a
	}
	return result
}

func axisNamespace(node helium.Node) []helium.Node {
	elem, ok := node.(*helium.Element)
	if !ok {
		return nil
	}

	// Collect ancestor chain (outermost first)
	var ancestors []helium.Node
	for cur := helium.Node(elem); cur != nil; cur = cur.Parent() {
		ancestors = append(ancestors, cur)
	}

	// Walk from outermost to innermost to collect in-scope namespaces.
	// Inner declarations override outer ones.
	seen := map[string]bool{}
	// First pass: determine which prefixes are in scope (inner wins)
	for i := 0; i < len(ancestors); i++ {
		nser, ok := ancestors[i].(interface{ Namespaces() []*helium.Namespace })
		if !ok {
			continue
		}
		for _, ns := range nser.Namespaces() {
			seen[ns.Prefix()] = ns.URI() != ""
		}
	}

	// Second pass: collect from outermost to innermost, matching libxml2 order
	seen2 := map[string]bool{}
	var result []helium.Node

	// xml namespace always first
	xmlNS := helium.NewNamespace("xml", "http://www.w3.org/XML/1998/namespace")
	result = append(result, helium.NewNamespaceNodeWrapper(xmlNS, elem))
	seen2["xml"] = true

	for i := len(ancestors) - 1; i >= 0; i-- {
		nser, ok := ancestors[i].(interface{ Namespaces() []*helium.Namespace })
		if !ok {
			continue
		}
		for _, ns := range nser.Namespaces() {
			prefix := ns.Prefix()
			if seen2[prefix] {
				continue
			}
			if !seen[prefix] {
				continue // undeclared by inner scope
			}
			seen2[prefix] = true
			if ns.URI() == "" {
				continue
			}
			result = append(result, helium.NewNamespaceNodeWrapper(ns, elem))
		}
	}

	return result
}
