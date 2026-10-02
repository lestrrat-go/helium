package helium_test

import (
	"errors"
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

// positionEncoding is one input encoding of the error-position tests. name is
// the declared encoding ("" declares none), bom adds a UTF-8 byte-order mark,
// and enc transcodes from UTF-8 (nil for UTF-8 itself; a UseBOM UTF-16
// encoder writes its own byte-order mark).
type positionEncoding struct {
	label string
	name  string
	bom   bool
	enc   xenc.Encoding
}

var positionEncodings = []positionEncoding{
	{label: "UTF-8 undeclared", name: ""},
	{label: "UTF-8", name: utf8Name},
	{label: "UTF-8 with BOM undeclared", name: "", bom: true},
	{label: "UTF-8 with BOM", name: utf8Name, bom: true},
	{label: "UTF-16LE with BOM", name: utf16Name, enc: unicode.UTF16(unicode.LittleEndian, unicode.UseBOM)},
	{label: "UTF-16BE with BOM", name: utf16Name, enc: unicode.UTF16(unicode.BigEndian, unicode.UseBOM)},
	{label: "UTF-16LE without BOM", name: "UTF-16LE", enc: unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)},
	{label: "UTF-16BE without BOM", name: "UTF-16BE", enc: unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM)},
	{label: "UCS-4LE", name: ucs4Name, enc: utf32.UTF32(utf32.LittleEndian, utf32.IgnoreBOM)},
	{label: "UCS-4BE", name: ucs4Name, enc: utf32.UTF32(utf32.BigEndian, utf32.IgnoreBOM)},
	{label: "EBCDIC 037", name: "IBM037", enc: charmap.CodePage037},
	{label: latin1Name, name: latin1Name, enc: charmap.ISO8859_1},
	{label: "windows-1252", name: "windows-1252", enc: charmap.Windows1252},
	{label: shiftJISName, name: shiftJISName, enc: japanese.ShiftJIS},
}

func (e positionEncoding) encode(t *testing.T, s string) []byte {
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

// declarations returns XML declarations of different lengths for the
// encoding, including one that spans two lines.
func (e positionEncoding) declarations() []string {
	enc := ""
	if e.name != "" {
		enc = ` encoding="` + e.name + `"`
	}
	split := enc
	if split == "" {
		split = ` encoding="` + utf8Name + `"`
	}
	return []string{
		`<?xml version="1.0"` + enc + `?>`,
		`<?xml version="1.0"` + enc + ` standalone="yes"?>`,
		`<?xml  version='1.0'` + enc + `   ?>`,
		`<?xml version="1.0"` + "\n" + split + `?>`,
	}
}

// textDeclaration returns the TextDecl of an external entity or DTD in the
// encoding.
func (e positionEncoding) textDeclaration() string {
	name := e.name
	if name == "" {
		name = utf8Name
	}
	return `<?xml encoding="` + name + `"?>`
}

// errorPosition is the location part of a parse error.
type errorPosition struct {
	line    int
	column  int
	context string
}

func parseErrorPosition(t *testing.T, err error) errorPosition {
	t.Helper()
	require.Error(t, err)
	var pe helium.ErrParseError
	require.True(t, errors.As(err, &pe), "error %q must be a helium.ErrParseError", err)
	return errorPosition{line: pe.LineNumber, column: pe.Column, context: pe.Line}
}

// afterText returns where an error lies when the text prefix precedes it on
// its input: the error's own position, moved past prefix when the error is on
// prefix's last line. Columns count bytes, and the prefix is ASCII.
func afterText(prefix string, at errorPosition) errorPosition {
	lines := strings.Count(prefix, "\n")
	if at.line > 1 {
		at.line += lines
		return at
	}
	last := prefix[strings.LastIndexByte(prefix, '\n')+1:]
	return errorPosition{line: at.line + lines, column: len(last) + at.column, context: last + at.context}
}

// positionBodies are document bodies with an error on their first line. Each
// follows the XML declaration directly, so the error shares the declaration's
// line, or follows it after a newline, so the error is on the next line.
var positionBodies = []struct {
	name string
	body string
}{
	{name: "undeclared entity", body: `<r>&bad;</r>`},
	{name: "mismatched end tag", body: `<r><a></b></r>`},
	{name: "duplicate attribute", body: `<r a="1" a="2"/>`},
	{name: "comment with double hyphen", body: `<!-- a -- b --><r/>`},
	{name: "undeclared entity on the next line", body: "\n" + `<r>&bad;</r>`},
}

// TestErrorPositionAfterXMLDeclaration parses documents whose error is on the
// line the XML declaration ends on, in every encoding the parser decodes. The
// line and column count the declaration, and the context line starts with it,
// as in libxml2: a document `<?xml version="1.0"?><r>&bad;</r>` reports column
// 30 for the undeclared entity, as xmllint does. A byte-order mark is not
// counted. ParseReader with one-byte reads and the push parser with one-byte
// pushes report the same error.
func TestErrorPositionAfterXMLDeclaration(t *testing.T) {
	t.Parallel()

	for _, b := range positionBodies {
		t.Run(b.name, func(t *testing.T) {
			t.Parallel()

			// The body's error position with nothing before it.
			_, err := helium.NewParser().Parse(t.Context(), []byte(strings.TrimPrefix(b.body, "\n")))
			bare := parseErrorPosition(t, err)
			if strings.HasPrefix(b.body, "\n") {
				bare.line++
			}

			for _, e := range positionEncodings {
				t.Run(e.label, func(t *testing.T) {
					t.Parallel()

					for _, decl := range e.declarations() {
						input := e.encode(t, decl+b.body)
						_, err := helium.NewParser().Parse(t.Context(), input)
						require.Equal(t, afterText(decl, bare), parseErrorPosition(t, err), "declaration %q", decl)

						want := err.Error()
						require.Equal(t, want, parseReaderChunked(t, input, 1), "ParseReader, declaration %q", decl)
						require.Equal(t, want, pushChunked(t, input, 1), "push parser, declaration %q", decl)
					}
				})
			}
		})
	}
}

// TestErrorPositionMatchesXmllint pins the positions xmllint (libxml2 2.9.14)
// reports for errors on the first line of a document and of an external
// entity.
func TestErrorPositionMatchesXmllint(t *testing.T) {
	t.Parallel()

	bom := "\xEF\xBB\xBF"
	cases := []struct {
		doc  string
		want errorPosition
	}{
		{doc: `<?xml version="1.0"?><r>&bad;</r>`, want: errorPosition{line: 1, column: 30, context: `<?xml version="1.0"?><r>&bad;`}},
		{doc: bom + `<?xml version="1.0"?><r>&bad;</r>`, want: errorPosition{line: 1, column: 30, context: `<?xml version="1.0"?><r>&bad;`}},
		{doc: `<?xml version="1.0" encoding="UTF-8"?><r>&bad;</r>`, want: errorPosition{line: 1, column: 47, context: `<?xml version="1.0" encoding="UTF-8"?><r>&bad;`}},
		{doc: "<?xml version=\"1.0\"\n encoding=\"UTF-8\"?><r>&bad;</r>", want: errorPosition{line: 2, column: 28, context: ` encoding="UTF-8"?><r>&bad;`}},
		{doc: "<?xml version=\"1.0\"?>\n<r>&bad;</r>", want: errorPosition{line: 2, column: 9, context: `<r>&bad;`}},
	}
	for _, c := range cases {
		_, err := helium.NewParser().Parse(t.Context(), []byte(c.doc))
		require.Equal(t, c.want, parseErrorPosition(t, err), "document %q", c.doc)
	}

	ext := parseResourceError(t, positionEncoding{}, `<!DOCTYPE r [<!ENTITY ext SYSTEM "res">]>`+"\n"+`<r>&ext;</r>`, `<?xml encoding="UTF-8"?><a>&bad;</a>`)
	require.Equal(t, errorPosition{line: 1, column: 33, context: `<?xml encoding="UTF-8"?><a>&bad;`}, ext, "external entity")
	subset := parseResourceError(t, positionEncoding{}, `<!DOCTYPE r SYSTEM "res">`+"\n"+`<r/>`, `<?xml encoding="UTF-8"?><!ELEMENT r (#PCDATA)><!BOGUS>`)
	require.Equal(t, 1, subset.line, "external subset")
	require.Equal(t, 47, subset.column, "external subset")
}

// TestErrorPositionWithoutXMLDeclaration checks that a document without an XML
// declaration, with or without a byte-order mark, reports its first-line
// error at the column xmllint gives: the mark is not counted.
func TestErrorPositionWithoutXMLDeclaration(t *testing.T) {
	t.Parallel()

	for _, input := range [][]byte{
		[]byte(`<r>&bad;</r>`),
		append([]byte{0xEF, 0xBB, 0xBF}, `<r>&bad;</r>`...),
	} {
		_, err := helium.NewParser().Parse(t.Context(), input)
		require.Equal(t, errorPosition{line: 1, column: 9, context: `<r>&bad;`}, parseErrorPosition(t, err), "input %q", input)
	}
}

// TestErrorPositionAfterTextDeclaration parses external parsed entities, an
// external DTD subset, and external parameter entities that start with a
// TextDecl and have an error on its line. Positions are relative to the
// external resource and count the TextDecl, as in libxml2: xmllint reports
// column 33 for `<?xml encoding="UTF-8"?><a>&bad;</a>` and column 47 for
// `<?xml encoding="UTF-8"?><!ELEMENT r (#PCDATA)><!BOGUS>`.
func TestErrorPositionAfterTextDeclaration(t *testing.T) {
	t.Parallel()

	const (
		entityBody = `<a>&bad;</a>`
		subsetBody = `<!ELEMENT r (#PCDATA)><!BOGUS>`
		peBody     = `<!ELEMENT r (#PCDATA)><!ATTLIST r a CDATA #BOGUS>`
	)
	resources := []struct {
		name string
		doc  string
		body string
	}{
		{
			name: "external general entity",
			doc:  `<!DOCTYPE r [<!ENTITY ext SYSTEM "res">]>` + "\n" + `<r>&ext;</r>`,
			body: entityBody,
		},
		{
			name: "external DTD subset",
			doc:  `<!DOCTYPE r SYSTEM "res">` + "\n" + `<r/>`,
			body: subsetBody,
		},
		{
			name: "external parameter entity",
			doc:  `<!DOCTYPE r [<!ENTITY % pe SYSTEM "res">` + "\n" + `%pe;]>` + "\n" + `<r/>`,
			body: peBody,
		},
	}

	for _, r := range resources {
		t.Run(r.name, func(t *testing.T) {
			t.Parallel()

			// The resource's error position without a TextDecl.
			bare := parseResourceError(t, positionEncoding{}, r.doc, r.body)
			nextLine := bare
			nextLine.line++

			for _, e := range positionEncodings {
				t.Run(e.label, func(t *testing.T) {
					t.Parallel()

					decl := e.textDeclaration()
					require.Equal(t, afterText(decl, bare), parseResourceError(t, e, r.doc, decl+r.body))
					require.Equal(t, nextLine, parseResourceError(t, e, r.doc, decl+"\n"+r.body), "error on the next line")
				})
			}
		})
	}
}

// parseResourceError parses doc, whose external resource res holds the text
// res encoded in e, and returns the error position.
func parseResourceError(t *testing.T, e positionEncoding, doc, res string) errorPosition {
	t.Helper()
	fsys := fstest.MapFS{"res": &fstest.MapFile{Data: e.encode(t, res)}}
	p := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).SubstituteEntities(true).FS(fsys)
	_, err := p.Parse(t.Context(), []byte(`<?xml version="1.0"?>`+"\n"+doc))
	return parseErrorPosition(t, err)
}
