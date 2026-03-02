package helium_test

import (
	"context"
	"errors"
	"io"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/sink"
	"github.com/stretchr/testify/require"
)

// compile-time assertions
var (
	_ helium.ErrorHandler = helium.NilErrorHandler{}
	_ helium.ErrorHandler = (*helium.ErrorCollector)(nil)
	_ io.Closer           = (*helium.ErrorCollector)(nil)
	_ helium.ErrorHandler = (*sink.Sink[error])(nil)
	_ helium.ErrorLeveler = helium.ErrParseError{}
)

func TestNilErrorHandler(t *testing.T) {
	var h helium.NilErrorHandler
	h.Handle(context.Background(), errors.New("should not panic"))
}

func TestErrorCollectorCollectsAll(t *testing.T) {
	ctx := context.Background()
	ec := helium.NewErrorCollector(ctx, helium.ErrorLevelNone)

	ec.Handle(ctx, errors.New("one"))
	ec.Handle(ctx, errors.New("two"))
	ec.Handle(ctx, errors.New("three"))

	require.NoError(t, ec.Close())

	errs := ec.Errors()
	require.Len(t, errs, 3)
	require.Equal(t, "one", errs[0].Error())
	require.Equal(t, "two", errs[1].Error())
	require.Equal(t, "three", errs[2].Error())
}

func TestErrorCollectorFiltersLevel(t *testing.T) {
	ctx := context.Background()

	t.Run("warnings only", func(t *testing.T) {
		ec := helium.NewErrorCollector(ctx, helium.ErrorLevelWarning)

		ec.Handle(ctx, helium.ErrParseError{Err: errors.New("warn"), Level: helium.ErrorLevelWarning})
		ec.Handle(ctx, helium.ErrParseError{Err: errors.New("err"), Level: helium.ErrorLevelError})
		ec.Handle(ctx, helium.ErrParseError{Err: errors.New("fatal"), Level: helium.ErrorLevelFatal})
		ec.Handle(ctx, errors.New("plain"))

		require.NoError(t, ec.Close())

		errs := ec.Errors()
		require.Len(t, errs, 2)
	})

	t.Run("fatal only", func(t *testing.T) {
		ec := helium.NewErrorCollector(ctx, helium.ErrorLevelFatal)

		ec.Handle(ctx, helium.ErrParseError{Err: errors.New("warn"), Level: helium.ErrorLevelWarning})
		ec.Handle(ctx, helium.ErrParseError{Err: errors.New("fatal"), Level: helium.ErrorLevelFatal})
		ec.Handle(ctx, errors.New("plain"))

		require.NoError(t, ec.Close())

		errs := ec.Errors()
		require.Len(t, errs, 1)
		require.Contains(t, errs[0].Error(), "fatal")
	})
}

func TestErrorCollectorErrorsReturnsCopy(t *testing.T) {
	ctx := context.Background()
	ec := helium.NewErrorCollector(ctx, helium.ErrorLevelNone)

	ec.Handle(ctx, errors.New("one"))
	require.NoError(t, ec.Close())

	errs1 := ec.Errors()
	errs2 := ec.Errors()
	require.Equal(t, errs1, errs2)

	errs1[0] = errors.New("modified")
	require.NotEqual(t, errs1[0], ec.Errors()[0])
}

func TestErrorCollectorCloseMultipleTimes(t *testing.T) {
	ctx := context.Background()
	ec := helium.NewErrorCollector(ctx, helium.ErrorLevelNone)
	require.NoError(t, ec.Close())
	require.NoError(t, ec.Close())
}

func TestErrParseErrorImplementsErrorLeveler(t *testing.T) {
	pe := helium.ErrParseError{
		Err:   errors.New("test"),
		Level: helium.ErrorLevelFatal,
	}
	require.Equal(t, helium.ErrorLevelFatal, pe.ErrorLevel())
}

func TestErrorCollectorPlainErrorsDefaultToWarning(t *testing.T) {
	ctx := context.Background()

	ec := helium.NewErrorCollector(ctx, helium.ErrorLevelWarning)
	ec.Handle(ctx, errors.New("plain error without ErrorLeveler"))
	require.NoError(t, ec.Close())

	errs := ec.Errors()
	require.Len(t, errs, 1)
}
