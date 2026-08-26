package helium

import (
	"fmt"
	"testing"
)

// BenchmarkAddChildDeepChain measures building a chain of depth N through
// AddChild, top-down, so each new element becomes the single child of the
// previous one. Every call runs the cycle guard against an insertion point one
// level deeper than the last, which is quadratic whenever the guard answers by
// walking that chain. attrs gives each element that many attributes before it
// is linked: an element carrying attributes IS claimed by them, so that arm
// exercises the claimant search rather than the empty-slot exit.
//
// The before numbers are reproducible from this repository and need no
// baseline file, because this benchmark calls only API the base already has:
// extract the base into a scratch tree (git archive origin/main | tar -x -C
// <dir>), copy this file in unmodified, and run it there. Measured that way at
// N=800/1600/3200/6400, base attrs=0 was 1.40/5.32/20.6/85.1 ms and attrs=2
// was 1.76/6.04/22.9/90.1 ms, against 118/190/383/710 us and
// 491/917/2624/3431 us here. Wall-clock figures are machine-dependent; the
// growth rate across N is the load-bearing part.
func BenchmarkAddChildDeepChain(b *testing.B) {
	sizes := []int{800, 1600, 3200, 6400}

	for _, attrs := range []int{0, 2} {
		b.Run(fmt.Sprintf("attrs=%d", attrs), func(b *testing.B) {
			for _, n := range sizes {
				b.Run(fmt.Sprintf("N=%d", n), func(b *testing.B) {
					for range b.N {
						doc := NewDocument("1.0", "UTF-8", StandaloneImplicitNo)
						root, err := doc.CreateElement("root")
						if err != nil {
							b.Fatal(err)
						}
						cur := MutableNode(root)
						for range n {
							e, err := doc.CreateElement("e")
							if err != nil {
								b.Fatal(err)
							}
							for a := range attrs {
								if err := e.SetAttribute(fmt.Sprintf("a%d", a), "v"); err != nil {
									b.Fatal(err)
								}
							}
							if err := cur.AddChild(e); err != nil {
								b.Fatal(err)
							}
							cur = e
						}
					}
				})
			}
		})
	}
}
