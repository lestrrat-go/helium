package xslt3_test

import (
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

	h := &recordingMessageHandler{}
	require.NotPanics(t, func() {
		err = ss.Transform(src).MessageHandler(h).WriteTo(ctx, nil)
	})
	require.Error(t, err, "WriteTo with nil writer must return an error")
	require.False(t, h.called, "transform side effects must not run when writer is nil")
}
