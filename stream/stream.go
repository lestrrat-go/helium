package stream

import (
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/lestrrat-go/helium/internal/encoding"
	"github.com/lestrrat-go/helium/internal/xmlchar"
)

var errNilOutputWriter = errors.New("stream: output writer is nil")

// isValidPITarget reports whether target is a valid XML processing
// instruction target. A PI target is an XML Name, which is an NCName
// optionally containing colons. The reserved "xml" target is rejected by
// StartPI separately (with a dedicated error) before this is reached, so the
// shared predicate's "xml" rejection is harmless here.
func isValidPITarget(target string) bool {
	return xmlchar.IsValidPITarget(target)
}

// writerState tracks what context the writer is currently in.
type writerState int

const (
	stateNone      writerState = iota // initial / after EndDocument
	stateDocument                     // after StartDocument, or after closing top-level constructs
	stateName                         // inside element opening tag (attributes allowed, > not yet written)
	stateAttribute                    // inside attribute value
	stateText                         // inside element body (after >)
	stateComment                      // inside comment body
	statePI                           // after PI target, before content
	statePIText                       // inside PI content
	stateCDATA                        // inside CDATA section
	stateDTD                          // inside DTD (before internal subset)
	stateDTDText                      // inside DTD internal subset ([ has been written)
)

// elementEntry tracks an open element on the stack.
type elementEntry struct {
	name     string // qualified name (prefix:local or local)
	empty    bool   // true until content is written; enables self-close
	hasText  bool   // true if text content was written (disables indent for end tag)
	hasChild bool   // true if child elements were written
}

// nsEntry tracks a namespace declaration for the current element scope.
type nsEntry struct {
	prefix string
	uri    string
}

// nsScope holds namespace declarations for one element level.
type nsScope struct {
	decls   []nsEntry
	emitted int // number of decls already emitted (indices < emitted are done)
}

// Writer writes XML incrementally to an io.Writer.
//
// Writer is not safe for concurrent use by multiple goroutines.
//
// The zero value of Writer is not ready to use because it has no output
// destination. Construct a Writer with NewWriter.
//
// (libxml2: xmlTextWriter)
type Writer struct {
	out           io.Writer
	indent        string // indent string per level; empty = no indentation
	quoteChar     byte   // attribute quote character ('"' or '\'')
	singleByte    [1]byte
	state         writerState
	elemStack     []elementEntry
	nsStack       []nsScope
	stateStack    []writerState // for comment/PI/CDATA nesting
	err           error         // sticky error
	depth         int           // current element nesting depth (for indentation)
	hasOutput     bool          // true after first output has been written
	wroteNL       bool          // true after EndComment/EndPI wrote trailing \n (suppresses writeIndent's \n)
	commentDash   bool          // true if the current comment body ends with '-' (would form '--->' on close)
	piQuestion    bool          // true if the current PI body ends with '?' (would form '?>' across writes)
	cdataBrackets int           // count (0,1,2) of trailing ']' in the current CDATA body, to detect ']]>' across writes
}

// NewWriter creates a Writer that writes to w. Configure the Writer
// with fluent methods (Indent, QuoteChar) before calling action methods.
// (libxml2: xmlNewTextWriterMemory)
func NewWriter(w io.Writer) Writer {
	return Writer{
		out:       w,
		quoteChar: '"',
		state:     stateNone,
	}
}

// Indent returns a copy of the Writer with indentation enabled.
// Each nested level is indented by the given string (e.g. "  " for
// two spaces, "\t" for tab).
func (w Writer) Indent(indent string) Writer {
	w.indent = indent
	return w
}

// QuoteChar returns a copy of the Writer with the attribute value
// quote character set to q. Must be '"' or '\”. Any other value
// is silently ignored. The default is '"'.
func (w Writer) QuoteChar(q byte) Writer {
	if q == '\'' || q == '"' {
		w.quoteChar = q
	}
	return w
}

// Error returns the sticky error, if any. Once an error occurs, all
// subsequent write operations become no-ops. The error is also returned
// by [Writer.Flush].
func (w *Writer) Error() error { return w.err }

type escapeMode int

const (
	escapeNone escapeMode = iota
	escapeText
	escapeAttr
)

func (w *Writer) ensureWritable() bool {
	if w.err != nil {
		return false
	}
	if w.out == nil {
		w.err = errNilOutputWriter
		return false
	}
	return true
}

// writeStr writes a raw string to the underlying writer.
func (w *Writer) writeStr(s string) {
	if !w.ensureWritable() {
		return
	}
	_, w.err = io.WriteString(w.out, s)
}

// writeEscaped writes a string with the specified escaping mode.
func (w *Writer) writeEscaped(s string, escape escapeMode) {
	if !w.ensureWritable() {
		return
	}
	if escape == escapeNone {
		_, w.err = io.WriteString(w.out, s)
		return
	}
	start := 0
	for i := range len(s) {
		replacement := ""
		writeRawByte := false
		switch s[i] {
		case '&':
			replacement = "&amp;"
		case '<':
			replacement = "&lt;"
		case '>':
			replacement = "&gt;"
		case '\r':
			replacement = "&#13;"
		case '"':
			if escape == escapeAttr && w.quoteChar == '"' {
				replacement = "&quot;"
			} else {
				writeRawByte = true
			}
		case '\'':
			if escape == escapeAttr && w.quoteChar == '\'' {
				replacement = "&apos;"
			} else {
				writeRawByte = true
			}
		case '\n':
			if escape == escapeAttr {
				replacement = "&#10;"
			} else {
				writeRawByte = true
			}
		case '\t':
			if escape == escapeAttr {
				replacement = "&#9;"
			} else {
				writeRawByte = true
			}
		default:
			continue
		}
		if start < i {
			w.writeStr(s[start:i])
		}
		if replacement != "" {
			w.writeStr(replacement)
		} else if writeRawByte {
			w.writeByte(s[i])
		}
		start = i + 1
	}
	if start < len(s) {
		w.writeStr(s[start:])
	}
}

// writeByte writes a single byte.
func (w *Writer) writeByte(b byte) {
	if !w.ensureWritable() {
		return
	}
	if bw, ok := w.out.(io.ByteWriter); ok {
		w.err = bw.WriteByte(b)
		return
	}
	w.singleByte[0] = b
	_, w.err = w.out.Write(w.singleByte[:])
}

// writeIndent writes a newline followed by indent*depth.
func (w *Writer) writeIndent() {
	if w.indent == "" {
		return
	}
	if !w.hasOutput {
		// Don't write a leading newline before the very first output
		return
	}
	if w.wroteNL {
		// EndComment/EndPI already wrote \n; skip the extra one.
		w.wroteNL = false
	} else if w.depth > 0 {
		// At depth 0 (root level), StartDocument already wrote a trailing
		// newline, so skip the extra newline that writeIndent would add.
		w.writeStr("\n")
	}
	for range w.depth {
		w.writeStr(w.indent)
	}
}

// writeEndIndent writes indentation for a closing tag (depth-1 level).
func (w *Writer) writeEndIndent() {
	if w.indent == "" {
		return
	}
	if w.wroteNL {
		w.wroteNL = false
	} else {
		w.writeStr("\n")
	}
	for range w.depth {
		w.writeStr(w.indent)
	}
}

// closeTagIfOpen closes the element opening tag (writes ">") if we are
// in stateName. This is called before writing any element content.
func (w *Writer) closeTagIfOpen() {
	if w.state != stateName {
		return
	}
	// Emit any pending namespace declarations
	w.emitPendingNS()
	w.writeByte('>')
	w.state = stateText
	if len(w.elemStack) > 0 {
		w.elemStack[len(w.elemStack)-1].empty = false
	}
}

// emitPendingNS writes xmlns declarations for the current element scope
// that haven't been emitted yet. Called when closing the opening tag.
// Declarations are kept in the scope for namespace lookup by child elements.
func (w *Writer) emitPendingNS() {
	if len(w.nsStack) == 0 {
		return
	}
	scope := &w.nsStack[len(w.nsStack)-1]
	for i := scope.emitted; i < len(scope.decls); i++ {
		ns := scope.decls[i]
		if ns.prefix == "" {
			w.writeStr(" xmlns=")
			w.writeByte(w.quoteChar)
			w.writeAttrEscaped(ns.uri)
			w.writeByte(w.quoteChar)
		} else {
			w.writeStr(" xmlns:")
			w.writeStr(ns.prefix)
			w.writeByte('=')
			w.writeByte(w.quoteChar)
			w.writeAttrEscaped(ns.uri)
			w.writeByte(w.quoteChar)
		}
	}
	scope.emitted = len(scope.decls)
}

// lookupNS checks if the prefix is already bound to the given URI in
// the current namespace stack.
func (w *Writer) lookupNS(prefix, uri string) bool {
	for _, v := range slices.Backward(w.nsStack) {
		for _, ns := range v.decls {
			if ns.prefix == prefix {
				return ns.uri == uri
			}
		}
	}
	return false
}

// hasDefaultNSInScope returns true if any ancestor has declared a
// non-empty default namespace (xmlns="...") that is still in scope.
func (w *Writer) hasDefaultNSInScope() bool {
	for _, v := range slices.Backward(w.nsStack) {
		for _, ns := range v.decls {
			if ns.prefix == "" {
				return ns.uri != ""
			}
		}
	}
	return false
}

// declareNS adds a namespace declaration to the current element scope
// if the prefix is not already bound to the given URI.
func (w *Writer) declareNS(prefix, uri string) {
	if w.lookupNS(prefix, uri) {
		return
	}
	if len(w.nsStack) == 0 {
		return
	}
	scope := &w.nsStack[len(w.nsStack)-1]
	// Check if prefix already declared in this scope
	for _, ns := range scope.decls {
		if ns.prefix == prefix {
			return // already declared at this level
		}
	}
	scope.decls = append(scope.decls, nsEntry{prefix: prefix, uri: uri})
}

// validateNSParts validates the prefix and local name supplied to a
// namespace-aware method. The local name must be a valid NCName; the prefix,
// if non-empty, must also be a valid NCName (no colon). This prevents an
// untrusted prefix or local name from injecting markup into the start tag.
// kind is "element" or "attribute" and only shapes the error message.
func validateNSParts(kind, prefix, localName string) error {
	if prefix != "" && !xmlchar.IsValidNCName(prefix) {
		return fmt.Errorf("stream: invalid %s prefix %q", kind, prefix)
	}
	if !xmlchar.IsValidNCName(localName) {
		return fmt.Errorf("stream: invalid %s local name %q", kind, localName)
	}
	return nil
}

// isValidXMLVersion reports whether v is a valid XML declaration VersionNum.
// VersionNum ::= '1.' [0-9]+ (XML 1.0 §2.8). Restricting to this grammar
// prevents an untrusted version string from injecting markup into the XML
// declaration (e.g. `1.0"?><x/>`).
func isValidXMLVersion(v string) bool {
	rest, ok := strings.CutPrefix(v, "1.")
	if !ok {
		return false
	}
	if rest == "" {
		return false
	}
	for i := range len(rest) {
		if rest[i] < '0' || rest[i] > '9' {
			return false
		}
	}
	return true
}

// isPubidChar reports whether r is a valid XML PubidChar (XML 1.0 §2.3):
//
//	PubidChar ::= #x20 | #xD | #xA | [a-zA-Z0-9] | [-'()+,./:=?;!*#@$_%]
func isPubidChar(r rune) bool {
	switch {
	case r == 0x20 || r == 0xD || r == 0xA:
		return true
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		return true
	case strings.ContainsRune("-'()+,./:=?;!*#@$_%", r):
		return true
	default:
		return false
	}
}

// validatePubid reports whether s consists solely of PubidChars, so it cannot
// inject markup or be unrepresentable when emitted as a DTD public identifier.
func validatePubid(s string) bool {
	if !utf8.ValidString(s) {
		return false
	}
	for _, r := range s {
		if !isPubidChar(r) {
			return false
		}
	}
	return true
}

// validateSystemID reports whether s is representable as a quoted DTD system
// identifier. Per the XML SystemLiteral grammar a system literal may contain
// any valid XML char except the delimiting quote, so the only constraints are
// that s be valid XML chars and not contain both quote characters (which would
// make it unquotable). '<' and '>' are permitted: they are harmless inside the
// quoted literal, and dtdQuoteFor prevents the value from breaking out of its
// quotes.
func validateSystemID(s string) bool {
	if strings.Contains(s, "'") && strings.Contains(s, `"`) {
		return false
	}
	if !utf8.ValidString(s) {
		return false
	}
	for _, r := range s {
		if !xmlchar.IsChar(r) {
			return false
		}
	}
	return true
}

// validateXMLChars reports whether s contains only valid XML 1.0 Chars.
// kind shapes the error message. It is used to reject control characters
// (e.g. NUL) from text, attribute, comment, PI, and CDATA output.
func validateXMLChars(kind, s string) error {
	if !utf8.ValidString(s) {
		return fmt.Errorf("stream: invalid UTF-8 byte sequence in %s content", kind)
	}
	for _, r := range s {
		if !xmlchar.IsChar(r) {
			return fmt.Errorf("stream: invalid XML character %#U in %s content", r, kind)
		}
	}
	return nil
}

// validateDTDFragment validates an element contentspec written verbatim into
// the internal subset. Beyond rejecting non-XML characters, it forbids the
// markup-delimiter characters '<' and '>', which can never legitimately appear
// in an element contentspec (a contentspec has no quoted literals). Allowing
// them would let an untrusted fragment terminate the current declaration and
// inject a new one (e.g. a content of "ANY><!ENTITY e \"pwn\"" smuggling an
// extra <!ENTITY declaration).
func validateDTDFragment(kind, s string) error {
	if err := validateXMLChars(kind, s); err != nil {
		return err
	}
	for _, r := range s {
		if r == '<' || r == '>' {
			return fmt.Errorf("stream: %s content must not contain %q", kind, string(r))
		}
	}
	return nil
}

// validateDTDAttlistFragment validates an attlist body written verbatim into
// the internal subset. Like validateDTDFragment it rejects non-XML characters
// and any '<'. Unlike a contentspec, an attlist body legitimately contains
// quoted attribute default values (e.g. a CDATA "a>b"), so a '>' inside a
// single- or double-quoted literal is permitted; a '>' outside any quote is
// rejected because it would terminate the <!ATTLIST and allow injection of a
// following declaration. An unterminated literal is treated as outside-quote
// for the trailing content, so a dangling '>' is still caught.
func validateDTDAttlistFragment(kind, s string) error {
	if err := validateXMLChars(kind, s); err != nil {
		return err
	}
	var quote rune // 0 when outside a literal, else the active quote char
	for _, r := range s {
		// '<' is never legal in an attlist body: raw '<' is forbidden even
		// inside an AttValue literal per the XML grammar, so reject it
		// regardless of quote state.
		if r == '<' {
			return fmt.Errorf("stream: %s content must not contain %q", kind, string(r))
		}
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			}
		case r == '\'' || r == '"':
			quote = r
		case r == '>':
			return fmt.Errorf("stream: %s content must not contain %q outside a quoted literal", kind, string(r))
		}
	}
	if quote != 0 {
		return fmt.Errorf("stream: %s content has an unterminated %q literal", kind, string(quote))
	}
	return nil
}

// qualifiedName returns prefix:localName or just localName if prefix is empty.
func qualifiedName(prefix, localName string) string {
	if prefix == "" {
		return localName
	}
	return prefix + ":" + localName
}

// --- Escaping ---

// writeTextEscaped writes text content with XML escaping for element bodies.
func (w *Writer) writeTextEscaped(s string) {
	w.writeEscaped(s, escapeText)
}

// writeAttrEscaped writes text content with XML escaping for attribute values.
func (w *Writer) writeAttrEscaped(s string) {
	w.writeEscaped(s, escapeAttr)
}

// writeCDATAEscaped writes content into the current CDATA section, splitting
// any "]]>" terminator across two CDATA sections so the emitted bytes never
// contain a raw "]]>" inside a single section. The "]]>" sequence is broken
// after the "]]" by closing and reopening the section, yielding
// "]]" + "]]><![CDATA[" + ">". The count of trailing ']' is tracked across
// calls so a "]]>" split over multiple WriteString calls is also handled.
func (w *Writer) writeCDATAEscaped(s string) {
	start := 0
	for i := range len(s) {
		c := s[i]
		if c == '>' && w.cdataBrackets >= 2 {
			// "]]>" detected: the preceding "]]" has already been
			// written into this section, so flush up to here, close the
			// section, and reopen a new one before emitting the '>'.
			// Guard against an empty write when "]]>" straddles a call
			// boundary (start == i), which would emit an avoidable empty
			// write and could trip side-effecting io.StringWriter impls.
			if start < i {
				w.writeStr(s[start:i])
			}
			w.writeStr("]]><![CDATA[")
			start = i
			w.cdataBrackets = 0
		}
		if c == ']' {
			w.cdataBrackets++
			continue
		}
		w.cdataBrackets = 0
	}
	w.writeStr(s[start:])
}

// --- Document lifecycle ---

// StartDocument writes the XML declaration. Pass "" for any parameter to
// use its default (version="1.0", no encoding, no standalone).
func (w *Writer) StartDocument(version, enc, standalone string) error {
	if w.err != nil {
		return w.err
	}
	if w.state != stateNone {
		return errors.New("stream: StartDocument called in invalid state")
	}
	if version == "" {
		version = "1.0"
	}
	// Validate version and standalone before writing, so an untrusted value
	// cannot inject markup into (or otherwise corrupt) the XML declaration.
	if !isValidXMLVersion(version) {
		return fmt.Errorf("stream: invalid XML version %q", version)
	}
	if standalone != "" && standalone != "yes" && standalone != "no" {
		return fmt.Errorf("stream: invalid standalone value %q (want \"yes\", \"no\", or empty)", standalone)
	}
	// Validate the encoding name BEFORE writing any output: first against the XML
	// EncName production (so a syntactically malformed value like "utf 8" cannot
	// be emitted raw into the declaration), then for actual support. encoding.Load
	// is lenient (it normalizes/accepts non-EncName spellings), so the syntactic
	// check must come first.
	if enc != "" {
		if !xmlchar.IsValidEncName(enc) {
			return fmt.Errorf("stream: invalid encoding name %q", enc)
		}
		if encoding.Load(enc) == nil {
			return fmt.Errorf("stream: unsupported encoding %q", enc)
		}
	}
	w.writeStr("<?xml version=")
	w.writeByte(w.quoteChar)
	w.writeStr(version)
	w.writeByte(w.quoteChar)
	if enc != "" {
		w.writeStr(" encoding=")
		w.writeByte(w.quoteChar)
		w.writeStr(enc)
		w.writeByte(w.quoteChar)
	}
	if standalone != "" {
		w.writeStr(" standalone=")
		w.writeByte(w.quoteChar)
		w.writeStr(standalone)
		w.writeByte(w.quoteChar)
	}
	w.writeStr("?>\n")
	w.state = stateDocument
	w.hasOutput = true
	return w.err
}

// EndDocument auto-closes any open constructs (PI, CDATA, comment, DTD,
// elements) and flushes the output.
func (w *Writer) EndDocument() error {
	if w.err != nil {
		return w.err
	}
	// Auto-close any open construct before closing elements.
	switch w.state {
	case statePI, statePIText:
		if err := w.EndPI(); err != nil {
			return err
		}
	case stateCDATA:
		if err := w.EndCDATA(); err != nil {
			return err
		}
	case stateComment:
		if err := w.EndComment(); err != nil {
			return err
		}
	case stateDTD, stateDTDText:
		if err := w.EndDTD(); err != nil {
			return err
		}
	}
	// Close all open elements
	for len(w.elemStack) > 0 {
		if err := w.EndElement(); err != nil {
			return err
		}
	}
	if w.indent == "" {
		w.writeStr("\n")
	}
	w.state = stateNone
	return w.Flush()
}

// --- Elements ---

// StartElement opens a new element with the given local name.
func (w *Writer) StartElement(name string) error {
	if name == "" {
		return errors.New("stream: element name must not be empty")
	}
	if w.err != nil {
		return w.err
	}
	// Validate the name before touching any writer state or output, so an
	// untrusted name cannot inject markup (extra attributes, '>') or produce
	// malformed XML. Namespaced names (prefix:local) and the xml:/xmlns
	// idioms are valid QNames and remain accepted.
	if !xmlchar.IsValidQName(name) {
		return fmt.Errorf("stream: invalid element name %q", name)
	}
	switch w.state {
	case stateNone, stateDocument, stateText, stateDTD:
		// ok — stateNone allows fragment writing without StartDocument
	case stateName:
		w.closeTagIfOpen()
	default:
		return errors.New("stream: StartElement called in invalid state")
	}

	// Mark parent as having children
	if len(w.elemStack) > 0 {
		w.elemStack[len(w.elemStack)-1].empty = false
		w.elemStack[len(w.elemStack)-1].hasChild = true
	}

	w.writeIndent()
	w.writeByte('<')
	w.writeStr(name)
	w.hasOutput = true

	w.elemStack = append(w.elemStack, elementEntry{name: name, empty: true})
	w.nsStack = append(w.nsStack, nsScope{})
	w.state = stateName
	w.depth++
	return w.err
}

// StartElementNS opens a new element with namespace. prefix and
// namespaceURI may be empty.
func (w *Writer) StartElementNS(prefix, localName, namespaceURI string) error {
	if w.err != nil {
		return w.err
	}
	if err := validateNSParts("element", prefix, localName); err != nil {
		return err
	}
	// Validate the namespace URI before StartElement emits anything or
	// declareNS records a declaration, so an untrusted URI cannot inject
	// markup as an xmlns attribute value and a rejected call leaves the
	// writer unmutated.
	if err := validateXMLChars("namespace URI", namespaceURI); err != nil {
		return err
	}
	qname := qualifiedName(prefix, localName)
	if err := w.StartElement(qname); err != nil {
		return err
	}
	if namespaceURI != "" {
		w.declareNS(prefix, namespaceURI)
	} else if prefix == "" && w.hasDefaultNSInScope() {
		// When an element has no namespace but a default namespace is
		// in scope from an ancestor, emit xmlns="" to undeclare it.
		w.declareNS("", "")
	}
	return w.err
}

// EndElement closes the current element. Uses self-closing form "/>"
// when the element has no content.
func (w *Writer) EndElement() error {
	if w.err != nil {
		return w.err
	}
	if len(w.elemStack) == 0 {
		return errors.New("stream: EndElement called with no open element")
	}
	if w.state == stateAttribute {
		if err := w.EndAttribute(); err != nil {
			return err
		}
	}

	w.depth--
	entry := w.elemStack[len(w.elemStack)-1]
	w.elemStack = w.elemStack[:len(w.elemStack)-1]

	if w.state == stateName && entry.empty {
		// Self-closing: emit pending NS then close
		w.emitPendingNS()
		w.writeStr("/>")
	} else {
		if w.state == stateName {
			w.closeTagIfOpen()
		}
		if entry.hasChild && !entry.hasText {
			w.writeEndIndent()
		}
		w.writeStr("</")
		w.writeStr(entry.name)
		w.writeByte('>')
	}

	// Pop namespace scope
	if len(w.nsStack) > 0 {
		w.nsStack = w.nsStack[:len(w.nsStack)-1]
	}

	// Restore state
	if len(w.elemStack) > 0 {
		w.state = stateText
	} else {
		w.state = stateDocument
	}
	return w.err
}

// FullEndElement closes the current element with an explicit end tag
// even when empty.
func (w *Writer) FullEndElement() error {
	if w.err != nil {
		return w.err
	}
	if len(w.elemStack) == 0 {
		return errors.New("stream: FullEndElement called with no open element")
	}
	if w.state == stateAttribute {
		if err := w.EndAttribute(); err != nil {
			return err
		}
	}

	// Force close the opening tag if still open
	if w.state == stateName {
		w.closeTagIfOpen()
	}

	w.depth--
	entry := w.elemStack[len(w.elemStack)-1]
	w.elemStack = w.elemStack[:len(w.elemStack)-1]

	if entry.hasChild && !entry.hasText {
		w.writeEndIndent()
	}
	w.writeStr("</")
	w.writeStr(entry.name)
	w.writeByte('>')

	// Pop namespace scope
	if len(w.nsStack) > 0 {
		w.nsStack = w.nsStack[:len(w.nsStack)-1]
	}

	if len(w.elemStack) > 0 {
		w.state = stateText
	} else {
		w.state = stateDocument
	}
	return w.err
}

// WriteElement is a convenience for StartElement + WriteString + EndElement.
func (w *Writer) WriteElement(name, content string) error {
	if w.err != nil {
		return w.err
	}
	// Pre-validate content before StartElement emits the opening tag, so a
	// rejected write leaves the writer unmutated.
	if err := validateXMLChars("text", content); err != nil {
		return err
	}
	if err := w.StartElement(name); err != nil {
		return err
	}
	if err := w.WriteString(content); err != nil {
		return err
	}
	return w.EndElement()
}

// WriteElementNS is a convenience for StartElementNS + WriteString + EndElement.
func (w *Writer) WriteElementNS(prefix, localName, namespaceURI, content string) error {
	if w.err != nil {
		return w.err
	}
	// Pre-validate content before StartElementNS declares the namespace or
	// emits markup, so a rejected write leaves the writer unmutated.
	if err := validateXMLChars("text", content); err != nil {
		return err
	}
	if err := w.StartElementNS(prefix, localName, namespaceURI); err != nil {
		return err
	}
	if err := w.WriteString(content); err != nil {
		return err
	}
	return w.EndElement()
}

// --- Attributes ---

// StartAttribute opens an attribute on the current element.
func (w *Writer) StartAttribute(name string) error {
	if name == "" {
		return errors.New("stream: attribute name must not be empty")
	}
	if w.err != nil {
		return w.err
	}
	// Validate the name before writing, so an untrusted name cannot inject
	// markup (extra attributes, '>') into the start tag. The xmlns/xmlns:
	// declaration idiom and namespaced names are valid QNames and remain
	// accepted (this is a low-level xmlTextWriter-style API).
	if !xmlchar.IsValidQName(name) {
		return fmt.Errorf("stream: invalid attribute name %q", name)
	}
	if w.state != stateName {
		return errors.New("stream: StartAttribute called outside element opening tag")
	}
	w.writeByte(' ')
	w.writeStr(name)
	w.writeByte('=')
	w.writeByte(w.quoteChar)
	w.stateStack = append(w.stateStack, w.state)
	w.state = stateAttribute
	return w.err
}

// StartAttributeNS opens a namespace-qualified attribute.
func (w *Writer) StartAttributeNS(prefix, localName, namespaceURI string) error {
	if w.err != nil {
		return w.err
	}
	// Validate parts before declareNS mutates the namespace scope, so an
	// invalid prefix/local name never leaks a declaration or markup.
	if err := validateNSParts("attribute", prefix, localName); err != nil {
		return err
	}
	// Validate the namespace URI before declareNS records a declaration or
	// StartAttribute emits anything, so an untrusted URI cannot inject markup
	// as an xmlns attribute value and a rejected call leaves the writer
	// unmutated.
	if err := validateXMLChars("namespace URI", namespaceURI); err != nil {
		return err
	}
	if namespaceURI != "" && prefix != "" {
		w.declareNS(prefix, namespaceURI)
	}
	return w.StartAttribute(qualifiedName(prefix, localName))
}

// EndAttribute closes the current attribute.
func (w *Writer) EndAttribute() error {
	if w.err != nil {
		return w.err
	}
	if w.state != stateAttribute {
		return errors.New("stream: EndAttribute called outside attribute")
	}
	w.writeByte(w.quoteChar)
	// Restore previous state
	if len(w.stateStack) > 0 {
		w.state = w.stateStack[len(w.stateStack)-1]
		w.stateStack = w.stateStack[:len(w.stateStack)-1]
	} else {
		w.state = stateName
	}
	return w.err
}

// WriteAttribute is a convenience for StartAttribute + WriteString + EndAttribute.
func (w *Writer) WriteAttribute(name, value string) error {
	if w.err != nil {
		return w.err
	}
	// Pre-validate the value before StartAttribute emits ` name="`, so a
	// rejected write leaves the writer unmutated.
	if err := validateXMLChars("attribute", value); err != nil {
		return err
	}
	if err := w.StartAttribute(name); err != nil {
		return err
	}
	if err := w.WriteString(value); err != nil {
		return err
	}
	return w.EndAttribute()
}

// WriteAttributeNS is a convenience for StartAttributeNS + WriteString + EndAttribute.
func (w *Writer) WriteAttributeNS(prefix, localName, namespaceURI, value string) error {
	if w.err != nil {
		return w.err
	}
	// Pre-validate the value before StartAttributeNS declares the namespace or
	// emits markup, so a rejected write leaves the writer unmutated.
	if err := validateXMLChars("attribute", value); err != nil {
		return err
	}
	if err := w.StartAttributeNS(prefix, localName, namespaceURI); err != nil {
		return err
	}
	if err := w.WriteString(value); err != nil {
		return err
	}
	return w.EndAttribute()
}

// --- Text content ---

// WriteString writes text content with automatic XML escaping appropriate
// for the current context.
func (w *Writer) WriteString(content string) error {
	if w.err != nil {
		return w.err
	}
	switch w.state {
	case stateName:
		if err := validateXMLChars("text", content); err != nil {
			return err
		}
		w.closeTagIfOpen()
		if len(w.elemStack) > 0 {
			w.elemStack[len(w.elemStack)-1].hasText = true
		}
		w.writeTextEscaped(content)
	case stateNone, stateText, stateDocument:
		if err := validateXMLChars("text", content); err != nil {
			return err
		}
		if len(w.elemStack) > 0 {
			w.elemStack[len(w.elemStack)-1].hasText = true
		}
		w.writeTextEscaped(content)
	case stateAttribute:
		if err := validateXMLChars("attribute", content); err != nil {
			return err
		}
		w.writeAttrEscaped(content)
	case stateComment:
		if err := validateXMLChars("comment", content); err != nil {
			return err
		}
		if w.commentDash && strings.HasPrefix(content, "-") {
			return errors.New("stream: comment content must not contain '--'")
		}
		if strings.Contains(content, "--") {
			return errors.New("stream: comment content must not contain '--'")
		}
		w.writeStr(content)
		if content != "" {
			w.commentDash = strings.HasSuffix(content, "-")
		}
	case statePI, statePIText:
		if err := validateXMLChars("processing instruction", content); err != nil {
			return err
		}
		if w.piQuestion && strings.HasPrefix(content, ">") {
			return errors.New("stream: processing instruction content must not contain '?>'")
		}
		if strings.Contains(content, "?>") {
			return errors.New("stream: processing instruction content must not contain '?>'")
		}
		w.writeStr(content)
		if content != "" {
			w.piQuestion = strings.HasSuffix(content, "?")
		}
		w.state = statePIText
	case stateCDATA:
		if err := validateXMLChars("CDATA", content); err != nil {
			return err
		}
		w.writeCDATAEscaped(content)
	default:
		return errors.New("stream: WriteString called in invalid state")
	}
	return w.err
}

// WriteRaw writes content directly without any escaping.
// Callers must ensure the content is well-formed XML; passing
// untrusted input may produce malformed output or introduce
// XML injection vulnerabilities.
func (w *Writer) WriteRaw(content string) error {
	if w.err != nil {
		return w.err
	}
	switch w.state {
	case stateName:
		w.closeTagIfOpen()
		if len(w.elemStack) > 0 {
			w.elemStack[len(w.elemStack)-1].hasText = true
		}
		w.writeStr(content)
	case stateNone, stateText, stateDocument:
		if len(w.elemStack) > 0 {
			w.elemStack[len(w.elemStack)-1].hasText = true
		}
		w.writeStr(content)
	case stateAttribute:
		w.writeStr(content)
	default:
		return errors.New("stream: WriteRaw called in invalid state")
	}
	return w.err
}

// --- Comments ---

// StartComment opens a comment (<!--).
func (w *Writer) StartComment() error {
	if w.err != nil {
		return w.err
	}
	switch w.state {
	case stateNone, stateDocument, stateText:
		// ok
	case stateName:
		w.closeTagIfOpen()
	default:
		return errors.New("stream: StartComment called in invalid state")
	}
	if len(w.elemStack) > 0 {
		w.elemStack[len(w.elemStack)-1].empty = false
		w.elemStack[len(w.elemStack)-1].hasChild = true
	}
	w.writeStr("<!--")
	w.commentDash = false
	w.stateStack = append(w.stateStack, w.state)
	w.state = stateComment
	return w.err
}

// EndComment closes a comment (-->).
func (w *Writer) EndComment() error {
	if w.err != nil {
		return w.err
	}
	if w.state != stateComment {
		return errors.New("stream: EndComment called outside comment")
	}
	if w.commentDash {
		return errors.New("stream: comment content must not end with '-'")
	}
	w.writeStr("-->")
	if w.indent != "" {
		w.writeStr("\n")
		w.wroteNL = true
	}
	if len(w.stateStack) > 0 {
		w.state = w.stateStack[len(w.stateStack)-1]
		w.stateStack = w.stateStack[:len(w.stateStack)-1]
	} else {
		w.state = stateDocument
	}
	return w.err
}

// WriteComment is a convenience for StartComment + WriteString + EndComment.
func (w *Writer) WriteComment(content string) error {
	// A prior sticky I/O error must win over the new content validation below.
	if w.err != nil {
		return w.err
	}
	// Pre-validate content before StartComment emits "<!--", so a rejected
	// write leaves the writer unmutated.
	if err := validateXMLChars("comment", content); err != nil {
		return err
	}
	if strings.Contains(content, "--") {
		return errors.New("stream: comment content must not contain '--'")
	}
	if strings.HasSuffix(content, "-") {
		return errors.New("stream: comment content must not end with '-'")
	}
	if err := w.StartComment(); err != nil {
		return err
	}
	if err := w.WriteString(content); err != nil {
		return err
	}
	return w.EndComment()
}

// --- Processing Instructions ---

// StartPI opens a processing instruction (<?target).
func (w *Writer) StartPI(target string) error {
	if target == "" {
		return errors.New("stream: PI target must not be empty")
	}
	if w.err != nil {
		return w.err
	}
	// Validate the target before touching any writer state or output, so a bad
	// target never closes an open start tag or otherwise mutates the writer.
	if strings.EqualFold(target, "xml") {
		return errors.New("stream: PI target cannot be 'xml'")
	}
	if !isValidPITarget(target) {
		return fmt.Errorf("stream: invalid PI target %q", target)
	}
	switch w.state {
	case stateNone, stateDocument, stateText:
		// ok
	case stateName:
		w.closeTagIfOpen()
	default:
		return errors.New("stream: StartPI called in invalid state")
	}
	if len(w.elemStack) > 0 {
		w.elemStack[len(w.elemStack)-1].empty = false
		w.elemStack[len(w.elemStack)-1].hasChild = true
	}
	w.writeStr("<?")
	w.writeStr(target)
	w.piQuestion = false
	w.stateStack = append(w.stateStack, w.state)
	w.state = statePI
	return w.err
}

// EndPI closes a processing instruction (?>).
func (w *Writer) EndPI() error {
	if w.err != nil {
		return w.err
	}
	if w.state != statePI && w.state != statePIText {
		return errors.New("stream: EndPI called outside processing instruction")
	}
	w.writeStr("?>")
	if w.indent != "" {
		w.writeStr("\n")
		w.wroteNL = true
	}
	if len(w.stateStack) > 0 {
		w.state = w.stateStack[len(w.stateStack)-1]
		w.stateStack = w.stateStack[:len(w.stateStack)-1]
	} else {
		w.state = stateDocument
	}
	return w.err
}

// WritePI is a convenience for StartPI + WriteString + EndPI.
func (w *Writer) WritePI(target, content string) error {
	// A prior sticky I/O error must win over the new content validation below.
	if w.err != nil {
		return w.err
	}
	// Pre-validate content before StartPI emits "<?target", so a rejected
	// write leaves the writer unmutated.
	if err := validateXMLChars("processing instruction", content); err != nil {
		return err
	}
	if strings.Contains(content, "?>") {
		return errors.New("stream: processing instruction content must not contain '?>'")
	}
	if err := w.StartPI(target); err != nil {
		return err
	}
	if content != "" {
		w.writeByte(' ')
		if err := w.WriteString(content); err != nil {
			return err
		}
	}
	return w.EndPI()
}

// --- CDATA ---

// StartCDATA opens a CDATA section.
func (w *Writer) StartCDATA() error {
	if w.err != nil {
		return w.err
	}
	switch w.state {
	case stateText:
		// ok
	case stateName:
		w.closeTagIfOpen()
	default:
		return errors.New("stream: StartCDATA called in invalid state")
	}
	if len(w.elemStack) > 0 {
		w.elemStack[len(w.elemStack)-1].empty = false
		w.elemStack[len(w.elemStack)-1].hasText = true
	}
	w.writeStr("<![CDATA[")
	w.cdataBrackets = 0
	w.stateStack = append(w.stateStack, w.state)
	w.state = stateCDATA
	return w.err
}

// EndCDATA closes a CDATA section.
func (w *Writer) EndCDATA() error {
	if w.err != nil {
		return w.err
	}
	if w.state != stateCDATA {
		return errors.New("stream: EndCDATA called outside CDATA section")
	}
	w.writeStr("]]>")
	if len(w.stateStack) > 0 {
		w.state = w.stateStack[len(w.stateStack)-1]
		w.stateStack = w.stateStack[:len(w.stateStack)-1]
	} else {
		w.state = stateText
	}
	return w.err
}

// WriteCDATA is a convenience for StartCDATA + WriteString + EndCDATA.
// If content contains "]]>", it is split into multiple CDATA sections
// per the XML serialization spec (e.g., "]]>" becomes "]]]]><![CDATA[>").
func (w *Writer) WriteCDATA(content string) error {
	if w.err != nil {
		return w.err
	}
	// Pre-validate content before StartCDATA emits "<![CDATA[", so a rejected
	// write leaves the writer unmutated.
	if err := validateXMLChars("CDATA", content); err != nil {
		return err
	}
	for {
		idx := strings.Index(content, "]]>")
		if idx < 0 {
			break
		}
		// Write everything up to and including "]]" as one CDATA section
		if err := w.StartCDATA(); err != nil {
			return err
		}
		if err := w.WriteString(content[:idx+2]); err != nil {
			return err
		}
		if err := w.EndCDATA(); err != nil {
			return err
		}
		// Continue with ">" onwards in the next CDATA section
		content = content[idx+2:]
	}
	if err := w.StartCDATA(); err != nil {
		return err
	}
	if err := w.WriteString(content); err != nil {
		return err
	}
	return w.EndCDATA()
}

// --- DTD ---

// StartDTD opens a DOCTYPE declaration. pubid and sysid may be empty.
// If pubid is non-empty, sysid must also be non-empty.
func (w *Writer) StartDTD(name, pubid, sysid string) error {
	if name == "" {
		return errors.New("stream: DTD name must not be empty")
	}
	if w.err != nil {
		return w.err
	}
	if w.state != stateDocument {
		return errors.New("stream: StartDTD called in invalid state")
	}
	if pubid != "" && sysid == "" {
		return errors.New("stream: StartDTD requires sysid when pubid is provided")
	}
	// Validate name and external identifiers before writing, so an untrusted
	// value cannot inject markup into (or produce an unquotable) DOCTYPE.
	if !xmlchar.IsValidName(name) {
		return fmt.Errorf("stream: invalid DTD name %q", name)
	}
	if pubid != "" && !validatePubid(pubid) {
		return fmt.Errorf("stream: invalid DTD public identifier %q", pubid)
	}
	if sysid != "" && !validateSystemID(sysid) {
		return fmt.Errorf("stream: invalid DTD system identifier %q", sysid)
	}
	if w.indent != "" {
		w.writeStr("\n")
	}
	w.writeStr("<!DOCTYPE ")
	w.writeStr(name)
	if pubid != "" {
		if w.indent != "" {
			w.writeStr("\nPUBLIC ")
		} else {
			w.writeStr(" PUBLIC ")
		}
		pubQ := dtdQuoteFor(pubid, w.quoteChar)
		w.writeByte(pubQ)
		w.writeStr(pubid)
		w.writeByte(pubQ)
		if w.indent != "" {
			w.writeStr("\n       ")
		} else {
			w.writeByte(' ')
		}
		sysQ := dtdQuoteFor(sysid, w.quoteChar)
		w.writeByte(sysQ)
		w.writeStr(sysid)
		w.writeByte(sysQ)
	} else if sysid != "" {
		if w.indent != "" {
			w.writeStr("\nSYSTEM ")
		} else {
			w.writeStr(" SYSTEM ")
		}
		sysQ := dtdQuoteFor(sysid, w.quoteChar)
		w.writeByte(sysQ)
		w.writeStr(sysid)
		w.writeByte(sysQ)
	}
	w.state = stateDTD
	return w.err
}

// dtdQuoteFor returns the appropriate quote character for a DTD identifier.
// If the value contains the preferred quote, use the other one.
func dtdQuoteFor(value string, preferred byte) byte {
	if strings.ContainsRune(value, rune(preferred)) {
		if preferred == '"' {
			return '\''
		}
		return '"'
	}
	return preferred
}

// ensureDTDInternalSubset writes the opening " [" for the DTD internal
// subset if it hasn't been written yet.
func (w *Writer) ensureDTDInternalSubset() {
	if w.state == stateDTD {
		w.writeStr(" [")
		w.state = stateDTDText
	}
}

// EndDTD closes the DOCTYPE declaration.
func (w *Writer) EndDTD() error {
	if w.err != nil {
		return w.err
	}
	if w.state != stateDTD && w.state != stateDTDText {
		return errors.New("stream: EndDTD called outside DTD")
	}
	if w.state == stateDTDText {
		w.writeStr("]>")
	} else {
		w.writeByte('>')
	}
	if w.indent != "" {
		w.writeStr("\n")
		w.wroteNL = true
	}
	w.state = stateDocument
	return w.err
}

// WriteDTD writes a complete DOCTYPE declaration. The name, pubid, and sysid
// arguments are validated, but subset is written verbatim into the internal
// subset ("[" ... "]") without any escaping or validation, like [Writer.WriteRaw].
// Callers must ensure subset is well-formed DTD content; passing untrusted
// input may produce malformed output or introduce XML injection
// vulnerabilities. To build an internal subset safely, use the
// WriteDTDElement, WriteDTDAttlist, WriteDTDEntity, and WriteDTDNotation
// methods between StartDTD and EndDTD instead.
func (w *Writer) WriteDTD(name, pubid, sysid, subset string) error {
	if err := w.StartDTD(name, pubid, sysid); err != nil {
		return err
	}
	if subset != "" {
		w.writeStr(" [")
		w.writeStr(subset)
		w.writeByte(']')
	}
	return w.EndDTD()
}

// WriteDTDElement writes a DTD element declaration.
func (w *Writer) WriteDTDElement(name, content string) error {
	if w.err != nil {
		return w.err
	}
	if w.state != stateDTD && w.state != stateDTDText {
		return errors.New("stream: WriteDTDElement called outside DTD")
	}
	if !xmlchar.IsValidName(name) {
		return fmt.Errorf("stream: invalid DTD element name %q", name)
	}
	if err := validateDTDFragment("DTD element", content); err != nil {
		return err
	}
	w.ensureDTDInternalSubset()
	w.writeStr("<!ELEMENT ")
	w.writeStr(name)
	w.writeByte(' ')
	w.writeStr(content)
	w.writeByte('>')
	return w.err
}

// WriteDTDAttlist writes a DTD attribute list declaration.
func (w *Writer) WriteDTDAttlist(name, content string) error {
	if w.err != nil {
		return w.err
	}
	if w.state != stateDTD && w.state != stateDTDText {
		return errors.New("stream: WriteDTDAttlist called outside DTD")
	}
	if !xmlchar.IsValidName(name) {
		return fmt.Errorf("stream: invalid DTD attlist name %q", name)
	}
	if err := validateDTDAttlistFragment("DTD attlist", content); err != nil {
		return err
	}
	w.ensureDTDInternalSubset()
	w.writeStr("<!ATTLIST ")
	w.writeStr(name)
	w.writeByte(' ')
	w.writeStr(content)
	w.writeByte('>')
	return w.err
}

// WriteDTDEntity writes an internal entity declaration.
// Set pe to true for parameter entities.
func (w *Writer) WriteDTDEntity(pe bool, name, content string) error {
	if w.err != nil {
		return w.err
	}
	if w.state != stateDTD && w.state != stateDTDText {
		return errors.New("stream: WriteDTDEntity called outside DTD")
	}
	// An entity name is an XML Name, but helium's parser forbids colons in entity
	// names, so validate as an NCName to match parser behavior (a "p:e" written
	// here would be unparseable).
	if !xmlchar.IsValidNCName(name) {
		return fmt.Errorf("stream: invalid DTD entity name %q", name)
	}
	if err := validateXMLChars("DTD entity", content); err != nil {
		return err
	}
	w.ensureDTDInternalSubset()
	w.writeStr("<!ENTITY ")
	if pe {
		w.writeStr("% ")
	}
	w.writeStr(name)
	w.writeByte(' ')
	w.writeByte(w.quoteChar)
	w.writeAttrEscaped(content)
	w.writeByte(w.quoteChar)
	w.writeByte('>')
	return w.err
}

// WriteDTDExternalEntity writes an external entity declaration.
func (w *Writer) WriteDTDExternalEntity(pe bool, name, pubid, sysid, ndata string) error {
	if w.err != nil {
		return w.err
	}
	if w.state != stateDTD && w.state != stateDTDText {
		return errors.New("stream: WriteDTDExternalEntity called outside DTD")
	}
	// An entity name is an XML Name, but helium's parser forbids colons in entity
	// names, so validate as an NCName to match parser behavior (a "p:e" written
	// here would be unparseable).
	if !xmlchar.IsValidNCName(name) {
		return fmt.Errorf("stream: invalid DTD entity name %q", name)
	}
	if pubid != "" && !validatePubid(pubid) {
		return fmt.Errorf("stream: invalid DTD public identifier %q", pubid)
	}
	if sysid != "" && !validateSystemID(sysid) {
		return fmt.Errorf("stream: invalid DTD system identifier %q", sysid)
	}
	if ndata != "" && !xmlchar.IsValidName(ndata) {
		return fmt.Errorf("stream: invalid DTD notation name %q", ndata)
	}
	w.ensureDTDInternalSubset()
	w.writeStr("<!ENTITY ")
	if pe {
		w.writeStr("% ")
	}
	w.writeStr(name)
	if pubid != "" {
		w.writeStr(" PUBLIC ")
		pubQ := dtdQuoteFor(pubid, w.quoteChar)
		w.writeByte(pubQ)
		w.writeStr(pubid)
		w.writeByte(pubQ)
		w.writeByte(' ')
		sysQ := dtdQuoteFor(sysid, w.quoteChar)
		w.writeByte(sysQ)
		w.writeStr(sysid)
		w.writeByte(sysQ)
	} else if sysid != "" {
		w.writeStr(" SYSTEM ")
		sysQ := dtdQuoteFor(sysid, w.quoteChar)
		w.writeByte(sysQ)
		w.writeStr(sysid)
		w.writeByte(sysQ)
	}
	if ndata != "" {
		w.writeStr(" NDATA ")
		w.writeStr(ndata)
	}
	w.writeByte('>')
	return w.err
}

// WriteDTDNotation writes a notation declaration.
func (w *Writer) WriteDTDNotation(name, pubid, sysid string) error {
	if w.err != nil {
		return w.err
	}
	if w.state != stateDTD && w.state != stateDTDText {
		return errors.New("stream: WriteDTDNotation called outside DTD")
	}
	if !xmlchar.IsValidName(name) {
		return fmt.Errorf("stream: invalid DTD notation name %q", name)
	}
	if pubid != "" && !validatePubid(pubid) {
		return fmt.Errorf("stream: invalid DTD public identifier %q", pubid)
	}
	if sysid != "" && !validateSystemID(sysid) {
		return fmt.Errorf("stream: invalid DTD system identifier %q", sysid)
	}
	w.ensureDTDInternalSubset()
	w.writeStr("<!NOTATION ")
	w.writeStr(name)
	if pubid != "" {
		w.writeStr(" PUBLIC ")
		pubQ := dtdQuoteFor(pubid, w.quoteChar)
		w.writeByte(pubQ)
		w.writeStr(pubid)
		w.writeByte(pubQ)
		if sysid != "" {
			w.writeByte(' ')
			sysQ := dtdQuoteFor(sysid, w.quoteChar)
			w.writeByte(sysQ)
			w.writeStr(sysid)
			w.writeByte(sysQ)
		}
	} else if sysid != "" {
		w.writeStr(" SYSTEM ")
		sysQ := dtdQuoteFor(sysid, w.quoteChar)
		w.writeByte(sysQ)
		w.writeStr(sysid)
		w.writeByte(sysQ)
	}
	w.writeByte('>')
	return w.err
}

// --- Flush ---

// Flush delegates to the underlying writer's Flush method if it
// implements one (e.g. *bufio.Writer). It is a no-op otherwise.
func (w *Writer) Flush() error {
	if w.err != nil {
		return w.err
	}
	if f, ok := w.out.(interface{ Flush() error }); ok {
		w.err = f.Flush()
	}
	return w.err
}
