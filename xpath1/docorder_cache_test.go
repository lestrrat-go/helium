package xpath1

import (
	"testing"

	ixpath "github.com/lestrrat-go/helium/internal/xpath"
	"github.com/stretchr/testify/require"
)

func TestEvalContextUsesInternalDocOrderCacheFromContext(t *testing.T) {
	defaultCtx := newEvalContextWithConfig(t.Context(), nil, nil)
	require.NotNil(t, defaultCtx.docOrder)

	cache := &ixpath.DocOrderCache{}
	ctx := ixpath.WithDocOrderCache(t.Context(), cache)
	cachedCtx := newEvalContextWithConfig(ctx, nil, nil)
	require.Same(t, cache, cachedCtx.docOrder)
	require.NotSame(t, defaultCtx.docOrder, cachedCtx.docOrder)

	clone := cachedCtx.withNode(nil, 1, 1)
	require.Same(t, cache, clone.docOrder)
}
