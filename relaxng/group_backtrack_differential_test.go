package relaxng_test

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/relaxng"
	"github.com/stretchr/testify/require"
)

// The differential harness behind the accept/reject safety record at the top of
// group_backtrack_test.go. It emits one deterministic line per case — the case
// identity, the verdict, and the exact error text — so the same file can be run
// on two revisions and the two outputs diffed. Comparing error text and not
// just the verdict is what makes the diff meaningful: a silently reworded
// diagnostic would otherwise pass unnoticed.
//
// Apart from two long-standing helpers in relaxng_test.go (testdataBase and
// partitionCompileErrors) the file touches only the exported API, so it can be
// dropped into an older checkout unchanged. See group_backtrack_test.go for the
// recorded procedure and result.

var (
	diffOut = flag.String("relaxng.differential.out", "",
		"write the group-backtracking differential record to this file; empty runs a small smoke subset and discards the output")
	diffCases = flag.Int("relaxng.differential.cases", 20000,
		"number of randomized group grammars for the differential record")
	diffSeed = flag.Int64("relaxng.differential.seed", 1,
		"seed for the randomized group grammars")
)

// diffSmokeCases and diffSmokeStride shrink the run when no output file is
// requested: every diffSmokeStride'th golden schema and diffSmokeCases random
// grammars, small enough for an ordinary `go test ./relaxng` run. A smoke run
// proves the harness still builds and runs; it is not the record.
const (
	diffSmokeCases  = 200
	diffSmokeStride = 8
)

// diffCorpusDir holds the golden RELAX NG schemas and instances used as the
// first differential corpus.
const diffCorpusDir = testdataBase + "/test"

// diffRandomLabel is the instance label every randomized case validates under.
// It is constant so two cases that fail the same way produce byte-identical
// error text.
const diffRandomLabel = "instance.xml"

// diffAlphabet is the element-name pool the randomized grammars and instances
// draw from. Keeping it tiny makes accepts and rejects both common.
var diffAlphabet = []string{"a", "b", "c"}

func TestGroupBacktrackDifferential(t *testing.T) {
	cases := *diffCases
	stride := 1
	out := io.Discard
	if *diffOut == "" {
		cases = diffSmokeCases
		stride = diffSmokeStride
	} else {
		f, err := os.Create(*diffOut)
		require.NoError(t, err)
		defer func() { _ = f.Close() }()
		w := bufio.NewWriter(f)
		defer func() { _ = w.Flush() }()
		out = w
	}

	corpusLines, valid, invalid := diffRunCorpus(t, out, stride)
	t.Logf("corpus: %d lines (%d VALID, %d INVALID)", corpusLines, valid, invalid)

	randomLines, valid, invalid, texts := diffRunRandom(t, out, cases, *diffSeed)
	t.Logf("random: %d lines (%d VALID, %d INVALID, %d distinct error texts)", randomLines, valid, invalid, texts)
}

// TestGroupBacktrackDifferentialDeterministic proves the harness output is
// reproducible, which is what makes a zero-line diff between two revisions
// meaningful: two runs of the same corpus in the same binary must be
// byte-identical.
func TestGroupBacktrackDifferentialDeterministic(t *testing.T) {
	var first, second strings.Builder
	diffRunCorpus(t, &first, diffSmokeStride*4)
	diffRunRandom(t, &first, diffSmokeCases, 1)
	diffRunCorpus(t, &second, diffSmokeStride*4)
	diffRunRandom(t, &second, diffSmokeCases, 1)
	require.NotEmpty(t, first.String())
	require.Equal(t, first.String(), second.String(), "differential harness output must be deterministic")
}

// diffRunCorpus validates every stride'th golden schema against every golden
// instance in the same directory, emitting one line per pair.
func diffRunCorpus(t *testing.T, out io.Writer, stride int) (lines, valid, invalid int) {
	t.Helper()

	entries, err := os.ReadDir(diffCorpusDir)
	require.NoError(t, err)

	var schemas, instances []string
	for _, e := range entries {
		switch {
		case e.IsDir():
		case strings.HasSuffix(e.Name(), ".rng"):
			schemas = append(schemas, e.Name())
		case strings.HasSuffix(e.Name(), ".xml"):
			instances = append(instances, e.Name())
		}
	}
	sort.Strings(schemas)
	sort.Strings(instances)
	require.NotEmpty(t, schemas)
	require.NotEmpty(t, instances)

	// The instance bytes are cached but re-parsed for every pair, so no
	// validation can observe a tree another validation touched.
	raw := make([][]byte, len(instances))
	for i, name := range instances {
		data, err := os.ReadFile(filepath.Join(diffCorpusDir, name))
		require.NoError(t, err)
		raw[i] = data
	}

	for si, rng := range schemas {
		if si%stride != 0 {
			continue
		}
		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		grammar, err := relaxng.NewCompiler().
			FS(helium.PermissiveFS()).
			Label(rng).
			ErrorHandler(collector).
			CompileFile(t.Context(), filepath.Join(diffCorpusDir, rng))
		_ = collector.Close()
		_, compileErrors := partitionCompileErrors(collector.Errors())
		if err != nil || compileErrors != "" {
			text := compileErrors
			if err != nil {
				text = err.Error() + "|" + compileErrors
			}
			fmt.Fprintf(out, "corpus\t%s\t-\tSCHEMA-ERROR\t%q\n", rng, text)
			lines++
			continue
		}

		for i, xml := range instances {
			doc, err := helium.NewParser().Parse(t.Context(), raw[i])
			if err != nil {
				fmt.Fprintf(out, "corpus\t%s\t%s\tPARSE-ERROR\t%q\n", rng, xml, err.Error())
				lines++
				continue
			}
			verdict, text := diffValidate(t, grammar, doc, xml)
			fmt.Fprintf(out, "corpus\t%s\t%s\t%s\t%q\n", rng, xml, verdict, text)
			lines++
			if verdict == "VALID" {
				valid++
				continue
			}
			invalid++
		}
	}
	return lines, valid, invalid
}

// diffRunRandom generates seeded random group grammars with matching instances
// and emits one line per case.
func diffRunRandom(t *testing.T, out io.Writer, cases int, seed int64) (lines, valid, invalid, texts int) {
	t.Helper()

	// An explicit source, not the global one: the sequence must be identical
	// on every revision the harness is run against.
	rnd := rand.New(rand.NewSource(seed))
	seen := make(map[string]struct{})

	for i := range cases {
		schema, instance := diffRandomCase(rnd)

		schemaDoc, err := helium.NewParser().Parse(t.Context(), []byte(schema))
		require.NoError(t, err, "case %d: generated schema must parse: %s", i, schema)

		collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
		grammar, err := relaxng.NewCompiler().ErrorHandler(collector).Compile(t.Context(), schemaDoc)
		_ = collector.Close()
		_, compileErrors := partitionCompileErrors(collector.Errors())
		if err != nil || compileErrors != "" {
			text := compileErrors
			if err != nil {
				text = err.Error() + "|" + compileErrors
			}
			fmt.Fprintf(out, "random\t%d\t%d\tSCHEMA-ERROR\t%q\n", seed, i, text)
			lines++
			continue
		}

		doc, err := helium.NewParser().Parse(t.Context(), []byte(instance))
		require.NoError(t, err, "case %d: generated instance must parse: %s", i, instance)

		verdict, text := diffValidate(t, grammar, doc, diffRandomLabel)
		fmt.Fprintf(out, "random\t%d\t%d\t%s\t%q\n", seed, i, verdict, text)
		lines++
		if verdict == "VALID" {
			valid++
			continue
		}
		invalid++
		if _, ok := seen[text]; !ok {
			seen[text] = struct{}{}
			texts++
		}
	}
	return lines, valid, invalid, texts
}

// diffValidate validates one document against one grammar and returns the
// verdict plus the exact concatenated error text.
func diffValidate(t *testing.T, grammar *relaxng.Grammar, doc *helium.Document, label string) (verdict, text string) {
	t.Helper()

	collector := helium.NewErrorCollector(t.Context(), helium.ErrorLevelNone)
	err := relaxng.NewValidator(grammar).Label(label).ErrorHandler(collector).Validate(t.Context(), doc)
	_ = collector.Close()
	if err == nil {
		return "VALID", ""
	}

	var b strings.Builder
	for _, ve := range collector.Errors() {
		b.WriteString(ve.Error())
	}
	if !errors.Is(err, relaxng.ErrValidationFailed) {
		return "ERROR", err.Error() + "|" + b.String()
	}
	return "INVALID", b.String()
}

// diffRandomCase builds one seeded random grammar and a matching instance. It
// rotates through three shapes; shape 0 is a bare <group> under <start>, the
// only shape that reaches the naive backtracker.
func diffRandomCase(rnd *rand.Rand) (schema, instance string) {
	const header = `<grammar xmlns="http://relaxng.org/ns/structure/1.0">`
	members := diffRandomMembers(rnd)

	switch rnd.Intn(3) {
	case 0:
		// Bare <group> under <start>: matches the single top-level element.
		schema = header + `<start><group>` + members + `</group></start></grammar>`
		instance = "<" + diffAlphabet[rnd.Intn(len(diffAlphabet))] + "/>"
	case 1:
		// Group as element content.
		schema = header + `<start><element name="root"><group>` + members +
			`</group></element></start></grammar>`
		instance = diffRandomInstance(rnd)
	default:
		// Nested groups as element content.
		schema = header + `<start><element name="root"><group><group>` + members +
			`</group>` + diffRandomMembers(rnd) + `</group></element></start></grammar>`
		instance = diffRandomInstance(rnd)
	}
	return schema, instance
}

// diffRandomMembers builds two to four group members.
func diffRandomMembers(rnd *rand.Rand) string {
	var b strings.Builder
	for range 2 + rnd.Intn(3) {
		b.WriteString(diffRandomMember(rnd))
	}
	return b.String()
}

// diffRandomMember wraps a random pattern in a random repetition operator.
func diffRandomMember(rnd *rand.Rand) string {
	inner := diffRandomPattern(rnd)
	switch rnd.Intn(4) {
	case 0:
		return inner
	case 1:
		return `<zeroOrMore>` + inner + `</zeroOrMore>`
	case 2:
		return `<oneOrMore>` + inner + `</oneOrMore>`
	default:
		return `<optional>` + inner + `</optional>`
	}
}

// diffRandomPattern builds either a single empty element or a choice of two.
func diffRandomPattern(rnd *rand.Rand) string {
	one := diffElementPattern(rnd)
	if rnd.Intn(4) != 0 {
		return one
	}
	return `<choice>` + one + diffElementPattern(rnd) + `</choice>`
}

// diffElementPattern builds one empty element pattern from the alphabet.
func diffElementPattern(rnd *rand.Rand) string {
	return `<element name="` + diffAlphabet[rnd.Intn(len(diffAlphabet))] + `"><empty/></element>`
}

// diffRandomInstance builds a <root> with zero to five random empty children.
func diffRandomInstance(rnd *rand.Rand) string {
	var b strings.Builder
	b.WriteString(`<root>`)
	for range rnd.Intn(6) {
		b.WriteString("<" + diffAlphabet[rnd.Intn(len(diffAlphabet))] + "/>")
	}
	b.WriteString(`</root>`)
	return b.String()
}
