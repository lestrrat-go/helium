package xsd_test

import (
	"io/fs"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// readFailSchemaFS opens successfully but its file's Read fails with a chosen
// error, modeling a schemaLocation that RESOLVES and OPENS but is then
// unreadable (a real post-open I/O failure).
type readFailSchemaFS struct{ readErr error }

func (f readFailSchemaFS) Open(string) (fs.File, error) { return readFailSchemaFile(f), nil }

type readFailSchemaFile struct{ readErr error }

func (readFailSchemaFile) Stat() (fs.FileInfo, error) { return nil, fs.ErrInvalid }
func (readFailSchemaFile) Close() error               { return nil }
func (f readFailSchemaFile) Read([]byte) (int, error) { return 0, f.readErr }

// openMissSchemaFS fails at Open with a chosen error, modeling a genuinely
// unresolvable schemaLocation hint (e.g. an http:// URL opened as a path). It is
// NOT the default deny-all FS, so the errno alone decides the classification.
type openMissSchemaFS struct{ openErr error }

func (f openMissSchemaFS) Open(string) (fs.File, error) { return nil, f.openErr }

const includeMainSchema = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:include schemaLocation="nested.xsd"/>
</xs:schema>`

// TestNestedIncludeReadFailAfterOpenIsFatal verifies that a nested
// schemaLocation which RESOLVES/opens but whose READ fails is a FATAL load,
// never demoted to a benign fetch-miss warning — even when the underlying read
// error is fs.ErrInvalid/fs.ErrNotExist (which at the Open step would warn).
func TestNestedIncludeReadFailAfterOpenIsFatal(t *testing.T) {
	t.Parallel()

	for _, readErr := range []error{fs.ErrInvalid, fs.ErrNotExist} {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(includeMainSchema))
		require.NoError(t, err)

		_, err = xsd.NewCompiler().FS(readFailSchemaFS{readErr}).BaseDir(".").Compile(t.Context(), doc)
		require.Error(t, err, "post-open read failure (%v) must abort compilation", readErr)
		require.True(t, xsd.IsFatalSchemaLoad(err), "post-open read failure (%v) must be a fatal load", readErr)
	}
}

// TestNestedIncludeOpenMissWarns verifies that a genuine unresolvable-at-open
// miss (fs.ErrNotExist / fs.ErrInvalid from Open) stays a benign fetch-miss:
// the include is skipped and compilation succeeds.
func TestNestedIncludeOpenMissWarns(t *testing.T) {
	t.Parallel()

	for _, openErr := range []error{fs.ErrNotExist, fs.ErrInvalid} {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(includeMainSchema))
		require.NoError(t, err)

		_, err = xsd.NewCompiler().FS(openMissSchemaFS{openErr}).BaseDir(".").Compile(t.Context(), doc)
		require.NoError(t, err, "open-miss (%v) must be a skipped fetch-miss, not fatal", openErr)
	}
}
