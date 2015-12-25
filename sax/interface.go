package sax

// TODO fix Context
type DocumentLocator interface{}

// TODO fix Context
type Entity interface {
	Name() string
	SetOrig(string)
	EntityType() int
	Content() string
}

// Context is always passed as the first argument to SAX handlers.
// It is intentionally left as an opaque value so applications can
// use type assertions to pass whatever object they need to pass.
type Context interface{}

// DTDHandler defines an interface for thos objects that would like to
// receive notification of basic DTD-related events.
// See http://sax.sourceforge.net/apidoc/org/xml/sax/DTDHandler.html
type DTDHandler interface {
	// Receive notification of a notation declaration event.
	// Parameters are the context object, name, publicID, and systemID.
	NotationDecl(ctx Context, name string, publicID string, systemID string) error

	//Receive notification of an unparsed entity declaration event.
	// Parameters are the context object, name, publicID, systemID, and notation name.
	UnparsedEntityDecl(ctx Context, name string, typ int, publicID string, systemID string, notation string) error
}

// ContentHandler is the interface defining the SAX2 handler.
// The first argument is always an opaque context value, which can
// be registered to the helium parser (XXX: not yet implemented).
type ContentHandler interface {
	// End the scope of a prefix-URI mapping.
	// EndPrefixMapping(ctx Context, string) error

	// Receive notification of ignorable whitespace in element content.
	IgnorableWhitespace(ctx Context, content []byte) error

	// Receive notification of a skipped entity.
	SkippedEntity(ctx Context, name string) error

	// Begin the scope of a prefix-URI Namespace mapping.
	// StartPrefixMapping(ctx Context, string, string) error

	// Receive an object for locating the origin of SAX document events.
	SetDocumentLocator(ctx Context, loc DocumentLocator) error
	// Receive notification of the beginning of a document.
	StartDocument(ctx Context) error
	EndDocument(ctx Context) error
	// Receive notification of a processing instruction.
	ProcessingInstruction(ctx Context, target string, data string) error
	// Receive notification of the beginning of an element.
	StartElement(ctx Context, elem ParsedElement) error
	EndElement(ctx Context, elem ParsedElement) error
	Characters(ctx Context, content []byte) error
}


type AttributeDefaultValue interface{}
type Enumeration interface{}
type ElementContent interface {}
// DeclHandler is a SAX2 extension handler for DTD declaration events.
type DeclHandler interface {
	AttributeDecl(ctx Context, elemName string, attrName string, typ int, deftype int, defvalue AttributeDefaultValue, enum Enumeration) error

	// ElementDecl is called when an element definition has been parsed.
	// Note that the signature differs from SAX2 API in http://sax.sourceforge.net/apidoc/org/xml/sax/ext/DeclHandler.html#elementDecl%28java.lang.String%2C%20java.lang.String%29
	ElementDecl(ctx Context, name string, typ int, content ElementContent) error
	ExternalEntityDecl(ctx Context, name string, publicID string, systemID string) error
	InternalEntityDecl(ctx Context, name string, value string) error
}

// LexicalHandler is SAX2 extension for lexical events
type LexicalHandler interface {
	Comment(ctx Context, content []byte) error
	EndCDATA(ctx Context) error
	EndDTD(ctx Context) error
	EndEntity(ctx Context, name string) error
	StartCDATA(ctx Context) error
	StartDTD(ctx Context, name string, publicID string, systemID string) error
	StartEntity(ctx Context, name string) error
}

// EntityResolver is an extended interface for mapping external entity
// references to input sources, or providing a missing external subset.
type EntityResolver interface {
	GetExternalSubset(ctx Context, name string, baseURI string) error
	ResolveEntity(ctx Context, name string, publicID string, baseURI string, systemID string) (Entity, error)
}

// Extensions defines some non-standard SAX extensions. This may be
// consolidaed later.
type Extensions interface {
	ExternalSubset(ctx Context, name string, publicID string, systemID string) error
	GetParameterEntity(ctx Context, nmae string) (Entity, error)
	InternalSubset(ctx Context, name string, publicID string, systemID string) error
	Reference(ctx Context, name string) error
}

type ParsedNamespace interface {
	Prefix() string
	URI() string
}

type ParsedElement interface {
	Prefix() string
	URI() string
	LocalName() string
	// Name returns the fully qualified name. That is, if the element has
	// a namespace prefix associated with it, it will return "prefix:localname"
	// and "localname" otherwise
	Name() string
	Attributes() []ParsedAttribute
	Namespaces() []ParsedNamespace
}

type ParsedAttribute interface {
	LocalName() string
	// Name returns the fully qualified name. That is, if the attribute has
	// a namespace prefix associated with it, it will return "prefix:localname"
	// and "localname" otherwise
	Name() string
	Prefix() string
	Value() string
	Defaulted() bool
}
