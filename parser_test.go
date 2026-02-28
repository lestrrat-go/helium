package helium

import (
	"bytes"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium/sax"
	"github.com/lestrrat-go/pdebug"
	"github.com/stretchr/testify/require"
)

func TestDetectBOM(t *testing.T) {
	data := map[string][][]byte{
		encUCS4BE:   {{0x00, 0x00, 0x00, 0x3C}},
		encUCS4LE:   {{0x3C, 0x00, 0x00, 0x00}},
		encUCS42143: {{0x00, 0x00, 0x3C, 0x00}},
		encUCS43412: {{0x00, 0x3C, 0x00, 0x00}},
		encEBCDIC:   {{0x4C, 0x6F, 0xA7, 0x94}},
		encUTF8:     {{0x3C, 0x3F, 0x78, 0x6D}, {0xEF, 0xBB, 0xBF}},
		encUTF16LE:  {{0x3C, 0x00, 0x3F, 0x00}, {0xFF, 0xFE}},
		encUTF16BE:  {{0x00, 0x3C, 0x00, 0x3F}, {0xFE, 0xFF}},
		"":          {{0xde, 0xad, 0xbe, 0xef}},
	}

	p := NewParser() // DUMMY
	for expected, inputs := range data {
		for i, input := range inputs {
			ctx := &parserCtx{}
			require.NoError(t, ctx.init(p, bytes.NewReader(input)))
			enc, err := ctx.detectEncoding()
			if expected == "" {
				t.Logf("checking [invalid] (%d)", i)
				require.Error(t, err, "detectEncoding should fail for sequence %#v", input)
			} else {
				t.Logf("checking %s (%d)", expected, i)
				require.NoError(t, err, "detectEncoding should succeed for sequence %#v", input)

				require.Equal(t, expected, enc, "detectEncoding returns as expected")
			}
		}
	}
}

func TestEmptyDocument(t *testing.T) {
	p := NewParser()
	// BOM only
	_, err := p.Parse([]byte{0x00, 0x00, 0x00, 0x3C})
	require.Error(t, err, "Parsing BOM only should fail")
}

func TestParseXMLDecl(t *testing.T) {
	const content = `<root />`
	inputs := map[string]struct {
		version    string
		encoding   string
		standalone int
	}{
		`<?xml version="1.0"?>` + content:                                   {"1.0", "utf8", int(StandaloneImplicitNo)},
		`<?xml version="1.0" encoding="euc-jp"?>` + content:                 {"1.0", "euc-jp", int(StandaloneImplicitNo)},
		`<?xml version="1.0" encoding="cp932" standalone='yes'?>` + content: {"1.0", "cp932", int(StandaloneExplicitYes)},
	}

	for input, expect := range inputs {
		p := NewParser()
		doc, err := p.Parse([]byte(input))
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
		p := NewParser()
		doc, err := p.Parse([]byte(input))
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
			case ProcessingInstructionNode:
				require.IsType(t, &ProcessingInstruction{}, n, "First child should be a processing instruction")

				require.IsType(t, &Element{}, n.NextSibling(), "NextSibling of PI should be Element node")
				break LOOP
			}
			n = n.NextSibling()
		}
	}
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
	p := NewParser()
	_, err := p.Parse([]byte(input))
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
	p := NewParser()
	for _, input := range inputs {
		_, err := p.Parse([]byte(input))
		require.Error(t, err, "Parse should fail for '%s'", input)
	}
}

func TestParseNamespace(t *testing.T) {
	const input = `<?xml version="1.0"?>
<helium:root xmlns:helium="https://github.com/lestrrat-go/helium">
  <helium:child>foo</helium:child>
</helium:root>`
	p := NewParser()
	doc, err := p.Parse([]byte(input))
	require.NoError(t, err, "Parse should succeed for '%s'", input)

	if pdebug.Enabled {
		pdebug.Dump(doc)
	}
}

func findDocumentElement(doc *Document) Node {
	return doc.DocumentElement()
}

func TestParseNoBlanks(t *testing.T) {
	const input = `<?xml version="1.0"?>
<root>
  <child>text</child>
</root>`
	p := NewParser()
	p.SetOption(ParseNoBlanks)
	doc, err := p.Parse([]byte(input))
	require.NoError(t, err, "Parse should succeed")

	// With ParseNoBlanks, blank-only text nodes between elements should be stripped.
	// The root element's first child should be <child>, not a whitespace text node.
	root := findDocumentElement(doc)
	require.NotNil(t, root, "document element must exist")
	first := root.FirstChild()
	require.NotNil(t, first, "root must have children")
	require.Equal(t, ElementNode, first.Type(), "first child should be element, not blank text")
}

func TestParseNoCDATA(t *testing.T) {
	const input = `<?xml version="1.0"?>
<root><![CDATA[hello]]></root>`

	// Without ParseNoCDATA: tree should have a CDATA node
	p1 := NewParser()
	doc1, err := p1.Parse([]byte(input))
	require.NoError(t, err, "Parse should succeed")
	root1 := findDocumentElement(doc1)
	require.NotNil(t, root1)
	child1 := root1.FirstChild()
	require.NotNil(t, child1)
	require.Equal(t, CDATASectionNode, child1.Type(), "without ParseNoCDATA, should be CDATA node")

	// With ParseNoCDATA: CDATA should be delivered as text
	p2 := NewParser()
	p2.SetOption(ParseNoCDATA)
	doc2, err := p2.Parse([]byte(input))
	require.NoError(t, err, "Parse should succeed")
	root2 := findDocumentElement(doc2)
	require.NotNil(t, root2)
	child2 := root2.FirstChild()
	require.NotNil(t, child2)
	require.Equal(t, TextNode, child2.Type(), "with ParseNoCDATA, CDATA should be a text node")
	require.Equal(t, "hello", string(child2.Content()))
}

func TestParsePedantic(t *testing.T) {
	// Pedantic mode requires absolute URIs in namespace declarations
	const input = `<?xml version="1.0"?>
<root xmlns:foo="relative/uri">
  <foo:child>text</foo:child>
</root>`

	// Without pedantic: should succeed
	p1 := NewParser()
	_, err := p1.Parse([]byte(input))
	require.NoError(t, err, "without pedantic, relative URI should be accepted")

	// With pedantic: should fail (relative URI)
	p2 := NewParser()
	p2.SetOption(ParsePedantic)
	_, err = p2.Parse([]byte(input))
	require.Error(t, err, "with pedantic, relative URI should be rejected")
}

func TestParseRecover(t *testing.T) {
	// Malformed XML: mismatched close tag
	const input = `<?xml version="1.0"?>
<root>
  <child>text</chld>
</root>`

	// Without ParseRecover: error, no document
	p1 := NewParser()
	doc1, err := p1.Parse([]byte(input))
	require.Error(t, err, "malformed XML should fail")
	require.Nil(t, doc1, "without recover, no document returned")

	// With ParseRecover: error, but partial document returned
	p2 := NewParser()
	p2.SetOption(ParseRecover)
	doc2, err := p2.Parse([]byte(input))
	require.Error(t, err, "malformed XML should still return error")
	require.NotNil(t, doc2, "with recover, partial document should be returned")
}

func TestParseExternalEntity(t *testing.T) {
	const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ENTITY ext SYSTEM "ext.xml">
]>
<doc>&ext;</doc>`

	// Provide a ResolveEntity handler that returns inline content
	s := sax.New()
	s.ResolveEntityHandler = sax.ResolveEntityFunc(func(_ sax.Context, publicID, systemID string) (sax.ParseInput, error) {
		if systemID == "ext.xml" {
			return newStringParseInput("<inner>hello</inner>", systemID), nil
		}
		return nil, sax.ErrHandlerUnspecified
	})

	p := NewParser()
	p.SetSAXHandler(s)
	p.SetOption(ParseNoEnt)
	_, err := p.Parse([]byte(input))
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

func TestParseDTDValidRequiredAttribute(t *testing.T) {
	// #REQUIRED attribute missing → validation error
	const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc EMPTY>
  <!ATTLIST doc id ID #REQUIRED>
]>
<doc/>`

	p := NewParser()
	p.SetOption(ParseDTDValid)
	doc, err := p.Parse([]byte(input))
	require.Error(t, err, "missing #REQUIRED attribute should fail validation")
	require.NotNil(t, doc, "document should still be returned with validation error")

	ve, ok := err.(*ValidationError)
	require.True(t, ok, "error should be *ValidationError")
	require.True(t, len(ve.Errors) > 0)
	require.Contains(t, ve.Errors[0], "required")
}

func TestParseDTDValidRequiredPresent(t *testing.T) {
	// #REQUIRED attribute present → no error
	const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc EMPTY>
  <!ATTLIST doc id ID #REQUIRED>
]>
<doc id="x1"/>`

	p := NewParser()
	p.SetOption(ParseDTDValid)
	_, err := p.Parse([]byte(input))
	require.NoError(t, err)
}

func TestParseDTDValidFixedMismatch(t *testing.T) {
	// #FIXED attribute with wrong value → validation error
	const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc EMPTY>
  <!ATTLIST doc version CDATA #FIXED "1.0">
]>
<doc version="2.0"/>`

	p := NewParser()
	p.SetOption(ParseDTDValid)
	_, err := p.Parse([]byte(input))
	require.Error(t, err, "#FIXED attribute value mismatch should fail")

	ve, ok := err.(*ValidationError)
	require.True(t, ok)
	require.Contains(t, ve.Error(), "must be")
}

func TestParseDTDValidFixedCorrect(t *testing.T) {
	// #FIXED attribute with correct value → no error
	const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc EMPTY>
  <!ATTLIST doc version CDATA #FIXED "1.0">
]>
<doc version="1.0"/>`

	p := NewParser()
	p.SetOption(ParseDTDValid)
	_, err := p.Parse([]byte(input))
	require.NoError(t, err)
}

func TestParseDTDValidEmptyElement(t *testing.T) {
	// EMPTY element with content → validation error
	const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (child)>
  <!ELEMENT child EMPTY>
]>
<doc><child>text</child></doc>`

	p := NewParser()
	p.SetOption(ParseDTDValid)
	_, err := p.Parse([]byte(input))
	require.Error(t, err, "EMPTY element with content should fail")

	ve, ok := err.(*ValidationError)
	require.True(t, ok)
	require.Contains(t, ve.Error(), "EMPTY")
}

func TestParseDTDValidElementContent(t *testing.T) {
	// Element content model (a, b) with correct content → no error
	const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (a, b)>
  <!ELEMENT a (#PCDATA)>
  <!ELEMENT b (#PCDATA)>
]>
<doc><a>hello</a><b>world</b></doc>`

	p := NewParser()
	p.SetOption(ParseDTDValid)
	_, err := p.Parse([]byte(input))
	require.NoError(t, err)
}

func TestParseDTDValidElementContentMismatch(t *testing.T) {
	// Element content model (a, b) with (b, a) → validation error
	const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (a, b)>
  <!ELEMENT a (#PCDATA)>
  <!ELEMENT b (#PCDATA)>
]>
<doc><b>world</b><a>hello</a></doc>`

	p := NewParser()
	p.SetOption(ParseDTDValid)
	_, err := p.Parse([]byte(input))
	require.Error(t, err, "wrong element order should fail content model")
}

func TestParseDTDValidMixedContent(t *testing.T) {
	// Mixed content (#PCDATA | a)* — text and <a> are allowed
	const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (#PCDATA | a)*>
  <!ELEMENT a (#PCDATA)>
]>
<doc>hello <a>world</a> end</doc>`

	p := NewParser()
	p.SetOption(ParseDTDValid)
	_, err := p.Parse([]byte(input))
	require.NoError(t, err)
}

func TestParseDTDValidMixedContentBadChild(t *testing.T) {
	// Mixed content (#PCDATA | a)* — <b> is NOT allowed
	const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (#PCDATA | a)*>
  <!ELEMENT a (#PCDATA)>
  <!ELEMENT b (#PCDATA)>
]>
<doc>hello <b>world</b></doc>`

	p := NewParser()
	p.SetOption(ParseDTDValid)
	_, err := p.Parse([]byte(input))
	require.Error(t, err, "<b> not allowed in mixed content (a)")
}

func TestParseDTDValidNoFlag(t *testing.T) {
	// Same invalid document but WITHOUT ParseDTDValid → should pass
	const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc EMPTY>
  <!ATTLIST doc id ID #REQUIRED>
]>
<doc/>`

	p := NewParser()
	// Don't set ParseDTDValid
	_, err := p.Parse([]byte(input))
	require.NoError(t, err, "without ParseDTDValid, validation should not run")
}

func TestParseDTDValidChoiceContent(t *testing.T) {
	// Choice content model (a | b) with <a> → valid
	const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (a | b)>
  <!ELEMENT a (#PCDATA)>
  <!ELEMENT b (#PCDATA)>
]>
<doc><a>hello</a></doc>`

	p := NewParser()
	p.SetOption(ParseDTDValid)
	_, err := p.Parse([]byte(input))
	require.NoError(t, err)
}

func TestParseDTDValidRepeatContent(t *testing.T) {
	// Repetition content model (a)+ with multiple <a> → valid
	const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (a)+>
  <!ELEMENT a (#PCDATA)>
]>
<doc><a>1</a><a>2</a><a>3</a></doc>`

	p := NewParser()
	p.SetOption(ParseDTDValid)
	_, err := p.Parse([]byte(input))
	require.NoError(t, err)
}

func TestParseDTDValidRepeatContentEmpty(t *testing.T) {
	// Repetition content model (a)+ with zero <a> → invalid
	const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (a)+>
  <!ELEMENT a (#PCDATA)>
]>
<doc></doc>`

	p := NewParser()
	p.SetOption(ParseDTDValid)
	_, err := p.Parse([]byte(input))
	require.Error(t, err, "(a)+ requires at least one <a>")
}

func TestValidateAttributeValueInternal(t *testing.T) {
	t.Run("ID valid", func(t *testing.T) {
		require.NoError(t, validateAttributeValueInternal(nil, AttrID, "myid"))
	})
	t.Run("ID invalid", func(t *testing.T) {
		require.Error(t, validateAttributeValueInternal(nil, AttrID, "123"))
	})
	t.Run("NMTOKEN valid", func(t *testing.T) {
		require.NoError(t, validateAttributeValueInternal(nil, AttrNmtoken, "hello-world"))
	})
	t.Run("NMTOKEN valid digits", func(t *testing.T) {
		require.NoError(t, validateAttributeValueInternal(nil, AttrNmtoken, "123"))
	})
	t.Run("NMTOKEN invalid", func(t *testing.T) {
		require.Error(t, validateAttributeValueInternal(nil, AttrNmtoken, "hello world"))
	})
	t.Run("NMTOKENS valid", func(t *testing.T) {
		require.NoError(t, validateAttributeValueInternal(nil, AttrNmtokens, "hello world"))
	})
	t.Run("IDREFS valid", func(t *testing.T) {
		require.NoError(t, validateAttributeValueInternal(nil, AttrIDRefs, "id1 id2"))
	})
	t.Run("IDREFS invalid", func(t *testing.T) {
		require.Error(t, validateAttributeValueInternal(nil, AttrIDRefs, "id1 123"))
	})
	t.Run("CDATA anything", func(t *testing.T) {
		require.NoError(t, validateAttributeValueInternal(nil, AttrCDATA, "anything goes here!"))
	})
}

func TestParseDTDValidIDUnique(t *testing.T) {
	const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (item, item)>
  <!ELEMENT item EMPTY>
  <!ATTLIST item id ID #REQUIRED>
]>
<doc><item id="a"/><item id="b"/></doc>`

	p := NewParser()
	p.SetOption(ParseDTDValid)
	_, err := p.Parse([]byte(input))
	require.NoError(t, err)
}

func TestParseDTDValidIDDuplicate(t *testing.T) {
	const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (item, item)>
  <!ELEMENT item EMPTY>
  <!ATTLIST item id ID #REQUIRED>
]>
<doc><item id="a"/><item id="a"/></doc>`

	p := NewParser()
	p.SetOption(ParseDTDValid)
	_, err := p.Parse([]byte(input))
	require.Error(t, err, "duplicate ID should fail")
	require.Contains(t, err.Error(), "duplicate ID")
}

func TestParseDTDValidIDRefValid(t *testing.T) {
	const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (item, ref)>
  <!ELEMENT item EMPTY>
  <!ELEMENT ref EMPTY>
  <!ATTLIST item id ID #REQUIRED>
  <!ATTLIST ref target IDREF #REQUIRED>
]>
<doc><item id="x"/><ref target="x"/></doc>`

	p := NewParser()
	p.SetOption(ParseDTDValid)
	_, err := p.Parse([]byte(input))
	require.NoError(t, err)
}

func TestParseDTDValidIDRefMissing(t *testing.T) {
	const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (item, ref)>
  <!ELEMENT item EMPTY>
  <!ELEMENT ref EMPTY>
  <!ATTLIST item id ID #REQUIRED>
  <!ATTLIST ref target IDREF #REQUIRED>
]>
<doc><item id="x"/><ref target="y"/></doc>`

	p := NewParser()
	p.SetOption(ParseDTDValid)
	_, err := p.Parse([]byte(input))
	require.Error(t, err, "IDREF to missing ID should fail")
	require.Contains(t, err.Error(), "unknown ID")
}

func TestParseDTDValidIDRefsValid(t *testing.T) {
	const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (item, item, refs)>
  <!ELEMENT item EMPTY>
  <!ELEMENT refs EMPTY>
  <!ATTLIST item id ID #REQUIRED>
  <!ATTLIST refs targets IDREFS #REQUIRED>
]>
<doc><item id="a"/><item id="b"/><refs targets="a b"/></doc>`

	p := NewParser()
	p.SetOption(ParseDTDValid)
	_, err := p.Parse([]byte(input))
	require.NoError(t, err)
}

func TestParseDTDValidIDRefsMissing(t *testing.T) {
	const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (item, refs)>
  <!ELEMENT item EMPTY>
  <!ELEMENT refs EMPTY>
  <!ATTLIST item id ID #REQUIRED>
  <!ATTLIST refs targets IDREFS #REQUIRED>
]>
<doc><item id="a"/><refs targets="a z"/></doc>`

	p := NewParser()
	p.SetOption(ParseDTDValid)
	_, err := p.Parse([]byte(input))
	require.Error(t, err, "IDREFS with missing ref should fail")
	require.Contains(t, err.Error(), "unknown ID")
}

func TestParseInNodeContext(t *testing.T) {
	t.Run("basic fragment", func(t *testing.T) {
		doc, err := Parse([]byte(`<root/>`))
		require.NoError(t, err)

		root := doc.DocumentElement()
		require.NotNil(t, root)

		result, err := ParseInNodeContext(root, []byte(`<child/>`))
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, ElementNode, result.Type())
		require.Equal(t, "child", result.Name())
		require.Nil(t, result.Parent())
	})

	t.Run("multiple sibling nodes", func(t *testing.T) {
		doc, err := Parse([]byte(`<root/>`))
		require.NoError(t, err)

		root := doc.DocumentElement()
		result, err := ParseInNodeContext(root, []byte(`<a/><b/>text`))
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, "a", result.Name())

		sib := result.NextSibling()
		require.NotNil(t, sib)
		require.Equal(t, "b", sib.Name())

		text := sib.NextSibling()
		require.NotNil(t, text)
		require.Equal(t, TextNode, text.Type())
		require.Equal(t, "text", string(text.Content()))
	})

	t.Run("namespace inheritance", func(t *testing.T) {
		doc, err := Parse([]byte(`<root xmlns:ns="http://example.com/ns"><child/></root>`))
		require.NoError(t, err)

		root := doc.DocumentElement()
		result, err := ParseInNodeContext(root, []byte(`<ns:item>hello</ns:item>`))
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, ElementNode, result.Type())
		// The element should have been parsed successfully using the inherited ns prefix
		elem, ok := result.(*Element)
		require.True(t, ok)
		require.Equal(t, "ns:item", elem.Name())
	})

	t.Run("nested namespace inheritance", func(t *testing.T) {
		doc, err := Parse([]byte(`<root xmlns:a="http://a.example.com"><middle xmlns:b="http://b.example.com"><child/></middle></root>`))
		require.NoError(t, err)

		root := doc.DocumentElement()
		middle := root.FirstChild()
		require.NotNil(t, middle)

		// Parse fragment in context of middle — should see both a: and b: prefixes
		result, err := ParseInNodeContext(middle, []byte(`<a:x/><b:y/>`))
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, "a:x", result.Name())
		sib := result.NextSibling()
		require.NotNil(t, sib)
		require.Equal(t, "b:y", sib.Name())
	})

	t.Run("fragment with own namespaces", func(t *testing.T) {
		doc, err := Parse([]byte(`<root/>`))
		require.NoError(t, err)

		root := doc.DocumentElement()
		result, err := ParseInNodeContext(root, []byte(`<ns:item xmlns:ns="http://example.com/ns">hello</ns:item>`))
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, "ns:item", result.Name())
	})

	t.Run("document as context", func(t *testing.T) {
		doc, err := Parse([]byte(`<root/>`))
		require.NoError(t, err)

		result, err := ParseInNodeContext(doc, []byte(`<elem/>`))
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, "elem", result.Name())
	})

	t.Run("non-element context walks up", func(t *testing.T) {
		doc, err := Parse([]byte(`<root xmlns:ns="http://example.com/ns"><child>some text</child></root>`))
		require.NoError(t, err)

		root := doc.DocumentElement()
		child := root.FirstChild()
		require.NotNil(t, child)
		textNode := child.FirstChild()
		require.NotNil(t, textNode)
		require.Equal(t, TextNode, textNode.Type())

		// Parse in context of text node — should walk up to <child> then <root>
		result, err := ParseInNodeContext(textNode, []byte(`<ns:item/>`))
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, "ns:item", result.Name())
	})

	t.Run("DTD entity resolution", func(t *testing.T) {
		doc, err := Parse([]byte(`<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ENTITY greeting "hello world">
]>
<doc/>`))
		require.NoError(t, err)

		root := doc.DocumentElement()
		p := NewParser()
		p.SetOption(ParseNoEnt)
		result, err := p.ParseInNodeContext(root, []byte(`<item>&greeting;</item>`))
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, "item", result.Name())
		// The entity should have been resolved
		require.Equal(t, "hello world", string(result.FirstChild().Content()))
	})

	t.Run("empty fragment", func(t *testing.T) {
		doc, err := Parse([]byte(`<root/>`))
		require.NoError(t, err)

		root := doc.DocumentElement()
		result, err := ParseInNodeContext(root, []byte(``))
		require.NoError(t, err)
		require.Nil(t, result)
	})

	t.Run("nil node", func(t *testing.T) {
		_, err := ParseInNodeContext(nil, []byte(`<child/>`))
		require.Error(t, err)
	})

	t.Run("original document preserved", func(t *testing.T) {
		doc, err := Parse([]byte(`<root><existing/></root>`))
		require.NoError(t, err)

		root := doc.DocumentElement()
		_, err = ParseInNodeContext(root, []byte(`<new/>`))
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
