package xsd_test

import (
	"strings"
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestRedefineAttrGroupRefSet verifies that an xs:redefine override REBUILDS the
// nested attribute-group ref set (c.attrGroupRefChildren) for the group it
// replaces. The Phase-A group's nested refs must not leak into the override, and
// the override's own non-self nested refs must be honored. Before the fix the
// override replaced c.schema.attrGroups but never touched attrGroupRefChildren,
// so checkAttrGroupDuplicates flattened the stale Phase-A ref set: old nested
// refs leaked (causing false duplicates) and new nested refs were ignored
// (missing real ag-props-correct.2 duplicates).
func TestRedefineAttrGroupRefSet(t *testing.T) {
	t.Parallel()

	compile := func(t *testing.T, fsys fstest.MapFS, mainXSD string) string {
		t.Helper()
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

	// The Phase-A group 'g' nests a ref to 'gOld' (attribute "a"). The override
	// of 'g' declares "a" directly and nests NO group ref. If the stale Phase-A
	// ref to 'gOld' leaked, the override's direct "a" would collide with the
	// leaked gOld."a" and a spurious duplicate would be reported. The override
	// must compile clean.
	t.Run("phase-A nested ref does not leak into override", func(t *testing.T) {
		t.Parallel()
		const (
			mainXSD = "leak_main.xsd"
			baseXSD = "leak_base.xsd"
		)
		fsys := fstest.MapFS{
			mainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:redefine schemaLocation="leak_base.xsd">
    <xs:attributeGroup name="g">
      <xs:attribute name="a" type="xs:string"/>
    </xs:attributeGroup>
  </xs:redefine>
  <xs:complexType name="t">
    <xs:attributeGroup ref="g"/>
  </xs:complexType>
  <xs:element name="root" type="t"/>
</xs:schema>`)},
			baseXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attributeGroup name="gOld">
    <xs:attribute name="a" type="xs:string"/>
  </xs:attributeGroup>
  <xs:attributeGroup name="g">
    <xs:attributeGroup ref="gOld"/>
  </xs:attributeGroup>
</xs:schema>`)},
		}
		errStr := compile(t, fsys, mainXSD)
		require.False(t, strings.Contains(errStr, "Duplicate attribute use"),
			"stale Phase-A nested ref must not leak into the override; got: %q", errStr)
	})

	// The override of 'g' declares "a" directly AND nests a ref to 'gNew' which
	// also declares "a". This is a real ag-props-correct.2 duplicate introduced
	// by the override's NEW nested ref, which must be detected.
	t.Run("override nested ref is honored", func(t *testing.T) {
		t.Parallel()
		const (
			mainXSD = "honor_main.xsd"
			baseXSD = "honor_base.xsd"
		)
		fsys := fstest.MapFS{
			mainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:redefine schemaLocation="honor_base.xsd">
    <xs:attributeGroup name="g">
      <xs:attribute name="a" type="xs:string"/>
      <xs:attributeGroup ref="gNew"/>
    </xs:attributeGroup>
  </xs:redefine>
  <xs:complexType name="t">
    <xs:attributeGroup ref="g"/>
  </xs:complexType>
  <xs:element name="root" type="t"/>
</xs:schema>`)},
			baseXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attributeGroup name="gNew">
    <xs:attribute name="a" type="xs:int"/>
  </xs:attributeGroup>
  <xs:attributeGroup name="g">
    <xs:attribute name="z" type="xs:string"/>
  </xs:attributeGroup>
</xs:schema>`)},
		}
		errStr := compile(t, fsys, mainXSD)
		require.Contains(t, errStr, "Duplicate attribute use",
			"override's new nested ref must be flattened so its duplicate is detected; got: %q", errStr)
	})

	t.Run("no-self-ref override cannot add attributes outside original", func(t *testing.T) {
		t.Parallel()
		const (
			mainXSD = "outside_main.xsd"
			baseXSD = "outside_base.xsd"
		)
		fsys := fstest.MapFS{
			mainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:redefine schemaLocation="outside_base.xsd">
    <xs:attributeGroup name="g">
      <xs:attributeGroup ref="added"/>
    </xs:attributeGroup>
  </xs:redefine>
  <xs:attributeGroup name="added">
    <xs:attribute name="z" type="xs:string"/>
  </xs:attributeGroup>
  <xs:complexType name="t"><xs:attributeGroup ref="g"/></xs:complexType>
  <xs:element name="root" type="t"/>
</xs:schema>`)},
			baseXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attributeGroup name="g">
    <xs:attribute name="a" type="xs:string"/>
  </xs:attributeGroup>
</xs:schema>`)},
		}
		errStr := compile(t, fsys, mainXSD)
		require.Contains(t, errStr, "src-redefine.7.2",
			"a no-self-ref attributeGroup redefine must restrict the original group; got: %q", errStr)
	})

	t.Run("no-self-ref override cannot drop required original attribute", func(t *testing.T) {
		t.Parallel()
		const (
			mainXSD = "required_main.xsd"
			baseXSD = "required_base.xsd"
		)
		fsys := fstest.MapFS{
			mainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:redefine schemaLocation="required_base.xsd">
    <xs:attributeGroup name="g">
      <xs:attribute name="b" type="xs:string"/>
    </xs:attributeGroup>
  </xs:redefine>
  <xs:complexType name="t"><xs:attributeGroup ref="g"/></xs:complexType>
  <xs:element name="root" type="t"/>
</xs:schema>`)},
			baseXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attributeGroup name="g">
    <xs:attribute name="a" type="xs:string" use="required"/>
    <xs:attribute name="b" type="xs:string"/>
  </xs:attributeGroup>
</xs:schema>`)},
		}
		errStr := compile(t, fsys, mainXSD)
		require.Contains(t, errStr, "src-redefine.7.2",
			"a no-self-ref attributeGroup redefine must keep required original attributes; got: %q", errStr)
	})
}
