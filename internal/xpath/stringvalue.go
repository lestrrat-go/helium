package xpath

import (
	"strings"

	"github.com/lestrrat-go/helium"
)

// StringValue returns the XPath string-value of a node.
// Rules are identical across XPath 1.0 and 3.1.
func StringValue(n helium.Node) string {
	// Check Attribute by type assertion first since etype may not be set
	if attr, ok := n.(*helium.Attribute); ok {
		return attr.Value()
	}
	switch n.Type() {
	case helium.DocumentNode, helium.ElementNode:
		// XPath spec 5.2: string-value of element/document is the
		// concatenation of string-values of all text node descendants.
		var b strings.Builder
		appendTextDescendants(&b, n)
		return b.String()
	case helium.TextNode, helium.CDATASectionNode:
		return string(n.Content())
	case helium.CommentNode:
		return string(n.Content())
	case helium.ProcessingInstructionNode:
		return string(n.Content())
	case helium.NamespaceNode:
		return string(n.Content())
	}
	return ""
}

// appendTextDescendants walks descendants iteratively so programmatically
// constructed deep trees do not truncate string-value computation.
func appendTextDescendants(b *strings.Builder, root helium.Node) {
	stack := []helium.Node{root}
	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		switch cur.Type() {
		case helium.TextNode, helium.CDATASectionNode:
			b.Write(cur.Content())
		}

		for child := cur.LastChild(); child != nil; child = child.PrevSibling() {
			stack = append(stack, child)
		}
	}
}

// LocalNameOf returns the local name of any node type.
func LocalNameOf(n helium.Node) string {
	switch v := n.(type) {
	case *helium.Element:
		return v.LocalName()
	case *helium.Attribute:
		ln := v.LocalName()
		if i := strings.IndexByte(ln, ':'); i >= 0 {
			return ln[i+1:]
		}
		return ln
	case *helium.ProcessingInstruction:
		return v.Name()
	case *helium.NamespaceNodeWrapper:
		return v.Name()
	default:
		// Document, text, comment nodes have no local name per XPath spec
		return ""
	}
}

// NodeNamespaceURI returns the namespace URI of any node type.
func NodeNamespaceURI(n helium.Node) string {
	type urier interface {
		URI() string
	}
	if u, ok := n.(urier); ok {
		return u.URI()
	}
	return ""
}

// NodePrefix returns the namespace prefix of any node type.
func NodePrefix(n helium.Node) string {
	type prefixer interface {
		Prefix() string
	}
	if p, ok := n.(prefixer); ok {
		return p.Prefix()
	}
	return ""
}
