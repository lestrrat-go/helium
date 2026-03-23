package xslt3

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/lestrrat-go/helium"
)

func parseStylesheetDocument(ctx context.Context, data []byte, baseURI string) (*helium.Document, error) {
	p := helium.NewParser()
	p.SetOption(helium.ParseDTDLoad | helium.ParseNoEnt)
	if baseURI != "" {
		p.SetBaseURI(baseURI)
	}
	return p.Parse(ctx, data)
}

// CompileStylesheet compiles a parsed XSLT stylesheet document into a
// reusable Stylesheet. Use WithCompileBaseURI and WithCompileURIResolver
// to configure compilation via ctx.
//
// Deprecated: use NewCompiler().Compile(ctx, doc) instead.
func CompileStylesheet(ctx context.Context, doc *helium.Document) (*Stylesheet, error) {
	cfg := deriveCompileConfig(ctx)
	return compile(doc, cfg)
}

// CompileFile parses and compiles an XSLT stylesheet from a file path.
// This is a convenience function that parses the file and delegates to
// NewCompiler().Compile.
func CompileFile(ctx context.Context, path string) (*Stylesheet, error) {
	absPath, absErr := filepath.Abs(path)
	if absErr != nil {
		absPath = path
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	doc, err := parseStylesheetDocument(ctx, data, absPath)
	if err != nil {
		return nil, err
	}
	cfg := deriveCompileConfig(ctx)
	if cfg.baseURI == "" {
		cfg.baseURI = absPath
	}
	return compile(doc, cfg)
}

// Transform applies the compiled stylesheet to the source document and
// returns the result document. Use WithParameter, WithInitialTemplate,
// and WithMessageHandler to configure the transformation via ctx.
//
// Deprecated: use ss.Transform(source).Do(ctx) instead.
func Transform(ctx context.Context, source *helium.Document, ss *Stylesheet) (*helium.Document, error) {
	cfg := getTransformConfig(ctx)
	return executeTransform(ctx, source, ss, cfg)
}

// TransformToWriter applies the compiled stylesheet to the source document
// and writes the serialized result to w.
//
// Deprecated: use ss.Transform(source).WriteTo(ctx, w) instead.
func TransformToWriter(ctx context.Context, source *helium.Document, ss *Stylesheet, w io.Writer) error {
	cfg := getTransformConfig(ctx)
	resultDoc, err := executeTransform(ctx, source, ss, cfg)
	if err != nil {
		return err
	}

	// Get the output definition
	outDef := ss.outputs[""]
	return SerializeResult(w, resultDoc, outDef)
}

// TransformString applies the compiled stylesheet to the source document
// and returns the serialized result as a string.
//
// Deprecated: use ss.Transform(source).Serialize(ctx) instead.
func TransformString(ctx context.Context, source *helium.Document, ss *Stylesheet) (string, error) {
	var buf bytes.Buffer
	if err := TransformToWriter(ctx, source, ss, &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}
