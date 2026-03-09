package sax

import "context"

type contextKey int

const (
	documentLocatorKey contextKey = iota
	stopperKey
)

// ContextOption configures a SAX context.
type ContextOption func(context.Context) context.Context

// NewContext creates a context.Context primed with SAX-specific values.
func NewContext(parent context.Context, opts ...ContextOption) context.Context {
	ctx := parent
	for _, opt := range opts {
		ctx = opt(ctx)
	}
	return ctx
}

// WithDocumentLocator returns a ContextOption that stores a DocumentLocator.
func WithDocumentLocator(loc DocumentLocator) ContextOption {
	return func(ctx context.Context) context.Context {
		return context.WithValue(ctx, documentLocatorKey, loc)
	}
}

// SetDocumentLocatorValue stores a DocumentLocator in the context, returning a new context.
func SetDocumentLocatorValue(ctx context.Context, loc DocumentLocator) context.Context {
	return context.WithValue(ctx, documentLocatorKey, loc)
}

// GetDocumentLocator retrieves the DocumentLocator from ctx. Returns nil if none is set.
func GetDocumentLocator(ctx context.Context) DocumentLocator {
	loc, _ := ctx.Value(documentLocatorKey).(DocumentLocator)
	return loc
}

type parserStopper interface {
	StopParser()
}

// WithParserStopper returns a ContextOption that stores a stop-parser callback.
func WithParserStopper(s parserStopper) ContextOption {
	return func(ctx context.Context) context.Context {
		return context.WithValue(ctx, stopperKey, s)
	}
}

// StopParser signals the parser to stop. No-op if no stopper is present.
func StopParser(ctx context.Context) {
	if s, _ := ctx.Value(stopperKey).(parserStopper); s != nil {
		s.StopParser()
	}
}
