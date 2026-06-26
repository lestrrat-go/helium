package encoding

// IsUTF8 returns true if the named encoding is UTF-8. The parser can use a
// direct byte cursor for these. US-ASCII is deliberately NOT included: it must
// route through Load's strict ASCII decoder, which rejects bytes >= 0x80.
func IsUTF8(name string) bool {
	switch normalizeEncodingName(name) {
	case "utf8", "unicode11utf8", "unicode20utf8", "xunicode20utf8":
		return true
	}
	return false
}

// IsASCII returns true if the named encoding is one of the US-ASCII aliases.
// Load maps these to a strict ASCII encoding whose encoder delegates to UTF-8,
// so callers that need byte-valid output for the declared encoding must detect
// ASCII separately.
func IsASCII(name string) bool {
	switch normalizeEncodingName(name) {
	case "usascii", "ascii", "ansix341968", "csascii":
		return true
	}
	return false
}
