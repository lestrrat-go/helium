package helium_test

import (
	"testing"

	"github.com/lestrrat-go/helium/v2"
	"github.com/stretchr/testify/assert"
)

func TestDocument(t *testing.T) {
	doc := helium.CreateDocument()
	if !assert.NotNil(t, doc, `helium.CreateDocument() should succeed`) {
		return
	}
}

func TestText(t *testing.T) {
	const val = "Hello, World!"
	doc := helium.CreateDocument()

	t.Run("CreateText", func(t *testing.T) {
		txt, err := doc.CreateText([]byte(val))
		if !assert.NoError(t, err, `doc.CreateText() should succeed`) {
			return
		}

		if !assert.Equal(t, []byte(val), txt.Content(), `content should match`) {
			return
		}

		if !assert.Equal(t, doc, txt.OwnerDocument(), `document should match`) {
			return
		}
	})
	t.Run("AddContent", func(t *testing.T) {
		doc := helium.CreateDocument()
		txt, err := doc.CreateText([]byte("Hello, "))
		if !assert.NoError(t, err, `doc.CreateText() should succeed`) {
			return
		}
		if !assert.NoError(t, txt.AddContent([]byte("World!")), `txt.AddContent() should succeed`) {
			return
		}
		if !assert.Equal(t, []byte(val), txt.Content(), `txt.AddContent() should succeed`) {
			return
		}
	})
	t.Run("AddChild", func(t *testing.T) {
		txt1, err := doc.CreateText([]byte("Hello, "))
		if !assert.NoError(t, err, `doc.CreateText() should succeed`) {
			return
		}
		txt2, err := doc.CreateText([]byte("World!"))
		if !assert.NoError(t, err, `doc.CreateText() should succeed`) {
			return
		}

		if !assert.NoError(t, txt1.AddChild(txt2), `txt1.AddChild should succeed`) {
			return
		}

		if !assert.Equal(t, []byte(val), txt1.Content(), `txt.AddContent() should succeed`) {
			return
		}
	})
}
