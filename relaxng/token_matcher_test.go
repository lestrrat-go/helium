package relaxng_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/relaxng"
	"github.com/stretchr/testify/require"
)

// validateWith compiles the given RELAX NG schema and validates the XML
// instance, returning the validation error (nil on success).
func validateWith(t *testing.T, schema, instance string) error {
	t.Helper()

	schemaDoc, err := helium.NewParser().Parse(t.Context(), []byte(schema))
	require.NoError(t, err, "schema should parse")

	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	grammar, err := relaxng.NewCompiler().ErrorHandler(collector).Compile(t.Context(), schemaDoc)
	require.NoError(t, err, "schema should compile")
	_ = collector.Close()
	_, compileErrors := partitionCompileErrors(collector.Errors())
	require.Empty(t, compileErrors, "schema should compile without errors")

	instanceDoc, err := helium.NewParser().Parse(t.Context(), []byte(instance))
	require.NoError(t, err, "instance should parse")

	return relaxng.NewValidator(grammar).Validate(t.Context(), instanceDoc)
}

// TestTokenMatcherChoiceShadow covers a <list> whose group has a leading
// choice between <empty/> and a consuming branch. The zero-token <empty/>
// branch must not shadow the consuming integer branch.
func TestTokenMatcherChoiceShadow(t *testing.T) {
	t.Parallel()

	schema := `<element name="a" xmlns="http://relaxng.org/ns/structure/1.0">
  <list>
    <group>
      <choice>
        <empty/>
        <data type="integer"/>
      </choice>
      <data type="integer"/>
    </group>
  </list>
</element>`

	t.Run("valid two tokens", func(t *testing.T) {
		t.Parallel()
		err := validateWith(t, schema, `<a>1 2</a>`)
		require.NoError(t, err, `"1 2" should validate`)
	})

	t.Run("invalid three tokens", func(t *testing.T) {
		t.Parallel()
		err := validateWith(t, schema, `<a>1 2 3</a>`)
		require.Error(t, err, `"1 2 3" should be rejected`)
	})
}

// TestTokenMatcherGroupBacktrack covers a <list> group where a greedy
// <oneOrMore> must give back a token so a later mandatory <value> matches.
func TestTokenMatcherGroupBacktrack(t *testing.T) {
	t.Parallel()

	schema := `<element name="a" xmlns="http://relaxng.org/ns/structure/1.0">
  <list>
    <group>
      <oneOrMore>
        <data type="NMTOKEN"/>
      </oneOrMore>
      <value>END</value>
    </group>
  </list>
</element>`

	t.Run("valid trailing value", func(t *testing.T) {
		t.Parallel()
		err := validateWith(t, schema, `<a>a b END</a>`)
		require.NoError(t, err, `"a b END" should validate`)
	})

	t.Run("invalid missing value", func(t *testing.T) {
		t.Parallel()
		err := validateWith(t, schema, `<a>a b c</a>`)
		require.Error(t, err, `missing END should be rejected`)
	})
}

// TestTokenMatcherChoiceShadowAttr covers the same choice-shadow shape inside
// an attribute value (driven through matchAttrContent's patternGroup case).
func TestTokenMatcherChoiceShadowAttr(t *testing.T) {
	t.Parallel()

	schema := `<element name="a" xmlns="http://relaxng.org/ns/structure/1.0">
  <attribute name="v">
    <group>
      <choice>
        <empty/>
        <data type="integer"/>
      </choice>
      <data type="integer"/>
    </group>
  </attribute>
</element>`

	t.Run("valid two tokens", func(t *testing.T) {
		t.Parallel()
		err := validateWith(t, schema, `<a v="1 2"/>`)
		require.NoError(t, err, `attribute "1 2" should validate`)
	})

	t.Run("invalid three tokens", func(t *testing.T) {
		t.Parallel()
		err := validateWith(t, schema, `<a v="1 2 3"/>`)
		require.Error(t, err, `attribute "1 2 3" should be rejected`)
	})
}

// TestTokenMatcherListInRepetitionAttr covers a <list> nested inside a
// repetition in an attribute (matchAttrTokensCounts patternList case). Each
// repetition iteration consumes one full run of the list's children; the
// repetition machinery chains those runs. Without the patternList case the
// list match returns no counts and every such attribute is wrongly rejected.
func TestTokenMatcherListInRepetitionAttr(t *testing.T) {
	t.Parallel()

	oneOrMore := `<element name="a" xmlns="http://relaxng.org/ns/structure/1.0">
  <attribute name="v">
    <oneOrMore>
      <list>
        <value>foo</value>
        <value>bar</value>
      </list>
    </oneOrMore>
  </attribute>
</element>`

	t.Run("oneOrMore single occurrence", func(t *testing.T) {
		t.Parallel()
		err := validateWith(t, oneOrMore, `<a v="foo bar"/>`)
		require.NoError(t, err, `"foo bar" should validate as one list occurrence`)
	})

	t.Run("oneOrMore two occurrences", func(t *testing.T) {
		t.Parallel()
		err := validateWith(t, oneOrMore, `<a v="foo bar foo bar"/>`)
		require.NoError(t, err, `"foo bar foo bar" should validate as two list occurrences`)
	})

	t.Run("oneOrMore incomplete list", func(t *testing.T) {
		t.Parallel()
		err := validateWith(t, oneOrMore, `<a v="foo"/>`)
		require.Error(t, err, `"foo" is an incomplete list and should be rejected`)
	})

	t.Run("oneOrMore wrong second token", func(t *testing.T) {
		t.Parallel()
		err := validateWith(t, oneOrMore, `<a v="foo baz"/>`)
		require.Error(t, err, `"foo baz" has a wrong second token and should be rejected`)
	})

	zeroOrMore := `<element name="a" xmlns="http://relaxng.org/ns/structure/1.0">
  <attribute name="v">
    <zeroOrMore>
      <list>
        <value>foo</value>
        <value>bar</value>
      </list>
    </zeroOrMore>
  </attribute>
</element>`

	t.Run("zeroOrMore empty", func(t *testing.T) {
		t.Parallel()
		err := validateWith(t, zeroOrMore, `<a v=""/>`)
		require.NoError(t, err, `empty attribute should validate against zeroOrMore-of-list`)
	})
}
