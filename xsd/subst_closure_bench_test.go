package xsd

import (
	"fmt"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
)

// withoutSubstClosureCache drops schema's precomputed substitution-group
// closures and returns it, putting every substitutableMembersFor /
// substMemberFor / instanceSubstMembers read back on the UNCACHED walk. It is
// the test-only entry point that gives the paired benchmarks below a baseline
// arm measured from this repository itself rather than from a checkout of the
// base branch.
//
// The uncached arm is a live, exercised code path, not a checked-in copy of an
// old implementation: lookupSubstClosure answers closureUncached whenever
// schema.substClosures is nil, which is exactly what every compile-time caller
// sees, since buildSubstClosures runs only at the very end of compileSchema.
func withoutSubstClosureCache(schema *Schema) *Schema {
	schema.substClosures = nil
	return schema
}

// runValidateBench times repeated validations of doc against schema.
func runValidateBench(b *testing.B, schema *Schema, doc *helium.Document) {
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if err := NewValidator(schema).Validate(b.Context(), doc); err != nil {
			b.Fatal(err)
		}
	}
}

// buildFlatSubstGroupSchema compiles a schema whose root accepts any number
// of <item> children (via <xs:element ref="item"/>) and, when members > 0, a
// flat substitution group of `members` elements each
// substitutionGroup="item".
func buildFlatSubstGroupSchema(b *testing.B, members int) *Schema {
	b.Helper()
	var sb strings.Builder
	sb.WriteString(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="item" type="xs:string"/>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element ref="item" minOccurs="0" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
`)
	for i := range members {
		fmt.Fprintf(&sb, "  <xs:element name=\"m%d\" type=\"xs:string\" substitutionGroup=\"item\"/>\n", i)
	}
	sb.WriteString(`</xs:schema>`)

	doc, err := helium.NewParser().Parse(b.Context(), []byte(sb.String()))
	if err != nil {
		b.Fatal(err)
	}
	schema, err := NewCompiler().Label("bench.xsd").Compile(b.Context(), doc)
	if err != nil {
		b.Fatal(err)
	}
	return schema
}

// buildSubstGroupInstance parses an instance of `children` elements all named
// `name`, wrapped in a <root>.
func buildSubstGroupInstance(b *testing.B, name string, children int) *helium.Document {
	b.Helper()
	var sb strings.Builder
	sb.WriteString("<root>")
	for range children {
		fmt.Fprintf(&sb, "<%s>x</%s>", name, name)
	}
	sb.WriteString("</root>")
	doc, err := helium.NewParser().Parse(b.Context(), []byte(sb.String()))
	if err != nil {
		b.Fatal(err)
	}
	return doc
}

// BenchmarkFlatSubstGroup validates a document of `children` elements — the
// last member of a flat `members`-element substitution group headed by
// "item" (or, at members=0, plain <item> elements against a schema with no
// substitution group at all) — against a root particle
// <xs:element ref="item" maxOccurs="unbounded"/>. Every child that is not the
// head itself forces elemMatchesDeclOrSubst to consult "item"'s substitution
// closure, so the uncached arm's cost scales with children*members while the
// cached arm's scales with children alone.
//
// The cached and uncached arms are PAIRED: identical schema source, identical
// instance, the same compiler, and the only difference is
// withoutSubstClosureCache on the baseline arm's schema. Both arms are
// compiled separately so neither can disturb the other's schema.
//
//	go test ./xsd -run '^$' -bench 'BenchmarkFlatSubstGroup' -benchmem
//
// linux/amd64 (WSL2), AMD Ryzen 9 7900X3D, Go 1.26.6, worst case
// members=32/children=6400:
//
//	cached      8.20 ms/op     7,042,262 B/op     51,529 allocs/op
//	uncached  151.72 ms/op   188,547,353 B/op    339,540 allocs/op
//
// members=0 is the control: there is no substitution group, so both arms do
// the same work — 6.72 ms/op cached vs 6.44 ms/op uncached at children=6400.
func BenchmarkFlatSubstGroup(b *testing.B) {
	for _, members := range []int{0, 1, 8, 32} {
		cached := buildFlatSubstGroupSchema(b, members)
		uncached := withoutSubstClosureCache(buildFlatSubstGroupSchema(b, members))
		name := "item"
		if members > 0 {
			name = fmt.Sprintf("m%d", members-1) // last member: worst case for a linear scan
		}
		for _, children := range []int{100, 400, 1600, 6400} {
			doc := buildSubstGroupInstance(b, name, children)
			b.Run(fmt.Sprintf("cached/members=%d/children=%d", members, children), func(b *testing.B) {
				runValidateBench(b, cached, doc)
			})
			b.Run(fmt.Sprintf("uncached/members=%d/children=%d", members, children), func(b *testing.B) {
				runValidateBench(b, uncached, doc)
			})
		}
	}
}

// buildChoiceSubstGroupSchema compiles a schema whose root is a choice of
// `heads` substitution-group heads (head0..headN-1), each with its own flat
// group of `members` elements substitutionGroup="headI". Validating a child
// against this content model probes every choice branch in the worst case
// (the branch order puts the actual match last), so the uncached arm's cost
// tracks heads*members per child.
func buildChoiceSubstGroupSchema(b *testing.B, heads, members int) *Schema {
	b.Helper()
	var sb strings.Builder
	sb.WriteString(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:choice minOccurs="0" maxOccurs="unbounded">
`)
	for h := range heads {
		fmt.Fprintf(&sb, "        <xs:element ref=\"head%d\"/>\n", h)
	}
	sb.WriteString(`      </xs:choice>
    </xs:complexType>
  </xs:element>
`)
	for h := range heads {
		fmt.Fprintf(&sb, "  <xs:element name=\"head%d\" type=\"xs:string\"/>\n", h)
		for m := range members {
			fmt.Fprintf(&sb, "  <xs:element name=\"h%dm%d\" type=\"xs:string\" substitutionGroup=\"head%d\"/>\n", h, m, h)
		}
	}
	sb.WriteString(`</xs:schema>`)

	doc, err := helium.NewParser().Parse(b.Context(), []byte(sb.String()))
	if err != nil {
		b.Fatal(err)
	}
	schema, err := NewCompiler().Label("bench.xsd").Compile(b.Context(), doc)
	if err != nil {
		b.Fatal(err)
	}
	return schema
}

// BenchmarkChoiceSubstGroup validates 800 children — all the LAST member of
// the LAST head's group, the worst case for a choice's left-to-right probe
// order — against a root <xs:choice> of `heads` substitution-group heads
// with `members` members each. The cached and uncached arms are paired the
// same way as in BenchmarkFlatSubstGroup.
//
//	go test ./xsd -run '^$' -bench 'BenchmarkChoiceSubstGroup' -benchmem
//
// linux/amd64 (WSL2), AMD Ryzen 9 7900X3D, Go 1.26.6, worst case
// heads=16/members=16:
//
//	cached     2.23 ms/op     1,208,514 B/op     30,601 allocs/op
//	uncached  61.31 ms/op    73,113,625 B/op    226,611 allocs/op
func BenchmarkChoiceSubstGroup(b *testing.B) {
	const children = 800
	for _, heads := range []int{1, 4, 16} {
		for _, members := range []int{4, 16} {
			cached := buildChoiceSubstGroupSchema(b, heads, members)
			uncached := withoutSubstClosureCache(buildChoiceSubstGroupSchema(b, heads, members))
			name := fmt.Sprintf("h%dm%d", heads-1, members-1)
			doc := buildSubstGroupInstance(b, name, children)
			b.Run(fmt.Sprintf("cached/heads=%d/members=%d", heads, members), func(b *testing.B) {
				runValidateBench(b, cached, doc)
			})
			b.Run(fmt.Sprintf("uncached/heads=%d/members=%d", heads, members), func(b *testing.B) {
				runValidateBench(b, uncached, doc)
			})
		}
	}
}
