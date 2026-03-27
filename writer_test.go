package helium_test

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/stretchr/testify/require"
)

func TestXMLToDOMToXMLString(t *testing.T) {
	skipped := map[string]struct{}{}
	only := map[string]struct{}{}
	if v := os.Getenv("HELIUM_DUMP_TEST_FILES"); v != "" {
		files := strings.Split(v, ",")
		for _, f := range files {
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

		doc, err := helium.NewParser().Parse(t.Context(), []byte(in))
		require.NoError(t, err, `Parse(...) succeeds`)

		str, err := doc.XMLString()
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
}

func TestDOMToXMLString(t *testing.T) {
	doc := helium.NewDefaultDocument()
	//	defer doc.Free()

	root := doc.CreateElement("root")

	require.NoError(t, doc.SetDocumentElement(root))
	require.NoError(t, root.AppendText([]byte(`Hello, World!`)))

	str, err := doc.XMLString()
	require.NoError(t, err, "XMLString(doc) succeeds")

	t.Logf("%s", str)
}

// BenchmarkWriteNonASCII serializes a document containing many non-ASCII
// characters with EscapeNonASCII enabled, exercising the hex char ref path.
func BenchmarkWriteNonASCII(b *testing.B) {
	var buf strings.Builder
	buf.WriteString("<root>")
	for i := 0; i < 200; i++ {
		buf.WriteString("<t>caf\u00e9 na\u00efve r\u00e9sum\u00e9 \u00fcber \u00e0 \u00e7a \u00f1</t>")
	}
	buf.WriteString("</root>")

	doc, err := helium.NewParser().Parse(b.Context(), []byte(buf.String()))
	require.NoError(b, err)

	w := helium.NewWriter().EscapeNonASCII(true)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := w.WriteDoc(io.Discard, doc)
		require.NoError(b, err)
	}
}

func TestDumpNsSkipsXmlPrefix(t *testing.T) {
	doc := helium.NewDefaultDocument()
	root := doc.CreateElement("root")
	require.NoError(t, doc.SetDocumentElement(root))

	// Add explicit xml: namespace declaration to the element
	require.NoError(t, root.DeclareNamespace("xml", lexicon.NamespaceXML))

	str, err := doc.XMLString()
	require.NoError(t, err)

	// The xml: namespace declaration must NOT appear in the output.
	// libxml2's xmlNsDumpOutput skips prefix "xml" unconditionally.
	require.NotContains(t, str, "xmlns:xml")
}

func TestDumpNsPropagatesWriteError(t *testing.T) {
	doc := helium.NewDefaultDocument()
	root := doc.CreateElement("root")
	require.NoError(t, doc.SetDocumentElement(root))
	require.NoError(t, root.DeclareNamespace("p", "urn:test"))

	writer := helium.NewWriter().XMLDeclaration(false)
	err := writer.WriteDoc(&namespaceFailWriter{failOn: "xmlns"}, doc)
	require.ErrorIs(t, err, errNamespaceWrite)
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

func TestFormatOutput(t *testing.T) {
	t.Run("nested elements", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?><root><child><grandchild/></child></root>`))
		require.NoError(t, err)

		str, err := doc.XMLString(helium.NewWriter().Format(true))
		require.NoError(t, err)

		expected := "<?xml version=\"1.0\"?>\n<root>\n  <child>\n    <grandchild/>\n  </child>\n</root>\n"
		require.Equal(t, expected, str)
	})

	t.Run("text-only element stays inline", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?><root><child>hello</child></root>`))
		require.NoError(t, err)

		str, err := doc.XMLString(helium.NewWriter().Format(true))
		require.NoError(t, err)

		expected := "<?xml version=\"1.0\"?>\n<root>\n  <child>hello</child>\n</root>\n"
		require.Equal(t, expected, str)
	})

	t.Run("custom indent string", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?><root><child><grandchild/></child></root>`))
		require.NoError(t, err)

		str, err := doc.XMLString(helium.NewWriter().Format(true).IndentString("\t"))
		require.NoError(t, err)

		expected := "<?xml version=\"1.0\"?>\n<root>\n\t<child>\n\t\t<grandchild/>\n\t</child>\n</root>\n"
		require.Equal(t, expected, str)
	})

	t.Run("without format stays compact", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?><root><child><grandchild/></child></root>`))
		require.NoError(t, err)

		str, err := doc.XMLString()
		require.NoError(t, err)

		expected := "<?xml version=\"1.0\"?>\n<root><child><grandchild/></child></root>\n"
		require.Equal(t, expected, str)
	})

	t.Run("multiple children", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?><root><a/><b/><c/></root>`))
		require.NoError(t, err)

		str, err := doc.XMLString(helium.NewWriter().Format(true))
		require.NoError(t, err)

		expected := "<?xml version=\"1.0\"?>\n<root>\n  <a/>\n  <b/>\n  <c/>\n</root>\n"
		require.Equal(t, expected, str)
	})

	t.Run("element XMLString with format", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?><root><child><grandchild/></child></root>`))
		require.NoError(t, err)

		root := doc.DocumentElement()
		require.NotNil(t, root)

		str, err := root.XMLString(helium.NewWriter().Format(true))
		require.NoError(t, err)

		expected := "<root>\n  <child>\n    <grandchild/>\n  </child>\n</root>"
		require.Equal(t, expected, str)
	})

	t.Run("comment and PI children", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?><root><!--comment--><child/><?pi data?></root>`))
		require.NoError(t, err)

		str, err := doc.XMLString(helium.NewWriter().Format(true))
		require.NoError(t, err)

		expected := "<?xml version=\"1.0\"?>\n<root>\n  <!--comment-->\n  <child/>\n  <?pi data?>\n</root>\n"
		require.Equal(t, expected, str)
	})

	t.Run("deeply nested", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?><a><b><c><d>text</d></c></b></a>`))
		require.NoError(t, err)

		str, err := doc.XMLString(helium.NewWriter().Format(true))
		require.NoError(t, err)

		expected := "<?xml version=\"1.0\"?>\n<a>\n  <b>\n    <c>\n      <d>text</d>\n    </c>\n  </b>\n</a>\n"
		require.Equal(t, expected, str)
	})
}

func TestXHTMLVoidElementDefaultNS(t *testing.T) {
	// XHTML void elements in the default namespace (prefix == "") should
	// use self-closing " />" syntax, matching libxml2's check:
	//   (cur->ns == NULL) || (cur->ns->prefix == NULL)
	input := `<?xml version="1.0"?>
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Strict//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd">
<html xmlns="http://www.w3.org/1999/xhtml"><head><title>T</title></head><body><br/></body></html>`
	doc, err := helium.NewParser().Parse(t.Context(), []byte(input))
	require.NoError(t, err)

	str, err := doc.XMLString()
	require.NoError(t, err)

	// <br> must be serialized as "<br />" (self-closing), not "<br></br>"
	require.Contains(t, str, "<br />")
	require.NotContains(t, str, "<br></br>")
}

func TestXHTMLFormat(t *testing.T) {
	t.Run("element children get indented", func(t *testing.T) {
		input := `<?xml version="1.0"?>
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Strict//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd">
<html xmlns="http://www.w3.org/1999/xhtml"><body><p>hello</p><p>world</p></body></html>`
		doc, err := helium.NewParser().Parse(t.Context(), []byte(input))
		require.NoError(t, err)

		str, err := doc.XMLString(helium.NewWriter().Format(true))
		require.NoError(t, err)

		// <body> has element children → they should be indented
		require.Contains(t, str, "<body>\n    <p>")
		require.Contains(t, str, "</p>\n  </body>")
	})

	t.Run("text-only elements stay inline", func(t *testing.T) {
		input := `<?xml version="1.0"?>
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Strict//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd">
<html xmlns="http://www.w3.org/1999/xhtml"><body><p>hello</p></body></html>`
		doc, err := helium.NewParser().Parse(t.Context(), []byte(input))
		require.NoError(t, err)

		str, err := doc.XMLString(helium.NewWriter().Format(true))
		require.NoError(t, err)

		// <p> has only text → no indentation inside
		require.Contains(t, str, "<p>hello</p>")
	})
}

func TestNoEmpty(t *testing.T) {
	t.Run("empty element uses open+close tags", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?><root><br/></root>`))
		require.NoError(t, err)

		str, err := doc.XMLString(helium.NewWriter().SelfCloseEmptyElements(false))
		require.NoError(t, err)

		expected := "<?xml version=\"1.0\"?>\n<root><br></br></root>\n"
		require.Equal(t, expected, str)
	})

	t.Run("non-empty element unchanged", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?><root><p>text</p></root>`))
		require.NoError(t, err)

		str, err := doc.XMLString(helium.NewWriter().SelfCloseEmptyElements(false))
		require.NoError(t, err)

		expected := "<?xml version=\"1.0\"?>\n<root><p>text</p></root>\n"
		require.Equal(t, expected, str)
	})

	t.Run("empty element with attributes", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?><root><img src="a.png"/></root>`))
		require.NoError(t, err)

		str, err := doc.XMLString(helium.NewWriter().SelfCloseEmptyElements(false))
		require.NoError(t, err)

		expected := "<?xml version=\"1.0\"?>\n<root><img src=\"a.png\"></img></root>\n"
		require.Equal(t, expected, str)
	})

	t.Run("combined with format", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?><root><a/><b/></root>`))
		require.NoError(t, err)

		str, err := doc.XMLString(helium.NewWriter().SelfCloseEmptyElements(false).Format(true))
		require.NoError(t, err)

		expected := "<?xml version=\"1.0\"?>\n<root>\n  <a></a>\n  <b></b>\n</root>\n"
		require.Equal(t, expected, str)
	})

	t.Run("without NoEmpty stays self-closing", func(t *testing.T) {
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?><root><br/></root>`))
		require.NoError(t, err)

		str, err := doc.XMLString()
		require.NoError(t, err)

		expected := "<?xml version=\"1.0\"?>\n<root><br/></root>\n"
		require.Equal(t, expected, str)
	})
}

func TestDumpQuotingViaPublicAPI(t *testing.T) {
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

			got, err := doc.XMLString()
			require.NoError(t, err)
			require.Contains(t, got, tt.expected)
		})
	}
}
