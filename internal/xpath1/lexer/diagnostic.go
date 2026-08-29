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
	var excerpt strings.Builder
	excerpt.Grow(DiagnosticExcerptByteLimit)
	for len(s) > 0 {
		r, size := utf8.DecodeRuneInString(s)
		s = s[size:]

		if excerpt.Len()+utf8.RuneLen(r) > DiagnosticExcerptByteLimit {
			return truncateDiagnosticExcerpt(excerpt.String())
		}
		excerpt.WriteRune(r)
	}
	return excerpt.String()
}

func truncateDiagnosticExcerpt(s string) string {
	end := DiagnosticExcerptByteLimit - len(diagnosticTruncationMarker)
	for !utf8.RuneStart(s[end]) {
		end--
	}
	return s[:end] + diagnosticTruncationMarker
}
