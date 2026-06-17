package xsd_test

import (
	"strings"
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// Shared filenames for the import test fixtures (the importing schema and the
// imported schema). Hoisted to constants so goconst does not flag the repeated
// FS keys / Label / ReadFile literals across the import test cases.
const (
	importMainXSD  = "main.xsd"
	importOtherXSD = "other.xsd"
)

// An xs:import declares the namespace it expects the referenced schema to
// contribute. If the schema found at schemaLocation has a *different*
// targetNamespace, the import must be rejected — otherwise a schema imported
// as one namespace silently contributes declarations from another. This
// mirrors the libxml2/XSD src-import constraint and the existing xs:include
// target-namespace check. Compile-time schema errors surface through the
// ErrorHandler (not the returned error), so the test inspects collected errors.
func TestCompile_ImportNamespaceMismatch(t *testing.T) {
	compileErrors := func(t *testing.T, fsys fstest.MapFS) string {
		t.Helper()
		data, err := fsys.ReadFile(importMainXSD)
		require.NoError(t, err)
		doc, err := helium.NewParser().Parse(t.Context(), data)
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, err = xsd.NewCompiler().Label(importMainXSD).ErrorHandler(collector).FS(fsys).Compile(t.Context(), doc)
		require.NoError(t, err)
		var b strings.Builder
		for _, e := range collector.Errors() {
			b.WriteString(e.Error())
		}
		return b.String()
	}

	// main imports urn:expected, but other.xsd declares urn:other. Reject.
	t.Run("declared namespace differs from imported targetNamespace", func(t *testing.T) {
		fsys := fstest.MapFS{
			importMainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="urn:main">
  <xs:import namespace="urn:expected" schemaLocation="other.xsd"/>
</xs:schema>`)},
			importOtherXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="urn:other">
  <xs:element name="e" type="xs:string"/>
</xs:schema>`)},
		}
		got := compileErrors(t, fsys)
		require.True(t, strings.Contains(got, "urn:other") && strings.Contains(got, "urn:expected"),
			"error must name both the imported targetNamespace and the requested namespace; got: %q", got)
	})

	// no-namespace import: namespace attr absent, but other.xsd has a TNS. Reject.
	t.Run("no-namespace import of a namespaced schema", func(t *testing.T) {
		fsys := fstest.MapFS{
			importMainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="urn:main">
  <xs:import schemaLocation="other.xsd"/>
</xs:schema>`)},
			importOtherXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="urn:other">
  <xs:element name="e" type="xs:string"/>
</xs:schema>`)},
		}
		got := compileErrors(t, fsys)
		require.True(t, strings.Contains(got, "urn:other"),
			"error must name the imported targetNamespace; got: %q", got)
	})

	// Matching namespace: import succeeds, no error, and the imported
	// declaration resolves.
	t.Run("matching namespace still works", func(t *testing.T) {
		fsys := fstest.MapFS{
			importMainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:o="urn:expected"
  targetNamespace="urn:main">
  <xs:import namespace="urn:expected" schemaLocation="other.xsd"/>
  <xs:element name="root" type="o:t"/>
</xs:schema>`)},
			importOtherXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="urn:expected">
  <xs:simpleType name="t">
    <xs:restriction base="xs:string"/>
  </xs:simpleType>
</xs:schema>`)},
		}
		got := compileErrors(t, fsys)
		require.Empty(t, got, "import with a matching targetNamespace must compile without error")
	})

	// No-namespace import of a no-namespace schema: valid, must still work.
	t.Run("no-namespace import of a no-namespace schema works", func(t *testing.T) {
		fsys := fstest.MapFS{
			importMainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="urn:main">
  <xs:import schemaLocation="other.xsd"/>
  <xs:element name="root" type="t"/>
</xs:schema>`)},
			importOtherXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="t">
    <xs:restriction base="xs:string"/>
  </xs:simpleType>
</xs:schema>`)},
		}
		got := compileErrors(t, fsys)
		require.Empty(t, got, "no-namespace import of a no-namespace schema must compile without error")
	})
}

// A target-namespace schema importing a no-targetNamespace schema and using its
// types via <xs:list itemType="t"/> or <xs:union memberTypes="t"/> must resolve
// the unprefixed ref against the empty namespace ({}t), not the importing
// schema's targetNamespace ({urn:main}t). resolveRefs must try the empty-NS
// fallback before reporting a fatal unresolved-type error — mirroring the
// element-type and base-type ref paths. A genuinely missing type must still
// fail.
func TestCompile_ImportNoNamespaceListUnionRefs(t *testing.T) {
	compileErrors := func(t *testing.T, fsys fstest.MapFS) string {
		t.Helper()
		data, err := fsys.ReadFile(importMainXSD)
		require.NoError(t, err)
		doc, err := helium.NewParser().Parse(t.Context(), data)
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, err = xsd.NewCompiler().Label(importMainXSD).ErrorHandler(collector).FS(fsys).Compile(t.Context(), doc)
		require.NoError(t, err)
		var b strings.Builder
		for _, e := range collector.Errors() {
			b.WriteString(e.Error())
		}
		return b.String()
	}

	// xs:list itemType referencing a type from a no-TNS imported schema.
	t.Run("list itemType resolves against empty namespace", func(t *testing.T) {
		fsys := fstest.MapFS{
			importMainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="urn:main">
  <xs:import schemaLocation="other.xsd"/>
  <xs:simpleType name="myList">
    <xs:list itemType="t"/>
  </xs:simpleType>
  <xs:element name="root" type="myList"/>
</xs:schema>`)},
			importOtherXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="t">
    <xs:restriction base="xs:string"/>
  </xs:simpleType>
</xs:schema>`)},
		}
		got := compileErrors(t, fsys)
		require.Empty(t, got, "list itemType from a no-namespace import must resolve to {}t, not error on {urn:main}t; got: %q", got)
	})

	// xs:union memberTypes referencing a type from a no-TNS imported schema.
	t.Run("union memberTypes resolves against empty namespace", func(t *testing.T) {
		fsys := fstest.MapFS{
			importMainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="urn:main">
  <xs:import schemaLocation="other.xsd"/>
  <xs:simpleType name="myUnion">
    <xs:union memberTypes="xs:string t"/>
  </xs:simpleType>
  <xs:element name="root" type="myUnion"/>
</xs:schema>`)},
			importOtherXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="t">
    <xs:restriction base="xs:string"/>
  </xs:simpleType>
</xs:schema>`)},
		}
		got := compileErrors(t, fsys)
		require.Empty(t, got, "union memberTypes from a no-namespace import must resolve to {}t, not error on {urn:main}t; got: %q", got)
	})

	// A genuinely missing itemType must still report a fatal error even after the
	// empty-namespace fallback: the fallback must not mask real errors.
	t.Run("missing list itemType still errors", func(t *testing.T) {
		fsys := fstest.MapFS{
			importMainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="urn:main">
  <xs:import schemaLocation="other.xsd"/>
  <xs:simpleType name="myList">
    <xs:list itemType="missing"/>
  </xs:simpleType>
  <xs:element name="root" type="myList"/>
</xs:schema>`)},
			importOtherXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="t">
    <xs:restriction base="xs:string"/>
  </xs:simpleType>
</xs:schema>`)},
		}
		got := compileErrors(t, fsys)
		require.Contains(t, got, "does not resolve to a(n) type definition",
			"a genuinely missing list itemType must still report a fatal error; got: %q", got)
	})
}
