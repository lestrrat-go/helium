package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// compileCTASchema compiles src in XSD 1.1 mode and returns the resulting error
// (nil when the schema is valid).
func compileCTASchema(t *testing.T, src string) error {
	t.Helper()
	doc, perr := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, perr)
	_, err := xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), doc)
	return err
}

// TestVersion11CTAStaticErrors covers the XSD 1.1 conditional-type-assignment
// schema-representation constraints on the xs:alternative @test XPath and on the
// {type table} ordering, mirroring the saxonData/CTA cta9001err-cta9003err cases.
func TestVersion11CTAStaticErrors(t *testing.T) {
	// The user types live in urn:t (prefix t), so a t:-prefixed type reference in a
	// @test exercises the user-defined-type rejection (an UNPREFIXED user type is
	// already rejected by the underlying XPath prefix validation).
	const head = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:t="urn:t" targetNamespace="urn:t">
  <xs:complexType name="base"><xs:simpleContent><xs:extension base="xs:string">
    <xs:attribute name="kind" type="xs:string"/></xs:extension></xs:simpleContent></xs:complexType>
  <xs:complexType name="der"><xs:simpleContent><xs:restriction base="t:base"/></xs:simpleContent></xs:complexType>
  <xs:simpleType name="smallInt"><xs:restriction base="xs:int"><xs:maxInclusive value="1"/></xs:restriction></xs:simpleType>`

	t.Run("testless alternative not last is invalid", func(t *testing.T) {
		t.Parallel()
		src := head + `
  <xs:element name="e" type="t:base">
    <xs:alternative type="t:der"/>
    <xs:alternative test="@kind='x'" type="t:der"/>
  </xs:element>
</xs:schema>`
		require.Error(t, compileCTASchema(t, src))
	})

	t.Run("testless final alternative is valid", func(t *testing.T) {
		t.Parallel()
		src := head + `
  <xs:element name="e" type="t:base">
    <xs:alternative test="@kind='x'" type="t:der"/>
    <xs:alternative type="t:der"/>
  </xs:element>
</xs:schema>`
		require.NoError(t, compileCTASchema(t, src))
	})

	t.Run("undefined variable in test is invalid", func(t *testing.T) {
		t.Parallel()
		src := head + `
  <xs:element name="e" type="t:base">
    <xs:alternative test="$kind='x'" type="t:der"/>
  </xs:element>
</xs:schema>`
		require.Error(t, compileCTASchema(t, src))
	})

	t.Run("user-defined type in instance-of test is invalid", func(t *testing.T) {
		t.Parallel()
		src := head + `
  <xs:element name="e" type="t:base">
    <xs:alternative test="@kind instance of t:smallInt" type="t:der"/>
  </xs:element>
</xs:schema>`
		require.Error(t, compileCTASchema(t, src))
	})

	t.Run("built-in type in instance-of test is valid", func(t *testing.T) {
		t.Parallel()
		src := head + `
  <xs:element name="e" type="t:base">
    <xs:alternative test="@kind instance of xs:string" type="t:der"/>
  </xs:element>
</xs:schema>`
		require.NoError(t, compileCTASchema(t, src))
	})

	t.Run("built-in type via non-xs prefix bound to XSD namespace is valid", func(t *testing.T) {
		t.Parallel()
		src := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:x1="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base"><xs:simpleContent><xs:extension base="xs:string">
    <xs:attribute name="kind" type="xs:string"/></xs:extension></xs:simpleContent></xs:complexType>
  <xs:complexType name="der"><xs:simpleContent><xs:restriction base="base"/></xs:simpleContent></xs:complexType>
  <xs:element name="e" type="base">
    <xs:alternative test="@kind instance of x1:string" type="der"/>
  </xs:element>
</xs:schema>`
		require.NoError(t, compileCTASchema(t, src))
	})

	t.Run("cast to user-defined type in test is invalid", func(t *testing.T) {
		t.Parallel()
		src := head + `
  <xs:element name="e" type="t:base">
    <xs:alternative test="@kind cast as t:der = 'x'" type="t:der"/>
  </xs:element>
</xs:schema>`
		require.Error(t, compileCTASchema(t, src))
	})

	t.Run("user-defined type hidden in element() kind test is invalid", func(t *testing.T) {
		t.Parallel()
		src := head + `
  <xs:element name="e" type="t:base">
    <xs:alternative test=". instance of element(*, t:smallInt)" type="t:der"/>
  </xs:element>
</xs:schema>`
		require.Error(t, compileCTASchema(t, src))
	})

	t.Run("user-defined type hidden in path-step kind test is invalid", func(t *testing.T) {
		t.Parallel()
		src := head + `
  <xs:element name="e" type="t:base">
    <xs:alternative test="self::element(*, t:smallInt)" type="t:der"/>
  </xs:element>
</xs:schema>`
		require.Error(t, compileCTASchema(t, src))
	})

	t.Run("user-defined type constructor function call in test is invalid", func(t *testing.T) {
		t.Parallel()
		src := head + `
  <xs:element name="e" type="t:base">
    <xs:alternative test="t:smallInt(@kind) = 1" type="t:der"/>
  </xs:element>
</xs:schema>`
		require.Error(t, compileCTASchema(t, src))
	})

	t.Run("built-in constructor and standard-library functions in test are valid", func(t *testing.T) {
		t.Parallel()
		for _, test := range []string{
			"xs:integer(@kind) = 1",   // built-in type constructor (xs:)
			"fn:string(@kind) = 'x'",  // explicit fn: prefix
			"string(@kind) = 'x'",     // unprefixed -> default function namespace (fn)
			"count(@kind) = 1",        // standard library, unprefixed
			"math:sqrt(2.0) &gt; 1.0", // math: standard library
		} {
			src := head + `
  <xs:element name="e" type="t:base">
    <xs:alternative test="` + test + `" type="t:der"/>
  </xs:element>
</xs:schema>`
			require.NoErrorf(t, compileCTASchema(t, src), "test=%q", test)
		}
	})

	// Braced-URI (Q{uri}local) names carry the namespace directly; the URI-based
	// allowlist must reject a user namespace and accept the XSD namespace in EVERY
	// position — type reference, constructor call, named-function ref, arrow target.
	const xsdNS = "http://www.w3.org/2001/XMLSchema"

	t.Run("braced-uri user names in test are invalid", func(t *testing.T) {
		t.Parallel()
		for _, test := range []string{
			". instance of Q{urn:t}smallInt",     // instance-of type reference
			"@kind cast as Q{urn:t}smallInt = 1", // cast type reference
			"Q{urn:t}smallInt(@kind) = 1",        // constructor call
			"exists(Q{urn:t}f#1)",                // named-function ref
			"(1 => Q{urn:t}f()) = 1",             // arrow target
		} {
			src := head + `
  <xs:element name="e" type="t:base">
    <xs:alternative test="` + test + `" type="t:der"/>
  </xs:element>
</xs:schema>`
			require.Errorf(t, compileCTASchema(t, src), "test=%q", test)
		}
	})

	t.Run("braced-uri built-in names in test are valid", func(t *testing.T) {
		t.Parallel()
		for _, test := range []string{
			". instance of Q{" + xsdNS + "}string",
			"@kind cast as Q{" + xsdNS + "}integer = 1",
			"Q{" + xsdNS + "}integer(@kind) = 1",
		} {
			src := head + `
  <xs:element name="e" type="t:base">
    <xs:alternative test="` + test + `" type="t:der"/>
  </xs:element>
</xs:schema>`
			require.NoErrorf(t, compileCTASchema(t, src), "test=%q", test)
		}
	})

	// An xs:-namespace (or standard-function) name must actually EXIST: a bogus local
	// name in a standard namespace, or a real function called at the wrong arity, is a
	// static error (XPST0008 / XPST0017), not silently swallowed as a non-matching
	// alternative.
	t.Run("unknown standard-namespace names and wrong arity in test are invalid", func(t *testing.T) {
		t.Parallel()
		for _, test := range []string{
			". instance of element(*, xs:noSuchType)", // unknown type in kind test
			"@kind cast as xs:noSuchType = 1",         // unknown cast type
			"xs:noSuchType(@kind) = 1",                // unknown type constructor
			"fn:noSuchFunction(@kind) = 1",            // unknown fn: function
			"fn:concat(@kind) = 'x'",                  // real fn:concat, but arity 1 (needs >= 2)
		} {
			src := head + `
  <xs:element name="e" type="t:base">
    <xs:alternative test="` + test + `" type="t:der"/>
  </xs:element>
</xs:schema>`
			require.Errorf(t, compileCTASchema(t, src), "test=%q", test)
		}
	})

	t.Run("known standard functions at valid arity in test are valid", func(t *testing.T) {
		t.Parallel()
		for _, test := range []string{
			"xs:integer(@kind) = 1",                            // real built-in constructor
			"fn:string(@kind) = 'x'",                           // real fn function
			"fn:concat(@kind, 'y') = 'xy'",                     // real fn:concat at arity 2
			"math:sqrt(2.0) &gt; 1.0",                          // real math function
			". instance of element(*, xs:integer)",             // real built-in type in kind test
			". instance of element(*, xs:untyped)",             // XDM-only type (cta0018/0022)
			"@kind instance of attribute(*, xs:untypedAtomic)", // XDM-only type (cta0019)
		} {
			src := head + `
  <xs:element name="e" type="t:base">
    <xs:alternative test="` + test + `" type="t:der"/>
  </xs:element>
</xs:schema>`
			require.NoErrorf(t, compileCTASchema(t, src), "test=%q", test)
		}
	})

	// schema-element()/schema-attribute() reference global declarations that the CTA
	// static context does not provide (§F.2), and a non-standard helium extension
	// function (fn:flatten) is not in the standard library — both are out of context.
	t.Run("schema-aware node tests and extension functions in test are invalid", func(t *testing.T) {
		t.Parallel()
		for _, test := range []string{
			". instance of schema-element(e)",          // schema-element node test
			"@kind instance of schema-attribute(kind)", // schema-attribute node test
			"count(fn:flatten((1, (2, 3)))) = 3",       // helium extension function fn:flatten
		} {
			src := head + `
  <xs:element name="e" type="t:base">
    <xs:alternative test="` + test + `" type="t:der"/>
  </xs:element>
</xs:schema>`
			require.Errorf(t, compileCTASchema(t, src), "test=%q", test)
		}
	})
}

// TestVersion11CTAStaticIsXSD10ByteIdentical confirms the new CTA static checks
// are gated on XSD 1.1: in 1.0 an xs:alternative is ignored entirely, so a schema
// that would trip a 1.1 CTA static error still compiles.
func TestVersion11CTAStaticIsXSD10ByteIdentical(t *testing.T) {
	src := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="base"><xs:simpleContent><xs:extension base="xs:string">
    <xs:attribute name="kind" type="xs:string"/></xs:extension></xs:simpleContent></xs:complexType>
  <xs:complexType name="der"><xs:simpleContent><xs:restriction base="base"/></xs:simpleContent></xs:complexType>
  <xs:element name="e" type="base">
    <xs:alternative test="$kind='x'" type="der"/>
    <xs:alternative type="der"/>
  </xs:element>
</xs:schema>`
	doc, perr := helium.NewParser().Parse(t.Context(), []byte(src))
	require.NoError(t, perr)
	_, err := xsd.NewCompiler().Compile(t.Context(), doc) // default = XSD 1.0
	require.NoError(t, err)
}

// TestVersion11CTAElementConsistentTypeTables covers the XSD 1.1 extension to
// Element Declarations Consistent (cos-element-consistent): two element particles
// with the same expanded name in one content model must have the SAME {type
// table}. Mirrors saxonData/CTA cta9009err (different tables) and cta9010err
// (table vs no table).
func TestVersion11CTAElementConsistentTypeTables(t *testing.T) {
	const types = `
  <xs:complexType name="zz"><xs:simpleContent><xs:extension base="xs:string">
    <xs:attribute name="type" type="xs:integer"/></xs:extension></xs:simpleContent></xs:complexType>
  <xs:complexType name="zzi"><xs:simpleContent><xs:restriction base="zz">
    <xs:assertion test="$value castable as xs:integer"/></xs:restriction></xs:simpleContent></xs:complexType>
  <xs:complexType name="zzd"><xs:simpleContent><xs:restriction base="zz">
    <xs:assertion test="$value castable as xs:double"/></xs:restriction></xs:simpleContent></xs:complexType>`

	t.Run("different type tables are inconsistent", func(t *testing.T) {
		t.Parallel()
		src := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="zing"/>
  <xs:complexType name="zing"><xs:sequence>
    <xs:element name="a" type="zz"><xs:alternative test="@type='1'" type="zzi"/><xs:alternative test="@type='2'" type="zzd"/></xs:element>
    <xs:element name="a" type="zz"><xs:alternative test="@type='1'" type="zzi"/></xs:element>
  </xs:sequence></xs:complexType>` + types + `
</xs:schema>`
		require.Error(t, compileCTASchema(t, src))
	})

	t.Run("type table vs no table is inconsistent", func(t *testing.T) {
		t.Parallel()
		src := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="zing"/>
  <xs:complexType name="zing"><xs:sequence>
    <xs:element name="a" type="zz"><xs:alternative test="@type='1'" type="zzi"/></xs:element>
    <xs:element name="a" type="zz"/>
  </xs:sequence></xs:complexType>` + types + `
</xs:schema>`
		require.Error(t, compileCTASchema(t, src))
	})

	t.Run("identical type tables are consistent", func(t *testing.T) {
		t.Parallel()
		src := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="zing"/>
  <xs:complexType name="zing"><xs:sequence>
    <xs:element name="a" type="zz"><xs:alternative test="@type='1'" type="zzi"/><xs:alternative test="@type='2'" type="zzd"/></xs:element>
    <xs:element name="a" type="zz"><xs:alternative test="@type='1'" type="zzi"/><xs:alternative test="@type='2'" type="zzd"/></xs:element>
  </xs:sequence></xs:complexType>` + types + `
</xs:schema>`
		require.NoError(t, compileCTASchema(t, src))
	})

	// Same @test TEXT and selected type, but the second element's alternative has a
	// DIFFERENT in-scope namespace context (an extra prefix binding), so the test
	// could resolve differently — the tables are NOT equivalent and EDC must
	// distinguish them. (An EXTRA prefix, not the default namespace, so the unprefixed
	// @type="zzi" still resolves to the no-namespace type and the only difference is
	// the namespace map.)
	t.Run("same test text but different namespace context is inconsistent", func(t *testing.T) {
		t.Parallel()
		src := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="zing"/>
  <xs:complexType name="zing"><xs:sequence>
    <xs:element name="a" type="zz"><xs:alternative test="@type='1'" type="zzi"/></xs:element>
    <xs:element name="a" type="zz"><xs:alternative test="@type='1'" type="zzi" xmlns:extra="urn:extra"/></xs:element>
  </xs:sequence></xs:complexType>` + types + `
</xs:schema>`
		require.Error(t, compileCTASchema(t, src))
	})
}

// TestVersion11CTABaseURIContextNode covers cta0021: fn:base-uri(.) in an
// xs:alternative @test must resolve to the INSTANCE document's URI (the element
// carries no xml:base), so a base-uri-driven alternative selects its type. The
// CTA context node is detached, so the synthetic document must carry the instance
// document URI.
func TestVersion11CTABaseURIContextNode(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="when" type="xs:date">
    <xs:alternative test="ends-with(base-uri(.), 'inst.xml')" type="xs:date"/>
    <xs:alternative type="xs:error"/>
  </xs:element>
</xs:schema>`
	doc, perr := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, perr)
	schema, cerr := xsd.NewCompiler().Version(xsd.Version11).Compile(t.Context(), doc)
	require.NoError(t, cerr)

	t.Run("base-uri matches: xs:date selected, valid date accepted", func(t *testing.T) {
		idoc, ierr := helium.NewParser().BaseURI("file:///tmp/inst.xml").Parse(t.Context(), []byte(`<when>2010-10-16</when>`))
		require.NoError(t, ierr)
		require.NoError(t, xsd.NewValidator(schema).Validate(t.Context(), idoc))
	})

	t.Run("base-uri does not match: xs:error selected, invalid", func(t *testing.T) {
		idoc, ierr := helium.NewParser().BaseURI("file:///tmp/other.xml").Parse(t.Context(), []byte(`<when>2010-10-16</when>`))
		require.NoError(t, ierr)
		require.ErrorIs(t, xsd.NewValidator(schema).Validate(t.Context(), idoc), xsd.ErrValidationFailed)
	})
}

// TestVersion11CTATypeTableRestrictionEquivalent covers the XSD 1.1 rule that when
// an element declaration restricts another, their {type table}s must be both absent
// or both present and EQUIVALENT (Particle Valid (Restriction) clause 4.6). Mirrors
// saxonData/CTA cta0043: a restriction whose stamp alternative selects a different
// type for the same @test than the base is invalid.
func TestVersion11CTATypeTableRestrictionEquivalent(t *testing.T) {
	const types = `
  <xs:complexType name="dateT"><xs:simpleContent><xs:extension base="xs:date">
    <xs:attribute name="type" type="xs:NCName"/></xs:extension></xs:simpleContent></xs:complexType>
  <xs:complexType name="dtsT"><xs:simpleContent><xs:extension base="xs:dateTimeStamp">
    <xs:attribute name="type" type="xs:NCName"/></xs:extension></xs:simpleContent></xs:complexType>
  <xs:complexType name="dtT"><xs:simpleContent><xs:extension base="xs:dateTime">
    <xs:attribute name="type" type="xs:NCName"/></xs:extension></xs:simpleContent></xs:complexType>`

	base := `
  <xs:complexType name="chapType"><xs:sequence>
    <xs:element name="stamp">
      <xs:alternative test="@type='date'" type="dateT"/>
      <xs:alternative test="@type='dateTime'" type="dtsT"/>
    </xs:element>
  </xs:sequence></xs:complexType>`

	t.Run("restriction with non-equivalent type table is invalid", func(t *testing.T) {
		t.Parallel()
		src := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">` + base + `
  <xs:complexType name="appType"><xs:complexContent><xs:restriction base="chapType">
    <xs:sequence><xs:element name="stamp">
      <xs:alternative test="@type='date'" type="dateT"/>
      <xs:alternative test="@type='dateTime'" type="dtT"/>
    </xs:element></xs:sequence>
  </xs:restriction></xs:complexContent></xs:complexType>` + types + `
</xs:schema>`
		require.Error(t, compileCTASchema(t, src))
	})

	t.Run("restriction with equivalent type table is valid", func(t *testing.T) {
		t.Parallel()
		src := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">` + base + `
  <xs:complexType name="appType"><xs:complexContent><xs:restriction base="chapType">
    <xs:sequence><xs:element name="stamp">
      <xs:alternative test="@type='date'" type="dateT"/>
      <xs:alternative test="@type='dateTime'" type="dtsT"/>
    </xs:element></xs:sequence>
  </xs:restriction></xs:complexContent></xs:complexType>` + types + `
</xs:schema>`
		require.NoError(t, compileCTASchema(t, src))
	})
}
