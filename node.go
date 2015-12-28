package helium

import (
	"bytes"
	"errors"

	"github.com/lestrrat/helium/internal/debug"
)

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
	debug.Printf("-----> addContent to %s (%p)", n.Name(), n)
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
debug.Printf("addChild n = %p", n)
	l := n.LastChild()
	if l == nil { // No children, set firstChild to cur
		debug.Printf("------------> node %s has no children", n.Name())
		n.setFirstChild(cur)
		n.setLastChild(cur)
	} else {
		debug.Printf("------------> node %s has children", n.Name())
		l.SetNextSibling(cur)
		cur.SetPrevSibling(l)
		n.setLastChild(cur)
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
