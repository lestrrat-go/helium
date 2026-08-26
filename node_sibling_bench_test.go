package helium_test

import (
	"fmt"
	"testing"

	"github.com/lestrrat-go/helium"
)

// siblingBenchFixture holds one prepared appending workload: a parent, the
// anchor AddSibling is called on, and the n childless elements to append. The
// benchmark builds it with the timer stopped so only the AddSibling calls are
// measured.
type siblingBenchFixture struct {
	anchor   *helium.Element
	children []*helium.Element
}

// newSiblingBenchFixture builds a parent holding a single anchor child plus n
// unlinked elements to append through that anchor.
func newSiblingBenchFixture(b *testing.B, n int) *siblingBenchFixture {
	b.Helper()
	doc := helium.NewDefaultDocument()
	parent, err := doc.CreateElement("parent")
	if err != nil {
		b.Fatal(err)
	}
	anchor, err := doc.CreateElement("n0")
	if err != nil {
		b.Fatal(err)
	}
	if err := parent.AddChild(anchor); err != nil {
		b.Fatal(err)
	}
	children := make([]*helium.Element, n)
	for i := range n {
		child, err := doc.CreateElement("n")
		if err != nil {
			b.Fatal(err)
		}
		children[i] = child
	}
	return &siblingBenchFixture{anchor: anchor, children: children}
}

// appendThroughAnchor appends every prepared element through the fixed anchor.
func (f *siblingBenchFixture) appendThroughAnchor(b *testing.B) {
	b.Helper()
	for _, child := range f.children {
		if err := f.anchor.AddSibling(child); err != nil {
			b.Fatal(err)
		}
	}
}

// appendThroughTail appends every prepared element through the current tail,
// re-anchoring on each appended node.
func (f *siblingBenchFixture) appendThroughTail(b *testing.B) {
	b.Helper()
	anchor := f.anchor
	for _, child := range f.children {
		if err := anchor.AddSibling(child); err != nil {
			b.Fatal(err)
		}
		anchor = child
	}
}

// newSiblingBenchMiddleFixture builds a parent already holding n children and
// anchors on the one at n/2 — a child that is neither parent.firstChild nor
// parent.lastChild — plus n more unlinked elements to append through it.
func newSiblingBenchMiddleFixture(b *testing.B, n int) *siblingBenchFixture {
	b.Helper()
	doc := helium.NewDefaultDocument()
	parent, err := doc.CreateElement("parent")
	if err != nil {
		b.Fatal(err)
	}
	var anchor *helium.Element
	for i := range n {
		child, err := doc.CreateElement("n")
		if err != nil {
			b.Fatal(err)
		}
		if err := parent.AddChild(child); err != nil {
			b.Fatal(err)
		}
		if i == n/2 {
			anchor = child
		}
	}
	children := make([]*helium.Element, n)
	for i := range n {
		child, err := doc.CreateElement("n")
		if err != nil {
			b.Fatal(err)
		}
		children[i] = child
	}
	return &siblingBenchFixture{anchor: anchor, children: children}
}

// docSiblingBenchFixture holds the same workload with a *Document as the parent.
// A document's child list is an ordinary sibling chain, and a document is the one
// parent that can be handed a node claiming it without being on that chain
// (CopyExtSubset), so this arm is what shows the O(1) resolution is declined for
// that CONDITION and not for the type.
type docSiblingBenchFixture struct {
	anchor   *helium.Comment
	children []*helium.Comment
}

// newDocSiblingBenchFixture builds a document holding a single anchor comment
// plus n unlinked comments to append through that anchor.
func newDocSiblingBenchFixture(b *testing.B, n int) *docSiblingBenchFixture {
	b.Helper()
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	anchor := doc.CreateComment([]byte("c0"))
	if err := doc.AddChild(anchor); err != nil {
		b.Fatal(err)
	}
	children := make([]*helium.Comment, n)
	for i := range n {
		children[i] = doc.CreateComment([]byte("c"))
	}
	return &docSiblingBenchFixture{anchor: anchor, children: children}
}

// newDocSiblingBenchMiddleFixture builds a document already holding n comments
// and anchors on the one at n/2, plus n more to append through it.
func newDocSiblingBenchMiddleFixture(b *testing.B, n int) *docSiblingBenchFixture {
	b.Helper()
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	var anchor *helium.Comment
	for i := range n {
		child := doc.CreateComment([]byte("c"))
		if err := doc.AddChild(child); err != nil {
			b.Fatal(err)
		}
		if i == n/2 {
			anchor = child
		}
	}
	children := make([]*helium.Comment, n)
	for i := range n {
		children[i] = doc.CreateComment([]byte("c"))
	}
	return &docSiblingBenchFixture{anchor: anchor, children: children}
}

// appendThroughAnchor appends every prepared comment through the fixed anchor.
func (f *docSiblingBenchFixture) appendThroughAnchor(b *testing.B) {
	b.Helper()
	for _, child := range f.children {
		if err := f.anchor.AddSibling(child); err != nil {
			b.Fatal(err)
		}
	}
}

// benchAddSiblingNonTail measures n appends through a FIXED first child. After
// the first append that anchor is no longer the tail, so every later call takes
// addSibling's non-tail path: the one that jumps to parent.lastChild instead of
// walking NextSibling() down the whole chain.
func benchAddSiblingNonTail(b *testing.B, n int) {
	for range b.N {
		b.StopTimer()
		fixture := newSiblingBenchFixture(b, n)
		b.StartTimer()
		fixture.appendThroughAnchor(b)
	}
}

// benchAddSiblingTail measures the same n appends through a MOVING anchor that
// is always the tail, which never needed a walk. It is the control: the non-tail
// case should track this shape once the tail jump is in place.
func benchAddSiblingTail(b *testing.B, n int) {
	for range b.N {
		b.StopTimer()
		fixture := newSiblingBenchFixture(b, n)
		b.StartTimer()
		fixture.appendThroughTail(b)
	}
}

// benchAddSiblingMiddle measures n appends through a FIXED middle child of an
// n-child parent. That anchor is neither firstChild nor lastChild, so no
// pointer comparison proves its membership: chainMember walks prev back to the
// chain head on every call, and the walk grows as the chain does. This is the
// shape that stays quadratic; it is here so the claim in BenchmarkAddSibling's
// comment is measured rather than assumed.
func benchAddSiblingMiddle(b *testing.B, n int) {
	for range b.N {
		b.StopTimer()
		fixture := newSiblingBenchMiddleFixture(b, n)
		b.StartTimer()
		fixture.appendThroughAnchor(b)
	}
}

// benchAddSiblingDocNonTail is benchAddSiblingNonTail with a *Document parent.
func benchAddSiblingDocNonTail(b *testing.B, n int) {
	for range b.N {
		b.StopTimer()
		fixture := newDocSiblingBenchFixture(b, n)
		b.StartTimer()
		fixture.appendThroughAnchor(b)
	}
}

// benchAddSiblingDocMiddle is benchAddSiblingMiddle with a *Document parent.
func benchAddSiblingDocMiddle(b *testing.B, n int) {
	for range b.N {
		b.StopTimer()
		fixture := newDocSiblingBenchMiddleFixture(b, n)
		b.StartTimer()
		fixture.appendThroughAnchor(b)
	}
}

// BenchmarkAddSibling measures appending n siblings at several sizes under three
// anchoring styles. Only the AddSibling calls are timed; building the document
// and the elements to append happens with the timer stopped.
//
// The growth SHAPE across the sizes is the point, and it differs by arm because
// the O(1) tail jump in addSibling is only reached once chainMember has proven
// the ANCHOR is on the parent's child chain.
//
//   - "tail" always appends at the true tail. The anchor's next is nil, so
//     addSibling links there directly without consulting chainMember at all,
//     and the total is O(n).
//   - "nontail" appends through the FIRST child, which stops being the tail
//     after the first call. The anchor is parent.firstChild, again a pointer
//     comparison, so the tail jump applies and the total is O(n) — where
//     walking NextSibling() from the anchor would cost O(n^2).
//   - "middle" appends through a fixed middle child of an n-child parent. No
//     pointer comparison proves that anchor's membership, so chainMember walks
//     prev back to the chain head on every call: the anchor's distance behind
//     the head grows with the chain, and the total stays O(n^2), showing up as
//     a ~4x rise per doubling. The tail jump does not make this shape linear.
//     What it does buy is a large constant factor over the full sibling walk,
//     because the prev walk covers only the chain BEHIND the anchor while the
//     NextSibling() walk covered the whole chain ahead of it.
//
// The "docnontail" and "docmiddle" arms repeat the first two with a *Document
// parent, which is the parent a document-scale append actually uses and which
// tracks the element arms exactly: the O(1) resolution is declined for the
// CONDITION that makes a document's tail record untrustworthy (an off-chain
// child claim, which only CopyExtSubset creates), never for the type.
func BenchmarkAddSibling(b *testing.B) {
	sizes := []int{500, 1000, 2000, 4000}

	b.Run("nontail", func(b *testing.B) {
		for _, n := range sizes {
			b.Run(fmt.Sprintf("N=%d", n), func(b *testing.B) {
				benchAddSiblingNonTail(b, n)
			})
		}
	})

	b.Run("middle", func(b *testing.B) {
		for _, n := range sizes {
			b.Run(fmt.Sprintf("N=%d", n), func(b *testing.B) {
				benchAddSiblingMiddle(b, n)
			})
		}
	})

	b.Run("docnontail", func(b *testing.B) {
		for _, n := range sizes {
			b.Run(fmt.Sprintf("N=%d", n), func(b *testing.B) {
				benchAddSiblingDocNonTail(b, n)
			})
		}
	})

	b.Run("docmiddle", func(b *testing.B) {
		for _, n := range sizes {
			b.Run(fmt.Sprintf("N=%d", n), func(b *testing.B) {
				benchAddSiblingDocMiddle(b, n)
			})
		}
	})

	b.Run("tail", func(b *testing.B) {
		for _, n := range sizes {
			b.Run(fmt.Sprintf("N=%d", n), func(b *testing.B) {
				benchAddSiblingTail(b, n)
			})
		}
	})
}
