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

// SerializeItems writes a sequence of items (maps, arrays, atomics, nodes)
// using the specified output definition's method (json or adaptive).
// This is used for result-documents with method="json" or method="adaptive".
func SerializeItems(w io.Writer, items xpath3.Sequence, doc *helium.Document, outDef *OutputDef) error {
	if outDef == nil {
		outDef = defaultOutputDef()
	}
	switch outDef.Method {
	case "json":
		if len(outDef.ResolvedCharMap) > 0 {
			var buf strings.Builder
			if err := serializeJSONItems(&buf, items, doc); err != nil {
				return err
			}
			_, err := io.WriteString(w, applyCharMap(buf.String(), outDef.ResolvedCharMap))
			return err
		}
		return serializeJSONItems(w, items, doc)
	case "adaptive":
		return serializeAdaptiveItems(w, items, doc, outDef.ResolvedCharMap)
	default:
		return SerializeResult(w, doc, outDef)
	}
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
	if !outDef.MethodExplicit && outDef.Method == "xml" {
		if root := doc.DocumentElement(); root != nil {
			if strings.EqualFold(root.Name(), "html") && root.URI() == "" {
				// Root is "html" in no namespace → HTML output method.
				outDef.Method = "html"
				outDef.OmitDeclaration = true
			} else if strings.EqualFold(string(root.LocalName()), "html") && string(root.URI()) == xhtmlNS {
				// Root is "html" in XHTML namespace → XHTML output method.
				outDef.Method = "xhtml"
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

	// Emit UTF-8 BOM if requested.
	if outDef.ByteOrderMark {
		if _, werr := w.Write([]byte{0xEF, 0xBB, 0xBF}); werr != nil {
			return werr
		}
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
	case "json":
		if len(charMap) > 0 {
			var jsonBuf strings.Builder
			if jerr := serializeJSONItems(&jsonBuf, nil, doc); jerr != nil {
				err = jerr
			} else {
				_, err = io.WriteString(target, applyCharMap(jsonBuf.String(), charMap))
			}
		} else {
			err = serializeJSONItems(target, nil, doc)
		}
	case "adaptive":
		err = serializeAdaptiveItems(target, nil, doc, charMap)
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

// serializeJSONItems serializes a sequence of items as JSON output.
// Per XSLT 3.0 §26: the sequence is serialized as a single JSON value.
func serializeJSONItems(w io.Writer, items xpath3.Sequence, doc *helium.Document) error {
	if len(items) == 0 && doc != nil {
		// No captured items: serialize DOM content as text for JSON
		return serializeAdaptiveItems(w, items, doc)
	}
	if len(items) == 1 {
		s, err := serializeItemJSON(items[0])
		if err != nil {
			return err
		}
		_, err = io.WriteString(w, s)
		return err
	}
	// Multiple items: serialize as JSON array
	members := make([]xpath3.Sequence, len(items))
	for i, item := range items {
		members[i] = xpath3.Sequence{item}
	}
	s, err := serializeItemJSON(xpath3.NewArray(members))
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, s)
	return err
}

// serializeAdaptiveItems serializes a sequence of items using the adaptive
// serialization method. Each item is serialized according to its type.
func serializeAdaptiveItems(w io.Writer, items xpath3.Sequence, doc *helium.Document, charMaps ...map[rune]string) error {
	if len(items) == 0 && doc != nil {
		// Fallback: serialize DOM as XML
		var cm map[rune]string
		if len(charMaps) > 0 {
			cm = charMaps[0]
		}
		return serializeXML(w, doc, defaultOutputDef(), cm)
	}
	for i, item := range items {
		if i > 0 {
			if _, err := io.WriteString(w, "\n"); err != nil {
				return err
			}
		}
		s := serializeItemAdaptive(item)
		if _, err := io.WriteString(w, s); err != nil {
			return err
		}
	}
	return nil
}

// serializeItemJSON serializes a single XDM item as JSON.
func serializeItemJSON(item xpath3.Item) (string, error) {
	switch v := item.(type) {
	case xpath3.MapItem:
		return serializeMapJSON(v)
	case xpath3.ArrayItem:
		return serializeArrayJSON(v)
	case xpath3.NodeItem:
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
func serializeMapJSON(m xpath3.MapItem) (string, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	first := true
	var serErr error
	m.ForEach(func(k xpath3.AtomicValue, v xpath3.Sequence) error {
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
		if len(v) == 1 {
			s, err := serializeItemJSON(v[0])
			if err != nil {
				serErr = err
				return err
			}
			buf.WriteString(s)
		} else if len(v) == 0 {
			buf.WriteString("null")
		} else {
			members := make([]xpath3.Sequence, len(v))
			for i, vi := range v {
				members[i] = xpath3.Sequence{vi}
			}
			s, err := serializeItemJSON(xpath3.NewArray(members))
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
func serializeArrayJSON(a xpath3.ArrayItem) (string, error) {
	var buf bytes.Buffer
	buf.WriteByte('[')
	members := a.Members()
	for i, member := range members {
		if i > 0 {
			buf.WriteByte(',')
		}
		if len(member) == 1 {
			s, err := serializeItemJSON(member[0])
			if err != nil {
				return "", err
			}
			buf.WriteString(s)
		} else if len(member) == 0 {
			buf.WriteString("null")
		} else {
			// Multi-item member: serialize as array
			submembers := make([]xpath3.Sequence, len(member))
			for j, mi := range member {
				submembers[j] = xpath3.Sequence{mi}
			}
			s, err := serializeItemJSON(xpath3.NewArray(submembers))
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
		default:
			if r < 0x20 {
				buf.WriteString(fmt.Sprintf("\\u%04x", r))
			} else {
				buf.WriteRune(r)
			}
		}
	}
	buf.WriteByte('"')
	return buf.String()
}

// serializeItemAdaptive serializes a single item using the adaptive method.
func serializeItemAdaptive(item xpath3.Item) string {
	switch v := item.(type) {
	case xpath3.MapItem:
		return serializeMapAdaptive(v)
	case xpath3.ArrayItem:
		return serializeArrayAdaptive(v)
	case xpath3.NodeItem:
		var buf bytes.Buffer
		if elem, ok := v.Node.(*helium.Element); ok {
			elem.XML(&buf, helium.WithNoDecl())
		} else if doc, ok := v.Node.(*helium.Document); ok {
			doc.XML(&buf, helium.WithNoDecl())
		} else {
			buf.WriteString(string(v.Node.Content()))
		}
		return buf.String()
	case xpath3.AtomicValue:
		s, _ := xpath3.AtomicToString(v)
		return s
	default:
		if av, ok := item.(xpath3.AtomicValue); ok {
			s, _ := xpath3.AtomicToString(av)
			return s
		}
		return fmt.Sprintf("%v", item)
	}
}

// serializeMapAdaptive serializes a map using adaptive serialization.
func serializeMapAdaptive(m xpath3.MapItem) string {
	var buf bytes.Buffer
	buf.WriteString("map{")
	first := true
	m.ForEach(func(k xpath3.AtomicValue, v xpath3.Sequence) error {
		if !first {
			buf.WriteByte(',')
		}
		first = false
		ks, _ := xpath3.AtomicToString(k)
		buf.WriteString(jsonEscapeString(ks))
		buf.WriteByte(':')
		if len(v) == 1 {
			buf.WriteString(serializeItemAdaptive(v[0]))
		} else if len(v) == 0 {
			buf.WriteString("()")
		} else {
			buf.WriteByte('(')
			for i, vi := range v {
				if i > 0 {
					buf.WriteByte(',')
				}
				buf.WriteString(serializeItemAdaptive(vi))
			}
			buf.WriteByte(')')
		}
		return nil
	})
	buf.WriteByte('}')
	return buf.String()
}

// serializeArrayAdaptive serializes an array using adaptive serialization.
func serializeArrayAdaptive(a xpath3.ArrayItem) string {
	var buf bytes.Buffer
	buf.WriteByte('[')
	members := a.Members()
	for i, member := range members {
		if i > 0 {
			buf.WriteByte(',')
		}
		if len(member) == 1 {
			buf.WriteString(serializeItemAdaptive(member[0]))
		} else if len(member) == 0 {
			buf.WriteString("()")
		} else {
			buf.WriteByte('(')
			for j, vi := range member {
				if j > 0 {
					buf.WriteByte(',')
				}
				buf.WriteString(serializeItemAdaptive(vi))
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

// validateSerializationParams checks serialization parameters for errors
// per the XSLT 3.0 serialization spec.
func validateSerializationParams(outDef *OutputDef, doc *helium.Document) error {
	method := outDef.Method

	// SEPM0004: standalone != "omit" with multiple element children of root
	if outDef.Standalone == "yes" || outDef.Standalone == "no" {
		if method == "xml" || method == "xhtml" {
			elemCount := countRootElements(doc)
			if elemCount > 1 {
				return dynamicError(errCodeSEPM0004,
					"standalone=%q specified but result has %d root elements", outDef.Standalone, elemCount)
			}
		}
	}

	// SEPM0004: doctype-system with multiple element children of root
	if outDef.DoctypeSystem != "" {
		if method == "xml" || method == "xhtml" {
			elemCount := countRootElements(doc)
			if elemCount > 1 {
				return dynamicError(errCodeSEPM0004,
					"doctype-system specified but result has %d root elements", elemCount)
			}
		}
	}

	// SEPM0009: omit-xml-declaration="yes" conflicts with standalone or doctype-system
	// Only applicable for xml/xhtml methods — text/html/json don't have XML declarations.
	if outDef.OmitDeclaration && (method == "xml" || method == "xhtml") {
		if outDef.Standalone == "yes" || outDef.Standalone == "no" {
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
	if method == "html" && outDef.MethodExplicit && outDef.Version != "" {
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
	// Check both html-version and version attributes for HTML5 detection.
	htmlVer := outDef.HTMLVersion
	if htmlVer == "" {
		htmlVer = outDef.Version
	}
	if method == "html" && !isHTMLVersion5(htmlVer) {
		if err := checkHTMLInvalidChars(doc); err != nil {
			return err
		}
	}

	// SERE0015: ">" in PI content for HTML output
	if method == "html" {
		if err := checkHTMLPIContent(doc); err != nil {
			return err
		}
	}

	return nil
}

// countRootElements counts the number of element children of the document root.
func countRootElements(doc *helium.Document) int {
	count := 0
	for child := doc.FirstChild(); child != nil; child = child.NextSibling() {
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
	_ = helium.Walk(doc, func(n helium.Node) error {
		if n.Type() == helium.TextNode || n.Type() == helium.CDATASectionNode {
			content := string(n.Content())
			for _, r := range content {
				if unicode.In(r, unicode.Mn, unicode.Mc, unicode.Me) {
					firstErr = dynamicError(errCodeSERE0012,
						"fully-normalized output begins with combining character U+%04X", r)
					return firstErr
				}
				break // only check first character
			}
		}
		return nil
	})
	return firstErr
}

// checkHTMLInvalidChars checks for characters in the #x7F-#x9F range in
// HTML text content (SERE0014).
func checkHTMLInvalidChars(doc *helium.Document) error {
	var firstErr error
	_ = helium.Walk(doc, func(n helium.Node) error {
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
	})
	return firstErr
}

// checkHTMLPIContent checks that no PI in the result tree contains ">".
func checkHTMLPIContent(doc *helium.Document) error {
	var err error
	_ = helium.Walk(doc, func(n helium.Node) error {
		if n.Type() == helium.ProcessingInstructionNode {
			content := string(n.Content())
			if strings.Contains(content, ">") {
				err = dynamicError(errCodeSERE0015,
					"processing instruction content contains '>' in HTML output")
				return err
			}
		}
		return nil
	})
	return err
}

func serializeXML(w io.Writer, doc *helium.Document, outDef *OutputDef, charMap map[rune]string) error {
	// For non-UTF-8 encodings, use the stream-based serializer which
	// always outputs UTF-8. The encoding conversion is handled by
	// serializeResult's transcoding layer.
	targetEnc := strings.ToLower(outDef.Encoding)
	isNonUTF8 := targetEnc != "" && targetEnc != "utf-8" && targetEnc != "utf8" && targetEnc != "utf-16" && targetEnc != "utf16"
	if len(charMap) > 0 || hasDOEMarkers(doc) || isNonUTF8 || len(outDef.CDATASections) > 0 || (outDef.Indent && len(outDef.SuppressIndentation) > 0) {
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

	err := serializeXMLNodeWithCharMap(sw, doc, charMap, cdataSet, enc, ictx)
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
		return "{" + uri + "}" + string(elem.LocalName())
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
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		children = append(children, child)
	}
	return children
}

func elemHasChildElements(elem *helium.Element) bool {
	for child := elem.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Type() == helium.ElementNode {
			return true
		}
	}
	return false
}

func serializeXMLNodeWithCharMap(sw *stream.Writer, n helium.Node, charMap map[rune]string, cdataElems map[string]struct{}, encoding string, ictx *xmlIndentCtx) error {
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
			if err := serializeXMLNodeWithCharMap(sw, elem, charMap, cdataElems, encoding, ictx); err != nil {
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
				if err := writeCDATAWithEncoding(sw, text, encoding); err != nil {
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
	isHTML5 := isHTMLVersion5(outDef.HTMLVersion)

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
	noEscapeURI := outDef.EscapeURIAttributes != nil && !*outDef.EscapeURIAttributes
	if isHTML5 && !hasDoctypeAttrs {
		nodeOpts := []htmlpkg.WriteOption{
			htmlpkg.WithNoFormat(),
			htmlpkg.WithPreserveCase(),
		}
		if noEscapeURI {
			nodeOpts = append(nodeOpts, htmlpkg.WithNoEscapeURIAttributes())
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
	if noEscapeURI {
		opts = append(opts, htmlpkg.WithNoEscapeURIAttributes())
	}
	return htmlpkg.WriteDoc(w, doc, opts...)
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
	if isHTML5 && outDef.DoctypeSystem == "" {
		// Remove existing DTD and replace with HTML5 DOCTYPE.
		// Use the root element's local name to preserve case (e.g. "HTML" vs "html").
		dtdName := "html"
		if root := doc.DocumentElement(); root != nil {
			dtdName = string(root.LocalName())
		}
		if dtd := doc.IntSubset(); dtd != nil {
			helium.UnlinkNode(dtd)
		}
		_, _ = doc.CreateInternalSubset(dtdName, "", "")
	}

	// Insert <meta http-equiv="Content-Type"> in <head>
	if outDef.IncludeContentType == nil || *outDef.IncludeContentType {
		insertHTMLMeta(doc, outDef)
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

const xhtmlNS = "http://www.w3.org/1999/xhtml"

// normalizeXHTMLNamespace walks the document and converts prefixed XHTML
// namespace elements to use the default namespace (unprefixed), as required
// by the XHTML output method. The default namespace declaration is added
// only to the root element; descendants inherit it.
func normalizeXHTMLNamespace(doc *helium.Document) {
	// First pass: find all XHTML-prefixed elements and track prefixes to remove.
	// Also find/create a shared default namespace node for XHTML.
	var sharedNS *helium.Namespace
	rootDone := false

	_ = helium.Walk(doc, func(n helium.Node) error {
		elem, ok := n.(*helium.Element)
		if !ok {
			return nil
		}
		if string(elem.URI()) != xhtmlNS {
			return nil
		}
		if string(elem.Prefix()) == "" {
			// Already using default namespace. Capture the NS node if we
			// haven't seen one yet.
			if sharedNS == nil {
				for _, ns := range elem.Namespaces() {
					if ns.Prefix() == "" && ns.URI() == xhtmlNS {
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
			_ = elem.DeclareNamespace("", xhtmlNS)
			// Find the namespace node we just created
			for _, ns := range elem.Namespaces() {
				if ns.Prefix() == "" && ns.URI() == xhtmlNS {
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
	})
}

const (
	svgNS     = "http://www.w3.org/2000/svg"
	mathmlNS  = "http://www.w3.org/1998/Math/MathML"
)

// normalizeForeignNamespaces converts prefixed SVG and MathML elements to use
// their default namespace (unprefixed) for HTML5 XHTML output. Each element
// in the SVG or MathML namespace gets a default xmlns declaration for its own
// namespace, so it serializes as e.g. <svg xmlns="..."> instead of <s:svg>.
func normalizeForeignNamespaces(doc *helium.Document) {
	_ = helium.Walk(doc, func(n helium.Node) error {
		elem, ok := n.(*helium.Element)
		if !ok {
			return nil
		}
		uri := string(elem.URI())
		if uri != svgNS && uri != mathmlNS {
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
	})
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
	clark := "{" + string(elem.URI()) + "}" + string(elem.LocalName())
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
func writeCDATAWithEncoding(sw *stream.Writer, text, encoding string) error {
	if !needsCDATASplit(encoding) {
		return sw.WriteCDATA(text)
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
	for child := root.FirstChild(); child != nil; child = child.NextSibling() {
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
	for child := head.FirstChild(); child != nil; child = child.NextSibling() {
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
	meta, err := doc.CreateElement("meta")
	if err != nil {
		return
	}
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
