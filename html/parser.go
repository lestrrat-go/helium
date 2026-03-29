package html

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

// insertMode tracks the parser's insertion context.
type insertMode int

const (
	insertInitial insertMode = 1
	insertInHead  insertMode = 3
	insertInBody  insertMode = 10
)

// parser is the HTML parser. It drives the tokenizer and fires SAX events.
type parser struct {
	input []byte
	pos   int
	line  int
	col   int

	sax       SAXHandler
	nameStack []string // open element name stack
	mode      insertMode
	sawRoot   bool // true once the root <html> element has been opened

	// locator for SetDocumentLocator
	locator *parserLocator

	// depth tracks discarded misplaced structural elements (html/head/body)
	// so their corresponding end tags are also silently skipped.
	depth int

	// encodingError is set when invalid bytes were found in a UTF-8 declared
	// document and replaced with U+FFFD. The SAX error is emitted once when
	// the first characters event containing U+FFFD is encountered.
	encodingError     bool
	encodingErrorLine int // line of first invalid byte (1-indexed)
	encodingErrorCol  int // column of first invalid byte (1-indexed)

	// detectedEncoding records the original encoding detected for the input.
	// Empty means UTF-8 (no conversion was needed). "ISO-8859-1" means the
	// input was Latin-1/Windows-1252 and was converted to UTF-8 for parsing.
	detectedEncoding string

	cfg parseConfig
}

// parserLocator implements DocumentLocator.
type parserLocator struct {
	p *parser
}

func (l *parserLocator) LineNumber() int   { return l.p.line }
func (l *parserLocator) ColumnNumber() int { return l.p.col }

// GetPublicID returns the public identifier of the document being parsed (libxml2: xmlSAXLocator.getPublicId).
func (l *parserLocator) GetPublicID() string { return "" }

// GetSystemID returns the system identifier (URI/filename) of the document being parsed (libxml2: xmlSAXLocator.getSystemId).
func (l *parserLocator) GetSystemID() string { return "" }

func newParser(_ context.Context, input []byte, sax SAXHandler, cfg parseConfig) *parser {
	// Normalize \r\n → \n and standalone \r → \n (HTML spec line normalization)
	normalized := normalizeNewlines(input)

	var encodingErr bool
	var encErrLine, encErrCol int
	var detectedEnc string
	if !utf8.Valid(normalized) {
		if declaredCharsetIsUTF8(normalized) {
			raw := normalized
			var invBytes invalidByteInfo
			normalized, encodingErr = replaceInvalidUTF8(raw, &invBytes)
			if encodingErr {
				encErrLine, encErrCol = lineColFromOffset(raw, invBytes.offset)
			}
		} else {
			// Assume Latin-1/Windows-1252 encoding and convert to UTF-8,
			// matching libxml2's default behavior for non-UTF-8 documents.
			// Distinguish explicit charset=iso-8859-1 from auto-detected:
			// - "ISO-8859-1": strict output with &#N; for runes > 0xFF
			// - "Windows-1252": Win-1252 reverse mapping for output
			if declaredCharsetIsLatin1(normalized) {
				detectedEnc = "ISO-8859-1"
			} else {
				detectedEnc = "Windows-1252"
			}
			normalized = latin1ToUTF8(normalized)
		}
	}

	p := &parser{
		input:             normalized,
		pos:               0,
		line:              1,
		col:               1,
		sax:               sax,
		mode:              insertInitial,
		encodingError:     encodingErr,
		encodingErrorLine: encErrLine,
		encodingErrorCol:  encErrCol,
		detectedEncoding:  detectedEnc,
		cfg:               cfg,
	}
	p.locator = &parserLocator{p: p}
	return p
}

// normalizeNewlines replaces \r\n with \n and standalone \r with \n.
func normalizeNewlines(data []byte) []byte {
	// Fast path: no \r in input
	if !bytes.ContainsRune(data, '\r') {
		return data
	}
	out := make([]byte, 0, len(data))
	for i := 0; i < len(data); i++ {
		if data[i] == '\r' {
			out = append(out, '\n')
			if i+1 < len(data) && data[i+1] == '\n' {
				i++ // skip the \n after \r
			}
		} else {
			out = append(out, data[i])
		}
	}
	return out
}

// lineColFromOffset computes the 1-indexed line and column for a byte offset
// within newline-normalized data. The offset is the position of the target byte.
func lineColFromOffset(data []byte, offset int) (int, int) {
	line := 1
	col := 1
	for i := 0; i < offset && i < len(data); i++ {
		if data[i] == '\n' {
			line++
			col = 1
		} else {
			col++
		}
	}
	return line, col
}

// win1252ToUnicode maps Windows-1252 bytes 0x80-0x9F to Unicode codepoints.
// These bytes differ from ISO-8859-1 (which maps them to C1 control characters).
// libxml2 uses this mapping for HTML when no encoding is specified.
var win1252ToUnicode = [32]rune{
	0x20AC, 0x0081, 0x201A, 0x0192, 0x201E, 0x2026, 0x2020, 0x2021, // 80-87
	0x02C6, 0x2030, 0x0160, 0x2039, 0x0152, 0x008D, 0x017D, 0x008F, // 88-8F
	0x0090, 0x2018, 0x2019, 0x201C, 0x201D, 0x2022, 0x2013, 0x2014, // 90-97
	0x02DC, 0x2122, 0x0161, 0x203A, 0x0153, 0x009D, 0x017E, 0x0178, // 98-9F
}

// latin1ToUTF8 converts Latin-1/Windows-1252 encoded bytes to UTF-8.
// Bytes 0x80-0x9F are mapped using Windows-1252 (matching libxml2's behavior).
// Other bytes > 0x7F are treated as their Latin-1 Unicode equivalents.
func latin1ToUTF8(data []byte) []byte {
	var buf bytes.Buffer
	buf.Grow(len(data) * 2)
	for _, b := range data {
		if b < 0x80 {
			buf.WriteByte(b)
		} else if b >= 0x80 && b <= 0x9F {
			buf.WriteRune(win1252ToUnicode[b-0x80])
		} else {
			buf.WriteRune(rune(b))
		}
	}
	return buf.Bytes()
}

// invalidByteInfo records the position and raw bytes of the first invalid
// byte sequence found during UTF-8 validation.
type invalidByteInfo struct {
	offset   int // byte offset of first invalid byte in newline-normalized input
	rawBytes [4]byte
	nBytes   int // number of valid bytes in rawBytes (0..4)
}

// replaceInvalidUTF8 replaces invalid byte sequences with U+FFFD.
// If info is non-nil, populates it with details of the first invalid byte.
// Returns the cleaned data and whether any replacements were made.
func replaceInvalidUTF8(data []byte, info *invalidByteInfo) ([]byte, bool) {
	var buf bytes.Buffer
	buf.Grow(len(data))
	found := false
	for i := 0; i < len(data); {
		r, size := utf8.DecodeRune(data[i:])
		if r == utf8.RuneError && size <= 1 {
			buf.WriteRune('\uFFFD')
			if !found && info != nil {
				info.offset = i
				end := min(i+4, len(data))
				info.nBytes = copy(info.rawBytes[:], data[i:end])
			}
			found = true
			i++
		} else {
			buf.Write(data[i : i+size])
			i += size
		}
	}
	return buf.Bytes(), found
}

// declaredCharsetIsUTF8 scans the raw (possibly invalid) input bytes for a
// <meta charset="utf-8"> declaration, returning true if found.
func declaredCharsetIsUTF8(data []byte) bool {
	// Case-insensitive search for charset="utf-8" or charset=utf-8
	lower := bytes.ToLower(data)
	// Limit scan to the first 1024 bytes (charset should appear early)
	if len(lower) > 1024 {
		lower = lower[:1024]
	}
	return bytes.Contains(lower, []byte("charset=\"utf-8\"")) ||
		bytes.Contains(lower, []byte("charset=utf-8"))
}

// declaredCharsetIsLatin1 scans the raw input bytes for an explicit
// charset=iso-8859-1 declaration. This distinguishes documents that
// declare ISO-8859-1 from those that are just auto-detected as non-UTF-8.
func declaredCharsetIsLatin1(data []byte) bool {
	lower := bytes.ToLower(data)
	if len(lower) > 1024 {
		lower = lower[:1024]
	}
	return bytes.Contains(lower, []byte("charset=iso-8859-1")) ||
		bytes.Contains(lower, []byte("charset=\"iso-8859-1\""))
}

// peek returns the byte at the current position, or 0 if at end.
func (p *parser) peek() byte {
	if p.pos >= len(p.input) {
		return 0
	}
	return p.input[p.pos]
}

// peekAt returns the byte at pos+offset, or 0 if out of bounds.
func (p *parser) peekAt(offset int) byte {
	idx := p.pos + offset
	if idx >= len(p.input) || idx < 0 {
		return 0
	}
	return p.input[idx]
}

// advance moves forward by n bytes, updating line/col tracking.
func (p *parser) advance(n int) {
	for range n {
		if p.pos >= len(p.input) {
			break
		}
		if p.input[p.pos] == '\n' {
			p.line++
			p.col = 1
		} else {
			p.col++
		}
		p.pos++
	}
}

// remaining returns the remaining input from current position.
func (p *parser) remaining() []byte {
	if p.pos >= len(p.input) {
		return nil
	}
	return p.input[p.pos:]
}

// atEnd returns true if the parser has consumed all input.
func (p *parser) atEnd() bool {
	return p.pos >= len(p.input)
}

// currentName returns the name on top of the element stack.
func (p *parser) currentName() string {
	if len(p.nameStack) == 0 {
		return ""
	}
	return p.nameStack[len(p.nameStack)-1]
}

// pushName pushes an element name onto the stack and tracks insert mode.
func (p *parser) pushName(name string) {
	if name == "html" {
		p.sawRoot = true
	}
	if p.mode < insertInHead && name == "head" { //nolint:goconst
		p.mode = insertInHead
	}
	if p.mode < insertInBody && name == "body" { //nolint:goconst
		p.mode = insertInBody
	}
	p.nameStack = append(p.nameStack, name)
}

// popName pops the top element name from the stack.
func (p *parser) popName() string {
	if len(p.nameStack) == 0 {
		return ""
	}
	name := p.nameStack[len(p.nameStack)-1]
	p.nameStack = p.nameStack[:len(p.nameStack)-1]
	return name
}

// hasOnStack checks if the given element name is on the open element stack.
func (p *parser) hasOnStack(name string) bool {
	return slices.Contains(p.nameStack, name)
}

// isMisplacedStructural checks whether a structural element (html/head/body)
// is misplaced and should be silently discarded. Matches libxml2's
// HTMLparser.c misplaced-element detection.
func (p *parser) isMisplacedStructural(name string) bool {
	switch name {
	case "html":
		return len(p.nameStack) > 0
	case "head":
		return len(p.nameStack) != 1
	case "body":
		return p.hasOnStack("body")
	}
	return false
}

// getEndPriority returns the priority for end tag handling.
func getEndPriority(name string) int {
	if pri, ok := htmlEndPriority[name]; ok {
		return pri
	}
	return 100
}

// htmlAutoClose fires end element events for elements that should be
// auto-closed when newTag is encountered.
func (p *parser) htmlAutoClose(newTag string) {
	for p.currentName() != "" && shouldAutoClose(p.currentName(), newTag) {
		name := p.popName()
		_ = p.sax.EndElement(name)
	}
}

// htmlAutoCloseOnClose handles end tags that close intermediate elements.
func (p *parser) htmlAutoCloseOnClose(endTag string) {
	priority := getEndPriority(endTag)

	// Check if the end tag matches anything on the stack
	found := false
	for i := len(p.nameStack) - 1; i >= 0; i-- {
		if p.nameStack[i] == endTag {
			found = true
			break
		}
		if getEndPriority(p.nameStack[i]) > priority {
			return
		}
	}
	if !found {
		return
	}

	// Pop elements until we find the matching one.
	// Emit "tag mismatch" error only for elements with endTag == 3
	// (inline formatting elements like b, em, font, etc.) matching libxml2.
	for p.currentName() != "" && p.currentName() != endTag {
		cur := p.currentName()
		if desc := lookupElement(cur); desc != nil && desc.endTag == 3 {
			_ = p.emitError("Opening and ending tag mismatch: %s and %s", endTag, cur)
		}
		p.popName()
		_ = p.sax.EndElement(cur)
	}
}

// htmlAutoCloseOnEnd closes all remaining open elements.
func (p *parser) htmlAutoCloseOnEnd() {
	for len(p.nameStack) > 0 {
		name := p.popName()
		_ = p.sax.EndElement(name)
	}
}

// htmlCheckImplied inserts implied html/head/body elements as needed.
func (p *parser) htmlCheckImplied(newTag string) {
	if p.cfg.noImplied {
		return
	}
	if newTag == "html" {
		return
	}

	// Ensure <html> exists
	if len(p.nameStack) == 0 {
		p.pushName("html")
		_ = p.sax.StartElement("html", nil)
	}

	if newTag == "body" || newTag == "head" {
		return
	}

	// Head elements: ensure <head> if not yet in head/body
	if len(p.nameStack) <= 1 && isHeadElement(newTag) {
		if p.mode >= insertInHead {
			return
		}
		p.pushName("head")
		_ = p.sax.StartElement("head", nil)
		return
	}

	// Body elements
	if newTag != "noframes" && newTag != "frame" && newTag != "frameset" {
		if p.mode >= insertInBody {
			return
		}
		// Check if body or head is already on the stack
		for _, n := range p.nameStack {
			if n == "body" || n == "head" {
				return
			}
		}
		p.pushName("body")
		_ = p.sax.StartElement("body", nil)
	}
}

// parse runs the main parsing loop.
func (p *parser) parse(ctx context.Context) error {
	_ = p.sax.SetDocumentLocator(p.locator)
	_ = p.sax.StartDocument()

	for !p.atEnd() {
		if err := ctx.Err(); err != nil {
			return err
		}
		if p.peek() == '<' {
			if p.peekAt(1) == '/' {
				p.parseEndTag()
			} else if p.peekAt(1) == '!' {
				if p.peekAt(2) == '-' && p.peekAt(3) == '-' {
					p.parseComment()
				} else if matchCaseInsensitive(p.remaining(), "<!DOCTYPE") {
					p.parseDoctype()
				} else {
					// Bogus comment or similar — treat as comment
					p.parseBogusComment()
				}
			} else if p.peekAt(1) == '?' {
				// Processing instruction — in HTML mode, treated as comment
				p.parsePI()
			} else if isASCIIAlpha(p.peekAt(1)) {
				p.parseStartTag()
			} else {
				// Lone '<' — emit as character data
				_ = p.emitCharacters([]byte("<"))
				p.advance(1)
			}
		} else {
			p.parseCharacters()
		}
	}

	p.htmlAutoCloseOnEnd()
	_ = p.sax.EndDocument()
	return nil
}

// parseStartTag parses an HTML start tag: <tagname attrs...>
func (p *parser) parseStartTag() {
	p.advance(1) // skip '<'

	name := p.parseName()
	if name == "" {
		// Not a valid tag, emit '<' as text
		_ = p.emitCharacters([]byte("<"))
		return
	}

	name = strings.ToLower(name)

	// Parse attributes
	attrs := p.parseAttributes()

	// Skip whitespace and close
	p.skipWhitespace()
	if p.peek() == '/' {
		p.advance(1) // skip '/'
	}
	if p.peek() == '>' {
		p.advance(1) // skip '>'
	}

	// Auto-close and implied element handling
	p.htmlAutoClose(name)
	p.htmlCheckImplied(name)

	// Discard misplaced structural elements (html/head/body)
	if p.isMisplacedStructural(name) {
		p.depth++
		return
	}

	// Fire SAX event
	p.pushName(name)
	_ = p.sax.StartElement(name, attrs)

	// Handle void elements — immediately close
	desc := lookupElement(name)
	if desc != nil && desc.empty {
		p.popName()
		_ = p.sax.EndElement(name)
		return
	}

	// Handle raw text/script/RCDATA elements
	if desc != nil {
		switch desc.dataMode {
		case dataScript, dataRawText:
			p.parseRawContent(name)
		case dataRCDATA:
			p.parseRCDATAContent(name)
		case dataPlaintext:
			p.parsePlaintext()
		}
	}
}

// parseEndTag parses an HTML end tag: </tagname>
func (p *parser) parseEndTag() {
	p.advance(2) // skip '</'

	name := p.parseName()
	name = strings.ToLower(name)

	// Detect malformed end tag: characters like '<' after the tag name
	// but before '>' indicate a malformed tag (e.g., </font<).
	malformed := false
	var junkChar byte
	if !p.atEnd() && p.peek() != '>' {
		ch := p.peek()
		if ch != ' ' && ch != '\t' && ch != '\n' && ch != '\r' {
			malformed = true
			junkChar = ch
		}
	}

	// Skip to closing '>'
	for !p.atEnd() && p.peek() != '>' {
		p.advance(1)
	}
	if p.peek() == '>' {
		p.advance(1)
	}

	if name == "" {
		return
	}

	if malformed {
		_ = p.emitError("Unexpected end tag : %s", name+string(junkChar))
		return
	}

	// Skip end tags for discarded misplaced structural elements
	if (name == "html" || name == "head" || name == "body") && p.depth > 0 {
		p.depth--
		return
	}

	// Check if this end tag matches anything on the stack
	if !p.hasOnStack(name) {
		_ = p.emitError("Unexpected end tag : %s", name)
		return
	}

	// Use auto-close-on-close logic
	p.htmlAutoCloseOnClose(name)

	// After auto-close, check for tag mismatch
	if p.currentName() != "" && p.currentName() != name {
		_ = p.emitError("Opening and ending tag mismatch: %s and %s", name, p.currentName())
	}

	// If the current open element matches, close it
	if p.currentName() == name {
		p.popName()
		_ = p.sax.EndElement(name)
	}
}

// parseComment parses an HTML comment: <!-- ... -->
func (p *parser) parseComment() {
	p.advance(4) // skip '<!--'

	// Handle short comments: <!-->  and <!--->
	if p.peek() == '>' {
		// <!-->  — empty comment
		p.advance(1)
		_ = p.sax.Comment(nil)
		return
	}
	if p.peek() == '-' && p.peekAt(1) == '>' {
		// <!---> — empty comment
		p.advance(2)
		_ = p.sax.Comment(nil)
		return
	}

	start := p.pos
	for !p.atEnd() {
		// Check for end of comment: -->
		if p.peek() == '-' && p.peekAt(1) == '-' && p.peekAt(2) == '>' {
			data := p.input[start:p.pos]
			p.advance(3) // skip '-->'
			_ = p.sax.Comment(data)
			return
		}
		// Also handle incorrectly closed comment: --!>
		if p.peek() == '-' && p.peekAt(1) == '-' && p.peekAt(2) == '!' && p.peekAt(3) == '>' {
			data := p.input[start:p.pos]
			p.advance(4) // skip '--!>'
			_ = p.sax.Comment(data)
			return
		}
		p.advance(1)
	}

	// Unterminated comment — emit everything as comment
	data := p.input[start:p.pos]
	_ = p.sax.Comment(data)
}

// parseBogusComment parses a bogus comment: <! ... >
func (p *parser) parseBogusComment() {
	p.advance(2) // skip '<!'
	start := p.pos
	for !p.atEnd() && p.peek() != '>' {
		p.advance(1)
	}
	data := p.input[start:p.pos]
	if p.peek() == '>' {
		p.advance(1)
	}
	_ = p.sax.Comment(data)
}

// parsePI parses a processing instruction in HTML mode.
// In HTML, <?...> is treated as a comment by libxml2.
func (p *parser) parsePI() {
	// libxml2 emits the entire <?...> content as a comment (without the < and >).
	start := p.pos
	p.advance(1) // skip '<' — keep the '?' as part of comment content

	for !p.atEnd() {
		if p.peek() == '>' {
			data := p.input[start+1 : p.pos] // skip the '<', include '?'
			p.advance(1)                     // skip '>'
			_ = p.sax.Comment(data)
			return
		}
		p.advance(1)
	}
	data := p.input[start+1 : p.pos]
	_ = p.sax.Comment(data)
}

// parseDoctype parses a DOCTYPE declaration.
func (p *parser) parseDoctype() {
	// Skip <!DOCTYPE
	p.advance(9)
	p.skipWhitespace()

	// Parse root element name
	name := p.parseName()
	p.skipWhitespace()

	externalID := ""
	systemID := ""

	// Check for PUBLIC or SYSTEM
	rest := string(p.remaining())
	if strings.HasPrefix(strings.ToUpper(rest), "PUBLIC") {
		p.advance(6)
		p.skipWhitespace()
		externalID = p.parseQuotedString()
		p.skipWhitespace()
		systemID = p.parseQuotedString()
	} else if strings.HasPrefix(strings.ToUpper(rest), "SYSTEM") {
		p.advance(6)
		p.skipWhitespace()
		systemID = p.parseQuotedString()
	}

	// Skip to '>'
	for !p.atEnd() && p.peek() != '>' {
		p.advance(1)
	}
	if p.peek() == '>' {
		p.advance(1)
	}

	_ = p.sax.InternalSubset(name, externalID, systemID)
}

// parseCharacters parses character data (text content).
func (p *parser) parseCharacters() {
	// Collect text up to the next '<' or '&'.
	// We need to split at whitespace→non-whitespace boundaries when inside
	// <head> so that whitespace is emitted in <head> and non-whitespace
	// triggers head-close + body-open.
	start := p.pos
	inHead := p.currentName() == "head"

	for !p.atEnd() && p.peek() != '<' && p.peek() != '&' {
		b := p.peek()
		if inHead && !isWhitespaceByte(b) {
			// Non-whitespace while inside head — break here to emit
			// the preceding whitespace in head, then handle the rest
			break
		}
		p.advance(1)
	}

	if p.pos > start {
		text := p.input[start:p.pos]
		if !isAllWhitespace(text) {
			p.htmlStartCharData()
		}
		// Suppress whitespace before the root element has been seen
		if !p.sawRoot && isAllWhitespace(text) {
			return
		}
		_ = p.emitCharacters(text)

		// After emitting whitespace in head, continue to collect the
		// non-whitespace part (which will trigger head close on next call)
		return
	}

	// If we're at a non-whitespace char (after whitespace in head), collect it
	if !p.atEnd() && p.peek() != '<' && p.peek() != '&' {
		start = p.pos
		for !p.atEnd() && p.peek() != '<' && p.peek() != '&' {
			p.advance(1)
		}
		if p.pos > start {
			text := p.input[start:p.pos]
			if !isAllWhitespace(text) {
				p.htmlStartCharData()
			}
			_ = p.emitCharacters(text)
		}
		return
	}

	// Handle entity references in character data
	if !p.atEnd() && p.peek() == '&' {
		p.parseCharRef()
	}
}

// htmlStartCharData handles non-whitespace character data that appears
// in positions requiring implied element insertion. Mirrors libxml2's
// htmlStartCharData() which auto-closes head and ensures body is open.
func (p *parser) htmlStartCharData() {
	// If current element is <head>, auto-close it
	if p.currentName() == "head" {
		p.htmlAutoClose("p")
	}
	p.htmlCheckImplied("p")
}

// parseCharRef handles entity references (&name; or &#num; or &#xhex;).
// Emits the resolved value as a Characters SAX event (entity splitting behavior).
func (p *parser) parseCharRef() {
	// Entity content is non-whitespace — ensure implied elements
	p.htmlStartCharData()

	p.advance(1) // skip '&'

	if p.peek() == '#' {
		p.advance(1) // skip '#'
		var codepoint int
		if p.peek() == 'x' || p.peek() == 'X' {
			p.advance(1) // skip 'x'
			hexStr := p.parseWhile(isHexDigit)
			codepoint64, err := strconv.ParseInt(hexStr, 16, 32)
			if err == nil {
				codepoint = int(codepoint64)
			}
		} else {
			numStr := p.parseWhile(isDigit)
			codepoint64, err := strconv.ParseInt(numStr, 10, 32)
			if err == nil {
				codepoint = int(codepoint64)
			}
		}
		if p.peek() == ';' {
			p.advance(1)
		}
		if codepoint > 0 {
			var buf [4]byte
			n := utf8.EncodeRune(buf[:], rune(codepoint))
			_ = p.emitCharacters(buf[:n])
		}
		return
	}

	// Named entity
	nameStart := p.pos
	name := p.parseWhile(isAlphanumeric)
	hasSemicolon := false
	if p.peek() == ';' {
		hasSemicolon = true
		p.advance(1)
	}

	if name != "" {
		if val, ok := lookupEntity(name); ok {
			if hasSemicolon {
				_ = p.emitCharacters([]byte(val))
				return
			}
			// Without semicolon — only resolve legacy (HTML4) entities.
			// HTML5-only entities require a trailing semicolon.
			if isLegacyEntity(name) {
				_ = p.emitCharacters([]byte(val))
				return
			}
		}
		// No semicolon and full name is not a legacy entity.
		// Try prefix matching: find the longest legacy entity prefix.
		if !hasSemicolon {
			for i := len(name) - 1; i > 0; i-- {
				prefix := name[:i]
				if isLegacyEntity(prefix) {
					if val, ok := lookupEntity(prefix); ok {
						_ = p.emitCharacters([]byte(val))
						remainder := name[i:]
						_ = p.emitCharacters([]byte(remainder))
						return
					}
				}
			}
		}
	}

	// Unknown entity — emit as literal text
	text := "&" + string(p.input[nameStart:p.pos])
	_ = p.emitCharacters([]byte(text))
}

// emitError fires a SAX Error event unless suppressed by WithNoError.
func (p *parser) emitError(msg string, args ...any) error {
	if p.cfg.noError {
		return nil
	}
	return p.sax.Error(fmt.Errorf(msg, args...))
}

// emitCharacters fires the appropriate SAX Characters event.
// When noBlanks is set, whitespace-only data is suppressed unless
// inside a raw-text element (script, style, etc.).
func (p *parser) emitCharacters(data []byte) error {
	if p.cfg.noBlanks && isAllWhitespace(data) {
		if !p.inRawTextElement() {
			return nil
		}
	}
	if p.encodingError && bytes.ContainsRune(data, '\uFFFD') {
		// Temporarily override line/col so the DocumentLocator reports
		// the position of the first invalid byte in the original input.
		savedLine, savedCol := p.line, p.col
		p.line, p.col = p.encodingErrorLine, p.encodingErrorCol
		_ = p.emitError("Invalid bytes in character encoding")
		p.line, p.col = savedLine, savedCol
		p.encodingError = false
	}
	return p.sax.Characters(data)
}

// inRawTextElement reports whether the parser is currently inside a raw-text
// element (script, style, iframe, xmp, etc.) or an RCDATA element (title,
// textarea). Whitespace inside these elements must be preserved even with
// noBlanks.
func (p *parser) inRawTextElement() bool {
	name := p.currentName()
	if name == "" {
		return false
	}
	desc := lookupElement(name)
	return desc != nil && desc.dataMode >= dataRCDATA
}

// scriptState tracks the parser state within script content.
type scriptState int

const (
	scriptNormal        scriptState = 0 // normal script data
	scriptEscaped       scriptState = 1 // after <!--
	scriptDoubleEscaped scriptState = 2 // after <script inside <!--
)

// parseRawContent parses raw text content for script/style/iframe/xmp etc.
// Content is delivered as a CDataBlock SAX event.
// For script elements, implements the HTML5 script data states:
// - Normal: </script> closes; <!-- enters escaped
// - Escaped: </script> closes; --> returns to normal; <script enters double-escaped
// - Double-escaped: </script> returns to escaped; --> returns to normal
func (p *parser) parseRawContent(tagName string) {
	endTag := "</" + tagName
	startTag := "<" + tagName
	start := p.pos
	isScript := tagName == "script"
	state := scriptNormal

	for !p.atEnd() {
		// Check for <!-- to enter escaped state
		if isScript && state == scriptNormal && p.peek() == '<' && p.peekAt(1) == '!' &&
			p.peekAt(2) == '-' && p.peekAt(3) == '-' {
			state = scriptEscaped
			p.advance(4)
			continue
		}

		// Check for --> to exit escaped/double-escaped state
		if isScript && state != scriptNormal && p.peek() == '-' && p.peekAt(1) == '-' && p.peekAt(2) == '>' {
			state = scriptNormal
			p.advance(3)
			continue
		}

		// Check for <script to enter double-escaped state (only from escaped)
		if isScript && state == scriptEscaped && p.peek() == '<' && p.peekAt(1) != '/' {
			remaining := p.remaining()
			if len(remaining) >= len(startTag) &&
				strings.EqualFold(string(remaining[:len(startTag)]), startTag) {
				// Check next char is >, whitespace, or end of tag
				afterTag := len(startTag)
				if p.pos+afterTag >= len(p.input) || !isNameChar(p.input[p.pos+afterTag]) {
					state = scriptDoubleEscaped
					p.advance(afterTag)
					continue
				}
			}
		}

		// Check for </script> end tag
		if p.peek() == '<' && p.peekAt(1) == '/' {
			remaining := p.remaining()
			if len(remaining) >= len(endTag) &&
				strings.EqualFold(string(remaining[:len(endTag)]), endTag) {
				afterTag := len(endTag)
				validEnd := false
				if p.pos+afterTag >= len(p.input) {
					validEnd = true
				} else {
					ch := p.input[p.pos+afterTag]
					if ch == '>' || ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
						validEnd = true
					}
				}
				if validEnd {
					if state == scriptDoubleEscaped {
						// In double-escaped, </script> returns to escaped
						state = scriptEscaped
						p.advance(afterTag)
						if p.peek() == '>' {
							p.advance(1)
						}
						continue
					}
					// In normal or escaped state, </script> closes the element
					content := p.input[start:p.pos]
					if len(content) > 0 {
						_ = p.sax.CDataBlock(content)
					}
					return // Let the main loop handle the end tag
				}
			}
		}
		p.advance(1)
	}

	// Unterminated — emit everything as cdata
	content := p.input[start:p.pos]
	if len(content) > 0 {
		_ = p.sax.CDataBlock(content)
	}
}

// parseRCDATAContent parses RCDATA content (title, textarea).
// Like raw text but entities are expanded.
func (p *parser) parseRCDATAContent(tagName string) {
	endTag := "</" + tagName

	for !p.atEnd() {
		if p.peek() == '<' && p.peekAt(1) == '/' {
			remaining := p.remaining()
			if len(remaining) >= len(endTag) &&
				strings.EqualFold(string(remaining[:len(endTag)]), endTag) {
				afterTag := len(endTag)
				if p.pos+afterTag < len(p.input) {
					ch := p.input[p.pos+afterTag]
					if ch == '>' || ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
						return
					}
				} else {
					return
				}
			}
		}

		if p.peek() == '&' {
			p.parseCharRef()
		} else {
			// Collect text up to next & or potential end tag
			start := p.pos
			for !p.atEnd() && p.peek() != '&' && p.peek() != '<' {
				p.advance(1)
			}
			if p.pos > start {
				_ = p.emitCharacters(p.input[start:p.pos])
			}
			if !p.atEnd() && p.peek() == '<' {
				// Check if this is the end tag — if not, emit '<' as text
				remaining := p.remaining()
				if p.peekAt(1) != '/' || len(remaining) < len(endTag) ||
					!strings.EqualFold(string(remaining[:len(endTag)]), endTag) {
					_ = p.emitCharacters([]byte("<"))
					p.advance(1)
				}
			}
		}
	}
}

// parsePlaintext parses plaintext content — everything until EOF.
func (p *parser) parsePlaintext() {
	start := p.pos
	for !p.atEnd() {
		p.advance(1)
	}
	if p.pos > start {
		_ = p.sax.Characters(p.input[start:p.pos])
	}
}

// parseName parses an HTML tag name (letters, digits, colons, hyphens).
func (p *parser) parseName() string {
	start := p.pos
	for !p.atEnd() {
		b := p.peek()
		if isNameChar(b) {
			p.advance(1)
		} else {
			break
		}
	}
	return string(p.input[start:p.pos])
}

// parseAttributes parses HTML tag attributes.
// Duplicate attribute names are silently dropped (first occurrence wins),
// matching libxml2's htmlParseStartTag behavior.
func (p *parser) parseAttributes() []Attribute {
	var attrs []Attribute
	var seen map[string]struct{}

	for {
		p.skipWhitespace()
		if p.atEnd() || p.peek() == '>' || p.peek() == '/' {
			break
		}

		name := p.parseAttrName()
		if name == "" {
			// Skip unknown character
			p.advance(1)
			continue
		}

		name = strings.ToLower(name)
		p.skipWhitespace()

		value := ""
		isBool := false
		if p.peek() == '=' {
			p.advance(1) // skip '='
			p.skipWhitespace()
			value = p.parseAttrValue()
		} else {
			// Boolean attribute — no value specified
			isBool = true
		}

		if seen == nil {
			seen = make(map[string]struct{})
		}
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}

		attrs = append(attrs, Attribute{Name: name, Value: value, Boolean: isBool})
	}

	return attrs
}

// parseAttrName parses an attribute name.
// Uses negative-logic terminators: any character that is not a terminator
// is accepted, matching HTML's liberal attribute name rules.
func (p *parser) parseAttrName() string {
	start := p.pos
	for !p.atEnd() {
		b := p.peek()
		if isWhitespaceByte(b) || b == '>' || b == '/' || b == '=' || b == '"' || b == '\'' || b == '<' || b == 0 {
			break
		}
		p.advance(1)
	}
	return string(p.input[start:p.pos])
}

// parseAttrValue parses an attribute value (quoted or unquoted).
func (p *parser) parseAttrValue() string {
	if p.peek() == '"' {
		return p.parseQuotedAttrValue('"')
	}
	if p.peek() == '\'' {
		return p.parseQuotedAttrValue('\'')
	}
	// Unquoted attribute value
	return p.parseUnquotedAttrValue()
}

// parseQuotedAttrValue parses a quoted attribute value with entity expansion.
func (p *parser) parseQuotedAttrValue(quote byte) string {
	p.advance(1) // skip opening quote
	var buf bytes.Buffer

	for !p.atEnd() && p.peek() != quote {
		if p.peek() == '&' {
			buf.WriteString(p.resolveEntityInAttr())
		} else {
			buf.WriteByte(p.peek())
			p.advance(1)
		}
	}
	if p.peek() == quote {
		p.advance(1) // skip closing quote
	}
	return buf.String()
}

// parseUnquotedAttrValue parses an unquoted attribute value.
func (p *parser) parseUnquotedAttrValue() string {
	var buf bytes.Buffer

	for !p.atEnd() {
		b := p.peek()
		if b == '>' || b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\f' {
			break
		}
		if b == '&' {
			buf.WriteString(p.resolveEntityInAttr())
		} else {
			buf.WriteByte(b)
			p.advance(1)
		}
	}
	return buf.String()
}

// resolveEntityInAttr resolves an entity reference inside an attribute value.
// In HTML, named entities without a trailing ';' are NOT resolved when followed
// by '=' or an alphanumeric character (prevents mis-interpreting URL query strings).
func (p *parser) resolveEntityInAttr() string {
	p.advance(1) // skip '&'

	if p.peek() == '#' {
		p.advance(1)
		var codepoint int
		if p.peek() == 'x' || p.peek() == 'X' {
			p.advance(1)
			hexStr := p.parseWhile(isHexDigit)
			cp, err := strconv.ParseInt(hexStr, 16, 32)
			if err == nil {
				codepoint = int(cp)
			}
		} else {
			numStr := p.parseWhile(isDigit)
			cp, err := strconv.ParseInt(numStr, 10, 32)
			if err == nil {
				codepoint = int(cp)
			}
		}
		if p.peek() == ';' {
			p.advance(1)
		}
		if codepoint > 0 {
			return string(rune(codepoint))
		}
		return ""
	}

	name := p.parseWhile(isAlphanumeric)
	hasSemicolon := false
	if p.peek() == ';' {
		hasSemicolon = true
		p.advance(1)
	}

	if name != "" {
		if val, ok := lookupEntity(name); ok {
			// In attributes, only resolve if there's a semicolon, or if
			// the next character is NOT '=' or alphanumeric.
			if hasSemicolon {
				return val
			}
			// Without semicolon: only resolve legacy (HTML4) entities
			if !isLegacyEntity(name) {
				return "&" + name
			}
			// Without semicolon, check what follows
			next := p.peek()
			if next == '=' || isAlphanumeric(next) {
				// Don't resolve — treat & as literal
				return "&" + name
			}
			return val
		}
	}
	return "&" + name
}

// parseQuotedString parses a quoted string (for DOCTYPE).
func (p *parser) parseQuotedString() string {
	if p.peek() != '"' && p.peek() != '\'' {
		return ""
	}
	quote := p.peek()
	p.advance(1)
	start := p.pos
	for !p.atEnd() && p.peek() != quote {
		p.advance(1)
	}
	s := string(p.input[start:p.pos])
	if p.peek() == quote {
		p.advance(1)
	}
	return s
}

// parseWhile collects characters while pred returns true.
func (p *parser) parseWhile(pred func(byte) bool) string {
	start := p.pos
	for !p.atEnd() && pred(p.peek()) {
		p.advance(1)
	}
	return string(p.input[start:p.pos])
}

// skipWhitespace skips whitespace characters.
func (p *parser) skipWhitespace() {
	for !p.atEnd() {
		b := p.peek()
		if b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\f' {
			p.advance(1)
		} else {
			break
		}
	}
}

// Helper functions

func isASCIIAlpha(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func isNameChar(b byte) bool {
	return isASCIIAlpha(b) || isDigit(b) || b == ':' || b == '-' || b == '_' || b == '.'
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

func isHexDigit(b byte) bool {
	return isDigit(b) || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
}

func isAlphanumeric(b byte) bool {
	return isASCIIAlpha(b) || isDigit(b)
}

func isWhitespaceByte(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\f'
}

func isAllWhitespace(data []byte) bool {
	for _, b := range data {
		if !isWhitespaceByte(b) {
			return false
		}
	}
	return true
}

// matchCaseInsensitive checks if data starts with the given prefix (case-insensitive).
func matchCaseInsensitive(data []byte, prefix string) bool {
	if len(data) < len(prefix) {
		return false
	}
	return strings.EqualFold(string(data[:len(prefix)]), prefix)
}
