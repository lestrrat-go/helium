package helium_test

import (
	"testing"

	"github.com/lestrrat/helium"
	"github.com/stretchr/testify/assert"
)

func TestXMLToDOMToXMLString(t *testing.T) {
	const input = `<root>Hello, World!</root>`
	doc, err := helium.Parse([]byte(input))
	if !assert.NoError(t, err, `Parse(...) succeeds`) {
		return
	}

	str, err := doc.XMLString()
	if !assert.NoError(t, err, "XMLString(doc) succeeds") {
		return
	}

	if !assert.Equal(t, input, str, "roundtrip works") {
		return
	}
}

func TestDOMToXMLString(t *testing.T) {
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
