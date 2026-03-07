package html_test

import (
	"context"
	"testing"

	"github.com/lestrrat-go/helium/html"
)

func FuzzParse(f *testing.F) {
	f.Add([]byte(`<html><head><title>Test</title></head><body><p>Hello</p></body></html>`))
	f.Add([]byte(`<!DOCTYPE html><html><body><div class="test"><img src="x.png"><br></div></body></html>`))
	f.Add([]byte(`<p>unclosed paragraph<p>another paragraph`))
	f.Add([]byte(`<table><tr><td>cell</td></tr></table>`))
	f.Add([]byte(`<script>var x = 1 < 2;</script><style>a { color: red; }</style>`))
	f.Add([]byte(`&amp;&lt;&gt;&quot;&apos;&copy;&nbsp;`))
	f.Add([]byte(``))
	f.Add([]byte(`not html at all`))

	f.Fuzz(func(_ *testing.T, data []byte) {
		_, _ = html.Parse(context.Background(), data)
	})
}
