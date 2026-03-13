// Package relaxng implements RELAX NG (XML syntax) schema validation.
package relaxng

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"

	helium "github.com/lestrrat-go/helium"
)

// Compile compiles a RELAX NG document into a Grammar.
// (libxml2: xmlRelaxNGNewParserCtxt + xmlRelaxNGParse)
func Compile(ctx context.Context, doc *helium.Document, opts ...CompileOption) (*Grammar, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	cfg := &compileConfig{}
	for _, o := range opts {
		o(cfg)
	}
	grammar, err := compileSchema(ctx, doc, "", cfg)
	if cfg.errorHandler != nil {
		if cl, ok := cfg.errorHandler.(io.Closer); ok {
			_ = cl.Close()
		}
	}
	return grammar, err
}

// CompileFile reads and compiles a RELAX NG file into a Grammar.
func CompileFile(ctx context.Context, path string, opts ...CompileOption) (*Grammar, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	cfg := &compileConfig{}
	for _, o := range opts {
		o(cfg)
	}

	closeHandler := func() {
		if cfg.errorHandler != nil {
			if cl, ok := cfg.errorHandler.(io.Closer); ok {
				_ = cl.Close()
			}
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		closeHandler()
		return nil, err
	}
	doc, err := helium.Parse(ctx, data)
	if err != nil {
		var pe helium.ErrParseError
		if errors.As(err, &pe) {
			filename := cfg.filename
			if filename == "" {
				filename = path
			}
			errs := formatXMLParseError(filename, pe)
			errs += rngParserError("xmlRelaxNGParse: could not load " + filename)
			if cfg.errorHandler != nil {
				cfg.errorHandler.Handle(ctx, helium.NewLeveledError(errs, helium.ErrorLevelFatal))
			}
			closeHandler()
			return &Grammar{}, nil
		}
		closeHandler()
		return nil, err
	}
	doc.SetURL(path)
	baseDir := filepath.Dir(path)
	grammar, compileErr := compileSchema(ctx, doc, baseDir, cfg)
	closeHandler()
	return grammar, compileErr
}

// ValidateError holds detailed validation failure output.
type ValidateError struct {
	Output string // libxml2-compatible validation output
}

func (e *ValidateError) Error() string {
	return e.Output
}

// Validate validates a document against a compiled grammar.
// It returns nil if the document is valid, or a *ValidateError with details.
// (libxml2: xmlRelaxNGValidateDoc)
func Validate(doc *helium.Document, grammar *Grammar, opts ...ValidateOption) error {
	cfg := &validateConfig{}
	for _, o := range opts {
		o(cfg)
	}
	output, valid := validateDocument(doc, grammar, cfg)
	if valid {
		return nil
	}
	return &ValidateError{Output: output}
}
