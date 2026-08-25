package xsd

import (
	"fmt"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
)

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
// head itself forces elemMatchesDeclOrSubst to walk "item"'s substitution
// closure, so cost is expected to scale with children*members before the
// closure is cached, and with children alone (independent of members) once
// it is.
func BenchmarkFlatSubstGroup(b *testing.B) {
	for _, members := range []int{0, 1, 8, 32} {
		schema := buildFlatSubstGroupSchema(b, members)
		name := "item"
		if members > 0 {
			name = fmt.Sprintf("m%d", members-1) // last member: worst case for a linear scan
		}
		for _, children := range []int{100, 400, 1600, 6400} {
			doc := buildSubstGroupInstance(b, name, children)
			b.Run(fmt.Sprintf("members=%d/children=%d", members, children), func(b *testing.B) {
				b.ResetTimer()
				for range b.N {
					if err := NewValidator(schema).Validate(b.Context(), doc); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

// buildChoiceSubstGroupSchema compiles a schema whose root is a choice of
// `heads` substitution-group heads (head0..headN-1), each with its own flat
// group of `members` elements substitutionGroup="headI". Validating a child
// against this content model probes every choice branch in the worst case
// (the branch order puts the actual match last), so cost tracks
// heads*members per child before the closure is cached.
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
// with `members` members each.
func BenchmarkChoiceSubstGroup(b *testing.B) {
	const children = 800
	for _, heads := range []int{1, 4, 16} {
		for _, members := range []int{4, 16} {
			schema := buildChoiceSubstGroupSchema(b, heads, members)
			name := fmt.Sprintf("h%dm%d", heads-1, members-1)
			doc := buildSubstGroupInstance(b, name, children)
			b.Run(fmt.Sprintf("heads=%d/members=%d", heads, members), func(b *testing.B) {
				b.ResetTimer()
				for range b.N {
					if err := NewValidator(schema).Validate(b.Context(), doc); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}
