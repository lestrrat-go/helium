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

// Compiler compiles RELAX NG documents into Grammars.
type Compiler struct {
	cfg *compilerCfg
}

// NewCompiler creates a new Compiler with default settings.
func NewCompiler() Compiler {
	return Compiler{cfg: &compilerCfg{}}
}

func (c Compiler) clone() Compiler {
	if c.cfg == nil {
		return Compiler{cfg: &compilerCfg{}}
	}
	cp := *c.cfg
	return Compiler{cfg: &cp}
}

// SchemaFilename sets the RNG filename used in schema compilation error messages.
func (c Compiler) SchemaFilename(name string) Compiler {
	c = c.clone()
	c.cfg.filename = name
	return c
}

// ErrorHandler sets a handler that receives schema compilation errors.
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

// Compile compiles a RELAX NG document into a Grammar.
// (libxml2: xmlRelaxNGNewParserCtxt + xmlRelaxNGParse)
func (c Compiler) Compile(ctx context.Context, doc *helium.Document) (*Grammar, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	cfg := c.cfg
	if cfg == nil {
		cfg = &compilerCfg{}
	}
	grammar, err := compileSchema(ctx, doc, "", cfg)
	c.closeHandler()
	return grammar, err
}

// CompileFile reads and compiles a RELAX NG file into a Grammar.
func (c Compiler) CompileFile(ctx context.Context, path string) (*Grammar, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	cfg := c.cfg
	if cfg == nil {
		cfg = &compilerCfg{}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		c.closeHandler()
		return nil, err
	}
	doc, err := helium.NewParser().Parse(ctx, data)
	if err != nil {
		if pe, ok := errors.AsType[helium.ErrParseError](err); ok {
			filename := cfg.filename
			if filename == "" {
				filename = path
			}
			errs := formatXMLParseError(filename, pe)
			errs += rngParserError("xmlRelaxNGParse: could not load " + filename)
			if cfg.errorHandler != nil {
				cfg.errorHandler.Handle(ctx, helium.NewLeveledError(errs, helium.ErrorLevelFatal))
			}
			c.closeHandler()
			return &Grammar{}, nil
		}
		c.closeHandler()
		return nil, err
	}
	doc.SetURL(path)
	baseDir := filepath.Dir(path)
	grammar, compileErr := compileSchema(ctx, doc, baseDir, cfg)
	c.closeHandler()
	return grammar, compileErr
}

// Validator validates documents against a compiled Grammar.
type Validator struct {
	cfg     *validatorCfg
	grammar *Grammar
}

// NewValidator creates a new Validator for the given grammar.
func NewValidator(grammar *Grammar) Validator {
	return Validator{cfg: &validatorCfg{}, grammar: grammar}
}

func (v Validator) clone() Validator {
	if v.cfg == nil {
		return Validator{cfg: &validatorCfg{}, grammar: v.grammar}
	}
	cp := *v.cfg
	return Validator{cfg: &cp, grammar: v.grammar}
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

// ValidateError holds detailed validation failure output.
type ValidateError struct {
	Output string // libxml2-compatible validation output
}

func (e *ValidateError) Error() string {
	return e.Output
}

// Validate validates a document against the compiled grammar.
// It returns nil if the document is valid, or a *ValidateError with details.
// (libxml2: xmlRelaxNGValidateDoc)
func (v Validator) Validate(ctx context.Context, doc *helium.Document) error {
	if ctx == nil {
		ctx = context.Background()
	}
	cfg := v.cfg
	if cfg == nil {
		cfg = &validatorCfg{}
	}
	output, valid := validateDocument(ctx, doc, v.grammar, cfg)
	if valid {
		return nil
	}
	return &ValidateError{Output: output}
}
