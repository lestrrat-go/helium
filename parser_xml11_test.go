package helium_test

import (
	"testing"
	"testing/fstest"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

const (
	commentCase    = "comment"
	restrictedChar = "\x7f"
)

func TestXML11Characters(t *testing.T) {
	t.Parallel()

	t.Run("raw restricted characters", func(t *testing.T) {
		for _, tc := range []struct {
			name   string
			source func(string, string) string
		}{
			{
				name: commentCase,
				source: func(version, value string) string {
					return `<?xml version="` + version + `"?><!--value` + value + `--><root/>`
				},
			},
			{
				name: "processing instruction",
				source: func(version, value string) string {
					return `<?xml version="` + version + `"?><root><?pi value` + value + `?></root>`
				},
			},
			{
				name: "CDATA",
				source: func(version, value string) string {
					return `<?xml version="` + version + `"?><root><![CDATA[value` + value + `]]></root>`
				},
			},
			{
				name: "text",
				source: func(version, value string) string {
					return `<?xml version="` + version + `"?><root>value` + value + `</root>`
				},
			},
			{
				name: "attribute",
				source: func(version, value string) string {
					return `<?xml version="` + version + `"?><root attr="value` + value + `"/>`
				},
			},
			{
				name: "internal entity value",
				source: func(version, value string) string {
					return `<?xml version="` + version + `"?><!DOCTYPE root [<!ENTITY entity "value` + value + `">]><root/>`
				},
			},
			{
				name: "DOCTYPE system literal",
				source: func(version, value string) string {
					return `<?xml version="` + version + `"?><!DOCTYPE root SYSTEM "id` + value + `"><root/>`
				},
			},
			{
				name: "notation system literal",
				source: func(version, value string) string {
					return `<?xml version="` + version + `"?><!DOCTYPE root [<!NOTATION notation SYSTEM "id` + value + `">]><root/>`
				},
			},
		} {
			for _, version := range []struct {
				name    string
				value   string
				wantErr bool
			}{
				{name: "XML 1.0", value: ver10},
				{name: "XML 1.1", value: ver11, wantErr: true},
			} {
				t.Run(tc.name+" "+version.name, func(t *testing.T) {
					t.Parallel()

					_, err := helium.NewParser().Parse(t.Context(), []byte(tc.source(version.value, restrictedChar)))
					if version.wantErr {
						require.Error(t, err)
						return
					}
					require.NoError(t, err)
				})
			}
		}
	})

	t.Run("restricted character references", func(t *testing.T) {
		for _, tc := range []struct {
			name   string
			source string
			check  func(*testing.T, *helium.Document)
		}{
			{
				name:   "text",
				source: `<?xml version="` + ver11 + `"?><root>&#127;</root>`,
				check: func(t *testing.T, doc *helium.Document) {
					require.Equal(t, "\x7f", string(doc.DocumentElement().Content()))
				},
			},
			{
				name:   "attribute",
				source: `<?xml version="` + ver11 + `"?><root attr="&#127;"/>`,
				check: func(t *testing.T, doc *helium.Document) {
					value, ok := doc.DocumentElement().GetAttribute("attr")
					require.True(t, ok)
					require.Equal(t, "\x7f", value)
				},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				doc, err := helium.NewParser().Parse(t.Context(), []byte(tc.source))
				require.NoError(t, err)
				tc.check(t, doc)
			})
		}
	})

	t.Run("entity-value character references", func(t *testing.T) {
		for _, tc := range []struct {
			name    string
			version string
			wantErr bool
		}{
			{name: "XML 1.0", version: ver10, wantErr: true},
			{name: "XML 1.1", version: ver11},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				source := `<?xml version="` + tc.version + `"?><!DOCTYPE root [<!ENTITY e "&#1;">]><root/>`
				doc, err := helium.NewParser().Parse(t.Context(), []byte(source))
				if tc.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				entity, found := doc.GetEntity("e")
				require.True(t, found)
				require.NotNil(t, entity)
			})
		}
	})

	t.Run("external entity literal characters", func(t *testing.T) {
		for _, tc := range []struct {
			name       string
			docVersion string
			entity     string
			wantErr    bool
		}{
			{
				name:       "XML 1.0 document inherits XML 1.0",
				docVersion: ver10,
				entity:     `<child>` + restrictedChar + `</child>`,
			},
			{
				name:       "XML 1.1 document inherits XML 1.1",
				docVersion: ver11,
				entity:     `<child>` + restrictedChar + `</child>`,
				wantErr:    true,
			},
			{
				name:       "XML 1.1 document honors XML 1.0 TextDecl",
				docVersion: ver11,
				entity:     `<?xml version="1.0" encoding="UTF-8"?><child>` + restrictedChar + `</child>`,
			},
			{
				name:       "XML 1.1 document honors XML 1.1 TextDecl",
				docVersion: ver11,
				entity:     `<?xml version="1.1" encoding="UTF-8"?><child>` + restrictedChar + `</child>`,
				wantErr:    true,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				source := `<?xml version="` + tc.docVersion + `"?><!DOCTYPE root [<!ENTITY e SYSTEM "entity.ent">]><root>&e;</root>`
				fsys := fstest.MapFS{
					"entity.ent": &fstest.MapFile{Data: []byte(tc.entity)},
				}
				doc, err := helium.NewParser().
					BlockXXE(false).
					SubstituteEntities(true).
					FS(fsys).
					Parse(t.Context(), []byte(source))
				if tc.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				require.Equal(t, restrictedChar, string(doc.DocumentElement().FirstChild().Content()))
			})
		}
	})

	t.Run("external DTD IGNORE literal characters", func(t *testing.T) {
		for _, tc := range []struct {
			name    string
			version string
			wantErr bool
		}{
			{name: "XML 1.0", version: ver10},
			{name: "XML 1.1", version: ver11, wantErr: true},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				source := `<?xml version="` + tc.version + `"?><!DOCTYPE root SYSTEM "ignore.dtd"><root/>`
				fsys := fstest.MapFS{
					"ignore.dtd": &fstest.MapFile{Data: []byte(`<![IGNORE[` + restrictedChar + `]]>`)},
				}
				_, err := helium.NewParser().
					BlockXXE(false).
					LoadExternalDTD(true).
					FS(fsys).
					Parse(t.Context(), []byte(source))
				if tc.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
			})
		}
	})

	t.Run("a control character reference", func(t *testing.T) {
		// XML 1.1 permits character references to the C0/C1 control characters
		// (all but U+0000) that the XML 1.0 Char production forbids.
		doc, err := helium.NewParser().Parse(t.Context(),
			[]byte(`<?xml version="1.1"?><root>&#7;&#131;&#133;</root>`))
		require.NoError(t, err, "XML 1.1 must accept control-character references")
		require.Equal(t, "\u0007\u0083\u0085", string(doc.DocumentElement().Content()))

		// XML 1.0 (and an implicit-1.0 document) must still reject them, and U+0000
		// is invalid in every XML version.
		for _, in := range []string{
			`<?xml version="1.0"?><root>&#7;</root>`,
			`<root>&#7;</root>`,
			`<?xml version="1.1"?><root>&#0;</root>`,
		} {
			_, err := helium.NewParser().Parse(t.Context(), []byte(in))
			require.Error(t, err, "must reject %q", in)
		}
	})
}

func TestXML11PrefixUndeclaration(t *testing.T) {
	t.Parallel()

	// Namespaces in XML 1.1 §5: a prefixed namespace declaration with an empty
	// value (xmlns:pfx="") undeclares the prefix. This is well-formed only in an
	// XML 1.1 document; XML 1.0 forbids it.
	const undecl = `<doc xmlns:a="http://a/"><para xmlns:a=""/></doc>`

	// XML 1.0: rejected.
	_, err := helium.NewParser().Parse(t.Context(),
		[]byte(`<?xml version="1.0"?>`+undecl))
	require.Error(t, err, "XML 1.0 must reject a prefixed namespace undeclaration")

	// No XML declaration defaults to XML 1.0: rejected.
	_, err = helium.NewParser().Parse(t.Context(), []byte(undecl))
	require.Error(t, err, "an implicit XML 1.0 document must reject xmlns:pfx=\"\"")

	// XML 1.1: accepted, and the prefix binding is removed on the inner element.
	doc, err := helium.NewParser().Parse(t.Context(),
		[]byte(`<?xml version="1.1"?>`+undecl))
	require.NoError(t, err, "XML 1.1 must accept a prefixed namespace undeclaration")

	para := doc.DocumentElement().FirstChild().(*helium.Element)
	require.Equal(t, "para", para.Name())
	var hasUndecl bool
	for _, ns := range para.Namespaces() {
		if ns.Prefix() == "a" {
			require.Equal(t, "", ns.URI(),
				"the prefix a must be undeclared (empty URI) on the inner element")
			hasUndecl = true
		}
	}
	require.True(t, hasUndecl, "the undeclaration must be recorded on the inner element")

	// The reserved xml/xmlns prefixes may never be undeclared, even in XML 1.1.
	for _, input := range []string{
		`<?xml version="1.1"?><doc xmlns:xml=""/>`,
		`<?xml version="1.1"?><doc xmlns:xmlns=""/>`,
	} {
		_, err := helium.NewParser().Parse(t.Context(), []byte(input))
		require.Error(t, err, "must reject undeclaring a reserved prefix in %q", input)
	}
}
