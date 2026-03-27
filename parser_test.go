package helium_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
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
		`<?xml version="1.0"?>` + content:                                   {"1.0", "utf8", int(helium.StandaloneImplicitNo)},
		`<?xml version="1.0" encoding="euc-jp"?>` + content:                 {"1.0", "euc-jp", int(helium.StandaloneImplicitNo)},
		`<?xml version="1.0" encoding="cp932" standalone='yes'?>` + content: {"1.0", "cp932", int(helium.StandaloneExplicitYes)},
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

func TestDisableSAXRecoverContinuesParsing(t *testing.T) {
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
}

func TestDisableSAXCallbacksSuppressed(t *testing.T) {
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
}

func TestDisableSAXNoEffectWithoutRecover(t *testing.T) {
	const input = `<?xml version="1.0"?>
<root>
  <bad>text</baaaad>
  <after/>
</root>`

	p := helium.NewParser()
	doc, err := p.Parse(t.Context(), []byte(input))
	require.Error(t, err, "malformed XML should fail")
	require.Nil(t, doc, "without RecoverOnError, no document should be returned")
}

func TestParseExternalEntity(t *testing.T) {
	const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ENTITY ext SYSTEM "ext.xml">
]>
<doc>&ext;</doc>`

	// Provide a ResolveEntity handler that returns inline content
	s := sax.New()
	s.SetOnResolveEntity(sax.ResolveEntityFunc(func(_ context.Context, publicID, systemID string) (sax.ParseInput, error) {
		if systemID == "ext.xml" {
			return newStringParseInput("<inner>hello</inner>", systemID), nil
		}
		return nil, sax.ErrHandlerUnspecified
	}))

	p := helium.NewParser().SAXHandler(s).SubstituteEntities(true)
	_, err := p.Parse(t.Context(), []byte(input))
	require.NoError(t, err, "Parse with external entity should succeed")
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

func TestValidateDTDRequiredAttribute(t *testing.T) {
	// #REQUIRED attribute missing → validation error
	const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc EMPTY>
  <!ATTLIST doc id ID #REQUIRED>
]>
<doc/>`

	p := helium.NewParser().ValidateDTD(true)
	doc, err := p.Parse(t.Context(), []byte(input))
	require.Error(t, err, "missing #REQUIRED attribute should fail validation")
	require.NotNil(t, doc, "document should still be returned with validation error")

	require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
}

func TestValidateDTDRequiredPresent(t *testing.T) {
	// #REQUIRED attribute present → no error
	const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc EMPTY>
  <!ATTLIST doc id ID #REQUIRED>
]>
<doc id="x1"/>`

	p := helium.NewParser().ValidateDTD(true)
	_, err := p.Parse(t.Context(), []byte(input))
	require.NoError(t, err)
}

func TestValidateDTDFixedMismatch(t *testing.T) {
	// #FIXED attribute with wrong value → validation error
	const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc EMPTY>
  <!ATTLIST doc version CDATA #FIXED "1.0">
]>
<doc version="2.0"/>`

	p := helium.NewParser().ValidateDTD(true)
	_, err := p.Parse(t.Context(), []byte(input))
	require.Error(t, err, "#FIXED attribute value mismatch should fail")

	require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
}

func TestValidateDTDFixedCorrect(t *testing.T) {
	// #FIXED attribute with correct value → no error
	const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc EMPTY>
  <!ATTLIST doc version CDATA #FIXED "1.0">
]>
<doc version="1.0"/>`

	p := helium.NewParser().ValidateDTD(true)
	_, err := p.Parse(t.Context(), []byte(input))
	require.NoError(t, err)
}

func TestValidateDTDEmptyElement(t *testing.T) {
	// EMPTY element with content → validation error
	const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (child)>
  <!ELEMENT child EMPTY>
]>
<doc><child>text</child></doc>`

	p := helium.NewParser().ValidateDTD(true)
	_, err := p.Parse(t.Context(), []byte(input))
	require.Error(t, err, "EMPTY element with content should fail")

	require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
}

func TestValidateDTDElementContent(t *testing.T) {
	// Element content model (a, b) with correct content → no error
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
}

func TestValidateDTDElementContentMismatch(t *testing.T) {
	// Element content model (a, b) with (b, a) → validation error
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
}

func TestValidateDTDMixedContent(t *testing.T) {
	// Mixed content (#PCDATA | a)* — text and <a> are allowed
	const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (#PCDATA | a)*>
  <!ELEMENT a (#PCDATA)>
]>
<doc>hello <a>world</a> end</doc>`

	p := helium.NewParser().ValidateDTD(true)
	_, err := p.Parse(t.Context(), []byte(input))
	require.NoError(t, err)
}

func TestValidateDTDMixedContentBadChild(t *testing.T) {
	// Mixed content (#PCDATA | a)* — <b> is NOT allowed
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
}

func TestValidateDTDNoFlag(t *testing.T) {
	// Same invalid document but WITHOUT ValidateDTD → should pass
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
}

func TestValidateDTDChoiceContent(t *testing.T) {
	// Choice content model (a | b) with <a> → valid
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
}

func TestValidateDTDRepeatContent(t *testing.T) {
	// Repetition content model (a)+ with multiple <a> → valid
	const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (a)+>
  <!ELEMENT a (#PCDATA)>
]>
<doc><a>1</a><a>2</a><a>3</a></doc>`

	p := helium.NewParser().ValidateDTD(true)
	_, err := p.Parse(t.Context(), []byte(input))
	require.NoError(t, err)
}

func TestValidateDTDRepeatContentEmpty(t *testing.T) {
	// Repetition content model (a)+ with zero <a> → invalid
	const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (a)+>
  <!ELEMENT a (#PCDATA)>
]>
<doc></doc>`

	p := helium.NewParser().ValidateDTD(true)
	_, err := p.Parse(t.Context(), []byte(input))
	require.Error(t, err, "(a)+ requires at least one <a>")
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

func TestValidateDTDIDUnique(t *testing.T) {
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
}

func TestValidateDTDIDDuplicate(t *testing.T) {
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
	_ = collector.Close()
	require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
	require.True(t, containsError(collector.Errors(), "duplicate ID"))
}

func TestValidateDTDIDRefValid(t *testing.T) {
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
}

func TestValidateDTDIDRefMissing(t *testing.T) {
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
	_ = collector.Close()
	require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
	require.True(t, containsError(collector.Errors(), "unknown ID"))
}

func TestValidateDTDIDRefsValid(t *testing.T) {
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
}

func TestValidateDTDIDRefsMissing(t *testing.T) {
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
	_ = collector.Close()
	require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
	require.True(t, containsError(collector.Errors(), "unknown ID"))
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

		// Parse fragment in context of middle — should see both a: and b: prefixes
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

		// Parse in context of text node — should walk up to <child> then <root>
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
}

func TestBlockXXEExternalDTD(t *testing.T) {
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

func TestEntityBoundaryElementDecl(t *testing.T) {
	t.Parallel()
	// PE starts the element declaration but the closing '>' is in the main DTD.
	// This crosses an entity boundary → parse error (syntax or boundary).
	const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ENTITY % start "<!ELEMENT doc EMPTY">
  %start;>
]>
<doc/>`

	p := helium.NewParser().LoadExternalDTD(true)
	_, err := p.Parse(t.Context(), []byte(input))
	require.Error(t, err, "boundary-violating PE should cause a parse error")
}

func TestEntityBoundaryAttributeListDecl(t *testing.T) {
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
}

func TestEntityBoundaryWellNestedPE(t *testing.T) {
	t.Parallel()
	// PE expands to a complete declaration — no boundary violation.
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

func TestConditionalSectionInclude(t *testing.T) {
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
}

func TestConditionalSectionIgnore(t *testing.T) {
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
}

func TestConditionalSectionInternalSubsetPE(t *testing.T) {
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
}

func TestConditionalSectionErrors(t *testing.T) {
	t.Run("invalid keyword", func(t *testing.T) {
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
}

func TestConditionalSectionExternalDTD(t *testing.T) {
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
	require.NoError(t, d.WriteDoc(&buf, doc))
	require.Equal(t, string(expected), buf.String())
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
