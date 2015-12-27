package helium_test

import (
	"testing"

	"github.com/lestrrat/helium"
	"github.com/stretchr/testify/assert"
)

func TestXMLString(t *testing.T) {
	doc := helium.CreateDocument()
	//	defer doc.Free()

	root, err := doc.CreateElement("root")
	if !assert.NoError(t, err, `CreateElement("root") succeeds`) {
		return
	}

	doc.SetDocumentElement(root)
	root.AddContent([]byte(`Hello, World!`))

	str, err := doc.XMLString()
	if !assert.NoError(t, err, "XMLString(doc) succeeds") {
		return
	}

	t.Logf("%s", str)
}
