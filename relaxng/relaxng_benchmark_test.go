package relaxng_test

import (
	"os"
	"path/filepath"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/relaxng"
)

// Both inputs come from the libxml2 compatibility corpus. Parsing is setup;
// the benchmarks measure only grammar compilation or instance validation.
func BenchmarkCompile(b *testing.B) {
	schema := benchmarkSchemaDocument(b)
	compiler := relaxng.NewCompiler()
	if _, err := compiler.Compile(b.Context(), schema); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		grammar, err := compiler.Compile(b.Context(), schema)
		if err != nil {
			b.Fatal(err)
		}
		if grammar == nil {
			b.Fatal("nil grammar")
		}
	}
}

func BenchmarkValidate(b *testing.B) {
	schema := benchmarkSchemaDocument(b)
	grammar, err := relaxng.NewCompiler().Compile(b.Context(), schema)
	if err != nil {
		b.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join("..", "testdata", "libxml2-compat", "relaxng", "test", "tutor10_8_1.xml"))
	if err != nil {
		b.Fatal(err)
	}
	doc, err := helium.NewParser().Parse(b.Context(), data)
	if err != nil {
		b.Fatal(err)
	}
	validator := relaxng.NewValidator(grammar)
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
}

func benchmarkSchemaDocument(b *testing.B) *helium.Document {
	b.Helper()
	data, err := os.ReadFile(filepath.Join("..", "testdata", "libxml2-compat", "relaxng", "test", "tutor10_8.rng"))
	if err != nil {
		b.Fatal(err)
	}
	doc, err := helium.NewParser().Parse(b.Context(), data)
	if err != nil {
		b.Fatal(err)
	}
	return doc
}
