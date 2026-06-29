package html

import (
	"errors"

	"github.com/lestrrat-go/helium/sax"
)

// ErrHandlerUnspecified is returned when a SAXCallbacks method is called
// but no handler is registered for that event. This is not a fatal error
// and can be ignored if the caller only listens to specific events.
var ErrHandlerUnspecified = errors.New("handler unspecified")

// ErrContentSizeExceeded is returned from parsing when a single comment,
// bogus comment, or processing instruction exceeds the configured
// [Parser.MaxContentSize] before reaching its terminator. Unlike raw-text
// content (script/style/textarea/plaintext), these constructs map to a single
// indivisible SAX event and DOM node, so they cannot be chunked without
// corrupting the document. The cap is therefore enforced as a hard limit and
// the parse fails rather than emitting a truncated node and leaking the
// remainder as stray text.
//
// It is also returned — in normal data-state text as well as the RCDATA path —
// for ANY unresolved named character-reference literal — an "&"-prefixed
// sequence that does not resolve to a known entity or legacy prefix — whether
// short, semicolon-terminated, or unbounded, once the literal bytes it would
// emit ("&" + name + optional ";") exceed the cap. A known-entity
// (';'-terminated) reference is exempt: it is resolved within a fixed lookahead
// window and never charged against the cap. A no-';' LEGACY resolution — a full
// legacy entity (e.g. "&amp") OR a legacy-PREFIX match (e.g. "&ampZ", where
// "amp" resolves and "Z" is echoed) — is exempt only when its whole consumed
// run ("&" + name) fits within the cap; over the cap it hard-fails with this
// error and emits NOTHING. This holds uniformly for both a SHORT
// within-lookahead run (e.g. "&ampZ" under a cap of 2) and a saturated ambiguous
// run (e.g. "&amp" followed by a long alphanumeric tail).
//
// Finally, it is returned for a normal-text whitespace case whenever a run's
// leading whitespace must be DEFERRED with its parent or significance still
// undecided: under [Parser.StripBlanks](true) (a run is suppressed only when
// entirely whitespace) OR during implied-<body> deferral (mode < insertInBody
// with implied insertion enabled, so the next non-whitespace byte would open the
// implied <body>). The scanner cannot flush a run whose leading whitespace prefix
// reaches the cap with yet more whitespace beyond it without buffering the run
// unbounded to learn its significance or parent — such a run fails rather than
// parsing successfully. Default-mode whitespace with a fixed insertion target and
// no StripBlanks stays a soft-cap stream and never hits this case.
//
// It is also returned for an over-cap indivisible STRUCTURAL token scan: a
// tag name, an end-tag name, an attribute name, a PUBLIC/SYSTEM DOCTYPE literal,
// or an intra-tag whitespace run. These are part of a single indivisible
// start/end-tag or DOCTYPE event and cannot be chunked, so each is a HARD cap.
// Their bound is the structural scan cap, which is NOT [Parser.MaxContentSize]
// (a content-chunking knob callers legitimately set very small): it is floored
// at the 16 MiB default so ordinary names like "script" always parse, and grows
// only when MaxContentSize is raised above that floor. An over-cap structural
// run fails the parse and emits no partial token.
//
// Lastly, [Parser.ParseReader] returns it for an UNDECLARED-charset stream whose
// bytes stay valid UTF-8 past the configured [Parser.MaxContentSize] (16 MiB by
// default) without ever revealing their encoding. Such a stream cannot be
// committed to UTF-8 within the memory bound — a later non-UTF-8 byte would flip
// the whole document to Windows-1252 (matching [Parser.Parse]) while EOF would
// keep it UTF-8 — so rather than buffer it unbounded or risk a silently
// mis-decoded result that diverges from the in-memory path, the reader fails
// closed with this error. A stream that declares its charset, or that settles its
// encoding (reaches EOF or a non-UTF-8 byte) below the limit, is unaffected.
var ErrContentSizeExceeded = errors.New("content size limit exceeded")

// DocumentLocator is an alias for [sax.DocumentLocator].
// (libxml2: xmlSAXLocator)
type DocumentLocator = sax.DocumentLocator

// Attribute represents an HTML element attribute (name=value pair).
type Attribute struct {
	Name    string
	Value   string
	Boolean bool // true for boolean attributes (no value specified in source)
}

// Per-method handler interfaces. Each follows the http.Handler pattern:
// an interface with a single method, paired with a func type adapter.
// Unlike the XML SAX2 handlers in package sax, these HTML SAX1 callbacks
// take no context.Context and report elements by simple name with no
// namespace information, matching libxml2's HTML SAX.

// SetDocumentLocatorHandler handles the SetDocumentLocator event, called once
// before any other event to supply the source-position locator.
type SetDocumentLocatorHandler interface {
	SetDocumentLocator(loc DocumentLocator) error
}

// StartDocumentHandler handles the StartDocument event, emitted once at the
// start of the document.
type StartDocumentHandler interface {
	StartDocument() error
}

// EndDocumentHandler handles the EndDocument event, emitted once at the end of
// the document.
type EndDocumentHandler interface {
	EndDocument() error
}

// StartElementHandler handles a start tag. name is the element name (no
// namespace) and attrs holds its attributes in source order.
type StartElementHandler interface {
	StartElement(name string, attrs []Attribute) error
}

// EndElementHandler handles an end tag for the element named name.
type EndElementHandler interface {
	EndElement(name string) error
}

// CharactersHandler handles a run of character data. The bytes are owned by the
// parser and must be copied if retained beyond the call.
type CharactersHandler interface {
	Characters(ch []byte) error
}

// CDataBlockHandler handles a CDATA section's content.
type CDataBlockHandler interface {
	CDataBlock(value []byte) error
}

// CommentHandler handles a comment's content (the text between <!-- and -->).
type CommentHandler interface {
	Comment(value []byte) error
}

// InternalSubsetHandler handles the document type declaration (<!DOCTYPE ...>).
type InternalSubsetHandler interface {
	InternalSubset(name, externalID, systemID string) error
}

// ProcessingInstructionHandler handles a processing instruction with the given
// target and data.
type ProcessingInstructionHandler interface {
	ProcessingInstruction(target, data string) error
}

// IgnorableWhitespaceHandler handles whitespace that is ignorable per the
// content model.
type IgnorableWhitespaceHandler interface {
	IgnorableWhitespace(ch []byte) error
}

// ErrorHandler handles a parse error. Returning a non-nil error aborts parsing.
type ErrorHandler interface {
	Error(err error) error
}

// WarningHandler handles a non-fatal parse warning.
type WarningHandler interface {
	Warning(err error) error
}

// Func type adapters. Each implements its corresponding handler interface,
// allowing a plain function to be used wherever the handler is expected.

// SetDocumentLocatorFunc adapts a function to SetDocumentLocatorHandler.
type SetDocumentLocatorFunc func(loc DocumentLocator) error

// SetDocumentLocator calls f.
func (f SetDocumentLocatorFunc) SetDocumentLocator(loc DocumentLocator) error { return f(loc) }

// StartDocumentFunc adapts a function to StartDocumentHandler.
type StartDocumentFunc func() error

// StartDocument calls f.
func (f StartDocumentFunc) StartDocument() error { return f() }

// EndDocumentFunc adapts a function to EndDocumentHandler.
type EndDocumentFunc func() error

// EndDocument calls f.
func (f EndDocumentFunc) EndDocument() error { return f() }

// StartElementFunc adapts a function to StartElementHandler.
type StartElementFunc func(name string, attrs []Attribute) error

// StartElement calls f.
func (f StartElementFunc) StartElement(name string, attrs []Attribute) error { return f(name, attrs) }

// EndElementFunc adapts a function to EndElementHandler.
type EndElementFunc func(name string) error

// EndElement calls f.
func (f EndElementFunc) EndElement(name string) error { return f(name) }

// CharactersFunc adapts a function to CharactersHandler.
type CharactersFunc func(ch []byte) error

// Characters calls f.
func (f CharactersFunc) Characters(ch []byte) error { return f(ch) }

// CDataBlockFunc adapts a function to CDataBlockHandler.
type CDataBlockFunc func(value []byte) error

// CDataBlock calls f.
func (f CDataBlockFunc) CDataBlock(value []byte) error { return f(value) }

// CommentFunc adapts a function to CommentHandler.
type CommentFunc func(value []byte) error

// Comment calls f.
func (f CommentFunc) Comment(value []byte) error { return f(value) }

// InternalSubsetFunc adapts a function to InternalSubsetHandler.
type InternalSubsetFunc func(name, externalID, systemID string) error

// InternalSubset calls f.
func (f InternalSubsetFunc) InternalSubset(name, externalID, systemID string) error {
	return f(name, externalID, systemID)
}

// ProcessingInstructionFunc adapts a function to ProcessingInstructionHandler.
type ProcessingInstructionFunc func(target, data string) error

// ProcessingInstruction calls f.
func (f ProcessingInstructionFunc) ProcessingInstruction(target, data string) error {
	return f(target, data)
}

// IgnorableWhitespaceFunc adapts a function to IgnorableWhitespaceHandler.
type IgnorableWhitespaceFunc func(ch []byte) error

// IgnorableWhitespace calls f.
func (f IgnorableWhitespaceFunc) IgnorableWhitespace(ch []byte) error { return f(ch) }

// ErrorFunc adapts a function to ErrorHandler.
type ErrorFunc func(err error) error

// Error calls f.
func (f ErrorFunc) Error(err error) error { return f(err) }

// WarningFunc adapts a function to WarningHandler.
type WarningFunc func(err error) error

// Warning calls f.
func (f WarningFunc) Warning(err error) error { return f(err) }

// SAXHandler is the HTML SAX1 handler interface. Unlike the XML SAX2Handler,
// this uses simple element names (no namespaces) matching libxml2's HTML SAX.
type SAXHandler interface {
	SetDocumentLocatorHandler
	StartDocumentHandler
	EndDocumentHandler
	StartElementHandler
	EndElementHandler
	CharactersHandler
	CDataBlockHandler
	CommentHandler
	InternalSubsetHandler
	ProcessingInstructionHandler
	IgnorableWhitespaceHandler
	ErrorHandler
	WarningHandler
}

// SAXCallbacks is a callback-based SAXHandler implementation.
// Use the SetOnXxx methods to register handlers.
// The zero value is ready to use; unset handlers return
// ErrHandlerUnspecified.
type SAXCallbacks struct {
	onSetDocumentLocator    SetDocumentLocatorHandler
	onStartDocument         StartDocumentHandler
	onEndDocument           EndDocumentHandler
	onStartElement          StartElementHandler
	onEndElement            EndElementHandler
	onCharacters            CharactersHandler
	onCDataBlock            CDataBlockHandler
	onComment               CommentHandler
	onInternalSubset        InternalSubsetHandler
	onProcessingInstruction ProcessingInstructionHandler
	onIgnorableWhitespace   IgnorableWhitespaceHandler
	onError                 ErrorHandler
	onWarning               WarningHandler
}

// SetOnSetDocumentLocator sets the handler for the SetDocumentLocator event.
func (s *SAXCallbacks) SetOnSetDocumentLocator(h SetDocumentLocatorHandler) {
	s.onSetDocumentLocator = h
}

// SetOnStartDocument sets the handler for the StartDocument event.
func (s *SAXCallbacks) SetOnStartDocument(h StartDocumentHandler) {
	s.onStartDocument = h
}

// SetOnEndDocument sets the handler for the EndDocument event.
func (s *SAXCallbacks) SetOnEndDocument(h EndDocumentHandler) {
	s.onEndDocument = h
}

// SetOnStartElement sets the handler for the StartElement event.
func (s *SAXCallbacks) SetOnStartElement(h StartElementHandler) {
	s.onStartElement = h
}

// SetOnEndElement sets the handler for the EndElement event.
func (s *SAXCallbacks) SetOnEndElement(h EndElementHandler) {
	s.onEndElement = h
}

// SetOnCharacters sets the handler for the Characters event.
func (s *SAXCallbacks) SetOnCharacters(h CharactersHandler) {
	s.onCharacters = h
}

// SetOnCDataBlock sets the handler for the CDataBlock event.
func (s *SAXCallbacks) SetOnCDataBlock(h CDataBlockHandler) {
	s.onCDataBlock = h
}

// SetOnComment sets the handler for the Comment event.
func (s *SAXCallbacks) SetOnComment(h CommentHandler) {
	s.onComment = h
}

// SetOnInternalSubset sets the handler for the InternalSubset event.
func (s *SAXCallbacks) SetOnInternalSubset(h InternalSubsetHandler) {
	s.onInternalSubset = h
}

// SetOnProcessingInstruction sets the handler for the ProcessingInstruction event.
func (s *SAXCallbacks) SetOnProcessingInstruction(h ProcessingInstructionHandler) {
	s.onProcessingInstruction = h
}

// SetOnIgnorableWhitespace sets the handler for the IgnorableWhitespace event.
func (s *SAXCallbacks) SetOnIgnorableWhitespace(h IgnorableWhitespaceHandler) {
	s.onIgnorableWhitespace = h
}

// SetOnError sets the handler for the Error event.
func (s *SAXCallbacks) SetOnError(h ErrorHandler) {
	s.onError = h
}

// SetOnWarning sets the handler for the Warning event.
func (s *SAXCallbacks) SetOnWarning(h WarningHandler) {
	s.onWarning = h
}

// SetDocumentLocator dispatches to the registered handler, or returns
// ErrHandlerUnspecified if none is set.
func (s *SAXCallbacks) SetDocumentLocator(loc DocumentLocator) error {
	if h := s.onSetDocumentLocator; h != nil {
		return h.SetDocumentLocator(loc)
	}
	return ErrHandlerUnspecified
}

// StartDocument dispatches to the registered handler, or returns
// ErrHandlerUnspecified if none is set.
func (s *SAXCallbacks) StartDocument() error {
	if h := s.onStartDocument; h != nil {
		return h.StartDocument()
	}
	return ErrHandlerUnspecified
}

// EndDocument dispatches to the registered handler, or returns
// ErrHandlerUnspecified if none is set.
func (s *SAXCallbacks) EndDocument() error {
	if h := s.onEndDocument; h != nil {
		return h.EndDocument()
	}
	return ErrHandlerUnspecified
}

// StartElement dispatches to the registered handler, or returns
// ErrHandlerUnspecified if none is set.
func (s *SAXCallbacks) StartElement(name string, attrs []Attribute) error {
	if h := s.onStartElement; h != nil {
		return h.StartElement(name, attrs)
	}
	return ErrHandlerUnspecified
}

// EndElement dispatches to the registered handler, or returns
// ErrHandlerUnspecified if none is set.
func (s *SAXCallbacks) EndElement(name string) error {
	if h := s.onEndElement; h != nil {
		return h.EndElement(name)
	}
	return ErrHandlerUnspecified
}

// Characters dispatches to the registered handler, or returns
// ErrHandlerUnspecified if none is set.
func (s *SAXCallbacks) Characters(ch []byte) error {
	if h := s.onCharacters; h != nil {
		return h.Characters(ch)
	}
	return ErrHandlerUnspecified
}

// CDataBlock dispatches to the registered handler, or returns
// ErrHandlerUnspecified if none is set.
func (s *SAXCallbacks) CDataBlock(value []byte) error {
	if h := s.onCDataBlock; h != nil {
		return h.CDataBlock(value)
	}
	return ErrHandlerUnspecified
}

// Comment dispatches to the registered handler, or returns
// ErrHandlerUnspecified if none is set.
func (s *SAXCallbacks) Comment(value []byte) error {
	if h := s.onComment; h != nil {
		return h.Comment(value)
	}
	return ErrHandlerUnspecified
}

// InternalSubset dispatches to the registered handler, or returns
// ErrHandlerUnspecified if none is set.
func (s *SAXCallbacks) InternalSubset(name, externalID, systemID string) error {
	if h := s.onInternalSubset; h != nil {
		return h.InternalSubset(name, externalID, systemID)
	}
	return ErrHandlerUnspecified
}

// ProcessingInstruction dispatches to the registered handler, or returns
// ErrHandlerUnspecified if none is set.
func (s *SAXCallbacks) ProcessingInstruction(target, data string) error {
	if h := s.onProcessingInstruction; h != nil {
		return h.ProcessingInstruction(target, data)
	}
	return ErrHandlerUnspecified
}

// IgnorableWhitespace dispatches to the registered handler, or returns
// ErrHandlerUnspecified if none is set.
func (s *SAXCallbacks) IgnorableWhitespace(ch []byte) error {
	if h := s.onIgnorableWhitespace; h != nil {
		return h.IgnorableWhitespace(ch)
	}
	return ErrHandlerUnspecified
}

// Error dispatches to the registered handler, or returns
// ErrHandlerUnspecified if none is set.
func (s *SAXCallbacks) Error(err error) error {
	if h := s.onError; h != nil {
		return h.Error(err)
	}
	return ErrHandlerUnspecified
}

// Warning dispatches to the registered handler, or returns
// ErrHandlerUnspecified if none is set.
func (s *SAXCallbacks) Warning(err error) error {
	if h := s.onWarning; h != nil {
		return h.Warning(err)
	}
	return ErrHandlerUnspecified
}
