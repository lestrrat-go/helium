package helium_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/sax"
	"github.com/lestrrat-go/pdebug"
	"github.com/stretchr/testify/require"
)

func TestDetectBOM(t *testing.T) {
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
}

func TestEmptyDocument(t *testing.T) {
	p := helium.NewParser()
	// BOM only
	_, err := p.Parse(t.Context(), []byte{0x00, 0x00, 0x00, 0x3C})
	require.Error(t, err, "Parsing BOM only should fail")
}

func TestParseXMLDecl(t *testing.T) {
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
}

func TestParseMisc(t *testing.T) {
	const decl = `<?xml version="1.0"?>` + "\n"
	const content = `<root />`
	inputs := []string{
		decl + `<?xml-stylesheet type="text/xsl" href="style.xsl"?>` + content,
		decl + `<?xml-stylesheet type="text/css" href="style.css"?>` + content,
	}

	for _, input := range inputs {
		p := helium.NewParser()
		doc, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err, "Parse should succeed for '%s'", input)

		// XXX Not sure if this is right, but I'm going to assume it's ok
		// to have a DTD in the child list
		n := doc.FirstChild()
	LOOP:
		for {
			t.Logf("%#v", n)
			if n == nil {
				t.Errorf("Could not find ProcessingInstruction node")
				return
			}

			switch n.Type() {
			case helium.ProcessingInstructionNode:
				require.IsType(t, &helium.ProcessingInstruction{}, n, "First child should be a processing instruction")

				require.IsType(t, &helium.Element{}, n.NextSibling(), "NextSibling of PI should be Element node")
				break LOOP
			}
			n = n.NextSibling()
		}
	}
}

func TestParseCharRefReplacementCharacter(t *testing.T) {
	doc, err := helium.NewParser().Parse(t.Context(), []byte("<root>&#xFFFD;</root>"))
	require.NoError(t, err)

	root := doc.DocumentElement()
	require.NotNil(t, root)
	require.Equal(t, "�", string(root.Content()))
}

func TestParse(t *testing.T) {
	//nolint:dupword // "L\nL" is intentional XML content
	const input = `<?xml version="1.0"?>
<root foo="bar">
	<!-- this is a sample comment -->
  <child>foo</child>
	<child><![CDATA[
H
E
L
L
O!]]></child>
</root>`
	p := helium.NewParser()
	_, err := p.Parse(t.Context(), []byte(input))
	require.NoError(t, err, "Parse should succeed for '%s'", input)
}

func TestParseBad(t *testing.T) {
	inputs := []string{
		`<?xml version="1.0"?>
<root foo="bar">
  <child>foo</chld>
</root>`,
		`<?xml version="abc">`,
		`<?xml varsion="1.0">`,
	}
	p := helium.NewParser()
	for _, input := range inputs {
		_, err := p.Parse(t.Context(), []byte(input))
		require.Error(t, err, "Parse should fail for '%s'", input)
	}
}

func TestParseRejectsMalformedComment(t *testing.T) {
	_, err := helium.NewParser().Parse(t.Context(), []byte("<A/><!---00\x10"))
	require.Error(t, err)
}

func TestParseRejectsDuplicateAttribute(t *testing.T) {
	// XML 1.0 §3.1 well-formedness: a start tag may not have two
	// attributes with the same (qualified) name. These must be rejected
	// even when not validating.
	reject := []string{
		`<root a="1" a="2"/>`,
		`<root xmlns:p="urn:x" p:a="1" p:a="2"/>`,
		// Duplicate default namespace declarations on the same element,
		// including when one or both are empty (xmlns="").
		`<root xmlns="urn:x" xmlns="urn:y"/>`,
		`<root xmlns="urn:x" xmlns="urn:x"/>`,
		`<root xmlns="" xmlns="urn:x"/>`,
		`<root xmlns="" xmlns=""/>`,
		// Two attributes with different prefixes but the same expanded
		// name ({urn:x}a). Forbidden by Namespaces in XML.
		`<root xmlns:p="urn:x" xmlns:q="urn:x" p:a="1" q:a="2"/>`,
	}
	for _, input := range reject {
		_, err := helium.NewParser().Parse(t.Context(), []byte(input))
		require.Error(t, err, "Parse should reject duplicate attribute in %q", input)
	}

	// The same well-formedness violations must remain fatal even with
	// CleanNamespaces(true) (parseNsClean), which only suppresses redundant
	// ancestor redeclarations, never same-element duplicates.
	rejectClean := []string{
		`<root xmlns:p="urn:x" xmlns:p="urn:x"/>`,
		`<root xmlns="urn:x" xmlns="urn:x"/>`,
		`<root xmlns="" xmlns=""/>`,
		`<root xmlns="" xmlns="urn:x"/>`,
		// A child re-declaring a prefix already bound by an ancestor to the
		// same URI is the parseNsClean redundant-redeclaration case; a
		// SECOND same-element declaration must still be fatal even though the
		// first one is skipped as redundant (not pushed onto the ns stack).
		`<root xmlns="urn:x"><child xmlns="urn:x" xmlns="urn:x"/></root>`,
		`<root xmlns:p="urn:x"><child xmlns:p="urn:x" xmlns:p="urn:x"/></root>`,
	}
	for _, input := range rejectClean {
		_, err := helium.NewParser().CleanNamespaces(true).Parse(t.Context(), []byte(input))
		require.Error(t, err, "Parse with CleanNamespaces should reject duplicate ns decl in %q", input)
	}

	// Distinct qualified names must still parse, including the same local
	// name carried by different prefixes mapped to different URIs.
	accept := []string{
		`<root a="1" b="2"/>`,
		`<root xmlns:p="urn:x" xmlns:q="urn:y" p:a="1" q:a="2"/>`,
		// Unprefixed attributes are in no namespace; a default xmlns does
		// not put them in a namespace, so distinct local names are fine.
		`<root xmlns="urn:x" a="1" b="2"/>`,
	}
	for _, input := range accept {
		_, err := helium.NewParser().Parse(t.Context(), []byte(input))
		require.NoError(t, err, "Parse should accept distinct attributes in %q", input)
	}

	// A single child redeclaration of an ancestor binding (no same-element
	// duplicate) is a legitimate parseNsClean redundant redeclaration and
	// must still parse.
	acceptClean := []string{
		`<root xmlns="urn:x"><child xmlns="urn:x"/></root>`,
		`<root xmlns:p="urn:x"><child xmlns:p="urn:x"/></root>`,
	}
	for _, input := range acceptClean {
		_, err := helium.NewParser().CleanNamespaces(true).Parse(t.Context(), []byte(input))
		require.NoError(t, err, "Parse with CleanNamespaces should accept redundant redecl in %q", input)
	}
}

func TestParseNamespace(t *testing.T) {
	const input = `<?xml version="1.0"?>
<helium:root xmlns:helium="https://github.com/lestrrat-go/helium">
  <helium:child>foo</helium:child>
</helium:root>`
	p := helium.NewParser()
	doc, err := p.Parse(t.Context(), []byte(input))
	require.NoError(t, err, "Parse should succeed for '%s'", input)

	if pdebug.Enabled {
		pdebug.Dump(doc)
	}

	root := doc.DocumentElement()
	require.NotNil(t, root)
	require.Equal(t, "https://github.com/lestrrat-go/helium", root.URI())

	const attrInput = `<?xml version="1.0"?>
<root xmlns:x="urn:test" x:attr="value"/>`
	doc, err = p.Parse(t.Context(), []byte(attrInput))
	require.NoError(t, err)

	root = doc.DocumentElement()
	require.NotNil(t, root)
	attr := root.GetAttributeNodeNS("attr", "urn:test")
	require.NotNil(t, attr)
	require.Equal(t, "x", attr.Prefix())
	require.Equal(t, "urn:test", attr.URI())
	require.Equal(t, "value", attr.Value())
}

func findDocumentElement(doc *helium.Document) helium.Node {
	return doc.DocumentElement()
}

func TestStripBlanks(t *testing.T) {
	const input = `<?xml version="1.0"?>
<root>
  <child>text</child>
</root>`
	p := helium.NewParser().StripBlanks(true)
	doc, err := p.Parse(t.Context(), []byte(input))
	require.NoError(t, err, "Parse should succeed")

	// With NoBlanks, blank-only text nodes between elements should be stripped.
	// The root element's first child should be <child>, not a whitespace text node.
	root := findDocumentElement(doc)
	require.NotNil(t, root, "document element must exist")
	first := root.FirstChild()
	require.NotNil(t, first, "root must have children")
	require.Equal(t, helium.ElementNode, first.Type(), "first child should be element, not blank text")
}

func TestMergeCDATA(t *testing.T) {
	const input = `<?xml version="1.0"?>
<root><![CDATA[hello]]></root>`

	// Without MergeCDATA: tree should have a CDATA node
	p1 := helium.NewParser()
	doc1, err := p1.Parse(t.Context(), []byte(input))
	require.NoError(t, err, "Parse should succeed")
	root1 := findDocumentElement(doc1)
	require.NotNil(t, root1)
	child1 := root1.FirstChild()
	require.NotNil(t, child1)
	require.Equal(t, helium.CDATASectionNode, child1.Type(), "without MergeCDATA, should be CDATA node")

	// With MergeCDATA: CDATA should be delivered as text
	p2 := helium.NewParser().MergeCDATA(true)
	doc2, err := p2.Parse(t.Context(), []byte(input))
	require.NoError(t, err, "Parse should succeed")
	root2 := findDocumentElement(doc2)
	require.NotNil(t, root2)
	child2 := root2.FirstChild()
	require.NotNil(t, child2)
	require.Equal(t, helium.TextNode, child2.Type(), "with MergeCDATA, CDATA should be a text node")
	require.Equal(t, "hello", string(child2.Content()))
}

func TestParsePedantic(t *testing.T) {
	// Pedantic mode requires absolute URIs in namespace declarations
	const input = `<?xml version="1.0"?>
<root xmlns:foo="relative/uri">
  <foo:child>text</foo:child>
</root>`

	// Without pedantic: should succeed
	p1 := helium.NewParser()
	_, err := p1.Parse(t.Context(), []byte(input))
	require.NoError(t, err, "without pedantic, relative URI should be accepted")

	// With pedantic: should fail (relative URI)
	p2 := helium.NewParser().PedanticErrors(true)
	_, err = p2.Parse(t.Context(), []byte(input))
	require.Error(t, err, "with pedantic, relative URI should be rejected")
}

func TestRecoverOnError(t *testing.T) {
	// Malformed XML: mismatched close tag
	const input = `<?xml version="1.0"?>
<root>
  <child>text</chld>
</root>`

	// Without RecoverOnError: error, no document
	p1 := helium.NewParser()
	doc1, err := p1.Parse(t.Context(), []byte(input))
	require.Error(t, err, "malformed XML should fail")
	require.Nil(t, doc1, "without recover, no document returned")

	// With RecoverOnError: error, but partial document returned
	p2 := helium.NewParser().RecoverOnError(true)
	doc2, err := p2.Parse(t.Context(), []byte(input))
	require.Error(t, err, "malformed XML should still return error")
	require.NotNil(t, doc2, "with recover, partial document should be returned")
}

func TestDisableSAX(t *testing.T) {
	t.Parallel()

	t.Run("recover continues parsing", func(t *testing.T) {
		t.Parallel()

		// XML with a broken sibling element (mismatched end tag) followed by valid content
		const input = `<?xml version="1.0"?>
<root>
  <good>ok</good>
  <bad>text</baaaad>
  <after>more</after>
</root>`

		p := helium.NewParser().RecoverOnError(true)
		doc, err := p.Parse(t.Context(), []byte(input))
		require.Error(t, err, "malformed XML should return error")
		require.NotNil(t, doc, "Recover should return a partial document")

		root := doc.DocumentElement()
		require.NotNil(t, root, "root element should exist")
		require.Equal(t, "root", root.Name())
	})

	t.Run("callbacks suppressed", func(t *testing.T) {
		t.Parallel()

		const input = `<?xml version="1.0"?>
<root>
  <before/>
  <bad>text</baaaad>
  <after/>
</root>`

		var elements []string
		sh := sax.New()
		sh.SetOnStartElementNS(sax.StartElementNSFunc(func(_ context.Context, localname string, _ string, _ string, _ []sax.Namespace, _ []sax.Attribute) error {
			elements = append(elements, localname)
			return nil
		}))

		p := helium.NewParser().SAXHandler(sh).RecoverOnError(true)
		_, err := p.Parse(t.Context(), []byte(input))
		require.Error(t, err)

		// "root" and "before" should have been delivered before the error.
		// "bad" starts parsing (StartElementNS fires before content error).
		// "after" should NOT appear because disableSAX is set after the error.
		require.Contains(t, elements, "root")
		require.Contains(t, elements, "before")
		require.NotContains(t, elements, "after", "elements after error should be suppressed")
	})

	t.Run("no effect without recover", func(t *testing.T) {
		t.Parallel()

		const input = `<?xml version="1.0"?>
<root>
  <bad>text</baaaad>
  <after/>
</root>`

		p := helium.NewParser()
		doc, err := p.Parse(t.Context(), []byte(input))
		require.Error(t, err, "malformed XML should fail")
		require.Nil(t, doc, "without RecoverOnError, no document should be returned")
	})
}

func TestParseExternalEntity(t *testing.T) {
	const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ENTITY ext SYSTEM "ext.xml">
]>
<doc>&ext;</doc>`

	// The external entity is declared in the internal subset and its content is
	// served through the configured FS, exercising the default resolution path.
	fsys := fstest.MapFS{
		"ext.xml": &fstest.MapFile{Data: []byte("<inner>hello</inner>")},
	}

	p := helium.NewParser().SubstituteEntities(true).FS(fsys)
	doc, err := p.Parse(t.Context(), []byte(input))
	require.NoError(t, err, "Parse with external entity should succeed")
	require.NotNil(t, doc, "external entity parse should produce a document")

	var buf bytes.Buffer
	require.NoError(t, helium.NewWriter().WriteTo(&buf, doc))
	out := buf.String()
	require.Contains(t, out, "<inner", "external entity element should be expanded into the document")
	require.Contains(t, out, ">hello</inner>", "external entity content should be expanded into the document")
}

// stringParseInput implements sax.ParseInput for testing.
type stringParseInput struct {
	*strings.Reader
	uri string
}

func newStringParseInput(content, uri string) *stringParseInput {
	return &stringParseInput{Reader: strings.NewReader(content), uri: uri}
}

func (s *stringParseInput) URI() string { return s.uri }

func TestParseExternalEntityMalformedEncoding(t *testing.T) {
	t.Parallel()

	const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ENTITY ext SYSTEM "ext.xml">
]>
<doc>&ext;</doc>`

	// External entity bytes: UTF-16BE BOM, then "<a>" and an unpaired high
	// surrogate (0xD800) before "</a>". The decoder would silently substitute
	// U+FFFD for the surrogate; the parser must instead treat it as fatal,
	// matching the document-level decode-error gate.
	utf16be := func(s string) []byte {
		b := make([]byte, 0, len(s)*2)
		for _, r := range s {
			b = append(b, byte(r>>8), byte(r))
		}
		return b
	}
	ent := []byte{0xFE, 0xFF} // BOM
	ent = append(ent, utf16be("<a>")...)
	ent = append(ent, 0xD8, 0x00) // unpaired high surrogate
	ent = append(ent, utf16be("</a>")...)

	fsys := fstest.MapFS{"ext.xml": &fstest.MapFile{Data: ent}}

	p := helium.NewParser().SubstituteEntities(true).FS(fsys)
	_, err := p.Parse(t.Context(), []byte(input))
	require.Error(t, err, "malformed UTF-16 external entity must fail rather than inserting U+FFFD")
}

func TestParseExternalDTDSizeLimit(t *testing.T) {
	t.Parallel()

	const input = `<?xml version="1.0"?>
<!DOCTYPE r SYSTEM "huge.dtd">
<r/>`

	// An oversized external DTD must be rejected with a parse error rather
	// than being read whole into memory (potential OOM/hang).
	oversized := bytes.Repeat([]byte(" "), helium.MaxExternalDTDSize+1)
	fsys := fstest.MapFS{"huge.dtd": &fstest.MapFile{Data: oversized}}

	p := helium.NewParser().LoadExternalDTD(true).DefaultDTDAttributes(true).FS(fsys)
	_, err := p.Parse(t.Context(), []byte(input))
	require.Error(t, err, "oversized external DTD must produce a parse error")
}

// underReportingFS serves a single DTD file whose Stat under-reports the
// size while Read produces far more than MaxExternalDTDSize. It also counts
// the bytes pulled from the underlying reader so the test can assert the
// read is bounded.
type underReportingFS struct {
	read *int64
}

func (fsys underReportingFS) Open(string) (fs.File, error) {
	return &underReportingFile{read: fsys.read}, nil
}

type underReportingFile struct {
	read *int64
}

// Stat lies: it claims a tiny size so the early precheck does not catch the
// oversized content.
func (f *underReportingFile) Stat() (fs.FileInfo, error) {
	return underReportingInfo{}, nil
}

func (f *underReportingFile) Read(p []byte) (int, error) {
	// Endless stream of spaces; the bounded read must stop it.
	for i := range p {
		p[i] = ' '
	}
	*f.read += int64(len(p))
	return len(p), nil
}

func (f *underReportingFile) Close() error { return nil }

type underReportingInfo struct{}

func (underReportingInfo) Name() string       { return "huge.dtd" }
func (underReportingInfo) Size() int64        { return 1 }
func (underReportingInfo) Mode() fs.FileMode  { return 0 }
func (underReportingInfo) ModTime() time.Time { return time.Time{} }
func (underReportingInfo) IsDir() bool        { return false }
func (underReportingInfo) Sys() any           { return nil }

func TestParseExternalDTDBoundedRead(t *testing.T) {
	t.Parallel()

	const input = `<?xml version="1.0"?>
<!DOCTYPE r SYSTEM "huge.dtd">
<r/>`

	// A source whose Stat under-reports its size must still be rejected by
	// the bounded read, and that read must not consume more than
	// MaxExternalDTDSize+1 bytes from the underlying reader.
	var read int64
	fsys := underReportingFS{read: &read}

	p := helium.NewParser().LoadExternalDTD(true).DefaultDTDAttributes(true).FS(fsys)
	_, err := p.Parse(t.Context(), []byte(input))
	require.Error(t, err, "oversized external DTD must produce a parse error even when Stat under-reports size")
	require.ErrorIs(t, err, helium.ErrExternalDTDTooLarge, "rejection must come from the byte-count cap")
	// The bounded read must consume exactly MaxExternalDTDSize+1 bytes: enough
	// to prove the cap was exceeded, but no more. An implementation that
	// rejected before reading (e.g. trusting an advisory Stat) would leave
	// read==0 and fail the lower bound; one without a cap would overrun it.
	require.Equal(t, int64(helium.MaxExternalDTDSize)+1, read, "bounded read must consume exactly MaxExternalDTDSize+1 bytes")
}

// errReadingFS serves a DTD whose Read returns a full buffer of bytes (taking
// the running total past the configured cap) together with a NON-EOF error on
// the cap-crossing read. The cap must be enforced against the bytes that were
// returned, before the read error is inspected, so the size-cap error still
// fires. A small cap is used so the bounded read does not pull megabytes; the
// fake records whether the simulated read error was actually returned so the
// test can prove the error path was exercised.
type errReadingFS struct {
	cap        int
	hitReadErr *bool
}

func (fsys errReadingFS) Open(string) (fs.File, error) {
	return &errReadingFile{cap: fsys.cap, hitReadErr: fsys.hitReadErr}, nil
}

type errReadingFile struct {
	cap        int
	read       int64
	hitReadErr *bool
}

func (f *errReadingFile) Stat() (fs.FileInfo, error) { return underReportingInfo{}, nil }

func (f *errReadingFile) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = ' '
	}
	f.read += int64(len(p))
	// Once enough bytes have been handed out to reach the cap, return the
	// filled buffer alongside a non-EOF error. A reader that handled the
	// error before checking the byte count would escape the cap.
	if f.read >= int64(f.cap) {
		*f.hitReadErr = true
		return len(p), errors.New("simulated transport failure")
	}
	return len(p), nil
}

func (f *errReadingFile) Close() error { return nil }

func TestParseExternalDTDReadErrorStillCapped(t *testing.T) {
	t.Parallel()

	const input = `<?xml version="1.0"?>
<!DOCTYPE r SYSTEM "huge.dtd">
<r/>`

	// Use a small cap so the bounded read does not pull a 10 MiB stream. The
	// cap-crossing read returns n>0 plus a non-EOF error. The size cap must
	// still fire: the returned bytes already exceed the configured cap.
	const smallCap = 4096
	var hitReadErr bool
	fsys := errReadingFS{cap: smallCap, hitReadErr: &hitReadErr}

	p := helium.NewParser().LoadExternalDTD(true).DefaultDTDAttributes(true).
		MaxExternalDTDBytes(smallCap).FS(fsys)
	_, err := p.Parse(t.Context(), []byte(input))
	require.True(t, hitReadErr, "the simulated non-EOF read error must actually be returned by the fake")
	require.Error(t, err, "oversized external DTD must produce a parse error even when the read also errors")
	require.ErrorIs(t, err, helium.ErrExternalDTDTooLarge, "size cap must be enforced before the read error is handled")
}

// overReportingFS serves a single small, valid DTD whose Stat over-reports the
// size as MaxExternalDTDSize+1. The actual content is well under the cap, so
// the parse must succeed: this proves Stat is advisory and never used to
// reject.
type overReportingFS struct {
	data []byte
}

func (fsys overReportingFS) Open(string) (fs.File, error) {
	return &overReportingFile{data: fsys.data}, nil
}

type overReportingFile struct {
	data []byte
	off  int
}

// Stat lies the other way: it claims a size above the cap even though Read
// yields only a few valid bytes.
func (f *overReportingFile) Stat() (fs.FileInfo, error) {
	return overReportingInfo{}, nil
}

func (f *overReportingFile) Read(p []byte) (int, error) {
	if f.off >= len(f.data) {
		return 0, io.EOF
	}
	n := copy(p, f.data[f.off:])
	f.off += n
	return n, nil
}

func (f *overReportingFile) Close() error { return nil }

type overReportingInfo struct{}

func (overReportingInfo) Name() string       { return "small.dtd" }
func (overReportingInfo) Size() int64        { return helium.MaxExternalDTDSize + 1 }
func (overReportingInfo) Mode() fs.FileMode  { return 0 }
func (overReportingInfo) ModTime() time.Time { return time.Time{} }
func (overReportingInfo) IsDir() bool        { return false }
func (overReportingInfo) Sys() any           { return nil }

func TestParseExternalDTDStatAdvisory(t *testing.T) {
	t.Parallel()

	const input = `<?xml version="1.0"?>
<!DOCTYPE r SYSTEM "small.dtd">
<r/>`

	// A small, valid DTD whose Stat over-reports its size must still load:
	// the cap is enforced against actual bytes read, not the advisory Stat.
	// The DTD defaults an attribute so the test observes that the external
	// subset was actually loaded and applied, not silently skipped.
	fsys := overReportingFS{data: []byte("<!ELEMENT r EMPTY>\n<!ATTLIST r x CDATA \"default\">")}

	p := helium.NewParser().LoadExternalDTD(true).DefaultDTDAttributes(true).FS(fsys)
	doc, err := p.Parse(t.Context(), []byte(input))
	require.NoError(t, err, "small valid DTD must load even when Stat over-reports its size")

	root := doc.DocumentElement()
	require.NotNil(t, root, "root element should exist")
	x, ok := root.GetAttribute("x")
	require.True(t, ok, "external DTD ATTLIST default must be applied, proving the DTD was loaded")
	require.Equal(t, "default", x, "defaulted attribute value must come from the external DTD")
}

func TestParseExternalDTDConfigurableLimit(t *testing.T) {
	t.Parallel()

	const input = `<?xml version="1.0"?>
<!DOCTYPE r SYSTEM "ext.dtd">
<r/>`

	t.Run("custom small limit rejects larger DTD", func(t *testing.T) {
		t.Parallel()

		// A 2 KiB DTD must be rejected when the configured cap is 1 KiB.
		oversized := bytes.Repeat([]byte(" "), 2<<10)
		fsys := fstest.MapFS{"ext.dtd": &fstest.MapFile{Data: oversized}}

		p := helium.NewParser().
			LoadExternalDTD(true).
			DefaultDTDAttributes(true).
			MaxExternalDTDBytes(1 << 10).
			FS(fsys)
		_, err := p.Parse(t.Context(), []byte(input))
		require.Error(t, err, "DTD larger than the configured cap must be rejected")
		require.ErrorIs(t, err, helium.ErrExternalDTDTooLarge, "rejection must come from the byte-count cap")
	})

	t.Run("custom small limit allows smaller DTD", func(t *testing.T) {
		t.Parallel()

		// A DTD well under the 1 KiB cap must still load. It defaults an
		// attribute so the test observes that the DTD was actually applied,
		// not silently skipped.
		fsys := fstest.MapFS{"ext.dtd": &fstest.MapFile{Data: []byte("<!ELEMENT r EMPTY>\n<!ATTLIST r x CDATA \"default\">")}}

		p := helium.NewParser().
			LoadExternalDTD(true).
			DefaultDTDAttributes(true).
			MaxExternalDTDBytes(1 << 10).
			FS(fsys)
		doc, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err, "DTD under the configured cap must load")

		root := doc.DocumentElement()
		require.NotNil(t, root, "root element should exist")
		x, ok := root.GetAttribute("x")
		require.True(t, ok, "external DTD ATTLIST default must be applied, proving the DTD was loaded")
		require.Equal(t, "default", x, "defaulted attribute value must come from the external DTD")
	})

	t.Run("default cap allows a normal DTD over a small custom cap", func(t *testing.T) {
		t.Parallel()

		// Without configuring a custom cap, a DTD larger than 1 KiB (but well
		// under the 10 MiB default) must still load. It defaults an attribute
		// so the test observes that the DTD was actually applied.
		large := append([]byte("<!ELEMENT r EMPTY>\n<!ATTLIST r x CDATA \"default\">"), bytes.Repeat([]byte(" "), 4<<10)...)
		fsys := fstest.MapFS{"ext.dtd": &fstest.MapFile{Data: large}}

		p := helium.NewParser().
			LoadExternalDTD(true).
			DefaultDTDAttributes(true).
			FS(fsys)
		doc, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err, "DTD under the default cap must load")

		root := doc.DocumentElement()
		require.NotNil(t, root, "root element should exist")
		x, ok := root.GetAttribute("x")
		require.True(t, ok, "external DTD ATTLIST default must be applied, proving the DTD was loaded")
		require.Equal(t, "default", x, "defaulted attribute value must come from the external DTD")
	})
}

// partialReadFS serves a small DTD whose Read hands back a few valid bytes and
// then a NON-EOF error (a truncated/partial read) well under the size cap. A
// truncated external subset must surface the read error rather than being
// silently treated as an absent DTD.
type partialReadFS struct {
	prefix []byte
}

func (fsys partialReadFS) Open(string) (fs.File, error) {
	return &partialReadFile{prefix: fsys.prefix}, nil
}

type partialReadFile struct {
	prefix []byte
	done   bool
}

func (f *partialReadFile) Stat() (fs.FileInfo, error) { return underReportingInfo{}, nil }

func (f *partialReadFile) Read(p []byte) (int, error) {
	if f.done {
		return 0, io.ErrUnexpectedEOF
	}
	n := copy(p, f.prefix)
	f.done = true
	return n, io.ErrUnexpectedEOF
}

func (f *partialReadFile) Close() error { return nil }

func TestParseExternalDTDPartialReadErrorSurfaces(t *testing.T) {
	t.Parallel()

	const input = `<?xml version="1.0"?>
<!DOCTYPE r SYSTEM "trunc.dtd">
<r/>`

	// The DTD content is well under the cap but Read returns a non-EOF error,
	// modelling a truncated transport. The parse must fail rather than silently
	// accept the document as if no external subset existed.
	fsys := partialReadFS{prefix: []byte("<!ELEMENT r EMPTY>")}

	p := helium.NewParser().LoadExternalDTD(true).DefaultDTDAttributes(true).FS(fsys)
	_, err := p.Parse(t.Context(), []byte(input))
	require.Error(t, err, "a truncated external DTD read must surface as a parse error")
}

func TestParseExternalDTDMalformedDeclTerminates(t *testing.T) {
	t.Parallel()

	const input = `<?xml version="1.0"?>
<!DOCTYPE r SYSTEM "bogus.dtd">
<r/>`

	// "<!BOGUS" is not a valid markup declaration and may neither advance the
	// cursor nor return an error. The progress guard must turn that into a
	// terminating error instead of an infinite loop.
	fsys := fstest.MapFS{"bogus.dtd": &fstest.MapFile{Data: []byte("<!BOGUS")}}

	p := helium.NewParser().LoadExternalDTD(true).DefaultDTDAttributes(true).FS(fsys)

	done := make(chan struct{})
	var err error
	go func() {
		defer close(done)
		_, err = p.Parse(t.Context(), []byte(input))
	}()

	select {
	case <-done:
		require.Error(t, err, "a malformed external DTD declaration must produce a parse error")
	case <-time.After(10 * time.Second):
		t.Fatal("parsing a malformed external DTD did not terminate (no cursor-progress guard)")
	}
}

func TestParseExternalDTDParameterEntityExpands(t *testing.T) {
	t.Parallel()

	const input = `<?xml version="1.0"?>
<!DOCTYPE r SYSTEM "pe.dtd">
<r/>`

	// The external subset declares a parameter entity whose replacement text is
	// a markup declaration, then references it. The reference must be expanded
	// (not merely validated and skipped) so the <!ATTLIST> takes effect and the
	// default attribute is applied to <r/>. A trailing newline after the
	// reference exercises the progress guard: it must not misfire on valid
	// PE-expanding input.
	const dtd = `<!ELEMENT r EMPTY>
<!ENTITY % defaults "<!ATTLIST r x CDATA 'default'>">
%defaults;
`
	fsys := fstest.MapFS{"pe.dtd": &fstest.MapFile{Data: []byte(dtd)}}

	p := helium.NewParser().LoadExternalDTD(true).DefaultDTDAttributes(true).FS(fsys)
	doc, err := p.Parse(t.Context(), []byte(input))
	require.NoError(t, err, "valid external-subset parameter-entity reference must parse")
	require.NotNil(t, doc, "document must be returned")

	root := doc.DocumentElement()
	require.NotNil(t, root, "root element must be available")

	val, ok := root.GetAttribute("x")
	require.True(t, ok, "default attribute from expanded PE must be present")
	require.Equal(t, "default", val, "expanded PE must apply the default attribute value")
}

func TestParseExternalDTDPEConditionalSectionFollowedByDecl(t *testing.T) {
	t.Parallel()

	const input = `<?xml version="1.0"?>
<!DOCTYPE r SYSTEM "cs.dtd">
<r/>`

	// The external subset declares a parameter entity whose replacement text is
	// a conditional section, references it, and THEN declares another markup
	// declaration. After the PE expands to the conditional section and is
	// exhausted, the loop must resume in the parent DTD and apply the trailing
	// <!ATTLIST>. Before the fix, the conditional-section path "continue"-d past
	// the cursor-cleanup/progress guard, leaving the spent PE cursor on the
	// stack so the next iteration broke the loop and the trailing declaration
	// was silently skipped.
	const dtd = `<!ELEMENT r EMPTY>
<!ENTITY % cs "<![INCLUDE[ <!ELEMENT a EMPTY> ]]>">
%cs;
<!ATTLIST r x CDATA 'd'>
`
	fsys := fstest.MapFS{"cs.dtd": &fstest.MapFile{Data: []byte(dtd)}}

	p := helium.NewParser().LoadExternalDTD(true).DefaultDTDAttributes(true).FS(fsys)
	doc, err := p.Parse(t.Context(), []byte(input))
	require.NoError(t, err, "PE expanding to a conditional section must parse")
	require.NotNil(t, doc, "document must be returned")

	root := doc.DocumentElement()
	require.NotNil(t, root, "root element must be available")

	val, ok := root.GetAttribute("x")
	require.True(t, ok, "declaration following the PE conditional section must be applied")
	require.Equal(t, "d", val, "trailing <!ATTLIST> must not be silently skipped")
}

func TestParseExternalDTDPEWhitespaceFollowedByDecl(t *testing.T) {
	t.Parallel()

	const input = `<?xml version="1.0"?>
<!DOCTYPE r SYSTEM "ws.dtd">
<r/>`

	// The external subset declares a parameter entity whose replacement text is
	// ONLY whitespace, references it, and THEN declares another markup
	// declaration. The blank skip consumes the PE's entire replacement text, so
	// the PE cursor is exhausted by the skip itself. The loop must pop the spent
	// PE cursor and resume in the parent DTD to apply the trailing <!ATTLIST>.
	// Before the fix, the blank-skip's Done()-cursor break exited the loop and
	// the deferred cleanup popped the parent DTD cursor too, silently skipping
	// the trailing declaration.
	const dtd = `<!ELEMENT r EMPTY>
<!ENTITY % ws "   ">
%ws;
<!ATTLIST r x CDATA 'd'>
`
	fsys := fstest.MapFS{"ws.dtd": &fstest.MapFile{Data: []byte(dtd)}}

	p := helium.NewParser().LoadExternalDTD(true).DefaultDTDAttributes(true).FS(fsys)
	doc, err := p.Parse(t.Context(), []byte(input))
	require.NoError(t, err, "PE expanding to only whitespace must parse")
	require.NotNil(t, doc, "document must be returned")

	root := doc.DocumentElement()
	require.NotNil(t, root, "root element must be available")

	val, ok := root.GetAttribute("x")
	require.True(t, ok, "declaration following the whitespace-only PE must be applied")
	require.Equal(t, "d", val, "trailing <!ATTLIST> must not be silently skipped")
}

func TestParseExternalDTDPEInIncludeSectionExpands(t *testing.T) {
	t.Parallel()

	const input = `<?xml version="1.0"?>
<!DOCTYPE r SYSTEM "inc.dtd">
<r/>`

	// The external subset wraps its declarations in an <![INCLUDE[ ... ]]>
	// conditional section. Inside that section it declares a parameter entity
	// whose replacement text is an <!ATTLIST> and then references it. The
	// reference must be expanded (not merely validated and skipped by the
	// blank-skip's handlePEReference) so the default attribute is applied to
	// <r/>. Before the fix, the INCLUDE loop's skipBlanks consumed "%attrs;"
	// without pushing its replacement text, silently dropping the declaration.
	const dtd = `<![INCLUDE[
<!ELEMENT r EMPTY>
<!ENTITY % attrs "<!ATTLIST r x CDATA 'inc'>">
%attrs;
]]>`
	fsys := fstest.MapFS{"inc.dtd": &fstest.MapFile{Data: []byte(dtd)}}

	p := helium.NewParser().LoadExternalDTD(true).DefaultDTDAttributes(true).FS(fsys)
	doc, err := p.Parse(t.Context(), []byte(input))
	require.NoError(t, err, "PE reference inside an INCLUDE section must parse")
	require.NotNil(t, doc, "document must be returned")

	root := doc.DocumentElement()
	require.NotNil(t, root, "root element must be available")

	val, ok := root.GetAttribute("x")
	require.True(t, ok, "default attribute from PE expanded inside INCLUDE must be present")
	require.Equal(t, "inc", val, "PE inside INCLUDE must apply the default attribute value")
}

func TestParseExternalDTDMalformedDeclLocation(t *testing.T) {
	t.Parallel()

	const input = `<?xml version="1.0"?>
<!DOCTYPE r SYSTEM "bogus.dtd">
<r/>`

	// A malformed declaration in the external subset must report the external
	// DTD's location, not the main document's doctype line. The progress-guard
	// error must be raised while the external DTD cursor and baseURI are still
	// active so the reported File carries the DTD path.
	fsys := fstest.MapFS{"bogus.dtd": &fstest.MapFile{Data: []byte("<!BOGUS")}}

	p := helium.NewParser().LoadExternalDTD(true).DefaultDTDAttributes(true).FS(fsys)
	_, err := p.Parse(t.Context(), []byte(input))
	require.Error(t, err, "a malformed external DTD declaration must produce a parse error")

	var pe helium.ErrParseError
	require.ErrorAs(t, err, &pe, "error must be a structured parse error")
	require.Equal(t, "bogus.dtd", pe.File, "error must reference the external DTD, not the main document")
}

func TestParseExternalDTDUnterminatedIncludeNoHang(t *testing.T) {
	t.Parallel()

	const input = `<?xml version="1.0"?>
<!DOCTYPE r SYSTEM "inc.dtd">
<r/>`

	// An external DTD whose <![INCLUDE[ ... ]]> section reaches EOF before its
	// "]]>" terminator must report an error PROMPTLY. The INCLUDE body loop reads
	// the section through the shared declaration step; when that step signals stop
	// (the section's own cursor is exhausted), getCursor() would auto-pop the
	// spent section cursor up to the main document cursor (which is not Done),
	// defeating the EOF check. Honoring the stop signal — and inspecting the floor
	// cursor directly — turns the former infinite loop into a prompt error.
	const dtd = `<![INCLUDE[
<!ELEMENT r EMPTY>`
	fsys := fstest.MapFS{"inc.dtd": &fstest.MapFile{Data: []byte(dtd)}}

	// Guard against a regression manifesting as a hang: run the parse on a
	// goroutine with a deadline so a re-introduced infinite loop fails the test
	// instead of hanging the whole suite. The external subset is tolerant of
	// conditional-section errors (it stops scanning without failing the parse),
	// so the requirement here is PROMPT completion, not a surfaced error.
	done := make(chan struct{})
	go func() {
		defer close(done)
		p := helium.NewParser().LoadExternalDTD(true).DefaultDTDAttributes(true).FS(fsys)
		_, _ = p.Parse(context.Background(), []byte(input))
	}()

	select {
	case <-done:
		// Completed promptly: the unterminated INCLUDE section did not loop.
	case <-time.After(5 * time.Second):
		t.Fatal("parsing an unterminated INCLUDE section hung (infinite loop regression)")
	}
}

func TestParseExternalDTDTrailingWSPreservesPostDoctypeMisc(t *testing.T) {
	t.Parallel()

	const input = `<?xml version="1.0"?>
<!DOCTYPE r SYSTEM "d.dtd"><!--after--><?pi go?><r/>`

	// The external DTD ends with trailing whitespace AFTER its last declaration.
	// The shared declaration step's blank-only skip consumes that whitespace and
	// reaches EOF on the pushed external-DTD (floor) cursor. getCursor() would
	// then auto-pop the exhausted floor cursor and return the cursor BELOW it —
	// the MAIN DOCUMENT cursor positioned right after the DOCTYPE — which is not
	// Done. The step would parse the document's post-DOCTYPE "<!--after-->"
	// comment and "<?pi go?>" PI as if they were external-subset markup, dropping
	// them from the parsed document. Inspecting the floor cursor directly (rather
	// than via getCursor) stops at the floor instead, so the misc nodes survive.
	const dtd = "<!ELEMENT r EMPTY>\n"
	fsys := fstest.MapFS{"d.dtd": &fstest.MapFile{Data: []byte(dtd)}}

	p := helium.NewParser().LoadExternalDTD(true).DefaultDTDAttributes(true).FS(fsys)
	doc, err := p.Parse(t.Context(), []byte(input))
	require.NoError(t, err, "external DTD with trailing whitespace must parse")
	require.NotNil(t, doc, "document must be returned")

	root := doc.DocumentElement()
	require.NotNil(t, root, "root element must be available")

	var sawComment, sawPI bool
	for n := doc.FirstChild(); n != nil; n = n.NextSibling() {
		switch n.Type() {
		case helium.CommentNode:
			require.Equal(t, "after", string(n.Content()), "post-DOCTYPE comment content must be preserved")
			sawComment = true
		case helium.ProcessingInstructionNode:
			sawPI = true
		}
	}
	require.True(t, sawComment, "post-DOCTYPE comment must not be consumed as external-subset markup")
	require.True(t, sawPI, "post-DOCTYPE PI must not be consumed as external-subset markup")
}

func TestParseExternalEntityValidEncoding(t *testing.T) {
	t.Parallel()

	const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ENTITY ext SYSTEM "ext.xml">
]>
<doc>&ext;</doc>`

	// A well-formed UTF-16BE external entity (BOM + "<a/>") must still load.
	utf16be := func(s string) []byte {
		b := make([]byte, 0, len(s)*2)
		for _, r := range s {
			b = append(b, byte(r>>8), byte(r))
		}
		return b
	}
	ent := append([]byte{0xFE, 0xFF}, utf16be("<a/>")...)

	fsys := fstest.MapFS{"ext.xml": &fstest.MapFile{Data: ent}}

	p := helium.NewParser().SubstituteEntities(true).FS(fsys)
	_, err := p.Parse(t.Context(), []byte(input))
	require.NoError(t, err, "well-formed UTF-16 external entity must load")
}

func TestValidateDTD(t *testing.T) {
	t.Parallel()

	t.Run("required attribute missing", func(t *testing.T) {
		t.Parallel()

		// #REQUIRED attribute missing -> validation error
		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc EMPTY>
  <!ATTLIST doc id ID #REQUIRED>
]>
<doc/>`

		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		p := helium.NewParser().ValidateDTD(true).ErrorHandler(collector)
		doc, err := p.Parse(t.Context(), []byte(input))

		require.Error(t, err, "missing #REQUIRED attribute should fail validation")
		require.NotNil(t, doc, "document should still be returned with validation error")
		require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
		require.True(t, containsError(collector.Errors(), "required"))
	})

	t.Run("required attribute present", func(t *testing.T) {
		t.Parallel()

		// #REQUIRED attribute present -> no error
		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc EMPTY>
  <!ATTLIST doc id ID #REQUIRED>
]>
<doc id="x1"/>`

		p := helium.NewParser().ValidateDTD(true)
		_, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err)
	})

	t.Run("fixed mismatch", func(t *testing.T) {
		t.Parallel()

		// #FIXED attribute with wrong value -> validation error
		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc EMPTY>
  <!ATTLIST doc version CDATA #FIXED "1.0">
]>
<doc version="2.0"/>`

		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		p := helium.NewParser().ValidateDTD(true).ErrorHandler(collector)
		_, err := p.Parse(t.Context(), []byte(input))

		require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
		require.True(t, containsError(collector.Errors(), "must be"))
	})

	t.Run("fixed correct", func(t *testing.T) {
		t.Parallel()

		// #FIXED attribute with correct value -> no error
		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc EMPTY>
  <!ATTLIST doc version CDATA #FIXED "1.0">
]>
<doc version="1.0"/>`

		p := helium.NewParser().ValidateDTD(true)
		_, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err)
	})

	t.Run("empty element with content", func(t *testing.T) {
		t.Parallel()

		// EMPTY element with content -> validation error
		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (child)>
  <!ELEMENT child EMPTY>
]>
<doc><child>text</child></doc>`

		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		p := helium.NewParser().ValidateDTD(true).ErrorHandler(collector)
		_, err := p.Parse(t.Context(), []byte(input))

		require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
		require.True(t, containsError(collector.Errors(), "EMPTY"))
	})

	t.Run("element content valid", func(t *testing.T) {
		t.Parallel()

		// Element content model (a, b) with correct content -> no error
		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (a, b)>
  <!ELEMENT a (#PCDATA)>
  <!ELEMENT b (#PCDATA)>
]>
<doc><a>hello</a><b>world</b></doc>`

		p := helium.NewParser().ValidateDTD(true)
		_, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err)
	})

	t.Run("element content mismatch", func(t *testing.T) {
		t.Parallel()

		// Element content model (a, b) with (b, a) -> validation error
		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (a, b)>
  <!ELEMENT a (#PCDATA)>
  <!ELEMENT b (#PCDATA)>
]>
<doc><b>world</b><a>hello</a></doc>`

		p := helium.NewParser().ValidateDTD(true)
		_, err := p.Parse(t.Context(), []byte(input))
		require.Error(t, err, "wrong element order should fail content model")
	})

	t.Run("mixed content valid", func(t *testing.T) {
		t.Parallel()

		// Mixed content (#PCDATA | a)* -- text and <a> are allowed
		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (#PCDATA | a)*>
  <!ELEMENT a (#PCDATA)>
]>
<doc>hello <a>world</a> end</doc>`

		p := helium.NewParser().ValidateDTD(true)
		_, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err)
	})

	t.Run("mixed content bad child", func(t *testing.T) {
		t.Parallel()

		// Mixed content (#PCDATA | a)* -- <b> is NOT allowed
		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (#PCDATA | a)*>
  <!ELEMENT a (#PCDATA)>
  <!ELEMENT b (#PCDATA)>
]>
<doc>hello <b>world</b></doc>`

		p := helium.NewParser().ValidateDTD(true)
		_, err := p.Parse(t.Context(), []byte(input))
		require.Error(t, err, "<b> not allowed in mixed content (a)")
	})

	t.Run("no flag skips validation", func(t *testing.T) {
		t.Parallel()

		// Same invalid document but WITHOUT ValidateDTD -> should pass
		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc EMPTY>
  <!ATTLIST doc id ID #REQUIRED>
]>
<doc/>`

		p := helium.NewParser()
		// Don't set ValidateDTD
		_, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err, "without ValidateDTD, validation should not run")
	})

	t.Run("choice content", func(t *testing.T) {
		t.Parallel()

		// Choice content model (a | b) with <a> -> valid
		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (a | b)>
  <!ELEMENT a (#PCDATA)>
  <!ELEMENT b (#PCDATA)>
]>
<doc><a>hello</a></doc>`

		p := helium.NewParser().ValidateDTD(true)
		_, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err)
	})

	t.Run("repeat content", func(t *testing.T) {
		t.Parallel()

		// Repetition content model (a)+ with multiple <a> -> valid
		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (a)+>
  <!ELEMENT a (#PCDATA)>
]>
<doc><a>1</a><a>2</a><a>3</a></doc>`

		p := helium.NewParser().ValidateDTD(true)
		_, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err)
	})

	t.Run("repeat content empty", func(t *testing.T) {
		t.Parallel()

		// Repetition content model (a)+ with zero <a> -> invalid
		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (a)+>
  <!ELEMENT a (#PCDATA)>
]>
<doc></doc>`

		p := helium.NewParser().ValidateDTD(true)
		_, err := p.Parse(t.Context(), []byte(input))
		require.Error(t, err, "(a)+ requires at least one <a>")
	})

	t.Run("ID unique", func(t *testing.T) {
		t.Parallel()

		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (item, item)>
  <!ELEMENT item EMPTY>
  <!ATTLIST item id ID #REQUIRED>
]>
<doc><item id="a"/><item id="b"/></doc>`

		p := helium.NewParser().ValidateDTD(true)
		_, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err)
	})

	t.Run("ID duplicate", func(t *testing.T) {
		t.Parallel()

		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (item, item)>
  <!ELEMENT item EMPTY>
  <!ATTLIST item id ID #REQUIRED>
]>
<doc><item id="a"/><item id="a"/></doc>`

		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		p := helium.NewParser().ValidateDTD(true).ErrorHandler(collector)
		_, err := p.Parse(t.Context(), []byte(input))

		require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
		require.True(t, containsError(collector.Errors(), "duplicate ID"))
	})

	t.Run("IDRef valid", func(t *testing.T) {
		t.Parallel()

		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (item, ref)>
  <!ELEMENT item EMPTY>
  <!ELEMENT ref EMPTY>
  <!ATTLIST item id ID #REQUIRED>
  <!ATTLIST ref target IDREF #REQUIRED>
]>
<doc><item id="x"/><ref target="x"/></doc>`

		p := helium.NewParser().ValidateDTD(true)
		_, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err)
	})

	t.Run("IDRef missing", func(t *testing.T) {
		t.Parallel()

		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (item, ref)>
  <!ELEMENT item EMPTY>
  <!ELEMENT ref EMPTY>
  <!ATTLIST item id ID #REQUIRED>
  <!ATTLIST ref target IDREF #REQUIRED>
]>
<doc><item id="x"/><ref target="y"/></doc>`

		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		p := helium.NewParser().ValidateDTD(true).ErrorHandler(collector)
		_, err := p.Parse(t.Context(), []byte(input))

		require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
		require.True(t, containsError(collector.Errors(), "unknown ID"))
	})

	t.Run("IDRefs valid", func(t *testing.T) {
		t.Parallel()

		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (item, item, refs)>
  <!ELEMENT item EMPTY>
  <!ELEMENT refs EMPTY>
  <!ATTLIST item id ID #REQUIRED>
  <!ATTLIST refs targets IDREFS #REQUIRED>
]>
<doc><item id="a"/><item id="b"/><refs targets="a b"/></doc>`

		p := helium.NewParser().ValidateDTD(true)
		_, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err)
	})

	t.Run("IDRefs missing", func(t *testing.T) {
		t.Parallel()

		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (item, refs)>
  <!ELEMENT item EMPTY>
  <!ELEMENT refs EMPTY>
  <!ATTLIST item id ID #REQUIRED>
  <!ATTLIST refs targets IDREFS #REQUIRED>
]>
<doc><item id="a"/><refs targets="a z"/></doc>`

		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		p := helium.NewParser().ValidateDTD(true).ErrorHandler(collector)
		_, err := p.Parse(t.Context(), []byte(input))

		require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
		require.True(t, containsError(collector.Errors(), "unknown ID"))
	})
}

func TestDTDDuplicateEnumerationTokens(t *testing.T) {
	t.Parallel()

	t.Run("enumeration with duplicate token", func(t *testing.T) {
		t.Parallel()

		const input = `<?xml version="1.0"?>
<!DOCTYPE r [<!ATTLIST r color (red|red) "red">]>
<r/>`

		p := helium.NewParser()
		_, err := p.Parse(t.Context(), []byte(input))

		require.Error(t, err, "duplicate enumeration token should be rejected")
		var dup helium.DTDDupTokenError
		require.True(t, errors.As(err, &dup), "error should be DTDDupTokenError")
		require.Equal(t, "red", dup.Name)
	})

	t.Run("notation with duplicate token", func(t *testing.T) {
		t.Parallel()

		const input = `<?xml version="1.0"?>
<!DOCTYPE r [
  <!NOTATION n PUBLIC "pub-n">
  <!ATTLIST r kind NOTATION (n|n) #IMPLIED>
]>
<r/>`

		p := helium.NewParser()
		_, err := p.Parse(t.Context(), []byte(input))

		require.Error(t, err, "duplicate notation token should be rejected")
		var dup helium.DTDDupTokenError
		require.True(t, errors.As(err, &dup), "error should be DTDDupTokenError")
		require.Equal(t, "n", dup.Name)
	})

	t.Run("enumeration with distinct tokens", func(t *testing.T) {
		t.Parallel()

		const input = `<?xml version="1.0"?>
<!DOCTYPE r [<!ATTLIST r color (red|green) "red">]>
<r/>`

		p := helium.NewParser()
		_, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err, "distinct enumeration tokens should parse")
	})
}

func TestParseInNodeContext(t *testing.T) {
	t.Run("basic fragment", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root/>`))
		require.NoError(t, err)

		root := doc.DocumentElement()
		require.NotNil(t, root)

		result, err := helium.NewParser().ParseInNodeContext(t.Context(), root, []byte(`<child/>`))
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, helium.ElementNode, result.Type())
		require.Equal(t, "child", result.Name())
		require.Nil(t, result.Parent())
	})

	t.Run("multiple sibling nodes", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root/>`))
		require.NoError(t, err)

		root := doc.DocumentElement()
		result, err := helium.NewParser().ParseInNodeContext(t.Context(), root, []byte(`<a/><b/>text`))
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, "a", result.Name())

		sib := result.NextSibling()
		require.NotNil(t, sib)
		require.Equal(t, "b", sib.Name())

		text := sib.NextSibling()
		require.NotNil(t, text)
		require.Equal(t, helium.TextNode, text.Type())
		require.Equal(t, "text", string(text.Content()))
	})

	t.Run("namespace inheritance", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root xmlns:ns="http://example.com/ns"><child/></root>`))
		require.NoError(t, err)

		root := doc.DocumentElement()
		result, err := helium.NewParser().ParseInNodeContext(t.Context(), root, []byte(`<ns:item>hello</ns:item>`))
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, helium.ElementNode, result.Type())
		// The element should have been parsed successfully using the inherited ns prefix
		elem, ok := result.(*helium.Element)
		require.True(t, ok)
		require.Equal(t, "ns:item", elem.Name())
	})

	t.Run("nested namespace inheritance", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root xmlns:a="http://a.example.com"><middle xmlns:b="http://b.example.com"><child/></middle></root>`))
		require.NoError(t, err)

		root := doc.DocumentElement()
		middle := root.FirstChild()
		require.NotNil(t, middle)

		// Parse fragment in context of middle -- should see both a: and b: prefixes
		result, err := helium.NewParser().ParseInNodeContext(t.Context(), middle, []byte(`<a:x/><b:y/>`))
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, "a:x", result.Name())
		sib := result.NextSibling()
		require.NotNil(t, sib)
		require.Equal(t, "b:y", sib.Name())
	})

	t.Run("fragment with own namespaces", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root/>`))
		require.NoError(t, err)

		root := doc.DocumentElement()
		result, err := helium.NewParser().ParseInNodeContext(t.Context(), root, []byte(`<ns:item xmlns:ns="http://example.com/ns">hello</ns:item>`))
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, "ns:item", result.Name())
	})

	t.Run("document as context", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root/>`))
		require.NoError(t, err)

		result, err := helium.NewParser().ParseInNodeContext(t.Context(), doc, []byte(`<elem/>`))
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, "elem", result.Name())
	})

	t.Run("non-element context walks up", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root xmlns:ns="http://example.com/ns"><child>some text</child></root>`))
		require.NoError(t, err)

		root := doc.DocumentElement()
		child := root.FirstChild()
		require.NotNil(t, child)
		textNode := child.FirstChild()
		require.NotNil(t, textNode)
		require.Equal(t, helium.TextNode, textNode.Type())

		// Parse in context of text node -- should walk up to <child> then <root>
		result, err := helium.NewParser().ParseInNodeContext(t.Context(), textNode, []byte(`<ns:item/>`))
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, "ns:item", result.Name())
	})

	t.Run("DTD entity resolution", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ENTITY greeting "hello world">
]>
<doc/>`))
		require.NoError(t, err)

		root := doc.DocumentElement()
		p := helium.NewParser().SubstituteEntities(true)
		result, err := p.ParseInNodeContext(t.Context(), root, []byte(`<item>&greeting;</item>`))
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, "item", result.Name())
		// The entity should have been resolved
		require.Equal(t, "hello world", string(result.FirstChild().Content()))
	})

	t.Run("empty fragment", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root/>`))
		require.NoError(t, err)

		root := doc.DocumentElement()
		result, err := helium.NewParser().ParseInNodeContext(t.Context(), root, []byte(``))
		require.NoError(t, err)
		require.Nil(t, result)
	})

	t.Run("nil node", func(t *testing.T) {
		_, err := helium.NewParser().ParseInNodeContext(t.Context(), nil, []byte(`<child/>`))
		require.Error(t, err)
	})

	t.Run("original document preserved", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><existing/></root>`))
		require.NoError(t, err)

		root := doc.DocumentElement()
		_, err = helium.NewParser().ParseInNodeContext(t.Context(), root, []byte(`<new/>`))
		require.NoError(t, err)

		// Original document should still have its children
		require.NotNil(t, doc.FirstChild())
		docRoot := doc.DocumentElement()
		require.NotNil(t, docRoot)
		require.Equal(t, "root", docRoot.Name())
		require.NotNil(t, docRoot.FirstChild())
		require.Equal(t, "existing", docRoot.FirstChild().Name())
	})
}

func TestBlockXXE(t *testing.T) {
	t.Parallel()

	t.Run("entity", func(t *testing.T) {
		t.Parallel()

		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ENTITY ext SYSTEM "ext.xml">
]>
<doc>&ext;</doc>`

		resolved := false
		s := sax.New()
		s.SetOnResolveEntity(sax.ResolveEntityFunc(func(_ context.Context, publicID, systemID string) (sax.ParseInput, error) {
			resolved = true
			return newStringParseInput("<inner>hello</inner>", systemID), nil
		}))

		p := helium.NewParser().SAXHandler(s).SubstituteEntities(true).BlockXXE(true)
		_, err := p.Parse(t.Context(), []byte(input))
		// With BlockXXE, external entity loading is blocked.
		// The entity reference remains unresolved; no error but external content not loaded.
		_ = err
		require.False(t, resolved, "ResolveEntity should not be called with BlockXXE")
	})

	t.Run("external DTD", func(t *testing.T) {
		t.Parallel()

		const input = `<?xml version="1.0"?>
<!DOCTYPE doc SYSTEM "ext.dtd">
<doc/>`

		resolved := false
		s := sax.New()
		s.SetOnResolveEntity(sax.ResolveEntityFunc(func(_ context.Context, publicID, systemID string) (sax.ParseInput, error) {
			resolved = true
			return newStringParseInput("<!ELEMENT doc EMPTY>", systemID), nil
		}))

		p := helium.NewParser().SAXHandler(s).LoadExternalDTD(true).BlockXXE(true)
		_, err := p.Parse(t.Context(), []byte(input))
		_ = err
		require.False(t, resolved, "external DTD should not be loaded with BlockXXE")
	})
}

func TestParserFS(t *testing.T) {
	t.Parallel()

	t.Run("external DTD loaded from custom FS", func(t *testing.T) {
		t.Parallel()

		const input = `<?xml version="1.0"?>
<!DOCTYPE doc SYSTEM "ext.dtd">
<doc/>`

		fsys := fstest.MapFS{
			"ext.dtd": &fstest.MapFile{Data: []byte("<!ELEMENT doc EMPTY>\n")},
		}

		p := helium.NewParser().LoadExternalDTD(true).FS(fsys)
		_, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err)
	})

	t.Run("FS error surfaces as missing resource (silent)", func(t *testing.T) {
		t.Parallel()

		const input = `<?xml version="1.0"?>
<!DOCTYPE doc SYSTEM "nope.dtd">
<doc/>`

		p := helium.NewParser().LoadExternalDTD(true).FS(fstest.MapFS{})
		_, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err, "missing external DTD is silently ignored, same as before")
	})

	t.Run("nil FS restores default", func(t *testing.T) {
		t.Parallel()

		const input = `<?xml version="1.0"?><doc/>`

		// Set a custom FS then clear it; parsing a doc that needs no external
		// resources must still work.
		p := helium.NewParser().FS(fstest.MapFS{}).FS(nil)
		_, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err)
	})

	// Compile-time check that fs.FS is the parameter type.
	var _ = helium.NewParser().FS(fs.FS(fstest.MapFS{}))
}

// recordingFS wraps an fs.FS and records every path passed to Open, so a test
// can assert which resources a parse attempted to load.
type recordingFS struct {
	inner  fs.FS
	mu     sync.Mutex
	opened []string
}

func (r *recordingFS) Open(name string) (fs.File, error) {
	r.mu.Lock()
	r.opened = append(r.opened, name)
	r.mu.Unlock()
	return r.inner.Open(name)
}

func (r *recordingFS) wasOpened(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Contains(r.opened, name)
}

// TestEntitySubParserFSSandbox guards against a sandbox escape where an external
// entity reached from inside another external entity's sub-parse was resolved
// via the default permissive os.Open path instead of the parser's configured FS.
func TestEntitySubParserFSSandbox(t *testing.T) {
	t.Parallel()

	// A real on-disk file outside any configured sandbox. If the nested external
	// entity escapes the FS via os.Open, its contents leak into the document.
	dir := t.TempDir()
	secretPath := filepath.Join(dir, "secret.xml")
	require.NoError(t, os.WriteFile(secretPath, []byte("<leaked>TOPSECRET</leaked>"), 0o600))

	t.Run("nested external entity confined to configured FS", func(t *testing.T) {
		t.Parallel()

		// outer.xml lives inside the sandbox and references &secret;, which is an
		// external SYSTEM entity pointing at the absolute on-disk path OUTSIDE the
		// sandbox. The sub-parse of outer.xml must resolve &secret; through the
		// same configured FS, which does not contain that path, so it must not be
		// readable and must not leak into the document.
		input := `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ENTITY secret SYSTEM "` + secretPath + `">
  <!ENTITY outer SYSTEM "outer.xml">
]>
<doc>&outer;</doc>`

		rfs := &recordingFS{inner: fstest.MapFS{
			"outer.xml": &fstest.MapFile{Data: []byte(`<wrap>&secret;</wrap>`)},
		}}
		p := helium.NewParser().SubstituteEntities(true).FS(rfs)
		doc, _ := p.Parse(t.Context(), []byte(input))

		// The on-disk secret must never surface in the resulting document.
		if doc != nil {
			var buf bytes.Buffer
			require.NoError(t, helium.NewWriter().WriteTo(&buf, doc))
			require.NotContains(t, buf.String(), "TOPSECRET",
				"out-of-sandbox file leaked into document")
		}
		// Resolution of the nested external entity must be routed through the
		// configured FS (recorded here). A leak would happen via os.Open, which
		// bypasses the recording FS entirely, so the path would never be seen.
		require.True(t, rfs.wasOpened(secretPath),
			"nested external entity escaped the configured FS sandbox")
	})

	t.Run("in-sandbox nested external entity still resolves", func(t *testing.T) {
		t.Parallel()

		// A legitimate external entity available within the configured FS must
		// still resolve when reached from inside another external entity.
		input := `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ENTITY allowed SYSTEM "allowed.xml">
  <!ENTITY outer SYSTEM "outer.xml">
]>
<doc>&outer;</doc>`

		rfs := &recordingFS{inner: fstest.MapFS{
			"outer.xml":   &fstest.MapFile{Data: []byte(`<wrap>&allowed;</wrap>`)},
			"allowed.xml": &fstest.MapFile{Data: []byte("<inner>ok</inner>")},
		}}
		p := helium.NewParser().SubstituteEntities(true).FS(rfs)
		doc, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err)
		require.True(t, rfs.wasOpened("allowed.xml"),
			"in-sandbox nested external entity was not loaded through the configured FS")

		var buf bytes.Buffer
		require.NoError(t, helium.NewWriter().WriteTo(&buf, doc))
		out := buf.String()
		require.Contains(t, out, "<inner", "in-sandbox nested external entity did not expand")
		require.Contains(t, out, ">ok</inner>", "in-sandbox nested external entity content missing")
	})
}

func TestSkipIDs(t *testing.T) {
	t.Parallel()

	const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc EMPTY>
  <!ATTLIST doc id ID #IMPLIED>
  <!ATTLIST doc name CDATA #IMPLIED>
]>
<doc id="x1" name="n1"/>`

	p := helium.NewParser().LoadExternalDTD(true).DefaultDTDAttributes(true).SkipIDs(true)
	doc, err := p.Parse(t.Context(), []byte(input))
	require.NoError(t, err)

	require.Nil(t, doc.GetElementByID("x1"), "ID should not be interned when SkipIDs is set")
	root := doc.DocumentElement()
	require.NotNil(t, root)
	name, ok := root.GetAttribute("name")
	require.True(t, ok, "non-ID attributes should still be available")
	require.Equal(t, "n1", name)
}

func TestEntityBoundary(t *testing.T) {
	t.Parallel()

	t.Run("element decl", func(t *testing.T) {
		t.Parallel()

		// PE starts the element declaration but the closing '>' is in the main DTD.
		// This crosses an entity boundary -> parse error (syntax or boundary).
		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ENTITY % start "<!ELEMENT doc EMPTY">
  %start;>
]>
<doc/>`

		p := helium.NewParser().LoadExternalDTD(true)
		_, err := p.Parse(t.Context(), []byte(input))
		require.Error(t, err, "boundary-violating PE should cause a parse error")
	})

	t.Run("attribute list decl", func(t *testing.T) {
		t.Parallel()

		// PE starts the ATTLIST declaration but the closing '>' is in the main DTD.
		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc EMPTY>
  <!ENTITY % start "<!ATTLIST doc attr CDATA #IMPLIED">
  %start;>
]>
<doc/>`

		p := helium.NewParser().LoadExternalDTD(true)
		_, err := p.Parse(t.Context(), []byte(input))
		require.Error(t, err, "boundary-violating PE should cause a parse error")
	})

	t.Run("well-nested PE", func(t *testing.T) {
		t.Parallel()

		// PE expands to a complete declaration -- no boundary violation.
		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ENTITY % decl "<!ELEMENT doc EMPTY>">
  %decl;
]>
<doc/>`

		p := helium.NewParser().LoadExternalDTD(true)
		doc, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err)
		require.NotNil(t, doc)
	})
}

func TestCurrentInputID(t *testing.T) {
	t.Parallel()

	const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ENTITY x "hello">
]>
<doc>&x;</doc>`

	p := helium.NewParser().SubstituteEntities(true)
	doc, err := p.Parse(t.Context(), []byte(input))
	require.NoError(t, err)
	require.NotNil(t, doc)
	require.Equal(t, "hello", string(doc.DocumentElement().Content()))
}

func TestConditionalSection(t *testing.T) {
	t.Parallel()

	t.Run("include", func(t *testing.T) {
		t.Parallel()

		// INCLUDE section via PE expansion: element declarations should be applied.
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
		doc, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err, "INCLUDE conditional section should parse successfully")
		require.NotNil(t, doc)

		root := doc.DocumentElement()
		require.NotNil(t, root)
		require.Equal(t, "doc", root.Name())
	})

	t.Run("ignore", func(t *testing.T) {
		t.Parallel()

		// Conditional sections in internal subset must come via PE expansion.
		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ENTITY % ign "IGNORE">
  <!ENTITY % sect "<![%ign;[<!ELEMENT doc (nonexistent)>]]>">
  %sect;
  <!ELEMENT doc (#PCDATA)>
]>
<doc>hello</doc>`

		p := helium.NewParser()
		_, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err, "IGNORE section content should be skipped")
	})

	t.Run("internal subset PE", func(t *testing.T) {
		t.Parallel()

		// Internal subset with PE that expands to conditional section content.
		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ENTITY % inc "INCLUDE">
  <!ENTITY % content "<![%inc;[<!ELEMENT doc (#PCDATA)>]]>">
  %content;
]>
<doc>hello</doc>`

		p := helium.NewParser()
		_, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err, "PE-expanded conditional section in internal subset should work")
	})

	t.Run("errors", func(t *testing.T) {
		t.Parallel()

		t.Run("invalid keyword", func(t *testing.T) {
			t.Parallel()

			const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ENTITY % kw "BOGUS">
  <!ENTITY % sect "<![%kw;[<!ELEMENT doc (#PCDATA)>]]>">
  %sect;
]>
<doc/>`
			p := helium.NewParser()
			_, err := p.Parse(t.Context(), []byte(input))
			require.Error(t, err, "invalid keyword should fail")
		})
	})

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
}

func TestXMLSpacePreserve(t *testing.T) {
	t.Run("preserve keeps whitespace with StripBlanks", func(t *testing.T) {
		const input = `<?xml version="1.0"?>
<root xml:space="preserve">
  <child>text</child>
</root>`
		p := helium.NewParser().StripBlanks(true)
		doc, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err, "Parse should succeed")

		root := findDocumentElement(doc)
		require.NotNil(t, root, "document element must exist")
		first := root.FirstChild()
		require.NotNil(t, first, "root must have children")
		// With xml:space="preserve", blank-only text nodes must be kept even with StripBlanks.
		require.Equal(t, helium.TextNode, first.Type(), "first child should be text node (preserved whitespace)")
	})

	t.Run("default reverts preserve", func(t *testing.T) {
		// xml:space="default" on an element should cause blanks to be stripped
		// even when a parent had xml:space="preserve".
		// Note: libxml2's spaceTab is per-element (not inherited), so only
		// the element with the explicit attribute is affected.
		const input = `<?xml version="1.0"?>
<root xml:space="preserve">
  <child xml:space="default">
    <leaf>text</leaf>
  </child>
</root>`
		p := helium.NewParser().StripBlanks(true)
		doc, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err, "Parse should succeed")

		root := findDocumentElement(doc)
		require.NotNil(t, root, "document element must exist")
		// root has xml:space="preserve", so its whitespace text nodes are kept
		require.Equal(t, helium.TextNode, root.FirstChild().Type(), "root whitespace should be preserved")

		// Find <child>
		var child helium.Node
		for c := root.FirstChild(); c != nil; c = c.NextSibling() {
			if c.Type() == helium.ElementNode && c.(*helium.Element).LocalName() == "child" {
				child = c
				break
			}
		}
		require.NotNil(t, child, "child element must exist")
		// child has xml:space="default", so blanks should be stripped
		first := child.FirstChild()
		require.NotNil(t, first, "child must have children")
		require.Equal(t, helium.ElementNode, first.Type(), "child's first child should be element (blanks stripped by default)")
	})

	t.Run("preserve pops correctly after element closes", func(t *testing.T) {
		const input = `<?xml version="1.0"?>
<root>
  <preserved xml:space="preserve">
    <child>text</child>
  </preserved>
  <normal>
    <child>text</child>
  </normal>
</root>`
		p := helium.NewParser().StripBlanks(true)
		doc, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err, "Parse should succeed")

		root := findDocumentElement(doc)
		require.NotNil(t, root, "document element must exist")

		// Find <preserved> and <normal>
		var preserved, normal helium.Node
		for c := root.FirstChild(); c != nil; c = c.NextSibling() {
			if c.Type() == helium.ElementNode {
				switch c.(*helium.Element).LocalName() {
				case "preserved":
					preserved = c
				case "normal":
					normal = c
				}
			}
		}
		require.NotNil(t, preserved, "preserved element must exist")
		require.NotNil(t, normal, "normal element must exist")

		// preserved's first child should be whitespace text
		require.Equal(t, helium.TextNode, preserved.FirstChild().Type(), "preserved whitespace should be kept")

		// normal's first child should be element (blanks stripped)
		require.Equal(t, helium.ElementNode, normal.FirstChild().Type(), "normal whitespace should be stripped")
	})
}

func parseWithDTDAttributeType(t *testing.T, typ enum.AttributeType, value string) error {
	t.Helper()

	var docDecl string
	var extraDecl string
	var body string
	var typeName string

	switch typ {
	case enum.AttrID:
		docDecl = "<!ELEMENT doc EMPTY>"
		body = fmt.Sprintf(`<doc attr=%q/>`, value)
		typeName = "ID"
	case enum.AttrNmtoken:
		docDecl = "<!ELEMENT doc EMPTY>"
		body = fmt.Sprintf(`<doc attr=%q/>`, value)
		typeName = "NMTOKEN"
	case enum.AttrNmtokens:
		docDecl = "<!ELEMENT doc EMPTY>"
		body = fmt.Sprintf(`<doc attr=%q/>`, value)
		typeName = "NMTOKENS"
	case enum.AttrIDRefs:
		docDecl = "<!ELEMENT doc (item*)>"
		extraDecl = "<!ELEMENT item EMPTY>\n  <!ATTLIST item id ID #IMPLIED>"
		body = fmt.Sprintf(`<doc attr=%q><item id="id1"/><item id="id2"/></doc>`, value)
		typeName = "IDREFS"
	case enum.AttrCDATA:
		docDecl = "<!ELEMENT doc EMPTY>"
		body = fmt.Sprintf(`<doc attr=%q/>`, value)
		typeName = "CDATA"
	default:
		t.Fatalf("unsupported attr type: %v", typ)
	}

	input := fmt.Sprintf(`<?xml version="1.0"?>
<!DOCTYPE doc [
  %s
  %s
  <!ATTLIST doc attr %s #IMPLIED>
]>
%s`, docDecl, extraDecl, typeName, body)

	p := helium.NewParser().ValidateDTD(true)
	_, err := p.Parse(t.Context(), []byte(input))
	return err
}

func TestValidateAttributeValueInternal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		typ     enum.AttributeType
		value   string
		wantErr bool
	}{
		{name: "ID valid", typ: enum.AttrID, value: "myid"},
		{name: "ID invalid", typ: enum.AttrID, value: "123", wantErr: true},
		{name: "NMTOKEN valid", typ: enum.AttrNmtoken, value: "hello-world"},
		{name: "NMTOKEN valid digits", typ: enum.AttrNmtoken, value: "123"},
		{name: "NMTOKEN invalid", typ: enum.AttrNmtoken, value: "hello world", wantErr: true},
		{name: "NMTOKENS valid", typ: enum.AttrNmtokens, value: "hello world"},
		{name: "IDREFS valid", typ: enum.AttrIDRefs, value: "id1 id2"},
		{name: "IDREFS invalid", typ: enum.AttrIDRefs, value: "id1 123", wantErr: true},
		{name: "CDATA anything", typ: enum.AttrCDATA, value: "anything goes here!"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := parseWithDTDAttributeType(t, tc.typ, tc.value)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestMaxDepth(t *testing.T) {
	t.Parallel()

	t.Run("exceeded", func(t *testing.T) {
		t.Parallel()

		input := []byte(strings.Repeat("<a>", 10) + strings.Repeat("</a>", 10))
		p := helium.NewParser().MaxDepth(5)

		_, err := p.Parse(t.Context(), input)
		require.Error(t, err)
		require.Contains(t, err.Error(), "exceeded max depth")
	})

	t.Run("within limit", func(t *testing.T) {
		t.Parallel()

		input := []byte(strings.Repeat("<a>", 5) + "hello" + strings.Repeat("</a>", 5))
		p := helium.NewParser().MaxDepth(10)

		doc, err := p.Parse(t.Context(), input)
		require.NoError(t, err)
		require.NotNil(t, doc)
	})

	t.Run("exact limit", func(t *testing.T) {
		t.Parallel()

		input := []byte(strings.Repeat("<a>", 5) + "hello" + strings.Repeat("</a>", 5))
		p := helium.NewParser().MaxDepth(5)

		doc, err := p.Parse(t.Context(), input)
		require.NoError(t, err)
		require.NotNil(t, doc)
	})

	t.Run("zero is unlimited", func(t *testing.T) {
		t.Parallel()

		input := []byte(strings.Repeat("<a>", 100) + "hello" + strings.Repeat("</a>", 100))
		p := helium.NewParser()

		doc, err := p.Parse(t.Context(), input)
		require.NoError(t, err)
		require.NotNil(t, doc)
	})

	t.Run("via ParseReader", func(t *testing.T) {
		t.Parallel()

		input := strings.Repeat("<a>", 10) + strings.Repeat("</a>", 10)
		p := helium.NewParser().MaxDepth(5)

		_, err := p.ParseReader(t.Context(), bytes.NewReader([]byte(input)))
		require.Error(t, err)
		require.Contains(t, err.Error(), "exceeded max depth")
	})
}

func TestParseLenientXMLDecl(t *testing.T) {
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
}

func TestParseLenientXMLDeclRejectsWithoutFlag(t *testing.T) {
	input := `<?xml encoding="utf-8" version="1.0"?><root />`
	p := helium.NewParser()
	_, err := p.Parse(t.Context(), []byte(input))
	require.Error(t, err, "strict parser should reject encoding before version")
}

func TestParseNameRejectsInvalidUTF8InContinuation(t *testing.T) {
	xml := []byte("<ro\xffoot/>")
	p := helium.NewParser()
	_, err := p.Parse(t.Context(), xml)
	require.Error(t, err)
}

func TestParseNCNameRejectsInvalidUTF8InContinuation(t *testing.T) {
	xml := []byte("<root at\xffr=\"v\"/>")
	p := helium.NewParser()
	_, err := p.Parse(t.Context(), xml)
	require.Error(t, err)
}

func TestParseNCNameReportsInvalidStartRune(t *testing.T) {
	xml := []byte("<root 1a=\"v\"/>")
	p := helium.NewParser()
	_, err := p.Parse(t.Context(), xml)
	require.Error(t, err)
}

// dataThenErrReader returns its payload together with a non-EOF error on the
// same Read (which io.Reader permits), then reports EOF. It models a reader
// that detects corruption/truncation only after emitting the final bytes,
// e.g. a checksumming or decompressing stream.
type dataThenErrReader struct {
	data []byte
	err  error
	done bool
}

func (r *dataThenErrReader) Read(p []byte) (int, error) {
	if r.done {
		return 0, io.EOF
	}
	r.done = true
	n := copy(p, r.data)
	return n, r.err
}

func TestParseReaderSurfacesErrorReturnedWithData(t *testing.T) {
	wantErr := errors.New("checksum mismatch")
	p := helium.NewParser()

	_, err := p.ParseReader(t.Context(), &dataThenErrReader{
		data: []byte("<root/>"),
		err:  wantErr,
	})
	require.ErrorIs(t, err, wantErr, "a reader error returned alongside the final bytes must not be swallowed")
}
