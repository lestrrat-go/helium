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

// fatalOpenError is a FatalSchemaLoader-satisfying open error: a resource/policy
// denial that must abort compilation rather than be demoted to a warning.
type fatalOpenError struct{}

func (fatalOpenError) Error() string         { return "xsd_test: fatal open policy denial" }
func (fatalOpenError) FatalSchemaLoad() bool { return true }

// openFatalReadFileFS fails at Open with a FATAL (FatalSchemaLoader) error but ALSO
// implements ReadFileFS, whose ReadFile returns a BENIGN miss. It models a
// ReadFileFS-backed FS that denies Open for policy/resource reasons: the fatal
// open denial must NOT be masked by falling through to the benign ReadFile miss.
type openFatalReadFileFS struct{ readErr error }

func (openFatalReadFileFS) Open(string) (fs.File, error)      { return nil, fatalOpenError{} }
func (f openFatalReadFileFS) ReadFile(string) ([]byte, error) { return nil, f.readErr }

// openErrReadFileFS fails at Open with a chosen (non-sentinel, non-fatal) error
// and ALSO implements ReadFileFS, whose ReadFile returns a chosen error. It models
// a ReadFileFS-backed FS whose Open is unsupported/denied: a NON-BENIGN Open error
// (e.g. fs.ErrPermission) must NOT be demoted by falling through to a benign
// ReadFile miss — the WHITELIST admits only fs.ErrNotExist/fs.ErrInvalid at Open.
type openErrReadFileFS struct {
	openErr error
	readErr error
}

func (f openErrReadFileFS) Open(string) (fs.File, error)    { return nil, f.openErr }
func (f openErrReadFileFS) ReadFile(string) ([]byte, error) { return nil, f.readErr }

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

// TestNestedIncludeFatalOpenNotMaskedByReadFile verifies that a FATAL open error
// (a FatalSchemaLoader resource/policy denial) is returned fatal even when the
// ReadFileFS fallback would return a benign fs.ErrNotExist/fs.ErrInvalid: the
// benign ReadFile miss must NOT mask the fatal open denial.
func TestNestedIncludeFatalOpenNotMaskedByReadFile(t *testing.T) {
	t.Parallel()

	for _, readErr := range []error{fs.ErrNotExist, fs.ErrInvalid} {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(includeMainSchema))
		require.NoError(t, err)

		_, err = xsd.NewCompiler().FS(openFatalReadFileFS{readErr}).BaseDir(".").Compile(t.Context(), doc)
		require.Error(t, err, "fatal open denial must abort even when ReadFile returns a benign miss (%v)", readErr)
		require.True(t, xsd.IsFatalSchemaLoad(err), "fatal open denial must stay fatal, not be masked by a benign ReadFile miss (%v)", readErr)
	}
}

// TestNestedIncludePermissionOpenNotDemoted verifies that a NON-BENIGN open error
// (fs.ErrPermission — not a fatal sentinel, not deny-all) is FATAL: it must not be
// demoted to a benign fetch-miss warning. This holds both when the FS is Open-only
// and when it also implements ReadFileFS whose ReadFile returns a benign
// fs.ErrNotExist miss — the whitelist must not let that miss mask the denial.
func TestNestedIncludePermissionOpenNotDemoted(t *testing.T) {
	t.Parallel()

	t.Run("open-only", func(t *testing.T) {
		t.Parallel()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(includeMainSchema))
		require.NoError(t, err)

		_, err = xsd.NewCompiler().FS(openMissSchemaFS{fs.ErrPermission}).BaseDir(".").Compile(t.Context(), doc)
		require.Error(t, err, "fs.ErrPermission at Open must abort compilation, not be demoted to a warning")
	})

	t.Run("readfile-benign-miss-must-not-mask", func(t *testing.T) {
		t.Parallel()
		for _, readErr := range []error{fs.ErrNotExist, fs.ErrInvalid} {
			doc, err := helium.NewParser().Parse(t.Context(), []byte(includeMainSchema))
			require.NoError(t, err)

			_, err = xsd.NewCompiler().
				FS(openErrReadFileFS{openErr: fs.ErrPermission, readErr: readErr}).
				BaseDir(".").Compile(t.Context(), doc)
			require.Error(t, err, "fs.ErrPermission at Open must abort even when ReadFile returns a benign miss (%v)", readErr)
		}
	})
}
