package helium_test

import (
	"testing"
	"testing/fstest"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

func TestValidateAttributeType(t *testing.T) {
	t.Parallel()

	// ID/IDREF/NMTOKEN/ENTITY attribute-type
	// validation paths.
	t.Run("built-in types", func(t *testing.T) {
		const dtd = `<!DOCTYPE doc [
<!ELEMENT doc (item+)>
<!ELEMENT item EMPTY>
<!ATTLIST item
  id   ID    #REQUIRED
  ref  IDREF #IMPLIED
  tok  NMTOKEN #IMPLIED>
]>`

		// Valid: unique IDs, IDREF resolves, NMTOKEN well-formed.
		require.Empty(t, parseValidating(t,
			dtd+`<doc><item id="a"/><item id="b" ref="a" tok="x1"/></doc>`))

		// Duplicate ID is a validation error.
		require.NotEmpty(t, parseValidating(t,
			dtd+`<doc><item id="a"/><item id="a"/></doc>`))

		// IDREF pointing at a non-existent ID is a validation error.
		require.NotEmpty(t, parseValidating(t,
			dtd+`<doc><item id="a" ref="missing"/></doc>`))
	})

	// the enumeration token check.
	t.Run("enumeration token", func(t *testing.T) {
		const dtd = `<!DOCTYPE doc [
<!ELEMENT doc EMPTY>
<!ATTLIST doc kind (red|green|blue) "red">
]>`

		errs := parseValidating(t, dtd+`<doc kind="green"/>`)
		require.Empty(t, errs, "an enumerated value within the set is valid")

		errs = parseValidating(t, dtd+`<doc kind="purple"/>`)
		require.NotEmpty(t, errs, "a value outside the enumeration is a validation error")
	})

	t.Run("enumeration attribute", func(t *testing.T) {
		t.Run("valid value accepted", func(t *testing.T) {
			t.Parallel()

			xml := `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root EMPTY>
  <!ATTLIST root color (red|green|blue) #REQUIRED>
]>
<root color="green"/>`
			p := helium.NewParser().ValidateDTD(true).DefaultDTDAttributes(true)
			_, err := p.Parse(t.Context(), []byte(xml))
			require.NoError(t, err)
		})

		t.Run("invalid value rejected", func(t *testing.T) {
			t.Parallel()

			xml := `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root EMPTY>
  <!ATTLIST root color (red|green|blue) #REQUIRED>
]>
<root color="yellow"/>`
			collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
			p := helium.NewParser().ValidateDTD(true).DefaultDTDAttributes(true).ErrorHandler(collector)
			_, err := p.Parse(t.Context(), []byte(xml))

			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
			require.True(t, containsError(collector.Errors(), "not among the enumerated set"))
		})

		t.Run("default value used when absent", func(t *testing.T) {
			t.Parallel()

			xml := `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root EMPTY>
  <!ATTLIST root color (red|green|blue) "red">
]>
<root/>`
			p := helium.NewParser().ValidateDTD(true).DefaultDTDAttributes(true)
			_, err := p.Parse(t.Context(), []byte(xml))
			require.NoError(t, err)
		})
	})

	// NOTATION-typed attribute validation.
	t.Run("notation attribute", func(t *testing.T) {
		// A NOTATION attribute is not allowed on an EMPTY element (No Notation on
		// Empty Element VC), so the element uses ANY content.
		const dtd = `<!DOCTYPE doc [
<!NOTATION gif SYSTEM "viewer">
<!ELEMENT doc ANY>
<!ATTLIST doc kind NOTATION (gif) #IMPLIED>
]>`

		require.Empty(t, parseValidating(t, dtd+`<doc kind="gif"/>`))
		require.NotEmpty(t, parseValidating(t, dtd+`<doc kind="png"/>`))
	})

	t.Run("notation attribute against declared notations", func(t *testing.T) {
		t.Run("valid notation", func(t *testing.T) {
			t.Parallel()

			// A NOTATION attribute is not allowed on an EMPTY element (No Notation on
			// Empty Element VC), so the element uses ANY content.
			xml := `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root ANY>
  <!NOTATION gif SYSTEM "image/gif">
  <!NOTATION png SYSTEM "image/png">
  <!ATTLIST root fmt NOTATION (gif|png) #REQUIRED>
]>
<root fmt="gif"/>`
			p := helium.NewParser().ValidateDTD(true).DefaultDTDAttributes(true)
			_, err := p.Parse(t.Context(), []byte(xml))
			require.NoError(t, err)
		})

		t.Run("undeclared notation", func(t *testing.T) {
			t.Parallel()

			xml := `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root ANY>
  <!NOTATION gif SYSTEM "image/gif">
  <!ATTLIST root fmt NOTATION (gif|png) #REQUIRED>
]>
<root fmt="png"/>`
			collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
			p := helium.NewParser().ValidateDTD(true).DefaultDTDAttributes(true).ErrorHandler(collector)
			_, err := p.Parse(t.Context(), []byte(xml))

			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
			require.True(t, containsError(collector.Errors(), "undeclared notation"))
		})
	})

	// DTD (non-namespace-aware) NMTOKEN /
	// NMTOKENS validation accepts the colon, which is part of the XML 1.0 NameChar
	// production, so a value like `x:image` is a valid NMTOKEN and must not be
	// rejected. A token with a genuinely illegal char still fails.
	t.Run("NMTOKEN accepts a colon", func(t *testing.T) {
		const dtd = `<!DOCTYPE doc [
<!ELEMENT doc EMPTY>
<!ATTLIST doc
  tok  NMTOKEN  #IMPLIED
  toks NMTOKENS #IMPLIED>
]>`

		// An NMTOKEN value containing a colon is valid (DTD is not namespace-aware).
		require.Empty(t, parseValidating(t, dtd+`<doc tok="x:image"/>`),
			"a colon is a valid NMTOKEN NameChar")

		// An unprefixed NMTOKEN is unchanged.
		require.Empty(t, parseValidating(t, dtd+`<doc tok="x1"/>`),
			"an unprefixed NMTOKEN still validates")

		// Each space-separated NMTOKENS token may carry a colon.
		require.Empty(t, parseValidating(t, dtd+`<doc toks="x:image y:photo z1"/>`),
			"colons are valid in each NMTOKENS token")

		// A token with a genuinely illegal char (@) is still rejected.
		require.NotEmpty(t, parseValidating(t, dtd+`<doc tok="a@b"/>`),
			"an illegal NameChar is not a valid NMTOKEN")
		require.NotEmpty(t, parseValidating(t, dtd+`<doc toks="ok x@y"/>`),
			"an illegal NameChar in one NMTOKENS token is rejected")
	})
}

func TestValidateEntityAttribute(t *testing.T) {
	t.Parallel()

	t.Run("ENTITY", func(t *testing.T) {
		t.Run("valid unparsed entity", func(t *testing.T) {
			t.Parallel()

			xml := `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root EMPTY>
  <!NOTATION gif SYSTEM "image/gif">
  <!ENTITY logo SYSTEM "logo.gif" NDATA gif>
  <!ATTLIST root img ENTITY #REQUIRED>
]>
<root img="logo"/>`
			p := helium.NewParser().ValidateDTD(true).DefaultDTDAttributes(true)
			_, err := p.Parse(t.Context(), []byte(xml))
			require.NoError(t, err)
		})

		t.Run("undeclared entity", func(t *testing.T) {
			t.Parallel()

			xml := `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root EMPTY>
  <!ATTLIST root img ENTITY #REQUIRED>
]>
<root img="noSuchEntity"/>`
			collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
			p := helium.NewParser().ValidateDTD(true).DefaultDTDAttributes(true).ErrorHandler(collector)
			_, err := p.Parse(t.Context(), []byte(xml))

			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
			require.True(t, containsError(collector.Errors(), "undeclared entity"))
		})

		t.Run("wrong entity type (internal)", func(t *testing.T) {
			t.Parallel()

			xml := `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root EMPTY>
  <!ENTITY internalEnt "hello">
  <!ATTLIST root img ENTITY #REQUIRED>
]>
<root img="internalEnt"/>`
			collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
			p := helium.NewParser().ValidateDTD(true).DefaultDTDAttributes(true).ErrorHandler(collector)
			_, err := p.Parse(t.Context(), []byte(xml))

			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
			require.True(t, containsError(collector.Errors(), "not unparsed"))
		})
	})

	t.Run("ENTITIES", func(t *testing.T) {
		t.Run("valid multiple unparsed entities", func(t *testing.T) {
			t.Parallel()

			xml := `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root EMPTY>
  <!NOTATION gif SYSTEM "image/gif">
  <!ENTITY logo1 SYSTEM "logo1.gif" NDATA gif>
  <!ENTITY logo2 SYSTEM "logo2.gif" NDATA gif>
  <!ATTLIST root imgs ENTITIES #REQUIRED>
]>
<root imgs="logo1 logo2"/>`
			p := helium.NewParser().ValidateDTD(true).DefaultDTDAttributes(true)
			_, err := p.Parse(t.Context(), []byte(xml))
			require.NoError(t, err)
		})

		t.Run("one undeclared entity", func(t *testing.T) {
			t.Parallel()

			xml := `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root EMPTY>
  <!NOTATION gif SYSTEM "image/gif">
  <!ENTITY logo1 SYSTEM "logo1.gif" NDATA gif>
  <!ATTLIST root imgs ENTITIES #REQUIRED>
]>
<root imgs="logo1 noSuchEntity"/>`
			collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
			p := helium.NewParser().ValidateDTD(true).DefaultDTDAttributes(true).ErrorHandler(collector)
			_, err := p.Parse(t.Context(), []byte(xml))

			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
			require.True(t, containsError(collector.Errors(), "undeclared entity"))
		})
	})

	// the VC: Entity Name for ENTITY/ENTITIES
	// attributes whose referenced unparsed entity is declared in the EXTERNAL
	// subset. A validating processor loads the external subset even for a
	// standalone="yes" document, so the entity lookup must search both subsets
	// regardless of standalone (W3C xml sun/valid/sa03, sa04). Parsed and
	// undeclared references must still be rejected.
	t.Run("declared in the external subset", func(t *testing.T) {
		// External subset declaring an unparsed entity plus a parsed one, and the
		// ENTITY/ENTITIES attribute declarations that reference them.
		const extDTD = `<!ELEMENT doc EMPTY>
<!NOTATION nonce SYSTEM "nonce.exe">
<!ENTITY unparsed-1 SYSTEM "u1.dat" NDATA nonce>
<!ENTITY unparsed-2 PUBLIC "pub-u2" "u2.dat" NDATA nonce>
<!ENTITY parsed-1 SYSTEM "p1.xml">
<!ATTLIST doc one ENTITY #IMPLIED
              many ENTITIES #IMPLIED>`

		// The core fix: an ENTITY attribute referencing an externally-declared
		// unparsed entity validates even under standalone="yes".
		t.Run("ENTITY referencing external unparsed entity accepted", func(t *testing.T) {
			t.Parallel()
			errs, err := parseStandalone(t, `<?xml version="1.0" standalone="yes"?>
<!DOCTYPE doc SYSTEM "ext.dtd">
<doc one="unparsed-1"/>`, extDTD)
			require.NoError(t, err)
			require.Empty(t, errs)
		})

		t.Run("ENTITIES referencing external unparsed entities accepted", func(t *testing.T) {
			t.Parallel()
			errs, err := parseStandalone(t, `<?xml version="1.0" standalone="yes"?>
<!DOCTYPE doc SYSTEM "ext.dtd">
<doc many="unparsed-1 unparsed-2"/>`, extDTD)
			require.NoError(t, err)
			require.Empty(t, errs)
		})

		// Guard: an undeclared entity reference is still a validity error.
		t.Run("ENTITY referencing undeclared entity rejected", func(t *testing.T) {
			t.Parallel()
			errs, err := parseStandalone(t, `<?xml version="1.0" standalone="yes"?>
<!DOCTYPE doc SYSTEM "ext.dtd">
<doc one="nonexistent"/>`, extDTD)
			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
			require.True(t, containsError(errs, `references undeclared entity "nonexistent"`))
		})

		// Guard: a PARSED entity (external, no NDATA) is not a valid ENTITY value.
		t.Run("ENTITY referencing external parsed entity rejected", func(t *testing.T) {
			t.Parallel()
			errs, err := parseStandalone(t, `<?xml version="1.0" standalone="yes"?>
<!DOCTYPE doc SYSTEM "ext.dtd">
<doc one="parsed-1"/>`, extDTD)
			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
			require.True(t, containsError(errs, `references entity "parsed-1" which is not unparsed`))
		})

		// Guard: a PARSED entity declared in the INTERNAL subset is likewise not a
		// valid ENTITY value (the not-unparsed check spans both subsets).
		t.Run("ENTITY referencing internal parsed entity rejected", func(t *testing.T) {
			t.Parallel()
			errs, err := parseStandalone(t, `<?xml version="1.0" standalone="no"?>
<!DOCTYPE doc SYSTEM "ext.dtd" [
  <!ENTITY internal-parsed "text">
]>
<doc one="internal-parsed"/>`, extDTD)
			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
			require.True(t, containsError(errs, `references entity "internal-parsed" which is not unparsed`))
		})
	})
}

// the attribute-default checks.
func TestValidateRequiredAndFixedAttributes(t *testing.T) {
	t.Parallel()

	const dtd = `<!DOCTYPE doc [
<!ELEMENT doc EMPTY>
<!ATTLIST doc
  req CDATA #REQUIRED
  fix CDATA #FIXED "yes">
]>`

	valid := dtd + `<doc req="x" fix="yes"/>`
	errs := parseValidating(t, valid)
	require.Empty(t, errs, "all required/fixed attributes satisfied")

	// Missing required attribute.
	errs = parseValidating(t, dtd+`<doc fix="yes"/>`)
	require.NotEmpty(t, errs, "missing #REQUIRED attribute is a validation error")

	// Wrong value for a #FIXED attribute.
	errs = parseValidating(t, dtd+`<doc req="x" fix="no"/>`)
	require.NotEmpty(t, errs, "wrong #FIXED value is a validation error")
}

func TestAttributeValidityCompleteness(t *testing.T) {
	t.Parallel()

	// the per-instance attribute VCs
	// enforced in validateElementAttributes: Attribute Value Type (every present
	// attribute must be declared, for ordinary attributes and for xmlns/xmlns:*
	// namespace declarations) and Fixed Attribute Default (a #FIXED namespace
	// declaration must match). Each VC has a rejecting case and a valid near-miss so
	// the check does not over-reject.
	t.Run("per-instance validity constraints", func(t *testing.T) {
		// VC: Attribute Value Type — an ordinary undeclared attribute is invalid
		// (W3C ibm-invalid-P41-ibm41i01.xml).
		t.Run("undeclared ordinary attribute", func(t *testing.T) {
			t.Parallel()

			t.Run("rejected", func(t *testing.T) {
				t.Parallel()
				errs, err := parseValidatingDTD(t, `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root (#PCDATA|b)*>
  <!ELEMENT b (#PCDATA)>
  <!ATTLIST b attr2 (abc|def) "abc">
]>
<root><b attr1="value1" attr2="def">x</b></root>`)
				require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
				require.True(t, containsError(errs, "no declaration for attribute attr1"))
			})

			t.Run("declared attribute accepted", func(t *testing.T) {
				t.Parallel()
				_, err := parseValidatingDTD(t, `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root (#PCDATA|b)*>
  <!ELEMENT b (#PCDATA)>
  <!ATTLIST b attr1 CDATA #IMPLIED attr2 (abc|def) "abc">
]>
<root><b attr1="value1" attr2="def">x</b></root>`)
				require.NoError(t, err)
			})
		})

		// VC: Attribute Value Type applied to attributes in the reserved xml
		// namespace (W3C inv-required01/inv-required02). xml:space / xml:lang are
		// ordinary attributes that still require declaration.
		t.Run("undeclared reserved xml attribute", func(t *testing.T) {
			t.Parallel()

			t.Run("xml:space rejected", func(t *testing.T) {
				t.Parallel()
				errs, err := parseValidatingDTD(t, `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root EMPTY>
]>
<root xml:space='preserve'/>`)
				require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
				require.True(t, containsError(errs, "no declaration for attribute xml:space"))
			})

			t.Run("declared xml:space accepted", func(t *testing.T) {
				t.Parallel()
				_, err := parseValidatingDTD(t, `<?xml version="1.0"?>
<!DOCTYPE root [
  <!ELEMENT root EMPTY>
  <!ATTLIST root xml:space (default|preserve) #IMPLIED>
]>
<root xml:space='preserve'/>`)
				require.NoError(t, err)
			})
		})

		// A namespace declaration (xmlns:*) is EXEMPT from the "must be declared" VC:
		// a namespaced document may DTD-validate against a namespace-agnostic DTD that
		// never declares its xmlns attributes. This is the over-rejection guard for
		// the namespace-declaration path (helium diverges from a namespace-UNAWARE
		// validator here, so W3C hst-bh-005/hst-bh-006 stay out of scope).
		t.Run("undeclared namespace declaration accepted", func(t *testing.T) {
			t.Parallel()
			_, err := parseValidatingDTD(t, `<?xml version="1.0"?>
<!DOCTYPE r [
  <!ELEMENT r (p:a)>
  <!ELEMENT p:a EMPTY>
]>
<r xmlns:p='http://example.org'><p:a/></r>`)
			require.NoError(t, err)
		})

		// VC: Fixed Attribute Default applied to a default (xmlns) namespace
		// declaration (W3C attr08). A #FIXED xmlns must match the declared value.
		t.Run("fixed namespace declaration", func(t *testing.T) {
			t.Parallel()

			t.Run("differing value rejected", func(t *testing.T) {
				t.Parallel()
				errs, err := parseValidatingDTD(t, `<?xml version="1.0"?>
<!DOCTYPE palimpest [
  <!ELEMENT palimpest EMPTY>
  <!ATTLIST palimpest xmlns CDATA #FIXED "http://java.sun.com/historical">
]>
<palimpest xmlns="http://over.the.rainbow.com/somewhere"/>`)
				require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
				require.True(t, containsError(errs, "attribute xmlns has value"))
			})

			t.Run("matching value accepted", func(t *testing.T) {
				t.Parallel()
				_, err := parseValidatingDTD(t, `<?xml version="1.0"?>
<!DOCTYPE palimpest [
  <!ELEMENT palimpest EMPTY>
  <!ATTLIST palimpest xmlns CDATA #FIXED "http://java.sun.com/historical">
]>
<palimpest xmlns="http://java.sun.com/historical"/>`)
				require.NoError(t, err)
			})
		})
	})

	// is the over-rejection guard for the
	// "must be declared" VC: helium injects a synthetic xml:base attribute onto the
	// top-level elements of an external parsed entity to record the entity's base
	// URI. That attribute is not in the source and must not trip the Attribute Value
	// Type VC (W3C valid ext-sa-005/013, sun/valid/ext01). libxml2 tracks the entity
	// base without materializing an attribute, so it never flags one.
	t.Run("synthetic external-entity base is not flagged", func(t *testing.T) {
		fsys := fstest.MapFS{
			"e.ent": {Data: []byte(`<e/><e/>`)},
		}
		src := `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (e*)>
  <!ELEMENT e EMPTY>
  <!ENTITY e SYSTEM "e.ent">
]>
<doc>&e;</doc>`

		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, err := helium.NewParser().
			BaseURI("doc.xml").
			BlockXXE(false).
			LoadExternalDTD(true).
			DefaultDTDAttributes(true).
			SubstituteEntities(true).
			ValidateDTD(true).
			FS(fsys).
			ErrorHandler(collector).
			Parse(t.Context(), []byte(src))
		require.NoError(t, err, "external-entity elements carry a synthetic xml:base that must not be flagged as undeclared; errors=%v", collector.Errors())
	})

	// is the adversarial counterpart: only the
	// parser-injected xml:base is exempt. An AUTHORED xml:base on an external-entity
	// element — even one whose value coincidentally equals the entity's base URI, so
	// the parser suppresses its own injection — is a real attribute and must be
	// rejected when undeclared (VC: Attribute Value Type), matching
	// `xmllint --valid --noent`. The exemption is marker-based, not value-based.
	t.Run("authored external-entity base is flagged", func(t *testing.T) {
		// The entity's base URI resolves to "e.ent" (relative to BaseURI "doc.xml"),
		// and the authored xml:base value is exactly that, so a value-equality
		// exemption would wrongly accept it.
		fsys := fstest.MapFS{
			"e.ent": {Data: []byte(`<e xml:base="e.ent"/>`)},
		}
		src := `<?xml version="1.0"?>
<!DOCTYPE doc [
  <!ELEMENT doc (e*)>
  <!ELEMENT e EMPTY>
  <!ENTITY e SYSTEM "e.ent">
]>
<doc>&e;</doc>`

		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, err := helium.NewParser().
			BaseURI("doc.xml").
			BlockXXE(false).
			LoadExternalDTD(true).
			DefaultDTDAttributes(true).
			SubstituteEntities(true).
			ValidateDTD(true).
			FS(fsys).
			ErrorHandler(collector).
			Parse(t.Context(), []byte(src))
		require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
		require.True(t, containsError(collector.Errors(), "no declaration for attribute xml:base"),
			"authored undeclared xml:base must be flagged; errors=%v", collector.Errors())
	})
}
