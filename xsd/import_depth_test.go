package xsd_test

import (
	"strings"
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// xsd:import cycles between distinct namespaces must not recurse without
// bound. The per-compiler `importedNS` map only catches reimports of the
// same namespace; a chain like A(urn:a) → B(urn:b) → C(urn:c) → A(urn:a)
// is invisible to that map because each step is a fresh namespace at the
// site of the import. A depth ceiling closes the loop.
func TestCompile_ImportCycle_BoundedByDepth(t *testing.T) {
	const (
		nsA = "urn:a"
		nsB = "urn:b"
		nsC = "urn:c"
	)

	mkSchema := func(targetNS, importNS, importLoc string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  targetNamespace="` + targetNS + `">
  <xs:import namespace="` + importNS + `" schemaLocation="` + importLoc + `"/>
</xs:schema>`
	}

	// A → B → C → A (and back). With max import depth enforced, compilation
	// must terminate with an error rather than spinning indefinitely.
	fsys := fstest.MapFS{
		"a.xsd": &fstest.MapFile{Data: []byte(mkSchema(nsA, nsB, "b.xsd"))},
		"b.xsd": &fstest.MapFile{Data: []byte(mkSchema(nsB, nsC, "c.xsd"))},
		"c.xsd": &fstest.MapFile{Data: []byte(mkSchema(nsC, nsA, "a.xsd"))},
	}

	data, err := fstest.MapFS(fsys).ReadFile("a.xsd")
	require.NoError(t, err)
	doc, err := helium.NewParser().Parse(t.Context(), data)
	require.NoError(t, err)

	_, err = xsd.NewCompiler().FS(fsys).Compile(t.Context(), doc)
	require.Error(t, err, "compilation must reject unbounded import cycle across distinct namespaces")
	require.True(t, strings.Contains(err.Error(), "max import depth"),
		"error must mention the depth limit; got: %v", err)
}
