package helium_test

import (
	"bytes"
	"testing"

	"github.com/lestrrat/helium"
	"github.com/stretchr/testify/assert"
)

func TestDump(t *testing.T) {
	doc := helium.CreateDocument()
	//	defer doc.Free()

	root, err := doc.CreateElement("root")
	if !assert.NoError(t, err, `CreateElement("root") succeeds`) {
		return
	}

	doc.SetDocumentElement(root)
	root.AddContent([]byte(`Hello, World!`))

	out := bytes.Buffer{}
	err = (&helium.Dumper{}).DumpDoc(&out, doc)
	if !assert.NoError(t, err, "DumpDoc(doc) succeeds") {
		return
	}

	t.Logf("%s", out.String())
}
