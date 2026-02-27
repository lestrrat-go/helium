package helium

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const testXML = `<?xml version="1.0"?>
<root foo="bar">
  <!-- a comment -->
  <child>hello</child>
  <ns:item xmlns:ns="urn:test">world</ns:item>
</root>`

func dumpDoc(t *testing.T, doc *Document) string {
	t.Helper()
	var buf bytes.Buffer
	var d Dumper
	require.NoError(t, d.DumpDoc(&buf, doc))
	return buf.String()
}

func TestPushParserSingleChunk(t *testing.T) {
	input := []byte(testXML)

	want, err := Parse(input)
	require.NoError(t, err)

	p := NewParser()
	pp := p.NewPushParser()
	require.NoError(t, pp.Push(input))
	got, err := pp.Close()
	require.NoError(t, err)
	require.Equal(t, dumpDoc(t, want), dumpDoc(t, got))
}

func TestPushParserMultiChunk(t *testing.T) {
	input := []byte(testXML)

	want, err := Parse(input)
	require.NoError(t, err)

	// Split at various positions: mid-tag, mid-attribute, mid-text
	splits := []int{5, 15, 30, 50, 80}
	p := NewParser()
	pp := p.NewPushParser()

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
}

func TestPushParserByteAtATime(t *testing.T) {
	input := []byte(testXML)

	want, err := Parse(input)
	require.NoError(t, err)

	p := NewParser()
	pp := p.NewPushParser()
	for i := 0; i < len(input); i++ {
		require.NoError(t, pp.Push(input[i:i+1]))
	}

	got, err := pp.Close()
	require.NoError(t, err)
	require.Equal(t, dumpDoc(t, want), dumpDoc(t, got))
}

func TestPushParserSAXEvents(t *testing.T) {
	input := []byte(testXML)

	// Capture SAX events from regular parse
	var wantBuf bytes.Buffer
	wantHandler := newEventEmitter(&wantBuf)
	p1 := NewParser()
	p1.SetSAXHandler(wantHandler)
	_, err := p1.Parse(input)
	require.NoError(t, err)

	// Capture SAX events from push parse
	var gotBuf bytes.Buffer
	gotHandler := newEventEmitter(&gotBuf)
	p2 := NewParser()
	p2.SetSAXHandler(gotHandler)
	pp := p2.NewPushParser()
	require.NoError(t, pp.Push(input))
	_, err = pp.Close()
	require.NoError(t, err)

	require.Equal(t, wantBuf.String(), gotBuf.String())
}

func TestPushParserMalformedXML(t *testing.T) {
	input := []byte(`<?xml version="1.0"?><root><child>text</chld></root>`)

	p := NewParser()
	pp := p.NewPushParser()
	require.NoError(t, pp.Push(input))
	_, err := pp.Close()
	require.Error(t, err)
}

func TestPushParserPushAfterError(t *testing.T) {
	malformed := []byte(`<?xml version="1.0"?><root><bad`)

	p := NewParser()
	pp := p.NewPushParser()
	require.NoError(t, pp.Push(malformed))

	// Close the stream to trigger EOF in the parser, which should fail
	pp.stream.Close()
	<-pp.done // wait for parser to finish

	// Now pushing should return an error
	err := pp.Push([]byte(`more data`))
	require.Error(t, err)
}

func TestPushParserEmptyInput(t *testing.T) {
	p := NewParser()
	pp := p.NewPushParser()
	_, err := pp.Close()
	require.Error(t, err, "empty input should produce an error")
}

func TestPushParserIOCopy(t *testing.T) {
	input := []byte(testXML)

	want, err := Parse(input)
	require.NoError(t, err)

	p := NewParser()
	pp := p.NewPushParser()
	n, err := io.Copy(pp, bytes.NewReader(input))
	require.NoError(t, err)
	require.Equal(t, int64(len(input)), n)

	got, err := pp.Close()
	require.NoError(t, err)
	require.Equal(t, dumpDoc(t, want), dumpDoc(t, got))
}

func TestPushParserCloseIdempotent(t *testing.T) {
	input := []byte(testXML)

	p := NewParser()
	pp := p.NewPushParser()
	require.NoError(t, pp.Push(input))

	doc1, err1 := pp.Close()
	doc2, err2 := pp.Close()

	require.Equal(t, err1, err2)
	require.Equal(t, doc1, doc2)
}

func TestPushParserWithDTD(t *testing.T) {
	input := []byte(`<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (child)+>
  <!ELEMENT child (#PCDATA)>
]>
<doc><child>hello</child></doc>`)

	want, err := Parse(input)
	require.NoError(t, err)

	p := NewParser()
	pp := p.NewPushParser()
	require.NoError(t, pp.Push(input))

	got, err := pp.Close()
	require.NoError(t, err)
	require.Equal(t, dumpDoc(t, want), dumpDoc(t, got))
}

func TestPushParserWithNamespaces(t *testing.T) {
	input := []byte(`<?xml version="1.0"?>
<root xmlns="urn:default" xmlns:x="urn:x">
  <x:child x:attr="val">text</x:child>
</root>`)

	want, err := Parse(input)
	require.NoError(t, err)

	p := NewParser()
	pp := p.NewPushParser()
	require.NoError(t, pp.Push(input))

	got, err := pp.Close()
	require.NoError(t, err)
	require.Equal(t, dumpDoc(t, want), dumpDoc(t, got))
}

func TestPushParserRealFile(t *testing.T) {
	const path = "testdata/libxml2/source/example/gjobs.xml"
	input, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("test file not available: %v", err)
	}

	want, err := Parse(input)
	require.NoError(t, err)

	// Push in 64-byte chunks
	p := NewParser()
	pp := p.NewPushParser()
	for i := 0; i < len(input); i += 64 {
		end := i + 64
		if end > len(input) {
			end = len(input)
		}
		require.NoError(t, pp.Push(input[i:end]))
	}

	got, err := pp.Close()
	require.NoError(t, err)
	require.Equal(t, dumpDoc(t, want), dumpDoc(t, got))
}
