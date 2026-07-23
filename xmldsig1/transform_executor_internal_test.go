package xmldsig1

import (
	"context"
	"slices"
	"sync"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

type transformCall struct {
	stylesheet []byte
	input      []byte
}

const xpathTrueExpr = "true()"

type pipelineRecordingTransformer struct {
	mu      sync.Mutex
	outputs [][]byte
	cancel  context.CancelFunc
	calls   []transformCall
}

func (r *pipelineRecordingTransformer) TransformXSLT(_ context.Context, stylesheet, input []byte) ([]byte, error) {
	r.mu.Lock()
	index := len(r.calls)
	r.calls = append(r.calls, transformCall{
		stylesheet: slices.Clone(stylesheet),
		input:      slices.Clone(input),
	})
	var output []byte
	if index < len(r.outputs) {
		output = slices.Clone(r.outputs[index])
	} else {
		output = slices.Clone(input)
	}
	cancel := r.cancel
	r.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	return output, nil
}

func (r *pipelineRecordingTransformer) snapshot() []transformCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.calls)
}

func parseTransformTestDoc(t *testing.T, input string) *helium.Document {
	t.Helper()
	doc, err := helium.NewParser().Parse(t.Context(), []byte(input))
	require.NoError(t, err)
	return doc
}

func TestExecuteTransformPipelineXSLTOrdering(t *testing.T) {
	transformer := &pipelineRecordingTransformer{
		outputs: [][]byte{[]byte("first"), []byte("second")},
	}
	runtime := transformRuntime{
		parser:          helium.NewParser(),
		xsltTransformer: transformer,
		external:        true,
	}
	steps := []transformStep{
		{algorithm: TransformXSLT, stylesheet: []byte("style-1")},
		{algorithm: TransformXSLT, stylesheet: []byte("style-2")},
	}

	out, err := externalReferenceDigestInput(t.Context(), []byte("initial"), steps, runtime)
	require.NoError(t, err)
	require.Equal(t, []byte("second"), out)
	calls := transformer.snapshot()
	require.Len(t, calls, 2)
	require.Equal(t, []byte("initial"), calls[0].input)
	require.Equal(t, []byte("first"), calls[1].input)
	require.Equal(t, []byte("style-1"), calls[0].stylesheet)
	require.Equal(t, []byte("style-2"), calls[1].stylesheet)
}

func TestExecuteTransformPipelineReparse(t *testing.T) {
	t.Run("XSLT output feeds XPath and final implicit c14n", func(t *testing.T) {
		transformer := &pipelineRecordingTransformer{
			outputs: [][]byte{[]byte(`<out><keep>value</keep><drop>gone</drop></out>`)},
		}
		runtime := transformRuntime{
			parser:          helium.NewParser(),
			xsltTransformer: transformer,
			external:        true,
		}
		steps := []transformStep{
			{algorithm: TransformXSLT, stylesheet: []byte("style")},
			{algorithm: TransformXPath, xpathExpr: "not(ancestor-or-self::drop)"},
		}

		out, err := externalReferenceDigestInput(t.Context(), []byte("raw input"), steps, runtime)
		require.NoError(t, err)
		require.Equal(t, `<out><keep>value</keep></out>`, string(out))
	})

	t.Run("Base64 output feeds XPath", func(t *testing.T) {
		runtime := transformRuntime{parser: helium.NewParser(), external: true}
		steps := []transformStep{
			{algorithm: TransformBase64},
			{algorithm: TransformXPath, xpathExpr: xpathTrueExpr},
		}
		out, err := externalReferenceDigestInput(t.Context(), []byte("PHJvb3Q+PHY+eDwvdj48L3Jvb3Q+"), steps, runtime)
		require.NoError(t, err)
		require.Equal(t, `<root><v>x</v></root>`, string(out))
	})

	t.Run("c14n output feeds a second c14n", func(t *testing.T) {
		runtime := transformRuntime{parser: helium.NewParser(), external: true}
		steps := []transformStep{
			{algorithm: C14N11URI},
			{algorithm: C14N10},
		}
		out, err := externalReferenceDigestInput(t.Context(), []byte(`<root><v/></root>`), steps, runtime)
		require.NoError(t, err)
		require.Equal(t, `<root><v></v></root>`, string(out))
	})
}

func TestExecuteTransformPipelineStaticValidationRunsFirst(t *testing.T) {
	transformer := &pipelineRecordingTransformer{outputs: [][]byte{[]byte("unused")}}
	runtime := transformRuntime{
		parser:          helium.NewParser(),
		xsltTransformer: transformer,
		external:        true,
	}
	steps := []transformStep{
		{algorithm: TransformXSLT, stylesheet: []byte("style")},
		{algorithm: "urn:example:unsupported"},
	}

	_, err := externalReferenceDigestInput(t.Context(), []byte("input"), steps, runtime)
	require.ErrorIs(t, err, ErrUnsupportedTransform)
	require.Empty(t, transformer.snapshot(), "an earlier injected transform must not run before static validation finishes")
}

func TestExecuteTransformPipelineMalformedXPathValidationRunsFirst(t *testing.T) {
	transformer := &pipelineRecordingTransformer{outputs: [][]byte{[]byte("unused")}}
	runtime := transformRuntime{
		parser:          helium.NewParser(),
		xsltTransformer: transformer,
		external:        true,
	}
	steps := []transformStep{
		{algorithm: TransformXSLT, stylesheet: []byte("style")},
		{algorithm: TransformXPath, xpathExpr: "["},
	}

	_, err := externalReferenceDigestInput(t.Context(), []byte("input"), steps, runtime)
	require.ErrorIs(t, err, ErrUnsupportedTransform)
	require.Empty(t, transformer.snapshot(), "an earlier injected transform must not run before every XPath expression is validated")
}

func TestExecuteTransformPipelineXPathStaticValidationRunsFirst(t *testing.T) {
	tests := []struct {
		name string
		step transformStep
	}{
		{
			name: "bound prefix with unknown function",
			step: transformStep{
				algorithm: TransformXPath,
				xpathExpr: "ext:missing()",
				xpathNS:   map[string]string{"ext": "urn:ext"},
			},
		},
		{
			name: "unbound name test prefix",
			step: transformStep{
				algorithm: TransformXPath,
				xpathExpr: "not(self::missing:secret)",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			transformer := &pipelineRecordingTransformer{outputs: [][]byte{[]byte("unused")}}
			runtime := transformRuntime{
				parser:          helium.NewParser(),
				xsltTransformer: transformer,
				external:        true,
			}
			steps := []transformStep{
				{algorithm: TransformXSLT, stylesheet: []byte("style")},
				test.step,
			}

			out, err := externalReferenceDigestInput(t.Context(), []byte(`<root xmlns:missing="urn:missing"/>`), steps, runtime)
			require.ErrorIs(t, err, ErrUnsupportedTransform)
			require.Nil(t, out)
			require.Empty(t, transformer.snapshot(), "an earlier injected transform must not run before XPath static validation finishes")
		})
	}
}

func TestExecuteTransformPipelineParseErrors(t *testing.T) {
	t.Run("initial external parse keeps ErrReferenceNotFound", func(t *testing.T) {
		runtime := transformRuntime{parser: helium.NewParser(), external: true}
		_, err := externalReferenceDigestInput(t.Context(), []byte("not XML"), []transformStep{
			{algorithm: TransformXPath, xpathExpr: xpathTrueExpr},
		}, runtime)
		require.ErrorIs(t, err, ErrReferenceNotFound)
	})

	t.Run("intermediate parse identifies producer and consumer", func(t *testing.T) {
		transformer := &pipelineRecordingTransformer{outputs: [][]byte{[]byte("not XML")}}
		runtime := transformRuntime{
			parser:          helium.NewParser(),
			xsltTransformer: transformer,
			external:        true,
		}
		_, err := externalReferenceDigestInput(t.Context(), []byte("raw"), []transformStep{
			{algorithm: TransformXSLT, stylesheet: []byte("style")},
			{algorithm: TransformXPath, xpathExpr: xpathTrueExpr},
		}, runtime)
		require.ErrorIs(t, err, ErrUnsupportedTransform)
		require.Contains(t, err.Error(), "transform 0")
		require.Contains(t, err.Error(), "transform 1")
	})
}

func TestExecuteTransformPipelineHereAfterReparse(t *testing.T) {
	hereDoc := parseTransformTestDoc(t, `<XPath/>`)
	transformer := &pipelineRecordingTransformer{outputs: [][]byte{[]byte(`<out/>`)}}
	runtime := transformRuntime{
		parser:          helium.NewParser(),
		xsltTransformer: transformer,
		external:        true,
	}
	steps := []transformStep{
		{algorithm: TransformXSLT, stylesheet: []byte("style")},
		{algorithm: TransformXPath, xpathExpr: "here()", xpathHere: hereDoc.DocumentElement()},
	}

	_, err := externalReferenceDigestInput(t.Context(), []byte("raw"), steps, runtime)
	require.ErrorIs(t, err, ErrHereUnavailable)
}

func TestExecuteTransformPipelineEnvelopedOrder(t *testing.T) {
	const input = `<root xmlns:ds="http://www.w3.org/2000/09/xmldsig#"><data>keep</data><ds:Signature><ds:Object>remove</ds:Object></ds:Signature></root>`
	for name, steps := range map[string][]transformStep{
		"XPath then enveloped": {
			{algorithm: TransformXPath, xpathExpr: xpathTrueExpr},
			{algorithm: TransformEnvelopedSignature},
			{algorithm: C14N10},
		},
		"enveloped then XPath": {
			{algorithm: TransformEnvelopedSignature},
			{algorithm: TransformXPath, xpathExpr: xpathTrueExpr},
			{algorithm: C14N10},
		},
	} {
		t.Run(name, func(t *testing.T) {
			doc := parseTransformTestDoc(t, input)
			sig := findSig(doc.DocumentElement())
			require.NotNil(t, sig)
			runtime := transformRuntime{
				parser:         helium.NewParser(),
				signature:      sig,
				allowEnveloped: true,
			}
			initial := newReferenceNodeSetValue(doc, doc.DocumentElement(), sig, true, true, nil)
			out, err := executeTransformPipeline(t.Context(), runtime, initial, steps)
			require.NoError(t, err)
			require.Contains(t, string(out), "keep")
			require.NotContains(t, string(out), "Signature")
			require.NotContains(t, string(out), "remove")
		})
	}
}

func TestExecuteTransformPipelineEmptyValues(t *testing.T) {
	t.Run("empty octets remain octets", func(t *testing.T) {
		out, err := externalReferenceDigestInput(t.Context(), []byte{}, nil, transformRuntime{parser: helium.NewParser(), external: true})
		require.NoError(t, err)
		require.NotNil(t, out)
		require.Empty(t, out)
	})

	t.Run("empty node-set receives final implicit c14n", func(t *testing.T) {
		doc := parseTransformTestDoc(t, `<root/>`)
		initial := newReferenceNodeSetValue(doc, doc.DocumentElement(), nil, false, true, nil)
		out, err := executeTransformPipeline(t.Context(), transformRuntime{parser: helium.NewParser(), allowEnveloped: true}, initial, []transformStep{
			{algorithm: TransformXPath, xpathExpr: "false()"},
		})
		require.NoError(t, err)
		require.Empty(t, out)
	})
}

func TestExecuteTransformPipelineCancellationBetweenSteps(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	transformer := &pipelineRecordingTransformer{cancel: cancel}
	runtime := transformRuntime{
		parser:          helium.NewParser(),
		xsltTransformer: transformer,
		external:        true,
	}
	steps := []transformStep{
		{algorithm: TransformXSLT, stylesheet: []byte("style-1")},
		{algorithm: TransformXSLT, stylesheet: []byte("style-2")},
	}

	_, err := externalReferenceDigestInput(ctx, []byte("input"), steps, runtime)
	require.ErrorIs(t, err, context.Canceled)
	require.Len(t, transformer.snapshot(), 1)
}
