package xsd_test

import (
	"os"
	"path/filepath"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
)

func BenchmarkCompileSchema(b *testing.B) {
	for _, fixture := range []string{"extension0_0", "nvdcve_0"} {
		b.Run(fixture, func(b *testing.B) {
			path := filepath.Join(testdataBase, "test", fixture+".xsd")
			data, err := os.ReadFile(path)
			if err != nil {
				b.Fatal(err)
			}
			doc, err := helium.NewParser().Parse(b.Context(), data)
			if err != nil {
				b.Fatal(err)
			}
			compiler := xsd.NewCompiler()

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

func BenchmarkValidateDocument(b *testing.B) {
	for _, fixture := range []string{"extension0_0", "nvdcve_0"} {
		b.Run(fixture, func(b *testing.B) {
			schemaPath := filepath.Join(testdataBase, "test", fixture+".xsd")
			schema, err := xsd.NewCompiler().CompileFile(b.Context(), schemaPath)
			if err != nil {
				b.Fatal(err)
			}
			instancePath := filepath.Join(testdataBase, "test", fixture+".xml")
			data, err := os.ReadFile(instancePath)
			if err != nil {
				b.Fatal(err)
			}
			doc, err := helium.NewParser().Parse(b.Context(), data)
			if err != nil {
				b.Fatal(err)
			}
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
