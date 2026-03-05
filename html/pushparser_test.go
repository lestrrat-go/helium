package html_test

import (
	"bytes"
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
	require.NoError(t, html.WriteDoc(&buf, doc))
	return buf.String()
}

func TestHTMLPushParserSingleChunk(t *testing.T) {
	input := []byte(testHTML)

	want, err := html.Parse(t.Context(), input)
	require.NoError(t, err)

	pp := html.NewPushParser(t.Context())
	require.NoError(t, pp.Push(input))
	got, err := pp.Close()
	require.NoError(t, err)
	require.Equal(t, dumpHTMLDoc(t, want), dumpHTMLDoc(t, got))
}

func TestHTMLPushParserMultiChunk(t *testing.T) {
	input := []byte(testHTML)

	want, err := html.Parse(t.Context(), input)
	require.NoError(t, err)

	pp := html.NewPushParser(t.Context())
	// Push in 20-byte chunks
	for i := 0; i < len(input); i += 20 {
		end := i + 20
		if end > len(input) {
			end = len(input)
		}
		require.NoError(t, pp.Push(input[i:end]))
	}

	got, err := pp.Close()
	require.NoError(t, err)
	require.Equal(t, dumpHTMLDoc(t, want), dumpHTMLDoc(t, got))
}

func TestHTMLPushParserSAXMode(t *testing.T) {
	input := []byte(testHTML)

	var elements []string
	handler := &html.SAXCallbacks{
		OnStartDocument: html.StartDocumentFunc(func() error { return nil }),
		OnEndDocument:   html.EndDocumentFunc(func() error { return nil }),
		OnStartElement: html.StartElementFunc(func(name string, attrs []html.Attribute) error {
			elements = append(elements, name)
			return nil
		}),
		OnEndElement:            html.EndElementFunc(func(name string) error { return nil }),
		OnCharacters:            html.CharactersFunc(func(ch []byte) error { return nil }),
		OnComment:               html.CommentFunc(func(value []byte) error { return nil }),
		OnCDataBlock:            html.CDataBlockFunc(func(value []byte) error { return nil }),
		OnInternalSubset:        html.InternalSubsetFunc(func(name, eid, sid string) error { return nil }),
		OnProcessingInstruction: html.ProcessingInstructionFunc(func(t, d string) error { return nil }),
		OnIgnorableWhitespace:   html.IgnorableWhitespaceFunc(func(ch []byte) error { return nil }),
		OnError:                 html.ErrorFunc(func(err error) error { return nil }),
		OnWarning:               html.WarningFunc(func(err error) error { return nil }),
		OnSetDocumentLocator:    html.SetDocumentLocatorFunc(func(loc html.DocumentLocator) error { return nil }),
	}

	pp := html.NewSAXPushParser(t.Context(), handler)
	require.NoError(t, pp.Push(input))
	doc, err := pp.Close()
	require.NoError(t, err)
	require.Nil(t, doc, "SAX mode should not return a document")
	require.Contains(t, elements, "html")
	require.Contains(t, elements, "body")
	require.Contains(t, elements, "h1")
	require.Contains(t, elements, "p")
}

func TestHTMLPushParserIOCopy(t *testing.T) {
	input := []byte(testHTML)

	want, err := html.Parse(t.Context(), input)
	require.NoError(t, err)

	pp := html.NewPushParser(t.Context())
	n, err := io.Copy(pp, bytes.NewReader(input))
	require.NoError(t, err)
	require.Equal(t, int64(len(input)), n)

	got, err := pp.Close()
	require.NoError(t, err)
	require.Equal(t, dumpHTMLDoc(t, want), dumpHTMLDoc(t, got))
}
