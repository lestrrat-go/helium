package xsd_test

import (
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// An xs:redefine override child is a declaration of the REDEFINING (main)
// schema, so its local declarations must use the REDEFINING schema's
// per-document defaults (elementFormDefault/attributeFormDefault/blockDefault/
// finalDefault), not the redefined (base) document's. Here main declares
// elementFormDefault="qualified" and the redefined base OMITS it; the redefine
// override adds a group containing a local element "child" that must therefore
// be QUALIFIED. Before the fix, the redefined document's defaults were still
// active while parsing the override children, so "child" was wrongly treated as
// unqualified.
func TestRedefine_OverrideUsesRedefiningSchemaFormDefault(t *testing.T) {
	t.Parallel()

	const ns = "urn:t"

	const (
		mainXSD = "rdf_main.xsd"
		baseXSD = "rdf_base.xsd"
	)
	fsys := fstest.MapFS{
		mainXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:t="` + ns + `" targetNamespace="` + ns + `" elementFormDefault="qualified">
  <xs:redefine schemaLocation="rdf_base.xsd">
    <xs:group name="g">
      <xs:sequence>
        <xs:element name="child" type="xs:string"/>
      </xs:sequence>
    </xs:group>
  </xs:redefine>
  <xs:complexType name="ct">
    <xs:group ref="t:g"/>
  </xs:complexType>
  <xs:element name="root" type="t:ct"/>
</xs:schema>`)},
		baseXSD: &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="` + ns + `">
  <xs:group name="g">
    <xs:sequence>
      <xs:any namespace="##targetNamespace" processContents="skip"/>
    </xs:sequence>
  </xs:group>
</xs:schema>`)},
	}

	data, err := fsys.ReadFile(mainXSD)
	require.NoError(t, err)
	doc, err := helium.NewParser().Parse(t.Context(), data)
	require.NoError(t, err)

	schema, err := xsd.NewCompiler().FS(fsys).Compile(t.Context(), doc)
	require.NoError(t, err, "redefine chain must compile")
	require.NotNil(t, schema)

	// main declares elementFormDefault="qualified", so the override-local
	// "child" must be QUALIFIED (in the target namespace).
	qualified := []byte(`<root xmlns="` + ns + `"><child>hello</child></root>`)
	qdoc, err := helium.NewParser().Parse(t.Context(), qualified)
	require.NoError(t, err)
	require.NoError(t, xsd.NewValidator(schema).Validate(t.Context(), qdoc),
		"override-local element must use the redefining schema's elementFormDefault=qualified")

	// The unqualified form must be REJECTED: if the redefined (base) document's
	// default had wrongly leaked into the override, this would be the form that
	// validates.
	unqualified := []byte(`<root xmlns="` + ns + `"><child xmlns="">hello</child></root>`)
	udoc, err := helium.NewParser().Parse(t.Context(), unqualified)
	require.NoError(t, err)
	require.Error(t, xsd.NewValidator(schema).Validate(t.Context(), udoc),
		"unqualified override-local element must be rejected when main defaults to qualified")
}
