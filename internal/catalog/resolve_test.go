package catalog_test

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/lestrrat-go/helium/internal/catalog"
	"github.com/stretchr/testify/require"
)

const sharedCatalogXML = "shared.xml"

const missingCatalogXML = "missing.xml"

const fooDTDSystemID = "http://example.com/foo.dtd"

const exampleBase = "http://example.com/"

func TestResolveURIUnwrapsURN(t *testing.T) {
	t.Parallel()

	// Catalog with a public entry for "-//OASIS//DTD DocBook XML V4.1.2//EN".
	cat := &catalog.Catalog{
		Entries: []catalog.Entry{
			{
				Type: catalog.EntryPublic,
				Name: "-//OASIS//DTD DocBook XML V4.1.2//EN",
				URL:  "file:///usr/share/xml/docbook.dtd",
			},
		},
		Prefer: catalog.PreferPublic,
	}

	// The URN encoding of "-//OASIS//DTD DocBook XML V4.1.2//EN" per RFC 3151:
	//   -  → -
	//   // → :
	//   (space) → +
	urn := "urn:publicid:-:OASIS:DTD+DocBook+XML+V4.1.2:EN"

	// ResolveURI should unwrap the URN and resolve via the public entry.
	got := cat.ResolveURI(t.Context(), urn)
	require.Equal(t, "file:///usr/share/xml/docbook.dtd", got)
}

func TestResolveURINonURNUnchanged(t *testing.T) {
	t.Parallel()

	cat := &catalog.Catalog{
		Entries: []catalog.Entry{
			{
				Type: catalog.EntryURI,
				Name: "http://example.com/schema.xsd",
				URL:  "file:///local/schema.xsd",
			},
		},
	}

	// Normal URI (not a URN) should resolve via URI entry as before.
	got := cat.ResolveURI(t.Context(), "http://example.com/schema.xsd")
	require.Equal(t, "file:///local/schema.xsd", got)
}

func TestResolveURIURNNotFound(t *testing.T) {
	t.Parallel()

	cat := &catalog.Catalog{
		Entries: []catalog.Entry{
			{
				Type: catalog.EntryPublic,
				Name: "-//Other//DTD//EN",
				URL:  "file:///other.dtd",
			},
		},
		Prefer: catalog.PreferPublic,
	}

	// URN that unwraps to a public ID not in the catalog.
	urn := "urn:publicid:-:OASIS:DTD+DocBook+XML+V4.1.2:EN"
	got := cat.ResolveURI(t.Context(), urn)
	require.Equal(t, "", got)
}

// A catalog containing only delegateSystem entries must NOT influence URI
// resolution. System-identifier delegation belongs to the system-id path only.
func TestResolveURIIgnoresDelegateSystem(t *testing.T) {
	t.Parallel()

	delegate := &catalog.Catalog{
		Entries: []catalog.Entry{
			{Type: catalog.EntryURI, Name: "http://example.com/asset", URL: "file:///delegated/asset"},
		},
	}
	loader := &countingLoader{
		counts: make(map[string]*atomic.Int32),
		cat:    delegate,
	}

	cat := &catalog.Catalog{
		Entries: []catalog.Entry{
			{Type: catalog.EntryDelegateSystem, Name: exampleBase, URL: sharedCatalogXML},
		},
		Loader: loader,
		Prefer: catalog.PreferPublic,
	}

	// delegateSystem must not be consulted for URI resolution.
	got := cat.ResolveURI(t.Context(), "http://example.com/asset")
	require.Equal(t, "", got)
	require.Nil(t, loader.counts[sharedCatalogXML], "delegateSystem must not be loaded during ResolveURI")
}

// TestConcurrentResolveSharedCatalog exercises a single *Catalog resolved from
// many goroutines at once over delegate and nextCatalog chains. With the
// shared-receiver bug it raced (or panicked) under -race; per-resolution state
// must keep it correct and race-free.
func TestConcurrentResolveSharedCatalog(t *testing.T) {
	t.Parallel()

	// Leaf catalogs reached through the delegate / next chains.
	systemLeaf := &catalog.Catalog{
		Entries: []catalog.Entry{
			{Type: catalog.EntrySystem, Name: "http://example.com/sys.dtd", URL: "file:///sys.dtd"},
		},
	}
	publicLeaf := &catalog.Catalog{
		Entries: []catalog.Entry{
			{Type: catalog.EntryPublic, Name: "-//Example//DTD Pub//EN", URL: "file:///pub.dtd"},
		},
		Prefer: catalog.PreferPublic,
	}
	uriLeaf := &catalog.Catalog{
		Entries: []catalog.Entry{
			{Type: catalog.EntryURI, Name: "http://example.com/asset", URL: "file:///asset"},
		},
	}

	loader := &multiLoader{
		counts: make(map[string]*atomic.Int32),
		cats: map[string]*catalog.Catalog{
			"sys.xml": systemLeaf,
			"pub.xml": publicLeaf,
			"uri.xml": uriLeaf,
		},
	}

	// One shared root combining a delegate chain (system + public) and a
	// nextCatalog fallback (uri leaf).
	root := &catalog.Catalog{
		Entries: []catalog.Entry{
			{Type: catalog.EntryDelegateSystem, Name: exampleBase, URL: "sys.xml"},
			{Type: catalog.EntryDelegatePublic, Name: "-//Example//", URL: "pub.xml", Prefer: catalog.PreferPublic},
			{Type: catalog.EntryNextCatalog, URL: "uri.xml"},
		},
		Loader: loader,
		Prefer: catalog.PreferPublic,
	}

	const goroutines = 32
	const iterations = 50

	// Workers must not call require.* (it uses runtime.Goexit, which is only
	// valid on the test goroutine). Collect mismatches and assert after Wait.
	type mismatch struct {
		want string
		got  string
	}
	var (
		mu         sync.Mutex
		mismatches []mismatch
	)
	record := func(want, got string) {
		if want == got {
			return
		}
		mu.Lock()
		mismatches = append(mismatches, mismatch{want: want, got: got})
		mu.Unlock()
	}

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			ctx := context.Background()
			for range iterations {
				record("file:///sys.dtd", root.Resolve(ctx, "", "http://example.com/sys.dtd"))
				record("file:///pub.dtd", root.Resolve(ctx, "-//Example//DTD Pub//EN", ""))
				record("file:///asset", root.ResolveURI(ctx, "http://example.com/asset"))
				record("", root.Resolve(ctx, "", "http://example.com/missing.dtd"))
			}
		}()
	}
	wg.Wait()

	for _, m := range mismatches {
		require.Equal(t, m.want, m.got)
	}

	// Each referenced catalog must be loaded at most once despite the storm of
	// concurrent resolutions (per-entry load mutex).
	for url, cnt := range loader.counts {
		require.LessOrEqual(t, cnt.Load(), int32(1), "catalog %q loaded more than once", url)
	}
}

// A transient first-load failure must NOT be cached: a later Resolve has to
// retry and succeed. Regression for the sticky-loadErr bug where the first
// failure (canceled context, transient error) was remembered forever.
func TestTransientLoadFailureRetries(t *testing.T) {
	t.Parallel()

	leaf := &catalog.Catalog{
		Entries: []catalog.Entry{
			{Type: catalog.EntrySystem, Name: fooDTDSystemID, URL: "file:///foo.dtd"},
		},
	}
	loader := &flakyLoader{cat: leaf, failuresLeft: 1}

	root := &catalog.Catalog{
		Entries: []catalog.Entry{
			{Type: catalog.EntryNextCatalog, URL: sharedCatalogXML},
		},
		Loader: loader,
	}

	// First resolution: the load fails, so the entry must not resolve.
	got := root.Resolve(t.Context(), "", fooDTDSystemID)
	require.Equal(t, "", got, "first resolve should fail while the loader is failing")

	// Second resolution: the loader now succeeds; the entry must NOT be stuck
	// on the cached failure and should resolve.
	got = root.Resolve(t.Context(), "", fooDTDSystemID)
	require.Equal(t, "file:///foo.dtd", got, "second resolve should retry the load and succeed")

	require.Equal(t, int32(2), loader.calls.Load(), "loader should be retried after a transient failure")
}

// flakyLoader fails its first N loads (errTransient) before serving cat.
type flakyLoader struct {
	mu           sync.Mutex
	failuresLeft int
	cat          *catalog.Catalog
	calls        atomic.Int32
}

var errTransient = errors.New("transient load failure")

func (l *flakyLoader) Load(_ context.Context, _ string) (*catalog.Catalog, error) {
	l.calls.Add(1)

	l.mu.Lock()
	defer l.mu.Unlock()
	if l.failuresLeft > 0 {
		l.failuresLeft--
		return nil, errTransient
	}
	return l.cat, nil
}

// multiLoader is a concurrency-safe Loader serving distinct catalogs by URL
// and counting loads per URL.
type multiLoader struct {
	mu     sync.Mutex
	counts map[string]*atomic.Int32
	cats   map[string]*catalog.Catalog
}

func (l *multiLoader) Load(_ context.Context, filename string) (*catalog.Catalog, error) {
	l.mu.Lock()
	if l.counts[filename] == nil {
		l.counts[filename] = &atomic.Int32{}
	}
	cnt := l.counts[filename]
	cat := l.cats[filename]
	l.mu.Unlock()

	cnt.Add(1)
	return cat, nil
}

// countingLoader is a Loader that counts how many times each URL is loaded.
type countingLoader struct {
	counts map[string]*atomic.Int32
	cat    *catalog.Catalog
}

func (l *countingLoader) Load(_ context.Context, filename string) (*catalog.Catalog, error) {
	if l.counts[filename] == nil {
		l.counts[filename] = &atomic.Int32{}
	}
	l.counts[filename].Add(1)
	// Return a copy so each entry gets its own catalog instance.
	cp := *l.cat
	return &cp, nil
}

// totalLoader counts the total number of Load calls across all URLs and
// serves an empty leaf catalog for every URL.
type totalLoader struct {
	calls atomic.Int32
}

func (l *totalLoader) Load(_ context.Context, _ string) (*catalog.Catalog, error) {
	l.calls.Add(1)
	return &catalog.Catalog{}, nil
}

// A catalog with more than MaxDelegates UNIQUE delegate entries must load at
// most MaxDelegates of them during a single resolution, not all of them.
func TestDelegateLoadCountBounded(t *testing.T) {
	t.Parallel()

	mkEntries := func(typ catalog.EntryType, prefer catalog.Prefer) []catalog.Entry {
		n := catalog.MaxDelegates + 10
		entries := make([]catalog.Entry, 0, n)
		for i := range n {
			// Distinct URL per entry so the seen-set dedup never applies.
			entries = append(entries, catalog.Entry{
				Type:   typ,
				Name:   exampleBase,
				URL:    "deleg-" + strconv.Itoa(i) + ".xml",
				Prefer: prefer,
			})
		}
		return entries
	}

	t.Run("delegateSystem", func(t *testing.T) {
		t.Parallel()
		loader := &totalLoader{}
		root := &catalog.Catalog{
			Entries: mkEntries(catalog.EntryDelegateSystem, 0),
			Loader:  loader,
		}
		got := root.Resolve(t.Context(), "", "http://example.com/notfound.dtd")
		require.Equal(t, "", got)
		require.LessOrEqual(t, int(loader.calls.Load()), catalog.MaxDelegates,
			"loaded more delegate catalogs than MaxDelegates")
	})

	t.Run("delegatePublic", func(t *testing.T) {
		t.Parallel()
		loader := &totalLoader{}
		root := &catalog.Catalog{
			Entries: mkEntries(catalog.EntryDelegatePublic, catalog.PreferPublic),
			Loader:  loader,
		}
		got := root.Resolve(t.Context(), "http://example.com/notfound", "")
		require.Equal(t, "", got)
		require.LessOrEqual(t, int(loader.calls.Load()), catalog.MaxDelegates,
			"loaded more delegate catalogs than MaxDelegates")
	})

	t.Run("delegateURI", func(t *testing.T) {
		t.Parallel()
		loader := &totalLoader{}
		root := &catalog.Catalog{
			Entries: mkEntries(catalog.EntryDelegateURI, 0),
			Loader:  loader,
		}
		got := root.ResolveURI(t.Context(), "http://example.com/notfound")
		require.Equal(t, "", got)
		require.LessOrEqual(t, int(loader.calls.Load()), catalog.MaxDelegates,
			"loaded more delegate catalogs than MaxDelegates")
	})
}

// A catalog with more than MaxNextCatalogs UNIQUE nextCatalog entries must load
// at most MaxNextCatalogs of them during a single resolution. The sibling
// fan-out is bounded by a total-load cap, not only by recursion depth (CAT-001).
func TestNextCatalogLoadCountBounded(t *testing.T) {
	t.Parallel()

	mkEntries := func() []catalog.Entry {
		n := catalog.MaxNextCatalogs + 10
		entries := make([]catalog.Entry, 0, n)
		for i := range n {
			// Distinct URL per entry so the visited-set dedup never applies.
			entries = append(entries, catalog.Entry{
				Type: catalog.EntryNextCatalog,
				URL:  "next-" + strconv.Itoa(i) + ".xml",
			})
		}
		return entries
	}

	t.Run("Resolve", func(t *testing.T) {
		t.Parallel()
		loader := &totalLoader{}
		root := &catalog.Catalog{Entries: mkEntries(), Loader: loader}
		got := root.Resolve(t.Context(), "", "http://example.com/notfound.dtd")
		require.Equal(t, "", got)
		require.LessOrEqual(t, int(loader.calls.Load()), catalog.MaxNextCatalogs,
			"loaded more nextCatalog targets than MaxNextCatalogs")
	})

	t.Run("ResolveURI", func(t *testing.T) {
		t.Parallel()
		loader := &totalLoader{}
		root := &catalog.Catalog{Entries: mkEntries(), Loader: loader}
		got := root.ResolveURI(t.Context(), "http://example.com/notfound")
		require.Equal(t, "", got)
		require.LessOrEqual(t, int(loader.calls.Load()), catalog.MaxNextCatalogs,
			"loaded more nextCatalog targets than MaxNextCatalogs")
	})
}

// Per the OASIS XML Catalogs spec, matching delegate entries must be tried
// most-specific-first: the entry with the longest matching start-string is
// followed before shorter ones. Here the longer prefix points at a catalog
// that resolves the identifier; if document order were used the shorter
// (earlier) delegate would be tried first and a recordingLoader would observe
// the wrong load order.
func TestDelegateLongestPrefixFirst(t *testing.T) {
	t.Parallel()

	specific := &catalog.Catalog{
		Entries: []catalog.Entry{
			{Type: catalog.EntrySystem, Name: "http://example.com/sub/foo.dtd", URL: "file:///specific.dtd"},
			{Type: catalog.EntryURI, Name: "http://example.com/sub/foo.xsd", URL: "file:///specific.xsd"},
			{Type: catalog.EntryPublic, Name: "-//Example//Sub Pub//EN", URL: "file:///specific-pub.dtd"},
		},
		Prefer: catalog.PreferPublic,
	}
	general := &catalog.Catalog{
		Entries: []catalog.Entry{
			{Type: catalog.EntrySystem, Name: "http://example.com/sub/foo.dtd", URL: "file:///general.dtd"},
			{Type: catalog.EntryURI, Name: "http://example.com/sub/foo.xsd", URL: "file:///general.xsd"},
			{Type: catalog.EntryPublic, Name: "-//Example//Sub Pub//EN", URL: "file:///general-pub.dtd"},
		},
		Prefer: catalog.PreferPublic,
	}

	const generalXML = "general.xml"
	const specificXML = "specific.xml"

	newLoader := func() *recordingLoader {
		return &recordingLoader{
			cats: map[string]*catalog.Catalog{
				generalXML:  general,
				specificXML: specific,
			},
		}
	}

	t.Run("delegateSystem", func(t *testing.T) {
		t.Parallel()
		loader := newLoader()
		// Entries listed shorter-prefix-first in document order; the longer
		// prefix must still be tried first.
		root := &catalog.Catalog{
			Entries: []catalog.Entry{
				{Type: catalog.EntryDelegateSystem, Name: exampleBase, URL: generalXML},
				{Type: catalog.EntryDelegateSystem, Name: "http://example.com/sub/", URL: specificXML},
			},
			Loader: loader,
		}
		got := root.Resolve(t.Context(), "", "http://example.com/sub/foo.dtd")
		require.Equal(t, "file:///specific.dtd", got)
		require.Equal(t, specificXML, loader.first(), "longest matching delegateSystem prefix must be tried first")
	})

	t.Run("delegatePublic", func(t *testing.T) {
		t.Parallel()
		loader := newLoader()
		root := &catalog.Catalog{
			Entries: []catalog.Entry{
				{Type: catalog.EntryDelegatePublic, Name: "-//Example//", URL: generalXML, Prefer: catalog.PreferPublic},
				{Type: catalog.EntryDelegatePublic, Name: "-//Example//Sub", URL: specificXML, Prefer: catalog.PreferPublic},
			},
			Loader: loader,
			Prefer: catalog.PreferPublic,
		}
		got := root.Resolve(t.Context(), "-//Example//Sub Pub//EN", "")
		require.Equal(t, "file:///specific-pub.dtd", got)
		require.Equal(t, specificXML, loader.first(), "longest matching delegatePublic prefix must be tried first")
	})

	t.Run("delegateURI", func(t *testing.T) {
		t.Parallel()
		loader := newLoader()
		root := &catalog.Catalog{
			Entries: []catalog.Entry{
				{Type: catalog.EntryDelegateURI, Name: exampleBase, URL: generalXML},
				{Type: catalog.EntryDelegateURI, Name: "http://example.com/sub/", URL: specificXML},
			},
			Loader: loader,
		}
		got := root.ResolveURI(t.Context(), "http://example.com/sub/foo.xsd")
		require.Equal(t, "file:///specific.xsd", got)
		require.Equal(t, specificXML, loader.first(), "longest matching delegateURI prefix must be tried first")
	})
}

// recordingLoader records the order in which catalog URLs are loaded.
type recordingLoader struct {
	mu    sync.Mutex
	order []string
	cats  map[string]*catalog.Catalog
}

func (l *recordingLoader) Load(_ context.Context, filename string) (*catalog.Catalog, error) {
	l.mu.Lock()
	l.order = append(l.order, filename)
	cat := l.cats[filename]
	l.mu.Unlock()
	return cat, nil
}

func (l *recordingLoader) first() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.order) == 0 {
		return ""
	}
	return l.order[0]
}

func TestVisitedCacheSkipsDuplicate(t *testing.T) {
	t.Parallel()

	target := &catalog.Catalog{
		Entries: []catalog.Entry{
			{Type: catalog.EntrySystem, Name: fooDTDSystemID, URL: "file:///local/foo.dtd"},
		},
	}

	loader := &countingLoader{
		counts: make(map[string]*atomic.Int32),
		cat:    target,
	}

	root := &catalog.Catalog{
		Entries: []catalog.Entry{
			{Type: catalog.EntryNextCatalog, URL: sharedCatalogXML},
			{Type: catalog.EntryNextCatalog, URL: sharedCatalogXML},
		},
		Loader: loader,
	}

	got := root.Resolve(t.Context(), "", "http://example.com/notfound.dtd")
	require.Equal(t, "", got)

	cnt := loader.counts[sharedCatalogXML]
	require.NotNil(t, cnt)
}

func TestVisitedCacheStillResolves(t *testing.T) {
	t.Parallel()

	target := &catalog.Catalog{
		Entries: []catalog.Entry{
			{Type: catalog.EntrySystem, Name: fooDTDSystemID, URL: "file:///local/foo.dtd"},
		},
	}

	loader := &countingLoader{
		counts: make(map[string]*atomic.Int32),
		cat:    target,
	}

	root := &catalog.Catalog{
		Entries: []catalog.Entry{
			{Type: catalog.EntryNextCatalog, URL: sharedCatalogXML},
			{Type: catalog.EntryNextCatalog, URL: sharedCatalogXML},
		},
		Loader: loader,
	}

	got := root.Resolve(t.Context(), "", fooDTDSystemID)
	require.Equal(t, "file:///local/foo.dtd", got)
}

func TestVisitedCachePerQuery(t *testing.T) {
	t.Parallel()

	target := &catalog.Catalog{
		Entries: []catalog.Entry{
			{Type: catalog.EntrySystem, Name: "http://example.com/a.dtd", URL: "file:///a.dtd"},
			{Type: catalog.EntrySystem, Name: "http://example.com/b.dtd", URL: "file:///b.dtd"},
		},
	}

	loader := &countingLoader{
		counts: make(map[string]*atomic.Int32),
		cat:    target,
	}

	root := &catalog.Catalog{
		Entries: []catalog.Entry{
			{Type: catalog.EntryNextCatalog, URL: sharedCatalogXML},
		},
		Loader: loader,
	}

	got := root.Resolve(t.Context(), "", "http://example.com/a.dtd")
	require.Equal(t, "file:///a.dtd", got)

	got = root.Resolve(t.Context(), "", "http://example.com/b.dtd")
	require.Equal(t, "file:///b.dtd", got)
}

// An in-memory catalog with nextCatalog/delegate entries but no Loader must
// skip the unresolvable entries instead of dereferencing a nil sub-catalog.
func TestNoLoaderDoesNotPanic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		entries []catalog.Entry
		resolve func(*catalog.Catalog) string
	}{
		{
			name:    "nextCatalog ResolveURI",
			entries: []catalog.Entry{{Type: catalog.EntryNextCatalog, URL: missingCatalogXML}},
			resolve: func(c *catalog.Catalog) string {
				return c.ResolveURI(context.Background(), "http://example.com/x")
			},
		},
		{
			name:    "nextCatalog Resolve system",
			entries: []catalog.Entry{{Type: catalog.EntryNextCatalog, URL: missingCatalogXML}},
			resolve: func(c *catalog.Catalog) string {
				return c.Resolve(context.Background(), "", "http://example.com/x")
			},
		},
		{
			name:    "nextCatalog Resolve public",
			entries: []catalog.Entry{{Type: catalog.EntryNextCatalog, URL: missingCatalogXML}},
			resolve: func(c *catalog.Catalog) string {
				return c.Resolve(context.Background(), "-//Some//DTD//EN", "")
			},
		},
		{
			name: "delegateSystem",
			entries: []catalog.Entry{
				{Type: catalog.EntryDelegateSystem, Name: exampleBase, URL: missingCatalogXML},
			},
			resolve: func(c *catalog.Catalog) string {
				return c.Resolve(context.Background(), "", "http://example.com/x")
			},
		},
		{
			name: "delegatePublic",
			entries: []catalog.Entry{
				{Type: catalog.EntryDelegatePublic, Name: "-//Some//", URL: missingCatalogXML, Prefer: catalog.PreferPublic},
			},
			resolve: func(c *catalog.Catalog) string {
				return c.Resolve(context.Background(), "-//Some//DTD//EN", "")
			},
		},
		{
			name: "delegateURI",
			entries: []catalog.Entry{
				{Type: catalog.EntryDelegateURI, Name: exampleBase, URL: missingCatalogXML},
			},
			resolve: func(c *catalog.Catalog) string {
				return c.ResolveURI(context.Background(), "http://example.com/x")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cat := &catalog.Catalog{Entries: tc.entries, Prefer: catalog.PreferPublic}
			var got string
			require.NotPanics(t, func() {
				got = tc.resolve(cat)
			})
			require.Equal(t, "", got)
		})
	}
}
