package c14n

import (
	"slices"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/c14nctl"
)

func init() {
	c14nctl.SubtreeRoot = subtreeRootHook
}

// subtreeRootHook adapts the subtree-root option to the untyped
// internal/c14nctl hook (any in, any out) so a sibling package can start a
// node-set walk at one subtree without a public method here or an import
// cycle.
func subtreeRootHook(c, root any) any {
	canon, ok := c.(Canonicalizer)
	if !ok {
		return c
	}
	elem, ok := root.(*helium.Element)
	if !ok {
		return c
	}
	canon = canon.clone()
	canon.cfg.subtreeRoot = elem
	return canon
}

// subtreeAncestors returns subtreeRoot's element ancestors, outermost first,
// and reports whether the walk can start at subtreeRoot. It cannot when an
// ancestor is not an element (subtreeRoot lies in entity replacement content),
// is the excluded element, has a member in the node set (the element itself,
// one of its namespace nodes or one of its attributes), when the chain does
// not end at the canonicalized document, or when the document declares an
// entity with element content (see hasElementEntities). The whole-document
// walk handles those cases.
func (c *canonicalizer) subtreeAncestors() ([]*helium.Element, bool) {
	if c.hasElementEntities() {
		return nil, false
	}
	var chain []*helium.Element
	for n := c.subtreeRoot.Parent(); ; n = n.Parent() {
		if n == nil {
			return nil, false
		}
		if n.Type() == helium.DocumentNode {
			if doc, ok := helium.AsNode[*helium.Document](n); !ok || doc != c.doc {
				return nil, false
			}
			break
		}
		e, ok := helium.AsNode[*helium.Element](n)
		if !ok || e == c.exclude || c.hasNodeSetMembers(e) {
			return nil, false
		}
		chain = append(chain, e)
	}
	slices.Reverse(chain)
	return chain, true
}

// hasElementEntities reports whether the document's internal or external DTD
// subset declares an entity whose replacement content holds an element. Such an
// element is shared by every reference to the entity, so a node set built from
// the subtree can hold its namespace nodes, and the whole-document walk renders
// those as text at a reference outside the subtree too (renderNSNodesAsText on
// the omitted expansion). Starting at the subtree would drop that text.
func (c *canonicalizer) hasElementEntities() bool {
	for _, dtd := range []*helium.DTD{c.doc.IntSubset(), c.doc.ExtSubset()} {
		if dtd == nil {
			continue
		}
		for n := dtd.FirstChild(); n != nil; n = n.NextSibling() {
			if n.Type() != helium.EntityNode {
				continue
			}
			for child := range helium.Children(n) {
				if child.Type() == helium.ElementNode {
					return true
				}
			}
		}
	}
	return false
}

// hasNodeSetMembers reports whether e, one of its namespace nodes, or one of
// its attributes is in the node set: the nodes an omitted element would
// render.
func (c *canonicalizer) hasNodeSetMembers(e *helium.Element) bool {
	if _, ok := c.nodeSet[e]; ok {
		return true
	}
	if _, ok := c.nsNodesByElement[e]; ok {
		return true
	}
	for _, attr := range c.elementAttributes(e) {
		if _, ok := c.nodeSet[attr]; ok {
			return true
		}
	}
	return false
}

// processSubtree canonicalizes a node set that lies in subtreeRoot's subtree
// without walking the rest of the document. Outside that subtree the
// whole-document walk renders nothing: every element there is omitted and has
// no node-set members, and no other node is a member. What it does do there is
// reject a relative namespace URI on any element and build the in-scope
// bindings of subtreeRoot's ancestors. processSubtree does both in the same
// document order, so the bytes and the first error match the whole-document
// walk: it checks the elements before subtreeRoot (each ancestor, then the
// siblings preceding the path), pushes each ancestor's scope frame outermost
// first, processes subtreeRoot, and then checks the elements after it.
func (c *canonicalizer) processSubtree(ancestors []*helium.Element) error {
	next := helium.Node(c.subtreeRoot)
	if len(ancestors) > 0 {
		next = ancestors[0]
	}
	if err := c.checkNamespacesBefore(c.doc, next); err != nil {
		return err
	}
	for i, anc := range ancestors {
		if err := c.checkForRelativeNamespaces(anc); err != nil {
			return err
		}
		c.pushScope(anc)
		c.rendered.push()
		next = helium.Node(c.subtreeRoot)
		if i+1 < len(ancestors) {
			next = ancestors[i+1]
		}
		if err := c.checkNamespacesBefore(anc, next); err != nil {
			return err
		}
	}

	if err := c.processElement(c.subtreeRoot); err != nil {
		return err
	}

	for cur := helium.Node(c.subtreeRoot); cur.Parent() != nil && cur.Type() != helium.DocumentNode; cur = cur.Parent() {
		for sib := cur.NextSibling(); sib != nil; sib = sib.NextSibling() {
			if err := c.checkNamespacesIn(sib); err != nil {
				return err
			}
		}
	}
	return nil
}

// checkNamespacesBefore checks the children of parent that precede stop.
func (c *canonicalizer) checkNamespacesBefore(parent, stop helium.Node) error {
	for child := parent.FirstChild(); child != nil && child != stop; child = child.NextSibling() {
		if err := c.checkNamespacesIn(child); err != nil {
			return err
		}
	}
	return nil
}

// checkNamespacesIn applies checkForRelativeNamespaces to every element the
// whole-document walk would visit under n, in the same order: element
// children, and the replacement content of entity references, skipping the
// excluded subtree.
func (c *canonicalizer) checkNamespacesIn(n helium.Node) error {
	switch n.Type() {
	case helium.ElementNode:
		e, ok := helium.AsNode[*helium.Element](n)
		if !ok || e == c.exclude {
			return nil
		}
		if err := c.checkForRelativeNamespaces(e); err != nil {
			return err
		}
	case helium.EntityRefNode, helium.EntityNode:
	default:
		return nil
	}
	for child := range helium.Children(n) {
		if err := c.checkNamespacesIn(child); err != nil {
			return err
		}
	}
	return nil
}
