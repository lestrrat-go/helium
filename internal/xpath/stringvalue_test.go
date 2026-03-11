package xpath_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	ixpath "github.com/lestrrat-go/helium/internal/xpath"
	"github.com/stretchr/testify/require"
)

func TestStringValue_Element(t *testing.T) {
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root, err := doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, doc.AddChild(root))

	txt1, err := doc.CreateText([]byte("hello "))
	require.NoError(t, err)
	require.NoError(t, root.AddChild(txt1))

	child, err := doc.CreateElement("child")
	require.NoError(t, err)
	require.NoError(t, root.AddChild(child))

	txt2, err := doc.CreateText([]byte("world"))
	require.NoError(t, err)
	require.NoError(t, child.AddChild(txt2))

	// Element string-value is concatenation of all text descendants.
	require.Equal(t, "hello world", ixpath.StringValue(root))
}

func TestStringValue_Document(t *testing.T) {
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root, err := doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, doc.AddChild(root))

	txt, err := doc.CreateText([]byte("doc text"))
	require.NoError(t, err)
	require.NoError(t, root.AddChild(txt))

	require.Equal(t, "doc text", ixpath.StringValue(doc))
}

func TestStringValue_ElementWithCDATA(t *testing.T) {
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root, err := doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, doc.AddChild(root))

	txt, err := doc.CreateText([]byte("before "))
	require.NoError(t, err)
	require.NoError(t, root.AddChild(txt))

	cdata, err := doc.CreateCDATASection([]byte("cdata content"))
	require.NoError(t, err)
	require.NoError(t, root.AddChild(cdata))

	// CDATA sections are included in text descendant concatenation.
	require.Equal(t, "before cdata content", ixpath.StringValue(root))
}

func TestStringValue_Attribute(t *testing.T) {
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root, err := doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, doc.AddChild(root))
	require.NoError(t, root.SetAttribute("key", "value"))

	attrs := root.Attributes()
	require.Len(t, attrs, 1)
	require.Equal(t, "value", ixpath.StringValue(attrs[0]))
}

func TestStringValue_Comment(t *testing.T) {
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	comment, err := doc.CreateComment([]byte("a comment"))
	require.NoError(t, err)
	require.Equal(t, "a comment", ixpath.StringValue(comment))
}

func TestStringValue_PI(t *testing.T) {
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	pi, err := doc.CreatePI("target", "data here")
	require.NoError(t, err)
	require.Equal(t, "data here", ixpath.StringValue(pi))
}

func TestStringValue_Namespace(t *testing.T) {
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root, err := doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, doc.AddChild(root))

	ns := helium.NewNamespace("ns", "urn:example")
	nsw := helium.NewNamespaceNodeWrapper(ns, root)
	require.Equal(t, "urn:example", ixpath.StringValue(nsw))
}

func TestLocalNameOf(t *testing.T) {
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root, err := doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, doc.AddChild(root))

	// Element local name
	require.Equal(t, "root", ixpath.LocalNameOf(root))

	// Namespaced attribute local name
	require.NoError(t, root.DeclareNamespace("ns", "urn:ns"))
	ns := helium.NewNamespace("ns", "urn:ns")
	require.NoError(t, root.SetAttributeNS("myattr", "v", ns))
	attrs := root.Attributes()
	require.NotEmpty(t, attrs)
	require.Equal(t, "myattr", ixpath.LocalNameOf(attrs[len(attrs)-1]))

	// Comment — Name() is empty
	comment, err := doc.CreateComment([]byte("x"))
	require.NoError(t, err)
	require.Equal(t, "", ixpath.LocalNameOf(comment))

	// PI — Name() returns the target
	pi, err := doc.CreatePI("mytarget", "data")
	require.NoError(t, err)
	require.Equal(t, "mytarget", ixpath.LocalNameOf(pi))
}

func TestNodeNamespaceURI(t *testing.T) {
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root, err := doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, doc.AddChild(root))

	// Non-namespaced element
	require.Equal(t, "", ixpath.NodeNamespaceURI(root))

	// Namespaced attribute
	require.NoError(t, root.DeclareNamespace("ns", "urn:ns"))
	ns := helium.NewNamespace("ns", "urn:ns")
	require.NoError(t, root.SetAttributeNS("a", "v", ns))
	attrs := root.Attributes()
	require.NotEmpty(t, attrs)
	require.Equal(t, "urn:ns", ixpath.NodeNamespaceURI(attrs[len(attrs)-1]))

	// Comment has no namespace
	comment, err := doc.CreateComment([]byte("x"))
	require.NoError(t, err)
	require.Equal(t, "", ixpath.NodeNamespaceURI(comment))
}

func TestNodePrefix(t *testing.T) {
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root, err := doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, doc.AddChild(root))

	// Non-prefixed element
	require.Equal(t, "", ixpath.NodePrefix(root))

	// Namespaced attribute has prefix
	require.NoError(t, root.DeclareNamespace("ns", "urn:ns"))
	ns := helium.NewNamespace("ns", "urn:ns")
	require.NoError(t, root.SetAttributeNS("a", "v", ns))
	attrs := root.Attributes()
	require.NotEmpty(t, attrs)
	require.Equal(t, "ns", ixpath.NodePrefix(attrs[len(attrs)-1]))

	// Comment has no prefix
	comment, err := doc.CreateComment([]byte("x"))
	require.NoError(t, err)
	require.Equal(t, "", ixpath.NodePrefix(comment))
}
