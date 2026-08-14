package helium_test

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/sax"
	"github.com/stretchr/testify/require"
)

// recordingXInclude is a fake helium.XIncludeProcessor that records how the
// parser invokes it, without pulling the real xinclude package into the core
// parser tests. The end-to-end path is exercised by an Example in examples/.
type recordingXInclude struct {
	calls int
	doc   *helium.Document
	n     int
	err   error
}

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
// file:// URI name, in place of a normalized path.
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

// netGateFS serves the same content for ANY name and records every name it is
// asked to open. It stands in for a caller-supplied, network-capable fs.FS: its
// Open never fails, so a network-scheme name reaches it unless the parser's
// AllowNetwork(false) gate refuses the load first.
type netGateFS struct {
	content []byte
	opened  *[]string
}

func (fsys netGateFS) Open(name string) (fs.File, error) {
	*fsys.opened = append(*fsys.opened, name)
	return &netGateFile{Reader: bytes.NewReader(fsys.content)}, nil
}

type netGateFile struct {
	*bytes.Reader
}

func (f *netGateFile) Stat() (fs.FileInfo, error) { return netGateInfo{size: f.Size()}, nil }
func (f *netGateFile) Close() error               { return nil }

type netGateInfo struct{ size int64 }

func (netGateInfo) Name() string       { return "resource" }
func (i netGateInfo) Size() int64      { return i.size }
func (netGateInfo) Mode() fs.FileMode  { return 0 }
func (netGateInfo) ModTime() time.Time { return time.Time{} }
func (netGateInfo) IsDir() bool        { return false }
func (netGateInfo) Sys() any           { return nil }

func openedNetwork(names []string) bool {
	for _, n := range names {
		low := strings.ToLower(n)
		if strings.HasPrefix(low, "http://") ||
			strings.HasPrefix(low, "https://") ||
			strings.HasPrefix(low, "ftp://") {
			return true
		}
	}
	return false
}

func TestConfinedFS(t *testing.T) {
	t.Parallel()

	// A confined os.DirFS rooted at the document's own directory must resolve a
	// relative SYSTEM id even though the parser resolves it against the document's
	// absolute base URI (which ParseFile always sets). The resolved name is an
	// absolute path that os.DirFS rejects as a non-fs.ValidPath name; the parser
	// retries with the base-relative name so the confined FS loads the external DTD.
	t.Run("loads a relative system ID", func(t *testing.T) {
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
	})

	// A nested external parameter entity that sits in a subdirectory must load
	// through a confined os.DirFS rooted at the document directory. The retry
	// relativizes against the FIXED document-root base, not the nested resource's
	// own moving base, so it yields "dtd/declarations.dtd" (root-relative) rather
	// than "declarations.dtd" (relative to dtd/, which does not exist at the root).
	t.Run("loads a nested relative system ID", func(t *testing.T) {
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
	})

	// The supported confined-FS document base is an ABSOLUTE path (or file: URI): the
	// base-relative retry re-relativizes an absolute base to recover the original
	// relative name. This is the positive counterpart to the relative-base scope note
	// on openExternalResource — with an absolute base, a relative SYSTEM id that
	// resolution turns into an absolute path is recovered and the confined FS loads
	// it. (A RELATIVE document base is out of scope: BuildURI yields a valid-but-absent
	// relative path that fails with fs.ErrNotExist, not fs.ErrInvalid, so the retry
	// never fires — the deferred root-aware helium.DirFS(root) adapter is the general
	// fix for an FS rooted elsewhere than the document directory.)
	t.Run("an absolute base loads a relative system ID", func(t *testing.T) {
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
	})

	// A SYSTEM id that resolves outside the FS root is refused. Here it is declared
	// as an absolute path, so it is not retry-eligible (originally absolute) and the
	// retry never fires — and even if it were, the base-relative name would ascend
	// via ".." and fail fs.ValidPath, so the confinement holds either way. The
	// out-of-tree file exists and is readable, proving the refusal is the guard, not a
	// missing file. (Neither guard is a symlink sandbox — os.DirFS follows an in-root
	// symlink out of the root; only os.Root.FS confines symlinks. See
	// TestDirFSFollowsSymlinkButRootFSConfines.)
	t.Run("refuses path traversal", func(t *testing.T) {
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
	})

	// A network-scheme SYSTEM id is refused before any Open, independent of the FS,
	// when network access is forbidden (the default).
	t.Run("refuses the network", func(t *testing.T) {
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
	})

	// The base-relative retry name is subject to the same network-access guard as
	// the primary name: a SYSTEM id whose retry name carries a network scheme is
	// refused before any Open, so it never reaches a network-capable caller FS, and
	// the parse returns ErrNetworkAccessForbidden.
	t.Run("refuses a network-scheme retry name", func(t *testing.T) {
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
	})

	// A SYSTEM id with leading XML whitespace before a network scheme (" http://x",
	// "\thttp://x") must NOT slip past the NONET gate: the whitespace is stripped
	// before the scheme check, so the network resource is refused before it can
	// reach a network-capable caller fs.FS. A genuine (whitespace-free) relative id
	// still loads normally.
	t.Run("refuses a whitespace-prefixed network scheme", func(t *testing.T) {
		for _, sysID := range []string{" http://x/evil.dtd", "\thttp://x/evil.dtd"} {
			doc := `<?xml version="1.0"?>` + "\n" +
				`<!DOCTYPE doc SYSTEM "` + sysID + `">` + "\n" +
				`<doc>hello</doc>`

			var opened []string
			_, err := helium.NewParser().
				BlockXXE(false).
				LoadExternalDTD(true).
				FS(validPathFS{content: []byte("<!ELEMENT doc (#PCDATA)>\n"), opened: &opened}).
				Parse(t.Context(), []byte(doc))

			require.ErrorIsf(t, err, helium.ErrNetworkAccessForbidden,
				"whitespace-prefixed network SYSTEM id %q must be refused under NONET", sysID)
			for _, n := range opened {
				require.NotContainsf(t, []string{"http", "https", "ftp"}, schemeOf(strings.TrimLeft(n, " \t\r\n")),
					"a whitespace-prefixed network id %q must be refused before it reaches Open (opened %q)", sysID, n)
			}
		}

		// A legitimate relative SYSTEM id (no whitespace) still loads through the same
		// network-capable FS.
		doc := `<?xml version="1.0"?>` + "\n" +
			`<!DOCTYPE doc SYSTEM "sub.dtd">` + "\n" +
			`<doc>hello</doc>`
		var opened []string
		parsed, err := helium.NewParser().
			BlockXXE(false).
			LoadExternalDTD(true).
			FS(validPathFS{content: []byte("<!ELEMENT doc (#PCDATA)>\n"), opened: &opened}).
			Parse(t.Context(), []byte(doc))

		require.NoError(t, err)
		require.NotNil(t, parsed)
		require.NotNil(t, parsed.ExtSubset(), "a relative SYSTEM id must still load")
		require.Contains(t, opened, "sub.dtd")
	})

	// An originally-ABSOLUTE SYSTEM id that names an in-root file is NEVER retried,
	// even though its base-relative form ("sub.dtd") would be a valid fs.ValidPath
	// that os.DirFS could open. Only an originally-relative reference is eligible for
	// the confined-FS base-relative retry; eligibility — not the fs.ValidPath shape
	// of the derived name — enforces the promise (filepath.Rel would happily
	// re-relativize the in-root absolute id). The confinement property still holds,
	// but the documented "absolute SYSTEM id is never retried" promise is now true by
	// construction. The in-root file exists and is readable, proving the refusal is
	// the eligibility gate, not a missing file.
	t.Run("does not retry an in-root absolute system ID", func(t *testing.T) {
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
	})

	// A one-letter-scheme SYSTEM id ("x:opaque") is a valid absolute URI per RFC
	// 3986 (scheme = ALPHA *( ALPHA / DIGIT / "+" / "-" / "." )). It must NEVER be
	// retried through the confined FS, even when a catalog remaps it to an in-root
	// absolute DTD whose base-relative form would open. This is the external-subset
	// path: eligibility is captured from the DECLARED id ("x:opaque") before catalog
	// mapping. The in-root file exists and is readable, so a load would succeed if
	// the retry fired — proving the refusal is the eligibility gate, not a miss.
	t.Run("does not retry a scheme system ID in the external subset", func(t *testing.T) {
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
	})

	// The ResolveEntity general-entity path: an external general entity declared
	// with a one-letter-scheme SYSTEM id ("x:opaque") that a catalog remaps to an
	// in-root absolute file must NOT be retried. Its eligibility is gated on the
	// declared id via parserCtx.extRefRelative.
	t.Run("does not retry a scheme system ID in a general entity", func(t *testing.T) {
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
	})

	// The ResolveEntity parameter-entity path: an external parameter entity declared
	// with a one-letter-scheme SYSTEM id ("x:opaque") that a catalog remaps to an
	// in-root absolute DTD must NOT be retried. Its eligibility is gated on the
	// declared id (loadExternalParameterEntityContent → systemIDRetryEligible).
	t.Run("does not retry a scheme system ID in a parameter entity", func(t *testing.T) {
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
	})

	// helium.DirFS confines to an ARBITRARY directory, not only the document's own
	// directory. Here the document lives in one directory and its external DTD in a
	// SEPARATE trusted directory, referenced by an ABSOLUTE SYSTEM id inside that
	// directory. A bare os.DirFS/os.Root.FS rooted there would reject the absolute
	// name (fs.ErrInvalid) and the base-relative retry — which relativizes against
	// the DOCUMENT directory — would not recover it; DirFS serves the in-root
	// absolute name directly, so the external subset loads.
	t.Run("a dir FS loads an in-root absolute system ID", func(t *testing.T) {
		dtdDir := t.TempDir()
		dtdPath := filepath.Join(dtdDir, "sub.dtd")
		require.NoError(t, os.WriteFile(dtdPath, []byte("<!ELEMENT doc (#PCDATA)>\n"), 0o600))

		docDir := t.TempDir()
		docPath := filepath.Join(docDir, "doc.xml")
		require.NoError(t, os.WriteFile(docPath, []byte(
			`<?xml version="1.0"?>`+"\n"+
				`<!DOCTYPE doc SYSTEM "`+dtdPath+`">`+"\n"+
				`<doc>hello</doc>`), 0o600))

		doc, err := helium.NewParser().
			BlockXXE(false).
			LoadExternalDTD(true).
			FS(helium.DirFS(dtdDir)).
			ParseFile(t.Context(), docPath)

		require.NoError(t, err)
		require.NotNil(t, doc)
		require.NotNil(t, doc.ExtSubset(),
			"helium.DirFS must serve an in-root absolute SYSTEM id from an arbitrary root directory")
	})

	// helium.DirFS refuses an absolute SYSTEM id that resolves OUTSIDE its root: the
	// out-of-root file exists and is readable, so the refusal is the confinement
	// guard, not a missing file. The parse stays lenient (no fatal error).
	t.Run("a dir FS refuses an out-of-root absolute system ID", func(t *testing.T) {
		outside := t.TempDir()
		secret := filepath.Join(outside, "secret.dtd")
		require.NoError(t, os.WriteFile(secret, []byte("<!ELEMENT doc EMPTY>\n"), 0o600))

		root := t.TempDir()
		docDir := t.TempDir()
		docPath := filepath.Join(docDir, "doc.xml")
		require.NoError(t, os.WriteFile(docPath, []byte(
			`<?xml version="1.0"?>`+"\n"+
				`<!DOCTYPE doc SYSTEM "`+secret+`">`+"\n"+
				`<doc>hello</doc>`), 0o600))

		doc, err := helium.NewParser().
			BlockXXE(false).
			LoadExternalDTD(true).
			FS(helium.DirFS(root)).
			ParseFile(t.Context(), docPath)

		require.NoError(t, err)
		require.NotNil(t, doc)
		require.Nil(t, doc.ExtSubset(),
			"helium.DirFS must not load an out-of-root absolute-path SYSTEM id")
	})

	// os.DirFS is NOT a symlink-confinement boundary: it follows an in-root symlink
	// that points outside the root and reads the out-of-root target through a plain
	// valid fs.ValidPath name. os.Root.FS (Go 1.24+) IS symlink-safe and refuses the
	// same open. This documents why the parser recommends os.Root.FS for confinement
	// and why the fs.ValidPath retry guard is a path-escape guard, not a sandbox.
	t.Run("a dir FS follows a symlink but a root FS confines it", func(t *testing.T) {
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
	})

	// PermissiveRoot still loads an external subset via its absolute resolved path:
	// the base-relative retry only fires for an fs.ErrInvalid rejection, which
	// os.Open-backed PermissiveRoot never returns, so its behavior is unchanged.
	t.Run("a permissive FS loads an absolute path", func(t *testing.T) {
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
	})

	// The direct (non-catalog) ResolveEntity branch opens the entity's raw resolved
	// systemID first — a "file:" URI verbatim — so a permissive caller FS keyed on
	// the file-URI name still resolves the entity. Normalization to a local path is
	// applied only for the confined-FS base-relative retry, never to the primary.
	t.Run("a direct ResolveEntity preserves the file URI primary name", func(t *testing.T) {
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
	})
}

func TestSafeDefaults(t *testing.T) {
	t.Parallel()

	// locks in the secure-by-default behavior of NewParser. Each
	// subtest asserts a guarantee that untrusted input relies on; a regression that
	// re-enables external loading, the host filesystem, or unbounded depth by
	// default must fail here.
	t.Run("defaults", func(t *testing.T) {
		// extDTD is an external subset that would default an attribute if loaded.
		// Observing the attribute on the root element proves the DTD was read.
		const extDTDName = "ext.dtd"
		const extDTD = `<!ELEMENT doc EMPTY>` + "\n" + `<!ATTLIST doc x CDATA "default">`
		const dtdDoc = `<?xml version="1.0"?>` + "\n" + `<!DOCTYPE doc SYSTEM "ext.dtd">` + "\n" + `<doc/>`

		t.Run("external entity blocked by default", func(t *testing.T) {
			t.Parallel()

			const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ENTITY ext SYSTEM "ext.xml">
]>
<doc>&ext;</doc>`

			resolved := false
			s := sax.New()
			s.SetOnResolveEntity(sax.ResolveEntityFunc(func(_ context.Context, _, systemID string) (sax.ParseInput, error) {
				resolved = true
				return newStringParseInput("<inner/>", systemID), nil
			}))

			// SubstituteEntities is requested but BlockXXE is NOT cleared, so the
			// default XXE guard must keep the external entity from being fetched.
			_, _ = helium.NewParser().SAXHandler(s).SubstituteEntities(true).Parse(t.Context(), []byte(input))
			require.False(t, resolved, "external entity must not be resolved by default")
		})

		t.Run("external DTD blocked by default", func(t *testing.T) {
			t.Parallel()

			resolved := false
			s := sax.New()
			s.SetOnResolveEntity(sax.ResolveEntityFunc(func(_ context.Context, _, systemID string) (sax.ParseInput, error) {
				resolved = true
				return newStringParseInput(extDTD, systemID), nil
			}))

			// LoadExternalDTD is requested but BlockXXE is NOT cleared.
			_, _ = helium.NewParser().SAXHandler(s).LoadExternalDTD(true).DefaultDTDAttributes(true).Parse(t.Context(), []byte(dtdDoc))
			require.False(t, resolved, "external DTD must not be loaded by default")
		})

		t.Run("default FS denies even with XXE lifted", func(t *testing.T) {
			t.Parallel()

			// XXE is lifted and external-DTD loading requested, but no FS is
			// supplied. The default deny-all FS must still prevent the load, so the
			// ATTLIST default never materializes.
			doc, err := helium.NewParser().
				BlockXXE(false).
				LoadExternalDTD(true).
				DefaultDTDAttributes(true).
				Parse(t.Context(), []byte(dtdDoc))
			require.NoError(t, err)
			root := doc.DocumentElement()
			require.NotNil(t, root)
			_, ok := root.GetAttribute("x")
			require.False(t, ok, "default deny-all FS must block external DTD loading")
		})

		t.Run("explicit FS re-enables loading", func(t *testing.T) {
			t.Parallel()

			// Same parser, but an explicit FS is supplied: the DTD now loads and the
			// defaulted attribute appears. This isolates the FS as the gate.
			fsys := fstest.MapFS{extDTDName: &fstest.MapFile{Data: []byte(extDTD)}}
			doc, err := helium.NewParser().
				BlockXXE(false).
				LoadExternalDTD(true).
				DefaultDTDAttributes(true).
				FS(fsys).
				Parse(t.Context(), []byte(dtdDoc))
			require.NoError(t, err)
			root := doc.DocumentElement()
			require.NotNil(t, root)
			x, ok := root.GetAttribute("x")
			require.True(t, ok, "an explicit FS must re-enable external DTD loading")
			require.Equal(t, "default", x)
		})

		t.Run("FS(nil) restores deny-all", func(t *testing.T) {
			t.Parallel()

			fsys := fstest.MapFS{extDTDName: &fstest.MapFile{Data: []byte(extDTD)}}
			// Supply a permissive FS, then reset to nil: nil must restore the
			// deny-all default, not the historical permissive root.
			doc, err := helium.NewParser().
				BlockXXE(false).
				LoadExternalDTD(true).
				DefaultDTDAttributes(true).
				FS(fsys).
				FS(nil).
				Parse(t.Context(), []byte(dtdDoc))
			require.NoError(t, err)
			root := doc.DocumentElement()
			require.NotNil(t, root)
			_, ok := root.GetAttribute("x")
			require.False(t, ok, "FS(nil) must restore the deny-all default")
		})

		t.Run("element depth capped at 256 by default", func(t *testing.T) {
			t.Parallel()

			atLimit := strings.Repeat("<a>", 256) + strings.Repeat("</a>", 256)
			_, err := helium.NewParser().Parse(t.Context(), []byte(atLimit))
			require.NoError(t, err, "nesting at the 256 default limit must parse")

			overLimit := strings.Repeat("<a>", 257) + strings.Repeat("</a>", 257)
			_, err = helium.NewParser().Parse(t.Context(), []byte(overLimit))
			require.Error(t, err, "nesting past the 256 default limit must be rejected")
			require.Contains(t, err.Error(), "exceeded max depth")
		})

		t.Run("MaxDepth(0) keeps the 256 default cap", func(t *testing.T) {
			t.Parallel()

			deep := strings.Repeat("<a>", 300) + strings.Repeat("</a>", 300)
			_, err := helium.NewParser().MaxDepth(0).Parse(t.Context(), []byte(deep))
			require.Error(t, err, "MaxDepth(0) must select the 256 default, not disable the cap")
			require.Contains(t, err.Error(), "exceeded max depth")
		})

		t.Run("MaxDepth(-1) disables the cap", func(t *testing.T) {
			t.Parallel()

			deep := strings.Repeat("<a>", 512) + strings.Repeat("</a>", 512)
			_, err := helium.NewParser().MaxDepth(-1).Parse(t.Context(), []byte(deep))
			require.NoError(t, err, "a negative MaxDepth must opt out of the depth cap")
		})
	})

	// the public escape hatch opens host paths via
	// os.Open — including absolute paths that os.DirFS would reject — so callers
	// that deliberately need the historical behavior can restore it.
	t.Run("permissive FS", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "x.txt")
		require.NoError(t, os.WriteFile(path, []byte("hi"), 0o600))

		f, err := helium.PermissiveFS().Open(path)
		require.NoError(t, err, "PermissiveFS must open an absolute host path")
		require.NoError(t, f.Close())

		_, err = helium.PermissiveFS().Open(filepath.Join(dir, "does-not-exist"))
		require.Error(t, err, "a missing path must still error")
	})
}

func TestAllowNetwork(t *testing.T) {
	t.Parallel()

	// A network-scheme external DTD subset must not reach the fs.FS when
	// AllowNetwork is false, and must reach it when AllowNetwork is true.
	t.Run("gates an external DTD", func(t *testing.T) {
		doc := `<?xml version="1.0"?>` + "\n" +
			`<!DOCTYPE root SYSTEM "http://example.com/x.dtd">` + "\n" +
			`<root/>`
		dtd := []byte(`<!ELEMENT root EMPTY>`)

		var openedOff []string
		_, err := helium.NewParser().
			BlockXXE(false).
			LoadExternalDTD(true).
			FS(netGateFS{content: dtd, opened: &openedOff}).
			Parse(t.Context(), []byte(doc))
		require.False(t, openedNetwork(openedOff), "the network DTD id must never reach the fs.FS when AllowNetwork is off")
		_ = err

		var openedOn []string
		_, err = helium.NewParser().
			BlockXXE(false).
			LoadExternalDTD(true).
			AllowNetwork(true).
			FS(netGateFS{content: dtd, opened: &openedOn}).
			Parse(t.Context(), []byte(doc))
		require.NoError(t, err)
		require.True(t, openedNetwork(openedOn), "the network DTD id must reach the fs.FS when AllowNetwork is on")
	})

	// A network-scheme external general entity must not reach the fs.FS when
	// AllowNetwork is false (the default), and must reach it when AllowNetwork is
	// true. The fs.FS here would happily serve any name, so the only thing that can
	// keep the network id from being opened is the NONET scheme gate.
	t.Run("gates an external entity", func(t *testing.T) {
		doc := `<?xml version="1.0"?>` + "\n" +
			`<!DOCTYPE root [<!ENTITY e SYSTEM "http://example.com/x.ent">]>` + "\n" +
			`<root>&e;</root>`
		entity := []byte(`<child>net</child>`)

		// Default (AllowNetwork off): the network entity must be refused before Open.
		var openedOff []string
		_, err := helium.NewParser().
			BlockXXE(false).
			LoadExternalDTD(true).
			SubstituteEntities(true).
			FS(netGateFS{content: entity, opened: &openedOff}).
			Parse(t.Context(), []byte(doc))
		require.Error(t, err, "a network-scheme entity must be refused when AllowNetwork is off")
		require.False(t, openedNetwork(openedOff), "the network id must never reach the fs.FS when AllowNetwork is off")

		// AllowNetwork on: the entity load reaches the fs.FS and expands.
		var openedOn []string
		parsed, err := helium.NewParser().
			BlockXXE(false).
			LoadExternalDTD(true).
			SubstituteEntities(true).
			AllowNetwork(true).
			FS(netGateFS{content: entity, opened: &openedOn}).
			Parse(t.Context(), []byte(doc))
		require.NoError(t, err, "a network-scheme entity must load when AllowNetwork is on")
		require.True(t, openedNetwork(openedOn), "the network id must reach the fs.FS when AllowNetwork is on")
		require.NotNil(t, parsed)
		root := parsed.DocumentElement()
		require.NotNil(t, root)
		child := root.FirstChild()
		require.NotNil(t, child, "the network entity replacement text must have expanded")
		require.Equal(t, "child", child.(*helium.Element).LocalName())
	})

	// A non-network (bare filename) external entity must load regardless of the
	// AllowNetwork setting: the scheme gate only bites http/https/ftp.
	t.Run("allows a local entity", func(t *testing.T) {
		doc := `<?xml version="1.0"?>` + "\n" +
			`<!DOCTYPE root [<!ENTITY e SYSTEM "local.ent">]>` + "\n" +
			`<root>&e;</root>`
		entity := []byte(`<child>local</child>`)

		for _, allow := range []bool{false, true} {
			var opened []string
			p := helium.NewParser().
				BlockXXE(false).
				LoadExternalDTD(true).
				SubstituteEntities(true).
				FS(netGateFS{content: entity, opened: &opened})
			p = p.AllowNetwork(allow)
			parsed, err := p.Parse(t.Context(), []byte(doc))
			require.NoErrorf(t, err, "a non-network entity must load with AllowNetwork(%v)", allow)
			require.NotNil(t, parsed)
			require.NotEmptyf(t, opened, "the local id must reach the fs.FS with AllowNetwork(%v)", allow)
			require.Falsef(t, openedNetwork(opened), "a bare filename is not a network id (AllowNetwork(%v))", allow)
		}
	})
}

func TestBlockXXE(t *testing.T) {
	t.Parallel()

	t.Run("blocked by default", func(t *testing.T) {
		t.Run("entity", func(t *testing.T) {
			t.Parallel()

			const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ENTITY ext SYSTEM "ext.xml">
]>
<doc>&ext;</doc>`

			resolved := false
			s := sax.New()
			s.SetOnResolveEntity(sax.ResolveEntityFunc(func(_ context.Context, publicID, systemID string) (sax.ParseInput, error) {
				resolved = true
				return newStringParseInput("<inner>hello</inner>", systemID), nil
			}))

			p := helium.NewParser().SAXHandler(s).SubstituteEntities(true).BlockXXE(true)
			_, err := p.Parse(t.Context(), []byte(input))
			// With BlockXXE, external entity loading is blocked.
			// The entity reference remains unresolved; no error but external content not loaded.
			_ = err
			require.False(t, resolved, "ResolveEntity should not be called with BlockXXE")
		})

		t.Run("external DTD", func(t *testing.T) {
			t.Parallel()

			const input = `<?xml version="1.0"?>
<!DOCTYPE doc SYSTEM "ext.dtd">
<doc/>`

			resolved := false
			s := sax.New()
			s.SetOnResolveEntity(sax.ResolveEntityFunc(func(_ context.Context, publicID, systemID string) (sax.ParseInput, error) {
				resolved = true
				return newStringParseInput("<!ELEMENT doc EMPTY>", systemID), nil
			}))

			p := helium.NewParser().SAXHandler(s).LoadExternalDTD(true).BlockXXE(true)
			_, err := p.Parse(t.Context(), []byte(input))
			_ = err
			require.False(t, resolved, "external DTD should not be loaded with BlockXXE")
		})
	})

	// against a sandbox escape where an external
	// entity reached from inside another external entity's sub-parse was resolved
	// via the default permissive os.Open path instead of the parser's configured FS.
	t.Run("the entity sub-parser stays in the FS sandbox", func(t *testing.T) {
		// A real on-disk file outside any configured sandbox. If the nested external
		// entity escapes the FS via os.Open, its contents leak into the document.
		dir := t.TempDir()
		secretPath := filepath.Join(dir, "secret.xml")
		require.NoError(t, os.WriteFile(secretPath, []byte("<leaked>TOPSECRET</leaked>"), 0o600))

		t.Run("nested external entity confined to configured FS", func(t *testing.T) {
			t.Parallel()

			// outer.xml lives inside the sandbox and references &secret;, which is an
			// external SYSTEM entity pointing at the absolute on-disk path OUTSIDE the
			// sandbox. The sub-parse of outer.xml must resolve &secret; through the
			// same configured FS, which does not contain that path, so it must not be
			// readable and must not leak into the document.
			input := `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ENTITY secret SYSTEM "` + secretPath + `">
  <!ENTITY outer SYSTEM "outer.xml">
]>
<doc>&outer;</doc>`

			rfs := &recordingFS{inner: fstest.MapFS{
				"outer.xml": &fstest.MapFile{Data: []byte(`<wrap>&secret;</wrap>`)},
			}}
			p := helium.NewParser().BlockXXE(false).SubstituteEntities(true).FS(rfs)
			doc, _ := p.Parse(t.Context(), []byte(input))

			// The on-disk secret must never surface in the resulting document.
			if doc != nil {
				var buf bytes.Buffer
				require.NoError(t, helium.NewWriter().WriteTo(&buf, doc))
				require.NotContains(t, buf.String(), "TOPSECRET",
					"out-of-sandbox file leaked into document")
			}
			// Resolution of the nested external entity must be routed through the
			// configured FS (recorded here). A leak would happen via os.Open, which
			// bypasses the recording FS entirely, so the path would never be seen.
			require.True(t, rfs.wasOpened(secretPath),
				"nested external entity escaped the configured FS sandbox")
		})

		t.Run("in-sandbox nested external entity still resolves", func(t *testing.T) {
			t.Parallel()

			// A legitimate external entity available within the configured FS must
			// still resolve when reached from inside another external entity.
			input := `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ENTITY allowed SYSTEM "allowed.xml">
  <!ENTITY outer SYSTEM "outer.xml">
]>
<doc>&outer;</doc>`

			rfs := &recordingFS{inner: fstest.MapFS{
				"outer.xml":   &fstest.MapFile{Data: []byte(`<wrap>&allowed;</wrap>`)},
				"allowed.xml": &fstest.MapFile{Data: []byte("<inner>ok</inner>")},
			}}
			p := helium.NewParser().BlockXXE(false).SubstituteEntities(true).FS(rfs)
			doc, err := p.Parse(t.Context(), []byte(input))
			require.NoError(t, err)
			require.True(t, rfs.wasOpened("allowed.xml"),
				"in-sandbox nested external entity was not loaded through the configured FS")

			var buf bytes.Buffer
			require.NoError(t, helium.NewWriter().WriteTo(&buf, doc))
			out := buf.String()
			require.Contains(t, out, "<inner", "in-sandbox nested external entity did not expand")
			require.Contains(t, out, ">ok</inner>", "in-sandbox nested external entity content missing")
		})
	})

	t.Run("the parser FS is honored", func(t *testing.T) {
		t.Run("external DTD loaded from custom FS", func(t *testing.T) {
			t.Parallel()

			const input = `<?xml version="1.0"?>
<!DOCTYPE doc SYSTEM "ext.dtd">
<doc/>`

			fsys := fstest.MapFS{
				extDTDName: &fstest.MapFile{Data: []byte("<!ELEMENT doc EMPTY>\n")},
			}

			p := helium.NewParser().LoadExternalDTD(true).FS(fsys)
			_, err := p.Parse(t.Context(), []byte(input))
			require.NoError(t, err)
		})

		t.Run("FS error surfaces as missing resource (silent)", func(t *testing.T) {
			t.Parallel()

			const input = `<?xml version="1.0"?>
<!DOCTYPE doc SYSTEM "nope.dtd">
<doc/>`

			p := helium.NewParser().LoadExternalDTD(true).FS(fstest.MapFS{})
			_, err := p.Parse(t.Context(), []byte(input))
			require.NoError(t, err, "missing external DTD is silently ignored, same as before")
		})

		t.Run("nil FS restores default", func(t *testing.T) {
			t.Parallel()

			const input = `<?xml version="1.0"?><doc/>`

			// Set a custom FS then clear it; parsing a doc that needs no external
			// resources must still work.
			p := helium.NewParser().FS(fstest.MapFS{}).FS(nil)
			_, err := p.Parse(t.Context(), []byte(input))
			require.NoError(t, err)
		})

		// Compile-time check that fs.FS is the parameter type.
		var _ = helium.NewParser().FS(fs.FS(fstest.MapFS{}))
	})

	t.Run("XInclude injection", func(t *testing.T) {
		const src = `<doc xmlns:xi="http://www.w3.org/2001/XInclude"><xi:include href="x.xml"/></doc>`

		t.Run("processor runs and receives the parsed document", func(t *testing.T) {
			rec := &recordingXInclude{n: 1}
			doc, err := helium.NewParser().XInclude(rec).Parse(context.Background(), []byte(src))
			require.NoError(t, err)
			require.NotNil(t, doc)
			require.Equal(t, 1, rec.calls)
			require.Same(t, doc, rec.doc, "the processor must run on the document Parse returns")
		})

		t.Run("no processor means no XInclude step", func(t *testing.T) {
			doc, err := helium.NewParser().Parse(context.Background(), []byte(src))
			require.NoError(t, err)
			require.NotNil(t, doc)
		})

		t.Run("nil processor disables processing", func(t *testing.T) {
			doc, err := helium.NewParser().XInclude(nil).Parse(context.Background(), []byte(src))
			require.NoError(t, err)
			require.NotNil(t, doc)
		})

		t.Run("processor error propagates from Parse", func(t *testing.T) {
			rec := &recordingXInclude{err: errors.New("xinclude boom")}
			_, err := helium.NewParser().XInclude(rec).Parse(context.Background(), []byte(src))
			require.Error(t, err)
			require.Contains(t, err.Error(), "xinclude boom")
		})

		t.Run("processor cancellation returns a nil document", func(t *testing.T) {
			// A cancelled/timed-out post-parse step must follow Parse's contract:
			// the context error with a nil document, never a partial tree.
			rec := &recordingXInclude{err: context.Canceled}
			doc, err := helium.NewParser().XInclude(rec).Parse(context.Background(), []byte(src))
			require.ErrorIs(t, err, context.Canceled)
			require.Nil(t, doc, "a cancelled post-parse step must not return a partial document")
		})

		t.Run("runs on the ParseReader path too", func(t *testing.T) {
			rec := &recordingXInclude{n: 1}
			doc, err := helium.NewParser().XInclude(rec).ParseReader(context.Background(), strings.NewReader(src))
			require.NoError(t, err)
			require.Equal(t, 1, rec.calls)
			require.Same(t, doc, rec.doc)
		})
	})
}
