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

// An IGNORE conditional section validates literal content as decoded runes. This
// keeps a legal non-ASCII XML 1.1 rune intact while still rejecting a raw XML
// 1.1 restricted character, including inside a nested IGNORE section.
func TestExternalSubsetIgnoreLiteralRuneValidation(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		dtd     string
		wantErr bool
	}{
		{
			name: "accepts legal multibyte rune in nested ignore section",
			dtd: `<?xml version="1.1" encoding="UTF-8"?>
<![IGNORE[outer <![IGNORE[` + "\u0100" + `]]> tail]]>`,
		},
		{
			name: "rejects raw restricted character",
			dtd: `<?xml version="1.1" encoding="UTF-8"?>
<![IGNORE[` + "\x7f" + `]]>`,
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fsys := fstest.MapFS{extSubsetName: &fstest.MapFile{Data: []byte(tc.dtd)}}
			input := `<?xml version="1.1"?>
<!DOCTYPE doc SYSTEM "` + extSubsetName + `"><doc/>`
			_, err := helium.NewParser().
				BlockXXE(false).
				LoadExternalDTD(true).
				FS(fsys).
				Parse(t.Context(), []byte(input))
			if tc.wantErr {
				require.ErrorIs(t, err, helium.ErrInvalidChar)
				return
			}
			require.NoError(t, err)
		})
	}
}

// A TextDecl version applies to the external DTD only. An XML 1.0 DTD under an
// XML 1.1 document retains XML 1.0 character-reference rules, and the document
// returns to its XML 1.1 rules after the DTD input is drained.
func TestExternalSubsetTextDeclVersionScoped(t *testing.T) {
	t.Parallel()

	t.Run("XML 1.0 DTD retains XML 1.0 character-reference rules", func(t *testing.T) {
		t.Parallel()

		const dtd = `<?xml version="1.0" encoding="UTF-8"?>
<!ENTITY control "&#1;">`
		fsys := fstest.MapFS{extSubsetName: &fstest.MapFile{Data: []byte(dtd)}}
		input := `<?xml version="1.1"?>
<!DOCTYPE doc SYSTEM "` + extSubsetName + `"><doc/>`

		_, err := helium.NewParser().
			BlockXXE(false).
			LoadExternalDTD(true).
			FS(fsys).
			Parse(t.Context(), []byte(input))
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid XML char value 1")
	})

	t.Run("document XML version is restored", func(t *testing.T) {
		t.Parallel()

		const dtd = `<?xml version="1.0" encoding="UTF-8"?>
<!ELEMENT doc ANY>`
		fsys := fstest.MapFS{extSubsetName: &fstest.MapFile{Data: []byte(dtd)}}
		input := `<?xml version="1.1"?>
<!DOCTYPE doc SYSTEM "` + extSubsetName + `"><doc>&#1;</doc>`

		parsed, err := helium.NewParser().
			BlockXXE(false).
			LoadExternalDTD(true).
			FS(fsys).
			Parse(t.Context(), []byte(input))
		require.NoError(t, err)
		require.Equal(t, "1.1", parsed.Version())
	})
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
