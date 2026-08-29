package xpath1_test

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/lestrrat-go/helium/xpath1"
	"github.com/stretchr/testify/require"
)

const maxXPathDiagnosticBytes = 768
const maxXPathDiagnosticAllocBytes = 1 << 20

func TestCompileDiagnosticExcerpt(t *testing.T) {
	t.Run("short diagnostic is unchanged", func(t *testing.T) {
		_, err := xpath1.NewCompiler().Compile("1 foo")
		require.EqualError(t, err, `xpath: unexpected token: Name("foo") after expression`)
		require.ErrorIs(t, err, xpath1.ErrUnexpectedToken)
	})

	for _, tc := range []struct {
		name  string
		token string
	}{
		{name: "ASCII", token: strings.Repeat("a", 1<<20)},
		{name: "multibyte UTF-8", token: strings.Repeat("界", 1<<18)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := xpath1.NewCompiler().Compile("1 " + tc.token)
			require.ErrorIs(t, err, xpath1.ErrUnexpectedToken)
			requireBoundedXPathDiagnostic(t, err)
		})
	}
}

func TestCompileDiagnosticAllocation(t *testing.T) {
	t.Run("invalid string token", func(t *testing.T) {
		payload := strings.Repeat("\xffa", 2<<20)
		input := "1 '" + payload + "'"
		var err error
		allocated := allocatedBytes(t, func() {
			_, err = xpath1.NewCompiler().Compile(input)
		})
		require.ErrorIs(t, err, xpath1.ErrUnexpectedToken)
		requireBoundedXPathDiagnostic(t, err)
		require.Less(t, allocated, uint64(maxXPathDiagnosticAllocBytes))
	})

	t.Run("QName function", func(t *testing.T) {
		prefix := strings.Repeat("p", 4<<20)
		name := strings.Repeat("n", 4<<20)
		input := prefix + ":" + name + "(1"
		var err error
		allocated := allocatedBytes(t, func() {
			_, err = xpath1.NewCompiler().Compile(input)
		})
		require.ErrorIs(t, err, xpath1.ErrExpectedToken)
		requireBoundedXPathDiagnostic(t, err)
		require.Less(t, allocated, uint64(maxXPathDiagnosticAllocBytes))
	})
}

func TestValidateDiagnosticExcerpt(t *testing.T) {
	t.Run("short diagnostic is unchanged", func(t *testing.T) {
		expr := xpath1.MustCompile("missing:node")
		err := xpath1.NewEvaluator().Validate(expr)
		require.EqualError(t, err, "xpath: unknown namespace prefix: missing")
		require.ErrorIs(t, err, xpath1.ErrUnknownNamespacePrefix)
	})

	t.Run("long prefix", func(t *testing.T) {
		prefix := strings.Repeat("界", 1<<18)
		expr := xpath1.MustCompile(prefix + ":node")
		err := xpath1.NewEvaluator().Validate(expr)
		require.ErrorIs(t, err, xpath1.ErrUnknownNamespacePrefix)
		requireBoundedXPathDiagnostic(t, err)
	})

	t.Run("long namespace URI", func(t *testing.T) {
		expr := xpath1.MustCompile("p:missing()")
		err := xpath1.NewEvaluator().
			Namespaces(map[string]string{"p": strings.Repeat("urn:x:", 1<<18)}).
			Validate(expr)
		require.ErrorIs(t, err, xpath1.ErrUnknownFunction)
		requireBoundedXPathDiagnostic(t, err)
	})
}

func TestEvaluateDiagnosticExcerpt(t *testing.T) {
	name := strings.Repeat("a", 1<<20)
	expr := xpath1.MustCompile(name + "()")
	_, err := xpath1.NewEvaluator().Evaluate(t.Context(), expr, nil)
	require.ErrorIs(t, err, xpath1.ErrUnknownFunction)
	requireBoundedXPathDiagnostic(t, err)
}

func TestMustCompileDiagnosticExcerpt(t *testing.T) {
	defer func() {
		value := recover()
		require.NotNil(t, value)
		message := fmt.Sprint(value)
		require.LessOrEqual(t, len(message), maxXPathDiagnosticBytes)
		require.True(t, utf8.ValidString(message))
		require.Contains(t, message, "[truncated]")
	}()

	xpath1.MustCompile("1 " + strings.Repeat("a", 1<<20))
}

func requireBoundedXPathDiagnostic(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err)
	require.LessOrEqual(t, len(err.Error()), maxXPathDiagnosticBytes)
	require.True(t, utf8.ValidString(err.Error()))
	require.Contains(t, err.Error(), "[truncated]")
}

func allocatedBytes(t *testing.T, fn func()) uint64 {
	t.Helper()
	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	fn()
	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	return after.TotalAlloc - before.TotalAlloc
}
