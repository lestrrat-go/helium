package relaxng

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"

	helium "github.com/lestrrat-go/helium"
)

type compileConfig struct {
	label        string // label for error messages (e.g. source filename)
	baseDir      string
	errorHandler helium.ErrorHandler
}

type validateConfig struct {
	label        string
	errorHandler helium.ErrorHandler
}

// ErrValidationFailed is returned by [Validator.Validate] when the document
// does not conform to the schema. Individual validation errors are delivered
// to the configured [helium.ErrorHandler].
var ErrValidationFailed = errors.New("relaxng: validation failed")

// Compiler compiles RELAX NG documents into Grammars.
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

// Label sets the label (typically a filename) used in schema compilation error messages.
func (c Compiler) Label(name string) Compiler {
	c = c.clone()
	c.cfg.label = name
	return c
}

// BaseDir sets the base directory used to resolve relative paths in
// include and externalRef elements during schema compilation.
func (c Compiler) BaseDir(dir string) Compiler {
	c = c.clone()
	c.cfg.baseDir = dir
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
		cfg = &compileConfig{}
	}
	grammar, err := compileSchema(ctx, doc, cfg.baseDir, cfg)
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
		cfg = &compileConfig{}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		c.closeHandler()
		return nil, err
	}
	doc, err := helium.NewParser().Parse(ctx, data)
	if err != nil {
		if pe, ok := errors.AsType[helium.ErrParseError](err); ok {
			filename := cfg.label
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
	cfg     *validateConfig
	grammar *Grammar
}

// NewValidator creates a new Validator for the given grammar.
func NewValidator(grammar *Grammar) Validator {
	return Validator{cfg: &validateConfig{}, grammar: grammar}
}

func (v Validator) clone() Validator {
	if v.cfg == nil {
		return Validator{cfg: &validateConfig{}, grammar: v.grammar}
	}
	cp := *v.cfg
	return Validator{cfg: &cp, grammar: v.grammar}
}

// Label sets the label (typically a filename) used in validation error messages.
func (v Validator) Label(name string) Validator {
	v = v.clone()
	v.cfg.label = name
	return v
}

// ErrorHandler sets a handler that receives validation errors.
func (v Validator) ErrorHandler(h helium.ErrorHandler) Validator {
	v = v.clone()
	v.cfg.errorHandler = h
	return v
}

func (v Validator) closeHandler() {
	if v.cfg != nil && v.cfg.errorHandler != nil {
		if cl, ok := v.cfg.errorHandler.(io.Closer); ok {
			_ = cl.Close()
		}
	}
}

// Validate validates a document against the compiled grammar.
// It returns nil if the document is valid, or [ErrValidationFailed].
// Individual validation errors are delivered to the configured [helium.ErrorHandler].
// (libxml2: xmlRelaxNGValidateDoc)
func (v Validator) Validate(ctx context.Context, doc *helium.Document) error {
	if ctx == nil {
		ctx = context.Background()
	}
	cfg := v.cfg
	if cfg == nil {
		cfg = &validateConfig{}
	}

	handler := cfg.errorHandler
	if handler == nil {
		handler = helium.NilErrorHandler{}
	}

	valid := validateDocument(ctx, doc, v.grammar, cfg, handler)
	v.closeHandler()
	if valid {
		return nil
	}
	return ErrValidationFailed
}
