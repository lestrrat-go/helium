package html

// DocumentLocator provides document position information to SAX handlers.
type DocumentLocator interface {
	LineNumber() int
	ColumnNumber() int
}

// Attribute represents an HTML element attribute (name=value pair).
type Attribute struct {
	Name    string
	Value   string
	Boolean bool // true for boolean attributes (no value specified in source)
}

// SAXHandler is the HTML SAX1 handler interface. Unlike the XML SAX2Handler,
// this uses simple element names (no namespaces) matching libxml2's HTML SAX.
type SAXHandler interface {
	SetDocumentLocator(loc DocumentLocator) error
	StartDocument() error
	EndDocument() error
	StartElement(name string, attrs []Attribute) error
	EndElement(name string) error
	Characters(ch []byte) error
	CDataBlock(value []byte) error
	Comment(value []byte) error
	InternalSubset(name, externalID, systemID string) error
	ProcessingInstruction(target, data string) error
	IgnorableWhitespace(ch []byte) error
	Error(msg string, args ...interface{}) error
	Warning(msg string, args ...interface{}) error
}

// SAXCallbacks is a callback-based SAXHandler implementation.
type SAXCallbacks struct {
	SetDocumentLocatorHandler func(loc DocumentLocator) error
	StartDocumentHandler      func() error
	EndDocumentHandler        func() error
	StartElementHandler       func(name string, attrs []Attribute) error
	EndElementHandler         func(name string) error
	CharactersHandler         func(ch []byte) error
	CDataBlockHandler         func(value []byte) error
	CommentHandler            func(value []byte) error
	InternalSubsetHandler     func(name, externalID, systemID string) error
	ProcessingInstructionHandler func(target, data string) error
	IgnorableWhitespaceHandler func(ch []byte) error
	ErrorHandler              func(msg string, args ...interface{}) error
	WarningHandler            func(msg string, args ...interface{}) error
}

func (s *SAXCallbacks) SetDocumentLocator(loc DocumentLocator) error {
	if h := s.SetDocumentLocatorHandler; h != nil {
		return h(loc)
	}
	return nil
}

func (s *SAXCallbacks) StartDocument() error {
	if h := s.StartDocumentHandler; h != nil {
		return h()
	}
	return nil
}

func (s *SAXCallbacks) EndDocument() error {
	if h := s.EndDocumentHandler; h != nil {
		return h()
	}
	return nil
}

func (s *SAXCallbacks) StartElement(name string, attrs []Attribute) error {
	if h := s.StartElementHandler; h != nil {
		return h(name, attrs)
	}
	return nil
}

func (s *SAXCallbacks) EndElement(name string) error {
	if h := s.EndElementHandler; h != nil {
		return h(name)
	}
	return nil
}

func (s *SAXCallbacks) Characters(ch []byte) error {
	if h := s.CharactersHandler; h != nil {
		return h(ch)
	}
	return nil
}

func (s *SAXCallbacks) CDataBlock(value []byte) error {
	if h := s.CDataBlockHandler; h != nil {
		return h(value)
	}
	return nil
}

func (s *SAXCallbacks) Comment(value []byte) error {
	if h := s.CommentHandler; h != nil {
		return h(value)
	}
	return nil
}

func (s *SAXCallbacks) InternalSubset(name, externalID, systemID string) error {
	if h := s.InternalSubsetHandler; h != nil {
		return h(name, externalID, systemID)
	}
	return nil
}

func (s *SAXCallbacks) ProcessingInstruction(target, data string) error {
	if h := s.ProcessingInstructionHandler; h != nil {
		return h(target, data)
	}
	return nil
}

func (s *SAXCallbacks) IgnorableWhitespace(ch []byte) error {
	if h := s.IgnorableWhitespaceHandler; h != nil {
		return h(ch)
	}
	return nil
}

func (s *SAXCallbacks) Error(msg string, args ...interface{}) error {
	if h := s.ErrorHandler; h != nil {
		return h(msg, args...)
	}
	return nil
}

func (s *SAXCallbacks) Warning(msg string, args ...interface{}) error {
	if h := s.WarningHandler; h != nil {
		return h(msg, args...)
	}
	return nil
}
