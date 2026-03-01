package helium

import (
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/lestrrat-go/helium/sax"
	"github.com/stretchr/testify/require"
)

func TestErrParseErrorUnwrap(t *testing.T) {
	pe := ErrParseError{
		Err:        ErrSpaceRequired,
		Level:      ErrorLevelError,
		Line:       "<foo>",
		LineNumber: 1,
		Column:     5,
	}
	require.True(t, errors.Is(pe, ErrSpaceRequired))
	require.False(t, errors.Is(pe, ErrEOF))
}

func TestErrParseErrorAs(t *testing.T) {
	pe := ErrParseError{
		Err:        ErrSpaceRequired,
		File:       "test.xml",
		Level:      ErrorLevelError,
		Line:       "<foo>",
		LineNumber: 3,
		Column:     10,
	}
	var wrapped error = pe

	var extracted ErrParseError
	require.True(t, errors.As(wrapped, &extracted))
	require.Equal(t, "test.xml", extracted.File)
	require.Equal(t, ErrorLevelError, extracted.Level)
	require.Equal(t, 3, extracted.LineNumber)
	require.Equal(t, 10, extracted.Column)
	require.Equal(t, "<foo>", extracted.Line)
}

func TestErrParseErrorErrorString(t *testing.T) {
	t.Run("without file", func(t *testing.T) {
		pe := ErrParseError{
			Err:        ErrSpaceRequired,
			Level:      ErrorLevelError,
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
		pe := ErrParseError{
			Err:        ErrSpaceRequired,
			File:       "test.xml",
			Level:      ErrorLevelError,
			Line:       "<foo>",
			LineNumber: 1,
			Column:     5,
		}
		msg := pe.Error()
		require.True(t, strings.HasPrefix(msg, "test.xml: "))
	})
}

func TestErrParseErrorLevel(t *testing.T) {
	_, err := Parse([]byte("<broken"))
	require.Error(t, err)

	var pe ErrParseError
	require.True(t, errors.As(err, &pe))
	require.Equal(t, ErrorLevelFatal, pe.Level)
}

func TestErrParseErrorWarningLevel(t *testing.T) {
	// XML with undefined entity reference in a non-standalone document
	// with an external subset. The parser emits a warning (not an error)
	// for undefined entities in this context.
	const input = `<?xml version="1.0"?>
<!DOCTYPE root SYSTEM "nonexistent.dtd">
<root>&undefined;</root>`

	s := sax.New()
	s.WarningHandler = sax.WarningFunc(func(_ sax.Context, err error) error {
		return errors.New("warning escalated")
	})

	p := NewParser()
	p.SetSAXHandler(s)
	_, err := p.Parse([]byte(input))
	require.Error(t, err)

	var pe ErrParseError
	require.True(t, errors.As(err, &pe))
	require.Equal(t, ErrorLevelWarning, pe.Level)
}

func TestErrDTDDupTokenFixed(t *testing.T) {
	e := ErrDTDDupToken{Name: "foo"}
	require.Contains(t, e.Error(), "standalone")
	require.NotContains(t, e.Error(), "standlone")
}

func TestErrorLevelConstants(t *testing.T) {
	require.Equal(t, ErrorLevel(0), ErrorLevelNone)
	require.Equal(t, ErrorLevel(1), ErrorLevelWarning)
	require.Equal(t, ErrorLevel(2), ErrorLevelError)
	require.Equal(t, ErrorLevel(3), ErrorLevelFatal)

	require.True(t, ErrorLevelNone < ErrorLevelWarning)
	require.True(t, ErrorLevelWarning < ErrorLevelError)
	require.True(t, ErrorLevelError < ErrorLevelFatal)
}

func TestParseNoError(t *testing.T) {
	// Malformed XML triggers SAX Error callback
	const input = `<?xml version="1.0"?><root><child>text</chld></root>`

	t.Run("SAX Error callback fires by default", func(t *testing.T) {
		var called atomic.Int32
		s := sax.New()
		s.ErrorHandler = sax.ErrorFunc(func(_ sax.Context, err error) error {
			called.Add(1)
			return nil
		})

		p := NewParser()
		p.SetSAXHandler(s)
		_, _ = p.Parse([]byte(input))
		require.Greater(t, called.Load(), int32(0), "SAX Error handler should be called")
	})

	t.Run("ParseNoError suppresses SAX Error callback", func(t *testing.T) {
		var called atomic.Int32
		s := sax.New()
		s.ErrorHandler = sax.ErrorFunc(func(_ sax.Context, err error) error {
			called.Add(1)
			return nil
		})

		p := NewParser()
		p.SetSAXHandler(s)
		p.SetOption(ParseNoError)
		_, err := p.Parse([]byte(input))
		require.Error(t, err, "parse should still return error")
		require.Equal(t, int32(0), called.Load(), "SAX Error handler should NOT be called with ParseNoError")
	})
}

func TestWarningLocationInfo(t *testing.T) {
	// XML with external subset makes undeclared entity references a warning
	// rather than an error.
	const input = "<!DOCTYPE doc SYSTEM \"foo\">\n<doc>&undeclared;</doc>"

	t.Run("warning handler error wrapped with ErrorLevelWarning", func(t *testing.T) {
		handlerErr := fmt.Errorf("warning escalated")
		s := sax.New()
		s.WarningHandler = sax.WarningFunc(func(_ sax.Context, err error) error {
			return handlerErr
		})

		p := NewParser()
		p.SetSAXHandler(s)
		_, err := p.Parse([]byte(input))
		require.Error(t, err)

		var pe ErrParseError
		require.True(t, errors.As(err, &pe), "error should be ErrParseError")
		require.Equal(t, ErrorLevelWarning, pe.Level, "warning should have ErrorLevelWarning")
		require.Greater(t, pe.LineNumber, 0, "warning should carry line number")
	})

	t.Run("warning handler nil does not produce error", func(t *testing.T) {
		var warnings []string
		s := sax.New()
		s.WarningHandler = sax.WarningFunc(func(_ sax.Context, err error) error {
			warnings = append(warnings, err.Error())
			return nil
		})

		p := NewParser()
		p.SetSAXHandler(s)
		_, err := p.Parse([]byte(input))
		require.NoError(t, err)
		require.NotEmpty(t, warnings, "warning handler should be called")
		require.Contains(t, warnings[0], "undeclared")
	})

	t.Run("ParseNoWarning suppresses warning callback", func(t *testing.T) {
		var called atomic.Int32
		s := sax.New()
		s.WarningHandler = sax.WarningFunc(func(_ sax.Context, err error) error {
			called.Add(1)
			return nil
		})

		p := NewParser()
		p.SetSAXHandler(s)
		p.SetOption(ParseNoWarning)
		_, _ = p.Parse([]byte(input))
		require.Equal(t, int32(0), called.Load(), "warning handler should NOT be called with ParseNoWarning")
	})
}

func TestParserLastError(t *testing.T) {
	p := NewParser()
	_, err := p.Parse([]byte("<broken"))
	require.Error(t, err)

	lastErr := p.LastError()
	require.NotNil(t, lastErr, "LastError should be set after parsing malformed XML")

	var pe ErrParseError
	require.True(t, errors.As(lastErr, &pe), "LastError should be ErrParseError")
	require.Greater(t, pe.LineNumber, 0, "LastError should carry line number")
}

func TestParserLastErrorNilOnSuccess(t *testing.T) {
	p := NewParser()
	_, err := p.Parse([]byte(`<?xml version="1.0"?><root/>`))
	require.NoError(t, err)
	require.Nil(t, p.LastError(), "LastError should be nil after successful parse")
}

func TestParserLastErrorWarning(t *testing.T) {
	const input = "<!DOCTYPE doc SYSTEM \"foo\">\n<doc>&undeclared;</doc>"

	s := sax.New()
	s.WarningHandler = sax.WarningFunc(func(_ sax.Context, err error) error {
		return nil
	})

	p := NewParser()
	p.SetSAXHandler(s)
	_, err := p.Parse([]byte(input))
	require.NoError(t, err)

	lastErr := p.LastError()
	require.NotNil(t, lastErr, "LastError should capture warnings")

	var pe ErrParseError
	require.True(t, errors.As(lastErr, &pe), "LastError should be ErrParseError")
	require.Equal(t, ErrorLevelWarning, pe.Level, "LastError should have ErrorLevelWarning")
}

func TestParserLastErrorOverwrittenBySubsequentParse(t *testing.T) {
	p := NewParser()

	// First parse: malformed XML sets lastError
	_, err := p.Parse([]byte("<broken"))
	require.Error(t, err)
	require.NotNil(t, p.LastError(), "LastError should be set after malformed parse")

	// Second parse: valid XML clears lastError
	_, err = p.Parse([]byte(`<?xml version="1.0"?><root/>`))
	require.NoError(t, err)
	require.Nil(t, p.LastError(), "LastError should be nil after subsequent successful parse")
}
