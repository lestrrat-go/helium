package helium

import (
	"bytes"
	"errors"

	"github.com/lestrrat-go/pdebug/v3"
)

func addContent(n Node, b []byte) error {
	t := newText(b)
	return n.AddChild(t)
}

func addSibling(n, cur Node) error {
	for n != nil {
		if n.NextSibling() == nil {
			n.SetNextSibling(cur)
			cur.SetPrevSibling(n)
			parent := n.Parent()
			cur.SetParent(parent)
			if parent != nil {
				parent.setLastChild(cur)
			}
			return nil
		}
		n = n.NextSibling()
	}

	return errors.New("cannot add sibling to nil node")
}

// Generic addChild() method. Adds n to cur
func addChild(n Node, cur Node) error {
	l := n.LastChild()
	if l == nil { // No children, set firstChild to cur
		if pdebug.Enabled {
			pdebug.Printf("LastChild is nil, setting firstChild and lastChild")
		}
		n.setFirstChild(cur)
		n.setLastChild(cur)
		cur.SetParent(n)
		return nil
	}

	// AddSibling handles setting the parent, and the
	// lastChild pointer
	if err := l.AddSibling(cur); err != nil {
		return err
	}

	// If the last child was a text node, keep the old LastChild
	if cur.Type() == TextNode && l.Type() == TextNode {
		n.setLastChild(l)
	}
	return nil
}

func getContent(n Node) []byte {
	var b bytes.Buffer
	for e := n.FirstChild(); e != nil; e = e.NextSibling() {
		b.Write(getContent(e))
	}
	return b.Bytes()
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


