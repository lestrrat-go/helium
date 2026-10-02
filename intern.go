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

// nameCacheSlots is the size of the direct-mapped cache internNameBytes
// consults before the interning maps. A document repeats a small set of
// element and attribute names, so most lookups hit their slot and cost one
// short string comparison instead of a map probe that hashes the whole name.
const nameCacheSlots = 256

// nameCacheSlot maps a scanned name to its slot in the direct-mapped cache
// from its length and three of its bytes. b must not be empty.
func nameCacheSlot(b []byte) int {
	n := len(b)
	return (n*131 + int(b[0])*31 + int(b[n-1])*7 + int(b[n/2])) & (nameCacheSlots - 1)
}

// internNameBytes returns a deduplicated string for the given byte slice. A
// hit in the direct-mapped cache returns the string an earlier call interned
// for the same bytes; a miss (an empty slot, or a different name sharing the
// slot) interns through internNameBytesMaps and replaces the slot. The cache
// lives on the parser context, so it never carries a name from one parse into
// another, and every string it returns is the one the maps hold.
func (pctx *parserCtx) internNameBytes(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	slot := nameCacheSlot(b)
	if s := pctx.nameCacheFast[slot]; s == string(b) {
		return s
	}
	s := pctx.internNameBytesMaps(b)
	pctx.nameCacheFast[slot] = s
	return s
}

// internNameBytesMaps interns b through the global well-known table and the
// per-parse map. Uses Go's map optimization: map[string]([]byte) lookups don't
// allocate when the key is a []byte→string conversion used only for the
// lookup.
func (pctx *parserCtx) internNameBytesMaps(b []byte) string {
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
