package xpath3

import (
	"fmt"
	"sync"
	"testing"
)

// TestRegexLRUCacheBounded asserts the LRU cache never exceeds its capacity and
// evicts least-recently-used entries.
func TestRegexLRUCacheBounded(t *testing.T) {
	c := newRegexLRUCache(8)
	for i := range 1000 {
		key := xpathRegexCacheKey{pattern: fmt.Sprintf("p%d", i)}
		c.LoadOrStore(key, &compiledXPathRegex{})
		if got := c.len(); got > 8 {
			t.Fatalf("cache size %d exceeded capacity 8 after %d inserts", got, i+1)
		}
	}
	if got := c.len(); got != 8 {
		t.Fatalf("expected cache to be full at capacity 8, got %d", got)
	}

	// Oldest entries must have been evicted; only the most recent 8 remain.
	for i := 992; i < 1000; i++ {
		key := xpathRegexCacheKey{pattern: fmt.Sprintf("p%d", i)}
		if _, ok := c.Load(key); !ok {
			t.Fatalf("expected recent key p%d to be present", i)
		}
	}
	if _, ok := c.Load(xpathRegexCacheKey{pattern: "p0"}); ok {
		t.Fatal("expected oldest key p0 to have been evicted")
	}
}

// TestRegexLRUCacheLoadOrStoreSemantics verifies LoadOrStore returns the
// existing value and reports loaded=true on a hit.
func TestRegexLRUCacheLoadOrStoreSemantics(t *testing.T) {
	c := newRegexLRUCache(4)
	key := xpathRegexCacheKey{pattern: "a", flags: "i"}
	first := &compiledXPathRegex{}
	got, loaded := c.LoadOrStore(key, first)
	if loaded || got != first {
		t.Fatalf("first store: loaded=%v got=%p want first=%p", loaded, got, first)
	}
	second := &compiledXPathRegex{}
	got, loaded = c.LoadOrStore(key, second)
	if !loaded || got != first {
		t.Fatalf("second store: loaded=%v got=%p want first=%p", loaded, got, first)
	}
}

// TestRegexLRUCacheConcurrent exercises concurrent access and asserts the bound
// holds under contention (run with -race).
func TestRegexLRUCacheConcurrent(t *testing.T) {
	c := newRegexLRUCache(16)
	var wg sync.WaitGroup
	for g := range 8 {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := range 500 {
				key := xpathRegexCacheKey{pattern: fmt.Sprintf("g%d-p%d", g, i)}
				c.LoadOrStore(key, &compiledXPathRegex{})
				c.Load(key)
			}
		}(g)
	}
	wg.Wait()
	if got := c.len(); got > 16 {
		t.Fatalf("cache size %d exceeded capacity 16 after concurrent use", got)
	}
}

// TestCompileXPathRegexCacheBounded asserts the production compile path keeps the
// shared cache bounded even when fed many distinct dynamic patterns.
func TestCompileXPathRegexCacheBounded(t *testing.T) {
	for i := range xpathRegexCacheCapacity * 4 {
		pattern := fmt.Sprintf("abc%d.*", i)
		if _, err := compileXPathRegex(pattern, ""); err != nil {
			t.Fatalf("compileXPathRegex(%q) failed: %v", pattern, err)
		}
	}
	if got := compiledXPathRegexCache.len(); got > xpathRegexCacheCapacity {
		t.Fatalf("shared regex cache grew to %d, exceeding bound %d", got, xpathRegexCacheCapacity)
	}
}
