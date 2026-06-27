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
	"github.com/lestrrat-go/helium/internal/xmlchar"
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

	// sawRoot records that the root <html> element has been opened at least once.
	// It distinguishes genuine PRE-root whitespace (empty stack, root never
	// opened — drop as ignorable) from TRAILING whitespace AFTER the root has
	// closed (empty stack, but root was opened — still emit). The insertion mode
	// alone cannot make this distinction: opening a bare <html> does not advance
	// mode past insertInitial, so `<html></html> \n` would otherwise mis-classify
	// the trailing run as pre-root and drop it. Set once in pushName(elemHTML).
	sawRoot bool

	// curTextRunSignificant records that the current normal data-state text run
	// has already emitted a non-whitespace byte and is therefore significant. A
	// run is the maximal sequence of character data — plain text, entity /
	// numeric char-ref output, U+FFFD, and lone literal '<' — between two real
	// markup tags. Once set it BYPASSES whitespace suppression so a
	// trailing/interior whitespace chunk of a known-significant run still emits.
	//
	// It is SET in exactly one place — emitCharacters, on any non-whitespace emit
	// — so EVERY emit path marks the run uniformly. It is RESET in exactly one
	// place: the main parse loop, immediately before dispatching to a real markup
	// tag (start/end tag, comment, DOCTYPE, bogus comment, PI). A char-ref and a
	// lone '<' are part of the same run and never reset it.
	curTextRunSignificant bool

	// pendingWS holds the LEADING whitespace of the current normal data-state run
	// that has NOT yet been committed, because two decisions can only be made once
	// the run's first non-whitespace byte is seen:
	//
	//   1. Significance (StripBlanks/noBlanks): a run is stripped only when EVERY
	//      byte is whitespace. A leading whitespace prefix must therefore not be
	//      suppressed on its own — a following non-whitespace byte (plain text,
	//      char-ref output, U+FFFD, lone '<') makes the whole run significant and
	//      the leading whitespace part of it.
	//   2. Insertion target (implied <body>): a run containing non-whitespace
	//      triggers htmlStartCharData, which opens the implied <body>. Emitting the
	//      leading whitespace BEFORE that runs would land it under <html> while the
	//      following text lands under <body>, splitting one logical run across two
	//      parents.
	//
	// So a still-undecided leading whitespace prefix is accumulated here instead of
	// being emitted. When the first non-whitespace byte arrives, emitCharacters
	// flushes it into the now-significant run (after the caller has established the
	// insertion target); when the run ends all-whitespace, flushPendingWSRunEnd
	// strips it (noBlanks) or emits it under the current element. char-data tokens
	// ('&', NUL, lone '<') do NOT flush it — they are folded into the same run. The
	// buffer is bounded by the content cap: a whitespace prefix that overruns the
	// cap before any significance is known hard-fails with ErrContentSizeExceeded
	// rather than buffering unbounded. Whitespace is only deferred while a decision
	// is genuinely pending: under StripBlanks (significance undecided) or before the
	// body subtree is entered (implied-<body> insertion undecided). With StripBlanks
	// OFF and the insertion target already established, in <head>, before the root
	// element, and inside raw-text/RCDATA elements, whitespace is committed directly
	// (its target is already fixed, it is ignorable, or it is significant under the
	// soft cap), so it never enters this buffer.
	pendingWS []byte

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
	// final detected encoding name. (An undeclared stream that stays valid
	// UTF-8 past the buffering cap fails closed with a bounded-input error
	// rather than committing to UTF-8, so the deferred reader never emits a
	// sanitized U+FFFD of its own.)
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
	// Detect the declared charset from the RAW input (before newline
	// normalization), bounded to the first 1024 raw bytes by extractMetaCharset.
	// The streaming ParseReader/push path prescans the first 1024 RAW bytes
	// (wrapReaderForHTML reads its sniff window straight off the reader, BEFORE
	// chaining the newlineNormReader), so Parse must detect against the same raw
	// window. Detecting against `normalized` instead would let a CRLF-heavy head
	// (each \r\n collapsing to \n) pull a <meta charset=...> that sits PAST raw
	// byte 1024 INTO the post-normalization window — Parse would honor a
	// declaration that ParseReader/push never sees, a parity divergence.
	declared := extractMetaCharset(input)

	// Normalize \r\n → \n and standalone \r → \n (HTML spec line normalization)
	normalized := normalizeNewlines(input)

	var encodingErr bool
	var encErrLine, encErrCol int
	var detectedEnc string
	switch {
	case declared == "iso-8859-1":
		// An explicit charset=iso-8859-1 commits to Latin-1 BEFORE the utf8.Valid
		// check below: a declared Latin-1 document whose bytes happen to be valid
		// UTF-8 must still decode as Latin-1, exactly as the streaming ParseReader
		// path (which routes a declared Latin-1 head straight to latin1Reader)
		// does. Without this hoist the two APIs would diverge on such an input.
		detectedEnc = encISO88591
		normalized = latin1ToUTF8(normalized)
	case utf8.Valid(normalized):
		// Already valid UTF-8 with no Latin-1 declaration: parse as-is.
	case declared == "utf-8":
		raw := normalized
		var invBytes invalidByteInfo
		normalized, encodingErr = replaceInvalidUTF8(raw, &invBytes)
		if encodingErr {
			encErrLine, encErrCol = lineColFromOffset(raw, invBytes.offset)
		}
	default:
		// Undeclared non-UTF-8: assume Windows-1252 and convert to UTF-8,
		// matching libxml2's default behavior for non-UTF-8 documents.
		detectedEnc = encWindows1252
		normalized = latin1ToUTF8(normalized)
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
	wrapped, detectedEnc, sanitizer, deferred := wrapReaderForHTML(r, cfg.contentLimit())

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

// isASCIIWhitespace reports whether b is one of the WHATWG ASCII whitespace
// characters (tab, LF, FF, CR, space).
func isASCIIWhitespace(b byte) bool {
	switch b {
	case '\t', '\n', '\f', '\r', ' ':
		return true
	default:
		return false
	}
}

// extractDeclaredCharset scans the raw (possibly invalid) input bytes for a
// charset declaration (as found in <meta charset=...> or in a <meta
// http-equiv="content-type" content="...; charset=..."> attribute) and returns
// the declared encoding name, lowercased, or "" if none is found.
//
// The value syntax mirrors the WHATWG "extracting a character encoding from a
// meta element" algorithm: ASCII whitespace may surround the "=", and the value
// may be wrapped in single or double quotes, or left unquoted (terminated by
// whitespace or ";").
func extractDeclaredCharset(data []byte) string {
	// Bound to the first 1024 RAW bytes BEFORE case folding so the lowercase
	// copy is never larger than the prescan window — folding the whole input
	// just to inspect its head would allocate proportional to the document
	// (and bytes.ToLower can expand invalid UTF-8). The charset appears early.
	if len(data) > 1024 {
		data = data[:1024]
	}
	lower := bytes.ToLower(data)
	const key = "charset"
	from := 0
	for {
		idx := bytes.Index(lower[from:], []byte(key))
		if idx < 0 {
			return ""
		}
		pos := from + idx + len(key)
		from = pos
		// Skip ASCII whitespace before '='.
		for pos < len(lower) && isASCIIWhitespace(lower[pos]) {
			pos++
		}
		// Require '='.
		if pos >= len(lower) || lower[pos] != '=' {
			continue
		}
		pos++
		// Skip ASCII whitespace after '='.
		for pos < len(lower) && isASCIIWhitespace(lower[pos]) {
			pos++
		}
		if pos >= len(lower) {
			return ""
		}
		// Quoted value: read up to the matching quote.
		if q := lower[pos]; q == '"' || q == '\'' {
			pos++
			end := bytes.IndexByte(lower[pos:], q)
			if end < 0 {
				return ""
			}
			return string(lower[pos : pos+end])
		}
		// Unquoted value: runs until ASCII whitespace, ';', tag close '>',
		// an enclosing quote, or end. (The crude prescan operates on raw
		// markup, so the tag terminator must bound a bare value like in
		// <meta charset=utf-8>. The quote terminators handle a value that
		// sits inside a still-quoted content attribute, as in
		// <meta http-equiv="Content-Type" content="text/html; charset=utf-8">,
		// where the char after "charset=" is not a quote but the value is
		// nonetheless bounded by the enclosing attribute quote.)
		start := pos
		for pos < len(lower) && !isASCIIWhitespace(lower[pos]) && lower[pos] != ';' && lower[pos] != '>' && lower[pos] != '"' && lower[pos] != '\'' {
			pos++
		}
		return string(lower[start:pos])
	}
}

// extractMetaCharset scans the input for a charset declared by a REAL HTML
// <meta> element — either <meta charset=...> or a <meta http-equiv=...
// content="...; charset=..."> — within the first 1024 bytes, returning the
// declared encoding name lowercased, or "" if none is found.
//
// This honors ONLY an actual meta element: a "charset=iso-8859-1" string that
// merely appears in ordinary text, an HTML comment, or a script does NOT count.
// Within a genuine <meta ...> tag it further honors only a real declaration per
// WHATWG (metaCharsetFromTag): a `charset` attribute, or a `content` attribute's
// charset when paired with `http-equiv="content-type"`. A `charset=` token in any
// other attribute (data-charset, name, a non-pragma content value, ...) is
// ignored. This precision matters because the eager Latin-1/UTF-8 encoding commit
// overrides utf8.Valid — a false positive there would corrupt a valid UTF-8
// document (café → cafÃ©). It mirrors the WHATWG "prescan a byte stream to
// determine its encoding" algorithm: comments are skipped, other tags are stepped
// over, and charset extraction parses the attributes of a genuine <meta ...> tag.
func extractMetaCharset(data []byte) string {
	// Bound to the first 1024 RAW bytes BEFORE case folding. newParser calls
	// this (via declaredCharsetIs{Latin1,UTF8}) ahead of utf8.Valid on every
	// in-memory Parse([]byte), so lowercasing the whole document just to look
	// at its head would allocate proportional to the input on the hot path
	// (and bytes.ToLower can expand invalid UTF-8). The work is bounded to the
	// prescan window regardless of input size.
	if len(data) > 1024 {
		data = data[:1024]
	}
	lower := bytes.ToLower(data)
	n := len(lower)
	for i := 0; i < n; {
		if lower[i] != '<' {
			i++
			continue
		}
		// Mirror the main parser's char-data rule (parser.go ~888): a '<' begins
		// markup ONLY when the byte after it is '/', '!', '?', or an ASCII letter.
		// Any other following byte makes the '<' literal character data (e.g.
		// `< " >` or `<x="...`). Treat it as char data — advance exactly ONE byte
		// WITHOUT entering the quote-aware tag-skip below. Otherwise a stray
		// non-markup '<' carrying a quote could put metaTagEnd into quote state and
		// swallow a later genuine <meta charset=...> tag, missing the declaration.
		var nextB byte
		if i+1 < n {
			nextB = lower[i+1]
		}
		if nextB != '/' && nextB != '!' && nextB != '?' && !isASCIIAlpha(nextB) {
			i++
			continue
		}
		// Comment: skip past the closing "-->".
		if bytes.HasPrefix(lower[i:], []byte("<!--")) {
			end := bytes.Index(lower[i+4:], []byte("-->"))
			if end < 0 {
				return ""
			}
			i += 4 + end + 3
			continue
		}
		// <meta followed by ASCII whitespace or '/': inspect its attributes only.
		if i+5 <= n && bytes.Equal(lower[i+1:i+5], []byte("meta")) &&
			(i+5 == n || isASCIIWhitespace(lower[i+5]) || lower[i+5] == '/') {
			tag := lower[i:]
			// Bound the tag at its first UNQUOTED '>'. A '>' inside a quoted
			// attribute value (e.g. <meta data-x=">" charset=iso-8859-1>) is
			// part of the value, not the tag terminator, so a naive IndexByte
			// would truncate the tag before charset= and miss the declaration.
			gt := metaTagEnd(tag)
			if gt < 0 {
				// No terminating unquoted '>' within the prescan window: the
				// tag is cut off by the window end. Per WHATWG, running out of
				// bytes aborts the prescan — a declaration severed by the
				// window must NOT count, so do not parse the truncated tag.
				return ""
			}
			tag = tag[:gt]
			if cs := metaCharsetFromTag(tag); cs != "" {
				return cs
			}
			i += gt + 1
			continue
		}
		// End tag, processing instruction, or markup declaration (</..., <?...,
		// <!... other than the <!-- comment handled above): per WHATWG these are
		// skipped to the next RAW '>'. Quotes are NOT significant inside them, so
		// the quote-aware scan must NOT be used — a stray quote inside one could
		// otherwise enter quote state and swallow a later genuine <meta charset=>.
		if nextB == '/' || nextB == '!' || nextB == '?' {
			rel := bytes.IndexByte(lower[i:], '>')
			if rel < 0 {
				return ""
			}
			i += rel + 1
			continue
		}
		// Any other start tag (<name ...>): step over it to its first UNQUOTED
		// '>' (a '>' inside a quoted attribute value is not the terminator).
		gt := metaTagEnd(lower[i:])
		if gt < 0 {
			return ""
		}
		i += gt + 1
	}
	return ""
}

// metaTagEnd returns the index of the byte that terminates the tag starting at
// tag[0] (its first UNQUOTED '>'), or -1 if no unquoted '>' is present. It
// tracks single/double quote state while scanning so a '>' that sits inside a
// quoted attribute value (e.g. <meta data-x=">" charset=iso-8859-1>) does not
// prematurely terminate the tag. The scan stays within the caller's bounded
// prescan window.
func metaTagEnd(tag []byte) int {
	var quote byte // 0 = unquoted, '"' or '\'' inside a value
	for j := range tag {
		c := tag[j]
		if quote != 0 {
			if c == quote {
				quote = 0
			}
			continue
		}
		switch c {
		case '"', '\'':
			quote = c
		case '>':
			return j
		}
	}
	return -1
}

// metaCharsetFromTag applies the WHATWG meta-element prescan attribute rules to
// the (already lowercased) bytes of a single tag that begins with "<meta". It
// parses the tag's attributes and returns a declared encoding ONLY when it comes
// from a real `charset` attribute, or from a `content` attribute paired with an
// `http-equiv="content-type"` attribute on the same element. A `charset=` token
// appearing in any other attribute (data-charset, name, a non-pragma content
// value, ...) is ignored. Returns "" when no qualifying declaration is present.
//
// This mirrors the WHATWG "prescan a byte stream" per-meta decision: track a
// candidate charset, whether a content-type pragma was seen (gotPragma), and
// whether the candidate requires that pragma to be trusted (needPragma). A
// `charset` attribute is always trusted; a `content` charset is trusted only when
// the pragma is also present.
func metaCharsetFromTag(tag []byte) string {
	pos := len("<meta")
	var charset string
	gotPragma := false
	// needPragma: 0 = unset (no candidate yet), 1 = content candidate (needs the
	// content-type pragma), 2 = charset attribute (trusted unconditionally).
	needPragma := 0
	// WHATWG keeps a list of attribute names already seen on this element and
	// IGNORES any duplicate name (first occurrence wins). Without this, a later
	// duplicate would override an earlier one — e.g.
	// `<meta charset=utf-8 charset=iso-8859-1>` would wrongly commit to Latin-1
	// and corrupt valid UTF-8.
	var seen map[string]struct{}
	for {
		name, value, next, ok := metaNextAttr(tag, pos)
		if !ok {
			break
		}
		pos = next
		nm := string(name)
		if _, dup := seen[nm]; dup {
			continue
		}
		if seen == nil {
			seen = make(map[string]struct{}, 4)
		}
		seen[nm] = struct{}{}
		switch nm {
		case "http-equiv":
			if string(value) == "content-type" {
				gotPragma = true
			}
		case "content":
			if charset == "" {
				if cs := extractDeclaredCharset(value); cs != "" {
					charset = cs
					needPragma = 1
				}
			}
		case "charset":
			if len(value) > 0 {
				charset = string(value)
				needPragma = 2
			}
		}
	}
	if needPragma == 0 {
		return ""
	}
	if needPragma == 1 && !gotPragma {
		return ""
	}
	return charset
}

// metaNextAttr implements the WHATWG "get an attribute" sub-algorithm of the meta
// prescan over the (already lowercased) tag bytes starting at pos. It returns the
// next attribute's name and value (sub-slices of tag; value is nil when the
// attribute has no value), the position to resume scanning, and ok=false when no
// further attribute is present. Quote-aware: a quoted value runs to its matching
// quote; an unquoted value runs to ASCII whitespace or the tag-end '>'.
func metaNextAttr(tag []byte, pos int) (name, value []byte, next int, ok bool) {
	n := len(tag)
	// Skip leading ASCII whitespace and '/' separators.
	for pos < n && (isASCIIWhitespace(tag[pos]) || tag[pos] == '/') {
		pos++
	}
	if pos >= n || tag[pos] == '>' {
		return nil, nil, pos, false
	}
	// Read the attribute name. A '=' terminates the name only when the name is
	// non-empty (a leading '=' is part of the name, per WHATWG).
	nameStart := pos
	for pos < n {
		c := tag[pos]
		if c == '=' && pos > nameStart {
			break
		}
		if isASCIIWhitespace(c) || c == '/' || c == '>' {
			break
		}
		pos++
	}
	name = tag[nameStart:pos]
	// Skip whitespace between the name and a possible '='.
	for pos < n && isASCIIWhitespace(tag[pos]) {
		pos++
	}
	if pos >= n || tag[pos] != '=' {
		return name, nil, pos, true
	}
	pos++ // consume '='
	// Skip whitespace after '='.
	for pos < n && isASCIIWhitespace(tag[pos]) {
		pos++
	}
	if pos >= n || tag[pos] == '>' {
		return name, nil, pos, true
	}
	if q := tag[pos]; q == '"' || q == '\'' {
		pos++
		valStart := pos
		for pos < n && tag[pos] != q {
			pos++
		}
		value = tag[valStart:pos]
		if pos < n {
			pos++ // consume the closing quote
		}
		return name, value, pos, true
	}
	valStart := pos
	for pos < n && !isASCIIWhitespace(tag[pos]) && tag[pos] != '>' {
		pos++
	}
	return name, tag[valStart:pos], pos, true
}

// declaredCharsetIsUTF8 reports whether a real <meta> element declares utf-8.
// It uses extractMetaCharset (a precise meta-element prescan), NOT a loose
// charset= scan, because its callers eagerly commit to the encoding and override
// utf8.Valid.
func declaredCharsetIsUTF8(data []byte) bool {
	return extractMetaCharset(data) == "utf-8"
}

// declaredCharsetIsLatin1 reports whether a real <meta> element declares
// iso-8859-1. This distinguishes documents that genuinely declare ISO-8859-1
// from those that are just auto-detected as non-UTF-8. It uses extractMetaCharset
// (a precise meta-element prescan), NOT a loose charset= scan, because its
// callers eagerly commit to Latin-1 and override utf8.Valid — trusting a stray
// "charset=iso-8859-1" in text/comment/script would corrupt a valid UTF-8
// document.
func declaredCharsetIsLatin1(data []byte) bool {
	return extractMetaCharset(data) == "iso-8859-1"
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

	// Check ctx/fatalErr/read-error BEFORE Done() so a condition set during the
	// previous iteration aborts immediately. Done() refills the buffer with a
	// blocking Read; checking afterward would let an over-cap fatalErr (or a
	// cancellation) trigger one more blocking read before the parse returns.
	for {
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
		if p.cur.Done() {
			break
		}
		if p.cur.Peek() == '<' {
			// Only an actual markup-tag dispatch ends the current normal-data text
			// run. A lone '<' that is not markup is ordinary character data
			// belonging to the SAME run, so it must NOT end the run (emitCharacters
			// marks the run significant via the '<' byte).
			next := p.cur.PeekAt(1)
			if next == '/' || next == '!' || next == '?' || isASCIIAlpha(next) {
				// A real markup tag ends the current normal-data text run: flush any
				// deferred all-whitespace leading run (it never became significant)
				// and forget the run's significance before dispatching.
				_ = p.flushPendingWSRunEnd()
				p.curTextRunSignificant = false
				switch next {
				case '/':
					p.parseEndTag()
				case '!':
					if p.cur.PeekAt(2) == '-' && p.cur.PeekAt(3) == '-' {
						p.parseComment(ctx)
					} else if p.hasPrefixFold("<!DOCTYPE") {
						p.parseDoctype()
					} else {
						// Bogus comment or similar — treat as comment
						p.parseBogusComment(ctx)
					}
				case '?':
					// Processing instruction — in HTML mode, treated as comment
					p.parsePI(ctx)
				default:
					// isASCIIAlpha(next): a start tag.
					p.parseStartTag(ctx)
				}
			} else {
				// Lone '<' — ordinary character data, part of the current text run.
				// Like the other non-whitespace emit paths (plain text, char-refs,
				// U+FFFD), establish the insertion target FIRST via htmlStartCharData
				// so this run's leading whitespace (held in pendingWS) and the '<' land
				// under the SAME element, then emit. Without it, for `<html> < b` the
				// leading space and '<' would flush under the pre-body target while the
				// following non-whitespace opens the implied <body>, splitting one
				// logical run across parents.
				p.htmlStartCharData()
				_ = p.emitCharacters([]byte("<"))
				_ = p.cur.Advance(1)
			}
		} else {
			p.parseCharacters(ctx)
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

	// A normal-data run that ended at EOF still all-whitespace has its leading
	// whitespace deferred in pendingWS; resolve it before closing open elements.
	_ = p.flushPendingWSRunEnd()

	p.htmlAutoCloseOnEnd()
	p.handleSAXErr(p.sax.EndDocument())
	return p.fatalSAXErr
}

// parseStartTag parses an HTML start tag: <tagname attrs...>
func (p *parser) parseStartTag(ctx context.Context) {
	_ = p.cur.Advance(1) // skip '<'

	name := p.parseName()
	// An over-cap tag name is an unrecoverable hard-cap failure, not a merely
	// invalid name: parseName set fatalErr and returned "". Return BEFORE the
	// name == "" text fallback so we never publish a stray '<' text node ahead of
	// the fatal. The main loop surfaces fatalErr.
	if p.fatalErr != nil {
		return
	}
	if name == "" {
		// Not a valid tag, emit '<' as text
		_ = p.emitCharacters([]byte("<"))
		return
	}

	name = strings.ToLower(name)

	// Parse attributes
	attrs := p.parseAttributes()

	// An over-cap attribute value is unrecoverable; abort without emitting a
	// truncated start element. The main loop surfaces fatalErr.
	if p.fatalErr != nil {
		return
	}

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
	// An over-cap end-tag name set fatalErr. Return IMMEDIATELY, before the
	// "skip to '>'" loop below, so an unterminated abusive stream is not drained
	// (read/work bound) before the main loop surfaces the fatal.
	if p.fatalErr != nil {
		return
	}
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

	// Bound the post-name intra-tag whitespace with the same HARD-capped
	// skipWhitespace the rest of the tag lexer uses, BEFORE the unbounded
	// "skip to '>'" drain loop below. Otherwise `</p` + an over-cap whitespace
	// run + `>` would be drained byte-by-byte without limit (an unbounded-scan
	// DoS), contradicting the documented intra-tag-whitespace hard cap. An
	// over-cap run sets fatalErr; check it and return PROMPTLY so an abusive
	// unterminated stream is not drained before the main loop surfaces the fatal
	// (mirrors the parseStartTag / parseDoctype fatal-check pattern).
	p.skipWhitespace()
	if p.fatalErr != nil {
		return
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

	// Each scanner below — skipWhitespace, parseName, parseQuotedString — can set
	// fatalErr on a hard-cap overflow (over-cap intra-tag whitespace, root name, or
	// PUBLIC/SYSTEM literal). Check fatalErr IMMEDIATELY after each one and return
	// before any further cursor read: on a streaming reader stalled right at the
	// over-cap boundary, the next scanner's PeekAt would issue another (blocking)
	// Read instead of promptly surfacing the fatal. The main loop surfaces fatalErr.
	p.skipWhitespace()
	if p.fatalErr != nil {
		return
	}

	// Parse root element name
	name := p.parseName()
	if p.fatalErr != nil {
		return
	}
	p.skipWhitespace()
	if p.fatalErr != nil {
		return
	}

	externalID := ""
	systemID := ""

	// Check for PUBLIC or SYSTEM
	if p.hasPrefixFold("PUBLIC") {
		_ = p.cur.Advance(6)
		p.skipWhitespace()
		if p.fatalErr != nil {
			return
		}
		externalID = p.parseQuotedString()
		if p.fatalErr != nil {
			return
		}
		p.skipWhitespace()
		if p.fatalErr != nil {
			return
		}
		systemID = p.parseQuotedString()
		if p.fatalErr != nil {
			return
		}
	} else if p.hasPrefixFold("SYSTEM") {
		_ = p.cur.Advance(6)
		p.skipWhitespace()
		if p.fatalErr != nil {
			return
		}
		systemID = p.parseQuotedString()
		if p.fatalErr != nil {
			return
		}
	}

	// Final gate before the "skip to '>'" drain loop and the InternalSubset SAX
	// emit (redundant given the per-scanner checks above, kept defensively): an
	// over-cap literal/name/whitespace must not drain an unterminated abusive
	// DOCTYPE or publish a partial subset.
	if p.fatalErr != nil {
		return
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
//
// Each call consumes one bounded chunk of a normal data-state text run and hands
// it to emitCharacters; the main parse loop re-enters once per chunk so one huge
// delimiter-free span is delivered to SAX in MaxContentSize-bounded pieces rather
// than buffered whole. The whitespace-significance and implied-<body> insertion
// decisions are NOT made here — they are centralized in emitCharacters, which
// defers a still-undecided leading whitespace prefix into pendingWS until the
// run's first non-whitespace byte is seen. parseCharacters therefore just scans
// and emits; it does not try to keep a leading whitespace prefix together with
// the following text.
func (p *parser) parseCharacters(ctx context.Context) {
	// Inside <head>, stop the scan at the first non-whitespace byte so leading
	// whitespace is emitted in <head> (its insertion target is already correct)
	// and the following non-whitespace re-enters and triggers head-close +
	// body-open via htmlStartCharData.
	inHead := p.currentName() == elemHead

	// A real U+0000 (NUL) byte is indistinguishable from EOF via Peek/PeekAt
	// (both return 0), so the scan loop below would break with no progress and
	// the outer parse loop would spin forever. Per HTML5 the data state treats
	// U+0000 as a parse error and replaces it with U+FFFD. Consume the NUL and
	// emit the replacement character, guaranteeing forward progress. EOF is
	// distinguished by Done().
	if !p.cur.Done() && p.cur.Peek() == 0 {
		_ = p.cur.Advance(1)
		// U+FFFD is non-whitespace: establish the insertion target, then emit. The
		// preceding leading whitespace (if any) is held in pendingWS and flushed by
		// emitCharacters into this now-significant run.
		p.htmlStartCharData()
		_ = p.emitCharacters([]byte("�"))
		return
	}

	limit := p.cfg.contentLimit()

	// Scan a run of ordinary character data up to the next char-data token
	// ('&', a real NUL, lone '<') or markup, bounded by the content cap.
	n := 0
	for {
		b := p.cur.PeekAt(n)
		if b == 0 || b == '<' || b == '&' {
			break
		}
		if inHead && !isWhitespaceByte(b) {
			break
		}
		n++
		if n >= limit {
			break
		}
	}
	// A cap-truncated run may land mid-rune; back off (or, for a single rune
	// larger than the cap, extend) to a whole-rune boundary so the emitted chunk
	// is valid UTF-8.
	n = p.clampTextChunkToRune(n, limit)

	if n > 0 {
		text := p.cur.PeekString(n)
		_ = p.cur.Advance(n)
		textBytes := []byte(text)
		// Non-whitespace establishes the run's insertion target before any byte is
		// emitted; emitCharacters then flushes any deferred leading whitespace into
		// the now-significant run. An all-whitespace chunk is deferred/suppressed by
		// emitCharacters without opening an implied <body>.
		if !isAllWhitespace(textBytes) {
			p.htmlStartCharData()
		}
		_ = p.emitCharacters(textBytes)
		return
	}

	// n == 0. Inside <head>, the scan above stops at the FIRST non-whitespace byte
	// (so leading whitespace is emitted in <head> first), which leaves nothing to
	// emit on the call that begins at that non-whitespace byte. Consume the
	// non-whitespace run here — without the in-head break — so the parse makes
	// forward progress; htmlStartCharData closes <head> and opens <body>. Outside
	// <head> the first scan already consumed any non-whitespace, so this only
	// triggers for the in-head case.
	if !p.cur.Done() && p.cur.Peek() != '<' && p.cur.Peek() != '&' {
		n = 0
		for {
			b := p.cur.PeekAt(n)
			if b == 0 || b == '<' || b == '&' {
				break
			}
			n++
			if n >= limit {
				break
			}
		}
		n = p.clampTextChunkToRune(n, limit)
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

	// At a char-data token. '&' starts a character reference. Use the cap-aware
	// variant so a long unresolved named/numeric reference is bounded (named via a
	// fixed lookahead, numeric via chunked digit consumption) exactly like the
	// RCDATA path, instead of buffering the whole run through unbounded parseWhile
	// scans. A char-ref is part of the SAME normal-data run, not a boundary: if it
	// emits non-whitespace, emitCharacters marks the run significant and flushes
	// any deferred leading whitespace; an all-whitespace resolution folds into
	// pendingWS. Only a real markup tag (handled in the main parse loop) ends it.
	if !p.cur.Done() && p.cur.Peek() == '&' {
		p.parseCharRefBounded(ctx, limit)
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

// scanNumericCharRef consumes a numeric character reference body, with the
// cursor positioned just past the leading '#'. It skips the '#', branches on an
// 'x'/'X' hex prefix versus a decimal run, parses the digits, and consumes an
// optional trailing ';'. It returns the resolved codepoint and whether any
// digits were present. Callers handle post-scan emission/normalization.
func (p *parser) scanNumericCharRef() (codepoint int, haveDigits bool) {
	_ = p.cur.Advance(1) // skip '#'
	// Bound the digit scan to maxEntityNameLen so a runaway '&#'-led digit run
	// cannot slurp unbounded into one allocation, bypassing the caller's per-byte
	// hard cap. Any over-cap tail falls through to the caller's capped loop; no
	// valid numeric reference needs this many digits.
	if p.cur.Peek() == 'x' || p.cur.Peek() == 'X' {
		_ = p.cur.Advance(1) // skip 'x'
		hexStr, _ := p.parseWhileMaxErr(xmlchar.IsHexDigit, maxEntityNameLen)
		codepoint, haveDigits = parseNumericCharRef(hexStr, 16)
	} else {
		numStr, _ := p.parseWhileMaxErr(xmlchar.IsASCIIDigit, maxEntityNameLen)
		codepoint, haveDigits = parseNumericCharRef(numStr, 10)
	}
	if p.cur.Peek() == ';' {
		_ = p.cur.Advance(1)
	}
	return codepoint, haveDigits
}

// maxEntityNameLen is one past the length of the longest named HTML entity
// ("CounterClockwiseContourIntegral", 31 chars). An alphanumeric run reaching
// this length cannot match any known entity, so the char-ref scan can stop and
// treat the run as unresolved literal text without consulting the entity table.
const maxEntityNameLen = 32

// saturatedCharRefChunk bounds how many bytes parseSaturatedCharRefLiteral
// consumes per iteration while spooling a saturated alphanumeric char-ref run.
// It caps both per-chunk buffering and the interval between context-cancellation
// and over-cap checks, independent of (and never larger than) MaxContentSize.
const saturatedCharRefChunk = 4096

// parseCharRefBounded handles an entity reference inside cap-aware content
// (the normal data state and RCDATA: title/textarea) where the surrounding
// text is flushed to SAX in chunks no larger than limit. Numeric references
// (including overlong/leading-zero/overflow runs) normalize to their HTML5
// output (U+FFFD on overflow/invalid), and named references resolve including
// legacy-prefix matching.
//
// Memory AND bytes-read work are kept bounded by two independent budgets:
//
//   - Entity-name resolution uses a FIXED lookahead of maxEntityNameLen bytes,
//     a constant independent of MaxContentSize. No known entity (longest 31
//     chars) or legacy prefix (≤6 chars) is longer, so a SHORT resolvable
//     reference whose run fits the cap is always decided within that constant
//     window — and therefore never rejected for being a small name. `&amp;`
//     resolves even under MaxContentSize(2).
//
//   - The MaxContentSize cap governs the LITERAL text that an UNRESOLVED run
//     would emit AND the work spent deciding an AMBIGUOUS legacy-prefix run.
//     An unbounded digit run is consumed in fixed-size chunks while tracking
//     value/overflow rather than buffered whole; an unresolved reference is
//     flushed as literal text in capped chunks; and a run that genuinely
//     EXCEEDS the cap before its outcome is decided fails the parse with
//     ErrContentSizeExceeded.
//
//     IMPORTANT — over-cap ambiguous legacy-prefix runs HARD-FAIL: `&amp` + an
//     alphanumeric tail is ambiguous until the run ends, because a trailing ';'
//     turns it into an over-long unknown LITERAL (no legacy resolution) while
//     its absence legacy-resolves the "amp" prefix and emits the tail. Settling
//     that requires reaching the run's end. To keep BOTH peak memory and
//     bytes-read bounded, the decision is made while CONSUMING the tail into a
//     spool capped at MaxContentSize: if the run exceeds the cap before the end
//     is reached, the parse hard-fails with ErrContentSizeExceeded and emits
//     NOTHING, rather than streaming an unbounded tail. A no-';' legacy-prefix
//     run is therefore resolved only when its whole run fits the cap.
//
// ctx is checked between bounded chunks so a cancelled parse aborts promptly
// without first draining a long digit or alphanumeric run.
func (p *parser) parseCharRefBounded(ctx context.Context, limit int) {
	// Entity content is non-whitespace — ensure implied elements.
	p.htmlStartCharData()

	_ = p.cur.Advance(1) // skip '&'

	if p.cur.Peek() == '#' {
		_ = p.cur.Advance(1) // skip '#'
		base := 10
		pred := xmlchar.IsASCIIDigit
		if p.cur.Peek() == 'x' || p.cur.Peek() == 'X' {
			base = 16
			pred = xmlchar.IsHexDigit
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
		semicolon := p.cur.Peek() == ';'
		// Peek may refill the buffer when the digit run ended at a buffer
		// boundary; abort without emitting if that hit a read error / cancel.
		if ctx.Err() != nil || p.cur.Err() != nil {
			return
		}
		if semicolon {
			_ = p.cur.Advance(1)
		}
		p.emitNumericCharRef(codepoint, haveDigits)
		return
	}

	// Named entity. Resolution uses a FIXED maxEntityNameLen-byte lookahead — a
	// constant, NOT MaxContentSize — because every known entity and legacy
	// prefix is shorter than that window, so this is O(1) memory regardless of
	// the user's cap.
	name, scanErr := p.parseWhileMaxErr(isAlphanumeric, maxEntityNameLen)
	// A short name scan is ambiguous: the name ended (next byte is not
	// alphanumeric, e.g. ';' or EOF) OR PeekAt/fillBuffer hit a read error /
	// cancellation. Disambiguate BEFORE resolving/emitting so a cancelled read
	// is never mistaken for a finished entity name that would then emit a
	// (partial) resolution or literal. The main loop re-checks ctx/p.cur.Err().
	if scanErr != nil || ctx.Err() != nil {
		return
	}

	// The lookahead SATURATES when the alphanumeric run reaches the fixed window
	// AND keeps going. The reference algorithm scans the WHOLE run before
	// deciding, so its
	// resolveNamedEntity sees the full (over-long) name; a long ';'-terminated
	// name is NOT legacy-prefix-resolved (that loop is gated on !hasSemicolon),
	// and an over-long no-semicolon name resolves only its longest legacy prefix.
	// Resolving from the truncated 32-byte window here — before knowing whether a
	// ';' eventually terminates the run — would wrongly legacy-resolve the prefix
	// of a ';'-terminated name. So when the run saturates the decision is DEFERRED
	// to parseSaturatedCharRefLiteral, which consumes the rest of the run into a
	// cap-bounded spool to learn whether a ';' terminates it. No known entity (≤31
	// chars) reaches the window, so a saturated run can never be a named-entity
	// match; the only resolution it could ever produce is a legacy prefix, and
	// that applies only when there is no ';' AND the whole run fits the cap (an
	// over-cap saturated run hard-fails — see that function).
	if len(name) >= maxEntityNameLen && isAlphanumeric(p.cur.Peek()) {
		p.parseSaturatedCharRefLiteral(ctx, name, limit)
		return
	}

	hasSemicolon := false
	semicolon := p.cur.Peek() == ';'
	// Peek may refill the buffer when the name ended at a buffer boundary; abort
	// without emitting if that hit a read error / cancellation.
	if ctx.Err() != nil || p.cur.Err() != nil {
		return
	}
	if semicolon {
		hasSemicolon = true
		_ = p.cur.Advance(1)
	}

	if val, remainder, ok := resolveNamedEntity(name, hasSemicolon); ok {
		// A ';'-terminated match is a KNOWN entity — a resolved character
		// reference that is always emitted intact and exempt from the cap (the
		// resolved value is one entity's worth of bytes within the fixed
		// lookahead). A NO-';' match is a LEGACY resolution (a full legacy
		// entity OR a legacy-prefix whose unmatched tail is echoed as literal
		// text); per the documented contract it is exempt ONLY when its whole
		// consumed run ("&" + name) fits the cap. Charge that run BEFORE emitting
		// so a legacy/legacy-prefix run over a tiny cap hard-fails with NOTHING
		// emitted, identical to the saturated path. (Example: `&ampZ` under
		// MaxContentSize(2) — the 5-byte run "&ampZ" exceeds 2, so it must fail
		// rather than emit "&" + "Z".)
		if !hasSemicolon {
			sizeCap := effectiveContentCap(limit)
			if 1+len(name) > sizeCap {
				p.fatalErr = fmt.Errorf("legacy character reference run exceeds %d bytes: %w", sizeCap, ErrContentSizeExceeded)
				return
			}
		}
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
	sizeCap := effectiveContentCap(limit)
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
// maxEntityNameLen bytes already CONSUMED and the run is known to continue. Such
// a run can never match a KNOWN entity (the longest is 31 chars), so the only
// resolution the reference algorithm could ever apply to it is a legacy-prefix match — and
// that applies ONLY when the run is NOT ';'-terminated (resolveNamedEntity gates
// its prefix loop on !hasSemicolon). The two interpretations differ:
//
//   - no ';': resolve the longest legacy prefix of the run (which lies entirely
//     within head, ≤6 chars) and emit the unmatched remainder (rest of head plus
//     the alphanumeric tail) as ordinary text. If head has no legacy prefix the
//     whole run is an unresolved literal (see below).
//   - ';'-terminated: an over-long UNKNOWN name. The reference algorithm does
//     NOT legacy-resolve it; the WHOLE run plus ';' is an unresolved literal.
//
// Because the two interpretations EMIT DIFFERENT bytes, nothing may be emitted
// before the ';'-vs-not decision is final — an optimistic legacy emit followed by
// the discovery of a trailing ';' would leave a partial Characters callback ahead
// of an error. The decision can only be settled at the run's END.
//
// BOUNDED SPOOL — peak memory AND bytes-read are both capped by MaxContentSize.
// The tail is CONSUMED in chunks no larger than sizeCap into a spool while the
// accumulated would-be literal length is tracked. As soon as that length exceeds
// sizeCap BEFORE the run's end is reached, the outcome is fixed (an over-cap
// unresolved literal if ';'-terminated, an over-cap ambiguous legacy run
// otherwise) and the parse HARD-FAILS with ErrContentSizeExceeded, emitting
// NOTHING — it never streams or buffers an unbounded tail. A no-';' legacy-prefix
// run therefore resolves only when its WHOLE run fits the cap; an over-cap one
// hard-fails. Once the decision is reached within cap, and only then, the chosen
// interpretation is emitted, so no partial emission ever precedes an error.
//
// ctx is checked between bounded chunks so a cancelled parse aborts promptly
// without consuming the whole run.
func (p *parser) parseSaturatedCharRefLiteral(ctx context.Context, head string, limit int) {
	sizeCap := effectiveContentCap(limit)

	// Does head begin with a resolvable legacy prefix? This is the ONLY way a
	// saturated run can resolve, and it depends solely on head (a legacy entity
	// is ≤6 chars). Compute it once: it selects between the resolve path and the
	// literal path, gated on the absence of a ';' (settled below).
	val, remainder, legacyResolves := resolveNamedEntity(head, false)

	// Consume the rest of the alphanumeric run into a cap-bounded spool, tracking
	// the would-be literal length ("&" + head + tail). chunkSize keeps each fill
	// bounded; the loop ends when a chunk shorter than chunkSize is returned (the
	// run is exhausted) — at which point the next byte settles the ';' question.
	// chunkSize is bounded by both sizeCap (so a single chunk never retains more
	// than the cap) and a small constant (so ctx cancellation and the over-cap
	// check are observed at fine granularity rather than once per huge chunk).
	chunkSize := min(saturatedCharRefChunk, sizeCap)
	literalLen := 1 + len(head) // '&' + head; grows as the tail is consumed
	var tail strings.Builder
	for {
		if ctx.Err() != nil {
			return
		}
		chunk, scanErr := p.parseWhileMaxErr(isAlphanumeric, chunkSize)
		// A short chunk is ambiguous: it can mean the alphanumeric run ended OR
		// that PeekAt/fillBuffer hit a read error / context cancellation (e.g. a
		// cancelled push-stream wait). Disambiguate BEFORE concluding "run
		// ended" or emitting: on a read error / cancel, unwind without emitting
		// any Characters/partial resolution and let the main loop surface the
		// error (it re-checks ctx.Err() and p.cur.Err()).
		if scanErr != nil || ctx.Err() != nil {
			return
		}
		literalLen += len(chunk)
		// Fail BEFORE retaining an over-cap spool: once the literal exceeds the
		// cap the outcome is ErrContentSizeExceeded regardless of any trailing
		// ';' (the run is either an over-cap literal or an over-cap ambiguous
		// legacy run, both hard failures), so stop consuming and emit nothing.
		if literalLen > sizeCap {
			p.fatalErr = fmt.Errorf("unresolved character reference exceeds %d bytes before terminator: %w", sizeCap, ErrContentSizeExceeded)
			return
		}
		tail.WriteString(chunk)
		if len(chunk) < chunkSize {
			break // run exhausted; cursor now sits just past the last alphanumeric
		}
	}

	// Run end reached within cap. Settle the ';' (it can only follow the run's
	// end, where the cursor now sits). Peek may refill the buffer when the run
	// ended exactly at a buffer boundary; if that refill hit a read error /
	// cancellation, abort without emitting so a cancelled push parse never
	// resolves a partial run.
	hasSemicolon := p.cur.Peek() == ';'
	if ctx.Err() != nil || p.cur.Err() != nil {
		return
	}
	if hasSemicolon {
		literalLen++
		if literalLen > sizeCap {
			p.fatalErr = fmt.Errorf("unresolved character reference exceeds %d bytes before terminator: %w", sizeCap, ErrContentSizeExceeded)
			return
		}
		_ = p.cur.Advance(1)
	}

	// No ';' and head legacy-resolves: mirror the reference algorithm's
	// no-semicolon
	// legacy-prefix path. The run fit the cap, so emit the resolution + head's
	// leftover + the spooled tail as ordinary text. Only now, with ';' ruled out,
	// does emission begin, so nothing was emitted prematurely.
	if !hasSemicolon && legacyResolves {
		_ = p.emitCharacters([]byte(val))
		if remainder != "" {
			p.emitLiteralChunked(remainder, limit)
		}
		p.emitLiteralChunked(tail.String(), limit)
		return
	}

	// ';'-terminated (over-long unknown name) or a no-';' run with no legacy
	// prefix: the WHOLE run is an unresolved literal. It fit the cap, so echo it
	// verbatim including any ';'.
	text := "&" + head + tail.String()
	if hasSemicolon {
		text += ";"
	}
	p.emitLiteralChunked(text, limit)
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
		chunk, scanErr := p.parseWhileMaxErr(pred, chunkSize)
		// An empty chunk is ambiguous: the digit run ended (next byte is not a
		// digit, e.g. ';' or EOF) OR PeekAt/fillBuffer hit a read error /
		// cancellation. Disambiguate so a cancelled read is NOT mistaken for a
		// finished run that would then emit a (partial) numeric entity: report
		// cancelled=true so the caller emits nothing and the main loop surfaces
		// the error.
		if scanErr != nil || ctx.Err() != nil {
			return 0, false, true
		}
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

// effectiveContentCap resolves a content-size limit to the value actually used
// for capping: the caller's limit when positive, otherwise defaultMaxContentSize.
func effectiveContentCap(limit int) int {
	if limit <= 0 {
		return defaultMaxContentSize
	}
	return limit
}

// emitLiteralChunked emits s as literal text in Characters events no larger
// than limit bytes. s contains only ASCII (the '&', '#', 'x', and alphanumeric
// or digit bytes of a character reference), so splitting on byte boundaries
// never breaks a multi-byte rune.
func (p *parser) emitLiteralChunked(s string, limit int) {
	limit = effectiveContentCap(limit)
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

// emitCharacters fires the appropriate SAX Characters event for normal
// data-state character data, and is the single chokepoint for the two text-run
// decisions that can only be made once the run's first non-whitespace byte is
// known: whitespace-significance (StripBlanks/noBlanks) and implied-<body>
// insertion (see the pendingWS field doc).
//
//   - Non-whitespace data marks the current run significant (curTextRunSignificant)
//     and FLUSHES any deferred leading whitespace into it. ANY non-whitespace emit
//     path — a plain text chunk, a resolved entity / numeric char-ref, an
//     unresolved char-ref literal, a U+FFFD replacement, or a lone literal '<' —
//     flows through here, so the run is marked uniformly.
//   - All-whitespace data whose run is not yet significant is, when its insertion
//     target is still undecided, DEFERRED into pendingWS rather than emitted: in
//     <head> (target already correct) it is emitted there or dropped under
//     noBlanks; before the root element it is ignorable and dropped; inside
//     raw-text/RCDATA elements it is always kept.
//
// curTextRunSignificant is RESET only when a real markup tag is dispatched in the
// main parse loop; a char-ref and a lone '<' are part of the same run and never
// reset it.
func (p *parser) emitCharacters(data []byte) error {
	if !isAllWhitespace(data) {
		p.curTextRunSignificant = true
		// Flush deferred leading whitespace into the now-significant run BEFORE
		// emitting this non-whitespace data. The caller has already established the
		// insertion target (htmlStartCharData / lone-'<'), so it lands correctly.
		// Route any SAX callback error through handleSAXErr exactly like the regular
		// character-emit path (e.g. parseRawContent): an UNSET Characters handler
		// returns ErrHandlerUnspecified and is filtered to a no-op, a real handler
		// error is captured (strict) or warned (tolerant). Either way we must NOT
		// short-circuit — the current non-whitespace chunk below (including the
		// encoding-error check) still has to be emitted and processed.
		p.handleSAXErr(p.flushPendingWS())
	} else if !p.curTextRunSignificant && !p.inRawTextElement() {
		// All-whitespace data whose run significance / insertion target may not yet
		// be established.
		switch {
		case len(p.nameStack) == 0 && !p.sawRoot:
			// No element is open yet AND the root <html> has never been opened, so this
			// whitespace precedes the document content and is ignorable; drop it. The
			// sawRoot guard distinguishes genuine pre-root whitespace from TRAILING
			// whitespace AFTER the root has closed (stack also empty, but sawRoot is
			// true) which must still be emitted: opening a bare <html> does not advance
			// the insertion mode past insertInitial, so a mode-based guard would
			// mis-drop the trailing run of `<html></html> \n`. Under SuppressImplied no
			// <html> root is ever created, so sawRoot stays false and pre-root
			// whitespace remains ignorable; once a non-html element is open the stack is
			// non-empty and this path is not reached.
			return nil
		case p.currentName() == elemHead:
			// Inside <head> the insertion target is already correct. StripBlanks still
			// strips a whitespace-only run; otherwise fall out of the switch and emit
			// it under <head>.
			if p.cfg.noBlanks {
				return nil
			}
		case p.cfg.noBlanks || (!p.cfg.noImplied && p.mode < insertInBody):
			// Significance or insertion is genuinely UNDECIDED, so defer. Under
			// StripBlanks a whitespace-only run is stripped, but a following
			// non-whitespace byte makes this leading whitespace significant. And while
			// implied insertion is ENABLED and the body subtree has not yet been
			// entered (mode < insertInBody) the next non-whitespace byte would open the
			// implied <body> via htmlStartCharData, so emitting now would split one
			// logical run across parents. When implied insertion is DISABLED
			// (SuppressImplied) and an element is already open, the insertion target is
			// fixed, so there is nothing to defer — fall through and emit immediately.
			// deferPendingWS hard-fails an over-cap undecidable whitespace prefix
			// rather than buffering unbounded.
			return p.deferPendingWS(data)
		}
		// Reaching here means inside <head> without StripBlanks, or an element is open
		// with a fixed insertion target (StripBlanks OFF and either the body subtree
		// has been entered or implied insertion is disabled). The whitespace is
		// significant and lands in the current element regardless of what follows, so
		// emit it immediately under the SOFT cap — no deferral, no undecidable-
		// whitespace hard-fail. This is the normal data-state text path, e.g.
		// <p> + over-cap spaces + </p>, including under SuppressImplied.
		//
		// Flush any deferred leading whitespace FIRST so output order is preserved.
		// The insertion target is now fixed (e.g. a whitespace-producing char-ref such
		// as &#9; / &Tab; just opened the implied <body> via htmlStartCharData), and
		// the pendingWS prefix lexically precedes this chunk; emitting this whitespace
		// ahead of the still-deferred prefix would reorder the run (e.g. `<html> &#9;a`
		// must yield " \ta", not "\t a"). flushPendingWS is a no-op when nothing was
		// deferred.
		p.handleSAXErr(p.flushPendingWS())
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
		// The deferred undeclared-charset reader never produces a sanitized U+FFFD
		// of its own: it stays pure UTF-8, switches the whole buffer to Latin-1, or
		// fails closed at the buffering cap. So a U+FFFD reaching here on that path
		// is genuine document content, not an encoding error — no diagnostic.
	}
	return p.sax.Characters(data)
}

// deferPendingWS appends still-undecided leading whitespace to the bounded
// pendingWS buffer. The buffer is bounded by the content cap: if the whitespace
// prefix alone reaches the cap before any non-whitespace byte establishes the
// run's significance, the parse hard-fails with ErrContentSizeExceeded rather
// than buffering unbounded — the same over-cap policy indivisible constructs use
// elsewhere.
func (p *parser) deferPendingWS(data []byte) error {
	limit := p.cfg.contentLimit()
	if len(p.pendingWS)+len(data) > limit {
		p.fatalErr = fmt.Errorf("character data exceeds %d bytes before its whitespace significance can be determined: %w", limit, ErrContentSizeExceeded)
		return p.fatalErr
	}
	p.pendingWS = append(p.pendingWS, data...)
	return nil
}

// flushPendingWS emits deferred leading whitespace as part of a run now known
// significant. The caller (about to emit non-whitespace) has already established
// the insertion target, so the whitespace lands in the correct element. It is
// emitted in cap-sized chunks (whitespace is ASCII, so splitting on byte
// boundaries never breaks a rune).
func (p *parser) flushPendingWS() error {
	if len(p.pendingWS) == 0 {
		return nil
	}
	ws := p.pendingWS
	p.pendingWS = nil
	return p.emitWSChunked(ws)
}

// flushPendingWSRunEnd resolves deferred leading whitespace when the run ends
// without ever becoming significant — a real markup tag or EOF closes it. The
// run is therefore entirely whitespace: StripBlanks (noBlanks) drops it,
// pre-root whitespace (empty stack, root never opened) drops it, and otherwise
// it is emitted under the current element. It is a no-op when nothing was
// deferred.
func (p *parser) flushPendingWSRunEnd() error {
	if len(p.pendingWS) == 0 {
		return nil
	}
	ws := p.pendingWS
	p.pendingWS = nil
	if p.cfg.noBlanks || (len(p.nameStack) == 0 && !p.sawRoot) {
		return nil
	}
	return p.emitWSChunked(ws)
}

// emitWSChunked writes whitespace directly to SAX in cap-sized chunks, bypassing
// the emitCharacters significance/deferral logic (the caller has already decided
// the whitespace is to be emitted).
func (p *parser) emitWSChunked(ws []byte) error {
	limit := p.cfg.contentLimit()
	for len(ws) > limit {
		if err := p.sax.Characters(ws[:limit]); err != nil {
			return err
		}
		ws = ws[limit:]
	}
	return p.sax.Characters(ws)
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

// isValidEndTagTerminator reports whether the byte following a matched end-tag
// prefix terminates the element. afterTag is the result of PeekAt(len(endTag)).
// A NUL sentinel here is ambiguous (it stands for both a real U+0000 byte and
// true EOF), so callers MUST handle true-EOF termination via HasByteAt BEFORE
// calling this: a real NUL is never a valid terminator, only the explicit
// '>'/space/'\t'/'\n'/'\r' characters are.
func isValidEndTagTerminator(afterTag byte) bool {
	switch afterTag {
	case '>', ' ', '\t', '\n', '\r':
		return true
	}
	return false
}

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

	// Check ctx/fatalErr BEFORE Done() so a condition set during the previous
	// iteration aborts immediately. Done() refills via a blocking Read; checking
	// afterward would let an over-cap fatalErr (or a cancellation) trigger one
	// more blocking read before this loop returns.
	for {
		// Abort promptly on context cancellation rather than buffering the
		// entire (possibly gigantic or unterminated) section first. The main
		// parse loop re-checks ctx.Err() and surfaces it.
		if ctx.Err() != nil {
			flushChunk()
			return
		}
		// An over-cap construct (e.g. via emitted SAX content in strict mode)
		// may set fatalErr; stop scanning immediately so the main loop surfaces
		// it instead of issuing another blocking read.
		if p.fatalErr != nil {
			flushChunk()
			return
		}
		if p.cur.Done() {
			break
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
				// True EOF after the matched tag terminates the element. A real
				// NUL byte (HasByteAt true, PeekAt 0) does not.
				validEnd := !p.cur.HasByteAt(afterTag) || isValidEndTagTerminator(p.cur.PeekAt(afterTag))
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
	// Determine the expected UTF-8 sequence width from the lead byte alone and
	// peek only that many bytes. Requesting utf8.UTFMax unconditionally would,
	// under PUSH parsing, block until four bytes are buffered (or EOF) even
	// though a complete 2- or 3-byte rune is already present — stalling
	// progressive emission of raw-text/RCDATA/plaintext content. The lead byte
	// tells us exactly how many bytes a valid sequence needs:
	//   0xC0-0xDF -> 2, 0xE0-0xEF -> 3, 0xF0-0xF7 -> 4; anything else is an
	// invalid lead (a stray continuation byte or 0xF8+) handled as one byte.
	width := utf8ExpectedWidth(b)
	if width == 1 {
		// Invalid lead byte: consume a single byte so the scan makes progress
		// and the caller never emits a partial rune.
		return string([]byte{b}), 1
	}
	// Peek exactly the bytes a valid sequence needs. DecodeRuneInString reports
	// a size of 1 for an invalid sequence and the true size for a valid rune
	// (including a genuine U+FFFD), so the size distinguishes the two cases.
	s := p.cur.PeekString(width)
	if s == "" {
		// Fewer than width bytes remain. Near true EOF, peek whatever is left
		// and decode/replace it; this is not a block-on-more-input case because
		// a complete rune of this width cannot fit in the remaining bytes.
		for n := width - 1; n >= 1; n-- {
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

// utf8ExpectedWidth returns the number of bytes a valid UTF-8 sequence starting
// with lead byte b occupies: 1 for ASCII, 2/3/4 for multi-byte leads. Any byte
// that cannot begin a valid sequence (a continuation byte 0x80-0xBF or 0xF8+)
// returns 1 so callers treat it as a single invalid byte.
func utf8ExpectedWidth(b byte) int {
	switch {
	case b < 0x80:
		return 1
	case b >= 0xF0 && b <= 0xF7:
		return 4
	case b >= 0xE0 && b <= 0xEF:
		return 3
	case b >= 0xC0 && b <= 0xDF:
		return 2
	default:
		return 1
	}
}

// isUTF8Continuation reports whether b is a UTF-8 continuation byte
// (0b10xxxxxx). A rune boundary is any byte that is not a continuation byte, so
// backing a byte index off continuation bytes lands on a whole-rune boundary.
func isUTF8Continuation(b byte) bool {
	return b&0xC0 == 0x80
}

// clampTextChunkToRune adjusts a text-run length scanned up to the content cap
// so a chunk flushed at the cap never splits a multi-byte UTF-8 sequence. When
// the run reached the cap (n >= limit) it backs n off to the last whole-rune
// boundary; if the run begins with a single rune larger than the cap it extends
// n forward to cover that rune whole rather than emitting a partial rune. A run
// shorter than the cap (stopped at a real delimiter) is returned unchanged.
func (p *parser) clampTextChunkToRune(n, limit int) int {
	if n < limit {
		return n
	}
	for n > 0 && isUTF8Continuation(p.cur.PeekAt(n)) {
		n--
	}
	if n == 0 {
		// A lone rune exceeds the cap. Extend to cover it whole so a partial
		// rune is never emitted.
		n = limit
		for p.cur.HasByteAt(n) && isUTF8Continuation(p.cur.PeekAt(n)) {
			n++
		}
	}
	return n
}

// parseRCDATAContent parses RCDATA content (title, textarea).
// Like raw text but entities are expanded.
func (p *parser) parseRCDATAContent(ctx context.Context, tagName string) {
	endTag := "</" + tagName
	limit := p.cfg.contentLimit()

	// Check ctx/fatalErr BEFORE Done() so a condition set during the previous
	// iteration aborts immediately. Done() refills via a blocking Read; checking
	// afterward would let an over-cap fatalErr (set by parseCharRefBounded at a
	// buffer boundary) or a cancellation trigger one more blocking read.
	for {
		// Abort promptly on context cancellation. The main parse loop
		// re-checks ctx.Err() and surfaces it.
		if ctx.Err() != nil {
			return
		}
		// A char-ref over the content cap sets fatalErr; stop scanning so the
		// main loop surfaces it instead of issuing another blocking read.
		if p.fatalErr != nil {
			return
		}
		if p.cur.Done() {
			break
		}
		if p.cur.Peek() == '<' && p.cur.PeekAt(1) == '/' {
			if p.hasPrefixFold(endTag) {
				afterTag := len(endTag)
				// True EOF after the matched tag terminates the element; a real
				// NUL byte (HasByteAt true, PeekAt 0) does not.
				if !p.cur.HasByteAt(afterTag) || isValidEndTagTerminator(p.cur.PeekAt(afterTag)) {
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
			// A char-ref over the content cap sets fatalErr; return from the
			// current iteration immediately rather than scanning further. The
			// loop-top fatalErr guard already prevents another blocking read.
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
			// boundary so the emitted chunk is valid UTF-8 (extending a lone
			// over-cap rune to cover it whole).
			n = p.clampTextChunkToRune(n, limit)
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
					// True EOF after the matched tag; a real NUL does not count
					// as a valid end-tag terminator.
					validEnd = !p.cur.HasByteAt(len(endTag)) || isValidEndTagTerminator(p.cur.PeekAt(len(endTag)))
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
	// Check ctx/fatalErr BEFORE Done() so a condition set during the previous
	// iteration aborts immediately. Done() refills via a blocking Read; checking
	// afterward would let a fatalErr or cancellation trigger one more blocking
	// read before this loop returns.
	for {
		// Abort promptly on context cancellation rather than buffering the
		// entire (possibly endless) plaintext section first.
		if ctx.Err() != nil {
			flushChunk()
			return
		}
		// An over-cap construct may set fatalErr; stop scanning immediately so
		// the main loop surfaces it instead of issuing another blocking read.
		if p.fatalErr != nil {
			flushChunk()
			return
		}
		if p.cur.Done() {
			break
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
	limit := p.cfg.scanTokenLimit()
	n := 0
	for {
		b := p.cur.PeekAt(n)
		if b == 0 || !isNameChar(b) {
			break
		}
		// A tag name is part of an indivisible start/end-tag event and cannot be
		// chunked, so bound the unbounded PeekAt scan as a HARD cap (see
		// scanTokenLimit): a gigantic (e.g. unterminated) name must not grow the
		// cursor buffer without limit. Exactly limit bytes are accepted; the
		// limit+1-th fails.
		if n >= limit {
			p.fatalErr = fmt.Errorf("tag name exceeds %d bytes: %w", limit, ErrContentSizeExceeded)
			return ""
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

	// An over-cap attribute value sets fatalErr; stop scanning immediately so the
	// main loop surfaces it rather than buffering further attributes.
	for p.fatalErr == nil {
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
	limit := p.cfg.scanTokenLimit()
	n := 0
	for {
		b := p.cur.PeekAt(n)
		if b == 0 || isWhitespaceByte(b) || b == '>' || b == '/' || b == '=' || b == '"' || b == '\'' || b == '<' {
			break
		}
		// An attribute name is part of an indivisible start-tag event and cannot
		// be chunked, so bound the unbounded PeekAt scan as a HARD cap (see
		// scanTokenLimit). Exactly limit bytes are accepted; the limit+1-th fails.
		if n >= limit {
			p.fatalErr = fmt.Errorf("attribute name exceeds %d bytes: %w", limit, ErrContentSizeExceeded)
			return ""
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
	limit := p.cfg.contentLimit()
	var buf bytes.Buffer

	for !p.cur.Done() && p.cur.Peek() != quote {
		// An attribute value is part of an indivisible start-tag event and cannot
		// be chunked, so enforce the content limit as a HARD cap: an unbounded
		// (e.g. unterminated) value must not buffer without limit. Fail before
		// accepting a byte that would push the accumulated value past the cap.
		if buf.Len() >= limit {
			p.fatalErr = fmt.Errorf("attribute value exceeds %d bytes before terminator: %w", limit, ErrContentSizeExceeded)
			return ""
		}
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
	limit := p.cfg.contentLimit()
	var buf bytes.Buffer

	for !p.cur.Done() {
		b := p.cur.Peek()
		if b == '>' || b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\f' {
			break
		}
		// Hard-cap the accumulated value the same way as the quoted form so an
		// unbounded unquoted value cannot buffer without limit.
		if buf.Len() >= limit {
			p.fatalErr = fmt.Errorf("attribute value exceeds %d bytes before terminator: %w", limit, ErrContentSizeExceeded)
			return ""
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
		codepoint, haveDigits := p.scanNumericCharRef()
		// Per HTML5, &#0; / &#x0; in an attribute value maps to U+FFFD rather
		// than being dropped. Emit nothing only for a bare "&#" with no digits.
		if haveDigits {
			return string(normalizeNumericCharRef(codepoint))
		}
		return ""
	}

	// Bound the entity-name scan to maxEntityNameLen so a runaway '&'-led
	// alphanumeric run cannot slurp unbounded into one allocation, bypassing the
	// caller's per-byte hard cap. No real entity name reaches this length, so any
	// over-cap tail falls through to the caller's capped loop and output for valid
	// entities is unchanged.
	name, _ := p.parseWhileMaxErr(isAlphanumeric, maxEntityNameLen)
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
	limit := p.cfg.scanTokenLimit()
	n := 0
	// HasByteAt in the loop condition distinguishes EOF from a real NUL byte:
	// PeekAt returns 0 for both, so a NUL inside the literal must NOT be mistaken
	// for end-of-input — doing so would exit the scan early WITHOUT setting
	// fatalErr and bypass the hard cap (letting parseDoctype drain an unterminated
	// abusive literal and emit a partial subset). A NUL counts as content, like
	// the comment/PI scanners.
	for p.cur.HasByteAt(n) {
		if p.cur.PeekAt(n) == quote {
			break
		}
		// A DOCTYPE PUBLIC/SYSTEM literal maps to a single InternalSubset SAX
		// argument and cannot be chunked, so bound the unbounded PeekAt scan as a
		// HARD cap (see scanTokenLimit): an unterminated huge literal must not grow
		// the cursor buffer without limit. Exactly limit bytes are accepted; the
		// limit+1-th fails. parseDoctype checks fatalErr and stops.
		if n >= limit {
			p.fatalErr = fmt.Errorf("doctype literal exceeds %d bytes before terminator: %w", limit, ErrContentSizeExceeded)
			return ""
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

// parseWhileMaxErr collects characters while pred returns true, with an upper bound: it consumes at most limit
// matching bytes, leaving any further matching bytes for the next call. It lets
// a caller bound an otherwise unbounded scan (e.g. a runaway entity name) and
// drain it in fixed-size pieces. A limit <= 0 is treated as unbounded.
//
// It ALSO reports whether the scan was cut short by a read error / context
// cancellation rather than by genuine exhaustion. PeekAt returns 0 both at true
// EOF and when fillBuffer hit a non-EOF read error (e.g. the push stream
// returning context.Canceled when its blocking wait is cancelled). Both stop the
// scan and produce a chunk shorter than limit, so length alone cannot tell "run
// ended" from "read failed". This is the disambiguation the bounded char-ref
// scanners need: a short chunk is EITHER true exhaustion (next byte does not
// match pred, or clean EOF) OR a read error (p.cur.Err() != nil). When the scan
// stops short, this consults p.cur.Err() so callers never conclude "run ended" —
// and never emit — on a cancelled/failed read.
//
// err is non-nil ONLY when the scan stopped short because the cursor could not
// supply the next byte AND that was due to a recorded read error. A scan that
// stops on a genuine non-matching byte, on clean EOF, or that fills the whole
// limit returns a nil err.
func (p *parser) parseWhileMaxErr(pred func(byte) bool, limit int) (string, error) {
	n := 0
	stoppedShort := false
	for limit <= 0 || n < limit {
		if !p.cur.HasByteAt(n) {
			// No byte available at n: either clean EOF or a read error. Mark it
			// so we can disambiguate against p.cur.Err() below.
			stoppedShort = true
			break
		}
		b := p.cur.PeekAt(n)
		if b == 0 || !pred(b) {
			break
		}
		n++
	}
	var err error
	if stoppedShort {
		err = p.cur.Err()
	}
	if n == 0 {
		return "", err
	}
	s := p.cur.PeekString(n)
	_ = p.cur.Advance(n)
	return s, err
}

// skipWhitespace skips whitespace characters.
func (p *parser) skipWhitespace() {
	limit := p.cfg.scanTokenLimit()
	n := 0
	for {
		b := p.cur.PeekAt(n)
		if b != ' ' && b != '\t' && b != '\n' && b != '\r' && b != '\f' {
			break
		}
		// skipWhitespace runs only in tag context (between attributes / around the
		// tag close), so a multi-megabyte whitespace run is pathological. Bound the
		// unbounded PeekAt scan as a HARD cap (see scanTokenLimit) so it cannot grow
		// the cursor buffer without limit. Exactly limit bytes are accepted; the
		// limit+1-th fails.
		if n >= limit {
			p.fatalErr = fmt.Errorf("whitespace run exceeds %d bytes: %w", limit, ErrContentSizeExceeded)
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
	return isASCIIAlpha(b) || xmlchar.IsASCIIDigit(b) || b == ':' || b == '-' || b == '_' || b == '.'
}

func isAlphanumeric(b byte) bool {
	return isASCIIAlpha(b) || xmlchar.IsASCIIDigit(b)
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
