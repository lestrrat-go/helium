package xsd_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// TestValidateDocumentLevelElements pins how Validate treats a document that
// carries more than one document-level element, which only the DOM API can
// build. Every element child of the document node is validated as a root
// against the global element declarations, in document order, and the
// diagnostics come out in that same order.
func TestValidateDocumentLevelElements(t *testing.T) {
	t.Parallel()

	const schemaSrc = `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="a" type="xs:int"/>
  <xs:element name="b">
    <xs:complexType>
      <xs:attribute name="n" type="xs:int" use="required"/>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	schemaDoc, err := helium.NewParser().Parse(t.Context(), []byte(schemaSrc))
	require.NoError(t, err)
	schema, err := xsd.NewCompiler().Compile(t.Context(), schemaDoc)
	require.NoError(t, err)

	doc := helium.NewDefaultDocument()
	a, err := doc.CreateElement("a")
	require.NoError(t, err)
	require.NoError(t, a.AddChild(doc.CreateText([]byte("not-an-int"))))
	require.NoError(t, doc.AddChild(a))

	require.NoError(t, doc.AddChild(doc.CreateComment([]byte("between roots"))))

	b, err := doc.CreateElement("b")
	require.NoError(t, err)
	require.NoError(t, doc.AddChild(b))

	c, err := doc.CreateElement("c")
	require.NoError(t, err)
	require.NoError(t, doc.AddChild(c))

	var errs string
	err = validateWithOutput(t, xsd.NewValidator(schema).Label("multi.xml"), doc, &errs)
	require.ErrorIs(t, err, xsd.ErrValidationFailed)
	require.Equal(t, expectedDocumentLevelErrors, errs)
}

const expectedDocumentLevelErrors = `multi.xml:0: Schemas validity error : Element 'a': 'not-an-int' is not a valid value of the atomic type 'xs:int'.
multi.xml:0: Schemas validity error : Element 'b': The attribute 'n' is required but missing.
multi.xml:0: Schemas validity error : Element 'c': No matching global declaration available for the validation root.
`

// attrValidationCase is one schema/instance pair for TestValidateAttributes.
// want pins the complete observable result: every diagnostic in order, the
// serialized instance after validation (defaulted attributes and namespace
// fixups included), and the PSVI type annotation of every element and
// attribute in document order.
type attrValidationCase struct {
	name     string
	version  xsd.Version
	schema   string
	instance string
	want     string
}

// TestValidateAttributes pins attribute validation end to end: declared,
// prohibited, required, defaulted and fixed uses; attribute wildcards; xsi:
// attributes; values whose types read the in-scope namespaces (QName,
// NOTATION, lists and unions of them, and XSD 1.1 assertion facets) with the
// prefixes declared on an ancestor or on the element itself; simple-content
// elements with QName values; and an element carrying more attribute uses
// than fit a small fixed-size set.
func TestValidateAttributes(t *testing.T) {
	t.Parallel()

	for _, tc := range attrValidationCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, runAttrValidationCase(t, tc))
		})
	}
}

func runAttrValidationCase(t *testing.T, tc attrValidationCase) string {
	t.Helper()
	schemaDoc, err := helium.NewParser().Parse(t.Context(), []byte(tc.schema))
	require.NoError(t, err)
	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	schema, err := xsd.NewCompiler().Version(tc.version).Label("test.xsd").ErrorHandler(collector).Compile(t.Context(), schemaDoc)
	require.NoError(t, err)

	doc, err := helium.NewParser().Parse(t.Context(), []byte(tc.instance))
	require.NoError(t, err)
	var ann xsd.TypeAnnotations
	var errs string
	verr := validateWithOutput(t, xsd.NewValidator(schema).Label("test.xml").Annotations(&ann), doc, &errs)

	var b strings.Builder
	for _, e := range collector.Errors() {
		b.WriteString(e.Error())
	}
	if verr != nil {
		b.WriteString("result: " + verr.Error() + "\n")
	} else {
		b.WriteString("result: valid\n")
	}
	b.WriteString(errs)
	b.WriteString("--- document\n")
	var buf bytes.Buffer
	require.NoError(t, helium.Write(&buf, doc.DocumentElement()))
	b.WriteString(buf.String())
	b.WriteString("\n--- annotations\n")
	for n := range helium.Descendants(doc) {
		elem, ok := n.(*helium.Element)
		if !ok {
			continue
		}
		fmt.Fprintf(&b, "%s = %s\n", elem.LocalName(), ann[elem])
		for a := range helium.Attributes(elem) {
			fmt.Fprintf(&b, "%s/@%s = %s\n", elem.LocalName(), helium.ClarkName(a.URI(), a.LocalName()), ann[a])
		}
	}
	return b.String()
}

// manyAttrSchema declares n attribute uses a01..aNN on one element: every
// fifth is required, every seventh has a default, every eleventh is fixed,
// and every thirteenth is prohibited. The uses alternate between xs:int and
// xs:QName. The element also admits lax wildcard attributes in urn:w.
func manyAttrSchema(n int) string {
	var b strings.Builder
	b.WriteString(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:w="urn:w">
  <xs:import namespace="urn:w"/>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="e" maxOccurs="unbounded">
          <xs:complexType>
`)
	for i := 1; i <= n; i++ {
		typ := "xs:int"
		val := "7"
		if i%2 == 0 {
			typ = "xs:QName"
			val = "w:v"
		}
		fmt.Fprintf(&b, `            <xs:attribute name="a%02d" type="%s"`, i, typ)
		switch {
		case i%13 == 0:
			b.WriteString(` use="prohibited"`)
		case i%5 == 0:
			b.WriteString(` use="required"`)
		case i%7 == 0:
			fmt.Fprintf(&b, ` default="%s"`, val)
		case i%11 == 0:
			fmt.Fprintf(&b, ` fixed="%s"`, val)
		}
		b.WriteString("/>\n")
	}
	b.WriteString(`            <xs:anyAttribute namespace="urn:w" processContents="lax"/>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`)
	return b.String()
}

const attrQNameSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:p="urn:a">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="e" maxOccurs="unbounded">
          <xs:complexType>
            <xs:attribute name="q" type="xs:QName"/>
            <xs:attribute name="qs">
              <xs:simpleType><xs:list itemType="xs:QName"/></xs:simpleType>
            </xs:attribute>
            <xs:attribute name="u">
              <xs:simpleType><xs:union memberTypes="xs:int xs:QName"/></xs:simpleType>
            </xs:attribute>
            <xs:attribute name="f" type="xs:QName" fixed="p:fixed"/>
            <xs:attribute name="s" type="xs:string"/>
            <xs:attribute name="i" type="xs:int"/>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

const attrQNameInstance = `<root xmlns:a="urn:a">
  <e q="a:x" qs="a:x b:y" u="a:z" f="a:fixed" s="plain" i="1" xmlns:b="urn:b"/>
  <e q="c:x" i="x"/>
  <e qs="a:x c:y"/>
  <e u="c:z"/>
  <e f="b:fixed" xmlns:b="urn:b"/>
  <e f="x:fixed" xmlns:x="urn:a"/>
  <e xmlns:a="urn:other" q="a:x" f="a:fixed"/>
</root>`

const attrNotationSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:n="urn:n" targetNamespace="urn:n">
  <xs:notation name="gif" public="image/gif"/>
  <xs:simpleType name="fmt">
    <xs:restriction base="xs:NOTATION"><xs:enumeration value="n:gif"/></xs:restriction>
  </xs:simpleType>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="e" form="unqualified" maxOccurs="unbounded">
          <xs:complexType><xs:attribute name="fmt" type="n:fmt"/></xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

const attrNotationInstance = `<n:root xmlns:n="urn:n" xmlns:m="urn:n">
  <e fmt="m:gif"/>
  <e fmt="k:gif" xmlns:k="urn:n"/>
  <e fmt="k:gif" xmlns:k="urn:other"/>
  <e fmt="z:gif"/>
</n:root>`

const attrAssertionSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="aQName">
    <xs:restriction base="xs:QName">
      <xs:assertion test="namespace-uri-from-QName($value) = 'urn:a'"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:simpleType name="short">
    <xs:restriction base="xs:string">
      <xs:assertion test="string-length($value) lt 5"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="e" maxOccurs="unbounded">
          <xs:complexType>
            <xs:attribute name="q" type="aQName"/>
            <xs:attribute name="s" type="short"/>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

const attrAssertionInstance = `<root xmlns:a="urn:a">
  <e q="a:x" s="abc"/>
  <e q="b:x" s="abcdef" xmlns:b="urn:b"/>
  <e q="b:x" xmlns:b="urn:a"/>
  <e q="c:x"/>
</root>`

const simpleContentQNameSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:p="urn:a">
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="q" type="xs:QName" maxOccurs="unbounded"/>
        <xs:element name="fq" type="xs:QName" fixed="p:v" maxOccurs="unbounded"/>
        <xs:element name="c" maxOccurs="unbounded">
          <xs:complexType>
            <xs:simpleContent>
              <xs:extension base="xs:QName">
                <xs:attribute name="i" type="xs:int"/>
              </xs:extension>
            </xs:simpleContent>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

const simpleContentQNameInstance = `<root xmlns:a="urn:a">
  <q>a:x</q>
  <q xmlns:b="urn:b">b:x</q>
  <q>c:x</q>
  <fq xmlns:z="urn:a">z:v</fq>
  <fq>a:w</fq>
  <fq/>
  <c i="1">a:x</c>
  <c i="x" xmlns:b="urn:b">b:x</c>
  <c>c:x</c>
</root>`

const attrWildcardSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:w="urn:w" xmlns:p="urn:a" targetNamespace="urn:w">
  <xs:attribute name="gq" type="xs:QName"/>
  <xs:attribute name="gi" type="xs:int"/>
  <xs:attribute name="gf" type="xs:QName" fixed="p:v"/>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="strict" form="unqualified" maxOccurs="unbounded">
          <xs:complexType><xs:anyAttribute namespace="##targetNamespace" processContents="strict"/></xs:complexType>
        </xs:element>
        <xs:element name="lax" form="unqualified" maxOccurs="unbounded">
          <xs:complexType><xs:anyAttribute namespace="##any" processContents="lax"/></xs:complexType>
        </xs:element>
        <xs:element name="skip" form="unqualified" maxOccurs="unbounded">
          <xs:complexType><xs:anyAttribute namespace="##any" processContents="skip"/></xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

const attrWildcardInstance = `<w:root xmlns:w="urn:w" xmlns:a="urn:a">
  <strict w:gq="a:x" w:gi="3" w:gf="a:v"/>
  <strict w:gq="c:x" w:gi="three"/>
  <strict w:gf="b:v" xmlns:b="urn:b"/>
  <strict w:gf="b:v" xmlns:b="urn:a" w:other="1"/>
  <lax w:gq="c:x" w:other="1" plain="p"/>
  <lax w:gq="a:x" xmlns:a="urn:x"/>
  <skip w:gq="c:x" w:gi="three"/>
</w:root>`

const attrUsesSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attributeGroup name="g">
    <xs:attribute name="shared" type="xs:string" use="prohibited"/>
    <xs:attribute name="gone" type="xs:string" use="prohibited"/>
  </xs:attributeGroup>
  <xs:complexType name="base">
    <xs:attribute name="req" type="xs:int" use="required"/>
    <xs:attribute name="def" type="xs:int" default="5"/>
    <xs:attribute name="fix" type="xs:int" fixed="9"/>
    <xs:attribute name="shared" type="xs:string"/>
    <xs:attributeGroup ref="g"/>
  </xs:complexType>
  <xs:complexType name="derived">
    <xs:complexContent>
      <xs:extension base="base">
        <xs:attribute name="extra" type="xs:boolean" use="required"/>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="e" type="base" nillable="true" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

const attrUsesInstance = `<root xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:noNamespaceSchemaLocation="s.xsd">
  <e req="1" def="2" fix="9" shared="s"/>
  <e req="1"/>
  <e def="x" fix="8"/>
  <e req="1" gone="g" unknown="u" xml:lang="en"/>
  <e req="1" xsi:nil="true"/>
  <e req="1" xsi:nil="false" xsi:type="derived" extra="true"/>
  <e req="1" xsi:type="derived"/>
  <e xsi:type="derived" extra="maybe" xsi:nil="bogus"/>
</root>`

const legacyGMonthSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="months">
    <xs:restriction>
      <xs:simpleType><xs:list itemType="xs:gMonth"/></xs:simpleType>
      <xs:maxLength value="3"/>
    </xs:restriction>
  </xs:simpleType>
  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="e" maxOccurs="unbounded">
          <xs:complexType>
            <xs:attribute name="l" type="months"/>
            <xs:attribute name="m" type="xs:gMonth"/>
            <xs:attribute name="l2" type="months"/>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

const legacyGMonthInstance = `<root>
  <e l="--05 --06" m="--05--" l2="--05--"/>
  <e m="--05--" l="--05--"/>
  <e l="--05-- --06--" m="--07"/>
</root>`

func manyAttrInstance() string {
	var b strings.Builder
	b.WriteString(`<root xmlns:w="urn:w" xmlns:q="urn:q">` + "\n")
	// Every non-prohibited use present with a valid value, except that the
	// QName uses bind their prefix on the element itself.
	b.WriteString("  <e")
	for i := 1; i <= 40; i++ {
		if i%13 == 0 {
			continue
		}
		if i%2 == 0 {
			fmt.Fprintf(&b, ` a%02d="x:v"`, i)
			continue
		}
		fmt.Fprintf(&b, ` a%02d="7"`, i)
	}
	b.WriteString(` xmlns:x="urn:w"/>` + "\n")
	// Only the required uses, some invalid, plus a prohibited use, a lax
	// wildcard attribute, an unknown attribute, and an unbound QName prefix.
	b.WriteString(`  <e a05="1" a10="q:v" a15="x" a20="u:v" a25="2" a26="7" w:any="1" zz="1"/>` + "\n")
	b.WriteString("</root>")
	return b.String()
}

func attrValidationCases() []attrValidationCase {
	var cases []attrValidationCase
	for _, v := range []struct {
		label   string
		version xsd.Version
	}{{"1.0", xsd.Version10}, {"1.1", xsd.Version11}} {
		cases = append(cases,
			attrValidationCase{name: "QName attributes " + v.label, version: v.version, schema: attrQNameSchema, instance: attrQNameInstance},
			attrValidationCase{name: "NOTATION attribute " + v.label, version: v.version, schema: attrNotationSchema, instance: attrNotationInstance},
			attrValidationCase{name: "QName simple content " + v.label, version: v.version, schema: simpleContentQNameSchema, instance: simpleContentQNameInstance},
			attrValidationCase{name: "wildcard attributes " + v.label, version: v.version, schema: attrWildcardSchema, instance: attrWildcardInstance},
			attrValidationCase{name: "attribute uses and xsi attributes " + v.label, version: v.version, schema: attrUsesSchema, instance: attrUsesInstance},
			attrValidationCase{name: "many attribute uses " + v.label, version: v.version, schema: manyAttrSchema(40), instance: manyAttrInstance()},
			attrValidationCase{name: "list then gMonth " + v.label, version: v.version, schema: legacyGMonthSchema, instance: legacyGMonthInstance},
		)
	}
	cases = append(cases, attrValidationCase{name: "assertion facets 1.1", version: xsd.Version11, schema: attrAssertionSchema, instance: attrAssertionInstance})
	for i := range cases {
		cases[i].want = attrValidationWant[cases[i].name]
	}
	return cases
}

var attrValidationWant = map[string]string{
	"QName attributes 1.0": `result: xsd: validation failed
test.xml:3: Schemas validity error : Element 'e', attribute 'q': The value 'c:x' is not valid for the type of attribute 'q'.
test.xml:3: Schemas validity error : Element 'e', attribute 'i': The value 'x' is not valid for the type of attribute 'i'.
test.xml:4: Schemas validity error : Element 'e', attribute 'qs': The value 'a:x c:y' is not valid for the type of attribute 'qs'.
test.xml:5: Schemas validity error : Element 'e', attribute 'u': The value 'c:z' is not valid for the type of attribute 'u'.
test.xml:6: Schemas validity error : Element 'e', attribute 'f': The value 'b:fixed' does not match the fixed value constraint 'p:fixed'.
test.xml:8: Schemas validity error : Element 'e', attribute 'f': The value 'a:fixed' does not match the fixed value constraint 'p:fixed'.
--- document
<root xmlns:a="urn:a">
  <e xmlns:b="urn:b" q="a:x" qs="a:x b:y" u="a:z" f="a:fixed" s="plain" i="1"/>
  <e q="c:x" i="x" f="p:fixed"/>
  <e qs="a:x c:y" f="p:fixed"/>
  <e u="c:z" f="p:fixed"/>
  <e xmlns:b="urn:b" f="b:fixed"/>
  <e xmlns:x="urn:a" f="x:fixed"/>
  <e xmlns:a="urn:other" q="a:x" f="a:fixed"/>
</root>
--- annotations
root = xs:anyType
e = xs:anyType
e/@{}q = xs:QName
e/@{}qs = xs:anyType
e/@{}u = xs:anyType
e/@{}f = xs:QName
e/@{}s = xs:string
e/@{}i = xs:int
e = xs:anyType
e/@{}q = xs:QName
e/@{}i = xs:int
e/@{}f = xs:QName
e = xs:anyType
e/@{}qs = xs:anyType
e/@{}f = xs:QName
e = xs:anyType
e/@{}u = xs:anyType
e/@{}f = xs:QName
e = xs:anyType
e/@{}f = xs:QName
e = xs:anyType
e/@{}f = xs:QName
e = xs:anyType
e/@{}q = xs:QName
e/@{}f = xs:QName
`,
	"NOTATION attribute 1.0": `result: xsd: validation failed
test.xml:4: Schemas validity error : Element 'e', attribute 'fmt': The value 'k:gif' is not valid for the type of attribute 'fmt'.
test.xml:5: Schemas validity error : Element 'e', attribute 'fmt': The value 'z:gif' is not valid for the type of attribute 'fmt'.
--- document
<n:root xmlns:n="urn:n" xmlns:m="urn:n">
  <e fmt="m:gif"/>
  <e xmlns:k="urn:n" fmt="k:gif"/>
  <e xmlns:k="urn:other" fmt="k:gif"/>
  <e fmt="z:gif"/>
</n:root>
--- annotations
root = xs:anyType
e = xs:anyType
e/@{}fmt = Q{urn:n}fmt
e = xs:anyType
e/@{}fmt = Q{urn:n}fmt
e = xs:anyType
e/@{}fmt = Q{urn:n}fmt
e = xs:anyType
e/@{}fmt = Q{urn:n}fmt
`,
	"QName simple content 1.0": `result: xsd: validation failed
test.xml:4: Schemas validity error : Element 'q': 'c:x' is not a valid value of the atomic type 'xs:QName'.
test.xml:6: Schemas validity error : Element 'fq': The element content 'a:w' does not match the fixed value constraint 'p:v'.
test.xml:7: Schemas validity error : Element 'fq': 'p:v' is not a valid value of the atomic type 'xs:QName'.
test.xml:9: Schemas validity error : Element 'c', attribute 'i': The value 'x' is not valid for the type of attribute 'i'.
test.xml:10: Schemas validity error : Element 'c': 'c:x' is not a valid value of the atomic type 'xs:QName'.
--- document
<root xmlns:a="urn:a">
  <q>a:x</q>
  <q xmlns:b="urn:b">b:x</q>
  <q>c:x</q>
  <fq xmlns:z="urn:a">z:v</fq>
  <fq>a:w</fq>
  <fq/>
  <c i="1">a:x</c>
  <c xmlns:b="urn:b" i="x">b:x</c>
  <c>c:x</c>
</root>
--- annotations
root = xs:anyType
q = xs:QName
q = xs:QName
q = xs:QName
fq = xs:QName
fq = xs:QName
fq = xs:QName
c = xs:QName
c/@{}i = xs:int
c = xs:QName
c/@{}i = xs:int
c = xs:QName
`,
	"wildcard attributes 1.0": `result: xsd: validation failed
test.xml:3: Schemas validity error : Element 'strict', attribute '{urn:w}gq': 'c:x' is not a valid value of the atomic type 'xs:QName'.
test.xml:3: Schemas validity error : Element 'strict', attribute '{urn:w}gi': 'three' is not a valid value of the atomic type 'xs:int'.
test.xml:4: Schemas validity error : Element 'strict', attribute '{urn:w}gf': The value 'b:v' does not match the fixed value constraint 'p:v'.
test.xml:5: Schemas validity error : Element 'strict', attribute '{urn:w}other': No matching global attribute declaration available, but demanded by the strict wildcard.
test.xml:6: Schemas validity error : Element 'lax', attribute '{urn:w}gq': 'c:x' is not a valid value of the atomic type 'xs:QName'.
--- document
<w:root xmlns:w="urn:w" xmlns:a="urn:a">
  <strict w:gq="a:x" w:gi="3" w:gf="a:v"/>
  <strict w:gq="c:x" w:gi="three"/>
  <strict xmlns:b="urn:b" w:gf="b:v"/>
  <strict xmlns:b="urn:a" w:gf="b:v" w:other="1"/>
  <lax w:gq="c:x" w:other="1" plain="p"/>
  <lax xmlns:a="urn:x" w:gq="a:x"/>
  <skip w:gq="c:x" w:gi="three"/>
</w:root>
--- annotations
root = xs:anyType
strict = xs:anyType
strict/@{urn:w}gq = xs:QName
strict/@{urn:w}gi = xs:int
strict/@{urn:w}gf = xs:QName
strict = xs:anyType
strict/@{urn:w}gq = 
strict/@{urn:w}gi = 
strict = xs:anyType
strict/@{urn:w}gf = 
strict = xs:anyType
strict/@{urn:w}gf = xs:QName
strict/@{urn:w}other = 
lax = xs:anyType
lax/@{urn:w}gq = 
lax/@{urn:w}other = 
lax/@{}plain = 
lax = xs:anyType
lax/@{urn:w}gq = xs:QName
skip = xs:anyType
skip/@{urn:w}gq = 
skip/@{urn:w}gi = 
`,
	"attribute uses and xsi attributes 1.0": `test.xsd:3: element attribute: Schemas parser warning : Element '{http://www.w3.org/2001/XMLSchema}attribute': Skipping attribute use prohibition, since it is pointless inside an <attributeGroup>.
test.xsd:4: element attribute: Schemas parser warning : Element '{http://www.w3.org/2001/XMLSchema}attribute': Skipping attribute use prohibition, since it is pointless inside an <attributeGroup>.
result: xsd: validation failed
test.xml:4: Schemas validity error : Element 'e', attribute 'def': The value 'x' is not valid for the type of attribute 'def'.
test.xml:4: Schemas validity error : Element 'e', attribute 'fix': The value '8' does not match the fixed value constraint '9'.
test.xml:4: Schemas validity error : Element 'e': The attribute 'req' is required but missing.
test.xml:5: Schemas validity error : Element 'e', attribute 'gone': The attribute 'gone' is not allowed.
test.xml:5: Schemas validity error : Element 'e', attribute 'unknown': The attribute 'unknown' is not allowed.
test.xml:8: Schemas validity error : Element 'e': The attribute 'extra' is required but missing.
test.xml:9: Schemas validity error : Element 'e', attribute '{http://www.w3.org/2001/XMLSchema-instance}nil': 'bogus' is not a valid value of the atomic type 'xs:boolean'.
--- document
<root xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:noNamespaceSchemaLocation="s.xsd">
  <e req="1" def="2" fix="9" shared="s"/>
  <e req="1" def="5" fix="9"/>
  <e def="x" fix="8"/>
  <e req="1" gone="g" unknown="u" xml:lang="en" def="5" fix="9"/>
  <e req="1" xsi:nil="true" def="5" fix="9"/>
  <e req="1" xsi:nil="false" xsi:type="derived" extra="true" def="5" fix="9"/>
  <e req="1" xsi:type="derived" def="5" fix="9"/>
  <e xsi:type="derived" extra="maybe" xsi:nil="bogus"/>
</root>
--- annotations
root = xs:anyType
root/@{http://www.w3.org/2001/XMLSchema-instance}noNamespaceSchemaLocation = 
e = Q{}base
e/@{}req = xs:int
e/@{}def = xs:int
e/@{}fix = xs:int
e/@{}shared = xs:string
e = Q{}base
e/@{}req = xs:int
e/@{}def = xs:int
e/@{}fix = xs:int
e = Q{}base
e/@{}def = xs:int
e/@{}fix = xs:int
e = Q{}base
e/@{}req = xs:int
e/@{}gone = 
e/@{}unknown = 
e/@{http://www.w3.org/XML/1998/namespace}lang = 
e/@{}def = xs:int
e/@{}fix = xs:int
e = Q{}base
e/@{}req = xs:int
e/@{http://www.w3.org/2001/XMLSchema-instance}nil = 
e/@{}def = xs:int
e/@{}fix = xs:int
e = Q{}derived
e/@{}req = xs:int
e/@{http://www.w3.org/2001/XMLSchema-instance}nil = 
e/@{http://www.w3.org/2001/XMLSchema-instance}type = 
e/@{}extra = xs:boolean
e/@{}def = xs:int
e/@{}fix = xs:int
e = Q{}derived
e/@{}req = xs:int
e/@{http://www.w3.org/2001/XMLSchema-instance}type = 
e/@{}def = xs:int
e/@{}fix = xs:int
e = Q{}derived
e/@{http://www.w3.org/2001/XMLSchema-instance}type = 
e/@{}extra = 
e/@{http://www.w3.org/2001/XMLSchema-instance}nil = 
`,
	"many attribute uses 1.0": `result: xsd: validation failed
test.xml:3: Schemas validity error : Element 'e', attribute 'a15': The value 'x' is not valid for the type of attribute 'a15'.
test.xml:3: Schemas validity error : Element 'e', attribute 'a20': The value 'u:v' is not valid for the type of attribute 'a20'.
test.xml:3: Schemas validity error : Element 'e', attribute 'a26': The attribute 'a26' is not allowed.
test.xml:3: Schemas validity error : Element 'e', attribute 'zz': The attribute 'zz' is not allowed.
test.xml:3: Schemas validity error : Element 'e': The attribute 'a30' is required but missing.
test.xml:3: Schemas validity error : Element 'e': The attribute 'a35' is required but missing.
test.xml:3: Schemas validity error : Element 'e': The attribute 'a40' is required but missing.
--- document
<root xmlns:w="urn:w" xmlns:q="urn:q">
  <e xmlns:x="urn:w" a01="7" a02="x:v" a03="7" a04="x:v" a05="7" a06="x:v" a07="7" a08="x:v" a09="7" a10="x:v" a11="7" a12="x:v" a14="x:v" a15="7" a16="x:v" a17="7" a18="x:v" a19="7" a20="x:v" a21="7" a22="x:v" a23="7" a24="x:v" a25="7" a27="7" a28="x:v" a29="7" a30="x:v" a31="7" a32="x:v" a33="7" a34="x:v" a35="7" a36="x:v" a37="7" a38="x:v" a40="x:v"/>
  <e a05="1" a10="q:v" a15="x" a20="u:v" a25="2" a26="7" w:any="1" zz="1" a07="7" a11="7" a14="w:v" a21="7" a22="w:v" a28="w:v" a33="7"/>
</root>
--- annotations
root = xs:anyType
e = xs:anyType
e/@{}a01 = xs:int
e/@{}a02 = xs:QName
e/@{}a03 = xs:int
e/@{}a04 = xs:QName
e/@{}a05 = xs:int
e/@{}a06 = xs:QName
e/@{}a07 = xs:int
e/@{}a08 = xs:QName
e/@{}a09 = xs:int
e/@{}a10 = xs:QName
e/@{}a11 = xs:int
e/@{}a12 = xs:QName
e/@{}a14 = xs:QName
e/@{}a15 = xs:int
e/@{}a16 = xs:QName
e/@{}a17 = xs:int
e/@{}a18 = xs:QName
e/@{}a19 = xs:int
e/@{}a20 = xs:QName
e/@{}a21 = xs:int
e/@{}a22 = xs:QName
e/@{}a23 = xs:int
e/@{}a24 = xs:QName
e/@{}a25 = xs:int
e/@{}a27 = xs:int
e/@{}a28 = xs:QName
e/@{}a29 = xs:int
e/@{}a30 = xs:QName
e/@{}a31 = xs:int
e/@{}a32 = xs:QName
e/@{}a33 = xs:int
e/@{}a34 = xs:QName
e/@{}a35 = xs:int
e/@{}a36 = xs:QName
e/@{}a37 = xs:int
e/@{}a38 = xs:QName
e/@{}a40 = xs:QName
e = xs:anyType
e/@{}a05 = xs:int
e/@{}a10 = xs:QName
e/@{}a15 = xs:int
e/@{}a20 = xs:QName
e/@{}a25 = xs:int
e/@{}a26 = 
e/@{urn:w}any = 
e/@{}zz = 
e/@{}a07 = xs:int
e/@{}a11 = xs:int
e/@{}a14 = xs:QName
e/@{}a21 = xs:int
e/@{}a22 = xs:QName
e/@{}a28 = xs:QName
e/@{}a33 = xs:int
`,
	"list then gMonth 1.0": `result: xsd: validation failed
test.xml:2: Schemas validity error : Element 'e', attribute 'l2': The value '--05--' is not valid for the type of attribute 'l2'.
test.xml:3: Schemas validity error : Element 'e', attribute 'l': The value '--05--' is not valid for the type of attribute 'l'.
test.xml:4: Schemas validity error : Element 'e', attribute 'l': The value '--05-- --06--' is not valid for the type of attribute 'l'.
--- document
<root>
  <e l="--05 --06" m="--05--" l2="--05--"/>
  <e m="--05--" l="--05--"/>
  <e l="--05-- --06--" m="--07"/>
</root>
--- annotations
root = xs:anyType
e = xs:anyType
e/@{}l = Q{}months
e/@{}m = xs:gMonth
e/@{}l2 = Q{}months
e = xs:anyType
e/@{}m = xs:gMonth
e/@{}l = Q{}months
e = xs:anyType
e/@{}l = Q{}months
e/@{}m = xs:gMonth
`,
	"QName attributes 1.1": `result: xsd: validation failed
test.xml:3: Schemas validity error : Element 'e', attribute 'q': The value 'c:x' is not valid for the type of attribute 'q'.
test.xml:3: Schemas validity error : Element 'e', attribute 'i': The value 'x' is not valid for the type of attribute 'i'.
test.xml:4: Schemas validity error : Element 'e', attribute 'qs': The value 'a:x c:y' is not valid for the type of attribute 'qs'.
test.xml:5: Schemas validity error : Element 'e', attribute 'u': The value 'c:z' is not valid for the type of attribute 'u'.
test.xml:6: Schemas validity error : Element 'e', attribute 'f': The value 'b:fixed' does not match the fixed value constraint 'p:fixed'.
test.xml:8: Schemas validity error : Element 'e', attribute 'f': The value 'a:fixed' does not match the fixed value constraint 'p:fixed'.
--- document
<root xmlns:a="urn:a">
  <e xmlns:b="urn:b" q="a:x" qs="a:x b:y" u="a:z" f="a:fixed" s="plain" i="1"/>
  <e xmlns:p="urn:a" q="c:x" i="x" f="p:fixed"/>
  <e xmlns:p="urn:a" qs="a:x c:y" f="p:fixed"/>
  <e xmlns:p="urn:a" u="c:z" f="p:fixed"/>
  <e xmlns:b="urn:b" f="b:fixed"/>
  <e xmlns:x="urn:a" f="x:fixed"/>
  <e xmlns:a="urn:other" q="a:x" f="a:fixed"/>
</root>
--- annotations
root = xs:anyType
e = xs:anyType
e/@{}q = xs:QName
e/@{}qs = xs:anyType
e/@{}u = xs:anyType
e/@{}f = xs:QName
e/@{}s = xs:string
e/@{}i = xs:int
e = xs:anyType
e/@{}q = xs:QName
e/@{}i = xs:int
e/@{}f = xs:QName
e = xs:anyType
e/@{}qs = xs:anyType
e/@{}f = xs:QName
e = xs:anyType
e/@{}u = xs:anyType
e/@{}f = xs:QName
e = xs:anyType
e/@{}f = xs:QName
e = xs:anyType
e/@{}f = xs:QName
e = xs:anyType
e/@{}q = xs:QName
e/@{}f = xs:QName
`,
	"NOTATION attribute 1.1": `result: xsd: validation failed
test.xml:4: Schemas validity error : Element 'e', attribute 'fmt': The value 'k:gif' is not valid for the type of attribute 'fmt'.
test.xml:5: Schemas validity error : Element 'e', attribute 'fmt': The value 'z:gif' is not valid for the type of attribute 'fmt'.
--- document
<n:root xmlns:n="urn:n" xmlns:m="urn:n">
  <e fmt="m:gif"/>
  <e xmlns:k="urn:n" fmt="k:gif"/>
  <e xmlns:k="urn:other" fmt="k:gif"/>
  <e fmt="z:gif"/>
</n:root>
--- annotations
root = xs:anyType
e = xs:anyType
e/@{}fmt = Q{urn:n}fmt
e = xs:anyType
e/@{}fmt = Q{urn:n}fmt
e = xs:anyType
e/@{}fmt = Q{urn:n}fmt
e = xs:anyType
e/@{}fmt = Q{urn:n}fmt
`,
	"QName simple content 1.1": `result: xsd: validation failed
test.xml:4: Schemas validity error : Element 'q': 'c:x' is not a valid value of the atomic type 'xs:QName'.
test.xml:6: Schemas validity error : Element 'fq': The element content 'a:w' does not match the fixed value constraint 'p:v'.
test.xml:9: Schemas validity error : Element 'c', attribute 'i': The value 'x' is not valid for the type of attribute 'i'.
test.xml:10: Schemas validity error : Element 'c': 'c:x' is not a valid value of the atomic type 'xs:QName'.
--- document
<root xmlns:a="urn:a">
  <q>a:x</q>
  <q xmlns:b="urn:b">b:x</q>
  <q>c:x</q>
  <fq xmlns:z="urn:a">z:v</fq>
  <fq>a:w</fq>
  <fq/>
  <c i="1">a:x</c>
  <c xmlns:b="urn:b" i="x">b:x</c>
  <c>c:x</c>
</root>
--- annotations
root = xs:anyType
q = xs:QName
q = xs:QName
q = xs:QName
fq = xs:QName
fq = xs:QName
fq = xs:QName
c = xs:QName
c/@{}i = xs:int
c = xs:QName
c/@{}i = xs:int
c = xs:QName
`,
	"wildcard attributes 1.1": `result: xsd: validation failed
test.xml:3: Schemas validity error : Element 'strict', attribute '{urn:w}gq': 'c:x' is not a valid value of the atomic type 'xs:QName'.
test.xml:3: Schemas validity error : Element 'strict', attribute '{urn:w}gi': 'three' is not a valid value of the atomic type 'xs:int'.
test.xml:4: Schemas validity error : Element 'strict', attribute '{urn:w}gf': The value 'b:v' does not match the fixed value constraint 'p:v'.
test.xml:5: Schemas validity error : Element 'strict', attribute '{urn:w}other': No matching global attribute declaration available, but demanded by the strict wildcard.
test.xml:6: Schemas validity error : Element 'lax', attribute '{urn:w}gq': 'c:x' is not a valid value of the atomic type 'xs:QName'.
--- document
<w:root xmlns:w="urn:w" xmlns:a="urn:a">
  <strict w:gq="a:x" w:gi="3" w:gf="a:v"/>
  <strict w:gq="c:x" w:gi="three"/>
  <strict xmlns:b="urn:b" w:gf="b:v"/>
  <strict xmlns:b="urn:a" w:gf="b:v" w:other="1"/>
  <lax w:gq="c:x" w:other="1" plain="p"/>
  <lax xmlns:a="urn:x" w:gq="a:x"/>
  <skip w:gq="c:x" w:gi="three"/>
</w:root>
--- annotations
root = xs:anyType
strict = xs:anyType
strict/@{urn:w}gq = xs:QName
strict/@{urn:w}gi = xs:int
strict/@{urn:w}gf = xs:QName
strict = xs:anyType
strict/@{urn:w}gq = 
strict/@{urn:w}gi = 
strict = xs:anyType
strict/@{urn:w}gf = 
strict = xs:anyType
strict/@{urn:w}gf = xs:QName
strict/@{urn:w}other = 
lax = xs:anyType
lax/@{urn:w}gq = 
lax/@{urn:w}other = 
lax/@{}plain = 
lax = xs:anyType
lax/@{urn:w}gq = xs:QName
skip = xs:anyType
skip/@{urn:w}gq = 
skip/@{urn:w}gi = 
`,
	"attribute uses and xsi attributes 1.1": `test.xsd:3: element attribute: Schemas parser warning : Element '{http://www.w3.org/2001/XMLSchema}attribute': Skipping attribute use prohibition, since it is pointless inside an <attributeGroup>.
test.xsd:4: element attribute: Schemas parser warning : Element '{http://www.w3.org/2001/XMLSchema}attribute': Skipping attribute use prohibition, since it is pointless inside an <attributeGroup>.
result: xsd: validation failed
test.xml:4: Schemas validity error : Element 'e', attribute 'def': The value 'x' is not valid for the type of attribute 'def'.
test.xml:4: Schemas validity error : Element 'e', attribute 'fix': The value '8' does not match the fixed value constraint '9'.
test.xml:4: Schemas validity error : Element 'e': The attribute 'req' is required but missing.
test.xml:5: Schemas validity error : Element 'e', attribute 'gone': The attribute 'gone' is not allowed.
test.xml:5: Schemas validity error : Element 'e', attribute 'unknown': The attribute 'unknown' is not allowed.
test.xml:5: Schemas validity error : Element 'e', attribute '{http://www.w3.org/XML/1998/namespace}lang': The attribute '{http://www.w3.org/XML/1998/namespace}lang' is not allowed.
test.xml:8: Schemas validity error : Element 'e': The attribute 'extra' is required but missing.
test.xml:9: Schemas validity error : Element 'e', attribute '{http://www.w3.org/2001/XMLSchema-instance}nil': 'bogus' is not a valid value of the atomic type 'xs:boolean'.
--- document
<root xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:noNamespaceSchemaLocation="s.xsd">
  <e req="1" def="2" fix="9" shared="s"/>
  <e req="1" def="5" fix="9"/>
  <e def="x" fix="8"/>
  <e req="1" gone="g" unknown="u" xml:lang="en" def="5" fix="9"/>
  <e req="1" xsi:nil="true" def="5" fix="9"/>
  <e req="1" xsi:nil="false" xsi:type="derived" extra="true" def="5" fix="9"/>
  <e req="1" xsi:type="derived" def="5" fix="9"/>
  <e xsi:type="derived" extra="maybe" xsi:nil="bogus"/>
</root>
--- annotations
root = xs:anyType
root/@{http://www.w3.org/2001/XMLSchema-instance}noNamespaceSchemaLocation = 
e = Q{}base
e/@{}req = xs:int
e/@{}def = xs:int
e/@{}fix = xs:int
e/@{}shared = xs:string
e = Q{}base
e/@{}req = xs:int
e/@{}def = xs:int
e/@{}fix = xs:int
e = Q{}base
e/@{}def = xs:int
e/@{}fix = xs:int
e = Q{}base
e/@{}req = xs:int
e/@{}gone = 
e/@{}unknown = 
e/@{http://www.w3.org/XML/1998/namespace}lang = 
e/@{}def = xs:int
e/@{}fix = xs:int
e = Q{}base
e/@{}req = xs:int
e/@{http://www.w3.org/2001/XMLSchema-instance}nil = 
e/@{}def = xs:int
e/@{}fix = xs:int
e = Q{}derived
e/@{}req = xs:int
e/@{http://www.w3.org/2001/XMLSchema-instance}nil = 
e/@{http://www.w3.org/2001/XMLSchema-instance}type = 
e/@{}extra = xs:boolean
e/@{}def = xs:int
e/@{}fix = xs:int
e = Q{}derived
e/@{}req = xs:int
e/@{http://www.w3.org/2001/XMLSchema-instance}type = 
e/@{}def = xs:int
e/@{}fix = xs:int
e = Q{}derived
e/@{http://www.w3.org/2001/XMLSchema-instance}type = 
e/@{}extra = 
e/@{http://www.w3.org/2001/XMLSchema-instance}nil = 
`,
	"many attribute uses 1.1": `result: xsd: validation failed
test.xml:3: Schemas validity error : Element 'e', attribute 'a15': The value 'x' is not valid for the type of attribute 'a15'.
test.xml:3: Schemas validity error : Element 'e', attribute 'a20': The value 'u:v' is not valid for the type of attribute 'a20'.
test.xml:3: Schemas validity error : Element 'e', attribute 'a26': The attribute 'a26' is not allowed.
test.xml:3: Schemas validity error : Element 'e', attribute 'zz': The attribute 'zz' is not allowed.
test.xml:3: Schemas validity error : Element 'e': The attribute 'a30' is required but missing.
test.xml:3: Schemas validity error : Element 'e': The attribute 'a35' is required but missing.
test.xml:3: Schemas validity error : Element 'e': The attribute 'a40' is required but missing.
--- document
<root xmlns:w="urn:w" xmlns:q="urn:q">
  <e xmlns:x="urn:w" a01="7" a02="x:v" a03="7" a04="x:v" a05="7" a06="x:v" a07="7" a08="x:v" a09="7" a10="x:v" a11="7" a12="x:v" a14="x:v" a15="7" a16="x:v" a17="7" a18="x:v" a19="7" a20="x:v" a21="7" a22="x:v" a23="7" a24="x:v" a25="7" a27="7" a28="x:v" a29="7" a30="x:v" a31="7" a32="x:v" a33="7" a34="x:v" a35="7" a36="x:v" a37="7" a38="x:v" a40="x:v"/>
  <e a05="1" a10="q:v" a15="x" a20="u:v" a25="2" a26="7" w:any="1" zz="1" a07="7" a11="7" a14="w:v" a21="7" a22="w:v" a28="w:v" a33="7"/>
</root>
--- annotations
root = xs:anyType
e = xs:anyType
e/@{}a01 = xs:int
e/@{}a02 = xs:QName
e/@{}a03 = xs:int
e/@{}a04 = xs:QName
e/@{}a05 = xs:int
e/@{}a06 = xs:QName
e/@{}a07 = xs:int
e/@{}a08 = xs:QName
e/@{}a09 = xs:int
e/@{}a10 = xs:QName
e/@{}a11 = xs:int
e/@{}a12 = xs:QName
e/@{}a14 = xs:QName
e/@{}a15 = xs:int
e/@{}a16 = xs:QName
e/@{}a17 = xs:int
e/@{}a18 = xs:QName
e/@{}a19 = xs:int
e/@{}a20 = xs:QName
e/@{}a21 = xs:int
e/@{}a22 = xs:QName
e/@{}a23 = xs:int
e/@{}a24 = xs:QName
e/@{}a25 = xs:int
e/@{}a27 = xs:int
e/@{}a28 = xs:QName
e/@{}a29 = xs:int
e/@{}a30 = xs:QName
e/@{}a31 = xs:int
e/@{}a32 = xs:QName
e/@{}a33 = xs:int
e/@{}a34 = xs:QName
e/@{}a35 = xs:int
e/@{}a36 = xs:QName
e/@{}a37 = xs:int
e/@{}a38 = xs:QName
e/@{}a40 = xs:QName
e = xs:anyType
e/@{}a05 = xs:int
e/@{}a10 = xs:QName
e/@{}a15 = xs:int
e/@{}a20 = xs:QName
e/@{}a25 = xs:int
e/@{}a26 = 
e/@{urn:w}any = 
e/@{}zz = 
e/@{}a07 = xs:int
e/@{}a11 = xs:int
e/@{}a14 = xs:QName
e/@{}a21 = xs:int
e/@{}a22 = xs:QName
e/@{}a28 = xs:QName
e/@{}a33 = xs:int
`,
	"list then gMonth 1.1": `result: xsd: validation failed
test.xml:2: Schemas validity error : Element 'e', attribute 'm': The value '--05--' is not valid for the type of attribute 'm'.
test.xml:2: Schemas validity error : Element 'e', attribute 'l2': The value '--05--' is not valid for the type of attribute 'l2'.
test.xml:3: Schemas validity error : Element 'e', attribute 'm': The value '--05--' is not valid for the type of attribute 'm'.
test.xml:3: Schemas validity error : Element 'e', attribute 'l': The value '--05--' is not valid for the type of attribute 'l'.
test.xml:4: Schemas validity error : Element 'e', attribute 'l': The value '--05-- --06--' is not valid for the type of attribute 'l'.
--- document
<root>
  <e l="--05 --06" m="--05--" l2="--05--"/>
  <e m="--05--" l="--05--"/>
  <e l="--05-- --06--" m="--07"/>
</root>
--- annotations
root = xs:anyType
e = xs:anyType
e/@{}l = Q{}months
e/@{}m = xs:gMonth
e/@{}l2 = Q{}months
e = xs:anyType
e/@{}m = xs:gMonth
e/@{}l = Q{}months
e = xs:anyType
e/@{}l = Q{}months
e/@{}m = xs:gMonth
`,
	"assertion facets 1.1": `result: xsd: validation failed
test.xml:3: Schemas validity error : Element 'e', attribute 'q': The value 'b:x' is not valid for the type of attribute 'q'.
test.xml:3: Schemas validity error : Element 'e', attribute 's': The value 'abcdef' is not valid for the type of attribute 's'.
test.xml:5: Schemas validity error : Element 'e', attribute 'q': The value 'c:x' is not valid for the type of attribute 'q'.
--- document
<root xmlns:a="urn:a">
  <e q="a:x" s="abc"/>
  <e xmlns:b="urn:b" q="b:x" s="abcdef"/>
  <e xmlns:b="urn:a" q="b:x"/>
  <e q="c:x"/>
</root>
--- annotations
root = xs:anyType
e = xs:anyType
e/@{}q = Q{}aQName
e/@{}s = Q{}short
e = xs:anyType
e/@{}q = Q{}aQName
e/@{}s = Q{}short
e = xs:anyType
e/@{}q = Q{}aQName
e = xs:anyType
e/@{}q = Q{}aQName
`,
}
