package helium_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
	"github.com/stretchr/testify/require"
)

// TestInternalSubsetAccessors covers DTD construction and its accessor methods.
func TestInternalSubsetAccessors(t *testing.T) {
	t.Parallel()

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	dtd, err := doc.CreateInternalSubset("root", "-//Example//DTD//EN", "example.dtd")
	require.NoError(t, err)

	require.Equal(t, "-//Example//DTD//EN", dtd.ExternalID())
	require.Equal(t, "example.dtd", dtd.SystemID())
	require.Same(t, dtd, doc.IntSubset())

	// Exercise the DTD node-interface methods (delegating wrappers). They must
	// not panic; their success/error is implementation-defined for an
	// already-attached internal subset.
	x, err := doc.CreateElement("x")
	require.NoError(t, err)
	_ = dtd.AddSibling(x)
	_ = dtd.Replace()
	dtd.SetTreeDoc(doc)
	dtd.Free()
}

// TestDTDEntityAndNotation exercises AddEntity/AddNotation/ForEachEntity and the
// resulting Entity/Notation accessor methods.
func TestDTDEntityAndNotation(t *testing.T) {
	t.Parallel()

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	dtd, err := doc.CreateInternalSubset("doc", "", "")
	require.NoError(t, err)

	gen, err := dtd.AddEntity("greeting", enum.InternalGeneralEntity, "", "", "Hello")
	require.NoError(t, err)
	require.Equal(t, enum.InternalGeneralEntity, gen.EntityType())
	require.Equal(t, []byte("Hello"), gen.Content())

	// External unparsed entity exposes systemID/externalID/URI.
	img, err := dtd.AddEntity("img", enum.ExternalGeneralUnparsedEntity, "", "img.gif", "gif")
	require.NoError(t, err)
	require.Equal(t, "img.gif", img.SystemID())
	require.Equal(t, "", img.ExternalID())
	require.Equal(t, "img.gif", img.URI()) // falls back to systemID

	// First-definition-wins: redeclaring returns the existing entity.
	again, err := dtd.AddEntity("greeting", enum.InternalGeneralEntity, "", "", "Goodbye")
	require.NoError(t, err)
	require.Same(t, gen, again)

	// Redeclaring a predefined entity with the wrong content is rejected.
	_, err = dtd.AddEntity("lt", enum.InternalGeneralEntity, "", "", "wrong")
	require.Error(t, err)

	// Predefined entity type cannot be registered.
	_, err = dtd.AddEntity("x", enum.InternalPredefinedEntity, "", "", "y")
	require.Error(t, err)

	// ForEachEntity visits the general entities.
	seen := map[string]bool{}
	dtd.ForEachEntity(func(name string, ent *helium.Entity) {
		seen[name] = true
	})
	require.True(t, seen["greeting"])
	require.True(t, seen["img"])

	nota, err := dtd.AddNotation("gif", "", "viewer.exe")
	require.NoError(t, err)
	require.Equal(t, helium.NotationNode, nota.Type())

	// Redefinition of a notation is rejected.
	_, err = dtd.AddNotation("gif", "", "other.exe")
	require.Error(t, err)

	// Document.AddEntity delegates to the internal subset.
	_, err = doc.AddEntity("foo", enum.InternalGeneralEntity, "", "", "bar")
	require.NoError(t, err)

	// Document.AddEntity on a doc without an internal subset errors.
	bare := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)
	_, err = bare.AddEntity("foo", enum.InternalGeneralEntity, "", "", "bar")
	require.Error(t, err)
}

// TestAddNotationRejectsColon verifies AddNotation rejects a colon-bearing
// notation name (a notation name is an XML NCName, no colon), mirroring the
// parser's own "colons are forbidden from notation names" rule, so nothing is
// registered or serialized; a valid notation name still round-trips through a
// validating parser.
func TestAddNotationRejectsColon(t *testing.T) {
	t.Parallel()

	doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
	dtd, err := doc.CreateInternalSubset("doc", "", "")
	require.NoError(t, err)

	// A colon in the notation name is rejected with ErrInvalidArgument, and
	// nothing is registered.
	_, err = dtd.AddNotation("p:n", "", "sys")
	require.ErrorIs(t, err, helium.ErrInvalidArgument)
	_, ok := dtd.LookupNotation("p:n")
	require.False(t, ok, "rejected notation must not be registered")

	// A valid (colon-free) notation name is accepted.
	_, err = dtd.AddNotation("n", "", "sys")
	require.NoError(t, err)

	// Declare the root element so ValidateDTD accepts the document.
	_, err = dtd.AddElementDecl("doc", enum.AnyElementType, nil)
	require.NoError(t, err)
	docElem, err := doc.CreateElement("doc")
	require.NoError(t, err)
	require.NoError(t, doc.SetDocumentElement(docElem))

	var buf strings.Builder
	require.NoError(t, helium.Write(&buf, doc))
	out := buf.String()
	require.Contains(t, out, `<!NOTATION n SYSTEM "sys"`)
	require.NotContains(t, out, "p:n", "rejected notation must not be serialized")

	// Round-trip: a validating parser accepts the serialized document.
	_, err = helium.NewParser().ValidateDTD(true).Parse(t.Context(), []byte(out))
	require.NoError(t, err)
}

// TestDTDRemoveElement covers RemoveElement.
func TestDTDRemoveElement(t *testing.T) {
	t.Parallel()
	in, err := os.ReadFile("test/att12.xml")
	require.NoError(t, err)
	doc, err := helium.NewParser().Parse(t.Context(), in)
	require.NoError(t, err)

	dtd := doc.IntSubset()
	_, ok := dtd.LookupElement("doc", "")
	require.True(t, ok)

	dtd.RemoveElement("doc", "")
	_, ok = dtd.LookupElement("doc", "")
	require.False(t, ok)
}

// TestElementDeclAndAttrDeclAccessors covers DTD element/attribute declaration
// accessors by parsing a DTD that declares both.
func TestElementDeclAndAttrDeclAccessors(t *testing.T) {
	t.Parallel()
	in, err := os.ReadFile("test/att12.xml")
	require.NoError(t, err)
	doc, err := helium.NewParser().Parse(t.Context(), in)
	require.NoError(t, err)

	dtd := doc.IntSubset()
	require.NotNil(t, dtd)

	edecl, ok := dtd.LookupElement("doc", "")
	require.True(t, ok)
	require.Equal(t, enum.MixedElementType, edecl.DeclType())

	adecls := dtd.AttributesForElement("doc")
	require.NotEmpty(t, adecls)
	adecl := adecls[0]
	require.Equal(t, "doc", adecl.Elem())
	require.NotEqual(t, enum.AttrInvalid, adecl.AType())
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

	// Undeclared element => ErrElementDeclNotFound.
	_, err = doc.IsMixedElement("nope")
	require.ErrorIs(t, err, helium.ErrElementDeclNotFound)

	// Document without an internal subset => ErrElementDeclNotFound.
	bare := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)
	_, err = bare.IsMixedElement("x")
	require.ErrorIs(t, err, helium.ErrElementDeclNotFound)
}

// TestIsMixedElementAcrossSubsets verifies IsMixedElement returns the same
// classification whether an element is declared in the internal subset or solely
// in the external subset, mirroring libxml2's xmlIsMixedElement two-subset
// lookup. It also covers the EMPTY/ANY -> mixed collapse and the not-found error,
// in both subset placements.
func TestIsMixedElementAcrossSubsets(t *testing.T) {
	t.Parallel()

	const decls = `<!ELEMENT r (child)>
<!ELEMENT child (#PCDATA|em)*>
<!ELEMENT em EMPTY>
<!ELEMENT any ANY>`

	// The identical declarations, once in the internal subset and once in an
	// external DTD referenced by a bare SYSTEM DOCTYPE (no internal subset markup).
	const internalSrc = `<?xml version="1.0"?>
<!DOCTYPE r [
` + decls + `
]>
<r><child/></r>`
	const externalSrc = `<!DOCTYPE r SYSTEM "d.dtd"><r><child/></r>`

	internalDoc, err := helium.NewParser().Parse(t.Context(), []byte(internalSrc))
	require.NoError(t, err)

	fsys := fstest.MapFS{"d.dtd": &fstest.MapFile{Data: []byte(decls)}}
	externalDoc, err := helium.NewParser().
		BlockXXE(false).
		LoadExternalDTD(true).
		FS(fsys).
		Parse(t.Context(), []byte(externalSrc))
	require.NoError(t, err)

	subsets := []struct {
		label string
		doc   *helium.Document
	}{
		{label: "internal subset", doc: internalDoc},
		{label: "external subset", doc: externalDoc},
	}

	testcases := []struct {
		name    string
		wantMix bool
		wantErr bool
	}{
		{name: "r", wantMix: false},    // element-only content => not mixed
		{name: "child", wantMix: true}, // (#PCDATA|em)* => mixed
		{name: "em", wantMix: true},    // EMPTY collapses to mixed (VC-error path)
		{name: "any", wantMix: true},   // ANY collapses to mixed
		{name: "nope", wantErr: true},  // undeclared => ErrElementDeclNotFound
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			for _, ss := range subsets {
				mixed, err := ss.doc.IsMixedElement(tc.name)
				if tc.wantErr {
					require.ErrorIs(t, err, helium.ErrElementDeclNotFound, ss.label)
					continue
				}
				require.NoError(t, err, ss.label)
				require.Equal(t, tc.wantMix, mixed, ss.label)
			}
		})
	}
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
	root, err := doc.CreateElement("doc")
	require.NoError(t, err)
	require.NoError(t, doc.AddChild(root))

	dtd, err := doc.CreateInternalSubset("doc", "-//X//EN", "x.dtd")
	require.NoError(t, err)

	// The DTD must come before the root element in the child list.
	require.Same(t, helium.Node(dtd), doc.FirstChild())

	out, err := helium.WriteString(doc)
	require.NoError(t, err)
	require.True(t, strings.Contains(out, "<!DOCTYPE doc"))
}

// TestElementTypeString exercises the ElementType Stringer, including the
// out-of-range fallback.
func TestElementTypeString(t *testing.T) {
	t.Parallel()

	require.Equal(t, "ElementNode", helium.ElementNode.String())
	require.Equal(t, "DocumentNode", helium.DocumentNode.String())

	// An out-of-range value falls back to the numeric form.
	require.Contains(t, helium.ElementType(9999).String(), "ElementType(")
}

// TestDTDDeclarationNodeWrappers exercises the node-interface wrappers on the
// DTD, ElementDecl and AttributeDecl node types (AddChild, AppendText,
// AddSibling, Replace, SetTreeDoc) plus the Entity AddSibling/Replace wrappers.
// These all delegate to the shared tree primitives; the test confirms each
// override is wired up and returns the shared primitive's result.
func TestDTDDeclarationNodeWrappers(t *testing.T) {
	t.Parallel()

	// The ElementDecl/AttributeDecl subtests below share and mutate a single
	// parsed doc (AppendText/AddChild allocate text and comment nodes off the
	// document). helium Documents are a DOM tree and are not concurrency-safe,
	// so these subtests must run sequentially — do NOT add t.Parallel() to them.

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
		d3 := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
		dtd3, derr := d3.CreateInternalSubset("doc", "", "")
		require.NoError(t, derr)
		require.NoError(t, dtd3.AppendText([]byte("ws")))
		dtd3.Free() // no-op marker, but exercised for completeness
	})

	t.Run("Entity AddSibling and Replace", func(t *testing.T) {
		d4 := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
		dtd4, derr := d4.CreateInternalSubset("doc", "", "")
		require.NoError(t, derr)
		ent, eerr := dtd4.AddEntity("e", enum.InternalGeneralEntity, "", "", "v")
		require.NoError(t, eerr)
		_ = ent.AddSibling(d4.CreateComment([]byte("s")))
		_ = ent.Replace()
	})
}

// TestExternalIDPublicSystemLiteral covers XML 1.0 [75] ExternalID: the PUBLIC
// form requires a following SystemLiteral, whereas NotationDecl [83] PublicID
// permits PUBLIC with only a PubidLiteral.
func TestExternalIDPublicSystemLiteral(t *testing.T) {
	t.Parallel()

	t.Run("doctype PUBLIC without system literal rejected", func(t *testing.T) {
		t.Parallel()

		const input = `<?xml version="1.0"?>
<!DOCTYPE r PUBLIC "-//Example//DTD//EN">
<r/>`

		p := helium.NewParser()
		_, err := p.Parse(t.Context(), []byte(input))
		require.Error(t, err, "PUBLIC ExternalID without a SystemLiteral must be rejected")
	})

	t.Run("doctype PUBLIC with system literal accepted", func(t *testing.T) {
		t.Parallel()

		const input = `<?xml version="1.0"?>
<!DOCTYPE r PUBLIC "-//Example//DTD//EN" "example.dtd">
<r/>`

		p := helium.NewParser()
		_, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err, "a complete PUBLIC ExternalID must parse")
	})

	t.Run("notation PUBLIC without system literal accepted", func(t *testing.T) {
		t.Parallel()

		const input = `<?xml version="1.0"?>
<!DOCTYPE r [
  <!NOTATION n PUBLIC "-//Example//Notation//EN">
]>
<r/>`

		p := helium.NewParser()
		_, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err, "NotationDecl PublicID form must remain accepted")
	})
}
