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

func TestParseXML11RawRestrictedChars(t *testing.T) {
	t.Parallel()

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
}

func TestParseXML11RestrictedCharacterReferences(t *testing.T) {
	t.Parallel()

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
}

func TestParseXML11EntityValueCharacterReferences(t *testing.T) {
	t.Parallel()

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
}

func TestParseXML11ExternalEntityLiteralCharacters(t *testing.T) {
	t.Parallel()

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
}

func TestParseXML11ExternalDTDIgnoreLiteralCharacters(t *testing.T) {
	t.Parallel()

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
}
