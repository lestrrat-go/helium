package relaxng_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestTokenMatcherNullableOneOrMore covers oneOrMore over content that can match
// empty (here optional) inside a <list>, followed by a mandatory member. A
// single nullable iteration must satisfy oneOrMore so it can consume zero
// tokens, letting the trailing member match — matching node-level
// validateOneOrMore semantics. Without that, `<a>END</a>` is wrongly rejected.
func TestTokenMatcherNullableOneOrMore(t *testing.T) {
	t.Parallel()

	schema := `<element name="a" xmlns="http://relaxng.org/ns/structure/1.0">
  <list>
    <group>
      <oneOrMore><optional><data type="integer"/></optional></oneOrMore>
      <value>END</value>
    </group>
  </list>
</element>`

	// oneOrMore matches one zero-width iteration (consumes 0), value END matches.
	require.NoError(t, validateWith(t, schema, `<a>END</a>`), "nullable oneOrMore then END should validate")
	// oneOrMore consumes the integers, value END matches the last token.
	require.NoError(t, validateWith(t, schema, `<a>1 2 END</a>`), "tokens then END should validate")
	// Missing the mandatory END member -> invalid.
	require.Error(t, validateWith(t, schema, `<a></a>`), "empty list must be rejected (END required)")
	require.Error(t, validateWith(t, schema, `<a>1 2</a>`), "missing END must be rejected")
}

// TestTokenMatcherNoExponentialBlowup guards against the combinatorial blowup in
// groupCounts/repeatCounts: a <list> group of many optionals over many tokens has
// only len+1 distinct totals, but without memoization the enumeration explores
// 2^N paths. With memoization this completes effectively instantly; a regression
// would make the test hang until the timeout.
func TestTokenMatcherNoExponentialBlowup(t *testing.T) {
	t.Parallel()

	const optionals = 40
	var b strings.Builder
	b.WriteString(`<element name="a" xmlns="http://relaxng.org/ns/structure/1.0"><list><group>`)
	for range optionals {
		b.WriteString(`<optional><data type="integer"/></optional>`)
	}
	b.WriteString(`</group></list></element>`)

	// 20 integer tokens: 20 optionals consume one each, the rest match empty.
	instance := `<a>1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20</a>`
	require.NoError(t, validateWith(t, b.String(), instance))
}

// TestListGroupChoiceUnderOptional covers <list>s whose children are group/choice
// nested under optional in element content. All such lists share the
// matchListContent token matcher (validateList now delegates to it too), so they
// must validate with full backtracking semantics rather than being mishandled.
func TestListGroupChoiceUnderOptional(t *testing.T) {
	t.Parallel()

	groupSchema := `<element name="a" xmlns="http://relaxng.org/ns/structure/1.0">
  <optional>
    <list><group><value>X</value><value>Y</value></group></list>
  </optional>
</element>`
	require.NoError(t, validateWith(t, groupSchema, `<a>X Y</a>`), "group list should validate")
	require.NoError(t, validateWith(t, groupSchema, `<a></a>`), "absent optional list should validate")
	require.Error(t, validateWith(t, groupSchema, `<a>X</a>`), "incomplete group must be rejected")

	choiceSchema := `<element name="a" xmlns="http://relaxng.org/ns/structure/1.0">
  <list><oneOrMore><choice><value>P</value><value>Q</value></choice></oneOrMore></list>
</element>`
	require.NoError(t, validateWith(t, choiceSchema, `<a>P Q P</a>`), "choice list should validate")
	require.Error(t, validateWith(t, choiceSchema, `<a>P Z</a>`), "non-member token must be rejected")
}
