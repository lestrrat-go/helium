package sax

import "errors"

// ErrHandlerUnspecified is returned when there is no Handler
// registered for that particular event callback. This is not
// a fatal error per se, and can be ignored if the implementation
// chooses to do so.
var ErrHandlerUnspecified = errors.New("handler unspecified")

// SAX2Handler is an interface for anything that can satisfy
// helium's expected SAX2 API
type SAX2Handler interface {
	AttributeDecl(ctx Context, elem string, fullname string, typ int, def int, defaultValue string, tree Enumeration) error
	CDataBlock(ctx Context, value []byte) error
	Characters(ctx Context, ch []byte) error
	Comment(ctx Context, value []byte) error
	ElementDecl(ctx Context, name string, typ int, content ElementContent) error
	EndDocument(ctx Context) error
	EndElementNS(ctx Context, localname string, prefix string, uri string) error
	EntityDecl(ctx Context, name string, typ int, publicID string, systemID string, content string) error
	Error(ctx Context, message string, args ...interface{}) error
	ExternalSubset(ctx Context, name string, externalID string, systemID string) error
	GetEntity(ctx Context, name string) (Entity, error)
	GetParameterEntity(ctx Context, name string) (Entity, error)
	HasExternalSubset(ctx Context) (bool, error)
	HasInternalSubset(ctx Context) (bool, error)
	IgnorableWhitespace(ctx Context, ch []byte) error
	InternalSubset(ctx Context, name string, externalID string, systemID string) error
	IsStandalone(ctx Context) (bool, error)
	NotationDecl(ctx Context, name string, publicID string, systemID string) error
	ProcessingInstruction(ctx Context, target string, data string) error
	Reference(ctx Context, name string) error
	ResolveEntity(ctx Context, publicID string, systemID string) (ParseInput, error)
	SetDocumentLocator(ctx Context, locator DocumentLocator) error
	StartDocument(ctx Context) error
	StartElementNS(ctx Context, localname string, prefix string, uri string, namespaces []Namespace, attrs []Attribute) error
	UnparsedEntityDecl(ctx Context, name string, publicID string, systemID string, notationName string) error
}

// SAX2 is the callback based SAX2 handler.
type SAX2 struct {
	AttributeDeclHandler AttributeDeclFunc
	CDataBlockHandler CDataBlockFunc
	CharactersHandler CharactersFunc
	CommentHandler CommentFunc
	ElementDeclHandler ElementDeclFunc
	EndDocumentHandler EndDocumentFunc
	EndElementNSHandler EndElementNSFunc
	EntityDeclHandler EntityDeclFunc
	ErrorHandler ErrorFunc
	ExternalSubsetHandler ExternalSubsetFunc
	GetEntityHandler GetEntityFunc
	GetParameterEntityHandler GetParameterEntityFunc
	HasExternalSubsetHandler HasExternalSubsetFunc
	HasInternalSubsetHandler HasInternalSubsetFunc
	IgnorableWhitespaceHandler IgnorableWhitespaceFunc
	InternalSubsetHandler InternalSubsetFunc
	IsStandaloneHandler IsStandaloneFunc
	NotationDeclHandler NotationDeclFunc
	ProcessingInstructionHandler ProcessingInstructionFunc
	ReferenceHandler ReferenceFunc
	ResolveEntityHandler ResolveEntityFunc
	SetDocumentLocatorHandler SetDocumentLocatorFunc
	StartDocumentHandler StartDocumentFunc
	StartElementNSHandler StartElementNSFunc
	UnparsedEntityDeclHandler UnparsedEntityDeclFunc
}

// New creates a new instance of SAX2. All callbacks are
// uninitialized.
func New() *SAX2 {
	return &SAX2{}
}

func (s SAX2) AttributeDecl(ctx Context, elem string, fullname string, typ int, def int, defaultValue string, tree Enumeration) error {
	if h := s.AttributeDeclHandler; h != nil {
		return h(ctx, elem, fullname, typ, def, defaultValue, tree)
	}
	return ErrHandlerUnspecified;
}

func (s SAX2) CDataBlock(ctx Context, value []byte) error {
	if h := s.CDataBlockHandler; h != nil {
		return h(ctx, value)
	}
	return ErrHandlerUnspecified;
}

func (s SAX2) Characters(ctx Context, ch []byte) error {
	if h := s.CharactersHandler; h != nil {
		return h(ctx, ch)
	}
	return ErrHandlerUnspecified;
}

func (s SAX2) Comment(ctx Context, value []byte) error {
	if h := s.CommentHandler; h != nil {
		return h(ctx, value)
	}
	return ErrHandlerUnspecified;
}

func (s SAX2) ElementDecl(ctx Context, name string, typ int, content ElementContent) error {
	if h := s.ElementDeclHandler; h != nil {
		return h(ctx, name, typ, content)
	}
	return ErrHandlerUnspecified;
}

func (s SAX2) EndDocument(ctx Context) error {
	if h := s.EndDocumentHandler; h != nil {
		return h(ctx)
	}
	return ErrHandlerUnspecified;
}

func (s SAX2) EndElementNS(ctx Context, localname string, prefix string, uri string) error {
	if h := s.EndElementNSHandler; h != nil {
		return h(ctx, localname, prefix, uri)
	}
	return ErrHandlerUnspecified;
}

func (s SAX2) EntityDecl(ctx Context, name string, typ int, publicID string, systemID string, content string) error {
	if h := s.EntityDeclHandler; h != nil {
		return h(ctx, name, typ, publicID, systemID, content)
	}
	return ErrHandlerUnspecified;
}

func (s SAX2) Error(ctx Context, message string, args ...interface{}) error {
	if h := s.ErrorHandler; h != nil {
		return h(ctx, message, args)
	}
	return ErrHandlerUnspecified;
}

func (s SAX2) ExternalSubset(ctx Context, name string, externalID string, systemID string) error {
	if h := s.ExternalSubsetHandler; h != nil {
		return h(ctx, name, externalID, systemID)
	}
	return ErrHandlerUnspecified;
}

func (s SAX2) GetEntity(ctx Context, name string) (Entity, error) {
	if h := s.GetEntityHandler; h != nil {
		return h(ctx, name)
	}
	return nil, ErrHandlerUnspecified;
}

func (s SAX2) GetParameterEntity(ctx Context, name string) (Entity, error) {
	if h := s.GetParameterEntityHandler; h != nil {
		return h(ctx, name)
	}
	return nil, ErrHandlerUnspecified;
}

func (s SAX2) HasExternalSubset(ctx Context) (bool, error) {
	if h := s.HasExternalSubsetHandler; h != nil {
		return h(ctx)
	}
	return false, ErrHandlerUnspecified;
}

func (s SAX2) HasInternalSubset(ctx Context) (bool, error) {
	if h := s.HasInternalSubsetHandler; h != nil {
		return h(ctx)
	}
	return false, ErrHandlerUnspecified;
}

func (s SAX2) IgnorableWhitespace(ctx Context, ch []byte) error {
	if h := s.IgnorableWhitespaceHandler; h != nil {
		return h(ctx, ch)
	}
	return ErrHandlerUnspecified;
}

func (s SAX2) InternalSubset(ctx Context, name string, externalID string, systemID string) error {
	if h := s.InternalSubsetHandler; h != nil {
		return h(ctx, name, externalID, systemID)
	}
	return ErrHandlerUnspecified;
}

func (s SAX2) IsStandalone(ctx Context) (bool, error) {
	if h := s.IsStandaloneHandler; h != nil {
		return h(ctx)
	}
	return false, ErrHandlerUnspecified;
}

func (s SAX2) NotationDecl(ctx Context, name string, publicID string, systemID string) error {
	if h := s.NotationDeclHandler; h != nil {
		return h(ctx, name, publicID, systemID)
	}
	return ErrHandlerUnspecified;
}

func (s SAX2) ProcessingInstruction(ctx Context, target string, data string) error {
	if h := s.ProcessingInstructionHandler; h != nil {
		return h(ctx, target, data)
	}
	return ErrHandlerUnspecified;
}

func (s SAX2) Reference(ctx Context, name string) error {
	if h := s.ReferenceHandler; h != nil {
		return h(ctx, name)
	}
	return ErrHandlerUnspecified;
}

func (s SAX2) ResolveEntity(ctx Context, publicID string, systemID string) (ParseInput, error) {
	if h := s.ResolveEntityHandler; h != nil {
		return h(ctx, publicID, systemID)
	}
	return nil, ErrHandlerUnspecified;
}

func (s SAX2) SetDocumentLocator(ctx Context, locator DocumentLocator) error {
	if h := s.SetDocumentLocatorHandler; h != nil {
		return h(ctx, locator)
	}
	return ErrHandlerUnspecified;
}

func (s SAX2) StartDocument(ctx Context) error {
	if h := s.StartDocumentHandler; h != nil {
		return h(ctx)
	}
	return ErrHandlerUnspecified;
}

func (s SAX2) StartElementNS(ctx Context, localname string, prefix string, uri string, namespaces []Namespace, attrs []Attribute) error {
	if h := s.StartElementNSHandler; h != nil {
		return h(ctx, localname, prefix, uri, namespaces, attrs)
	}
	return ErrHandlerUnspecified;
}

func (s SAX2) UnparsedEntityDecl(ctx Context, name string, publicID string, systemID string, notationName string) error {
	if h := s.UnparsedEntityDeclHandler; h != nil {
		return h(ctx, name, publicID, systemID, notationName)
	}
	return ErrHandlerUnspecified;
}

