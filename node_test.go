package helium_test

import (
	"context"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

func TestAddChildShape(t *testing.T) {
	// elem.AddChild(attr) must route the attribute into the element's property
	// list, NOT the child list: it appears in Attributes()/GetAttribute, is absent
	// from Children(), and serializes as an attribute rather than a child element.
	t.Run("attribute is routed to the property list", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		elem, err := doc.CreateElement("root")
		require.NoError(t, err)

		attr, err := doc.CreateAttribute("orphan", "v", nil)
		require.NoError(t, err)

		require.NoError(t, elem.AddChild(attr))

		// Present as an attribute.
		got, ok := elem.GetAttribute("orphan")
		require.True(t, ok, "attribute must be reachable via GetAttribute")
		require.Equal(t, "v", got)

		attrs := elem.Attributes()
		require.Len(t, attrs, 1)
		require.Equal(t, "orphan", attrs[0].Name())

		// Absent from the child list.
		for child := range helium.Children(elem) {
			t.Fatalf("attribute must not appear in the child list, found %s", child.Type())
		}

		// Serializes as an attribute, not a child element.
		out, err := helium.WriteString(elem)
		require.NoError(t, err)
		require.Contains(t, out, `orphan="v"`)
		require.NotContains(t, out, "<orphan>")
	})

	// Routing an attribute through AddChild replaces an existing same-named
	// attribute in place (libxml2 xmlAddChild parity via addProperty).
	t.Run("attribute replaces a same-name attribute", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		elem, err := doc.CreateElement("root")
		require.NoError(t, err)

		err = elem.SetAttribute("id", "first")
		require.NoError(t, err)

		replacement, err := doc.CreateAttribute("id", "second", nil)
		require.NoError(t, err)
		require.NoError(t, elem.AddChild(replacement))

		got, ok := elem.GetAttribute("id")
		require.True(t, ok)
		require.Equal(t, "second", got)
		require.Len(t, elem.Attributes(), 1, "same-named attribute must be replaced, not duplicated")
	})

	// An attribute already parented on one element is detached from it before being
	// spliced onto the new element.
	t.Run("attribute detaches from its previous element", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		src, err := doc.CreateElement("src")
		require.NoError(t, err)
		dst, err := doc.CreateElement("dst")
		require.NoError(t, err)

		err = src.SetAttribute("moved", "v")
		require.NoError(t, err)
		attr := src.Attributes()[0]

		require.NoError(t, dst.AddChild(attr))

		_, ok := src.GetAttribute("moved")
		require.False(t, ok, "attribute must be removed from its previous element")
		require.Empty(t, src.Attributes())

		got, ok := dst.GetAttribute("moved")
		require.True(t, ok)
		require.Equal(t, "v", got)
	})

	// A document accepts an element through AddChild (an element is a valid child of
	// a document node), and the attribute-routing type switch must not block it.
	t.Run("document accepts an element child", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		root, err := doc.CreateElement("root")
		require.NoError(t, err)
		require.NoError(t, doc.AddChild(root))
		require.Equal(t, root, doc.DocumentElement())
	})

	// An attribute has no valid placement on a document and is rejected.
	t.Run("document rejects an attribute", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		attr, err := doc.CreateAttribute("a", "v", nil)
		require.NoError(t, err)

		err = doc.AddChild(attr)
		require.Error(t, err)
		require.ErrorIs(t, err, helium.ErrInvalidOperation)
	})

	// An attribute has no valid placement on a non-element parent (Text) and is
	// rejected with a descriptive %w-wrapped ErrInvalidOperation, not a bare
	// sentinel.
	t.Run("text rejects an attribute", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		text := doc.CreateText([]byte("hello"))
		attr, err := doc.CreateAttribute("a", "v", nil)
		require.NoError(t, err)

		err = text.AddChild(attr)
		require.Error(t, err)
		require.ErrorIs(t, err, helium.ErrInvalidOperation)
		// The message carries context beyond the bare sentinel text.
		require.NotEqual(t, helium.ErrInvalidOperation.Error(), err.Error())
		require.Contains(t, err.Error(), "cannot add")
	})

	// A Text node merges only another Text node; any other operand (here an Element)
	// is rejected with a descriptive %w-wrapped ErrInvalidOperation.
	t.Run("text rejects a non-text child", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		text := doc.CreateText([]byte("hello"))

		child, err := doc.CreateElement("child")
		require.NoError(t, err)
		err = text.AddChild(child)
		require.Error(t, err)
		require.ErrorIs(t, err, helium.ErrInvalidOperation)
		require.NotEqual(t, helium.ErrInvalidOperation.Error(), err.Error())
		require.Contains(t, err.Error(), "cannot add")
	})

	// A Comment node merges only another Comment node; any other operand (here an
	// Element) is rejected with a descriptive %w-wrapped ErrInvalidOperation.
	t.Run("comment rejects a non-comment child", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		comment := doc.CreateComment([]byte("c"))

		child, err := doc.CreateElement("child")
		require.NoError(t, err)
		err = comment.AddChild(child)
		require.Error(t, err)
		require.ErrorIs(t, err, helium.ErrInvalidOperation)
		require.NotEqual(t, helium.ErrInvalidOperation.Error(), err.Error())
		require.Contains(t, err.Error(), "cannot add")
	})

	// A CDATA section carries character data, not child nodes, so every operand is
	// rejected with a descriptive %w-wrapped ErrInvalidOperation.
	t.Run("CDATA section rejects any child", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		cdata := doc.CreateCDATASection([]byte("x"))

		err := cdata.AddChild(doc.CreateText([]byte("y")))
		require.Error(t, err)
		require.ErrorIs(t, err, helium.ErrInvalidOperation)
		require.NotEqual(t, helium.ErrInvalidOperation.Error(), err.Error())
		require.Contains(t, err.Error(), "cannot add")

		// A nil operand is rejected with ErrNilNode, not a panic.
		require.ErrorIs(t, cdata.AddChild(nil), helium.ErrNilNode)
	})

	// A ProcessingInstruction carries its content as a string, not as child nodes,
	// so an attribute has no valid placement on it. Its AddChild override handles
	// the operand itself (never reaching the shared addChild rejection), so it must
	// reject an Attribute operand with a wrapped ErrInvalidOperation and leave the
	// PI unchanged.
	t.Run("PI rejects an attribute", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		pi := doc.CreatePI("target", "data")

		attr, err := doc.CreateAttribute("a", "v", nil)
		require.NoError(t, err)

		err = pi.AddChild(attr)
		require.Error(t, err)
		require.ErrorIs(t, err, helium.ErrInvalidOperation)

		// The PI content is unchanged by the rejected operand.
		require.Equal(t, "data", string(pi.Content()))
	})

	// Regression guard: routing an attribute through AddChild must not leave a
	// stray child element in the serialized output.
	t.Run("attribute serialization shape", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		elem, err := doc.CreateElement("root")
		require.NoError(t, err)
		attr, err := doc.CreateAttribute("k", "v", nil)
		require.NoError(t, err)
		require.NoError(t, elem.AddChild(attr))

		out, err := helium.WriteString(elem)
		require.NoError(t, err)
		require.False(t, strings.Contains(out, "</root><"), "no sibling/child element must follow root")
		require.Contains(t, out, `<root k="v"`)
	})
}

func TestNodeConsistency(t *testing.T) {
	// the reachable typed-nil path: a document with
	// no root element yields a typed-nil *Element from DocumentElement(), and the
	// public node helpers must treat it as nil rather than panicking.
	t.Run("typed-nil node", func(t *testing.T) {
		doc := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)
		de := doc.DocumentElement() // typed-nil *helium.Element for a rootless doc
		require.Nil(t, de, "DocumentElement of an empty document is a nil *Element")

		// n holds a typed-nil *Element: the interface value is non-nil (it carries a
		// type) even though the pointer is nil, which is the case that used to panic.
		var n helium.Node = de

		t.Run("AsNode reports not-ok for a typed nil", func(t *testing.T) {
			got, ok := helium.AsNode[*helium.Element](n)
			require.False(t, ok, "AsNode must not report ok for a typed-nil pointer")
			require.Nil(t, got)
		})

		t.Run("CopyNode returns ErrNilNode", func(t *testing.T) {
			_, err := helium.CopyNode(n, doc)
			require.ErrorIs(t, err, helium.ErrNilNode)
		})

		t.Run("Children yields nothing", func(t *testing.T) {
			count := 0
			for range helium.Children(n) {
				count++
			}
			require.Zero(t, count)
		})

		t.Run("Walk returns ErrNilNode", func(t *testing.T) {
			err := helium.Walk(n, helium.NodeWalkerFunc(func(helium.Node) error { return nil }))
			require.ErrorIs(t, err, helium.ErrNilNode)
		})

		t.Run("UnlinkNode is a no-op", func(t *testing.T) {
			require.NotPanics(t, func() { helium.UnlinkNode(de) })
		})

		t.Run("ParseInNodeContext returns ErrNilNode", func(t *testing.T) {
			_, err := helium.NewParser().ParseInNodeContext(context.Background(), n, []byte("<a/>"))
			require.ErrorIs(t, err, helium.ErrNilNode)
		})
	})

	// the empty-Replace contract: an element's
	// Replace() with no arguments returns ErrInvalidOperation, matching
	// Document.Replace().
	t.Run("empty Replace contract", func(t *testing.T) {
		doc := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)
		el, err := doc.CreateElement("root")
		require.NoError(t, err)

		errNode := el.Replace()
		require.ErrorIs(t, errNode, helium.ErrInvalidOperation)

		errDoc := doc.Replace()
		require.ErrorIs(t, errDoc, helium.ErrInvalidOperation)
	})

	// the matchable sentinels for the guarded
	// mutation operations.
	t.Run("cyclic-node sentinel", func(t *testing.T) {
		t.Run("AddChild self insertion", func(t *testing.T) {
			doc := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)
			el, err := doc.CreateElement("root")
			require.NoError(t, err)
			require.ErrorIs(t, el.AddChild(el), helium.ErrCyclicNode)
		})

		t.Run("AddSibling self insertion", func(t *testing.T) {
			doc := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)
			parent, err := doc.CreateElement("parent")
			require.NoError(t, err)
			child, err := doc.CreateElement("child")
			require.NoError(t, err)
			require.NoError(t, parent.AddChild(child))
			require.ErrorIs(t, child.AddSibling(child), helium.ErrCyclicNode)
		})

		t.Run("Replace with an ancestor", func(t *testing.T) {
			doc := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)
			parent, err := doc.CreateElement("parent")
			require.NoError(t, err)
			child, err := doc.CreateElement("child")
			require.NoError(t, err)
			require.NoError(t, parent.AddChild(child))
			require.ErrorIs(t, child.Replace(parent), helium.ErrCyclicNode)
		})
	})
}

func TestDefensiveCopy(t *testing.T) {
	// the exported Content() on the leaf
	// node types (Text, Comment, CDATASection) returns a defensive copy of the
	// node's internal bytes. Mutating the returned slice must NOT corrupt the DOM,
	// and a subsequent read must still return the original content.
	t.Run("Content returns a copy", func(t *testing.T) {
		doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneExplicitNo)

		const original = "hello world"

		makers := map[string]func() helium.Node{
			"Text": func() helium.Node {
				n := doc.CreateText([]byte(original))
				return n
			},
			"Comment": func() helium.Node {
				n := doc.CreateComment([]byte(original))
				return n
			},
			"CDATASection": func() helium.Node {
				n := doc.CreateCDATASection([]byte(original))
				return n
			},
		}

		for name, make := range makers {
			t.Run(name, func(t *testing.T) {
				n := make()
				require.Equal(t, original, string(n.Content()), "initial content")

				// Mutating the returned slice must not affect the node.
				got := n.Content()
				require.Len(t, got, len(original))
				for i := range got {
					got[i] = 'X'
				}

				// Re-read must return the untouched original.
				require.Equal(t, original, string(n.Content()), "content after caller mutation")

				// Two separate Content() calls must not alias each other either.
				a := n.Content()
				b := n.Content()
				if len(a) > 0 {
					a[0] = 'Z'
					require.NotEqual(t, a[0], b[0], "second Content() call must not alias the first")
				}
			})
		}
	})

	// the leaf constructors copy the
	// caller's input slice on store. Mutating the original input slice AFTER the
	// Create* call must NOT change the node's content (the DOM must not alias the
	// caller's buffer).
	t.Run("leaf constructors copy their input", func(t *testing.T) {
		doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneExplicitNo)

		const original = "hello world"

		makers := map[string]func(buf []byte) helium.Node{
			"Text": func(buf []byte) helium.Node {
				return doc.CreateText(buf)
			},
			"Comment": func(buf []byte) helium.Node {
				return doc.CreateComment(buf)
			},
			"CDATASection": func(buf []byte) helium.Node {
				return doc.CreateCDATASection(buf)
			},
		}

		for name, make := range makers {
			t.Run(name, func(t *testing.T) {
				buf := []byte(original)
				n := make(buf)
				require.Equal(t, original, string(n.Content()), "initial content")

				// Mutate the caller's input slice AFTER constructing the node.
				for i := range buf {
					buf[i] = 'X'
				}

				require.Equal(t, original, string(n.Content()), "content after input-slice mutation")
			})
		}
	})

	// the exported Namespaces() accessor
	// returns a defensive copy of the node's internal nsDefs slice. Mutating the
	// returned slice (overwriting or appending) must NOT corrupt the node's
	// internal namespace state, and a subsequent read must still return the
	// original declarations.
	t.Run("Namespaces returns a copy", func(t *testing.T) {
		doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneExplicitNo)

		elem, err := doc.CreateElement("root")
		require.NoError(t, err)
		require.NoError(t, elem.DeclareNamespace("a", "urn:a"))
		require.NoError(t, elem.DeclareNamespace("b", "urn:b"))

		first := elem.Namespaces()
		require.Len(t, first, 2, "initial namespace count")
		require.Equal(t, "a", first[0].Prefix())
		require.Equal(t, "b", first[1].Prefix())

		// Overwrite an element of the returned slice. This must not change the
		// node's internal state.
		first[0] = nil

		// Append to the returned slice. If the slice aliases the node's internal
		// backing array (and has spare capacity), this could clobber internal
		// state too.
		first = append(first, nil)
		_ = first

		got := elem.Namespaces()
		require.Len(t, got, 2, "namespace count after caller mutation")
		require.NotNil(t, got[0], "first namespace must be untouched")
		require.Equal(t, "a", got[0].Prefix(), "first namespace prefix after caller mutation")
		require.Equal(t, "b", got[1].Prefix(), "second namespace prefix after caller mutation")

		// Two separate Namespaces() calls must not alias each other either.
		a := elem.Namespaces()
		b := elem.Namespaces()
		a[0] = nil
		require.NotNil(t, b[0], "second Namespaces() call must not alias the first")
	})
}
