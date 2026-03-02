package html_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/lestrrat-go/helium/html"
	"github.com/stretchr/testify/require"
)

// htmlReverseEntityMap maps Unicode codepoints to HTML4 entity names.
// Built to match libxml2's htmlEntityValueLookup used in htmlEncodeEntities.
var htmlReverseEntityMap map[rune]string

func init() {
	// Build reverse lookup: codepoint → shortest entity name.
	// libxml2's htmlEntityValueLookup returns the first match from its table;
	// we pick the shortest name for each codepoint to match libxml2 output.
	forward := map[string]string{
		"AElig": "\u00C6", "Aacute": "\u00C1", "Acirc": "\u00C2", "Agrave": "\u00C0",
		"Alpha": "\u0391", "Aring": "\u00C5", "Atilde": "\u00C3", "Auml": "\u00C4",
		"Beta": "\u0392", "Ccedil": "\u00C7", "Chi": "\u03A7", "Dagger": "\u2021",
		"Delta": "\u0394", "ETH": "\u00D0", "Eacute": "\u00C9", "Ecirc": "\u00CA",
		"Egrave": "\u00C8", "Epsilon": "\u0395", "Eta": "\u0397", "Euml": "\u00CB",
		"Gamma": "\u0393", "Iacute": "\u00CD", "Icirc": "\u00CE", "Igrave": "\u00CC",
		"Iota": "\u0399", "Iuml": "\u00CF", "Kappa": "\u039A", "Lambda": "\u039B",
		"Mu": "\u039C", "Ntilde": "\u00D1", "Nu": "\u039D", "OElig": "\u0152",
		"Oacute": "\u00D3", "Ocirc": "\u00D4", "Ograve": "\u00D2", "Omega": "\u03A9",
		"Omicron": "\u039F", "Oslash": "\u00D8", "Otilde": "\u00D5", "Ouml": "\u00D6",
		"Phi": "\u03A6", "Pi": "\u03A0", "Prime": "\u2033", "Psi": "\u03A8",
		"Rho": "\u03A1", "Scaron": "\u0160", "Sigma": "\u03A3", "THORN": "\u00DE",
		"Tau": "\u03A4", "Theta": "\u0398", "Uacute": "\u00DA", "Ucirc": "\u00DB",
		"Ugrave": "\u00D9", "Upsilon": "\u03A5", "Uuml": "\u00DC", "Xi": "\u039E",
		"Yacute": "\u00DD", "Yuml": "\u0178", "Zeta": "\u0396",
		"aacute": "\u00E1", "acirc": "\u00E2", "acute": "\u00B4", "aelig": "\u00E6",
		"agrave": "\u00E0", "alefsym": "\u2135", "alpha": "\u03B1", "amp": "&", "apos": "'",
		"and": "\u2227", "ang": "\u2220", "aring": "\u00E5", "asymp": "\u2248",
		"atilde": "\u00E3", "auml": "\u00E4", "bdquo": "\u201E", "beta": "\u03B2",
		"brvbar": "\u00A6", "bull": "\u2022", "cap": "\u2229", "ccedil": "\u00E7",
		"cedil": "\u00B8", "cent": "\u00A2", "chi": "\u03C7", "circ": "\u02C6",
		"clubs": "\u2663", "cong": "\u2245", "copy": "\u00A9", "crarr": "\u21B5",
		"cup": "\u222A", "curren": "\u00A4", "dArr": "\u21D3", "dagger": "\u2020",
		"darr": "\u2193", "deg": "\u00B0", "delta": "\u03B4", "diams": "\u2666",
		"divide": "\u00F7", "eacute": "\u00E9", "ecirc": "\u00EA", "egrave": "\u00E8",
		"empty": "\u2205", "emsp": "\u2003", "ensp": "\u2002", "epsilon": "\u03B5",
		"equiv": "\u2261", "eta": "\u03B7", "eth": "\u00F0", "euml": "\u00EB",
		"euro": "\u20AC", "exist": "\u2203", "fnof": "\u0192", "forall": "\u2200",
		"frac12": "\u00BD", "frac14": "\u00BC", "frac34": "\u00BE", "frasl": "\u2044",
		"gamma": "\u03B3", "ge": "\u2265", "gt": ">", "hArr": "\u21D4",
		"harr": "\u2194", "hearts": "\u2665", "hellip": "\u2026", "iacute": "\u00ED",
		"icirc": "\u00EE", "iexcl": "\u00A1", "igrave": "\u00EC", "image": "\u2111",
		"infin": "\u221E", "int": "\u222B", "iota": "\u03B9", "iquest": "\u00BF",
		"isin": "\u2208", "iuml": "\u00EF", "kappa": "\u03BA", "lArr": "\u21D0",
		"lambda": "\u03BB", "lang": "\u2329", "laquo": "\u00AB", "larr": "\u2190",
		"lceil": "\u2308", "ldquo": "\u201C", "le": "\u2264", "lfloor": "\u230A",
		"lowast": "\u2217", "loz": "\u25CA", "lrm": "\u200E", "lsaquo": "\u2039",
		"lsquo": "\u2018", "lt": "<", "macr": "\u00AF", "mdash": "\u2014",
		"micro": "\u00B5", "middot": "\u00B7", "minus": "\u2212", "mu": "\u03BC",
		"nabla": "\u2207", "nbsp": "\u00A0", "ndash": "\u2013", "ne": "\u2260",
		"ni": "\u220B", "not": "\u00AC", "notin": "\u2209", "nsub": "\u2284",
		"ntilde": "\u00F1", "nu": "\u03BD", "oacute": "\u00F3", "ocirc": "\u00F4",
		"oelig": "\u0153", "ograve": "\u00F2", "oline": "\u203E", "omega": "\u03C9",
		"omicron": "\u03BF", "oplus": "\u2295", "or": "\u2228", "ordf": "\u00AA",
		"ordm": "\u00BA", "oslash": "\u00F8", "otilde": "\u00F5", "otimes": "\u2297",
		"ouml": "\u00F6", "para": "\u00B6", "part": "\u2202", "permil": "\u2030",
		"perp": "\u22A5", "phi": "\u03C6", "pi": "\u03C0", "piv": "\u03D6",
		"plusmn": "\u00B1", "pound": "\u00A3", "prime": "\u2032", "prod": "\u220F",
		"prop": "\u221D", "psi": "\u03C8", "quot": "\"", "rArr": "\u21D2",
		"radic": "\u221A", "rang": "\u232A", "raquo": "\u00BB", "rarr": "\u2192",
		"rceil": "\u2309", "rdquo": "\u201D", "real": "\u211C", "reg": "\u00AE",
		"rfloor": "\u230B", "rho": "\u03C1", "rlm": "\u200F", "rsaquo": "\u203A",
		"rsquo": "\u2019", "sbquo": "\u201A", "scaron": "\u0161", "sdot": "\u22C5",
		"sect": "\u00A7", "shy": "\u00AD", "sigma": "\u03C3", "sigmaf": "\u03C2",
		"sim": "\u223C", "spades": "\u2660", "sub": "\u2282", "sube": "\u2286",
		"sum": "\u2211", "sup": "\u2283", "sup1": "\u00B9", "sup2": "\u00B2",
		"sup3": "\u00B3", "supe": "\u2287", "szlig": "\u00DF", "tau": "\u03C4",
		"there4": "\u2234", "theta": "\u03B8", "thetasym": "\u03D1", "thinsp": "\u2009",
		"thorn": "\u00FE", "tilde": "\u02DC", "times": "\u00D7", "trade": "\u2122",
		"uArr": "\u21D1", "uacute": "\u00FA", "uarr": "\u2191", "ucirc": "\u00FB",
		"ugrave": "\u00F9", "uml": "\u00A8", "upsih": "\u03D2", "upsilon": "\u03C5",
		"uuml": "\u00FC", "weierp": "\u2118", "xi": "\u03BE", "yacute": "\u00FD",
		"yen": "\u00A5", "yuml": "\u00FF", "zeta": "\u03B6", "zwj": "\u200D",
		"zwnj": "\u200C",
	}
	htmlReverseEntityMap = make(map[rune]string, len(forward))
	for name, val := range forward {
		r, _ := utf8.DecodeRuneInString(val)
		if r == utf8.RuneError {
			continue
		}
		if existing, ok := htmlReverseEntityMap[r]; ok {
			if len(name) < len(existing) {
				htmlReverseEntityMap[r] = name
			}
		} else {
			htmlReverseEntityMap[r] = name
		}
	}
}

// htmlEncodeEntities converts a byte slice to display form matching libxml2's
// htmlEncodeEntities: ASCII printable chars pass through, &/</>  are entity-encoded,
// non-ASCII chars are looked up by codepoint and displayed as &name; or &#num;.
// quoteChar is additionally encoded (0 for none, '\'' for attribute values).
// The output is truncated to outMax bytes (matching libxml2's outlen=30 buffer).
func htmlEncodeEntities(data []byte, outMax int, quoteChar rune) string {
	var out bytes.Buffer
	i := 0
	for i < len(data) {
		r, size := utf8.DecodeRune(data[i:])
		if r == utf8.RuneError && size <= 1 {
			// Invalid byte — encode as &#num;
			s := fmt.Sprintf("&#%d;", data[i])
			if out.Len()+len(s) > outMax {
				break
			}
			out.WriteString(s)
			i++
			continue
		}

		if r < 0x80 && r != '&' && r != '<' && r != '>' && (quoteChar == 0 || r != quoteChar) {
			// Plain ASCII — pass through
			if out.Len()+1 > outMax {
				break
			}
			out.WriteByte(byte(r))
			i += size
			continue
		}

		// Try to find an HTML entity name
		if name, ok := htmlReverseEntityMap[r]; ok {
			s := "&" + name + ";"
			if out.Len()+len(s) > outMax {
				break
			}
			out.WriteString(s)
		} else {
			s := fmt.Sprintf("&#%d;", r)
			if out.Len()+len(s) > outMax {
				break
			}
			out.WriteString(s)
		}
		i += size
	}
	return out.String()
}

// htmlEncodeEntitiesAttr encodes attribute value for display (no length limit,
// quote char '\'' is also encoded).
func htmlEncodeEntitiesAttr(data string) string {
	return htmlEncodeEntities([]byte(data), len(data)*10, '\'')
}

func newHTMLSAXEventEmitter(out *bytes.Buffer) html.SAXHandler {
	s := &html.SAXCallbacks{}

	s.SetDocumentLocatorHandler = html.SetDocumentLocatorFunc(func(_ html.DocumentLocator) error {
		fmt.Fprintf(out, "SAX.setDocumentLocator()\n")
		return nil
	})
	s.StartDocumentHandler = html.StartDocumentFunc(func() error {
		fmt.Fprintf(out, "SAX.startDocument()\n")
		return nil
	})
	s.EndDocumentHandler = html.EndDocumentFunc(func() error {
		fmt.Fprintf(out, "SAX.endDocument()\n")
		return nil
	})
	s.InternalSubsetHandler = html.InternalSubsetFunc(func(name, externalID, systemID string) error {
		fmt.Fprintf(out, "SAX.internalSubset(%s, %s, %s)\n", name, externalID, systemID)
		return nil
	})
	s.StartElementHandler = html.StartElementFunc(func(name string, attrs []html.Attribute) error {
		if len(attrs) == 0 {
			fmt.Fprintf(out, "SAX.startElement(%s)\n", name)
		} else {
			parts := make([]string, 0, len(attrs))
			for _, a := range attrs {
				if a.Boolean {
					parts = append(parts, a.Name)
				} else {
					val := htmlEncodeEntitiesAttr(a.Value)
					parts = append(parts, fmt.Sprintf("%s='%s'", a.Name, val))
				}
			}
			fmt.Fprintf(out, "SAX.startElement(%s, %s)\n", name, strings.Join(parts, ", "))
		}
		return nil
	})
	s.EndElementHandler = html.EndElementFunc(func(name string) error {
		fmt.Fprintf(out, "SAX.endElement(%s)\n", name)
		return nil
	})
	s.CharactersHandler = html.CharactersFunc(func(ch []byte) error {
		display := htmlEncodeEntities(ch, 30, 0)
		fmt.Fprintf(out, "SAX.characters(%s, %d)\n", display, len(ch))
		return nil
	})
	s.CDataBlockHandler = html.CDataBlockFunc(func(value []byte) error {
		display := htmlEncodeEntities(value, 30, 0)
		fmt.Fprintf(out, "SAX.cdata(%s, %d)\n", display, len(value))
		return nil
	})
	s.CommentHandler = html.CommentFunc(func(value []byte) error {
		fmt.Fprintf(out, "SAX.comment(%s)\n", string(value))
		return nil
	})
	s.ProcessingInstructionHandler = html.ProcessingInstructionFunc(func(target, data string) error {
		fmt.Fprintf(out, "SAX.processingInstruction(%s, %s)\n", target, data)
		return nil
	})
	s.IgnorableWhitespaceHandler = html.IgnorableWhitespaceFunc(func(ch []byte) error {
		display := htmlEncodeEntities(ch, 30, 0)
		fmt.Fprintf(out, "SAX.characters(%s, %d)\n", display, len(ch))
		return nil
	})
	s.WarningHandler = html.WarningFunc(func(msg string, args ...interface{}) error {
		fmt.Fprintf(out, "SAX.warning: %s\n", fmt.Sprintf(msg, args...))
		return nil
	})
	s.ErrorHandler = html.ErrorFunc(func(msg string, args ...interface{}) error {
		fmt.Fprintf(out, "SAX.error: %s\n", fmt.Sprintf(msg, args...))
		return nil
	})
	return s
}

// mergeHTMLCharEvents merges consecutive SAX.characters() and SAX.cdata() events.
var reHTMLCharEvent = regexp.MustCompile(`(?s)^SAX\.characters\((.*), (\d+)\)\n$`)
var reHTMLCdataEvent = regexp.MustCompile(`(?s)^SAX\.cdata\((.*), (\d+)\)\n$`)

func mergeHTMLCharEvents(s string) string {
	var events []string
	cur := ""
	for _, line := range strings.SplitAfter(s, "\n") {
		if strings.HasPrefix(line, "SAX.") && cur != "" {
			events = append(events, cur)
			cur = line
		} else {
			cur += line
		}
	}
	if cur != "" {
		events = append(events, cur)
	}

	var out []string
	mergedCharData := ""
	mergedCharLen := 0
	mergedCdataData := ""
	mergedCdataLen := 0

	flushChars := func() {
		if mergedCharLen > 0 {
			if len(mergedCharData) > 30 {
				mergedCharData = mergedCharData[:30]
			}
			out = append(out, fmt.Sprintf("SAX.characters(%s, %d)\n", mergedCharData, mergedCharLen))
			mergedCharData = ""
			mergedCharLen = 0
		}
	}

	flushCdata := func() {
		if mergedCdataLen > 0 {
			if len(mergedCdataData) > 30 {
				mergedCdataData = mergedCdataData[:30]
			}
			out = append(out, fmt.Sprintf("SAX.cdata(%s, %d)\n", mergedCdataData, mergedCdataLen))
			mergedCdataData = ""
			mergedCdataLen = 0
		}
	}

	for _, ev := range events {
		if m := reHTMLCharEvent.FindStringSubmatch(ev); m != nil {
			flushCdata()
			mergedCharData += m[1]
			n, _ := strconv.Atoi(m[2])
			mergedCharLen += n
		} else if m := reHTMLCdataEvent.FindStringSubmatch(ev); m != nil {
			flushChars()
			mergedCdataData += m[1]
			n, _ := strconv.Atoi(m[2])
			mergedCdataLen += n
		} else {
			flushChars()
			flushCdata()
			out = append(out, ev)
		}
	}
	flushChars()
	flushCdata()
	return strings.Join(out, "")
}

// normalizeCharDisplays replaces the display portion of merged
// SAX.characters() and SAX.cdata() events with a fixed placeholder.
// This accounts for display differences when events are merged from
// different split boundaries (htmlEncodeEntities truncation at 30 chars
// depends on entity encoding boundaries which shift when events split
// at different positions).
var reCharForNorm = regexp.MustCompile(`(?m)^(SAX\.characters\().*?(, \d+\))$`)
var reCdataForNorm = regexp.MustCompile(`(?m)^(SAX\.cdata\().*?(, \d+\))$`)

func normalizeCharDisplays(s string) string {
	s = reCharForNorm.ReplaceAllString(s, "${1}_${2}")
	s = reCdataForNorm.ReplaceAllString(s, "${1}_${2}")
	return s
}

// TestLibxml2CompatHTMLSAX runs helium's HTML SAX event stream against
// libxml2's HTML SAX golden files (.sax) in testdata/libxml2-compat/html/.
//
// Environment variable HELIUM_HTML_TEST_FILES can be set to test only
// specific files:
//
//	HELIUM_HTML_TEST_FILES=autoclose,implied1 go test -run TestLibxml2CompatHTMLSAX
func TestLibxml2CompatHTMLSAX(t *testing.T) {
	dir := "../testdata/libxml2-compat/html"

	if _, err := os.Stat(dir); err != nil {
		t.Skipf("testdata/libxml2-compat/html not found; run testdata/libxml2/generate.sh first")
	}

	skipped := map[string]string{}

	only := map[string]struct{}{}
	if v := os.Getenv("HELIUM_HTML_TEST_FILES"); v != "" {
		for _, f := range strings.Split(v, ",") {
			only[strings.TrimSpace(f)] = struct{}{}
		}
	}

	files, err := os.ReadDir(dir)
	require.NoError(t, err, "os.ReadDir should succeed")

	for _, fi := range files {
		if fi.IsDir() {
			continue
		}

		name := fi.Name()

		// Skip golden/err/expected files — only process HTML input files
		if strings.HasSuffix(name, ".sax") || strings.HasSuffix(name, ".err") ||
			strings.HasSuffix(name, ".expected") {
			continue
		}

		// Check if a SAX golden file exists for this input
		saxPath := filepath.Join(dir, name+".sax")
		if _, err := os.Stat(saxPath); err != nil {
			continue
		}

		// Strip extension for filtering
		baseName := strings.TrimSuffix(name, filepath.Ext(name))
		if len(only) > 0 {
			if _, ok := only[baseName]; !ok {
				if _, ok := only[name]; !ok {
					continue
				}
			}
		}

		if reason, ok := skipped[name]; ok {
			t.Logf("Skipping %s: %s", name, reason)
			continue
		}

		t.Run(name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic: %v", r)
				}
			}()

			input, err := os.ReadFile(filepath.Join(dir, name))
			require.NoError(t, err, "reading input file")

			expected, err := os.ReadFile(saxPath)
			require.NoError(t, err, "reading expected SAX file")

			var buf bytes.Buffer
			handler := newHTMLSAXEventEmitter(&buf)
			err = html.ParseWithSAX(input, handler)
			require.NoError(t, err, "ParseWithSAX should succeed (file = %s)", name)

			actual := buf.String()
			normalizedExpected := mergeHTMLCharEvents(string(expected))
			normalizedActual := mergeHTMLCharEvents(actual)

			// When events are merged from different split boundaries,
			// the display strings may differ while byte counts match.
			// Normalize character/cdata displays for comparison.
			cmpExpected := normalizeCharDisplays(normalizedExpected)
			cmpActual := normalizeCharDisplays(normalizedActual)

			if cmpExpected != cmpActual {
				errPath := filepath.Join(dir, name+".sax.actual")
				_ = os.WriteFile(errPath, []byte(actual), 0600)
				t.Logf("Actual output saved to %s", errPath)
			}
			require.Equal(t, cmpExpected, cmpActual,
				"HTML SAX event streams should match (file = %s)", name)
		})
	}
}

// TestHTMLSerialization parses HTML files to DOM, serializes with WriteDoc,
// and compares output against .expected golden files.
//
// Environment variable HELIUM_HTML_TEST_FILES can be set to test only
// specific files:
//
//	HELIUM_HTML_TEST_FILES=autoclose,entities go test -run TestHTMLSerialization
func TestHTMLSerialization(t *testing.T) {
	dir := "../testdata/libxml2-compat/html"

	if _, err := os.Stat(dir); err != nil {
		t.Skipf("testdata/libxml2-compat/html not found")
	}

	only := map[string]struct{}{}
	if v := os.Getenv("HELIUM_HTML_TEST_FILES"); v != "" {
		for _, f := range strings.Split(v, ",") {
			only[strings.TrimSpace(f)] = struct{}{}
		}
	}

	files, err := os.ReadDir(dir)
	require.NoError(t, err, "os.ReadDir should succeed")

	for _, fi := range files {
		if fi.IsDir() {
			continue
		}

		name := fi.Name()

		// Only process .html and .htm files
		if !strings.HasSuffix(name, ".html") && !strings.HasSuffix(name, ".htm") {
			continue
		}

		// Check if an .expected golden file exists
		expectedPath := filepath.Join(dir, name+".expected")
		if _, err := os.Stat(expectedPath); err != nil {
			continue
		}

		// Strip extension for filtering
		baseName := strings.TrimSuffix(name, filepath.Ext(name))
		if len(only) > 0 {
			if _, ok := only[baseName]; !ok {
				if _, ok := only[name]; !ok {
					continue
				}
			}
		}

		t.Run(name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic: %v", r)
				}
			}()

			doc, err := html.ParseFile(filepath.Join(dir, name))
			require.NoError(t, err, "ParseFile should succeed (file = %s)", name)

			var buf bytes.Buffer
			err = html.WriteDoc(&buf, doc)
			require.NoError(t, err, "WriteDoc should succeed (file = %s)", name)

			expected, err := os.ReadFile(expectedPath)
			require.NoError(t, err, "reading expected file")

			actual := buf.String()
			if actual != string(expected) {
				errPath := filepath.Join(dir, name+".dump.actual")
				_ = os.WriteFile(errPath, []byte(actual), 0600)
				t.Logf("Actual output saved to %s", errPath)
			}
			require.Equal(t, string(expected), actual,
				"HTML serialization should match (file = %s)", name)
		})
	}
}

// htmlError captures an error emitted by the HTML SAX parser.
type htmlError struct {
	line int
	col  int
	msg  string
}

// newHTMLErrorCollector returns a SAX handler that only collects errors,
// plus a pointer to the accumulated error slice.
func newHTMLErrorCollector() (html.SAXHandler, *[]htmlError) {
	var errors []htmlError
	var loc html.DocumentLocator

	s := &html.SAXCallbacks{}
	s.SetDocumentLocatorHandler = html.SetDocumentLocatorFunc(func(l html.DocumentLocator) error {
		loc = l
		return nil
	})
	s.ErrorHandler = html.ErrorFunc(func(msg string, args ...any) error {
		errors = append(errors, htmlError{
			line: loc.LineNumber(),
			col:  loc.ColumnNumber(),
			msg:  fmt.Sprintf(msg, args...),
		})
		return nil
	})
	return s, &errors
}

// formatHTMLErrors formats collected errors in libxml2's .err output format.
//
// Each error block is 3 lines:
//
//	./test/HTML/<filename>:<line>: HTML parser error : <message>
//	<source context up to 80 chars>
//	<spaces/tabs + ^ at error column>
//
// The context extraction matches libxml2's xmlParserInputGetWindow algorithm.
func formatHTMLErrors(filename string, input []byte, errors []htmlError) string {
	// Normalize line endings to match parser's normalization
	normalized := bytes.ReplaceAll(input, []byte("\r\n"), []byte("\n"))
	normalized = bytes.ReplaceAll(normalized, []byte("\r"), []byte("\n"))
	lines := bytes.Split(normalized, []byte("\n"))

	const maxCtx = 80 // sizeof(content) - 1 in libxml2

	var buf strings.Builder
	for _, e := range errors {
		// Header line
		fmt.Fprintf(&buf, "./test/HTML/%s:%d: HTML parser error : %s\n",
			filename, e.line, e.msg)

		lineIdx := e.line - 1
		if lineIdx < 0 || lineIdx >= len(lines) {
			buf.WriteString("\n^\n")
			continue
		}
		srcLine := lines[lineIdx]
		lineLen := len(srcLine)

		// errPos: 0-indexed position past last consumed byte (may equal lineLen)
		errPos := e.col - 1

		// Step 1: skip back over virtual newline (mirrors libxml2's skip-eol)
		adjustedPos := errPos
		if adjustedPos >= lineLen {
			if lineLen > 0 {
				adjustedPos = lineLen - 1
			} else {
				adjustedPos = 0
			}
		}

		// Step 2: walk back up to maxCtx bytes from adjustedPos
		start := max(0, adjustedPos-maxCtx)

		// Step 3: forward walk from start to end of line, limited to maxCtx bytes
		end := min(start+maxCtx, lineLen)
		content := srcLine[start:end]
		contentLen := len(content)

		// Step 4: caret column (using original errPos, not adjusted)
		col := errPos - start
		// Step 5: cap column if it exceeds content
		if col >= contentLen {
			if contentLen < maxCtx {
				col = contentLen
			} else {
				col = maxCtx - 1
			}
		}

		// For encoding errors, insert "Bytes:" line and truncate context
		// at the error position (matching libxml2's error.c behavior).
		if e.msg == "Invalid bytes in character encoding" {
			// Compute byte offset in normalized input
			byteOffset := 0
			for i := 0; i < lineIdx; i++ {
				byteOffset += len(lines[i]) + 1 // +1 for \n separator
			}
			byteOffset += errPos

			endOfs := byteOffset + 4
			if endOfs > len(normalized) {
				endOfs = len(normalized)
			}
			rawBytes := normalized[byteOffset:endOfs]
			var parts []string
			for _, b := range rawBytes {
				parts = append(parts, fmt.Sprintf("0x%02X", b))
			}
			fmt.Fprintf(&buf, "Bytes: %s\n", strings.Join(parts, " "))

			// Show only content before the invalid byte
			if col < len(content) {
				content = content[:col]
			}
		}

		// Write context line
		buf.Write(content)
		buf.WriteByte('\n')

		// Write caret line (preserving tabs from source)
		for i := 0; i < col; i++ {
			if i < contentLen && content[i] == '\t' {
				buf.WriteByte('\t')
			} else {
				buf.WriteByte(' ')
			}
		}
		buf.WriteString("^\n")
	}
	return buf.String()
}

// TestHTMLErrors parses HTML files with SAX error collection, formats
// errors in libxml2 style, and compares against .err golden files.
//
// Environment variable HELIUM_HTML_TEST_FILES can be set to test only
// specific files:
//
//	HELIUM_HTML_TEST_FILES=reg4,test3 go test -run TestHTMLErrors
func TestHTMLErrors(t *testing.T) {
	dir := "../testdata/libxml2-compat/html"

	if _, err := os.Stat(dir); err != nil {
		t.Skipf("testdata/libxml2-compat/html not found; run testdata/libxml2/generate.sh first")
	}

	skipped := map[string]string{}

	only := map[string]struct{}{}
	if v := os.Getenv("HELIUM_HTML_TEST_FILES"); v != "" {
		for _, f := range strings.Split(v, ",") {
			only[strings.TrimSpace(f)] = struct{}{}
		}
	}

	errFiles, err := filepath.Glob(filepath.Join(dir, "*.err"))
	require.NoError(t, err, "filepath.Glob should succeed")

	for _, errFile := range errFiles {
		// Derive input filename by stripping .err suffix
		name := strings.TrimSuffix(filepath.Base(errFile), ".err")

		// Strip extension for filtering
		baseName := strings.TrimSuffix(name, filepath.Ext(name))
		if len(only) > 0 {
			if _, ok := only[baseName]; !ok {
				if _, ok := only[name]; !ok {
					continue
				}
			}
		}

		if reason, ok := skipped[name]; ok {
			t.Logf("Skipping %s: %s", name, reason)
			continue
		}

		t.Run(name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic: %v", r)
				}
			}()

			input, err := os.ReadFile(filepath.Join(dir, name))
			require.NoError(t, err, "reading input file")

			expected, err := os.ReadFile(errFile)
			require.NoError(t, err, "reading expected .err file")

			handler, errors := newHTMLErrorCollector()
			err = html.ParseWithSAX(input, handler)
			require.NoError(t, err, "ParseWithSAX should succeed (file = %s)", name)

			actual := formatHTMLErrors(name, input, *errors)

			if actual != string(expected) {
				actualPath := filepath.Join(dir, name+".err.actual")
				_ = os.WriteFile(actualPath, []byte(actual), 0600)
				t.Logf("Actual output saved to %s", actualPath)
			}
			require.Equal(t, string(expected), actual,
				"HTML error output should match (file = %s)", name)
		})
	}
}
