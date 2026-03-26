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
	htmlpkg "github.com/lestrrat-go/helium/html"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/internal/sequence"
	"github.com/lestrrat-go/helium/stream"
	"github.com/lestrrat-go/helium/xpath3"
	"golang.org/x/text/encoding/htmlindex"
	xtunicode "golang.org/x/text/encoding/unicode"
	"golang.org/x/text/unicode/norm"
)

// outputFrame represents the current output target during transformation.
type outputFrame struct {
	doc               *helium.Document // result document being built
	current           helium.Node      // current insertion point
	captureItems      bool             // when true, xsl:sequence adds to pendingItems instead of DOM
	separateTextNodes bool             // when true, text nodes are captured as separate string items (prevents DOM merging)
	sequenceMode      bool             // when true, all nodes (text, element, attr, comment, PI) are captured as separate items
	mapConstructor    bool             // when true, xsl:map-entry emits single-entry maps into pendingItems
	pendingItems      xpath3.ItemSlice // captured items from xsl:sequence
	prevWasAtomic     bool             // true when last xsl:sequence output was an atomic value (for inter-call space separation)
	emptyAtomicGen    uint64           // seqConstructorGen when prevWasAtomic was set by an empty-string atomic
	wherePopulated    bool             // when true, xsl:document emits document node (not children) so xsl:where-populated can check emptiness
	itemSeparator     *string          // item-separator serialization parameter; nil means default (" " between adjacent atomics)
	prevHadOutput     bool             // true when any item (node or atomic) was previously output; used for item-separator between non-atomic items
	outputSerial      int              // monotonically increases whenever visible output is produced
	seqConstructorGen uint64           // incremented each time executeSequenceConstructor is called
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
	placeholder   helium.Node
	prevWasAtomic bool // whether the output preceding this action was an atomic value
}

type conditionalScope struct {
	hasOutput      bool
	actions        []conditionalAction
	untrackedNodes []helium.Node // nodes added via addNodeUntracked; removed when on-empty fires
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
			switch n := v.Node.(type) {
			case *helium.Element:
				_ = n.XML(&buf, helium.NewWriter().XMLDeclaration(false))
			case *helium.Document:
				_ = n.XML(&buf, helium.NewWriter().XMLDeclaration(false))
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

// serializeJSONItems serializes a sequence of items as JSON output.
// Per XSLT 3.0 §26: the sequence is serialized as a single JSON value.
func serializeJSONItems(w io.Writer, items xpath3.Sequence, doc *helium.Document, outDef *OutputDef) error {
	itemsLen := 0
	if items != nil {
		itemsLen = sequence.Len(items)
	}
	if itemsLen == 0 && doc != nil {
		// No captured items: serialize DOM content as text for JSON
		return serializeAdaptiveItems(w, items, doc, nil)
	}
	nodeMethod := ""
	if outDef != nil {
		nodeMethod = outDef.JSONNodeOutputMethod
	}
	if itemsLen == 1 {
		s, err := serializeItemJSON(items.Get(0), nodeMethod)
		if err != nil {
			return err
		}
		_, err = io.WriteString(w, s)
		return err
	}
	// Multiple items: serialize as JSON array
	members := make([]xpath3.Sequence, itemsLen)
	for i := range itemsLen {
		members[i] = xpath3.ItemSlice{items.Get(i)}
	}
	s, err := serializeItemJSON(xpath3.NewArray(members), nodeMethod)
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, s)
	return err
}

// serializeAdaptiveItems serializes a sequence of items using the adaptive
// serialization method. Each item is serialized according to its type.
func serializeAdaptiveItems(w io.Writer, items xpath3.Sequence, doc *helium.Document, itemSep *string, charMaps ...map[rune]string) error {
	if (items == nil || sequence.Len(items) == 0) && doc != nil {
		var cm map[rune]string
		if len(charMaps) > 0 {
			cm = charMaps[0]
		}
		return serializeXML(w, doc, defaultOutputDef(), cm)
	}
	var cm map[rune]string
	if len(charMaps) > 0 {
		cm = charMaps[0]
	}
	sep := "\n"
	if itemSep != nil {
		sep = *itemSep
	}
	// Per XSLT 3.0 §20: when adaptive output contains a single document or
	// element node, serialize it using the XML method (which includes the
	// XML declaration by default). When there are multiple items, serialize
	// each without a declaration.
	singleNodeItem := items != nil && sequence.Len(items) == 1
	adaptIdx := 0
	for item := range sequence.Items(items) {
		if adaptIdx > 0 && sep != "" {
			if _, err := io.WriteString(w, sep); err != nil {
				return err
			}
		}
		if singleNodeItem {
			if ni, ok := item.(xpath3.NodeItem); ok {
				if elem, ok := ni.Node.(*helium.Element); ok {
					// Wrap in a temp document to get the XML declaration.
					tmpDoc := helium.NewDefaultDocument()
					clone, _ := helium.CopyNode(elem, tmpDoc)
					if clone != nil {
						_ = tmpDoc.AddChild(clone)
					}
					xmlOutDef := defaultOutputDef()
					if err := serializeXML(w, tmpDoc, xmlOutDef, cm); err != nil {
						return err
					}
					continue
				}
			}
		}
		s := serializeItemAdaptive(item, cm)
		if _, err := io.WriteString(w, s); err != nil {
			return err
		}
		adaptIdx++
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
		if elem, ok := node.(*helium.Element); ok {
			_ = elem.XML(&buf, helium.NewWriter().XMLDeclaration(false))
		} else if doc, ok := node.(*helium.Document); ok {
			_ = doc.XML(&buf, helium.NewWriter().XMLDeclaration(false))
		} else {
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

// serializeItemJSON serializes a single XDM item as JSON.
// nodeMethod specifies the json-node-output-method for serializing nodes.
func serializeItemJSON(item xpath3.Item, nodeMethod string) (string, error) {
	switch v := item.(type) {
	case xpath3.MapItem:
		return serializeMapJSON(v, nodeMethod)
	case xpath3.ArrayItem:
		return serializeArrayJSON(v, nodeMethod)
	case xpath3.NodeItem:
		if nodeMethod != "" && nodeMethod != methodText {
			return jsonEscapeString(serializeNodeWithMethod(v.Node, nodeMethod)), nil
		}
		return jsonEscapeString(nodeStringValue(v.Node)), nil
	case xpath3.AtomicValue:
		return serializeAtomicJSON(v), nil
	default:
		if av, ok := item.(xpath3.AtomicValue); ok {
			return serializeAtomicJSON(av), nil
		}
		return jsonEscapeString(fmt.Sprintf("%v", item)), nil
	}
}

// serializeMapJSON serializes a map as a JSON object.
func serializeMapJSON(m xpath3.MapItem, nodeMethod string) (string, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	first := true
	var serErr error
	_ = m.ForEach(func(k xpath3.AtomicValue, v xpath3.Sequence) error {
		if serErr != nil {
			return serErr
		}
		if !first {
			buf.WriteByte(',')
		}
		first = false
		ks, _ := xpath3.AtomicToString(k)
		buf.WriteString(jsonEscapeString(ks))
		buf.WriteByte(':')
		vLen := 0
		if v != nil {
			vLen = sequence.Len(v)
		}
		switch vLen {
		case 1:
			s, err := serializeItemJSON(v.Get(0), nodeMethod)
			if err != nil {
				serErr = err
				return err
			}
			buf.WriteString(s)
		case 0:
			buf.WriteString("null")
		default:
			members := make([]xpath3.Sequence, vLen)
			for i := range vLen {
				members[i] = xpath3.ItemSlice{v.Get(i)}
			}
			s, err := serializeItemJSON(xpath3.NewArray(members), nodeMethod)
			if err != nil {
				serErr = err
				return err
			}
			buf.WriteString(s)
		}
		return nil
	})
	if serErr != nil {
		return "", serErr
	}
	buf.WriteByte('}')
	return buf.String(), nil
}

// serializeArrayJSON serializes an array as a JSON array.
func serializeArrayJSON(a xpath3.ArrayItem, nodeMethod string) (string, error) {
	var buf bytes.Buffer
	buf.WriteByte('[')
	members := a.Members()
	for i, member := range members {
		if i > 0 {
			buf.WriteByte(',')
		}
		mLen := 0
		if member != nil {
			mLen = sequence.Len(member)
		}
		switch mLen {
		case 1:
			s, err := serializeItemJSON(member.Get(0), nodeMethod)
			if err != nil {
				return "", err
			}
			buf.WriteString(s)
		case 0:
			buf.WriteString("null")
		default:
			// Multi-item member: serialize as array
			submembers := make([]xpath3.Sequence, mLen)
			for j := range mLen {
				submembers[j] = xpath3.ItemSlice{member.Get(j)}
			}
			s, err := serializeItemJSON(xpath3.NewArray(submembers), nodeMethod)
			if err != nil {
				return "", err
			}
			buf.WriteString(s)
		}
	}
	buf.WriteByte(']')
	return buf.String(), nil
}

// serializeAtomicJSON serializes an atomic value as JSON.
func serializeAtomicJSON(v xpath3.AtomicValue) string {
	typeName := v.TypeName
	switch typeName {
	case "xs:boolean":
		b, _ := xpath3.AtomicToString(v)
		return b
	case "xs:integer", "xs:int", "xs:long", "xs:short", "xs:byte",
		"xs:unsignedInt", "xs:unsignedLong", "xs:unsignedShort", "xs:unsignedByte",
		"xs:positiveInteger", "xs:nonNegativeInteger", "xs:negativeInteger", "xs:nonPositiveInteger":
		s, _ := xpath3.AtomicToString(v)
		return s
	case "xs:double", "xs:float", "xs:decimal":
		s, _ := xpath3.AtomicToString(v)
		if s == "NaN" || s == "INF" || s == "-INF" || s == "+INF" {
			return jsonEscapeString(s)
		}
		return s
	default:
		s, _ := xpath3.AtomicToString(v)
		return jsonEscapeString(s)
	}
}

// jsonEscapeString returns a JSON-escaped double-quoted string.
func jsonEscapeString(s string) string {
	var buf bytes.Buffer
	buf.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			buf.WriteString("\\\"")
		case '\\':
			buf.WriteString("\\\\")
		case '\n':
			buf.WriteString("\\n")
		case '\r':
			buf.WriteString("\\r")
		case '\t':
			buf.WriteString("\\t")
		case '\b':
			buf.WriteString("\\b")
		case '\f':
			buf.WriteString("\\f")
		case '/':
			buf.WriteString("\\/")
		default:
			if r < 0x20 {
				fmt.Fprintf(&buf, "\\u%04x", r)
			} else {
				buf.WriteRune(r)
			}
		}
	}
	buf.WriteByte('"')
	return buf.String()
}

// adaptiveQuoteString wraps a string in double quotes for adaptive output.
// Unlike JSON escaping, adaptive serialization only escapes embedded double
// quotes by doubling them (XSLT 3.0 §26.3.5). Other characters such as
// backslashes are emitted literally.
func adaptiveQuoteString(s string) string {
	var buf bytes.Buffer
	buf.WriteByte('"')
	for _, r := range s {
		if r == '"' {
			buf.WriteString("\"\"")
		} else {
			buf.WriteRune(r)
		}
	}
	buf.WriteByte('"')
	return buf.String()
}

// isAdaptiveQuotedType returns true when an atomic type should be serialized
// with double-quote wrapping in adaptive output (XSLT 3.0 §26.3.5).
// String-like types use JSON quoting; numeric types and xs:boolean do not.
func isAdaptiveQuotedType(typeName string) bool {
	switch typeName {
	case "xs:boolean",
		"xs:integer", "xs:decimal", "xs:float", "xs:double",
		"xs:long", "xs:int", "xs:short", "xs:byte",
		"xs:unsignedLong", "xs:unsignedInt", "xs:unsignedShort", "xs:unsignedByte",
		"xs:nonNegativeInteger", "xs:nonPositiveInteger",
		"xs:positiveInteger", "xs:negativeInteger":
		return false
	}
	return true
}

// serializeItemAdaptive serializes a single item using the adaptive method.
func serializeItemAdaptive(item xpath3.Item, charMap map[rune]string) string {
	maybeApply := func(s string) string {
		if len(charMap) > 0 {
			return applyCharMap(s, charMap)
		}
		return s
	}
	switch v := item.(type) {
	case xpath3.MapItem:
		return serializeMapAdaptive(v, charMap)
	case xpath3.ArrayItem:
		return serializeArrayAdaptive(v, charMap)
	case xpath3.NodeItem:
		var buf bytes.Buffer
		if elem, ok := v.Node.(*helium.Element); ok {
			_ = elem.XML(&buf, helium.NewWriter().XMLDeclaration(false))
		} else if doc, ok := v.Node.(*helium.Document); ok {
			_ = doc.XML(&buf, helium.NewWriter().XMLDeclaration(false))
		} else if attr, ok := v.Node.(*helium.Attribute); ok {
			buf.WriteString(attr.Name())
			buf.WriteString("=\"")
			buf.WriteString(string(attr.Content()))
			buf.WriteString("\"")
		} else {
			buf.WriteString(string(v.Node.Content()))
		}
		return maybeApply(buf.String())
	case xpath3.AtomicValue:
		s, _ := xpath3.AtomicToString(v)
		s = maybeApply(s)
		if isAdaptiveQuotedType(v.TypeName) {
			return adaptiveQuoteString(s)
		}
		return s
	default:
		if av, ok := item.(xpath3.AtomicValue); ok {
			s, _ := xpath3.AtomicToString(av)
			s = maybeApply(s)
			if isAdaptiveQuotedType(av.TypeName) {
				return adaptiveQuoteString(s)
			}
			return s
		}
		return fmt.Sprintf("%v", item)
	}
}

// serializeMapAdaptive serializes a map using adaptive serialization.
func serializeMapAdaptive(m xpath3.MapItem, charMap map[rune]string) string {
	var buf bytes.Buffer
	buf.WriteString("map{")
	first := true
	_ = m.ForEach(func(k xpath3.AtomicValue, v xpath3.Sequence) error {
		if !first {
			buf.WriteByte(',')
		}
		first = false
		ks, _ := xpath3.AtomicToString(k)
		buf.WriteString(jsonEscapeString(ks))
		buf.WriteByte(':')
		vLen2 := 0
		if v != nil {
			vLen2 = sequence.Len(v)
		}
		switch vLen2 {
		case 1:
			buf.WriteString(serializeItemAdaptive(v.Get(0), charMap))
		case 0:
			buf.WriteString("()")
		default:
			buf.WriteByte('(')
			for i := range vLen2 {
				if i > 0 {
					buf.WriteByte(',')
				}
				buf.WriteString(serializeItemAdaptive(v.Get(i), charMap))
			}
			buf.WriteByte(')')
		}
		return nil
	})
	buf.WriteByte('}')
	return buf.String()
}

// serializeArrayAdaptive serializes an array using adaptive serialization.
func serializeArrayAdaptive(a xpath3.ArrayItem, charMap map[rune]string) string {
	var buf bytes.Buffer
	buf.WriteByte('[')
	members := a.Members()
	for i, member := range members {
		if i > 0 {
			buf.WriteByte(',')
		}
		mLen2 := 0
		if member != nil {
			mLen2 = sequence.Len(member)
		}
		switch mLen2 {
		case 1:
			buf.WriteString(serializeItemAdaptive(member.Get(0), charMap))
		case 0:
			buf.WriteString("()")
		default:
			buf.WriteByte('(')
			for j := range mLen2 {
				if j > 0 {
					buf.WriteByte(',')
				}
				buf.WriteString(serializeItemAdaptive(member.Get(j), charMap))
			}
			buf.WriteByte(')')
		}
	}
	buf.WriteByte(']')
	return buf.String()
}

// nodeStringValue returns the string value of a node.
func nodeStringValue(n helium.Node) string {
	if n == nil {
		return ""
	}
	return string(n.Content())
}

// resolveNormForm returns the norm.Form for the given normalization form name.
// Returns (form, true) on success or (0, false) for unknown/NONE forms.
func resolveNormForm(form string) (norm.Form, bool) {
	switch form {
	case "NFC", "FULLY-NORMALIZED":
		return norm.NFC, true
	case "NFD":
		return norm.NFD, true
	case "NFKC":
		return norm.NFKC, true
	case "NFKD":
		return norm.NFKD, true
	default:
		return 0, false
	}
}

// applyUnicodeNormalization applies the specified Unicode normalization form
// to text content and attribute values in serialized XML/HTML output, while
// leaving element/attribute names and markup untouched (per XSLT 3.0 spec).
func applyUnicodeNormalization(data []byte, form string) []byte {
	nf, ok := resolveNormForm(form)
	if !ok {
		return data
	}
	return normalizeXMLContent(data, nf)
}

// normalizeSentinelAware applies Unicode normalization while preserving
// sentinel-wrapped character map segments intact.  Segments delimited by
// \x00CMSTART\x00 ... \x00CMEND\x00 are extracted before normalization
// and re-inserted afterwards, then the sentinel markers are stripped.
func normalizeSentinelAware(data []byte, form string) []byte {
	nf, ok := resolveNormForm(form)
	if !ok {
		// Unknown form — just strip sentinels.
		s := strings.ReplaceAll(string(data), "\x00CMSTART\x00", "")
		return []byte(strings.ReplaceAll(s, "\x00CMEND\x00", ""))
	}

	// Split on sentinels, normalize non-sentinel parts, recombine.
	s := string(data)
	var out strings.Builder
	out.Grow(len(s))
	for {
		startIdx := strings.Index(s, "\x00CMSTART\x00")
		if startIdx < 0 {
			// No more sentinels — normalize the rest.
			out.Write(normalizeXMLContent([]byte(s), nf))
			break
		}
		// Normalize the part before the sentinel.
		out.Write(normalizeXMLContent([]byte(s[:startIdx]), nf))
		s = s[startIdx+len("\x00CMSTART\x00"):]
		endIdx := strings.Index(s, "\x00CMEND\x00")
		if endIdx < 0 {
			// Malformed — write remainder as-is.
			out.WriteString(s)
			break
		}
		// Write the char-map segment un-normalized.
		out.WriteString(s[:endIdx])
		s = s[endIdx+len("\x00CMEND\x00"):]
	}
	return []byte(out.String())
}

// normalizeXMLContent applies Unicode normalization to text content and
// attribute values in serialized XML, preserving element/attribute names
// and other markup verbatim.
func normalizeXMLContent(data []byte, nf norm.Form) []byte {
	var out bytes.Buffer
	out.Grow(len(data))
	i := 0
	for i < len(data) {
		if data[i] == '<' {
			// Inside a tag — copy the tag verbatim but normalize attribute values.
			j := i + 1
			if j < len(data) && data[j] == '!' {
				// Comment (<!-- ... -->) or CDATA (<![CDATA[ ... ]]>)
				if j+1 < len(data) && data[j+1] == '-' {
					// Comment: copy verbatim until -->
					end := bytes.Index(data[i:], []byte("-->"))
					if end < 0 {
						out.Write(data[i:])
						return out.Bytes()
					}
					out.Write(data[i : i+end+3])
					i += end + 3
					continue
				}
				if j+7 < len(data) && string(data[j:j+7]) == "[CDATA[" {
					// CDATA: normalize content inside
					end := bytes.Index(data[i:], []byte("]]>"))
					if end < 0 {
						out.Write(data[i:])
						return out.Bytes()
					}
					cdataStart := i + 9 // after <![CDATA[
					cdataEnd := i + end
					out.Write(data[i:cdataStart])
					out.Write(nf.Bytes(data[cdataStart:cdataEnd]))
					out.Write([]byte("]]>"))
					i += end + 3
					continue
				}
			}
			if j < len(data) && data[j] == '?' {
				// Processing instruction: copy verbatim
				end := bytes.Index(data[i:], []byte("?>"))
				if end < 0 {
					out.Write(data[i:])
					return out.Bytes()
				}
				out.Write(data[i : i+end+2])
				i += end + 2
				continue
			}
			// Regular tag: copy tag name verbatim, normalize attribute values
			normalizeTag(&out, data, &i, nf)
			continue
		}
		// Text content outside tags — normalize it
		j := bytes.IndexByte(data[i:], '<')
		if j < 0 {
			out.Write(nf.Bytes(data[i:]))
			i = len(data)
		} else {
			out.Write(nf.Bytes(data[i : i+j]))
			i += j
		}
	}
	return out.Bytes()
}

// normalizeTag copies an XML tag, normalizing only attribute values.
func normalizeTag(out *bytes.Buffer, data []byte, pos *int, nf norm.Form) {
	i := *pos
	out.WriteByte('<')
	i++ // skip '<'

	// Copy tag name (and optional '/' for closing tags) verbatim
	for i < len(data) && data[i] != '>' && data[i] != ' ' && data[i] != '\t' && data[i] != '\n' && data[i] != '\r' && data[i] != '/' {
		out.WriteByte(data[i])
		i++
	}

	// Process attributes and whitespace until '>'
	for i < len(data) && data[i] != '>' {
		if data[i] == '/' {
			out.WriteByte(data[i])
			i++
			continue
		}
		if data[i] == ' ' || data[i] == '\t' || data[i] == '\n' || data[i] == '\r' {
			// Whitespace — copy verbatim
			out.WriteByte(data[i])
			i++
			continue
		}
		if data[i] == '"' || data[i] == '\'' {
			// Attribute value — normalize content
			quote := data[i]
			out.WriteByte(quote)
			i++
			start := i
			for i < len(data) && data[i] != quote {
				i++
			}
			out.Write(nf.Bytes(data[start:i]))
			if i < len(data) {
				out.WriteByte(quote)
				i++
			}
			continue
		}
		// Attribute name or '=' — copy verbatim
		out.WriteByte(data[i])
		i++
	}
	if i < len(data) {
		out.WriteByte('>') // closing '>'
		i++
	}
	*pos = i
}

// transcodeToUTF16 converts UTF-8 bytes to UTF-16 big-endian (without BOM;
// the BOM is emitted separately by the caller).
func transcodeToUTF16(w io.Writer, utf8Data []byte) error {
	enc := xtunicode.UTF16(xtunicode.BigEndian, xtunicode.IgnoreBOM)
	encoded, err := enc.NewEncoder().Bytes(utf8Data)
	if err != nil {
		_, werr := w.Write(utf8Data)
		return werr
	}
	_, werr := w.Write(encoded)
	return werr
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
		if err := doc.XML(&buf, writer); err != nil {
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
	return doc.XML(w, writer)
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

func serializeHTML(w io.Writer, doc *helium.Document, outDef *OutputDef) error {
	// Determine DOCTYPE handling.
	hasDoctypeAttrs := outDef.DoctypePublic != "" || outDef.DoctypeSystem != ""
	// Use explicit HTMLVersion for DOCTYPE/structural decisions.
	isHTML5 := isHTMLVersion5(outDef.HTMLVersion)
	// Use effective version (with XSLT 3.0 default=5) for character escaping.
	escapeCtrl := effectiveHTMLVersion5(outDef)

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
			if err := hw.WriteNode(w, child); err != nil {
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
	return hw.WriteDoc(w, doc)
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
		if root != nil && strings.EqualFold(string(root.LocalName()), "html") &&
			string(root.URI()) == lexicon.NamespaceXHTML {
			dtdName := string(root.LocalName())
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
		if root != nil && string(root.URI()) == lexicon.NamespaceXHTML {
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
		if string(elem.URI()) != lexicon.NamespaceXHTML {
			return nil
		}
		if string(elem.Prefix()) == "" {
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

		oldPrefix := string(elem.Prefix())
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
		uri := string(elem.URI())
		if uri != lexicon.NamespaceSVG && uri != lexicon.NamespaceMathML {
			return nil
		}
		if string(elem.Prefix()) == "" {
			return nil // already unprefixed
		}
		oldPrefix := string(elem.Prefix())
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

// applyCharMap applies a character map to a serialized string, replacing
// each mapped character with its replacement string.
func applyCharMap(s string, charMap map[rune]string) string {
	var out strings.Builder
	out.Grow(len(s))
	for _, r := range s {
		if repl, ok := charMap[r]; ok {
			out.WriteString(repl)
		} else {
			out.WriteRune(r)
		}
	}
	return out.String()
}

// applyCharMapJSON applies a character map to JSON-serialized output.
// JSON escape sequences (e.g., \/) are recognized: if the unescaped
// character is in the character map, the entire escape sequence is
// replaced with the map value.
func applyCharMapJSON(s string, charMap map[rune]string) string {
	var out strings.Builder
	out.Grow(len(s))
	i := 0
	for i < len(s) {
		if s[i] == '\\' && i+1 < len(s) {
			next := s[i+1]
			// Map JSON escape sequences to their unescaped character
			var unescaped rune
			switch next {
			case '/':
				unescaped = '/'
			case 'n':
				unescaped = '\n'
			case 'r':
				unescaped = '\r'
			case 't':
				unescaped = '\t'
			case 'b':
				unescaped = '\b'
			case 'f':
				unescaped = '\f'
			case '"':
				unescaped = '"'
			case '\\':
				unescaped = '\\'
			default:
				out.WriteByte(s[i])
				i++
				continue
			}
			if repl, ok := charMap[unescaped]; ok {
				out.WriteString(repl)
				i += 2
				continue
			}
			out.WriteByte(s[i])
			i++
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if repl, ok := charMap[r]; ok {
			out.WriteString(repl)
		} else {
			out.WriteRune(r)
		}
		i += size
	}
	return out.String()
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
	_ = helium.Walk(doc, helium.NodeWalkerFunc(func(n helium.Node) error {
		if n.Type() == helium.ProcessingInstructionNode && string(n.Name()) == "disable-output-escaping" {
			found = true
		}
		return nil
	}))
	return found
}

// inCDATAElement checks if the parent node is an element whose name matches
// one of the cdata-section-elements. Names can be local names (e.g., "item2")
// or Clark notation (e.g., "{http://ns}item2").
func inCDATAElement(parent helium.Node, cdataElems map[string]struct{}) bool {
	if len(cdataElems) == 0 {
		return false
	}
	elem, ok := parent.(*helium.Element)
	if !ok {
		return false
	}
	// Check Clark notation: {uri}local
	clark := helium.ClarkName(string(elem.URI()), string(elem.LocalName()))
	if _, ok := cdataElems[clark]; ok {
		return true
	}
	// Check QName (prefix:local or just local)
	name := elem.Name()
	if _, ok := cdataElems[name]; ok {
		return true
	}
	// Check local name only (for unprefixed elements)
	local := string(elem.LocalName())
	if _, ok := cdataElems[local]; ok {
		return true
	}
	return false
}

// writeCDATAWithEncoding writes text inside a CDATA section, splitting it
// when the text contains characters that cannot be represented in the target
// encoding. Non-representable characters are emitted as character references
// between CDATA sections. The stream.Writer.WriteCDATA method already handles
// splitting ]]> sequences.
func writeCDATAWithEncoding(sw *stream.Writer, text, encoding, normForm string) error {
	if !needsCDATASplit(encoding) {
		return sw.WriteCDATA(text)
	}
	// Apply Unicode normalization before CDATA splitting so that
	// decomposed characters are split at the correct boundaries.
	// For example, NFD of ç (U+00E7) is c (U+0063) + combining cedilla
	// (U+0327); 'c' is representable in US-ASCII and stays in CDATA,
	// while U+0327 must be emitted as a character reference.
	if nf, ok := resolveNormForm(normForm); ok {
		text = string(nf.Bytes([]byte(text)))
	}
	// Split text into runs of representable and non-representable characters.
	var buf strings.Builder
	for _, r := range text {
		if canRepresentInEncoding(r, encoding) {
			buf.WriteRune(r)
			continue
		}
		// Flush pending representable text as CDATA
		if buf.Len() > 0 {
			if err := sw.WriteCDATA(buf.String()); err != nil {
				return err
			}
			buf.Reset()
		}
		// Write non-representable char as character reference (outside CDATA)
		if err := sw.WriteRaw(fmt.Sprintf("&#x%X;", r)); err != nil {
			return err
		}
	}
	if buf.Len() > 0 {
		return sw.WriteCDATA(buf.String())
	}
	return nil
}

// needsCDATASplit returns true if the encoding might require CDATA splitting
// for non-representable characters.
func needsCDATASplit(encoding string) bool {
	switch encoding {
	case "", "utf-8", "utf8", "utf-16", "utf16":
		return false
	default:
		return true
	}
}

// canRepresentInEncoding returns true if rune r can be represented in the
// given encoding without a character reference.
func canRepresentInEncoding(r rune, encoding string) bool {
	switch encoding {
	case "us-ascii", "ascii":
		return r < 128
	case "iso-8859-1", "latin1", "latin-1":
		return r < 256
	default:
		// For unknown encodings, assume ASCII-safe
		return r < 128
	}
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
	// Find the <head> element (case-insensitive, using local name for namespace support).
	var head *helium.Element
	for child := range helium.Children(root) {
		if e, ok := child.(*helium.Element); ok && strings.EqualFold(string(e.LocalName()), "head") {
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
		if e, ok := child.(*helium.Element); ok && strings.EqualFold(string(e.LocalName()), "meta") {
			for _, attr := range e.Attributes() {
				if strings.EqualFold(attr.Name(), "http-equiv") && strings.EqualFold(attr.Value(), "Content-Type") {
					// Update the existing content attribute
					e.SetLiteralAttribute("content", contentValue)
					return
				}
			}
		}
	}
	// Create and insert the meta element.
	meta := doc.CreateElement("meta")
	// If the head element is in a namespace, put the meta element in the same namespace.
	if headURI := string(head.URI()); headURI != "" {
		_ = meta.SetActiveNamespace(string(head.Prefix()), headURI)
	}
	meta.SetLiteralAttribute("http-equiv", "Content-Type")
	meta.SetLiteralAttribute("content", contentValue)
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
