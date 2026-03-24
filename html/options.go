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
	noDefaultDTD          bool
	noFormat              bool
	preserveCase          bool
	noEscapeURIAttributes bool
	escapeControlChars    bool
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

// WithNoFormat suppresses formatting whitespace (newlines) in HTML output.
func WithNoFormat() WriteOption {
	return func(c *dumpConfig) {
		c.noFormat = true
	}
}

// WithPreserveCase preserves the original case of element and attribute names
// instead of lowercasing them. Used by XSLT HTML output method.
func WithPreserveCase() WriteOption {
	return func(c *dumpConfig) {
		c.preserveCase = true
	}
}

// WithNoEscapeURIAttributes disables percent-encoding of non-ASCII characters
// in URI attributes (href, src, action, etc.). Corresponds to
// escape-uri-attributes="no" in the XSLT serialization spec.
func WithNoEscapeURIAttributes() WriteOption {
	return func(c *dumpConfig) {
		c.noEscapeURIAttributes = true
	}
}

// WithEscapeControlChars causes characters in the U+007F-U+009F range to be
// emitted as numeric character references (e.g. &#x9F;). HTML5 serialization
// requires this instead of raising SERE0014.
func WithEscapeControlChars() WriteOption {
	return func(c *dumpConfig) {
		c.escapeControlChars = true
	}
}
