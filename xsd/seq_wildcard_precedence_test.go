package xsd_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestVersion11SeqLeadingWildcardElementPrecedence covers XSD 1.1
// element-over-wildcard precedence for the SEQUENCE case
// (sequenceElementReservedLimit): a leading minOccurs=0 wildcard must not
// greedily consume children that a later required element particle in the same
// sequence is responsible for. This is the shape produced by extending a base
// whose content model is just a lax unbounded wildcard (W3C
// ibmMeta/typeAlternatives.testSet/s3_12v09).
func TestVersion11SeqLeadingWildcardElementPrecedence(t *testing.T) {
	// Mirrors the s3_12v09 effective model: a complexType whose base content is
	// only `xs:any{0,unbounded}` extended with required named elements, yielding
	// sequence( any{0,unbounded} lax, state, currency, zip ).
	const extSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:t="urn:t" targetNamespace="urn:t" elementFormDefault="unqualified">
  <xs:element name="addr" type="t:usAddr"/>
  <xs:complexType name="base" mixed="true">
    <xs:sequence>
      <xs:any processContents="lax" minOccurs="0" maxOccurs="unbounded"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="usAddr" mixed="true">
    <xs:complexContent>
      <xs:extension base="t:base">
        <xs:sequence>
          <xs:element name="state" type="xs:string"/>
          <xs:element name="currency" type="xs:string"/>
          <xs:element name="zip" type="xs:int"/>
        </xs:sequence>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`

	t.Run("1.1 leading wildcard yields to following required elements (valid)", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), extSchema,
			`<t:addr xmlns:t="urn:t"><state>TX</state><currency>USD</currency><zip>75244</zip></t:addr>`)
		require.NoError(t, err)
	})

	// Reachability gating regression (gauntlet finding): the reservation must only
	// fire for a later element that is REACHABLE from the current position — i.e.
	// every intervening particle is emptiable. A required element BETWEEN the
	// wildcard and a trailing OPTIONAL element must not have its child stolen by
	// that optional element. Model:
	//   sequence( any[##targetNamespace, notQName="t:b"]{0,unbounded}, b, a? )
	// Instance <a/><b/><a/> is VALID: a0->wildcard, b->element b, a2->element a.
	const reachSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:t="urn:t" targetNamespace="urn:t" elementFormDefault="qualified">
  <xs:element name="root" type="t:rootType"/>
  <xs:complexType name="rootType">
    <xs:sequence>
      <xs:any namespace="##targetNamespace" notQName="t:b" processContents="lax" minOccurs="0" maxOccurs="unbounded"/>
      <xs:element name="b" type="xs:string"/>
      <xs:element name="a" type="xs:string" minOccurs="0"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`

	t.Run("1.1 optional trailing element does not steal a required element's child (valid)", func(t *testing.T) {
		t.Parallel()
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), reachSchema,
			`<t:root xmlns:t="urn:t"><t:a/><t:b/><t:a/></t:root>`)
		require.NoError(t, err)
	})

	t.Run("1.1 still reports a genuinely missing required element", func(t *testing.T) {
		t.Parallel()
		// No <t:b> present: the wildcard admits the t:a children, but required b is
		// truly absent, so validation must still fail.
		err := compileAndValidateV(t, xsd.NewCompiler().Version(xsd.Version11), reachSchema,
			`<t:root xmlns:t="urn:t"><t:a/><t:a/></t:root>`)
		require.ErrorIs(t, err, xsd.ErrValidationFailed)
	})
}
