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
	skipped := map[string]struct{}{}
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

		goldenfn := strings.ReplaceAll(fn, ".xml", ".dump")
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
			defer func() { _ = errout.Close() }()

			_, _ = errout.WriteString(str)
		}
		require.Equal(t, string(golden), str, "roundtrip works")
	}
}

func TestDOMToXMLString(t *testing.T) {
	doc := helium.CreateDocument()
	//	defer doc.Free()

	root, err := doc.CreateElement("root")
	require.NoError(t, err, `CreateElement("root") succeeds`)

	require.NoError(t, doc.SetDocumentElement(root))
	require.NoError(t, root.AddContent([]byte(`Hello, World!`)))

	str, err := doc.XMLString()
	require.NoError(t, err, "XMLString(doc) succeeds")

	t.Logf("%s", str)
}

func TestFormatOutput(t *testing.T) {
	t.Run("nested elements", func(t *testing.T) {
		doc, err := helium.Parse([]byte(`<?xml version="1.0"?><root><child><grandchild/></child></root>`))
		require.NoError(t, err)

		str, err := doc.XMLString(helium.WithFormat())
		require.NoError(t, err)

		expected := "<?xml version=\"1.0\"?>\n<root>\n  <child>\n    <grandchild/>\n  </child>\n</root>\n"
		require.Equal(t, expected, str)
	})

	t.Run("text-only element stays inline", func(t *testing.T) {
		doc, err := helium.Parse([]byte(`<?xml version="1.0"?><root><child>hello</child></root>`))
		require.NoError(t, err)

		str, err := doc.XMLString(helium.WithFormat())
		require.NoError(t, err)

		expected := "<?xml version=\"1.0\"?>\n<root>\n  <child>hello</child>\n</root>\n"
		require.Equal(t, expected, str)
	})

	t.Run("custom indent string", func(t *testing.T) {
		doc, err := helium.Parse([]byte(`<?xml version="1.0"?><root><child><grandchild/></child></root>`))
		require.NoError(t, err)

		str, err := doc.XMLString(helium.WithFormat(), helium.WithIndentString("\t"))
		require.NoError(t, err)

		expected := "<?xml version=\"1.0\"?>\n<root>\n\t<child>\n\t\t<grandchild/>\n\t</child>\n</root>\n"
		require.Equal(t, expected, str)
	})

	t.Run("without format stays compact", func(t *testing.T) {
		doc, err := helium.Parse([]byte(`<?xml version="1.0"?><root><child><grandchild/></child></root>`))
		require.NoError(t, err)

		str, err := doc.XMLString()
		require.NoError(t, err)

		expected := "<?xml version=\"1.0\"?>\n<root><child><grandchild/></child></root>\n"
		require.Equal(t, expected, str)
	})

	t.Run("multiple children", func(t *testing.T) {
		doc, err := helium.Parse([]byte(`<?xml version="1.0"?><root><a/><b/><c/></root>`))
		require.NoError(t, err)

		str, err := doc.XMLString(helium.WithFormat())
		require.NoError(t, err)

		expected := "<?xml version=\"1.0\"?>\n<root>\n  <a/>\n  <b/>\n  <c/>\n</root>\n"
		require.Equal(t, expected, str)
	})

	t.Run("element XMLString with format", func(t *testing.T) {
		doc, err := helium.Parse([]byte(`<?xml version="1.0"?><root><child><grandchild/></child></root>`))
		require.NoError(t, err)

		root := doc.DocumentElement()
		require.NotNil(t, root)

		str, err := root.XMLString(helium.WithFormat())
		require.NoError(t, err)

		expected := "<root>\n  <child>\n    <grandchild/>\n  </child>\n</root>"
		require.Equal(t, expected, str)
	})

	t.Run("comment and PI children", func(t *testing.T) {
		doc, err := helium.Parse([]byte(`<?xml version="1.0"?><root><!--comment--><child/><?pi data?></root>`))
		require.NoError(t, err)

		str, err := doc.XMLString(helium.WithFormat())
		require.NoError(t, err)

		expected := "<?xml version=\"1.0\"?>\n<root>\n  <!--comment-->\n  <child/>\n  <?pi data?>\n</root>\n"
		require.Equal(t, expected, str)
	})

	t.Run("deeply nested", func(t *testing.T) {
		doc, err := helium.Parse([]byte(`<?xml version="1.0"?><a><b><c><d>text</d></c></b></a>`))
		require.NoError(t, err)

		str, err := doc.XMLString(helium.WithFormat())
		require.NoError(t, err)

		expected := "<?xml version=\"1.0\"?>\n<a>\n  <b>\n    <c>\n      <d>text</d>\n    </c>\n  </b>\n</a>\n"
		require.Equal(t, expected, str)
	})
}
