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

// Compiler compiles XSD documents into Schema values.
// It uses clone-on-write semantics: each builder method returns
// a new Compiler sharing the underlying config until mutation.
type Compiler struct {
	cfg *compileConfig
}

// NewCompiler creates a new Compiler with default settings.
func NewCompiler() Compiler {
	return Compiler{cfg: &compileConfig{}}
}

func (c Compiler) clone() Compiler {
	if c.cfg == nil {
		return Compiler{cfg: &compileConfig{}}
	}
	cp := *c.cfg
	return Compiler{cfg: &cp}
}

// SchemaFilename sets the XSD filename used in compilation error messages.
func (c Compiler) SchemaFilename(name string) Compiler {
	c = c.clone()
	c.cfg.filename = name
	return c
}

// BaseDir sets the base directory used to resolve relative paths in
// xs:include and xs:redefine elements during schema compilation.
func (c Compiler) BaseDir(dir string) Compiler {
	c = c.clone()
	c.cfg.baseDir = dir
	return c
}

// ErrorHandler sets a handler that receives compilation errors.
// When set, errors are delivered to the handler instead of being discarded.
func (c Compiler) ErrorHandler(h helium.ErrorHandler) Compiler {
	c = c.clone()
	c.cfg.errorHandler = h
	return c
}

func (c Compiler) closeHandler() {
	if c.cfg != nil && c.cfg.errorHandler != nil {
		if cl, ok := c.cfg.errorHandler.(io.Closer); ok {
			_ = cl.Close()
		}
	}
}

// Compile compiles an XSD document into a Schema.
// (libxml2: xmlSchemaNewParserCtxt + xmlSchemaParse)
func (c Compiler) Compile(ctx context.Context, doc *helium.Document) (*Schema, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	cfg := c.cfg
	if cfg == nil {
		cfg = &compileConfig{}
	}
	schema, err := compileSchema(ctx, doc, cfg.baseDir, cfg)
	c.closeHandler()
	return schema, err
}

// CompileFile reads and compiles an XSD file into a Schema.
func (c Compiler) CompileFile(ctx context.Context, path string) (*Schema, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	data, err := os.ReadFile(path) //nolint:gosec // path is caller-supplied schema file
	if err != nil {
		return nil, fmt.Errorf("xsd: failed to read %q: %w", path, err)
	}
	doc, err := helium.NewParser().Parse(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("xsd: failed to parse %q: %w", path, err)
	}
	cfg := c.cfg
	if cfg == nil {
		cfg = &compileConfig{}
	}
	baseDir := filepath.Dir(path)
	schema, compileErr := compileSchema(ctx, doc, baseDir, cfg)
	c.closeHandler()
	return schema, compileErr
}

// ValidateError holds detailed validation failure output.
type ValidateError struct {
	Output string // libxml2-compatible validation output
}

func (e *ValidateError) Error() string {
	return e.Output
}

// Validator validates documents against a compiled XSD schema.
// It uses clone-on-write semantics: each builder method returns
// a new Validator sharing the underlying config until mutation.
type Validator struct {
	cfg    *validateConfig
	schema *Schema
}

// NewValidator creates a new Validator for the given schema.
func NewValidator(schema *Schema) Validator {
	return Validator{cfg: &validateConfig{}, schema: schema}
}

func (v Validator) clone() Validator {
	if v.cfg == nil {
		return Validator{cfg: &validateConfig{}, schema: v.schema}
	}
	cp := *v.cfg
	return Validator{cfg: &cp, schema: v.schema}
}

// Filename sets the filename used in validation error messages.
func (v Validator) Filename(name string) Validator {
	v = v.clone()
	v.cfg.filename = name
	return v
}

// ErrorHandler sets a handler that receives validation errors.
func (v Validator) ErrorHandler(h helium.ErrorHandler) Validator {
	v = v.clone()
	v.cfg.errorHandler = h
	return v
}

// Annotations enables collection of per-node type annotations during
// validation. The caller must provide a non-nil pointer to a TypeAnnotations
// value; the map is populated during validation.
func (v Validator) Annotations(ann *TypeAnnotations) Validator {
	v = v.clone()
	v.cfg.annotations = ann
	return v
}

// NilledElements enables collection of nilled element information during
// validation. The caller must provide a non-nil pointer to a NilledElements
// value; the map is populated during validation.
func (v Validator) NilledElements(ne *NilledElements) Validator {
	v = v.clone()
	v.cfg.nilledElements = ne
	return v
}

// Validate validates a document against the compiled schema.
// It returns nil if the document is valid, or a *ValidateError with details.
// (libxml2: xmlSchemaValidateDoc)
func (v Validator) Validate(ctx context.Context, doc *helium.Document) error {
	cfg := v.cfg
	if cfg == nil {
		cfg = &validateConfig{}
	}
	output, valid := validateDocument(ctx, doc, v.schema, cfg)
	if valid {
		return nil
	}
	return &ValidateError{Output: output}
}
