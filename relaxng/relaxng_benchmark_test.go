package relaxng_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/relaxng"
)

// disks > 0 repeats the instance's single <disk> element that many times, so
// the per-element validation cost is visible over a larger document.
var benchmarkCases = []struct {
	name, schema, instance string
	disks                  int
}{
	{"small", "tutor10_8.rng", "tutor10_8_1.xml", 0},
	{"libvirt", "libvirt.rng", "libvirt_0.xml", 0},
	{"large", "libvirt.rng", "libvirt_0.xml", 300},
}

// Inputs come from the libxml2 compatibility corpus. Parsing is setup;
// the benchmarks measure only grammar compilation or instance validation.
func BenchmarkCompile(b *testing.B) {
	for _, tc := range benchmarkCases {
		if tc.disks > 0 {
			// Same schema as an earlier case; only the instance differs.
			continue
		}
		b.Run(tc.name, func(b *testing.B) {
			schema := benchmarkSchemaDocument(b, tc.schema)
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
		})
	}
}

func BenchmarkValidate(b *testing.B) {
	for _, tc := range benchmarkCases {
		b.Run(tc.name, func(b *testing.B) {
			schema := benchmarkSchemaDocument(b, tc.schema)
			grammar, err := relaxng.NewCompiler().Compile(b.Context(), schema)
			if err != nil {
				b.Fatal(err)
			}
			data, err := os.ReadFile(benchmarkFixturePath(tc.instance))
			if err != nil {
				b.Fatal(err)
			}
			if tc.disks > 0 {
				data = repeatDisk(b, data, tc.disks)
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
		})
	}
}

// repeatDisk replaces the single <disk>...</disk> element of a libvirt
// instance with n copies of it.
func repeatDisk(b *testing.B, data []byte, n int) []byte {
	b.Helper()
	start := bytes.Index(data, []byte("<disk "))
	end := bytes.Index(data, []byte("</disk>"))
	if start < 0 || end < start {
		b.Fatal("instance has no <disk> element")
	}
	end += len("</disk>")
	var out bytes.Buffer
	out.Write(data[:start])
	for range n {
		out.Write(data[start:end])
		out.WriteString("\n    ")
	}
	out.Write(data[end:])
	return out.Bytes()
}

func benchmarkSchemaDocument(b *testing.B, name string) *helium.Document {
	b.Helper()
	data, err := os.ReadFile(benchmarkFixturePath(name))
	if err != nil {
		b.Fatal(err)
	}
	doc, err := helium.NewParser().Parse(b.Context(), data)
	if err != nil {
		b.Fatal(err)
	}
	return doc
}

func benchmarkFixturePath(name string) string {
	return filepath.Join("..", "testdata", "libxml2-compat", "relaxng", "test", name)
}
