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
	filename string
}

// WithFilename sets the filename used in validation error messages.
func WithFilename(name string) ValidateOption {
	return func(c *validateConfig) {
		c.filename = name
	}
}
