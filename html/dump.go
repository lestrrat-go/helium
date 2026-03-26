package html

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/lestrrat-go/helium"
	"golang.org/x/text/unicode/norm"
)

// HTML attribute value escaping sequences.
var (
	htmlAttrEscAmp  = []byte("&amp;")
	htmlAttrEscLt   = []byte("&lt;")
	htmlAttrEscGt   = []byte("&gt;")
	htmlAttrEscQuot = []byte("&quot;")
)

// htmlURIAttrs is the set of HTML attributes that always contain URIs.
// "name" is treated as a URI attribute only on <a> elements (see isURIAttr).
var htmlURIAttrs = map[string]bool{
	"href":   true,
	"action": true,
	"src":    true,
}

// defaultHTMLDTD is the default DOCTYPE for HTML documents without one.
// Mirrors libxml2's htmlDocContentDumpOutput behavior.
const defaultHTMLDTD = `<!DOCTYPE html PUBLIC "-//W3C//DTD HTML 4.0 Transitional//EN" "http://www.w3.org/TR/REC-html40/loose.dtd">` + "\n"

// WriteDoc serializes an HTML document to the writer
// (libxml2: htmlDocContentDumpOutput).
func (w Writer) WriteDoc(out io.Writer, doc *helium.Document) error {
	var cfg dumpConfig
	if w.cfg != nil {
		cfg = w.cfg.dumpConfig
	}

	d := htmlDumper{
		format:                !cfg.noFormat,
		preserveCase:          cfg.preserveCase,
		noDefaultDTD:          cfg.noDefaultDTD,
		noEscapeURIAttributes: cfg.noEscapeURIAttributes,
		escapeControlChars:    cfg.escapeControlChars,
	}
	return d.dumpDocument(out, doc)
}

func (d *htmlDumper) dumpDocument(out io.Writer, doc *helium.Document) error {
	if doc == nil {
		return nil
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
	} else if !d.noDefaultDTD && doc.Type() == helium.HTMLDocumentNode {
		_, _ = io.WriteString(out, defaultHTMLDTD)
	}

	// Serialize all children of the document
	for child := range helium.Children(doc) {
		if child.Type() == helium.DTDNode {
			continue // already handled above
		}
		if err := d.dumpNode(out, child); err != nil {
			return err
		}
	}
	if d.format {
		_, _ = io.WriteString(out, "\n")
	}
	return nil
}

type htmlDumper struct {
	format                bool
	preserveCase          bool
	noDefaultDTD          bool
	noEscapeURIAttributes bool
	escapeControlChars    bool
	// nsEmitted tracks namespace prefix→URI bindings that were actually
	// emitted (serialized) by ancestor elements. Used to suppress
	// redundant declarations.
	nsEmitted map[string]string
	// nsAvailable tracks ALL namespace prefix→URI bindings from ancestor
	// elements, including those suppressed on non-root HTML elements.
	// Used to find URIs for attribute namespace prefixes.
	nsAvailable map[string]string
}

// WriteNode serializes an HTML node to the writer
// (libxml2: htmlNodeDumpOutput).
func (w Writer) WriteNode(out io.Writer, n helium.Node) error {
	var cfg dumpConfig
	if w.cfg != nil {
		cfg = w.cfg.dumpConfig
	}
	d := htmlDumper{
		format:                !cfg.noFormat,
		preserveCase:          cfg.preserveCase,
		noDefaultDTD:          cfg.noDefaultDTD,
		noEscapeURIAttributes: cfg.noEscapeURIAttributes,
		escapeControlChars:    cfg.escapeControlChars,
	}
	return d.dumpNode(out, n)
}

func (d *htmlDumper) dumpNode(out io.Writer, n helium.Node) error {
	switch n.Type() {
	case helium.DocumentNode, helium.HTMLDocumentNode:
		return d.dumpDocument(out, n.(*helium.Document))
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
		return d.dumpText(out, n)
	case helium.ElementNode:
		return d.dumpElement(out, n.(*helium.Element))
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
	} else if sysID != "" && sysID != "about:legacy-compat" {
		_, _ = io.WriteString(out, " SYSTEM \"")
		_, _ = io.WriteString(out, sysID)
		_, _ = io.WriteString(out, "\"")
	}

	_, _ = io.WriteString(out, ">\n")
	return nil
}

// dumpText outputs text content, escaping &, <, > unless inside a raw text element.
func (d *htmlDumper) dumpText(out io.Writer, n helium.Node) error {
	parent := n.Parent()
	if parent != nil && parent.Type() == helium.ElementNode {
		parentName := strings.ToLower(parent.Name())
		if desc := lookupElement(parentName); desc != nil && desc.dataMode >= dataRawText {
			// Raw text element: no escaping
			_, _ = out.Write(n.Content())
			return nil
		}
	}

	// Normal text: escape &, <, >
	return htmlEscapeText(out, n.Content(), d.escapeControlChars)
}

// htmlEscapeText escapes &, <, > in text content for HTML output.
// Unlike XML escaping, \n, \r, \t are NOT escaped.
// When escCtrl is true, characters in the U+007F-U+009F range are emitted
// as hexadecimal numeric character references (HTML5 serialization).
func htmlEscapeText(w io.Writer, s []byte, escCtrl bool) error {
	var esc []byte
	last := 0
	for i := 0; i < len(s); {
		r, width := utf8.DecodeRune(s[i:])
		i += width
		switch {
		case r == '&':
			esc = htmlAttrEscAmp
		case r == '<':
			esc = htmlAttrEscLt
		case r == '>':
			esc = htmlAttrEscGt
		case escCtrl && r >= 0x7F && r <= 0x9F:
			if _, err := w.Write(s[last : i-width]); err != nil {
				return err
			}
			ref := fmt.Sprintf("&#x%X;", r)
			if _, err := io.WriteString(w, ref); err != nil {
				return err
			}
			last = i
			continue
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
// Non-ASCII characters with named HTML4 entities are output as &name;
// unless noEntityEnc is true (used by XSLT HTML output which emits UTF-8 directly).
func htmlEscapeAttrValue(w io.Writer, s string, isURI bool, noEntityEnc bool) error {
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
			if noEntityEnc {
				i += width
				continue
			}
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
func (d *htmlDumper) dumpElement(out io.Writer, e *helium.Element) error {
	nameLower := strings.ToLower(e.Name())
	info := lookupElement(nameLower)

	name := nameLower
	if d.preserveCase {
		name = e.Name()
	}

	// Opening tag
	_, _ = io.WriteString(out, "<")
	_, _ = io.WriteString(out, name)

	// Namespace declarations for HTML output with preserveCase (XSLT HTML5).
	//
	// Rules:
	// 1. Non-root HTML elements suppress prefixed namespace declarations
	//    (HTML parsers don't process them). The bindings are still tracked
	//    in nsAvailable so descendant non-HTML elements can pick them up.
	// 2. The root element and non-HTML elements emit namespace declarations
	//    that aren't already in scope (emitted by an ancestor).
	// 3. Non-HTML elements also emit namespace declarations for any
	//    prefixes used by their attributes that aren't already covered.
	if d.preserveCase {
		isRootElem := e.Parent() != nil && e.Parent().Type() != helium.ElementNode
		isHTMLElem := info != nil
		emitDecls := isRootElem || !isHTMLElem

		if emitDecls {
			// Emit declarations from this element's Namespaces() that
			// aren't already emitted by an ancestor.
			for _, ns := range e.Namespaces() {
				prefix := ns.Prefix()
				uri := ns.URI()
				if d.nsEmitted != nil && d.nsEmitted[prefix] == uri {
					continue
				}
				d.writeNSDecl(out, prefix, uri)
			}
			// For non-HTML elements, also emit namespace declarations
			// for attribute prefixes not already declared or emitted.
			if !isHTMLElem {
				d.emitAttrNSDecls(out, e)
			}
		}
	}

	// Attributes
	if err := d.dumpAttributes(out, e); err != nil {
		return err
	}

	// Void element: no closing tag
	if info != nil && info.empty {
		_, _ = io.WriteString(out, ">")
		if d.format && shouldNewlineAfterVoid(e, info) {
			_, _ = io.WriteString(out, "\n")
		}
		return nil
	}

	_, _ = io.WriteString(out, ">")

	if d.format && shouldNewlineAfterOpen(e, info) {
		_, _ = io.WriteString(out, "\n")
	}

	// Update namespace scope for children.
	var savedEmitted, savedAvailable map[string]string
	if d.preserveCase {
		isRootElem := e.Parent() != nil && e.Parent().Type() != helium.ElementNode
		isHTMLElem := info != nil
		emitDecls := isRootElem || !isHTMLElem

		savedEmitted = d.nsEmitted
		savedAvailable = d.nsAvailable
		newEmitted := copyMap(d.nsEmitted)
		newAvailable := copyMap(d.nsAvailable)
		for _, ns := range e.Namespaces() {
			newAvailable[ns.Prefix()] = ns.URI()
			if emitDecls {
				newEmitted[ns.Prefix()] = ns.URI()
			}
		}
		// Also track attribute namespace declarations that were emitted
		if emitDecls && !isHTMLElem {
			for _, attr := range e.Attributes() {
				if p := attr.Prefix(); p != "" && attr.URI() != "" {
					newEmitted[p] = attr.URI()
					newAvailable[p] = attr.URI()
				}
			}
		}
		d.nsEmitted = newEmitted
		d.nsAvailable = newAvailable
	}

	// Children
	for child := range helium.Children(e) {
		if err := d.dumpNode(out, child); err != nil {
			return err
		}
	}

	// Restore namespace scope
	if d.preserveCase {
		d.nsEmitted = savedEmitted
		d.nsAvailable = savedAvailable
	}

	if d.format && shouldNewlineBeforeClose(e, info) {
		_, _ = io.WriteString(out, "\n")
	}

	// Closing tag
	_, _ = io.WriteString(out, "</")
	_, _ = io.WriteString(out, name)
	_, _ = io.WriteString(out, ">")

	if d.format && shouldNewlineAfterClose(e, info) {
		_, _ = io.WriteString(out, "\n")
	}

	return nil
}

// htmlBooleanAttrs is the set of HTML attributes that should be minimized
// (output as just the attribute name with no value) per the XSLT/HTML
// serialization spec. See https://www.w3.org/TR/xslt-xquery-serialization-31/#HTML_BOOLEAN
var htmlBooleanAttrs = map[string]struct{}{
	"checked":  {},
	"compact":  {},
	"declare":  {},
	"defer":    {},
	"disabled": {},
	"ismap":    {},
	"multiple": {},
	"nohref":   {},
	"noresize": {},
	"noshade":  {},
	"nowrap":   {},
	"readonly": {},
	"selected": {},
}

// dumpAttributes serializes HTML element attributes.
func (d *htmlDumper) dumpAttributes(out io.Writer, e *helium.Element) error {
	for _, attr := range e.Attributes() {
		attrNameLower := strings.ToLower(attr.Name())
		attrName := attrNameLower
		if d.preserveCase {
			attrName = attr.Name()
		}
		_, _ = io.WriteString(out, " ")
		_, _ = io.WriteString(out, attrName)

		// Boolean attributes: just the name, no ="..."
		// Matches libxml2: if the attribute has no children (boolean attr
		// in the source), output just the name. Attributes with empty string
		// values (e.g., alt="") have an empty text child and get ="".
		if attr.FirstChild() == nil {
			continue
		}

		// HTML boolean attributes: minimize when value equals the
		// attribute name (case-insensitive), per XSLT serialization spec.
		if _, ok := htmlBooleanAttrs[attrNameLower]; ok {
			val := attr.Value()
			if strings.EqualFold(val, attrNameLower) {
				continue
			}
		}

		val := attr.Value()
		elemName := strings.ToLower(e.LocalName())
		isURI := htmlURIAttrs[attrNameLower] || (attrNameLower == "name" && elemName == "a")
		if isURI && !d.noEscapeURIAttributes {
			val = uriEscapeStr(val)
		}

		_, _ = io.WriteString(out, "=\"")
		if err := htmlEscapeAttrValue(out, val, isURI, d.preserveCase); err != nil {
			return err
		}
		_, _ = io.WriteString(out, "\"")
	}
	return nil
}

// writeNSDecl writes a single namespace declaration attribute.
func (d *htmlDumper) writeNSDecl(out io.Writer, prefix, uri string) {
	if prefix == "" {
		_, _ = io.WriteString(out, " xmlns=\"")
		_, _ = io.WriteString(out, uri)
		_, _ = io.WriteString(out, "\"")
	} else {
		_, _ = io.WriteString(out, " xmlns:")
		_, _ = io.WriteString(out, prefix)
		_, _ = io.WriteString(out, "=\"")
		_, _ = io.WriteString(out, uri)
		_, _ = io.WriteString(out, "\"")
	}
}

// emitAttrNSDecls emits namespace declarations for attribute prefixes
// that aren't already declared on the element or emitted by an ancestor.
func (d *htmlDumper) emitAttrNSDecls(out io.Writer, e *helium.Element) {
	// Build set of prefixes already declared on this element
	declared := make(map[string]struct{})
	for _, ns := range e.Namespaces() {
		declared[ns.Prefix()] = struct{}{}
	}
	// Check each attribute's prefix
	seen := make(map[string]struct{})
	for _, attr := range e.Attributes() {
		prefix := attr.Prefix()
		if prefix == "" {
			continue
		}
		if _, ok := declared[prefix]; ok {
			continue
		}
		if _, ok := seen[prefix]; ok {
			continue
		}
		uri := attr.URI()
		if uri == "" {
			continue
		}
		if d.nsEmitted != nil && d.nsEmitted[prefix] == uri {
			continue
		}
		seen[prefix] = struct{}{}
		d.writeNSDecl(out, prefix, uri)
	}
}

// copyMap returns a shallow copy of a string map. If src is nil, returns
// an empty map.
func copyMap(src map[string]string) map[string]string {
	m := make(map[string]string, len(src))
	for k, v := range src {
		m[k] = v
	}
	return m
}

// uriEscapeStr percent-encodes characters that are not URI-safe.
// Mirrors libxml2's xmlURIEscapeStr with allowed set "@/:=?;#%&,+".
// Unreserved chars (letters, digits, -_.~) and the allowed set are not encoded.
// The input is normalized to NFC before encoding per IRI-to-URI conversion
// (RFC 3987 §3.1).
func uriEscapeStr(s string) string {
	// Normalize to NFC before percent-encoding so that precomposed forms
	// (e.g. U+00E5 å) are used instead of decomposed sequences (a + U+030A).
	s = norm.NFC.String(s)
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
	if info != nil && info.inline != 0 {
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
	if info != nil && info.inline != 0 {
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
	if info != nil && info.inline != 0 {
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
	if info != nil && info.inline != 0 {
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
