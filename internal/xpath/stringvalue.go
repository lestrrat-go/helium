package xpath

import (
	"strings"

	"github.com/lestrrat-go/helium"
)

// StringValue returns the XPath string-value of a node.
// Rules are identical across XPath 1.0 and 3.1.
//
// An element's string-value is its Content(): the text of every Text and CDATA
// descendant, with every entity reference expanded and comments and PIs left
// out (libxml2: xmlXPathCastNodeToString -> xmlNodeGetContent). The XDM has no
// entity references, so the value an entity reference adds is the text it
// expands to, and both parse modes give the value of the tree a
// SubstituteEntities(true) parse builds. The expansion follows only owned
// children, so a reference never reaches text of other declarations in the DTD
// that owns its entity. A document's string-value is that of its element,
// text and entity-reference children; its DTD is not part of the XPath data
// model and adds nothing (documentStringValue).
func StringValue(n helium.Node) string {
	// Check Attribute by type assertion first since etype may not be set
	if attr, ok := n.(*helium.Attribute); ok {
		return attr.Value()
	}
	switch n.Type() {
	case helium.DocumentNode:
		return documentStringValue(n)
	case helium.ElementNode, helium.TextNode, helium.CDATASectionNode, helium.CommentNode,
		helium.ProcessingInstructionNode, helium.NamespaceNode:
		return string(n.Content())
	}
	return ""
}

// documentStringValue concatenates the content of doc's element, text, CDATA
// and entity-reference children. The DTD, comments and PIs add nothing. The
// common single-element document returns that element's content without an
// extra copy through a builder.
func documentStringValue(doc helium.Node) string {
	var single []byte
	var b strings.Builder
	parts := 0
	for child := range helium.Children(doc) {
		switch child.Type() {
		case helium.ElementNode, helium.TextNode, helium.CDATASectionNode, helium.EntityRefNode:
		default:
			continue
		}
		c := child.Content()
		if len(c) == 0 {
			continue
		}
		parts++
		if parts == 1 {
			single = c
			continue
		}
		if parts == 2 {
			b.Write(single)
		}
		b.Write(c)
	}
	if parts > 1 {
		return b.String()
	}
	return string(single)
}

// LocalNameOf returns the local name of any node type.
func LocalNameOf(n helium.Node) string {
	switch v := n.(type) {
	case *helium.Element:
		return v.LocalName()
	case *helium.Attribute:
		ln := v.LocalName()
		if _, after, ok := strings.Cut(ln, ":"); ok {
			return after
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
