package xsd_test

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"syscall"
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

// openErrReadFileFS fails at Open with a chosen error and ALSO implements
// ReadFileFS, whose ReadFile returns a chosen error. It models a ReadFileFS-backed
// FS whose Open is unsupported/denied. Three invariants ride on it: a NON-BENIGN
// Open error (e.g. fs.ErrPermission) must not be demoted by falling through to a
// benign ReadFile miss (the Open WHITELIST admits only fs.ErrNotExist/fs.ErrInvalid);
// a benign Open miss whose ReadFile fallback reports a CANONICAL fs.ErrNotExist is a
// confirmed miss and demoted; but a benign Open miss whose fallback reports anything
// ELSE (fs.ErrInvalid, a message-wrapped errno) is phase-unclassifiable and stays
// FATAL, since the atomic fs.ReadFile cannot distinguish a genuine miss from a
// post-resolution read failure.
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

// contentBytesFS serves fixed nested-schema bytes: Open succeeds and the file
// streams the whole content then EOF, so readNestedSchema completes and the
// bytes reach the CONTENT (parse) phase. It models a schemaLocation that
// RESOLVES and READS cleanly, isolating the later content phase.
type contentBytesFS struct{ data []byte }

func (f contentBytesFS) Open(string) (fs.File, error) { return &contentBytesFile{data: f.data}, nil }

type contentBytesFile struct {
	data []byte
	off  int
}

func (contentBytesFile) Stat() (fs.FileInfo, error) { return nil, fs.ErrInvalid }
func (contentBytesFile) Close() error               { return nil }
func (f *contentBytesFile) Read(p []byte) (int, error) {
	if f.off >= len(f.data) {
		return 0, io.EOF
	}
	n := copy(p, f.data[f.off:])
	f.off += n
	return n, nil
}

// entityReadFailFS opens successfully but every read returns fs.ErrInvalid,
// modeling an external entity whose file RESOLVES/opens but is then unreadable.
// Wired as the PARSER's fs.FS, a schema referencing an external entity makes the
// parser surface an error whose chain contains fs.ErrInvalid.
type entityReadFailFS struct{}

func (entityReadFailFS) Open(string) (fs.File, error) { return entityReadFailFile{}, nil }

type entityReadFailFile struct{}

func (entityReadFailFile) Stat() (fs.FileInfo, error) { return nil, fs.ErrInvalid }
func (entityReadFailFile) Close() error               { return nil }
func (entityReadFailFile) Read([]byte) (int, error)   { return 0, fs.ErrInvalid }

// nestedExternalEntitySchema is a well-formed xs:schema whose content references
// an external general entity. Parsed with external-entity substitution enabled,
// the failing external-entity read makes the parse error's chain contain
// fs.ErrInvalid — a CONTENT-phase failure carrying the same errno a benign
// resolution miss would, which must NOT be demoted.
const nestedExternalEntitySchema = `<?xml version="1.0"?>
<!DOCTYPE xs:schema [
  <!ENTITY ext SYSTEM "ext.ent">
]>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="e">
    <xs:annotation><xs:documentation>&ext;</xs:documentation></xs:annotation>
  </xs:element>
</xs:schema>`

// TestNestedIncludeContentPhaseErrnoIsFatal verifies the positive-tagging
// invariant: a nested include whose schemaLocation RESOLVES and READS cleanly but
// whose CONTENT (parse) phase then fails with an error wrapping fs.ErrInvalid — an
// external-entity read failure — is FATAL, never demoted to a benign fetch-miss
// warning. Only readNestedSchema's resolution-phase miss carries the demotable
// errSchemaFetchMiss tag; a downstream content error with the SAME errno does not,
// so it cannot masquerade as a resolution miss.
func TestNestedIncludeContentPhaseErrnoIsFatal(t *testing.T) {
	t.Parallel()

	doc, err := helium.NewParser().Parse(t.Context(), []byte(includeMainSchema))
	require.NoError(t, err)

	// The injected parser loads external entities from a filesystem whose reads
	// fail with fs.ErrInvalid, so parsing the nested schema fails in the content
	// phase with fs.ErrInvalid in its error chain.
	entityParser := helium.NewParser().
		BlockXXE(false).
		LoadExternalDTD(true).
		SubstituteEntities(true).
		FS(entityReadFailFS{})

	_, err = xsd.NewCompiler().
		FS(contentBytesFS{[]byte(nestedExternalEntitySchema)}).
		Parser(entityParser).
		BaseDir(".").
		Compile(t.Context(), doc)

	require.Error(t, err, "a content-phase parse failure must abort compilation")
	// The vulnerability being guarded: the error DOES wrap fs.ErrInvalid, so an
	// errno-based whitelist would wrongly demote it, yet it is NOT a fatal sentinel
	// — only the absence of the resolution-phase tag keeps it fatal.
	require.ErrorIs(t, err, fs.ErrInvalid, "content-phase error wraps fs.ErrInvalid (the phase-confusion the tag prevents)")
	require.False(t, xsd.IsFatalSchemaLoad(err), "not a fatal sentinel; the positive tag — absent here — is what would demote")
}

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
// miss that POSITIVELY signals absence (fs.ErrNotExist from Open) stays a benign
// fetch-miss: the include is skipped and compilation succeeds.
func TestNestedIncludeOpenMissWarns(t *testing.T) {
	t.Parallel()

	doc, err := helium.NewParser().Parse(t.Context(), []byte(includeMainSchema))
	require.NoError(t, err)

	_, err = xsd.NewCompiler().FS(openMissSchemaFS{fs.ErrNotExist}).BaseDir(".").Compile(t.Context(), doc)
	require.NoError(t, err, "an fs.ErrNotExist open-miss must be a skipped fetch-miss, not fatal")
}

// TestNestedIncludeOpenInvalidIsFatal verifies that a plain-fs.FS Open error that
// does NOT positively signal absence — fs.ErrInvalid (a malformed/invalid local
// open), delivered as a *fs.PathError{Op:"open", Err: fs.ErrInvalid} — is FATAL,
// not swallowed as a benign resolution miss. Only fs.ErrNotExist demotes; every
// other Open errno stays fatal (fail-closed).
func TestNestedIncludeOpenInvalidIsFatal(t *testing.T) {
	t.Parallel()

	for _, openErr := range []error{
		fs.ErrInvalid,
		&fs.PathError{Op: "open", Path: "nested.xsd", Err: fs.ErrInvalid},
	} {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(includeMainSchema))
		require.NoError(t, err)

		_, err = xsd.NewCompiler().FS(openMissSchemaFS{openErr}).BaseDir(".").Compile(t.Context(), doc)
		require.Error(t, err, "an fs.ErrInvalid open error (%v) must be fatal, not swallowed as a resolution miss", openErr)
	}
}

// TestNestedIncludeOpenJoinedPermissionMissIsFatal verifies the OPEN-path half of
// the shared demotion guard: an Open error that is an errors.Join of a permission
// denial AND a benign fs.ErrNotExist must NOT be demoted to a warning at the Open
// resolution phase, exactly as the fs.ReadFile fallback rejects the same shape. Both
// join operands would, in isolation, be benign-or-selectable — errors.Is(err,
// fs.ErrNotExist) is true on the whole tree — but the shared notDemotable veto rejects
// the multi-error class outright. Covered both when the FS is Open-only and when it
// also implements ReadFileFS whose ReadFile would return a benign miss.
func TestNestedIncludeOpenJoinedPermissionMissIsFatal(t *testing.T) {
	t.Parallel()

	joined := errors.Join(fs.ErrPermission, fs.ErrNotExist)

	t.Run("open-only", func(t *testing.T) {
		t.Parallel()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(includeMainSchema))
		require.NoError(t, err)

		_, err = xsd.NewCompiler().FS(openMissSchemaFS{joined}).BaseDir(".").Compile(t.Context(), doc)
		require.Error(t, err, "a joined permission+notexist Open error must abort, not be demoted")
	})

	t.Run("with-readfilefs", func(t *testing.T) {
		t.Parallel()
		for _, readErr := range []error{fs.ErrNotExist, fs.ErrInvalid} {
			doc, err := helium.NewParser().Parse(t.Context(), []byte(includeMainSchema))
			require.NoError(t, err)

			_, err = xsd.NewCompiler().
				FS(openErrReadFileFS{openErr: joined, readErr: readErr}).
				BaseDir(".").Compile(t.Context(), doc)
			require.Error(t, err, "a joined permission+notexist Open error must abort even when ReadFile returns a benign miss (%v)", readErr)
		}
	})
}

// TestNestedIncludeOpenSingleMissWarnsWithReadFileFS verifies the complementary
// sound case for the OPEN path: a GENUINE single benign Open miss (fs.ErrNotExist /
// fs.ErrInvalid, no join, no permission) whose ReadFileFS fallback confirms the miss
// still WARNS (skipped, compilation succeeds) — the shared guard does not over-reject
// a legitimate single-error resolution miss. (The Open-only single-miss case is
// TestNestedIncludeOpenMissWarns.)
func TestNestedIncludeOpenSingleMissWarnsWithReadFileFS(t *testing.T) {
	t.Parallel()

	for _, openErr := range []error{fs.ErrNotExist, fs.ErrInvalid} {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(includeMainSchema))
		require.NoError(t, err)

		_, err = xsd.NewCompiler().
			FS(openErrReadFileFS{openErr: openErr, readErr: fs.ErrNotExist}).
			BaseDir(".").Compile(t.Context(), doc)
		require.NoError(t, err, "a single benign Open miss (%v) confirmed by the ReadFile fallback must be a skipped fetch-miss", openErr)
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

// TestNestedIncludeReadFileFallbackErrorIsFatal verifies the reviewer's repro: a
// benign Open miss that falls through to the fs.ReadFile fallback, whose ReadFile
// then ERRORS with anything OTHER than a canonical "file not found" — an
// fs.ErrInvalid, OR a message-WRAPPED errno (even one wrapping fs.ErrNotExist/
// fs.ErrInvalid) — is FATAL, never demoted. fs.ReadFile is atomic, so such an error
// is phase-unclassifiable (a post-resolution read failure is indistinguishable from
// a miss); the fallback error is therefore returned UNTAGGED and stays fatal. Only a
// CANONICAL fs.ErrNotExist from the fallback (a genuine miss) is demotable.
func TestNestedIncludeReadFileFallbackErrorIsFatal(t *testing.T) {
	t.Parallel()

	for _, openErr := range []error{fs.ErrInvalid, fs.ErrNotExist} {
		for _, readErr := range []error{
			// fs.ErrInvalid is not a "not found" errno at all.
			fs.ErrInvalid,
			// A message wrap carries EXTRA, non-canonical context — even wrapping the
			// ErrNotExist sentinel it is a post-resolution annotation, not a direct miss.
			fmt.Errorf("post-resolution read failed: %w", fs.ErrInvalid),
			fmt.Errorf("post-resolution read failed: %w", fs.ErrNotExist),
		} {
			doc, err := helium.NewParser().Parse(t.Context(), []byte(includeMainSchema))
			require.NoError(t, err)

			_, err = xsd.NewCompiler().
				FS(openErrReadFileFS{openErr: openErr, readErr: readErr}).
				BaseDir(".").Compile(t.Context(), doc)
			require.Error(t, err, "unclassifiable ReadFile-fallback error (open=%v, read=%v) must abort, not be demoted", openErr, readErr)
		}
	}
}

// TestNestedIncludeReadFileFallbackNotExistWarns verifies the complementary sound
// case: a benign Open miss whose fs.ReadFile fallback then reports a CANONICAL
// "file not found" (the bare fs.ErrNotExist sentinel) is DEMOTED (skipped,
// compilation succeeds) — a genuine miss confirmed by the fallback, the resolution
// phase, so it warns like any missing schemaLocation hint. (A *fs.PathError-wrapped
// fs.ErrNotExist — what fstest.MapFS / os.DirFS return — is covered by the MapFS
// missing-include tests.)
func TestNestedIncludeReadFileFallbackNotExistWarns(t *testing.T) {
	t.Parallel()

	for _, openErr := range []error{fs.ErrInvalid, fs.ErrNotExist} {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(includeMainSchema))
		require.NoError(t, err)

		_, err = xsd.NewCompiler().
			FS(openErrReadFileFS{openErr: openErr, readErr: fs.ErrNotExist}).
			BaseDir(".").Compile(t.Context(), doc)
		require.NoError(t, err, "a canonical fs.ErrNotExist from the fallback (open=%v) must be a skipped fetch-miss", openErr)
	}
}

// TestNestedIncludeOsDirFSMissWarns verifies the fix against a GENUINE [os.DirFS]:
// opening an absent nested schema returns *fs.PathError{Op:"open", Err: syscall.ENOENT},
// which satisfies fs.ErrNotExist under errors.Is but is NOT the fs.ErrNotExist sentinel
// by ==. os.DirFS is a fully supported FS surface, so a real missing include over it
// must be demoted to a skipped fetch-miss (compilation succeeds), not wrongly made
// fatal by an exact-sentinel comparison.
func TestNestedIncludeOsDirFSMissWarns(t *testing.T) {
	t.Parallel()

	doc, err := helium.NewParser().Parse(t.Context(), []byte(includeMainSchema))
	require.NoError(t, err)

	// An empty directory: nested.xsd is absent, so os.DirFS reports syscall.ENOENT.
	_, err = xsd.NewCompiler().FS(os.DirFS(t.TempDir())).BaseDir(".").Compile(t.Context(), doc)
	require.NoError(t, err, "a genuine os.DirFS missing include (syscall.ENOENT) must be a skipped fetch-miss, not fatal")
}

// TestNestedIncludeReadFileFallbackPathErrorOp verifies the Op discrimination of
// the fs.ReadFile fallback classifier: a *fs.PathError whose Err satisfies
// fs.ErrNotExist (under errors.Is, so a real syscall.ENOENT counts, not just the bare
// sentinel) is a resolution miss (demoted, WARN) ONLY when its Op is exactly "open" —
// the resolution/open operation. A *fs.PathError{Op:"read", Err: fs.ErrNotExist} or
// {Op:"stat", …} is a POST-resolution failure that merely happens to report the
// ErrNotExist errno, so it is phase-UNCLASSIFIABLE and stays FATAL. This is the shape
// fs.ReadFile itself returns from real FSes (fstest.MapFS / os.DirFS surface Op:"open"
// with syscall.ENOENT), so the Op check is sound: a well-behaved FS labels a
// resolution miss "open" and a post-open failure something else.
func TestNestedIncludeReadFileFallbackPathErrorOp(t *testing.T) {
	t.Parallel()

	const nestedPath = "nested.xsd"
	const opOpen = "open" // the resolution/open op a well-behaved FS labels a miss with

	compile := func(t *testing.T, readErr error) error {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(includeMainSchema))
		require.NoError(t, err)
		_, err = xsd.NewCompiler().
			// A benign Open miss forces the ReadFileFS fallback, whose ReadFile
			// returns the chosen *fs.PathError.
			FS(openErrReadFileFS{openErr: fs.ErrNotExist, readErr: readErr}).
			BaseDir(".").Compile(t.Context(), doc)
		return err
	}

	t.Run("open-op-not-exist warns", func(t *testing.T) {
		t.Parallel()
		// The canonical resolution miss a real FS returns for a missing file.
		err := compile(t, &fs.PathError{Op: opOpen, Path: nestedPath, Err: fs.ErrNotExist})
		require.NoError(t, err, "a *fs.PathError{Op:\"open\", Err: fs.ErrNotExist} is a resolution miss and must be demoted to a skipped fetch-miss")
	})

	t.Run("read-op-not-exist fatal", func(t *testing.T) {
		t.Parallel()
		// A post-open read failure that happens to carry the ErrNotExist errno: it
		// resolved and opened, so it must NOT masquerade as a resolution miss.
		err := compile(t, &fs.PathError{Op: "read", Path: nestedPath, Err: fs.ErrNotExist})
		require.Error(t, err, "a *fs.PathError{Op:\"read\", Err: fs.ErrNotExist} is a post-resolution read failure and must abort compilation, not be demoted")
	})

	t.Run("stat-op-not-exist fatal", func(t *testing.T) {
		t.Parallel()
		// Any non-open op is likewise not a resolution/open miss.
		err := compile(t, &fs.PathError{Op: "stat", Path: nestedPath, Err: fs.ErrNotExist})
		require.Error(t, err, "a *fs.PathError with a non-open Op must not be demoted")
	})

	t.Run("open-op-enoent warns", func(t *testing.T) {
		t.Parallel()
		// The real shape os.DirFS returns for a missing file: Op:"open" with a
		// syscall.ENOENT Err. It satisfies fs.ErrNotExist under errors.Is but is NOT the
		// fs.ErrNotExist sentinel by ==. Op:"open" is the resolution phase, so it is a
		// canonical miss and must be demoted.
		err := compile(t, &fs.PathError{Op: opOpen, Path: nestedPath, Err: syscall.ENOENT})
		require.NoError(t, err, "a *fs.PathError{Op:\"open\", Err: syscall.ENOENT} is a real os.DirFS resolution miss and must be demoted to a skipped fetch-miss")
	})

	t.Run("open-op-wrapped-err warns", func(t *testing.T) {
		t.Parallel()
		// Op is "open" but the PathError.Err WRAPS ErrNotExist with extra context rather
		// than being the bare sentinel. The Op=="open" guard already establishes the
		// resolution phase, and errors.Is accepts the wrapped errno, so it is demoted. A
		// lying FS that reports Op:"open" for a post-read failure is out of fs convention
		// and beyond reasonable defense — the load-bearing guard is the Op, not the Err
		// exactness.
		err := compile(t, &fs.PathError{Op: opOpen, Path: nestedPath, Err: fmt.Errorf("context: %w", fs.ErrNotExist)})
		require.NoError(t, err, "a *fs.PathError{Op:\"open\"} whose Err satisfies fs.ErrNotExist is a resolution miss and must be demoted")
	})

	t.Run("bare-sentinel warns", func(t *testing.T) {
		t.Parallel()
		// The bare sentinel (no PathError envelope) is still a canonical miss.
		err := compile(t, fs.ErrNotExist)
		require.NoError(t, err, "the bare fs.ErrNotExist sentinel from the fallback must be demoted to a skipped fetch-miss")
	})

	t.Run("joined-open-permission-and-notexist fatal", func(t *testing.T) {
		t.Parallel()
		// A JOINED error whose Op:"open" PathError actually reports a PERMISSION denial,
		// with a bare fs.ErrNotExist joined ALONGSIDE. errors.Is(err, fs.ErrNotExist)
		// is true on the whole tree (the sibling sentinel), and errors.As would select
		// the Op:"open" PathError — so a first-match classifier could reach it. The
		// whole multi-error is rejected up front (errors.Join implements Unwrap()
		// []error), so it stays fatal regardless.
		err := compile(t, errors.Join(
			&fs.PathError{Op: opOpen, Path: nestedPath, Err: fs.ErrPermission},
			fs.ErrNotExist,
		))
		require.Error(t, err, "a joined open-permission denial must not be demoted just because fs.ErrNotExist is joined alongside")
	})

	t.Run("joined-notexist-open-and-permission fatal", func(t *testing.T) {
		t.Parallel()
		// The mirror image: a benign Op:"open"/ErrNotExist PathError joined ALONGSIDE a
		// bare fs.ErrPermission. errors.As would select the benign PathError, so the
		// pe.Err check alone would demote it; the multi-error rejection keeps it fatal
		// so a fatal error joined anywhere in the chain is never masked.
		err := compile(t, errors.Join(
			&fs.PathError{Op: opOpen, Path: nestedPath, Err: fs.ErrNotExist},
			fs.ErrPermission,
		))
		require.Error(t, err, "a permission error joined alongside a benign miss must not be demoted")
	})

	t.Run("joined-open-notexist-and-read-notexist fatal", func(t *testing.T) {
		t.Parallel()
		// The core repro: a JOINED error of two PathErrors — a benign Op:"open"/
		// ErrNotExist FIRST and a post-open Op:"read"/ErrNotExist SECOND. errors.As
		// selects the first (open/ErrNotExist) and there is no whole-tree permission/
		// fatal cause to veto on, so a first-match classifier would demote a constructed
		// post-open read failure. A real filesystem never joins two errors for one
		// file-open, so the multi-error is rejected outright and stays fatal.
		err := compile(t, errors.Join(
			&fs.PathError{Op: opOpen, Path: nestedPath, Err: fs.ErrNotExist},
			&fs.PathError{Op: "read", Path: nestedPath, Err: fs.ErrNotExist},
		))
		require.Error(t, err, "a joined open+read miss must not be demoted; a real FS never joins errors for one file-open")
	})

	t.Run("wrapped-join-open-and-read fatal", func(t *testing.T) {
		t.Parallel()
		// A multi-error NESTED inside a single-chain message wrapper is still rejected:
		// containsMultiError walks the linear Unwrap() error chain and finds the join.
		err := compile(t, fmt.Errorf("fetch failed: %w", errors.Join(
			&fs.PathError{Op: opOpen, Path: nestedPath, Err: fs.ErrNotExist},
			&fs.PathError{Op: "read", Path: nestedPath, Err: fs.ErrNotExist},
		)))
		require.Error(t, err, "a multi-error wrapped in a single-chain wrapper must not be demoted")
	})
}
