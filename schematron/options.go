package schematron

// CompileOption configures schema compilation.
type CompileOption func(*compileConfig)

type compileConfig struct {
	filename string
}

// WithSchemaFilename sets the schema filename used in compilation error messages.
func WithSchemaFilename(name string) CompileOption {
	return func(c *compileConfig) {
		c.filename = name
	}
}

// ValidateOption configures schema validation.
type ValidateOption func(*validateConfig)

type validateConfig struct {
	filename     string
	quiet        bool
	errorHandler ErrorHandler
}

// WithFilename sets the filename used in validation error messages.
func WithFilename(name string) ValidateOption {
	return func(c *validateConfig) {
		c.filename = name
	}
}

// WithQuiet suppresses all per-error output in the returned string.
// Only the final "validates" / "fails to validate" line is emitted.
// This corresponds to libxml2's XML_SCHEMATRON_OUT_QUIET flag.
// If an ErrorHandler is also set, errors are still delivered to the handler.
func WithQuiet() ValidateOption {
	return func(c *validateConfig) {
		c.quiet = true
	}
}

// WithErrorHandler sets a handler that receives each validation error.
// When set, per-error messages are routed to the handler instead of
// being accumulated in the returned string. This corresponds to
// libxml2's XML_SCHEMATRON_OUT_ERROR flag.
func WithErrorHandler(h ErrorHandler) ValidateOption {
	return func(c *validateConfig) {
		c.errorHandler = h
	}
}

// ErrorHandler receives individual schematron validation errors.
type ErrorHandler interface {
	HandleError(e ValidationError)
}

// ErrorHandlerFunc is a function adapter for ErrorHandler.
type ErrorHandlerFunc func(ValidationError)

// HandleError implements ErrorHandler.
func (f ErrorHandlerFunc) HandleError(e ValidationError) { f(e) }

// ValidationError represents a single schematron validation error.
type ValidationError struct {
	Filename string
	Line     int
	Element  string
	Path     string
	Message  string
}
