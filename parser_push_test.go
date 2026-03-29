package helium_test

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

const testXML = `<?xml version="1.0"?>
<root foo="bar">
  <!-- a comment -->
  <child>hello</child>
  <ns:item xmlns:ns="urn:test">world</ns:item>
</root>`

func dumpDoc(t *testing.T, doc *helium.Document) string {
	t.Helper()
	var buf bytes.Buffer
	d := helium.NewWriter()
	require.NoError(t, d.WriteTo(&buf, doc))
	return buf.String()
}

func TestPushParser(t *testing.T) {
	t.Parallel()

	t.Run("single chunk", func(t *testing.T) {
		t.Parallel()
		input := []byte(testXML)

		p := helium.NewParser()
		want, err := p.Parse(t.Context(), input)
		require.NoError(t, err)

		pp := p.NewPushParser(t.Context())
		require.NoError(t, pp.Push(input))
		got, err := pp.Close()
		require.NoError(t, err)
		require.Equal(t, dumpDoc(t, want), dumpDoc(t, got))
	})

	t.Run("multi chunk", func(t *testing.T) {
		t.Parallel()
		input := []byte(testXML)

		p := helium.NewParser()
		want, err := p.Parse(t.Context(), input)
		require.NoError(t, err)

		// Split at various positions: mid-tag, mid-attribute, mid-text
		splits := []int{5, 15, 30, 50, 80}
		pp := p.NewPushParser(t.Context())

		prev := 0
		for _, pos := range splits {
			if pos > len(input) {
				break
			}
			require.NoError(t, pp.Push(input[prev:pos]))
			prev = pos
		}
		if prev < len(input) {
			require.NoError(t, pp.Push(input[prev:]))
		}

		got, err := pp.Close()
		require.NoError(t, err)
		require.Equal(t, dumpDoc(t, want), dumpDoc(t, got))
	})

	t.Run("byte at a time", func(t *testing.T) {
		t.Parallel()
		input := []byte(testXML)

		p := helium.NewParser()
		want, err := p.Parse(t.Context(), input)
		require.NoError(t, err)
		pp := p.NewPushParser(t.Context())
		for i := range input {
			require.NoError(t, pp.Push(input[i:i+1]))
		}

		got, err := pp.Close()
		require.NoError(t, err)
		require.Equal(t, dumpDoc(t, want), dumpDoc(t, got))
	})

	t.Run("SAX events", func(t *testing.T) {
		t.Parallel()
		input := []byte(testXML)

		// Capture SAX events from regular parse
		var wantBuf bytes.Buffer
		wantHandler := newEventEmitter(&wantBuf)
		p1 := helium.NewParser().SAXHandler(wantHandler)
		_, err := p1.Parse(t.Context(), input)
		require.NoError(t, err)

		// Capture SAX events from push parse
		var gotBuf bytes.Buffer
		gotHandler := newEventEmitter(&gotBuf)
		p2 := helium.NewParser().SAXHandler(gotHandler)
		pp := p2.NewPushParser(t.Context())
		require.NoError(t, pp.Push(input))
		_, err = pp.Close()
		require.NoError(t, err)

		require.Equal(t, wantBuf.String(), gotBuf.String())
	})

	t.Run("malformed XML", func(t *testing.T) {
		t.Parallel()
		input := []byte(`<?xml version="1.0"?><root><child>text</chld></root>`)

		p := helium.NewParser()
		pp := p.NewPushParser(t.Context())
		require.NoError(t, pp.Push(input))
		_, err := pp.Close()
		require.Error(t, err)
	})

	t.Run("push after error", func(t *testing.T) {
		t.Parallel()
		malformed := []byte(`<?xml version="1.0"?><root><bad`)

		p := helium.NewParser()
		pp := p.NewPushParser(t.Context())
		require.NoError(t, pp.Push(malformed))

		_, err := pp.Close()
		require.Error(t, err)

		// Now pushing should return an error
		err = pp.Push([]byte(`more data`))
		require.Error(t, err)
	})

	t.Run("empty input", func(t *testing.T) {
		t.Parallel()
		p := helium.NewParser()
		pp := p.NewPushParser(t.Context())
		_, err := pp.Close()
		require.Error(t, err, "empty input should produce an error")
	})

	t.Run("io.Copy", func(t *testing.T) {
		t.Parallel()
		input := []byte(testXML)

		p := helium.NewParser()
		want, err := p.Parse(t.Context(), input)
		require.NoError(t, err)
		pp := p.NewPushParser(t.Context())
		n, err := io.Copy(pp, bytes.NewReader(input))
		require.NoError(t, err)
		require.Equal(t, int64(len(input)), n)

		got, err := pp.Close()
		require.NoError(t, err)
		require.Equal(t, dumpDoc(t, want), dumpDoc(t, got))
	})

	t.Run("close idempotent", func(t *testing.T) {
		t.Parallel()
		input := []byte(testXML)

		p := helium.NewParser()
		pp := p.NewPushParser(t.Context())
		require.NoError(t, pp.Push(input))

		doc1, err1 := pp.Close()
		doc2, err2 := pp.Close()

		require.Equal(t, err1, err2)
		require.Equal(t, doc1, doc2)
	})

	t.Run("with DTD", func(t *testing.T) {
		t.Parallel()
		input := []byte(`<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (child)+>
  <!ELEMENT child (#PCDATA)>
]>
<doc><child>hello</child></doc>`)

		p := helium.NewParser()
		want, err := p.Parse(t.Context(), input)
		require.NoError(t, err)
		pp := p.NewPushParser(t.Context())
		require.NoError(t, pp.Push(input))

		got, err := pp.Close()
		require.NoError(t, err)
		require.Equal(t, dumpDoc(t, want), dumpDoc(t, got))
	})

	t.Run("with namespaces", func(t *testing.T) {
		t.Parallel()
		input := []byte(`<?xml version="1.0"?>
<root xmlns="urn:default" xmlns:x="urn:x">
  <x:child x:attr="val">text</x:child>
</root>`)

		p := helium.NewParser()
		want, err := p.Parse(t.Context(), input)
		require.NoError(t, err)
		pp := p.NewPushParser(t.Context())
		require.NoError(t, pp.Push(input))

		got, err := pp.Close()
		require.NoError(t, err)
		require.Equal(t, dumpDoc(t, want), dumpDoc(t, got))
	})

	t.Run("real file", func(t *testing.T) {
		t.Parallel()
		const path = "testdata/libxml2/source/example/gjobs.xml"
		input, err := os.ReadFile(path)
		if err != nil {
			t.Skipf("test file not available: %v", err)
		}

		p := helium.NewParser()
		want, err := p.Parse(t.Context(), input)
		require.NoError(t, err)

		// Push in 64-byte chunks
		pp := p.NewPushParser(t.Context())
		for i := 0; i < len(input); i += 64 {
			end := min(i+64, len(input))
			require.NoError(t, pp.Push(input[i:end]))
		}

		got, err := pp.Close()
		require.NoError(t, err)
		require.Equal(t, dumpDoc(t, want), dumpDoc(t, got))
	})
}
