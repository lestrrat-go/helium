package heliumcmd

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"math/rand/v2"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/lestrrat-go/helium/internal/iofs"
)

// nextRandom returns a non-negative random value used to build unique temp file
// names. math/rand/v2 is process-safe for concurrent use and needs no seeding.
func nextRandom() uint64 { return rand.Uint64() }

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

	// Read up to the cap, then probe for one more byte to distinguish
	// "exactly at cap" from "over cap". Probing separately (rather than
	// reading maxBytes+1) avoids overflow when maxBytes == math.MaxInt64.
	buf, err := io.ReadAll(io.LimitReader(r, maxBytes))
	if err != nil {
		return nil, err //nolint:wrapcheck // caller reports raw error
	}
	var probe [1]byte
	n, err := r.Read(probe[:])
	if n > 0 {
		return nil, &inputTooLargeError{name: name, max: maxBytes}
	}
	if err != nil && err != io.EOF {
		return nil, err //nolint:wrapcheck // caller reports raw error
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

// newPendingOutput creates a temporary file in the same directory as the final
// target and prepares the atomic rename. When dest is a symlink, the rename
// target is the resolved real file rather than the link itself: os.Rename would
// otherwise replace the symlink with a regular file, leaving the linked-to file
// untouched (a regression from os.Create, which writes THROUGH symlinks). For a
// non-symlink dest the target is dest unchanged.
//
// The temp file is created with os.OpenFile + O_EXCL and mode 0666 so the
// KERNEL applies the process umask to the new file (matching os.Create
// semantics) without ever reading or mutating the global umask. Keeping the
// temp in the same directory as the target keeps the eventual os.Rename atomic
// (no cross-device copy). The caller writes to the returned *pendingOutput's
// File and must call either Commit (on success) or Cleanup (on failure).
func newPendingOutput(dest string) (*pendingOutput, error) {
	target := dest
	if fi, err := os.Lstat(dest); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		resolved, rerr := resolveSymlinkTarget(dest)
		if rerr != nil {
			return nil, rerr
		}
		target = resolved
	}

	// When the resolved target already exists, Commit will rename the temp file
	// onto it. os.Rename replaces the destination regardless of its mode, so a
	// read-only (e.g. 0444) existing file would be silently overwritten and the
	// command would still exit 0. Refuse up front unless the existing target is a
	// regular, writable file, matching what os.Create would have been able to do.
	if err := checkExistingTargetWritable(target); err != nil {
		return nil, err
	}

	dir := filepath.Dir(target)
	tmp, err := newTempFile(dir)
	if err != nil {
		return nil, err
	}
	return &pendingOutput{f: tmp.f, tmp: tmp.name, dest: target}, nil
}

// checkExistingTargetWritable verifies that an already-existing output target can
// be replaced. It is a no-op when target does not exist (the common create
// case). When target exists it must be a regular file (refusing directories,
// devices, fifos, etc.) and must be writable: os.Rename would otherwise
// overwrite a read-only file and the command would exit 0, masking the failure.
// Writability is probed with an O_WRONLY open (closed immediately) rather than
// inferred from the mode bits, so it honors ownership and ACLs the way the
// subsequent write would.
func checkExistingTargetWritable(target string) error {
	fi, err := os.Stat(target)
	if err != nil {
		// Does not exist (or cannot be stat'd): nothing to protect. A real
		// permission problem on the parent directory surfaces later at temp-file
		// creation or rename time.
		return nil //nolint:nilerr // missing target is the normal create path
	}
	if !fi.Mode().IsRegular() {
		return fmt.Errorf("refusing to overwrite %q: not a regular file", target)
	}
	f, err := os.OpenFile(target, os.O_WRONLY, 0) //nolint:gosec // CLI output path is user supplied
	if err != nil {
		return fmt.Errorf("cannot write to %q: %w", target, err)
	}
	_ = f.Close()
	return nil
}

// maxSymlinkHops bounds how many links resolveSymlinkTarget will follow before
// declaring a loop. It mirrors the conventional ELOOP limit.
const maxSymlinkHops = 40

// resolveSymlinkTarget resolves a symlink to the path it ultimately points at,
// WITHOUT requiring that path to exist. filepath.EvalSymlinks cannot be used
// here: it stats every component and fails with ENOENT on a link whose target
// is missing (link.xml -> missing.xml), yet plain os.Create would happily write
// THROUGH such a link, creating the missing target. We replicate that
// write-through behavior so the output lands on the resolved target and the
// link itself is preserved.
//
// The chain is followed with os.Readlink: each relative link is resolved
// against the directory of the link that named it, and a hop cap detects loops
// (a self- or mutually-referential link never resolves to a real file).
func resolveSymlinkTarget(path string) (string, error) {
	current := path
	for range maxSymlinkHops {
		dest, err := os.Readlink(current)
		if err != nil {
			// Not a symlink (EINVAL) or a broken/missing component (ENOENT):
			// the last resolved path is the write-through target. The common
			// case is the final target not existing yet, which is exactly the
			// path we want os.Create to write to. The Readlink error is
			// expected end-of-chain, not a failure, so it is intentionally not
			// propagated.
			return current, nil //nolint:nilerr // Readlink error marks chain end, not a failure
		}
		if !filepath.IsAbs(dest) {
			dest = filepath.Join(filepath.Dir(current), dest)
		}
		current = filepath.Clean(dest)
	}
	return "", fmt.Errorf("too many levels of symbolic links resolving %q", path)
}

// tempFile is a freshly created, exclusively owned temp file.
type tempFile struct {
	f    *os.File
	name string
}

// newTempFile creates a uniquely named temp file in dir using O_CREATE|O_EXCL so
// the kernel applies the process umask to the new file's 0666 mode. It retries
// on the rare name collision.
func newTempFile(dir string) (*tempFile, error) {
	for range 10000 {
		name := filepath.Join(dir, fmt.Sprintf(".helium-out-%d-%d", os.Getpid(), nextRandom()))
		f, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o666) //nolint:gosec // umask applied by kernel; mode adjusted at Commit
		if err == nil {
			return &tempFile{f: f, name: name}, nil
		}
		if !os.IsExist(err) {
			return nil, err //nolint:wrapcheck // caller reports raw error
		}
	}
	return nil, fmt.Errorf("could not create temp output file in %s", dir)
}

// File returns the underlying temp file the caller writes output to.
func (p *pendingOutput) File() *os.File { return p.f }

// Commit closes the temp file and atomically renames it onto the destination.
// It must be called only after all output has been written and all inputs have
// been read. A non-nil error means the destination was left untouched.
//
// The temp file was created with mode 0666 masked by the process umask, which
// already matches the os.Create default for a NEW destination, so no chmod is
// needed in that case. When the destination already EXISTS, os.Create would
// keep the file's current permissions; we replicate that by chmod-ing the temp
// to the existing mode (permission plus sticky/setuid/setgid bits) before the
// rename.
func (p *pendingOutput) Commit() error {
	if p.done {
		return nil
	}
	p.done = true
	if err := p.f.Close(); err != nil {
		_ = os.Remove(p.tmp)
		return err //nolint:wrapcheck // caller reports raw error
	}
	if fi, err := os.Stat(p.dest); err == nil {
		// Preserve the permission bits AND the sticky/setuid/setgid bits the
		// existing destination carries; fi.Mode().Perm() drops the latter.
		mode := fi.Mode() & (os.ModePerm | os.ModeSetuid | os.ModeSetgid | os.ModeSticky)
		if cerr := os.Chmod(p.tmp, mode); cerr != nil {
			_ = os.Remove(p.tmp)
			return cerr //nolint:wrapcheck // caller reports raw error
		}
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

// newConfinedDirFS returns a confined fs.FS rooted at the directory containing
// file (the stylesheet path). It is the xslt command's opt-in FS for loading a
// stylesheet's external DTD/entities (--noent / --loaddtd): unlike
// PermissiveRoot it refuses any path that resolves outside the stylesheet's own
// directory, so even with external loading enabled an attacker-supplied SYSTEM
// identifier ("/etc/passwd", "../../secret") cannot exfiltrate arbitrary local
// files into the transform output. It delegates to [iofs.NewConfinedDir], the
// shared implementation behind the public helium.DirFS adapter.
func newConfinedDirFS(file string) fs.FS {
	return iofs.NewConfinedDir(filepath.Dir(file))
}

// fileResolver loads stylesheet modules off the local filesystem. It is the
// CLI's explicit opt-in to xsl:include/xsl:import resolution: the xslt3
// compiler default-denies module loading unless a URIResolver is installed.
// The compiler resolves hrefs against the stylesheet's base URI before
// calling Resolve, so the supplied uri is already an absolute path.
//
// maxInputBytes carries the configured --max-input-bytes cap so modules loaded
// here (xsl:include/xsl:import, plus stylesheet-location reads via the retained
// resolver) are subject to the same byte limit as the top-level inputs. xslt3
// drains the returned reader with io.ReadAll, so the resolver must enforce the
// cap itself rather than relying on the caller; a maxInputBytes <= 0 disables
// it (unbounded read).
type fileResolver struct {
	maxInputBytes int64
}

func (r fileResolver) Resolve(uri string) (io.ReadCloser, error) {
	path, err := localFilePath(uri)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path) //nolint:gosec // CLI stylesheet path is user supplied
	if err != nil {
		return nil, err //nolint:wrapcheck // compiler wraps the resolve error
	}
	defer func() { _ = f.Close() }()

	// Read the whole module through readInput so the byte cap applies. xslt3
	// reads the returned ReadCloser with io.ReadAll, so a streaming-but-capped
	// reader would not help: the cap must reject oversized modules up front.
	data, err := readInput(f, path, r.maxInputBytes)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

// localFilePath converts a stylesheet href into a local filesystem path. Plain
// relative or absolute paths pass through unchanged. A "file:" URI is parsed
// and converted to a filesystem path: only an empty or "localhost" host is
// accepted, the percent-encoded path is decoded, and on Windows a leading slash
// before a drive letter is stripped. Any other scheme (http, https, ftp, ...)
// is rejected so the resolver never reaches across the network.
func localFilePath(uri string) (string, error) {
	// A bare path with no scheme is the common case: relative or absolute
	// filesystem path. Detect a scheme only when it looks like "<scheme>:".
	if !hasURIScheme(uri) {
		return uri, nil
	}

	u, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("invalid URI %q: %w", uri, err)
	}
	if u.Scheme != "file" {
		return "", fmt.Errorf("unsupported URI scheme %q: only file: and local paths are allowed", u.Scheme)
	}
	if u.Host != "" && !strings.EqualFold(u.Host, "localhost") {
		return "", fmt.Errorf("unsupported file URI host %q: only local files are allowed", u.Host)
	}

	// A "file:////server/share" URI parses to an empty host with a path that
	// begins with two separators; on Windows that path becomes a UNC path
	// (\\server\share) reaching a remote SMB host, defeating the local-only
	// policy. Reject every UNC form outright (shared with iofs.FileURIToPath via
	// iofs.IsUNCFileURIPath, which also catches %5C-decoded backslashes).
	if iofs.IsUNCFileURIPath(u.Path) {
		return "", fmt.Errorf("unsupported UNC file URI %q: only local files are allowed", uri)
	}

	// u.Path is already percent-decoded by url.Parse.
	return fileURIPathToLocal(u.Path, filepath.Separator == '\\'), nil
}

// fileURIPathToLocal converts the (already percent-decoded) path component of a
// "file:" URI into a local filesystem path. The windows argument selects the
// host platform's conventions so the logic can be unit-tested for both.
//
// On Windows a URI like "file:///C:/x" parses to path "/C:/x"; the leading
// slash before the drive letter must be dropped and separators normalized so it
// becomes "C:\\x". A rooted, non-drive path such as "/tmp/x" must keep its
// leading slash so it stays absolute (and not become the relative "tmp\\x").
// On non-Windows the path passes through unchanged: "/tmp/x" stays "/tmp/x".
func fileURIPathToLocal(path string, windows bool) string {
	if !windows {
		return path
	}
	if isDriveLetterPath(path) {
		path = strings.TrimPrefix(path, "/")
	}
	// Normalize to backslashes explicitly: filepath.FromSlash is a no-op when
	// the host is not Windows, so the conversion must not depend on GOOS here.
	return strings.ReplaceAll(path, "/", `\`)
}

// isDriveLetterPath reports whether path has the leading-slash drive-letter form
// "/X:/..." or "/X:" produced by parsing a Windows "file:" URI.
func isDriveLetterPath(path string) bool {
	if len(path) < 3 {
		return false
	}
	if path[0] != '/' {
		return false
	}
	c := path[1]
	isAlpha := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
	if !isAlpha {
		return false
	}
	if path[2] != ':' {
		return false
	}
	// Either "/X:" exactly, or "/X:/..." — a bare colon not followed by a
	// separator (e.g. "/X:foo") is not a valid rooted drive path.
	return len(path) == 3 || path[3] == '/'
}

// hasURIScheme reports whether uri begins with an RFC 3986 scheme followed by
// ":". A single Windows drive letter ("C:\\...") is NOT treated as a scheme.
func hasURIScheme(uri string) bool {
	colon := strings.IndexByte(uri, ':')
	if colon <= 0 {
		return false
	}
	scheme := uri[:colon]
	// A single-character "scheme" is almost certainly a Windows drive letter.
	if len(scheme) == 1 {
		return false
	}
	for i, c := range []byte(scheme) {
		isAlpha := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
		isDigit := c >= '0' && c <= '9'
		isOther := c == '+' || c == '-' || c == '.'
		if i == 0 && !isAlpha {
			return false
		}
		if !isAlpha && !isDigit && !isOther {
			return false
		}
	}
	return true
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
