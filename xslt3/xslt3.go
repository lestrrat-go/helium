package xslt3

import (
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

// CompileStylesheet compiles a parsed XSLT stylesheet document.
// This is a convenience wrapper over NewCompiler().Compile(ctx, doc).
func CompileStylesheet(ctx context.Context, doc *helium.Document) (*Stylesheet, error) {
	return NewCompiler().Compile(ctx, doc)
}

// CompileFile parses and compiles an XSLT stylesheet from a file path.
// This is a convenience wrapper that parses the file, sets the base URI,
// and delegates to NewCompiler().Compile.
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
	return NewCompiler().BaseURI(absPath).Compile(ctx, doc)
}

// Transform applies the compiled stylesheet to the source document.
// This is a convenience wrapper over ss.Transform(source).Do(ctx).
func Transform(ctx context.Context, source *helium.Document, ss *Stylesheet) (*helium.Document, error) {
	if ss == nil {
		return nil, errNilStylesheet
	}
	return ss.Transform(source).Do(ctx)
}

// TransformString applies the compiled stylesheet and returns the serialized result.
// This is a convenience wrapper over ss.Transform(source).Serialize(ctx).
func TransformString(ctx context.Context, source *helium.Document, ss *Stylesheet) (string, error) {
	if ss == nil {
		return "", errNilStylesheet
	}
	return ss.Transform(source).Serialize(ctx)
}

// TransformToWriter applies the compiled stylesheet and writes the result to w.
// This is a convenience wrapper over ss.Transform(source).WriteTo(ctx, w).
func TransformToWriter(ctx context.Context, source *helium.Document, ss *Stylesheet, w io.Writer) error {
	if ss == nil {
		return errNilStylesheet
	}
	return ss.Transform(source).WriteTo(ctx, w)
}
