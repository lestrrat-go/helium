package html

// parseConfig holds configuration for the HTML parser.
type parseConfig struct {
	noImplied bool
	noBlanks  bool
	noError   bool
	noWarning bool
}

// ParseOption configures HTML parsing behavior.
type ParseOption func(*parseConfig)

// WithNoImplied suppresses automatic insertion of implied html/head/body elements.
// (libxml2: HTML_PARSE_NOIMPLIED)
func WithNoImplied() ParseOption {
	return func(c *parseConfig) {
		c.noImplied = true
	}
}

// WithNoBlanks removes whitespace-only text nodes from the DOM.
// (libxml2: HTML_PARSE_NOBLANKS)
func WithNoBlanks() ParseOption {
	return func(c *parseConfig) {
		c.noBlanks = true
	}
}

// WithNoError suppresses error messages from the SAX error handler.
// (libxml2: HTML_PARSE_NOERROR)
func WithNoError() ParseOption {
	return func(c *parseConfig) {
		c.noError = true
	}
}

// WithNoWarning suppresses warning messages from the SAX warning handler.
// (libxml2: HTML_PARSE_NOWARNING)
func WithNoWarning() ParseOption {
	return func(c *parseConfig) {
		c.noWarning = true
	}
}

// dumpConfig holds configuration for HTML serialization.
type dumpConfig struct {
	noDefaultDTD bool
}

// WriteOption configures HTML serialization behavior.
type WriteOption func(*dumpConfig)

// WithNoDefaultDTD suppresses output of a default DOCTYPE when the document
// has no DTD.
func WithNoDefaultDTD() WriteOption {
	return func(c *dumpConfig) {
		c.noDefaultDTD = true
	}
}
