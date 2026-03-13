package sax

import "context"

type documentLocatorKey struct{}

// WithDocumentLocator stores a DocumentLocator in the returned context.
func WithDocumentLocator(ctx context.Context, loc DocumentLocator) context.Context {
	return context.WithValue(ctx, documentLocatorKey{}, loc)
}

// GetDocumentLocator retrieves the DocumentLocator from ctx. Returns nil if none is set.
func GetDocumentLocator(ctx context.Context) DocumentLocator {
	if ctx == nil {
		return nil
	}
	loc, _ := ctx.Value(documentLocatorKey{}).(DocumentLocator)
	return loc
}
