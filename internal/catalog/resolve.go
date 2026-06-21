package catalog

import (
	"context"
	"errors"
	"strings"
)

// CatalogBreak is a sentinel returned internally to signal "delegates were
// tried but all failed" — stops nextCatalog fallback (the libxml2 "cut"
// algorithm).
const CatalogBreak = "\x00CATAL_BREAK"

// errNoLoader is returned by lazyLoad when an entry references a catalog that
// must be loaded but no Loader is configured. Callers treat it like any other
// load failure and skip the entry.
var errNoLoader = errors.New("catalog: no loader configured")

// resolveState holds the per-resolution mutable bookkeeping (recursion depth
// and visited cache). It is allocated once per top-level Resolve/ResolveURI
// call and threaded through the recursion so that a single *Catalog can be
// resolved concurrently from multiple goroutines without data races. The
// receiver carries only immutable configuration; all run state lives here.
type resolveState struct {
	depth   int
	visited map[visitedKey]struct{}
	// delegates counts how many delegate catalogs have actually been loaded
	// and followed during this resolution. It bounds I/O at MaxDelegates so a
	// catalog with many UNIQUE delegate entries cannot trigger an unbounded
	// load fan-out (CAT-005).
	delegates int
}

// checkVisited returns true if the (url, id1, id2) combination has already
// been visited during this resolution. If not, marks it as visited.
// Matches libxml2's xmlCatalogResolveCacheVisited.
func (s *resolveState) checkVisited(url, id1, id2 string) bool {
	key := visitedKey{url: url, id1: id1, id2: id2}
	if _, ok := s.visited[key]; ok {
		return true
	}
	s.visited[key] = struct{}{}
	return false
}

// Resolve resolves an external identifier (pubID and/or sysID) to a URI.
// Returns the resolved URI or "" if not found.
//
// Resolve is safe to call concurrently on a single *Catalog: per-resolution
// state lives in a local resolveState rather than on the receiver.
func (c *Catalog) Resolve(ctx context.Context, pubID, sysID string) string {
	if c == nil {
		return ""
	}
	if pubID == "" && sysID == "" {
		return ""
	}

	st := &resolveState{visited: make(map[visitedKey]struct{})}
	ret := c.resolveTop(ctx, st, pubID, sysID)
	if ret == CatalogBreak {
		return ""
	}
	return ret
}

// resolveTop performs URN unwrapping and public-ID normalization before
// entering the core resolve algorithm. It shares the visited cache across the
// (possibly recursive) unwrap steps via st.
func (c *Catalog) resolveTop(ctx context.Context, st *resolveState, pubID, sysID string) string {
	// Normalize public ID.
	if pubID != "" {
		pubID = NormalizePublicID(pubID)
	}

	// Unwrap URNs.
	if pubID != "" {
		if urnPub := UnwrapURN(pubID); urnPub != "" {
			return c.resolveTop(ctx, st, urnPub, sysID)
		}
	}
	if sysID != "" {
		if urnSys := UnwrapURN(sysID); urnSys != "" {
			if pubID == "" {
				return c.resolveTop(ctx, st, urnSys, "")
			}
			if pubID == urnSys {
				return c.resolveTop(ctx, st, pubID, "")
			}
			return c.resolveTop(ctx, st, pubID, urnSys)
		}
	}

	return c.resolve(ctx, st, pubID, sysID)
}

// ResolveURI resolves a URI reference.
// Returns the resolved URI or "" if not found.
//
// ResolveURI is safe to call concurrently on a single *Catalog.
func (c *Catalog) ResolveURI(ctx context.Context, uri string) string {
	if c == nil || uri == "" {
		return ""
	}

	st := &resolveState{visited: make(map[visitedKey]struct{})}

	// Unwrap urn:publicid: URNs and delegate to Resolve as a public ID,
	// matching libxml2's xmlCatalogListXMLResolveURI behavior.
	if pubID := UnwrapURN(uri); pubID != "" {
		ret := c.resolveTop(ctx, st, pubID, "")
		if ret == CatalogBreak {
			return ""
		}
		return ret
	}

	ret := c.resolveURI(ctx, st, uri)
	if ret == CatalogBreak {
		return ""
	}
	return ret
}

// resolve implements the core resolution algorithm from libxml2's
// xmlCatalogXMLResolve. Returns "" if not found or CatalogBreak to
// signal cut.
func (c *Catalog) resolve(ctx context.Context, st *resolveState, pubID, sysID string) string {
	if st.depth > MaxDepth {
		return ""
	}
	st.depth++
	defer func() { st.depth-- }()

	haveDelegate := 0
	haveNext := 0

	var rewrite *Entry
	lenRewrite := 0

	// First pass: scan entries for system ID resolution.
	if sysID != "" {
		for i := range c.Entries {
			e := &c.Entries[i]
			switch e.Type {
			case EntrySystem:
				if e.Name == sysID {
					return e.URL
				}
			case EntryRewriteSystem:
				if strings.HasPrefix(sysID, e.Name) && len(e.Name) > lenRewrite {
					rewrite = e
					lenRewrite = len(e.Name)
				}
			case EntryDelegateSystem:
				if strings.HasPrefix(sysID, e.Name) {
					haveDelegate++
				}
			case EntryNextCatalog:
				haveNext++
			}
		}

		if rewrite != nil {
			return rewrite.URL + sysID[lenRewrite:]
		}

		if haveDelegate > 0 {
			ret := c.resolveDelegateSystem(ctx, st, sysID)
			if ret != "" {
				return ret
			}
			// Cut: delegates existed but none resolved.
			return CatalogBreak
		}
	}

	// Second pass: public ID resolution.
	haveDelegate = 0
	if pubID != "" {
		for i := range c.Entries {
			e := &c.Entries[i]
			switch e.Type {
			case EntryPublic:
				if e.Name == pubID {
					return e.URL
				}
			case EntryDelegatePublic:
				if strings.HasPrefix(pubID, e.Name) && e.Prefer == PreferPublic {
					haveDelegate++
				}
			case EntryNextCatalog:
				if sysID == "" {
					haveNext++
				}
			}
		}

		if haveDelegate > 0 {
			ret := c.resolveDelegatePublic(ctx, st, pubID)
			if ret != "" {
				return ret
			}
			return CatalogBreak
		}
	}

	// nextCatalog fallback.
	if haveNext > 0 {
		return c.resolveNextCatalogs(ctx, st, pubID, sysID)
	}

	return ""
}

// resolveURI implements URI resolution from libxml2's xmlCatalogXMLResolveURI.
func (c *Catalog) resolveURI(ctx context.Context, st *resolveState, uri string) string {
	if st.depth > MaxDepth {
		return ""
	}
	st.depth++
	defer func() { st.depth-- }()

	haveDelegate := 0
	haveNext := 0

	var rewrite *Entry
	lenRewrite := 0

	for i := range c.Entries {
		e := &c.Entries[i]
		switch e.Type {
		case EntryURI:
			if e.Name == uri {
				return e.URL
			}
		case EntryRewriteURI:
			if strings.HasPrefix(uri, e.Name) && len(e.Name) > lenRewrite {
				rewrite = e
				lenRewrite = len(e.Name)
			}
		case EntryDelegateURI:
			if strings.HasPrefix(uri, e.Name) {
				haveDelegate++
			}
		case EntryNextCatalog:
			haveNext++
		}
	}

	if rewrite != nil {
		return rewrite.URL + uri[lenRewrite:]
	}

	if haveDelegate > 0 {
		ret := c.resolveDelegateURI(ctx, st, uri)
		if ret != "" {
			return ret
		}
		return CatalogBreak
	}

	if haveNext > 0 {
		return c.resolveNextCatalogsURI(ctx, st, uri)
	}

	return ""
}

// resolveDelegateSystem tries all matching delegateSystem entries.
func (c *Catalog) resolveDelegateSystem(ctx context.Context, st *resolveState, sysID string) string {
	seen := make(map[string]struct{}, MaxDelegates)

	for i := range c.Entries {
		e := &c.Entries[i]
		if e.Type != EntryDelegateSystem {
			continue
		}
		if !strings.HasPrefix(sysID, e.Name) {
			continue
		}
		if _, dup := seen[e.URL]; dup {
			continue
		}
		seen[e.URL] = struct{}{}
		// Bound the number of delegate catalogs actually loaded across the whole
		// resolution, not just the size of the dedup set (CAT-005).
		if st.delegates >= MaxDelegates {
			break
		}
		st.delegates++

		sub, err := c.lazyLoad(ctx, e)
		if err != nil {
			continue
		}
		if st.checkVisited(e.URL, "", sysID) {
			continue
		}
		// Delegate with sysID only (pubID=nil per libxml2).
		ret := sub.resolve(ctx, st, "", sysID)
		if ret != "" && ret != CatalogBreak {
			return ret
		}
	}
	return ""
}

// resolveDelegatePublic tries all matching delegatePublic entries.
func (c *Catalog) resolveDelegatePublic(ctx context.Context, st *resolveState, pubID string) string {
	seen := make(map[string]struct{}, MaxDelegates)

	for i := range c.Entries {
		e := &c.Entries[i]
		if e.Type != EntryDelegatePublic {
			continue
		}
		if e.Prefer != PreferPublic {
			continue
		}
		if !strings.HasPrefix(pubID, e.Name) {
			continue
		}
		if _, dup := seen[e.URL]; dup {
			continue
		}
		seen[e.URL] = struct{}{}
		// Bound the number of delegate catalogs actually loaded across the whole
		// resolution, not just the size of the dedup set (CAT-005).
		if st.delegates >= MaxDelegates {
			break
		}
		st.delegates++

		sub, err := c.lazyLoad(ctx, e)
		if err != nil {
			continue
		}
		if st.checkVisited(e.URL, pubID, "") {
			continue
		}
		// Delegate with pubID only (sysID=nil per libxml2).
		ret := sub.resolve(ctx, st, pubID, "")
		if ret != "" && ret != CatalogBreak {
			return ret
		}
	}
	return ""
}

// resolveDelegateURI tries all matching delegateURI entries.
func (c *Catalog) resolveDelegateURI(ctx context.Context, st *resolveState, uri string) string {
	seen := make(map[string]struct{}, MaxDelegates)

	for i := range c.Entries {
		e := &c.Entries[i]
		if e.Type != EntryDelegateURI {
			continue
		}
		if !strings.HasPrefix(uri, e.Name) {
			continue
		}
		if _, dup := seen[e.URL]; dup {
			continue
		}
		seen[e.URL] = struct{}{}
		// Bound the number of delegate catalogs actually loaded across the whole
		// resolution, not just the size of the dedup set (CAT-005).
		if st.delegates >= MaxDelegates {
			break
		}
		st.delegates++

		sub, err := c.lazyLoad(ctx, e)
		if err != nil {
			continue
		}
		if st.checkVisited(e.URL, uri, "") {
			continue
		}
		ret := sub.resolveURI(ctx, st, uri)
		if ret != "" && ret != CatalogBreak {
			return ret
		}
	}
	return ""
}

// resolveNextCatalogs tries all nextCatalog entries for Resolve.
func (c *Catalog) resolveNextCatalogs(ctx context.Context, st *resolveState, pubID, sysID string) string {
	for i := range c.Entries {
		e := &c.Entries[i]
		if e.Type != EntryNextCatalog {
			continue
		}
		sub, err := c.lazyLoad(ctx, e)
		if err != nil {
			continue
		}
		if st.checkVisited(e.URL, pubID, sysID) {
			continue
		}
		ret := sub.resolve(ctx, st, pubID, sysID)
		if ret != "" && ret != CatalogBreak {
			return ret
		}
		if st.depth > MaxDepth {
			return ""
		}
	}
	return ""
}

// resolveNextCatalogsURI tries all nextCatalog entries for ResolveURI.
func (c *Catalog) resolveNextCatalogsURI(ctx context.Context, st *resolveState, uri string) string {
	for i := range c.Entries {
		e := &c.Entries[i]
		if e.Type != EntryNextCatalog {
			continue
		}
		sub, err := c.lazyLoad(ctx, e)
		if err != nil {
			continue
		}
		if st.checkVisited(e.URL, uri, "") {
			continue
		}
		ret := sub.resolveURI(ctx, st, uri)
		if ret != "" && ret != CatalogBreak {
			return ret
		}
		if st.depth > MaxDepth {
			return ""
		}
	}
	return ""
}

// lazyLoad loads the catalog file for a delegate or nextCatalog entry on first
// access via the Loader and returns the (possibly cached) sub-catalog. The load
// is guarded by a per-entry mutex so that concurrent resolutions sharing one
// *Catalog load the referenced catalog at most once and never observe a
// partially-populated entry. Only SUCCESSFUL loads are cached: an error (a
// missing loader, a transient/Loader failure, or a load that yields no
// catalog) is returned without marking the entry loaded, so a later healthy
// Resolve can retry. The returned sub-catalog carries no run state — depth and
// the visited cache live in the caller's resolveState.
func (c *Catalog) lazyLoad(ctx context.Context, e *Entry) (*Catalog, error) {
	e.loadMu.Lock()
	defer e.loadMu.Unlock()

	if e.loaded {
		return e.Catalog, nil
	}
	if e.Catalog != nil {
		// Pre-populated (e.g. tests, or an inlined sub-catalog).
		e.loaded = true
		return e.Catalog, nil
	}
	if c.Loader == nil {
		// The referenced catalog needs loading but no loader is configured.
		return nil, errNoLoader
	}

	cat, err := c.Loader.Load(ctx, e.URL)
	if err != nil {
		// Do not cache the failure: a later call may succeed.
		return nil, err
	}
	if cat == nil {
		// A load that neither errored nor produced a catalog: treat as a miss
		// and allow a retry.
		return nil, errNoLoader
	}

	e.Catalog = cat
	e.loaded = true
	return e.Catalog, nil
}
