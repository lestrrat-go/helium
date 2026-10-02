package xpath3_test

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xsd"
	"github.com/stretchr/testify/require"
)

// nodeListExprs are expressions in which an operand that evaluates to a list
// of nodes (a location path, a union, an intersect/except, a path ending in
// an axis step, or a filter over one of these) is read as a node list
// instead of a sequence of node items. Each operand under test is marked
// «X». The fast form puts X in parentheses, which only group it; the
// reference form spells the operand as `(if (true()) then X else ())`, which
// evaluates X to its sequence of node items and passes it on unchanged,
// charging no operations of its own.
var nodeListExprs = []string{
	// The result of the whole expression.
	"«//b | //c»", "«//b»", "«//c | //b | //@id»", "«//b except //b[c]»", "«//* intersect //b»",
	"«(//b)[2]»", "«(//b | //c)[last()]»", "«(//*)[@id]»", "«//b/c»", "«(//b)[1]/c»",
	"«/*/*//node()»", "«$nodes | //b»", "«$other//b | //b»", "«//nosuch | //nosuch»",
	// Operands of union and intersect/except.
	"«//b» | «//c»", "«//b» | «$other//b»", "«$other//c» | «//c»", "«//@id» | «//b»",
	"«//b» except «//b[c]»", "«//*» intersect «//c»", "«//b» | «//b»", "«//nosuch» | «//c»",
	"(«//b» | «//c») | «//a»", "«//b | //c» | «//a»", "«/*/*» | «/*»",
	// Functions reading the node list.
	"count(«//b | //c»)", "exists(«//b except //b»)", "empty(«//nosuch | //b»)", "boolean(«//b | //c»)",
	"not(«//b intersect //c»)", "head(«//c | //b»)", "head(«//nosuch | //nosuch»)", "count(«(//b)[2]»)",
	"string-join(«//b | //c»/@id, ',')", "sum(«//b | //c» ! 1)",
	// The base of a filter, a path step, a path and a simple map.
	"(«//b | //c»)[2]", "(«//b | //c»)[last()]", "(«//b except //c»)[@id]", "(«//b | //c»)[1][1]",
	"«//b | //c»/@id", "«//b | //c»/c", "«//b»/string(@id)", "«//b | //c»/(c | .)",
	"«//b»/(1)", "«//b»/(1, .)", "«//b» ! string(@id)", "«//b» ! position()", "«//b | //c» ! last()",
	"«//b» ! .", "«//nosuch» ! 1",
	// Effective boolean values of operands that are not location paths or
	// path expressions.
	"//*[«@id | c»]", "//*[«b except b[c]»]", "//b[«(c)[1]»]",
	// Errors.
	"«//b» | 1", "1 | «//b»", "«//b» except 1", "(1, «//b»)[1]/c", "«//b»/(., 1)",
}

// nodeListExistsExprs are nodeListExprs templates whose marked operand is a
// location path or a path expression in a place that only takes whether it
// selects a node, so the fast form stops at the first node it selects
// (pathExists) while the reference form evaluates it in full. Both select the
// same nodes, but the fast form may charge fewer operations and may stay
// under a node-set limit the reference form hits.
var nodeListExistsExprs = []string{
	"//*[«.//c»]", "//b[«c»]", "(//*)[«c»]",
	"if («//c») then 1 else 2", "if («//nosuch») then 1 else 2", "«//c» and «//b»", "«//nosuch» or «//b»",
	"«//b» and «//nosuch»", "some $x in //b satisfies «$x/c»", "every $x in //b satisfies «$x//c»",
	"//b[«.//c» or «@id»]", "//*[not(«c»)]",
}

// nodeListForms returns the fast and the reference form of a nodeListExprs
// template.
func nodeListForms(tmpl string) (string, string) {
	fast := strings.NewReplacer("«", "(", "»", ")").Replace(tmpl)
	ref := strings.NewReplacer("«", "(if (true()) then ", "»", " else ())").Replace(tmpl)
	return fast, ref
}

// TestNodeListConsumers checks that every expression of nodeListExprs
// evaluates exactly like its reference form, over documents with namespaces,
// mixed content, entity references and DTDs, from every kind of context node.
func TestNodeListConsumers(t *testing.T) {
	t.Parallel()
	for _, d := range stepOrderDocs {
		f := newStepOrderFixture(t, d)
		for _, c := range stepOrderContexts {
			ctxNode := f.contextNode(t, c.expr, c.pick)
			if ctxNode == nil {
				continue
			}
			for _, tmpl := range slices.Concat(nodeListExprs, nodeListExistsExprs) {
				fast, ref := nodeListForms(tmpl)
				want := f.describeResult(t, ctxNode, ref, false)
				got := f.describeResult(t, ctxNode, fast, false)
				require.Equal(t, want, got, "%s|%s|%s", d.name, c.name, fast)
			}
		}
	}
}

// TestNodeListConsumersRandom repeats TestNodeListConsumers over generated
// documents.
func TestNodeListConsumersRandom(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewPCG(5, 6)) //nolint:gosec // deterministic test input
	for range 20 {
		f := newStepOrderFixture(t, stepOrderDoc{name: "random", src: randomStepOrderDoc(rng)})
		for _, c := range stepOrderContexts {
			ctxNode := f.contextNode(t, c.expr, c.pick)
			if ctxNode == nil {
				continue
			}
			for _, tmpl := range slices.Concat(nodeListExprs, nodeListExistsExprs) {
				fast, ref := nodeListForms(tmpl)
				require.Equal(t, f.describeResult(t, ctxNode, ref, false), f.describeResult(t, ctxNode, fast, false), fast)
			}
		}
	}
}

// TestNodeListConsumersLimits checks that reading an operand as a node list
// charges the same operations and fails on the same node-set limits as
// reading its node items. An operand of nodeListExistsExprs stops at its
// first node, so it only has to do no more work than its reference.
func TestNodeListConsumersLimits(t *testing.T) {
	t.Parallel()
	for _, d := range stepOrderDocs {
		f := newStepOrderFixture(t, d)
		for _, tmpl := range nodeListExprs {
			fast, ref := nodeListForms(tmpl)
			name := d.name + "|" + fast
			if _, err := evalWith(t, f.eval, f.doc, ref); err != nil {
				continue
			}
			require.Equal(t, smallestOpLimitWith(t, f.eval, f.doc, ref), smallestOpLimitWith(t, f.eval, f.doc, fast), name)
			for _, limit := range []int{1, 2, 3, 5, 8, 13} {
				limited := f.eval.MaxNodesForTesting(limit)
				_, fastErr := evalWith(t, limited, f.doc, fast)
				_, refErr := evalWith(t, limited, f.doc, ref)
				require.Equal(t, errors.Is(refErr, xpath3.ErrNodeSetLimit), errors.Is(fastErr, xpath3.ErrNodeSetLimit), "%s maxNodes=%d", name, limit)
			}
		}
		for _, tmpl := range nodeListExistsExprs {
			fast, ref := nodeListForms(tmpl)
			name := d.name + "|" + fast
			if _, err := evalWith(t, f.eval, f.doc, ref); err != nil {
				continue
			}
			requireNoMoreWork(t, f.eval, f.doc, fast, ref, name)
		}
	}
}

// requireNoMoreWork checks that fast, an expression whose location paths stop
// at the first node they select, needs no more operations than ref, the same
// expression evaluated in full, and only fails on a node-set limit that ref
// fails on too.
func requireNoMoreWork(t *testing.T, eval xpath3.Evaluator, node helium.Node, fast, ref, name string) {
	t.Helper()
	require.LessOrEqual(t, smallestOpLimitWith(t, eval, node, fast), smallestOpLimitWith(t, eval, node, ref), name)
	for _, limit := range []int{1, 2, 3, 5, 8, 13} {
		limited := eval.MaxNodesForTesting(limit)
		_, fastErr := evalWith(t, limited, node, fast)
		if fastErr == nil {
			continue
		}
		require.ErrorIs(t, fastErr, xpath3.ErrNodeSetLimit, "%s maxNodes=%d", name, limit)
		_, refErr := evalWith(t, limited, node, ref)
		require.Error(t, refErr, "%s maxNodes=%d", name, limit)
	}
}

// TestNodeListConsumersCancel checks that a cancelled context stops the
// node-list consumers.
func TestNodeListConsumersCancel(t *testing.T) {
	t.Parallel()
	doc := parseStepOrderDoc(t, stepOrderDocs[0].src, false)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	for _, expr := range []string{"//b | //c", "count(//b | //c)", "//*[.//c]", "//b ! 1", "(//b | //c)[1]"} {
		compiled := xpath3.NewCompiler().MustCompile(expr)
		_, err := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).Evaluate(ctx, compiled, doc)
		require.ErrorIs(t, err, context.Canceled, expr)
	}
}

// TestNodeListResult checks the Result of an expression that evaluates to a
// node list: Sequence returns an ItemSlice holding the node items the
// reference form holds, Nodes returns a copy, and the other accessors answer
// as they do for the reference.
func TestNodeListResult(t *testing.T) {
	t.Parallel()
	f := newStepOrderFixture(t, stepOrderDocs[1])
	for _, tmpl := range []string{"«//b | //c»", "«//b»", "«//nosuch»", "«(//b)[1]»", "«//b/@id»", "«/*»"} {
		fast, ref := nodeListForms(tmpl)
		want, err := evalWith(t, f.eval, f.doc, ref)
		require.NoError(t, err, ref)
		got, err := evalWith(t, f.eval, f.doc, fast)
		require.NoError(t, err, fast)

		wantSeq, ok := want.Sequence().(xpath3.ItemSlice)
		require.True(t, ok, ref)
		gotSeq, ok := got.Sequence().(xpath3.ItemSlice)
		require.True(t, ok, fast)
		requireSameItems(t, wantSeq, gotSeq, fast)
		// Every call returns the same sequence.
		again, ok := got.Sequence().(xpath3.ItemSlice)
		require.True(t, ok, fast)
		require.Equal(t, fmt.Sprintf("%p", gotSeq), fmt.Sprintf("%p", again), fast)

		require.Equal(t, want.IsNodeSet(), got.IsNodeSet(), fast)
		require.Equal(t, want.IsAtomic(), got.IsAtomic(), fast)
		require.Equal(t, want.StringValue(), got.StringValue(), fast)
		wantB, wantOK := want.IsBoolean()
		gotB, gotOK := got.IsBoolean()
		require.Equal(t, []bool{wantB, wantOK}, []bool{gotB, gotOK}, fast)
		_, wantOK = want.IsNumber()
		_, gotOK = got.IsNumber()
		require.Equal(t, wantOK, gotOK, fast)
		_, wantOK = want.IsString()
		_, gotOK = got.IsString()
		require.Equal(t, wantOK, gotOK, fast)
		_, wantErr := want.Atomics()
		_, gotErr := got.Atomics()
		require.Equal(t, fmt.Sprint(wantErr), fmt.Sprint(gotErr), fast)

		wantNodes, err := want.Nodes()
		require.NoError(t, err)
		gotNodes, err := got.Nodes()
		require.NoError(t, err)
		require.Equal(t, f.describeNodes(wantNodes), f.describeNodes(gotNodes), fast)
		require.Equal(t, wantNodes == nil, gotNodes == nil, fast)
		if len(gotNodes) > 0 {
			// Nodes returns a copy the caller owns.
			gotNodes[0] = nil
			again, err := got.Nodes()
			require.NoError(t, err)
			require.Equal(t, f.describeNodes(wantNodes), f.describeNodes(again), fast)
		}

		wantCopy := want.Copy()
		gotCopy := got.Copy()
		requireSameItems(t, wantCopy.Sequence(), gotCopy.Sequence(), fast)
		require.Equal(t, wantCopy.Sequence() == nil, gotCopy.Sequence() == nil, fast)
	}
}

// TestNodeListResultReuse checks EvaluateReuse on an expression that
// evaluates to a node list.
func TestNodeListResultReuse(t *testing.T) {
	t.Parallel()
	f := newStepOrderFixture(t, stepOrderDocs[0])
	state := f.eval.NewEvalState(f.doc)
	compiled := xpath3.NewCompiler().MustCompile("//b | //c")
	first, err := compiled.EvaluateReuse(t.Context(), state, f.doc)
	require.NoError(t, err)
	kept := first.Copy()
	elem := f.doc.DocumentElement()
	second, err := compiled.EvaluateReuse(t.Context(), state, elem.FirstChild())
	require.NoError(t, err)

	want, err := evalWith(t, f.eval, f.doc, "if (true()) then //b | //c else ()")
	require.NoError(t, err)
	requireSameItems(t, want.Sequence(), kept.Sequence(), "kept")
	want, err = evalWith(t, f.eval, elem.FirstChild(), "if (true()) then //b | //c else ()")
	require.NoError(t, err)
	requireSameItems(t, want.Sequence(), second.Sequence(), "second")
}

// TestNodeListResultConcurrentSequence checks that goroutines calling
// Sequence on one Result all get the same sequence.
func TestNodeListResultConcurrentSequence(t *testing.T) {
	t.Parallel()
	f := newStepOrderFixture(t, stepOrderDocs[0])
	r, err := evalWith(t, f.eval, f.doc, "//b | //c")
	require.NoError(t, err)
	seqs := make([]xpath3.Sequence, 8)
	var wg sync.WaitGroup
	for i := range seqs {
		wg.Go(func() {
			seqs[i] = r.Sequence()
		})
	}
	wg.Wait()
	for _, seq := range seqs {
		require.Equal(t, fmt.Sprintf("%p", seqs[0]), fmt.Sprintf("%p", seq))
	}
}

// nodeListSchema types the elements of nodeListSchemaDoc as a list, a union
// and an integer, so their node items carry type annotations.
const nodeListSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:simpleType name="IntList"><xs:list itemType="xs:int"/></xs:simpleType>
  <xs:simpleType name="IntOrWord"><xs:union memberTypes="xs:int xs:NCName"/></xs:simpleType>
  <xs:element name="r">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="v" type="IntList" maxOccurs="unbounded"/>
        <xs:element name="u" type="IntOrWord" maxOccurs="unbounded"/>
        <xs:element name="n" type="xs:integer" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

const nodeListSchemaDoc = `<r><v>1 2</v><v>3</v><u>abc</u><u>42</u><n>7</n><n>8</n></r>`

// TestNodeListTypeAnnotations checks that the node items of expressions
// that read node lists carry the type annotations of a schema-validated
// document: every item equals the item the context item expression gives
// for its node, and atomization yields the typed values.
func TestNodeListTypeAnnotations(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	schema, err := xsd.NewCompiler().Compile(ctx, mustParseXML(t, nodeListSchema))
	require.NoError(t, err)
	doc := mustParseXML(t, nodeListSchemaDoc)
	ann := make(xsd.TypeAnnotations)
	require.NoError(t, xsd.NewValidator(schema).Annotations(&ann).Validate(ctx, doc))
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		SchemaDeclarations(schema.Declarations()).
		TypeAnnotations(ann)

	tmpls := []string{
		"«//v | //u | //n»", "«//v» | «//u»", "«//* except //r»", "«(//v | //n)[last()]»", "«(//u)[2]»",
		"«/r/*»", "«(/r)/*»", "head(«//u | //n»)", "(«//v | //u»)[2]",
	}
	annotated := 0
	for _, tmpl := range tmpls {
		fast, ref := nodeListForms(tmpl)
		want, err := evalWith(t, eval, doc, ref)
		require.NoError(t, err, ref)
		got, err := evalWith(t, eval, doc, fast)
		require.NoError(t, err, fast)
		requireSameItems(t, want.Sequence(), got.Sequence(), fast)
		for item := range got.Sequence().Items() {
			ni, ok := item.(xpath3.NodeItem)
			require.True(t, ok, fast)
			self, err := evalWith(t, eval, ni.Node, ".")
			require.NoError(t, err)
			requireSameItems(t, self.Sequence(), xpath3.ItemSlice{ni}, fast)
			if ni.ListItemType != "" || ni.ActiveUnionMember != nil {
				annotated++
			}
		}

		fastData, err := evalWith(t, eval, doc, "data("+fast+")")
		require.NoError(t, err, fast)
		refData, err := evalWith(t, eval, doc, "data("+ref+")")
		require.NoError(t, err, ref)
		requireSameItems(t, refData.Sequence(), fastData.Sequence(), "data "+fast)
	}
	require.Positive(t, annotated)

	r, err := evalWith(t, eval, doc, "data(//v | //u)")
	require.NoError(t, err)
	var types []string
	for item := range r.Sequence().Items() {
		av, ok := item.(xpath3.AtomicValue)
		require.True(t, ok)
		types = append(types, av.TypeName)
	}
	require.Equal(t, []string{"xs:int", "xs:int", "xs:int", "xs:NCName", "xs:int"}, types)
}

// requireSameItems checks that got holds the items of want: the same nodes
// with the same node item fields, and equal non-node items.
func requireSameItems(t *testing.T, want, got xpath3.Sequence, msg string) {
	t.Helper()
	require.Equal(t, sequenceLen(want), sequenceLen(got), msg)
	for i := range sequenceLen(want) {
		wantNI, wantIsNode := want.Get(i).(xpath3.NodeItem)
		gotNI, gotIsNode := got.Get(i).(xpath3.NodeItem)
		require.Equal(t, wantIsNode, gotIsNode, "%s: item %d", msg, i)
		if !wantIsNode {
			require.Equal(t, want.Get(i), got.Get(i), "%s: item %d", msg, i)
			continue
		}
		require.True(t, wantNI.Node == gotNI.Node, "%s: item %d is another node", msg, i)
		wantNI.Node = nil
		gotNI.Node = nil
		require.Equal(t, wantNI, gotNI, "%s: item %d", msg, i)
	}
}

// sequenceLen returns the length of seq, which may be nil.
func sequenceLen(seq xpath3.Sequence) int {
	if seq == nil {
		return 0
	}
	return seq.Len()
}

// smallestOpLimitWith returns the smallest operation limit under which eval
// evaluates expr without ErrOpLimit.
func smallestOpLimitWith(t *testing.T, eval xpath3.Evaluator, node helium.Node, expr string) int {
	t.Helper()
	lo, hi := 1, 1
	for {
		_, err := evalWith(t, eval.OpLimit(hi), node, expr)
		if err == nil {
			break
		}
		require.ErrorIs(t, err, xpath3.ErrOpLimit, expr)
		lo = hi + 1
		hi *= 2
	}
	for lo < hi {
		mid := lo + (hi-lo)/2
		_, err := evalWith(t, eval.OpLimit(mid), node, expr)
		if err == nil {
			hi = mid
			continue
		}
		require.ErrorIs(t, err, xpath3.ErrOpLimit, expr)
		lo = mid + 1
	}
	return hi
}

// TestNodeListConsumersAllocate checks that the node-list consumers do not
// create a node item per node: on a document of 2,000 elements each fast
// form allocates at least 1,000 fewer times than its reference form. It
// measures allocations, so it does not run in parallel.
func TestNodeListConsumersAllocate(t *testing.T) {
	var b strings.Builder
	b.WriteString("<r>")
	for i := range 2000 {
		fmt.Fprintf(&b, `<b id="%d"/>`, i)
	}
	b.WriteString("<c/></r>")
	doc := mustParseXML(t, b.String())
	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions)
	for _, tmpl := range []string{
		"«//b | //c»", "«//b» | «//c»", "count(«//b | //c»)", "(«//b | //c»)[1]", "«//b» ! 1",
		"/r[«.//b»]", "if («//b») then 1 else 2", "«//b»/c", "«//b» except «//c»",
	} {
		fast, ref := nodeListForms(tmpl)
		fastExpr := xpath3.NewCompiler().MustCompile(fast)
		refExpr := xpath3.NewCompiler().MustCompile(ref)
		fastAllocs := testing.AllocsPerRun(5, func() {
			_, err := eval.Evaluate(t.Context(), fastExpr, doc)
			require.NoError(t, err)
		})
		refAllocs := testing.AllocsPerRun(5, func() {
			_, err := eval.Evaluate(t.Context(), refExpr, doc)
			require.NoError(t, err)
		})
		require.Less(t, fastAllocs+1000, refAllocs, fast)
	}
}
