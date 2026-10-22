package xsd_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xsd"
)

func TestCompileReturnsContextCancellation(t *testing.T) {
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"/>`))
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err = xsd.NewCompiler().Compile(ctx, doc)
	require.ErrorIs(t, err, context.Canceled)
}

func TestCompileStopsAfterCancellationDuringComponentWalk(t *testing.T) {
	doc, err := helium.NewParser().Parse(t.Context(), []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		<xs:unknown/>
		<xs:element name="later" type="xs:string"/>
	</xs:schema>`))
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	handler := &cancelOnError{cancel: cancel}
	_, err = xsd.NewCompiler().ErrorHandler(handler).Compile(ctx, doc)
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 1, handler.calls)
}

func TestValidateReturnsContextCancellation(t *testing.T) {
	schemaDoc, err := helium.NewParser().Parse(t.Context(), []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		<xs:element name="root" type="xs:string"/>
	</xs:schema>`))
	require.NoError(t, err)
	schema, err := xsd.NewCompiler().Compile(t.Context(), schemaDoc)
	require.NoError(t, err)
	instance, err := helium.NewParser().Parse(t.Context(), []byte(`<root>value</root>`))
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err = xsd.NewValidator(schema).Validate(ctx, instance)
	require.ErrorIs(t, err, context.Canceled)
}

func TestValidateReturnsCancellationRaisedDuringValidation(t *testing.T) {
	schemaDoc, err := helium.NewParser().Parse(t.Context(), []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		<xs:element name="root" type="xs:int"/>
	</xs:schema>`))
	require.NoError(t, err)
	schema, err := xsd.NewCompiler().Compile(t.Context(), schemaDoc)
	require.NoError(t, err)
	instance, err := helium.NewParser().Parse(t.Context(), []byte(`<root>invalid</root>`))
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	handler := &cancelOnError{cancel: cancel}
	err = xsd.NewValidator(schema).ErrorHandler(handler).Validate(ctx, instance)
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 1, handler.calls)
}

type cancelOnError struct {
	cancel context.CancelFunc
	calls  int
}

func (h *cancelOnError) Handle(context.Context, error) {
	h.calls++
	h.cancel()
}
