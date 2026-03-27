package xslt3

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/stream"
	"golang.org/x/text/encoding/htmlindex"
	xtunicode "golang.org/x/text/encoding/unicode"
	"golang.org/x/text/unicode/norm"
)

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
