package html

// parseConfig holds configuration for the HTML parser.
type parseConfig struct {
	noImplied bool
	noBlanks  bool
	noError   bool
	noWarning bool
	// strict promotes warnings forwarded from silenced SAX callbacks into a
	// fatal parse error. Default false preserves libxml2-style tolerance.
	strict bool
	// maxContentSize bounds, in bytes, the size of a single content section.
	// For raw-text/RCDATA/plaintext it is an approximate soft cap: content is
	// flushed to SAX in chunks targeting this size, but a chunk may slightly
	// exceed it because an indivisible token (a whole multi-byte UTF-8 rune, or
	// a resolved character reference) is never split — a single rune larger than
	// the cap is emitted whole. For comment/bogus-comment/PI it is a hard cap:
	// exceeding it fails the parse with ErrContentSizeExceeded, since those
	// constructs cannot be chunked without corrupting the document. It is also a
	// hard cap for an unresolved named character-reference run in RCDATA: an
	// "&"-prefixed alphanumeric run that never resolves and never terminates
	// fails the parse with ErrContentSizeExceeded once its literal bytes exceed
	// the cap. Zero selects defaultMaxContentSize. It guards against unbounded
	// memory growth on a gigantic or unterminated section.
	maxContentSize int
}

// defaultMaxContentSize is the default content cap, used when maxContentSize is
// 0. Raw-text/RCDATA/plaintext content is delivered to SAX in chunks targeting
// this size (an indivisible token may push a chunk slightly over); comments/PIs
// exceeding it fail the parse. Either way a section with gigabytes of data (or
// one that never terminates) is bounded in memory.
const defaultMaxContentSize = 16 << 20 // 16 MiB

// contentLimit returns the effective content cap.
func (c parseConfig) contentLimit() int {
	if c.maxContentSize > 0 {
		return c.maxContentSize
	}
	return defaultMaxContentSize
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
