package helium_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

func TestParseXML11RawRestrictedChars(t *testing.T) {
	t.Parallel()

	const restricted = "\x7f"
	for _, tc := range []struct {
		name   string
		source func(string, string) string
	}{
		{
			name: "comment",
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
	} {
		for _, version := range []struct {
			name    string
			value   string
			wantErr bool
		}{
			{name: "XML 1.0", value: "1.0"},
			{name: "XML 1.1", value: "1.1", wantErr: true},
		} {
			t.Run(tc.name+" "+version.name, func(t *testing.T) {
				t.Parallel()

				_, err := helium.NewParser().Parse(t.Context(), []byte(tc.source(version.value, restricted)))
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
			source: `<?xml version="1.1"?><root>&#127;</root>`,
			check: func(t *testing.T, doc *helium.Document) {
				require.Equal(t, "\x7f", string(doc.DocumentElement().Content()))
			},
		},
		{
			name:   "attribute",
			source: `<?xml version="1.1"?><root attr="&#127;"/>`,
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
