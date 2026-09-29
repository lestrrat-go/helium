package c14n_test

import (
	"os"
	"strconv"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/c14n"
	"github.com/lestrrat-go/helium/xpath1"
	"github.com/stretchr/testify/require"
)

func benchmarkC14NDocument(b *testing.B, path string) *helium.Document {
	b.Helper()
	data, err := os.ReadFile(path)
	require.NoError(b, err)
	doc, err := helium.NewParser().Parse(b.Context(), data)
	require.NoError(b, err)
	return doc
}

func BenchmarkCanonicalize(b *testing.B) {
	for _, input := range []struct {
		name string
		path string
	}{
		{"C14NFixture", "../testdata/libxml2-compat/c14n/exc-without-comments/test/test-0.xml"},
		{"NVDEntries", "../testdata/libxml2-compat/schemas/test/nvdcve_0.xml"},
	} {
		doc := benchmarkC14NDocument(b, input.path)
		for _, tc := range []struct {
			name string
			mode c14n.Mode
		}{
			{"Inclusive10", c14n.C14N10},
			{"Exclusive10", c14n.ExclusiveC14N10},
		} {
			b.Run(input.name+"/"+tc.name, func(b *testing.B) {
				canonicalizer := c14n.NewCanonicalizer(tc.mode)
				var output []byte
				var err error
				b.ReportAllocs()
				b.ResetTimer()
				for range b.N {
					output, err = canonicalizer.CanonicalizeTo(doc)
					if err != nil {
						b.Fatal(err)
					}
				}
				b.StopTimer()
				require.NotEmpty(b, output)
			})
		}
	}
}

// benchmarkModes are the modes the generated-input benchmarks compare.
var benchmarkModes = []struct {
	name string
	mode c14n.Mode
}{
	{"Inclusive10", c14n.C14N10},
	{"Exclusive10", c14n.ExclusiveC14N10},
}

// benchmarkNamespaceDecls returns k prefixed namespace declarations.
func benchmarkNamespaceDecls(k int) string {
	var sb strings.Builder
	for i := range k {
		sb.WriteString(` xmlns:p` + strconv.Itoa(i) + `="http://example.com/ns/` + strconv.Itoa(i) + `"`)
	}
	return sb.String()
}

// benchmarkDeepDocument builds a chain of depth elements under a root that
// declares k namespaces.
func benchmarkDeepDocument(b *testing.B, depth, k int) *helium.Document {
	b.Helper()
	var sb strings.Builder
	sb.WriteString("<r" + benchmarkNamespaceDecls(k) + ">")
	for range depth {
		sb.WriteString(`<e a="1">`)
	}
	sb.WriteString("x")
	for range depth {
		sb.WriteString("</e>")
	}
	sb.WriteString("</r>")
	doc, err := helium.NewParser().MaxDepth(depth+16).Parse(b.Context(), []byte(sb.String()))
	require.NoError(b, err)
	return doc
}

// benchmarkFlatDocument builds n sibling elements, each with one child, under
// a root that declares k namespaces.
func benchmarkFlatDocument(b *testing.B, n, k int) *helium.Document {
	b.Helper()
	var sb strings.Builder
	sb.WriteString("<r" + benchmarkNamespaceDecls(k) + ">")
	for range n {
		sb.WriteString(`<e a="1"><f>x</f></e>`)
	}
	sb.WriteString("</r>")
	doc, err := helium.NewParser().Parse(b.Context(), []byte(sb.String()))
	require.NoError(b, err)
	return doc
}

func benchmarkCanonicalizeLoop(b *testing.B, canonicalizer c14n.Canonicalizer, doc *helium.Document) {
	b.Helper()
	var output []byte
	var err error
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		output, err = canonicalizer.CanonicalizeTo(doc)
		if err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	require.NotEmpty(b, output)
}

// BenchmarkCanonicalizeDeep measures whole-document canonicalization of a deep
// element chain. The cost should grow linearly with depth.
func BenchmarkCanonicalizeDeep(b *testing.B) {
	for _, depth := range []int{256, 2048} {
		doc := benchmarkDeepDocument(b, depth, 4)
		for _, tc := range benchmarkModes {
			b.Run("Depth"+strconv.Itoa(depth)+"/"+tc.name, func(b *testing.B) {
				benchmarkCanonicalizeLoop(b, c14n.NewCanonicalizer(tc.mode), doc)
			})
		}
	}
}

// BenchmarkCanonicalizeManyNamespaces measures whole-document canonicalization
// of 10k elements under a root that declares many namespaces. The cost should
// not depend on the number of root bindings.
func BenchmarkCanonicalizeManyNamespaces(b *testing.B) {
	for _, k := range []int{2, 200} {
		doc := benchmarkFlatDocument(b, 5000, k)
		for _, tc := range benchmarkModes {
			b.Run("Bindings"+strconv.Itoa(k)+"/"+tc.name, func(b *testing.B) {
				benchmarkCanonicalizeLoop(b, c14n.NewCanonicalizer(tc.mode), doc)
			})
		}
	}
}

// BenchmarkCanonicalizeNodeSet measures node-set canonicalization with the full
// namespace axis selected, 2000 elements and 50 root bindings.
func BenchmarkCanonicalizeNodeSet(b *testing.B) {
	doc := benchmarkFlatDocument(b, 1000, 50)
	compiled, err := xpath1.Compile("//. | //@* | //namespace::*")
	require.NoError(b, err)
	result, err := xpath1.NewEvaluator().Evaluate(b.Context(), compiled, doc)
	require.NoError(b, err)
	for _, tc := range benchmarkModes {
		b.Run(tc.name, func(b *testing.B) {
			benchmarkCanonicalizeLoop(b, c14n.NewCanonicalizer(tc.mode).NodeSet(result.NodeSet), doc)
		})
	}
}
