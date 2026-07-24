package xslt3

import (
	"bytes"
	"fmt"
	"io"
	"maps"
	"strings"
	"unicode/utf8"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/lestrrat-go/helium/stream"
	"golang.org/x/text/encoding/htmlindex"
	xtunicode "golang.org/x/text/encoding/unicode"
	"golang.org/x/text/unicode/norm"
)

const normalizationFormNFC = "NFC"

// resolveNormForm returns the norm.Form for the given normalization form name.
// Returns (form, true) on success or (0, false) for unknown/NONE forms.
func resolveNormForm(form string) (norm.Form, bool) {
	switch form {
	case normalizationFormNFC, lexicon.NormFullyNormalized:
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
				if bytes.HasPrefix(data[i:], []byte("<![CDATA[")) {
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
					out.WriteString("]]>")
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
		return writeFullBytes(w, utf8Data)
	}
	return writeFullBytes(w, encoded)
}

// transcodeToEncoding converts UTF-8 bytes to the target encoding,
// replacing characters that cannot be represented with XML character references.
func transcodeToEncoding(w io.Writer, utf8Data []byte, encName string) error {
	codec, err := htmlindex.Get(encName)
	if err != nil {
		// Unknown encoding — fall back to writing UTF-8
		return writeFullBytes(w, utf8Data)
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
			if werr := writeFullString(w, ref); werr != nil {
				return werr
			}
			// Reset encoder state after error
			encoder = codec.NewEncoder()
		} else {
			if werr := writeFullBytes(w, encoded); werr != nil {
				return werr
			}
		}
		utf8Data = utf8Data[size:]
	}
	return nil
}

// normalizeText applies the requested normalization form to text. An absent
// or unsupported form leaves the text unchanged; parameter validation reports
// unsupported forms before serialization begins.
func normalizeText(text, form string) string {
	nf, ok := resolveNormForm(form)
	if !ok {
		return text
	}
	return nf.String(text)
}

// normalizeRawXMLContent preserves markup in raw DOE output while normalizing
// its character content, matching the normal XML post-processing path.
func normalizeRawXMLContent(text, form string) string {
	nf, ok := resolveNormForm(form)
	if !ok {
		return text
	}
	return string(normalizeXMLContent([]byte(text), nf))
}

// applyCharacterMapWithNormalization normalizes only runs that do not produce
// a character-map replacement. The replacement itself remains byte-for-byte
// intact, as required by the serialization specification.
func applyCharacterMapWithNormalization(text string, charMap map[rune]string, form string) string {
	if len(charMap) == 0 {
		return normalizeText(text, form)
	}

	var out strings.Builder
	out.Grow(len(text))
	var unmapped strings.Builder
	flushUnmapped := func() {
		if unmapped.Len() == 0 {
			return
		}
		out.WriteString(normalizeText(unmapped.String(), form))
		unmapped.Reset()
	}
	for _, r := range text {
		if repl, ok := charMap[r]; ok {
			flushUnmapped()
			out.WriteString(repl)
			continue
		}
		unmapped.WriteRune(r)
	}
	flushUnmapped()
	return out.String()
}

// applyCharMapJSON applies a character map to JSON-serialized output.
// JSON escape sequences (e.g., \/) are recognized: if the unescaped
// character is in the character map, the entire escape sequence is
// replaced with the map value.
func applyCharMapJSON(s string, charMap map[rune]string, normalizationForm string) string {
	var out strings.Builder
	out.Grow(len(s))
	var unmapped strings.Builder
	flushUnmapped := func() {
		if unmapped.Len() == 0 {
			return
		}
		out.WriteString(normalizeText(unmapped.String(), normalizationForm))
		unmapped.Reset()
	}
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
				unmapped.WriteByte(s[i])
				i++
				continue
			}
			if repl, ok := charMap[unescaped]; ok {
				flushUnmapped()
				out.WriteString(repl)
				i += 2
				continue
			}
			unmapped.WriteString(s[i : i+2])
			i += 2
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if repl, ok := charMap[r]; ok {
			flushUnmapped()
			out.WriteString(repl)
		} else {
			unmapped.WriteRune(r)
		}
		i += size
	}
	flushUnmapped()
	return out.String()
}

// applyCharMapToHTMLText applies a character map to serialized HTML output,
// applying to text content and non-URI attribute values, but skipping
// URI attributes (href, src, etc.) per the XSLT serialization spec.
func applyCharMapToHTMLText(html string, charMap map[rune]string, normalizationForm string) string {
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
			out.WriteString(applyCharMapToHTMLTag(tag, charMap, normalizationForm))
			i += tagEnd + 1
			continue
		}
		// Text content — normalize unmapped runs and apply replacements.
		tagStart := strings.IndexByte(html[i:], '<')
		if tagStart < 0 {
			out.WriteString(applyCharacterMapWithNormalization(html[i:], charMap, normalizationForm))
			break
		}
		out.WriteString(applyCharacterMapWithNormalization(html[i:i+tagStart], charMap, normalizationForm))
		i += tagStart
	}
	return out.String()
}

// applyCharMapToHTMLTag applies character map to attribute values within an
// HTML tag, skipping URI attributes.
func applyCharMapToHTMLTag(tag string, charMap map[rune]string, normalizationForm string) string {
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
				out.WriteString(normalizeText(attrVal, normalizationForm))
			} else {
				out.WriteString(applyCharacterMapWithNormalization(attrVal, charMap, normalizationForm))
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
	// The result tree is freshly built via guarded AddChild, so it is acyclic
	// and Walk cannot return ErrWalkCycle here; the error is safely ignored.
	_ = helium.Walk(doc, helium.NodeWalkerFunc(func(n helium.Node) error {
		if n.Type() == helium.ProcessingInstructionNode && n.Name() == "disable-output-escaping" {
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
	clark := helium.ClarkName(elem.URI(), elem.LocalName())
	if _, ok := cdataElems[clark]; ok {
		return true
	}
	// Check QName (prefix:local or just local)
	name := elem.Name()
	if _, ok := cdataElems[name]; ok {
		return true
	}
	// Check local name only (for unprefixed elements)
	local := elem.LocalName()
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
	// Apply Unicode normalization before CDATA serialization so that the
	// content follows the same rule as ordinary text, including UTF-8 output.
	text = normalizeText(text, normForm)
	if !needsCDATASplit(encoding) {
		return sw.WriteCDATA(text)
	}
	// Normalization runs before CDATA splitting so that decomposed characters
	// are split at the correct boundaries.
	// For example, NFD of ç (U+00E7) is c (U+0063) + combining cedilla
	// (U+0327); 'c' is representable in US-ASCII and stays in CDATA,
	// while U+0327 must be emitted as a character reference.
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
	case "", "utf-8", lexicon.EncodingUTF8Alt, "utf-16", "utf16":
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
		maps.Copy(merged, cm.Mappings)
	}
	for _, name := range names {
		resolve(name)
	}
	if len(merged) == 0 {
		return nil
	}
	return merged
}
