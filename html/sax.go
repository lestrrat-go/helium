package html

import (
	"errors"

	"github.com/lestrrat-go/helium/sax"
)

// ErrHandlerUnspecified is returned when a SAXCallbacks method is called
// but no handler is registered for that event. This is not a fatal error
// and can be ignored if the caller only listens to specific events.
var ErrHandlerUnspecified = errors.New("handler unspecified")

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

type SetDocumentLocatorHandler interface {
	SetDocumentLocator(loc DocumentLocator) error
}

type StartDocumentHandler interface {
	StartDocument() error
}

type EndDocumentHandler interface {
	EndDocument() error
}

type StartElementHandler interface {
	StartElement(name string, attrs []Attribute) error
}

type EndElementHandler interface {
	EndElement(name string) error
}

type CharactersHandler interface {
	Characters(ch []byte) error
}

type CDataBlockHandler interface {
	CDataBlock(value []byte) error
}

type CommentHandler interface {
	Comment(value []byte) error
}

type InternalSubsetHandler interface {
	InternalSubset(name, externalID, systemID string) error
}

type ProcessingInstructionHandler interface {
	ProcessingInstruction(target, data string) error
}

type IgnorableWhitespaceHandler interface {
	IgnorableWhitespace(ch []byte) error
}

type ErrorHandler interface {
	Error(err error) error
}

type WarningHandler interface {
	Warning(err error) error
}

// Func type adapters. Each implements its corresponding handler interface.

type SetDocumentLocatorFunc func(loc DocumentLocator) error

func (f SetDocumentLocatorFunc) SetDocumentLocator(loc DocumentLocator) error { return f(loc) }

type StartDocumentFunc func() error

func (f StartDocumentFunc) StartDocument() error { return f() }

type EndDocumentFunc func() error

func (f EndDocumentFunc) EndDocument() error { return f() }

type StartElementFunc func(name string, attrs []Attribute) error

func (f StartElementFunc) StartElement(name string, attrs []Attribute) error { return f(name, attrs) }

type EndElementFunc func(name string) error

func (f EndElementFunc) EndElement(name string) error { return f(name) }

type CharactersFunc func(ch []byte) error

func (f CharactersFunc) Characters(ch []byte) error { return f(ch) }

type CDataBlockFunc func(value []byte) error

func (f CDataBlockFunc) CDataBlock(value []byte) error { return f(value) }

type CommentFunc func(value []byte) error

func (f CommentFunc) Comment(value []byte) error { return f(value) }

type InternalSubsetFunc func(name, externalID, systemID string) error

func (f InternalSubsetFunc) InternalSubset(name, externalID, systemID string) error {
	return f(name, externalID, systemID)
}

type ProcessingInstructionFunc func(target, data string) error

func (f ProcessingInstructionFunc) ProcessingInstruction(target, data string) error {
	return f(target, data)
}

type IgnorableWhitespaceFunc func(ch []byte) error

func (f IgnorableWhitespaceFunc) IgnorableWhitespace(ch []byte) error { return f(ch) }

type ErrorFunc func(err error) error

func (f ErrorFunc) Error(err error) error { return f(err) }

type WarningFunc func(err error) error

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
// Each field accepts either a Func adapter or any type implementing
// the corresponding single-method Handler interface.
// The zero value is ready to use; nil handler fields are silently
// skipped (no-op).
type SAXCallbacks struct {
	OnSetDocumentLocator    SetDocumentLocatorHandler
	OnStartDocument         StartDocumentHandler
	OnEndDocument           EndDocumentHandler
	OnStartElement          StartElementHandler
	OnEndElement            EndElementHandler
	OnCharacters            CharactersHandler
	OnCDataBlock            CDataBlockHandler
	OnComment               CommentHandler
	OnInternalSubset        InternalSubsetHandler
	OnProcessingInstruction ProcessingInstructionHandler
	OnIgnorableWhitespace   IgnorableWhitespaceHandler
	OnError                 ErrorHandler
	OnWarning               WarningHandler
}

func (s *SAXCallbacks) SetDocumentLocator(loc DocumentLocator) error {
	if h := s.OnSetDocumentLocator; h != nil {
		return h.SetDocumentLocator(loc)
	}
	return ErrHandlerUnspecified
}

func (s *SAXCallbacks) StartDocument() error {
	if h := s.OnStartDocument; h != nil {
		return h.StartDocument()
	}
	return ErrHandlerUnspecified
}

func (s *SAXCallbacks) EndDocument() error {
	if h := s.OnEndDocument; h != nil {
		return h.EndDocument()
	}
	return ErrHandlerUnspecified
}

func (s *SAXCallbacks) StartElement(name string, attrs []Attribute) error {
	if h := s.OnStartElement; h != nil {
		return h.StartElement(name, attrs)
	}
	return ErrHandlerUnspecified
}

func (s *SAXCallbacks) EndElement(name string) error {
	if h := s.OnEndElement; h != nil {
		return h.EndElement(name)
	}
	return ErrHandlerUnspecified
}

func (s *SAXCallbacks) Characters(ch []byte) error {
	if h := s.OnCharacters; h != nil {
		return h.Characters(ch)
	}
	return ErrHandlerUnspecified
}

func (s *SAXCallbacks) CDataBlock(value []byte) error {
	if h := s.OnCDataBlock; h != nil {
		return h.CDataBlock(value)
	}
	return ErrHandlerUnspecified
}

func (s *SAXCallbacks) Comment(value []byte) error {
	if h := s.OnComment; h != nil {
		return h.Comment(value)
	}
	return ErrHandlerUnspecified
}

func (s *SAXCallbacks) InternalSubset(name, externalID, systemID string) error {
	if h := s.OnInternalSubset; h != nil {
		return h.InternalSubset(name, externalID, systemID)
	}
	return ErrHandlerUnspecified
}

func (s *SAXCallbacks) ProcessingInstruction(target, data string) error {
	if h := s.OnProcessingInstruction; h != nil {
		return h.ProcessingInstruction(target, data)
	}
	return ErrHandlerUnspecified
}

func (s *SAXCallbacks) IgnorableWhitespace(ch []byte) error {
	if h := s.OnIgnorableWhitespace; h != nil {
		return h.IgnorableWhitespace(ch)
	}
	return ErrHandlerUnspecified
}

func (s *SAXCallbacks) Error(err error) error {
	if h := s.OnError; h != nil {
		return h.Error(err)
	}
	return ErrHandlerUnspecified
}

func (s *SAXCallbacks) Warning(err error) error {
	if h := s.OnWarning; h != nil {
		return h.Warning(err)
	}
	return ErrHandlerUnspecified
}
