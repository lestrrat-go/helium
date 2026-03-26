package html

// parseConfig holds configuration for the HTML parser.
type parseConfig struct {
	noImplied bool
	noBlanks  bool
	noError   bool
	noWarning bool
}

// Writer configures HTML serialization. It is a value-style wrapper:
// fluent methods return updated copies and the original is never mutated.
type Writer struct {
	dumpConfig
}

// NewWriter creates a new HTML Writer with default settings.
func NewWriter() Writer {
	return Writer{}
}

// DefaultDTD controls whether a default DOCTYPE is emitted when the document
// has no DTD.
func (w Writer) DefaultDTD(v bool) Writer {
	w.noDefaultDTD = !v
	return w
}

// Format controls whether formatting whitespace is emitted in HTML output.
func (w Writer) Format(v bool) Writer {
	w.noFormat = !v
	return w
}

// PreserveCase controls whether the original case of element and attribute
// names is preserved instead of lowercasing them.
func (w Writer) PreserveCase(v bool) Writer {
	w.preserveCase = v
	return w
}

// EscapeURIAttributes controls whether non-ASCII characters in URI attributes
// are percent-encoded.
func (w Writer) EscapeURIAttributes(v bool) Writer {
	w.noEscapeURIAttributes = !v
	return w
}

// EscapeControlChars controls whether characters in the U+007F-U+009F range
// are emitted as numeric character references.
func (w Writer) EscapeControlChars(v bool) Writer {
	w.escapeControlChars = v
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
