package xmlwriter

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStartDocument(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	require.NoError(t, w.StartDocument("", "", ""))
	require.NoError(t, w.EndDocument())
	require.Equal(t, `<?xml version="1.0"?>`, strings.TrimRight(buf.String(), "\n"))
}

func TestStartDocumentWithEncoding(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	require.NoError(t, w.StartDocument("1.0", "UTF-8", ""))
	require.NoError(t, w.EndDocument())
	require.Equal(t, `<?xml version="1.0" encoding="UTF-8"?>`, strings.TrimRight(buf.String(), "\n"))
}

func TestStartDocumentWithStandalone(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	require.NoError(t, w.StartDocument("1.0", "", "yes"))
	require.NoError(t, w.EndDocument())
	require.Equal(t, `<?xml version="1.0" standalone="yes"?>`, strings.TrimRight(buf.String(), "\n"))
}

func TestStartDocumentFull(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	require.NoError(t, w.StartDocument("1.0", "ISO-8859-1", "no"))
	require.NoError(t, w.EndDocument())
	require.Equal(t, `<?xml version="1.0" encoding="ISO-8859-1" standalone="no"?>`, strings.TrimRight(buf.String(), "\n"))
}

func TestSimpleElement(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	require.NoError(t, w.StartDocument("", "", ""))
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.EndElement())
	require.NoError(t, w.EndDocument())
	require.Equal(t, `<?xml version="1.0"?><root/>`, strings.TrimRight(buf.String(), "\n"))
}

func TestElementWithText(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	require.NoError(t, w.StartDocument("", "", ""))
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.WriteString("hello"))
	require.NoError(t, w.EndElement())
	require.NoError(t, w.EndDocument())
	require.Equal(t, `<?xml version="1.0"?><root>hello</root>`, strings.TrimRight(buf.String(), "\n"))
}

func TestElementWithEscaping(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.WriteString("a < b & c > d"))
	require.NoError(t, w.EndElement())
	require.Equal(t, `<root>a &lt; b &amp; c &gt; d</root>`, buf.String())
}

func TestNestedElements(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	require.NoError(t, w.StartDocument("", "", ""))
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.StartElement("child"))
	require.NoError(t, w.WriteString("text"))
	require.NoError(t, w.EndElement())
	require.NoError(t, w.EndElement())
	require.NoError(t, w.EndDocument())
	require.Equal(t, `<?xml version="1.0"?><root><child>text</child></root>`, strings.TrimRight(buf.String(), "\n"))
}

func TestAttribute(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.WriteAttribute("key", "value"))
	require.NoError(t, w.EndElement())
	require.Equal(t, `<root key="value"/>`, buf.String())
}

func TestAttributeEscaping(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.WriteAttribute("key", `a"b<c&d`))
	require.NoError(t, w.EndElement())
	require.Equal(t, `<root key="a&quot;b&lt;c&amp;d"/>`, buf.String())
}

func TestAttributeWhitespaceEscaping(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.WriteAttribute("key", "a\nb\tc\rd"))
	require.NoError(t, w.EndElement())
	require.Equal(t, `<root key="a&#10;b&#9;c&#13;d"/>`, buf.String())
}

func TestMultipleAttributes(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.WriteAttribute("a", "1"))
	require.NoError(t, w.WriteAttribute("b", "2"))
	require.NoError(t, w.EndElement())
	require.Equal(t, `<root a="1" b="2"/>`, buf.String())
}

func TestAttributeWithContent(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.WriteAttribute("a", "1"))
	require.NoError(t, w.WriteString("text"))
	require.NoError(t, w.EndElement())
	require.Equal(t, `<root a="1">text</root>`, buf.String())
}

func TestSingleQuotes(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf, WithQuoteChar('\''))
	require.NoError(t, w.StartDocument("", "", ""))
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.WriteAttribute("key", "it's"))
	require.NoError(t, w.EndElement())
	require.NoError(t, w.EndDocument())
	require.Equal(t, `<?xml version='1.0'?><root key='it&apos;s'/>`, strings.TrimRight(buf.String(), "\n"))
}

func TestDoubleQuotesInSingleQuoteMode(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf, WithQuoteChar('\''))
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.WriteAttribute("key", `say "hi"`))
	require.NoError(t, w.EndElement())
	require.Equal(t, `<root key='say "hi"'/>`, buf.String())
}

func TestFullEndElement(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.FullEndElement())
	require.Equal(t, `<root></root>`, buf.String())
}

func TestFullEndElementWithAttr(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.WriteAttribute("id", "1"))
	require.NoError(t, w.FullEndElement())
	require.Equal(t, `<root id="1"></root>`, buf.String())
}

func TestWriteElement(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	require.NoError(t, w.WriteElement("item", "hello"))
	require.Equal(t, `<item>hello</item>`, buf.String())
}

func TestComment(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	require.NoError(t, w.StartDocument("", "", ""))
	require.NoError(t, w.WriteComment(" a comment "))
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.EndElement())
	require.NoError(t, w.EndDocument())
	require.Equal(t, `<?xml version="1.0"?><!-- a comment --><root/>`, strings.TrimRight(buf.String(), "\n"))
}

func TestCommentInsideElement(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.WriteComment("inside"))
	require.NoError(t, w.EndElement())
	require.Equal(t, `<root><!--inside--></root>`, buf.String())
}

func TestPI(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	require.NoError(t, w.StartDocument("", "", ""))
	require.NoError(t, w.WritePI("php", `echo "hello";`))
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.EndElement())
	require.NoError(t, w.EndDocument())
	require.Equal(t, `<?xml version="1.0"?><?php echo "hello";?><root/>`, strings.TrimRight(buf.String(), "\n"))
}

func TestPIEmptyContent(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	require.NoError(t, w.WritePI("target", ""))
	require.Equal(t, `<?target?>`, buf.String())
}

func TestPIXmlForbidden(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	err := w.StartPI("xml")
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot be 'xml'")
}

func TestPIXmlCaseForbidden(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	err := w.StartPI("XML")
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot be 'xml'")
}

func TestCDATA(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.WriteCDATA("some <special> & data"))
	require.NoError(t, w.EndElement())
	require.Equal(t, `<root><![CDATA[some <special> & data]]></root>`, buf.String())
}

func TestCDATAInvalidState(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	// CDATA can only be inside an element
	err := w.StartCDATA()
	require.Error(t, err)
}

func TestWriteRaw(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.WriteRaw("<already>&escaped;</already>"))
	require.NoError(t, w.EndElement())
	require.Equal(t, `<root><already>&escaped;</already></root>`, buf.String())
}

func TestNamespaceElement(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	require.NoError(t, w.StartElementNS("ns", "root", "http://example.com"))
	require.NoError(t, w.EndElement())
	require.Equal(t, `<ns:root xmlns:ns="http://example.com"/>`, buf.String())
}

func TestDefaultNamespace(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	require.NoError(t, w.StartElementNS("", "root", "http://example.com"))
	require.NoError(t, w.EndElement())
	require.Equal(t, `<root xmlns="http://example.com"/>`, buf.String())
}

func TestNamespaceAttribute(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.WriteAttributeNS("xlink", "href", "http://www.w3.org/1999/xlink", "http://example.com"))
	require.NoError(t, w.EndElement())
	require.Equal(t, `<root xlink:href="http://example.com" xmlns:xlink="http://www.w3.org/1999/xlink"/>`, buf.String())
}

func TestNamespaceNotRedeclared(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	require.NoError(t, w.StartElementNS("ns", "root", "http://example.com"))
	require.NoError(t, w.StartElementNS("ns", "child", "http://example.com"))
	require.NoError(t, w.EndElement())
	require.NoError(t, w.EndElement())
	require.Equal(t, `<ns:root xmlns:ns="http://example.com"><ns:child/></ns:root>`, buf.String())
}

func TestNamespaceRedeclaredDifferentURI(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	require.NoError(t, w.StartElementNS("ns", "root", "http://example.com/1"))
	require.NoError(t, w.StartElementNS("ns", "child", "http://example.com/2"))
	require.NoError(t, w.EndElement())
	require.NoError(t, w.EndElement())
	require.Equal(t, `<ns:root xmlns:ns="http://example.com/1"><ns:child xmlns:ns="http://example.com/2"/></ns:root>`, buf.String())
}

func TestWriteElementNS(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	require.NoError(t, w.WriteElementNS("ns", "item", "http://example.com", "content"))
	require.Equal(t, `<ns:item xmlns:ns="http://example.com">content</ns:item>`, buf.String())
}

func TestDTD(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	require.NoError(t, w.StartDocument("", "", ""))
	require.NoError(t, w.WriteDTD("html", "-//W3C//DTD XHTML 1.0 Strict//EN", "http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd", ""))
	require.NoError(t, w.StartElement("html"))
	require.NoError(t, w.EndElement())
	require.NoError(t, w.EndDocument())
	expected := `<?xml version="1.0"?><!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Strict//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd"><html/>`
	require.Equal(t, expected, strings.TrimRight(buf.String(), "\n"))
}

func TestDTDSystem(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	require.NoError(t, w.StartDocument("", "", ""))
	require.NoError(t, w.WriteDTD("root", "", "root.dtd", ""))
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.EndElement())
	require.NoError(t, w.EndDocument())
	expected := `<?xml version="1.0"?><!DOCTYPE root SYSTEM "root.dtd"><root/>`
	require.Equal(t, expected, strings.TrimRight(buf.String(), "\n"))
}

func TestDTDWithSubset(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	require.NoError(t, w.StartDocument("", "", ""))
	require.NoError(t, w.WriteDTD("root", "", "", `<!ELEMENT root (#PCDATA)>`))
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.EndElement())
	require.NoError(t, w.EndDocument())
	expected := `<?xml version="1.0"?><!DOCTYPE root [<!ELEMENT root (#PCDATA)>]><root/>`
	require.Equal(t, expected, strings.TrimRight(buf.String(), "\n"))
}

func TestStartDTDWithInternalDecls(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	require.NoError(t, w.StartDocument("", "", ""))
	require.NoError(t, w.StartDTD("root", "", ""))
	require.NoError(t, w.WriteDTDElement("root", "(child)"))
	require.NoError(t, w.WriteDTDElement("child", "(#PCDATA)"))
	require.NoError(t, w.WriteDTDAttlist("root", `id CDATA #IMPLIED`))
	require.NoError(t, w.WriteDTDEntity(false, "copy", "\u00A9"))
	require.NoError(t, w.WriteDTDEntity(true, "content", "stuff"))
	require.NoError(t, w.EndDTD())
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.EndElement())
	require.NoError(t, w.EndDocument())
	expected := "<?xml version=\"1.0\"?>" +
		"<!DOCTYPE root [" +
		"<!ELEMENT root (child)>" +
		"<!ELEMENT child (#PCDATA)>" +
		"<!ATTLIST root id CDATA #IMPLIED>" +
		"<!ENTITY copy \"\u00A9\">" +
		"<!ENTITY % content \"stuff\">" +
		"]><root/>"
	require.Equal(t, expected, strings.TrimRight(buf.String(), "\n"))
}

func TestDTDExternalEntity(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	require.NoError(t, w.StartDocument("", "", ""))
	require.NoError(t, w.StartDTD("root", "", ""))
	require.NoError(t, w.WriteDTDExternalEntity(false, "logo", "", "logo.gif", "gif"))
	require.NoError(t, w.EndDTD())
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.EndElement())
	require.NoError(t, w.EndDocument())
	expected := `<?xml version="1.0"?><!DOCTYPE root [<!ENTITY logo SYSTEM "logo.gif" NDATA gif>]><root/>`
	require.Equal(t, expected, strings.TrimRight(buf.String(), "\n"))
}

func TestDTDNotation(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	require.NoError(t, w.StartDocument("", "", ""))
	require.NoError(t, w.StartDTD("root", "", ""))
	require.NoError(t, w.WriteDTDNotation("gif", "", "image/gif"))
	require.NoError(t, w.EndDTD())
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.EndElement())
	require.NoError(t, w.EndDocument())
	expected := `<?xml version="1.0"?><!DOCTYPE root [<!NOTATION gif SYSTEM "image/gif">]><root/>`
	require.Equal(t, expected, strings.TrimRight(buf.String(), "\n"))
}

func TestIndentation(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf, WithIndent("  "))
	require.NoError(t, w.StartDocument("", "", ""))
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.StartElement("child"))
	require.NoError(t, w.WriteString("text"))
	require.NoError(t, w.EndElement())
	require.NoError(t, w.StartElement("child"))
	require.NoError(t, w.EndElement())
	require.NoError(t, w.EndElement())
	require.NoError(t, w.EndDocument())
	expected := "<?xml version=\"1.0\"?>\n<root>\n  <child>text</child>\n  <child/>\n</root>"
	require.Equal(t, expected, buf.String())
}

func TestIndentationDeepNesting(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf, WithIndent("\t"))
	require.NoError(t, w.StartElement("a"))
	require.NoError(t, w.StartElement("b"))
	require.NoError(t, w.StartElement("c"))
	require.NoError(t, w.EndElement())
	require.NoError(t, w.EndElement())
	require.NoError(t, w.EndElement())
	expected := "<a>\n\t<b>\n\t\t<c/>\n\t</b>\n</a>"
	require.Equal(t, expected, buf.String())
}

func TestIndentationMixedContent(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf, WithIndent("  "))
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.WriteString("text"))
	require.NoError(t, w.StartElement("child"))
	require.NoError(t, w.EndElement())
	require.NoError(t, w.EndElement())
	// When text and elements are mixed, the end tag should not be indented
	require.Equal(t, "<root>text\n  <child/></root>", buf.String())
}

func TestEndDocumentClosesAll(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	require.NoError(t, w.StartDocument("", "", ""))
	require.NoError(t, w.StartElement("a"))
	require.NoError(t, w.StartElement("b"))
	require.NoError(t, w.StartElement("c"))
	require.NoError(t, w.EndDocument())
	require.Equal(t, `<?xml version="1.0"?><a><b><c/></b></a>`, strings.TrimRight(buf.String(), "\n"))
}

func TestStateValidation(t *testing.T) {
	t.Run("StartDocumentTwice", func(t *testing.T) {
		var buf bytes.Buffer
		w := New(&buf)
		require.NoError(t, w.StartDocument("", "", ""))
		err := w.StartDocument("", "", "")
		require.Error(t, err)
	})

	t.Run("AttributeOutsideElement", func(t *testing.T) {
		var buf bytes.Buffer
		w := New(&buf)
		err := w.StartAttribute("name")
		require.Error(t, err)
	})

	t.Run("EndElementWithoutStart", func(t *testing.T) {
		var buf bytes.Buffer
		w := New(&buf)
		err := w.EndElement()
		require.Error(t, err)
	})

	t.Run("EndCommentWithoutStart", func(t *testing.T) {
		var buf bytes.Buffer
		w := New(&buf)
		err := w.EndComment()
		require.Error(t, err)
	})

	t.Run("EndPIWithoutStart", func(t *testing.T) {
		var buf bytes.Buffer
		w := New(&buf)
		err := w.EndPI()
		require.Error(t, err)
	})

	t.Run("EndCDATAWithoutStart", func(t *testing.T) {
		var buf bytes.Buffer
		w := New(&buf)
		err := w.EndCDATA()
		require.Error(t, err)
	})

	t.Run("EndDTDWithoutStart", func(t *testing.T) {
		var buf bytes.Buffer
		w := New(&buf)
		err := w.EndDTD()
		require.Error(t, err)
	})

	t.Run("WriteStringInNone", func(t *testing.T) {
		var buf bytes.Buffer
		w := New(&buf)
		err := w.WriteString("text")
		require.Error(t, err)
	})
}

func TestStickyError(t *testing.T) {
	// Use a writer that fails after N bytes
	fw := &failWriter{failAfter: 5}
	w := New(fw)
	_ = w.StartDocument("", "", "")
	// After the error, all subsequent calls should return the same error
	require.Error(t, w.err)
	err := w.StartElement("root")
	require.Error(t, err)
}

func TestFlush(t *testing.T) {
	fb := &flushableBuffer{}
	w := New(fb)
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.EndElement())
	require.NoError(t, w.Flush())
	require.True(t, fb.flushed)
}

func TestStartAttributeViaStartEnd(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.StartAttribute("key"))
	require.NoError(t, w.WriteString("val"))
	require.NoError(t, w.EndAttribute())
	require.NoError(t, w.EndElement())
	require.Equal(t, `<root key="val"/>`, buf.String())
}

func TestStartAttributeMultiPart(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.StartAttribute("key"))
	require.NoError(t, w.WriteString("part1"))
	require.NoError(t, w.WriteString("part2"))
	require.NoError(t, w.EndAttribute())
	require.NoError(t, w.EndElement())
	require.Equal(t, `<root key="part1part2"/>`, buf.String())
}

func TestEndElementClosesAttribute(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.StartAttribute("key"))
	require.NoError(t, w.WriteString("val"))
	// EndElement should auto-close the attribute
	require.NoError(t, w.EndElement())
	require.Equal(t, `<root key="val"/>`, buf.String())
}

func TestStartCommentViaStartEnd(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	require.NoError(t, w.StartDocument("", "", ""))
	require.NoError(t, w.StartComment())
	require.NoError(t, w.WriteString(" hello "))
	require.NoError(t, w.EndComment())
	require.NoError(t, w.EndDocument())
	require.Equal(t, `<?xml version="1.0"?><!-- hello -->`, strings.TrimRight(buf.String(), "\n"))
}

func TestStartPIViaStartEnd(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	require.NoError(t, w.StartDocument("", "", ""))
	require.NoError(t, w.StartPI("target"))
	require.NoError(t, w.WriteString(" data"))
	require.NoError(t, w.EndPI())
	require.NoError(t, w.EndDocument())
	require.Equal(t, `<?xml version="1.0"?><?target data?>`, strings.TrimRight(buf.String(), "\n"))
}

func TestCompleteDocument(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	require.NoError(t, w.StartDocument("1.0", "UTF-8", ""))
	require.NoError(t, w.WriteComment(" Generated document "))
	require.NoError(t, w.StartElement("catalog"))
	require.NoError(t, w.WriteAttribute("xmlns", "http://example.com/catalog"))

	require.NoError(t, w.StartElement("book"))
	require.NoError(t, w.WriteAttribute("id", "bk101"))
	require.NoError(t, w.WriteElement("title", "XML Developer's Guide"))
	require.NoError(t, w.WriteElement("price", "44.95"))
	require.NoError(t, w.EndElement())

	require.NoError(t, w.StartElement("book"))
	require.NoError(t, w.WriteAttribute("id", "bk102"))
	require.NoError(t, w.WriteElement("title", "Midnight Rain"))
	require.NoError(t, w.WriteElement("price", "5.95"))
	require.NoError(t, w.EndElement())

	require.NoError(t, w.EndElement())
	require.NoError(t, w.EndDocument())

	expected := `<?xml version="1.0" encoding="UTF-8"?>` +
		`<!-- Generated document -->` +
		`<catalog xmlns="http://example.com/catalog">` +
		`<book id="bk101"><title>XML Developer's Guide</title><price>44.95</price></book>` +
		`<book id="bk102"><title>Midnight Rain</title><price>5.95</price></book>` +
		`</catalog>`
	require.Equal(t, expected, strings.TrimRight(buf.String(), "\n"))
}

func TestCompleteDocumentIndented(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf, WithIndent("  "))
	require.NoError(t, w.StartDocument("1.0", "UTF-8", ""))
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.WriteElement("name", "John"))
	require.NoError(t, w.WriteElement("age", "30"))
	require.NoError(t, w.EndElement())
	require.NoError(t, w.EndDocument())

	expected := "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n" +
		"<root>\n" +
		"  <name>John</name>\n" +
		"  <age>30</age>\n" +
		"</root>"
	require.Equal(t, expected, buf.String())
}

func TestWithoutStartDocument(t *testing.T) {
	// Writer should work without StartDocument for fragment output
	var buf bytes.Buffer
	w := New(&buf)
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.WriteString("hello"))
	require.NoError(t, w.EndElement())
	require.Equal(t, `<root>hello</root>`, buf.String())
}

func TestWriteRawInDocument(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	require.NoError(t, w.StartDocument("", "", ""))
	require.NoError(t, w.WriteRaw("\n"))
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.EndElement())
	require.NoError(t, w.EndDocument())
	require.Equal(t, "<?xml version=\"1.0\"?>\n<root/>", strings.TrimRight(buf.String(), "\n"))
}

func TestNamespaceElementWithAttribute(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	require.NoError(t, w.StartElementNS("soap", "Envelope", "http://schemas.xmlsoap.org/soap/envelope/"))
	require.NoError(t, w.WriteAttributeNS("soap", "encodingStyle", "http://schemas.xmlsoap.org/soap/envelope/", "http://schemas.xmlsoap.org/soap/encoding/"))
	require.NoError(t, w.StartElementNS("soap", "Body", "http://schemas.xmlsoap.org/soap/envelope/"))
	require.NoError(t, w.EndElement())
	require.NoError(t, w.EndElement())
	// The xmlns:soap should only appear once (on Envelope), not on Body or attribute
	expected := `<soap:Envelope soap:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/" xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"><soap:Body/></soap:Envelope>`
	require.Equal(t, expected, buf.String())
}

func TestEmptyElementSibling(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.StartElement("a"))
	require.NoError(t, w.EndElement())
	require.NoError(t, w.StartElement("b"))
	require.NoError(t, w.EndElement())
	require.NoError(t, w.EndElement())
	require.Equal(t, `<root><a/><b/></root>`, buf.String())
}

func TestWriteAttributeNS(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf)
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.WriteAttributeNS("xml", "lang", "", "en"))
	require.NoError(t, w.EndElement())
	// xml prefix should not generate xmlns declaration (empty URI)
	require.Equal(t, `<root xml:lang="en"/>`, buf.String())
}

// -- helpers --

type failWriter struct {
	written   int
	failAfter int
}

func (fw *failWriter) Write(p []byte) (int, error) {
	if fw.written+len(p) > fw.failAfter {
		remaining := fw.failAfter - fw.written
		if remaining <= 0 {
			return 0, errors.New("write failed")
		}
		fw.written += remaining
		return remaining, errors.New("write failed")
	}
	fw.written += len(p)
	return len(p), nil
}

type flushableBuffer struct {
	bytes.Buffer
	flushed bool
}

func (fb *flushableBuffer) Flush() error {
	fb.flushed = true
	return nil
}
