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
type Enumeration any

// ElementContent represents a content model from an ELEMENT declaration
// (libxml2: xmlElementContent). The helium root package provides the
// concrete implementation; this interface allows the SAX layer to pass
// it without a circular import.
type ElementContent any

// Namespace is a namespace declaration in scope for an element, exposed to
// StartElementNS handlers. Prefix returns the declared prefix (empty for the
// default namespace) and URI returns the namespace name it binds to.
type Namespace interface {
	Prefix() string
	URI() string
}

// Attribute is a single attribute passed to StartElementNS handlers.
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
//
// Each type is the signature of one SAX2 callback. The parser invokes the
// callback at the corresponding point in the parse; returning a non-nil error
// aborts parsing. Byte slices passed to a callback are owned by the parser and
// must be copied if retained beyond the call.

// AttributeDeclFunc is called for an attribute declaration in a DTD (<!ATTLIST>).
type AttributeDeclFunc func(ctx context.Context, elem string, fullname string, typ enum.AttributeType, def enum.AttributeDefault, defaultValue string, tree Enumeration) error

// CDataBlockFunc is called for the content of a CDATA section.
type CDataBlockFunc func(ctx context.Context, value []byte) error

// CharactersFunc is called for a run of character data.
type CharactersFunc func(ctx context.Context, ch []byte) error

// CommentFunc is called for a comment's content.
type CommentFunc func(ctx context.Context, value []byte) error

// ElementDeclFunc is called for an element declaration in a DTD (<!ELEMENT>).
type ElementDeclFunc func(ctx context.Context, name string, typ enum.ElementType, content ElementContent) error

// EndDocumentFunc is called once when the end of the document is reached.
type EndDocumentFunc func(ctx context.Context) error

// EndElementNSFunc is called for an element's end tag, with the resolved
// namespace components of its name.
type EndElementNSFunc func(ctx context.Context, localname string, prefix string, uri string) error

// EntityDeclFunc is called for an entity declaration in a DTD.
type EntityDeclFunc func(ctx context.Context, name string, typ enum.EntityType, publicID string, systemID string, content string) error

// ErrorFunc is called for a parse error.
type ErrorFunc func(ctx context.Context, err error) error

// ExternalSubsetFunc is called for the external DTD subset reference.
type ExternalSubsetFunc func(ctx context.Context, name string, externalID string, systemID string) error

// GetEntityFunc is called to resolve a general entity by name.
type GetEntityFunc func(ctx context.Context, name string) (Entity, error)

// GetParameterEntityFunc is called to resolve a parameter entity by name.
type GetParameterEntityFunc func(ctx context.Context, name string) (Entity, error)

// HasExternalSubsetFunc is called to query whether the document has an external
// DTD subset.
type HasExternalSubsetFunc func(ctx context.Context) (bool, error)

// HasInternalSubsetFunc is called to query whether the document has an internal
// DTD subset.
type HasInternalSubsetFunc func(ctx context.Context) (bool, error)

// IgnorableWhitespaceFunc is called for ignorable whitespace per the content
// model.
type IgnorableWhitespaceFunc func(ctx context.Context, ch []byte) error

// InternalSubsetFunc is called for the internal DTD subset (<!DOCTYPE>).
type InternalSubsetFunc func(ctx context.Context, name string, externalID string, systemID string) error

// IsStandaloneFunc is called to query the document's standalone status.
type IsStandaloneFunc func(ctx context.Context) (bool, error)

// NotationDeclFunc is called for a notation declaration in a DTD (<!NOTATION>).
type NotationDeclFunc func(ctx context.Context, name string, publicID string, systemID string) error

// ProcessingInstructionFunc is called for a processing instruction.
type ProcessingInstructionFunc func(ctx context.Context, target string, data string) error

// ReferenceFunc is called for an entity reference that is not expanded inline.
type ReferenceFunc func(ctx context.Context, name string) error

// ResolveEntityFunc controls the loading of external entities.
// The application can either override this callback in the SAX block
// or use a custom entity resolution routine.
type ResolveEntityFunc func(ctx context.Context, publicID string, systemID string) (ParseInput, error)

// SetDocumentLocatorFunc is called once before any other callback to supply the
// source-position locator.
type SetDocumentLocatorFunc func(ctx context.Context, locator DocumentLocator) error

// StartDocumentFunc is called once at the start of the document.
type StartDocumentFunc func(ctx context.Context) error

// StartElementNSFunc is called for an element's start tag, with the resolved
// namespace components of its name plus its in-scope namespace declarations and
// attributes.
type StartElementNSFunc func(ctx context.Context, localname string, prefix string, uri string, namespaces []Namespace, attrs []Attribute) error

// UnparsedEntityDeclFunc is called for an unparsed entity declaration (one with
// an associated notation).
type UnparsedEntityDeclFunc func(ctx context.Context, name string, publicID string, systemID string, notationName string) error

// WarningFunc is called for a non-fatal parse warning.
type WarningFunc func(ctx context.Context, err error) error
