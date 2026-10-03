package xmldsig1

import (
	"bytes"
	"context"
	"fmt"
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

const (
	xpathHereExpr = "here()"
	xpathTrueExpr = "true()"
)

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
		{
			name: "undefined variable",
			step: transformStep{
				algorithm: TransformXPath,
				xpathExpr: "$missing",
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
	t.Run("initial octets", func(t *testing.T) {
		hereDoc := parseTransformTestDoc(t, `<XPath/>`)
		transformer := &pipelineRecordingTransformer{outputs: [][]byte{[]byte(`<out/>`)}}
		runtime := transformRuntime{
			parser:          helium.NewParser(),
			xsltTransformer: transformer,
			external:        true,
		}
		steps := []transformStep{
			{algorithm: TransformXSLT, stylesheet: []byte("style")},
			{algorithm: TransformXPath, xpathExpr: xpathHereExpr, xpathHere: hereDoc.DocumentElement()},
		}

		_, err := externalReferenceDigestInput(t.Context(), []byte("raw"), steps, runtime)
		require.ErrorIs(t, err, ErrHereUnavailable)
		require.Empty(t, transformer.snapshot(), "here() must be rejected before an earlier transform callback runs")
	})

	t.Run("earlier octet boundary", func(t *testing.T) {
		doc := parseTransformTestDoc(t, `<root><XPath/></root>`)
		hereNode := findLocal(doc.DocumentElement(), "XPath")
		require.NotNil(t, hereNode)
		transformer := &pipelineRecordingTransformer{outputs: [][]byte{[]byte(`<out/>`)}}
		runtime := transformRuntime{
			parser:          helium.NewParser(),
			xsltTransformer: transformer,
		}
		initial := newReferenceNodeSetValue(doc, doc.DocumentElement(), nil, false, false, nil)
		steps := []transformStep{
			{algorithm: TransformXSLT, stylesheet: []byte("style")},
			{algorithm: TransformXPath, xpathExpr: xpathHereExpr, xpathHere: hereNode},
		}

		_, err := executeTransformPipeline(t.Context(), runtime, initial, steps, nil)
		require.ErrorIs(t, err, ErrHereUnavailable)
		require.Empty(t, transformer.snapshot(), "here() must be rejected before the octet-producing callback runs")
	})
}

func TestExpressionReferencesHere(t *testing.T) {
	cases := map[string]bool{
		xpathHereExpr:                    true,
		"here \n\t()":                    true,
		"boolean(here()/parent::node())": true,
		`contains("here()", "here")`:     false,
		"somehere()":                     false,
		"ext:here()":                     false,
	}
	for expr, want := range cases {
		t.Run(expr, func(t *testing.T) {
			require.Equal(t, want, expressionReferencesHere(expr))
		})
	}
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
			out, err := executeTransformPipeline(t.Context(), runtime, initial, steps, nil)
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
		}, nil)
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

// TestExecuteTransformPipelineStreamsSameOctets checks that writing the
// pipeline's output to a writer, which streams a final canonicalization, gives
// exactly the octets (or the error) the buffered result does.
func TestExecuteTransformPipelineStreamsSameOctets(t *testing.T) {
	const input = `<!-- lead --><r xmlns="urn:d" xmlns:a="urn:a" xml:lang="en"><a:p Id="t" a:x="1">text &amp; more<!-- c -->` +
		`<b64>PHg+eTwveD4=</b64><ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#"><ds:SignedInfo/></ds:Signature></a:p></r>`
	cases := map[string]struct {
		input    string
		wholeDoc bool
		steps    []transformStep
		wantErr  bool
	}{
		"implicit final c14n":       {input: input, wholeDoc: true},
		"c14n only":                 {input: input, wholeDoc: true, steps: []transformStep{{algorithm: C14N10Comments}}},
		"enveloped then exc c14n":   {input: input, wholeDoc: true, steps: []transformStep{{algorithm: TransformEnvelopedSignature}, {algorithm: ExcC14N10Comments}}},
		"enveloped id exc prefixes": {input: input, steps: []transformStep{{algorithm: TransformEnvelopedSignature}, {algorithm: ExcC14N10, prefixes: []string{"a", "#default"}}}},
		"xpath then c14n 1.1":       {input: input, steps: []transformStep{{algorithm: TransformXPath, xpathExpr: xpathTrueExpr}, {algorithm: C14N11URI}}},
		"c14n then c14n":            {input: input, steps: []transformStep{{algorithm: ExcC14N10}, {algorithm: C14N10}}},
		"xpath last":                {input: input, wholeDoc: true, steps: []transformStep{{algorithm: TransformXPath, xpathExpr: "not(self::comment())"}}},
		"enveloped last":            {input: input, steps: []transformStep{{algorithm: TransformEnvelopedSignature}}},
		"base64 octets":             {input: `<r Id="t"><b>PHg+eTwveD4=</b></r>`, steps: []transformStep{{algorithm: TransformBase64}}},
		"relative namespace error":  {input: `<r><x xmlns:rel="rel/uri"/><s Id="t"/></r>`, steps: []transformStep{{algorithm: C14N10}}, wantErr: true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			buffered, bufferedErr := runTransformPipelineForTest(t, tc.input, tc.wholeDoc, tc.steps, false)
			streamed, streamedErr := runTransformPipelineForTest(t, tc.input, tc.wholeDoc, tc.steps, true)
			require.Equal(t, fmt.Sprint(bufferedErr), fmt.Sprint(streamedErr))
			require.Equal(t, string(buffered), string(streamed))
			require.Equal(t, tc.wantErr, streamedErr != nil)
			require.Equal(t, tc.wantErr, len(streamed) == 0)
		})
	}
}

// runTransformPipelineForTest runs steps over a fresh parse of input, selecting
// the whole document or the element with Id="t", and returns the octets either
// as executeTransformPipeline's result or as what it wrote to a writer.
func runTransformPipelineForTest(t *testing.T, input string, wholeDoc bool, steps []transformStep, stream bool) ([]byte, error) {
	t.Helper()
	doc := parseTransformTestDoc(t, input)
	target := doc.DocumentElement()
	if !wholeDoc {
		target = findLocalWithID(doc, "t")
		require.NotNil(t, target)
	}
	sig := findSig(doc.DocumentElement())
	runtime := transformRuntime{parser: helium.NewParser(), signature: sig, allowEnveloped: true}
	initial := newReferenceNodeSetValue(doc, target, sig, wholeDoc, true, nil)
	if !stream {
		return executeTransformPipeline(t.Context(), runtime, initial, steps, nil)
	}
	var buf bytes.Buffer
	out, err := executeTransformPipeline(t.Context(), runtime, initial, steps, &buf)
	require.Nil(t, out)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// findLocalWithID returns the first element under n whose Id attribute is id.
func findLocalWithID(n helium.Node, id string) *helium.Element {
	for c := range helium.Children(n) {
		if e, ok := helium.AsNode[*helium.Element](c); ok {
			if v, _ := e.GetAttribute("Id"); v == id {
				return e
			}
		}
		if found := findLocalWithID(c, id); found != nil {
			return found
		}
	}
	return nil
}
