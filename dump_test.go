package helium_test

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/s11n"
	"github.com/stretchr/testify/require"
)

func TestXMLToDOMToXMLString(t *testing.T) {
	ctx := context.Background()
	if testing.Verbose() {
		ctx = helium.WithTraceLogger(ctx, slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))
	}

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

		doc, err := helium.Parse(ctx, []byte(in))
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
	// Enable logging for this test
	ctx := context.Background()
	if testing.Verbose() {
		ctx = helium.WithTraceLogger(ctx, slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))
	}
	doc := helium.CreateDocument()
	//	defer doc.Free()

	root := doc.CreateElement("root")
	require.NotNil(t, root, `CreateElement("root") succeeds`)

	require.NoError(t, doc.SetDocumentElement(root))
	require.NoError(t, root.AddContent([]byte(`Hello, World!`)))

	var output strings.Builder
	d := s11n.Dumper{}
	require.NoError(t, d.DumpDoc(&output, doc), "DumpDoc(doc) succeeds")

	t.Logf("%s", output.String())
}
