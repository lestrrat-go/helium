package html_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/html"

	"github.com/stretchr/testify/require"
)

func TestZeroValueParser(t *testing.T) {
	var p html.Parser
	doc, err := p.Parse(t.Context(), []byte(`<p>hello</p>`))
	require.NoError(t, err)
	require.NotNil(t, doc)
}

func TestZeroValueParserFluent(t *testing.T) {
	var p html.Parser
	doc, err := p.SuppressImplied(true).Parse(t.Context(), []byte(`<p>hello</p>`))
	require.NoError(t, err)
	require.Equal(t, "p", doc.FirstChild().Name())
}

func TestZeroValueWriter(t *testing.T) {
	doc, err := html.NewParser().Parse(t.Context(), []byte(`<html><body><p>hi</p></body></html>`))
	require.NoError(t, err)

	var w html.Writer
	var buf bytes.Buffer
	require.NoError(t, w.WriteTo(&buf, doc))
	require.Contains(t, buf.String(), "<p>hi</p>")
}

func TestOptionsNoImplied(t *testing.T) {
	doc, err := html.NewParser().SuppressImplied(true).Parse(t.Context(), []byte(`<p>hello</p>`))
	require.NoError(t, err)

	// Without NoImplied, the parser would insert <html><body> around <p>.
	// With NoImplied, the first child should be <p> directly.
	first := doc.FirstChild()
	require.NotNil(t, first, "document should have a child")
	require.Equal(t, "p", first.Name())
}

func TestOptionsNoDefaultDTD(t *testing.T) {
	// Parse a document without any DOCTYPE
	doc, err := html.NewParser().Parse(t.Context(), []byte(`<html><body><p>hi</p></body></html>`))
	require.NoError(t, err)

	// Without NoDefaultDTD, serialization adds a default DOCTYPE
	var withDTD bytes.Buffer
	require.NoError(t, html.NewWriter().WriteTo(&withDTD, doc))
	require.True(t, strings.Contains(withDTD.String(), "<!DOCTYPE"), "default should include DOCTYPE")

	// With NoDefaultDTD, no DOCTYPE in output
	var noDTD bytes.Buffer
	require.NoError(t, html.NewWriter().DefaultDTD(false).WriteTo(&noDTD, doc))
	require.False(t, strings.Contains(noDTD.String(), "<!DOCTYPE"), "WithNoDefaultDTD should suppress DOCTYPE")
}

func TestWriteNodeDocumentPreservesWriterOptions(t *testing.T) {
	doc := helium.NewHTMLDocument()

	root := doc.CreateElement("HTML")

	body := doc.CreateElement("Body")

	link := doc.CreateElement("A")
	_ = link.SetLiteralAttribute("HREF", "caf\u00e9")

	text := doc.CreateText([]byte("\u0080"))
	require.NoError(t, link.AddChild(text))
	require.NoError(t, body.AddChild(link))
	require.NoError(t, root.AddChild(body))
	require.NoError(t, doc.SetDocumentElement(root))

	writer := html.NewWriter().
		DefaultDTD(false).
		Format(false).
		PreserveCase(true).
		EscapeURIAttributes(false).
		EscapeControlChars(true)

	var want bytes.Buffer
	require.NoError(t, writer.WriteTo(&want, doc))

	var got bytes.Buffer
	require.NoError(t, writer.WriteTo(&got, doc))

	require.Equal(t, want.String(), got.String())
	require.Equal(t, "<HTML><Body><A HREF=\"caf\u00e9\">&#x80;</A></Body></HTML>", got.String())
}

func TestWriteLegacyCompatSuppressed(t *testing.T) {
	input := `<!DOCTYPE html SYSTEM "about:legacy-compat"><html><body><p>hi</p></body></html>`
	doc, err := html.NewParser().Parse(t.Context(), []byte(input))
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, html.NewWriter().WriteTo(&buf, doc))
	output := buf.String()
	require.True(t, strings.HasPrefix(output, "<!DOCTYPE html>\n"), "about:legacy-compat SYSTEM ID should be suppressed, got: %s", output)
}

func TestWriteNameAttrURIOnAnchor(t *testing.T) {
	// "name" on <a> should be URI-escaped (space -> %20)
	input := `<html><body><a name="foo bar">link</a></body></html>`
	doc, err := html.NewParser().Parse(t.Context(), []byte(input))
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, html.NewWriter().DefaultDTD(false).WriteTo(&buf, doc))
	output := buf.String()
	require.Contains(t, output, `name="foo%20bar"`, "name on <a> should be URI-escaped")
}

func TestWriteNameAttrNonURIOnMeta(t *testing.T) {
	// "name" on non-<a> elements should NOT be URI-escaped
	input := `<html><head><meta name="foo bar"></head><body></body></html>`
	doc, err := html.NewParser().Parse(t.Context(), []byte(input))
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, html.NewWriter().DefaultDTD(false).WriteTo(&buf, doc))
	output := buf.String()
	require.Contains(t, output, `name="foo bar"`, "name on <meta> should not be URI-escaped")
}

func TestDuplicateAttrKeepsFirst(t *testing.T) {
	// libxml2 keeps the first occurrence and silently drops duplicates.
	input := `<html><body><div class="first" class="second">x</div></body></html>`
	doc, err := html.NewParser().Parse(t.Context(), []byte(input))
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, html.NewWriter().DefaultDTD(false).WriteTo(&buf, doc))
	output := buf.String()
	require.Contains(t, output, `class="first"`, "first occurrence should be kept")
	require.NotContains(t, output, `class="second"`, "duplicate should be dropped")
}

func TestDuplicateAttrCaseInsensitive(t *testing.T) {
	// HTML attribute names are case-insensitive; CLASS and class are the same.
	input := `<html><body><div CLASS="upper" class="lower">x</div></body></html>`
	doc, err := html.NewParser().Parse(t.Context(), []byte(input))
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, html.NewWriter().DefaultDTD(false).WriteTo(&buf, doc))
	output := buf.String()
	require.Contains(t, output, `class="upper"`, "first (case-insensitive) should be kept")
	require.NotContains(t, output, `class="lower"`, "duplicate should be dropped")
}

func TestOptionsNoBlanks(t *testing.T) {
	input := `<html> <body> <p>text</p> </body> </html>`
	doc, err := html.NewParser().StripBlanks(true).Parse(t.Context(), []byte(input))
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, html.NewWriter().DefaultDTD(false).WriteTo(&buf, doc))
	output := buf.String()

	// The output should not contain whitespace between tags
	// (the original spaces " " between <html> and <body>, etc. should be stripped)
	require.False(t, strings.Contains(output, "<html> <body>"),
		"whitespace-only text nodes should be stripped")
	require.True(t, strings.Contains(output, "text"),
		"non-whitespace text should be preserved")
}

func TestOptionsNoBlanksTinyChunkPreservesLeadingSpace(t *testing.T) {
	// Regression: chunking normal data-state text at MaxContentSize must not
	// suppress significant leading whitespace under StripBlanks. With a tiny
	// cap, " a" would otherwise split into a whitespace-only chunk (wrongly
	// stripped) and "a". The run as a whole is not all-whitespace, so the
	// leading space must be preserved.
	var collected []byte
	sax := &html.SAXCallbacks{}
	sax.SetOnCharacters(html.CharactersFunc(func(ch []byte) error {
		collected = append(collected, ch...)
		return nil
	}))

	input := `<p> a</p>`
	err := html.NewParser().
		StripBlanks(true).
		MaxContentSize(1).
		ParseWithSAX(t.Context(), []byte(input), sax)
	require.NoError(t, err)
	require.Equal(t, " a", string(collected),
		"significant leading whitespace must survive tiny-chunk StripBlanks")
}

func TestOptionsNoBlanksOverCapHardErrors(t *testing.T) {
	// Under StripBlanks a text run's whitespace-significance can only be decided
	// after the whole run is known, so the cap may not split it early. But a run
	// whose leading whitespace prefix alone overruns the cap (with yet more
	// whitespace beyond it) cannot be decided without unbounded buffering, so it
	// must HARD-FAIL with ErrContentSizeExceeded rather than buffer the run whole.
	input := `<p>  a</p>`
	_, err := html.NewParser().
		StripBlanks(true).
		MaxContentSize(1).
		Parse(t.Context(), []byte(input))
	require.ErrorIs(t, err, html.ErrContentSizeExceeded,
		"over-cap whitespace prefix under StripBlanks must fail, not buffer unbounded")
}

func TestOptionsNoBlanksTinyChunkWhitespaceAcrossChunks(t *testing.T) {
	// Whitespace-significance must persist across the capped chunks of ONE text
	// run. With MaxContentSize(1) the parser re-enters parseCharacters per byte,
	// so it must remember a run is already significant once its first
	// non-whitespace byte has been emitted. Otherwise a TRAILING whitespace chunk
	// would be wrongly suppressed and an INTERIOR whitespace chunk would wrongly
	// hard-fail.
	for _, tc := range []struct {
		name  string
		input string
		want  string
	}{
		{name: "trailing", input: `<p>a </p>`, want: "a "},
		{name: "interior", input: `<p>a  b</p>`, want: "a  b"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var collected []byte
			sax := &html.SAXCallbacks{}
			sax.SetOnCharacters(html.CharactersFunc(func(ch []byte) error {
				collected = append(collected, ch...)
				return nil
			}))

			err := html.NewParser().
				StripBlanks(true).
				MaxContentSize(1).
				ParseWithSAX(t.Context(), []byte(tc.input), sax)
			require.NoError(t, err,
				"a significant run must not hard-fail on a later whitespace chunk")
			require.Equal(t, tc.want, string(collected),
				"whitespace of a known-significant run must survive across chunks")
		})
	}
}

func TestOptionsNoBlanksTinyChunkSignificanceAcrossEmbeds(t *testing.T) {
	// Run significance must be remembered across EVERY non-whitespace emit path,
	// not just plain text chunks. A char-ref (entity output) and a lone literal
	// '<' are part of the SAME normal-data text run; once any of them has emitted
	// non-whitespace, a later over-cap whitespace chunk must NOT hard-fail and
	// must NOT be suppressed. Under StripBlanks+MaxContentSize(1) these inputs
	// previously hard-failed with ErrContentSizeExceeded because the flag was
	// cleared before char-ref / lone-'<' resolution and never re-marked.
	for _, tc := range []struct {
		name  string
		input string
		want  string
	}{
		{name: "entity-then-ws-text", input: `<p>a&amp;  b</p>`, want: "a&  b"},
		{name: "entity-only-then-ws-text", input: `<p>&amp;  b</p>`, want: "&  b"},
		{name: "lone-lt-then-ws-text", input: `<p>a<  b</p>`, want: "a<  b"},
		{name: "lone-lt-only-then-ws-text", input: `<p><  b</p>`, want: "<  b"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var collected []byte
			sax := &html.SAXCallbacks{}
			sax.SetOnCharacters(html.CharactersFunc(func(ch []byte) error {
				collected = append(collected, ch...)
				return nil
			}))

			err := html.NewParser().
				StripBlanks(true).
				MaxContentSize(1).
				ParseWithSAX(t.Context(), []byte(tc.input), sax)
			require.NoError(t, err,
				"a run made significant by entity/lone-'<' output must not hard-fail on later whitespace")
			require.Equal(t, tc.want, string(collected),
				"whitespace following entity/lone-'<' output must survive across chunks")
		})
	}
}

func TestOptionsNoError(t *testing.T) {
	var errorCalled bool
	sax := &html.SAXCallbacks{}
	sax.SetOnError(html.ErrorFunc(func(err error) error {
		errorCalled = true
		return nil
	}))

	// Parse malformed HTML that would normally trigger errors
	// (e.g., unexpected end tag)
	input := `<html><body></nonexistent></body></html>`
	err := html.NewParser().SuppressErrors(true).ParseWithSAX(t.Context(), []byte(input), sax)
	require.NoError(t, err)
	require.False(t, errorCalled, "error handler should not be called with WithNoError")
}

func TestOptionsNoErrorDefault(t *testing.T) {
	var errorCalled bool
	sax := &html.SAXCallbacks{}
	sax.SetOnError(html.ErrorFunc(func(err error) error {
		errorCalled = true
		return nil
	}))

	// Without NoError, the error handler should be called
	input := `<html><body></nonexistent></body></html>`
	err := html.NewParser().ParseWithSAX(t.Context(), []byte(input), sax)
	require.NoError(t, err)
	require.True(t, errorCalled, "error handler should be called without WithNoError")
}

func TestOptionsNoWarning(t *testing.T) {
	var warningCalled bool
	sax := &html.SAXCallbacks{}
	sax.SetOnWarning(html.WarningFunc(func(err error) error {
		warningCalled = true
		return nil
	}))

	// Parse valid HTML with WithNoWarning
	input := `<html><body><p>hello</p></body></html>`
	err := html.NewParser().SuppressWarnings(true).ParseWithSAX(t.Context(), []byte(input), sax)
	require.NoError(t, err)
	require.False(t, warningCalled, "warning handler should not be called with WithNoWarning")
}

func TestParseFileSetsURL(t *testing.T) {
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "test.html")
	require.NoError(t, os.WriteFile(f, []byte(`<html><body><p>hi</p></body></html>`), 0o644))

	doc, err := html.NewParser().ParseFile(t.Context(), f)
	require.NoError(t, err)

	abs, err := filepath.Abs(f)
	require.NoError(t, err)
	require.Equal(t, abs, doc.URL())
}

func TestParseFileMissingFileErrors(t *testing.T) {
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "does-not-exist.html")

	_, err := html.NewParser().ParseFile(t.Context(), f)
	require.Error(t, err, "parsing a missing file must error")
}

func TestOptionsPushParserCarriesOptions(t *testing.T) {
	pp := html.NewParser().SuppressImplied(true).NewPushParser(t.Context())
	require.NoError(t, pp.Push([]byte(`<p>hello</p>`)))
	doc, err := pp.Close()
	require.NoError(t, err)

	first := doc.FirstChild()
	require.NotNil(t, first)
	require.Equal(t, "p", first.Name())
}
