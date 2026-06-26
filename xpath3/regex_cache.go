package xpath3

import (
	"container/list"
	"sync"
)

// xpathRegexCacheCapacity bounds the number of distinct compiled XPath regexes
// retained across the process. Repeated matches/replace/tokenize/analyze-string
// calls with many distinct dynamic patterns would otherwise grow the cache
// without limit, retaining every compiled regex for the process lifetime and
// presenting an unbounded cross-request memory-growth vector. The bound keeps
// the perf benefit for hot/repeated patterns while capping worst-case
// retention; least-recently-used entries are evicted past the cap.
const xpathRegexCacheCapacity = 1024

// regexLRUCache is a concurrency-safe, bounded least-recently-used cache mapping
// pattern+flags keys to compiled XPath regexes. It is a drop-in replacement for
// the sync.Map previously used, exposing Load/LoadOrStore with the same
// semantics but with a hard size bound and LRU eviction.
type regexLRUCache struct {
	mu    sync.Mutex
	cap   int
	ll    *list.List // front = most recently used
	items map[xpathRegexCacheKey]*list.Element
}

type regexLRUEntry struct {
	key   xpathRegexCacheKey
	value *compiledXPathRegex
}

func newRegexLRUCache(capacity int) *regexLRUCache {
	if capacity < 1 {
		capacity = 1
	}
	return &regexLRUCache{
		cap:   capacity,
		ll:    list.New(),
		items: make(map[xpathRegexCacheKey]*list.Element, capacity),
	}
}

// Load returns the cached compiled regex for key, marking it most-recently-used.
func (c *regexLRUCache) Load(key xpathRegexCacheKey) (*compiledXPathRegex, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.items[key]
	if !ok {
		return nil, false
	}
	c.ll.MoveToFront(el)
	return el.Value.(*regexLRUEntry).value, true //nolint:forcetypeassert
}

// LoadOrStore returns the existing value for key if present (marking it
// most-recently-used); otherwise it stores value and returns it. It mirrors
// sync.Map.LoadOrStore: the returned bool reports whether the value was loaded.
// Inserting past the capacity evicts the least-recently-used entry.
func (c *regexLRUCache) LoadOrStore(key xpathRegexCacheKey, value *compiledXPathRegex) (*compiledXPathRegex, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		c.ll.MoveToFront(el)
		return el.Value.(*regexLRUEntry).value, true //nolint:forcetypeassert
	}
	el := c.ll.PushFront(&regexLRUEntry{key: key, value: value})
	c.items[key] = el
	if c.ll.Len() > c.cap {
		if oldest := c.ll.Back(); oldest != nil {
			c.ll.Remove(oldest)
			delete(c.items, oldest.Value.(*regexLRUEntry).key) //nolint:forcetypeassert
		}
	}
	return value, false
}

// len reports the current number of cached entries (used in tests).
func (c *regexLRUCache) len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ll.Len()
}
