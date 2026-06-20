package xpath3_test

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

// cancelAfterNContext reports a cancelled error from Err() only after Err has
// been consulted cancelAfter times, simulating a context cancelled AFTER
// evaluation (and thus axis enumeration) has begun. It implements
// context.Context directly (no embedding) to satisfy the containedctx linter.
type cancelAfterNContext struct {
	cancelAfter int
	calls       int
}

func (c *cancelAfterNContext) Deadline() (time.Time, bool) { return time.Time{}, false }
func (c *cancelAfterNContext) Done() <-chan struct{}       { return nil }
func (c *cancelAfterNContext) Value(any) any               { return nil }

func (c *cancelAfterNContext) Err() error {
	c.calls++
	if c.calls > c.cancelAfter {
		return context.Canceled
	}
	return nil
}

// A context cancelled after evaluation starts must abort a wide child-axis
// enumeration promptly with context.Canceled, rather than scanning the entire
// child set before the per-step countOps observes the cancellation. The
// cancel-after context lets evaluation start and only reports cancellation once
// the hot child loop is already iterating, so without the in-loop ctx.Err()
// check the loop would scan all children before the next countOps boundary.
func TestEvalChildAxis_ContextCancelledWideChildSet(t *testing.T) {
	const width = 50000
	const cancelAfter = 50

	var b strings.Builder
	b.WriteString("<root>")
	for range width {
		b.WriteString("<c/>")
	}
	b.WriteString("</root>")

	doc := mustParseXML(t, b.String())
	root := doc.DocumentElement()

	compiled, err := xpath3.NewCompiler().Compile("child::*")
	require.NoError(t, err)

	ctx := &cancelAfterNContext{cancelAfter: cancelAfter}

	_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(ctx, compiled, root)
	require.ErrorIs(t, err, context.Canceled)
	// The hot child loop checks ctx.Err() once per child, so the cancel-after
	// context reaches its trip count mid-enumeration and the loop bails on the
	// first cancelled observation. Without the in-loop check the per-child
	// consultations would not happen, the trip count would never be reached, and
	// Evaluate would scan all width children and return success.
	require.LessOrEqual(t, ctx.calls, cancelAfter+1,
		"child enumeration must bail on the first cancelled Err() observation")
}

// A context cancelled after evaluation starts must abort a wide attribute-axis
// enumeration promptly with context.Canceled. The ForEachAttribute callback
// path must stop iterating and surface the cancellation error rather than
// scanning the full attribute list before the next countOps boundary.
func TestEvalAttributeAxis_ContextCancelledWideAttrSet(t *testing.T) {
	const width = 5000
	const cancelAfter = 50

	var b strings.Builder
	b.WriteString("<root")
	for i := range width {
		b.WriteString(" a")
		b.WriteString(strconv.Itoa(i))
		b.WriteString(`="v"`)
	}
	b.WriteString("/>")

	doc := mustParseXML(t, b.String())
	root := doc.DocumentElement()

	compiled, err := xpath3.NewCompiler().Compile("attribute::*")
	require.NoError(t, err)

	ctx := &cancelAfterNContext{cancelAfter: cancelAfter}

	_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(ctx, compiled, root)
	require.ErrorIs(t, err, context.Canceled)
	require.LessOrEqual(t, ctx.calls, cancelAfter+1,
		"attribute enumeration must bail on the first cancelled Err() observation")
}

// buildWideAttrElement returns "<root a0="v" a1="v" ... />" with width
// attributes, used to exercise the VM predicate fast-path attribute scans.
func buildWideAttrElement(width int) string {
	var b strings.Builder
	b.WriteString("<root")
	for i := range width {
		b.WriteString(" a")
		b.WriteString(strconv.Itoa(i))
		b.WriteString(`="v"`)
	}
	b.WriteString("/>")
	return b.String()
}

// The VM predicate fast path for `[@missing]` (vmAttributeExistsPredicateExpr)
// scans an element's attribute set looking for a matching attribute. Over a wide
// attribute set with no match it would otherwise walk every attribute before any
// later step consulted the context. With the in-loop ctx.Err() checks inside the
// ForEachAttribute callback the scan bails on the first cancelled observation and
// surfaces context.Canceled.
func TestEvalVMAttributeExistsPredicate_ContextCancelledWideAttrSet(t *testing.T) {
	const width = 5000
	const cancelAfter = 50

	xml := "<wrap>" + buildWideAttrElement(width) + "</wrap>"
	doc := mustParseXML(t, xml)
	root := doc.DocumentElement()

	compiled, err := xpath3.NewCompiler().Compile(`child::*[@missing]`)
	require.NoError(t, err)

	ctx := &cancelAfterNContext{cancelAfter: cancelAfter}

	_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(ctx, compiled, root)
	require.ErrorIs(t, err, context.Canceled)
	require.LessOrEqual(t, ctx.calls, cancelAfter+1,
		"attribute-exists predicate scan must bail on the first cancelled Err() observation")
}

// The VM predicate fast path for `[@a = "v"]`
// (vmAttributeEqualsStringPredicateExpr) scans an element's attribute set looking
// for a matching attribute whose value equals the literal. The matching
// attribute is placed last so the scan must walk the whole set; with the in-loop
// ctx.Err() checks inside the ForEachAttribute callback the scan bails on the
// first cancelled observation and surfaces context.Canceled.
func TestEvalVMAttributeEqualsStringPredicate_ContextCancelledWideAttrSet(t *testing.T) {
	const width = 5000
	const cancelAfter = 50

	var b strings.Builder
	b.WriteString("<wrap><root")
	for i := range width {
		b.WriteString(" a")
		b.WriteString(strconv.Itoa(i))
		b.WriteString(`="x"`)
	}
	// The single matching attribute sits at the end of the set.
	b.WriteString(` match="v"/></wrap>`)

	doc := mustParseXML(t, b.String())
	root := doc.DocumentElement()

	compiled, err := xpath3.NewCompiler().Compile(`child::*[@match = "v"]`)
	require.NoError(t, err)

	ctx := &cancelAfterNContext{cancelAfter: cancelAfter}

	_, err = xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(ctx, compiled, root)
	require.ErrorIs(t, err, context.Canceled)
	require.LessOrEqual(t, ctx.calls, cancelAfter+1,
		"attribute-equals predicate scan must bail on the first cancelled Err() observation")
}
