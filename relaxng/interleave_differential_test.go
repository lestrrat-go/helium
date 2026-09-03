package relaxng_test

import (
	"bufio"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/relaxng"
	"github.com/stretchr/testify/require"
)

// TestInterleaveDifferential compares two revisions' interleave verdicts
// against libxml2 xmllint as an oracle. It generates 20,000 seeded (seed 1)
// random interleave grammars over three shapes (bare interleave content,
// interleave then a sibling element, a sibling then interleave), with
// branches drawn from three disjoint two-letter alphabets in six composite
// forms (a bare element, group, nested interleave, choice(element,
// group), group+optional, group+zeroOrMore) each optionally wrapped in
// optional/zeroOrMore/oneOrMore, and a <text/> branch one case in four. It
// prints one line per case (index, verdict, quoted error text, schema,
// document) to -relaxng.idiff.out, and, with -relaxng.idiff.xmllint=<dir>,
// also runs the local xmllint against the same schema/document and records
// its verdict.
//
// Procedure. Run on the merge base of this branch with origin/main (main,
// before the RELAX NG §7.4 partition engine) and on this branch (head):
//
//	BASE=$(git merge-base HEAD origin/main)
//	mkdir -p /tmp/base && git archive "$BASE" | tar -x -C /tmp/base
//	cp relaxng/interleave_differential_test.go /tmp/base/relaxng/
//	go -C /tmp/base test ./relaxng -run '^TestInterleaveDifferential$' \
//	    -relaxng.idiff.out=/tmp/idiff-base.txt
//	go test ./relaxng -run '^TestInterleaveDifferential$' \
//	    -relaxng.idiff.out=/tmp/idiff-head.txt -relaxng.idiff.xmllint=/tmp/idiff-xl
//
// Result, against xmllint 2.9.14: base agrees with xmllint on 16,069/20,000
// cases; head agrees on 20,000/20,000. Every VALID->INVALID and
// INVALID->VALID flip (9 and 19 cases) agrees with xmllint on the head side;
// every ->SCHEMA-ERROR flip (47 VALID->SCHEMA-ERROR, 3,856
// INVALID->SCHEMA-ERROR) is "Element or text conflicts in interleave", also
// matching xmllint; a head-vs-xmllint mismatch listing (comparing every head
// verdict against the recorded xmllint verdict) is empty.

var (
	idiffOut     = flag.String("relaxng.idiff.out", "", "write the interleave differential record here")
	idiffCases   = flag.Int("relaxng.idiff.cases", 20000, "number of random interleave grammars")
	idiffSeed    = flag.Int64("relaxng.idiff.seed", 1, "seed")
	idiffXmllint = flag.String("relaxng.idiff.xmllint", "", "dir: also write each case and record xmllint's verdict in <dir>/xmllint.txt")
)

var idiffAlpha = [][]string{{"a", "b"}, {"c", "d"}, {"e", "f"}}

func TestInterleaveDifferential(t *testing.T) {
	if *idiffOut == "" {
		t.Skip("no -relaxng.idiff.out")
	}
	f, err := os.Create(*idiffOut)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()
	w := bufio.NewWriter(f)
	defer func() { _ = w.Flush() }()

	var xw *bufio.Writer
	if *idiffXmllint != "" {
		require.NoError(t, os.MkdirAll(*idiffXmllint, 0o755))
		xf, err := os.Create(filepath.Join(*idiffXmllint, "xmllint.txt"))
		require.NoError(t, err)
		defer func() { _ = xf.Close() }()
		xw = bufio.NewWriter(xf)
		defer func() { _ = xw.Flush() }()
	}

	rnd := rand.New(rand.NewSource(*idiffSeed))
	for i := range *idiffCases {
		schema, instance := idiffCase(rnd)
		schemaDoc, err := helium.NewParser().Parse(t.Context(), []byte(schema))
		require.NoError(t, err)
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		grammar, err := relaxng.NewCompiler().Label("s.rng").ErrorHandler(collector).Compile(t.Context(), schemaDoc)
		_ = collector.Close()
		_, compileErrors := partitionCompileErrors(collector.Errors())
		if err != nil || compileErrors != "" {
			fmt.Fprintf(w, "idiff\t%d\tSCHEMA-ERROR\t%q\t%s\t%s\n", i, compileErrors, schema, instance)
		} else {
			doc, err := helium.NewParser().Parse(t.Context(), []byte(instance))
			require.NoError(t, err)
			verdict, text := diffValidate(t, grammar, doc, "d.xml")
			fmt.Fprintf(w, "idiff\t%d\t%s\t%q\t%s\t%s\n", i, verdict, text, schema, instance)
		}
		if xw == nil {
			continue
		}
		sp := filepath.Join(*idiffXmllint, "s.rng")
		dp := filepath.Join(*idiffXmllint, "d.xml")
		require.NoError(t, os.WriteFile(sp, []byte(schema), 0o644))
		require.NoError(t, os.WriteFile(dp, []byte(instance), 0o644))
		cmd := exec.CommandContext(t.Context(), "xmllint", "--noout", "--relaxng", sp, dp)
		cmd.Dir = *idiffXmllint
		outb, runErr := cmd.CombinedOutput()
		out := string(outb)
		v := "VALID"
		switch {
		case strings.Contains(out, "failed to compile"):
			v = "SCHEMA-ERROR"
		case runErr != nil:
			v = "INVALID"
		}
		fmt.Fprintf(xw, "%d\t%s\t%q\n", i, v, out)
	}
}

func idiffElem(name string) string { return `<element name="` + name + `"><empty/></element>` }

func idiffBranch(rnd *rand.Rand, alpha []string) string {
	x := idiffElem(alpha[rnd.Intn(2)])
	y := idiffElem(alpha[rnd.Intn(2)])
	var inner string
	switch rnd.Intn(6) {
	case 0:
		inner = x
	case 1:
		inner = `<group>` + x + y + `</group>`
	case 2:
		inner = `<interleave>` + x + y + `</interleave>`
	case 3:
		inner = `<choice>` + x + `<group>` + x + y + `</group></choice>`
	case 4:
		inner = `<group>` + x + `<optional>` + y + `</optional></group>`
	default:
		inner = `<group>` + x + `<zeroOrMore>` + y + `</zeroOrMore></group>`
	}
	switch rnd.Intn(4) {
	case 0:
		return inner
	case 1:
		return `<optional>` + inner + `</optional>`
	case 2:
		return `<zeroOrMore>` + inner + `</zeroOrMore>`
	default:
		return `<oneOrMore>` + inner + `</oneOrMore>`
	}
}

func idiffCase(rnd *rand.Rand) (schema, instance string) {
	nb := 2 + rnd.Intn(2)
	var b strings.Builder
	b.WriteString(`<interleave>`)
	for i := range nb {
		b.WriteString(idiffBranch(rnd, idiffAlpha[i]))
	}
	mixed := rnd.Intn(4) == 0
	if mixed {
		b.WriteString(`<text/>`)
	}
	b.WriteString(`</interleave>`)
	body := b.String()
	var content string
	switch rnd.Intn(3) {
	case 0:
		content = body
	case 1:
		content = `<group>` + body + idiffElem("g") + `</group>`
	default:
		content = `<group>` + idiffElem("g") + body + `</group>`
	}
	schema = `<grammar xmlns="http://relaxng.org/ns/structure/1.0"><start><element name="root">` + content +
		`</element></start></grammar>`
	letters := []string{"a", "b", "c", "d", "e", "f", "g"}
	var d strings.Builder
	d.WriteString(`<root>`)
	for range rnd.Intn(7) {
		if mixed && rnd.Intn(3) == 0 {
			d.WriteString("t")
		}
		d.WriteString("<" + letters[rnd.Intn(len(letters))] + "/>")
	}
	d.WriteString(`</root>`)
	return schema, d.String()
}
