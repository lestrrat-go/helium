package catalog_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium/catalog"
	"github.com/lestrrat-go/helium/internal/heliumtest"
	"github.com/stretchr/testify/require"
)

func libxml2TestDir() string {
	return filepath.Join(heliumtest.CallerDir(0), "..", "testdata", "libxml2-compat", "catalogs")
}

func libxml2ResultDir() string {
	return filepath.Join(heliumtest.CallerDir(0), "..", "testdata", "libxml2-compat", "catalogs", "result")
}

// TestLibxml2Compat runs the same catalog resolution tests as libxml2's
// test/catalogs/test.sh. For each .script + .xml pair that has a
// corresponding result golden file, it parses the script commands,
// runs catalog resolution, and compares against the golden output.
func TestLibxml2Compat(t *testing.T) {
	t.Parallel()

	testDir := libxml2TestDir()
	resultDir := libxml2ResultDir()

	if _, err := os.Stat(testDir); err != nil {
		t.Skipf("testdata/libxml2-compat/catalogs not found; run testdata/libxml2/generate.sh first")
	}

	scripts, err := filepath.Glob(filepath.Join(testDir, "*.script"))
	require.NoError(t, err)

	for _, scriptPath := range scripts {
		base := strings.TrimSuffix(filepath.Base(scriptPath), ".script")
		xmlPath := filepath.Join(testDir, base+".xml")
		resultPath := filepath.Join(resultDir, base)

		// Only run tests where both .xml catalog and result golden exist.
		if _, err := os.Stat(xmlPath); err != nil {
			continue
		}
		if _, err := os.Stat(resultPath); err != nil {
			continue
		}

		t.Run(base, func(t *testing.T) {
			t.Parallel()

			cat, err := catalog.Load(context.Background(), xmlPath)
			require.NoError(t, err, "loading catalog %s", xmlPath)

			scriptData, err := os.ReadFile(scriptPath)
			require.NoError(t, err)

			resultData, err := os.ReadFile(resultPath)
			require.NoError(t, err)

			commands := parseScript(string(scriptData))
			expected := parseResults(string(resultData))

			require.Equal(t, len(expected), len(commands),
				"script has %d commands but result has %d entries", len(commands), len(expected))

			for i, cmd := range commands {
				var got string
				switch cmd.typ {
				case "resolve":
					got = cat.Resolve(t.Context(), cmd.arg1, cmd.arg2)
				case "public":
					got = cat.Resolve(t.Context(), cmd.arg1, "")
				case "system":
					got = cat.Resolve(t.Context(), "", cmd.arg1)
				default:
					t.Fatalf("unknown command %q at line %d", cmd.typ, i+1)
				}
				require.Equal(t, expected[i], got,
					"command %d: %s %q %q", i+1, cmd.typ, cmd.arg1, cmd.arg2)
			}
		})
	}
}

type scriptCmd struct {
	typ  string // "resolve", "public", "system"
	arg1 string
	arg2 string // only for "resolve"
}

// parseScript parses xmlcatalog --shell commands from a .script file.
func parseScript(s string) []scriptCmd {
	var cmds []scriptCmd
	for line := range strings.SplitSeq(s, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}

		cmd, rest := splitFirst(line)
		switch cmd {
		case "resolve":
			a1, r := parseArg(rest)
			a2, _ := parseArg(r)
			cmds = append(cmds, scriptCmd{typ: "resolve", arg1: a1, arg2: a2})
		case "public":
			a1, _ := parseArg(rest)
			cmds = append(cmds, scriptCmd{typ: "public", arg1: a1})
		case "system":
			a1, _ := parseArg(rest)
			cmds = append(cmds, scriptCmd{typ: "system", arg1: a1})
		}
	}
	return cmds
}

// splitFirst splits off the first whitespace-delimited word.
func splitFirst(s string) (string, string) {
	s = strings.TrimLeft(s, " \t")
	i := strings.IndexAny(s, " \t")
	if i < 0 {
		return s, ""
	}
	return s[:i], s[i+1:]
}

// parseArg extracts one argument, respecting double-quote delimiters.
// Returns the argument and the remaining string.
func parseArg(s string) (string, string) {
	s = strings.TrimLeft(s, " \t")
	if s == "" {
		return "", ""
	}
	if s[0] == '"' {
		// Quoted argument: find closing quote.
		end := strings.IndexByte(s[1:], '"')
		if end < 0 {
			return s[1:], ""
		}
		return s[1 : end+1], strings.TrimLeft(s[end+2:], " \t")
	}
	// Unquoted: take until next whitespace.
	i := strings.IndexAny(s, " \t")
	if i < 0 {
		return s, ""
	}
	return s[:i], s[i+1:]
}

// parseResults extracts expected resolution results from a golden file.
// Each line is "> <result>" where <result> may be empty. The final
// trailing "> " (the shell exit prompt) is excluded.
func parseResults(s string) []string {
	var results []string
	for line := range strings.SplitSeq(s, "\n") {
		line = strings.TrimRight(line, "\r")
		if !strings.HasPrefix(line, "> ") {
			continue
		}
		results = append(results, line[2:])
	}
	// The last "> " is the shell exit prompt — remove it.
	if len(results) > 0 && results[len(results)-1] == "" {
		results = results[:len(results)-1]
	}
	return results
}
