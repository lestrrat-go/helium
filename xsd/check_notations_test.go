package xsd_test

import (
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestNotationStructuralRules exercises the version-independent schema-for-schemas
// structural rules for <xs:notation> (XSD Structures §3.14.2): placement,
// @name validity, the @public/@system requirement, the (annotation?) content
// model, disallowed attributes, and name uniqueness. All run in DEFAULT (1.0)
// mode. A valid notation — including one that is the enumeration base of an
// xs:NOTATION restriction — must still compile.
func TestNotationStructuralRules(t *testing.T) {
	t.Parallel()

	reject := []struct {
		name   string
		schema string
	}{
		{
			name: "missing name",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:notation public="image/jpeg" system="viewer.exe"/>
</xs:schema>`,
		},
		{
			name: "empty name",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:notation name="" public="image/jpeg"/>
</xs:schema>`,
		},
		{
			name: "name with colon is not an NCName",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:notation name="foo:bar" public="image/jpeg"/>
</xs:schema>`,
		},
		{
			name: "name starting with digit is not an NCName",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:notation name="-2.5foo" public="image/jpeg"/>
</xs:schema>`,
		},
		{
			name: "neither public nor system",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:notation name="foo"/>
</xs:schema>`,
		},
		{
			name: "duplicate notation name",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:notation name="jpeg" public="image/jpeg" system="viewer.exe"/>
  <xs:notation name="jpeg" public="image/jpeg" system="viewer.exe"/>
</xs:schema>`,
		},
		{
			name: "non-whitespace text content",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:notation name="jpeg" public="image/jpeg">Some Text</xs:notation>
</xs:schema>`,
		},
		{
			name: "disallowed non-annotation child",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:notation name="jpeg" public="image/jpeg"><xs:sequence/></xs:notation>
</xs:schema>`,
		},
		{
			name: "misplaced inside complexType",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="foo">
    <xs:sequence>
      <xs:notation name="jpeg" public="image/jpeg" system="viewer.exe"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`,
		},
		{
			name: "disallowed XSD-namespaced attribute",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:a="http://www.w3.org/2001/XMLSchema">
  <xs:notation a:b="c" name="jpeg" public="image/jpeg" system="viewer.exe"/>
</xs:schema>`,
		},
		{
			name: "disallowed unqualified attribute",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:notation foo="bar" name="jpeg" public="image/jpeg" system="viewer.exe"/>
</xs:schema>`,
		},
	}

	for _, tc := range reject {
		t.Run("reject/"+tc.name, func(t *testing.T) {
			t.Parallel()
			errs := compileSchemaErrors(t, tc.schema)
			require.NotEmpty(t, errs, "expected a compile error rejecting the invalid notation")
		})
	}

	accept := []struct {
		name   string
		schema string
	}{
		{
			name: "public only",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:notation name="jpeg" public="image/jpeg"/>
</xs:schema>`,
		},
		{
			name: "system only",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:notation name="png" system="viewer.exe"/>
</xs:schema>`,
		},
		{
			name: "empty system present",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:notation name="jpeg" public="image/jpeg" system=""/>
</xs:schema>`,
		},
		{
			name: "foreign-namespaced attribute allowed",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:a="urn:foo">
  <xs:notation a:b="c" name="jpeg" public="image/jpeg"/>
</xs:schema>`,
		},
		{
			name: "annotation child allowed",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:notation name="jpeg" public="image/jpeg">
    <xs:annotation><xs:documentation>a JPEG image</xs:documentation></xs:annotation>
  </xs:notation>
</xs:schema>`,
		},
		{
			name: "notation as enumeration base of xs:NOTATION restriction",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:p="urn:p" targetNamespace="urn:p">
  <xs:notation name="jpeg" public="image/jpeg"/>
  <xs:notation name="png" public="image/png"/>
  <xs:simpleType name="imageNotation">
    <xs:restriction base="xs:NOTATION">
      <xs:enumeration value="p:jpeg"/>
      <xs:enumeration value="p:png"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="n" type="p:imageNotation"/>
</xs:schema>`,
		},
	}

	for _, tc := range accept {
		t.Run("accept/"+tc.name, func(t *testing.T) {
			t.Parallel()
			doc, err := helium.NewParser().Parse(t.Context(), []byte(tc.schema))
			require.NoError(t, err)
			_, err = xsd.NewCompiler().Compile(t.Context(), doc)
			require.NoError(t, err, "valid notation schema must compile")
		})
	}
}

// TestNotationUnderOverride verifies that xs:override's content model admits
// <xs:notation> (XSD 1.1, a wholesale-replacement component): a valid notation
// inside xs:override COMPILES, an INVALID one (missing both public and system)
// is still REJECTED (the non-placement checks still run under override), and a
// notation inside xs:redefine — whose content model does NOT admit notation —
// stays a schema error.
func TestNotationUnderOverride(t *testing.T) {
	t.Parallel()

	const (
		mainXSD   = "main.xsd"
		targetXSD = "target.xsd"
	)

	target := &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:notation name="jpeg" public="old/type"/>
</xs:schema>`)}

	compile := func(t *testing.T, main string) (string, error) {
		t.Helper()
		fsys := fstest.MapFS{
			mainXSD:   &fstest.MapFile{Data: []byte(main)},
			targetXSD: target,
		}
		data, err := fsys.ReadFile(mainXSD)
		require.NoError(t, err)
		doc, err := helium.NewParser().Parse(t.Context(), data)
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, cerr := xsd.NewCompiler().
			Label(mainXSD).
			Version(xsd.Version11).
			ErrorHandler(collector).
			FS(fsys).
			Compile(t.Context(), doc)
		require.NoError(t, collector.Close())
		var b []byte
		for _, e := range collector.Errors() {
			b = append(b, e.Error()...)
			b = append(b, '\n')
		}
		return string(b), cerr
	}

	t.Run("valid notation under override compiles", func(t *testing.T) {
		t.Parallel()
		errs, cerr := compile(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:override schemaLocation="target.xsd">
    <xs:notation name="jpeg" public="image/jpeg" system="viewer.exe"/>
  </xs:override>
  <xs:element name="root" type="xs:string"/>
</xs:schema>`)
		require.NoError(t, cerr, "a valid notation inside xs:override must compile; got errors: %q", errs)
		require.Empty(t, errs)
	})

	t.Run("invalid notation under override still rejected", func(t *testing.T) {
		t.Parallel()
		// Missing both @public and @system: the non-placement checks still run.
		errs, _ := compile(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:override schemaLocation="target.xsd">
    <xs:notation name="jpeg"/>
  </xs:override>
  <xs:element name="root" type="xs:string"/>
</xs:schema>`)
		require.Contains(t, errs, "public")
	})

	t.Run("notation under redefine still rejected", func(t *testing.T) {
		t.Parallel()
		errs, _ := compile(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:redefine schemaLocation="target.xsd">
    <xs:notation name="jpeg" public="image/jpeg" system="viewer.exe"/>
  </xs:redefine>
  <xs:element name="root" type="xs:string"/>
</xs:schema>`)
		require.Contains(t, errs, "only allowed as a child of xs:schema or xs:override")
	})
}
