package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestAnnotationSchemaRepresentation exercises the schema-for-schemas XML
// representation of xs:annotation / xs:appinfo / xs:documentation (XSD Structures
// §3.13.2), enforced by the version-independent checkAnnotations DOM walk. These
// constructs were previously accepted (false-accept) because the attribute /
// content-model checks only ran for annotations nested in an element's inline
// type and never for schema-level or leaf-host annotations.
func TestAnnotationSchemaRepresentation(t *testing.T) {
	t.Parallel()

	compile := func(t *testing.T, schemaXML string) bool {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, gotErr := xsd.NewCompiler().Label("test.xsd").ErrorHandler(collector).Compile(t.Context(), doc)
		return gotErr != nil
	}

	for _, tc := range []struct {
		name       string
		schema     string
		wantReject bool
	}{
		{
			name: "annotation with arbitrary attribute",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:annotation foo="bar"/>
</xs:schema>`,
			wantReject: true,
		},
		{
			name: "annotation containing nested annotation",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:annotation>
    <xs:annotation><xs:documentation>x</xs:documentation></xs:annotation>
    <xs:documentation>y</xs:documentation>
  </xs:annotation>
</xs:schema>`,
			wantReject: true,
		},
		{
			name: "annotation containing stray XSD element",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:annotation><xs:element name="stray"/></xs:annotation>
</xs:schema>`,
			wantReject: true,
		},
		{
			name: "documentation empty xml:lang",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:annotation><xs:documentation source="urn:x" xml:lang=""/></xs:annotation>
</xs:schema>`,
			wantReject: true,
		},
		{
			name: "documentation whitespace xml:lang",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:annotation><xs:documentation xml:lang=" "/></xs:annotation>
</xs:schema>`,
			wantReject: true,
		},
		{
			name: "appinfo with disallowed attribute",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:annotation><xs:appinfo id="a"/></xs:annotation>
</xs:schema>`,
			wantReject: true,
		},
		{
			name: "documentation with disallowed attribute",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:annotation><xs:documentation frob="x"/></xs:annotation>
</xs:schema>`,
			wantReject: true,
		},
		{
			name: "two annotations under element",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="foo">
    <xs:annotation><xs:documentation>a</xs:documentation></xs:annotation>
    <xs:annotation><xs:documentation>b</xs:documentation></xs:annotation>
  </xs:element>
</xs:schema>`,
			wantReject: true,
		},
		{
			name: "two annotations under attribute",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attribute name="att" type="xs:string">
    <xs:annotation><xs:documentation>a</xs:documentation></xs:annotation>
    <xs:annotation><xs:documentation>b</xs:documentation></xs:annotation>
  </xs:attribute>
</xs:schema>`,
			wantReject: true,
		},
		// Valid annotations must still compile.
		{
			name: "valid annotation appinfo documentation with source and lang",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:annotation id="a">
    <xs:appinfo source="urn:app"><Anything/></xs:appinfo>
    <xs:documentation source="urn:doc" xml:lang="en">Free text.</xs:documentation>
  </xs:annotation>
  <xs:element name="foo" type="xs:string"/>
</xs:schema>`,
			wantReject: false,
		},
		{
			name: "valid single annotation under element",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="foo">
    <xs:annotation><xs:documentation>ok</xs:documentation></xs:annotation>
    <xs:complexType><xs:sequence/></xs:complexType>
  </xs:element>
</xs:schema>`,
			wantReject: false,
		},
		{
			name: "valid multiple annotations under schema",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:annotation><xs:documentation>one</xs:documentation></xs:annotation>
  <xs:annotation><xs:documentation>two</xs:documentation></xs:annotation>
  <xs:element name="foo" type="xs:string"/>
</xs:schema>`,
			wantReject: false,
		},
		{
			name: "valid appinfo with arbitrary content",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:annotation><xs:appinfo><foo bar="baz"><nested/></foo></xs:appinfo></xs:annotation>
  <xs:element name="foo" type="xs:string"/>
</xs:schema>`,
			wantReject: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.wantReject, compile(t, tc.schema))
		})
	}
}
