package sax

// Context is always passed as the first argument to SAX handlers.
// It is intentionally left as an opaque value so applications can
// use type assertions to pass whatever object they need to pass.
type Context interface{}

// TODO fix Context
type DocumentLocator interface{}

// TODO
type ParseInput interface{}

// TODO fix Context
type Entity interface {
	Name() string
	SetOrig(string)
	EntityType() int
	Content() []byte
	Checked() bool
	MarkChecked()
}


type AttributeDefaultValue interface{}
type Enumeration interface{}
type ElementContent interface{}

type Namespace interface {
	Prefix() string
	URI() string
}

type Attribute interface {
	LocalName() string
	// Name returns the fully qualified name. That is, if the attribute has
	// a namespace prefix associated with it, it will return "prefix:localname"
	// and "localname" otherwise
	Name() string
	Prefix() string
	Value() string
	IsDefault() bool
}

// SAX functions

type AttributeDeclFunc func(ctx Context, elem string, fullname string, typ int, def int, defaultValue string, tree Enumeration) error
type CDataBlockFunc func(ctx Context, value []byte) error
type CharactersFunc func(ctx Context, ch []byte) error
type CommentFunc func(ctx Context, value []byte) error
type ElementDeclFunc func(ctx Context, name string, typ int, content ElementContent) error
type EndDocumentFunc func(ctx Context) error
type EndElementNSFunc func(ctx Context, localname string, prefix string, uri string) error
type EntityDeclFunc func(ctx Context, name string, typ int, publicID string, systemID string, content string) error
type ErrorFunc func(ctx Context, message string, args ...interface{}) error
type ExternalSubsetFunc func(ctx Context, name string, externalID string, systemID string) error
type GetEntityFunc func(ctx Context, name string) (Entity, error)
type GetParameterEntityFunc func(ctx Context, name string) (Entity, error)
type HasExternalSubsetFunc func(ctx Context) (bool, error)
type HasInternalSubsetFunc func(ctx Context) (bool, error)
type IgnorableWhitespaceFunc func(ctx Context, ch []byte) error
type InternalSubsetFunc func(ctx Context, name string, externalID string, systemID string) error
type IsStandaloneFunc func(ctx Context) (bool, error)
type NotationDeclFunc func(ctx Context, name string, publicID string, systemID string) error
type ProcessingInstructionFunc func(ctx Context, target string, data string) error
type ReferenceFunc func(ctx Context, name string) error

/*
 * The entity loader, to control the loading of external entities,
 * the application can either:
 *    - override this resolveEntity() callback in the SAX block
 *    - or better use the xmlSetExternalEntityLoader() function to
 *      set up it's own entity resolution routine
 *
 * Returns the xmlParserInputPtr if inlined or NULL for DOM behaviour.
 */
type ResolveEntityFunc func(ctx Context, publicID string, systemID string) (ParseInput, error)
type SetDocumentLocatorFunc func(ctx Context, locator DocumentLocator) error
type StartDocumentFunc func(ctx Context) error
type StartElementNSFunc func(ctx Context, localname string, prefix string, uri string, namespaces []Namespace, attrs []Attribute) error
type UnparsedEntityDeclFunc func(ctx Context, name string, publicID string, systemID string, notationName string) error

