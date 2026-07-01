package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestVersion11UPAWeakening verifies that in XSD 1.1 an element particle
// competing with a wildcard is not a UPA (cos-nonambig) violation (the element
// wins), while in XSD 1.0 the same content model is rejected as ambiguous.
func TestVersion11UPAWeakening(t *testing.T) {
	// A choice of an element and a wildcard that admits the element's namespace:
	// ambiguous under 1.0, allowed under 1.1.
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:choice maxOccurs="unbounded">
        <xs:element name="a" type="xs:string"/>
        <xs:any processContents="lax"/>
      </xs:choice>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	compile := func(t *testing.T, c xsd.Compiler) (*xsd.Schema, error) {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		return c.Compile(t.Context(), doc)
	}

	t.Run("1.0 rejects element-vs-wildcard as non-deterministic", func(t *testing.T) {
		t.Parallel()
		_, err := compile(t, xsd.NewCompiler())
		require.ErrorIs(t, err, xsd.ErrCompilationFailed)
	})

	t.Run("1.1 accepts the content model and validates", func(t *testing.T) {
		t.Parallel()
		schema, err := compile(t, xsd.NewCompiler().Version(xsd.Version11))
		require.NoError(t, err)

		// The declared element matches the element particle; an unknown element is
		// admitted by the lax wildcard.
		instance := `<root><a>hi</a><other/></root>`
		doc, err := helium.NewParser().Parse(t.Context(), []byte(instance))
		require.NoError(t, err)
		require.NoError(t, xsd.NewValidator(schema).Validate(t.Context(), doc))
	})
}

// TestVersion11WildcardPrecedence verifies that in XSD 1.1 a non-wildcard
// element particle takes precedence over a wildcard particle declared BEFORE it
// in a choice, so a skip wildcard preceding an element cannot steal a child the
// element declaration must validate (XSD11-001).
func TestVersion11WildcardPrecedence(t *testing.T) {
	// The skip wildcard is declared BEFORE the typed element. Under naive
	// declaration-order matching the wildcard would match <a> and the xs:int
	// element declaration would never run, false-accepting "not-int".
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:choice maxOccurs="unbounded">
        <xs:any processContents="skip"/>
        <xs:element name="a" type="xs:int"/>
      </xs:choice>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	compile := func(t *testing.T) *xsd.Schema {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		schema, err := xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), doc)
		require.NoError(t, err)
		return schema
	}

	validate := func(t *testing.T, schema *xsd.Schema, instance string) error {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(instance))
		require.NoError(t, err)
		return xsd.NewValidator(schema).Validate(t.Context(), doc)
	}

	t.Run("element declaration wins over preceding wildcard (invalid value rejected)", func(t *testing.T) {
		t.Parallel()
		schema := compile(t)
		err := validate(t, schema, `<root><a>not-int</a></root>`)
		require.ErrorIs(t, err, xsd.ErrValidationFailed)
	})

	t.Run("element declaration wins over preceding wildcard (valid value accepted)", func(t *testing.T) {
		t.Parallel()
		schema := compile(t)
		require.NoError(t, validate(t, schema, `<root><a>5</a></root>`))
	})

	t.Run("unknown element still matches the wildcard", func(t *testing.T) {
		t.Parallel()
		schema := compile(t)
		require.NoError(t, validate(t, schema, `<root><other/></root>`))
	})
}

// TestVersion11WildcardPrecedenceNested verifies that element-over-wildcard
// precedence also holds when the wildcard is NOT a direct choice branch but is
// wrapped inside a model group (here a sequence). The typed element declaration
// must still win over the nested skip wildcard so an invalid value is rejected
// rather than swallowed by the wildcard branch (XSD11-001, nested case).
func TestVersion11WildcardPrecedenceNested(t *testing.T) {
	// Branch 1 wraps the skip wildcard in a sequence; branch 2 is the typed
	// element. Naive top-level classification treats branch 1 as "non-wildcard"
	// (its term is a sequence), so it would steal <a> and false-accept "not-int".
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:choice maxOccurs="unbounded">
        <xs:sequence>
          <xs:any processContents="skip"/>
        </xs:sequence>
        <xs:element name="a" type="xs:int"/>
      </xs:choice>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	compile := func(t *testing.T) *xsd.Schema {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		schema, err := xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), doc)
		require.NoError(t, err)
		return schema
	}

	validate := func(t *testing.T, schema *xsd.Schema, instance string) error {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(instance))
		require.NoError(t, err)
		return xsd.NewValidator(schema).Validate(t.Context(), doc)
	}

	t.Run("typed element wins over wildcard nested in a sequence (invalid rejected)", func(t *testing.T) {
		t.Parallel()
		schema := compile(t)
		err := validate(t, schema, `<root><a>not-int</a></root>`)
		require.ErrorIs(t, err, xsd.ErrValidationFailed)
	})

	t.Run("typed element wins over wildcard nested in a sequence (valid accepted)", func(t *testing.T) {
		t.Parallel()
		schema := compile(t)
		require.NoError(t, validate(t, schema, `<root><a>5</a></root>`))
	})

	t.Run("unknown element still matches the nested wildcard branch", func(t *testing.T) {
		t.Parallel()
		schema := compile(t)
		require.NoError(t, validate(t, schema, `<root><other/></root>`))
	})
}

// TestVersion11WildcardPrecedenceNestedLeading verifies the path-aware first
// consumer rule: a choice branch whose sequence begins with a wildcard and is
// FOLLOWED by an element leaf must NOT be classified as element-first for that
// child — the leading wildcard would consume it first. A broad "any descendant
// element matches" classifier wrongly prefers this branch and lets the leading
// skip wildcard swallow the first child, false-accepting an invalid value
// (XSD11-001, nested leading-wildcard case).
func TestVersion11WildcardPrecedenceNestedLeading(t *testing.T) {
	// Branch 1 is sequence(any skip, element a:int); branch 2 is element a:int.
	// For the first <a>, branch 1's LEADING wildcard would consume it (via a
	// wildcard leaf, no int validation), so precedence must select branch 2 (the
	// element declaration) and reject an invalid value.
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:choice maxOccurs="unbounded">
        <xs:sequence>
          <xs:any processContents="skip"/>
          <xs:element name="a" type="xs:int"/>
        </xs:sequence>
        <xs:element name="a" type="xs:int"/>
      </xs:choice>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	compile := func(t *testing.T) *xsd.Schema {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		schema, err := xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), doc)
		require.NoError(t, err)
		return schema
	}

	validate := func(t *testing.T, schema *xsd.Schema, instance string) error {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(instance))
		require.NoError(t, err)
		return xsd.NewValidator(schema).Validate(t.Context(), doc)
	}

	t.Run("element wins over a leading nested wildcard (invalid first child rejected)", func(t *testing.T) {
		t.Parallel()
		schema := compile(t)
		// The first <a> must be validated by the element declaration (xs:int),
		// not swallowed by branch 1's leading skip wildcard.
		err := validate(t, schema, `<root><a>not-int</a><a>5</a></root>`)
		require.ErrorIs(t, err, xsd.ErrValidationFailed)
	})

	t.Run("valid document still accepted", func(t *testing.T) {
		t.Parallel()
		schema := compile(t)
		require.NoError(t, validate(t, schema, `<root><a>5</a></root>`))
	})

	t.Run("unknown element still matched by the leading wildcard branch", func(t *testing.T) {
		t.Parallel()
		schema := compile(t)
		// <other/> can only be consumed by branch 1's leading wildcard, with the
		// trailing element a satisfied by <a>5</a>.
		require.NoError(t, validate(t, schema, `<root><other/><a>5</a></root>`))
	})
}

// TestVersion11WildcardPrecedenceCommit verifies that once a choice selects an
// element-first branch for a child it COMMITS to that branch and does NOT fall
// back to a wildcard branch when the element branch later fails — structurally
// or by content. The element branch here is sequence(a:int, b:int); the first
// child <a> selects it, and its failure (bad content and/or missing b) must NOT
// be rescued by the skip wildcard (XSD11-001, commit-no-fallback case).
func TestVersion11WildcardPrecedenceCommit(t *testing.T) {
	// Branch 1 is sequence(element a:int, element b:int); branch 2 is a skip
	// wildcard. For a first <a>, branch 1 is element-first and is selected. If
	// branch 1 then fails (a's value is not an int, and/or b is missing), the
	// choice must report that failure rather than letting the wildcard swallow
	// <a> and false-accept the document.
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:choice maxOccurs="unbounded">
        <xs:sequence>
          <xs:element name="a" type="xs:int"/>
          <xs:element name="b" type="xs:int"/>
        </xs:sequence>
        <xs:any processContents="skip"/>
      </xs:choice>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	compile := func(t *testing.T) *xsd.Schema {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		schema, err := xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), doc)
		require.NoError(t, err)
		return schema
	}

	validate := func(t *testing.T, schema *xsd.Schema, instance string) error {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(instance))
		require.NoError(t, err)
		return xsd.NewValidator(schema).Validate(t.Context(), doc)
	}

	t.Run("committed element branch failure is not rescued by the wildcard", func(t *testing.T) {
		t.Parallel()
		schema := compile(t)
		// <a> selects the element-first sequence branch; its content is not an int
		// and b is missing, so the document must be rejected — the skip wildcard
		// must NOT fall back to swallow <a>.
		err := validate(t, schema, `<root><a>not-int</a></root>`)
		require.ErrorIs(t, err, xsd.ErrValidationFailed)
	})

	t.Run("valid element branch accepted", func(t *testing.T) {
		t.Parallel()
		schema := compile(t)
		require.NoError(t, validate(t, schema, `<root><a>1</a><b>2</b></root>`))
	})

	t.Run("unknown element still matched by the wildcard branch", func(t *testing.T) {
		t.Parallel()
		schema := compile(t)
		// <other/> has no element-first candidate, so the wildcard branch applies.
		require.NoError(t, validate(t, schema, `<root><other/></root>`))
	})
}
