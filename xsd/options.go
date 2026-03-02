package xsd

import helium "github.com/lestrrat-go/helium"

// CompileOption configures schema compilation.
type CompileOption func(*compileConfig)

type compileConfig struct {
	filename     string               // XSD filename for error messages
	errorHandler helium.ErrorHandler
}

// WithSchemaFilename sets the XSD filename used in schema compilation error messages.
func WithSchemaFilename(name string) CompileOption {
	return func(c *compileConfig) {
		c.filename = name
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
