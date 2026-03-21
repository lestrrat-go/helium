package xsd

import helium "github.com/lestrrat-go/helium"

// hasAttr checks whether an attribute is physically present on the element.
// Unlike getAttr, this distinguishes absent from empty-string.
func hasAttr(elem *helium.Element, name string) bool {
	for _, a := range elem.Attributes() {
		if a.LocalName() == name {
			return true
		}
	}
	return false
}

// isValidFinal checks if a value is valid for the 'final' attribute on elements.
func isValidFinal(v string) bool {
	if v == "#all" {
		return true
	}
	for _, part := range splitSpace(v) {
		if part != "extension" && part != "restriction" {
			return false
		}
	}
	return true
}

// isValidBlock checks if a value is valid for the 'block' attribute.
func isValidBlock(v string) bool {
	if v == "#all" {
		return true
	}
	for _, part := range splitSpace(v) {
		if part != "extension" && part != "restriction" && part != "substitution" {
			return false
		}
	}
	return true
}

// isValidFinalDefault checks if a value is valid for the 'finalDefault' attribute on xs:schema.
// Accepts #all or space-separated list of extension|restriction|list|union.
func isValidFinalDefault(v string) bool {
	if v == "#all" {
		return true
	}
	for _, part := range splitSpace(v) {
		if part != "extension" && part != "restriction" && part != "list" && part != "union" {
			return false
		}
	}
	return true
}

// splitSpace splits a string on whitespace.
func splitSpace(s string) []string {
	var parts []string
	start := -1
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r' {
			if start >= 0 {
				parts = append(parts, s[start:i])
				start = -1
			}
		} else if start < 0 {
			start = i
		}
	}
	if start >= 0 {
		parts = append(parts, s[start:])
	}
	return parts
}
