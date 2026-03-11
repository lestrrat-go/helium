package xpath

import (
	"strings"

	helium "github.com/lestrrat-go/helium"
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
		CollectTextDescendants(n, &b)
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

// CollectTextDescendants recursively collects text from Text and CDATA descendants.
// Recursion depth is bounded by the document tree depth, which the parser already
// caps (maxElemDepth, default 256 / huge-mode 2048). No additional guard needed.
func CollectTextDescendants(n helium.Node, b *strings.Builder) {
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		switch c.Type() {
		case helium.TextNode, helium.CDATASectionNode:
			b.Write(c.Content())
		default:
			CollectTextDescendants(c, b)
		}
	}
}

// LocalNameOf returns the local name of any node type.
func LocalNameOf(n helium.Node) string {
	switch v := n.(type) {
	case *helium.Element:
		return v.LocalName()
	case *helium.Attribute:
		return v.LocalName()
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
