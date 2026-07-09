package helium_test

import (
	"testing"
	"testing/fstest"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// extEntityName is the external general-entity filename shared by the TextDecl
// tests.
const extEntityName = "ent.ent"

func extEntityDoc() string {
	return `<?xml version="1.0"?>` + "\n" +
		`<!DOCTYPE root [<!ENTITY e SYSTEM "` + extEntityName + `">]>` + "\n" +
		`<root>&e;</root>`
}

func parseExtEntity(t *testing.T, entBytes []byte) (*helium.Document, error) {
	t.Helper()
	fsys := fstest.MapFS{extEntityName: &fstest.MapFile{Data: entBytes}}
	return helium.NewParser().
		BlockXXE(false).
		LoadExternalDTD(true).
		SubstituteEntities(true).
		FS(fsys).
		Parse(t.Context(), []byte(extEntityDoc()))
}

// An external parsed general entity's replacement text may begin with a TextDecl
// — '<?xml' VersionInfo? EncodingDecl S? '?>' — where VersionInfo is OPTIONAL,
// EncodingDecl REQUIRED, and no StandaloneDecl is permitted. A version-less
// TextDecl must be consumed and its body parsed as the replacement text, not
// rejected for a missing version pseudo-attribute.
func TestExternalGeneralEntityTextDeclNoVersion(t *testing.T) {
	t.Parallel()

	parsed, err := parseExtEntity(t, []byte(`<?xml encoding="UTF-8"?><child>hi</child>`))
	require.NoError(t, err, "a version-less TextDecl on an external entity must be accepted")
	require.NotNil(t, parsed)

	root := parsed.DocumentElement()
	require.NotNil(t, root)
	require.Equal(t, "root", root.LocalName())
	child := root.FirstChild()
	require.NotNil(t, child, "the entity replacement text must have expanded into a child element")
	require.Equal(t, "child", child.(*helium.Element).LocalName())
	require.Equal(t, "hi", string(child.Content()))
}

// A version-bearing TextDecl at the start of an external general entity is
// equally valid and must also be accepted.
func TestExternalGeneralEntityTextDeclWithVersion(t *testing.T) {
	t.Parallel()

	parsed, err := parseExtEntity(t, []byte(`<?xml version="1.0" encoding="UTF-8"?><child>hi</child>`))
	require.NoError(t, err)
	require.NotNil(t, parsed)
	require.Equal(t, "hi", string(parsed.DocumentElement().FirstChild().Content()))
}

// A TextDecl carrying a StandaloneDecl is forbidden by the grammar (a TextDecl
// permits no 'standalone' pseudo-attribute) and must be rejected.
func TestExternalGeneralEntityTextDeclStandaloneRejected(t *testing.T) {
	t.Parallel()

	_, err := parseExtEntity(t, []byte(`<?xml encoding="UTF-8" standalone="yes"?><child>hi</child>`))
	require.Error(t, err, "a standalone pseudo-attribute in an external-entity TextDecl must be rejected")
}

// A version-only TextDecl (no EncodingDecl, which is required in a TextDecl) must
// be rejected.
func TestExternalGeneralEntityTextDeclMissingEncodingRejected(t *testing.T) {
	t.Parallel()

	_, err := parseExtEntity(t, []byte(`<?xml version="1.0"?><child>hi</child>`))
	require.Error(t, err, "a TextDecl missing the required encoding declaration must be rejected")
}
