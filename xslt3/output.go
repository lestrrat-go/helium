package xslt3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/lestrrat-go/helium"
	htmlpkg "github.com/lestrrat-go/helium/html"
	"github.com/lestrrat-go/helium/stream"
	"github.com/lestrrat-go/helium/xpath3"
	"golang.org/x/text/encoding/htmlindex"
	"golang.org/x/text/unicode/norm"
)

// outputFrame represents the current output target during transformation.
type outputFrame struct {
	doc                 *helium.Document // result document being built
	current             helium.Node      // current insertion point
	captureItems        bool             // when true, xsl:sequence adds to pendingItems instead of DOM
	separateTextNodes   bool             // when true, text nodes are captured as separate string items (prevents DOM merging)
	sequenceMode        bool             // when true, all nodes (text, element, attr, comment, PI) are captured as separate items
	mapConstructor      bool             // when true, xsl:map-entry emits single-entry maps into pendingItems
	pendingItems        xpath3.Sequence  // captured items from xsl:sequence
	prevWasAtomic       bool             // true when last xsl:sequence output was an atomic value (for inter-call space separation)
	wherePopulated      bool             // when true, xsl:document emits document node (not children) so xsl:where-populated can check emptiness
	itemSeparator       *string          // item-separator serialization parameter; nil means default (" " between adjacent atomics)
	outputSerial        int              // monotonically increases whenever visible output is produced
	conditionalScopes   []conditionalScope
}

type conditionalKind int

const (
	conditionalOnEmpty conditionalKind = iota + 1
	conditionalOnNonEmpty
)

type conditionalAction struct {
	ctx            context.Context
	kind           conditionalKind
	content        xpath3.Sequence
	placeholder    helium.Node
	prevWasAtomic  bool // whether the output preceding this action was an atomic value
}

type conditionalScope struct {
	hasOutput bool
	actions   []conditionalAction
}

func (out *outputFrame) noteOutput() {
	out.outputSerial++
}

// SerializeResult writes the result document to a writer according to the
// output definition. If outDef is nil, defaults to XML output.
func SerializeResult(w io.Writer, doc *helium.Document, outDef *OutputDef) error {
	var charMap map[rune]string
	if outDef != nil {
		charMap = outDef.ResolvedCharMap
	}
	return serializeResult(w, doc, outDef, charMap)
}

func serializeResult(w io.Writer, doc *helium.Document, outDef *OutputDef, charMaps ...map[rune]string) error {
	if outDef == nil {
		outDef = defaultOutputDef()
	}

	// XSLT 3.0 §20: When no output method is explicitly specified, auto-detect
	// based on the document element. If the root element is "html" (case-insensitive)
	// in no namespace, default to HTML output method.
	if !outDef.MethodExplicit && outDef.Method == "xml" {
		if root := doc.DocumentElement(); root != nil {
			if strings.EqualFold(root.Name(), "html") && root.URI() == "" {
				outDef.Method = "html"
				outDef.OmitDeclaration = true
			}
		}
	}

	var charMap map[rune]string
	if len(charMaps) > 0 {
		charMap = charMaps[0]
	}

	// Check if we need encoding conversion (non-UTF-8/UTF-16)
	enc := strings.ToLower(outDef.Encoding)
	needsEncodingConversion := enc != "" && enc != "utf-8" && enc != "utf8" && enc != "utf-16" && enc != "utf16"

	// Check if we need Unicode normalization
	needsNormalization := outDef.NormalizationForm != "" && outDef.NormalizationForm != "NONE"

	// Buffer when post-processing is needed
	needsBuffer := needsEncodingConversion || needsNormalization
	var target io.Writer
	var buf bytes.Buffer
	if needsBuffer {
		target = &buf
	} else {
		target = w
	}

	var err error
	switch outDef.Method {
	case "text":
		err = serializeText(target, doc, charMap)
	case "html":
		if len(charMap) > 0 {
			// For HTML with character maps, serialize to buffer, then apply
			// character map to text content only (not inside tags).
			var htmlBuf bytes.Buffer
			if herr := serializeHTML(&htmlBuf, doc, outDef); herr != nil {
				err = herr
			} else {
				_, err = io.WriteString(target, applyCharMapToHTMLText(htmlBuf.String(), charMap))
			}
		} else {
			err = serializeHTML(target, doc, outDef)
		}
	case "xhtml":
		err = serializeXHTML(target, doc, outDef, charMap)
	default:
		err = serializeXML(target, doc, outDef, charMap)
	}
	if err != nil {
		return err
	}

	if needsBuffer {
		data := buf.Bytes()

		// Apply Unicode normalization if requested
		if needsNormalization {
			data = applyUnicodeNormalization(data, outDef.NormalizationForm)
		}

		if needsEncodingConversion {
			return transcodeToEncoding(w, data, enc)
		}
		_, err = w.Write(data)
		return err
	}
	return nil
}

// applyUnicodeNormalization applies the specified Unicode normalization form
// to the given UTF-8 data.
func applyUnicodeNormalization(data []byte, form string) []byte {
	var nf norm.Form
	switch form {
	case "NFC":
		nf = norm.NFC
	case "NFD":
		nf = norm.NFD
	case "NFKC":
		nf = norm.NFKC
	case "NFKD":
		nf = norm.NFKD
	case "FULLY-NORMALIZED":
		// Fully-normalized is NFC plus additional constraints.
		// For practical purposes, NFC is sufficient.
		nf = norm.NFC
	default:
		return data
	}
	return nf.Bytes(data)
}

// transcodeToEncoding converts UTF-8 bytes to the target encoding,
// replacing characters that cannot be represented with XML character references.
func transcodeToEncoding(w io.Writer, utf8Data []byte, encName string) error {
	codec, err := htmlindex.Get(encName)
	if err != nil {
		// Unknown encoding — fall back to writing UTF-8
		_, werr := w.Write(utf8Data)
		return werr
	}

	encoder := codec.NewEncoder()

	// Process character by character: try to encode each rune,
	// and if it fails, output a character reference instead.
	for len(utf8Data) > 0 {
		r, size := utf8.DecodeRune(utf8Data)
		if r == utf8.RuneError && size <= 1 {
			utf8Data = utf8Data[1:]
			continue
		}

		s := string(utf8Data[:size])
		encoded, err := encoder.Bytes([]byte(s))
		if err != nil {
			// Character cannot be encoded — use character reference
			ref := fmt.Sprintf("&#x%X;", r)
			if _, werr := io.WriteString(w, ref); werr != nil {
				return werr
			}
			// Reset encoder state after error
			encoder = codec.NewEncoder()
		} else {
			if _, werr := w.Write(encoded); werr != nil {
				return werr
			}
		}
		utf8Data = utf8Data[size:]
	}
	return nil
}

func defaultOutputDef() *OutputDef {
	return &OutputDef{
		Method:   "xml",
		Encoding: "UTF-8",
		Version:  "1.0",
	}
}

func serializeXML(w io.Writer, doc *helium.Document, outDef *OutputDef, charMap map[rune]string) error {
	// For non-UTF-8 encodings, use the stream-based serializer which
	// always outputs UTF-8. The encoding conversion is handled by
	// serializeResult's transcoding layer.
	targetEnc := strings.ToLower(outDef.Encoding)
	isNonUTF8 := targetEnc != "" && targetEnc != "utf-8" && targetEnc != "utf8" && targetEnc != "utf-16" && targetEnc != "utf16"
	if len(charMap) > 0 || hasDOEMarkers(doc) || isNonUTF8 {
		return serializeXMLWithCharMap(w, doc, outDef, charMap)
	}
	// Set encoding on the document so the XML declaration includes it.
	if outDef.Encoding != "" && doc.Encoding() == "utf8" {
		doc.SetEncoding(outDef.Encoding)
	}
	// Add DOCTYPE if doctype-public or doctype-system is specified and
	// the document doesn't already have a DTD.
	if (outDef.DoctypePublic != "" || outDef.DoctypeSystem != "") && doc.IntSubset() == nil {
		rootName := "html" // default
		if root := doc.DocumentElement(); root != nil {
			rootName = root.Name()
		}
		if _, err := doc.CreateInternalSubset(rootName, outDef.DoctypePublic, outDef.DoctypeSystem); err != nil {
			return err
		}
	}
	opts := []helium.WriteOption{
		helium.WithNoEscapeNonASCII(),
	}
	if outDef.Indent {
		opts = append(opts, helium.WithFormat())
	}
	if outDef.OmitDeclaration {
		opts = append(opts, helium.WithNoDecl())
	}
	// When standalone is "yes" or "no", or when indent="no" and
	// the declaration is not omitted, buffer and post-process.
	needStandalone := !outDef.OmitDeclaration && (outDef.Standalone == "yes" || outDef.Standalone == "no")
	needStripNewline := !outDef.Indent && !outDef.OmitDeclaration
	if needStandalone || needStripNewline {
		var buf strings.Builder
		if err := doc.XML(&buf, opts...); err != nil {
			return err
		}
		out := buf.String()
		if needStandalone {
			out = injectStandalone(out, outDef.Standalone)
		}
		if needStripNewline {
			if idx := strings.Index(out, "?>\n"); idx >= 0 {
				out = out[:idx+2] + out[idx+3:]
			}
		}
		_, err := io.WriteString(w, out)
		return err
	}
	return doc.XML(w, opts...)
}

// injectStandalone inserts standalone="yes" or standalone="no" into the
// XML declaration before the closing "?>".
func injectStandalone(xml, value string) string {
	const declEnd = "?>"
	idx := strings.Index(xml, declEnd)
	if idx < 0 {
		return xml
	}
	return xml[:idx] + " standalone=\"" + value + "\"" + xml[idx:]
}

// serializeXMLWithCharMap serializes an XML document applying character map
// substitutions. Replacement strings are written raw (not escaped).
func serializeXMLWithCharMap(w io.Writer, doc *helium.Document, outDef *OutputDef, charMap map[rune]string) error {
	// Buffer and post-process when standalone or indent="no".
	needStandalone := !outDef.OmitDeclaration && (outDef.Standalone == "yes" || outDef.Standalone == "no")
	needStripNewline := !outDef.Indent && !outDef.OmitDeclaration
	if needStandalone || needStripNewline {
		var buf strings.Builder
		if err := serializeXMLWithCharMapInner(&buf, doc, outDef, charMap); err != nil {
			return err
		}
		out := buf.String()
		if needStandalone {
			out = injectStandalone(out, outDef.Standalone)
		}
		if needStripNewline {
			if idx := strings.Index(out, "?>\n"); idx >= 0 {
				out = out[:idx+2] + out[idx+3:]
			}
		}
		_, err := io.WriteString(w, out)
		return err
	}
	return serializeXMLWithCharMapInner(w, doc, outDef, charMap)
}

func serializeXMLWithCharMapInner(w io.Writer, doc *helium.Document, outDef *OutputDef, charMap map[rune]string) error {
	sw := stream.NewWriter(w)

	if !outDef.OmitDeclaration {
		enc := outDef.Encoding
		if enc == "" {
			enc = "UTF-8"
		}
		if err := sw.StartDocument("1.0", enc, ""); err != nil {
			return err
		}
	}

	// Add DOCTYPE if doctype-public or doctype-system is specified.
	if outDef.DoctypePublic != "" || outDef.DoctypeSystem != "" {
		rootName := "html"
		if root := doc.DocumentElement(); root != nil {
			rootName = root.Name()
		}
		if err := sw.WriteDTD(rootName, outDef.DoctypePublic, outDef.DoctypeSystem, ""); err != nil {
			return err
		}
	}

	err := serializeXMLNodeWithCharMap(sw, doc, charMap)
	if err != nil {
		return err
	}
	return sw.Flush()
}

func serializeXMLNodeWithCharMap(sw *stream.Writer, n helium.Node, charMap map[rune]string) error {
	doeActive := false
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		// Handle DOE marker PIs
		if child.Type() == helium.ProcessingInstructionNode {
			piName := string(child.Name())
			if piName == "disable-output-escaping" {
				doeActive = true
				continue
			}
			if piName == "enable-output-escaping" {
				doeActive = false
				continue
			}
		}
		switch child.Type() {
		case helium.ElementNode:
			elem := child.(*helium.Element)
			prefix := string(elem.Prefix())
			local := string(elem.LocalName())
			uri := string(elem.URI())
			if err := sw.StartElementNS(prefix, local, uri); err != nil {
				return err
			}
			// Write additional namespace declarations not handled by StartElementNS
			elemPrefix := string(elem.Prefix())
			for _, ns := range elem.Namespaces() {
				if ns.Prefix() == elemPrefix {
					continue // already declared by StartElementNS
				}
				if ns.Prefix() == "" {
					if err := sw.WriteAttribute("xmlns", ns.URI()); err != nil {
						return err
					}
				} else {
					if err := sw.WriteAttribute("xmlns:"+ns.Prefix(), ns.URI()); err != nil {
						return err
					}
				}
			}
			// Write attributes
			for _, attr := range elem.Attributes() {
				attrVal := applyCharacterMap(attr.Value(), charMap)
				if err := sw.WriteAttribute(attr.Name(), attrVal); err != nil {
					return err
				}
			}
			// Recurse into children
			if err := serializeXMLNodeWithCharMap(sw, elem, charMap); err != nil {
				return err
			}
			if err := sw.EndElement(); err != nil {
				return err
			}
		case helium.TextNode, helium.CDATASectionNode:
			text := string(child.Content())
			if doeActive {
				if err := sw.WriteRaw(text); err != nil {
					return err
				}
			} else if err := writeTextWithCharMap(sw, text, charMap); err != nil {
				return err
			}
		case helium.CommentNode:
			if err := sw.WriteComment(string(child.Content())); err != nil {
				return err
			}
		case helium.ProcessingInstructionNode:
			if err := sw.WritePI(string(child.Name()), string(child.Content())); err != nil {
				return err
			}
		}
	}
	return nil
}

// writeTextWithCharMap writes text content, applying character map substitutions.
// Mapped characters are written raw (unescaped), unmapped characters are written
// as normal text (with XML escaping).
func writeTextWithCharMap(sw *stream.Writer, text string, charMap map[rune]string) error {
	var unmapped strings.Builder
	for _, r := range text {
		if repl, ok := charMap[r]; ok {
			// Flush any accumulated unmapped text first
			if unmapped.Len() > 0 {
				if err := sw.WriteString(unmapped.String()); err != nil {
					return err
				}
				unmapped.Reset()
			}
			// Write the replacement raw
			if err := sw.WriteRaw(repl); err != nil {
				return err
			}
		} else {
			unmapped.WriteRune(r)
		}
	}
	if unmapped.Len() > 0 {
		return sw.WriteString(unmapped.String())
	}
	return nil
}

func serializeText(w io.Writer, doc *helium.Document, charMap map[rune]string) error {
	// Text output: just write the text content of the document
	sw := stream.NewWriter(w)
	err := helium.Walk(doc, func(n helium.Node) error {
		switch n.Type() {
		case helium.TextNode, helium.CDATASectionNode:
			text := string(n.Content())
			if len(charMap) > 0 {
				text = applyCharacterMap(text, charMap)
			}
			return sw.WriteRaw(text)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return sw.Flush()
}

func serializeHTML(w io.Writer, doc *helium.Document, outDef *OutputDef) error {
	// Determine DOCTYPE handling.
	hasDoctypeAttrs := outDef.DoctypePublic != "" || outDef.DoctypeSystem != ""
	isHTML5 := outDef.HTMLVersion == "5" || outDef.HTMLVersion == "5.0"

	if hasDoctypeAttrs {
		rootName := "html"
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

	// For HTML5 without explicit doctype attrs, we need to serialize
	// children manually to insert <!DOCTYPE html> before the first element.
	if isHTML5 && !hasDoctypeAttrs {
		nodeOpts := []htmlpkg.WriteOption{
			htmlpkg.WithNoFormat(),
			htmlpkg.WithPreserveCase(),
		}
		doctypeEmitted := false
		for child := doc.FirstChild(); child != nil; child = child.NextSibling() {
			if child.Type() == helium.DTDNode {
				continue
			}
			if child.Type() == helium.ElementNode && !doctypeEmitted {
				_, _ = io.WriteString(w, "<!DOCTYPE html>")
				doctypeEmitted = true
			}
			if err := htmlpkg.WriteNode(w, child, nodeOpts...); err != nil {
				return err
			}
		}
		return nil
	}

	opts := []htmlpkg.WriteOption{
		htmlpkg.WithNoDefaultDTD(),
		htmlpkg.WithNoFormat(),
		htmlpkg.WithPreserveCase(),
	}
	return htmlpkg.WriteDoc(w, doc, opts...)
}

// serializeXHTML serializes using the XHTML output method.
// XHTML is essentially XML with HTML-specific additions:
// - meta charset tag in <head>
// - For HTML5, simplified DOCTYPE
// - Self-closing void elements with a space before />
func serializeXHTML(w io.Writer, doc *helium.Document, outDef *OutputDef, charMap map[rune]string) error {
	isHTML5 := outDef.HTMLVersion == "5" || outDef.HTMLVersion == "5.0"

	// For HTML5, replace the doctype with <!DOCTYPE html>
	if isHTML5 {
		// Override doctype: for XHTML5, the spec says to emit <!DOCTYPE html>
		// regardless of doctype-public/doctype-system when html-version="5".
		outDef = &OutputDef{
			Name:               outDef.Name,
			Method:             outDef.Method,
			MethodExplicit:     outDef.MethodExplicit,
			Encoding:           outDef.Encoding,
			Indent:             outDef.Indent,
			OmitDeclaration:    outDef.OmitDeclaration,
			Standalone:         outDef.Standalone,
			CDATASections:      outDef.CDATASections,
			MediaType:          outDef.MediaType,
			Version:            outDef.Version,
			UndeclarePrefixes:  outDef.UndeclarePrefixes,
			IncludeContentType: outDef.IncludeContentType,
			ItemSeparator:      outDef.ItemSeparator,
			HTMLVersion:        outDef.HTMLVersion,
			NormalizationForm:  outDef.NormalizationForm,
			UseCharacterMaps:   outDef.UseCharacterMaps,
			ResolvedCharMap:    outDef.ResolvedCharMap,
		}
		// Remove existing DTD and replace with HTML5 DOCTYPE
		if dtd := doc.IntSubset(); dtd != nil {
			helium.UnlinkNode(dtd)
		}
		_, _ = doc.CreateInternalSubset("html", "", "")
	}

	// Insert <meta http-equiv="Content-Type"> in <head>
	if outDef.IncludeContentType == nil || *outDef.IncludeContentType {
		insertHTMLMeta(doc, outDef)
	}

	// Serialize as XML, then post-process for XHTML rules:
	// - Void elements: add space before /> (e.g., <br /> not <br/>)
	// - Non-void elements: expand self-closing to open+close (e.g., <Option></Option>)
	var buf bytes.Buffer
	if err := serializeXML(&buf, doc, outDef, charMap); err != nil {
		return err
	}
	_, err := io.WriteString(w, fixXHTMLSelfClosing(buf.String()))
	return err
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

// htmlURIAttrs lists HTML attributes whose values are URIs and should not
// have character maps applied (they use URI-escaping instead).
var htmlURIAttrs = map[string]struct{}{
	"href": {}, "src": {}, "action": {}, "cite": {}, "data": {},
	"formaction": {}, "poster": {}, "codebase": {}, "longdesc": {},
	"usemap": {}, "background": {}, "profile": {},
}

// applyCharMapToHTMLText applies a character map to serialized HTML output,
// applying to text content and non-URI attribute values, but skipping
// URI attributes (href, src, etc.) per the XSLT serialization spec.
func applyCharMapToHTMLText(html string, charMap map[rune]string) string {
	var out strings.Builder
	out.Grow(len(html))
	i := 0
	for i < len(html) {
		if html[i] == '<' {
			// Inside a tag — process attribute by attribute
			tagEnd := strings.IndexByte(html[i:], '>')
			if tagEnd < 0 {
				out.WriteString(html[i:])
				break
			}
			tag := html[i : i+tagEnd+1]
			out.WriteString(applyCharMapToHTMLTag(tag, charMap))
			i += tagEnd + 1
			continue
		}
		// Text content — apply character map
		r, size := utf8.DecodeRuneInString(html[i:])
		if repl, ok := charMap[r]; ok {
			out.WriteString(repl)
		} else {
			out.WriteString(html[i : i+size])
		}
		i += size
	}
	return out.String()
}

// applyCharMapToHTMLTag applies character map to attribute values within an
// HTML tag, skipping URI attributes.
func applyCharMapToHTMLTag(tag string, charMap map[rune]string) string {
	// For closing tags and self-closing without attributes, return as-is
	if strings.HasPrefix(tag, "</") || !strings.Contains(tag, "=") {
		return tag
	}
	var out strings.Builder
	out.Grow(len(tag))
	i := 0
	for i < len(tag) {
		// Find attribute name=value pairs
		eqIdx := strings.IndexByte(tag[i:], '=')
		if eqIdx < 0 {
			out.WriteString(tag[i:])
			break
		}
		// Find the attribute name (word before =)
		nameEnd := i + eqIdx
		nameStart := nameEnd - 1
		for nameStart > i && tag[nameStart] != ' ' && tag[nameStart] != '\t' && tag[nameStart] != '\n' {
			nameStart--
		}
		if tag[nameStart] == ' ' || tag[nameStart] == '\t' || tag[nameStart] == '\n' {
			nameStart++
		}
		attrName := strings.ToLower(tag[nameStart:nameEnd])
		_, isURI := htmlURIAttrs[attrName]

		// Write everything up to and including the =
		out.WriteString(tag[i : i+eqIdx+1])
		i += eqIdx + 1

		// Read the attribute value
		if i >= len(tag) {
			break
		}
		quote := tag[i]
		if quote == '"' || quote == '\'' {
			out.WriteByte(quote)
			i++
			endQuote := strings.IndexByte(tag[i:], quote)
			if endQuote < 0 {
				out.WriteString(tag[i:])
				break
			}
			attrVal := tag[i : i+endQuote]
			if isURI {
				out.WriteString(attrVal)
			} else {
				out.WriteString(applyCharacterMap(attrVal, charMap))
			}
			out.WriteByte(quote)
			i += endQuote + 1
		}
	}
	return out.String()
}

// hasDOEMarkers checks if the document contains any disable-output-escaping markers.
func hasDOEMarkers(doc *helium.Document) bool {
	found := false
	_ = helium.Walk(doc, func(n helium.Node) error {
		if n.Type() == helium.ProcessingInstructionNode && string(n.Name()) == "disable-output-escaping" {
			found = true
		}
		return nil
	})
	return found
}

// applyCharacterMap replaces characters in text according to the character map.
func applyCharacterMap(text string, charMap map[rune]string) string {
	var b strings.Builder
	for _, r := range text {
		if repl, ok := charMap[r]; ok {
			b.WriteString(repl)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// resolveCharacterMaps builds a merged character map from a list of map names.
func resolveCharacterMaps(ss *Stylesheet, names []string) map[rune]string {
	if len(names) == 0 || ss == nil || len(ss.characterMaps) == 0 {
		return nil
	}
	merged := make(map[rune]string)
	visited := make(map[string]bool)
	var resolve func(name string)
	resolve = func(name string) {
		if visited[name] {
			return
		}
		visited[name] = true
		cm := ss.characterMaps[name]
		if cm == nil {
			return
		}
		// Resolve referenced maps first (lower priority)
		for _, ref := range cm.UseCharacterMaps {
			resolve(ref)
		}
		// This map's entries override
		for r, s := range cm.Mappings {
			merged[r] = s
		}
	}
	for _, name := range names {
		resolve(name)
	}
	if len(merged) == 0 {
		return nil
	}
	return merged
}

// insertHTMLMeta inserts a <meta http-equiv="Content-Type"> element as the
// first child of the <head> element if not already present.
func insertHTMLMeta(doc *helium.Document, outDef *OutputDef) {
	root := doc.DocumentElement()
	if root == nil {
		return
	}
	// Find the <head> element (case-insensitive).
	var head *helium.Element
	for child := root.FirstChild(); child != nil; child = child.NextSibling() {
		if e, ok := child.(*helium.Element); ok && strings.EqualFold(e.Name(), "head") {
			head = e
			break
		}
	}
	if head == nil {
		return
	}
	// Check if a <meta http-equiv="Content-Type"> already exists.
	// Use case-insensitive attribute name matching for HTML compatibility.
	for child := head.FirstChild(); child != nil; child = child.NextSibling() {
		if e, ok := child.(*helium.Element); ok && strings.EqualFold(e.Name(), "meta") {
			for _, attr := range e.Attributes() {
				if strings.EqualFold(attr.Name(), "http-equiv") && strings.EqualFold(attr.Value(), "Content-Type") {
					return // already present
				}
			}
		}
	}
	// Create and insert the meta element.
	enc := outDef.Encoding
	if enc == "" {
		enc = "UTF-8"
	}
	meta, err := doc.CreateElement("meta")
	if err != nil {
		return
	}
	meta.SetLiteralAttribute("http-equiv", "Content-Type")
	// Use media-type if specified; otherwise default based on output method.
	mediaType := outDef.MediaType
	if mediaType == "" {
		if outDef.Method == "xhtml" {
			mediaType = "application/xhtml+xml"
		} else {
			mediaType = "text/html"
		}
	}
	meta.SetLiteralAttribute("content", mediaType+"; charset="+enc)
	// Insert meta as first child of <head>.
	// Unlink existing children, add meta, then re-add them.
	var children []helium.Node
	for child := head.FirstChild(); child != nil; {
		next := child.NextSibling()
		helium.UnlinkNode(child)
		children = append(children, child)
		child = next
	}
	_ = head.AddChild(meta)
	for _, child := range children {
		_ = head.AddChild(child)
	}
}
