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

// BenchmarkAddSibling measures appending n siblings at several sizes under two
// anchoring styles. Only the AddSibling calls are timed; building the document
// and the elements to append happens with the timer stopped.
//
// The growth SHAPE across the sizes is the point. "nontail" appends through an
// anchor that stops being the tail after the first call, the workload the tail
// jump in addSibling targets; it costs O(n) in total. Walking NextSibling() from
// the anchor instead costs O(n^2), which shows up as a ~4x rise per doubling.
// "tail" always appends at the true tail and is O(n) either way.
func BenchmarkAddSibling(b *testing.B) {
	sizes := []int{500, 1000, 2000, 4000}

	b.Run("nontail", func(b *testing.B) {
		for _, n := range sizes {
			b.Run(fmt.Sprintf("N=%d", n), func(b *testing.B) {
				benchAddSiblingNonTail(b, n)
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
