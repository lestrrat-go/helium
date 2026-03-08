// Package xsd implements XML Schema (XSD) validation.
//
// It supports a subset of the W3C XML Schema Definition Language 1.0.
package xsd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	helium "github.com/lestrrat-go/helium"
)

// Compile compiles an XSD document into a Schema.
// (libxml2: xmlSchemaNewParserCtxt + xmlSchemaParse)
func Compile(doc *helium.Document, opts ...CompileOption) (*Schema, error) {
	cfg := &compileConfig{}
	for _, o := range opts {
		o(cfg)
	}
	schema, err := compileSchema(doc, "", cfg)
	if cfg.errorHandler != nil {
		if cl, ok := cfg.errorHandler.(io.Closer); ok {
			_ = cl.Close()
		}
	}
	return schema, err
}

// CompileFile reads and compiles an XSD file into a Schema.
func CompileFile(path string, opts ...CompileOption) (*Schema, error) {
	cfg := &compileConfig{}
	for _, o := range opts {
		o(cfg)
	}
	data, err := os.ReadFile(path) //nolint:gosec // path is caller-supplied schema file
	if err != nil {
		return nil, fmt.Errorf("xsd: failed to read %q: %w", path, err)
	}
	doc, err := helium.Parse(context.Background(), data)
	if err != nil {
		return nil, fmt.Errorf("xsd: failed to parse %q: %w", path, err)
	}
	baseDir := filepath.Dir(path)
	schema, compileErr := compileSchema(doc, baseDir, cfg)
	if cfg.errorHandler != nil {
		if cl, ok := cfg.errorHandler.(io.Closer); ok {
			_ = cl.Close()
		}
	}
	return schema, compileErr
}

// ValidateError holds detailed validation failure output.
type ValidateError struct {
	Output string // libxml2-compatible validation output
}

func (e *ValidateError) Error() string {
	return e.Output
}

// Validate validates a document against a compiled schema.
// It returns nil if the document is valid, or a *ValidateError with details.
// (libxml2: xmlSchemaValidateDoc)
func Validate(ctx context.Context, doc *helium.Document, schema *Schema, opts ...ValidateOption) error {
	cfg := &validateConfig{}
	for _, o := range opts {
		o(cfg)
	}
	output, valid := validateDocument(ctx, doc, schema, cfg)
	if valid {
		return nil
	}
	return &ValidateError{Output: output}
}
