package encoding

// IsUTF8 returns true if the named encoding is UTF-8 or ASCII (a subset of
// UTF-8). The parser can use a direct byte cursor for these.
func IsUTF8(name string) bool {
	switch normalizeEncodingName(name) {
	case "utf8", "unicode11utf8", "unicode20utf8", "xunicode20utf8",
		"usascii", "ascii", "ansix341968", "csascii":
		return true
	}
	return false
}
