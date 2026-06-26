package xsd_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// uriReadFS is a minimal in-memory fs.FS keyed by the canonical URI names the
// xsd compiler hands to its loader, used to exercise the resolver/in-memory
// Compile path. fs.ReadFile prefers ReadFile when available, so only that
// method needs real backing.
type uriReadFS map[string][]byte

func (uriReadFS) Open(string) (fs.File, error) { return nil, fs.ErrNotExist }

func (f uriReadFS) ReadFile(name string) ([]byte, error) {
	data, ok := f[name]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return data, nil
}

// A circular xs:include chain (main -> inc -> main) must compile cleanly: the
// re-include of the top-level schema has to be recognized as already-loaded
// rather than re-parsed. Before the fix includeVisited only contained documents
// pulled in via loadInclude/loadRedefine, so the back-reference to main re-parsed
// it and re-registered its declarations, producing spurious duplicate-component
// errors. CompileFile seeds includeVisited with the root's resolved key to close
// the cycle.
func TestCompileFile_CircularInclude(t *testing.T) {
	const ns = "urn:c"

	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.xsd")
	require.NoError(t, os.WriteFile(mainPath, []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:t="`+ns+`" targetNamespace="`+ns+`">
  <xs:include schemaLocation="inc.xsd"/>
  <xs:element name="root" type="t:LeafType"/>
</xs:schema>`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "inc.xsd"), []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="`+ns+`">
  <xs:include schemaLocation="main.xsd"/>
  <xs:complexType name="LeafType">
    <xs:sequence>
      <xs:element name="x" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`), 0o600))

	schema, err := xsd.NewCompiler().CompileFile(t.Context(), mainPath)
	require.NoError(t, err, "circular include back to the root schema must compile without duplicate-component errors")
	require.NotNil(t, schema)
}

// The resolver/in-memory Compile path (no filesystem root path, e.g. xslt3
// loading a schema through its URIResolver) must close the same circular include
// cycle as CompileFile. Compile derives the root key from the document's own URL
// (or a full-URI BaseDir); without it a back-reference (main -> inc -> main)
// re-parses main and emits spurious duplicate-component errors.
func TestCompile_CircularInclude_URIBase(t *testing.T) {
	const ns = "urn:c"

	mainBytes := []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:t="` + ns + `" targetNamespace="` + ns + `">
  <xs:include schemaLocation="inc.xsd"/>
  <xs:element name="root" type="t:LeafType"/>
</xs:schema>`)
	incBytes := []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="` + ns + `">
  <xs:include schemaLocation="main.xsd"/>
  <xs:complexType name="LeafType">
    <xs:sequence>
      <xs:element name="x" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`)

	const mainURI = "mem:/schemas/main.xsd"
	fsys := uriReadFS{
		mainURI:                mainBytes,
		"mem:/schemas/inc.xsd": incBytes,
	}

	// Parse main with its own URI as base so doc.URL() carries the canonical
	// location the compiler seeds the cycle guard from.
	doc, err := helium.NewParser().BaseURI(mainURI).Parse(t.Context(), mainBytes)
	require.NoError(t, err)
	require.Equal(t, mainURI, doc.URL())

	schema, err := xsd.NewCompiler().
		BaseDir(mainURI).
		FS(fsys).
		Compile(t.Context(), doc)
	require.NoError(t, err, "circular include back to the root schema must compile cleanly via Compile")
	require.NotNil(t, schema)
}
