package helium

import "github.com/lestrrat-go/pdebug"

// Text represents an XML text node (libxml2: xmlNode with type XML_TEXT_NODE).
type Text struct {
	node
}

const textNodeName = "(text)"

func newText(b []byte) *Text {
	t := Text{}
	t.etype = TextNode
	t.content = make([]byte, len(b))
	copy(t.content, b)
	t.name = textNodeName
	return &t
}

func (n *Text) AddChild(cur Node) error {
	if t, ok := cur.(*Text); ok {
		return n.AppendText(t.content)
	}
	return ErrInvalidOperation
}

func (n *Text) AppendText(b []byte) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START Text.AppendText '%s' (%p)", b, n)
		defer func() {
			g.IRelease("END Text.AppendText '%s'", n.content)
		}()
	}
	if doc := n.doc; doc != nil {
		n.content = doc.growOwnedTextContent(n.content, len(b))
	}
	n.content = append(n.content, b...)
	return nil
}

func (n *Text) AddSibling(cur Node) error {
	if pdebug.Enabled {
		g := pdebug.IPrintf("START Text.AddSibling '%s'", cur.Content())
		defer g.IRelease("END Text.AddSibling")
	}
	if cur.Type() == TextNode {
		return n.AppendText(cur.Content())
	}
	return addSibling(n, cur)
}

func (n Text) Content() []byte {
	return n.content
}

func (n *Text) Replace(nodes ...Node) error {
	return replaceNode(n, nodes...)
}

func (n *Text) SetTreeDoc(doc *Document) {
	setTreeDoc(n, doc)
}
