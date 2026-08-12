package helium_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"testing"
	"unicode/utf16"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/sax"
	"github.com/stretchr/testify/require"
)

// utf16be encodes an ASCII string as UTF-16BE bytes (each ASCII code point
// becomes 0x00 followed by the byte), the form used for the fixed-width test
// documents below.
func utf16be(s string) []byte {
	out := make([]byte, 0, len(s)*2)
	for i := range len(s) {
		out = append(out, 0x00, s[i])
	}
	return out
}

// utf16le encodes an ASCII string as UTF-16LE bytes (byte then 0x00).
func utf16le(s string) []byte {
	out := make([]byte, 0, len(s)*2)
	for i := range len(s) {
		out = append(out, s[i], 0x00)
	}
	return out
}

// encodeUCS4 encodes s as UCS-4 (4 bytes per code point) using the byte order
// described by order, which maps the four output byte positions to a shift
// amount (in bits) applied to the rune. This lets a single helper produce all
// four UCS-4 byte orders recognized by the encoding auto-detector.
func encodeUCS4(s string, order [4]uint) []byte {
	var buf bytes.Buffer
	for _, r := range s {
		u := uint32(r)
		for _, shift := range order {
			buf.WriteByte(byte(u >> shift))
		}
	}
	return buf.Bytes()
}

// utf16beWithUnits encodes s as UTF-16BE (with a BOM) and appends any extra raw
// 16-bit code units verbatim, so malformed sequences (e.g. an unpaired
// surrogate) can be injected. The pieces are concatenated in order.
func utf16beDoc(parts ...any) []byte {
	out := []byte{0xFE, 0xFF} // UTF-16BE BOM
	for _, p := range parts {
		switch v := p.(type) {
		case string:
			for _, u := range utf16.Encode([]rune(v)) {
				out = binary.BigEndian.AppendUint16(out, u)
			}
		case uint16:
			out = binary.BigEndian.AppendUint16(out, v)
		default:
			panic("unsupported part type")
		}
	}
	return out
}

func TestXMLDecl(t *testing.T) {
	t.Parallel()

	t.Run("forms", func(t *testing.T) {
		const content = `<root />`
		inputs := map[string]struct {
			version    string
			encoding   string
			standalone int
		}{
			`<?xml version="1.0"?>` + content:                                   {lexicon.XSLTVersion10, "utf8", int(helium.StandaloneImplicitNo)},
			`<?xml version="1.0" encoding="euc-jp"?>` + content:                 {lexicon.XSLTVersion10, "euc-jp", int(helium.StandaloneImplicitNo)},
			`<?xml version="1.0" encoding="cp932" standalone='yes'?>` + content: {lexicon.XSLTVersion10, "cp932", int(helium.StandaloneExplicitYes)},
		}

		for input, expect := range inputs {
			p := helium.NewParser()
			doc, err := p.Parse(t.Context(), []byte(input))
			require.NoError(t, err, "Parse should succeed for '%s'", input)

			require.Equal(t, expect.version, doc.Version(), "version matches")
			require.Equal(t, expect.encoding, doc.Encoding(), "encoding matches")
			require.Equal(t, expect.standalone, int(doc.Standalone()), "standalone matches")
		}
	})

	// XML-declaration error branches.
	t.Run("malformed", func(t *testing.T) {
		bad := []string{
			`<?xml?><root/>`,                         // missing version
			`<?xml version="1.0" foo="bar"?><root/>`, // unknown pseudo-attr / unclosed
			`<?xml version=1.0?><root/>`,             // unquoted version value
		}
		for _, in := range bad {
			_, err := helium.NewParser().Parse(t.Context(), []byte(in))
			require.Error(t, err, "malformed decl %q should error", in)
		}
	})

	// the XML §2.8 VersionNum constraint as
	// libxml2 applies it in xmlParseXMLDecl: VersionNum ::= '1.' [0-9]+ since XML 1.0
	// 5th edition, so a version outside the 1.x family is fatal
	// (XML_ERR_UNKNOWN_VERSION), while a 1.x version other than "1.0"/"1.1" only
	// warns (XML_WAR_UNKNOWN_VERSION) and parsing continues — helium implements XML
	// 1.1, so it stays silent there where libxml2 warns. The grammar itself is looser
	// than the constraint — versionNumLen, which both the byte path and the
	// rune-cursor (UTF-16) path scan through, accepts [0-9] '.' [0-9]+, mirroring
	// libxml2's xmlParseVersionNum — so the constraint must be enforced separately.
	t.Run("unsupported version", func(t *testing.T) {
		const content = `<root/>`

		t.Run("a non-1.x version is fatal", func(t *testing.T) {
			t.Parallel()

			for _, version := range []string{"0.0", "2.0", "9.9", "0.9"} {
				src := `<?xml version="` + version + `"?>` + content
				doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
				require.Error(t, err, "version %q must be rejected", version)
				require.ErrorIs(t, err, helium.ErrUnsupportedXMLVersion)
				require.Contains(t, err.Error(), version, "the error names the offending version")
				require.Nil(t, doc)
			}
		})

		t.Run("a 1.x version warns but parses", func(t *testing.T) {
			t.Parallel()

			for _, version := range []string{"1.2", "1.9"} {
				src := `<?xml version="` + version + `"?>` + content
				doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
				require.NoError(t, err, "version %q must parse", version)
				require.NotNil(t, doc)
				require.Equal(t, version, doc.Version(), "the declared version is preserved")
			}
		})

		t.Run("1.0 and 1.1 are silent", func(t *testing.T) {
			t.Parallel()

			for _, version := range []string{ver10, ver11} {
				var warnings []string
				s := sax.New()
				s.SetOnWarning(sax.WarningFunc(func(_ context.Context, err error) error {
					warnings = append(warnings, err.Error())
					return nil
				}))
				src := `<?xml version="` + version + `"?>` + content
				_, err := helium.NewParser().SAXHandler(s).Parse(t.Context(), []byte(src))
				require.NoError(t, err)
				require.Empty(t, warnings, "version %q must not warn", version)
			}
		})

		t.Run("the fatal version is rejected on the UTF-16 path", func(t *testing.T) {
			t.Parallel()

			src := utf16Bytes(`<?xml version="2.0" encoding="utf-16"?>`+content, true)
			_, err := helium.NewParser().Parse(t.Context(), src)
			require.ErrorIs(t, err, helium.ErrUnsupportedXMLVersion)
		})

		// The "1.x" family check that turns a fatal version into a warning presumes
		// the value already satisfies VersionNum. Both parse paths must therefore
		// enforce that grammar, or a value like "1.x" is warned past on one of them
		// and then refused by the Writer, which is stricter than the family check.
		t.Run("a malformed VersionNum is rejected on both parse paths", func(t *testing.T) {
			t.Parallel()

			for _, version := range []string{"1.x", "1.0.0", "1.", "x.0"} {
				utf16Src := utf16Bytes(`<?xml version="`+version+`" encoding="utf-16"?>`+content, true)
				_, err := helium.NewParser().Parse(t.Context(), utf16Src)
				require.Error(t, err, "version %q must be rejected on the UTF-16 path", version)

				_, err = helium.NewParser().Parse(t.Context(), []byte(`<?xml version="`+version+`"?>`+content))
				require.Error(t, err, "version %q must be rejected on the byte path", version)
			}
		})

		t.Run("LenientXMLDecl relaxes order, not the version constraint", func(t *testing.T) {
			t.Parallel()

			src := `<?xml encoding="utf-8" version="2.0"?>` + content
			_, err := helium.NewParser().LenientXMLDecl(true).Parse(t.Context(), []byte(src))
			require.ErrorIs(t, err, helium.ErrUnsupportedXMLVersion)
		})

		// The fuzz roundtrip that surfaced this: the parser accepted a version the
		// Writer's stricter VersionNum check then refused to serialize, so a parsed
		// document could not be written back out. Rejecting at parse time keeps the
		// two grammars from disagreeing.
		t.Run("every parsed version serializes", func(t *testing.T) {
			t.Parallel()

			for _, version := range []string{ver10, ver11, "1.9"} {
				src := `<?xml version="` + version + `"?>` + content
				doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
				require.NoError(t, err)

				var buf bytes.Buffer
				require.NoError(t, helium.NewWriter().WriteTo(&buf, doc),
					"a document the parser accepted must serialize")

				_, err = helium.NewParser().Parse(t.Context(), buf.Bytes())
				require.NoError(t, err, "the serialized output must parse back")
			}
		})
	})

	// three DTD/XML declarations that are
	// missing a component the grammar makes mandatory. Each production has a
	// malformed form that must now be a fatal well-formedness error AND a
	// well-formed near-miss (including a present-but-empty literal) that must still
	// parse, guarding against over-rejection.
	t.Run("missing mandatory part", func(t *testing.T) {
		// NotationDecl [82]: '<!NOTATION' S Name S (ExternalID | PublicID) S? '>'.
		// The ExternalID/PublicID is mandatory (W3C ibm-not-wf-P82-ibm82n03).
		t.Run("notation", func(t *testing.T) {
			t.Parallel()

			t.Run("missing ExternalID/PublicID rejected", func(t *testing.T) {
				t.Parallel()
				_, err := helium.NewParser().Parse(t.Context(),
					[]byte(`<?xml version="1.0"?><!DOCTYPE root [<!NOTATION n >]><root/>`))
				require.ErrorIs(t, err, helium.ErrNotationExternalIDRequired)
			})

			t.Run("SYSTEM form parses", func(t *testing.T) {
				t.Parallel()
				_, err := helium.NewParser().Parse(t.Context(),
					[]byte(`<?xml version="1.0"?><!DOCTYPE root [<!NOTATION n SYSTEM "n.dtd">]><root/>`))
				require.NoError(t, err)
			})

			t.Run("PUBLIC-only form parses", func(t *testing.T) {
				t.Parallel()
				_, err := helium.NewParser().Parse(t.Context(),
					[]byte(`<?xml version="1.0"?><!DOCTYPE root [<!NOTATION n PUBLIC "pub-id">]><root/>`))
				require.NoError(t, err)
			})

			t.Run("SYSTEM empty literal parses", func(t *testing.T) {
				t.Parallel()
				// A present-but-empty SystemLiteral is well formed; found=true so it
				// is not mistaken for a missing ExternalID.
				_, err := helium.NewParser().Parse(t.Context(),
					[]byte(`<?xml version="1.0"?><!DOCTYPE root [<!NOTATION n SYSTEM "">]><root/>`))
				require.NoError(t, err)
			})

			t.Run("colon in name rejected", func(t *testing.T) {
				t.Parallel()
				// A NotationDecl Name is a non-namespaced Name; a colon is forbidden
				// under namespace processing (W3C not-wf rmt-ns10-044).
				_, err := helium.NewParser().Parse(t.Context(),
					[]byte(`<?xml version="1.0"?><!DOCTYPE root [<!NOTATION a:b SYSTEM "n.dtd">]><root/>`))
				require.Error(t, err)
				require.Contains(t, err.Error(), "colons are forbidden from notation names")
			})
		})

		// EntityDecl [73] EntityDef / [74] PEDef: EntityValue | ExternalID(...).
		// A declaration with neither is fatal (W3C o-p73fail4).
		t.Run("entity", func(t *testing.T) {
			t.Parallel()

			t.Run("general missing value/ExternalID rejected", func(t *testing.T) {
				t.Parallel()
				_, err := helium.NewParser().Parse(t.Context(),
					[]byte(`<?xml version="1.0"?><!DOCTYPE root [<!ENTITY ge >]><root/>`))
				require.ErrorIs(t, err, helium.ErrValueRequired)
			})

			t.Run("parameter missing value/ExternalID rejected", func(t *testing.T) {
				t.Parallel()
				_, err := helium.NewParser().Parse(t.Context(),
					[]byte(`<?xml version="1.0"?><!DOCTYPE root [<!ENTITY % pe >]><root/>`))
				require.ErrorIs(t, err, helium.ErrValueRequired)
			})

			t.Run("general EntityValue parses", func(t *testing.T) {
				t.Parallel()
				_, err := helium.NewParser().Parse(t.Context(),
					[]byte(`<?xml version="1.0"?><!DOCTYPE root [<!ENTITY ge "value">]><root/>`))
				require.NoError(t, err)
			})

			t.Run("general ExternalID parses", func(t *testing.T) {
				t.Parallel()
				// Declared but not referenced, so the external resource is never
				// loaded; the declaration alone must be accepted.
				_, err := helium.NewParser().Parse(t.Context(),
					[]byte(`<?xml version="1.0"?><!DOCTYPE root [<!ENTITY ge SYSTEM "ge.ent">]><root/>`))
				require.NoError(t, err)
			})

			t.Run("general SYSTEM empty literal parses", func(t *testing.T) {
				t.Parallel()
				_, err := helium.NewParser().Parse(t.Context(),
					[]byte(`<?xml version="1.0"?><!DOCTYPE root [<!ENTITY ge SYSTEM "">]><root/>`))
				require.NoError(t, err)
			})

			t.Run("parameter SYSTEM empty literal registers", func(t *testing.T) {
				t.Parallel()
				// A present-but-empty SystemLiteral is a valid ExternalID, so the PE
				// must be registered even though its literal is empty.
				doc, err := helium.NewParser().Parse(t.Context(),
					[]byte(`<?xml version="1.0"?><!DOCTYPE root [<!ENTITY % pe SYSTEM "">]><root/>`))
				require.NoError(t, err)
				_, ok := doc.GetParameterEntity("pe")
				require.True(t, ok, "external PE with empty SystemLiteral must register")
			})
		})

		// EncodingDecl [80]: S 'encoding' Eq ('"' EncName '"' | "'" EncName "'").
		// A present "encoding" keyword with no EncName is fatal (W3C
		// ibm-not-wf-P80-ibm80n03); an absent keyword is benign.
		t.Run("encoding", func(t *testing.T) {
			t.Parallel()

			t.Run("missing EncName (no quote) rejected", func(t *testing.T) {
				t.Parallel()
				_, err := helium.NewParser().Parse(t.Context(),
					[]byte(`<?xml version="1.0" encoding= ?><root/>`))
				require.Error(t, err)
			})

			t.Run("empty EncName rejected", func(t *testing.T) {
				t.Parallel()
				_, err := helium.NewParser().Parse(t.Context(),
					[]byte(`<?xml version="1.0" encoding=""?><root/>`))
				require.Error(t, err)
			})

			t.Run("valid EncName parses", func(t *testing.T) {
				t.Parallel()
				_, err := helium.NewParser().Parse(t.Context(),
					[]byte(`<?xml version="1.0" encoding="UTF-8"?><root/>`))
				require.NoError(t, err)
			})

			t.Run("absent encoding with standalone parses", func(t *testing.T) {
				t.Parallel()
				// No encoding keyword at all: the benign AttrNotFoundError must fall
				// through to the optional StandaloneDecl, not become fatal.
				_, err := helium.NewParser().Parse(t.Context(),
					[]byte(`<?xml version="1.0" standalone="yes"?><root/>`))
				require.NoError(t, err)
			})

			// The UTF-16 XMLDecl is parsed on the rune cursor
			// (parseEncodingDeclFromCursor), which must enforce EncName [81] just like
			// the byte path.
			t.Run("UTF-16 empty EncName rejected", func(t *testing.T) {
				t.Parallel()
				doc := utf16Bytes(`<?xml version="1.0" encoding=""?><root/>`, true)
				_, err := helium.NewParser().Parse(t.Context(), doc)
				require.Error(t, err)
			})

			t.Run("UTF-16 invalid EncName rejected", func(t *testing.T) {
				t.Parallel()
				// "1bad" violates EncName [81] (must start with a letter).
				doc := utf16Bytes(`<?xml version="1.0" encoding="1bad"?><root/>`, true)
				_, err := helium.NewParser().Parse(t.Context(), doc)
				require.Error(t, err)
			})

			t.Run("UTF-16 valid EncName parses", func(t *testing.T) {
				t.Parallel()
				doc := utf16Bytes(`<?xml version="1.0" encoding="UTF-16"?><root/>`, true)
				_, err := helium.NewParser().Parse(t.Context(), doc)
				require.NoError(t, err)
			})
		})
	})
}

func TestLenientXMLDecl(t *testing.T) {
	t.Parallel()

	// the LenientXMLDecl parse path, including
	// pseudo-attributes presented out of the canonical order.
	t.Run("relaxed order", func(t *testing.T) {
		inputs := []string{
			`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><root/>`,
			`<?xml encoding="UTF-8" version="1.0"?><root/>`,
			`<?xml standalone="no" version="1.0"?><root/>`,
			`<?xml version="1.0"?><root/>`,
		}
		for _, in := range inputs {
			doc, err := helium.NewParser().LenientXMLDecl(true).Parse(t.Context(), []byte(in))
			require.NoError(t, err, "lenient parse of %q", in)
			require.NotNil(t, doc.DocumentElement())
		}
	})

	t.Run("parse", func(t *testing.T) {
		const content = `<root />`

		tests := []struct {
			name       string
			input      string
			version    string
			encoding   string
			standalone helium.DocumentStandaloneType
		}{
			{
				name:       "standard order: version encoding standalone",
				input:      `<?xml version="1.0" encoding="utf-8" standalone="yes"?>` + content,
				version:    lexicon.XSLTVersion10,
				encoding:   "utf-8",
				standalone: helium.StandaloneExplicitYes,
			},
			{
				name:       "encoding before version",
				input:      `<?xml encoding="utf-8" version="1.0"?>` + content,
				version:    lexicon.XSLTVersion10,
				encoding:   "utf-8",
				standalone: helium.StandaloneImplicitNo,
			},
			{
				name:       "standalone before version",
				input:      `<?xml standalone="no" version="1.0"?>` + content,
				version:    lexicon.XSLTVersion10,
				encoding:   "",
				standalone: helium.StandaloneExplicitNo,
			},
			{
				name:       "encoding standalone version",
				input:      `<?xml encoding="euc-jp" standalone="yes" version="1.0"?>` + content,
				version:    lexicon.XSLTVersion10,
				encoding:   "euc-jp",
				standalone: helium.StandaloneExplicitYes,
			},
			{
				name:       "standalone version encoding",
				input:      `<?xml standalone="no" version="1.1" encoding="cp932"?>` + content,
				version:    "1.1",
				encoding:   "cp932",
				standalone: helium.StandaloneExplicitNo,
			},
			{
				name:       "version only",
				input:      `<?xml version="1.0"?>` + content,
				version:    lexicon.XSLTVersion10,
				encoding:   "",
				standalone: helium.StandaloneImplicitNo,
			},
			{
				name:       "encoding only (no version)",
				input:      `<?xml encoding="utf-8"?>` + content,
				version:    "",
				encoding:   "utf-8",
				standalone: helium.StandaloneImplicitNo,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				p := helium.NewParser().LenientXMLDecl(true)
				doc, err := p.Parse(t.Context(), []byte(tt.input))
				require.NoError(t, err, "Parse should succeed")
				require.Equal(t, tt.version, doc.Version(), "version")
				if tt.encoding != "" {
					require.Equal(t, tt.encoding, doc.Encoding(), "encoding")
				}
				require.Equal(t, int(tt.standalone), int(doc.Standalone()), "standalone")
			})
		}
	})

	t.Run("rejected without the flag", func(t *testing.T) {
		input := `<?xml encoding="utf-8" version="1.0"?><root />`
		p := helium.NewParser()
		_, err := p.Parse(t.Context(), []byte(input))
		require.Error(t, err, "strict parser should reject encoding before version")
	})
}

func TestEncodingDetection(t *testing.T) {
	t.Parallel()

	t.Run("BOM detection", func(t *testing.T) {
		tests := []struct {
			name    string
			input   []byte
			wantErr bool
		}{
			{
				name:    "utf8 xml declaration",
				input:   []byte(`<?xml version="1.0"?><root/>`),
				wantErr: false,
			},
			{
				name:    "utf8 bom",
				input:   append([]byte{0xEF, 0xBB, 0xBF}, []byte(`<?xml version="1.0"?><root/>`)...),
				wantErr: false,
			},
			{
				name:    "invalid bytes",
				input:   []byte{0xde, 0xad, 0xbe, 0xef},
				wantErr: true,
			},
		}

		for _, tt := range tests {
			p := helium.NewParser()
			_, err := p.Parse(t.Context(), tt.input)
			if tt.wantErr {
				require.Error(t, err, tt.name)
				continue
			}
			require.NoError(t, err, tt.name)
		}
	})

	// XML §4.3.3: a byte-order mark asserts the
	// entity's encoding, so a declaration naming a contradicting encoding is a
	// fatal well-formedness error (W3C xml suite hst-lhs-007, hst-lhs-008). Each
	// contradicting document must be fatal, while the well-formed near-misses (a
	// matching declaration, a BOM alias, and — crucially — a BOM-less document
	// declaring a single-byte encoding) must still parse, guarding against
	// over-rejection.
	t.Run("BOM conflicts with the declared encoding", func(t *testing.T) {
		bomUTF8 := []byte{0xEF, 0xBB, 0xBF}
		bomUTF16BE := []byte{0xFE, 0xFF}
		bomUTF16LE := []byte{0xFF, 0xFE}

		t.Run("rejected", func(t *testing.T) {
			t.Parallel()

			t.Run("utf-8 BOM with iso-8859-1 declaration", func(t *testing.T) {
				t.Parallel()
				src := append(append([]byte{}, bomUTF8...),
					[]byte(`<?xml version='1.0' encoding='iso-8859-1'?><x/>`)...)
				_, err := helium.NewParser().Parse(t.Context(), src)
				require.ErrorIs(t, err, helium.ErrEncodingBOMMismatch)
			})

			t.Run("utf-16be BOM with utf-8 declaration", func(t *testing.T) {
				t.Parallel()
				src := append(append([]byte{}, bomUTF16BE...),
					utf16be(`<?xml version='1.0' encoding='utf-8'?><x/>`)...)
				_, err := helium.NewParser().Parse(t.Context(), src)
				require.ErrorIs(t, err, helium.ErrEncodingBOMMismatch)
			})

			t.Run("utf-16le BOM with utf-8 declaration", func(t *testing.T) {
				t.Parallel()
				src := append(append([]byte{}, bomUTF16LE...),
					utf16le(`<?xml version='1.0' encoding='utf-8'?><x/>`)...)
				_, err := helium.NewParser().Parse(t.Context(), src)
				require.ErrorIs(t, err, helium.ErrEncodingBOMMismatch)
			})

			// An explicit OPPOSITE-endian UTF-16 declaration is a definite mismatch
			// (only a generic "utf-16" is compatible with either BOM).
			t.Run("utf-16be BOM with explicit utf-16le declaration", func(t *testing.T) {
				t.Parallel()
				src := append(append([]byte{}, bomUTF16BE...),
					utf16be(`<?xml version='1.0' encoding='UTF-16LE'?><x/>`)...)
				_, err := helium.NewParser().Parse(t.Context(), src)
				require.ErrorIs(t, err, helium.ErrEncodingBOMMismatch)
			})

			t.Run("utf-16le BOM with explicit utf-16be declaration", func(t *testing.T) {
				t.Parallel()
				src := append(append([]byte{}, bomUTF16LE...),
					utf16le(`<?xml version='1.0' encoding='UTF-16BE'?><x/>`)...)
				_, err := helium.NewParser().Parse(t.Context(), src)
				require.ErrorIs(t, err, helium.ErrEncodingBOMMismatch)
			})
		})

		t.Run("accepted", func(t *testing.T) {
			t.Parallel()

			t.Run("utf-8 BOM with matching declaration", func(t *testing.T) {
				t.Parallel()
				src := append(append([]byte{}, bomUTF8...),
					[]byte(`<?xml version='1.0' encoding='UTF-8'?><x/>`)...)
				_, err := helium.NewParser().Parse(t.Context(), src)
				require.NoError(t, err)
			})

			t.Run("utf-8 BOM with no encoding declaration", func(t *testing.T) {
				t.Parallel()
				src := append(append([]byte{}, bomUTF8...),
					[]byte(`<?xml version='1.0'?><x/>`)...)
				_, err := helium.NewParser().Parse(t.Context(), src)
				require.NoError(t, err)
			})

			t.Run("utf-16be BOM with utf-16 alias", func(t *testing.T) {
				t.Parallel()
				src := append(append([]byte{}, bomUTF16BE...),
					utf16be(`<?xml version='1.0' encoding='UTF-16'?><x/>`)...)
				_, err := helium.NewParser().Parse(t.Context(), src)
				require.NoError(t, err)
			})

			t.Run("utf-16be BOM with utf-16be declaration", func(t *testing.T) {
				t.Parallel()
				src := append(append([]byte{}, bomUTF16BE...),
					utf16be(`<?xml version='1.0' encoding='UTF-16BE'?><x/>`)...)
				_, err := helium.NewParser().Parse(t.Context(), src)
				require.NoError(t, err)
			})

			// The hyphen-less endian alias (UTF16BE) is a name internal/encoding.Load
			// and xmllint both accept, so a BOM declaring it must NOT be a conflict.
			t.Run("utf-16be BOM with hyphenless utf16be declaration", func(t *testing.T) {
				t.Parallel()
				src := append(append([]byte{}, bomUTF16BE...),
					utf16be(`<?xml version='1.0' encoding='UTF16BE'?><x/>`)...)
				_, err := helium.NewParser().Parse(t.Context(), src)
				require.NoError(t, err)
			})

			// Every alias internal/encoding.Load resolves to the BOM's Unicode
			// family must parse — the check canonicalizes through Load, so it is not
			// limited to a hand-listed set. These are aliases earlier review rounds
			// found missing from the retired parallel table.
			t.Run("utf-8 BOM with unicode-1-1-utf-8 alias", func(t *testing.T) {
				t.Parallel()
				src := append(append([]byte{}, bomUTF8...),
					[]byte(`<?xml version='1.0' encoding='unicode-1-1-utf-8'?><x/>`)...)
				_, err := helium.NewParser().Parse(t.Context(), src)
				require.NoError(t, err)
			})

			t.Run("utf-16be BOM with unicodeFFFE alias", func(t *testing.T) {
				t.Parallel()
				src := append(append([]byte{}, bomUTF16BE...),
					utf16be(`<?xml version='1.0' encoding='unicodeFFFE'?><x/>`)...)
				_, err := helium.NewParser().Parse(t.Context(), src)
				require.NoError(t, err)
			})

			t.Run("utf-16le BOM with unicodeFEFF alias", func(t *testing.T) {
				t.Parallel()
				src := append(append([]byte{}, bomUTF16LE...),
					utf16le(`<?xml version='1.0' encoding='unicodeFEFF'?><x/>`)...)
				_, err := helium.NewParser().Parse(t.Context(), src)
				require.NoError(t, err)
			})

			// A generic "utf-16" / "unicode" / "csUnicode" declaration carries no
			// endianness, so it matches EITHER UTF-16 BOM.
			t.Run("utf-16le BOM with generic unicode alias", func(t *testing.T) {
				t.Parallel()
				src := append(append([]byte{}, bomUTF16LE...),
					utf16le(`<?xml version='1.0' encoding='unicode'?><x/>`)...)
				_, err := helium.NewParser().Parse(t.Context(), src)
				require.NoError(t, err)
			})

			t.Run("utf-16be BOM with csUnicode alias", func(t *testing.T) {
				t.Parallel()
				src := append(append([]byte{}, bomUTF16BE...),
					utf16be(`<?xml version='1.0' encoding='csUnicode'?><x/>`)...)
				_, err := helium.NewParser().Parse(t.Context(), src)
				require.NoError(t, err)
			})

			// The key over-rejection guard: a BOM-less, ASCII-compatible document
			// that declares a single-byte encoding must NOT be treated as a
			// conflict — no BOM was consumed, so autoEncoding is empty.
			t.Run("no BOM with iso-8859-1 declaration", func(t *testing.T) {
				t.Parallel()
				_, err := helium.NewParser().Parse(t.Context(),
					[]byte(`<?xml version='1.0' encoding='iso-8859-1'?><x/>`))
				require.NoError(t, err)
			})
		})

		// IgnoreEncoding(true) suppresses the decoder switch but must NOT suppress
		// the BOM/encoding-mismatch well-formedness check — the declared encoding is
		// still recorded for the check even though ctx.encoding is erased.
		t.Run("ignore-encoding", func(t *testing.T) {
			t.Parallel()

			t.Run("utf-8 BOM with iso-8859-1 declaration still fatal", func(t *testing.T) {
				t.Parallel()
				src := append(append([]byte{}, bomUTF8...),
					[]byte(`<?xml version='1.0' encoding='iso-8859-1'?><x/>`)...)
				_, err := helium.NewParser().IgnoreEncoding(true).Parse(t.Context(), src)
				require.ErrorIs(t, err, helium.ErrEncodingBOMMismatch)
			})

			t.Run("utf-8 BOM with matching declaration parses", func(t *testing.T) {
				t.Parallel()
				src := append(append([]byte{}, bomUTF8...),
					[]byte(`<?xml version='1.0' encoding='UTF-8'?><x/>`)...)
				_, err := helium.NewParser().IgnoreEncoding(true).Parse(t.Context(), src)
				require.NoError(t, err)
			})

			t.Run("no BOM with iso-8859-1 declaration parses", func(t *testing.T) {
				t.Parallel()
				_, err := helium.NewParser().IgnoreEncoding(true).Parse(t.Context(),
					[]byte(`<?xml version='1.0' encoding='iso-8859-1'?><x/>`))
				require.NoError(t, err)
			})
		})

		// LenientXMLDecl(true) relaxes declaration parsing but must NOT suppress the
		// BOM/encoding-mismatch check. The declared EncName is recorded at the leaf
		// EncName parser, so it is authoritative on the lenient path too. The check
		// must also hold when LenientXMLDecl and IgnoreEncoding are combined.
		t.Run("lenient-decl", func(t *testing.T) {
			t.Parallel()

			t.Run("utf-8 BOM with iso-8859-1 declaration still fatal", func(t *testing.T) {
				t.Parallel()
				src := append(append([]byte{}, bomUTF8...),
					[]byte(`<?xml version='1.0' encoding='iso-8859-1'?><x/>`)...)
				_, err := helium.NewParser().LenientXMLDecl(true).Parse(t.Context(), src)
				require.ErrorIs(t, err, helium.ErrEncodingBOMMismatch)
			})

			t.Run("utf-16be BOM with utf-8 declaration still fatal", func(t *testing.T) {
				t.Parallel()
				src := append(append([]byte{}, bomUTF16BE...),
					utf16be(`<?xml version='1.0' encoding='utf-8'?><x/>`)...)
				_, err := helium.NewParser().LenientXMLDecl(true).Parse(t.Context(), src)
				require.ErrorIs(t, err, helium.ErrEncodingBOMMismatch)
			})

			t.Run("lenient plus ignore-encoding still fatal", func(t *testing.T) {
				t.Parallel()
				src := append(append([]byte{}, bomUTF8...),
					[]byte(`<?xml version='1.0' encoding='iso-8859-1'?><x/>`)...)
				_, err := helium.NewParser().
					LenientXMLDecl(true).IgnoreEncoding(true).Parse(t.Context(), src)
				require.ErrorIs(t, err, helium.ErrEncodingBOMMismatch)
			})

			// The combined knobs must not over-reject: a matching BOM+declaration
			// pair and a BOM-less document still parse under LenientXMLDecl+IgnoreEncoding.
			t.Run("lenient plus ignore-encoding matching declaration parses", func(t *testing.T) {
				t.Parallel()
				src := append(append([]byte{}, bomUTF8...),
					[]byte(`<?xml version='1.0' encoding='UTF-8'?><x/>`)...)
				_, err := helium.NewParser().
					LenientXMLDecl(true).IgnoreEncoding(true).Parse(t.Context(), src)
				require.NoError(t, err)
			})

			t.Run("lenient plus ignore-encoding no BOM parses", func(t *testing.T) {
				t.Parallel()
				_, err := helium.NewParser().
					LenientXMLDecl(true).IgnoreEncoding(true).Parse(t.Context(),
					[]byte(`<?xml version='1.0' encoding='iso-8859-1'?><x/>`))
				require.NoError(t, err)
			})

			t.Run("utf-8 BOM with matching declaration parses", func(t *testing.T) {
				t.Parallel()
				src := append(append([]byte{}, bomUTF8...),
					[]byte(`<?xml version='1.0' encoding='UTF-8'?><x/>`)...)
				_, err := helium.NewParser().LenientXMLDecl(true).Parse(t.Context(), src)
				require.NoError(t, err)
			})

			t.Run("no BOM with iso-8859-1 declaration parses", func(t *testing.T) {
				t.Parallel()
				_, err := helium.NewParser().LenientXMLDecl(true).Parse(t.Context(),
					[]byte(`<?xml version='1.0' encoding='iso-8859-1'?><x/>`))
				require.NoError(t, err)
			})
		})
	})

	// is a regression for E-UCS4-CONSUMES-LT: the
	// encoding auto-detector used to CONSUME the four leading bytes during UCS-4
	// detection. Those bytes are the encoded first '<' character (not a BOM), so a
	// genuine UCS-4 document lost its leading '<' and failed with "start tag
	// expected, '<' not found". Detection must peek, not consume.
	t.Run("UCS-4 first byte is not consumed", func(t *testing.T) {
		const doc = `<?xml version="1.0" encoding="ISO-10646-UCS-4"?><root>hi</root>`

		// byte orders matching the detector patterns:
		//   BE   : 00 00 00 3C  -> shifts 24,16,8,0
		//   LE   : 3C 00 00 00  -> shifts 0,8,16,24
		//   2143 : 00 00 3C 00  -> shifts 16,24,0,8
		//   3412 : 00 3C 00 00  -> shifts 8,0,24,16
		orders := map[string][4]uint{
			"BE":   {24, 16, 8, 0},
			"LE":   {0, 8, 16, 24},
			"2143": {16, 24, 0, 8},
			"3412": {8, 0, 24, 16},
		}

		for name, order := range orders {
			t.Run(name, func(t *testing.T) {
				in := encodeUCS4(doc, order)

				// Sanity: the first encoded code point is '<', and its four bytes
				// must contain exactly one 0x3C so this really exercises a
				// UCS-4-looking leading sequence.
				require.Len(t, in[:4], 4)

				parsed, err := helium.NewParser().Parse(t.Context(), in)
				require.NoError(t, err, "genuine UCS-4 document must parse with its first '<' intact")

				root := parsed.DocumentElement()
				require.NotNil(t, root, "document element must be present")
				require.Equal(t, "root", root.Name())
				require.Equal(t, "hi", string(root.Content()))
			})
		}
	})

	// is a regression for the UCS-4 fixed-width
	// prolog probe misclassifying a legal leading PI as an XML declaration. The
	// probe used HasPrefixString("<?xml"), which matches "<?xml-stylesheet ...?>"
	// and then failed with "blank needed after '<?xml'". The probe must use
	// looksLikeXMLDeclString to distinguish "<?xml " (decl) from "<?xml-stylesheet"
	// (PI), so a UCS-4 document beginning with such a PI parses with the PI intact.
	t.Run("UCS-4 leading stylesheet PI", func(t *testing.T) {
		const doc = `<?xml-stylesheet type="text/xsl" href="x.xsl"?><root>hi</root>`

		orders := map[string][4]uint{
			"BE":   {24, 16, 8, 0},
			"LE":   {0, 8, 16, 24},
			"2143": {16, 24, 0, 8},
			"3412": {8, 0, 24, 16},
		}

		for name, order := range orders {
			t.Run(name, func(t *testing.T) {
				in := encodeUCS4(doc, order)

				parsed, err := helium.NewParser().Parse(t.Context(), in)
				require.NoError(t, err, "UCS-4 document with a leading PI must not be misread as an XML declaration")

				var pi *helium.ProcessingInstruction
				for c := parsed.FirstChild(); c != nil; c = c.NextSibling() {
					if got, ok := helium.AsNode[*helium.ProcessingInstruction](c); ok {
						pi = got
						break
					}
				}
				require.NotNil(t, pi, "leading processing instruction must be preserved")
				require.Equal(t, "xml-stylesheet", pi.Name())
				require.Equal(t, `type="text/xsl" href="x.xsl"`, string(pi.Content()))

				root := parsed.DocumentElement()
				require.NotNil(t, root, "document element must be present")
				require.Equal(t, "root", root.Name())
				require.Equal(t, "hi", string(root.Content()))
			})
		}
	})

	t.Run("UTF-16 unpaired surrogate", func(t *testing.T) {
		const unpairedHigh = uint16(0xD800) // lone high surrogate: malformed input

		t.Run("unpaired surrogate in text content is rejected", func(t *testing.T) {
			doc := utf16beDoc("<r>", unpairedHigh, "</r>")
			_, err := helium.NewParser().Parse(t.Context(), doc)
			require.Error(t, err, "unpaired surrogate in text content must be a fatal error")
		})

		t.Run("unpaired surrogate in attribute value is rejected", func(t *testing.T) {
			doc := utf16beDoc(`<r a="`, unpairedHigh, `"/>`)
			_, err := helium.NewParser().Parse(t.Context(), doc)
			require.Error(t, err, "unpaired surrogate in attribute value must be a fatal error")
		})

		t.Run("genuine U+FFFD in text content is accepted", func(t *testing.T) {
			doc := utf16beDoc("<r>�</r>")
			parsed, err := helium.NewParser().Parse(t.Context(), doc)
			require.NoError(t, err, "genuine U+FFFD is a valid XML character")
			root := parsed.DocumentElement()
			require.NotNil(t, root)
			require.Equal(t, "�", string(root.Content()))
		})

		t.Run("genuine U+FFFD in attribute value is accepted", func(t *testing.T) {
			doc := utf16beDoc("<r a=\"�\"/>")
			parsed, err := helium.NewParser().Parse(t.Context(), doc)
			require.NoError(t, err, "genuine U+FFFD is a valid XML character")
			require.NotNil(t, parsed.DocumentElement())
		})
	})
}

func TestEncodingDeclaration(t *testing.T) {
	t.Parallel()

	// parses documents with various encoding declarations
	// to exercise the encoding-switch paths.
	t.Run("declarations", func(t *testing.T) {
		inputs := []string{
			`<?xml version="1.0" encoding="UTF-8"?><root>ascii</root>`,
			`<?xml version="1.0" encoding="utf-8"?><root>ascii</root>`,
			`<?xml version="1.0" encoding="US-ASCII"?><root>ascii</root>`,
		}
		for _, in := range inputs {
			doc, err := helium.NewParser().Parse(t.Context(), []byte(in))
			require.NoError(t, err, "parse %q", in)
			require.NotNil(t, doc.DocumentElement())
		}
	})

	// the IgnoreEncoding option does not break a parse
	// that declares an encoding.
	t.Run("ignored encoding", func(t *testing.T) {
		const in = `<?xml version="1.0" encoding="ISO-8859-1"?><root>x</root>`
		doc, err := helium.NewParser().IgnoreEncoding(true).Parse(t.Context(), []byte(in))
		require.NoError(t, err)
		require.NotNil(t, doc.DocumentElement())
	})

	t.Run("strict US-ASCII", func(t *testing.T) {
		// A document declaring US-ASCII is strictly 7-bit; a byte >= 0x80 is
		// malformed even when it forms a valid UTF-8 multibyte sequence.
		highByte := "<?xml version=\"1.0\" encoding=\"US-ASCII\"?><root>\xc3\xa9</root>"
		_, err := helium.NewParser().Parse(t.Context(), []byte(highByte))
		require.Error(t, err, "US-ASCII document with a byte >= 0x80 must be rejected")

		// Valid 7-bit US-ASCII still parses.
		valid := `<?xml version="1.0" encoding="US-ASCII"?><root>hello</root>`
		_, err = helium.NewParser().Parse(t.Context(), []byte(valid))
		require.NoError(t, err, "valid 7-bit US-ASCII must parse")
	})
}
