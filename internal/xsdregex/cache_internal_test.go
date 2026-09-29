package xsdregex

import (
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPatternCache(t *testing.T) {
	t.Run("evicts the least recently used entry past capacity", func(t *testing.T) {
		c := newPatternCache(2)
		a := &Regexp{std: regexp.MustCompile(`a`)}
		b := &Regexp{std: regexp.MustCompile(`b`)}
		d := &Regexp{std: regexp.MustCompile(`d`)}
		keyA := patternCacheKey{pattern: "a"}
		keyB := patternCacheKey{pattern: "b"}
		keyD := patternCacheKey{pattern: "d"}

		require.Same(t, a, c.loadOrStore(keyA, a))
		require.Same(t, b, c.loadOrStore(keyB, b))
		got, ok := c.load(keyA) // a becomes most recently used
		require.True(t, ok)
		require.Same(t, a, got)

		require.Same(t, d, c.loadOrStore(keyD, d))
		require.Equal(t, 2, c.len())
		_, ok = c.load(keyB)
		require.False(t, ok, "b was least recently used and must be evicted")
		_, ok = c.load(keyA)
		require.True(t, ok)
		_, ok = c.load(keyD)
		require.True(t, ok)
	})

	t.Run("loadOrStore keeps the first stored value", func(t *testing.T) {
		c := newPatternCache(4)
		key := patternCacheKey{pattern: "x", xsd11: true}
		first := &Regexp{std: regexp.MustCompile(`x`)}
		second := &Regexp{std: regexp.MustCompile(`x`)}
		require.Same(t, first, c.loadOrStore(key, first))
		require.Same(t, first, c.loadOrStore(key, second))
		require.Equal(t, 1, c.len())
	})
}

func TestCompileVersionCacheContents(t *testing.T) {
	t.Run("invalid pattern leaves no entry", func(t *testing.T) {
		const pattern = `cache-internal-bad-[a`
		_, err := CompileVersion(pattern, false)
		require.Error(t, err)
		_, ok := compiledPatternCache.load(patternCacheKey{pattern: pattern})
		require.False(t, ok)
	})

	t.Run("backtracking pattern leaves no entry", func(t *testing.T) {
		const pattern = `cache-internal-sub-[a-z-[aeiou]]`
		_, err := CompileVersion(pattern, false)
		require.NoError(t, err)
		_, ok := compiledPatternCache.load(patternCacheKey{pattern: pattern})
		require.False(t, ok)
	})

	t.Run("RE2 pattern is stored under its version key", func(t *testing.T) {
		const pattern = `cache-internal-ok-[a-z]`
		re, err := CompileVersion(pattern, true)
		require.NoError(t, err)
		got, ok := compiledPatternCache.load(patternCacheKey{pattern: pattern, xsd11: true})
		require.True(t, ok)
		require.Same(t, re, got)
		_, ok = compiledPatternCache.load(patternCacheKey{pattern: pattern})
		require.False(t, ok)
	})

	t.Run("match timeout applies to each new backtracking compile", func(t *testing.T) {
		orig := DefaultMatchTimeout()
		defer SetDefaultMatchTimeout(orig)

		const pattern = `cache-internal-timeout-[a-z-[aeiou]]`
		SetDefaultMatchTimeout(time.Second)
		first, err := CompileVersion(pattern, false)
		require.NoError(t, err)
		require.Equal(t, time.Second, first.backtrack.MatchTimeout)

		SetDefaultMatchTimeout(2 * time.Second)
		second, err := CompileVersion(pattern, false)
		require.NoError(t, err)
		require.Equal(t, 2*time.Second, second.backtrack.MatchTimeout)
	})
}
