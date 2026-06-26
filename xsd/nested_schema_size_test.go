package xsd_test

import (
	"errors"
	"io/fs"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// endlessSchemaFS hands out a reader that never reaches EOF, modeling a hostile
// schemaLocation that points at an endless device (e.g. /dev/zero).
type endlessSchemaFS struct{}

func (endlessSchemaFS) Open(string) (fs.File, error) { return endlessSchemaFile{}, nil }

type endlessSchemaFile struct{}

func (endlessSchemaFile) Stat() (fs.FileInfo, error) {
	return nil, errors.New("endlessSchemaFile: no stat")
}
func (endlessSchemaFile) Close() error { return nil }
func (endlessSchemaFile) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = ' '
	}
	return len(p), nil
}

// TestNestedIncludeSizeCapped verifies that a nested xs:include read is bounded
// by a byte cap regardless of the configured FS, so an endless source cannot
// exhaust memory. The oversize condition must be classified as a fatal schema
// load. This guards the per-nested-schema cap restored after a transitive
// xs:include refactor briefly routed the loaders through an unbounded read.
func TestNestedIncludeSizeCapped(t *testing.T) {
	t.Parallel()

	const main = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:include schemaLocation="huge.xsd"/>
</xs:schema>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(main))
	require.NoError(t, err)

	_, err = xsd.NewCompiler().FS(endlessSchemaFS{}).BaseDir(".").Compile(t.Context(), doc)
	require.Error(t, err)
	require.True(t, xsd.IsFatalSchemaLoad(err), "oversized nested schema must be a fatal load")
}

// TestNestedImportSizeCapped verifies the same byte cap applies to the
// xs:import loader, whose load failures are otherwise demoted to warnings.
func TestNestedImportSizeCapped(t *testing.T) {
	t.Parallel()

	const main = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:import namespace="urn:other" schemaLocation="huge.xsd"/>
</xs:schema>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(main))
	require.NoError(t, err)

	_, err = xsd.NewCompiler().FS(endlessSchemaFS{}).BaseDir(".").Compile(t.Context(), doc)
	require.Error(t, err)
	require.True(t, xsd.IsFatalSchemaLoad(err), "oversized imported schema must be a fatal load")
}
