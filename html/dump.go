package html

import (
	"bytes"
	"fmt"
	"io"
	"maps"
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

// Write serializes an HTML node to the given writer using default settings.
func Write(out io.Writer, node helium.Node) error {
	return NewWriter().WriteTo(out, node)
}

// WriteString serializes an HTML node to a string using default settings.
func WriteString(node helium.Node) (string, error) {
	var buf strings.Builder
	if err := Write(&buf, node); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// WriteTo serializes an HTML node to the given writer.
func (w Writer) WriteTo(out io.Writer, node helium.Node) error {
	cfg := w.dumpConfig
	d := htmlDumper{
		format:                !cfg.noFormat,
		preserveCase:          cfg.preserveCase,
		noDefaultDTD:          cfg.noDefaultDTD,
		noEscapeURIAttributes: cfg.noEscapeURIAttributes,
		escapeControlChars:    cfg.escapeControlChars,
		nullNamespaceHTMLOnly: cfg.nullNamespaceHTMLOnly,
		charMap:               cfg.charMap,
		normalize:             cfg.normalize,
		normForm:              cfg.normForm,
	}
	return d.dumpNode(out, node)
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
	case encISO88591:
		out = &latin1EncodingWriter{w: out, strict: true}
	case encWindows1252:
		out = &latin1EncodingWriter{w: out, strict: false}
	}

	// Output DTD if present, or default DTD for HTML documents
	if dtd := doc.IntSubset(); dtd != nil {
		d.dumpDTD(out, dtd)
	} else if !d.noDefaultDTD && doc.Type() == helium.HTMLDocumentNode {
		d.writeString(out, defaultHTMLDTD)
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
		d.writeString(out, "\n")
	}
	return d.err
}

type htmlDumper struct {
	format                bool
	preserveCase          bool
	noDefaultDTD          bool
	noEscapeURIAttributes bool
	escapeControlChars    bool
	nullNamespaceHTMLOnly bool
	// charMap substitutes a mapped rune in text and attribute-value content with
	// its literal (unescaped) replacement string (Serialization 3.1 character
	// maps). Empty/nil disables the feature.
	charMap map[rune]string
	// normalize / normForm request Unicode normalization of text-node and
	// attribute-value character content (Serialization 3.1 §4), scoped to text and
	// attribute nodes. false keeps output byte-identical.
	normalize bool
	normForm  norm.Form
	// err is a sticky serialization error. Once set, all checked write
	// helpers become no-ops and terminal methods return it. This mirrors
	// the writeSession sticky-error pattern in writer.go so that a writer
	// failing mid-stream surfaces an error instead of silently truncating.
	err error
	// nsEmitted tracks namespace prefix→URI bindings that were actually
	// emitted (serialized) by ancestor elements. Used to suppress
	// redundant declarations.
	nsEmitted map[string]string
	// nsAvailable tracks ALL namespace prefix→URI bindings from ancestor
	// elements, including those suppressed on non-root HTML elements.
	// Used to find URIs for attribute namespace prefixes.
	nsAvailable map[string]string
}

// check records err as the sticky error if no error has been recorded yet.
func (d *htmlDumper) check(err error) {
	if d.err == nil && err != nil {
		d.err = err
	}
}

// writeString writes s to out, recording the first failure as the sticky
// error. It is a no-op once an error has been recorded. A short write
// reported with a nil error by a non-conformant io.Writer is promoted to
// io.ErrShortWrite, mirroring latin1EncodingWriter, so silent truncation
// is never reported as success.
func (d *htmlDumper) writeString(out io.Writer, s string) {
	if d.err != nil {
		return
	}
	n, err := io.WriteString(out, s)
	if err == nil && n < len(s) {
		err = io.ErrShortWrite
	}
	d.check(err)
}

// checkName verifies that an element or attribute name is safe to write into
// an HTML tag. Names built through public DOM construction paths
// (CreateElement, SetAttribute, ...) are not validated, so a name
// containing characters that terminate or escape a tag — whitespace, quotes,
// '<', '>', '=', '/', or control characters — would otherwise be written
// verbatim and produce malformed or injected markup. When name is unsafe (or
// empty), the first such failure is recorded as the sticky error so
// serialization aborts before writing.
//
// This is intentionally permissive about characters HTML tolerates but XML
// does not (e.g. '?' or '.' in "gentus?.?"), matching libxml2's HTML
// serializer, which preserves such names verbatim. Only characters that can
// break out of the tag are rejected.
func (d *htmlDumper) checkName(kind, name string) {
	if d.err != nil {
		return
	}
	if name == "" {
		d.check(fmt.Errorf("invalid HTML %s name: empty", kind))
		return
	}
	// Decode rune-by-rune rather than ranging over the string: a range loop
	// yields utf8.RuneError both for genuinely invalid bytes and for a
	// validly-encoded U+FFFD, so we cannot tell them apart by rune alone.
	// DecodeRuneInString returns size==1 for invalid encodings, letting us
	// reject only the former while permitting a real U+FFFD (size==3), which
	// the parser accepts and which does not break out of the tag.
	for i := 0; i < len(name); {
		r, size := utf8.DecodeRuneInString(name[i:])
		if r == utf8.RuneError && size == 1 {
			d.check(fmt.Errorf("invalid HTML %s name %q", kind, name))
			return
		}
		if isUnsafeNameRune(r) {
			d.check(fmt.Errorf("invalid HTML %s name %q", kind, name))
			return
		}
		i += size
	}
}

// isUnsafeNameRune reports whether r may not appear in a serialized HTML
// element or attribute name because it would terminate or escape the tag.
// The set mirrors the characters that terminate an attribute name in the
// HTML parser (see parser.parseAttrName) plus ASCII control characters.
// '&' is intentionally excluded: the parser's liberal attribute-name rule
// accepts it and it does not break out of the tag, so rejecting it would
// regress parse/serialize round-trip parity. A validly-encoded U+FFFD is
// likewise accepted; only invalid UTF-8 is rejected, by the caller.
//
// This is the loose rule used for ATTRIBUTE names. Element names use the
// stricter checkElementName / isElementNameRune grammar instead.
func isUnsafeNameRune(r rune) bool {
	switch r {
	case ' ', '\t', '\n', '\r', '\f', '"', '\'', '<', '>', '=', '/':
		return true
	}
	return r < 0x20 || r == 0x7f
}

// checkElementName verifies that an element name is a valid HTML tag name
// before it is written into a tag. Unlike checkName (the loose attribute
// rule), this enforces the HTML tag-name grammar so that names libxml2 treats
// as malformed — e.g. CreateElement("a?b") or CreateElement("a&b") — are
// rejected rather than serialized verbatim as <a?b> / <a&b>.
//
// The accepted set is derived from what the HTML parser tokenizes as a tag
// name (parser.parseName via isNameChar): ASCII letters, digits, ':', '-',
// '_', '.'. ASCII punctuation outside that set — '?', '&', '=', '/', quotes,
// angle brackets, whitespace, control chars — is rejected. Non-ASCII runes
// are permitted so that legitimate Unicode element names (e.g. produced by
// XSLT HTML output) are not over-rejected; a validly-encoded U+FFFD is one
// such rune and is accepted, while invalid UTF-8 is rejected.
func (d *htmlDumper) checkElementName(name string) {
	if d.err != nil {
		return
	}
	if name == "" {
		d.check(fmt.Errorf("invalid HTML element name: empty"))
		return
	}
	// Decode rune-by-rune rather than ranging over the string: a range loop
	// yields utf8.RuneError both for genuinely invalid bytes and for a
	// validly-encoded U+FFFD. DecodeRuneInString returns size==1 for invalid
	// encodings, letting us reject only the former while permitting a real
	// U+FFFD (size==3), which the parser accepts and which does not break out
	// of the tag.
	for i := 0; i < len(name); {
		r, size := utf8.DecodeRuneInString(name[i:])
		if r == utf8.RuneError && size == 1 {
			d.check(fmt.Errorf("invalid HTML element name %q", name))
			return
		}
		if !isElementNameRune(r) {
			d.check(fmt.Errorf("invalid HTML element name %q", name))
			return
		}
		i += size
	}
}

// isElementNameRune reports whether r may appear in a serialized HTML element
// name. ASCII runes must be a valid HTML tag-name character (letters, digits,
// ':', '-', '_', '.'), mirroring the parser's isNameChar; this rejects
// tag-breaking and clearly-malformed characters such as '?', '&', '=', '/',
// quotes, angle brackets, whitespace and control characters. Non-ASCII runes
// are permitted to avoid over-rejecting legitimate Unicode element names.
func isElementNameRune(r rune) bool {
	if r >= 0x80 {
		return true
	}
	switch {
	case r >= 'A' && r <= 'Z':
		return true
	case r >= 'a' && r <= 'z':
		return true
	case r >= '0' && r <= '9':
		return true
	}
	switch r {
	case ':', '-', '_', '.':
		return true
	}
	return false
}

// writeBytes writes b to out, recording the first failure as the sticky
// error. It is a no-op once an error has been recorded. A short write
// reported with a nil error by a non-conformant io.Writer is promoted to
// io.ErrShortWrite, mirroring latin1EncodingWriter, so silent truncation
// is never reported as success.
func (d *htmlDumper) writeBytes(out io.Writer, b []byte) {
	if d.err != nil {
		return
	}
	n, err := out.Write(b)
	if err == nil && n < len(b) {
		err = io.ErrShortWrite
	}
	d.check(err)
}

func (d *htmlDumper) dumpNode(out io.Writer, n helium.Node) error {
	switch n.Type() {
	case helium.DocumentNode, helium.HTMLDocumentNode:
		doc, ok := helium.AsNode[*helium.Document](n)
		if !ok {
			return nil
		}
		return d.dumpDocument(out, doc)
	case helium.DTDNode:
		dtd, ok := helium.AsNode[*helium.DTD](n)
		if !ok {
			return nil
		}
		d.dumpDTD(out, dtd)
		return d.err
	case helium.CommentNode:
		d.writeString(out, "<!--")
		d.writeBytes(out, n.Content())
		d.writeString(out, "-->")
		return d.err
	case helium.ProcessingInstructionNode:
		d.writeString(out, "<?")
		d.writeString(out, n.Name())
		if c := n.Content(); len(c) > 0 {
			d.writeString(out, " ")
			d.writeBytes(out, c)
		}
		d.writeString(out, ">")
		return d.err
	case helium.EntityRefNode:
		d.writeString(out, "&")
		d.writeString(out, n.Name())
		d.writeString(out, ";")
		return d.err
	case helium.TextNode:
		return d.dumpText(out, n)
	case helium.ElementNode:
		elem, ok := helium.AsNode[*helium.Element](n)
		if !ok {
			return nil
		}
		return d.dumpElement(out, elem)
	}
	return nil
}

// dumpDTD outputs <!DOCTYPE name PUBLIC "extID" "sysID">\n
func (d *htmlDumper) dumpDTD(out io.Writer, dtd *helium.DTD) {
	d.writeString(out, "<!DOCTYPE ")
	d.writeString(out, dtd.Name())

	extID := dtd.ExternalID()
	sysID := dtd.SystemID()
	if extID != "" {
		d.writeString(out, " PUBLIC \"")
		d.writeString(out, extID)
		d.writeString(out, "\"")
		if sysID != "" {
			d.writeString(out, " \"")
			d.writeString(out, sysID)
			d.writeString(out, "\"")
		}
	} else if sysID != "" && sysID != "about:legacy-compat" {
		d.writeString(out, " SYSTEM \"")
		d.writeString(out, sysID)
		d.writeString(out, "\"")
	}

	d.writeString(out, ">\n")
}

// htmlNormalizationForm maps a normalization-form parameter name to its
// golang.org/x/text norm.Form and reports whether normalization is active.
func htmlNormalizationForm(form string) (norm.Form, bool) {
	switch form {
	case "NFC":
		return norm.NFC, true
	case "NFD":
		return norm.NFD, true
	case "NFKC":
		return norm.NFKC, true
	case "NFKD":
		return norm.NFKD, true
	}
	return norm.NFC, false
}

// htmlContentSegment is one piece of a text or attribute node's character
// content after pre-normalization character-map matching: either a normalized
// run of non-mapped characters the escape funnel processes normally, or the
// replacement for one mapped input rune, which the caller emits verbatim.
type htmlContentSegment struct {
	// text is the normalized non-mapped run (mapped is false).
	text []byte
	// repl is the character-map replacement for one mapped input rune (mapped
	// is true).
	repl   string
	mapped bool
}

// normalizeContent applies the dumper's requested Unicode normalization to a
// text or attribute node's character content and returns it as segments.
// Character-map matching is decided on the PRE-normalization content
// (Serialization 3.1 §4 applies character mapping — rule c — before Unicode
// normalization — rule d — and never re-applies it): the content is split at
// each mapped input rune, every maximal run of non-mapped characters is
// normalized on its own, and each mapped rune becomes a replacement segment the
// caller emits verbatim — never normalized (§11) and never escaped. Splitting
// keeps a literal content rune, whatever codepoint it is, from ever being read
// as a replacement placeholder. Only called when d.normalize is true.
func (d *htmlDumper) normalizeContent(s []byte) []htmlContentSegment {
	if len(d.charMap) == 0 {
		return []htmlContentSegment{{text: d.normForm.Bytes(s)}}
	}
	var segs []htmlContentSegment
	seg := 0
	for i := 0; i < len(s); {
		r, width := utf8.DecodeRune(s[i:])
		repl, mapped := d.charMap[r]
		if !mapped {
			i += width
			continue
		}
		// Close the non-mapped run ending just before this mapped character,
		// then record the replacement for exactly this pre-normalization
		// occurrence.
		if i > seg {
			segs = append(segs, htmlContentSegment{text: d.normForm.Bytes(s[seg:i])})
		}
		segs = append(segs, htmlContentSegment{repl: repl, mapped: true})
		i += width
		seg = i
	}
	if segs == nil {
		// No mapped rune in the content: normalize it whole.
		return []htmlContentSegment{{text: d.normForm.Bytes(s)}}
	}
	if seg < len(s) {
		segs = append(segs, htmlContentSegment{text: d.normForm.Bytes(s[seg:])})
	}
	return segs
}

// dumpText outputs text content, escaping &, <, > unless inside a raw text element.
func (d *htmlDumper) dumpText(out io.Writer, n helium.Node) error {
	content := n.Content()
	rawText := false
	parent := n.Parent()
	if parent != nil && parent.Type() == helium.ElementNode {
		parentName := strings.ToLower(parent.Name())
		if desc := lookupElement(parentName); desc != nil && desc.dataMode >= dataRawText {
			rawText = true
		}
	}

	// Unicode normalization is scoped to text nodes: normalizeContent splits the
	// content at each mapped rune (matching decided on the pre-normalization
	// content), each non-mapped run is normalized and escaped with a nil map, and
	// each replacement is emitted verbatim.
	if d.normalize {
		return d.writeNormalizedText(out, content, rawText)
	}

	if rawText {
		// Raw text element: no escaping (character maps still apply).
		if len(d.charMap) == 0 {
			d.writeBytes(out, content)
		} else {
			d.check(writeHTMLCharMapped(out, content, d.charMap))
		}
		return d.err
	}

	// Normal text: escape &, <, >
	if d.err != nil {
		return d.err
	}
	d.check(htmlEscapeText(out, content, d.escapeControlChars, d.charMap))
	return d.err
}

// writeNormalizedText writes a text node's character content under an active
// Normalization request: each non-mapped segment is normalized and (outside a
// raw text element) escaped as ordinary text with no character map, and each
// replacement segment is emitted verbatim. Only called when d.normalize is true.
func (d *htmlDumper) writeNormalizedText(out io.Writer, content []byte, rawText bool) error {
	for _, seg := range d.normalizeContent(content) {
		if d.err != nil {
			return d.err
		}
		if seg.mapped {
			d.writeString(out, seg.repl)
			continue
		}
		if rawText {
			d.writeBytes(out, seg.text)
			continue
		}
		d.check(htmlEscapeText(out, seg.text, d.escapeControlChars, nil))
	}
	return d.err
}

// writeNormalizedAttrValue writes a non-URI-escaped attribute value under an
// active Normalization request: each non-mapped segment is normalized and
// escaped with a nil map, and each replacement segment is emitted verbatim.
// Only called when d.normalize is true.
func (d *htmlDumper) writeNormalizedAttrValue(out io.Writer, val string, isURI bool) {
	for _, seg := range d.normalizeContent([]byte(val)) {
		if d.err != nil {
			return
		}
		if seg.mapped {
			d.writeString(out, seg.repl)
			continue
		}
		d.check(htmlEscapeAttrValue(out, string(seg.text), isURI, d.preserveCase, nil))
	}
}

// writeHTMLCharMapped writes s to w, substituting each character-map rune with
// its literal (unescaped) replacement string. With a nil/empty map it writes s
// unchanged.
func writeHTMLCharMapped(w io.Writer, s []byte, charMap map[rune]string) error {
	if len(charMap) == 0 {
		_, err := w.Write(s)
		return err
	}
	last := 0
	for i := 0; i < len(s); {
		r, width := utf8.DecodeRune(s[i:])
		if repl, ok := charMap[r]; ok {
			if _, err := w.Write(s[last:i]); err != nil {
				return err
			}
			if _, err := io.WriteString(w, repl); err != nil {
				return err
			}
			i += width
			last = i
			continue
		}
		i += width
	}
	_, err := w.Write(s[last:])
	return err
}

// htmlEscapeText escapes &, <, > in text content for HTML output.
// Unlike XML escaping, \n, \r, \t are NOT escaped.
// When escCtrl is true, characters in the U+007F-U+009F range are emitted
// as hexadecimal numeric character references (HTML5 serialization).
func htmlEscapeText(w io.Writer, s []byte, escCtrl bool, charMap map[rune]string) error {
	var esc []byte
	last := 0
	for i := 0; i < len(s); {
		r, width := utf8.DecodeRune(s[i:])
		i += width
		if repl, ok := charMap[r]; ok {
			// Character map: emit the literal replacement verbatim (not escaped).
			if _, err := w.Write(s[last : i-width]); err != nil {
				return err
			}
			if _, err := io.WriteString(w, repl); err != nil {
				return err
			}
			last = i
			continue
		}
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
func htmlEscapeAttrValue(w io.Writer, s string, isURI bool, noEntityEnc bool, charMap map[rune]string) error {
	last := 0
	for i := 0; i < len(s); {
		r, width := utf8.DecodeRuneInString(s[i:])
		if repl, ok := charMap[r]; ok {
			// Character map: emit the literal replacement verbatim (not escaped).
			if _, err := io.WriteString(w, s[last:i]); err != nil {
				return err
			}
			if _, err := io.WriteString(w, repl); err != nil {
				return err
			}
			i += width
			last = i
			continue
		}
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

	// Reject element names that are not valid HTML tag names (malformed names
	// such as "a?b" or "a&b", or names containing tag-breaking characters)
	// before writing them into the tag.
	d.checkElementName(name)
	if d.err != nil {
		return d.err
	}

	// Opening tag
	d.writeString(out, "<")
	d.writeString(out, name)

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

	// Void element: no closing tag. Under HTML 4.01 rules
	// (nullNamespaceHTMLOnly), an element in a non-null namespace is not an
	// HTML element and is serialized with an explicit end tag instead.
	isVoid := info != nil && info.empty
	if d.nullNamespaceHTMLOnly && e.URI() != "" {
		isVoid = false
	}
	if isVoid {
		d.writeString(out, ">")
		if d.format && shouldNewlineAfterVoid(e, info) {
			d.writeString(out, "\n")
		}
		return d.err
	}

	d.writeString(out, ">")

	if d.format && shouldNewlineAfterOpen(e, info) {
		d.writeString(out, "\n")
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
		d.writeString(out, "\n")
	}

	// Closing tag
	d.writeString(out, "</")
	d.writeString(out, name)
	d.writeString(out, ">")

	if d.format && shouldNewlineAfterClose(e, info) {
		d.writeString(out, "\n")
	}

	return d.err
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
		// Reject names that would inject markup into the tag.
		d.checkName("attribute", attrName)
		if d.err != nil {
			return d.err
		}
		d.writeString(out, " ")
		d.writeString(out, attrName)

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
		uriEscaped := isURI && !d.noEscapeURIAttributes
		if uriEscaped {
			val = uriEscapeStr(val)
		}

		d.writeString(out, "=\"")
		if d.err == nil {
			// Serialization 3.1 §7 (character-expansion phase): when URI escaping
			// is applied to a URI attribute value (escape-uri-attributes=yes), the
			// serializer skips character mapping for that value. Character maps
			// apply to non-URI attributes, and to URI attributes only when URI
			// escaping is disabled (escape-uri-attributes=no). Unicode normalization
			// is likewise scoped to the attribute value's character content; a
			// URI-escaped value is already ASCII (percent-encoded), so normalization
			// is a no-op there and character mapping is skipped.
			switch {
			case uriEscaped:
				d.check(htmlEscapeAttrValue(out, val, isURI, d.preserveCase, nil))
			case d.normalize:
				d.writeNormalizedAttrValue(out, val, isURI)
			default:
				d.check(htmlEscapeAttrValue(out, val, isURI, d.preserveCase, d.charMap))
			}
		}
		d.writeString(out, "\"")
	}
	return d.err
}

// writeNSDecl writes a single namespace declaration attribute.
func (d *htmlDumper) writeNSDecl(out io.Writer, prefix, uri string) {
	// The "xml" prefix is predefined by the Namespaces in XML spec (bound
	// implicitly to the XML namespace everywhere), so a literal
	// xmlns:xml="http://www.w3.org/XML/1998/namespace" declaration is
	// redundant and non-canonical and must never be emitted.
	if prefix == "xml" {
		return
	}
	if prefix == "" {
		d.writeString(out, " xmlns=\"")
		d.writeString(out, uri)
		d.writeString(out, "\"")
		return
	}
	// A non-empty prefix is written verbatim into the xmlns: attribute name,
	// so an unsafe prefix could break out of the tag (e.g. a prefix
	// containing a space and quote yields a separate injected attribute).
	// Reject it the same way element/attribute names are rejected.
	d.checkName("namespace prefix", prefix)
	if d.err != nil {
		return
	}
	d.writeString(out, " xmlns:")
	d.writeString(out, prefix)
	d.writeString(out, "=\"")
	d.writeString(out, uri)
	d.writeString(out, "\"")
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
	maps.Copy(m, src)
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
			for j := range width {
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
	out, align := utf8ToLatin1WithAlign(p, lw.strict)
	n, err := lw.w.Write(out)
	// A well-behaved [io.Writer] reports err != nil when n < len(out).
	// Defensively promote a silent short write to io.ErrShortWrite so a
	// non-conformant inner writer cannot mask data loss as success.
	if err == nil && n < len(out) {
		err = io.ErrShortWrite
	}
	if err != nil {
		// align is nil on the ASCII fast path (1:1 mapping); otherwise it
		// records (inputBytes, outputBytes) at every rune boundary. Return
		// the largest input prefix whose encoded output fully fit within
		// the inner writer's reported n bytes, so callers see an honest
		// consumed count for io.Copy-style bookkeeping.
		if align == nil {
			return n, err
		}
		return inputConsumedAt(align, n), err
	}
	return len(p), nil
}

// alignPoint records that consuming inputBytes of the source produced
// outputBytes of encoded output at a clean rune boundary.
type alignPoint struct{ inputBytes, outputBytes int }

// inputConsumedAt returns the largest inputBytes from align where
// outputBytes <= n. align must be monotonically increasing in both fields.
func inputConsumedAt(align []alignPoint, n int) int {
	consumed := 0
	for _, ap := range align {
		if ap.outputBytes > n {
			break
		}
		consumed = ap.inputBytes
	}
	return consumed
}

// utf8ToLatin1WithAlign converts UTF-8 encoded data back to Latin-1 /
// Windows-1252 and additionally returns an alignment table recording
// (inputBytes, outputBytes) at every rune boundary, so callers can map a
// partial-output byte count back to a precise consumed-input prefix.
//
// Runes U+0080-U+00FF are written as single bytes.
// When strict is true (explicit ISO-8859-1), runes > U+00FF are emitted
// as numeric character references (&#N;).
// When strict is false (auto-detected Win-1252), Windows-1252 runes are
// reverse-mapped to single bytes and other runes pass through as UTF-8.
//
// On the all-ASCII fast path the returned slice is the input verbatim and
// align is nil — input and output are 1:1 so no table is needed.
func utf8ToLatin1WithAlign(data []byte, strict bool) ([]byte, []alignPoint) {
	allASCII := true
	for _, b := range data {
		if b >= 0x80 {
			allASCII = false
			break
		}
	}
	if allASCII {
		return data, nil
	}

	var buf bytes.Buffer
	buf.Grow(len(data))
	align := make([]alignPoint, 0, len(data)/2+1)
	align = append(align, alignPoint{0, 0})
	for i := 0; i < len(data); {
		b := data[i]
		if b < 0x80 {
			buf.WriteByte(b)
			i++
			align = append(align, alignPoint{i, buf.Len()})
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
		align = append(align, alignPoint{i, buf.Len()})
	}
	return buf.Bytes(), align
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
