package xslt3_test

import (
	"bytes"
	"testing"

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

