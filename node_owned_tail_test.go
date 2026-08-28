package helium_test

import (
	"os"
	"testing"
	"time"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
	"github.com/stretchr/testify/require"
)

const appendCostRuns = 7

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

// TestAppendCostIsLinearBesideAnOffChainClaim guards the per-parent scope of
// off-chain child claims. CopyExtSubset gives the document such a claim, but an
// element the document owns must still resolve appends through its first child
// from the element's own last-child record. If the claim were read per document,
// every append after the first would walk the growing sibling list and doubling
// the workload would quadruple the cost.
func TestAppendCostIsLinearBesideAnOffChainClaim(t *testing.T) {
	const (
		small    = 4000
		large    = 8000
		maxRatio = 3.2
	)

	src := documentWithExternalSubset(t)
	smallCost := bestAppendCostBesideOffChainClaim(t, src, small)
	largeCost := bestAppendCostBesideOffChainClaim(t, src, large)
	ratio := float64(largeCost) / float64(smallCost)

	t.Logf("append cost beside off-chain claim: n=%d -> %s, n=%d -> %s, ratio=%.2f",
		small, smallCost, large, largeCost, ratio)
	require.LessOrEqual(t, ratio, maxRatio,
		"append cost looks quadratic: doubling a linear workload should stay below a %.1fx cost increase", maxRatio)
}

func documentWithExternalSubset(t *testing.T) *helium.Document {
	t.Helper()

	dtdPath := t.TempDir() + "/ext.dtd"
	require.NoError(t, os.WriteFile(dtdPath, []byte(`<!ELEMENT root (#PCDATA)>`), 0600))
	xml := `<?xml version="1.0"?>
<!DOCTYPE root SYSTEM "` + dtdPath + `">
<root/>`
	doc, err := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).FS(helium.PermissiveFS()).Parse(t.Context(), []byte(xml))
	require.NoError(t, err)
	require.NotNil(t, doc.ExtSubset())
	return doc
}

func bestAppendCostBesideOffChainClaim(t *testing.T, src *helium.Document, n int) time.Duration {
	t.Helper()

	best := time.Duration(-1)
	for range appendCostRuns {
		doc := helium.NewDefaultDocument()
		helium.CopyExtSubset(src, doc)
		require.Equal(t, helium.Node(doc), doc.ExtSubset().Parent())

		parent, err := doc.CreateElement("parent")
		require.NoError(t, err)
		require.NoError(t, doc.AddChild(parent))
		anchor, err := doc.CreateElement("anchor")
		require.NoError(t, err)
		require.NoError(t, parent.AddChild(anchor))

		children := make([]*helium.Element, n)
		for i := range n {
			children[i], err = doc.CreateElement("child")
			require.NoError(t, err)
		}

		start := time.Now()
		for _, child := range children {
			if err = anchor.AddSibling(child); err != nil {
				break
			}
		}
		elapsed := time.Since(start)
		require.NoError(t, err)
		require.Equal(t, helium.Node(children[n-1]), parent.LastChild())
		if best < 0 || elapsed < best {
			best = elapsed
		}
	}
	return best
}

func collectChildren(parent helium.Node) []helium.Node {
	var children []helium.Node
	for child := range helium.Children(parent) {
		children = append(children, child)
	}
	return children
}
