package helium_test

import (
	"testing"
	"testing/fstest"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// extSubsetName is the external-subset filename shared by the TextDecl tests.
const extSubsetName = "sub.dtd"

func extSubsetDoc() string {
	return `<?xml version="1.0"?>` + "\n" +
		`<!DOCTYPE doc SYSTEM "` + extSubsetName + `">` + "\n" +
		`<doc>&greeting;</doc>`
}

// An external DTD subset (loaded via <!DOCTYPE ... SYSTEM>) may begin with a
// TextDecl — '<?xml' VersionInfo? EncodingDecl S? '?>' — where VersionInfo is
// optional, EncodingDecl required, and no StandaloneDecl is permitted. It must
// be consumed, not rejected as a misplaced XML declaration. libxml2 accepts such
// documents; the W3C XML conformance suite has many valid cases whose external
// DTD opens with "<?xml encoding=...?>".
func TestExternalSubsetTextDecl(t *testing.T) {
	t.Parallel()

	const dtd = `<?xml encoding="utf-8"?>
<!ELEMENT doc (#PCDATA)>
<!ENTITY greeting "hello from ext subset">
`
	fsys := fstest.MapFS{extSubsetName: &fstest.MapFile{Data: []byte(dtd)}}
	parsed, err := helium.NewParser().
		BlockXXE(false).
		LoadExternalDTD(true).
		SubstituteEntities(true).
		FS(fsys).
		Parse(t.Context(), []byte(extSubsetDoc()))
	require.NoError(t, err, "a TextDecl at the start of the external subset must be accepted")
	require.NotNil(t, parsed)

	root := parsed.DocumentElement()
	require.NotNil(t, root)
	require.Equal(t, "doc", root.LocalName())
	// The general entity declared in the external subset resolved, proving the
	// subset was parsed past its TextDecl rather than abandoned.
	require.Equal(t, "hello from ext subset", string(root.Content()))
}

// A version-bearing TextDecl (VersionInfo present) at the start of the external
// subset is equally valid and must also be accepted.
func TestExternalSubsetTextDeclWithVersion(t *testing.T) {
	t.Parallel()

	const dtd = `<?xml version="1.0" encoding="UTF-8"?>
<!ELEMENT doc (#PCDATA)>
<!ENTITY greeting "versioned">
`
	fsys := fstest.MapFS{extSubsetName: &fstest.MapFile{Data: []byte(dtd)}}
	parsed, err := helium.NewParser().
		BlockXXE(false).
		LoadExternalDTD(true).
		SubstituteEntities(true).
		FS(fsys).
		Parse(t.Context(), []byte(extSubsetDoc()))
	require.NoError(t, err)
	require.NotNil(t, parsed)
	require.Equal(t, "versioned", string(parsed.DocumentElement().Content()))
}

// A malformed TextDecl in the external subset (here version-only: EncodingDecl is
// required in a TextDecl) must be rejected AND the error must locate the external
// DTD file, matching every other declaration error reported from that resource.
func TestExternalSubsetTextDeclMalformedReportsFile(t *testing.T) {
	t.Parallel()

	const dtd = `<?xml version="1.0"?>
<!ELEMENT doc (#PCDATA)>
`
	fsys := fstest.MapFS{extSubsetName: &fstest.MapFile{Data: []byte(dtd)}}
	_, err := helium.NewParser().
		BlockXXE(false).
		LoadExternalDTD(true).
		SubstituteEntities(true).
		FS(fsys).
		Parse(t.Context(), []byte(extSubsetDoc()))
	require.Error(t, err)
	require.Contains(t, err.Error(), extSubsetName,
		"a malformed external-subset TextDecl error must name the DTD file")
}

// An unsupported encoding declared in the external-subset TextDecl must also fail
// with the DTD file located — the error travels the switchEncoding branch, not
// parseTextDecl, so it must be wrapped with the source URI too.
func TestExternalSubsetTextDeclUnsupportedEncodingReportsFile(t *testing.T) {
	t.Parallel()

	const dtd = `<?xml encoding="X-UNKNOWN-ENC"?>
<!ELEMENT doc (#PCDATA)>
`
	fsys := fstest.MapFS{extSubsetName: &fstest.MapFile{Data: []byte(dtd)}}
	_, err := helium.NewParser().
		BlockXXE(false).
		LoadExternalDTD(true).
		SubstituteEntities(true).
		FS(fsys).
		Parse(t.Context(), []byte(extSubsetDoc()))
	require.Error(t, err)
	require.Contains(t, err.Error(), extSubsetName,
		"an unsupported external-subset TextDecl encoding error must name the DTD file")
}
