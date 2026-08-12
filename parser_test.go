package helium_test

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

func findDocumentElement(doc *helium.Document) helium.Node {
	return doc.DocumentElement()
}

// stringParseInput implements sax.ParseInput for testing.
type stringParseInput struct {
	*strings.Reader
	uri string
}

func newStringParseInput(content, uri string) *stringParseInput {
	return &stringParseInput{Reader: strings.NewReader(content), uri: uri}
}

func (s *stringParseInput) URI() string { return s.uri }

func (fsys underReportingFS) Open(string) (fs.File, error) {
	return &underReportingFile{read: fsys.read}, nil
}

func (fsys errReadingFS) Open(string) (fs.File, error) {
	return &errReadingFile{cap: fsys.cap, hitReadErr: fsys.hitReadErr}, nil
}

func (fsys overReportingFS) Open(string) (fs.File, error) {
	return &overReportingFile{data: fsys.data}, nil
}

const extDTDName = "ext.dtd"

func (fsys partialReadFS) Open(string) (fs.File, error) {
	return &partialReadFile{prefix: fsys.prefix}, nil
}

// dtdSystemID is the external-DTD SYSTEM identifier (and MapFS filename) shared
// by the external-subset parser tests in this package.
const dtdSystemID = "d.dtd"

// peSystemID is the external parameter-entity SYSTEM identifier (and MapFS
// filename) shared by the external-PE parser tests in this package.
const peSystemID = "pe.ent"

func (r *recordingFS) Open(name string) (fs.File, error) {
	r.mu.Lock()
	r.opened = append(r.opened, name)
	r.mu.Unlock()
	return r.inner.Open(name)
}

func (r *recordingFS) wasOpened(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Contains(r.opened, name)
}

func (r *dataThenErrReader) Read(p []byte) (int, error) {
	if r.done {
		return 0, io.EOF
	}
	r.done = true
	n := copy(p, r.data)
	return n, r.err
}

func (r *blockingReader) Read([]byte) (int, error) {
	r.once.Do(func() { close(r.entered) })
	<-r.done
	return 0, io.EOF
}

func (r *stoppableEBCDICReader) Stop() { r.stopped.Store(true) }

func (r *stoppableEBCDICReader) Read(p []byte) (int, error) {
	if r.stopped.Load() {
		return 0, io.EOF
	}
	if r.pos < len(r.head) {
		n := copy(p, r.head[r.pos:])
		r.pos += n
		return n, nil
	}
	for i := range p {
		p[i] = 0x40 // EBCDIC space: an unterminated whitespace run, never EOF
	}
	return len(p), nil
}

func (r *infiniteBlankReader) Stop() { r.stopped.Store(true) }

func (r *infiniteBlankReader) Read(p []byte) (int, error) {
	if r.stopped.Load() {
		return 0, io.EOF
	}
	if r.pos < len(r.head) {
		n := copy(p, r.head[r.pos:])
		r.pos += n
		return n, nil
	}
	for i := range p {
		p[i] = ' '
	}
	return len(p), nil
}

func (r *prefixThenZeroThenRestReader) Read(p []byte) (int, error) {
	if r.pos < r.prefixLen {
		n := copy(p, r.data[r.pos:r.prefixLen])
		r.pos += n
		return n, nil
	}
	if !r.emittedZero {
		r.emittedZero = true
		return 0, nil // transient empty read mid-sniff
	}
	if r.pos < len(r.data) {
		n := copy(p, r.data[r.pos:])
		r.pos += n
		return n, nil
	}
	return 0, io.EOF
}

func (r *ebcdicSlowTailReader) Read(p []byte) (int, error) {
	if r.served == 0 && len(r.prefix) > 0 {
		// First Read: hand back the EBCDIC prefix so detection succeeds.
		n := copy(p, r.prefix)
		r.prefix = nil
		return n, nil
	}
	if r.served == 0 {
		// First tail Read: signal the test, then wait until it has cancelled the
		// context before returning one byte. This makes the test deterministic:
		// when the drain loop iterates and re-checks ctx on the next pass, the
		// cancellation is guaranteed to be observable, so a ctx-honoring loop
		// returns instead of entering the blocking Read below.
		r.served++
		r.once.Do(func() { close(r.entered) })
		<-r.cancelled
		if len(p) > 0 {
			p[0] = ' '
			return 1, nil
		}
		return 0, nil
	}
	// Later tail Reads block until the test tears the reader down. A
	// ctx-honoring drain loop never reaches here after cancellation.
	<-r.gate
	return 0, io.EOF
}

func (r *genCharDataReader) Read(p []byte) (int, error) {
	n := 0
	if len(r.prefix) > 0 {
		c := copy(p, r.prefix)
		r.prefix = r.prefix[c:]
		n += c
	}
	if n < len(p) && r.remain > 0 {
		c := min(len(p)-n, r.remain)
		for i := range c {
			p[n+i] = r.fill
		}
		r.remain -= c
		n += c
	}
	if n < len(p) && len(r.prefix) == 0 && r.remain == 0 && len(r.suffix) > 0 {
		c := copy(p[n:], r.suffix)
		r.suffix = r.suffix[c:]
		n += c
	}
	if n == 0 {
		return 0, io.EOF
	}
	r.nread += n
	return n, nil
}

// nopCatalog is a CatalogResolver that never resolves anything. It exists only
// to drive the Parser.Catalog configuration path.
type nopCatalog struct{}

func (nopCatalog) Resolve(_ context.Context, _, _ string) string { return "" }
func (nopCatalog) ResolveURI(_ context.Context, _ string) string { return "" }

func (r *blockUntilCancelledBlankReader) Read(p []byte) (int, error) {
	if r.pos < len(r.head) {
		n := copy(p, r.head[r.pos:])
		r.pos += n
		return n, nil
	}
	r.once.Do(func() {
		close(r.entered)
		<-r.cancelled
	})
	for i := range p {
		p[i] = ' '
	}
	return len(p), nil
}

func (r *headThenReadErrReader) Read(p []byte) (int, error) {
	if r.pos < len(r.head) {
		n := copy(p, r.head[r.pos:])
		r.pos += n
		return n, nil
	}
	return 0, r.err
}

func (r *recordingXInclude) Process(_ context.Context, doc *helium.Document) (int, error) {
	r.calls++
	r.doc = doc
	return r.n, r.err
}

func TestParse(t *testing.T) {
	t.Parallel()

	t.Run("documents", func(t *testing.T) {
		//nolint:dupword // "L\nL" is intentional XML content
		const input = `<?xml version="1.0"?>
<root foo="bar">
	<!-- this is a sample comment -->
  <child>foo</child>
	<child><![CDATA[
H
E
L
L
O!]]></child>
</root>`
		p := helium.NewParser()
		_, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err, "Parse should succeed for '%s'", input)
	})

	t.Run("an empty document", func(t *testing.T) {
		p := helium.NewParser()
		// BOM only
		_, err := p.Parse(t.Context(), []byte{0x00, 0x00, 0x00, 0x3C})
		require.Error(t, err, "Parsing BOM only should fail")
	})

	// parses a variety of well-formed constructs that
	// exercise the success branches of the same parser functions the malformed tests
	// hit on the error side: a leading PI, a comment, a CDATA section, namespaced
	// elements/attributes, character references, and an explicit encoding/standalone
	// declaration.
	t.Run("a variety of well-formed documents", func(t *testing.T) {
		const src = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<?pi-target some data?>
<!-- a leading comment -->
<p:root xmlns:p="urn:p" xmlns="urn:default" p:attr="v" plain="w">
  <![CDATA[ raw <markup> & stuff ]]>
  text &#65; &#x42; &amp; more
  <p:child/>
  <plain-child attr="x"/>
</p:root>`

		doc, err := helium.NewParser().Parse(context.Background(), []byte(src))
		require.NoError(t, err)
		root := doc.DocumentElement()
		require.NotNil(t, root)
		require.Equal(t, "root", root.LocalName())

		out, err := helium.WriteString(doc)
		require.NoError(t, err)
		require.Contains(t, out, "urn:p")
		require.Contains(t, out, "CDATA")
	})

	t.Run("in a node context", func(t *testing.T) {
		t.Run("basic fragment", func(t *testing.T) {
			doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root/>`))
			require.NoError(t, err)

			root := doc.DocumentElement()
			require.NotNil(t, root)

			result, err := helium.NewParser().ParseInNodeContext(t.Context(), root, []byte(`<child/>`))
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Equal(t, helium.ElementNode, result.Type())
			require.Equal(t, "child", result.Name())
			require.Nil(t, result.Parent())
		})

		t.Run("multiple sibling nodes", func(t *testing.T) {
			doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root/>`))
			require.NoError(t, err)

			root := doc.DocumentElement()
			result, err := helium.NewParser().ParseInNodeContext(t.Context(), root, []byte(`<a/><b/>text`))
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Equal(t, "a", result.Name())

			sib := result.NextSibling()
			require.NotNil(t, sib)
			require.Equal(t, "b", sib.Name())

			text := sib.NextSibling()
			require.NotNil(t, text)
			require.Equal(t, helium.TextNode, text.Type())
			require.Equal(t, "text", string(text.Content()))
		})

		t.Run("namespace inheritance", func(t *testing.T) {
			doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root xmlns:ns="http://example.com/ns"><child/></root>`))
			require.NoError(t, err)

			root := doc.DocumentElement()
			result, err := helium.NewParser().ParseInNodeContext(t.Context(), root, []byte(`<ns:item>hello</ns:item>`))
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Equal(t, helium.ElementNode, result.Type())
			// The element should have been parsed successfully using the inherited ns prefix
			elem, ok := result.(*helium.Element)
			require.True(t, ok)
			require.Equal(t, "ns:item", elem.Name())
		})

		t.Run("nested namespace inheritance", func(t *testing.T) {
			doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root xmlns:a="http://a.example.com"><middle xmlns:b="http://b.example.com"><child/></middle></root>`))
			require.NoError(t, err)

			root := doc.DocumentElement()
			middle := root.FirstChild()
			require.NotNil(t, middle)

			// Parse fragment in context of middle -- should see both a: and b: prefixes
			result, err := helium.NewParser().ParseInNodeContext(t.Context(), middle, []byte(`<a:x/><b:y/>`))
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Equal(t, "a:x", result.Name())
			sib := result.NextSibling()
			require.NotNil(t, sib)
			require.Equal(t, "b:y", sib.Name())
		})

		t.Run("fragment with own namespaces", func(t *testing.T) {
			doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root/>`))
			require.NoError(t, err)

			root := doc.DocumentElement()
			result, err := helium.NewParser().ParseInNodeContext(t.Context(), root, []byte(`<ns:item xmlns:ns="http://example.com/ns">hello</ns:item>`))
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Equal(t, "ns:item", result.Name())
		})

		t.Run("document as context", func(t *testing.T) {
			doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root/>`))
			require.NoError(t, err)

			result, err := helium.NewParser().ParseInNodeContext(t.Context(), doc, []byte(`<elem/>`))
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Equal(t, "elem", result.Name())
		})

		t.Run("non-element context walks up", func(t *testing.T) {
			doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root xmlns:ns="http://example.com/ns"><child>some text</child></root>`))
			require.NoError(t, err)

			root := doc.DocumentElement()
			child := root.FirstChild()
			require.NotNil(t, child)
			textNode := child.FirstChild()
			require.NotNil(t, textNode)
			require.Equal(t, helium.TextNode, textNode.Type())

			// Parse in context of text node -- should walk up to <child> then <root>
			result, err := helium.NewParser().ParseInNodeContext(t.Context(), textNode, []byte(`<ns:item/>`))
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Equal(t, "ns:item", result.Name())
		})

		t.Run("DTD entity resolution", func(t *testing.T) {
			doc, err := helium.NewParser().Parse(t.Context(), []byte(`<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ENTITY greeting "hello world">
]>
<doc/>`))
			require.NoError(t, err)

			root := doc.DocumentElement()
			p := helium.NewParser().SubstituteEntities(true)
			result, err := p.ParseInNodeContext(t.Context(), root, []byte(`<item>&greeting;</item>`))
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Equal(t, "item", result.Name())
			// The entity should have been resolved
			require.Equal(t, "hello world", string(result.FirstChild().Content()))
		})

		t.Run("empty fragment", func(t *testing.T) {
			doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root/>`))
			require.NoError(t, err)

			root := doc.DocumentElement()
			result, err := helium.NewParser().ParseInNodeContext(t.Context(), root, []byte(``))
			require.NoError(t, err)
			require.Nil(t, result)
		})

		t.Run("nil node", func(t *testing.T) {
			_, err := helium.NewParser().ParseInNodeContext(t.Context(), nil, []byte(`<child/>`))
			require.Error(t, err)
		})

		t.Run("original document preserved", func(t *testing.T) {
			doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root><existing/></root>`))
			require.NoError(t, err)

			root := doc.DocumentElement()
			_, err = helium.NewParser().ParseInNodeContext(t.Context(), root, []byte(`<new/>`))
			require.NoError(t, err)

			// Original document should still have its children
			require.NotNil(t, doc.FirstChild())
			docRoot := doc.DocumentElement()
			require.NotNil(t, docRoot)
			require.Equal(t, "root", docRoot.Name())
			require.NotNil(t, docRoot.FirstChild())
			require.Equal(t, "existing", docRoot.FirstChild().Name())
		})
	})
}

func TestParseFile(t *testing.T) {
	t.Parallel()

	t.Run("a normal file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "doc.xml")
		require.NoError(t, os.WriteFile(path, []byte(`<root><child>hi</child></root>`), 0o600))

		doc, err := helium.NewParser().ParseFile(t.Context(), path)
		require.NoError(t, err)
		require.NotNil(t, doc)

		abs, err := filepath.Abs(path)
		require.NoError(t, err)
		require.Equal(t, abs, doc.URL(), "document URL should be the absolute path")
	})

	t.Run("a missing file errors", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "does-not-exist.xml")

		_, err := helium.NewParser().ParseFile(t.Context(), path)
		require.Error(t, err, "parsing a missing file must error")
	})

	// ParseFile propagates a recovered
	// partial document alongside the error, matching Parse/ParseReader. A recover
	// parse must not silently discard the tree it built.
	t.Run("a recovered document is returned", func(t *testing.T) {
		// Malformed XML: mismatched close tag (same input as TestRecoverOnError).
		const input = `<?xml version="1.0"?>
<root>
  <child>text</chld>
</root>`

		dir := t.TempDir()
		path := filepath.Join(dir, "malformed.xml")
		require.NoError(t, os.WriteFile(path, []byte(input), 0o600))

		// Without recover: error and no document, exactly as before.
		doc, err := helium.NewParser().ParseFile(t.Context(), path)
		require.Error(t, err, "malformed file must error")
		require.Nil(t, doc, "without recover, ParseFile returns no document")

		// With recover: error AND the partial document, with its source URL set.
		doc, err = helium.NewParser().RecoverOnError(true).ParseFile(t.Context(), path)
		require.Error(t, err, "malformed file must still error under recover")
		require.NotNil(t, doc, "with recover, ParseFile must return the partial document")
		abs, absErr := filepath.Abs(path)
		require.NoError(t, absErr)
		require.Equal(t, abs, doc.URL(), "the recovered document keeps its source URL")
	})

	t.Run("a relative external entity is resolved", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "child.xml"), []byte("WORLD"), 0o600))

		main := `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ENTITY child SYSTEM "child.xml">
]>
<doc>&child;</doc>`
		mainPath := filepath.Join(dir, "main.xml")
		require.NoError(t, os.WriteFile(mainPath, []byte(main), 0o600))

		doc, err := helium.NewParser().BlockXXE(false).SubstituteEntities(true).FS(helium.PermissiveFS()).ParseFile(t.Context(), mainPath)
		require.NoError(t, err)
		require.NotNil(t, doc)

		var buf bytes.Buffer
		require.NoError(t, helium.NewWriter().WriteTo(&buf, doc))
		require.Contains(t, buf.String(), "WORLD",
			"relative external entity must resolve against the file's base URI")
	})
}

func TestParseMalformed(t *testing.T) {
	t.Parallel()

	// well-formedness error branches across the
	// parser. Each input is malformed and must surface an error.
	t.Run("documents", func(t *testing.T) {
		bad := []string{
			`<root>`,                         // unclosed root
			`<root></notroot>`,               // mismatched end tag
			`<root attr></root>`,             // attribute without value
			`<root attr=value></root>`,       // unquoted attribute value
			`<root>&undefinedentity;</root>`, // reference to undeclared entity
			`<root><![CDATA[ unterminated`,   // unterminated CDATA
			`<!-- unterminated comment`,      // unterminated comment
			`<root>&#xZZ;</root>`,            // invalid hex char ref
			`<root>&;</root>`,                // empty reference
			`<>`,                             // empty tag name
			`<root></root><second/>`,         // two root elements
		}
		for _, in := range bad {
			_, err := helium.NewParser().Parse(t.Context(), []byte(in))
			require.Error(t, err, "malformed input %q should error", in)
		}
	})

	t.Run("bad input", func(t *testing.T) {
		inputs := []string{
			`<?xml version="1.0"?>
<root foo="bar">
  <child>foo</chld>
</root>`,
			`<?xml version="abc">`,
			`<?xml varsion="1.0">`,
		}
		p := helium.NewParser()
		for _, input := range inputs {
			_, err := p.Parse(t.Context(), []byte(input))
			require.Error(t, err, "Parse should fail for '%s'", input)
		}
	})

	// feeds a battery of distinct malformed inputs, each
	// designed to drive a specific parser error branch (XML declaration version /
	// encoding / standalone parsing, PI target and delimiter rules, comment and
	// CDATA termination, QName / Name lexical errors). Every input must be rejected;
	// the value is in exercising the otherwise-unreached error returns.
	t.Run("parser branches", func(t *testing.T) {
		bad := []struct {
			name string
			src  string
		}{
			{"xml decl version unquoted", `<?xml version=1.0?><root/>`},
			{"xml decl bad standalone", `<?xml version="1.0" standalone="maybe"?><root/>`},
			{"xml decl encoding unquoted", `<?xml version="1.0" encoding=UTF-8?><root/>`},
			{"xml decl encoding bad first char", `<?xml version="1.0" encoding="1bad"?><root/>`},
			{"xml decl missing version", `<?xml encoding="UTF-8"?><root/>`},
			{"pi target named xml mid-document", `<root><?xml data?></root>`},
			{"pi missing space after target", `<root><?targetdata</root>`},
			{"pi unterminated", `<root><?target data </root>`},
			{"comment with double hyphen", `<root><!-- a -- b --></root>`},
			{"cdata unterminated", `<root><![CDATA[unterminated</root>`},
			{"bad qname trailing colon", `<root:></root:>`},
			{"name starts with digit", `<1root/>`},
			{"attribute missing equals", `<root attr "v"/>`},
			{"unterminated start tag", `<root attr="v"`},
			{"text with raw less-than via entity ok but bad amp", `<root>a & b</root>`},
		}

		for _, tc := range bad {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				_, err := helium.NewParser().Parse(context.Background(), []byte(tc.src))
				require.Error(t, err, "malformed input %q must be rejected", tc.src)
			})
		}
	})

	t.Run("a malformed comment", func(t *testing.T) {
		_, err := helium.NewParser().Parse(t.Context(), []byte("<A/><!---00\x10"))
		require.Error(t, err)
	})

	t.Run("a missing space between attributes", func(t *testing.T) {
		// XML §3.1 P40/P44: attributes in a start/empty-element tag must be
		// separated by whitespace ('(S Attribute)*'). Two attributes written back
		// to back with no intervening S are a fatal well-formedness error. This
		// covers W3C xml-suite cases sun/attlist10, sun/attlist11, oasis/p40fail1,
		// oasis/p44fail4 and xmltest/not-wf/sa/186.
		reject := []string{
			// STag (P40) and EmptyElemTag (P44), no DTD.
			`<doc att="val"att2="val2"></doc>`,
			`<doc att="val"att2="val2"/>`,
			// With an internal DTD declaring the attributes (sun attlist10/11, sa186).
			"<!DOCTYPE root [\n<!ELEMENT root ANY>\n<!ATTLIST root att1 CDATA #IMPLIED>\n<!ATTLIST root att2 CDATA #IMPLIED>\n]>\n<root att1=\"value1\"att2=\"value2\"></root>",
			"<!DOCTYPE root [\n<!ELEMENT root ANY>\n<!ATTLIST root att1 CDATA #IMPLIED>\n<!ATTLIST root att2 CDATA #IMPLIED>\n]>\n<root att1=\"value1\"att2=\"value2\"/>",
			"<!DOCTYPE a [\n<!ELEMENT a EMPTY>\n<!ATTLIST a b CDATA #IMPLIED d CDATA #IMPLIED>\n]>\n<a b=\"c\"d=\"e\"/>",
			// A missing space before a namespace-declaration attribute is also
			// caught (the namespace branch already enforced this; keep it covered).
			`<doc att="val"xmlns:p="urn:x"/>`,
			// Namespace-declaration attribute FIRST, then a regular attribute with no
			// separating space: the new regular-attribute check must fire after the
			// namespace attribute is consumed (proves the two checks compose).
			`<doc xmlns:p="urn:x"att="val"/>`,
		}
		for _, input := range reject {
			_, err := helium.NewParser().Parse(t.Context(), []byte(input))
			require.Error(t, err, "Parse should reject missing space between attributes in %q", input)
			require.ErrorIs(t, err, helium.ErrSpaceRequired, "should be a space-required error for %q", input)
		}

		// The well-formed counterparts — attributes separated by a space, a
		// newline, or any XML whitespace — must still parse cleanly (no
		// over-rejection). A single attribute and the two tag-close forms are
		// included to exercise the '>' / '/>' branches of the new check.
		accept := []string{
			`<doc att="val" att2="val2"/>`,
			`<doc att="val" att2="val2"></doc>`,
			"<doc att=\"val\"\natt2=\"val2\"/>",
			"<doc att=\"val\"\t att2=\"val2\"/>",
			"<doc att=\"val\"\ratt2=\"val2\"/>",
			`<doc att="val"/>`,
			`<doc att="val"></doc>`,
			`<doc/>`,
			`<doc att="val" ></doc>`,
			// A space between a namespace-declaration attribute and a following
			// regular attribute must still parse (well-formed composition).
			`<doc xmlns:p="urn:x" att="val"/>`,
		}
		for _, input := range accept {
			_, err := helium.NewParser().Parse(t.Context(), []byte(input))
			require.NoError(t, err, "Parse should accept space-separated attributes in %q", input)
		}
	})

	t.Run("a duplicate attribute", func(t *testing.T) {
		// XML 1.0 §3.1 well-formedness: a start tag may not have two
		// attributes with the same (qualified) name. These must be rejected
		// even when not validating.
		reject := []string{
			`<root a="1" a="2"/>`,
			`<root xmlns:p="urn:x" p:a="1" p:a="2"/>`,
			// Duplicate default namespace declarations on the same element,
			// including when one or both are empty (xmlns="").
			`<root xmlns="urn:x" xmlns="urn:y"/>`,
			`<root xmlns="urn:x" xmlns="urn:x"/>`,
			`<root xmlns="" xmlns="urn:x"/>`,
			`<root xmlns="" xmlns=""/>`,
			// Two attributes with different prefixes but the same expanded
			// name ({urn:x}a). Forbidden by Namespaces in XML.
			`<root xmlns:p="urn:x" xmlns:q="urn:x" p:a="1" q:a="2"/>`,
		}
		for _, input := range reject {
			_, err := helium.NewParser().Parse(t.Context(), []byte(input))
			require.Error(t, err, "Parse should reject duplicate attribute in %q", input)
		}

		// The same well-formedness violations must remain fatal even with
		// CleanNamespaces(true) (parseNsClean), which only suppresses redundant
		// ancestor redeclarations, never same-element duplicates.
		rejectClean := []string{
			`<root xmlns:p="urn:x" xmlns:p="urn:x"/>`,
			`<root xmlns="urn:x" xmlns="urn:x"/>`,
			`<root xmlns="" xmlns=""/>`,
			`<root xmlns="" xmlns="urn:x"/>`,
			// A child re-declaring a prefix already bound by an ancestor to the
			// same URI is the parseNsClean redundant-redeclaration case; a
			// SECOND same-element declaration must still be fatal even though the
			// first one is skipped as redundant (not pushed onto the ns stack).
			`<root xmlns="urn:x"><child xmlns="urn:x" xmlns="urn:x"/></root>`,
			`<root xmlns:p="urn:x"><child xmlns:p="urn:x" xmlns:p="urn:x"/></root>`,
		}
		for _, input := range rejectClean {
			_, err := helium.NewParser().CleanNamespaces(true).Parse(t.Context(), []byte(input))
			require.Error(t, err, "Parse with CleanNamespaces should reject duplicate ns decl in %q", input)
		}

		// Distinct qualified names must still parse, including the same local
		// name carried by different prefixes mapped to different URIs.
		accept := []string{
			`<root a="1" b="2"/>`,
			`<root xmlns:p="urn:x" xmlns:q="urn:y" p:a="1" q:a="2"/>`,
			// Unprefixed attributes are in no namespace; a default xmlns does
			// not put them in a namespace, so distinct local names are fine.
			`<root xmlns="urn:x" a="1" b="2"/>`,
		}
		for _, input := range accept {
			_, err := helium.NewParser().Parse(t.Context(), []byte(input))
			require.NoError(t, err, "Parse should accept distinct attributes in %q", input)
		}

		// A single child redeclaration of an ancestor binding (no same-element
		// duplicate) is a legitimate parseNsClean redundant redeclaration and
		// must still parse.
		acceptClean := []string{
			`<root xmlns="urn:x"><child xmlns="urn:x"/></root>`,
			`<root xmlns:p="urn:x"><child xmlns:p="urn:x"/></root>`,
		}
		for _, input := range acceptClean {
			_, err := helium.NewParser().CleanNamespaces(true).Parse(t.Context(), []byte(input))
			require.NoError(t, err, "Parse with CleanNamespaces should accept redundant redecl in %q", input)
		}
	})

	t.Run("a malformed DTD default namespace", func(t *testing.T) {
		// A namespace declaration supplied as a DTD <!ATTLIST> default must be
		// subject to the same Namespaces in XML validity checks an inline xmlns
		// declaration gets. These all bind a prefix illegally and must be
		// rejected even though the binding is never written on the element.
		reject := []string{
			// reuse of the reserved xmlns namespace name
			`<?xml version="1.0"?>
<!DOCTYPE root [
<!ELEMENT root EMPTY>
<!ATTLIST root xmlns:foo CDATA #FIXED "http://www.w3.org/2000/xmlns/">
]>
<root/>`,
			// reserved xml prefix bound to the wrong URI
			`<?xml version="1.0"?>
<!DOCTYPE root [
<!ELEMENT root EMPTY>
<!ATTLIST root xmlns:xml CDATA #FIXED "urn:wrong">
]>
<root/>`,
		}
		p := helium.NewParser()
		for _, input := range reject {
			_, err := p.Parse(t.Context(), []byte(input))
			require.Error(t, err, "Parse should reject malformed DTD-defaulted namespace: %s", input)
		}

		// A well-formed DTD-defaulted xmlns:xml mapped to its canonical URI is
		// accepted (and not pushed as a real binding).
		const ok = `<?xml version="1.0"?>
<!DOCTYPE root [
<!ELEMENT root EMPTY>
<!ATTLIST root xmlns:xml CDATA #FIXED "http://www.w3.org/XML/1998/namespace">
]>
<root/>`
		_, err := p.Parse(t.Context(), []byte(ok))
		require.NoError(t, err, "canonical xml prefix binding must be accepted")
	})
}

func TestParseName(t *testing.T) {
	t.Parallel()

	t.Run("an NCName with an invalid start rune", func(t *testing.T) {
		xml := []byte("<root 1a=\"v\"/>")
		p := helium.NewParser()
		_, err := p.Parse(t.Context(), xml)
		require.Error(t, err)
	})

	t.Run("an NCName with invalid UTF-8 in a continuation", func(t *testing.T) {
		xml := []byte("<root at\xffr=\"v\"/>")
		p := helium.NewParser()
		_, err := p.Parse(t.Context(), xml)
		require.Error(t, err)
	})

	t.Run("a name with invalid UTF-8 in a continuation", func(t *testing.T) {
		xml := []byte("<ro\xffoot/>")
		p := helium.NewParser()
		_, err := p.Parse(t.Context(), xml)
		require.Error(t, err)
	})
}

func TestParseNamespace(t *testing.T) {
	t.Parallel()

	t.Run("namespaces", func(t *testing.T) {
		const input = `<?xml version="1.0"?>
<helium:root xmlns:helium="https://github.com/lestrrat-go/helium">
  <helium:child>foo</helium:child>
</helium:root>`
		p := helium.NewParser()
		doc, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err, "Parse should succeed for '%s'", input)

		root := doc.DocumentElement()
		require.NotNil(t, root)
		require.Equal(t, "https://github.com/lestrrat-go/helium", root.URI())

		const attrInput = `<?xml version="1.0"?>
<root xmlns:x="urn:test" x:attr="value"/>`
		doc, err = p.Parse(t.Context(), []byte(attrInput))
		require.NoError(t, err)

		root = doc.DocumentElement()
		require.NotNil(t, root)
		attr := root.GetAttributeNodeNS("attr", "urn:test")
		require.NotNil(t, attr)
		require.Equal(t, "x", attr.Prefix())
		require.Equal(t, "urn:test", attr.URI())
		require.Equal(t, "value", attr.Value())
	})

	// parses namespaced elements and attributes.
	t.Run("namespaced attributes", func(t *testing.T) {
		const src = `<root xmlns="urn:default" xmlns:p="urn:p" p:attr="v" plain="w">` +
			`<p:child/><plain/></root>`
		doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
		require.NoError(t, err)

		out, err := helium.WriteString(doc)
		require.NoError(t, err)
		require.Contains(t, out, `p:attr="v"`)
		require.Contains(t, out, `xmlns:p="urn:p"`)
	})
}

func TestParseMisc(t *testing.T) {
	t.Parallel()

	t.Run("misc", func(t *testing.T) {
		const decl = `<?xml version="1.0"?>` + "\n"
		const content = `<root />`
		inputs := []string{
			decl + `<?xml-stylesheet type="text/xsl" href="style.xsl"?>` + content,
			decl + `<?xml-stylesheet type="text/css" href="style.css"?>` + content,
		}

		for _, input := range inputs {
			p := helium.NewParser()
			doc, err := p.Parse(t.Context(), []byte(input))
			require.NoError(t, err, "Parse should succeed for '%s'", input)

			// XXX Not sure if this is right, but I'm going to assume it's ok
			// to have a DTD in the child list
			n := doc.FirstChild()
		LOOP:
			for {
				t.Logf("%#v", n)
				if n == nil {
					t.Errorf("Could not find ProcessingInstruction node")
					return
				}

				switch n.Type() {
				case helium.ProcessingInstructionNode:
					require.IsType(t, &helium.ProcessingInstruction{}, n, "First child should be a processing instruction")

					require.IsType(t, &helium.Element{}, n.NextSibling(), "NextSibling of PI should be Element node")
					break LOOP
				}
				n = n.NextSibling()
			}
		}
	})

	// parses PIs and comments in the prolog,
	// content, and epilog positions.
	t.Run("processing instructions and comments", func(t *testing.T) {
		const src = `<?xml version="1.0"?>
<?pi-prolog data?>
<!-- prolog comment -->
<root>
  <?pi-content more?>
  <!-- content comment -->
  text
</root>
<!-- epilog comment -->
<?pi-epilog x?>`

		doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
		require.NoError(t, err)

		out, err := helium.WriteString(doc)
		require.NoError(t, err)
		require.Contains(t, out, "<?pi-prolog")
		require.Contains(t, out, "<!-- prolog comment -->")
	})
}

func TestCDATA(t *testing.T) {
	t.Parallel()

	// parses CDATA sections including the tricky ]]> boundary.
	t.Run("sections", func(t *testing.T) {
		const src = `<root><![CDATA[ raw <tag> & ]]> normal text <child/></root>`
		doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
		require.NoError(t, err)

		out, err := helium.WriteString(doc)
		require.NoError(t, err)
		require.Contains(t, out, "<![CDATA[")
	})

	t.Run("merged sections", func(t *testing.T) {
		const input = `<?xml version="1.0"?>
<root><![CDATA[hello]]></root>`

		// Without MergeCDATA: tree should have a CDATA node
		p1 := helium.NewParser()
		doc1, err := p1.Parse(t.Context(), []byte(input))
		require.NoError(t, err, "Parse should succeed")
		root1 := findDocumentElement(doc1)
		require.NotNil(t, root1)
		child1 := root1.FirstChild()
		require.NotNil(t, child1)
		require.Equal(t, helium.CDATASectionNode, child1.Type(), "without MergeCDATA, should be CDATA node")

		// With MergeCDATA: CDATA should be delivered as text
		p2 := helium.NewParser().MergeCDATA(true)
		doc2, err := p2.Parse(t.Context(), []byte(input))
		require.NoError(t, err, "Parse should succeed")
		root2 := findDocumentElement(doc2)
		require.NotNil(t, root2)
		child2 := root2.FirstChild()
		require.NotNil(t, child2)
		require.Equal(t, helium.TextNode, child2.Type(), "with MergeCDATA, CDATA should be a text node")
		require.Equal(t, "hello", string(child2.Content()))
	})
}

func TestParserOptions(t *testing.T) {
	t.Parallel()

	// every boolean parser option setter with both
	// true and false (so both the Set and Clear branches run) plus the scalar/object
	// setters, then performs a parse to confirm the configured parser still works.
	t.Run("setters", func(t *testing.T) {
		p := helium.NewParser().
			RecoverOnError(true).RecoverOnError(false).
			SubstituteEntities(true).SubstituteEntities(false).
			LoadExternalDTD(true).LoadExternalDTD(false).
			DefaultDTDAttributes(true).DefaultDTDAttributes(false).
			ValidateDTD(true).ValidateDTD(false).
			SuppressErrors(true).SuppressErrors(false).
			SuppressWarnings(true).SuppressWarnings(false).
			PedanticErrors(true).PedanticErrors(false).
			StripBlanks(true).StripBlanks(false).
			AllowNetwork(true).AllowNetwork(false).
			CleanNamespaces(true).CleanNamespaces(false).
			MergeCDATA(true).MergeCDATA(false).
			FixBaseURIs(true).FixBaseURIs(false).
			MaxNameLength(-1).MaxNameLength(0).
			MaxEntityAmplification(-1).MaxEntityAmplification(0).
			MaxContentModelDepth(-1).MaxContentModelDepth(0).
			IgnoreEncoding(true).IgnoreEncoding(false).
			BlockXXE(true).BlockXXE(false).
			SkipIDs(true).SkipIDs(false).
			LenientXMLDecl(true).LenientXMLDecl(false).
			CharBufferSize(8192).
			MaxDepth(256).
			MaxExternalDTDBytes(1 << 20).
			Catalog(nopCatalog{}).
			BaseURI("http://example.com/base.xml")

		doc, err := p.Parse(t.Context(), []byte(`<?xml version="1.0"?><root><child>text</child></root>`))
		require.NoError(t, err, "a fully-configured parser parses a simple document")
		require.NotNil(t, doc.DocumentElement())
	})

	t.Run("pedantic", func(t *testing.T) {
		// Pedantic mode requires absolute URIs in namespace declarations
		const input = `<?xml version="1.0"?>
<root xmlns:foo="relative/uri">
  <foo:child>text</foo:child>
</root>`

		// Without pedantic: should succeed
		p1 := helium.NewParser()
		_, err := p1.Parse(t.Context(), []byte(input))
		require.NoError(t, err, "without pedantic, relative URI should be accepted")

		// With pedantic: should fail (relative URI)
		p2 := helium.NewParser().PedanticErrors(true)
		_, err = p2.Parse(t.Context(), []byte(input))
		require.Error(t, err, "with pedantic, relative URI should be rejected")
	})

	t.Run("skip IDs", func(t *testing.T) {
		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc EMPTY>
  <!ATTLIST doc id ID #IMPLIED>
  <!ATTLIST doc name CDATA #IMPLIED>
]>
<doc id="x1" name="n1"/>`

		p := helium.NewParser().LoadExternalDTD(true).DefaultDTDAttributes(true).SkipIDs(true)
		doc, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err)

		require.Nil(t, doc.GetElementByID("x1"), "ID should not be interned when SkipIDs is set")
		root := doc.DocumentElement()
		require.NotNil(t, root)
		name, ok := root.GetAttribute("name")
		require.True(t, ok, "non-ID attributes should still be available")
		require.Equal(t, "n1", name)
	})

	// confirms SAXHandler(nil) restores the
	// default TreeBuilder rather than leaving the parser handler-less. A nil handler
	// otherwise builds no tree, so Parse would report success with a nil *Document —
	// a silent no-output result for a caller passing an optional handler straight
	// through. This mirrors Parser.FS(nil), which likewise restores its default.
	t.Run("a nil SAX handler restores the default", func(t *testing.T) {
		const (
			src      = `<?xml version="1.0"?><root><child>text</child></root>`
			rootElem = "root"
		)

		t.Run("nil handler builds the tree", func(t *testing.T) {
			t.Parallel()

			doc, err := helium.NewParser().SAXHandler(nil).Parse(t.Context(), []byte(src))
			require.NoError(t, err)
			require.NotNil(t, doc, "SAXHandler(nil) must not yield a nil document with a nil error")
			require.NotNil(t, doc.DocumentElement())
			require.Equal(t, rootElem, doc.DocumentElement().Name())
		})

		t.Run("nil clears a previously set handler", func(t *testing.T) {
			t.Parallel()

			var seen []string
			p := helium.NewParser().SAXHandler(startElementRecorder(&seen))
			doc, err := p.SAXHandler(nil).Parse(t.Context(), []byte(src))
			require.NoError(t, err)
			require.NotNil(t, doc, "the last SAXHandler call wins, so nil restores the default builder")
			require.NotNil(t, doc.DocumentElement())
			require.Empty(t, seen, "the replaced handler must receive no events")
		})

		t.Run("an explicit handler still replaces the default", func(t *testing.T) {
			t.Parallel()

			var seen []string
			p := helium.NewParser().SAXHandler(nil).SAXHandler(startElementRecorder(&seen))
			_, err := p.Parse(t.Context(), []byte(src))
			require.NoError(t, err)
			require.Equal(t, []string{rootElem, "child"}, seen)
		})
	})

	// a zero-value helium.Parser behaves
	// identically to one returned by helium.NewParser() — same secure defaults,
	// same parse results, and usable both directly and as the head of an
	// option-method chain, without panicking on its nil config.
	t.Run("the zero-value parser", func(t *testing.T) {
		serialize := func(t *testing.T, doc *helium.Document) string {
			t.Helper()
			var buf bytes.Buffer
			require.NoError(t, helium.NewWriter().WriteTo(&buf, doc))
			return buf.String()
		}

		t.Run("Parse matches NewParser", func(t *testing.T) {
			const src = `<root att="v"><child>text</child><child/></root>`
			var zero helium.Parser
			zdoc, err := zero.Parse(context.Background(), []byte(src))
			require.NoError(t, err)
			ndoc, err := helium.NewParser().Parse(context.Background(), []byte(src))
			require.NoError(t, err)
			require.Equal(t, serialize(t, ndoc), serialize(t, zdoc))
		})

		t.Run("ParseReader does not panic", func(t *testing.T) {
			var zero helium.Parser
			doc, err := zero.ParseReader(context.Background(), strings.NewReader(`<r><a/></r>`))
			require.NoError(t, err)
			require.NotNil(t, doc)
		})

		t.Run("option chaining works and matches NewParser", func(t *testing.T) {
			const src = `<r>  <a/>  </r>`
			var zero helium.Parser
			zdoc, err := zero.StripBlanks(true).Parse(context.Background(), []byte(src))
			require.NoError(t, err)
			ndoc, err := helium.NewParser().StripBlanks(true).Parse(context.Background(), []byte(src))
			require.NoError(t, err)
			require.Equal(t, serialize(t, ndoc), serialize(t, zdoc))
		})

		t.Run("secure defaults: element depth cap matches NewParser", func(t *testing.T) {
			// NewParser caps nesting at 256; a zero-value Parser must apply the same
			// cap, so a document nested well past it fails for both, identically.
			var b strings.Builder
			const depth = 400
			for range depth {
				b.WriteString("<a>")
			}
			for range depth {
				b.WriteString("</a>")
			}
			src := []byte(b.String())

			var zero helium.Parser
			_, zerr := zero.Parse(context.Background(), src)
			_, nerr := helium.NewParser().Parse(context.Background(), src)
			require.Error(t, nerr, "NewParser should reject nesting past its depth cap")
			require.Error(t, zerr, "zero-value Parser must apply the same depth cap")
		})
	})
}

func TestRecoverOnError(t *testing.T) {
	t.Parallel()

	t.Run("recovery", func(t *testing.T) {
		// Malformed XML: mismatched close tag
		const input = `<?xml version="1.0"?>
<root>
  <child>text</chld>
</root>`

		// Without RecoverOnError: error, no document
		p1 := helium.NewParser()
		doc1, err := p1.Parse(t.Context(), []byte(input))
		require.Error(t, err, "malformed XML should fail")
		require.Nil(t, doc1, "without recover, no document returned")

		// With RecoverOnError: error, but partial document returned
		p2 := helium.NewParser().RecoverOnError(true)
		doc2, err := p2.Parse(t.Context(), []byte(input))
		require.Error(t, err, "malformed XML should still return error")
		require.NotNil(t, doc2, "with recover, partial document should be returned")
	})

	// the recover path: a malformed document returns
	// both a (partial) document and an error.
	t.Run("a partial document", func(t *testing.T) {
		const src = `<root><a>text</a><b></root>`
		doc, err := helium.NewParser().RecoverOnError(true).Parse(t.Context(), []byte(src))
		// With recovery the parser returns a partial document; an error may or may
		// not be reported depending on how far recovery proceeds.
		_ = err
		require.NotNil(t, doc)
	})
}

func TestEntityBoundary(t *testing.T) {
	t.Parallel()

	t.Run("entity boundary", func(t *testing.T) {
		t.Run("element decl", func(t *testing.T) {
			t.Parallel()

			// PE starts the element declaration but the closing '>' is in the main DTD.
			// This crosses an entity boundary -> parse error (syntax or boundary).
			const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ENTITY % start "<!ELEMENT doc EMPTY">
  %start;>
]>
<doc/>`

			p := helium.NewParser().LoadExternalDTD(true)
			_, err := p.Parse(t.Context(), []byte(input))
			require.Error(t, err, "boundary-violating PE should cause a parse error")
		})

		t.Run("attribute list decl", func(t *testing.T) {
			t.Parallel()

			// PE starts the ATTLIST declaration but the closing '>' is in the main DTD.
			const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc EMPTY>
  <!ENTITY % start "<!ATTLIST doc attr CDATA #IMPLIED">
  %start;>
]>
<doc/>`

			p := helium.NewParser().LoadExternalDTD(true)
			_, err := p.Parse(t.Context(), []byte(input))
			require.Error(t, err, "boundary-violating PE should cause a parse error")
		})

		t.Run("well-nested PE", func(t *testing.T) {
			t.Parallel()

			// PE expands to a complete declaration -- no boundary violation.
			const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ENTITY % decl "<!ELEMENT doc EMPTY>">
  %decl;
]>
<doc/>`

			p := helium.NewParser().LoadExternalDTD(true)
			doc, err := p.Parse(t.Context(), []byte(input))
			require.NoError(t, err)
			require.NotNil(t, doc)
		})
	})
}

func TestParseExternalEntity(t *testing.T) {
	t.Parallel()

	t.Run("external entities", func(t *testing.T) {
		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ENTITY ext SYSTEM "ext.xml">
]>
<doc>&ext;</doc>`

		// The external entity is declared in the internal subset and its content is
		// served through the configured FS, exercising the default resolution path.
		fsys := fstest.MapFS{
			"ext.xml": &fstest.MapFile{Data: []byte("<inner>hello</inner>")},
		}

		p := helium.NewParser().BlockXXE(false).SubstituteEntities(true).FS(fsys)
		doc, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err, "Parse with external entity should succeed")
		require.NotNil(t, doc, "external entity parse should produce a document")

		var buf bytes.Buffer
		require.NoError(t, helium.NewWriter().WriteTo(&buf, doc))
		out := buf.String()
		require.Contains(t, out, "<inner", "external entity element should be expanded into the document")
		require.Contains(t, out, ">hello</inner>", "external entity content should be expanded into the document")
	})

	t.Run("a malformed encoding", func(t *testing.T) {
		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ENTITY ext SYSTEM "ext.xml">
]>
<doc>&ext;</doc>`

		// External entity bytes: UTF-16BE BOM, then "<a>" and an unpaired high
		// surrogate (0xD800) before "</a>". The decoder would silently substitute
		// U+FFFD for the surrogate; the parser must instead treat it as fatal,
		// matching the document-level decode-error gate.
		utf16be := func(s string) []byte {
			b := make([]byte, 0, len(s)*2)
			for _, r := range s {
				b = append(b, byte(r>>8), byte(r))
			}
			return b
		}
		ent := []byte{0xFE, 0xFF} // BOM
		ent = append(ent, utf16be("<a>")...)
		ent = append(ent, 0xD8, 0x00) // unpaired high surrogate
		ent = append(ent, utf16be("</a>")...)

		fsys := fstest.MapFS{"ext.xml": &fstest.MapFile{Data: ent}}

		p := helium.NewParser().BlockXXE(false).SubstituteEntities(true).FS(fsys)
		_, err := p.Parse(t.Context(), []byte(input))
		require.Error(t, err, "malformed UTF-16 external entity must fail rather than inserting U+FFFD")
	})

	t.Run("a valid encoding", func(t *testing.T) {
		const input = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ENTITY ext SYSTEM "ext.xml">
]>
<doc>&ext;</doc>`

		// A well-formed UTF-16BE external entity (BOM + "<a/>") must still load.
		utf16be := func(s string) []byte {
			b := make([]byte, 0, len(s)*2)
			for _, r := range s {
				b = append(b, byte(r>>8), byte(r))
			}
			return b
		}
		ent := append([]byte{0xFE, 0xFF}, utf16be("<a/>")...)

		fsys := fstest.MapFS{"ext.xml": &fstest.MapFile{Data: ent}}

		p := helium.NewParser().SubstituteEntities(true).FS(fsys)
		_, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err, "well-formed UTF-16 external entity must load")
	})
}
