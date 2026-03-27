package bench_test

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/sax"
)

type parseMode struct {
	name   string
	parser func() helium.Parser
}

type experimentCorpus struct {
	name string
	data []byte
}

var (
	syntheticCorpus []experimentCorpus
	synthOnce       sync.Once
)

func loadSyntheticCorpus() {
	synthOnce.Do(func() {
		syntheticCorpus = []experimentCorpus{
			{name: "tiny-setup-1KB", data: syntheticTinySetupDoc()},
			{name: "attr-flat-55KB", data: syntheticAttrHeavyDoc()},
			{name: "ns-flat-40KB", data: syntheticNamespaceHeavyDoc()},
			{name: "text-dense-256KB", data: syntheticTextDenseDoc()},
			{name: "deep-tree-11KB", data: syntheticDeepTreeDoc()},
		}
	})
}

func benchmarkHeliumCorpus(b *testing.B, corpora []experimentCorpus) {
	modes := []parseMode{
		{
			name: "dom",
			parser: func() helium.Parser {
				return helium.NewParser()
			},
		},
		{
			name: "dom-skip-ids",
			parser: func() helium.Parser {
				return helium.NewParser().SkipIDs(true)
			},
		},
		{
			name: "sax-noop",
			parser: func() helium.Parser {
				return helium.NewParser().SAXHandler(newCountingSAXHandler())
			},
		},
	}

	for _, mode := range modes {
		mode := mode
		b.Run(mode.name, func(b *testing.B) {
			for _, tc := range corpora {
				tc := tc
				b.Run(tc.name, func(b *testing.B) {
					parser := mode.parser()
					data := tc.data
					b.SetBytes(int64(len(data)))
					b.ReportAllocs()
					b.ResetTimer()
					for range b.N {
						doc, err := parser.Parse(context.Background(), data)
						if err != nil {
							b.Fatal(err)
						}
						if doc != nil {
							doc.Free()
						}
					}
				})
			}
		})
	}
}

func BenchmarkHeliumParseModesRealWorld(b *testing.B) {
	loadCorpus(b)
	realWorld := make([]experimentCorpus, 0, len(corpus))
	for _, tc := range corpus {
		realWorld = append(realWorld, experimentCorpus{
			name: tc.name,
			data: *tc.data,
		})
	}
	benchmarkHeliumCorpus(b, realWorld)
}

func BenchmarkHeliumParseModesSynthetic(b *testing.B) {
	loadSyntheticCorpus()
	benchmarkHeliumCorpus(b, syntheticCorpus)
}

type countingSAXHandler struct {
	elements int
	attrs    int
	text     int
}

func newCountingSAXHandler() sax.SAX2Handler {
	h := &countingSAXHandler{}
	s := sax.New()
	s.SetOnStartDocument(sax.StartDocumentFunc(func(context.Context) error { return nil }))
	s.SetOnEndDocument(sax.EndDocumentFunc(func(context.Context) error { return nil }))
	s.SetOnSetDocumentLocator(sax.SetDocumentLocatorFunc(func(context.Context, sax.DocumentLocator) error { return nil }))
	s.SetOnStartElementNS(sax.StartElementNSFunc(func(_ context.Context, _ string, _ string, _ string, _ []sax.Namespace, attrs []sax.Attribute) error {
		h.elements++
		h.attrs += len(attrs)
		return nil
	}))
	s.SetOnEndElementNS(sax.EndElementNSFunc(func(context.Context, string, string, string) error { return nil }))
	s.SetOnCharacters(sax.CharactersFunc(func(_ context.Context, ch []byte) error {
		h.text += len(ch)
		return nil
	}))
	s.SetOnIgnorableWhitespace(sax.IgnorableWhitespaceFunc(func(_ context.Context, ch []byte) error {
		h.text += len(ch)
		return nil
	}))
	s.SetOnCDataBlock(sax.CDataBlockFunc(func(_ context.Context, ch []byte) error {
		h.text += len(ch)
		return nil
	}))
	s.SetOnComment(sax.CommentFunc(func(context.Context, []byte) error { return nil }))
	s.SetOnProcessingInstruction(sax.ProcessingInstructionFunc(func(context.Context, string, string) error { return nil }))
	s.SetOnReference(sax.ReferenceFunc(func(context.Context, string) error { return nil }))
	s.SetOnGetEntity(sax.GetEntityFunc(func(context.Context, string) (sax.Entity, error) { return nil, nil }))
	s.SetOnGetParameterEntity(sax.GetParameterEntityFunc(func(context.Context, string) (sax.Entity, error) { return nil, nil }))
	s.SetOnHasExternalSubset(sax.HasExternalSubsetFunc(func(context.Context) (bool, error) { return false, nil }))
	s.SetOnHasInternalSubset(sax.HasInternalSubsetFunc(func(context.Context) (bool, error) { return false, nil }))
	s.SetOnIsStandalone(sax.IsStandaloneFunc(func(context.Context) (bool, error) { return false, nil }))
	s.SetOnResolveEntity(sax.ResolveEntityFunc(func(context.Context, string, string) (sax.ParseInput, error) { return nil, nil }))
	s.SetOnAttributeDecl(sax.AttributeDeclFunc(func(context.Context, string, string, enum.AttributeType, enum.AttributeDefault, string, sax.Enumeration) error {
		return nil
	}))
	s.SetOnElementDecl(sax.ElementDeclFunc(func(context.Context, string, enum.ElementType, sax.ElementContent) error { return nil }))
	s.SetOnEntityDecl(sax.EntityDeclFunc(func(context.Context, string, enum.EntityType, string, string, string) error { return nil }))
	s.SetOnExternalSubset(sax.ExternalSubsetFunc(func(context.Context, string, string, string) error { return nil }))
	s.SetOnInternalSubset(sax.InternalSubsetFunc(func(context.Context, string, string, string) error { return nil }))
	s.SetOnNotationDecl(sax.NotationDeclFunc(func(context.Context, string, string, string) error { return nil }))
	s.SetOnUnparsedEntityDecl(sax.UnparsedEntityDeclFunc(func(context.Context, string, string, string, string) error { return nil }))
	s.SetOnError(sax.ErrorFunc(func(_ context.Context, err error) error { return err }))
	s.SetOnWarning(sax.WarningFunc(func(context.Context, error) error { return nil }))
	return s
}

func syntheticTinySetupDoc() []byte {
	var b strings.Builder
	b.Grow(2048)
	b.WriteString(`<?xml version="1.0"?><root>`)
	for i := 0; i < 48; i++ {
		b.WriteString(`<item id="`)
		b.WriteString(strings.Repeat("x", 8))
		b.WriteString(`" class="tiny">v</item>`)
	}
	b.WriteString(`</root>`)
	return []byte(b.String())
}

func syntheticAttrHeavyDoc() []byte {
	var b strings.Builder
	b.Grow(80 * 1024)
	b.WriteString(`<?xml version="1.0"?><root>`)
	for i := 0; i < 320; i++ {
		b.WriteString(`<item a="alpha" b="beta" c="gamma" d="delta" e="epsilon" f="zeta" g="eta" h="theta"/>`)
	}
	b.WriteString(`</root>`)
	return []byte(b.String())
}

func syntheticNamespaceHeavyDoc() []byte {
	var b strings.Builder
	b.Grow(64 * 1024)
	b.WriteString(`<?xml version="1.0"?><ns0:root xmlns:ns0="urn:root" xmlns:a="urn:a" xmlns:b="urn:b" xmlns:c="urn:c" xmlns:d="urn:d">`)
	for i := 0; i < 240; i++ {
		b.WriteString(`<a:item b:kind="kind" c:code="code" d:flag="true"><b:child c:value="value">text</b:child></a:item>`)
	}
	b.WriteString(`</ns0:root>`)
	return []byte(b.String())
}

func syntheticTextDenseDoc() []byte {
	var b strings.Builder
	payload := strings.Repeat("Helium text payload with repeated content. ", 128)
	b.Grow(300 * 1024)
	b.WriteString(`<?xml version="1.0"?><root>`)
	for i := 0; i < 40; i++ {
		b.WriteString(`<section><p>`)
		b.WriteString(payload)
		b.WriteString(`</p><p>`)
		b.WriteString(payload)
		b.WriteString(`</p></section>`)
	}
	b.WriteString(`</root>`)
	return []byte(b.String())
}

func syntheticDeepTreeDoc() []byte {
	var b bytes.Buffer
	b.Grow(16 * 1024)
	b.WriteString(`<?xml version="1.0"?>`)
	for i := 0; i < 512; i++ {
		b.WriteString(`<n>`)
	}
	b.WriteString(`leaf`)
	for i := 0; i < 512; i++ {
		b.WriteString(`</n>`)
	}
	return b.Bytes()
}
