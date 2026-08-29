package xsd_test

import (
	"runtime"
	"strings"
	"testing"
	"unicode/utf8"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

func TestIDCXPathDiagnosticExcerpt(t *testing.T) {
	schema := func(xpath string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">` +
			`<xs:element name="root"><xs:key name="k">` +
			`<xs:selector xpath="` + xpath + `"/><xs:field xpath="@id"/>` +
			`</xs:key></xs:element></xs:schema>`
	}

	t.Run("short wording is unchanged", func(t *testing.T) {
		_, errs, err := compileWith(t, xsd.Version10, schema("1 foo"))
		require.ErrorIs(t, err, xsd.ErrCompilationFailed)
		require.Contains(t, errs,
			`The selector XPath '1 foo' is not a valid selector expression: `+
				`xpath: unexpected token: Name("foo") after expression.`)
	})

	t.Run("long multibyte selector is bounded", func(t *testing.T) {
		small := measureXSDCompile(t, schema("1 "+strings.Repeat("界", (1<<19)/len("界"))))
		large := measureXSDCompile(t, schema("1 "+strings.Repeat("界", (1<<20)/len("界"))))

		require.ErrorIs(t, small.compileErr, xsd.ErrCompilationFailed)
		require.ErrorIs(t, large.compileErr, xsd.ErrCompilationFailed)
		require.LessOrEqual(t, len(large.diagnostic), 2048)
		require.True(t, utf8.ValidString(large.diagnostic))
		require.Contains(t, large.diagnostic, "[truncated]")
		// The differential removes fixed compiler and race-runtime allocations.
		// Reintroducing the source-sized diagnostic copies exceeds this bound.
		require.GreaterOrEqual(t, large.allocated, small.allocated)
		require.Less(t, large.allocated-small.allocated, uint64(5<<19))
	})

	t.Run("long unbound prefix is bounded", func(t *testing.T) {
		prefix := strings.Repeat("界", (1<<20)/len("界"))
		_, errs, err := compileWith(t, xsd.Version10, schema(prefix+":item"))

		require.ErrorIs(t, err, xsd.ErrCompilationFailed)
		require.LessOrEqual(t, len(errs), 2048)
		require.True(t, utf8.ValidString(errs))
		require.Contains(t, errs, "[truncated]")
		require.Contains(t, errs, "is not bound to a namespace")
	})
}

type xsdCompileMeasurement struct {
	allocated  uint64
	compileErr error
	diagnostic string
}

func measureXSDCompile(t *testing.T, schemaXML string) xsdCompileMeasurement {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(schemaXML))
	require.NoError(t, err)
	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	compiler := xsd.NewCompiler().Version(xsd.Version10).Label("test.xsd").ErrorHandler(collector)

	var compileErr error
	allocated := xsdCompileAllocatedBytes(t, func() {
		_, compileErr = compiler.Compile(t.Context(), doc)
	})
	require.NoError(t, collector.Close())
	return xsdCompileMeasurement{
		allocated:  allocated,
		compileErr: compileErr,
		diagnostic: compileErrorsString(collector.Errors()),
	}
}

func xsdCompileAllocatedBytes(t *testing.T, fn func()) uint64 {
	t.Helper()
	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	fn()
	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	return after.TotalAlloc - before.TotalAlloc
}
