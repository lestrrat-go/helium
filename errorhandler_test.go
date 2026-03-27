package helium_test

import (
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
	t.Parallel()

	helium.NilErrorHandler{}.Handle(t.Context(), errors.New("should not panic"))
}

func TestErrParseErrorImplementsErrorLeveler(t *testing.T) {
	t.Parallel()

	pe := helium.ErrParseError{
		Err:   errors.New("test"),
		Level: helium.ErrorLevelFatal,
	}
	require.Equal(t, helium.ErrorLevelFatal, pe.ErrorLevel())
}

func TestErrorCollector(t *testing.T) {
	t.Parallel()

	t.Run("collects all", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		ec := helium.NewErrorCollector(ctx, helium.ErrorLevelNone)

		want := []string{"one", "two", "three"}
		for _, msg := range want {
			ec.Handle(ctx, errors.New(msg))
		}

		require.NoError(t, ec.Close())

		errs := ec.Errors()
		require.Len(t, errs, len(want))
		for i, err := range errs {
			require.Equal(t, want[i], err.Error())
		}
	})

	t.Run("filters level", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name     string
			level    helium.ErrorLevel
			inputs   []error
			wantLen  int
			contains string
		}{
			{
				name:  "warnings only",
				level: helium.ErrorLevelWarning,
				inputs: []error{
					helium.ErrParseError{Err: errors.New("warn"), Level: helium.ErrorLevelWarning},
					helium.ErrParseError{Err: errors.New("err"), Level: helium.ErrorLevelError},
					helium.ErrParseError{Err: errors.New("fatal"), Level: helium.ErrorLevelFatal},
					errors.New("plain"),
				},
				wantLen: 2,
			},
			{
				name:  "fatal only",
				level: helium.ErrorLevelFatal,
				inputs: []error{
					helium.ErrParseError{Err: errors.New("warn"), Level: helium.ErrorLevelWarning},
					helium.ErrParseError{Err: errors.New("fatal"), Level: helium.ErrorLevelFatal},
					errors.New("plain"),
				},
				wantLen:  1,
				contains: "fatal",
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				ctx := t.Context()
				ec := helium.NewErrorCollector(ctx, tc.level)
				for _, in := range tc.inputs {
					ec.Handle(ctx, in)
				}

				require.NoError(t, ec.Close())

				errs := ec.Errors()
				require.Len(t, errs, tc.wantLen)
				if tc.contains != "" {
					require.Contains(t, errs[0].Error(), tc.contains)
				}
			})
		}
	})

	t.Run("returns copy", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		ec := helium.NewErrorCollector(ctx, helium.ErrorLevelNone)

		ec.Handle(ctx, errors.New("one"))
		require.NoError(t, ec.Close())

		errs1 := ec.Errors()
		errs2 := ec.Errors()
		require.Equal(t, errs1, errs2)

		errs1[0] = errors.New("modified")
		require.NotEqual(t, errs1[0], ec.Errors()[0])
	})

	t.Run("close idempotent", func(t *testing.T) {
		t.Parallel()

		ec := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		require.NoError(t, ec.Close())
		require.NoError(t, ec.Close())
	})

	t.Run("plain errors default to warning", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()

		ec := helium.NewErrorCollector(ctx, helium.ErrorLevelWarning)
		ec.Handle(ctx, errors.New("plain error without ErrorLeveler"))
		require.NoError(t, ec.Close())

		errs := ec.Errors()
		require.Len(t, errs, 1)
	})
}
