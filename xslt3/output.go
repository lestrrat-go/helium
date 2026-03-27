package xslt3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/sequence"
	"github.com/lestrrat-go/helium/stream"
	"github.com/lestrrat-go/helium/xpath3"
	"golang.org/x/text/encoding/htmlindex"
)

// outputFrame represents the current output target during transformation.
type outputFrame struct {
	doc               *helium.Document   // result document being built
	current           helium.MutableNode // current insertion point
	captureItems      bool               // when true, xsl:sequence adds to pendingItems instead of DOM
	separateTextNodes bool               // when true, text nodes are captured as separate string items (prevents DOM merging)
	sequenceMode      bool               // when true, all nodes (text, element, attr, comment, PI) are captured as separate items
	mapConstructor    bool               // when true, xsl:map-entry emits single-entry maps into pendingItems
	pendingItems      xpath3.ItemSlice   // captured items from xsl:sequence
	prevWasAtomic     bool               // true when last xsl:sequence output was an atomic value (for inter-call space separation)
	emptyAtomicGen    uint64             // seqConstructorGen when prevWasAtomic was set by an empty-string atomic
	wherePopulated    bool               // when true, xsl:document emits document node (not children) so xsl:where-populated can check emptiness
	itemSeparator     *string            // item-separator serialization parameter; nil means default (" " between adjacent atomics)
	prevHadOutput     bool               // true when any item (node or atomic) was previously output; used for item-separator between non-atomic items
	outputSerial      int                // monotonically increases whenever visible output is produced
	seqConstructorGen uint64             // incremented each time executeSequenceConstructor is called
	conditionalScopes []conditionalScope
}

type conditionalKind int

const (
	conditionalOnEmpty conditionalKind = iota + 1
	conditionalOnNonEmpty
)

type conditionalAction struct {
	ctx           context.Context
	kind          conditionalKind
	content       xpath3.Sequence
	placeholder   helium.MutableNode
	prevWasAtomic bool // whether the output preceding this action was an atomic value
}

type conditionalScope struct {
	hasOutput      bool
	actions        []conditionalAction
	untrackedNodes []helium.MutableNode // nodes added via addNodeUntracked; removed when on-empty fires
}

func (out *outputFrame) noteOutput() {
	out.outputSerial++
}

// SerializeItems writes a sequence of items (maps, arrays, atomics, nodes)
// using the specified output definition's method (json or adaptive).
// This is used for result-documents with method="json" or method="adaptive".
func SerializeItems(w io.Writer, items xpath3.Sequence, doc *helium.Document, outDef *OutputDef) error {
	if outDef == nil {
		outDef = defaultOutputDef()
	}
	switch outDef.Method {
	case methodJSON:
		if len(outDef.ResolvedCharMap) > 0 {
			var buf strings.Builder
			if err := serializeJSONItems(&buf, items, doc, outDef); err != nil {
				return err
			}
			_, err := io.WriteString(w, applyCharMapJSON(buf.String(), outDef.ResolvedCharMap))
			return err
		}
		return serializeJSONItems(w, items, doc, outDef)
	case methodAdaptive:
		return serializeAdaptiveItems(w, items, doc, outDef.ItemSeparator, outDef.ResolvedCharMap)
	default:
		if items != nil && sequence.Len(items) > 0 {
			return serializeItemsWithSeparator(w, items, doc, outDef)
		}
		return SerializeResult(w, doc, outDef)
	}
}

// serializeItemsWithSeparator serializes a sequence of items using the specified
// output method, joining them with the item-separator.
func serializeItemsWithSeparator(w io.Writer, items xpath3.Sequence, doc *helium.Document, outDef *OutputDef) error {
	sep := "\n"
	if outDef.ItemSeparator != nil {
		sep = *outDef.ItemSeparator
	} else if outDef.ItemSeparatorAbsent {
		sep = ""
	}
	idx := 0
	for item := range sequence.Items(items) {
		if idx > 0 && sep != "" {
			if _, err := io.WriteString(w, sep); err != nil {
				return err
			}
		}
		switch v := item.(type) {
		case xpath3.NodeItem:
			var buf bytes.Buffer
			switch v.Node.(type) {
			case *helium.Element, *helium.Document:
				_ = helium.NewWriter().XMLDeclaration(false).WriteTo(&buf, v.Node)
			default:
				if v.Node.Type() == helium.CommentNode {
					buf.WriteString("<!--")
					buf.WriteString(string(v.Node.Content()))
					buf.WriteString("-->")
				} else if v.Node.Type() == helium.ProcessingInstructionNode {
					buf.WriteString("<?")
					buf.WriteString(v.Node.Name())
					if c := string(v.Node.Content()); c != "" {
						buf.WriteByte(' ')
						buf.WriteString(c)
					}
					buf.WriteString("?>")
				} else {
					buf.WriteString(string(v.Node.Content()))
				}
			}
			if _, err := w.Write(buf.Bytes()); err != nil {
				return err
			}
		case xpath3.AtomicValue:
			s, _ := xpath3.AtomicToString(v)
			if _, err := io.WriteString(w, s); err != nil {
				return err
			}
		default:
			s := fmt.Sprintf("%v", item)
			if _, err := io.WriteString(w, s); err != nil {
				return err
			}
		}
		idx++
	}
	return nil
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
	// based on the document element.
	if !outDef.MethodExplicit && outDef.Method == methodXML {
		if root := doc.DocumentElement(); root != nil {
			if strings.EqualFold(root.Name(), "html") && root.URI() == "" {
				// Root is "html" in no namespace → HTML output method.
				outDef.Method = methodHTML
				outDef.OmitDeclaration = true
			} else if strings.EqualFold(string(root.LocalName()), "html") && string(root.URI()) == lexicon.NamespaceXHTML {
				// Root is "html" in XHTML namespace → XHTML output method.
				outDef.Method = methodXHTML
			}
		}
	}

	// Validate serialization parameters before proceeding.
	if err := validateSerializationParams(outDef, doc); err != nil {
		return err
	}

	var charMap map[rune]string
	if len(charMaps) > 0 {
		charMap = charMaps[0]
	}

	// Check if we need encoding conversion (non-UTF-8)
	enc := strings.ToLower(outDef.Encoding)
	isUTF16 := enc == "utf-16" || enc == "utf16"
	needsEncodingConversion := enc != "" && enc != "utf-8" && enc != "utf8" && !isUTF16

	// Check if we need Unicode normalization
	needsNormalization := outDef.NormalizationForm != "" && outDef.NormalizationForm != "NONE"

	// Per the serialization spec, character map output is "immune" to
	// normalization. When both are active, use sentinel-wrapped char
	// map substitutions so that normalization skips them.
	serCharMap := charMap
	var sentinelCharMap map[rune]string
	if needsNormalization && len(charMap) > 0 {
		sentinelCharMap = make(map[rune]string, len(charMap))
		for k, v := range charMap {
			sentinelCharMap[k] = "\x00CMSTART\x00" + v + "\x00CMEND\x00"
		}
		serCharMap = sentinelCharMap
	}

	// Buffer when post-processing is needed
	needsBuffer := needsEncodingConversion || needsNormalization || isUTF16
	var target io.Writer
	var buf bytes.Buffer
	if needsBuffer {
		target = &buf
	} else {
		target = w
	}

	// Emit BOM if requested or if UTF-16 (UTF-16 always gets a BOM per spec).
	if outDef.ByteOrderMark || isUTF16 {
		if isUTF16 {
			// UTF-16 BE BOM
			if _, werr := w.Write([]byte{0xFE, 0xFF}); werr != nil {
				return werr
			}
		} else {
			// UTF-8 BOM
			if _, werr := w.Write([]byte{0xEF, 0xBB, 0xBF}); werr != nil {
				return werr
			}
		}
	}

	var err error
	switch outDef.Method {
	case methodText:
		err = serializeText(target, doc, serCharMap)
	case methodHTML:
		var htmlBuf bytes.Buffer
		err = serializeHTML(&htmlBuf, doc, outDef)
		if err != nil {
			break
		}
		result := htmlBuf.String()
		if len(serCharMap) > 0 {
			result = applyCharMapToHTMLText(result, serCharMap)
		}
		_, err = io.WriteString(target, escapeC1ControlsInString(result))
	case methodXHTML:
		err = serializeXHTML(target, doc, outDef, serCharMap)
	case methodJSON:
		if len(serCharMap) == 0 {
			err = serializeJSONItems(target, nil, doc, outDef)
			break
		}
		var jsonBuf strings.Builder
		err = serializeJSONItems(&jsonBuf, nil, doc, outDef)
		if err != nil {
			break
		}
		_, err = io.WriteString(target, applyCharMapJSON(jsonBuf.String(), serCharMap))
	case methodAdaptive:
		err = serializeAdaptiveItems(target, nil, doc, outDef.ItemSeparator, serCharMap)
	default:
		err = serializeXML(target, doc, outDef, serCharMap)
	}
	if err != nil {
		return err
	}

	if needsBuffer {
		data := buf.Bytes()

		// Apply Unicode normalization if requested
		if needsNormalization {
			if sentinelCharMap != nil {
				// Extract sentinel-wrapped segments, normalize the rest,
				// then re-insert the original (un-normalized) segments.
				data = normalizeSentinelAware(data, outDef.NormalizationForm)
			} else {
				data = applyUnicodeNormalization(data, outDef.NormalizationForm)
			}
		}

		if isUTF16 {
			return transcodeToUTF16(w, data)
		}
		if needsEncodingConversion {
			return transcodeToEncoding(w, data, enc)
		}
		_, err = w.Write(data)
		return err
	}
	return nil
}

// serializeNodeWithMethod serializes a node using the specified output method.
// This is used for json-node-output-method to serialize nodes within JSON output.
func serializeNodeWithMethod(node helium.Node, method string) string {
	var buf bytes.Buffer
	switch method {
	case methodHTML:
		doc := wrapNodeInHTMLDoc(node)
		outDef := defaultOutputDef()
		outDef.Method = methodHTML
		outDef.OmitDeclaration = true
		_ = serializeHTML(&buf, doc, outDef)
		s := buf.String()
		// If we wrapped the node in <html>, strip the wrapper tags
		if elem, ok := node.(*helium.Element); ok && !strings.EqualFold(string(elem.LocalName()), "html") {
			s = strings.TrimPrefix(s, "<html>")
			s = strings.TrimSuffix(s, "</html>")
		}
		return s
	case methodXHTML:
		doc := wrapNodeInDoc(node)
		outDef := defaultOutputDef()
		outDef.Method = methodXHTML
		_ = serializeXHTML(&buf, doc, outDef, nil)
		return buf.String()
	case methodText:
		return nodeStringValue(node)
	default: // "xml" or empty
		switch node.(type) {
		case *helium.Element, *helium.Document:
			_ = helium.NewWriter().XMLDeclaration(false).WriteTo(&buf, node)
		default:
			buf.WriteString(string(node.Content()))
		}
		return buf.String()
	}
}

// wrapNodeInHTMLDoc wraps a node in an HTML document structure.
// If the node is already an <html> element, it becomes the document element.
// Otherwise it is wrapped inside an <html> element so that insertHTMLMeta
// can locate the <head> element as a child of the root.
func wrapNodeInHTMLDoc(node helium.Node) *helium.Document {
	if doc, ok := node.(*helium.Document); ok {
		return doc
	}
	doc := helium.NewDocument("", "", helium.StandaloneNoXMLDecl)
	if elem, ok := node.(*helium.Element); ok {
		copied, err := helium.CopyNode(elem, doc)
		if err != nil {
			return doc
		}
		copiedElem := copied.(*helium.Element)
		// Remove redundant namespace declarations from descendants
		removeRedundantNamespaces(copiedElem)
		if strings.EqualFold(string(copiedElem.LocalName()), "html") {
			_ = doc.AddChild(copiedElem)
		} else {
			// Wrap in an <html> element
			htmlElem := doc.CreateElement("html")
			_ = doc.AddChild(htmlElem)
			_ = htmlElem.AddChild(copiedElem)
		}
	}
	return doc
}

// removeRedundantNamespaces removes namespace declarations from descendant
// elements that are the same as their parent's. After CopyNode, each element
// may carry its own copy of namespace declarations that were inherited in the
// original tree.
func removeRedundantNamespaces(root *helium.Element) {
	rootNS := map[string]string{} // prefix -> uri
	for _, ns := range root.Namespaces() {
		rootNS[ns.Prefix()] = ns.URI()
	}
	_ = helium.Walk(root, helium.NodeWalkerFunc(func(n helium.Node) error {
		if n == root {
			return nil
		}
		child, ok := n.(*helium.Element)
		if !ok {
			return nil
		}
		for _, ns := range child.Namespaces() {
			prefix := ns.Prefix()
			uri := ns.URI()
			if parentURI, exists := rootNS[prefix]; exists && parentURI == uri {
				child.RemoveNamespaceByPrefix(prefix)
			}
		}
		return nil
	}))
}

// wrapNodeInDoc wraps a node in a Document for serialization purposes.
func wrapNodeInDoc(node helium.Node) *helium.Document {
	if doc, ok := node.(*helium.Document); ok {
		return doc
	}
	doc := helium.NewDocument("", "", helium.StandaloneNoXMLDecl)
	if elem, ok := node.(*helium.Element); ok {
		copied, err := helium.CopyNode(elem, doc)
		if err == nil {
			_ = doc.AddChild(copied)
		}
	}
	return doc
}

// nodeStringValue returns the string value of a node.
func nodeStringValue(n helium.Node) string {
	if n == nil {
		return ""
	}
	return string(n.Content())
}

func serializeText(w io.Writer, doc *helium.Document, charMap map[rune]string) error {
	// Text output: just write the text content of the document
	sw := stream.NewWriter(w)
	err := helium.Walk(doc, helium.NodeWalkerFunc(func(n helium.Node) error {
		switch n.Type() {
		case helium.TextNode, helium.CDATASectionNode:
			text := string(n.Content())
			if len(charMap) > 0 {
				text = applyCharacterMap(text, charMap)
			}
			return sw.WriteRaw(text)
		}
		return nil
	}))
	if err != nil {
		return err
	}
	return sw.Flush()
}

func defaultOutputDef() *OutputDef {
	return &OutputDef{
		Method:   methodXML,
		Encoding: "UTF-8",
		Version:  "1.0",
	}
}

// validateSerializationParams checks serialization parameters for errors
// per the XSLT 3.0 serialization spec.
func validateSerializationParams(outDef *OutputDef, doc *helium.Document) error {
	method := outDef.Method

	// SEPM0004: standalone != "omit" with multiple element children of root
	if outDef.Standalone == lexicon.ValueYes || outDef.Standalone == lexicon.ValueNo {
		if method == methodXML || method == methodXHTML {
			elemCount := countRootElements(doc)
			if elemCount > 1 {
				return dynamicError(errCodeSEPM0004,
					"standalone=%q specified but result has %d root elements", outDef.Standalone, elemCount)
			}
		}
	}

	// SEPM0004: doctype-system with multiple element children of root
	if outDef.DoctypeSystem != "" {
		if method == methodXML || method == methodXHTML {
			elemCount := countRootElements(doc)
			if elemCount > 1 {
				return dynamicError(errCodeSEPM0004,
					"doctype-system specified but result has %d root elements", elemCount)
			}
		}
	}

	// SEPM0009: omit-xml-declaration="yes" conflicts with standalone or doctype-system
	// Only applicable for xml/xhtml methods — text/html/json don't have XML declarations.
	if outDef.OmitDeclaration && (method == methodXML || method == methodXHTML) {
		if outDef.Standalone == lexicon.ValueYes || outDef.Standalone == lexicon.ValueNo {
			return dynamicError(errCodeSEPM0009,
				"omit-xml-declaration=\"yes\" conflicts with standalone=%q", outDef.Standalone)
		}
		if outDef.DoctypeSystem != "" && outDef.Version != "" && outDef.Version != "1.0" {
			return dynamicError(errCodeSEPM0009,
				"omit-xml-declaration=\"yes\" conflicts with doctype-system and version=%q", outDef.Version)
		}
	}

	// SEPM0010: undeclare-prefixes="yes" with version="1.0"
	if outDef.UndeclarePrefixes && outDef.Version == "1.0" {
		return dynamicError(errCodeSEPM0010,
			"undeclare-prefixes=\"yes\" is not allowed with version=\"1.0\"")
	}

	// SEPM0016: invalid doctype-public (contains non-pubid characters)
	if outDef.DoctypePublic != "" {
		if !isValidPublicID(outDef.DoctypePublic) {
			return dynamicError(errCodeSEPM0016,
				"doctype-public %q contains invalid characters", outDef.DoctypePublic)
		}
	}

	// SESU0007: unsupported encoding for any output method
	{
		enc := strings.ToLower(outDef.Encoding)
		if enc != "" && enc != "utf-8" && enc != "utf8" && enc != "utf-16" && enc != "utf16" {
			_, encErr := htmlindex.Get(enc)
			if encErr != nil {
				return dynamicError(errCodeSESU0007,
					"unsupported encoding %q for %s output method", outDef.Encoding, method)
			}
		}
	}

	// SESU0007: unsupported version for html output (only when method explicitly set)
	if method == methodHTML && outDef.MethodExplicit && outDef.Version != "" {
		v, err := strconv.ParseFloat(outDef.Version, 64)
		if err == nil && v != 4.0 && v != 4.01 && v != 5.0 {
			return dynamicError(errCodeSESU0007,
				"unsupported version %q for html output method", outDef.Version)
		}
	}

	// SESU0011: unsupported normalization-form
	if outDef.NormalizationForm != "" && outDef.NormalizationForm != "NONE" {
		switch outDef.NormalizationForm {
		case "NFC", "NFD", "NFKC", "NFKD", "FULLY-NORMALIZED":
			// supported
		default:
			return dynamicError(errCodeSESU0011,
				"unsupported normalization-form %q", outDef.NormalizationForm)
		}
	}

	// SERE0012: fully-normalized and result begins with combining character
	if outDef.NormalizationForm == "FULLY-NORMALIZED" {
		if err := checkFullyNormalized(doc); err != nil {
			return err
		}
	}

	// SERE0014: HTML method with characters in #x7F-#x9F range in text.
	// HTML5 allows these characters as character references, so skip for version >= 5.
	// For HTML 4.x, raise the error as required by the spec.
	// XSLT 3.0 §20: the default value of html-version is 5.
	if method == methodHTML && !effectiveHTMLVersion5(outDef) {
		if err := checkHTMLInvalidChars(doc); err != nil {
			return err
		}
	}

	// SERE0015: ">" in PI content for HTML output
	if method == methodHTML {
		if err := checkHTMLPIContent(doc); err != nil {
			return err
		}
	}

	return nil
}

// countRootElements counts the number of element children of the document root.
func countRootElements(doc *helium.Document) int {
	count := 0
	for child := range helium.Children(doc) {
		if child.Type() == helium.ElementNode {
			count++
		}
	}
	return count
}

// isValidPublicID checks if a string is a valid public identifier.
// Valid characters: [a-zA-Z0-9], space, newline, '-', '(', ')', '+', ',',
// '.', '/', ':', '=', '?', ';', '!', '*', '#', '@', '$', '_', '%'
func isValidPublicID(s string) bool {
	for _, r := range s {
		if !isPubidChar(r) {
			return false
		}
	}
	return true
}

func isPubidChar(r rune) bool {
	if r >= 'a' && r <= 'z' {
		return true
	}
	if r >= 'A' && r <= 'Z' {
		return true
	}
	if r >= '0' && r <= '9' {
		return true
	}
	switch r {
	case ' ', '\n', '\r', '-', '\'', '(', ')', '+', ',', '.', '/', ':', '=', '?', ';', '!', '*', '#', '@', '$', '_', '%':
		return true
	}
	return false
}

// checkFullyNormalized checks if the result tree violates fully-normalized
// constraints (e.g., starts with a combining character).
func checkFullyNormalized(doc *helium.Document) error {
	var firstErr error
	_ = helium.Walk(doc, helium.NodeWalkerFunc(func(n helium.Node) error {
		if n.Type() == helium.TextNode || n.Type() == helium.CDATASectionNode {
			content := string(n.Content())
			if len(content) > 0 {
				r, _ := utf8.DecodeRuneInString(content)
				if unicode.In(r, unicode.Mn, unicode.Mc, unicode.Me) {
					firstErr = dynamicError(errCodeSERE0012,
						"fully-normalized output begins with combining character U+%04X", r)
					return firstErr
				}
			}
		}
		return nil
	}))
	return firstErr
}

// checkHTMLInvalidChars checks for characters in the #x7F-#x9F range in
// HTML text content (SERE0014).
func checkHTMLInvalidChars(doc *helium.Document) error {
	var firstErr error
	_ = helium.Walk(doc, helium.NodeWalkerFunc(func(n helium.Node) error {
		if n.Type() == helium.TextNode || n.Type() == helium.CDATASectionNode {
			content := string(n.Content())
			for _, r := range content {
				if r >= 0x7F && r <= 0x9F && r != 0x85 {
					firstErr = dynamicError(errCodeSERE0014,
						"HTML output contains character U+%04X in #x7F-#x9F range", r)
					return firstErr
				}
			}
		}
		return nil
	}))
	return firstErr
}

// checkHTMLPIContent checks that no PI in the result tree contains ">".
func checkHTMLPIContent(doc *helium.Document) error {
	var err error
	_ = helium.Walk(doc, helium.NodeWalkerFunc(func(n helium.Node) error {
		if n.Type() == helium.ProcessingInstructionNode {
			content := string(n.Content())
			if strings.Contains(content, ">") {
				err = dynamicError(errCodeSERE0015,
					"processing instruction content contains '>' in HTML output")
				return err
			}
		}
		return nil
	}))
	return err
}

// effectiveHTMLVersion5 returns true when the output definition's effective
// HTML version is 5 or higher. It checks HTMLVersion first, falls back to
// Version, and defaults to 5 per XSLT 3.0 §20.
func effectiveHTMLVersion5(outDef *OutputDef) bool {
	if outDef.HTMLVersion != "" {
		return isHTMLVersion5(outDef.HTMLVersion)
	}
	if outDef.Version != "" {
		return isHTMLVersion5(outDef.Version)
	}
	// XSLT 3.0 default: html-version=5
	return true
}

// isHTMLVersion5 returns true when the html-version string represents
// version 5 or higher (e.g. "5", "5.0", "5.00", "5.1").
func isHTMLVersion5(v string) bool {
	if v == "" {
		return false
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return false
	}
	return f >= 5.0
}
