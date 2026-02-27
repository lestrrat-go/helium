package relaxng

// CompileOption configures schema compilation.
type CompileOption func(*compileConfig)

type compileConfig struct {
	filename string // RNG filename for error messages
}

// WithSchemaFilename sets the RNG filename used in schema compilation error messages.
func WithSchemaFilename(name string) CompileOption {
	return func(c *compileConfig) {
		c.filename = name
	}
}

// ValidateOption configures validation.
type ValidateOption func(*validateConfig)

type validateConfig struct {
	filename string
}

// WithFilename sets the filename used in validation error messages.
func WithFilename(name string) ValidateOption {
	return func(c *validateConfig) {
		c.filename = name
	}
}
