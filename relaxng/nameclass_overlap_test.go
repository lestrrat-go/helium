package relaxng_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/relaxng"
	"github.com/stretchr/testify/require"
)

// TestNameClassOverlapExceptChoice covers the attribute name-class conflict
// check. An <anyName> with an <except> listing several names does NOT overlap a
// <choice> of exactly those excluded names, so a grammar pairing them as two
// attributes must compile cleanly. A choice that includes a name NOT excluded
// genuinely overlaps and must still be reported.
func TestNameClassOverlapExceptChoice(t *testing.T) {
	t.Parallel()

	compile := func(t *testing.T, schema string) string {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schema))
		require.NoError(t, err)

		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, err = relaxng.NewCompiler().ErrorHandler(collector).Compile(t.Context(), doc)
		require.NoError(t, err)
		_ = collector.Close()
		_, compileErrors := partitionCompileErrors(collector.Errors())
		return compileErrors
	}

	t.Run("disjoint anyName-except vs choice compiles", func(t *testing.T) {
		const schema = `<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start>
    <element name="root">
      <attribute>
        <anyName>
          <except><name>foo</name><name>bar</name></except>
        </anyName>
      </attribute>
      <attribute>
        <choice><name>foo</name><name>bar</name></choice>
      </attribute>
    </element>
  </start>
</grammar>`
		require.Empty(t, compile(t, schema),
			"anyName-except{foo,bar} and choice(foo,bar) are disjoint and must not conflict")
	})

	t.Run("genuinely overlapping anyName-except vs choice errors", func(t *testing.T) {
		const schema = `<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start>
    <element name="root">
      <attribute>
        <anyName>
          <except><name>foo</name></except>
        </anyName>
      </attribute>
      <attribute>
        <choice><name>foo</name><name>bar</name></choice>
      </attribute>
    </element>
  </start>
</grammar>`
		require.NotEmpty(t, compile(t, schema),
			"anyName-except{foo} overlaps choice(foo,bar) on bar and must conflict")
	})

	t.Run("disjoint anyName-except-nsName vs nsName compiles", func(t *testing.T) {
		const schema = `<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start>
    <element name="root">
      <attribute>
        <anyName>
          <except><nsName ns="http://example.com/X"/></except>
        </anyName>
      </attribute>
      <attribute>
        <nsName ns="http://example.com/X"/>
      </attribute>
    </element>
  </start>
</grammar>`
		require.Empty(t, compile(t, schema),
			"anyName-except-nsName(X) and nsName(X) are disjoint and must not conflict")
	})

	t.Run("genuinely overlapping anyName-except-name vs nsName errors", func(t *testing.T) {
		const schema = `<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start>
    <element name="root">
      <attribute>
        <anyName>
          <except><name ns="http://example.com/X">foo</name></except>
        </anyName>
      </attribute>
      <attribute>
        <nsName ns="http://example.com/X"/>
      </attribute>
    </element>
  </start>
</grammar>`
		require.NotEmpty(t, compile(t, schema),
			"anyName-except{foo} still overlaps nsName(X) on every other name in X and must conflict")
	})

	// anyName except (nsName(X) except name(foo)) excludes every name in X but
	// foo, so within X it matches ONLY foo. nsName(X) except name(foo) matches
	// every name in X but foo. The two are disjoint (no shared name) and must
	// compile — the excluded class carries its own finite except, which the
	// containment check must recurse into rather than bail on.
	t.Run("disjoint anyName-except-nsName-except vs nsName-except compiles", func(t *testing.T) {
		const schema = `<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start>
    <element name="root">
      <attribute>
        <anyName>
          <except>
            <nsName ns="http://example.com/X">
              <except><name ns="http://example.com/X">foo</name></except>
            </nsName>
          </except>
        </anyName>
      </attribute>
      <attribute>
        <nsName ns="http://example.com/X">
          <except><name ns="http://example.com/X">foo</name></except>
        </nsName>
      </attribute>
    </element>
  </start>
</grammar>`
		require.Empty(t, compile(t, schema),
			"anyName except (nsName(X) except foo) and nsName(X) except foo share no name and must not conflict")
	})

	// anyName except (nsName(X) except name(foo)) matches only foo within X.
	// nsName(X) except name(bar) matches every name in X but bar, which INCLUDES
	// foo (foo != bar). They share foo@X, genuinely overlap, and must conflict.
	t.Run("genuinely overlapping anyName-except-nsName-except vs nsName-except errors", func(t *testing.T) {
		const schema = `<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start>
    <element name="root">
      <attribute>
        <anyName>
          <except>
            <nsName ns="http://example.com/X">
              <except><name ns="http://example.com/X">foo</name></except>
            </nsName>
          </except>
        </anyName>
      </attribute>
      <attribute>
        <nsName ns="http://example.com/X">
          <except><name ns="http://example.com/X">bar</name></except>
        </nsName>
      </attribute>
    </element>
  </start>
</grammar>`
		require.NotEmpty(t, compile(t, schema),
			"both classes match foo@X and must conflict")
	})

	// The excluded class covers a whole namespace by UNION, not by any single
	// branch: (nsName(X) except foo) | name(foo) together match every name in X.
	// So anyName except that choice matches NO name in X and is disjoint from
	// nsName(X). The containment check must account for the union, not test each
	// branch for full coverage independently.
	t.Run("disjoint anyName-except-union-choice vs nsName compiles", func(t *testing.T) {
		const schema = `<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start>
    <element name="root">
      <attribute>
        <anyName>
          <except>
            <choice>
              <nsName ns="http://example.com/X">
                <except><name ns="http://example.com/X">foo</name></except>
              </nsName>
              <name ns="http://example.com/X">foo</name>
            </choice>
          </except>
        </anyName>
      </attribute>
      <attribute>
        <nsName ns="http://example.com/X"/>
      </attribute>
    </element>
  </start>
</grammar>`
		require.Empty(t, compile(t, schema),
			"(nsName(X) except foo) | name(foo) covers all of X by union, so anyName-except-it is disjoint from nsName(X)")
	})

	// Same shape but the union LEAVES A GAP: the choice is only
	// (nsName(X) except foo), with no sibling filling foo. So anyName-except-it
	// matches foo@X, which nsName(X) also matches — a genuine overlap.
	t.Run("genuinely overlapping anyName-except-partial-union vs nsName errors", func(t *testing.T) {
		const schema = `<grammar xmlns="http://relaxng.org/ns/structure/1.0">
  <start>
    <element name="root">
      <attribute>
        <anyName>
          <except>
            <choice>
              <nsName ns="http://example.com/X">
                <except><name ns="http://example.com/X">foo</name></except>
              </nsName>
              <name ns="http://example.com/X">bar</name>
            </choice>
          </except>
        </anyName>
      </attribute>
      <attribute>
        <nsName ns="http://example.com/X"/>
      </attribute>
    </element>
  </start>
</grammar>`
		require.NotEmpty(t, compile(t, schema),
			"the union leaves foo@X uncovered, so anyName-except-it matches foo@X and overlaps nsName(X)")
	})
}
