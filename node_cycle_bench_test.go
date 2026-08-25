package helium_test

import (
	"fmt"
	"testing"

	"github.com/lestrrat-go/helium"
)

// BenchmarkAddChildDeepChain measures building a chain of depth N through
// AddChild, top-down (each new element becomes the single child of the
// previous one). Every AddChild call in the chain runs the cycle guard's
// ancestor walk over the growing chain, so this shape is quadratic until the
// guard's childless-operand fast path removes the walk. attrs adds N
// attributes to each element before linking it, to exercise the
// properties-chain fast path alongside the plain childless case.
func BenchmarkAddChildDeepChain(b *testing.B) {
	sizes := []int{800, 1600, 3200, 6400}

	for _, attrs := range []int{0, 2} {
		b.Run(fmt.Sprintf("attrs=%d", attrs), func(b *testing.B) {
			for _, n := range sizes {
				b.Run(fmt.Sprintf("N=%d", n), func(b *testing.B) {
					for range b.N {
						doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
						root, err := doc.CreateElement("root")
						if err != nil {
							b.Fatal(err)
						}
						cur := helium.MutableNode(root)
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
