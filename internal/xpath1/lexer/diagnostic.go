package lexer

import (
	"strings"
	"unicode/utf8"
)

// DiagnosticExcerptByteLimit is the maximum byte length returned by
// DiagnosticExcerpt, including its truncation marker.
const DiagnosticExcerptByteLimit = 128

const diagnosticTruncationMarker = "...[truncated]"

// DiagnosticExcerpt returns a bounded, valid UTF-8 excerpt for diagnostics.
// Valid UTF-8 strings within DiagnosticExcerptByteLimit are returned unchanged.
func DiagnosticExcerpt(s string) string {
	s = strings.ToValidUTF8(s, "\uFFFD")
	if len(s) <= DiagnosticExcerptByteLimit {
		return s
	}

	end := DiagnosticExcerptByteLimit - len(diagnosticTruncationMarker)
	for !utf8.RuneStart(s[end]) {
		end--
	}
	return s[:end] + diagnosticTruncationMarker
}
