package xslt3

import (
	"io"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/stream"
)

func serializeXML(w io.Writer, doc *helium.Document, outDef *OutputDef, charMap map[rune]string) error {
	// For non-UTF-8 encodings, use the stream-based serializer which
	// always outputs UTF-8. The encoding conversion is handled by
	// serializeResult's transcoding layer.
	targetEnc := strings.ToLower(outDef.Encoding)
	isNonUTF8 := targetEnc != "" && targetEnc != "utf-8" && targetEnc != "utf8"
	// When the document has no document element (e.g., result-document
	// producing only comments and text), use the stream-based serializer
	// which does not inject newlines between top-level children.
	noDocElem := doc.DocumentElement() == nil
	if len(charMap) > 0 || hasDOEMarkers(doc) || isNonUTF8 || len(outDef.CDATASections) > 0 || (outDef.Indent && len(outDef.SuppressIndentation) > 0) || noDocElem {
		return serializeXMLWithCharMap(w, doc, outDef, charMap)
	}
	// Set encoding on the document so the XML declaration includes it.
	if outDef.Encoding != "" && doc.Encoding() == "utf8" {
		doc.SetEncoding(outDef.Encoding)
	}
	// Per XSLT spec, doctype-public without doctype-system is ignored for xml method.
	if outDef.DoctypeSystem == "" && outDef.DoctypePublic != "" {
		outDef.DoctypePublic = ""
	}
	// Add DOCTYPE if doctype-system is specified and
	// the document doesn't already have a DTD.
	if outDef.DoctypeSystem != "" && doc.IntSubset() == nil {
		rootName := "html" // default
		if root := doc.DocumentElement(); root != nil {
			rootName = root.Name()
		}
		if _, err := doc.CreateInternalSubset(rootName, outDef.DoctypePublic, outDef.DoctypeSystem); err != nil {
			return err
		}
	}
	writer := helium.NewWriter().EscapeNonASCII(false)
	if outDef.Indent {
		writer = writer.Format(true)
	}
	if outDef.OmitDeclaration {
		writer = writer.XMLDeclaration(false)
	}
	// When standalone is "yes" or "no", or when indent="no" and
	// the declaration is not omitted, buffer and post-process.
	needStandalone := !outDef.OmitDeclaration && (outDef.Standalone == lexicon.ValueYes || outDef.Standalone == lexicon.ValueNo)
	needStripNewline := !outDef.Indent && !outDef.OmitDeclaration
	if needStandalone || needStripNewline {
		var buf strings.Builder
		if err := writer.WriteTo(&buf, doc); err != nil {
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
	return writer.WriteTo(w, doc)
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
	needStandalone := !outDef.OmitDeclaration && (outDef.Standalone == lexicon.ValueYes || outDef.Standalone == lexicon.ValueNo)
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

	// Add DOCTYPE if doctype-system is specified (doctype-public alone is ignored for xml).
	if outDef.DoctypeSystem != "" {
		rootName := "html"
		if root := doc.DocumentElement(); root != nil {
			rootName = root.Name()
		}
		if err := sw.WriteDTD(rootName, outDef.DoctypePublic, outDef.DoctypeSystem, ""); err != nil {
			return err
		}
	}

	// Build CDATA section element set for fast lookup.
	var cdataSet map[string]struct{}
	if len(outDef.CDATASections) > 0 {
		cdataSet = make(map[string]struct{}, len(outDef.CDATASections))
		for _, name := range outDef.CDATASections {
			cdataSet[name] = struct{}{}
		}
	}

	enc := strings.ToLower(outDef.Encoding)

	// Build suppress-indentation set
	var suppressSet map[string]struct{}
	if len(outDef.SuppressIndentation) > 0 {
		suppressSet = make(map[string]struct{}, len(outDef.SuppressIndentation))
		for _, name := range outDef.SuppressIndentation {
			suppressSet[name] = struct{}{}
		}
	}

	ictx := &xmlIndentCtx{
		indent:      outDef.Indent,
		suppressSet: suppressSet,
	}

	err := serializeXMLNodeWithCharMap(&sw, doc, charMap, cdataSet, enc, outDef.NormalizationForm, ictx)
	if err != nil {
		return err
	}
	return sw.Flush()
}

// xmlIndentCtx tracks indentation state for XML serialization with
// suppress-indentation support.
type xmlIndentCtx struct {
	indent      bool
	depth       int
	suppressSet map[string]struct{}
	suppressed  bool // true when inside a suppress-indentation element
}

func (ic *xmlIndentCtx) writeIndent(sw *stream.Writer) error {
	if !ic.indent || ic.suppressed {
		return nil
	}
	buf := make([]byte, 1+ic.depth*2)
	buf[0] = '\n'
	for i := 1; i < len(buf); i++ {
		buf[i] = ' '
	}
	return sw.WriteRaw(string(buf))
}

// expandedElemName returns the expanded name for matching suppress-indentation.
func expandedElemName(elem *helium.Element) string {
	if uri := string(elem.URI()); uri != "" {
		return helium.ClarkName(uri, string(elem.LocalName()))
	}
	return string(elem.LocalName())
}

// elemMatchesSuppressSet checks if the element name (with prefix or expanded)
// matches the suppress-indentation set.
func elemMatchesSuppressSet(elem *helium.Element, suppressSet map[string]struct{}) bool {
	if len(suppressSet) == 0 {
		return false
	}
	// Check expanded name
	if _, ok := suppressSet[expandedElemName(elem)]; ok {
		return true
	}
	// Check prefixed name
	name := elem.Name()
	if _, ok := suppressSet[name]; ok {
		return true
	}
	// Check local name
	if _, ok := suppressSet[string(elem.LocalName())]; ok {
		return true
	}
	return false
}

func collectChildren(n helium.Node) []helium.Node {
	var children []helium.Node
	for child := range helium.Children(n) {
		children = append(children, child)
	}
	return children
}

func elemHasChildElements(elem *helium.Element) bool {
	for child := range helium.Children(elem) {
		if child.Type() == helium.ElementNode {
			return true
		}
	}
	return false
}

func serializeXMLNodeWithCharMap(sw *stream.Writer, n helium.Node, charMap map[rune]string, cdataElems map[string]struct{}, encoding string, normForm string, ictx *xmlIndentCtx) error {
	doeActive := false
	children := collectChildren(n)
	hasChildElements := false
	for _, child := range children {
		if child.Type() == helium.ElementNode {
			hasChildElements = true
			break
		}
	}
	for _, child := range children {
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
			// Write indentation before start tag
			if err := ictx.writeIndent(sw); err != nil {
				return err
			}
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
				if err := writeAttrWithCharMap(sw, attr.Name(), attr.Value(), charMap); err != nil {
					return err
				}
			}
			// Track suppress-indentation
			wasSuppressed := ictx.suppressed
			if elemMatchesSuppressSet(elem, ictx.suppressSet) {
				ictx.suppressed = true
			}
			ictx.depth++
			// Recurse into children
			if err := serializeXMLNodeWithCharMap(sw, elem, charMap, cdataElems, encoding, normForm, ictx); err != nil {
				return err
			}
			ictx.depth--
			// Write indentation before end tag (only if element has child elements)
			if elemHasChildElements(elem) {
				if err := ictx.writeIndent(sw); err != nil {
					return err
				}
			}
			ictx.suppressed = wasSuppressed
			if err := sw.EndElement(); err != nil {
				return err
			}
		case helium.TextNode, helium.CDATASectionNode:
			text := string(child.Content())
			// When indenting and not suppressed, trim whitespace-only text
			// nodes that exist between elements (they are formatting whitespace).
			if ictx.indent && !ictx.suppressed && hasChildElements && strings.TrimSpace(text) == "" {
				continue
			}
			if doeActive {
				if err := sw.WriteRaw(text); err != nil {
					return err
				}
			} else if inCDATAElement(n, cdataElems) {
				if err := writeCDATAWithEncoding(sw, text, encoding, normForm); err != nil {
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

// writeAttrWithCharMap writes an XML attribute with character map awareness.
// Mapped characters are written raw (unescaped) while unmapped characters
// go through normal XML attribute escaping.
func writeAttrWithCharMap(sw *stream.Writer, name, value string, charMap map[rune]string) error {
	if len(charMap) == 0 {
		return sw.WriteAttribute(name, value)
	}
	// Check if any character in the value has a mapping
	hasMapped := false
	for _, r := range value {
		if _, ok := charMap[r]; ok {
			hasMapped = true
			break
		}
	}
	if !hasMapped {
		return sw.WriteAttribute(name, value)
	}
	// Write attribute with mixed raw/escaped content
	if err := sw.StartAttribute(name); err != nil {
		return err
	}
	var unmapped strings.Builder
	for _, r := range value {
		if repl, ok := charMap[r]; ok {
			if unmapped.Len() > 0 {
				if err := sw.WriteString(unmapped.String()); err != nil {
					return err
				}
				unmapped.Reset()
			}
			if err := sw.WriteRaw(repl); err != nil {
				return err
			}
		} else {
			unmapped.WriteRune(r)
		}
	}
	if unmapped.Len() > 0 {
		if err := sw.WriteString(unmapped.String()); err != nil {
			return err
		}
	}
	return sw.EndAttribute()
}
