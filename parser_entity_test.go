package helium_test

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"

	"github.com/lestrrat-go/helium"

	"github.com/lestrrat-go/helium/enum"
	"github.com/stretchr/testify/require"
)

// externalEntityMaxBytes mirrors the unexported cap in parserctx.go. The size
// guard is what this test exercises; keep the two in sync.
const externalEntityMaxBytes int64 = 10 * 1024 * 1024 // 10 MiB

// entityAllowedExpansionBytes and entityFixedCostBytes mirror the unexported
// amplification constants in parserctx.go (entityAllowedExpansion and
// entityFixedCost). They let the single-reference accounting test sit exactly
// at the boundary where charging a second fixed cost would cross the baseline.
// Keep them in sync with parserctx.go.
const (
	entityAllowedExpansionBytes int64 = 1_000_000 // 1 MB baseline before ratio check
	entityFixedCostBytes        int64 = 20        // fixed byte cost per entity reference
)

// countingFS hands out the same byte content on every Open and records how many
// times Open was called, so a test can prove that repeated references to one
// external entity read the source only once (the rest hit the cached
// expandedSize accounting).
type countingFS struct {
	data  string
	opens *atomic.Int64
}

func (s countingFS) Open(string) (fs.File, error) {
	s.opens.Add(1)
	return &readCloserFile{
		r:      io.NewSectionReader(strings.NewReader(s.data), 0, int64(len(s.data))),
		closed: &atomic.Bool{},
	}, nil
}

// wantAttrA1V1 is the serialized default attribute the PE-in-markup fixtures
// assemble; shared to keep the repeated literal in one place.
const wantAttrA1V1 = `a1="v1"`

// writeDTD writes a DTD file into a fresh temp dir and returns its path.
func writeDTD(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "ext.dtd")
	require.NoError(t, os.WriteFile(p, []byte(body), 0600))
	return p
}

func TestEntitySubstitution(t *testing.T) {
	t.Parallel()

	// entity expansion in content and attributes.
	t.Run("general entities", func(t *testing.T) {
		const src = `<?xml version="1.0"?>
<!DOCTYPE doc [
<!ENTITY greeting "hello world">
]>
<doc attr="&greeting;">&greeting;</doc>`

		doc, err := helium.NewParser().SubstituteEntities(true).Parse(t.Context(), []byte(src))
		require.NoError(t, err)
		require.Contains(t, string(doc.DocumentElement().Content()), "hello world")
	})

	// keeps an XML 1.1
	// restricted character reference as a reference until the nested internal-entity
	// parser handles it. The XML 1.1 reference is therefore valid, while an XML 1.0
	// reference and an XML 1.1 raw restricted character remain invalid.
	t.Run("an XML 1.1 character reference in an internal entity", func(t *testing.T) {
		t.Run("XML 1.1 character reference expands", func(t *testing.T) {
			t.Parallel()

			const input = `<?xml version="1.1"?>
<!DOCTYPE r [<!ENTITY e "&#1;">]><r>&e;</r>`
			doc, err := helium.NewParser().SubstituteEntities(true).Parse(t.Context(), []byte(input))
			require.NoError(t, err)
			require.Equal(t, "\x01", string(doc.DocumentElement().Content()))
			ent, ok := doc.GetEntity("e")
			require.True(t, ok)
			require.Equal(t, "\x01", string(ent.Content()))
		})

		t.Run("XML 1.0 character reference remains invalid", func(t *testing.T) {
			t.Parallel()

			const input = `<?xml version="1.0"?>
<!DOCTYPE r [<!ENTITY e "&#1;">]><r>&e;</r>`
			_, err := helium.NewParser().SubstituteEntities(true).Parse(t.Context(), []byte(input))
			require.Error(t, err)
			require.Contains(t, err.Error(), "invalid XML char value 1")
		})

		t.Run("XML 1.1 raw restricted character remains invalid", func(t *testing.T) {
			t.Parallel()

			input := `<?xml version="1.1"?>` + "\n" +
				`<!DOCTYPE r [<!ENTITY e "` + "\x01" + `">]><r>&e;</r>`
			_, err := helium.NewParser().SubstituteEntities(true).Parse(t.Context(), []byte(input))
			require.Error(t, err)
		})
	})

	// A text-only internal entity remains accepted.
	t.Run("balanced text is accepted", func(t *testing.T) {
		const src = `<!DOCTYPE doc [
<!ENTITY e "plain text">
]>
<doc>&e;</doc>`
		doc, err := helium.NewParser().SubstituteEntities(true).Parse(t.Context(), []byte(src))
		require.NoError(t, err, "a text entity must be accepted")
		require.NotNil(t, doc)
		require.Equal(t, "plain text", string(doc.DocumentElement().Content()))
	})

	// A well-balanced internal entity — a complete element subtree — is accepted and
	// its content is spliced into the referencing element.
	t.Run("a balanced element is accepted", func(t *testing.T) {
		const src = `<!DOCTYPE doc [
<!ENTITY e "<b>x</b>">
]>
<doc>&e;</doc>`
		doc, err := helium.NewParser().SubstituteEntities(true).Parse(t.Context(), []byte(src))
		require.NoError(t, err, "a balanced element subtree entity must be accepted")
		require.NotNil(t, doc)
	})

	// An internal general entity whose replacement text straddles element
	// boundaries — here it closes an element opened outside the entity and opens a
	// fresh one closed outside — is not well balanced. Referencing it in element
	// content is a fatal well-formedness error (W3C xmlconf not-wf-sa-074; WFC:
	// parsed entities must be well-formed, XML §4.3.2).
	t.Run("unbalanced nesting is rejected", func(t *testing.T) {
		const src = `<!DOCTYPE doc [
<!ENTITY e "</foo><foo>">
]>
<doc>
<foo>&e;</foo>
</doc>`
		_, err := helium.NewParser().SubstituteEntities(true).Parse(t.Context(), []byte(src))
		require.Error(t, err, "an entity that breaks element nesting must be rejected")
		require.Contains(t, err.Error(), "not well balanced")
	})

	// An internal entity that closes, mid-content, an element opened OUTSIDE it is
	// equally unbalanced (a trailing stray end-tag, not a leading one).
	t.Run("a trailing end tag is rejected", func(t *testing.T) {
		const src = `<!DOCTYPE doc [
<!ENTITY e "text</foo>">
]>
<doc><foo>&e;more</foo></doc>`
		_, err := helium.NewParser().SubstituteEntities(true).Parse(t.Context(), []byte(src))
		require.Error(t, err, "an entity closing an outer element must be rejected")
		require.Contains(t, err.Error(), "not well balanced")
	})
}

func TestPredefinedEntities(t *testing.T) {
	t.Parallel()

	t.Run("predefined set", func(t *testing.T) {
		// Predefined entities (&lt; &gt; &amp; &apos; &quot;) must never trigger the guard.
		xml := `<?xml version="1.0"?>
<root>&lt;&gt;&amp;&apos;&quot;</root>`

		p := helium.NewParser()
		doc, err := p.Parse(t.Context(), []byte(xml))
		require.NoError(t, err)
		require.NotNil(t, doc)
	})

	t.Run("redeclaration", func(t *testing.T) {
		t.Run("valid redeclaration accepted", func(t *testing.T) {
			// §4.6: redeclaring lt with content "<" (via &#60;) is allowed.
			xml := `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ENTITY lt "&#60;">
]>
<root>&lt;</root>`
			_, err := helium.NewParser().Parse(t.Context(), []byte(xml))
			require.NoError(t, err)
		})

		t.Run("invalid redeclaration rejected", func(t *testing.T) {
			// §4.6: redeclaring lt with wrong content is a hard error.
			xml := `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ENTITY lt "X">
]>
<root>&lt;</root>`
			_, err := helium.NewParser().Parse(t.Context(), []byte(xml))
			require.Error(t, err)
			require.Contains(t, err.Error(), "redeclared")
		})

		t.Run("valid redeclaration with char ref accepted", func(t *testing.T) {
			// Content is &#60; (char ref for <), which resolves to <
			xml := `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ENTITY lt "&#60;">
  <!ENTITY gt "&#62;">
  <!ENTITY amp "&#38;">
  <!ENTITY apos "&#39;">
  <!ENTITY quot "&#34;">
]>
<root/>`
			_, err := helium.NewParser().Parse(t.Context(), []byte(xml))
			require.NoError(t, err)
		})

		t.Run("DTD.AddEntity rejects wrong content", func(t *testing.T) {
			doc := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)
			dtd, err := doc.CreateDTD()
			require.NoError(t, err)
			_, err = dtd.AddEntity("amp", enum.InternalGeneralEntity, "", "", "wrong")
			require.Error(t, err)
			require.Contains(t, err.Error(), "redeclared")
		})

		t.Run("DTD.AddEntity accepts correct content", func(t *testing.T) {
			doc := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)
			dtd, err := doc.CreateDTD()
			require.NoError(t, err)
			_, err = dtd.AddEntity("amp", enum.InternalGeneralEntity, "", "", "&")
			require.NoError(t, err)
		})

		t.Run("DTD.AddEntity accepts char ref content", func(t *testing.T) {
			doc := helium.NewDocument("1.0", "", helium.StandaloneImplicitNo)
			dtd, err := doc.CreateDTD()
			require.NoError(t, err)
			// &#60; resolves to <
			_, err = dtd.AddEntity("lt", enum.InternalGeneralEntity, "", "", "&#60;")
			require.NoError(t, err)
		})
	})
}

func TestEntityReference(t *testing.T) {
	t.Parallel()

	// CreateReference both for a
	// predefined entity (resolvable) and an undeclared name (no entity attached).
	t.Run("CreateReference with a declared entity", func(t *testing.T) {
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
	})

	// drives the "entity reference to unparsed entity"
	// error branch of parseEntityRef.
	t.Run("a reference to an unparsed entity", func(t *testing.T) {
		const src = `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!NOTATION gif SYSTEM "viewer">
  <!ENTITY img SYSTEM "img.gif" NDATA gif>
]>
<doc>&img;</doc>`

		_, err := helium.NewParser().Parse(t.Context(), []byte(src))
		require.Error(t, err)
	})

	// GetEntity's external-subset lookup and
	// the standalone short-circuit, plus GetParameterEntity.
	t.Run("lookup in the internal subset", func(t *testing.T) {
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
	})

	// Entity.URI's fallback to SystemID.
	t.Run("URI fallback", func(t *testing.T) {
		doc := helium.NewDocument("1.0", "UTF-8", helium.StandaloneImplicitNo)
		dtd, err := doc.CreateInternalSubset("doc", "", "")
		require.NoError(t, err)

		ext, err := dtd.AddEntity("e", enum.ExternalGeneralParsedEntity, "pub", "sys.ent", "")
		require.NoError(t, err)
		// No resolved URI set => falls back to SystemID.
		require.Equal(t, "sys.ent", ext.URI())
		require.Equal(t, "sys.ent", ext.SystemID())
		require.Equal(t, "pub", ext.ExternalID())
	})
}

func TestEntityInAttributeValue(t *testing.T) {
	t.Parallel()

	// the XML 1.0 attribute-value
	// well-formedness constraints ("No External Entity References", "No < in
	// Attribute Values") against an INDIRECT reference — a general entity whose OWN
	// replacement text is harmless but which transitively references an external or
	// unparsed entity, or reaches a literal '<'. Under SubstituteEntities(false) the
	// stricter attribute-value WFCs must fire on EVERY attribute occurrence,
	// independent of whether the entity was already expanded (and its weaker
	// element-content check recorded) in element content first.
	t.Run("an indirect reference", func(t *testing.T) {
		const head = "<!DOCTYPE r [\n" +
			"<!ELEMENT r ANY>\n" +
			"<!ELEMENT e EMPTY>\n" +
			"<!ATTLIST e a CDATA #IMPLIED>\n"

		testcases := []struct {
			name    string
			src     string
			wantMsg string
		}{
			{
				// The entity is expanded in element content FIRST (setting its
				// checked bit), then referenced from an attribute. The checked bit
				// must NOT suppress the attribute-value WFC re-validation.
				name: "external-indirect-after-content",
				src: head +
					"<!ENTITY ext SYSTEM \"nul\">\n" +
					"<!ENTITY outer \"&ext;\">\n" +
					"]>\n<r>&outer;<e a=\"&outer;\"/></r>",
				wantMsg: "attribute references external entity",
			},
			{
				name: "external-indirect-attr-only",
				src: head +
					"<!ENTITY ext SYSTEM \"nul\">\n" +
					"<!ENTITY outer \"&ext;\">\n" +
					"]>\n<r><e a=\"&outer;\"/></r>",
				wantMsg: "attribute references external entity",
			},
			{
				// An unparsed (NDATA) entity reached only through an attribute value.
				name: "unparsed-indirect-attr-only",
				src: head +
					"<!NOTATION gif PUBLIC \"gif\">\n" +
					"<!ENTITY pic SYSTEM \"nul\" NDATA gif>\n" +
					"<!ENTITY outer \"&pic;\">\n" +
					"]>\n<r><e a=\"&outer;\"/></r>",
				wantMsg: "entity reference to unparsed entity",
			},
			{
				// A nested entity whose replacement text contains a '<' (via a char
				// reference resolved into its stored content).
				name: "lessthan-indirect-attr-only",
				src: head +
					"<!ENTITY inner \"x&#60;y\">\n" +
					"<!ENTITY outer \"&inner;\">\n" +
					"]>\n<r><e a=\"&outer;\"/></r>",
				wantMsg: "'<' in entity is not allowed in attribute values",
			},
			{
				// The nested external entity is declared AFTER the ATTLIST default
				// value that transitively references it (forward reference). The WFC
				// classification must NOT be memoized against the incomplete entity
				// tables seen while the default value is parsed — a cached result
				// would let the document be accepted once the entity is declared.
				name: "external-indirect-attlist-default-forward",
				src: "<!DOCTYPE r [\n" +
					"<!ELEMENT r EMPTY>\n" +
					"<!ENTITY outer \"&ext;\">\n" +
					"<!ATTLIST r a CDATA \"&outer;\">\n" +
					"<!ENTITY ext SYSTEM \"nul\">\n" +
					"]>\n<r/>",
				wantMsg: "not defined",
			},
		}

		for _, tc := range testcases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				doc, err := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).
					DefaultDTDAttributes(true).SubstituteEntities(false).ValidateDTD(false).
					Parse(t.Context(), []byte(tc.src))
				require.Error(t, err, "indirect entity reference must violate the attribute-value WFC")
				require.Nil(t, doc, "no document on a fatal well-formedness error")
				require.Contains(t, err.Error(), tc.wantMsg)
			})
		}
	})

	// confirms the attribute-value WFC
	// walk does not over-reject: an indirect general entity whose transitive
	// replacement text is plain character data (no external/unparsed reference, no
	// '<') is accepted, including when referenced multiple times.
	t.Run("an indirect harmless reference", func(t *testing.T) {
		src := "<!DOCTYPE r [\n" +
			"<!ELEMENT r ANY>\n" +
			"<!ELEMENT e EMPTY>\n" +
			"<!ATTLIST e a CDATA #IMPLIED>\n" +
			"<!ENTITY inner \"value\">\n" +
			"<!ENTITY outer \"a&inner;b\">\n" +
			"]>\n<r>&outer;<e a=\"&outer; &outer;\"/></r>"

		doc, err := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).
			DefaultDTDAttributes(true).SubstituteEntities(false).ValidateDTD(false).
			Parse(t.Context(), []byte(src))
		require.NoError(t, err, "a harmless indirect entity must be accepted in an attribute value")
		require.NotNil(t, doc)
	})

	// a DTD attribute default
	// value that transitively references an external entity declared AFTER it (a
	// forward reference). The parse-time check cannot see the entity yet, so the
	// well-formedness violation is caught by the post-DTD re-validation once the
	// entity tables are complete. An external subset makes the early undefined
	// reference non-fatal, reproducing the case that would otherwise slip through.
	t.Run("a forward-referenced entity in a default", func(t *testing.T) {
		const doc = "<!DOCTYPE r SYSTEM \"d.dtd\" [\n" +
			"<!ELEMENT r EMPTY>\n" +
			"<!ENTITY outer \"&ext;\">\n" +
			"<!ATTLIST r a CDATA \"&outer;\">\n" +
			"<!ENTITY ext SYSTEM \"x\">\n" +
			"]>\n<r/>"
		fsys := fstest.MapFS{"d.dtd": {Data: []byte("<!-- external subset -->")}}

		got, err := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).
			DefaultDTDAttributes(true).SubstituteEntities(false).ValidateDTD(false).
			FS(fsys).Parse(t.Context(), []byte(doc))
		require.Error(t, err, "a forward-referenced external entity in a default value must be rejected")
		require.Nil(t, got)
		require.Contains(t, err.Error(), "attribute references external entity")
	})
}

func TestEntityValueRefValidation(t *testing.T) {
	t.Parallel()

	// ensures that a legitimately valid
	// EntityValue containing a direct character reference is still accepted and
	// stored literally (not expanded). The inert-placeholder treatment in the
	// reference-validation pass must not reject valid char refs.
	t.Run("a direct character reference is accepted", func(t *testing.T) {
		t.Run("standalone char ref", func(t *testing.T) {
			t.Parallel()
			const input = `<!DOCTYPE r [<!ENTITY e "x&#97;y">]><r/>`
			doc, err := helium.NewParser().Parse(t.Context(), []byte(input))
			require.NoError(t, err, "a standalone direct char ref must be accepted")
			require.NotNil(t, doc)

			ent, ok := doc.GetEntity("e")
			require.True(t, ok, "entity e must be declared")
			// A direct character reference is character data: it is resolved to its
			// character in the stored value ("&#97;" -> "a"), unlike a general
			// reference such as "&amp;" which is stored verbatim.
			require.Equal(t, "xay", string(ent.Content()),
				"direct char refs are character data, resolved in the stored value")
		})

		t.Run("predefined amp entity", func(t *testing.T) {
			t.Parallel()
			const input = `<!DOCTYPE r [<!ENTITY e "&amp;">]><r/>`
			doc, err := helium.NewParser().Parse(t.Context(), []byte(input))
			require.NoError(t, err, "a well-formed &amp; must be accepted")
			require.NotNil(t, doc)

			ent, ok := doc.GetEntity("e")
			require.True(t, ok, "entity e must be declared")
			require.Equal(t, "&amp;", string(ent.Content()),
				"general references must be stored literally, not expanded")
		})

		t.Run("direct ampersand char ref is inert", func(t *testing.T) {
			t.Parallel()
			// A direct "&#38;" is character data: it resolves to a literal '&' in
			// the stored value but does NOT combine with the following NameChars
			// into a synthesized general reference. The reference-validation pass
			// must therefore treat the direct char ref as inert and accept the
			// declaration, unlike "&broken" written directly (a malformed ref) or a
			// '&' re-introduced through a parameter entity.
			const input = `<!DOCTYPE r [<!ENTITY e "&#38;broken">]><r/>`
			doc, err := helium.NewParser().Parse(t.Context(), []byte(input))
			require.NoError(t, err,
				"a direct &#38; must be inert and not synthesize a malformed &broken ref")
			require.NotNil(t, doc)

			ent, ok := doc.GetEntity("e")
			require.True(t, ok, "entity e must be declared")
			require.Equal(t, "&broken", string(ent.Content()),
				"direct char refs are resolved in the stored value (&#38; -> &), not expanded as a general ref")
		})
	})

	// ensures that a well-formed general
	// reference in an EntityValue is accepted AND stored literally (not expanded):
	// the stored entity content must still contain "&amp; &good;" verbatim.
	t.Run("a valid general reference literal", func(t *testing.T) {
		const input = `<!DOCTYPE r [<!ENTITY e "&amp; &good;">]><r/>`

		doc, err := helium.NewParser().Parse(t.Context(), []byte(input))
		require.NoError(t, err, "well-formed general references in entity value must be accepted")
		require.NotNil(t, doc)

		ent, ok := doc.GetEntity("e")
		require.True(t, ok, "entity e must be declared")
		require.Equal(t, "&amp; &good;", string(ent.Content()),
			"general references must be stored literally, not expanded")
	})

	// ensures that a general reference inside an
	// EntityValue is syntax-checked: a missing semicolon must be rejected even
	// though the general reference itself is not expanded.
	t.Run("a malformed general reference", func(t *testing.T) {
		const input = `<!DOCTYPE r [<!ENTITY e "&broken">]><r/>`

		p := helium.NewParser()
		_, err := p.Parse(t.Context(), []byte(input))
		require.Error(t, err, "malformed general reference in entity value must be rejected")
	})

	// ensures that a DIRECT
	// character reference adjacent to a bare '&' or a name does not synthesize a
	// well-formed general reference. A direct char ref is character data; it must
	// never combine with surrounding text to manufacture a "&Name;". Both repros
	// would be wrongly accepted if direct char refs were resolved into the
	// validation stream instead of staying inert character data.
	t.Run("a malformed general reference after a character reference", func(t *testing.T) {
		t.Run("char ref completes a bare ampersand name", func(t *testing.T) {
			t.Parallel()
			// "&&#97;;" must NOT be read as "&a;": the first '&' is a bare ampersand
			// (malformed) and "&#97;" is character data.
			const input = `<!DOCTYPE r [<!ENTITY e "&&#97;;">]><r/>`
			_, err := helium.NewParser().Parse(t.Context(), []byte(input))
			require.Error(t, err,
				"a char ref must not complete a bare '&' into a general reference")
		})

		t.Run("char ref supplies a trailing semicolon", func(t *testing.T) {
			t.Parallel()
			// "&a&#59;" must NOT be read as "&a;": the trailing ';' is character data
			// (&#59;), not the terminator of a general reference.
			const input = `<!DOCTYPE r [<!ENTITY e "&a&#59;">]><r/>`
			_, err := helium.NewParser().Parse(t.Context(), []byte(input))
			require.Error(t, err,
				"a char ref must not supply the ';' that completes a general reference")
		})
	})

	// ensures that a malformed general
	// reference re-introduced through a parameter-entity reference is rejected, even
	// though the parameter entity itself only contributes a character reference. The
	// EntityValue "%amp;broken" with "%amp;" -> "&#38;" -> "&" expands to "&broken",
	// which is malformed and must be rejected (matching libxml2/xmllint), whereas a
	// direct "&#38;" in an EntityValue is character data and is accepted.
	t.Run("a malformed general reference via a PE", func(t *testing.T) {
		t.Run("internal subset", func(t *testing.T) {
			t.Parallel()
			// A PE reference inside an entity value in the internal subset violates
			// the PEs in Internal Subset WFC (XML §2.8) and is rejected outright,
			// before any general-reference well-formedness of the (would-be)
			// replacement text is considered. This holds whether the PE would expand
			// to a well-formed or a malformed general reference. The malformed
			// general-reference-via-PE path itself is exercised in the external
			// subset subtest, where PE references in entity values are permitted.
			wouldBeGood := `<!DOCTYPE r [<!ENTITY % p "&#38;amp;"><!ENTITY e "%p; ok">]><r/>`
			_, errGood := helium.NewParser().Parse(t.Context(), []byte(wouldBeGood))
			require.Error(t, errGood,
				"a PE reference in an internal-subset entity value is not well formed")
			require.Contains(t, errGood.Error(), "PEReferences forbidden in internal subset")

			wouldBeBad := `<!DOCTYPE r [<!ENTITY % amp "&#38;"><!ENTITY e "%amp;broken">]><r/>`
			_, errBad := helium.NewParser().Parse(t.Context(), []byte(wouldBeBad))
			require.Error(t, errBad,
				"a PE reference in an internal-subset entity value is not well formed")
			require.Contains(t, errBad.Error(), "PEReferences forbidden in internal subset")
		})

		t.Run("external subset", func(t *testing.T) {
			t.Parallel()
			// External DTD repro from the issue: a PE expands to "&" which combines
			// with following text into the malformed reference "&broken". The
			// malformed entity (e) must not be stored, while a control entity (c)
			// declared before it is stored, proving the parse reaches the entities and
			// the rejection is specific to the malformed declaration.
			//
			// The malformed per-declaration error in the external subset must now
			// surface as a top-level parse error, and is never swallowed.
			fsys := fstest.MapFS{
				dtdSystemID: {Data: []byte(
					`<!ENTITY c "control">` + "\n" +
						`<!ENTITY % amp "&#38;">` + "\n" +
						`<!ENTITY e "%amp;broken">`)},
			}
			const input = `<?xml version="1.0"?>` + "\n" +
				`<!DOCTYPE r SYSTEM "d.dtd"><r/>`

			_, err := helium.NewParser().BlockXXE(false).
				LoadExternalDTD(true).
				FS(fsys).
				Parse(t.Context(), []byte(input))
			require.Error(t, err, "malformed entity reference in an external subset declaration must surface as a parse error")
			require.Contains(t, err.Error(), "malformed entity reference in entity value")
		})
	})

	// the reference
	// validation in parseEntityValue does NOT perturb the entity-amplification
	// accounting. validateEntityValueRefs PE-expands the literal to scan for
	// malformed general references, and the real PE substitution that follows
	// expands the same literal again. Each expansion charges the amplification
	// counters; validateEntityValueRefs must snapshot and restore sizeentcopy so
	// the same parameter entity is not charged twice.
	//
	// The repro is an external DTD that declares a parameter entity just under the
	// 1 MiB baseline and a general entity referencing it via %p;. A single charge
	// stays under the baseline (no ratio check); a double charge crosses the
	// baseline and trips the amplification-ratio guard against the tiny main-
	// document input, which would reject the declaration. The entity therefore
	// being stored successfully is what proves the validation pass is side-effect
	// free.
	t.Run("validation is side-effect free", func(t *testing.T) {
		// Just under the 1 MiB amplification baseline (entityAllowedExpansion):
		// one expansion stays below it, a double charge crosses it.
		big := strings.Repeat("A", 1_000_000-100)
		dtd := `<!ENTITY c "control">` + "\n" +
			`<!ENTITY % p "` + big + `">` + "\n" +
			`<!ENTITY e "%p;">`
		fsys := fstest.MapFS{dtdSystemID: {Data: []byte(dtd)}}
		input := `<?xml version="1.0"?>` + "\n" +
			`<!DOCTYPE r SYSTEM "d.dtd"><r/>`

		doc, err := helium.NewParser().BlockXXE(false).
			LoadExternalDTD(true).
			FS(fsys).
			Parse(t.Context(), []byte(input))
		require.NoError(t, err, "a near-baseline parameter entity must not trip the amplification guard")
		require.NotNil(t, doc)

		_, cOK := doc.GetEntity("c")
		require.True(t, cOK, "control entity must be stored, proving the external DTD was loaded")

		ent, eOK := doc.GetEntity("e")
		require.True(t, eOK,
			"a PE-expanding entity just under the baseline must be stored; double-charging would reject it")
		require.Equal(t, len(big), len(ent.Content()),
			"the stored value is the single PE expansion, not a doubled one")
	})
}

func TestUndeclaredEntity(t *testing.T) {
	t.Parallel()

	// The SAME document is accepted by a non-validating processor: with an internal
	// PE reference present the undeclared entity is only a validity constraint, not a
	// well-formedness error, so it is a warning and the parse succeeds.
	t.Run("accepted when not validating", func(t *testing.T) {
		const src = `<!DOCTYPE foo [
<!ENTITY % pe "<!ENTITY ent1 'text'>">
%pe;
<!ELEMENT foo ANY>
]>
<foo>&ent2;</foo>`
		doc, err := helium.NewParser().
			SubstituteEntities(true).
			Parse(t.Context(), []byte(src))
		require.NoError(t, err, "a non-validating parse only warns on the undeclared entity")
		require.NotNil(t, doc)
	})

	// An internal parameter-entity reference downgrades an undeclared general entity
	// from a fatal well-formedness error to the "Entity Declared" VALIDITY
	// constraint. In a fully-internal DTD a validating processor must report it (W3C
	// xmlconf rmt-e3e-13).
	t.Run("a validity error when validating", func(t *testing.T) {
		const src = `<!DOCTYPE foo [
<!ENTITY % pe "<!ENTITY ent1 'text'>">
%pe;
<!ELEMENT foo ANY>
]>
<foo>&ent2;</foo>`
		_, err := helium.NewParser().
			SubstituteEntities(true).
			ValidateDTD(true).
			Parse(t.Context(), []byte(src))
		require.Error(t, err, "an undeclared entity must be a validity error when validating")
		require.Contains(t, err.Error(), "undeclared entity")
	})

	// When an EXTERNAL parameter entity is involved, helium cannot be certain the
	// entity is not declared in unread/incompletely-resolved external markup, so it
	// stays lenient even when validating — a still-undeclared entity is NOT promoted
	// to a fatal error (guards against over-rejecting a valid document; W3C
	// rmt-e2e-18).
	t.Run("lenient with an external PE", func(t *testing.T) {
		const src = `<!DOCTYPE foo [
<!ENTITY % pe SYSTEM "pe.ent">
%pe;
<!ELEMENT foo ANY>
]>
<foo>&ent2;</foo>`
		fsys := fstest.MapFS{"pe.ent": &fstest.MapFile{Data: []byte("<!-- external PE, declares nothing -->")}}
		doc, err := helium.NewParser().
			BlockXXE(false).
			LoadExternalDTD(true).
			SubstituteEntities(true).
			ValidateDTD(true).
			FS(fsys).
			Parse(t.Context(), []byte(src))
		require.NoError(t, err, "an external PE keeps the undeclared entity a non-fatal warning")
		require.NotNil(t, doc)
	})

	// A present-but-empty external ID (`SYSTEM ""`) still marks an external subset,
	// so the fully-internal precondition of the undeclared-entity VC is not met and
	// the undeclared entity stays a non-fatal warning even when validating.
	t.Run("lenient with an empty external ID", func(t *testing.T) {
		const src = `<!DOCTYPE foo SYSTEM "" [
<!ELEMENT foo ANY>
]>
<foo>&ent2;</foo>`
		doc, err := helium.NewParser().
			BlockXXE(false).
			LoadExternalDTD(true).
			SubstituteEntities(true).
			ValidateDTD(true).
			FS(fstest.MapFS{}).
			Parse(t.Context(), []byte(src))
		require.NoError(t, err, "a present-but-empty external ID keeps the undeclared entity a non-fatal warning")
		require.NotNil(t, doc)
	})

	// The same attribute-value case parses (only warns) when NOT validating.
	t.Run("in an attribute, accepted when not validating", func(t *testing.T) {
		const src = `<!DOCTYPE foo [
<!ENTITY % pe "<!ENTITY ent1 'text'>">
%pe;
<!ELEMENT foo EMPTY>
<!ATTLIST foo a CDATA #IMPLIED>
<!ENTITY bad "&ent2;">
]>
<foo a="&bad;"/>`
		doc, err := helium.NewParser().
			SubstituteEntities(true).
			Parse(t.Context(), []byte(src))
		require.NoError(t, err, "a non-validating parse only warns on the undeclared entity")
		require.NotNil(t, doc)
	})

	// The external-PE leniency holds for the attribute-value path too: with an
	// external PE involved, a still-undeclared entity referenced from an attribute
	// value stays a non-fatal warning even when validating.
	t.Run("in an attribute, lenient with an external PE", func(t *testing.T) {
		const src = `<!DOCTYPE foo [
<!ENTITY % pe SYSTEM "pe.ent">
%pe;
<!ELEMENT foo EMPTY>
<!ATTLIST foo a CDATA #IMPLIED>
<!ENTITY bad "&ent2;">
]>
<foo a="&bad;"/>`
		fsys := fstest.MapFS{"pe.ent": &fstest.MapFile{Data: []byte("<!-- external PE, declares nothing -->")}}
		doc, err := helium.NewParser().
			BlockXXE(false).
			LoadExternalDTD(true).
			SubstituteEntities(true).
			ValidateDTD(true).
			FS(fsys).
			Parse(t.Context(), []byte(src))
		require.NoError(t, err, "an external PE keeps the attribute-value undeclared entity a warning")
		require.NotNil(t, doc)
	})

	// The "Entity Declared" VC applies to attribute values too, not only element
	// content. An undeclared general entity nested inside an entity referenced from
	// an attribute value is a validity error under a fully-internal DTD when
	// validating — through the substitute-entities string path.
	t.Run("in an attribute, a validity error on the substitute path", func(t *testing.T) {
		const src = `<!DOCTYPE foo [
<!ENTITY % pe "<!ENTITY ent1 'text'>">
%pe;
<!ELEMENT foo EMPTY>
<!ATTLIST foo a CDATA #IMPLIED>
<!ENTITY bad "&ent2;">
]>
<foo a="&bad;"/>`
		_, err := helium.NewParser().
			SubstituteEntities(true).
			ValidateDTD(true).
			Parse(t.Context(), []byte(src))
		require.Error(t, err, "an undeclared entity in an attribute value must be a validity error")
		require.Contains(t, err.Error(), "undeclared entity")
	})

	// The same attribute-value case through the NON-substituting attribute-value WFC
	// walk (handleUndeclaredEntity) is equally a validity error when validating.
	t.Run("in an attribute, a validity error on the well-formedness walk", func(t *testing.T) {
		const src = `<!DOCTYPE foo [
<!ENTITY % pe "<!ENTITY ent1 'text'>">
%pe;
<!ELEMENT foo EMPTY>
<!ATTLIST foo a CDATA #IMPLIED>
<!ENTITY bad "&ent2;">
]>
<foo a="&bad;"/>`
		_, err := helium.NewParser().
			ValidateDTD(true).
			Parse(t.Context(), []byte(src))
		require.Error(t, err, "an undeclared entity in an attribute value must be a validity error")
		require.Contains(t, err.Error(), "undeclared entity")
	})

	t.Run("fatal without a DTD", func(t *testing.T) {
		// An undeclared general entity reference, with no DTD/external subset
		// and no parameter-entity references, is a fatal well-formedness error.
		xml := `<r>&bogus;</r>`

		doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
		require.Error(t, err, "undeclared entity with no DTD must be fatal")
		require.Nil(t, doc, "no document should be returned on a fatal error")
		require.ErrorIs(t, err, helium.ErrUndeclaredEntity, "error must be the undeclared-entity sentinel")

		var pe helium.ErrParseError
		require.True(t, errors.As(err, &pe), "error must be an ErrParseError")
		require.Equal(t, helium.ErrorLevelFatal, pe.Level, "undeclared entity must be a fatal-level error")
	})

	// An internal general entity whose replacement text references an undefined
	// entity, referenced from an attribute value, must be reported as a
	// well-formedness error — not crash the parser. getEntity returns a typed-nil
	// *Entity for the undefined inner entity, which a naive interface nil-check
	// misses. (W3C not-wf-sa-077.)
	t.Run("a nested undefined entity in an attribute value does not panic", func(t *testing.T) {
		const doc = `<!DOCTYPE doc [
<!ENTITY foo "&bar;">
]>
<doc a="&foo;"></doc>`

		require.NotPanics(t, func() {
			_, err := helium.NewParser().Parse(t.Context(), []byte(doc))
			require.Error(t, err, "a reference to an undefined entity must be a well-formedness error")
		})
	})
}
