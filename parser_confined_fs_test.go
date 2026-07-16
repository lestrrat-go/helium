package helium_test

import (
	"bytes"
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// remapToInRootAbsCatalog stands in for a catalog that maps a scheme-carrying
// SYSTEM id to an in-root ABSOLUTE path. It matches any system id containing
// marker (the declared id reaches Resolve verbatim for the external subset,
// and as the entity's already-resolved URI for ResolveEntity), returning the
// absolute target. It is the adversarial case for systemIDRetryEligible: the
// declared id carries a URI scheme (or drive letter), so even though the target
// is a valid in-root file whose base-relative form would open, the confined-FS
// retry must NOT fire.
type remapToInRootAbsCatalog struct {
	marker string
	target string
}

func (c remapToInRootAbsCatalog) Resolve(_ context.Context, _, sysID string) string {
	if strings.Contains(sysID, c.marker) {
		return c.target
	}
	return ""
}

func (c remapToInRootAbsCatalog) ResolveURI(_ context.Context, _ string) string { return "" }

// validPathFS is a caller-supplied, network-capable fs.FS that ALSO enforces
// fs.ValidPath the way os.DirFS / os.Root.FS / fstest.MapFS do: an invalid
// (absolute / "..") name is rejected with fs.ErrInvalid, any valid name is
// served with fixed content. It records every name it is asked to open.
type validPathFS struct {
	content []byte
	opened  *[]string
}

func (f validPathFS) Open(name string) (fs.File, error) {
	*f.opened = append(*f.opened, name)
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}
	return &memFile{Reader: bytes.NewReader(f.content)}, nil
}

// exactNameFS accepts ONLY one exact name (the historical file-URI name) and
// records every open, standing in for a permissive caller FS keyed on the
// file:// URI name rather than a normalized path.
type exactNameFS struct {
	accept  string
	content []byte
	opened  *[]string
}

func (f exactNameFS) Open(name string) (fs.File, error) {
	*f.opened = append(*f.opened, name)
	if name == f.accept {
		return &memFile{Reader: bytes.NewReader(f.content)}, nil
	}
	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
}

type memFile struct{ *bytes.Reader }

func (f *memFile) Stat() (fs.FileInfo, error) { return memInfo{size: f.Size()}, nil }
func (f *memFile) Close() error               { return nil }

type memInfo struct{ size int64 }

func (memInfo) Name() string       { return "m" }
func (i memInfo) Size() int64      { return i.size }
func (memInfo) Mode() fs.FileMode  { return 0 }
func (memInfo) ModTime() time.Time { return time.Time{} }
func (memInfo) IsDir() bool        { return false }
func (memInfo) Sys() any           { return nil }

// A confined os.DirFS rooted at the document's own directory must resolve a
// relative SYSTEM id even though the parser resolves it against the document's
// absolute base URI (which ParseFile always sets). The resolved name is an
// absolute path that os.DirFS rejects as a non-fs.ValidPath name; the parser
// retries with the base-relative name so the confined FS loads the external DTD.
func TestConfinedDirFSLoadsRelativeSystemID(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sub.dtd"),
		[]byte("<!ELEMENT doc (#PCDATA)>\n"), 0o600))
	docPath := filepath.Join(dir, "doc.xml")
	require.NoError(t, os.WriteFile(docPath, []byte(
		`<?xml version="1.0"?>`+"\n"+
			`<!DOCTYPE doc SYSTEM "sub.dtd">`+"\n"+
			`<doc>hello</doc>`), 0o600))

	doc, err := helium.NewParser().
		BlockXXE(false).
		LoadExternalDTD(true).
		FS(os.DirFS(dir)).
		ParseFile(t.Context(), docPath)

	require.NoError(t, err)
	require.NotNil(t, doc)
	require.NotNil(t, doc.ExtSubset(),
		"a confined os.DirFS rooted at the document directory must load the external subset")
}

// A SYSTEM id that resolves outside the FS root is refused. Here it is declared
// as an absolute path, so it is not retry-eligible (originally absolute) and the
// retry never fires — and even if it were, the base-relative name would ascend
// via ".." and fail fs.ValidPath, so the confinement holds either way. The
// out-of-tree file exists and is readable, proving the refusal is the guard, not a
// missing file. (Neither guard is a symlink sandbox — os.DirFS follows an in-root
// symlink out of the root; only os.Root.FS confines symlinks. See
// TestDirFSFollowsSymlinkButRootFSConfines.)
func TestConfinedDirFSRefusesPathTraversal(t *testing.T) {
	t.Parallel()

	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.dtd")
	require.NoError(t, os.WriteFile(secret, []byte("<!ELEMENT doc EMPTY>\n"), 0o600))

	root := t.TempDir()
	docPath := filepath.Join(root, "doc.xml")
	require.NoError(t, os.WriteFile(docPath, []byte(
		`<?xml version="1.0"?>`+"\n"+
			`<!DOCTYPE doc SYSTEM "`+secret+`">`+"\n"+
			`<doc>hello</doc>`), 0o600))

	doc, err := helium.NewParser().
		BlockXXE(false).
		LoadExternalDTD(true).
		FS(os.DirFS(root)).
		ParseFile(t.Context(), docPath)

	// The load is refused (an originally-absolute id is not retry-eligible), but
	// the parse stays lenient: no fatal error, document returned, subset not loaded.
	require.NoError(t, err)
	require.NotNil(t, doc)
	require.Nil(t, doc.ExtSubset(),
		"an out-of-root absolute-path SYSTEM id must not load through the confined FS")
}

// An originally-ABSOLUTE SYSTEM id that names an in-root file is NEVER retried,
// even though its base-relative form ("sub.dtd") would be a valid fs.ValidPath
// that os.DirFS could open. Only an originally-relative reference is eligible for
// the confined-FS base-relative retry; eligibility — not the fs.ValidPath shape
// of the derived name — enforces the promise (filepath.Rel would happily
// re-relativize the in-root absolute id). The confinement property still holds,
// but the documented "absolute SYSTEM id is never retried" promise is now true by
// construction. The in-root file exists and is readable, proving the refusal is
// the eligibility gate, not a missing file.
func TestConfinedDirFSDoesNotRetryInRootAbsoluteSystemID(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sub.dtd"),
		[]byte("<!ELEMENT doc (#PCDATA)>\n"), 0o600))
	abs := filepath.Join(dir, "sub.dtd") // an ABSOLUTE path under the FS root
	docPath := filepath.Join(dir, "doc.xml")
	require.NoError(t, os.WriteFile(docPath, []byte(
		`<?xml version="1.0"?>`+"\n"+
			`<!DOCTYPE doc SYSTEM "`+abs+`">`+"\n"+
			`<doc>hello</doc>`), 0o600))

	doc, err := helium.NewParser().
		BlockXXE(false).
		LoadExternalDTD(true).
		FS(os.DirFS(dir)).
		ParseFile(t.Context(), docPath)

	require.NoError(t, err)
	require.NotNil(t, doc)
	require.Nil(t, doc.ExtSubset(),
		"an originally-absolute in-root SYSTEM id must not be loaded via the base-relative retry")
}

// The supported confined-FS document base is an ABSOLUTE path (or file: URI): the
// base-relative retry re-relativizes an absolute base to recover the original
// relative name. This is the positive counterpart to the relative-base scope note
// on openExternalResource — with an absolute base, a relative SYSTEM id that
// resolution turns into an absolute path is recovered and the confined FS loads
// it. (A RELATIVE document base is out of scope: BuildURI yields a valid-but-absent
// relative path that fails with fs.ErrNotExist, not fs.ErrInvalid, so the retry
// never fires — the deferred root-aware helium.DirFS(root) adapter is the general
// fix for an FS rooted elsewhere than the document directory.)
func TestConfinedDirFSAbsoluteBaseLoadsRelativeSystemID(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sub.dtd"),
		[]byte("<!ELEMENT doc (#PCDATA)>\n"), 0o600))
	absBase := filepath.Join(dir, "doc.xml")
	doc := `<?xml version="1.0"?>` + "\n" +
		`<!DOCTYPE doc SYSTEM "sub.dtd">` + "\n" +
		`<doc>hello</doc>`

	parsed, err := helium.NewParser().
		BlockXXE(false).
		LoadExternalDTD(true).
		BaseURI(absBase). // ABSOLUTE base — the supported confined-FS scenario
		FS(os.DirFS(dir)).
		Parse(t.Context(), []byte(doc))

	require.NoError(t, err)
	require.NotNil(t, parsed)
	require.NotNil(t, parsed.ExtSubset(),
		"an absolute document base must let the confined FS recover the relative SYSTEM id")
}

// A network-scheme SYSTEM id is refused before any Open, independent of the FS,
// when network access is forbidden (the default).
func TestConfinedDirFSRefusesNetwork(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	docPath := filepath.Join(root, "doc.xml")
	require.NoError(t, os.WriteFile(docPath, []byte(
		`<?xml version="1.0"?>`+"\n"+
			`<!DOCTYPE doc SYSTEM "http://example.com/evil.dtd">`+"\n"+
			`<doc>hello</doc>`), 0o600))

	_, err := helium.NewParser().
		BlockXXE(false).
		LoadExternalDTD(true).
		FS(os.DirFS(root)).
		ParseFile(t.Context(), docPath)

	require.ErrorIs(t, err, helium.ErrNetworkAccessForbidden)
}

// PermissiveRoot still loads an external subset via its absolute resolved path:
// the base-relative retry only fires for an fs.ErrInvalid rejection, which
// os.Open-backed PermissiveRoot never returns, so its behavior is unchanged.
func TestPermissiveFSLoadsAbsolute(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sub.dtd"),
		[]byte("<!ELEMENT doc (#PCDATA)>\n"), 0o600))
	docPath := filepath.Join(dir, "doc.xml")
	require.NoError(t, os.WriteFile(docPath, []byte(
		`<?xml version="1.0"?>`+"\n"+
			`<!DOCTYPE doc SYSTEM "sub.dtd">`+"\n"+
			`<doc>hello</doc>`), 0o600))

	doc, err := helium.NewParser().
		BlockXXE(false).
		LoadExternalDTD(true).
		FS(helium.PermissiveFS()).
		ParseFile(t.Context(), docPath)

	require.NoError(t, err)
	require.NotNil(t, doc)
	require.NotNil(t, doc.ExtSubset())
}

// A nested external parameter entity that sits in a subdirectory must load
// through a confined os.DirFS rooted at the document directory. The retry
// relativizes against the FIXED document-root base, not the nested resource's
// own moving base, so it yields "dtd/declarations.dtd" (root-relative) rather
// than "declarations.dtd" (relative to dtd/, which does not exist at the root).
func TestConfinedDirFSLoadsNestedRelativeSystemID(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "dtd"), 0o755))
	// declarations.dtd sits BESIDE main.dtd, inside dtd/.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "dtd", "declarations.dtd"),
		[]byte(`<!ENTITY greeting "hello from nested PE">`+"\n"), 0o600))
	// main.dtd references declarations.dtd with a RELATIVE system id (beside it).
	require.NoError(t, os.WriteFile(filepath.Join(dir, "dtd", "main.dtd"),
		[]byte(`<!ENTITY % decls SYSTEM "declarations.dtd">`+"\n"+`%decls;`+"\n"+
			`<!ELEMENT doc (#PCDATA)>`+"\n"), 0o600))
	docPath := filepath.Join(dir, "doc.xml")
	require.NoError(t, os.WriteFile(docPath, []byte(
		`<?xml version="1.0"?>`+"\n"+
			`<!DOCTYPE doc SYSTEM "dtd/main.dtd">`+"\n"+
			`<doc>&greeting;</doc>`), 0o600))

	rec := &recordingFS{inner: os.DirFS(dir)}
	doc, err := helium.NewParser().
		BlockXXE(false).
		LoadExternalDTD(true).
		SubstituteEntities(true).
		FS(rec).
		ParseFile(t.Context(), docPath)

	require.NoError(t, err)
	require.NotNil(t, doc)
	require.True(t, rec.wasOpened("dtd/declarations.dtd"),
		"the nested PE must be retried at the document-root-relative name dtd/declarations.dtd")

	de := doc.DocumentElement()
	require.NotNil(t, de)
	require.NotNil(t, de.FirstChild())
	require.Equal(t, "hello from nested PE", string(de.FirstChild().Content()),
		"the nested PE's general entity must have loaded and expanded")
}

// The base-relative retry name is subject to the same network-access guard as
// the primary name: a SYSTEM id whose retry name carries a network scheme is
// refused before any Open, so it never reaches a network-capable caller FS, and
// the parse returns ErrNetworkAccessForbidden.
func TestConfinedFSRefusesNetworkSchemeRetryName(t *testing.T) {
	t.Parallel()

	// "./http:/evil.dtd" resolves against "/tmp/doc.xml" to "/tmp/http:/evil.dtd"
	// (scheme "", primary passes the guard, rejected as a non-ValidPath), whose
	// base-relative retry is "http:/evil.dtd" (scheme "http").
	doc := `<?xml version="1.0"?>` + "\n" +
		`<!DOCTYPE doc SYSTEM "./http:/evil.dtd">` + "\n" +
		`<doc>hello</doc>`

	var opened []string
	_, err := helium.NewParser().
		BlockXXE(false).
		LoadExternalDTD(true).
		BaseURI("/tmp/doc.xml").
		FS(validPathFS{content: []byte("<!ELEMENT doc (#PCDATA)>\n"), opened: &opened}).
		Parse(t.Context(), []byte(doc))

	require.ErrorIs(t, err, helium.ErrNetworkAccessForbidden)
	for _, n := range opened {
		require.NotContains(t, []string{"http", "https", "ftp"}, schemeOf(n),
			"a network-scheme retry name %q must be refused before it reaches Open", n)
	}
}

// schemeOf mirrors internal/uripath.URIScheme for the ASCII-scheme cases the
// network guard checks (first char a letter, a ':' at index >= 2).
func schemeOf(s string) string {
	isLetter := func(b byte) bool { return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') }
	if len(s) < 2 || !isLetter(s[0]) {
		return ""
	}
	for i := 1; i < len(s); i++ {
		c := s[i]
		switch {
		case isLetter(c) || (c >= '0' && c <= '9') || c == '+' || c == '-' || c == '.':
		case c == ':':
			if i < 2 {
				return ""
			}
			return string(bytes.ToLower([]byte(s[:i])))
		default:
			return ""
		}
	}
	return ""
}

// os.DirFS is NOT a symlink-confinement boundary: it follows an in-root symlink
// that points outside the root and reads the out-of-root target through a plain
// valid fs.ValidPath name. os.Root.FS (Go 1.24+) IS symlink-safe and refuses the
// same open. This documents why the parser recommends os.Root.FS for confinement
// and why the fs.ValidPath retry guard is a path-escape guard, not a sandbox.
func TestDirFSFollowsSymlinkButRootFSConfines(t *testing.T) {
	t.Parallel()

	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.dtd")
	require.NoError(t, os.WriteFile(secret, []byte("TOP SECRET OUT-OF-ROOT\n"), 0o600))

	root := t.TempDir()
	require.NoError(t, os.Symlink(outside, filepath.Join(root, "escape")))

	name := "escape/secret.dtd" // a valid fs.ValidPath: no "..", not absolute
	require.True(t, fs.ValidPath(name))

	// os.DirFS follows the symlink and reads the out-of-root file.
	data, err := fs.ReadFile(os.DirFS(root), name)
	require.NoError(t, err, "os.DirFS follows an in-root symlink out of the root")
	require.Equal(t, "TOP SECRET OUT-OF-ROOT\n", string(data))

	// os.Root.FS refuses the symlink escape.
	r, err := os.OpenRoot(root)
	require.NoError(t, err)
	defer r.Close()
	_, err = fs.ReadFile(r.FS(), name)
	require.Error(t, err, "os.Root.FS must refuse an open that escapes the root via a symlink")
}

// The direct (non-catalog) ResolveEntity branch opens the entity's raw resolved
// systemID first — a "file:" URI verbatim — so a permissive caller FS keyed on
// the file-URI name still resolves the entity. Normalization to a local path is
// applied only for the confined-FS base-relative retry, never to the primary.
func TestDirectResolveEntityPreservesFileURIPrimaryName(t *testing.T) {
	t.Parallel()

	doc := `<?xml version="1.0"?>` + "\n" +
		`<!DOCTYPE root [<!ENTITY e SYSTEM "file:///virtual/entity.xml">]>` + "\n" +
		`<root>&e;</root>`

	var opened []string
	parsed, err := helium.NewParser().
		BlockXXE(false).
		SubstituteEntities(true).
		LoadExternalDTD(true).
		FS(exactNameFS{
			accept:  "file:///virtual/entity.xml",
			content: []byte(`<child>x</child>`),
			opened:  &opened,
		}).
		Parse(t.Context(), []byte(doc))

	require.NoError(t, err)
	require.NotEmpty(t, opened)
	require.Equal(t, "file:///virtual/entity.xml", opened[0],
		"the direct ResolveEntity branch must open the raw file-URI systemID first")

	require.NotNil(t, parsed)
	de := parsed.DocumentElement()
	require.NotNil(t, de)
	require.NotNil(t, de.FirstChild(), "the external general entity must have resolved and expanded")
	require.Equal(t, "child", de.FirstChild().Name())
}

// A one-letter-scheme SYSTEM id ("x:opaque") is a valid absolute URI per RFC
// 3986 (scheme = ALPHA *( ALPHA / DIGIT / "+" / "-" / "." )). It must NEVER be
// retried through the confined FS, even when a catalog remaps it to an in-root
// absolute DTD whose base-relative form would open. This is the external-subset
// path: eligibility is captured from the DECLARED id ("x:opaque") before catalog
// mapping. The in-root file exists and is readable, so a load would succeed if
// the retry fired — proving the refusal is the eligibility gate, not a miss.
func TestConfinedDirFSDoesNotRetrySchemeSystemIDExternalSubset(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sub.dtd"),
		[]byte("<!ELEMENT doc (#PCDATA)>\n"), 0o600))
	target := filepath.Join(dir, "sub.dtd") // an in-root ABSOLUTE path
	docPath := filepath.Join(dir, "doc.xml")
	require.NoError(t, os.WriteFile(docPath, []byte(
		`<?xml version="1.0"?>`+"\n"+
			`<!DOCTYPE doc SYSTEM "x:opaque">`+"\n"+
			`<doc>hello</doc>`), 0o600))

	rec := &recordingFS{inner: os.DirFS(dir)}
	doc, err := helium.NewParser().
		BlockXXE(false).
		LoadExternalDTD(true).
		Catalog(remapToInRootAbsCatalog{marker: "x:opaque", target: target}).
		FS(rec).
		ParseFile(t.Context(), docPath)

	require.NoError(t, err)
	require.NotNil(t, doc)
	require.Nil(t, doc.ExtSubset(),
		"a one-letter-scheme SYSTEM id must not load the external subset via the confined-FS retry")
	require.False(t, rec.wasOpened("sub.dtd"),
		"the base-relative retry name must never be opened for a scheme-carrying SYSTEM id")
}

// The ResolveEntity general-entity path: an external general entity declared
// with a one-letter-scheme SYSTEM id ("x:opaque") that a catalog remaps to an
// in-root absolute file must NOT be retried. Its eligibility is gated on the
// declared id via parserCtx.extRefRelative.
func TestConfinedDirFSDoesNotRetrySchemeSystemIDGeneralEntity(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ent.xml"),
		[]byte(`<child>x</child>`), 0o600))
	target := filepath.Join(dir, "ent.xml") // an in-root ABSOLUTE path
	docPath := filepath.Join(dir, "doc.xml")
	require.NoError(t, os.WriteFile(docPath, []byte(
		`<?xml version="1.0"?>`+"\n"+
			`<!DOCTYPE root [<!ENTITY e SYSTEM "x:opaque">]>`+"\n"+
			`<root>&e;</root>`), 0o600))

	rec := &recordingFS{inner: os.DirFS(dir)}
	_, err := helium.NewParser().
		BlockXXE(false).
		SubstituteEntities(true).
		LoadExternalDTD(true).
		Catalog(remapToInRootAbsCatalog{marker: "x:opaque", target: target}).
		FS(rec).
		ParseFile(t.Context(), docPath)

	// The retry is refused, so the scheme-carrying general entity never resolves;
	// with entity substitution requested, an unresolvable external general entity
	// is a fatal parse error. The load never reaching the in-root file is the
	// point — the base-relative retry name must never be opened.
	require.Error(t, err,
		"a scheme-carrying external general entity must not resolve via the confined-FS retry")
	require.False(t, rec.wasOpened("ent.xml"),
		"the base-relative retry name must never be opened for a scheme-carrying general-entity SYSTEM id")
}

// The ResolveEntity parameter-entity path: an external parameter entity declared
// with a one-letter-scheme SYSTEM id ("x:opaque") that a catalog remaps to an
// in-root absolute DTD must NOT be retried. Its eligibility is gated on the
// declared id (loadExternalParameterEntityContent → systemIDRetryEligible).
func TestConfinedDirFSDoesNotRetrySchemeSystemIDParameterEntity(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "decls.dtd"),
		[]byte(`<!ENTITY greeting "hello from PE">`+"\n"), 0o600))
	target := filepath.Join(dir, "decls.dtd") // an in-root ABSOLUTE path
	docPath := filepath.Join(dir, "doc.xml")
	require.NoError(t, os.WriteFile(docPath, []byte(
		`<?xml version="1.0"?>`+"\n"+
			`<!DOCTYPE doc [`+"\n"+
			`<!ENTITY % pe SYSTEM "x:opaque">`+"\n"+
			`%pe;`+"\n"+
			`]>`+"\n"+
			`<doc>&greeting;</doc>`), 0o600))

	rec := &recordingFS{inner: os.DirFS(dir)}
	doc, err := helium.NewParser().
		BlockXXE(false).
		LoadExternalDTD(true).
		SubstituteEntities(true).
		Catalog(remapToInRootAbsCatalog{marker: "x:opaque", target: target}).
		FS(rec).
		ParseFile(t.Context(), docPath)

	require.NoError(t, err)
	require.NotNil(t, doc)
	require.False(t, rec.wasOpened("decls.dtd"),
		"the base-relative retry name must never be opened for a scheme-carrying parameter-entity SYSTEM id")
	_, ok := doc.GetEntity("greeting")
	require.False(t, ok,
		"a scheme-carrying external parameter entity must not load its declarations via the confined-FS retry")
}
