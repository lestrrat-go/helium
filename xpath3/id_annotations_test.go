package xpath3_test

import (
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

func (idSubtypeDecls) ValidateCast(value, typeName string) error {
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

	ctx := xpath3.WithTypeAnnotations(t.Context(), map[helium.Node]string{
		attr: xpath3.QAnnotation("urn:test", "myID"),
	})
	ctx = xpath3.WithSchemaDeclarations(ctx, idSubtypeDecls{})

	seq := evalExprCtx(t, ctx, doc, `id("alpha")/name()`)
	require.Len(t, seq, 1)

	av, ok := seq[0].(xpath3.AtomicValue)
	require.True(t, ok)
	require.Equal(t, "root", av.StringVal())
}
