package xsd_test

import (
	"testing"
)

// TestFixedValueMixedContent verifies cvc-elt.5.2.2 for an element whose content
// type is MIXED and whose declaration carries a fixed value constraint:
//   - an empty element is clause 5.1 (the fixed value is assigned) and is valid;
//   - matching character content with no element children is valid (5.2.2.2.2);
//   - non-matching character content is rejected (5.2.2.2.2);
//   - the presence of element children is rejected regardless of the character
//     content, even when the direct text matches the fixed value (5.2.2.1).
//
// This is a string comparison of the initial value (direct character data,
// element descendants removed), not a typed value-space comparison. The rule is
// version-independent (it applies in XSD 1.0 and 1.1).
func TestFixedValueMixedContent(t *testing.T) {
	// A mixed complex type with an optional element child, declared on a global
	// element carrying fixed="abc".
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" type="CT" fixed="abc"/>
  <xs:complexType name="CT" mixed="true">
    <xs:sequence>
      <xs:element name="a" type="xs:byte" minOccurs="0"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`

	cases := []struct {
		name       string
		instance   string
		wantReject bool
	}{
		{name: "empty element assigns fixed", instance: `<root></root>`},
		{name: "matching character content", instance: `<root>abc</root>`},
		{name: "non-matching character content", instance: `<root>def</root>`, wantReject: true},
		{name: "element child before matching text", instance: `<root>abc<a>1</a></root>`, wantReject: true},
		{name: "element child after matching text", instance: `<root><a>1</a>abc</root>`, wantReject: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			runFixedValueCase(t, schemaXML, tc.instance, tc.wantReject)
		})
	}
}

// TestFixedValueMixedAnyType verifies the mixed-content fixed check also fires
// when the governing type has no declared model group (an xs:anyType / empty
// mixed complex type): the initial value must equal the fixed value.
func TestFixedValueMixedAnyType(t *testing.T) {
	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="root" fixed="abc">
    <xs:complexType mixed="true"/>
  </xs:element>
</xs:schema>`

	t.Run("match", func(t *testing.T) {
		t.Parallel()
		runFixedValueCase(t, schemaXML, `<root>abc</root>`, false)
	})
	t.Run("mismatch", func(t *testing.T) {
		t.Parallel()
		runFixedValueCase(t, schemaXML, `<root>def</root>`, true)
	})
}
