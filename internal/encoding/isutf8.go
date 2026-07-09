package encoding

import "golang.org/x/text/encoding/unicode"

// UnicodeBOMFamily classifies a declared encoding name into the canonical
// Unicode family relevant to byte-order-mark conflict detection, deriving the
// answer entirely from Load's own alias resolution (no parallel alias table):
//
//   - "utf-8"     — Load resolves the name to UTF-8
//   - "utf-16be"  — Load resolves it to a fixed big-endian UTF-16 (no BOM use)
//   - "utf-16le"  — Load resolves it to a fixed little-endian UTF-16
//   - "utf-16"    — Load resolves it to a generic UTF-16 whose endianness is
//     taken from the BOM (compatible with either UTF-16 BOM)
//   - ""          — the name is unresolved or resolves to any non-UTF-16/UTF-8
//     encoding (US-ASCII, a charmap, UTF-32/UCS-4, ...)
//
// Because it consults Load, every alias Load accepts (unicode-1-1-utf-8,
// unicodefffe, csunicode, ...) is classified without a separate mirror.
func UnicodeBOMFamily(name string) string {
	e := Load(name)
	if e == nil {
		return ""
	}
	if e == unicode.UTF8 {
		return "utf-8"
	}
	se, ok := e.(*strictEncoding)
	if !ok || se.width != 2 {
		return ""
	}
	if se.useBOM {
		return "utf-16"
	}
	if len(se.order.perm) == 2 && se.order.perm[0] == 0 {
		return "utf-16be"
	}
	return "utf-16le"
}

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
