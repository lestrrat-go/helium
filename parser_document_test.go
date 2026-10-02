package helium_test

import (
	"bytes"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
	xenc "golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/encoding/unicode/utf32"
)

// contentEncoding is one input encoding of the content-cursor matrix. name is
// the value of the encoding declaration ("" for none), version the XML
// version, bom whether the input starts with a byte-order mark, and enc the
// transcoder from UTF-8 (nil for UTF-8 itself). nameText is a name and text
// the encoding can represent, so element names, attribute values, and text
// carry characters beyond ASCII wherever the encoding has them.
type contentEncoding struct {
	label    string
	name     string
	version  string
	bom      bool
	enc      xenc.Encoding
	nameText string
}

const (
	// utf8Name is the encoding declaration value of the UTF-8 cases.
	utf8Name = "UTF-8"
	// The encoding declaration values of the UTF-16 cases with a byte-order
	// mark, the UCS-4 cases, and the Shift_JIS cases.
	utf16Name    = "UTF-16"
	ucs4Name     = "UCS-4"
	shiftJISName = "Shift_JIS"
	// The nameText values: Latin and CJK for the Unicode encodings, Latin-1
	// for the single-byte ones, and Japanese for Shift_JIS and EUC-JP.
	mixedNameText    = "café-日本"
	latinNameText    = "café-üß"
	japaneseNameText = "日本-テキスト"
)

var contentEncodings = []contentEncoding{
	{label: "UTF-8 without declaration", version: ver10, nameText: mixedNameText},
	{label: "UTF-8 declared", name: utf8Name, version: ver10, nameText: mixedNameText},
	{label: "UTF-8 with BOM", version: ver10, bom: true, nameText: mixedNameText},
	{label: "UTF-8 declared with BOM", name: utf8Name, version: ver10, bom: true, nameText: mixedNameText},
	{label: "XML 1.1 UTF-8", name: utf8Name, version: ver11, nameText: mixedNameText},
	{label: "XML 1.1 ISO-8859-1", name: "ISO-8859-1", version: ver11, enc: charmap.ISO8859_1, nameText: latinNameText},
	{label: "US-ASCII", name: "US-ASCII", version: ver10, enc: charmap.ISO8859_1, nameText: "plain-ascii"},
	{label: "ISO-8859-1", name: "ISO-8859-1", version: ver10, enc: charmap.ISO8859_1, nameText: latinNameText},
	{label: "windows-1252", name: "windows-1252", version: ver10, enc: charmap.Windows1252, nameText: "café-œž"},
	{label: "UTF-16LE with BOM", name: utf16Name, version: ver10, enc: unicode.UTF16(unicode.LittleEndian, unicode.UseBOM), nameText: mixedNameText},
	{label: "UTF-16BE with BOM", name: utf16Name, version: ver10, enc: unicode.UTF16(unicode.BigEndian, unicode.UseBOM), nameText: mixedNameText},
	{label: "UTF-16LE with BOM undeclared", version: ver10, enc: unicode.UTF16(unicode.LittleEndian, unicode.UseBOM), nameText: mixedNameText},
	{label: "UTF-16LE without BOM", name: "UTF-16LE", version: ver10, enc: unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM), nameText: mixedNameText},
	{label: "UTF-16BE without BOM", name: "UTF-16BE", version: ver10, enc: unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM), nameText: mixedNameText},
	{label: "UCS-4BE", name: ucs4Name, version: ver10, enc: utf32.UTF32(utf32.BigEndian, utf32.IgnoreBOM), nameText: mixedNameText},
	{label: "UCS-4LE", name: ucs4Name, version: ver10, enc: utf32.UTF32(utf32.LittleEndian, utf32.IgnoreBOM), nameText: mixedNameText},
	{label: "EBCDIC 037", name: "IBM037", version: ver10, enc: charmap.CodePage037, nameText: latinNameText},
	{label: shiftJISName, name: shiftJISName, version: ver10, enc: japanese.ShiftJIS, nameText: japaneseNameText},
	{label: "EUC-JP", name: "EUC-JP", version: ver10, enc: japanese.EUCJP, nameText: japaneseNameText},
}

// contentBody is the root element of the content-cursor documents: nested and
// prefixed elements, attributes needing normalization, references, character
// references, CDATA, comments, and PIs. NAME is replaced by the encoding's
// nameText.
const contentBody = `<r xmlns="urn:d" xmlns:p="urn:p" tok="  a   b  " p:at="v&#x20;&lt;w&gt;" >
  <p:c a="1" b='2'>text NAME &#xe9;&#x4e2d; <![CDATA[cd <x> NAME]]><!-- c NAME --><?pi data NAME?></p:c>
  <NAME NAME="NAME">NAME&amp;&lt;</NAME>
  <e/>
  <x   y = "z" ></x  >
  &int;
  BAD
</r>`

// contentErrors are the variants of contentBody, named in failure messages:
// the "well-formed" variant drops the BAD marker; each other variant replaces
// it with a malformed construct on the same line.
var contentErrors = []struct {
	name string
	bad  string
}{
	{name: "well-formed", bad: ""},
	{name: "duplicate attribute", bad: `<d a="1" a="2"/>`},
	{name: "mismatched end tag", bad: `<m></n>`},
	{name: "undeclared entity", bad: `&nope;`},
	{name: "invalid character reference", bad: `&#x0;`},
}

// contentSubset is the internal subset every content-cursor document carries.
// The entity int adds element content parsed from an internal entity's
// replacement text.
const contentSubset = `<!DOCTYPE r [
<!ENTITY int "<i a='1 &#38;amp; 2'>int NAME &#38;#xe9;</i>">
<!ATTLIST r tok NMTOKENS #IMPLIED def CDATA "dflt">
]>
`

// declLine returns the first line of a content-cursor document: the XML
// declaration the encoding names, or an empty line, so every document keeps
// its content on the same lines as its UTF-8 baseline.
func (e contentEncoding) declLine() string {
	if e.name == "" {
		return "\n"
	}
	return `<?xml version="` + e.version + `" encoding="` + e.name + `"?>` + "\n"
}

// firstLineDecl is the XML declaration of a document whose prolog and root
// start tag share the declaration's line: declLine without its line end.
func (e contentEncoding) firstLineDecl() string {
	return strings.TrimSuffix(e.declLine(), "\n")
}

// baselineDeclLine is declLine for the UTF-8 baseline of the same document.
func (e contentEncoding) baselineDeclLine() string {
	return `<?xml version="` + e.version + `"?>` + "\n"
}

// encode returns s in the encoding, with a byte-order mark when the encoding
// asks for one.
func (e contentEncoding) encode(t *testing.T, s string) []byte {
	t.Helper()
	if e.enc == nil {
		if e.bom {
			return append([]byte{0xEF, 0xBB, 0xBF}, s...)
		}
		return []byte(s)
	}
	out, err := e.enc.NewEncoder().Bytes([]byte(s))
	require.NoError(t, err, "encode as %s", e.label)
	return out
}

func (e contentEncoding) fill(s string) string {
	return strings.ReplaceAll(s, "NAME", e.nameText)
}

func contentBodyWith(bad string) string {
	return strings.Replace(contentBody, "BAD", bad, 1)
}

// contentBodyOnFirstLine is contentBodyWith(bad) with the malformed construct
// moved onto the root start tag's line, right after the tag.
func contentBodyOnFirstLine(bad string) string {
	return strings.Replace(contentBodyWith(""), ">\n", ">"+bad+"\n", 1)
}

// entityValue escapes s for use inside a double-quoted entity value.
func entityValue(s string) string {
	s = strings.ReplaceAll(s, "&", "&#38;")
	s = strings.ReplaceAll(s, `"`, "&#34;")
	return strings.ReplaceAll(s, "%", "&#37;")
}

// contentResult renders a parse result for comparison: the serialized root
// element (or fragment) and the error text.
func contentResult(t *testing.T, node helium.Node, err error) string {
	t.Helper()
	require.NotErrorIs(t, err, helium.ErrContentCursorForTesting,
		"element content must be reached on a UTF-8 cursor")

	var b strings.Builder
	for n := node; n != nil; n = n.NextSibling() {
		s, werr := helium.WriteString(n)
		require.NoError(t, werr)
		b.WriteString(s)
	}
	if err != nil {
		b.WriteString("\nERROR: ")
		b.WriteString(err.Error())
	}
	return b.String()
}

func rootOf(doc *helium.Document) helium.Node {
	if doc == nil {
		return nil
	}
	if root := doc.DocumentElement(); root != nil {
		return root
	}
	return nil
}

// contentEntryPoint parses one content-cursor case through one entry point. It
// receives the document text and the text of the external entity ext.ent, and
// encodes whichever of the two the entry point exercises.
type contentEntryPoint struct {
	name  string
	parse func(t *testing.T, p helium.Parser, e contentEncoding, doc, ext string) string
}

func contentParse(t *testing.T, p helium.Parser, e contentEncoding, doc, _ string) string {
	t.Helper()
	got, err := p.Parse(t.Context(), e.encode(t, doc))
	return contentResult(t, rootOf(got), err)
}

func contentParseReader(t *testing.T, p helium.Parser, e contentEncoding, doc, _ string) string {
	t.Helper()
	got, err := p.ParseReader(t.Context(), bytes.NewReader(e.encode(t, doc)))
	return contentResult(t, rootOf(got), err)
}

func contentPush(t *testing.T, p helium.Parser, e contentEncoding, doc, _ string) string {
	t.Helper()
	input := e.encode(t, doc)
	pp := p.NewPushParser(t.Context())
	var pushErr error
	for i := range input {
		if pushErr = pp.Push(input[i : i+1]); pushErr != nil {
			break
		}
	}
	got, err := pp.Close()
	if err == nil {
		err = pushErr
	}
	return contentResult(t, rootOf(got), err)
}

// contentInNodeContext parses the encoded document, then a UTF-8 fragment in
// the context of its root element. The fragment uses the document's namespace
// prefix and internal entity.
func contentInNodeContext(t *testing.T, p helium.Parser, e contentEncoding, doc, _ string) string {
	t.Helper()
	ctxDoc, err := p.Parse(t.Context(), e.encode(t, doc))
	if err != nil {
		return contentResult(t, nil, err)
	}
	fragment := e.fill(`<p:f q="NAME" r=" a  b ">NAME<NAME/>&int;</p:f>tail &amp; NAME<!-- c -->`)
	got, err := p.ParseInNodeContext(t.Context(), ctxDoc.DocumentElement(), []byte(fragment))
	return contentResult(t, got, err)
}

// contentInternalEntity moves the whole root content into an internal entity,
// so it is parsed from the entity's replacement text.
func contentInternalEntity(t *testing.T, p helium.Parser, e contentEncoding, doc, _ string) string {
	t.Helper()
	start := strings.Index(doc, "<r ")
	open := start + strings.Index(doc[start:], ">") + 1
	end := strings.LastIndex(doc, "</r>")
	subset := strings.Replace(doc[:start], "]>", `<!ENTITY all "`+entityValue(doc[open:end])+`">`+"\n]>", 1)
	moved := subset + doc[start:open] + "&all;" + doc[end:]
	got, err := p.Parse(t.Context(), e.encode(t, moved))
	return contentResult(t, rootOf(got), err)
}

// contentExternalEntity parses a UTF-8 document whose root content is the
// external entity ext.ent, encoded in e.
func contentExternalEntity(t *testing.T, p helium.Parser, e contentEncoding, _, ext string) string {
	t.Helper()
	doc := `<?xml version="` + e.version + `"?>` + "\n" +
		`<!DOCTYPE r [<!ENTITY int "int"><!ENTITY ext SYSTEM "ext.ent">]>` + "\n" +
		`<r>&ext;</r>`
	fsys := fstest.MapFS{"ext.ent": &fstest.MapFile{Data: e.encode(t, ext)}}
	got, err := p.BlockXXE(false).FS(fsys).Parse(t.Context(), []byte(doc))
	return contentResult(t, rootOf(got), err)
}

// textDecl returns the external entity's text declaration, or nothing when the
// encoding declares no name.
func (e contentEncoding) textDecl() string {
	if e.name == "" {
		return ""
	}
	return `<?xml version="` + e.version + `" encoding="` + e.name + `"?>`
}

// TestContentCursor parses one set of documents in every input encoding the
// parser decodes, through every entry point that reaches element content, and
// requires the tree and the error text to match the same document in UTF-8.
// Element content is parsed from a *strcursor.UTF8Cursor taken once where
// content starts; none of these cases may reach it on another cursor type.
// Each document is also parsed with its prolog and the malformed construct on
// the XML declaration's (or TextDecl's) line. The error's column and context
// line then count the declaration, whose text names the encoding, so that
// document is compared with the same text read as UTF-8 with the declared
// encoding ignored.
func TestContentCursor(t *testing.T) {
	t.Parallel()

	entryPoints := []contentEntryPoint{
		{name: "Parse", parse: contentParse},
		{name: "ParseReader", parse: contentParseReader},
		{name: "push parser one byte at a time", parse: contentPush},
		{name: "ParseInNodeContext", parse: contentInNodeContext},
		{name: "internal entity content", parse: contentInternalEntity},
		{name: "external entity content", parse: contentExternalEntity},
	}
	parsers := []struct {
		name string
		p    helium.Parser
	}{
		{name: "defaults", p: helium.NewParser()},
		{name: "substituted", p: helium.NewParser().SubstituteEntities(true)},
	}

	for _, ep := range entryPoints {
		t.Run(ep.name, func(t *testing.T) {
			t.Parallel()
			for _, e := range contentEncodings {
				t.Run(e.label, func(t *testing.T) {
					t.Parallel()
					baseline := e
					baseline.name = ""
					baseline.enc = nil
					baseline.bom = false
					for _, variant := range contentErrors {
						body := e.fill(contentBodyWith(variant.bad))
						subset := e.fill(contentSubset)
						ext := "\n" + body
						for _, cfg := range parsers {
							want := ep.parse(t, cfg.p, baseline,
								baseline.baselineDeclLine()+subset+body,
								`<?xml version="`+e.version+`" encoding="`+utf8Name+`"?>`+ext)
							if variant.bad == "" {
								require.NotContains(t, want, "\nERROR: ", "%s parser", cfg.name)
								require.NotEmpty(t, want, "%s parser", cfg.name)
							}
							got := ep.parse(t, cfg.p, e, e.declLine()+subset+body, e.textDecl()+ext)
							require.Equal(t, want, got, "%s, %s parser", variant.name, cfg.name)
						}

						lineBody := e.fill(contentBodyOnFirstLine(variant.bad))
						lineDoc := e.firstLineDecl() + strings.ReplaceAll(subset, "\n", "") + lineBody
						lineExt := e.textDecl() + lineBody
						for _, cfg := range parsers {
							want := ep.parse(t, cfg.p.IgnoreEncoding(true), baseline, lineDoc, lineExt)
							if variant.bad == "" {
								require.NotContains(t, want, "\nERROR: ", "%s parser, declaration line", cfg.name)
							} else if strings.Contains(want, "\nERROR: ") {
								require.Contains(t, want, " at line 1, column ", "%s, %s parser, declaration line", variant.name, cfg.name)
							}
							got := ep.parse(t, cfg.p, e, lineDoc, lineExt)
							require.Equal(t, want, got, "%s, %s parser, declaration line", variant.name, cfg.name)
						}
					}
				})
			}
		})
	}
}
