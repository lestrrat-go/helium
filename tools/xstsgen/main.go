// Command xstsgen parses the W3C XML Schema Test Suite (XSTS) and generates
// table-driven Go test files for the xsd package, restricted to the XSD 1.1
// subset of the suite.
//
// Usage:
//
//	go run ./tools/xstsgen
//
// Prerequisites:
//
//	The upstream suite must be cloned at testdata/xsdtests/source/ (gitignored).
//
// Output:
//
//	xsd/w3c_xsts_<contributor>_gen_test.go   (table-driven cases per contributor)
//
// The schema/instance file contents are EMBEDDED directly into the generated Go
// source, so the committed tests run with NO clone present.
package main

import (
	"encoding/xml"
	"fmt"
	"go/format"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	suiteRoot = "testdata/xsdtests/source"
	suiteFile = "testdata/xsdtests/source/suite.xml"
	outDir    = "xsd"
)

// ---- suite.xml ----

type suiteDoc struct {
	TestSetRefs []testSetRef `xml:"testSetRef"`
}

type testSetRef struct {
	Href string `xml:"href,attr"`
}

// ---- *.testSet ----

type testSet struct {
	Name       string      `xml:"name,attr"`
	Version    string      `xml:"version,attr"`
	TestGroups []testGroup `xml:"testGroup"`
}

type testGroup struct {
	Name         string         `xml:"name,attr"`
	Version      string         `xml:"version,attr"`
	SchemaTest   *schemaTest    `xml:"schemaTest"`
	InstanceTest []instanceTest `xml:"instanceTest"`
}

type schemaTest struct {
	Documents []docRef   `xml:"schemaDocument"`
	Expected  []expected `xml:"expected"`
}

type instanceTest struct {
	Name     string     `xml:"name,attr"`
	Document docRef     `xml:"instanceDocument"`
	Expected []expected `xml:"expected"`
}

type docRef struct {
	Href string `xml:"href,attr"`
}

type expected struct {
	Validity string `xml:"validity,attr"`
	Version  string `xml:"version,attr"`
}

// ---- emitted model ----

type genCase struct {
	ID          string
	SchemaRel   string
	SchemaValid bool
	Files       map[string]string
	Instances   []genInstance
}

type genInstance struct {
	Name  string
	Rel   string
	Valid bool
}

var schemaLocationRe = regexp.MustCompile(`schemaLocation\s*=\s*["']([^"']+)["']`)

func main() {
	log.SetFlags(0)

	suiteBytes, err := os.ReadFile(suiteFile)
	if err != nil {
		log.Fatalf("read suite.xml: %v", err)
	}
	var suite suiteDoc
	if err := xml.Unmarshal(suiteBytes, &suite); err != nil {
		log.Fatalf("parse suite.xml: %v", err)
	}

	// contributor -> ordered list of cases
	byContributor := map[string][]genCase{}
	contributorOrder := []string{}

	var (
		setsScanned int
		groupsOut   int
	)

	for _, ref := range suite.TestSetRefs {
		href := ref.Href
		if href == "" {
			continue
		}
		tsPath := path.Join(suiteRoot, href)
		tsBytes, err := os.ReadFile(tsPath)
		if err != nil {
			// missing testSet file: skip
			continue
		}
		setsScanned++

		var ts testSet
		if err := xml.Unmarshal(tsBytes, &ts); err != nil {
			log.Printf("warning: parse %s: %v", tsPath, err)
			continue
		}

		// dir of the .testSet file, relative to suite root, slash-style
		tsDir := path.Dir(href)
		contributor := contributorName(href)

		for _, g := range ts.TestGroups {
			if !is11(g.Version, ts.Version) {
				continue
			}
			gc, ok := buildCase(href, tsDir, g)
			if !ok {
				continue
			}
			if _, seen := byContributor[contributor]; !seen {
				contributorOrder = append(contributorOrder, contributor)
			}
			byContributor[contributor] = append(byContributor[contributor], gc)
			groupsOut++
		}
	}

	// Emit per-contributor files.
	sort.Strings(contributorOrder)
	var totalFiles, totalBytes int
	for _, contributor := range contributorOrder {
		cases := byContributor[contributor]
		nf, nb := emitFile(contributor, cases)
		totalFiles += nf
		totalBytes += nb
	}

	fmt.Printf("xstsgen: testSets scanned=%d, 1.1 groups emitted=%d, embedded files=%d, embedded bytes=%d\n",
		setsScanned, groupsOut, totalFiles, totalBytes)
}

// contributorName maps a testSetRef href's first path segment to a contributor
// key, stripping a trailing "Meta" (e.g. ibmMeta -> ibm, wgMeta -> wg).
func contributorName(href string) string {
	seg := href
	if i := strings.IndexByte(seg, '/'); i >= 0 {
		seg = seg[:i]
	}
	seg = strings.TrimSuffix(seg, "Meta")
	if seg == "" {
		seg = "misc"
	}
	return seg
}

func is11(groupVersion, setVersion string) bool {
	return groupVersion == "1.1" || setVersion == "1.1"
}

// pickValidity selects the version-aware expected validity. Returns (valid, ok).
func pickValidity(exps []expected) (bool, bool) {
	if len(exps) == 0 {
		return false, false
	}
	var chosen *expected
	for i := range exps {
		if exps[i].Version == "1.1" {
			chosen = &exps[i]
			break
		}
	}
	if chosen == nil {
		for i := range exps {
			if exps[i].Version == "" {
				chosen = &exps[i]
				break
			}
		}
	}
	if chosen == nil {
		chosen = &exps[0]
	}
	switch chosen.Validity {
	case "valid":
		return true, true
	case "invalid":
		return false, true
	default:
		return false, false
	}
}

// resolveRel resolves an href (relative to baseDir, both slash-style and
// suite-root-relative) into a clean suite-root-relative slash path.
func resolveRel(baseDir, href string) string {
	return path.Clean(path.Join(baseDir, href))
}

// buildCase assembles a genCase for a 1.1 testGroup, embedding the schema (and
// its transitive includes) plus all instance documents. Returns (case, ok);
// ok is false when there is no usable schemaTest or the primary schema is
// missing on disk.
func buildCase(href, tsDir string, g testGroup) (genCase, bool) {
	if g.SchemaTest == nil || len(g.SchemaTest.Documents) == 0 {
		return genCase{}, false
	}
	schemaValid, ok := pickValidity(g.SchemaTest.Expected)
	if !ok {
		return genCase{}, false
	}

	files := map[string]string{}

	primaryRel := resolveRel(tsDir, g.SchemaTest.Documents[0].Href)

	// BFS over schema documents + transitive includes.
	queue := []string{}
	enqueued := map[string]bool{}
	add := func(rel string) {
		if !enqueued[rel] {
			enqueued[rel] = true
			queue = append(queue, rel)
		}
	}
	for _, d := range g.SchemaTest.Documents {
		add(resolveRel(tsDir, d.Href))
	}

	primaryFound := false
	for len(queue) > 0 {
		rel := queue[0]
		queue = queue[1:]
		data, err := os.ReadFile(filepath.Join(suiteRoot, filepath.FromSlash(rel)))
		if err != nil {
			// missing include: skip (embed only what exists)
			continue
		}
		if rel == primaryRel {
			primaryFound = true
		}
		files[rel] = string(data)

		// scan for transitive schemaLocation references
		dir := path.Dir(rel)
		for _, m := range schemaLocationRe.FindAllSubmatch(data, -1) {
			loc := string(m[1])
			if loc == "" || isAbsoluteURL(loc) {
				continue
			}
			add(resolveRel(dir, loc))
		}
	}

	if !primaryFound {
		// primary schema missing → skip the group
		return genCase{}, false
	}

	gc := genCase{
		ID:          href + "/" + g.Name,
		SchemaRel:   primaryRel,
		SchemaValid: schemaValid,
		Files:       files,
	}

	for _, it := range g.InstanceTest {
		valid, ok := pickValidity(it.Expected)
		if !ok {
			continue
		}
		rel := resolveRel(tsDir, it.Document.Href)
		data, err := os.ReadFile(filepath.Join(suiteRoot, filepath.FromSlash(rel)))
		if err != nil {
			// missing instance: record nothing for it, skip
			continue
		}
		files[rel] = string(data)
		gc.Instances = append(gc.Instances, genInstance{
			Name:  it.Name,
			Rel:   rel,
			Valid: valid,
		})
	}

	return gc, true
}

func isAbsoluteURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

// goLiteral emits a Go string literal: a raw (backtick) string when safe,
// otherwise a strconv.Quote'd interpreted string.
func goLiteral(s string) string {
	if strings.ContainsRune(s, '`') {
		return strconv.Quote(s)
	}
	// Raw string literals cannot contain a carriage return faithfully on all
	// toolchains' gofmt round-trips; fall back to quoting when present.
	if strings.ContainsRune(s, '\r') {
		return strconv.Quote(s)
	}
	return "`" + s + "`"
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func emitFile(contributor string, cases []genCase) (int, int) {
	varName := "xsts" + titleCase(contributor) + "Cases"

	var b strings.Builder
	b.WriteString("// Code generated by tools/xstsgen; DO NOT EDIT.\n\n")
	b.WriteString("package xsd_test\n\n")
	fmt.Fprintf(&b, "var %s = []xstsCase{\n", varName)

	var totalFiles, totalBytes int
	for _, c := range cases {
		b.WriteString("\t{\n")
		fmt.Fprintf(&b, "\t\tID: %s,\n", strconv.Quote(c.ID))
		fmt.Fprintf(&b, "\t\tSchemaRel: %s,\n", strconv.Quote(c.SchemaRel))
		fmt.Fprintf(&b, "\t\tSchemaValid: %t,\n", c.SchemaValid)

		// Files map (sorted keys for stable output)
		b.WriteString("\t\tFiles: map[string]string{\n")
		keys := make([]string, 0, len(c.Files))
		for k := range c.Files {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			content := c.Files[k]
			totalFiles++
			totalBytes += len(content)
			fmt.Fprintf(&b, "\t\t\t%s: %s,\n", strconv.Quote(k), goLiteral(content))
		}
		b.WriteString("\t\t},\n")

		if len(c.Instances) > 0 {
			b.WriteString("\t\tInstances: []xstsInstance{\n")
			for _, in := range c.Instances {
				fmt.Fprintf(&b, "\t\t\t{Name: %s, Rel: %s, Valid: %t},\n",
					strconv.Quote(in.Name), strconv.Quote(in.Rel), in.Valid)
			}
			b.WriteString("\t\t},\n")
		}

		b.WriteString("\t},\n")
	}
	b.WriteString("}\n")

	outPath := filepath.Join(outDir, "w3c_xsts_"+contributor+"_gen_test.go")
	src := b.String()
	formatted, err := format.Source([]byte(src))
	if err != nil {
		// write unformatted to aid debugging, then fail
		_ = os.WriteFile(outPath, []byte(src), 0o644)
		log.Fatalf("gofmt failed for %s (raw written): %v", outPath, err)
	}
	if err := os.WriteFile(outPath, formatted, 0o644); err != nil {
		log.Fatalf("write %s: %v", outPath, err)
	}
	return totalFiles, totalBytes
}
