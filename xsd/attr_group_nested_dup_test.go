package xsd_test

import (
	"strings"
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestAttrGroupNestedRefDuplicate verifies that a duplicate attribute use
// (ag-props-correct.2) introduced through a NESTED xs:attributeGroup ref child is
// detected. parseNamedAttributeGroup formerly recorded only direct xs:attribute
// children, so a group that referenced another group sharing an attribute name
// with the group's own attribute compiled clean even though xmllint rejects it.
func TestAttrGroupNestedRefDuplicate(t *testing.T) {
	t.Parallel()

	const mainXSD = "main.xsd"
	const dup = "Duplicate attribute use"

	compile := func(t *testing.T, src string) string {
		t.Helper()
		fsys := fstest.MapFS{mainXSD: &fstest.MapFile{Data: []byte(src)}}
		data, err := fsys.ReadFile(mainXSD)
		require.NoError(t, err)
		doc, err := helium.NewParser().Parse(t.Context(), data)
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, _ = xsd.NewCompiler().Label(mainXSD).ErrorHandler(collector).FS(fsys).Compile(t.Context(), doc)
		require.NoError(t, collector.Close())
		_, errStr := partitionCompileErrors(collector.Errors())
		return errStr
	}

	// g2 references g1 (which defines x) AND declares its own x: a duplicate.
	t.Run("referenced parent group rejected", func(t *testing.T) {
		t.Parallel()
		errStr := compile(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attributeGroup name="g1">
    <xs:attribute name="x" type="xs:string"/>
  </xs:attributeGroup>
  <xs:attributeGroup name="g2">
    <xs:attributeGroup ref="g1"/>
    <xs:attribute name="x" type="xs:int"/>
  </xs:attributeGroup>
  <xs:complexType name="t">
    <xs:attributeGroup ref="g2"/>
  </xs:complexType>
  <xs:element name="root" type="t"/>
</xs:schema>`)
		require.Contains(t, errStr, dup,
			"expected duplicate-attribute-use across nested ref; got: %q", errStr)
	})

	// Same dup, but the offending group g2 is never referenced by any type. The
	// per-type check never inspects it, so only the group-level flatten catches it.
	t.Run("unreferenced parent group rejected", func(t *testing.T) {
		t.Parallel()
		errStr := compile(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attributeGroup name="g1">
    <xs:attribute name="x" type="xs:string"/>
  </xs:attributeGroup>
  <xs:attributeGroup name="g2">
    <xs:attributeGroup ref="g1"/>
    <xs:attribute name="x" type="xs:int"/>
  </xs:attributeGroup>
  <xs:element name="root" type="xs:string"/>
</xs:schema>`)
		require.Contains(t, errStr, dup,
			"expected duplicate-attribute-use in unreferenced nested ref; got: %q", errStr)
	})

	// Transitive: g3 -> g2 -> g1, all contributing distinct names plus a final
	// collision on x between g1 (deepest) and g3's own x.
	t.Run("transitive nested ref rejected", func(t *testing.T) {
		t.Parallel()
		errStr := compile(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attributeGroup name="g1">
    <xs:attribute name="x" type="xs:string"/>
  </xs:attributeGroup>
  <xs:attributeGroup name="g2">
    <xs:attributeGroup ref="g1"/>
    <xs:attribute name="y" type="xs:int"/>
  </xs:attributeGroup>
  <xs:attributeGroup name="g3">
    <xs:attributeGroup ref="g2"/>
    <xs:attribute name="x" type="xs:int"/>
  </xs:attributeGroup>
  <xs:element name="root" type="xs:string"/>
</xs:schema>`)
		require.Contains(t, errStr, dup,
			"expected duplicate-attribute-use across transitive nested ref; got: %q", errStr)
	})

	// A non-duplicating nested ref must still compile clean (no over-rejection).
	t.Run("non-duplicating nested ref compiles", func(t *testing.T) {
		t.Parallel()
		errStr := compile(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attributeGroup name="g1">
    <xs:attribute name="a" type="xs:string"/>
  </xs:attributeGroup>
  <xs:attributeGroup name="g2">
    <xs:attributeGroup ref="g1"/>
    <xs:attribute name="b" type="xs:int"/>
  </xs:attributeGroup>
  <xs:complexType name="t">
    <xs:attributeGroup ref="g2"/>
  </xs:complexType>
  <xs:element name="root" type="t"/>
</xs:schema>`)
		require.False(t, strings.Contains(errStr, dup),
			"non-duplicating nested ref must compile clean; got: %q", errStr)
	})
}
