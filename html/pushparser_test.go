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

	want, err := Parse(t.Context(), input)
	require.NoError(t, err)

	pp := NewPushParser(t.Context())
	require.NoError(t, pp.Push(input))
	got, err := pp.Close()
	require.NoError(t, err)
	require.Equal(t, dumpHTMLDoc(t, want), dumpHTMLDoc(t, got))
}

func TestHTMLPushParserMultiChunk(t *testing.T) {
	input := []byte(testHTML)

	want, err := Parse(t.Context(), input)
	require.NoError(t, err)

	pp := NewPushParser(t.Context())
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
		OnStartDocument: StartDocumentFunc(func() error { return nil }),
		OnEndDocument:   EndDocumentFunc(func() error { return nil }),
		OnStartElement: StartElementFunc(func(name string, attrs []Attribute) error {
			elements = append(elements, name)
			return nil
		}),
		OnEndElement:            EndElementFunc(func(name string) error { return nil }),
		OnCharacters:            CharactersFunc(func(ch []byte) error { return nil }),
		OnComment:               CommentFunc(func(value []byte) error { return nil }),
		OnCDataBlock:            CDataBlockFunc(func(value []byte) error { return nil }),
		OnInternalSubset:        InternalSubsetFunc(func(name, eid, sid string) error { return nil }),
		OnProcessingInstruction: ProcessingInstructionFunc(func(t, d string) error { return nil }),
		OnIgnorableWhitespace:   IgnorableWhitespaceFunc(func(ch []byte) error { return nil }),
		OnError:                 ErrorFunc(func(err error) error { return nil }),
		OnWarning:               WarningFunc(func(err error) error { return nil }),
		OnSetDocumentLocator:    SetDocumentLocatorFunc(func(loc DocumentLocator) error { return nil }),
	}

	pp := NewSAXPushParser(t.Context(), handler)
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

	want, err := Parse(t.Context(), input)
	require.NoError(t, err)

	pp := NewPushParser(t.Context())
	n, err := io.Copy(pp, bytes.NewReader(input))
	require.NoError(t, err)
	require.Equal(t, int64(len(input)), n)

	got, err := pp.Close()
	require.NoError(t, err)
	require.Equal(t, dumpHTMLDoc(t, want), dumpHTMLDoc(t, got))
}
