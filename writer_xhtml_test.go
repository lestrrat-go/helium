package helium_test

import (
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

func TestSerializeXHTML(t *testing.T) {
	t.Parallel()

	// parses an XHTML 1.0 document (recognized via its PUBLIC
	// identifier) and serializes it, exercising the XHTML-specific writer paths:
	// void elements, the html xmlns injection, head content-type meta injection, and
	// boolean attribute handling.
	t.Run("basic", func(t *testing.T) {
		const src = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Strict//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>t</title></head>
<body>
<p>para<br/>after break</p>
<img src="x.png" alt="x"/>
<form action="/go"><input type="checkbox" checked="checked"/></form>
<hr/>
</body>
</html>`

		doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
		require.NoError(t, err, "parse XHTML document")

		out, err := helium.WriteString(doc)
		require.NoError(t, err, "serialize XHTML document")

		// Void elements use the " />" form in XHTML output.
		require.Contains(t, out, "<br />")
		require.Contains(t, out, "<hr />")
		require.True(t, strings.Contains(out, "<img"), "img element serialized")

		// Re-parse the serialized output to confirm well-formedness.
		_, err = helium.NewParser().Parse(t.Context(), []byte(out))
		require.NoError(t, err, "re-parse serialized XHTML")
	})

	// serializes XHTML with formatting enabled to drive
	// the indentation branches of the XHTML writer.
	t.Run("formatted", func(t *testing.T) {
		const src = `<?xml version="1.0"?>
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>t</title></head>
<body><div><p>x</p></div></body>
</html>`

		doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
		require.NoError(t, err)

		var buf strings.Builder
		err = helium.NewWriter().Format(true).WriteTo(&buf, doc)
		require.NoError(t, err)
		require.Contains(t, buf.String(), "<html")
	})

	// the XHTML serializer honors the
	// mixed-content rule: an element with any text-like child (mixed content) is
	// serialized inline with no injected indentation whitespace, and the suppression
	// propagates across the whole subtree — matching the regular writeNode path and
	// libxml2's xhtmlNodeDumpOutput. A second parse+format pass must be byte-identical
	// (idempotent); injected whitespace would corrupt the text content on each pass.
	t.Run("mixed content", func(t *testing.T) {
		const src = `<?xml version="1.0"?>
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
<body><p>left<b><i/></b>right</p></body>
</html>`

		doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
		require.NoError(t, err, "parse XHTML document")

		var buf strings.Builder
		err = helium.NewWriter().Format(true).WriteTo(&buf, doc)
		require.NoError(t, err, "serialize XHTML with formatting")
		out := buf.String()

		// The mixed-content <p> (and the nested <b>) serialize inline: the text runs
		// abut their sibling elements with no injected newline/indent.
		require.Contains(t, out, "<p>left<b><i></i></b>right</p>", "output:\n%s", out)

		// A second parse+format pass must reproduce the first byte-for-byte.
		doc2, err := helium.NewParser().Parse(t.Context(), []byte(out))
		require.NoError(t, err, "re-parse serialized XHTML")
		var buf2 strings.Builder
		err = helium.NewWriter().Format(true).WriteTo(&buf2, doc2)
		require.NoError(t, err, "re-serialize XHTML with formatting")
		require.Equal(t, out, buf2.String(), "XHTML mixed-content formatting must be idempotent")
	})

	// Writer.CharacterMap applies to XHTML
	// attribute values (including the synthesized id-from-name attribute) and text
	// content in the XHTML serialization path (Serialization 3.1 §6). This XHTML path
	// performs no URI percent-encoding, so character maps apply to every attribute.
	t.Run("character map", func(t *testing.T) {
		const src = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Strict//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>t</title></head>
<body><a name="foo" title="foo">foo</a></body>
</html>`

		doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
		require.NoError(t, err, "parse XHTML document")

		var buf strings.Builder
		err = helium.NewWriter().CharacterMap(map[rune]string{'o': "0"}).WriteTo(&buf, doc)
		require.NoError(t, err, "serialize XHTML with character map")
		out := buf.String()

		// The character map ('o' -> "0") applies to the source attribute values, the
		// synthesized id (derived from @name on <a>), and text content.
		require.Contains(t, out, `name="f00"`, "output:\n%s", out)
		require.Contains(t, out, `title="f00"`, "output:\n%s", out)
		require.Contains(t, out, `id="f00"`, "output:\n%s", out)
		require.Contains(t, out, `>f00<`, "output:\n%s", out)
	})

	t.Run("through the public API", func(t *testing.T) {
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
	})

	// the generic XML writer serializes a
	// document node of HTMLDocumentNode type (as produced by NewHTMLDocument and by
	// the HTML parser) through the same DocumentNode path as an XML document, rather
	// than falling to the element path and rejecting the "(document)" node name.
	t.Run("HTML document node", func(t *testing.T) {
		doc := helium.NewHTMLDocument()
		root, err := doc.CreateElement("html")
		require.NoError(t, err)
		require.NoError(t, doc.SetDocumentElement(root))

		body, err := doc.CreateElement("body")
		require.NoError(t, err)
		require.NoError(t, root.AddChild(body))
		require.NoError(t, body.AddChild(doc.CreateText([]byte("hi"))))

		out, err := helium.WriteString(doc)
		require.NoError(t, err, "generic writer must serialize an HTMLDocumentNode document")

		require.True(t, strings.Contains(out, "<body>hi</body>") || strings.Contains(out, "<body>hi</body >"),
			"the HTML document's children must serialize, got %q", out)
		require.Contains(t, out, "<html>", "the document element must serialize")
	})
}

func TestXHTMLRejects(t *testing.T) {
	t.Parallel()

	// Finding 1: when an XHTML
	// element has an invalid attribute name AND non-element child content, the
	// serializer must abort at the first error and must NOT emit any of the child
	// content before returning the error.
	t.Run("an attribute error emits no partial children", func(t *testing.T) {
		doc := helium.NewDefaultDocument()
		_, err := doc.CreateInternalSubset(
			"html",
			"-//W3C//DTD XHTML 1.0 Strict//EN",
			"http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd",
		)
		require.NoError(t, err)

		root, err := doc.CreateElement("html")
		require.NoError(t, err)
		require.NoError(t, doc.SetDocumentElement(root))

		// Invalid attribute name on the element: serialization must fail before any
		// child content is written.
		err = root.SetAttribute(`x onmouseover`, "1")
		require.NoError(t, err)

		const childMarker = "SECRET_CHILD_TEXT"
		text := doc.CreateText([]byte(childMarker))
		require.NoError(t, root.AddChild(text))

		var buf strings.Builder
		err = helium.NewWriter().WriteTo(&buf, doc)
		require.Error(t, err, "invalid XHTML attribute name must fail serialization")
		require.NotContains(t, buf.String(), childMarker,
			"no child content must be emitted after an attribute-name error")
	})

	t.Run("injected names", func(t *testing.T) {
		// newXHTMLDoc builds a document whose internal subset is an XHTML DTD, so
		// serialization routes through dumpXHTMLNode / dumpXHTMLAttrList, bypassing
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
			root, err := doc.CreateElement(`html injected="1"`)
			require.NoError(t, err)
			require.NoError(t, doc.SetDocumentElement(root))

			_, err = helium.WriteString(doc)
			require.Error(t, err, "injected XHTML element name must not serialize")
		})

		t.Run("attribute name injection", func(t *testing.T) {
			t.Parallel()
			doc := newXHTMLDoc(t)
			root, err := doc.CreateElement("html")
			require.NoError(t, err)
			require.NoError(t, doc.SetDocumentElement(root))
			err = root.SetAttribute(`x onmouseover`, "1")
			require.NoError(t, err)

			_, err = helium.WriteString(doc)
			require.Error(t, err, "injected XHTML attribute name must not serialize")
		})

		t.Run("valid element name serializes", func(t *testing.T) {
			t.Parallel()
			doc := newXHTMLDoc(t)
			root, err := doc.CreateElement("html")
			require.NoError(t, err)
			require.NoError(t, doc.SetDocumentElement(root))

			str, err := helium.WriteString(doc)
			require.NoError(t, err)
			require.Contains(t, str, "<html")
		})

		t.Run("valid namespaced name serializes", func(t *testing.T) {
			t.Parallel()
			doc := newXHTMLDoc(t)
			root, err := doc.CreateElement("html")
			require.NoError(t, err)
			require.NoError(t, doc.SetDocumentElement(root))
			require.NoError(t, root.SetActiveNamespace("p", "urn:example"))

			str, err := helium.WriteString(doc)
			require.NoError(t, err)
			require.Contains(t, str, "<p:html")
		})
	})

	t.Run("injected namespace prefix", func(t *testing.T) {
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
			root, err := doc.CreateElement("html")
			require.NoError(t, err)
			require.NoError(t, doc.SetDocumentElement(root))
			require.NoError(t, root.DeclareNamespace(`p injected="1`, "urn"))

			_, err = helium.WriteString(doc)
			require.Error(t, err, "injected XHTML namespace prefix must not serialize")
		})

		t.Run("valid prefix serializes", func(t *testing.T) {
			t.Parallel()
			doc := newXHTMLDoc(t)
			root, err := doc.CreateElement("html")
			require.NoError(t, err)
			require.NoError(t, doc.SetDocumentElement(root))
			require.NoError(t, root.DeclareNamespace("p", "urn:example"))

			str, err := helium.WriteString(doc)
			require.NoError(t, err)
			require.Contains(t, str, `xmlns:p="urn:example"`)
		})
	})

	// the XHTML
	// serialization path (taken when the internal subset is an XHTML 1.0 DTD)
	// applies the same unbound-prefix guard as the generic writeNode path: a QName
	// whose prefix is bound to an empty namespace URI has no reparseable
	// serialization and must fail with ErrWriterUnboundNamespacePrefix.
	t.Run("unbound namespace prefix", func(t *testing.T) {
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

		t.Run("unbound element prefix rejected", func(t *testing.T) {
			t.Parallel()
			doc := newXHTMLDoc(t)
			ns, err := doc.CreateNamespace("foo", "")
			require.NoError(t, err)
			root, err := doc.CreateElementNS("html", ns)
			require.NoError(t, err)
			require.NoError(t, doc.SetDocumentElement(root))

			_, err = helium.WriteString(doc)
			require.Error(t, err, "unbound XHTML element prefix must not serialize")
			require.ErrorIs(t, err, helium.ErrWriterUnboundNamespacePrefix)
		})

		t.Run("unbound attribute prefix rejected", func(t *testing.T) {
			t.Parallel()
			doc := newXHTMLDoc(t)
			root, err := doc.CreateElement("html")
			require.NoError(t, err)
			require.NoError(t, doc.SetDocumentElement(root))
			ns, err := doc.CreateNamespace("foo", "")
			require.NoError(t, err)
			require.NoError(t, root.SetAttributeNS("bar", "baz", ns))

			_, err = helium.WriteString(doc)
			require.Error(t, err, "unbound XHTML attribute prefix must not serialize")
			require.ErrorIs(t, err, helium.ErrWriterUnboundNamespacePrefix)
		})

		t.Run("well-formed XHTML serializes unchanged", func(t *testing.T) {
			t.Parallel()
			doc := newXHTMLDoc(t)
			root, err := doc.CreateElement("html")
			require.NoError(t, err)
			require.NoError(t, doc.SetDocumentElement(root))
			require.NoError(t, root.SetAttribute("lang", "en"))

			str, err := helium.WriteString(doc)
			require.NoError(t, err)
			require.Equal(t, `<?xml version="1.0"?>
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Strict//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd">
<html xmlns="http://www.w3.org/1999/xhtml" lang="en" xml:lang="en"></html>
`, str)
		})
	})
}
