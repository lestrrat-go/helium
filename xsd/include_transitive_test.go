package xsd_test

import (
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// A transitive xs:include chain (main -> inc1 -> inc2) must compile: the type
// defined only in inc2 has to be reachable from main even though main only
// includes inc1. Before the fix, loadInclude parsed an included schema's
// declarations but never processed that schema's OWN xs:include, so inc2 was
// never loaded and the type reference in main failed to resolve.
func TestCompile_TransitiveInclude(t *testing.T) {
	const ns = "urn:t"

	fsys := fstest.MapFS{
		"main.xsd": &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:t="` + ns + `" targetNamespace="` + ns + `">
  <xs:include schemaLocation="inc1.xsd"/>
  <xs:element name="root" type="t:LeafType"/>
</xs:schema>`)},
		"inc1.xsd": &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="` + ns + `">
  <xs:include schemaLocation="inc2.xsd"/>
</xs:schema>`)},
		"inc2.xsd": &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="` + ns + `">
  <xs:complexType name="LeafType">
    <xs:sequence>
      <xs:element name="x" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`)},
	}

	data, err := fsys.ReadFile("main.xsd")
	require.NoError(t, err)
	doc, err := helium.NewParser().Parse(t.Context(), data)
	require.NoError(t, err)

	schema, err := xsd.NewCompiler().FS(fsys).Compile(t.Context(), doc)
	require.NoError(t, err, "transitive include chain must compile")
	require.NotNil(t, schema)

	// Prove the transitively-included type is actually wired into the schema by
	// validating an instance that uses it.
	// elementFormDefault is unqualified, so the local element "x" lives in no
	// namespace (xmlns="" on it) while the global "root" is in the target ns.
	instance := []byte(`<root xmlns="` + ns + `"><x xmlns="">hello</x></root>`)
	idoc, err := helium.NewParser().Parse(t.Context(), instance)
	require.NoError(t, err)
	require.NoError(t, xsd.NewValidator(schema).Validate(t.Context(), idoc),
		"instance using the transitively-included type must validate")
}
