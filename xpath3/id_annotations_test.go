package xpath3_test

import (
	"context"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/stretchr/testify/require"
)

type idSubtypeDecls struct{}

func (idSubtypeDecls) LookupSchemaElement(local, ns string) (string, bool) {
	return "", false
}

func (idSubtypeDecls) LookupSchemaAttribute(local, ns string) (string, bool) {
	return "", false
}

func (idSubtypeDecls) LookupSchemaType(local, ns string) (string, bool) {
	if local == "myID" && ns == "urn:test" {
		return xpath3.TypeID, true
	}
	return "", false
}

func (idSubtypeDecls) IsSubtypeOf(typeName, baseTypeName string) bool {
	return typeName == xpath3.QAnnotation("urn:test", "myID") && baseTypeName == xpath3.TypeID
}

func (idSubtypeDecls) ValidateCast(_ context.Context, value, typeName string) error {
	return nil
}

func (idSubtypeDecls) ValidateCastWithNS(_ context.Context, value, typeName string, nsMap map[string]string) error {
	return nil
}

func (idSubtypeDecls) ListItemType(typeName string) (string, bool) {
	return "", false
}
func (idSubtypeDecls) UnionMemberTypes(typeName string) []string {
	return nil
}

func TestFnIDUsesTypeAnnotationsForIDSubtype(t *testing.T) {
	doc := mustParseXML(t, `<root id="alpha"/>`)
	attr := doc.DocumentElement().Attributes()[0]

	eval := xpath3.NewEvaluator(xpath3.DefaultEvaluatorOptions).
		TypeAnnotations(map[helium.Node]string{
			attr: xpath3.QAnnotation("urn:test", "myID"),
		}).
		SchemaDeclarations(idSubtypeDecls{})

	seq := evalExprWithEval(t, eval, doc, `id("alpha")/name()`)
	require.Len(t, seq, 1)

	av, ok := seq.Get(0).(xpath3.AtomicValue)
	require.True(t, ok)
	require.Equal(t, "root", av.StringVal())
}
