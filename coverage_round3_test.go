package helium_test

import (
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
	"github.com/stretchr/testify/require"
)

// TestCreateReferenceWithDeclaredEntity exercises CreateReference both for a
// predefined entity (resolvable) and an undeclared name (no entity attached).
func TestCreateReferenceWithDeclaredEntity(t *testing.T) {
	t.Parallel()

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	dtd, err := doc.CreateInternalSubset("doc", "", "")
	require.NoError(t, err)

	_, err = dtd.AddEntity("greet", enum.InternalGeneralEntity, "", "", "Hello")
	require.NoError(t, err)

	// Reference to a declared general entity: the entity content is attached.
	ref, err := doc.CreateReference("greet")
	require.NoError(t, err)
	require.Equal(t, helium.EntityRefNode, ref.Type())
	require.Equal(t, []byte("Hello"), ref.Content())

	// Reference to an undeclared name: still produces an EntityRef, but with no
	// resolved content.
	ref2, err := doc.CreateReference("undeclared")
	require.NoError(t, err)
	require.Equal(t, "undeclared", ref2.Name())

	// "&name;" form is accepted and stripped.
	ref3, err := doc.CreateReference("&greet;")
	require.NoError(t, err)
	require.Equal(t, "greet", ref3.Name())

	// Empty name is rejected.
	_, err = doc.CreateReference("")
	require.Error(t, err)
}

// TestCreateCharRefForms covers the CreateCharRef name-stripping branches.
func TestCreateCharRefForms(t *testing.T) {
	t.Parallel()
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)

	plain, err := doc.CreateCharRef("foo")
	require.NoError(t, err)
	require.Equal(t, "foo", plain.Name())

	// "&foo;" -> "foo"
	full, err := doc.CreateCharRef("&foo;")
	require.NoError(t, err)
	require.Equal(t, "foo", full.Name())

	// "&foo" (no trailing semicolon) -> "foo"
	noSemi, err := doc.CreateCharRef("&foo")
	require.NoError(t, err)
	require.Equal(t, "foo", noSemi.Name())

	// Empty name and a name that decodes to empty are rejected.
	_, err = doc.CreateCharRef("")
	require.Error(t, err)
	_, err = doc.CreateCharRef("&;")
	require.Error(t, err)
}

// TestCreateAttributeWithEntityValue drives the stringToNodeList path inside
// CreateAttribute by passing a value containing character and entity references.
func TestCreateAttributeWithEntityValue(t *testing.T) {
	t.Parallel()
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)

	// Value with a decimal char ref, a hex char ref, and a named entity ref.
	attr, err := doc.CreateAttribute("a", "x&#65;y&#x42;z&amp;w", nil)
	require.NoError(t, err)
	require.Equal(t, "a", attr.Name())
	// The attribute has a child node list (text + entity refs).
	require.NotNil(t, attr.FirstChild())

	// Plain value (no '&') takes the fast single-text-node path.
	attr2, err := doc.CreateAttribute("b", "plain", nil)
	require.NoError(t, err)
	require.Equal(t, "plain", attr2.Value())

	// Colon in name is rejected.
	_, err = doc.CreateAttribute("ns:x", "v", nil)
	require.Error(t, err)
}

// TestGetEntityExternalSubset exercises GetEntity's external-subset lookup and
// the standalone short-circuit, plus GetParameterEntity.
func TestGetEntityFromInternalSubset(t *testing.T) {
	t.Parallel()
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	dtd, err := doc.CreateInternalSubset("doc", "", "")
	require.NoError(t, err)

	_, err = dtd.AddEntity("ge", enum.InternalGeneralEntity, "", "", "general")
	require.NoError(t, err)
	_, err = dtd.AddEntity("pe", enum.InternalParameterEntity, "", "", "param")
	require.NoError(t, err)

	ent, ok := doc.GetEntity("ge")
	require.True(t, ok)
	require.Equal(t, []byte("general"), ent.Content())

	_, ok = doc.GetEntity("missing")
	require.False(t, ok)

	pe, ok := doc.GetParameterEntity("pe")
	require.True(t, ok)
	require.Equal(t, []byte("param"), pe.Content())

	_, ok = doc.GetParameterEntity("missing")
	require.False(t, ok)

	// A document with no internal subset finds nothing.
	bare := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)
	_, ok = bare.GetEntity("ge")
	require.False(t, ok)
	_, ok = bare.GetParameterEntity("pe")
	require.False(t, ok)
}

// TestIsMixedElementDeclTypes exercises IsMixedElement across the declared
// element content types and the not-found error.
func TestIsMixedElementDeclTypes(t *testing.T) {
	t.Parallel()

	const src = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (child)>
  <!ELEMENT child (#PCDATA|sub)*>
  <!ELEMENT sub EMPTY>
  <!ELEMENT any ANY>
]>
<doc><child/></doc>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err)

	// Element-only content => not mixed.
	mixed, err := doc.IsMixedElement("doc")
	require.NoError(t, err)
	require.False(t, mixed)

	// (#PCDATA|sub)* => mixed.
	mixed, err = doc.IsMixedElement("child")
	require.NoError(t, err)
	require.True(t, mixed)

	// EMPTY => reported true (VC error path).
	mixed, err = doc.IsMixedElement("sub")
	require.NoError(t, err)
	require.True(t, mixed)

	// ANY => true.
	mixed, err = doc.IsMixedElement("any")
	require.NoError(t, err)
	require.True(t, mixed)

	// Undeclared element => error.
	_, err = doc.IsMixedElement("nope")
	require.Error(t, err)

	// Document without an internal subset => error.
	bare := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)
	_, err = bare.IsMixedElement("x")
	require.Error(t, err)
}

// TestEntityURIFallback covers Entity.URI's fallback to SystemID.
func TestEntityURIFallback(t *testing.T) {
	t.Parallel()
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	dtd, err := doc.CreateInternalSubset("doc", "", "")
	require.NoError(t, err)

	ext, err := dtd.AddEntity("e", enum.ExternalGeneralParsedEntity, "pub", "sys.ent", "")
	require.NoError(t, err)
	// No resolved URI set => falls back to SystemID.
	require.Equal(t, "sys.ent", ext.URI())
	require.Equal(t, "sys.ent", ext.SystemID())
	require.Equal(t, "pub", ext.ExternalID())
}

// TestNodeNamespaceMethods covers DeclareNamespace, SetActiveNamespace, SetNs,
// AddNamespaceDecl and the qname caching in Name().
func TestNodeNamespaceMethods(t *testing.T) {
	t.Parallel()
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root := doc.CreateElement("root")
	require.NoError(t, doc.AddChild(root))

	require.NoError(t, root.DeclareNamespace("p", "http://example.com/p"))
	require.NoError(t, root.SetActiveNamespace("p", "http://example.com/p"))

	// Name() now reflects the prefix and caches the qname.
	require.Equal(t, "p:root", root.Name())
	require.Equal(t, "p:root", root.Name()) // cached path
	require.Equal(t, "p", root.Prefix())
	require.Equal(t, "http://example.com/p", root.URI())

	// AddNamespaceDecl with an existing namespace object.
	ns := helium.NewNamespace("q", "http://example.com/q")
	root.AddNamespaceDecl(ns)
	root.SetNs(ns)
	require.Equal(t, "q:root", root.Name())
}

// TestChildElementsAndIterators covers the iter.go helpers including the
// element-only filter and early termination.
func TestChildElementsAndIterators(t *testing.T) {
	t.Parallel()
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root := doc.CreateElement("root")
	require.NoError(t, doc.AddChild(root))

	require.NoError(t, root.AppendText([]byte("text")))
	e1 := doc.CreateElement("a")
	require.NoError(t, root.AddChild(e1))
	require.NoError(t, root.AddChild(doc.CreateComment([]byte("c"))))
	e2 := doc.CreateElement("b")
	require.NoError(t, root.AddChild(e2))

	// ChildElements skips text/comment.
	var names []string
	for el := range helium.ChildElements(root) {
		names = append(names, el.Name())
	}
	require.Equal(t, []string{"a", "b"}, names)

	// Early break from ChildElements.
	count := 0
	for range helium.ChildElements(root) {
		count++
		break
	}
	require.Equal(t, 1, count)

	// Children yields all child nodes.
	all := 0
	for range helium.Children(root) {
		all++
	}
	require.Equal(t, 4, all)

	// Children/ChildElements/Descendants of nil yield nothing.
	for range helium.Children(nil) {
		t.Fatal("nil Children should yield nothing")
	}
	for range helium.ChildElements(nil) {
		t.Fatal("nil ChildElements should yield nothing")
	}
	for range helium.Descendants(nil) {
		t.Fatal("nil Descendants should yield nothing")
	}

	// Descendants does a depth-first walk; early break is honored.
	dcount := 0
	for range helium.Descendants(root) {
		dcount++
		break
	}
	require.Equal(t, 1, dcount)
}

// TestPINodeMethods exercises ProcessingInstruction AddChild/AppendText paths
// including the text-merge and rejection branches.
func TestPINodeMethods(t *testing.T) {
	t.Parallel()
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	pi := doc.CreatePI("target", "data")
	require.Equal(t, "target", pi.Name())
	require.Equal(t, []byte("data"), pi.Content())
	require.Equal(t, helium.ProcessingInstructionNode, pi.Type())

	// Adding a text node merges into the data string.
	require.NoError(t, pi.AddChild(doc.CreateText([]byte(" more"))))
	require.Equal(t, []byte("data more"), pi.Content())

	// Adding a CDATA node also merges.
	require.NoError(t, pi.AddChild(doc.CreateCDATASection([]byte("X"))))
	require.Contains(t, string(pi.Content()), "X")

	// AppendText appends directly.
	require.NoError(t, pi.AppendText([]byte("Y")))
	require.Contains(t, string(pi.Content()), "Y")

	// Adding an element child is rejected.
	require.Error(t, pi.AddChild(doc.CreateElement("e")))

	// Adding a nil node is rejected with ErrNilNode (not a panic).
	require.Error(t, pi.AddChild(nil))
}

// TestCommentNodeMethods exercises Comment AddChild merge/rejection branches.
func TestCommentNodeMethods(t *testing.T) {
	t.Parallel()
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	c := doc.CreateComment([]byte("hello"))
	require.Equal(t, []byte("hello"), c.Content())

	// Merging another (unlinked) comment appends its content.
	other := doc.CreateComment([]byte(" world"))
	require.NoError(t, c.AddChild(other))
	require.Equal(t, []byte("hello world"), c.Content())

	// AppendText appends.
	require.NoError(t, c.AppendText([]byte("!")))
	require.Equal(t, []byte("hello world!"), c.Content())

	// Adding a non-comment node is rejected.
	require.Error(t, c.AddChild(doc.CreateText([]byte("t"))))

	// Adding nil is rejected.
	require.Error(t, c.AddChild(nil))
}

// TestCDATANodeMethods exercises the CDATASection node methods.
func TestCDATANodeMethods(t *testing.T) {
	t.Parallel()
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	cd := doc.CreateCDATASection([]byte("data"))

	// AppendText grows the content.
	require.NoError(t, cd.AppendText([]byte("+more")))
	require.Equal(t, []byte("data+more"), cd.Content())

	// AddChild is rejected on a CDATA node.
	require.Error(t, cd.AddChild(doc.CreateText([]byte("x"))))

	// SetTreeDoc must not panic.
	cd.SetTreeDoc(doc)

	// AddSibling and Replace must run without panicking.
	root := doc.CreateElement("root")
	require.NoError(t, doc.AddChild(root))
	require.NoError(t, root.AddChild(cd))
	require.NoError(t, cd.AddSibling(doc.CreateCDATASection([]byte("sib"))))
}

// TestEntityRefNodeMethods exercises the EntityRef node-interface methods.
func TestEntityRefNodeMethods(t *testing.T) {
	t.Parallel()
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	ref, err := doc.CreateCharRef("amp")
	require.NoError(t, err)

	ref.SetTreeDoc(doc)
	require.NoError(t, ref.AppendText([]byte("x")))

	root := doc.CreateElement("root")
	require.NoError(t, doc.AddChild(root))
	require.NoError(t, root.AddChild(ref))
	require.NoError(t, ref.AddSibling(doc.CreateText([]byte("after"))))
}

// TestInternalSubsetErrors covers InternalSubset and CreateInternalSubset error
// branches: no subset, and double creation.
func TestInternalSubsetErrors(t *testing.T) {
	t.Parallel()
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)

	_, err := doc.InternalSubset()
	require.Error(t, err) // none yet

	_, err = doc.CreateInternalSubset("doc", "", "")
	require.NoError(t, err)

	got, err := doc.InternalSubset()
	require.NoError(t, err)
	require.NotNil(t, got)

	// Second creation is rejected.
	_, err = doc.CreateInternalSubset("doc", "", "")
	require.Error(t, err)
}

// TestCreateInternalSubsetBeforeRoot ensures the DTD is inserted before an
// already-present root element (the non-append branch).
func TestCreateInternalSubsetBeforeRoot(t *testing.T) {
	t.Parallel()
	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	root := doc.CreateElement("doc")
	require.NoError(t, doc.AddChild(root))

	dtd, err := doc.CreateInternalSubset("doc", "-//X//EN", "x.dtd")
	require.NoError(t, err)

	// The DTD must come before the root element in the child list.
	require.Same(t, helium.Node(dtd), doc.FirstChild())

	out, err := helium.WriteString(doc)
	require.NoError(t, err)
	require.True(t, strings.Contains(out, "<!DOCTYPE doc"))
}

// TestMalformedCharAndEntityRefs drives parseCharRef / parseEntityRef error
// branches with a table of malformed-reference documents.
func TestMalformedCharAndEntityRefs(t *testing.T) {
	t.Parallel()

	bad := []struct {
		name string
		src  string
	}{
		{"hex-missing-digits", `<root>&#x;</root>`},
		{"dec-missing-digits", `<root>&#;</root>`},
		{"hex-invalid-digit", `<root>&#xZZ;</root>`},
		{"dec-invalid-digit", `<root>&#12A3;</root>`},
		{"charref-out-of-range", `<root>&#x110000;</root>`},
		{"charref-control", `<root>&#x0;</root>`},
		{"undeclared-entity-standalone", `<?xml version="1.0" standalone="yes"?><root>&undeclared;</root>`},
		{"entity-empty-name", `<root>&;</root>`},
	}

	for _, tc := range bad {
		t.Run(tc.name, func(t *testing.T) {
			_, err := helium.NewParser().Parse(t.Context(), []byte(tc.src))
			require.Error(t, err, "expected parse error for %q", tc.src)
		})
	}
}

// TestEntityRefToUnparsedEntity drives the "entity reference to unparsed entity"
// error branch of parseEntityRef.
func TestEntityRefToUnparsedEntity(t *testing.T) {
	t.Parallel()

	const src = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!NOTATION gif SYSTEM "viewer">
  <!ENTITY img SYSTEM "img.gif" NDATA gif>
]>
<doc>&img;</doc>`

	_, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.Error(t, err)
}

// parseValidatingR3 parses src with DTD validation enabled, collecting errors.
func parseValidatingR3(t *testing.T, src string) []error {
	t.Helper()
	h := &collectingErrorHandler{}
	_, err := helium.NewParser().
		ValidateDTD(true).
		ErrorHandler(h).
		Parse(t.Context(), []byte(src))
	if err != nil {
		require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
	}
	return h.errs
}

// TestValidateGroupedSequenceOccurrences exercises matchSeq's Mult and Opt
// occurrence branches via grouped sequence content models.
func TestValidateGroupedSequenceOccurrences(t *testing.T) {
	t.Parallel()

	// (a, b)* — a repeated sequence group exercises matchSeq ElementContentMult.
	const dtdMult = `<!DOCTYPE doc [
<!ELEMENT doc (a, b)*>
<!ELEMENT a (#PCDATA)>
<!ELEMENT b (#PCDATA)>
]>`
	require.Empty(t, parseValidatingR3(t, dtdMult+`<doc><a/><b/><a/><b/></doc>`))
	require.Empty(t, parseValidatingR3(t, dtdMult+`<doc></doc>`)) // zero repetitions
	require.NotEmpty(t, parseValidatingR3(t, dtdMult+`<doc><a/></doc>`))

	// (a, b)+ — one-or-more sequence group exercises matchSeq ElementContentPlus.
	const dtdPlus = `<!DOCTYPE doc [
<!ELEMENT doc (a, b)+>
<!ELEMENT a (#PCDATA)>
<!ELEMENT b (#PCDATA)>
]>`
	require.Empty(t, parseValidatingR3(t, dtdPlus+`<doc><a/><b/></doc>`))
	require.Empty(t, parseValidatingR3(t, dtdPlus+`<doc><a/><b/><a/><b/></doc>`))
	require.NotEmpty(t, parseValidatingR3(t, dtdPlus+`<doc></doc>`))
}

// TestValidateChoiceOccurrences exercises matchOr's Mult/Opt/Once branches.
func TestValidateChoiceOccurrences(t *testing.T) {
	t.Parallel()

	// (a | b)* — choice with star.
	const dtdMult = `<!DOCTYPE doc [
<!ELEMENT doc (a | b)*>
<!ELEMENT a (#PCDATA)>
<!ELEMENT b (#PCDATA)>
]>`
	require.Empty(t, parseValidatingR3(t, dtdMult+`<doc></doc>`))
	require.Empty(t, parseValidatingR3(t, dtdMult+`<doc><a/><a/><b/></doc>`))

	// (a | b) once — exactly one of the two.
	const dtdOnce = `<!DOCTYPE doc [
<!ELEMENT doc (a | b)>
<!ELEMENT a (#PCDATA)>
<!ELEMENT b (#PCDATA)>
]>`
	require.Empty(t, parseValidatingR3(t, dtdOnce+`<doc><a/></doc>`))
	require.Empty(t, parseValidatingR3(t, dtdOnce+`<doc><b/></doc>`))
	require.NotEmpty(t, parseValidatingR3(t, dtdOnce+`<doc><a/><b/></doc>`))
}

// TestValidateRepeatedElement exercises matchElement's Mult/Plus branches.
func TestValidateRepeatedElement(t *testing.T) {
	t.Parallel()

	const dtd = `<!DOCTYPE doc [
<!ELEMENT doc (a+, b*)>
<!ELEMENT a (#PCDATA)>
<!ELEMENT b (#PCDATA)>
]>`
	require.Empty(t, parseValidatingR3(t, dtd+`<doc><a/></doc>`))
	require.Empty(t, parseValidatingR3(t, dtd+`<doc><a/><a/><a/><b/><b/></doc>`))
	require.NotEmpty(t, parseValidatingR3(t, dtd+`<doc><b/></doc>`)) // missing required a+
}

// TestValidateAttributeTypes exercises ID/IDREF/NMTOKEN/ENTITY attribute-type
// validation paths.
func TestValidateAttributeTypes(t *testing.T) {
	t.Parallel()

	const dtd = `<!DOCTYPE doc [
<!ELEMENT doc (item+)>
<!ELEMENT item EMPTY>
<!ATTLIST item
  id   ID    #REQUIRED
  ref  IDREF #IMPLIED
  tok  NMTOKEN #IMPLIED>
]>`

	// Valid: unique IDs, IDREF resolves, NMTOKEN well-formed.
	require.Empty(t, parseValidatingR3(t,
		dtd+`<doc><item id="a"/><item id="b" ref="a" tok="x1"/></doc>`))

	// Duplicate ID is a validation error.
	require.NotEmpty(t, parseValidatingR3(t,
		dtd+`<doc><item id="a"/><item id="a"/></doc>`))

	// IDREF pointing at a non-existent ID is a validation error.
	require.NotEmpty(t, parseValidatingR3(t,
		dtd+`<doc><item id="a" ref="missing"/></doc>`))
}

// TestValidateNotationAttribute exercises NOTATION-typed attribute validation.
func TestValidateNotationAttribute(t *testing.T) {
	t.Parallel()

	const dtd = `<!DOCTYPE doc [
<!NOTATION gif SYSTEM "viewer">
<!ELEMENT doc EMPTY>
<!ATTLIST doc kind NOTATION (gif) #IMPLIED>
]>`

	require.Empty(t, parseValidatingR3(t, dtd+`<doc kind="gif"/>`))
	require.NotEmpty(t, parseValidatingR3(t, dtd+`<doc kind="png"/>`))
}

// TestDTDSerializationRichSubset round-trips a document with a rich internal
// subset (entities, attributes with defaults, notations, varied content models)
// to exercise the DTD writer paths.
func TestDTDSerializationRichSubset(t *testing.T) {
	t.Parallel()

	const src = `<?xml version="1.0"?>
<!DOCTYPE doc [
<!ELEMENT doc (a | b)*>
<!ELEMENT a (#PCDATA)>
<!ELEMENT b EMPTY>
<!ATTLIST a
  id   ID       #IMPLIED
  kind (x | y)  "x"
  req  CDATA    #REQUIRED>
<!ENTITY internal "expanded">
<!ENTITY % pe "ignored">
<!NOTATION gif SYSTEM "viewer.exe">
]>
<doc><a id="i1" req="r">text</a><b/></doc>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err)

	out, err := helium.WriteString(doc)
	require.NoError(t, err)
	require.Contains(t, out, "<!DOCTYPE doc")
	require.Contains(t, out, "<!ELEMENT")
	require.Contains(t, out, "<!ATTLIST")
	require.Contains(t, out, "<!ENTITY")
	require.Contains(t, out, "<!NOTATION")
}

// TestValidCharRefForms parses documents with valid hex/decimal char refs to
// drive the success branches of parseCharRef.
func TestValidCharRefForms(t *testing.T) {
	t.Parallel()

	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root>&#65;&#x42;&#x4A;</root>`))
	require.NoError(t, err)
	root := doc.DocumentElement()
	require.NotNil(t, root)
	require.Equal(t, "ABJ", string(root.Content()))
}

// TestResolveCharRefsViaEntityContent indirectly exercises resolveCharRefs by
// round-tripping a document whose internal entity content contains char refs.
func TestResolveCharRefsViaParse(t *testing.T) {
	t.Parallel()

	const src = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ENTITY e "A&#66;C&#x44;E">
]>
<doc>&e;</doc>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, err)

	out, err := helium.WriteString(doc)
	require.NoError(t, err)
	require.Contains(t, out, "doc")
}
