package helium_test

import (
	"context"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
	"github.com/stretchr/testify/require"
)

func TestAddChildShape(t *testing.T) {
	// elem.AddChild(attr) must route the attribute into the element's property
	// list, NOT the child list: it appears in Attributes()/GetAttribute, is absent
	// from Children(), and serializes as an attribute, never as a child element.
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
	// public node helpers must treat it as nil and never panic.
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

		// An operand with an EMPTY child list can still be named as a parent
		// from outside every slot it owns. Document.stringToNodeList leaves an
		// entity referenced from an attribute value with a firstChild and no
		// lastChild; an append onto that shape overwrites firstChild and
		// detaches the child that was there while the child goes on naming the
		// entity, and unlinking the replacement empties the child list again.
		// Adding the entity under its own off-chain claimant would close a
		// two-node parent-pointer loop, so the guard must refuse it.
		t.Run("AddChild whose operand is claimed off its child list", func(t *testing.T) {
			doc, err := helium.NewParser().Parse(t.Context(), []byte(`<!DOCTYPE d [<!ENTITY e "xy">]><d a="&e;"/>`))
			require.NoError(t, err)

			ent, ok := doc.GetEntity("e")
			require.True(t, ok)
			require.NotNil(t, ent.FirstChild(), "the attribute-value expansion is the entity's child")
			require.Nil(t, ent.LastChild(), "and it was recorded without a lastChild")

			elem, err := doc.CreateElement("claimant")
			require.NoError(t, err)
			expansion, ok := ent.FirstChild().(helium.MutableNode)
			require.True(t, ok)
			require.NoError(t, expansion.Replace(elem))
			require.Equal(t, helium.Node(ent), elem.Parent())

			comment := doc.CreateComment([]byte("filler"))
			require.NoError(t, ent.AddChild(comment))
			helium.UnlinkNode(comment)
			require.Nil(t, ent.FirstChild())
			require.Nil(t, ent.LastChild())
			require.Equal(t, helium.Node(ent), elem.Parent(), "the detached child still claims the entity")

			require.ErrorIs(t, elem.AddChild(ent), helium.ErrCyclicNode)
			require.Equal(t, helium.Node(ent), elem.Parent(), "the refused insertion leaves both nodes as they were")
			require.NotEqual(t, helium.Node(elem), ent.Parent())
		})

		// A child pointer is not an ancestor edge. An attribute value holding an
		// entity reference expands to a reference node whose child is the SHARED
		// entity, and that entity's parent stays the DTD, so the entity is on no
		// chain of parent pointers running down from the element that carries the
		// attribute. Linking that element under the entity closes no loop, and the
		// guard must allow it. This is the only shape here an ordinary parse
		// produces on its own.
		t.Run("AddChild whose operand holds a foreign entity child in an attribute", func(t *testing.T) {
			doc, err := helium.NewParser().Parse(t.Context(), []byte(
				`<!DOCTYPE root [<!ENTITY e "val">]><root><child a="&e;"/></root>`))
			require.NoError(t, err)

			ent, ok := doc.GetEntity("e")
			require.True(t, ok)

			root := doc.DocumentElement()
			require.NotNil(t, root)
			elem, ok := root.FirstChild().(*helium.Element)
			require.True(t, ok)
			require.Nil(t, elem.FirstChild(), "the element carrying the attribute has no children of its own")

			attrs := elem.Attributes()
			require.Len(t, attrs, 1)
			ref := attrs[0].FirstChild()
			require.Equal(t, helium.EntityRefNode, ref.Type())
			require.Equal(t, helium.Node(ent), ref.FirstChild(), "the reference's child is the shared entity")
			require.Equal(t, helium.DTDNode, ent.Parent().Type(), "which the DTD, not the reference, owns")

			require.NoError(t, ent.AddChild(elem))
			require.Equal(t, helium.Node(ent), elem.Parent())
		})

		// The chain from the insertion point up to the operand leaves a child
		// list at EVERY hop: the operand is claimed by its attribute, and the
		// element under that attribute is claimed by an attribute of its own.
		// Both hops are parent edges the ancestor walk steps in reverse, so the
		// insertion closes a loop and all three guarded entry points must refuse
		// it.
		t.Run("guarded insertions whose operand is claimed through a nested attribute", func(t *testing.T) {
			build := func(t *testing.T) (*helium.Element, *helium.Attribute) {
				t.Helper()

				doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneExplicitNo)
				cur, err := doc.CreateElement("cur")
				require.NoError(t, err)
				require.NoError(t, cur.SetAttribute("a", "v"))
				attrs := cur.Attributes()
				require.Len(t, attrs, 1)

				mid, err := doc.CreateElement("mid")
				require.NoError(t, err)
				require.NoError(t, attrs[0].AddChild(mid))
				require.NoError(t, mid.SetAttribute("p", "x"))
				deep := mid.Attributes()
				require.Len(t, deep, 1)

				require.Nil(t, cur.FirstChild(), "the operand carries no children of its own")
				require.Equal(t, helium.Node(mid), deep[0].Parent())
				require.Equal(t, helium.Node(attrs[0]), mid.Parent())
				require.Equal(t, helium.Node(cur), attrs[0].Parent())

				return cur, deep[0]
			}

			t.Run("AddChild", func(t *testing.T) {
				cur, deep := build(t)
				require.ErrorIs(t, deep.AddChild(cur), helium.ErrCyclicNode)
				require.Nil(t, cur.Parent(), "the refused insertion leaves the operand unlinked")
			})

			t.Run("AddSibling", func(t *testing.T) {
				cur, deep := build(t)
				sib, err := deep.OwnerDocument().CreateElement("sib")
				require.NoError(t, err)
				require.NoError(t, deep.AddChild(sib))
				require.ErrorIs(t, sib.AddSibling(cur), helium.ErrCyclicNode)
				require.Nil(t, cur.Parent())
			})

			t.Run("Replace", func(t *testing.T) {
				cur, deep := build(t)
				victim, err := deep.OwnerDocument().CreateElement("victim")
				require.NoError(t, err)
				require.NoError(t, deep.AddChild(victim))
				require.ErrorIs(t, victim.Replace(cur), helium.ErrCyclicNode)
				require.Nil(t, cur.Parent())
			})
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

// sharedEntityFixture builds a document whose DTD declares two general entities
// (so the Entity nodes are siblings in the DTD declaration list) and returns a
// root element plus an entity reference to the first entity, attached under
// root. An entity reference's child is the shared first Entity node, whose
// sibling pointers belong to the DTD list — the shape that lets a naive sibling
// walk wander out of the reference's own children into unrelated declarations.
func sharedEntityFixture(t *testing.T) (root *helium.Element, ref *helium.EntityRef, ent helium.Node) {
	t.Helper()

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	dtd, err := doc.CreateInternalSubset("root", "", "")
	require.NoError(t, err)
	e1, err := dtd.AddEntity("e1", enum.InternalGeneralEntity, "", "", "x")
	require.NoError(t, err)
	_, err = dtd.AddEntity("e2", enum.InternalGeneralEntity, "", "", "y")
	require.NoError(t, err)

	root, err = doc.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, doc.SetDocumentElement(root))

	ref, err = doc.CreateReference("e1")
	require.NoError(t, err)
	require.Equal(t, e1, ref.FirstChild(), "reference's child is the shared first Entity node")
	require.NoError(t, root.AddChild(ref))

	// The second entity is the first entity's sibling in the DTD declaration
	// list, so a walk that followed raw sibling pointers past the shared Entity
	// would reach it.
	require.Equal(t, "e2", e1.NextSibling().Name(),
		"the second entity is the first's DTD sibling — the foreign spill target")

	return root, ref, e1
}

func TestCycleGuards(t *testing.T) {
	t.Parallel()

	// the cycle that the ancestor-only
	// guard cannot see: an entity reference's child is the shared Entity node, whose
	// parent pointer stays the DTD (mirroring libxml2 / Document.CreateReference).
	// Because the Entity's parent is NOT the reference, adding that reference back
	// under the Entity forms a child-pointer cycle Entity -> ref -> Entity that the
	// ancestor walk (which follows PARENT pointers from the insertion point) never
	// detects. AddChild must reject it so downstream tree walkers cannot loop.
	t.Run("AddChild rejects an entity child cycle", func(t *testing.T) {
		doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
		dtd, err := doc.CreateInternalSubset("root", "", "")
		require.NoError(t, err)
		ent, err := dtd.AddEntity("e", enum.InternalGeneralEntity, "", "", "x")
		require.NoError(t, err)

		// CreateReference links the shared Entity as the reference's child without
		// setting the Entity's parent to the reference.
		ref, err := doc.CreateReference("e")
		require.NoError(t, err)
		require.Equal(t, ent, ref.FirstChild(), "reference's child is the shared Entity node")

		// ref's child is ent, so adding ref under ent closes a child-pointer cycle.
		err = ent.AddChild(ref)
		require.Error(t, err, "adding a reference under its own Entity child must be rejected")
		require.ErrorContains(t, err, "cannot add a node as a child of itself or one of its descendants")

		// The tree must be untouched: ent must not have gained ref as a child.
		require.Nil(t, ent.FirstChild(), "Entity must not gain the reference as a child")
		require.Nil(t, ent.LastChild(), "Entity must not gain the reference as a child")
	})

	// against over-rejection: a
	// reference whose Entity child does NOT reach the insertion parent is a normal,
	// legal insertion and must succeed. This is the shape produced when parsing
	// <root>&e;</root>.
	t.Run("AddChild allows a legitimate entity reference", func(t *testing.T) {
		doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
		dtd, err := doc.CreateInternalSubset("root", "", "")
		require.NoError(t, err)
		_, err = dtd.AddEntity("e", enum.InternalGeneralEntity, "", "", "x")
		require.NoError(t, err)

		root, err := doc.CreateElement("root")
		require.NoError(t, err)
		require.NoError(t, doc.SetDocumentElement(root))

		ref, err := doc.CreateReference("e")
		require.NoError(t, err)

		require.NoError(t, root.AddChild(ref), "a reference whose Entity does not reach root is a legal child")
		require.Equal(t, ref, root.FirstChild(), "reference must be attached under root")
	})

	// Walk applies the
	// owned-boundary rule: descending into a reference's shared Entity child and
	// then advancing must NOT follow the Entity's sibling pointer into the DTD's
	// unrelated declarations.
	t.Run("Walk stays within the subtree across a shared entity", func(t *testing.T) {
		root, _, _ := sharedEntityFixture(t)

		var visited []string
		err := helium.Walk(root, helium.NodeWalkerFunc(func(n helium.Node) error {
			visited = append(visited, n.Name())
			return nil
		}))
		require.NoError(t, err)
		require.Equal(t, []string{"root", "e1", "e1"}, visited,
			"Walk visits root, the reference (named for e1), and the shared Entity — not the foreign e2 sibling")
		require.NotContains(t, visited, "e2", "Walk must not spill into the DTD's other entity declarations")
	})

	// Walk returns ErrWalkCycle, and never reports
	// SUCCESS, on a corrupt ONE-node sibling self-loop: a single child
	// whose next pointer points at itself (c.next == c). nextWalkSibling must NOT
	// silently terminate the self-loop — the duplicate flows back to the per-frame
	// seenChildren set, which detects it, exactly as for a longer sibling cycle.
	t.Run("Walk rejects a self sibling loop", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		parent, err := doc.CreateElement("parent")
		require.NoError(t, err)
		c, err := doc.CreateElement("c")
		require.NoError(t, err)
		require.NoError(t, parent.AddChild(c))

		// Corrupt the sibling list into a one-node self-loop: c.next = c.
		helium.UnsafeSetNextSiblingForTesting(c, c)

		err = helium.Walk(parent, helium.NodeWalkerFunc(func(helium.Node) error { return nil }))
		require.ErrorIs(t, err, helium.ErrWalkCycle,
			"Walk must return ErrWalkCycle on a one-node sibling self-loop, not report success")
	})

	// the requirement that Walk does not
	// switch to a global visited set: two references to the same entity form a DAG
	// where the shared Entity node is reached on two different paths, and Walk must
	// visit it on each occurrence, deduplicating nothing away.
	t.Run("Walk visits a shared entity twice", func(t *testing.T) {
		doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
		dtd, err := doc.CreateInternalSubset("root", "", "")
		require.NoError(t, err)
		ent, err := dtd.AddEntity("e", enum.InternalGeneralEntity, "", "", "x")
		require.NoError(t, err)

		root, err := doc.CreateElement("root")
		require.NoError(t, err)
		require.NoError(t, doc.SetDocumentElement(root))

		ref1, err := doc.CreateReference("e")
		require.NoError(t, err)
		ref2, err := doc.CreateReference("e")
		require.NoError(t, err)
		require.Equal(t, ent, ref1.FirstChild())
		require.Equal(t, ent, ref2.FirstChild())
		require.NoError(t, root.AddChild(ref1))
		require.NoError(t, root.AddChild(ref2))

		var entityVisits int
		err = helium.Walk(root, helium.NodeWalkerFunc(func(n helium.Node) error {
			if n.Type() == helium.EntityNode {
				entityVisits++
			}
			return nil
		}))
		require.NoError(t, err)
		require.Equal(t, 2, entityVisits,
			"the shared Entity reached via two references must be visited twice — no global dedup")
	})

	// Children and ChildElements do not
	// follow a foreign child's sibling pointers out of the reference's own list.
	t.Run("Children respect the owned boundary", func(t *testing.T) {
		_, ref, ent := sharedEntityFixture(t)

		var kids []helium.Node
		for c := range helium.Children(ref) {
			kids = append(kids, c)
		}
		require.Equal(t, []helium.Node{ent}, kids,
			"Children(ref) yields only the shared Entity, stopping at the owned boundary")
	})

	// Descendants stays within the
	// reference's own subtree across the shared Entity child.
	t.Run("Descendants respect the owned boundary", func(t *testing.T) {
		_, ref, ent := sharedEntityFixture(t)

		var got []helium.Node
		for d := range helium.Descendants(ref) {
			got = append(got, d)
		}
		require.Equal(t, []helium.Node{ent}, got,
			"Descendants(ref) yields only the shared Entity, not the DTD siblings")
	})

	// the aggregating Content() of an
	// entity reference returns only its shared Entity's content and does not spill
	// into the DTD's following declarations.
	t.Run("Content stays within the owned boundary", func(t *testing.T) {
		_, ref, _ := sharedEntityFixture(t)

		require.Equal(t, []byte("x"), ref.Content(),
			"Content(ref) is the shared Entity's text, not concatenated with foreign DTD siblings")
	})

	// the aggregating Content()
	// terminates when a child's sibling pointer forms a cycle.
	t.Run("Content terminates on a cyclic sibling list", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		root, err := doc.CreateElement("root")
		require.NoError(t, err)
		txt := doc.CreateText([]byte("a"))
		require.NoError(t, root.AddChild(txt))

		// Corrupt the sibling list into a self-cycle.
		helium.UnsafeSetNextSiblingForTesting(txt, txt)

		require.Equal(t, []byte("a"), root.Content(),
			"Content must terminate on a cyclic sibling list instead of looping forever")
	})
}
