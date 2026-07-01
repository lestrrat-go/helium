package xsd_test

import (
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestIDCMalformedXPath covers the case where an identity constraint's selector
// or field XPath is not a valid XPath expression. Previously the compile error
// was silently dropped, leaving the constraint disabled at validation time so a
// schema with a duplicate key would wrongly validate. A malformed selector/field
// XPath must now be a fatal schema compilation error.
func TestIDCMalformedXPath(t *testing.T) {
	t.Parallel()

	compileErrors := func(t *testing.T, schemaXML string) string {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, err = xsd.NewCompiler().Label("test.xsd").ErrorHandler(collector).Compile(t.Context(), doc)
		requireCompileResultErr(t, err)
		_, errors := partitionCompileErrors(collector.Errors())
		return errors
	}

	t.Run("malformed selector xpath", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" maxOccurs="unbounded">
          <xs:complexType>
            <xs:attribute name="id" type="xs:string"/>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
    <xs:unique name="itemKey">
      <xs:selector xpath="["/>
      <xs:field xpath="@id"/>
    </xs:unique>
  </xs:element>
</xs:schema>`
		require.Contains(t, compileErrors(t, schemaXML), "is not a valid selector")
	})

	t.Run("malformed field xpath", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" maxOccurs="unbounded">
          <xs:complexType>
            <xs:attribute name="id" type="xs:string"/>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
    <xs:unique name="itemKey">
      <xs:selector xpath="item"/>
      <xs:field xpath="@"/>
    </xs:unique>
  </xs:element>
</xs:schema>`
		require.Contains(t, compileErrors(t, schemaXML), "is not a valid field")
	})

	t.Run("valid xpath compiles clean", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" maxOccurs="unbounded">
          <xs:complexType>
            <xs:attribute name="id" type="xs:string"/>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
    <xs:unique name="itemKey">
      <xs:selector xpath="item"/>
      <xs:field xpath="@id"/>
    </xs:unique>
  </xs:element>
</xs:schema>`
		require.Empty(t, compileErrors(t, schemaXML))
	})
}

// TestIDCXPathSubset verifies that identity-constraint selector/field XPath
// expressions are gated against the restricted XPath subset XSD permits
// (Structures 3.11.6). The full XPath 1.0 grammar is broader than that subset,
// so syntactically valid but out-of-subset constructs (string/number literals,
// function calls, variable references, operators, predicates, the attribute
// axis in a selector) must be rejected at schema-compile time, while genuine
// subset expressions still compile clean.
func TestIDCXPathSubset(t *testing.T) {
	t.Parallel()

	const (
		okSelector = "item" // a valid in-subset selector
		okField    = "@id"  // a valid in-subset field
	)

	schema := func(selector, field string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" maxOccurs="unbounded">
          <xs:complexType>
            <xs:attribute name="id" type="xs:string"/>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
    <xs:unique name="itemKey">
      <xs:selector xpath="` + selector + `"/>
      <xs:field xpath="` + field + `"/>
    </xs:unique>
  </xs:element>
</xs:schema>`
	}

	t.Run("selector rejected", func(t *testing.T) {
		t.Parallel()
		// String/number literals, function calls, predicates, the attribute
		// axis, absolute paths and operators are all outside the subset. The
		// spelled-out 'self::node()' and a repeated/mid-path './/' normalize to
		// the same AST as the permitted '.' / leading './/' steps, so they are
		// caught lexically over the raw expression.
		for _, sel := range []string{
			"'literal'", "42", "foo()", "item[@id='x']", "@id", "/item", "item + 1",
			"self::node()", ".//.//item", ".//item/.//item",
		} {
			t.Run(sel, func(t *testing.T) {
				t.Parallel()
				_, errs := compileXSD(t, schema(sel, okField))
				require.NotEmpty(t, errs, "out-of-subset selector must be rejected")
				require.Contains(t, errs, "is not a valid selector")
			})
		}
	})

	t.Run("field rejected", func(t *testing.T) {
		t.Parallel()
		// Like selectors, plus a non-final attribute step is out of subset.
		for _, f := range []string{
			"'literal'", "string(@id)", "$x", "@id[1]", "@id/item",
			"self::node()", ".//id/.//id",
		} {
			t.Run(f, func(t *testing.T) {
				t.Parallel()
				_, errs := compileXSD(t, schema(okSelector, f))
				require.NotEmpty(t, errs, "out-of-subset field must be rejected")
				require.Contains(t, errs, "is not a valid field")
			})
		}
	})

	t.Run("accepted", func(t *testing.T) {
		t.Parallel()
		for name, tc := range map[string]struct {
			selector, field string
		}{
			"child name test":     {okSelector, okField},
			"self step":           {".", okField},
			"descendant-or-self":  {".//item", okField},
			"descendant then dot": {".//.", okField},
			// XPath permits insignificant whitespace, so '. //.' / '.// .' are
			// the same leading './/' step as './/.'.
			"whitespace descendant":       {". //.", okField},
			"trailing-space descendant":   {".// .", okField},
			"wildcard name test":          {"*", okField},
			"union of paths":              {".//item | item", okField},
			"explicit child axis":         {"child::item", okField},
			"explicit attribute in field": {okSelector, "attribute::id"},
			"field self":                  {okSelector, "."},
			"multi-step child":            {"item/item", okField},
		} {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				_, errs := compileXSD(t, schema(tc.selector, tc.field))
				require.Empty(t, errs, "valid subset XPath must compile clean")
			})
		}
	})

	t.Run("unbound prefix rejected", func(t *testing.T) {
		t.Parallel()
		// A QName name test whose prefix is not bound in the constraint's in-scope
		// namespaces cannot be resolved (XSD Structures 3.11.6.1), so it is a
		// schema error; a bound prefix (p → urn:p) compiles clean.
		prefixed := func(selector, field string) string {
			return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:p="urn:p">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" maxOccurs="unbounded">
          <xs:complexType>
            <xs:attribute name="id" type="xs:string"/>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
    <xs:unique name="itemKey">
      <xs:selector xpath="` + selector + `"/>
      <xs:field xpath="` + field + `"/>
    </xs:unique>
  </xs:element>
</xs:schema>`
		}
		t.Run("selector", func(t *testing.T) {
			t.Parallel()
			_, errs := compileXSD(t, prefixed("q:item", okField))
			require.Contains(t, errs, "is not a valid selector")
		})
		t.Run("field", func(t *testing.T) {
			t.Parallel()
			_, errs := compileXSD(t, prefixed(okSelector, "q:id"))
			require.Contains(t, errs, "is not a valid field")
		})
		t.Run("bound prefix accepted", func(t *testing.T) {
			t.Parallel()
			_, errs := compileXSD(t, prefixed("p:item | item", okField))
			require.Empty(t, errs, "a bound-prefix name test must compile clean")
		})
	})
}

// TestIDCKeyRefUnresolvedRefer covers an xs:keyref whose @refer names a
// key/unique constraint that does not exist. Previously such a keyref was
// silently skipped at validation time, so referential integrity was not
// enforced. The unresolved @refer must now be a fatal schema compilation error.
func TestIDCKeyRefUnresolvedRefer(t *testing.T) {
	t.Parallel()

	compileErrors := func(t *testing.T, schemaXML string) string {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		_, err = xsd.NewCompiler().Label("test.xsd").ErrorHandler(collector).Compile(t.Context(), doc)
		requireCompileResultErr(t, err)
		_, errors := partitionCompileErrors(collector.Errors())
		return errors
	}

	t.Run("keyref refers to missing key", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" maxOccurs="unbounded">
          <xs:complexType>
            <xs:attribute name="ref" type="xs:string"/>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
    <xs:keyref name="itemRef" refer="missingKey">
      <xs:selector xpath="item"/>
      <xs:field xpath="@ref"/>
    </xs:keyref>
  </xs:element>
</xs:schema>`
		require.Contains(t, compileErrors(t, schemaXML), "missingKey")
	})

	t.Run("keyref refers to existing key compiles clean", func(t *testing.T) {
		t.Parallel()
		schemaXML := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" maxOccurs="unbounded">
          <xs:complexType>
            <xs:attribute name="id" type="xs:string"/>
          </xs:complexType>
        </xs:element>
        <xs:element name="ref" maxOccurs="unbounded">
          <xs:complexType>
            <xs:attribute name="to" type="xs:string"/>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
    <xs:key name="itemKey">
      <xs:selector xpath="item"/>
      <xs:field xpath="@id"/>
    </xs:key>
    <xs:keyref name="itemRef" refer="itemKey">
      <xs:selector xpath="ref"/>
      <xs:field xpath="@to"/>
    </xs:keyref>
  </xs:element>
</xs:schema>`
		require.Empty(t, compileErrors(t, schemaXML))
	})
}

// TestIDCWorkingKeyRefValidation confirms that a valid schema with a working
// unique/key/keyref still validates conforming instances and rejects
// non-conforming ones, ensuring the compile-time checks do not break the
// validation-time enforcement path.
func TestIDCWorkingKeyRefValidation(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" maxOccurs="unbounded">
          <xs:complexType>
            <xs:attribute name="id" type="xs:string"/>
          </xs:complexType>
        </xs:element>
        <xs:element name="ref" maxOccurs="unbounded">
          <xs:complexType>
            <xs:attribute name="to" type="xs:string"/>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
    <xs:key name="itemKey">
      <xs:selector xpath="item"/>
      <xs:field xpath="@id"/>
    </xs:key>
    <xs:keyref name="itemRef" refer="itemKey">
      <xs:selector xpath="ref"/>
      <xs:field xpath="@to"/>
    </xs:keyref>
  </xs:element>
</xs:schema>`

	compile := func(t *testing.T) xsd.Validator {
		t.Helper()
		doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		s, err := xsd.NewCompiler().ErrorHandler(collector).Compile(t.Context(), doc)
		require.NoError(t, err)
		_, compileErrors := partitionCompileErrors(collector.Errors())
		require.Empty(t, compileErrors, "unexpected compile errors")
		return xsd.NewValidator(s)
	}

	t.Run("matching keyref validates", func(t *testing.T) {
		t.Parallel()
		v := compile(t)
		doc, err := helium.NewParser().Parse(t.Context(),
			[]byte(`<root><item id="a"/><item id="b"/><ref to="a"/></root>`))
		require.NoError(t, err)
		var errs string
		err = validateWithOutput(t, v, doc, &errs)
		require.NoError(t, err, "expected valid, got: %s", errs)
	})

	t.Run("dangling keyref fails", func(t *testing.T) {
		t.Parallel()
		v := compile(t)
		doc, err := helium.NewParser().Parse(t.Context(),
			[]byte(`<root><item id="a"/><ref to="missing"/></root>`))
		require.NoError(t, err)
		var errs string
		err = validateWithOutput(t, v, doc, &errs)
		require.Error(t, err)
		require.Contains(t, errs, "No match found")
	})

	t.Run("duplicate key fails", func(t *testing.T) {
		t.Parallel()
		v := compile(t)
		doc, err := helium.NewParser().Parse(t.Context(),
			[]byte(`<root><item id="a"/><item id="a"/></root>`))
		require.NoError(t, err)
		var errs string
		err = validateWithOutput(t, v, doc, &errs)
		require.Error(t, err)
		require.Contains(t, errs, "Duplicate key-sequence")
	})
}
