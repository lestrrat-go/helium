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
	"github.com/lestrrat-go/helium/enum"
	"github.com/lestrrat-go/helium/internal/lexicon"
	"github.com/stretchr/testify/require"
)

func TestXMLToDOMToXMLString(t *testing.T) {
	t.Parallel()

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
}

func TestDOMToXMLString(t *testing.T) {
	t.Parallel()

	doc := helium.NewDefaultDocument()
	//	defer doc.Free()

	root := doc.CreateElement("root")

	require.NoError(t, doc.SetDocumentElement(root))
	require.NoError(t, root.AppendText([]byte(`Hello, World!`)))

	str, err := helium.WriteString(doc)
	require.NoError(t, err, "XMLString(doc) succeeds")

	t.Logf("%s", str)
}

func TestWriteRejectsInjectedNames(t *testing.T) {
	t.Parallel()

	t.Run("element name injection", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		root := doc.CreateElement(`root injected="1"`)
		require.NoError(t, doc.SetDocumentElement(root))

		_, err := helium.WriteString(doc)
		require.Error(t, err, "injected element name must not serialize")
	})

	t.Run("attribute name injection", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		root := doc.CreateElement("root")
		require.NoError(t, doc.SetDocumentElement(root))
		// SetAttribute only rejects colons, so a space-bearing name slips
		// through and would inject a second attribute on serialization.
		_, err := root.SetAttribute(`x onmouseover`, "1")
		require.NoError(t, err)

		_, err = helium.WriteString(doc)
		require.Error(t, err, "injected attribute name must not serialize")
	})

	t.Run("reserved xmlns attribute name rejected", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		root := doc.CreateElement("root")
		require.NoError(t, doc.SetDocumentElement(root))
		// "xmlns" is a valid NCName, but a normal attribute named "xmlns"
		// would be emitted as a namespace declaration that never went through
		// DeclareNamespace.
		_, err := root.SetAttribute("xmlns", "urn:evil")
		require.NoError(t, err)

		_, err = helium.WriteString(doc)
		require.Error(t, err, "reserved xmlns attribute name must not serialize")
	})

	t.Run("reserved xmlns-prefixed attribute name rejected", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		root := doc.CreateElement("root")
		require.NoError(t, doc.SetDocumentElement(root))
		// An attribute whose QName prefix is "xmlns" (e.g. "xmlns:foo") is a
		// namespace declaration and must not be emitted as a normal attribute.
		ns, err := doc.CreateNamespace("xmlns", "urn:x")
		require.NoError(t, err)
		_, err = root.SetAttributeNS("foo", "v", ns)
		require.NoError(t, err)

		_, err = helium.WriteString(doc)
		require.Error(t, err, "reserved xmlns-prefixed attribute name must not serialize")
	})

	t.Run("valid element name serializes", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		root := doc.CreateElement("root")
		require.NoError(t, doc.SetDocumentElement(root))

		str, err := helium.WriteString(doc)
		require.NoError(t, err)
		require.Contains(t, str, "<root/>")
	})

	t.Run("valid namespaced name serializes", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		root := doc.CreateElement("root")
		require.NoError(t, doc.SetDocumentElement(root))
		require.NoError(t, root.SetActiveNamespace("p", "urn:example"))

		str, err := helium.WriteString(doc)
		require.NoError(t, err)
		require.Contains(t, str, "<p:root")
	})
}

func TestWriteActiveDefaultNamespace(t *testing.T) {
	t.Parallel()

	t.Run("active default namespace emits xmlns", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		root := doc.CreateElement("root")
		require.NoError(t, doc.SetDocumentElement(root))
		require.NoError(t, root.SetActiveNamespace("", "urn:x"))

		str, err := helium.WriteString(doc)
		require.NoError(t, err)
		require.Contains(t, str, `xmlns="urn:x"`)
	})

	t.Run("active prefixed namespace still emits xmlns:p", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		root := doc.CreateElement("root")
		require.NoError(t, doc.SetDocumentElement(root))
		require.NoError(t, root.SetActiveNamespace("p", "urn:x"))

		str, err := helium.WriteString(doc)
		require.NoError(t, err)
		require.Contains(t, str, `xmlns:p="urn:x"`)
	})

	t.Run("parsed default namespace declared exactly once", func(t *testing.T) {
		t.Parallel()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(`<x xmlns="urn:x"/>`))
		require.NoError(t, err)

		str, err := helium.WriteString(doc)
		require.NoError(t, err)
		require.Equal(t, 1, strings.Count(str, `xmlns="urn:x"`))
	})

	t.Run("unprefixed attribute gains no spurious xmlns", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		root := doc.CreateElement("root")
		require.NoError(t, doc.SetDocumentElement(root))
		require.NoError(t, root.SetActiveNamespace("p", "urn:x"))
		_, err := root.SetAttribute("id", "1")
		require.NoError(t, err)

		str, err := helium.WriteString(doc)
		require.NoError(t, err)
		require.NotContains(t, str, `xmlns=""`)
	})

	t.Run("conflicting declared and active default emits a single reparseable xmlns", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		root := doc.CreateElement("root")
		require.NoError(t, doc.SetDocumentElement(root))
		// A declared default that conflicts with the element's active default: the
		// active binding wins and only one xmlns is emitted, so the output reparses.
		require.NoError(t, root.DeclareNamespace("", "urn:declared"))
		require.NoError(t, root.SetActiveNamespace("", "urn:active"))

		str, err := helium.WriteString(doc)
		require.NoError(t, err)
		require.Equal(t, 1, strings.Count(str, "xmlns="), "exactly one default declaration: %s", str)
		require.Contains(t, str, `xmlns="urn:active"`)
		require.NotContains(t, str, `xmlns="urn:declared"`)

		_, err = helium.NewParser().Parse(t.Context(), []byte(str))
		require.NoError(t, err, "serialized output must reparse: %s", str)
	})
}

func TestWriteInheritedNamespaces(t *testing.T) {
	t.Parallel()

	t.Run("seeded prefix is not re-declared on a using element", func(t *testing.T) {
		t.Parallel()
		// A fragment whose prefix is bound only on an ancestor outside the output:
		// seeding that binding suppresses the otherwise-synthesized re-declaration.
		doc, err := helium.NewParser().Parse(t.Context(),
			[]byte(`<root xmlns:p="urn:p"><child><p:leaf/></child></root>`))
		require.NoError(t, err)
		root := doc.DocumentElement()
		require.NotNil(t, root)
		child := root.FirstChild()
		require.NotNil(t, child)

		var b bytes.Buffer
		w := helium.NewWriter().XMLDeclaration(false).
			InheritedNamespaces(map[string]string{"p": "urn:p"})
		require.NoError(t, w.WriteTo(&b, child))
		require.Equal(t, `<child><p:leaf/></child>`, b.String())
	})

	t.Run("without seeding the inherited prefix is re-declared", func(t *testing.T) {
		t.Parallel()
		doc, err := helium.NewParser().Parse(t.Context(),
			[]byte(`<root xmlns:p="urn:p"><child><p:leaf/></child></root>`))
		require.NoError(t, err)
		child := doc.DocumentElement().FirstChild()
		require.NotNil(t, child)

		var b bytes.Buffer
		w := helium.NewWriter().XMLDeclaration(false)
		require.NoError(t, w.WriteTo(&b, child))
		require.Contains(t, b.String(), `xmlns:p="urn:p"`)
	})
}

func TestXHTMLWriteRejectsInjectedNames(t *testing.T) {
	t.Parallel()

	// newXHTMLDoc builds a document whose internal subset is an XHTML DTD, so
	// serialization routes through dumpXHTMLNode / dumpXHTMLAttrList rather than
	// the generic writeNode path.
	newXHTMLDoc := func(t *testing.T) *helium.Document {
		t.Helper()
		doc := helium.NewDefaultDocument()
		_, err := doc.CreateInternalSubset(
			"html",
			"-//W3C//DTD XHTML 1.0 Strict//EN",
			"http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd",
		)
		require.NoError(t, err)
		return doc
	}

	t.Run("element name injection", func(t *testing.T) {
		t.Parallel()
		doc := newXHTMLDoc(t)
		root := doc.CreateElement(`html injected="1"`)
		require.NoError(t, doc.SetDocumentElement(root))

		_, err := helium.WriteString(doc)
		require.Error(t, err, "injected XHTML element name must not serialize")
	})

	t.Run("attribute name injection", func(t *testing.T) {
		t.Parallel()
		doc := newXHTMLDoc(t)
		root := doc.CreateElement("html")
		require.NoError(t, doc.SetDocumentElement(root))
		_, err := root.SetAttribute(`x onmouseover`, "1")
		require.NoError(t, err)

		_, err = helium.WriteString(doc)
		require.Error(t, err, "injected XHTML attribute name must not serialize")
	})

	t.Run("valid element name serializes", func(t *testing.T) {
		t.Parallel()
		doc := newXHTMLDoc(t)
		root := doc.CreateElement("html")
		require.NoError(t, doc.SetDocumentElement(root))

		str, err := helium.WriteString(doc)
		require.NoError(t, err)
		require.Contains(t, str, "<html")
	})

	t.Run("valid namespaced name serializes", func(t *testing.T) {
		t.Parallel()
		doc := newXHTMLDoc(t)
		root := doc.CreateElement("html")
		require.NoError(t, doc.SetDocumentElement(root))
		require.NoError(t, root.SetActiveNamespace("p", "urn:example"))

		str, err := helium.WriteString(doc)
		require.NoError(t, err)
		require.Contains(t, str, "<p:html")
	})
}

func TestWriteRejectsInjectedNamespacePrefix(t *testing.T) {
	t.Parallel()

	t.Run("namespace prefix injection", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		root := doc.CreateElement("root")
		require.NoError(t, doc.SetDocumentElement(root))
		// DeclareNamespace does not validate the prefix, so a crafted prefix
		// would inject raw markup into the start tag on serialization.
		require.NoError(t, root.DeclareNamespace(`p injected="1`, "urn"))

		_, err := helium.WriteString(doc)
		require.Error(t, err, "injected namespace prefix must not serialize")
	})

	t.Run("valid prefix serializes", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		root := doc.CreateElement("root")
		require.NoError(t, doc.SetDocumentElement(root))
		require.NoError(t, root.DeclareNamespace("p", "urn:example"))

		str, err := helium.WriteString(doc)
		require.NoError(t, err)
		require.Contains(t, str, `xmlns:p="urn:example"`)
	})

	t.Run("default namespace serializes", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		root := doc.CreateElement("root")
		require.NoError(t, doc.SetDocumentElement(root))
		require.NoError(t, root.DeclareNamespace("", "urn:default"))

		str, err := helium.WriteString(doc)
		require.NoError(t, err)
		require.Contains(t, str, `xmlns="urn:default"`)
	})

	t.Run("reserved xml prefix serializes", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		root := doc.CreateElement("root")
		require.NoError(t, doc.SetDocumentElement(root))
		require.NoError(t, root.DeclareNamespace("xml", lexicon.NamespaceXML))

		_, err := helium.WriteString(doc)
		require.NoError(t, err, "reserved xml prefix must still serialize")
	})

	t.Run("reserved xmlns prefix rejected", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		root := doc.CreateElement("root")
		require.NoError(t, doc.SetDocumentElement(root))
		// Namespaces-in-XML forbids declaring the xmlns prefix; the serializer
		// must not emit xmlns:xmlns="...".
		require.NoError(t, root.DeclareNamespace("xmlns", "urn"))

		_, err := helium.WriteString(doc)
		require.Error(t, err, "reserved xmlns prefix must not serialize")
	})
}

func TestXHTMLWriteRejectsInjectedNamespacePrefix(t *testing.T) {
	t.Parallel()

	newXHTMLDoc := func(t *testing.T) *helium.Document {
		t.Helper()
		doc := helium.NewDefaultDocument()
		_, err := doc.CreateInternalSubset(
			"html",
			"-//W3C//DTD XHTML 1.0 Strict//EN",
			"http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd",
		)
		require.NoError(t, err)
		return doc
	}

	t.Run("namespace prefix injection", func(t *testing.T) {
		t.Parallel()
		doc := newXHTMLDoc(t)
		root := doc.CreateElement("html")
		require.NoError(t, doc.SetDocumentElement(root))
		require.NoError(t, root.DeclareNamespace(`p injected="1`, "urn"))

		_, err := helium.WriteString(doc)
		require.Error(t, err, "injected XHTML namespace prefix must not serialize")
	})

	t.Run("valid prefix serializes", func(t *testing.T) {
		t.Parallel()
		doc := newXHTMLDoc(t)
		root := doc.CreateElement("html")
		require.NoError(t, doc.SetDocumentElement(root))
		require.NoError(t, root.DeclareNamespace("p", "urn:example"))

		str, err := helium.WriteString(doc)
		require.NoError(t, err)
		require.Contains(t, str, `xmlns:p="urn:example"`)
	})
}

// TestXHTMLAttrErrorEmitsNoPartialChildren reproduces Finding 1: when an XHTML
// element has an invalid attribute name AND non-element child content, the
// serializer must abort at the first error and must NOT emit any of the child
// content before returning the error.
func TestXHTMLAttrErrorEmitsNoPartialChildren(t *testing.T) {
	t.Parallel()

	doc := helium.NewDefaultDocument()
	_, err := doc.CreateInternalSubset(
		"html",
		"-//W3C//DTD XHTML 1.0 Strict//EN",
		"http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd",
	)
	require.NoError(t, err)

	root := doc.CreateElement("html")
	require.NoError(t, doc.SetDocumentElement(root))

	// Invalid attribute name on the element: serialization must fail before any
	// child content is written.
	_, err = root.SetAttribute(`x onmouseover`, "1")
	require.NoError(t, err)

	const childMarker = "SECRET_CHILD_TEXT"
	text := doc.CreateText([]byte(childMarker))
	require.NoError(t, root.AddChild(text))

	var buf strings.Builder
	err = helium.NewWriter().WriteTo(&buf, doc)
	require.Error(t, err, "invalid XHTML attribute name must fail serialization")
	require.NotContains(t, buf.String(), childMarker,
		"no child content must be emitted after an attribute-name error")
}

// TestWriteRejectsXmlnsElementName reproduces Finding 2: an element whose QName
// prefix is the reserved "xmlns" prefix must not serialize, even when an active
// namespace bypasses dumpNs.
func TestWriteRejectsXmlnsElementName(t *testing.T) {
	t.Parallel()

	t.Run("xmlns-prefixed element name rejected", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		root := doc.CreateElement("root")
		require.NoError(t, doc.SetDocumentElement(root))
		// SetActiveNamespace sets the node's active namespace directly, so the
		// "xmlns" prefix is emitted as the element QName prefix (<xmlns:root/>),
		// which Namespaces-in-XML forbids.
		require.NoError(t, root.SetActiveNamespace("xmlns", "urn:evil"))

		_, err := helium.WriteString(doc)
		require.Error(t, err, "xmlns-prefixed element name must not serialize")
	})

	t.Run("valid namespaced element name serializes", func(t *testing.T) {
		t.Parallel()
		doc := helium.NewDefaultDocument()
		root := doc.CreateElement("root")
		require.NoError(t, doc.SetDocumentElement(root))
		require.NoError(t, root.SetActiveNamespace("p", "urn:example"))

		str, err := helium.WriteString(doc)
		require.NoError(t, err)
		require.Contains(t, str, "<p:root")
	})

	t.Run("bare xmlns element name serializes", func(t *testing.T) {
		t.Parallel()
		// "xmlns" is a valid element name: <xmlns>...</xmlns> is well-formed XML.
		// It is reserved only as an attribute name (default-namespace decl), so
		// an element literally named "xmlns" must serialize without error. This
		// is the regression case from xslt3 test si-element-261.
		doc := helium.NewDefaultDocument()
		root := doc.CreateElement("xmlns")
		require.NoError(t, doc.SetDocumentElement(root))

		str, err := helium.WriteString(doc)
		require.NoError(t, err, "bare xmlns element name must serialize")
		require.Contains(t, str, "<xmlns")
	})
}

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

func TestDumpNsSkipsXmlPrefix(t *testing.T) {
	t.Parallel()

	doc := helium.NewDefaultDocument()
	root := doc.CreateElement("root")
	require.NoError(t, doc.SetDocumentElement(root))

	// Add explicit xml: namespace declaration to the element
	require.NoError(t, root.DeclareNamespace("xml", lexicon.NamespaceXML))

	str, err := helium.WriteString(doc)
	require.NoError(t, err)

	// The xml: namespace declaration must NOT appear in the output.
	// libxml2's xmlNsDumpOutput skips prefix "xml" unconditionally.
	require.NotContains(t, str, "xmlns:xml")
}

func TestDumpNsPropagatesWriteError(t *testing.T) {
	t.Parallel()

	doc := helium.NewDefaultDocument()
	root := doc.CreateElement("root")
	require.NoError(t, doc.SetDocumentElement(root))
	require.NoError(t, root.DeclareNamespace("p", "urn:test"))

	writer := helium.NewWriter().XMLDeclaration(false)
	err := writer.WriteTo(&namespaceFailWriter{failOn: "xmlns"}, doc)
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

func TestWritePropagatesWriteError(t *testing.T) {
	t.Parallel()

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
}

func TestWriteOutputUnchanged(t *testing.T) {
	t.Parallel()

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
}

func TestFormatOutput(t *testing.T) {
	t.Parallel()

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
}

func TestXHTML(t *testing.T) {
	t.Parallel()

	t.Run("void element default NS self-closes", func(t *testing.T) {
		t.Parallel()
		// XHTML void elements in the default namespace (prefix == "") should
		// use self-closing " />" syntax, matching libxml2's check:
		//   (cur->ns == NULL) || (cur->ns->prefix == NULL)
		input := `<?xml version="1.0"?>
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Strict//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd">
<html xmlns="http://www.w3.org/1999/xhtml"><head><title>T</title></head><body><br/></body></html>`
		doc, err := helium.NewParser().Parse(t.Context(), []byte(input))
		require.NoError(t, err)

		str, err := helium.WriteString(doc)
		require.NoError(t, err)

		// <br> must be serialized as "<br />" (self-closing), not "<br></br>"
		require.Contains(t, str, "<br />")
		require.NotContains(t, str, "<br></br>")
	})

	t.Run("element children get indented", func(t *testing.T) {
		t.Parallel()
		input := `<?xml version="1.0"?>
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Strict//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd">
<html xmlns="http://www.w3.org/1999/xhtml"><body><p>hello</p><p>world</p></body></html>`
		doc, err := helium.NewParser().Parse(t.Context(), []byte(input))
		require.NoError(t, err)

		var buf strings.Builder
		require.NoError(t, helium.NewWriter().Format(true).WriteTo(&buf, doc))
		str := buf.String()

		// <body> has element children → they should be indented
		require.Contains(t, str, "<body>\n    <p>")
		require.Contains(t, str, "</p>\n  </body>")
	})

	t.Run("text-only elements stay inline", func(t *testing.T) {
		t.Parallel()
		input := `<?xml version="1.0"?>
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Strict//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd">
<html xmlns="http://www.w3.org/1999/xhtml"><body><p>hello</p></body></html>`
		doc, err := helium.NewParser().Parse(t.Context(), []byte(input))
		require.NoError(t, err)

		var buf strings.Builder
		require.NoError(t, helium.NewWriter().Format(true).WriteTo(&buf, doc))
		str := buf.String()

		// <p> has only text → no indentation inside
		require.Contains(t, str, "<p>hello</p>")
	})
}

func TestNoEmpty(t *testing.T) {
	t.Parallel()

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
}

func TestDumpQuotingViaPublicAPI(t *testing.T) {
	t.Parallel()

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
}
func TestWriteRejectsMalformedCommentPI(t *testing.T) {
	doc := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)

	var sb strings.Builder
	require.Error(t, helium.Write(&sb, doc.CreateComment([]byte("a--b"))),
		"comment containing -- must be rejected")
	sb.Reset()
	require.Error(t, helium.Write(&sb, doc.CreateComment([]byte("a-"))),
		"comment ending in - must be rejected")
	sb.Reset()
	require.Error(t, helium.Write(&sb, doc.CreateComment([]byte("-"))),
		"single-dash comment must be rejected")
	sb.Reset()
	require.Error(t, helium.Write(&sb, doc.CreatePI("t", "a?>b")),
		"PI content containing ?> must be rejected")

	// Valid comment/PI still serialize.
	sb.Reset()
	require.NoError(t, helium.Write(&sb, doc.CreateComment([]byte(" ok "))))
	sb.Reset()
	require.NoError(t, helium.Write(&sb, doc.CreateComment([]byte(""))),
		"empty comment must serialize without an out-of-range panic")
	sb.Reset()
	require.NoError(t, helium.Write(&sb, doc.CreatePI("php", "echo 1")))
}

// TestWriteRejectsMalformedPITarget ensures that an invalid PI target — in
// particular one that injects markup — is rejected before being emitted, so
// the serialized output never contains the injection.
func TestWriteRejectsMalformedPITarget(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		target string
	}{
		{name: "injection", target: "x?><evil/><?x"},
		{name: "empty", target: ""},
		{name: "starts-digit", target: "1bad"},
		{name: "has-space", target: "a b"},
		{name: "reserved-xml", target: "xml"},
		{name: "invalid-utf8", target: "\xff\xfe"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			doc := helium.NewDefaultDocument()
			root := doc.CreateElement("r")
			require.NoError(t, doc.SetDocumentElement(root))
			require.NoError(t, root.AddChild(doc.CreatePI(tc.target, "")))

			var sb strings.Builder
			err := helium.Write(&sb, doc)
			require.Error(t, err, "invalid PI target must be rejected")
			require.NotContains(t, sb.String(), "<evil/>",
				"injection must not be emitted")
		})
	}

	// A valid target still serializes.
	doc := helium.NewDefaultDocument()
	root := doc.CreateElement("r")
	require.NoError(t, doc.SetDocumentElement(root))
	require.NoError(t, root.AddChild(doc.CreatePI("php", "echo 1")))
	var sb strings.Builder
	require.NoError(t, helium.Write(&sb, doc))
	require.Contains(t, sb.String(), "<?php echo 1?>")
}

// failOnSubstringWriter fails the first Write whose accumulated tail+payload
// contains trigger, and accepts everything else. It is used to make a specific
// serialization step fail while earlier steps succeed.
type failOnSubstringWriter struct {
	trigger string
	tail    string
}

func (w *failOnSubstringWriter) Write(p []byte) (int, error) {
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

// TestWriteValidationPreservesStickyIOError ensures that when an earlier
// io.Writer failure has already set the sticky error, a subsequent malformed
// comment/PI sibling does not clobber it: WriteTo must surface the original I/O
// error, not the validation error.
func TestWriteValidationPreservesStickyIOError(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		sibling func(*helium.Document) helium.Node
	}{
		{
			name:    "comment",
			sibling: func(d *helium.Document) helium.Node { return d.CreateComment([]byte("a--b")) },
		},
		{
			name:    "pi",
			sibling: func(d *helium.Document) helium.Node { return d.CreatePI("t", "a?>b") },
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			doc := helium.NewDefaultDocument()
			root := doc.CreateElement("r")
			require.NoError(t, doc.SetDocumentElement(root))
			// A malformed top-level sibling serialized after the root element.
			// The newline separator written between top-level nodes is forced
			// to fail, setting the sticky I/O error before the malformed
			// sibling's validation runs. (Unlike a failed element write, the
			// separator failure does not short-circuit the child loop, so the
			// sibling is still reached.)
			require.NoError(t, doc.AddChild(tc.sibling(doc)))

			err := helium.NewWriter().XMLDeclaration(false).WriteTo(&failOnSubstringWriter{trigger: "\n"}, doc)
			require.ErrorIs(t, err, errShortWrite,
				"original I/O error must win over the sibling validation error")
		})
	}
}

func TestWriteNilNode(t *testing.T) {
	t.Parallel()

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
}

// TestSerializeQuotedStringBranches drives the dumpQuotedString writer helper
// through all three quoting branches by serializing notation system IDs that
// contain: no quote, a double quote only (forces single-quote delimiting), and
// both quote kinds (forces double-quote delimiting with &quot; escaping).
func TestSerializeQuotedStringBranches(t *testing.T) {
	t.Parallel()

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	dtd, err := doc.CreateInternalSubset("doc", "", "")
	require.NoError(t, err)

	// no quote -> double-quote delimited.
	_, err = dtd.AddNotation("plain", "", "plain.exe")
	require.NoError(t, err)
	// double quote only -> single-quote delimited.
	_, err = dtd.AddNotation("dq", "", `has"dquote`)
	require.NoError(t, err)
	// both quotes -> double-quote delimited with &quot; escaping of the dquote.
	_, err = dtd.AddNotation("both", "", `has"dq and 'sq'`)
	require.NoError(t, err)

	root := doc.CreateElement("doc")
	require.NoError(t, doc.AddChild(root))

	out, err := helium.WriteString(doc)
	require.NoError(t, err)
	require.Contains(t, out, `"plain.exe"`, "no-quote value double-quote delimited")
	require.Contains(t, out, `'has"dquote'`, "double-quote-only value single-quote delimited")
	require.Contains(t, out, "&quot;", "both-quote value escapes the embedded double quote")
}

// TestSerializeEntityContentWithPercent serializes an internal general entity
// whose replacement text contains a literal '%', driving the dumpEntityContent
// percent-escaping branch in the DTD writer.
func TestSerializeEntityContentWithPercent(t *testing.T) {
	t.Parallel()

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	dtd, err := doc.CreateInternalSubset("doc", "", "")
	require.NoError(t, err)

	// content has no orig set (AddEntity passes orig=""), so the writer falls
	// through to dumpEntityContent; the '%' forces the escaping branch and the
	// '"' forces the &quot; branch.
	_, err = dtd.AddEntity("pct", enum.InternalGeneralEntity, "", "", `50% "done"`)
	require.NoError(t, err)

	root := doc.CreateElement("doc")
	require.NoError(t, doc.AddChild(root))

	out, err := helium.WriteString(doc)
	require.NoError(t, err)
	require.Contains(t, out, "<!ENTITY pct", "entity declaration serialized")
	require.Contains(t, out, "&#x25;", "percent escaped via dumpEntityContent")
	require.Contains(t, out, "&quot;", "embedded quote escaped via dumpEntityContent")
}

// TestWriterOptions exercises the Writer option toggles and serialization paths.
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
	r := d2.CreateElement("r")
	require.NoError(t, d2.AddChild(r))
	require.NoError(t, r.AppendText([]byte("café")))

	buf.Reset()
	err = helium.NewWriter().EscapeNonASCII(true).WriteTo(&buf, d2)
	require.NoError(t, err)
	require.Contains(t, buf.String(), "&#")
}

func TestWriterRejectInvalidChars(t *testing.T) {
	t.Parallel()

	// A C0 control character (U+0007) is invalid in XML 1.0. By default the
	// writer replaces it with U+FFFD; with RejectInvalidChars it fails with
	// ErrInvalidXMLChar (the SERE0006 serialization error).
	textDoc := func() *helium.Document {
		d := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
		r := d.CreateElement("r")
		require.NoError(t, d.AddChild(r))
		require.NoError(t, r.AppendText([]byte("a\x07b")))
		return d
	}

	// Default (with EscapeNonASCII off, matching the xslt3 XML path): the
	// control char is replaced with U+FFFD and no error is raised.
	var buf bytes.Buffer
	require.NoError(t, helium.NewWriter().EscapeNonASCII(false).WriteTo(&buf, textDoc()))
	require.Contains(t, buf.String(), "\uFFFD")

	// RejectInvalidChars rejects the control char regardless of the
	// EscapeNonASCII setting (the check runs before char-reference escaping).
	buf.Reset()
	err := helium.NewWriter().RejectInvalidChars(true).WriteTo(&buf, textDoc())
	require.ErrorIs(t, err, helium.ErrInvalidXMLChar)
	buf.Reset()
	err = helium.NewWriter().EscapeNonASCII(false).RejectInvalidChars(true).WriteTo(&buf, textDoc())
	require.ErrorIs(t, err, helium.ErrInvalidXMLChar)

	// A control char in an attribute value is rejected too (escaping covers
	// attribute values, not only text nodes).
	attrDoc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	r := attrDoc.CreateElement("r")
	require.NoError(t, attrDoc.AddChild(r))
	_, err = r.SetAttribute("v", "x\x07y")
	require.NoError(t, err)
	buf.Reset()
	err = helium.NewWriter().RejectInvalidChars(true).WriteTo(&buf, attrDoc)
	require.ErrorIs(t, err, helium.ErrInvalidXMLChar)

	// A valid document still serializes cleanly with rejection enabled.
	okDoc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	e := okDoc.CreateElement("r")
	require.NoError(t, okDoc.AddChild(e))
	require.NoError(t, e.AppendText([]byte("plain text\tok")))
	buf.Reset()
	require.NoError(t, helium.NewWriter().RejectInvalidChars(true).WriteTo(&buf, okDoc))
	require.Contains(t, buf.String(), "plain text\tok")
}

// TestWriteStringWithoutDTD verifies WriteString on a programmatically built doc.
func TestWriteStringWithoutDTD(t *testing.T) {
	t.Parallel()
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root := doc.CreateElement("root")
	require.NoError(t, doc.AddChild(root))
	require.NoError(t, root.AppendText([]byte("text & more")))

	s, err := helium.WriteString(doc)
	require.NoError(t, err)
	require.True(t, strings.Contains(s, "<root>"))
	require.Contains(t, s, "&amp;")
}

// TestWriterNormalization exercises Writer.Normalization: Unicode normalization is
// scoped to text-node and attribute-value character content (Serialization 3.1
// §4). Element/attribute names, comments, and PIs are never normalized.
func TestWriterNormalization(t *testing.T) {
	t.Parallel()
	const decomposed = "e\u0301" // "e" + combining acute
	const composed = "\u00e9"    // U+00E9
	src := "<caf" + decomposed + " at" + decomposed + "=\"" + decomposed + "\">" +
		"<!--" + decomposed + "--><?p " + decomposed + "?>" + decomposed +
		"</caf" + decomposed + ">"
	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err)

	var buf strings.Builder
	// EscapeNonASCII(false) so the composed é appears literally, isolating the
	// normalization effect from the writer's numeric-reference escaping.
	err = helium.NewWriter().XMLDeclaration(false).EscapeNonASCII(false).
		Normalization("NFC").WriteTo(&buf, doc)
	require.NoError(t, err)
	out := buf.String()

	// Text and attribute value are composed; names, comment, and PI stay decomposed.
	require.Contains(t, out, ">"+composed+"</caf"+decomposed+">", "text normalized: %q", out)
	require.Contains(t, out, "at"+decomposed+"=\""+composed+"\"", "attr value normalized, name not: %q", out)
	require.Contains(t, out, "<caf"+decomposed, "element name not normalized: %q", out)
	require.Contains(t, out, "<!--"+decomposed+"-->", "comment not normalized: %q", out)
	require.Contains(t, out, "<?p "+decomposed+"?>", "PI not normalized: %q", out)
	require.NotContains(t, out, "caf"+composed, "name must stay decomposed: %q", out)

	// Without normalization the output is byte-identical to the source content.
	var raw strings.Builder
	err = helium.NewWriter().XMLDeclaration(false).EscapeNonASCII(false).WriteTo(&raw, doc)
	require.NoError(t, err)
	require.NotContains(t, raw.String(), composed, "no normalization by default: %q", raw.String())
}

// TestWriteReconcilesSubtreeNamespaces verifies that serializing a subtree
// re-declares any namespace prefix its elements or attributes use but that was
// bound only on an ancestor outside the subtree, so the output reparses. This
// is the situation an XSLT result tree creates when it grafts in nodes from a
// source document (W3C xslt30 si-lre-029/904/905, si-element-029).
func TestWriteReconcilesSubtreeNamespaces(t *testing.T) {
	t.Parallel()

	// firstElem returns the first element child of n, or n itself.
	firstElem := func(n helium.Node) helium.Node {
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			if c.Type() == helium.ElementNode {
				return c
			}
		}
		return n
	}

	serializeSubtree := func(t *testing.T, src string) string {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
		require.NoError(t, err)
		var buf bytes.Buffer
		err = helium.NewWriter().XMLDeclaration(false).WriteTo(&buf, firstElem(doc.DocumentElement()))
		require.NoError(t, err)
		return buf.String()
	}

	t.Run("prefixed attribute bound on outer ancestor", func(t *testing.T) {
		t.Parallel()
		// gml is declared on the root; the serialized <b:elem> subtree uses it
		// only on the gml:id attribute. Both b (element prefix) and gml (attr
		// prefix) must be declared in the fragment.
		out := serializeSubtree(t, `<root xmlns:gml="urn:g" xmlns:b="urn:b"><b:elem gml:id="x1"/></root>`)
		require.Contains(t, out, `xmlns:gml="urn:g"`)
		require.Contains(t, out, `xmlns:b="urn:b"`)
		_, err := helium.NewParser().Parse(t.Context(), []byte(out))
		require.NoError(t, err, "reconciled fragment must reparse: %q", out)
	})

	t.Run("prefix used only on element name", func(t *testing.T) {
		t.Parallel()
		out := serializeSubtree(t, `<root xmlns:p="urn:p"><p:child>text</p:child></root>`)
		require.Contains(t, out, `xmlns:p="urn:p"`)
		_, err := helium.NewParser().Parse(t.Context(), []byte(out))
		require.NoError(t, err, "reconciled fragment must reparse: %q", out)
	})

	t.Run("locally declared prefix not duplicated", func(t *testing.T) {
		t.Parallel()
		// gml is declared on <b:elem> itself; reconciliation must not emit a
		// second xmlns:gml.
		out := serializeSubtree(t, `<root xmlns:b="urn:b"><b:elem xmlns:gml="urn:g" gml:id="x1"/></root>`)
		require.Equal(t, 1, strings.Count(out, `xmlns:gml=`), "no duplicate declaration: %q", out)
		require.Contains(t, out, `xmlns:b="urn:b"`)
		_, err := helium.NewParser().Parse(t.Context(), []byte(out))
		require.NoError(t, err, "reconciled fragment must reparse: %q", out)
	})

	t.Run("xml prefix never reconciled", func(t *testing.T) {
		t.Parallel()
		// xml:lang uses the implicitly-bound xml prefix; the serializer must not
		// synthesize xmlns:xml.
		out := serializeSubtree(t, `<root xmlns:b="urn:b"><b:elem xml:lang="en"/></root>`)
		require.NotContains(t, out, `xmlns:xml=`, "xml prefix is implicitly bound: %q", out)
		_, err := helium.NewParser().Parse(t.Context(), []byte(out))
		require.NoError(t, err, "reconciled fragment must reparse: %q", out)
	})
}

// TestWriterMapSettersCloneInput verifies the value-style contract for every
// map-taking Writer setter: the setter copies its input, so mutating the
// caller's map AFTER the setter call does not change the configured Writer.
// Without cloning the setters would retain the caller's map by reference, so a
// post-set mutation (or a concurrent write) would silently change serialization.
func TestWriterMapSettersCloneInput(t *testing.T) {
	t.Parallel()

	serialize := func(t *testing.T, w helium.Writer, n helium.Node) string {
		t.Helper()
		var buf bytes.Buffer
		require.NoError(t, w.WriteTo(&buf, n))
		return buf.String()
	}

	parse := func(t *testing.T, src string) *helium.Document {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
		require.NoError(t, err)
		return doc
	}

	t.Run("CharacterMap", func(t *testing.T) {
		t.Parallel()
		doc := parse(t, `<a>o</a>`)
		m := map[rune]string{}
		w := helium.NewWriter().CharacterMap(m)
		m['o'] = "0" // mutate after set — must not reach the Writer
		out := serialize(t, w, doc)
		require.Contains(t, out, ">o</a>", "unmutated char preserved: %q", out)
		require.NotContains(t, out, ">0</a>", "post-set mutation must not apply: %q", out)
	})

	t.Run("CDATASectionElements", func(t *testing.T) {
		t.Parallel()
		doc := parse(t, `<a>hi</a>`)
		m := map[string]struct{}{}
		w := helium.NewWriter().CDATASectionElements(m)
		m["{}a"] = struct{}{} // mutate after set
		out := serialize(t, w, doc)
		require.NotContains(t, out, "CDATA", "post-set mutation must not apply: %q", out)
	})

	t.Run("SuppressIndentElements", func(t *testing.T) {
		t.Parallel()
		doc := parse(t, `<a><b>x</b></a>`)
		m := map[string]struct{}{}
		w := helium.NewWriter().XMLDeclaration(false).Format(true).SuppressIndentElements(m)
		m["{}a"] = struct{}{} // mutate after set
		out := serialize(t, w, doc)
		// With no suppression the formatter indents <b> onto its own line.
		require.Contains(t, out, "\n", "output stays formatted: %q", out)
		require.NotContains(t, out, "<a><b>", "post-set suppression must not apply: %q", out)
	})

	t.Run("InheritedNamespaces", func(t *testing.T) {
		t.Parallel()
		firstElem := func(n helium.Node) helium.Node {
			for c := n.FirstChild(); c != nil; c = c.NextSibling() {
				if c.Type() == helium.ElementNode {
					return c
				}
			}
			return n
		}
		doc := parse(t, `<root xmlns:p="urn:p"><p:child>text</p:child></root>`)
		m := map[string]string{}
		w := helium.NewWriter().XMLDeclaration(false).InheritedNamespaces(m)
		m["p"] = "urn:p" // mutate after set — must not suppress the redeclaration
		out := serialize(t, w, firstElem(doc.DocumentElement()))
		require.Contains(t, out, `xmlns:p="urn:p"`, "post-set inherited ns must not apply: %q", out)
	})
}

// TestWriterStructuralErrorsMatchable verifies that each structural-serialization
// failure the writer detects is reported via a named sentinel a caller can match
// with errors.Is, rather than an anonymous string-literal error.
func TestWriterStructuralErrorsMatchable(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		build    func(t *testing.T) *helium.Document
		sentinel error
	}{
		{
			name: "invalid element name",
			build: func(t *testing.T) *helium.Document {
				doc := helium.NewDefaultDocument()
				root := doc.CreateElement(`root injected="1"`)
				require.NoError(t, doc.SetDocumentElement(root))
				return doc
			},
			sentinel: helium.ErrWriterInvalidElementName,
		},
		{
			name: "reserved element name",
			build: func(t *testing.T) *helium.Document {
				doc := helium.NewDefaultDocument()
				root := doc.CreateElement("xmlns:root")
				require.NoError(t, doc.SetDocumentElement(root))
				return doc
			},
			sentinel: helium.ErrWriterReservedElementName,
		},
		{
			name: "invalid attribute name",
			build: func(t *testing.T) *helium.Document {
				doc := helium.NewDefaultDocument()
				root := doc.CreateElement("root")
				require.NoError(t, doc.SetDocumentElement(root))
				_, err := root.SetAttribute(`x onmouseover`, "1")
				require.NoError(t, err)
				return doc
			},
			sentinel: helium.ErrWriterInvalidAttributeName,
		},
		{
			name: "reserved attribute name",
			build: func(t *testing.T) *helium.Document {
				doc := helium.NewDefaultDocument()
				root := doc.CreateElement("root")
				require.NoError(t, doc.SetDocumentElement(root))
				_, err := root.SetAttribute("xmlns", "urn:evil")
				require.NoError(t, err)
				return doc
			},
			sentinel: helium.ErrWriterReservedAttributeName,
		},
		{
			name: "reserved namespace prefix",
			build: func(t *testing.T) *helium.Document {
				doc := helium.NewDefaultDocument()
				root := doc.CreateElement("root")
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
				root := doc.CreateElement("root")
				require.NoError(t, doc.SetDocumentElement(root))
				require.NoError(t, root.DeclareNamespace("bad prefix", "urn:x"))
				return doc
			},
			sentinel: helium.ErrWriterInvalidNamespacePrefix,
		},
		{
			name: "invalid comment content",
			build: func(t *testing.T) *helium.Document {
				doc := helium.NewDefaultDocument()
				root := doc.CreateElement("root")
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
				root := doc.CreateElement("root")
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
				root := doc.CreateElement("root")
				require.NoError(t, doc.SetDocumentElement(root))
				require.NoError(t, root.AddChild(doc.CreatePI("t", "a?>b")))
				return doc
			},
			sentinel: helium.ErrWriterInvalidPIContent,
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
}

// TestWriterNormalizationRejectsInvalidForm verifies that an unrecognized
// normalization-form value is observable (ErrUnsupportedNormalizationForm) rather
// than silently disabling normalization, while the supported forms and the
// disabling values ("", "none") are accepted.
func TestWriterNormalizationRejectsInvalidForm(t *testing.T) {
	t.Parallel()

	newDoc := func(t *testing.T) *helium.Document {
		doc := helium.NewDefaultDocument()
		root := doc.CreateElement("root")
		require.NoError(t, doc.SetDocumentElement(root))
		require.NoError(t, root.AppendText([]byte("x")))
		return doc
	}

	for _, form := range []string{"NFCC", "nfc", "fully-normalized", "NONE", "garbage"} {
		t.Run("rejects "+form, func(t *testing.T) {
			t.Parallel()
			var buf strings.Builder
			err := helium.NewWriter().Normalization(form).WriteTo(&buf, newDoc(t))
			require.Error(t, err, "unrecognized normalization form must be rejected")
			require.ErrorIs(t, err, helium.ErrUnsupportedNormalizationForm)
			require.Empty(t, buf.String(), "no output byte before the rejection")
		})
	}

	for _, form := range []string{"", "none", "NFC", "NFD", "NFKC", "NFKD"} {
		t.Run("accepts "+form, func(t *testing.T) {
			t.Parallel()
			var buf strings.Builder
			err := helium.NewWriter().Normalization(form).WriteTo(&buf, newDoc(t))
			require.NoError(t, err, "supported normalization form must serialize")
		})
	}

	// A bare element (non-Document) path is also validated.
	t.Run("bare element path", func(t *testing.T) {
		t.Parallel()
		doc := newDoc(t)
		var buf strings.Builder
		err := helium.NewWriter().Normalization("bogus").WriteTo(&buf, doc.DocumentElement())
		require.ErrorIs(t, err, helium.ErrUnsupportedNormalizationForm)
	})
}
