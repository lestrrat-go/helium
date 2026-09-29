package xsdregex_test

import (
	"errors"
	"sync"
	"testing"

	"github.com/lestrrat-go/helium/internal/xsdregex"
	"github.com/stretchr/testify/require"
)

func TestCompileVersionCache(t *testing.T) {
	t.Run("repeated compile returns the cached pointer", func(t *testing.T) {
		const pattern = `cache-hit-[a-c]+\d`
		first, err := xsdregex.CompileVersion(pattern, false)
		require.NoError(t, err)
		second, err := xsdregex.CompileVersion(pattern, false)
		require.NoError(t, err)
		require.Same(t, first, second)

		viaCompile, err := xsdregex.Compile(pattern)
		require.NoError(t, err)
		require.Same(t, first, viaCompile, "Compile shares the XSD 1.0 cache entry")
	})

	t.Run("XSD 1.0 and 1.1 are distinct entries", func(t *testing.T) {
		const pattern = `cache-version-\p{IsBasicLatin}+`
		re10, err := xsdregex.CompileVersion(pattern, false)
		require.NoError(t, err)
		re11, err := xsdregex.CompileVersion(pattern, true)
		require.NoError(t, err)
		require.NotSame(t, re10, re11)

		again10, err := xsdregex.CompileVersion(pattern, false)
		require.NoError(t, err)
		require.Same(t, re10, again10)
		again11, err := xsdregex.CompileVersion(pattern, true)
		require.NoError(t, err)
		require.Same(t, re11, again11)
	})

	t.Run("backtracking-engine patterns are not cached", func(t *testing.T) {
		patterns := []string{
			`cache-sub-[a-z-[aeiou]]+`,
			`cache-quant-a{1,5000}`,
		}
		for _, pattern := range patterns {
			first, err := xsdregex.CompileVersion(pattern, false)
			require.NoError(t, err, pattern)
			second, err := xsdregex.CompileVersion(pattern, false)
			require.NoError(t, err, pattern)
			require.NotSame(t, first, second, pattern)
		}
	})

	t.Run("invalid pattern reports the same error on every compile", func(t *testing.T) {
		const pattern = `cache-bad-(a`
		_, firstErr := xsdregex.CompileVersion(pattern, false)
		require.Error(t, firstErr)
		_, secondErr := xsdregex.CompileVersion(pattern, false)
		require.Error(t, secondErr)
		require.Equal(t, firstErr.Error(), secondErr.Error())
	})
}

// TestCompileVersionConcurrent compiles and matches the same patterns from many
// goroutines. Run under -race it checks that a cached *Regexp shared between
// goroutines is only read.
func TestCompileVersionConcurrent(t *testing.T) {
	const (
		workers    = 16
		iterations = 50
		re2Pattern = `cache-concurrent-[0-9]{2,4}`
		btPattern  = `cache-concurrent-[a-z-[aeiou]]+`
	)

	var wg sync.WaitGroup
	results := make([]*xsdregex.Regexp, workers)
	errs := make([]error, workers)
	for w := range workers {
		wg.Add(1)
		go compileAndMatchConcurrently(&wg, w, iterations, re2Pattern, btPattern, results, errs)
	}
	wg.Wait()

	for w := range workers {
		require.NoError(t, errs[w], "worker %d", w)
		require.Same(t, results[0], results[w], "worker %d got a different cached pointer", w)
	}
}

func compileAndMatchConcurrently(wg *sync.WaitGroup, w, iterations int, re2Pattern, btPattern string, results []*xsdregex.Regexp, errs []error) {
	defer wg.Done()
	for range iterations {
		re, err := xsdregex.CompileVersion(re2Pattern, false)
		if err != nil {
			errs[w] = err
			return
		}
		if !re.MatchString("cache-concurrent-123") || re.MatchString("cache-concurrent-1") {
			errs[w] = errMismatch
			return
		}
		results[w] = re

		bt, err := xsdregex.CompileVersion(btPattern, false)
		if err != nil {
			errs[w] = err
			return
		}
		if !bt.MatchString("cache-concurrent-xyz") || bt.MatchString("cache-concurrent-abc") {
			errs[w] = errMismatch
			return
		}
	}
}

var errMismatch = errors.New("unexpected match result")
