package helium_test

import (
	"fmt"
	"testing"

	"github.com/lestrrat-go/helium"
)

// buildChainTree builds a document whose root is the head of a chain of n
// nested elements, each the single child of the previous. It returns the
// root, the sole entry point CopyNode needs to copy the whole chain.
func buildChainTree(b *testing.B, n int) *helium.Element {
	b.Helper()
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root, err := doc.CreateElement("root")
	if err != nil {
		b.Fatal(err)
	}
	prev := helium.MutableNode(root)
	for range n {
		cur, err := doc.CreateElement("n")
		if err != nil {
			b.Fatal(err)
		}
		if err := prev.AddChild(cur); err != nil {
			b.Fatal(err)
		}
		prev = cur
	}
	return root
}

// buildFlatTree builds a document whose root has n childless element
// children.
func buildFlatTree(b *testing.B, n int) *helium.Element {
	b.Helper()
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root, err := doc.CreateElement("root")
	if err != nil {
		b.Fatal(err)
	}
	for range n {
		c, err := doc.CreateElement("c")
		if err != nil {
			b.Fatal(err)
		}
		if err := root.AddChild(c); err != nil {
			b.Fatal(err)
		}
	}
	return root
}

// BenchmarkCopyNode measures helium.CopyNode over two tree shapes at several
// sizes: a "chain" (N nested elements, each the single child of the
// previous), which drives CopyNode's bottom-up child-linking through the
// insertion cycle guard once per level, and a "flat" tree (N children of one
// root), whose childless children take the cycle guard's fast exit. The chain
// shape is expected to grow quadratically in node count until the copier's
// child-linking path is changed to skip the guard (a separate change); the
// flat shape stays linear.
func BenchmarkCopyNode(b *testing.B) {
	sizes := []int{200, 400, 800, 1600}

	b.Run("chain", func(b *testing.B) {
		for _, n := range sizes {
			b.Run(fmt.Sprintf("N=%d", n), func(b *testing.B) {
				src := buildChainTree(b, n)
				dst := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
				b.ResetTimer()
				for range b.N {
					if _, err := helium.CopyNode(src, dst); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	})

	b.Run("flat", func(b *testing.B) {
		for _, n := range sizes {
			b.Run(fmt.Sprintf("N=%d", n), func(b *testing.B) {
				src := buildFlatTree(b, n)
				dst := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
				b.ResetTimer()
				for range b.N {
					if _, err := helium.CopyNode(src, dst); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	})
}
