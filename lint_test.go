package helium_test

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/stretchr/testify/require"
)

// TestHeliumLintGolden tests helium lint output against golden files.
//
// This test looks for .xml files in the test/ directory and compares the output
// of helium lint with corresponding .lint golden files. To create a golden file:
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
		for f := range strings.SplitSeq(v, ",") {
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

		t.Logf("Testing helium lint logic for %s...", fn)

		// Read the XML file
		input, err := os.ReadFile(fn)
		require.NoError(t, err, "os.ReadFile should succeed for input file")

		// Mimic what helium lint does internally
		p := helium.NewParser()
		doc, err := p.Parse(t.Context(), input)
		require.NoError(t, err, "helium.Parse should succeed for %s", fn)

		// Generate output using helium.Writer like helium lint does
		var output bytes.Buffer
		d := helium.NewWriter()
		require.NoError(t, d.WriteTo(&output, doc))

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
		require.Equal(t, expected, actual, "helium lint output should match golden file for %s", fn)
	}
}

// elementContentParentWrites lists the assignments to a `parent` field that are
// NOT writes of docnode.parent: ElementContent (dtd_elem.go) has a `parent`
// field of its own, naming the enclosing content-model node, and the DTD
// content-model builders assemble those trees by hand. Each key is the file
// base name and the receiver expression exactly as the source writes it.
//
// Neither file calls baseDocNode (asserted below), so a docnode.parent write
// hiding here would have to reuse one of these exact receiver names on a
// node-typed variable. Every other `parent` assignment in the package must go
// through setParent.
var elementContentParentWrites = map[string]bool{
	"valid.go:c1":                      true,
	"valid.go:c2":                      true,
	"valid.go:ret.c1":                  true,
	"valid.go:tmp.c1":                  true,
	"parser_dtd_element.go:curelem":    true,
	"parser_dtd_element.go:curelem.c2": true,
	"parser_dtd_element.go:n":          true,
	"parser_dtd_element.go:n.c1":       true,
	"parser_dtd_element.go:op":         true,
	"parser_dtd_element.go:last":       true,
	"parser_dtd_element.go:retelem":    true,
}

// TestNoDirectParentAssignment pins the insertion cycle guard's chokepoint:
// setParent is the SOLE writer of docnode.parent, so docnode.claims — the count
// of nodes naming a given node as their parent — is exact by construction. A
// direct `x.parent = y` anywhere else installs a claim the counter never sees,
// which is precisely the failure mode the counter replaced, so the assignment is
// rejected here rather than discovered later as a cycle the guard let through.
//
// It scans the package's own sources, tests included: a test that corrupts a
// tree on purpose must corrupt it through unsafeSetParent, which routes through
// the chokepoint too.
func TestNoDirectParentAssignment(t *testing.T) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", nil, 0)
	require.NoError(t, err, "the package sources must parse")

	found := map[string]bool{}
	for _, pkg := range pkgs {
		for path, file := range pkg.Files {
			name := filepath.Base(path)
			src, err := os.ReadFile(path)
			require.NoError(t, err)

			var setParentBody *ast.FuncDecl
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if ok && fn.Recv == nil && fn.Name.Name == "setParent" {
					setParentBody = fn
				}
			}

			ast.Inspect(file, func(n ast.Node) bool {
				assign, ok := n.(*ast.AssignStmt)
				if !ok {
					return true
				}
				for _, lhs := range assign.Lhs {
					sel, ok := lhs.(*ast.SelectorExpr)
					if !ok || sel.Sel.Name != "parent" {
						continue
					}
					if setParentBody != nil && sel.Pos() >= setParentBody.Pos() && sel.End() <= setParentBody.End() {
						continue
					}
					recv := string(src[fset.Position(sel.X.Pos()).Offset:fset.Position(sel.X.End()).Offset])
					if key := name + ":" + recv; elementContentParentWrites[key] {
						found[key] = true
						continue
					}
					t.Errorf("%s: direct assignment to %s.parent; docnode.parent has exactly one writer, setParent (node.go)",
						fset.Position(sel.Pos()), recv)
				}
				return true
			})
		}
	}

	// A stale exemption is as bad as a missing one: it would silently license a
	// future docnode.parent write on that receiver name.
	for key := range elementContentParentWrites {
		require.True(t, found[key],
			"stale exemption %q: that parent assignment is gone; drop the entry", key)
		name, _, _ := strings.Cut(key, ":")
		src, err := os.ReadFile(name)
		require.NoError(t, err)
		require.NotContains(t, string(src), "baseDocNode",
			"%s reaches docnode; its parent-assignment exemptions can no longer be assumed to be ElementContent writes", name)
	}
}
