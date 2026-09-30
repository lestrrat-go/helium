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
	// sibling guard, which detects it, exactly as for a longer sibling cycle.
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

// A tree parsed with SubstituteEntities(false) keeps each entity reference as
// an EntityRef node. Content must expand every reference through the entity's
// parsed children, recursively, the way libxml2's xmlNodeGetContent does, so it
// returns the same text a SubstituteEntities(true) parse stores. The expected
// strings match `xmllint --xpath 'string(...)'`.
func TestContentExpandsEntityReferences(t *testing.T) {
	t.Parallel()

	const src = `<!DOCTYPE r [` +
		`<!ENTITY e "x">` +
		`<!ENTITY f "y">` +
		`<!ENTITY g "a&f;b">` +
		`<!ENTITY h "&#38;#60;">` +
		`<!ENTITY p "p&amp;q">` +
		`<!ENTITY m "a<b>c&f;</b>d">` +
		`]>` +
		`<r n="1&g;2" h="1&h;2" p="&p;">` +
		`<t>1&e;2</t><g>1&g;2</g><m>1&m;2</m><p>&p;</p><h>1&h;2</h>` +
		`</r>`

	want := map[string]string{
		"t": "1x2",
		"g": "1ayb2",
		"m": "1acyd2",
		"p": "p&q",
		"h": "1<2",
	}
	wantAttr := map[string]string{
		"n": "1ayb2",
		"h": "1<2",
		"p": "p&q",
	}

	for _, substitute := range []bool{false, true} {
		doc, err := helium.NewParser().SubstituteEntities(substitute).Parse(t.Context(), []byte(src))
		require.NoError(t, err)
		root := doc.DocumentElement()
		for child := range helium.Children(root) {
			value, ok := want[child.Name()]
			require.True(t, ok, "unexpected child %s", child.Name())
			require.Equal(t, value, string(child.Content()),
				"Content of <%s> (SubstituteEntities(%t))", child.Name(), substitute)
		}
		for name, value := range wantAttr {
			attr := root.GetAttributeNodeNS(name, "")
			require.NotNil(t, attr, "attribute %s", name)
			require.Equal(t, value, string(attr.Content()),
				"Content of @%s (SubstituteEntities(%t))", name, substitute)
			require.Equal(t, attr.Value(), string(attr.Content()),
				"Content and Value of @%s agree (SubstituteEntities(%t))", name, substitute)
		}
		require.Equal(t, "1x21ayb21acyd2p&q1<2", string(root.Content()),
			"Content of the root (SubstituteEntities(%t))", substitute)
	}

	// Content of the EntityRef itself is the expanded entity value.
	t.Run("entity reference node", func(t *testing.T) {
		t.Parallel()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
		require.NoError(t, err)
		g := doc.DocumentElement().FirstChild().NextSibling()
		require.Equal(t, "g", g.Name())
		ref := g.FirstChild().NextSibling()
		require.Equal(t, helium.EntityRefNode, ref.Type())
		require.Equal(t, "ayb", string(ref.Content()))
	})

	// A reference built through the tree API resolves its entity the same way.
	t.Run("reference built through the tree API", func(t *testing.T) {
		t.Parallel()
		doc, err := helium.NewParser().Parse(t.Context(),
			[]byte(`<!DOCTYPE r [<!ENTITY e "x"><!ENTITY g "a&e;b">]><r>&g;</r>`))
		require.NoError(t, err)
		attr, err := doc.CreateAttribute("a", "1&g;2", nil)
		require.NoError(t, err)
		require.Equal(t, "1axb2", string(attr.Content()))
		ref, err := doc.CreateReference("g")
		require.NoError(t, err)
		require.Equal(t, "axb", string(ref.Content()))
	})

	// libxml2's xmlBufGetChildContent adds only Text and CDATA text and
	// descends into other children, so a comment or PI adds nothing at any
	// depth, whether it sits inside an entity value or in the tree itself. A
	// SubstituteEntities(true) parse copies an entity's comment and PI into the
	// tree, so both parse modes read the same text.
	t.Run("comments and PIs", func(t *testing.T) {
		t.Parallel()
		const src = `<!DOCTYPE r [` +
			`<!ENTITY c "x<!--k-->y<?p q?>z">` +
			`<!ENTITY n "a<b><!--k-->c<?p q?></b>d">` +
			`<!ENTITY o "1&c;2">` +
			`]>` +
			`<r><c>&c;</c><n>&n;</n><o>&o;</o><t>1<!--top-->2<?pi v?>3</t>` +
			`<d>1<e>2<!--deep-->3<?pi w?></e>4</d></r>`
		want := map[string]string{
			"c": "xyz",
			"n": "acd",
			"o": "1xyz2",
			"t": "123",
			"d": "1234",
		}
		for _, substitute := range []bool{false, true} {
			doc, err := helium.NewParser().SubstituteEntities(substitute).Parse(t.Context(), []byte(src))
			require.NoError(t, err)
			root := doc.DocumentElement()
			for child := range helium.Children(root) {
				value, ok := want[child.Name()]
				require.True(t, ok, "unexpected child %s", child.Name())
				require.Equal(t, value, string(child.Content()),
					"Content of <%s> (SubstituteEntities(%t))", child.Name(), substitute)
			}
			require.Equal(t, "xyzacd1xyz21231234", string(root.Content()),
				"Content of the root (SubstituteEntities(%t))", substitute)
		}

		doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
		require.NoError(t, err)
		ref := doc.DocumentElement().FirstChild().FirstChild()
		require.Equal(t, helium.EntityRefNode, ref.Type())
		require.Equal(t, "xyz", string(ref.Content()), "Content of the reference")

		attr, err := doc.CreateAttribute("a", "1&c;2", nil)
		require.NoError(t, err)
		require.Equal(t, "1xyz2", string(attr.Content()), "Content of an attribute built through the tree API")
		require.Equal(t, attr.Value(), string(attr.Content()), "Content and Value of the attribute agree")
	})

	// An element whose only child is a comment or PI has no content, while the
	// comment or PI node itself still returns its own text (libxml2:
	// xmlNodeGetContent on XML_COMMENT_NODE / XML_PI_NODE).
	t.Run("only a comment or PI child", func(t *testing.T) {
		t.Parallel()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<r><a><!--c--></a><b><?p d?></b></r>`))
		require.NoError(t, err)
		a := doc.DocumentElement().FirstChild()
		require.Nil(t, a.Content(), "Content of <a>")
		require.Equal(t, "c", string(a.FirstChild().Content()), "Content of the comment")
		b := a.NextSibling()
		require.Nil(t, b.Content(), "Content of <b>")
		require.Equal(t, "d", string(b.FirstChild().Content()), "Content of the PI")
	})

	// Content of the document node leaves out the comments and PIs around and
	// inside the document element.
	t.Run("document node", func(t *testing.T) {
		t.Parallel()
		doc, err := helium.NewParser().Parse(t.Context(),
			[]byte(`<?p a?><!--b--><r>x<!--c-->y<?q d?></r><!--e--><?s f?>`))
		require.NoError(t, err)
		require.Equal(t, "xy", string(doc.Content()))
	})

	// An attribute built through the tree API can hold a comment or PI child.
	// Its Content leaves them out and agrees with Value.
	t.Run("attribute with a comment or PI child", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		attr, err := doc.CreateAttribute("a", "1", nil)
		require.NoError(t, err)
		require.NoError(t, attr.AddChild(doc.CreateComment([]byte("c"))))
		require.NoError(t, attr.AddChild(doc.CreateText([]byte("2"))))
		require.NoError(t, attr.AddChild(doc.CreatePI("p", "d")))
		require.Equal(t, "12", string(attr.Content()))
		require.Equal(t, attr.Value(), string(attr.Content()))
	})
}

// CharacterData returns only the character data a node holds directly, with
// entity references expanded: element children, elements inside an entity, and
// comments and PIs at any depth add nothing. It is the same for a document
// parsed with or without entity substitution.
func TestCharacterData(t *testing.T) {
	t.Parallel()

	const src = `<!DOCTYPE r [` +
		`<!ENTITY e "x">` +
		`<!ENTITY f "y">` +
		`<!ENTITY g "a&f;b">` +
		`<!ENTITY h "&#38;#60;">` +
		`<!ENTITY m "a<b>c&f;</b>d">` +
		`<!ENTITY c "x<!--k-->y<?p q?>z">` +
		`]>` +
		`<r a="1&g;2">` +
		`<t>1&e;2</t><g>1&g;2</g><m>1&m;2</m><c>&c;</c><k>1<!--top-->2<?pi v?>3<![CDATA[4]]></k><h>1&h;2</h><z/>` +
		`</r>`

	want := map[string]string{
		"t": "1x2",
		"g": "1ayb2",
		"m": "1ad2",
		"c": "xyz",
		"k": "1234",
		"h": "1<2",
		"z": "",
	}

	for _, substitute := range []bool{false, true} {
		doc, err := helium.NewParser().SubstituteEntities(substitute).Parse(t.Context(), []byte(src))
		require.NoError(t, err)
		root := doc.DocumentElement()
		require.Empty(t, helium.CharacterData(root), "an element with only element children (SubstituteEntities(%t))", substitute)
		for child := range helium.Children(root) {
			value, ok := want[child.Name()]
			require.True(t, ok, "unexpected child %s", child.Name())
			require.Equal(t, value, helium.CharacterData(child),
				"CharacterData of <%s> (SubstituteEntities(%t))", child.Name(), substitute)
		}
		attr := root.GetAttributeNodeNS("a", "")
		require.NotNil(t, attr)
		require.Equal(t, "1ayb2", helium.CharacterData(attr), "CharacterData of @a (SubstituteEntities(%t))", substitute)
	}

	t.Run("leaf nodes", func(t *testing.T) {
		t.Parallel()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
		require.NoError(t, err)
		root := doc.DocumentElement()

		m := root.FirstChild().NextSibling().NextSibling()
		require.Equal(t, "m", m.Name())
		require.Equal(t, "1", helium.CharacterData(m.FirstChild()), "a Text node is its own text")
		ref := m.FirstChild().NextSibling()
		require.Equal(t, helium.EntityRefNode, ref.Type())
		require.Equal(t, "ad", helium.CharacterData(ref), "an entity reference is its expansion's character data")

		k := m.NextSibling().NextSibling()
		require.Equal(t, "k", k.Name())
		comment := k.FirstChild().NextSibling()
		require.Equal(t, helium.CommentNode, comment.Type())
		require.Empty(t, helium.CharacterData(comment), "a comment holds no character data")
		require.Equal(t, "4", helium.CharacterData(k.LastChild()), "a CDATA section is its own text")
	})
}
