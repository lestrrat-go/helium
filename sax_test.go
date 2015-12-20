package helium_test

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lestrrat/helium"
	"github.com/lestrrat/helium/sax"
	"github.com/stretchr/testify/assert"
)

func newEventEmitter(out io.Writer) sax.Handler {
	s := sax.New()
	s.StartDocumentHandler = func(_ interface{}) error {
		fmt.Fprintf(out, "SAX.StartDocument()\n")
		return nil
	}

	return s
}

func TestSAXEvents(t *testing.T) {
	t.Skip("parts handling this test aren't implemented yet")

	dir := "test"
	files, err := ioutil.ReadDir(dir)
	if !assert.NoError(t, err, "ioutil.ReadDir should succeed") {
		return
	}

	for _, fi := range files {
		if fi.IsDir() {
			continue
		}
		fn := filepath.Join(dir, fi.Name())
		if !strings.HasSuffix(fn, ".xml") {
			continue
		}

		in, err := ioutil.ReadFile(fn)
		if !assert.NoError(t, err, "ioutil.ReadFile should succeed") {
			return
		}

		golden, err := ioutil.ReadFile(strings.Replace(fn, ".xml", ".sax2", -1))
		if !assert.NoError(t, err, "ioutil.ReadFile should succeed") {
			return
		}

		out := bytes.Buffer{}
		p := helium.NewParser()
		p.SetSAXHandler(newEventEmitter(&out))

		_, err = p.Parse(in)
		if !assert.NoError(t, err, "Parse should succeed") {
			t.Logf("source XML: %s", in)
			return
		}

		if !assert.Equal(t, golden, out.Bytes(), "SAX event streams should match") {
			return
		}
	}
}