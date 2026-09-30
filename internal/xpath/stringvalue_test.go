package xpath_test

import (
	"strconv"
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

	txt1 := doc.CreateText([]byte("hello "))
	require.NoError(t, root.AddChild(txt1))

	child, err := doc.CreateElement("child")
	require.NoError(t, err)
	require.NoError(t, root.AddChild(child))

	txt2 := doc.CreateText([]byte("world"))
	require.NoError(t, child.AddChild(txt2))

	// Element string-value is concatenation of all text descendants.
	require.Equal(t, "hello world", ixpath.StringValue(root))
}

func TestStringValue_Document(t *testing.T) {
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root, err := doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, doc.AddChild(root))

	txt := doc.CreateText([]byte("doc text"))
	require.NoError(t, root.AddChild(txt))

	require.Equal(t, "doc text", ixpath.StringValue(doc))
}

func TestStringValue_ElementWithCDATA(t *testing.T) {
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root, err := doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, doc.AddChild(root))

	txt := doc.CreateText([]byte("before "))
	require.NoError(t, root.AddChild(txt))

	cdata := doc.CreateCDATASection([]byte("cdata content"))
	require.NoError(t, root.AddChild(cdata))

	// CDATA sections are included in text descendant concatenation.
	require.Equal(t, "before cdata content", ixpath.StringValue(root))
}

func TestStringValue_MixedContent(t *testing.T) {
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)

	comment := doc.CreateComment([]byte("ignored document comment"))
	require.NoError(t, doc.AddChild(comment))

	root, err := doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, doc.AddChild(root))

	text1 := doc.CreateText([]byte("alpha"))
	require.NoError(t, root.AddChild(text1))

	comment = doc.CreateComment([]byte("ignored element comment"))
	require.NoError(t, root.AddChild(comment))

	child, err := doc.CreateElement("child")
	require.NoError(t, err)
	require.NoError(t, root.AddChild(child))

	text2 := doc.CreateText([]byte("beta"))
	require.NoError(t, child.AddChild(text2))

	pi := doc.CreatePI("target", "ignored processing instruction")
	require.NoError(t, child.AddChild(pi))

	cdata := doc.CreateCDATASection([]byte("gamma"))
	require.NoError(t, child.AddChild(cdata))

	text3 := doc.CreateText([]byte("delta"))
	require.NoError(t, root.AddChild(text3))

	require.Equal(t, "alphabetagammadelta", ixpath.StringValue(root))
	require.Equal(t, "alphabetagammadelta", ixpath.StringValue(doc))
}

func TestStringValue_Attribute(t *testing.T) {
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root, err := doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, doc.AddChild(root))
	err = root.SetAttribute("key", "value")
	require.NoError(t, err)

	attrs := root.Attributes()
	require.Len(t, attrs, 1)
	require.Equal(t, "value", ixpath.StringValue(attrs[0]))
}

func TestStringValue_Comment(t *testing.T) {
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	comment := doc.CreateComment([]byte("a comment"))
	require.Equal(t, "a comment", ixpath.StringValue(comment))
}

func TestStringValue_PI(t *testing.T) {
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	pi := doc.CreatePI("target", "data here")
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
	err = root.SetAttributeNS("myattr", "v", ns)
	require.NoError(t, err)
	attrs := root.Attributes()
	require.NotEmpty(t, attrs)
	require.Equal(t, "myattr", ixpath.LocalNameOf(attrs[len(attrs)-1]))

	// Comment — Name() is empty
	comment := doc.CreateComment([]byte("x"))
	require.Equal(t, "", ixpath.LocalNameOf(comment))

	// PI — Name() returns the target
	pi := doc.CreatePI("mytarget", "data")
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
	err = root.SetAttributeNS("a", "v", ns)
	require.NoError(t, err)
	attrs := root.Attributes()
	require.NotEmpty(t, attrs)
	require.Equal(t, "urn:ns", ixpath.NodeNamespaceURI(attrs[len(attrs)-1]))

	// Comment has no namespace
	comment := doc.CreateComment([]byte("x"))
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
	err = root.SetAttributeNS("a", "v", ns)
	require.NoError(t, err)
	attrs := root.Attributes()
	require.NotEmpty(t, attrs)
	require.Equal(t, "ns", ixpath.NodePrefix(attrs[len(attrs)-1]))

	// Comment has no prefix
	comment := doc.CreateComment([]byte("x"))
	require.Equal(t, "", ixpath.NodePrefix(comment))
}

func TestStringValue_DeepTreeDoesNotTruncate(t *testing.T) {
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root, err := doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, doc.AddChild(root))

	prefix := doc.CreateText([]byte("prefix"))
	require.NoError(t, root.AddChild(prefix))

	parent := root
	for range 4096 {
		child, err := doc.CreateElement("level")
		require.NoError(t, err)
		require.NoError(t, parent.AddChild(child))
		parent = child
	}

	leaf := doc.CreateText([]byte("leaf"))
	require.NoError(t, parent.AddChild(leaf))

	require.Equal(t, "prefixleaf", ixpath.StringValue(root))
}

// entityStringValueSrc declares entities between comments and PIs of the
// internal subset: f holds a PI and a comment, and e, declared after f, nests
// a reference to f inside markup. The content references e, and g inside an
// attribute value, in a single-reference element and in an adjacent pair.
const entityStringValueSrc = `<!DOCTYPE a [<!ENTITY g "G"><!ENTITY f "F<?pf x?><!--fc-->">` +
	`<!--c0--><?p0 x?><!ENTITY e "<b>ent<!--ec-->&f;</b>t"><!--c1--><?p1 y?>]>` +
	`<a>1&e;2<x v="q&g;r&amp;">&e;<!--cx--><![CDATA[c]]></x><y>&g;</y><z>&g;&g;</z></a>`

// TestStringValue_EntityReferences checks that the string-value of a node
// holding entity references is the text the references expand to, so both
// parse modes read what a substituted parse stores: a reference never reaches
// text of other declarations in the DTD that owns its entity, and the DTD of a
// document is not part of the document's string-value.
func TestStringValue_EntityReferences(t *testing.T) {
	want := map[string]string{
		"doc": "1entFt2entFtcGGG",
		"a":   "1entFt2entFtcGGG",
		"x":   "entFtc",
		"y":   "G",
		"z":   "GG",
		"@v":  "qGr&",
	}
	for _, substitute := range []bool{false, true} {
		t.Run("substitute="+strconv.FormatBool(substitute), func(t *testing.T) {
			doc, err := helium.NewParser().SubstituteEntities(substitute).
				Parse(t.Context(), []byte(entityStringValueSrc))
			require.NoError(t, err)
			got := map[string]string{"doc": ixpath.StringValue(doc)}
			root := doc.DocumentElement()
			got[root.Name()] = ixpath.StringValue(root)
			for child := range helium.ChildElements(root) {
				if _, ok := want[child.Name()]; !ok {
					continue // b is a child of a only in a substituted parse
				}
				got[child.Name()] = ixpath.StringValue(child)
				for _, attr := range child.Attributes() {
					got["@"+attr.Name()] = ixpath.StringValue(attr)
				}
			}
			require.Equal(t, want, got)
		})
	}
}
