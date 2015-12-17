package helium

import (
	"bytes"
	"errors"
)

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

func (n node) Type() NodeType {
	return n.typ
}

func (n node) FirstChild() Node {
	l := len(n.children)
	if l == 0 {
		return nil
	}

	return n.children[0]
}

func (n node) LastChild() Node {
	l := len(n.children)
	if l == 0 {
		return nil
	}

	return n.children[l-1]
}

func (n *node) AddChild(cur Node) error {
	if l := len(n.children); l > 0 {
		n.children[l-1].AddSibling(cur)
	}
	n.children = append(n.children, cur)
	return nil
}

func (n node) AddContent(b []byte) error {
	return nil
}

func (n *node) AddSibling(cur Node) error {
	n.SetNextSibling(cur)
	cur.SetPrevSibling(n)
	return nil
}

func (n node) Content() []byte {
	b := bytes.Buffer{}
	for i := 0; i < len(n.children); i++ {
		chld := n.children[i]
		b.Write(chld.Content())
	}
	return b.Bytes()
}

func (n node) NextSibling() Node {
	return n.next
}

func (n node) PrevSibling() Node {
	return n.prev
}

func (n *node) SetPrevSibling(cur Node) {
	n.prev =  cur
}

func (n *node) SetNextSibling(cur Node) {
	n.next =  cur
}
