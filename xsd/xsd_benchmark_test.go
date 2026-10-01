package xsd_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
)

var (
	bothVersions = []xsd.Version{xsd.Version10, xsd.Version11}
	only11       = []xsd.Version{xsd.Version11}
)

// A case with a fixture name loads <fixture>.xsd and <fixture>.xml from the
// libxml2 compatibility corpus. A case with an inline schema builds an
// assertCTAInstance of the given item count in memory, so a large document
// needs no committed fixture.
// Every case runs once per listed version, as <case>/<version>.
var xsdBenchmarkCases = []struct {
	name     string
	fixture  string
	schema   string
	items    int
	versions []xsd.Version
}{
	{name: "extension0_0", fixture: "extension0_0", versions: bothVersions},
	{name: "nvdcve_0", fixture: "nvdcve_0", versions: bothVersions},
	// xs:assert and xs:alternative exist only in XSD 1.1.
	{name: "assert_cta_1000", schema: assertCTASchema, items: 1000, versions: only11},
}

// assertCTASchema selects each <item>'s type through conditional type
// assignment on @kind, and both candidate types carry an xs:assert, so every
// item costs one alternative test and at least one assertion.
const assertCTASchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="order">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="item" type="itemType" maxOccurs="unbounded">
          <xs:alternative test="@kind = 'bulk'" type="bulkItemType"/>
          <xs:alternative type="itemType"/>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
  <xs:complexType name="itemType">
    <xs:sequence>
      <xs:element name="sku" type="xs:string"/>
      <xs:element name="qty" type="xs:integer"/>
    </xs:sequence>
    <xs:attribute name="kind" type="xs:string"/>
    <xs:assert test="qty ge 1"/>
  </xs:complexType>
  <xs:complexType name="bulkItemType">
    <xs:complexContent>
      <xs:restriction base="itemType">
        <xs:sequence>
          <xs:element name="sku" type="xs:string"/>
          <xs:element name="qty" type="xs:integer"/>
        </xs:sequence>
        <xs:attribute name="kind" type="xs:string"/>
        <xs:assert test="qty ge 100"/>
      </xs:restriction>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`

// assertCTAInstance builds an <order> of n items that alternate between the
// bulk and the default alternative.
func assertCTAInstance(n int) []byte {
	var sb strings.Builder
	sb.WriteString("<order>\n")
	for i := range n {
		if i%2 == 0 {
			fmt.Fprintf(&sb, "  <item kind=\"bulk\"><sku>B%d</sku><qty>%d</qty></item>\n", i, 100+i)
			continue
		}
		fmt.Fprintf(&sb, "  <item kind=\"unit\"><sku>U%d</sku><qty>%d</qty></item>\n", i, 1+i)
	}
	sb.WriteString("</order>\n")
	return []byte(sb.String())
}

// Parsing is setup; the benchmarks measure only schema compilation or
// instance validation.
func BenchmarkCompileSchema(b *testing.B) {
	for _, tc := range xsdBenchmarkCases {
		for _, v := range tc.versions {
			b.Run(tc.name+"/"+v.String(), func(b *testing.B) {
				doc := benchmarkParse(b, benchmarkSchemaBytes(b, tc.fixture, tc.schema))
				compiler := xsd.NewCompiler().Version(v)

				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					if _, err := compiler.Compile(b.Context(), doc); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

func BenchmarkValidateDocument(b *testing.B) {
	for _, tc := range xsdBenchmarkCases {
		for _, v := range tc.versions {
			b.Run(tc.name+"/"+v.String(), func(b *testing.B) {
				compiler := xsd.NewCompiler().Version(v)
				var schema *xsd.Schema
				var err error
				var data []byte
				if tc.fixture != "" {
					schema, err = compiler.CompileFile(b.Context(), benchmarkFixturePath(tc.fixture+".xsd"))
					if err != nil {
						b.Fatal(err)
					}
					data, err = os.ReadFile(benchmarkFixturePath(tc.fixture + ".xml"))
					if err != nil {
						b.Fatal(err)
					}
				} else {
					schema, err = compiler.Compile(b.Context(), benchmarkParse(b, []byte(tc.schema)))
					if err != nil {
						b.Fatal(err)
					}
					data = assertCTAInstance(tc.items)
				}
				doc := benchmarkParse(b, data)
				validator := xsd.NewValidator(schema)
				if err := validator.Validate(b.Context(), doc); err != nil {
					b.Fatal(err)
				}

				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					if err := validator.Validate(b.Context(), doc); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

// benchmarkSchemaBytes returns the inline schema, or the fixture's .xsd file
// when fixture is set.
func benchmarkSchemaBytes(b *testing.B, fixture, inline string) []byte {
	b.Helper()
	if fixture == "" {
		return []byte(inline)
	}
	data, err := os.ReadFile(benchmarkFixturePath(fixture + ".xsd"))
	if err != nil {
		b.Fatal(err)
	}
	return data
}

func benchmarkParse(b *testing.B, data []byte) *helium.Document {
	b.Helper()
	doc, err := helium.NewParser().Parse(b.Context(), data)
	if err != nil {
		b.Fatal(err)
	}
	return doc
}

func benchmarkFixturePath(name string) string {
	return filepath.Join(testdataBase, "test", name)
}
