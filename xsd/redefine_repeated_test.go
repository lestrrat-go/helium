package xsd_test

import (
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// compileRedefineFS compiles mainXSD from fsys and returns the concatenated
// schema-error text for assertions plus the compile error (nil on success).
func compileRedefineFS(t *testing.T, fsys fstest.MapFS, mainXSD string) (string, error) {
	t.Helper()
	data, err := fsys.ReadFile(mainXSD)
	require.NoError(t, err)
	doc, err := helium.NewParser().Parse(t.Context(), data)
	require.NoError(t, err)
	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	_, compileErr := xsd.NewCompiler().Label(mainXSD).ErrorHandler(collector).FS(fsys).Compile(t.Context(), doc)
	require.NoError(t, collector.Close())
	_, errStr := partitionCompileErrors(collector.Errors())
	return errStr, compileErr
}

// XSD permits more than one xs:redefine targeting the same schema document, so
// long as no single component is redefined more than once. A document path
// repeating across xs:redefine elements is NOT itself a schema-properties
// violation. The cases below pin the accepted forms (disjoint redefinitions, a
// no-op repeat, a redefine of a separately-included document) and the one
// genuine error (the same component redefined twice).
func TestRedefine_RepeatedDocument(t *testing.T) {
	t.Parallel()

	// (a) Two xs:redefine of the same base document redefining DIFFERENT
	// components must both apply with no error. The second redefine targets an
	// already-loaded document but a disjoint component, so it is processed
	// against the cached Phase-A set rather than rejected.
	t.Run("disjoint components both apply", func(t *testing.T) {
		t.Parallel()
		const mainXSD = "disjoint_main.xsd"
		fsys := fstest.MapFS{
			mainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:redefine schemaLocation="disjoint_base.xsd">
    <xs:attributeGroup name="g1">
      <xs:attributeGroup ref="g1"/>
      <xs:attribute name="a1" type="xs:string"/>
    </xs:attributeGroup>
  </xs:redefine>
  <xs:redefine schemaLocation="disjoint_base.xsd">
    <xs:attributeGroup name="g2">
      <xs:attributeGroup ref="g2"/>
      <xs:attribute name="b1" type="xs:string"/>
    </xs:attributeGroup>
  </xs:redefine>
  <xs:complexType name="t">
    <xs:attributeGroup ref="g1"/>
    <xs:attributeGroup ref="g2"/>
  </xs:complexType>
  <xs:element name="root" type="t"/>
</xs:schema>`)},
			"disjoint_base.xsd": &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attributeGroup name="g1">
    <xs:attribute name="a" type="xs:string"/>
  </xs:attributeGroup>
  <xs:attributeGroup name="g2">
    <xs:attribute name="b" type="xs:string"/>
  </xs:attributeGroup>
</xs:schema>`)},
		}
		errStr, err := compileRedefineFS(t, fsys, mainXSD)
		require.NoError(t, err, "disjoint redefinitions of the same base must both apply; got: %q", errStr)
	})

	// (b) An equivalent/no-op repeated redefine (a second xs:redefine of the same
	// document with no override children) must be accepted: it consumes nothing,
	// so there is no duplicate to report.
	t.Run("no-op repeated redefine accepted", func(t *testing.T) {
		t.Parallel()
		const mainXSD = "noop_main.xsd"
		fsys := fstest.MapFS{
			mainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:redefine schemaLocation="noop_base.xsd">
    <xs:attributeGroup name="g">
      <xs:attribute name="a" type="xs:string"/>
    </xs:attributeGroup>
  </xs:redefine>
  <xs:redefine schemaLocation="noop_base.xsd">
    <xs:annotation><xs:documentation>no overrides</xs:documentation></xs:annotation>
  </xs:redefine>
  <xs:complexType name="t">
    <xs:attributeGroup ref="g"/>
  </xs:complexType>
  <xs:element name="root" type="t"/>
</xs:schema>`)},
			"noop_base.xsd": &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attributeGroup name="g">
    <xs:attribute name="a" type="xs:string"/>
  </xs:attributeGroup>
</xs:schema>`)},
		}
		errStr, err := compileRedefineFS(t, fsys, mainXSD)
		require.NoError(t, err, "a no-op repeated redefine must be accepted; got: %q", errStr)
	})

	// A redefine of a document already pulled in by a separate xs:include must
	// apply its override against the included document's components rather than
	// being rejected for the document path repeating. (Before the fix this was
	// reported as "already loaded".)
	t.Run("redefine of separately-included document applies", func(t *testing.T) {
		t.Parallel()
		const mainXSD = "inc_main.xsd"
		fsys := fstest.MapFS{
			mainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:include schemaLocation="inc_base.xsd"/>
  <xs:redefine schemaLocation="inc_base.xsd">
    <xs:attributeGroup name="g">
      <xs:attribute name="a" type="xs:string"/>
    </xs:attributeGroup>
  </xs:redefine>
  <xs:complexType name="t">
    <xs:attributeGroup ref="g"/>
  </xs:complexType>
  <xs:element name="root" type="t"/>
</xs:schema>`)},
			"inc_base.xsd": &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attributeGroup name="g">
    <xs:attribute name="a" type="xs:string"/>
  </xs:attributeGroup>
</xs:schema>`)},
		}
		errStr, err := compileRedefineFS(t, fsys, mainXSD)
		require.NoError(t, err, "redefine of a separately-included document must apply, not be rejected; got: %q", errStr)
	})

	// (c) The genuine error: two xs:redefine of the same document redefining the
	// SAME component. The second redefinition of an already-consumed component is
	// reported as a duplicate.
	t.Run("same component redefined twice is reported", func(t *testing.T) {
		t.Parallel()
		const mainXSD = "dup_main.xsd"
		fsys := fstest.MapFS{
			mainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:redefine schemaLocation="dup_base.xsd">
    <xs:attributeGroup name="g">
      <xs:attribute name="a" type="xs:string"/>
    </xs:attributeGroup>
  </xs:redefine>
  <xs:redefine schemaLocation="dup_base.xsd">
    <xs:attributeGroup name="g">
      <xs:attribute name="a" type="xs:int"/>
    </xs:attributeGroup>
  </xs:redefine>
  <xs:complexType name="t">
    <xs:attributeGroup ref="g"/>
  </xs:complexType>
  <xs:element name="root" type="t"/>
</xs:schema>`)},
			"dup_base.xsd": &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attributeGroup name="g">
    <xs:attribute name="a" type="xs:string"/>
  </xs:attributeGroup>
</xs:schema>`)},
		}
		errStr, err := compileRedefineFS(t, fsys, mainXSD)
		require.ErrorIs(t, err, xsd.ErrCompilationFailed,
			"redefining the same component twice must be reported")
		require.Contains(t, errStr, "does already exist")
	})
}
