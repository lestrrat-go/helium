package helium_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/sax"
	"github.com/stretchr/testify/require"
)

type parserChunkedReader struct {
	data  []byte
	chunk int
}

func (r *parserChunkedReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	n := r.chunk
	if n > len(r.data) {
		n = len(r.data)
	}
	if n > len(p) {
		n = len(p)
	}
	copy(p, r.data[:n])
	r.data = r.data[n:]
	return n, nil
}

func TestChunkedReaderPreservesIgnorableWhitespaceClassification(t *testing.T) {
	var events []string

	h := sax.New()
	h.SetOnSetDocumentLocator(sax.SetDocumentLocatorFunc(func(context.Context, sax.DocumentLocator) error { return nil }))
	h.SetOnStartDocument(sax.StartDocumentFunc(func(context.Context) error { return nil }))
	h.SetOnEndDocument(sax.EndDocumentFunc(func(context.Context) error { return nil }))
	h.SetOnStartElementNS(sax.StartElementNSFunc(func(context.Context, string, string, string, []sax.Namespace, []sax.Attribute) error {
		return nil
	}))
	h.SetOnEndElementNS(sax.EndElementNSFunc(func(context.Context, string, string, string) error { return nil }))
	h.SetOnCharacters(sax.CharactersFunc(func(_ context.Context, ch []byte) error {
		events = append(events, "characters:"+string(ch))
		return nil
	}))
	h.SetOnIgnorableWhitespace(sax.IgnorableWhitespaceFunc(func(_ context.Context, ch []byte) error {
		events = append(events, "ignorable:"+string(ch))
		return nil
	}))

	xml := "<root>\n  <a/>\n  <b/>\n</root>"
	reader := &parserChunkedReader{
		data:  []byte(xml),
		chunk: 2,
	}

	doc, err := helium.NewParser().SAXHandler(h).ParseReader(t.Context(), reader)
	require.NoError(t, err)
	require.Nil(t, doc)

	for _, event := range events {
		require.False(t, strings.HasPrefix(event, "characters:"), "unexpected character event: %s", event)
	}
	require.Equal(t, []string{
		"ignorable:\n  ",
		"ignorable:\n  ",
		"ignorable:\n",
	}, events)
}
