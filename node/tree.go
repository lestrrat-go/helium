package node

import (
	"errors"
)

// treeNode is the part of a Node that handles the tree structure.
type treeNode struct {
	name       string
	firstChild Node
	lastChild  Node
	parent     Node
	next       Node
	prev       Node
	doc        *Document
}

func (n *treeNode) getTreeNode() *treeNode {
	return n
}

func (n *treeNode) OwnerDocument() *Document {
	return n.doc
}

func (n *treeNode) FirstChild() Node {
	return n.firstChild
}

func (n *treeNode) LastChild() Node {
	return n.lastChild
}

func (n *treeNode) Parent() Node {
	return n.parent
}

func (n *treeNode) NextSibling() Node {
	return n.next
}

func (n *treeNode) PrevSibling() Node {
	return n.prev
}

func (n *treeNode) Content(dst []byte) ([]byte, error) {
	result := dst
	for e := n.firstChild; e != nil; e = e.NextSibling() {
		var err error
		result, err = e.Content(result)
		if err != nil {
			return result, err
		}
	}
	return result, nil
}

func (n *treeNode) SetOwnerDocument(doc *Document) error {
	if n == nil {
		return errors.New("cannot set owner document to nil node")
	}
	if doc == nil {
		return errors.New("cannot set nil document")
	}

	n.doc = doc
	return nil
}

func (n *treeNode) SetParent(p Node) error {
	if n == nil {
		return errors.New("cannot set parent to nil node")
	}
	if p == nil {
		return errors.New("cannot set nil parent")
	}

	n.parent = p
	return nil
}

func addSibling(n, sibling Node) error {
	if n == nil {
		return errors.New("cannot add sibling to nil node")
	}
	if sibling == nil {
		return errors.New("cannot add nil sibling")
	}

	l := n
	lt := n.getTreeNode()
	st := sibling.getTreeNode()

	for lt.next != nil {
		l = lt.next
		lt = l.getTreeNode()
	}

	lt.next = sibling
	st.prev = l
	if lt.parent != nil {
		st.parent = lt.parent
		lt.parent.getTreeNode().lastChild = sibling
	}
	return nil
}

func addChild(parent, child Node) error {
	pt := parent.getTreeNode()
	ct := child.getTreeNode()

	l := pt.lastChild
	if l == nil { // No children, set firstChild to cur, and bail out
		pt.firstChild = child
		pt.lastChild = child
		ct.parent = parent
		return nil
	}

	// AddSibling handles setting the parent, and the
	// lastChild pointer
	if err := addSibling(l, child); err != nil {
		return err
	}

	/*
		// If the last child was a text node, keep the old LastChild
		if child.Type() == TextNodeType && l.Type() == TextNode {
			n.setLastChild(l)
		}
	*/
	return nil
}

func addContent(n Node, content []byte) error {
	t := NewText(content)
	return n.AddChild(t)
}

func replaceNode(n Node, cur Node) error {
	if next := n.NextSibling(); next != nil {
		cur.getTreeNode().next = next // cur.next = n.next
		next.getTreeNode().prev = cur // n.next.prev = cur
	}

	if prev := n.PrevSibling(); prev != nil {
		cur.getTreeNode().prev = prev // cur.prev = n.prev
		prev.getTreeNode().next = cur // n.prev.next = cur
	}

	if parent := n.Parent(); parent != nil {
		if parent.FirstChild() == n {
			parent.getTreeNode().firstChild = cur // parent.firstChild = cur
		}
		if parent.LastChild() == n {
			parent.getTreeNode().lastChild = cur // parent.lastChild = cur
		}
		cur.getTreeNode().parent = parent
	}
	return nil
}

func setNextSibling(n, sibling Node) error {
	if n == nil {
		return errors.New("cannot set next sibling to nil node")
	}
	if sibling == nil {
		return errors.New("cannot set nil sibling")
	}

	n.getTreeNode().next = sibling
	sibling.getTreeNode().prev = n

	if parent := n.Parent(); parent != nil {
		sibling.getTreeNode().parent = parent
		if parent.getTreeNode().lastChild == n {
			parent.getTreeNode().lastChild = sibling
		}
	}
	return nil
}

func setPrevSibling(n, sibling Node) error {
	if n == nil {
		return errors.New("cannot set previous sibling to nil node")
	}
	if sibling == nil {
		return errors.New("cannot set nil sibling")
	}

	n.getTreeNode().prev = sibling
	sibling.getTreeNode().next = n

	if parent := n.Parent(); parent != nil {
		sibling.getTreeNode().parent = parent
		if parent.getTreeNode().firstChild == n {
			parent.getTreeNode().firstChild = sibling
		}
	}
	return nil
}
