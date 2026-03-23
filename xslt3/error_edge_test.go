package xslt3_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

func TestTransformNilStylesheet(t *testing.T) {
	_, err := xslt3.Transform(t.Context(), nil, nil)
	require.EqualError(t, err, "xslt3: nil stylesheet")
}

func TestTransformStringNilStylesheet(t *testing.T) {
	_, err := xslt3.TransformString(t.Context(), nil, nil)
	require.EqualError(t, err, "xslt3: nil stylesheet")
}

func TestTransformToWriterNilStylesheet(t *testing.T) {
	var buf bytes.Buffer
	err := xslt3.TransformToWriter(t.Context(), nil, nil, &buf)
	require.EqualError(t, err, "xslt3: nil stylesheet")
}

func TestReceiverBadType(t *testing.T) {
	ctx := t.Context()

	doc, err := helium.Parse(ctx, []byte(`<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="/"><out/></xsl:template>
</xsl:stylesheet>`))
	require.NoError(t, err)

	ss, err := xslt3.CompileStylesheet(ctx, doc)
	require.NoError(t, err)

	_, err = ss.Transform(nil).Receiver("not a receiver").Do(ctx)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "does not implement any known receiver interface"),
		"unexpected error: %s", err)
}
