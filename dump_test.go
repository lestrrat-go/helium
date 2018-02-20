package helium_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/assert"
)

func TestXMLToDOMToXMLString(t *testing.T) {
	skipped := map[string]struct{}{
		"comment4.xml": {},
	}
	only := map[string]struct{}{}
	if v := os.Getenv("HELIUM_DUMP_TEST_FILES"); v != "" {
		files := strings.Split(v, ",")
		for _, f := range files {
			n := strings.TrimSpace(f)
			only[n] = struct{}{}
		}
	}

	dir := "test"
	files, err := ioutil.ReadDir(dir)
	if !assert.NoError(t, err, "ioutil.ReadDir should succeed") {
		return
	}

	for _, fi := range files {
		if fi.IsDir() {
			continue
		}

		if len(only) > 0 {
			if _, ok := only[fi.Name()]; !ok {
				continue
			}
		} else {
			if _, ok := skipped[fi.Name()]; ok {
				t.Logf("Skipping test for '%s' for now...", fi.Name())
				continue
			}
		}

		fn := filepath.Join(dir, fi.Name())
		if !strings.HasSuffix(fn, ".xml") {
			continue
		}

		goldenfn := strings.Replace(fn, ".xml", ".dump", -1)
		if _, err := os.Stat(goldenfn); err != nil {
			t.Logf("%s does not exist, skipping...", goldenfn)
			continue
		}
		golden, err := ioutil.ReadFile(goldenfn)
		if !assert.NoError(t, err, "ioutil.ReadFile should succeed") {
			return
		}

		t.Logf("Parsing %s...", fn)
		in, err := ioutil.ReadFile(fn)
		if !assert.NoError(t, err, "ioutil.ReadFile should succeed") {
			return
		}

		doc, err := helium.Parse([]byte(in))
		if !assert.NoError(t, err, `Parse(...) succeeds`) {
			return
		}

		str, err := doc.XMLString()
		if !assert.NoError(t, err, "XMLString(doc) succeeds") {
			return
		}

		if !assert.Equal(t, string(golden), str, "roundtrip works") {
			errout, err := os.OpenFile(fn+".err", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
			if err != nil {
				t.Logf("Failed to file to save output: %s", err)
				return
			}
			defer errout.Close()

			errout.WriteString(str)
			return
		}
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
