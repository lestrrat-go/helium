// Package xmlwriter implements a streaming XML writer equivalent to
// libxml2's xmlTextWriter. It produces well-formed XML incrementally
// via method calls, writing directly to an io.Writer without building
// an in-memory DOM tree.
package xmlwriter

import (
	"errors"
	"io"
	"strings"
)

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
type Writer struct {
	out        io.Writer
	indent     string // indent string per level; empty = no indentation
	quoteChar  byte   // attribute quote character ('"' or '\'')
	state      writerState
	elemStack  []elementEntry
	nsStack    []nsScope
	stateStack []writerState // for comment/PI/CDATA nesting
	err        error         // sticky error
	depth      int           // current element nesting depth (for indentation)
	hasOutput  bool          // true after first output has been written
}

// Option configures a Writer.
type Option func(*Writer)

// WithIndent enables indentation. Each nested level is indented by the
// given string (e.g. "  " for two spaces, "\t" for tab).
func WithIndent(indent string) Option {
	return func(w *Writer) { w.indent = indent }
}

// WithQuoteChar sets the attribute value quote character. Must be '"' or '\''.
// Default is '"'.
func WithQuoteChar(q byte) Option {
	return func(w *Writer) {
		if q == '\'' || q == '"' {
			w.quoteChar = q
		}
	}
}

// New creates a Writer that writes to w.
func New(w io.Writer, opts ...Option) *Writer {
	wr := &Writer{
		out:       w,
		quoteChar: '"',
		state:     stateNone,
	}
	for _, o := range opts {
		o(wr)
	}
	return wr
}

// writeStr writes a string to the underlying writer, recording any error.
func (w *Writer) writeStr(s string) {
	if w.err != nil {
		return
	}
	_, w.err = io.WriteString(w.out, s)
}

// writeByte writes a single byte.
func (w *Writer) writeByte(b byte) {
	if w.err != nil {
		return
	}
	_, w.err = w.out.Write([]byte{b})
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
	// At depth 0 (root level), StartDocument already wrote a trailing
	// newline, so skip the extra newline that writeIndent would add.
	if w.depth > 0 {
		w.writeStr("\n")
	}
	for i := 0; i < w.depth; i++ {
		w.writeStr(w.indent)
	}
}

// writeEndIndent writes indentation for a closing tag (depth-1 level).
func (w *Writer) writeEndIndent() {
	if w.indent == "" {
		return
	}
	w.writeStr("\n")
	for i := 0; i < w.depth; i++ {
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
	for i := len(w.nsStack) - 1; i >= 0; i-- {
		for _, ns := range w.nsStack[i].decls {
			if ns.prefix == prefix {
				return ns.uri == uri
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
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '&':
			w.writeStr("&amp;")
		case '<':
			w.writeStr("&lt;")
		case '>':
			w.writeStr("&gt;")
		case '\r':
			w.writeStr("&#13;")
		default:
			w.writeByte(s[i])
		}
	}
}

// writeAttrEscaped writes text content with XML escaping for attribute values.
func (w *Writer) writeAttrEscaped(s string) {
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '&':
			w.writeStr("&amp;")
		case '<':
			w.writeStr("&lt;")
		case '>':
			w.writeStr("&gt;")
		case '"':
			if w.quoteChar == '"' {
				w.writeStr("&quot;")
			} else {
				w.writeByte('"')
			}
		case '\'':
			if w.quoteChar == '\'' {
				w.writeStr("&apos;")
			} else {
				w.writeByte('\'')
			}
		case '\n':
			w.writeStr("&#10;")
		case '\r':
			w.writeStr("&#13;")
		case '\t':
			w.writeStr("&#9;")
		default:
			w.writeByte(s[i])
		}
	}
}

// --- Document lifecycle ---

// StartDocument writes the XML declaration. Pass "" for any parameter to
// use its default (version="1.0", no encoding, no standalone).
func (w *Writer) StartDocument(version, encoding, standalone string) error {
	if w.err != nil {
		return w.err
	}
	if w.state != stateNone {
		return errors.New("xmlwriter: StartDocument called in invalid state")
	}
	if version == "" {
		version = "1.0"
	}
	w.writeStr("<?xml version=")
	w.writeByte(w.quoteChar)
	w.writeStr(version)
	w.writeByte(w.quoteChar)
	if encoding != "" {
		w.writeStr(" encoding=")
		w.writeByte(w.quoteChar)
		w.writeStr(encoding)
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

// EndDocument closes all open elements and flushes the output.
func (w *Writer) EndDocument() error {
	if w.err != nil {
		return w.err
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
	if w.err != nil {
		return w.err
	}
	switch w.state {
	case stateNone, stateDocument, stateText, stateDTD:
		// ok — stateNone allows fragment writing without StartDocument
	case stateName:
		w.closeTagIfOpen()
	default:
		return errors.New("xmlwriter: StartElement called in invalid state")
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
	qname := qualifiedName(prefix, localName)
	if err := w.StartElement(qname); err != nil {
		return err
	}
	if namespaceURI != "" {
		w.declareNS(prefix, namespaceURI)
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
		return errors.New("xmlwriter: EndElement called with no open element")
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
		return errors.New("xmlwriter: FullEndElement called with no open element")
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
	if w.err != nil {
		return w.err
	}
	if w.state != stateName {
		return errors.New("xmlwriter: StartAttribute called outside element opening tag")
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
		return errors.New("xmlwriter: EndAttribute called outside attribute")
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
		w.closeTagIfOpen()
		if len(w.elemStack) > 0 {
			w.elemStack[len(w.elemStack)-1].hasText = true
		}
		w.writeTextEscaped(content)
	case stateText:
		if len(w.elemStack) > 0 {
			w.elemStack[len(w.elemStack)-1].hasText = true
		}
		w.writeTextEscaped(content)
	case stateAttribute:
		w.writeAttrEscaped(content)
	case stateComment:
		w.writeStr(content)
	case statePI, statePIText:
		w.writeStr(content)
		w.state = statePIText
	case stateCDATA:
		w.writeStr(content)
	default:
		return errors.New("xmlwriter: WriteString called in invalid state")
	}
	return w.err
}

// WriteRaw writes content without any escaping.
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
	case stateText, stateDocument:
		if len(w.elemStack) > 0 {
			w.elemStack[len(w.elemStack)-1].hasText = true
		}
		w.writeStr(content)
	case stateAttribute:
		w.writeStr(content)
	default:
		return errors.New("xmlwriter: WriteRaw called in invalid state")
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
		return errors.New("xmlwriter: StartComment called in invalid state")
	}
	if len(w.elemStack) > 0 {
		w.elemStack[len(w.elemStack)-1].empty = false
		w.elemStack[len(w.elemStack)-1].hasChild = true
	}
	w.writeStr("<!--")
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
		return errors.New("xmlwriter: EndComment called outside comment")
	}
	w.writeStr("-->")
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
	if w.err != nil {
		return w.err
	}
	switch w.state {
	case stateNone, stateDocument, stateText:
		// ok
	case stateName:
		w.closeTagIfOpen()
	default:
		return errors.New("xmlwriter: StartPI called in invalid state")
	}
	if strings.EqualFold(target, "xml") {
		return errors.New("xmlwriter: PI target cannot be 'xml'")
	}
	if len(w.elemStack) > 0 {
		w.elemStack[len(w.elemStack)-1].empty = false
		w.elemStack[len(w.elemStack)-1].hasChild = true
	}
	w.writeStr("<?")
	w.writeStr(target)
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
		return errors.New("xmlwriter: EndPI called outside processing instruction")
	}
	w.writeStr("?>")
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
		return errors.New("xmlwriter: StartCDATA called in invalid state")
	}
	if len(w.elemStack) > 0 {
		w.elemStack[len(w.elemStack)-1].empty = false
		w.elemStack[len(w.elemStack)-1].hasText = true
	}
	w.writeStr("<![CDATA[")
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
		return errors.New("xmlwriter: EndCDATA called outside CDATA section")
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
func (w *Writer) WriteCDATA(content string) error {
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
func (w *Writer) StartDTD(name, pubid, sysid string) error {
	if w.err != nil {
		return w.err
	}
	if w.state != stateDocument {
		return errors.New("xmlwriter: StartDTD called in invalid state")
	}
	if w.indent != "" {
		w.writeStr("\n")
	}
	w.writeStr("<!DOCTYPE ")
	w.writeStr(name)
	if pubid != "" {
		w.writeStr(" PUBLIC ")
		w.writeByte(w.quoteChar)
		w.writeStr(pubid)
		w.writeByte(w.quoteChar)
		w.writeByte(' ')
		w.writeByte(w.quoteChar)
		w.writeStr(sysid)
		w.writeByte(w.quoteChar)
	} else if sysid != "" {
		w.writeStr(" SYSTEM ")
		w.writeByte(w.quoteChar)
		w.writeStr(sysid)
		w.writeByte(w.quoteChar)
	}
	w.state = stateDTD
	return w.err
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
		return errors.New("xmlwriter: EndDTD called outside DTD")
	}
	if w.state == stateDTDText {
		w.writeStr("]>")
	} else {
		w.writeByte('>')
	}
	w.state = stateDocument
	return w.err
}

// WriteDTD writes a complete DOCTYPE declaration.
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
		return errors.New("xmlwriter: WriteDTDElement called outside DTD")
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
		return errors.New("xmlwriter: WriteDTDAttlist called outside DTD")
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
		return errors.New("xmlwriter: WriteDTDEntity called outside DTD")
	}
	w.ensureDTDInternalSubset()
	w.writeStr("<!ENTITY ")
	if pe {
		w.writeStr("% ")
	}
	w.writeStr(name)
	w.writeByte(' ')
	w.writeByte(w.quoteChar)
	w.writeStr(content)
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
		return errors.New("xmlwriter: WriteDTDExternalEntity called outside DTD")
	}
	w.ensureDTDInternalSubset()
	w.writeStr("<!ENTITY ")
	if pe {
		w.writeStr("% ")
	}
	w.writeStr(name)
	if pubid != "" {
		w.writeStr(" PUBLIC ")
		w.writeByte(w.quoteChar)
		w.writeStr(pubid)
		w.writeByte(w.quoteChar)
		w.writeByte(' ')
		w.writeByte(w.quoteChar)
		w.writeStr(sysid)
		w.writeByte(w.quoteChar)
	} else if sysid != "" {
		w.writeStr(" SYSTEM ")
		w.writeByte(w.quoteChar)
		w.writeStr(sysid)
		w.writeByte(w.quoteChar)
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
		return errors.New("xmlwriter: WriteDTDNotation called outside DTD")
	}
	w.ensureDTDInternalSubset()
	w.writeStr("<!NOTATION ")
	w.writeStr(name)
	if pubid != "" {
		w.writeStr(" PUBLIC ")
		w.writeByte(w.quoteChar)
		w.writeStr(pubid)
		w.writeByte(w.quoteChar)
		if sysid != "" {
			w.writeByte(' ')
			w.writeByte(w.quoteChar)
			w.writeStr(sysid)
			w.writeByte(w.quoteChar)
		}
	} else if sysid != "" {
		w.writeStr(" SYSTEM ")
		w.writeByte(w.quoteChar)
		w.writeStr(sysid)
		w.writeByte(w.quoteChar)
	}
	w.writeByte('>')
	return w.err
}

// --- Flush ---

// Flush flushes any buffered data to the underlying writer.
func (w *Writer) Flush() error {
	if w.err != nil {
		return w.err
	}
	if f, ok := w.out.(interface{ Flush() error }); ok {
		w.err = f.Flush()
	}
	return w.err
}
