package catalog

import (
	"context"
	"strings"
)

// CatalogBreak is a sentinel returned internally to signal "delegates were
// tried but all failed" — stops nextCatalog fallback (the libxml2 "cut"
// algorithm).
const CatalogBreak = "\x00CATAL_BREAK"

// Resolve resolves an external identifier (pubID and/or sysID) to a URI.
// Returns the resolved URI or "" if not found.
func (c *Catalog) Resolve(ctx context.Context, pubID, sysID string) string {
	if c == nil {
		return ""
	}
	if pubID == "" && sysID == "" {
		return ""
	}

	// Initialize visited cache for this top-level resolution,
	// matching libxml2's xmlResetCatalogResolveCache pattern.
	topLevel := c.visited == nil
	if topLevel {
		c.visited = make(map[visitedKey]struct{})
		defer func() { c.visited = nil }()
	}

	// Normalize public ID.
	if pubID != "" {
		pubID = NormalizePublicID(pubID)
	}

	// Unwrap URNs.
	if pubID != "" {
		if urnPub := UnwrapURN(pubID); urnPub != "" {
			return c.Resolve(ctx, urnPub, sysID)
		}
	}
	if sysID != "" {
		if urnSys := UnwrapURN(sysID); urnSys != "" {
			if pubID == "" {
				return c.Resolve(ctx, urnSys, "")
			}
			if pubID == urnSys {
				return c.Resolve(ctx, pubID, "")
			}
			return c.Resolve(ctx, pubID, urnSys)
		}
	}

	ret := c.resolve(ctx, pubID, sysID)
	if ret == CatalogBreak {
		return ""
	}
	return ret
}

// ResolveURI resolves a URI reference.
// Returns the resolved URI or "" if not found.
func (c *Catalog) ResolveURI(ctx context.Context, uri string) string {
	if c == nil || uri == "" {
		return ""
	}

	// Initialize visited cache for this top-level resolution.
	topLevel := c.visited == nil
	if topLevel {
		c.visited = make(map[visitedKey]struct{})
		defer func() { c.visited = nil }()
	}

	// Unwrap urn:publicid: URNs and delegate to Resolve as a public ID,
	// matching libxml2's xmlCatalogListXMLResolveURI behavior.
	if pubID := UnwrapURN(uri); pubID != "" {
		return c.Resolve(ctx, pubID, "")
	}

	ret := c.resolveURI(ctx, uri)
	if ret == CatalogBreak {
		return ""
	}
	return ret
}

// resolve implements the core resolution algorithm from libxml2's
// xmlCatalogXMLResolve. Returns "" if not found or CatalogBreak to
// signal cut.
func (c *Catalog) resolve(ctx context.Context, pubID, sysID string) string {
	if c.Depth > MaxDepth {
		return ""
	}
	c.Depth++
	defer func() { c.Depth-- }()

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
			ret := c.resolveDelegateSystem(ctx, sysID)
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
			ret := c.resolveDelegatePublic(ctx, pubID)
			if ret != "" {
				return ret
			}
			return CatalogBreak
		}
	}

	// nextCatalog fallback.
	if haveNext > 0 {
		return c.resolveNextCatalogs(ctx, pubID, sysID)
	}

	return ""
}

// resolveURI implements URI resolution from libxml2's xmlCatalogXMLResolveURI.
func (c *Catalog) resolveURI(ctx context.Context, uri string) string {
	if c.Depth > MaxDepth {
		return ""
	}
	c.Depth++
	defer func() { c.Depth-- }()

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
		case EntryDelegateURI, EntryDelegateSystem:
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
		ret := c.resolveDelegateURI(ctx, uri)
		if ret != "" {
			return ret
		}
		return CatalogBreak
	}

	if haveNext > 0 {
		return c.resolveNextCatalogsURI(ctx, uri)
	}

	return ""
}

// resolveDelegateSystem tries all matching delegateSystem entries.
func (c *Catalog) resolveDelegateSystem(ctx context.Context, sysID string) string {
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
		if len(seen) < MaxDelegates {
			seen[e.URL] = struct{}{}
		}

		if err := c.lazyLoad(ctx, e); err != nil {
			continue
		}
		if c.checkVisited(e.URL, "", sysID) {
			continue
		}
		// Delegate with sysID only (pubID=nil per libxml2).
		ret := e.Catalog.resolve(ctx, "", sysID)
		if ret != "" && ret != CatalogBreak {
			return ret
		}
	}
	return ""
}

// resolveDelegatePublic tries all matching delegatePublic entries.
func (c *Catalog) resolveDelegatePublic(ctx context.Context, pubID string) string {
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
		if len(seen) < MaxDelegates {
			seen[e.URL] = struct{}{}
		}

		if err := c.lazyLoad(ctx, e); err != nil {
			continue
		}
		if c.checkVisited(e.URL, pubID, "") {
			continue
		}
		// Delegate with pubID only (sysID=nil per libxml2).
		ret := e.Catalog.resolve(ctx, pubID, "")
		if ret != "" && ret != CatalogBreak {
			return ret
		}
	}
	return ""
}

// resolveDelegateURI tries all matching delegateURI and delegateSystem entries.
func (c *Catalog) resolveDelegateURI(ctx context.Context, uri string) string {
	seen := make(map[string]struct{}, MaxDelegates)

	for i := range c.Entries {
		e := &c.Entries[i]
		if e.Type != EntryDelegateURI && e.Type != EntryDelegateSystem {
			continue
		}
		if !strings.HasPrefix(uri, e.Name) {
			continue
		}
		if _, dup := seen[e.URL]; dup {
			continue
		}
		if len(seen) < MaxDelegates {
			seen[e.URL] = struct{}{}
		}

		if err := c.lazyLoad(ctx, e); err != nil {
			continue
		}
		if c.checkVisited(e.URL, uri, "") {
			continue
		}
		ret := e.Catalog.resolveURI(ctx, uri)
		if ret != "" && ret != CatalogBreak {
			return ret
		}
	}
	return ""
}

// resolveNextCatalogs tries all nextCatalog entries for Resolve.
func (c *Catalog) resolveNextCatalogs(ctx context.Context, pubID, sysID string) string {
	for i := range c.Entries {
		e := &c.Entries[i]
		if e.Type != EntryNextCatalog {
			continue
		}
		if err := c.lazyLoad(ctx, e); err != nil {
			continue
		}
		if c.checkVisited(e.URL, pubID, sysID) {
			continue
		}
		ret := e.Catalog.resolve(ctx, pubID, sysID)
		if ret != "" && ret != CatalogBreak {
			return ret
		}
		if e.Catalog.Depth > MaxDepth {
			return ""
		}
	}
	return ""
}

// resolveNextCatalogsURI tries all nextCatalog entries for ResolveURI.
func (c *Catalog) resolveNextCatalogsURI(ctx context.Context, uri string) string {
	for i := range c.Entries {
		e := &c.Entries[i]
		if e.Type != EntryNextCatalog {
			continue
		}
		if err := c.lazyLoad(ctx, e); err != nil {
			continue
		}
		if c.checkVisited(e.URL, uri, "") {
			continue
		}
		ret := e.Catalog.resolveURI(ctx, uri)
		if ret != "" && ret != CatalogBreak {
			return ret
		}
		if e.Catalog.Depth > MaxDepth {
			return ""
		}
	}
	return ""
}

// checkVisited returns true if the (url, id1, id2) combination has already
// been visited during this resolution. If not, marks it as visited.
// Matches libxml2's xmlCatalogResolveCacheVisited.
func (c *Catalog) checkVisited(url, id1, id2 string) bool {
	if c.visited == nil {
		return false
	}
	key := visitedKey{url: url, id1: id1, id2: id2}
	if _, ok := c.visited[key]; ok {
		return true
	}
	c.visited[key] = struct{}{}
	return false
}

// lazyLoad loads the catalog file for a delegate or nextCatalog entry
// on first access via the Loader. The loaded catalog shares the parent's
// depth counter and visited cache for recursion detection.
func (c *Catalog) lazyLoad(ctx context.Context, e *Entry) error {
	if e.Catalog != nil {
		// Already loaded — share depth and visited cache.
		e.Catalog.Depth = c.Depth
		e.Catalog.visited = c.visited
		return nil
	}
	if c.Loader == nil {
		return nil
	}
	cat, err := c.Loader.Load(ctx, e.URL)
	if err != nil {
		return err
	}
	// Share the parent's depth counter and visited cache.
	cat.Depth = c.Depth
	cat.visited = c.visited
	e.Catalog = cat
	return nil
}
