package sax

import "errors"

var ErrHandlerUnspecified = errors.New("handler unspecified")

// AttributeDeclFunc defines the function type for SAX2.AttributeDeclHandler
type AttributeDeclFunc func(ctx Context, elemName string, attrName string, typ int, deftype int, defvalue AttributeDefaultValue, enum Enumeration) (error)

// CharactersFunc defines the function type for SAX2.CharactersHandler
type CharactersFunc func(ctx Context, content []byte) (error)

// CommentFunc defines the function type for SAX2.CommentHandler
type CommentFunc func(ctx Context, content []byte) (error)

// ElementDeclFunc defines the function type for SAX2.ElementDeclHandler
type ElementDeclFunc func(ctx Context, name string, typ int, content ElementContent) (error)

// EndCDATAFunc defines the function type for SAX2.EndCDATAHandler
type EndCDATAFunc func(ctx Context) (error)

// EndDTDFunc defines the function type for SAX2.EndDTDHandler
type EndDTDFunc func(ctx Context) (error)

// EndDocumentFunc defines the function type for SAX2.EndDocumentHandler
type EndDocumentFunc func(ctx Context) (error)

// EndElementFunc defines the function type for SAX2.EndElementHandler
type EndElementFunc func(ctx Context, elem ParsedElement) (error)

// EndEntityFunc defines the function type for SAX2.EndEntityHandler
type EndEntityFunc func(ctx Context, name string) (error)

// ExternalEntityDeclFunc defines the function type for SAX2.ExternalEntityDeclHandler
type ExternalEntityDeclFunc func(ctx Context, name string, publicID string, systemID string) (error)

// ExternalSubsetFunc defines the function type for SAX2.ExternalSubsetHandler
type ExternalSubsetFunc func(ctx Context, name string, publicID string, systemID string) (error)

// GetExternalSubsetFunc defines the function type for SAX2.GetExternalSubsetHandler
type GetExternalSubsetFunc func(ctx Context, name string, baseURI string) (error)

// GetParameterEntityFunc defines the function type for SAX2.GetParameterEntityHandler
type GetParameterEntityFunc func(ctx Context, nmae string) (Entity, error)

// IgnorableWhitespaceFunc defines the function type for SAX2.IgnorableWhitespaceHandler
type IgnorableWhitespaceFunc func(ctx Context, content []byte) (error)

// InternalEntityDeclFunc defines the function type for SAX2.InternalEntityDeclHandler
type InternalEntityDeclFunc func(ctx Context, name string, value string) (error)

// InternalSubsetFunc defines the function type for SAX2.InternalSubsetHandler
type InternalSubsetFunc func(ctx Context, name string, publicID string, systemID string) (error)

// NotationDeclFunc defines the function type for SAX2.NotationDeclHandler
type NotationDeclFunc func(ctx Context, name string, publicID string, systemID string) (error)

// ProcessingInstructionFunc defines the function type for SAX2.ProcessingInstructionHandler
type ProcessingInstructionFunc func(ctx Context, target string, data string) (error)

// ReferenceFunc defines the function type for SAX2.ReferenceHandler
type ReferenceFunc func(ctx Context, name string) (error)

// ResolveEntityFunc defines the function type for SAX2.ResolveEntityHandler
type ResolveEntityFunc func(ctx Context, name string, publicID string, baseURI string, systemID string) (Entity, error)

// SetDocumentLocatorFunc defines the function type for SAX2.SetDocumentLocatorHandler
type SetDocumentLocatorFunc func(ctx Context, loc DocumentLocator) (error)

// SkippedEntityFunc defines the function type for SAX2.SkippedEntityHandler
type SkippedEntityFunc func(ctx Context, name string) (error)

// StartCDATAFunc defines the function type for SAX2.StartCDATAHandler
type StartCDATAFunc func(ctx Context) (error)

// StartDTDFunc defines the function type for SAX2.StartDTDHandler
type StartDTDFunc func(ctx Context, name string, publicID string, systemID string) (error)

// StartDocumentFunc defines the function type for SAX2.StartDocumentHandler
type StartDocumentFunc func(ctx Context) (error)

// StartElementFunc defines the function type for SAX2.StartElementHandler
type StartElementFunc func(ctx Context, elem ParsedElement) (error)

// StartEntityFunc defines the function type for SAX2.StartEntityHandler
type StartEntityFunc func(ctx Context, name string) (error)

// UnparsedEntityDeclFunc defines the function type for SAX2.UnparsedEntityDeclHandler
type UnparsedEntityDeclFunc func(ctx Context, name string, typ int, publicID string, systemID string, notation string) (error)

type SAX2 struct {
	AttributeDeclHandler AttributeDeclFunc
	CharactersHandler CharactersFunc
	CommentHandler CommentFunc
	ElementDeclHandler ElementDeclFunc
	EndCDATAHandler EndCDATAFunc
	EndDTDHandler EndDTDFunc
	EndDocumentHandler EndDocumentFunc
	EndElementHandler EndElementFunc
	EndEntityHandler EndEntityFunc
	ExternalEntityDeclHandler ExternalEntityDeclFunc
	ExternalSubsetHandler ExternalSubsetFunc
	GetExternalSubsetHandler GetExternalSubsetFunc
	GetParameterEntityHandler GetParameterEntityFunc
	IgnorableWhitespaceHandler IgnorableWhitespaceFunc
	InternalEntityDeclHandler InternalEntityDeclFunc
	InternalSubsetHandler InternalSubsetFunc
	NotationDeclHandler NotationDeclFunc
	ProcessingInstructionHandler ProcessingInstructionFunc
	ReferenceHandler ReferenceFunc
	ResolveEntityHandler ResolveEntityFunc
	SetDocumentLocatorHandler SetDocumentLocatorFunc
	SkippedEntityHandler SkippedEntityFunc
	StartCDATAHandler StartCDATAFunc
	StartDTDHandler StartDTDFunc
	StartDocumentHandler StartDocumentFunc
	StartElementHandler StartElementFunc
	StartEntityHandler StartEntityFunc
	UnparsedEntityDeclHandler UnparsedEntityDeclFunc
}
func New() *SAX2 {
	return &SAX2{}
}

// AttributeDecl satisfies the DeclHandler interface
func (s *SAX2) AttributeDecl(ctx Context, elemName string, attrName string, typ int, deftype int, defvalue AttributeDefaultValue, enum Enumeration) error {
	if h := s.AttributeDeclHandler; h != nil {
		return h(ctx, elemName, attrName, typ, deftype, defvalue, enum)
	}
	return ErrHandlerUnspecified
}

// Characters satisfies the ContentHandler interface
func (s *SAX2) Characters(ctx Context, content []byte) error {
	if h := s.CharactersHandler; h != nil {
		return h(ctx, content)
	}
	return ErrHandlerUnspecified
}

// Comment satisfies the LexicalHandler interface
func (s *SAX2) Comment(ctx Context, content []byte) error {
	if h := s.CommentHandler; h != nil {
		return h(ctx, content)
	}
	return ErrHandlerUnspecified
}

// ElementDecl satisfies the DeclHandler interface
func (s *SAX2) ElementDecl(ctx Context, name string, typ int, content ElementContent) error {
	if h := s.ElementDeclHandler; h != nil {
		return h(ctx, name, typ, content)
	}
	return ErrHandlerUnspecified
}

// EndCDATA satisfies the LexicalHandler interface
func (s *SAX2) EndCDATA(ctx Context) error {
	if h := s.EndCDATAHandler; h != nil {
		return h(ctx)
	}
	return ErrHandlerUnspecified
}

// EndDTD satisfies the LexicalHandler interface
func (s *SAX2) EndDTD(ctx Context) error {
	if h := s.EndDTDHandler; h != nil {
		return h(ctx)
	}
	return ErrHandlerUnspecified
}

// EndDocument satisfies the ContentHandler interface
func (s *SAX2) EndDocument(ctx Context) error {
	if h := s.EndDocumentHandler; h != nil {
		return h(ctx)
	}
	return ErrHandlerUnspecified
}

// EndElement satisfies the ContentHandler interface
func (s *SAX2) EndElement(ctx Context, elem ParsedElement) error {
	if h := s.EndElementHandler; h != nil {
		return h(ctx, elem)
	}
	return ErrHandlerUnspecified
}

// EndEntity satisfies the LexicalHandler interface
func (s *SAX2) EndEntity(ctx Context, name string) error {
	if h := s.EndEntityHandler; h != nil {
		return h(ctx, name)
	}
	return ErrHandlerUnspecified
}

// ExternalEntityDecl satisfies the DeclHandler interface
func (s *SAX2) ExternalEntityDecl(ctx Context, name string, publicID string, systemID string) error {
	if h := s.ExternalEntityDeclHandler; h != nil {
		return h(ctx, name, publicID, systemID)
	}
	return ErrHandlerUnspecified
}

// ExternalSubset satisfies the Extensions interface
func (s *SAX2) ExternalSubset(ctx Context, name string, publicID string, systemID string) error {
	if h := s.ExternalSubsetHandler; h != nil {
		return h(ctx, name, publicID, systemID)
	}
	return ErrHandlerUnspecified
}

// GetExternalSubset satisfies the EntityResolver interface
func (s *SAX2) GetExternalSubset(ctx Context, name string, baseURI string) error {
	if h := s.GetExternalSubsetHandler; h != nil {
		return h(ctx, name, baseURI)
	}
	return ErrHandlerUnspecified
}

// GetParameterEntity satisfies the Extensions interface
func (s *SAX2) GetParameterEntity(ctx Context, nmae string) (Entity, error) {
	if h := s.GetParameterEntityHandler; h != nil {
		return h(ctx, nmae)
	}
	return nil, ErrHandlerUnspecified
}

// IgnorableWhitespace satisfies the ContentHandler interface
func (s *SAX2) IgnorableWhitespace(ctx Context, content []byte) error {
	if h := s.IgnorableWhitespaceHandler; h != nil {
		return h(ctx, content)
	}
	return ErrHandlerUnspecified
}

// InternalEntityDecl satisfies the DeclHandler interface
func (s *SAX2) InternalEntityDecl(ctx Context, name string, value string) error {
	if h := s.InternalEntityDeclHandler; h != nil {
		return h(ctx, name, value)
	}
	return ErrHandlerUnspecified
}

// InternalSubset satisfies the Extensions interface
func (s *SAX2) InternalSubset(ctx Context, name string, publicID string, systemID string) error {
	if h := s.InternalSubsetHandler; h != nil {
		return h(ctx, name, publicID, systemID)
	}
	return ErrHandlerUnspecified
}

// NotationDecl satisfies the DTDHandler interface
func (s *SAX2) NotationDecl(ctx Context, name string, publicID string, systemID string) error {
	if h := s.NotationDeclHandler; h != nil {
		return h(ctx, name, publicID, systemID)
	}
	return ErrHandlerUnspecified
}

// ProcessingInstruction satisfies the ContentHandler interface
func (s *SAX2) ProcessingInstruction(ctx Context, target string, data string) error {
	if h := s.ProcessingInstructionHandler; h != nil {
		return h(ctx, target, data)
	}
	return ErrHandlerUnspecified
}

// Reference satisfies the Extensions interface
func (s *SAX2) Reference(ctx Context, name string) error {
	if h := s.ReferenceHandler; h != nil {
		return h(ctx, name)
	}
	return ErrHandlerUnspecified
}

// ResolveEntity satisfies the EntityResolver interface
func (s *SAX2) ResolveEntity(ctx Context, name string, publicID string, baseURI string, systemID string) (Entity, error) {
	if h := s.ResolveEntityHandler; h != nil {
		return h(ctx, name, publicID, baseURI, systemID)
	}
	return nil, ErrHandlerUnspecified
}

// SetDocumentLocator satisfies the ContentHandler interface
func (s *SAX2) SetDocumentLocator(ctx Context, loc DocumentLocator) error {
	if h := s.SetDocumentLocatorHandler; h != nil {
		return h(ctx, loc)
	}
	return ErrHandlerUnspecified
}

// SkippedEntity satisfies the ContentHandler interface
func (s *SAX2) SkippedEntity(ctx Context, name string) error {
	if h := s.SkippedEntityHandler; h != nil {
		return h(ctx, name)
	}
	return ErrHandlerUnspecified
}

// StartCDATA satisfies the LexicalHandler interface
func (s *SAX2) StartCDATA(ctx Context) error {
	if h := s.StartCDATAHandler; h != nil {
		return h(ctx)
	}
	return ErrHandlerUnspecified
}

// StartDTD satisfies the LexicalHandler interface
func (s *SAX2) StartDTD(ctx Context, name string, publicID string, systemID string) error {
	if h := s.StartDTDHandler; h != nil {
		return h(ctx, name, publicID, systemID)
	}
	return ErrHandlerUnspecified
}

// StartDocument satisfies the ContentHandler interface
func (s *SAX2) StartDocument(ctx Context) error {
	if h := s.StartDocumentHandler; h != nil {
		return h(ctx)
	}
	return ErrHandlerUnspecified
}

// StartElement satisfies the ContentHandler interface
func (s *SAX2) StartElement(ctx Context, elem ParsedElement) error {
	if h := s.StartElementHandler; h != nil {
		return h(ctx, elem)
	}
	return ErrHandlerUnspecified
}

// StartEntity satisfies the LexicalHandler interface
func (s *SAX2) StartEntity(ctx Context, name string) error {
	if h := s.StartEntityHandler; h != nil {
		return h(ctx, name)
	}
	return ErrHandlerUnspecified
}

// UnparsedEntityDecl satisfies the DTDHandler interface
func (s *SAX2) UnparsedEntityDecl(ctx Context, name string, typ int, publicID string, systemID string, notation string) error {
	if h := s.UnparsedEntityDeclHandler; h != nil {
		return h(ctx, name, typ, publicID, systemID, notation)
	}
	return ErrHandlerUnspecified
}

