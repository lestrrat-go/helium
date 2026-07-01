package xsd_test

import (
	"testing"
	"testing/fstest"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// A circular xs:import whose back-edge points at a document still mid-parse on the
// active import ancestry is short-circuited to break the cycle, but src-import
// validity is NOT waived: the back-edge's requested namespace must still match the
// target's targetNamespace, exactly as the acyclic path requires. A mismatched
// back-edge (or the namespace-absent case, which may only import a no-targetNamespace
// schema) is a fatal schema error even though the reload itself is skipped; a
// correctly-declared cycle continues to compile cleanly.
func TestCompile_ImportCycle_NamespaceMismatch(t *testing.T) {
	const fileC = "c.xsd"

	// schema builds a document with targetNamespace targetNS importing importLoc; an
	// empty importNS emits a namespace-absent <xs:import>.
	schema := func(targetNS, importNS, importLoc string) string {
		imp := `<xs:import namespace="` + importNS + `" schemaLocation="` + importLoc + `"/>`
		if importNS == "" {
			imp = `<xs:import schemaLocation="` + importLoc + `"/>`
		}
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="` +
			targetNS + `">` + imp + `</xs:schema>`
	}

	// A → B → C → B. When C imports B, B is mid-parse (on the active ancestry) so the
	// back-edge is short-circuited; its namespace claim must still be validated. Only
	// C's import-of-B namespace (cImportNS) varies per case.
	build := func(cImportNS string) fstest.MapFS {
		return fstest.MapFS{
			fileA:       &fstest.MapFile{Data: []byte(schema("urn:a", "urn:b", residueBXSD))},
			residueBXSD: &fstest.MapFile{Data: []byte(schema("urn:b", "urn:c", fileC))},
			fileC:       &fstest.MapFile{Data: []byte(schema("urn:c", cImportNS, residueBXSD))},
		}
	}

	compile := func(t *testing.T, fsys fstest.MapFS) error {
		t.Helper()
		data, err := fsys.ReadFile(fileA)
		require.NoError(t, err)
		doc, err := helium.NewParser().Parse(t.Context(), data)
		require.NoError(t, err)
		_, err = xsd.NewCompiler().FS(fsys).Compile(t.Context(), doc)
		return err
	}

	t.Run("mismatched back-edge namespace is rejected", func(t *testing.T) {
		// C imports B (mid-parse) claiming urn:WRONG; B's targetNamespace is urn:b.
		require.Error(t, compile(t, build("urn:WRONG")),
			"a cyclic back-edge declaring a namespace that differs from the mid-parse target's targetNamespace must be a src-import error")
	})

	t.Run("namespace-absent back-edge onto a namespaced target is rejected", func(t *testing.T) {
		// A namespace-absent import may only import a no-targetNamespace schema, but
		// B has targetNamespace urn:b.
		require.Error(t, compile(t, build("")),
			"a namespace-absent cyclic back-edge onto a namespaced target must be a src-import error")
	})

	t.Run("correctly-declared cycle compiles", func(t *testing.T) {
		// C imports B (mid-parse) with the correct namespace urn:b.
		require.NoError(t, compile(t, build("urn:b")),
			"a cyclic back-edge with the correct namespace must still compile via the short-circuit")
	})
}
