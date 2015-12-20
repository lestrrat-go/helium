package sax

// Handler is the interface defining the SAX2 handler.
// The first argument is always an opaque context value, which can
// be registered to the helium parser (XXX: note yet implemented).
type Handler interface {
	StartDocument(interface{}) error
	EndDocument(interface{}) error
	ProcessingInstruction(interface{}, string, string) error
	StartElement(interface{}, ParsedElement) error
	EndElement(interface{}, ParsedElement) error
	Characters(interface{}, []byte) error
	CDATABlock(interface{}, []byte) error
	Comment(interface{}, []byte) error
	InternalSubset(interface{}, string, string, string) error
	GetParameterEntity(interface{}, string) (string, error)
}

type ParsedElement interface {
	Prefix() string
	LocalName() string
	Attributes() []ParsedAttribute
}

type ParsedAttribute interface {
	Prefix() string
	LocalName() string
	Value() string
}

type StartDocumentFunc func(interface{}) error
type EndDocumentFunc func(interface{}) error
type ProcessingInstructionFunc func(interface{}, string, string) error
type StartElementFunc func(interface{}, ParsedElement) error
type EndElementFunc func(interface{}, ParsedElement) error
type CharactersFunc func(interface{}, []byte) error
type CDATABlockFunc func(interface{}, []byte) error
type CommentFunc func(interface{}, []byte) error
type InternalSubsetFunc func(interface{}, string, string, string) error
type GetParameterEntityFunc func(interface{}, string) (string, error)

type SAX2 struct {
	CDATABlockHandler            CDATABlockFunc
	CharactersHandler            CharactersFunc
	CommentHandler               CommentFunc
	EndDocumentHandler           EndDocumentFunc
	EndElementHandler            EndElementFunc
	ProcessingInstructionHandler ProcessingInstructionFunc
	StartDocumentHandler         StartDocumentFunc
	StartElementHandler          StartElementFunc
	InternalSubsetHandler        InternalSubsetFunc
	GetParameterEntityHandler    GetParameterEntityFunc
}
