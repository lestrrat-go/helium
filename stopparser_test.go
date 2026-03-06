package helium_test

import (
	"bytes"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/sax"
	"github.com/stretchr/testify/require"
)

func TestStopParserInCharacters(t *testing.T) {
	const input = `<?xml version="1.0"?>
<root>
  <a>hello</a>
  <b>world</b>
</root>`

	s := sax.New()
	s.OnCharacters = sax.CharactersFunc(func(ctx sax.Context, ch []byte) error {
		if string(ch) == "hello" {
			helium.StopParser(ctx)
		}
		return nil
	})

	p := helium.NewParser()
	p.SetSAXHandler(s)

	_, err := p.Parse(t.Context(), []byte(input))
	require.NoError(t, err, "StopParser should not produce an error")
}

func TestStopParserInStartElementNS(t *testing.T) {
	const input = `<?xml version="1.0"?>
<root>
  <a>hello</a>
  <target>stop here</target>
  <c>should not reach</c>
</root>`

	var seen []string
	s := sax.New()
	s.OnStartElementNS = sax.StartElementNSFunc(func(ctx sax.Context, localname, prefix, uri string, namespaces []sax.Namespace, attrs []sax.Attribute) error {
		seen = append(seen, localname)
		if localname == "target" {
			helium.StopParser(ctx)
		}
		return nil
	})

	p := helium.NewParser()
	p.SetSAXHandler(s)

	_, err := p.Parse(t.Context(), []byte(input))
	require.NoError(t, err, "StopParser should not produce an error")
	require.Contains(t, seen, "root")
	require.Contains(t, seen, "a")
	require.Contains(t, seen, "target")
	require.NotContains(t, seen, "c", "elements after stop should not be seen")
}

func TestStopParserViaPushParser(t *testing.T) {
	const input = `<?xml version="1.0"?>
<root>
  <a>hello</a>
  <b>world</b>
</root>`

	s := sax.New()
	s.OnStartElementNS = sax.StartElementNSFunc(func(ctx sax.Context, localname, prefix, uri string, namespaces []sax.Namespace, attrs []sax.Attribute) error {
		if localname == "b" {
			helium.StopParser(ctx)
		}
		return nil
	})

	p := helium.NewParser()
	p.SetSAXHandler(s)
	pp := p.NewPushParser(t.Context())
	require.NoError(t, pp.Push([]byte(input)))
	_, err := pp.Close()
	require.NoError(t, err, "PushParser Close should not produce an error after StopParser")
}

func TestStopParserInStartDocument(t *testing.T) {
	const input = `<?xml version="1.0"?><root><child/></root>`

	s := sax.New()
	s.OnStartDocument = sax.StartDocumentFunc(func(ctx sax.Context) error {
		helium.StopParser(ctx)
		return nil
	})

	p := helium.NewParser()
	p.SetSAXHandler(s)

	_, err := p.Parse(t.Context(), []byte(input))
	require.NoError(t, err, "StopParser in StartDocument should not produce an error")
}

func TestStopParserReturnsPartialDoc(t *testing.T) {
	const input = `<?xml version="1.0"?>
<root>
  <a>hello</a>
  <b>world</b>
  <c>end</c>
</root>`

	// Use a tree builder as base, add stop logic on top
	tb := helium.NewTreeBuilder()

	// Wrap the tree builder so it builds the tree, but stop at <b>
	wrapper := sax.New()
	wrapper.OnSetDocumentLocator = sax.SetDocumentLocatorFunc(func(ctx sax.Context, loc sax.DocumentLocator) error {
		return tb.SetDocumentLocator(ctx, loc)
	})
	wrapper.OnStartDocument = sax.StartDocumentFunc(func(ctx sax.Context) error {
		return tb.StartDocument(ctx)
	})
	wrapper.OnEndDocument = sax.EndDocumentFunc(func(ctx sax.Context) error {
		return tb.EndDocument(ctx)
	})
	wrapper.OnStartElementNS = sax.StartElementNSFunc(func(ctx sax.Context, localname, prefix, uri string, namespaces []sax.Namespace, attrs []sax.Attribute) error {
		if localname == "b" {
			helium.StopParser(ctx)
			return nil
		}
		return tb.StartElementNS(ctx, localname, prefix, uri, namespaces, attrs)
	})
	wrapper.OnEndElementNS = sax.EndElementNSFunc(func(ctx sax.Context, localname, prefix, uri string) error {
		return tb.EndElementNS(ctx, localname, prefix, uri)
	})
	wrapper.OnCharacters = sax.CharactersFunc(func(ctx sax.Context, ch []byte) error {
		return tb.Characters(ctx, ch)
	})
	wrapper.OnIgnorableWhitespace = sax.IgnorableWhitespaceFunc(func(ctx sax.Context, ch []byte) error {
		return tb.IgnorableWhitespace(ctx, ch)
	})
	wrapper.OnComment = sax.CommentFunc(func(ctx sax.Context, value []byte) error {
		return tb.Comment(ctx, value)
	})
	wrapper.OnProcessingInstruction = sax.ProcessingInstructionFunc(func(ctx sax.Context, target, data string) error {
		return tb.ProcessingInstruction(ctx, target, data)
	})
	wrapper.OnCDataBlock = sax.CDataBlockFunc(func(ctx sax.Context, value []byte) error {
		return tb.CDataBlock(ctx, value)
	})
	wrapper.OnReference = sax.ReferenceFunc(func(ctx sax.Context, name string) error {
		return tb.Reference(ctx, name)
	})

	p := helium.NewParser()
	p.SetSAXHandler(wrapper)

	doc, err := p.Parse(t.Context(), []byte(input))
	require.NoError(t, err)
	require.NotNil(t, doc)

	// The partial doc should have <root> with <a> but not <b> or <c>
	var buf bytes.Buffer
	d := helium.NewWriter()
	require.NoError(t, d.WriteDoc(&buf, doc))
	out := buf.String()
	require.Contains(t, out, "<a>")
	require.Contains(t, out, "hello")
	require.NotContains(t, out, "<b>")
	require.NotContains(t, out, "<c>")
}

func TestStopParserWithNilContext(t *testing.T) {
	// StopParser with a non-ParserStopper context should be a no-op
	helium.StopParser(nil)
	helium.StopParser("not a parser stopper")
}

func TestStopParserViaParseReader(t *testing.T) {
	const input = `<?xml version="1.0"?>
<root>
  <a>hello</a>
  <b>world</b>
</root>`

	s := sax.New()
	s.OnStartElementNS = sax.StartElementNSFunc(func(ctx sax.Context, localname, prefix, uri string, namespaces []sax.Namespace, attrs []sax.Attribute) error {
		if localname == "b" {
			helium.StopParser(ctx)
		}
		return nil
	})

	p := helium.NewParser()
	p.SetSAXHandler(s)

	_, err := p.ParseReader(t.Context(), bytes.NewReader([]byte(input)))
	require.NoError(t, err, "StopParser should not produce an error via ParseReader")
}
