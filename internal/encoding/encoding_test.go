package encoding

import (
	"encoding/binary"
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

func TestUCS4Decode(t *testing.T) {
	// "A" = U+0041, "<" = U+003C
	codePoints := []uint32{0x0041, 0x003C}
	wantUTF8 := "A<"

	t.Run("UCS-4BE", func(t *testing.T) {
		e := Load("ucs4be")
		require.NotNil(t, e)
		var buf []byte
		for _, cp := range codePoints {
			b := make([]byte, 4)
			binary.BigEndian.PutUint32(b, cp)
			buf = append(buf, b...)
		}
		got, err := e.NewDecoder().Bytes(buf)
		require.NoError(t, err)
		require.Equal(t, wantUTF8, string(got))
	})

	t.Run("UCS-4LE", func(t *testing.T) {
		e := Load("ucs4le")
		require.NotNil(t, e)
		var buf []byte
		for _, cp := range codePoints {
			b := make([]byte, 4)
			binary.LittleEndian.PutUint32(b, cp)
			buf = append(buf, b...)
		}
		got, err := e.NewDecoder().Bytes(buf)
		require.NoError(t, err)
		require.Equal(t, wantUTF8, string(got))
	})

	t.Run("UCS-4 2143", func(t *testing.T) {
		e := Load("ucs4_2143")
		require.NotNil(t, e)
		// 2143 byte order: for code point 0x00000041 (BE: 00 00 00 41)
		// positions 2,1,4,3 → [00, 00, 41, 00]
		var buf []byte
		for _, cp := range codePoints {
			be := make([]byte, 4)
			binary.BigEndian.PutUint32(be, cp)
			// BE [b0,b1,b2,b3] → 2143 [b1,b0,b3,b2]
			buf = append(buf, be[1], be[0], be[3], be[2])
		}
		got, err := e.NewDecoder().Bytes(buf)
		require.NoError(t, err)
		require.Equal(t, wantUTF8, string(got))
	})

	t.Run("UCS-4 3412", func(t *testing.T) {
		e := Load("ucs4_3412")
		require.NotNil(t, e)
		// 3412 byte order: for code point 0x00000041 (BE: 00 00 00 41)
		// positions 3,4,1,2 → [00, 41, 00, 00]
		var buf []byte
		for _, cp := range codePoints {
			be := make([]byte, 4)
			binary.BigEndian.PutUint32(be, cp)
			// BE [b0,b1,b2,b3] → 3412 [b2,b3,b0,b1]
			buf = append(buf, be[2], be[3], be[0], be[1])
		}
		got, err := e.NewDecoder().Bytes(buf)
		require.NoError(t, err)
		require.Equal(t, wantUTF8, string(got))
	})
}

func TestUCS4Aliases(t *testing.T) {
	aliases := []string{
		"ucs4be", "ucs-4be", "utf-32be", "utf32be", "ISO-10646-UCS-4",
	}
	for _, name := range aliases {
		require.NotNil(t, Load(name), "expected %q to be loadable", name)
	}

	leAliases := []string{"ucs4le", "ucs-4le", "utf-32le", "utf32le"}
	for _, name := range leAliases {
		require.NotNil(t, Load(name), "expected %q to be loadable", name)
	}
}

func TestUCS2(t *testing.T) {
	e := Load("ucs-2")
	require.NotNil(t, e)

	// UCS-2 is essentially UTF-16BE without surrogates.
	// "A" = U+0041 → [0x00, 0x41]
	input := []byte{0x00, 0x41, 0x00, 0x3C}
	got, err := e.NewDecoder().Bytes(input)
	require.NoError(t, err)
	require.Equal(t, "A<", string(got))
}

func TestUCS4RoundTrip(t *testing.T) {
	// Test that encode → decode is identity for BMP characters.
	testStr := "Hello, World!"

	for _, name := range []string{"ucs4be", "ucs4le", "ucs4_2143", "ucs4_3412"} {
		t.Run(name, func(t *testing.T) {
			e := Load(name)
			require.NotNil(t, e)

			encoded, err := e.NewEncoder().Bytes([]byte(testStr))
			require.NoError(t, err)
			require.True(t, len(encoded) > 0)

			decoded, err := e.NewDecoder().Bytes(encoded)
			require.NoError(t, err)
			require.Equal(t, testStr, string(decoded))
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
