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
