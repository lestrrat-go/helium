package helium_test

import (
	"context"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

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
