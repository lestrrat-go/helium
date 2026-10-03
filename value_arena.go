package helium

import "unsafe"

const (
	// valueArenaFirstChunk and valueArenaMaxChunk bound the chunks a
	// valueArena allocates: the first is small, so a small document's few
	// attribute values cost little, and each next chunk doubles up to the
	// maximum.
	valueArenaFirstChunk = 256
	valueArenaMaxChunk   = 2 << 10

	// valueArenaLargeValue is the longest value valueArena packs into a chunk.
	// A longer one gets an allocation of its own, so the arena leaves fewer
	// than this many bytes unused when it moves on to a new chunk.
	valueArenaLargeValue = 256
)

// valueArena turns parsed attribute values into strings with one allocation per
// chunk of values, in place of one per value.
//
// Each parse owns its arena (parserCtx.values), so concurrent parses never
// share one. Its chunks are plain heap memory: they never go to a pool and the
// arena never writes a byte it has handed out, so a string from it never
// changes, whoever keeps it (the DOM, a SAX handler, an ID table) and for as
// long as they keep it. A retained value keeps its whole chunk, at most
// valueArenaMaxChunk bytes, alive.
type valueArena struct {
	free []byte // unused tail of the current chunk
	next int    // size of the next chunk; 0 before the first
}

// String returns a string holding a copy of b. b may be borrowed (a cursor
// buffer slice); the result does not alias it.
func (a *valueArena) String(b []byte) string {
	n := len(b)
	if n == 0 {
		return ""
	}
	if n > valueArenaLargeValue {
		return string(b)
	}
	if n > len(a.free) {
		size := max(a.next, valueArenaFirstChunk)
		a.next = min(2*size, valueArenaMaxChunk)
		a.free = make([]byte, max(size, n))
	}
	dst := a.free[:n:n]
	a.free = a.free[n:]
	copy(dst, b)
	return unsafe.String(unsafe.SliceData(dst), n)
}
