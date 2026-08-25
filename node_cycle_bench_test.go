package helium

import (
	"fmt"
	"testing"
)

// BenchmarkAddChildDeepChain measures building a chain of depth N through
// AddChild, top-down (each new element becomes the single child of the
// previous one). Every AddChild call in the chain would run the cycle guard's
// ancestor walk over the growing chain, so this shape is quadratic unless the
// guard settles a freshly built operand without the walk. attrs adds N
// attributes to each element before linking it: an element carrying attributes
// IS claimed by them, so that arm measures the claim search rather than the
// bare claim-count exit the zero-attribute arm takes.
//
// The "before" arm IS reproducible from this repository, and needs no separate
// baseline file. This benchmark is package helium and calls only API that
// exists on the base, so extracting the base into a scratch tree
// (git archive origin/main | tar -x -C <dir>), copying this file in unmodified
// and running it there yields the before numbers directly. Measured that way
// at N=800/1600/3200/6400: base attrs=0 1.51/5.75/22.4/89.4 ms and attrs=2
// 1.59/5.67/23.7/95.3 ms, against 0.12/0.25/0.27/0.62 ms and
// 0.47/1.01/2.34/5.01 ms here. Wall-clock figures are machine-dependent and
// vary run to run; the growth rate across N is the load-bearing part.
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
