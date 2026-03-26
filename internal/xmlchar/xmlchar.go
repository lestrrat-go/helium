// Package xmlchar provides XML 1.0 NCName character classification functions.
package xmlchar

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

// IsValidNCName checks whether s is a valid XML NCName (non-colonized name).
func IsValidNCName(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 && !IsNCNameStartChar(r) {
			return false
		}
		if i > 0 && !IsNCNameChar(r) {
			return false
		}
	}
	return true
}
