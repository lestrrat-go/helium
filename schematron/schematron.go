// Package schematron implements Schematron validation.
//
// It supports a subset of Schematron matching libxml2's implementation:
// schema, pattern, rule, assert, report, let, name, value-of.
package schematron

import (
	"context"
	"fmt"
	"io"
	"os"

	helium "github.com/lestrrat-go/helium"
)

// Compiler compiles Schematron documents into Schema values.
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
	cp := *c.cfg
	return Compiler{cfg: &cp}
}

// SchemaFilename sets the schema filename used in compilation error messages.
func (c Compiler) SchemaFilename(name string) Compiler {
	c = c.clone()
	c.cfg.filename = name
	return c
}

// ErrorHandler sets a handler that receives compilation errors.
// When set, errors are delivered to the handler instead of being discarded.
func (c Compiler) ErrorHandler(h helium.ErrorHandler) Compiler {
	c = c.clone()
	c.cfg.errorHandler = h
	return c
}

// Compile compiles a Schematron document into a Schema.
// (libxml2: xmlSchematronNewParserCtxt + xmlSchematronParse)
func (c Compiler) Compile(ctx context.Context, doc *helium.Document) (*Schema, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	schema, err := compileSchema(ctx, doc, c.cfg)
	if c.cfg.errorHandler != nil {
		if cl, ok := c.cfg.errorHandler.(io.Closer); ok {
			_ = cl.Close()
		}
	}
	return schema, err
}

// CompileFile reads and compiles a Schematron file into a Schema.
func (c Compiler) CompileFile(ctx context.Context, path string) (*Schema, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	data, err := os.ReadFile(path) //nolint:gosec // path is caller-supplied schema file
	if err != nil {
		return nil, fmt.Errorf("schematron: read file: %w", err)
	}
	doc, err := helium.NewParser().Parse(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("schematron: parse document: %w", err)
	}
	schema, compileErr := compileSchema(ctx, doc, c.cfg)
	if c.cfg.errorHandler != nil {
		if cl, ok := c.cfg.errorHandler.(io.Closer); ok {
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

// Validator validates documents against a compiled Schematron schema.
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
	cp := *v.cfg
	return Validator{cfg: &cp, schema: v.schema}
}

// Filename sets the filename used in validation error messages.
func (v Validator) Filename(name string) Validator {
	v = v.clone()
	v.cfg.filename = name
	return v
}

// Quiet suppresses all per-error output in the returned string.
// Only the final "validates" / "fails to validate" line is emitted.
// This corresponds to libxml2's XML_SCHEMATRON_OUT_QUIET flag.
// If an ErrorHandler is also set, errors are still delivered to the handler.
func (v Validator) Quiet() Validator {
	v = v.clone()
	v.cfg.quiet = true
	return v
}

// ErrorHandler sets a handler that receives each validation error.
// When set, per-error messages are routed to the handler instead of
// being accumulated in the returned string. This corresponds to
// libxml2's XML_SCHEMATRON_OUT_ERROR flag.
func (v Validator) ErrorHandler(h ErrorHandler) Validator {
	v = v.clone()
	v.cfg.errorHandler = h
	return v
}

// Validate validates a document against the compiled schema.
// It returns nil if the document is valid, or a *ValidateError with details.
// (libxml2: xmlSchematronValidateDoc)
func (v Validator) Validate(ctx context.Context, doc *helium.Document) error {
	output, valid := validateDocument(ctx, doc, v.schema, v.cfg)
	if valid {
		return nil
	}
	return &ValidateError{Output: output}
}
