package helium_test

import (
	"encoding/binary"
	"testing"
	"testing/fstest"
	"unicode/utf16"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// utf16Bytes encodes s as UTF-16 with a leading byte-order mark, in the requested
// byte order — the on-disk shape of the W3C UTF-16 external-content fixtures.
func utf16Bytes(s string, bigEndian bool) []byte {
	units := utf16.Encode([]rune(s))
	out := make([]byte, 0, (len(units)+1)*2)
	var scratch [2]byte
	put := func(u uint16) {
		if bigEndian {
			binary.BigEndian.PutUint16(scratch[:], u)
		} else {
			binary.LittleEndian.PutUint16(scratch[:], u)
		}
		out = append(out, scratch[0], scratch[1])
	}
	put(0xFEFF) // BOM
	for _, u := range units {
		put(u)
	}
	return out
}

// A UTF-16 external parsed general entity — a BOM followed by a TextDecl declaring
// encoding="UTF-16" and the body — must be decoded and expanded cleanly. The
// TextDecl is itself UTF-16-encoded, so it cannot be recognized at byte level; the
// content must be decoded from the BOM first and the TextDecl consumed on the
// decoded stream. Covers W3C sun/valid/ext02 (utf16b.xml / utf16l.xml) and
// xmltest/valid/ext-sa/008.
func TestExternalGeneralEntityUTF16TextDecl(t *testing.T) {
	t.Parallel()

	for _, bigEndian := range []bool{true, false} {
		name := "LE"
		if bigEndian {
			name = "BE"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ent := utf16Bytes(`<?xml version="1.0" encoding="UTF-16"?><child>hi</child>`, bigEndian)
			parsed, err := parseExtEntity(t, ent)
			require.NoError(t, err, "a UTF-16 BOM+TextDecl external entity must expand cleanly")
			require.NotNil(t, parsed)

			child := parsed.DocumentElement().FirstChild()
			require.NotNil(t, child, "the entity replacement text must have expanded into a child element")
			require.Equal(t, "child", child.(*helium.Element).LocalName())
			require.Equal(t, "hi", string(child.Content()))
		})
	}
}

// A UTF-16 external DTD subset must be decoded so its declarations register — even
// when the declared element name is non-ASCII (the W3C japanese/weekly-* cases
// declare Japanese-named elements). With DTD validation on, a subset that failed
// to decode would leave the element undeclared ("no declaration found").
func TestExternalSubsetUTF16NonASCIIElementDecl(t *testing.T) {
	t.Parallel()

	doc := `<?xml version="1.0"?>` + "\n" +
		`<!DOCTYPE 週報 SYSTEM "` + extSubsetName + `">` + "\n" +
		`<週報>x</週報>`

	// A UTF-16 subset with a leading TextDecl.
	dtdWithDecl := utf16Bytes(`<?xml encoding="UTF-16"?>`+"\n"+`<!ELEMENT 週報 (#PCDATA)>`+"\n", true)
	// A UTF-16 subset with NO TextDecl, opening on a comment — the on-disk shape
	// of weekly-utf-16.dtd (BOM, then "<!--"). It must still decode by BOM.
	dtdNoDecl := utf16Bytes(`<!-- weekly -->`+"\n"+`<!ELEMENT 週報 (#PCDATA)>`+"\n", false)

	for _, tc := range []struct {
		name string
		dtd  []byte
	}{
		{"with-textdecl", dtdWithDecl},
		{"no-textdecl", dtdNoDecl},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fsys := fstest.MapFS{extSubsetName: &fstest.MapFile{Data: tc.dtd}}
			parsed, err := helium.NewParser().
				BlockXXE(false).
				LoadExternalDTD(true).
				ValidateDTD(true).
				FS(fsys).
				Parse(t.Context(), []byte(doc))
			require.NoError(t, err, "the non-ASCII element declared in the UTF-16 subset must be recognized")
			require.NotNil(t, parsed)
			require.Equal(t, "週報", parsed.DocumentElement().LocalName())
		})
	}
}

// A UTF-16 external-entity TextDecl carrying a forbidden 'standalone'
// pseudo-attribute must be rejected on the decoded stream, exactly as the
// ASCII-shaped TextDecl is.
func TestExternalGeneralEntityUTF16TextDeclStandaloneRejected(t *testing.T) {
	t.Parallel()

	ent := utf16Bytes(`<?xml encoding="UTF-16" standalone="yes"?><child>hi</child>`, true)
	_, err := parseExtEntity(t, ent)
	require.Error(t, err, "a standalone pseudo-attribute in a UTF-16 TextDecl must be rejected")
}
