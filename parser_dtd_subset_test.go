package helium_test

import (
	"bytes"
	"encoding/binary"
	"os"
	"testing"
	"testing/fstest"
	"unicode/utf16"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// condSectExtName is the external-subset filename shared by the conditional-
// section tests.
const condSectExtName = "cond.dtd"

func condSectDoc() string {
	return `<?xml version="1.0"?>` + "\n" +
		`<!DOCTYPE doc SYSTEM "` + condSectExtName + `">` + "\n" +
		`<doc>&greeting;</doc>`
}

func condSectParse(t *testing.T, dtd string) (*helium.Document, error) {
	t.Helper()
	fsys := fstest.MapFS{condSectExtName: &fstest.MapFile{Data: []byte(dtd)}}
	return helium.NewParser().
		BlockXXE(false).
		LoadExternalDTD(true).
		SubstituteEntities(true).
		FS(fsys).
		Parse(t.Context(), []byte(condSectDoc()))
}

// condSectParseValidating is condSectParse with DTD validation enabled, so the
// validity-only "Proper Conditional Section/PE Nesting" constraint is enforced.
func condSectParseValidating(t *testing.T, dtd string) (*helium.Document, error) {
	t.Helper()
	fsys := fstest.MapFS{condSectExtName: &fstest.MapFile{Data: []byte(dtd)}}
	return helium.NewParser().
		BlockXXE(false).
		LoadExternalDTD(true).
		SubstituteEntities(true).
		ValidateDTD(true).
		FS(fsys).
		Parse(t.Context(), []byte(condSectDoc()))
}

// extSubsetName is the external-subset filename shared by the TextDecl tests.
const extSubsetName = "sub.dtd"

func extSubsetDoc() string {
	return `<?xml version="1.0"?>` + "\n" +
		`<!DOCTYPE doc SYSTEM "` + extSubsetName + `">` + "\n" +
		`<doc>&greeting;</doc>`
}

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

func TestConditionalSection(t *testing.T) {
	t.Parallel()

	// A correctly-cased INCLUDE section parses cleanly and its declarations take
	// effect (the general entity declared inside it resolves in the document).
	t.Run("INCLUDE is accepted", func(t *testing.T) {
		const dtd = "<![INCLUDE[\n<!ELEMENT doc (#PCDATA)>\n<!ENTITY greeting \"hi from include\">\n]]>\n"
		doc, err := condSectParse(t, dtd)
		require.NoError(t, err, "a well-formed INCLUDE section must parse")
		require.NotNil(t, doc)
		require.Equal(t, "hi from include", string(doc.DocumentElement().Content()))
	})

	// A correctly-cased IGNORE section parses cleanly; its body is discarded, so a
	// declaration OUTSIDE the section is the one that takes effect.
	t.Run("IGNORE is accepted", func(t *testing.T) {
		const dtd = "<![IGNORE[\n<!ENTITY greeting \"ignored\">\n]]>\n" +
			"<!ELEMENT doc (#PCDATA)>\n<!ENTITY greeting \"kept\">\n"
		doc, err := condSectParse(t, dtd)
		require.NoError(t, err, "a well-formed IGNORE section must parse")
		require.NotNil(t, doc)
		require.Equal(t, "kept", string(doc.DocumentElement().Content()))
	})

	// A conditional section keyword is case-sensitive (XML §3.4 P62/P63): only the
	// exact literals INCLUDE and IGNORE are permitted. A miscased keyword such as
	// lowercase "include" is a fatal well-formedness error and must be reported even
	// from the top level of the external subset.
	t.Run("lowercase include is rejected", func(t *testing.T) {
		const dtd = "<![ include [\n<!ELEMENT doc (#PCDATA)>\n]]>\n"
		_, err := condSectParse(t, dtd)
		require.Error(t, err, "lowercase 'include' keyword must be a fatal error")
		require.Contains(t, err.Error(), "INCLUDE or IGNORE keyword")
	})

	// Lowercase "ignore" is equally a fatal keyword error.
	t.Run("lowercase ignore is rejected", func(t *testing.T) {
		const dtd = "<![ ignore [\n]]>\n<!ELEMENT doc (#PCDATA)>\n"
		_, err := condSectParse(t, dtd)
		require.Error(t, err, "lowercase 'ignore' keyword must be a fatal error")
		require.Contains(t, err.Error(), "INCLUDE or IGNORE keyword")
	})

	// A misspelled / non-keyword token where INCLUDE|IGNORE is required is fatal.
	t.Run("a bogus keyword is rejected", func(t *testing.T) {
		const dtd = "<![ CDATA [\n<!ELEMENT doc (#PCDATA)>\n]]>\n"
		_, err := condSectParse(t, dtd)
		require.Error(t, err, "a non-INCLUDE/IGNORE keyword must be a fatal error")
		require.Contains(t, err.Error(), "INCLUDE or IGNORE keyword")
	})

	// A missing '[' after a valid INCLUDE keyword is malformed and fatal
	// (P62: '<![' S? 'INCLUDE' S? '[').
	t.Run("a missing bracket is rejected", func(t *testing.T) {
		const dtd = "<![INCLUDE\n<!ELEMENT doc (#PCDATA)>\n]]>\n"
		_, err := condSectParse(t, dtd)
		require.Error(t, err, "missing '[' after INCLUDE must be a fatal error")
		require.Contains(t, err.Error(), "INCLUDE or IGNORE keyword")
	})

	// An INCLUDE conditional section left unterminated (no closing "]]>") at the end
	// of the fully-buffered external subset is a genuine truncation, not a streaming
	// boundary, and is a fatal error (XML §3.4; W3C not-wf-not-sa-004,
	// ibm-not-wf-P62-ibm62n07; libxml2 "Content error in the external subset").
	t.Run("an unterminated INCLUDE is rejected", func(t *testing.T) {
		const dtd = "<![ INCLUDE [\n<!ELEMENT doc (#PCDATA)>\n<!ENTITY greeting \"x\">\n"
		_, err := condSectParse(t, dtd)
		require.Error(t, err, "an unterminated INCLUDE section must be a fatal error")
		require.Contains(t, err.Error(), "conditional section")
	})

	// An unterminated IGNORE section is equally fatal.
	t.Run("an unterminated IGNORE is rejected", func(t *testing.T) {
		const dtd = "<![ IGNORE [\n<!ELEMENT doc (#PCDATA)>\n"
		_, err := condSectParse(t, dtd)
		require.Error(t, err, "an unterminated IGNORE section must be a fatal error")
		require.Contains(t, err.Error(), "conditional section")
	})

	// The INCLUDE|IGNORE keyword may be supplied by a parameter entity. The keyword
	// is validated AFTER PE expansion, so a PE resolving to INCLUDE keeps the
	// section well-formed and must NOT be rejected.
	t.Run("a PE-supplied INCLUDE is accepted", func(t *testing.T) {
		const dtd = "<!ENTITY % inc \"INCLUDE\">\n<![ %inc; [\n" +
			"<!ELEMENT doc (#PCDATA)>\n<!ENTITY greeting \"pe include\">\n]]>\n"
		doc, err := condSectParse(t, dtd)
		require.NoError(t, err, "a PE-supplied INCLUDE keyword must be accepted")
		require.NotNil(t, doc)
		require.Equal(t, "pe include", string(doc.DocumentElement().Content()))
	})

	// A parameter entity supplying INCLUDE[ (keyword plus opening bracket) in one
	// expansion is also well-formed and must be accepted.
	t.Run("a PE-supplied INCLUDE with a bracket is accepted", func(t *testing.T) {
		const dtd = "<!ENTITY % inc \"INCLUDE[\">\n<![ %inc;\n" +
			"<!ELEMENT doc (#PCDATA)>\n<!ENTITY greeting \"pe inc bracket\">\n]]>\n"
		doc, err := condSectParse(t, dtd)
		require.NoError(t, err, "a PE-supplied 'INCLUDE[' must be accepted")
		require.NotNil(t, doc)
		require.Equal(t, "pe inc bracket", string(doc.DocumentElement().Content()))
	})

	// VC "Proper Conditional Section/PE Nesting" (XML §3.4): when the "<![" opens in
	// the external subset but the INCLUDE keyword and its "[" are supplied by a
	// parameter entity, the section markup straddles an entity boundary. A
	// validating processor must report it (W3C xmlconf invalid-not-sa-022). It is a
	// validity constraint, so it is reported ONLY when validating; the
	// non-validating counterpart above accepts the same DTD.
	t.Run("a PE straddling the markup is rejected when validating", func(t *testing.T) {
		const dtd = "<!ENTITY % e \"INCLUDE[\">\n<![ %e; <!ELEMENT doc (#PCDATA)> ]]>\n" +
			"<!ENTITY greeting \"boundary\">\n"
		_, err := condSectParseValidating(t, dtd)
		require.Error(t, err, "a PE straddling the conditional-section markup must be a validity error")
		require.Contains(t, err.Error(), "not in the same entity")
	})

	// When the ENTIRE conditional section — "<![", keyword, "[", body and "]]>" —
	// comes from a single parameter-entity replacement text, the markup does NOT
	// straddle an entity boundary and a validating processor must accept it.
	t.Run("a section wholly inside a PE is accepted when validating", func(t *testing.T) {
		const dtd = "<!ENTITY % sec \"<![INCLUDE[ <!ELEMENT doc (#PCDATA)> " +
			"<!ENTITY greeting 'whole pe'> ]]>\">\n%sec;\n"
		doc, err := condSectParseValidating(t, dtd)
		require.NoError(t, err, "a conditional section wholly inside one PE must be accepted")
		require.NotNil(t, doc)
		require.Equal(t, "whole pe", string(doc.DocumentElement().Content()))
	})

	// A literal (non-PE) INCLUDE section is well within a single entity and must be
	// accepted even when validating.
	t.Run("a literal section is accepted when validating", func(t *testing.T) {
		const dtd = "<![INCLUDE[\n<!ELEMENT doc (#PCDATA)>\n<!ENTITY greeting \"lit\">\n]]>\n"
		doc, err := condSectParseValidating(t, dtd)
		require.NoError(t, err, "a literal INCLUDE section must be accepted when validating")
		require.NotNil(t, doc)
		require.Equal(t, "lit", string(doc.DocumentElement().Content()))
	})

	t.Run("in the internal subset", func(t *testing.T) {
		t.Run("internal subset PE is not well formed", func(t *testing.T) {
			t.Parallel()

			// A conditional section supplied through a parameter entity requires the
			// PE reference to sit inside an entity value in the internal subset, which
			// violates the PEs in Internal Subset WFC (XML §2.8) and is fatal —
			// matching libxml2. Conditional sections through a PE are only valid in
			// the external subset (see the "external DTD" subtest and
			// TestParseExternalDTDPEInIncludeSectionExpands).
			const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ENTITY % inc "INCLUDE">
  <!ENTITY % sect "<![%inc;[<!ELEMENT doc (child)><!ELEMENT child (#PCDATA)>]]>">
  %sect;
]>
<doc>
    <child>text</child>
</doc>`

			p := helium.NewParser().ValidateDTD(true)
			_, err := p.Parse(t.Context(), []byte(input))
			require.Error(t, err, "a PE reference in an internal-subset entity value is not well formed")
			require.Contains(t, err.Error(), "PEReferences forbidden in internal subset")
		})

		// Invalid conditional-section keywords are covered where they are legal — the
		// external subset — in parser_condsect_test.go (miscased/non-keyword tokens
		// all raising "INCLUDE or IGNORE keyword expected").

		t.Run("external DTD", func(t *testing.T) {
			t.Parallel()

			path := "testdata/libxml2-compat/valid/cond_sect1.xml"
			input, err := os.ReadFile(path)
			require.NoError(t, err)

			p := helium.NewParser().LoadExternalDTD(true).BaseURI(path)
			doc, err := p.Parse(t.Context(), input)
			require.NoError(t, err, "external DTD with conditional sections should parse")
			require.NotNil(t, doc)

			expected, err := os.ReadFile("testdata/libxml2-compat/valid/cond_sect1.xml.expected")
			require.NoError(t, err)

			var buf bytes.Buffer
			d := helium.NewWriter()
			require.NoError(t, d.WriteTo(&buf, doc))
			require.Equal(t, string(expected), buf.String())
		})
	})

	// an external-subset
	// conditional section whose INCLUDE keyword is supplied by a parameter entity
	// (`<![ %e; ... ]]>` with %e; -> "INCLUDE["). The blank skip after "<![" must
	// leave the "%" for expansion (not consume it unexpanded), and the spent PE
	// cursor must be popped before the body floor is captured, so the body
	// declarations (here a defaulting <!ATTLIST>) are parsed and applied.
	t.Run("a keyword supplied by a parameter entity", func(t *testing.T) {
		const input = `<!DOCTYPE doc SYSTEM "d.dtd"><doc></doc>`
		dtd := "<!ENTITY % e \"INCLUDE[\">\n" +
			"<!ELEMENT doc (#PCDATA)>\n" +
			"<![ %e; <!ATTLIST doc a1 CDATA \"v1\"> ]]>\n"
		fsys := fstest.MapFS{dtdSystemID: {Data: []byte(dtd)}}

		doc, err := helium.NewParser().BlockXXE(false).
			LoadExternalDTD(true).DefaultDTDAttributes(true).SubstituteEntities(true).
			FS(fsys).Parse(t.Context(), []byte(input))
		require.NoError(t, err)
		require.NotNil(t, doc)

		str, werr := helium.WriteString(doc.DocumentElement())
		require.NoError(t, werr)
		require.Contains(t, str, wantAttrA1V1, "the <!ATTLIST> inside the PE-supplied INCLUDE section must supply the default attribute")
	})
}

func TestExternalSubsetTextDecl(t *testing.T) {
	t.Parallel()

	// An external DTD subset (loaded via <!DOCTYPE ... SYSTEM>) may begin with a
	// TextDecl — '<?xml' VersionInfo? EncodingDecl S? '?>' — where VersionInfo is
	// optional, EncodingDecl required, and no StandaloneDecl is permitted. It must
	// be consumed, not rejected as a misplaced XML declaration. libxml2 accepts such
	// documents; the W3C XML conformance suite has many valid cases whose external
	// DTD opens with "<?xml encoding=...?>".
	t.Run("text declaration", func(t *testing.T) {
		const dtd = `<?xml encoding="utf-8"?>
<!ELEMENT doc (#PCDATA)>
<!ENTITY greeting "hello from ext subset">
`
		fsys := fstest.MapFS{extSubsetName: &fstest.MapFile{Data: []byte(dtd)}}
		parsed, err := helium.NewParser().
			BlockXXE(false).
			LoadExternalDTD(true).
			SubstituteEntities(true).
			FS(fsys).
			Parse(t.Context(), []byte(extSubsetDoc()))
		require.NoError(t, err, "a TextDecl at the start of the external subset must be accepted")
		require.NotNil(t, parsed)

		root := parsed.DocumentElement()
		require.NotNil(t, root)
		require.Equal(t, "doc", root.LocalName())
		// The general entity declared in the external subset resolved, proving the
		// subset was parsed past its TextDecl, and never abandoned.
		require.Equal(t, "hello from ext subset", string(root.Content()))
	})

	// A version-bearing TextDecl (VersionInfo present) at the start of the external
	// subset is equally valid and must also be accepted.
	t.Run("with a version", func(t *testing.T) {
		const dtd = `<?xml version="1.0" encoding="UTF-8"?>
<!ELEMENT doc (#PCDATA)>
<!ENTITY greeting "versioned">
`
		fsys := fstest.MapFS{extSubsetName: &fstest.MapFile{Data: []byte(dtd)}}
		parsed, err := helium.NewParser().
			BlockXXE(false).
			LoadExternalDTD(true).
			SubstituteEntities(true).
			FS(fsys).
			Parse(t.Context(), []byte(extSubsetDoc()))
		require.NoError(t, err)
		require.NotNil(t, parsed)
		require.Equal(t, "versioned", string(parsed.DocumentElement().Content()))
	})

	// A TextDecl version applies to the external DTD only. An XML 1.0 DTD under an
	// XML 1.1 document retains XML 1.0 character-reference rules, and the document
	// returns to its XML 1.1 rules after the DTD input is drained.
	t.Run("the version is scoped to the subset", func(t *testing.T) {
		t.Run("XML 1.0 DTD retains XML 1.0 character-reference rules", func(t *testing.T) {
			t.Parallel()

			const dtd = `<?xml version="1.0" encoding="UTF-8"?>
<!ENTITY control "&#1;">`
			fsys := fstest.MapFS{extSubsetName: &fstest.MapFile{Data: []byte(dtd)}}
			input := `<?xml version="1.1"?>
<!DOCTYPE doc SYSTEM "` + extSubsetName + `"><doc/>`

			_, err := helium.NewParser().
				BlockXXE(false).
				LoadExternalDTD(true).
				FS(fsys).
				Parse(t.Context(), []byte(input))
			require.Error(t, err)
			require.Contains(t, err.Error(), "invalid XML char value 1")
		})

		t.Run("document XML version is restored", func(t *testing.T) {
			t.Parallel()

			const dtd = `<?xml version="1.0" encoding="UTF-8"?>
<!ELEMENT doc ANY>`
			fsys := fstest.MapFS{extSubsetName: &fstest.MapFile{Data: []byte(dtd)}}
			input := `<?xml version="1.1"?>
<!DOCTYPE doc SYSTEM "` + extSubsetName + `"><doc>&#1;</doc>`

			parsed, err := helium.NewParser().
				BlockXXE(false).
				LoadExternalDTD(true).
				FS(fsys).
				Parse(t.Context(), []byte(input))
			require.NoError(t, err)
			require.Equal(t, "1.1", parsed.Version())
		})
	})

	// A malformed TextDecl in the external subset (here version-only: EncodingDecl is
	// required in a TextDecl) must be rejected AND the error must locate the external
	// DTD file, matching every other declaration error reported from that resource.
	t.Run("a malformed declaration reports its file", func(t *testing.T) {
		const dtd = `<?xml version="1.0"?>
<!ELEMENT doc (#PCDATA)>
`
		fsys := fstest.MapFS{extSubsetName: &fstest.MapFile{Data: []byte(dtd)}}
		_, err := helium.NewParser().
			BlockXXE(false).
			LoadExternalDTD(true).
			SubstituteEntities(true).
			FS(fsys).
			Parse(t.Context(), []byte(extSubsetDoc()))
		require.Error(t, err)
		require.Contains(t, err.Error(), extSubsetName,
			"a malformed external-subset TextDecl error must name the DTD file")
	})

	// An unsupported encoding declared in the external-subset TextDecl must also fail
	// with the DTD file located — the error travels the switchEncoding branch, not
	// parseTextDecl, so it must be wrapped with the source URI too.
	t.Run("an unsupported encoding reports its file", func(t *testing.T) {
		const dtd = `<?xml encoding="X-UNKNOWN-ENC"?>
<!ELEMENT doc (#PCDATA)>
`
		fsys := fstest.MapFS{extSubsetName: &fstest.MapFile{Data: []byte(dtd)}}
		_, err := helium.NewParser().
			BlockXXE(false).
			LoadExternalDTD(true).
			SubstituteEntities(true).
			FS(fsys).
			Parse(t.Context(), []byte(extSubsetDoc()))
		require.Error(t, err)
		require.Contains(t, err.Error(), extSubsetName,
			"an unsupported external-subset TextDecl encoding error must name the DTD file")
	})

	// An IGNORE conditional section validates literal content as decoded runes. This
	// keeps a legal non-ASCII XML 1.1 rune intact while still rejecting a raw XML
	// 1.1 restricted character, including inside a nested IGNORE section.
	t.Run("IGNORE literal rune validation", func(t *testing.T) {
		for _, tc := range []struct {
			name    string
			dtd     string
			wantErr bool
		}{
			{
				name: "accepts legal multibyte rune in nested ignore section",
				dtd: `<?xml version="1.1" encoding="UTF-8"?>
<![IGNORE[outer <![IGNORE[` + "\u0100" + `]]> tail]]>`,
			},
			{
				name: "rejects raw restricted character",
				dtd: `<?xml version="1.1" encoding="UTF-8"?>
<![IGNORE[` + "\x7f" + `]]>`,
				wantErr: true,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				fsys := fstest.MapFS{extSubsetName: &fstest.MapFile{Data: []byte(tc.dtd)}}
				input := `<?xml version="1.1"?>
<!DOCTYPE doc SYSTEM "` + extSubsetName + `"><doc/>`
				_, err := helium.NewParser().
					BlockXXE(false).
					LoadExternalDTD(true).
					FS(fsys).
					Parse(t.Context(), []byte(input))
				if tc.wantErr {
					require.ErrorIs(t, err, helium.ErrInvalidChar)
					return
				}
				require.NoError(t, err)
			})
		}
	})
}

func TestExternalSubsetUTF16(t *testing.T) {
	t.Parallel()

	// A UTF-16 external parsed general entity — a BOM followed by a TextDecl declaring
	// encoding="UTF-16" and the body — must be decoded and expanded cleanly. The
	// TextDecl is itself UTF-16-encoded, so it cannot be recognized at byte level; the
	// content must be decoded from the BOM first and the TextDecl consumed on the
	// decoded stream. Covers W3C sun/valid/ext02 (utf16b.xml / utf16l.xml) and
	// xmltest/valid/ext-sa/008.
	t.Run("a general entity text declaration", func(t *testing.T) {
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
	})

	// A UTF-16 external-entity TextDecl carrying a forbidden 'standalone'
	// pseudo-attribute must be rejected on the decoded stream, exactly as the
	// ASCII-shaped TextDecl is.
	t.Run("a standalone general entity text declaration is rejected", func(t *testing.T) {
		ent := utf16Bytes(`<?xml encoding="UTF-16" standalone="yes"?><child>hi</child>`, true)
		_, err := parseExtEntity(t, ent)
		require.Error(t, err, "a standalone pseudo-attribute in a UTF-16 TextDecl must be rejected")
	})

	// A UTF-16 external DTD subset must be decoded so its declarations register — even
	// when the declared element name is non-ASCII (the W3C japanese/weekly-* cases
	// declare Japanese-named elements). With DTD validation on, a subset that failed
	// to decode would leave the element undeclared ("no declaration found").
	t.Run("a non-ASCII element declaration", func(t *testing.T) {
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
	})
}
