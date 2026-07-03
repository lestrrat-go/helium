package xslt3_test

import (
	"io"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xslt3"
)

// BenchmarkSerializeResultXML measures the default XML-method serialization
// path (version unset → XML 1.0) over a document with many text nodes. It
// guards against reintroducing an extra whole-tree traversal for XML-version
// character validation: the SERE0006 check is folded into the writer's escape
// pass, so this benchmark should match the plain-serialization fast path.
func BenchmarkSerializeResultXML(b *testing.B) {
	const textNodes = 20000
	var sb strings.Builder
	sb.WriteString("<root>")
	for range textNodes {
		sb.WriteString("<item>lorem ipsum dolor sit amet consectetur</item>")
	}
	sb.WriteString("</root>")

	doc, err := helium.NewParser().Parse(b.Context(), []byte(sb.String()))
	if err != nil {
		b.Fatalf("parse: %v", err)
	}
	outDef := &xslt3.OutputDef{Method: "xml"}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if err := xslt3.SerializeResult(io.Discard, doc, outDef); err != nil {
			b.Fatalf("serialize: %v", err)
		}
	}
}
