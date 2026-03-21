package xsd

import helium "github.com/lestrrat-go/helium"

// CompileOption configures schema compilation.
type CompileOption func(*compileConfig)

type compileConfig struct {
	filename     string // XSD filename for error messages
	baseDir      string // base directory for resolving relative includes
	errorHandler helium.ErrorHandler
}

// WithSchemaFilename sets the XSD filename used in schema compilation error messages.
func WithSchemaFilename(name string) CompileOption {
	return func(c *compileConfig) {
		c.filename = name
	}
}

// WithBaseDir sets the base directory used to resolve relative paths in
// xs:include and xs:redefine elements during schema compilation.
func WithBaseDir(dir string) CompileOption {
	return func(c *compileConfig) {
		c.baseDir = dir
	}
}

// WithCompileErrorHandler sets the error handler for schema compilation.
func WithCompileErrorHandler(h helium.ErrorHandler) CompileOption {
	return func(c *compileConfig) {
		c.errorHandler = h
	}
}

// ValidateOption configures schema validation.
type ValidateOption func(*validateConfig)

type validateConfig struct {
	filename     string
	errorHandler helium.ErrorHandler
	annotations  *TypeAnnotations
}

// WithFilename sets the filename used in error messages.
func WithFilename(name string) ValidateOption {
	return func(c *validateConfig) {
		c.filename = name
	}
}

// WithValidateErrorHandler sets the error handler for schema validation.
func WithValidateErrorHandler(h helium.ErrorHandler) ValidateOption {
	return func(c *validateConfig) {
		c.errorHandler = h
	}
}

// TypeAnnotations maps document nodes to their XSD type names.
// Type names use the "xs:localName" format for built-in types and
// "Q{ns}localName" for user-defined types.
type TypeAnnotations map[helium.Node]string

// WithAnnotations enables collection of per-node type annotations during
// validation. The caller must provide a non-nil pointer to a TypeAnnotations
// value; the map is populated during validation.
func WithAnnotations(ann *TypeAnnotations) ValidateOption {
	return func(c *validateConfig) {
		c.annotations = ann
	}
}
