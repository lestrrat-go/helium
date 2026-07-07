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
		requireCompileResultErr(t, err)
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
	// compileSchema compiles the importMainXSD entry of fsys and returns the
	// compiled schema together with the concatenated ErrorHandler diagnostics.
	// Returning the schema (not just the error string) lets the no-namespace
	// cases assert the *actual* resolution target, so they fail against the old
	// silent-placeholder behavior — not merely against a compile error.
	compileSchema := func(t *testing.T, fsys fstest.MapFS) (*xsd.Schema, string) {
		t.Helper()
		data, err := fsys.ReadFile(importMainXSD)
		require.NoError(t, err)
		doc, err := helium.NewParser().Parse(t.Context(), data)
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		schema, err := xsd.NewCompiler().Label(importMainXSD).ErrorHandler(collector).FS(fsys).Compile(t.Context(), doc)
		requireCompileResultErr(t, err)
		var b strings.Builder
		for _, e := range collector.Errors() {
			b.WriteString(e.Error())
		}
		return schema, b.String()
	}

	// xs:list itemType referencing a type from a no-TNS imported schema. The
	// unprefixed itemType="t" must resolve to the empty-namespace {}t the
	// imported schema actually defines — not to a silent {urn:main}t placeholder
	// (the old behavior). Asserting ItemType.Name == {Local:"t", NS:""} makes
	// this test fail against that placeholder behavior.
	t.Run("list itemType resolves against empty namespace", func(t *testing.T) {
		fsys := fstest.MapFS{
			importMainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:m="urn:main" targetNamespace="urn:main">
  <xs:import schemaLocation="other.xsd"/>
  <xs:simpleType name="myList">
    <xs:list itemType="t"/>
  </xs:simpleType>
  <xs:element name="root" type="m:myList"/>
</xs:schema>`)},
			importOtherXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="t">
    <xs:restriction base="xs:string"/>
  </xs:simpleType>
</xs:schema>`)},
		}
		schema, got := compileSchema(t, fsys)
		require.Empty(t, got, "list itemType from a no-namespace import must resolve to {}t, not error on {urn:main}t; got: %q", got)
		myList, ok := schema.LookupType("myList", "urn:main")
		require.True(t, ok, "myList must be present in the compiled schema")
		require.NotNil(t, myList.ItemType, "myList.ItemType must be resolved")
		require.Equal(t, xsd.QName{Local: "t", NS: ""}, myList.ItemType.Name,
			"itemType must resolve to the imported {}t, not a {urn:main}t placeholder")
	})

	// xs:union memberTypes referencing a type from a no-TNS imported schema. The
	// unprefixed member "t" must resolve to {}t; assert the resolved member set
	// to fail against the old {urn:main}t placeholder behavior.
	t.Run("union memberTypes resolves against empty namespace", func(t *testing.T) {
		fsys := fstest.MapFS{
			importMainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:m="urn:main" targetNamespace="urn:main">
  <xs:import schemaLocation="other.xsd"/>
  <xs:simpleType name="myUnion">
    <xs:union memberTypes="xs:string t"/>
  </xs:simpleType>
  <xs:element name="root" type="m:myUnion"/>
</xs:schema>`)},
			importOtherXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="t">
    <xs:restriction base="xs:string"/>
  </xs:simpleType>
</xs:schema>`)},
		}
		schema, got := compileSchema(t, fsys)
		require.Empty(t, got, "union memberTypes from a no-namespace import must resolve to {}t, not error on {urn:main}t; got: %q", got)
		myUnion, ok := schema.LookupType("myUnion", "urn:main")
		require.True(t, ok, "myUnion must be present in the compiled schema")
		memberNames := make([]xsd.QName, 0, len(myUnion.MemberTypes))
		for _, m := range myUnion.MemberTypes {
			memberNames = append(memberNames, m.Name)
		}
		require.Contains(t, memberNames, xsd.QName{Local: "t", NS: ""},
			"a union member must resolve to the imported {}t, not a {urn:main}t placeholder; got members %v", memberNames)
	})

	// A genuinely missing itemType must still report a fatal error even after the
	// empty-namespace fallback: the fallback must not mask real errors.
	t.Run("missing list itemType still errors", func(t *testing.T) {
		fsys := fstest.MapFS{
			importMainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:m="urn:main" targetNamespace="urn:main">
  <xs:import schemaLocation="other.xsd"/>
  <xs:simpleType name="myList">
    <xs:list itemType="missing"/>
  </xs:simpleType>
  <xs:element name="root" type="m:myList"/>
</xs:schema>`)},
			importOtherXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="t">
    <xs:restriction base="xs:string"/>
  </xs:simpleType>
</xs:schema>`)},
		}
		_, got := compileSchema(t, fsys)
		require.Contains(t, got, "does not resolve to a(n) type definition",
			"a genuinely missing list itemType must still report a fatal error; got: %q", got)
	})
}

// The no-targetNamespace ({}) fallback for unresolved type references must apply
// ONLY to unprefixed (chameleon-style) refs. A PREFIXED ref binds to its
// prefix's namespace: if xmlns:o="urn:other" and the no-TNS imported schema
// defines an UNQUALIFIED type t, then a ref written as "o:t" must report
// unresolved {urn:other}t rather than silently falling back to {}t. This guards
// all four ref kinds (element type, base, list itemType, union memberTypes).
func TestCompile_PrefixedRefNoEmptyNamespaceFallback(t *testing.T) {
	compileErrors := func(t *testing.T, fsys fstest.MapFS) string {
		t.Helper()
		data, err := fsys.ReadFile(importMainXSD)
		require.NoError(t, err)
		doc, err := helium.NewParser().Parse(t.Context(), data)
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, err = xsd.NewCompiler().Label(importMainXSD).ErrorHandler(collector).FS(fsys).Compile(t.Context(), doc)
		requireCompileResultErr(t, err)
		var b strings.Builder
		for _, e := range collector.Errors() {
			b.WriteString(e.Error())
		}
		return b.String()
	}

	// other.xsd has no targetNamespace and defines an unqualified type t.
	const otherNoTNS = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="t">
    <xs:restriction base="xs:string"/>
  </xs:simpleType>
  <xs:complexType name="ct"/>
</xs:schema>`

	// Each case binds prefix o to urn:other (a namespace that contains no type
	// t) and references o:t. The empty-NS fallback must NOT fire, so resolution
	// must report unresolved {urn:other}t.
	cases := []struct {
		name string
		main string
	}{
		{
			name: "element type",
			main: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:o="urn:other" targetNamespace="urn:main">
  <xs:import schemaLocation="other.xsd"/>
  <xs:element name="root" type="o:t"/>
</xs:schema>`,
		},
		{
			name: "base type",
			main: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:o="urn:other" targetNamespace="urn:main">
  <xs:import schemaLocation="other.xsd"/>
  <xs:simpleType name="d">
    <xs:restriction base="o:t"/>
  </xs:simpleType>
  <xs:element name="root" type="d"/>
</xs:schema>`,
		},
		{
			name: "list itemType",
			main: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:o="urn:other" targetNamespace="urn:main">
  <xs:import schemaLocation="other.xsd"/>
  <xs:simpleType name="myList">
    <xs:list itemType="o:t"/>
  </xs:simpleType>
  <xs:element name="root" type="m:myList"/>
</xs:schema>`,
		},
		{
			name: "union memberTypes",
			main: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:o="urn:other" targetNamespace="urn:main">
  <xs:import schemaLocation="other.xsd"/>
  <xs:simpleType name="myUnion">
    <xs:union memberTypes="xs:string o:t"/>
  </xs:simpleType>
  <xs:element name="root" type="m:myUnion"/>
</xs:schema>`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fsys := fstest.MapFS{
				importMainXSD:  &fstest.MapFile{Data: []byte(tc.main)},
				importOtherXSD: &fstest.MapFile{Data: []byte(otherNoTNS)},
			}
			got := compileErrors(t, fsys)
			require.Contains(t, got, "{urn:other}t",
				"a prefixed o:t (o=urn:other) must report unresolved {urn:other}t, not silently fall back to {}t; got: %q", got)
			require.Contains(t, got, "does not resolve to a(n) type definition",
				"prefixed unresolved ref must produce a fatal unresolved-type error; got: %q", got)
		})
	}
}

// The no-targetNamespace ({}) fallback for unresolved type references must NOT
// fire for an UNPREFIXED ref that a DEFAULT namespace declaration (xmlns="...")
// binds to a namespace other than the schema's own target namespace. resolveQName
// binds unprefixed lexical QNames through the in-scope default namespace, so with
// xmlns="urn:other" and a no-TNS imported schema that defines an unqualified type
// t, an unprefixed ref "t" resolves to {urn:other}t — which must report unresolved
// {urn:other}t rather than silently masking to {}t. A "not prefixed" gate alone
// would wrongly fire the fallback here; the gate must instead require the resolved
// namespace to equal the schema's target namespace. Guards all four ref kinds.
func TestCompile_DefaultNamespaceBoundRefNoEmptyNamespaceFallback(t *testing.T) {
	compileErrors := func(t *testing.T, fsys fstest.MapFS) string {
		t.Helper()
		data, err := fsys.ReadFile(importMainXSD)
		require.NoError(t, err)
		doc, err := helium.NewParser().Parse(t.Context(), data)
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, err = xsd.NewCompiler().Label(importMainXSD).ErrorHandler(collector).FS(fsys).Compile(t.Context(), doc)
		requireCompileResultErr(t, err)
		var b strings.Builder
		for _, e := range collector.Errors() {
			b.WriteString(e.Error())
		}
		return b.String()
	}

	// other.xsd has no targetNamespace and defines an unqualified type t.
	const otherNoTNS = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="t">
    <xs:restriction base="xs:string"/>
  </xs:simpleType>
  <xs:complexType name="ct"/>
</xs:schema>`

	// Each main schema declares a DEFAULT namespace xmlns="urn:other" (which holds
	// no type t) and references an unprefixed t. resolveQName binds the unprefixed
	// name through the default ns to {urn:other}t, so the empty-NS fallback must
	// NOT fire and resolution must report unresolved {urn:other}t.
	cases := []struct {
		name string
		main string
	}{
		{
			name: "element type",
			main: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns="urn:other" targetNamespace="urn:main">
  <xs:import schemaLocation="other.xsd"/>
  <xs:element name="root" type="t"/>
</xs:schema>`,
		},
		{
			name: "base type",
			main: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns="urn:other" targetNamespace="urn:main">
  <xs:import schemaLocation="other.xsd"/>
  <xs:simpleType name="d">
    <xs:restriction base="t"/>
  </xs:simpleType>
  <xs:element name="root" type="d"/>
</xs:schema>`,
		},
		{
			name: "list itemType",
			main: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns="urn:other" targetNamespace="urn:main">
  <xs:import schemaLocation="other.xsd"/>
  <xs:simpleType name="myList">
    <xs:list itemType="t"/>
  </xs:simpleType>
  <xs:element name="root" type="m:myList"/>
</xs:schema>`,
		},
		{
			name: "union memberTypes",
			main: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns="urn:other" targetNamespace="urn:main">
  <xs:import schemaLocation="other.xsd"/>
  <xs:simpleType name="myUnion">
    <xs:union memberTypes="xs:string t"/>
  </xs:simpleType>
  <xs:element name="root" type="m:myUnion"/>
</xs:schema>`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fsys := fstest.MapFS{
				importMainXSD:  &fstest.MapFile{Data: []byte(tc.main)},
				importOtherXSD: &fstest.MapFile{Data: []byte(otherNoTNS)},
			}
			got := compileErrors(t, fsys)
			require.Contains(t, got, "{urn:other}t",
				"a default-ns-bound unprefixed t (xmlns=urn:other) must report unresolved {urn:other}t, not silently fall back to {}t; got: %q", got)
			require.Contains(t, got, "does not resolve to a(n) type definition",
				"default-ns-bound unresolved ref must produce a fatal unresolved-type error; got: %q", got)
		})
	}
}

// The no-targetNamespace ({}) fallback must NOT fire for a ref that is QUALIFIED
// to the schema's OWN target namespace — whether via a prefix bound to the TNS
// (xmlns:m="urn:main" -> m:t) or via a default namespace equal to the TNS
// (xmlns="urn:main" -> t). Such a ref binds to {urn:main}t; with only an
// imported no-targetNamespace {}t available it must report a FATAL unresolved
// {urn:main}t, NOT silently bind to {}t. A "resolved namespace == target
// namespace" gate (the prior approach) wrongly fired the fallback here, masking
// the explicit target-namespace ref. Eligibility must instead be tracked from
// the LEXICAL ref: only an UNPREFIXED ref with NO in-scope default namespace is
// eligible. Guards all four ref kinds (element type, base, list itemType, union
// memberTypes) for both the prefixed-target and default-target forms.
func TestCompile_QualifiedTargetNamespaceRefNoEmptyNamespaceFallback(t *testing.T) {
	compileErrors := func(t *testing.T, fsys fstest.MapFS) string {
		t.Helper()
		data, err := fsys.ReadFile(importMainXSD)
		require.NoError(t, err)
		doc, err := helium.NewParser().Parse(t.Context(), data)
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, err = xsd.NewCompiler().Label(importMainXSD).ErrorHandler(collector).FS(fsys).Compile(t.Context(), doc)
		requireCompileResultErr(t, err)
		var b strings.Builder
		for _, e := range collector.Errors() {
			b.WriteString(e.Error())
		}
		return b.String()
	}

	// other.xsd has no targetNamespace and defines an unqualified type t. The
	// main schema's own target namespace is urn:main, which contains no type t.
	const otherNoTNS = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="t">
    <xs:restriction base="xs:string"/>
  </xs:simpleType>
</xs:schema>`

	cases := []struct {
		name string
		main string
	}{
		// Prefix m bound to the schema's own target namespace urn:main: m:t binds
		// to {urn:main}t, which is qualified and must NOT fall back to {}t.
		{
			name: "prefixed-target element type",
			main: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:m="urn:main" targetNamespace="urn:main">
  <xs:import schemaLocation="other.xsd"/>
  <xs:element name="root" type="m:t"/>
</xs:schema>`,
		},
		{
			name: "prefixed-target base type",
			main: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:m="urn:main" targetNamespace="urn:main">
  <xs:import schemaLocation="other.xsd"/>
  <xs:simpleType name="d">
    <xs:restriction base="m:t"/>
  </xs:simpleType>
  <xs:element name="root" type="d"/>
</xs:schema>`,
		},
		{
			name: "prefixed-target list itemType",
			main: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:m="urn:main" targetNamespace="urn:main">
  <xs:import schemaLocation="other.xsd"/>
  <xs:simpleType name="myList">
    <xs:list itemType="m:t"/>
  </xs:simpleType>
  <xs:element name="root" type="m:myList"/>
</xs:schema>`,
		},
		{
			name: "prefixed-target union memberTypes",
			main: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:m="urn:main" targetNamespace="urn:main">
  <xs:import schemaLocation="other.xsd"/>
  <xs:simpleType name="myUnion">
    <xs:union memberTypes="xs:string m:t"/>
  </xs:simpleType>
  <xs:element name="root" type="m:myUnion"/>
</xs:schema>`,
		},
		// Default namespace equal to the schema's own target namespace urn:main:
		// unprefixed t binds to {urn:main}t, which is qualified and must NOT fall
		// back to {}t even though it resolves to the target namespace.
		{
			name: "default-target element type",
			main: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns="urn:main" targetNamespace="urn:main">
  <xs:import schemaLocation="other.xsd"/>
  <xs:element name="root" type="t"/>
</xs:schema>`,
		},
		{
			name: "default-target base type",
			main: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns="urn:main" targetNamespace="urn:main">
  <xs:import schemaLocation="other.xsd"/>
  <xs:simpleType name="d">
    <xs:restriction base="t"/>
  </xs:simpleType>
  <xs:element name="root" type="d"/>
</xs:schema>`,
		},
		{
			name: "default-target list itemType",
			main: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns="urn:main" targetNamespace="urn:main">
  <xs:import schemaLocation="other.xsd"/>
  <xs:simpleType name="myList">
    <xs:list itemType="t"/>
  </xs:simpleType>
  <xs:element name="root" type="m:myList"/>
</xs:schema>`,
		},
		{
			name: "default-target union memberTypes",
			main: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns="urn:main" targetNamespace="urn:main">
  <xs:import schemaLocation="other.xsd"/>
  <xs:simpleType name="myUnion">
    <xs:union memberTypes="xs:string t"/>
  </xs:simpleType>
  <xs:element name="root" type="m:myUnion"/>
</xs:schema>`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fsys := fstest.MapFS{
				importMainXSD:  &fstest.MapFile{Data: []byte(tc.main)},
				importOtherXSD: &fstest.MapFile{Data: []byte(otherNoTNS)},
			}
			got := compileErrors(t, fsys)
			require.Contains(t, got, "{urn:main}t",
				"a ref qualified to the schema's own target namespace must report unresolved {urn:main}t, not silently fall back to {}t; got: %q", got)
			require.Contains(t, got, "does not resolve to a(n) type definition",
				"qualified target-namespace unresolved ref must produce a fatal unresolved-type error; got: %q", got)
		})
	}
}
