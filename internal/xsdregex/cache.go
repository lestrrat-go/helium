package xsdregex

import (
	"container/list"
	"sync"
)

// patternCacheCapacity bounds the number of compiled pattern facets retained
// across the process. Schemas with many distinct patterns would otherwise keep
// every compiled regex for the process lifetime; past the cap the least recently
// used entry is evicted.
const patternCacheCapacity = 1024

// compiledPatternCache is the process-wide cache CompileVersion consults. It
// holds only RE2-engine results (Regexp.std set), keyed by pattern text and XSD
// version:
//
//   - A *Regexp is never written after CompileVersion returns it, and
//     MatchString only reads it. Its *regexp.Regexp is documented as safe for
//     concurrent use except for configuration methods such as Longest, which
//     this package never calls. One entry can therefore be shared by every
//     schema and goroutine.
//   - regexp2-engine results are not stored: CompileVersion copies
//     DefaultMatchTimeout into each one at compile time, and
//     SetDefaultMatchTimeout promises to affect every pattern compiled after it.
//   - Errors are not stored: each compile recomputes the diagnostic text.
var compiledPatternCache = newPatternCache(patternCacheCapacity)

type patternCacheKey struct {
	pattern string
	xsd11   bool
}

// patternCache is a mutex-guarded, bounded least-recently-used map from
// patternCacheKey to compiled *Regexp.
type patternCache struct {
	mu    sync.Mutex
	cap   int
	ll    *list.List // front = most recently used
	items map[patternCacheKey]*list.Element
}

type patternCacheEntry struct {
	key   patternCacheKey
	value *Regexp
}

func newPatternCache(capacity int) *patternCache {
	if capacity < 1 {
		capacity = 1
	}
	return &patternCache{
		cap:   capacity,
		ll:    list.New(),
		items: make(map[patternCacheKey]*list.Element, capacity),
	}
}

// load returns the cached value for key and marks it most recently used.
func (c *patternCache) load(key patternCacheKey) (*Regexp, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.items[key]
	if !ok {
		return nil, false
	}
	c.ll.MoveToFront(el)
	return el.Value.(*patternCacheEntry).value, true //nolint:forcetypeassert
}

// loadOrStore returns the value already cached for key, or stores value and
// returns it. Two goroutines that miss on the same key at once both compile,
// and both receive the value stored first. Storing past the capacity evicts
// the least recently used entry.
func (c *patternCache) loadOrStore(key patternCacheKey, value *Regexp) *Regexp {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		c.ll.MoveToFront(el)
		return el.Value.(*patternCacheEntry).value //nolint:forcetypeassert
	}
	c.items[key] = c.ll.PushFront(&patternCacheEntry{key: key, value: value})
	if c.ll.Len() > c.cap {
		oldest := c.ll.Back()
		c.ll.Remove(oldest)
		delete(c.items, oldest.Value.(*patternCacheEntry).key) //nolint:forcetypeassert
	}
	return value
}

// len reports the number of cached entries.
func (c *patternCache) len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ll.Len()
}
