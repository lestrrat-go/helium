package helium

import "github.com/lestrrat-go/helium/internal/lexicon"

// globalNames is a pre-built map of well-known XML name strings.
// Looking up a scanned name here returns the compile-time constant,
// avoiding a heap allocation for the string.
var globalNames map[string]string

func init() {
	globalNames = make(map[string]string, len(lexicon.WellKnownNames))
	for _, s := range lexicon.WellKnownNames {
		globalNames[s] = s
	}
}

// internName returns a deduplicated version of s. It checks the global
// well-known table first (zero allocation on hit), then the per-parse table.
func (pctx *parserCtx) internName(s string) string {
	// Tier 1: global well-known names (zero alloc on hit).
	if interned, ok := globalNames[s]; ok {
		return interned
	}
	// Tier 2: per-parse deduplication.
	if pctx.nameCache == nil {
		pctx.nameCache = make(map[string]string)
	}
	if interned, ok := pctx.nameCache[s]; ok {
		return interned
	}
	pctx.nameCache[s] = s
	return s
}

// internNameBytes returns a deduplicated string for the given byte slice.
// Uses Go's map optimization: map[string]([]byte) lookups don't allocate
// when the key is a []byte→string conversion used only for the lookup.
func (pctx *parserCtx) internNameBytes(b []byte) string {
	// Tier 1: global well-known names.
	if interned, ok := globalNames[string(b)]; ok {
		return interned
	}
	// Tier 2: per-parse deduplication.
	if pctx.nameCache == nil {
		pctx.nameCache = make(map[string]string)
	}
	if interned, ok := pctx.nameCache[string(b)]; ok {
		return interned
	}
	s := string(b)
	pctx.nameCache[s] = s
	return s
}
