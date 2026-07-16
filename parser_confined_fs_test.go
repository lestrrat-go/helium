package helium_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

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

// The base-relative retry is confined: a SYSTEM id that resolves outside the FS
// root (absolute, or ascending via "..") is not a valid fs.ValidPath, so it is
// never retried and the confined FS refuses it. The out-of-tree file exists and
// is readable, proving the refusal is the confinement, not a missing file.
func TestConfinedDirFSRefusesTraversal(t *testing.T) {
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

	// The load is refused (a confined FS cannot reach outside its root), but the
	// parse stays lenient: no fatal error, document returned, subset not loaded.
	require.NoError(t, err)
	require.NotNil(t, doc)
	require.Nil(t, doc.ExtSubset(),
		"a confined os.DirFS must not load an external subset outside its root")
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
