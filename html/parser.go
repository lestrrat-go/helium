package html

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/lestrrat-go/helium/internal/strcursor"
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
	cur *strcursor.UTF8Cursor

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

	// encodingSanitizer is non-nil when using the streaming reader path
	// (newParserFromReader). It is queried for deferred encoding error
	// position when the first U+FFFD is encountered in emitCharacters.
	encodingSanitizer *utf8SanitizeReader

	// deferredEncoding is non-nil when streaming an undeclared-charset
	// document whose encoding (UTF-8 vs Windows-1252) can only be decided
	// once a non-UTF-8 byte is seen. It is queried after parsing for the
	// final detected encoding name.
	deferredEncoding *deferredLatin1Reader

	// fatalSAXErr is set by handleSAXErr when cfg.strict is true and a SAX
	// callback returns a non-ErrHandlerUnspecified error. parse() surfaces
	// this value as the parse error. Outside strict mode it stays nil.
	fatalSAXErr error

	// fatalErr is set by a sub-parser that hits an unrecoverable condition
	// (e.g. a comment/PI exceeding MaxContentSize). The main parse loop checks
	// it and aborts, surfacing the value. Unlike fatalSAXErr this is always
	// fatal regardless of strict mode.
	fatalErr error

	cfg parseConfig
}

// handleSAXErr filters the by-design ErrHandlerUnspecified signal and routes
// every other non-nil return from a SAX callback. In the default (non-strict)
// mode the error is forwarded to the warning channel and parsing continues,
// preserving HTML's libxml2-compatible tolerance. In strict mode the first
// such error is captured and surfaced from parse().
func (p *parser) handleSAXErr(err error) {
	if err == nil || errors.Is(err, ErrHandlerUnspecified) {
		return
	}
	if p.cfg.strict {
		if p.fatalSAXErr == nil {
			p.fatalSAXErr = err
		}
		return
	}
	_ = p.emitWarning("%w", err)
}

// emitWarning routes a parser-tolerated condition to the SAX warning slot.
// Mirrors emitError but fires Warning(...) and is gated by cfg.noWarning so
// callers can silence these messages alongside emitError's output.
func (p *parser) emitWarning(msg string, args ...any) error {
	if p.cfg.noWarning {
		return nil
	}
	return p.sax.Warning(fmt.Errorf(msg, args...))
}

// parserLocator implements DocumentLocator.
type parserLocator struct {
	p        *parser
	overLine int // 0 = use cursor
	overCol  int
}

func (l *parserLocator) LineNumber() int {
	if l.overLine > 0 {
		return l.overLine
	}
	return l.p.cur.LineNumber()
}

func (l *parserLocator) ColumnNumber() int {
	if l.overCol > 0 {
		return l.overCol
	}
	return l.p.cur.Column()
}

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
		cur:               strcursor.NewUTF8Cursor(bytes.NewReader(normalized)),
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

// newParserFromReader creates a parser that reads from an io.Reader using
// streaming encoding wrappers. Unlike newParser (which pre-processes the
// entire []byte), this chains io.Reader wrappers for newline normalization
// and encoding conversion, feeding the result directly into the cursor.
func newParserFromReader(_ context.Context, r io.Reader, sax SAXHandler, cfg parseConfig) *parser {
	wrapped, detectedEnc, sanitizer, deferred := wrapReaderForHTML(r)

	p := &parser{
		cur:               strcursor.NewUTF8Cursor(wrapped),
		sax:               sax,
		mode:              insertInitial,
		detectedEncoding:  detectedEnc,
		encodingSanitizer: sanitizer,
		deferredEncoding:  deferred,
		cfg:               cfg,
	}
	p.locator = &parserLocator{p: p}
	return p
}

// finalEncoding returns the encoding detected for the input, accounting for
// the streaming deferred-Latin-1 path where the encoding name is only known
// after the full stream has been consumed during parsing.
func (p *parser) finalEncoding() string {
	if p.deferredEncoding != nil {
		if enc := p.deferredEncoding.detectedEncoding(); enc != "" {
			return enc
		}
	}
	return p.detectedEncoding
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

// hasPrefixFold checks if the current input starts with the given prefix
// (case-insensitive comparison).
func (p *parser) hasPrefixFold(prefix string) bool {
	got := p.cur.PeekString(len(prefix))
	return len(got) == len(prefix) && strings.EqualFold(got, prefix)
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
	if name == elemHTML {
		p.sawRoot = true
	}
	if p.mode < insertInHead && name == elemHead {
		p.mode = insertInHead
	}
	if p.mode < insertInBody && name == elemBody {
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
	case elemHTML:
		return len(p.nameStack) > 0
	case elemHead:
		return len(p.nameStack) != 1
	case elemBody:
		return p.hasOnStack(elemBody)
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
		p.handleSAXErr(p.sax.EndElement(name))
	}
}

// htmlAutoCloseOnClose handles end tags that close intermediate elements.
func (p *parser) htmlAutoCloseOnClose(endTag string) {
	priority := getEndPriority(endTag)

	// Check if the end tag matches anything on the stack
	found := false
	for _, v := range slices.Backward(p.nameStack) {
		if v == endTag {
			found = true
			break
		}
		if getEndPriority(v) > priority {
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
		p.handleSAXErr(p.sax.EndElement(cur))
	}
}

// htmlAutoCloseOnEnd closes all remaining open elements.
func (p *parser) htmlAutoCloseOnEnd() {
	for len(p.nameStack) > 0 {
		name := p.popName()
		p.handleSAXErr(p.sax.EndElement(name))
	}
}

// htmlCheckImplied inserts implied html/head/body elements as needed.
func (p *parser) htmlCheckImplied(newTag string) {
	if p.cfg.noImplied {
		return
	}
	if newTag == elemHTML {
		return
	}

	// Ensure <html> exists
	if len(p.nameStack) == 0 {
		p.pushName(elemHTML)
		p.handleSAXErr(p.sax.StartElement(elemHTML, nil))
	}

	if newTag == elemBody || newTag == elemHead {
		return
	}

	// Head elements: ensure <head> if not yet in head/body
	if len(p.nameStack) <= 1 && isHeadElement(newTag) {
		if p.mode >= insertInHead {
			return
		}
		p.pushName(elemHead)
		p.handleSAXErr(p.sax.StartElement(elemHead, nil))
		return
	}

	// Body elements
	if newTag != "noframes" && newTag != "frame" && newTag != elemFrameset {
		if p.mode >= insertInBody {
			return
		}
		// Check if body or head is already on the stack
		for _, n := range p.nameStack {
			if n == elemBody || n == elemHead {
				return
			}
		}
		p.pushName(elemBody)
		p.handleSAXErr(p.sax.StartElement(elemBody, nil))
	}
}

// parse runs the main parsing loop.
func (p *parser) parse(ctx context.Context) error {
	p.handleSAXErr(p.sax.SetDocumentLocator(p.locator))
	p.handleSAXErr(p.sax.StartDocument())

	for !p.cur.Done() {
		if err := ctx.Err(); err != nil {
			return err
		}
		if p.fatalErr != nil {
			return p.fatalErr
		}
		// A non-EOF read error from the underlying reader (e.g. the push
		// parser's stream returning the context error when its blocking wait
		// is cancelled) is recorded by the cursor and otherwise masked by
		// Done(). Surface it so a cancelled or failed read aborts the parse
		// rather than silently accepting truncated input.
		if err := p.cur.Err(); err != nil {
			return err
		}
		if p.cur.Peek() == '<' {
			if p.cur.PeekAt(1) == '/' {
				p.parseEndTag()
			} else if p.cur.PeekAt(1) == '!' {
				if p.cur.PeekAt(2) == '-' && p.cur.PeekAt(3) == '-' {
					p.parseComment(ctx)
				} else if p.hasPrefixFold("<!DOCTYPE") {
					p.parseDoctype()
				} else {
					// Bogus comment or similar — treat as comment
					p.parseBogusComment(ctx)
				}
			} else if p.cur.PeekAt(1) == '?' {
				// Processing instruction — in HTML mode, treated as comment
				p.parsePI(ctx)
			} else if isASCIIAlpha(p.cur.PeekAt(1)) {
				p.parseStartTag(ctx)
			} else {
				// Lone '<' — emit as character data
				_ = p.emitCharacters([]byte("<"))
				_ = p.cur.Advance(1)
			}
		} else {
			p.parseCharacters()
		}
	}

	if p.fatalErr != nil {
		return p.fatalErr
	}
	// A clean Done() may mask an underlying read error (e.g. a truncated or
	// checksummed stream that returned data together with a non-EOF error, or a
	// cancelled push-stream wait). Surface it as a fatal parse error rather than
	// accepting the input as a cleanly terminated document. Mirrors the XML
	// parser's cursorDecodeErr.
	if err := p.cur.Err(); err != nil {
		return err
	}

	p.htmlAutoCloseOnEnd()
	p.handleSAXErr(p.sax.EndDocument())
	return p.fatalSAXErr
}

// parseStartTag parses an HTML start tag: <tagname attrs...>
func (p *parser) parseStartTag(ctx context.Context) {
	_ = p.cur.Advance(1) // skip '<'

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
	if p.cur.Peek() == '/' {
		_ = p.cur.Advance(1) // skip '/'
	}
	if p.cur.Peek() == '>' {
		_ = p.cur.Advance(1) // skip '>'
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
	p.handleSAXErr(p.sax.StartElement(name, attrs))

	// Handle void elements — immediately close
	desc := lookupElement(name)
	if desc != nil && desc.empty {
		p.popName()
		p.handleSAXErr(p.sax.EndElement(name))
		return
	}

	// Handle raw text/script/RCDATA elements
	if desc != nil {
		switch desc.dataMode {
		case dataScript, dataRawText:
			p.parseRawContent(ctx, name)
		case dataRCDATA:
			p.parseRCDATAContent(ctx, name)
		case dataPlaintext:
			p.parsePlaintext(ctx)
		}
	}
}

// parseEndTag parses an HTML end tag: </tagname>
func (p *parser) parseEndTag() {
	_ = p.cur.Advance(2) // skip '</'

	name := p.parseName()
	name = strings.ToLower(name)

	// Detect malformed end tag: characters like '<' after the tag name
	// but before '>' indicate a malformed tag (e.g., </font<).
	malformed := false
	var junkChar byte
	if !p.cur.Done() && p.cur.Peek() != '>' {
		ch := p.cur.Peek()
		if ch != ' ' && ch != '\t' && ch != '\n' && ch != '\r' {
			malformed = true
			junkChar = ch
		}
	}

	// Skip to closing '>'
	for !p.cur.Done() && p.cur.Peek() != '>' {
		_ = p.cur.Advance(1)
	}
	if p.cur.Peek() == '>' {
		_ = p.cur.Advance(1)
	}

	if name == "" {
		return
	}

	if malformed {
		_ = p.emitError("Unexpected end tag : %s", name+string(junkChar))
		return
	}

	// Skip end tags for discarded misplaced structural elements
	if (name == elemHTML || name == elemHead || name == elemBody) && p.depth > 0 {
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
		p.handleSAXErr(p.sax.EndElement(name))
	}
}

// parseComment parses an HTML comment: <!-- ... -->
func (p *parser) parseComment(ctx context.Context) {
	_ = p.cur.Advance(4) // skip '<!--'

	// Handle short comments: <!-->  and <!--->
	if p.cur.Peek() == '>' {
		// <!-->  — empty comment
		_ = p.cur.Advance(1)
		p.handleSAXErr(p.sax.Comment(nil))
		return
	}
	if p.cur.Peek() == '-' && p.cur.PeekAt(1) == '>' {
		// <!---> — empty comment
		_ = p.cur.Advance(2)
		p.handleSAXErr(p.sax.Comment(nil))
		return
	}

	limit := p.cfg.contentLimit()
	n := 0
	// A comment maps to a single indivisible SAX event / DOM node, so it cannot
	// be chunked: emitting a truncated first chunk and returning mid-construct
	// would corrupt the document (the remainder leaks as stray text). Enforce
	// the content limit as a HARD cap — fail the parse if the comment exceeds
	// it before its terminator. Also abort promptly on cancellation rather than
	// scanning the whole (possibly unterminated) comment.
	for ctx.Err() == nil {
		// Use HasByteAt to distinguish EOF from a real NUL byte: PeekAt returns 0
		// for both, so a NUL inside the comment would otherwise be mistaken for
		// end-of-input and bypass the hard cap. A NUL counts as content.
		if !p.cur.HasByteAt(n) {
			break
		}
		b := p.cur.PeekAt(n)
		// Check for end of comment: -->
		if b == '-' && p.cur.PeekAt(n+1) == '-' && p.cur.PeekAt(n+2) == '>' {
			data := p.cur.PeekString(n)
			_ = p.cur.Advance(n + 3) // skip data + '-->'
			p.handleSAXErr(p.sax.Comment([]byte(data)))
			return
		}
		// Also handle incorrectly closed comment: --!>
		if b == '-' && p.cur.PeekAt(n+1) == '-' && p.cur.PeekAt(n+2) == '!' && p.cur.PeekAt(n+3) == '>' {
			data := p.cur.PeekString(n)
			_ = p.cur.Advance(n + 4) // skip data + '--!>'
			p.handleSAXErr(p.sax.Comment([]byte(data)))
			return
		}
		// No terminator at offset n: this byte is content. Accepting it would
		// make the content length n+1; fail only if that strictly exceeds the
		// limit. Content of exactly `limit` bytes followed by its terminator is
		// fine (the terminator checks above already ran for offset == limit).
		if n >= limit {
			p.fatalErr = fmt.Errorf("comment exceeds %d bytes before terminator: %w", limit, ErrContentSizeExceeded)
			return
		}
		n++
	}

	// The loop also exits on cancellation mid-scan. A comment is an indivisible
	// node, so emitting the bytes scanned so far would publish a truncated
	// comment with the remainder leaking as stray text. Abort without emitting.
	if ctx.Err() != nil {
		return
	}

	// Unterminated comment reaching EOF — emit everything as comment. (n is
	// bounded by limit, so this allocation is bounded.)
	data := p.cur.PeekString(n)
	_ = p.cur.Advance(n)
	p.handleSAXErr(p.sax.Comment([]byte(data)))
}

// parseBogusComment parses a bogus comment: <! ... >
func (p *parser) parseBogusComment(ctx context.Context) {
	_ = p.cur.Advance(2) // skip '<!'
	limit := p.cfg.contentLimit()
	n := 0
	// A bogus comment maps to a single indivisible SAX event / DOM node and
	// cannot be chunked, so enforce the content limit as a HARD cap: fail the
	// parse if it exceeds the limit before its '>' terminator rather than
	// emitting a truncated comment. Abort promptly on cancellation too.
	for ctx.Err() == nil {
		// HasByteAt distinguishes EOF from a real NUL (PeekAt returns 0 for both),
		// so a NUL inside the bogus comment is counted as content rather than
		// mistaken for end-of-input and bypassing the hard cap.
		if !p.cur.HasByteAt(n) {
			break
		}
		b := p.cur.PeekAt(n)
		if b == '>' {
			break
		}
		// No terminator at offset n: this byte is content. Accepting it would
		// make the content length n+1; fail only if that strictly exceeds the
		// limit so that exactly `limit` content bytes before '>' is accepted.
		if n >= limit {
			p.fatalErr = fmt.Errorf("bogus comment exceeds %d bytes before terminator: %w", limit, ErrContentSizeExceeded)
			return
		}
		n++
	}
	// Cancellation mid-scan must not publish a truncated (indivisible) comment.
	if ctx.Err() != nil {
		return
	}
	data := p.cur.PeekString(n)
	_ = p.cur.Advance(n)
	if p.cur.Peek() == '>' {
		_ = p.cur.Advance(1)
	}
	p.handleSAXErr(p.sax.Comment([]byte(data)))
}

// parsePI parses a processing instruction in HTML mode.
// In HTML, <?...> is treated as a comment by libxml2.
func (p *parser) parsePI(ctx context.Context) {
	// libxml2 emits the entire <?...> content as a comment (without the < and >).
	_ = p.cur.Advance(1) // skip '<' — keep the '?' as part of comment content

	limit := p.cfg.contentLimit()
	n := 0
	// A PI is emitted as a single indivisible comment SAX event / DOM node and
	// cannot be chunked, so enforce the content limit as a HARD cap: fail the
	// parse if it exceeds the limit before its '>' terminator rather than
	// emitting a truncated comment. Abort promptly on cancellation too.
	for ctx.Err() == nil {
		// HasByteAt distinguishes EOF from a real NUL (PeekAt returns 0 for both),
		// so a NUL inside the PI is counted as content rather than mistaken for
		// end-of-input and bypassing the hard cap.
		if !p.cur.HasByteAt(n) {
			break
		}
		b := p.cur.PeekAt(n)
		if b == '>' {
			data := p.cur.PeekString(n)
			_ = p.cur.Advance(n + 1) // skip data + '>'
			p.handleSAXErr(p.sax.Comment([]byte(data)))
			return
		}
		// No terminator at offset n: this byte is content. Accepting it would
		// make the content length n+1; fail only if that strictly exceeds the
		// limit so that exactly `limit` content bytes before '>' is accepted.
		if n >= limit {
			p.fatalErr = fmt.Errorf("processing instruction exceeds %d bytes before terminator: %w", limit, ErrContentSizeExceeded)
			return
		}
		n++
	}
	// Cancellation mid-scan must not publish a truncated (indivisible) PI/comment.
	if ctx.Err() != nil {
		return
	}
	data := p.cur.PeekString(n)
	_ = p.cur.Advance(n)
	p.handleSAXErr(p.sax.Comment([]byte(data)))
}

// parseDoctype parses a DOCTYPE declaration.
func (p *parser) parseDoctype() {
	// Skip <!DOCTYPE
	_ = p.cur.Advance(9)
	p.skipWhitespace()

	// Parse root element name
	name := p.parseName()
	p.skipWhitespace()

	externalID := ""
	systemID := ""

	// Check for PUBLIC or SYSTEM
	if p.hasPrefixFold("PUBLIC") {
		_ = p.cur.Advance(6)
		p.skipWhitespace()
		externalID = p.parseQuotedString()
		p.skipWhitespace()
		systemID = p.parseQuotedString()
	} else if p.hasPrefixFold("SYSTEM") {
		_ = p.cur.Advance(6)
		p.skipWhitespace()
		systemID = p.parseQuotedString()
	}

	// Skip to '>'
	for !p.cur.Done() && p.cur.Peek() != '>' {
		_ = p.cur.Advance(1)
	}
	if p.cur.Peek() == '>' {
		_ = p.cur.Advance(1)
	}

	p.handleSAXErr(p.sax.InternalSubset(name, externalID, systemID))
}

// parseCharacters parses character data (text content).
func (p *parser) parseCharacters() {
	// Collect text up to the next '<' or '&'.
	// We need to split at whitespace→non-whitespace boundaries when inside
	// <head> so that whitespace is emitted in <head> and non-whitespace
	// triggers head-close + body-open.
	inHead := p.currentName() == elemHead

	// A real U+0000 (NUL) byte is indistinguishable from EOF via Peek/PeekAt
	// (both return 0), so the scan loops below would break with no progress and
	// the outer parse loop would spin forever. Per HTML5 the data state treats
	// U+0000 as a parse error and replaces it with U+FFFD. Consume the NUL and
	// emit the replacement character, guaranteeing forward progress. EOF is
	// distinguished by Done().
	if !p.cur.Done() && p.cur.Peek() == 0 {
		_ = p.cur.Advance(1)
		p.htmlStartCharData()
		_ = p.emitCharacters([]byte("�"))
		return
	}

	n := 0
	for {
		b := p.cur.PeekAt(n)
		if b == 0 || b == '<' || b == '&' {
			break
		}
		if inHead && !isWhitespaceByte(b) {
			// Non-whitespace while inside head — break here to emit
			// the preceding whitespace in head, then handle the rest
			break
		}
		n++
	}

	if n > 0 {
		text := p.cur.PeekString(n)
		_ = p.cur.Advance(n)
		textBytes := []byte(text)
		if !isAllWhitespace(textBytes) {
			p.htmlStartCharData()
		}
		// Suppress whitespace before the root element has been seen
		if !p.sawRoot && isAllWhitespace(textBytes) {
			return
		}
		_ = p.emitCharacters(textBytes)

		// After emitting whitespace in head, continue to collect the
		// non-whitespace part (which will trigger head close on next call)
		return
	}

	// If we're at a non-whitespace char (after whitespace in head), collect it
	if !p.cur.Done() && p.cur.Peek() != '<' && p.cur.Peek() != '&' {
		n = 0
		for {
			b := p.cur.PeekAt(n)
			if b == 0 || b == '<' || b == '&' {
				break
			}
			n++
		}
		if n > 0 {
			text := p.cur.PeekString(n)
			_ = p.cur.Advance(n)
			textBytes := []byte(text)
			if !isAllWhitespace(textBytes) {
				p.htmlStartCharData()
			}
			_ = p.emitCharacters(textBytes)
		}
		return
	}

	// Handle entity references in character data
	if !p.cur.Done() && p.cur.Peek() == '&' {
		p.parseCharRef()
	}
}

// htmlStartCharData handles non-whitespace character data that appears
// in positions requiring implied element insertion. Mirrors libxml2's
// htmlStartCharData() which auto-closes head and ensures body is open.
func (p *parser) htmlStartCharData() {
	// If current element is <head>, auto-close it
	if p.currentName() == elemHead {
		p.htmlAutoClose("p")
	}
	p.htmlCheckImplied("p")
}

// normalizeNumericCharRef applies the HTML5 numeric-character-reference fixups
// relevant to NUL handling. A U+0000 reference is a parse error that maps to the
// replacement character U+FFFD rather than being dropped. Out-of-range and
// surrogate code points likewise map to U+FFFD instead of producing an invalid
// rune.
func normalizeNumericCharRef(cp int) rune {
	if cp == 0 || cp > 0x10FFFF || (cp >= 0xD800 && cp <= 0xDFFF) {
		return '�'
	}
	return rune(cp)
}

// parseNumericCharRef converts a numeric character reference's digit string
// (already extracted) into a code point. ok reports whether any digits were
// present at all; a bare "&#" / "&#x" with no digits yields ok==false so the
// caller emits nothing. When digits are present but the value overflows or
// exceeds the maximum Unicode code point, ok is true and cp is forced above the
// valid range so normalizeNumericCharRef maps it to U+FFFD (per HTML5) rather
// than being dropped.
func parseNumericCharRef(digits string, base int) (int, bool) {
	if digits == "" {
		return 0, false
	}
	v, err := strconv.ParseInt(digits, base, 32)
	if err != nil {
		// Out-of-range (overflow) or otherwise unrepresentable in 32 bits:
		// route to normalizeNumericCharRef via an out-of-range sentinel.
		return 0x110000, true
	}
	return int(v), true
}

// emitNumericCharRef emits the normalized output of a numeric character
// reference. codepoint/haveDigits come from parsing the digit run (decimal or
// hex). Per HTML5 a U+0000 reference, an overflowing/out-of-range value, or a
// surrogate maps to U+FFFD via normalizeNumericCharRef rather than being
// dropped; nothing is emitted only when no digits were present at all.
func (p *parser) emitNumericCharRef(codepoint int, haveDigits bool) {
	if !haveDigits {
		return
	}
	cp := normalizeNumericCharRef(codepoint)
	var buf [4]byte
	n := utf8.EncodeRune(buf[:], cp)
	_ = p.emitCharacters(buf[:n])
}

// resolveNamedEntity applies the HTML5 named-character-reference matching rules
// to an already-scanned entity name (without the leading '&' or trailing ';').
// hasSemicolon reports whether a ';' followed the name. It returns the resolved
// replacement bytes and true when the name (or a legacy prefix of it) resolves;
// the remainder string holds any unmatched suffix that follows a legacy-prefix
// match and must be emitted as literal text after the replacement. When nothing
// resolves it returns ok==false and the caller emits the run as literal text.
func resolveNamedEntity(name string, hasSemicolon bool) (val, remainder string, ok bool) {
	if name == "" {
		return "", "", false
	}
	if v, found := lookupEntity(name); found {
		if hasSemicolon {
			return v, "", true
		}
		// Without semicolon — only resolve legacy (HTML4) entities.
		// HTML5-only entities require a trailing semicolon.
		if isLegacyEntity(name) {
			return v, "", true
		}
	}
	// No semicolon and full name is not a legacy entity.
	// Try prefix matching: find the longest legacy entity prefix.
	if !hasSemicolon {
		for i := len(name) - 1; i > 0; i-- {
			prefix := name[:i]
			if !isLegacyEntity(prefix) {
				continue
			}
			if v, found := lookupEntity(prefix); found {
				return v, name[i:], true
			}
		}
	}
	return "", "", false
}

// parseCharRef handles entity references (&name; or &#num; or &#xhex;).
// Emits the resolved value as a Characters SAX event (entity splitting behavior).
func (p *parser) parseCharRef() {
	// Entity content is non-whitespace — ensure implied elements
	p.htmlStartCharData()

	_ = p.cur.Advance(1) // skip '&'

	if p.cur.Peek() == '#' {
		_ = p.cur.Advance(1) // skip '#'
		var codepoint int
		var haveDigits bool
		if p.cur.Peek() == 'x' || p.cur.Peek() == 'X' {
			_ = p.cur.Advance(1) // skip 'x'
			hexStr := p.parseWhile(isHexDigit)
			codepoint, haveDigits = parseNumericCharRef(hexStr, 16)
		} else {
			numStr := p.parseWhile(isDigit)
			codepoint, haveDigits = parseNumericCharRef(numStr, 10)
		}
		if p.cur.Peek() == ';' {
			_ = p.cur.Advance(1)
		}
		p.emitNumericCharRef(codepoint, haveDigits)
		return
	}

	// Named entity
	name := p.parseWhile(isAlphanumeric)
	hasSemicolon := false
	if p.cur.Peek() == ';' {
		hasSemicolon = true
		_ = p.cur.Advance(1)
	}

	if val, remainder, ok := resolveNamedEntity(name, hasSemicolon); ok {
		_ = p.emitCharacters([]byte(val))
		if remainder != "" {
			_ = p.emitCharacters([]byte(remainder))
		}
		return
	}

	// Unknown entity — emit as literal text
	text := "&" + name
	if hasSemicolon {
		text += ";"
	}
	_ = p.emitCharacters([]byte(text))
}

// maxEntityNameLen is one past the length of the longest named HTML entity
// ("CounterClockwiseContourIntegral", 31 chars). An alphanumeric run reaching
// this length cannot match any known entity, so the char-ref scan can stop and
// treat the run as unresolved literal text without consulting the entity table.
const maxEntityNameLen = 32

// parseCharRefBounded handles an entity reference inside cap-aware content
// (RCDATA: title/textarea) where the surrounding text is flushed to SAX in
// chunks no larger than limit. It makes the SAME entity-resolution decisions as
// parseCharRef — numeric references (including overlong/leading-zero/overflow
// runs) normalize to their HTML5 output (U+FFFD on overflow/invalid), and named
// references resolve identically including legacy-prefix matching.
//
// Memory is kept bounded by two independent budgets:
//
//   - Entity-name resolution uses a FIXED lookahead of maxEntityNameLen bytes,
//     a constant independent of MaxContentSize. No known entity (longest 31
//     chars) or legacy prefix (≤6 chars) is longer, so a resolvable reference
//     is always decided within that constant window — and therefore NEVER
//     rejected, regardless of how small MaxContentSize is. `&amp;` resolves
//     even under MaxContentSize(2); `&amp` + a long alphanumeric tail resolves
//     the legacy "amp" prefix and emits the tail as ordinary (chunked) text.
//
//   - The MaxContentSize cap governs only the LITERAL text emitted for an
//     UNRESOLVED run. An unbounded digit run is consumed in fixed-size chunks
//     while tracking value/overflow rather than buffered whole; an unresolved
//     reference is flushed as literal text in capped chunks; and an unresolved
//     literal run that genuinely EXCEEDS the cap before any terminator fails
//     the parse with ErrContentSizeExceeded so peak retained memory stays
//     bounded.
//
// ctx is checked between bounded numeric chunks so a cancelled parse aborts
// promptly without first draining a long digit run.
func (p *parser) parseCharRefBounded(ctx context.Context, limit int) {
	// Entity content is non-whitespace — ensure implied elements.
	p.htmlStartCharData()

	_ = p.cur.Advance(1) // skip '&'

	if p.cur.Peek() == '#' {
		_ = p.cur.Advance(1) // skip '#'
		base := 10
		pred := isDigit
		if p.cur.Peek() == 'x' || p.cur.Peek() == 'X' {
			base = 16
			pred = isHexDigit
			_ = p.cur.Advance(1) // skip 'x'
		}
		// Consume the digit run in bounded chunks, accumulating the code point
		// with overflow saturation so an arbitrarily long (e.g. leading-zero or
		// overflowing) run never materializes as one string. The result matches
		// parseNumericCharRef: a value above U+10FFFF maps to U+FFFD via
		// emitNumericCharRef, and a leading-zero run still resolves to its value.
		// A cancelled context unwinds without emitting a partial entity.
		codepoint, haveDigits, cancelled := p.consumeNumericCharRefBounded(ctx, pred, base, limit)
		if cancelled {
			return
		}
		if p.cur.Peek() == ';' {
			_ = p.cur.Advance(1)
		}
		p.emitNumericCharRef(codepoint, haveDigits)
		return
	}

	// Named entity. Resolution uses a FIXED maxEntityNameLen-byte lookahead — a
	// constant, NOT MaxContentSize — because every known entity and legacy
	// prefix is shorter than that window, so this is O(1) memory regardless of
	// the user's cap.
	name := p.parseWhileMax(isAlphanumeric, maxEntityNameLen)

	// The lookahead SATURATES when the alphanumeric run reaches the fixed window
	// AND keeps going. parseCharRef scans the WHOLE run before deciding, so its
	// resolveNamedEntity sees the full (over-long) name; a long ';'-terminated
	// name is NOT legacy-prefix-resolved (that loop is gated on !hasSemicolon),
	// and an over-long no-semicolon name resolves only its longest legacy prefix.
	// We must mirror that exactly. Resolving from the truncated 32-byte window
	// here — before knowing whether a ';' eventually terminates the run — would
	// wrongly legacy-resolve the prefix of a ';'-terminated name. So when the run
	// saturates, DEFER the decision: drain the rest of the run to learn whether
	// it is ';'-terminated, then treat it as an unresolved literal (cap-enforced).
	// No known entity (≤31 chars) reaches the window, so a saturated run can
	// never be a named-entity match; the only resolution it could ever produce is
	// a legacy prefix, and that applies only when there is no semicolon.
	if len(name) >= maxEntityNameLen && isAlphanumeric(p.cur.Peek()) {
		p.parseSaturatedCharRefLiteral(ctx, name, limit)
		return
	}

	hasSemicolon := false
	if p.cur.Peek() == ';' {
		hasSemicolon = true
		_ = p.cur.Advance(1)
	}

	if val, remainder, ok := resolveNamedEntity(name, hasSemicolon); ok {
		_ = p.emitCharacters([]byte(val))
		if remainder != "" {
			// remainder is ASCII (alphanumeric tail of the run); chunk it so it
			// is never emitted as one oversized Characters event.
			p.emitLiteralChunked(remainder, limit)
		}
		return
	}

	// Unknown entity within the lookahead — emit "&" + name (and any ';') as
	// literal text in capped chunks. Even though the name fits the fixed
	// lookahead window, the LITERAL run it produces is still charged against
	// MaxContentSize: "&" plus the name length (plus any ';'). If that exceeds
	// the cap, fail with ErrContentSizeExceeded rather than emitting an over-cap
	// literal.
	sizeCap := limit
	if sizeCap <= 0 {
		sizeCap = defaultMaxContentSize
	}
	// Charge the full literal that will be emitted: '&' + name plus the trailing
	// ';' when one was consumed as part of the unresolved run. Omitting the ';'
	// would undercount the literal and let an over-cap run slip past.
	literalLen := 1 + len(name)
	if hasSemicolon {
		literalLen++
	}
	if literalLen > sizeCap {
		p.fatalErr = fmt.Errorf("unresolved character reference exceeds %d bytes before terminator: %w", sizeCap, ErrContentSizeExceeded)
		return
	}
	text := "&" + name
	if hasSemicolon {
		text += ";"
	}
	p.emitLiteralChunked(text, limit)
}

// parseSaturatedCharRefLiteral handles a named char-ref whose alphanumeric run
// overflowed the fixed maxEntityNameLen lookahead window: head holds the first
// maxEntityNameLen bytes already scanned and the run is known to continue. Such
// a run can never match a KNOWN entity (the longest is 31 chars), so the only
// resolution parseCharRef could ever apply to it is a legacy-prefix match — and
// that applies ONLY when the run is NOT ';'-terminated (resolveNamedEntity gates
// its prefix loop on !hasSemicolon). This function mirrors parseCharRef's exact
// decision tree for the saturated run:
//
//   - no ';': resolve the longest legacy prefix of the run (which lies entirely
//     within head, ≤6 chars) and emit the unmatched remainder as ordinary text.
//     The remainder is the rest of head plus the alphanumeric tail still in the
//     stream. This function drains that tail ITSELF (it must, to learn whether a
//     trailing ';' exists): we resolve from head, emit head's leftover, then emit
//     the tail — buffering it while the literal it would form stays within cap, or
//     streaming over-cap legacy-tail chunks instead of retaining them so a
//     (possibly unbounded) tail is never held whole. If head has no legacy prefix
//     the whole run is literal (see below).
//   - ';'-terminated: an over-long UNKNOWN name. parseCharRef does NOT
//     legacy-resolve it; the WHOLE run plus ';' is emitted literally and charged
//     against MaxContentSize, failing with ErrContentSizeExceeded once it
//     exceeds the cap before the terminator.
//
// The legacy decision depends only on head, but the ';' decision requires
// knowing whether a terminator ends the run. We therefore peek the tail just far
// enough to settle that: if head has a legacy prefix we optimistically take the
// no-';' path UNLESS a ';' is found, in which case the run is literal.
//
// Bounded WORK, not just bounded MEMORY: when the run already exceeds the cap and
// head does NOT legacy-resolve, the outcome is ErrContentSizeExceeded regardless
// of any trailing ';', so the function fails IMMEDIATELY instead of draining the
// rest of the (possibly unbounded) tail. ctx is also checked between bounded
// chunks so a cancelled parse aborts promptly without consuming the whole run.
func (p *parser) parseSaturatedCharRefLiteral(ctx context.Context, head string, limit int) {
	sizeCap := limit
	if sizeCap <= 0 {
		sizeCap = defaultMaxContentSize
	}

	// Does head begin with a resolvable legacy prefix? This is the ONLY way a
	// saturated run can resolve, and it depends solely on head (a legacy entity
	// is ≤6 chars). Compute it once: it selects between the resolve path and the
	// literal path, gated on the absence of a ';' (settled below).
	val, remainder, legacyResolves := resolveNamedEntity(head, false)

	// We must learn whether a ';' terminates the run to choose between the legacy
	// resolve path (no ';') and the literal path (';'-terminated unknown name).
	// The ';' can only follow the END of the alphanumeric run, so the decision is
	// not known until the run's end is reached. The two interpretations emit
	// DIFFERENT things, so NOTHING may be emitted until the ';' question is
	// settled — emitting on the optimistic legacy path and then discovering a
	// trailing ';' would leave a partial Characters callback ALREADY delivered
	// ahead of an ErrContentSizeExceeded, corrupting downstream output.
	//
	// The ';' is settled by a NON-CONSUMING lookahead over the rest of the run
	// (parseSaturatedCharRefSemicolon below) so that, whichever interpretation
	// wins, NOTHING is emitted before the decision is final. The decision tree:
	//
	//   - head does NOT legacy-resolve: the whole run is an unresolved literal no
	//     matter whether a ';' terminates it. If it already exceeds the cap the
	//     outcome is ErrContentSizeExceeded regardless of the (possibly unbounded)
	//     tail — so we fail IMMEDIATELY without scanning the rest (bounded WORK,
	//     not merely bounded memory) and emit nothing.
	//   - head legacy-resolves, no ';': mirror parseCharRef's no-semicolon
	//     legacy-prefix path — emit the resolution + head's leftover, then the
	//     tail as ordinary text. Only now, with ';' ruled out, does emission
	//     begin; the tail is consumed and emitted in capped chunks so an unbounded
	//     no-';' tail is never delivered as one oversized Characters event.
	//   - ';'-terminated: an over-long unknown name. parseCharRef does NOT
	//     legacy-resolve a ';'-terminated name; the whole run is literal. Within
	//     the cap it is echoed verbatim; over the cap it is ErrContentSizeExceeded
	//     with nothing emitted.
	literalLen := 1 + len(head) // '&' + head; grows as the run is scanned
	overCap := literalLen > sizeCap

	// Fast-fail the non-legacy over-cap case before touching the tail: the run is
	// literal regardless of any trailing ';', and it already exceeds the cap, so
	// the outcome is fixed. Bail without scanning a possibly unbounded tail and
	// without any SAX callback.
	if overCap && !legacyResolves {
		p.fatalErr = fmt.Errorf("unresolved character reference exceeds %d bytes before terminator: %w", sizeCap, ErrContentSizeExceeded)
		return
	}

	// Settle the ';' decision WITHOUT consuming or emitting anything. tailLen is
	// the byte length of the alphanumeric run after head; hasSemicolon reports a
	// terminator immediately past it. tail holds the run bytes only while the
	// literal it would form still fits the cap; tailComplete reports whether tail
	// holds the entire run.
	hasSemicolon, tailLen, tail, tailComplete, cancelled := p.parseSaturatedCharRefSemicolon(ctx, sizeCap, literalLen)
	if cancelled {
		return
	}
	literalLen += tailLen
	if hasSemicolon {
		literalLen++
	}
	if literalLen > sizeCap {
		overCap = true
	}

	// Decision settled. No ';' and head legacy-resolves: mirror parseCharRef's
	// no-semicolon legacy-prefix path. Emit the resolution + head's leftover, then
	// the tail as ordinary text — only now, with ';' ruled out, so nothing was
	// emitted prematurely. The tail bytes are still unconsumed in the stream: drain
	// and emit them in capped chunks (an unbounded no-';' tail is never buffered
	// whole nor delivered as one oversized event).
	if !hasSemicolon && legacyResolves {
		_ = p.emitCharacters([]byte(val))
		if remainder != "" {
			p.emitLiteralChunked(remainder, limit)
		}
		for {
			if ctx.Err() != nil {
				return
			}
			chunk := p.parseWhileMax(isAlphanumeric, sizeCap)
			if chunk == "" {
				break
			}
			p.emitLiteralChunked(chunk, limit)
		}
		return
	}

	// ';'-terminated (over-long unknown name) or a no-';' run with no legacy
	// prefix: the WHOLE run is an unresolved literal charged against the cap.
	if overCap {
		p.fatalErr = fmt.Errorf("unresolved character reference exceeds %d bytes before terminator: %w", sizeCap, ErrContentSizeExceeded)
		return
	}
	// Within cap: the whole run was retained during the lookahead (the cap was
	// never crossed). Consume it and emit it literally including any ';'.
	if !tailComplete { // defensive: a within-cap run is always fully retained
		p.fatalErr = fmt.Errorf("unresolved character reference exceeds %d bytes before terminator: %w", sizeCap, ErrContentSizeExceeded)
		return
	}
	_ = p.cur.Advance(tailLen)
	if hasSemicolon {
		_ = p.cur.Advance(1)
	}
	text := "&" + head + tail
	if hasSemicolon {
		text += ";"
	}
	p.emitLiteralChunked(text, limit)
}

// parseSaturatedCharRefSemicolon performs a NON-CONSUMING lookahead over the
// remainder of a saturated alphanumeric char-ref run (the bytes after head) to
// settle whether the run is ';'-terminated, WITHOUT advancing the cursor or
// emitting anything. The caller needs this decision before choosing between the
// legacy-resolve path and the literal path, and neither path may emit a partial
// result ahead of the decision.
//
// It returns:
//   - hasSemicolon: a ';' immediately follows the alphanumeric run.
//   - tailLen: byte length of the alphanumeric run after head.
//   - tail: the run bytes, retained ONLY while the literal "&"+head+tail (plus a
//     possible ';') still fits the cap; empty/partial once the cap is crossed.
//   - tailComplete: whether tail holds the entire run (cap never crossed).
//   - cancelled: ctx was cancelled mid-scan.
//
// The lookahead is bounded WORK as well as bounded memory: once the literal
// exceeds the cap the only remaining question is whether a ';' makes the run an
// over-cap literal (a hard failure) or a no-';' legacy-resolve — both of which
// the caller handles by consuming the run itself afterwards — so we keep peeking
// only far enough to find the run's end. Because nothing is consumed, the caller
// can later drain the same bytes to emit them on whichever path wins.
func (p *parser) parseSaturatedCharRefSemicolon(ctx context.Context, sizeCap, baseLen int) (hasSemicolon bool, tailLen int, tail string, tailComplete bool, cancelled bool) {
	var buf strings.Builder
	tailComplete = baseLen <= sizeCap
	off := 0
	const ctxCheckStride = 4096
	for {
		// Abort promptly on context cancellation so a cancelled parse never has
		// to scan a long (possibly unbounded) run.
		if off%ctxCheckStride == 0 && ctx.Err() != nil {
			return false, 0, "", false, true
		}
		b := p.cur.PeekAt(off)
		if b == 0 || !isAlphanumeric(b) {
			break
		}
		off++
		if baseLen+off > sizeCap {
			tailComplete = false
		}
		if tailComplete {
			buf.WriteByte(b)
		}
	}
	tailLen = off
	hasSemicolon = p.cur.PeekAt(off) == ';'
	return hasSemicolon, tailLen, buf.String(), tailComplete, false
}

// consumeNumericCharRefBounded reads a numeric character reference's digit run
// (digits matching pred, interpreted in the given base) in chunks no larger
// than limit, accumulating the code-point value with overflow saturation so an
// arbitrarily long run is never buffered whole. It returns the accumulated code
// point, whether any digits were present, and whether ctx was cancelled
// mid-run. A value that exceeds U+10FFFF (or otherwise overflows) saturates
// above the valid range so emitNumericCharRef maps it to U+FFFD, matching
// parseNumericCharRef; leading zeros are tolerated.
//
// ctx is checked between bounded chunks so a cancelled parse inside a long
// digit run (e.g. <title>&#9999...) aborts promptly instead of consuming the
// whole run. On cancellation it returns cancelled=true and the caller emits no
// (partial) entity, letting the outer parse surface context.Canceled.
func (p *parser) consumeNumericCharRefBounded(ctx context.Context, pred func(byte) bool, base, limit int) (int, bool, bool) {
	chunkSize := limit
	if chunkSize <= 0 {
		chunkSize = maxNumericRefLen
	}
	const overflow = 0x110000 // one past U+10FFFF — normalizes to U+FFFD
	value := 0
	haveDigits := false
	saturated := false
	for {
		if ctx.Err() != nil {
			return 0, false, true
		}
		chunk := p.parseWhileMax(pred, chunkSize)
		if chunk == "" {
			break
		}
		haveDigits = true
		if saturated {
			continue // value already pinned above the valid range
		}
		for i := range len(chunk) {
			value = value*base + digitValue(chunk[i], base)
			if value > overflow {
				value = overflow
				saturated = true
				break
			}
		}
	}
	if !haveDigits {
		return 0, false, false
	}
	return value, true, false
}

// digitValue returns the numeric value of an ASCII hex/decimal digit byte for
// the given base. The byte is assumed to satisfy the matching digit predicate.
func digitValue(b byte, base int) int {
	switch {
	case b >= '0' && b <= '9':
		return int(b - '0')
	case base == 16 && b >= 'a' && b <= 'f':
		return int(b-'a') + 10
	case base == 16 && b >= 'A' && b <= 'F':
		return int(b-'A') + 10
	}
	return 0
}

// maxNumericRefLen is the fallback chunk size for draining a numeric digit run
// when no positive content limit is configured. It is generous so legitimate
// (zero-padded) references still resolve while keeping per-chunk allocations
// bounded.
const maxNumericRefLen = 32

// emitLiteralChunked emits s as literal text in Characters events no larger
// than limit bytes. s contains only ASCII (the '&', '#', 'x', and alphanumeric
// or digit bytes of a character reference), so splitting on byte boundaries
// never breaks a multi-byte rune.
func (p *parser) emitLiteralChunked(s string, limit int) {
	if limit <= 0 {
		limit = defaultMaxContentSize
	}
	for len(s) > limit {
		_ = p.emitCharacters([]byte(s[:limit]))
		s = s[limit:]
	}
	if len(s) > 0 {
		_ = p.emitCharacters([]byte(s))
	}
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
	if bytes.ContainsRune(data, '\uFFFD') {
		if p.encodingError {
			// Batch path: error position was computed during pre-processing.
			p.locator.overLine = p.encodingErrorLine
			p.locator.overCol = p.encodingErrorCol
			_ = p.emitError("Invalid bytes in character encoding")
			p.locator.overLine = 0
			p.locator.overCol = 0
			p.encodingError = false
		} else if p.encodingSanitizer != nil {
			// Streaming path: query sanitizer for deferred error position.
			if hasErr, line, col := p.encodingSanitizer.EncodingError(); hasErr {
				p.locator.overLine = line
				p.locator.overCol = col
				_ = p.emitError("Invalid bytes in character encoding")
				p.locator.overLine = 0
				p.locator.overCol = 0
				p.encodingSanitizer = nil
			}
		}
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
func (p *parser) parseRawContent(ctx context.Context, tagName string) {
	endTag := "</" + tagName
	startTag := "<" + tagName
	isScript := tagName == "script"
	state := scriptNormal
	limit := p.cfg.contentLimit()
	var content bytes.Buffer

	flushChunk := func() {
		// Clone the bytes before Reset: bytes.Buffer.Reset reuses the same
		// backing array, so a SAX handler that retains the slice would see
		// this chunk overwritten by subsequent content.
		if content.Len() > 0 {
			p.handleSAXErr(p.sax.CDataBlock(bytes.Clone(content.Bytes())))
			content.Reset()
		}
	}
	// append writes a whole token (a single rune or a complete ASCII tag
	// fragment) to the buffer, flushing first if it would push the chunk past
	// the cap. Flushing on whole-token boundaries keeps every emitted chunk
	// valid UTF-8: a multi-byte rune is never split across two chunks. A single
	// token larger than the cap is emitted whole as its own chunk rather than
	// split, so no partial rune is ever produced.
	appendToken := func(s string) {
		if content.Len() > 0 && content.Len()+len(s) > limit {
			flushChunk()
		}
		content.WriteString(s)
	}

	for !p.cur.Done() {
		// Abort promptly on context cancellation rather than buffering the
		// entire (possibly gigantic or unterminated) section first. The main
		// parse loop re-checks ctx.Err() and surfaces it.
		if ctx.Err() != nil {
			flushChunk()
			return
		}
		// Check for <!-- to enter escaped state
		if isScript && state == scriptNormal && p.cur.Peek() == '<' && p.cur.PeekAt(1) == '!' &&
			p.cur.PeekAt(2) == '-' && p.cur.PeekAt(3) == '-' {
			state = scriptEscaped
			appendToken(p.cur.PeekString(4))
			_ = p.cur.Advance(4)
			continue
		}

		// Check for --> to exit escaped/double-escaped state
		if isScript && state != scriptNormal && p.cur.Peek() == '-' && p.cur.PeekAt(1) == '-' && p.cur.PeekAt(2) == '>' {
			state = scriptNormal
			appendToken(p.cur.PeekString(3))
			_ = p.cur.Advance(3)
			continue
		}

		// Check for <script to enter double-escaped state (only from escaped)
		if isScript && state == scriptEscaped && p.cur.Peek() == '<' && p.cur.PeekAt(1) != '/' {
			if p.hasPrefixFold(startTag) {
				// Check next char is >, whitespace, or end of tag
				afterTag := len(startTag)
				if p.cur.PeekAt(afterTag) == 0 || !isNameChar(p.cur.PeekAt(afterTag)) {
					state = scriptDoubleEscaped
					appendToken(p.cur.PeekString(afterTag))
					_ = p.cur.Advance(afterTag)
					continue
				}
			}
		}

		// Check for </script> end tag
		if p.cur.Peek() == '<' && p.cur.PeekAt(1) == '/' {
			if p.hasPrefixFold(endTag) {
				afterTag := len(endTag)
				validEnd := false
				switch {
				case !p.cur.HasByteAt(afterTag):
					// True EOF after the matched tag terminates the element.
					// A real NUL byte (HasByteAt true, PeekAt 0) does not.
					validEnd = true
				default:
					switch p.cur.PeekAt(afterTag) {
					case '>', ' ', '\t', '\n', '\r':
						validEnd = true
					}
				}
				if validEnd {
					if state == scriptDoubleEscaped {
						// In double-escaped, </script> returns to escaped
						state = scriptEscaped
						appendToken(p.cur.PeekString(afterTag))
						_ = p.cur.Advance(afterTag)
						if p.cur.Peek() == '>' {
							appendToken(">")
							_ = p.cur.Advance(1)
						}
						continue
					}
					// In normal or escaped state, </script> closes the element
					flushChunk()
					return // Let the main loop handle the end tag
				}
			}
		}
		// Per HTML5 the RAWTEXT/script-data states replace U+0000 with U+FFFD.
		// The loop already advances on every byte (so no spin), but emit the
		// replacement character instead of a literal NUL for correctness.
		if p.cur.Peek() == 0 {
			appendToken("�")
			_ = p.cur.Advance(1)
			continue
		}
		// Consume a whole rune at a time and append it as one indivisible
		// token, so the cap-aware flush in appendToken never splits a
		// multi-byte UTF-8 sequence across two chunks.
		s, n := p.peekRuneToken()
		appendToken(s)
		_ = p.cur.Advance(n)
	}

	// Unterminated — emit everything as cdata
	flushChunk()
}

// peekRuneToken returns the next whole UTF-8 rune at the cursor as a string
// together with its byte length, without advancing. A byte that is not a valid
// UTF-8 lead/sequence is returned as a single byte (length 1) so the scan
// always makes progress and the caller never emits a partial rune. A validly
// encoded U+FFFD is returned whole (length 3), unlike a lone bad byte. Callers
// must have already confirmed at least one byte is available (not Done).
func (p *parser) peekRuneToken() (string, int) {
	b := p.cur.Peek()
	if b < 0x80 {
		return string([]byte{b}), 1
	}
	// Peek up to utf8.UTFMax bytes and decode. DecodeRuneInString reports a
	// size of 1 for an invalid sequence and the true size for a valid rune
	// (including a genuine U+FFFD), so the size distinguishes the two cases.
	s := p.cur.PeekString(utf8.UTFMax)
	if s == "" {
		// Fewer than UTFMax bytes remain near EOF; peek whatever is left.
		for n := utf8.UTFMax - 1; n >= 1; n-- {
			if s = p.cur.PeekString(n); s != "" {
				break
			}
		}
	}
	if s == "" {
		return string([]byte{b}), 1
	}
	_, size := utf8.DecodeRuneInString(s)
	if size <= 0 {
		size = 1
	}
	return s[:size], size
}

// isUTF8Continuation reports whether b is a UTF-8 continuation byte
// (0b10xxxxxx). A rune boundary is any byte that is not a continuation byte, so
// backing a byte index off continuation bytes lands on a whole-rune boundary.
func isUTF8Continuation(b byte) bool {
	return b&0xC0 == 0x80
}

// parseRCDATAContent parses RCDATA content (title, textarea).
// Like raw text but entities are expanded.
func (p *parser) parseRCDATAContent(ctx context.Context, tagName string) {
	endTag := "</" + tagName
	limit := p.cfg.contentLimit()

	for !p.cur.Done() {
		// Abort promptly on context cancellation. The main parse loop
		// re-checks ctx.Err() and surfaces it.
		if ctx.Err() != nil {
			return
		}
		if p.cur.Peek() == '<' && p.cur.PeekAt(1) == '/' {
			if p.hasPrefixFold(endTag) {
				afterTag := len(endTag)
				// True EOF after the matched tag terminates the element; a real
				// NUL byte (HasByteAt true, PeekAt 0) does not.
				if !p.cur.HasByteAt(afterTag) {
					return
				}
				switch p.cur.PeekAt(afterTag) {
				case '>', ' ', '\t', '\n', '\r':
					return
				}
			}
		}

		// A real U+0000 (NUL) byte reads as 0, the same sentinel used to stop
		// the text scan below, so without this guard the scan makes no progress
		// and the loop spins forever. Per HTML5 the RCDATA state replaces U+0000
		// with U+FFFD. EOF is distinguished by Done() in the loop condition.
		if p.cur.Peek() == 0 {
			_ = p.cur.Advance(1)
			_ = p.emitCharacters([]byte("�"))
			continue
		}

		if p.cur.Peek() == '&' {
			p.parseCharRefBounded(ctx, limit)
			// A char-ref over the content cap sets fatalErr; stop scanning
			// immediately so the main loop surfaces it instead of running on.
			if p.fatalErr != nil {
				return
			}
		} else {
			// Collect text up to next & or potential end tag, but cap the run
			// at the content limit so one huge text span is emitted in bounded
			// chunks instead of buffered whole.
			n := 0
			for {
				b := p.cur.PeekAt(n)
				if b == 0 || b == '&' || b == '<' {
					break
				}
				n++
				if n >= limit {
					break
				}
			}
			// The cap may land mid-rune. Back off to the last whole-rune
			// boundary so the emitted chunk is valid UTF-8 — but only while a
			// complete rune still precedes the boundary. If the run begins with
			// a single rune that is itself larger than the cap, keep extending
			// until that rune is whole rather than splitting it.
			if n >= limit {
				for n > 0 && isUTF8Continuation(p.cur.PeekAt(n)) {
					n--
				}
				if n == 0 {
					// A lone rune exceeds the cap. Extend to cover it whole so a
					// partial rune is never emitted.
					n = limit
					for p.cur.HasByteAt(n) && isUTF8Continuation(p.cur.PeekAt(n)) {
						n++
					}
				}
			}
			if n > 0 {
				text := p.cur.PeekString(n)
				_ = p.cur.Advance(n)
				_ = p.emitCharacters([]byte(text))
			}
			if !p.cur.Done() && p.cur.Peek() == '<' {
				// Only stop on a *valid* end tag: matched prefix followed by a
				// valid terminator char. A matched-prefix-but-invalid tag such
				// as "</titlex" must NOT be treated as the end tag; otherwise
				// the '<' would be neither emitted nor advanced and the
				// for-loop would spin forever. Mirror the RAWTEXT validEnd
				// check, and always advance past '<' when it is not a valid end
				// tag so the cursor is guaranteed to progress.
				validEnd := false
				if p.cur.PeekAt(1) == '/' && p.hasPrefixFold(endTag) {
					switch {
					case !p.cur.HasByteAt(len(endTag)):
						// True EOF after the matched tag; a real NUL does not
						// count as a valid end-tag terminator.
						validEnd = true
					default:
						switch p.cur.PeekAt(len(endTag)) {
						case '>', ' ', '\t', '\n', '\r':
							validEnd = true
						}
					}
				}
				if !validEnd {
					_ = p.emitCharacters([]byte("<"))
					_ = p.cur.Advance(1)
				}
			}
		}
	}
}

// parsePlaintext parses plaintext content — everything until EOF.
func (p *parser) parsePlaintext(ctx context.Context) {
	limit := p.cfg.contentLimit()
	var content bytes.Buffer
	flushChunk := func() {
		// Clone the bytes before Reset: bytes.Buffer.Reset reuses the same
		// backing array, so a SAX handler that retains the slice would see
		// this chunk overwritten by subsequent content.
		if content.Len() > 0 {
			p.handleSAXErr(p.sax.Characters(bytes.Clone(content.Bytes())))
			content.Reset()
		}
	}
	// appendToken writes a whole rune to the buffer, flushing first if it would
	// push the chunk past the cap. Flushing on rune boundaries keeps every
	// emitted chunk valid UTF-8 (no multi-byte rune split across chunks); a
	// single rune larger than the cap is emitted whole rather than split.
	appendToken := func(s string) {
		if content.Len() > 0 && content.Len()+len(s) > limit {
			flushChunk()
		}
		content.WriteString(s)
	}
	for !p.cur.Done() {
		// Abort promptly on context cancellation rather than buffering the
		// entire (possibly endless) plaintext section first.
		if ctx.Err() != nil {
			flushChunk()
			return
		}
		// A real U+0000 (NUL) byte reads as 0, which the previous PeekAt-based
		// scan treated as EOF, truncating plaintext early. Per HTML5 the
		// PLAINTEXT state replaces U+0000 with U+FFFD; consume the rest of the
		// input verbatim, distinguishing genuine EOF via Done().
		if p.cur.Peek() == 0 {
			appendToken("�")
			_ = p.cur.Advance(1)
			continue
		}
		s, n := p.peekRuneToken()
		appendToken(s)
		_ = p.cur.Advance(n)
	}
	flushChunk()
}

// parseName parses an HTML tag name (letters, digits, colons, hyphens).
func (p *parser) parseName() string {
	n := 0
	for {
		b := p.cur.PeekAt(n)
		if b == 0 || !isNameChar(b) {
			break
		}
		n++
	}
	if n == 0 {
		return ""
	}
	name := p.cur.PeekString(n)
	_ = p.cur.Advance(n)
	return name
}

// parseAttributes parses HTML tag attributes.
// Duplicate attribute names are silently dropped (first occurrence wins),
// matching libxml2's htmlParseStartTag behavior.
func (p *parser) parseAttributes() []Attribute {
	var attrs []Attribute
	var seen map[string]struct{}

	for {
		p.skipWhitespace()
		if p.cur.Done() || p.cur.Peek() == '>' || p.cur.Peek() == '/' {
			break
		}

		name := p.parseAttrName()
		if name == "" {
			// Skip unknown character
			_ = p.cur.Advance(1)
			continue
		}

		name = strings.ToLower(name)
		p.skipWhitespace()

		value := ""
		isBool := false
		if p.cur.Peek() == '=' {
			_ = p.cur.Advance(1) // skip '='
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
	n := 0
	for {
		b := p.cur.PeekAt(n)
		if b == 0 || isWhitespaceByte(b) || b == '>' || b == '/' || b == '=' || b == '"' || b == '\'' || b == '<' {
			break
		}
		n++
	}
	if n == 0 {
		return ""
	}
	name := p.cur.PeekString(n)
	_ = p.cur.Advance(n)
	return name
}

// parseAttrValue parses an attribute value (quoted or unquoted).
func (p *parser) parseAttrValue() string {
	if p.cur.Peek() == '"' {
		return p.parseQuotedAttrValue('"')
	}
	if p.cur.Peek() == '\'' {
		return p.parseQuotedAttrValue('\'')
	}
	// Unquoted attribute value
	return p.parseUnquotedAttrValue()
}

// parseQuotedAttrValue parses a quoted attribute value with entity expansion.
func (p *parser) parseQuotedAttrValue(quote byte) string {
	_ = p.cur.Advance(1) // skip opening quote
	var buf bytes.Buffer

	for !p.cur.Done() && p.cur.Peek() != quote {
		if p.cur.Peek() == '&' {
			buf.WriteString(p.resolveEntityInAttr())
		} else {
			buf.WriteByte(p.cur.Peek())
			_ = p.cur.Advance(1)
		}
	}
	if p.cur.Peek() == quote {
		_ = p.cur.Advance(1) // skip closing quote
	}
	return buf.String()
}

// parseUnquotedAttrValue parses an unquoted attribute value.
func (p *parser) parseUnquotedAttrValue() string {
	var buf bytes.Buffer

	for !p.cur.Done() {
		b := p.cur.Peek()
		if b == '>' || b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\f' {
			break
		}
		if b == '&' {
			buf.WriteString(p.resolveEntityInAttr())
		} else {
			buf.WriteByte(b)
			_ = p.cur.Advance(1)
		}
	}
	return buf.String()
}

// resolveEntityInAttr resolves an entity reference inside an attribute value.
// In HTML, named entities without a trailing ';' are NOT resolved when followed
// by '=' or an alphanumeric character (prevents mis-interpreting URL query strings).
func (p *parser) resolveEntityInAttr() string {
	_ = p.cur.Advance(1) // skip '&'

	if p.cur.Peek() == '#' {
		_ = p.cur.Advance(1)
		var codepoint int
		var haveDigits bool
		if p.cur.Peek() == 'x' || p.cur.Peek() == 'X' {
			_ = p.cur.Advance(1)
			hexStr := p.parseWhile(isHexDigit)
			codepoint, haveDigits = parseNumericCharRef(hexStr, 16)
		} else {
			numStr := p.parseWhile(isDigit)
			codepoint, haveDigits = parseNumericCharRef(numStr, 10)
		}
		if p.cur.Peek() == ';' {
			_ = p.cur.Advance(1)
		}
		// Per HTML5, &#0; / &#x0; in an attribute value maps to U+FFFD rather
		// than being dropped. Emit nothing only for a bare "&#" with no digits.
		if haveDigits {
			return string(normalizeNumericCharRef(codepoint))
		}
		return ""
	}

	name := p.parseWhile(isAlphanumeric)
	hasSemicolon := false
	if p.cur.Peek() == ';' {
		hasSemicolon = true
		_ = p.cur.Advance(1)
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
			next := p.cur.Peek()
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
	if p.cur.Peek() != '"' && p.cur.Peek() != '\'' {
		return ""
	}
	quote := p.cur.Peek()
	_ = p.cur.Advance(1)
	n := 0
	for {
		b := p.cur.PeekAt(n)
		if b == 0 || b == quote {
			break
		}
		n++
	}
	s := p.cur.PeekString(n)
	_ = p.cur.Advance(n)
	if p.cur.Peek() == quote {
		_ = p.cur.Advance(1)
	}
	return s
}

// parseWhile collects characters while pred returns true.
func (p *parser) parseWhile(pred func(byte) bool) string {
	n := 0
	for {
		b := p.cur.PeekAt(n)
		if b == 0 || !pred(b) {
			break
		}
		n++
	}
	if n == 0 {
		return ""
	}
	s := p.cur.PeekString(n)
	_ = p.cur.Advance(n)
	return s
}

// parseWhileMax is parseWhile with an upper bound: it consumes at most limit
// matching bytes, leaving any further matching bytes for the next call. It lets
// a caller bound an otherwise unbounded scan (e.g. a runaway entity name) and
// drain it in fixed-size pieces. A limit <= 0 is treated as unbounded.
func (p *parser) parseWhileMax(pred func(byte) bool, limit int) string {
	n := 0
	for limit <= 0 || n < limit {
		b := p.cur.PeekAt(n)
		if b == 0 || !pred(b) {
			break
		}
		n++
	}
	if n == 0 {
		return ""
	}
	s := p.cur.PeekString(n)
	_ = p.cur.Advance(n)
	return s
}

// skipWhitespace skips whitespace characters.
func (p *parser) skipWhitespace() {
	n := 0
	for {
		b := p.cur.PeekAt(n)
		if b != ' ' && b != '\t' && b != '\n' && b != '\r' && b != '\f' {
			break
		}
		n++
	}
	if n > 0 {
		_ = p.cur.Advance(n)
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
