package xslt3

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/lestrrat-go/helium"
	htmlpkg "github.com/lestrrat-go/helium/html"
	"github.com/lestrrat-go/helium/internal/lexicon"
)

func serializeHTML(w io.Writer, doc *helium.Document, outDef *OutputDef) error {
	// Determine DOCTYPE handling.
	hasDoctypeAttrs := outDef.DoctypePublic != "" || outDef.DoctypeSystem != ""
	// Use explicit HTMLVersion for DOCTYPE/structural decisions.
	isHTML5 := isHTMLVersion5(outDef.HTMLVersion)
	// Use effective version (with XSLT 3.0 default=5) for character escaping.
	escapeCtrl := effectiveHTMLVersion5(outDef)

	if hasDoctypeAttrs {
		rootName := "html" //nolint:goconst
		if root := doc.DocumentElement(); root != nil {
			rootName = root.Name()
		}
		_, _ = io.WriteString(w, "<!DOCTYPE ")
		_, _ = io.WriteString(w, rootName)
		if outDef.DoctypePublic != "" {
			_, _ = io.WriteString(w, " PUBLIC \"")
			_, _ = io.WriteString(w, outDef.DoctypePublic)
			_, _ = io.WriteString(w, "\"")
			if outDef.DoctypeSystem != "" {
				_, _ = io.WriteString(w, " \"")
				_, _ = io.WriteString(w, outDef.DoctypeSystem)
				_, _ = io.WriteString(w, "\"")
			}
		} else if outDef.DoctypeSystem != "" {
			_, _ = io.WriteString(w, " SYSTEM \"")
			_, _ = io.WriteString(w, outDef.DoctypeSystem)
			_, _ = io.WriteString(w, "\"")
		}
		_, _ = io.WriteString(w, ">\n")
	}

	// Insert <meta http-equiv="Content-Type"> in <head> if not already present.
	if outDef.IncludeContentType == nil || *outDef.IncludeContentType {
		insertHTMLMeta(doc, outDef)
	}

	// For HTML5: normalize SVG and MathML namespaces so that elements in
	// those namespaces use the default namespace (unprefixed) per the
	// HTML5 serialization spec.
	if isHTML5 {
		normalizeForeignNamespaces(doc)
	}

	// For HTML5 without explicit doctype attrs, we need to serialize
	// children manually to insert <!DOCTYPE html> before the first element.
	noEscapeURI := outDef.EscapeURIAttributes != nil && !*outDef.EscapeURIAttributes
	if isHTML5 && !hasDoctypeAttrs {
		hw := htmlpkg.NewWriter().Format(false).PreserveCase(true)
		if escapeCtrl {
			hw = hw.EscapeControlChars(true)
		}
		if noEscapeURI {
			hw = hw.EscapeURIAttributes(false)
		}
		doctypeEmitted := false
		for child := range helium.Children(doc) {
			if child.Type() == helium.DTDNode {
				continue
			}
			if child.Type() == helium.ElementNode && !doctypeEmitted {
				_, _ = io.WriteString(w, "<!DOCTYPE html>")
				doctypeEmitted = true
			}
			if err := hw.WriteTo(w, child); err != nil {
				return err
			}
		}
		return nil
	}

	hw := htmlpkg.NewWriter().DefaultDTD(false).Format(false).PreserveCase(true)
	if noEscapeURI {
		hw = hw.EscapeURIAttributes(false)
	}
	if escapeCtrl {
		hw = hw.EscapeControlChars(true)
	}
	return hw.WriteTo(w, doc)
}

// serializeXHTML serializes using the XHTML output method.
// XHTML is essentially XML with HTML-specific additions:
// - meta charset tag in <head>
// - For HTML5, simplified DOCTYPE
// - Self-closing void elements with a space before />
func serializeXHTML(w io.Writer, doc *helium.Document, outDef *OutputDef, charMap map[rune]string) error {
	isHTML5 := isHTMLVersion5(outDef.HTMLVersion)

	// XSLT 3.0 §20: for xhtml method with html-version >= 5, the default
	// for omit-xml-declaration is "yes" (unless explicitly set otherwise).
	if isHTML5 && !outDef.OmitDeclarationExplicit {
		outDef.OmitDeclaration = true
	}

	// Per XSLT spec, doctype-public without doctype-system is ignored for xhtml.
	if outDef.DoctypeSystem == "" && outDef.DoctypePublic != "" {
		outDef.DoctypePublic = ""
	}

	// For HTML5: only use explicit doctype when doctype-system is specified.
	// Without doctype-system (even if doctype-public is set), use <!DOCTYPE html>.
	// Only emit the HTML5 DOCTYPE when the root element is "html" in the XHTML namespace.
	if isHTML5 && outDef.DoctypeSystem == "" {
		root := doc.DocumentElement()
		if root != nil && strings.EqualFold(root.LocalName(), "html") &&
			root.URI() == lexicon.NamespaceXHTML {
			dtdName := root.LocalName()
			if dtd := doc.IntSubset(); dtd != nil {
				helium.UnlinkNode(dtd)
			}
			_, _ = doc.CreateInternalSubset(dtdName, "", "")
		}
	}

	// Insert <meta http-equiv="Content-Type"> in <head>, but only when the
	// root element is in the XHTML namespace (non-XHTML documents should not
	// get an injected meta tag).
	if outDef.IncludeContentType == nil || *outDef.IncludeContentType {
		root := doc.DocumentElement()
		if root != nil && root.URI() == lexicon.NamespaceXHTML {
			insertHTMLMeta(doc, outDef)
		}
	}

	// Normalize XHTML namespace: elements in http://www.w3.org/1999/xhtml
	// that use a prefix should be converted to use the default namespace.
	normalizeXHTMLNamespace(doc)

	// For HTML5: normalize SVG and MathML namespaces so that elements in
	// those namespaces use the default namespace (unprefixed) per the
	// HTML5 serialization spec.
	if isHTML5 {
		normalizeForeignNamespaces(doc)
	}

	// Serialize as XML first.
	var buf bytes.Buffer
	if err := serializeXML(&buf, doc, outDef, charMap); err != nil {
		return err
	}
	result := buf.String()

	// Post-process for XHTML rules:
	// 0. Replace &quot; with &#34; — XHTML uses numeric character references
	//    for double quotes in attribute values, not the named entity.
	result = strings.ReplaceAll(result, "&quot;", "&#34;")
	// 1. URI attribute escaping (percent-encode non-ASCII in href, src, etc.)
	escapeURI := outDef.EscapeURIAttributes == nil || *outDef.EscapeURIAttributes
	if escapeURI {
		result = escapeXHTMLURIAttrsInString(result)
	}
	// 2. C1 control character escaping (U+0080-U+009F as &#NNN;)
	result = escapeC1ControlsInString(result)
	// 3. Void elements: add space before /> (e.g., <br /> not <br/>)
	// 4. Non-void elements: expand self-closing to open+close
	result = fixXHTMLSelfClosing(result)

	_, err := io.WriteString(w, result)
	return err
}

// normalizeXHTMLNamespace walks the document and converts prefixed XHTML
// namespace elements to use the default namespace (unprefixed), as required
// by the XHTML output method. The default namespace declaration is added
// only to the root element; descendants inherit it.
func normalizeXHTMLNamespace(doc *helium.Document) {
	// First pass: find all XHTML-prefixed elements and track prefixes to remove.
	// Also find/create a shared default namespace node for XHTML.
	var sharedNS *helium.Namespace
	rootDone := false

	_ = helium.Walk(doc, helium.NodeWalkerFunc(func(n helium.Node) error {
		elem, ok := n.(*helium.Element)
		if !ok {
			return nil
		}
		if elem.URI() != lexicon.NamespaceXHTML {
			return nil
		}
		if elem.Prefix() == "" {
			// Already using default namespace. Capture the NS node if we
			// haven't seen one yet.
			if sharedNS == nil {
				for _, ns := range elem.Namespaces() {
					if ns.Prefix() == "" && ns.URI() == lexicon.NamespaceXHTML {
						sharedNS = ns
						break
					}
				}
			}
			return nil
		}

		oldPrefix := elem.Prefix()
		// Remove the prefixed namespace declaration from this element
		elem.RemoveNamespaceByPrefix(oldPrefix)

		if !rootDone {
			// First prefixed element: declare default XHTML namespace here
			_ = elem.DeclareNamespace("", lexicon.NamespaceXHTML)
			// Find the namespace node we just created
			for _, ns := range elem.Namespaces() {
				if ns.Prefix() == "" && ns.URI() == lexicon.NamespaceXHTML {
					sharedNS = ns
					break
				}
			}
			rootDone = true
		}

		// Set the element's namespace to the shared default NS node
		if sharedNS != nil {
			elem.SetNs(sharedNS)
		}

		return nil
	}))
}

// normalizeForeignNamespaces converts prefixed SVG and MathML elements to use
// their default namespace (unprefixed) for HTML5 XHTML output. Each element
// in the SVG or MathML namespace gets a default xmlns declaration for its own
// namespace, so it serializes as e.g. <svg xmlns="..."> instead of <s:svg>.
func normalizeForeignNamespaces(doc *helium.Document) {
	_ = helium.Walk(doc, helium.NodeWalkerFunc(func(n helium.Node) error {
		elem, ok := n.(*helium.Element)
		if !ok {
			return nil
		}
		uri := elem.URI()
		if uri != lexicon.NamespaceSVG && uri != lexicon.NamespaceMathML {
			return nil
		}
		if elem.Prefix() == "" {
			return nil // already unprefixed
		}
		oldPrefix := elem.Prefix()
		// Remove the old prefixed namespace declaration
		elem.RemoveNamespaceByPrefix(oldPrefix)
		// Declare the element's namespace as the default on this element
		_ = elem.DeclareNamespace("", uri)
		// Find the ns node we just created and set it as the element's ns
		for _, ns := range elem.Namespaces() {
			if ns.Prefix() == "" && ns.URI() == uri {
				elem.SetNs(ns)
				break
			}
		}
		return nil
	}))
}

// xhtmlVoidElements lists HTML void elements that should be self-closed
// with a space before /> in XHTML output.
var xhtmlVoidElements = map[string]struct{}{
	"area": {}, "base": {}, "br": {}, "col": {}, "embed": {},
	"hr": {}, "img": {}, "input": {}, "link": {}, "meta": {},
	"param": {}, "source": {}, "track": {}, "wbr": {},
	// XHTML 1.x additional void elements
	"basefont": {}, "frame": {}, "isindex": {},
}

// fixXHTMLSelfClosing post-processes XML output for XHTML serialization rules:
// - Void elements get space before />: <br/> -> <br />
// - Non-void elements are expanded: <Option.../> -> <Option...></Option>
func fixXHTMLSelfClosing(xml string) string {
	var out strings.Builder
	out.Grow(len(xml))
	i := 0
	for i < len(xml) {
		if xml[i] != '<' {
			out.WriteByte(xml[i])
			i++
			continue
		}
		// Find end of tag
		tagEnd := strings.IndexByte(xml[i:], '>')
		if tagEnd < 0 {
			out.WriteString(xml[i:])
			break
		}
		tag := xml[i : i+tagEnd+1]
		if strings.HasSuffix(tag, "/>") && !strings.HasPrefix(tag, "<?") {
			// Self-closing element. Extract element name.
			nameStart := 1 // skip '<'
			nameEnd := nameStart
			for nameEnd < len(tag) && tag[nameEnd] != ' ' && tag[nameEnd] != '/' && tag[nameEnd] != '>' && tag[nameEnd] != '\t' && tag[nameEnd] != '\n' {
				nameEnd++
			}
			elemName := tag[nameStart:nameEnd]
			// Check for namespace prefix — use local name
			localName := elemName
			if idx := strings.IndexByte(elemName, ':'); idx >= 0 {
				localName = elemName[idx+1:]
			}
			if _, isVoid := xhtmlVoidElements[strings.ToLower(localName)]; isVoid {
				// Void element: add space before />
				out.WriteString(tag[:len(tag)-2])
				out.WriteString(" />")
			} else {
				// Non-void element: expand to open+close tags
				out.WriteString(tag[:len(tag)-2])
				out.WriteString("></")
				out.WriteString(elemName)
				out.WriteString(">")
			}
		} else {
			out.WriteString(tag)
		}
		i += tagEnd + 1
	}
	return out.String()
}

// escapeXHTMLURIAttrsInString post-processes serialized XML to percent-encode
// non-ASCII characters in URI attribute values (href, src, action, etc.)
func escapeXHTMLURIAttrsInString(xml string) string {
	var out strings.Builder
	out.Grow(len(xml))
	i := 0
	for i < len(xml) {
		if xml[i] != '<' {
			out.WriteByte(xml[i])
			i++
			continue
		}
		// Find end of tag
		tagEnd := strings.IndexByte(xml[i:], '>')
		if tagEnd < 0 {
			out.WriteString(xml[i:])
			break
		}
		tag := xml[i : i+tagEnd+1]
		out.WriteString(escapeURIAttrsInTag(tag))
		i += tagEnd + 1
	}
	return out.String()
}

// escapeURIAttrsInTag finds URI attributes in a tag and percent-encodes
// non-ASCII characters in their values.
func escapeURIAttrsInTag(tag string) string {
	if len(tag) < 2 || tag[1] == '/' || tag[1] == '?' || tag[1] == '!' {
		return tag
	}
	var out strings.Builder
	out.Grow(len(tag))
	i := 0
	for i < len(tag) {
		// Find attribute name
		if tag[i] == '=' && i > 0 {
			// Look back for attribute name
			nameEnd := i
			nameStart := nameEnd - 1
			for nameStart > 0 && tag[nameStart] != ' ' && tag[nameStart] != '\t' && tag[nameStart] != '\n' {
				nameStart--
			}
			if tag[nameStart] == ' ' || tag[nameStart] == '\t' || tag[nameStart] == '\n' {
				nameStart++
			}
			attrName := strings.ToLower(tag[nameStart:nameEnd])
			_, isURI := htmlURIAttrs[attrName]
			out.WriteByte('=')
			i++
			if i < len(tag) && (tag[i] == '"' || tag[i] == '\'') {
				quote := tag[i]
				out.WriteByte(quote)
				i++
				valStart := i
				for i < len(tag) && tag[i] != quote {
					i++
				}
				val := tag[valStart:i]
				if isURI {
					out.WriteString(percentEncodeNonASCII(val))
				} else {
					out.WriteString(val)
				}
				if i < len(tag) {
					out.WriteByte(quote)
					i++
				}
			}
		} else {
			out.WriteByte(tag[i])
			i++
		}
	}
	return out.String()
}

// percentEncodeNonASCII percent-encodes non-ASCII bytes in a string.
func percentEncodeNonASCII(s string) string {
	var buf strings.Builder
	b := []byte(s)
	for i := 0; i < len(b); i++ {
		c := b[i]
		if c > 0x7E {
			fmt.Fprintf(&buf, "%%%02X", c)
		} else {
			buf.WriteByte(c)
		}
	}
	return buf.String()
}

// escapeC1ControlsInString replaces C1 control characters (U+0080-U+009F)
// in the serialized string with numeric character references.
func escapeC1ControlsInString(s string) string {
	var buf strings.Builder
	buf.Grow(len(s))
	changed := false
	for _, r := range s {
		if r >= 0x80 && r <= 0x9F {
			fmt.Fprintf(&buf, "&#%d;", r)
			changed = true
		} else {
			buf.WriteRune(r)
		}
	}
	if !changed {
		return s
	}
	return buf.String()
}

// htmlURIAttrs lists HTML attributes whose values are URIs and should not
// have character maps applied (they use URI-escaping instead).
var htmlURIAttrs = map[string]struct{}{
	"href": {}, "src": {}, "action": {}, "cite": {}, "data": {},
	"formaction": {}, "poster": {}, "codebase": {}, "longdesc": {},
	"usemap": {}, "background": {}, "profile": {},
}

// insertHTMLMeta inserts a <meta http-equiv="Content-Type"> element as the
// first child of the <head> element if not already present.
func insertHTMLMeta(doc *helium.Document, outDef *OutputDef) {
	root := doc.DocumentElement()
	if root == nil {
		return
	}
	// Find the <head> element (case-insensitive, using local name for namespace support).
	var head *helium.Element
	for child := range helium.Children(root) {
		if e, ok := child.(*helium.Element); ok && strings.EqualFold(e.LocalName(), "head") {
			head = e
			break
		}
	}
	if head == nil {
		return
	}
	enc := outDef.Encoding
	if enc == "" {
		enc = "UTF-8"
	}
	mediaType := outDef.MediaType
	if mediaType == "" {
		mediaType = "text/html"
	}
	contentValue := mediaType + "; charset=" + enc

	// Check if a <meta http-equiv="Content-Type"> already exists.
	// If so, update its content attribute to match the output encoding.
	for child := range helium.Children(head) {
		if e, ok := child.(*helium.Element); ok && strings.EqualFold(e.LocalName(), "meta") {
			for _, attr := range e.Attributes() {
				if strings.EqualFold(attr.Name(), "http-equiv") && strings.EqualFold(attr.Value(), "Content-Type") {
					// Update the existing content attribute
					_ = e.SetLiteralAttribute("content", contentValue)
					return
				}
			}
		}
	}
	// Create and insert the meta element.
	meta := doc.CreateElement("meta")
	// If the head element is in a namespace, put the meta element in the same namespace.
	if headURI := head.URI(); headURI != "" {
		_ = meta.SetActiveNamespace(head.Prefix(), headURI)
	}
	_ = meta.SetLiteralAttribute("http-equiv", "Content-Type")
	_ = meta.SetLiteralAttribute("content", contentValue)
	// Insert meta as first child of <head>.
	// Unlink existing children, add meta, then re-add them.
	var children []helium.Node
	for child := head.FirstChild(); child != nil; {
		next := child.NextSibling()
		helium.UnlinkNode(child.(helium.MutableNode)) //nolint:forcetypeassert
		children = append(children, child)
		child = next
	}
	_ = head.AddChild(meta)
	for _, child := range children {
		_ = head.AddChild(child)
	}
}
