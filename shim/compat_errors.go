package shim

import (
	stdxml "encoding/xml"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	helium "github.com/lestrrat-go/helium"
)

// convertParseError converts a helium parse error into an encoding/xml SyntaxError
// so that callers checking for *xml.SyntaxError get the expected type.
// It also maps helium error messages to stdlib's expected phrasing.
func convertParseError(err error) error {
	if err == nil {
		return nil
	}

	var pe helium.ErrParseError
	if errors.As(err, &pe) {
		msg := mapErrorMessage(pe)
		return &stdxml.SyntaxError{
			Line: pe.LineNumber,
			Msg:  msg,
		}
	}

	return err
}

// mapErrorMessage translates a helium parse error message to the equivalent
// stdlib encoding/xml error message.
func mapErrorMessage(pe helium.ErrParseError) string {
	raw := pe.Err.Error()

	switch raw {
	case "invalid name start char":
		// Check if this is actually an unexpected end element like </foo>
		if tag := extractEndElement(pe.Line); tag != "" {
			return fmt.Sprintf("unexpected end element </%s>", tag)
		}
		return "expected element name after <" //nolint:goconst
	}

	switch {
	case strings.HasPrefix(raw, "invalid name start char"):
		if tag := extractEndElement(pe.Line); tag != "" {
			return fmt.Sprintf("unexpected end element </%s>", tag)
		}
		return "expected element name after <"

	case strings.HasPrefix(raw, "failed to parse QName"):
		return "expected attribute name in element"

	case raw == "start tag expected, '<' not found":
		return raw

	case raw == "invalid char data":
		return raw

	case raw == "';' is required":
		if entity := extractEntityName(pe.Line); entity != "" {
			return fmt.Sprintf("invalid character entity &%s (no semicolon)", entity)
		}
		return raw

	case raw == "name is required":
		if entity := extractBadEntity(pe.Line); entity != "" {
			return fmt.Sprintf("invalid character entity &%s;", entity)
		}
		return "invalid character entity & (no semicolon)"

	case strings.HasPrefix(raw, "local name empty!"):
		if strings.Contains(raw, "failed to parse QName") {
			return "expected element name after <"
		}
		return raw

	case strings.HasPrefix(raw, "expected end tag '"):
		// "expected end tag 'zzz:foo'" → try to convert to stdlib format
		tag := raw[len("expected end tag '") : len(raw)-1]
		return convertEndTagMismatch(tag, pe.Line)

	case strings.HasPrefix(raw, "namespace '") && strings.HasSuffix(raw, "' not found"):
		// "namespace 'x' not found" → need to extract element info from context
		ns := raw[len("namespace '") : len(raw)-len("' not found")]
		return convertNamespaceError(ns, pe.Line)
	}

	return raw
}

// convertEndTagMismatch converts "expected end tag 'prefix:local'" to stdlib's
// "element <local> in space prefix closed by </local> in space \"\"" format.
func convertEndTagMismatch(expectedTag, contextLine string) string {
	// Parse prefix:local from expected tag
	prefix, local := splitPrefixLocal(expectedTag)

	// Try to find the closing tag name from context
	closeTag := ""
	if idx := strings.LastIndex(contextLine, "</"); idx >= 0 {
		rest := contextLine[idx+2:]
		if end := strings.IndexByte(rest, '>'); end >= 0 {
			closeTag = rest[:end]
		}
	}

	if closeTag != "" {
		closePrefix, closeLocal := splitPrefixLocal(closeTag)
		if local != closeLocal {
			return fmt.Sprintf("element <%s> closed by </%s>", expectedTag, closeTag)
		}
		if prefix != "" && closePrefix == "" {
			return fmt.Sprintf("element <%s> in space %s closed by </%s> in space \"\"",
				local, prefix, closeLocal)
		}
		if prefix != closePrefix {
			return fmt.Sprintf("element <%s> in space %s closed by </%s> in space %s",
				local, prefix, closeLocal, closePrefix)
		}
	}

	// If we couldn't find the close tag in context but the expected tag has
	// a prefix, the close tag is likely the local name without a prefix.
	if prefix != "" {
		return fmt.Sprintf("element <%s> in space %s closed by </%s> in space \"\"",
			local, prefix, local)
	}

	return fmt.Sprintf("expected end tag '%s'", expectedTag)
}

// convertNamespaceError converts namespace not found errors.
func convertNamespaceError(ns, contextLine string) string {
	// Parse context to find element names
	// "namespace 'x' not found" happens when <x:foo></y:foo> and 'x' is undeclared
	// Need to find both the open and close element names

	// Find element from context: look for the pattern in the context line
	if idx := strings.Index(contextLine, "<"+ns+":"); idx >= 0 {
		rest := contextLine[idx+1:]
		if end := strings.IndexAny(rest, " >"); end >= 0 {
			openTag := rest[:end]
			openPrefix, openLocal := splitPrefixLocal(openTag)

			// Look for close tag
			if closeIdx := strings.Index(contextLine, "</"); closeIdx >= 0 {
				closeRest := contextLine[closeIdx+2:]
				if closeEnd := strings.IndexByte(closeRest, '>'); closeEnd >= 0 {
					closeTag := closeRest[:closeEnd]
					closePrefix, closeLocal := splitPrefixLocal(closeTag)
					if openLocal == closeLocal && openPrefix != closePrefix {
						return fmt.Sprintf("element <%s> in space %s closed by </%s> in space %s",
							openLocal, openPrefix, closeLocal, closePrefix)
					}
				}
			}

			return fmt.Sprintf("namespace '%s' not found", ns)
		}
	}

	return fmt.Sprintf("namespace '%s' not found", ns)
}

// extractEndElement checks if the context line contains a </tag> pattern
// and returns the tag name. Used to detect "unexpected end element" errors.
func extractEndElement(line string) string {
	idx := strings.Index(line, "</")
	if idx < 0 {
		return ""
	}
	rest := line[idx+2:]
	end := strings.IndexAny(rest, "> \t\n\r")
	if end < 0 {
		if len(rest) > 0 {
			return rest
		}
		return ""
	}
	if end == 0 {
		return ""
	}
	return rest[:end]
}

func splitPrefixLocal(name string) (string, string) {
	if i := strings.IndexByte(name, ':'); i >= 0 {
		return name[:i], name[i+1:]
	}
	return "", name
}

// extractEntityName tries to find the entity name from a context line
// containing a broken entity reference like "&abc\x01;".
func extractEntityName(line string) string {
	idx := strings.LastIndex(line, "&")
	if idx < 0 {
		return ""
	}
	name := line[idx+1:]
	end := 0
	for end < len(name) {
		r, size := utf8.DecodeRuneInString(name[end:])
		if !isXMLNameChar(r) {
			break
		}
		end += size
	}
	if end == 0 {
		return ""
	}
	return name[:end]
}

// extractBadEntity tries to find a bad entity reference from context.
// For cases like "&\uFFFE;" it returns the bad character.
func extractBadEntity(line string) string {
	idx := strings.LastIndex(line, "&")
	if idx < 0 {
		return ""
	}
	rest := line[idx+1:]
	if len(rest) == 0 {
		return ""
	}
	r, size := utf8.DecodeRuneInString(rest)
	if !isXMLNameChar(r) && r != ';' {
		if size < len(rest) && rest[size] == ';' {
			return string(r)
		}
	}
	return ""
}

// isXMLNameChar returns true if r can appear in an XML name.
func isXMLNameChar(r rune) bool {
	if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
		return true
	}
	if r == '_' || r == '-' || r == '.' || r == ':' {
		return true
	}
	if r >= 0xC0 && r <= 0xD6 || r >= 0xD8 && r <= 0xF6 || r >= 0xF8 && r <= 0x2FF {
		return true
	}
	if r >= 0x300 && r <= 0x36F || r == 0xB7 {
		return true
	}
	if r >= 0x370 && r <= 0x37D || r >= 0x37F && r <= 0x1FFF {
		return true
	}
	return false
}
