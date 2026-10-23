package schematron_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/heliumtest"
	"github.com/lestrrat-go/helium/schematron"
	"github.com/stretchr/testify/require"
)

type cancelValidationHandler struct {
	cancel context.CancelFunc
	calls  int
}

func (h *cancelValidationHandler) Handle(context.Context, error) {
	h.calls++
	h.cancel()
}

func TestValidateCancellation(t *testing.T) {
	t.Parallel()
	for _, binding := range []string{testBindingXSLT, testBindingXSLT3} {
		t.Run(binding, func(t *testing.T) {
			src := strings.Replace(schemaSrc(binding, `<assert test="false()">invalid</assert>`),
				`context="root"`, `context="item"`, 1)
			schema, diagnostics := compileTestSchema(t, src)
			require.Empty(t, diagnostics)
			doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root>`+strings.Repeat(`<item/>`, 64)+`</root>`))
			require.NoError(t, err)
			validator := schematron.NewValidator(schema)
			for _, cause := range []error{context.Canceled, context.DeadlineExceeded} {
				ctx := heliumtest.NewPollContext(t.Context(), 1, cause)
				require.ErrorIs(t, validator.Validate(ctx, doc), cause)
			}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			handler := &cancelValidationHandler{cancel: cancel}
			require.ErrorIs(t, validator.ErrorHandler(handler).Validate(ctx, doc), context.Canceled)
			require.Equal(t, 1, handler.calls)
		})
	}
}

func TestValidateDiagnosticPaths(t *testing.T) {
	t.Parallel()
	schema, diagnostics := compileTestSchema(t, schemaSrc(testBindingXSLT, `<assert test="false()">invalid</assert>`))
	require.Empty(t, diagnostics)
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root/>`))
	require.NoError(t, err)
	collected, err := validateAndCollect(t, schema, doc)
	require.ErrorIs(t, err, schematron.ErrValidationFailed)
	require.Len(t, collected, 1)
	require.Equal(t, "/root", collected[0].Path)

	src := strings.Replace(schemaSrc(testBindingXSLT3, `<assert test="false()">invalid</assert>`),
		`context="root"`, `context="item | item/@id"`, 1)
	schema, diagnostics = compileTestSchema(t, src)
	require.Empty(t, diagnostics)
	doc, err = helium.NewParser().Parse(t.Context(), []byte(`<root><item id="a"/><other/><item id="b"/></root>`))
	require.NoError(t, err)
	for _, count := range []int{2, 3} {
		if count == 3 {
			n, createErr := doc.CreateElement("item")
			require.NoError(t, createErr)
			require.NoError(t, doc.DocumentElement().AddChild(n))
		}
		collected, err = validateAndCollect(t, schema, doc)
		require.ErrorIs(t, err, schematron.ErrValidationFailed)
		require.Len(t, collected, count+2)
		var paths []string
		for _, diagnostic := range collected {
			paths = append(paths, diagnostic.Path)
		}
		require.Contains(t, paths, "/root/item[1]/@id")
		require.Contains(t, paths, "/root/item[2]/@id")
		require.Contains(t, paths, fmt.Sprintf("/root/item[%d]", count))
	}
}

func TestDiagnosticPathSiblingWork(t *testing.T) {
	t.Parallel()
	const count = 128
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<root>`+strings.Repeat(`<item/>`, count)+`</root>`))
	require.NoError(t, err)
	var nodes []helium.Node
	for child := range helium.Children(doc.DocumentElement()) {
		nodes = append(nodes, child)
	}
	ctx := heliumtest.NewPollContext(t.Context(), 0, context.Canceled)
	paths, err := schematron.NodePathsForTesting(ctx, nodes)
	require.NoError(t, err)
	for i, path := range paths {
		require.Equal(t, fmt.Sprintf("/root/item[%d]", i+1), path)
	}
	require.Positive(t, ctx.Polls())
	require.LessOrEqual(t, ctx.Polls(), 6*count, "sibling lists must be indexed once per validation")
}
