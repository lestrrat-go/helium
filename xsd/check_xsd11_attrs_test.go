package xsd_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// xsd11OnlyAttrCase is one schema carrying a single attribute that exists only
// in the XSD 1.1 schema-for-schemas.
type xsd11OnlyAttrCase struct {
	name   string
	attr   string
	schema string
}

const xsd11AttrIDCBody = `<xs:complexType><xs:sequence><xs:element name="a" maxOccurs="unbounded"/></xs:sequence></xs:complexType>`

var xsd11OnlyAttrCases = []xsd11OnlyAttrCase{
	{"any/notNamespace", "notNamespace", `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="r"><xs:complexType><xs:sequence><xs:any notNamespace="##local"/></xs:sequence></xs:complexType></xs:element>
</xs:schema>`},
	{"any/notQName", "notQName", `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="r"><xs:complexType><xs:sequence><xs:any notQName="##defined"/></xs:sequence></xs:complexType></xs:element>
</xs:schema>`},
	{"anyAttribute/notNamespace", "notNamespace", `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="r"><xs:complexType><xs:anyAttribute notNamespace="##local"/></xs:complexType></xs:element>
</xs:schema>`},
	{"anyAttribute/notQName", "notQName", `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="r"><xs:complexType><xs:anyAttribute notQName="##defined"/></xs:complexType></xs:element>
</xs:schema>`},
	{"complexType/defaultAttributesApply", "defaultAttributesApply", `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="r"><xs:complexType defaultAttributesApply="false"/></xs:element>
</xs:schema>`},
	{"element/targetNamespace", "targetNamespace", `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t">
  <xs:element name="r"><xs:complexType><xs:sequence><xs:element name="a" targetNamespace="urn:t"/></xs:sequence></xs:complexType></xs:element>
</xs:schema>`},
	{"selector/xpathDefaultNamespace", "xpathDefaultNamespace", `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="r">` + xsd11AttrIDCBody + `<xs:unique name="u"><xs:selector xpath="a" xpathDefaultNamespace="##local"/><xs:field xpath="."/></xs:unique></xs:element>
</xs:schema>`},
	{"field/xpathDefaultNamespace", "xpathDefaultNamespace", `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="r">` + xsd11AttrIDCBody + `<xs:unique name="u"><xs:selector xpath="a"/><xs:field xpath="." xpathDefaultNamespace="##local"/></xs:unique></xs:element>
</xs:schema>`},
	{"unique/ref", "ref", `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="r">` + xsd11AttrIDCBody + `<xs:unique name="u"><xs:selector xpath="a"/><xs:field xpath="."/></xs:unique></xs:element>
  <xs:element name="s">` + xsd11AttrIDCBody + `<xs:unique ref="u"/></xs:element>
</xs:schema>`},
	{"key/ref", "ref", `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="r">` + xsd11AttrIDCBody + `<xs:key name="k"><xs:selector xpath="a"/><xs:field xpath="."/></xs:key></xs:element>
  <xs:element name="s">` + xsd11AttrIDCBody + `<xs:key ref="k"/></xs:element>
</xs:schema>`},
	{"keyref/ref", "ref", `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="r">` + xsd11AttrIDCBody + `<xs:key name="k"><xs:selector xpath="a"/><xs:field xpath="."/></xs:key><xs:keyref name="kr" refer="k"><xs:selector xpath="a"/><xs:field xpath="."/></xs:keyref></xs:element>
  <xs:element name="s">` + xsd11AttrIDCBody + `<xs:key name="k2"><xs:selector xpath="a"/><xs:field xpath="."/></xs:key><xs:keyref ref="kr"/></xs:element>
</xs:schema>`},
	{"schema/defaultAttributes", "defaultAttributes", `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" defaultAttributes="g">
  <xs:attributeGroup name="g"/>
  <xs:element name="r"/>
</xs:schema>`},
	{"schema/xpathDefaultNamespace", "xpathDefaultNamespace", `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xpathDefaultNamespace="##local">
  <xs:element name="r"/>
</xs:schema>`},
}

// TestXSD11OnlyAttributes verifies that an attribute the XSD 1.1
// schema-for-schemas adds to an element that also exists in XSD 1.0 is a schema
// error in 1.0 mode (it is outside the 1.0 XML representation, Part 1 §3), and
// compiles cleanly in 1.1 mode.
func TestXSD11OnlyAttributes(t *testing.T) {
	t.Parallel()
	for _, tc := range xsd11OnlyAttrCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			checkXSD11OnlyAttrCase(t, tc)
		})
	}
}

func checkXSD11OnlyAttrCase(t *testing.T, tc xsd11OnlyAttrCase) {
	t.Helper()
	want := "The attribute '" + tc.attr + "' is not allowed."

	_, errs10, cerr10 := compileWith(t, xsd.Version10, tc.schema)
	require.Error(t, cerr10, "1.0 must reject %s", tc.name)
	require.Contains(t, errs10, want)

	_, errs11, cerr11 := compileWith(t, xsd.Version11, tc.schema)
	require.NoError(t, cerr11, "1.1 must accept %s: %s", tc.name, errs11)
}

// TestXSD11OnlyAttributesConditionalInclusion verifies that a 1.1-only attribute
// on an element that conditional inclusion removes in 1.0 (vc:minVersion="1.1")
// is not reported: the pruned element never reaches the attribute check.
func TestXSD11OnlyAttributesConditionalInclusion(t *testing.T) {
	t.Parallel()
	const schema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning">
  <xs:element name="r">
    <xs:complexType>
      <xs:choice>
        <xs:any vc:minVersion="1.1" notNamespace="##local" processContents="lax"/>
        <xs:any vc:maxVersion="1.1" namespace="##other" processContents="lax"/>
      </xs:choice>
    </xs:complexType>
  </xs:element>
</xs:schema>`
	_, errs, err := compileWith(t, xsd.Version10, schema)
	require.NoError(t, err, errs)
}
