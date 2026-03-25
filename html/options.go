package html

// parseConfig holds configuration for the HTML parser.
type parseConfig struct {
	noImplied bool
	noBlanks  bool
	noError   bool
	noWarning bool
}

// writerCfg holds configuration for a Writer.
type writerCfg struct {
	dumpConfig
}

// Writer configures HTML serialization. It is a value-style wrapper:
// fluent methods return updated copies and the original is never mutated.
type Writer struct {
	cfg *writerCfg
}

// NewWriter creates a new HTML Writer with default settings.
func NewWriter() Writer {
	return Writer{cfg: &writerCfg{}}
}

func (w Writer) clone() Writer {
	if w.cfg == nil {
		return Writer{cfg: &writerCfg{}}
	}
	cp := *w.cfg
	return Writer{cfg: &cp}
}

// NoDefaultDTD suppresses output of a default DOCTYPE when the document
// has no DTD.
func (w Writer) NoDefaultDTD() Writer {
	w = w.clone()
	w.cfg.noDefaultDTD = true
	return w
}

// NoFormat suppresses formatting whitespace (newlines) in HTML output.
func (w Writer) NoFormat() Writer {
	w = w.clone()
	w.cfg.noFormat = true
	return w
}

// PreserveCase preserves the original case of element and attribute names
// instead of lowercasing them. Used by XSLT HTML output method.
func (w Writer) PreserveCase() Writer {
	w = w.clone()
	w.cfg.preserveCase = true
	return w
}

// NoEscapeURIAttributes disables percent-encoding of non-ASCII characters
// in URI attributes (href, src, action, etc.). Corresponds to
// escape-uri-attributes="no" in the XSLT serialization spec.
func (w Writer) NoEscapeURIAttributes() Writer {
	w = w.clone()
	w.cfg.noEscapeURIAttributes = true
	return w
}

// EscapeControlChars causes characters in the U+007F-U+009F range to be
// emitted as numeric character references (e.g. &#x9F;). HTML5 serialization
// requires this instead of raising SERE0014.
func (w Writer) EscapeControlChars() Writer {
	w = w.clone()
	w.cfg.escapeControlChars = true
	return w
}

// dumpConfig holds configuration for HTML serialization.
type dumpConfig struct {
	noDefaultDTD          bool
	noFormat              bool
	preserveCase          bool
	noEscapeURIAttributes bool
	escapeControlChars    bool
}
