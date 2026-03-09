package sax

import "context"

type contextKey int

const (
	documentLocatorKey contextKey = iota
)

// SetDocumentLocatorValue stores a DocumentLocator in the context, returning a new context.
func SetDocumentLocatorValue(ctx context.Context, loc DocumentLocator) context.Context {
	return context.WithValue(ctx, documentLocatorKey, loc)
}

// GetDocumentLocator retrieves the DocumentLocator from ctx. Returns nil if none is set.
func GetDocumentLocator(ctx context.Context) DocumentLocator {
	loc, _ := ctx.Value(documentLocatorKey).(DocumentLocator)
	return loc
}
