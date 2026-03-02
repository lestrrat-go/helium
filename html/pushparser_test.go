package html

import (
	"bytes"
	"io"
	"testing"

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
	require.NoError(t, WriteDoc(&buf, doc))
	return buf.String()
}

func TestHTMLPushParserSingleChunk(t *testing.T) {
	input := []byte(testHTML)

	want, err := Parse(input)
	require.NoError(t, err)

	pp := NewPushParser()
	require.NoError(t, pp.Push(input))
	got, err := pp.Close()
	require.NoError(t, err)
	require.Equal(t, dumpHTMLDoc(t, want), dumpHTMLDoc(t, got))
}

func TestHTMLPushParserMultiChunk(t *testing.T) {
	input := []byte(testHTML)

	want, err := Parse(input)
	require.NoError(t, err)

	pp := NewPushParser()
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
	handler := &SAXCallbacks{
		StartDocumentHandler: StartDocumentFunc(func() error { return nil }),
		EndDocumentHandler:   EndDocumentFunc(func() error { return nil }),
		StartElementHandler: StartElementFunc(func(name string, attrs []Attribute) error {
			elements = append(elements, name)
			return nil
		}),
		EndElementHandler:            EndElementFunc(func(name string) error { return nil }),
		CharactersHandler:            CharactersFunc(func(ch []byte) error { return nil }),
		CommentHandler:               CommentFunc(func(value []byte) error { return nil }),
		CDataBlockHandler:            CDataBlockFunc(func(value []byte) error { return nil }),
		InternalSubsetHandler:        InternalSubsetFunc(func(name, eid, sid string) error { return nil }),
		ProcessingInstructionHandler: ProcessingInstructionFunc(func(t, d string) error { return nil }),
		IgnorableWhitespaceHandler:   IgnorableWhitespaceFunc(func(ch []byte) error { return nil }),
		ErrorHandler:                 ErrorFunc(func(msg string, args ...interface{}) error { return nil }),
		WarningHandler:               WarningFunc(func(msg string, args ...interface{}) error { return nil }),
		SetDocumentLocatorHandler:    SetDocumentLocatorFunc(func(loc DocumentLocator) error { return nil }),
	}

	pp := NewPushParserWithSAX(handler)
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

	want, err := Parse(input)
	require.NoError(t, err)

	pp := NewPushParser()
	n, err := io.Copy(pp, bytes.NewReader(input))
	require.NoError(t, err)
	require.Equal(t, int64(len(input)), n)

	got, err := pp.Close()
	require.NoError(t, err)
	require.Equal(t, dumpHTMLDoc(t, want), dumpHTMLDoc(t, got))
}
