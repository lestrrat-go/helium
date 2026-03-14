package xslt3

import (
	"bytes"
	"context"
	"io"
	"os"

	"github.com/lestrrat-go/helium"
)

// CompileStylesheet compiles a parsed XSLT stylesheet document into a
// reusable Stylesheet.
func CompileStylesheet(doc *helium.Document, opts ...CompileOption) (*Stylesheet, error) {
	cfg := &compileConfig{}
	for _, o := range opts {
		o(cfg)
	}
	return compile(doc, cfg)
}

// CompileFile parses and compiles an XSLT stylesheet from a file path.
func CompileFile(path string, opts ...CompileOption) (*Stylesheet, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	doc, err := helium.Parse(context.Background(), data)
	if err != nil {
		return nil, err
	}
	// Set baseURI from file path if not already set
	cfg := &compileConfig{}
	for _, o := range opts {
		o(cfg)
	}
	if cfg.baseURI == "" {
		cfg.baseURI = path
	}
	return compile(doc, cfg)
}

// Transform applies the compiled stylesheet to the source document and
// returns the result document. Use WithParameter, WithInitialTemplate,
// and WithMessageHandler to configure the transformation via ctx.
func Transform(ctx context.Context, source *helium.Document, ss *Stylesheet) (*helium.Document, error) {
	cfg := getTransformConfig(ctx)
	return executeTransform(ctx, source, ss, cfg)
}

// TransformToWriter applies the compiled stylesheet to the source document
// and writes the serialized result to w.
func TransformToWriter(ctx context.Context, source *helium.Document, ss *Stylesheet, w io.Writer) error {
	cfg := getTransformConfig(ctx)
	resultDoc, err := executeTransform(ctx, source, ss, cfg)
	if err != nil {
		return err
	}

	// Get the output definition
	outDef := ss.outputs[""]
	return serializeResult(w, resultDoc, outDef)
}

// TransformString applies the compiled stylesheet to the source document
// and returns the serialized result as a string.
func TransformString(ctx context.Context, source *helium.Document, ss *Stylesheet) (string, error) {
	var buf bytes.Buffer
	if err := TransformToWriter(ctx, source, ss, &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}
