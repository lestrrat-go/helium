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
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"
	"unicode/utf8"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
	heliumencoding "github.com/lestrrat-go/helium/internal/encoding"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/sax"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/encoding/charmap"
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

func TestParseRejectsMissingSpaceBetweenAttributes(t *testing.T) {
	// XML §3.1 P40/P44: attributes in a start/empty-element tag must be
	// separated by whitespace ('(S Attribute)*'). Two attributes written back
	// to back with no intervening S are a fatal well-formedness error. This
	// covers W3C xml-suite cases sun/attlist10, sun/attlist11, oasis/p40fail1,
	// oasis/p44fail4 and xmltest/not-wf/sa/186.
	reject := []string{
		// STag (P40) and EmptyElemTag (P44), no DTD.
		`<doc att="val"att2="val2"></doc>`,
		`<doc att="val"att2="val2"/>`,
		// With an internal DTD declaring the attributes (sun attlist10/11, sa186).
		"<!DOCTYPE root [\n<!ELEMENT root ANY>\n<!ATTLIST root att1 CDATA #IMPLIED>\n<!ATTLIST root att2 CDATA #IMPLIED>\n]>\n<root att1=\"value1\"att2=\"value2\"></root>",
		"<!DOCTYPE root [\n<!ELEMENT root ANY>\n<!ATTLIST root att1 CDATA #IMPLIED>\n<!ATTLIST root att2 CDATA #IMPLIED>\n]>\n<root att1=\"value1\"att2=\"value2\"/>",
		"<!DOCTYPE a [\n<!ELEMENT a EMPTY>\n<!ATTLIST a b CDATA #IMPLIED d CDATA #IMPLIED>\n]>\n<a b=\"c\"d=\"e\"/>",
		// A missing space before a namespace-declaration attribute is also
		// caught (the namespace branch already enforced this; keep it covered).
		`<doc att="val"xmlns:p="urn:x"/>`,
		// Namespace-declaration attribute FIRST, then a regular attribute with no
		// separating space: the new regular-attribute check must fire after the
		// namespace attribute is consumed (proves the two checks compose).
		`<doc xmlns:p="urn:x"att="val"/>`,
	}
	for _, input := range reject {
		_, err := helium.NewParser().Parse(t.Context(), []byte(input))
		require.Error(t, err, "Parse should reject missing space between attributes in %q", input)
		require.ErrorIs(t, err, helium.ErrSpaceRequired, "should be a space-required error for %q", input)
	}

	// The well-formed counterparts — attributes separated by a space, a
	// newline, or any XML whitespace — must still parse cleanly (no
	// over-rejection). A single attribute and the two tag-close forms are
	// included to exercise the '>' / '/>' branches of the new check.
	accept := []string{
		`<doc att="val" att2="val2"/>`,
		`<doc att="val" att2="val2"></doc>`,
		"<doc att=\"val\"\natt2=\"val2\"/>",
		"<doc att=\"val\"\t att2=\"val2\"/>",
		"<doc att=\"val\"\ratt2=\"val2\"/>",
		`<doc att="val"/>`,
		`<doc att="val"></doc>`,
		`<doc/>`,
		`<doc att="val" ></doc>`,
		// A space between a namespace-declaration attribute and a following
		// regular attribute must still parse (well-formed composition).
		`<doc xmlns:p="urn:x" att="val"/>`,
	}
	for _, input := range accept {
		_, err := helium.NewParser().Parse(t.Context(), []byte(input))
		require.NoError(t, err, "Parse should accept space-separated attributes in %q", input)
	}
}

func TestParseXML11PrefixUndeclaration(t *testing.T) {
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

func TestParseNamespace(t *testing.T) {
	const input = `<?xml version="1.0"?>
<helium:root xmlns:helium="https://github.com/lestrrat-go/helium">
  <helium:child>foo</helium:child>
</helium:root>`
	p := helium.NewParser()
	doc, err := p.Parse(t.Context(), []byte(input))
	require.NoError(t, err, "Parse should succeed for '%s'", input)

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

func TestParseRejectsMalformedDTDDefaultNamespace(t *testing.T) {
	// A namespace declaration supplied as a DTD <!ATTLIST> default must be
	// subject to the same Namespaces in XML validity checks an inline xmlns
	// declaration gets. These all bind a prefix illegally and must be
	// rejected even though the binding is never written on the element.
	reject := []string{
		// reuse of the reserved xmlns namespace name
		`<?xml version="1.0"?>
<!DOCTYPE root [
<!ELEMENT root EMPTY>
<!ATTLIST root xmlns:foo CDATA #FIXED "http://www.w3.org/2000/xmlns/">
]>
<root/>`,
		// reserved xml prefix bound to the wrong URI
		`<?xml version="1.0"?>
<!DOCTYPE root [
<!ELEMENT root EMPTY>
<!ATTLIST root xmlns:xml CDATA #FIXED "urn:wrong">
]>
<root/>`,
	}
	p := helium.NewParser()
	for _, input := range reject {
		_, err := p.Parse(t.Context(), []byte(input))
		require.Error(t, err, "Parse should reject malformed DTD-defaulted namespace: %s", input)
	}

	// A well-formed DTD-defaulted xmlns:xml mapped to its canonical URI is
	// accepted (and not pushed as a real binding).
	const ok = `<?xml version="1.0"?>
<!DOCTYPE root [
<!ELEMENT root EMPTY>
<!ATTLIST root xmlns:xml CDATA #FIXED "http://www.w3.org/XML/1998/namespace">
]>
<root/>`
	_, err := p.Parse(t.Context(), []byte(ok))
	require.NoError(t, err, "canonical xml prefix binding must be accepted")
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

// TestStripBlanksEntityEquivalence verifies that, under StripBlanks(true),
// whitespace adjacent to a decoded entity reference (e.g. &gt;) is treated
// exactly like whitespace adjacent to a literal character. Per XML 1.0 §4.4 an
// entity reference and the character it expands to are equivalent, so both
// forms must yield identical text content. The whitespace here abuts character
// data (it is not ignorable inter-element whitespace) and must be preserved.
func TestStripBlanksEntityEquivalence(t *testing.T) {
	testcases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "literal trailing space", input: `<r>x </r>`, want: "x "},
		{name: "entity trailing space", input: `<r>&gt; </r>`, want: "> "},
		{name: "entity leading space", input: `<r> &gt;</r>`, want: " >"},
		{name: "entity surrounded by spaces", input: `<r> &gt; </r>`, want: " > "},
		{name: "entity then literal", input: `<r>&gt;x</r>`, want: ">x"},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			p := helium.NewParser().StripBlanks(true)
			doc, err := p.Parse(t.Context(), []byte(tc.input))
			require.NoError(t, err, "Parse should succeed")

			root := findDocumentElement(doc)
			require.NotNil(t, root, "document element must exist")

			var got []byte
			for child := root.FirstChild(); child != nil; child = child.NextSibling() {
				if child.Type() == helium.TextNode {
					got = append(got, child.Content()...)
				}
			}
			require.Equal(t, tc.want, string(got), "text content must match; entity and literal whitespace are equivalent")
		})
	}
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

	p := helium.NewParser().BlockXXE(false).SubstituteEntities(true).FS(fsys)
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

	p := helium.NewParser().BlockXXE(false).SubstituteEntities(true).FS(fsys)
	_, err := p.Parse(t.Context(), []byte(input))
	require.Error(t, err, "malformed UTF-16 external entity must fail rather than inserting U+FFFD")
}

func TestParseUSASCIIStrict(t *testing.T) {
	t.Parallel()

	// A document declaring US-ASCII is strictly 7-bit; a byte >= 0x80 is
	// malformed even when it forms a valid UTF-8 multibyte sequence.
	highByte := "<?xml version=\"1.0\" encoding=\"US-ASCII\"?><root>\xc3\xa9</root>"
	_, err := helium.NewParser().Parse(t.Context(), []byte(highByte))
	require.Error(t, err, "US-ASCII document with a byte >= 0x80 must be rejected")

	// Valid 7-bit US-ASCII still parses.
	valid := `<?xml version="1.0" encoding="US-ASCII"?><root>hello</root>`
	_, err = helium.NewParser().Parse(t.Context(), []byte(valid))
	require.NoError(t, err, "valid 7-bit US-ASCII must parse")
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

	p := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).DefaultDTDAttributes(true).FS(fsys)
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

	p := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).DefaultDTDAttributes(true).FS(fsys)
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

	p := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).DefaultDTDAttributes(true).
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

	p := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).DefaultDTDAttributes(true).FS(fsys)
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

		p := helium.NewParser().BlockXXE(false).
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

		p := helium.NewParser().BlockXXE(false).
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

		p := helium.NewParser().BlockXXE(false).
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

	p := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).DefaultDTDAttributes(true).FS(fsys)
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

	p := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).DefaultDTDAttributes(true).FS(fsys)

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

	p := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).DefaultDTDAttributes(true).FS(fsys)
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

	p := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).DefaultDTDAttributes(true).FS(fsys)
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

	p := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).DefaultDTDAttributes(true).FS(fsys)
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

	p := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).DefaultDTDAttributes(true).FS(fsys)
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

	p := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).DefaultDTDAttributes(true).FS(fsys)
	_, err := p.Parse(t.Context(), []byte(input))
	require.Error(t, err, "a malformed external DTD declaration must produce a parse error")

	var pe helium.ErrParseError
	require.ErrorAs(t, err, &pe, "error must be a structured parse error")
	require.Equal(t, "bogus.dtd", pe.File, "error must reference the external DTD, not the main document")
}

func TestParseExternalDTDMalformedDeclInIncludeSurfaces(t *testing.T) {
	t.Parallel()

	const input = `<?xml version="1.0"?>
<!DOCTYPE r SYSTEM "inc.dtd">
<r/>`

	// A malformed declaration ("<!BOGUS") inside a WELL-FORMED, properly
	// terminated top-level <![INCLUDE[ ... ]]> section must surface as a parse
	// error. Previously the top-level external-subset loop swallowed EVERY error
	// from parseConditionalSections, silently accepting the bogus declaration.
	// Now conditional-section errors propagate: a missing/malformed keyword and
	// an unterminated "]]>" section are both fatal, and an actual declaration
	// parse error inside the INCLUDE body propagates.
	const dtd = `<![INCLUDE[ <!BOGUS ]]>`
	fsys := fstest.MapFS{"inc.dtd": &fstest.MapFile{Data: []byte(dtd)}}

	p := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).DefaultDTDAttributes(true).FS(fsys)
	_, err := p.Parse(t.Context(), []byte(input))
	require.Error(t, err, "a malformed declaration inside a top-level INCLUDE section must surface as a parse error")
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
	// instead of hanging the whole suite. The requirement here is PROMPT
	// completion (whether or not the parse surfaces a conditional-section error),
	// not a hang.
	ctx := t.Context()
	done := make(chan struct{})
	go func() {
		defer close(done)
		p := helium.NewParser().LoadExternalDTD(true).DefaultDTDAttributes(true).FS(fsys)
		_, _ = p.Parse(ctx, []byte(input))
	}()

	select {
	case <-done:
		// Completed promptly: the unterminated INCLUDE section did not loop.
	case <-time.After(5 * time.Second):
		t.Fatal("parsing an unterminated INCLUDE section hung (infinite loop regression)")
	}
}

// dtdSystemID is the external-DTD SYSTEM identifier (and MapFS filename) shared
// by the external-subset parser tests in this package.
const dtdSystemID = "d.dtd"

// peSystemID is the external parameter-entity SYSTEM identifier (and MapFS
// filename) shared by the external-PE parser tests in this package.
const peSystemID = "pe.ent"

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
	fsys := fstest.MapFS{dtdSystemID: &fstest.MapFile{Data: []byte(dtd)}}

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
		p := helium.NewParser().BlockXXE(false).SubstituteEntities(true).FS(rfs)
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
		p := helium.NewParser().BlockXXE(false).SubstituteEntities(true).FS(rfs)
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

	t.Run("enforced within substituted entity", func(t *testing.T) {
		t.Parallel()

		// The replacement text expands to two nested elements. With entity
		// substitution enabled the depth check must still apply to the chunk
		// parsed for the entity, so MaxDepth(1) rejects the inner <b/>.
		input := []byte(`<!DOCTYPE r [<!ENTITY e "<a><b/></a>">]><r>&e;</r>`)
		p := helium.NewParser().SubstituteEntities(true).MaxDepth(1)

		_, err := p.Parse(t.Context(), input)
		require.Error(t, err)
		require.Contains(t, err.Error(), "exceeded max depth")
	})

	t.Run("within limit inside substituted entity", func(t *testing.T) {
		t.Parallel()

		input := []byte(`<!DOCTYPE r [<!ENTITY e "<a><b/></a>">]><r>&e;</r>`)
		p := helium.NewParser().SubstituteEntities(true).MaxDepth(10)

		doc, err := p.Parse(t.Context(), input)
		require.NoError(t, err)
		require.NotNil(t, doc)
	})

	t.Run("single level via substituted entity counts parent depth", func(t *testing.T) {
		t.Parallel()

		// The entity replacement text is a single element, but it is substituted
		// inside <r>, so the literal document is <r><a/></r> (depth 2). The nested
		// chunk parse must continue counting from the parent's current element
		// depth (1) instead of restarting at 0, so MaxDepth(1) must reject it.
		input := []byte(`<!DOCTYPE r [<!ENTITY e "<a/>">]><r>&e;</r>`)
		p := helium.NewParser().SubstituteEntities(true).MaxDepth(1)

		_, err := p.Parse(t.Context(), input)
		require.Error(t, err)
		require.Contains(t, err.Error(), "exceeded max depth")
	})

	t.Run("enforced within external substituted entity", func(t *testing.T) {
		t.Parallel()

		// The external entity replacement text adds element nesting. With the
		// parent's current element depth carried into the external chunk parse,
		// MaxDepth(1) must reject the element delivered by the external entity.
		fsys := fstest.MapFS{
			"nested.xml": &fstest.MapFile{Data: []byte(`<a/>`)},
		}
		input := []byte(`<!DOCTYPE r [<!ENTITY e SYSTEM "nested.xml">]><r>&e;</r>`)
		p := helium.NewParser().BlockXXE(false).SubstituteEntities(true).MaxDepth(1).FS(fsys)

		_, err := p.Parse(t.Context(), input)
		require.Error(t, err)
		require.Contains(t, err.Error(), "exceeded max depth")
	})

	t.Run("cached entity replay under deeper element exceeds limit", func(t *testing.T) {
		t.Parallel()

		// The first &e; expands as a direct child of <r> (depth 2) and caches
		// its subtree. The second &e; is referenced inside <x> (depth 2), so its
		// element reaches depth 3. The cached subtree must still be charged
		// against MaxDepth on replay, otherwise the deeper reuse is wrongly
		// accepted.
		input := []byte(`<!DOCTYPE r [<!ENTITY e "<a/>">]><r>&e;<x>&e;</x></r>`)
		p := helium.NewParser().SubstituteEntities(true).MaxDepth(2)

		_, err := p.Parse(t.Context(), input)
		require.Error(t, err)
		require.Contains(t, err.Error(), "exceeded max depth")
	})

	t.Run("cached entity replay within limit succeeds", func(t *testing.T) {
		t.Parallel()

		// Same shape as above, but MaxDepth(3) accommodates the deeper reuse, so
		// both expansions parse cleanly.
		input := []byte(`<!DOCTYPE r [<!ENTITY e "<a/>">]><r>&e;<x>&e;</x></r>`)
		p := helium.NewParser().SubstituteEntities(true).MaxDepth(3)

		doc, err := p.Parse(t.Context(), input)
		require.NoError(t, err)
		require.NotNil(t, doc)
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

// zeroProgressReader always returns (0, nil) for a non-empty request, never
// advancing and never erroring. A naive fill loop spins on it forever.
type zeroProgressReader struct{}

func (zeroProgressReader) Read(p []byte) (int, error) {
	return 0, nil
}

func TestParseReaderZeroProgressReaderDoesNotHang(t *testing.T) {
	p := helium.NewParser()

	done := make(chan error, 1)
	go func() {
		_, err := p.ParseReader(t.Context(), zeroProgressReader{})
		done <- err
	}()

	select {
	case err := <-done:
		require.ErrorIs(t, err, io.ErrNoProgress, "a zero-progress reader must fail with io.ErrNoProgress, not be accepted")
	case <-time.After(5 * time.Second):
		t.Fatal("ParseReader hung on a zero-progress reader instead of failing fast")
	}
}

// startElementRecorder builds a SAX handler that records start-element local
// names, so a test can prove the buffered bytes were parsed before any read
// error was surfaced.
func startElementRecorder(seen *[]string) sax.SAX2Handler {
	h := sax.New()
	h.SetOnStartElementNS(sax.StartElementNSFunc(func(_ context.Context, local, _, _ string, _ []sax.Namespace, _ []sax.Attribute) error {
		*seen = append(*seen, local)
		return nil
	}))
	return h
}

// TestParseReaderParsesBytesThenSurfacesError is the convergence regression: a
// reader that returns (n>0, non-EOF err) in a single Read must have its bytes
// PARSED first and the error surfaced only AFTER they drain — on BOTH the
// non-EBCDIC streaming path and the EBCDIC byte-slice path. Bytes returned
// alongside a non-EOF error are never discarded.
func TestParseReaderParsesBytesThenSurfacesError(t *testing.T) {
	wantErr := errors.New("checksum mismatch")

	ebcdic := func(s string) []byte {
		b, err := charmap.CodePage037.NewEncoder().Bytes([]byte(s))
		require.NoError(t, err)
		require.Equal(t, []byte{0x4C, 0x6F, 0xA7, 0x94}, b[:4],
			"encoded bytes must start with the EBCDIC invariant prefix")
		return b
	}

	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "non-EBCDIC",
			data: []byte(`<root><child/></root>`),
		},
		{
			name: "EBCDIC",
			data: ebcdic(`<?xml version="1.0" encoding="IBM037"?><root><child/></root>`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var seen []string
			p := helium.NewParser().SAXHandler(startElementRecorder(&seen))

			_, err := p.ParseReader(t.Context(), &dataThenErrReader{
				data: tt.data,
				err:  wantErr,
			})
			require.ErrorIs(t, err, wantErr,
				"a non-EOF read error returned alongside the bytes must be surfaced")
			require.Equal(t, []string{"root", "child"}, seen,
				"the bytes returned alongside the error must be parsed before the error surfaces")
		})
	}
}

// blockingReader blocks forever inside Read until its done channel is closed,
// then returns io.EOF. It models a non-context-aware reader (e.g. a slow
// network stream) whose Read cannot be interrupted generically.
type blockingReader struct {
	done    chan struct{}
	entered chan struct{} // closed the first time Read is entered
	once    sync.Once
}

func (r *blockingReader) Read([]byte) (int, error) {
	r.once.Do(func() { close(r.entered) })
	<-r.done
	return 0, io.EOF
}

// TestParseReaderCancelledUpFrontDoesNotBlock guards the "cancelled before any
// blocking read" contract: when the context is already cancelled, ParseReader
// must return the context error promptly WITHOUT ever entering the underlying
// reader's Read (the EBCDIC sniff must check ctx first).
func TestParseReaderCancelledUpFrontDoesNotBlock(t *testing.T) {
	t.Parallel()

	r := &blockingReader{done: make(chan struct{}), entered: make(chan struct{})}
	// Never unblock the reader: if ParseReader touches it, the test deadlocks
	// and is caught by the timeout below.
	defer close(r.done)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled before the call

	type result struct {
		doc *helium.Document
		err error
	}
	resCh := make(chan result, 1)
	go func() {
		doc, err := helium.NewParser().ParseReader(ctx, r)
		resCh <- result{doc, err}
	}()

	select {
	case res := <-resCh:
		require.ErrorIs(t, res.err, context.Canceled,
			"a context cancelled before any read must surface as context.Canceled")
		require.Nil(t, res.doc, "a cancelled parse must not return a document")
	case <-time.After(2 * time.Second):
		t.Fatal("ParseReader blocked on a non-context-aware reader despite an already-cancelled context")
	}

	select {
	case <-r.entered:
		t.Fatal("ParseReader read from the underlying reader despite an already-cancelled context")
	default:
	}
}

func TestParseFileParsesNormalFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "doc.xml")
	require.NoError(t, os.WriteFile(path, []byte(`<root><child>hi</child></root>`), 0o600))

	doc, err := helium.NewParser().ParseFile(t.Context(), path)
	require.NoError(t, err)
	require.NotNil(t, doc)

	abs, err := filepath.Abs(path)
	require.NoError(t, err)
	require.Equal(t, abs, doc.URL(), "document URL should be the absolute path")
}

func TestParseFileMissingFileErrors(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "does-not-exist.xml")

	_, err := helium.NewParser().ParseFile(t.Context(), path)
	require.Error(t, err, "parsing a missing file must error")
}

// TestParseFileEBCDICMatchesParse guards EBCDIC encoding parity across entry
// points: an EBCDIC-encoded document must parse identically whether read via
// ParseFile, ParseReader, or Parse([]byte). EBCDIC detection/decode relies on
// the original raw bytes, so the reader-based paths must buffer the input.
func TestParseFileEBCDICMatchesParse(t *testing.T) {
	t.Parallel()

	const xml = `<?xml version="1.0" encoding="IBM037"?><root><child>hi</child></root>`
	ebcdic, err := charmap.CodePage037.NewEncoder().Bytes([]byte(xml))
	require.NoError(t, err)
	// Sanity: the encoded bytes must carry the EBCDIC invariant prefix that
	// triggers detection (otherwise the test would not exercise the path).
	require.Equal(t, []byte{0x4C, 0x6F, 0xA7, 0x94}, ebcdic[:4],
		"encoded bytes must start with the EBCDIC invariant prefix")

	serialize := func(doc *helium.Document) string {
		var buf bytes.Buffer
		require.NoError(t, helium.NewWriter().WriteTo(&buf, doc))
		return buf.String()
	}

	bytesDoc, err := helium.NewParser().Parse(t.Context(), ebcdic)
	require.NoError(t, err, "Parse([]byte) must handle EBCDIC")
	want := serialize(bytesDoc)

	readerDoc, err := helium.NewParser().ParseReader(t.Context(), bytes.NewReader(ebcdic))
	require.NoError(t, err, "ParseReader must handle EBCDIC")
	require.Equal(t, want, serialize(readerDoc),
		"ParseReader output must match Parse([]byte) for EBCDIC")

	dir := t.TempDir()
	path := filepath.Join(dir, "doc.xml")
	require.NoError(t, os.WriteFile(path, ebcdic, 0o600))
	fileDoc, err := helium.NewParser().ParseFile(t.Context(), path)
	require.NoError(t, err, "ParseFile must handle EBCDIC")
	require.Equal(t, want, serialize(fileDoc),
		"ParseFile output must match Parse([]byte) for EBCDIC")
}

// TestParseReaderEBCDICDataWithEOFInFirstRead guards the EBCDIC sniff against a
// reader that returns its entire payload together with io.EOF in a single Read
// (which io.Reader explicitly permits). EBCDIC decoding requires the full raw
// input up front, so detection must happen even when the head read ends with
// io.EOF; otherwise the streaming path resets the cursor from a nil rawInput and
// loses the document.
func TestParseReaderEBCDICDataWithEOFInFirstRead(t *testing.T) {
	t.Parallel()

	const xml = `<?xml version="1.0" encoding="IBM037"?><root><child>hi</child></root>`
	ebcdic, err := charmap.CodePage037.NewEncoder().Bytes([]byte(xml))
	require.NoError(t, err)
	require.Equal(t, []byte{0x4C, 0x6F, 0xA7, 0x94}, ebcdic[:4],
		"encoded bytes must start with the EBCDIC invariant prefix")

	serialize := func(doc *helium.Document) string {
		var buf bytes.Buffer
		require.NoError(t, helium.NewWriter().WriteTo(&buf, doc))
		return buf.String()
	}

	bytesDoc, err := helium.NewParser().Parse(t.Context(), ebcdic)
	require.NoError(t, err, "Parse([]byte) must handle EBCDIC")
	want := serialize(bytesDoc)

	// dataThenErrReader with err == io.EOF returns all bytes plus io.EOF in the
	// FIRST Read, exactly the case that previously fell through to the streaming
	// path and produced a parse error.
	r := &dataThenErrReader{data: ebcdic, err: io.EOF}
	doc, err := helium.NewParser().ParseReader(t.Context(), r)
	require.NoError(t, err, "ParseReader must parse EBCDIC delivered with io.EOF in the first read")
	require.Equal(t, want, serialize(doc),
		"ParseReader output must match Parse([]byte) when EBCDIC arrives with io.EOF in one read")
}

// stoppableEBCDICReader serves a finite EBCDIC head (whose invariant prefix
// triggers EBCDIC detection) on the first Reads, then an endless run of EBCDIC
// space bytes — an unterminated whitespace character-data run inside <root> —
// that never reaches EOF, until Stop is called (after which Read returns
// io.EOF). It models a hostile, never-ending stream. The parser must terminate
// it via its incremental per-node content cap WITHOUT buffering the tail; Stop
// plus a test timeout guarantee that even a regression cannot hang or OOM the
// test process.
type stoppableEBCDICReader struct {
	head    []byte
	pos     int
	stopped atomic.Bool
}

func (r *stoppableEBCDICReader) Stop() { r.stopped.Store(true) }

func (r *stoppableEBCDICReader) Read(p []byte) (int, error) {
	if r.stopped.Load() {
		return 0, io.EOF
	}
	if r.pos < len(r.head) {
		n := copy(p, r.head[r.pos:])
		r.pos += n
		return n, nil
	}
	for i := range p {
		p[i] = 0x40 // EBCDIC space: an unterminated whitespace run, never EOF
	}
	return len(p), nil
}

// TestParseReaderEBCDICUnboundedStreamBoundedByNodeCap guards the streaming
// EBCDIC reader path against unbounded memory growth: EBCDIC now streams through
// the normal cursor pipeline, so a hostile never-ending stream is bounded by the
// parser's incremental per-node content cap (the single whitespace run inside
// <root> exceeds MaxNodeContentSize) and fails with ErrNodeContentTooLarge —
// never buffered whole into memory. The reader runs in a goroutine with a
// timeout and a Stop so a regression that reintroduced whole-stream buffering
// cannot hang or OOM the test.
func TestParseReaderEBCDICUnboundedStreamBoundedByNodeCap(t *testing.T) {
	t.Parallel()

	const decl = `<?xml version="1.0" encoding="IBM037"?><root>`
	head, err := charmap.CodePage037.NewEncoder().Bytes([]byte(decl))
	require.NoError(t, err)
	require.Equal(t, []byte{0x4C, 0x6F, 0xA7, 0x94}, head[:4],
		"encoded head must start with the EBCDIC invariant prefix")

	r := &stoppableEBCDICReader{head: head}
	defer r.Stop()

	errCh := make(chan error, 1)
	go func() {
		// A small cap keeps the test fast: the per-node cap must fire well before
		// the infinite tail could exhaust memory.
		_, perr := helium.NewParser().MaxNodeContentSize(4096).ParseReader(t.Context(), r)
		errCh <- perr
	}()

	select {
	case perr := <-errCh:
		require.ErrorIs(t, perr, helium.ErrNodeContentTooLarge,
			"an unbounded EBCDIC stream must be bounded by the per-node content cap")
	case <-time.After(5 * time.Second):
		r.Stop() // unblock the parser so the leaked goroutine can exit
		t.Fatal("ParseReader did not terminate an unbounded EBCDIC stream within the timeout")
	}
}

// TestParseReaderEBCDICLargeFiniteDocUnderNodeCap guards the key property of the
// streaming EBCDIC path: a finite document whose TOTAL size exceeds
// MaxNodeContentSize but whose every individual node is well under the cap must
// parse successfully and identically to Parse([]byte). A total-input cap (the
// earlier, wrong approach) would have wrongly rejected this document and
// diverged from Parse([]byte); the per-node cap accepts it.
func TestParseReaderEBCDICLargeFiniteDocUnderNodeCap(t *testing.T) {
	t.Parallel()

	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="IBM037"?><root>`)
	for range 500 {
		sb.WriteString(`<c>x</c>`)
	}
	sb.WriteString(`</root>`)
	xml := sb.String()

	ebcdic, err := charmap.CodePage037.NewEncoder().Bytes([]byte(xml))
	require.NoError(t, err)
	require.Greater(t, len(ebcdic), 1024,
		"the document must be larger than the per-node cap used below")
	require.Equal(t, []byte{0x4C, 0x6F, 0xA7, 0x94}, ebcdic[:4],
		"encoded bytes must start with the EBCDIC invariant prefix")

	serialize := func(doc *helium.Document) string {
		var buf bytes.Buffer
		require.NoError(t, helium.NewWriter().WriteTo(&buf, doc))
		return buf.String()
	}

	// A cap smaller than the total document but larger than any single node.
	bytesDoc, err := helium.NewParser().MaxNodeContentSize(1024).Parse(t.Context(), ebcdic)
	require.NoError(t, err, "Parse([]byte) must accept a large finite EBCDIC doc with small nodes")
	want := serialize(bytesDoc)

	readerDoc, err := helium.NewParser().MaxNodeContentSize(1024).ParseReader(t.Context(), bytes.NewReader(ebcdic))
	require.NoError(t, err,
		"ParseReader must accept a large finite EBCDIC doc whose nodes are under the per-node cap")
	require.Equal(t, want, serialize(readerDoc),
		"ParseReader output must match Parse([]byte) for a large finite EBCDIC doc")
}

// TestParseReaderEBCDICLargeEntityNotFalselyAmplified guards against a
// regression where the streaming EBCDIC path left inputSize seeded from the
// bounded sniff prefix (~256/512 bytes) rather than the real document size. A
// large internal entity referenced exactly once is legitimate (no
// amplification), and Parse([]byte) accepts it because inputSize is the full
// slice length. With only the prefix length as the divisor, the
// amplification-ratio guard would falsely reject the same document over
// ParseReader. Tracking the bytes consumed from the stream fixes it.
func TestParseReaderEBCDICLargeEntityNotFalselyAmplified(t *testing.T) {
	t.Parallel()

	// Entity content just over the 1 MiB ratio-check baseline, referenced once.
	bigContent := strings.Repeat("A", 1_500_000)
	xml := fmt.Sprintf(`<?xml version="1.0" encoding="IBM037"?><!DOCTYPE root [<!ENTITY big "%s">]><root>&big;</root>`, bigContent)

	ebcdic, err := charmap.CodePage037.NewEncoder().Bytes([]byte(xml))
	require.NoError(t, err)
	require.Equal(t, []byte{0x4C, 0x6F, 0xA7, 0x94}, ebcdic[:4],
		"encoded bytes must start with the EBCDIC invariant prefix")

	serialize := func(doc *helium.Document) string {
		var buf bytes.Buffer
		require.NoError(t, helium.NewWriter().WriteTo(&buf, doc))
		return buf.String()
	}

	// Baseline: Parse([]byte) accepts it (inputSize == len(ebcdic)). Use a large
	// per-node cap so the 1.5 MiB text node is not what trips — this test is about
	// the amplification guard, not the node-content cap.
	bytesDoc, err := helium.NewParser().SubstituteEntities(true).MaxNodeContentSize(-1).Parse(t.Context(), ebcdic)
	require.NoError(t, err, "Parse([]byte) must accept a large EBCDIC entity referenced once")
	want := serialize(bytesDoc)

	// ParseReader (unknown source size) must match: the amplification guard must
	// use the real consumed-byte count, not the sniff-prefix length.
	readerDoc, err := helium.NewParser().SubstituteEntities(true).MaxNodeContentSize(-1).ParseReader(t.Context(), bytes.NewReader(ebcdic))
	require.NoError(t, err,
		"ParseReader must accept the same large-entity EBCDIC document as Parse([]byte)")
	require.Equal(t, want, serialize(readerDoc),
		"ParseReader output must match Parse([]byte) for a large-entity EBCDIC doc")
}

// TestParseReaderEBCDICNestedLargeEntityNotFalselyAmplified guards the nested
// entity sub-parse path. When a large entity is reached INDIRECTLY through
// another entity's replacement text (&wrap; -> &big;), the inner &big; expansion
// runs in a nested parser context. That nested context copied inputSize from the
// parent (the bounded EBCDIC sniff prefix) but, before this fix, did NOT carry
// the live ebcdicConsumed byte-counter, so the amplification-ratio guard divided
// by the prefix length and falsely rejected a document Parse([]byte) accepts.
// Propagating ebcdicConsumed through inheritNestedParserState fixes it.
func TestParseReaderEBCDICNestedLargeEntityNotFalselyAmplified(t *testing.T) {
	t.Parallel()

	// Entity content just over the 1 MiB ratio-check baseline, referenced once
	// through a wrapping entity so the expansion happens inside a nested context.
	bigContent := strings.Repeat("A", 1_500_000)
	xml := fmt.Sprintf(`<?xml version="1.0" encoding="IBM037"?><!DOCTYPE root [<!ENTITY big "%s"><!ENTITY wrap "&big;">]><root>&wrap;</root>`, bigContent)

	ebcdic, err := charmap.CodePage037.NewEncoder().Bytes([]byte(xml))
	require.NoError(t, err)
	require.Equal(t, []byte{0x4C, 0x6F, 0xA7, 0x94}, ebcdic[:4],
		"encoded bytes must start with the EBCDIC invariant prefix")

	serialize := func(doc *helium.Document) string {
		var buf bytes.Buffer
		require.NoError(t, helium.NewWriter().WriteTo(&buf, doc))
		return buf.String()
	}

	// Baseline: Parse([]byte) accepts it (inputSize == len(ebcdic)). Disable the
	// per-node cap so the 1.5 MiB text node is not what trips — this test is about
	// the amplification guard inside the nested entity context.
	bytesDoc, err := helium.NewParser().SubstituteEntities(true).MaxNodeContentSize(-1).Parse(t.Context(), ebcdic)
	require.NoError(t, err, "Parse([]byte) must accept a nested large EBCDIC entity referenced once")
	want := serialize(bytesDoc)

	// ParseReader (unknown source size) must match: the nested context's
	// amplification guard must use the real consumed-byte count, not the
	// sniff-prefix length.
	readerDoc, err := helium.NewParser().SubstituteEntities(true).MaxNodeContentSize(-1).ParseReader(t.Context(), bytes.NewReader(ebcdic))
	require.NoError(t, err,
		"ParseReader must accept the same nested large-entity EBCDIC document as Parse([]byte)")
	require.Equal(t, want, serialize(readerDoc),
		"ParseReader output must match Parse([]byte) for a nested large-entity EBCDIC doc")
}

// TestParseReaderEBCDICAmplificationAttackStillRejected confirms the
// inputSize-vs-consumed fix did NOT reopen the entity-amplification DoS over the
// streaming EBCDIC reader path. The fix divides sizeentcopy by the real
// consumed-byte count so a large entity referenced ONCE passes; an actual attack
// (a modestly sized entity referenced MANY times) expands far beyond the
// amplification factor of the real document size and must STILL be rejected,
// exactly as Parse([]byte) rejects it. The per-node cap is disabled so the
// failure is attributable to the amplification guard, not the node-content cap.
func TestParseReaderEBCDICAmplificationAttackStillRejected(t *testing.T) {
	t.Parallel()

	// ~250 KB entity referenced 30 times: the document is ~250 KB on disk but
	// expands to ~7.5 MB, well past 5x the real consumed size, so the
	// amplification-ratio guard must fire (and not the 1 GB hard ceiling).
	body := strings.Repeat("A", 250_000)
	refs := strings.Repeat("&a;", 30)
	xml := fmt.Sprintf(`<?xml version="1.0" encoding="IBM037"?><!DOCTYPE root [<!ENTITY a "%s">]><root>%s</root>`, body, refs)

	ebcdic, err := charmap.CodePage037.NewEncoder().Bytes([]byte(xml))
	require.NoError(t, err)
	require.Equal(t, []byte{0x4C, 0x6F, 0xA7, 0x94}, ebcdic[:4],
		"encoded bytes must start with the EBCDIC invariant prefix")

	// Parse([]byte) baseline: the attack must be rejected.
	_, err = helium.NewParser().SubstituteEntities(true).MaxNodeContentSize(-1).Parse(t.Context(), ebcdic)
	require.Error(t, err, "Parse([]byte) must reject an EBCDIC entity-amplification attack")
	require.Contains(t, err.Error(), "amplification",
		"Parse([]byte) must reject via the amplification guard")

	// ParseReader (unknown source size) must STILL reject it: the consumed-byte
	// divisor reflects the real (small) document, so the ratio guard fires.
	_, err = helium.NewParser().SubstituteEntities(true).MaxNodeContentSize(-1).ParseReader(t.Context(), bytes.NewReader(ebcdic))
	require.Error(t, err, "ParseReader must reject an EBCDIC entity-amplification attack")
	require.Contains(t, err.Error(), "amplification",
		"ParseReader must reject via the amplification guard, not silently accept the DoS")
}

// TestParseReaderEBCDICSmallDocUnderCap confirms the streaming EBCDIC reader
// path parses a normal, small EBCDIC document identically to Parse([]byte).
func TestParseReaderEBCDICSmallDocUnderCap(t *testing.T) {
	t.Parallel()

	const xml = `<?xml version="1.0" encoding="IBM037"?><root><child>hi</child></root>`
	ebcdic, err := charmap.CodePage037.NewEncoder().Bytes([]byte(xml))
	require.NoError(t, err)
	require.Equal(t, []byte{0x4C, 0x6F, 0xA7, 0x94}, ebcdic[:4],
		"encoded bytes must start with the EBCDIC invariant prefix")

	serialize := func(doc *helium.Document) string {
		var buf bytes.Buffer
		require.NoError(t, helium.NewWriter().WriteTo(&buf, doc))
		return buf.String()
	}

	bytesDoc, err := helium.NewParser().Parse(t.Context(), ebcdic)
	require.NoError(t, err, "Parse([]byte) must handle EBCDIC")
	want := serialize(bytesDoc)

	readerDoc, err := helium.NewParser().ParseReader(t.Context(), bytes.NewReader(ebcdic))
	require.NoError(t, err, "a small EBCDIC doc must parse under the default ingestion cap")
	require.Equal(t, want, serialize(readerDoc),
		"ParseReader output must match Parse([]byte) for a small EBCDIC doc")
}

// infiniteBlankReader streams a fixed head followed by an unbounded run of
// ASCII spaces and never reaches EOF. It models a prolog / inter-root
// whitespace DoS: input that keeps the parser in skipBlanks forever. The blank
// run is OUTSIDE any element, so the char-data node-content cap never applies —
// only the blank-run guard in skipBlanks bounds it.
type infiniteBlankReader struct {
	head    []byte
	pos     int
	stopped atomic.Bool
}

func (r *infiniteBlankReader) Stop() { r.stopped.Store(true) }

func (r *infiniteBlankReader) Read(p []byte) (int, error) {
	if r.stopped.Load() {
		return 0, io.EOF
	}
	if r.pos < len(r.head) {
		n := copy(p, r.head[r.pos:])
		r.pos += n
		return n, nil
	}
	for i := range p {
		p[i] = ' '
	}
	return len(p), nil
}

// TestParseReaderUnboundedPrologWhitespaceBounded guards the prolog/inter-root
// whitespace DoS that the per-node content cap does NOT cover: an unbounded run
// of whitespace BEFORE the root element (or after it) is consumed by skipBlanks,
// which formerly peeked an ever-growing offset and grew the cursor buffer
// without bound. The blank-run guard now caps it and fails the parse instead of
// hanging or exhausting memory. A goroutine + timeout + Stop guarantees a
// regression cannot hang or OOM the test process.
func TestParseReaderUnboundedPrologWhitespaceBounded(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		// Infinite whitespace between the XML declaration and the root element.
		"prolog before root": `<?xml version="1.0"?>`,
		// Infinite whitespace in the epilogue, after a complete root element.
		"epilogue after root": `<?xml version="1.0"?><root/>`,
	}

	for name, head := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			r := &infiniteBlankReader{head: []byte(head)}
			defer r.Stop()

			errCh := make(chan error, 1)
			go func() {
				// A small cap keeps the test fast: the blank-run guard must fire
				// well before the infinite tail could exhaust memory.
				_, perr := helium.NewParser().MaxNodeContentSize(4096).ParseReader(t.Context(), r)
				errCh <- perr
			}()

			select {
			case perr := <-errCh:
				require.ErrorIs(t, perr, helium.ErrNodeContentTooLarge,
					"unbounded %s whitespace must be bounded by the blank-run guard", name)
			case <-time.After(5 * time.Second):
				r.Stop() // unblock the parser so the leaked goroutine can exit
				t.Fatalf("ParseReader did not terminate unbounded %s whitespace within the timeout", name)
			}
		})
	}
}

// TestParseLeadingAndTrailingWhitespacePreserved confirms the bounded blank
// skip still handles ordinary (within-cap) whitespace correctly: leading
// whitespace before the root, whitespace around the XML declaration, and
// trailing whitespace in the epilogue all parse without error.
func TestParseLeadingAndTrailingWhitespacePreserved(t *testing.T) {
	t.Parallel()

	docs := map[string]string{
		"leading before root":   "<?xml version=\"1.0\"?>\n\n  \t<root/>",
		"trailing epilogue":     "<root/>\n  \t\n",
		"between prolog nodes":  "<?xml version=\"1.0\"?>\n<!-- c -->\n  <root/>\n",
		"large within-cap blob": "<?xml version=\"1.0\"?>" + strings.Repeat(" ", 5000) + "<root/>",
	}

	for name, doc := range docs {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			// A cap larger than the within-cap blob; ordinary whitespace must
			// never trip the blank-run guard.
			d, err := helium.NewParser().MaxNodeContentSize(1<<20).Parse(t.Context(), []byte(doc))
			require.NoError(t, err)
			require.NotNil(t, d)
		})
	}
}

// prefixThenZeroThenRestReader delivers a leading prefix, then a single
// transient (0, nil) read (which io.Reader explicitly permits while a slow
// producer waits for more input), then the remaining bytes, then io.EOF. It
// models a stream that splits the EBCDIC sniff prefix across a zero-progress
// read.
type prefixThenZeroThenRestReader struct {
	data        []byte
	prefixLen   int
	pos         int
	emittedZero bool
}

func (r *prefixThenZeroThenRestReader) Read(p []byte) (int, error) {
	if r.pos < r.prefixLen {
		n := copy(p, r.data[r.pos:r.prefixLen])
		r.pos += n
		return n, nil
	}
	if !r.emittedZero {
		r.emittedZero = true
		return 0, nil // transient empty read mid-sniff
	}
	if r.pos < len(r.data) {
		n := copy(p, r.data[r.pos:])
		r.pos += n
		return n, nil
	}
	return 0, io.EOF
}

// TestParseReaderEBCDICNonIBM037SplitByZeroProgressRead guards the EBCDIC
// sniff-extension loop against a transient (0, nil) read that splits the sniff
// prefix before the encoding declaration has been buffered. A non-IBM037 EBCDIC
// variant (CP1141) is declared and its content uses byte 0x4A, which decodes to
// U+00C4 (Ä) under CP1141 but to U+00A2 (¢) under CP037. If a (0, nil) read
// truncated the sniff prefix, ExtractEBCDICEncoding would miss the declaration,
// the parser would default to IBM-037, and the text would wrongly decode to ¢.
// The bounded zero-progress retry must keep reading so CP1141 is detected and
// the document parses identically to Parse([]byte).
func TestParseReaderEBCDICNonIBM037SplitByZeroProgressRead(t *testing.T) {
	t.Parallel()

	const xml = `<?xml version="1.0" encoding="IBM1141"?><root>Ä</root>`
	cp1141 := heliumencoding.Load("ibm1141")
	require.NotNil(t, cp1141, "internal encoding registry must know CP1141")
	ebcdic, err := cp1141.NewEncoder().Bytes([]byte(xml))
	require.NoError(t, err)
	require.Equal(t, []byte{0x4C, 0x6F, 0xA7, 0x94}, ebcdic[:4],
		"encoded bytes must start with the EBCDIC invariant prefix")

	serialize := func(doc *helium.Document) string {
		var buf bytes.Buffer
		require.NoError(t, helium.NewWriter().WriteTo(&buf, doc))
		return buf.String()
	}

	bytesDoc, err := helium.NewParser().Parse(t.Context(), ebcdic)
	require.NoError(t, err, "Parse([]byte) must handle CP1141 EBCDIC")
	want := serialize(bytesDoc)
	// The writer re-encodes to the document's declared EBCDIC encoding, so assert
	// the decoded DOM text directly: CP1141 byte 0x4A is Ä (U+00C4); the IBM-037
	// default would have decoded it to ¢ (U+00A2).
	require.Equal(t, "Ä", string(bytesDoc.DocumentElement().Content()),
		"CP1141 content byte 0x4A must decode to Ä, not the IBM-037 default (¢)")

	// Split the stream right after the invariant prefix so the (0, nil) read lands
	// inside the sniff-extension loop, before the encoding declaration is buffered.
	r := &prefixThenZeroThenRestReader{data: ebcdic, prefixLen: len(ebcdic[:4])}
	doc, err := helium.NewParser().ParseReader(t.Context(), r)
	require.NoError(t, err,
		"ParseReader must parse CP1141 EBCDIC when a transient (0, nil) read splits the sniff prefix")
	require.Equal(t, want, serialize(doc),
		"ParseReader must detect CP1141 (not default to IBM-037) across a zero-progress read")
}

// ebcdicSlowTailReader returns the EBCDIC invariant prefix on its first Read (so
// detection succeeds), signals when its tail is first read, then drips the tail
// one byte at a time. The first tail byte is served immediately so the parser
// regains control (and re-checks ctx); every later tail Read blocks until the
// reader is cancelled, modeling a stalled tail whose remaining bytes never
// arrive. If ParseReader honored ctx between reads it returns before consuming
// the whole tail; if it drained eagerly it would block forever.
type ebcdicSlowTailReader struct {
	prefix    []byte
	gate      chan struct{} // closed to unblock later tail reads (test cleanup only)
	entered   chan struct{} // closed the first time a tail byte is served
	cancelled chan struct{} // closed by the test once ctx has been cancelled
	once      sync.Once
	served    int // number of tail bytes already delivered
}

func (r *ebcdicSlowTailReader) Read(p []byte) (int, error) {
	if r.served == 0 && len(r.prefix) > 0 {
		// First Read: hand back the EBCDIC prefix so detection succeeds.
		n := copy(p, r.prefix)
		r.prefix = nil
		return n, nil
	}
	if r.served == 0 {
		// First tail Read: signal the test, then wait until it has cancelled the
		// context before returning one byte. This makes the test deterministic:
		// when the drain loop iterates and re-checks ctx on the next pass, the
		// cancellation is guaranteed to be observable, so a ctx-honoring loop
		// returns instead of entering the blocking Read below.
		r.served++
		r.once.Do(func() { close(r.entered) })
		<-r.cancelled
		if len(p) > 0 {
			p[0] = ' '
			return 1, nil
		}
		return 0, nil
	}
	// Later tail Reads block until the test tears the reader down. A
	// ctx-honoring drain loop never reaches here after cancellation.
	<-r.gate
	return 0, io.EOF
}

// TestParseReaderEBCDICTailCancelledDoesNotDrain guards the ctx-cancellation
// contract on the EBCDIC tail-drain path: once the EBCDIC prefix is detected,
// the remainder of the stream must be read through a loop that re-checks ctx
// BEFORE each Read. When the context is cancelled after the prefix read and the
// tail stalls, ParseReader must return context.Canceled promptly WITHOUT
// blocking on the unread tail.
func TestParseReaderEBCDICTailCancelledDoesNotDrain(t *testing.T) {
	t.Parallel()

	r := &ebcdicSlowTailReader{
		prefix:    []byte{0x4C, 0x6F, 0xA7, 0x94},
		gate:      make(chan struct{}),
		entered:   make(chan struct{}),
		cancelled: make(chan struct{}),
	}
	// Never unblock the stalled tail: if ParseReader drains past the ctx check,
	// the test deadlocks and is caught by the timeout below.
	defer close(r.gate)

	ctx, cancel := context.WithCancel(context.Background())

	type result struct {
		doc *helium.Document
		err error
	}
	resCh := make(chan result, 1)
	go func() {
		doc, err := helium.NewParser().ParseReader(ctx, r)
		resCh <- result{doc, err}
	}()

	// Wait until the parser has entered the tail-drain loop (prefix consumed),
	// then cancel: this exercises a cancellation observed after the prefix read.
	select {
	case <-r.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("ParseReader did not begin draining the EBCDIC tail")
	}
	cancel()
	close(r.cancelled)

	select {
	case res := <-resCh:
		require.ErrorIs(t, res.err, context.Canceled,
			"a context cancelled while draining the EBCDIC tail must surface as context.Canceled")
		require.Nil(t, res.doc, "a cancelled parse must not return a document")
	case <-time.After(2 * time.Second):
		t.Fatal("ParseReader blocked on the EBCDIC tail despite a cancelled context")
	}
}

func TestParseFileResolvesRelativeExternalEntity(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "child.xml"), []byte("WORLD"), 0o600))

	main := `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ENTITY child SYSTEM "child.xml">
]>
<doc>&child;</doc>`
	mainPath := filepath.Join(dir, "main.xml")
	require.NoError(t, os.WriteFile(mainPath, []byte(main), 0o600))

	doc, err := helium.NewParser().BlockXXE(false).SubstituteEntities(true).FS(helium.PermissiveFS()).ParseFile(t.Context(), mainPath)
	require.NoError(t, err)
	require.NotNil(t, doc)

	var buf bytes.Buffer
	require.NoError(t, helium.NewWriter().WriteTo(&buf, doc))
	require.Contains(t, buf.String(), "WORLD",
		"relative external entity must resolve against the file's base URI")
}

// TestLenientXMLDecl exercises the LenientXMLDecl parse path, including
// pseudo-attributes presented out of the canonical order.
func TestLenientXMLDecl(t *testing.T) {
	t.Parallel()

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
}

// TestMalformedXMLDecl exercises XML-declaration error branches.
func TestMalformedXMLDecl(t *testing.T) {
	t.Parallel()

	bad := []string{
		`<?xml?><root/>`,                         // missing version
		`<?xml version="1.0" foo="bar"?><root/>`, // unknown pseudo-attr / unclosed
		`<?xml version=1.0?><root/>`,             // unquoted version value
	}
	for _, in := range bad {
		_, err := helium.NewParser().Parse(t.Context(), []byte(in))
		require.Error(t, err, "malformed decl %q should error", in)
	}
}

// TestProcessingInstructionsAndComments parses PIs and comments in the prolog,
// content, and epilog positions.
func TestProcessingInstructionsAndComments(t *testing.T) {
	t.Parallel()

	const src = `<?xml version="1.0"?>
<?pi-prolog data?>
<!-- prolog comment -->
<root>
  <?pi-content more?>
  <!-- content comment -->
  text
</root>
<!-- epilog comment -->
<?pi-epilog x?>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err)

	out, err := helium.WriteString(doc)
	require.NoError(t, err)
	require.Contains(t, out, "<?pi-prolog")
	require.Contains(t, out, "<!-- prolog comment -->")
}

// TestCDATASection parses CDATA sections including the tricky ]]> boundary.
func TestCDATASection(t *testing.T) {
	t.Parallel()

	const src = `<root><![CDATA[ raw <tag> & ]]> normal text <child/></root>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err)

	out, err := helium.WriteString(doc)
	require.NoError(t, err)
	require.Contains(t, out, "<![CDATA[")
}

// TestCharacterReferences exercises numeric and hex character references.
func TestCharacterReferences(t *testing.T) {
	t.Parallel()

	const src = `<root>dec=&#65; hex=&#x42; high=&#x1F600;</root>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err)
	require.Equal(t, "dec=A hex=B high=\U0001F600", string(doc.DocumentElement().Content()))
}

func TestParseXML11ControlCharRef(t *testing.T) {
	t.Parallel()

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
}

// TestMalformedDocuments exercises well-formedness error branches across the
// parser. Each input is malformed and must surface an error.
func TestMalformedDocuments(t *testing.T) {
	t.Parallel()

	bad := []string{
		`<root>`,                         // unclosed root
		`<root></notroot>`,               // mismatched end tag
		`<root attr></root>`,             // attribute without value
		`<root attr=value></root>`,       // unquoted attribute value
		`<root>&undefinedentity;</root>`, // reference to undeclared entity
		`<root><![CDATA[ unterminated`,   // unterminated CDATA
		`<!-- unterminated comment`,      // unterminated comment
		`<root>&#xZZ;</root>`,            // invalid hex char ref
		`<root>&;</root>`,                // empty reference
		`<>`,                             // empty tag name
		`<root></root><second/>`,         // two root elements
	}
	for _, in := range bad {
		_, err := helium.NewParser().Parse(t.Context(), []byte(in))
		require.Error(t, err, "malformed input %q should error", in)
	}
}

// TestRecoverOnError exercises the recover path: a malformed document returns
// both a (partial) document and an error.
func TestRecoverOnErrorPartialDoc(t *testing.T) {
	t.Parallel()

	const src = `<root><a>text</a><b></root>`
	doc, err := helium.NewParser().RecoverOnError(true).Parse(t.Context(), []byte(src))
	// With recovery the parser returns a partial document; an error may or may
	// not be reported depending on how far recovery proceeds.
	_ = err
	require.NotNil(t, doc)
}

// TestNamespacedAttributes parses namespaced elements and attributes.
func TestNamespacedAttributes(t *testing.T) {
	t.Parallel()

	const src = `<root xmlns="urn:default" xmlns:p="urn:p" p:attr="v" plain="w">` +
		`<p:child/><plain/></root>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err)

	out, err := helium.WriteString(doc)
	require.NoError(t, err)
	require.Contains(t, out, `p:attr="v"`)
	require.Contains(t, out, `xmlns:p="urn:p"`)
}

// TestParserOptionSetters exercises every boolean parser option setter with both
// true and false (so both the Set and Clear branches run) plus the scalar/object
// setters, then performs a parse to confirm the configured parser still works.
func TestParserOptionSetters(t *testing.T) {
	t.Parallel()

	p := helium.NewParser().
		RecoverOnError(true).RecoverOnError(false).
		SubstituteEntities(true).SubstituteEntities(false).
		LoadExternalDTD(true).LoadExternalDTD(false).
		DefaultDTDAttributes(true).DefaultDTDAttributes(false).
		ValidateDTD(true).ValidateDTD(false).
		SuppressErrors(true).SuppressErrors(false).
		SuppressWarnings(true).SuppressWarnings(false).
		PedanticErrors(true).PedanticErrors(false).
		StripBlanks(true).StripBlanks(false).
		ProcessXInclude(true).ProcessXInclude(false).
		AllowNetwork(true).AllowNetwork(false).
		CleanNamespaces(true).CleanNamespaces(false).
		MergeCDATA(true).MergeCDATA(false).
		XIncludeNodes(true).XIncludeNodes(false).
		CompactTextNodes(true).CompactTextNodes(false).
		FixBaseURIs(true).FixBaseURIs(false).
		MaxNameLength(-1).MaxNameLength(0).
		MaxEntityAmplification(-1).MaxEntityAmplification(0).
		MaxContentModelDepth(-1).MaxContentModelDepth(0).
		IgnoreEncoding(true).IgnoreEncoding(false).
		BigLineNumbers(true).BigLineNumbers(false).
		BlockXXE(true).BlockXXE(false).
		ReuseDict(true).ReuseDict(false).
		SkipIDs(true).SkipIDs(false).
		LenientXMLDecl(true).LenientXMLDecl(false).
		CharBufferSize(8192).
		MaxDepth(256).
		MaxExternalDTDBytes(1 << 20).
		Catalog(nopCatalog{}).
		BaseURI("http://example.com/base.xml")

	doc, err := p.Parse(t.Context(), []byte(`<?xml version="1.0"?><root><child>text</child></root>`))
	require.NoError(t, err, "a fully-configured parser parses a simple document")
	require.NotNil(t, doc.DocumentElement())
}

// TestParserCharBufferSizeAffectsParse confirms a tiny char buffer (which forces
// repeated cursor refills) still parses a larger document correctly.
func TestParserCharBufferSizeAffectsParse(t *testing.T) {
	t.Parallel()

	var b []byte
	b = append(b, []byte(`<root>`)...)
	for range 200 {
		b = append(b, []byte(`<item>x</item>`)...)
	}
	b = append(b, []byte(`</root>`)...)

	doc, err := helium.NewParser().CharBufferSize(16).Parse(t.Context(), b)
	require.NoError(t, err)
	require.Equal(t, "root", doc.DocumentElement().Name())
}

// genCharDataReader lazily produces "<root>" + n copies of fill + "</root>"
// without ever holding the whole payload in memory, so the parser's own
// character-data buffering is the only large allocation under test. It records
// the cumulative number of payload bytes it has handed out in nread, which the
// memory-bounds test samples at the first SAX callback to prove the parser
// streams rather than draining the whole run before delivering anything.
type genCharDataReader struct {
	prefix []byte
	fill   byte
	remain int
	suffix []byte
	nread  int
}

func (r *genCharDataReader) Read(p []byte) (int, error) {
	n := 0
	if len(r.prefix) > 0 {
		c := copy(p, r.prefix)
		r.prefix = r.prefix[c:]
		n += c
	}
	if n < len(p) && r.remain > 0 {
		c := min(len(p)-n, r.remain)
		for i := range c {
			p[n+i] = r.fill
		}
		r.remain -= c
		n += c
	}
	if n < len(p) && len(r.prefix) == 0 && r.remain == 0 && len(r.suffix) > 0 {
		c := copy(p[n:], r.suffix)
		r.suffix = r.suffix[c:]
		n += c
	}
	if n == 0 {
		return 0, io.EOF
	}
	r.nread += n
	return n, nil
}

// TestParserCharBufferSizeBoundsCharDataMemory verifies that a large
// delimiter-free character-data run delivered to a streaming SAX consumer is
// scanned and delivered in bounded chunks rather than materialized whole. Before
// the fix the entire run was buffered (in charBuf and the cursor's internal
// buffer) before the first chunk was delivered.
//
// Rather than sampling a global heap signal (non-deterministic, especially
// under t.Parallel), this instruments the reader: the parser pulls bytes from r
// on demand, so the count of bytes read at the first SAX callback shows whether
// delivery began before the whole payload was drained. A streaming parser fires
// the first callback after reading only a bounded prefix (one cursor buffer ≈
// 8 KiB), far short of the multi-megabyte run; the pre-fix whole-run buffering
// would have drained nearly all of it first.
func TestParserCharBufferSizeBoundsCharDataMemory(t *testing.T) {
	t.Parallel()

	const fillBytes = 32 << 20 // 32 MiB delimiter-free run
	const bufSize = 8192

	r := &genCharDataReader{
		prefix: []byte("<root>"),
		fill:   'a',
		remain: fillBytes,
		suffix: []byte("</root>"),
	}

	var chunkCount, total, maxChunk, readAtFirstChunk int

	handler := sax.New()
	record := func(_ context.Context, ch []byte) error {
		if chunkCount == 0 {
			readAtFirstChunk = r.nread
		}
		chunkCount++
		total += len(ch)
		maxChunk = max(maxChunk, len(ch))
		return nil
	}
	handler.SetOnCharacters(sax.CharactersFunc(record))
	handler.SetOnIgnorableWhitespace(sax.IgnorableWhitespaceFunc(record))

	p := helium.NewParser().SAXHandler(handler).CharBufferSize(bufSize)
	_, err := p.ParseReader(context.Background(), r)
	require.NoError(t, err)

	require.Equal(t, fillBytes, total, "every character byte is delivered")
	require.LessOrEqual(t, maxChunk, bufSize, "no chunk exceeds the configured buffer size")
	require.Greater(t, chunkCount, 1, "the run is delivered in multiple chunks")

	// The first callback must fire before the whole run has been read; a bound
	// well below the payload (yet generously above one cursor buffer) proves the
	// scan buffer stays bounded instead of materializing the whole run first.
	require.Less(t, readAtFirstChunk, 1<<20,
		"streaming must deliver the first chunk after reading only a bounded prefix; read %d bytes first", readAtFirstChunk)
}

// TestParserCharBufferSizeConsistentWhitespaceClassification pins the chunked
// streaming-SAX path's whitespace classification against the single-shot path,
// which classifies the WHOLE run as one unit. The single-shot path looks over
// the whole run to decide ignorable-whitespace vs. character data; the chunked
// path delivers in bounded pieces and cannot, so it must not emit any
// IgnorableWhitespace event until the whole run is proven blank: a fully-blank
// run is ignorable whitespace, but a run containing any text is character data
// in its entirety, leading blanks included.
//
// This guards against two earlier bugs: per-chunk classification, where a blank
// run could arrive as Characters then IgnorableWhitespace under a tiny buffer;
// and "sticky downgrade", where the leading blanks of <root>  text</root> were
// emitted as an early IgnorableWhitespace chunk before the text was seen.
func TestParserCharBufferSizeConsistentWhitespaceClassification(t *testing.T) {
	t.Parallel()

	type event struct {
		ignorable bool
		content   string
	}

	run := func(src string, bufSize int) []event {
		var events []event
		handler := sax.New()
		handler.SetOnCharacters(sax.CharactersFunc(func(_ context.Context, ch []byte) error {
			events = append(events, event{ignorable: false, content: string(ch)})
			return nil
		}))
		handler.SetOnIgnorableWhitespace(sax.IgnorableWhitespaceFunc(func(_ context.Context, ch []byte) error {
			events = append(events, event{ignorable: true, content: string(ch)})
			return nil
		}))
		_, err := helium.NewParser().SAXHandler(handler).CharBufferSize(bufSize).
			Parse(context.Background(), []byte(src))
		require.NoError(t, err)
		return events
	}

	concat := func(events []event) string {
		var b strings.Builder
		for _, e := range events {
			b.WriteString(e.content)
		}
		return b.String()
	}

	t.Run("all-whitespace run stays ignorable across chunks", func(t *testing.T) {
		t.Parallel()
		events := run("<root>        </root>", 2)
		require.Greater(t, len(events), 1, "the tiny buffer must split the run")
		require.Equal(t, "        ", concat(events), "every whitespace byte is delivered")
		for _, e := range events {
			require.True(t, e.ignorable, "a fully-blank run must classify every chunk as ignorable whitespace")
		}
	})

	t.Run("run containing text is all characters, no leading ignorable", func(t *testing.T) {
		t.Parallel()
		events := run("<root>      text</root>", 2)
		require.Equal(t, "      text", concat(events), "every character byte is delivered")
		require.NotEmpty(t, events, "a run containing text must deliver characters")
		for _, e := range events {
			require.False(t, e.ignorable,
				"a run containing text is character data in its entirety; leading blanks must not be emitted as ignorable whitespace")
		}
	})
}

// TestParserCharBufferSizeWhitespaceBeforeEntity pins the chunked streaming-SAX
// path's classification of an all-whitespace run that ends at an entity
// reference ('&') rather than a start/end tag ('<'). The single-shot path
// (areBlanksBytes) treats such a run as character data — it is ignorable
// whitespace only when the delimiter that ends it is '<' or CR. An earlier
// chunked implementation dropped that trailing-delimiter check and misreported
// the leading blanks as IgnorableWhitespace. Both paths must agree.
func TestParserCharBufferSizeWhitespaceBeforeEntity(t *testing.T) {
	t.Parallel()

	type event struct {
		ignorable bool
		content   string
	}

	run := func(src string, bufSize int) []event {
		var events []event
		handler := sax.New()
		handler.SetOnCharacters(sax.CharactersFunc(func(_ context.Context, ch []byte) error {
			events = append(events, event{ignorable: false, content: string(ch)})
			return nil
		}))
		handler.SetOnIgnorableWhitespace(sax.IgnorableWhitespaceFunc(func(_ context.Context, ch []byte) error {
			events = append(events, event{ignorable: true, content: string(ch)})
			return nil
		}))
		_, err := helium.NewParser().SAXHandler(handler).CharBufferSize(bufSize).
			Parse(context.Background(), []byte(src))
		require.NoError(t, err)
		return events
	}

	// Leading whitespace, then an entity reference. The run "   " ends at '&',
	// so it is character data, not ignorable whitespace.
	const src = `<root>   &amp;</root>`

	// Single-shot path (no chunking) is the reference classification.
	single := run(src, 0)
	require.NotEmpty(t, single)
	for _, e := range single {
		require.False(t, e.ignorable,
			"single-shot: a whitespace run ending at '&' is character data, not ignorable whitespace")
	}

	// Chunked path must match: no IgnorableWhitespace event, leading blanks
	// delivered as characters.
	chunked := run(src, 2)
	require.NotEmpty(t, chunked)
	var b strings.Builder
	for _, e := range chunked {
		require.False(t, e.ignorable,
			"chunked: a whitespace run ending at '&' must match the single-shot path (character data)")
		b.WriteString(e.content)
	}
	require.Equal(t, "   &", b.String(), "every character byte (including leading blanks) is delivered")
}

// TestParserCharBufferSizeBoundsWhitespaceMemory verifies that a large
// delimiter-free all-whitespace run delivered to a streaming SAX consumer is
// bounded in memory. A blank run cannot be proven ignorable whitespace until
// its end is in view, so a naive implementation buffers it whole before the
// first callback. The bounded-whitespace policy downgrades an over-budget blank
// prefix to character data and streams the rest in fixed-size chunks, so the
// first callback fires after reading only a bounded prefix. This mirrors
// TestParserCharBufferSizeBoundsCharDataMemory but uses a whitespace fill byte.
func TestParserCharBufferSizeBoundsWhitespaceMemory(t *testing.T) {
	t.Parallel()

	const fillBytes = 32 << 20 // 32 MiB delimiter-free whitespace run
	const bufSize = 8192

	r := &genCharDataReader{
		prefix: []byte("<root>"),
		fill:   ' ',
		remain: fillBytes,
		suffix: []byte("</root>"),
	}

	var chunkCount, total, maxChunk, readAtFirstChunk int
	sawIgnorable := false

	handler := sax.New()
	record := func(ignorable bool) func(context.Context, []byte) error {
		return func(_ context.Context, ch []byte) error {
			if chunkCount == 0 {
				readAtFirstChunk = r.nread
			}
			if ignorable {
				sawIgnorable = true
			}
			chunkCount++
			total += len(ch)
			maxChunk = max(maxChunk, len(ch))
			return nil
		}
	}
	handler.SetOnCharacters(sax.CharactersFunc(record(false)))
	handler.SetOnIgnorableWhitespace(sax.IgnorableWhitespaceFunc(record(true)))

	p := helium.NewParser().SAXHandler(handler).CharBufferSize(bufSize)
	_, err := p.ParseReader(context.Background(), r)
	require.NoError(t, err)

	require.Equal(t, fillBytes, total, "every whitespace byte is delivered")
	require.LessOrEqual(t, maxChunk, bufSize, "no chunk exceeds the configured buffer size")
	require.Greater(t, chunkCount, 1, "the run is delivered in multiple chunks")
	require.False(t, sawIgnorable,
		"an over-budget blank run is downgraded to character data, not buffered whole as ignorable whitespace")

	// The first callback must fire well before the whole run has been read,
	// proving the blank prefix is bounded rather than materialized whole.
	require.Less(t, readAtFirstChunk, 1<<20,
		"bounded-whitespace policy must deliver the first chunk after reading only a bounded prefix; read %d bytes first", readAtFirstChunk)
}

// TestParserCharBufferSizeNeverSplitsRune verifies the chunked streaming-SAX
// path never splits a UTF-8 rune, even when CharBufferSize is smaller than a
// single multi-byte rune. ScanCharDataSlice returns a lone over-budget rune
// whole (to guarantee progress), and deliverCharacters must then deliver it
// whole rather than slicing it into invalid UTF-8 fragments.
func TestParserCharBufferSizeNeverSplitsRune(t *testing.T) {
	t.Parallel()

	// Mix 2-, 3-, and 4-byte runes so a 1-byte buffer is narrower than every
	// rune in the run.
	const content = "héllo—世界🌍ok"

	var chunks []string
	handler := sax.New()
	collect := func(_ context.Context, ch []byte) error {
		require.True(t, utf8.Valid(ch), "no chunk may contain a split (invalid) UTF-8 rune: %q", ch)
		chunks = append(chunks, string(ch))
		return nil
	}
	handler.SetOnCharacters(sax.CharactersFunc(collect))
	handler.SetOnIgnorableWhitespace(sax.IgnorableWhitespaceFunc(collect))

	_, err := helium.NewParser().SAXHandler(handler).CharBufferSize(1).
		Parse(context.Background(), []byte("<root>"+content+"</root>"))
	require.NoError(t, err)

	require.Equal(t, content, strings.Join(chunks, ""), "every byte is delivered intact")
}

// nopCatalog is a CatalogResolver that never resolves anything. It exists only
// to drive the Parser.Catalog configuration path.
type nopCatalog struct{}

func (nopCatalog) Resolve(_ context.Context, _, _ string) string { return "" }
func (nopCatalog) ResolveURI(_ context.Context, _ string) string { return "" }

// TestParserMalformedBranches feeds a battery of distinct malformed inputs, each
// designed to drive a specific parser error branch (XML declaration version /
// encoding / standalone parsing, PI target and delimiter rules, comment and
// CDATA termination, QName / Name lexical errors). Every input must be rejected;
// the value is in exercising the otherwise-unreached error returns.
func TestParserMalformedBranches(t *testing.T) {
	t.Parallel()

	bad := []struct {
		name string
		src  string
	}{
		{"xml decl version unquoted", `<?xml version=1.0?><root/>`},
		{"xml decl bad standalone", `<?xml version="1.0" standalone="maybe"?><root/>`},
		{"xml decl encoding unquoted", `<?xml version="1.0" encoding=UTF-8?><root/>`},
		{"xml decl encoding bad first char", `<?xml version="1.0" encoding="1bad"?><root/>`},
		{"xml decl missing version", `<?xml encoding="UTF-8"?><root/>`},
		{"pi target named xml mid-document", `<root><?xml data?></root>`},
		{"pi missing space after target", `<root><?targetdata</root>`},
		{"pi unterminated", `<root><?target data </root>`},
		{"comment with double hyphen", `<root><!-- a -- b --></root>`},
		{"cdata unterminated", `<root><![CDATA[unterminated</root>`},
		{"bad qname trailing colon", `<root:></root:>`},
		{"name starts with digit", `<1root/>`},
		{"attribute missing equals", `<root attr "v"/>`},
		{"unterminated start tag", `<root attr="v"`},
		{"text with raw less-than via entity ok but bad amp", `<root>a & b</root>`},
	}

	for _, tc := range bad {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := helium.NewParser().Parse(context.Background(), []byte(tc.src))
			require.Error(t, err, "malformed input %q must be rejected", tc.src)
		})
	}
}

// TestParserWellFormedVariety parses a variety of well-formed constructs that
// exercise the success branches of the same parser functions the malformed tests
// hit on the error side: a leading PI, a comment, a CDATA section, namespaced
// elements/attributes, character references, and an explicit encoding/standalone
// declaration.
func TestParserWellFormedVariety(t *testing.T) {
	t.Parallel()

	const src = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<?pi-target some data?>
<!-- a leading comment -->
<p:root xmlns:p="urn:p" xmlns="urn:default" p:attr="v" plain="w">
  <![CDATA[ raw <markup> & stuff ]]>
  text &#65; &#x42; &amp; more
  <p:child/>
  <plain-child attr="x"/>
</p:root>`

	doc, err := helium.NewParser().Parse(context.Background(), []byte(src))
	require.NoError(t, err)
	root := doc.DocumentElement()
	require.NotNil(t, root)
	require.Equal(t, "root", root.LocalName())

	out, err := helium.WriteString(doc)
	require.NoError(t, err)
	require.Contains(t, out, "urn:p")
	require.Contains(t, out, "CDATA")
}

// TestEncodingDeclarations parses documents with various encoding declarations
// to exercise the encoding-switch paths.
func TestEncodingDeclarations(t *testing.T) {
	t.Parallel()

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
}

// TestEncodingIgnored verifies the IgnoreEncoding option does not break a parse
// that declares an encoding.
func TestEncodingIgnored(t *testing.T) {
	t.Parallel()

	const in = `<?xml version="1.0" encoding="ISO-8859-1"?><root>x</root>`
	doc, err := helium.NewParser().IgnoreEncoding(true).Parse(t.Context(), []byte(in))
	require.NoError(t, err)
	require.NotNil(t, doc.DocumentElement())
}

// TestParseOverCapWhitespaceInDeclAndDTD guards that an over-cap whitespace run
// inside the XML DECLARATION and inside the DTD internal subset surfaces
// ErrNodeContentTooLarge, not a generic XML-decl / DTD syntax error. The
// blank-skip helpers only return a bool, so the callers in those positions keep
// going after the sticky over-cap error is recorded; without the central
// preference for the blank-run error in errorAtLevel the follow-on syntax error
// (a malformed version info / "DOCTYPE not finished") would mask the real cause.
func TestParseOverCapWhitespaceInDeclAndDTD(t *testing.T) {
	t.Parallel()

	const limit = 4096
	blanks := strings.Repeat(" ", limit*2)

	cases := map[string]string{
		// Whitespace between '<?xml' and the version pseudo-attribute.
		"xml declaration whitespace": "<?xml" + blanks + `version="1.0"?><root/>`,
		// Whitespace inside the DTD internal subset.
		"dtd subset whitespace": `<?xml version="1.0"?><!DOCTYPE root [` + blanks + `]><root/>`,
	}

	for name, doc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := helium.NewParser().MaxNodeContentSize(limit).Parse(t.Context(), []byte(doc))
			require.ErrorIs(t, err, helium.ErrNodeContentTooLarge,
				"over-cap %s must surface ErrNodeContentTooLarge, not a masking syntax error", name)
		})
	}
}

// TestParseOverCapWhitespaceInExternalSubset pins the blank-run cap on the two
// external-subset blank skips that intentionally bypass skipBlanks to preserve
// %pe; expansion: the declaration-step skip (parseExternalSubsetDeclStep) and
// the INCLUDE-terminator skip (parseConditionalSections). Both now route through
// skipBlankRun, so an over-cap contiguous whitespace run in either position must
// fail with ErrNodeContentTooLarge instead of forcing the cursor to buffer the
// whole run.
func TestParseOverCapWhitespaceInExternalSubset(t *testing.T) {
	t.Parallel()

	const limit = 4096
	blanks := strings.Repeat(" ", limit*2)

	const input = `<?xml version="1.0"?>
<!DOCTYPE r SYSTEM "ws.dtd">
<r/>`

	cases := map[string]string{
		// Whitespace between two declarations in the external subset, consumed by
		// parseExternalSubsetDeclStep's blank-only skip.
		"external subset declaration step": "<!ELEMENT r EMPTY>" + blanks + "<!ATTLIST r x CDATA 'd'>",
		// Whitespace just before the INCLUDE section's "]]>" terminator, consumed
		// by parseConditionalSections's section-cursor blank skip.
		"include section terminator": "<![INCLUDE[<!ELEMENT r EMPTY>" + blanks + "]]>",
	}

	for name, dtd := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			fsys := fstest.MapFS{"ws.dtd": &fstest.MapFile{Data: []byte(dtd)}}
			p := helium.NewParser().
				BlockXXE(false).
				LoadExternalDTD(true).
				DefaultDTDAttributes(true).
				MaxNodeContentSize(limit).
				FS(fsys)
			_, err := p.Parse(t.Context(), []byte(input))
			require.ErrorIs(t, err, helium.ErrNodeContentTooLarge,
				"over-cap whitespace in %s must surface ErrNodeContentTooLarge", name)
		})
	}
}

// TestParseOverCapWhitespaceInConditionalSectionHeader pins the blank-run cap on
// the conditional-section HEADER whitespace skips in parseConditionalSections:
// after "<![", after the INCLUDE keyword, and after the IGNORE keyword. These
// positions route through skipBlanks (which records ErrNodeContentTooLarge in
// pctx.blankRunErr but returns the conditional-section sentinel), and that
// sentinel was previously TOLERATED by the top-level external-subset loop —
// silently downgrading a resource-limit violation to "stop parsing the subset".
// Each over-cap run in the header must instead fail closed with
// ErrNodeContentTooLarge. Distinct from over-limit whitespace inside the INCLUDE
// body / before its terminator (covered by TestParseOverCapWhitespaceInExternalSubset).
func TestParseOverCapWhitespaceInConditionalSectionHeader(t *testing.T) {
	t.Parallel()

	const limit = 4096
	blanks := strings.Repeat(" ", limit*2)

	const input = `<?xml version="1.0"?>
<!DOCTYPE r SYSTEM "ws.dtd">
<r/>`

	cases := map[string]string{
		// Whitespace immediately after "<![", before the INCLUDE keyword.
		"after open bracket": "<![" + blanks + "INCLUDE[<!ELEMENT r EMPTY>]]>",
		// Whitespace after the INCLUDE keyword, before its "[".
		"after include keyword": "<![INCLUDE" + blanks + "[<!ELEMENT r EMPTY>]]>",
		// Whitespace after the IGNORE keyword, before its "[".
		"after ignore keyword": "<!ELEMENT r EMPTY><![IGNORE" + blanks + "[ <!ELEMENT q EMPTY> ]]>",
	}

	for name, dtd := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			fsys := fstest.MapFS{"ws.dtd": &fstest.MapFile{Data: []byte(dtd)}}
			p := helium.NewParser().
				BlockXXE(false).
				LoadExternalDTD(true).
				DefaultDTDAttributes(true).
				MaxNodeContentSize(limit).
				FS(fsys)
			_, err := p.Parse(t.Context(), []byte(input))
			require.ErrorIs(t, err, helium.ErrNodeContentTooLarge,
				"over-cap whitespace %s must surface ErrNodeContentTooLarge (not be tolerated as a conditional-section sentinel)", name)
		})
	}
}

// TestParseReaderEBCDICOverCapWhitespaceInExternalSubset proves the
// external-subset blank-run cap holds regardless of how the MAIN document is
// delivered: an EBCDIC document fed through ParseReader, whose external subset
// (loaded from the fs.FS) carries an over-cap contiguous whitespace run between
// declarations, must still fail closed with ErrNodeContentTooLarge instead of
// letting the external-subset declaration step buffer the whole run. The same
// EBCDIC bytes via Parse([]byte) must fail identically — parity between the two
// entry points.
func TestParseReaderEBCDICOverCapWhitespaceInExternalSubset(t *testing.T) {
	t.Parallel()

	const limit = 4096
	blanks := strings.Repeat(" ", limit*2)
	dtd := "<!ELEMENT r EMPTY>" + blanks + "<!ATTLIST r x CDATA 'd'>"

	const decl = `<?xml version="1.0" encoding="IBM037"?>` + "\n" +
		`<!DOCTYPE r SYSTEM "ws.dtd">` + "\n" + `<r/>`
	ebcdic, err := charmap.CodePage037.NewEncoder().Bytes([]byte(decl))
	require.NoError(t, err)
	require.Equal(t, []byte{0x4C, 0x6F, 0xA7, 0x94}, ebcdic[:4],
		"encoded bytes must start with the EBCDIC invariant prefix")

	newParser := func() helium.Parser {
		return helium.NewParser().
			BlockXXE(false).
			LoadExternalDTD(true).
			DefaultDTDAttributes(true).
			MaxNodeContentSize(limit).
			FS(fstest.MapFS{"ws.dtd": &fstest.MapFile{Data: []byte(dtd)}})
	}

	_, rerr := newParser().ParseReader(t.Context(), bytes.NewReader(ebcdic))
	require.ErrorIs(t, rerr, helium.ErrNodeContentTooLarge,
		"over-cap external-subset whitespace must surface ErrNodeContentTooLarge via ParseReader/EBCDIC")

	_, berr := newParser().Parse(t.Context(), ebcdic)
	require.ErrorIs(t, berr, helium.ErrNodeContentTooLarge,
		"the same EBCDIC bytes via Parse([]byte) must fail identically")
}

// TestExternalSubsetPEReferenceAfterWhitespaceStillExpands guards the property
// the blank-run cap MUST NOT break: the external-subset declaration step uses a
// blank-ONLY skip (skipBlankRun) precisely so a "%pe;" reference that follows
// whitespace is left for parsePEReference to expand, rather than being consumed
// by skipBlanks/handlePEReference without pushing its replacement text. With a
// non-trivial (but under-cap) whitespace run before "%pe;", the PE must still
// expand and its declarations apply — here a general entity declared inside the
// PE is registered.
func TestExternalSubsetPEReferenceAfterWhitespaceStillExpands(t *testing.T) {
	t.Parallel()

	ws := strings.Repeat(" ", 2048)
	fsys := fstest.MapFS{
		dtdSystemID: {Data: []byte(
			`<!ENTITY ctrl "control">` + "\n" +
				`<!ENTITY % pe SYSTEM "pe.ent">` + ws + `%pe;`)},
		peSystemID: {Data: []byte(`<!ENTITY fromPE "loaded-from-external-pe">`)},
	}
	const input = `<?xml version="1.0"?>` + "\n" +
		`<!DOCTYPE r SYSTEM "d.dtd"><r/>`

	doc, err := helium.NewParser().
		BlockXXE(false).
		LoadExternalDTD(true).
		MaxNodeContentSize(4096).
		FS(fsys).
		Parse(t.Context(), []byte(input))
	require.NoError(t, err, "under-cap whitespace before %%pe; must not break parsing")
	require.NotNil(t, doc)

	ent, ok := doc.GetEntity("fromPE")
	require.True(t, ok, "the PE following whitespace must still expand and register its declarations")
	require.Equal(t, "loaded-from-external-pe", string(ent.Content()))
}

// blockUntilCancelledBlankReader serves a fixed head, then (on the first read
// past the head) signals it has reached the trailing whitespace run and blocks
// until the test cancels the context, after which it streams ASCII spaces
// forever. It drives a cancellation first observable while the parser is
// scanning a whitespace run in the XML declaration or DTD subset.
type blockUntilCancelledBlankReader struct {
	head      []byte
	pos       int
	entered   chan struct{}
	cancelled chan struct{}
	once      sync.Once
}

func (r *blockUntilCancelledBlankReader) Read(p []byte) (int, error) {
	if r.pos < len(r.head) {
		n := copy(p, r.head[r.pos:])
		r.pos += n
		return n, nil
	}
	r.once.Do(func() {
		close(r.entered)
		<-r.cancelled
	})
	for i := range p {
		p[i] = ' '
	}
	return len(p), nil
}

// TestParseReaderCancelInDeclAndDTDWhitespace guards the cancellation contract
// in the XML DECLARATION and DTD whitespace positions (not only prolog /
// epilogue): a context cancelled while the parser is scanning whitespace there
// must surface as context.Canceled with no partial document, never as a syntax
// error. The blank-skip helpers only return a bool, so without the central
// preference for the sticky blank-run error in errorAtLevel a follow-on syntax
// error (e.g. "blank needed after '<?xml'") would mask the cancellation.
func TestParseReaderCancelInDeclAndDTDWhitespace(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"xml declaration whitespace": "<?xml",
		"dtd subset whitespace":      `<?xml version="1.0"?><!DOCTYPE root [`,
	}

	for name, head := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			r := &blockUntilCancelledBlankReader{
				head:      []byte(head),
				entered:   make(chan struct{}),
				cancelled: make(chan struct{}),
			}

			ctx, cancel := context.WithCancel(context.Background())

			type result struct {
				doc *helium.Document
				err error
			}
			resCh := make(chan result, 1)
			go func() {
				// A generous cap so the over-cap guard does not fire first; the
				// cancellation must win.
				doc, err := helium.NewParser().MaxNodeContentSize(1<<20).ParseReader(ctx, r)
				resCh <- result{doc, err}
			}()

			select {
			case <-r.entered:
			case <-time.After(2 * time.Second):
				close(r.cancelled) // unblock the reader so the goroutine can exit
				t.Fatalf("parser did not reach the %s run", name)
			}
			cancel()
			close(r.cancelled)

			select {
			case res := <-resCh:
				require.ErrorIs(t, res.err, context.Canceled,
					"cancellation while scanning %s must surface as context.Canceled", name)
				require.Nil(t, res.doc, "a cancelled parse must not return a partial document")
			case <-time.After(2 * time.Second):
				t.Fatalf("ParseReader did not return after cancellation in %s", name)
			}
		})
	}
}

// headThenReadErrReader serves head bytes once, then fails every subsequent
// Read with err. It models the push/streaming stream whose blocking Read
// returns context.Canceled when cancellation unblocks a pending wait: the
// ByteCursor records that as a sticky Err() while PeekAt reports 0, the same 0
// a genuine non-blank byte / clean EOF yields. ctx is never cancelled here, so
// the ONLY signal of the failure is the cursor's sticky read error.
type headThenReadErrReader struct {
	head []byte
	pos  int
	err  error
}

func (r *headThenReadErrReader) Read(p []byte) (int, error) {
	if r.pos < len(r.head) {
		n := copy(p, r.head[r.pos:])
		r.pos += n
		return n, nil
	}
	return 0, r.err
}

// TestParseReaderReadErrorInXMLDeclNotMaskedAsSyntaxError pins the cancellation
// contract for a read failure that lands RIGHT AFTER "<?xml", before the
// declaration's required trailing blank has been read. Because the sixth byte
// is never delivered, looksLikeXMLDecl cannot confirm the declaration and the
// parser falls through to treating "<?xml" as a processing instruction, whose
// reserved "xml" target then synthesizes "XML declaration allowed only at the
// start of the document". That synthesized syntax error must NOT mask the
// underlying read failure: a parse whose stream fails (a push-stream Read
// returning context.Canceled on cancellation) must surface that error, never a
// malformed-document diagnostic, and must return no partial document.
//
// ctx is context.Background() (never cancelled) on purpose: the only signal of
// the failure is the cursor's sticky Err(), exactly as in the push/streaming
// path where the stream Read returns the context error.
func TestParseReaderReadErrorInXMLDeclNotMaskedAsSyntaxError(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		// Read fails immediately after "<?xml" with no trailing blank read, so
		// looksLikeXMLDecl is false and "<?xml" is reparsed as a reserved PI.
		"no blank after <?xml": "<?xml",
		// A blank was read but the read fails before the version pseudo-attribute,
		// so the declaration parse begins and then stalls on the failed read.
		"blank then read error": "<?xml ",
	}

	for name, head := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			r := &headThenReadErrReader{head: []byte(head), err: context.Canceled}
			doc, err := helium.NewParser().ParseReader(context.Background(), r)
			require.ErrorIs(t, err, context.Canceled,
				"a read failure in the XML declaration (%s) must surface as context.Canceled, not a synthesized syntax error", name)
			require.Nil(t, doc, "a failed parse must not return a partial document")
		})
	}
}
