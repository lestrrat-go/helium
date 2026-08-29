package schematron_test

import (
	"runtime"
	"strings"
	"testing"
	"unicode/utf8"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/schematron"
	"github.com/stretchr/testify/require"
)

func TestCompileXPathDiagnosticExcerpt(t *testing.T) {
	const prefix = `<schema xmlns="http://purl.oclc.org/dsdl/schematron"><pattern>`
	const suffix = `</pattern></schema>`
	cases := []struct {
		name        string
		schema      func(string) string
		shortNeedle string
		maxAlloc    uint64
	}{
		{
			name: "rule context",
			schema: func(expr string) string {
				return prefix + `<rule context="` + expr + `"><assert test="true()">ok</assert></rule>` + suffix
			},
			shortNeedle: "Failed to compile context expression '1 foo'",
			maxAlloc:    5 << 20,
		},
		{
			name: "assert test",
			schema: func(expr string) string {
				return prefix + `<rule context="/root"><assert test="` + expr + `">ok</assert></rule>` + suffix
			},
			shortNeedle: "Failed to compile test expression '1 foo'",
			maxAlloc:    4 << 20,
		},
		{
			name: "name path",
			schema: func(expr string) string {
				return prefix + `<rule context="/root"><assert test="false()"><name path="` + expr + `"/></assert></rule>` + suffix
			},
			shortNeedle: "Failed to compile path '1 foo'",
			maxAlloc:    4 << 20,
		},
		{
			name: "value-of select",
			schema: func(expr string) string {
				return prefix + `<rule context="/root"><assert test="false()"><value-of select="` + expr + `"/></assert></rule>` + suffix
			},
			shortNeedle: "Failed to compile select expression '1 foo'",
			maxAlloc:    4 << 20,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, shortErrs := compileTestSchema(t, tc.schema("1 foo"))
			require.Contains(t, shortErrs, tc.shortNeedle)

			doc, err := helium.NewParser().Parse(t.Context(),
				[]byte(tc.schema("1 "+strings.Repeat("界", (1<<20)/len("界")))))
			require.NoError(t, err)
			collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
			compiler := schematron.NewCompiler().ErrorHandler(collector)

			var compileErr error
			allocated := schematronCompileAllocatedBytes(t, func() {
				_, compileErr = compiler.Compile(t.Context(), doc)
			})
			require.NoError(t, collector.Close())
			var diagnostic strings.Builder
			for _, err := range collector.Errors() {
				diagnostic.WriteString(err.Error())
			}
			message := diagnostic.String()

			require.ErrorIs(t, compileErr, schematron.ErrCompileFailed)
			require.LessOrEqual(t, len(message), 2048)
			require.True(t, utf8.ValidString(message))
			require.Contains(t, message, "[truncated]")
			require.Less(t, allocated, tc.maxAlloc)
		})
	}
}

func schematronCompileAllocatedBytes(t *testing.T, fn func()) uint64 {
	t.Helper()
	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	fn()
	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	return after.TotalAlloc - before.TotalAlloc
}
