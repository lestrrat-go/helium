package helium_test

import (
	"io"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// BenchmarkWriteNonASCII serializes a document containing many non-ASCII
// characters with EscapeNonASCII enabled, exercising the hex char ref path.
func BenchmarkWriteNonASCII(b *testing.B) {
	// Build a document with 200 text nodes containing Latin-1 characters.
	var buf strings.Builder
	buf.WriteString("<root>")
	for i := 0; i < 200; i++ {
		buf.WriteString("<t>caf\u00e9 na\u00efve r\u00e9sum\u00e9 \u00fcber \u00e0 \u00e7a \u00f1</t>")
	}
	buf.WriteString("</root>")

	doc, err := helium.NewParser().Parse(b.Context(), []byte(buf.String()))
	require.NoError(b, err)

	w := helium.NewWriter().EscapeNonASCII(true)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := w.WriteDoc(io.Discard, doc)
		require.NoError(b, err)
	}
}
