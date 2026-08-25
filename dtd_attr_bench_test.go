package helium_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
)

// attrsPerElem is how many <!ATTLIST> declarations each benchmark element
// carries. A subset of n elements therefore holds n*attrsPerElem declarations
// in total, against attrsPerElem declarations on any one element — the two
// quantities an element-keyed lookup can cost.
const attrsPerElem = 4

// buildAttrDTD builds a document whose internal subset declares n elements,
// each with attrsPerElem attribute declarations. It returns the subset, the
// receiver AttributesForElement is called on.
func buildAttrDTD(b *testing.B, n int) *helium.DTD {
	b.Helper()
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	dtd, err := doc.CreateInternalSubset("root", "", "")
	if err != nil {
		b.Fatal(err)
	}
	for i := range n {
		elem := fmt.Sprintf("e%d", i)
		for j := range attrsPerElem {
			attr := fmt.Sprintf("a%d", j)
			if _, err := dtd.AddAttributeDecl(elem, attr, enum.AttrCDATA, enum.AttrDefaultImplied, "", nil); err != nil {
				b.Fatal(err)
			}
		}
	}
	return dtd
}

// buildAttrDocument renders a valid document whose internal subset declares n
// elements with attrsPerElem attributes each, and whose root holds exactly one
// instance of every declared element. Validating it performs n element-keyed
// attribute-declaration lookups against a subset holding n*attrsPerElem
// declarations.
func buildAttrDocument(n int) []byte {
	var buf strings.Builder
	buf.WriteString("<?xml version=\"1.0\"?>\n<!DOCTYPE root [\n<!ELEMENT root ANY>\n")
	for i := range n {
		fmt.Fprintf(&buf, "<!ELEMENT e%d EMPTY>\n", i)
		for j := range attrsPerElem {
			fmt.Fprintf(&buf, "<!ATTLIST e%d a%d CDATA #IMPLIED>\n", i, j)
		}
	}
	buf.WriteString("]>\n<root>")
	for i := range n {
		fmt.Fprintf(&buf, "<e%d/>", i)
	}
	buf.WriteString("</root>")
	return []byte(buf.String())
}

// BenchmarkDTDAttrLookup measures DTD.AttributesForElement in isolation: one
// lookup per declared element name against a subset of N elements holding
// N*attrsPerElem declarations in total. Serving the lookup from the
// per-element index makes a single lookup cost O(declarations on that element)
// instead of O(declarations in the whole subset), so the per-op cost over all
// N names grows linearly in N rather than quadratically.
//
// Run it with:
//
//	go test -run '^$' -bench 'BenchmarkDTDAttr' -count=6 .
//
// Obtain the baseline by copying THIS FILE unchanged into an extraction of the
// pre-change tree and running the identical command there:
//
//	git archive origin/main | tar -x -C /some/dir
//	cp dtd_attr_bench_test.go /some/dir/
//	cd /some/dir && GOWORK=off go test -run '^$' -bench 'BenchmarkDTDAttr' -count=6 .
//
// The file uses only exported API, so it compiles unmodified against both
// trees. GOWORK=off keeps a surrounding go.work from redirecting the module
// back at the working checkout.
//
// Measured on linux/amd64, AMD Ryzen 9 7900X3D, Go 1.26.6, best of 6 runs,
// against base commit 918a7d79:
//
//	         base (origin/main)          this change            speedup
//	N=100      270.1 µs/op   300 allocs    4.0 µs/op  100 allocs   66.8x
//	N=200    1,019.7 µs/op   599 allocs    8.4 µs/op  200 allocs  121.8x
//	N=400    5,025.1 µs/op 1,199 allocs   19.2 µs/op  400 allocs  261.1x
//	N=800   20,448.2 µs/op 2,399 allocs   35.3 µs/op  800 allocs  579.1x
//
// Base time roughly quadruples for each doubling of N (quadratic); this
// change's time doubles (linear). Allocations move the same way: the base
// grows a fresh result slice per lookup (several allocations each as it
// reallocs), this change clones an exactly-sized index slice (one each).
func BenchmarkDTDAttrLookup(b *testing.B) {
	for _, n := range []int{100, 200, 400, 800} {
		b.Run(fmt.Sprintf("N=%d", n), func(b *testing.B) {
			dtd := buildAttrDTD(b, n)
			names := make([]string, n)
			for i := range n {
				names[i] = fmt.Sprintf("e%d", i)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				for _, name := range names {
					if got := dtd.AttributesForElement(name); len(got) != attrsPerElem {
						b.Fatalf("expected %d declarations for %s, got %d", attrsPerElem, name, len(got))
					}
				}
			}
		})
	}
}

// BenchmarkDTDAttrValidate measures the same lookup through the path that
// actually drives it in production: a validating parse. The fixture declares N
// elements with attrsPerElem attributes each and instantiates every one, so
// validation performs N element-keyed lookups against N*attrsPerElem
// declarations. Parsing itself is linear in N, so the quadratic term in the
// base numbers is the attribute-declaration scan.
//
// Same command and same baseline procedure as BenchmarkDTDAttrLookup.
//
// Measured on linux/amd64, AMD Ryzen 9 7900X3D, Go 1.26.6, best of 6 runs,
// against base commit 918a7d79:
//
//	         base (origin/main)             this change            speedup
//	N=100      758.6 µs/op  2,426 allocs    448.6 µs/op  2,637 allocs  1.7x
//	N=200    1,976.0 µs/op  4,742 allocs    874.6 µs/op  5,155 allocs  2.3x
//	N=400    6,125.1 µs/op  9,370 allocs  1,729.8 µs/op 10,186 allocs  3.5x
//	N=800   22,736.0 µs/op 18,617 allocs  3,701.3 µs/op 20,235 allocs  6.1x
//
// This change's time doubles per doubling of N (linear, and now dominated by
// the parse itself); the base climbs faster as N grows and the declaration
// scan overtakes the parse. The index costs roughly N extra allocations per
// parse — one slice per declared element name, built once while the subset is
// read — which is why this change's allocation count is slightly the higher of
// the two even as its time falls.
func BenchmarkDTDAttrValidate(b *testing.B) {
	for _, n := range []int{100, 200, 400, 800} {
		b.Run(fmt.Sprintf("N=%d", n), func(b *testing.B) {
			src := buildAttrDocument(n)
			p := helium.NewParser().ValidateDTD(true)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if _, err := p.Parse(b.Context(), src); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
