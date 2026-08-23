package helium_test

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/internal/iofs"
	"github.com/stretchr/testify/require"
)

// finiteFile is an fs.File that yields exactly n bytes of 'A' and then io.EOF.
// Unlike an unbounded reader, it cannot hang or OOM if the size guard ever
// regresses: a finite (cap+1) source still trips the cap deterministically. It
// records whether Close was called.
type finiteFile struct {
	remaining int64
	closed    *atomic.Bool
}

func (f *finiteFile) Stat() (fs.FileInfo, error) { return nil, fs.ErrInvalid }

func (f *finiteFile) Read(p []byte) (int, error) {
	if f.remaining <= 0 {
		return 0, io.EOF
	}
	n := min(int64(len(p)), f.remaining)
	for i := range n {
		p[i] = 'A'
	}
	f.remaining -= n
	return int(n), nil
}

func (f *finiteFile) Close() error {
	f.closed.Store(true)
	return nil
}

// finiteFS hands out a single finiteFile of the configured size, recording
// closure.
type finiteFS struct {
	size   int64
	closed *atomic.Bool
}

func (s finiteFS) Open(string) (fs.File, error) {
	return &finiteFile{remaining: s.size, closed: s.closed}, nil
}

// readCloserFile wraps a string and records Close, used to verify external
// entity inputs are closed even on the success path.
type readCloserFile struct {
	r      *io.SectionReader
	closed *atomic.Bool
}

func (f *readCloserFile) Stat() (fs.FileInfo, error) { return nil, fs.ErrInvalid }

func (f *readCloserFile) Read(p []byte) (int, error) { return f.r.Read(p) }

func (f *readCloserFile) Close() error {
	f.closed.Store(true)
	return nil
}

type closingFS struct {
	data   string
	closed *atomic.Bool
}

func (s closingFS) Open(string) (fs.File, error) {
	return &readCloserFile{
		r:      io.NewSectionReader(strings.NewReader(s.data), 0, int64(len(s.data))),
		closed: s.closed,
	}, nil
}

// orderingFS resolves two external entities, "ext" and "ext2", and records the
// closed state of "ext" at the moment "ext2" is opened. "ext"'s buffered content
// references "ext2" (&y;), so "ext2" is opened only while "ext"'s already-read
// content is being parsed. If the parser closes "ext" promptly — at the read
// boundary, before parsing the buffered content — then "ext" is already closed
// by the time "ext2" is opened. If it deferred the close to function return,
// "ext" would still be open here.
type orderingFS struct {
	extClosed           *atomic.Bool
	extClosedAtNestOpen *atomic.Bool
	nestOpened          *atomic.Bool
}

func (s orderingFS) Open(name string) (fs.File, error) {
	if name == "ext2" {
		// "ext2" is opened only during the parse of "ext"'s buffered content.
		// Capture whether "ext" was already closed at this instant.
		s.nestOpened.Store(true)
		s.extClosedAtNestOpen.Store(s.extClosed.Load())
		const data = "<inner/>"
		return &readCloserFile{
			r:      io.NewSectionReader(strings.NewReader(data), 0, int64(len(data))),
			closed: &atomic.Bool{},
		}, nil
	}
	// "ext": its buffered content references the nested external entity &y;.
	const data = "<e>&y;</e>"
	return &readCloserFile{
		r:      io.NewSectionReader(strings.NewReader(data), 0, int64(len(data))),
		closed: s.extClosed,
	}, nil
}

// peCatalog resolves the external PE's system ID to a DIFFERENT URI than the
// entity's declared one, modeling the catalog/custom-resolver case where
// input.URI() (the URI actually opened) differs from the entity's URI().
type peCatalog struct{ from, to string }

func (c peCatalog) Resolve(_ context.Context, _, sysID string) string {
	if sysID == c.from {
		return c.to
	}
	return ""
}

func (c peCatalog) ResolveURI(_ context.Context, _ string) string { return "" }

// trimSlashFS adapts an fs.FS so a leading-slash absolute name (such as the
// "/C:/..." path FileURIToPath yields on a POSIX host) is accepted by an
// fs.ValidPath-enforcing FS like fstest.MapFS.
type trimSlashFS struct{ inner fs.FS }

func (f trimSlashFS) Open(name string) (fs.File, error) {
	// Normalize to forward slashes (the native open name is backslash-separated
	// on Windows: "C:\\win\\dir\\ext.dtd") and drop the leading slash the POSIX
	// "/C:/..." form carries, so fstest.MapFS (keyed "C:/win/dir/ext.dtd")
	// serves it on every OS.
	return f.inner.Open(strings.TrimPrefix(filepath.ToSlash(name), "/")) //nolint:wrapcheck // test helper
}

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

func TestExternalGeneralEntity(t *testing.T) {
	t.Parallel()

	t.Run("input is closed", func(t *testing.T) {
		const input = `<?xml version="1.0"?>
<!DOCTYPE r [<!ENTITY x SYSTEM "ext">]><r>&x;</r>`

		var closed atomic.Bool
		p := helium.NewParser().BlockXXE(false).SubstituteEntities(true).FS(closingFS{data: "<e>ok</e>", closed: &closed})
		_, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err)
		require.True(t, closed.Load(), "resolved external entity input must be closed on success")
	})

	// the resolved external input
	// is closed at the read boundary — BEFORE its already-buffered content is parsed
	// — not merely before Parse returns. The first entity's content references a
	// second external entity, so the second entity's Open happens mid-parse of the
	// first entity's buffered bytes; at that point the first input must already be
	// closed.
	t.Run("input is closed before the content is parsed", func(t *testing.T) {
		const input = `<?xml version="1.0"?>
<!DOCTYPE r [
  <!ENTITY x SYSTEM "ext">
  <!ENTITY y SYSTEM "ext2">
]><r>&x;</r>`

		var extClosed, extClosedAtNestOpen, nestOpened atomic.Bool
		p := helium.NewParser().BlockXXE(false).SubstituteEntities(true).FS(orderingFS{
			extClosed:           &extClosed,
			extClosedAtNestOpen: &extClosedAtNestOpen,
			nestOpened:          &nestOpened,
		})
		_, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err)
		require.True(t, nestOpened.Load(),
			"nested external entity must be opened while the first entity's content is parsed")
		require.True(t, extClosedAtNestOpen.Load(),
			"the first external input must be closed BEFORE its buffered content is parsed")
	})

	// the WFC: Entity Declared
	// (XML §4.1) as constrained by the Standalone Document Declaration (§2.9): in a
	// standalone="yes" document a reference to a general entity declared ONLY in the
	// external subset is a fatal well-formedness error, because a standalone="yes"
	// document asserts that no external markup declarations affect its content. The
	// same reference in a standalone="no" document, or a reference to an
	// internally-declared entity in a standalone="yes" document, is well-formed and
	// must still parse. This mirrors libxml2's xmlSAX2GetEntity XML_ERR_NOT_STANDALONE
	// check and closes W3C not-wf cases ibm-not-wf-P32-ibm32n09, ibm-not-wf-P68-ibm68n06
	// and not-wf-sa03.
	t.Run("a standalone document rejects a reference", func(t *testing.T) {
		const extDTD = "ext.dtd"

		newParser := func(fsys fstest.MapFS) helium.Parser {
			return helium.NewParser().
				BlockXXE(false).
				LoadExternalDTD(true).
				DefaultDTDAttributes(true).
				SubstituteEntities(true).
				FS(fsys)
		}

		// The external subset declares a general entity referenced by the document.
		extEntityFS := fstest.MapFS{
			extDTD: &fstest.MapFile{Data: []byte(`<!ENTITY ext "from external subset">` + "\n")},
		}

		t.Run("rejected: standalone yes, entity declared only externally, in content", func(t *testing.T) {
			t.Parallel()

			// ibm-not-wf-P32-ibm32n09 shape: reference in element content.
			doc := `<?xml version="1.0" standalone="yes"?>` + "\n" +
				`<!DOCTYPE root SYSTEM "` + extDTD + `" [` + "\n" +
				`<!ELEMENT root (#PCDATA)>` + "\n" +
				`]>` + "\n" +
				`<root>&ext;</root>`
			_, err := newParser(extEntityFS).Parse(t.Context(), []byte(doc))
			require.Error(t, err, "standalone=yes referencing an externally-declared entity must be a WF error")
			require.ErrorIs(t, err, helium.ErrNotStandalone)
		})

		t.Run("rejected: standalone yes, entity declared only externally, in attribute value", func(t *testing.T) {
			t.Parallel()

			// ibm-not-wf-P68-ibm68n06 / not-wf-sa03 shape: reference in an
			// attribute value.
			doc := `<?xml version="1.0" standalone="yes"?>` + "\n" +
				`<!DOCTYPE root SYSTEM "` + extDTD + `" [` + "\n" +
				`<!ELEMENT root (#PCDATA)>` + "\n" +
				`<!ATTLIST root att CDATA #IMPLIED>` + "\n" +
				`]>` + "\n" +
				`<root att="x-&ext;-y">content</root>`
			_, err := newParser(extEntityFS).Parse(t.Context(), []byte(doc))
			require.Error(t, err, "standalone=yes referencing an externally-declared entity in an attribute value must be a WF error")
			require.ErrorIs(t, err, helium.ErrNotStandalone)
		})

		t.Run("accepted: standalone no, entity declared externally", func(t *testing.T) {
			t.Parallel()

			// The identical document with standalone="no" is well-formed: external
			// declarations are permitted to supply entities.
			doc := `<?xml version="1.0" standalone="no"?>` + "\n" +
				`<!DOCTYPE root SYSTEM "` + extDTD + `" [` + "\n" +
				`<!ELEMENT root (#PCDATA)>` + "\n" +
				`]>` + "\n" +
				`<root>&ext;</root>`
			parsed, err := newParser(extEntityFS).Parse(t.Context(), []byte(doc))
			require.NoError(t, err, "standalone=no referencing an externally-declared entity must still parse")
			require.Equal(t, "from external subset", string(parsed.DocumentElement().Content()))
		})

		t.Run("accepted: standalone yes, entity declared internally", func(t *testing.T) {
			t.Parallel()

			// A standalone="yes" document referencing an entity declared in the
			// INTERNAL subset is well-formed even when an external subset is present.
			doc := `<?xml version="1.0" standalone="yes"?>` + "\n" +
				`<!DOCTYPE root SYSTEM "` + extDTD + `" [` + "\n" +
				`<!ELEMENT root (#PCDATA)>` + "\n" +
				`<!ENTITY ext "from internal subset">` + "\n" +
				`]>` + "\n" +
				`<root>&ext;</root>`
			parsed, err := newParser(extEntityFS).Parse(t.Context(), []byte(doc))
			require.NoError(t, err, "an internally-declared entity must resolve under standalone=yes")
			require.Equal(t, "from internal subset", string(parsed.DocumentElement().Content()))
		})
	})
}

func TestExternalGeneralEntityTextDecl(t *testing.T) {
	t.Parallel()

	// A version-bearing TextDecl at the start of an external general entity is
	// equally valid and must also be accepted.
	t.Run("with a version", func(t *testing.T) {
		parsed, err := parseExtEntity(t, []byte(`<?xml version="1.0" encoding="UTF-8"?><child>hi</child>`))
		require.NoError(t, err)
		require.NotNil(t, parsed)
		require.Equal(t, "hi", string(parsed.DocumentElement().FirstChild().Content()))
	})

	// An external parsed general entity's replacement text may begin with a TextDecl
	// — '<?xml' VersionInfo? EncodingDecl S? '?>' — where VersionInfo is OPTIONAL,
	// EncodingDecl REQUIRED, and no StandaloneDecl is permitted. A version-less
	// TextDecl must be consumed and its body parsed as the replacement text, not
	// rejected for a missing version pseudo-attribute.
	t.Run("without a version", func(t *testing.T) {
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
	})

	// A version-only TextDecl (no EncodingDecl, which is required in a TextDecl) must
	// be rejected.
	t.Run("missing encoding is rejected", func(t *testing.T) {
		_, err := parseExtEntity(t, []byte(`<?xml version="1.0"?><child>hi</child>`))
		require.Error(t, err, "a TextDecl missing the required encoding declaration must be rejected")
	})

	// A TextDecl carrying a StandaloneDecl is forbidden by the grammar (a TextDecl
	// permits no 'standalone' pseudo-attribute) and must be rejected.
	t.Run("standalone is rejected", func(t *testing.T) {
		_, err := parseExtEntity(t, []byte(`<?xml encoding="UTF-8" standalone="yes"?><child>hi</child>`))
		require.Error(t, err, "a standalone pseudo-attribute in an external-entity TextDecl must be rejected")
	})

	// The XML §4.3.4 version-compatibility matrix: an external parsed entity's
	// TextDecl version must not be LATER than the referencing document's. A 1.0
	// document may not reference a 1.1 entity (fatal, W3C rmt-e2e-38); every other
	// combination is accepted. The version comparison is against the ACTUAL document
	// version, so a 1.1 document referencing a 1.1 (or 1.0) entity must be accepted —
	// the TextDecl is decoded on a doc-less sub-context that is seeded with the
	// parent document's version.
	t.Run("version matrix", func(t *testing.T) {
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
	})
}

func TestExternalParameterEntity(t *testing.T) {
	t.Parallel()

	// referencing an external
	// SYSTEM parameter entity in the external subset actually loads its content and
	// applies the declarations it contains. The external PE pe.ent declares a general
	// entity; with external DTD loading enabled that entity must be registered,
	// proving the external PE content was pulled in and parsed (not silently dropped).
	t.Run("content is loaded", func(t *testing.T) {
		fsys := fstest.MapFS{
			dtdSystemID: {Data: []byte(
				`<!ENTITY ctrl "control">` + "\n" +
					`<!ENTITY % pe SYSTEM "pe.ent">` + "\n" +
					`%pe;`)},
			peSystemID: {Data: []byte(`<!ENTITY fromPE "loaded-from-external-pe">`)},
		}
		const input = `<?xml version="1.0"?>` + "\n" +
			`<!DOCTYPE r SYSTEM "d.dtd"><r/>`

		doc, err := helium.NewParser().BlockXXE(false).
			LoadExternalDTD(true).
			FS(fsys).
			Parse(t.Context(), []byte(input))
		require.NoError(t, err)
		require.NotNil(t, doc)

		_, ctrlOK := doc.GetEntity("ctrl")
		require.True(t, ctrlOK, "control general entity must be stored, proving the external subset loaded")

		ent, ok := doc.GetEntity("fromPE")
		require.True(t, ok, "the general entity declared inside the external PE must be registered")
		require.Equal(t, "loaded-from-external-pe", string(ent.Content()))
	})

	// the secure default
	// (XXE blocked) loads no external parameter entity content: with the default
	// parser the external subset is not loaded at all, so the general entity declared
	// inside the external PE is absent. Behavior is unchanged from before the fix.
	t.Run("not loaded under the secure default", func(t *testing.T) {
		fsys := fstest.MapFS{
			dtdSystemID: {Data: []byte(
				`<!ENTITY ctrl "control">` + "\n" +
					`<!ENTITY % pe SYSTEM "pe.ent">` + "\n" +
					`%pe;`)},
			peSystemID: {Data: []byte(`<!ENTITY fromPE "loaded-from-external-pe">`)},
		}
		const input = `<?xml version="1.0"?>` + "\n" +
			`<!DOCTYPE r SYSTEM "d.dtd"><r/>`

		doc, err := helium.NewParser().FS(fsys).Parse(t.Context(), []byte(input))
		require.NoError(t, err)
		require.NotNil(t, doc)

		_, ok := doc.GetEntity("fromPE")
		require.False(t, ok, "secure default must not load external parameter entity content")
	})

	// an external
	// parameter entity referenced inside a general entity's VALUE is loaded and
	// expanded regardless of whether the PE was ever referenced at the top level of
	// the DTD first. The external subset declares "%p;" (SYSTEM "value.ent") and a
	// general entity g whose value is "%p;" — WITHOUT any top-level "%p;" reference
	// that would otherwise be what first caches p's content. g's expanded value must
	// equal value.ent's content, proving the load happens through the centralized
	// PE-replacement path, independent of reference order.
	//
	// The leading control general entity sidesteps the same separate pre-existing
	// first-declaration bug noted on TestExternalParameterEntityNestedRelativeBaseURI.
	t.Run("in an entity value, loaded", func(t *testing.T) {
		fsys := fstest.MapFS{
			dtdSystemID: {Data: []byte(
				`<!ENTITY ctrl "control">` + "\n" +
					`<!ENTITY % p SYSTEM "value.ent">` + "\n" +
					`<!ENTITY g "%p;">`)},
			"value.ent": {Data: []byte(`expanded-from-external-pe`)},
		}
		const input = `<?xml version="1.0"?>` + "\n" +
			`<!DOCTYPE r SYSTEM "d.dtd"><r/>`

		doc, err := helium.NewParser().BlockXXE(false).
			LoadExternalDTD(true).
			FS(fsys).
			Parse(t.Context(), []byte(input))
		require.NoError(t, err)
		require.NotNil(t, doc)

		ent, ok := doc.GetEntity("g")
		require.True(t, ok, "general entity g must be registered")
		require.Equal(t, "expanded-from-external-pe", string(ent.Content()),
			"external PE referenced in an entity value must be loaded and expanded regardless of reference order")
	})

	// the secure
	// default does NOT load an external PE referenced inside an entity value: with
	// the default parser the external subset is not loaded at all, so g is absent.
	t.Run("in an entity value, not loaded under the secure default", func(t *testing.T) {
		fsys := fstest.MapFS{
			dtdSystemID: {Data: []byte(
				`<!ENTITY ctrl "control">` + "\n" +
					`<!ENTITY % p SYSTEM "value.ent">` + "\n" +
					`<!ENTITY g "%p;">`)},
			"value.ent": {Data: []byte(`expanded-from-external-pe`)},
		}
		const input = `<?xml version="1.0"?>` + "\n" +
			`<!DOCTYPE r SYSTEM "d.dtd"><r/>`

		doc, err := helium.NewParser().FS(fsys).Parse(t.Context(), []byte(input))
		require.NoError(t, err)
		require.NotNil(t, doc)

		_, ok := doc.GetEntity("g")
		require.False(t, ok, "secure default must not load the external subset or its PE-referencing entity value")
	})

	// when an
	// external parameter entity whose replacement text begins with a TextDecl is
	// referenced inside a GENERAL entity's value ("<!ENTITY g "%p;">"), the stored
	// value of g is the PE's POST-TextDecl bytes only — the leading
	// "<?xml ... encoding=...?>" must NOT be embedded into g. The decode is
	// centralized at the shared load/cache chokepoint, so the entity-value
	// expansion path and the top-level "%pe;" path both see the stripped bytes.
	t.Run("in an entity value, the text declaration is stripped", func(t *testing.T) {
		fsys := fstest.MapFS{
			dtdSystemID: {Data: []byte(
				`<!ENTITY ctrl "control">` + "\n" +
					`<!ENTITY % p SYSTEM "value.ent">` + "\n" +
					`<!ENTITY g "%p;">`)},
			"value.ent": {Data: []byte(`<?xml version="1.0" encoding="UTF-8"?>` +
				`post-textdecl-value`)},
		}
		const input = `<?xml version="1.0"?>` + "\n" +
			`<!DOCTYPE r SYSTEM "d.dtd"><r/>`

		doc, err := helium.NewParser().BlockXXE(false).
			LoadExternalDTD(true).
			FS(fsys).
			Parse(t.Context(), []byte(input))
		require.NoError(t, err)
		require.NotNil(t, doc)

		ent, ok := doc.GetEntity("g")
		require.True(t, ok, "general entity g must be registered")
		require.Equal(t, "post-textdecl-value", string(ent.Content()),
			"external PE referenced in an entity value must contribute only its post-TextDecl bytes, not the TextDecl itself")
	})

	// a self-recursive
	// external parameter entity is rejected with a recursion error, well before
	// cursors pile up to the entity-amplification ceiling (or OOM): pe.ent's
	// replacement text references the very PE that loaded it, so the active-PE guard
	// must fail the parse the moment the nested "%pe;" is seen.
	t.Run("self recursion is rejected", func(t *testing.T) {
		fsys := fstest.MapFS{
			dtdSystemID: {Data: []byte(
				`<!ENTITY ctrl "control">` + "\n" +
					`<!ENTITY % pe SYSTEM "pe.ent">` + "\n" +
					`%pe;`)},
			peSystemID: {Data: []byte(`%pe;`)},
		}
		const input = `<?xml version="1.0"?>` + "\n" +
			`<!DOCTYPE r SYSTEM "d.dtd"><r/>`

		_, err := helium.NewParser().BlockXXE(false).
			LoadExternalDTD(true).
			FS(fsys).
			Parse(t.Context(), []byte(input))
		require.Error(t, err, "self-recursive external parameter entity must be rejected")
		require.Contains(t, err.Error(), "references itself",
			"rejection must be the recursion guard, not an amplification/ceiling trip")
	})

	// an external PE
	// used only inside another entity value. That path caches the decoded body rather
	// than pushing it through the DTD declaration loop, so raw literals must be
	// checked before caching under the PE's TextDecl version.
	t.Run("entity-value literal validation", func(t *testing.T) {
		for _, tc := range []struct {
			name       string
			textDecl   string
			body       string
			wantErr    bool
			wantEntity string
		}{
			{
				name:     "XML 1.1 rejects raw restricted character",
				textDecl: `<?xml version="1.1" encoding="UTF-8"?>`,
				body:     "\x7f",
				wantErr:  true,
			},
			{
				name:       "XML 1.1 accepts character reference",
				textDecl:   `<?xml version="1.1" encoding="UTF-8"?>`,
				body:       "&#127;",
				wantEntity: "\n\x7f",
			},
			{
				name:       "XML 1.0 accepts raw C1 character",
				textDecl:   `<?xml version="1.0" encoding="UTF-8"?>`,
				body:       "\x7f",
				wantEntity: "\n\x7f",
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				fsys := fstest.MapFS{
					dtdSystemID: {Data: []byte(
						`<!ENTITY ctrl "control">` + "\n" +
							`<!ENTITY % pe SYSTEM "pe.ent">` + "\n" +
							`<!ENTITY e "%pe;">`)},
					peSystemID: {Data: []byte(tc.textDecl + "\n" + tc.body)},
				}
				const input = `<?xml version="1.1"?>` + "\n" +
					`<!DOCTYPE r SYSTEM "d.dtd"><r/>`

				doc, err := helium.NewParser().BlockXXE(false).
					LoadExternalDTD(true).
					FS(fsys).
					Parse(t.Context(), []byte(input))
				if tc.wantErr {
					require.ErrorIs(t, err, helium.ErrInvalidChar)
					return
				}
				require.NoError(t, err)
				ent, ok := doc.GetEntity("e")
				require.True(t, ok, "general entity using the external PE must be registered")
				require.Equal(t, tc.wantEntity, string(ent.Content()))
			})
		}
	})

	// the PUBLIC external parameter
	// entity path records the public and system IDs in the correct fields, with no
	// swap.
	t.Run("a PUBLIC declaration is captured", func(t *testing.T) {
		fsys := fstest.MapFS{
			dtdSystemID: {Data: []byte(
				`<!ENTITY ctrl "control">` + "\n" +
					`<!ENTITY % pe PUBLIC "-//x//pe" "pe.ent">`)},
		}
		const input = `<?xml version="1.0"?>` + "\n" +
			`<!DOCTYPE r SYSTEM "d.dtd"><r/>`

		doc, err := helium.NewParser().BlockXXE(false).
			LoadExternalDTD(true).
			FS(fsys).
			Parse(t.Context(), []byte(input))
		require.NoError(t, err)
		require.NotNil(t, doc)

		ent, ok := doc.GetParameterEntity("pe")
		require.True(t, ok, "external PUBLIC parameter entity must be registered")
		require.Equal(t, enum.ExternalParameterEntity, ent.EntityType())
		require.Equal(t, peSystemID, ent.SystemID(), "the system literal must be the system ID")
		require.Equal(t, "-//x//pe", ent.ExternalID(), "the public ID must be the external ID")
	})

	// an external SYSTEM
	// parameter entity declared in the external subset is registered. parseExternalID
	// returns (systemURI, publicID); the external-PE declaration path must guard on
	// the systemURI and record it. Guarding on the publicID instead drops a SYSTEM PE
	// entirely (a SYSTEM declaration has no public ID), so it would never be stored.
	//
	// A control general entity declared on the line before proves the external subset
	// is loaded; only the SYSTEM external PE was being dropped.
	t.Run("a SYSTEM declaration is captured", func(t *testing.T) {
		fsys := fstest.MapFS{
			dtdSystemID: {Data: []byte(
				`<!ENTITY ctrl "control">` + "\n" +
					`<!ENTITY % pe SYSTEM "pe.ent">`)},
		}
		const input = `<?xml version="1.0"?>` + "\n" +
			`<!DOCTYPE r SYSTEM "d.dtd"><r/>`

		doc, err := helium.NewParser().BlockXXE(false).
			LoadExternalDTD(true).
			FS(fsys).
			Parse(t.Context(), []byte(input))
		require.NoError(t, err)
		require.NotNil(t, doc)

		_, ctrlOK := doc.GetEntity("ctrl")
		require.True(t, ctrlOK, "control general entity must be stored, proving the external subset loaded")

		ent, ok := doc.GetParameterEntity("pe")
		require.True(t, ok, "external SYSTEM parameter entity must be registered")
		require.Equal(t, enum.ExternalParameterEntity, ent.EntityType())
		require.Equal(t, peSystemID, ent.SystemID(), "the SYSTEM literal must be recorded as the system ID")
		require.Empty(t, ent.ExternalID(), "a SYSTEM declaration has no public ID")
	})
}

func TestExternalParameterEntityTextDecl(t *testing.T) {
	t.Parallel()

	// an external parameter
	// entity whose replacement text begins with a TextDecl
	// ("<?xml ... encoding=...?>") is parsed: the TextDecl is consumed before the
	// declaration loop instead of being rejected as a processing instruction.
	t.Run("with a text declaration", func(t *testing.T) {
		fsys := fstest.MapFS{
			dtdSystemID: {Data: []byte(
				`<!ENTITY ctrl "control">` + "\n" +
					`<!ENTITY % pe SYSTEM "pe.ent">` + "\n" +
					`%pe;`)},
			peSystemID: {Data: []byte(`<?xml version="1.0" encoding="UTF-8"?>` + "\n" +
				`<!ENTITY td "from-textdecl-pe">`)},
		}
		const input = `<?xml version="1.0"?>` + "\n" +
			`<!DOCTYPE r SYSTEM "d.dtd"><r/>`

		doc, err := helium.NewParser().BlockXXE(false).
			LoadExternalDTD(true).
			FS(fsys).
			Parse(t.Context(), []byte(input))
		require.NoError(t, err)
		require.NotNil(t, doc)

		ent, ok := doc.GetEntity("td")
		require.True(t, ok, "entity declared after a TextDecl in an external PE must be registered")
		require.Equal(t, "from-textdecl-pe", string(ent.Content()),
			"external PE beginning with a TextDecl must parse its declarations")
	})

	// an external
	// parameter entity whose replacement text begins with a declaration carrying a
	// 'standalone' pseudo-attribute is rejected: a TextDecl does not permit a
	// StandaloneDecl, so such a declaration is malformed and must not be leniently
	// accepted.
	t.Run("standalone is rejected", func(t *testing.T) {
		fsys := fstest.MapFS{
			dtdSystemID: {Data: []byte(
				`<!ENTITY ctrl "control">` + "\n" +
					`<!ENTITY % pe SYSTEM "pe.ent">` + "\n" +
					`%pe;`)},
			peSystemID: {Data: []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n" +
				`<!ENTITY td "x">`)},
		}
		const input = `<?xml version="1.0"?>` + "\n" +
			`<!DOCTYPE r SYSTEM "d.dtd"><r/>`

		_, err := helium.NewParser().BlockXXE(false).
			LoadExternalDTD(true).
			FS(fsys).
			Parse(t.Context(), []byte(input))
		require.Error(t, err,
			"a TextDecl carrying a standalone pseudo-attribute is malformed and must be rejected")
	})

	// A parameter entity's TextDecl version applies while that entity input is on
	// the DTD stack, then the surrounding external DTD's version is restored. This
	// keeps a TextDecl 1.0 PE under an XML 1.1 document from accepting XML 1.1-only
	// character references, without leaking that stricter rule to declarations after
	// it.
	t.Run("the version is scoped to the entity", func(t *testing.T) {
		t.Run("XML 1.0 PE retains XML 1.0 character-reference rules", func(t *testing.T) {
			t.Parallel()

			fsys := fstest.MapFS{
				dtdSystemID: {Data: []byte(
					`<!ENTITY ctrl "control">` + "\n" +
						`<!ENTITY % pe SYSTEM "pe.ent">` + "\n" +
						`%pe;`)},
				peSystemID: {Data: []byte(`<?xml version="1.0" encoding="UTF-8"?>` + "\n" +
					`<!ENTITY fromPE "&#1;">`)},
			}
			const input = `<?xml version="1.1"?>` + "\n" +
				`<!DOCTYPE r SYSTEM "d.dtd"><r/>`

			_, err := helium.NewParser().BlockXXE(false).
				LoadExternalDTD(true).
				FS(fsys).
				Parse(t.Context(), []byte(input))
			require.Error(t, err)
			require.Contains(t, err.Error(), "invalid XML char value 1")
		})

		t.Run("parent XML version is restored", func(t *testing.T) {
			t.Parallel()

			fsys := fstest.MapFS{
				dtdSystemID: {Data: []byte(
					`<!ENTITY ctrl "control">` + "\n" +
						`<!ENTITY % pe SYSTEM "pe.ent">` + "\n" +
						`%pe;` + "\n" +
						`<!ENTITY afterPE "&#1;">`)},
				peSystemID: {Data: []byte(`<?xml version="1.0" encoding="UTF-8"?>` + "\n" +
					`<!ENTITY fromPE "loaded">`)},
			}
			const input = `<?xml version="1.1"?>` + "\n" +
				`<!DOCTYPE r SYSTEM "d.dtd"><r/>`

			parsed, err := helium.NewParser().BlockXXE(false).
				LoadExternalDTD(true).
				FS(fsys).
				Parse(t.Context(), []byte(input))
			require.NoError(t, err)
			require.Equal(t, "1.1", parsed.Version())
			_, ok := parsed.GetEntity("fromPE")
			require.True(t, ok)
			_, ok = parsed.GetEntity("afterPE")
			require.True(t, ok)
		})
	})

	// an external
	// parameter entity whose replacement text begins with a version-only
	// declaration ("<?xml version="1.0"?>") is rejected: a TextDecl REQUIRES an
	// EncodingDecl, so a version-only declaration is not a valid TextDecl and must
	// not be leniently accepted.
	t.Run("a version-only declaration is rejected", func(t *testing.T) {
		fsys := fstest.MapFS{
			dtdSystemID: {Data: []byte(
				`<!ENTITY ctrl "control">` + "\n" +
					`<!ENTITY % pe SYSTEM "pe.ent">` + "\n" +
					`%pe;`)},
			peSystemID: {Data: []byte(`<?xml version="1.0"?>` + "\n" +
				`<!ENTITY td "x">`)},
		}
		const input = `<?xml version="1.0"?>` + "\n" +
			`<!DOCTYPE r SYSTEM "d.dtd"><r/>`

		_, err := helium.NewParser().BlockXXE(false).
			LoadExternalDTD(true).
			FS(fsys).
			Parse(t.Context(), []byte(input))
		require.Error(t, err,
			"a version-only declaration is not a valid TextDecl (encoding is required) and must be rejected")
	})
}

func TestExternalEntityBaseURI(t *testing.T) {
	t.Parallel()

	// a relative system
	// ID in a declaration INSIDE an external parameter entity resolves against the
	// PE's OWN location, not the containing DTD. d.dtd declares a PE at "sub/pe.ent"
	// and references it; sub/pe.ent declares a general entity "e" with a relative
	// SYSTEM id "leaf.ent". With baseURI scoped to the PE while its replacement text
	// is parsed, e must resolve to "sub/leaf.ent" (sibling of pe.ent), NOT "leaf.ent"
	// (sibling of d.dtd).
	//
	// The leading control general entity sidesteps a SEPARATE pre-existing parser
	// bug (present on main, orthogonal to base-URI scoping): a parameter-entity
	// declaration as the VERY FIRST declaration of an external subset fails with
	// "space required". The control entity keeps this regression focused on scoping.
	t.Run("a nested relative base URI", func(t *testing.T) {
		fsys := fstest.MapFS{
			dtdSystemID: {Data: []byte(
				`<!ENTITY ctrl "control">` + "\n" +
					`<!ENTITY % pe SYSTEM "sub/pe.ent">` + "\n" +
					`%pe;`)},
			"sub/pe.ent": {Data: []byte(`<!ENTITY e SYSTEM "leaf.ent">`)},
		}
		const input = `<?xml version="1.0"?>` + "\n" +
			`<!DOCTYPE r SYSTEM "d.dtd"><r/>`

		doc, err := helium.NewParser().BlockXXE(false).
			LoadExternalDTD(true).
			FS(fsys).
			Parse(t.Context(), []byte(input))
		require.NoError(t, err)
		require.NotNil(t, doc)

		ent, ok := doc.GetEntity("e")
		require.True(t, ok, "general entity declared inside the external PE must be registered")
		require.Equal(t, "sub/leaf.ent", ent.URI(),
			"relative system ID inside the external PE must resolve against the PE's own location")
	})

	// when an
	// external PE is loaded FIRST through a general entity's value (caching its
	// content), a later top-level "%pe;" parses the cached bytes against the URI the
	// bytes were ACTUALLY loaded from — the catalog-resolved "sub/pe.ent" — not the
	// entity's declared "pe.ent". A relative SYSTEM id inside the PE ("leaf.ent")
	// must therefore resolve to "sub/leaf.ent". Before the resolved-URI cache fix the
	// cached path returned e.URI() ("pe.ent"), wrongly resolving leaf to "leaf.ent".
	t.Run("the first resolved URI is cached", func(t *testing.T) {
		fsys := fstest.MapFS{
			dtdSystemID: {Data: []byte(
				`<!ENTITY ctrl "control">` + "\n" +
					`<!ENTITY % pe SYSTEM "pe.ent">` + "\n" +
					`<!ENTITY g "%pe;">` + "\n" +
					`%pe;`)},
			"sub/pe.ent": {Data: []byte(`<!ENTITY leaf SYSTEM "leaf.ent">`)},
		}
		const input = `<?xml version="1.0"?>` + "\n" +
			`<!DOCTYPE r SYSTEM "d.dtd"><r/>`

		doc, err := helium.NewParser().BlockXXE(false).
			LoadExternalDTD(true).
			Catalog(peCatalog{from: peSystemID, to: "sub/pe.ent"}).
			FS(fsys).
			Parse(t.Context(), []byte(input))
		require.NoError(t, err)
		require.NotNil(t, doc)

		ent, ok := doc.GetEntity("leaf")
		require.True(t, ok, "general entity declared inside the cached external PE must be registered")
		require.Equal(t, "sub/leaf.ent", ent.URI(),
			"cached external PE must resolve relative IDs against the URI it was actually loaded from")
	})

	// is the string-shaped
	// (GOOS-independent) regression for the Windows nested-external-DTD failure: a
	// document parsed with a Windows-drive "file:" base URI
	// ("file:///C:/win/dir/doc.xml") that declares a RELATIVE external DTD
	// ("ext.dtd"). The resolver must combine them into a proper "file:" URI (via
	// BuildURI) and convert it to a local path before Open, NOT mangle it with
	// filepath.Dir/Join — on Windows that cleared the directory and dropped the DTD.
	// The base is a plain string, so this exercises the Windows branch on every OS.
	// The resolved open name is whatever FileURIToPath yields for the combined
	// "file:///C:/win/dir/ext.dtd": "/C:/win/dir/ext.dtd" on a POSIX host,
	// "C:\\win\\dir\\ext.dtd" on Windows. Derive it the same way so the assertion
	// is correct on both, and let trimSlashFS normalize either form to the MapFS key.
	t.Run("a Windows drive file URI base", func(t *testing.T) {
		const dtd = `<!ELEMENT chapter (#PCDATA)>
<!ENTITY greet "hello from nested dtd">`

		openName, err := iofs.FileURIToPath("file:///C:/win/dir/ext.dtd")
		require.NoError(t, err)
		fsys := &recordingFS{inner: trimSlashFS{fstest.MapFS{"C:/win/dir/ext.dtd": {Data: []byte(dtd)}}}}

		xml := `<?xml version="1.0"?>` +
			`<!DOCTYPE chapter SYSTEM "ext.dtd">` +
			`<chapter>text</chapter>`

		doc, err := helium.NewParser().BlockXXE(false).
			LoadExternalDTD(true).
			BaseURI("file:///C:/win/dir/doc.xml").
			FS(fsys).
			Parse(t.Context(), []byte(xml))
		require.NoError(t, err)

		// The relative SYSTEM id resolved into the base directory (never dropped to a
		// bare "ext.dtd", the Windows filepath.Join failure mode), so the DTD was
		// found and its general entity declared.
		require.True(t, fsys.wasOpened(openName),
			"relative SYSTEM id must resolve against the windows-drive file: base")
		_, found := doc.GetEntity("greet")
		require.True(t, found, "entity from external DTD must be declared, proving the file: DTD URI was resolved")
	})
}

func TestParseExternalEntity(t *testing.T) {
	t.Parallel()

	t.Run("external entities", func(t *testing.T) {
		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ENTITY ext SYSTEM "ext.xml">
]>
<doc>&ext;</doc>`

		// The external entity is declared in the internal subset and its content is
		// served through the configured FS, exercising the default resolution path.
		fsys := fstest.MapFS{
			"ext.xml": &fstest.MapFile{Data: []byte("<inner>hello</inner>")},
		}

		p := helium.NewParser().BlockXXE(false).SubstituteEntities(true).FS(fsys)
		doc, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err, "Parse with external entity should succeed")
		require.NotNil(t, doc, "external entity parse should produce a document")

		var buf bytes.Buffer
		require.NoError(t, helium.NewWriter().WriteTo(&buf, doc))
		out := buf.String()
		require.Contains(t, out, "<inner", "external entity element should be expanded into the document")
		require.Contains(t, out, ">hello</inner>", "external entity content should be expanded into the document")
	})

	t.Run("a malformed encoding", func(t *testing.T) {
		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ENTITY ext SYSTEM "ext.xml">
]>
<doc>&ext;</doc>`

		// External entity bytes: UTF-16BE BOM, then "<a>" and an unpaired high
		// surrogate (0xD800) before "</a>". The decoder would silently substitute
		// U+FFFD for the surrogate; the parser must instead treat it as fatal,
		// matching the document-level decode-error gate.
		utf16be := func(s string) []byte {
			b := make([]byte, 0, len(s)*2)
			for _, r := range s {
				b = append(b, byte(r>>8), byte(r))
			}
			return b
		}
		ent := []byte{0xFE, 0xFF} // BOM
		ent = append(ent, utf16be("<a>")...)
		ent = append(ent, 0xD8, 0x00) // unpaired high surrogate
		ent = append(ent, utf16be("</a>")...)

		fsys := fstest.MapFS{"ext.xml": &fstest.MapFile{Data: ent}}

		p := helium.NewParser().BlockXXE(false).SubstituteEntities(true).FS(fsys)
		_, err := p.Parse(t.Context(), []byte(input))
		require.Error(t, err, "malformed UTF-16 external entity must fail and insert no U+FFFD")
	})

	t.Run("a valid encoding", func(t *testing.T) {
		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ENTITY ext SYSTEM "ext.xml">
]>
<doc>&ext;</doc>`

		// A well-formed UTF-16BE external entity (BOM + "<a/>") must still load.
		utf16be := func(s string) []byte {
			b := make([]byte, 0, len(s)*2)
			for _, r := range s {
				b = append(b, byte(r>>8), byte(r))
			}
			return b
		}
		ent := append([]byte{0xFE, 0xFF}, utf16be("<a/>")...)

		fsys := fstest.MapFS{"ext.xml": &fstest.MapFile{Data: ent}}

		p := helium.NewParser().SubstituteEntities(true).FS(fsys)
		_, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err, "well-formed UTF-16 external entity must load")
	})
}
