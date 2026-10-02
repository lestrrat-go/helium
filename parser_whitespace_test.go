package helium_test

import (
	"bytes"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/encoding/charmap"
)

func TestStripBlanks(t *testing.T) {
	t.Parallel()

	t.Run("strip blanks", func(t *testing.T) {
		const input = `<?xml version="1.0"?>
<root>
  <child>text</child>
</root>`
		p := helium.NewParser().StripBlanks(true)
		doc, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err, "Parse should succeed")

		// With NoBlanks, blank-only text nodes between elements should be stripped.
		// The root element's first child should be <child>, not a whitespace text node.
		root := findDocumentElement(doc)
		require.NotNil(t, root, "document element must exist")
		first := root.FirstChild()
		require.NotNil(t, first, "root must have children")
		require.Equal(t, helium.ElementNode, first.Type(), "first child should be element, not blank text")
	})

	// that, under StripBlanks(true),
	// whitespace adjacent to a decoded entity reference (e.g. &gt;) is treated
	// exactly like whitespace adjacent to a literal character. Per XML 1.0 §4.4 an
	// entity reference and the character it expands to are equivalent, so both
	// forms must yield identical text content. The whitespace here abuts character
	// data (it is not ignorable inter-element whitespace) and must be preserved.
	t.Run("entity equivalence", func(t *testing.T) {
		testcases := []struct {
			name  string
			input string
			want  string
		}{
			{name: "literal trailing space", input: `<r>x </r>`, want: "x "},
			{name: "entity trailing space", input: `<r>&gt; </r>`, want: "> "},
			{name: "entity leading space", input: `<r> &gt;</r>`, want: " >"},
			{name: "entity surrounded by spaces", input: `<r> &gt; </r>`, want: " > "},
			{name: "entity then literal", input: `<r>&gt;x</r>`, want: ">x"},
		}

		for _, tc := range testcases {
			t.Run(tc.name, func(t *testing.T) {
				p := helium.NewParser().StripBlanks(true)
				doc, err := p.Parse(t.Context(), []byte(tc.input))
				require.NoError(t, err, "Parse should succeed")

				root := findDocumentElement(doc)
				require.NotNil(t, root, "document element must exist")

				var got []byte
				for child := root.FirstChild(); child != nil; child = child.NextSibling() {
					if child.Type() == helium.TextNode {
						got = append(got, child.Content()...)
					}
				}
				require.Equal(t, tc.want, string(got), "text content must match; entity and literal whitespace are equivalent")
			})
		}
	})

	// a DTD element
	// declaration whose name collides with the parser's internal synthetic
	// pseudo-root (used to wrap entity replacement text) cannot hijack the
	// whitespace classification of the entity's content. Under
	// StripBlanks(true)+SubstituteEntities(true), an entity expanding to "> " must
	// yield the same text as the literal "> ", even when the document declares
	// <!ELEMENT pseudoroot (pseudoroot)> (element content) — the synthetic
	// wrapper's name is chosen by the parser, not the document, so the DTD lookup
	// is skipped for it and the trailing space is preserved (XML §4.4 entity/literal
	// equivalence).
	t.Run("entity pseudo-root collision", func(t *testing.T) {
		testcases := []struct {
			name  string
			input string
		}{
			{name: "literal", input: `<r>&gt; </r>`},
			{
				name:  "general entity with colliding pseudoroot decl",
				input: `<!DOCTYPE r [<!ELEMENT pseudoroot (pseudoroot)><!ENTITY e "&gt; ">]><r>&e;</r>`,
			},
		}

		for _, tc := range testcases {
			t.Run(tc.name, func(t *testing.T) {
				p := helium.NewParser().StripBlanks(true).SubstituteEntities(true)
				doc, err := p.Parse(t.Context(), []byte(tc.input))
				require.NoError(t, err, "Parse should succeed")

				root := findDocumentElement(doc)
				require.NotNil(t, root, "document element must exist")

				var got []byte
				for child := root.FirstChild(); child != nil; child = child.NextSibling() {
					if child.Type() == helium.TextNode {
						got = append(got, child.Content()...)
					}
				}
				require.Equal(t, "> ", string(got), "entity content whitespace must match the literal regardless of a colliding pseudoroot decl")
			})
		}
	})

	// whitespace
	// classification (StripBlanks) consults the EXTERNAL subset, not only the
	// internal one. An element declared ANY has a mixed-like content model, so
	// whitespace inside it is significant and must survive StripBlanks(true).
	// libxml2's areBlanks checks both doc->intSubset and doc->extSubset; helium's
	// whitespace path uses elementDeclType, which likewise searches both subsets,
	// so declaring the model in the external DTD must behave exactly like declaring
	// it in the internal subset. (The public Document.IsMixedElement is the mixed
	// bool, not the raw content-model type, and is not the path exercised here.)
	t.Run("an external-subset content model", func(t *testing.T) {
		const extDTD = `<!ELEMENT r ANY>
<!ELEMENT c EMPTY>`

		testcases := []struct {
			name  string
			build func() helium.Parser
		}{
			{
				name: "internal subset ANY",
				build: func() helium.Parser {
					return helium.NewParser().StripBlanks(true)
				},
			},
			{
				name: "external subset ANY",
				build: func() helium.Parser {
					fsys := fstest.MapFS{"d.dtd": &fstest.MapFile{Data: []byte(extDTD)}}
					return helium.NewParser().
						StripBlanks(true).
						BlockXXE(false).
						LoadExternalDTD(true).
						FS(fsys)
				},
			},
		}

		// The trailing space after <c/> abuts ANY (mixed-like) content, so it is
		// significant and must be preserved as a text node under StripBlanks(true).
		const internalInput = `<!DOCTYPE r [<!ELEMENT r ANY><!ELEMENT c EMPTY>]><r><c/> </r>`
		const externalInput = `<!DOCTYPE r SYSTEM "d.dtd"><r><c/> </r>`

		for _, tc := range testcases {
			t.Run(tc.name, func(t *testing.T) {
				input := externalInput
				if tc.name == "internal subset ANY" {
					input = internalInput
				}

				doc, err := tc.build().Parse(t.Context(), []byte(input))
				require.NoError(t, err, "Parse should succeed")

				root := findDocumentElement(doc)
				require.NotNil(t, root, "document element must exist")

				var got []byte
				for child := root.FirstChild(); child != nil; child = child.NextSibling() {
					if child.Type() == helium.TextNode {
						got = append(got, child.Content()...)
					}
				}
				require.Equal(t, " ", string(got), "trailing whitespace inside an ANY-content element must be preserved")
			})
		}
	})

	// an EMPTY-declared element does
	// NOT preserve a stray inter-element whitespace run under StripBlanks(true).
	// libxml2's areBlanks uses its own decl switch (distinct from xmlIsMixedElement,
	// which maps EMPTY to "mixed"): only ELEMENT content is ignorable and only
	// ANY/MIXED is significant; EMPTY and UNDEFINED fall through to the heuristic,
	// which here classifies the run as ignorable. So whitespace must be stripped for
	// an EMPTY-declared element, matching an element-content model, as opposed to an
	// ANY/MIXED one. Both the internal and external subset declaration paths must
	// behave identically (areBlanks consults doc->intSubset then doc->extSubset).
	t.Run("an empty content model", func(t *testing.T) {
		const extDTD = `<!ELEMENT r EMPTY>
<!ELEMENT c EMPTY>`

		// <c/> inside an EMPTY-declared r is technically invalid, but a non-validating
		// parse still builds the tree; the trailing space after <c/> sits purely
		// between markup, so it is ignorable whitespace and dropped by StripBlanks.
		const internalInput = `<!DOCTYPE r [<!ELEMENT r EMPTY><!ELEMENT c EMPTY>]><r><c/> </r>`
		const externalInput = `<!DOCTYPE r SYSTEM "d.dtd"><r><c/> </r>`

		testcases := []struct {
			name  string
			input string
			build func() helium.Parser
		}{
			{
				name:  "internal subset EMPTY",
				input: internalInput,
				build: func() helium.Parser {
					return helium.NewParser().StripBlanks(true)
				},
			},
			{
				name:  "external subset EMPTY",
				input: externalInput,
				build: func() helium.Parser {
					fsys := fstest.MapFS{"d.dtd": &fstest.MapFile{Data: []byte(extDTD)}}
					return helium.NewParser().
						StripBlanks(true).
						BlockXXE(false).
						LoadExternalDTD(true).
						FS(fsys)
				},
			},
		}

		for _, tc := range testcases {
			t.Run(tc.name, func(t *testing.T) {
				doc, err := tc.build().Parse(t.Context(), []byte(tc.input))
				require.NoError(t, err, "Parse should succeed")

				root := findDocumentElement(doc)
				require.NotNil(t, root, "document element must exist")

				var got []byte
				for child := root.FirstChild(); child != nil; child = child.NextSibling() {
					if child.Type() == helium.TextNode {
						got = append(got, child.Content()...)
					}
				}
				require.Equal(t, "", string(got), "inter-element whitespace inside an EMPTY-content element must be stripped")
			})
		}
	})

	// The element declaration that classifies a whitespace run is found by the
	// element's qualified name exactly as written: a declaration for p:r applies
	// to <p:r> and not to <r>, and the other way round. Names longer than a short
	// lookup buffer and non-ASCII names resolve the same way. With no usable
	// declaration the run falls back to the markup heuristic, which keeps
	// whitespace that follows a text child.
	t.Run("element declarations keyed by qualified name", func(t *testing.T) {
		long := strings.Repeat("l", 70)
		testcases := []struct {
			name  string
			input string
			want  string
		}{
			{
				name:  "prefixed element content",
				input: "<!DOCTYPE p:r [<!ELEMENT p:r (p:c)*><!ELEMENT p:c (#PCDATA)>]><p:r xmlns:p=\"urn:p\">\n  <p:c>x</p:c>\n</p:r>",
				want:  `<p:r xmlns:p="urn:p"><p:c>x</p:c></p:r>`,
			},
			{
				name:  "prefixed ANY",
				input: "<!DOCTYPE p:r [<!ELEMENT p:r ANY>]><p:r xmlns:p=\"urn:p\">\n  <p:c>x</p:c>\n</p:r>",
				want:  "<p:r xmlns:p=\"urn:p\">\n  <p:c>x</p:c>\n</p:r>",
			},
			{
				name:  "unprefixed declaration does not match a prefixed element",
				input: "<!DOCTYPE p:r [<!ELEMENT r ANY>]><p:r xmlns:p=\"urn:p\">\n  <c>x</c>\n</p:r>",
				want:  `<p:r xmlns:p="urn:p"><c>x</c></p:r>`,
			},
			{
				name:  "prefixed declaration does not match an unprefixed element",
				input: "<!DOCTYPE r [<!ELEMENT p:r ANY>]><r xmlns:p=\"urn:p\">\n  <c>x</c>\n</r>",
				want:  `<r xmlns:p="urn:p"><c>x</c></r>`,
			},
			{
				name:  "unprefixed ANY",
				input: "<!DOCTYPE r [<!ELEMENT r ANY>]><r>\n  <c>x</c>\n</r>",
				want:  "<r>\n  <c>x</c>\n</r>",
			},
			{
				name:  "prefixed EMPTY falls back to the heuristic",
				input: "<!DOCTYPE p:r [<!ELEMENT p:r EMPTY>]><p:r xmlns:p=\"urn:p\">\n  <c>x</c>\n</p:r>",
				want:  `<p:r xmlns:p="urn:p"><c>x</c></p:r>`,
			},
			{
				name:  "long prefixed name",
				input: "<!DOCTYPE p:" + long + " [<!ELEMENT p:" + long + " ANY>]><p:" + long + " xmlns:p=\"urn:p\">\n  <c>x</c>\n</p:" + long + ">",
				want:  "<p:" + long + " xmlns:p=\"urn:p\">\n  <c>x</c>\n</p:" + long + ">",
			},
			{
				name:  "long unprefixed name",
				input: "<!DOCTYPE " + long + " [<!ELEMENT " + long + " ANY>]><" + long + ">\n  <c>x</c>\n</" + long + ">",
				want:  "<" + long + ">\n  <c>x</c>\n</" + long + ">",
			},
			{
				name:  "non-ASCII prefixed ANY",
				input: "<!DOCTYPE é:r [<!ELEMENT é:r ANY>]><é:r xmlns:é=\"urn:e\">\n  <c>x</c>\n</é:r>",
				want:  "<é:r xmlns:é=\"urn:e\">\n  <c>x</c>\n</é:r>",
			},
			{
				name:  "non-ASCII prefixed element content",
				input: "<!DOCTYPE é:r [<!ELEMENT é:r (c)*>]><é:r xmlns:é=\"urn:e\">\n  <c>x</c>\n</é:r>",
				want:  `<é:r xmlns:é="urn:e"><c>x</c></é:r>`,
			},
			{
				name:  "no declaration",
				input: "<r>\n  <c>x</c>\n</r>",
				want:  `<r><c>x</c></r>`,
			},
			{
				name:  "no declaration with a text first child",
				input: "<r>x<c/>  <c/></r>",
				want:  "<r>x<c/>  <c/></r>",
			},
			{
				name:  "no declaration with a text last child",
				input: "<r><c/>x  <c/></r>",
				want:  "<r><c/>x  <c/></r>",
			},
		}

		for _, tc := range testcases {
			t.Run(tc.name, func(t *testing.T) {
				doc, err := helium.NewParser().StripBlanks(true).Parse(t.Context(), []byte(tc.input))
				require.NoError(t, err, "Parse should succeed")

				got, err := helium.WriteString(doc.DocumentElement())
				require.NoError(t, err, "WriteString should succeed")
				require.Equal(t, tc.want, got, "whitespace classification must follow the declaration for the element's qualified name")
			})
		}
	})

	t.Run("declared content type per element", func(t *testing.T) {
		t.Parallel()

		// Element content (a, p:a) drops every blank run, including the one
		// after a character reference; mixed and ANY content (m, p:m, q:a)
		// keeps them all; EMPTY (e) and undeclared (u, p:u) elements drop only
		// the runs between markup. Entity replacement text is classified
		// without the enclosing element's declaration.
		const (
			body = `<r xmlns:p="urn:p" xmlns:q="urn:p">` +
				`<a><b/><b/>&amp;<b/></a><p:a><b/><b/>&amp;<b/></p:a><q:a> <b/> <b/> &amp; <b/> </q:a>` +
				`<m> <b/> <b/> &amp; <b/> </m><p:m> <b/> <b/> &amp; <b/> </p:m><e><b/> &amp; <b/></e>` +
				`<u><b/><b/> &amp; <b/></u><p:u><b/><b/> &amp; <b/></p:u>` +
				`<a><a><b/></a><m> <b/> </m><b/></a>`
			wantRefs        = body + `<a>&ws;</a><m>&ws;</m><u>  </u></r>`
			wantSubstituted = body + `<a><b/> </a><m><b/> </m><u>  </u></r>`
		)
		for _, tc := range declaredWhitespaceCases {
			for _, substitute := range []bool{false, true} {
				for _, saxPath := range []bool{false, true} {
					p := helium.NewParser().StripBlanks(true).SubstituteEntities(substitute)
					if tc.external {
						fsys := fstest.MapFS{dtdSystemID: &fstest.MapFile{Data: []byte(declaredWhitespaceDecls)}}
						p = p.BlockXXE(false).LoadExternalDTD(true).FS(fsys)
					}
					if saxPath {
						p = p.SAXHandler(&saxTreeBuilder{TreeBuilder: helium.NewTreeBuilder()})
					}
					doc, err := p.Parse(t.Context(), []byte(tc.src))
					require.NoError(t, err)
					got, err := helium.WriteString(doc.DocumentElement())
					require.NoError(t, err)
					want := wantRefs
					if substitute {
						want = wantSubstituted
					}
					require.Equal(t, want, got, "%s, substitute %t, SAX path %t", tc.name, substitute, saxPath)

					if saxPath {
						continue
					}
					// The push parser and a fragment parsed in the document's
					// context classify the same way.
					pp := p.NewPushParser(t.Context())
					for chunk := range slices.Chunk([]byte(tc.src), 7) {
						require.NoError(t, pp.Push(chunk))
					}
					pushed, err := pp.Close()
					require.NoError(t, err)
					got, err = helium.WriteString(pushed.DocumentElement())
					require.NoError(t, err)
					require.Equal(t, want, got, "%s, substitute %t, push parser", tc.name, substitute)

					first, err := p.ParseInNodeContext(t.Context(), doc.DocumentElement(),
						[]byte(`<a> <b/> &#38; <b/> </a> <m> <b/> </m> <p:a> <b/> &#38; </p:a> <u> <b/> &#38; </u>`))
					require.NoError(t, err)
					var fragment strings.Builder
					for n := first; n != nil; n = n.NextSibling() {
						s, err := helium.WriteString(n)
						require.NoError(t, err)
						fragment.WriteString(s)
					}
					require.Equal(t, `<a><b/>&amp;<b/></a><m> <b/> </m><p:a xmlns:p="urn:p"><b/>&amp;</p:a><u><b/> &amp; </u>`,
						fragment.String(), "%s, substitute %t, fragment", tc.name, substitute)
				}
			}
		}
	})
}

func TestWhitespacePreserved(t *testing.T) {
	t.Parallel()

	// confirms the bounded blank
	// skip still handles ordinary (within-cap) whitespace correctly: leading
	// whitespace before the root, whitespace around the XML declaration, and
	// trailing whitespace in the epilogue all parse without error.
	t.Run("leading and trailing whitespace", func(t *testing.T) {
		docs := map[string]string{
			"leading before root":   "<?xml version=\"1.0\"?>\n\n  \t<root/>",
			"trailing epilogue":     "<root/>\n  \t\n",
			"between prolog nodes":  "<?xml version=\"1.0\"?>\n<!-- c -->\n  <root/>\n",
			"large within-cap blob": "<?xml version=\"1.0\"?>" + strings.Repeat(" ", 5000) + "<root/>",
		}

		for name, doc := range docs {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				// A cap larger than the within-cap blob; ordinary whitespace must
				// never trip the blank-run guard.
				d, err := helium.NewParser().MaxNodeContentSize(1<<20).Parse(t.Context(), []byte(doc))
				require.NoError(t, err)
				require.NotNil(t, d)
			})
		}
	})

	t.Run("xml:space", func(t *testing.T) {
		t.Run("preserve keeps whitespace with StripBlanks", func(t *testing.T) {
			const input = `<?xml version="1.0"?>
<root xml:space="preserve">
  <child>text</child>
</root>`
			p := helium.NewParser().StripBlanks(true)
			doc, err := p.Parse(t.Context(), []byte(input))
			require.NoError(t, err, "Parse should succeed")

			root := findDocumentElement(doc)
			require.NotNil(t, root, "document element must exist")
			first := root.FirstChild()
			require.NotNil(t, first, "root must have children")
			// With xml:space="preserve", blank-only text nodes must be kept even with StripBlanks.
			require.Equal(t, helium.TextNode, first.Type(), "first child should be text node (preserved whitespace)")
		})

		t.Run("default reverts preserve", func(t *testing.T) {
			// xml:space="default" on an element should cause blanks to be stripped
			// even when a parent had xml:space="preserve".
			// Note: libxml2's spaceTab is per-element (not inherited), so only
			// the element with the explicit attribute is affected.
			const input = `<?xml version="1.0"?>
<root xml:space="preserve">
  <child xml:space="default">
    <leaf>text</leaf>
  </child>
</root>`
			p := helium.NewParser().StripBlanks(true)
			doc, err := p.Parse(t.Context(), []byte(input))
			require.NoError(t, err, "Parse should succeed")

			root := findDocumentElement(doc)
			require.NotNil(t, root, "document element must exist")
			// root has xml:space="preserve", so its whitespace text nodes are kept
			require.Equal(t, helium.TextNode, root.FirstChild().Type(), "root whitespace should be preserved")

			// Find <child>
			var child helium.Node
			for c := root.FirstChild(); c != nil; c = c.NextSibling() {
				if c.Type() == helium.ElementNode && c.(*helium.Element).LocalName() == "child" {
					child = c
					break
				}
			}
			require.NotNil(t, child, "child element must exist")
			// child has xml:space="default", so blanks should be stripped
			first := child.FirstChild()
			require.NotNil(t, first, "child must have children")
			require.Equal(t, helium.ElementNode, first.Type(), "child's first child should be element (blanks stripped by default)")
		})

		t.Run("preserve pops correctly after element closes", func(t *testing.T) {
			const input = `<?xml version="1.0"?>
<root>
  <preserved xml:space="preserve">
    <child>text</child>
  </preserved>
  <normal>
    <child>text</child>
  </normal>
</root>`
			p := helium.NewParser().StripBlanks(true)
			doc, err := p.Parse(t.Context(), []byte(input))
			require.NoError(t, err, "Parse should succeed")

			root := findDocumentElement(doc)
			require.NotNil(t, root, "document element must exist")

			// Find <preserved> and <normal>
			var preserved, normal helium.Node
			for c := root.FirstChild(); c != nil; c = c.NextSibling() {
				if c.Type() == helium.ElementNode {
					switch c.(*helium.Element).LocalName() {
					case "preserved":
						preserved = c
					case "normal":
						normal = c
					}
				}
			}
			require.NotNil(t, preserved, "preserved element must exist")
			require.NotNil(t, normal, "normal element must exist")

			// preserved's first child should be whitespace text
			require.Equal(t, helium.TextNode, preserved.FirstChild().Type(), "preserved whitespace should be kept")

			// normal's first child should be element (blanks stripped)
			require.Equal(t, helium.ElementNode, normal.FirstChild().Type(), "normal whitespace should be stripped")
		})
	})
}

func TestOverCapWhitespace(t *testing.T) {
	t.Parallel()

	// an over-cap whitespace run
	// inside the XML DECLARATION and inside the DTD internal subset surfaces
	// ErrNodeContentTooLarge, not a generic XML-decl / DTD syntax error. The
	// blank-skip helpers only return a bool, so the callers in those positions keep
	// going after the sticky over-cap error is recorded; without the central
	// preference for the blank-run error in errorAtLevel the follow-on syntax error
	// (a malformed version info / "DOCTYPE not finished") would mask the real cause.
	t.Run("in the declaration and DTD", func(t *testing.T) {
		const limit = 4096
		blanks := strings.Repeat(" ", limit*2)

		cases := map[string]string{
			// Whitespace between '<?xml' and the version pseudo-attribute.
			"xml declaration whitespace": "<?xml" + blanks + `version="1.0"?><root/>`,
			// Whitespace inside the DTD internal subset.
			"dtd subset whitespace": `<?xml version="1.0"?><!DOCTYPE root [` + blanks + `]><root/>`,
		}

		for name, doc := range cases {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				_, err := helium.NewParser().MaxNodeContentSize(limit).Parse(t.Context(), []byte(doc))
				require.ErrorIs(t, err, helium.ErrNodeContentTooLarge,
					"over-cap %s must surface ErrNodeContentTooLarge, not a masking syntax error", name)
			})
		}
	})

	// pins the blank-run cap on
	// the conditional-section HEADER whitespace skips in parseConditionalSections:
	// after "<![", after the INCLUDE keyword, and after the IGNORE keyword. These
	// positions route through skipBlanks (which records ErrNodeContentTooLarge in
	// pctx.blankRunErr but returns the conditional-section sentinel), and that
	// sentinel was previously TOLERATED by the top-level external-subset loop —
	// silently downgrading a resource-limit violation to "stop parsing the subset".
	// Each over-cap run in the header must instead fail closed with
	// ErrNodeContentTooLarge. Distinct from over-limit whitespace inside the INCLUDE
	// body / before its terminator (covered by TestParseOverCapWhitespaceInExternalSubset).
	t.Run("in a conditional-section header", func(t *testing.T) {
		const limit = 4096
		blanks := strings.Repeat(" ", limit*2)

		const input = `<?xml version="1.0"?>
<!DOCTYPE r SYSTEM "ws.dtd">
<r/>`

		cases := map[string]string{
			// Whitespace immediately after "<![", before the INCLUDE keyword.
			"after open bracket": "<![" + blanks + "INCLUDE[<!ELEMENT r EMPTY>]]>",
			// Whitespace after the INCLUDE keyword, before its "[".
			"after include keyword": "<![INCLUDE" + blanks + "[<!ELEMENT r EMPTY>]]>",
			// Whitespace after the IGNORE keyword, before its "[".
			"after ignore keyword": "<!ELEMENT r EMPTY><![IGNORE" + blanks + "[ <!ELEMENT q EMPTY> ]]>",
		}

		for name, dtd := range cases {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				fsys := fstest.MapFS{"ws.dtd": &fstest.MapFile{Data: []byte(dtd)}}
				p := helium.NewParser().
					BlockXXE(false).
					LoadExternalDTD(true).
					DefaultDTDAttributes(true).
					MaxNodeContentSize(limit).
					FS(fsys)
				_, err := p.Parse(t.Context(), []byte(input))
				require.ErrorIs(t, err, helium.ErrNodeContentTooLarge,
					"over-cap whitespace %s must surface ErrNodeContentTooLarge (not be tolerated as a conditional-section sentinel)", name)
			})
		}
	})

	// pins the blank-run cap on the two
	// external-subset blank skips that intentionally bypass skipBlanks to preserve
	// %pe; expansion: the declaration-step skip (parseExternalSubsetDeclStep) and
	// the INCLUDE-terminator skip (parseConditionalSections). Both now route through
	// skipBlankRun, so an over-cap contiguous whitespace run in either position must
	// fail with ErrNodeContentTooLarge instead of forcing the cursor to buffer the
	// whole run.
	t.Run("in the external subset", func(t *testing.T) {
		const limit = 4096
		blanks := strings.Repeat(" ", limit*2)

		const input = `<?xml version="1.0"?>
<!DOCTYPE r SYSTEM "ws.dtd">
<r/>`

		cases := map[string]string{
			// Whitespace between two declarations in the external subset, consumed by
			// parseExternalSubsetDeclStep's blank-only skip.
			"external subset declaration step": "<!ELEMENT r EMPTY>" + blanks + "<!ATTLIST r x CDATA 'd'>",
			// Whitespace just before the INCLUDE section's "]]>" terminator, consumed
			// by parseConditionalSections's section-cursor blank skip.
			"include section terminator": "<![INCLUDE[<!ELEMENT r EMPTY>" + blanks + "]]>",
		}

		for name, dtd := range cases {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				fsys := fstest.MapFS{"ws.dtd": &fstest.MapFile{Data: []byte(dtd)}}
				p := helium.NewParser().
					BlockXXE(false).
					LoadExternalDTD(true).
					DefaultDTDAttributes(true).
					MaxNodeContentSize(limit).
					FS(fsys)
				_, err := p.Parse(t.Context(), []byte(input))
				require.ErrorIs(t, err, helium.ErrNodeContentTooLarge,
					"over-cap whitespace in %s must surface ErrNodeContentTooLarge", name)
			})
		}
	})

	// the
	// external-subset blank-run cap holds regardless of how the MAIN document is
	// delivered: an EBCDIC document fed through ParseReader, whose external subset
	// (loaded from the fs.FS) carries an over-cap contiguous whitespace run between
	// declarations, must still fail closed with ErrNodeContentTooLarge instead of
	// letting the external-subset declaration step buffer the whole run. The same
	// EBCDIC bytes via Parse([]byte) must fail identically — parity between the two
	// entry points.
	t.Run("in an EBCDIC external subset", func(t *testing.T) {
		const limit = 4096
		blanks := strings.Repeat(" ", limit*2)
		dtd := "<!ELEMENT r EMPTY>" + blanks + "<!ATTLIST r x CDATA 'd'>"

		const decl = `<?xml version="1.0" encoding="IBM037"?>` + "\n" +
			`<!DOCTYPE r SYSTEM "ws.dtd">` + "\n" + `<r/>`
		ebcdic, err := charmap.CodePage037.NewEncoder().Bytes([]byte(decl))
		require.NoError(t, err)
		require.Equal(t, []byte{0x4C, 0x6F, 0xA7, 0x94}, ebcdic[:4],
			"encoded bytes must start with the EBCDIC invariant prefix")

		newParser := func() helium.Parser {
			return helium.NewParser().
				BlockXXE(false).
				LoadExternalDTD(true).
				DefaultDTDAttributes(true).
				MaxNodeContentSize(limit).
				FS(fstest.MapFS{"ws.dtd": &fstest.MapFile{Data: []byte(dtd)}})
		}

		_, rerr := newParser().ParseReader(t.Context(), bytes.NewReader(ebcdic))
		require.ErrorIs(t, rerr, helium.ErrNodeContentTooLarge,
			"over-cap external-subset whitespace must surface ErrNodeContentTooLarge via ParseReader/EBCDIC")

		_, berr := newParser().Parse(t.Context(), ebcdic)
		require.ErrorIs(t, berr, helium.ErrNodeContentTooLarge,
			"the same EBCDIC bytes via Parse([]byte) must fail identically")
	})
}

// declaredWhitespaceDecls declares element, mixed, ANY and EMPTY content for
// unprefixed and prefixed element names. p:a and q:a bind the same namespace
// and local name but are declared differently, and p:u is not declared, since
// a declaration is looked up by the name as written.
const declaredWhitespaceDecls = `<!ELEMENT r (a|p:a|q:a|m|p:m|e|u|p:u)*>
<!ELEMENT a (b|a|m)*>
<!ELEMENT p:a (b)*>
<!ELEMENT q:a ANY>
<!ELEMENT m (#PCDATA|b)*>
<!ELEMENT p:m ANY>
<!ELEMENT e EMPTY>
<!ELEMENT b EMPTY>
<!ENTITY ws " <b/> ">`

// declaredWhitespaceBody gives each element several whitespace runs, one of
// them after a character reference, where only a declaration makes it
// ignorable; nests declared elements inside one another; and expands an
// entity holding whitespace inside element and mixed content.
const declaredWhitespaceBody = `<r xmlns:p="urn:p" xmlns:q="urn:p">
 <a> <b/> <b/> &#38; <b/> </a>
 <p:a> <b/> <b/> &#38; <b/> </p:a>
 <q:a> <b/> <b/> &#38; <b/> </q:a>
 <m> <b/> <b/> &#38; <b/> </m>
 <p:m> <b/> <b/> &#38; <b/> </p:m>
 <e> <b/> &#38; <b/> </e>
 <u> <b/> <b/> &#38; <b/> </u>
 <p:u> <b/> <b/> &#38; <b/> </p:u>
 <a> <a> <b/> </a> <m> <b/> </m> <b/> </a>
 <a>&ws;</a>
 <m>&ws;</m>
 <u>  </u>
</r>`

// declaredWhitespaceCases parse the same document with its declarations in
// the internal subset and in an external subset.
var declaredWhitespaceCases = []struct {
	name     string
	src      string
	external bool
}{
	{name: "internal subset", src: "<!DOCTYPE r [\n" + declaredWhitespaceDecls + "\n]>\n" + declaredWhitespaceBody},
	{name: "external subset", src: `<!DOCTYPE r SYSTEM "` + dtdSystemID + `">` + "\n" + declaredWhitespaceBody, external: true},
}
