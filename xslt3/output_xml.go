package xslt3

import (
	"errors"
	"io"
	"strings"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/writerctl"
	"github.com/lestrrat-go/helium/stream"
)

func serializeXML(w io.Writer, doc *helium.Document, outDef *OutputDef, charMap map[rune]string) error {
	// For non-UTF-8 encodings, use the stream-based serializer which
	// always outputs UTF-8. The encoding conversion is handled by
	// serializeResult's transcoding layer.
	targetEnc := strings.ToLower(outDef.Encoding)
	isNonUTF8 := targetEnc != "" && targetEnc != "utf-8" && targetEnc != lexicon.EncodingUTF8Alt
	// When the document has no document element (e.g., result-document
	// producing only comments and text), use the stream-based serializer
	// which does not inject newlines between top-level children.
	noDocElem := doc.DocumentElement() == nil
	// The stream-based serializer honors the XML 1.1 output-version declaration
	// and the undeclare-prefixes namespace-undeclaration output; the default
	// helium.Writer path does neither, so route to it when either applies.
	needsXML11 := outDef.UndeclarePrefixes || (outDef.Version != "" && outDef.Version != lexicon.XSLTVersion10)
	if len(charMap) > 0 || hasDOEMarkers(doc) || isNonUTF8 || len(outDef.CDATASections) > 0 || (outDef.Indent && len(outDef.SuppressIndentation) > 0) || noDocElem || needsXML11 {
		return serializeXMLWithCharMap(w, doc, outDef, charMap)
	}
	// Set encoding on the document so the XML declaration includes it.
	if outDef.Encoding != "" && doc.Encoding() == lexicon.EncodingUTF8Alt {
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
	// This path only handles XML 1.0-family output (needsXML11 is false above),
	// so a character invalid in XML 1.0 is a SERE0006 serialization error. The
	// check is folded into the writer's existing escape pass (no extra traversal).
	// Apply a valid output definition version so XML 1.0 overrides a document's
	// own version on this direct writer path.
	writer := helium.NewWriter().EscapeNonASCII(false).RejectInvalidChars(true)
	if version := validOutputXMLVersion(outDef.Version); version != "" {
		writer = writer.OutputVersion(version)
	}
	writerNormalizationForm := outDef.NormalizationForm
	if writerNormalizationForm == lexicon.NormFullyNormalized {
		writerNormalizationForm = normalizationFormNFC
	} else if _, ok := resolveNormForm(writerNormalizationForm); !ok {
		writerNormalizationForm = ""
	}
	writer = writer.Normalization(writerNormalizationForm)
	if outDef.Indent {
		writer = writer.Format(true)
	}
	if outDef.OmitDeclaration {
		writer = writer.XMLDeclaration(false)
	}
	needStandalone := !outDef.OmitDeclaration && (outDef.Standalone == lexicon.ValueYes || outDef.Standalone == lexicon.ValueNo)
	needStripNewline := !outDef.Indent && !outDef.OmitDeclaration
	if !needStandalone && !needStripNewline {
		return xmlInvalidCharError(writeExactXMLNode(w, writer, doc))
	}
	var buf strings.Builder
	if err := writeExactXMLNode(&buf, writer, doc); err != nil {
		return xmlInvalidCharError(err)
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
	return writeFullString(w, out)
}

// writeExactXMLNode writes an XML node without adding bytes that are not part
// of the XSLT result. helium.Writer normally appends a newline after each
// Document child, so document nodes use its internal exact-document mode.
func writeExactXMLNode(w io.Writer, writer helium.Writer, node helium.Node) error {
	if _, ok := node.(*helium.Document); ok {
		if configured, ok := writerctl.OmitDocumentChildTerminators(writer).(helium.Writer); ok {
			writer = configured
		}
	}
	return writer.WriteTo(w, node)
}

func writeFullString(w io.Writer, s string) error {
	n, err := io.WriteString(w, s)
	if err != nil {
		return err
	}
	if n != len(s) {
		return io.ErrShortWrite
	}
	return nil
}

// xmlInvalidCharError maps the writer's ErrInvalidXMLChar sentinel to the
// XSLT serialization error SERE0006, passing every other error (and nil)
// through unchanged.
func xmlInvalidCharError(err error) error {
	if err != nil && errors.Is(err, helium.ErrInvalidXMLChar) {
		return dynamicError(errCodeSERE0006, "%s", err.Error())
	}
	return err
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
		return writeFullString(w, out)
	}
	return serializeXMLWithCharMapInner(w, doc, outDef, charMap)
}

func serializeXMLWithCharMapInner(w io.Writer, doc *helium.Document, outDef *OutputDef, charMap map[rune]string) error {
	sw := stream.NewWriter(w)
	// XML 1.1 output serializes restricted control characters as decimal
	// character references. Set this up front so it applies even when the XML
	// declaration is omitted (StartDocument, when written, sets it too).
	if outDef.Version == "1.1" {
		sw = sw.XMLVersion("1.1")
	}

	if !outDef.OmitDeclaration {
		enc := outDef.Encoding
		if enc == "" {
			enc = lexicon.EncodingUTF8U
		}
		// The output XML version defaults to 1.0; xsl:output/@version="1.1"
		// selects an XML 1.1 declaration.
		xmlVersion := lexicon.XSLTVersion10
		if outDef.Version == "1.1" {
			xmlVersion = outDef.Version
		}
		if err := sw.StartDocument(xmlVersion, enc, ""); err != nil {
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
		indent:            outDef.Indent,
		suppressSet:       suppressSet,
		undeclarePrefixes: outDef.UndeclarePrefixes,
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
	// undeclarePrefixes enables serialization of XML 1.1 prefixed namespace
	// undeclarations (xmlns:pfx="") from empty-URI namespace declarations in the
	// result tree. A default-namespace undeclaration (xmlns="") is always
	// serialized regardless (it is valid in XML 1.0).
	undeclarePrefixes bool
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
	if uri := elem.URI(); uri != "" {
		return helium.ClarkName(uri, elem.LocalName())
	}
	return elem.LocalName()
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
	if _, ok := suppressSet[elem.LocalName()]; ok {
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
			piName := child.Name()
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
			elem, _ := helium.AsNode[*helium.Element](child)
			// Write indentation before start tag
			if err := ictx.writeIndent(sw); err != nil {
				return err
			}
			prefix := elem.Prefix()
			local := elem.LocalName()
			uri := elem.URI()
			if err := sw.StartElementNS(prefix, local, uri); err != nil {
				return err
			}
			// Write additional namespace declarations not handled by StartElementNS
			elemPrefix := elem.Prefix()
			for _, ns := range elem.Namespaces() {
				if ns.Prefix() == elemPrefix {
					continue // already declared by StartElementNS
				}
				// The "xml" prefix is predefined by the Namespaces in XML
				// spec and bound implicitly everywhere, so a redundant,
				// non-canonical xmlns:xml declaration must never be emitted.
				if ns.Prefix() == "xml" {
					continue
				}
				if ns.Prefix() == "" {
					if err := sw.WriteAttribute("xmlns", ns.URI()); err != nil {
						return err
					}
				} else {
					// A prefixed namespace undeclaration (empty URI) is an XML 1.1
					// construct; emit it only when undeclare-prefixes is enabled,
					// otherwise the prefix is simply left undeclared in the output.
					if ns.URI() == "" && !ictx.undeclarePrefixes {
						continue
					}
					if err := sw.WriteAttribute("xmlns:"+ns.Prefix(), ns.URI()); err != nil {
						return err
					}
				}
			}
			// Write attributes
			for _, attr := range elem.Attributes() {
				if err := writeAttrWithCharMap(sw, attr.Name(), attr.Value(), charMap, normForm); err != nil {
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
				if err := sw.WriteRaw(normalizeRawXMLContent(text, normForm)); err != nil {
					return err
				}
			} else if inCDATAElement(n, cdataElems) {
				if err := writeCDATAWithEncoding(sw, text, encoding, normForm); err != nil {
					return err
				}
			} else if err := writeTextWithCharMap(sw, text, charMap, normForm); err != nil {
				return err
			}
		case helium.CommentNode:
			if err := sw.WriteComment(string(child.Content())); err != nil {
				return err
			}
		case helium.ProcessingInstructionNode:
			if err := sw.WritePI(child.Name(), string(child.Content())); err != nil {
				return err
			}
		}
	}
	return nil
}

// writeTextWithCharMap writes text content, applying character map substitutions.
// Mapped characters are written raw (unescaped), unmapped characters are written
// as normal text (with XML escaping).
func writeTextWithCharMap(sw *stream.Writer, text string, charMap map[rune]string, normalizationForm string) error {
	var unmapped strings.Builder
	flushUnmapped := func() error {
		if unmapped.Len() == 0 {
			return nil
		}
		if err := sw.WriteString(normalizeText(unmapped.String(), normalizationForm)); err != nil {
			return err
		}
		unmapped.Reset()
		return nil
	}
	for _, r := range text {
		if repl, ok := charMap[r]; ok {
			// Flush any accumulated unmapped text first
			if err := flushUnmapped(); err != nil {
				return err
			}
			// Write the replacement raw
			if err := sw.WriteRaw(repl); err != nil {
				return err
			}
		} else {
			unmapped.WriteRune(r)
		}
	}
	return flushUnmapped()
}

// writeAttrWithCharMap writes an XML attribute with character map awareness.
// Mapped characters are written raw (unescaped) while unmapped characters
// go through normal XML attribute escaping.
func writeAttrWithCharMap(sw *stream.Writer, name, value string, charMap map[rune]string, normalizationForm string) error {
	if len(charMap) == 0 {
		return sw.WriteAttribute(name, normalizeText(value, normalizationForm))
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
		return sw.WriteAttribute(name, normalizeText(value, normalizationForm))
	}
	// Write attribute with mixed raw/escaped content
	if err := sw.StartAttribute(name); err != nil {
		return err
	}
	var unmapped strings.Builder
	flushUnmapped := func() error {
		if unmapped.Len() == 0 {
			return nil
		}
		if err := sw.WriteString(normalizeText(unmapped.String(), normalizationForm)); err != nil {
			return err
		}
		unmapped.Reset()
		return nil
	}
	for _, r := range value {
		if repl, ok := charMap[r]; ok {
			if err := flushUnmapped(); err != nil {
				return err
			}
			if err := sw.WriteRaw(repl); err != nil {
				return err
			}
		} else {
			unmapped.WriteRune(r)
		}
	}
	if err := flushUnmapped(); err != nil {
		return err
	}
	return sw.EndAttribute()
}
