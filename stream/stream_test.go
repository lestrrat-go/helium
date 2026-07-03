package stream_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
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

func TestStartElementRejectsInvalidName(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		arg  string
	}{
		{"injected attribute", `root injected="1"`},
		{"close bracket", "root>"},
		{"leading digit", "1root"},
		{"whitespace", "ro ot"},
		{"quote", `root"`},
		{"two colons", "a:b:c"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			w := stream.NewWriter(&buf)
			require.Error(t, w.StartElement(tc.arg), "StartElement(%q) must reject invalid name", tc.arg)
			require.Empty(t, buf.String(), "no markup must be emitted for an invalid name")
		})
	}
}

func TestStartElementAcceptsValidName(t *testing.T) {
	t.Parallel()
	cases := []string{"root", "a:b", "xml:lang", "_x", "ns0:elem", "名前"}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			w := stream.NewWriter(&buf)
			require.NoError(t, w.StartElement(name))
			require.NoError(t, w.EndElement())
			require.Equal(t, "<"+name+"/>", buf.String())
		})
	}
}

func TestStartAttributeRejectsInvalidName(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		arg  string
	}{
		{"injected attribute", `key="1" injected`},
		{"close bracket", "key>"},
		{"whitespace", "k ey"},
		{"quote", `key"`},
		{"leading digit", "1key"},
		{"two colons", "a:b:c"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			w := stream.NewWriter(&buf)
			require.NoError(t, w.StartElement("root"))
			require.Error(t, w.StartAttribute(tc.arg), "StartAttribute(%q) must reject invalid name", tc.arg)
		})
	}
}

func TestStartAttributeAcceptsValidName(t *testing.T) {
	t.Parallel()
	cases := []string{"key", "a:b", "xml:lang", "_x", "名前", "xmlns", "xmlns:foo"}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			w := stream.NewWriter(&buf)
			require.NoError(t, w.StartElement("root"))
			require.NoError(t, w.WriteAttribute(name, "v"))
			require.NoError(t, w.EndElement())
			require.Equal(t, `<root `+name+`="v"/>`, buf.String())
		})
	}
}

func TestStartElementNSRejectsInvalidParts(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.Error(t, w.StartElementNS("bad prefix", "local", "urn:x"))
	require.Empty(t, buf.String())

	buf.Reset()
	w = stream.NewWriter(&buf)
	require.Error(t, w.StartElementNS("p", "bad local", "urn:x"))
	require.Empty(t, buf.String())
}

func TestStartAttributeNSRejectsInvalidParts(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartElement("root"))
	require.Error(t, w.StartAttributeNS("bad prefix", "local", "urn:x"))

	buf.Reset()
	w = stream.NewWriter(&buf)
	require.NoError(t, w.StartElement("root"))
	require.Error(t, w.StartAttributeNS("p", "bad local", "urn:x"))
}

func TestNamespaceNSRejectsReservedMisuse(t *testing.T) {
	t.Parallel()
	const (
		xmlNS   = "http://www.w3.org/XML/1998/namespace"
		xmlnsNS = "http://www.w3.org/2000/xmlns/"
	)

	t.Run("element rebinding xml prefix rejected", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.Error(t, w.StartElementNS("xml", "foo", "urn:bad"))
		require.Empty(t, buf.String())
	})
	t.Run("element xmlns prefix rejected", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.Error(t, w.StartElementNS("xmlns", "foo", "urn:x"))
		require.Empty(t, buf.String())
	})
	t.Run("element foreign prefix on XML namespace rejected", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.Error(t, w.StartElementNS("p", "foo", xmlNS))
		require.Empty(t, buf.String())
	})
	t.Run("element default binding of xmlns namespace rejected", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.Error(t, w.StartElementNS("", "foo", xmlnsNS))
		require.Empty(t, buf.String())
	})
	t.Run("element xml prefix bound to XML namespace accepted", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartElementNS("xml", "foo", xmlNS))
		require.NoError(t, w.EndElement())
	})
	t.Run("attribute rebinding xml prefix rejected", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartElement("root"))
		buf.Reset()
		require.Error(t, w.StartAttributeNS("xml", "lang", "urn:bad"))
		require.Empty(t, buf.String())
	})
	t.Run("attribute xmlns prefix rejected", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartElement("root"))
		buf.Reset()
		require.Error(t, w.StartAttributeNS("xmlns", "p", "urn:x"))
		require.Empty(t, buf.String())
	})
	t.Run("attribute xml:lang with empty URI accepted", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartElement("root"))
		require.NoError(t, w.WriteAttributeNS("xml", "lang", "", "en"))
		require.NoError(t, w.EndElement())
		require.Equal(t, `<root xml:lang="en"/>`, buf.String())
	})
}

func TestStartAttributeNSRejectsSameScopeConflict(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartElementNS("a", "root", "urn:A"))
	before := buf.String()
	// Reusing prefix "a" with a different URI on the same element must be
	// rejected, not silently dropped.
	require.Error(t, w.StartAttributeNS("a", "attr", "urn:B"))
	require.Equal(t, before, buf.String())

	// Reusing the same prefix with the same URI is a harmless no-op.
	require.NoError(t, w.StartAttributeNS("a", "attr", "urn:A"))
	require.NoError(t, w.EndAttribute())
	require.NoError(t, w.EndElement())
}

func TestNamespaceURIRejectsInvalidChars(t *testing.T) {
	t.Parallel()
	badUTF8 := string([]byte{0xff})
	t.Run("element NS NUL rejected", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.Error(t, w.StartElementNS("", "a", "x\x00"))
		require.Empty(t, buf.String())
	})
	t.Run("element NS invalid UTF-8 rejected", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.Error(t, w.StartElementNS("p", "a", "urn:"+badUTF8))
		require.Empty(t, buf.String())
	})
	t.Run("attribute NS NUL rejected", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartElement("root"))
		buf.Reset()
		require.Error(t, w.StartAttributeNS("p", "a", "x\x00"))
		require.Empty(t, buf.String())
	})
	t.Run("attribute NS invalid UTF-8 rejected", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartElement("root"))
		buf.Reset()
		require.Error(t, w.StartAttributeNS("p", "a", "urn:"+badUTF8))
		require.Empty(t, buf.String())
	})
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

// TestWriteElementRestrictedCharEquivalence asserts that WriteElement /
// WriteElementNS behave identically to StartElement(NS) + WriteString +
// EndElement for an XML 1.1 restricted control character: rejected in XML 1.0
// output, and emitted as a decimal character reference under XMLVersion("1.1").
func TestWriteElementRestrictedCharEquivalence(t *testing.T) {
	t.Parallel()
	const content = "A\x01B" // U+0001: XML 1.1 restricted control char

	writeConvenience := func(w stream.Writer) error {
		return w.WriteElement("e", content)
	}
	writeConvenienceNS := func(w stream.Writer) error {
		return w.WriteElementNS("", "e", "", content)
	}
	writeExpanded := func(w stream.Writer) error {
		if err := w.StartElement("e"); err != nil {
			return err
		}
		if err := w.WriteString(content); err != nil {
			return err
		}
		return w.EndElement()
	}

	run := func(t *testing.T, xml11 bool, write func(stream.Writer) error) (string, error) {
		t.Helper()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		if xml11 {
			w = w.XMLVersion("1.1")
		}
		err := write(w)
		return buf.String(), err
	}

	// XML 1.0: all three paths reject the restricted char.
	for _, write := range []func(stream.Writer) error{writeConvenience, writeConvenienceNS, writeExpanded} {
		_, err := run(t, false, write)
		require.Error(t, err, "xml 1.0 must reject the restricted control char")
	}

	// XML 1.1: all three paths succeed and emit the decimal character reference,
	// and the two convenience methods match the expanded StartElement+WriteString
	// path byte-for-byte.
	expanded, err := run(t, true, writeExpanded)
	require.NoError(t, err)
	require.Equal(t, "<e>A&#1;B</e>", expanded)

	got, err := run(t, true, writeConvenience)
	require.NoError(t, err)
	require.Equal(t, expanded, got)

	gotNS, err := run(t, true, writeConvenienceNS)
	require.NoError(t, err)
	require.Equal(t, expanded, gotNS)
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

func TestPIInvalidUTF8TargetRejected(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	// An invalid UTF-8 byte decodes to U+FFFD, which is a valid NCName char,
	// so the target must be rejected on the encoding check, not accepted.
	err := w.StartPI(string([]byte{0xff}))
	require.Error(t, err)
	require.Empty(t, buf.String(), "no PI bytes must be emitted for an invalid target")
}

func TestPINormalTargetAccepted(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.StartPI("target"))
	require.NoError(t, w.EndPI())
	require.NoError(t, w.EndElement())
	require.Equal(t, `<root><?target?></root>`, buf.String())
}

func TestPIColonTargetRejected(t *testing.T) {
	t.Parallel()
	for _, target := range []string{"a:b", ":", "a:", ":a"} {
		t.Run(target, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			w := stream.NewWriter(&buf)
			err := w.StartPI(target)
			require.Error(t, err)
			require.Empty(t, buf.String(), "no PI bytes must be emitted for a colon target")
		})
	}
}

func TestPIReplacementCharTargetAccepted(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	// A genuinely encoded U+FFFD is valid UTF-8 and a valid NCName char.
	require.NoError(t, w.StartPI("�"))
	require.NoError(t, w.EndPI())
	require.Equal(t, "<?�?>", buf.String())
}

func TestPIBadTargetInOpenStartTagDoesNotMutate(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartElement("r"))
	// StartElement writes "<r" and leaves the tag open (state stateName) so it
	// can still self-close. A rejected PI must NOT call closeTagIfOpen (which
	// would emit ">" and force "<r></r>"); the element must remain self-closeable.
	require.Equal(t, "<r", buf.String())
	err := w.StartPI("1bad")
	require.Error(t, err)
	require.Equal(t, "<r", buf.String(), "the open start tag must not be flushed by a rejected PI")
	require.NoError(t, w.EndElement())
	require.Equal(t, "<r/>", buf.String())
}

func TestCommentWellFormednessRejected(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		content string
	}{
		{name: "double-dash", content: "a--b"},
		{name: "trailing-dash", content: "a-"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			w := stream.NewWriter(&buf)
			require.Error(t, w.WriteComment(tc.content))
		})
	}
}

func TestCommentDashSplitAcrossWrites(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartComment())
	require.NoError(t, w.WriteString("a-"))
	require.Error(t, w.WriteString("-b"))
}

func TestCommentTrailingDashSplitAtEnd(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartComment())
	require.NoError(t, w.WriteString("a-"))
	require.Error(t, w.EndComment())
}

func TestCommentDashChunksNoFalsePositive(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.StartComment())
	require.NoError(t, w.WriteString("a-"))
	require.NoError(t, w.WriteString("b"))
	require.NoError(t, w.EndComment())
	require.NoError(t, w.EndElement())
	require.Equal(t, `<root><!--a-b--></root>`, buf.String())
}

func TestCommentValidStillSucceeds(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.WriteComment(" ok "))
	require.NoError(t, w.EndElement())
	require.Equal(t, `<root><!-- ok --></root>`, buf.String())
}

func TestPIWellFormednessRejected(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		target  string
		content string
	}{
		{name: "content-end-delim", target: "t", content: "a?>b"},
		{name: "target-starts-digit", target: "123bad", content: "x"},
		{name: "target-has-space", target: "a b", content: "x"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			w := stream.NewWriter(&buf)
			require.Error(t, w.WritePI(tc.target, tc.content))
		})
	}
}

func TestPIDelimSplitAcrossWrites(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartPI("t"))
	require.NoError(t, w.WriteString(" a?"))
	require.Error(t, w.WriteString(">b"))
}

func TestPIQuestionChunksNoFalsePositive(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.StartPI("t"))
	require.NoError(t, w.WriteString(" a?"))
	require.NoError(t, w.WriteString("x"))
	require.NoError(t, w.EndPI())
	require.NoError(t, w.EndElement())
	require.Equal(t, `<root><?t a?x?></root>`, buf.String())
}

func TestPIValidStillSucceeds(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.WritePI("php", "echo 1"))
	require.NoError(t, w.EndElement())
	require.Equal(t, `<root><?php echo 1?></root>`, buf.String())
}

func TestCommentPIRoundTripsThroughParser(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartDocument("", "", ""))
	require.NoError(t, w.WritePI("php", "echo 1"))
	require.NoError(t, w.StartElement("root"))
	require.NoError(t, w.WriteComment(" a comment "))
	require.NoError(t, w.WriteString("text"))
	require.NoError(t, w.EndElement())
	require.NoError(t, w.EndDocument())

	_, err := helium.NewParser().Parse(t.Context(), buf.Bytes())
	require.NoError(t, err)
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

func TestCDATAWriteStringEscapesTerminator(t *testing.T) {
	t.Parallel()
	t.Run("single write", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartElement("root"))
		require.NoError(t, w.StartCDATA())
		require.NoError(t, w.WriteString("]]>"))
		require.NoError(t, w.EndCDATA())
		require.NoError(t, w.EndElement())
		out := buf.String()
		require.Equal(t, `<root><![CDATA[]]]]><![CDATA[>]]></root>`, out)
		requireNoRawCDATATerminator(t, out)
	})
	t.Run("split across writes", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartElement("root"))
		require.NoError(t, w.StartCDATA())
		require.NoError(t, w.WriteString("foo]]"))
		require.NoError(t, w.WriteString(">bar"))
		require.NoError(t, w.EndCDATA())
		require.NoError(t, w.EndElement())
		out := buf.String()
		require.Equal(t, `<root><![CDATA[foo]]]]><![CDATA[>bar]]></root>`, out)
		requireNoRawCDATATerminator(t, out)
	})
	t.Run("terminator straddling call boundary emits no empty write", func(t *testing.T) {
		t.Parallel()
		// "]]>" split as "]]" then ">" forces the split point to land at
		// the very start of the second WriteString. The Writer must not
		// flush an empty string before the CDATA split marker, which would
		// trip a side-effecting io.StringWriter implementation.
		sw := &noEmptyStringWriter{}
		w := stream.NewWriter(sw)
		require.NoError(t, w.StartElement("root"))
		require.NoError(t, w.StartCDATA())
		require.NoError(t, w.WriteString("]]"))
		require.NoError(t, w.WriteString(">"))
		require.NoError(t, w.EndCDATA())
		require.NoError(t, w.EndElement())
		require.NoError(t, w.Error())
		require.Empty(t, sw.empties, "Writer emitted an empty WriteString")
		out := sw.buf.String()
		require.Equal(t, `<root><![CDATA[]]]]><![CDATA[>]]></root>`, out)
		requireNoRawCDATATerminator(t, out)
	})
}

// noEmptyStringWriter is an io.StringWriter that records any empty WriteString
// call so tests can assert the Writer never emits an avoidable empty write.
type noEmptyStringWriter struct {
	buf     bytes.Buffer
	empties int
}

func (w *noEmptyStringWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		w.empties++
	}
	return w.buf.Write(p)
}

func (w *noEmptyStringWriter) WriteString(s string) (int, error) {
	if s == "" {
		w.empties++
	}
	return w.buf.WriteString(s)
}

// requireNoRawCDATATerminator scans the emitted XML and asserts that no CDATA
// section contains a raw "]]>" before its own closing delimiter, which would
// make the output malformed.
func requireNoRawCDATATerminator(t *testing.T, s string) {
	t.Helper()
	const open = "<![CDATA["
	for {
		i := strings.Index(s, open)
		if i < 0 {
			break
		}
		body := s[i+len(open):]
		end := strings.Index(body, "]]>")
		require.GreaterOrEqual(t, end, 0, "unterminated CDATA section in %q", s)
		require.NotContains(t, body[:end], "]]>", "raw ]]> inside CDATA section in %q", s)
		s = body[end+3:]
	}
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

func TestStartAttributeNSAfterTagCloseLeavesNamespaceUnmutated(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartElement("root"))
	// Write text to close the start tag so the writer is no longer in
	// stateName. A StartAttributeNS call now must be rejected.
	require.NoError(t, w.WriteString("x"))
	require.Error(t, w.StartAttributeNS("ns", "attr", "http://example.com"))
	// The rejected call must not have recorded the ns:->uri declaration, so a
	// child element with the same prefix still emits its xmlns:ns binding.
	require.NoError(t, w.StartElementNS("ns", "child", "http://example.com"))
	require.NoError(t, w.EndElement())
	require.NoError(t, w.EndElement())
	require.Equal(t, `<root>x<ns:child xmlns:ns="http://example.com"/></root>`, buf.String())
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

func TestStickyErrorPreservedInConvenienceHelpers(t *testing.T) {
	t.Parallel()
	// With an underlying writer that fails immediately, the sticky I/O error
	// must win over the new content/target validation error in the one-shot
	// convenience helpers.
	t.Run("WriteComment", func(t *testing.T) {
		t.Parallel()
		fw := &failWriter{failAfter: 0}
		w := stream.NewWriter(fw)
		require.Error(t, w.StartElement("root"))
		err := w.WriteComment("a--b")
		require.Error(t, err)
		require.Equal(t, "write failed", err.Error())
	})
	t.Run("WritePI", func(t *testing.T) {
		t.Parallel()
		fw := &failWriter{failAfter: 0}
		w := stream.NewWriter(fw)
		require.Error(t, w.StartElement("root"))
		err := w.WritePI("t", "a?>b")
		require.Error(t, err)
		require.Equal(t, "write failed", err.Error())
	})
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

func TestStartDocumentVersionStandaloneValidation(t *testing.T) {
	t.Parallel()
	t.Run("version injection rejected", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		err := w.StartDocument(`1.0"?><x/>`, "", "")
		require.Error(t, err)
		require.Empty(t, buf.String())
	})
	t.Run("non-1.x version rejected", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.Error(t, w.StartDocument("2.0", "", ""))
	})
	t.Run("valid 1.1 accepted", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartDocument("1.1", "", ""))
	})
	t.Run("invalid standalone rejected", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		err := w.StartDocument("1.0", "", `yes"?><x/>`)
		require.Error(t, err)
		require.Empty(t, buf.String())
	})
	t.Run("standalone yes/no accepted", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartDocument("1.0", "", "yes"))
		require.Contains(t, buf.String(), `standalone="yes"`)
	})
}

func TestDTDIdentifierValidation(t *testing.T) {
	t.Parallel()
	t.Run("name injection rejected", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartDocument("", "", ""))
		require.Error(t, w.StartDTD(`x><!ENTITY e "pwn">`, "", ""))
	})
	t.Run("sysid with both quotes rejected", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartDocument("", "", ""))
		require.Error(t, w.StartDTD("root", "", `a'b"c`))
	})
	t.Run("sysid with angle brackets accepted", func(t *testing.T) {
		t.Parallel()
		// A SystemLiteral may contain any XML char except the delimiting
		// quote, so '<' and '>' are valid and must round-trip.
		for _, sysid := range []string{"a>b", "a<b"} {
			var buf bytes.Buffer
			w := stream.NewWriter(&buf)
			require.NoError(t, w.StartDocument("", "", ""))
			require.NoError(t, w.StartDTD("root", "", sysid))
			require.NoError(t, w.EndDTD())
			require.NoError(t, w.StartElement("root"))
			require.NoError(t, w.EndElement())
			require.NoError(t, w.EndDocument())
			require.Contains(t, buf.String(), `SYSTEM "`+sysid+`"`)
		}
	})
	t.Run("pubid invalid char rejected", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartDocument("", "", ""))
		require.Error(t, w.StartDTD("root", "pub<id", "sys"))
	})
	t.Run("notation identifiers validated", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartDocument("", "", ""))
		require.NoError(t, w.StartDTD("root", "", "sys"))
		require.Error(t, w.WriteDTDNotation("n", "", `a'b"c`))
	})
	t.Run("valid dtd accepted", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartDocument("", "", ""))
		require.NoError(t, w.StartDTD("root", "-//W3C//DTD//EN", "http://example.com/x.dtd"))
		require.NoError(t, w.EndDTD())
	})
}

func TestInvalidXMLCharRejection(t *testing.T) {
	t.Parallel()
	t.Run("text NUL rejected", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartElement("a"))
		require.Error(t, w.WriteString("x\x00y"))
	})
	t.Run("attribute NUL rejected", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartElement("a"))
		require.Error(t, w.WriteAttribute("k", "v\x00"))
	})
	t.Run("comment control char rejected", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartElement("a"))
		require.Error(t, w.WriteComment("c\x01"))
	})
	t.Run("PI control char rejected", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartElement("a"))
		require.Error(t, w.WritePI("tgt", "d\x00"))
	})
	t.Run("CDATA NUL rejected", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartElement("a"))
		require.Error(t, w.WriteCDATA("c\x00d"))
	})
	t.Run("valid chars accepted", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartElement("a"))
		require.NoError(t, w.WriteString("hello\tworld\n"))
		require.NoError(t, w.EndElement())
	})
}

func TestInvalidUTF8Rejection(t *testing.T) {
	t.Parallel()
	bad := string([]byte{0xff})
	t.Run("text content rejected", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartElement("a"))
		require.Error(t, w.WriteString(bad))
	})
	t.Run("system id rejected", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartDocument("", "", ""))
		require.Error(t, w.StartDTD("root", "", "sys"+bad))
	})
}

func TestDTDFragmentInjectionRejected(t *testing.T) {
	t.Parallel()
	t.Run("element contentspec injection rejected", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartDocument("", "", ""))
		require.NoError(t, w.StartDTD("root", "", ""))
		require.Error(t, w.WriteDTDElement("root", `ANY><!ENTITY e "pwn"`))
		require.NotContains(t, buf.String(), "ENTITY")
	})
	t.Run("element contentspec less-than rejected", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartDocument("", "", ""))
		require.NoError(t, w.StartDTD("root", "", ""))
		require.Error(t, w.WriteDTDElement("root", `<!ELEMENT x ANY`))
	})
	t.Run("attlist body injection rejected", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartDocument("", "", ""))
		require.NoError(t, w.StartDTD("root", "", ""))
		require.Error(t, w.WriteDTDAttlist("root", `id CDATA #IMPLIED><!ENTITY e "pwn"`))
		require.NotContains(t, buf.String(), "ENTITY")
	})
	t.Run("attlist unquoted greater-than injection rejected", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartDocument("", "", ""))
		require.NoError(t, w.StartDTD("root", "", ""))
		require.Error(t, w.WriteDTDAttlist("root", `x CDATA "v"> <!ENTITY e "pwn"`))
		require.NotContains(t, buf.String(), "ENTITY")
	})
	t.Run("attlist quoted greater-than accepted", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartDocument("", "", ""))
		require.NoError(t, w.StartDTD("root", "", ""))
		require.NoError(t, w.WriteDTDAttlist("root", `a CDATA "a>b"`))
		require.NoError(t, w.EndDTD())
		require.Contains(t, buf.String(), `<!ATTLIST root a CDATA "a>b">`)
	})
	t.Run("attlist quoted less-than rejected", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartDocument("", "", ""))
		require.NoError(t, w.StartDTD("root", "", ""))
		// '<' is forbidden in an AttValue even inside a quoted literal.
		require.Error(t, w.WriteDTDAttlist("root", `a CDATA "<"`))
		require.NotContains(t, buf.String(), "ATTLIST")
	})
	t.Run("attlist unterminated literal trailing greater-than rejected", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartDocument("", "", ""))
		require.NoError(t, w.StartDTD("root", "", ""))
		// The opening quote is never closed; malformed (unterminated)
		// quoting is rejected, so the trailing '>' cannot smuggle markup.
		require.Error(t, w.WriteDTDAttlist("root", `a CDATA "unterminated literal >`))
		require.NotContains(t, buf.String(), "ATTLIST")
	})
	t.Run("valid contentspec and attlist still accepted", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartDocument("", "", ""))
		require.NoError(t, w.StartDTD("root", "", ""))
		require.NoError(t, w.WriteDTDElement("root", "(a|b)*"))
		require.NoError(t, w.WriteDTDElement("a", "(#PCDATA)"))
		require.NoError(t, w.WriteDTDElement("b", "EMPTY"))
		require.NoError(t, w.WriteDTDAttlist("a", `id ID #REQUIRED kind CDATA #IMPLIED`))
		require.NoError(t, w.EndDTD())
	})
}

func TestOneShotHelpersPreValidateBeforeMutation(t *testing.T) {
	t.Parallel()
	bad := "x\x00"
	t.Run("WriteComment", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartElement("root"))
		buf.Reset()
		require.Error(t, w.WriteComment(bad))
		require.Empty(t, buf.String())
	})
	t.Run("WritePI", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartElement("root"))
		buf.Reset()
		require.Error(t, w.WritePI("target", bad))
		require.Empty(t, buf.String())
	})
	t.Run("WriteCDATA", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartElement("root"))
		buf.Reset()
		require.Error(t, w.WriteCDATA(bad))
		require.Empty(t, buf.String())
	})
	t.Run("WriteAttribute", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartElement("root"))
		buf.Reset()
		require.Error(t, w.WriteAttribute("attr", bad))
		require.Empty(t, buf.String())
	})
	t.Run("WriteElement", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartElement("root"))
		buf.Reset()
		require.Error(t, w.WriteElement("child", bad))
		require.Empty(t, buf.String())
	})
}

func TestStartDocumentRejectsMalformedEncName(t *testing.T) {
	t.Parallel()
	// A value that is not a valid XML EncName (contains a space) must be rejected
	// before any output is written, even though the lenient encoding.Load would
	// otherwise normalize and accept it.
	for _, bad := range []string{"utf 8", "1utf8", "utf8\"?><x/>", "utf+8"} {
		t.Run(bad, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			w := stream.NewWriter(&buf)
			err := w.StartDocument("1.0", bad, "")
			require.Error(t, err)
			require.Contains(t, err.Error(), "invalid encoding name")
			require.Empty(t, buf.String(), "writer must be unmutated on rejection")
		})
	}
}

func TestStartDocumentUnsupportedEncNameLeavesWriterUnmutated(t *testing.T) {
	t.Parallel()
	// A syntactically valid but unsupported EncName must be rejected before the
	// XML-declaration prefix is written.
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	err := w.StartDocument("1.0", "BOGUS-999", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported encoding")
	require.Empty(t, buf.String(), "writer must be unmutated on rejection")
}

func TestWriteDTDEntityRejectsColonizedName(t *testing.T) {
	t.Parallel()
	// Entity names are NCNames in helium's parser (colons are forbidden), so a
	// "p:e" name must be rejected without mutating the writer.
	t.Run("internal", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartDocument("", "", ""))
		require.NoError(t, w.StartDTD("root", "", ""))
		buf.Reset()
		err := w.WriteDTDEntity(false, "p:e", "v")
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid DTD entity name")
		require.Empty(t, buf.String(), "writer must be unmutated on rejection")
	})
	t.Run("external", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartDocument("", "", ""))
		require.NoError(t, w.StartDTD("root", "", ""))
		buf.Reset()
		err := w.WriteDTDExternalEntity(false, "p:e", "", "ext.dtd", "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid DTD entity name")
		require.Empty(t, buf.String(), "writer must be unmutated on rejection")
	})
}

func TestWriteDTDEntityAcceptsNCName(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	w := stream.NewWriter(&buf)
	require.NoError(t, w.StartDocument("", "", ""))
	require.NoError(t, w.StartDTD("root", "", ""))
	require.NoError(t, w.WriteDTDEntity(false, "ent", "v"))
	require.NoError(t, w.WriteDTDExternalEntity(false, "logo", "", "logo.gif", "gif"))
}

// DTD element/attlist/notation/doctype/NDATA names follow the XML Name
// production, which is broader than QName: a Name may contain multiple colons
// (or leading/trailing colons). Such names must be accepted, while genuinely
// malformed names (injection, empty) must still be rejected.
func TestDTDNamesUseXMLNameProduction(t *testing.T) {
	t.Parallel()
	t.Run("multi-colon names accepted", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartDocument("", "", ""))
		// "a:b:c" is a valid Name but NOT a valid QName.
		require.NoError(t, w.StartDTD("a:b:c", "", ""))
		require.NoError(t, w.WriteDTDElement("a:b:c", "(#PCDATA)"))
		require.NoError(t, w.WriteDTDAttlist("a:b:c", "x CDATA #IMPLIED"))
		require.NoError(t, w.WriteDTDNotation("n:o:t", "", "sys"))
		require.NoError(t, w.WriteDTDExternalEntity(false, "img", "", "img.gif", "g:i:f"))
		require.NoError(t, w.EndDTD())
		out := buf.String()
		require.Contains(t, out, "<!DOCTYPE a:b:c")
		require.Contains(t, out, "<!ELEMENT a:b:c")
		require.Contains(t, out, "<!ATTLIST a:b:c")
		require.Contains(t, out, "<!NOTATION n:o:t")
		require.Contains(t, out, "NDATA g:i:f")
	})
	t.Run("invalid names still rejected", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		w := stream.NewWriter(&buf)
		require.NoError(t, w.StartDocument("", "", ""))
		require.NoError(t, w.StartDTD("root", "", ""))
		require.Error(t, w.WriteDTDElement("a b", "(#PCDATA)"), "space not a NameChar")
		require.Error(t, w.WriteDTDElement(`x><!ENTITY e "pwn">`, "(#PCDATA)"), "injection")
		require.Error(t, w.WriteDTDElement("", "(#PCDATA)"), "empty")
		require.Error(t, w.WriteDTDElement("1bad", "(#PCDATA)"), "bad start char")
		require.Error(t, w.WriteDTDAttlist("a b", "x CDATA #IMPLIED"), "space not a NameChar")
		require.Error(t, w.WriteDTDNotation("n>x", "", "sys"), "injection")
		require.Error(t, w.WriteDTDExternalEntity(false, "img", "", "img.gif", "g h"), "bad NDATA")
	})
}
