package heliumcmd

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/lestrrat-go/helium/internal/iofs"
)

// iofsPermissiveRoot returns the default permissive fs.FS helium uses to open
// external resources. The --path search FS falls back to it before trying the
// configured search directories.
func iofsPermissiveRoot() fs.FS {
	return iofs.PermissiveRoot{}
}

// DefaultMaxInputBytes caps the number of bytes read from a single XML input
// (file or stdin) by the CLI. It guards against hostile or pathological
// sources (e.g. /dev/zero, an unbounded pipe) that would otherwise exhaust
// memory before parse limits apply. It can be overridden per-command via the
// --max-input-bytes flag.
const DefaultMaxInputBytes = 100 << 20 // 100 MiB

// inputTooLargeError is returned by readInput when the source exceeds the cap.
type inputTooLargeError struct {
	name string
	max  int64
}

func (e *inputTooLargeError) Error() string {
	return fmt.Sprintf("%s: input exceeds maximum size of %d bytes", e.name, e.max)
}

// readInput reads up to maxBytes from r, returning an *inputTooLargeError when
// the source is larger. A maxBytes <= 0 disables the cap (unbounded read). The
// name is used only for diagnostics.
func readInput(r io.Reader, name string, maxBytes int64) ([]byte, error) {
	if maxBytes <= 0 {
		buf, err := io.ReadAll(r)
		if err != nil {
			return nil, err //nolint:wrapcheck // caller reports raw error
		}
		return buf, nil
	}

	// Read one extra byte so we can distinguish "exactly at cap" from
	// "over cap".
	buf, err := io.ReadAll(io.LimitReader(r, maxBytes+1))
	if err != nil {
		return nil, err //nolint:wrapcheck // caller reports raw error
	}
	if int64(len(buf)) > maxBytes {
		return nil, &inputTooLargeError{name: name, max: maxBytes}
	}
	return buf, nil
}

// readInputFile opens name and reads it through readInput so the byte cap is
// enforced for files as well as stdin.
func readInputFile(name string, maxBytes int64) ([]byte, error) {
	f, err := os.Open(name) //nolint:gosec // CLI input path is user supplied
	if err != nil {
		return nil, err //nolint:wrapcheck // caller reports raw error
	}
	defer func() { _ = f.Close() }()
	return readInput(f, name, maxBytes)
}

// pendingOutput is a write-to-temp-then-atomic-rename target. The CLI writes
// command output to a temporary file in the same directory as the final
// destination and only renames it onto the destination after processing
// completes successfully. This closes a truncate-before-read hole: os.Create
// on the destination would truncate it up front, destroying any input, DTD, or
// runtime-resolved stylesheet that the same path is read from LATER during
// processing. Writing to a sibling temp file leaves the destination untouched
// until Commit, so those later reads still see the original contents. On any
// processing error the temp file is removed via Cleanup and the destination is
// never modified.
type pendingOutput struct {
	f    *os.File
	tmp  string
	dest string
	done bool
}

// newPendingOutput creates a temporary file in the same directory as dest.
// Using the same directory keeps the eventual os.Rename atomic (no cross-device
// copy). The caller writes to the returned *pendingOutput's File and must call
// either Commit (on success) or Cleanup (on failure).
func newPendingOutput(dest string) (*pendingOutput, error) {
	dir := filepath.Dir(dest)
	f, err := os.CreateTemp(dir, ".helium-out-*")
	if err != nil {
		return nil, err //nolint:wrapcheck // caller reports raw error
	}
	return &pendingOutput{f: f, tmp: f.Name(), dest: dest}, nil
}

// File returns the underlying temp file the caller writes output to.
func (p *pendingOutput) File() *os.File { return p.f }

// Commit closes the temp file and atomically renames it onto the destination.
// It must be called only after all output has been written and all inputs have
// been read. A non-nil error means the destination was left untouched.
func (p *pendingOutput) Commit() error {
	if p.done {
		return nil
	}
	p.done = true
	if err := p.f.Close(); err != nil {
		_ = os.Remove(p.tmp)
		return err //nolint:wrapcheck // caller reports raw error
	}
	if err := os.Rename(p.tmp, p.dest); err != nil {
		_ = os.Remove(p.tmp)
		return err //nolint:wrapcheck // caller reports raw error
	}
	return nil
}

// Cleanup closes and removes the temp file without touching the destination. It
// is safe to call after Commit (it becomes a no-op) so callers can defer it.
func (p *pendingOutput) Cleanup() {
	if p.done {
		return
	}
	p.done = true
	_ = p.f.Close()
	_ = os.Remove(p.tmp)
}

// samePath reports whether a and b refer to the same file on disk. It compares
// resolved absolute paths first, then falls back to os.SameFile (inode/device)
// so symlinks and "./" prefixes are caught even when the lexical paths differ.
// A path that does not yet exist is compared by absolute path only.
func samePath(a, b string) bool {
	absA, errA := filepath.Abs(a)
	absB, errB := filepath.Abs(b)
	if errA == nil && errB == nil && absA == absB {
		return true
	}

	fiA, errA := os.Stat(a)
	if errA != nil {
		return false
	}
	fiB, errB := os.Stat(b)
	if errB != nil {
		return false
	}
	return os.SameFile(fiA, fiB)
}

// fileResolver loads stylesheet modules off the local filesystem. It is the
// CLI's explicit opt-in to xsl:include/xsl:import resolution: the xslt3
// compiler default-denies module loading unless a URIResolver is installed.
// The compiler resolves hrefs against the stylesheet's base URI before
// calling Resolve, so the supplied uri is already an absolute path.
type fileResolver struct{}

func (fileResolver) Resolve(uri string) (io.ReadCloser, error) {
	f, err := os.Open(uri) //nolint:gosec // CLI stylesheet path is user supplied
	if err != nil {
		return nil, err //nolint:wrapcheck // compiler wraps the resolve error
	}
	return f, nil
}

// pathSearchFS wraps a base fs.FS with a list of additional search directories.
// When the base FS fails to open a name, each search directory is tried with
// the name's base component appended. This mirrors xmllint's --path behavior
// for DTD/entity lookup.
type pathSearchFS struct {
	base fs.FS
	dirs []string
}

func (p pathSearchFS) Open(name string) (fs.File, error) {
	f, err := p.base.Open(name)
	if err == nil {
		return f, nil
	}
	base := filepath.Base(name)
	for _, dir := range p.dirs {
		candidate := filepath.Join(dir, base)
		if cf, cerr := os.Open(candidate); cerr == nil { //nolint:gosec // CLI --path dirs are user supplied
			return cf, nil
		}
	}
	return nil, err //nolint:wrapcheck // return the original base-FS error
}
