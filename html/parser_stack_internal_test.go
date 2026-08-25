package html

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// naiveHasOnStack is the pre-index linear-scan reference for hasOnStack: it is
// exactly what hasOnStack was before the namePos index (slices.Contains over
// nameStack).
func naiveHasOnStack(stack []string, name string) bool {
	return slices.Contains(stack, name)
}

// naiveAutoCloseWouldRun reproduces the ORIGINAL htmlAutoCloseOnClose
// backward-scan-with-priority-abort exactly, reporting whether the pop loop
// would run (true) or the function would return without popping anything
// (false). It is the reference the indexed pre-scan is checked against.
func naiveAutoCloseWouldRun(stack []string, endTag string) bool {
	priority := getEndPriority(endTag)
	found := false
	for _, v := range slices.Backward(stack) {
		if v == endTag {
			found = true
			break
		}
		if getEndPriority(v) > priority {
			return false
		}
	}
	return found
}

// indexedAutoCloseWouldRun mirrors htmlAutoCloseOnClose's pre-scan decision
// without running the pop loop, so it can be compared against
// naiveAutoCloseWouldRun without side effects on the parser.
func indexedAutoCloseWouldRun(p *parser, endTag string) bool {
	pos, ok := p.topmostPos(endTag)
	if !ok {
		return false
	}
	if blocked, at := p.topmostAbovePriority(getEndPriority(endTag)); blocked && at > pos {
		return false
	}
	return true
}

// diffCheckAdversarialInputs mirrors the shapes captured in
// .tmp/html-stack-baseline.txt: absent end tags, blocked end tags, table
// auto-close, inline mismatch, a stray end tag, and a pre-root end tag.
var diffCheckAdversarialInputs = []string{
	`<html><body><div><div></span>x`,
	`<html><body><span><div><b><b></span></span>x`,
	`<html><body><div><b><i></div>x`,
	`<html><body><table><tr><td><div></tr>x`,
	`<html><body><table><tr><td></table>x`,
	`<html><body><p>a<p>b</p></p>x`,
	`<html><body><div></div></div></div>x`,
	`<html><body><b><b><b></b>x`,
	`<html><head><title>t</title></head><body></head></body>x`,
	`<html><body><ul><li><li></ul>x`,
	`</div><html><body>x</body></html>`,
	`<html><body><div><span></div></span>x`,
	`<html><body><b><div></b>x`,
	`<html><body><font><table><td></font>x`,
	`<html><body><div><div><div></body>x`,
}

// diffCheckFuzzSeeds mirrors FuzzParse's seed corpus (fuzz_test.go).
var diffCheckFuzzSeeds = []string{
	`<html><head><title>Test</title></head><body><p>Hello</p></body></html>`,
	`<!DOCTYPE html><html><body><div class="test"><img src="x.png"><br></div></body></html>`,
	`<p>unclosed paragraph<p>another paragraph`,
	`<table><tr><td>cell</td></tr></table>`,
	`<script>var x = 1 < 2;</script><style>a { color: red; }</style>`,
	`&amp;&lt;&gt;&quot;&apos;&copy;&nbsp;`,
	``,
	`not html at all`,
}

// stackDiffChecker installs itself as stackDiffCheck for the duration of one
// parse and compares the indexed hasOnStack/htmlAutoCloseOnClose answers
// against the naive linear-scan reference after every stack mutation. seen
// accumulates every name observed on the stack, plus every name in
// htmlEndPriority, so coverage widens as more inputs are checked.
type stackDiffChecker struct {
	t    *testing.T
	seen map[string]struct{}
}

// check is installed as stackDiffCheck. It runs after every pushName/popName.
func (c *stackDiffChecker) check(p *parser) {
	c.t.Helper()
	if len(p.nameStack) > 0 {
		c.seen[p.nameStack[len(p.nameStack)-1]] = struct{}{}
	}
	for name := range c.seen {
		if want, got := naiveHasOnStack(p.nameStack, name), p.hasOnStack(name); want != got {
			c.t.Errorf("hasOnStack(%q): naive=%v indexed=%v stack=%v", name, want, got, p.nameStack)
		}
		if want, got := naiveAutoCloseWouldRun(p.nameStack, name), indexedAutoCloseWouldRun(p, name); want != got {
			c.t.Errorf("autoCloseWouldRun(%q): naive=%v indexed=%v stack=%v", name, want, got, p.nameStack)
		}
	}
}

// runStackDiffCheck parses input with a stackDiffChecker installed, so every
// pushName/popName mutation during the parse is checked against the naive
// reference.
func runStackDiffCheck(t *testing.T, seen map[string]struct{}, input []byte) {
	t.Helper()
	checker := &stackDiffChecker{t: t, seen: seen}
	stackDiffCheck = checker.check
	defer func() { stackDiffCheck = nil }()
	p := NewParser().SuppressErrors(true).SuppressWarnings(true)
	_ = p.ParseWithSAX(t.Context(), input, &SAXCallbacks{})
}

// TestStackIndexesMatchNaive is a differential test: it drives real parses
// over the adversarial blocked/mismatched-end-tag shapes, the FuzzParse seed
// corpus, and the 45-file libxml2-compat golden corpus, and after EVERY
// pushName/popName mutation asserts that hasOnStack and the
// htmlAutoCloseOnClose pre-scan give the same answer as a naive linear-scan
// reference. This is what catches a desync between nameStack and the derived
// namePos/hiPrioPos indexes; the golden and growth tests only observe the
// final SAX stream, which a subtly wrong index could still reproduce by
// accident.
func TestStackIndexesMatchNaive(t *testing.T) {
	seen := map[string]struct{}{}
	for name := range htmlEndPriority {
		seen[name] = struct{}{}
	}

	for i, in := range diffCheckAdversarialInputs {
		name := fmt.Sprintf("adversarial_%d", i)
		input := []byte(in)
		t.Run(name, func(t *testing.T) { runStackDiffCheck(t, seen, input) })
	}
	for i, in := range diffCheckFuzzSeeds {
		name := fmt.Sprintf("fuzzseed_%d", i)
		input := []byte(in)
		t.Run(name, func(t *testing.T) { runStackDiffCheck(t, seen, input) })
	}

	dir := "../testdata/libxml2-compat/html"
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("testdata/libxml2-compat/html not found: %v", err)
	}
	for _, fi := range entries {
		fileName := fi.Name()
		if fi.IsDir() || strings.HasSuffix(fileName, ".sax") ||
			strings.HasSuffix(fileName, ".err") || strings.HasSuffix(fileName, ".expected") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, fileName))
		if err != nil {
			t.Fatalf("reading %s: %v", fileName, err)
		}
		t.Run("golden_"+fileName, func(t *testing.T) { runStackDiffCheck(t, seen, data) })
	}
}
