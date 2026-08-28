package helium_test

import (
	"math"
	"os"
	"testing"
	"time"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
	"github.com/stretchr/testify/require"
)

// TestAddChildResolvesAnOwnedTail drives the shapes that safe API can build in
// which a parent's recorded lastChild is not the last node that actually claims
// that parent. AddChild must land the new child after the tail its own child
// list reaches, exactly as AddSibling already does, rather than trusting the
// recorded pointer.
func TestAddChildResolvesAnOwnedTail(t *testing.T) {
	t.Parallel()

	// A copied external subset claims the document as its parent while living
	// only in extSubset, so appending through it records a tail that is on no
	// child list. A later AddChild must not link behind that record and lose the
	// reachable list.
	t.Run("recorded tail is off the child list", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		dtdPath := dir + "/ext.dtd"
		require.NoError(t, os.WriteFile(dtdPath, []byte(`<!ELEMENT root (#PCDATA)>`), 0600))

		xml := `<?xml version="1.0"?>
<!DOCTYPE root SYSTEM "` + dtdPath + `">
<root/>`
		src, err := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).FS(helium.PermissiveFS()).Parse(t.Context(), []byte(xml))
		require.NoError(t, err)
		require.NotNil(t, src.ExtSubset())

		dst := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
		helium.CopyExtSubset(src, dst)
		ext := dst.ExtSubset()
		require.NotNil(t, ext)
		require.Equal(t, helium.Node(dst), ext.Parent())

		// Records dst.lastChild = comment while dst.firstChild stays nil.
		comment := dst.CreateComment([]byte("stray"))
		require.NoError(t, ext.AddSibling(comment))

		root, err := dst.CreateElement("root")
		require.NoError(t, err)
		require.NoError(t, dst.AddChild(root))

		require.Equal(t, helium.Node(root), dst.DocumentElement(),
			"the document element must be reachable after the append")
		require.Equal(t, helium.Node(root), dst.FirstChild(),
			"an empty reachable child list must start at the appended node")
	})

	// stringToNodeList materializes an entity's replacement children with
	// firstChild set and lastChild nil. AddChild must append after the existing
	// children instead of discarding them.
	t.Run("recorded tail is nil while a child list exists", func(t *testing.T) {
		t.Parallel()

		doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
		dtd, err := doc.CreateInternalSubset("root", "", "")
		require.NoError(t, err)
		_, err = dtd.AddEntity("e", enum.InternalGeneralEntity, "", "", "val")
		require.NoError(t, err)

		root, err := doc.CreateElement("root")
		require.NoError(t, err)
		require.NoError(t, doc.SetDocumentElement(root))
		require.NoError(t, root.SetParsedAttribute("a", "&e;"))

		ent, ok := doc.GetEntity("e")
		require.True(t, ok)
		first := ent.FirstChild()
		require.NotNil(t, first, "the entity carries its replacement children")

		extra, err := doc.CreateElement("extra")
		require.NoError(t, err)
		require.NoError(t, ent.AddChild(extra))

		require.Equal(t, first, ent.FirstChild(),
			"the existing child list must survive the append")
		var names []string
		for child := range helium.Children(ent) {
			names = append(names, child.Type().String())
		}
		require.Contains(t, names, "ElementNode", "the appended child must be reachable")
	})

	// CreateReference hands the EntityRef the DTD's shared Entity as its child
	// while that Entity's parent stays the DTD. Appending the Entity into its own
	// reference must not link it after itself.
	t.Run("operand is the parent's own foreign child", func(t *testing.T) {
		t.Parallel()

		doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
		dtd, err := doc.CreateInternalSubset("root", "", "")
		require.NoError(t, err)
		ent, err := dtd.AddEntity("e", enum.InternalGeneralEntity, "", "", "val")
		require.NoError(t, err)

		ref, err := doc.CreateReference("e")
		require.NoError(t, err)
		require.Equal(t, helium.Node(ent), ref.FirstChild())

		_ = ref.AddChild(ent)

		require.NotEqual(t, helium.Node(ent), ent.NextSibling(),
			"a node must never become its own next sibling")
		require.NotEqual(t, helium.Node(ent), ent.PrevSibling(),
			"a node must never become its own previous sibling")
	})

	// The same EntityRef shape, with a DIFFERENT node appended. The reference's
	// shared Entity child still belongs to the DTD, so its sibling links must not
	// be used as an append point for the reference.
	t.Run("a foreign first child is not an append anchor", func(t *testing.T) {
		t.Parallel()

		doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
		dtd, err := doc.CreateInternalSubset("root", "", "")
		require.NoError(t, err)
		ent, err := dtd.AddEntity("e1", enum.InternalGeneralEntity, "", "", "x")
		require.NoError(t, err)
		next, err := dtd.AddEntity("e2", enum.InternalGeneralEntity, "", "", "y")
		require.NoError(t, err)
		dtdChildren := []helium.Node{ent, next}
		require.Equal(t, dtdChildren, collectChildren(dtd))

		ref, err := doc.CreateReference("e1")
		require.NoError(t, err)
		require.Equal(t, helium.Node(ent), ref.FirstChild())

		comment := doc.CreateComment([]byte("added"))
		require.NoError(t, ref.AddChild(comment))

		require.Equal(t, helium.Node(ref), comment.Parent(),
			"the appended node must belong to the requested parent")
		require.Equal(t, helium.Node(comment), ref.FirstChild())
		require.Equal(t, helium.Node(comment), ref.LastChild())
		require.Equal(t, dtdChildren, collectChildren(dtd),
			"appending to the reference must not change the DTD declaration chain")
		require.Equal(t, helium.Node(next), ent.NextSibling())
		require.Equal(t, helium.Node(ent), next.PrevSibling())
	})
}

func collectChildren(parent helium.Node) []helium.Node {
	var children []helium.Node
	for child := range helium.Children(parent) {
		children = append(children, child)
	}
	return children
}

// appendChildren times n AddChild calls onto a freshly created element in doc.
// The element is well formed and holds no stale record of its own, so the cost
// is entirely the cost of resolving the append point.
func appendChildren(t *testing.T, doc *helium.Document, n int) time.Duration {
	t.Helper()

	host, err := doc.CreateElement("host")
	require.NoError(t, err)
	children := make([]*helium.Comment, n)
	for i := range children {
		children[i] = doc.CreateComment([]byte("c"))
	}

	start := time.Now()
	for _, child := range children {
		if err := host.AddChild(child); err != nil {
			require.NoError(t, err)
		}
	}
	return time.Since(start)
}

// TestAppendCostIsLinearBesideAnOffChainClaim pins the SCOPE of the off-chain
// claim record. A parent that has been handed a child claiming it from off its
// child list cannot trust its own lastChild, but every OTHER parent still can,
// including every other parent in the same document. Reading the record from the
// owning document instead of from the parent makes one claim anywhere turn every
// later append in that document into a walk of the whole child list, which is
// quadratic over N appends.
//
// The assertion is a SHAPE, not a time: doubling the number of appends must
// roughly double the cost (ratio near 2), not quadruple it (ratio near 4). A
// loaded machine moves both measurements together, and the best of several
// attempts is taken so a single scheduling stall cannot fail the test.
func TestAppendCostIsLinearBesideAnOffChainClaim(t *testing.T) {
	// Not parallel: it measures elapsed time.
	doc, err := helium.NewParser().Parse(t.Context(),
		[]byte(`<!DOCTYPE r [<!ENTITY e "x">]><r/>`))
	require.NoError(t, err)

	// The sequence that used to record a document-wide claim: the reference's
	// firstChild is the DTD's shared Entity, which claims the DTD and not the
	// reference, so the append finds no tail the reference owns.
	ref, err := doc.CreateReference("e")
	require.NoError(t, err)
	require.NoError(t, ref.AddChild(doc.CreateComment([]byte("stray"))))

	const n = 8000
	appendChildren(t, doc, n) // warm up allocator and caches

	best := math.Inf(1)
	for range 3 {
		single := appendChildren(t, doc, n)
		double := appendChildren(t, doc, 2*n)
		if single <= 0 {
			continue
		}
		if ratio := float64(double) / float64(single); ratio < best {
			best = ratio
		}
	}

	require.Less(t, best, 3.0,
		"doubling the appends must roughly double the cost, not quadruple it; "+
			"a ratio near 4 means every append is walking the whole child list")
}
