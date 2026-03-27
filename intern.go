package helium

import "github.com/lestrrat-go/helium/internal/lexicon"

// globalNames is a pre-built map of well-known XML name strings.
// Looking up a scanned name here returns the compile-time constant,
// avoiding a heap allocation for the string.
var globalNames map[string]string
var globalNameCandidateMask [256]uint32

func init() {
	globalNames = make(map[string]string, len(lexicon.WellKnownNames))
	for _, s := range lexicon.WellKnownNames {
		globalNames[s] = s
		if len(s) < 32 {
			globalNameCandidateMask[s[0]] |= 1 << len(s)
		}
	}
}

func couldBeGlobalNameBytes(b []byte) bool {
	if len(b) == 0 || len(b) >= 32 {
		return false
	}
	return globalNameCandidateMask[b[0]]&(1<<len(b)) != 0
}

func couldBeGlobalNameString(s string) bool {
	if len(s) == 0 || len(s) >= 32 {
		return false
	}
	return globalNameCandidateMask[s[0]]&(1<<len(s)) != 0
}

// internName returns a deduplicated version of s. It checks the global
// well-known table first (zero allocation on hit), then the per-parse table.
func (pctx *parserCtx) internName(s string) string {
	// Tier 1: global well-known names (zero alloc on hit).
	if couldBeGlobalNameString(s) {
		if interned, ok := globalNames[s]; ok {
			return interned
		}
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
	if couldBeGlobalNameBytes(b) {
		if interned, ok := globalNames[string(b)]; ok {
			return interned
		}
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
