package xsd_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestOpenContent_MixedEmptyTypeEnforced covers the gauntlet finding that a
// mixed="true" complex type with NO model group must ENFORCE its effective open
// content at validation time: children outside the open-content wildcard are
// rejected, while text (mixed) and in-wildcard children are accepted.
func TestOpenContent_MixedEmptyTypeEnforced(t *testing.T) {
	t.Parallel()
	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xmlns:t="urn:t">
  <xs:defaultOpenContent>
    <xs:any namespace="urn:open" processContents="skip"/>
  </xs:defaultOpenContent>
  <xs:element name="root"><xs:complexType mixed="true"/></xs:element>
</xs:schema>`

	t.Run("child outside the open wildcard is rejected", func(t *testing.T) {
		t.Parallel()
		require.Error(t, validateOC(t, schema,
			`<t:root xmlns:t="urn:t"><bad/></t:root>`),
			"a child not in urn:open must be rejected by the mixed type's open content")
	})

	t.Run("text and an in-namespace child are accepted", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, validateOC(t, schema,
			`<t:root xmlns:t="urn:t" xmlns:o="urn:open">some text<o:ok/></t:root>`),
			"mixed text plus a urn:open child must be accepted")
	})
}

// TestOpenContent_GrammarStrictChildren covers the gauntlet finding that the local
// child grammar of <xs:openContent>/<xs:defaultOpenContent> — (annotation?, any?) —
// is enforced: a DUPLICATE <xs:openContent> sibling, a stray non-(annotation|any)
// child, and an annotation appearing AFTER the any are all schema errors.
func TestOpenContent_GrammarStrictChildren(t *testing.T) {
	t.Parallel()
	const head = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">`

	cases := map[string]string{
		"duplicate openContent siblings": head + `
  <xs:element name="doc"><xs:complexType>
    <xs:openContent mode="suffix"><xs:any namespace="urn:a" processContents="lax"/></xs:openContent>
    <xs:openContent mode="suffix"><xs:any namespace="urn:b" processContents="lax"/></xs:openContent>
    <xs:sequence><xs:element name="a"/></xs:sequence>
  </xs:complexType></xs:element></xs:schema>`,
		"stray element after any in openContent": head + `
  <xs:element name="doc"><xs:complexType>
    <xs:openContent mode="suffix">
      <xs:any namespace="urn:a" processContents="lax"/>
      <xs:element name="stray"/>
    </xs:openContent>
    <xs:sequence><xs:element name="a"/></xs:sequence>
  </xs:complexType></xs:element></xs:schema>`,
		"annotation after any in openContent": head + `
  <xs:element name="doc"><xs:complexType>
    <xs:openContent mode="suffix">
      <xs:any namespace="urn:a" processContents="lax"/>
      <xs:annotation><xs:documentation>late</xs:documentation></xs:annotation>
    </xs:openContent>
    <xs:sequence><xs:element name="a"/></xs:sequence>
  </xs:complexType></xs:element></xs:schema>`,
		"stray element after any in defaultOpenContent": head + `
  <xs:defaultOpenContent>
    <xs:any namespace="urn:a" processContents="lax"/>
    <xs:element name="stray"/>
  </xs:defaultOpenContent>
  <xs:element name="doc"><xs:complexType><xs:sequence><xs:element name="a"/></xs:sequence></xs:complexType></xs:element></xs:schema>`,
		"annotation after any in defaultOpenContent": head + `
  <xs:defaultOpenContent>
    <xs:any namespace="urn:a" processContents="lax"/>
    <xs:annotation><xs:documentation>late</xs:documentation></xs:annotation>
  </xs:defaultOpenContent>
  <xs:element name="doc"><xs:complexType><xs:sequence><xs:element name="a"/></xs:sequence></xs:complexType></xs:element></xs:schema>`,
	}
	for name, schema := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, _, cerr := compileV11(t, schema)
			require.Error(t, cerr, "invalid open-content child grammar must be rejected")
		})
	}
}
