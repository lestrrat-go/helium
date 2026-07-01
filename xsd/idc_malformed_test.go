package xsd_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestMalformedIDConstraint verifies that an identity-constraint declaration
// (xs:unique/xs:key/xs:keyref) missing a required component — @name, the
// <selector> child, or at least one <field> child — is rejected at compile time
// (src-identity-constraint, XSD Structures 3.11) rather than silently accepted
// and reduced to a no-op constraint. Before the fix every malformed case below
// compiled clean and produced a constraint that never fired at validation time.
func TestMalformedIDConstraint(t *testing.T) {
	t.Parallel()

	const valid = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
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
    %s
  </xs:element>
</xs:schema>`

	t.Run("valid key compiles clean", func(t *testing.T) {
		t.Parallel()
		idc := `<xs:key name="k"><xs:selector xpath="item"/><xs:field xpath="@id"/></xs:key>`
		errs := compileSchemaErrors(t, fmt.Sprintf(valid, idc))
		require.Empty(t, errs, "expected no compile error for a well-formed key")
	})

	t.Run("key missing name rejected", func(t *testing.T) {
		t.Parallel()
		idc := `<xs:key><xs:selector xpath="item"/><xs:field xpath="@id"/></xs:key>`
		errs := compileSchemaErrors(t, fmt.Sprintf(valid, idc))
		require.Contains(t, errs, "The attribute 'name' is required but missing.")
	})

	t.Run("key missing selector rejected", func(t *testing.T) {
		t.Parallel()
		idc := `<xs:key name="k"><xs:field xpath="@id"/></xs:key>`
		errs := compileSchemaErrors(t, fmt.Sprintf(valid, idc))
		require.Contains(t, errs, "A child element is missing.")
	})

	t.Run("key missing field rejected", func(t *testing.T) {
		t.Parallel()
		idc := `<xs:key name="k"><xs:selector xpath="item"/></xs:key>`
		errs := compileSchemaErrors(t, fmt.Sprintf(valid, idc))
		require.Contains(t, errs, "Expected is (annotation?, (selector, field+)).")
	})

	t.Run("unique missing field rejected", func(t *testing.T) {
		t.Parallel()
		idc := `<xs:unique name="u"><xs:selector xpath="item"/></xs:unique>`
		errs := compileSchemaErrors(t, fmt.Sprintf(valid, idc))
		require.Contains(t, errs, "Expected is (annotation?, (selector, field+)).")
	})

	t.Run("keyref missing name rejected", func(t *testing.T) {
		t.Parallel()
		idc := `<xs:key name="k"><xs:selector xpath="item"/><xs:field xpath="@id"/></xs:key>` +
			`<xs:keyref refer="k"><xs:selector xpath="item"/><xs:field xpath="@id"/></xs:keyref>`
		errs := compileSchemaErrors(t, fmt.Sprintf(valid, idc))
		require.Contains(t, errs, "The attribute 'name' is required but missing.")
	})

	t.Run("field before selector rejected", func(t *testing.T) {
		t.Parallel()
		idc := `<xs:key name="k"><xs:field xpath="@id"/><xs:selector xpath="item"/></xs:key>`
		errs := compileSchemaErrors(t, fmt.Sprintf(valid, idc))
		require.Contains(t, errs, "The content is not valid. Expected is (annotation?, (selector, field+)).")
	})

	t.Run("duplicate selector rejected", func(t *testing.T) {
		t.Parallel()
		idc := `<xs:key name="k"><xs:selector xpath="item"/><xs:selector xpath="item"/><xs:field xpath="@id"/></xs:key>`
		errs := compileSchemaErrors(t, fmt.Sprintf(valid, idc))
		require.Contains(t, errs, "The content is not valid. Expected is (annotation?, (selector, field+)).")
	})

	t.Run("unexpected xsd child rejected", func(t *testing.T) {
		t.Parallel()
		idc := `<xs:key name="k"><xs:selector xpath="item"/><xs:field xpath="@id"/><xs:attribute name="x"/></xs:key>`
		errs := compileSchemaErrors(t, fmt.Sprintf(valid, idc))
		require.Contains(t, errs, "The content is not valid. Expected is (annotation?, (selector, field+)).")
	})

	t.Run("selector without xpath rejected", func(t *testing.T) {
		t.Parallel()
		idc := `<xs:key name="k"><xs:selector/><xs:field xpath="@id"/></xs:key>`
		errs := compileSchemaErrors(t, fmt.Sprintf(valid, idc))
		require.Contains(t, errs, "Element '{http://www.w3.org/2001/XMLSchema}selector': The attribute 'xpath' is required but missing.")
	})

	t.Run("selector with empty xpath rejected", func(t *testing.T) {
		t.Parallel()
		idc := `<xs:key name="k"><xs:selector xpath=""/><xs:field xpath="@id"/></xs:key>`
		errs := compileSchemaErrors(t, fmt.Sprintf(valid, idc))
		require.Contains(t, errs, "Element '{http://www.w3.org/2001/XMLSchema}selector': The attribute 'xpath' is required but missing.")
	})

	t.Run("field without xpath rejected", func(t *testing.T) {
		t.Parallel()
		idc := `<xs:key name="k"><xs:selector xpath="item"/><xs:field/></xs:key>`
		errs := compileSchemaErrors(t, fmt.Sprintf(valid, idc))
		require.Contains(t, errs, "Element '{http://www.w3.org/2001/XMLSchema}field': The attribute 'xpath' is required but missing.")
	})

	t.Run("foreign direct child rejected", func(t *testing.T) {
		t.Parallel()
		// The IDC content model (annotation?, (selector, field+)) has NO element
		// wildcard. Extension content belongs inside xs:annotation/xs:appinfo, not
		// as an arbitrary foreign-namespaced direct child. libxml2 rejects this with
		// the same content error.
		idc := `<xs:key name="k" xmlns:ext="urn:ext"><xs:selector xpath="item"/><xs:field xpath="@id"/><ext:x/></xs:key>`
		errs := compileSchemaErrors(t, fmt.Sprintf(valid, idc))
		require.Contains(t, errs, "The content is not valid. Expected is (annotation?, (selector, field+)).")
	})

	t.Run("annotation before selector compiles clean", func(t *testing.T) {
		t.Parallel()
		idc := `<xs:key name="k"><xs:annotation><xs:documentation>doc</xs:documentation></xs:annotation>` +
			`<xs:selector xpath="item"/><xs:field xpath="@id"/></xs:key>`
		errs := compileSchemaErrors(t, fmt.Sprintf(valid, idc))
		require.Empty(t, errs, "a leading annotation is permitted by (annotation?, (selector, field+))")
	})

	t.Run("annotation after selector rejected", func(t *testing.T) {
		t.Parallel()
		idc := `<xs:key name="k"><xs:selector xpath="item"/>` +
			`<xs:annotation><xs:documentation>doc</xs:documentation></xs:annotation>` +
			`<xs:field xpath="@id"/></xs:key>`
		errs := compileSchemaErrors(t, fmt.Sprintf(valid, idc))
		require.Contains(t, errs, "The content is not valid. Expected is (annotation?, (selector, field+)).")
	})
}
