// Package xmlchar provides XML 1.0 NCName character classification functions.
package xmlchar

import (
	"strings"
	"unicode/utf8"
)

// IsChar reports whether r is a valid XML 1.0 §2.2 Char.
// Char ::= #x9 | #xA | #xD | [#x20-#xD7FF] | [#xE000-#xFFFD] | [#x10000-#x10FFFF]
func IsChar(r rune) bool {
	if r < 0x100 {
		return r == 0x9 || r == 0xA || r == 0xD || r >= 0x20
	}
	return (r >= 0x100 && r <= 0xD7FF) || (r >= 0xE000 && r <= 0xFFFD) || (r >= 0x10000 && r <= 0x10FFFF)
}

// IsNCNameStartChar checks the XML 1.0 NCName start character production.
// NCNameStartChar ::= [A-Z] | "_" | [a-z] | [#xC0-#xD6] | [#xD8-#xF6]
//
//	| [#xF8-#x2FF] | [#x370-#x37D] | [#x37F-#x1FFF] | [#x200C-#x200D]
//	| [#x2070-#x218F] | [#x2C00-#x2FEF] | [#x3001-#xD7FF]
//	| [#xF900-#xFDCF] | [#xFDF0-#xFFFD] | [#x10000-#xEFFFF]
func IsNCNameStartChar(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || r == '_' ||
		(r >= 0xC0 && r <= 0xD6) || (r >= 0xD8 && r <= 0xF6) ||
		(r >= 0xF8 && r <= 0x2FF) || (r >= 0x370 && r <= 0x37D) ||
		(r >= 0x37F && r <= 0x1FFF) || (r >= 0x200C && r <= 0x200D) ||
		(r >= 0x2070 && r <= 0x218F) || (r >= 0x2C00 && r <= 0x2FEF) ||
		(r >= 0x3001 && r <= 0xD7FF) || (r >= 0xF900 && r <= 0xFDCF) ||
		(r >= 0xFDF0 && r <= 0xFFFD) || (r >= 0x10000 && r <= 0xEFFFF)
}

// IsNCNameChar checks the XML 1.0 NCName character production.
// NCNameChar ::= NCNameStartChar | "-" | "." | [0-9] | #xB7
//
//	| [#x0300-#x036F] | [#x203F-#x2040]
func IsNCNameChar(r rune) bool {
	return IsNCNameStartChar(r) ||
		(r >= '0' && r <= '9') || r == '-' || r == '.' ||
		r == 0xB7 || (r >= 0x0300 && r <= 0x036F) || (r >= 0x203F && r <= 0x2040)
}

// IsValidPITarget reports whether target is a valid XML processing
// instruction target. A PI target is an NCName (colons are forbidden, matching
// helium's parser, which rejects colons in PI targets) and must not be the
// reserved name "xml" (any case).
func IsValidPITarget(target string) bool {
	if target == "" {
		return false
	}
	if strings.EqualFold(target, "xml") {
		return false
	}
	// Ranging over an invalid byte yields utf8.RuneError (U+FFFD), which is
	// itself a valid NCName character, so IsValidNCName would accept invalid
	// UTF-8 and emit raw bytes. Reject invalid encodings up front; a genuinely
	// encoded U+FFFD is valid UTF-8 and still passes.
	if !utf8.ValidString(target) {
		return false
	}
	return IsValidNCName(target)
}

// IsValidQName checks whether s is a valid XML QName (Namespaces in XML §3):
//
//	QName ::= PrefixedName | UnprefixedName
//	PrefixedName ::= Prefix ':' LocalPart
//	UnprefixedName ::= LocalPart
//
// where Prefix and LocalPart are each an NCName. An unprefixed name is a bare
// NCName; a prefixed name has exactly one colon separating two NCNames.
func IsValidQName(s string) bool {
	if s == "" {
		return false
	}
	if prefix, local, found := strings.Cut(s, ":"); found {
		// Exactly one colon, splitting two NCNames. A second colon makes
		// the local part fail IsValidNCName, so no explicit second-colon
		// check is needed.
		return IsValidNCName(prefix) && IsValidNCName(local)
	}
	return IsValidNCName(s)
}

// IsValidNCName checks whether s is a valid XML NCName (non-colonized name).
func IsValidNCName(s string) bool {
	if s == "" {
		return false
	}
	// Decode explicitly (not range) so invalid UTF-8 — which range reports as
	// RuneError indistinguishable from a real U+FFFD — is rejected by width.
	first := true
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			return false
		}
		if first {
			if !IsNCNameStartChar(r) {
				return false
			}
			first = false
		} else if !IsNCNameChar(r) {
			return false
		}
		i += size
	}
	return true
}
