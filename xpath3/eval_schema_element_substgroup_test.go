package xpath3_test

import (
	"context"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

const sgTestNS = "urn:t"

// substGroupDecls declares a head element {urn:t}h of type Q{urn:t}HT, a
// substitution-group member {urn:t}m (type Q{urn:t}MT, derived from HT), and a
// NON-member {urn:t}e (type Q{urn:t}ET, ALSO derived from HT). Type derivation
// and substitution-group membership are DELIBERATELY decoupled: e derives from
// HT but is not a member of h's substitution group. schema-element(H) must key
// on substitution-group membership, not type derivation, so both the step
// node-test path (eval_path.go) and the sequence-type / instance-of path
// (eval_types.go) must reject e and accept m — and must agree with each other.
type substGroupDecls struct{}

func (substGroupDecls) LookupSchemaElement(local, ns string) (string, bool) {
	if ns != sgTestNS {
		return "", false
	}
	switch local {
	case "h":
		return "Q{urn:t}HT", true
	case "m":
		return "Q{urn:t}MT", true
	case "e":
		return "Q{urn:t}ET", true
	}
	return "", false
}
func (substGroupDecls) LookupSchemaAttribute(local, ns string) (string, bool) { return "", false }
func (substGroupDecls) LookupSchemaType(local, ns string) (string, bool)      { return "", false }
func (substGroupDecls) IsSubtypeOf(typeName, baseTypeName string) bool {
	if typeName == baseTypeName {
		return true
	}
	// Both MT and ET derive from HT.
	switch typeName {
	case "Q{urn:t}MT", "Q{urn:t}ET":
		return baseTypeName == "Q{urn:t}HT"
	}
	return false
}
func (substGroupDecls) ValidateCast(_ context.Context, value, typeName string) error { return nil }
func (substGroupDecls) ValidateCastWithNS(_ context.Context, value, typeName string, nsMap map[string]string) error {
	return nil
}
func (substGroupDecls) ListItemType(typeName string) (string, bool) { return "", false }
func (substGroupDecls) UnionMemberTypes(typeName string) []string   { return nil }
func (substGroupDecls) IsSubstitutionGroupMember(memberLocal, memberNS, headLocal, headNS string) bool {
	// Only m is substitutable for h; e is not — even though ET derives from HT.
	return headNS == sgTestNS && headLocal == "h" && memberNS == sgTestNS && memberLocal == "m"
}

// TestSchemaElementStepAndInstanceOfAgree verifies that the two schema-element()
// matchers — the step node-test (eval_path.go) and the sequence-type instance-of
// path (eval_types.go) — give the SAME answer, keying on substitution-group
// membership rather than type derivation. The member m is selected/true by both;
// the non-member e, whose type still derives from the head's type, is
// rejected/false by both.
func TestSchemaElementStepAndInstanceOfAgree(t *testing.T) {
	t.Parallel()

	doc := `<root xmlns:t="` + sgTestNS + `"><t:m/><t:e/></root>`

	build := func(t *testing.T) (*helium.Document, map[helium.Node]string) {
		t.Helper()
		parsed, err := helium.NewParser().Parse(t.Context(), []byte(doc))
		require.NoError(t, err)
		annotations := map[helium.Node]string{}
		for c := range helium.Children(parsed.DocumentElement()) {
			elem, ok := helium.AsNode[*helium.Element](c)
			if !ok {
				continue
			}
			switch elem.LocalName() {
			case "m":
				annotations[elem] = "Q{urn:t}MT"
			case "e":
				annotations[elem] = "Q{urn:t}ET"
			}
		}
		require.Len(t, annotations, 2, "fixture must annotate both t:m and t:e")
		return parsed, annotations
	}

	newEval := func(annotations map[helium.Node]string) xpath3.Evaluator {
		return xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
			Namespaces(map[string]string{"t": sgTestNS}).
			TypeAnnotations(annotations).
			SchemaDeclarations(substGroupDecls{})
	}

	t.Run("step selects only the substitution-group member", func(t *testing.T) {
		t.Parallel()
		parsed, annotations := build(t)
		compiled, err := xpath3.NewCompiler().Compile(`//schema-element(t:h)`)
		require.NoError(t, err)
		r, err := newEval(annotations).Evaluate(t.Context(), compiled, parsed)
		require.NoError(t, err)
		nodes, err := r.Nodes()
		require.NoError(t, err)
		require.Len(t, nodes, 1, "only t:m (the substitution member) must be selected, not t:e")
		require.Equal(t, "m", xpath3NodeLocal(nodes[0]))
	})

	t.Run("instance of true for the member, false for the non-member", func(t *testing.T) {
		t.Parallel()
		parsed, annotations := build(t)

		memberOK := xpath3.NewCompiler()
		compiledMember, err := memberOK.Compile(`//t:m instance of schema-element(t:h)`)
		require.NoError(t, err)
		rm, err := newEval(annotations).Evaluate(t.Context(), compiledMember, parsed)
		require.NoError(t, err)
		bm, ok := rm.IsBoolean()
		require.True(t, ok)
		require.True(t, bm, "t:m is a substitution member of t:h, so instance of must be true")

		compiledNonMember, err := xpath3.NewCompiler().Compile(`//t:e instance of schema-element(t:h)`)
		require.NoError(t, err)
		re, err := newEval(annotations).Evaluate(t.Context(), compiledNonMember, parsed)
		require.NoError(t, err)
		be, ok := re.IsBoolean()
		require.True(t, ok)
		require.False(t, be,
			"t:e's type derives from t:h's type but t:e is NOT a substitution member, so instance of must be false — agreeing with the step path")
	})
}

func xpath3NodeLocal(n helium.Node) string {
	if e, ok := helium.AsNode[*helium.Element](n); ok {
		return e.LocalName()
	}
	return n.Name()
}
