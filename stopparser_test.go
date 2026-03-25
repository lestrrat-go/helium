package helium_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/sax"
	"github.com/stretchr/testify/require"
)

func newStopParserEntityHandler(seen *[]string, resolve sax.ResolveEntityFunc) sax.SAX2Handler {
	tb := helium.NewTreeBuilder()
	wrapper := sax.New()

	wrapper.SetOnSetDocumentLocator(sax.SetDocumentLocatorFunc(func(ctx context.Context, loc sax.DocumentLocator) error {
		return tb.SetDocumentLocator(ctx, loc)
	}))
	wrapper.SetOnStartDocument(sax.StartDocumentFunc(func(ctx context.Context) error {
		return tb.StartDocument(ctx)
	}))
	wrapper.SetOnEndDocument(sax.EndDocumentFunc(func(ctx context.Context) error {
		return tb.EndDocument(ctx)
	}))
	wrapper.SetOnInternalSubset(sax.InternalSubsetFunc(func(ctx context.Context, name, externalID, systemID string) error {
		return tb.InternalSubset(ctx, name, externalID, systemID)
	}))
	wrapper.SetOnExternalSubset(sax.ExternalSubsetFunc(func(ctx context.Context, name, externalID, systemID string) error {
		return tb.ExternalSubset(ctx, name, externalID, systemID)
	}))
	wrapper.SetOnEntityDecl(sax.EntityDeclFunc(func(ctx context.Context, name string, typ enum.EntityType, publicID, systemID, notation string) error {
		return tb.EntityDecl(ctx, name, typ, publicID, systemID, notation)
	}))
	wrapper.SetOnGetEntity(sax.GetEntityFunc(func(ctx context.Context, name string) (sax.Entity, error) {
		return tb.GetEntity(ctx, name)
	}))
	wrapper.SetOnGetParameterEntity(sax.GetParameterEntityFunc(func(ctx context.Context, name string) (sax.Entity, error) {
		return tb.GetParameterEntity(ctx, name)
	}))
	wrapper.SetOnResolveEntity(sax.ResolveEntityFunc(func(ctx context.Context, publicID, systemID string) (sax.ParseInput, error) {
		if resolve != nil {
			return resolve(ctx, publicID, systemID)
		}
		return tb.ResolveEntity(ctx, publicID, systemID)
	}))
	wrapper.SetOnStartElementNS(sax.StartElementNSFunc(func(ctx context.Context, localname, prefix, uri string, namespaces []sax.Namespace, attrs []sax.Attribute) error {
		*seen = append(*seen, localname)
		if err := tb.StartElementNS(ctx, localname, prefix, uri, namespaces, attrs); err != nil {
			return err
		}
		if localname == "stop" {
			helium.StopParser(ctx)
			return nil
		}
		return nil
	}))
	wrapper.SetOnEndElementNS(sax.EndElementNSFunc(func(ctx context.Context, localname, prefix, uri string) error {
		return tb.EndElementNS(ctx, localname, prefix, uri)
	}))
	wrapper.SetOnCharacters(sax.CharactersFunc(func(ctx context.Context, ch []byte) error {
		return tb.Characters(ctx, ch)
	}))
	wrapper.SetOnIgnorableWhitespace(sax.IgnorableWhitespaceFunc(func(ctx context.Context, ch []byte) error {
		return tb.IgnorableWhitespace(ctx, ch)
	}))
	wrapper.SetOnComment(sax.CommentFunc(func(ctx context.Context, value []byte) error {
		return tb.Comment(ctx, value)
	}))
	wrapper.SetOnProcessingInstruction(sax.ProcessingInstructionFunc(func(ctx context.Context, target, data string) error {
		return tb.ProcessingInstruction(ctx, target, data)
	}))
	wrapper.SetOnCDataBlock(sax.CDataBlockFunc(func(ctx context.Context, value []byte) error {
		return tb.CDataBlock(ctx, value)
	}))
	wrapper.SetOnReference(sax.ReferenceFunc(func(ctx context.Context, name string) error {
		return tb.Reference(ctx, name)
	}))

	return wrapper
}

func TestStopParserInCharacters(t *testing.T) {
	const input = `<?xml version="1.0"?>
<root>
  <a>hello</a>
  <b>world</b>
</root>`

	s := sax.New()
	s.SetOnCharacters(sax.CharactersFunc(func(ctx context.Context, ch []byte) error {
		if string(ch) == "hello" {
			helium.StopParser(ctx)
		}
		return nil
	}))

	p := helium.NewParser().SAXHandler(s)

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
	s.SetOnStartElementNS(sax.StartElementNSFunc(func(ctx context.Context, localname, prefix, uri string, namespaces []sax.Namespace, attrs []sax.Attribute) error {
		seen = append(seen, localname)
		if localname == "target" {
			helium.StopParser(ctx)
		}
		return nil
	}))

	p := helium.NewParser().SAXHandler(s)

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
	s.SetOnStartElementNS(sax.StartElementNSFunc(func(ctx context.Context, localname, prefix, uri string, namespaces []sax.Namespace, attrs []sax.Attribute) error {
		if localname == "b" {
			helium.StopParser(ctx)
		}
		return nil
	}))

	p := helium.NewParser().SAXHandler(s)
	pp := p.NewPushParser(t.Context())
	require.NoError(t, pp.Push([]byte(input)))
	_, err := pp.Close()
	require.NoError(t, err, "PushParser Close should not produce an error after StopParser")
}

func TestStopParserInStartDocument(t *testing.T) {
	const input = `<?xml version="1.0"?><root><child/></root>`

	s := sax.New()
	s.SetOnStartDocument(sax.StartDocumentFunc(func(ctx context.Context) error {
		helium.StopParser(ctx)
		return nil
	}))

	p := helium.NewParser().SAXHandler(s)

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
	wrapper.SetOnSetDocumentLocator(sax.SetDocumentLocatorFunc(func(ctx context.Context, loc sax.DocumentLocator) error {
		return tb.SetDocumentLocator(ctx, loc)
	}))
	wrapper.SetOnStartDocument(sax.StartDocumentFunc(func(ctx context.Context) error {
		return tb.StartDocument(ctx)
	}))
	wrapper.SetOnEndDocument(sax.EndDocumentFunc(func(ctx context.Context) error {
		return tb.EndDocument(ctx)
	}))
	wrapper.SetOnStartElementNS(sax.StartElementNSFunc(func(ctx context.Context, localname, prefix, uri string, namespaces []sax.Namespace, attrs []sax.Attribute) error {
		if localname == "b" {
			helium.StopParser(ctx)
			return nil
		}
		return tb.StartElementNS(ctx, localname, prefix, uri, namespaces, attrs)
	}))
	wrapper.SetOnEndElementNS(sax.EndElementNSFunc(func(ctx context.Context, localname, prefix, uri string) error {
		return tb.EndElementNS(ctx, localname, prefix, uri)
	}))
	wrapper.SetOnCharacters(sax.CharactersFunc(func(ctx context.Context, ch []byte) error {
		return tb.Characters(ctx, ch)
	}))
	wrapper.SetOnIgnorableWhitespace(sax.IgnorableWhitespaceFunc(func(ctx context.Context, ch []byte) error {
		return tb.IgnorableWhitespace(ctx, ch)
	}))
	wrapper.SetOnComment(sax.CommentFunc(func(ctx context.Context, value []byte) error {
		return tb.Comment(ctx, value)
	}))
	wrapper.SetOnProcessingInstruction(sax.ProcessingInstructionFunc(func(ctx context.Context, target, data string) error {
		return tb.ProcessingInstruction(ctx, target, data)
	}))
	wrapper.SetOnCDataBlock(sax.CDataBlockFunc(func(ctx context.Context, value []byte) error {
		return tb.CDataBlock(ctx, value)
	}))
	wrapper.SetOnReference(sax.ReferenceFunc(func(ctx context.Context, name string) error {
		return tb.Reference(ctx, name)
	}))

	p := helium.NewParser().SAXHandler(wrapper)

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
	// StopParser with a nil context should be a no-op and must not panic.
	helium.StopParser(nil) //nolint:staticcheck // intentional: verifying nil-context guard in StopParser
	// StopParser with a context that has no stopper should be a no-op.
	helium.StopParser(context.Background())
}

func TestStopParserViaParseReader(t *testing.T) {
	const input = `<?xml version="1.0"?>
<root>
  <a>hello</a>
  <b>world</b>
</root>`

	s := sax.New()
	s.SetOnStartElementNS(sax.StartElementNSFunc(func(ctx context.Context, localname, prefix, uri string, namespaces []sax.Namespace, attrs []sax.Attribute) error {
		if localname == "b" {
			helium.StopParser(ctx)
		}
		return nil
	}))

	p := helium.NewParser().SAXHandler(s)

	_, err := p.ParseReader(t.Context(), bytes.NewReader([]byte(input)))
	require.NoError(t, err, "StopParser should not produce an error via ParseReader")
}

func TestStopParserInExpandedInternalEntity(t *testing.T) {
	const input = `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ENTITY inner "<stop/><entity-after/>">
]>
<root>&inner;<after/></root>`

	var seen []string
	s := newStopParserEntityHandler(&seen, nil)

	p := helium.NewParser().SAXHandler(s).NoEnt(true)

	_, err := p.Parse(t.Context(), []byte(input))
	require.NoError(t, err, "StopParser should work while expanding internal parsed entities")
	require.Contains(t, seen, "root")
	require.Contains(t, seen, "stop")
	require.NotContains(t, seen, "entity-after")
	require.NotContains(t, seen, "after")
}

func TestStopParserInExpandedExternalEntity(t *testing.T) {
	const input = `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ENTITY ext SYSTEM "ext.xml">
]>
<root>&ext;<after/></root>`

	var seen []string
	s := newStopParserEntityHandler(&seen, sax.ResolveEntityFunc(func(_ context.Context, publicID, systemID string) (sax.ParseInput, error) {
		if systemID == "ext.xml" {
			return newStringParseInput("<stop/><entity-after/>", systemID), nil
		}
		return nil, sax.ErrHandlerUnspecified
	}))

	p := helium.NewParser().SAXHandler(s).NoEnt(true)

	_, err := p.Parse(t.Context(), []byte(input))
	require.NoError(t, err, "StopParser should work while expanding external parsed entities")
	require.Contains(t, seen, "root")
	require.Contains(t, seen, "stop")
	require.NotContains(t, seen, "entity-after")
	require.NotContains(t, seen, "after")
}
