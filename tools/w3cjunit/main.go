// Command w3cjunit converts `go test -json` output for the xslt3 W3C XSLT 3.0
// conformance suite into a JUnit XML report.
//
// Usage:
//
//	go test -json -run TestW3C ./xslt3/ > w3c.json
//	go run ./tools/w3cjunit \
//	  -suite-commit "$(git -C testdata/xslt30/source rev-parse HEAD)" \
//	  -in w3c.json -out xslt3/w3c-results.xml
//
// With no -in the report is read from stdin; with no -out it is written to stdout.
//
// Only leaf subtests (those whose name contains "/") are emitted as <testcase>
// elements, so each W3C catalog test maps to exactly one case. The conformance
// suite provenance (repo + commit) is recorded as <properties> on the suite.
package main

import (
	"bufio"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
)

// testEvent mirrors the JSON objects emitted by `go test -json`.
type testEvent struct {
	Action  string  `json:"Action"`
	Package string  `json:"Package"`
	Test    string  `json:"Test"`
	Elapsed float64 `json:"Elapsed"`
	Output  string  `json:"Output"`
}

type caseResult struct {
	name    string
	status  string // "pass", "fail", "skip"
	elapsed float64
	output  []string
}

// JUnit XML model.
type junitSuites struct {
	XMLName  xml.Name     `xml:"testsuites"`
	Tests    int          `xml:"tests,attr"`
	Failures int          `xml:"failures,attr"`
	Skipped  int          `xml:"skipped,attr"`
	Time     string       `xml:"time,attr"`
	Suites   []junitSuite `xml:"testsuite"`
}

type junitSuite struct {
	Name       string          `xml:"name,attr"`
	Tests      int             `xml:"tests,attr"`
	Failures   int             `xml:"failures,attr"`
	Skipped    int             `xml:"skipped,attr"`
	Time       string          `xml:"time,attr"`
	Properties []junitProperty `xml:"properties>property"`
	Cases      []junitCase     `xml:"testcase"`
}

type junitProperty struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

type junitCase struct {
	Name      string        `xml:"name,attr"`
	Classname string        `xml:"classname,attr"`
	Time      string        `xml:"time,attr"`
	Skipped   *junitMessage `xml:"skipped,omitempty"`
	Failure   *junitMessage `xml:"failure,omitempty"`
}

type junitMessage struct {
	Message string `xml:"message,attr"`
	Body    string `xml:",chardata"`
}

func main() {
	suiteCommit := flag.String("suite-commit", "", "conformance suite commit ID (recorded as a property)")
	suiteRepo := flag.String("suite-repo", "https://github.com/w3c/xslt30-test.git", "conformance suite repository URL")
	suiteName := flag.String("name", "xslt3-w3c-conformance", "JUnit testsuite name")
	in := flag.String("in", "", "input `go test -json` file (default stdin)")
	out := flag.String("out", "", "output file (default stdout)")
	flag.Parse()

	input := os.Stdin
	if *in != "" {
		f, err := os.Open(*in)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to open %s: %s\n", *in, err)
			os.Exit(1)
		}
		defer f.Close()
		input = f
	}

	cases, err := parse(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse test output: %s\n", err)
		os.Exit(1)
	}

	doc := build(cases, *suiteName, *suiteRepo, *suiteCommit)
	body, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to marshal JUnit XML: %s\n", err)
		os.Exit(1)
	}
	body = append([]byte(xml.Header), append(body, '\n')...)

	if *out == "" {
		os.Stdout.Write(body)
		return
	}
	if err := os.WriteFile(*out, body, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write %s: %s\n", *out, err)
		os.Exit(1)
	}
}

// parse reads `go test -json` events and returns one caseResult per leaf
// subtest, ordered by name.
func parse(r *os.File) ([]caseResult, error) {
	byName := map[string]*caseResult{}
	hasChild := map[string]struct{}{}

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 1024*1024), 16*1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 || line[0] != '{' {
			continue
		}
		var ev testEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			continue
		}
		if ev.Test == "" {
			continue
		}
		if idx := strings.LastIndex(ev.Test, "/"); idx >= 0 {
			hasChild[ev.Test[:idx]] = struct{}{}
		}
		c := byName[ev.Test]
		if c == nil {
			c = &caseResult{name: ev.Test}
			byName[ev.Test] = c
		}
		switch ev.Action {
		case "output":
			c.output = append(c.output, ev.Output)
		case "pass", "fail", "skip":
			c.status = ev.Action
			c.elapsed = ev.Elapsed
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	var leaves []caseResult
	for name, c := range byName {
		// Only leaf subtests: must be a subtest and have no children itself.
		if !strings.Contains(name, "/") {
			continue
		}
		if _, parent := hasChild[name]; parent {
			continue
		}
		leaves = append(leaves, *c)
	}
	sort.Slice(leaves, func(i, j int) bool { return leaves[i].name < leaves[j].name })
	return leaves, nil
}

func build(cases []caseResult, name, repo, commit string) junitSuites {
	suite := junitSuite{
		Name: name,
		Properties: []junitProperty{
			{Name: "conformance-suite-repo", Value: repo},
			{Name: "conformance-suite-commit", Value: commit},
		},
	}
	var total float64
	for _, c := range cases {
		classname, leaf := splitClass(c.name)
		jc := junitCase{Name: leaf, Classname: classname, Time: fmt.Sprintf("%.3f", c.elapsed)}
		switch c.status {
		case "skip":
			jc.Skipped = &junitMessage{Message: skipReason(c.output)}
			suite.Skipped++
		case "fail":
			jc.Failure = &junitMessage{Message: "test failed", Body: strings.Join(c.output, "")}
			suite.Failures++
		}
		suite.Cases = append(suite.Cases, jc)
		suite.Tests++
		total += c.elapsed
	}
	suite.Time = fmt.Sprintf("%.3f", total)
	return junitSuites{
		Tests:    suite.Tests,
		Failures: suite.Failures,
		Skipped:  suite.Skipped,
		Time:     suite.Time,
		Suites:   []junitSuite{suite},
	}
}

// splitClass splits "TestW3C_match/match-054" into classname "TestW3C_match"
// and leaf "match-054". Nested subtests keep their intermediate path in the
// leaf name.
func splitClass(full string) (string, string) {
	class, leaf, found := strings.Cut(full, "/")
	if !found {
		return full, full
	}
	return class, leaf
}

// skipReason extracts the t.Skip message from a leaf test's output lines. Go
// emits it as "    <file>_test.go:<line>: <reason>"; we return the reason.
func skipReason(output []string) string {
	for _, line := range output {
		trimmed := strings.TrimSpace(line)
		if !strings.Contains(trimmed, "_test.go:") {
			continue
		}
		if _, reason, found := strings.Cut(trimmed, ": "); found {
			return strings.TrimSpace(reason)
		}
	}
	return ""
}
