package xsd_test

import (
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// A mutual xs:import cycle between distinct namespaces is legal XSD and must
// TERMINATE cleanly rather than recurse without bound. A chain like
// A(urn:a) → B(urn:b) → C(urn:c) → A(urn:a) is invisible to the per-compiler
// `importedNS` map (each step is a fresh namespace at its import site); the
// shared active-import-ancestry set short-circuits the back-edge to a document
// already mid-load, so the closed loop compiles without spinning or tripping
// the depth ceiling (which remains a defensive net for genuinely deep chains).
func TestCompile_ImportCycle_Terminates(t *testing.T) {
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

	// A → B → C → A (and back). The cycle must resolve to a single set of
	// constituent documents and compile cleanly.
	fsys := fstest.MapFS{
		"a.xsd": &fstest.MapFile{Data: []byte(mkSchema(nsA, nsB, "b.xsd"))},
		"b.xsd": &fstest.MapFile{Data: []byte(mkSchema(nsB, nsC, "c.xsd"))},
		"c.xsd": &fstest.MapFile{Data: []byte(mkSchema(nsC, nsA, "a.xsd"))},
	}

	data, err := fsys.ReadFile("a.xsd")
	require.NoError(t, err)
	doc, err := helium.NewParser().Parse(t.Context(), data)
	require.NoError(t, err)

	schema, err := xsd.NewCompiler().FS(fsys).Compile(t.Context(), doc)
	require.NoError(t, err, "a mutual import cycle across distinct namespaces must compile cleanly")
	require.NotNil(t, schema)
}
