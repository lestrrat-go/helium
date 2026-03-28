package helium_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/sax"
	"github.com/lestrrat-go/pdebug"
	"github.com/stretchr/testify/require"
)

func newEventEmitter(out io.Writer) sax.SAX2Handler {
	entities := map[string]*helium.Entity{}
	s := sax.New()
	s.SetOnSetDocumentLocator(sax.SetDocumentLocatorFunc(func(_ context.Context, loc sax.DocumentLocator) error {
		_, _ = fmt.Fprintf(out, "SAX.SetDocumentLocator()\n")
		return nil
	}))
	s.SetOnAttributeDecl(sax.AttributeDeclFunc(func(_ context.Context, elemName string, attrName string, typ enum.AttributeType, deftype enum.AttributeDefault, defvalue string, enum sax.Enumeration) error {
		// eek, defvalue is an interface, and interface == nil is only true
		// if the interface has no value AND not type, so.. hmmm.
		if defvalue == "" {
			defvalue = "NULL"
		}
		_, _ = fmt.Fprintf(out, "SAX.AttributeDecl(%s, %s, %d, %d, %s, ...)\n", elemName, attrName, typ, deftype, defvalue)
		return nil
	}))
	s.SetOnInternalSubset(sax.InternalSubsetFunc(func(_ context.Context, name, externalID, systemID string) error {
		_, _ = fmt.Fprintf(out, "SAX.InternalSubset(%s, %s, %s)\n", name, externalID, systemID)
		return nil
	}))
	s.SetOnReference(sax.ReferenceFunc(func(_ context.Context, name string) error {
		_, _ = fmt.Fprintf(out, "SAX.Reference(%s)\n", name)
		return nil
	}))
	s.SetOnGetEntity(sax.GetEntityFunc(func(_ context.Context, name string) (sax.Entity, error) {
		_, _ = fmt.Fprintf(out, "SAX.ResolveEntity(%s)\n", name)

		ent, ok := entities[name]
		if !ok {
			return nil, errors.New("entity not found")
		}
		return ent, nil
	}))

	s.SetOnGetParameterEntity(sax.GetParameterEntityFunc(func(_ context.Context, name string) (sax.Entity, error) {
		_, _ = fmt.Fprintf(out, "SAX.ResolveEntity(%s)\n", name)

		ent, ok := entities[name]
		if !ok {
			return nil, errors.New("entity not found")
		}
		return ent, nil
	}))

	s.SetOnEntityDecl(sax.EntityDeclFunc(func(ctxif context.Context, name string, typ enum.EntityType, publicID string, systemID string, notation string) error {
		if pdebug.Enabled {
			g := pdebug.Marker("EntityDecl handler for sax_test.go")
			defer g.End()
		}
		_, _ = fmt.Fprintf(out, "SAX.UnparsedEntityDecl(%s, %d, %s, %s, %s)\n",
			name, typ, publicID, systemID, notation)

		doc := helium.NewDefaultDocument()
		dtd, err := doc.CreateInternalSubset("root", "", "")
		if err != nil {
			return err
		}
		if typ == enum.ExternalGeneralUnparsedEntity {
			if _, err := dtd.AddNotation(notation, "", ""); err != nil {
				return err
			}
		}
		ent, err := dtd.AddEntity(name, typ, publicID, systemID, notation)
		if err != nil {
			return err
		}
		entities[name] = ent
		if pdebug.Enabled {
			pdebug.Printf("registered entity '%s' (entity type = '%s', publicID = '%s', systemID = '%s', notation = '%s')", name, typ, publicID, systemID, notation)
		}
		return nil
	}))
	s.SetOnExternalSubset(sax.ExternalSubsetFunc(func(_ context.Context, name, externalID, systemID string) error {
		_, _ = fmt.Fprintf(out, "SAX.ExternalSubset(%s, %s, %s)\n", name, externalID, systemID)
		return nil
	}))
	s.SetOnElementDecl(sax.ElementDeclFunc(func(_ context.Context, name string, typ enum.ElementType, content sax.ElementContent) error {
		_, _ = fmt.Fprintf(out, "SAX.ElementDecl(%s, %d, ...)\n", name, typ)
		return nil
	}))
	s.SetOnStartDocument(sax.StartDocumentFunc(func(_ context.Context) error {
		_, _ = fmt.Fprintf(out, "SAX.StartDocument()\n")
		return nil
	}))
	s.SetOnEndDocument(sax.EndDocumentFunc(func(_ context.Context) error {
		_, _ = fmt.Fprintf(out, "SAX.EndDocument()\n")
		return nil
	}))
	s.SetOnComment(sax.CommentFunc(func(_ context.Context, data []byte) error {
		_, _ = fmt.Fprintf(out, "SAX.Comment(%s)\n", data)
		return nil
	}))
	charHandler := func(name string, _ context.Context, data []byte) error { //nolint:unparam // always nil but matches SAX handler signature
		var output string
		if len(data) > 30 {
			output = string(data[:30])
		} else {
			output = string(data)
		}

		_, _ = fmt.Fprintf(out, "SAX.%s(%s, %d)\n", name, output, len(data))
		return nil
	}
	s.SetOnIgnorableWhitespace(sax.IgnorableWhitespaceFunc(func(ctx context.Context, data []byte) error {
		return charHandler("IgnorableWhitespace", ctx, data)
	}))
	s.SetOnCharacters(sax.CharactersFunc(func(ctx context.Context, data []byte) error {
		return charHandler("Characters", ctx, data)
	}))
	s.SetOnStartElementNS(sax.StartElementNSFunc(func(_ context.Context, localname, prefix, uri string, namespaces []sax.Namespace, attrs []sax.Attribute) error {
		_, _ = fmt.Fprintf(out, "SAX.StartElementNS(%s, ", localname)

		if prefix != "" {
			_, _ = fmt.Fprintf(out, "%s, ", prefix)
		} else {
			_, _ = fmt.Fprintf(out, "NULL, ")
		}

		if uri != "" {
			_, _ = fmt.Fprintf(out, "'%s', ", uri)
		} else {
			_, _ = fmt.Fprintf(out, "NULL, ")
		}

		lns := len(namespaces)
		_, _ = fmt.Fprintf(out, "%d, ", lns)
		for _, ns := range namespaces {
			if prefix := ns.Prefix(); prefix != "" {
				_, _ = fmt.Fprintf(out, "xmlns:%s='%s'", ns.Prefix(), ns.URI())
			} else {
				_, _ = fmt.Fprintf(out, "xmlns='%s'", ns.URI())
			}
			_, _ = fmt.Fprintf(out, ", ")
		}

		defaulted := 0
		for _, attr := range attrs {
			if attr.IsDefault() {
				defaulted++
			}
		}

		_, _ = fmt.Fprintf(out, "%d, %d",
			len(attrs),
			defaulted, /* TODO - number of defaulted attributes */
		)

		if len(attrs) > 0 {
			_, _ = fmt.Fprintf(out, ", ")
			for i, attr := range attrs {
				_, _ = fmt.Fprintf(out, "%s='%.4s...', %d", attr.Name(), attr.Value(), len(attr.Value()))
				if i < len(attrs)-1 {
					_, _ = fmt.Fprintf(out, ", ")
				}
			}
		}

		_, _ = fmt.Fprintln(out, ")")

		return nil
	}))
	s.SetOnEndElementNS(sax.EndElementNSFunc(func(_ context.Context, localname, prefix, uri string) error {
		_, _ = fmt.Fprintf(out, "SAX.EndElementNS(%s, ", localname)

		if prefix != "" {
			_, _ = fmt.Fprintf(out, "%s, ", prefix)
		} else {
			_, _ = fmt.Fprintf(out, "NULL, ")
		}

		if uri != "" {
			_, _ = fmt.Fprintf(out, "'%s')\n", uri)
		} else {
			_, _ = fmt.Fprintf(out, "NULL)\n")
		}

		return nil
	}))
	return s
}

func TestDocumentLocatorIDs(t *testing.T) {
	const baseURI = "test://document.xml"
	var gotPublicID, gotSystemID string

	s := sax.New()
	s.SetOnSetDocumentLocator(sax.SetDocumentLocatorFunc(func(_ context.Context, loc sax.DocumentLocator) error {
		gotPublicID = loc.GetPublicID()
		gotSystemID = loc.GetSystemID()
		return nil
	}))
	s.SetOnStartDocument(sax.StartDocumentFunc(func(_ context.Context) error { return nil }))
	s.SetOnEndDocument(sax.EndDocumentFunc(func(_ context.Context) error { return nil }))
	s.SetOnStartElementNS(sax.StartElementNSFunc(func(_ context.Context, _, _, _ string, _ []sax.Namespace, _ []sax.Attribute) error {
		return nil
	}))
	s.SetOnEndElementNS(sax.EndElementNSFunc(func(_ context.Context, _, _, _ string) error { return nil }))
	s.SetOnCharacters(sax.CharactersFunc(func(_ context.Context, _ []byte) error { return nil }))

	p := helium.NewParser().SAXHandler(s).BaseURI(baseURI)

	_, err := p.Parse(t.Context(), []byte(`<root/>`))
	require.NoError(t, err, "Parse should succeed")
	require.Equal(t, "", gotPublicID, "GetPublicID should return empty string")
	require.Equal(t, baseURI, gotSystemID, "GetSystemID should return base URI")
}

func TestSAXEvents(t *testing.T) {
	skipped := map[string]struct{}{}

	dir := "test"
	files, err := os.ReadDir(dir)
	require.NoError(t, err, "os.ReadDir should succeed")

	for _, fi := range files {
		if fi.IsDir() {
			continue
		}

		if _, ok := skipped[fi.Name()]; ok {
			t.Logf("Skipping test for '%s' for now...", fi.Name())
			continue
		}

		fn := filepath.Join(dir, fi.Name())
		if !strings.HasSuffix(fn, ".xml") {
			continue
		}

		goldenfn := strings.ReplaceAll(fn, ".xml", ".sax2")
		if _, err := os.Stat(goldenfn); err != nil {
			continue
		}

		t.Logf("Testing %s...", fn)

		in, err := os.ReadFile(fn)
		require.NoError(t, err, "os.ReadFile should succeed")

		golden, err := os.ReadFile(goldenfn)
		require.NoError(t, err, "os.ReadFile should succeed")

		out := bytes.Buffer{}
		p := helium.NewParser().SAXHandler(newEventEmitter(&out))

		_, err = p.Parse(t.Context(), in)
		if err != nil {
			t.Logf("source XML: %s", in)
		}
		require.NoError(t, err, "Parse should succeed (file = %s)", fn)

		if string(golden) != out.String() {
			errout, err := os.OpenFile(fn+".err", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
			if err != nil {
				t.Logf("Failed to file to save output: %s", err)
				return
			}
			defer func() { _ = errout.Close() }()

			_, _ = errout.Write(out.Bytes())
		}
		require.Equal(t, string(golden), out.String(), "SAX event streams should match (file = %s)", fn)
	}
}

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

func TestStopParser(t *testing.T) {
	t.Parallel()

	t.Run("in Characters", func(t *testing.T) {
		t.Parallel()
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
	})

	t.Run("in StartElementNS", func(t *testing.T) {
		t.Parallel()
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
	})

	t.Run("via PushParser", func(t *testing.T) {
		t.Parallel()
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
	})

	t.Run("in StartDocument", func(t *testing.T) {
		t.Parallel()
		const input = `<?xml version="1.0"?><root><child/></root>`

		s := sax.New()
		s.SetOnStartDocument(sax.StartDocumentFunc(func(ctx context.Context) error {
			helium.StopParser(ctx)
			return nil
		}))

		p := helium.NewParser().SAXHandler(s)

		_, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err, "StopParser in StartDocument should not produce an error")
	})

	t.Run("returns partial doc", func(t *testing.T) {
		t.Parallel()
		const input = `<?xml version="1.0"?>
<root>
  <a>hello</a>
  <b>world</b>
  <c>end</c>
</root>`

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

		var buf bytes.Buffer
		d := helium.NewWriter()
		require.NoError(t, d.WriteTo(&buf, doc))
		out := buf.String()
		require.Contains(t, out, "<a>")
		require.Contains(t, out, "hello")
		require.NotContains(t, out, "<b>")
		require.NotContains(t, out, "<c>")
	})

	t.Run("nil context", func(t *testing.T) {
		t.Parallel()
		helium.StopParser(nil) //nolint:staticcheck // intentional: verifying nil-context guard in StopParser
		helium.StopParser(context.Background())
	})

	t.Run("via ParseReader", func(t *testing.T) {
		t.Parallel()
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
	})

	t.Run("in expanded internal entity", func(t *testing.T) {
		// Not parallel: uses shared `seen` slice via newStopParserEntityHandler
		const input = `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ENTITY inner "<stop/><entity-after/>">
]>
<root>&inner;<after/></root>`

		var seen []string
		s := newStopParserEntityHandler(&seen, nil)

		p := helium.NewParser().SAXHandler(s).SubstituteEntities(true)

		_, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err, "StopParser should work while expanding internal parsed entities")
		require.Contains(t, seen, "root")
		require.Contains(t, seen, "stop")
		require.NotContains(t, seen, "entity-after")
		require.NotContains(t, seen, "after")
	})

	t.Run("in expanded external entity", func(t *testing.T) {
		// Not parallel: uses shared `seen` slice via newStopParserEntityHandler
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

		p := helium.NewParser().SAXHandler(s).SubstituteEntities(true)

		_, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err, "StopParser should work while expanding external parsed entities")
		require.Contains(t, seen, "root")
		require.Contains(t, seen, "stop")
		require.NotContains(t, seen, "entity-after")
		require.NotContains(t, seen, "after")
	})
}
