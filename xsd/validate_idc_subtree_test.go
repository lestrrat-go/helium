package xsd_test

import (
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestIDCSubtreeKeyGatherReportsAbsentFieldOnce pins the diagnostic contract
// collectSubtreeKeyTable's descendant key-table gathering must not break: a
// descendant xs:key's own "absent field" error is reported exactly once, no
// matter how many ancestor keyref hosts gather that same descendant occurrence
// out of their own subtree.
//
// "outer" and "inner" are both "node" elements with a keyref referring to
// "ItemKey", declared on the single "items" descendant nested under "inner".
// Because "inner" is itself a descendant of "outer", BOTH hosts' subtree scans
// reach the very same "items" occurrence and its ItemKey constraint: outer's
// keyref gathers it directly, and inner's keyref gathers it again. That
// gathering is deliberately suppressed (it exists only to collect key-sequence
// VALUES for keyref resolution) — the canonical report comes from "items"'s own
// pass-2 walk, which runs unsuppressed exactly once. A cache that leaks a
// suppressed evaluation into the reporting path would double the message; a
// cache that swallows the real pass-2 evaluation would drop it. Either is a
// regression this test catches.
func TestIDCSubtreeKeyGatherReportsAbsentFieldOnce(t *testing.T) {
	t.Parallel()

	const schemaXML = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="items">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" maxOccurs="unbounded">
          <xs:complexType>
            <xs:attribute name="id" type="xs:string"/>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
    <xs:key name="ItemKey">
      <xs:selector xpath="item"/>
      <xs:field xpath="@id"/>
    </xs:key>
  </xs:element>
  <xs:element name="node">
    <xs:complexType>
      <xs:sequence>
        <xs:element ref="items" minOccurs="0"/>
        <xs:element name="ref" type="xs:string"/>
        <xs:element ref="node" minOccurs="0"/>
      </xs:sequence>
    </xs:complexType>
    <xs:keyref name="ItemRef" refer="ItemKey">
      <xs:selector xpath="ref"/>
      <xs:field xpath="."/>
    </xs:keyref>
  </xs:element>
</xs:schema>`

	// "outer" declares no items of its own, so its keyref must gather ItemKey
	// from its subtree, which includes "inner". "inner" ALSO declares no items
	// of its own directly (items lives one level further down), so its keyref
	// gathers the SAME "items" occurrence out of its own (smaller) subtree.
	// One <item> is missing @id, which is an absent xs:key field.
	const instanceXML = `<node>
  <ref>a</ref>
  <node>
    <items>
      <item id="a"/>
      <item/>
    </items>
    <ref>a</ref>
  </node>
</node>`

	sdoc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	schema, err := xsd.NewCompiler().Compile(t.Context(), sdoc)
	require.NoError(t, err)

	idoc, err := helium.NewParser().Parse(t.Context(), []byte(instanceXML))
	require.NoError(t, err)

	var errs string
	verr := validateWithOutput(t, xsd.NewValidator(schema), idoc, &errs)
	require.Error(t, verr, "expected the absent xs:key field to fail validation")

	const wantMessage = "Not all fields of key identity-constraint 'ItemKey' evaluate to a node."
	require.Equal(t, 1, strings.Count(errs, wantMessage),
		"expected %q exactly once in %q", wantMessage, errs)
}
