package html

import "golang.org/x/text/unicode/norm"

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
	// For normal data-state text and raw-text/RCDATA/plaintext it is an
	// approximate soft cap: content is
	// flushed to SAX in chunks targeting this size, but a chunk may slightly
	// exceed it because an indivisible token (a whole multi-byte UTF-8 rune, or
	// a resolved character reference) is never split — a single rune larger than
	// the cap is emitted whole. For comment/bogus-comment/PI it is a hard cap:
	// exceeding it fails the parse with ErrContentSizeExceeded, since those
	// constructs cannot be chunked without corrupting the document. It is also a
	// hard cap for an unresolved named character-reference literal in normal
	// data-state text as well as RCDATA: ANY
	// "&"-prefixed run that does not resolve (whether short, semicolon-terminated,
	// or unbounded) fails the parse with ErrContentSizeExceeded once the literal
	// bytes it would emit ("&" + name + optional ";") exceed the cap. A known
	// (';'-terminated) entity reference is exempt (a resolved character reference,
	// emitted intact within a fixed lookahead window, never charged). A no-';'
	// LEGACY resolution — a full legacy entity (e.g. "&amp") OR a legacy-PREFIX
	// match (e.g. "&ampZ", the "amp" prefix resolving with "Z" echoed) — is exempt
	// only when its whole consumed run ("&" + name) fits the cap; over the cap it
	// hard-fails with ErrContentSizeExceeded and emits NOTHING, uniformly across
	// the short within-lookahead path and the saturated ambiguous path. Under
	// noBlanks (StripBlanks) the soft cap on normal data-state text also has a
	// hard-fail case: a run is suppressed only when entirely whitespace, so the
	// scanner cannot flush a run whose leading whitespace prefix reaches the cap
	// with more whitespace beyond it without buffering unbounded to decide
	// significance — such a run fails with ErrContentSizeExceeded. It also bounds
	// the undecided-encoding deferred prefix: an undeclared-charset ParseReader or
	// push stream that keeps proving valid UTF-8 buffers undecided only up to this
	// cap, and an over-cap undecided-encoding stream is rejected with
	// ErrContentSizeExceeded rather than committing to a Latin-1/UTF-8
	// interpretation. Zero
	// selects defaultMaxContentSize. It
	// guards against unbounded memory growth on a gigantic or unterminated section.
	maxContentSize int
}

// defaultMaxContentSize is the default content cap, used when maxContentSize is
// 0. Normal data-state text and raw-text/RCDATA/plaintext content is delivered
// to SAX in chunks targeting
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

// scanTokenLimit bounds the unbounded PeekAt scans for indivisible tag-level
// tokens (tag/attribute names and intra-tag whitespace runs). These are not
// chunkable content, and MaxContentSize is a content-chunking granularity knob
// that callers legitimately set very small (e.g. 1) for fine-grained streaming;
// binding tag names to it would reject ordinary multi-character names like
// "script". The cap is therefore floored at defaultMaxContentSize and only grows
// when the caller raises MaxContentSize above it, so it never rejects realistic
// markup while still preventing an unbounded-PeekAt buffering DoS.
func (c parseConfig) scanTokenLimit() int {
	if c.maxContentSize > defaultMaxContentSize {
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

// CharacterMap installs a character map: each mapped rune appearing in text or
// attribute-value content is replaced by its literal replacement string, emitted
// verbatim (not re-escaped), per XSLT/XQuery Serialization 3.1 §7 (character maps
// apply to the html output method). A nil or empty map disables the feature,
// leaving output byte-identical.
func (w Writer) CharacterMap(m map[rune]string) Writer {
	w.charMap = m
	return w
}

// Normalization requests Unicode normalization of text-node and attribute-value
// character content (the normalization-form serialization parameter). form is one
// of "NFC", "NFD", "NFKC", "NFKD"; "", "none", or any other value disables it,
// leaving output byte-identical. Normalization is scoped to text and attribute
// nodes (Serialization 3.1 §4 character-expansion phase) — element/attribute
// names, comments, PIs, and the DOCTYPE are never normalized. A character map is
// expected to carry normalization-inert replacements (fn:serialize substitutes
// sentinel runes), so a mapped character's replacement is not normalized.
func (w Writer) Normalization(form string) Writer {
	w.normForm, w.normalize = htmlNormalizationForm(form)
	return w
}

// NullNamespaceHTMLOnly controls whether an element is recognized as an HTML
// void element (serialized with no closing tag) only when it is in no
// namespace. When true, an otherwise-void element (e.g. <meta>) in a non-null
// namespace such as XHTML is serialized with an explicit end tag. This matches
// HTML 4.01 serialization, where only no-namespace elements are HTML elements;
// HTML5 (the default, v=false) recognizes HTML void elements by local name
// regardless of namespace.
func (w Writer) NullNamespaceHTMLOnly(v bool) Writer {
	w.nullNamespaceHTMLOnly = v
	return w
}

// dumpConfig holds configuration for HTML serialization.
type dumpConfig struct {
	noDefaultDTD          bool
	noFormat              bool
	preserveCase          bool
	noEscapeURIAttributes bool
	escapeControlChars    bool
	nullNamespaceHTMLOnly bool
	// charMap substitutes a mapped rune in text and attribute-value content with
	// its literal replacement string (Serialization 3.1 character maps).
	// Empty/nil disables the feature.
	charMap map[rune]string
	// normalize / normForm request Unicode normalization of text-node and
	// attribute-value character content (the normalization-form serialization
	// parameter, Serialization 3.1 §4). Scoped to text and attribute nodes; false
	// by default, keeping output byte-identical.
	normalize bool
	normForm  norm.Form
}
