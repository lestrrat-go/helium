package sax

import (
	"context"

	"github.com/lestrrat-go/helium/enum"
)

// DocumentLocator provides document position information to SAX handlers.
// The parser passes an implementation to SetDocumentLocator at the start
// of parsing; handlers can query it during callbacks for source location.
type DocumentLocator interface {
	LineNumber() int
	ColumnNumber() int
	// GetPublicID returns the public identifier of the document being parsed (libxml2: xmlSAXLocator.getPublicId).
	// In practice this always returns an empty string (libxml2 always returns NULL).
	GetPublicID() string
	// GetSystemID returns the system identifier (URI/filename) of the document being parsed (libxml2: xmlSAXLocator.getSystemId).
	GetSystemID() string
}

// ParseInput represents an input source for the parser, typically used
// for external entity resolution. Implementations must provide an
// io.Reader for the content. The URI method returns the resolved URI
// of the input source (used for relative URI resolution).
type ParseInput interface {
	Read(p []byte) (int, error)
	URI() string
}

// Entity represents a parsed or unparsed entity declaration.
type Entity interface {
	Name() string
	SetOrig(string)
	EntityType() enum.EntityType
	Content() []byte
	// Checked reports whether this entity's content has been parsed (libxml2: ent->checked).
	Checked() bool
	// MarkChecked marks this entity as having been parsed (libxml2: ent->checked).
	MarkChecked()
}

// Enumeration represents a list of allowed attribute values in an
// ATTLIST declaration (libxml2: xmlEnumeration). The helium root
// package provides the concrete []string implementation; this
// interface allows the SAX layer to pass it without a circular import.
type Enumeration interface{}

// ElementContent represents a content model from an ELEMENT declaration
// (libxml2: xmlElementContent). The helium root package provides the
// concrete implementation; this interface allows the SAX layer to pass
// it without a circular import.
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

type AttributeDeclFunc func(ctx context.Context, elem string, fullname string, typ enum.AttributeType, def enum.AttributeDefault, defaultValue string, tree Enumeration) error
type CDataBlockFunc func(ctx context.Context, value []byte) error
type CharactersFunc func(ctx context.Context, ch []byte) error
type CommentFunc func(ctx context.Context, value []byte) error
type ElementDeclFunc func(ctx context.Context, name string, typ enum.ElementType, content ElementContent) error
type EndDocumentFunc func(ctx context.Context) error
type EndElementNSFunc func(ctx context.Context, localname string, prefix string, uri string) error
type EntityDeclFunc func(ctx context.Context, name string, typ enum.EntityType, publicID string, systemID string, content string) error
type ErrorFunc func(ctx context.Context, err error) error
type ExternalSubsetFunc func(ctx context.Context, name string, externalID string, systemID string) error
type GetEntityFunc func(ctx context.Context, name string) (Entity, error)
type GetParameterEntityFunc func(ctx context.Context, name string) (Entity, error)
type HasExternalSubsetFunc func(ctx context.Context) (bool, error)
type HasInternalSubsetFunc func(ctx context.Context) (bool, error)
type IgnorableWhitespaceFunc func(ctx context.Context, ch []byte) error
type InternalSubsetFunc func(ctx context.Context, name string, externalID string, systemID string) error
type IsStandaloneFunc func(ctx context.Context) (bool, error)
type NotationDeclFunc func(ctx context.Context, name string, publicID string, systemID string) error
type ProcessingInstructionFunc func(ctx context.Context, target string, data string) error
type ReferenceFunc func(ctx context.Context, name string) error

// ResolveEntityFunc controls the loading of external entities.
// The application can either override this callback in the SAX block
// or use a custom entity resolution routine.
type ResolveEntityFunc func(ctx context.Context, publicID string, systemID string) (ParseInput, error)
type SetDocumentLocatorFunc func(ctx context.Context, locator DocumentLocator) error
type StartDocumentFunc func(ctx context.Context) error
type StartElementNSFunc func(ctx context.Context, localname string, prefix string, uri string, namespaces []Namespace, attrs []Attribute) error
type UnparsedEntityDeclFunc func(ctx context.Context, name string, publicID string, systemID string, notationName string) error
type WarningFunc func(ctx context.Context, err error) error
