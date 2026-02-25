package xmlschema

// CompileOption configures schema compilation.
type CompileOption func(*compileConfig)

type compileConfig struct{}

// ValidateOption configures schema validation.
type ValidateOption func(*validateConfig)

type validateConfig struct {
	filename string
}

// WithFilename sets the filename used in error messages.
func WithFilename(name string) ValidateOption {
	return func(c *validateConfig) {
		c.filename = name
	}
}
