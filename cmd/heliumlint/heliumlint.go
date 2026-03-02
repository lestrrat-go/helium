package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/c14n"
	"github.com/lestrrat-go/helium/catalog"
	"github.com/lestrrat-go/helium/internal/cliutil"
	"github.com/lestrrat-go/helium/xinclude"
	"github.com/lestrrat-go/helium/xpath"
	"github.com/lestrrat-go/helium/xsd"
)

// Exit codes matching xmllint conventions.
const (
	exitOK         = 0
	exitErr        = 1
	exitValidation = 3
	exitReadFile   = 4
	exitSchemaComp = 5
	exitXPath      = 10
)

type config struct {
	// Parser flags
	parseOptions helium.ParseOption

	// Feature integration
	doXInclude bool
	c14nMode   int // 0=off, 1=c14n10, 2=c14n11, 3=exc-c14n
	schemaFile string
	xpathExpr  string
	catalogs   bool
	noCatalogs bool
	pathDirs   string

	// Output control
	noout      bool
	format     bool
	outputFile string
	encode     string
	pretty     int // -1=unset, 0=none, 1=indent, 2=indent+attrs

	// Behavioral
	quiet   bool
	timing  bool
	dropdtd bool
	repeat  int

	// Info
	version bool
}

func main() {
	os.Exit(run())
}

func showVersion() {
	fmt.Fprintf(os.Stderr, "heliumlint: using helium version %s\n", helium.Version)
}

func showUsage() {
	fmt.Fprintf(os.Stderr, `Usage : heliumlint [options] XMLfiles ...
	Parse the XML files and output the result of the parsing
	--version : display the version of the XML library used
	--recover : output what was parsable on broken XML documents
	--noent : substitute entity references by their value
	--loaddtd : fetch external DTD
	--dtdattr : loaddtd + populate tree with inherited attributes
	--valid : validate the document with the DTD
	--nowarning : do not emit warnings from parser/validator
	--pedantic : enable pedantic error reporting
	--noblanks : drop (ignorable?) blanks spaces
	--nsclean : remove redundant namespace declarations
	--nocdata : replace CDATA section by equivalent text nodes
	--nonet : refuse to fetch DTDs or entities over network
	--huge : remove any internal arbitrary parser limits
	--noenc : ignore any encoding specified inside the document
	--noxincludenode : do not generate XInclude START/END nodes
	--nofixup-base-uris : do not fixup xml:base URIs in XInclude
	--noout : do not print the result tree
	--format : reformat/reindent the output
	--pretty LEVEL : pretty-print the output (0=none, 1=format, 2=format+attrs)
	--encode ENCODING : output in the given encoding
	--output FILE : save to a given file
	--c14n : save in W3C canonical format v1.0 (with comments)
	--c14n11 : save in W3C canonical format v1.1 (with comments)
	--exc-c14n : save in W3C exclusive canonical format (with comments)
	--xinclude : do XInclude processing
	--schema FILE : do validation against the WXS schema
	--xpath EXPR : evaluate the XPath expression, imply --noout
	--catalogs : use catalogs from $XML_CATALOG_FILES
	--nocatalogs : do not use any catalogs
	--path DIRS : set search path for DTD/entities (colon-separated)
	--quiet : suppress non-error output
	--timing : print timing information to stderr
	--dropdtd : remove the DOCTYPE of the result
	--repeat N : parse N times for benchmarking
`)
}

func parseArgs(args []string) (*config, []string) {
	cfg := &config{
		pretty: -1,
		repeat: 1,
	}
	var files []string

	for i := 0; i < len(args); i++ {
		arg := args[i]

		// Accept both -flag and --flag (matching xmllint)
		if strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "--") && len(arg) > 1 {
			arg = "-" + arg
		}

		switch arg {
		case "--version":
			cfg.version = true
		case "--recover":
			cfg.parseOptions.Set(helium.ParseRecover)
		case "--noent":
			cfg.parseOptions.Set(helium.ParseNoEnt)
		case "--loaddtd":
			cfg.parseOptions.Set(helium.ParseDTDLoad)
		case "--dtdattr":
			cfg.parseOptions.Set(helium.ParseDTDAttr)
			cfg.parseOptions.Set(helium.ParseDTDLoad)
		case "--valid":
			cfg.parseOptions.Set(helium.ParseDTDValid)
			cfg.parseOptions.Set(helium.ParseDTDLoad)
		case "--nowarning":
			cfg.parseOptions.Set(helium.ParseNoWarning)
		case "--pedantic":
			cfg.parseOptions.Set(helium.ParsePedantic)
		case "--noblanks":
			cfg.parseOptions.Set(helium.ParseNoBlanks)
		case "--nsclean":
			cfg.parseOptions.Set(helium.ParseNsClean)
		case "--nocdata":
			cfg.parseOptions.Set(helium.ParseNoCDATA)
		case "--nonet":
			cfg.parseOptions.Set(helium.ParseNoNet)
		case "--huge":
			cfg.parseOptions.Set(helium.ParseHuge)
		case "--noenc":
			cfg.parseOptions.Set(helium.ParseIgnoreEnc)
		case "--noxincludenode":
			cfg.parseOptions.Set(helium.ParseNoXIncNode)
		case "--nofixup-base-uris":
			cfg.parseOptions.Set(helium.ParseNoBaseFix)
		case "--noout":
			cfg.noout = true
		case "--format":
			cfg.format = true
		case "--c14n":
			cfg.c14nMode = 1
		case "--c14n11":
			cfg.c14nMode = 2
		case "--exc-c14n":
			cfg.c14nMode = 3
		case "--xinclude":
			cfg.doXInclude = true
			cfg.parseOptions.Set(helium.ParseXInclude)
		case "--catalogs":
			cfg.catalogs = true
		case "--nocatalogs":
			cfg.noCatalogs = true
		case "--quiet":
			cfg.quiet = true
		case "--timing":
			cfg.timing = true
		case "--dropdtd":
			cfg.dropdtd = true

		// Flags that take a value argument
		case "--schema":
			i++
			if i >= len(args) {
				fmt.Fprintf(os.Stderr, "heliumlint: --schema requires an argument\n")
				return nil, nil
			}
			cfg.schemaFile = args[i]
		case "--xpath":
			i++
			if i >= len(args) {
				fmt.Fprintf(os.Stderr, "heliumlint: --xpath requires an argument\n")
				return nil, nil
			}
			cfg.xpathExpr = args[i]
			cfg.noout = true
		case "--output":
			i++
			if i >= len(args) {
				fmt.Fprintf(os.Stderr, "heliumlint: --output requires an argument\n")
				return nil, nil
			}
			cfg.outputFile = args[i]
		case "--encode":
			i++
			if i >= len(args) {
				fmt.Fprintf(os.Stderr, "heliumlint: --encode requires an argument\n")
				return nil, nil
			}
			cfg.encode = args[i]
		case "--pretty":
			i++
			if i >= len(args) {
				fmt.Fprintf(os.Stderr, "heliumlint: --pretty requires an argument\n")
				return nil, nil
			}
			n, err := strconv.Atoi(args[i])
			if err != nil {
				fmt.Fprintf(os.Stderr, "heliumlint: --pretty: invalid argument %q\n", args[i])
				return nil, nil
			}
			cfg.pretty = n
		case "--path":
			i++
			if i >= len(args) {
				fmt.Fprintf(os.Stderr, "heliumlint: --path requires an argument\n")
				return nil, nil
			}
			cfg.pathDirs = args[i]
		case "--repeat":
			i++
			if i >= len(args) {
				fmt.Fprintf(os.Stderr, "heliumlint: --repeat requires an argument\n")
				return nil, nil
			}
			n, err := strconv.Atoi(args[i])
			if err != nil || n < 1 {
				fmt.Fprintf(os.Stderr, "heliumlint: --repeat: invalid argument %q\n", args[i])
				return nil, nil
			}
			cfg.repeat = n

		default:
			if strings.HasPrefix(arg, "--") {
				fmt.Fprintf(os.Stderr, "heliumlint: unrecognized option %s\n", arg)
				return nil, nil
			}
			files = append(files, arg)
		}
	}

	return cfg, files
}

func run() int {
	cfg, files := parseArgs(os.Args[1:])
	if cfg == nil {
		showUsage()
		return exitErr
	}

	if cfg.version {
		showVersion()
		return exitOK
	}

	// Determine format settings
	if cfg.pretty >= 1 {
		cfg.format = true
	}

	// Collect inputs
	var inputs []namedInput
	if len(files) > 0 {
		for _, f := range files {
			inputs = append(inputs, namedInput{name: f})
		}
	} else if !cliutil.IsTty(os.Stdin.Fd()) {
		inputs = append(inputs, namedInput{name: "-", stdin: true})
	} else {
		showUsage()
		return exitErr
	}

	// Load catalogs if requested
	var cat *catalog.Catalog
	if cfg.catalogs && !cfg.noCatalogs {
		var err error
		cat, err = loadCatalogFromEnv()
		if err != nil {
			fmt.Fprintf(os.Stderr, "heliumlint: %s\n", err)
		}
	}

	// Compile schema if requested
	var schema *xsd.Schema
	if cfg.schemaFile != "" {
		var t0 time.Time
		if cfg.timing {
			t0 = time.Now()
		}
		var err error
		schema, err = compileSchema(cfg)
		if cfg.timing {
			fmt.Fprintf(os.Stderr, "Compiling schema took %s\n", time.Since(t0))
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			return exitSchemaComp
		}
	}

	// Determine output destination
	var out io.Writer = os.Stdout
	if cfg.outputFile != "" {
		f, err := os.Create(cfg.outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "heliumlint: %s\n", err)
			return exitErr
		}
		defer func() { _ = f.Close() }()
		out = f
	}

	exitCode := exitOK
	for _, input := range inputs {
		code := processInput(cfg, input, cat, schema, out)
		if code != exitOK {
			exitCode = code
		}
	}
	return exitCode
}

func loadCatalogFromEnv() (*catalog.Catalog, error) {
	envFiles := os.Getenv("XML_CATALOG_FILES")
	if envFiles == "" {
		return nil, nil
	}
	for _, f := range strings.Split(envFiles, " ") {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		c, err := catalog.Load(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "heliumlint: failed to load catalog %s: %s\n", f, err)
			continue
		}
		return c, nil
	}
	return nil, nil
}

func compileSchema(cfg *config) (*xsd.Schema, error) {
	return xsd.CompileFile(cfg.schemaFile)
}

type namedInput struct {
	name  string
	stdin bool
}

func processInput(cfg *config, input namedInput, cat *catalog.Catalog, schema *xsd.Schema, out io.Writer) int {
	// Read input
	var buf []byte
	var err error
	if input.stdin {
		buf, err = io.ReadAll(os.Stdin)
	} else {
		buf, err = os.ReadFile(input.name)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "heliumlint: %s\n", err)
		return exitReadFile
	}

	// Parse (possibly repeated for benchmarking)
	var doc *helium.Document
	for rep := 0; rep < cfg.repeat; rep++ {
		var t0 time.Time
		if cfg.timing {
			t0 = time.Now()
		}

		p := helium.NewParser()
		p.SetOption(cfg.parseOptions)
		if !input.stdin {
			p.SetBaseURI(input.name)
		}
		if cat != nil {
			p.SetCatalog(cat)
		}

		doc, err = p.Parse(buf)
		if cfg.timing {
			fmt.Fprintf(os.Stderr, "Parsing took %s\n", time.Since(t0))
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			if doc == nil {
				return exitErr
			}
			// With --recover, doc may be non-nil
		}
	}

	if doc == nil {
		return exitErr
	}

	// XInclude processing
	if cfg.doXInclude {
		var t0 time.Time
		if cfg.timing {
			t0 = time.Now()
		}
		var xiOpts []xinclude.Option
		xiOpts = append(xiOpts, xinclude.WithParseFlags(cfg.parseOptions))
		if !input.stdin {
			xiOpts = append(xiOpts, xinclude.WithBaseURI(input.name))
		}
		_, xiErr := xinclude.Process(doc, xiOpts...)
		if cfg.timing {
			fmt.Fprintf(os.Stderr, "XInclude took %s\n", time.Since(t0))
		}
		if xiErr != nil {
			fmt.Fprintf(os.Stderr, "%s\n", xiErr)
		}
	}

	// Schema validation
	if schema != nil {
		var t0 time.Time
		if cfg.timing {
			t0 = time.Now()
		}
		err := xsd.Validate(doc, schema)
		if cfg.timing {
			fmt.Fprintf(os.Stderr, "Validating took %s\n", time.Since(t0))
		}
		if err != nil {
			fmt.Fprint(os.Stderr, err)
			return exitValidation
		}
	}

	// DTD validation result (already done during parsing if --valid)
	if cfg.parseOptions.IsSet(helium.ParseDTDValid) && err != nil {
		return exitValidation
	}

	// XPath query
	if cfg.xpathExpr != "" {
		return evalXPath(cfg, doc, out)
	}

	// Output
	if cfg.noout {
		return exitOK
	}

	var t0 time.Time
	if cfg.timing {
		t0 = time.Now()
	}

	// C14N output
	if cfg.c14nMode > 0 {
		var mode c14n.Mode
		switch cfg.c14nMode {
		case 1:
			mode = c14n.C14N10
		case 2:
			mode = c14n.C14N11
		case 3:
			mode = c14n.ExclusiveC14N10
		}
		cErr := c14n.Canonicalize(out, doc, mode, c14n.WithComments())
		if cfg.timing {
			fmt.Fprintf(os.Stderr, "Saving took %s\n", time.Since(t0))
		}
		if cErr != nil {
			fmt.Fprintf(os.Stderr, "%s\n", cErr)
			return exitErr
		}
		return exitOK
	}

	// Standard dump
	var opts []helium.WriteOption
	if cfg.format {
		opts = append(opts, helium.WithFormat())
	}
	opts = append(opts, helium.WithIndentString("  "))
	if cfg.dropdtd {
		opts = append(opts, helium.WithSkipDTD())
	}
	d := helium.NewWriter(opts...)
	if dErr := d.WriteDoc(out, doc); dErr != nil {
		if cfg.timing {
			fmt.Fprintf(os.Stderr, "Saving took %s\n", time.Since(t0))
		}
		fmt.Fprintf(os.Stderr, "%s\n", dErr)
		return exitErr
	}
	if cfg.timing {
		fmt.Fprintf(os.Stderr, "Saving took %s\n", time.Since(t0))
	}
	return exitOK
}

func evalXPath(cfg *config, doc *helium.Document, out io.Writer) int {
	result, err := xpath.Evaluate(doc, cfg.xpathExpr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "XPath error: %s\n", err)
		return exitXPath
	}

	d := helium.NewWriter()

	switch result.Type {
	case xpath.NodeSetResult:
		for _, node := range result.NodeSet {
			switch node.Type() {
			case helium.AttributeNode:
				if _, wErr := fmt.Fprintf(out, " %s=\"%s\"\n", node.Name(), string(node.Content())); wErr != nil {
					return exitXPath
				}
			case helium.NamespaceNode:
				if _, wErr := fmt.Fprintf(out, " xmlns:%s=\"%s\"\n", node.Name(), string(node.Content())); wErr != nil {
					return exitXPath
				}
			default:
				if dErr := d.WriteNode(out, node); dErr != nil {
					fmt.Fprintf(os.Stderr, "%s\n", dErr)
					return exitXPath
				}
				if _, wErr := fmt.Fprintln(out); wErr != nil {
					return exitXPath
				}
			}
		}
	case xpath.BooleanResult:
		if result.Boolean {
			if _, wErr := fmt.Fprintln(out, "true"); wErr != nil {
				return exitXPath
			}
		} else {
			if _, wErr := fmt.Fprintln(out, "false"); wErr != nil {
				return exitXPath
			}
		}
	case xpath.NumberResult:
		if _, wErr := fmt.Fprintln(out, result.Number); wErr != nil {
			return exitXPath
		}
	case xpath.StringResult:
		if _, wErr := fmt.Fprintln(out, result.String); wErr != nil {
			return exitXPath
		}
	}
	return exitOK
}

