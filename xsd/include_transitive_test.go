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

// The elementFormDefault/attributeFormDefault/blockDefault/finalDefault
// attributes are PER schema document and must NOT be inherited across an
// xs:include chain. Here the intermediate include (inc1) declares
// elementFormDefault="qualified" but the child include (inc2) OMITS it, so
// inc2's local elements must default to UNQUALIFIED (the spec default), not
// inherit inc1's "qualified". Before the fix, loadInclude left the parent's
// per-document default active when the included schema omitted it.
func TestCompile_TransitiveInclude_FormDefaultNotInherited(t *testing.T) {
	const ns = "urn:t"

	fsys := fstest.MapFS{
		"main.xsd": &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:t="` + ns + `" targetNamespace="` + ns + `">
  <xs:include schemaLocation="inc1.xsd"/>
  <xs:element name="root" type="t:LeafType"/>
</xs:schema>`)},
		"inc1.xsd": &fstest.MapFile{Data: []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="` + ns + `" elementFormDefault="qualified">
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

	// inc2 omits elementFormDefault, so its local element "x" defaults to
	// UNQUALIFIED (no namespace) despite inc1 declaring elementFormDefault=
	// "qualified". The instance must therefore place "x" in NO namespace.
	unqualified := []byte(`<root xmlns="` + ns + `"><x xmlns="">hello</x></root>`)
	idoc, err := helium.NewParser().Parse(t.Context(), unqualified)
	require.NoError(t, err)
	require.NoError(t, xsd.NewValidator(schema).Validate(t.Context(), idoc),
		"local element from inc2 must default to unqualified, not inherit inc1's qualified")

	// The qualified form must be REJECTED: if inc1's "qualified" had wrongly
	// leaked into inc2, this would be the form that validates.
	qualified := []byte(`<root xmlns="` + ns + `"><x>hello</x></root>`)
	qdoc, err := helium.NewParser().Parse(t.Context(), qualified)
	require.NoError(t, err)
	require.Error(t, xsd.NewValidator(schema).Validate(t.Context(), qdoc),
		"qualified local element must be rejected when inc2 defaults to unqualified")
}
