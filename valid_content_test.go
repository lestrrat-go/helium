package helium_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// parseValidating parses src as a validating processor with entity substitution,
// returning the DTD-validation errors collected (nil error means the document is
// valid).
func parseECValidate(t *testing.T, src string) error {
	t.Helper()
	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	_, err := helium.NewParser().
		SubstituteEntities(true).
		ValidateDTD(true).
		DefaultDTDAttributes(true).
		ErrorHandler(collector).
		Parse(t.Context(), []byte(src))
	return err
}

func TestValidateContentModel(t *testing.T) {
	t.Parallel()

	// matchSeq for a valid and an invalid
	// (a, b, c) sequence content model.
	t.Run("sequence", func(t *testing.T) {
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
	})

	// matchOr with a repeated choice.
	t.Run("choice", func(t *testing.T) {
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
	})

	// validateMixedContent.
	t.Run("mixed content", func(t *testing.T) {
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
	})

	// the ? occurrence in a sequence.
	t.Run("optional element", func(t *testing.T) {
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
	})

	// matchElement's Mult/Plus branches.
	t.Run("repeated element", func(t *testing.T) {
		const dtd = `<!DOCTYPE doc [
<!ELEMENT doc (a+, b*)>
<!ELEMENT a (#PCDATA)>
<!ELEMENT b (#PCDATA)>
]>`
		require.Empty(t, parseValidating(t, dtd+`<doc><a/></doc>`))
		require.Empty(t, parseValidating(t, dtd+`<doc><a/><a/><a/><b/><b/></doc>`))
		require.NotEmpty(t, parseValidating(t, dtd+`<doc><b/></doc>`)) // missing required a+
	})

	// drives the occurrence variants of matchSeq
	// and matchOr (optional, zero-or-more, one-or-more) plus nested optional
	// sequences, exercising the seq/or matcher branches in valid.go that the simpler
	// once-only models did not reach.
	t.Run("sequence occurrences", func(t *testing.T) {
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
	})

	// matchSeq's Mult and Opt
	// occurrence branches via grouped sequence content models.
	t.Run("grouped sequence occurrences", func(t *testing.T) {
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
	})

	// matchOr's Mult/Opt/Once branches.
	t.Run("choice occurrences", func(t *testing.T) {
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
	})

	// content models where a
	// greedy inner repetition would starve a later iteration of an outer
	// repetition, which the greedy recursive-descent matcher cannot resolve on its
	// own. The exact reachability fallback must accept the language members and
	// still reject genuine non-members.
	t.Run("nested repetition backtracking", func(t *testing.T) {
		t.Run("(lhs,(rhs,(com|wfc|vc)*)+)", func(t *testing.T) {
			t.Parallel()
			const dtd = `<!DOCTYPE prod [
<!ELEMENT prod (lhs,(rhs,(com|wfc|vc)*)+)>
<!ELEMENT lhs EMPTY>
<!ELEMENT rhs EMPTY>
<!ELEMENT com EMPTY>
<!ELEMENT wfc EMPTY>
<!ELEMENT vc EMPTY>
]>`
			// Single iteration of the outer plus-group.
			require.Empty(t, parseValidating(t, dtd+`<prod><lhs/><rhs/><com/></prod>`),
				"single iteration is valid")

			// Two iterations: [rhs com] then [rhs vc]. The inner (com|wfc|vc)* of
			// the first iteration must NOT greedily swallow the second rhs.
			require.Empty(t, parseValidating(t, dtd+`<prod><lhs/><rhs/><com/><rhs/><vc/></prod>`),
				"two iterations is valid")

			// A bare rhs with no trailing choice items (choice group is *).
			require.Empty(t, parseValidating(t, dtd+`<prod><lhs/><rhs/></prod>`),
				"rhs with zero choice items is valid")

			// Missing the mandatory leading lhs.
			require.NotEmpty(t, parseValidating(t, dtd+`<prod><rhs/><com/></prod>`),
				"missing lhs is invalid")

			// A trailing element outside the model.
			require.NotEmpty(t, parseValidating(t, dtd+`<prod><lhs/><rhs/><com/><bogus/></prod>`),
				"trailing undeclared element is invalid")

			// Only lhs: the outer group requires at least one rhs.
			require.NotEmpty(t, parseValidating(t, dtd+`<prod><lhs/></prod>`),
				"missing mandatory rhs is invalid")
		})

		t.Run("(a,(b,c*)+)", func(t *testing.T) {
			t.Parallel()
			const dtd = `<!DOCTYPE r [
<!ELEMENT r (a,(b,c*)+)>
<!ELEMENT a EMPTY>
<!ELEMENT b EMPTY>
<!ELEMENT c EMPTY>
]>`
			require.Empty(t, parseValidating(t, dtd+`<r><a/><b/></r>`), "a,b valid")
			require.Empty(t, parseValidating(t, dtd+`<r><a/><b/><c/><b/><c/><c/></r>`),
				"a then (b,c*) twice valid")
			require.Empty(t, parseValidating(t, dtd+`<r><a/><b/><c/><c/></r>`), "a,b,c,c valid")
			require.NotEmpty(t, parseValidating(t, dtd+`<r><a/></r>`), "missing b invalid")
			require.NotEmpty(t, parseValidating(t, dtd+`<r><a/><c/></r>`), "c before b invalid")
		})

		t.Run("(a|b)+", func(t *testing.T) {
			t.Parallel()
			const dtd = `<!DOCTYPE r [
<!ELEMENT r (a|b)+>
<!ELEMENT a EMPTY>
<!ELEMENT b EMPTY>
]>`
			require.Empty(t, parseValidating(t, dtd+`<r><a/></r>`), "single a valid")
			require.Empty(t, parseValidating(t, dtd+`<r><a/><b/><a/><a/><b/></r>`), "mixed valid")
			require.NotEmpty(t, parseValidating(t, dtd+`<r></r>`), "empty invalid (needs one)")
			require.NotEmpty(t, parseValidating(t, dtd+`<r><a/><c/></r>`), "undeclared c invalid")
		})

		t.Run("((a,b)|c)*", func(t *testing.T) {
			t.Parallel()
			const dtd = `<!DOCTYPE r [
<!ELEMENT r ((a,b)|c)*>
<!ELEMENT a EMPTY>
<!ELEMENT b EMPTY>
<!ELEMENT c EMPTY>
]>`
			require.Empty(t, parseValidating(t, dtd+`<r></r>`), "empty valid (star)")
			require.Empty(t, parseValidating(t, dtd+`<r><c/></r>`), "single c valid")
			require.Empty(t, parseValidating(t, dtd+`<r><a/><b/></r>`), "a,b valid")
			require.Empty(t, parseValidating(t, dtd+`<r><a/><b/><c/><a/><b/></r>`),
				"interleaved groups valid")
			require.NotEmpty(t, parseValidating(t, dtd+`<r><a/></r>`), "a without b invalid")
			require.NotEmpty(t, parseValidating(t, dtd+`<r><a/><c/></r>`), "a then c (no b) invalid")
			require.NotEmpty(t, parseValidating(t, dtd+`<r><b/><a/></r>`), "b before a invalid")
		})

		// A genuine backtracking case the greedy matcher alone cannot handle even
		// with correct grouping: the optional a? greedily consumes the only a,
		// leaving the required a unmatched. The exact fallback must accept it.
		t.Run("(a?,a) backtracking", func(t *testing.T) {
			t.Parallel()
			const dtd = `<!DOCTYPE r [
<!ELEMENT r (a?,a)>
<!ELEMENT a EMPTY>
]>`
			require.Empty(t, parseValidating(t, dtd+`<r><a/></r>`), "single a satisfies (a?,a)")
			require.Empty(t, parseValidating(t, dtd+`<r><a/><a/></r>`), "two a satisfies (a?,a)")
			require.NotEmpty(t, parseValidating(t, dtd+`<r></r>`), "empty invalid (needs one a)")
			require.NotEmpty(t, parseValidating(t, dtd+`<r><a/><a/><a/></r>`), "three a invalid")
		})
	})

	// DTD content-model validation
	// compares raw qualified names (prefix + local, as written), not local names
	// only. DTD validation is not namespace-aware: a declared prefix is an opaque
	// part of the element name and must match the instance tag literally.
	t.Run("QName matching", func(t *testing.T) {
		parse := func(t *testing.T, xml string) error {
			t.Helper()
			collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
			p := helium.NewParser().ValidateDTD(true).ErrorHandler(collector)
			_, err := p.Parse(t.Context(), []byte(xml))
			return err
		}

		t.Run("element content (p:a) rejects unprefixed <a/>", func(t *testing.T) {
			t.Parallel()
			xml := `<?xml version="1.0"?>
<!DOCTYPE r [
  <!ELEMENT r (p:a)>
  <!ELEMENT p:a EMPTY>
  <!ELEMENT a EMPTY>
]>
<r xmlns:p="u"><a/></r>`
			require.ErrorIs(t, parse(t, xml), helium.ErrDTDValidationFailed)
		})

		t.Run("element content (p:a) accepts <p:a/>", func(t *testing.T) {
			t.Parallel()
			xml := `<?xml version="1.0"?>
<!DOCTYPE r [
  <!ELEMENT r (p:a)>
  <!ELEMENT p:a EMPTY>
  <!ELEMENT a EMPTY>
]>
<r xmlns:p="u"><p:a/></r>`
			require.NoError(t, parse(t, xml))
		})

		t.Run("element content (a) rejects prefixed <p:a/>", func(t *testing.T) {
			t.Parallel()
			xml := `<?xml version="1.0"?>
<!DOCTYPE r [
  <!ELEMENT r (a)>
  <!ELEMENT p:a EMPTY>
  <!ELEMENT a EMPTY>
]>
<r xmlns:p="u"><p:a/></r>`
			require.ErrorIs(t, parse(t, xml), helium.ErrDTDValidationFailed)
		})

		t.Run("mixed content (#PCDATA|p:a)* rejects unprefixed <a/>", func(t *testing.T) {
			t.Parallel()
			xml := `<?xml version="1.0"?>
<!DOCTYPE r [
  <!ELEMENT r (#PCDATA|p:a)*>
  <!ELEMENT p:a EMPTY>
  <!ELEMENT a EMPTY>
]>
<r xmlns:p="u">text<a/></r>`
			require.ErrorIs(t, parse(t, xml), helium.ErrDTDValidationFailed)
		})

		t.Run("mixed content (#PCDATA|p:a)* accepts <p:a/>", func(t *testing.T) {
			t.Parallel()
			xml := `<?xml version="1.0"?>
<!DOCTYPE r [
  <!ELEMENT r (#PCDATA|p:a)*>
  <!ELEMENT p:a EMPTY>
  <!ELEMENT a EMPTY>
]>
<r xmlns:p="u">text<p:a/></r>`
			require.NoError(t, parse(t, xml))
		})

		t.Run("unprefixed element content unchanged", func(t *testing.T) {
			t.Parallel()
			xml := `<?xml version="1.0"?>
<!DOCTYPE r [
  <!ELEMENT r (a, b)>
  <!ELEMENT a EMPTY>
  <!ELEMENT b EMPTY>
]>
<r><a/><b/></r>`
			require.NoError(t, parse(t, xml))
		})

		t.Run("unprefixed mixed content unchanged", func(t *testing.T) {
			t.Parallel()
			xml := `<?xml version="1.0"?>
<!DOCTYPE r [
  <!ELEMENT r (#PCDATA|a)*>
  <!ELEMENT a EMPTY>
]>
<r>text<a/>more</r>`
			require.NoError(t, parse(t, xml))
		})
	})
}

func TestValidateElementContent(t *testing.T) {
	t.Parallel()

	// asserts that a CDATA section in element-only
	// content is a validity error (VC: Element Valid) even when the CDATA section is
	// empty or whitespace-only: a CDATA section is character data and never matches
	// the S nonterminal (XML §2.4/§3.2.1). W3C xmlconf "empty" (sun/invalid).
	t.Run("CDATA is invalid", func(t *testing.T) {
		t.Run("whitespace-only CDATA in element content is invalid", func(t *testing.T) {
			t.Parallel()
			err := parseECValidate(t, `<!DOCTYPE foo [
<!ELEMENT foo (a+)>
<!ELEMENT a EMPTY>
]>
<foo><a/><![CDATA[ ]]><a/></foo>`)
			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
		})

		t.Run("empty CDATA in element content is invalid", func(t *testing.T) {
			t.Parallel()
			err := parseECValidate(t, `<!DOCTYPE foo [
<!ELEMENT foo (a+)>
<!ELEMENT a EMPTY>
]>
<foo><a/><![CDATA[]]><a/></foo>`)
			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
		})
	})

	// against over-rejection: literal
	// ignorable whitespace between child elements in element-only content is valid,
	// and CDATA is still permitted in mixed and ANY content.
	t.Run("whitespace stays valid", func(t *testing.T) {
		t.Run("literal ignorable whitespace in element content is valid", func(t *testing.T) {
			t.Parallel()
			err := parseECValidate(t, `<!DOCTYPE foo [
<!ELEMENT foo (a+)>
<!ELEMENT a EMPTY>
]>
<foo>
  <a/>
  <a/>
</foo>`)
			require.NoError(t, err)
		})

		t.Run("CDATA in mixed content is valid", func(t *testing.T) {
			t.Parallel()
			err := parseECValidate(t, `<!DOCTYPE foo [
<!ELEMENT foo (#PCDATA|a)*>
<!ELEMENT a EMPTY>
]>
<foo><a/><![CDATA[ text ]]><a/></foo>`)
			require.NoError(t, err)
		})

		t.Run("CDATA in ANY content is valid", func(t *testing.T) {
			t.Parallel()
			err := parseECValidate(t, `<!DOCTYPE foo [
<!ELEMENT foo ANY>
<!ELEMENT a EMPTY>
]>
<foo><a/><![CDATA[ text ]]><a/></foo>`)
			require.NoError(t, err)
		})
	})

	// asserts that whitespace introduced by a
	// character reference (directly or via a general entity whose replacement text
	// is a character reference) does NOT match the S nonterminal and is therefore a
	// validity error in element-only content, while literal whitespace — including
	// an internal entity whose replacement text is itself literal whitespace — stays
	// ignorable (XML §3.2.1 as clarified by errata 2e E15). W3C xmlconf rmt-e2e-15*.
	t.Run("char-ref whitespace", func(t *testing.T) {
		// E15g: a direct character reference producing whitespace between element
		// children in element-only content is not ignorable.
		t.Run("direct char-ref whitespace is invalid (E15g)", func(t *testing.T) {
			t.Parallel()
			err := parseECValidate(t, `<!DOCTYPE foo [
<!ELEMENT foo (foo*)>
]>
<foo><foo/>&#32;<foo/></foo>`)
			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
		})

		// E15h: a general entity whose replacement text is a character reference
		// (&#38;#32; declares the entity value "&#32;") re-parses to a space at
		// inclusion time — that whitespace came from a character reference.
		t.Run("entity-of-char-ref whitespace is invalid (E15h)", func(t *testing.T) {
			t.Parallel()
			err := parseECValidate(t, `<!DOCTYPE foo [
<!ELEMENT foo (foo*)>
<!ENTITY space "&#38;#32;">
]>
<foo><foo/>&space;<foo/></foo>`)
			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
		})

		// E15e: an internal entity whose replacement text is a literal space is
		// ignorable whitespace — valid.
		t.Run("entity-of-literal-space whitespace is valid (E15e)", func(t *testing.T) {
			t.Parallel()
			err := parseECValidate(t, `<!DOCTYPE foo [
<!ELEMENT foo (foo*)>
<!ENTITY space " ">
]>
<foo><foo/>&space;<foo/></foo>`)
			require.NoError(t, err)
		})

		// E15f: a character reference in an entity's LITERAL value expands at
		// declaration time, so the replacement text is a literal space — the
		// whitespace does not originate from a character reference at inclusion time
		// and stays ignorable — valid.
		t.Run("entity literal &#32; expands at decl time and is valid (E15f)", func(t *testing.T) {
			t.Parallel()
			err := parseECValidate(t, `<!DOCTYPE foo [
<!ELEMENT foo (foo*)>
<!ENTITY space "&#32;">
]>
<foo><foo/>&space;<foo/></foo>`)
			require.NoError(t, err)
		})

		// E15a: a reference is content per XML production [43]; an element declared
		// EMPTY that contains one is invalid even when the reference expands to
		// nothing and leaves the element with no child node.
		t.Run("empty entity reference in an EMPTY element is invalid (E15a)", func(t *testing.T) {
			t.Parallel()
			err := parseECValidate(t, `<!DOCTYPE foo [
<!ELEMENT foo EMPTY>
<!ENTITY empty "">
]>
<foo>&empty;</foo>`)
			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
		})

		// Literal whitespace mixed with a character reference in element-only content
		// is still invalid — the merged text node carries the char-reference origin.
		t.Run("literal-plus-char-ref whitespace is invalid", func(t *testing.T) {
			t.Parallel()
			err := parseECValidate(t, `<!DOCTYPE foo [
<!ELEMENT foo (foo*)>
]>
<foo><foo/> &#32; <foo/></foo>`)
			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
		})

		// A character reference producing whitespace in MIXED content is character
		// data, which mixed content permits — still valid.
		t.Run("char-ref whitespace in mixed content is valid", func(t *testing.T) {
			t.Parallel()
			err := parseECValidate(t, `<!DOCTYPE foo [
<!ELEMENT foo (#PCDATA|a)*>
<!ELEMENT a EMPTY>
]>
<foo><a/>&#32;<a/></foo>`)
			require.NoError(t, err)
		})
	})

	// asserts that a whitespace character introduced by
	// a character reference in an NMTOKENS attribute value is NOT a token separator:
	// attribute-value normalization (XML §3.3.3) folds literal whitespace to a single
	// space but leaves character-reference whitespace verbatim, so `abc&#9;xyz`
	// normalizes to the single token "abc\txyz", which is not a valid NMTOKEN
	// (the tab is not a NameChar). W3C xmlconf rmt-e2e-20.
	t.Run("NMTOKENS char-ref whitespace", func(t *testing.T) {
		t.Run("char-ref tab makes an invalid NMTOKEN", func(t *testing.T) {
			t.Parallel()
			err := parseECValidate(t, `<!DOCTYPE foo [
<!ELEMENT foo ANY>
<!ATTLIST foo bar NMTOKENS #IMPLIED>
]>
<foo bar="abc&#9;xyz"/>`)
			require.ErrorIs(t, err, helium.ErrDTDValidationFailed)
		})

		t.Run("literal-space-separated NMTOKENS is valid", func(t *testing.T) {
			t.Parallel()
			err := parseECValidate(t, `<!DOCTYPE foo [
<!ELEMENT foo ANY>
<!ATTLIST foo bar NMTOKENS #IMPLIED>
]>
<foo bar="abc   xyz"/>`)
			require.NoError(t, err)
		})

		t.Run("literal-tab-separated NMTOKENS is valid (normalized to space)", func(t *testing.T) {
			t.Parallel()
			err := parseECValidate(t, "<!DOCTYPE foo [\n<!ELEMENT foo ANY>\n<!ATTLIST foo bar NMTOKENS #IMPLIED>\n]>\n<foo bar=\"abc\txyz\"/>")
			require.NoError(t, err)
		})
	})

	// the blast radius: the
	// char-reference-origin marker on a Text node is invisible to serialization —
	// a document whose text came from a character reference serializes byte-for-byte
	// identically to the same document written with literal text.
	t.Run("char-ref provenance leaves serialization unchanged", func(t *testing.T) {
		parse := func(src string) *helium.Document {
			t.Helper()
			doc, err := helium.NewParser().
				SubstituteEntities(true).
				ValidateDTD(true).
				DefaultDTDAttributes(true).
				Parse(t.Context(), []byte(src))
			require.NoError(t, err)
			return doc
		}

		literal := parse(`<!DOCTYPE foo [<!ELEMENT foo (#PCDATA)>]><foo>a b</foo>`)
		charRef := parse(`<!DOCTYPE foo [<!ELEMENT foo (#PCDATA)>]><foo>a&#32;b</foo>`)

		litStr, err := helium.WriteString(literal)
		require.NoError(t, err)
		refStr, err := helium.WriteString(charRef)
		require.NoError(t, err)
		require.Equal(t, litStr, refStr)
	})
}
