package helium_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/sax"
	"github.com/stretchr/testify/require"
)

const (
	commentCase    = "comment"
	textCase       = "text"
	restrictedChar = "\x7f"
	xml10Name      = "XML 1.0"
	xml11Name      = "XML 1.1"
)

func TestXML11Characters(t *testing.T) {
	t.Parallel()

	t.Run("raw restricted characters", func(t *testing.T) {
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
				name: textCase,
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
				{name: xml10Name, value: ver10},
				{name: xml11Name, value: ver11, wantErr: true},
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
	})

	t.Run("restricted character references", func(t *testing.T) {
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
	})

	t.Run("entity-value character references", func(t *testing.T) {
		for _, tc := range []struct {
			name    string
			version string
			wantErr bool
		}{
			{name: xml10Name, version: ver10, wantErr: true},
			{name: xml11Name, version: ver11},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				source := `<?xml version="` + tc.version + `"?><!DOCTYPE root [<!ENTITY e "&#1;">]><root/>`
				doc, err := helium.NewParser().Parse(t.Context(), []byte(source))
				if tc.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				entity, found := doc.GetEntity("e")
				require.True(t, found)
				require.NotNil(t, entity)
			})
		}
	})

	t.Run("external entity literal characters", func(t *testing.T) {
		for _, tc := range []struct {
			name       string
			docVersion string
			entity     string
			wantErr    bool
		}{
			{
				name:       "XML 1.0 document inherits XML 1.0",
				docVersion: ver10,
				entity:     `<child>` + restrictedChar + `</child>`,
			},
			{
				name:       "XML 1.1 document inherits XML 1.1",
				docVersion: ver11,
				entity:     `<child>` + restrictedChar + `</child>`,
				wantErr:    true,
			},
			{
				name:       "XML 1.1 document honors XML 1.0 TextDecl",
				docVersion: ver11,
				entity:     `<?xml version="1.0" encoding="UTF-8"?><child>` + restrictedChar + `</child>`,
			},
			{
				name:       "XML 1.1 document honors XML 1.1 TextDecl",
				docVersion: ver11,
				entity:     `<?xml version="1.1" encoding="UTF-8"?><child>` + restrictedChar + `</child>`,
				wantErr:    true,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				source := `<?xml version="` + tc.docVersion + `"?><!DOCTYPE root [<!ENTITY e SYSTEM "entity.ent">]><root>&e;</root>`
				fsys := fstest.MapFS{
					"entity.ent": &fstest.MapFile{Data: []byte(tc.entity)},
				}
				doc, err := helium.NewParser().
					BlockXXE(false).
					SubstituteEntities(true).
					FS(fsys).
					Parse(t.Context(), []byte(source))
				if tc.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				require.Equal(t, restrictedChar, string(doc.DocumentElement().FirstChild().Content()))
			})
		}
	})

	t.Run("external DTD IGNORE literal characters", func(t *testing.T) {
		for _, tc := range []struct {
			name    string
			version string
			wantErr bool
		}{
			{name: xml10Name, version: ver10},
			{name: xml11Name, version: ver11, wantErr: true},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				source := `<?xml version="` + tc.version + `"?><!DOCTYPE root SYSTEM "ignore.dtd"><root/>`
				fsys := fstest.MapFS{
					"ignore.dtd": &fstest.MapFile{Data: []byte(`<![IGNORE[` + restrictedChar + `]]>`)},
				}
				_, err := helium.NewParser().
					BlockXXE(false).
					LoadExternalDTD(true).
					FS(fsys).
					Parse(t.Context(), []byte(source))
				if tc.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
			})
		}
	})

	t.Run("literal character matrix", func(t *testing.T) {
		for _, lc := range literalCharContexts {
			for _, lv := range literalCharValues {
				for _, version := range []struct {
					name  string
					value string
					valid bool
				}{
					{name: xml10Name, value: ver10, valid: lv.valid10},
					{name: xml11Name, value: ver11, valid: lv.valid11},
				} {
					t.Run(lc.name+" "+lv.name+" "+version.name, func(t *testing.T) {
						t.Parallel()

						// The padding puts the tested character past the first
						// eight bytes, so word-at-a-time scanning reaches it.
						value := "0123456789" + lv.value + "0123456789"
						err := parseLiteralCharContext(t, lc, version.value, value)
						if version.valid {
							require.NoError(t, err)
							return
						}
						require.Error(t, err)
					})
				}
			}
		}
	})

	t.Run("literal character in text runs", func(t *testing.T) {
		// The text scanner validates each character as it scans the run.
		// Place every character at each offset of the scanner's 16-byte and
		// 8-byte blocks, before each run delimiter, across the cursor's 8 KiB
		// buffer, at the node-content cap, and across reads and push chunks.
		for _, lv := range literalCharValues {
			for _, version := range []struct {
				name  string
				value string
				valid bool
			}{
				{name: xml10Name, value: ver10, valid: lv.valid10},
				{name: xml11Name, value: ver11, valid: lv.valid11},
			} {
				t.Run(lv.name+" "+version.name, func(t *testing.T) {
					t.Parallel()

					// Under XML 1.1 a RestrictedChar is a Char, so the
					// scanner keeps it in the run and reports the whole run
					// invalid with ErrInvalidChar.
					restricted := version.value == ver11 && lv.valid10 && !lv.valid11
					want := textRunWant{valid: version.valid, restricted: restricted}

					for pre := range 34 {
						for _, suffix := range []string{"", "<b/>", "&amp;", "]", "]]&gt;", "\r\n"} {
							text := strings.Repeat("a", pre) + lv.value + suffix + "z"
							source := fmt.Sprintf(`<?xml version="%s"?><root>%s</root>`, version.value, text)
							for _, mode := range textRunModes {
								checkTextRun(t, helium.NewParser(), mode, source, want)
							}
						}
					}

					header := fmt.Sprintf(`<?xml version="%s"?><root>`, version.value)
					for _, edge := range []int{8192, 16384} {
						for d := -4; d <= 4; d++ {
							text := strings.Repeat("a", edge-len(header)+d) + lv.value + "tail"
							source := header + text + "</root>"
							for _, mode := range textRunModes {
								checkTextRun(t, helium.NewParser(), mode, source, want)
							}
						}
					}

					// With a 64-byte cap, the run is too large unless the
					// scan stops at the character before the cap: a
					// character outside the XML 1.0 Char production ends
					// the run, while a RestrictedChar does not.
					capped := helium.NewParser().MaxNodeContentSize(64)
					for pre := 56; pre <= 72; pre++ {
						text := strings.Repeat("a", pre) + lv.value + strings.Repeat("b", 80)
						source := fmt.Sprintf(`<?xml version="%s"?><root>%s</root>`, version.value, text)
						wantCap := textRunWant{tooLarge: lv.valid10 || pre > 64}
						checkTextRun(t, capped, textRunParse, source, wantCap)
						checkTextRun(t, capped, textRunReader, source, wantCap)
					}
				})
			}
		}
	})

	t.Run("a control character reference", func(t *testing.T) {
		// XML 1.1 permits character references to the C0/C1 control characters
		// (all but U+0000) that the XML 1.0 Char production forbids.
		doc, err := helium.NewParser().Parse(t.Context(),
			[]byte(`<?xml version="1.1"?><root>&#7;&#131;&#133;</root>`))
		require.NoError(t, err, "XML 1.1 must accept control-character references")
		require.Equal(t, "\u0007\u0083\u0085", string(doc.DocumentElement().Content()))

		// XML 1.0 (and an implicit-1.0 document) must still reject them, and U+0000
		// is invalid in every XML version.
		for _, in := range []string{
			`<?xml version="1.0"?><root>&#7;</root>`,
			`<root>&#7;</root>`,
			`<?xml version="1.1"?><root>&#0;</root>`,
		} {
			_, err := helium.NewParser().Parse(t.Context(), []byte(in))
			require.Error(t, err, "must reject %q", in)
		}
	})
}

// literalCharMode selects how a literalCharContext document reaches the parser.
type literalCharMode int

const (
	literalCharParse literalCharMode = iota
	literalCharParseReader
	literalCharChunkedSAX
	literalCharExternalPE
)

// literalCharContext is one place a literal character can appear. template is
// a fmt format whose first argument is the XML version and second the value.
// In literalCharExternalPE mode, template is the document and peTemplate the
// external parameter entity body, whose only argument is the value.
type literalCharContext struct {
	name       string
	template   string
	peTemplate string
	mode       literalCharMode
}

var literalCharContexts = []literalCharContext{
	{name: textCase, template: `<?xml version="%s"?><root>%s</root>`},
	{name: "text via reader", template: `<?xml version="%s"?><root>%s</root>`, mode: literalCharParseReader},
	{name: "text via chunked SAX", template: `<?xml version="%s"?><root>%s</root>`, mode: literalCharChunkedSAX},
	{name: "CDATA", template: `<?xml version="%s"?><root><![CDATA[%s]]></root>`},
	{name: "attribute", template: `<?xml version="%s"?><root attr="%s"/>`},
	{name: "attribute after a reference", template: `<?xml version="%s"?><root attr="&amp;%s"/>`},
	{name: commentCase, template: `<?xml version="%s"?><root><!--%s--></root>`},
	{name: "processing instruction", template: `<?xml version="%s"?><root><?pi %s?></root>`},
	{name: "internal entity value", template: `<?xml version="%s"?><!DOCTYPE root [<!ENTITY e "%s">]><root/>`},
	{
		name:       "external parameter entity",
		template:   `<?xml version="%s"?><!DOCTYPE root [<!ENTITY %% pe SYSTEM "pe.ent">%%pe;]><root/>`,
		peTemplate: `<!ENTITY e "%s">`,
		mode:       literalCharExternalPE,
	},
}

// literalCharValues lists characters with their literal validity in XML 1.0
// (Char) and XML 1.1 (Char minus RestrictedChar).
var literalCharValues = []struct {
	name    string
	value   string
	valid10 bool
	valid11 bool
}{
	{name: "ASCII", value: "plain text", valid10: true, valid11: true},
	{name: "tab and newline", value: "a\tb\nc", valid10: true, valid11: true},
	{name: "multi-byte UTF-8", value: "é日\U0001F600\U0010FFFF", valid10: true, valid11: true},
	{name: "U+FFFD", value: "�", valid10: true, valid11: true},
	{name: "DEL", value: "\x7f", valid10: true, valid11: false},
	{name: "C0 U+0001", value: "\x01", valid10: false, valid11: false},
	{name: "C0 U+001F", value: "\x1f", valid10: false, valid11: false},
	{name: "C1 U+0080", value: "\u0080", valid10: true, valid11: false},
	{name: "C1 U+0084", value: "\u0084", valid10: true, valid11: false},
	{name: "C1 U+0085", value: "\u0085", valid10: true, valid11: true},
	{name: "C1 U+009F", value: "\u009f", valid10: true, valid11: false},
	{name: "U+FFFE", value: "￾", valid10: false, valid11: false},
	{name: "U+FFFF", value: "￿", valid10: false, valid11: false},
	{name: "lone surrogate", value: "\xed\xa0\x80", valid10: false, valid11: false},
	{name: "truncated UTF-8", value: "\xe2\x82", valid10: false, valid11: false},
	{name: "stray continuation byte", value: "\x80", valid10: false, valid11: false},
	{name: "invalid byte", value: "\xff", valid10: false, valid11: false},
}

// textRunMode selects how checkTextRun feeds a document to the parser.
type textRunMode int

const (
	textRunParse textRunMode = iota
	textRunReader
	textRunReaderSplit
	textRunChunkedSAX
	textRunPush
)

var textRunModes = []textRunMode{textRunParse, textRunReader, textRunReaderSplit, textRunChunkedSAX, textRunPush}

// textRunWant is the outcome checkTextRun expects: success when valid; else an
// error that is ErrNodeContentTooLarge when tooLarge and ErrInvalidChar when
// restricted.
type textRunWant struct {
	valid      bool
	restricted bool
	tooLarge   bool
}

// checkTextRun parses source with p in the given mode and checks the outcome.
// textRunReaderSplit and textRunPush split the input into small pieces (1 or
// 3 bytes; 509 for documents past 1 KiB, to keep the test fast) so that
// character data and multi-byte characters straddle reads.
func checkTextRun(t *testing.T, p helium.Parser, mode textRunMode, source string, want textRunWant) {
	t.Helper()

	small := len(source) <= 1024
	var err error
	switch mode {
	case textRunParse:
		_, err = p.Parse(t.Context(), []byte(source))
	case textRunReader:
		_, err = p.ParseReader(t.Context(), strings.NewReader(source))
	case textRunReaderSplit:
		chunk := 509
		if small {
			chunk = 1
		}
		_, err = p.ParseReader(t.Context(), &parserChunkedReader{data: []byte(source), chunk: chunk})
	case textRunChunkedSAX:
		_, err = p.SAXHandler(sax.New()).
			CharBufferSize(4).
			ParseReader(t.Context(), &parserChunkedReader{data: []byte(source), chunk: 5})
	case textRunPush:
		chunk := 509
		if small {
			chunk = 3
		}
		err = pushInChunks(t, p, []byte(source), chunk)
	}

	if want.valid {
		require.NoError(t, err, "mode %d: %q", mode, tailString(source))
		return
	}
	require.Error(t, err, "mode %d: %q", mode, tailString(source))
	require.Equal(t, want.tooLarge, errors.Is(err, helium.ErrNodeContentTooLarge),
		"mode %d: %q: ErrNodeContentTooLarge, got %v", mode, tailString(source), err)
	if want.restricted {
		require.ErrorIs(t, err, helium.ErrInvalidChar, "mode %d: %q", mode, tailString(source))
	}
}

// pushInChunks feeds b to a push parser chunk bytes at a time and returns the
// parse error.
func pushInChunks(t *testing.T, p helium.Parser, b []byte, chunk int) error {
	t.Helper()

	pp := p.NewPushParser(t.Context())
	for len(b) > 0 {
		n := min(chunk, len(b))
		if err := pp.Push(b[:n]); err != nil {
			break
		}
		b = b[n:]
	}
	_, err := pp.Close()
	return err
}

// tailString returns at most the last 64 bytes of s, keeping failure messages
// short for documents that cross the 8 KiB cursor buffer.
func tailString(s string) string {
	return s[max(0, len(s)-64):]
}

func parseLiteralCharContext(t *testing.T, lc literalCharContext, version, value string) error {
	t.Helper()

	switch lc.mode {
	case literalCharParseReader:
		source := fmt.Sprintf(lc.template, version, value)
		_, err := helium.NewParser().ParseReader(t.Context(), strings.NewReader(source))
		return err
	case literalCharChunkedSAX:
		source := fmt.Sprintf(lc.template, version, value)
		_, err := helium.NewParser().
			SAXHandler(sax.New()).
			CharBufferSize(4).
			ParseReader(t.Context(), strings.NewReader(source))
		return err
	case literalCharExternalPE:
		fsys := fstest.MapFS{
			"pe.ent": &fstest.MapFile{Data: fmt.Appendf(nil, lc.peTemplate, value)},
		}
		_, err := helium.NewParser().
			BlockXXE(false).
			LoadExternalDTD(true).
			FS(fsys).
			Parse(t.Context(), fmt.Appendf(nil, lc.template, version))
		return err
	default:
		source := fmt.Sprintf(lc.template, version, value)
		_, err := helium.NewParser().Parse(t.Context(), []byte(source))
		return err
	}
}

func TestXML11PrefixUndeclaration(t *testing.T) {
	t.Parallel()

	// Namespaces in XML 1.1 §5: a prefixed namespace declaration with an empty
	// value (xmlns:pfx="") undeclares the prefix. This is well-formed only in an
	// XML 1.1 document; XML 1.0 forbids it.
	const undecl = `<doc xmlns:a="http://a/"><para xmlns:a=""/></doc>`

	// XML 1.0: rejected.
	_, err := helium.NewParser().Parse(t.Context(),
		[]byte(`<?xml version="1.0"?>`+undecl))
	require.Error(t, err, "XML 1.0 must reject a prefixed namespace undeclaration")

	// No XML declaration defaults to XML 1.0: rejected.
	_, err = helium.NewParser().Parse(t.Context(), []byte(undecl))
	require.Error(t, err, "an implicit XML 1.0 document must reject xmlns:pfx=\"\"")

	// XML 1.1: accepted, and the prefix binding is removed on the inner element.
	doc, err := helium.NewParser().Parse(t.Context(),
		[]byte(`<?xml version="1.1"?>`+undecl))
	require.NoError(t, err, "XML 1.1 must accept a prefixed namespace undeclaration")

	para := doc.DocumentElement().FirstChild().(*helium.Element)
	require.Equal(t, "para", para.Name())
	var hasUndecl bool
	for _, ns := range para.Namespaces() {
		if ns.Prefix() == "a" {
			require.Equal(t, "", ns.URI(),
				"the prefix a must be undeclared (empty URI) on the inner element")
			hasUndecl = true
		}
	}
	require.True(t, hasUndecl, "the undeclaration must be recorded on the inner element")

	// The reserved xml/xmlns prefixes may never be undeclared, even in XML 1.1.
	for _, input := range []string{
		`<?xml version="1.1"?><doc xmlns:xml=""/>`,
		`<?xml version="1.1"?><doc xmlns:xmlns=""/>`,
	} {
		_, err := helium.NewParser().Parse(t.Context(), []byte(input))
		require.Error(t, err, "must reject undeclaring a reserved prefix in %q", input)
	}
}
