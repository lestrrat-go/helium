package xslt3_test

import (
	"testing"

	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

func TestNilStylesheetTransform(t *testing.T) {
	ctx := t.Context()
	var ss *xslt3.Stylesheet
	require.NotPanics(t, func() {
		_, err := ss.Transform(nil).Do(ctx)
		require.Error(t, err)
	})
}

func TestNilStylesheetApplyTemplates(t *testing.T) {
	ctx := t.Context()
	var ss *xslt3.Stylesheet
	require.NotPanics(t, func() {
		_, err := ss.ApplyTemplates(nil).Do(ctx)
		require.Error(t, err)
	})
}

func TestNilStylesheetCallTemplate(t *testing.T) {
	ctx := t.Context()
	var ss *xslt3.Stylesheet
	require.NotPanics(t, func() {
		_, err := ss.CallTemplate("x").Do(ctx)
		require.Error(t, err)
	})
}

func TestNilStylesheetCallFunction(t *testing.T) {
	ctx := t.Context()
	var ss *xslt3.Stylesheet
	require.NotPanics(t, func() {
		_, err := ss.CallFunction("x").Do(ctx)
		require.Error(t, err)
	})
}

func TestNilStylesheetSerialize(t *testing.T) {
	ctx := t.Context()
	var ss *xslt3.Stylesheet
	require.NotPanics(t, func() {
		_, err := ss.Transform(nil).Serialize(ctx)
		require.Error(t, err)
	})
}

func TestNilStylesheetWriteTo(t *testing.T) {
	ctx := t.Context()
	var ss *xslt3.Stylesheet
	require.NotPanics(t, func() {
		var buf []byte
		err := ss.Transform(nil).WriteTo(ctx, nil)
		_ = buf
		require.Error(t, err)
	})
}

func TestCompilerCompileNilDoc(t *testing.T) {
	ctx := t.Context()
	require.NotPanics(t, func() {
		_, err := xslt3.NewCompiler().Compile(ctx, nil)
		require.Error(t, err)
	})
}

func TestCompileStylesheetNilDoc(t *testing.T) {
	ctx := t.Context()
	require.NotPanics(t, func() {
		_, err := xslt3.CompileStylesheet(ctx, nil)
		require.Error(t, err)
	})
}

func TestPackageLevelTransformNilStylesheet(t *testing.T) {
	ctx := t.Context()
	require.NotPanics(t, func() {
		_, err := xslt3.Transform(ctx, nil, nil)
		require.Error(t, err)
	})
}

func TestZeroInvocationDo(t *testing.T) {
	ctx := t.Context()
	var inv xslt3.Invocation
	require.NotPanics(t, func() {
		_, err := inv.Do(ctx)
		require.Error(t, err)
	})
}

func TestZeroInvocationSerialize(t *testing.T) {
	ctx := t.Context()
	var inv xslt3.Invocation
	require.NotPanics(t, func() {
		_, err := inv.Serialize(ctx)
		require.Error(t, err)
	})
}
