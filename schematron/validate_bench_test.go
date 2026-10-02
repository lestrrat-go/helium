package schematron_test

import (
	"fmt"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/schematron"
	"github.com/stretchr/testify/require"
)

// benchRecords is the number of <book> records in the instance document.
// 500 records is about 110 KB of XML.
const benchRecords = 500

// benchInstance builds an instance document with records that all satisfy
// the rules, so validation always walks every rule over every node.
func benchInstance(records int) string {
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n<catalog>\n")
	for i := range records {
		fmt.Fprintf(&sb, `  <book isbn="978000000%04d" year="20%02d">
    <title>Title number %d</title>
    <author>Author %d</author>
    <price currency="EUR">%d.99</price>
    <tags><tag>alpha</tag><tag>beta</tag><tag>gamma</tag></tags>
  </book>
`, i%10000, i%100, i, i%50, 10+i%90)
	}
	sb.WriteString("</catalog>\n")
	return sb.String()
}

// benchSchema is a schema with several rules, all written so that they mean
// the same thing under both query language bindings.
func benchSchema(binding string) string {
	attr := ""
	if binding != "" {
		attr = ` queryBinding="` + binding + `"`
	}
	return `<?xml version="1.0" encoding="UTF-8"?>
<schema xmlns="http://purl.oclc.org/dsdl/schematron"` + attr + `>
  <pattern>
    <rule context="book">
      <let name="isbn" value="string(@isbn)"/>
      <assert test="string-length($isbn) = 13">isbn must be 13 characters</assert>
      <assert test="title">title is required</assert>
      <assert test="author">author is required</assert>
      <assert test="count(tags/tag) = 3">expected three tags</assert>
    </rule>
  </pattern>
  <pattern>
    <rule context="price">
      <assert test="@currency = 'EUR'">price must be in euros</assert>
    </rule>
  </pattern>
  <pattern>
    <rule context="title">
      <assert test="normalize-space(.) != ''">title must not be empty</assert>
    </rule>
  </pattern>
</schema>`
}

func benchCompile(b *testing.B, binding string) *schematron.Schema {
	b.Helper()

	doc, err := helium.NewParser().Parse(b.Context(), []byte(benchSchema(binding)))
	require.NoError(b, err, "parse schema")

	schema, err := schematron.NewCompiler().Compile(b.Context(), doc)
	require.NoError(b, err, "compile schema")
	return schema
}

func benchParseInstance(b *testing.B, records int) *helium.Document {
	b.Helper()

	doc, err := helium.NewParser().Parse(b.Context(), []byte(benchInstance(records)))
	require.NoError(b, err, "parse instance")
	return doc
}

func benchValidate(b *testing.B, binding string, records int) {
	b.Helper()

	schema := benchCompile(b, binding)
	doc := benchParseInstance(b, records)

	b.ResetTimer()
	for b.Loop() {
		if err := schematron.NewValidator(schema).Quiet().Validate(b.Context(), doc); err != nil {
			b.Fatalf("validation failed: %s", err)
		}
	}
}

func BenchmarkValidateDefaultBinding(b *testing.B) {
	benchValidate(b, "", benchRecords)
}

func BenchmarkValidateXSLT1Binding(b *testing.B) {
	benchValidate(b, "xslt", benchRecords)
}

func BenchmarkValidateXSLT3Binding(b *testing.B) {
	benchValidate(b, "xslt3", benchRecords)
}

// benchNodePathSchema is an xslt3-binding schema whose tests take the
// effective boolean value of node paths: a path on its own, exists() and
// empty() of a path, count() of a path compared with zero, and report tests
// that never fire. Every assertion holds on benchInstance, so validation
// evaluates every test on every rule context.
const benchNodePathSchema = `<?xml version="1.0" encoding="UTF-8"?>
<schema xmlns="http://purl.oclc.org/dsdl/schematron" queryBinding="xslt3">
  <pattern>
    <rule context="catalog">
      <assert test="exists(//book)">a catalog holds books</assert>
      <assert test="//tag">a catalog holds tags</assert>
      <assert test="count(//price) > 0">a catalog holds prices</assert>
      <report test="//book[not(title)]">every book has a title</report>
    </rule>
  </pattern>
  <pattern>
    <rule context="book">
      <assert test=".//tag">a book has tags</assert>
      <assert test="exists(tags/tag)">a book lists its tags</assert>
      <assert test="count(.//tag) > 0">a book counts its tags</assert>
      <assert test="title and author">a book has a title and an author</assert>
      <report test=".//nosuch">a book has no nosuch</report>
      <report test="empty(price/@currency)">a price has a currency</report>
    </rule>
  </pattern>
  <pattern>
    <rule context="tag">
      <assert test="ancestor::book">a tag sits in a book</assert>
      <report test="following-sibling::nosuch">a tag has no nosuch sibling</report>
    </rule>
  </pattern>
</schema>`

// BenchmarkValidateNodePathTests validates benchInstance against
// benchNodePathSchema, whose tests only need to know whether a node path
// selects a node.
func BenchmarkValidateNodePathTests(b *testing.B) {
	sdoc, err := helium.NewParser().Parse(b.Context(), []byte(benchNodePathSchema))
	require.NoError(b, err, "parse schema")
	schema, err := schematron.NewCompiler().Compile(b.Context(), sdoc)
	require.NoError(b, err, "compile schema")
	doc := benchParseInstance(b, benchRecords)
	require.NoError(b, schematron.NewValidator(schema).Quiet().Validate(b.Context(), doc))

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if err := schematron.NewValidator(schema).Quiet().Validate(b.Context(), doc); err != nil {
			b.Fatalf("validation failed: %s", err)
		}
	}
}
