package xslt3_test

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/xslt3"
	"github.com/stretchr/testify/require"
)

// This file is a small, realistic "real stylesheet" integration corpus for
// xslt3. Unlike the W3C conformance suite (which lives in the sibling
// helium-w3c-tests module and emphasizes spec compliance), these cases pin
// end-to-end integration behavior that conformance suites under-exercise:
// cross-file URI resolution (xsl:import / xsl:include / xsl:use-package /
// document()), the three serializer output methods, keys + grouping, a
// large-ish param set threaded through the public API, and — critically —
// reuse of a single compiled *Stylesheet across multiple source documents.
//
// Each case lives under testdata/xslt3-corpus/<name>/ as self-contained real
// stylesheets, modules, inputs, and committed .expected goldens.
//
// Regenerate the goldens with:
//
//	HELIUM_XSLT3_CORPUS_UPDATE=1 go test ./xslt3/ -run TestXSLT3Corpus
//
// The goldens pin the serialized output byte-for-byte, so a serializer
// regression (or a transform regression) fails the test.

const corpusRoot = "testdata/xslt3-corpus"

// corpusBaseURI is the (virtual) base URI every case is compiled with. Relative
// xsl:import / xsl:include / document() hrefs resolve against it via RFC 3986,
// and the confined resolver maps the resulting "corpus:/<file>" URIs to files
// inside the single case directory — nothing outside it is reachable.
const corpusBaseURI = "corpus:/main.xsl"

const corpusURIPrefix = "corpus:/"

// Shared input filenames (each case's golden is <base>.expected).
const (
	corpusInput1 = "input-1.xml"
	corpusInput2 = "input-2.xml"
)

// confinedResolver serves stylesheet modules (xsl:import/include, compile time),
// runtime documents (document()/doc(), transform time), and packages
// (xsl:use-package) — all strictly confined to a single fs.FS root. It doubles
// as a security-posture example: it satisfies all three resolver interfaces
// while refusing any URI that does not map to a valid path within the root.
type confinedResolver struct {
	fsys     fs.FS
	packages map[string]string // package name -> filename within fsys
}

// name maps an incoming "corpus:/<path>" URI to a validated path within the
// root, rejecting anything with another scheme, a query/fragment, or a path
// that escapes the root.
func (r confinedResolver) name(uri string) (string, error) {
	rest, ok := strings.CutPrefix(uri, corpusURIPrefix)
	if !ok {
		return "", fmt.Errorf("refused: %q is outside the confined root %q", uri, corpusURIPrefix)
	}
	if strings.ContainsAny(rest, "?#") {
		return "", fmt.Errorf("refused: %q carries a query or fragment", uri)
	}
	// Absorb any leading "/" so ".." segments cannot climb above the root, then
	// require the result to be a valid (relative, traversal-free) fs path.
	name := strings.TrimPrefix(path.Clean("/"+rest), "/")
	if name == "" || !fs.ValidPath(name) {
		return "", fmt.Errorf("refused: %q does not resolve within the confined root", uri)
	}
	return name, nil
}

func (r confinedResolver) open(uri string) (io.ReadCloser, error) {
	name, err := r.name(uri)
	if err != nil {
		return nil, err
	}
	return r.fsys.Open(name)
}

// Resolve satisfies xslt3.URIResolver (compile time: xsl:import / xsl:include).
func (r confinedResolver) Resolve(uri string) (io.ReadCloser, error) { return r.open(uri) }

// ResolveURI satisfies xpath3.URIResolver (transform time: document() / doc()).
func (r confinedResolver) ResolveURI(uri string) (io.ReadCloser, error) { return r.open(uri) }

// ResolvePackage satisfies xslt3.PackageResolver (compile time: xsl:use-package).
func (r confinedResolver) ResolvePackage(name, _ string) (io.ReadCloser, string, error) {
	file, ok := r.packages[name]
	if !ok {
		return nil, "", fmt.Errorf("refused: unknown package %q", name)
	}
	rc, err := r.fsys.Open(file)
	if err != nil {
		return nil, "", err
	}
	return rc, corpusURIPrefix + file, nil
}

var (
	_ xslt3.URIResolver     = confinedResolver{}
	_ xpath3.URIResolver    = confinedResolver{}
	_ xslt3.PackageResolver = confinedResolver{}
)

type corpusCase struct {
	name string // subdirectory under corpusRoot
	// exercises documents which integration surfaces the case covers.
	exercises string
	// packages maps xsl:use-package names to a filename in the case dir.
	packages map[string]string
	// params builds the global parameter set passed via the public API.
	params func() *xslt3.Parameters
	// inputs are the source XML filenames; the golden for input <base>.xml is
	// <base>.expected in the same directory.
	inputs []string
}

func corpusCases() []corpusCase {
	return []corpusCase{
		{
			name:      "catalog-html",
			exercises: "xsl:import + xsl:include, named templates (call-template), nav/body modes, params, HTML output",
			params: func() *xslt3.Parameters {
				p := xslt3.NewParameters()
				p.SetString("siteTitle", "Corner Grocery")
				p.SetString("currency", "USD")
				p.Set("showSku", xpath3.SingleBoolean(true))
				return p
			},
			inputs: []string{corpusInput1, corpusInput2},
		},
		{
			name:      "sales-report-text",
			exercises: "xsl:include, xsl:key + key(), xsl:for-each-group + sort, params, text output",
			params: func() *xslt3.Parameters {
				p := xslt3.NewParameters()
				p.SetString("reportTitle", "Q1 FY26")
				p.SetString("currencySymbol", "$")
				p.Set("minTotal", xpath3.SingleDouble(0))
				return p
			},
			inputs: []string{corpusInput1, corpusInput2},
		},
		{
			name:      "catalog-merge-xml",
			exercises: "multi-document input via document() (confined runtime resolver), modes, params, indented XML output",
			params: func() *xslt3.Parameters {
				p := xslt3.NewParameters()
				p.SetString("storeName", "Helium Store")
				p.Set("taxRate", xpath3.SingleDouble(0.08))
				return p
			},
			inputs: []string{corpusInput1},
		},
		{
			name:      "sessions-functions-xml",
			exercises: "xsl:include of an xsl:function module, named template with typed required param, params, XML output",
			params: func() *xslt3.Parameters {
				p := xslt3.NewParameters()
				p.SetString("conferenceName", "HeliumConf")
				p.Set("year", xpath3.SingleInteger(2026))
				return p
			},
			inputs: []string{corpusInput1},
		},
		{
			name:      "glossary-package",
			exercises: "xsl:use-package (confined PackageResolver), public xsl:function from the package, for-each-group + sort, XML output",
			packages:  map[string]string{"urn:helium:corpus:text-utils": "text-utils.xsl"},
			params: func() *xslt3.Parameters {
				p := xslt3.NewParameters()
				p.SetString("glossaryTitle", "XSLT Glossary")
				return p
			},
			inputs: []string{corpusInput1},
		},
		{
			name:      "staff-directory-html",
			exercises: "xsl:import precedence with xsl:apply-imports, xsl:key lookup, param, HTML output",
			params: func() *xslt3.Parameters {
				p := xslt3.NewParameters()
				p.SetString("orgName", "Helium Labs")
				return p
			},
			inputs: []string{corpusInput1},
		},
	}
}

// TestXSLT3CorpusResolverConfined asserts the corpus resolver is genuinely
// confined: it serves files inside its root but refuses a foreign scheme, a
// query/fragment, and any path that tries to climb out of the root. This is
// what makes the corpus double as a security-posture example — the resolver
// handed to Compile/Transform cannot be steered outside the trusted directory.
func TestXSLT3CorpusResolverConfined(t *testing.T) {
	t.Parallel()

	// Root the resolver at a case dir that actually contains main.xsl.
	r := confinedResolver{fsys: os.DirFS(filepath.Join(corpusRoot, "catalog-html"))}

	// In-root: allowed.
	rc, err := r.Resolve("corpus:/main.xsl")
	require.NoError(t, err, "in-root URI must resolve")
	require.NoError(t, rc.Close())

	refused := []string{
		"file:///etc/passwd",          // foreign scheme
		"http://example.com/evil.xsl", // foreign scheme
		"corpus:/../secret.xsl",       // traversal out of root
		"corpus:/../../etc/passwd",    // deeper traversal
		"corpus:/main.xsl?x=1",        // query
		"corpus:/main.xsl#frag",       // fragment
	}
	for _, uri := range refused {
		_, err := r.Resolve(uri)
		require.Error(t, err, "resolver must refuse %q", uri)
		// ResolveURI (runtime path) must refuse it too.
		_, err = r.ResolveURI(uri)
		require.Error(t, err, "runtime resolver must refuse %q", uri)
	}
}

func TestXSLT3Corpus(t *testing.T) {
	t.Parallel()

	update := os.Getenv("HELIUM_XSLT3_CORPUS_UPDATE") != ""

	for _, tc := range corpusCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			t.Logf("exercises: %s", tc.exercises)

			caseDir := filepath.Join(corpusRoot, tc.name)
			resolver := confinedResolver{
				fsys:     os.DirFS(caseDir),
				packages: tc.packages,
			}

			// Compile the entry stylesheet ONCE. The same compiled *Stylesheet
			// is reused for every input and every pass below.
			ssBytes, err := os.ReadFile(filepath.Join(caseDir, "main.xsl"))
			require.NoError(t, err)
			ssDoc, err := helium.NewParser().Parse(t.Context(), ssBytes)
			require.NoError(t, err)

			ss, err := xslt3.NewCompiler().
				BaseURI(corpusBaseURI).
				URIResolver(resolver).
				PackageResolver(resolver).
				Compile(t.Context(), ssDoc)
			require.NoError(t, err, "compile %s", tc.name)

			transform := func(input string) string {
				srcBytes, err := os.ReadFile(filepath.Join(caseDir, input))
				require.NoError(t, err)
				src, err := helium.NewParser().Parse(t.Context(), srcBytes)
				require.NoError(t, err)

				out, err := ss.Transform(src).
					URIResolver(resolver).
					GlobalParameters(tc.params()).
					Serialize(t.Context())
				require.NoError(t, err, "transform %s/%s", tc.name, input)
				return out
			}

			// Two full passes over all inputs, interleaved: pass 2 of input-1
			// runs AFTER pass 1 of input-2, so any state leaking from one
			// source document into the reused compiled stylesheet would make a
			// later run of an earlier input diverge from its golden.
			results := make(map[string][2]string)
			for pass := range 2 {
				for _, input := range tc.inputs {
					got := transform(input)
					r := results[input]
					r[pass] = got
					results[input] = r
				}
			}

			for _, input := range tc.inputs {
				base := strings.TrimSuffix(input, filepath.Ext(input))
				goldenPath := filepath.Join(caseDir, base+".expected")

				pass1, pass2 := results[input][0], results[input][1]
				// Reuse contract: the same compiled stylesheet must produce
				// identical output for the same input across repeated,
				// interleaved transforms.
				require.Equal(t, pass1, pass2,
					"%s/%s: reused compiled stylesheet produced different output across passes", tc.name, input)

				if update {
					require.NoError(t, os.WriteFile(goldenPath, []byte(pass1), 0o644))
					continue
				}

				want, err := os.ReadFile(goldenPath)
				require.NoError(t, err, "read golden %s (regenerate with HELIUM_XSLT3_CORPUS_UPDATE=1)", goldenPath)
				require.Equal(t, string(want), pass1,
					"%s/%s: serialized output does not match golden %s", tc.name, input, goldenPath)
			}
		})
	}
}
