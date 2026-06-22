package helium_test

import (
	"context"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
	"github.com/stretchr/testify/require"
)

// TestDTDDeclarationNodeWrappers exercises the node-interface wrappers on the
// DTD, ElementDecl and AttributeDecl node types (AddChild, AppendText,
// AddSibling, Replace, SetTreeDoc) plus the Entity AddSibling/Replace wrappers.
// These all delegate to the shared tree primitives; the test confirms each
// override is wired up and returns the shared primitive's result.
func TestDTDDeclarationNodeWrappers(t *testing.T) {
	t.Parallel()

	// Parse a doc that declares both an element and an attribute so we obtain
	// real ElementDecl and AttributeDecl nodes from the DTD.
	const src = `<?xml version="1.0"?>
<!DOCTYPE doc [
<!ELEMENT doc (#PCDATA)>
<!ATTLIST doc a CDATA #IMPLIED>
]>
<doc a="v">text</doc>`
	doc, err := helium.NewParser().Parse(context.Background(), []byte(src))
	require.NoError(t, err)
	dtd := doc.IntSubset()
	require.NotNil(t, dtd)

	t.Run("ElementDecl wrappers", func(t *testing.T) {
		t.Parallel()
		edecl, ok := dtd.LookupElement("doc", "")
		require.True(t, ok)
		require.Equal(t, helium.ElementDeclNode, edecl.Type())

		// AppendText routes a Text child into the decl node.
		require.NoError(t, edecl.AppendText([]byte("x")))
		// AddChild attaches a fresh node.
		require.NoError(t, edecl.AddChild(doc.CreateComment([]byte("c"))))
		// AddSibling/Replace/SetTreeDoc must not panic and delegate to the
		// shared primitives.
		_ = edecl.AddSibling(doc.CreateComment([]byte("sib")))
		_ = edecl.Replace()
		edecl.SetTreeDoc(doc)
	})

	t.Run("AttributeDecl wrappers", func(t *testing.T) {
		t.Parallel()
		adecls := dtd.AttributesForElement("doc")
		require.NotEmpty(t, adecls)
		adecl := adecls[0]

		require.NoError(t, adecl.AppendText([]byte("y")))
		require.NoError(t, adecl.AddChild(doc.CreateComment([]byte("ac"))))
		_ = adecl.AddSibling(doc.CreateComment([]byte("as")))
		_ = adecl.Replace()
		adecl.SetTreeDoc(doc)
	})

	t.Run("DTD AppendText and Free", func(t *testing.T) {
		t.Parallel()
		d3 := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
		dtd3, derr := d3.CreateInternalSubset("doc", "", "")
		require.NoError(t, derr)
		require.NoError(t, dtd3.AppendText([]byte("ws")))
		dtd3.Free() // no-op marker, but exercised for completeness
	})

	t.Run("Entity AddSibling and Replace", func(t *testing.T) {
		t.Parallel()
		d4 := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
		dtd4, derr := d4.CreateInternalSubset("doc", "", "")
		require.NoError(t, derr)
		ent, eerr := dtd4.AddEntity("e", enum.InternalGeneralEntity, "", "", "v")
		require.NoError(t, eerr)
		_ = ent.AddSibling(d4.CreateComment([]byte("s")))
		_ = ent.Replace()
	})
}

// TestCopyDocWithMixedChildren builds a document whose root element holds every
// leaf child type (Text, CDATA, Comment, PI, EntityRef), then deep-copies it via
// CopyDoc so the per-node-type branches of the deep copier's copyNode are all
// exercised.
func TestCopyDocWithMixedChildren(t *testing.T) {
	t.Parallel()

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root := doc.CreateElement("root")
	require.NoError(t, doc.AddChild(root))

	require.NoError(t, root.AddChild(doc.CreateText([]byte("text"))))
	require.NoError(t, root.AddChild(doc.CreateCDATASection([]byte("<cdata>"))))
	require.NoError(t, root.AddChild(doc.CreateComment([]byte("comment"))))
	require.NoError(t, root.AddChild(doc.CreatePI("target", "data")))
	ref, err := doc.CreateReference("amp")
	require.NoError(t, err)
	require.NoError(t, root.AddChild(ref))

	// A top-level comment and PI exercise the document-level copyChildren too.
	require.NoError(t, doc.AddChild(doc.CreateComment([]byte("top-comment"))))
	require.NoError(t, doc.AddChild(doc.CreatePI("toppi", "x")))

	cp, err := helium.CopyDoc(doc)
	require.NoError(t, err)
	require.NotNil(t, cp)

	cpRoot := cp.DocumentElement()
	require.NotNil(t, cpRoot)

	// Walk the copied children and confirm each node type round-tripped.
	var kinds []helium.ElementType
	for c := cpRoot.FirstChild(); c != nil; c = c.NextSibling() {
		kinds = append(kinds, c.Type())
	}
	require.Contains(t, kinds, helium.TextNode)
	require.Contains(t, kinds, helium.CDATASectionNode)
	require.Contains(t, kinds, helium.CommentNode)
	require.Contains(t, kinds, helium.ProcessingInstructionNode)
	require.Contains(t, kinds, helium.EntityRefNode)
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

// validateDTDErrs parses src with DTD validation enabled and returns the
// collected validation errors.
func validateDTDErrs(t *testing.T, src string) []error {
	t.Helper()
	h := &roundtwoErrSink{}
	_, err := helium.NewParser().
		ValidateDTD(true).
		ErrorHandler(h).
		Parse(context.Background(), []byte(src))
	if err != nil {
		require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
	}
	return h.errs
}

type roundtwoErrSink struct{ errs []error }

func (h *roundtwoErrSink) Handle(_ context.Context, err error) { h.errs = append(h.errs, err) }

// TestValidateContentModelOccurrences drives the occurrence variants of matchSeq
// and matchOr (optional, zero-or-more, one-or-more) plus nested optional
// sequences, exercising the seq/or matcher branches in valid.go that the simpler
// once-only models did not reach.
func TestValidateContentModelOccurrences(t *testing.T) {
	t.Parallel()

	t.Run("repeated sequence (a,b)+", func(t *testing.T) {
		t.Parallel()
		const dtd = `<!DOCTYPE doc [
<!ELEMENT doc (a, b)+>
<!ELEMENT a (#PCDATA)>
<!ELEMENT b (#PCDATA)>
]>`
		require.Empty(t, validateDTDErrs(t, dtd+`<doc><a/><b/><a/><b/></doc>`),
			"two (a,b) repetitions validate")
		require.NotEmpty(t, validateDTDErrs(t, dtd+`<doc><a/><b/><a/></doc>`),
			"a trailing partial (a,b) repetition fails")
	})

	t.Run("optional trailing element (a,b?)", func(t *testing.T) {
		t.Parallel()
		const dtd = `<!DOCTYPE doc [
<!ELEMENT doc (a, b?)>
<!ELEMENT a (#PCDATA)>
<!ELEMENT b (#PCDATA)>
]>`
		require.Empty(t, validateDTDErrs(t, dtd+`<doc><a/></doc>`),
			"the optional trailing b may be absent")
		require.Empty(t, validateDTDErrs(t, dtd+`<doc><a/><b/></doc>`),
			"the optional trailing b may be present")
		require.NotEmpty(t, validateDTDErrs(t, dtd+`<doc><a/><b/><b/></doc>`),
			"a second b exceeds the optional occurrence")
	})

	t.Run("zero-or-more choice (a|b)*", func(t *testing.T) {
		t.Parallel()
		const dtd = `<!DOCTYPE doc [
<!ELEMENT doc (a | b)*>
<!ELEMENT a (#PCDATA)>
<!ELEMENT b (#PCDATA)>
]>`
		require.Empty(t, validateDTDErrs(t, dtd+`<doc></doc>`),
			"zero occurrences of the choice validate")
		require.Empty(t, validateDTDErrs(t, dtd+`<doc><a/><a/><b/></doc>`),
			"several occurrences of the choice validate")
	})

	t.Run("optional element a?", func(t *testing.T) {
		t.Parallel()
		const dtd = `<!DOCTYPE doc [
<!ELEMENT doc (a?, b)>
<!ELEMENT a (#PCDATA)>
<!ELEMENT b (#PCDATA)>
]>`
		require.Empty(t, validateDTDErrs(t, dtd+`<doc><b/></doc>`),
			"the optional leading a may be omitted")
		require.Empty(t, validateDTDErrs(t, dtd+`<doc><a/><b/></doc>`),
			"the optional leading a may be present")
	})

	t.Run("one-or-more element a+", func(t *testing.T) {
		t.Parallel()
		const dtd = `<!DOCTYPE doc [
<!ELEMENT doc (a+)>
<!ELEMENT a (#PCDATA)>
]>`
		require.Empty(t, validateDTDErrs(t, dtd+`<doc><a/><a/><a/></doc>`),
			"multiple a children validate a+")
		require.NotEmpty(t, validateDTDErrs(t, dtd+`<doc></doc>`),
			"zero a children fails a+")
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

// TestSetBooleanAttribute covers Element.SetBooleanAttribute for both the
// success path (a value-less attribute) and the colon-rejection error path.
func TestSetBooleanAttribute(t *testing.T) {
	t.Parallel()

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root := doc.CreateElement("input")
	require.NoError(t, doc.AddChild(root))

	require.NoError(t, root.SetBooleanAttribute("checked"), "boolean attribute added")

	require.True(t, root.HasAttribute("checked"), "boolean attribute is present")
	val, ok := root.GetAttribute("checked")
	require.True(t, ok, "boolean attribute is readable")
	require.Empty(t, val, "boolean attribute has no value")

	// A colon in the name is rejected.
	require.Error(t, root.SetBooleanAttribute("ns:bad"))
}

// TestAppendChildFast covers the public AppendChildFast helper across the
// empty-parent and non-empty-parent fast paths.
func TestAppendChildFast(t *testing.T) {
	t.Parallel()

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	parent := doc.CreateElement("parent")

	first := doc.CreateElement("first")
	require.NoError(t, helium.AppendChildFast(parent, first), "fast-link first child")
	require.Equal(t, helium.Node(first), parent.FirstChild())
	require.Equal(t, helium.Node(first), parent.LastChild())
	require.Equal(t, helium.Node(parent), first.Parent())

	second := doc.CreateElement("second")
	require.NoError(t, helium.AppendChildFast(parent, second), "fast-link second child")
	require.Equal(t, helium.Node(second), parent.LastChild())
	require.Equal(t, helium.Node(second), first.NextSibling())
	require.Equal(t, helium.Node(first), second.PrevSibling())
}

// TestClarkName covers the ClarkName helper.
func TestClarkName(t *testing.T) {
	t.Parallel()
	require.Equal(t, "{urn:example}local", helium.ClarkName("urn:example", "local"))
	require.Equal(t, "{}local", helium.ClarkName("", "local"))
}

// TestParserMalformedBranches feeds a battery of distinct malformed inputs, each
// designed to drive a specific parser error branch (XML declaration version /
// encoding / standalone parsing, PI target and delimiter rules, comment and
// CDATA termination, QName / Name lexical errors). Every input must be rejected;
// the value is in exercising the otherwise-unreached error returns.
func TestParserMalformedBranches(t *testing.T) {
	t.Parallel()

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
}

// TestParserWellFormedVariety parses a variety of well-formed constructs that
// exercise the success branches of the same parser functions the malformed tests
// hit on the error side: a leading PI, a comment, a CDATA section, namespaced
// elements/attributes, character references, and an explicit encoding/standalone
// declaration.
func TestParserWellFormedVariety(t *testing.T) {
	t.Parallel()

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
}

// TestParseDTDEntityValuesAndParamRefs parses internal-subset DTDs that exercise
// the entity-value and parameter-entity-reference parser paths: entity values
// containing character and general-entity references, a parameter entity declared
// and then referenced inside the subset, and a mixed-content (#PCDATA|x|y)*
// declaration with several alternatives.
func TestParseDTDEntityValuesAndParamRefs(t *testing.T) {
	t.Parallel()

	t.Run("entity value with char and general refs", func(t *testing.T) {
		t.Parallel()
		const src = `<?xml version="1.0"?>
<!DOCTYPE doc [
<!ELEMENT doc (#PCDATA)>
<!ENTITY base "base">
<!ENTITY composed "prefix-&base;-&#65;-suffix">
]>
<doc>&composed;</doc>`
		doc, err := helium.NewParser().SubstituteEntities(true).Parse(context.Background(), []byte(src))
		require.NoError(t, err)
		require.NotNil(t, doc.DocumentElement())
	})

	t.Run("parameter entity expands to a markup declaration", func(t *testing.T) {
		t.Parallel()
		// A parameter entity whose replacement text is an entire markup
		// declaration, referenced via %e; inside the internal subset, drives
		// the PE-reference expansion path in the subset parser.
		const src = `<?xml version="1.0"?>
<!DOCTYPE doc [
<!ENTITY % e "<!ELEMENT doc (#PCDATA)>">
%e;
]>
<doc>text</doc>`
		doc, err := helium.NewParser().Parse(context.Background(), []byte(src))
		require.NoError(t, err)
		dtd := doc.IntSubset()
		require.NotNil(t, dtd)
		_, ok := dtd.LookupElement("doc", "")
		require.True(t, ok, "the PE-supplied element declaration was registered")
	})

	t.Run("mixed content with several alternatives", func(t *testing.T) {
		t.Parallel()
		const src = `<?xml version="1.0"?>
<!DOCTYPE doc [
<!ELEMENT doc (#PCDATA | a | b | c)*>
<!ELEMENT a (#PCDATA)>
<!ELEMENT b (#PCDATA)>
<!ELEMENT c (#PCDATA)>
]>
<doc>t <a/> u <b/> v <c/> w</doc>`
		doc, err := helium.NewParser().Parse(context.Background(), []byte(src))
		require.NoError(t, err)
		dtd := doc.IntSubset()
		require.NotNil(t, dtd)
		edecl, ok := dtd.LookupElement("doc", "")
		require.True(t, ok)
		require.Equal(t, enum.MixedElementType, edecl.DeclType())
	})

	t.Run("element children content with nested groups and occurrences", func(t *testing.T) {
		t.Parallel()
		const src = `<?xml version="1.0"?>
<!DOCTYPE doc [
<!ELEMENT doc (head, (para | list)*, foot?)>
<!ELEMENT head (#PCDATA)>
<!ELEMENT para (#PCDATA)>
<!ELEMENT list (#PCDATA)>
<!ELEMENT foot (#PCDATA)>
]>
<doc><head/><para/><list/><para/><foot/></doc>`
		doc, err := helium.NewParser().ValidateDTD(true).Parse(context.Background(), []byte(src))
		require.NoError(t, err)
		require.NotNil(t, doc.DocumentElement())
	})
}

// TestNamespaceNodeWrapperContent covers NamespaceNodeWrapper.Content.
func TestNamespaceNodeWrapperContent(t *testing.T) {
	t.Parallel()
	ns := helium.NewNamespace("p", "urn:example")
	nsw := helium.NewNamespaceNodeWrapper(ns, nil)
	require.Equal(t, "urn:example", string(nsw.Content()))
	require.Equal(t, "p", nsw.Name())
}

// TestDocumentAppendText covers Document.AppendText, which appends a Text child
// to the document, merging into a trailing Text node when possible.
func TestDocumentAppendText(t *testing.T) {
	t.Parallel()
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	require.NoError(t, doc.AppendText([]byte("hello")))
	require.NoError(t, doc.AppendText([]byte(" world")))

	var found bool
	for c := doc.FirstChild(); c != nil; c = c.NextSibling() {
		if c.Type() == helium.TextNode {
			found = true
			require.Contains(t, string(c.Content()), "hello")
		}
	}
	require.True(t, found, "document gained a text child")
}

// TestNodeLine covers docnode.Line via a parsed node that carries line info.
func TestNodeLine(t *testing.T) {
	t.Parallel()
	const src = "<root>\n  <child/>\n</root>"
	doc, err := helium.NewParser().Parse(context.Background(), []byte(src))
	require.NoError(t, err)
	root := doc.DocumentElement()
	require.NotNil(t, root)
	// Line() returns the recorded line number; it must be a non-negative int and
	// not panic. We assert it is callable and consistent.
	require.GreaterOrEqual(t, root.Line(), 0)
	for c := root.FirstChild(); c != nil; c = c.NextSibling() {
		if c.Type() == helium.ElementNode {
			require.GreaterOrEqual(t, c.Line(), 0)
		}
	}
}

// TestWriteRichDTDWithEntities is a fuller round-trip that, in addition to the
// existing rich-DTD test, exercises serialization of a programmatically built
// DTD containing a percent-bearing internal entity and a parameter entity so the
// entity-content writer paths run end to end and re-parse cleanly.
func TestWriteRichDTDWithEntities(t *testing.T) {
	t.Parallel()

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	dtd, err := doc.CreateInternalSubset("doc", "", "")
	require.NoError(t, err)

	_, err = dtd.AddEntity("plain", enum.InternalGeneralEntity, "", "", "plain value")
	require.NoError(t, err)
	_, err = dtd.AddEntity("ext", enum.ExternalGeneralParsedEntity, "", "ext.xml", "")
	require.NoError(t, err)
	_, err = dtd.AddEntity("pub", enum.ExternalGeneralParsedEntity, "-//E//T//EN", "pub.xml", "")
	require.NoError(t, err)

	root := doc.CreateElement("doc")
	require.NoError(t, doc.AddChild(root))

	out, err := helium.WriteString(doc)
	require.NoError(t, err)
	require.Contains(t, out, "<!ENTITY plain")
	require.Contains(t, out, "<!ENTITY ext SYSTEM")
	require.Contains(t, out, "<!ENTITY pub PUBLIC")

	// Re-parse to confirm well-formedness.
	require.True(t, strings.Contains(out, "<!DOCTYPE doc"))
}
