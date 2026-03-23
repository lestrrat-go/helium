package xpath3

import (
	"context"
	"io"

	"github.com/lestrrat-go/helium"
	ixpath "github.com/lestrrat-go/helium/internal/xpath"
)

// fnContextKey is used exclusively by the unexported withFnContext/getFnContext
// pair to pass evalContext to built-in function implementations. It is never
// exposed to external callers.
type fnContextKey struct{}

// dynamicCallKey marks a function call as originating from a dynamic
// function reference (e.g. $f(args) where $f holds a named function ref).
type dynamicCallKey struct{}

func withDynamicCall(ctx context.Context) context.Context {
	return context.WithValue(ctx, dynamicCallKey{}, true)
}

// IsDynamicCall returns true if the current function call was dispatched
// through a dynamic function reference (NamedFunctionRef → FunctionItem).
func IsDynamicCall(ctx context.Context) bool {
	v, _ := ctx.Value(dynamicCallKey{}).(bool)
	return v
}

// QualifiedName identifies a function in a specific namespace.
type QualifiedName struct {
	URI  string
	Name string
}

// URIResolver resolves URIs to readable content for fn:unparsed-text and fn:doc.
// The resolved URI is the absolute URI after base URI resolution.
type URIResolver interface {
	ResolveURI(uri string) (io.ReadCloser, error)
}

// CollectionResolver resolves fn:collection and fn:uri-collection lookups.
// The empty string identifies the default collection.
type CollectionResolver interface {
	ResolveCollection(uri string) (Sequence, error)
	ResolveURICollection(uri string) ([]string, error)
}

// VariableResolver provides lazy variable resolution for variables not found
// in the static variable scope. This is used by the XSLT executor to lazily
// evaluate global variables on demand.
type VariableResolver interface {
	ResolveVariable(ctx context.Context, name string) (Sequence, bool, error)
}

// FunctionResolver provides lazy function resolution for functions not found
// in the static function scope. Unlike fnsNS, functions registered via this
// interface are NOT discoverable via fn:function-lookup — they are only
// resolved for direct function calls. This is used by xslt3 to resolve
// xsl:original() without exposing it to function-lookup.
type FunctionResolver interface {
	ResolveFunction(ctx context.Context, uri, name string, arity int) (Function, bool, error)
}

// SchemaDeclarations provides schema element/attribute/type lookup for
// schema-element(), schema-attribute() node tests and schema-aware casting.
type SchemaDeclarations interface {
	LookupSchemaElement(local, ns string) (typeName string, ok bool)
	LookupSchemaAttribute(local, ns string) (typeName string, ok bool)
	LookupSchemaType(local, ns string) (baseType string, ok bool)
	// IsSubtypeOf returns true if typeName is the same as or a subtype of baseTypeName.
	// Both names use the annotation format: "xs:localName" for XSD built-ins,
	// "Q{ns}localName" for user-defined types.
	IsSubtypeOf(typeName, baseTypeName string) bool
	// ValidateCast checks whether a string value is valid for a user-defined
	// schema type (including facet constraints). Returns nil if valid or the
	// type is not found; returns an error if the value violates facets.
	ValidateCast(value, typeName string) error
	// ListItemType returns the item type name for a list type. If the type
	// is not a list, returns ("", false).
	ListItemType(typeName string) (itemType string, ok bool)
	// UnionMemberTypes returns the member type names for a union type.
	// If the type is not a union, returns nil.
	UnionMemberTypes(typeName string) []string
}

// withFnContext stores the evalContext in a context.Context so built-in
// functions can access the evaluation state (position, size, context node).
func withFnContext(ctx context.Context, ec *evalContext) context.Context {
	return context.WithValue(ctx, fnContextKey{}, ec)
}

// getFnContext retrieves the evalContext stashed by the evaluator.
// Returns nil if not in an evaluation.
func getFnContext(ctx context.Context) *evalContext {
	ec, _ := ctx.Value(fnContextKey{}).(*evalContext)
	return ec
}

// FnContextNode returns the current XPath context node from a function call
// context. This is the context.Context passed to Function.Call by the evaluator.
// Returns nil if the context does not carry an evaluation state or the context
// item is not a node.
func FnContextNode(ctx context.Context) helium.Node {
	ec := getFnContext(ctx)
	if ec == nil {
		return nil
	}
	return ec.node
}

// DocOrderCache is a shared document-order cache that can be passed
// across evaluations to ensure consistent cross-document ordering.
type DocOrderCache = ixpath.DocOrderCache

// NewDocOrderCache creates a new shared document-order cache.
func NewDocOrderCache() *DocOrderCache {
	return &ixpath.DocOrderCache{}
}

