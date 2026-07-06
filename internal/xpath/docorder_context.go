package xpath

import "context"

type docOrderCacheContextKey struct{}

// WithDocOrderCache returns a context that makes cache available to XPath
// evaluators in this module. It is for internal callers that can guarantee the
// described documents are not mutated while the cache is reused.
func WithDocOrderCache(ctx context.Context, cache *DocOrderCache) context.Context {
	if cache == nil {
		return ctx
	}
	return context.WithValue(ctx, docOrderCacheContextKey{}, cache)
}

// DocOrderCacheFromContext returns the document-order cache carried by ctx.
func DocOrderCacheFromContext(ctx context.Context) *DocOrderCache {
	cache, _ := ctx.Value(docOrderCacheContextKey{}).(*DocOrderCache)
	return cache
}
