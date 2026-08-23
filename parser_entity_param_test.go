package helium_test

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/enum"
	"github.com/stretchr/testify/require"
)

func TestParameterEntity(t *testing.T) {
	t.Parallel()

	// parameter-entity declaration and reference in
	// the internal subset. A PE reference between declarations (supplying a whole
	// markup declaration) is well formed; a PE reference WITHIN a declaration — here
	// inside another entity's value — violates the "PEs in Internal Subset" WFC
	// (XML §2.8) and is fatal, matching libxml2.
	t.Run("declaration and use", func(t *testing.T) {
		// A parameter entity referenced BETWEEN declarations, supplying a complete
		// markup declaration. This is where PE references may occur in the internal
		// subset, so it must parse.
		const good = `<?xml version="1.0"?>
<!DOCTYPE doc [
<!ENTITY % decls "<!ELEMENT doc (#PCDATA)>">
%decls;
<!ENTITY greeting "Hello World">
]>
<doc>&greeting;</doc>`

		doc, err := helium.NewParser().SubstituteEntities(true).Parse(t.Context(), []byte(good))
		require.NoError(t, err, "a PE reference between declarations is well formed")
		require.NotNil(t, doc.DocumentElement())

		// A parameter entity referenced WITHIN a markup declaration (inside an
		// entity value) in the internal subset violates the PEs in Internal Subset
		// WFC and is a fatal error.
		const bad = `<?xml version="1.0"?>
<!DOCTYPE doc [
<!ENTITY % name "World">
<!ENTITY greeting "Hello %name;">
<!ELEMENT doc (#PCDATA)>
]>
<doc>&greeting;</doc>`

		_, err = helium.NewParser().SubstituteEntities(true).Parse(t.Context(), []byte(bad))
		require.Error(t, err, "a PE reference within a declaration in the internal subset is not well formed")
		require.Contains(t, err.Error(), "PEReferences forbidden in internal subset")
	})

	// XML §4.4.8 "Included as PE": in the
	// EXTERNAL subset a parameter-entity reference is recognized and included
	// ANYWHERE a markup declaration occurs — INSIDE or ADJACENT to the declaration,
	// not only between declarations — and its replacement text is padded with one
	// leading and one trailing space. Each DTD here is a valid external subset that
	// must parse AND apply the resulting declaration (W3C xmlconf valid/not-sa
	// 019/020/021 and the japanese/spec.dtd content-model & common-attribute
	// patterns).
	t.Run("in a markup declaration", func(t *testing.T) {
		const input = `<!DOCTYPE doc SYSTEM "d.dtd"><doc></doc>`

		testcases := []struct {
			name string
			dtd  string
			want string // substring the serialized <doc> must contain
		}{
			{
				// PE adjacent to an attribute type: `CDATA%e;` with %e; -> "'v1'". The
				// §4.4.8 leading space separates the type from the default value.
				name: "pe-adjacent-to-attribute-type",
				dtd:  "<!ELEMENT doc (#PCDATA)>\n<!ENTITY % e \"'v1'\">\n<!ATTLIST doc a1 CDATA%e;>\n",
				want: wantAttrA1V1,
			},
			{
				// PE supplies the element name immediately after <!ATTLIST (no space):
				// `<!ATTLIST%e;a1` with %e; -> "doc".
				name: "pe-supplies-element-name",
				dtd:  "<!ENTITY % e \"doc\">\n<!ELEMENT doc (#PCDATA)>\n<!ATTLIST%e;a1 CDATA \"v1\">\n",
				want: wantAttrA1V1,
			},
			{
				// PE supplies most of the ATTLIST body: %e; -> "doc a1 CDATA".
				name: "pe-supplies-attlist-body-head",
				dtd:  "<!ENTITY % e \"doc a1 CDATA\">\n<!ELEMENT doc (#PCDATA)>\n<!ATTLIST %e; \"v1\">\n",
				want: wantAttrA1V1,
			},
			{
				// PE supplies the whole attribute definition list.
				name: "pe-supplies-full-attlist-body",
				dtd:  "<!ENTITY % att \"a1 CDATA 'v1'\">\n<!ELEMENT doc (#PCDATA)>\n<!ATTLIST doc %att;>\n",
				want: wantAttrA1V1,
			},
			{
				// PE inside an element content model, recursively nested through an
				// empty PE (the japanese/spec.dtd `%class;` idiom).
				name: "pe-in-content-model",
				dtd: "<!ENTITY % local \"\">\n<!ENTITY % kids \"a %local;\">\n" +
					"<!ELEMENT a (#PCDATA)>\n<!ELEMENT head (#PCDATA)>\n" +
					"<!ELEMENT doc (head?, (%kids;)*)>\n<!ATTLIST doc a1 CDATA 'v1'>\n",
				want: wantAttrA1V1,
			},
			{
				// PE supplies an ATTLIST enumeration name list: `(%vals;)`.
				name: "pe-in-attribute-enumeration",
				dtd:  "<!ENTITY % vals \"red|green|blue\">\n<!ELEMENT doc (#PCDATA)>\n<!ATTLIST doc a1 (%vals;) \"red\">\n",
				want: `a1="red"`,
			},
			{
				// PE supplies a #FIXED default value.
				name: "pe-in-fixed-default",
				dtd:  "<!ENTITY % v \"'red'\">\n<!ELEMENT doc (#PCDATA)>\n<!ATTLIST doc a1 CDATA #FIXED %v;>\n",
				want: `a1="red"`,
			},
			{
				// PE supplies a NOTATION type name list, plus the notation declarations.
				name: "pe-in-notation-type-list",
				dtd: "<!ENTITY % ns \"gif|jpg\">\n<!ELEMENT doc (#PCDATA)>\n" +
					"<!NOTATION gif SYSTEM \"gif\">\n<!NOTATION jpg SYSTEM \"jpg\">\n" +
					"<!ATTLIST doc t NOTATION (%ns;) #IMPLIED a1 CDATA 'v1'>\n",
				want: wantAttrA1V1,
			},
			{
				// PE supplies a NOTATION declaration's SYSTEM literal: `SYSTEM %sid;`.
				name: "pe-in-notation-decl-system-id",
				dtd: "<!ENTITY % sid \"'g.dtd'\">\n<!ELEMENT doc (#PCDATA)>\n" +
					"<!NOTATION gif SYSTEM %sid;>\n<!ATTLIST doc a1 CDATA 'v1'>\n",
				want: wantAttrA1V1,
			},
		}

		for _, tc := range testcases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				fsys := fstest.MapFS{dtdSystemID: {Data: []byte(tc.dtd)}}
				doc, err := helium.NewParser().BlockXXE(false).
					LoadExternalDTD(true).DefaultDTDAttributes(true).SubstituteEntities(true).
					ValidateDTD(true).FS(fsys).Parse(t.Context(), []byte(input))
				require.NoError(t, err, "a valid external subset with a PE in/adjacent to a markup declaration must parse")
				require.NotNil(t, doc)

				str, werr := helium.WriteString(doc.DocumentElement())
				require.NoError(t, werr)
				require.Contains(t, str, tc.want, "the declaration assembled across the PE boundary must be applied")
			})
		}
	})

	// a parameter entity supplying an
	// internal general entity's value in the external subset:
	// `<!ENTITY greet %pub;>` with %pub; -> "'hello'" declares greet as an INTERNAL
	// entity whose value is `hello` (not an empty external entity), so a later
	// `&greet;` reference expands to `hello`.
	t.Run("supplying an entity value", func(t *testing.T) {
		dtd := "<!ENTITY % pub \"'hello'\">\n<!ELEMENT doc (#PCDATA)>\n<!ENTITY greet %pub;>\n"
		const input = `<!DOCTYPE doc SYSTEM "d.dtd"><doc>&greet;</doc>`
		fsys := fstest.MapFS{dtdSystemID: {Data: []byte(dtd)}}

		doc, err := helium.NewParser().BlockXXE(false).
			LoadExternalDTD(true).SubstituteEntities(true).
			FS(fsys).Parse(t.Context(), []byte(input))
		require.NoError(t, err)
		require.NotNil(t, doc)

		str, werr := helium.WriteString(doc.DocumentElement())
		require.NoError(t, werr)
		require.Equal(t, "<doc>hello</doc>", str, "the PE-supplied internal entity value must expand, not become an empty external entity")
	})

	// the case where a
	// parameter-entity DECLARATION (<!ENTITY % ...>) is the FIRST declaration of an
	// external subset. The '%' marker following <!ENTITY must not be mis-parsed as a
	// parameter-entity REFERENCE, which previously produced a spurious
	// "space required at line 1, column 2" (the psDTD-only marker guard did not
	// apply because parseMarkupDecl sets psDTD only AFTER the first declaration).
	t.Run("declared first in the external subset", func(t *testing.T) {
		const input = `<?xml version="1.0"?>` + "\n" +
			`<!DOCTYPE doc SYSTEM "d.dtd"><doc/>`

		testcases := []struct {
			name string
			dtd  string
		}{
			{
				// Internal PE declared first, then referenced to supply the element decl.
				name: "internal-pe-decl-first",
				dtd:  `<!ENTITY % pe "<!ELEMENT doc EMPTY>">` + "\n" + `%pe;`,
			},
			{
				// External PE declared first (never referenced): the declaration alone
				// must not trip the marker guard.
				name: "external-pe-decl-first",
				dtd:  `<!ENTITY % bad SYSTEM "bad.ent">` + "\n" + `<!ELEMENT doc EMPTY>`,
			},
			{
				// PE declaration is the first declaration INSIDE an INCLUDE section.
				name: "pe-decl-first-in-include",
				dtd: `<![INCLUDE[` + "\n" + `<!ENTITY % rootel "<!ELEMENT doc EMPTY>">` + "\n" +
					`]]>` + "\n" + `%rootel;`,
			},
		}

		for _, tc := range testcases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				fsys := fstest.MapFS{dtdSystemID: {Data: []byte(tc.dtd)}}
				doc, err := helium.NewParser().BlockXXE(false).
					LoadExternalDTD(true).SubstituteEntities(true).FS(fsys).
					Parse(t.Context(), []byte(input))
				require.NoError(t, err, "a parameter-entity declaration as the first external-subset declaration must parse")
				require.NotNil(t, doc)
			})
		}
	})

	// parses internal-subset DTDs that exercise
	// the entity-value and parameter-entity-reference parser paths: entity values
	// containing character and general-entity references, a parameter entity declared
	// and then referenced inside the subset, and a mixed-content (#PCDATA|x|y)*
	// declaration with several alternatives.
	t.Run("entity values and parameter references in a DTD", func(t *testing.T) {
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
	})
}

func TestParameterEntityBoundary(t *testing.T) {
	t.Parallel()

	t.Run("an element declaration split across a PE boundary", func(t *testing.T) {
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

	t.Run("an attribute-list declaration split across a PE boundary", func(t *testing.T) {
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

	t.Run("a well-nested PE parses", func(t *testing.T) {
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

	// the XML validity constraints
	// that a markup declaration (and a parenthesized content-model group) must start
	// and stop in the SAME entity: a closing '>' or ')' supplied by a DIFFERENT
	// parameter-entity replacement text than the one that opened the declaration/group
	// is a boundary violation and must be rejected (W3C xmlconf invalid cases E14,
	// invalid/002, ibm P49/P50/P51 invalid). Rejecting these must NOT regress the
	// valid PE-in-markup documents above.
	t.Run("markup boundary violation", func(t *testing.T) {
		const input = `<!DOCTYPE doc SYSTEM "d.dtd"><doc></doc>`

		testcases := []struct {
			name string
			dtd  string
		}{
			{
				// The ATTLIST closing '>' comes from inside the PE (W3C errata E14).
				name: "attlist-close-in-pe",
				dtd:  "<!ELEMENT doc ANY>\n<!ENTITY % e \"a1 CDATA #IMPLIED>\">\n<!ATTLIST doc %e;\n",
			},
			{
				// Content-model '(' in a PE, ')' in the containing DTD (W3C invalid/002).
				name: "content-model-open-in-pe-close-in-dtd",
				dtd:  "<!ENTITY % e \"(#PCDATA\">\n<!ELEMENT doc %e;)>\n",
			},
			{
				// Content-model '(' in one PE, ')' in another (W3C ibm P49 invalid).
				name: "content-model-group-split-across-pes",
				dtd: "<!ELEMENT a EMPTY>\n<!ELEMENT b (#PCDATA)>\n" +
					"<!ENTITY % choice1 \"(a|b\">\n<!ENTITY % choice2 \"|c)\">\n" +
					"<!ELEMENT c ANY>\n<!ELEMENT child1 %choice1;%choice2; >\n",
			},
			{
				// The <!ENTITY> closing '>' comes from a PE (%close; -> ">").
				name: "entity-close-in-pe",
				dtd:  "<!ELEMENT doc (#PCDATA)>\n<!ENTITY % close \">\">\n<!ENTITY greet 'hi'%close;\n",
			},
			{
				// The <!NOTATION> closing '>' comes from a PE (%close; -> ">").
				name: "notation-close-in-pe",
				dtd:  "<!ELEMENT doc (#PCDATA)>\n<!ENTITY % close \">\">\n<!NOTATION gif SYSTEM 'gif'%close;\n",
			},
		}

		for _, tc := range testcases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				fsys := fstest.MapFS{dtdSystemID: {Data: []byte(tc.dtd)}}
				_, err := helium.NewParser().BlockXXE(false).
					LoadExternalDTD(true).DefaultDTDAttributes(true).SubstituteEntities(true).
					ValidateDTD(true).FS(fsys).Parse(t.Context(), []byte(input))
				require.Error(t, err, "a markup declaration / group that crosses a PE boundary must be rejected")
			})
		}
	})

	// asserts the INTERNAL subset stays
	// byte-identical to origin: a parameter entity must NOT supply part of (or be
	// adjacent to) a markup declaration there (WFC: PEs in Internal Subset). PE
	// expansion inside markup is EXTERNAL-subset-only; a '%' where an "S", token, or
	// '>' is required is rejected exactly as before, never silently accepted.
	t.Run("a PE in an internal-subset markup declaration is rejected", func(t *testing.T) {
		testcases := []struct {
			name    string
			doctype string
		}{
			{
				// A '%' where an "S" or '>' is required in an <!ENTITY> declaration.
				name:    "stray-percent-in-entity-decl",
				doctype: `<!DOCTYPE doc [<!ELEMENT doc EMPTY><!ENTITY e SYSTEM "x"%p;>]><doc/>`,
			},
			{
				// A PE supplying the <!ATTLIST> body in the internal subset.
				name:    "pe-supplies-attlist-body",
				doctype: `<!DOCTYPE doc [<!ELEMENT doc EMPTY><!ENTITY % att "a1 CDATA 'v1'"><!ATTLIST doc %att;>]><doc/>`,
			},
			{
				// A PE supplying an <!ELEMENT> content model in the internal subset.
				name:    "pe-supplies-content-model",
				doctype: `<!DOCTYPE doc [<!ENTITY % m "#PCDATA"><!ELEMENT doc (%m;)>]><doc/>`,
			},
		}

		for _, tc := range testcases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				_, err := helium.NewParser().LoadExternalDTD(true).SubstituteEntities(true).
					Parse(t.Context(), []byte(tc.doctype))
				require.Error(t, err, "a parameter entity inside a markup declaration in the internal subset must be rejected")
			})
		}
	})

	// a parameter entity used
	// inside the internal subset to pull in further declarations.
	t.Run("internal-subset inclusion", func(t *testing.T) {
		const xml = `<?xml version="1.0"?>
<!DOCTYPE root [
<!ENTITY % decls "<!ELEMENT root (#PCDATA)><!ENTITY inner 'inner-value'>">
%decls;
]>
<root/>`

		doc, err := helium.NewParser().Parse(t.Context(), []byte(xml))
		require.NoError(t, err)
		_, found := doc.GetEntity("inner")
		require.True(t, found, "entity declared via internal-subset PE inclusion must be present")
	})

	// the "PEs in Internal
	// Subset" WFC (XML §2.8): in the internal DTD subset a parameter-entity
	// reference must not occur WITHIN a markup declaration, only where a markup
	// declaration can occur. A PE reference inside an EntityValue literal is a fatal
	// well-formedness error in the internal subset, while the same construct is
	// permitted in the external subset (and within an external parameter entity).
	// This matches libxml2's xmlExpandPEsInEntityValue PARSER_EXTERNAL gate and
	// closes W3C not-wf cases not-wf-sa-160, not-wf-sa-162, ibm-not-wf-P29-ibm29n04,
	// ibm-not-wf-P69-ibm69n06 and ibm-not-wf-P69-ibm69n07.
	{
		newParser := func() helium.Parser {
			return helium.NewParser().
				BlockXXE(false).
				LoadExternalDTD(true).
				DefaultDTDAttributes(true).
				SubstituteEntities(true).
				FS(helium.PermissiveFS())
		}

		t.Run("a PE reference inside an internal-subset entity value is rejected", func(t *testing.T) {
			t.Parallel()

			notWF := map[string]string{
				// not-wf-sa-160: general entity value contains a PE reference.
				"general entity value": "<!DOCTYPE doc [\n" +
					"<!ELEMENT doc (#PCDATA)>\n" +
					"<!ENTITY % e \"\">\n" +
					"<!ENTITY foo \"%e;\">\n" +
					"]>\n<doc></doc>",
				// not-wf-sa-162: parameter entity value contains a PE reference.
				"parameter entity value": "<!DOCTYPE doc [\n" +
					"<!ELEMENT doc (#PCDATA)>\n" +
					"<!ENTITY % e1 \"\">\n" +
					"<!ENTITY % e2 \"%e1;\">\n" +
					"]>\n<doc></doc>",
				// ibm-not-wf-P29-ibm29n04: PE reference inside an entity declaration.
				"non-empty PE in entity value": "<!DOCTYPE animal [\n" +
					"<!ELEMENT animal ANY>\n" +
					"<!ENTITY % parameterE \"A leopard\">\n" +
					"<!ENTITY content \"%parameterE;\">\n" +
					"]>\n<animal>stuff</animal>",
				// ibm-not-wf-P69-ibm69n06: the recursive PE cycle is unreachable
				// because the PE reference inside <!ENTITY bbb "%paaa;"> is rejected
				// first.
				"recursive PE via general entity": "<!DOCTYPE root [\n" +
					"<!ELEMENT root (#PCDATA)>\n" +
					"<!ENTITY % paaa \"&bbb;\">\n" +
					"<!ENTITY bbb \"%paaa;\">\n" +
					"]>\n<root/>",
			}

			for name, src := range notWF {
				t.Run(name, func(t *testing.T) {
					t.Parallel()
					_, err := newParser().Parse(t.Context(), []byte(src))
					require.Error(t, err, "a PE reference within an internal-subset declaration is not well formed")
					require.Contains(t, err.Error(), "PEReferences forbidden in internal subset")
				})
			}
		})

		t.Run("a PE reference inside an external-subset entity value is permitted", func(t *testing.T) {
			t.Parallel()

			// The very same construct (a PE reference inside an entity value) is
			// legal in the external subset, where it must still expand normally.
			fsys := fstest.MapFS{
				"pe-ext.dtd": &fstest.MapFile{Data: []byte(
					"<!ENTITY % e \"expanded\">\n<!ENTITY foo \"%e;\">\n")},
			}
			const src = "<!DOCTYPE doc SYSTEM \"pe-ext.dtd\" [\n<!ELEMENT doc (#PCDATA)>\n]>\n<doc>&foo;</doc>"

			p := helium.NewParser().
				BlockXXE(false).
				LoadExternalDTD(true).
				DefaultDTDAttributes(true).
				SubstituteEntities(true).
				FS(fsys)
			doc, err := p.Parse(t.Context(), []byte(src))
			require.NoError(t, err, "a PE reference in an external-subset entity value is well formed")
			require.Equal(t, "expanded", string(doc.DocumentElement().Content()),
				"the external-subset PE must still expand inside the entity value")
		})

		t.Run("PE between declarations in internal subset is well formed", func(t *testing.T) {
			t.Parallel()

			// A PE reference BETWEEN declarations (supplying a complete markup
			// declaration) is where PE references may occur in the internal subset,
			// so it must parse.
			const src = "<!DOCTYPE doc [\n" +
				"<!ENTITY % decls \"<!ELEMENT doc (#PCDATA)>\">\n" +
				"%decls;\n" +
				"<!ENTITY greeting \"hi\">\n" +
				"]>\n<doc>&greeting;</doc>"

			doc, err := newParser().Parse(t.Context(), []byte(src))
			require.NoError(t, err, "a PE reference between declarations is well formed in the internal subset")
			require.Equal(t, "hi", string(doc.DocumentElement().Content()))
		})
	}
}
