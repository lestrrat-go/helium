package helium_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

func containsError(errs []error, substr string) bool {
	for _, e := range errs {
		if strings.Contains(e.Error(), substr) {
			return true
		}
	}
	return false
}

func TestExtSubsetLookup_ElementInExtSubset(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dtdPath := filepath.Join(dir, "ext.dtd")
	require.NoError(t, os.WriteFile(dtdPath, []byte(`<!ELEMENT root (child)>
<!ELEMENT child EMPTY>
<!ATTLIST child role CDATA #REQUIRED>`), 0600))

	xml := `<?xml version="1.0"?>
<!DOCTYPE root SYSTEM "` + dtdPath + `">
<root><child role="main"/></root>`

	p := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).ValidateDTD(true).FS(helium.PermissiveFS())
	_, err := p.Parse(t.Context(), []byte(xml))
	require.NoError(t, err, "validation should pass when declarations are in extSubset")
}

func TestExtSubsetLookup_EntityInExtSubset(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dtdPath := filepath.Join(dir, "ext.dtd")
	require.NoError(t, os.WriteFile(dtdPath, []byte(`<!ELEMENT root (#PCDATA)>
<!ENTITY extEnt "hello">`), 0600))

	xml := `<?xml version="1.0"?>
<!DOCTYPE root SYSTEM "` + dtdPath + `">
<root/>`

	p := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).FS(helium.PermissiveFS())
	doc, err := p.Parse(t.Context(), []byte(xml))
	require.NoError(t, err)

	ent, found := doc.GetEntity("extEnt")
	require.True(t, found, "entity in extSubset should be found")
	require.Equal(t, "hello", string(ent.Content()))
}

func TestExtSubsetLookup_AttributeInExtSubset(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dtdPath := filepath.Join(dir, "ext.dtd")
	require.NoError(t, os.WriteFile(dtdPath, []byte(`<!ELEMENT root (child)>
<!ELEMENT child EMPTY>
<!ATTLIST child role CDATA #REQUIRED>`), 0600))

	xml := `<?xml version="1.0"?>
<!DOCTYPE root SYSTEM "` + dtdPath + `">
<root><child/></root>`

	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	p := helium.NewParser().BlockXXE(false).LoadExternalDTD(true).ValidateDTD(true).ErrorHandler(collector).FS(helium.PermissiveFS())
	_, err := p.Parse(t.Context(), []byte(xml))

	require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
	require.True(t, containsError(collector.Errors(), "attribute role is required"))
}

func TestExtSubsetLookup_StandaloneYesPreventsExtSubset(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dtdPath := filepath.Join(dir, "ext.dtd")
	require.NoError(t, os.WriteFile(dtdPath, []byte(`<!ELEMENT root (child)>
<!ELEMENT child EMPTY>
<!ENTITY extEnt "hello">`), 0600))

	xml := `<?xml version="1.0" standalone="yes"?>
<!DOCTYPE root SYSTEM "` + dtdPath + `">
<root><child/></root>`

	p := helium.NewParser().LoadExternalDTD(true)
	doc, err := p.Parse(t.Context(), []byte(xml))
	require.NoError(t, err)

	_, found := doc.GetEntity("extEnt")
	require.False(t, found, "standalone=yes should prevent extSubset entity lookup")
}

func TestEnumerationAttributeValidation(t *testing.T) {
	t.Parallel()

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
}

func TestEntityAttributeValidation(t *testing.T) {
	t.Parallel()

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
}

func TestEntitiesAttributeValidation(t *testing.T) {
	t.Parallel()

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
}

func TestNotationAttributeValidation(t *testing.T) {
	t.Parallel()

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
}

// TestValidateSequenceContentModel exercises matchSeq for a valid and an invalid
// (a, b, c) sequence content model.
func TestValidateSequenceContentModel(t *testing.T) {
	t.Parallel()

	const dtd = `<!DOCTYPE doc [
<!ELEMENT doc (a, b, c)>
<!ELEMENT a (#PCDATA)>
<!ELEMENT b (#PCDATA)>
<!ELEMENT c (#PCDATA)>
]>`

	valid := dtd + `<doc><a/><b/><c/></doc>`
	errs := parseValidating(t, valid)
	require.Empty(t, errs, "a valid (a,b,c) sequence has no validation errors")

	// Out-of-order children violate the sequence.
	invalid := dtd + `<doc><b/><a/><c/></doc>`
	errs = parseValidating(t, invalid)
	require.NotEmpty(t, errs, "an out-of-order sequence is a validation error")
}

// TestValidateChoiceContentModel exercises matchOr with a repeated choice.
func TestValidateChoiceContentModel(t *testing.T) {
	t.Parallel()

	const dtd = `<!DOCTYPE doc [
<!ELEMENT doc (a | b)+>
<!ELEMENT a (#PCDATA)>
<!ELEMENT b (#PCDATA)>
]>`

	valid := dtd + `<doc><a/><b/><a/></doc>`
	errs := parseValidating(t, valid)
	require.Empty(t, errs, "a valid (a|b)+ choice has no validation errors")

	// An undeclared child element c is not part of the choice.
	invalid := dtd + `<doc><a/><c/></doc>`
	errs = parseValidating(t, invalid)
	require.NotEmpty(t, errs, "a child outside the choice is a validation error")
}

// TestValidateMixedContent exercises validateMixedContent.
func TestValidateMixedContent(t *testing.T) {
	t.Parallel()

	const dtd = `<!DOCTYPE doc [
<!ELEMENT doc (#PCDATA|em)*>
<!ELEMENT em (#PCDATA)>
]>`

	valid := dtd + `<doc>text <em>strong</em> more text</doc>`
	errs := parseValidating(t, valid)
	require.Empty(t, errs, "valid mixed content has no validation errors")

	// A child not allowed by the mixed model.
	invalid := dtd + `<doc>text <strong>bad</strong></doc>`
	errs = parseValidating(t, invalid)
	require.NotEmpty(t, errs, "an undeclared child in mixed content is a validation error")
}

// TestValidateRequiredAndFixedAttributes exercises the attribute-default checks.
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

// TestValidateEnumerationAttribute exercises the enumeration token check.
func TestValidateEnumerationAttribute(t *testing.T) {
	t.Parallel()

	const dtd = `<!DOCTYPE doc [
<!ELEMENT doc EMPTY>
<!ATTLIST doc kind (red|green|blue) "red">
]>`

	errs := parseValidating(t, dtd+`<doc kind="green"/>`)
	require.Empty(t, errs, "an enumerated value within the set is valid")

	errs = parseValidating(t, dtd+`<doc kind="purple"/>`)
	require.NotEmpty(t, errs, "a value outside the enumeration is a validation error")
}

// TestValidateUndeclaredElement exercises the "no declaration found" branch.
func TestValidateUndeclaredElement(t *testing.T) {
	t.Parallel()

	const src = `<!DOCTYPE doc [
<!ELEMENT doc (a)>
<!ELEMENT a (#PCDATA)>
]>
<doc><a/><undeclared/></doc>`

	errs := parseValidating(t, src)
	require.NotEmpty(t, errs, "an undeclared element is a validation error")
}

// TestValidateRootMismatch exercises the root-name-vs-DTD-name check.
func TestValidateRootMismatch(t *testing.T) {
	t.Parallel()

	const src = `<!DOCTYPE wrong [
<!ELEMENT root EMPTY>
]>
<root/>`

	errs := parseValidating(t, src)
	require.NotEmpty(t, errs, "root element not matching the DTD name is a validation error")
}

// TestValidateOptionalElementContent exercises the ? occurrence in a sequence.
func TestValidateOptionalElementContent(t *testing.T) {
	t.Parallel()

	const dtd = `<!DOCTYPE doc [
<!ELEMENT doc (a?, b)>
<!ELEMENT a (#PCDATA)>
<!ELEMENT b (#PCDATA)>
]>`

	// Optional a omitted is valid.
	errs := parseValidating(t, dtd+`<doc><b/></doc>`)
	require.Empty(t, errs, "omitting an optional element is valid")

	// Optional a present is also valid.
	errs = parseValidating(t, dtd+`<doc><a/><b/></doc>`)
	require.Empty(t, errs, "including an optional element is valid")
}

// collectingErrorHandler records every validation error delivered during a
// validating parse so tests can assert on the failure surface.
type collectingErrorHandler struct {
	errs []error
}

func (h *collectingErrorHandler) Handle(_ context.Context, err error) {
	h.errs = append(h.errs, err)
}

// parseValidating parses src with DTD validation enabled, routing validation
// errors into a collector.
func parseValidating(t *testing.T, src string) []error {
	t.Helper()
	h := &collectingErrorHandler{}
	_, err := helium.NewParser().
		ValidateDTD(true).
		ErrorHandler(h).
		Parse(t.Context(), []byte(src))
	// A validation failure returns ErrDTDValidationFailed; the document is still
	// returned. Parser-level (well-formedness) errors are a different matter and
	// are not expected for these well-formed inputs.
	if err != nil {
		require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
	}
	return h.errs
}

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
		require.Empty(t, parseValidating(t, dtd+`<doc><a/><b/><a/><b/></doc>`),
			"two (a,b) repetitions validate")
		require.NotEmpty(t, parseValidating(t, dtd+`<doc><a/><b/><a/></doc>`),
			"a trailing partial (a,b) repetition fails")
	})

	t.Run("optional trailing element (a,b?)", func(t *testing.T) {
		t.Parallel()
		const dtd = `<!DOCTYPE doc [
<!ELEMENT doc (a, b?)>
<!ELEMENT a (#PCDATA)>
<!ELEMENT b (#PCDATA)>
]>`
		require.Empty(t, parseValidating(t, dtd+`<doc><a/></doc>`),
			"the optional trailing b may be absent")
		require.Empty(t, parseValidating(t, dtd+`<doc><a/><b/></doc>`),
			"the optional trailing b may be present")
		require.NotEmpty(t, parseValidating(t, dtd+`<doc><a/><b/><b/></doc>`),
			"a second b exceeds the optional occurrence")
	})

	t.Run("zero-or-more choice (a|b)*", func(t *testing.T) {
		t.Parallel()
		const dtd = `<!DOCTYPE doc [
<!ELEMENT doc (a | b)*>
<!ELEMENT a (#PCDATA)>
<!ELEMENT b (#PCDATA)>
]>`
		require.Empty(t, parseValidating(t, dtd+`<doc></doc>`),
			"zero occurrences of the choice validate")
		require.Empty(t, parseValidating(t, dtd+`<doc><a/><a/><b/></doc>`),
			"several occurrences of the choice validate")
	})

	t.Run("optional element a?", func(t *testing.T) {
		t.Parallel()
		const dtd = `<!DOCTYPE doc [
<!ELEMENT doc (a?, b)>
<!ELEMENT a (#PCDATA)>
<!ELEMENT b (#PCDATA)>
]>`
		require.Empty(t, parseValidating(t, dtd+`<doc><b/></doc>`),
			"the optional leading a may be omitted")
		require.Empty(t, parseValidating(t, dtd+`<doc><a/><b/></doc>`),
			"the optional leading a may be present")
	})

	t.Run("one-or-more element a+", func(t *testing.T) {
		t.Parallel()
		const dtd = `<!DOCTYPE doc [
<!ELEMENT doc (a+)>
<!ELEMENT a (#PCDATA)>
]>`
		require.Empty(t, parseValidating(t, dtd+`<doc><a/><a/><a/></doc>`),
			"multiple a children validate a+")
		require.NotEmpty(t, parseValidating(t, dtd+`<doc></doc>`),
			"zero a children fails a+")
	})
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
	require.Empty(t, parseValidating(t, dtdMult+`<doc><a/><b/><a/><b/></doc>`))
	require.Empty(t, parseValidating(t, dtdMult+`<doc></doc>`)) // zero repetitions
	require.NotEmpty(t, parseValidating(t, dtdMult+`<doc><a/></doc>`))

	// (a, b)+ — one-or-more sequence group exercises matchSeq ElementContentPlus.
	const dtdPlus = `<!DOCTYPE doc [
<!ELEMENT doc (a, b)+>
<!ELEMENT a (#PCDATA)>
<!ELEMENT b (#PCDATA)>
]>`
	require.Empty(t, parseValidating(t, dtdPlus+`<doc><a/><b/></doc>`))
	require.Empty(t, parseValidating(t, dtdPlus+`<doc><a/><b/><a/><b/></doc>`))
	require.NotEmpty(t, parseValidating(t, dtdPlus+`<doc></doc>`))
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
	require.Empty(t, parseValidating(t, dtdMult+`<doc></doc>`))
	require.Empty(t, parseValidating(t, dtdMult+`<doc><a/><a/><b/></doc>`))

	// (a | b) once — exactly one of the two.
	const dtdOnce = `<!DOCTYPE doc [
<!ELEMENT doc (a | b)>
<!ELEMENT a (#PCDATA)>
<!ELEMENT b (#PCDATA)>
]>`
	require.Empty(t, parseValidating(t, dtdOnce+`<doc><a/></doc>`))
	require.Empty(t, parseValidating(t, dtdOnce+`<doc><b/></doc>`))
	require.NotEmpty(t, parseValidating(t, dtdOnce+`<doc><a/><b/></doc>`))
}

// TestValidateRepeatedElement exercises matchElement's Mult/Plus branches.
func TestValidateRepeatedElement(t *testing.T) {
	t.Parallel()

	const dtd = `<!DOCTYPE doc [
<!ELEMENT doc (a+, b*)>
<!ELEMENT a (#PCDATA)>
<!ELEMENT b (#PCDATA)>
]>`
	require.Empty(t, parseValidating(t, dtd+`<doc><a/></doc>`))
	require.Empty(t, parseValidating(t, dtd+`<doc><a/><a/><a/><b/><b/></doc>`))
	require.NotEmpty(t, parseValidating(t, dtd+`<doc><b/></doc>`)) // missing required a+
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
	require.Empty(t, parseValidating(t,
		dtd+`<doc><item id="a"/><item id="b" ref="a" tok="x1"/></doc>`))

	// Duplicate ID is a validation error.
	require.NotEmpty(t, parseValidating(t,
		dtd+`<doc><item id="a"/><item id="a"/></doc>`))

	// IDREF pointing at a non-existent ID is a validation error.
	require.NotEmpty(t, parseValidating(t,
		dtd+`<doc><item id="a" ref="missing"/></doc>`))
}

// TestValidateNmtokenColon verifies that DTD (non-namespace-aware) NMTOKEN /
// NMTOKENS validation accepts the colon, which is part of the XML 1.0 NameChar
// production, so a value like `x:image` is a valid NMTOKEN and must not be
// rejected. A token with a genuinely illegal char still fails.
func TestValidateNmtokenColon(t *testing.T) {
	t.Parallel()

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
}

// TestValidateNotationAttribute exercises NOTATION-typed attribute validation.
func TestValidateNotationAttribute(t *testing.T) {
	t.Parallel()

	// A NOTATION attribute is not allowed on an EMPTY element (No Notation on
	// Empty Element VC), so the element uses ANY content.
	const dtd = `<!DOCTYPE doc [
<!NOTATION gif SYSTEM "viewer">
<!ELEMENT doc ANY>
<!ATTLIST doc kind NOTATION (gif) #IMPLIED>
]>`

	require.Empty(t, parseValidating(t, dtd+`<doc kind="gif"/>`))
	require.NotEmpty(t, parseValidating(t, dtd+`<doc kind="png"/>`))
}

// TestValidateNoDTD verifies that requesting DTD validation on a document with
// neither an internal nor an external subset is a validity error (libxml2
// XML_DTD_NO_DTD "no DTD found!"), while the same document parsed without
// ValidateDTD succeeds and a document carrying a DTD still validates.
func TestValidateNoDTD(t *testing.T) {
	t.Parallel()

	const noDTD = `<?xml version="1.0"?>
<root><child/></root>`

	t.Run("ValidateDTD(true) with no DTD is invalid", func(t *testing.T) {
		h := &collectingErrorHandler{}
		_, err := helium.NewParser().
			ValidateDTD(true).
			ErrorHandler(h).
			Parse(t.Context(), []byte(noDTD))
		require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
		require.True(t, containsError(h.errs, "no DTD found"),
			"expected a 'no DTD found' validity error, got %v", h.errs)
	})

	t.Run("ValidateDTD(false) with no DTD succeeds", func(t *testing.T) {
		_, err := helium.NewParser().
			ValidateDTD(false).
			Parse(t.Context(), []byte(noDTD))
		require.NoError(t, err)
	})

	t.Run("ValidateDTD(true) with a DTD still validates", func(t *testing.T) {
		const withDTD = `<?xml version="1.0"?>
<!DOCTYPE root [
<!ELEMENT root (child)>
<!ELEMENT child EMPTY>
]>
<root><child/></root>`
		require.Empty(t, parseValidating(t, withDTD))
	})
}
