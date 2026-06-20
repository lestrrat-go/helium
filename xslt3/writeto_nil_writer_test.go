package xslt3_test

import (
	"bytes"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// recordingMessageHandler records whether the transform ran by tracking
// xsl:message invocations.
type recordingMessageHandler struct {
	called bool
}

func (h *recordingMessageHandler) HandleMessage(string, bool) error {
	h.called = true
	return nil
}

// TestWriteToNilWriterRejectsBeforeTransform verifies that WriteTo with a nil
// writer returns an error without panicking and, crucially, without executing
// the transform (and thus without running its side effects).
func TestWriteToNilWriterRejectsBeforeTransform(t *testing.T) {
	ctx := t.Context()

	xsltSrc := `<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/">
    <xsl:message>side effect</xsl:message>
    <out/>
  </xsl:template>
</xsl:stylesheet>`

	doc, err := helium.NewParser().Parse(ctx, []byte(xsltSrc))
	require.NoError(t, err)
	ss, err := xslt3.CompileStylesheet(ctx, doc)
	require.NoError(t, err)

	src, err := helium.NewParser().Parse(ctx, []byte(`<root/>`))
	require.NoError(t, err)

	t.Run("nil interface", func(t *testing.T) {
		h := &recordingMessageHandler{}
		require.NotPanics(t, func() {
			err = ss.Transform(src).MessageHandler(h).WriteTo(ctx, nil)
		})
		require.Error(t, err, "WriteTo with nil writer must return an error")
		require.False(t, h.called, "transform side effects must not run when writer is nil")
	})

	t.Run("typed nil", func(t *testing.T) {
		// A typed-nil writer is a non-nil io.Writer interface wrapping a nil
		// pointer. It must be rejected before the transform runs, not panic
		// during serialization.
		var b *bytes.Buffer
		h := &recordingMessageHandler{}
		require.NotPanics(t, func() {
			err = ss.Transform(src).MessageHandler(h).WriteTo(ctx, b)
		})
		require.Error(t, err, "WriteTo with typed-nil writer must return an error")
		require.False(t, h.called, "transform side effects must not run when writer is a typed nil")
	})
}

// TestWriteToValidationPrecedesNilWriter verifies that configuration
// validation errors keep precedence over the nil-writer guard: an invalid
// invocation invoked with a nil writer must surface the more-specific
// validation error, not the generic nil-writer error.
func TestWriteToValidationPrecedesNilWriter(t *testing.T) {
	ctx := t.Context()

	src, err := helium.NewParser().Parse(ctx, []byte(`<root/>`))
	require.NoError(t, err)

	// A nil stylesheet produces an invocation that fails validation. Calling
	// WriteTo with a nil writer must return the validation error, not the
	// nil-writer error.
	var nilSS *xslt3.Stylesheet

	t.Run("nil interface writer", func(t *testing.T) {
		err := nilSS.Transform(src).WriteTo(ctx, nil)
		require.Error(t, err)
		require.ErrorContains(t, err, "nil stylesheet",
			"validation error must take precedence over the nil-writer error")
	})

	t.Run("typed nil writer", func(t *testing.T) {
		var b *bytes.Buffer
		err := nilSS.Transform(src).WriteTo(ctx, b)
		require.Error(t, err)
		require.ErrorContains(t, err, "nil stylesheet",
			"validation error must take precedence over the nil-writer error")
	})
}
