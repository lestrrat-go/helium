package relaxng_test

import (
	"errors"
	"os"
	"strings"
	"sync"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/relaxng"
	"github.com/stretchr/testify/require"
)

// concurrentGrammar is one compiled golden schema and the instances validated
// against it.
type concurrentGrammar struct {
	grammar   *relaxng.Grammar
	instances []concurrentInstance
}

// concurrentInstance is one golden instance: its source bytes, the label used
// in diagnostics, and the result a single-goroutine validation produced.
type concurrentInstance struct {
	label string
	data  []byte
	want  string
}

// validationRecord validates data against grammar and returns the verdict and
// every diagnostic as one string.
func validationRecord(t *testing.T, grammar *relaxng.Grammar, label string, data []byte) string {
	doc, err := helium.NewParser().Parse(t.Context(), data)
	if err != nil {
		return "parse error: " + err.Error()
	}
	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	verr := relaxng.NewValidator(grammar).Label(label).ErrorHandler(collector).Validate(t.Context(), doc)
	_ = collector.Close()
	var sb strings.Builder
	for _, e := range collector.Errors() {
		sb.WriteString(e.Error())
	}
	switch {
	case verr == nil:
		sb.WriteString("validates\n")
	case errors.Is(verr, relaxng.ErrValidationFailed):
		sb.WriteString("fails to validate\n")
	default:
		sb.WriteString("error: " + verr.Error() + "\n")
	}
	return sb.String()
}

// validateAllInstances validates every instance of every grammar in order and
// returns one record per instance.
func validateAllInstances(t *testing.T, grammars []*concurrentGrammar) []string {
	var got []string
	for _, g := range grammars {
		for _, inst := range g.instances {
			got = append(got, validationRecord(t, g.grammar, inst.label, inst.data))
		}
	}
	return got
}

// TestGrammarConcurrentValidation validates every golden instance from several
// goroutines at once, each goroutine sharing the one compiled Grammar per
// schema. Validation must never write to the Grammar, so under -race this
// fails on any shared write, and every goroutine must reproduce the
// single-goroutine result byte for byte.
func TestGrammarConcurrentValidation(t *testing.T) {
	t.Parallel()
	cases := discoverTests(t)
	require.NotEmpty(t, cases, "no test cases discovered")

	byRNG := map[string]*concurrentGrammar{}
	var order []*concurrentGrammar
	for _, tc := range cases {
		g, seen := byRNG[tc.rngPath]
		if !seen {
			collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
			grammar, err := relaxng.NewCompiler().FS(helium.PermissiveFS()).ErrorHandler(collector).CompileFile(t.Context(), tc.rngPath)
			_ = collector.Close()
			require.NoError(t, err, "compile %s", tc.rngPath)
			_, compileErrors := partitionCompileErrors(collector.Errors())
			if compileErrors != "" {
				g = nil
			} else {
				g = &concurrentGrammar{grammar: grammar}
				order = append(order, g)
			}
			byRNG[tc.rngPath] = g
		}
		if g == nil {
			continue
		}
		data, err := os.ReadFile(tc.xmlPath)
		require.NoError(t, err)
		label := "./test/relaxng/" + tc.xmlBase
		g.instances = append(g.instances, concurrentInstance{
			label: label,
			data:  data,
			want:  validationRecord(t, g.grammar, label, data),
		})
	}
	require.NotEmpty(t, order, "no golden schema compiled cleanly")

	const workers = 8
	results := make([][]string, workers)
	var wg sync.WaitGroup
	for w := range workers {
		wg.Go(func() { results[w] = validateAllInstances(t, order) })
	}
	wg.Wait()

	var want []string
	for _, g := range order {
		for _, inst := range g.instances {
			want = append(want, inst.want)
		}
	}
	for w, got := range results {
		require.Equal(t, want, got, "goroutine %d diverged from the single-goroutine result", w)
	}
}
