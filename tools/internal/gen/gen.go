// Package gen holds helpers shared by the helium code generators
// (tools/qt3gen). It is module-internal tooling, not part of the helium
// library API.
package gen

import (
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// CopyFile copies the file at src to dst, creating dst's parent directory if
// needed.
func CopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()
	_, err = io.Copy(out, in)
	return err
}

// DecodeXMLText decodes XML character/entity references and CDATA sections in s
// into their literal text.
func DecodeXMLText(s string) string {
	var b strings.Builder
	for len(s) > 0 {
		// Handle CDATA sections
		if strings.HasPrefix(s, "<![CDATA[") {
			end := strings.Index(s, "]]>")
			if end < 0 {
				b.WriteString(s[len("<![CDATA["):])
				break
			}
			b.WriteString(s[len("<![CDATA["):end])
			s = s[end+len("]]>"):]
			continue
		}
		// Handle entity/character references
		amp := strings.IndexByte(s, '&')
		cdata := strings.Index(s, "<![CDATA[")
		// Find the nearest special construct
		next := len(s)
		if amp >= 0 {
			next = amp
		}
		if cdata >= 0 && cdata < next {
			b.WriteString(s[:cdata])
			s = s[cdata:]
			continue
		}
		if amp < 0 {
			b.WriteString(s)
			break
		}
		b.WriteString(s[:amp])
		s = s[amp:]
		semi := strings.IndexByte(s, ';')
		if semi < 0 {
			b.WriteString(s)
			break
		}
		ref := s[1:semi]
		s = s[semi+1:]
		if strings.HasPrefix(ref, "#x") || strings.HasPrefix(ref, "#X") {
			if n, err := strconv.ParseInt(ref[2:], 16, 32); err == nil {
				b.WriteRune(rune(n))
			}
		} else if strings.HasPrefix(ref, "#") {
			if n, err := strconv.ParseInt(ref[1:], 10, 32); err == nil {
				b.WriteRune(rune(n))
			}
		} else {
			switch ref {
			case "lt":
				b.WriteByte('<')
			case "gt":
				b.WriteByte('>')
			case "amp":
				b.WriteByte('&')
			case "apos":
				b.WriteByte('\'')
			case "quot":
				b.WriteByte('"')
			default:
				b.WriteByte('&')
				b.WriteString(ref)
				b.WriteByte(';')
			}
		}
	}
	return b.String()
}

var nonIdentRE = regexp.MustCompile(`[^a-zA-Z0-9_]`)

// GoIdentifier converts s into a valid Go identifier fragment by replacing
// non-identifier characters with underscores.
func GoIdentifier(s string) string {
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, ".", "_")
	return nonIdentRE.ReplaceAllString(s, "_")
}
