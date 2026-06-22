package helium_test

import (
	"context"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// nopCatalog is a CatalogResolver that never resolves anything. It exists only
// to drive the Parser.Catalog configuration path.
type nopCatalog struct{}

func (nopCatalog) Resolve(_ context.Context, _, _ string) string { return "" }
func (nopCatalog) ResolveURI(_ context.Context, _ string) string { return "" }

// TestParserOptionSetters exercises every boolean parser option setter with both
// true and false (so both the Set and Clear branches run) plus the scalar/object
// setters, then performs a parse to confirm the configured parser still works.
func TestParserOptionSetters(t *testing.T) {
	t.Parallel()

	p := helium.NewParser().
		RecoverOnError(true).RecoverOnError(false).
		SubstituteEntities(true).SubstituteEntities(false).
		LoadExternalDTD(true).LoadExternalDTD(false).
		DefaultDTDAttributes(true).DefaultDTDAttributes(false).
		ValidateDTD(true).ValidateDTD(false).
		SuppressErrors(true).SuppressErrors(false).
		SuppressWarnings(true).SuppressWarnings(false).
		PedanticErrors(true).PedanticErrors(false).
		StripBlanks(true).StripBlanks(false).
		ProcessXInclude(true).ProcessXInclude(false).
		AllowNetwork(true).AllowNetwork(false).
		CleanNamespaces(true).CleanNamespaces(false).
		MergeCDATA(true).MergeCDATA(false).
		XIncludeNodes(true).XIncludeNodes(false).
		CompactTextNodes(true).CompactTextNodes(false).
		FixBaseURIs(true).FixBaseURIs(false).
		RelaxLimits(true).RelaxLimits(false).
		IgnoreEncoding(true).IgnoreEncoding(false).
		BigLineNumbers(true).BigLineNumbers(false).
		BlockXXE(true).BlockXXE(false).
		ReuseDict(true).ReuseDict(false).
		SkipIDs(true).SkipIDs(false).
		LenientXMLDecl(true).LenientXMLDecl(false).
		CharBufferSize(8192).
		MaxDepth(256).
		MaxExternalDTDBytes(1 << 20).
		Catalog(nopCatalog{}).
		BaseURI("http://example.com/base.xml")

	doc, err := p.Parse(t.Context(), []byte(`<?xml version="1.0"?><root><child>text</child></root>`))
	require.NoError(t, err, "a fully-configured parser parses a simple document")
	require.NotNil(t, doc.DocumentElement())
}

// TestParserCharBufferSizeAffectsParse confirms a tiny char buffer (which forces
// repeated cursor refills) still parses a larger document correctly.
func TestParserCharBufferSizeAffectsParse(t *testing.T) {
	t.Parallel()

	var b []byte
	b = append(b, []byte(`<root>`)...)
	for range 200 {
		b = append(b, []byte(`<item>x</item>`)...)
	}
	b = append(b, []byte(`</root>`)...)

	doc, err := helium.NewParser().CharBufferSize(16).Parse(t.Context(), b)
	require.NoError(t, err)
	require.Equal(t, "root", doc.DocumentElement().Name())
}
