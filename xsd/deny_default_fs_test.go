package xsd_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestDefaultFSDeniesUntrustedInclude verifies that NewCompiler denies
// nested-schema filesystem access by default: an xs:include whose
// schemaLocation points at a real on-disk file is NOT read unless the caller
// opts in via Compiler.FS. This is the XSD analogue of the secure-by-default
// flip applied to helium.NewParser, and guards against local-file disclosure /
// resource exhaustion driven by an untrusted schema's schemaLocation.
func TestDefaultFSDeniesUntrustedInclude(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "inc.xsd"), []byte(`<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="fromInclude" type="xs:string"/>
</xs:schema>`), 0o600)) //nolint:gosec // test fixture

	mainPath := filepath.Join(dir, "main.xsd")
	require.NoError(t, os.WriteFile(mainPath, []byte(`<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:include schemaLocation="inc.xsd"/>
  <xs:element name="root" type="xs:string"/>
</xs:schema>`), 0o600)) //nolint:gosec // test fixture

	t.Run("default deny-all refuses the on-disk include", func(t *testing.T) {
		t.Parallel()
		_, err := xsd.NewCompiler().CompileFile(t.Context(), mainPath)
		require.Error(t, err, "default compiler must not load the on-disk include")
		require.ErrorIs(t, err, fs.ErrNotExist)
	})

	t.Run("PermissiveFS opts back into host access", func(t *testing.T) {
		t.Parallel()
		schema, err := xsd.NewCompiler().FS(helium.PermissiveFS()).CompileFile(t.Context(), mainPath)
		require.NoError(t, err)
		require.NotNil(t, schema)
	})
}

// endlessFS hands out a reader that never reaches EOF, modeling a hostile
// schemaLocation that points at an endless device (e.g. /dev/zero).
type endlessFS struct{}

func (endlessFS) Open(string) (fs.File, error) { return endlessFile{}, nil }

type endlessFile struct{}

func (endlessFile) Stat() (fs.FileInfo, error) { return nil, errors.New("endlessFile: no stat") }
func (endlessFile) Close() error               { return nil }
func (endlessFile) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = ' '
	}
	return len(p), nil
}

// TestNestedSchemaSizeCapped verifies that a nested schema read is bounded by a
// byte cap regardless of the configured FS, so an endless source cannot exhaust
// memory. The oversize condition is classified as a fatal schema load.
func TestNestedSchemaSizeCapped(t *testing.T) {
	t.Parallel()

	const main = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:include schemaLocation="huge.xsd"/>
</xs:schema>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(main))
	require.NoError(t, err)

	_, err = xsd.NewCompiler().FS(endlessFS{}).BaseDir(".").Compile(t.Context(), doc)
	require.Error(t, err)
	require.True(t, xsd.IsFatalSchemaLoad(err), "oversized nested schema must be a fatal load")
}
