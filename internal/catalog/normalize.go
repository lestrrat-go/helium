package catalog

import "strings"

// NormalizePublicID normalizes a public identifier per the OASIS XML Catalog
// specification: leading/trailing whitespace is removed, and runs of whitespace
// characters (space, tab, newline, carriage return) are collapsed to a single
// space (U+0020).
func NormalizePublicID(pubID string) string {
	var b strings.Builder
	b.Grow(len(pubID))
	inSpace := true // treat leading whitespace as a run
	for i := range len(pubID) {
		c := pubID[i]
		switch c {
		case ' ', '\t', '\n', '\r':
			if !inSpace {
				inSpace = true
			}
		default:
			if inSpace && b.Len() > 0 {
				b.WriteByte(' ')
			}
			inSpace = false
			b.WriteByte(c)
		}
	}
	return b.String()
}
