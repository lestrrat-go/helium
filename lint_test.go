package helium_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// TestHeliumLintGolden tests helium-lint command output against golden files.
//
// This test looks for .xml files in the test/ directory and compares the output
// of helium-lint with corresponding .lint golden files. To create a golden file:
//
//	xmllint test/example.xml > test/example.lint
//
// The test will automatically pick up any .xml file that has a corresponding .lint file.
//
// Environment variable HELIUM_LINT_TEST_FILES can be set to test only specific files:
//
//	HELIUM_LINT_TEST_FILES=xml2.xml,comment.xml go test -run TestHeliumLintGolden
func TestHeliumLintGolden(t *testing.T) {
	// Skip files that are known to have issues or different behavior
	skipped := map[string]struct{}{
		// Add any files that need to be skipped here
	}

	// Allow testing only specific files via environment variable
	only := map[string]struct{}{}
	if v := os.Getenv("HELIUM_LINT_TEST_FILES"); v != "" {
		for _, f := range strings.Split(v, ",") {
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
				t.Logf("Skipping lint test for '%s' for now...", fi.Name())
				continue
			}
		}

		fn := filepath.Join(dir, fi.Name())
		if !strings.HasSuffix(fn, ".xml") {
			continue
		}

		// Look for corresponding .lint golden file
		goldenfn := strings.ReplaceAll(fn, ".xml", ".lint")
		if _, err := os.Stat(goldenfn); err != nil {
			t.Logf("%s does not exist, skipping lint test...", goldenfn)
			continue
		}

		golden, err := os.ReadFile(goldenfn)
		require.NoError(t, err, "os.ReadFile should succeed for golden file")

		t.Logf("Testing helium-lint logic for %s...", fn)

		// Read the XML file
		input, err := os.ReadFile(fn)
		require.NoError(t, err, "os.ReadFile should succeed for input file")

		// Mimic what helium-lint does internally
		p := helium.NewParser()
		doc, err := p.Parse(context.Background(), input)
		require.NoError(t, err, "helium.Parse should succeed for %s", fn)

		// Generate output using helium.Dumper like helium-lint does
		var output bytes.Buffer
		d := helium.Dumper{}
		require.NoError(t, d.DumpDoc(&output, doc))

		actual := output.String()
		expected := string(golden)

		if expected != actual {
			// Save the actual output to .err file for debugging
			errout, err := os.OpenFile(fn+".lint.err", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
			if err != nil {
				t.Logf("Failed to create file to save output: %s", err)
				return
			}
			defer func() { _ = errout.Close() }()

			_, _ = errout.WriteString(actual)
			t.Logf("Actual output saved to %s", fn+".lint.err")
		}
		require.Equal(t, expected, actual, "helium-lint output should match golden file for %s", fn)
	}
}
