package xpath3_test

import (
	"sync"
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// A single DocOrderCache shared across concurrent Evaluate calls must not race
// on its internal maps. Run under `go test -race` to catch concurrent map
// writes in BuildFrom/computeSortKey/cachedRootLocked.
func TestDocOrderCache_ConcurrentEvaluate(t *testing.T) {
	const xml = `<root>
		<a><b/><c/></a>
		<a><b/><c/></a>
		<a><b/><c/></a>
	</root>`

	doc := mustParseXML(t, xml)

	// A union expression forces document-order deduplication/sorting, which
	// drives the DocOrderCache map mutations.
	compiled, err := xpath3.NewCompiler().Compile("//b | //c | //a")
	require.NoError(t, err)

	cache := xpath3.NewDocOrderCache()
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).DocOrderCache(cache)

	const goroutines = 8
	var wg sync.WaitGroup
	wg.Add(goroutines)
	errs := make([]error, goroutines)
	for i := range goroutines {
		go func(idx int) {
			defer wg.Done()
			for range 50 {
				_, evalErr := eval.Evaluate(t.Context(), compiled, doc)
				if evalErr != nil {
					errs[idx] = evalErr
					return
				}
			}
		}(i)
	}
	wg.Wait()

	for _, e := range errs {
		require.NoError(t, e)
	}
}
