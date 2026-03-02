package html

// DocumentLocator provides document position information to SAX handlers.
type DocumentLocator interface {
	LineNumber() int
	ColumnNumber() int
	// GetPublicId returns the public identifier of the document being parsed.
	// In practice this always returns an empty string.
	GetPublicId() string
	// GetSystemId returns the system identifier (URI/filename) of the document being parsed.
	GetSystemId() string
}

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
	SetDocumentLocatorHandler    SetDocumentLocatorHandler
	StartDocumentHandler         StartDocumentHandler
	EndDocumentHandler           EndDocumentHandler
	StartElementHandler          StartElementHandler
	EndElementHandler            EndElementHandler
	CharactersHandler            CharactersHandler
	CDataBlockHandler            CDataBlockHandler
	CommentHandler               CommentHandler
	InternalSubsetHandler        InternalSubsetHandler
	ProcessingInstructionHandler ProcessingInstructionHandler
	IgnorableWhitespaceHandler   IgnorableWhitespaceHandler
	ErrorHandler                 ErrorHandler
	WarningHandler               WarningHandler
}

func (s *SAXCallbacks) SetDocumentLocator(loc DocumentLocator) error {
	if h := s.SetDocumentLocatorHandler; h != nil {
		return h.SetDocumentLocator(loc)
	}
	return nil
}

func (s *SAXCallbacks) StartDocument() error {
	if h := s.StartDocumentHandler; h != nil {
		return h.StartDocument()
	}
	return nil
}

func (s *SAXCallbacks) EndDocument() error {
	if h := s.EndDocumentHandler; h != nil {
		return h.EndDocument()
	}
	return nil
}

func (s *SAXCallbacks) StartElement(name string, attrs []Attribute) error {
	if h := s.StartElementHandler; h != nil {
		return h.StartElement(name, attrs)
	}
	return nil
}

func (s *SAXCallbacks) EndElement(name string) error {
	if h := s.EndElementHandler; h != nil {
		return h.EndElement(name)
	}
	return nil
}

func (s *SAXCallbacks) Characters(ch []byte) error {
	if h := s.CharactersHandler; h != nil {
		return h.Characters(ch)
	}
	return nil
}

func (s *SAXCallbacks) CDataBlock(value []byte) error {
	if h := s.CDataBlockHandler; h != nil {
		return h.CDataBlock(value)
	}
	return nil
}

func (s *SAXCallbacks) Comment(value []byte) error {
	if h := s.CommentHandler; h != nil {
		return h.Comment(value)
	}
	return nil
}

func (s *SAXCallbacks) InternalSubset(name, externalID, systemID string) error {
	if h := s.InternalSubsetHandler; h != nil {
		return h.InternalSubset(name, externalID, systemID)
	}
	return nil
}

func (s *SAXCallbacks) ProcessingInstruction(target, data string) error {
	if h := s.ProcessingInstructionHandler; h != nil {
		return h.ProcessingInstruction(target, data)
	}
	return nil
}

func (s *SAXCallbacks) IgnorableWhitespace(ch []byte) error {
	if h := s.IgnorableWhitespaceHandler; h != nil {
		return h.IgnorableWhitespace(ch)
	}
	return nil
}

func (s *SAXCallbacks) Error(err error) error {
	if h := s.ErrorHandler; h != nil {
		return h.Error(err)
	}
	return nil
}

func (s *SAXCallbacks) Warning(err error) error {
	if h := s.WarningHandler; h != nil {
		return h.Warning(err)
	}
	return nil
}
