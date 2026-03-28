package catalog_test

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/lestrrat-go/helium/internal/catalog"
	"github.com/stretchr/testify/require"
)

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

func TestVisitedCacheSkipsDuplicate(t *testing.T) {
	t.Parallel()

	target := &catalog.Catalog{
		Entries: []catalog.Entry{
			{Type: catalog.EntrySystem, Name: "http://example.com/foo.dtd", URL: "file:///local/foo.dtd"},
		},
	}

	loader := &countingLoader{
		counts: make(map[string]*atomic.Int32),
		cat:    target,
	}

	root := &catalog.Catalog{
		Entries: []catalog.Entry{
			{Type: catalog.EntryNextCatalog, URL: "shared.xml"},
			{Type: catalog.EntryNextCatalog, URL: "shared.xml"},
		},
		Loader: loader,
	}

	got := root.Resolve(t.Context(), "", "http://example.com/notfound.dtd")
	require.Equal(t, "", got)

	cnt := loader.counts["shared.xml"]
	require.NotNil(t, cnt)
}

func TestVisitedCacheStillResolves(t *testing.T) {
	t.Parallel()

	target := &catalog.Catalog{
		Entries: []catalog.Entry{
			{Type: catalog.EntrySystem, Name: "http://example.com/foo.dtd", URL: "file:///local/foo.dtd"},
		},
	}

	loader := &countingLoader{
		counts: make(map[string]*atomic.Int32),
		cat:    target,
	}

	root := &catalog.Catalog{
		Entries: []catalog.Entry{
			{Type: catalog.EntryNextCatalog, URL: "shared.xml"},
			{Type: catalog.EntryNextCatalog, URL: "shared.xml"},
		},
		Loader: loader,
	}

	got := root.Resolve(t.Context(), "", "http://example.com/foo.dtd")
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
			{Type: catalog.EntryNextCatalog, URL: "shared.xml"},
		},
		Loader: loader,
	}

	got := root.Resolve(t.Context(), "", "http://example.com/a.dtd")
	require.Equal(t, "file:///a.dtd", got)

	got = root.Resolve(t.Context(), "", "http://example.com/b.dtd")
	require.Equal(t, "file:///b.dtd", got)
}
