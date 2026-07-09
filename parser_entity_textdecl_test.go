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

// parseExtEntityVersioned parses a document with the given XML-declaration
// version (empty for no declaration) that references an external general entity
// whose replacement text begins with entBytes.
func parseExtEntityVersioned(t *testing.T, docVersion string, entBytes []byte) (*helium.Document, error) {
	t.Helper()
	decl := ""
	if docVersion != "" {
		decl = `<?xml version="` + docVersion + `"?>` + "\n"
	}
	src := decl +
		`<!DOCTYPE root [<!ENTITY e SYSTEM "` + extEntityName + `">]>` + "\n" +
		`<root>&e;</root>`
	fsys := fstest.MapFS{extEntityName: &fstest.MapFile{Data: entBytes}}
	return helium.NewParser().
		BlockXXE(false).
		LoadExternalDTD(true).
		SubstituteEntities(true).
		FS(fsys).
		Parse(t.Context(), []byte(src))
}

// The XML §4.3.4 version-compatibility matrix: an external parsed entity's
// TextDecl version must not be LATER than the referencing document's. A 1.0
// document may not reference a 1.1 entity (fatal, W3C rmt-e2e-38); every other
// combination is accepted. The version comparison is against the ACTUAL document
// version, so a 1.1 document referencing a 1.1 (or 1.0) entity must be accepted —
// the TextDecl is decoded on a doc-less sub-context that is seeded with the
// parent document's version.
func TestExternalGeneralEntityTextDeclVersionMatrix(t *testing.T) {
	t.Parallel()

	const ent11 = `<?xml version="1.1" encoding="UTF-8"?><child>hi</child>`
	const ent10 = `<?xml version="1.0" encoding="UTF-8"?><child>hi</child>`
	const entNoVer = `<?xml encoding="UTF-8"?><child>hi</child>`

	for _, tc := range []struct {
		name       string
		docVersion string
		ent        string
		wantErr    bool
	}{
		{"1.0 doc + 1.1 entity is fatal", "1.0", ent11, true},
		{"no-decl doc (treated 1.0) + 1.1 entity is fatal", "", ent11, true},
		{"1.0 doc + 1.0 entity is ok", "1.0", ent10, false},
		{"1.0 doc + versionless entity is ok", "1.0", entNoVer, false},
		{"1.1 doc + 1.1 entity is ok", "1.1", ent11, false},
		{"1.1 doc + 1.0 entity is ok", "1.1", ent10, false},
		{"1.1 doc + versionless entity is ok", "1.1", entNoVer, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			doc, err := parseExtEntityVersioned(t, tc.docVersion, []byte(tc.ent))
			if tc.wantErr {
				require.Error(t, err, "expected a version mismatch")
				require.Contains(t, err.Error(), "version mismatch")
				return
			}
			require.NoError(t, err, "version combination must be accepted")
			require.NotNil(t, doc)
		})
	}
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
