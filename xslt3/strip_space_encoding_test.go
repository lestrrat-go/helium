package xslt3_test

import (
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// TestStripSpaceCopyNoEncoding verifies that the single-pass strip-space copy
// faithfully reproduces a source document's encoding state. A source whose XML
// declaration omitted an encoding must yield a copy that ALSO has no encoding
// declaration: neither the copy's recorded encoding nor its serialized form may
// carry a synthesized encoding="utf8" the source never had. See finding 664-5.
func TestStripSpaceCopyNoEncoding(t *testing.T) {
	t.Parallel()

	src, err := helium.NewParser().Parse(t.Context(),
		[]byte("<doc>\n  <item>x</item>\n</doc>"))
	require.NoError(t, err)
	require.Empty(t, src.RawEncoding(),
		"fixture source must have no encoding declaration to be meaningful")

	dst, err := xslt3.CopyAndStripForTest(src)
	require.NoError(t, err)

	require.Empty(t, dst.RawEncoding(),
		"strip-space copy of an encoding-less source must have no encoding (got %q)", dst.RawEncoding())

	srcOut, err := helium.WriteString(src)
	require.NoError(t, err)
	dstOut, err := helium.WriteString(dst)
	require.NoError(t, err)

	require.NotContains(t, srcOut, "encoding=",
		"sanity: encoding-less source must not serialize an encoding= attribute (got %q)", srcOut)
	require.NotContains(t, dstOut, "encoding=",
		"strip-space copy must not serialize a spurious encoding= attribute (got %q)", dstOut)
}

// TestStripSpaceCopyEncodingFaithful verifies the converse: when the source DOES
// declare an encoding, version, or standalone, the strip-space copy reproduces
// each exactly.
func TestStripSpaceCopyEncodingFaithful(t *testing.T) {
	t.Parallel()

	src, err := helium.NewParser().Parse(t.Context(),
		[]byte(`<?xml version="1.1" encoding="UTF-8" standalone="yes"?>`+"\n<doc><item>x</item></doc>"))
	require.NoError(t, err)
	require.Equal(t, "UTF-8", src.RawEncoding())
	require.Equal(t, "1.1", src.Version())

	dst, err := xslt3.CopyAndStripForTest(src)
	require.NoError(t, err)

	require.Equal(t, src.RawEncoding(), dst.RawEncoding(),
		"strip-space copy must reproduce the source encoding exactly")
	require.Equal(t, src.Version(), dst.Version(),
		"strip-space copy must reproduce the source version exactly")
	require.Equal(t, src.Standalone(), dst.Standalone(),
		"strip-space copy must reproduce the source standalone exactly")

	dstOut, err := helium.WriteString(dst)
	require.NoError(t, err)
	require.Contains(t, dstOut, `encoding="UTF-8"`,
		"strip-space copy of an encoded source must serialize its encoding (got %q)", dstOut)
}
