package helium

import (
	"bytes"
	"errors"
)

func (n docnode) Content() []byte {
	b := bytes.Buffer{}
	for i := 0; i < len(n.children); i++ {
		chld := n.children[i]
		b.Write(chld.Content())
	}
	return b.Bytes()
}

func (n *docnode) AddContent(b []byte) error {
	t := newText(b)
	return n.AddChild(t)
}

func (n *node) AddChild(cur Node) error {
	switch cur.Type() {
	case TextNode:
		if lc := n.LastChild(); lc != nil && lc.Type() == TextNode {
			return lc.AddContent(cur.Content())
		}
	}

	n.children = append(n.children, cur)
	return nil
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
	l := len(n.children)
	if l == 0 {
		return nil
	}

	return n.children[0]
}

func (n docnode) LastChild() Node {
	l := len(n.children)
	if l == 0 {
		return nil
	}

	return n.children[l-1]
}

func (n *docnode) AddChild(cur Node) error {
	if l := len(n.children); l > 0 {
		n.children[l-1].AddSibling(cur)
	}
	n.children = append(n.children, cur)
	return nil
}

func (n *docnode) AddSibling(cur Node) error {
	n.SetNextSibling(cur)
	cur.SetPrevSibling(n)
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
