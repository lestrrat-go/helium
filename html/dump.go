package html

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/lestrrat-go/helium"
)

// HTML attribute value escaping sequences.
var (
	htmlAttrEscAmp  = []byte("&amp;")
	htmlAttrEscLt   = []byte("&lt;")
	htmlAttrEscGt   = []byte("&gt;")
	htmlAttrEscQuot = []byte("&quot;")
)

// htmlURIAttrs is the set of HTML attributes that contain URIs.
// Based on libxml2's htmlAttrDumpOutput (HTMLtree.c).
// Note: libxml2 also includes "name" but that causes issues since name
// is used for non-URI purposes (meta name, form element name, etc.).
var htmlURIAttrs = map[string]bool{
	"href":   true,
	"action": true,
	"src":    true,
}

// defaultHTMLDTD is the default DOCTYPE for HTML documents without one.
// Mirrors libxml2's htmlDocContentDumpOutput behavior.
const defaultHTMLDTD = `<!DOCTYPE html PUBLIC "-//W3C//DTD HTML 4.0 Transitional//EN" "http://www.w3.org/TR/REC-html40/loose.dtd">` + "\n"

// DumpDoc serializes an HTML document to the writer.
// Mirrors libxml2's htmlDocContentDumpOutput.
func DumpDoc(out io.Writer, doc *helium.Document, options ...DumpOption) error {
	var cfg dumpConfig
	for _, o := range options {
		o(&cfg)
	}

	// If the document was parsed from Latin-1/Windows-1252, convert
	// UTF-8 output back to single-byte encoding to match libxml2.
	// strict=true for explicit ISO-8859-1 charset (numeric char refs for
	// runes > 0xFF); strict=false for auto-detected Win-1252 (raw bytes).
	enc := doc.Encoding()
	switch enc {
	case "ISO-8859-1":
		out = &latin1EncodingWriter{w: out, strict: true}
	case "Windows-1252":
		out = &latin1EncodingWriter{w: out, strict: false}
	}

	// Output DTD if present, or default DTD for HTML documents
	if dtd := doc.IntSubset(); dtd != nil {
		if err := dumpDTD(out, dtd); err != nil {
			return err
		}
	} else if !cfg.noDefaultDTD && doc.Type() == helium.HTMLDocumentNode {
		_, _ = io.WriteString(out, defaultHTMLDTD)
	}

	// Serialize all children of the document
	for child := doc.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Type() == helium.DTDNode {
			continue // already handled above
		}
		if err := dumpNode(out, child); err != nil {
			return err
		}
	}
	_, _ = io.WriteString(out, "\n")
	return nil
}

// DumpNode serializes an HTML node to the writer.
func DumpNode(out io.Writer, n helium.Node) error {
	return dumpNode(out, n)
}

func dumpNode(out io.Writer, n helium.Node) error {
	switch n.Type() {
	case helium.DocumentNode, helium.HTMLDocumentNode:
		return DumpDoc(out, n.(*helium.Document))
	case helium.DTDNode:
		return dumpDTD(out, n.(*helium.DTD))
	case helium.CommentNode:
		_, _ = io.WriteString(out, "<!--")
		_, _ = out.Write(n.Content())
		_, _ = io.WriteString(out, "-->")
		return nil
	case helium.ProcessingInstructionNode:
		_, _ = io.WriteString(out, "<?")
		_, _ = io.WriteString(out, n.Name())
		if c := n.Content(); len(c) > 0 {
			_, _ = io.WriteString(out, " ")
			_, _ = out.Write(c)
		}
		_, _ = io.WriteString(out, ">")
		return nil
	case helium.EntityRefNode:
		_, _ = io.WriteString(out, "&")
		_, _ = io.WriteString(out, n.Name())
		_, _ = io.WriteString(out, ";")
		return nil
	case helium.TextNode:
		return dumpText(out, n)
	case helium.ElementNode:
		return dumpElement(out, n.(*helium.Element))
	}
	return nil
}

// dumpDTD outputs <!DOCTYPE name PUBLIC "extID" "sysID">\n
func dumpDTD(out io.Writer, dtd *helium.DTD) error {
	_, _ = io.WriteString(out, "<!DOCTYPE ")
	_, _ = io.WriteString(out, dtd.Name())

	extID := dtd.ExternalID()
	sysID := dtd.SystemID()
	if extID != "" {
		_, _ = io.WriteString(out, " PUBLIC \"")
		_, _ = io.WriteString(out, extID)
		_, _ = io.WriteString(out, "\"")
		if sysID != "" {
			_, _ = io.WriteString(out, " \"")
			_, _ = io.WriteString(out, sysID)
			_, _ = io.WriteString(out, "\"")
		}
	} else if sysID != "" {
		_, _ = io.WriteString(out, " SYSTEM \"")
		_, _ = io.WriteString(out, sysID)
		_, _ = io.WriteString(out, "\"")
	}

	_, _ = io.WriteString(out, ">\n")
	return nil
}

// dumpText outputs text content, escaping &, <, > unless inside a raw text element.
func dumpText(out io.Writer, n helium.Node) error {
	parent := n.Parent()
	if parent != nil && parent.Type() == helium.ElementNode {
		parentName := strings.ToLower(parent.Name())
		if desc := lookupElement(parentName); desc != nil && desc.DataMode >= dataRawText {
			// Raw text element: no escaping
			_, _ = out.Write(n.Content())
			return nil
		}
	}

	// Normal text: escape &, <, >
	return htmlEscapeText(out, n.Content())
}

// htmlEscapeText escapes &, <, > in text content for HTML output.
// Unlike XML escaping, \n, \r, \t are NOT escaped.
func htmlEscapeText(w io.Writer, s []byte) error {
	var esc []byte
	last := 0
	for i := 0; i < len(s); {
		r, width := utf8.DecodeRune(s[i:])
		i += width
		switch r {
		case '&':
			esc = htmlAttrEscAmp
		case '<':
			esc = htmlAttrEscLt
		case '>':
			esc = htmlAttrEscGt
		default:
			continue
		}

		if _, err := w.Write(s[last : i-width]); err != nil {
			return err
		}
		if _, err := w.Write(esc); err != nil {
			return err
		}
		last = i
	}
	if _, err := w.Write(s[last:]); err != nil {
		return err
	}
	return nil
}

// htmlEscapeAttrValue escapes attribute values for HTML output.
// For non-URI attributes: escapes &, ", <, >.
// For URI attributes: escapes only & and " (matching libxml2's htmlAttrDumpOutput).
// Non-ASCII characters with named HTML4 entities are output as &name;.
func htmlEscapeAttrValue(w io.Writer, s string, isURI bool) error {
	last := 0
	for i := 0; i < len(s); {
		r, width := utf8.DecodeRune([]byte(s[i:]))
		var esc []byte
		switch {
		case r == '&':
			esc = htmlAttrEscAmp
		case r == '"':
			esc = htmlAttrEscQuot
		case r == '<' && !isURI:
			esc = htmlAttrEscLt
		case r == '>' && !isURI:
			esc = htmlAttrEscGt
		case r >= 0x80:
			if entName := lookupEntityByRune(r); entName != "" {
				if _, err := io.WriteString(w, s[last:i]); err != nil {
					return err
				}
				if _, err := io.WriteString(w, "&"); err != nil {
					return err
				}
				if _, err := io.WriteString(w, entName); err != nil {
					return err
				}
				if _, err := io.WriteString(w, ";"); err != nil {
					return err
				}
				i += width
				last = i
				continue
			}
			i += width
			continue
		default:
			i += width
			continue
		}
		if _, err := io.WriteString(w, s[last:i]); err != nil {
			return err
		}
		if _, err := w.Write(esc); err != nil {
			return err
		}
		i += width
		last = i
	}
	if _, err := io.WriteString(w, s[last:]); err != nil {
		return err
	}
	return nil
}

// dumpElement serializes an HTML element.
func dumpElement(out io.Writer, e *helium.Element) error {
	name := strings.ToLower(e.Name())
	info := lookupElement(name)

	// Opening tag
	_, _ = io.WriteString(out, "<")
	_, _ = io.WriteString(out, name)

	// Attributes
	if err := dumpAttributes(out, e); err != nil {
		return err
	}

	// Void element: no closing tag
	if info != nil && info.Empty {
		_, _ = io.WriteString(out, ">")
		// Format newline after void element
		if shouldNewlineAfterVoid(e, info) {
			_, _ = io.WriteString(out, "\n")
		}
		return nil
	}

	_, _ = io.WriteString(out, ">")

	// Format newline after opening tag (before children)
	if shouldNewlineAfterOpen(e, info) {
		_, _ = io.WriteString(out, "\n")
	}

	// Children
	for child := e.FirstChild(); child != nil; child = child.NextSibling() {
		if err := dumpNode(out, child); err != nil {
			return err
		}
	}

	// Format newline before closing tag
	if shouldNewlineBeforeClose(e, info) {
		_, _ = io.WriteString(out, "\n")
	}

	// Closing tag
	_, _ = io.WriteString(out, "</")
	_, _ = io.WriteString(out, name)
	_, _ = io.WriteString(out, ">")

	// Format newline after closing tag
	if shouldNewlineAfterClose(e, info) {
		_, _ = io.WriteString(out, "\n")
	}

	return nil
}

// dumpAttributes serializes HTML element attributes.
func dumpAttributes(out io.Writer, e *helium.Element) error {
	for _, attr := range e.Attributes() {
		attrName := strings.ToLower(attr.Name())
		_, _ = io.WriteString(out, " ")
		_, _ = io.WriteString(out, attrName)

		// Boolean attributes: just the name, no ="..."
		// Matches libxml2: if the attribute has no children (boolean attr
		// in the source), output just the name. Attributes with empty string
		// values (e.g., alt="") have an empty text child and get ="".
		if attr.FirstChild() == nil {
			continue
		}

		val := attr.Value()
		isURI := htmlURIAttrs[attrName]
		if isURI {
			val = uriEscapeStr(val)
		}

		_, _ = io.WriteString(out, "=\"")
		if err := htmlEscapeAttrValue(out, val, isURI); err != nil {
			return err
		}
		_, _ = io.WriteString(out, "\"")
	}
	return nil
}

// uriEscapeStr percent-encodes characters that are not URI-safe.
// Mirrors libxml2's xmlURIEscapeStr with allowed set "@/:=?;#%&,+".
// Unreserved chars (letters, digits, -_.~) and the allowed set are not encoded.
func uriEscapeStr(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		c := s[i]
		if isURISafe(c) {
			b.WriteByte(c)
			i++
		} else if c >= 0x80 {
			// Multi-byte UTF-8: percent-encode each byte
			_, width := utf8.DecodeRuneInString(s[i:])
			for j := 0; j < width; j++ {
				fmt.Fprintf(&b, "%%%02X", s[i+j])
			}
			i += width
		} else {
			fmt.Fprintf(&b, "%%%02X", c)
			i++
		}
	}
	return b.String()
}

// unicodeToWin1252 is the reverse mapping of win1252ToUnicode (parser.go).
// Maps Unicode codepoints back to Windows-1252 bytes 0x80-0x9F.
var unicodeToWin1252 = map[rune]byte{
	0x20AC: 0x80, 0x201A: 0x82, 0x0192: 0x83, 0x201E: 0x84,
	0x2026: 0x85, 0x2020: 0x86, 0x2021: 0x87, 0x02C6: 0x88,
	0x2030: 0x89, 0x0160: 0x8A, 0x2039: 0x8B, 0x0152: 0x8C,
	0x017D: 0x8E, 0x2018: 0x91, 0x2019: 0x92, 0x201C: 0x93,
	0x201D: 0x94, 0x2022: 0x95, 0x2013: 0x96, 0x2014: 0x97,
	0x02DC: 0x98, 0x2122: 0x99, 0x0161: 0x9A, 0x203A: 0x9B,
	0x0153: 0x9C, 0x017E: 0x9E, 0x0178: 0x9F,
}

// latin1EncodingWriter wraps an io.Writer and converts multi-byte UTF-8
// runes in the Latin-1/Windows-1252 range back to single bytes.
// When strict is true (explicit ISO-8859-1 charset), runes > 0xFF that
// have no Latin-1 representation are emitted as numeric character references.
// When strict is false (auto-detected Win-1252), Win-1252 reverse mapping
// is used and unmapped runes pass through as UTF-8.
type latin1EncodingWriter struct {
	w      io.Writer
	strict bool
}

func (lw *latin1EncodingWriter) Write(p []byte) (int, error) {
	out := utf8ToLatin1(p, lw.strict)
	_, err := lw.w.Write(out)
	// Report the original input length as consumed
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

// utf8ToLatin1 converts UTF-8 encoded data back to Latin-1/Windows-1252.
// Runes U+0080-U+00FF are written as single bytes.
// When strict is true (explicit ISO-8859-1), runes > U+00FF are emitted
// as numeric character references (&#N;).
// When strict is false (auto-detected Win-1252), Windows-1252 runes are
// reverse-mapped to single bytes and other runes pass through as UTF-8.
func utf8ToLatin1(data []byte, strict bool) []byte {
	// Fast path: if all bytes are ASCII, no conversion needed
	allASCII := true
	for _, b := range data {
		if b >= 0x80 {
			allASCII = false
			break
		}
	}
	if allASCII {
		return data
	}

	var buf bytes.Buffer
	buf.Grow(len(data))
	for i := 0; i < len(data); {
		b := data[i]
		if b < 0x80 {
			buf.WriteByte(b)
			i++
			continue
		}
		r, size := utf8.DecodeRune(data[i:])
		if r >= 0x80 && r <= 0xFF {
			buf.WriteByte(byte(r))
		} else if strict {
			fmt.Fprintf(&buf, "&#%d;", r)
		} else if wb, ok := unicodeToWin1252[r]; ok {
			buf.WriteByte(wb)
		} else {
			buf.Write(data[i : i+size])
		}
		i += size
	}
	return buf.Bytes()
}

// isURISafe returns true if the byte should NOT be percent-encoded.
// Matches libxml2's xmlURIEscapeStr unreserved set plus allowed set "@/:=?;#%&,+".
// Also includes <, >, \, and other printable chars that our parser may have
// produced by resolving entity refs (libxml2 preserves entity ref nodes and
// doesn't encounter these as literal chars during URI escaping).
func isURISafe(c byte) bool {
	if c >= 'A' && c <= 'Z' {
		return true
	}
	if c >= 'a' && c <= 'z' {
		return true
	}
	if c >= '0' && c <= '9' {
		return true
	}
	switch c {
	case '-', '_', '.', '~': // unreserved (RFC 3986)
		return true
	case '!', '*', '\'', '(', ')': // unreserved (libxml2)
		return true
	case '@', '/', ':', '=', '?', ';', '#', '%', '&', ',', '+': // libxml2 allowed set
		return true
	case '<', '>', '\\', '[', ']', '{', '}', '|', '^', '`', '"', '$': // chars from resolved entity refs
		return true
	}
	return false
}

// Format newline helper functions matching libxml2's HTMLtree.c:968-1108.

// parentNameStartsWithP checks if the parent element's name starts with 'p'.
func parentNameStartsWithP(n helium.Node) bool {
	parent := n.Parent()
	if parent == nil || parent.Type() != helium.ElementNode {
		return false
	}
	name := strings.ToLower(parent.Name())
	return len(name) > 0 && name[0] == 'p'
}

// isTextOrEntityRef checks if a node is a TextNode or EntityRefNode.
func isTextOrEntityRef(n helium.Node) bool {
	if n == nil {
		return false
	}
	t := n.Type()
	return t == helium.TextNode || t == helium.EntityRefNode
}

// shouldNewlineAfterOpen returns true if a \n should be inserted after the
// opening tag and before the first child.
func shouldNewlineAfterOpen(e *helium.Element, info *htmlElemDesc) bool {
	if info != nil && info.Inline != 0 {
		return false
	}
	first := e.FirstChild()
	if first == nil {
		return false
	}
	if isTextOrEntityRef(first) {
		return false
	}
	if e.FirstChild() == e.LastChild() {
		return false
	}
	name := strings.ToLower(e.Name())
	if len(name) > 0 && name[0] == 'p' {
		return false
	}
	return true
}

// shouldNewlineAfterVoid returns true if a \n should be inserted after a void element.
func shouldNewlineAfterVoid(e *helium.Element, info *htmlElemDesc) bool {
	if info != nil && info.Inline != 0 {
		return false
	}
	next := e.NextSibling()
	if next == nil {
		return false
	}
	if isTextOrEntityRef(next) {
		return false
	}
	if parentNameStartsWithP(e) {
		return false
	}
	return true
}

// shouldNewlineBeforeClose returns true if a \n should be inserted before
// the closing tag.
func shouldNewlineBeforeClose(e *helium.Element, info *htmlElemDesc) bool {
	if info != nil && info.Inline != 0 {
		return false
	}
	last := e.LastChild()
	if last == nil {
		return false
	}
	if isTextOrEntityRef(last) {
		return false
	}
	if e.FirstChild() == e.LastChild() {
		return false
	}
	name := strings.ToLower(e.Name())
	if len(name) > 0 && name[0] == 'p' {
		return false
	}
	return true
}

// shouldNewlineAfterClose returns true if a \n should be inserted after
// the closing tag.
func shouldNewlineAfterClose(e *helium.Element, info *htmlElemDesc) bool {
	if info != nil && info.Inline != 0 {
		return false
	}
	next := e.NextSibling()
	if next == nil {
		return false
	}
	if isTextOrEntityRef(next) {
		return false
	}
	if parentNameStartsWithP(e) {
		return false
	}
	return true
}
