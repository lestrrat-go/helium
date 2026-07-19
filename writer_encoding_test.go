package helium_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/internal/writerctl"
	"github.com/stretchr/testify/require"
)

// requireASCIIRejected asserts that a US-ASCII serialization failed with
// ErrUnsupportedOutputEncoding and that no raw non-ASCII octet leaked into buf
// before the error fired.
func requireASCIIRejected(t *testing.T, buf *bytes.Buffer, err error) {
	t.Helper()
	require.ErrorIs(t, err, helium.ErrUnsupportedOutputEncoding)
	for i := range buf.Len() {
		require.Less(t, buf.Bytes()[i], byte(0x80), "leaked non-ASCII octet 0x%X at %d", buf.Bytes()[i], i)
	}
}

// TestOutputEncodingMatchesEmittedBytes asserts that when OutputEncoding
// overrides the declaration to a transcodable encoding, the emitted octets are
// re-encoded to match — the declaration and the bytes agree. With
// EscapeNonASCII disabled a non-ASCII character is written literally, so the
// mismatch (raw UTF-8 octets under a Latin-1 declaration) is observable.
func TestOutputEncodingMatchesEmittedBytes(t *testing.T) {
	t.Parallel()

	src := []byte(`<?xml version="1.0" encoding="UTF-8"?><root>é</root>`)
	doc, err := helium.NewParser().Parse(t.Context(), src)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = helium.NewWriter().EscapeNonASCII(false).OutputEncoding("ISO-8859-1").WriteTo(&buf, doc)
	require.NoError(t, err)

	out := buf.String()
	require.Contains(t, out, `encoding="ISO-8859-1"`)
	// U+00E9 must be the single ISO-8859-1 byte 0xE9, NOT the two UTF-8 bytes
	// 0xC3 0xA9 that a Latin-1 declaration would misdescribe.
	require.Contains(t, out, "\xe9")
	require.NotContains(t, out, "\xc3\xa9")
}

// TestOutputEncodingUnsupportedErrors asserts that an explicitly set
// OutputEncoding naming an encoding the writer cannot emit is a hard error,
// rather than silently emitting UTF-8 under a false declaration.
func TestOutputEncodingUnsupportedErrors(t *testing.T) {
	t.Parallel()

	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root/>`))
	require.NoError(t, err)

	var buf bytes.Buffer
	err = helium.NewWriter().OutputEncoding("no-such-enc").WriteTo(&buf, doc)
	require.ErrorIs(t, err, helium.ErrUnsupportedOutputEncoding)
}

// TestOutputEncodingUSASCIIEscapesNonASCII asserts that an explicit US-ASCII
// OutputEncoding override escapes every non-ASCII character as a numeric
// character reference rather than emitting raw UTF-8 under a US-ASCII
// declaration. US-ASCII can represent any document via character references, so
// this is not an unsupported-encoding error; the result is valid US-ASCII and
// reparses to the original content.
func TestOutputEncodingUSASCIIEscapesNonASCII(t *testing.T) {
	t.Parallel()

	// U+2603 (BMP, beyond Latin-1) and U+00E9 (Latin-1) together confirm the full
	// non-ASCII range is escaped, not just Latin-1.
	src := []byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?><root a=\"é\">☃</root>")
	doc, err := helium.NewParser().Parse(t.Context(), src)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = helium.NewWriter().OutputEncoding("US-ASCII").WriteTo(&buf, doc)
	require.NoError(t, err)

	out := buf.String()
	require.Contains(t, out, `encoding="US-ASCII"`)
	require.Contains(t, out, "&#x2603;")
	require.Contains(t, out, "&#xE9;")
	// No raw non-ASCII octet may appear under the US-ASCII declaration.
	for i := range len(out) {
		require.Less(t, out[i], byte(0x80), "non-ASCII octet 0x%X at %d in %q", out[i], i, out)
	}

	// The escaped output reparses, and re-serializing (default UTF-8) recovers the
	// original non-ASCII characters — the character references round-trip.
	rt, err := helium.NewParser().Parse(t.Context(), buf.Bytes())
	require.NoError(t, err)
	var rtbuf bytes.Buffer
	require.NoError(t, helium.NewWriter().EscapeNonASCII(false).WriteTo(&rtbuf, rt))
	require.Contains(t, rtbuf.String(), "☃")
	require.Contains(t, rtbuf.String(), "é")
}

// TestOutputEncodingUSASCIIRejectsNonCharRefContexts asserts that under an
// explicit US-ASCII OutputEncoding a non-ASCII character in a context that
// cannot hold a character reference — comment text, CDATA section, PI
// target/data, an element/attribute name — fails with
// ErrUnsupportedOutputEncoding rather than emitting raw UTF-8 under a US-ASCII
// declaration.
func TestOutputEncodingUSASCIIRejectsNonCharRefContexts(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"comment":        `<?xml version="1.0" encoding="UTF-8"?><root><!--é--></root>`,
		"cdata":          `<?xml version="1.0" encoding="UTF-8"?><root><![CDATA[é]]></root>`,
		"pi-data":        `<?xml version="1.0" encoding="UTF-8"?><root><?pi dáta?></root>`,
		"element-name":   `<?xml version="1.0" encoding="UTF-8"?><rôot></rôot>`,
		"attribute-name": `<?xml version="1.0" encoding="UTF-8"?><root ättr="v"></root>`,
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
			require.NoError(t, err)

			var buf bytes.Buffer
			err = helium.NewWriter().OutputEncoding("US-ASCII").WriteTo(&buf, doc)
			require.ErrorIs(t, err, helium.ErrUnsupportedOutputEncoding)
			// No raw non-ASCII octet may have leaked before the error fired.
			for i := range buf.Len() {
				require.Less(t, buf.Bytes()[i], byte(0x80), "leaked non-ASCII octet 0x%X", buf.Bytes()[i])
			}
		})
	}
}

// TestOutputEncodingASCIIAliasesMatchUSASCII asserts that every registered
// US-ASCII alias takes the same char-referencing path as the canonical
// "US-ASCII" name, rather than emitting raw UTF-8 because the loaded encoder
// delegates to UTF-8.
func TestOutputEncodingASCIIAliasesMatchUSASCII(t *testing.T) {
	t.Parallel()

	src := []byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?><root a=\"é\">☃</root>")
	doc, err := helium.NewParser().Parse(t.Context(), src)
	require.NoError(t, err)

	serialize := func(t *testing.T, enc string) string {
		t.Helper()
		var buf bytes.Buffer
		require.NoError(t, helium.NewWriter().OutputEncoding(enc).WriteTo(&buf, doc))
		return buf.String()
	}

	// The canonical form, minus the declaration line (which carries the differing
	// encoding label), is the body every alias must reproduce.
	canonical := serialize(t, "US-ASCII")
	canonicalBody := canonical[strings.IndexByte(canonical, '\n'):]

	for _, alias := range []string{"csASCII", "ANSI_X3.4-1968", "ANSI_X3.4-1986", "iso-ir-6", "ISO646-US", "us", "ascii"} {
		out := serialize(t, alias)
		require.Contains(t, out, `encoding="`+alias+`"`)
		require.Contains(t, out, "&#x2603;")
		require.Contains(t, out, "&#xE9;")
		for i := range len(out) {
			require.Less(t, out[i], byte(0x80), "alias %q leaked non-ASCII octet 0x%X", alias, out[i])
		}
		require.Equal(t, canonicalBody, out[strings.IndexByte(out, '\n'):], "alias %q body differs from US-ASCII", alias)
	}
}

// TestOutputEncodingElementFragmentUnchanged asserts OutputEncoding affects the
// Document path only: serializing a bare element (a fragment, which carries no
// XML declaration) with OutputEncoding("US-ASCII") is byte-identical to
// serializing it with no override. A character above Latin-1 (which the default
// escapeNonASCII does not touch) makes any leaked US-ASCII escaping observable.
func TestOutputEncodingElementFragmentUnchanged(t *testing.T) {
	t.Parallel()

	src := []byte(`<?xml version="1.0" encoding="UTF-8"?><root>世</root>`)
	doc, err := helium.NewParser().Parse(t.Context(), src)
	require.NoError(t, err)
	root := doc.DocumentElement()

	var plain, ascii bytes.Buffer
	require.NoError(t, helium.NewWriter().WriteTo(&plain, root))
	require.NoError(t, helium.NewWriter().OutputEncoding("US-ASCII").WriteTo(&ascii, root))
	require.Equal(t, plain.String(), ascii.String())
	require.Contains(t, plain.String(), "世")
}

// TestSerializeNoEncodingOverrideUnchanged asserts the no-override path is
// byte-identical to prior behavior: default escaping still emits a hex
// character reference under the document's own declaration, and an unloadable
// parsed encoding is NOT turned into a hard error when no override is set.
func TestSerializeNoEncodingOverrideUnchanged(t *testing.T) {
	t.Parallel()

	src := []byte(`<?xml version="1.0" encoding="UTF-8"?><root>é</root>`)
	doc, err := helium.NewParser().Parse(t.Context(), src)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = helium.NewWriter().WriteTo(&buf, doc)
	require.NoError(t, err)

	out := buf.String()
	require.Contains(t, out, `encoding="UTF-8"`)
	require.Contains(t, out, "&#xE9;")
	require.NotContains(t, out, "\xc3\xa9")

	// An unloadable encoding recorded on the document must still serialize
	// (declaration-only) when there is no OutputEncoding override.
	built := helium.NewDefaultDocument()
	root, err := built.CreateElement("root")
	require.NoError(t, err)
	require.NoError(t, built.SetDocumentElement(root))
	built.SetEncoding("x-unknown-enc")

	var buf2 bytes.Buffer
	err = helium.NewWriter().WriteTo(&buf2, built)
	require.NoError(t, err)
	require.Contains(t, buf2.String(), `encoding="x-unknown-enc"`)
}

// TestOutputEncodingUSASCIIRejectsNonASCIINames asserts that under an explicit
// US-ASCII OutputEncoding a non-ASCII entity-reference name or DTD-internal name
// (DOCTYPE, <!ENTITY>, <!ELEMENT>, <!ATTLIST> element/attribute name, enumeration
// token) fails with ErrUnsupportedOutputEncoding rather than emitting raw UTF-8
// under a US-ASCII declaration, and that no raw non-ASCII octet leaks into the
// buffer before the error fires.
func TestOutputEncodingUSASCIIRejectsNonASCIINames(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"entity-ref-and-decl": `<?xml version="1.0" encoding="UTF-8"?><!DOCTYPE root [<!ENTITY é "x">]><root>&é;</root>`,
		"element-decl-name":   `<?xml version="1.0" encoding="UTF-8"?><!DOCTYPE root [<!ELEMENT rôot ANY>]><root/>`,
		"attlist-attr-name":   `<?xml version="1.0" encoding="UTF-8"?><!DOCTYPE root [<!ATTLIST root ättr CDATA #IMPLIED>]><root/>`,
		"enumeration-token":   `<?xml version="1.0" encoding="UTF-8"?><!DOCTYPE root [<!ATTLIST root a (é|b) #IMPLIED>]><root/>`,
		"doctype-name":        `<?xml version="1.0" encoding="UTF-8"?><!DOCTYPE rôot [<!ELEMENT a ANY>]><rôot/>`,
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
			require.NoError(t, err)

			var buf bytes.Buffer
			err = helium.NewWriter().OutputEncoding("US-ASCII").WriteTo(&buf, doc)
			require.ErrorIs(t, err, helium.ErrUnsupportedOutputEncoding)
			// No raw non-ASCII octet may have leaked before the error fired.
			for i := range buf.Len() {
				require.Less(t, buf.Bytes()[i], byte(0x80), "leaked non-ASCII octet 0x%X", buf.Bytes()[i])
			}
		})
	}
}

// TestOutputEncodingUSASCIIRejectsCharMapReplacement asserts that under an
// explicit US-ASCII OutputEncoding (the octet-producing WriteTo path) a
// character-map replacement string carrying a non-ASCII character fails with a
// labelled ErrUnsupportedOutputEncoding rather than emitting the raw replacement
// verbatim (a character map is never re-escaped, so a non-ASCII replacement would
// leak raw UTF-8 under the US-ASCII declaration). Covers U1 in both text and
// attribute-value content.
func TestOutputEncodingUSASCIIRejectsCharMapReplacement(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"text": `<?xml version="1.0" encoding="UTF-8"?><root>@</root>`,
		"attr": `<?xml version="1.0" encoding="UTF-8"?><root a="@"/>`,
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
			require.NoError(t, err)

			var buf bytes.Buffer
			err = helium.NewWriter().
				CharacterMap(map[rune]string{'@': "€"}). // '@' -> EURO SIGN
				OutputEncoding("US-ASCII").
				WriteTo(&buf, doc)
			requireASCIIRejected(t, &buf, err)
		})
	}
}

// TestOutputEncodingUSASCIIAllowsASCIICharMapReplacement asserts the positive
// case: an ASCII-only character-map replacement under US-ASCII succeeds and the
// replacement is emitted verbatim.
func TestOutputEncodingUSASCIIAllowsASCIICharMapReplacement(t *testing.T) {
	t.Parallel()

	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0" encoding="UTF-8"?><root>@</root>`))
	require.NoError(t, err)

	var buf bytes.Buffer
	err = helium.NewWriter().
		CharacterMap(map[rune]string{'@': "[AT]"}).
		OutputEncoding("US-ASCII").
		WriteTo(&buf, doc)
	require.NoError(t, err)
	require.Contains(t, buf.String(), "<root>[AT]</root>")
}

// TestOutputEncodingUSASCIIRejectsDTDLiterals asserts that under an explicit
// US-ASCII OutputEncoding a non-ASCII value in a reference-less DTD literal —
// DOCTYPE external/system ID, entity external/system ID, internal entity value,
// an NDATA notation name, or a notation external/system ID — fails with
// ErrUnsupportedOutputEncoding without leaking a raw non-ASCII octet. These
// literals write to the output directly and are covered by the ASCII-reject net.
// Covers U3–U7.
func TestOutputEncodingUSASCIIRejectsDTDLiterals(t *testing.T) {
	t.Parallel()

	// nonASCII is a value carrying a non-ASCII character placed at each literal.
	const nonASCII = "café" // "café"

	build := map[string]func(*testing.T) *helium.Document{
		"doctype-system-id": func(t *testing.T) *helium.Document {
			doc := helium.NewDefaultDocument()
			_, err := doc.CreateInternalSubset("root", "", nonASCII+".dtd")
			require.NoError(t, err)
			root, err := doc.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, doc.SetDocumentElement(root))
			return doc
		},
		"doctype-public-id": func(t *testing.T) *helium.Document {
			doc := helium.NewDefaultDocument()
			_, err := doc.CreateInternalSubset("root", "-//"+nonASCII+"//EN", "sys.dtd")
			require.NoError(t, err)
			root, err := doc.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, doc.SetDocumentElement(root))
			return doc
		},
		"entity-external-system-id": func(t *testing.T) *helium.Document {
			doc := helium.NewDefaultDocument()
			dtd, err := doc.CreateInternalSubset("root", "", "")
			require.NoError(t, err)
			_, err = dtd.AddEntity("e", enum.ExternalGeneralParsedEntity, "", nonASCII+".ent", "")
			require.NoError(t, err)
			root, err := doc.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, doc.SetDocumentElement(root))
			return doc
		},
		"entity-internal-value": func(t *testing.T) *helium.Document {
			doc := helium.NewDefaultDocument()
			dtd, err := doc.CreateInternalSubset("root", "", "")
			require.NoError(t, err)
			_, err = dtd.AddEntity("e", enum.InternalGeneralEntity, "", "", nonASCII)
			require.NoError(t, err)
			root, err := doc.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, doc.SetDocumentElement(root))
			return doc
		},
		"ndata-notation-name": func(t *testing.T) *helium.Document {
			doc := helium.NewDefaultDocument()
			dtd, err := doc.CreateInternalSubset("root", "", "")
			require.NoError(t, err)
			// The NDATA notation name (the entity's content) is non-ASCII; the
			// external ID literal is ASCII so the leak is isolated to the name.
			_, err = dtd.AddEntity("e", enum.ExternalGeneralUnparsedEntity, "", "e.dat", nonASCII)
			require.NoError(t, err)
			root, err := doc.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, doc.SetDocumentElement(root))
			return doc
		},
		"notation-system-id": func(t *testing.T) *helium.Document {
			doc := helium.NewDefaultDocument()
			dtd, err := doc.CreateInternalSubset("root", "", "")
			require.NoError(t, err)
			_, err = dtd.AddNotation("n", "", nonASCII+".not")
			require.NoError(t, err)
			root, err := doc.CreateElement("root")
			require.NoError(t, err)
			require.NoError(t, doc.SetDocumentElement(root))
			return doc
		},
	}
	for name, mk := range build {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			doc := mk(t)
			var buf bytes.Buffer
			err := helium.NewWriter().OutputEncoding("US-ASCII").WriteTo(&buf, doc)
			requireASCIIRejected(t, &buf, err)
		})
	}
}

// TestSerializeDeclarationOnlyUSASCIIKeepsRawCharMap asserts that fn:serialize's
// declaration-only US-ASCII mode (asciiOutput set, but the octets stay a UTF-8
// string) keeps a non-ASCII character-map replacement RAW rather than rejecting
// it: the ASCII-reject net and the character-map reject both key on
// asciiOutput && !declOnlyEncoding, so declaration-only output is unchanged. Text
// outside the character map is still char-referenced.
func TestSerializeDeclarationOnlyUSASCIIKeepsRawCharMap(t *testing.T) {
	t.Parallel()

	doc, err := helium.NewParser().Parse(t.Context(), []byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?><root>@é</root>"))
	require.NoError(t, err)

	w := helium.NewWriter().
		CharacterMap(map[rune]string{'@': "€"}). // '@' -> EURO SIGN
		OutputEncoding("US-ASCII")
	declOnly, ok := writerctl.EnableDeclarationOnlyEncoding(w).(helium.Writer)
	require.True(t, ok)

	var buf bytes.Buffer
	require.NoError(t, declOnly.WriteTo(&buf, doc))

	out := buf.String()
	require.Contains(t, out, `encoding="US-ASCII"`)
	// The character-map replacement stays raw (declaration-only keeps octets UTF-8).
	require.Contains(t, out, "€")
	// A non-mapped non-ASCII text character is still char-referenced.
	require.Contains(t, out, "&#xE9;")
}

// TestOutputEncodingTrimsPaddedLabel asserts a whitespace-padded OutputEncoding
// label is trimmed before it reaches the declaration, so the encoder lookup and
// the emitted EncName agree and the declaration is well-formed. The body is
// transcoded to the resolved encoding and the result reparses.
func TestOutputEncodingTrimsPaddedLabel(t *testing.T) {
	t.Parallel()

	src := []byte(`<?xml version="1.0" encoding="UTF-8"?><root>é</root>`)
	doc, err := helium.NewParser().Parse(t.Context(), src)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = helium.NewWriter().EscapeNonASCII(false).OutputEncoding(" ISO-8859-1 ").WriteTo(&buf, doc)
	require.NoError(t, err)

	out := buf.String()
	// The declaration carries the trimmed, valid EncName — not the padded label.
	require.Contains(t, out, `encoding="ISO-8859-1"`)
	require.NotContains(t, out, `encoding=" ISO-8859-1 "`)
	// Body byte 0xE9 confirms the ISO-8859-1 encoder was actually installed.
	require.Contains(t, out, "\xe9")
	require.NotContains(t, out, "\xc3\xa9")

	// The declaration is a valid EncName, so the output reparses and the content
	// round-trips. Re-serialize forcing UTF-8 so the recovered character is the
	// raw UTF-8 "é" rather than its Latin-1 byte.
	rt, err := helium.NewParser().Parse(t.Context(), buf.Bytes())
	require.NoError(t, err)
	var rtbuf bytes.Buffer
	require.NoError(t, helium.NewWriter().EscapeNonASCII(false).OutputEncoding("UTF-8").WriteTo(&rtbuf, rt))
	require.Contains(t, rtbuf.String(), "é")
}

// TestSerializeNoOverrideASCIIAliasBytesUnchanged asserts the NO-OVERRIDE path
// stays byte-identical to origin for US-ASCII aliases recorded on the document:
// csASCII / ANSI_X3.4-1968 emit raw UTF-8 (via the historical passthrough), while
// us-ascii / ascii / iso-ir-6 character-reference. This is the binding non-goal —
// no OutputEncoding override, so nothing may divert these bytes.
func TestSerializeNoOverrideASCIIAliasBytesUnchanged(t *testing.T) {
	t.Parallel()

	// raw: the doc encoding names whose no-override output keeps the non-ASCII
	// character as raw UTF-8 octets. charref: those that emit a hex reference.
	raw := []string{"csASCII", "ANSI_X3.4-1968"}
	charref := []string{"us-ascii", "ascii", "iso-ir-6"}

	build := func(t *testing.T, enc string) string {
		t.Helper()
		doc := helium.NewDefaultDocument()
		root, err := doc.CreateElement("root")
		require.NoError(t, err)
		require.NoError(t, doc.SetDocumentElement(root))
		require.NoError(t, root.AddChild(doc.CreateText([]byte("é"))))
		doc.SetEncoding(enc)
		var buf bytes.Buffer
		require.NoError(t, helium.NewWriter().WriteTo(&buf, doc))
		return buf.String()
	}

	for _, enc := range raw {
		out := build(t, enc)
		require.Contains(t, out, "\xc3\xa9", "no-override %q must emit raw UTF-8", enc)
		require.NotContains(t, out, "&#xE9;", "no-override %q must not char-reference", enc)
	}
	for _, enc := range charref {
		out := build(t, enc)
		require.Contains(t, out, "&#xE9;", "no-override %q must char-reference", enc)
		require.NotContains(t, out, "\xc3\xa9", "no-override %q must not emit raw UTF-8", enc)
	}
}
