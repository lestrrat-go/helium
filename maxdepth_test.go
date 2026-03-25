package helium_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

func TestMaxDepthExceeded(t *testing.T) {
	// 10 nested <a> elements, limit set to 5
	input := []byte(strings.Repeat("<a>", 10) + strings.Repeat("</a>", 10))

	p := helium.NewParser().MaxDepth(5)

	_, err := p.Parse(context.Background(), input)
	require.Error(t, err)
	require.Contains(t, err.Error(), "exceeded max depth")
}

func TestMaxDepthWithinLimit(t *testing.T) {
	// 5 nested <a> elements, limit set to 10
	input := []byte(strings.Repeat("<a>", 5) + "hello" + strings.Repeat("</a>", 5))

	p := helium.NewParser().MaxDepth(10)

	doc, err := p.Parse(context.Background(), input)
	require.NoError(t, err)
	require.NotNil(t, doc)
}

func TestMaxDepthExactLimit(t *testing.T) {
	// 5 nested <a> elements, limit set to exactly 5
	input := []byte(strings.Repeat("<a>", 5) + "hello" + strings.Repeat("</a>", 5))

	p := helium.NewParser().MaxDepth(5)

	doc, err := p.Parse(context.Background(), input)
	require.NoError(t, err)
	require.NotNil(t, doc)
}

func TestMaxDepthZeroUnlimited(t *testing.T) {
	// No limit set (default 0) — deep nesting should succeed
	input := []byte(strings.Repeat("<a>", 100) + "hello" + strings.Repeat("</a>", 100))

	p := helium.NewParser()
	// maxDepth defaults to 0 (unlimited)

	doc, err := p.Parse(context.Background(), input)
	require.NoError(t, err)
	require.NotNil(t, doc)
}

func TestMaxDepthParseReader(t *testing.T) {
	// Verify max depth is enforced via ParseReader too
	input := strings.Repeat("<a>", 10) + strings.Repeat("</a>", 10)

	p := helium.NewParser().MaxDepth(5)

	_, err := p.ParseReader(context.Background(), bytes.NewReader([]byte(input)))
	require.Error(t, err)
	require.Contains(t, err.Error(), "exceeded max depth")
}
