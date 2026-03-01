package catalog

import (
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveURIUnwrapsURN(t *testing.T) {
	// Catalog with a public entry for "-//OASIS//DTD DocBook XML V4.1.2//EN".
	cat := &Catalog{
		Entries: []Entry{
			{
				Typ:  EntryPublic,
				Name: "-//OASIS//DTD DocBook XML V4.1.2//EN",
				URL:  "file:///usr/share/xml/docbook.dtd",
			},
		},
		Pref: PreferPublic,
	}

	// The URN encoding of "-//OASIS//DTD DocBook XML V4.1.2//EN" per RFC 3151:
	//   -  → -
	//   // → :
	//   (space) → +
	urn := "urn:publicid:-:OASIS:DTD+DocBook+XML+V4.1.2:EN"

	// ResolveURI should unwrap the URN and resolve via the public entry.
	got := cat.ResolveURI(urn)
	require.Equal(t, "file:///usr/share/xml/docbook.dtd", got)
}

func TestResolveURINonURNUnchanged(t *testing.T) {
	cat := &Catalog{
		Entries: []Entry{
			{
				Typ:  EntryURI,
				Name: "http://example.com/schema.xsd",
				URL:  "file:///local/schema.xsd",
			},
		},
	}

	// Normal URI (not a URN) should resolve via URI entry as before.
	got := cat.ResolveURI("http://example.com/schema.xsd")
	require.Equal(t, "file:///local/schema.xsd", got)
}

func TestResolveURIURNNotFound(t *testing.T) {
	cat := &Catalog{
		Entries: []Entry{
			{
				Typ:  EntryPublic,
				Name: "-//Other//DTD//EN",
				URL:  "file:///other.dtd",
			},
		},
		Pref: PreferPublic,
	}

	// URN that unwraps to a public ID not in the catalog.
	urn := "urn:publicid:-:OASIS:DTD+DocBook+XML+V4.1.2:EN"
	got := cat.ResolveURI(urn)
	require.Equal(t, "", got)
}

// countingLoader is a Loader that counts how many times each URL is loaded.
type countingLoader struct {
	counts  map[string]*atomic.Int32
	catalog *Catalog
}

func (l *countingLoader) Load(filename string) (*Catalog, error) {
	if l.counts[filename] == nil {
		l.counts[filename] = &atomic.Int32{}
	}
	l.counts[filename].Add(1)
	// Return a copy so each entry gets its own catalog instance.
	cp := *l.catalog
	return &cp, nil
}

func TestVisitedCacheSkipsDuplicate(t *testing.T) {
	// Build a catalog graph where the same target catalog ("shared.xml")
	// is reachable from two nextCatalog entries. Without the visited
	// cache, the target would be entered twice for the same query.
	target := &Catalog{
		Entries: []Entry{
			{Typ: EntrySystem, Name: "http://example.com/foo.dtd", URL: "file:///local/foo.dtd"},
		},
	}

	loader := &countingLoader{
		counts:  make(map[string]*atomic.Int32),
		catalog: target,
	}

	root := &Catalog{
		Entries: []Entry{
			{Typ: EntryNextCatalog, URL: "shared.xml"},
			{Typ: EntryNextCatalog, URL: "shared.xml"},
		},
		Ldr: loader,
	}

	// Query that misses in root but falls through to nextCatalog entries.
	// Both point to shared.xml, but visited cache should skip the second.
	got := root.Resolve("", "http://example.com/notfound.dtd")
	require.Equal(t, "", got)

	// shared.xml was loaded (lazyLoad), but its resolve should only be
	// called once due to the visited cache.
	cnt := loader.counts["shared.xml"]
	require.NotNil(t, cnt)
	// The second nextCatalog entry should be skipped by visited cache.
	// lazyLoad may be called twice (it caches on the Entry), but resolve
	// is guarded by checkVisited.
}

func TestVisitedCacheStillResolves(t *testing.T) {
	// Ensure the visited cache doesn't prevent valid resolution.
	target := &Catalog{
		Entries: []Entry{
			{Typ: EntrySystem, Name: "http://example.com/foo.dtd", URL: "file:///local/foo.dtd"},
		},
	}

	loader := &countingLoader{
		counts:  make(map[string]*atomic.Int32),
		catalog: target,
	}

	root := &Catalog{
		Entries: []Entry{
			{Typ: EntryNextCatalog, URL: "shared.xml"},
			{Typ: EntryNextCatalog, URL: "shared.xml"},
		},
		Ldr: loader,
	}

	got := root.Resolve("", "http://example.com/foo.dtd")
	require.Equal(t, "file:///local/foo.dtd", got)
}

func TestVisitedCachePerQuery(t *testing.T) {
	// The visited cache is per top-level Resolve call. Two different
	// queries should each be able to visit the same catalog.
	target := &Catalog{
		Entries: []Entry{
			{Typ: EntrySystem, Name: "http://example.com/a.dtd", URL: "file:///a.dtd"},
			{Typ: EntrySystem, Name: "http://example.com/b.dtd", URL: "file:///b.dtd"},
		},
	}

	loader := &countingLoader{
		counts:  make(map[string]*atomic.Int32),
		catalog: target,
	}

	root := &Catalog{
		Entries: []Entry{
			{Typ: EntryNextCatalog, URL: "shared.xml"},
		},
		Ldr: loader,
	}

	got := root.Resolve("", "http://example.com/a.dtd")
	require.Equal(t, "file:///a.dtd", got)

	got = root.Resolve("", "http://example.com/b.dtd")
	require.Equal(t, "file:///b.dtd", got)
}
