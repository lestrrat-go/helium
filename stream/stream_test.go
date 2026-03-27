package stream_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium/stream"
	"github.com/stretchr/testify/require"
)

type countingWriter struct {
	buf    bytes.Buffer
	writes int
}

func (w *countingWriter) Write(p []byte) (int, error) {
	w.writes++
	return w.buf.Write(p)
}

func (w *countingWriter) WriteByte(b byte) error {
	w.writes++
	return w.buf.WriteByte(b)
}

func (w *countingWriter) WriteString(s string) (int, error) {
	w.writes++
	return w.buf.WriteString(s)
}

func TestStartDocument(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartDocument("", "", ""))
	require.NoError(t, w.EndDocument())
	require.Equal(t, `<?xml version="1.0"?>`, strings.TrimRight(buf.String(), "\n"))
}

func TestStartDocumentZeroValueWriterReturnsError(t *testing.T) {
	t.Parallel()
	var w stream.Writer
	require.NotPanics(t, func() {
		err := w.StartDocument("", "", "")
		require.ErrorContains(t, err, "output writer is nil")
	})
}

func TestStartDocumentWithEncoding(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartDocument("1.0", "UTF-8", ""))
	require.NoError(t, w.EndDocument())
	require.Equal(t, `<?xml version="1.0" encoding="UTF-8"?>`, strings.TrimRight(buf.String(), "\n"))
}

func TestStartDocumentWithStandalone(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartDocument("1.0", "", "yes"))
	require.NoError(t, w.EndDocument())
	require.Equal(t, `<?xml version="1.0" standalone="yes"?>`, strings.TrimRight(buf.String(), "\n"))
}

func TestStartDocumentFull(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartDocument("1.0", "ISO-8859-1", "no"))
	require.NoError(t, w.EndDocument())
	require.Equal(t, `<?xml version="1.0" encoding="ISO-8859-1" standalone="no"?>`, strings.TrimRight(buf.String(), "\n"))
}

func TestStartDocumentEncodingValidation(t *testing.T) {
	t.Parallel()
	t.Run("valid encoding", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartDocument("1.0", "UTF-8", ""))
	})

	t.Run("valid encoding case insensitive", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartDocument("1.0", "utf-8", ""))
	})

	t.Run("invalid encoding", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		err := w.StartDocument("1.0", "BOGUS-999", "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "unsupported encoding")
	})

	t.Run("empty encoding skips validation", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartDocument("1.0", "", ""))
	})
}

func TestSimpleElement(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartDocument("", "", ""))
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.EndElement())
	require.NoError(t, w.EndDocument())
	require.Equal(t, "<?xml version=\"1.0\"?>\n<root/>", strings.TrimRight(buf.String(), "\n"))
}

func TestElementWithText(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartDocument("", "", ""))
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.WriteString("hello"))
	require.NoError(t, w.EndElement())
	require.NoError(t, w.EndDocument())
	require.Equal(t, "<?xml version=\"1.0\"?>\n<root>hello</root>", strings.TrimRight(buf.String(), "\n"))
}

func TestElementWithEscaping(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.WriteString("a < b & c > d"))
	require.NoError(t, w.EndElement())
	require.Equal(t, `<root>a &lt; b &amp; c &gt; d</root>`, buf.String())
}

func TestWriteStringBatchesPlainText(t *testing.T) {
	t.Parallel()
	out := &countingWriter{}
	w := stream.NewWriter(out)

	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.WriteString(strings.Repeat("a", 64)))
	require.NoError(t, w.EndElement())

	require.Equal(t, 7, out.writes)
}

func TestNestedElements(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartDocument("", "", ""))
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.StartElement("child"))
	require.NoError(t, w.WriteString("text"))
	require.NoError(t, w.EndElement())
	require.NoError(t, w.EndElement())
	require.NoError(t, w.EndDocument())
	require.Equal(t, "<?xml version=\"1.0\"?>\n<root><child>text</child></root>", strings.TrimRight(buf.String(), "\n"))
}

func TestAttribute(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.WriteAttribute("key", "value"))
	require.NoError(t, w.EndElement())
	require.Equal(t, `<root key="value"/>`, buf.String())
}

func TestAttributeEscaping(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.WriteAttribute("key", `a"b<c&d`))
	require.NoError(t, w.EndElement())
	require.Equal(t, `<root key="a&quot;b&lt;c&amp;d"/>`, buf.String())
}

func TestAttributeWhitespaceEscaping(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.WriteAttribute("key", "a\nb\tc\rd"))
	require.NoError(t, w.EndElement())
	require.Equal(t, `<root key="a&#10;b&#9;c&#13;d"/>`, buf.String())
}

func TestMultipleAttributes(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.WriteAttribute("a", "1"))
	require.NoError(t, w.WriteAttribute("b", "2"))
	require.NoError(t, w.EndElement())
	require.Equal(t, `<root a="1" b="2"/>`, buf.String())
}

func TestAttributeWithContent(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.WriteAttribute("a", "1"))
	require.NoError(t, w.WriteString("text"))
	require.NoError(t, w.EndElement())
	require.Equal(t, `<root a="1">text</root>`, buf.String())
}

func TestSingleQuotes(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf).QuoteChar('\'')
	require.NoError(t, w.StartDocument("", "", ""))
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.WriteAttribute("key", "it's"))
	require.NoError(t, w.EndElement())
	require.NoError(t, w.EndDocument())
	require.Equal(t, "<?xml version='1.0'?>\n<root key='it&apos;s'/>", strings.TrimRight(buf.String(), "\n"))
}

func TestDoubleQuotesInSingleQuoteMode(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf).QuoteChar('\'')
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.WriteAttribute("key", `say "hi"`))
	require.NoError(t, w.EndElement())
	require.Equal(t, `<root key='say "hi"'/>`, buf.String())
}

func TestFullEndElement(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.FullEndElement())
	require.Equal(t, `<root></root>`, buf.String())
}

func TestFullEndElementWithAttr(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.WriteAttribute("id", "1"))
	require.NoError(t, w.FullEndElement())
	require.Equal(t, `<root id="1"></root>`, buf.String())
}

func TestWriteElement(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.WriteElement("item", "hello"))
	require.Equal(t, `<item>hello</item>`, buf.String())
}

func TestComment(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartDocument("", "", ""))
	require.NoError(t, w.WriteComment(" a comment "))
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.EndElement())
	require.NoError(t, w.EndDocument())
	require.Equal(t, "<?xml version=\"1.0\"?>\n<!-- a comment --><root/>", strings.TrimRight(buf.String(), "\n"))
}

func TestCommentInsideElement(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.WriteComment("inside"))
	require.NoError(t, w.EndElement())
	require.Equal(t, `<root><!--inside--></root>`, buf.String())
}

func TestPI(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartDocument("", "", ""))
	require.NoError(t, w.WritePI("php", `echo "hello";`))
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.EndElement())
	require.NoError(t, w.EndDocument())
	require.Equal(t, "<?xml version=\"1.0\"?>\n<?php echo \"hello\";?><root/>", strings.TrimRight(buf.String(), "\n"))
}

func TestPIEmptyContent(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.WritePI("target", ""))
	require.Equal(t, `<?target?>`, buf.String())
}

func TestPIXmlForbidden(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	err := w.StartPI("xml")
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot be 'xml'")
}

func TestPIXmlCaseForbidden(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	err := w.StartPI("XML")
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot be 'xml'")
}

func TestCDATA(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.WriteCDATA("some <special> & data"))
	require.NoError(t, w.EndElement())
	require.Equal(t, `<root><![CDATA[some <special> & data]]></root>`, buf.String())
}

func TestCDATAInvalidState(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	// CDATA can only be inside an element
	err := w.StartCDATA()
	require.Error(t, err)
}

func TestWriteRaw(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.WriteRaw("<already>&escaped;</already>"))
	require.NoError(t, w.EndElement())
	require.Equal(t, `<root><already>&escaped;</already></root>`, buf.String())
}

func TestNamespaceElement(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartElementNS("ns", "root", "http://example.com"))
	require.NoError(t, w.EndElement())
	require.Equal(t, `<ns:root xmlns:ns="http://example.com"/>`, buf.String())
}

func TestDefaultNamespace(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartElementNS("", "root", "http://example.com"))
	require.NoError(t, w.EndElement())
	require.Equal(t, `<root xmlns="http://example.com"/>`, buf.String())
}

func TestNamespaceAttribute(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.WriteAttributeNS("xlink", "href", "http://www.w3.org/1999/xlink", "http://example.com"))
	require.NoError(t, w.EndElement())
	require.Equal(t, `<root xlink:href="http://example.com" xmlns:xlink="http://www.w3.org/1999/xlink"/>`, buf.String())
}

func TestNamespaceNotRedeclared(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartElementNS("ns", "root", "http://example.com"))
	require.NoError(t, w.StartElementNS("ns", "child", "http://example.com"))
	require.NoError(t, w.EndElement())
	require.NoError(t, w.EndElement())
	require.Equal(t, `<ns:root xmlns:ns="http://example.com"><ns:child/></ns:root>`, buf.String())
}

func TestNamespaceRedeclaredDifferentURI(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartElementNS("ns", "root", "http://example.com/1"))
	require.NoError(t, w.StartElementNS("ns", "child", "http://example.com/2"))
	require.NoError(t, w.EndElement())
	require.NoError(t, w.EndElement())
	require.Equal(t, `<ns:root xmlns:ns="http://example.com/1"><ns:child xmlns:ns="http://example.com/2"/></ns:root>`, buf.String())
}

func TestWriteElementNS(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.WriteElementNS("ns", "item", "http://example.com", "content"))
	require.Equal(t, `<ns:item xmlns:ns="http://example.com">content</ns:item>`, buf.String())
}

func TestDTD(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartDocument("", "", ""))
	require.NoError(t, w.WriteDTD("html", "-//W3C//DTD XHTML 1.0 Strict//EN", "http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd", ""))
	require.NoError(t, w.StartElement("html"))
	require.NoError(t, w.EndElement())
	require.NoError(t, w.EndDocument())
	expected := "<?xml version=\"1.0\"?>\n<!DOCTYPE html PUBLIC \"-//W3C//DTD XHTML 1.0 Strict//EN\" \"http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd\"><html/>"
	require.Equal(t, expected, strings.TrimRight(buf.String(), "\n"))
}

func TestDTDSystem(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartDocument("", "", ""))
	require.NoError(t, w.WriteDTD("root", "", "root.dtd", ""))
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.EndElement())
	require.NoError(t, w.EndDocument())
	expected := "<?xml version=\"1.0\"?>\n<!DOCTYPE root SYSTEM \"root.dtd\"><root/>"
	require.Equal(t, expected, strings.TrimRight(buf.String(), "\n"))
}

func TestDTDWithSubset(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartDocument("", "", ""))
	require.NoError(t, w.WriteDTD("root", "", "", `<!ELEMENT root (#PCDATA)>`))
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.EndElement())
	require.NoError(t, w.EndDocument())
	expected := "<?xml version=\"1.0\"?>\n<!DOCTYPE root [<!ELEMENT root (#PCDATA)>]><root/>"
	require.Equal(t, expected, strings.TrimRight(buf.String(), "\n"))
}

func TestStartDTDPubidRequiresSysid(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartDocument("", "", ""))
	err := w.StartDTD("root", "-//Example//DTD Test//EN", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "requires sysid when pubid is provided")
}

func TestStartDTDWithInternalDecls(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
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
	expected := "<?xml version=\"1.0\"?>\n" +
		"<!DOCTYPE root [" +
		"<!ELEMENT root (child)>" +
		"<!ELEMENT child (#PCDATA)>" +
		"<!ATTLIST root id CDATA #IMPLIED>" +
		"<!ENTITY copy \"\u00A9\">" +
		"<!ENTITY % content \"stuff\">" +
		"]><root/>"
	require.Equal(t, expected, strings.TrimRight(buf.String(), "\n"))
}

func TestWriteDTDEntityEscaping(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartDocument("", "", ""))
	require.NoError(t, w.StartDTD("root", "", ""))
	require.NoError(t, w.WriteDTDEntity(false, "ent", `he said "hello" & <goodbye>`))
	require.NoError(t, w.EndDTD())
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.EndElement())
	require.NoError(t, w.EndDocument())
	expected := "<?xml version=\"1.0\"?>\n<!DOCTYPE root [<!ENTITY ent \"he said &quot;hello&quot; &amp; &lt;goodbye&gt;\">]><root/>"
	require.Equal(t, expected, strings.TrimRight(buf.String(), "\n"))
}

func TestDTDExternalEntity(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartDocument("", "", ""))
	require.NoError(t, w.StartDTD("root", "", ""))
	require.NoError(t, w.WriteDTDExternalEntity(false, "logo", "", "logo.gif", "gif"))
	require.NoError(t, w.EndDTD())
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.EndElement())
	require.NoError(t, w.EndDocument())
	expected := "<?xml version=\"1.0\"?>\n<!DOCTYPE root [<!ENTITY logo SYSTEM \"logo.gif\" NDATA gif>]><root/>"
	require.Equal(t, expected, strings.TrimRight(buf.String(), "\n"))
}

func TestDTDNotation(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartDocument("", "", ""))
	require.NoError(t, w.StartDTD("root", "", ""))
	require.NoError(t, w.WriteDTDNotation("gif", "", "image/gif"))
	require.NoError(t, w.EndDTD())
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.EndElement())
	require.NoError(t, w.EndDocument())
	expected := "<?xml version=\"1.0\"?>\n<!DOCTYPE root [<!NOTATION gif SYSTEM \"image/gif\">]><root/>"
	require.Equal(t, expected, strings.TrimRight(buf.String(), "\n"))
}

func TestDTDIndentPublicSystem(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf).Indent("  ")
	require.NoError(t, w.StartDocument("", "", ""))
	require.NoError(t, w.WriteDTD("html", "-//W3C//DTD XHTML 1.0//EN", "xhtml1.dtd", ""))
	require.NoError(t, w.StartElement("html"))
	require.NoError(t, w.EndElement())
	require.NoError(t, w.EndDocument())
	expected := "<?xml version=\"1.0\"?>\n\n" +
		"<!DOCTYPE html\nPUBLIC \"-//W3C//DTD XHTML 1.0//EN\"\n       \"xhtml1.dtd\">\n<html/>"
	require.Equal(t, expected, buf.String())
}

func TestDTDIndentSystemOnly(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf).Indent("  ")
	require.NoError(t, w.StartDocument("", "", ""))
	require.NoError(t, w.WriteDTD("root", "", "root.dtd", ""))
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.EndElement())
	require.NoError(t, w.EndDocument())
	expected := "<?xml version=\"1.0\"?>\n\n" +
		"<!DOCTYPE root\nSYSTEM \"root.dtd\">\n<root/>"
	require.Equal(t, expected, buf.String())
}

func TestIndentation(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf).Indent("  ")
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
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf).Indent("\t")
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
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf).Indent("  ")
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.WriteString("text"))
	require.NoError(t, w.StartElement("child"))
	require.NoError(t, w.EndElement())
	require.NoError(t, w.EndElement())
	// When text and elements are mixed, the end tag should not be indented
	require.Equal(t, "<root>text\n  <child/></root>", buf.String())
}

func TestIndentCommentNewline(t *testing.T) {
	t.Parallel()
	t.Run("comment followed by element", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf).Indent("  ")
		require.NoError(t, w.StartElement("root"))
		require.NoError(t, w.WriteComment(" hello "))
		require.NoError(t, w.StartElement("child"))
		require.NoError(t, w.EndElement())
		require.NoError(t, w.EndElement())
		// Trailing \n after --> prevents double newline with next element's indent
		expected := "<root><!-- hello -->\n  <child/>\n</root>"
		require.Equal(t, expected, buf.String())
	})

	t.Run("comment as last child", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf).Indent("  ")
		require.NoError(t, w.StartElement("root"))
		require.NoError(t, w.WriteComment(" hello "))
		require.NoError(t, w.EndElement())
		// Trailing \n after -->, then closing tag indented
		expected := "<root><!-- hello -->\n</root>"
		require.Equal(t, expected, buf.String())
	})

	t.Run("comment at document level", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf).Indent("  ")
		require.NoError(t, w.StartDocument("", "", ""))
		require.NoError(t, w.WriteComment(" hello "))
		require.NoError(t, w.StartElement("root"))
		require.NoError(t, w.EndElement())
		require.NoError(t, w.EndDocument())
		// Trailing \n after --> separates comment from root element
		expected := "<?xml version=\"1.0\"?>\n<!-- hello -->\n<root/>"
		require.Equal(t, expected, buf.String())
	})

	t.Run("no indent no trailing newline", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartElement("root"))
		require.NoError(t, w.WriteComment(" hello "))
		require.NoError(t, w.EndElement())
		expected := "<root><!-- hello --></root>"
		require.Equal(t, expected, buf.String())
	})
}

func TestIndentPINewline(t *testing.T) {
	t.Parallel()
	t.Run("PI followed by element", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf).Indent("  ")
		require.NoError(t, w.StartElement("root"))
		require.NoError(t, w.WritePI("app", "data"))
		require.NoError(t, w.StartElement("child"))
		require.NoError(t, w.EndElement())
		require.NoError(t, w.EndElement())
		expected := "<root><?app data?>\n  <child/>\n</root>"
		require.Equal(t, expected, buf.String())
	})

	t.Run("PI at document level", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf).Indent("  ")
		require.NoError(t, w.StartDocument("", "", ""))
		require.NoError(t, w.WritePI("app", "data"))
		require.NoError(t, w.StartElement("root"))
		require.NoError(t, w.EndElement())
		require.NoError(t, w.EndDocument())
		expected := "<?xml version=\"1.0\"?>\n<?app data?>\n<root/>"
		require.Equal(t, expected, buf.String())
	})

	t.Run("no indent no trailing newline", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartElement("root"))
		require.NoError(t, w.WritePI("app", "data"))
		require.NoError(t, w.EndElement())
		expected := "<root><?app data?></root>"
		require.Equal(t, expected, buf.String())
	})
}

func TestEndDocumentClosesAll(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartDocument("", "", ""))
	require.NoError(t, w.StartElement("a"))
	require.NoError(t, w.StartElement("b"))
	require.NoError(t, w.StartElement("c"))
	require.NoError(t, w.EndDocument())
	require.Equal(t, "<?xml version=\"1.0\"?>\n<a><b><c/></b></a>", strings.TrimRight(buf.String(), "\n"))
}

func TestEndDocumentClosesOpenPI(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartDocument("", "", ""))
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.StartPI("test"))
	require.NoError(t, w.WriteString(" data"))
	require.NoError(t, w.EndDocument())
	require.Equal(t, "<?xml version=\"1.0\"?>\n<root><?test data?></root>", strings.TrimRight(buf.String(), "\n"))
}

func TestEndDocumentClosesOpenCDATA(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartDocument("", "", ""))
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.StartCDATA())
	require.NoError(t, w.WriteString("data"))
	require.NoError(t, w.EndDocument())
	require.Equal(t, "<?xml version=\"1.0\"?>\n<root><![CDATA[data]]></root>", strings.TrimRight(buf.String(), "\n"))
}

func TestEndDocumentClosesOpenComment(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartDocument("", "", ""))
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.StartComment())
	require.NoError(t, w.WriteString("note"))
	require.NoError(t, w.EndDocument())
	require.Equal(t, "<?xml version=\"1.0\"?>\n<root><!--note--></root>", strings.TrimRight(buf.String(), "\n"))
}

func TestEndDocumentClosesOpenDTD(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartDocument("", "", ""))
	require.NoError(t, w.StartDTD("root", "", ""))
	require.NoError(t, w.EndDocument())
	require.Equal(t, "<?xml version=\"1.0\"?>\n<!DOCTYPE root>", strings.TrimRight(buf.String(), "\n"))
}

func TestStateValidation(t *testing.T) {
	t.Parallel()
	t.Run("StartDocumentTwice", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartDocument("", "", ""))
		err := w.StartDocument("", "", "")
		require.Error(t, err)
	})

	t.Run("AttributeOutsideElement", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		err := w.StartAttribute("name")
		require.Error(t, err)
	})

	t.Run("EndElementWithoutStart", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		err := w.EndElement()
		require.Error(t, err)
	})

	t.Run("EndCommentWithoutStart", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		err := w.EndComment()
		require.Error(t, err)
	})

	t.Run("EndPIWithoutStart", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		err := w.EndPI()
		require.Error(t, err)
	})

	t.Run("EndCDATAWithoutStart", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		err := w.EndCDATA()
		require.Error(t, err)
	})

	t.Run("EndDTDWithoutStart", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		err := w.EndDTD()
		require.Error(t, err)
	})

	t.Run("WriteStringInNone", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		err := w.WriteString("text")
		require.NoError(t, err) // stateNone allows fragment writing
	})
}

func TestStickyError(t *testing.T) {
	t.Parallel()
	// Use a writer that fails after N bytes
	fw := &failWriter{failAfter: 5}
	w := stream.NewWriter(fw)
	_ = w.StartDocument("", "", "")
	// After the error, subsequent calls should continue returning an error.
	require.Error(t, w.Flush())
	err := w.StartElement("root")
	require.Error(t, err)
}

func TestFlush(t *testing.T) {
	t.Parallel()
	fb := &flushableBuffer{}
	w := stream.NewWriter(fb)
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.EndElement())
	require.NoError(t, w.Flush())
	require.True(t, fb.flushed)
}

func TestStartAttributeViaStartEnd(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.StartAttribute("key"))
	require.NoError(t, w.WriteString("val"))
	require.NoError(t, w.EndAttribute())
	require.NoError(t, w.EndElement())
	require.Equal(t, `<root key="val"/>`, buf.String())
}

func TestStartAttributeMultiPart(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.StartAttribute("key"))
	require.NoError(t, w.WriteString("part1"))
	require.NoError(t, w.WriteString("part2"))
	require.NoError(t, w.EndAttribute())
	require.NoError(t, w.EndElement())
	require.Equal(t, `<root key="part1part2"/>`, buf.String())
}

func TestEndElementClosesAttribute(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.StartAttribute("key"))
	require.NoError(t, w.WriteString("val"))
	// EndElement should auto-close the attribute
	require.NoError(t, w.EndElement())
	require.Equal(t, `<root key="val"/>`, buf.String())
}

func TestStartCommentViaStartEnd(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartDocument("", "", ""))
	require.NoError(t, w.StartComment())
	require.NoError(t, w.WriteString(" hello "))
	require.NoError(t, w.EndComment())
	require.NoError(t, w.EndDocument())
	require.Equal(t, "<?xml version=\"1.0\"?>\n<!-- hello -->", strings.TrimRight(buf.String(), "\n"))
}

func TestStartPIViaStartEnd(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartDocument("", "", ""))
	require.NoError(t, w.StartPI("target"))
	require.NoError(t, w.WriteString(" data"))
	require.NoError(t, w.EndPI())
	require.NoError(t, w.EndDocument())
	require.Equal(t, "<?xml version=\"1.0\"?>\n<?target data?>", strings.TrimRight(buf.String(), "\n"))
}

func TestCompleteDocument(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
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

	expected := "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n" +
		`<!-- Generated document -->` +
		`<catalog xmlns="http://example.com/catalog">` +
		`<book id="bk101"><title>XML Developer's Guide</title><price>44.95</price></book>` +
		`<book id="bk102"><title>Midnight Rain</title><price>5.95</price></book>` +
		`</catalog>`
	require.Equal(t, expected, strings.TrimRight(buf.String(), "\n"))
}

func TestCompleteDocumentIndented(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf).Indent("  ")
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
	t.Parallel()
	// Writer should work without StartDocument for fragment output
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.WriteString("hello"))
	require.NoError(t, w.EndElement())
	require.Equal(t, `<root>hello</root>`, buf.String())
}

func TestWriteRawInDocument(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartDocument("", "", ""))
	require.NoError(t, w.WriteRaw("\n"))
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.EndElement())
	require.NoError(t, w.EndDocument())
	require.Equal(t, "<?xml version=\"1.0\"?>\n\n<root/>", strings.TrimRight(buf.String(), "\n"))
}

func TestNamespaceElementWithAttribute(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
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
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.StartElement("a"))
	require.NoError(t, w.EndElement())
	require.NoError(t, w.StartElement("b"))
	require.NoError(t, w.EndElement())
	require.NoError(t, w.EndElement())
	require.Equal(t, `<root><a/><b/></root>`, buf.String())
}

func TestWriteAttributeNS(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
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
