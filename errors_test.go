package helium_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/lestrrat-go/helium"

	"github.com/lestrrat-go/helium/sax"
	"github.com/stretchr/testify/require"
)

func TestErrParseError(t *testing.T) {
	t.Parallel()

	t.Run("Unwrap", func(t *testing.T) {
		t.Parallel()
		pe := helium.ErrParseError{
			Err:        helium.ErrSpaceRequired,
			Level:      helium.ErrorLevelError,
			Line:       "<foo>",
			LineNumber: 1,
			Column:     5,
		}
		require.True(t, errors.Is(pe, helium.ErrSpaceRequired))
		require.False(t, errors.Is(pe, helium.ErrEOF))
	})

	t.Run("As", func(t *testing.T) {
		t.Parallel()
		pe := helium.ErrParseError{
			Err:        helium.ErrSpaceRequired,
			File:       "test.xml",
			Level:      helium.ErrorLevelError,
			Line:       "<foo>",
			LineNumber: 3,
			Column:     10,
		}
		var wrapped error = pe

		var extracted helium.ErrParseError
		require.True(t, errors.As(wrapped, &extracted))
		require.Equal(t, "test.xml", extracted.File)
		require.Equal(t, helium.ErrorLevelError, extracted.Level)
		require.Equal(t, 3, extracted.LineNumber)
		require.Equal(t, 10, extracted.Column)
		require.Equal(t, "<foo>", extracted.Line)
	})

	t.Run("ErrorString", func(t *testing.T) {
		t.Parallel()

		t.Run("without file", func(t *testing.T) {
			t.Parallel()
			pe := helium.ErrParseError{
				Err:        helium.ErrSpaceRequired,
				Level:      helium.ErrorLevelError,
				Line:       "<foo>",
				LineNumber: 1,
				Column:     5,
			}
			msg := pe.Error()
			require.Contains(t, msg, "space required")
			require.Contains(t, msg, "line 1")
			require.Contains(t, msg, "column 5")
			require.False(t, strings.HasPrefix(msg, ":"))
		})

		t.Run("with file", func(t *testing.T) {
			t.Parallel()
			pe := helium.ErrParseError{
				Err:        helium.ErrSpaceRequired,
				File:       "test.xml",
				Level:      helium.ErrorLevelError,
				Line:       "<foo>",
				LineNumber: 1,
				Column:     5,
			}
			msg := pe.Error()
			require.True(t, strings.HasPrefix(msg, "test.xml: "))
		})
	})

	t.Run("Level", func(t *testing.T) {
		t.Parallel()
		_, err := helium.NewParser().Parse(t.Context(), []byte("<broken"))
		require.Error(t, err)

		var pe helium.ErrParseError
		require.True(t, errors.As(err, &pe))
		require.Equal(t, helium.ErrorLevelFatal, pe.Level)
	})

	t.Run("WarningLevel", func(t *testing.T) {
		t.Parallel()
		// XML with undefined entity reference in a non-standalone document
		// with an external subset. The parser emits a warning (not an error)
		// for undefined entities in this context.
		const input = `<?xml version="1.0"?>
<!DOCTYPE root SYSTEM "nonexistent.dtd">
<root>&undefined;</root>`

		s := sax.New()
		s.SetOnWarning(sax.WarningFunc(func(_ context.Context, err error) error {
			return errors.New("warning escalated")
		}))

		p := helium.NewParser().SAXHandler(s)
		_, err := p.Parse(t.Context(), []byte(input))
		require.Error(t, err)

		var pe helium.ErrParseError
		require.True(t, errors.As(err, &pe))
		require.Equal(t, helium.ErrorLevelWarning, pe.Level)
	})
}

func TestErrorDomain(t *testing.T) {
	t.Parallel()

	t.Run("default is parser", func(t *testing.T) {
		t.Parallel()
		var pe helium.ErrParseError
		require.Equal(t, helium.ErrorDomainParser, pe.Domain)
	})

	t.Run("namespace error", func(t *testing.T) {
		t.Parallel()
		const input = `<root xmlns:a="urn:a"><a:child xmlns:a="">text</a:child></root>`

		_, err := helium.NewParser().Parse(t.Context(), []byte(input))
		require.Error(t, err)

		var pe helium.ErrParseError
		require.True(t, errors.As(err, &pe))
		require.Equal(t, helium.ErrorDomainNamespace, pe.Domain)
	})
}

func TestErrDTDDupTokenFixed(t *testing.T) {
	t.Parallel()
	e := helium.ErrDTDDupToken{Name: "foo"}
	require.Contains(t, e.Error(), "standalone")
	require.NotContains(t, e.Error(), "standlone")
}

func TestErrorLevelConstants(t *testing.T) {
	t.Parallel()
	require.Equal(t, helium.ErrorLevel(0), helium.ErrorLevelNone)
	require.Equal(t, helium.ErrorLevel(1), helium.ErrorLevelWarning)
	require.Equal(t, helium.ErrorLevel(2), helium.ErrorLevelError)
	require.Equal(t, helium.ErrorLevel(3), helium.ErrorLevelFatal)

	require.True(t, helium.ErrorLevelNone < helium.ErrorLevelWarning)
	require.True(t, helium.ErrorLevelWarning < helium.ErrorLevelError)
	require.True(t, helium.ErrorLevelError < helium.ErrorLevelFatal)
}

func TestParseNoError(t *testing.T) {
	t.Parallel()
	// Malformed XML triggers SAX Error callback
	const input = `<?xml version="1.0"?><root><child>text</chld></root>`

	t.Run("SAX Error callback fires by default", func(t *testing.T) {
		t.Parallel()
		var called atomic.Int32
		s := sax.New()
		s.SetOnError(sax.ErrorFunc(func(_ context.Context, err error) error {
			called.Add(1)
			return nil
		}))

		p := helium.NewParser().SAXHandler(s)
		_, _ = p.Parse(t.Context(), []byte(input))
		require.Greater(t, called.Load(), int32(0), "SAX Error handler should be called")
	})

	t.Run("SuppressErrors suppresses SAX Error callback", func(t *testing.T) {
		t.Parallel()
		var called atomic.Int32
		s := sax.New()
		s.SetOnError(sax.ErrorFunc(func(_ context.Context, err error) error {
			called.Add(1)
			return nil
		}))

		p := helium.NewParser().SAXHandler(s).SuppressErrors(true)
		_, err := p.Parse(t.Context(), []byte(input))
		require.Error(t, err, "parse should still return error")
		require.Equal(t, int32(0), called.Load(), "SAX Error handler should NOT be called with SuppressErrors")
	})
}

func TestWarningLocationInfo(t *testing.T) {
	t.Parallel()
	// XML with external subset makes undeclared entity references a warning
	// rather than an error.
	const input = "<!DOCTYPE doc SYSTEM \"foo\">\n<doc>&undeclared;</doc>"

	t.Run("warning handler error wrapped with helium.ErrorLevelWarning", func(t *testing.T) {
		t.Parallel()
		handlerErr := fmt.Errorf("warning escalated")
		s := sax.New()
		s.SetOnWarning(sax.WarningFunc(func(_ context.Context, err error) error {
			return handlerErr
		}))

		p := helium.NewParser().SAXHandler(s)
		_, err := p.Parse(t.Context(), []byte(input))
		require.Error(t, err)

		var pe helium.ErrParseError
		require.True(t, errors.As(err, &pe), "error should be helium.ErrParseError")
		require.Equal(t, helium.ErrorLevelWarning, pe.Level, "warning should have helium.ErrorLevelWarning")
		require.Greater(t, pe.LineNumber, 0, "warning should carry line number")
	})

	t.Run("warning handler nil does not produce error", func(t *testing.T) {
		t.Parallel()
		var warnings []string
		s := sax.New()
		s.SetOnWarning(sax.WarningFunc(func(_ context.Context, err error) error {
			warnings = append(warnings, err.Error())
			return nil
		}))

		p := helium.NewParser().SAXHandler(s)
		_, err := p.Parse(t.Context(), []byte(input))
		require.NoError(t, err)
		require.NotEmpty(t, warnings, "warning handler should be called")
		require.Contains(t, warnings[0], "undeclared")
	})

	t.Run("SuppressWarnings suppresses warning callback", func(t *testing.T) {
		t.Parallel()
		var called atomic.Int32
		s := sax.New()
		s.SetOnWarning(sax.WarningFunc(func(_ context.Context, err error) error {
			called.Add(1)
			return nil
		}))

		p := helium.NewParser().SAXHandler(s).SuppressWarnings(true)
		_, _ = p.Parse(t.Context(), []byte(input))
		require.Equal(t, int32(0), called.Load(), "warning handler should NOT be called with SuppressWarnings")
	})
}

func TestFormatError(t *testing.T) {
	t.Parallel()

	t.Run("parser error with file", func(t *testing.T) {
		t.Parallel()
		pe := helium.ErrParseError{
			Err:        helium.ErrGtRequired,
			File:       "test.xml",
			Level:      helium.ErrorLevelFatal,
			Line:       "<foo bar",
			LineNumber: 1,
			Column:     9,
		}
		got := pe.FormatError()
		require.Equal(t, "test.xml:1: parser error : '>' was required here\n<foo bar\n        ^", got)
	})

	t.Run("namespace error with file", func(t *testing.T) {
		t.Parallel()
		pe := helium.ErrParseError{
			Domain:     helium.ErrorDomainNamespace,
			Err:        errors.New("namespace 'xLink' not found"),
			File:       "gradient.xml",
			Level:      helium.ErrorLevelFatal,
			Line:       "<linearGradient xLink:href='#g'/>",
			LineNumber: 5,
			Column:     34,
		}
		got := pe.FormatError()
		require.Equal(t, "gradient.xml:5: namespace error : namespace 'xLink' not found\n<linearGradient xLink:href='#g'/>\n                                 ^", got)
	})

	t.Run("warning level", func(t *testing.T) {
		t.Parallel()
		pe := helium.ErrParseError{
			Err:        errors.New("something fishy"),
			Level:      helium.ErrorLevelWarning,
			Line:       "<root/>",
			LineNumber: 1,
			Column:     3,
		}
		got := pe.FormatError()
		require.Equal(t, "parser warning : something fishy\n<root/>\n  ^", got)
	})

	t.Run("without file", func(t *testing.T) {
		t.Parallel()
		pe := helium.ErrParseError{
			Err:        helium.ErrSpaceRequired,
			Level:      helium.ErrorLevelError,
			Line:       "<foo>",
			LineNumber: 1,
			Column:     5,
		}
		got := pe.FormatError()
		require.Equal(t, "parser error : space required\n<foo>\n    ^", got)
	})

	t.Run("no context line", func(t *testing.T) {
		t.Parallel()
		pe := helium.ErrParseError{
			Err:        helium.ErrPrematureEOF,
			File:       "empty.xml",
			Level:      helium.ErrorLevelFatal,
			LineNumber: 1,
			Column:     1,
		}
		got := pe.FormatError()
		require.Equal(t, "empty.xml:1: parser error : end of document reached", got)
	})
}
