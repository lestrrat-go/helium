package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// compileAndValidate compiles schemaXML with the given compiler and validates
// instanceXML against it, returning the validation error (or nil).
func compileAndValidateV(t *testing.T, c xsd.Compiler, schemaXML, instanceXML string) error {
	t.Helper()
	schemaDOC, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	schema, err := c.Compile(t.Context(), schemaDOC)
	require.NoError(t, err)
	doc, err := helium.NewParser().Parse(t.Context(), []byte(instanceXML))
	require.NoError(t, err)
	return xsd.NewValidator(schema).Validate(t.Context(), doc)
}

// TestVersionToggle exercises the XSD-version selection end-to-end through the
// public Compiler API and the vc:minVersion auto-detection, using the "+INF"
// xs:double lexical form (valid only in XSD 1.1).
func TestVersionToggle(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="v" type="xs:double"/>
</xs:schema>`
	const schemaVC11 = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning" vc:minVersion="1.1">
  <xs:element name="v" type="xs:double"/>
</xs:schema>`
	const instancePlusINF = `<v>+INF</v>`
	const instanceINF = `<v>INF</v>`

	t.Run("default (1.0) rejects +INF", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler(), schemaXML, instancePlusINF)
		require.ErrorIs(t, err, xsd.ErrValidationFailed)
	})

	t.Run("explicit 1.0 rejects +INF", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version10), schemaXML, instancePlusINF)
		require.ErrorIs(t, err, xsd.ErrValidationFailed)
	})

	t.Run("explicit 1.1 accepts +INF", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML, instancePlusINF)
		require.NoError(t, err)
	})

	t.Run("vc:minVersion=1.1 auto-detects 1.1 and accepts +INF", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler(), schemaVC11, instancePlusINF)
		require.NoError(t, err)
	})

	t.Run("explicit 1.0 overrides vc:minVersion=1.1", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version10), schemaVC11, instancePlusINF)
		require.ErrorIs(t, err, xsd.ErrValidationFailed)
	})

	t.Run("plain INF accepted in both versions", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, compileAndValidateV(t, xsd.NewCompiler(), schemaXML, instanceINF))
		require.NoError(t, compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML, instanceINF))
	})
}

func TestVersion10LegacyGMonthInstanceLexical(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="v" type="xs:gMonth"/>
</xs:schema>`

	for _, instance := range []string{`<v>--03--</v>`, `<v>--05---05:00</v>`} {
		t.Run("xsd10 accepts "+instance, func(t *testing.T) {
			t.Parallel()
			err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version10), schemaXML, instance)
			require.NoError(t, err)
		})

		t.Run("xsd11 rejects "+instance, func(t *testing.T) {
			t.Parallel()
			err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schemaXML, instance)
			require.ErrorIs(t, err, xsd.ErrValidationFailed)
		})
	}
}

func TestVersion10LegacyGMonthPatternRestrictionDoesNotAcceptLegacyLexical(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="v">
    <xs:simpleType>
      <xs:restriction base="xs:gMonth">
        <xs:pattern value="--[0-9]{2}--"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`

	err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version10), schemaXML, `<v>--03--</v>`)
	require.ErrorIs(t, err, xsd.ErrValidationFailed)
}

func TestVersion10LegacyGMonthFacetLexicalRejected(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="v">
    <xs:simpleType>
      <xs:restriction base="xs:gMonth">
        <xs:enumeration value="--10--"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`

	doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	_, err = xsd.NewCompiler().Version(xsd.Version10).Compile(t.Context(), doc)
	require.ErrorIs(t, err, xsd.ErrCompilationFailed)
}

// TestVersion11BuiltinTypes verifies the XSD 1.1-only built-in datatypes are
// registered (and resolve) only in 1.1 mode, and validate per their lexical
// space.
func TestVersion11BuiltinTypes(t *testing.T) {
	const schemaDTS = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="v" type="xs:dateTimeStamp"/>
</xs:schema>`

	t.Run("1.1 resolves xs:dateTimeStamp and validates", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schemaDTS, `<v>2020-01-01T00:00:00Z</v>`)
		require.NoError(t, err)
	})

	t.Run("1.1 rejects xs:dateTimeStamp without timezone", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schemaDTS, `<v>2020-01-01T00:00:00</v>`)
		require.ErrorIs(t, err, xsd.ErrValidationFailed)
	})

	t.Run("1.0 fails to compile a schema referencing xs:dateTimeStamp", func(t *testing.T) {
		t.Parallel()
		schemaDOC, err := helium.NewParser().Parse(t.Context(), []byte(schemaDTS))
		require.NoError(t, err)
		_, err = xsd.NewCompiler().Compile(t.Context(), schemaDOC)
		require.ErrorIs(t, err, xsd.ErrCompilationFailed)
	})
}

// TestVersion11UnionActiveMember covers the XSD-version threading through union
// active-member resolution (fixedUnionActiveMember): a 1.1-only lexical form
// ("+INF" for xs:double) appearing inside a union fixed-value or enumeration
// literal must be accepted under 1.1 — not rejected because the throwaway
// validation context built during active-member resolution defaulted to
// Version10. The union excludes xs:string so "+INF" is not trivially valid as a
// string member; in 1.0 "+INF" is valid against neither member, so the instance
// is rejected.
func TestVersion11UnionActiveMember(t *testing.T) {
	const schemaFixed = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="v" fixed="+INF">
    <xs:simpleType>
      <xs:union memberTypes="xs:double xs:date"/>
    </xs:simpleType>
  </xs:element>
</xs:schema>`
	const schemaEnum = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="v">
    <xs:simpleType>
      <xs:restriction>
        <xs:simpleType>
          <xs:union memberTypes="xs:double xs:date"/>
        </xs:simpleType>
        <xs:enumeration value="+INF"/>
      </xs:restriction>
    </xs:simpleType>
  </xs:element>
</xs:schema>`

	t.Run("1.1 union fixed value +INF accepts instance +INF", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schemaFixed, `<v>+INF</v>`)
		require.NoError(t, err)
	})

	t.Run("1.0 union fixed value +INF rejects at compile time", func(t *testing.T) {
		t.Parallel()
		// In 1.0 "+INF" is valid against neither union member, so the element's fixed
		// value is invalid against its type — an Element Default Valid (§3.3.6) schema
		// error caught at compile time (version-independent), before any instance.
		schemaDOC, err := helium.NewParser().Parse(t.Context(), []byte(schemaFixed))
		require.NoError(t, err)
		_, err = xsd.NewCompiler().Version(xsd.Version10).Compile(t.Context(), schemaDOC)
		require.ErrorIs(t, err, xsd.ErrCompilationFailed)
	})

	t.Run("1.1 union enumeration +INF accepts instance +INF", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schemaEnum, `<v>+INF</v>`)
		require.NoError(t, err)
	})

	t.Run("1.0 union enumeration +INF rejects instance +INF", func(t *testing.T) {
		t.Parallel()
		// In 1.0 "+INF" is valid against neither union member, so the enumeration
		// literal is rejected at compile time; the schema does not accept "+INF".
		schemaDOC, err := helium.NewParser().Parse(t.Context(), []byte(schemaEnum))
		require.NoError(t, err)
		_, err = xsd.NewCompiler().Version(xsd.Version10).Compile(t.Context(), schemaDOC)
		require.ErrorIs(t, err, xsd.ErrCompilationFailed)
	})
}

// TestVersion11UnionAttribute covers the XSD-version threading through ATTRIBUTE
// value validation. Both the compile-time attribute default/fixed constraint
// check and the runtime instance-attribute validation must honor the schema's
// version: a 1.1-only lexical form ("+INF" for xs:double) on a union(xs:double
// xs:date) attribute is accepted under 1.1 and rejected under 1.0. Without the
// version-aware path these sites built a Version10 validation context and would
// reject "+INF" even in 1.1 mode.
func TestVersion11UnionAttribute(t *testing.T) {
	// Plain union attribute (no fixed): the instance value exercises the runtime
	// attribute-validation site directly. The schema compiles in both versions.
	const schemaPlain = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="v">
    <xs:complexType>
      <xs:attribute name="a">
        <xs:simpleType>
          <xs:union memberTypes="xs:double xs:date"/>
        </xs:simpleType>
      </xs:attribute>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	// Union attribute with fixed="+INF": exercises the compile-time attribute
	// default/fixed constraint check.
	const schemaFixed = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="v">
    <xs:complexType>
      <xs:attribute name="a" fixed="+INF">
        <xs:simpleType>
          <xs:union memberTypes="xs:double xs:date"/>
        </xs:simpleType>
      </xs:attribute>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	t.Run("1.1 union attribute accepts instance +INF", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schemaPlain, `<v a="+INF"/>`)
		require.NoError(t, err)
	})

	t.Run("1.0 union attribute rejects instance +INF", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version10), schemaPlain, `<v a="+INF"/>`)
		require.ErrorIs(t, err, xsd.ErrValidationFailed)
	})

	t.Run("1.1 union attribute fixed +INF accepts instance +INF", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), schemaFixed, `<v a="+INF"/>`)
		require.NoError(t, err)
	})

	t.Run("1.0 union attribute fixed +INF rejects at compile time", func(t *testing.T) {
		t.Parallel()
		// In 1.0 "+INF" is valid against neither union member, so the fixed
		// constraint literal is rejected at compile time.
		schemaDOC, err := helium.NewParser().Parse(t.Context(), []byte(schemaFixed))
		require.NoError(t, err)
		_, err = xsd.NewCompiler().Version(xsd.Version10).Compile(t.Context(), schemaDOC)
		require.ErrorIs(t, err, xsd.ErrCompilationFailed)
	})
}
