package helium

import (
	"bytes"
	"errors"
)

// because docnode contains links to other nodes, one tends to want to make
// methods for docnodes that cover the rest of the Node types. However,
// this cannot be done because the way Go does method reuse -- by delegation.
// For example, a method that changes the parent's point to the current node would
// be bad:
//
// func (n *docnode) MakeMeYourParent(cur Node) {
//   cur.SetParent(n)
// }
//
// Wait, you just passed a pointer to the docnode, not the container node
// such as Element, Text, Comment, etc.
//
// So basically the deal is: if you need methods that may mutate the current
// node AND the operand node, DO NOT implement it for docnode. That includes
// things like AddSibling, or AddChild.
//
// On the other hand, methods like setFirstChild and setLastChild are OK,
// as they only mutate the current (docnode)'s pointers.

func (n *docnode) setFirstChild(cur Node) {
	n.firstChild = cur
}

func (n *docnode) setLastChild(cur Node) {
	n.lastChild = cur
}

func (n docnode) Parent() Node {
	return n.parent
}

func (n docnode) Content() []byte {
	b := bytes.Buffer{}
	for e := n.firstChild; e != nil; e = e.NextSibling() {
		b.Write(e.Content())
	}
	return b.Bytes()
}

func addContent(n Node, b []byte) error {
	t := newText(b)
	return n.AddChild(t)
}

type WalkFunc func(Node) error

func Walk(n Node, f WalkFunc) error {
	if n == nil {
		return errors.New("nil node")
	}

	if err := f(n); err != nil {
		return err
	}
	for chld := n.FirstChild(); chld != nil; chld = chld.NextSibling() {
		if err := Walk(chld, f); err != nil {
			return err
		}
	}
	return nil
}

func (n docnode) LocalName() string {
	return n.name
}

func (n docnode) Name() string {
	return n.name
}

func (n docnode) Type() ElementType {
	return n.etype
}

func (n docnode) FirstChild() Node {
	return n.firstChild
}

func (n docnode) LastChild() Node {
	return n.lastChild
}

func addChild(n Node, cur Node) error {
	l := n.LastChild()
	if l == nil { // No children, set firstChild to cur
		n.setFirstChild(cur)
		n.setLastChild(cur)
	} else {
		if err := l.AddSibling(cur); err != nil {
			return err
		}
		// If the last child was a text node, keep the old LastChild
		if cur.Type() != TextNode || l.Type() != TextNode {
			n.setLastChild(cur)
		}
	}
	cur.SetParent(n)
	return nil
}

func (n docnode) NextSibling() Node {
	if n.next == nil {
		return nil
	}
	return n.next
}

func (n docnode) PrevSibling() Node {
	return n.prev
}

func addSibling(n, cur Node) error {
	for n != nil {
		if n.NextSibling() == nil {
			n.SetNextSibling(cur)
			cur.SetPrevSibling(n)
			return nil
		}
		n = n.NextSibling()
	}

	return errors.New("cannot add sibling to nil node")
}

func (n *docnode) SetPrevSibling(cur Node) {
	n.prev = cur
}

func (n *docnode) SetNextSibling(cur Node) {
	n.next = cur
}

func (n *docnode) SetParent(cur Node) {
	n.parent = cur
}

func replaceNode(n Node, cur Node) {
	if next := n.NextSibling(); next != nil {
		cur.SetNextSibling(next) // cur.next = n.next
		next.SetPrevSibling(cur) // n.next.prev = cur
	}

	if prev := n.PrevSibling(); prev != nil {
		cur.SetPrevSibling(prev) // cur.prev = n.prev
		prev.SetNextSibling(cur) // n.prev.next = cur
	}

	if parent := n.Parent(); parent != nil {
		if parent.FirstChild() == n {
			parent.setFirstChild(cur)
		}
		if parent.LastChild() == n {
			parent.setLastChild(cur)
		}
		cur.SetParent(parent)
	}
}

func (n node) Namespaces() []*Namespace {
	return n.nsDefs
}

func (n *node) SetNamespace(prefix, uri string, activate ...bool) error {
	ns, err := n.doc.CreateNamespace(prefix, uri)
	if err != nil {
		return err
	}

	a := false
	if len(activate) > 0 {
		a = activate[0]
	}
	if a {
		n.ns = ns
	}

	return nil
}

func (n node) Prefix() string {
	if ns := n.ns; ns != nil {
		return ns.Prefix()
	}
	return ""
}

func (n node) URI() string {
	if ns := n.ns; ns != nil {
		return ns.URI()
	}
	return ""
}

func (n node) Name() string {
	if ns := n.ns; ns != nil && ns.Prefix() != "" {
		return ns.Prefix() + ":" + n.name
	}
	return n.name
}
