package helium_test

import (
	"testing"

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
