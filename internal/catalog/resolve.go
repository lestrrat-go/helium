package catalog

import (
	"strings"
)

// CatalogBreak is a sentinel returned internally to signal "delegates were
// tried but all failed" — stops nextCatalog fallback (the libxml2 "cut"
// algorithm).
const CatalogBreak = "\x00CATAL_BREAK"

// Resolve resolves an external identifier (pubID and/or sysID) to a URI.
// Returns the resolved URI or "" if not found.
func (c *Catalog) Resolve(pubID, sysID string) string {
	if c == nil {
		return ""
	}
	if pubID == "" && sysID == "" {
		return ""
	}

	// Normalize public ID.
	if pubID != "" {
		pubID = NormalizePublicID(pubID)
	}

	// Unwrap URNs.
	if pubID != "" {
		if urnPub := UnwrapURN(pubID); urnPub != "" {
			return c.Resolve(urnPub, sysID)
		}
	}
	if sysID != "" {
		if urnSys := UnwrapURN(sysID); urnSys != "" {
			if pubID == "" {
				return c.Resolve(urnSys, "")
			}
			if pubID == urnSys {
				return c.Resolve(pubID, "")
			}
			return c.Resolve(pubID, urnSys)
		}
	}

	ret := c.resolve(pubID, sysID)
	if ret == CatalogBreak {
		return ""
	}
	return ret
}

// ResolveURI resolves a URI reference.
// Returns the resolved URI or "" if not found.
func (c *Catalog) ResolveURI(uri string) string {
	if c == nil || uri == "" {
		return ""
	}

	// Unwrap urn:publicid: URNs and delegate to Resolve as a public ID,
	// matching libxml2's xmlCatalogListXMLResolveURI behavior.
	if pubID := UnwrapURN(uri); pubID != "" {
		return c.Resolve(pubID, "")
	}

	ret := c.resolveURI(uri)
	if ret == CatalogBreak {
		return ""
	}
	return ret
}

// resolve implements the core resolution algorithm from libxml2's
// xmlCatalogXMLResolve. Returns "" if not found or CatalogBreak to
// signal cut.
func (c *Catalog) resolve(pubID, sysID string) string {
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
			switch e.Typ {
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
			ret := c.resolveDelegateSystem(sysID)
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
			switch e.Typ {
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
			ret := c.resolveDelegatePublic(pubID)
			if ret != "" {
				return ret
			}
			return CatalogBreak
		}
	}

	// nextCatalog fallback.
	if haveNext > 0 {
		return c.resolveNextCatalogs(pubID, sysID)
	}

	return ""
}

// resolveURI implements URI resolution from libxml2's xmlCatalogXMLResolveURI.
func (c *Catalog) resolveURI(uri string) string {
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
		switch e.Typ {
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
		ret := c.resolveDelegateURI(uri)
		if ret != "" {
			return ret
		}
		return CatalogBreak
	}

	if haveNext > 0 {
		return c.resolveNextCatalogsURI(uri)
	}

	return ""
}

// resolveDelegateSystem tries all matching delegateSystem entries.
func (c *Catalog) resolveDelegateSystem(sysID string) string {
	seen := make(map[string]struct{}, MaxDelegates)

	for i := range c.Entries {
		e := &c.Entries[i]
		if e.Typ != EntryDelegateSystem {
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

		if err := c.lazyLoad(e); err != nil {
			continue
		}
		// Delegate with sysID only (pubID=nil per libxml2).
		ret := e.Catalog.resolve("", sysID)
		if ret != "" && ret != CatalogBreak {
			return ret
		}
	}
	return ""
}

// resolveDelegatePublic tries all matching delegatePublic entries.
func (c *Catalog) resolveDelegatePublic(pubID string) string {
	seen := make(map[string]struct{}, MaxDelegates)

	for i := range c.Entries {
		e := &c.Entries[i]
		if e.Typ != EntryDelegatePublic {
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

		if err := c.lazyLoad(e); err != nil {
			continue
		}
		// Delegate with pubID only (sysID=nil per libxml2).
		ret := e.Catalog.resolve(pubID, "")
		if ret != "" && ret != CatalogBreak {
			return ret
		}
	}
	return ""
}

// resolveDelegateURI tries all matching delegateURI and delegateSystem entries.
func (c *Catalog) resolveDelegateURI(uri string) string {
	seen := make(map[string]struct{}, MaxDelegates)

	for i := range c.Entries {
		e := &c.Entries[i]
		if e.Typ != EntryDelegateURI && e.Typ != EntryDelegateSystem {
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

		if err := c.lazyLoad(e); err != nil {
			continue
		}
		ret := e.Catalog.resolveURI(uri)
		if ret != "" && ret != CatalogBreak {
			return ret
		}
	}
	return ""
}

// resolveNextCatalogs tries all nextCatalog entries for Resolve.
func (c *Catalog) resolveNextCatalogs(pubID, sysID string) string {
	for i := range c.Entries {
		e := &c.Entries[i]
		if e.Typ != EntryNextCatalog {
			continue
		}
		if err := c.lazyLoad(e); err != nil {
			continue
		}
		ret := e.Catalog.resolve(pubID, sysID)
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
func (c *Catalog) resolveNextCatalogsURI(uri string) string {
	for i := range c.Entries {
		e := &c.Entries[i]
		if e.Typ != EntryNextCatalog {
			continue
		}
		if err := c.lazyLoad(e); err != nil {
			continue
		}
		ret := e.Catalog.resolveURI(uri)
		if ret != "" && ret != CatalogBreak {
			return ret
		}
		if e.Catalog.Depth > MaxDepth {
			return ""
		}
	}
	return ""
}

// lazyLoad loads the catalog file for a delegate or nextCatalog entry
// on first access via the Loader. The loaded catalog shares the parent's
// depth counter for recursion detection.
func (c *Catalog) lazyLoad(e *Entry) error {
	if e.Catalog != nil {
		// Already loaded — share depth.
		e.Catalog.Depth = c.Depth
		return nil
	}
	if c.Ldr == nil {
		return nil
	}
	cat, err := c.Ldr.Load(e.URL)
	if err != nil {
		return err
	}
	// Share the parent's depth counter for recursion detection.
	cat.Depth = c.Depth
	e.Catalog = cat
	return nil
}
