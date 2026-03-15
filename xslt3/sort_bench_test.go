package xslt3

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
)

// benchSortEnv holds pre-built test data for sort benchmarks.
type benchSortEnv struct {
	ctx     context.Context
	ec      *execContext
	nodes   []helium.Node
	items   xpath3.Sequence
	sortKey []*SortKey
}

func newBenchSortEnv(b *testing.B, n int, sortKeys []*SortKey) *benchSortEnv {
	b.Helper()

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root, err := doc.CreateElement("root")
	if err != nil {
		b.Fatal(err)
	}
	if err := doc.AddChild(root); err != nil {
		b.Fatal(err)
	}

	nodes := make([]helium.Node, n)
	for i := range n {
		elem, err := doc.CreateElement("item")
		if err != nil {
			b.Fatal(err)
		}
		text, err := doc.CreateText([]byte(strconv.Itoa(n - i)))
		if err != nil {
			b.Fatal(err)
		}
		if err := elem.AddChild(text); err != nil {
			b.Fatal(err)
		}
		if err := root.AddChild(elem); err != nil {
			b.Fatal(err)
		}
		nodes[i] = elem
	}

	ss := &Stylesheet{
		namespaces: map[string]string{},
	}

	ec := &execContext{
		stylesheet:  ss,
		sourceDoc:   doc,
		resultDoc:   doc,
		contextNode: root,
		currentNode: root,
		globalVars:  map[string]xpath3.Sequence{},
	}

	items := make(xpath3.Sequence, len(nodes))
	for i, node := range nodes {
		items[i] = xpath3.NodeItem{Node: node}
	}

	return &benchSortEnv{
		ctx:     context.Background(),
		ec:      ec,
		nodes:   nodes,
		items:   items,
		sortKey: sortKeys,
	}
}

func mustCompileSortXPath(b *testing.B, expr string) *xpath3.Expression {
	b.Helper()
	e, err := xpath3.Compile(expr)
	if err != nil {
		b.Fatalf("compiling %q: %v", expr, err)
	}
	return e
}

func mustCompileSortAVT(b *testing.B, s string) *AVT {
	b.Helper()
	avt, err := compileAVT(s, nil)
	if err != nil {
		b.Fatalf("compiling AVT %q: %v", s, err)
	}
	return avt
}

func BenchmarkSortNodes(b *testing.B) {
	textKey := []*SortKey{{
		Select: mustCompileSortXPath(b, "."),
	}}
	numericKey := []*SortKey{{
		Select:   mustCompileSortXPath(b, "."),
		DataType: mustCompileSortAVT(b, "number"),
	}}
	autoNumericKey := []*SortKey{{
		Select: mustCompileSortXPath(b, "number(.)"),
	}}

	for _, size := range []int{100, 1000, 10000} {
		b.Run(fmt.Sprintf("TextAsc/%d", size), func(b *testing.B) {
			env := newBenchSortEnv(b, size, textKey)
			b.ResetTimer()
			b.ReportAllocs()
			for range b.N {
				work := make([]helium.Node, len(env.nodes))
				copy(work, env.nodes)
				if _, err := sortNodes(env.ctx, env.ec, work, env.sortKey); err != nil {
					b.Fatal(err)
				}
			}
		})

		b.Run(fmt.Sprintf("NumericAsc/%d", size), func(b *testing.B) {
			env := newBenchSortEnv(b, size, numericKey)
			b.ResetTimer()
			b.ReportAllocs()
			for range b.N {
				work := make([]helium.Node, len(env.nodes))
				copy(work, env.nodes)
				if _, err := sortNodes(env.ctx, env.ec, work, env.sortKey); err != nil {
					b.Fatal(err)
				}
			}
		})

		b.Run(fmt.Sprintf("AutoNumericAsc/%d", size), func(b *testing.B) {
			env := newBenchSortEnv(b, size, autoNumericKey)
			b.ResetTimer()
			b.ReportAllocs()
			for range b.N {
				work := make([]helium.Node, len(env.nodes))
				copy(work, env.nodes)
				if _, err := sortNodes(env.ctx, env.ec, work, env.sortKey); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkSortItems(b *testing.B) {
	textKey := []*SortKey{{
		Select: mustCompileSortXPath(b, "."),
	}}

	for _, size := range []int{100, 1000} {
		b.Run(fmt.Sprintf("AtomicText/%d", size), func(b *testing.B) {
			env := newBenchSortEnv(b, size, textKey)
			b.ResetTimer()
			b.ReportAllocs()
			for range b.N {
				work := make(xpath3.Sequence, len(env.items))
				copy(work, env.items)
				if _, err := sortItems(env.ctx, env.ec, work, env.sortKey); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkSortNodesMultiKey(b *testing.B) {
	threeKeys := []*SortKey{
		{Select: mustCompileSortXPath(b, ".")},
		{
			Select:   mustCompileSortXPath(b, "."),
			DataType: mustCompileSortAVT(b, "number"),
		},
		{
			Select: mustCompileSortXPath(b, "."),
			Order:  mustCompileSortAVT(b, "descending"),
		},
	}

	for _, size := range []int{100, 1000} {
		b.Run(fmt.Sprintf("ThreeKey/%d", size), func(b *testing.B) {
			env := newBenchSortEnv(b, size, threeKeys)
			b.ResetTimer()
			b.ReportAllocs()
			for range b.N {
				work := make([]helium.Node, len(env.nodes))
				copy(work, env.nodes)
				if _, err := sortNodes(env.ctx, env.ec, work, env.sortKey); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
