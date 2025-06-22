package helium_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
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
	files, err := os.ReadDir(dir)
	require.NoError(t, err, "os.ReadDir should succeed")

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
		golden, err := os.ReadFile(goldenfn)
		require.NoError(t, err, "os.ReadFile should succeed")

		t.Logf("Parsing %s...", fn)
		in, err := os.ReadFile(fn)
		require.NoError(t, err, "os.ReadFile should succeed")

		doc, err := helium.Parse([]byte(in))
		require.NoError(t, err, `Parse(...) succeeds`)

		str, err := doc.XMLString()
		require.NoError(t, err, "XMLString(doc) succeeds")

		if string(golden) != str {
			errout, err := os.OpenFile(fn+".err", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
			if err != nil {
				t.Logf("Failed to file to save output: %s", err)
				return
			}
			defer errout.Close()

			errout.WriteString(str)
		}
		require.Equal(t, string(golden), str, "roundtrip works")
	}
}

func TestDOMToXMLString(t *testing.T) {
	doc := helium.CreateDocument()
	//	defer doc.Free()

	root, err := doc.CreateElement("root")
	require.NoError(t, err, `CreateElement("root") succeeds`)

	doc.SetDocumentElement(root)
	root.AddContent([]byte(`Hello, World!`))

	str, err := doc.XMLString()
	require.NoError(t, err, "XMLString(doc) succeeds")

	t.Logf("%s", str)
}
