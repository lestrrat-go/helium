package xsd_test

import (
	"fmt"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// nestedKeyrefSchema declares a keyref ("ItemRef") on "node" that refers to a
// key ("ItemUnique") declared on the CHILD "items" element. "node" nests a
// child "node", so a keyref host at depth D must gather the key table from
// every descendant occurrence in its subtree, and every ancestor host repeats
// that gathering over the same descendants.
const nestedKeyrefSchema = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="items">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" type="xs:string" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
    <xs:unique name="ItemUnique">
      <xs:selector xpath="item"/>
      <xs:field xpath="."/>
    </xs:unique>
  </xs:element>
  <xs:element name="node">
    <xs:complexType>
      <xs:sequence>
        <xs:element ref="items"/>
        <xs:element name="ref" type="xs:string"/>
        <xs:element ref="node" minOccurs="0"/>
      </xs:sequence>
    </xs:complexType>
    <xs:keyref name="ItemRef" refer="ItemUnique">
      <xs:selector xpath="ref"/>
      <xs:field xpath="."/>
    </xs:keyref>
  </xs:element>
</xs:schema>`

// nestedKeyrefItemsBlock returns a k-item <items> block used at every nesting
// level of nestedKeyrefDoc.
func nestedKeyrefItemsBlock(k int) string {
	var b strings.Builder
	b.WriteString("<items>")
	for i := range k {
		fmt.Fprintf(&b, "<item>v%d</item>", i)
	}
	b.WriteString("</items>")
	return b.String()
}

// nestedKeyrefDoc builds a "node" chain depth levels deep, each level carrying
// its own k-item <items> block and a <ref> that resolves against the innermost
// level's key (v0 is present at every level).
func nestedKeyrefDoc(depth, k int) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0"?>`)
	blk := nestedKeyrefItemsBlock(k)
	for range depth {
		b.WriteString("<node>")
		b.WriteString(blk)
		b.WriteString("<ref>v0</ref>")
	}
	for range depth {
		b.WriteString("</node>")
	}
	return b.String()
}

// BenchmarkIDCKeyrefNestedSubtree measures xsd.Validator.Validate over a
// document whose keyref hosts nest, so collectSubtreeKeyTable's descendant
// key-table gathering repeats at every ancestor level (xsd/validate_idc.go).
// Depths were chosen to keep the benchmark runnable while still exposing the
// super-linear growth: doubling depth should roughly double the per-op time
// after the fix, not multiply it ~7x as the unfixed cubic gathering does.
func BenchmarkIDCKeyrefNestedSubtree(b *testing.B) {
	sdoc, err := helium.NewParser().Parse(b.Context(), []byte(nestedKeyrefSchema))
	require.NoError(b, err)
	schema, err := xsd.NewCompiler().Compile(b.Context(), sdoc)
	require.NoError(b, err)

	for _, depth := range []int{100, 400} {
		b.Run(fmt.Sprintf("depth=%d", depth), func(b *testing.B) {
			idoc, err := helium.NewParser().MaxDepth(-1).Parse(b.Context(), []byte(nestedKeyrefDoc(depth, 5)))
			require.NoError(b, err)
			v := xsd.NewValidator(schema)

			b.ResetTimer()
			for range b.N {
				require.NoError(b, v.Validate(b.Context(), idoc))
			}
		})
	}
}
