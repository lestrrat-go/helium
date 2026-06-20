package xpath3_test

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// A context cancelled after evaluation starts must abort a wide child-axis
// enumeration promptly with context.Canceled, rather than scanning the entire
// child set before the per-step countOps observes the cancellation.
func TestEvalChildAxis_ContextCancelledWideChildSet(t *testing.T) {
	var b strings.Builder
	b.WriteString("<root>")
	for range 50000 {
		b.WriteString("<c/>")
	}
	b.WriteString("</root>")

	doc := mustParseXML(t, b.String())
	root := doc.DocumentElement()

	compiled, err := xpath3.NewCompiler().Compile("child::*")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(ctx, compiled, root)
	require.ErrorIs(t, err, context.Canceled)
}

// A context cancelled after evaluation starts must abort a wide attribute-axis
// enumeration promptly with context.Canceled. The ForEachAttribute callback
// path must stop iterating and surface the cancellation error.
func TestEvalAttributeAxis_ContextCancelledWideAttrSet(t *testing.T) {
	var b strings.Builder
	b.WriteString("<root")
	for i := range 5000 {
		b.WriteString(" a")
		b.WriteString(strconv.Itoa(i))
		b.WriteString(`="v"`)
	}
	b.WriteString("/>")

	doc := mustParseXML(t, b.String())
	root := doc.DocumentElement()

	compiled, err := xpath3.NewCompiler().Compile("attribute::*")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(ctx, compiled, root)
	require.ErrorIs(t, err, context.Canceled)
}
