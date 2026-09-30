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

// entityRefTextSchema declares one global element per way the validator reads
// element or attribute text: simple content (with facets, fixed and default
// values), attribute values, element-only, empty and nilled content, xs:ID
// content, and xs:key/xs:keyref fields. The 1.1 variant adds xs:assert on
// simple and complex content and an xs:alternative whose test reads an
// attribute.
func entityRefTextSchema(v11 bool) string {
	var b strings.Builder
	b.WriteString(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="int" type="xs:int"/>
  <xs:element name="short">
    <xs:simpleType>
      <xs:restriction base="xs:string"><xs:maxLength value="3"/></xs:restriction>
    </xs:simpleType>
  </xs:element>
  <xs:element name="fixed" type="xs:string" fixed="123"/>
  <xs:element name="dflt" type="xs:int" default="5"/>
  <xs:element name="attr">
    <xs:complexType><xs:attribute name="n" type="xs:int"/></xs:complexType>
  </xs:element>
  <xs:element name="eonly">
    <xs:complexType><xs:sequence><xs:element name="k" type="xs:int"/></xs:sequence></xs:complexType>
  </xs:element>
  <xs:element name="empty"><xs:complexType/></xs:element>
  <xs:element name="nil" type="xs:int" nillable="true"/>
  <xs:element name="ids">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="w" maxOccurs="unbounded">
          <xs:complexType><xs:sequence><xs:element name="id" type="xs:ID"/></xs:sequence></xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
  <xs:element name="keys">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="k" type="xs:string" minOccurs="0" maxOccurs="unbounded"/>
        <xs:element name="ref" type="xs:string" minOccurs="0" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
    <xs:key name="kk"><xs:selector xpath="k"/><xs:field xpath="."/></xs:key>
    <xs:keyref name="kr" refer="kk"><xs:selector xpath="ref"/><xs:field xpath="."/></xs:keyref>
  </xs:element>
`)
	if v11 {
		b.WriteString(`  <xs:element name="big">
    <xs:complexType>
      <xs:simpleContent>
        <xs:extension base="xs:int"><xs:assert test="$value gt 10"/></xs:extension>
      </xs:simpleContent>
    </xs:complexType>
  </xs:element>
  <xs:element name="cmp">
    <xs:complexType>
      <xs:sequence><xs:element name="v" type="xs:string"/></xs:sequence>
      <xs:assert test="v = 'ab'"/>
    </xs:complexType>
  </xs:element>
  <xs:element name="acmp">
    <xs:complexType>
      <xs:attribute name="n" type="xs:int"/>
      <xs:assert test="@n = 42"/>
    </xs:complexType>
  </xs:element>
  <xs:element name="alt" type="T">
    <xs:alternative test="@t = 'int'" type="TInt"/>
  </xs:element>
  <xs:complexType name="T">
    <xs:simpleContent>
      <xs:extension base="xs:string"><xs:attribute name="t" type="xs:string"/></xs:extension>
    </xs:simpleContent>
  </xs:complexType>
  <xs:complexType name="TInt">
    <xs:simpleContent>
      <xs:restriction base="T"><xs:pattern value="[0-9]+"/></xs:restriction>
    </xs:simpleContent>
  </xs:complexType>
`)
	}
	b.WriteString(`</xs:schema>`)
	return b.String()
}

type entityRefTextCase struct {
	name     string
	entities string // internal DTD subset
	body     string // document element
	valid    bool
	v11Only  bool
}

func entityRefTextCases() []entityRefTextCase {
	const xsi = `xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"`
	return []entityRefTextCase{
		{name: "int valid", entities: `<!ENTITY c "42">`, body: `<int>&c;</int>`, valid: true},
		{name: "int invalid", entities: `<!ENTITY c "4x">`, body: `<int>&c;</int>`},
		{name: "int nested valid", entities: `<!ENTITY a "4"><!ENTITY b "&a;2">`, body: `<int>&b;</int>`, valid: true},
		{name: "int nested invalid", entities: `<!ENTITY a "x"><!ENTITY b "&a;2">`, body: `<int>&b;</int>`},
		{name: "int mixed with literal text", entities: `<!ENTITY c "2">`, body: `<int>1&c;3</int>`, valid: true},
		{name: "int entity holding a comment", entities: `<!ENTITY c "4<!--x-->2">`, body: `<int>&c;</int>`, valid: true},
		{name: "maxLength exceeded through entity", entities: `<!ENTITY c "cd">`, body: `<short>ab&c;</short>`},
		{name: "fixed matched through entity", entities: `<!ENTITY c "2">`, body: `<fixed>1&c;3</fixed>`, valid: true},
		{name: "fixed mismatched through entity", entities: `<!ENTITY c "9">`, body: `<fixed>1&c;3</fixed>`},
		{name: "empty entity takes default", entities: `<!ENTITY e "">`, body: `<dflt>&e;</dflt>`, valid: true},
		{name: "attribute valid", entities: `<!ENTITY c "42">`, body: `<attr n="&c;"/>`, valid: true},
		{name: "attribute nested mixed valid", entities: `<!ENTITY a "2"><!ENTITY b "&a;3">`, body: `<attr n="1&b;"/>`, valid: true},
		{name: "attribute invalid", entities: `<!ENTITY c "x">`, body: `<attr n="&c;"/>`},
		{name: "element-only text through entity", entities: `<!ENTITY c "x">`, body: `<eonly>&c;<k>1</k></eonly>`},
		{name: "element-only whitespace through entity", entities: `<!ENTITY sp " ">`, body: `<eonly>&sp;<k>1</k></eonly>`, valid: true},
		{name: "empty content text through entity", entities: `<!ENTITY c "x">`, body: `<empty>&c;</empty>`},
		{name: "nilled text through entity", entities: `<!ENTITY c "1">`, body: `<nil ` + xsi + ` xsi:nil="true">&c;</nil>`},
		{name: "distinct IDs through entity", entities: `<!ENTITY c "b">`, body: `<ids><w><id>&c;</id></w><w><id>a</id></w></ids>`, valid: true},
		{name: "duplicate ID through entity", entities: `<!ENTITY c "a">`, body: `<ids><w><id>&c;</id></w><w><id>a</id></w></ids>`},
		{name: "distinct keys through entities", entities: `<!ENTITY c "1"><!ENTITY d "2">`, body: `<keys><k>&c;</k><k>&d;</k></keys>`, valid: true},
		{name: "duplicate key through entity", entities: `<!ENTITY c "2">`, body: `<keys><k>&c;</k><k>2</k></keys>`},
		{name: "keyref resolved through entity", entities: `<!ENTITY c "2">`, body: `<keys><k>1</k><k>2</k><ref>&c;</ref></keys>`, valid: true},
		{name: "keyref dangling through entity", entities: `<!ENTITY c "3">`, body: `<keys><k>1</k><k>2</k><ref>&c;</ref></keys>`},
		{name: "simple assert valid", entities: `<!ENTITY c "42">`, body: `<big>&c;</big>`, valid: true, v11Only: true},
		{name: "simple assert invalid", entities: `<!ENTITY c "5">`, body: `<big>&c;</big>`, v11Only: true},
		{name: "complex assert valid", entities: `<!ENTITY c "b">`, body: `<cmp><v>a&c;</v></cmp>`, valid: true, v11Only: true},
		{name: "complex assert invalid", entities: `<!ENTITY c "c">`, body: `<cmp><v>a&c;</v></cmp>`, v11Only: true},
		{name: "complex assert on entity-only text", entities: `<!ENTITY c "b">`, body: `<cmp><v>&c;</v></cmp>`, v11Only: true},
		{name: "attribute assert valid", entities: `<!ENTITY a "4"><!ENTITY c "&a;2">`, body: `<acmp n="&c;"/>`, valid: true, v11Only: true},
		{name: "attribute assert invalid", entities: `<!ENTITY c "41">`, body: `<acmp n="&c;"/>`, v11Only: true},
		{name: "alternative selected through entity", entities: `<!ENTITY c "int">`, body: `<alt t="&c;">ab</alt>`, v11Only: true},
		{name: "alternative content through entity", entities: `<!ENTITY c "int"><!ENTITY v "12">`, body: `<alt t="&c;">&v;</alt>`, valid: true, v11Only: true},
	}
}

// TestValidateEntityRefText verifies that element and attribute text reached
// through entity references is validated as its expansion. A document parsed
// without entity substitution (the default) keeps EntityRef nodes in the tree;
// its validation result and diagnostics must equal those of the same document
// parsed with SubstituteEntities(true), in XSD 1.0 and 1.1.
func TestValidateEntityRefText(t *testing.T) {
	t.Parallel()

	for _, version := range []xsd.Version{xsd.Version10, xsd.Version11} {
		schemaDoc, err := helium.NewParser().Parse(t.Context(), []byte(entityRefTextSchema(version == xsd.Version11)))
		require.NoError(t, err)
		schema, err := xsd.NewCompiler().Version(version).Compile(t.Context(), schemaDoc)
		require.NoError(t, err)

		for _, tc := range entityRefTextCases() {
			if tc.v11Only && version != xsd.Version11 {
				continue
			}
			t.Run(version.String()+"/"+tc.name, func(t *testing.T) {
				t.Parallel()
				instance := "<!DOCTYPE r [" + tc.entities + "]>\n" + tc.body

				refDoc, err := helium.NewParser().Parse(t.Context(), []byte(instance))
				require.NoError(t, err)
				var buf bytes.Buffer
				require.NoError(t, helium.Write(&buf, refDoc.DocumentElement()))
				require.Contains(t, buf.String(), "&", "the default parse keeps entity references")

				substDoc, err := helium.NewParser().SubstituteEntities(true).Parse(t.Context(), []byte(instance))
				require.NoError(t, err)

				var substOut, refOut string
				substErr := validateWithOutput(t, xsd.NewValidator(schema).Label("test.xml"), substDoc, &substOut)
				refErr := validateWithOutput(t, xsd.NewValidator(schema).Label("test.xml"), refDoc, &refOut)

				require.Equal(t, tc.valid, substErr == nil, "substituted parse: %s", substOut)
				require.Equal(t, substOut, refOut)
				require.Equal(t, substErr == nil, refErr == nil)
			})
		}
	}
}

// entityBorneSchema declares one global element per check that reads an
// element's element children: element-only, mixed and xs:all content models,
// simple, empty and nilled content, xs:anyType content, strict and lax
// wildcards, xs:ID/xs:IDREF content, and xs:key/xs:keyref on a host reached
// through an entity or holding one. The 1.1 variant adds xs:assert over child
// elements.
func entityBorneSchema(v11 bool) string {
	var b strings.Builder
	b.WriteString(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="seq">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="a" type="xs:int"/>
        <xs:element name="b" type="xs:int" minOccurs="0" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
  <xs:element name="mix">
    <xs:complexType mixed="true">
      <xs:sequence>
        <xs:element name="a" type="xs:int"/>
        <xs:element name="b" type="xs:int" minOccurs="0"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
  <xs:element name="all">
    <xs:complexType>
      <xs:all><xs:element name="a" type="xs:int"/><xs:element name="b" type="xs:int"/></xs:all>
    </xs:complexType>
  </xs:element>
  <xs:element name="int" type="xs:int"/>
  <xs:element name="empty"><xs:complexType/></xs:element>
  <xs:element name="nil" nillable="true">
    <xs:complexType>
      <xs:sequence><xs:element name="a" type="xs:int" minOccurs="0"/></xs:sequence>
    </xs:complexType>
  </xs:element>
  <xs:element name="g" type="xs:int"/>
  <xs:element name="any" type="xs:anyType"/>
  <xs:element name="wild">
    <xs:complexType>
      <xs:sequence><xs:any processContents="strict" maxOccurs="unbounded"/></xs:sequence>
    </xs:complexType>
  </xs:element>
  <xs:element name="lax">
    <xs:complexType>
      <xs:sequence><xs:any processContents="lax" maxOccurs="unbounded"/></xs:sequence>
    </xs:complexType>
  </xs:element>
  <xs:element name="ids">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="w" maxOccurs="unbounded">
          <xs:complexType>
            <xs:sequence><xs:element name="id" type="xs:ID" maxOccurs="unbounded"/></xs:sequence>
            <xs:attribute name="ref" type="xs:IDREF"/>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
  <xs:element name="box">
    <xs:complexType>
      <xs:sequence><xs:element ref="keys" maxOccurs="unbounded"/></xs:sequence>
    </xs:complexType>
  </xs:element>
  <xs:element name="keys">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="k" type="xs:string" minOccurs="0" maxOccurs="unbounded"/>
        <xs:element name="ref" type="xs:string" minOccurs="0" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
    <xs:key name="kk"><xs:selector xpath="k"/><xs:field xpath="."/></xs:key>
    <xs:keyref name="kr" refer="kk"><xs:selector xpath="ref"/><xs:field xpath="."/></xs:keyref>
  </xs:element>
  <xs:element name="outer">
    <xs:complexType>
      <xs:sequence>
        <xs:element ref="keys" minOccurs="0" maxOccurs="unbounded"/>
        <xs:element name="use" type="xs:string" minOccurs="0" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
    <xs:keyref name="ou" refer="kk"><xs:selector xpath="use"/><xs:field xpath="."/></xs:keyref>
  </xs:element>
`)
	if v11 {
		b.WriteString(`  <xs:element name="cmp">
    <xs:complexType>
      <xs:sequence><xs:element name="v" type="xs:int" maxOccurs="unbounded"/></xs:sequence>
      <xs:assert test="count(v) eq 2 and data(v[1]) instance of xs:int and v[2] eq 2"/>
    </xs:complexType>
  </xs:element>
  <xs:element name="cmpm">
    <xs:complexType mixed="true">
      <xs:sequence><xs:element name="v" type="xs:int"/></xs:sequence>
      <xs:assert test="string(.) eq 'x1y' and count(text()) eq 2 and data(v) instance of xs:int"/>
    </xs:complexType>
  </xs:element>
`)
	}
	b.WriteString(`</xs:schema>`)
	return b.String()
}

type entityBorneCase struct {
	name     string
	entities string // internal DTD subset
	body     string // document element, starting on line 2
	valid    bool
	v10Only  bool
	v11Only  bool
}

func entityBorneCases() []entityBorneCase {
	const xsi = `xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"`
	return []entityBorneCase{
		{name: "element-only valid", entities: `<!ENTITY c "<a>1</a><b>2</b>">`, body: `<seq>&c;</seq>`, valid: true},
		{name: "element-only wrong order", entities: `<!ENTITY c "<b>2</b><a>1</a>">`, body: "<seq>\n&c;</seq>"},
		{name: "element-only child content invalid", entities: `<!ENTITY c "<a>x</a>">`, body: "<seq>\n\n&c;</seq>"},
		{name: "element-only descendant on a later entity line", entities: "<!ENTITY c \"<a>1</a>\n<b>x</b>\">", body: "<seq>\n&c;</seq>"},
		{name: "element-only entity then literal", entities: `<!ENTITY c "<a>1</a>">`, body: `<seq>&c;<b>2</b></seq>`, valid: true},
		{name: "element-only literal then entity", entities: `<!ENTITY c "<a>1</a>">`, body: "<seq><b>2</b>\n&c;</seq>"},
		{name: "element-only nested entities valid", entities: `<!ENTITY d "<b>2</b>"><!ENTITY c "<a>1</a>&d;">`, body: `<seq>&c;</seq>`, valid: true},
		{name: "element-only nested entities invalid", entities: `<!ENTITY d "<a>2</a>"><!ENTITY c "<a>1</a>&d;">`, body: "<seq>\n&c;</seq>"},
		{name: "element-only nested entity content invalid", entities: "<!ENTITY d \"<b>y</b>\"><!ENTITY c \"<a>1</a>\n&d;\">", body: "<seq>\n\n&c;</seq>"},
		{name: "element-only entity with whitespace between elements", entities: "<!ENTITY c \"<a>1</a>\n <b>2</b> \">", body: `<seq>&c;</seq>`, valid: true},
		{name: "mixed text and elements valid", entities: `<!ENTITY c "x<a>1</a>y">`, body: `<mix>&c;<b>2</b></mix>`, valid: true},
		{name: "mixed text and elements invalid", entities: `<!ENTITY c "x<b>1</b>">`, body: "<mix>\n&c;</mix>"},
		{name: "all valid", entities: `<!ENTITY c "<b>2</b><a>1</a>">`, body: `<all>&c;</all>`, valid: true},
		{name: "all repeated through entity", entities: `<!ENTITY c "<a>1</a>">`, body: "<all>&c;\n&c;<b>1</b></all>"},
		{name: "simple content with element", entities: `<!ENTITY c "<x/>42">`, body: `<int>&c;</int>`},
		{name: "simple content with nested element", entities: `<!ENTITY d "<x/>"><!ENTITY c "4&d;2">`, body: `<int>&c;</int>`},
		{name: "simple content with text-only entity", entities: `<!ENTITY d "2"><!ENTITY c "4&d;">`, body: `<int>&c;</int>`, valid: true},
		{name: "empty content with element", entities: `<!ENTITY c "<x/>">`, body: "<empty>\n&c;</empty>"},
		{name: "empty content with whitespace then element", entities: `<!ENTITY c " <x/>">`, body: "<empty>\n&c;</empty>"},
		{name: "empty content with text then element", entities: `<!ENTITY c "t<x/>">`, body: "<empty>\n&c;</empty>"},
		{name: "empty content with entity expanding to nothing", entities: `<!ENTITY c "">`, body: `<empty>&c;</empty>`, valid: true},
		{name: "nilled with element", entities: `<!ENTITY c "<a>1</a>">`, body: `<nil ` + xsi + ` xsi:nil="true">` + "\n&c;</nil>"},
		{name: "nilled with entity expanding to nothing", entities: `<!ENTITY c "">`, body: `<nil ` + xsi + ` xsi:nil="true">&c;</nil>`, valid: true},
		{name: "not nilled with element", entities: `<!ENTITY c "<a>1</a>">`, body: `<nil ` + xsi + ` xsi:nil="false">&c;</nil>`, valid: true},
		{name: "anyType with valid global element", entities: `<!ENTITY c "<g>1</g>">`, body: `<any>&c;</any>`, valid: true},
		{name: "anyType with invalid global element", entities: `<!ENTITY c "<g>x</g>">`, body: "<any>\n&c;</any>"},
		{name: "strict wildcard valid", entities: `<!ENTITY c "<g>1</g><g>2</g>">`, body: `<wild>&c;</wild>`, valid: true},
		{name: "strict wildcard undeclared", entities: `<!ENTITY c "<q/>">`, body: "<wild>\n&c;</wild>"},
		{name: "lax wildcard invalid global element", entities: `<!ENTITY c "<g>z</g>">`, body: "<lax>\n&c;</lax>"},
		{name: "IDs distinct", entities: `<!ENTITY c "<w><id>a</id></w>">`, body: `<ids>&c;<w><id>b</id></w></ids>`, valid: true},
		{name: "ID from entity referenced twice", entities: `<!ENTITY c "<w><id>a</id></w>">`, body: "<ids>&c;\n&c;</ids>"},
		{name: "IDREF to an entity-borne ID", entities: `<!ENTITY c "<w><id>a</id></w>">`, body: `<ids>&c;<w ref="a"><id>b</id></w></ids>`, valid: true},
		{name: "IDREF from an entity-borne element", entities: `<!ENTITY c "<w ref='z'><id>a</id></w>">`, body: "<ids>\n&c;</ids>"},
		{name: "entity-borne IDs under one host", entities: `<!ENTITY c "<id>a</id><id>b</id>">`, body: `<ids><w>&c;</w></ids>`, valid: true},
		{name: "entity-borne ID repeated under one host", entities: `<!ENTITY c "<id>a</id>">`, body: "<ids><w>&c;\n&c;</w></ids>", v10Only: true},
		{name: "entity-borne ID repeated under one host", entities: `<!ENTITY c "<id>a</id>">`, body: "<ids><w>&c;\n&c;</w></ids>", valid: true, v11Only: true},
		{name: "key on entity-borne host valid", entities: `<!ENTITY c "<keys><k>1</k><k>2</k><ref>2</ref></keys>">`, body: `<box>&c;</box>`, valid: true},
		{name: "key on entity-borne host duplicate", entities: `<!ENTITY c "<keys><k>1</k><k>1</k></keys>">`, body: "<box>\n&c;</box>"},
		{name: "keyref on entity-borne host dangling", entities: `<!ENTITY c "<keys><k>1</k><ref>3</ref></keys>">`, body: "<box>\n&c;</box>"},
		{name: "entity-borne key host referenced twice", entities: `<!ENTITY c "<keys><k>1</k><k>1</k></keys>">`, body: "<box>&c;\n&c;</box>"},
		{name: "keyref resolved by entity-borne descendant key", entities: `<!ENTITY c "<keys><k>1</k></keys>">`, body: `<outer>&c;<use>1</use></outer>`, valid: true},
		{name: "keyref dangling against entity-borne descendant key", entities: `<!ENTITY c "<keys><k>1</k></keys>">`, body: "<outer>&c;\n<use>2</use></outer>"},
		{name: "assert over entity-borne elements valid", entities: `<!ENTITY c "<v>1</v><v>2</v>">`, body: `<cmp>&c;</cmp>`, valid: true, v11Only: true},
		{name: "assert over entity-borne elements invalid", entities: `<!ENTITY c "<v>2</v><v>1</v>">`, body: "<cmp>\n&c;</cmp>", v11Only: true},
		{name: "assert over nested entity-borne elements", entities: `<!ENTITY d "<v>2</v>"><!ENTITY c "<v>1</v>&d;">`, body: `<cmp>&c;</cmp>`, valid: true, v11Only: true},
		{name: "assert over entity mixing text and elements", entities: `<!ENTITY c "x<v>1</v>y">`, body: `<cmpm>&c;</cmpm>`, valid: true, v11Only: true},
	}
}

// TestValidateEntityBorneElements verifies that elements inside an entity's
// replacement text are validated as the element children entity substitution
// would give their host. A document parsed without entity substitution (the
// default) keeps EntityRef nodes whose expansion holds those elements; its
// verdict, diagnostics (lines included), and PSVI type annotations must equal
// those of the same document parsed with SubstituteEntities(true), in XSD 1.0
// and 1.1.
func TestValidateEntityBorneElements(t *testing.T) {
	t.Parallel()

	for _, version := range []xsd.Version{xsd.Version10, xsd.Version11} {
		schemaDoc, err := helium.NewParser().Parse(t.Context(), []byte(entityBorneSchema(version == xsd.Version11)))
		require.NoError(t, err)
		schema, err := xsd.NewCompiler().Version(version).Compile(t.Context(), schemaDoc)
		require.NoError(t, err)

		for _, tc := range entityBorneCases() {
			if (tc.v11Only && version != xsd.Version11) || (tc.v10Only && version != xsd.Version10) {
				continue
			}
			t.Run(version.String()+"/"+tc.name, func(t *testing.T) {
				t.Parallel()
				instance := "<!DOCTYPE r [" + tc.entities + "]>\n" + tc.body

				refDoc, err := helium.NewParser().Parse(t.Context(), []byte(instance))
				require.NoError(t, err)
				var buf bytes.Buffer
				require.NoError(t, helium.Write(&buf, refDoc.DocumentElement()))
				require.Contains(t, buf.String(), "&", "the default parse keeps entity references")

				substDoc, err := helium.NewParser().SubstituteEntities(true).Parse(t.Context(), []byte(instance))
				require.NoError(t, err)

				var substOut, refOut string
				var substAnn, refAnn xsd.TypeAnnotations
				substErr := validateWithOutput(t, xsd.NewValidator(schema).Label("test.xml").Annotations(&substAnn), substDoc, &substOut)
				refErr := validateWithOutput(t, xsd.NewValidator(schema).Label("test.xml").Annotations(&refAnn), refDoc, &refOut)

				require.Equal(t, tc.valid, substErr == nil, "substituted parse: %s", substOut)
				require.Equal(t, substOut, refOut)
				require.Equal(t, substErr == nil, refErr == nil)
				require.Equal(t, elementOccurrenceTypes(substDoc, substAnn), elementOccurrenceTypes(refDoc, refAnn))
			})
		}
	}
}

// elementOccurrenceTypes lists the type annotation of every element of doc's
// document element tree in document order, descending through each entity
// reference into its entity's children, so a document parsed with and without
// entity substitution list the same elements.
func elementOccurrenceTypes(doc *helium.Document, ann xsd.TypeAnnotations) string {
	var b strings.Builder
	root := doc.DocumentElement()
	fmt.Fprintf(&b, "%s = %s\n", root.LocalName(), ann[root])
	appendOccurrenceTypes(&b, root, ann)
	return b.String()
}

func appendOccurrenceTypes(b *strings.Builder, n helium.Node, ann xsd.TypeAnnotations) {
	for c := range helium.Children(n) {
		switch c.Type() {
		case helium.ElementNode:
			e, ok := c.(*helium.Element)
			if !ok {
				continue
			}
			fmt.Fprintf(b, "%s = %s\n", e.LocalName(), ann[e])
			appendOccurrenceTypes(b, e, ann)
		case helium.EntityRefNode:
			if ent, ok := c.FirstChild().(*helium.Entity); ok {
				appendOccurrenceTypes(b, ent, ann)
			}
		}
	}
}
