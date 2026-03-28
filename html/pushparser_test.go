package html_test

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/lestrrat-go/helium/html"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

const testHTML = `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
  <h1>Hello</h1>
  <p>World &amp; friends</p>
</body>
</html>`

func dumpHTMLDoc(t *testing.T, doc *helium.Document) string {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, html.NewWriter().WriteTo(&buf, doc))
	return buf.String()
}

func TestHTMLPushParserSingleChunk(t *testing.T) {
	input := []byte(testHTML)

	want, err := html.NewParser().Parse(t.Context(), input)
	require.NoError(t, err)

	pp := html.NewParser().NewPushParser()
	require.NoError(t, pp.Push(input))
	got, err := pp.Close(t.Context())
	require.NoError(t, err)
	require.Equal(t, dumpHTMLDoc(t, want), dumpHTMLDoc(t, got))
}

func TestHTMLPushParserMultiChunk(t *testing.T) {
	input := []byte(testHTML)

	want, err := html.NewParser().Parse(t.Context(), input)
	require.NoError(t, err)

	pp := html.NewParser().NewPushParser()
	// Push in 20-byte chunks
	for i := 0; i < len(input); i += 20 {
		end := i + 20
		if end > len(input) {
			end = len(input)
		}
		require.NoError(t, pp.Push(input[i:end]))
	}

	got, err := pp.Close(t.Context())
	require.NoError(t, err)
	require.Equal(t, dumpHTMLDoc(t, want), dumpHTMLDoc(t, got))
}

func TestHTMLPushParserSAXMode(t *testing.T) {
	input := []byte(testHTML)

	var elements []string
	handler := &html.SAXCallbacks{}
	handler.SetOnStartDocument(html.StartDocumentFunc(func() error { return nil }))
	handler.SetOnEndDocument(html.EndDocumentFunc(func() error { return nil }))
	handler.SetOnStartElement(html.StartElementFunc(func(name string, attrs []html.Attribute) error {
		elements = append(elements, name)
		return nil
	}))
	handler.SetOnEndElement(html.EndElementFunc(func(name string) error { return nil }))
	handler.SetOnCharacters(html.CharactersFunc(func(ch []byte) error { return nil }))
	handler.SetOnComment(html.CommentFunc(func(value []byte) error { return nil }))
	handler.SetOnCDataBlock(html.CDataBlockFunc(func(value []byte) error { return nil }))
	handler.SetOnInternalSubset(html.InternalSubsetFunc(func(name, eid, sid string) error { return nil }))
	handler.SetOnProcessingInstruction(html.ProcessingInstructionFunc(func(t, d string) error { return nil }))
	handler.SetOnIgnorableWhitespace(html.IgnorableWhitespaceFunc(func(ch []byte) error { return nil }))
	handler.SetOnError(html.ErrorFunc(func(err error) error { return nil }))
	handler.SetOnWarning(html.WarningFunc(func(err error) error { return nil }))
	handler.SetOnSetDocumentLocator(html.SetDocumentLocatorFunc(func(loc html.DocumentLocator) error { return nil }))

	pp := html.NewParser().NewSAXPushParser(handler)
	require.NoError(t, pp.Push(input))
	doc, err := pp.Close(t.Context())
	require.NoError(t, err)
	require.Nil(t, doc, "SAX mode should not return a document")
	require.Contains(t, elements, "html")
	require.Contains(t, elements, "body")
	require.Contains(t, elements, "h1")
	require.Contains(t, elements, "p")
}

func TestHTMLParseRespectsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel() // cancel immediately

	_, err := html.NewParser().Parse(ctx, []byte(testHTML))
	require.ErrorIs(t, err, context.Canceled)
}

func TestHTMLParseWithSAXRespectsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	handler := &html.SAXCallbacks{}
	err := html.NewParser().ParseWithSAX(ctx, []byte(testHTML), handler)
	require.ErrorIs(t, err, context.Canceled)
}

func TestHTMLPushParserCloseRespectsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())

	pp := html.NewParser().NewPushParser()
	require.NoError(t, pp.Push([]byte(testHTML)))

	cancel()

	_, err := pp.Close(ctx)
	require.ErrorIs(t, err, context.Canceled)
}

func TestHTMLPushParserIOCopy(t *testing.T) {
	input := []byte(testHTML)

	want, err := html.NewParser().Parse(t.Context(), input)
	require.NoError(t, err)

	pp := html.NewParser().NewPushParser()
	n, err := io.Copy(pp, bytes.NewReader(input))
	require.NoError(t, err)
	require.Equal(t, int64(len(input)), n)

	got, err := pp.Close(t.Context())
	require.NoError(t, err)
	require.Equal(t, dumpHTMLDoc(t, want), dumpHTMLDoc(t, got))
}
