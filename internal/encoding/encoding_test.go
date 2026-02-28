package encoding

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestISO88591(t *testing.T) {
	e := Load("iso-8859-1")
	require.NotNil(t, e)

	dec := e.NewDecoder()
	enc := e.NewEncoder()
	for i := 0; i <= 255; i++ {
		v := string([]byte{byte(i)})
		s, err := dec.String(v)
		require.NoError(t, err)

		if i >= 0x80 && i <= 0x9f {
			continue
		}
		v1, err := enc.String(s)
		require.NoError(t, err)
		require.Equal(t, v, v1)
	}
}

func TestLoadAliasCoverage(t *testing.T) {
	tests := []struct {
		canonical string
		aliases   []string
	}{
		{
			canonical: "iso-8859-1",
			aliases: []string{
				"latin1",
				"l1",
				"iso-ir-100",
				"csisolatin1",
				"ibm819",
				"cp819",
				"iso_8859-1:1987",
			},
		},
		{
			canonical: "iso-8859-9",
			aliases: []string{
				"latin5",
				"l5",
				"iso-ir-148",
				"csisolatin5",
				"iso_8859-9:1989",
			},
		},
		{
			canonical: "iso-8859-11",
			aliases: []string{
				"tis-620",
			},
		},
		{
			canonical: "shift_jis",
			aliases: []string{
				"sjis",
				"ms932",
				"windows-31j",
				"x-sjis",
			},
		},
		{
			canonical: "euc-jp",
			aliases: []string{
				"x-euc-jp",
			},
		},
		{
			canonical: "windows1252",
			aliases: []string{
				"windows-1252",
				"cp1252",
			},
		},
		{
			canonical: "koi8u",
			aliases: []string{
				"koir8u",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.canonical, func(t *testing.T) {
			requireEncodingLoadable(t, tt.canonical)
			for _, alias := range tt.aliases {
				requireEquivalentEncoding(t, tt.canonical, alias)
			}
		})
	}
}

func TestLoadUnknown(t *testing.T) {
	require.Nil(t, Load("definitely-not-an-encoding"))
}

func requireEncodingLoadable(t *testing.T, name string) {
	t.Helper()
	require.NotNil(t, Load(name), "expected encoding %q to be loadable", name)
}

func requireEquivalentEncoding(t *testing.T, canonical, alias string) {
	t.Helper()

	canonicalEnc := Load(canonical)
	aliasEnc := Load(alias)
	require.NotNil(t, canonicalEnc, "canonical encoding %q is not loadable", canonical)
	require.NotNil(t, aliasEnc, "alias encoding %q is not loadable", alias)

	// Use bytes that exercise both ASCII and high-byte conversion.
	samples := []string{
		string([]byte{0x41}),
		string([]byte{0x80}),
		string([]byte{0xA4}),
	}
	for _, sample := range samples {
		gotCanonical, errCanonical := canonicalEnc.NewDecoder().String(sample)
		gotAlias, errAlias := aliasEnc.NewDecoder().String(sample)
		require.Equal(t, errCanonical == nil, errAlias == nil, "decode error mismatch: canonical=%q alias=%q sample=%#v", canonical, alias, sample)
		require.Equal(t, gotCanonical, gotAlias, "decoded output mismatch: canonical=%q alias=%q sample=%#v", canonical, alias, sample)
	}
}
