package helium_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// BenchmarkWriteNonASCII serializes a document containing many non-ASCII
// characters with EscapeNonASCII enabled, exercising the hex char ref path.
func BenchmarkWriteNonASCII(b *testing.B) {
	var buf strings.Builder
	buf.WriteString("<root>")
	for range 200 {
		buf.WriteString("<t>caf\u00e9 na\u00efve r\u00e9sum\u00e9 \u00fcber \u00e0 \u00e7a \u00f1</t>")
	}
	buf.WriteString("</root>")

	doc, err := helium.NewParser().Parse(b.Context(), []byte(buf.String()))
	require.NoError(b, err)

	w := helium.NewWriter().EscapeNonASCII(true)

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		err := w.WriteTo(io.Discard, doc)
		require.NoError(b, err)
	}
}

var errNamespaceWrite = errors.New("namespace write failed")

type namespaceFailWriter struct {
	failOn string
	tail   string
}

func (w *namespaceFailWriter) Write(p []byte) (int, error) {
	window := w.tail + string(p)
	if strings.Contains(window, w.failOn) {
		return 0, errNamespaceWrite
	}
	if keep := len(w.failOn) - 1; keep > 0 {
		if len(window) > keep {
			w.tail = window[len(window)-keep:]
		} else {
			w.tail = window
		}
	}
	return len(p), nil
}

// errShortWrite is returned by failAfterNWriter once its byte budget runs out.
var errShortWrite = errors.New("short write")

// failAfterNWriter accepts up to limit bytes and then fails every subsequent
// write, simulating an io.Writer that breaks mid-stream.
type failAfterNWriter struct {
	limit   int
	written int
}

func (w *failAfterNWriter) Write(p []byte) (int, error) {
	if w.written >= w.limit {
		return 0, errShortWrite
	}
	remaining := w.limit - w.written
	if len(p) <= remaining {
		w.written += len(p)
		return len(p), nil
	}
	w.written = w.limit
	return remaining, errShortWrite
}

// docWithEverything builds a document that exercises the XML declaration, a
// DTD (with ENTITY/ELEMENT/ATTLIST/NOTATION decls), comments, PIs, an entity
// reference, CDATA, and nested elements with attributes.
const docWithEverything = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE root [
<!ELEMENT root (child*)>
<!ATTLIST root id CDATA #IMPLIED>
<!ENTITY greeting "hello">
<!NOTATION gif SYSTEM "image/gif">
]>
<!--a top level comment-->
<?app instruction?>
<root id="r1"><child>&greeting; world</child><child><![CDATA[<raw> & data]]></child></root>`

// noEmptyWriteWriter is a strict io.Writer that rejects a zero-length Write,
// mirroring writers that treat an empty chunk as an error. It records the bytes
// written so callers can assert on the serialized output.
type noEmptyWriteWriter struct {
	buf bytes.Buffer
}

func (w *noEmptyWriteWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, errors.New("empty write rejected")
	}
	return w.buf.Write(p)
}

// failOnSubstringWriter fails the first Write whose accumulated tail+payload
// contains trigger, and accepts everything else. It is used to make a specific
// serialization step fail while earlier steps succeed.
type failOnSubstringWriter struct {
	trigger string
	tail    string
	writes  int
}

func (w *failOnSubstringWriter) Write(p []byte) (int, error) {
	w.writes++
	window := w.tail + string(p)
	if strings.Contains(window, w.trigger) {
		return 0, errShortWrite
	}
	if keep := len(w.trigger) - 1; keep > 0 {
		if len(window) > keep {
			w.tail = window[len(window)-keep:]
		} else {
			w.tail = window
		}
	}
	return len(p), nil
}

func TestSerialize(t *testing.T) {
	t.Parallel()

	t.Run("XML to DOM to XML", func(t *testing.T) {
		skipped := map[string]struct{}{}
		only := map[string]struct{}{}
		if v := os.Getenv("HELIUM_DUMP_TEST_FILES"); v != "" {
			for f := range strings.SplitSeq(v, ",") {
				n := strings.TrimSpace(f)
				only[n] = struct{}{}
			}
		}

		dir := "test"
		files, err := os.ReadDir(dir)
		require.NoError(t, err, "os.ReadDir should succeed")

		for _, fi := range files {
			if fi.IsDir() {
				continue
			}

			if len(only) > 0 {
				if _, ok := only[fi.Name()]; !ok {
					continue
				}
			} else {
				if _, ok := skipped[fi.Name()]; ok {
					t.Logf("Skipping test for '%s' for now...", fi.Name())
					continue
				}
			}

			fn := filepath.Join(dir, fi.Name())
			if !strings.HasSuffix(fn, ".xml") {
				continue
			}

			goldenfn := strings.ReplaceAll(fn, ".xml", ".dump")
			if _, err := os.Stat(goldenfn); err != nil {
				t.Logf("%s does not exist, skipping...", goldenfn)
				continue
			}
			golden, err := os.ReadFile(goldenfn)
			require.NoError(t, err, "os.ReadFile should succeed")

			t.Logf("Parsing %s...", fn)
			in, err := os.ReadFile(fn)
			require.NoError(t, err, "os.ReadFile should succeed")

			doc, err := helium.NewParser().Parse(t.Context(), in)
			require.NoError(t, err, `Parse(...) succeeds`)

			str, err := helium.WriteString(doc)
			require.NoError(t, err, "XMLString(doc) succeeds")

			if string(golden) != str {
				errout, err := os.OpenFile(fn+".err", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
				if err != nil {
					t.Logf("Failed to file to save output: %s", err)
					return
				}
				defer func() { _ = errout.Close() }()

				_, _ = errout.WriteString(str)
			}
			require.Equal(t, string(golden), str, "roundtrip works")
		}
	})

	t.Run("DOM to XML", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		//	defer doc.Free()

		root, err := doc.CreateElement("root")
		require.NoError(t, err)

		require.NoError(t, doc.SetDocumentElement(root))
		require.NoError(t, root.AppendText([]byte(`Hello, World!`)))

		str, err := helium.WriteString(doc)
		require.NoError(t, err, "XMLString(doc) succeeds")

		t.Logf("%s", str)
	})

	// WriteString on a programmatically built doc.
	t.Run("without a DTD", func(t *testing.T) {
		doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
		root, err := doc.CreateElement("root")
		require.NoError(t, err)
		require.NoError(t, doc.AddChild(root))
		require.NoError(t, root.AppendText([]byte("text & more")))

		s, err := helium.WriteString(doc)
		require.NoError(t, err)
		require.True(t, strings.Contains(s, "<root>"))
		require.Contains(t, s, "&amp;")
	})

	t.Run("format output", func(t *testing.T) {
		t.Run("nested elements", func(t *testing.T) {
			t.Parallel()

			doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?><root><child><grandchild/></child></root>`))
			require.NoError(t, err)

			var buf strings.Builder
			require.NoError(t, helium.NewWriter().Format(true).WriteTo(&buf, doc))
			str := buf.String()

			expected := "<?xml version=\"1.0\"?>\n<root>\n  <child>\n    <grandchild/>\n  </child>\n</root>\n"
			require.Equal(t, expected, str)
		})

		t.Run("text-only element stays inline", func(t *testing.T) {
			t.Parallel()

			doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?><root><child>hello</child></root>`))
			require.NoError(t, err)

			var buf strings.Builder
			require.NoError(t, helium.NewWriter().Format(true).WriteTo(&buf, doc))
			str := buf.String()

			expected := "<?xml version=\"1.0\"?>\n<root>\n  <child>hello</child>\n</root>\n"
			require.Equal(t, expected, str)
		})

		t.Run("custom indent string", func(t *testing.T) {
			t.Parallel()

			doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?><root><child><grandchild/></child></root>`))
			require.NoError(t, err)

			var buf strings.Builder
			require.NoError(t, helium.NewWriter().Format(true).IndentString("\t").WriteTo(&buf, doc))
			str := buf.String()

			expected := "<?xml version=\"1.0\"?>\n<root>\n\t<child>\n\t\t<grandchild/>\n\t</child>\n</root>\n"
			require.Equal(t, expected, str)
		})

		t.Run("explicit empty indent yields newlines with no indentation", func(t *testing.T) {
			t.Parallel()

			doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?><root><child><grandchild/></child></root>`))
			require.NoError(t, err)

			var buf strings.Builder
			require.NoError(t, helium.NewWriter().Format(true).IndentString("").WriteTo(&buf, doc))

			expected := "<?xml version=\"1.0\"?>\n<root>\n<child>\n<grandchild/>\n</child>\n</root>\n"
			require.Equal(t, expected, buf.String())
		})

		t.Run("explicit empty indent never writes an empty chunk", func(t *testing.T) {
			t.Parallel()

			doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?><root><child><grandchild/></child></root>`))
			require.NoError(t, err)

			w := &noEmptyWriteWriter{}
			require.NoError(t, helium.NewWriter().Format(true).IndentString("").WriteTo(w, doc))

			expected := "<?xml version=\"1.0\"?>\n<root>\n<child>\n<grandchild/>\n</child>\n</root>\n"
			require.Equal(t, expected, w.buf.String())
		})

		t.Run("unset indent uses two-space default", func(t *testing.T) {
			t.Parallel()

			doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?><root><child><grandchild/></child></root>`))
			require.NoError(t, err)

			var buf strings.Builder
			require.NoError(t, helium.NewWriter().Format(true).WriteTo(&buf, doc))

			expected := "<?xml version=\"1.0\"?>\n<root>\n  <child>\n    <grandchild/>\n  </child>\n</root>\n"
			require.Equal(t, expected, buf.String())
		})

		t.Run("without format stays compact", func(t *testing.T) {
			t.Parallel()

			doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?><root><child><grandchild/></child></root>`))
			require.NoError(t, err)

			str, err := helium.WriteString(doc)
			require.NoError(t, err)

			expected := "<?xml version=\"1.0\"?>\n<root><child><grandchild/></child></root>\n"
			require.Equal(t, expected, str)
		})

		t.Run("multiple children", func(t *testing.T) {
			t.Parallel()

			doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?><root><a/><b/><c/></root>`))
			require.NoError(t, err)

			var buf strings.Builder
			require.NoError(t, helium.NewWriter().Format(true).WriteTo(&buf, doc))
			str := buf.String()

			expected := "<?xml version=\"1.0\"?>\n<root>\n  <a/>\n  <b/>\n  <c/>\n</root>\n"
			require.Equal(t, expected, str)
		})

		t.Run("element XMLString with format", func(t *testing.T) {
			t.Parallel()

			doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?><root><child><grandchild/></child></root>`))
			require.NoError(t, err)

			root := doc.DocumentElement()
			require.NotNil(t, root)

			var buf strings.Builder
			require.NoError(t, helium.NewWriter().Format(true).WriteTo(&buf, root))
			str := buf.String()

			expected := "<root>\n  <child>\n    <grandchild/>\n  </child>\n</root>"
			require.Equal(t, expected, str)
		})

		t.Run("comment and PI children", func(t *testing.T) {
			t.Parallel()

			doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?><root><!--comment--><child/><?pi data?></root>`))
			require.NoError(t, err)

			var buf strings.Builder
			require.NoError(t, helium.NewWriter().Format(true).WriteTo(&buf, doc))
			str := buf.String()

			expected := "<?xml version=\"1.0\"?>\n<root>\n  <!--comment-->\n  <child/>\n  <?pi data?>\n</root>\n"
			require.Equal(t, expected, str)
		})

		t.Run("deeply nested", func(t *testing.T) {
			t.Parallel()

			doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?><a><b><c><d>text</d></c></b></a>`))
			require.NoError(t, err)

			var buf strings.Builder
			require.NoError(t, helium.NewWriter().Format(true).WriteTo(&buf, doc))
			str := buf.String()

			expected := "<?xml version=\"1.0\"?>\n<a>\n  <b>\n    <c>\n      <d>text</d>\n    </c>\n  </b>\n</a>\n"
			require.Equal(t, expected, str)
		})

		t.Run("mixed content stays inline", func(t *testing.T) {
			t.Parallel()

			const input = `<resources><string name="welcome">Hello <b>world</b></string><version>1.0</version></resources>`

			doc, err := helium.NewParser().StripBlanks(true).Parse(t.Context(), []byte(input))
			require.NoError(t, err)

			var buf strings.Builder
			require.NoError(t, helium.NewWriter().Format(true).IndentString("  ").XMLDeclaration(false).WriteTo(&buf, doc))

			// The mixed-content <string> element (non-whitespace text alongside a <b>
			// child) must not have indentation injected around its children — doing so
			// would corrupt "Hello " into "Hello\n      ". Only the pure-element
			// container <resources> is formatted.
			expected := "<resources>\n  <string name=\"welcome\">Hello <b>world</b></string>\n  <version>1.0</version>\n</resources>\n"
			require.Equal(t, expected, buf.String())
		})

		t.Run("mixed content format is idempotent", func(t *testing.T) {
			t.Parallel()

			const input = `<resources><string name="welcome">Hello <b>world</b></string><version>1.0</version></resources>`

			format := func(src []byte) string {
				doc, err := helium.NewParser().StripBlanks(true).Parse(t.Context(), src)
				require.NoError(t, err)
				var buf strings.Builder
				require.NoError(t, helium.NewWriter().Format(true).IndentString("  ").XMLDeclaration(false).WriteTo(&buf, doc))
				return buf.String()
			}

			first := format([]byte(input))
			// Re-parsing and re-formatting the already-formatted output must yield the
			// exact same bytes; injected whitespace inside mixed content would become a
			// real text node on reparse and compound on each pass.
			second := format([]byte(first))
			require.Equal(t, first, second)
		})

		t.Run("mixed content suppresses formatting subtree-wide", func(t *testing.T) {
			t.Parallel()

			// A pure-element descendant (<b> holding only <i/>) nested inside a
			// mixed-content element (<p>) must NOT be formatted: libxml2 disables
			// formatting for the whole subtree of a mixed element until it closes, so
			// no whitespace may be injected anywhere inside <p>.
			const input = `<p>left<b><i/></b>right</p>`

			doc, err := helium.NewParser().StripBlanks(true).Parse(t.Context(), []byte(input))
			require.NoError(t, err)

			var buf strings.Builder
			require.NoError(t, helium.NewWriter().Format(true).IndentString("  ").XMLDeclaration(false).WriteTo(&buf, doc))

			expected := "<p>left<b><i/></b>right</p>\n"
			require.Equal(t, expected, buf.String())
		})

		t.Run("direct CDATA child suppresses formatting", func(t *testing.T) {
			t.Parallel()

			// A direct CDATASection child is one of libxml2's mixed-content triggers
			// (xmlNodeDumpOutputInternal treats TEXT, CDATA, and ENTITY_REF children
			// alike). <data> mixes a pure-element child (<child> holding a nested
			// <nested/>) ALONGSIDE the CDATA section, so hasOnlyTextChildren is false —
			// only hasTextlikeChild keeps the subtree inline. The whole <data> subtree,
			// including the nested element child, must serialize verbatim with no
			// indentation injected, while the pure-element container <resources> is
			// still formatted.
			const input = `<resources><data><child><nested/></child><![CDATA[<raw> & data]]></data><version>1.0</version></resources>`

			doc, err := helium.NewParser().StripBlanks(true).Parse(t.Context(), []byte(input))
			require.NoError(t, err)

			var buf strings.Builder
			require.NoError(t, helium.NewWriter().Format(true).IndentString("  ").XMLDeclaration(false).WriteTo(&buf, doc))

			expected := "<resources>\n  <data><child><nested/></child><![CDATA[<raw> & data]]></data>\n  <version>1.0</version>\n</resources>\n"
			require.Equal(t, expected, buf.String())
		})

		t.Run("direct entity-reference child suppresses formatting", func(t *testing.T) {
			t.Parallel()

			// A direct EntityRef child is a libxml2 mixed-content trigger. <msg> mixes a
			// pure-element child (<child> holding a nested <nested/>) ALONGSIDE an
			// unexpanded &g; reference (default parser keeps entity references as nodes),
			// so hasOnlyTextChildren is false — only hasTextlikeChild keeps the subtree
			// inline. The whole <msg> subtree, including the nested element child,
			// serializes verbatim with no indentation, while the container <resources>
			// is formatted.
			const input = `<!DOCTYPE resources [<!ENTITY g "hi there">]><resources><msg><child><nested/></child>&g;</msg><version>1.0</version></resources>`

			doc, err := helium.NewParser().StripBlanks(true).Parse(t.Context(), []byte(input))
			require.NoError(t, err)

			var buf strings.Builder
			require.NoError(t, helium.NewWriter().Format(true).IndentString("  ").XMLDeclaration(false).WriteTo(&buf, doc))

			expected := "<!DOCTYPE resources [\n<!ENTITY g \"hi there\">\n]>\n<resources>\n  <msg><child><nested/></child>&g;</msg>\n  <version>1.0</version>\n</resources>\n"
			require.Equal(t, expected, buf.String())
		})

		t.Run("whitespace-only text child suppresses formatting", func(t *testing.T) {
			t.Parallel()

			// A whitespace-only text child is still a TEXT node, so it triggers
			// libxml2's mixed-content suppression. <item> mixes a pure-element child
			// (<child> holding a nested <nested/>) ALONGSIDE a whitespace-only text
			// node, so hasOnlyTextChildren is false — only hasTextlikeChild keeps the
			// subtree inline. The significant whitespace must be preserved verbatim (no
			// indentation injected) and the nested element child must not be formatted,
			// which would corrupt the content and break idempotence. Blanks are NOT
			// stripped here so the text child survives.
			const input = `<resources><item><child><nested/></child>   </item><version>1.0</version></resources>`

			doc, err := helium.NewParser().StripBlanks(false).Parse(t.Context(), []byte(input))
			require.NoError(t, err)

			var buf strings.Builder
			require.NoError(t, helium.NewWriter().Format(true).IndentString("  ").XMLDeclaration(false).WriteTo(&buf, doc))

			expected := "<resources>\n  <item><child><nested/></child>   </item>\n  <version>1.0</version>\n</resources>\n"
			require.Equal(t, expected, buf.String())
		})

		t.Run("pure-element child stays formatted", func(t *testing.T) {
			t.Parallel()

			// Control for the suppression cases above: with the same container shape
			// but a pure-element child <a><b/></a> (no text-like child anywhere), the
			// mixed-content rule does NOT fire, so <a> is formatted with normal nested
			// indentation.
			const input = `<resources><a><b/></a><version>1.0</version></resources>`

			doc, err := helium.NewParser().StripBlanks(true).Parse(t.Context(), []byte(input))
			require.NoError(t, err)

			var buf strings.Builder
			require.NoError(t, helium.NewWriter().Format(true).IndentString("  ").XMLDeclaration(false).WriteTo(&buf, doc))

			expected := "<resources>\n  <a>\n    <b/>\n  </a>\n  <version>1.0</version>\n</resources>\n"
			require.Equal(t, expected, buf.String())
		})

		t.Run("pure-text child stays inline", func(t *testing.T) {
			t.Parallel()

			// Control: a child with ONLY text content (<a>text</a>) is emitted inline
			// (text-only elements never get their sole text wrapped in indentation),
			// but this is the ordinary text-only path, not subtree-wide suppression:
			// the container <resources> stays formatted.
			const input = `<resources><a>text</a><version>1.0</version></resources>`

			doc, err := helium.NewParser().StripBlanks(true).Parse(t.Context(), []byte(input))
			require.NoError(t, err)

			var buf strings.Builder
			require.NoError(t, helium.NewWriter().Format(true).IndentString("  ").XMLDeclaration(false).WriteTo(&buf, doc))

			expected := "<resources>\n  <a>text</a>\n  <version>1.0</version>\n</resources>\n"
			require.Equal(t, expected, buf.String())
		})
	})

	t.Run("no-empty toggle", func(t *testing.T) {
		t.Run("empty element uses open+close tags", func(t *testing.T) {
			t.Parallel()

			doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?><root><br/></root>`))
			require.NoError(t, err)

			var buf strings.Builder
			require.NoError(t, helium.NewWriter().SelfCloseEmptyElements(false).WriteTo(&buf, doc))
			str := buf.String()

			expected := "<?xml version=\"1.0\"?>\n<root><br></br></root>\n"
			require.Equal(t, expected, str)
		})

		t.Run("non-empty element unchanged", func(t *testing.T) {
			t.Parallel()

			doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?><root><p>text</p></root>`))
			require.NoError(t, err)

			var buf strings.Builder
			require.NoError(t, helium.NewWriter().SelfCloseEmptyElements(false).WriteTo(&buf, doc))
			str := buf.String()

			expected := "<?xml version=\"1.0\"?>\n<root><p>text</p></root>\n"
			require.Equal(t, expected, str)
		})

		t.Run("empty element with attributes", func(t *testing.T) {
			t.Parallel()

			doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?><root><img src="a.png"/></root>`))
			require.NoError(t, err)

			var buf strings.Builder
			require.NoError(t, helium.NewWriter().SelfCloseEmptyElements(false).WriteTo(&buf, doc))
			str := buf.String()

			expected := "<?xml version=\"1.0\"?>\n<root><img src=\"a.png\"></img></root>\n"
			require.Equal(t, expected, str)
		})

		t.Run("combined with format", func(t *testing.T) {
			t.Parallel()

			doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?><root><a/><b/></root>`))
			require.NoError(t, err)

			var buf strings.Builder
			require.NoError(t, helium.NewWriter().SelfCloseEmptyElements(false).Format(true).WriteTo(&buf, doc))
			str := buf.String()

			expected := "<?xml version=\"1.0\"?>\n<root>\n  <a></a>\n  <b></b>\n</root>\n"
			require.Equal(t, expected, str)
		})

		t.Run("without NoEmpty stays self-closing", func(t *testing.T) {
			t.Parallel()

			doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?><root><br/></root>`))
			require.NoError(t, err)

			str, err := helium.WriteString(doc)
			require.NoError(t, err)

			expected := "<?xml version=\"1.0\"?>\n<root><br/></root>\n"
			require.Equal(t, expected, str)
		})
	})

	t.Run("output unchanged", func(t *testing.T) {
		// The success path must remain byte-for-byte identical after routing all
		// writes through the sticky-error session helpers.
		doc, err := helium.NewParser().Parse(t.Context(), []byte(docWithEverything))
		require.NoError(t, err)

		str, err := helium.WriteString(doc)
		require.NoError(t, err)

		expected := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE root [
<!ELEMENT root (child)*>
<!ATTLIST root id CDATA #IMPLIED>
<!ENTITY greeting "hello">
<!NOTATION gif SYSTEM "image/gif" >
]>
<!--a top level comment-->
<?app instruction?>
<root id="r1"><child>&greeting; world</child><child><![CDATA[<raw> & data]]></child></root>
`
		require.Equal(t, expected, str)
	})

	t.Run("attribute quoting", func(t *testing.T) {
		tests := []struct {
			name     string
			inputXML string
			expected string
		}{
			{
				name: "doctype system id without quotes",
				inputXML: `<?xml version="1.0"?>
<!DOCTYPE root SYSTEM "hello">
<root/>`,
				expected: `<!DOCTYPE root SYSTEM "hello">`,
			},
			{
				name: "doctype system id with only single quotes",
				inputXML: `<?xml version="1.0"?>
<!DOCTYPE root SYSTEM "it's">
<root/>`,
				expected: `<!DOCTYPE root SYSTEM "it's">`,
			},
			{
				name: "internal entity with both quote kinds",
				inputXML: `<?xml version="1.0"?>
<!DOCTYPE root [
<!ENTITY e "it's a &quot;test&quot;">
]>
<root/>`,
				expected: `<!ENTITY e "it's a &quot;test&quot;">`,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				doc, err := helium.NewParser().Parse(t.Context(), []byte(tt.inputXML))
				require.NoError(t, err)

				got, err := helium.WriteString(doc)
				require.NoError(t, err)
				require.Contains(t, got, tt.expected)
			})
		}
	})
}

func TestWriteErrors(t *testing.T) {
	t.Parallel()

	t.Run("nil node", func(t *testing.T) {
		t.Run("WriteString interface nil", func(t *testing.T) {
			t.Parallel()
			s, err := helium.WriteString(nil)
			require.ErrorIs(t, err, helium.ErrNilNode, "nil node must return ErrNilNode")
			require.Empty(t, s)
		})

		t.Run("Write interface nil", func(t *testing.T) {
			t.Parallel()
			err := helium.Write(io.Discard, nil)
			require.ErrorIs(t, err, helium.ErrNilNode, "nil node must return ErrNilNode")
		})

		t.Run("WriteTo interface nil", func(t *testing.T) {
			t.Parallel()
			err := helium.NewWriter().WriteTo(io.Discard, nil)
			require.ErrorIs(t, err, helium.ErrNilNode, "nil node must return ErrNilNode")
		})

		t.Run("WriteTo typed nil", func(t *testing.T) {
			t.Parallel()
			var typedNil *helium.Element
			err := helium.NewWriter().WriteTo(io.Discard, typedNil)
			require.ErrorIs(t, err, helium.ErrNilNode, "typed-nil node must return ErrNilNode")
		})

		t.Run("WriteTo typed nil document", func(t *testing.T) {
			t.Parallel()
			var typedNil *helium.Document
			err := helium.NewWriter().WriteTo(io.Discard, typedNil)
			require.ErrorIs(t, err, helium.ErrNilNode, "typed-nil document must return ErrNilNode")
		})
	})

	t.Run("write error propagates", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(docWithEverything))
		require.NoError(t, err)

		// Determine the full serialized length so we can fail at every prefix.
		full, err := helium.WriteString(doc)
		require.NoError(t, err)
		require.NotEmpty(t, full)

		// Failing immediately must surface a non-nil error (previously nil).
		require.ErrorIs(t, helium.Write(&failAfterNWriter{limit: 0}, doc), errShortWrite,
			"serialization must report the writer error")

		// Failing at any intermediate offset must also surface a non-nil error.
		for limit := 1; limit < len(full); limit += 7 {
			err := helium.Write(&failAfterNWriter{limit: limit}, doc)
			require.Errorf(t, err, "write that fails after %d bytes must return an error", limit)
		}
	})

	// malformed nodes are rejected
	// during the discard pass, before WriteTo calls the caller's writer.
	t.Run("validation precedes output", func(t *testing.T) {
		for _, tc := range []struct {
			name    string
			sibling func(*helium.Document) helium.Node
			wantErr error
		}{
			{
				name:    "comment",
				sibling: func(d *helium.Document) helium.Node { return d.CreateComment([]byte("a--b")) },
				wantErr: helium.ErrWriterInvalidComment,
			},
			{
				name:    "pi",
				sibling: func(d *helium.Document) helium.Node { return d.CreatePI("t", "a?>b") },
				wantErr: helium.ErrWriterInvalidPIContent,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				doc := helium.NewDefaultDocument()
				root, err := doc.CreateElement("r")
				require.NoError(t, err)
				require.NoError(t, doc.SetDocumentElement(root))
				// A malformed top-level sibling follows the root element. The target
				// writer would reject the separator between them if it were called.
				require.NoError(t, doc.AddChild(tc.sibling(doc)))

				out := &failOnSubstringWriter{trigger: "\n"}
				err = helium.NewWriter().XMLDeclaration(false).WriteTo(out, doc)
				require.ErrorIs(t, err, tc.wantErr)
				require.Zero(t, out.writes, "validation must not call the target writer")
			})
		}
	})

	t.Run("validation errors leave no output", func(t *testing.T) {
		for _, tc := range []struct {
			name    string
			build   func(*testing.T) *helium.Document
			wantErr error
		}{
			{
				name: "empty DTD name",
				build: func(t *testing.T) *helium.Document {
					doc := helium.NewDefaultDocument()
					_, err := doc.CreateInternalSubset("", "", "")
					require.NoError(t, err)
					root, err := doc.CreateElement("root")
					require.NoError(t, err)
					require.NoError(t, doc.SetDocumentElement(root))
					return doc
				},
				wantErr: helium.ErrWriterInvalidDTDNode,
			},
			{
				name: "invalid text character",
				build: func(t *testing.T) *helium.Document {
					doc := helium.NewDefaultDocument()
					root, err := doc.CreateElement("root")
					require.NoError(t, err)
					require.NoError(t, doc.SetDocumentElement(root))
					require.NoError(t, root.AppendText([]byte("bad\x01text")))
					return doc
				},
				wantErr: helium.ErrInvalidXMLChar,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				var out bytes.Buffer
				err := helium.NewWriter().WriteTo(&out, tc.build(t))
				require.ErrorIs(t, err, tc.wantErr)
				require.Empty(t, out.String(), "validation errors must leave no output bytes")
			})
		}
	})

	// each structural-serialization
	// failure the writer detects is reported via a named sentinel a caller can match
	// with errors.Is, and never as an anonymous string-literal error.
	t.Run("structural errors are matchable", func(t *testing.T) {
		for _, tc := range []struct {
			name     string
			build    func(t *testing.T) *helium.Document
			sentinel error
		}{
			{
				name: "invalid element name",
				build: func(t *testing.T) *helium.Document {
					doc := helium.NewDefaultDocument()
					root, err := doc.CreateElement(`root injected="1"`)
					require.NoError(t, err)
					require.NoError(t, doc.SetDocumentElement(root))
					return doc
				},
				sentinel: helium.ErrWriterInvalidElementName,
			},
			{
				name: "reserved element name",
				build: func(t *testing.T) *helium.Document {
					doc := helium.NewDefaultDocument()
					// CreateElement rejects a colon, so build the reserved
					// "xmlns:root" name through an active namespace whose prefix
					// is the reserved "xmlns" — the element's Name() is then
					// "xmlns:root", which the writer must reject.
					ns, err := doc.CreateNamespace("xmlns", "urn:reserved")
					require.NoError(t, err)
					root, err := doc.CreateElementNS("root", ns)
					require.NoError(t, err)
					require.NoError(t, doc.SetDocumentElement(root))
					return doc
				},
				sentinel: helium.ErrWriterReservedElementName,
			},
			{
				name: "invalid attribute name",
				build: func(t *testing.T) *helium.Document {
					doc := helium.NewDefaultDocument()
					root, err := doc.CreateElement("root")
					require.NoError(t, err)
					require.NoError(t, doc.SetDocumentElement(root))
					err = root.SetAttribute(`x onmouseover`, "1")
					require.NoError(t, err)
					return doc
				},
				sentinel: helium.ErrWriterInvalidAttributeName,
			},
			{
				name: "reserved attribute name",
				build: func(t *testing.T) *helium.Document {
					doc := helium.NewDefaultDocument()
					root, err := doc.CreateElement("root")
					require.NoError(t, err)
					require.NoError(t, doc.SetDocumentElement(root))
					err = root.SetAttribute("xmlns", "urn:evil")
					require.NoError(t, err)
					return doc
				},
				sentinel: helium.ErrWriterReservedAttributeName,
			},
			{
				name: "reserved namespace prefix",
				build: func(t *testing.T) *helium.Document {
					doc := helium.NewDefaultDocument()
					root, err := doc.CreateElement("root")
					require.NoError(t, err)
					require.NoError(t, doc.SetDocumentElement(root))
					require.NoError(t, root.DeclareNamespace("xmlns", "urn:x"))
					return doc
				},
				sentinel: helium.ErrWriterReservedNamespacePrefix,
			},
			{
				name: "invalid namespace prefix",
				build: func(t *testing.T) *helium.Document {
					doc := helium.NewDefaultDocument()
					root, err := doc.CreateElement("root")
					require.NoError(t, err)
					require.NoError(t, doc.SetDocumentElement(root))
					require.NoError(t, root.DeclareNamespace("bad prefix", "urn:x"))
					return doc
				},
				sentinel: helium.ErrWriterInvalidNamespacePrefix,
			},
			{
				name: "unbound element namespace prefix",
				build: func(t *testing.T) *helium.Document {
					doc := helium.NewDefaultDocument()
					// A non-empty prefix bound to an empty URI: the element name
					// serializes as "foo:root" with no xmlns:foo, which the parser
					// cannot reparse.
					ns, err := doc.CreateNamespace("foo", "")
					require.NoError(t, err)
					root, err := doc.CreateElementNS("root", ns)
					require.NoError(t, err)
					require.NoError(t, doc.SetDocumentElement(root))
					return doc
				},
				sentinel: helium.ErrWriterUnboundNamespacePrefix,
			},
			{
				name: "unbound attribute namespace prefix",
				build: func(t *testing.T) *helium.Document {
					doc := helium.NewDefaultDocument()
					root, err := doc.CreateElement("root")
					require.NoError(t, err)
					require.NoError(t, doc.SetDocumentElement(root))
					ns, err := doc.CreateNamespace("foo", "")
					require.NoError(t, err)
					require.NoError(t, root.SetAttributeNS("bar", "v", ns))
					return doc
				},
				sentinel: helium.ErrWriterUnboundNamespacePrefix,
			},
			{
				name: "invalid comment content",
				build: func(t *testing.T) *helium.Document {
					doc := helium.NewDefaultDocument()
					root, err := doc.CreateElement("root")
					require.NoError(t, err)
					require.NoError(t, doc.SetDocumentElement(root))
					require.NoError(t, root.AddChild(doc.CreateComment([]byte("a--b"))))
					return doc
				},
				sentinel: helium.ErrWriterInvalidComment,
			},
			{
				name: "invalid PI target",
				build: func(t *testing.T) *helium.Document {
					doc := helium.NewDefaultDocument()
					root, err := doc.CreateElement("root")
					require.NoError(t, err)
					require.NoError(t, doc.SetDocumentElement(root))
					require.NoError(t, root.AddChild(doc.CreatePI("1bad", "")))
					return doc
				},
				sentinel: helium.ErrWriterInvalidPITarget,
			},
			{
				name: "invalid PI content",
				build: func(t *testing.T) *helium.Document {
					doc := helium.NewDefaultDocument()
					root, err := doc.CreateElement("root")
					require.NoError(t, err)
					require.NoError(t, doc.SetDocumentElement(root))
					require.NoError(t, root.AddChild(doc.CreatePI("t", "a?>b")))
					return doc
				},
				sentinel: helium.ErrWriterInvalidPIContent,
			},
			{
				name: "empty DOCTYPE name",
				build: func(t *testing.T) *helium.Document {
					doc := helium.NewDefaultDocument()
					_, err := doc.CreateInternalSubset("   ", "", "")
					require.NoError(t, err)
					root, err := doc.CreateElement("root")
					require.NoError(t, err)
					require.NoError(t, doc.SetDocumentElement(root))
					return doc
				},
				sentinel: helium.ErrWriterInvalidDTDNode,
			},
			{
				name: "DOCTYPE system literal with both quotes",
				build: func(t *testing.T) *helium.Document {
					doc := helium.NewDefaultDocument()
					_, err := doc.CreateInternalSubset("root", "", `a"b'c.dtd`)
					require.NoError(t, err)
					root, err := doc.CreateElement("root")
					require.NoError(t, err)
					require.NoError(t, doc.SetDocumentElement(root))
					return doc
				},
				sentinel: helium.ErrWriterInvalidDTDNode,
			},
			{
				name: "DOCTYPE public id with invalid PubidChar",
				build: func(t *testing.T) *helium.Document {
					doc := helium.NewDefaultDocument()
					_, err := doc.CreateInternalSubset("root", "bad{pubid", "sys.dtd")
					require.NoError(t, err)
					root, err := doc.CreateElement("root")
					require.NoError(t, err)
					require.NoError(t, doc.SetDocumentElement(root))
					return doc
				},
				sentinel: helium.ErrWriterInvalidDTDNode,
			},
			{
				name: "notation public id with invalid PubidChar",
				build: func(t *testing.T) *helium.Document {
					doc := helium.NewDefaultDocument()
					dtd, err := doc.CreateInternalSubset("root", "", "")
					require.NoError(t, err)
					_, err = dtd.AddNotation("n", "bad{pubid", "n.exe")
					require.NoError(t, err)
					root, err := doc.CreateElement("root")
					require.NoError(t, err)
					require.NoError(t, doc.SetDocumentElement(root))
					return doc
				},
				sentinel: helium.ErrWriterInvalidDTDNode,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				doc := tc.build(t)
				var buf strings.Builder
				err := helium.Write(&buf, doc)
				require.Error(t, err, "malformed node must be rejected")
				require.ErrorIs(t, err, tc.sentinel, "error must match the named sentinel")
			})
		}
	})
}

// the Writer option toggles and serialization paths.
func TestWriterOptions(t *testing.T) {
	t.Parallel()
	in, err := os.ReadFile("test/att12.xml")
	require.NoError(t, err)
	doc, err := helium.NewParser().Parse(t.Context(), in)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = helium.NewWriter().
		IncludeDTD(false).
		AllowPrefixUndeclarations(true).
		WriteTo(&buf, doc)
	require.NoError(t, err)
	// With the DTD excluded, the DOCTYPE must not appear.
	require.NotContains(t, buf.String(), "<!DOCTYPE")

	buf.Reset()
	err = helium.NewWriter().IncludeDTD(true).WriteTo(&buf, doc)
	require.NoError(t, err)
	require.Contains(t, buf.String(), "<!DOCTYPE")

	// EscapeNonASCII path with a non-ASCII text node.
	d2 := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	r, err := d2.CreateElement("r")
	require.NoError(t, err)
	require.NoError(t, d2.AddChild(r))
	require.NoError(t, r.AppendText([]byte("café")))

	buf.Reset()
	err = helium.NewWriter().EscapeNonASCII(true).WriteTo(&buf, d2)
	require.NoError(t, err)
	require.Contains(t, buf.String(), "&#")
}
